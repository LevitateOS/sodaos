"""Soda-owned activation/packaging fixtures and optional real Caddy route checks.

No real Forgejo, host configuration, credentials or provider state are changed.
SODA_CADDY_BINARY opts into a test-owned loopback proxy using the selected Caddy.
"""

import contextlib
import http.client
import http.server
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))


class AvatarActivation(unittest.TestCase):
    """CLI surface of the Rust soda-activate binary.

    Deep activation behavior (provider derivation, file modes, ownership,
    health probes) is covered by rust/soda-activate's own tests with
    injected paths and fake system services.
    """

    @classmethod
    def setUpClass(cls):
        build = subprocess.run(
            ['cargo', 'build', '--offline', '-p', 'soda-activate'],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        if build.returncode != 0:
            raise AssertionError(f'cannot build soda-activate: {build.stderr[-2000:]}')
        cls.binary = str(ROOT / 'target/debug/soda-activate')

    def run_activate(self, *argv):
        return subprocess.run([self.binary, *argv], capture_output=True, text=True)

    def test_help_reports_private_activation_surface(self):
        proc = self.run_activate('--help')
        self.assertEqual(proc.returncode, 0)
        for flag in ('--bind-ip', '--certificate', '--private-key', '--local-tls'):
            self.assertIn(flag, proc.stdout)
        self.assertIn('explicit private appliance address', proc.stdout)

    def test_missing_bind_ip_rejected(self):
        proc = self.run_activate('--local-tls')
        self.assertEqual(proc.returncode, 2)
        self.assertIn('the following arguments are required: --bind-ip', proc.stderr)

    def test_tls_modes_rejected_before_effects(self):
        for extra in (
            [],
            ['--certificate', '/missing'],
            ['--local-tls', '--certificate', '/missing'],
            ['--local-tls', '--private-key', '/missing'],
        ):
            with self.subTest(extra=extra):
                proc = self.run_activate('--bind-ip', '192.168.1.5', *extra)
                self.assertEqual(proc.returncode, 2)

    def test_unknown_flags_rejected(self):
        proc = self.run_activate('--bind-ip', '192.168.1.5', '--bogus')
        self.assertEqual(proc.returncode, 2)
        self.assertIn('unrecognized arguments: --bogus', proc.stderr)

    @unittest.skipIf(os.geteuid() == 0, 'root passes the operator check')
    def test_nonroot_refused_without_effects(self):
        proc = self.run_activate('--bind-ip', '192.168.1.5', '--local-tls')
        self.assertEqual(proc.returncode, 2)
        self.assertEqual(
            proc.stderr.strip().splitlines()[-1],
            'soda-activate: error: native host operator/root required',
        )


class AvatarPackaging(unittest.TestCase):
    def test_no_avatar_runtime_command(self):
        self.assertFalse((ROOT / 'cmd/soda-avatars').exists())
        self.assertFalse((ROOT / 'scripts/build-native.sh').exists())
        self.assertNotIn('soda-avatars', (ROOT / 'tools/soda-build/main.go').read_text())


@unittest.skipUnless(os.environ.get('SODA_CADDY_BINARY'), 'set SODA_CADDY_BINARY for real loopback routing checks')
class AvatarProxy(unittest.TestCase):
    def test_local_tls_uses_native_issuer_without_automatic_client_trust(self):
        binary = str(Path(os.environ['SODA_CADDY_BINARY']).resolve())
        for address, origin, listener in (
            ('192.168.2.100', 'https://192.168.2.100', '192.168.2.100:443'),
            ('fd00::5', 'https://[fd00::5]', '[fd00::5]:443'),
        ):
            with self.subTest(address=address):
                env = dict(os.environ, FORGEJO_ORIGIN=origin, SODA_BIND=address, SODA_TLS='internal')
                adapted = subprocess.run(
                    [binary, 'adapt', '--config', str(ROOT / 'appliance/config/proxy.Caddyfile')],
                    env=env,
                    check=True,
                    capture_output=True,
                    text=True,
                )
                config = json.loads(adapted.stdout)
                self.assertTrue(config['admin']['disabled'])
                self.assertFalse(config['apps']['pki']['certificate_authorities']['local']['install_trust'])
                policies = config['apps']['tls']['automation']['policies']
                self.assertEqual(policies, [{'subjects': [address], 'issuers': [{'module': 'internal'}]}])
                servers = list(config['apps']['http']['servers'].values())
                self.assertEqual(len(servers), 1)
                self.assertEqual(servers[0]['listen'], [listener])
                self.assertTrue(servers[0]['automatic_https']['disable_redirects'])

    def test_production_routes_with_test_owned_upstreams(self):
        binary = str(Path(os.environ['SODA_CADDY_BINARY']).resolve())
        with tempfile.TemporaryDirectory() as directory, contextlib.ExitStack() as stack:
            temp = Path(directory)
            env = dict(
                os.environ,
                FORGEJO_ORIGIN='https://forge.example.test',
                SODA_BIND='127.0.0.1',
                XDG_CONFIG_HOME=str(temp / 'config'),
                XDG_DATA_HOME=str(temp / 'data'),
            )
            result = subprocess.run(
                [binary, 'adapt', '--adapter', 'caddyfile', '--config', str(ROOT / 'appliance/config/proxy.Caddyfile')],
                env=env,
                check=True,
                capture_output=True,
                text=True,
            )
            config = json.loads(result.stdout)

            def upstream(name):
                class Handler(http.server.BaseHTTPRequestHandler):
                    def do_GET(self):
                        body = json.dumps(
                            {
                                'upstream': name,
                                'path': self.path,
                                'cookie': self.headers.get('Cookie'),
                                'authorization': self.headers.get('Authorization'),
                            }
                        ).encode()
                        self.send_response(200)
                        self.send_header('Content-Length', str(len(body)))
                        self.end_headers()
                        self.wfile.write(body)

                    def log_message(self, *_):
                        pass

                server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
                thread = threading.Thread(target=server.serve_forever, daemon=True)
                thread.start()
                stack.callback(server.server_close)
                stack.callback(server.shutdown)
                return server.server_address[1]

            ports = {'127.0.0.1:3000': upstream('forgejo'), '127.0.0.1:8080': upstream('soda')}

            def replace_dials(node):
                if isinstance(node, dict):
                    if node.get('dial') in ports:
                        node['dial'] = '127.0.0.1:' + str(ports[node['dial']])
                    for value in node.values():
                        replace_dials(value)
                elif isinstance(node, list):
                    for value in node:
                        replace_dials(value)

            replace_dials(config)
            # Keep all adapted host/path/header routes. Only the test listeners,
            # upstream addresses and TLS transport differ from the shipped config.
            config['apps'].pop('tls')
            self.assertTrue(config['admin']['disabled'])
            with socket.socket() as sock:
                sock.bind(('127.0.0.1', 0))
                port = sock.getsockname()[1]
            servers = config['apps']['http']['servers']
            self.assertEqual(len(servers), 1)
            server = next(iter(servers.values()))
            server['listen'] = ['127.0.0.1:' + str(port)]
            server.pop('tls_connection_policies')
            server['automatic_https'] = {'disable': True}
            conf = temp / 'caddy.json'
            conf.write_text(json.dumps(config))
            log = stack.enter_context((temp / 'caddy.log').open('w+'))
            process = subprocess.Popen([binary, 'run', '--config', str(conf)], env=env, stdout=log, stderr=log)

            def stop():
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)

            stack.callback(stop)

            def request(path, host='forge.example.test'):
                connection = http.client.HTTPConnection('127.0.0.1', port, timeout=3)
                try:
                    connection.request(
                        'GET',
                        path,
                        headers={'Host': host, 'Cookie': 'fixture=value', 'Authorization': 'Bearer synthetic'},
                    )
                    response = connection.getresponse()
                    self.assertEqual(response.status, 200)
                    return json.loads(response.read())
                finally:
                    connection.close()

            for _ in range(100):
                try:
                    request('/')
                    break
                except (ConnectionRefusedError, ConnectionResetError):
                    if process.poll() is not None:
                        log.seek(0)
                        self.fail(log.read())
                    time.sleep(0.02)
            else:
                self.fail('test proxy did not start')
            path = '/-/soda/avatars/v1/' + 'a' * 32 + '?s=64&d=identicon'
            response = request(path)
            self.assertEqual(response, {'upstream': 'soda', 'path': path, 'cookie': None, 'authorization': None})
            for path in (
                '/',
                '/api/v1/users/alice',
                '/user/login',
                '/assets/img/logo.png',
                '/alice/repo.git/info/refs?service=git-upload-pack',
                '/alice/repo.git/info/lfs/objects/batch',
                '/api/packages/alice',
                '/-/unrelated',
            ):
                response = request(path)
                self.assertEqual(response['upstream'], 'forgejo', path)
                self.assertEqual(response['path'], path)
                self.assertEqual(response['cookie'], 'fixture=value')
            for path in (
                '/-/soda/avatars',
                '/-/soda/avatars-other/a',
                '/-/soda/avatarsx/a',
            ):
                response = request(path)
                self.assertEqual(response['upstream'], 'forgejo', path)
                self.assertEqual(response['path'], path)
                self.assertEqual(response['cookie'], 'fixture=value')
                self.assertEqual(response['authorization'], 'Bearer synthetic')


if __name__ == '__main__':
    unittest.main()

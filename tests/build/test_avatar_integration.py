"""Soda-owned activation/packaging fixtures and optional real Caddy route checks.

No real Forgejo, host configuration, credentials or provider state are changed.
SODA_CADDY_BINARY opts into a test-owned loopback proxy using the selected Caddy.
"""
import contextlib
import http.client
import http.server
import importlib.util
import json
import os
from pathlib import Path
import runpy
import shutil
import socket
import subprocess
import tempfile
import threading
import time
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


class AvatarActivation(unittest.TestCase):
    def test_first_activation_derives_provider_from_forgejo_origin(self):
        for origin in ('https://forge.example.test:8443', 'https://[fd00::5]:8443/'):
            with self.subTest(origin=origin), tempfile.TemporaryDirectory() as directory:
                temp = Path(directory)
                config_root = temp / 'etc/soda'
                config_root.mkdir(parents=True)
                config = {
                    'forgejo_url': origin,
                    'forgejo_internal_url': 'http://127.0.0.1:3000',
                    'listen': '127.0.0.1:8080',
                    'oauth_secret_file': str(config_root / 'oauth'),
                    'admin_token_file': str(config_root / 'token'),
                    'grant_key_file': str(config_root / 'grant'),
                }
                (config_root / 'dashboard.json').write_text(json.dumps(config))
                for name in ('oauth', 'token', 'grant'):
                    (config_root / name).write_text('synthetic, not a credential')
                (config_root / 'forgejo.env').write_text(
                    'FORGEJO__ui__DEFAULT_THEME=soda-auto\n'
                    'FORGEJO__server__SSH_DOMAIN=retained.example.test\n')
                for name in ('certificate', 'private-key'):
                    (temp / name).write_text('synthetic fixture')

                def mapped_path(value):
                    path = Path(value)
                    if path == Path('/etc/soda'):
                        return config_root
                    if str(path).startswith('/etc/containers/'):
                        return temp / path.relative_to('/')
                    return path

                args = ['soda-activate', '--bind-ip', '127.0.0.1',
                        '--certificate', str(temp / 'certificate'),
                        '--private-key', str(temp / 'private-key')]
                with patch('sys.argv', args), patch('pathlib.Path', side_effect=mapped_path), \
                        patch('os.geteuid', return_value=0), patch('os.chown'), \
                        patch('pwd.getpwnam', return_value=SimpleNamespace(pw_uid=2000, pw_gid=2000)), \
                        patch('subprocess.run') as run, patch('builtins.print'):
                    runpy.run_path(str(ROOT / 'appliance/bin/soda-activate'), run_name='__main__')
                values = dict(line.split('=', 1) for line in (config_root / 'forgejo.env').read_text().splitlines())
                self.assertEqual(values['FORGEJO__picture__GRAVATAR_SOURCE'], origin.rstrip('/') + '/-/soda/avatars/v1/')
                self.assertEqual(values['FORGEJO__server__SSH_DOMAIN'], 'retained.example.test')
                self.assertEqual(values['FORGEJO__ui__DEFAULT_THEME'], 'soda-auto')
                self.assertFalse(any('DISABLE_GRAVATAR' in k or 'FEDERATED' in k or 'OFFLINE_MODE' in k for k in values))
                self.assertEqual(run.call_count, 3)  # existing activation phases only


class AvatarPackaging(unittest.TestCase):
    def test_real_metadata_collector_copies_dependency_notices(self):
        spec = importlib.util.spec_from_file_location('avatar_build_info', ROOT / 'scripts/native-build-info.py')
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            # Actual public inputs; only the build observations below are synthetic.
            for source in ('go.mod', 'go.sum', 'project-os/locks/tea-source.toml',
                           'appliance/locks/github-runner-source.toml', 'appliance/locks/coreos-qemu.json',
                           'cockpit/package.json', 'cockpit/bun.lock', 'scripts/install-native.sh',
                           'docs/native-support-notices.md', 'project-os/licenses/tea-LICENSE',
                           'appliance/licenses/avatar-dependencies.txt', 'LICENSE', 'NOTICE'):
                destination = root / source
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / source, destination)
            stage = root / '.artifacts/native/x86_64'
            stage.mkdir(parents=True)
            for name in ('base', 'project-os', 'dashboard', 'forgejo', 'caddy'):
                (stage / (name + '.iid')).write_text('sha256:' + 'a' * 64)
            def observed_output(args):
                if '{{json .RepoDigests}}' in args:
                    return '[]'
                return 'synthetic build observation'
            with patch.object(module.platform, 'system', return_value='Linux'), \
                    patch.object(module.platform, 'machine', return_value='x86_64'), \
                    patch.object(module, 'output', side_effect=observed_output):
                module.collect(root, 'x86_64', 'a' * 40)
            self.assertEqual((stage / 'notices/avatar-dependencies.txt').read_bytes(),
                             (ROOT / 'appliance/licenses/avatar-dependencies.txt').read_bytes())
            self.assertFalse((ROOT / 'cmd/soda-avatars').exists())
            self.assertNotIn('soda-avatars', (ROOT / 'scripts/build-native.sh').read_text())


@unittest.skipUnless(os.environ.get('SODA_CADDY_BINARY'), 'set SODA_CADDY_BINARY for real loopback routing checks')
class AvatarProxy(unittest.TestCase):
    def test_production_routes_with_test_owned_upstreams(self):
        binary = str(Path(os.environ['SODA_CADDY_BINARY']).resolve())
        with tempfile.TemporaryDirectory() as directory, contextlib.ExitStack() as stack:
            temp = Path(directory)
            env = dict(os.environ, FORGEJO_ORIGIN='https://forge.example.test', SODA_BIND='127.0.0.1',
                       XDG_CONFIG_HOME=str(temp / 'config'), XDG_DATA_HOME=str(temp / 'data'))
            result = subprocess.run([binary, 'adapt', '--adapter', 'caddyfile', '--config',
                                     str(ROOT / 'appliance/config/proxy.Caddyfile')],
                                    env=env, check=True, capture_output=True, text=True)
            config = json.loads(result.stdout)
            def upstream(name):
                class Handler(http.server.BaseHTTPRequestHandler):
                    def do_GET(self):
                        body = json.dumps({'upstream': name, 'path': self.path,
                                           'cookie': self.headers.get('Cookie'),
                                           'authorization': self.headers.get('Authorization')}).encode()
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
                    connection.request('GET', path, headers={'Host': host, 'Cookie': 'fixture=value', 'Authorization': 'Bearer synthetic'})
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
            for path in ('/', '/api/v1/users/alice', '/user/login', '/assets/img/logo.png',
                         '/alice/repo.git/info/refs?service=git-upload-pack',
                         '/alice/repo.git/info/lfs/objects/batch', '/api/packages/alice',
                         '/-/unrelated'):
                response = request(path)
                self.assertEqual(response['upstream'], 'forgejo', path)
                self.assertEqual(response['path'], path)
                self.assertEqual(response['cookie'], 'fixture=value')
            for path in ('/-/soda/avatars', '/-/soda/avatars-other/a', '/-/soda/api/session',
                         '/-/soda/login', '/-/soda/avatarsx/a', '/-/soda/oauth/callback'):
                response = request(path)
                self.assertEqual(response['upstream'], 'soda', path)
                self.assertEqual(response['path'], path)
                self.assertEqual(response['cookie'], 'fixture=value')
                self.assertEqual(response['authorization'], 'Bearer synthetic')


if __name__ == '__main__':
    unittest.main()

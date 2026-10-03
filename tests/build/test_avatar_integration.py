"""Soda-owned activation/packaging fixtures and optional real Caddy route checks.

No real Forgejo, host configuration, credentials or provider state are changed.
SODA_CADDY_BINARY opts into a test-owned loopback proxy using the selected Caddy.
"""

import contextlib
import http.client
import http.server
import importlib.util
import io
import json
import os
from pathlib import Path
import runpy
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))


class AvatarActivation(unittest.TestCase):
    def test_first_activation_derives_provider_from_forgejo_origin(self):
        for origin, local_tls in (
            ('https://forge.example.test:8443', False),
            ('https://[fd00::5]:8443/', False),
            ('https://192.168.2.100', True),
            ('https://[fd00::5]', True),
        ):
            with self.subTest(origin=origin), tempfile.TemporaryDirectory() as directory:
                temp = Path(directory)
                config_root = temp / 'etc/soda'
                config_root.mkdir(parents=True)
                config = {
                    'forgejo_url': origin,
                    'forgejo_internal_url': 'http://127.0.0.1:3000',
                    'listen': '127.0.0.1:8080',
                    'grant_key_file': '/etc/soda/grant-key',
                    'host_socket': '/run/soda/host.sock',
                    'identity_socket': '/run/soda/identity/admin.sock',
                    'operator_id': 42,
                }
                (config_root / 'dashboard.json').write_text(json.dumps(config))
                for name in ('grant-key',):
                    (config_root / name).write_text('synthetic, not a credential')
                    (config_root / name).chmod(0o600)
                extension_root = temp / 'var/lib/soda/forgejo/gitea/extensions'
                extension_root.parent.mkdir(parents=True)
                (config_root / 'forgejo.env').write_text(
                    'FORGEJO__ui__DEFAULT_THEME=soda-auto\nFORGEJO__server__SSH_DOMAIN=retained.example.test\n'
                )
                for name in ('certificate', 'private-key'):
                    (temp / name).write_text('synthetic fixture')

                def mapped_path(value):
                    path = Path(value)
                    if str(path) == '/etc/soda' or str(path).startswith('/etc/soda/'):
                        return config_root / path.relative_to('/etc/soda')
                    if str(path).startswith('/var/lib/soda/'):
                        return temp / path.relative_to('/')
                    if str(path).startswith('/etc/containers/'):
                        return temp / path.relative_to('/')
                    return path

                address = 'fd00::5' if origin == 'https://[fd00::5]' else '192.168.2.100'
                args = ['soda-activate', '--bind-ip', address]
                if local_tls:
                    args += ['--local-tls']
                else:
                    args += ['--certificate', str(temp / 'certificate'), '--private-key', str(temp / 'private-key')]
                with (
                    patch('sys.argv', args),
                    patch('pathlib.Path', side_effect=mapped_path),
                    patch('os.geteuid', return_value=0),
                    patch('os.chown') as chown,
                    patch('pwd.getpwnam', return_value=SimpleNamespace(pw_uid=2000, pw_gid=2000)),
                    patch('subprocess.run') as run,
                    patch('builtins.print'),
                ):
                    run.return_value.returncode = 0
                    runpy.run_path(str(ROOT / 'appliance/bin/soda-activate'), run_name='__main__')
                values = dict(line.split('=', 1) for line in (config_root / 'forgejo.env').read_text().splitlines())
                self.assertEqual(
                    values['FORGEJO__picture__GRAVATAR_SOURCE'], origin.rstrip('/') + '/-/soda/avatars/v1/'
                )
                self.assertEqual(values['FORGEJO__server__SSH_DOMAIN'], 'retained.example.test')
                self.assertEqual(values['FORGEJO__ui__DEFAULT_THEME'], 'soda-auto')
                self.assertEqual(values['FORGEJO__extensions__REQUIRED_IDS'], 'soda')
                self.assertEqual(values['FORGEJO__extensions__SERVICE_CALLBACK_PATH'], '/ipc/host.sock')
                self.assertFalse(
                    any('DISABLE_GRAVATAR' in k or 'FEDERATED' in k or 'OFFLINE_MODE' in k for k in values)
                )
                self.assertEqual(run.call_count, 6)  # 3 activation phases + 3 is-active health probes
                self.assertEqual(
                    [call.args for call in chown.call_args_list],
                    [
                        (config_root, 0, 2000),
                        (config_root / 'dashboard.json', 0, 2000),
                        (config_root / 'grant-key', 0, 2000),
                        (extension_root, 1000, 1000),
                        (extension_root / '.data', 1000, 1000),
                        (extension_root / '.data/soda', 1000, 1000),
                        (extension_root / '.data/soda/operator-id', 1000, 1000),
                    ],
                )
                for name in ('dashboard.json', 'grant-key'):
                    self.assertEqual((config_root / name).stat().st_mode & 0o777, 0o640)
                operator_file = extension_root / '.data/soda/operator-id'
                self.assertEqual(operator_file.read_text(), '42\n')
                self.assertEqual(operator_file.stat().st_mode & 0o777, 0o600)
                self.assertEqual(extension_root.stat().st_mode & 0o777, 0o700)
                proxy = (config_root / 'proxy.env').read_text()
                self.assertIn(
                    'SODA_TLS=internal\n' if local_tls else 'SODA_TLS=/etc/soda/tls/cert.pem /etc/soda/tls/key.pem\n',
                    proxy,
                )
                self.assertEqual((config_root / 'tls/cert.pem').exists(), not local_tls)

    def test_tls_mode_rejects_missing_or_mixed_inputs_before_effects(self):
        for extra in (
            [],
            ['--certificate', '/missing'],
            ['--local-tls', '--certificate', '/missing'],
            ['--local-tls', '--private-key', '/missing'],
        ):
            with (
                self.subTest(extra=extra),
                patch('sys.argv', ['soda-activate', '--bind-ip', '192.168.1.5', *extra]),
                patch('os.geteuid', return_value=0),
                patch('subprocess.run') as run,
                patch('sys.stderr', new_callable=io.StringIO),
            ):
                with self.assertRaises(SystemExit):
                    runpy.run_path(str(ROOT / 'appliance/bin/soda-activate'), run_name='__main__')
                run.assert_not_called()

    def test_empty_identity_socket_uses_standard_admin_socket(self):
        # soda-setup records Go's unset value ("") for identity_socket; the Go
        # loader fills the standard path. Activation must accept the same
        # encoding instead of rejecting its own setup output.
        with tempfile.TemporaryDirectory() as directory:
            temp = Path(directory)
            config_root = temp / 'etc/soda'
            config_root.mkdir(parents=True)
            config = {
                'forgejo_url': 'https://192.168.2.100',
                'forgejo_internal_url': 'http://127.0.0.1:3000',
                'listen': '127.0.0.1:8080',
                'grant_key_file': '/etc/soda/grant-key',
                'host_socket': '/run/soda/host.sock',
                'identity_socket': '',
                'operator_id': 7,
            }
            (config_root / 'dashboard.json').write_text(json.dumps(config))
            (config_root / 'grant-key').write_text('synthetic, not a credential')
            (config_root / 'grant-key').chmod(0o600)
            extension_root = temp / 'var/lib/soda/forgejo/gitea/extensions'
            extension_root.parent.mkdir(parents=True)
            (config_root / 'forgejo.env').write_text('FORGEJO__ui__DEFAULT_THEME=soda-auto\n')

            def mapped_path(value):
                path = Path(value)
                if str(path) == '/etc/soda' or str(path).startswith('/etc/soda/'):
                    return config_root / path.relative_to('/etc/soda')
                if str(path).startswith('/var/lib/soda/'):
                    return temp / path.relative_to('/')
                if str(path).startswith('/etc/containers/'):
                    return temp / path.relative_to('/')
                return path

            with (
                patch(
                    'sys.argv',
                    ['soda-activate', '--bind-ip', '192.168.2.100', '--local-tls'],
                ),
                patch('pathlib.Path', side_effect=mapped_path),
                patch('os.geteuid', return_value=0),
                patch('os.chown'),
                patch('pwd.getpwnam', return_value=SimpleNamespace(pw_uid=2000, pw_gid=2000)),
                patch('subprocess.run') as run,
                patch('builtins.print'),
            ):
                run.return_value.returncode = 0
                runpy.run_path(str(ROOT / 'appliance/bin/soda-activate'), run_name='__main__')
            self.assertEqual((extension_root / '.data/soda/operator-id').read_text(), '7\n')
            env = dict(
                line.split('=', 1)
                for line in (config_root / 'forgejo.env').read_text().splitlines()
                if line and not line.startswith('#') and '=' in line
            )
            self.assertEqual(env['FORGEJO__extensions__SERVICE_BRIDGE_PEERS'], '2000:soda')

    def test_activation_reports_units_that_never_become_active(self):
        with tempfile.TemporaryDirectory() as directory:
            temp = Path(directory)
            config_root = temp / 'etc/soda'
            config_root.mkdir(parents=True)
            config = {
                'forgejo_url': 'https://192.168.2.100',
                'forgejo_internal_url': 'http://127.0.0.1:3000',
                'listen': '127.0.0.1:8080',
                'grant_key_file': '/etc/soda/grant-key',
                'host_socket': '/run/soda/host.sock',
                'identity_socket': '/run/soda/identity/admin.sock',
                'operator_id': 7,
            }
            (config_root / 'dashboard.json').write_text(json.dumps(config))
            (config_root / 'grant-key').write_text('synthetic, not a credential')
            (config_root / 'grant-key').chmod(0o600)
            extension_root = temp / 'var/lib/soda/forgejo/gitea/extensions'
            extension_root.parent.mkdir(parents=True)
            (config_root / 'forgejo.env').write_text('FORGEJO__ui__DEFAULT_THEME=soda-auto\n')

            def mapped_path(value):
                path = Path(value)
                if str(path) == '/etc/soda' or str(path).startswith('/etc/soda/'):
                    return config_root / path.relative_to('/etc/soda')
                if str(path).startswith('/var/lib/soda/'):
                    return temp / path.relative_to('/')
                if str(path).startswith('/etc/containers/'):
                    return temp / path.relative_to('/')
                return path

            with (
                patch(
                    'sys.argv',
                    ['soda-activate', '--bind-ip', '192.168.2.100', '--local-tls'],
                ),
                patch('pathlib.Path', side_effect=mapped_path),
                patch('os.geteuid', return_value=0),
                patch('os.chown'),
                patch('pwd.getpwnam', return_value=SimpleNamespace(pw_uid=2000, pw_gid=2000)),
                patch('subprocess.run') as run,
                patch('time.monotonic', side_effect=[0.0, 61.0, 61.0]),
                patch('time.sleep'),
                patch('builtins.print'),
            ):
                run.return_value.returncode = 1
                with self.assertRaises(SystemExit) as failure:
                    runpy.run_path(str(ROOT / 'appliance/bin/soda-activate'), run_name='__main__')
                self.assertEqual(failure.exception.code, 2)


class AvatarPackaging(unittest.TestCase):
    def test_real_metadata_collector_copies_dependency_notices(self):
        spec = importlib.util.spec_from_file_location('avatar_build_info', ROOT / 'scripts/native-build-info.py')
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            # Actual public inputs; only the build observations below are synthetic.
            for source in (
                'go.mod',
                'go.sum',
                'package.json',
                'tools/lit-check/package.json',
                'bun.lock',
                'bunfig.toml',
                'scripts/install-native.sh',
                'docs/research/notices.md',
                'project-os/licenses/tea-LICENSE',
                'appliance/licenses/avatar-dependencies.txt',
                'LICENSE',
                'NOTICE',
            ):
                destination = root / source
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / source, destination)
            stage = root / '.artifacts/native/x86_64'
            stage.mkdir(parents=True)
            for name in ('base', 'project-os', 'dashboard', 'forgejo', 'caddy', 'tailnet'):
                (stage / (name + '.iid')).write_text('sha256:' + 'a' * 64)

            def observed_output(args):
                if '--entrypoint=/usr/local/bin/tailscale' in args:
                    return json.dumps({'short': 'synthetic tailscale version'})
                if '--entrypoint=/usr/local/bin/tailscaled' in args:
                    return 'synthetic tailscaled version'
                if '{{json .RepoDigests}}' in args:
                    return '[]'
                return 'synthetic build observation'

            with (
                patch.object(module.platform, 'system', return_value='Linux'),
                patch.object(module.platform, 'machine', return_value='x86_64'),
                patch.object(module, 'output', side_effect=observed_output),
            ):
                module.collect(root, 'x86_64', 'a' * 40)
            self.assertEqual(
                (stage / 'notices/avatar-dependencies.txt').read_bytes(),
                (ROOT / 'appliance/licenses/avatar-dependencies.txt').read_bytes(),
            )
            self.assertEqual(
                (stage / 'inputs/lit-check-package.json').read_bytes(),
                (ROOT / 'tools/lit-check/package.json').read_bytes(),
            )
            self.assertFalse((stage / 'inputs/github-runner-source.toml').exists())
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

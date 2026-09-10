"""Local media/provisioning contract doubles, not ISO boot or disk-write tests."""
import base64
import gzip
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import shutil
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, ROOT / path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class MediaConfiguration(unittest.TestCase):
    def setUp(self):
        self.module = load('media_fixture', 'scripts/build-installer.py')

    def test_media_command_cannot_overwrite_the_running_bootstrap_from_runtime_bundle(self):
        self.assertTrue((ROOT / 'appliance/installer/main.go').is_file())
        self.assertFalse((ROOT / 'cmd/soda-install').exists())
        self.assertNotIn('soda-install', (ROOT / 'scripts/stage.py').read_text())

    def test_payload_url_has_no_credentials_or_unsafe_transport(self):
        self.assertEqual(self.module.payload_url('https://media.example.test/soda/', 'console'),
                         'https://media.example.test/soda/console')
        for url in ('http://media.test', 'https://user:secret@media.test', 'https://media.test/?token=x',
                    'https://media.test/#fragment', 'https://media.test/?', 'file:///tmp/a', 'https://media.test/\n'):
            with self.assertRaises(ValueError):
                self.module.payload_url(url, 'console')

    def test_live_configuration_is_public_and_has_no_automatic_erase_target(self):
        config = self.module.live_config('https://media.test/console', 'a' * 64,
            {'ignition': {'version': '3.5.0'}}, {'Architecture': 'x86_64'}, 'canonical-artwork')
        raw = json.dumps(config)
        self.assertNotIn('passwd', config)
        self.assertNotIn('coreos/installer.d', raw)
        self.assertNotIn('dest-device', raw)
        self.assertNotIn('passwordHash', raw)
        self.assertNotIn('enforcing=0', raw)
        files = {f['path']: f for f in config['storage']['files']}
        binary = files[self.module.BINARY_PATH]
        self.assertEqual(binary['contents']['verification']['hash'], 'sha256-' + 'a' * 64)
        self.assertEqual(binary['mode'], 0o755)
        self.assertEqual(files['/etc/motd']['contents']['inline'].splitlines()[0], 'canonical-artwork')
        service = config['systemd']['units'][0]['contents']
        self.assertIn('ExecStart=/usr/local/libexec/soda/soda-install disk', service)
        self.assertIn('StandardError=tty', service)
        self.assertIn('Restart=no', service)
        self.assertNotIn('coreos-installer install', service)
        self.assertIn({'name': 'getty@tty1.service', 'mask': True}, config['systemd']['units'])

    def test_readback_understands_exact_upstream_gzip_merge_wrapper(self):
        expected = {'ignition': {'version': '3.5.0'}, 'storage': {'files': []}}
        fragment = {'compression': 'gzip', 'source': 'data:;base64,' + base64.b64encode(
            gzip.compress(json.dumps(expected).encode())).decode()}
        wrapper = {'ignition': {'version': '3.3.0', 'config': {'merge': [fragment]}}}
        self.module.verify_embedded(json.dumps(wrapper), expected)
        with self.assertRaises(ValueError):
            self.module.verify_embedded(json.dumps(wrapper), {'different': True})
        wrapper['storage'] = {'files': [{'path': '/etc/coreos/installer.d/auto.yaml'}]}
        with self.assertRaises(ValueError):
            self.module.verify_embedded(json.dumps(wrapper), expected)
        del wrapper['storage']
        wrapper['ignition']['config']['merge'].append(fragment)
        with self.assertRaises(ValueError):
            self.module.verify_embedded(json.dumps(wrapper), expected)

    def test_conversion_is_strict_and_failure_does_not_create_output(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'live.ign'
            result = subprocess.CompletedProcess([], 0,
                stdout=b'{"ignition":{"version":"3.5.0"}}')
            with patch.object(self.module.subprocess, 'run', return_value=result) as run:
                self.module.convert(Path('/tools/butane'), {'variant': 'fcos'}, output)
            self.assertEqual(run.call_args.args[0], ['/tools/butane', '--strict'])
            self.assertEqual(run.call_args.kwargs['stderr'], subprocess.DEVNULL)
            with patch.object(self.module.subprocess, 'run', return_value=result):
                with self.assertRaises(FileExistsError):
                    self.module.convert(Path('/tools/butane'), {}, output)
            failed = Path(directory) / 'failed.ign'
            with patch.object(self.module.subprocess, 'run', side_effect=subprocess.CalledProcessError(1, 'butane')):
                with self.assertRaises(subprocess.CalledProcessError):
                    self.module.convert(Path('/tools/butane'), {}, failed)
            self.assertFalse(failed.exists())

    def test_private_static_network_snapshot_refuses_public_or_symlink_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'network'
            source.write_text('[connection]\nid=fixture\n')
            source.chmod(0o644)
            with self.assertRaises(ValueError):
                self.module.snapshot_network(source, root)
            source.chmod(0o600)
            snapshot = self.module.snapshot_network(source, root)
            self.assertEqual(snapshot.read_bytes(), source.read_bytes())
            self.assertEqual(snapshot.stat().st_mode & 0o777, 0o600)
            with self.assertRaises(FileExistsError):
                self.module.snapshot_network(source, root)
            link = root / 'link'
            link.symlink_to(source)
            with self.assertRaises(OSError):
                self.module.snapshot_network(link, root)

    def test_build_pipeline_uses_only_live_customization_and_retains_outputs(self):
        # Commands are doubles: this proves the caller wiring, never ISO validity.
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('scripts/render-provisioning.py', 'appliance/provisioning/base.json',
                         'appliance/locks/coreos-iso.json', 'assets/branding/terminal/sodaos.txt', 'LICENSE', 'NOTICE'):
                dest = root / name
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / name, dest)
            (root / '.artifacts').mkdir()
            for name in ('butane', 'coreos-installer'):
                tool = root / name
                tool.write_text('fixture tool, not executable code')
                tool.chmod(0o755)
            args = SimpleNamespace(arch='x86_64', butane=str(root / 'butane'),
                coreos_installer=str(root / 'coreos-installer'), keyring=str(root / 'trusted.gpg'),
                signer='A' * 40, payload_base_url='https://media.test/soda',
                out=str(root / '.artifacts/media'), network_keyfile=None)
            calls = []
            def check_output(argv, **kwargs):
                if argv[:3] == ['git', 'rev-parse', 'HEAD']:
                    return ('a' * 40).encode()
                if argv[:2] == ['git', 'status']:
                    return b''
                if argv[:3] == ['go', 'env', 'GOVERSION']:
                    return b'go1.26.7'
                if argv[-1] == '--version':
                    return b'coreos-installer 0.26.0' if 'coreos-installer' in argv[0] else b'butane fixture'
                if argv[1:4] == ['iso', 'ignition', 'show']:
                    raw = (Path(args.out) / 'live.ign').read_bytes()
                    resource = {'compression': 'gzip', 'source': 'data:;base64,' + base64.b64encode(gzip.compress(raw)).decode()}
                    return json.dumps({'ignition': {'version': '3.3.0', 'config': {'merge': [resource]}}}).encode()
                self.fail(f'unexpected observation {argv}')
            def run(argv, **kwargs):
                calls.append(argv)
                if argv == ['go', 'mod', 'verify']:
                    pass
                elif argv[:2] == ['go', 'build']:
                    Path(argv[argv.index('-o') + 1]).write_bytes(b'synthetic-build-output')
                elif argv[1] == 'fetch-coreos-iso':
                    dest = Path(argv[argv.index('--out') + 1])
                    dest.mkdir()
                    (dest / 'coreos.iso').write_bytes(b'synthetic-upstream-iso')
                elif argv[0] == args.butane:
                    config = json.loads(kwargs['input'])
                    config.pop('variant')
                    config.pop('version')
                    config['ignition'] = {'version': '3.5.0'}
                    return subprocess.CompletedProcess(argv, 0, stdout=json.dumps(config).encode())
                elif argv[1:3] == ['iso', 'customize']:
                    self.assertEqual(kwargs['stdout'], subprocess.DEVNULL)
                    self.assertEqual(kwargs['stderr'], subprocess.DEVNULL)
                    self.assertIn('--live-ignition', argv)
                    self.assertNotIn('--dest-device', argv)
                    self.assertNotIn('--dest-ignition', argv)
                    self.assertNotIn('--force', argv)
                    Path(argv[argv.index('--output') + 1]).write_bytes(b'synthetic-customized-iso')
                else:
                    self.fail(f'unexpected effect {argv}')
                return subprocess.CompletedProcess(argv, 0)
            with patch.object(self.module, 'ROOT', root), patch.object(self.module.platform, 'machine', return_value='x86_64'), patch.object(self.module.platform, 'system', return_value='Linux'), patch.object(self.module.subprocess, 'run', side_effect=run), patch.object(self.module.subprocess, 'check_output', side_effect=check_output), patch('sys.stdout', new_callable=io.StringIO):
                self.module.build(args)
                count = len(calls)
                with self.assertRaises(FileExistsError):
                    self.module.build(args)
                self.assertEqual(len(calls), count)
            record = json.loads((Path(args.out) / 'media-build.json').read_text())
            self.assertFalse(record['PrivateMedia'])
            self.assertEqual(record['Revision'], 'a' * 40)
            self.assertTrue((Path(args.out) / 'SHA256SUMS').exists())
            self.assertFalse(any('install' in argv or 'reboot' in argv for argv in calls))

    def test_selected_iso_lock_contains_real_distinct_architecture_inputs(self):
        lock = json.loads((ROOT / 'appliance/locks/coreos-iso.json').read_text())
        self.assertEqual(set(lock['Architectures']), {'x86_64', 'aarch64'})
        for arch, image in lock['Architectures'].items():
            self.assertTrue(image['URL'].endswith(f'live-iso.{arch}.iso'))
            self.assertEqual(image['SignatureURL'], image['URL'] + '.sig')
            self.assertRegex(image['SHA256'], '^[0-9a-f]{64}$')
            self.assertNotIn('UncompressedSHA256', image)


class ProductProvisioning(unittest.TestCase):
    def setUp(self):
        self.module = load('product_provisioning_fixture', 'scripts/render-provisioning.py')
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.key = self.root / 'operator.pub'
        self.key.write_text('ssh-ed25519 AAAA synthetic-input-only\n')
        self.password = self.root / 'root.hash'
        self.password.write_text('$6$synthetic$fixture-only\n')
        self.password.chmod(0o600)

    def test_product_hostname_does_not_weaken_fixture_contract(self):
        path = self.root / 'product.bu'
        self.module.render(self.key, self.password, path, product_hostname='soda.example.test')
        data = json.loads(path.read_text())
        host = next(f for f in data['storage']['files'] if f['path'] == '/etc/hostname')
        self.assertEqual(host['contents']['inline'], 'soda.example.test\n')
        for name in ('-soda', 'soda..test', 'soda\nreboot', 'Soda', 'a' * 64):
            with self.assertRaises(ValueError):
                self.module.render(self.key, self.password, self.root / 'invalid.bu', product_hostname=name)
        with self.assertRaises(ValueError):
            self.module.render(self.key, self.password, self.root / 'fixture.bu', hostname='soda.example.test')
        with self.assertRaises(ValueError):
            self.module.render(self.key, self.password, self.root / 'both.bu', hostname='soda-native-fixture', product_hostname='soda')

    def test_public_template_remains_single_identity_free_source(self):
        data = self.module.public_config()
        self.assertEqual(data, json.loads((ROOT / 'appliance/provisioning/base.json').read_text()))
        self.assertNotIn('passwd', data)
        data['passwd'] = {'fixture': True}
        self.assertNotIn('passwd', self.module.public_config())

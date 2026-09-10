"""Local media/provisioning contract doubles, not ISO boot or disk-write tests."""
import base64
import gzip
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import shutil
import struct
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

    def test_console_launcher_is_on_media_and_verifies_before_execution(self):
        script = (ROOT / 'appliance/installer/load-console.sh').read_text()
        subprocess.run(['bash', '-n'], input=script.encode(), check=True)
        self.assertIn('mountpoint -q /run/media/iso', script)
        self.assertIn('source=/run/media/iso/soda/soda-install', script)
        self.assertIn('! -e "$destination"', script)
        self.assertLess(script.index('sha256sum --check --status'), script.index('exec "$destination" disk'))
        self.assertLess(script.index('restorecon -F'), script.index('exec "$destination" disk'))
        self.assertNotIn('curl', script)
        self.assertNotIn('wget', script)
        self.assertNotIn('coreos-installer install', script)

    def test_launcher_copy_hash_failure_and_existing_destination(self):
        # Substitute only fixed filesystem roots; root/mount/SELinux commands
        # are doubles. The real shell copy/hash/exec path runs on synthetic files.
        script = (ROOT / 'appliance/installer/load-console.sh').read_text()
        for case in ('valid', 'wrong-hash', 'occupied', 'missing-mount', 'relabel-failure'):
            with self.subTest(case=case), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                media = root / 'iso'
                (media / 'soda').mkdir(parents=True)
                signal = root / 'executed'
                source = media / 'soda/soda-install'
                source.write_text(f'#!/bin/sh\nprintf reached > {signal}\n')
                destination = root / 'console'
                marker = root / 'ostree-live'
                marker.touch()
                tools = root / 'tools'
                tools.mkdir()
                for name, body in {'id': 'echo 0',
                                   'mountpoint': 'exit ' + ('1' if case == 'missing-mount' else '0'),
                                   'restorecon': 'exit ' + ('1' if case == 'relabel-failure' else '0')}.items():
                    tool = tools / name
                    tool.write_text('#!/bin/sh\n' + body + '\n')
                    tool.chmod(0o755)
                local = script.replace('/run/media/iso', str(media)).replace('/run/ostree-live', str(marker))
                local = local.replace('/usr/local/libexec/soda/soda-install', str(destination))
                launcher = root / 'loader.sh'
                launcher.write_text(local)
                expected = hashlib.sha256(source.read_bytes()).hexdigest()
                if case == 'wrong-hash':
                    expected = '0' * 64
                if case == 'occupied':
                    destination.write_text('preserve existing file')
                result = subprocess.run(['bash', str(launcher), expected],
                    env=dict(os.environ, PATH=str(tools) + os.pathsep + os.environ['PATH']),
                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                self.assertEqual(result.returncode == 0, case == 'valid')
                self.assertEqual(signal.exists(), case == 'valid')
                if case == 'occupied':
                    self.assertEqual(destination.read_text(), 'preserve existing file')
                if case == 'missing-mount':
                    self.assertFalse(destination.exists())
                staging = Path(str(destination) + '.incoming')
                if case == 'valid':
                    self.assertEqual(destination.stat().st_mode & 0o777, 0o700)
                    self.assertFalse(staging.exists())
                if case == 'wrong-hash':
                    self.assertFalse(destination.exists())
                    self.assertEqual(staging.stat().st_mode & 0o777, 0o600)
                if case == 'relabel-failure':
                    self.assertEqual(destination.stat().st_mode & 0o777, 0o600)
                    self.assertEqual(staging.stat().st_mode & 0o777, 0o600)

    def test_live_configuration_is_public_and_has_no_automatic_erase_target(self):
        config = self.module.live_config('a' * 64,
            {'ignition': {'version': '3.5.0'}}, {'Architecture': 'x86_64'}, 'canonical-artwork',
            load('branding_fixture', 'scripts/render-provisioning.py').branding_files())
        raw = json.dumps(config)
        self.assertNotIn('passwd', config)
        self.assertNotIn('coreos/installer.d', raw)
        self.assertNotIn('dest-device', raw)
        self.assertNotIn('passwordHash', raw)
        self.assertNotIn('enforcing=0', raw)
        files = {f['path']: f for f in config['storage']['files']}
        loader = files[self.module.LOADER_PATH]
        self.assertEqual(loader['contents']['inline'], (ROOT / 'appliance/installer/load-console.sh').read_text())
        self.assertEqual(loader['mode'], 0o755)
        self.assertTrue(all(set(f['contents']) == {'inline'} for f in files.values()))
        self.assertEqual(files['/etc/motd']['contents']['inline'].splitlines()[0], 'canonical-artwork')
        self.assertEqual({f['path'] for f in files.values() if f.get('overwrite')}, {'/etc/motd', '/etc/os-release'})
        self.assertIs(files['/etc/motd']['overwrite'], True)
        service = config['systemd']['units'][0]['contents']
        self.assertIn('ExecStart=/usr/local/libexec/soda/load-install-console ' + 'a' * 64, service)
        self.assertIn('RequiresMountsFor=/run/media/iso', service)
        self.assertIn('StandardError=tty', service)
        self.assertIn('Type=idle', service)
        self.assertIn('PRETTY_NAME="SodaOS"', files['/etc/os-release']['contents']['inline'])
        self.assertNotIn('IMAGE_VERSION=', files['/etc/os-release']['contents']['inline'])
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

    def test_readback_allows_only_selected_serde_null_defaults(self):
        expected = {'ignition': {'version': '3.5.0'},
                    'storage': {'files': [{'path': '/fixture', 'mode': 0o644}]},
                    'systemd': {'units': [{'name': 'fixture.service', 'enabled': True, 'contents': 'fixture'}]}}
        serialized = json.loads(json.dumps(expected))
        serialized['storage']['files'][0]['overwrite'] = None
        serialized['systemd']['units'][0]['mask'] = None
        def wrapper(config):
            return json.dumps({'ignition': {'version': '3.3.0', 'config': {'merge': [{
                'compression': 'gzip', 'source': 'data:;base64,' + base64.b64encode(
                    gzip.compress(json.dumps(config).encode())).decode()}]}}})
        self.module.verify_embedded(wrapper(serialized), expected)
        self.assertNotIn('overwrite', expected['storage']['files'][0])
        for field, value in (('overwrite', False), ('overwrite', True), ('unrecognizedEffect', None)):
            modified = json.loads(json.dumps(serialized))
            modified['storage']['files'][0][field] = value
            with self.assertRaises(ValueError):
                self.module.verify_embedded(wrapper(modified), expected)
        serialized['systemd']['units'][0]['mask'] = True
        with self.assertRaises(ValueError):
            self.module.verify_embedded(wrapper(serialized), expected)

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
                         'appliance/locks/coreos-iso.json', 'appliance/installer/load-console.sh',
                         'assets/branding/terminal/sodaos.txt', 'assets/branding/host/os-release',
                         'assets/branding/source/soda-symbol.svg', 'LICENSE', 'NOTICE'):
                dest = root / name
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / name, dest)
            (root / '.artifacts').mkdir()
            for name in ('butane', 'coreos-installer', 'xorriso'):
                tool = root / name
                tool.write_text('fixture tool, not executable code')
                tool.chmod(0o755)
            args = SimpleNamespace(arch='x86_64', butane=str(root / 'butane'),
                coreos_installer=str(root / 'coreos-installer'), keyring=str(root / 'trusted.gpg'),
                signer='A' * 40, xorriso=str(root / 'xorriso'),
                out=str(root / '.artifacts/media'), network_keyfile=None)
            calls = []
            def check_output(argv, **kwargs):
                if argv[:3] == ['git', 'rev-parse', 'HEAD']:
                    return ('a' * 40).encode()
                if argv[:2] == ['git', 'status']:
                    return b''
                if argv[:3] == ['go', 'env', 'GOVERSION']:
                    return b'go1.26.7'
                if argv[-1] == '-version':
                    return b'xorriso fixture'
                if argv[1:4] == ['iso', 'kargs', 'show']:
                    return b'fixture stock live kargs'
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
                elif argv[0] == args.xorriso:
                    self.assertEqual(argv[argv.index('-boot_image') + 1:argv.index('-boot_image') + 3], ['any', 'replay'])
                    self.assertIn('/coreos/miniso.dat', argv)
                    Path(argv[argv.index('-outdev') + 1]).write_bytes(b'synthetic-remastered-iso')
                elif argv[1:3] == ['iso', 'customize']:
                    self.assertEqual(kwargs['stdout'], subprocess.DEVNULL)
                    self.assertEqual(kwargs['stderr'], subprocess.DEVNULL)
                    self.assertIn('--live-ignition', argv)
                    self.assertEqual(argv[-1], str(Path(args.out) / 'with-console.iso'))
                    self.assertNotIn('--dest-device', argv)
                    self.assertNotIn('--dest-ignition', argv)
                    self.assertNotIn('--force', argv)
                    Path(argv[argv.index('--output') + 1]).write_bytes(b'synthetic-customized-iso')
                else:
                    self.fail(f'unexpected effect {argv}')
                return subprocess.CompletedProcess(argv, 0)
            with patch.object(self.module, 'ROOT', root), patch.object(self.module.platform, 'machine', return_value='x86_64'), patch.object(self.module.platform, 'system', return_value='Linux'), patch.object(self.module.subprocess, 'run', side_effect=run), patch.object(self.module.subprocess, 'check_output', side_effect=check_output), patch.object(self.module, 'verify_remaster', return_value={'SyntheticFixtureOnly': True}) as inspect_iso, patch.object(self.module, 'brand_boot_files', return_value={}), patch('sys.stdout', new_callable=io.StringIO):
                self.module.build(args)
                inspect_iso.assert_called_once_with(root / 'xorriso', Path(args.out) / 'upstream/coreos.iso',
                                                   Path(args.out) / 'soda.iso', Path(args.out) / 'payload')
                count = len(calls)
                with self.assertRaises(FileExistsError):
                    self.module.build(args)
                self.assertEqual(len(calls), count)
            record = json.loads((Path(args.out) / 'media-build.json').read_text())
            self.assertFalse(record['PrivateMedia'])
            self.assertEqual(record['Revision'], 'a' * 40)
            self.assertEqual(record['ConsoleISOPath'], '/soda/soda-install')
            self.assertNotIn('PayloadURL', record)
            self.assertTrue((Path(args.out) / 'SHA256SUMS').exists())
            self.assertFalse(any('install' in argv or 'reboot' in argv for argv in calls))

    def test_boot_branding_preserves_offsets_and_native_commands(self):
        for name, data in (
                ('/EFI/fedora/grub.cfg', b"menuentry 'Fedora CoreOS (Live)' --class fedora {\nlinux /kernel coreos.liveiso=fixture\n}"),
                ('/isolinux/isolinux.cfg', b'menu title Fedora CoreOS\nmenu label ^Fedora CoreOS (Live)\nappend coreos.liveiso=fixture\n')):
            with self.subTest(name=name):
                branded = self.module.branded_boot_config(name, data)
                self.assertEqual(len(branded), len(data))
                self.assertEqual(branded.index(b'coreos.liveiso'), data.index(b'coreos.liveiso'))
                self.assertIn(b'SodaOS Installer', branded)
                self.assertNotIn(b'Fedora CoreOS', branded)
                with self.assertRaises(ValueError):
                    self.module.branded_boot_config(name, data.replace(b'Fedora CoreOS (Live)', b'changed upstream'))

    def test_bios_table_relocation_and_mirrored_volume_descriptor(self):
        # Synthetic bytes only: validates the inspector, not BIOS execution.
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            hashes = []
            for index, (lba, pvd_lba) in enumerate(((64, 16), (96, 48))):
                image = bytearray(256 * 2048)
                pvd = bytearray(2048)
                pvd[:7] = b'\x01CD001\x01'
                pvd[80:88] = (256).to_bytes(4, 'little') + (256).to_bytes(4, 'big')
                image[16 * 2048:17 * 2048] = pvd
                count = 256 - (pvd_lba - 16)
                pvd[80:88] = count.to_bytes(4, 'little') + count.to_bytes(4, 'big')
                image[pvd_lba * 2048:(pvd_lba + 1) * 2048] = pvd
                binary = bytearray(bytes(range(256)) * 2)
                checksum = sum(struct.unpack('<' + 'I' * 112, binary[64:])) & 0xffffffff
                struct.pack_into('<IIII', binary, 8, pvd_lba, lba, len(binary), checksum)
                offset = lba * 2048
                image[offset:offset + len(binary)] = binary
                path = root / f'{index}.iso'
                path.write_bytes(image)
                hashes.append(self.module.iso_digest(path, (offset, len(binary)), True))
            self.assertEqual(*hashes)
            for position in (offset + 12, offset + 20, pvd_lba * 2048 + 156, pvd_lba * 2048 + 84):
                damaged = bytearray(image)
                damaged[position] ^= 1
                path.write_bytes(damaged)
                with self.assertRaises(ValueError):
                    self.module.iso_digest(path, (offset, len(binary)), True)

    @unittest.skipUnless(shutil.which('xorriso'), 'xorriso required for synthetic ISO roundtrip')
    def test_remaster_preserves_files_and_detects_tampering(self):
        # Real ISO filesystem manipulation, deliberately nonbootable dummy EFI
        # and OS bytes. This is not a CoreOS boot/install fixture.
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            tree = root / 'tree'
            for name in ('coreos/miniso.dat', 'coreos/kargs.json', 'coreos/igninfo.json',
                         'images/ignition.img', 'images/pxeboot/rootfs.img',
                         'images/pxeboot/initrd.img', 'images/pxeboot/vmlinuz'):
                path = tree / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('synthetic ISO fixture: ' + name)
            (tree / 'images/efiboot.img').write_bytes(bytes(4096))
            (tree / 'EFI/fedora').mkdir(parents=True)
            (tree / 'EFI/fedora/grub.cfg').write_text("menuentry 'Fedora CoreOS (Live)' --class fedora {\nfixture\n}\n")
            upstream = root / 'upstream.iso'
            xorriso = Path(shutil.which('xorriso'))
            subprocess.run([str(xorriso), '-as', 'mkisofs', '-r', '-V', 'fixture',
                '-e', 'images/efiboot.img', '-no-emul-boot', '-o', str(upstream), str(tree)],
                check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            payload = root / 'payload'
            payload.mkdir()
            (payload / 'soda-install').write_text('synthetic non-executable console fixture')
            final = root / 'final.iso'
            self.module.remaster(xorriso, upstream, payload, final, root / 'remaster.log')
            receipt = self.module.verify_remaster(xorriso, upstream, final, payload)
            self.assertEqual(receipt['RemovedMetadata'], ['/coreos/miniso.dat'])
            self.assertEqual(receipt['BootMetadata']['volume'], 'fixture')
            primary = self.module.iso_files(xorriso, final, 'ecma119')
            self.assertIn('/COREOS/KARGS.JSO', primary)
            self.assertIn('/COREOS/IGNINFO.JSO', primary)
            self.assertNotIn('/COREOS/KARGS.JSON', primary)
            original = final.read_bytes()
            inventory = self.module.iso_files(xorriso, final)
            for name in ('/soda/soda-install', '/images/pxeboot/rootfs.img', '/images/efiboot.img', '/EFI/fedora/grub.cfg'):
                changed = bytearray(original)
                changed[inventory[name][0]] ^= 1
                final.write_bytes(changed)
                with self.assertRaises(ValueError):
                    self.module.verify_remaster(xorriso, upstream, final, payload)
            final.write_bytes(original)
            with self.assertRaises(FileExistsError):
                self.module.remaster(xorriso, upstream, payload, final, root / 'not-created.log')
            self.assertFalse((root / 'not-created.log').exists())
            self.assertEqual(final.read_bytes(), original)

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
        base = json.loads((ROOT / 'appliance/provisioning/base.json').read_text())
        self.assertEqual(data['systemd'], base['systemd'])
        self.assertEqual(data['storage']['files'][:-2], base['storage']['files'])
        identity, icon = data['storage']['files'][-2:]
        self.assertEqual(identity['path'], '/etc/os-release')
        self.assertIs(identity['overwrite'], True)
        self.assertEqual(identity['contents']['inline'], (ROOT / 'assets/branding/host/os-release').read_text())
        self.assertEqual(icon['contents']['inline'], (ROOT / 'assets/branding/source/soda-symbol.svg').read_text())
        self.assertNotIn('passwd', data)
        data['passwd'] = {'fixture': True}
        self.assertNotIn('passwd', self.module.public_config())

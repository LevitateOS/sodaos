"""Authored source/fixture checks, not native target or media evidence."""
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


def load(name, file):
    spec = importlib.util.spec_from_file_location(name, ROOT / file)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class Provisioning(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.root.chmod(0o700)
        self.module = load('provisioning_fixture', 'scripts/render-provisioning.py')
        self.public = self.root / 'operator.pub'
        self.public.write_text('ssh-ed25519 AAAA synthetic-public-fixture\n')
        self.password = self.root / 'password.hash'
        self.password.write_text('$6$synthetic$not-a-real-password-hash\n')
        self.password.chmod(0o600)

    def test_minimal_and_extension_profiles_are_distinct(self):
        for profile in ('minimal', 'extensions'):
            dest = self.root / (profile + '.bu')
            self.module.render(self.public, self.password, dest, 'soda-native-fixture', bootstrap=profile)
            data = json.loads(dest.read_text())
            self.assertEqual([u['name'] for u in data['passwd']['users']], ['root'])
            self.assertEqual('systemd' in data, profile == 'extensions')
            self.assertEqual(dest.stat().st_mode & 0o777, 0o600)
            self.assertEqual(next(f['contents']['inline'] for f in data['storage']['files'] if f['path'] == '/etc/hostname'), 'soda-native-fixture\n')
            with self.assertRaises(FileExistsError):
                self.module.render(self.public, self.password, dest)

    def test_private_modes_and_plaintext_are_rejected_before_output(self):
        dest = self.root / 'refused.bu'
        self.password.chmod(0o644)
        with self.assertRaises(ValueError):
            self.module.render(self.public, self.password, dest)
        self.assertFalse(dest.exists())
        self.password.chmod(0o600)
        self.password.write_text('synthetic plaintext\n')
        with self.assertRaises(ValueError):
            self.module.render(self.public, self.password, dest)
        self.assertFalse(dest.exists())

    def test_host_key_contents_never_become_process_arguments(self):
        key = self.root / 'host-key'
        key.write_text('synthetic-private-host-key\n')
        key.chmod(0o600)
        result = subprocess.CompletedProcess([], 0, stdout='ssh-ed25519 AAAA fixture\n', stderr='')
        dest = self.root / 'instance.bu'
        with patch.object(self.module.subprocess, 'run', return_value=result) as run:
            self.module.render(self.public, self.password, dest, 'soda-native-fixture', key)
        arguments = run.call_args.args[0]
        self.assertNotIn(key.read_text(), arguments)
        self.assertIn(str(key), arguments)
        data = json.loads(dest.read_text())
        private = next(f for f in data['storage']['files'] if f['path'] == '/etc/ssh/ssh_host_ed25519_key')
        self.assertEqual(private['mode'], 0o600)
        self.assertEqual(private['contents']['inline'], key.read_text())

    def test_public_bootstrap_has_no_identity_or_automatic_reboot(self):
        raw = (ROOT / 'appliance/provisioning/base.json').read_text()
        data = json.loads(raw)
        self.assertNotIn('passwd', data)
        self.assertNotIn('password_hash', raw)
        self.assertNotIn('reboot', raw)
        self.assertIn('rpm-ostree install', raw)


class OutsideContracts(unittest.TestCase):
    def test_support_tools_are_not_appliance_commands(self):
        for name in ('soda-artifacts', 'soda-acceptance'):
            self.assertTrue((ROOT / 'tools' / name / 'main.go').is_file())
            self.assertFalse((ROOT / 'cmd' / name).exists())
        build = (ROOT / 'scripts/build-native.sh').read_text()
        self.assertIn('podman save --format oci-archive', build)
        self.assertIn('--iidfile', build)
        self.assertIn('flock -n', build)
        self.assertNotIn('podman push', build)

    def test_installer_verifies_before_copy_and_retains_first_install_guard(self):
        source = (ROOT / 'scripts/install-native.sh').read_text()
        self.assertLess(source.index('soda-artifacts" verify'), source.index('tar -C'))
        self.assertLess(source.index('configure_network check'), source.index('tar -C'))
        self.assertIn('/etc/soda/install-started', source)
        self.assertIn('--no-same-owner --no-overwrite-dir', source)
        self.assertIn('podman tag "$id" "$reference"', source)
        self.assertNotIn('podman pull', source)

    def test_https_origin_refuses_credentials_and_downgrades(self):
        module = load('https_fixture', 'tests/installed/service-https.py')
        import argparse
        self.assertEqual(module.origin('https://example.test:443'), 'https://example.test:443/')
        for value in ('http://example.test', 'https://user:secret@example.test', 'https://example.test/?code=x', 'https://example.test/path', 'https://example.test:0'):
            with self.assertRaises(argparse.ArgumentTypeError):
                module.origin(value)

    def test_remote_dispatch_has_no_runtime_or_publication_phases(self):
        module = load('remote_fixture', 'internal/acceptance/remote_executor.py')
        import io
        request = {'Revision': 'a' * 40, 'Architecture': 'x86_64', 'Target': 'builder', 'Work': '/new/private', 'Phase': 'publish'}
        class Input:
            buffer = io.BytesIO(json.dumps(request).encode())
        with patch.object(module.sys, 'stdin', Input()), patch.object(module.platform, 'system', return_value='Linux'), patch.object(module.platform, 'machine', return_value='x86_64'), patch.object(module.platform, 'node', return_value='builder'), patch.object(module.subprocess, 'run') as run:
            with self.assertRaises(ValueError):
                module.main()
            run.assert_not_called()

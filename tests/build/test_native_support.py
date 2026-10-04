"""Authored source/fixture checks, not native target or media evidence."""

import importlib.util
import json
import os
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
    @classmethod
    def setUpClass(cls):
        build = subprocess.run(
            ['cargo', 'build', '-p', 'soda-stage-render', '--bin', 'soda-render-provisioning'],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        if build.returncode != 0:
            raise AssertionError(f'cannot build soda-render-provisioning: {build.stderr[-2000:]}')
        cls.render = str(ROOT / 'target/debug/soda-render-provisioning')

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.root.chmod(0o700)
        self.public = self.root / 'operator.pub'
        self.public.write_text('ssh-ed25519 AAAA synthetic-public-fixture\n')
        self.password = self.root / 'password.hash'
        self.password.write_text('$6$synthetic$not-a-real-password-hash\n')
        self.password.chmod(0o600)

    def render_to(self, dest, *extra, env=None):
        return subprocess.run(
            [
                self.render,
                '--operator-key-file',
                str(self.public),
                '--root-password-hash-file',
                str(self.password),
                *extra,
                '--out',
                str(dest),
            ],
            cwd=ROOT,
            capture_output=True,
            text=True,
            timeout=60,
            env=env,
        )

    def test_minimal_and_extension_profiles_are_distinct(self):
        for profile in ('minimal', 'extensions'):
            dest = self.root / (profile + '.bu')
            first = self.render_to(dest, '--hostname', 'soda-native-fixture', '--bootstrap', profile)
            self.assertEqual(first.returncode, 0, first.stderr[-2000:])
            data = json.loads(dest.read_text())
            self.assertEqual([u['name'] for u in data['passwd']['users']], ['root'])
            self.assertEqual('systemd' in data, profile == 'extensions')
            self.assertEqual(dest.stat().st_mode & 0o777, 0o600)
            self.assertEqual(
                next(f['contents']['inline'] for f in data['storage']['files'] if f['path'] == '/etc/hostname'),
                'soda-native-fixture\n',
            )
            before = dest.read_bytes()
            rerun = self.render_to(dest)
            self.assertEqual(rerun.returncode, 1, rerun.stderr[-2000:])
            self.assertEqual(
                rerun.stderr,
                'Private provisioning failed (FileExistsError); check paths/modes/key format.\n',
            )
            self.assertEqual(dest.read_bytes(), before)

    def test_private_modes_and_plaintext_are_rejected_before_output(self):
        dest = self.root / 'refused.bu'
        self.password.chmod(0o644)
        refused = self.render_to(dest)
        self.assertEqual(refused.returncode, 1, refused.stderr[-2000:])
        self.assertEqual(
            refused.stderr,
            'Private provisioning failed (ValueError); check paths/modes/key format.\n',
        )
        self.assertFalse(dest.exists())
        self.password.chmod(0o600)
        self.password.write_text('synthetic plaintext\n')
        refused = self.render_to(dest)
        self.assertEqual(refused.returncode, 1, refused.stderr[-2000:])
        self.assertFalse(dest.exists())

    def test_host_key_contents_never_become_process_arguments(self):
        key = self.root / 'host-key'
        key.write_text('synthetic-private-host-key\n')
        key.chmod(0o600)
        helper = self.root / 'fakebin'
        helper.mkdir()
        log = self.root / 'argv.log'
        keygen = helper / 'ssh-keygen'
        keygen.write_text(f'#!/bin/sh\nprintf "%s\\n" "$@" > {log}\nprintf "ssh-ed25519 AAAA fixture\\n"\n')
        keygen.chmod(0o755)
        dest = self.root / 'instance.bu'
        env = {**os.environ, 'PATH': str(helper) + os.pathsep + os.environ['PATH']}
        done = self.render_to(dest, '--hostname', 'soda-native-fixture', '--ssh-host-key-file', str(key), env=env)
        self.assertEqual(done.returncode, 0, done.stderr[-2000:])
        arguments = log.read_text()
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

    def test_installed_console_welcome_runs_before_login(self):
        data = json.loads((ROOT / 'appliance/provisioning/candidate.json').read_text())
        units = {u['name']: u for u in data['systemd']['units']}
        self.assertTrue(units['soda-console.service']['enabled'])
        body = (ROOT / 'appliance/services/soda-console.service').read_text()
        self.assertIn('Before=getty@tty1.service', body)
        self.assertIn('50-soda.issue', body)

    def test_installed_system_defaults_to_password_ssh(self):
        raw = (ROOT / 'appliance/provisioning/candidate.json').read_text()
        data = json.loads(raw)
        dropin = next(f for f in data['storage']['files'] if f['path'].startswith('/etc/ssh/sshd_config.d/'))
        self.assertNotIn('key-only', dropin['path'])
        config = dropin['contents']['inline']
        self.assertIn('PermitRootLogin yes', config)
        self.assertIn('PasswordAuthentication yes', config)
        self.assertNotIn('prohibit-password', config)
        self.assertNotIn(' no\n', config)


class OutsideContracts(unittest.TestCase):
    def test_support_tools_are_not_appliance_commands(self):
        for name in ('soda-artifacts', 'soda-acceptance'):
            self.assertTrue((ROOT / 'tools' / name / 'main.go').is_file())
            self.assertFalse((ROOT / 'cmd' / name).exists())
        self.assertTrue((ROOT / 'tools' / 'soda-build' / 'main.go').is_file())
        self.assertFalse((ROOT / 'scripts/build-native.sh').exists())
        self.assertFalse((ROOT / 'tools/soda-host-image').exists())
        producer = (ROOT / 'internal/release/build/production.go').read_text()
        self.assertIn('"save", "--format=oci-archive"', producer)
        self.assertIn('--iidfile', producer)
        self.assertNotIn('"push"', producer)

    def test_boot_binds_browser_services_without_quadlet_enable(self):
        # Quadlet-generated units refuse enable; the console unit binds them.
        unit = (ROOT / 'appliance/services/soda-console.service').read_text()
        self.assertIn('Wants=forgejo.service soda-dashboard.service soda-proxy.service', unit)
        self.assertNotIn('enable --now', unit)
        for name in ('soda-dashboard.container', 'soda-proxy.container'):
            container = (ROOT / 'appliance/services' / name).read_text()
            self.assertIn('ConditionPathExists=/etc/soda/activated', container)

    # Origin validation moved with service-https.py to Go; see
    # internal/acceptance/service_https_test.go TestHTTPSOrigin.

    def test_remote_dispatch_has_no_runtime_or_publication_phases(self):
        module = load('remote_fixture', 'internal/acceptance/remote_executor.py')
        import io

        request = {
            'Revision': 'a' * 40,
            'Architecture': 'x86_64',
            'Target': 'builder',
            'Work': '/new/private',
            'Phase': 'publish',
        }

        class Input:
            buffer = io.BytesIO(json.dumps(request).encode())

        with (
            patch.object(module.sys, 'stdin', Input()),
            patch.object(module.platform, 'system', return_value='Linux'),
            patch.object(module.platform, 'machine', return_value='x86_64'),
            patch.object(module.platform, 'node', return_value='builder'),
            patch.object(module.subprocess, 'run') as run,
        ):
            with self.assertRaises(ValueError):
                module.main()
            run.assert_not_called()

"""Binary staging checks with local fixtures; no compilation or CLI execution."""
import hashlib
import importlib.util
import io
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import patch


class ProjectTools(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        source = Path(__file__).resolve().parents[2] / 'scripts/fetch-tea.py'
        spec = importlib.util.spec_from_file_location('tea_fixture', source)
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        self.module.ROOT = self.root
        self.out = self.root / 'output'
        self.license = self.root / 'project-os/licenses/tea-LICENSE'
        self.license.parent.mkdir(parents=True)
        self.license.write_bytes(b'fixture license\n')
        self.binaries = {}
        lock = 'version = "0.16.0"\nlicense_sha256 = "' + hashlib.sha256(self.license.read_bytes()).hexdigest() + '"\n'
        for arch, machine in [('x86_64', 62), ('aarch64', 183)]:
            body = bytearray(64)
            body[:6] = b'\x7fELF\x02\x01'
            body[18:20] = machine.to_bytes(2, 'little')
            self.binaries[arch] = bytes(body)
            lock += f'\n[linux.{arch}]\nurl = "https://example.test/tea-{arch}"\nsha256 = "{hashlib.sha256(body).hexdigest()}"\n'
        self.lock = self.root / 'project-os/locks/tea-binary.toml'
        self.lock.parent.mkdir()
        self.lock.write_text(lock)

    def test_each_architecture_stages_exact_upstream_bytes_and_license(self):
        for arch, body in self.binaries.items():
            with self.subTest(arch=arch), patch.object(self.module.urllib.request, 'urlopen', return_value=io.BytesIO(body)) as download:
                out = self.out / arch
                self.module.fetch(arch, out)
                download.assert_called_once()
                self.assertEqual(download.call_args.args[0].full_url, f'https://example.test/tea-{arch}')
                self.assertEqual((out / 'bin/tea').read_bytes(), body)
                self.assertEqual((out / 'bin/tea').stat().st_mode & 0o777, 0o755)
                self.assertEqual((out / 'licenses/tea/LICENSE').read_bytes(), self.license.read_bytes())
                self.assertEqual((out / 'licenses/tea/LICENSE').stat().st_mode & 0o777, 0o644)

    def test_corrupt_download_leaves_no_stage(self):
        with patch.object(self.module.urllib.request, 'urlopen', return_value=io.BytesIO(b'corrupt')):
            with self.assertRaisesRegex(ValueError, 'binary checksum mismatch'):
                self.module.fetch('x86_64', self.out)
        self.assertFalse(self.out.exists())

    def test_incorrect_architecture_in_manifest_leaves_no_stage(self):
        old = hashlib.sha256(self.binaries['x86_64']).hexdigest()
        new = hashlib.sha256(self.binaries['aarch64']).hexdigest()
        self.lock.write_text(self.lock.read_text().replace(old, new))
        with patch.object(self.module.urllib.request, 'urlopen', return_value=io.BytesIO(self.binaries['aarch64'])):
            with self.assertRaisesRegex(ValueError, 'requested architecture'):
                self.module.fetch('x86_64', self.out)
        self.assertFalse(self.out.exists())

    def test_changed_license_fails_before_network(self):
        self.license.write_bytes(b'changed')
        with patch.object(self.module.urllib.request, 'urlopen') as download:
            with self.assertRaisesRegex(ValueError, 'license checksum mismatch'):
                self.module.fetch('x86_64', self.out)
            download.assert_not_called()
        self.assertFalse(self.out.exists())

    def test_existing_output_is_preserved(self):
        self.out.mkdir()
        existing = self.out / 'retained'
        existing.write_bytes(b'keep')
        with patch.object(self.module.urllib.request, 'urlopen') as download:
            with self.assertRaisesRegex(ValueError, 'output already exists'):
                self.module.fetch('x86_64', self.out)
            download.assert_not_called()
        self.assertEqual(existing.read_bytes(), b'keep')

    def test_network_failure_leaves_no_stage(self):
        with patch.object(self.module.urllib.request, 'urlopen', side_effect=urllib.error.URLError('unavailable')):
            with self.assertRaises(urllib.error.URLError):
                self.module.fetch('x86_64', self.out)
        self.assertFalse(self.out.exists())


if __name__ == '__main__':
    unittest.main()

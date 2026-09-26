"""Binary staging checks with local fixtures; no compilation or CLI execution."""

import hashlib
import importlib.util
import io
import json
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import patch


def elf_body(machine):
    body = bytearray(64)
    body[:6] = b'\x7fELF\x02\x01'
    body[18:20] = machine.to_bytes(2, 'little')
    return bytes(body)


class ProjectTools(unittest.TestCase):
    ARCHES = {'x86_64': ('amd64', 62), 'aarch64': ('arm64', 183)}
    VERSION = '0.99.1'
    TAG = 'v0.99.1'

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
        self.bodies = {arch: elf_body(machine) for arch, (_, machine) in self.ARCHES.items()}
        self.license = b'fixture license\n'
        self.corrupt = None

    def dispatch(self, request, *args, **kwargs):
        url = request.full_url if hasattr(request, 'full_url') else request
        if url == self.module.RELEASES_API:
            if self.corrupt == 'tag':
                return io.BytesIO(json.dumps({'tag_name': 'yesterday'}).encode())
            return io.BytesIO(json.dumps({'tag_name': self.TAG}).encode())
        if url.endswith('/checksums.txt'):
            return io.BytesIO(self.checksums())
        if url.endswith('/LICENSE'):
            if self.corrupt == 'license':
                return io.BytesIO(b'')
            return io.BytesIO(self.license)
        for arch, (gnu, _) in self.ARCHES.items():
            if url.endswith(f'tea-{self.VERSION}-linux-{gnu}'):
                if self.corrupt == 'binary':
                    return io.BytesIO(b'corrupt')
                if self.corrupt == 'arch':
                    other = 'aarch64' if arch == 'x86_64' else 'x86_64'
                    return io.BytesIO(self.bodies[other])
                return io.BytesIO(self.bodies[arch])
        raise AssertionError(f'unexpected fetch: {url}')

    def checksums(self):
        lines = ''
        for arch, (gnu, _) in self.ARCHES.items():
            body = self.bodies[arch]
            if self.corrupt == 'arch':
                other = 'aarch64' if arch == 'x86_64' else 'x86_64'
                body = self.bodies[other]
            lines += f'{hashlib.sha256(body).hexdigest()}  tea-{self.VERSION}-linux-{gnu}\n'
        return lines.encode()

    def fetch(self, arch, out=None):
        with patch.object(self.module.urllib.request, 'urlopen', side_effect=self.dispatch):
            self.module.fetch(arch, out or self.out / arch)

    def test_each_architecture_stages_exact_upstream_bytes_and_license(self):
        for arch, body in self.bodies.items():
            with self.subTest(arch=arch):
                self.fetch(arch)
                out = self.out / arch
                self.assertEqual((out / 'bin/tea').read_bytes(), body)
                self.assertEqual((out / 'bin/tea').stat().st_mode & 0o777, 0o755)
                self.assertEqual((out / 'licenses/tea/LICENSE').read_bytes(), self.license)
                self.assertEqual((out / 'licenses/tea/LICENSE').stat().st_mode & 0o777, 0o644)

    def test_corrupt_download_leaves_no_stage(self):
        self.corrupt = 'binary'
        with self.assertRaisesRegex(ValueError, 'binary checksum mismatch'):
            self.fetch('x86_64')
        self.assertFalse(self.out.exists())

    def test_wrong_architecture_binary_leaves_no_stage(self):
        self.corrupt = 'arch'
        with self.assertRaisesRegex(ValueError, 'requested architecture'):
            self.fetch('x86_64')
        self.assertFalse(self.out.exists())

    def test_non_version_tag_leaves_no_stage(self):
        self.corrupt = 'tag'
        with self.assertRaisesRegex(ValueError, 'not a version tag'):
            self.fetch('x86_64')
        self.assertFalse(self.out.exists())

    def test_empty_license_leaves_no_stage(self):
        self.corrupt = 'license'
        with self.assertRaisesRegex(ValueError, 'license download is empty'):
            self.fetch('x86_64')
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
        out = self.out / 'x86_64'
        with patch.object(self.module.urllib.request, 'urlopen', side_effect=urllib.error.URLError('unavailable')):
            with self.assertRaises(urllib.error.URLError):
                self.module.fetch('x86_64', out)
        self.assertFalse(self.out.exists())


if __name__ == '__main__':
    unittest.main()

"""Packaging/probe guards only; not an installed RPM or compiler claim."""
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class FoundationContracts(unittest.TestCase):
    def test_recipe_declares_native_development_foundation(self):
        recipe = (ROOT / 'project-os/Containerfile').read_text().replace('\\\n', ' ')
        match = re.search(r'RUN dnf -y install (.*?) && dnf clean all', recipe)
        self.assertIsNotNone(match)
        packages = set(match.group(1).split())
        self.assertLessEqual({'gcc', 'gcc-c++', 'glibc-devel', 'libstdc++-devel', 'make', 'cmake', 'ninja-build', 'pkgconf-pkg-config', 'binutils', 'gdb', 'strace', 'openssl-devel', 'zlib-devel', 'rsync', 'iproute', 'iputils', 'bind-utils', 'lsof', 'jq'}, packages)
        self.assertNotIn('--nogpgcheck', recipe)
        self.assertNotIn('dnf upgrade', recipe)

    def test_installed_probe_requires_scope_before_writes(self):
        script = ROOT / 'tests/installed/project-foundation.sh'
        subprocess.run(['/bin/sh', '-n', str(script)], check=True)
        with tempfile.TemporaryDirectory() as temporary:
            result = subprocess.run(['/bin/sh', str(script)], env={'PATH': os.defpath, 'TMPDIR': temporary}, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(b'SODA_NATIVE_VALIDATE', result.stderr)
            self.assertEqual(list(Path(temporary).iterdir()), [])
        source = script.read_text()
        self.assertIn('test "$(id -u)" != 0', source)
        self.assertIn('HOME="$work/home"', source)
        self.assertIn('OpenSSL::Crypto ZLIB::ZLIB', source)
        self.assertIn('--return-child-result', source)
        self.assertNotIn('sudo ', source)
        self.assertNotIn('rm -', source)

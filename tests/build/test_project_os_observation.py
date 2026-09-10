"""Real parser/filesystem checks; no project or host OS is changed."""
import importlib.util
import os
from pathlib import Path
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parents[2] / 'internal/host/project_os.py'
spec = importlib.util.spec_from_file_location('soda_os_observation', SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class OSObservation(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.release = self.root / 'os-release'

    def test_selected_fields_and_standard_symlink(self):
        self.release.write_text('ID="rocky"\nVERSION_ID="9.7"\nPRETTY_NAME="Rocky Linux 9.7"\nOTHER="not returned"\n')
        alias = self.root / 'link'
        alias.symlink_to('os-release')
        self.assertEqual(module.observe(alias), {'id': 'rocky', 'version': '9.7', 'name': 'Rocky Linux 9.7'})

    def test_shell_syntax_never_executes(self):
        marker = self.root / 'must-not-exist'
        self.release.write_text(f'ID=rocky\nVERSION_ID=9.7\nPRETTY_NAME="$(touch {marker})"\n')
        self.assertIn('$(touch ', module.observe(self.release)['name'])
        self.assertFalse(marker.exists())

    def test_ambiguous_malformed_or_oversized_fields_are_unknown(self):
        for contents in [b'ID=rocky\nID=fedora\nVERSION_ID=9', b'ID=rocky', b'ID=rocky\nVERSION_ID=9\nPRETTY_NAME="oops', b'ID=rocky\nVERSION_ID=9\nPRETTY_NAME="a\x00b"', b'ID=rocky\nVERSION_ID=9\nPRETTY_NAME="\xff"', b'ID=rocky\nVERSION_ID=9\nPRETTY_NAME="' + b'x' * 257 + b'"', b'x' * 4097]:
            with self.subTest(contents=contents[:24]):
                self.release.write_bytes(contents)
                with self.assertRaises((ValueError, UnicodeError)):
                    module.observe(self.release)

    def test_special_file_refused_without_waiting_for_a_writer(self):
        os.mkfifo(self.release)
        with self.assertRaises(ValueError):
            module.observe(self.release)

    def test_directory_missing_file_and_missing_version_are_not_inferred(self):
        for path in [self.root, self.release]:
            with self.assertRaises((OSError, ValueError)):
                module.observe(path)
        self.release.write_text('ID=rocky\n')
        with self.assertRaises(ValueError):
            module.observe(self.release)

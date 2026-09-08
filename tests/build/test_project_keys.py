"""Real atomic key-file updates in owned temp dirs; mocked root/account metadata.

Never creates host accounts or accesses appliance/project paths.
"""
import hashlib
import importlib.util
import os
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2] / 'internal/host'
identity = types.ModuleType('project_terminal')
exec((ROOT / 'project_terminal.py').read_text(), identity.__dict__)
with patch.dict(sys.modules, project_terminal=identity):
    spec = importlib.util.spec_from_file_location('project_keys', ROOT / 'project_keys.py')
    keys = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(keys)


class ProjectKeys(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.file = self.root / 'alice'
        self.file.write_bytes(b'ssh-ed25519 YWJj\n')  # Format fixture, not a cryptographic SSH key.
        self.file.chmod(0o644)
        self.other = self.root / 'bob'
        self.other.write_bytes(b'preserve')
        self.addCleanup(patch.stopall)
        patch.object(keys, 'account_for', return_value=object()).start()
        patch.object(keys, 'directory', side_effect=lambda _: os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)).start()
        original = os.fstat
        def root_metadata(fd):
            info = list(original(fd)); info[4] = info[5] = 0
            return os.stat_result(info)
        patch.object(keys.os, 'fstat', side_effect=root_metadata).start()

    def request(self, apply=False, revision='', values=None):
        return {'login': 'alice', 'identity': 1, 'apply': apply, 'revision': revision, 'keys': values or []}

    def test_preview_apply_empty_and_preservation(self):
        before = keys.update(self.request())
        self.assertEqual(before['revision'], hashlib.sha256(self.file.read_bytes()).hexdigest())
        result = keys.update(self.request(True, before['revision'], ['ssh-ed25519 ZGVm']))
        self.assertEqual(result['keys'], ['ssh-ed25519 ZGVm'])
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 ZGVm\n')
        self.assertEqual(keys.update(self.request(True, result['revision']))['keys'], [])
        self.assertEqual(self.file.read_bytes(), b'')
        self.assertEqual(self.other.read_bytes(), b'preserve')
        self.assertEqual(sorted(p.name for p in self.root.iterdir()), ['alice', 'bob'])

    def test_changed_revision_never_writes(self):
        before = self.file.read_bytes()
        with self.assertRaises(ValueError): keys.update(self.request(True, '0' * 64))
        self.assertEqual(self.file.read_bytes(), before)

    def test_unmanaged_content_symlink_hardlink_and_mode_refused(self):
        for content in [b'# operator annotation\nssh-ed25519 YWJj\n', b'command="id" ssh-ed25519 YWJj\n', b'ssh-ed25519 YWJj', b'\xff\n']:
            self.file.write_bytes(content)
            with self.assertRaises(ValueError): keys.update(self.request())
            self.assertEqual(self.file.read_bytes(), content)
        self.file.unlink(); self.file.symlink_to(self.other)
        with self.assertRaises(OSError): keys.update(self.request())
        self.file.unlink(); os.link(self.other, self.file)
        with self.assertRaises(ValueError): keys.update(self.request())
        self.file.unlink(); self.file.write_bytes(b'ssh-ed25519 YWJj\n'); self.file.chmod(0o666)
        with self.assertRaises(ValueError): keys.update(self.request())
        self.assertEqual(self.other.read_bytes(), b'preserve')

    def test_replace_failure_preserves_original_and_cleans_only_owned_temp(self):
        before = keys.update(self.request())
        with patch.object(keys.os, 'replace', side_effect=OSError('fixture')):
            with self.assertRaises(OSError): keys.update(self.request(True, before['revision']))
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')
        self.assertEqual(sorted(p.name for p in self.root.iterdir()), ['alice', 'bob'])

    def test_account_refusal_precedes_file_access(self):
        with patch.object(keys, 'account_for', side_effect=ValueError('mismatch')):
            with self.assertRaises(ValueError): keys.update(self.request())
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')

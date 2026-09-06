import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('project_state', ROOT / 'tests/installed/project-state.py')
state = importlib.util.module_from_spec(spec)
spec.loader.exec_module(state)


class SnapshotContracts(unittest.TestCase):
    def test_private_metadata_does_not_export_hash_or_content(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory) / 'private-input'
            p.write_text('synthetic-sensitive-input')
            p.chmod(0o600)
            entry = state.entry(p, contents=False)
            self.assertNotIn('sha256', entry)
            self.assertEqual(entry['mode'], 0o600)
            self.assertEqual(entry['size'], p.stat().st_size)

    def test_required_missing_file_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(FileNotFoundError):
                state.entry(Path(directory) / 'missing')

    def test_public_hash_changes_and_symlink_is_not_followed(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory) / 'public'; p.write_text('before')
            before = state.entry(p)
            p.write_text('after')
            self.assertNotEqual(before['sha256'], state.entry(p)['sha256'])
            link = Path(directory) / 'link'; link.symlink_to(p)
            self.assertEqual(state.entry(link)['link'], str(p))
            self.assertNotIn('sha256', state.entry(link))


if __name__ == '__main__':
    unittest.main()

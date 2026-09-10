"""Local fixtures only; never invoke the installed host entrypoint or native CLI."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[1] / 'installed' / 'runner-state.py'
spec = importlib.util.spec_from_file_location('runner_state', SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class RunnerState(unittest.TestCase):
    def test_snapshot_hashes_private_data_without_returning_it(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'one'
            root.mkdir()
            state = root / 'state'
            state.mkdir(mode=0o700)
            token = state / 'forgejo-token'
            token.write_text('synthetic-private-registration-token')
            token.chmod(0o600)
            (root / 'descriptor.json').write_text('{}')
            configuration = state / 'forgejo-runner.yml'
            configuration.write_text('synthetic-config')
            configuration.chmod(0o600)
            person = SimpleNamespace(pw_uid=os.getuid(), pw_gid=os.getgid(), pw_dir=str(state), pw_shell='/usr/sbin/nologin')
            first = module.runner_state('one', root.parent, lambda _: person)
            self.assertEqual(first, module.runner_state('one', root.parent, lambda _: person))
            self.assertNotIn('synthetic-private', repr(first))
            self.assertNotIn('forgejo-token', repr(first))
            token.write_text('later private write')
            self.assertNotEqual(first['tree'], module.runner_state('one', root.parent, lambda _: person)['tree'])
            self.assertEqual(token.read_text(), 'later private write')
            token.chmod(0o644)
            with self.assertRaises(RuntimeError):
                module.runner_state('one', root.parent, lambda _: person)

    def test_links_do_not_escape_the_owned_tree(self):
        with tempfile.TemporaryDirectory() as directory:
            parent = Path(directory)
            root = parent / 'one'
            root.mkdir()
            outside = parent / 'private-outside'
            outside.write_text('outside credential')
            (root / 'link').symlink_to(outside)
            first = module.tree_digest(root)
            outside.write_text('changed outside credential')
            self.assertEqual(first, module.tree_digest(root))
            (parent / 'alias').symlink_to(root, target_is_directory=True)
            with self.assertRaises(RuntimeError):
                module.runner_state('alias', parent, lambda _: SimpleNamespace())
            self.assertEqual(outside.read_text(), 'changed outside credential')

    def test_partial_native_state_is_unavailable_not_absent(self):
        def missing(_):
            raise KeyError('missing')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.assertEqual(module.runner_state('one', root, missing), {'present': False})
            with self.assertRaises(RuntimeError):
                module.runner_state('one', root, lambda _: SimpleNamespace())
            (root / 'one').mkdir()
            with self.assertRaises(RuntimeError):
                module.runner_state('one', root, missing)
            with self.assertRaises(ValueError):
                module.runner_state('../one', root, missing)

    def test_fifo_and_changing_file_are_refused_without_blocking(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'one'
            root.mkdir()
            fifo = root / 'live-pipe'
            os.mkfifo(fifo)
            with self.assertRaises(RuntimeError):
                module.tree_digest(root)
            fifo.unlink()
            work = root / 'work'
            work.write_bytes(b'initial')
            read = os.read
            changed = False
            def changing(fd, count):
                nonlocal changed
                data = read(fd, count)
                if not changed:
                    changed = True
                    work.write_bytes(b'concurrent later write')
                return data
            with patch.object(module.os, 'read', side_effect=changing):
                with self.assertRaises(RuntimeError):
                    module.tree_digest(root)
            self.assertEqual(work.read_bytes(), b'concurrent later write')

    def test_exact_job_proof_rejects_wrong_identity_extra_fields_and_links(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            work = root / 'one' / 'state' / 'work'
            work.mkdir(parents=True)
            person = SimpleNamespace(pw_uid=os.getuid(), pw_name='soda-runner-one')
            account = lambda _: person
            self.assertIsNone(module.job_proof('one', 'unique', root, account))
            record = dict(observation='unique', account=person.pw_name, uid=person.pw_uid,
                          pid=999999999, start='1', steps=2)
            proof = work / 'soda-native-proof-unique.json'
            proof.write_text(json.dumps(record))
            proof.chmod(0o600)
            self.assertEqual(module.job_proof('one', 'unique', root, account), {**record, 'alive': False})
            for change in ({'observation': 'other'}, {'account': 'root'}, {'pid': 0}, {'steps': 3}, {'token': 'synthetic-secret'}):
                proof.write_text(json.dumps({**record, **change}))
                with self.assertRaises(RuntimeError):
                    module.job_proof('one', 'unique', root, account)
            proof.unlink()
            outside = root / 'outside'
            outside.write_text('must not read')
            proof.symlink_to(outside)
            with self.assertRaises(OSError):
                module.job_proof('one', 'unique', root, account)
            self.assertEqual(outside.read_text(), 'must not read')

    def test_gate_precedes_native_commands_or_state_reads(self):
        with patch.object(module, 'command') as command, patch.object(module, 'runner_state') as state:
            with patch.dict(os.environ, {'SODA_NATIVE_VALIDATE': 'other-target'}):
                with self.assertRaises(ValueError):
                    module.main(['fixture', 'one'])
            with self.assertRaises(ValueError):
                module.main([])
            command.assert_not_called()
            state.assert_not_called()


if __name__ == '__main__':
    unittest.main()

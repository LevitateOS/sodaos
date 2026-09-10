"""Real atomic key-file updates in owned temp dirs; mocked root/account metadata.

Never creates host accounts or accesses appliance/project paths.
"""
from contextlib import contextmanager, redirect_stderr, redirect_stdout
import fcntl
import hashlib
import io
import importlib.util
import multiprocessing
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
        patch.object(keys.os, 'geteuid', return_value=0).start()
        patch.object(keys, 'account_for', return_value=object()).start()
        patch.object(keys, 'directory', side_effect=lambda _: os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)).start()
        original = os.fstat
        def root_metadata(fd):
            info = original(fd)
            fields = {name: getattr(info, name) for name in dir(info) if name.startswith('st_')}
            fields.update(st_uid=0, st_gid=0)
            return types.SimpleNamespace(**fields)
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

    def test_flush_precedes_file_sync_and_publish(self):
        revision = keys.update(self.request())['revision']
        native_sync, native_replace = os.fsync, os.replace
        events = []
        desired = b'ssh-ed25519 ZGVm\n'
        def sync(fd):
            if keys.stat.S_ISREG(os.fstat(fd).st_mode):
                pending, = self.root.glob('.soda-keys-*')
                self.assertEqual(pending.read_bytes(), desired)
                self.assertEqual(pending.stat().st_mode & 0o777, 0o644)
                events.append('file sync')
            else:
                events.append('directory sync')
            native_sync(fd)
        def publish(*args, **kwargs):
            self.assertEqual(events, ['file sync'])
            events.append('publish')
            native_replace(*args, **kwargs)
        with patch.object(keys.os, 'fsync', side_effect=sync), patch.object(keys.os, 'replace', side_effect=publish):
            keys.update(self.request(True, revision, ['ssh-ed25519 ZGVm']))
        self.assertEqual(events, ['file sync', 'publish', 'directory sync'])

    def test_occupied_temporary_name_is_not_owned_or_removed(self):
        occupied = self.root / '.soda-keys-collision'
        occupied.write_bytes(b'other writer evidence')
        revision = keys.update(self.request())['revision']
        with patch.object(keys.uuid, 'uuid4', return_value=types.SimpleNamespace(hex='collision')):
            with self.assertRaises(FileExistsError):
                keys.update(self.request(True, revision))
        self.assertEqual(occupied.read_bytes(), b'other writer evidence')
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')

    @contextmanager
    def writer_process(self, operation):
        # POSIX project helpers: fork inherits only test-owned paths/root doubles.
        # Pipes are event barriers; no timing sleeps or real host accounts.
        context = multiprocessing.get_context('fork')
        parent, child = context.Pipe()
        def worker():
            parent.close()
            try:
                operation(child)
            finally:
                child.close()
        process = context.Process(target=worker)
        process.start()
        child.close()
        try:
            yield parent
        finally:
            parent.close()
            process.join(5)
            if process.is_alive():
                process.terminate()  # This exact test-owned child only.
                process.join(5)
                self.fail('writer did not finish')
            self.assertEqual(process.exitcode, 0)

    def test_lock_precedes_account_observation(self):
        fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            with patch.object(keys, 'account_for') as account:
                with self.assertRaises(BlockingIOError):
                    keys.update(self.request())
                account.assert_not_called()
        finally:
            os.close(fd)

    def test_apply_excludes_other_soda_and_native_writers_through_publication(self):
        revision = keys.update(self.request())['revision']
        def soda_writer(pipe):
            native_replace = os.replace
            def published(*args, **kwargs):
                native_replace(*args, **kwargs)
                pipe.send('published, still locked')
                pipe.recv()
            with patch.object(keys.os, 'replace', side_effect=published):
                result = keys.update(self.request(True, revision, ['ssh-ed25519 ZGVm']))
            pipe.send(result['keys'])
        with self.writer_process(soda_writer) as pipe:
            self.assertTrue(pipe.poll(5))
            self.assertEqual(pipe.recv(), 'published, still locked')
            with self.assertRaises(BlockingIOError):
                keys.update(self.request(True, revision))
            fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
            try:
                with self.assertRaises(BlockingIOError):
                    fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            finally:
                os.close(fd)
            pipe.send('finish verification')
            self.assertTrue(pipe.poll(5))
            self.assertEqual(pipe.recv(), ['ssh-ed25519 ZGVm'])
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 ZGVm\n')

    def test_cooperating_native_edits_before_admission_and_content_revision(self):
        for edit in ('append', 'edit', 'replace', 'same-bytes replacement'):
            with self.subTest(edit=edit):
                original = b'ssh-ed25519 YWJj\n'
                self.file.write_bytes(original)
                revision = keys.update(self.request())['revision']
                before_inode = self.file.stat().st_ino
                later = original if edit == 'same-bytes replacement' else b'ssh-ed25519 ZGVm\n'
                if edit == 'append':
                    later = original + later
                def native_writer(pipe):
                    fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
                    try:
                        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
                        pipe.send('locked')
                        pipe.recv()
                        if edit in ('replace', 'same-bytes replacement'):
                            pending = self.root / 'native-pending'
                            pending.write_bytes(later)
                            pending.chmod(0o644)
                            os.replace(pending, self.file)
                        else:
                            self.file.write_bytes(later)
                        os.fsync(fd)
                    finally:
                        os.close(fd)
                    pipe.send('finished')
                with self.writer_process(native_writer) as pipe:
                    self.assertTrue(pipe.poll(5))
                    self.assertEqual(pipe.recv(), 'locked')
                    with self.assertRaises(BlockingIOError):
                        keys.update(self.request(True, revision))
                    pipe.send('edit')
                    self.assertTrue(pipe.poll(5))
                    self.assertEqual(pipe.recv(), 'finished')
                self.assertEqual(self.file.read_bytes(), later)
                if edit == 'same-bytes replacement':
                    self.assertNotEqual(self.file.stat().st_ino, before_inode)
                    self.assertEqual(keys.update(self.request(True, revision))['keys'], [])
                else:
                    with self.assertRaisesRegex(ValueError, 'changed since preview'):
                        keys.update(self.request(True, revision))
                    self.assertEqual(self.file.read_bytes(), later)
                self.assertEqual(sorted(p.name for p in self.root.iterdir()), ['alice', 'bob'])

    def test_noncooperating_writer_can_bypass_advisory_lock(self):
        # Deliberately demonstrate the limitation, NOT universal CAS/safety proof.
        # This test-owned process has owner write access, analogous to project root.
        revision = keys.update(self.request())['revision']
        def ignores_lock(pipe):
            pipe.send('ready')
            pipe.recv()
            self.file.write_bytes(b'')  # Native revocation ignores the directory lock.
            pipe.send('revoked')
        native_replace = os.replace
        with self.writer_process(ignores_lock) as pipe:
            self.assertTrue(pipe.poll(5))
            self.assertEqual(pipe.recv(), 'ready')
            def race_after_last_check(*args, **kwargs):
                pipe.send('write after final check')
                self.assertTrue(pipe.poll(5))
                self.assertEqual(pipe.recv(), 'revoked')
                native_replace(*args, **kwargs)
            with patch.object(keys.os, 'replace', side_effect=race_after_last_check):
                keys.update(self.request(True, revision, ['ssh-ed25519 YWJj']))
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')

    def test_nanosecond_drift_during_update_is_not_hidden_by_metadata_double(self):
        revision = keys.update(self.request())['revision']
        before = self.file.stat()
        native_sync = os.fsync
        def change_metadata(fd):
            native_sync(fd)
            if keys.stat.S_ISREG(os.fstat(fd).st_mode):
                os.utime(self.file, ns=(before.st_atime_ns, before.st_mtime_ns + 1))
        with patch.object(keys.os, 'fsync', side_effect=change_metadata):
            with self.assertRaisesRegex(ValueError, 'during update'):
                keys.update(self.request(True, revision))
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')
        self.assertEqual(self.file.stat().st_mtime_ns, before.st_mtime_ns + 1)

    def test_write_flush_sync_and_verification_failures_do_not_restore_snapshots(self):
        for failure in ('write', 'flush', 'file sync', 'directory sync', 'verification'):
            with self.subTest(failure=failure):
                self.file.write_bytes(b'ssh-ed25519 YWJj\n')
                revision = keys.update(self.request())['revision']
                native_open, native_sync, native_read = os.fdopen, os.fsync, keys.read_keys
                @contextmanager
                def opened(fd, mode):
                    with native_open(fd, mode) as stream:
                        def write(data):
                            if failure == 'write':
                                stream.write(data[:1])
                                raise OSError('synthetic partial write')
                            return stream.write(data)
                        def flush():
                            if failure == 'flush':
                                raise OSError('synthetic flush failure')
                            stream.flush()
                        yield types.SimpleNamespace(write=write, flush=flush, fileno=stream.fileno)
                def sync(fd):
                    regular = keys.stat.S_ISREG(os.fstat(fd).st_mode)
                    if (failure == 'file sync' and regular) or (failure == 'directory sync' and not regular):
                        raise OSError('synthetic sync failure')
                    native_sync(fd)
                reads = 0
                def read(*args):
                    nonlocal reads
                    reads += 1
                    if failure == 'verification' and reads == 3:
                        raise OSError('synthetic post-publication read failure')
                    return native_read(*args)
                with patch.object(keys.os, 'fdopen', side_effect=opened), patch.object(keys.os, 'fsync', side_effect=sync), patch.object(keys, 'read_keys', side_effect=read):
                    with self.assertRaises(OSError):
                        keys.update(self.request(True, revision, ['ssh-ed25519 ZGVm']))
                published = failure in ('directory sync', 'verification')
                self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 ZGVm\n' if published else b'ssh-ed25519 YWJj\n')
                self.assertEqual(sorted(p.name for p in self.root.iterdir()), ['alice', 'bob'])
                keys.update(self.request())  # Every failure releases the lock.

    def test_later_cooperating_write_survives_uncertain_publication(self):
        revision = keys.update(self.request())['revision']
        def native_writer(pipe):
            fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
            try:
                pipe.send('ready')
                pipe.recv()
                fcntl.flock(fd, fcntl.LOCK_EX)
                self.file.write_bytes(b'')
                os.fsync(fd)
            finally:
                os.close(fd)
            pipe.send('later revocation')
        with self.writer_process(native_writer) as pipe:
            self.assertTrue(pipe.poll(5))
            self.assertEqual(pipe.recv(), 'ready')
            native_sync = os.fsync
            def fail_after_publish(fd):
                if keys.stat.S_ISDIR(os.fstat(fd).st_mode):
                    pipe.send('wait for lock release')
                    raise OSError('uncertain directory sync')
                native_sync(fd)
            with patch.object(keys.os, 'fsync', side_effect=fail_after_publish):
                with self.assertRaises(OSError):
                    keys.update(self.request(True, revision, ['ssh-ed25519 ZGVm']))
            self.assertTrue(pipe.poll(5))
            self.assertEqual(pipe.recv(), 'later revocation')
        self.assertEqual(self.file.read_bytes(), b'')
        self.assertEqual(self.other.read_bytes(), b'preserve')

    def test_cleanup_failure_releases_lock_and_keeps_evidence(self):
        revision = keys.update(self.request())['revision']
        with patch.object(keys.os, 'replace', side_effect=OSError('publish refused')), patch.object(keys.os, 'unlink', side_effect=OSError('cleanup refused')):
            with self.assertRaises(OSError):
                keys.update(self.request(True, revision))
        self.assertEqual(len(list(self.root.glob('.soda-keys-*'))), 1)
        keys.update(self.request())
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')

    def test_cli_failures_do_not_expose_exception_contents(self):
        for error in (BlockingIOError('private input'), ValueError('private input'), OSError('private input')):
            with self.subTest(error=type(error).__name__):
                stdout, stderr = io.StringIO(), io.StringIO()
                with patch.object(keys.sys, 'stdin', types.SimpleNamespace(buffer=io.BytesIO(b'{}'))), patch.object(keys, 'update', side_effect=error), redirect_stdout(stdout), redirect_stderr(stderr):
                    self.assertEqual(keys.key_main(), 1)
                self.assertEqual(stdout.getvalue(), '')
                self.assertEqual(stderr.getvalue(), 'native key operation not confirmed\n')

    def test_account_refusal_precedes_file_access(self):
        with patch.object(keys, 'account_for', side_effect=ValueError('mismatch')):
            with self.assertRaises(ValueError): keys.update(self.request())
        self.assertEqual(self.file.read_bytes(), b'ssh-ed25519 YWJj\n')

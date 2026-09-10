"""Local PTY supervision and fixed launcher contracts; no appliance or host users.

Process tests replace only launch_attach with an unprivileged, clean test shell.
They prove real local PTY lifetime, NOT project credential dropping/native Podman.
"""
import base64
import importlib.util
import json
import os
from pathlib import Path
import pwd
import select
import signal
import stat
import subprocess
import sys
import tempfile
import time
import types
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[2] / 'internal/host/project_terminal.py'
spec = importlib.util.spec_from_file_location('project_terminal', SOURCE)
terminal = importlib.util.module_from_spec(spec)
spec.loader.exec_module(terminal)


class TerminalProtocol(unittest.TestCase):
    def test_controls_are_bounded_and_not_commands(self):
        for value in [b'null', b'{"type":"command","data":"id"}',
                      b'{"type":"heartbeat","type":"close"}',
                      b'{"type":"heartbeat","command":"id"}',
                      b'{"type":"resize","cols":true,"rows":24}',
                      b'{"type":"resize","cols":501,"rows":24}',
                      b'{"type":"input","data":"!"}',
                      json.dumps({'type': 'input', 'data': base64.b64encode(b'x'*16385).decode()}).encode()]:
            with self.assertRaises((ValueError, TypeError)):
                terminal.decode_frame(value)
        self.assertEqual(terminal.decode_frame(b'{"type":"input","data":"Aw=="}'), ('input', b'\x03'))

    def test_credential_drop_precedes_personal_files(self):
        a = types.SimpleNamespace(pw_name='alice', pw_uid=1001, pw_gid=1001,
                                  pw_dir='/home/alice', pw_shell='/bin/bash')
        calls = []
        with patch.multiple(terminal.os,
                            initgroups=lambda *v: calls.append(('groups', v)),
                            setresgid=lambda *v: calls.append(('gid', v)),
                            setresuid=lambda *v: calls.append(('uid', v)),
                            chdir=lambda *v: calls.append(('home', v)),
                            umask=lambda *v: calls.append(('umask', v)),
                            execve=lambda *v: calls.append(('exec', v))):
            terminal.launch_attach(a, '/run/soda-terminals/' + 'a'*32 + '/screen/socket')
        self.assertEqual([c[0] for c in calls], ['groups', 'gid', 'uid', 'home', 'umask', 'exec'])
        self.assertEqual(calls[2][1], (1001, 1001, 1001))
        shell, args, env = calls[-1][1]
        self.assertEqual(shell, '/usr/bin/tmux')
        self.assertEqual(args, ['tmux', '-N', '-S', '/run/soda-terminals/' + 'a'*32 + '/screen/socket', 'attach-session', '-E', '-t', '=soda'])
        self.assertEqual(env['HOME'], '/home/alice')
        self.assertEqual(set(env), {'HOME', 'USER', 'LOGNAME', 'SHELL', 'PATH', 'TERM', 'LANG'})

    def test_markers_and_ancestors_refuse_adoption(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            accounts = root/'var/lib/soda/accounts'
            accounts.mkdir(parents=True)
            marker = accounts/'alice'
            marker.write_text('2')
            marker.chmod(0o600)
            real_open, real_stat = os.open, os.fstat
            def opened(path, flags, *args, **kwargs):
                return real_open(root if path == '/' else path, flags, *args, **kwargs)
            def inspected(fd):
                s = real_stat(fd)
                return types.SimpleNamespace(st_uid=0, st_mode=s.st_mode)
            a = types.SimpleNamespace(pw_name='alice', pw_uid=1001, pw_dir='/home/alice', pw_shell='/bin/bash')
            with patch.object(terminal.os, 'open', opened), patch.object(terminal.os, 'fstat', inspected), \
                 patch.object(terminal.os, 'geteuid', return_value=0), patch.object(terminal.pwd, 'getpwnam', return_value=a):
                self.assertIs(terminal.account_for('alice', 2), a)
                with self.assertRaises(ValueError): terminal.account_for('alice', 3)
                marker.chmod(0o644)
                with self.assertRaises(ValueError): terminal.account_for('alice', 2)
                marker.unlink();marker.symlink_to('/etc/passwd')
                with self.assertRaises(OSError): terminal.account_for('alice', 2)
                marker.unlink();os.mkfifo(marker, 0o600)
                with self.assertRaises(ValueError): terminal.account_for('alice', 2)
                marker.unlink();marker.write_text('2');marker.chmod(0o600)
                accounts.chmod(0o777)
                with self.assertRaises(ValueError): terminal.account_for('alice', 2)
                accounts.chmod(0o700)
                a.pw_uid = 0
                with self.assertRaises(ValueError): terminal.account_for('alice', 2)

    def test_unprivileged_entrypoint_never_launches(self):
        if os.geteuid() == 0:
            self.skipTest('this check requires the unprivileged development user')
        p = subprocess.run([sys.executable, '-I', str(SOURCE), 'create', 'a'*32, 'alice', '2', '80', '24', '30', '0'*64], capture_output=True, timeout=5)
        self.assertEqual(p.returncode, 1)
        self.assertEqual(json.loads(p.stdout), {'type': 'closed', 'reason': 'launch_failed'})
        self.assertEqual(p.stderr, b'')


class ManagedTerminalBoundary(unittest.TestCase):
    """Temporary filesystem/command doubles, not tmux or systemd execution."""
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.identifier = 'a'*32
        self.directory = self.root/'run/soda-terminals'/self.identifier
        self.directory.mkdir(parents=True)
        (self.directory/'screen').mkdir()
        self.account = types.SimpleNamespace(pw_name='alice', pw_uid=1001, pw_gid=1002,
                                            pw_dir='/home/alice', pw_shell='/bin/bash')
        data = {'account': terminal.account_binding(self.account), 'identity': 2, 'cols': 80,
                'rows': 24, 'deadline': time.monotonic()+120}
        for name, value in {'binding': json.dumps(data), 'ready': json.dumps({'pid': 42, 'socket': [1, 2]}),
                            'lease': str(time.monotonic()+60), 'writer': ''}.items():
            p = self.directory/name; p.write_text(value); p.chmod(0o600)
        opened, inspected = os.open, os.fstat
        def open_root(path, flags, *args, **kwargs):
            return opened(self.root if path == '/' else path, flags, *args, **kwargs)
        def root_owner(fd):
            fields = list(inspected(fd)); fields[4] = 0
            return os.stat_result(fields)
        self.open_patch = patch.object(terminal.os, 'open', open_root)
        self.stat_patch = patch.object(terminal.os, 'fstat', root_owner)
        self.open_patch.start(); self.stat_patch.start()
        self.addCleanup(self.open_patch.stop); self.addCleanup(self.stat_patch.stop)

    def test_attach_checks_original_binding_and_never_creates(self):
        with patch.object(terminal, 'service_state', return_value='active'), \
             patch.object(terminal, 'socket_identity', return_value=[1, 2]), \
             patch.object(terminal, 'run_terminal', return_value=0) as attach, \
             patch.object(terminal.subprocess, 'run') as commands:
            self.assertEqual(terminal.attach_terminal(self.identifier, self.account, 2, 80, 24, 60), 0)
            self.assertEqual(attach.call_args.args[-1], '/run/soda-terminals/' + self.identifier + '/screen/socket')
            commands.assert_not_called()
            with self.assertRaises(ValueError): terminal.attach_terminal(self.identifier, self.account, 3, 80, 24, 60)
            with self.assertRaises(FileNotFoundError): terminal.attach_terminal('b'*32, self.account, 2, 80, 24, 60)
            (self.directory/'lease').write_text('0')
            with self.assertRaises(ValueError): terminal.attach_terminal(self.identifier, self.account, 2, 80, 24, 60)
            self.assertEqual(attach.call_count, 1)

    def test_nonfinite_lease_refuses(self):
        for value in ('nan', 'inf', '-inf'):
            (self.directory/'lease').write_text(value)
            with (self.directory/'lease').open() as f:
                with self.assertRaises(ValueError): terminal.lease_value(f.fileno())

    def test_writer_lock_and_socket_replacement_refuse(self):
        with (self.directory/'writer').open('r+') as writer:
            terminal.fcntl.flock(writer, terminal.fcntl.LOCK_EX)
            with patch.object(terminal, 'run_terminal') as attach:
                with self.assertRaises(BlockingIOError): terminal.attach_terminal(self.identifier, self.account, 2, 80, 24, 60)
                attach.assert_not_called()
        with patch.object(terminal, 'service_state', return_value='active'), \
             patch.object(terminal, 'socket_identity', return_value=[1, 999]), \
             patch.object(terminal, 'run_terminal') as attach:
            with self.assertRaises(ValueError): terminal.attach_terminal(self.identifier, self.account, 2, 80, 24, 60)
            attach.assert_not_called()

    def test_unsafe_paths_and_metadata_refuse(self):
        with patch.object(terminal, 'run_terminal') as attach:
            for name in ('../x', '', 'A'*32, 'a'*33):
                with self.assertRaises(ValueError): terminal.attach_terminal(name, self.account, 2, 80, 24, 60)
            (self.directory/'writer').chmod(0o666)
            with self.assertRaises(ValueError): terminal.attach_terminal(self.identifier, self.account, 2, 80, 24, 60)
            (self.directory/'writer').unlink(); (self.directory/'writer').symlink_to('/etc/passwd')
            with self.assertRaises(OSError): terminal.attach_terminal(self.identifier, self.account, 2, 80, 24, 60)
            attach.assert_not_called()

    def test_cgroup_population_not_unit_status_proves_cleanup(self):
        group = self.root/terminal.cgroup_directory(self.identifier).lstrip('/')
        group.mkdir(parents=True)
        self.mock_kernel_filesystems()
        (group/'cgroup.events').write_text('populated 1\nfrozen 0\n')
        self.assertFalse(terminal.cgroup_empty(self.identifier))
        with patch.object(terminal, 'service_state', return_value='inactive'):
            with self.assertRaises(ValueError): terminal.stop_service(self.identifier)
        (group/'cgroup.events').write_text('populated 0\nfrozen 0\n')
        self.assertTrue(terminal.cgroup_empty(self.identifier))
        (group/'cgroup.events').unlink(); group.rmdir()
        self.assertTrue(terminal.cgroup_empty(self.identifier))
        group.parent.rmdir()
        with self.assertRaises(FileNotFoundError): terminal.cgroup_empty(self.identifier)

    def mock_kernel_filesystems(self):
        def stat_command(args, **kwargs):
            fd = kwargs['pass_fds'][0]
            self.assertEqual(args, ['/usr/bin/stat', '-f', '-c', '%T', '/proc/self/fd/' + str(fd)])
            path = Path(os.readlink('/proc/self/fd/' + str(fd)))
            return types.SimpleNamespace(stdout=b'sysfs\n' if path.name in ('sys', 'fs') else b'cgroup2fs\n')
        commands = patch.object(terminal.subprocess, 'run', side_effect=stat_command)
        readonly = patch.object(terminal.os, 'fstatvfs', return_value=types.SimpleNamespace(f_flag=os.ST_RDONLY))
        commands.start(); readonly.start()
        self.addCleanup(commands.stop); self.addCleanup(readonly.stop)

    def test_kernel_ancestry_accepts_unmapped_sysfs_but_not_mutable_or_wrong_filesystems(self):
        group = self.root/'sys/fs/cgroup/system.slice'; group.mkdir(parents=True)
        self.mock_kernel_filesystems()
        inspected = terminal.os.fstat
        def mapped_owner(fd):
            fields = list(inspected(fd))
            if Path(os.readlink('/proc/self/fd/' + str(fd))).name in ('sys', 'fs'):
                fields[4] = 65534
            return os.stat_result(fields)
        with patch.object(terminal.os, 'fstat', side_effect=mapped_owner):
            os.close(terminal.cgroup_parent())
            with patch.object(terminal.os, 'fstatvfs', return_value=types.SimpleNamespace(f_flag=0)):
                with self.assertRaises(ValueError): terminal.cgroup_parent()
            with patch.object(terminal.subprocess, 'run', return_value=types.SimpleNamespace(stdout=b'ext2/ext3\n')):
                with self.assertRaises(ValueError): terminal.cgroup_parent()
        group.chmod(0o777)
        with self.assertRaises(ValueError): terminal.cgroup_parent()
        group.chmod(0o755); group.rmdir(); group.symlink_to(self.directory)
        with self.assertRaises(OSError): terminal.cgroup_parent()

    def test_occupied_creation_and_missing_program_never_start_service(self):
        with patch.object(terminal.subprocess, 'run') as run:
            with self.assertRaises(FileNotFoundError): terminal.own_terminal(self.identifier, self.account, 2, 80, 24, 60, '0'*64)
            run.assert_not_called()
            program = self.root/'usr/libexec/soda/project-terminal'; program.parent.mkdir(parents=True)
            program.write_bytes(SOURCE.read_bytes()); program.chmod(0o644)
            digest = terminal.hashlib.sha256(program.read_bytes()).hexdigest()
            with self.assertRaises(FileExistsError): terminal.own_terminal(self.identifier, self.account, 2, 80, 24, 60, digest)
            self.assertEqual(run.call_count, 2)  # terminfo observations only
            self.assertTrue(all(c.args[0][0] == '/usr/bin/infocmp' for c in run.call_args_list))

    def test_control_commands_are_attach_only_and_clean_user_scoped(self):
        with patch.object(terminal.subprocess, 'run') as run:
            terminal.tmux_control(self.account, '/owned/socket', 'has-session', '-t', '=soda')
            args, = run.call_args.args
            self.assertEqual(args, ['/usr/bin/tmux', '-N', '-S', '/owned/socket', 'has-session', '-t', '=soda'])
            self.assertEqual(run.call_args.kwargs['env'], terminal.user_environment(self.account))
            self.assertNotIn('new-session', args)
        self.assertIn(b'set -g status off', terminal.TMUX_CONFIG)
        self.assertIn(b'set -s set-clipboard off', terminal.TMUX_CONFIG)
        self.assertNotIn(b'pipe-pane', terminal.TMUX_CONFIG)


class LocalTerminalProcess(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.buffer = bytearray()
        self.processes = []
        self.addCleanup(self.close_processes)

    def close_processes(self):
        for p in self.processes:
            if p.poll() is None:
                p.terminate()
                try: p.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    p.kill();p.wait(timeout=5)
            for f in (p.stdin, p.stdout, p.stderr):
                if f: f.close()

    def start(self, lease=3, failed=False, ignored_signals=False):
        harness = '''import importlib.util,os,signal,sys,types
from pathlib import Path
s=importlib.util.spec_from_file_location('terminal',sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
m.LEASE_SECONDS=float(sys.argv[3])
if sys.argv[5]=='ignored':
 signal.signal(signal.SIGINT,signal.SIG_IGN)
 signal.signal(signal.SIGQUIT,signal.SIG_IGN)
def shell(account,path):
 # This fixture substitutes Bash for tmux. Do not inherit a detached build
 # coordinator's ignored interrupts into Bash's future foreground children.
 signal.signal(signal.SIGINT,signal.SIG_DFL)
 signal.signal(signal.SIGQUIT,signal.SIG_DFL)
 Path(account.pw_dir,'pid').write_text(str(os.getpid()))
 if sys.argv[4]=='fail':raise ValueError('SYNTHETIC_PRIVATE_ERROR')
 os.chdir(account.pw_dir)
 os.execve('/bin/bash',['bash','--noprofile','--norc','-i'],{'HOME':account.pw_dir,'PATH':'/usr/bin:/bin','TERM':'xterm-256color','PS1':'TEST> '})
m.launch_attach=shell
sys.exit(m.run_terminal(types.SimpleNamespace(pw_dir=sys.argv[2]),80,24,10,'synthetic-socket'))
'''
        p = subprocess.Popen([sys.executable, '-I', '-c', harness, str(SOURCE), str(self.root), str(lease), 'fail' if failed else 'ok', 'ignored' if ignored_signals else 'normal'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.processes.append(p)
        if not failed:
            self.assertEqual(self.receive(p), {'type': 'ready'})
        return p

    def receive(self, p, timeout=5):
        until = time.monotonic()+timeout
        while b'\n' not in self.buffer:
            left = until-time.monotonic()
            self.assertGreater(left, 0, 'terminal response timeout')
            r, _, _ = select.select([p.stdout], [], [], left)
            self.assertTrue(r, 'terminal response timeout')
            data = os.read(p.stdout.fileno(), 4096)
            self.assertTrue(data, 'terminal stream ended before status')
            self.buffer.extend(data)
        raw, _, rest = self.buffer.partition(b'\n');self.buffer = bytearray(rest)
        return json.loads(raw)

    def send(self, p, value):
        p.stdin.write(json.dumps(value).encode()+b'\n');p.stdin.flush()

    def input(self, p, data):
        self.send(p, {'type': 'input', 'data': base64.b64encode(data).decode()})

    def output_until(self, p, expected):
        text = b''
        for _ in range(60):
            v = self.receive(p)
            self.assertEqual(v['type'], 'output')
            text += base64.b64decode(v['data'])
            if expected in text:return text
        self.fail('expected output absent')

    def closed(self, p, reason):
        while True:
            v = self.receive(p)
            if v['type'] == 'closed':
                self.assertEqual(v['reason'], reason);break
        p.wait(timeout=5)
        self.assertEqual(p.stderr.read(), b'')
        pid = int((self.root/'pid').read_text())
        with self.assertRaises(ProcessLookupError): os.kill(pid, 0)

    def test_real_pty_resize_interrupt_and_eof(self):
        self.exercise_pty_resize_interrupt_and_eof(False)

    def test_real_pty_with_ignored_coordinator_interrupts(self):
        self.exercise_pty_resize_interrupt_and_eof(True)

    def exercise_pty_resize_interrupt_and_eof(self, ignored_signals):
        p = self.start(ignored_signals=ignored_signals)
        self.input(p, b'printf "__TTY__%s\\n" "$(tty)"\n')
        self.output_until(p, b'__TTY__/dev/pts/')
        self.send(p, {'type': 'resize', 'cols': 103, 'rows': 37})
        self.input(p, b'printf "__SIZE__%s\\n" "$(stty size)"\n')
        self.output_until(p, b'__SIZE__37 103')
        # Observe the foreground child, not a guessed scheduling delay. Its
        # marker is constructed so the echoed command cannot satisfy the wait.
        self.input(p, b'''python3 -u -c 'import time; print("__JOB__"+"ready"); time.sleep(30)'\n''')
        self.output_until(p, b'__JOB__ready')
        # VINTR may flush queued input. Send the next command only after Bash
        # has regained the foreground and displayed its fixture-owned prompt.
        self.input(p, b'\x03')
        self.output_until(p, b'TEST> ')
        self.input(p, b'printf "__ALIVE__%s\\n" "ok"\n')
        self.output_until(p, b'__ALIVE__ok')
        p.stdin.close()
        self.closed(p, 'disconnected')

    def test_lost_heartbeat_ends_only_owned_login(self):
        unrelated = subprocess.Popen(['/bin/sleep', '30'])
        self.processes.append(unrelated)
        p = self.start(lease=0.4)
        self.closed(p, 'expired')
        self.assertIsNone(unrelated.poll())

    def test_input_does_not_renew_lease(self):
        p = self.start(lease=0.4)
        self.input(p, b'printf hello\n')
        self.closed(p, 'expired')

    def test_close_and_final_output(self):
        p = self.start()
        self.input(p, b'printf "__FINAL__%s\\n" "ok"; exit\n')
        self.output_until(p, b'__FINAL__ok')
        self.closed(p, 'exited')

    def test_expiry_is_not_blocked_by_a_slow_output_consumer(self):
        p = self.start(lease=0.4)
        self.input(p, b'yes terminal-output\n')
        # Do not read stdout: queues/pipes must not prevent lease enforcement.
        p.wait(timeout=5)
        pid = int((self.root/'pid').read_text())
        with self.assertRaises(ProcessLookupError): os.kill(pid, 0)
        self.assertEqual(p.stderr.read(), b'')

    def test_launch_failure_is_sanitized(self):
        p = self.start(failed=True)
        self.closed(p, 'launch_failed')
        self.assertEqual(p.returncode, 1)

    def test_bad_frame_ends_shell_without_echoing_payload(self):
        p = self.start()
        self.send(p, {'type': 'SYNTHETIC_PRIVATE_ERROR'})
        self.closed(p, 'stream_failed')
        self.assertEqual(p.returncode, 1)

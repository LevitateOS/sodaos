"""Local PTY supervision and fixed launcher contracts; no appliance or host users.

Process tests replace only launch_shell with an unprivileged, clean test shell.
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
            terminal.launch_shell(a)
        self.assertEqual([c[0] for c in calls], ['groups', 'gid', 'uid', 'home', 'umask', 'exec'])
        self.assertEqual(calls[2][1], (1001, 1001, 1001))
        shell, args, env = calls[-1][1]
        self.assertEqual((shell, args, env['HOME']), ('/bin/bash', ['-bash'], '/home/alice'))
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
        p = subprocess.run([sys.executable, '-I', str(SOURCE), 'alice', '2', '80', '24', '30'], capture_output=True, timeout=5)
        self.assertEqual(p.returncode, 1)
        self.assertEqual(json.loads(p.stdout), {'type': 'closed', 'reason': 'launch_failed'})
        self.assertEqual(p.stderr, b'')


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

    def start(self, lease=3, failed=False):
        harness = '''import importlib.util,os,sys,types
from pathlib import Path
s=importlib.util.spec_from_file_location('terminal',sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
m.LEASE_SECONDS=float(sys.argv[3])
def shell(account):
 Path(account.pw_dir,'pid').write_text(str(os.getpid()))
 if sys.argv[4]=='fail':raise ValueError('SYNTHETIC_PRIVATE_ERROR')
 os.chdir(account.pw_dir)
 os.execve('/bin/bash',['bash','--noprofile','--norc','-i'],{'HOME':account.pw_dir,'PATH':'/usr/bin:/bin','TERM':'xterm-256color','PS1':'TEST> '})
m.launch_shell=shell
sys.exit(m.run_terminal(types.SimpleNamespace(pw_dir=sys.argv[2]),80,24,10))
'''
        p = subprocess.Popen([sys.executable, '-I', '-c', harness, str(SOURCE), str(self.root), str(lease), 'fail' if failed else 'ok'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
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
        p = self.start()
        self.input(p, b'printf "__TTY__%s\\n" "$(tty)"\n')
        self.output_until(p, b'__TTY__/dev/pts/')
        self.send(p, {'type': 'resize', 'cols': 103, 'rows': 37})
        self.input(p, b'printf "__SIZE__%s\\n" "$(stty size)"\n')
        self.output_until(p, b'__SIZE__37 103')
        self.input(p, b'sleep 30\n')
        time.sleep(0.1)
        self.input(p, b'\x03printf "__ALIVE__%s\\n" "ok"\n')
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

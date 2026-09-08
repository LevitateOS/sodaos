"""Fixed project-local PTY launcher, embedded by soda-host; never a user command API.

stdin/stdout are bounded JSON lines, not a terminal. Only the child PTY receives
input. A dead peer expires even when Podman leaves its exec session attached to
conmon. No transcript or diagnostic containing terminal bytes is emitted.
"""
import base64
import binascii
import errno
import fcntl
import json
import os
import pwd
import re
import select
import signal
import stat
import struct
import sys
import termios
import time

FRAME_LIMIT = 32768
QUEUE_LIMIT = 262144
LEASE_SECONDS = 60


def dimensions(cols, rows):
    return type(cols) is int and type(rows) is int and 2 <= cols <= 500 and 2 <= rows <= 300


def decode_frame(raw):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError('duplicate field')
            result[key] = value
        return result
    value = json.loads(raw.decode('utf-8'), object_pairs_hook=unique)
    if not isinstance(value, dict):
        raise ValueError('object required')
    kind = value.get('type')
    if kind == 'input' and set(value) == {'type', 'data'} and isinstance(value['data'], str):
        data = base64.b64decode(value['data'], validate=True)
        if not data or len(data) > 16384:
            raise ValueError('input size')
        return kind, data
    if kind == 'resize' and set(value) == {'type', 'cols', 'rows'} and dimensions(value['cols'], value['rows']):
        return kind, (value['cols'], value['rows'])
    if kind in ('heartbeat', 'close') and set(value) == {'type'}:
        return kind, None
    raise ValueError('unsupported control')


def account_for(login, identity):
    if os.geteuid() != 0 or not re.fullmatch(r'[a-z][a-z0-9_-]{0,30}', login) or login == 'root' or identity <= 0:
        raise ValueError('invalid identity')
    # Refuse symlink/special-file/unsafe-ancestor adoption. The marker is not a
    # substitute for the web layer's authenticated membership authorization.
    fd = os.open('/', os.O_RDONLY | os.O_DIRECTORY)
    try:
        for part in ('var', 'lib', 'soda', 'accounts'):
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            os.close(fd)
            fd = child
            info = os.fstat(fd)
            if info.st_uid != 0 or info.st_mode & 0o022:
                raise ValueError('unsafe account directory')
        marker = os.open(login, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=fd)
        try:
            info = os.fstat(marker)
            if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o077:
                raise ValueError('unsafe identity marker')
            if os.read(marker, 33) != str(identity).encode('ascii'):
                raise ValueError('identity mismatch')
        finally:
            os.close(marker)
    finally:
        os.close(fd)
    account = pwd.getpwnam(login)
    if account.pw_name != login or account.pw_uid <= 0 or not os.path.isabs(account.pw_dir) or not os.path.isabs(account.pw_shell):
        raise ValueError('invalid native account')
    if os.path.basename(account.pw_shell) in ('false', 'nologin'):
        raise ValueError('login disabled')
    return account


def launch_shell(account):
    # Drop real/effective/saved credentials before touching personal startup files.
    os.initgroups(account.pw_name, account.pw_gid)
    os.setresgid(account.pw_gid, account.pw_gid, account.pw_gid)
    os.setresuid(account.pw_uid, account.pw_uid, account.pw_uid)
    os.chdir(account.pw_dir)
    os.umask(0o022)
    env = {'HOME': account.pw_dir, 'USER': account.pw_name, 'LOGNAME': account.pw_name,
           'SHELL': account.pw_shell, 'PATH': '/usr/local/bin:/usr/bin:/bin',
           'TERM': 'xterm-256color', 'LANG': 'C.UTF-8'}
    os.execve(account.pw_shell, ['-' + os.path.basename(account.pw_shell)], env)


def set_size(master, cols, rows):
    fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack('HHHH', rows, cols, 0, 0))


def child_exited(pid):
    # Do not reap before all possible signals: the child PID cannot be reused.
    return os.waitid(os.P_PID, pid, os.WEXITED | os.WNOHANG | os.WNOWAIT) is not None


def end_child(pid, master):
    os.close(master)  # Kernel hangup targets this terminal, not all of a user's jobs.
    for sig in (signal.SIGTERM, signal.SIGKILL):
        if child_exited(pid):
            break
        os.kill(pid, sig)
        until = time.monotonic() + 2
        while not child_exited(pid) and time.monotonic() < until:
            time.sleep(0.02)
    if not child_exited(pid):
        return None  # Honest uncertainty, never signal unrelated or recycled PIDs.
    return os.waitpid(pid, 0)[1]


def line(value):
    return json.dumps(value, separators=(',', ':')).encode('ascii') + b'\n'


def run_terminal(account, cols, rows, seconds):
    if not dimensions(cols, rows) or type(seconds) is not int or not 1 <= seconds <= 7200:
        raise ValueError('invalid terminal bounds')
    # Pipe EOF confirms successful exec (CLOEXEC). PTY output is withheld until then.
    ready_read, ready_write = os.pipe2(os.O_CLOEXEC)
    # A gate lets the parent set initial dimensions before the login shell executes.
    gate_read, gate_write = os.pipe2(os.O_CLOEXEC)
    pid, master = os.forkpty()
    if pid == 0:
        try:
            os.close(ready_read)
            os.close(gate_write)
            if os.read(gate_read, 1) != b'1':
                os._exit(1)
            os.close(gate_read)
            launch_shell(account)
        except BaseException:
            os.write(ready_write, b'failed')
        os._exit(1)
    os.close(ready_write)
    os.close(gate_read)
    reason = 'disconnected'
    old_handlers = {}
    stopped = False
    outgoing = bytearray()

    def stop(_sig, _frame):
        nonlocal stopped
        stopped = True

    try:
        for sig in (signal.SIGTERM, signal.SIGHUP, signal.SIGINT):
            old_handlers[sig] = signal.signal(sig, stop)
        set_size(master, cols, rows)
        os.write(gate_write, b'1')
        os.close(gate_write)
        gate_write = None
        ready, _, _ = select.select([ready_read], [], [], 5)
        if not ready or os.read(ready_read, 16):
            reason = 'launch_failed'
        else:
            for fd in (0, 1, master):
                os.set_blocking(fd, False)
            incoming, to_pty = bytearray(), bytearray()
            outgoing = bytearray(line({'type': 'ready'}))
            deadline = time.monotonic() + seconds
            lease = time.monotonic() + LEASE_SECONDS
            while not stopped:
                now = time.monotonic()
                if now >= min(deadline, lease):
                    if reason != 'exited':
                        reason = 'expired'
                    break
                if reason != 'exited' and child_exited(pid):
                    reason = 'exited'
                    # Drain final output, but do not retain the PTY indefinitely
                    # for descendants after its login shell has exited.
                    deadline = min(deadline, now + 0.5)
                # Input is not accumulated without bound while the shell is blocked.
                reads = [0] if len(to_pty) <= QUEUE_LIMIT - 16384 else []
                if len(outgoing) <= QUEUE_LIMIT - 8192:
                    reads.append(master)
                writes = ([1] if outgoing else []) + ([master] if to_pty else [])
                readable, writable, _ = select.select(reads, writes, [], min(0.1, deadline-now, lease-now))
                if 0 in readable:
                    data = os.read(0, 4096)
                    if not data:
                        break
                    incoming.extend(data)
                    while b'\n' in incoming:
                        raw, _, tail = incoming.partition(b'\n')
                        incoming = bytearray(tail)
                        if len(raw) > FRAME_LIMIT:
                            raise ValueError('frame size')
                        kind, value = decode_frame(raw)
                        if kind == 'close':
                            stopped = True
                            break
                        if kind == 'heartbeat':
                            lease = time.monotonic() + LEASE_SECONDS
                        elif kind == 'resize':
                            set_size(master, *value)
                        elif len(to_pty) + len(value) <= QUEUE_LIMIT:
                            to_pty.extend(value)
                        else:
                            raise ValueError('input queue full')
                    if len(incoming) > FRAME_LIMIT:
                        raise ValueError('frame size')
                if stopped:
                    break
                if master in readable:
                    try:
                        data = os.read(master, 4096)
                    except OSError as error:
                        if error.errno != errno.EIO:
                            raise
                        data = b''
                    if not data:
                        reason = 'exited'
                        break
                    outgoing.extend(line({'type': 'output', 'data': base64.b64encode(data).decode('ascii')}))
                if master in writable:
                    del to_pty[:os.write(master, to_pty)]
                if 1 in writable:
                    del outgoing[:os.write(1, outgoing)]
    except (OSError, ValueError, binascii.Error, RecursionError):
        reason = 'stream_failed'
    finally:
        os.close(ready_read)
        if gate_write is not None:
            os.close(gate_write)
        status = end_child(pid, master)
        for sig, handler in old_handlers.items():
            signal.signal(sig, handler)
    # Best effort, bounded status delivery AFTER teardown. No output transcript.
    outgoing.extend(line({'type': 'closed', 'reason': reason if status is not None else 'cleanup_unconfirmed'}))
    try:
        os.set_blocking(1, False)
        until = time.monotonic() + 0.5
        while outgoing and time.monotonic() < until:
            _, ready, _ = select.select([], [1], [], max(0, until - time.monotonic()))
            if ready:
                del outgoing[:os.write(1, outgoing)]
    except OSError:
        pass
    return 0 if status is not None and reason not in ('launch_failed', 'stream_failed') else 1


def main():
    try:
        if len(sys.argv) != 6:
            raise ValueError('arguments')
        login, identity, cols, rows, seconds = sys.argv[1:]
        # Bound native NSS/marker validation too; no child exists during this alarm.
        signal.alarm(5)
        account = account_for(login, int(identity))
        signal.alarm(0)
        return run_terminal(account, int(cols), int(rows), int(seconds))
    except (OSError, ValueError, KeyError):
        # No exception body, argv, terminal bytes or native account details in logs.
        os.write(1, line({'type': 'closed', 'reason': 'launch_failed'}))
        return 1


if __name__ == '__main__':
    sys.exit(main())

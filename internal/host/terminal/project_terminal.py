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
import hashlib
import socket
import subprocess
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
import unicodedata

FRAME_LIMIT = 32768
QUEUE_LIMIT = 262144
HEARTBEAT_SECONDS = 60


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
    if (
        account.pw_name != login
        or account.pw_uid <= 0
        or not os.path.isabs(account.pw_dir)
        or not os.path.isabs(account.pw_shell)
    ):
        raise ValueError('invalid native account')
    if os.path.basename(account.pw_shell) in ('false', 'nologin'):
        raise ValueError('login disabled')
    return account


def user_environment(account):
    return {
        'HOME': account.pw_dir,
        'USER': account.pw_name,
        'LOGNAME': account.pw_name,
        'SHELL': account.pw_shell,
        'PATH': '/usr/local/bin:/usr/bin:/bin',
        'TERM': 'xterm-256color',
        'LANG': 'C.UTF-8',
    }


def become_user(account):
    # Drop real/effective/saved credentials before touching personal startup files.
    os.initgroups(account.pw_name, account.pw_gid)
    os.setresgid(account.pw_gid, account.pw_gid, account.pw_gid)
    os.setresuid(account.pw_uid, account.pw_uid, account.pw_uid)
    os.chdir(account.pw_dir)
    os.umask(0o022)


def launch_attach(account, path):
    become_user(account)
    os.execve(
        '/usr/bin/tmux', ['tmux', '-N', '-S', path, 'attach-session', '-E', '-t', '=soda'], user_environment(account)
    )


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


def spawn_login_pty(account, path):
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
            launch_attach(account, path)
        except BaseException:
            os.write(ready_write, b'failed')
        os._exit(1)
    os.close(ready_write)
    os.close(gate_read)
    return pid, master, ready_read, gate_write


def apply_control_frame(kind, value, master, to_pty, stopped):
    if kind == 'close':
        stopped[0] = True
        return None
    if kind == 'heartbeat':
        return time.monotonic() + HEARTBEAT_SECONDS
    if kind == 'resize':
        set_size(master, *value)
        return None
    if len(to_pty) + len(value) <= QUEUE_LIMIT:
        to_pty.extend(value)
        return None
    raise ValueError('input queue full')


def ingest_control_bytes(incoming, master, to_pty, lease, stopped):
    while b'\n' in incoming:
        raw, _, tail = incoming.partition(b'\n')
        incoming[:] = tail
        if len(raw) > FRAME_LIMIT:
            raise ValueError('frame size')
        kind, value = decode_frame(raw)
        renewed = apply_control_frame(kind, value, master, to_pty, stopped)
        if renewed is not None:
            lease = renewed
        if stopped[0]:
            break
    if len(incoming) > FRAME_LIMIT:
        raise ValueError('frame size')
    return lease


def read_pty_output(master, outgoing, attaching):
    try:
        data = os.read(master, 4096)
    except OSError as error:
        if error.errno != errno.EIO:
            raise
        data = b''
    if not data:
        return 'exited', attaching
    if attaching:
        # exec success is not tmux input readiness. Its tty_start_tty
        # enters raw mode and flushes queued input BEFORE emitting
        # the initial screen. Wait for that output, retaining it;
        # otherwise immediate input can be echoed then discarded.
        outgoing.extend(line({'type': 'ready'}))
        attaching = False
    outgoing.extend(line({'type': 'output', 'data': base64.b64encode(data).decode('ascii')}))
    return None, attaching


def session_should_stop(now, deadline, lease, attaching, attach_deadline, pid, reason):
    if now >= min(deadline, lease):
        return True, 'expired' if reason != 'exited' else reason, deadline
    if attaching and now >= attach_deadline:
        return True, 'launch_failed', deadline
    if reason != 'exited' and child_exited(pid):
        return False, 'exited', min(deadline, now + 0.5)
    return False, reason, deadline


def pty_select(master, to_pty, outgoing, attaching, timeout):
    reads = [0] if len(to_pty) <= QUEUE_LIMIT - 16384 else []
    if len(outgoing) <= QUEUE_LIMIT - 8192:
        reads.append(master)
    writes = ([1] if outgoing else []) + ([master] if to_pty and not attaching else [])
    return select.select(reads, writes, [], timeout)


def take_control_input(incoming, master, to_pty, lease, stopped):
    data = os.read(0, 4096)
    if not data:
        return lease, 'eof'
    incoming.extend(data)
    lease = ingest_control_bytes(incoming, master, to_pty, lease, stopped)
    return lease, 'stop' if stopped[0] else None


def flush_pty_queues(master, to_pty, outgoing, writable):
    if master in writable:
        del to_pty[: os.write(master, to_pty)]
    if 1 in writable:
        del outgoing[: os.write(1, outgoing)]


def relay_pty_session(pid, master, seconds, stopped, outgoing):
    incoming, to_pty = bytearray(), bytearray()
    attaching = True
    attach_deadline = time.monotonic() + 5
    deadline = time.monotonic() + seconds
    lease = time.monotonic() + HEARTBEAT_SECONDS
    reason = 'disconnected'
    while not stopped[0]:
        now = time.monotonic()
        halt, reason, deadline = session_should_stop(now, deadline, lease, attaching, attach_deadline, pid, reason)
        if halt:
            return reason
        readable, writable, _ = pty_select(master, to_pty, outgoing, attaching, min(0.1, deadline - now, lease - now))
        if 0 in readable:
            lease, event = take_control_input(incoming, master, to_pty, lease, stopped)
            if event:
                break
        if master in readable:
            ended, attaching = read_pty_output(master, outgoing, attaching)
            if ended:
                return ended
        flush_pty_queues(master, to_pty, outgoing, writable)
    return reason


def flush_closed(outgoing):
    # Best effort, bounded status delivery AFTER teardown. No output transcript.
    try:
        os.set_blocking(1, False)
        until = time.monotonic() + 0.5
        while outgoing and time.monotonic() < until:
            _, ready, _ = select.select([], [1], [], max(0, until - time.monotonic()))
            if ready:
                del outgoing[: os.write(1, outgoing)]
    except OSError:
        pass


def run_terminal(account, cols, rows, seconds, path):
    if not dimensions(cols, rows) or type(seconds) is not int or not 1 <= seconds <= 43200:
        raise ValueError('invalid terminal bounds')
    pid, master, ready_read, gate_write = spawn_login_pty(account, path)
    reason = 'disconnected'
    old_handlers = {}
    stopped = [False]
    outgoing = bytearray()

    def stop(_sig, _frame):
        stopped[0] = True

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
            reason = relay_pty_session(pid, master, seconds, stopped, outgoing)
    except (OSError, ValueError, binascii.Error, RecursionError):
        reason = 'stream_failed'
    finally:
        os.close(ready_read)
        if gate_write is not None:
            os.close(gate_write)
        status = end_child(pid, master)
        for sig, handler in old_handlers.items():
            signal.signal(sig, handler)
    outgoing.extend(line({'type': 'closed', 'reason': reason if status is not None else 'cleanup_unconfirmed'}))
    flush_closed(outgoing)
    return 0 if status is not None and reason not in ('launch_failed', 'stream_failed') else 1


# Systemd owns the foreground tmux server and its descendants. Its bounded
# ExecStartPost prepares the socket/session; no browser-owned lifetime supervisor.
# Only an attached PTY has a heartbeat timeout. Losing it never ends the server.
TERMINALS = '/run/soda-terminals'
PROGRAM = '/usr/libexec/soda/project-terminal'
TMUX_CONFIG = b'''set -g status off
set -g history-limit 10000
set -s buffer-limit 10
set -s set-clipboard off
set -s escape-time 10
set -g default-terminal screen-256color
set -g update-environment ""
set -s exit-unattached off
'''


def terminal_path(identifier):
    if not re.fullmatch(r'[0-9a-f]{32}', identifier):
        raise ValueError('invalid terminal identifier')
    return TERMINALS + '/' + identifier


def root_directory(path):
    fd = os.open('/', os.O_RDONLY | os.O_DIRECTORY)
    try:
        for part in path.strip('/').split('/'):
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            os.close(fd)
            fd = child
            info = os.fstat(fd)
            if info.st_uid != 0 or info.st_mode & 0o022:
                raise ValueError('unsafe terminal directory')
        return fd
    except BaseException:
        os.close(fd)
        raise


def root_file(directory, name, writable=False):
    fd = os.open(name, (os.O_RDWR if writable else os.O_RDONLY) | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
    info = os.fstat(fd)
    if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_nlink != 1 or info.st_mode & 0o077:
        os.close(fd)
        raise ValueError('unsafe terminal file')
    return fd


def read_record(directory, name):
    fd = root_file(directory, name)
    try:
        raw = os.read(fd, 4097)
        if len(raw) > 4096:
            raise ValueError('terminal record size')
        return json.loads(raw)
    finally:
        os.close(fd)


def new_file(directory, name, data, mode=0o600):
    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, mode, dir_fd=directory)
    try:
        os.fchmod(fd, mode)
        if os.write(fd, data) != len(data):
            raise OSError('short terminal write')
    finally:
        os.close(fd)


def account_binding(account):
    return [account.pw_name, account.pw_uid, account.pw_gid, account.pw_dir, account.pw_shell]


def binding(directory, account=None, identity=None):
    record = read_record(directory, 'binding')
    if not isinstance(record, dict) or set(record) != {'account', 'identity', 'cols', 'rows', 'created_at'}:
        raise ValueError('terminal binding')
    a = record['account']
    if (
        not isinstance(a, list)
        or len(a) != 5
        or not isinstance(a[0], str)
        or not re.fullmatch(r'[a-z][a-z0-9_-]{0,30}', a[0])
        or a[0] == 'root'
        or type(a[1]) is not int
        or a[1] <= 0
        or type(a[2]) is not int
        or a[2] < 0
        or not all(isinstance(v, str) and os.path.isabs(v) for v in a[3:])
        or type(record['identity']) is not int
        or record['identity'] <= 0
        or not dimensions(record['cols'], record['rows'])
        or type(record['created_at']) is not int
        or not 0 < record['created_at'] <= 9007199254740991
    ):
        raise ValueError('terminal binding values')
    if account is not None and (record['identity'] != identity or record['account'] != account_binding(account)):
        raise ValueError('terminal account changed')
    return record


def valid_name(value):
    return (
        isinstance(value, str) and len(value) <= 80 and all(unicodedata.category(c) not in ('Cc', 'Cf') for c in value)
    )


def write_name(directory, name):
    if not valid_name(name):
        raise ValueError('terminal name')
    # A prior interrupted rename may have left only this exact private staging
    # file. Readers/writers share the parent lock; never overwrite unknown files.
    try:
        os.close(root_file(directory, 'name.next'))
        os.unlink('name.next', dir_fd=directory)
    except FileNotFoundError:
        pass
    new_file(directory, 'name.next', line(name))
    os.replace('name.next', 'name', src_dir_fd=directory, dst_dir_fd=directory)


def tmux_control(account, path, *args):
    subprocess.run(
        ['/usr/bin/tmux', '-N', '-S', path, *args],
        check=True,
        timeout=2,
        preexec_fn=lambda: become_user(account),
        env=user_environment(account),
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def socket_identity(path, account, pid):
    info = os.stat(path, follow_symlinks=False)
    if not stat.S_ISSOCK(info.st_mode) or info.st_uid != account.pw_uid or info.st_mode & 0o007:
        raise ValueError('unsafe tmux socket')
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as peer:
        peer.settimeout(1)
        peer.connect(path)
        server, uid, _gid = struct.unpack('3i', peer.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, 12))
    if server != pid or uid != account.pw_uid:
        raise ValueError('wrong tmux server')
    return [info.st_dev, info.st_ino]


def cgroup_directory(identifier):
    terminal_path(identifier)
    return '/sys/fs/cgroup/system.slice/soda-terminal-' + identifier + '.service'


def cgroup_parent():
    # /sys is host-owned read-only sysfs: host root is deliberately unmapped in
    # the project user namespace. Do not chown it or relax mutable-path checks.
    # Admit its kernel filesystem through held no-follow descriptors instead;
    # the delegated cgroup hierarchy must still be owned by project root.
    fd = os.open('/', os.O_RDONLY | os.O_DIRECTORY)
    try:
        for part in ('sys', 'fs', 'cgroup', 'system.slice'):
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            os.close(fd)
            fd = child
            info = os.fstat(fd)
            result = subprocess.run(
                ['/usr/bin/stat', '-f', '-c', '%T', '/proc/self/fd/' + str(fd)],
                pass_fds=(fd,),
                check=True,
                timeout=2,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
            )
            if part in ('sys', 'fs'):
                if result.stdout != b'sysfs\n' or not os.fstatvfs(fd).f_flag & os.ST_RDONLY:
                    raise ValueError('expected read-only kernel sysfs')
            elif result.stdout != b'cgroup2fs\n' or info.st_uid != 0 or info.st_mode & 0o022:
                raise ValueError('unsafe delegated cgroup directory')
        return fd
    except BaseException:
        os.close(fd)
        raise


def cgroup_empty(identifier):
    parent = cgroup_parent()
    try:
        try:
            directory = os.open(
                'soda-terminal-' + identifier + '.service', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent
            )
        except FileNotFoundError:
            return True  # Kernel cannot remove a populated cgroup.
    finally:
        os.close(parent)
    try:
        fd = os.open('cgroup.events', os.O_RDONLY | os.O_NOFOLLOW, dir_fd=directory)
        try:
            values = dict(row.split() for row in os.read(fd, 4096).decode('ascii').splitlines())
            return values.get('populated') == '0'
        finally:
            os.close(fd)
    finally:
        os.close(directory)


def prepare(identifier):
    path = terminal_path(identifier)
    parent = cgroup_parent()
    try:
        group = os.open(
            'soda-terminal-' + identifier + '.service', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent
        )
    finally:
        os.close(parent)
    info = os.fstat(group)
    if info.st_uid != 0 or info.st_mode & 0o022:
        os.close(group)
        raise ValueError('unsafe terminal cgroup')
    try:
        fd = os.open('cgroup.procs', os.O_RDONLY | os.O_NOFOLLOW, dir_fd=group)
        try:
            if str(os.getpid()) not in os.read(fd, 16384).decode('ascii').splitlines():
                raise ValueError('preparation outside owned cgroup')
        finally:
            os.close(fd)
    finally:
        os.close(group)
    directory = root_directory(path)
    record = binding(directory)
    account = account_for(record['account'][0], record['identity'])
    binding(directory, account, record['identity'])
    sock = path + '/screen/socket'
    pid = int(os.environ['MAINPID'])  # supplied by systemd, never by the browser
    if pid <= 0:
        raise ValueError('missing main process')
    until = time.monotonic() + 5
    while True:
        if time.monotonic() >= until:
            raise ValueError('tmux startup failed')
        try:
            if os.path.exists(sock + '.lock'):
                raise FileNotFoundError('startup lock')
            socket_identity(sock, account, pid)
            break
        except (FileNotFoundError, ConnectionRefusedError):
            time.sleep(0.02)
    # Seal the socket's parent before publishing readiness: the project user may
    # connect but cannot replace this socket with another context/personal server.
    os.chown(path + '/screen', 0, 0, follow_symlinks=False)
    os.chmod(path + '/screen', 0o711, follow_symlinks=False)
    inode = socket_identity(sock, account, pid)
    # -D starts empty and disables exit-empty. Restore normal empty-server exit
    # AFTER new-session, in the same native command queue (also supported by 3.2a).
    tmux_control(
        account,
        sock,
        'new-session',
        '-d',
        '-s',
        'soda',
        '-x',
        str(record['cols']),
        '-y',
        str(record['rows']),
        '-c',
        account.pw_dir,
        ';',
        'set-option',
        '-s',
        'exit-empty',
        'on',
    )
    new_file(directory, 'ready', line({'pid': pid, 'socket': inode}))
    os.close(directory)
    return 0


def service_state(identifier, account=None):
    unit = 'soda-terminal-' + identifier + '.service'
    properties = 'LoadState,ActiveState,Description,FragmentPath,DropInPaths,Type,User,KillMode,Restart,SendSIGKILL,TimeoutStopUSec,StandardInput,StandardOutput,StandardError'
    result = subprocess.run(
        ['/usr/bin/systemctl', 'show', unit, '--property=' + properties],
        check=False,
        timeout=3,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
    )
    if len(result.stdout) > 4096:
        raise ValueError('unit response size')
    fields = dict(line.split('=', 1) for line in result.stdout.decode('ascii').splitlines())
    if set(fields) != set(properties.split(',')) or (result.returncode and fields.get('LoadState') != 'not-found'):
        raise ValueError('unit inspection unavailable')
    if fields.get('LoadState') != 'not-found' and (
        fields.get('Description') != 'Soda terminal ' + identifier
        or fields.get('FragmentPath') != '/run/systemd/transient/' + unit
    ):
        raise ValueError('not the owned terminal unit')
    if fields['LoadState'] != 'not-found':
        if account is None:
            raise ValueError('terminal unit occupied')
        selected = {
            'DropInPaths': '',
            'Type': 'exec',
            'User': account.pw_name,
            'KillMode': 'control-group',
            'Restart': 'no',
            'SendSIGKILL': 'yes',
            'TimeoutStopUSec': '3s',
            'StandardInput': 'null',
            'StandardOutput': 'null',
            'StandardError': 'null',
        }
        if any(fields[k] != value for k, value in selected.items()):
            raise ValueError('terminal supervision changed')
    return fields.get('ActiveState')


def stop_service(identifier, account):
    state = service_state(identifier, account)
    if state not in ('inactive', 'failed'):
        subprocess.run(
            ['/usr/bin/systemctl', 'stop', 'soda-terminal-' + identifier + '.service'],
            check=True,
            timeout=8,
            stdin=subprocess.DEVNULL,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
    if service_state(identifier, account) not in ('inactive', 'failed') or not cgroup_empty(identifier):
        raise ValueError('terminal cleanup unconfirmed')


def remove_screen_contents(directory, account):
    screen = os.open('screen', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=directory)
    try:
        info = os.fstat(screen)
        # Failed startup may precede the root seal. The parent is held and
        # the unit/cgroup is already empty; accept only its original account.
        if info.st_uid not in (0, account.pw_uid) or info.st_mode & 0o022:
            raise ValueError('unexpected terminal screen')
        for name in os.listdir(screen):
            info = os.stat(name, dir_fd=screen, follow_symlinks=False)
            expected = (
                name == 'socket' and stat.S_ISSOCK(info.st_mode) or name == 'socket.lock' and stat.S_ISREG(info.st_mode)
            )
            if not expected or info.st_uid != account.pw_uid or info.st_nlink != 1:
                raise ValueError('unexpected terminal socket')
            os.unlink(name, dir_fd=screen)
    finally:
        os.close(screen)


def remove_owned_files(path, directory, account):
    # Only exact run-owned files after unit shutdown, never recursive tree removal.
    names = set(os.listdir(directory))
    if names - {'binding', 'reservation', 'name', 'name.next', 'writer', 'tmux.conf', 'ready', 'screen'}:
        raise ValueError('unexpected terminal files')
    if 'screen' in names:
        remove_screen_contents(directory, account)
    for name in names - {'screen'}:
        info = os.stat(name, dir_fd=directory, follow_symlinks=False)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_nlink != 1:
            raise ValueError('unexpected terminal file')
    for name in names - {'screen'}:
        os.unlink(name, dir_fd=directory)
    if 'screen' in names:
        os.rmdir('screen', dir_fd=directory)
    os.rmdir(path)


def reserve_terminal(identifier, account, identity, cols, rows, name, source_hash, scope):
    # Missing/older project support refuses, never installs itself on Open.
    parent = root_directory('/usr/libexec/soda')
    program = os.open('project-terminal', os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
    try:
        info = os.fstat(program)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022 or info.st_size > 65536:
            raise ValueError('unsafe terminal program')
        if hashlib.sha256(os.read(program, 65537)).hexdigest() != source_hash:
            raise ValueError('terminal support version differs')
    finally:
        os.close(program)
        os.close(parent)
    for term in ('xterm-256color', 'screen-256color'):
        subprocess.run(
            ['/usr/bin/infocmp', term], check=True, timeout=2, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        )
    path = terminal_path(identifier)
    # Refuse occupied locators before retiring any completed runtime files.
    # The caller holds the native parent lock across preparation/admission.
    parent = root_directory(TERMINALS)
    try:
        try:
            os.stat(identifier, dir_fd=parent, follow_symlinks=False)
        except FileNotFoundError:
            pass
        else:
            raise FileExistsError('terminal identifier occupied')
        if service_state(identifier) != 'inactive' or not cgroup_empty(identifier):
            raise ValueError('terminal unit occupied')
        collect_finished()
        if len(terminal_directories()) >= 128:
            raise ValueError('native reservation capacity')
        os.mkdir(identifier, 0o711, dir_fd=parent)
    finally:
        os.close(parent)
    directory = root_directory(path)
    os.fchmod(directory, 0o711)
    record = {
        'account': account_binding(account),
        'identity': identity,
        'cols': cols,
        'rows': rows,
        'created_at': int(time.time()),
    }
    new_file(directory, 'binding', line(record))
    new_file(directory, 'reservation', line({'expires': int(time.time()) + 120, 'scope': scope}))
    write_name(directory, name)
    new_file(directory, 'writer', b'')
    new_file(directory, 'tmux.conf', TMUX_CONFIG, 0o644)
    os.mkdir('screen', 0o700, dir_fd=directory)
    os.chown(path + '/screen', account.pw_uid, account.pw_gid)
    os.close(directory)
    return terminal_status(identifier, account, identity)


def reservation(directory):
    try:
        value = read_record(directory, 'reservation')
    except FileNotFoundError:
        return None
    if (
        not isinstance(value, dict)
        or set(value) != {'expires', 'scope'}
        or type(value['expires']) is not int
        or value['expires'] <= 0
        or not isinstance(value['scope'], str)
        or not re.fullmatch(r'[0-9a-f]{64}', value['scope'])
    ):
        raise ValueError('invalid creation reservation')
    return value


def create_terminal(identifier, account, identity, cols, rows, name, scope):
    path = terminal_path(identifier)
    directory = root_directory(path)  # no creation on a missing/ended locator
    try:
        record = binding(directory, account, identity)
        permit = reservation(directory)
        if (
            permit is None
            or permit['expires'] <= time.time()
            or permit['scope'] != scope
            or (cols, rows) != (record['cols'], record['rows'])
        ):
            raise ValueError('creation reservation expired, consumed or changed')
        if collect_finished() >= 64:
            raise ValueError('native terminal capacity')
        # Same native lock as End: a late Create cannot run after End removed its
        # one-use permission, even across web/helper restarts or a lost reply.
        os.unlink('reservation', dir_fd=directory)
        write_name(directory, name)
    finally:
        os.close(directory)
    if service_state(identifier) != 'inactive':
        raise ValueError('terminal unit occupied')
    subprocess.run(
        [
            '/usr/bin/systemd-run',
            '--quiet',
            '--collect',
            '--unit=soda-terminal-' + identifier,
            '--description=Soda terminal ' + identifier,
            '--service-type=exec',
            '--property=User=' + account.pw_name,
            '--property=Group=' + str(account.pw_gid),
            '--property=WorkingDirectory=' + account.pw_dir,
            '--slice=system.slice',
            '--property=KillMode=control-group',
            '--property=SendSIGKILL=yes',
            '--property=Restart=no',
            '--property=TimeoutStartSec=10s',
            '--property=TimeoutStopSec=3s',
            '--property=UMask=0077',
            '--property=LimitCORE=0',
            '--property=StandardInput=null',
            '--property=StandardOutput=null',
            '--property=StandardError=null',
            '--property=ExecStartPost=+/usr/bin/python3 -I ' + PROGRAM + ' prepare ' + identifier,
            *['--setenv=' + key + '=' + value for key, value in user_environment(account).items()],
            '/usr/bin/tmux',
            '-D',
            '-S',
            path + '/screen/socket',
            '-f',
            path + '/tmux.conf',
        ],
        check=True,
        timeout=15,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    # No owner stream or lifetime timer. systemd's start job includes the bounded
    # privileged preparation hook; it cleans the cgroup if that hook fails.
    return terminal_status(identifier, account, identity)


def attach_terminal(identifier, account, identity, cols, rows, seconds):
    path = terminal_path(identifier)
    directory = root_directory(path)
    writer = root_file(directory, 'writer', True)
    try:
        fcntl.flock(writer, fcntl.LOCK_EX | fcntl.LOCK_NB)  # never evict a writer
        binding(directory, account, identity)
        screen = root_directory(path + '/screen')
        os.close(screen)
        ready = read_record(directory, 'ready')
        sock = path + '/screen/socket'
        if (
            service_state(identifier, account) != 'active'
            or socket_identity(sock, account, ready['pid']) != ready['socket']
        ):
            raise ValueError('terminal absent')
        return run_terminal(account, cols, rows, seconds, sock)
    finally:
        os.close(writer)
        os.close(directory)


def writer_attached(directory):
    writer = root_file(directory, 'writer', True)
    try:
        try:
            fcntl.flock(writer, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            return True
        return False
    finally:
        os.close(writer)


def observe_ready_terminal(path, directory, account, identifier):
    ready = read_record(directory, 'ready')
    screen = root_directory(path + '/screen')
    os.close(screen)
    if socket_identity(path + '/screen/socket', account, ready['pid']) != ready['socket']:
        raise ValueError('terminal socket changed')
    return writer_attached(directory)


def classify_unit_state(path, directory, account, identifier, state, permit):
    attached = False
    if state in ('inactive', 'failed'):
        if not cgroup_empty(identifier):
            raise ValueError('terminal cleanup unconfirmed')
        state = 'opening' if permit is not None and permit['expires'] > time.time() else 'ended'
    elif permit is not None:
        raise ValueError('unconsumed reservation has a native unit')
    elif state == 'active':
        attached = observe_ready_terminal(path, directory, account, identifier)
        state = 'ready'
    elif state in ('activating', 'deactivating'):
        state = 'opening' if state == 'activating' else 'ending'
    else:
        raise ValueError('terminal state unavailable')
    return state, attached


def terminal_status(identifier, account, identity):
    path = terminal_path(identifier)
    try:
        directory = root_directory(path)
    except FileNotFoundError:
        if service_state(identifier) not in ('inactive', 'failed') or not cgroup_empty(identifier):
            raise ValueError('unidentified native terminal')
        return None  # observed absence, not an expired web receipt
    try:
        record = binding(directory, account, identity)
        permit = reservation(directory)
        state, attached = classify_unit_state(
            path, directory, account, identifier, service_state(identifier, account), permit
        )
        name = read_record(directory, 'name')
        if not valid_name(name):
            raise ValueError('terminal name')
        return {
            'id': identifier,
            'name': name,
            'created_at': record['created_at'],
            'ready': state == 'ready',
            'attached': attached,
            'state': state,
        }
    finally:
        os.close(directory)


def terminal_directories():
    identifiers = sorted(os.listdir(TERMINALS))
    if len(identifiers) > 128:
        raise ValueError('native terminal inventory bound')
    for identifier in identifiers:
        terminal_path(identifier)
    return identifiers


def collect_finished():
    # Create, not a passive read, retires exact completed runtime files. Capacity
    # counts native units; a lost browser acknowledgement cannot reserve it forever.
    active = 0
    for identifier in terminal_directories():
        path = terminal_path(identifier)
        directory = root_directory(path)
        try:
            try:
                record = binding(directory)
            except (FileNotFoundError, ValueError):
                # A crash before the initial binding write cannot have launched
                # a unit. Confirm native absence and retire only that exact stub;
                # older/unknown metadata layouts are not converted or discarded.
                names = set(os.listdir(directory))
                if names - {'binding'} or service_state(identifier) != 'inactive' or not cgroup_empty(identifier):
                    raise
                if 'binding' in names:
                    fd = root_file(directory, 'binding')
                    try:
                        if os.read(fd, 1):
                            raise ValueError('unknown terminal binding')
                    finally:
                        os.close(fd)
                    os.unlink('binding', dir_fd=directory)
                os.rmdir(path)
                continue
            login, uid, gid, home, shell = record['account']
            account = pwd.struct_passwd((login, '', uid, gid, '', home, shell))
            if service_state(identifier, account) in ('inactive', 'failed') and cgroup_empty(identifier):
                permit = reservation(directory)
                if permit is None or permit['expires'] <= time.time():
                    remove_owned_files(path, directory, account)
            else:
                active += 1
        finally:
            os.close(directory)
    return active


def list_owned_terminals(account, identity):
    values = []
    for identifier in terminal_directories():
        directory = root_directory(terminal_path(identifier))
        try:
            record = binding(directory)
            pending = reservation(directory)
        finally:
            os.close(directory)
        if pending is None and record['identity'] == identity and record['account'] == account_binding(account):
            value = terminal_status(identifier, account, identity)
            if value is not None and value['state'] != 'ended':
                values.append(value)
    return values


def mutate_terminal(action, identifier, account, identity, name):
    try:
        directory = root_directory(terminal_path(identifier))
    except FileNotFoundError:
        if action == 'end' and terminal_status(identifier, account, identity) is None:
            return []
        raise
    try:
        binding(directory, account, identity)
        if action == 'end':
            stop_service(identifier, account)
            remove_owned_files(terminal_path(identifier), directory, account)
            return []
        if action != 'rename':
            raise ValueError('terminal action')
        write_name(directory, name)
    finally:
        os.close(directory)
    value = terminal_status(identifier, account, identity)
    return [] if value is None else [value]


def control_terminal(action, identifier, account, identity, cols, rows, name, source_hash, scope):
    parent = root_directory(TERMINALS)
    try:
        fcntl.flock(parent, fcntl.LOCK_SH if action in ('list', 'inspect') else fcntl.LOCK_EX)
        if action == 'reserve':
            value = reserve_terminal(identifier, account, identity, cols, rows, name, source_hash, scope)
        elif action == 'create':
            value = create_terminal(identifier, account, identity, cols, rows, name, scope)
        elif action == 'list':
            return list_owned_terminals(account, identity)
        elif action == 'inspect':
            value = terminal_status(identifier, account, identity)
        else:
            return mutate_terminal(action, identifier, account, identity, name)
        return [] if value is None else [value]
    finally:
        os.close(parent)


def require_terminal_target(action, identifier):
    if action != 'list':
        terminal_path(identifier)
    elif identifier:
        raise ValueError('list target')


def require_creation_scope(action, scope):
    if action in ('reserve', 'create'):
        if not re.fullmatch(r'[0-9a-f]{64}', scope):
            raise ValueError('creation scope')
    elif scope:
        raise ValueError('unexpected creation scope')


def parse_control_argv():
    action, identifier, login, identity, cols, rows, seconds, source_hash, name, scope = sys.argv[1:]
    identity, cols, rows, seconds = int(identity), int(cols), int(rows), int(seconds)
    if action not in ('reserve', 'create', 'attach', 'inspect', 'list', 'end', 'rename') or not 1 <= seconds <= 43200:
        raise ValueError('terminal bounds')
    require_terminal_target(action, identifier)
    if action in ('reserve', 'create', 'attach') and not dimensions(cols, rows) or not valid_name(name):
        raise ValueError('terminal arguments')
    require_creation_scope(action, scope)
    return action, identifier, login, identity, cols, rows, seconds, source_hash, name, scope


def main():
    try:
        if len(sys.argv) == 3 and sys.argv[1] == 'prepare':
            return prepare(sys.argv[2])
        if len(sys.argv) != 11:
            raise ValueError('arguments')
        action, identifier, login, identity, cols, rows, seconds, source_hash, name, scope = parse_control_argv()
        signal.alarm(5)
        account = account_for(login, identity)
        signal.alarm(0)
        if action == 'attach':
            return attach_terminal(identifier, account, identity, cols, rows, seconds)
        signal.alarm(min(seconds, 30))
        values = control_terminal(action, identifier, account, identity, cols, rows, name, source_hash, scope)
        os.write(1, line({'type': 'metadata', 'terminals': values}))
        return 0
    except (OSError, ValueError, KeyError, subprocess.SubprocessError):
        # No exception details, argv or terminal bytes in diagnostics.
        os.write(1, line({'type': 'closed', 'reason': 'launch_failed'}))
        return 1


if __name__ == '__main__':
    sys.exit(main())

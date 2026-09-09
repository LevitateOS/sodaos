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
import math
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


def user_environment(account):
    return {'HOME': account.pw_dir, 'USER': account.pw_name, 'LOGNAME': account.pw_name,
            'SHELL': account.pw_shell, 'PATH': '/usr/local/bin:/usr/bin:/bin',
            'TERM': 'xterm-256color', 'LANG': 'C.UTF-8'}


def become_user(account):
    # Drop real/effective/saved credentials before touching personal startup files.
    os.initgroups(account.pw_name, account.pw_gid)
    os.setresgid(account.pw_gid, account.pw_gid, account.pw_gid)
    os.setresuid(account.pw_uid, account.pw_uid, account.pw_uid)
    os.chdir(account.pw_dir)
    os.umask(0o022)


def launch_attach(account, path):
    become_user(account)
    os.execve('/usr/bin/tmux', ['tmux', '-N', '-S', path, 'attach-session', '-E', '-t', '=soda'],
              user_environment(account))


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


def run_terminal(account, cols, rows, seconds, path):
    if not dimensions(cols, rows) or type(seconds) is not int or not 1 <= seconds <= 43200:
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
            launch_attach(account, path)
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


# The owner pipe is independent of every attachment PTY. Systemd owns the guard,
# foreground tmux server and descendants; a frozen/dead guard is bounded by its
# watchdog/start timeout as well as the short owner lease. No restart/adoption.
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
    fd = os.open(name, (os.O_RDWR if writable else os.O_RDONLY) | os.O_NOFOLLOW | os.O_NONBLOCK,
                 dir_fd=directory)
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
    if set(record) != {'account', 'identity', 'cols', 'rows', 'deadline'}:
        raise ValueError('terminal binding')
    if account is not None and (record['identity'] != identity or record['account'] != account_binding(account)):
        raise ValueError('terminal account changed')
    return record


def lease_value(fd, renew=False):
    fcntl.flock(fd, fcntl.LOCK_EX if renew else fcntl.LOCK_SH)
    try:
        if renew:
            raw = str(time.monotonic() + LEASE_SECONDS).encode('ascii')
            if os.pwrite(fd, raw, 0) != len(raw):
                raise OSError('short lease write')
            os.ftruncate(fd, len(raw))
        value = float(os.pread(fd, 64, 0))
        if not math.isfinite(value):
            raise ValueError('invalid lease')
        return value
    finally:
        fcntl.flock(fd, fcntl.LOCK_UN)


def tmux_control(account, path, *args):
    subprocess.run(['/usr/bin/tmux', '-N', '-S', path, *args], check=True, timeout=2,
                   preexec_fn=lambda: become_user(account), env=user_environment(account),
                   stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


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


def notify(message):
    address = os.environ['NOTIFY_SOCKET']
    if address.startswith('@'):
        address = '\0' + address[1:]
    with socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM) as peer:
        peer.settimeout(1)
        peer.sendto(message.encode('ascii'), address)


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
            result = subprocess.run(['/usr/bin/stat', '-f', '-c', '%T', '/proc/self/fd/' + str(fd)],
                                    pass_fds=(fd,), check=True, timeout=2,
                                    stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
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
            directory = os.open('soda-terminal-' + identifier + '.service', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
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


def guard(identifier):
    path = terminal_path(identifier)
    parent = cgroup_parent()
    try:
        group = os.open('soda-terminal-' + identifier + '.service', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
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
                raise ValueError('guard outside owned cgroup')
        finally:
            os.close(fd)
    finally:
        os.close(group)
    directory = root_directory(path)
    lease = root_file(directory, 'lease')
    record = binding(directory)
    account = account_for(record['account'][0], record['identity'])
    binding(directory, account, record['identity'])
    sock = path + '/screen/socket'
    child = subprocess.Popen(['/usr/bin/tmux', '-D', '-S', sock, '-f', path + '/tmux.conf'],
                             preexec_fn=lambda: become_user(account), env=user_environment(account),
                             stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                             stderr=subprocess.DEVNULL, start_new_session=True)
    # Do not reap/signal arbitrary PIDs. Systemd cleans the entire owned cgroup on
    # every guard exit, including failed startup, normal session exit and SIGKILL.
    until = time.monotonic() + 5
    while True:
        if child.poll() is not None or time.monotonic() >= until:
            raise ValueError('tmux startup failed')
        try:
            if os.path.exists(sock + '.lock'):
                raise FileNotFoundError('startup lock')
            socket_identity(sock, account, child.pid)
            break
        except (FileNotFoundError, ConnectionRefusedError):
            time.sleep(0.02)
    # Seal the socket's parent before publishing readiness: the project user may
    # connect but cannot replace this socket with another context/personal server.
    os.chown(path + '/screen', 0, 0, follow_symlinks=False)
    os.chmod(path + '/screen', 0o711, follow_symlinks=False)
    inode = socket_identity(sock, account, child.pid)
    tmux_control(account, sock, 'new-session', '-d', '-s', 'soda', '-x', str(record['cols']),
                 '-y', str(record['rows']), '-c', account.pw_dir)
    new_file(directory, 'ready', line({'pid': child.pid, 'socket': inode}))
    notify('READY=1')
    while child.poll() is None:
        if time.monotonic() >= min(record['deadline'], lease_value(lease)):
            return 1
        tmux_control(account, sock, 'has-session', '-t', '=soda')
        notify('WATCHDOG=1')
        time.sleep(1)
    return 1


def service_state(identifier):
    unit = 'soda-terminal-' + identifier + '.service'
    properties = 'LoadState,ActiveState,Description,FragmentPath,DropInPaths,Type,User,KillMode,Restart,SendSIGKILL,WatchdogUSec,TimeoutStopUSec,StandardInput,StandardOutput,StandardError'
    result = subprocess.run(['/usr/bin/systemctl', 'show', unit, '--property=' + properties],
                            check=False, timeout=3, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    if len(result.stdout) > 4096:
        raise ValueError('unit response size')
    fields = dict(line.split('=', 1) for line in result.stdout.decode('ascii').splitlines())
    if set(fields) != set(properties.split(',')) or (result.returncode and fields.get('LoadState') != 'not-found'):
        raise ValueError('unit inspection unavailable')
    if fields.get('LoadState') != 'not-found' and (fields.get('Description') != 'Soda terminal ' + identifier or
            fields.get('FragmentPath') != '/run/systemd/transient/' + unit):
        raise ValueError('not the owned terminal unit')
    if fields['LoadState'] != 'not-found':
        selected = {'DropInPaths': '', 'Type': 'notify', 'User': 'root', 'KillMode': 'control-group',
                    'Restart': 'no', 'SendSIGKILL': 'yes', 'WatchdogUSec': '10s', 'TimeoutStopUSec': '3s',
                    'StandardInput': 'null', 'StandardOutput': 'null', 'StandardError': 'null'}
        if any(fields[k] != value for k, value in selected.items()):
            raise ValueError('terminal supervision changed')
    return fields.get('ActiveState')


def stop_service(identifier):
    state = service_state(identifier)
    if state not in ('inactive', 'failed'):
        subprocess.run(['/usr/bin/systemctl', 'stop', 'soda-terminal-' + identifier + '.service'],
                       check=True, timeout=8, stdin=subprocess.DEVNULL,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if service_state(identifier) not in ('inactive', 'failed') or not cgroup_empty(identifier):
        raise ValueError('terminal cleanup unconfirmed')


def remove_owned_files(path, directory, account):
    # Only exact run-owned files after unit shutdown, never recursive tree removal.
    names = set(os.listdir(directory))
    if names - {'binding', 'lease', 'writer', 'tmux.conf', 'ready', 'screen'}:
        raise ValueError('unexpected terminal files')
    screen = root_directory(path + '/screen')
    try:
        for name in os.listdir(screen):
            info = os.stat(name, dir_fd=screen, follow_symlinks=False)
            if name != 'socket' or not stat.S_ISSOCK(info.st_mode) or info.st_uid != account.pw_uid:
                raise ValueError('unexpected terminal socket')
            os.unlink(name, dir_fd=screen)
    finally:
        os.close(screen)
    for name in names - {'screen'}:
        info = os.stat(name, dir_fd=directory, follow_symlinks=False)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_nlink != 1:
            raise ValueError('unexpected terminal file')
    for name in names - {'screen'}:
        os.unlink(name, dir_fd=directory)
    os.rmdir('screen', dir_fd=directory)
    os.rmdir(path)


def own_terminal(identifier, account, identity, cols, rows, seconds, source_hash):
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
        subprocess.run(['/usr/bin/infocmp', term], check=True, timeout=2,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    path = terminal_path(identifier)
    parent = root_directory(TERMINALS)
    try:
        fcntl.flock(parent, fcntl.LOCK_EX)
        if len(os.listdir(parent)) >= 64:
            raise ValueError('native terminal capacity')
        os.mkdir(identifier, 0o711, dir_fd=parent)  # occupied paths always refuse
    finally:
        os.close(parent)
    directory = root_directory(path)
    os.fchmod(directory, 0o711)
    # A failed partial preparation is retained for inspection, not adopted/retried.
    record = {'account': account_binding(account), 'identity': identity, 'cols': cols, 'rows': rows,
              'deadline': time.monotonic() + seconds}
    new_file(directory, 'binding', line(record))
    new_file(directory, 'lease', b'0')
    new_file(directory, 'writer', b'')
    new_file(directory, 'tmux.conf', TMUX_CONFIG, 0o644)
    os.mkdir('screen', 0o700, dir_fd=directory)
    os.chown(path + '/screen', account.pw_uid, account.pw_gid)
    lease = root_file(directory, 'lease', True)
    lease_value(lease, True)
    reason = 'launch_failed'
    stopped = False

    def stop(_sig, _frame):
        nonlocal stopped
        stopped = True

    for sig in (signal.SIGTERM, signal.SIGHUP, signal.SIGINT):
        signal.signal(sig, stop)
    try:
        # No static/overridden unit is admitted; systemd-run itself refuses an
        # occupied name. All unit settings/command/paths are product constants.
        if service_state(identifier) != 'inactive':
            raise ValueError('terminal unit occupied')
        subprocess.run(['/usr/bin/systemd-run', '--quiet', '--collect',
                        '--unit=soda-terminal-' + identifier, '--description=Soda terminal ' + identifier,
                        '--service-type=notify', '--property=NotifyAccess=main', '--property=User=root',
                        '--slice=system.slice',
                        '--property=KillMode=control-group', '--property=SendSIGKILL=yes',
                        '--property=Restart=no', '--property=TimeoutStartSec=10s',
                        '--property=TimeoutStopSec=3s', '--property=TimeoutAbortSec=3s', '--property=WatchdogSec=10s',
                        '--property=RuntimeMaxSec=43200s', '--property=UMask=0077', '--property=LimitCORE=0',
                        '--property=StandardInput=null', '--property=StandardOutput=null',
                        '--property=StandardError=null', '/usr/bin/python3', '-I', PROGRAM, 'guard', identifier],
                       check=True, timeout=15, stdin=subprocess.DEVNULL,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        ready = read_record(directory, 'ready')
        socket_identity(path + '/screen/socket', account, ready['pid'])
        os.write(1, line({'type': 'ready'}))
        reason = 'disconnected'
        incoming = bytearray()
        while not stopped:
            if time.monotonic() >= min(record['deadline'], lease_value(lease)):
                reason = 'expired'
                break
            if service_state(identifier) != 'active':
                reason = 'exited'
                break
            readable, _, _ = select.select([0], [], [], 1)
            if not readable:
                continue
            data = os.read(0, 4096)
            if not data:
                break
            incoming.extend(data)
            while b'\n' in incoming:
                raw, _, tail = incoming.partition(b'\n')
                incoming = bytearray(tail)
                kind, _ = decode_frame(raw)
                if kind == 'close':
                    stopped = True
                    break
                if kind != 'heartbeat':
                    raise ValueError('owner control only')
                lease_value(lease, True)
            if len(incoming) > FRAME_LIMIT:
                raise ValueError('owner frame size')
    except (OSError, ValueError, subprocess.SubprocessError):
        pass
    finally:
        try:
            stop_service(identifier)
            remove_owned_files(path, directory, account)
        except (OSError, ValueError, subprocess.SubprocessError):
            reason = 'cleanup_unconfirmed'
        os.close(lease)
        os.close(directory)
    os.set_blocking(1, False)
    try:
        os.write(1, line({'type': 'closed', 'reason': reason}))
    except OSError:
        pass
    return 0 if reason in ('disconnected', 'expired', 'exited') else 1


def attach_terminal(identifier, account, identity, cols, rows, seconds):
    path = terminal_path(identifier)
    directory = root_directory(path)
    writer = root_file(directory, 'writer', True)
    lease = root_file(directory, 'lease')
    try:
        fcntl.flock(writer, fcntl.LOCK_EX | fcntl.LOCK_NB)  # never evict a writer
        record = binding(directory, account, identity)
        if time.monotonic() >= min(record['deadline'], lease_value(lease)):
            raise ValueError('terminal expired')
        screen = root_directory(path + '/screen')
        os.close(screen)
        ready = read_record(directory, 'ready')
        sock = path + '/screen/socket'
        if service_state(identifier) != 'active' or socket_identity(sock, account, ready['pid']) != ready['socket']:
            raise ValueError('terminal absent')
        return run_terminal(account, cols, rows, seconds, sock)
    finally:
        os.close(lease)
        os.close(writer)
        os.close(directory)


def main():
    try:
        if len(sys.argv) == 3 and sys.argv[1] == 'guard':
            return guard(sys.argv[2])
        if len(sys.argv) != 9:
            raise ValueError('arguments')
        action, identifier, login, identity, cols, rows, seconds, source_hash = sys.argv[1:]
        terminal_path(identifier)
        identity, cols, rows, seconds = int(identity), int(cols), int(rows), int(seconds)
        if action not in ('create', 'attach') or not dimensions(cols, rows) or not 1 <= seconds <= 43200:
            raise ValueError('terminal bounds')
        signal.alarm(5)
        account = account_for(login, identity)
        signal.alarm(0)
        if action == 'create':
            return own_terminal(identifier, account, identity, cols, rows, seconds, source_hash)
        return attach_terminal(identifier, account, identity, cols, rows, seconds)
    except (OSError, ValueError, KeyError, subprocess.SubprocessError):
        # No exception details, argv or terminal bytes in diagnostics.
        os.write(1, line({'type': 'closed', 'reason': 'launch_failed'}))
        return 1


if __name__ == '__main__':
    sys.exit(main())

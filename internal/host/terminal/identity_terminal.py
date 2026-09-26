"""Fixed broker operations composed with the managed terminal supervisor."""

import base64
import datetime
import fcntl
import hashlib
import json
import os
import re
import signal
import stat
import subprocess
import sys
import time
from typing import TYPE_CHECKING

# The host evaluates this module in the fixed supervisor's namespace.
if TYPE_CHECKING:
    from .project_terminal import (
        TERMINALS,
        account_for,
        binding,
        cgroup_empty,
        cgroup_parent,
        create_terminal,
        line,
        new_file,
        read_record,
        remove_owned_files,
        reserve_terminal,
        root_directory,
        service_state,
        subscription_command,
        tmux_control,
        stop_service,
        terminal_path,
    )


def subscription_path(identifier):
    return terminal_path(identifier) + '/model'


def subscription_mount(path, options):
    subprocess.run(
        ['/usr/bin/mount', '-t', 'tmpfs', '-o', options, 'tmpfs', path],
        check=True,
        timeout=5,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def subscription_prepare(request):
    lease = request['delivery']['lease']
    identifier = lease['execution_id']
    account = account_for(request['login'], int(lease['actor_id']))
    deadline = int(datetime.datetime.fromisoformat(lease['deadline'].replace('Z', '+00:00')).timestamp())
    if not 0 < deadline - int(time.time()) <= 12 * 3600:
        raise ValueError('deadline')
    parent = root_directory(TERMINALS)
    try:
        reserve_terminal(
            identifier,
            account,
            int(lease['actor_id']),
            request['cols'],
            request['rows'],
            'Codex',
            request['source_hash'],
            request['scope'],
        )
        try:
            return subscription_provision(request, lease, account, deadline)
        except (OSError, ValueError, KeyError, subprocess.SubprocessError):
            stop_service(identifier, account)
            subscription_retire(lease, account)
            raise
    finally:
        os.close(parent)


def subscription_provision(request, lease, account, deadline):
    identifier = lease['execution_id']
    path = terminal_path(identifier)
    directory = root_directory(path)
    try:
        native = {
            'kind': 'terminal',
            'id': identifier,
            'project': request['container'],
            'login': account.pw_name,
            'generation': lease['generation'],
        }
        lease['binding'] = native
        profile = {'lease': lease, 'binding': native, 'deadline': deadline, 'scope': request['scope']}
        new_file(directory, 'subscription', line(profile))
        os.mkdir('model', 0o711, dir_fd=directory)
        model = path + '/model'
        os.chmod(model, 0o711)
        os.mkdir(model + '/harness', 0o755)
        os.mkdir(model + '/auth', 0o700)
        subscription_mount(model + '/harness', 'size=1g,nosuid,nodev,mode=755')
        subscription_mount(model + '/auth', 'size=64m,nosuid,nodev,mode=700')
        os.chown(model + '/auth', account.pw_uid, account.pw_gid)
    finally:
        os.close(directory)

    create_terminal(
        identifier, account, int(lease['actor_id']), request['cols'], request['rows'], 'Codex', request['scope']
    )
    subscription_check_unit(lease, account, True)
    return {'lease': lease, 'credential': ''}


def subscription_profile(lease):
    identifier = lease['execution_id']
    directory = root_directory(terminal_path(identifier))
    try:
        profile = read_record(directory, 'subscription')
        if profile['lease'] != lease or profile['binding'] != lease['binding']:
            raise ValueError('lease changed')
        account = account_for(lease['binding']['login'], int(lease['actor_id']))
        binding(directory, account, int(lease['actor_id']))
        return profile, account
    finally:
        os.close(directory)


def subscription_invocation(identifier):
    value = (
        subprocess.check_output(
            [
                '/usr/bin/systemctl',
                'show',
                '--value',
                '--property=InvocationID',
                'soda-terminal-' + identifier + '.service',
            ],
            timeout=2,
            stderr=subprocess.DEVNULL,
        )
        .decode()
        .strip()
    )
    return value


def subscription_check_unit(lease, account, running):
    identifier = lease['execution_id']
    state = service_state(identifier, account)
    directory = root_directory(terminal_path(identifier))
    try:
        try:
            expected = read_record(directory, 'subscription-unit')['invocation_id']
        except FileNotFoundError:
            if state != 'inactive' or not cgroup_empty(identifier):
                raise ValueError('unexpected unit')
            if running:
                raise ValueError('session not started')
            return False
        observed = subscription_invocation(identifier)
        if not re.fullmatch(r'[0-9a-f]{32}', expected) or observed and observed != expected:
            raise ValueError('unit incarnation changed')
        if state in ('inactive', 'failed') and cgroup_empty(identifier):
            if running:
                raise ValueError('session ended')
            return False
        if observed != expected:
            raise ValueError('unit incarnation absent')
        if running and state != 'active':
            raise ValueError('session unavailable')
        return True
    finally:
        os.close(directory)


def subscription_auth_directory(lease, account):
    model = root_directory(subscription_path(lease['execution_id']))
    try:
        directory = os.open('auth', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=model)
    finally:
        os.close(model)
    info = os.fstat(directory)
    if info.st_uid != account.pw_uid or info.st_gid != account.pw_gid or info.st_mode & 0o077:
        os.close(directory)
        raise ValueError('unsafe auth directory')
    return directory


def subscription_seed(lease, account, encoded):
    state = base64.b64decode(encoded, validate=True)
    if not 0 < len(state) <= 256 * 1024:
        raise ValueError('credential size')
    json.loads(state)
    directory = subscription_auth_directory(lease, account)
    try:
        new_file(directory, 'auth.json', state)
        os.chown('auth.json', account.pw_uid, account.pw_gid, dir_fd=directory, follow_symlinks=False)
    finally:
        os.close(directory)


def subscription_stage(request, lease, profile, account):
    if profile['deadline'] <= time.time():
        raise ValueError('session expired')
    subscription_check_unit(lease, account, True)
    subscription_seed(lease, account, request['delivery']['credential'])
    return {'lease': lease, 'credential': ''}


def subscription_harness_digest(harness):
    directory = root_directory(harness + '/bin')
    try:
        fd = os.open('codex', os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
    finally:
        os.close(directory)
    with os.fdopen(fd, 'rb') as binary:
        info = os.fstat(binary.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022 or info.st_nlink != 1:
            raise ValueError('unsafe harness executable')
        digest = hashlib.sha256()
        for chunk in iter(lambda: binary.read(128 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def subscription_start(request, lease, profile, account):
    if profile['deadline'] <= time.time():
        raise ValueError('session expired')
    path = terminal_path(lease['execution_id'])
    harness = subscription_path(lease['execution_id']) + '/harness'
    subprocess.run(
        ['/usr/bin/chown', '-R', '0:0', harness],
        check=True,
        timeout=10,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    subprocess.run(
        ['/usr/bin/chmod', '-R', 'go-w', harness],
        check=True,
        timeout=10,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    digest = subscription_harness_digest(harness)
    if digest != request['harness_sha256']:
        raise ValueError('guest harness digest differs')
    directory = root_directory(path)
    try:
        binding(directory, account, int(lease['actor_id']))
    finally:
        os.close(directory)
    subscription_check_unit(lease, account, True)
    directory = root_directory(path)
    try:
        new_file(directory, 'subscription-started', b'1')
    finally:
        os.close(directory)
    tmux_control(
        account,
        path + '/screen/socket',
        'respawn-pane',
        '-k',
        '-t',
        'soda:0.0',
        '-c',
        account.pw_dir,
        *subscription_command(path),
    )
    subscription_check_unit(lease, account, True)
    return {'lease': lease, 'credential': ''}


def subscription_cgroup(identifier):
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
        raise ValueError('unsafe cgroup')
    return group


def subscription_kernel_write(directory, name, value):
    fd = os.open(name, os.O_WRONLY | os.O_NOFOLLOW, dir_fd=directory)
    try:
        if os.write(fd, value) != len(value):
            raise ValueError('short cgroup write')
    finally:
        os.close(fd)


def subscription_freeze(group):
    subscription_kernel_write(group, 'cgroup.freeze', b'1\n')
    until = time.monotonic() + 5
    while time.monotonic() < until:
        fd = os.open('cgroup.events', os.O_RDONLY | os.O_NOFOLLOW, dir_fd=group)
        try:
            values = dict(row.split() for row in os.read(fd, 4096).decode().splitlines())
        finally:
            os.close(fd)
        if values.get('frozen') == '1':
            return
        time.sleep(0.02)
    raise ValueError('freeze unconfirmed')


def subscription_capture(lease, account):
    directory = subscription_auth_directory(lease, account)
    try:
        fd = os.open('auth.json', os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
        try:
            info = os.fstat(fd)
            if (
                not stat.S_ISREG(info.st_mode)
                or info.st_uid != account.pw_uid
                or info.st_nlink != 1
                or info.st_mode & 0o077
            ):
                raise ValueError('unsafe credential file')
            state = os.read(fd, 256 * 1024 + 1)
        finally:
            os.close(fd)
    finally:
        os.close(directory)
    if not 0 < len(state) <= 256 * 1024:
        raise ValueError('credential size')
    json.loads(state)
    return state


def subscription_remove_model(directory):
    for name in ('subscription', 'subscription-started', 'subscription-unit'):
        try:
            os.unlink(name, dir_fd=directory)
        except FileNotFoundError:
            pass
    for name in ('model/auth', 'model/harness', 'model'):
        try:
            os.rmdir(name, dir_fd=directory)
        except FileNotFoundError:
            pass


def subscription_retire(lease, account):
    identifier = lease['execution_id']
    path = terminal_path(identifier)
    for name in ('auth', 'harness'):
        mount = subscription_path(identifier) + '/' + name
        if os.path.ismount(mount):
            subprocess.run(
                ['/usr/bin/umount', mount], check=True, timeout=5, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
            )
    directory = root_directory(path)
    try:
        subscription_remove_model(directory)
        remove_owned_files(path, directory, account)
    finally:
        os.close(directory)


def subscription_finish(action, lease, profile, account):
    live = subscription_check_unit(lease, account, False)
    group = subscription_cgroup(lease['execution_id']) if live else None
    state = b''
    try:
        if group is not None:
            subscription_freeze(group)
        if action == 'finish':
            state = subscription_capture(lease, account)
    finally:
        if group is not None:
            try:
                subscription_kernel_write(group, 'cgroup.kill', b'1\n')
            finally:
                os.close(group)
        stop_service(lease['execution_id'], account)
    if action == 'stop':
        subscription_retire(lease, account)
    return {'lease': lease, 'credential': base64.b64encode(state).decode()}


def subscription_lookup(request):
    lease = request['delivery']['lease']
    account = account_for(request['login'], int(lease['actor_id']))
    try:
        directory = root_directory(terminal_path(lease['execution_id']))
    except FileNotFoundError:
        return {'lease': {}, 'credential': ''}
    try:
        binding(directory, account, int(lease['actor_id']))
        if 'subscription' not in os.listdir(directory):
            return {'lease': {}, 'credential': ''}
        profile = read_record(directory, 'subscription')
        if profile['lease']['execution_id'] != lease['execution_id'] or int(profile['lease']['actor_id']) != int(
            lease['actor_id']
        ):
            raise ValueError('lookup binding')
        return {'lease': profile['lease'], 'credential': ''}
    finally:
        os.close(directory)


def subscription_resolve(action, lease):
    try:
        return subscription_profile(lease)
    except FileNotFoundError:
        if action != 'stop':
            raise
        account = account_for(lease['binding']['login'], int(lease['actor_id']))
        if service_state(lease['execution_id'], account) not in ('inactive', 'failed') or not cgroup_empty(
            lease['execution_id']
        ):
            raise ValueError('termination unknown')
        return None


def subscription_dispatch(request):
    action = request['action']
    if action == 'prepare':
        return subscription_prepare(request)
    if action == 'lookup':
        return subscription_lookup(request)
    lease = request['delivery']['lease']
    resolved = subscription_resolve(action, lease)
    if resolved is None:
        return {'lease': lease, 'credential': ''}
    profile, account = resolved
    if action == 'stage':
        return subscription_stage(request, lease, profile, account)
    if action == 'start':
        return subscription_start(request, lease, profile, account)
    if action == 'validate':
        if profile['deadline'] <= time.time():
            raise ValueError('session expired')
        subscription_check_unit(lease, account, True)
        return {'lease': lease, 'credential': ''}
    if action in ('finish', 'stop'):
        return subscription_finish(action, lease, profile, account)
    raise ValueError('operation')


def subscription_main():
    try:
        signal.alarm(45)
        body = sys.stdin.buffer.read(512 * 1024 + 1)
        if len(body) > 512 * 1024:
            raise ValueError('request size')
        request = json.loads(body)
        parent = root_directory(TERMINALS)
        try:
            fcntl.flock(parent, fcntl.LOCK_EX)
            result = subscription_dispatch(request)
        finally:
            os.close(parent)
        os.write(1, json.dumps(result).encode())
        return 0
    except (OSError, ValueError, KeyError, subprocess.SubprocessError):
        os.write(2, b'managed Codex operation failed\n')
        return 1


raise SystemExit(subscription_main())

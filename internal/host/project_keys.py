"""Explicit compare-and-swap of one existing Soda-managed account's SSH key file.

The host preloads the fixed project_terminal module in memory; no user command,
account creation, privilege/group change, home write or process termination here.
"""
import fcntl
import hashlib
import json
import os
import re
import stat
import sys
import uuid
from project_terminal import account_for


def directory(parts):
    fd = os.open('/', os.O_RDONLY | os.O_DIRECTORY)
    try:
        for part in parts:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            os.close(fd)
            fd = child
            info = os.fstat(fd)
            if info.st_uid != 0 or info.st_gid != 0 or info.st_mode & 0o2022:
                raise ValueError('unsafe key directory')
        return fd
    except BaseException:
        os.close(fd)
        raise


def canonical_lines(raw):
    if raw == b'':
        return []
    if len(raw) > 65536 or not raw.endswith(b'\n'):
        raise ValueError('not a managed key file')
    text = raw.decode('ascii')
    keys = text[:-1].split('\n')
    if len(keys) > 32 or len(set(keys)) != len(keys) or any(not re.fullmatch(r'(ssh-[\w-]+|ecdsa-[\w-]+|sk-[\w@.-]+) [A-Za-z0-9+/=]+', key) for key in keys):
        raise ValueError('not a managed key file')
    return keys


def read_keys(fd, login):
    key = os.open(login, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=fd)
    try:
        info = os.fstat(key)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_gid != 0 or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o644:
            raise ValueError('unsafe managed key file')
        raw = os.read(key, 65537)
        return raw, canonical_lines(raw), info
    finally:
        os.close(key)


def update(data):
    if set(data) != {'login', 'identity', 'apply', 'revision', 'keys'} or type(data['apply']) is not bool or type(data['identity']) is not int:
        raise ValueError('invalid key operation')
    login = data['login']
    account_for(login, data['identity'])
    keys = data['keys']
    if not isinstance(keys, list) or any(not isinstance(k, str) for k in keys):
        raise ValueError('invalid keys')
    desired = ('\n'.join(keys) + ('\n' if keys else '')).encode('ascii')
    if canonical_lines(desired) != keys:
        raise ValueError('invalid keys')
    fd = directory(('etc', 'ssh', 'authorized_keys'))
    pending = None
    try:
        # Serializes this fixed operation; root-owned directory is never rewritten.
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        raw, installed, old = read_keys(fd, login)
        revision = hashlib.sha256(raw).hexdigest()
        if not data['apply']:
            if data['revision'] or keys:
                raise ValueError('invalid preview')
            return {'revision': revision, 'keys': installed}
        if data['revision'] != revision:
            raise ValueError('key file changed since preview')
        # Preview explicitly confirms the exact managed file/set, including any
        # canonical root edits already present. Noncanonical content is refused,
        # never merged/adopted. Edits after preview must fail compare-and-swap.
        pending = '.soda-keys-' + uuid.uuid4().hex
        out = os.open(pending, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600, dir_fd=fd)
        with os.fdopen(out, 'wb') as stream:
            stream.write(desired)
            os.fchmod(stream.fileno(), 0o644)
            os.fsync(stream.fileno())
        current, _, info = read_keys(fd, login)
        if current != raw or (info.st_dev, info.st_ino, info.st_mtime_ns) != (old.st_dev, old.st_ino, old.st_mtime_ns):
            raise ValueError('key file changed during update')
        os.replace(pending, login, src_dir_fd=fd, dst_dir_fd=fd)
        pending = None
        os.fsync(fd)
        actual, installed, _ = read_keys(fd, login)
        if actual != desired:
            raise ValueError('key result unconfirmed')
        return {'revision': hashlib.sha256(actual).hexdigest(), 'keys': installed}
    finally:
        if pending is not None:
            os.unlink(pending, dir_fd=fd)  # Only this operation's exclusive temp file.
        os.close(fd)


def key_main():
    try:
        def unique(pairs):
            result = {}
            for key, value in pairs:
                if key in result:
                    raise ValueError('duplicate field')
                result[key] = value
            return result
        data = json.loads(sys.stdin.buffer.read(65537), object_pairs_hook=unique)
        result = update(data)
        print(json.dumps(result, separators=(',', ':')))
        return 0
    except (OSError, ValueError, KeyError, TypeError, RecursionError):
        # No request bytes, key contents, paths or exception body in diagnostics.
        sys.stderr.write('native key operation not confirmed\n')
        return 1


if __name__ == '__main__':
    sys.exit(key_main())

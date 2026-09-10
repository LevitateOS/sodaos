#!/usr/bin/env python3
"""Opt-in read-only host runner observation; no provisioning or cleanup.

Transport supplies this file on stdin to python3 -I - TARGET ID [...]. Output is
selected public facts and aggregate private-tree hashes, never credentials/paths
inside work trees. Hashes are bounded observations, not quiesced state backups.
"""
import fcntl
import hashlib
import json
import os
from pathlib import Path
import platform
import pwd
import re
import socket
import stat
import subprocess
import sys
import time

ROOT = Path('/var/lib/soda/runners')
CLI = '/usr/local/libexec/soda/soda-runners'
ID = re.compile(r'[a-z][a-z0-9-]{0,15}')


def command(args, body=None):
    result = subprocess.run(args, input=body, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, timeout=15)
    if result.returncode or len(result.stdout) > 65536:
        raise RuntimeError('native observation unavailable')
    return result.stdout.decode('utf-8')


def tree_digest(root):
    """Descriptor-relative reads never follow a job-created link outside its tree."""
    digest = hashlib.sha256()
    count = total = 0
    deadline = time.monotonic() + 20

    def visit(parent, name, relative):
        nonlocal count, total
        count += 1
        if count > 20000 or time.monotonic() > deadline:
            raise RuntimeError('state observation bound exceeded')
        before = os.stat(name, dir_fd=parent, follow_symlinks=False)
        digest.update(json.dumps([relative, before.st_uid, before.st_gid,
                                  before.st_mode, before.st_size if stat.S_ISREG(before.st_mode) else 0]).encode())
        if stat.S_ISLNK(before.st_mode):
            digest.update(os.fsencode(os.readlink(name, dir_fd=parent)))
            return
        if not (stat.S_ISDIR(before.st_mode) or stat.S_ISREG(before.st_mode)):
            raise RuntimeError('unsupported live state entry')
        flags = os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK
        if stat.S_ISDIR(before.st_mode):
            flags |= os.O_DIRECTORY
        fd = os.open(name, flags, dir_fd=parent)
        try:
            opened = os.fstat(fd)
            if (before.st_dev, before.st_ino, before.st_mode) != (opened.st_dev, opened.st_ino, opened.st_mode):
                raise RuntimeError('state entry changed during observation')
            if stat.S_ISDIR(opened.st_mode):
                names = sorted(os.listdir(fd))
                for child in names:
                    visit(fd, child, relative + '/' + child)
                if names != sorted(os.listdir(fd)):
                    raise RuntimeError('state directory changed during observation')
            else:
                while True:
                    block = os.read(fd, 1024 * 1024)
                    if not block:
                        break
                    total += len(block)
                    if total > 512 * 1024 * 1024 or time.monotonic() > deadline:
                        raise RuntimeError('state observation bound exceeded')
                    digest.update(block)
                after = os.fstat(fd)
                if (opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (after.st_size, after.st_mtime_ns, after.st_ctime_ns):
                    raise RuntimeError('state file changed during observation')
        finally:
            os.close(fd)
    parent = os.open(root.parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        visit(parent, root.name, '.')
    finally:
        os.close(parent)
    return {'sha256': digest.hexdigest(), 'entries': count, 'bytes': total}


def runner_state(identifier, root=ROOT, account=pwd.getpwnam):
    if not ID.fullmatch(identifier):
        raise ValueError('invalid runner ID')
    directory = root / identifier
    try:
        person = account('soda-runner-' + identifier)
    except KeyError:
        person = None
    if directory.is_symlink():
        raise RuntimeError('runner root must not be a symlink')
    if not directory.exists():
        if person is not None:
            raise RuntimeError('account remains without state')
        return {'present': False}
    if person is None:
        raise RuntimeError('state remains without account')
    state = directory / 'state'
    protected = [('state', state, 0o700, True),
                 ('token', state / 'forgejo-token', 0o600, False),
                 ('configuration', state / 'forgejo-runner.yml', 0o600, False)]
    registration = {'descriptor': tree_digest(directory / 'descriptor.json')}
    for label, item, mode, folder in protected:
        metadata = item.lstat()
        if (stat.S_IMODE(metadata.st_mode) != mode
                or metadata.st_uid != person.pw_uid or metadata.st_gid != person.pw_gid
                or not (stat.S_ISDIR(metadata.st_mode) if folder else stat.S_ISREG(metadata.st_mode))):
            raise RuntimeError('runner private state ownership/mode differs')
        if not folder:
            registration[label] = tree_digest(item)
    return {'present': True, 'uid': person.pw_uid, 'gid': person.pw_gid,
            'home': person.pw_dir, 'shell': person.pw_shell,
            'registration': registration, 'tree': tree_digest(directory)}


def job_proof(identifier, observation, root=ROOT, account=pwd.getpwnam):
    if not ID.fullmatch(identifier) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_-]{0,79}', observation):
        raise ValueError('invalid job observation')
    person = account('soda-runner-' + identifier)
    proof = root / identifier / 'state' / 'work' / ('soda-native-proof-' + observation + '.json')
    try:
        parent = os.open(proof.parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    except FileNotFoundError:
        # A newly registered listener may not have run any job yet. The caller
        # independently checks complete account/registration state first.
        return None
    try:
        try:
            fd = os.open(proof.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
        except FileNotFoundError:
            return None
        with os.fdopen(fd) as stream:
            metadata = os.fstat(stream.fileno())
            if not stat.S_ISREG(metadata.st_mode) or stat.S_IMODE(metadata.st_mode) != 0o600 or metadata.st_uid != person.pw_uid:
                raise RuntimeError('job proof ownership/mode differs')
            text = stream.read(4097)
            if len(text) > 4096:
                raise RuntimeError('job proof exceeds bound')
    finally:
        os.close(parent)
    def unique_fields(pairs):
        fields = dict(pairs)
        if len(fields) != len(pairs):
            raise RuntimeError('ambiguous job proof')
        return fields
    record = json.loads(text, object_pairs_hook=unique_fields)
    if (not isinstance(record, dict) or set(record) != {'observation', 'account', 'uid', 'pid', 'start', 'steps'}
            or record['observation'] != observation or record['account'] != person.pw_name
            or type(record['uid']) is not int or record['uid'] != person.pw_uid
            or type(record['pid']) is not int or record['pid'] <= 0
            or not isinstance(record['start'], str) or not re.fullmatch(r'[0-9]+', record['start'])
            or type(record['steps']) is not int or record['steps'] not in (1, 2)):
        raise RuntimeError('job proof does not match fixture')
    process = Path('/proc') / str(record['pid'])
    try:
        start = (process / 'stat').read_text().rsplit(') ', 1)[1].split()[19]
        alive = start == record['start'] and process.stat().st_uid == person.pw_uid
        if alive and not any(('soda-runner@' + identifier + '.service') in line.split(':', 2)[-1].split('/')
                             for line in (process / 'cgroup').read_text().splitlines()):
            raise RuntimeError('job process outside runner unit')
    except FileNotFoundError:
        alive = False
    return {**record, 'alive': alive}


def main(argv):
    if len(argv) < 2:
        raise ValueError('target and exact runner IDs required')
    target, *ids = argv
    if (not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{0,252}', target)
            or os.environ.get('SODA_NATIVE_VALIDATE') != target
            or socket.gethostname() != target or os.getuid() != 0 or os.geteuid() != 0
            or platform.system() != 'Linux'):
        raise ValueError('explicit native root target required')
    if len(ids) > 65 or len(set(ids)) != len(ids) or not all(ID.fullmatch(i) for i in ids):
        raise ValueError('exact distinct runner IDs required')
    observation = os.environ.get('SODA_RUNNER_OBSERVATION')
    if observation is not None and not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_-]{0,79}', observation):
        raise ValueError('invalid job observation')
    # Existing production CLI owns config/root admission and the list protocol.
    before = json.loads(command([CLI, 'list'], b'{}\n'))
    lock = os.open('/run/lock/soda/runners.lock', os.O_RDWR | os.O_NOFOLLOW)
    try:
        deadline = time.monotonic() + 5
        while True:
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() >= deadline:
                    raise RuntimeError('runner management is busy')
                time.sleep(.025)
        states = {i: runner_state(i) for i in ids}
        proof = job_proof(ids[0], observation) if observation else None
        # Only fixed confinement properties, never Environment, credentials or logs.
        confinement = {}
        properties = 'User,Group,NoNewPrivileges,CapabilityBoundingSet,ProtectSystem,ReadWritePaths'
        for identifier in ids:
            if states[identifier]['present']:
                text = command(['systemctl', 'show', '--all', '--property=' + properties,
                                'soda-runner@' + identifier + '.service'])
                fields = dict(line.split('=', 1) for line in text.splitlines())
                if set(fields) != set(properties.split(',')):
                    raise RuntimeError('confinement observation incomplete')
                # Retain only whether the expected effective boundary was observed.
                confinement[identifier] = (fields == {
                    'User': 'soda-runner-' + identifier, 'Group': 'soda-runners',
                    'NoNewPrivileges': 'yes', 'CapabilityBoundingSet': '',
                    'ProtectSystem': 'strict', 'ReadWritePaths': str(ROOT / identifier / 'state')})
    finally:
        os.close(lock)
    after = json.loads(command([CLI, 'list'], b'{}\n'))
    if before != after:
        raise RuntimeError('inventory changed across state observation')
    # Strip saved registration URLs: only the configured public origin is emitted.
    for row in after['runners']:
        row['registration_url'] = after['forgejo_url']
    versions = command(['rpm', '-q', '--qf', '%{NAME} %{VERSION}-%{RELEASE}.%{ARCH}\n',
                        'forgejo-runner', 'systemd'])
    print(json.dumps({'target': target, 'architecture': platform.machine(), 'inventory': after,
                      'states': states, 'confinement': confinement, 'packages': versions.splitlines(), 'job_proof': proof,
                      'consistency': 'bounded observation, not a quiesced backup'}))


if __name__ == '__main__':
    try:
        main(sys.argv[1:])
    except Exception:
        # Neither private filenames nor subprocess diagnostics enter evidence.
        print('Runner state observation unavailable; preserve partial state.', file=sys.stderr)
        sys.exit(1)

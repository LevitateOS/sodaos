#!/usr/bin/env python3
"""Bounded, read-only U08 state snapshot; run as root INSIDE a selected project.

No environment/secret/DB credential/shadow/private-key contents are exported.
The caller supplies this on stdin over the selected administrator SSH session.
"""
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import stat
import subprocess


def command(*args, env=None):
    r = subprocess.run(args, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
    if r.returncode:
        raise RuntimeError('Required snapshot command failed: ' + ' '.join(args[:5]))
    if len(r.stdout) > 4 * 1024 * 1024:
        raise RuntimeError('Snapshot output exceeded bound')
    return r.stdout.decode().strip()


def entry(path, contents=True):
    p = Path(path); s = p.lstat()
    result = {'uid': s.st_uid, 'gid': s.st_gid, 'mode': stat.S_IMODE(s.st_mode)}
    if p.is_symlink(): result['link'] = os.readlink(p)
    elif p.is_file() and contents:
        if s.st_size > 512 * 1024 * 1024: raise RuntimeError('Snapshot file too large')
        h = hashlib.sha256()
        with p.open('rb') as f:
            while data := f.read(1024 * 1024): h.update(data)
        result['sha256'] = h.hexdigest()
    elif p.is_file(): result['size'] = s.st_size
    elif not p.is_dir(): raise RuntimeError('Unsupported snapshot file')
    return result


def main():
    assert os.geteuid() == 0 and Path('/run/.containerenv').exists()
    data = {'files': {}, 'people': {}, 'git': {}, 'workloads': [], 'volumes': []}
    paths = ['/etc/passwd', '/etc/group', '/etc/ssh/sshd_config.d/10-soda.conf',
             '/etc/ssh/ssh_host_ed25519_key.pub', '/etc/mise/config.toml',
             '/etc/containers/containers.conf', '/etc/containers/storage.conf',
             '/etc/systemd/system/soda-podman.service', '/usr/libexec/soda/project-init',
             '/usr/libexec/soda/project-account', '/etc/sudoers.d/soda-project']
    optional = Path('/etc/systemd/system/soda-podman.socket')
    if optional.exists(): paths.append(str(optional))
    for base in ['/etc/ssh/authorized_keys', '/var/lib/soda/accounts']:
        p = Path(base); assert p.is_dir()
        paths.extend(str(x) for x in sorted(p.iterdir()) if x.is_file())
    for login in ['u08-alice-8417', 'u08-bob-8417']:
        home = Path('/home') / login
        if not home.exists(): continue
        data['people'][login] = {'identity': command('id', login), 'home': entry(home), 'shared': entry(home / 'shared')}
        checkout = home / 'u08-personal-checkout'
        if checkout.exists():
            # Only the known run-owned repository; export hashes, never content.
            files = list(checkout.rglob('*'))
            if len(files) > 10000: raise RuntimeError('Checkout snapshot exceeded bound')
            for file in sorted(files):
                if '.git' in file.relative_to(checkout).parts: continue
                if file.is_file() or file.is_symlink(): paths.append(str(file))
            data['git'][login] = {'head': command('runuser', '-u', login, '--', 'git', '-C', str(checkout), 'rev-parse', 'HEAD'),
                                  'status': command('runuser', '-u', login, '--', 'git', '-C', str(checkout), 'status', '--porcelain=v1'),
                                  'refs': command('runuser', '-u', login, '--', 'git', '-C', str(checkout), 'show-ref')}
        for private in ['.ssh/u08-personal-git/identity', '.config/soda-u08-workload/database-password']:
            f = home / private
            if f.exists(): data['files'][str(f)] = entry(f, contents=False)
        for probe in home.glob('u08-access-*'):
            assert probe.is_dir() and not probe.is_symlink()
            paths.extend(str(x) for x in sorted(probe.iterdir()) if x.is_file())
    shared = Path('/srv/project/shared')
    paths.append(str(shared))
    for probe in shared.glob('u08-shared-*'):
        paths.append(str(probe)); paths.append(str(probe / 'members'))
    node = Path('/opt/mise/installs/node/24.20.0/bin/node')
    if node.exists(): paths.append(str(node))
    for p in sorted(set(paths)): data['files'][p] = entry(p)
    # Fixed project-local API. A second project without initialized workload
    # storage still gets complete account/rootfs coverage, not fabricated lists.
    assert os.environ.get('SODA_EXPECT_WORKLOADS') in ('0', '1'), 'Caller must declare required workload observations'
    if os.environ['SODA_EXPECT_WORKLOADS'] == '1':
        podman = ['podman', '--url', 'unix:///run/soda-podman/podman.sock']
        ids = command(*podman, 'ps', '-aq', '--no-trunc').splitlines()
        assert len(ids) >= 2, 'Missing required workload containers'
        for identifier in sorted(ids):
            raw = command(*podman, 'container', 'inspect', '--format', '{{json .ID}} {{json .Name}} {{json .Image}} {{json .HostConfig.NetworkMode}} {{json .Mounts}}', identifier)
            data['workloads'].append(raw)
        data['volumes'] = sorted(command(*podman, 'volume', 'ls', '--format', '{{.Name}}').splitlines())
        databases = [name for name in command(*podman, 'ps', '--format', '{{.Names}}').splitlines() if name in ['u08-projectnet-database', 'workload_database_1']]
        assert databases, 'No running database; restore existing workload before snapshot'
        data['database'] = {}
        ip = os.environ['SODA_PROJECT_IP']
        assert ipaddress.ip_address(ip) in ipaddress.ip_network('10.89.0.0/24')
        passfiles = sorted(Path('/home/u08-alice-8417/.config').glob('u08-db-client-*/pgpass'))
        assert passfiles, 'Missing native client credential input'
        passfile = passfiles[-1]
        assert not passfile.is_symlink() and not passfile.stat().st_mode & 0o077
        assert passfile.read_text().split(':', 1)[0] == ip, 'Cached credential endpoint differs from live native target'
        for name in databases:
            # Native TCP client, also used by both real developers; no dependency
            # on exec into a different-UID workload and no credential export.
            data['database'][name] = command('psql', '-X', '-w', '-h', ip, '-p', '5432', '-U', 'developer', '-d', 'soda_example', '-At', '-c', 'SELECT run_id,value FROM soda_u08_probe ORDER BY run_id', env={**os.environ, 'PGPASSFILE': str(passfile), 'PGCONNECT_TIMEOUT': '5'})
            assert data['database'][name], 'Empty required database snapshot'
    print(json.dumps(data, sort_keys=True))


if __name__ == '__main__':
    try:
        main()
    except Exception as failure:
        # Fail closed without printing exception values/private native output.
        import sys
        detail = str(failure) if isinstance(failure, (RuntimeError, AssertionError)) else ''
        print('Project snapshot failed: ' + type(failure).__name__ + ' ' + detail, file=sys.stderr)
        sys.exit(1)

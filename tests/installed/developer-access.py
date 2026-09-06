#!/usr/bin/env python3
"""Opt-in U08 direct project-IP SSH/PTY/SCP/SFTP and sudo checks.

Uses retained private fixture inputs; never transfers private keys. Creates only
new run-owned probe directories in the three selected project memberships and
retains them as evidence. No lifecycle changes or automatic cleanup.
"""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import uuid


def main():
    assert os.environ.get('SODA_NATIVE_VALIDATE') == 'soda-test'
    assert len(sys.argv) == 2
    root = Path(sys.argv[1])
    assert root.is_absolute()
    st = root.lstat()
    assert root.is_dir() and not root.is_symlink() and st.st_uid == os.getuid() and not st.st_mode & 0o077
    target = json.loads((root / 'target.json').read_text()) if (root / 'target.json').exists() else None
    run_name = 'u08-access-' + uuid.uuid4().hex
    output = root / run_name
    output.mkdir(mode=0o700)
    payload = (run_name + '\n').encode() * 8192
    source = output / 'payload'
    source.write_bytes(payload)
    source.chmod(0o600)
    results = []

    def execute(args, data=None):
        return subprocess.run(args, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)

    def checked(args, data=None):
        result = execute(args, data)
        if result.returncode:
            # Never dump SSH command logs, environment or credential material.
            raise RuntimeError('Selected access operation failed (exit %d)' % result.returncode)
        return result.stdout

    for who in ('alice', 'bob'):
        login = 'u08-' + who + '-8417'
        key = root / who / 'development'
        ks = key.lstat()
        assert key.is_file() and not key.is_symlink() and ks.st_uid == os.getuid() and not ks.st_mode & 0o077
        known = root / (login + '-known-hosts')
        assert known.is_file() and not known.is_symlink()
        connections = json.loads((root / (login + '-connections.json')).read_text())
        assert len(connections) == (1 if target or who == 'alice' else 2)
        bindings = json.loads((root / 'observed-bindings.json').read_text())
        for connection in connections:
            identifier, ip = connection['id'], connection['ip']
            assert re.fullmatch(r'p[0-9a-f]{24}', identifier)
            assert re.fullmatch(r'10\.89\.0\.[0-9]{1,3}', ip) and 1 < int(ip.split('.')[-1]) < 255
            assert connection['login'] == login
            binding = next(b for b in bindings if b['environmentID'] == identifier)
            target = login + '@' + ip
            options = ['-F', '/dev/null', '-o', 'ForwardAgent=no', '-o', 'ClearAllForwardings=yes',
                       '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', '-o', 'IdentitiesOnly=yes',
                       '-o', 'StrictHostKeyChecking=yes', '-o', 'UserKnownHostsFile=' + str(known), '-i', str(key)]
            ssh = ['ssh', *options]
            print('Checking direct project access:', login, identifier, ip, flush=True)
            identity = checked(ssh + [target, 'id -un; id -u; printf "%s\\n" "$HOME"']).decode().splitlines()
            assert identity[0] == login and int(identity[1]) != 0 and identity[2] == '/home/' + login
            mapping = checked(ssh + [target, '/usr/bin/head -n 1 /proc/self/uid_map']).decode().split()
            assert int(mapping[0]) == 0 and int(mapping[1]) != 0, 'Project root must not map to host root'
            terminal = checked(ssh + ['-tt', target], b'test -t 0 && printf "\\nSODA-PTY:%s\\n" "$(id -un)"\nexit\n')
            assert ('SODA-PTY:' + login).encode() in terminal.replace(b'\r', b'').splitlines()
            assert checked(ssh + [target, 'command -v sudo']).strip() == b'/usr/bin/sudo'
            privilege = execute(ssh + [target, 'sudo -n /usr/bin/id -u'])
            administrator = binding['login'] == login
            if administrator:
                assert privilege.returncode == 0 and privilege.stdout.strip() == b'0'
            else:
                assert privilege.returncode == 1 and (b'password is required' in privilege.stderr or b'not allowed' in privilege.stderr or b'not in the sudoers' in privilege.stderr)
            destination = '/home/' + login + '/' + run_name
            checked(ssh + [target, 'umask 077; mkdir ' + destination])
            prefix = login + '-' + identifier
            received = output / (prefix + '-scp')
            checked(['scp', *options, str(source), target + ':' + destination + '/scp'])
            checked(['scp', *options, target + ':' + destination + '/scp', str(received)])
            assert received.read_bytes() == payload
            sftp_copy = output / (prefix + '-sftp')
            # Local paths are quoted for the native SFTP batch grammar.
            def quote(p):
                s = str(p)
                assert not any(c in s for c in '\r\n')
                return '"' + s.replace('\\', '\\\\').replace('"', '\\"') + '"'
            batch = 'put ' + quote(source) + ' ' + destination + '/sftp\nget ' + destination + '/sftp ' + quote(sftp_copy) + '\n'
            checked(['sftp', *options, '-b', '-', target], batch.encode())
            assert sftp_copy.read_bytes() == payload
            results.append({'login': login, 'project': identifier, 'ip': ip, 'project_administrator': administrator,
                            'direct_ssh': True, 'interactive_pty': True, 'scp_roundtrip': True, 'sftp_roundtrip': True,
                            'probe_directory': destination})
            (output / 'results.json').write_text(json.dumps(results, indent=2))
    if target:
        bob_ip = target['isolation_ip']
        assert re.fullmatch(r'10\.89\.0\.[0-9]{1,3}', bob_ip) and 1 < int(bob_ip.split('.')[-1]) < 255
    else:
        bob_project = next(b for b in bindings if b['login'] == 'u08-bob-8417')
        bob_ip = next(c['ip'] for c in connections if c['id'] == bob_project['environmentID'])
    # Bob's known-host file contains independently verified keys for both projects.
    # It is public trust material, not Bob's authentication credential.
    denied = execute(['ssh', '-F', '/dev/null', '-o', 'ForwardAgent=no', '-o', 'ClearAllForwardings=yes',
                      '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', '-o', 'IdentitiesOnly=yes',
                      '-o', 'StrictHostKeyChecking=yes', '-o', 'UserKnownHostsFile=' + str(known),
                      '-i', str(root / 'alice/development'), 'u08-alice-8417@' + bob_ip, 'true'])
    assert denied.returncode == 255 and b'Permission denied (publickey' in denied.stderr, 'Require actual cross-project authentication denial, not a routing/host-key failure'
    (output / 'cross-project-denial.json').write_text(json.dumps({'alice_to_bob_project': 'public-key authentication denied'}))
    print('Selected memberships passed direct SSH, interactive PTY, bidirectional SCP/SFTP and expected sudo boundaries; Alice cannot authenticate to the unjoined second project.')
    print('Shared tools, personal Git, nested workloads and persistence are separate checks.')


if __name__ == '__main__':
    try:
        main()
    except Exception as failure:
        print('Developer access incomplete; retained probe state; failure type: ' + type(failure).__name__, file=sys.stderr)
        sys.exit(1)

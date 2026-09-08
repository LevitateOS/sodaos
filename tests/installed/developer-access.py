#!/usr/bin/env python3
"""Opt-in Sodaspaces direct project-IP SSH/PTY/SCP/SFTP and sudo checks.

Consumes the native browser's public access result and an independently verified
public host key. Private authentication keys stay on this client. Retains only
new run-owned probe directories; no lifecycle actions or automatic cleanup.
Historical U08 fixed-name invocation remains in Git, not a second access scenario.
"""
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import uuid


def private_file(name, limit=65536):
    p = Path(name)
    st = p.lstat()
    assert p.is_absolute() and p.is_file() and not p.is_symlink()
    assert not st.st_mode & 0o077 and st.st_size <= limit
    return p.read_bytes()


def main():
    os.umask(0o077)
    assert len(sys.argv) == 2
    root = Path(sys.argv[1])
    assert root.is_absolute() and root.is_dir() and not root.is_symlink()
    assert not root.stat().st_mode & 0o077
    request = json.loads(private_file(root / 'target.json'))
    required = {'target', 'revision', 'project_id', 'subnet', 'browser_result', 'host_key_file', 'users'}
    assert set(request) in (required, required | {'ssh_config_file'})
    ssh_config = request.get('ssh_config_file', '/dev/null')
    if ssh_config != '/dev/null':
        private_file(ssh_config, 16384)  # Explicit trusted client transport, not a route claim.
    assert re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{0,252}', request['target'])
    assert os.environ.get('SODA_NATIVE_VALIDATE') == request['target']
    assert re.fullmatch(r'[0-9a-f]{40}', request['revision'])
    identifier = request['project_id']
    assert re.fullmatch(r'p[0-9a-f]{24}', identifier)
    subnet = ipaddress.IPv4Network(request['subnet'])
    assert subnet.is_private
    browser = json.loads(private_file(request['browser_result']))
    assert browser['target'] == request['target'] and browser['revision'] == request['revision']
    assert browser['outcome'] == 'passed-scoped-journey' and browser['access']['native_join_confirmed'] is True
    assert browser['access']['reservation_id'] == identifier
    public = private_file(request['host_key_file']).decode().strip()
    assert re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/]{68}(?: [^\r\n]*)?', public)
    public = ' '.join(public.split()[:2])
    assert len(request['users']) == len(browser['access']['users']) == 2
    assert len({u['id'] for u in request['users']}) == 2
    for user in request['users']:
        assert set(user) == {'id', 'login', 'key_file', 'administrator'}
        assert re.fullmatch(r'[1-9][0-9]{0,18}', user['id'])
        assert re.fullmatch(r'[a-z][a-z0-9_-]{0,30}', user['login']) and user['login'] != 'root'
        assert isinstance(user['administrator'], bool)
        private_file(user['key_file'], 16384)  # Validate the file; never transfer or print its contents.
    assert request['users'][0]['administrator'] and not request['users'][1]['administrator']

    output = root / ('sodaspaces-access-' + uuid.uuid4().hex)
    output.mkdir(mode=0o700)
    results = {'revision': request['revision'], 'target': request['target'], 'project': identifier,
               'client': platform.node(), 'client_arch': platform.machine(),
               'transport': 'direct project IP' if ssh_config == '/dev/null' else 'explicit SSH configuration; not direct-route proof',
               'script_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), 'users': []}

    def execute(args, data=None):
        return subprocess.run(args, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45)

    def checked(args, data=None):
        result = execute(args, data)
        if result.returncode:
            raise RuntimeError('Selected access operation failed (exit %d)' % result.returncode)
        return result.stdout

    try:
        fingerprint = checked(['ssh-keygen', '-lf', request['host_key_file']]).decode().split()[1]
        assert re.fullmatch(r'SHA256:[A-Za-z0-9+/]{43}', fingerprint)
        payload = (output.name + '\n').encode() * 8192
        source = output / 'payload'
        source.write_bytes(payload)
        endpoints = []
        for user, connection in zip(request['users'], browser['access']['users']):
            login = user['login']
            assert connection['id'] == user['id'] and connection['login'] == login
            native = connection['connection']
            assert native['environment']['id'] == identifier and native['environment']['running'] is True
            ip = str(ipaddress.IPv4Address(native['environment']['ip']))
            assert ipaddress.IPv4Address(ip) in subnet and ip not in (str(subnet.network_address), str(subnet.broadcast_address))
            assert native['host_key'].strip() == public and native['fingerprint'] == fingerprint
            endpoints.append(ip)
            known = output / (login + '-known-hosts')
            known.write_text(ip + ' ' + public + '\n')
            options = ['-F', ssh_config, '-o', 'ControlMaster=no', '-o', 'ControlPath=none',
                       '-o', 'IdentityAgent=none', '-o', 'PreferredAuthentications=publickey',
                       '-o', 'ForwardAgent=no', '-o', 'ClearAllForwardings=yes',
                       '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', '-o', 'IdentitiesOnly=yes',
                       '-o', 'StrictHostKeyChecking=yes', '-o', 'UserKnownHostsFile=' + str(known), '-i', user['key_file']]
            ssh = ['ssh', *options]
            target = login + '@' + ip
            identity = checked(ssh + [target, 'id -un; id -u; printf "%s\\n" "$HOME"']).decode().splitlines()
            assert identity[0] == login and int(identity[1]) != 0 and identity[2] == '/home/' + login
            mapping = checked(ssh + [target, '/usr/bin/head -n 1 /proc/self/uid_map']).decode().split()
            assert int(mapping[0]) == 0 and int(mapping[1]) != 0, 'Project root must not map to host root'
            terminal = checked(ssh + ['-tt', target], b'test -t 0 && printf "\\nSODA-PTY:%s\\n" "$(id -un)"\nexit\n')
            assert ('SODA-PTY:' + login).encode() in terminal.replace(b'\r', b'').splitlines()
            privilege = execute(ssh + [target, 'sudo -n /usr/bin/id -u'])
            if user['administrator']:
                assert privilege.returncode == 0 and privilege.stdout.strip() == b'0'
            else:
                assert privilege.returncode == 1 and any(s in privilege.stderr for s in [b'password is required', b'not allowed', b'not in the sudoers'])
            destination = '/home/' + login + '/' + output.name
            checked(ssh + [target, 'umask 077; mkdir ' + destination])
            received = output / (login + '-scp')
            checked(['scp', *options, str(source), target + ':' + destination + '/scp'])
            checked(['scp', *options, target + ':' + destination + '/scp', str(received)])
            assert received.read_bytes() == payload
            sftp_copy = output / (login + '-sftp')

            def quote(p):
                s = str(p)
                assert not any(c in s for c in '\r\n')
                return '"' + s.replace('\\', '\\\\').replace('"', '\\"') + '"'

            batch = 'put ' + quote(source) + ' ' + destination + '/sftp\nget ' + destination + '/sftp ' + quote(sftp_copy) + '\n'
            checked(['sftp', *options, '-b', '-', target], batch.encode())
            assert sftp_copy.read_bytes() == payload
            results['users'].append({'id': user['id'], 'login': login, 'ip': ip, 'project_administrator': user['administrator'],
                                     'direct_ssh': ssh_config == '/dev/null', 'ssh_authentication': True, 'interactive_pty': True, 'scp_roundtrip': True, 'sftp_roundtrip': True,
                                     'probe_directory': destination})
        assert endpoints[0] == endpoints[1]
        # Same host key and reachable endpoint: require public-key denial, not a transport failure.
        first, second = request['users']
        denied = execute(['ssh', '-F', ssh_config, '-o', 'ControlMaster=no', '-o', 'ControlPath=none',
                          '-o', 'IdentityAgent=none', '-o', 'PreferredAuthentications=publickey',
                          '-o', 'ForwardAgent=no', '-o', 'ClearAllForwardings=yes',
                          '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', '-o', 'IdentitiesOnly=yes',
                          '-o', 'StrictHostKeyChecking=yes', '-o', 'UserKnownHostsFile=' + str(known),
                          '-i', first['key_file'], second['login'] + '@' + endpoints[1], 'true'])
        assert denied.returncode == 255 and b'Permission denied (publickey' in denied.stderr
        results['cross_user_key'] = 'public-key authentication denied'
        results['outcome'] = 'passed-scoped-access'
    except Exception:
        results['outcome'] = 'failed; retained partial access state'
        raise
    finally:
        (output / 'results.json').write_text(json.dumps(results, indent=2) + '\n')
    print('Declared memberships passed native SSH, PTY, SCP/SFTP, owner sudo and cross-user key denial.')
    print('Client placement/routing is recorded separately; not laptop, lifecycle or workload acceptance.')


if __name__ == '__main__':
    try:
        main()
    except Exception as failure:
        print('Developer access incomplete; retained probe state; failure type: ' + type(failure).__name__, file=sys.stderr)
        sys.exit(1)

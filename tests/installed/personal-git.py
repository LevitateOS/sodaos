#!/usr/bin/env python3
"""U08 personal Git on the exact retained fixtures: prepare, then exercise.

Prepare generates encrypted Git keys inside each user's project home and loads
local (not forwarded) SSH agents. Only public keys are returned. Register those
through dashboard-react.mjs / SODA_U08_GIT_DIR before exercise. The operator must
first establish the explicitly verified loopback Git transport and host key.
Existing keys/checkouts are never replaced or automatically retried.
"""
import json
import ipaddress
import re
import secrets
import os
from pathlib import Path
import subprocess
import sys
import textwrap
import urllib.parse


def main():
    assert os.environ.get('SODA_NATIVE_VALIDATE') == 'soda-test'
    assert len(sys.argv) == 3 and sys.argv[1] in ('prepare', 'exercise', 'unlock')
    phase = sys.argv[1]
    fixture = Path(sys.argv[2]); assert fixture.is_absolute()
    st = fixture.lstat(); assert fixture.is_dir() and not fixture.is_symlink() and st.st_uid == os.getuid() and not st.st_mode & 0o077
    target = json.loads((fixture / 'target.json').read_text()) if (fixture / 'target.json').exists() else {'ip': '10.89.0.2', 'repository': 'shared-alice'}
    ip = target['ip']; repository = target['repository']
    assert ipaddress.ip_address(ip) in ipaddress.ip_network('10.89.0.0/24')
    assert re.fullmatch(r'[a-z0-9][a-z0-9-]{0,99}', repository)
    directory = fixture / 'personal-git'
    if phase == 'prepare': directory.mkdir(mode=0o700)

    def checked(command, data=None):
        result = subprocess.run(command, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=90)
        if result.returncode: raise RuntimeError('Native personal Git operation failed; no retry (exit %d)' % result.returncode)
        return result.stdout

    outcomes = []
    for who in ('alice', 'bob'):
        login = 'u08-' + who + '-8417'
        ssh = ['ssh', '-F', '/dev/null', '-o', 'ForwardAgent=no', '-o', 'ClearAllForwardings=yes',
               '-o', 'BatchMode=yes', '-o', 'IdentitiesOnly=yes', '-o', 'StrictHostKeyChecking=yes',
               '-o', 'UserKnownHostsFile=' + str(fixture / (login + '-known-hosts')),
               '-i', str(fixture / who / 'development'), login + '@' + ip]
        passfile = directory / (login + '-passphrase')
        if phase == 'prepare':
            with open(passfile, 'x', opener=lambda p, flags: os.open(p, flags, 0o600)) as stream:
                stream.write(secrets.token_urlsafe(32))
        if phase in ('prepare', 'unlock'):
            assert passfile.stat().st_uid == os.getuid() and not passfile.stat().st_mode & 0o077
            passphrase = passfile.read_text(); assert re.fullmatch(r'[A-Za-z0-9_-]{43}', passphrase)
            program = textwrap.dedent('''\
                import os, pathlib, re, secrets, subprocess
                base = pathlib.Path.home()/'.ssh/u08-personal-git'
                base.parent.mkdir(mode=0o700,exist_ok=True)
                PREPARE_DIRECTORY
                os.umask(0o077)
                password = base/'temporary-passphrase'
                ask = base/'temporary-askpass'
                password.write_text(PASSPHRASE_INPUT)
                ask.write_text('#!/bin/sh\\nexec /usr/bin/head -c 128 '+str(password)+'\\n')
                ask.chmod(0o700)
                env = {**os.environ,'SSH_ASKPASS':str(ask),'SSH_ASKPASS_REQUIRE':'force','DISPLAY':'soda-u08'}
                def run(args):
                    r=subprocess.run(args,env=env,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
                    if r.returncode: raise RuntimeError('Git key/agent operation failed')
                    return r.stdout
                PREPARE_KEY
                # A socket without a live agent is a retained run-owned transient.
                if (base/'agent').exists():
                    probe=subprocess.run(['ssh-add','-l'],env={**env,'SSH_AUTH_SOCK':str(base/'agent')},stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
                    if probe.returncode != 2: raise RuntimeError('Agent already live; no duplicate start')
                    (base/'agent').unlink()
                agent=run(['ssh-agent','-a',str(base/'agent'),'-s']).decode()
                pid=re.search(r'SSH_AGENT_PID=(\\d+)',agent).group(1)
                (base/'agent.pid').write_text(pid+'\\n')
                env['SSH_AUTH_SOCK']=str(base/'agent')
                run(['ssh-add',str(base/'identity')])
                # Only these exact run-owned temporary secret inputs are removed.
                # The encrypted key and live agent remain in this user's home.
                password.unlink(); ask.unlink()
                (base/'git-ssh').write_text('#!/bin/sh\\nexport SSH_AUTH_SOCK='+str(base/'agent')+'\\nexec /usr/bin/ssh -F /dev/null -o BatchMode=yes -o ForwardAgent=no -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile='+str(base/'known_hosts')+' -i '+str(base/'identity')+' "$@"\\n')
                (base/'git-ssh').chmod(0o700)
                print((base/'identity.pub').read_text().strip())
                ''')
            program = program.replace('PASSPHRASE_INPUT', repr(passphrase)).replace('PREPARE_DIRECTORY', 'base.mkdir(mode=0o700)' if phase == 'prepare' else 'assert base.is_dir()').replace('PREPARE_KEY', "run(['ssh-keygen','-q','-t','ed25519','-f',str(base/'identity'),'-C','U08 personal project Git'])" if phase == 'prepare' else "assert (base/'identity').is_file()")
            public = checked(ssh + ['python3 -'], program.encode())
            del passphrase, program
            assert public.startswith(b'ssh-ed25519 ') and public.count(b'\n') == 1
            if phase == 'prepare': (directory / (login + '.pub')).write_bytes(public)
            else: assert public == (directory / (login + '.pub')).read_bytes()
            print(login, 'encrypted project-local Git key/agent ready; only public part exported', flush=True)
            continue
        repo = json.loads((directory / (login + '-repository.json')).read_text())
        url = urllib.parse.urlsplit(repo['ssh_url'])
        # Use the actual returned advertisement, never guessed predecessor ports.
        assert url.scheme == 'ssh' and url.hostname == '127.0.0.1' and url.port == 2222
        assert url.username == 'git' and not url.password and not url.query and not url.fragment
        assert url.path == '/u08-alice-8417/' + repository + '.git'
        clone = repo['ssh_url']
        branch = 'u08-native-' + who
        command = f'''set -eu
export GIT_SSH="$HOME/.ssh/u08-personal-git/git-ssh"
test ! -e "$HOME/u08-personal-checkout"
git clone '{clone}' "$HOME/u08-personal-checkout"
cd "$HOME/u08-personal-checkout"
git config user.name '{login}'
git config user.email '{login}@example.test'
git switch -c {branch}
printf '%s\\n' '{login} personal native Git proof' > {who}-native-git.txt
git add {who}-native-git.txt
git commit -m 'U08 personal Git proof for {who}'
git push --set-upstream origin {branch}
local=$(git rev-parse HEAD)
remote=$(git ls-remote origin refs/heads/{branch} | cut -f1)
test "$local" = "$remote"
printf 'U08-COMMIT:%s\\n' "$local"
'''
        result = checked(ssh + ['sh -se'], command.encode()).decode()
        commit = next(line[11:] for line in result.splitlines() if line.startswith('U08-COMMIT:'))
        assert len(commit) == 40 and all(c in '0123456789abcdef' for c in commit)
        outcomes.append({'login': login, 'branch': branch, 'commit': commit, 'repository_id': repo['id']})
        (directory / 'results.json').write_text(json.dumps(outcomes, indent=2))
        print(login, 'personal clone/commit/push/native ref readback verified', flush=True)
    if phase == 'exercise':
        # Bob independently fetches Alice's branch into his own ordinary checkout.
        checked(ssh + ['sh -se'], b'''set -eu
export GIT_SSH="$HOME/.ssh/u08-personal-git/git-ssh"
cd "$HOME/u08-personal-checkout"
git fetch origin u08-native-alice
git show FETCH_HEAD:alice-native-git.txt | grep -qx 'u08-alice-8417 personal native Git proof'
''')
        print('Independent collaborator fetch/readback verified; no merge or environment promotion performed.')


if __name__ == '__main__':
    try:
        main()
    except Exception as failure:
        print('Personal Git incomplete; retained state; failure type: ' + type(failure).__name__, file=sys.stderr)
        sys.exit(1)

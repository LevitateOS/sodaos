"""One explicitly requested native phase; embedded by the outside Go tool.
Adapted from soda-os bc1d3e0 soda-release-executor's exact-source/fresh-run idea.
No release account, login shell, publication, install, VM or product phase.
"""
import json
import os
from pathlib import Path
import platform
import re
import stat
import subprocess
import sys


def main():
    raw = sys.stdin.buffer.read(16385)
    if len(raw) > 16384:
        raise ValueError('request too large')
    x = json.loads(raw)
    if set(x) != {'Revision', 'Architecture', 'Target', 'Work', 'Phase'}:
        raise ValueError('unexpected request fields')
    if not all(isinstance(v, str) for v in x.values()):
        raise ValueError('request values must be strings')
    if not re.fullmatch('[0-9a-f]{40}', x['Revision']):
        raise ValueError('full revision required')
    if x['Architecture'] not in ('x86_64', 'aarch64') or platform.system() != 'Linux' or platform.machine() != x['Architecture']:
        raise ValueError('matching-native Linux required')
    if platform.node() != x['Target']:
        raise ValueError('actual native hostname does not match target')
    if x['Phase'] not in ('prepare', 'build', 'check', 'bundle'):
        raise ValueError('unknown phase; no install/VM/provider/release phases')
    work = Path(x['Work'])
    if not work.is_absolute() or work.parent.resolve() != work.parent or not work.parent.is_dir():
        raise ValueError('absolute fresh work path with existing real parent required')
    checkout = work / 'source'
    receipt = {k: v for k, v in x.items() if k != 'Phase'}
    env = dict(os.environ, GIT_TERMINAL_PROMPT='0', GOTOOLCHAIN='local')
    if x['Phase'] == 'prepare':
        work.mkdir(mode=0o700)  # exclusive; never adopt an old checkout
        with (work / 'request.json').open('x') as f:
            json.dump(receipt, f)
        subprocess.run(['git', '-c', 'core.hooksPath=/dev/null', 'clone', '--no-checkout', '--', 'https://github.com/LevitateOS/sodaos.git', str(checkout)], check=True, env=env)
        subprocess.run(['git', '-c', 'core.hooksPath=/dev/null', 'checkout', '--detach', x['Revision']], cwd=checkout, check=True, env=env)
    else:
        st = work.lstat()
        if not stat.S_ISDIR(st.st_mode) or st.st_uid != os.getuid() or st.st_mode & 0o077:
            raise ValueError('private owned run directory required')
        if json.loads((work / 'request.json').read_text()) != receipt:
            raise ValueError('phase does not belong to this source/target/run')
        if not (work / 'prepare.completed').is_file():
            raise ValueError('prepare did not complete')
    if checkout.is_symlink() or not checkout.is_dir():
        raise ValueError('real checkout required')
    head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=checkout, env=env).decode().strip()
    dirty = subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=normal'], cwd=checkout, env=env)
    if head != x['Revision'] or dirty:
        raise ValueError('checkout revision/content changed; use a fresh run')
    phase = x['Phase']
    prerequisite = {'check': 'build.completed', 'bundle': 'check.completed'}.get(phase)
    if prerequisite and not (work / prerequisite).is_file():
        raise ValueError('required earlier phase did not complete')
    with (work / (phase + '.started')).open('x') as f:
        json.dump(receipt, f)
    if phase == 'build':
        command = ['bash', 'scripts/build-native.sh', x['Architecture']]
    elif phase == 'check':
        command = ['bash', 'scripts/check-native.sh', x['Architecture']]
    elif phase == 'bundle':
        (work / 'bundle').mkdir(mode=0o700)
        stage = checkout / '.artifacts' / 'native' / x['Architecture']
        command = [str(stage / 'tools' / 'soda-artifacts'), 'bundle', '--source', str(stage), '--out', str(work / 'bundle' / x['Architecture']), '--arch', x['Architecture'], '--revision', x['Revision']]
    else:
        command = None
    if command:
        subprocess.run(command, cwd=checkout, check=True, env=env)
    if subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=checkout, env=env).decode().strip() != head or subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=normal'], cwd=checkout, env=env):
        raise ValueError('source changed during phase')
    with (work / (phase + '.completed')).open('x') as f:
        json.dump(receipt, f)
    print('Native phase completed; no later phase was requested or implied.')


if __name__ == '__main__':
    os.umask(0o077)
    try:
        main()
    except (ValueError, OSError, subprocess.SubprocessError) as exc:
        # No request body, commands, environment or private file contents.
        print('Native phase failed (' + type(exc).__name__ + '); run retained.', file=sys.stderr)
        sys.exit(1)

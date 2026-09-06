#!/usr/bin/env python3
"""Core U08 all-project before/after snapshot and comparison, no lifecycle action.

Usage: lifecycle-state.py FIXTURE_DIR snapshot LABEL
       lifecycle-state.py FIXTURE_DIR compare BEFORE AFTER
The private target.json selects the fresh project; old fixture identities are
retained explicitly. No secret-bearing inspection or credential contents.
"""
import json
import ipaddress
import os
from pathlib import Path
import re
import subprocess
import sys


def main():
    assert os.environ.get('SODA_NATIVE_VALIDATE') == 'soda-test'
    root = Path(sys.argv[1]); assert root.is_absolute()
    st = root.lstat(); assert root.is_dir() and not root.is_symlink() and st.st_uid == os.getuid() and not st.st_mode & 0o077
    mode = sys.argv[2]; labels = sys.argv[3:]
    assert all(re.fullmatch(r'[a-z][a-z0-9-]{0,50}', label) for label in labels)
    if mode == 'compare':
        assert len(labels) == 2
        a, b = [json.loads((root / (label + '.json')).read_text()) for label in labels]
        assert a['stable'] == b['stable'], 'Stable state differs; inspect private snapshots, do not repair by reset'
        print('All declared project/account/Git/tool/workload/database state matches; boot identity compared separately.')
        return
    assert mode == 'snapshot' and len(labels) == 1
    target = json.loads((root / 'target.json').read_text())
    repo = Path(__file__).resolve().parents[2]
    vm = str(repo / 'scripts/test-vm.sh')
    old = json.loads((repo / '.artifacts/test-vm/u08-8417a90/observed-bindings.json').read_text())
    entries = [(b['environmentID'], b['login'], True if b['login'] == 'u08-alice-8417' else False) for b in old]
    entries.append((target['environment_id'], 'u08-alice-8417', True))
    assert len(entries) == 3 and len({x[0] for x in entries}) == 3
    def run(args, data=None):
        result = subprocess.run(args, input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=180)
        if result.returncode:
            # The project snapshot deliberately emits only its safe operation/type.
            detail = result.stderr.decode().strip() if result.stderr.startswith(b'Project snapshot failed:') else 'operator observation failed'
            raise RuntimeError('Snapshot command failed; no empty substitution: ' + detail)
        if len(result.stdout) > 8 * 1024 * 1024: raise RuntimeError('Snapshot exceeded bound')
        return result.stdout
    host = '''import json,sqlite3,pathlib
p=pathlib.Path('/etc/soda/dashboard.json'); config=json.loads(p.read_text())
with sqlite3.connect('file:'+config['database']+'?mode=ro',uri=True) as db:
 assert db.execute('pragma integrity_check').fetchall()==[('ok',)]
 data={table:sorted(db.execute('select * from '+table).fetchall(),key=repr) for table in ['users','keys','projects','memberships']}
 print(json.dumps(data,sort_keys=True))
'''
    stable = {'soda': json.loads(run([vm, 'ssh', 'python3 -'], host.encode())), 'projects': {}}
    program = (repo / 'tests/installed/project-state.py').read_bytes()
    for identifier, login, workloads in entries:
        assert re.fullmatch(r'p[0-9a-f]{24}', identifier)
        assert login in ['u08-alice-8417', 'u08-bob-8417']
        # Operator transport observes exact trusted containers, not a substitute
        # for the separately executed real developer-client access tests.
        identity = run([vm, 'ssh', 'podman inspect --format "{{.Id}} {{.Image}} {{.Name}}" soda-' + identifier]).decode().strip()
        network = json.loads(run([vm, 'ssh', 'podman inspect --format "{{json .NetworkSettings.Networks}}" soda-' + identifier]))
        ip = network['soda-projects']['IPAddress']
        assert ipaddress.ip_address(ip) in ipaddress.ip_network('10.89.0.0/24')
        state = json.loads(run([vm, 'ssh', 'podman exec -i --env SODA_PROJECT_IP=' + ip + ' --env SODA_EXPECT_WORKLOADS=' + ('1' if workloads else '0') + ' soda-' + identifier + ' python3 -'], program))
        assert state['people'] and state['files']
        stable['projects'][identifier] = {'identity': identity, 'state': state}
    boot = run([vm, 'ssh', 'head -c 64 /proc/sys/kernel/random/boot_id']).decode().strip()
    assert re.fullmatch(r'[a-f0-9-]{36}', boot)
    with open(root / (labels[0] + '.json'), 'x', opener=lambda p, flags: os.open(p, flags, 0o600)) as file:
        json.dump({'stable': stable, 'boot_id': boot}, file, sort_keys=True)
    print('Complete bounded snapshot captured for three projects and Soda associations; no secret/private-key contents exported.')


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        detail = str(error) if isinstance(error, (RuntimeError, AssertionError)) else ''
        print('Lifecycle snapshot/comparison incomplete: ' + type(error).__name__ + ' ' + detail, file=sys.stderr)
        sys.exit(1)

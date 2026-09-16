"""Exercise the actual read-only RPM preflights without running an installer."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]

FAKE_DNF = """#!/usr/bin/env python3
import json
import os
import sys

db = json.load(open(os.environ['FAKE_DNF_DB']))
fail = {op for op in os.environ.get('FAKE_DNF_FAIL', '').split(',') if op}
args = sys.argv[1:]
kinds = ('--available', '--whatprovides', '--requires', '--whatrequires', '--whatrecommends', '--whatsuggests')
kind = next((a for a in kinds if a in args), None)
if kind is None:
    sys.exit(2)
if kind in fail:
    sys.stderr.write(f'simulated repo failure for {kind}\\n')
    sys.exit(1)
tables = {'--whatprovides': 'provides', '--requires': 'requires'}
if kind == '--available':
    lines = db['available']
elif kind in tables:
    operand = args[args.index(kind) + 1]
    lines = db[tables[kind]].get(operand, [])
else:
    lines = []
if lines:
    sys.stdout.write('\\n'.join(lines) + '\\n')
"""

FAKE_CURL = """#!/bin/sh
while [ $# -gt 0 ]; do
    if [ "$1" = -o ]; then
        printf '[tailscale-stable]\\nname=fake\\nbaseurl=file:///dev/null\\nenabled=1\\ngpgcheck=0\\n' > "$2"
        exit 0
    fi
    shift
done
exit 1
"""

# Miniature of the cockpit-ws/selinux/policy-base drift: both layered pins are
# pruned upstream, the virtual capability's only downloadable provider is a
# different NEVRA than the pruned one installed in the base, and the bump's
# subtree replaces a pruned base package. A sound relock must refuse; the
# pre-fix script accepted the qa bump because it skipped the conditional (no
# exact-NEVRA match) and dropped pruned base packages from its world.
RELOCK_DB = {
    'available': [
        'qa-0:2.0-1.fc44.x86_64',
        'qs-0:2.0-1.fc44.x86_64',
        'lib-0:2.0-1.fc44.x86_64',
        'capprov-0:9.0-1.fc44.noarch',
    ],
    'provides': {
        'vitcap': ['capprov-0:9.0-1.fc44.noarch'],
        'qs = 0:2.0-1.fc44': ['qs-0:2.0-1.fc44.x86_64'],
        'lib >= 0:2.0': ['lib-0:2.0-1.fc44.x86_64'],
    },
    'requires': {
        'qa-0:2.0-1.fc44.x86_64': ['(qs = 0:2.0-1.fc44 if vitcap)'],
        'qs-0:2.0-1.fc44.x86_64': ['lib >= 0:2.0'],
    },
}

RELOCK_LOCK = {
    'Requested': ['qa', 'qs'],
    'Install': ['qa-0:1.0-1.fc44.x86_64', 'qs-0:1.0-1.fc44.x86_64'],
    'Inventory': [
        'capprov 0:1.0-1.fc44.noarch',
        'lib 0:1.0-1.fc44.x86_64',
        'qa 0:1.0-1.fc44.x86_64',
        'qs 0:1.0-1.fc44.x86_64',
    ],
}


class HostPackages(unittest.TestCase):
    def test_capability_providers_and_missing_dependencies(self):
        for source in ['scripts/install-native.sh', 'tests/installed/host.sh']:
            lines = [line for line in (ROOT / source).read_text().splitlines() if line.startswith('rpm -q ')]
            self.assertEqual(len(lines), 2)
            for missing, expected in [
                ('', 0),
                ('nodejs', 1),
                ('cockpit-ws', 1),
                ('libicu', 0),
                ('openssl-libs', 0),
                ('krb5-libs', 0),
                ('zlib', 0),
            ]:
                with self.subTest(source=source, missing=missing), tempfile.TemporaryDirectory() as name:
                    fake = Path(name) / 'rpm'
                    fake.write_text('''#!/bin/sh
[ "$1" = -q ] || exit 99
shift
provides=no
if [ "$1" = --whatprovides ]; then provides=yes; shift; fi
for package do
    [ "$package" != "$MISSING" ] || exit 1
    case "$package" in
        nodejs) [ "$provides" = yes ] || exit 1 ;;
    esac
done
''')
                    fake.chmod(0o755)
                    result = subprocess.run(
                        ['/bin/sh', '-ec', '\n'.join(lines)],
                        env={**os.environ, 'PATH': name, 'MISSING': missing},
                        capture_output=True,
                        timeout=5,
                    )
                    self.assertEqual(result.returncode, expected, result.stderr)


class RelockScript(unittest.TestCase):
    def run_relock(self, directory, fail_ops=''):
        lock = Path(directory) / 'lock.json'
        lock.write_text(json.dumps(RELOCK_LOCK))
        for name, body in (('dnf', FAKE_DNF), ('curl', FAKE_CURL)):
            fake = Path(directory) / name
            fake.write_text(body)
            fake.chmod(0o755)
        (Path(directory) / 'db.json').write_text(json.dumps(RELOCK_DB))
        return lock, subprocess.run(
            ['/bin/bash', str(ROOT / 'scripts/relock-host-packages.sh'), '--check', f'--lock={lock}'],
            env={
                **os.environ,
                'PATH': f'{directory}:{os.environ["PATH"]}',
                'FAKE_DNF_DB': str(Path(directory) / 'db.json'),
                'FAKE_DNF_FAIL': fail_ops,
            },
            cwd=ROOT,
            capture_output=True,
            text=True,
            timeout=120,
        )

    def test_pruned_conditional_and_base_refuse_relock(self):
        with tempfile.TemporaryDirectory() as directory:
            lock, result = self.run_relock(directory)
            # The drift report prints to stdout; the refusal exits via
            # sys.exit and lands on stderr, so assert against both streams.
            output = result.stdout + result.stderr
            self.assertEqual(result.returncode, 1, output)
            self.assertIn('drifted pins (2)', output)
            self.assertIn('relock refused', output)
            # The pruned base package stays in the resolution world, so the
            # subtree's base replacement is reported, not silently added.
            self.assertIn('replacing lib-0:1.0-1.fc44.x86_64', output)
            # The joint move with the drifted sibling chains to its own block.
            self.assertIn('is itself blocked', output)
            self.assertEqual(lock.read_bytes(), json.dumps(RELOCK_LOCK).encode())

    def test_failed_repo_query_aborts_instead_of_accepting(self):
        with tempfile.TemporaryDirectory() as directory:
            _, result = self.run_relock(directory, fail_ops='--requires')
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn('repo query failed', result.stderr)


if __name__ == '__main__':
    unittest.main()

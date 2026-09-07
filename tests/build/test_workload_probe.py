"""Readiness polling must not replay native workload mutations."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class WorkloadProbe(unittest.TestCase):
    def invoke(self, mode, fail_http=False):
        with tempfile.TemporaryDirectory() as name:
            directory = Path(name)
            log = directory / 'calls.jsonl'
            fake = directory / 'command'
            fake.write_text('#!' + sys.executable + '\n' + '''import json,os,pathlib,sys
p=pathlib.Path(os.environ['CALL_LOG']);calls=[json.loads(x) for x in p.read_text().splitlines()] if p.exists() else []
call=[pathlib.Path(sys.argv[0]).name,*sys.argv[1:]]
with p.open('a') as f:f.write(json.dumps(call)+'\\n')
if call[0]=='curl':
 sys.exit(7 if os.environ['FAIL_HTTP']=='1' or not any(x[0]=='curl' for x in calls) else 0)
if 'pg_isready' in call:sys.exit(0 if any('pg_isready' in x for x in calls) else 1)
if call[0]=='sleep' or call[:3] in [['podman','compose','ps'],['podman','compose','up'],['podman','secret','inspect']] or 'psql' in call:sys.exit(0)
sys.exit(99)
''')
            fake.chmod(0o755)
            for command in ['podman', 'curl', 'sleep']:
                (directory / command).symlink_to(fake)
            result = subprocess.run(['/bin/sh', str(ROOT / 'tests/installed/workloads.sh'), mode],
                                    env={**os.environ, 'PATH':str(directory), 'CALL_LOG':str(log),
                                         'SODA_NATIVE_VALIDATE':'soda-test', 'FAIL_HTTP':'1' if fail_http else '0'},
                                    capture_output=True, timeout=15)
            calls = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
            return result, calls

    def test_start_once_then_readiness(self):
        result, calls = self.invoke('start')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(sum(c[:3] == ['podman','compose','up'] for c in calls), 1)
        self.assertEqual(sum(c[0] == 'curl' for c in calls), 2)
        self.assertEqual(sum('pg_isready' in c for c in calls), 2)
        self.assertEqual(sum('psql' in c for c in calls), 1)

    def test_check_never_starts_or_builds(self):
        result, calls = self.invoke('check')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(any('up' in c or 'build' in c or 'secret' in c for c in calls))

    def test_failed_readiness_is_bounded_and_not_success(self):
        result, calls = self.invoke('check', fail_http=True)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(sum(c[0] == 'curl' for c in calls), 30)
        self.assertFalse(any('up' in c or 'psql' in c for c in calls))

    def test_invalid_mode_never_invokes_native_commands(self):
        for mode in ['replace', '']:
            with self.subTest(mode=mode):
                result, calls = self.invoke(mode)
                self.assertEqual(result.returncode, 2)
                self.assertEqual(calls, [])


if __name__ == '__main__':
    unittest.main()

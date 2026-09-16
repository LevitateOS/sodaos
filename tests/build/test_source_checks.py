"""Source-command sequencing/fail-fast behavior without invoking compilers or services."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class SourceChecks(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        # Child getcwd() reports the physical path (e.g. macOS /private/var).
        self.root = Path(self.tmp.name).resolve()
        (self.root / 'scripts').mkdir()
        shutil.copyfile(ROOT / 'scripts/check-source.sh', self.root / 'scripts/check-source.sh')
        # check-source.sh runs the SQL locality gate as a real script, so the
        # fixture root needs a stub that logs and fails like the PATH tools.
        (self.root / 'scripts' / 'check-sql-locality.sh').write_text(
            '#!/bin/bash\n'
            'printf \'{"command": ["bash", "scripts/check-sql-locality.sh"], "cwd": "%s", '
            '"env": {"GOWORK": "%s", "GOFLAGS": "%s", "CGO_ENABLED": "%s", "GOTOOLCHAIN": "%s"}}\\n\' '
            '"$PWD" "$GOWORK" "$GOFLAGS" "$CGO_ENABLED" "$GOTOOLCHAIN" >> "$COMMAND_LOG"\n'
            'if [ "bash scripts/check-sql-locality.sh" = "${FAIL_COMMAND:-}" ]; then exit 7; fi\n'
        )
        self.tools = self.root / 'tools'
        self.tools.mkdir()
        self.log = self.root / 'commands.jsonl'
        for name in ('go', 'bun', 'python3'):
            tool = self.tools / name
            tool.write_text(
                f'#!{sys.executable}\n'
                + '''import json, os, pathlib, sys
command = [pathlib.Path(sys.argv[0]).name, *sys.argv[1:]]
with open(os.environ['COMMAND_LOG'], 'a') as log:
    log.write(json.dumps({'command': command, 'cwd': os.getcwd(), 'env': {key: os.environ.get(key) for key in ('GOWORK', 'GOFLAGS', 'CGO_ENABLED', 'GOTOOLCHAIN')}}) + '\\n')
if ' '.join(command) == os.environ.get('FAIL_COMMAND'): sys.exit(7)
'''
            )
            tool.chmod(0o700)
        self.expected = [
            ['go', 'mod', 'verify'],
            ['go', 'test', '-mod=readonly', './...'],
            ['bash', 'scripts/check-sql-locality.sh'],
            ['bun', 'run', 'typecheck'],
            ['bun', 'run', 'test'],
            ['python3', '-m', 'unittest', 'discover', '-s', 'tests/build'],
        ]

    def run_checks(self, fail=''):
        env = dict(
            os.environ,
            PATH=str(self.tools) + os.pathsep + os.environ['PATH'],
            COMMAND_LOG=str(self.log),
            FAIL_COMMAND=fail,
            GOTOOLCHAIN='auto',
        )
        return subprocess.run(
            ['/bin/bash', str(self.root / 'scripts/check-source.sh')],
            cwd=self.tools,
            env=env,
            capture_output=True,
            text=True,
            timeout=30,
        )

    def commands(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def test_source_commands_work_without_a_stage_or_git_checkout(self):
        result = self.run_checks()
        self.assertEqual(result.returncode, 0, result.stderr)
        commands = self.commands()
        self.assertEqual([entry['command'] for entry in commands], self.expected)
        for entry in commands:
            self.assertEqual(entry['cwd'], str(self.root))
            self.assertEqual(
                entry['env'], {'GOWORK': 'off', 'GOFLAGS': '-mod=readonly', 'CGO_ENABLED': '0', 'GOTOOLCHAIN': 'local'}
            )
        self.assertIn('Local source checks executed', result.stdout)

    def test_each_failure_stops_remaining_suites(self):
        for index, command in enumerate(self.expected):
            with self.subTest(command=command):
                self.log.write_text('')
                result = self.run_checks(' '.join(command))
                self.assertEqual(result.returncode, 7)
                self.assertEqual([entry['command'] for entry in self.commands()], self.expected[: index + 1])
                self.assertNotIn('Local source checks executed', result.stdout)

    def test_native_gate_still_surrounds_shared_source_checks(self):
        scripts = json.loads((ROOT / 'package.json').read_text())['scripts']
        self.assertEqual(scripts['check:source'], 'bash scripts/check-source.sh')
        native = (ROOT / 'scripts/check-native.sh').read_text()
        before, after = native.split('bun run check:source\n')
        self.assertIn('$(uname -s) == Linux && $(uname -m) == "$arch"', before)
        self.assertIn('Pinned native source-check tools required', before)
        self.assertIn('Check requires a clean exact-revision checkout', before)
        self.assertIn(
            'soda-build candidate artifacts directory required (payload.json, candidate.json, host.oci)', before
        )
        self.assertIn('verifier=$artifacts/tools/soda-artifacts', before)
        self.assertIn('"$verifier" verify --source "$artifacts" --arch "$arch" --revision "$revision"', before)
        self.assertIn('candidate architecture mismatch', before)
        self.assertIn('missing application archives: ', before)
        self.assertIn('python3 -m unittest discover -s tests/packaging', after)
        self.assertIn('$(git rev-parse HEAD) == "$revision"', after)
        self.assertIn('git status --porcelain --untracked-files=normal', after)

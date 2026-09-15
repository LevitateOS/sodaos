"""Scope of the Go complexity gate: tools/ is shipping code."""

import subprocess
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


def run_gate(*paths):
    return subprocess.run(
        ['/bin/bash', str(ROOT / 'scripts/check-complexity.sh'), *paths],
        cwd=ROOT,
        capture_output=True,
        text=True,
        timeout=120,
    )


class ComplexityScope(unittest.TestCase):
    def test_tools_files_pass_as_shipping(self):
        for path in ('tools/soda-candidate/main.go', 'tools/soda-avatars/main.go'):
            with self.subTest(path=path):
                result = run_gate(path)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                # 'below 10' proves the file reached gocyclo; excluded files
                # report 'no shipping Go files' instead.
                self.assertIn('below 10', result.stdout)

    def test_scripts_still_out_of_scope(self):
        result = run_gate('scripts/fake.go')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('no shipping Go files', result.stdout)

    def test_precommit_routes_staged_tools_files(self):
        hook = (ROOT / '.githooks/pre-commit').read_text()
        prodline = next(line for line in hook.splitlines() if line.startswith('prodgofiles='))
        self.assertIn('tools', prodline)

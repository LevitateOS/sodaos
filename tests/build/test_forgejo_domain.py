"""soda-forgejo-domain CLI surface of the Rust binary.

Deep behavior (app.ini resolution, marker mapping, verb sequencing) is
covered by rust/soda-forgejo-domain's own tests with injected paths and
fake system services.
"""

import os
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


class ForgejoDomainCLI(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        build = subprocess.run(
            ['cargo', 'build', '--offline', '-p', 'soda-forgejo-domain'],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        if build.returncode != 0:
            raise AssertionError(f'cannot build soda-forgejo-domain: {build.stderr[-2000:]}')
        cls.binary = str(ROOT / 'target/debug/soda-forgejo-domain')

    def run_domain(self, *argv):
        return subprocess.run([self.binary, *argv], capture_output=True, text=True)

    def test_help_reports_recovery_verbs(self):
        proc = self.run_domain('--help')
        self.assertEqual(proc.returncode, 0)
        self.assertIn('{stop,inhibit,status,lift,start}', proc.stdout)

    def test_missing_verb_rejected(self):
        proc = self.run_domain()
        self.assertEqual(proc.returncode, 2)
        self.assertIn('the following arguments are required: verb', proc.stderr)

    def test_invalid_verb_rejected(self):
        proc = self.run_domain('freeze')
        self.assertEqual(proc.returncode, 2)
        self.assertIn(
            "argument verb: invalid choice: 'freeze' (choose from stop, inhibit, status, lift, start)",
            proc.stderr,
        )

    @unittest.skipIf(os.geteuid() == 0, 'root passes the operator check')
    def test_nonroot_refused_without_effects(self):
        proc = self.run_domain('status')
        self.assertEqual(proc.returncode, 2)
        self.assertEqual(
            proc.stderr.strip().splitlines()[-1],
            'soda-forgejo-domain: error: native host operator/root required',
        )


if __name__ == '__main__':
    unittest.main()

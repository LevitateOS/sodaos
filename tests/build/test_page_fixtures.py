"""Exercise the existing Bun orchestrator with command doubles, not a browser/provider."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class PageFixtures(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root / 'scripts').mkdir()
        shutil.copyfile(ROOT / 'scripts/test-spaces-page.ts', self.root / 'scripts/test-spaces-page.ts')
        self.tools = self.root / 'tools'
        self.tools.mkdir()
        self.log = self.root / 'commands.jsonl'
        tool = self.tools / 'go'
        tool.write_text(f'#!{sys.executable}\n' + '''import json, os, sys
with open(os.environ['COMMAND_LOG'], 'a') as log:
    log.write(json.dumps({'args': sys.argv[1:], 'fixture': os.environ['SODA_NATIVE_CONNECTION_FIXTURE'], 'consumers': os.environ['SODA_PAGE_CONSUMERS']}) + '\\n')
sys.exit(int(os.environ.get('GO_STATUS', '0')))
''')
        tool.chmod(0o700)

    def run_pages(self, **changes):
        bun = shutil.which('bun')
        self.assertIsNotNone(bun, 'Bun is required for page-orchestrator checks')
        env = dict(os.environ, PATH=str(self.tools) + os.pathsep + os.environ['PATH'], COMMAND_LOG=str(self.log),
                   SODA_NATIVE_CONNECTION_FIXTURE='/must-not-reuse', SODA_PAGE_CONSUMERS='0')
        env.update(changes)
        return subprocess.run([bun, str(self.root / 'scripts/test-spaces-page.ts')], env=env,
                              capture_output=True, text=True, timeout=30)

    def commands(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def test_fresh_native_fixture_and_required_consumers(self):
        for _ in range(2):
            result = self.run_pages()
            self.assertEqual(result.returncode, 0, result.stderr)
        commands = self.commands()
        self.assertEqual(len(commands), 2)
        self.assertNotEqual(commands[0]['fixture'], commands[1]['fixture'])
        for command in commands:
            self.assertEqual(command['consumers'], '1')
            self.assertIn('-count=1', command['args'])
            self.assertIn('-mod=readonly', command['args'])
            self.assertEqual(command['args'][-1], '^TestNativeConnectionFixture$')
            self.assertTrue(Path(command['fixture']).resolve().is_relative_to((self.root / '.artifacts').resolve()))
            self.assertFalse(Path(command['fixture']).exists())

    def test_native_or_consumer_failure_is_not_hidden(self):
        result = self.run_pages(GO_STATUS='7')
        self.assertEqual(result.returncode, 7)
        self.assertEqual(len(self.commands()), 1)

    def test_missing_required_command_fails(self):
        (self.tools / 'go').unlink()
        result = self.run_pages(PATH=str(self.tools))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Native page fixtures retained at', result.stdout)

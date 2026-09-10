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
        for name in ('go', 'bun'):
            tool = self.tools / name
            tool.write_text(f'#!{sys.executable}\n' + '''import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
fixtures = {key: os.environ[key] for key in ('SODA_SPACES_PAGE_HTML', 'SODA_RUNNERS_PAGE_HTML', 'SODA_REPOSITORY_SETTINGS_HTML')}
with open(os.environ['COMMAND_LOG'], 'a') as log:
    log.write(json.dumps({'name': name, 'args': sys.argv[1:], 'fixtures': fixtures}) + '\\n')
if name == 'go':
    for key, file in fixtures.items():
        if key == os.environ.get('OMIT_FIXTURE'): continue
        with open(file, 'x') as output:
            if key != os.environ.get('EMPTY_FIXTURE'): output.write('{}')
    sys.exit(int(os.environ.get('GO_STATUS', '0')))
assert all(pathlib.Path(file).is_file() for file in fixtures.values())
sys.exit(int(os.environ.get('BROWSER_STATUS', '0')))
''')
            tool.chmod(0o700)

    def run_pages(self, **changes):
        bun = shutil.which('bun')
        self.assertIsNotNone(bun, 'Bun is required for page-orchestrator checks')
        env = dict(os.environ, PATH=str(self.tools) + os.pathsep + os.environ['PATH'], COMMAND_LOG=str(self.log),
                   SODA_SPACES_PAGE_HTML='/must-not-reuse/spaces.json',
                   SODA_RUNNERS_PAGE_HTML='/must-not-reuse/runners.json',
                   SODA_REPOSITORY_SETTINGS_HTML='/must-not-reuse/repository.json')
        env.update(changes)
        return subprocess.run([bun, str(self.root / 'scripts/test-spaces-page.ts')], env=env,
                              capture_output=True, text=True, timeout=30)

    def commands(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def test_fresh_required_producers_and_consumers(self):
        for _ in range(2):
            result = self.run_pages()
            self.assertEqual(result.returncode, 0, result.stderr)
        commands = self.commands()
        self.assertEqual([command['name'] for command in commands], ['go', 'bun', 'go', 'bun'])
        self.assertNotEqual(commands[0]['fixtures'], commands[2]['fixtures'])
        for go, browser in (commands[:2], commands[2:]):
            self.assertEqual(go['fixtures'], browser['fixtures'])
            self.assertIn('-count=1', go['args'])
            self.assertIn('-mod=readonly', go['args'])
            self.assertEqual(go['args'][-1], '^(TestSpacesHTMLSessionAuthorityAndBoundedException|TestRunnerOperatorGatesBeforeNativeAndDecode|TestRepositorySettingsUsesFreshStableIdentityAndSharedControls)$')
            self.assertEqual(browser['args'], ['test', '--timeout', '90000', 'tests/frontend/spaces-page.test.ts', 'tests/frontend/runners.test.ts', 'tests/frontend/repository-settings.test.ts'])
            for file in go['fixtures'].values():
                self.assertTrue(Path(file).is_relative_to(self.root / '.artifacts'))
                self.assertTrue(Path(file).is_file())

    def test_missing_or_empty_fixture_never_launches_browser(self):
        for setting in ('OMIT_FIXTURE', 'EMPTY_FIXTURE'):
            for key in ('SODA_SPACES_PAGE_HTML', 'SODA_RUNNERS_PAGE_HTML', 'SODA_REPOSITORY_SETTINGS_HTML'):
                with self.subTest(setting=setting, key=key):
                    result = self.run_pages(**{setting: key})
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('Go/browser fixtures retained at', result.stdout)
        self.assertTrue(all(command['name'] == 'go' for command in self.commands()))

    def test_producer_failure_stops_before_browser(self):
        result = self.run_pages(GO_STATUS='7')
        self.assertEqual(result.returncode, 7)
        self.assertEqual([command['name'] for command in self.commands()], ['go'])

    def test_browser_failure_is_not_hidden(self):
        result = self.run_pages(BROWSER_STATUS='9')
        self.assertEqual(result.returncode, 9)
        self.assertEqual([command['name'] for command in self.commands()], ['go', 'bun'])

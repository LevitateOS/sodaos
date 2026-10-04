"""Locked terminal distributions: the fetcher itself is Rust (PR30), so this
module pins the browser-build wiring and the shipping lock only."""

import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class TerminalAssets(unittest.TestCase):
    def test_browser_build_prepares_locked_renderer_before_emitted_modules(self):
        scripts = json.loads((ROOT / 'package.json').read_text())['scripts']
        prepare, emit = scripts['build:forgejo'].split(' && ', 1)
        self.assertEqual(
            prepare,
            'cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-terminal'
            ' -- --out .artifacts/browser-terminal/vendor',
        )
        self.assertEqual(emit, 'bun scripts/build-forgejo.ts')
        groups = ('frontend', 'forgejo')
        self.assertEqual(
            scripts['test'].split(' && '),
            [
                'bun run build:forgejo',
                *(f'bun run test:{group}:prepared' for group in groups),
            ],
        )
        for group in groups:
            self.assertEqual(scripts[f'test:{group}'], f'bun run build:forgejo && bun run test:{group}:prepared')
            self.assertNotIn('build:forgejo', scripts[f'test:{group}:prepared'])
        self.assertEqual(
            scripts['test:frontend:prepared'],
            'SODA_TAILNET_COMPONENT=1 bun test tests/frontend/*.test.ts',
        )
        self.assertEqual(
            scripts['test:forgejo:prepared'],
            'SODA_LIT_BROWSER=1 bun test --timeout 120000 tests/forgejo/*.test.ts tests/forgejo/presentation/*.test.ts',
        )
        self.assertTrue(scripts['test:lit'].startswith('bun run build:forgejo && SODA_LIT_BROWSER=1 bun test '))
        self.assertNotIn('tests/forgejo/settings-link.test.ts', scripts['test:lit'].split())

    def test_shipping_lock_has_only_exact_local_renderer_files(self):
        lock = json.loads((ROOT / 'appliance/terminal-assets.lock.json').read_text())
        self.assertEqual(
            [(i['package'], i['version']) for i in lock], [('@xterm/xterm', '6.0.0'), ('@xterm/addon-fit', '0.11.0')]
        )
        self.assertEqual(
            {f['file'] for i in lock for f in i['files']},
            {'xterm.mjs', 'xterm.css', 'addon-fit.mjs', 'xterm.LICENSE', 'fit.LICENSE'},
        )
        for item in lock:
            self.assertTrue(item['url'].startswith('https://registry.npmjs.org/'))
            for asset in item['files']:
                self.assertEqual(len(asset['sha256']), 64)

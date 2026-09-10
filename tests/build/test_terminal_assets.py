"""Locked distributions in temporary fixtures, no network or native stage."""
import base64
import hashlib
import io
import json
from pathlib import Path
import runpy
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


class TerminalAssets(unittest.TestCase):
    def test_exact_members_checksums_and_cached_bytes(self):
        module = runpy.run_path(str(ROOT / 'scripts/fetch-terminal.py'))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / 'appliance').mkdir()
            archive = io.BytesIO()
            with tarfile.open(fileobj=archive, mode='w:gz') as tar:
                member = tarfile.TarInfo('package/lib/xterm.mjs')
                member.size = 9
                tar.addfile(member, io.BytesIO(b'synthetic'))
            body = archive.getvalue()
            lock = [{'url': 'https://example.invalid/fixture', 'integrity': 'sha512-' + base64.b64encode(hashlib.sha512(body).digest()).decode(), 'files': [{'member': 'package/lib/xterm.mjs', 'file': 'xterm.mjs', 'sha256': hashlib.sha256(b'synthetic').hexdigest()}]}]
            (root / 'appliance/terminal-assets.lock.json').write_text(json.dumps(lock))
            fetch = module['fetch']
            with patch.dict(fetch.__globals__, ROOT=root), patch('urllib.request.urlopen', return_value=io.BytesIO(body)) as download:
                fetch(root / 'out')
                self.assertEqual((root / 'out/xterm.mjs').read_bytes(), b'synthetic')
                fetch(root / 'out')
                self.assertEqual(download.call_count, 1)
            (root / 'out/xterm.mjs').write_bytes(b'changed')
            with patch.dict(fetch.__globals__, ROOT=root), patch('urllib.request.urlopen', return_value=io.BytesIO(b'bad archive')):
                with self.assertRaisesRegex(ValueError, 'integrity'):
                    fetch(root / 'out')
            self.assertEqual((root / 'out/xterm.mjs').read_bytes(), b'changed')

    def test_browser_build_prepares_locked_renderer_before_emitted_modules(self):
        scripts = json.loads((ROOT / 'package.json').read_text())['scripts']
        prepare, emit = scripts['build:forgejo'].split(' && ', 1)
        self.assertEqual(prepare, 'python3 scripts/fetch-terminal.py --out .artifacts/browser-terminal/vendor')
        self.assertEqual(emit, 'bun scripts/build-forgejo.ts')
        for gate in ('test:frontend', 'test:layout', 'test:spaces-page', 'test:forgejo', 'test:lit'):
            self.assertTrue(scripts[gate].startswith('bun run build:forgejo && '), gate)

    def test_shipping_lock_has_only_exact_local_renderer_files(self):
        lock = json.loads((ROOT / 'appliance/terminal-assets.lock.json').read_text())
        self.assertEqual([(i['package'], i['version']) for i in lock], [('@xterm/xterm', '6.0.0'), ('@xterm/addon-fit', '0.11.0')])
        self.assertEqual({f['file'] for i in lock for f in i['files']}, {'xterm.mjs', 'xterm.css', 'addon-fit.mjs', 'xterm.LICENSE', 'fit.LICENSE'})
        for item in lock:
            self.assertTrue(item['url'].startswith('https://registry.npmjs.org/'))
            for asset in item['files']:
                self.assertEqual(len(asset['sha256']), 64)

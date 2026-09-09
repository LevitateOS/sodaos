"""Exact presentation payload and locked locale build tests; no network/install."""
import hashlib
import io
import json
from pathlib import Path
import re
import runpy
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


class ForgejoPayload(unittest.TestCase):
    def test_exact_sources_template_closure_and_notices(self):
        files = json.loads((ROOT / 'internal/nativebuild/forgejo-payload.json').read_text())
        templates = {'templates/' + p.relative_to(ROOT / 'appliance/forgejo/templates').as_posix()
                     for p in (ROOT / 'appliance/forgejo/templates').rglob('*.tmpl')}
        self.assertEqual(templates, {name for name in files if name.startswith('templates/')})
        for dest, source in files.items():
            self.assertNotIn('..', Path(dest).parts)
            self.assertFalse(Path(dest).is_absolute())
            self.assertNotIn('..', Path(source).parts)
            if source.startswith('@build/'):
                self.assertTrue(source.startswith(('@build/terminal-assets/', '@build/forgejo-locales/', '@build/forgejo-js/')))
                continue
            p = ROOT / source
            self.assertTrue(p.is_file(), source)
            self.assertFalse(p.is_symlink(), source)
            if p.suffix == '.tmpl':
                for name in re.findall(r'{{-?\s*template\s+"(custom/soda/[^"]+)"', p.read_text()):
                    self.assertIn('templates/' + name + '.tmpl', files)
        for font in ('barlow', 'fraunces', 'ibm-plex-mono'):
            self.assertIn(f'public/assets/soda/fonts/{font}/LICENSE', files)
        self.assertIn('public/assets/soda/forgejo/forgejo-LICENSE', files)
        self.assertEqual(files['public/assets/soda/forgejo/lit.js'], '@build/forgejo-js/lit.js')
        self.assertEqual(files['public/assets/soda/forgejo/lit.LICENSE'], 'appliance/licenses/lit-LICENSE')
        self.assertIn('options/locale/locale_en-US.ini', files)

    def test_spaces_page_assets_use_the_canonical_public_payload(self):
        files = json.loads((ROOT / 'internal/nativebuild/forgejo-payload.json').read_text())
        page = (ROOT / 'internal/web/templates/spaces.html').read_text()
        references = re.findall(r'(?:href|src|srcset)="(/assets/[^" ]+)"', page)
        self.assertGreaterEqual(len(references), 10)
        for reference in references:
            self.assertIn('public' + reference, files, reference)
        self.assertIn('public/assets/sodaspaces-page.js', files)
        self.assertIn('public/assets/sodaspaces-project.js', files)
        self.assertIn('public/assets/sodaspaces-drawer.js', files)
        self.assertNotIn('window.config', page)
        self.assertNotIn('webcomponents-loader', page)
        self.assertNotIn('htmx', page)
        self.assertNotIn('type="application/json"', page)

    def test_locale_fetch_is_locked_and_preserves_native_catalog(self):
        native = b'[common]\nname = Native\n[settings]\ntitle = Settings\n'
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            lock = root / 'lock.json'
            lock.write_text(json.dumps({'url': 'https://codeberg.org/forgejo/forgejo/raw/tag/v15.0.7/options/locale/locale_en-US.ini', 'sha256': hashlib.sha256(native).hexdigest()}))
            output = root / 'locale.ini'
            args = ['forgejo-locales.py', '--lock', str(lock), '--additions', str(ROOT / 'appliance/forgejo/i18n/en-US.ini'), '--out', str(output)]
            with patch('sys.argv', args), patch('urllib.request.urlopen', return_value=io.BytesIO(b'incorrect')):
                with self.assertRaises(SystemExit): runpy.run_path(str(ROOT / 'scripts/forgejo-locales.py'), run_name='__main__')
            self.assertFalse(output.exists())
            with patch('sys.argv', args), patch('urllib.request.urlopen', return_value=io.BytesIO(native)):
                runpy.run_path(str(ROOT / 'scripts/forgejo-locales.py'), run_name='__main__')
            self.assertTrue(output.read_bytes().startswith(native))
            self.assertIn('[soda]', output.read_text())

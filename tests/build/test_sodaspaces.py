"""Production staging/preflight in temporary filesystems; never host installation."""

import hashlib
import json
import os
from pathlib import Path
import runpy
import shutil
import stat
import struct
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
FILES = (
    'templates/custom/header.tmpl',
    'templates/custom/footer.tmpl',
    'public/assets/soda/forgejo/repository-actions.js',
    'public/assets/soda/forgejo/notification-preview.js',
)


class SodaspacesPackaging(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        build = subprocess.run(
            ['cargo', 'build', '-p', 'soda-test-vm'],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        if build.returncode != 0:
            raise AssertionError(f'cannot build soda-test-vm: {build.stderr[-2000:]}')
        cls.test_vm = str(ROOT / 'target/debug/soda-test-vm')
    def test_spaces_entry_is_packaged_by_the_extension(self):
        extension = json.loads((ROOT / 'appliance/soda-extension/extension.json').read_text())
        manifest = json.loads((ROOT / 'internal/release/build/forgejo-payload.json').read_text())
        spaces = next(page for page in extension['pages'] if page['id'] == 'spaces')
        self.assertEqual(spaces['entry'], 'soda-spaces-entry.js')
        self.assertTrue((ROOT / 'frontend/spaces/soda-spaces-entry.ts').is_file())
        self.assertNotIn('public/assets/sodaspaces.js', manifest)
        self.assertNotIn('public/assets/soda-spaces-entry.js', manifest)

    def test_vm_web_tunnel_uses_only_the_native_browser_origin(self):
        with tempfile.TemporaryDirectory() as tmp:
            ssh = Path(tmp) / 'ssh'
            ssh.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            ssh.chmod(0o755)
            result = subprocess.run(
                [self.test_vm, 'web-tunnel'],
                env={**os.environ, 'PATH': tmp + os.pathsep + os.environ['PATH']},
                capture_output=True,
                text=True,
                timeout=10,
            )
            self.assertEqual(result.returncode, 0)
            self.assertIn('Forgejo + Sodaspaces https://localhost:24444', result.stdout)
            self.assertIn('127.0.0.1:24444:127.0.0.1:24444', result.stdout)
            self.assertNotIn('24443', result.stdout)

    def test_access_probe_rejects_private_bad_request_before_native_commands(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            root.chmod(0o700)
            request = root / 'target.json'
            request.write_text('{"SYNTHETIC_PRIVATE_MARKER":true}')
            request.chmod(0o600)
            # The installed client probe is Go now; run it from the checkout
            # with the pinned toolchain check-source.sh already requires.
            result = subprocess.run(
                ['go', 'run', './tools/soda-installed-probes', 'developer-access', str(root)],
                cwd=ROOT,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn('Developer access incomplete', result.stderr)
            self.assertNotIn('SYNTHETIC_PRIVATE_MARKER', result.stdout + result.stderr)
            self.assertEqual(list(root.iterdir()), [request])

    def test_actual_stage_recipe_with_synthetic_build_inputs(self):
        # No generated artifact is placed in the production .artifacts/native tree.
        with tempfile.TemporaryDirectory() as tmp:
            checkout = Path(tmp).resolve()
            (checkout / 'scripts').mkdir()
            shutil.copyfile(ROOT / 'scripts/stage.py', checkout / 'scripts/stage.py')
            shutil.copytree(ROOT / 'assets', checkout / 'assets')
            for asset in (checkout / 'assets').rglob('*'):
                asset.chmod(0o700 if asset.is_dir() else 0o600)
            shutil.copytree(ROOT / 'appliance', checkout / 'appliance')
            shutil.copytree(ROOT / 'frontend', checkout / 'frontend')
            (checkout / 'internal/release/build').mkdir(parents=True)
            shutil.copyfile(
                ROOT / 'internal/release/build/forgejo-payload.json',
                checkout / 'internal/release/build/forgejo-payload.json',
            )
            for name in ('LICENSE', 'NOTICE'):
                shutil.copyfile(ROOT / name, checkout / name)
            build = checkout / '.artifacts/native/x86_64'
            (build / 'project-tools/bin').mkdir(parents=True)
            for name in ('muse', 'muse-native', 'soda-identity-compose'):
                (build / 'project-tools/bin' / name).write_text('synthetic; never executed')
            (build / 'forgejo-js').mkdir()
            for origin in json.loads((checkout / 'internal/release/build/forgejo-payload.json').read_text()).values():
                if origin.startswith('@build/forgejo-js/'):
                    (build / origin.removeprefix('@build/')).write_text('// synthetic compiled browser fixture\n')
            (build / 'forgejo-locales').mkdir()
            (build / 'forgejo-locales/locale_en-US.ini').write_text('synthetic full-catalog output; not native proof')
            # Synthetic bytes and lock only inside this temporary checkout.
            (build / 'terminal-assets').mkdir()
            lock_path = checkout / 'appliance/terminal-assets.lock.json'
            lock = json.loads(lock_path.read_text())
            for item in lock:
                for asset in item['files']:
                    data = ('synthetic asset ' + asset['file']).encode()
                    (build / 'terminal-assets' / asset['file']).write_bytes(data)
                    asset['sha256'] = hashlib.sha256(data).hexdigest()
            lock_path.write_text(json.dumps(lock))
            # Stale ignored build output must not resurrect the retired package.
            for page in ('tailscale', 'runners'):
                folder = checkout / 'cockpit/dist' / f'soda-{page}'
                folder.mkdir(parents=True)
                (folder / 'index.html').write_text('synthetic Cockpit package')
            host = checkout / 'host-context'
            vendor_root = host / 'rootfs'
            marker = vendor_root / 'usr/libexec/soda/soda-dashboard'
            marker.parent.mkdir(parents=True)
            marker.write_text('already compiled program; never executed')
            forgejo_context = checkout / 'forgejo-context'
            forgejo_context.mkdir(mode=0o700)
            host.chmod(0o700)
            args = [
                'stage.py',
                '--arch',
                'x86_64',
                '--host-context',
                str(host),
                '--forgejo-context',
                str(forgejo_context),
            ]

            def vendor_stage(argv=args):
                previous = os.umask(0o077)
                try:
                    with (
                        patch('sys.argv', argv),
                        patch('platform.system', return_value='Linux'),
                        patch('platform.machine', return_value='x86_64'),
                    ):
                        runpy.run_path(str(checkout / 'scripts/stage.py'), run_name='__main__')
                finally:
                    os.umask(previous)

            vendor_stage()
            # The single image layout owns every path; no writable tree is created.
            self.assertFalse((build / 'rootfs').exists())
            self.assertFalse((vendor_root / 'var').exists())
            self.assertFalse((vendor_root / 'usr/local').exists())
            self.assertFalse((vendor_root / 'etc/cockpit/users.override.json').exists())
            self.assertFalse((vendor_root / 'etc/cockpit/users.override.json').is_symlink())
            self.assertEqual(marker.read_text(), 'already compiled program; never executed')
            for directory in (host, forgejo_context):
                self.assertEqual(stat.S_IMODE(directory.stat().st_mode), 0o700, 'changed private context ancestor')
            presentation = forgejo_context / 'forgejo'
            manifest = json.loads((checkout / 'internal/release/build/forgejo-payload.json').read_text())

            def original(origin):
                return build / origin.removeprefix('@build/') if origin.startswith('@build/') else checkout / origin

            for path in [presentation, *presentation.rglob('*')]:
                self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o755 if path.is_dir() else 0o644)
            for name in FILES:
                self.assertEqual((presentation / name).read_bytes(), original(manifest[name]).read_bytes())
            for name, origin in manifest.items():
                staged = presentation / name
                self.assertEqual(staged.read_bytes(), original(origin).read_bytes())
                self.assertEqual(stat.S_IMODE(staged.stat().st_mode), 0o644)
                for parent in staged.parents:
                    if parent == presentation:
                        break
                    self.assertEqual(stat.S_IMODE(parent.stat().st_mode), 0o755)
            # The payload-served copies resolve the upstream theme three levels
            # up; the staged native-css copies must resolve it from their own
            # directory instead, or the import 404s in browsers.
            for theme in ('dark', 'light'):
                source = (checkout / f'assets/branding/forgejo/css/theme-soda-{theme}.css').read_text()
                self.assertIn(f'@import "../../../css/theme-forgejo-{theme}.css";', source)
                staged = presentation / f'public/assets/css/theme-soda-{theme}.css'
                self.assertIn(f'@import "theme-forgejo-{theme}.css";', staged.read_text())
                self.assertNotIn('css/theme-forgejo-', staged.read_text())
            brand = vendor_root / 'etc/cockpit/branding'
            self.assertEqual(
                (brand / 'soda-symbol-brutalist.svg').read_bytes(),
                (ROOT / 'assets/branding/source/soda-symbol-brutalist.svg').read_bytes(),
            )
            self.assertTrue((brand / 'fonts/barlow-condensed/LICENSE').is_file())
            self.assertEqual(
                (brand / 'apple-touch-icon.png').read_bytes(),
                (ROOT / 'assets/branding/forgejo/apple-touch-icon.png').read_bytes(),
            )
            icon = (brand / 'favicon.ico').read_bytes()
            self.assertEqual(struct.unpack_from('<HHH', icon), (0, 1, 2))
            for i, (size, name) in enumerate(((16, 'favicon-16.png'), (32, 'favicon.png'))):
                w, h, colors, reserved, planes, bits, length, offset = struct.unpack_from('<BBBBHHII', icon, 6 + 16 * i)
                self.assertEqual((w, h, colors, reserved, planes, bits), (size, size, 0, 0, 1, 32))
                self.assertEqual(icon[offset : offset + length], (ROOT / 'assets/branding/forgejo' / name).read_bytes())
            self.assertFalse((brand / 'login-background-light.svg').exists())
            self.assertFalse((brand / 'soda-symbol.svg').exists())
            for path in (vendor_root / 'etc/cockpit/branding').rglob('*'):
                self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o755 if path.is_dir() else 0o644)
            logo = vendor_root / 'usr/share/soda/fastfetch/sodaos.txt'
            self.assertEqual(logo.read_bytes(), (ROOT / 'assets/branding/terminal/sodaos.txt').read_bytes())
            self.assertIn('/usr/share/soda/fastfetch/', (vendor_root / 'etc/fastfetch/config.jsonc').read_text())
            self.assertEqual(stat.S_IMODE((vendor_root / 'etc/soda/forgejo.env').stat().st_mode), 0o600)
            for tool in ('muse', 'muse-native', 'soda-identity-compose'):
                staged = vendor_root / 'usr/share/soda/muse-tools' / tool
                self.assertEqual(stat.S_IMODE(staged.stat().st_mode), 0o755)
            for asset in (presentation / 'public/assets').rglob('*'):
                self.assertEqual(stat.S_IMODE(asset.stat().st_mode), 0o755 if asset.is_dir() else 0o644)
                self.assertNotIn('soda-runners', asset.parts)
                self.assertNotIn('soda-tailscale', asset.parts)

            def inventory(root):
                return {str(p.relative_to(root)): p.read_bytes() for p in root.rglob('*') if p.is_file()}

            before = inventory(vendor_root), inventory(presentation)
            with self.assertRaises(SystemExit):
                vendor_stage()
            self.assertEqual(before, (inventory(vendor_root), inventory(presentation)))
            with self.assertRaises(SystemExit):
                vendor_stage(['stage.py', '--arch', 'x86_64', '--host-context', str(host)])
            link = checkout / 'linked-forgejo-context'
            link.symlink_to(forgejo_context, target_is_directory=True)
            with self.assertRaises(SystemExit):
                vendor_stage(args[:-1] + [str(link)])
            # Copying public data must not normalize the canonical/private inputs.
            self.assertEqual(
                stat.S_IMODE((checkout / 'assets/branding/source/soda-symbol-brutalist.svg').stat().st_mode),
                0o600,
            )

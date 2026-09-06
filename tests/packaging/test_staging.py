"""Authored staging checks. Run later against an explicitly built native stage."""
import os
import unittest
from pathlib import Path


class NativeStage(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not os.environ.get('SODA_STAGE'):
            raise RuntimeError('SODA_STAGE must identify the actual native build rootfs')
        cls.root = Path(os.environ['SODA_STAGE'])

    def test_commands_and_extensions(self):
        for name in ['soda-dashboard', 'soda-host', 'soda-setup', 'soda-tailnet', 'soda-forgejo-tailnet', 'soda-runners', 'soda-runner-helper', 'soda-runner-launch']:
            p = self.root / 'usr/local/libexec/soda' / name
            self.assertTrue(p.is_file(), str(p))
            self.assertTrue(p.stat().st_mode & 0o111, str(p))
        for name in ['soda-runners', 'soda-tailscale']:
            folder = self.root / 'usr/local/share/cockpit' / name
            self.assertTrue((folder / 'index.html').is_file())
            self.assertTrue((folder / 'manifest.json').is_file())
        self.assertFalse((self.root / 'usr/local/share/cockpit/soda-updates').exists())
        self.assertFalse((self.root / 'usr/local/share/cockpit/soda-projects').exists())

    def test_persistence_and_privilege_wiring(self):
        service = (self.root / 'etc/systemd/system/soda-project@.service').read_text()
        self.assertIn('podman start --attach', service)
        self.assertNotIn('--replace', service)
        socket = (self.root / 'etc/systemd/system/soda-host.socket').read_text()
        self.assertIn('SocketGroup=soda', socket)
        self.assertIn('SocketMode=0660', socket)
        pam = (self.root / 'etc/pam.d/cockpit').read_text()
        self.assertIn('uid = 0', pam)

    def test_branding_closure(self):
        brand = self.root / 'etc/cockpit/branding'
        css = (brand / 'branding.css').read_text()
        self.assertNotIn('../theme/palette.css', css)
        for name in ['palette.css', 'theme.css', 'soda-logo-horizontal.svg', 'soda-logo-horizontal-dark.svg']:
            self.assertTrue((brand / name).is_file(), name)

    def test_forgejo_native_asset_paths(self):
        public = self.root / 'var/lib/soda/forgejo/gitea/public/assets'
        for name in ['logo.svg', 'favicon.svg', 'favicon.png']:
            self.assertTrue((public / 'img' / name).is_file(), name)
        css = (public / 'css/theme-soda-light.css').read_text()
        self.assertIn('../theme/palette.css', css)
        self.assertNotIn('../../theme/palette.css', css)
        self.assertTrue((public / 'theme/palette.css').is_file())

    def test_forgejo_branding_settings_and_review_exclusion(self):
        env = self.root / 'etc/soda/forgejo.env'
        values = dict(line.split('=', 1) for line in env.read_text().splitlines()
                      if line and not line.startswith('#'))
        self.assertEqual(values['FORGEJO____APP_NAME'], 'Soda OS')
        self.assertEqual(values['FORGEJO__ui_0x2E_meta__AUTHOR'], 'Soda OS')
        self.assertEqual(values['FORGEJO__server__STATIC_CACHE_TIME'], '0')
        for theme in ['soda-auto', 'forgejo-auto', 'forgejo-auto-deuteranopia-protanopia', 'forgejo-auto-tritanopia']:
            self.assertIn(theme, values['FORGEJO__ui__THEMES'].split(','))
        self.assertEqual(env.stat().st_mode & 0o777, 0o600)
        self.assertFalse((self.root / 'var/lib/soda/forgejo/gitea/public/assets/soda-theme-preview.html').exists())

    def test_operator_console_delivery(self):
        renderer = self.root / 'usr/local/libexec/soda/soda-console-welcome'
        self.assertTrue(renderer.is_file())
        self.assertTrue(renderer.stat().st_mode & 0o111)
        hook = (self.root / 'etc/profile.d/soda-console-welcome.sh').read_text()
        self.assertIn('case $- in', hook)
        self.assertIn('/usr/local/libexec/soda/soda-console-welcome', hook)
        link = self.root / 'usr/local/bin/soda-tailnet'
        self.assertTrue(link.is_symlink())
        self.assertEqual(os.readlink(link), '/usr/local/libexec/soda/soda-tailnet')


if __name__ == '__main__':
    unittest.main()

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
        for name in ['soda-dashboard', 'soda-host', 'soda-setup', 'soda-forgejo-tailnet', 'soda-runners', 'soda-runner-helper', 'soda-runner-launch']:
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


if __name__ == '__main__':
    unittest.main()

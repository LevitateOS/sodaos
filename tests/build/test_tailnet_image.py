"""Tailnet float checks; no image, daemon or provider operations."""

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class TailnetImage(unittest.TestCase):
    def test_no_stored_tailnet_pins(self):
        self.assertFalse((ROOT / 'appliance/locks').exists())
        build = (ROOT / 'internal/release/build/production.go').read_text()
        self.assertNotIn('locks/', build)

    def test_recipe_floats_on_build_args(self):
        recipe = (ROOT / 'appliance/tailnet.Containerfile').read_text()
        self.assertIn('ADD --checksum=sha256:${ARCHIVE_SHA256}', recipe)
        self.assertIn('https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz', recipe)
        self.assertIn('ENTRYPOINT ["/usr/local/bin/tailscaled"]', recipe)
        self.assertIn('COPY appliance/licenses/tailscale-LICENSE /usr/share/licenses/tailscale/LICENSE', recipe)
        self.assertIn(
            'Copyright (c) 2020 Tailscale Inc & contributors.',
            (ROOT / 'appliance/licenses/tailscale-LICENSE').read_text(),
        )

    def test_build_wires_live_inputs_without_lock(self):
        build = (ROOT / 'internal/release/build/production.go').read_text()
        self.assertIn('"--build-arg=TAILSCALE_VERSION="+tail.Version', build.replace(' ', ''))
        self.assertIn('"--build-arg=ARCHIVE_SHA256="+tail.SHA256', build.replace(' ', ''))
        self.assertIn('"appliance/tailnet.Containerfile"', build)

    def test_observed_versions_recorded_without_gate(self):
        info = (ROOT / 'scripts/native-build-info.py').read_text()
        self.assertNotIn('require_tailnet_release', info)
        self.assertIn('recorded in native-build.json as-is', info)


if __name__ == '__main__':
    unittest.main()

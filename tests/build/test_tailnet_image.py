"""Tailnet build inputs/version checks; no image, daemon or provider operations."""

import json
from pathlib import Path
import runpy
import sys
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))


class TailnetImage(unittest.TestCase):
    def test_locked_upstream_base_and_release_checksums(self):
        lock = json.loads((ROOT / 'appliance/locks/tailscale-image.json').read_text())
        # Build pins remain authoritative; live management has no release-number veto.
        self.assertRegex(lock['version'], r'^[0-9]+\.[0-9]+\.[0-9]+$')
        self.assertRegex(lock['base'], r'^docker.io/tailscale/alpine-base@sha256:[0-9a-f]{64}$')
        self.assertEqual(set(lock['sha256']), {'amd64', 'arm64'})
        for digest in lock['sha256'].values():
            self.assertRegex(digest, r'^[0-9a-f]{64}$')
        recipe = (ROOT / 'appliance/tailnet.Containerfile').read_text()
        self.assertIn('ADD --checksum=sha256:${ARCHIVE_SHA256}', recipe)
        self.assertIn('https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz', recipe)
        self.assertIn('ENTRYPOINT ["/usr/local/bin/tailscaled"]', recipe)
        self.assertIn('COPY appliance/licenses/tailscale-LICENSE /usr/share/licenses/tailscale/LICENSE', recipe)
        self.assertIn(
            'Copyright (c) 2020 Tailscale Inc & contributors.',
            (ROOT / 'appliance/licenses/tailscale-LICENSE').read_text(),
        )
        build = (ROOT / 'internal/nativebuild/production.go').read_text()
        self.assertIn('"--build-arg=TAILSCALE_VERSION="+tail.Version', build.replace(' ', ''))
        self.assertIn('"--build-arg=ARCHIVE_SHA256="+tail.SHA256[platform]', build.replace(' ', ''))
        self.assertIn('"appliance/tailnet.Containerfile"', build)

    def test_metadata_refuses_wrong_or_malformed_binary_versions(self):
        check = runpy.run_path(str(ROOT / 'scripts/native-build-info.py'))['require_tailnet_release']
        version = json.loads((ROOT / 'appliance/locks/tailscale-image.json').read_text())['version']
        clis = {
            '/usr/local/bin/tailscale': json.dumps({'short': version}),
            '/usr/local/bin/tailscaled': version + '-tscommit\n',
        }
        check(clis, version)
        for binary, output in [
            ('/usr/local/bin/tailscale', '{}'),
            ('/usr/local/bin/tailscale', '[]'),
            ('/usr/local/bin/tailscale', '{"short":"1.0.0"}'),
            ('/usr/local/bin/tailscaled', ''),
            ('/usr/local/bin/tailscaled', '1.0.0'),
        ]:
            with self.subTest(binary=binary, output=output), self.assertRaises(ValueError):
                check({**clis, binary: output}, version)

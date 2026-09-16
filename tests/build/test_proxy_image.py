"""Proxy image pin checks; no container, daemon or provider operations."""

import re
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
UNIT = ROOT / 'appliance/services/soda-proxy.container'


class ProxyImage(unittest.TestCase):
    def test_proxy_image_is_digest_pinned(self):
        images = [
            line.split('=', 1)[1].strip()
            for line in UNIT.read_text().splitlines()
            if line.startswith('Image=')
        ]
        self.assertEqual(len(images), 1)
        self.assertRegex(
            images[0],
            r'^docker\.io/library/caddy:2\.10\.2@sha256:[0-9a-f]{64}$',
        )

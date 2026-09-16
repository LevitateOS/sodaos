#!/usr/bin/env python3
"""Stage an upstream Linux Tea binary and license for the project image."""

import argparse
import hashlib
import json
import re
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RELEASES_API = 'https://gitea.com/api/v1/repos/gitea/tea/releases/latest'


def download(url, limit):
    request = urllib.request.Request(url, headers={'User-Agent': 'SodaOS-build'})
    with urllib.request.urlopen(request, timeout=60) as response:
        return response.read(limit + 1)


def latest_tag():
    release = json.loads(download(RELEASES_API, 1_000_000).decode('utf-8'))
    tag = release.get('tag_name', '') if isinstance(release, dict) else ''
    if not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+', tag):
        raise ValueError('Tea latest release is not a version tag')
    return tag


def fetch(arch, out):
    machine = {'x86_64': 62, 'aarch64': 183}[arch]
    if out.exists() or out.is_symlink():
        raise ValueError('Tea output already exists; select a fresh output directory')
    tag = latest_tag()
    version = tag[1:]
    filename = f'tea-{version}-linux-{"amd64" if arch == "x86_64" else "arm64"}'
    base = f'https://dl.gitea.com/tea/{version}'
    expected = None
    for line in download(base + '/checksums.txt', 1_000_000).decode('utf-8').splitlines():
        parts = line.split()
        if len(parts) == 2 and parts[1] == filename:
            expected = parts[0]
    if expected is None:
        raise ValueError('Tea checksums omit the requested archive')
    body = download(base + '/' + filename, 64_000_000)
    if len(body) > 64_000_000 or hashlib.sha256(body).hexdigest() != expected:
        raise ValueError('Tea binary checksum mismatch')
    if len(body) < 64 or body[:6] != b'\x7fELF\x02\x01' or int.from_bytes(body[18:20], 'little') != machine:
        raise ValueError('Tea binary is not ELF64 for the requested architecture')
    license_body = download(f'https://gitea.com/gitea/tea/raw/tag/{tag}/LICENSE', 1_000_000)
    if not license_body:
        raise ValueError('Tea license download is empty')
    out.mkdir(parents=True, exist_ok=False)
    (out / 'bin').mkdir()
    binary = out / 'bin/tea'
    binary.write_bytes(body)
    binary.chmod(0o755)
    license_dir = out / 'licenses/tea'
    license_dir.mkdir(parents=True)
    (license_dir / 'LICENSE').write_bytes(license_body)
    (license_dir / 'LICENSE').chmod(0o644)
    print(f'Upstream Tea {version} ({arch}) staged at {out}')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--arch', choices=('x86_64', 'aarch64'), required=True)
    parser.add_argument('--out', type=Path)
    args = parser.parse_args()
    fetch(args.arch, args.out or ROOT / '.artifacts/native' / args.arch / 'project-tools')

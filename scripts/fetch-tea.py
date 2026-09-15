#!/usr/bin/env python3
"""Stage an upstream Linux Tea binary and license for the project image."""

import argparse
import hashlib
import tomllib
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fetch(arch, out):
    machine = {'x86_64': 62, 'aarch64': 183}[arch]
    lock = tomllib.loads((ROOT / 'project-os/locks/tea-binary.toml').read_text())
    release = lock['linux'][arch]
    license_body = (ROOT / 'project-os/licenses/tea-LICENSE').read_bytes()
    if hashlib.sha256(license_body).hexdigest() != lock['license_sha256']:
        raise ValueError('Tea license checksum mismatch')
    if out.exists() or out.is_symlink():
        raise ValueError('Tea output already exists; select a fresh output directory')
    request = urllib.request.Request(release['url'], headers={'User-Agent': 'SodaOS-build'})
    with urllib.request.urlopen(request, timeout=60) as response:
        body = response.read(64_000_001)
    if len(body) > 64_000_000 or hashlib.sha256(body).hexdigest() != release['sha256']:
        raise ValueError('Tea binary checksum mismatch')
    if len(body) < 64 or body[:6] != b'\x7fELF\x02\x01' or int.from_bytes(body[18:20], 'little') != machine:
        raise ValueError('Tea binary is not ELF64 for the requested architecture')
    out.mkdir(parents=True, exist_ok=False)
    (out / 'bin').mkdir()
    binary = out / 'bin/tea'
    binary.write_bytes(body)
    binary.chmod(0o755)
    license_dir = out / 'licenses/tea'
    license_dir.mkdir(parents=True)
    (license_dir / 'LICENSE').write_bytes(license_body)
    (license_dir / 'LICENSE').chmod(0o644)
    print(f'Upstream Tea {lock["version"]} ({arch}) staged at {out}')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--arch', choices=('x86_64', 'aarch64'), required=True)
    parser.add_argument('--out', type=Path)
    args = parser.parse_args()
    fetch(args.arch, args.out or ROOT / '.artifacts/native' / args.arch / 'project-tools')

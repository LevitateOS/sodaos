#!/usr/bin/env python3
"""Later native build input fetch; never invoked by source-only milestones."""
import argparse, hashlib, platform, tarfile, tempfile, tomllib, urllib.request
from pathlib import Path
p = argparse.ArgumentParser()
p.add_argument('--arch', choices=['x86_64', 'aarch64'], required=True)
p.add_argument('--out', required=True)
a = p.parse_args()
if platform.system() != 'Linux' or platform.machine() != a.arch:
    p.error('matching native Linux required')
root = Path(__file__).resolve().parents[1]
lock = tomllib.loads((root / 'appliance/locks/github-runner-source.toml').read_text())
asset = next(x for x in lock['asset'] if x['architecture'] == a.arch)
output = Path(a.out)
output.mkdir(parents=True, exist_ok=False)
with tempfile.TemporaryDirectory(prefix='soda-runner-') as temp:
    archive = Path(temp) / asset['archive']
    digest = hashlib.sha256()
    with urllib.request.urlopen(asset['url'], timeout=120) as response, archive.open('wb') as f:
        while block := response.read(1024 * 1024):
            digest.update(block)
            f.write(block)
    if digest.hexdigest() != asset['sha256']:
        raise SystemExit('GitHub runner checksum mismatch')
    with tarfile.open(archive) as tar:
        tar.extractall(output, filter='data')

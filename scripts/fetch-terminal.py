#!/usr/bin/env python3
"""Fetch exact upstream terminal distributions, never runtime/CDN dependencies."""
import base64
import hashlib
import io
import json
from pathlib import Path
import tarfile
import urllib.request

ROOT = Path(__file__).resolve().parents[1]


def fetch(out):
    out.mkdir(parents=True, exist_ok=True)
    for item in json.loads((ROOT / 'appliance/terminal-assets.lock.json').read_text()):
        if all((out / f['file']).is_file() and hashlib.sha256((out / f['file']).read_bytes()).hexdigest() == f['sha256'] for f in item['files']):
            continue
        with urllib.request.urlopen(item['url'], timeout=30) as response:
            body = response.read(10_000_001)
        if len(body) > 10_000_000 or 'sha512-' + base64.b64encode(hashlib.sha512(body).digest()).decode() != item['integrity']:
            raise ValueError('terminal archive integrity mismatch')
        with tarfile.open(fileobj=io.BytesIO(body), mode='r:gz') as archive:
            for f in item['files']:
                member = archive.getmember(f['member'])
                if not member.isfile() or member.size > 2_000_000 or Path(f['file']).name != f['file']:
                    raise ValueError('invalid terminal distribution member')
                data = archive.extractfile(member).read()
                if hashlib.sha256(data).hexdigest() != f['sha256']:
                    raise ValueError('terminal asset integrity mismatch')
                (out / f['file']).write_bytes(data)


if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=Path, required=True)
    fetch(parser.parse_args().out)

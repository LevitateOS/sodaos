#!/usr/bin/env python3
"""Combine exact native Forgejo INI bytes with Soda additions, without reserializing.

Native JSON catalogs remain untouched. Custom INI catalogs replace native files;
this command therefore requires the complete extracted native catalog as input.
"""
import argparse
import configparser
import hashlib
import json
import urllib.request
from pathlib import Path


def merge(native: str, additions: str) -> str:
    def parse(text):
        ini = configparser.ConfigParser(interpolation=None, strict=True, delimiters=('=',), comment_prefixes=('#', ';'), empty_lines_in_values=False)
        ini.optionxform = str
        ini.read_string(text)
        return ini
    base, extra = parse(native), parse(additions)
    if not base.has_section('common') or not base.has_section('settings'):
        raise ValueError('Expected a complete native Forgejo English catalog')
    if extra.sections() != ['soda']:
        raise ValueError('Additions must use only the Soda namespace')
    # Reject the entire section collision: appending a second INI section can
    # change parser behavior even when the individual keys differ.
    if base.has_section('soda'):
        raise ValueError('Native catalog already owns the Soda namespace')
    return native.rstrip() + '\n\n' + additions.rstrip() + '\n'


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    native = parser.add_mutually_exclusive_group(required=True)
    native.add_argument('--native', type=Path)
    native.add_argument('--lock', type=Path, help='Fetch the exact locked native catalog at build time')
    parser.add_argument('--additions', type=Path, default=Path('appliance/forgejo/i18n/en-US.ini'))
    parser.add_argument('--out', required=True, type=Path)
    args = parser.parse_args()
    if args.lock:
        lock = json.loads(args.lock.read_text())
        if not lock['url'].startswith('https://codeberg.org/forgejo/forgejo/raw/tag/'):
            parser.error('unexpected native catalog source')
        with urllib.request.urlopen(lock['url'], timeout=30) as response:
            data = response.read(1024 * 1024 + 1)
        if len(data) > 1024 * 1024 or hashlib.sha256(data).hexdigest() != lock['sha256']:
            parser.error('native catalog differs from locked bytes')
        native = data.decode('utf-8')
    else:
        native = args.native.read_text()
    output = merge(native, args.additions.read_text())
    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open('x') as stream:
        stream.write(output)

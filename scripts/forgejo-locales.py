#!/usr/bin/env python3
"""Combine exact native Forgejo INI bytes with Soda additions, without reserializing.

Native JSON catalogs remain untouched. Custom INI catalogs replace native files;
this command therefore requires the complete extracted native catalog as input.
"""
import argparse
import configparser
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
    parser.add_argument('--native', required=True, type=Path)
    parser.add_argument('--additions', type=Path, default=Path('appliance/forgejo/i18n/en-US.ini'))
    parser.add_argument('--out', required=True, type=Path)
    args = parser.parse_args()
    output = merge(args.native.read_text(), args.additions.read_text())
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(output)

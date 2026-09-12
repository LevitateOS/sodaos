#!/usr/bin/env python3
"""Sample the canonical polygon emblem into a 32-column, 16-row ASCII mark."""
import argparse
import re
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'assets/branding/source/soda-symbol-brutalist.svg'
OUT = ROOT / 'assets/branding/terminal'


def polygons(data):
    # This emblem uses only absolute polygon commands. Fail on changed geometry syntax.
    tokens = re.findall(r'[A-Za-z]|-?\d+(?:\.\d+)?', data)
    result, points, x, y = [], [], 0.0, 0.0
    i = 0
    while i < len(tokens):
        command = tokens[i]
        i += 1
        if command in ('M', 'L'):
            x, y = float(tokens[i]), float(tokens[i + 1])
            i += 2
        elif command == 'H':
            x = float(tokens[i]); i += 1
        elif command == 'V':
            y = float(tokens[i]); i += 1
        elif command == 'Z':
            assert len(points) >= 3
            result.append(points)
            points = []
            continue
        else:
            raise ValueError(f'Unsupported emblem command: {command}')
        points.append((x, y))
    assert not points and result
    return result


def inside(x, y, polygon):
    hit = False
    for (ax, ay), (bx, by) in zip(polygon, polygon[1:] + polygon[:1]):
        if (ay > y) != (by > y) and x < (bx - ax) * (y - ay) / (by - ay) + ax:
            hit = not hit
    return hit


def render():
    svg = ET.parse(SOURCE).getroot()
    assert svg.attrib['viewBox'] == '0 0 128 128'
    paths = svg.findall('{http://www.w3.org/2000/svg}path')
    assert [p.attrib['fill'] for p in paths] == ['#df001b', '#101010']
    assert all(p.attrib['fill-rule'] == 'evenodd' for p in paths)
    layers = [polygons(p.attrib['d']) for p in paths]
    rows = []
    for row in range(16):
        cells = []
        for column in range(32):
            char = ' '
            for shape, glyph in zip(layers, '#@'):
                if sum(inside((column + .5) * 4, (row + .5) * 8, p) for p in shape) % 2:
                    char = glyph
            cells.append(char)
        rows.append(''.join(cells).rstrip())
    colored = []
    for row in rows:
        line, previous = '', None
        for char in row:
            color = '1' if char == '#' else '2'
            if char != ' ' and color != previous:
                line += '$' + color
                previous = color
            line += char
        colored.append(line + '$2')
    return '\n'.join(colored) + '\n', '\n'.join(rows) + '\n\nSODA OS\n'


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    for name, text in zip(('sodaos.txt', 'motd.txt'), render()):
        path = OUT / name
        if args.check:
            assert path.read_text() == text, f'Stale terminal branding: {path}'
        else:
            path.write_text(text)

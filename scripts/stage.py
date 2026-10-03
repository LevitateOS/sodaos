#!/usr/bin/env python3
"""Stage existing native build outputs plus configuration; does not build or install."""

import argparse
import hashlib
import json
import platform
import shutil
import struct
from pathlib import Path

p = argparse.ArgumentParser()
p.add_argument('--arch', choices=['x86_64'], required=True)
p.add_argument('--host-context', type=Path, required=True)
p.add_argument('--forgejo-context', type=Path, required=True)
a = p.parse_args()
if platform.system() != 'Linux' or platform.machine() != a.arch:
    p.error('matching native Linux required')
source = Path(__file__).resolve().parents[1]
build = source / '.artifacts/native' / a.arch
stage = a.host_context / 'rootfs'
forgejo = a.forgejo_context / 'forgejo'
for directory in (stage, a.forgejo_context):
    if not directory.is_absolute() or directory.resolve() != directory or not directory.is_dir():
        p.error('real prepared host and fresh Forgejo context directories required')
if forgejo.exists() or forgejo.is_symlink():
    p.error('occupied Forgejo presentation refused')


def copy(src, dest, mode=None, root=stage):
    target = root / dest.lstrip('/')
    fresh = {parent for parent in target.parents if not parent.exists()}
    target.parent.mkdir(parents=True, exist_ok=True)
    for parent in target.parents:
        if parent == root:
            break
        if parent in fresh:
            parent.chmod(0o755)
    if target.exists() or target.is_symlink():
        p.error('occupied staging file refused')
    shutil.copy2(src, target)
    if mode is not None:
        target.chmod(mode)
    return target


# Programs/units/configuration are already emitted directly by Go.
# Public, pinned tools only; runtime credentials never enter a build context.
muse_tools = '/usr/share/soda/muse-tools'
for name in ['muse', 'muse-native', 'soda-identity-compose']:
    copy(build / 'project-tools/bin' / name, muse_tools + '/' + name, 0o755)
# Config-root branding avoids immutable /usr/share and native package conflicts.
brand = stage / 'etc/cockpit/branding'
brand.mkdir(parents=True)
for name in ['branding.css', 'theme.css']:
    shutil.copy2(source / 'assets/branding/cockpit' / name, brand / name)
css = brand / 'branding.css'
css.write_text(css.read_text().replace('../theme/palette.css', 'palette.css'))
shutil.copy2(source / 'assets/branding/theme/palette.css', brand / 'palette.css')
for name in ['soda-symbol-brutalist.svg', 'soda-symbol-brutalist-dark.svg']:
    shutil.copy2(source / 'assets/branding/source' / name, brand / name)
shutil.copy2(source / 'assets/branding/forgejo/apple-touch-icon.png', brand / 'apple-touch-icon.png')
# ICO is just a container: reuse the canonical rendered PNGs without redrawing.
frames = [
    (size, (source / 'assets/branding/forgejo' / name).read_bytes())
    for size, name in [(16, 'favicon-16.png'), (32, 'favicon.png')]
]
entries, offset = [], 6 + 16 * len(frames)
for size, data in frames:
    entries.append(struct.pack('<BBBBHHII', size, size, 0, 0, 1, 32, len(data), offset))
    offset += len(data)
(brand / 'favicon.ico').write_bytes(
    struct.pack('<HHH', 0, 1, len(frames)) + b''.join(entries) + b''.join(data for _, data in frames)
)
for target in [brand, *brand.rglob('*')]:
    target.chmod(0o755 if target.is_dir() else 0o644)
# Adapt directly to the final Forgejo build context for vendor images.
forgejo.mkdir(parents=True)
forgejo.chmod(0o755)
custom = forgejo / 'public/assets'
shutil.copytree(source / 'assets/branding/forgejo/css', custom / 'css')
shutil.copytree(source / 'assets/branding/theme', custom / 'theme')
for stylesheet in (custom / 'css').glob('*.css'):
    # The staged copy lives beside Forgejo's native css/ instead of the Soda
    # payload dir, so upstream theme imports become same-directory too.
    stylesheet.write_text(
        stylesheet.read_text()
        .replace('../../theme/palette.css', '../theme/palette.css')
        .replace('../../../css/theme-forgejo-', 'theme-forgejo-')
    )
images = custom / 'img'
images.mkdir()
for name in ['logo.svg', 'favicon.svg']:
    shutil.copy2(source / 'assets/branding/source/soda-symbol-brutalist.svg', images / name)
for name in ['logo.png', 'favicon.png', 'apple-touch-icon.png']:
    shutil.copy2(source / 'assets/branding/forgejo' / name, images / name)
# copytree/copy2 preserve checkout modes (a private worktree may be 0700/0600).
# Normalize only this run-owned public adaptation, never canonical source assets.
for target in custom.rglob('*'):
    target.chmod(0o755 if target.is_dir() else 0o644)
# Exact reviewed presentation payload: templates, local assets, fonts/notices and
# generated full native locale. Never copy a mutable Forgejo tree or partial hooks.
payload = json.loads((source / 'internal/release/build/forgejo-payload.json').read_text())
# Branding uses the same reviewed font files/notices, not a Cockpit extension.
for dest, src in payload.items():
    if dest.startswith('public/assets/soda/fonts/'):
        relative = dest.removeprefix('public/assets/soda/fonts/')
        copy(source / src, '/etc/cockpit/branding/fonts/' + relative, 0o644)
locked = {
    asset['file']: asset['sha256']
    for item in json.loads((source / 'appliance/terminal-assets.lock.json').read_text())
    for asset in item['files']
}
for name, origin in payload.items():
    if Path(name).is_absolute() or '..' in Path(name).parts:
        p.error('invalid Forgejo payload destination')
    if origin.startswith('@build/'):
        src = build / origin.removeprefix('@build/')
    else:
        src = source / origin
    if src.is_symlink() or not src.is_file():
        p.error('missing or unsafe Forgejo payload input')
    if (
        origin.startswith('@build/terminal-assets/')
        and hashlib.sha256(src.read_bytes()).hexdigest() != locked[src.name]
    ):
        p.error('terminal asset differs from locked upstream bytes')
    target = copy(src, name, 0o644, root=forgejo)
    for parent in target.parents:
        parent.chmod(0o755)
        if parent == forgejo:
            break
# MOTD is plain text; fastfetch alone interprets the logo's color placeholders.
copy(source / 'assets/branding/terminal/motd.txt', '/etc/motd', 0o644)
copy(source / 'assets/branding/terminal/sodaos.txt', '/usr/share/soda/fastfetch/sodaos.txt', 0o644)
fastfetch = copy(source / 'assets/branding/terminal/fastfetch.jsonc', '/etc/fastfetch/config.jsonc', 0o644)
fastfetch.write_text(fastfetch.read_text().replace('/usr/local/share/soda/', '/usr/share/soda/'))
copy(source / 'appliance/config/forgejo.env', '/etc/soda/forgejo.env', 0o600)
copy(source / 'appliance/config/proxy.Caddyfile', '/etc/soda/proxy.Caddyfile', 0o644)
print(stage)

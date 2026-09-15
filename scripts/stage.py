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
p.add_argument('--arch', choices=['x86_64', 'aarch64'], required=True)
p.add_argument('--host-context', type=Path)
p.add_argument('--forgejo-context', type=Path)
a = p.parse_args()
if bool(a.host_context) != bool(a.forgejo_context):
    p.error('host and Forgejo contexts must be supplied together')
vendor = a.host_context is not None
if platform.system() != 'Linux' or platform.machine() != a.arch:
    p.error('matching native Linux required')
source = Path(__file__).resolve().parents[1]
build = source / '.artifacts/native' / a.arch
stage = a.host_context / 'rootfs' if vendor else build / 'rootfs'
forgejo = a.forgejo_context / 'forgejo' if vendor else stage / 'var/lib/soda/forgejo/gitea'
if vendor:
    for directory in (stage, a.forgejo_context):
        if not directory.is_absolute() or directory.resolve() != directory or not directory.is_dir():
            p.error('real prepared host and fresh Forgejo context directories required')
    if forgejo.exists() or forgejo.is_symlink():
        p.error('occupied Forgejo presentation refused')
else:
    if stage.exists() or stage.is_symlink():
        p.error('rootfs staging already exists; retain the attempt and use fresh output')
    stage.mkdir(parents=True)


def copy(src, dest, mode=None, root=stage):
    target = root / dest.lstrip('/')
    target.parent.mkdir(parents=True, exist_ok=True)
    for parent in target.parents:
        if parent == root:
            break
        parent.chmod(0o755)
    if target.exists() or target.is_symlink():
        p.error('occupied staging file refused')
    shutil.copy2(src, target)
    if mode is not None:
        target.chmod(mode)
    return target


# Vendor programs/units/configuration are already emitted directly by Go.
# Keep the retiring writable layout usable; never create it for vendor assets.
if not vendor:
    for command in (source / 'cmd').iterdir():
        if command.is_dir():
            if command.name in {'soda-artifacts', 'soda-acceptance'}:
                p.error('outside support tools must not be staged on the appliance')
            copy(build / 'bin' / command.name, f'/usr/local/libexec/soda/{command.name}', 0o755)
    for unit in (source / 'appliance/services').iterdir():
        folder = '/etc/containers/systemd' if unit.suffix == '.container' else '/etc/systemd/system'
        copy(unit, f'{folder}/{unit.name}', 0o644)
# Stock Cockpit only. Never copy an ignored retired cockpit/dist tree.
configs = {
    'soda.sysusers': '/etc/sysusers.d/soda.conf',
    'runners.sysusers': '/etc/sysusers.d/soda-runners.conf',
    'soda.tmpfiles': '/etc/tmpfiles.d/soda.conf',
    'runners.tmpfiles': '/etc/tmpfiles.d/soda-runners.conf',
    '90-soda-routing.conf': '/etc/sysctl.d/90-soda-routing.conf',
    'cockpit.pam': '/etc/pam.d/cockpit',
    'cockpit.conf': '/etc/cockpit/cockpit.conf',
    'cockpit.socket.conf': '/etc/systemd/system/cockpit.socket.d/10-soda.conf',
    'console-welcome.sh': '/etc/profile.d/soda-console-welcome.sh',
}
if not vendor:
    for src, dest in configs.items():
        copy(source / 'appliance/config' / src, dest, 0o644)
    (stage / 'etc/cockpit/disallowed-users').write_text('')
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
    stylesheet.write_text(stylesheet.read_text().replace('../../theme/palette.css', '../theme/palette.css'))
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
        if parent == (forgejo if vendor else stage / 'var'):
            break
# MOTD is plain text; fastfetch alone interprets the logo's color placeholders.
copy(source / 'assets/branding/terminal/motd.txt', '/etc/motd', 0o644)
share = '/usr/share/soda' if vendor else '/usr/local/share/soda'
copy(source / 'assets/branding/terminal/sodaos.txt', share + '/fastfetch/sodaos.txt', 0o644)
fastfetch = copy(source / 'assets/branding/terminal/fastfetch.jsonc', '/etc/fastfetch/config.jsonc', 0o644)
if vendor:
    fastfetch.write_text(fastfetch.read_text().replace('/usr/local/share/soda/', '/usr/share/soda/'))
else:
    copy(source / 'appliance/bin/soda-activate', '/usr/local/sbin/soda-activate', 0o750)
    copy(source / 'appliance/bin/soda-console-welcome', '/usr/local/libexec/soda/soda-console-welcome', 0o755)
    tailnet_cli = stage / 'usr/local/bin/soda-tailnet'
    tailnet_cli.parent.mkdir(parents=True, exist_ok=True)
    tailnet_cli.symlink_to('/usr/local/libexec/soda/soda-tailnet')
    link = stage / 'usr/local/sbin/soda-setup'
    link.symlink_to('/usr/local/libexec/soda/soda-setup')
copy(source / 'appliance/config/forgejo.env', '/etc/soda/forgejo.env', 0o600)
copy(source / 'appliance/config/proxy.Caddyfile', '/etc/soda/proxy.Caddyfile', 0o644)
print(stage)

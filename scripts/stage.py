#!/usr/bin/env python3
"""Stage existing native build outputs plus configuration; does not build or install."""
import argparse, os, platform, shutil
from pathlib import Path
p = argparse.ArgumentParser()
p.add_argument('--arch', choices=['x86_64', 'aarch64'], required=True)
a = p.parse_args()
if platform.system() != 'Linux' or platform.machine() != a.arch:
    p.error('matching native Linux required')
source = Path(__file__).resolve().parents[1]
build = source / '.artifacts/native' / a.arch
stage = build / 'rootfs'
if stage.exists():
    p.error('rootfs staging already exists; inspect and explicitly clear only this generated directory before restaging')
stage.mkdir(parents=True)
def copy(src, dest, mode=None):
    target = stage / dest.lstrip('/')
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, target)
    if mode is not None:
        target.chmod(mode)
    return target
for command in (source / 'cmd').iterdir():
    if command.is_dir():
        copy(build / 'bin' / command.name, f'/usr/local/libexec/soda/{command.name}', 0o755)
for unit in (source / 'appliance/services').iterdir():
    folder = '/etc/containers/systemd' if unit.suffix == '.container' else '/etc/systemd/system'
    copy(unit, f'{folder}/{unit.name}', 0o644)
for page in ['tailscale', 'runners']:
    shutil.copytree(source / 'cockpit/dist' / f'soda-{page}', stage / 'usr/local/share/cockpit' / f'soda-{page}')
shutil.copytree(build / 'github-actions-runner', stage / 'usr/local/lib/soda/github-actions-runner')
configs = {
    'soda.sysusers': '/etc/sysusers.d/soda.conf',
    'runners.sysusers': '/etc/sysusers.d/soda-runners.conf',
    'soda.tmpfiles': '/etc/tmpfiles.d/soda.conf',
    'runners.tmpfiles': '/etc/tmpfiles.d/soda-runners.conf',
    '90-soda-routing.conf': '/etc/sysctl.d/90-soda-routing.conf',
    'cockpit.pam': '/etc/pam.d/cockpit',
    'cockpit.conf': '/etc/cockpit/cockpit.conf',
    'cockpit.socket.conf': '/etc/systemd/system/cockpit.socket.d/10-soda.conf',
    'proxy.Caddyfile': '/etc/soda/proxy.Caddyfile',
    'console-welcome.sh': '/etc/profile.d/soda-console-welcome.sh',
}
for src, dest in configs.items():
    copy(source / 'appliance/config' / src, dest, 0o644)
(stage / 'etc/cockpit/disallowed-users').write_text('')
# Config-root branding avoids immutable /usr/share and native package conflicts.
brand = stage / 'etc/cockpit/branding'
brand.mkdir(parents=True)
for name in ['branding.css', 'theme.css', 'login-background-light.svg', 'login-background-dark.svg', 'favicon.ico', 'apple-touch-icon.png']:
    shutil.copy2(source / 'assets/branding/cockpit' / name, brand / name)
css = brand / 'branding.css'
css.write_text(css.read_text().replace('../theme/palette.css', 'palette.css'))
shutil.copy2(source / 'assets/branding/theme/palette.css', brand / 'palette.css')
for name in ['soda-logo-horizontal.svg', 'soda-logo-horizontal-dark.svg', 'soda-symbol.svg']:
    shutil.copy2(source / 'assets/branding/source' / name, brand / name)
# Adapt the canonical asset tree to Forgejo's native /assets URL root.
custom = stage / 'var/lib/soda/forgejo/gitea/public/assets'
shutil.copytree(source / 'assets/branding/forgejo/css', custom / 'css')
shutil.copytree(source / 'assets/branding/theme', custom / 'theme')
for stylesheet in (custom / 'css').glob('*.css'):
    stylesheet.write_text(stylesheet.read_text().replace('../../theme/palette.css', '../theme/palette.css'))
images = custom / 'img'
images.mkdir()
for name in ['logo.svg', 'favicon.svg']:
    shutil.copy2(source / 'assets/branding/source/soda-symbol.svg', images / name)
for name in ['logo.png', 'favicon.png', 'apple-touch-icon.png']:
    shutil.copy2(source / 'assets/branding/forgejo' / name, images / name)
copy(source / 'assets/branding/terminal/sodaos.txt', '/etc/motd', 0o644)
copy(source / 'appliance/bin/soda-activate', '/usr/local/sbin/soda-activate', 0o750)
copy(source / 'appliance/bin/soda-console-welcome', '/usr/local/libexec/soda/soda-console-welcome', 0o755)
tailnet_cli = stage / 'usr/local/bin/soda-tailnet'
tailnet_cli.parent.mkdir(parents=True, exist_ok=True)
tailnet_cli.symlink_to('/usr/local/libexec/soda/soda-tailnet')
link = stage / 'usr/local/sbin/soda-setup'
link.symlink_to('/usr/local/libexec/soda/soda-setup')
copy(source / 'appliance/config/forgejo.env', '/etc/soda/forgejo.env', 0o600)
print(stage)

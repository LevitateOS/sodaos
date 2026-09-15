#!/usr/bin/env python3
"""Seal public build inputs around the core-owned stage, never instance state."""
import argparse
import json
import platform
import re
import shutil
import signal
import sys
import subprocess
from pathlib import Path

from pathlib import Path


class Progress:
    def next(self, label):
        print('PROGRESS', label, file=sys.stderr)

    def done(self):
        return None

    def end(self, code=None):
        return code


def exit_code(error):
    return getattr(error, 'returncode', 1) or 1


progress = Progress()


def output(args):
    if args[0] == 'podman':
        args = [args[0], '--remote=false', *args[1:]]
    return subprocess.check_output(args, text=True).strip()


def require_tailnet_release(clis, version):
    cli = json.loads(clis['/usr/local/bin/tailscale'])
    daemon = clis['/usr/local/bin/tailscaled'].splitlines()
    if (not isinstance(cli, dict) or cli.get('short') != version or not daemon
            or daemon[0].split('-', 1)[0] != version):
        raise ValueError('Tailnet image binaries differ from locked release')


def collect(root, arch, revision):
    progress.next('Native / Collect build inputs and tool versions')
    if platform.system() != 'Linux' or platform.machine() != arch:
        raise ValueError('matching-native Linux required')
    if not re.fullmatch('[0-9a-f]{40}', revision):
        raise ValueError('full revision required')
    stage = root / '.artifacts/native' / arch
    inputs = stage / 'inputs'
    inputs.mkdir()
    for source, name in (
        ('go.mod', 'go.mod'), ('go.sum', 'go.sum'),
        ('project-os/locks/tea-binary.toml', 'tea-binary.toml'),
        ('appliance/locks/coreos-qemu.json', 'coreos-qemu.json'),
        ('appliance/locks/tailscale-image.json', 'tailscale-image.json'),
        ('package.json', 'package.json'),
        ('tools/lit-check/package.json', 'lit-check-package.json'),
        ('bun.lock', 'bun.lock'),
        ('bunfig.toml', 'bunfig.toml'),
    ):
        shutil.copyfile(root / source, inputs / name)
    shutil.copyfile(root / 'scripts/install-native.sh', stage / 'install-native.sh')
    (stage / 'install-native.sh').chmod(0o755)
    notices = stage / 'notices'
    notices.mkdir()
    shutil.copyfile(root / 'docs/native-support-notices.md', notices / 'README.md')
    shutil.copyfile(root / 'project-os/licenses/tea-LICENSE', notices / 'tea-LICENSE')
    shutil.copyfile(root / 'LICENSE', notices / 'soda-LICENSE')
    shutil.copyfile(root / 'NOTICE', notices / 'soda-NOTICE')
    shutil.copyfile(root / 'appliance/licenses/avatar-dependencies.txt', notices / 'avatar-dependencies.txt')
    tools = {name: output(command) for name, command in {
        'go': ['go', 'version'],
        'bun': ['bun', '--version'], 'podman': ['podman', '--version'],
        'python': ['python3', '--version'], 'kernel': ['uname', '-r'],
    }.items()}
    images = {}
    for name in ('base', 'project-os', 'dashboard', 'forgejo', 'caddy', 'tailnet'):
        progress.next('Native / Inspect built image: ' + name)
        image = (stage / (name + '.iid')).read_text().strip()
        if not re.fullmatch('(?:sha256:)?[0-9a-f]{64}', image):
            raise ValueError('invalid resolved image ID')
        images[name] = {
            'ID': image,
            'RegistryDigest': output(['podman', 'image', 'inspect', '--format', '{{.Digest}}', image]),
            'RepositoryDigests': json.loads(output(['podman', 'image', 'inspect', '--format', '{{json .RepoDigests}}', image])),
        }
        if name in ('project-os', 'dashboard'):
            # A fresh read-only build-inspection container, never a Soda project.
            # No application entrypoint, network, persistent mount or provider use.
            images[name]['RPMs'] = sorted(output(['podman', 'run', '--rm', '--read-only', '--network=none', '--entrypoint=/usr/bin/rpm', image, '-qa', '--qf', '%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}\n']).splitlines())
        if name == 'tailnet':
            # Version-only, read-only and networkless: no daemon or enrollment.
            images[name]['CLIs'] = {binary: output(['podman', 'run', '--rm', '--read-only', '--network=none', '--entrypoint=' + binary, image, *flags]) for binary, flags in (('/usr/local/bin/tailscale', ('version', '--json')), ('/usr/local/bin/tailscaled', ('--version',)))}
            version = json.loads((root / 'appliance/locks/tailscale-image.json').read_text())['version']
            require_tailnet_release(images[name]['CLIs'], version)
        if name == 'project-os':
            images[name]['CLIs'] = {binary: output(['podman', 'run', '--rm', '--read-only', '--network=none', '--entrypoint=' + binary, image, flag]) for binary, flag in (('/usr/local/bin/tea', '--version'), ('/usr/bin/gh', '--version'), ('/usr/bin/tmux', '-V'))}
    progress.next('Native / Finish build metadata')
    with (inputs / 'native-build.json').open('x') as f:
        json.dump({'Revision': revision, 'Architecture': arch, 'Tools': tools, 'Images': images}, f, indent=2)
        f.write('\n')
    progress.end()


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--arch', choices=('x86_64', 'aarch64'), required=True)
    p.add_argument('--revision', required=True)
    args = p.parse_args()
    signal.signal(signal.SIGTERM, lambda signum, frame: sys.exit(143))
    try:
        collect(Path(__file__).resolve().parents[1], args.arch, args.revision)
    except BaseException as error:
        progress.end(exit_code(error))
        raise

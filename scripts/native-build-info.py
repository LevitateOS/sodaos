#!/usr/bin/env python3
"""Seal public build inputs around the core-owned stage, never instance state."""
import argparse
import json
import platform
import re
import shutil
import subprocess
from pathlib import Path


def output(args):
    if args[0] == 'podman':
        args = [args[0], '--remote=false', *args[1:]]
    return subprocess.check_output(args, text=True).strip()


def collect(root, arch, revision):
    if platform.system() != 'Linux' or platform.machine() != arch:
        raise ValueError('matching-native Linux required')
    if not re.fullmatch('[0-9a-f]{40}', revision):
        raise ValueError('full revision required')
    stage = root / '.artifacts/native' / arch
    inputs = stage / 'inputs'
    inputs.mkdir()
    for source, name in (
        ('go.mod', 'go.mod'), ('go.sum', 'go.sum'),
        ('project-os/locks/tea-source.toml', 'tea-source.toml'),
        ('appliance/locks/github-runner-source.toml', 'github-runner-source.toml'),
        ('appliance/locks/coreos-qemu.json', 'coreos-qemu.json'),
        ('cockpit/package.json', 'cockpit-package.json'),
        ('cockpit/pnpm-lock.yaml', 'cockpit-pnpm-lock.yaml'),
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
        'go': ['go', 'version'], 'node': ['node', '--version'],
        'pnpm': ['pnpm', '--version'], 'podman': ['podman', '--version'],
        'python': ['python3', '--version'], 'kernel': ['uname', '-r'],
    }.items()}
    images = {}
    for name in ('base', 'project-os', 'dashboard', 'forgejo', 'caddy'):
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
        if name == 'project-os':
            images[name]['CLIs'] = {binary: output(['podman', 'run', '--rm', '--read-only', '--network=none', '--entrypoint=' + binary, image, flag]) for binary, flag in (('/usr/local/bin/tea', '--version'), ('/usr/bin/gh', '--version'), ('/usr/bin/tmux', '-V'))}
    with (inputs / 'native-build.json').open('x') as f:
        json.dump({'Revision': revision, 'Architecture': arch, 'Tools': tools, 'Images': images}, f, indent=2)
        f.write('\n')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--arch', choices=('x86_64', 'aarch64'), required=True)
    p.add_argument('--revision', required=True)
    args = p.parse_args()
    collect(Path(__file__).resolve().parents[1], args.arch, args.revision)

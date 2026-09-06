#!/usr/bin/env python3
"""Build the predecessor's pinned Tea source for the project image, not the host.

Authored for later execution: fetches dependencies, invokes upstream make/Go,
and executes the resulting CLI's --version. No login or provider registration.
"""
import argparse
import os
import platform
import shutil
import subprocess
import tarfile
import tempfile
import tomllib
from pathlib import Path


def build(arch):
    if platform.system() != 'Linux' or platform.machine() != arch:
        raise SystemExit('matching-native Linux required')
    root = Path(__file__).resolve().parents[1]
    lock = tomllib.loads((root / 'project-os/locks/tea-source.toml').read_text())
    output = root / '.artifacts/native' / arch / 'project-tools'
    output.mkdir(parents=True, exist_ok=False)
    subprocess.run([str(root / 'scripts/fetch-tea-source.sh')], check=True)
    archive = root / '.artifacts/tools' / lock['source_archive']
    with tempfile.TemporaryDirectory(prefix='tea-source-', dir=output.parent) as work:
        with tarfile.open(archive) as tar:
            tar.extractall(work, filter='data')
        sources = [p for p in Path(work).iterdir()
                   if p.is_dir() and (p / 'go.mod').is_file() and (p / 'Makefile').is_file()]
        if len(sources) != 1:
            raise SystemExit('expected one upstream Tea source root')
        env = os.environ | {
            'CGO_ENABLED': '0', 'GOOS': 'linux',
            'GOARCH': {'x86_64': 'amd64', 'aarch64': 'arm64'}[arch],
            'GOTOOLCHAIN': 'local',
            'TEA_VERSION': lock['version'],
        }
        # Tea appends shell-quoted tags/linker arguments to its Make GOFLAGS.
        # An inherited GOFLAGS would remain exported to Go, where those separate
        # argument values are invalid. Keep readonly mode on the build command,
        # while preserving upstream's own version/SDK linker flags.
        env.pop('GOFLAGS', None)
        subprocess.run(['make', 'BUILDMODE=-buildvcs=false -mod=readonly', 'build'], cwd=sources[0], env=env, check=True)
        (output / 'bin').mkdir()
        binary = output / 'bin/tea'
        shutil.copy2(sources[0] / 'tea', binary)
        binary.chmod(0o755)
        version = subprocess.run([str(binary), '--version'], check=True, capture_output=True, text=True)
        # Pinned Tea's version.Format always surrounds Version with bold/reset
        # SGR, even with NO_COLOR and piped stdout. Match its actual version field.
        plain_version = version.stdout.replace('\x1b[1m', '').replace('\x1b[0m', '')
        if not plain_version.startswith('Version: ' + lock['version'] + '\tgolang: '):
            raise SystemExit('built Tea version does not match its source lock')
    license_dir = output / 'licenses/tea'
    license_dir.mkdir(parents=True)
    shutil.copy2(root / 'project-os/licenses/tea-LICENSE', license_dir / 'LICENSE')
    print(f'Project Tea binary staged at {output}; no login or installation performed.')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--arch', choices=['x86_64', 'aarch64'], required=True)
    build(parser.parse_args().arch)

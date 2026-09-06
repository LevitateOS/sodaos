#!/usr/bin/env python3
"""Build the predecessor's pinned Tea source for the project image, not the host.

Authored for later execution: fetches dependencies, invokes upstream make/Go,
and executes the resulting CLI's --version. No login or provider registration.
"""
import argparse
import os
import platform
import re
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
            'GOTOOLCHAIN': 'local', 'GOFLAGS': '-mod=readonly',
            'TEA_VERSION': lock['version'],
        }
        # Preserve upstream's version/SDK linker flags instead of inventing them.
        subprocess.run(['make', 'BUILDMODE=-buildvcs=false', 'build'], cwd=sources[0], env=env, check=True)
        (output / 'bin').mkdir()
        binary = output / 'bin/tea'
        shutil.copy2(sources[0] / 'tea', binary)
        binary.chmod(0o755)
        version = subprocess.run([str(binary), '--version'], check=True, capture_output=True, text=True)
        if not re.search(r'\b' + re.escape(lock['version']) + r'\b', version.stdout):
            raise SystemExit('built Tea version does not match its source lock')
    license_dir = output / 'licenses/tea'
    license_dir.mkdir(parents=True)
    shutil.copy2(root / 'project-os/licenses/tea-LICENSE', license_dir / 'LICENSE')
    print(f'Project Tea binary staged at {output}; no login or installation performed.')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--arch', choices=['x86_64', 'aarch64'], required=True)
    build(parser.parse_args().arch)

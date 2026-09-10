#!/usr/bin/env python3
"""Build a network-assisted stock CoreOS ISO; never boot, publish or install it.

Requires explicitly supplied native Butane/CoreOS Installer tools and trusted
Fedora signing inputs. Outputs are fresh and retained, including failed attempts.
"""
import argparse
import base64
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import re
import stat
import subprocess
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[1]
INSTALLER_VERSION = 'coreos-installer 0.26.0'
BINARY_PATH = '/var/usrlocal/libexec/soda/soda-install'
DATA_PATH = '/var/usrlocal/share/soda-installer'


def sha256(path):
    with Path(path).open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def payload_url(base, filename):
    u = urlsplit(base)
    if (u.scheme != 'https' or not u.hostname or u.username is not None
            or u.password is not None or u.query or u.fragment or '?' in base
            or '#' in base or any(ord(c) <= 32 for c in base)):
        raise ValueError('public credential-free HTTPS payload base URL required')
    return base.rstrip('/') + '/' + filename


def live_config(binary_url, binary_hash, destination, media, artwork):
    # Only the executable exceeds the upstream 256 KiB embed area. Fetch that
    # exact public output using Ignition's hash verification, not a shell downloader.
    if not re.fullmatch('[0-9a-f]{64}', binary_hash):
        raise ValueError('payload digest required')
    profile = """# Guidance only; no automatic disk action or credential collection.
case $- in
  *i*) if [ -t 1 ]; then
    printf '%s\\n' 'SodaOS installer: sudo /usr/local/libexec/soda/soda-install disk'
    printf '%s\\n' 'Fresh installation only. Disk erasure requires explicit confirmation.'
  fi ;;
esac
"""
    return {
        'variant': 'fcos', 'version': '1.6.0',
        # A supplied live Ignition disables CoreOS's default console autologin.
        # Run just the bounded UI on tty1, not an unauthenticated root shell.
        'systemd': {'units': [{'name': 'soda-installer-console.service', 'enabled': True,
            'contents': '[Unit]\nDescription=SodaOS installation console\n'
                        'After=systemd-user-sessions.service NetworkManager.service\nConflicts=getty@tty1.service\n'
                        '[Service]\nType=simple\n'
                        'ExecStart=/usr/local/libexec/soda/soda-install disk\n'
                        'StandardInput=tty-force\nStandardOutput=tty\nStandardError=tty\n'
                        'TTYPath=/dev/tty1\nTTYReset=yes\nTTYVHangup=yes\n'
                        'Restart=no\n[Install]\nWantedBy=multi-user.target\n'},
            {'name': 'getty@tty1.service', 'mask': True}]},
        'storage': {'files': [
            {'path': BINARY_PATH, 'mode': 0o755,
             'contents': {'source': binary_url,
                          'verification': {'hash': 'sha256-' + binary_hash}}},
            {'path': DATA_PATH + '/destination.ign', 'mode': 0o644,
             'contents': {'inline': json.dumps(destination)}},
            {'path': DATA_PATH + '/media.json', 'mode': 0o644,
             'contents': {'inline': json.dumps(media)}},
            {'path': '/etc/profile.d/soda-installer.sh', 'mode': 0o644,
             'contents': {'inline': profile}},
            {'path': '/etc/motd', 'mode': 0o644,
             'contents': {'inline': artwork + '\nSodaOS — CoreOS installation media\n'
                          'Run: sudo /usr/local/libexec/soda/soda-install disk\n'
                          'No disk is selected or erased automatically.\n'}},
        ]},
    }


def convert(butane, config, out):
    # No credentials reach Butane in this path. Still suppress converter
    # diagnostics: this is the same protection as the private provisioning caller.
    result = subprocess.run([str(butane), '--strict'],
                            input=json.dumps(config).encode(), stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, check=True, timeout=60)
    data = json.loads(result.stdout)
    if data.get('ignition', {}).get('version') != '3.5.0':
        raise ValueError('expected strict FCOS 1.6.0 -> Ignition 3.5.0 conversion')
    with out.open('xb') as f:
        f.write(result.stdout)
    return data


def output(args, cwd=ROOT, env=None):
    return subprocess.check_output(args, cwd=cwd, env=env, timeout=120).decode().strip()


def tool(path):
    path = Path(path)
    if not path.is_absolute() or not path.is_file() or not os.access(path, os.X_OK):
        raise ValueError('absolute existing native tool executable required')
    return path.resolve()


def build(args):
    if platform.system() != 'Linux' or platform.machine() != args.arch:
        raise ValueError('matching native Linux required')
    if output(['git', 'status', '--porcelain', '--untracked-files=normal']):
        raise ValueError('clean exact-revision source required')
    revision = output(['git', 'rev-parse', 'HEAD'])
    butane, installer = tool(args.butane), tool(args.coreos_installer)
    installer_version = output([str(installer), '--version'])
    if installer_version != INSTALLER_VERSION:
        raise ValueError('selected CoreOS Installer 0.26.0 required')
    butane_version = output([str(butane), '--version'])
    env = dict(os.environ, GOTOOLCHAIN='local', GOWORK='off', GOFLAGS='-mod=readonly',
               GOOS='linux', GOARCH={'x86_64': 'amd64', 'aarch64': 'arm64'}[args.arch],
               CGO_ENABLED='0')
    go_version = output(['go', 'env', 'GOVERSION'], env=env)
    if go_version != 'go1.26.7':
        raise ValueError('repository-pinned Go toolchain required')
    lock = ROOT / 'appliance/locks/coreos-iso.json'
    selected = json.loads(lock.read_text())
    filename = f'soda-install-{revision}-{args.arch}'
    url = payload_url(args.payload_base_url, filename)
    out = Path(args.out)
    artifact_root = ROOT / '.artifacts'
    if (not out.is_absolute() or out.parent.resolve() != out.parent
            or not out.is_relative_to(artifact_root) or not out.parent.is_dir()):
        raise ValueError('new absolute output below real existing .artifacts parent required')
    for parent in (out.parent, *out.parent.parents):
        st = parent.stat()
        if st.st_uid != os.getuid() or st.st_mode & 0o022:
            raise ValueError('output ancestry must be owned and not writable by others')
        if parent == ROOT:
            break
    out.mkdir(mode=0o700)  # exclusive; never clear previous attempts
    payload = out / 'payload'
    payload.mkdir()
    subprocess.run(['go', 'mod', 'verify'], cwd=ROOT, env=env, check=True)
    subprocess.run(['go', 'build', '-trimpath', '-buildvcs=true', '-ldflags=-s -w',
                    '-o', str(payload / filename), './appliance/installer'],
                   cwd=ROOT, env=env, check=True)
    subprocess.run(['go', 'build', '-trimpath', '-buildvcs=true',
                    '-o', str(out / 'soda-artifacts'), './tools/soda-artifacts'],
                   cwd=ROOT, env=env, check=True)
    subprocess.run([str(out / 'soda-artifacts'), 'fetch-coreos-iso', '--arch', args.arch,
                    '--lock', str(lock), '--keyring', str(Path(args.keyring).absolute()),
                    '--signer', args.signer, '--out', str(out / 'upstream')], check=True)
    # Reuse the production public bootstrap; all per-machine fields are collected
    # privately on the live host, never embedded in general-purpose output.
    import importlib.util
    spec = importlib.util.spec_from_file_location('soda_provisioning', ROOT / 'scripts/render-provisioning.py')
    provisioning = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(provisioning)
    destination = convert(butane, provisioning.public_config(), out / 'destination.ign')
    media = {'Architecture': args.arch, 'Release': selected['Release'],
             'InstallerVersion': installer_version, 'Revision': revision}
    artwork = (ROOT / 'assets/branding/terminal/sodaos.txt').read_text()
    for marker in ('$1', '$2', '$3'):
        artwork = artwork.replace(marker, '')
    config = live_config(url, sha256(payload / filename), destination, media, artwork)
    convert(butane, config, out / 'live.ign')
    customize = [str(installer), 'iso', 'customize', '--live-ignition', str(out / 'live.ign')]
    network = None
    if args.network_keyfile:
        network = snapshot_network(Path(args.network_keyfile), out)
        customize += ['--network-keyfile', str(network)]
    subprocess.run(customize + ['--output', str(out / 'soda.iso'),
                               str(out / 'upstream/coreos.iso')], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if network:
        extracted = out / 'network-readback'
        extracted.mkdir(mode=0o700)
        subprocess.run([str(installer), 'iso', 'network', 'extract', '--directory',
                        str(extracted), str(out / 'soda.iso')], check=True,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if list(extracted.iterdir()) != [extracted / network.name] or sha256(extracted / network.name) != sha256(network):
            raise ValueError('embedded network configuration differs from private input')
    # Read back the exact public live configuration. Optional private networking
    # is separately verified above. No destination-device/config, insecure kargs,
    # initramfs rebuild or OS source patch occurs.
    embedded = subprocess.check_output([str(installer), 'iso', 'ignition', 'show',
                                       str(out / 'soda.iso')], timeout=60,
                                      stderr=subprocess.DEVNULL)
    verify_embedded(embedded, json.loads((out / 'live.ign').read_bytes()))
    if output(['git', 'rev-parse', 'HEAD']) != revision or output(['git', 'status', '--porcelain', '--untracked-files=normal']):
        raise ValueError('source changed during media build; output is not sealed')
    for name in ('LICENSE', 'NOTICE'):
        (payload / name).write_bytes((ROOT / name).read_bytes())
    record = dict(media, PayloadURL=url, PayloadSHA256=sha256(payload / filename),
                  ISOSHA256=sha256(out / 'soda.iso'), LiveIgnitionSHA256=sha256(out / 'live.ign'),
                  PrivateMedia=network is not None,
                  NetworkKeyfileSHA256=sha256(network) if network else None,
                  CoreOSLockSHA256=sha256(lock), GoVersion=go_version,
                  ButaneVersion=butane_version,
                  ButaneSHA256=sha256(butane), CoreOSInstallerSHA256=sha256(installer))
    (out / 'media-build.json').write_text(json.dumps(record, indent=2) + '\n')
    (out / 'SHA256SUMS').write_text(''.join(f'{sha256(p)}  {p.relative_to(out)}\n' for p in
        (out / 'soda.iso', out / 'live.ign', out / 'destination.ign', payload / filename, out / 'media-build.json')))
    print(f'Media built, not booted or installed: {out}')
    if network:
        print('PRIVATE PER-MACHINE ISO: contains network configuration; do not publish or distribute as general media.')
    print(f'Before boot, serve the exact payload file at {url} through your existing trusted HTTPS hosting.')
    print('No hosting, publication, certificate changes, disk writes or validation VM were performed.')


def verify_embedded(raw, expected):
    # v0.26.0 LiveInitrd::live_config wraps each fragment in a gzip data-URL
    # merge. Admit that one exact wrapper, not arbitrary extra installer effects.
    wrapper = json.loads(raw)
    if (set(wrapper) != {'ignition'} or set(wrapper['ignition']) != {'version', 'config'}
            or wrapper['ignition']['version'] != '3.3.0'):
        raise ValueError('unexpected embedded Ignition effects')
    config = wrapper['ignition']['config']
    if set(config) != {'merge'} or len(config['merge']) != 1:
        raise ValueError('expected one embedded live fragment')
    resource = config['merge'][0]
    if set(resource) != {'source', 'compression'} or resource['compression'] != 'gzip' or not resource['source'].startswith('data:;base64,'):
        raise ValueError('unexpected live fragment reference')
    with gzip.GzipFile(fileobj=io.BytesIO(base64.b64decode(resource['source'].split(',', 1)[1], validate=True))) as source:
        decoded = source.read(4 * 1024 * 1024 + 1)
    if len(decoded) > 4 * 1024 * 1024 or json.loads(decoded) != expected:
        raise ValueError('ISO embedded Ignition differs from generated live config')


def snapshot_network(source, out):
    if not source.is_absolute() or source.parent.resolve() != source.parent:
        raise ValueError('absolute private NetworkManager keyfile required')
    with os.fdopen(os.open(source, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), 'rb') as f:
        st = os.fstat(f.fileno())
        if not stat.S_ISREG(st.st_mode) or st.st_mode & 0o077 or st.st_size > 65536:
            raise ValueError('bounded mode-0600 network keyfile required')
        data = f.read(65537)
    if len(data) > 65536:
        raise ValueError('network keyfile exceeds limit')
    dest = out / 'soda-installer.nmconnection'
    with dest.open('xb') as f:
        os.chmod(dest, 0o600)
        f.write(data)
    return dest


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--arch', choices=('x86_64', 'aarch64'), required=True)
    p.add_argument('--butane', required=True)
    p.add_argument('--coreos-installer', required=True)
    p.add_argument('--keyring', required=True)
    p.add_argument('--signer', required=True)
    p.add_argument('--payload-base-url', required=True)
    p.add_argument('--network-keyfile', help='optional private NetworkManager keyfile for pre-Ignition/static networking; makes the ISO private')
    p.add_argument('--out', required=True)
    args = p.parse_args()
    try:
        build(args)
    except (ValueError, OSError, subprocess.SubprocessError) as err:
        p.exit(1, f'Media build failed ({type(err).__name__}); retained outputs are not boot/install proof.\n')


if __name__ == '__main__':
    main()

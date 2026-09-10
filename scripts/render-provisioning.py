#!/usr/bin/env python3
"""Merge public Butane bootstrap with private per-instance operator inputs.
No conversion, installation, account enrollment or reboot is implicit.
"""
import argparse
import json
import os
from pathlib import Path
import re
import stat
import subprocess


def regular(path, private=False):
    p = Path(path)
    st = p.lstat()
    if not stat.S_ISREG(st.st_mode) or st.st_size > 1024 * 1024:
        raise ValueError('bounded regular input required')
    if private and st.st_mode & 0o077:
        raise ValueError('secret input must not be accessible to group/others')
    return p.read_text()


def branding_files():
    """Shared live/installed display identity; never copy a stale base version."""
    root = Path(__file__).resolve().parents[1]
    return [
        {'path': '/etc/os-release', 'mode': 0o644, 'overwrite': True,
         'contents': {'inline': (root / 'assets/branding/host/os-release').read_text()}},
        {'path': '/var/usrlocal/share/icons/hicolor/scalable/apps/sodaos-icon.svg', 'mode': 0o644,
         'contents': {'inline': (root / 'assets/branding/source/soda-symbol.svg').read_text()}},
    ]


def public_config():
    """One public bootstrap for private provisioning and installer-media conversion."""
    config = json.loads((Path(__file__).resolve().parents[1] / 'appliance/provisioning/base.json').read_text())
    config['storage']['files'].extend(branding_files())
    return config


def appliance_hostname(value):
    return (0 < len(value) <= 253 and all(re.fullmatch(
        r'[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?', label)
        for label in value.split('.')))


def render(operator_key, password_hash, out, hostname=None, host_key=None, bootstrap='extensions', product_hostname=None):
    key = regular(operator_key).strip()
    password = regular(password_hash, private=True).strip()
    if not key.startswith(('ssh-', 'ecdsa-', 'sk-')) or '\n' in key or not password.startswith('$') or '\n' in password:
        raise ValueError('provide a public SSH key and crypt(3) hash, not plaintext')
    x = public_config()
    if product_hostname is not None:
        if hostname is not None or not appliance_hostname(product_hostname):
            raise ValueError('valid appliance hostname or fixture hostname required, not both')
    if bootstrap == 'minimal':
        x.pop('systemd')
        x['storage']['files'] = []
    elif bootstrap != 'extensions':
        raise ValueError('unknown bootstrap profile')
    x['passwd'] = {'users': [{'name': 'root', 'ssh_authorized_keys': [key], 'password_hash': password}]}
    if hostname is not None:
        if not re.fullmatch(r'soda-native-[a-z0-9](?:[a-z0-9-]{0,40}[a-z0-9])?', hostname):
            raise ValueError('fresh soda-native-* fixture hostname required')
        x['storage']['files'].append({'path': '/etc/hostname', 'mode': 0o644, 'contents': {'inline': hostname + '\n'}})
    if product_hostname is not None:
        x['storage']['files'].append({'path': '/etc/hostname', 'mode': 0o644, 'contents': {'inline': product_hostname + '\n'}})
    if host_key is not None:
        private_key = regular(host_key, private=True)
        # Derive the public half without putting private contents in argv/logs.
        # Encrypted/wrong-type inputs fail without an interactive prompt.
        proc = subprocess.run(['ssh-keygen', '-y', '-P', '', '-f', str(Path(host_key).absolute())], stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=10)
        if proc.returncode or not proc.stdout.startswith('ssh-ed25519 '):
            raise ValueError('unencrypted per-instance Ed25519 host key required')
        for suffix, content, mode in [('', private_key, 0o600), ('.pub', proc.stdout, 0o644)]:
            x['storage']['files'].append({'path': '/etc/ssh/ssh_host_ed25519_key' + suffix, 'mode': mode, 'contents': {'inline': content}})
    dest = Path(out)
    if not dest.is_absolute() or dest.parent.resolve() != dest.parent:
        raise ValueError('absolute output under a real private parent required')
    if dest.parent.stat().st_mode & 0o077:
        raise ValueError('per-instance parent must be private')
    fd = os.open(dest, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as stream:
        json.dump(x, stream, indent=2)
        stream.write('\n')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--operator-key-file', required=True)
    p.add_argument('--root-password-hash-file', required=True)
    names = p.add_mutually_exclusive_group()
    names.add_argument('--hostname', help='fixture-only soda-native-* name')
    names.add_argument('--appliance-hostname', help='product hostname, default is unchanged CoreOS behavior')
    p.add_argument('--bootstrap', choices=('minimal', 'extensions'), default='extensions')
    p.add_argument('--ssh-host-key-file')
    p.add_argument('--out', required=True)
    a = p.parse_args()
    try:
        render(a.operator_key_file, a.root_password_hash_file, a.out, a.hostname, a.ssh_host_key_file, a.bootstrap, a.appliance_hostname)
    except (ValueError, OSError, subprocess.SubprocessError) as err:
        # No input values or subprocess diagnostics: either can contain secrets.
        p.exit(1, 'Private provisioning failed (' + type(err).__name__ + '); check paths/modes/key format.\n')

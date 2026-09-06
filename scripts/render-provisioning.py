#!/usr/bin/env python3
"""Produce private Butane input for a later explicitly authorized native install."""
import argparse, json, os
from pathlib import Path
p = argparse.ArgumentParser()
p.add_argument('--operator-key-file', required=True)
p.add_argument('--root-password-hash-file', required=True)
p.add_argument('--out', required=True)
a = p.parse_args()
key = Path(a.operator_key_file).read_text().strip()
password = Path(a.root_password_hash_file).read_text().strip()
if not key.startswith(('ssh-', 'ecdsa-', 'sk-')) or '\n' in key or not password.startswith('$') or '\n' in password:
    p.error('provide a native public SSH key and a crypt(3) password hash, not plaintext')
unit = '''[Unit]
Description=Provision native Soda appliance extensions
Wants=network-online.target
After=network-online.target
ConditionPathExists=!/var/lib/soda/extensions-requested
[Service]
Type=oneshot
StateDirectory=soda
ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive cockpit-system cockpit-ws cockpit-bridge cockpit-storaged cockpit-networkmanager cockpit-ostree tailscale forgejo-runner git nodejs python3 libicu openssl-libs krb5-libs zlib tar gzip
ExecStartPost=/usr/bin/touch /var/lib/soda/extensions-requested
RemainAfterExit=yes
[Install]
WantedBy=multi-user.target
'''
x = {'variant': 'fcos', 'version': '1.6.0', 'passwd': {'users': [{'name': 'root', 'ssh_authorized_keys': [key], 'password_hash': password}]}, 'storage': {'files': [{'path': '/etc/yum.repos.d/tailscale.repo', 'mode': 420, 'contents': {'source': 'https://pkgs.tailscale.com/stable/fedora/tailscale.repo'}}]}, 'systemd': {'units': [{'name': 'soda-extensions.service', 'enabled': True, 'contents': unit}]}}
f = os.open(a.out, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(f, 'w') as stream:
    json.dump(x, stream, indent=2)
    stream.write('\n')
# Compile this Butane input and perform the native install only in the later stage.

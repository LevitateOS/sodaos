#!/bin/bash
# Explicit first installation on an authorized native target, not an updater.
set -euo pipefail
bundle=${1:?usage: install-native.sh /path/to/native/ARCH PRIVATE_PROJECT_SUBNET}
subnet=${2:?Private routed IPv4 project subnet required}
[[ $(id -u) == 0 && $(uname -s) == Linux ]] || { echo 'Native host root required' >&2; exit 1; }
[[ $(basename "$bundle") == "$(uname -m)" ]] || { echo 'Bundle must match native architecture' >&2; exit 1; }
[[ ! -e /etc/soda/installed && ! -e /etc/soda/dashboard.json ]] || { echo 'Refusing blind reinstall over existing Soda state' >&2; exit 1; }
[[ -d "$bundle/rootfs" && -f "$bundle/images/project-os.oci" && -f "$bundle/images/dashboard.oci" ]] || exit 1
command -v python3 >/dev/null
command -v restorecon >/dev/null
command -v forgejo-runner >/dev/null
command -v tailscale >/dev/null
# Fail on a conflicting native service identity rather than adopting a person.
if getent passwd soda >/dev/null; then [[ $(id -u soda) == 2000 && $(id -g soda) == 2000 ]] || exit 1; fi
if getent passwd 2000 >/dev/null; then [[ $(getent passwd 2000 | cut -d: -f1) == soda ]] || exit 1; fi
# CoreOS has read-only /usr and /usr/local -> /var/usrlocal. Enter each
# writable destination before extracting rather than traversing that symlink
# inside an archive. Keep existing parent metadata and use root ownership,
# never the builder's UID/GID, for newly installed privileged files.
for prefix in etc usr/local var; do
  tar -C "$bundle/rootfs/$prefix" -cf - . | tar -C "/$prefix" -xf - \
    --no-same-owner --no-overwrite-dir
done
# Apply native SELinux labels only to delivered paths, before any service starts.
find "$bundle/rootfs" -mindepth 1 -printf '/%P\0' | xargs -0 -r restorecon -F
systemd-sysusers /etc/sysusers.d/soda.conf /etc/sysusers.d/soda-runners.conf
systemd-tmpfiles --create /etc/tmpfiles.d/soda.conf /etc/tmpfiles.d/soda-runners.conf
python3 - "$subnet" <<'PY'
import ipaddress, json, os, sys
from pathlib import Path
network = ipaddress.ip_network(sys.argv[1], strict=True)
if network.version != 4 or not network.is_private:
    sys.exit('private IPv4 project network required')
# Reserve a native containers subordinate range only on a non-conflicting host.
start, count = 1000000, 268435456
for name in ('/etc/subuid', '/etc/subgid'):
    p = Path(name)
    lines = p.read_text().splitlines() if p.exists() else []
    existing = False
    for line in lines:
        if not line or line.startswith('#'): continue
        owner, first, size = line.split(':')
        first, size = int(first), int(size)
        if owner == 'containers':
            if first != start or size != count: sys.exit(f'operator must resolve existing containers mapping in {name}')
            existing = True
        elif first < start + count and start < first + size:
            sys.exit(f'operator must resolve overlapping subordinate IDs in {name}')
    if not existing:
        with p.open('a') as f: f.write(f'containers:{start}:{count}\n')
root = Path('/etc/soda')
root.chmod(0o700)
p = root / 'host.json'
p.write_text(json.dumps({'image':'localhost/soda-project-os:dev','network':'soda-projects','subnet':str(network),'bridge':'soda0'},indent=2)+'\n')
p.chmod(0o600)
PY
chown -R 1000:1000 /var/lib/soda/forgejo/gitea/public
podman load -i "$bundle/images/project-os.oci"
podman load -i "$bundle/images/dashboard.oci"
systemctl daemon-reload
sysctl --system
systemctl enable --now soda-host.socket cockpit.socket tailscaled.service
systemctl start forgejo.service
touch /etc/soda/installed
printf 'Native installation requested. Complete operator Forgejo setup through the loopback tunnel, run soda-setup and soda-activate, and establish the private project-subnet route. No validation is implied.\n'

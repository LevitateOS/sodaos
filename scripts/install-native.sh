#!/bin/bash
# Explicit first installation on an authorized native target, not an updater.
# P04/P05 transport/identity boundary; core owns application setup/activation.
set -euo pipefail
podman() { command podman --remote=false "$@"; }
bundle=${1:?usage: install-native.sh /path/to/native/ARCH PRIVATE_PROJECT_SUBNET}
subnet=${2:?Private routed IPv4 project subnet required}
[[ $(id -u) == 0 && $(uname -s) == Linux ]] || { echo 'Native host root required' >&2; exit 1; }
[[ "$bundle" == /* && $(basename "$bundle") == "$(uname -m)" ]] || { echo 'Absolute bundle must match native architecture' >&2; exit 1; }
for state in /etc/soda/installed /etc/soda/install-started /etc/soda/dashboard.json /etc/soda/host.json; do
  [[ ! -e "$state" && ! -L "$state" ]] || { echo 'Refusing blind reinstall over existing or partial Soda state' >&2; exit 1; }
done
. /etc/os-release
[[ "$ID" == fedora && ${VARIANT_ID:-} == coreos ]] || { echo 'Upstream Fedora CoreOS target required, not the builder' >&2; exit 1; }
for command in python3 restorecon matchpathcon rpm-ostree podman ip systemd-sysusers systemd-tmpfiles sysctl; do command -v "$command" >/dev/null; done
rpm -q cockpit-system cockpit-ws cockpit-bridge cockpit-storaged cockpit-networkmanager cockpit-ostree tailscale forgejo-runner git python3 libicu openssl-libs krb5-libs tar gzip >/dev/null
# Fedora may satisfy these capabilities with versioned/replacement packages.
rpm -q --whatprovides nodejs zlib >/dev/null
[[ $(getenforce) == Enforcing ]] || { echo 'Host SELinux must remain enforcing' >&2; exit 1; }
# Verify using the bundle's matching-native support tool; it is not installed.
# The operator must establish the external SHA256SUMS identity before invoking
# any bundled executable. This is integrity, not a signature/trust bootstrap.
revision=$(python3 - "$bundle/build-info.json" <<'PY'
import json, re, sys
x = json.load(open(sys.argv[1]))
if not re.fullmatch('[0-9a-f]{40}', x['Revision']): raise SystemExit('invalid revision')
print(x['Revision'])
PY
)
"$bundle/tools/soda-artifacts" verify --source "$bundle" --arch "$(uname -m)" --revision "$revision"
# Fail on a conflicting service identity rather than adopting a person.
if getent passwd soda >/dev/null; then
  [[ $(id -u soda) == 2000 && $(id -g soda) == 2000 ]]
  [[ $(getent passwd soda | cut -d: -f6-7) == /var/lib/soda/dashboard:/usr/sbin/nologin ]]
fi
if getent passwd 2000 >/dev/null; then [[ $(getent passwd 2000 | cut -d: -f1) == soda ]]; fi
if getent group soda >/dev/null; then [[ $(getent group soda | cut -d: -f3) == 2000 ]]; fi
if getent group 2000 >/dev/null; then [[ $(getent group 2000 | cut -d: -f1) == soda ]]; fi
containers=$(podman ps -a --format '{{.Names}}')
if grep -Eq '^(soda-forgejo|soda-dashboard|soda-proxy|p[0-9a-f]{24})$' <<<"$containers"; then
  echo 'Existing appliance/project containers require an operator decision; not a first install' >&2; exit 1
fi
configure_network() {
python3 - "$subnet" "$1" <<'PY'
import ipaddress, json, subprocess, sys
from pathlib import Path
network = ipaddress.ip_network(sys.argv[1], strict=True)
private = [ipaddress.ip_network(x) for x in ('10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16')]
if network.version != 4 or not any(network.subnet_of(x) for x in private):
    sys.exit('RFC1918 IPv4 project network required')
start, count = 1000000, 268435456
updates = []
for name in ('/etc/subuid', '/etc/subgid'):
    p = Path(name)
    if p.is_symlink(): sys.exit('symlinked subordinate mapping refused')
    lines = p.read_text().splitlines() if p.exists() else []
    existing = False
    for line in lines:
        if not line or line.startswith('#'): continue
        owner, first, size = line.split(':')
        first, size = int(first), int(size)
        if owner == 'containers':
            if existing or first != start or size != count: sys.exit(f'operator must resolve containers mapping in {name}')
            existing = True
        elif first < start + count and start < first + size:
            sys.exit(f'operator must resolve overlapping subordinate IDs in {name}')
    if not existing: updates.append(p)
if sys.argv[2] == 'check':
    routes = json.loads(subprocess.check_output(['ip', '-json', '-4', 'route', 'show', 'table', 'all']))
    for route in routes:
        dst = route.get('dst', 'default')
        if dst == 'default': continue
        other = ipaddress.ip_network(dst, strict=False)
        if other.prefixlen and network.overlaps(other):
            sys.exit('project subnet overlaps an existing host route')
    links = json.loads(subprocess.check_output(['ip', '-json', 'link', 'show']))
    if any(link.get('ifname') == 'soda0' for link in links):
        sys.exit('existing soda0 requires an operator decision')
    names = subprocess.check_output(['podman', '--remote=false', 'network', 'ls', '--format', '{{.Name}}'], text=True).splitlines()
    if 'soda-projects' in names: sys.exit('existing soda-projects network refused')
    if names:
        networks = json.loads(subprocess.check_output(['podman', '--remote=false', 'network', 'inspect', *names]))
        for item in networks:
            for entry in item.get('subnets') or []:
                other = ipaddress.ip_network(entry['subnet'])
                if other.version == 4 and network.overlaps(other): sys.exit('project subnet overlaps an existing container network')
    sys.exit(0)
for p in updates:
    with p.open('a') as f: f.write(f'containers:{start}:{count}\n')
root = Path('/etc/soda')
root.chmod(0o700)
p = root / 'host.json'
with p.open('x') as f:
    json.dump({'image':'localhost/soda-project-os:dev','network':'soda-projects','subnet':str(network),'bridge':'soda0'}, f, indent=2)
    f.write('\n')
p.chmod(0o600)
PY
}
# All read-only platform/bundle/network/identity preflight precedes host writes.
configure_network check
# Inspect existing destination ancestors before the first installation write.
# The one supported CoreOS link is /usr/local -> /var/usrlocal.
python3 - "$bundle/build-info.json" <<'PY'
import json, stat, sys
from pathlib import Path
def check_destinations(root, files):
    # Fixed first-install destinations, not a custom-template merge/adoption policy.
    hooks = {'templates/custom/header.tmpl', 'templates/custom/footer.tmpl',
             'public/assets/sodaspaces.css', 'public/assets/sodaspaces.js',
             'public/assets/sodaspaces-terminal.js', 'public/assets/sodaspaces-terminal.css',
             'public/assets/sodaspaces-drawer.js', 'public/assets/sodaspaces-drawer.css',
             'public/assets/soda-terminal/xterm.mjs', 'public/assets/soda-terminal/xterm.css',
             'public/assets/soda-terminal/addon-fit.mjs', 'public/assets/soda-terminal/xterm.LICENSE',
             'public/assets/soda-terminal/fit.LICENSE'}
    protected = {'rootfs/var/lib/soda/forgejo/gitea/' + name for name in hooks}
    for name in files:
        if not name.startswith('rootfs/'): continue
        parts = Path(name.removeprefix('rootfs/')).parts
        target = root.joinpath(*parts)
        for path in [root.joinpath(*parts[:i]) for i in range(len(parts) + 1)]:
            if path.is_symlink():
                if path == root / 'usr/local' and path.resolve() == root / 'var/usrlocal':
                    continue
                raise SystemExit('unexpected symlink in installation destination')
            if path.exists() and path != target:
                info = path.stat()
                if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
                    raise SystemExit('unsafe installation destination ancestor')
        if name in protected and target.exists():
            raise SystemExit('occupied Sodaspaces hook/asset; operator decision required')
check_destinations(Path('/'), json.loads(Path(sys.argv[1]).read_text())['Files'])
PY
rpm-ostree status --json | python3 -c 'import json,sys; deployments=json.load(sys.stdin)["deployments"]; assert sum(d.get("booted") is True for d in deployments)==1, "one observed booted deployment required"'
umask 077
install -d -m 0700 /etc/soda
(set -o noclobber; printf '%s\n' "$revision" > /etc/soda/install-started)
# CoreOS /usr/local -> /var/usrlocal: enter writable prefixes before extraction.
# Preserve existing parent metadata, never builder ownership.
for prefix in etc usr/local var; do
  tar -C "$bundle/rootfs/$prefix" -cf - . | tar -C "/$prefix" -xf - \
    --no-same-owner --no-overwrite-dir
done
find "$bundle/rootfs" -mindepth 1 -printf '/%P\0' | xargs -0 -r restorecon -F
systemd-sysusers /etc/sysusers.d/soda.conf /etc/sysusers.d/soda-runners.conf
systemd-tmpfiles --create /etc/tmpfiles.d/soda.conf /etc/tmpfiles.d/soda-runners.conf
configure_network apply
chown -R 1000:1000 /var/lib/soda/forgejo/gitea/public
# Only the new exact template paths; never recursively adopt mutable Forgejo data.
chown 1000:1000 /var/lib/soda/forgejo/gitea/templates{,/custom,/custom/header.tmpl,/custom/footer.tmpl}
# OCI image-ID saves need not preserve tag annotations. Restore precisely the
# existing core references, from the verified config identity, without repulling.
for image in project-os dashboard forgejo caddy; do
  podman load -i "$bundle/images/$image.oci"
  id=$(python3 - "$bundle/build-info.json" "$image" <<'PY'
import json, sys
print(json.load(open(sys.argv[1]))['Images'][sys.argv[2]]['Config'])
PY
)
  case "$image" in
    project-os) reference=localhost/soda-project-os:dev;;
    dashboard) reference=localhost/soda-dashboard:dev;;
    forgejo) reference=$(awk -F= '$1=="Image" {print $2}' "$bundle/rootfs/etc/containers/systemd/forgejo.container");;
    caddy) reference=$(awk -F= '$1=="Image" {print $2}' "$bundle/rootfs/etc/containers/systemd/soda-proxy.container");;
  esac
  [[ -n "$reference" ]]
  podman tag "$id" "$reference"
done
restorecon -F /etc/soda /etc/soda/host.json /etc/soda/install-started
systemctl daemon-reload
sysctl --system
systemctl enable --now soda-host.socket cockpit.socket tailscaled.service
systemctl start forgejo.service
touch /etc/soda/installed
restorecon -F /etc/soda/installed
printf 'Native installation requested. Complete core-owned Forgejo setup, soda-setup and soda-activate, and establish the authorized project route. No product validation is implied.\n'

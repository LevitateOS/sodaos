#!/bin/bash
# Read-only first-install checks; no account creation, enrollment or restart.
set -euo pipefail
podman() { command podman --remote=false "$@"; }
[[ ${SODA_NATIVE_VALIDATE:?Set to the explicitly selected host name} == "$(hostname)" ]] || {
  echo 'SODA_NATIVE_VALIDATE must match this host name' >&2; exit 1;
}
[[ $(id -u) == 0 && -f /etc/soda/installed ]]
[[ $(getenforce) == Enforcing ]]
if [[ -n ${SODA_BUNDLE:-} ]]; then
  "$SODA_BUNDLE/tools/soda-artifacts" verify-installed --source "$SODA_BUNDLE" \
    --arch "$(uname -m)" --revision "${SODA_REVISION:?Selected bundle revision required}"
else
  printf 'Artifact identity not checked: substrate facts only; supply SODA_BUNDLE/SODA_REVISION for candidate binding.\n'
fi
. /etc/os-release
[[ "$ID" == fedora && ${VARIANT_ID:-} == coreos ]]
rpm -q cockpit-system cockpit-ws cockpit-bridge cockpit-storaged cockpit-networkmanager cockpit-ostree tailscale forgejo-runner git python3 tar gzip
# Fedora may satisfy these capabilities with versioned/replacement packages.
rpm -q --whatprovides nodejs
rpm-ostree status --json | python3 -c 'import json,sys; x=json.load(sys.stdin); print(json.dumps([{k:d.get(k) for k in ("booted","version","checksum","requested-packages")} for d in x["deployments"]]))'
printf 'Actual substrate: %s %s; host %s\n' "$(uname -s)" "$(uname -m)" "$(hostname)"
for unit in soda-host.socket forgejo.service cockpit.socket tailscaled.service; do
  systemctl is-active --quiet "$unit"
done
for directory in /etc /var /var/lib; do
  [[ $(stat -c '%u:%g:%a' "$directory") == 0:0:755 ]]
done
for command in soda-dashboard soda-host soda-setup soda-forgejo-tailnet soda-runners soda-runner-launch soda-tailnet; do
  file="/usr/local/libexec/soda/$command"
  [[ $(stat -c '%u:%g:%a' "$file") == 0:0:755 ]]
  matchpathcon -V "$file"
done
[[ -S /run/soda/host.sock ]]
[[ $(stat -c '%u:%g:%a' /run/soda/host.sock) == 0:2000:660 ]]
[[ $(stat -c '%u:%g:%a' /var/lib/soda/dashboard) == 2000:2000:700 ]]
for file in /etc/cockpit/cockpit.conf /etc/pam.d/cockpit; do
  [[ $(stat -c '%u:%g:%a' "$file") == 0:0:644 ]]
  matchpathcon -V "$file"
done
[[ ! -e /usr/local/share/cockpit/soda-runners && ! -L /usr/local/share/cockpit/soda-runners ]]
for page in soda-tailscale; do
  [[ -r /usr/local/share/cockpit/$page/index.html ]]
  [[ -r /usr/local/share/cockpit/$page/manifest.json ]]
done
phase=pre-activation
if [[ -e /etc/soda/activated ]]; then
  phase=activated
  for unit in soda-dashboard.service soda-proxy.service; do systemctl is-active --quiet "$unit"; done
  pid=$(podman inspect --format '{{.State.Pid}}' soda-dashboard)
  [[ "$pid" =~ ^[1-9][0-9]*$ && -r /proc/$pid/status ]]
  awk '/^Uid:|^Gid:/ { ids++; if ($2 != 2000 || $3 != 2000 || $4 != 2000 || $5 != 2000) bad=1 } /^CapEff:|^CapPrm:|^CapBnd:/ { caps++; if ($2 !~ /^0+$/) bad=1 } /^NoNewPrivs:/ { privilege++; if ($2 != 1) bad=1 } END { if (bad || ids!=2 || caps!=3 || privilege!=1) exit 1 }' "/proc/$pid/status"
  [[ $(podman inspect --format '{{.HostConfig.ReadonlyRootfs}}' soda-dashboard) == true ]]
  [[ $(stat -c '%u:%g:%a' /etc/soda/dashboard.json) == 0:2000:640 ]]
else
  [[ $(podman port soda-forgejo 3000/tcp) == 127.0.0.1:3000 ]]
  [[ $(podman port soda-forgejo 22/tcp) == 127.0.0.1:2222 ]]
fi
[[ ${SODA_HOST_PHASE:-$phase} == "$phase" ]] || { echo 'Observed activation phase differs from requested phase' >&2; exit 1; }
printf 'Observed phase: %s. Native listeners (not routed-client proof):\n' "$phase"
ss -lnt
python3 - "$phase" <<'PY'
import ipaddress, json, socket, stat, subprocess, sys
from pathlib import Path
from urllib.parse import urlsplit
rows = subprocess.check_output(['ss', '-H', '-ltn'], text=True).splitlines()
listeners = set()
for row in rows:
    fields = row.split()
    if len(fields) < 5: raise SystemExit('malformed listener observation')
    address, port = fields[3].rsplit(':', 1)
    listeners.add((address.strip('[]'), int(port)))
def bound(address, port):
    if (address, port) not in listeners: raise SystemExit('required service binding missing')
    if any(p == port and a not in (address, '::1' if address == '127.0.0.1' else address) for a, p in listeners):
        raise SystemExit('unexpected additional service binding')
bound('127.0.0.1', 9090)
def published(address, host_port, container_port):
    mapping = subprocess.check_output(['podman', '--remote=false', 'port', 'soda-forgejo', f'{container_port}/tcp'], text=True).strip()
    expected = f'[{address}]:{host_port}' if ':' in address else f'{address}:{host_port}'
    if mapping != expected: raise SystemExit('unexpected published Git/HTTP mapping')
    # Rootful bridge publication may use DNAT, not a host listening process.
    if any(p == host_port and a != address for a, p in listeners): raise SystemExit('unexpected host listener on published port')
    with socket.create_connection((address, host_port), timeout=5): pass
published('127.0.0.1', 3000, 3000)
if sys.argv[1] == 'activated':
    cfg = json.loads(Path('/etc/soda/dashboard.json').read_text())
    # Retired bootstrap files are not dashboard credentials and need not exist.
    for key in ('oauth_secret_file', 'grant_key_file'):
        path = Path(cfg[key])
        info = path.lstat()
        if not path.is_absolute() or not stat.S_ISREG(info.st_mode) or (info.st_uid, info.st_gid, stat.S_IMODE(info.st_mode)) != (0, 2000, 0o640):
            raise SystemExit('configured dashboard credential permissions invalid')
    for name in ('cert.pem', 'key.pem'):
        info = (Path('/etc/soda/tls') / name).lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or stat.S_IMODE(info.st_mode) != 0o600:
            raise SystemExit('TLS material permissions invalid')
    # These are the variables actually consumed by the packaged Caddy config,
    # not ports reconstructed from a client's unrelated browser tunnel.
    env = dict(line.split('=', 1) for line in Path('/etc/soda/proxy.env').read_text().splitlines() if line)
    address = str(ipaddress.ip_address(env['SODA_BIND']))
    if not ipaddress.ip_address(address).is_private or ipaddress.ip_address(address).is_unspecified:
        raise SystemExit('private explicit proxy bind required')
    if 'public_url' in cfg or 'SODA_ORIGIN' in env:
        raise SystemExit('legacy separate-origin configuration requires rehearsed maintenance')
    if env['FORGEJO_ORIGIN'].rstrip('/') != cfg['forgejo_url'].rstrip('/'):
        raise SystemExit('proxy and API browser origin mismatch')
    for key in ('FORGEJO_ORIGIN',):
        origin = urlsplit(env[key])
        if origin.scheme != 'https' or not origin.hostname: raise SystemExit('invalid configured HTTPS origin')
        bound(address, origin.port or 443)
    host, port = cfg['listen'].rsplit(':', 1)
    bound(host.strip('[]'), int(port))
    published(address, 2222, 22)
else:
    published('127.0.0.1', 2222, 22)
print('Configured native listener and credential-mode boundaries observed; not a login or client-route proof.')
PY
printf 'First-install services, ownership, labels and page files checked on %s. Dashboard/project/provider journeys remain separate.\n' "$(hostname)"

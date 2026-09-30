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
phase=pre-activation
if [[ -e /etc/soda/activated ]]; then phase=activated; fi
python3 - "$phase" <<'PY'
import hashlib, json, platform, subprocess, sys
from pathlib import Path

payload = json.loads(Path('/usr/share/soda/release.json').read_text())
content = json.loads(Path('/usr/share/soda/host-image/content.json').read_text())
fixed = {
    'dashboard:/usr/local/bin/soda-dashboard',
    'forgejo:/usr/local/bin/gitea',
    'extension:/usr/local/bin/gitea',
    'extension:/usr/share/soda/extension/extension.json',
    'extension:/usr/share/soda/extension/backend',
    'extension:/usr/share/soda/extension/run',
    'host:/usr/share/containers/systemd/forgejo.container',
    'host:/usr/share/containers/systemd/soda-dashboard.container',
    'host:/usr/lib/systemd/system/soda-extension-install.service',
}
asset_prefix = 'extension:/usr/share/soda/extension/assets/'
assets = {name for name in content if name.startswith(asset_prefix)}
if not assets or set(content) != fixed | assets:
    raise SystemExit('incomplete installed content inventory')
for name in assets:
    relative = name.removeprefix(asset_prefix)
    if not relative or '..' in Path(relative).parts or Path(relative).is_absolute():
        raise SystemExit('unsafe installed extension asset inventory')
if payload['Architecture'] != platform.machine():
    raise SystemExit('installed release architecture differs from native host')
packages = Path('/usr/share/soda/host-image/packages.txt').read_bytes()
if hashlib.sha256(packages).hexdigest() != payload['HostPackagesSHA256']:
    raise SystemExit('installed RPM inventory differs from release metadata')
images = payload['Images']
if images['forgejo']['Config'] == images['extension']['Config']:
    raise SystemExit('independent extension image required')
if content['forgejo:/usr/local/bin/gitea'] != content['extension:/usr/local/bin/gitea']:
    raise SystemExit('extension installer CLI differs from patched Forgejo')
for name in ('dashboard', 'forgejo', 'extension'):
    image = images[name]['Config']
    observed = subprocess.check_output(['podman', '--remote=false', 'image', 'inspect', '--format', '{{.Id}}', image], text=True).strip()
    if observed != image:
        raise SystemExit(f'{name} image identity differs from installed release')
for name, expected in content.items():
    component, path = name.split(':', 1)
    if component == 'host':
        actual = hashlib.sha256(Path(path).read_bytes()).hexdigest()
    else:
        image = images[component]['Config']
        output = subprocess.check_output(['podman', '--remote=false', 'run', '--rm', '--network=none', '--read-only', '--cap-drop=all', '--security-opt=no-new-privileges', '--entrypoint=/usr/bin/sha256sum', image, path], text=True).strip()
        actual = output.split('  ', 1)[0] if output.endswith('  ' + path) else ''
    if actual != expected:
        raise SystemExit(f'installed content differs from release inventory: {name}')
if sys.argv[1] == 'activated':
    package_prefix = 'extension:/usr/share/soda/extension/'
    package_root = Path('/var/lib/soda/forgejo/gitea/extensions/soda')
    expected_package = {
        name.removeprefix(package_prefix): digest
        for name, digest in content.items()
        if name.startswith(package_prefix)
    }
    if not package_root.is_dir() or package_root.is_symlink():
        raise SystemExit('installed Soda extension package is not a regular directory')
    observed_package = set()
    for path in package_root.rglob('*'):
        if path.is_symlink():
            raise SystemExit('installed Soda extension package contains a symlink')
        if path.is_file():
            relative = path.relative_to(package_root).as_posix()
            if relative != '.disabled':
                observed_package.add(relative)
    if observed_package != set(expected_package):
        raise SystemExit('installed Soda extension package differs from candidate inventory')
    for relative, expected in expected_package.items():
        actual = hashlib.sha256((package_root / relative).read_bytes()).hexdigest()
        if actual != expected:
            raise SystemExit(f'installed package replacement differs from candidate inventory: {relative}')
    checked = 'installed package replacement'
else:
    checked = 'extension image package'
print(f'Installed fork, {checked}, Soda service bytes and native architecture match the release inventory.')
PY
rpm -q cockpit-system cockpit-ws cockpit-bridge cockpit-storaged cockpit-networkmanager cockpit-ostree tailscale git python3 tar gzip
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
for command in soda-dashboard soda-host soda-setup soda-forgejo-tailnet soda-runners soda-tailnet; do
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
if [[ "$phase" == activated ]]; then
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

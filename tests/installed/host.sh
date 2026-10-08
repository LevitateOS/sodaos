#!/bin/bash
# Read-only first-install checks; no account creation, enrollment or restart.
set -euo pipefail
podman() { command podman --remote=false "$@"; }
[[ ${SODA_NATIVE_VALIDATE:?Set to the explicitly selected host name} == "$(hostname)" ]] || {
  echo 'SODA_NATIVE_VALIDATE must match this host name' >&2; exit 1;
}
[[ $(id -u) == 0 && -f /etc/soda/installed ]]
[[ $(getenforce) == Enforcing ]]
. /etc/os-release
[[ "$ID" == fedora && ${VARIANT_ID:-} == coreos ]]
phase=pre-activation
if [[ -e /etc/soda/activated ]]; then phase=activated; fi
/usr/libexec/soda/soda-host-probes host-content "$phase"
rpm -q cockpit-system cockpit-ws cockpit-bridge cockpit-storaged cockpit-networkmanager cockpit-ostree tailscale git tar gzip
# Fedora may satisfy these capabilities with versioned/replacement packages.
rpm -q --whatprovides nodejs
rpm-ostree status --json | /usr/libexec/soda/soda-host-probes host-deployments
printf 'Actual substrate: %s %s; host %s\n' "$(uname -s)" "$(uname -m)" "$(hostname)"
for unit in soda-host.socket forgejo.service cockpit.socket tailscaled.service; do
  systemctl is-active --quiet "$unit"
done
for directory in /etc /var /var/lib; do
  [[ $(stat -c '%u:%g:%a' "$directory") == 0:0:755 ]]
done
for command in soda-dashboard soda-host soda-setup soda-forgejo-tailnet soda-tailnet; do
  file="/usr/libexec/soda/$command"
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
/usr/libexec/soda/soda-host-probes host-listeners "$phase"
printf 'First-install services, ownership, labels and retained Cockpit configuration checked on %s. Dashboard/project/provider journeys remain separate.\n' "$(hostname)"

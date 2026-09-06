#!/bin/bash
# Read-only first-install checks; no account creation, enrollment or restart.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Set to the explicitly selected host name} == "$(hostname)" ]] || {
  echo 'SODA_NATIVE_VALIDATE must match this host name' >&2; exit 1;
}
[[ $(id -u) == 0 && -f /etc/soda/installed ]]
for unit in soda-host.socket forgejo.service cockpit.socket tailscaled.service; do
  systemctl is-active --quiet "$unit"
done
for directory in /etc /var /var/lib; do
  [[ $(stat -c '%u:%g:%a' "$directory") == 0:0:755 ]]
done
for command in soda-dashboard soda-host soda-setup soda-forgejo-tailnet soda-runners soda-runner-helper soda-runner-launch soda-tailnet; do
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
for page in soda-runners soda-tailscale; do
  [[ -r /usr/local/share/cockpit/$page/index.html ]]
  [[ -r /usr/local/share/cockpit/$page/manifest.json ]]
done
printf 'First-install services, ownership, labels and page files checked on %s. Dashboard/project/provider journeys remain separate.\n' "$(hostname)"

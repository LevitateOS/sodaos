#!/bin/bash
# P06. Adapted from soda-os bc1d3e0 check-native-service-ordering.sh.
# Current generated units only; no cloud-init/old Forgejo bootstrap assumptions.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Explicit host required} == "$(hostname)" && $(id -u) == 0 ]]
contains_property() {
  local actual
  actual=$(systemctl show "$1" --property="$2" --value)
  [[ " $actual " == *" $3 "* ]] || { printf 'Missing %s=%s on %s\n' "$2" "$3" "$1" >&2; return 1; }
}
contains_property soda-host.service Requires soda-host.socket
contains_property soda-dashboard.service Requires soda-host.socket
contains_property soda-dashboard.service After forgejo.service
contains_property soda-dashboard.service After soda-host.socket
contains_property soda-proxy.service After soda-dashboard.service
contains_property soda-proxy.service After forgejo.service
contains_property forgejo.service After network-online.target
for unit in forgejo.service soda-dashboard.service soda-proxy.service; do
  path=$(systemctl show "$unit" --property=FragmentPath --value)
  [[ "$path" == /run/systemd/generator*/* && -f "$path" ]]
  [[ $(systemctl show "$unit" --property=LoadState --value) == loaded ]]
done
[[ $(systemctl show soda-host.service --property=User --value) == root ]]
[[ $(systemctl show soda-host.service --property=Group --value) == root ]]
failed=$(systemctl --failed --no-legend --plain --no-pager)
[[ -z "$failed" ]] || { printf 'Failed units remain:\n%s\n' "$failed" >&2; exit 1; }
printf 'Current native unit ordering/properties observed; no service was started or restarted.\n'

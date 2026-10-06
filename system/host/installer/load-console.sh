#!/bin/bash
# Live-media launcher only. No downloads, destination-disk writes or repair.
set -euo pipefail
umask 077
expected=${1:?expected console SHA-256 required}
[[ "$expected" =~ ^[0-9a-f]{64}$ ]]
[[ $(id -u) == 0 && -f /run/ostree-live ]]
mountpoint -q /run/media/iso
source=/run/media/iso/soda/soda-install
destination=/usr/local/libexec/soda/soda-install
staging=${destination}.incoming
[[ -f "$source" && ! -L "$source" && ! -e "$destination" && ! -L "$destination" ]]
[[ ! -e "$staging" && ! -L "$staging" ]]
# The ordinary ISO mount is read-only. Verify the copied bytes, not a later
# reopening of the source. Failed copies remain non-executable for inspection.
install -D -m 0600 "$source" "$staging"
printf '%s  %s\n' "$expected" "$staging" | sha256sum --check --status
# Publish without replacing an existing destination, then label that exact path.
ln -- "$staging" "$destination"
restorecon -F "$destination"
chmod 0700 "$destination"
rm -- "$staging" # only this invocation's successfully verified staging link
exec "$destination" disk

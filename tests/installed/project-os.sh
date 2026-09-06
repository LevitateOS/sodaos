#!/bin/sh
# Run later inside an explicitly selected disposable project as its administrator.
set -eu
: "${SODA_NATIVE_VALIDATE:?Set only on the authorized native validation target}"
test "$(id -u)" != 0
command -v mise
command -v git
command -v tea
command -v gh
tea --version
gh --version
# CLI availability is not login or provider compatibility evidence.
test -d "$HOME/shared"
test -r "/etc/ssh/authorized_keys/$(id -un)"
test "$(readlink "$HOME/shared")" = /srv/project/shared
printf 'Inspect sudo -l and verify ordinary SSH/SCP/SFTP from the developer client.\n'

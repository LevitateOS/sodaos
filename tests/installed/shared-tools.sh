#!/bin/sh
# Later, on the developer client with TWO accounts on the same native project.
set -eu
: "${SODA_NATIVE_VALIDATE:?Explicit native validation authorization required}"
: "${PROJECT_IP:?}" "${ALICE:?}" "${BOB:?}"
# Operator first installs node@24 from a project-local root login with umask 022.
a=$(ssh "$ALICE@$PROJECT_IP" 'mise where node')
b=$(ssh "$BOB@$PROJECT_IP" 'mise where node')
test "$a" = "$b"
case "$a" in /opt/mise/installs/*) ;; *) exit 1;; esac
ssh "$ALICE@$PROJECT_IP" 'node --version'
ssh "$BOB@$PROJECT_IP" 'node --version'
# Neither command should download/install anything; inspect the native install
# tree and permissions before declaring the shared-installed-tool outcome proved.

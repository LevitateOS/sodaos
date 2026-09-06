#!/bin/bash
# Explicit later source-test execution; never installs, registers or reboots.
set -euo pipefail
arch=${1:?usage: check-native.sh x86_64|aarch64}
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64|aarch64) ;; *) exit 2;; esac
cd "$(dirname "$0")/.."
go test -mod=readonly ./...
(cd cockpit && pnpm exec tsc --noEmit && pnpm exec vp test --run)
python3 -m unittest discover -s tests/build
SODA_STAGE="$PWD/.artifacts/native/$arch/rootfs" python3 -m unittest discover -s tests/packaging
printf 'Source/staging checks executed; this is not installed appliance validation.\n'

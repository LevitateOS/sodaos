#!/bin/bash
# Local source checks; no sealed stage, appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux
# The checkout pins its toolchain in go.mod (Linux-only files such as the
# linux-tagged candidate tests are silently skipped otherwise); refuse to
# check with anything else before the first real command runs.
pinned="$(grep '^go ' go.mod | awk '{print $2}')"
[[ -n "$pinned" ]] || { echo 'go.mod pins no Go version' >&2; exit 1; }
# Fail with the toolchain's own status so a broken `go` still stops the suites.
got_version="$(go version)" || exit $?
[[ "$got_version" == "go version go$pinned "* ]] || { echo "pinned Go $pinned required" >&2; exit 1; }
go mod verify
go test -mod=readonly ./...
bash scripts/check-sql-locality.sh
bun run typecheck
bun run test
python3 -m unittest discover -s tests/build
printf 'Local source checks executed; optional installed/provider gates were not enabled by this command.\n'

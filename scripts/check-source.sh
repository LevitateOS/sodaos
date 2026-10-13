#!/bin/bash
# Local source checks; no sealed stage, appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."
suite=${1:-all}
if (( $# > 1 )) || [[ "$suite" != all && "$suite" != go && "$suite" != browser ]]; then
  echo 'usage: check-source.sh [all|go|browser]' >&2
  exit 2
fi
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux
failed=0
check() {
  if "$@"; then return 0; else
    local status=$?
    (( failed != 0 )) || failed=$status
  fi
}
# Independent suites report their failures in the same run. Prepared browser
# checks still require a successful asset build and every failure stays nonzero.
if [[ "$suite" != browser ]]; then
  # Linux-only tests require the exact go.mod toolchain. Admission failures
  # stop before running the Go suite; the browser suite needs no Go compiler.
  pinned="$(grep '^go ' go.mod | awk '{print $2}')"
  [[ -n "$pinned" ]] || { echo 'go.mod pins no Go version' >&2; exit 1; }
  got_version="$(go version)" || exit $?
  [[ "$got_version" == "go version go$pinned "* ]] || { echo "pinned Go $pinned required" >&2; exit 1; }
  go mod verify
  check go test -mod=readonly ./...
  check bash scripts/check-sql-locality.sh
  check bash scripts/check-no-npm.sh
  check bash scripts/check-no-python.sh
fi
if [[ "$suite" != go ]]; then
  check bun run typecheck
  if bun run build:forgejo; then
    check bun run test:frontend:prepared
    check bun run test:forgejo:prepared
  else
    status=$?
    (( failed != 0 )) || failed=$status
  fi
fi
printf 'Local source checks executed; optional installed/provider gates were not enabled by this command.\n'
exit "$failed"

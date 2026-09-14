#!/bin/bash
# staticcheck gate for Go code. Analyzes the linux build (GOOS=linux): several
# installer paths are linux-only and drop out under darwin, which buries real
# findings under false undefined-symbol errors. Test files are included; use
# the pre-commit hook for staged-only scope. No installation or provider setup.
#
# The tool binary is built for the host first: GOOS=linux must apply to the
# ANALYSIS, not to the tool build, or the built binary cannot execute here.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local
hostbin="$(mktemp -d)/staticcheck"
trap 'rm -rf "$(dirname "$hostbin")"' EXIT
GOOS= GOARCH= go build -o "$hostbin" honnef.co/go/tools/cmd/staticcheck
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  GOOS=linux "$hostbin" $*
else
  GOOS=linux "$hostbin" ./internal/... ./cmd/... ./tools/...
fi

#!/bin/bash
# gofumpt gate: stricter-than-gofmt formatting, zero tolerance. Accepts file
# paths; defaults to every tracked Go file. Semantics-preserving; fix with
# `go tool gofumpt -w` on the listed files. A failing tool fails the gate:
# only a successful tool run with no listed files passes.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  unformatted=$(go tool gofumpt -l $*) || { echo 'gofumpt tool failed' >&2; exit 2; }
else
  unformatted=$(git ls-files '*.go' | xargs go tool gofumpt -l) || { echo 'gofumpt tool failed' >&2; exit 2; }
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  count=$(printf '%s\n' "$unformatted" | wc -l)
  printf 'gofumpt gate failed: %s file(s) need formatting.\n' "$count"
  exit 1
fi
printf 'gofumpt gate passed.\n'

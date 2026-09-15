#!/bin/bash
# gofumpt gate: stricter-than-gofmt formatting, zero tolerance. Accepts file
# paths; defaults to every tracked Go file. Semantics-preserving; fix with
# `go tool gofumpt -w` on the listed files.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  unformatted=$(go tool gofumpt -l $* || true)
else
  unformatted=$(git ls-files '*.go' | xargs go tool gofumpt -l || true)
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  count=$(printf '%s\n' "$unformatted" | wc -l)
  printf 'gofumpt gate failed: %s file(s) need formatting.\n' "$count"
  exit 1
fi
printf 'gofumpt gate passed.\n'

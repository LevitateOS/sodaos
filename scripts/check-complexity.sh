#!/bin/bash
# Cyclomatic complexity gate for production Go code: every non-test function
# must stay below 10 (gocyclo flags complexity 10 and above). Test files are
# deliberately out of scope. No appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local
if [ "$#" -gt 0 ]; then
  files="$*"
else
  files=$(find internal cmd tools scripts appliance project-os tests \
    -name '*.go' ! -name '*_test.go')
fi
# shellcheck disable=SC2086
violations=$(go tool gocyclo -over 9 $files || true)
if [ -n "$violations" ]; then
  printf '%s\n' "$violations"
  count=$(printf '%s\n' "$violations" | wc -l)
  printf 'Complexity gate failed: %s production function(s) at cyclomatic complexity 10 or above.\n' "$count"
  exit 1
fi
printf 'Complexity gate passed: all production Go functions below 10.\n'

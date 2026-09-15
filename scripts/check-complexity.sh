#!/bin/bash
# Cyclomatic complexity gate for shipping Go: every non-test function in
# internal/, cmd/, tools/, appliance/, and project-os/ must stay below 10
# (gocyclo flags 10 and above). scripts/, tests/, and *_test.go are out of
# scope. No appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local

shipping() {
  printf '%s\n' "$@" | grep -E '^(internal|cmd|tools|appliance|project-os)/' | grep -v '_test\.go$' || true
}

if [ "$#" -gt 0 ]; then
  files=$(shipping "$@")
else
  files=$(find internal cmd tools appliance project-os -name '*.go' ! -name '*_test.go')
fi
if [ -z "${files:-}" ]; then
  printf 'Complexity gate passed: no shipping Go files to check.\n'
  exit 0
fi
# shellcheck disable=SC2086
violations=$(go tool gocyclo -over 9 $files || true)
if [ -n "$violations" ]; then
  printf '%s\n' "$violations"
  count=$(printf '%s\n' "$violations" | wc -l)
  printf 'Complexity gate failed: %s shipping function(s) at cyclomatic complexity 10 or above.\n' "$count"
  exit 1
fi
printf 'Complexity gate passed: all shipping Go functions below 10.\n'

#!/bin/bash
# Cyclomatic complexity gate for production TypeScript: every non-test function
# must stay below 10 (oxlint eslint/complexity flags complexity 10 and above,
# matching gocyclo -over 9). Tests, fixtures and testdata are out of scope.
set -euo pipefail
cd "$(dirname "$0")/.."
oxlint=node_modules/.bin/oxlint
if [ ! -x "$oxlint" ]; then
  printf 'oxlint is not installed; run bun install --frozen-lockfile\n'
  exit 1
fi
if [ "$#" -gt 0 ]; then
  files="$*"
else
  files=$(git ls-files '*.ts' '*.tsx' | grep -v -E '(^tests/|^docs/|/fixtures/|/testdata/|\.test\.ts$)' || true)
fi
if [ -z "$files" ]; then
  printf 'Complexity gate passed: no production TypeScript files to check.\n'
  exit 0
fi
# shellcheck disable=SC2086
out=$("$oxlint" --format unix $files || true)
violations=$(printf '%s\n' "$out" | grep 'eslint(complexity)]' || true)
if [ -n "$violations" ]; then
  printf '%s\n' "$violations"
  count=$(printf '%s\n' "$violations" | wc -l | tr -d ' ')
  printf 'Complexity gate failed: %s production function(s) at cyclomatic complexity 10 or above.\n' "$count"
  exit 1
fi
printf 'Complexity gate passed: all production TypeScript functions below 10.\n'

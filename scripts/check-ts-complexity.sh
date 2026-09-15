#!/bin/bash
# Cyclomatic complexity gate for browser-payload TypeScript: frontend/ and
# assets/branding/ only. Every function must stay below 10 (oxlint
# eslint/complexity flags 10 and above, matching gocyclo -over 9). scripts/,
# tools/, tests, docs, fixtures and testdata are out of scope.
set -euo pipefail
cd "$(dirname "$0")/.."
oxlint=node_modules/.bin/oxlint
if [ ! -x "$oxlint" ]; then
  printf 'oxlint is not installed; run bun install --frozen-lockfile\n'
  exit 1
fi

payload() {
  printf '%s\n' "$@" | grep -E '^(frontend/|assets/branding/)' | grep -v -E '(/fixtures/|/testdata/|\.test\.ts$)' || true
}

if [ "$#" -gt 0 ]; then
  files=$(payload "$@")
else
  files=$(payload $(git ls-files '*.ts' '*.tsx'))
fi
if [ -z "${files:-}" ]; then
  printf 'Complexity gate passed: no browser-payload TypeScript files to check.\n'
  exit 0
fi
# shellcheck disable=SC2086
out=$("$oxlint" --format=agent $files || true)
violations=$(printf '%s\n' "$out" | grep 'eslint(complexity)' || true)
if [ -n "$violations" ]; then
  printf '%s\n' "$violations"
  count=$(printf '%s\n' "$violations" | wc -l | tr -d ' ')
  printf 'Complexity gate failed: %s browser-payload function(s) at cyclomatic complexity 10 or above.\n' "$count"
  exit 1
fi
printf 'Complexity gate passed: all browser-payload TypeScript functions below 10.\n'

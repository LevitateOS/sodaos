#!/bin/bash
# Cyclomatic complexity gate for shipping Python: internal/, cmd/, appliance/,
# and project-os/ only. Every function must stay below 10 (Ruff C901 / mccabe
# max-complexity 9, matching gocyclo -over 9). scripts/, tools/, and tests/ are
# out of scope.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/ruff-env.sh
. scripts/ruff-env.sh

shipping() {
  printf '%s\n' "$@" | grep -E '^(internal|cmd|appliance|project-os)/' | grep -v '_test\.py$' || true
}

if [ "$#" -gt 0 ]; then
  files=$(shipping "$@")
else
  files=$(shipping $(git ls-files '*.py'))
fi
if [ -z "${files:-}" ]; then
  printf 'Complexity gate passed: no shipping Python files to check.\n'
  exit 0
fi
# shellcheck disable=SC2086
out=$("${ruff_cmd[@]}" check --select C901 --output-format=concise $files || true)
violations=$(printf '%s\n' "$out" | grep 'C901' || true)
if [ -n "$violations" ]; then
  printf '%s\n' "$violations"
  count=$(printf '%s\n' "$violations" | grep -c 'C901' || true)
  printf 'Complexity gate failed: %s shipping function(s) at cyclomatic complexity 10 or above.\n' "$count"
  exit 1
fi
printf 'Complexity gate passed: all shipping Python functions below 10.\n'

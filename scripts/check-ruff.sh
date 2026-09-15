#!/bin/bash
# Ruff gate for Python correctness (E4/E7/E9/F). Complexity is owned by
# scripts/check-py-complexity.sh (same cyclomatic threshold as shipping Go).
# Accepts paths; defaults to every tracked .py file. Test files are included.
# Use the pre-commit hook for staged-only scope.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/ruff-env.sh
. scripts/ruff-env.sh
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  "${ruff_cmd[@]}" check --output-format=concise $*
else
  files=$(git ls-files '*.py')
  if [ -z "$files" ]; then
    printf 'ruff gate passed.\n'
    exit 0
  fi
  # shellcheck disable=SC2086
  "${ruff_cmd[@]}" check --output-format=concise $files
fi
printf 'ruff gate passed.\n'

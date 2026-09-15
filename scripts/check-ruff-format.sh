#!/bin/bash
# Ruff format gate for Python: zero-tolerance formatting. Accepts file paths;
# defaults to every tracked .py file. Fix with `ruff format` on the listed files.
# The pre-commit hook checks staged files only so the current whole-tree format
# backlog does not block unrelated commits. Ruff's check output includes a
# success summary, so this gate uses the process exit code, not nonempty stdout.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/ruff-env.sh
. scripts/ruff-env.sh
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  "${ruff_cmd[@]}" format --check --output-format=concise $*
else
  files=$(git ls-files '*.py')
  if [ -z "$files" ]; then
    printf 'ruff format gate passed.\n'
    exit 0
  fi
  # shellcheck disable=SC2086
  "${ruff_cmd[@]}" format --check --output-format=concise $files
fi
printf 'ruff format gate passed.\n'

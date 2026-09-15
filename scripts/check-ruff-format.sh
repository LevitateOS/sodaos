#!/bin/bash
# Ruff format gate for Python: zero-tolerance formatting. Accepts file paths;
# defaults to every tracked .py file. Fix with `ruff format` on the listed files.
# The pre-commit hook checks staged files only so the current whole-tree format
# backlog does not block unrelated commits.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/ruff-env.sh
. scripts/ruff-env.sh
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  unformatted=$("${ruff_cmd[@]}" format --check --output-format=concise $* || true)
else
  files=$(git ls-files '*.py')
  if [ -z "$files" ]; then
    printf 'ruff format gate passed.\n'
    exit 0
  fi
  # shellcheck disable=SC2086
  unformatted=$("${ruff_cmd[@]}" format --check --output-format=concise $files || true)
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  count=$(printf '%s\n' "$unformatted" | grep -c 'unformatted' || true)
  printf 'ruff format gate failed: %s file(s) need formatting (run ruff format).\n' "$count"
  exit 1
fi
printf 'ruff format gate passed.\n'

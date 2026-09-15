#!/bin/bash
# Prettier gate for TypeScript: zero-tolerance formatting. Accepts file paths;
# defaults to every tracked .ts/.tsx file. Semantics-preserving; fix with
# `bunx prettier --write` on the listed files. The pre-commit hook checks staged
# files only so the current whole-tree format backlog does not block unrelated
# commits.
set -euo pipefail
cd "$(dirname "$0")/.."
prettier=node_modules/.bin/prettier
if [ ! -x "$prettier" ]; then
  printf 'prettier is not installed; run bun install --frozen-lockfile\n'
  exit 1
fi
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  unformatted=$("$prettier" --list-different $* || true)
else
  files=$(git ls-files '*.ts' '*.tsx')
  if [ -z "$files" ]; then
    printf 'prettier gate passed.\n'
    exit 0
  fi
  # shellcheck disable=SC2086
  unformatted=$("$prettier" --list-different $files || true)
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  count=$(printf '%s\n' "$unformatted" | wc -l | tr -d ' ')
  printf 'prettier gate failed: %s file(s) need formatting.\n' "$count"
  exit 1
fi
printf 'prettier gate passed.\n'

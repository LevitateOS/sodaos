#!/bin/bash
# Oxfmt gate for TypeScript: zero-tolerance formatting. Accepts file paths;
# defaults to every tracked .ts/.tsx file. Semantics-preserving; fix with
# `bunx oxfmt --write` on the listed files. The pre-commit hook checks staged
# files only so the current whole-tree format backlog does not block unrelated
# commits.
set -euo pipefail
cd "$(dirname "$0")/.."
oxfmt=node_modules/.bin/oxfmt
if [ ! -x "$oxfmt" ]; then
  printf 'oxfmt is not installed; run bun install --frozen-lockfile\n'
  exit 1
fi
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  unformatted=$("$oxfmt" --list-different $* || true)
else
  files=$(git ls-files '*.ts' '*.tsx')
  if [ -z "$files" ]; then
    printf 'oxfmt gate passed.\n'
    exit 0
  fi
  # shellcheck disable=SC2086
  unformatted=$("$oxfmt" --list-different $files || true)
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  count=$(printf '%s\n' "$unformatted" | wc -l | tr -d ' ')
  printf 'oxfmt gate failed: %s file(s) need formatting.\n' "$count"
  exit 1
fi
printf 'oxfmt gate passed.\n'

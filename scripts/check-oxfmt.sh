#!/bin/bash
# Oxfmt gate for TypeScript: zero-tolerance formatting. Accepts file paths;
# defaults to every tracked .ts/.tsx file. Semantics-preserving; fix with
# `bunx oxfmt --write` on the listed files. Pass selected files for focused checks.
# commits.
set -euo pipefail
cd "$(dirname "$0")/.."
oxfmt=node_modules/.bin/oxfmt
if [ ! -x "$oxfmt" ]; then
  printf 'oxfmt is not installed; run bun install --frozen-lockfile\n'
  exit 1
fi
files=("$@")
if [ "$#" -eq 0 ]; then
  scratch_dir="$PWD/.artifacts/oxfmt"
  mkdir -p "$scratch_dir"
  discovery_file=$(mktemp "$scratch_dir/check-oxfmt-find.XXXXXX")
  trap 'rm -f "$discovery_file"' EXIT
  git ls-files -z '*.ts' '*.tsx' >"$discovery_file"
  mapfile -d '' -t files <"$discovery_file"
fi
if [ "${#files[@]}" -eq 0 ]; then
  printf 'oxfmt gate passed.\n'
  exit 0
fi
if unformatted=$("$oxfmt" --list-different "${files[@]}"); then
  formatter_status=0
else
  formatter_status=$?
fi
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
  count=$(printf '%s\n' "$unformatted" | wc -l | tr -d ' ')
  printf 'oxfmt gate failed: %s file(s) need formatting.\n' "$count"
  exit 1
fi
if [ "$formatter_status" -ne 0 ]; then
  printf 'oxfmt gate failed: formatter exited with status %s.\n' "$formatter_status" >&2
  exit 1
fi
printf 'oxfmt gate passed.\n'

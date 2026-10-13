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

selected=("$@")
if [ "$#" -eq 0 ]; then
  scratch_dir="$PWD/.artifacts/ts-complexity"
  mkdir -p "$scratch_dir"
  discovery_file=$(mktemp "$scratch_dir/check-ts-complexity-find.XXXXXX")
  trap 'rm -f "$discovery_file"' EXIT
  git ls-files -z '*.ts' '*.tsx' >"$discovery_file"
  mapfile -d '' -t selected <"$discovery_file"
fi
files=()
for path in "${selected[@]}"; do
  case "$path" in
    frontend/*|assets/branding/*)
      case "$path" in
        */fixtures/*|*/testdata/*|*.test.ts) ;;
        *) files+=("$path") ;;
      esac
      ;;
  esac
done
if [ "${#files[@]}" -eq 0 ]; then
  printf 'Complexity gate passed: no browser-payload TypeScript files to check.\n'
  exit 0
fi
# Keep the configured complexity max=9; denying the rule again on the CLI
# would reset its options. Other correctness rules belong to check-oxlint.sh.
if out=$("$oxlint" -A correctness --format=agent "${files[@]}"); then
  printf 'Complexity gate passed: all browser-payload TypeScript functions below 10.\n'
else
  analyzer_status=$?
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
  fi
  printf 'Complexity gate failed: oxlint exited with status %s.\n' "$analyzer_status" >&2
  exit 1
fi

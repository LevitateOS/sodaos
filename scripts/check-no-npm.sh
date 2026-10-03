#!/bin/bash
# No-npm gate: the npm CLI is banned from this repository. Bun v1.4.2
# (pinned via package.json packageManager) is the only JS toolchain;
# install scripts are already disabled via trustedDependencies: [].
# This gate refuses npm lockfiles/configs and npm/npx invocations in
# tracked source. Mentions of the npm *registry* as a package host
# (provenance records, registry protocol names, schema URLs) are not
# tool usage and are out of scope.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0

locks=$(git ls-files 'package-lock.json' '**/package-lock.json' 'npm-shrinkwrap.json' '**/npm-shrinkwrap.json' '.npmrc' '**/.npmrc' || true)
if [ -n "$locks" ]; then
  printf 'npm lockfiles or configs are tracked (bun.lock only):\n%s\n' "$locks"
  fail=1
fi

files=$(git ls-files '*.go' '*.py' '*.sh' '*.ts' '*.js' '*.mjs' '*.json' '*.toml' '*.container' '*Dockerfile*' '*.md' || true)
if [ -n "$files" ]; then
  # shellcheck disable=SC2086
  hits=$(grep -nE "(^|[^a-zA-Z])npx([^a-zA-Z]|$)|npm (install|i|ci|run|exec|test|publish)([^a-zA-Z]|$)" $files || true)
  if [ -n "$hits" ]; then
    printf 'npm/npx invocations found (use bun v1.4.2):\n%s\n' "$hits"
    fail=1
  fi
fi

if [ "$fail" -ne 0 ]; then
  printf 'no-npm gate failed.\n'
  exit 1
fi
printf 'no-npm gate passed.\n'

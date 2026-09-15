#!/bin/bash
# Oxlint gate for TypeScript correctness. Complexity is owned by
# scripts/check-ts-complexity.sh (same cyclomatic threshold as production Go).
# Accepts paths; defaults to the repository. Test files are included. Use the
# pre-commit hook for staged-only scope. Does not duplicate bun run typecheck.
set -euo pipefail
cd "$(dirname "$0")/.."
oxlint=node_modules/.bin/oxlint
if [ ! -x "$oxlint" ]; then
  printf 'oxlint is not installed; run bun install --frozen-lockfile\n'
  exit 1
fi
# Complexity stays on the dedicated cyclo gate so this report matches lint, not
# the production complexity backlog.
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  "$oxlint" -A complexity $*
else
  "$oxlint" -A complexity .
fi
printf 'oxlint gate passed.\n'

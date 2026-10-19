#!/bin/bash
# No-unit-tests gate: unit testing is bureaucracy; integration and e2e only.
# Fails on (1) Rust test attributes or test-only modules under src/,
# (2) Go *_test.go files outside tools/unitban/allow-go.txt,
# (3) TS *.test.ts files outside tools/unitban/allow-ts.txt,
# (4) any *.test.js/*.test.mjs/*.test.jsx/*.spec.* files outside docs/examples/
# (docs/examples/ holds user-walkthrough fixtures executed in the reader's own
# disposable repo; they are content, not repo suites).
# New unit tests are banned by AGENTS.md. If a flagged file is genuinely
# integration/e2e tier (real PostgreSQL, real browser, real subprocesses),
# extend the matching allowlist and say why in the change.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0

# --- Rust: no test code under src/ (crate-top tests/ dirs are integration) ---
srcrs=$(git ls-files '*.rs' | grep -E '^(cmd|lib|tools)/[^/]+/src/' || true)
if [ -n "$srcrs" ]; then
  # shellcheck disable=SC2086
  hits=$(printf '%s\n' "$srcrs" | xargs grep -nE '#\[test\]|cfg\(test' -- || true)
  if [ -n "$hits" ]; then
    printf 'Rust test attributes under src/:\n%s\n' "$hits"
    fail=1
  fi
  hits=$(printf '%s\n' "$srcrs" | grep -E '/tests?\.rs$|/tests/|_tests\.rs$' || true)
  if [ -n "$hits" ]; then
    printf 'Rust test-only modules under src/:\n%s\n' "$hits"
    fail=1
  fi
  # shellcheck disable=SC2086
  hits=$(printf '%s\n' "$srcrs" | xargs grep -nE '^[[:space:]]*(pub[^;]*[[:space:]])?mod[[:space:]]+tests?[[:space:];]' -- || true)
  if [ -n "$hits" ]; then
    printf 'Rust test module declarations under src/:\n%s\n' "$hits"
    fail=1
  fi
fi

# --- Go: integration allowlist (exact paths) ---
gofiles=$(git ls-files '*_test.go' | sort || true)
if [ -n "$gofiles" ]; then
  hits=$(comm -23 <(printf '%s\n' "$gofiles") <(grep -vE '^\s*(#|$)' tools/unitban/allow-go.txt | sort -u) || true)
  if [ -n "$hits" ]; then
    printf 'Go test files outside the integration allowlist (unit tests are banned):\n%s\n' "$hits"
    fail=1
  fi
fi

# --- TS: browser-test allowlist (exact paths) ---
tsfiles=$(git ls-files '*.test.ts' '*.test.tsx' | sort || true)
if [ -n "$tsfiles" ]; then
  hits=$(comm -23 <(printf '%s\n' "$tsfiles") <(grep -vE '^\s*(#|$)' tools/unitban/allow-ts.txt | sort -u) || true)
  if [ -n "$hits" ]; then
    printf 'TS test files outside the browser-test allowlist (unit tests are banned):\n%s\n' "$hits"
    fail=1
  fi
fi

# --- Zero-tolerance test filename shapes ---
hits=$(git ls-files '*.test.js' '*.test.mjs' '*.test.jsx' '*.spec.ts' '*.spec.tsx' '*.spec.js' '*.spec.jsx' | grep -v '^docs/examples/' || true)
if [ -n "$hits" ]; then
  printf 'banned test filename shapes (unit tests are banned):\n%s\n' "$hits"
  fail=1
fi

# --- Hygiene: warn (do not fail) on stale allowlist entries ---
stale=$(comm -13 <(git ls-files '*.go' | sort) <(grep -vE '^\s*(#|$)' tools/unitban/allow-go.txt | sort -u) || true)
[ -n "$stale" ] && printf 'warning: stale Go allowlist entries (file deleted, list not pruned):\n%s\n' "$stale" >&2
stale=$(comm -13 <(git ls-files '*.ts' | sort) <(grep -vE '^\s*(#|$)' tools/unitban/allow-ts.txt | sort -u) || true)
[ -n "$stale" ] && printf 'warning: stale TS allowlist entries (file deleted, list not pruned):\n%s\n' "$stale" >&2

if [ "$fail" -ne 0 ]; then
  printf 'no-unit-tests gate failed. Unit tests are forbidden: see docs/development/why-no-unit-tests.md.\n'
  exit 1
fi
printf 'no-unit-tests gate passed: integration and e2e tests only.\n'

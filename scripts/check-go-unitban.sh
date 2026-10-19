#!/bin/bash
# Go unitban gate: runs the unitban vet analyzer (tools/unitban) over ./...
# plus the analyzer's self-test. Unit testing is bureaucracy (AGENTS.md):
# test entry points may only exist in tools/unitban/allow-go.txt.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux

mkdir -p .artifacts
vetbin=".artifacts/unitban-vettool.$$"
auditdir="tools/unitban/tmpaudit-$$"
cleanup() { rm -rf "$vetbin" "$auditdir"; }
trap cleanup EXIT

go build -mod=readonly -o "$vetbin" ./tools/unitban

# Self-test 1: a staged fixture copy outside the skipped directory must FAIL.
mkdir -p "$auditdir"
cp tools/unitban/testfixtures/unit.go "$auditdir"/
if go vet -mod=readonly -vettool="$PWD/$vetbin" "./$auditdir" >/dev/null 2>&1; then
  printf 'unitban self-test failed: planted unit test was not flagged\n' >&2
  exit 1
fi
# Self-test 2: the same copy with -allow must PASS (exemption works).
if ! go vet -mod=readonly -vettool="$PWD/$vetbin" -allow="$auditdir/unit.go" "./$auditdir" >/dev/null 2>&1; then
  printf 'unitban self-test failed: -allow exemption did not pass\n' >&2
  exit 1
fi
rm -rf "$auditdir"
# Self-test 3: the fixture directory itself must PASS (skip works).
if ! go vet -mod=readonly -vettool="$PWD/$vetbin" ./tools/unitban/testfixtures/ >/dev/null 2>&1; then
  printf 'unitban self-test failed: fixture skip broken\n' >&2
  exit 1
fi

# The gate: no test entry points outside the integration allowlist.
go vet -mod=readonly -vettool="$PWD/$vetbin" ./...

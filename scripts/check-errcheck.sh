#!/bin/bash
# errcheck gate for unchecked errors. Analyzes the linux build (GOOS=linux)
# like staticcheck so installer packages are real. Accepts package paths;
# defaults to ./internal/... ./cmd/... ./tools/.... The pre-commit hook is
# staged-package-only so the whole-tree backlog does not block unrelated
# commits.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local
hostbin="$(mktemp -d)/errcheck"
trap 'rm -rf "$(dirname "$hostbin")"' EXIT
GOOS= GOARCH= go build -o "$hostbin" github.com/kisielk/errcheck
# defer f.Close() is the Go close idiom; golangci's default errcheck set
# excludes (io.Closer).Close for the same reason. Other ignored errors stay.
if [ "$#" -gt 0 ]; then
  # shellcheck disable=SC2086
  if ! GOOS=linux "$hostbin" -ignore Close $*; then
    printf 'errcheck gate failed on the given packages.\n'
    exit 1
  fi
else
  if ! GOOS=linux "$hostbin" -ignore Close ./internal/... ./cmd/... ./tools/...; then
    printf 'errcheck gate failed.\n'
    exit 1
  fi
fi
printf 'errcheck gate passed.\n'

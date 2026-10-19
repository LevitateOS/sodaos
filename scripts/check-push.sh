#!/usr/bin/env bash
# Focused push contracts; complete native/cross-language suites live in deep CI.
set -euo pipefail
cd "$(dirname "$0")/.."
[[ $# == 1 ]] || exit 2
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux
case $1 in
  core|store|web)
    : "${SODA_PG_HOST:?Disposable PostgreSQL is required}"
    : "${SODA_PG_PORT:?Disposable PostgreSQL is required}"
    [[ -s ${SODA_PG_SUPER_PASSWORD_FILE:?Restricted PostgreSQL password file is required} ]]
    case $1 in
      core)
        go mod verify
        listed=$(go list ./cmd/... ./internal/... ./tools/...)
        mapfile -t packages < <(printf '%s\n' "$listed" | \
          grep -vE '/(internal/(store|web(/.*)?|factory/control|identity/client)|tools/soda-installed-probes)$')
        (( ${#packages[@]} > 0 ))
        go test -count=1 "${packages[@]}"
        go test -count=1 ./tests/build -run '^(TestCI|TestSource|TestPush|TestDockerInstaller)'
        bash scripts/check-sql-locality.sh
        bash scripts/check-no-npm.sh
        bash scripts/check-no-python.sh
        bash scripts/check-no-unit-tests.sh
        bash scripts/check-go-unitban.sh
        ;;
      store) go test -count=1 ./internal/store ;;
      web) go test -count=1 ./internal/web/... ;;
    esac
    ;;
  browser)
    bun run typecheck
    # Unit source-contract tests were removed; browser suites run in deep CI
    # where Playwright Chromium is installed (see check:source browser).
    ;;
  rust)
    cargo test --locked -p soda-wire-time -p soda-unix-http -p soda-build-tools -p soda-release-deliver --lib
    ;;
  *) exit 2 ;;
esac

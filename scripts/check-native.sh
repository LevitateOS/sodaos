#!/bin/bash
# Explicit later source-test execution; never installs, registers or reboots.
set -euo pipefail
arch=${1:?usage: check-native.sh x86_64|aarch64}
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64) export GOARCH=amd64;; aarch64) export GOARCH=arm64;; *) exit 2;; esac
export GOOS=linux GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0
cd "$(dirname "$0")/.."
export GOTOOLCHAIN=local
[[ $(go env GOVERSION) == go1.26.7 && $(node --version) == v24.20.0 && $(bun --version) == 1.4.0 ]] || { echo 'Pinned native source-check tools required' >&2; exit 1; }
go mod verify
revision=$(git rev-parse HEAD)
[[ -z $(git status --porcelain --untracked-files=normal) ]] || { echo 'Check requires a clean exact-revision checkout' >&2; exit 1; }
".artifacts/native/$arch/tools/soda-artifacts" verify --source "$PWD/.artifacts/native/$arch" --arch "$arch" --revision "$revision"
go test -mod=readonly ./...
node --test tests/frontend/*.test.mjs
(cd cockpit && bun run typecheck && bun run test)
python3 -m unittest discover -s tests/build
SODA_STAGE="$PWD/.artifacts/native/$arch/rootfs" python3 -m unittest discover -s tests/packaging
[[ $(git rev-parse HEAD) == "$revision" && -z $(git status --porcelain --untracked-files=normal) ]]
".artifacts/native/$arch/tools/soda-artifacts" verify --source "$PWD/.artifacts/native/$arch" --arch "$arch" --revision "$revision"
printf 'Source/staging checks executed; this is not installed appliance validation.\n'

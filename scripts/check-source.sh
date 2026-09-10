#!/bin/bash
# Local source checks; no sealed stage, appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOTOOLCHAIN=local
go mod verify
go test -mod=readonly ./...
bun run typecheck
bun run test
python3 -m unittest discover -s tests/build
printf 'Local source checks executed; optional installed/provider gates were not enabled by this command.\n'

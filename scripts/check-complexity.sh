#!/bin/bash
# Shared Go/Rust/TypeScript policy; explicit paths remain supported.
set -euo pipefail
cd "$(dirname "$0")/.."
exec bun tools/source-complexity/check.ts "$@"

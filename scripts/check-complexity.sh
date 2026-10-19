#!/bin/bash
# Advisory Go/Rust/TypeScript report; explicit paths remain supported.
set -euo pipefail
cd "$(dirname "$0")/.."
exec bun tools/source-complexity/check.ts --report "$@"

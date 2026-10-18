#!/bin/bash
# TypeScript view of the shared changed-function complexity policy.
set -euo pipefail
cd "$(dirname "$0")/.."
exec bun tools/source-complexity/check.ts --language typescript "$@"

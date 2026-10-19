#!/bin/bash
# TypeScript view of the advisory changed-function complexity report.
set -euo pipefail
cd "$(dirname "$0")/.."
exec bun tools/source-complexity/check.ts --report --language typescript "$@"

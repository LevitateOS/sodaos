#!/bin/bash
# Build native artifacts and then the ISO with the existing recipe owners.
set -euo pipefail
exec python3 "$(dirname "$0")/build-installer.py" --build-native "$@"

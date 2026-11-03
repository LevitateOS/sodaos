#!/usr/bin/env bash
# Install the shared UI source dependency; this never builds or starts Factory.
set -euo pipefail
cd "$(dirname "$0")/.."
if (( $# > 1 )); then
  printf 'usage: setup-factory-ui.sh [FACTORY_CHECKOUT]\n' >&2
  exit 2
fi
case "${TMPDIR:-}" in
  /home/*) ;;
  *) export TMPDIR="$HOME/tmp"; mkdir -p "$TMPDIR" ;;
esac
if (( $# == 1 )); then
  factory_checkout=$(cd "$1" && pwd -P)
else
  factory_version=$(awk '$1 == "github.com/levitateos/factory" { print $2 }' go.mod)
  factory_revision=${factory_version##*-}
  [[ "$factory_revision" =~ ^[0-9a-f]{12}$ ]] || {
    printf 'Factory UI setup requires the commit selected by go.mod.\n' >&2
    exit 1
  }
  factory_checkout=$(mktemp -d "$HOME/factory-source.XXXXXX")
  git clone --filter=blob:none --no-checkout https://github.com/LevitateOS/factory.git "$factory_checkout"
  git -C "$factory_checkout" checkout --detach "$factory_revision"
fi
bun install --cwd "$factory_checkout" --frozen-lockfile --ignore-scripts
bun link --cwd "$factory_checkout" --no-save
bun install --frozen-lockfile --ignore-scripts
printf 'Forge UI consumes Factory source at %s\n' "$factory_checkout"

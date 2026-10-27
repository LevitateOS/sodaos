#!/bin/sh
# Render installer, desktop and web icons from the current SVG masters.
set -eu
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"
mode=${1:-write}
case "$mode" in write|--check) ;; *) echo "usage: $0 [--check]" >&2; exit 2 ;; esac
[ "$#" -le 1 ] || exit 2
command -v rsvg-convert >/dev/null
workdir=$(mktemp -d "$repo_root/../soda-branding.XXXXXX")
trap 'rm -rf "$workdir"' EXIT HUP INT TERM
export TMPDIR="$workdir"
tab=$(printf '\t')
while IFS="$tab" read -r source output width height background; do
    case "$source" in \#*|'') continue ;; esac
    generated="$workdir/asset.png"
    rsvg-convert --format=png --width="$width" --height="$height" --keep-aspect-ratio --background-color="$background" --output="$generated" "$source"
    if [ "$mode" = "--check" ]; then
        go run ./tools/png-equal "$generated" "$output"
    else
        install -m 0644 "$generated" "$output"
    fi
done < assets/branding/manifest.tsv
sh scripts/render-forgejo-branding.sh "$mode"

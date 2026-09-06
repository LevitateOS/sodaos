#!/bin/bash
# Explicit native dashboard-only build. Does not install or restart anything.
set -euo pipefail
arch=${1:?usage: build-dashboard.sh x86_64|aarch64 [--payload-only]}
# Full-native packaging consumes this same core-owned payload without building
# or overwriting a second dashboard image. Standalone behavior stays unchanged.
payload_only=false
case "${2:-}" in '') ;; --payload-only) payload_only=true;; *) echo 'Unknown build mode' >&2; exit 2;; esac
[[ $# -le 2 ]] || exit 2
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64|aarch64) ;; *) exit 2;; esac
cd "$(dirname "$0")/.."
[[ -f dashboard/pnpm-lock.yaml ]] || { echo 'Dashboard dependency resolution is pending: explicitly resolve/review/commit its real lockfile before this frozen build.' >&2; exit 1; }
[[ $(go env GOVERSION) == go1.26.7 ]] || { echo 'Go 1.26.7 required' >&2; exit 1; }
[[ $(node --version) == v24.20.0 && $(pnpm --version) == 11.25.0 ]] || { echo 'Use the pinned Node/pnpm baseline' >&2; exit 1; }
out=".artifacts/native/$arch"
[[ ! -e "$out/dashboard" && ! -e "$out/images/dashboard.oci" ]] || { echo 'Dashboard output exists; preserve it and explicitly clear only approved generated outputs before rebuilding.' >&2; exit 1; }
umask 022
mkdir -p "$out/bin" "$out/images"
(cd dashboard && pnpm install --frozen-lockfile && pnpm exec vp build)
cp -a dashboard/dist "$out/dashboard"
CGO_ENABLED=0 go build -mod=readonly -buildvcs=true -trimpath -o "$out/bin/soda-dashboard" ./cmd/soda-dashboard
if [[ "$payload_only" == false ]]; then
  podman build --build-arg "ARTIFACT_DIR=$out" -t localhost/soda-dashboard:dev -f appliance/dashboard.Containerfile .
  podman save -o "$out/images/dashboard.oci" localhost/soda-dashboard:dev
fi
printf 'Dashboard source built into %s; no deployment or runtime validation performed.\n' "$out"

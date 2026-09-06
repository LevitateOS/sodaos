#!/bin/bash
# Authored now; run only in the later authorized build phase.
set -euo pipefail
arch=${1:?usage: build-native.sh x86_64|aarch64}
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64|aarch64) ;; *) exit 2;; esac
cd "$(dirname "$0")/.."
[[ -f dashboard/pnpm-lock.yaml ]] || { echo 'Resolve/review/commit the dashboard lockfile in an authorized dependency phase before building.' >&2; exit 1; }
[[ -f go.sum ]] || { echo 'First resolve dependencies with go mod tidy on the native builder; inspect and commit the resulting go.mod/go.sum before building.' >&2; exit 1; }
[[ $(go env GOVERSION) == go1.26.7 ]] || { echo 'Go 1.26.7 required' >&2; exit 1; }
[[ $(node --version) == v24.20.0 && $(pnpm --version) == 11.25.0 ]] || { echo 'Use the pinned Cockpit Node/pnpm baseline' >&2; exit 1; }
out=".artifacts/native/$arch"
mkdir -p "$out/bin" "$out/images"
for command in cmd/*; do
  [[ -d "$command" && "$command" != cmd/soda-dashboard ]] || continue
  CGO_ENABLED=0 go build -mod=readonly -trimpath -o "$out/bin/$(basename "$command")" "./$command"
done
bash scripts/build-dashboard.sh "$arch"
(cd cockpit && pnpm install --frozen-lockfile && pnpm exec vp build)
python3 scripts/build-project-tools.py --arch "$arch"
podman build --build-arg "ARTIFACT_DIR=$out" -t localhost/soda-project-os:dev -f project-os/Containerfile .
podman save -o "$out/images/project-os.oci" localhost/soda-project-os:dev
python3 scripts/fetch-runner.py --arch "$arch" --out "$out/github-actions-runner"
python3 scripts/stage.py --arch "$arch"
printf 'Native artifacts staged at %s; no installation, publication or runtime validation was performed.\n' "$out"

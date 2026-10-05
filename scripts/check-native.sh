#!/bin/bash
# Verify a soda-build candidate artifacts directory. Does not build, install or publish.
set -euo pipefail
arch=${1:?usage: check-native.sh x86_64 CANDIDATE_ARTIFACTS_DIR}
candidate=${2:?usage: check-native.sh x86_64 CANDIDATE_ARTIFACTS_DIR}
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64) export GOARCH=amd64;; *) exit 2;; esac
export GOOS=linux GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0
cd "$(dirname "$0")/.."
export GOTOOLCHAIN=local
[[ $(go env GOVERSION) == go1.26.7 && $(bun --version) == 1.4.2 ]] || { echo 'Pinned native source-check tools required' >&2; exit 1; }
go mod verify
revision=$(git rev-parse HEAD)
[[ -z $(git status --porcelain --untracked-files=normal) ]] || { echo 'Check requires a clean exact-revision checkout' >&2; exit 1; }
[[ -d $candidate && -f $candidate/payload.json && -f $candidate/candidate.json && -f $candidate/host.oci ]] || {
  echo 'soda-build candidate artifacts directory required (payload.json, candidate.json, host.oci)' >&2
  exit 1
}
artifacts=$(cd "$candidate" && pwd)
verifier=$artifacts/tools/soda-artifacts
if [[ -x $verifier && -f $artifacts/SHA256SUMS && -d $artifacts/rootfs ]]; then
  # Retained legacy sealed native stages remain verifiable; soda-build candidates use the checks below.
  "$verifier" verify --source "$artifacts" --arch "$arch" --revision "$revision"
fi
# The delivered candidate must match the requested checkouts; the Rust owner below
# (soda-release-tools soda-candidate-check) binds those identities and verifies
# the delivered archives themselves.
forgejo_revision=$(git -C ../forgejo-ext rev-parse HEAD 2>/dev/null) || { echo 'Sibling Fountain checkout required' >&2; exit 1; }
[[ $forgejo_revision =~ ^[0-9a-f]{40}$ ]] || { echo 'Exact Fountain source revision required' >&2; exit 1; }
cargo run --locked -p soda-release-tools --bin soda-candidate-check -- --candidate "$artifacts" --arch "$arch" --soda-revision "$revision" --forgejo-revision "$forgejo_revision"
bun run check:source
[[ $(git rev-parse HEAD) == "$revision" && -z $(git status --porcelain --untracked-files=normal) ]]
printf 'Source/candidate checks executed against %s; this is not installed appliance validation.\n' "$artifacts"

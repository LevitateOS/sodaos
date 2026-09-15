#!/bin/bash
# Verify a soda-build candidate artifacts directory. Does not build, install or publish.
set -euo pipefail
arch=${1:?usage: check-native.sh x86_64|aarch64 CANDIDATE_ARTIFACTS_DIR}
candidate=${2:?usage: check-native.sh x86_64|aarch64 CANDIDATE_ARTIFACTS_DIR}
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64) export GOARCH=amd64;; aarch64) export GOARCH=arm64;; *) exit 2;; esac
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
python3 - <<PY
import json, sys
from pathlib import Path
root = Path("$artifacts")
payload = json.loads((root / "payload.json").read_text())
candidate = json.loads((root / "candidate.json").read_text())
if payload.get("Architecture") != "$arch":
    sys.exit("candidate architecture mismatch")
images = root / "images"
required = {"dashboard", "forgejo", "project-os", "proxy", "tailnet"}
present = {p.stem for p in images.glob("*.oci")} if images.is_dir() else set()
missing = sorted(required - present)
if missing:
    sys.exit("missing application archives: " + ", ".join(missing))
print("Candidate identity files present; not an installed or published release.")
PY
bun run check:source
if [[ -n ${SODA_STAGE:-} ]]; then
  # Optional retained writable rootfs reader; never produced by soda-build.
  python3 -m unittest discover -s tests/packaging
fi
[[ $(git rev-parse HEAD) == "$revision" && -z $(git status --porcelain --untracked-files=normal) ]]
printf 'Source/candidate checks executed against %s; this is not installed appliance validation.\n' "$artifacts"

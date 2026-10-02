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
python3 - <<PY
import hashlib, json, sys
from pathlib import Path
root = Path("$artifacts")
payload = json.loads((root / "payload.json").read_text())
candidate = json.loads((root / "candidate.json").read_text())
if payload.get("Architecture") != "$arch":
    sys.exit("candidate architecture mismatch")
if candidate.get("Architecture") != "$arch":
    sys.exit("candidate provenance architecture mismatch")
if payload.get("Images", {}).get("extension", {}).get("Config") == payload.get("Images", {}).get("forgejo", {}).get("Config"):
    sys.exit("independent extension image identity required")
content = candidate.get("ContentSHA256", {})
toolchain = candidate.get("ForgejoToolchain", {})
packages = toolchain.get("APKPackages", [])
if toolchain.get("CompilerImage") != "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468":
    sys.exit("pinned Forgejo compiler provenance required")
if packages != sorted(set(packages)) or "build-base-0.5-r4" not in packages or not any(p.startswith("gcc-") for p in packages) or not any(p.startswith("musl-dev-") for p in packages):
    sys.exit("resolved Forgejo APK provenance required")
embedded = root.parent / "work/host-context/rootfs/usr/share/soda/host-image/content.json"
if json.loads(embedded.read_text()) != content:
    sys.exit("host and candidate content inventories differ")
paths = {
    "forgejo:/usr/local/bin/gitea": root / "forgejo-context/forgejo-bin",
    "extension:/usr/local/bin/gitea": root / "forgejo-context/forgejo-bin",
    "extension:/usr/share/soda/extension/extension.json": root / "extension-context/extension/extension.json",
    "extension:/usr/share/soda/extension/backend": root / "extension-context/extension/backend",
    "extension:/usr/share/soda/extension/run": root / "extension-context/extension/run",
    "host:/usr/share/containers/systemd/forgejo.container": root.parent / "work/host-context/rootfs/usr/share/containers/systemd/forgejo.container",
    "host:/usr/lib/systemd/system/soda-extension-install.service": root.parent / "work/host-context/rootfs/usr/lib/systemd/system/soda-extension-install.service",
}
if content.get("forgejo:/usr/local/bin/gitea") != content.get("extension:/usr/local/bin/gitea"):
    sys.exit("extension installer CLI differs from patched Forgejo")
for name in content:
    prefix = "extension:/usr/share/soda/extension/assets/"
    if name.startswith(prefix):
        relative = name.removeprefix(prefix)
        if not relative or ".." in Path(relative).parts or Path(relative).is_absolute():
            sys.exit("unsafe extension asset inventory")
        paths[name] = root / "extension-context/extension/assets" / relative
if set(paths) != set(content):
    sys.exit("incomplete candidate content inventory")
for name, path in paths.items():
    if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != content[name]:
        sys.exit("candidate content hash mismatch: " + name)
packages = root / "packages.txt"
if hashlib.sha256(packages.read_bytes()).hexdigest() != payload.get("HostPackagesSHA256"):
    sys.exit("host package inventory hash mismatch")
images = root / "images"
required = {"dashboard", "forgejo", "extension", "project-os", "proxy", "tailnet"}
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

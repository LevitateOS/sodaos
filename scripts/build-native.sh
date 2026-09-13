#!/bin/bash
# Authored now; run only in the later authorized build phase.
set -euo pipefail
if [[ ${SODA_BUILD_SUPERVISED:-0} != 1 ]]; then
  exec python3 "$(dirname "$0")/build_progress.py" supervise bash "$0" "$@"
fi
source "$(dirname "$0")/build-progress.sh"
progress_next 'Native / Check build prerequisites'
# Never inherit a remote engine as matching-native build evidence.
podman() { command podman --remote=false "$@"; }
# Public payload modes, inside an explicitly private attempt directory.
umask 022
export GOTOOLCHAIN=local
export GOWORK=off
arch=${1:?usage: build-native.sh x86_64|aarch64}
[[ $(uname -s) == Linux && $(uname -m) == "$arch" ]] || { echo 'Matching native Linux required' >&2; exit 1; }
case "$arch" in x86_64) export GOARCH=amd64;; aarch64) export GOARCH=arm64;; *) exit 2;; esac
export GOOS=linux
export GOFLAGS=-mod=readonly
cd "$(dirname "$0")/.."
[[ -f go.sum ]] || { echo 'First resolve dependencies with go mod tidy on the native builder; inspect and commit the resulting go.mod/go.sum before building.' >&2; exit 1; }
[[ $(go env GOVERSION) == go1.26.7 ]] || { echo 'Go 1.26.7 required' >&2; exit 1; }
[[ $(bun --version) == 1.4.2 ]] || { echo 'Use the pinned Bun baseline' >&2; exit 1; }
revision=$(git rev-parse HEAD)
[[ "$revision" == "${SODA_BUILD_REVISION:-$revision}" ]] || { echo "Source revision differs from requested build" >&2; exit 1; }
[[ -z $(git status --porcelain --untracked-files=normal) ]] || { echo 'Build requires a clean exact-revision checkout' >&2; exit 1; }
command -v flock >/dev/null
# Serialize this checkout, without adopting or deleting an earlier attempt.
python3 - <<'PY'
import os
from pathlib import Path
p = Path('.artifacts/native')
for d in (Path('.artifacts'), p):
    # Validate each ancestor before creating anything below it.
    if d.is_symlink():
        raise SystemExit('symlinked native output parent refused')
    if not d.exists():
        d.mkdir(mode=0o700)
    if not d.is_dir() or d.resolve() != d.absolute() or d.stat().st_uid != os.getuid() or d.stat().st_mode & 0o022:
        raise SystemExit('native output parent must be real, owned and not writable by others')
lock = p / '.build.lock'
if lock.is_symlink() or (lock.exists() and not lock.is_file()):
    raise SystemExit('unsafe build lock')
PY
exec 9>.artifacts/native/.build.lock
flock -n 9 || { echo 'Another native build owns this checkout' >&2; exit 1; }
out=".artifacts/native/$arch"
[[ ! -e "$out" && ! -L "$out" ]] || { echo 'Native output already exists; select a fresh checkout, do not clear shared artifacts' >&2; exit 1; }
mkdir -m 0700 "$out"
mkdir "$out/bin" "$out/images" "$out/tools"
if [[ -z ${SODA_BUILD_TIMING_LOG:-} ]]; then
  timing_log="$PWD/.artifacts/native/$arch.timing.log"
  (set -o noclobber; umask 077; : > "$timing_log")
  export SODA_BUILD_TIMING_LOG="$timing_log"
fi
printf 'BUILD    Native payload · %s · revision %s · cache state unknown\n' "$arch" "$revision" >&2
for tool in soda-artifacts soda-acceptance; do
  progress_next "Native / Compile $tool"
  CGO_ENABLED=0 go build -mod=readonly -buildvcs=true -trimpath -o "$out/tools/$tool" "./tools/$tool"
done
for command in cmd/*; do
  [[ -d "$command" ]] || continue
  progress_next "Native / Compile ${command##*/}"
  case "${command##*/}" in soda-artifacts|soda-acceptance) echo 'Support tools must remain outside cmd/' >&2; exit 1;; esac
  CGO_ENABLED=0 go build -mod=readonly -buildvcs=true -trimpath -o "$out/bin/$(basename "$command")" "./$command"
done
progress_next 'Native / Verify Go dependencies'
go mod verify
# Soda is Go/Lit. Stock Cockpit packages and branding need no custom frontend build.
progress_next 'Native / Install frontend dependencies'
bun install --frozen-lockfile
progress_next 'Native / Build frontend assets'
bun scripts/build-forgejo.ts --out "$out/forgejo-js"
progress_next 'Native / Fetch upstream Tea binary'
python3 scripts/fetch-tea.py --arch "$arch"
case "$arch" in x86_64) oci_arch=amd64;; aarch64) oci_arch=arm64;; esac
# Resolve the unchanged core-owned base and service references for this platform.
progress_next 'Native / Pull and resolve Rocky base'
base_ref=$(awk -F= '$1=="ARG BASE_IMAGE" {print $2}' project-os/Containerfile)
[[ -n "$base_ref" && "$base_ref" == "$(awk -F= '$1=="ARG BASE_IMAGE" {print $2}' appliance/dashboard.Containerfile)" ]]
base_id=$(podman pull --quiet --platform "linux/$oci_arch" "$base_ref")
printf '%s\n' "$base_id" > "$out/base.iid"
base_digest=$(podman image inspect --format '{{.Digest}}' "$base_id")
base_pinned=$(podman image inspect --format '{{index .RepoDigests 0}}' "$base_id")
[[ "$base_digest" == sha256:* && "$base_pinned" == *@sha256:* ]]
for image in project-os dashboard; do
  containerfile="project-os/Containerfile"
  [[ "$image" != dashboard ]] || containerfile=appliance/dashboard.Containerfile
  progress_next "Native / Build image: $image"
  podman build --pull=never --platform "linux/$oci_arch" --build-arg "ARTIFACT_DIR=$out" --build-arg "BASE_IMAGE=$base_pinned" \
    --label "org.opencontainers.image.revision=$revision" \
    --label 'org.opencontainers.image.source=https://github.com/LevitateOS/sodaos' \
    --label "org.opencontainers.image.base.name=$base_ref" \
    --label "org.opencontainers.image.base.digest=$base_digest" \
    --iidfile "$out/$image.iid" -f "$containerfile" .
  # Image IDs, not mutable :dev lookups; the installer restores core-owned tags.
  progress_next "Native / Export image: $image"
  podman save --format oci-archive -o "$out/images/$image.oci" "$(<"$out/$image.iid")"
done
# The selected patch release has public binaries but no published container tag.
# Reuse upstream's immutable Alpine base and checksum-locked release binaries.
progress_next 'Native / Resolve and pull Tailnet base'
readarray -t tailnet_inputs < <(python3 - "$oci_arch" <<'PY'
import json, re, sys
v = json.load(open('appliance/locks/tailscale-image.json'))
assert re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+', v['version'])
assert re.fullmatch(r'docker.io/tailscale/alpine-base@sha256:[0-9a-f]{64}', v['base'])
assert re.fullmatch(r'[0-9a-f]{64}', v['sha256'][sys.argv[1]])
print(v['base']); print(v['version']); print(v['sha256'][sys.argv[1]])
PY
)
[[ ${#tailnet_inputs[@]} == 3 ]]
podman pull --quiet --platform "linux/$oci_arch" "${tailnet_inputs[0]}" > "$out/tailnet-base.iid"
progress_next 'Native / Build Tailnet companion image'
podman build --pull=never --platform "linux/$oci_arch" \
  --build-arg "BASE_IMAGE=${tailnet_inputs[0]}" --build-arg "TAILSCALE_VERSION=${tailnet_inputs[1]}" \
  --build-arg "TARGETARCH=$oci_arch" --build-arg "ARCHIVE_SHA256=${tailnet_inputs[2]}" \
  --label "org.opencontainers.image.revision=$revision" \
  --label 'org.opencontainers.image.source=https://github.com/LevitateOS/sodaos' \
  --iidfile "$out/tailnet.iid" -f appliance/tailnet.Containerfile .
progress_next 'Native / Export Tailnet companion image'
podman save --format oci-archive -o "$out/images/tailnet.oci" "$(<"$out/tailnet.iid")"
for image in forgejo caddy; do
  unit=appliance/services/forgejo.container
  [[ "$image" != caddy ]] || unit=appliance/services/soda-proxy.container
  progress_next "Native / Pull image: $image"
  reference=$(awk -F= '$1=="Image" {print $2}' "$unit")
  id=$(podman pull --quiet --platform "linux/$oci_arch" "$reference")
  printf '%s\n' "$id" > "$out/$image.iid"
  progress_next "Native / Export image: $image"
  podman save --format oci-archive -o "$out/images/$image.oci" "$id"
done
progress_next 'Native / Fetch terminal assets'
python3 scripts/fetch-terminal.py --out "$out/terminal-assets"
progress_next 'Native / Prepare Forgejo translations'
python3 scripts/forgejo-locales.py --lock appliance/forgejo/locale.lock.json --out "$out/forgejo-locales/locale_en-US.ini"
progress_next 'Native / Stage appliance files'
python3 scripts/stage.py --arch "$arch"
progress_done
python3 scripts/native-build-info.py --arch "$arch" --revision "$revision"
progress_next 'Native / Confirm unchanged source'
[[ $(git rev-parse HEAD) == "$revision" && -z $(git status --porcelain --untracked-files=normal) ]] || { echo 'Source changed during build; output is not sealed' >&2; exit 1; }
progress_next 'Native / Seal native payload'
"$out/tools/soda-artifacts" seal --source "$PWD/$out" --arch "$arch" --revision "$revision"
progress_done
printf 'Native artifacts staged at %s; no installation, publication or runtime validation was performed.\n' "$out"

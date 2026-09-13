#!/bin/bash
# Legacy writable-layout adapter. Shared component production is owned by the
# Go image builder; this script retains checkout admission, locking and sealing.
set -euo pipefail
if [[ ${SODA_BUILD_SUPERVISED:-0} != 1 ]]; then
  exec python3 "$(dirname "$0")/build_progress.py" supervise bash "$0" "$@"
fi
source "$(dirname "$0")/build-progress.sh"
progress_next 'Native / Check build prerequisites'
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
producer="$out.producer"
for path in "$out" "$producer"; do
  [[ ! -e "$path" && ! -L "$path" ]] || { echo 'Native output already exists; do not clear shared artifacts' >&2; exit 1; }
done
if [[ -z ${SODA_BUILD_TIMING_LOG:-} ]]; then
  timing_log="$PWD/.artifacts/native/$arch.timing.log"
  (set -o noclobber; umask 077; : > "$timing_log")
  export SODA_BUILD_TIMING_LOG="$timing_log"
fi
printf 'BUILD    Legacy native payload · %s · revision %s · cache state unknown\n' "$arch" "$revision" >&2
progress_next 'Native / Compile shared artifact producer'
CGO_ENABLED=0 go build -mod=readonly -buildvcs=true -trimpath -o "$producer" ./tools/soda-host-image
progress_done
# The same producer used for release images, with the explicit legacy layout.
# Execute the compiled program directly so child exit codes are not flattened by
# go run. No application/asset/image recipe is repeated in this adapter.
SODA_BUILD_CHILD=1 "$producer" --legacy-native --arch "$arch" --out "$PWD/$out"
python3 scripts/native-build-info.py --arch "$arch" --revision "$revision"
progress_next 'Native / Confirm unchanged source'
[[ $(git rev-parse HEAD) == "$revision" && -z $(git status --porcelain --untracked-files=normal) ]] || { echo 'Source changed during build; output is not sealed' >&2; exit 1; }
progress_next 'Native / Seal native payload'
"$out/tools/soda-artifacts" seal --source "$PWD/$out" --arch "$arch" --revision "$revision"
progress_done
printf 'Legacy native artifacts staged at %s; no installation, publication or runtime validation was performed.\n' "$out"

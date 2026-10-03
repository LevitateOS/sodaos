#!/bin/bash
# setup-soda-candidate.sh prepares this machine so `sudo soda-candidate`
# runs without a coding agent: wrapper on sudo's PATH, admitted controller,
# worker directories, restricted worker config, and a fixture-only media
# authority. Development and fixture scope only: it never creates
# qualification or signing configs, and the generated keys must never
# stand in for release keys.
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/candidate-storage.sh
. ./scripts/candidate-storage.sh

PREFIX="${SODA_REPOSITORY_PREFIX:-ghcr.io/levitateos/sodaos}"
REFRESH="${SODA_REFRESH_AUTHORITY:-0}"
FORGEJO_SOURCE="${SODA_FORGEJO_SOURCE:-}"
# Pickup folder for the built system image. The ISO is only the boot menu;
# the installer downloads the big rootfs file from the persistent server,
# which serves this same directory. One rootfs-only location on the roomy
# disk: never the guest-disk directory, never the small root filesystem.
ROOTFS_DIR="/home/soda-rootfs"
# The installing machine fetches the rootfs over HTTP, so the address must
# be reachable from the guest: the libvirt bridge when present. Loopback
# always points at the guest itself and the media build refuses it.
ROOTFS_URL=""
if BRIDGE_IP="$(ip -4 -o addr show virbr0 2>/dev/null | awk '{print $4}' | cut -d/ -f1 | head -1)" && [ -n "$BRIDGE_IP" ]; then
  ROOTFS_URL="http://$BRIDGE_IP:8080"
fi
ADMITTED="/usr/local/lib/soda/soda-build"
PINNED_GO="/usr/local/lib/soda/pinned-go"
WRAPPER="/usr/sbin/soda-candidate"
OUTPUT_PARENT="$PWD/.artifacts/releases/isolated"
# Heavy worker state on /home via the single storage configuration (D1).
# TOOLS (one bun binary) and AUTHORITY (keys plus worker.json) stay small
# on root; everything heavy — engine storage, Go caches, browsers,
# per-attempt runtime and setup scratch — lives under the storage root.
BUILD_HOME="$SODA_CANDIDATE_HOME"
RUNTIME="$SODA_CANDIDATE_RUN"
SCRATCH_ROOT="$SODA_CANDIDATE_SCRATCH"
TOOLS="/var/lib/soda-candidate-tools"
AUTHORITY="/var/lib/soda-candidate-authority"
WORKER_JSON="$AUTHORITY/worker.json"

fail() { printf 'setup-soda-candidate: %s\n' "$*" >&2; exit 1; }

# Setup is the only writer of the shared provisioned tools, worker policy,
# and fixture authority, so concurrent setups would publish over each other.
# The lock lives in /tmp, never in the checkout: an untracked file there
# would trip the controller's clean-tree admission.
claim_setup_lease() {
  local lock="/tmp/soda-setup-$(id -un).lock"
  exec 9>"$lock" || fail "cannot open setup lease $lock"
  flock -n 9 || fail "another setup is already running for this operator"
}

# Refuse while a build worker unit is alive. The ACTIVE column (not a
# running-only SUB-state filter) also covers units that are still starting,
# so setup never deletes a compiler or policy out from under a live build.
# Failed or collected units are already gone and do not block setup.
refuse_active_build() {
  local units
  units="$(systemctl list-units --all --type=service --no-legend --plain 'soda-build-*' 2>/dev/null)" || fail "cannot list worker units; refusing to touch shared build state"
  if printf '%s' "$units" | awk '$3 == "active" || $3 == "activating" || $3 == "deactivating" { found=1 } END { exit !found }'; then
    fail "a candidate build is still active; finish it before rerunning setup"
  fi
}

# migrate_candidate_home moves heavy worker HOME content from the legacy
# root-backed path to the /home storage root. Engine-aware and preserving:
# it refuses while a build is active or the worker owns any process, copies
# (never moves) only into an empty new home, and leaves the legacy tree in
# place for the owner to retire (D2). The worker unit always binds this
# home at the same guest path, so recorded engine paths keep working.
# No pruning here.
migrate_candidate_home() {
  if [ ! -d "$SODA_LEGACY_HOME" ]; then return 0; fi
  if [ -n "$(ls -A "$BUILD_HOME" 2>/dev/null)" ]; then
    echo "-- new worker home already populated; legacy $SODA_LEGACY_HOME preserved untouched"
    return 0
  fi
  refuse_active_build
  if command -v pgrep >/dev/null && pgrep -u soda-build-worker >/dev/null 2>&1; then
    fail "soda-build-worker still owns processes; finish them before migrating heavy state"
  fi
  echo "-- migrating legacy worker home to $BUILD_HOME (legacy preserved)"
  sudo cp -a "$SODA_LEGACY_HOME/." "$BUILD_HOME/"
  sudo chown -R soda-build-worker:soda-build-worker "$BUILD_HOME"
  echo "-- legacy $SODA_LEGACY_HOME preserved; retire it explicitly (D2) after the new home proves itself"
}

[ -f go.mod ] || fail "run from the repository root"
[ -n "$FORGEJO_SOURCE" ] || fail "set SODA_FORGEJO_SOURCE to the clean canonical Forgejo fork checkout"
[ -d "$FORGEJO_SOURCE/.git" ] && [ "$(realpath "$FORGEJO_SOURCE")" = "$FORGEJO_SOURCE" ] || fail "canonical Forgejo checkout required"
[ -z "$(git -c "safe.directory=$FORGEJO_SOURCE" -C "$FORGEJO_SOURCE" status --porcelain --untracked-files=normal)" ] || fail "Forgejo source must be clean and committed"
[ "$(uname -m)" = "x86_64" ] || fail "matching native x86_64 required on this host"
command -v go bun podman skopeo python3 flock >/dev/null || fail "pinned go, bun, podman, skopeo, python3 and flock required"
id soda-build-worker >/dev/null 2>&1 || fail "soda-build-worker user missing"
[ -n "$(git status --porcelain --untracked-files=no)" ] && fail "commit or stash tracked changes first; the controller refuses dirty source"
PINNED="$(grep '^go ' go.mod | awk '{print $2}')"
[ -n "$PINNED" ] || fail "go.mod pins no Go version"
# The worker requires the exact upstream pinned toolchain. Anything else
# fails admission: the local toolchain may be newer, and Red Hat rebuilds
# poison runtime.Version. Toolchain switching downloads it once; the exact
# stamp is verified before anything is admitted.
export GOTOOLCHAIN="go$PINNED"
go version >/dev/null || fail "cannot fetch Go $PINNED"
PINNED_GOROOT="$(go env GOROOT)"
WANT="go version go$PINNED linux/amd64"

# Singular setup (lease held until exit), and no setup while a build is
# alive. The build refusal is re-checked at each destructive step below;
# this early check fails fast.
claim_setup_lease
refuse_active_build

echo "-- candidate storage root ($SODA_CANDIDATE_ROOT)"
sudo mkdir -p "$SCRATCH_ROOT"
sudo chown "${SUDO_USER:-$(id -un)}" "$SCRATCH_ROOT"

echo "-- build tools from committed source"
BINDIR="$(mktemp -d "$SCRATCH_ROOT/setup-bindir.XXXXXXXX")"
trap 'rm -rf "$BINDIR"' EXIT
go build -o "$BINDIR/soda-build" ./tools/soda-build
go build -o "$BINDIR/soda-candidate" ./tools/soda-candidate
[ "$(go version "$BINDIR/soda-build")" = "$BINDIR/soda-build: go$PINNED" ] || fail "controller stamp is not Go $PINNED; refusing to admit it"

echo "-- install wrapper and admitted controller"
sudo install -m 0755 "$BINDIR/soda-candidate" "$WRAPPER"
sudo install -D -m 0755 "$BINDIR/soda-build" "$ADMITTED"
# Canonicalize: /sbin may be a symlink to /usr/sbin, so compare targets.
GOT_WRAPPER="$(sudo readlink -f "$(sudo which soda-candidate)")"
WANT_WRAPPER="$(readlink -f "$WRAPPER")"
[ "$GOT_WRAPPER" = "$WANT_WRAPPER" ] || fail "wrapper not visible on sudo secure_path (got $GOT_WRAPPER)"

echo "-- worker directories"
sudo mkdir -p "$OUTPUT_PARENT" "$BUILD_HOME" "$RUNTIME" "$TOOLS/bin" "$AUTHORITY"
migrate_candidate_home
if [ -n "$(ls -A "$SODA_LEGACY_RUN" 2>/dev/null)" ]; then
  echo "-- legacy $SODA_LEGACY_RUN holds leftovers; preserved untouched (new runs use $RUNTIME)"
fi
# The pinned GOROOT installs at a fixed host path instead of the bound
# tools dir so it carries lib_t and stays executable for the worker
# domain, and /usr/local stays readable under ProtectSystem=strict. The
# single-file bun binary tolerates the bind, so it stays in $TOOLS.
# Stage replacements before publishing: the old tree stays live until the
# new one is complete and verified, so a failed copy never removes the
# compiler out from under the next build.
sudo rm -rf "$PINNED_GO.new" "$TOOLS/bin/bun.new"
sudo cp -a "$PINNED_GOROOT" "$PINNED_GO.new"
[ "$("$PINNED_GO.new/bin/go" version)" = "$WANT" ] || fail "staged Go is not $PINNED; refusing to publish it"
sudo cp "$(command -v bun)" "$TOOLS/bin/bun.new"
sudo -u soda-build-worker "$TOOLS/bin/bun.new" --version >/dev/null || fail "staged bun is not worker-runnable; refusing to publish it"
# Re-check immediately before the destructive publish: no build may start
# between the early check and this swap.
refuse_active_build
sudo rm -rf "$TOOLS/go" "$PINNED_GO"
sudo mv "$PINNED_GO.new" "$PINNED_GO"
sudo mv "$TOOLS/bin/bun.new" "$TOOLS/bin/bun"
sudo chown -R root:root "$PINNED_GO" "$TOOLS"
if command -v semanage >/dev/null; then
  sudo semanage fcontext -a -t bin_t "$TOOLS(/.*)?" 2>/dev/null || sudo semanage fcontext -m -t bin_t "$TOOLS(/.*)?"
fi
command -v restorecon >/dev/null && sudo restorecon -R "$PINNED_GO" "$TOOLS"
# Module-cache toolchain files arrive owner-read-only; the worker compiles
# against them, so directories need traversal and files need read bits.
sudo find "$PINNED_GO" -type d -exec chmod 0755 {} +
sudo find "$PINNED_GO" -type f -exec chmod a+r {} +
sudo stat -c %C "$PINNED_GO/bin/go" | grep -q ":lib_t:" || fail "pinned GOROOT is not lib_t; the sandboxed worker could not execute it"
sudo chown soda-build-worker:soda-build-worker "$OUTPUT_PARENT" "$BUILD_HOME" "$RUNTIME"
# The storage root itself stays traversable for both the worker (bound
# home/runtime below it) and the operator (scratch): label it like the
# other build state so the sandboxed worker can reach its binds.
if command -v semanage >/dev/null; then
  sudo semanage fcontext -a -t var_lib_t "$SODA_CANDIDATE_ROOT(/.*)?" 2>/dev/null || sudo semanage fcontext -m -t var_lib_t "$SODA_CANDIDATE_ROOT(/.*)?"
fi
command -v restorecon >/dev/null && sudo restorecon "$SODA_CANDIDATE_ROOT"
# The service worker is denied file creation on user_home_t, so the output
# parent inside the checkout needs var_lib_t to take build output.
if command -v semanage >/dev/null; then
  sudo semanage fcontext -a -t var_lib_t "$OUTPUT_PARENT(/.*)?" 2>/dev/null || sudo semanage fcontext -m -t var_lib_t "$OUTPUT_PARENT(/.*)?"
fi
command -v restorecon >/dev/null && sudo restorecon -R "$OUTPUT_PARENT"
sudo chmod 0755 "$TOOLS" "$TOOLS/bin"
sudo chmod 0700 "$AUTHORITY"
echo "-- verify tools as the worker user"
[ "$(sudo -u soda-build-worker env GOTOOLCHAIN=local HOME="$BUILD_HOME" "$PINNED_GO/bin/go" version)" = "$WANT" ] || fail "provisioned Go is not $PINNED or not worker-runnable"
sudo -u soda-build-worker test -r "$PINNED_GO/src/net/textproto/header.go" || fail "provisioned GOROOT sources are not worker-readable"
sudo -u soda-build-worker "$TOOLS/bin/bun" --version >/dev/null || fail "provisioned bun is not worker-runnable"

echo "-- warm worker caches (the isolated worker has no network)"
# Go 1.26 defaults GOPROXY/GOSUMDB to empty and the service worker is
# denied egress, so all Go modules must be cached and Go locked offline.
# `go build` warms exactly the readonly build set without touching the
# checkout; fetched as root, then handed to the worker.
sudo mkdir -p "$BUILD_HOME/go-mod" "$BUILD_HOME/go-build"
# `go mod download all`, not just `go build ./...`: the worker runs
# `go mod verify` over the whole build list while offline.
sudo env HOME="$BUILD_HOME" GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org GOMODCACHE="$BUILD_HOME/go-mod" GOCACHE="$BUILD_HOME/go-build" GOTOOLCHAIN="go$PINNED" GOFLAGS=-mod=readonly CGO_ENABLED=0 "$PINNED_GO/bin/go" build ./... || fail "cannot warm Go module cache"
sudo env HOME="$BUILD_HOME" GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org GOMODCACHE="$BUILD_HOME/go-mod" GOCACHE="$BUILD_HOME/go-build" GOTOOLCHAIN="go$PINNED" GOFLAGS=-mod=readonly CGO_ENABLED=0 "$PINNED_GO/bin/go" mod download all || fail "cannot warm full Go module set"
sudo -u soda-build-worker env HOME="$BUILD_HOME" "$PINNED_GO/bin/go" env -w GOPROXY=off || fail "cannot lock worker Go offline"
# Bun installs only from its HOME cache inside: warm it from a scratch copy
# (mirroring the workspaces list) so node_modules never lands in the
# checkout. Bun refuses files owned by another user, so the scratch tree
# belongs to the worker first.
TMPW="$(mktemp -d "$SCRATCH_ROOT/setup-bun.XXXXXXXX")"
mkdir -p "$TMPW/tools"
cp package.json bun.lock bunfig.toml "$TMPW/"
cp -a tools/lit-check "$TMPW/tools/"
sudo chown -R soda-build-worker:soda-build-worker "$TMPW"
sudo find "$TMPW" -type d -exec chmod 0755 {} +
(cd "$TMPW" && sudo -u soda-build-worker env HOME="$BUILD_HOME" "$TOOLS/bin/bun" install --frozen-lockfile >/dev/null) || fail "cannot warm Bun cache"
# Playwright browsers cannot be fetched offline, so the pinned chromium
# ships in the worker home now (setup has network; the worker does not).
# Install from the scratch tree so the frozen playwright is used.
sudo install -d -o soda-build-worker -g soda-build-worker "$BUILD_HOME/browsers"
(cd "$TMPW" && sudo -u soda-build-worker env HOME="$BUILD_HOME" PLAYWRIGHT_BROWSERS_PATH="$BUILD_HOME/browsers" "$TOOLS/bin/bun" x playwright install chromium) || fail "cannot stage Playwright chromium"
sudo rm -rf "$TMPW"
sudo chown -R soda-build-worker:soda-build-worker "$BUILD_HOME"

echo "-- worker SELinux policy (process groups plus Go cache mapping)"
command -v checkmodule semodule_package semodule >/dev/null || fail "policycoreutils tooling required for the worker SELinux module"
checkmodule -M -m -o "$BINDIR/soda-build-worker.mod" scripts/selinux/soda-build-worker.te
semodule_package -o "$BINDIR/soda-build-worker.pp" -m "$BINDIR/soda-build-worker.mod"
# Compile before removing: the old module stays loaded until the new one
# is ready, and no build may start between the check and the swap.
refuse_active_build
sudo semodule -r soda-build-setpgid 2>/dev/null || true
sudo semodule -i "$BINDIR/soda-build-worker.pp"
# Go runs as init_t in the worker and mmaps its cache files; the default
# var_lib_t home withholds map, so the Go state dirs carry a dedicated
# type. Podman storage, the runtime dir and frontend caches keep theirs.
sudo mkdir -p "$BUILD_HOME/go-build" "$BUILD_HOME/go-mod" "$BUILD_HOME/.config"
# Podman builds run inside the worker without a systemd user session, so
# build containers cannot use the systemd cgroup driver (crun fails on an
# sd-bus polkit denial). Pin the worker to cgroupfs; the unit's CPU and
# memory bounds still apply. BUILD_HOME is the worker HOME inside the unit.
sudo install -d -o soda-build-worker -g soda-build-worker "$BUILD_HOME/.config/containers"
printf '[engine]\ncgroup_manager = "cgroupfs"\n' | sudo tee "$BUILD_HOME/.config/containers/containers.conf" >/dev/null
sudo chown soda-build-worker:soda-build-worker "$BUILD_HOME/.config/containers/containers.conf"
sudo chmod 0644 "$BUILD_HOME/.config/containers/containers.conf"
if command -v semanage >/dev/null; then
  for d in go-build go-mod .config; do
    sudo semanage fcontext -a -t soda_build_cache_t "$BUILD_HOME/$d(/.*)?" 2>/dev/null || sudo semanage fcontext -m -t soda_build_cache_t "$BUILD_HOME/$d(/.*)?"
  done
  sudo semanage fcontext -a -t soda_build_runtime_t "$RUNTIME(/.*)?" 2>/dev/null || sudo semanage fcontext -m -t soda_build_runtime_t "$RUNTIME(/.*)?"
fi
command -v restorecon >/dev/null && sudo restorecon -R "$BUILD_HOME/go-build" "$BUILD_HOME/go-mod" "$BUILD_HOME/.config" "$RUNTIME"
sudo stat -c %C "$BUILD_HOME/go-build" | grep -q ":soda_build_cache_t:" || fail "worker Go cache is not soda_build_cache_t; the sandboxed worker could not map it"
sudo stat -c %C "$RUNTIME" | grep -q ":soda_build_runtime_t:" || fail "worker runtime is not soda_build_runtime_t; pasta could not use its netns dir"

echo "-- worker git ownership exception"
if [ ! -f /etc/gitconfig ] || ! grep -qF "directory = /run/soda-build-source" /etc/gitconfig; then
  printf '# Soda build worker: the isolated worker sees the canonical checkout\n# only at /run/soda-build-source, owned by the operator. Mark it expected.\n[safe]\n\tdirectory = /run/soda-build-source\n' | sudo tee -a /etc/gitconfig >/dev/null
  sudo chmod 0644 /etc/gitconfig
fi

echo "-- restricted worker config"
sudo python3 - "$WORKER_JSON" "$FORGEJO_SOURCE" <<EOF
import json, sys
json.dump({
  "Executable": "$ADMITTED",
  "Source": "$PWD",
  "ForgejoSource": sys.argv[2],
  "OutputParent": "$OUTPUT_PARENT",
  "StorageRoot": "$SODA_CANDIDATE_ROOT",
  "BuildHome": "$BUILD_HOME",
  "Runtime": "$RUNTIME",
  "Tools": "$TOOLS",
  "MediaAuthorityDirectory": "$AUTHORITY",
}, open(sys.argv[1], "w"), indent=2)
EOF
sudo chmod 0600 "$WORKER_JSON"

if [ -f "$AUTHORITY/artifact.private" ] && [ "$REFRESH" != "1" ]; then
  echo "-- fixture authority already exists; keeping it (SODA_REFRESH_AUTHORITY=1 to regenerate)"
else
  echo "-- fixture-only media authority (never release keys)"
  refuse_active_build
  TMPD="$(mktemp -d "$SCRATCH_ROOT/setup-authority.XXXXXXXX")"
  trap 'rm -rf "$BINDIR" "$TMPD"' EXIT
  head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' >"$TMPD/passphrase"
  for role in artifact candidate preview stable; do
    skopeo generate-sigstore-key --output-prefix "$TMPD/$role" --passphrase-file "$TMPD/passphrase" >/dev/null
  done
  NOW="$(date +%s)"
  sudo python3 - "$TMPD" "$NOW" "$PREFIX" <<'EOF'
import json, sys
tmpd, now, prefix = sys.argv[1], int(sys.argv[2]), sys.argv[3]
roles = ["artifact", "candidate", "preview", "stable"]
keys = {r: [open(f"{tmpd}/{r}.pub").read()] for r in roles}
trust = {
  "Format": 1, "Prefix": prefix, "Epoch": 1, "Keys": keys,
  "NotBefore": now - 600, "MaxAgeSeconds": 3600, "ClockSkewSeconds": 10,
  "MinimumSequence": {"candidate": 1, "preview": 1, "stable": 1},
}
open(f"{tmpd}/trust.json", "w").write(json.dumps(trust, indent=2) + "\n")
open(f"{tmpd}/config.json", "w").write(json.dumps({
  "Trust": "/run/soda-media-authority/trust.json",
  "Keys": {"Key": "/run/soda-media-authority/artifact.private",
           "Passphrase": "/run/soda-media-authority/passphrase"},
}, indent=2) + "\n")
EOF
  sudo install -m 0600 "$TMPD/trust.json" "$AUTHORITY/trust.json"
  sudo install -m 0600 "$TMPD/artifact.private" "$AUTHORITY/artifact.private"
  sudo install -m 0600 "$TMPD/passphrase" "$AUTHORITY/passphrase"
  sudo install -m 0600 "$TMPD/config.json" "$AUTHORITY/config.json"
  sudo chmod 0700 "$AUTHORITY"
fi
# The isolated worker admits only caller-owned 0600 files in a 0700
# directory, so the fixture authority it reads belongs to the worker.
# worker.json stays root-owned: the root parent admits that one.
sudo chown soda-build-worker:soda-build-worker "$AUTHORITY" "$AUTHORITY/trust.json" "$AUTHORITY/artifact.private" "$AUTHORITY/passphrase" "$AUTHORITY/config.json"

sudo mkdir -p "$ROOTFS_DIR"
# $USER is unset under set -u or points at root under sudo; the invoking
# operator is always the id outside sudo.
sudo chown "$(id -un)" "$ROOTFS_DIR"
command -v restorecon >/dev/null && sudo restorecon "$ROOTFS_DIR"

cat <<EOF
-- ready. Development candidate command from $PWD:
  sudo $ADMITTED --worker-config $WORKER_JSON --arch x86_64 \\
    --out $OUTPUT_PARENT/manual-01 --development --target candidate \\
    --forgejo-source $FORGEJO_SOURCE
-- installer rootfs pickup: $ROOTFS_DIR (served by soda-rootfs-server.service)
  media builds file their hash-named rootfs here: pass --rootfs-dir $ROOTFS_DIR
EOF
if [ -n "$ROOTFS_URL" ]; then
  printf '%s\n' "-- guests fetch the filed rootfs from: $ROOTFS_URL"
fi

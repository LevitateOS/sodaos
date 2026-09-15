#!/bin/bash
# setup-soda-candidate.sh prepares this machine so `sudo soda-candidate`
# runs without a coding agent: wrapper on sudo's PATH, admitted controller,
# worker directories, restricted worker config, and a fixture-only media
# authority. Development and fixture scope only: it never creates
# qualification or signing configs, and the generated keys must never
# stand in for release keys.
set -euo pipefail
cd "$(dirname "$0")/.."

PREFIX="${SODA_REPOSITORY_PREFIX:-ghcr.io/levitateos/sodaos}"
REFRESH="${SODA_REFRESH_AUTHORITY:-0}"
# Pickup folder for the built system image. The ISO is only the boot menu;
# the installer downloads the big rootfs file from the address below.
ROOTFS_DIR="/var/lib/soda-rootfs"
ROOTFS_URL="http://127.0.0.1:8080"
ADMITTED="/usr/local/lib/soda/soda-build"
PINNED_GO="/usr/local/lib/soda/pinned-go"
WRAPPER="/usr/sbin/soda-candidate"
OUTPUT_PARENT="$PWD/.artifacts/releases/isolated"
BUILD_HOME="/var/lib/soda-candidate-home"
RUNTIME="/var/lib/soda-candidate-run"
TOOLS="/var/lib/soda-candidate-tools"
AUTHORITY="/var/lib/soda-candidate-authority"
WORKER_JSON="$AUTHORITY/worker.json"

fail() { printf 'setup-soda-candidate: %s\n' "$*" >&2; exit 1; }

[ -f go.mod ] || fail "run from the repository root"
[ "$(uname -m)" = "x86_64" ] || fail "matching native x86_64 required on this host"
command -v go bun podman skopeo python3 >/dev/null || fail "pinned go, bun, podman, skopeo and python3 required"
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

echo "-- build tools from committed source"
BINDIR="$(mktemp -d)"
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
# The pinned GOROOT installs at a fixed host path instead of the bound
# tools dir so it carries lib_t and stays executable for the worker
# domain, and /usr/local stays readable under ProtectSystem=strict. The
# single-file bun binary tolerates the bind, so it stays in $TOOLS.
sudo rm -rf "$TOOLS/go" "$PINNED_GO" "$TOOLS/bin/bun"
sudo cp -a "$PINNED_GOROOT" "$PINNED_GO"
sudo cp "$(command -v bun)" "$TOOLS/bin/bun"
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
sudo env HOME="$BUILD_HOME" GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org GOMODCACHE="$BUILD_HOME/go-mod" GOCACHE="$BUILD_HOME/go-build" GOTOOLCHAIN="go$PINNED" GOFLAGS=-mod=readonly CGO_ENABLED=0 "$PINNED_GO/bin/go" build ./... || fail "cannot warm Go module cache"
sudo -u soda-build-worker env HOME="$BUILD_HOME" "$PINNED_GO/bin/go" env -w GOPROXY=off || fail "cannot lock worker Go offline"
# Bun installs only from its HOME cache inside: warm it from a scratch copy
# (mirroring the workspaces list) so node_modules never lands in the
# checkout. Bun refuses files owned by another user, so the scratch tree
# belongs to the worker first.
TMPW="$(mktemp -d)"
mkdir -p "$TMPW/tools"
cp package.json bun.lock bunfig.toml "$TMPW/"
cp -a tools/lit-check "$TMPW/tools/"
sudo chown -R soda-build-worker:soda-build-worker "$TMPW"
sudo find "$TMPW" -type d -exec chmod 0755 {} +
(cd "$TMPW" && sudo -u soda-build-worker env HOME="$BUILD_HOME" "$TOOLS/bin/bun" install --frozen-lockfile >/dev/null) || fail "cannot warm Bun cache"
sudo rm -rf "$TMPW"
sudo chown -R soda-build-worker:soda-build-worker "$BUILD_HOME"

echo "-- worker SELinux policy (process groups plus Go cache mapping)"
command -v checkmodule semodule_package semodule >/dev/null || fail "policycoreutils tooling required for the worker SELinux module"
sudo semodule -r soda-build-setpgid 2>/dev/null || true
checkmodule -M -m -o "$BINDIR/soda-build-worker.mod" scripts/selinux/soda-build-worker.te
semodule_package -o "$BINDIR/soda-build-worker.pp" -m "$BINDIR/soda-build-worker.mod"
sudo semodule -i "$BINDIR/soda-build-worker.pp"
# Go runs as init_t in the worker and mmaps its cache files; the default
# var_lib_t home withholds map, so the Go state dirs carry a dedicated
# type. Podman storage, the runtime dir and frontend caches keep theirs.
sudo mkdir -p "$BUILD_HOME/go-build" "$BUILD_HOME/go-mod" "$BUILD_HOME/.config"
if command -v semanage >/dev/null; then
  for d in go-build go-mod .config; do
    sudo semanage fcontext -a -t soda_build_cache_t "$BUILD_HOME/$d(/.*)?" 2>/dev/null || sudo semanage fcontext -m -t soda_build_cache_t "$BUILD_HOME/$d(/.*)?"
  done
fi
command -v restorecon >/dev/null && sudo restorecon -R "$BUILD_HOME/go-build" "$BUILD_HOME/go-mod" "$BUILD_HOME/.config"
sudo stat -c %C "$BUILD_HOME/go-build" | grep -q ":soda_build_cache_t:" || fail "worker Go cache is not soda_build_cache_t; the sandboxed worker could not map it"

echo "-- worker git ownership exception"
if [ ! -f /etc/gitconfig ] || ! grep -qF "directory = /run/soda-build-source" /etc/gitconfig; then
  printf '# Soda build worker: the isolated worker sees the canonical checkout\n# only at /run/soda-build-source, owned by the operator. Mark it expected.\n[safe]\n\tdirectory = /run/soda-build-source\n' | sudo tee -a /etc/gitconfig >/dev/null
  sudo chmod 0644 /etc/gitconfig
fi

echo "-- restricted worker config"
sudo python3 - "$WORKER_JSON" <<EOF
import json, sys
json.dump({
  "Executable": "$ADMITTED",
  "Source": "$PWD",
  "OutputParent": "$OUTPUT_PARENT",
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
  TMPD="$(mktemp -d)"
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
keys = {r: open(f"{tmpd}/{r}.pub").read() for r in roles}
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
sudo chown "$USER" "$ROOTFS_DIR"

cat <<EOF
-- ready. One command from $PWD:
  sudo soda-candidate   (press go)
The wrapper serves $ROOTFS_DIR on $ROOTFS_URL itself during the build and
files the built rootfs image there afterwards. Flags still pre-seed answers:
  sudo soda-candidate --controller $ADMITTED --worker-config $WORKER_JSON \\
    --out $OUTPUT_PARENT/manual-01 --rootfs-base-url $ROOTFS_URL
EOF

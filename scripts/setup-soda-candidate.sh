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
export GOTOOLCHAIN=local

echo "-- build tools from committed source"
BINDIR="$(mktemp -d)"
trap 'rm -rf "$BINDIR"' EXIT
go build -o "$BINDIR/soda-build" ./tools/soda-build
go build -o "$BINDIR/soda-candidate" ./tools/soda-candidate

echo "-- install wrapper and admitted controller"
sudo install -m 0755 "$BINDIR/soda-candidate" "$WRAPPER"
sudo install -D -m 0755 "$BINDIR/soda-build" "$ADMITTED"
# Canonicalize: /sbin may be a symlink to /usr/sbin, so compare targets.
GOT="$(sudo readlink -f "$(sudo which soda-candidate)")"
WANT="$(readlink -f "$WRAPPER")"
[ "$GOT" = "$WANT" ] || fail "wrapper not visible on sudo secure_path (got $GOT)"

echo "-- worker directories"
sudo mkdir -p "$OUTPUT_PARENT" "$BUILD_HOME" "$RUNTIME" "$TOOLS/bin" "$AUTHORITY"
# Self-contained tools: the worker runs under ProtectHome=tmpfs and
# ProtectSystem=strict, so symlinks into /usr/local or $HOME would dangle.
# Copy the full GOROOT tree (a bare go binary cannot find its stdlib) and
# the single-file bun binary; the worker bind-mounts only this directory.
sudo rm -rf "$TOOLS/go" "$TOOLS/bin/bun"
sudo cp -a "$(go env GOROOT)" "$TOOLS/go"
sudo cp "$(command -v bun)" "$TOOLS/bin/bun"
sudo chown -R root:root "$TOOLS"
sudo chown soda-build-worker:soda-build-worker "$OUTPUT_PARENT" "$BUILD_HOME" "$RUNTIME"
sudo chmod 0755 "$TOOLS" "$TOOLS/bin"
sudo chmod 0700 "$AUTHORITY"

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

sudo mkdir -p "$ROOTFS_DIR"
sudo chown "$USER" "$ROOTFS_DIR"

cat <<EOF
-- ready. The rootfs URL defaults to $ROOTFS_URL in the TUI.
Next, from $PWD:
  1. python3 -m http.server 8080 --directory $ROOTFS_DIR &
  2. sudo soda-candidate   (press go; the pickup address is prefilled)
  3. cp <out>/artifacts/media/*-rootfs.img $ROOTFS_DIR/
     so the installer can download the built system image.
Or pass everything as flags:
  sudo soda-candidate --controller $ADMITTED --worker-config $WORKER_JSON \\
    --out $OUTPUT_PARENT/manual-01 --rootfs-base-url $ROOTFS_URL
EOF

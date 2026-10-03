#!/bin/bash
# rotate-lab-creds.sh — lab credential inventory and owner-gated rotation (B7).
#
# Context: the D3 rootfs HTTP exposure served live guest disks and images
# world-readable over HTTP. Any secret that ever lived in those bytes must
# be treated as exposed until the owner rotates it. Rotation itself is an
# explicit owner decision: this script defaults to read-only inventory and
# prints runbooks; only --execute with SODA_ROTATE_ACK=<class> mutates, and
# only for fully scriptable classes. Secrets travel via 0600 files, never
# argv, environment values in logs, or stdout.
set -euo pipefail
umask 077

CLASS="${1:-inventory}"
case "$CLASS" in
  -h|--help|help) CLASS="help" ;;
esac
EXECUTE=0
if [ "${2:-}" = "--execute" ] || [ "${3:-}" = "--execute" ]; then EXECUTE=1; fi

fail() { printf 'rotate-lab-creds: %s\n' "$*" >&2; exit 1; }
note() { printf '%-7s %s\n' "$1" "$2"; }

# stat_one prints mode/owner/size/mtime for a path, never content.
stat_one() {
  local path="$1" want="$2"
  if [ ! -e "$path" ]; then note SKIP "$path absent or not visible to $USER"; return 0; fi
  local st
  if ! st="$(stat -c '%a %U:%G %s %y' "$path" 2>/dev/null)"; then
    note SKIP "$path unreadable (run with read access)"; return 0
  fi
  if [ "$(stat -c '%a' "$path")" = "$want" ]; then
    note PASS "$path [$st]"
  else
    note WARN "$path mode is $(stat -c '%a' "$path"), want $want [$st]"
  fi
}

inventory() {
  echo "-- fixture-only media authority (dev scope; regenerable)"
  for f in artifact.private passphrase config.json trust.json; do
    stat_one "/var/lib/soda-candidate-authority/$f" 600
  done
  stat_one "/var/lib/soda-candidate-authority/worker.json" 600
  echo "-- cloudflared tunnel credentials (dashboard-issued)"
  stat_one "/etc/cloudflared/dimensionlab-forgejo-https.token" 640
  stat_one "/etc/cloudflared/dimensionlab-forgejo-https.id" 640
  echo "-- forgejo runner registration (host service config)"
  stat_one "$HOME/containers/forgejo-runner/data/config.yaml" 600
  stat_one "$HOME/containers/forgejo-runner/data/.runner" 600
  echo "-- build metadata (public pins only; informational)"
  for f in "$PWD"/.artifacts/releases/isolated/soda-live-inputs-*.json; do
    [ -e "$f" ] || continue
    note INFO "$f carries public URLs/hashes only; 0644 is expected"
    break
  done
  echo "-- D3 exposure reminder"
  note INFO "served QCOW2/ISO/rootfs bytes are 0644-or-readable over HTTP;"
  note INFO "guest runtime secrets inside them are UNKNOWN until rotated."
  echo "inventory complete; no values printed, nothing mutated."
}

runbook_fixture_authority() {
  cat <<'EOF'
-- runbook: fixture-authority (fully scriptable with --execute)
Regenerates the fixture-only media authority (never release keys):
  SODA_ROTATE_ACK=fixture-authority scripts/ops/rotate-lab-creds.sh --rotate fixture-authority --execute
Effect: old fixture signatures stop verifying; in-flight development
attempts using the old authority fail closed and must rerun setup.
EOF
}

runbook_cloudflared() {
  cat <<'EOF'
-- runbook: cloudflared-token (owner-manual; dashboard-issued)
1. Rotate the tunnel token in the Cloudflare dashboard.
2. As root, install it with secret-file handling only:
     install -m 0640 -o root -g cloudflared /path/to/new.token \
       /etc/cloudflared/dimensionlab-forgejo-https.token
   Never pass the token on a command line; shred the staging copy after.
3. Restart the tunnel service and re-run this script (inventory).
EOF
}

runbook_runner() {
  cat <<'EOF'
-- runbook: forgejo-runner (owner-manual; needs a Forgejo admin token)
1. Revoke the runner registration in Forgejo and create a new token.
2. Stop the runner, replace the secret file (0600) without argv exposure,
   re-register, and restart the runner.
3. Re-run this script (inventory) to confirm modes.
EOF
}

runbook_lab_vm() {
  cat <<'EOF'
-- runbook: lab-vm-operator (owner-manual; protected VMs)
Guest operator credentials possibly baked into the served QCOW2s cannot be
audited from the host (live disks are never mounted here). After the D3
replacement is installed, rotate operator/SSH material from the guest
consoles, then record the rotation date with the owner.
EOF
}

rotate_fixture_authority() {
  [ "$EXECUTE" = 1 ] || { runbook_fixture_authority; return 0; }
  [ "${SODA_ROTATE_ACK:-}" = "fixture-authority" ] \
    || fail "refusing: set SODA_ROTATE_ACK=fixture-authority to execute"
  command -v skopeo >/dev/null || fail "skopeo required"
  local authority="/var/lib/soda-candidate-authority" tmpd now
  tmpd="$(mktemp -d)" || fail "cannot stage secrets"
  [ -n "$tmpd" ] && [ -d "$tmpd" ] || fail "cannot stage secrets"
  # Expand now: tmpd is function-local, so a single-quoted trap would
  # evaluate empty after return and retain staging (B7). This removes
  # only this call's own staging directory on success or failure.
  # shellcheck disable=SC2064
  trap "rm -rf -- \"$tmpd\"" EXIT
  chmod 0700 "$tmpd"
  head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' >"$tmpd/passphrase"
  chmod 0600 "$tmpd/passphrase"
  local role
  for role in artifact candidate preview stable; do
    skopeo generate-sigstore-key --output-prefix "$tmpd/$role" \
      --passphrase-file "$tmpd/passphrase" >/dev/null
  done
  now="$(date +%s)"
  SODA_ROTATE_PREFIX="${SODA_REPOSITORY_PREFIX:-ghcr.io/levitateos/sodaos}" \
    python3 - "$tmpd" "$now" <<'EOF'
import json, os, sys
tmpd, now = sys.argv[1], int(sys.argv[2])
roles = ["artifact", "candidate", "preview", "stable"]
keys = {r: [open(f"{tmpd}/{r}.pub").read()] for r in roles}
trust = {
  "Format": 1, "Prefix": os.environ["SODA_ROTATE_PREFIX"], "Epoch": 1,
  "Keys": keys, "NotBefore": now - 600, "MaxAgeSeconds": 3600,
  "ClockSkewSeconds": 10,
  "MinimumSequence": {"candidate": 1, "preview": 1, "stable": 1},
}
open(f"{tmpd}/trust.json", "w").write(json.dumps(trust, indent=2) + "\n")
open(f"{tmpd}/config.json", "w").write(json.dumps({
  "Trust": "/run/soda-media-authority/trust.json",
  "Keys": {"Key": "/run/soda-media-authority/artifact.private",
           "Passphrase": "/run/soda-media-authority/passphrase"},
}, indent=2) + "\n")
EOF
  for f in trust.json artifact.private passphrase config.json; do
    sudo install -m 0600 "$tmpd/$f" "$authority/$f.new"
    sudo mv "$authority/$f.new" "$authority/$f"
  done
  sudo chown soda-build-worker:soda-build-worker "$authority" \
    "$authority/trust.json" "$authority/artifact.private" \
    "$authority/passphrase" "$authority/config.json"
  echo "fixture authority rotated; old fixture signatures no longer verify."
}

case "$CLASS" in
  inventory) inventory ;;
  help|--help|-h)
    sed -n '2,10p' "$0"
    echo "usage: $0 [inventory|--rotate CLASS [--execute]]"
    echo "classes: fixture-authority cloudflared-token forgejo-runner lab-vm-operator"
    ;;
  --rotate)
    case "${2:-}" in
      fixture-authority)
        if [ "$EXECUTE" = 1 ]; then rotate_fixture_authority
        else runbook_fixture_authority; fi ;;
      cloudflared-token) runbook_cloudflared ;;
      forgejo-runner) runbook_runner ;;
      lab-vm-operator) runbook_lab_vm ;;
      *) fail "unknown class '${2:-}'; see --help" ;;
    esac
    ;;
  *) fail "unknown command '$CLASS'; see --help" ;;
esac

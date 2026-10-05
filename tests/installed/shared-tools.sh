#!/bin/bash
# Core U08: real shared installation/files through a routed developer client.
# Install the selected tool as project administrator first; this check never
# installs separate per-user copies. Retains a new exact run-owned shared probe.
set -euo pipefail
: "${SODA_NATIVE_VALIDATE:?Explicit native validation authorization required}"
: "${PROJECT_IP:?}" "${ISOLATION_IP:?Live second-project address required}" "${ALICE:?}" "${BOB:?}" "${SSH_CONFIG:?Pinned private client configuration required}"
[[ "$SODA_NATIVE_VALIDATE" == soda-test ]]
in_subnet() {
  local t=$1 rest
  case "$t" in 10.89.0.*) rest=${t#10.89.0.};; *) return 1;; esac
  case "$rest" in ''|*[!0-9]*) return 1;; esac
  case "$rest" in 0[0-9]*) return 1;; esac
  [ "$rest" -ge 0 ] && [ "$rest" -le 255 ]
}
in_subnet "$PROJECT_IP" && in_subnet "$ISOLATION_IP"
[[ "$ALICE" == u08-alice-8417 && "$BOB" == u08-bob-8417 ]]
remote() { local user=$1; shift; ssh -F "$SSH_CONFIG" -o BatchMode=yes -o ForwardAgent=no "$user@$PROJECT_IP" "$@"; }
a=$(remote "$ALICE" 'mise where node')
b=$(remote "$BOB" 'mise where node')
[[ "$a" == "$b" && "$a" == /opt/mise/installs/node/24.20.0 ]]
# Observe the executable actually running, not only the shim or reported version.
probe='node -p '\''JSON.stringify({path:process.execPath,inode:require("fs").statSync(process.execPath).ino,device:require("fs").statSync(process.execPath).dev,uid:require("fs").statSync(process.execPath).uid,version:process.version})'\'''
a=$(remote "$ALICE" "$probe")
b=$(remote "$BOB" "$probe")
[[ "$a" == "$b" ]]
case "$a" in *'"path":"/opt/mise/installs/node/24.20.0/bin/node"'*) ;; *) echo 'shared node identity mismatch' >&2; exit 1;; esac
case "$a" in *'"uid":0'[},]*) ;; *) echo 'shared node identity mismatch' >&2; exit 1;; esac
case "$a" in *'"version":"v24.20.0"'*) ;; *) echo 'shared node identity mismatch' >&2; exit 1;; esac
remote "$BOB" 'sh -s' <<'SH'
for p in /opt/mise/installs/node/24.20.0/bin/node /opt/mise/installs/node/24.20.0/bin /opt/mise/installs/node/24.20.0 /opt/mise/installs/node /opt/mise/installs /opt/mise /opt / /opt/mise/shims /etc/mise /etc/mise/config.toml; do
  if [ -w "$p" ]; then echo "ordinary member can replace $p"; exit 1; fi
done
# O_WRONLY open attempt without truncation or write, even on unexpected access.
if err=$(LC_ALL=C sh -c 'exec 3>>/opt/mise/installs/node/24.20.0/bin/node' 2>&1); then
  echo 'ordinary member obtained write access to administrator tool'; exit 1
fi
printf '%s' "$err" | grep -q 'Permission denied' || exit 1
SH
run="u08-shared-$(od -A n -t x1 -N 8 /dev/urandom | tr -d ' \n')"
remote "$ALICE" "umask 0002; mkdir -m2770 /srv/project/shared/$run; printf '%s\n' alice > /srv/project/shared/$run/members"
remote "$BOB" "test \"\$(readlink -f ~/shared/$run/members)\" = /srv/project/shared/$run/members; grep -qx alice ~/shared/$run/members; printf '%s\n' bob >> ~/shared/$run/members"
expected=$'alice\nbob'
[[ $(remote "$ALICE" "head -n 2 ~/shared/$run/members") == "$expected" ]]
[[ $(remote "$ALICE" "stat -c '%d:%i' ~/shared/$run/members") == $(remote "$BOB" "stat -c '%d:%i' ~/shared/$run/members") ]]
# The second project's independently verified host key is in the client config.
ssh -F "$SSH_CONFIG" -o BatchMode=yes -o ForwardAgent=no "$BOB@$ISOLATION_IP" "test ! -e /srv/project/shared/$run"
printf 'Shared root-owned Node executable/inode and member write denial verified.\nShared file updates/inode verified for both users; absent in second project.\nRetained probe: /srv/project/shared/%s\n' "$run"

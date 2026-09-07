#!/bin/bash
# Core U08: real shared installation/files through a routed developer client.
# Install the selected tool as project administrator first; this check never
# installs separate per-user copies. Retains a new exact run-owned shared probe.
set -euo pipefail
: "${SODA_NATIVE_VALIDATE:?Explicit native validation authorization required}"
: "${PROJECT_IP:?}" "${ALICE:?}" "${BOB:?}" "${SSH_CONFIG:?Pinned private client configuration required}"
[[ "$SODA_NATIVE_VALIDATE" == soda-test ]]
python3 -c 'import ipaddress,sys; assert ipaddress.ip_address(sys.argv[1]) in ipaddress.ip_network("10.89.0.0/24")' "$PROJECT_IP"
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
python3 -c 'import json,sys; p=json.loads(sys.argv[1]); assert p["path"]=="/opt/mise/installs/node/24.20.0/bin/node" and p["uid"]==0 and p["version"]=="v24.20.0"' "$a"
remote "$BOB" 'python3 -' <<'PY'
import errno, os
from pathlib import Path
binary=Path('/opt/mise/installs/node/24.20.0/bin/node')
for p in [binary,*binary.parents,Path('/opt/mise/shims'),Path('/etc/mise'),Path('/etc/mise/config.toml')]:
    assert not os.access(p,os.W_OK), 'Ordinary member can replace shared tool/configuration'
try:
    fd=os.open(binary,os.O_WRONLY)  # No truncation/write, even on unexpected access.
except OSError as error:
    assert error.errno==errno.EACCES
else:
    os.close(fd)
    raise AssertionError('Ordinary member obtained write access to administrator tool')
PY
run="u08-shared-$(python3 -c 'import secrets; print(secrets.token_hex(8))')"
remote "$ALICE" "umask 0002; mkdir -m2770 /srv/project/shared/$run; printf '%s\n' alice > /srv/project/shared/$run/members"
remote "$BOB" "test \"\$(readlink -f ~/shared/$run/members)\" = /srv/project/shared/$run/members; grep -qx alice ~/shared/$run/members; printf '%s\n' bob >> ~/shared/$run/members"
expected=$'alice\nbob'
[[ $(remote "$ALICE" "head -n 2 ~/shared/$run/members") == "$expected" ]]
[[ $(remote "$ALICE" "stat -c '%d:%i' ~/shared/$run/members") == $(remote "$BOB" "stat -c '%d:%i' ~/shared/$run/members") ]]
# The second project's independently verified host key is in the client config.
ssh -F "$SSH_CONFIG" -o BatchMode=yes -o ForwardAgent=no "$BOB@10.89.0.3" "test ! -e /srv/project/shared/$run"
printf 'Shared root-owned Node executable/inode and member write denial verified.\nShared file updates/inode verified for both users; absent in second project.\nRetained probe: /srv/project/shared/%s\n' "$run"

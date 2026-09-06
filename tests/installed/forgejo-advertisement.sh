#!/bin/bash
# P11: explicit advertisement mutation, using the existing helper only.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Explicit host required} == "$(hostname)" && $(id -u) == 0 ]]
[[ ${SODA_ALLOW_FORGEJO_ADVERTISEMENT_REFRESH:-} == 1 ]] || { echo 'Advertisement refresh permission not selected' >&2; exit 1; }
origins() {
  python3 - <<'PY'
import json
x=json.load(open('/etc/soda/dashboard.json'))
print(json.dumps({k:x[k] for k in ('public_url','forgejo_url','forgejo_internal_url')},sort_keys=True))
PY
}
/usr/bin/tailscale status --json | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d.get("BackendState")=="Running" and not (d.get("Self") or {}).get("Expired",False), "approved running Tailnet required; no enrollment is performed"'
before=$(origins)
/usr/local/libexec/soda/soda-forgejo-tailnet
[[ "$(origins)" == "$before" ]] || { echo 'Core browser origins changed during advertisement refresh' >&2; exit 1; }
python3 - <<'PY'
import subprocess
values={}
for line in open('/etc/soda/forgejo.env'):
    if '=' in line and not line.startswith('#'):
        key,value=line.rstrip('\n').split('=',1); values[key]=value
host=values['FORGEJO__server__SSH_DOMAIN']
port=values['FORGEJO__server__SSH_PORT']
status=__import__('json').loads(subprocess.check_output(['/usr/bin/tailscale','status','--json'],text=True))
assert host in status.get('TailscaleIPs',[]), 'advertisement does not match a current Tailnet address'
listeners=subprocess.check_output(['podman','--remote=false','port','soda-forgejo','22/tcp'],text=True).splitlines()
assert f'{host}:{port}' in listeners, 'advertised endpoint is not a real selected private listener'
print('Listener-checked Git SSH advertisement:',host,port)
print('Core browser origins unchanged. Probe the pinned endpoint from the intended client; this is not routed Git authentication proof.')
PY

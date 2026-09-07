#!/bin/bash
# P11: retained native observations only. No enrollment/runner mutation.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Explicit host required} == "$(hostname)" && $(id -u) == 0 ]]
[[ $(getenforce) == Enforcing ]]
/usr/bin/tailscale status --json | python3 -c 'import json,sys; d=json.load(sys.stdin); assert isinstance(d.get("BackendState"),str); print(json.dumps({"BackendState":d["BackendState"],"Expired":(d.get("Self") or {}).get("Expired")}))'
printf '{}\n' | /usr/local/libexec/soda/soda-runners list | python3 -c 'import json,sys; d=json.load(sys.stdin); assert isinstance(d.get("runner_count"),int); print(json.dumps({k:d[k] for k in ("runner_count","active_listeners","total_capacity")})); print(json.dumps([{k:r[k] for k in ("ID","Provider","Account","Architecture","version","capacity","service")} for r in d["runners"]]))'
/usr/bin/tailscale version
/usr/bin/forgejo-runner --version
[[ -s /etc/cockpit/branding/favicon.ico && -s /etc/cockpit/branding/branding.css ]]
[[ -s /var/lib/soda/forgejo/gitea/public/assets/img/logo.svg ]]
[[ -s /usr/local/share/cockpit/soda-tailscale/index.html && -s /usr/local/share/cockpit/soda-runners/index.html ]]
[[ ! -e /usr/local/share/cockpit/soda-updates ]]
# Test the delivered noninteractive hook without a login, TTY or MOTD trace.
quiet=$(bash --noprofile --norc -ec 'source /etc/profile.d/soda-console-welcome.sh; printf sentinel')
[[ "$quiet" == sentinel ]]
printf 'Retained native read-only observations completed. Enrollment, runner jobs, interactive console and personal provider authentication remain separate.\n'

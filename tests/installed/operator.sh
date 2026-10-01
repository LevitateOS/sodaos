#!/bin/bash
# P11: retained native observations only. No enrollment/runner mutation.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Explicit host required} == "$(hostname)" && $(id -u) == 0 ]]
[[ $(getenforce) == Enforcing ]]
/usr/bin/tailscale status --json | python3 -c 'import json,sys; d=json.load(sys.stdin); assert isinstance(d.get("BackendState"),str); print(json.dumps({"BackendState":d["BackendState"],"Expired":(d.get("Self") or {}).get("Expired")}))'
/usr/bin/tailscale version
[[ -s /etc/cockpit/branding/favicon.ico && -s /etc/cockpit/branding/branding.css ]]
[[ -s /var/lib/soda/forgejo/gitea/public/assets/img/logo.svg ]]
[[ ! -e /usr/local/share/cockpit/soda-tailscale && ! -L /usr/local/share/cockpit/soda-tailscale ]]
[[ ! -e /usr/local/share/cockpit/soda-runners && ! -L /usr/local/share/cockpit/soda-runners ]]
[[ ! -e /usr/local/share/cockpit/soda-updates ]]
# Test the delivered noninteractive hook without a login, TTY or MOTD trace.
quiet=$(bash --noprofile --norc -ec 'source /etc/profile.d/soda-console-welcome.sh; printf sentinel')
[[ "$quiet" == sentinel ]]
printf 'Retained native read-only observations completed. Enrollment, runner jobs, interactive console and personal provider authentication remain separate.\n'

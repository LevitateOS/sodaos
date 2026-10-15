#!/bin/bash
# P11: explicit advertisement mutation, using the existing helper only.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Explicit host required} == "$(hostname)" && $(id -u) == 0 ]]
[[ ${SODA_ALLOW_FORGEJO_ADVERTISEMENT_REFRESH:-} == 1 ]] || { echo 'Advertisement refresh permission not selected' >&2; exit 1; }
origins() {
  /usr/libexec/soda/soda-host-probes forgejo-origins
}
curl --fail --silent --show-error --unix-socket /run/soda-tailnet/host/control/tailscaled.sock http://local-tailscaled.sock/localapi/v0/status | /usr/libexec/soda/soda-host-probes forgejo-tailnet
before=$(origins)
/usr/libexec/soda/soda-forgejo-tailnet
[[ "$(origins)" == "$before" ]] || { echo 'Core browser origins changed during advertisement refresh' >&2; exit 1; }
/usr/libexec/soda/soda-host-probes forgejo-advertisement

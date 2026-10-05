#!/bin/bash
# P11: explicit advertisement mutation, using the existing helper only.
set -euo pipefail
[[ ${SODA_NATIVE_VALIDATE:?Explicit host required} == "$(hostname)" && $(id -u) == 0 ]]
[[ ${SODA_ALLOW_FORGEJO_ADVERTISEMENT_REFRESH:-} == 1 ]] || { echo 'Advertisement refresh permission not selected' >&2; exit 1; }
origins() {
  /usr/libexec/soda/soda-host-probes forgejo-origins
}
/usr/bin/tailscale status --json | /usr/libexec/soda/soda-host-probes forgejo-tailnet
before=$(origins)
/usr/local/libexec/soda/soda-forgejo-tailnet
[[ "$(origins)" == "$before" ]] || { echo 'Core browser origins changed during advertisement refresh' >&2; exit 1; }
/usr/libexec/soda/soda-host-probes forgejo-advertisement

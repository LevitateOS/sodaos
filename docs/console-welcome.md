# Operator console welcome

The predecessor's read-only welcome and main-table/NetworkManager uplink discovery are adapted in `appliance/bin/soda-console-welcome`. Staging installs it under `/usr/local/libexec/soda/` and installs an interactive-only `/etc/profile.d/` hook. The renderer additionally requires the native root UID; it is not developer onboarding or a guest agent. The existing `soda-tailnet` command gains a native CLI symlink for this caller.

The message shows observed local uplink IPv4 addresses, loopback-first Cockpit tunnel guidance, and only the public HTTPS origins from `/etc/soda/dashboard.json`. It does not dump configuration/credentials, guess Forgejo's old port, claim a configured origin is listening, change firewall policy or enroll Tailscale. The optional configuration-file argument supports explicit operator inspection; the installed hook uses the default file.

The hook does not emit a banner into noninteractive SSH/SCP/SFTP streams and is installed on the host, not in project homes. Missing configuration produces setup guidance rather than a fabricated service URL. Existing installations are not updated by editing this source; applying a changed hook/configuration remains an explicitly authorized native operation.

Command-double/staging checks passed in the `8b823db` native build/check. The
U08 closure run found that the existing VM lacks the profile hook: source/staging
success did not install it. `tests/installed/operator.sh` now fails on a missing
or failing hook rather than allowing a subsequent sentinel to hide the failure;
its focused missing/failing/noisy/quiet regression passed. Native console delivery
and interactive verification remain outstanding, not a passing full operator
journey. On an approved rollout, install the matching renderer/hook and verify
root interactive output, configured URLs, no secret leakage and quiet
noninteractive transfers separately from the [native product journey](native-validation.md).

# Operator console welcome

The predecessor's read-only welcome and main-table/NetworkManager uplink discovery are adapted in `appliance/bin/soda-console-welcome`. Staging installs it under `/usr/local/libexec/soda/` and installs an interactive-only `/etc/profile.d/` hook. The renderer additionally requires the native root UID; it is not developer onboarding or a guest agent. The existing `soda-tailnet` command gains a native CLI symlink for this caller.

The message shows observed local uplink IPv4 addresses, the direct Cockpit URL on the first observed address, dashboard tunnel guidance with the first observed address filled into the SSH commands (never a direct `http://IP:port` claim: the dashboard listener is loopback-only), the dashboard's configured loopback port, and only the configured Forgejo/Sodaspaces HTTPS origin from `/etc/soda/dashboard.json`. Before operator setup exists it also shows the native Forgejo installer's loopback listener (`http://127.0.0.1:3000`, grounded in the shipped `forgejo.container` publish port and the setup tunnel) with its tunnel command instead of a bare "complete setup" line. It does not dump configuration/credentials, claim a configured origin is listening, change firewall policy or enroll Tailscale. The optional configuration-file argument supports explicit operator inspection; the installed hook uses the default file.

The installed system also renders this message before login. `soda-console.service` (`appliance/services/soda-console.service`, enabled in `appliance/provisioning/candidate.json`) runs after `network-online.target` and before `getty@tty1.service`, writing the renderer output to `/etc/issue.d/50-soda.issue`, and enables and starts the loopback Forgejo installer so the advertised `:3000` tunnel works on first boot without manual service steps. A unit failure never blocks the login prompt (ordering only, no `Requires`). `soda-activate` enables `forgejo.service`, `soda-dashboard.service` and `soda-proxy.service` so activated listeners survive reboot; previously it only started them.

The hook does not emit a banner into noninteractive SSH/SCP/SFTP streams and is installed on the host, not in project homes. Missing configuration produces setup guidance rather than a fabricated service URL. Existing installations are not updated by editing this source; applying a changed hook/configuration remains an explicitly authorized native operation.

Command-double/staging checks passed in the `8b823db` native build/check. The
U08 closure run found that the existing VM lacks the profile hook: source/staging
success did not install it. `tests/installed/operator.sh` now fails on a missing
or failing hook rather than allowing a subsequent sentinel to hide the failure;
its focused missing/failing/noisy/quiet regression passed. Native console delivery
and interactive verification remain outstanding, not a passing full operator
journey. On an approved rollout, install the matching renderer/hook and verify
root interactive output, configured URLs, no secret leakage and quiet
noninteractive transfers separately from the [native product journey](../development/testing.md).

# Operator Cockpit source port

Only Tailnet/Tailscale and Runners are retained. Common PatternFly/Cockpit components, frontend tooling and their upstream licenses are copied from predecessor commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. Projects/People/Updates entrypoints are excluded. Dependency manifests/lockfile are retained. Native x86_64 bundles and source checks have now run; see [local testing](local-testing.md) for installed evidence and remaining gaps.

## Native delivery candidate

Use native Cockpit packages layered onto the upstream Fedora CoreOS deployment, following its [OS extension mechanism](https://docs.fedoraproject.org/en-US/fedora-coreos/os-extensions/). Layering requires an explicitly authorized reboot; it is provisioning, not a Soda update/release platform. Package installation and initial Cockpit access have been exercised on the isolated x86_64 VM, not on all supported targets.

Install the native PAM configuration from `appliance/config/cockpit.pam`, explicitly allow root by replacing `/etc/cockpit/disallowed-users` with an empty file, and retain normal PAM authentication. Only UID 0 may pass the account check. Developer/project-owner identities do not get host Cockpit access. The socket binds loopback by default; use an operator SSH tunnel to port 9090 unless private access is configured explicitly through native tooling.

Preserve Fedora's native Cockpit PAM stack when applying the UID-0 account restriction. In particular, `pam_selinux.so close` must be the first session rule, and `pam_selinux.so open env_params` must precede sessions executed in the operator's context. Omitting these rules leaves authenticated root in `cockpit_session_t`, causing SELinux denials for Tailscale's socket and ordinary systemd administration. After a PAM correction, log out and back in; reloading a page does not change an existing session context. Do not widen socket permissions or disable SELinux to hide this failure.

Branding is installed under `/etc/cockpit/branding`, a native configuration branding root verified in Cockpit 366's `src/ws/cockpitbranding.c`. Staging rewrites only the palette import to the local staged filename and copies the canonical logos/theme/backgrounds; source artwork is unchanged. Extension bundles install under `/usr/local/share/cockpit`, avoiding writes to immutable `/usr/share` outside native package layering.

The retained pages still use the actual Cockpit bridge/native API. They are not replacements implemented in the Soda dashboard. On the installed target, verify root login, denial of non-operator host accounts, package discovery, command execution and branding separately from source completion.

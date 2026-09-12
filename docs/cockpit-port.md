# Operator Cockpit source port

Tailnet/Tailscale is the retained Soda Cockpit extension. Runners presentation is
retired in source; its native Forgejo owner is `/?soda-view=runners`. Installed
fallback removal is tracked separately in the [handoff](implementation-status.md). Common PatternFly/Cockpit components, frontend tooling and their upstream licenses are copied from predecessor commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. Projects/People/Updates entrypoints are excluded. Dependency manifests/lockfile are retained. Native x86_64 bundles and source checks have now run; see [local testing](local-testing.md) for installed evidence and remaining gaps.

## Selected Tailnet move and stock administration

The user selected the [Tailnet implementation plan](tailnet-integration-plan.md):
native dashboard host controls and automatic project enrollment, followed by
retirement of this last Soda Cockpit extension. It is planned, not implemented or
installed. The plan owns parity, source/workspace removal and per-target delivery;
this guide continues to own ordinary Cockpit configuration below. Existing source
and installed fallbacks remain until their respective acceptance/removal stages.

“Stock” here removes Soda extension pages, not root-only PAM/private access, native
SELinux transitions, branding or the existing Accounts navigation policy. No native
Cockpit package removal or security-policy reset is selected by the UI move.

## Native delivery candidate

Use native Cockpit packages layered onto the upstream Fedora CoreOS deployment, following its [OS extension mechanism](https://docs.fedoraproject.org/en-US/fedora-coreos/os-extensions/). Layering requires an explicitly authorized reboot; it is provisioning, not a Soda update/release platform. Package installation and initial Cockpit access have been exercised on the isolated x86_64 VM, not on all supported targets.

Install the native PAM configuration from `appliance/config/cockpit.pam`, explicitly allow root by replacing `/etc/cockpit/disallowed-users` with an empty file, and retain normal PAM authentication. Only UID 0 may pass the account check. Developer/project-owner identities do not get host Cockpit access. The socket binds loopback by default; use an operator SSH tunnel to port 9090 unless private access is configured explicitly through native tooling.

Preserve Fedora's native Cockpit PAM stack when applying the UID-0 account restriction. In particular, `pam_selinux.so close` must be the first session rule, and `pam_selinux.so open env_params` must precede sessions executed in the operator's context. Omitting these rules leaves authenticated root in `cockpit_session_t`, causing SELinux denials for Tailscale's socket and ordinary systemd administration. After a PAM correction, log out and back in; reloading a page does not change an existing session context. Do not widen socket permissions or disable SELinux to hide this failure.

Branding is installed under `/etc/cockpit/branding`, a native configuration branding root verified in Cockpit 366's `src/ws/cockpitbranding.c`. Staging rewrites only the palette import to the local staged filename and copies the canonical logos/theme/backgrounds; source artwork is unchanged. Extension bundles install under `/usr/local/share/cockpit`, avoiding writes to immutable `/usr/share` outside native package layering.

The stock **Accounts** navigation entry is hidden with `/etc/cockpit/users.override.json`, using Cockpit's native [manifest override](https://docs.cockpit-project.org/cockpit-guide/latest/guide/packages.html#package-manifest-override) mechanism. It removes only `users.menu.index`: the upstream package, host accounts, native account tools and all other operator pages remain intact. This is navigation cleanup, not an authorization boundary. Soda people belong in the dashboard/Forgejo, and developer Linux accounts belong inside projects. Log out and back in if an existing Cockpit session still displays its cached Accounts entry.

The retained Tailnet page still uses the actual Cockpit bridge/native API. It is not a replacement implemented in the Soda dashboard. On the installed target, verify root login, denial of non-operator host accounts, package discovery, command execution and branding separately from source completion.

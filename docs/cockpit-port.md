# Operator Cockpit source port

All Soda custom pages are retired in the source candidate, including Tailnet and
Runners. Their Forgejo owners are `/?soda-view=tailnet` and `/?soda-view=runners`.
The custom workspace/build and unused React/PatternFly/Zustand dependencies are
removed; `cockpit/vendor/` retains attribution and license provenance, not an app.
Both installed fallbacks remain on the retained targets in the
[handoff](implementation-status.md). Source removal is not installed retirement.

## Selected Tailnet move and stock administration

The user selected the [Tailnet implementation plan](tailnet-integration-plan.md):
native dashboard host controls and automatic project enrollment, followed by
retirement of the last Soda Cockpit extension. Dashboard host controls, project
runtime/UI and the stock-only source candidate are implemented. The
[x86_64 candidate is built/exported](implementation-status.md#latest-built-source-candidate);
native acceptance and authorized per-target delivery/removal remain pending.
The plan owns those stages; this guide owns ordinary Cockpit configuration below.
The user's source-completion instruction did not authorize installed fallback removal.

The selected end state is **Soda-branded Cockpit with stock pages only**. Branding
must follow Soda's current design, not freeze the older identity-kit presentation.
Use supported upstream branding/theme mechanisms for the login and shell; retain
native administration pages, controls and behavior. Do not introduce replacement
pages or keep a Soda extension merely to supply branding. Review the existing
branding assets against the current design and validate the actual installed
Cockpit version; shared palette imports alone do not prove visual alignment.

“Stock” here removes all Soda extension pages, not root-only PAM/private access,
native SELinux transitions, branding or the existing Accounts navigation policy.
No native Cockpit package removal or security-policy reset is selected by the UI move.

## Native delivery candidate

Use native Cockpit packages layered onto the upstream Fedora CoreOS deployment, following its [OS extension mechanism](https://docs.fedoraproject.org/en-US/fedora-coreos/os-extensions/). Layering requires an explicitly authorized reboot; it is provisioning, not a Soda update/release platform. Package installation and initial Cockpit access have been exercised on the isolated x86_64 VM, not on all supported targets.

Install the native PAM configuration from `appliance/config/cockpit.pam`, explicitly allow root by replacing `/etc/cockpit/disallowed-users` with an empty file, and retain normal PAM authentication. Only UID 0 may pass the account check. Developer/project-owner identities do not get host Cockpit access. The socket binds loopback by default; use an operator SSH tunnel to port 9090 unless private access is configured explicitly through native tooling.

Preserve Fedora's native Cockpit PAM stack when applying the UID-0 account restriction. In particular, `pam_selinux.so close` must be the first session rule, and `pam_selinux.so open env_params` must precede sessions executed in the operator's context. Omitting these rules leaves authenticated root in `cockpit_session_t`, causing SELinux denials for Tailscale's socket and ordinary systemd administration. After a PAM correction, log out and back in; reloading a page does not change an existing session context. Do not widen socket permissions or disable SELinux to hide this failure.

Branding installs under `/etc/cockpit/branding`, the native configuration root
reviewed in Cockpit 366's `src/ws/cockpitbranding.c`. Its login, shell and stock
Overview load the upstream branding stylesheet. The adapter now uses the current
canonical red/white/near-black palette, brutalist symbols, Barlow/Barlow Condensed
and IBM Plex Mono fonts, square controls and native light/dark ownership. It follows
the current sign-in/administration language, not the marketing homepage artwork.
Native warning/disabled/status/terminal colors remain upstream-owned. Staging copies
licensed fonts, current Apple icon and an ICO containing the canonical favicon PNGs;
old kit backgrounds/wordmarks are not staged. No custom login/script/page is added.
See [branding assets](../assets/branding/cockpit/README.md) for the source-check scope.
New bundles forbid custom `/usr/local/share/cockpit` payload; no immutable stock
package files are overwritten.

The stock **Accounts** navigation entry is hidden with `/etc/cockpit/users.override.json`, using Cockpit's native [manifest override](https://docs.cockpit-project.org/cockpit-guide/latest/guide/packages.html#package-manifest-override) mechanism. It removes only `users.menu.index`: the upstream package, host accounts, native account tools and all other operator pages remain intact. This is navigation cleanup, not an authorization boundary. Soda people belong in the dashboard/Forgejo, and developer Linux accounts belong inside projects. Log out and back in if an existing Cockpit session still displays its cached Accounts entry.

The installed operator journey now selects stock Overview and `--stock-read-only`;
it retains root/non-root login, SELinux/socket/native CLI, Services/Logs and logout
checks without an advertisement-refresh effect. It rejects Soda custom packages.
It is authored coverage, not proof on a retained target: validate these controls and
actual branding separately during the authorized replacement/removal window.

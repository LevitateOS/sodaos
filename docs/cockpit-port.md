# Operator Cockpit source port

All Soda custom pages are retired in the source candidate, including Tailnet and
Runners. Their Forgejo owners are `/admin?soda-view=tailnet` and `/admin?soda-view=runners`.
The custom workspace/build and unused React/PatternFly/Zustand dependencies are
removed, including the top-level `cockpit/` directory. Its attribution and upstream
provenance now live in [`assets/branding/cockpit/provenance/`](../assets/branding/cockpit/provenance/README.md),
not an app or staging input. Active branding remains in `assets/branding/cockpit/`;
native service/security configuration and installed checks remain in their owners.
Both installed fallbacks remain on the retained targets in the
[handoff](development-handoff.md). Source removal is not installed retirement.

## Selected Tailnet move and stock administration

The user selected the [Tailnet implementation plan](tailnet-integration-plan.md):
native dashboard host controls and automatic project enrollment, followed by
retirement of the last Soda Cockpit extension. Dashboard host controls, project
runtime/UI and the stock-only source candidate are implemented. The
[x86_64 candidate is built/exported](development-handoff.md#latest-built-source-candidate);
the [fresh VM access receipt](implementation-history.md#fresh-tailnet-vm-installation-and-access-smoke)
now includes native root/stock administration and logout checks. Full visual/theme/
keyboard acceptance and authorized retained-target delivery/removal remain pending.
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
native SELinux transitions or branding. Accounts now retains upstream visibility
for root host administration.
No native Cockpit package removal or security-policy reset is selected by the UI move.

## Page recommendations

The previous blanket exclusion of Podman and other administrative extensions is
withdrawn. **Cockpit is the root administrator's host console, not a restricted
copy of the project dashboard.** Ability to change Soda-managed resources is not
by itself a reason to omit a page: Terminal, Services, Storage and Networking
already provide that authority. Preserve the root-only/private access boundary;
explain resource ownership consistently across all administrative tools.

The user selected the baseline below. Provisioning and first-install checks now
include its four additional native packages, and Accounts is no longer hidden.
The [installation receipt](implementation-history.md#native-cockpit-administration-additions)
records additive live activation and native read-only page/PAM checks on the fresh
x86_64 fixture, not validation of every administrative mutation or delivery to older
retained targets. The [earlier review](implementation-history.md#cockpit-administration-recommendation-review)
separates package/source research from that installed evidence.

### Selected root-administration baseline

| Page | Recommendation | Actual administrative purpose and qualification |
| --- | --- | --- |
| Overview | Keep | Host health, clock, hardware information, shutdown/reboot. |
| Logs | Keep | Native journal diagnosis, including failures outside Soda's application logs. |
| Storage | Keep | Disks, filesystems, mounts and capacity; Soda's project UI is not a host storage manager. Destructive changes need the same preservation discipline as CLI changes. |
| Networking | Keep | Native host interfaces and firewall configuration. This is broader than dashboard Tailnet controls; preserve a recovery access path when changing connectivity. |
| Services | Keep | Host services, sockets and timers, including appliance Quadlets and Soda project units. Use their native systemd ownership during maintenance. |
| Podman Containers | Include | Host container, image, pod and volume administration, logs and troubleshooting. Soda does not supply a general host-engine UI. Upstream already supports systemd/Quadlet lifecycle; see the verified mechanism below. |
| Terminal | Keep | General root diagnosis/recovery, not developer terminals or evidence that every graphical tool is redundant. |
| Files (`cockpit-files`, shown as File browser) | Include | Graphical host file inspection, transfer and deliberate configuration maintenance. Root permissions still apply; do not expose private files in support evidence. |
| SELinux | Include | Diagnose policy denials on the enforcing appliance. The ability to change policy is not a reason to hide diagnosis; disabling enforcement or automatically allowing denials is not the recommendation. |
| Diagnostic Reports | Include, with manual collection/export | Collect a native support bundle when needed. Reports can contain private configuration and must be reviewed before sharing; no automatic collection/upload is proposed. |
| Accounts | Show for root administration | Manages host Linux accounts/passwords/keys, not Forgejo identities or project-local accounts. Those different identity domains do not make the page redundant. Showing it does not broaden Cockpit login eligibility. |
| Software updates (OSTree) | Keep | Native CoreOS deployment updates and rollback. It is not an updater for Soda application images. |
| Metrics / hardware details | Keep available | Host resource diagnosis and hardware inventory. Continuous PCP history/recording is a separate resource/retention decision, not implied by the page being present. |

### Role-dependent extensions, not blanket exclusions

| Extension | Recommendation and reason |
| --- | --- |
| Virtual Machines | Include when the appliance is also a VM host. It manages libvirt-backed VMs, a legitimate administrative need distinct from Soda's containers. No libvirt workload has been established on the current fixture; running Soda inside a QEMU VM does not make its parent hypervisor available to the guest's Cockpit. Validate the chosen host's virtualization/storage/network setup when this role is selected. |
| Kernel Dump | Offer for kernel/crash diagnosis. Enabling crash capture entails backend configuration, reserved resources and possibly boot changes; installing a page alone does not establish capture. |
| Session Recording | Explicit opt-in for an audit requirement. Recording introduces retention/access/privacy obligations and can capture credentials, unlike merely providing an interactive administration page. |
| PackageKit updates / Applications installers | Do not substitute these for the selected CoreOS rpm-ostree layering/update path. Cockpit extensions are useful; the installation mechanism must match the host. No compatible general add-on installer has been validated here. |
| Image Builder, directory/HA servers, file-sharing/ZFS or other role-specific extensions | Assess against the actual host role and native backend, rather than exclude them because an operator could change the system. They are not replacements for the existing Soda project workflow. |

### Checked ownership and upstream mechanisms

- [Cockpit's application catalog](https://cockpit-project.org/applications) identifies
  the separate Podman, Files, SELinux, reports, VMs, OSTree and PackageKit roles.
  Fedora 44 package metadata was checked for Podman, Files, SELinux and reports;
  availability is not a successful dependency transaction or native UI test.
- [Published Fedora 44 Podman metadata](https://packages.fedoraproject.org/pkgs/cockpit-podman/cockpit-podman/fedora-44.html)
  lists 123. Its [tagged container caller](https://github.com/cockpit-project/cockpit-podman/blob/123/src/Containers.jsx)
  uses `systemctl` for recognized `PODMAN_SYSTEMD_UNIT` start/stop/restart, rather
  than always bypassing systemd. Its detector includes inactive Quadlets. There is
  no basis for adding a Soda adapter or claiming Quadlet incompatibility here.
- Its [tagged transport](https://github.com/cockpit-project/cockpit-podman/blob/123/src/rest.ts)
  selects the host system or user Podman sockets. It is not automatically a manager
  for the separate engines and stores **inside** Soda projects; see
  [Project OS state/lifecycle](project-os.md#persistent-state-and-lifecycle).
- Soda's appliance services use Quadlets; projects use the explicit
  [`soda-project@.service`](../appliance/services/soda-project@.service) and helper.
  Dashboard project actions also own Soda records, memberships and Tailnet intent.
  A host engine UI does not replace those operations, but remains appropriate for
  deliberate root inspection, recovery and administration of other host workloads.

No new custom Tailnet/Runners pages are proposed: their selected dashboard move
is a separate product decision. `base1`, shell, static, branding and issue assets
remain upstream infrastructure, not additional product pages to remove.

## Native delivery candidate

Use native Cockpit packages layered onto the upstream Fedora CoreOS deployment,
following its [OS extension mechanism](https://docs.fedoraproject.org/en-US/fedora-coreos/os-extensions/).
The selected baseline adds `cockpit-podman`, `cockpit-files`, `cockpit-selinux` and
`cockpit-sosreport` to ordinary system/storage/network/OSTree administration.
[Installation](installation.md#cockpit-addon-maintenance) owns activation and
preservation, including the additive live-update option for existing deployments.
This is not a Soda update/release platform. Native checks cover the recorded
x86_64 fixture, not every supported host or administrative operation.

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

The stock **Accounts** page is visible for host administration. Soda no longer
ships `/etc/cockpit/users.override.json`; the root-only PAM gate remains unchanged.
Forgejo identities and project-local Linux accounts still belong to their own
systems. On an authorized existing target, preserve and inspect the exact old
override before removing Soda's known hiding configuration; do not delete an
operator's unrelated customization. Log out and back in after package/navigation
changes to refresh Cockpit's cached manifests, rather than interrupt every session.

The installed operator journey now selects stock Overview and `--stock-read-only`;
it retains root/non-root authorization, SELinux/socket/native CLI, Services/Logs
and logout checks without an advertisement-refresh effect. It also opens Podman,
File browser, SELinux, Diagnostic reports and Accounts without management writes
or report collection. It rejects Soda custom packages. The receipt records the
checks actually run; full visual/keyboard/theme and destructive-operation acceptance
remain separate from these native access checks.

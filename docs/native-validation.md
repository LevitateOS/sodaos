# Native validation

**Partial x86_64 build and installed evidence; full acceptance remains pending.** [Implementation status](implementation-status.md) records the sealed `8417a90` build, component checks, corrected pinned Go-suite pass and installed core subsets. The complete aggregate native check has not been successfully rerun. [Local testing](local-testing.md) records the existing guest and approved infra-to-project routing; neither is a fresh support-fixture/install proof. See the [native porting audit](native-porting-audit.md) for remaining source defects, tests and native exits. Name the actual builder, installed target, architecture, developer client and permitted actions before additional execution. A build permit is not a disk-install, provider-registration, service-stop or reboot permit. Missing access is **unverified**, not passed or simulated.

**Ownership:** U08/U20 in the [leading core plan](dashboard-implementation-plan.md#core-owned-native-proof-detail) own the browser/developer/workload/persistence journeys and overall product acceptance; individual features own their focused tests. The subordinate [native support plan](native-porting-plan.md) supplies VM/SSH/evidence/artifact/provisioning helpers, P06 host observations and P11 retained operator integrations. P07/P08 redirect to the core rather than create a second suite. Optional media and unfinished support ports do not block using the existing authorized entrypoints.

The [support tools and check entrypoints](native-support.md) have partial build/source-check evidence, but their new VM/remote/transfer/install/operator paths are not all natively exercised. P12/P13 report interfaces do not establish those results. Audited source fixes, a successful exact-candidate aggregate build/check, clean first installation, QEMU/firmware/Ignition behavior, retained operator journeys and independent aarch64 proof remain. Each native action still needs its applicable authorization.

This guide records operational checks, not another implementation roadmap. The M15–M18 labels below identify the historical proof stages and existing commands; current ownership follows U/P above. Record reused evidence by exact revision/bytes/target, never as a second independent PASS. No new execution is authorized by these plans.

## M15: native source/build evidence

Follow [installation](installation.md) on the matching native x86_64 builder. Resolve/review the real Go dependency metadata, build with `scripts/build-native.sh x86_64`, then explicitly run `scripts/check-native.sh x86_64`. The latter runs the authored Go, TypeScript/UI and native staging checks; it does not enroll, install or restart services. Dependency/compiler/test failures belong in their source, not suppressed flags.

Record actual source revision, native OS/architecture/tool versions, commands, output and defects in an ordinary operator log or issue. Do not manufacture PASS lines or an acceptance schema. No CI workflow runs automatically.

## Read-only installed observations

After separately authorized installation, inspect:

- `rpm-ostree status`, the extensions journal and the active native packages;
- `systemctl status soda-host.socket forgejo.service soda-dashboard.service soda-proxy.service cockpit.socket tailscaled.service`;
- native service journals, listener addresses, subordinate UID/GID mappings, helper socket owner/group/mode, service UID and persistent directory ownership;
- TLS trust and both configured browser origins, native Forgejo clone URLs, Cockpit's root-only PAM policy;
- actual project inspection/IPs, client routes and existing firewall policy.

Do not paste credentials, full container environment dumps, provisioning password hashes or private keys into evidence. Service/listener state alone does not prove a login or development journey.

## M16–M17: explicitly permitted Alice/Bob journey

These actions **change real state**. Use operator-approved users, projects/repositories and credentials, with real client reachability. Extend/invoke the core-owned installed tests and [U08 proof detail](dashboard-implementation-plan.md#core-owned-native-proof-detail), including shared installation identity, precise denials and failed-snapshot handling; do not implement this sequence again in the P harness.

1. Complete the core-owned native Forgejo operator setup and OAuth bootstrap. Sign in through the browser. For the current HTMX baseline verify its configured-operator People restriction; after U05 verify actual Forgejo administrator authority instead. Ordinary non-admin developers cannot use administrator APIs; Forgejo administrator status alone never grants native Cockpit/root or extra Soda operator authority.
2. Create `alice` and `bob` through People; each changes their initial password in Forgejo and signs into Soda separately. Register each person's public development-access key. Private keys stay on their clients.
3. Alice creates an ordinary Forgejo repository with native Git credentials and then creates its Soda environment. Only its human owner may create that environment. Both explicitly select **Add me to this project**. No creator auto-enrollment is assumed.
4. From the real developer client, verify the SSH host key through native operator access and connect to the displayed project IP as Alice and Bob. Exercise interactive SSH, a noninteractive command, SCP and SFTP. Do not disable host-key checking to manufacture a pass.
5. Run `tests/installed/project-os.sh` inside the project with `SODA_NATIVE_VALIDATE` set only for that named target. Inspect `sudo -l`: Alice is project-local administrator, Bob is not. Neither acquires a host account or the host engine socket. A second project must have separate writable state and native identities.
6. Both use personal home checkouts, ordinary Git commits/pushes/merges and `~/shared`. Verify shared writes/readback by both. No Soda-managed branches/selectors or cleanup are expected.
7. Alice installs a shared tool with native mise using the documented root/global path (see [development environment](development-environment.md)). Execute `tests/installed/shared-tools.sh` from the real client with explicit Alice/Bob/project inputs. Verify both resolve and execute the same `/opt/mise/installs` installation without independent downloads.
8. In Alice's ordinary checkout of `tests/fixtures/workload`, provide a disposable database password and explicitly run `tests/installed/workloads.sh`. This builds images and starts services. Confirm the web/database from Bob and the actual client, not only localhost. Inspect a real bind-mounted file and persistent database write/read. Do not delete volumes automatically.
9. With separate permission, stop/start the existing `soda-project@ID.service`; then authorize a host reboot independently. Verify the same project container, accounts/homes/SSH host keys, shared installs/files, service configuration and database data survive. Do not remove/replace the project container to make it start.

The highest-risk profile is nested Podman with private cgroups, user-namespace allocation, fuse and the selected capabilities/seccomp/SELinux arrangement. If a real blocker appears, correct the concrete mechanism. Only then investigate the project-scoped host fallback; do not expose an unrestricted host socket, enable a privileged parent or introduce a VM substitute without revisiting the design.

## Operator Tailnet journey

Read-only: inspect the actual native state/peers and UI consistency. With separate permissions, exercise browser sign-in, exit-node selection/advertisement, LAN preference and provider approval. Preserve the native daemon's state rather than replacing it with a Soda inventory.

Forgejo Git advertisement is refreshed only when its actual private listener accepts the Tailnet IP. The supplied first-install/activation may bind a selected LAN IP instead: choose the intended Tailnet private address for activation or explicitly change/restart the native Forgejo port mapping before requesting refresh. The helper must not advertise an unreachable endpoint. Browser/OAuth origins remain fixed operator configuration, not an invented `host:30000` URL. Tailnet enrollment does not itself establish project subnet routes; inspect and approve those separately.

## Operator Runners journey

Provider registration and jobs are **not read-only checks**. Supply explicitly approved Forgejo/GitHub resources and tokens. Verify create/list/start/stop/restart, actual native runner account/capacity and a genuine provider-scheduled job on trusted code. Verify configured Forgejo administration links and one local slot per runner. Provider workflows/results stay provider-owned.

Removing a runner destroys its local state. Only exercise removal on an explicitly disposable runner with permission, and inspect provider-side cleanup separately. An unavailable provider/account is unverified, not a local-fake success.

## Compatible follow-up checks

- On the authorized native builder, the source entrypoint also exercises console/fetch process doubles and project-tool build-boundary tests. They do not query a real Tailnet or authenticate provider CLIs.
- On the actual host, inspect the [operator welcome](console-welcome.md) in an interactive root login and confirm noninteractive SSH/transfer output stays quiet. Printed configured origins are not proof of a listening service.
- Follow [branding review](branding-review.md) separately for the optional `branding` renderer tests and actual browser component check. It needs explicit native browser/renderer prerequisites, a disposable target and a fresh evidence directory; it is not an automatic source-gate side effect.
- In Alice/Bob's project, inspect `tea --version`/`gh --version`. With separate provider/account permission, follow [CLI authentication](project-clis.md) and confirm each uses only their own credential state. Mere CLI availability is not API compatibility evidence.
- Capture actual UI only under the [screenshot brief](screenshot-capture.md); no generated or component-sheet images stand in for installed product behavior.

## M18: independent AArch64 evidence

Repeat on actual matching-native aarch64 access when authorized. No sibling barrier, cross-build substitute or emulator result is native installed proof. Record that architecture's own results and unresolved differences.

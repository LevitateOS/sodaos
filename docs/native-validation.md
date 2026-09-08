# Native validation

**Historical bounded U08 native x86_64 proof is accepted; new UI and final product
acceptance are pending.** The [handoff](implementation-status.md) records exact
`8b823db` build/rollout/regressions and scoped earlier lifecycle/fresh-project proof.
The four retained environments and infra route are not a fresh appliance install
or independent aarch64 result. No old browser harness validates current API-only source.

This is the product validation guide, not another roadmap. Features own their
focused tests; extend existing product entrypoints rather than copying scenarios
into the [outside support tools](native-support.md). Those tools have separate
[remaining checks](native-support.md#remaining-validation), not a second readiness
gate. Optional media/helper completion is not required to use authorized tools.

Name the exact revision/artifacts, builder, target, architecture, developer client,
actions and cleanup limits before execution. Rehearse preserved-state changes before
separately approved cutover. A build permit is not install/restart, disk erasure,
provider registration, routing or fixture permission. Record reused evidence by
exact bytes/target, not a second independent PASS. Missing access is **unverified**.

## Native source and build evidence

Follow [installation](installation.md) on the matching native x86_64 builder. Resolve/review the real Go dependency metadata, build with `scripts/build-native.sh x86_64`, then explicitly run `scripts/check-native.sh x86_64`. The latter runs the authored Go, TypeScript/UI and native staging checks; it does not enroll, install or restart services. Dependency/compiler/test failures belong in their source, not suppressed flags.

Record actual source revision, native OS/architecture/tool versions, commands, output and defects in an ordinary operator log or issue. Do not manufacture PASS lines or an acceptance schema. No CI workflow runs automatically.

## Read-only Sodaspaces browser probe

`tests/installed/sodaspaces.mjs` is authored and opt-in, **not executed native proof**.
It uses stock 15.0.7, the candidate's served CSS/JS, real native forms and OAuth,
then read-only drawer states. It never seeds cookies/sessions, substitutes responses,
creates repositories/environments, joins or installs keys. Protective request
interception on the two exercised pages aborts unapproved origins/writes and makes
the run fail, not pass. Chromium CDP Fetch pauses each redirect hop before sending
it; Playwright route callbacks do not include those hops. No request/response is
fulfilled or replaced by the probe.

Execution needs explicit target and authentication-transition permission. Supply
an existing approved **public repository with Issues enabled and a plain new-issue
form**, two existing password-login fixture users who can view it, automatic native
theme selection, and the existing Soda OAuth client ID. MFA/captcha/insufficient
consent are not bypassed. The probe changes native browser sessions and Soda login/
logout/grants, which can affect upstream refresh counters; it does not revoke grants
or edit callbacks/secrets. It types, preserves, then discards only its own synthetic
unsaved issue title without submitting it. An existing-environment view requires
approved helper read scope; an absent reservation suffices for initial OAuth proof.

Use a fresh restricted browser home with the selected CA already trusted by
Chromium. All input/password/CA files are absolute regular mode-0600 files; the
home is mode 0700. No TLS bypass or sandbox disabling is selected. Reuse prepared
pinned Playwright/Chromium; the probe does not download browsers. Private request:

```json
{
  "origin": "https://approved-fixture.example",
  "target": "approved-fixture",
  "revision": "FULL_40_CHARACTER_CANDIDATE_REVISION",
  "repository_path": "/alice/approved-repository",
  "repository_id": "42",
  "oauth_client_id": "EXISTING_PUBLIC_CLIENT_ID",
  "ca_file": "/private/fixture-ca.pem",
  "users": [
    {"id": "1", "login": "alice", "password_file": "/private/alice-password"},
    {"id": "2", "login": "bob", "password_file": "/private/bob-password"}
  ]
}
```

Later authorized invocation (not permission):

```sh
SODA_NATIVE_VALIDATE=approved-fixture node tests/installed/sodaspaces.mjs \
  /private/request.json /private/browser-home --allow-auth-transitions
```

The source checkout must be clean and match the declared revision. A fresh
`sodaspaces-run/` below the home retains the private profile and exclusive sanitized
`result.json`, including failures. Existing run directories are refused untouched.
No screenshots, traces, raw callback URLs, provider bodies or credential dumps are
recorded. The caller must separately bind the installed backend/config/artifacts;
served asset hashes alone do not prove its binary revision or installed migration.

Coverage includes raw proxy alias/encoding denials, native version/asset routes,
conditional asset revalidation, actual Soda cookie attributes, anonymous/native-only
cookies, two OAuth returns, actor/CSRF logout denials, native-only account switching,
Soda-only logout, stale tabs, native unsaved form coexistence, keyboard/focus/backdrop,
narrow/wide automatic themes and actual back-forward restoration. Playwright's
inspected default `--disable-back-forward-cache` is omitted for this check. If a real
BFCache restoration does not occur, the probe records incomplete scope and exits 2,
not a synthetic pass; failure exits 1. Exit 0 is only this scoped journey. Current
asset revalidation is not an update rehearsal; native provisioning/SSH, final product,
backend artifact binding, cutover and independent aarch64 acceptance remain separate.

## Read-only installed observations

After separately authorized installation, inspect:

- `rpm-ostree status`, the extensions journal and the active native packages;
- `systemctl status soda-host.socket forgejo.service soda-dashboard.service soda-proxy.service cockpit.socket tailscaled.service`;
- native service journals, listener addresses, subordinate UID/GID mappings, helper socket owner/group/mode, service UID and persistent directory ownership;
- TLS trust and the configured Forgejo/Sodaspaces browser origin, native Forgejo clone URLs, Cockpit's root-only PAM policy;
- actual project inspection/IPs, client routes and existing firewall policy.

Do not paste credentials, full container environment dumps, provisioning password hashes or private keys into evidence. Service/listener state alone does not prove a login or development journey.

## Explicitly permitted product journey

These actions **change real state**. Use explicitly approved users, repositories,
projects and credentials with real client reachability; existing fixture grants
are not reusable permission. Extend/invoke product-owned installed tests. Keep shared
installation identity, precise denial results and failure-safe bounded snapshots;
a failed inspection is not evidence of absent state or forbidden access.

1. Complete the core-owned native Forgejo operator setup and OAuth bootstrap. Sign in through the browser. Both old Soda frontends are removed from current source; native Sodaspaces browser coverage remains pending. Use native Forgejo operator setup, not the retired Soda People form. Test Soda's authority boundary: Forgejo administrator status alone never grants native Cockpit/root or extra Soda operator authority. Do not duplicate upstream administrator-API permission tests.
2. Provision the approved Alice/Bob identities through native Forgejo; each completes native password/security requirements and authenticates independently. Exercise native-page/Sodaspaces identity matching and development-key controls once implemented. Private keys stay on their clients; no database-seeded browser success.
3. Alice creates an ordinary Forgejo repository with native Git credentials and then creates its Soda environment. Only its human owner may create that environment. Both explicitly select **Add me to this project**. No creator auto-enrollment is assumed.
4. From the real developer client, verify the SSH host key through native operator access and connect to the displayed project IP as Alice and Bob. Exercise interactive SSH, a noninteractive command, SCP and SFTP. Do not disable host-key checking to manufacture a pass.
5. Run `tests/installed/project-os.sh` inside the project with `SODA_NATIVE_VALIDATE` set only for that named target. Inspect `sudo -l`: Alice is project-local administrator, Bob is not. Neither acquires a host account or the host engine socket. A second project must have separate writable state and native identities.
6. Both use personal home checkouts, ordinary Git commits/pushes/merges and `~/shared`. Verify shared writes/readback by both. No Soda-managed branches/selectors or cleanup are expected.
7. Alice installs a shared tool with native mise using the documented root/global path (see [development environment](development-environment.md)). Execute `tests/installed/shared-tools.sh` from the real client with explicit Alice/Bob/project inputs. Verify both resolve and execute the same `/opt/mise/installs` installation without independent downloads.
8. In Alice's ordinary checkout of `tests/fixtures/workload`, provide a disposable database password and explicitly run `tests/installed/workloads.sh`. This builds images and starts services. Confirm the web/database from Bob and the actual client, not only localhost. Inspect a real bind-mounted file and persistent database write/read. Do not delete volumes automatically.
9. With separate permission, stop/start the existing `soda-project@ID.service`; then authorize a host reboot independently. Verify the same project container, accounts/homes/SSH host keys, shared installs/files, service configuration and database data survive. Do not remove/replace the project container to make it start.
10. Once implemented, verify the browser terminal is the user's existing project-local account/home, with explicit session lifetime, origin/CSRF, bounded transport and cross-project denials. Opening it must not create/join/start anything or expose host root.

Test Soda's customization and integration, not upstream Forgejo business logic:
OAuth/session/CSRF handling, actor matching, environment authorization after native
rename/transfer, template/asset delivery and Caddy route boundaries. Cover the
Sodaspaces drawer's stale/denied/error states, keyboard/focus, narrow/wide display
and exact-version hook compatibility, without automatic Linux remapping. Native
account/repository operations above supply fixtures and exercise Soda's project
integration; they are not independent tests of Forgejo's implementation. Keep only
focused native smoke checks where Soda changes could cause a regression. Do not
add general upstream login/MFA, administration, collaboration or Git/LFS/package
conformance suites, or rebuild the retired 179-group JSON register.

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

## Independent aarch64 evidence

Repeat on actual matching-native aarch64 access when authorized. No sibling barrier, cross-build substitute or emulator result is native installed proof. Record that architecture's own results and unresolved differences.

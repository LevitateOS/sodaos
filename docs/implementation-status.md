# Implementation handoff

## Current execution — 2026-09-06

**Recorded M15 x86_64 build/source checks passed; the local dashboard is activated and its operator browser journey is verified.** The isolated `soda-test` CoreOS KVM host runs Forgejo, the Soda dashboard, Caddy and native operator services. See [local testing](local-testing.md) for URLs, private credential locations, exact scope and logs. Initial native startup fixes and resolved Go metadata are in `88be176`; navigation changes are in `95a194d`, and dashboard-access/repository-picker changes are in `c96530c`. The combined tree with branding, console and project-CLI follow-ups has not been rebuilt or retested. No artifact publication was performed.

The full M16–M17 developer/project/workload journeys remain pending. AArch64 M18 remains unverified. A first install recovered from the discovered copy/label defects is not a fresh-disk proof of the final installer. Nested Podman and direct client routing remain the highest native risks. Additional installations, network changes and provider/lifecycle operations still require named targets and explicit permission.

### Dashboard activation and browser access

Completed Forgejo's native installer on the isolated VM with a new local `operator` administrator, created a scoped native token, and ran the real `soda-setup` OAuth bootstrap and `soda-activate`. Soda is at `https://localhost:24443`, Forgejo at `https://localhost:24444`, through loopback-only SSH tunnels; Cockpit remains at port 29090. The existing localhost test certificate/CA is used, not public production TLS. Added `scripts/test-vm.sh web-tunnel` and documented the separate laptop forwards and private credentials.

Added and executed `tests/installed/dashboard.mjs`: real browser TLS verification, Forgejo password authentication, consent/S256 OAuth callback into Soda, secure HTTP-only session, authenticated Projects/Profile/operator People navigation, and CSRF-protected sign-out. Chromium trusts the CA in an isolated NSS home; no certificate bypass or normal browser/system trust changes were used. The native dashboard process is UID/GID 2000 with zero effective capabilities, services are active, and no failed systemd units were observed. Evidence is in `dashboard-browser-check.log` and `dashboard-services.log`.

No developer users, repositories, project containers or provider runners were created. This is operator authentication/navigation evidence, not the full product journey. Corrected operator command examples to use `/usr/local/sbin` explicitly because the CoreOS root SSH PATH omits it. Dashboard bootstrap intentionally restarted only the VM's Forgejo service, not the host or VM itself.

### Forgejo repository picker

Replaced manual `owner/repository` entry on Projects with an accessible native select populated from Forgejo's `GET /users/{username}/repos`, verified against the installed 15.0.7 API schema. Requests target the signed-in user, not `/user/repos` for the privileged operator token. Pagination continues until an empty page; the dashboard filters by stable owner ID and excludes every existing project reservation, including incomplete provisioning, then sorts names. Creation still re-fetches the selected repository and enforces ownership server-side.

Added native Forgejo creation/refresh links and distinct empty/provider-error states that retain the existing Projects table. Go tests cover pagination (including server page-size caps), malformed/failed responses, cancellation, ownership/privacy filtering, existing reservations, empty states and forged selection rejection. All source/staging checks pass. Rebuilt and loaded only the dashboard image, updated its staged/native binary, and restarted only `soda-dashboard.service`; application data and the other services were preserved. The old image archive was retained after Podman's refusal to overwrite it directly.

The real browser check now verifies the picker or its empty state, native create/refresh links, and the existing OAuth/navigation/logout journey. The test operator currently owns zero repositories, so the live run verified the empty state; populated/filtering cases are covered by Go tests, not claimed as an installed populated-repository journey. No repository or project fixture was created. Evidence: `repository-picker-tests.log`, `repository-picker-build.log`, `repository-picker-deploy.log`, and `dashboard-browser-check.log`. Changes are committed in `c96530c`.

### Cockpit/Tailnet native correction

The first interactive Tailnet read exposed a missing SELinux PAM session transition: root authenticated successfully but its bridge remained in `cockpit_session_t`, where Tailscale socket access and stock systemd operations were denied. Restored the native Fedora Cockpit PAM stack while retaining the required UID-0 account gate. A new authenticated Cockpit WebSocket session now runs in the native operator context and successfully reads `tailscale status --json` and LocalAPI preferences. SELinux remains enforcing; no socket permission changes, daemon restart or Tailnet enrollment were performed. Native PAM account checks allow root and deny the existing non-operator `core` account.

Added a staging regression for the root-only gate and ordered SELinux session rules. The old staged config fails it; the corrected stage passes all five packaging checks, along with Go, TypeScript, 60 Cockpit tests and installed host checks. Existing Cockpit users must log out and back in to receive the correction. This correction is included in `88be176`.

### Accounts navigation

At the operator's request, hide only Cockpit's stock Accounts menu entry through the native `/etc/cockpit/users.override.json` merge patch. The source config is staged for future installations and applied to `soda-test`; no packages, host accounts or native account tools were removed, and no services restarted. Native `cockpit-bridge --packages` before/after output confirms that `users` loses only its Accounts label and every other menu entry is unchanged (`cockpit-packages-before.log` / `cockpit-packages-after.log`). Added a packaging regression; all six packaging checks, Go tests, TypeScript checks and 60 Cockpit tests pass. Browser sessions may need logout/login to discard cached manifests. This navigation change is committed in `95a194d`.

## Core/native plan coordination merge

Pulled `origin/main` (`9c8d672`) into local `55ce5cb` with `git pull --no-rebase --no-commit origin main`, preserving both documentation histories. The only textual conflict was the introduction to `docs/implementation-plan.md`; resolved it by retaining the historical M01–M18 context, the leading U plan and subordinate native-support reference. No application-source conflict or application change was involved.

At the user's direction, [dashboard-implementation-plan.md](dashboard-implementation-plan.md) leads the core—including Go/API/auth/data, production `internal/host`/`project-os`, shared build/config contracts and U08/U20 product acceptance. Reworked [native-porting-plan.md](native-porting-plan.md) around outside VM/QMP/SSH/evidence/artifact tools, provisioning transport and retained host-operator integrations. Former P07/P08 redirect to U08/U20; their useful direct-IP, shared-installation, workload and bounded-persistence test details are retained in the core plan. P06 owns host/service observations, P11 outside integrations, and P12/P13 scoped support evidence—not parallel browser/product suites or readiness verdicts. P09/P10 media remains conditional and is not a core gate.

Added explicit shared-file/input/output ownership, evidence reuse and non-circular ordering to both plans; aligned AGENTS, README, architecture, inventory, deferred scope, historical plan, installation, native validation and reuse guidance. Existing authorized tools may support U08 without waiting for the P port. No U/E/P implementation milestone is completed by this coordination, and existing native limitations/evidence are unchanged.

This merge/coordination performs documentation/diff, conflict-marker, local link/anchor, milestone-reference and whitespace review only. No builds, product tests, dependency installation/resolution, artifact generation, VM/service/network/provider operations, push or publication ran. The merged application remains unrebuilt/unretested.

## Unified React dashboard planning

Recorded the user's next frontend direction and a proposed full page/direct-dependency inventory in [dashboard planning](dashboard-plan.md): client-rendered TypeScript/React, PatternFly, Vite+ and Zustand, without SSR, Tailwind or TanStack. Go remains the application API and Forgejo the identity/Git/collaboration authority. The plan separates the first working repository-to-environment flow, later functional coverage and native screens/API gaps; it does not add a host-administration frontend or a second password authority.

Inspected current source/manifests and the previously saved installed Forgejo 15.0.7 API schema. The current OAuth flow requests only `read:user` and does not retain access/refresh tokens for subsequent user-scoped operations; the plan explicitly requires extending that lifecycle rather than using the operator credential as a universal proxy. Updated guidance to distinguish the selected next direction from the still-existing HTMX implementation. Only documentation was changed and whitespace/diff review performed. No dependencies were installed or changed; no builds, product tests, VM operations or live provider requests ran for this planning work. Page scope and new dependency pins remain proposals, not implemented capabilities.

Clarified the upstream boundary at the user's request: Soda extends Forgejo with development environments, project-local access and public-key provisioning; frontend ownership does not transfer Forgejo's backend/data/permission responsibilities. Expanded the plan to explicitly include upstream-authorized Forgejo site-administrator views instead of permanently relegating them to the native UI. Native screens remain fallbacks for unverified interfaces; Cockpit remains host administration. Removed the illustrative backend ratio from the plan. Reviewed the saved `/admin` API schema and updated guidance only; no application, dependency or runtime changes were made.

### Multi-milestone frontend/backend plan

Added [the U01–U20 implementation plan](dashboard-implementation-plan.md) at the user's request. It sequences source-backed capability research, React/static packaging, JSON/schema migration, per-session Forgejo credentials, accounts/admin onboarding, repositories, native environments and early installed proof before broader collaboration/admin coverage, cutover, polish and final verification. It includes source ownership, API/state/security contracts, data/config migration and rollback constraints, page-to-milestone coverage, dependencies, acceptance cases and evidence levels. Forgejo remains upstream; the Go layer does not acquire its business or permission ownership.

Rocky/Fedora profiles, existing-environment lifecycle controls and basic resource caps are conditional E01–E03 tracks, not silently approved scope. Other previously deferred lifecycle/ownership/recovery features remain explicit decisions. Linked the current plan from guidance, architecture, inventory and README and marked the original M01–M18 plan historical. This is documentation only; no implementation milestone, dependency resolution, compilation, product test, provider call, VM action or deployment ran. Documentation paths/anchors, milestone coverage references and whitespace were reviewed; the planning work was committed as `55ce5cb`.

## Source merge — 2026-09-06

Merged the local predecessor follow-ups (`0f25570`, `4e751ac`, `f2523ac`) with remote history through `95a194d`. Resolved conflicts in agent guidance, README navigation/status, operator setup and the project image. The image retains native `curl-minimal` and mise checksum fixes alongside Tea/GitHub CLI inputs; combined staging tests retain both the PAM regression and the branding/console checks. The real Go metadata and native installation fixes are preserved.

The earlier native evidence does **not** validate this merged tree or its additional checks. This merge performs source/diff, conflict-marker, whitespace and local documentation-link review only. No dependency resolution, builds, product tests, VM/service operations or provider actions were run. The current request authorizes Git merge/commit/push, not additional native execution.

### Dashboard follow-up merge

Merged `origin/main` through `f4fe066` into `c96530c`, preserving both histories. Resolved five documentation conflicts by retaining the newer dashboard activation/browser and repository-picker evidence alongside the predecessor follow-ups and their unvalidated status. Reviewed the automatic staging/configuration merge; Accounts navigation, PAM checks and branding/console checks are retained. Updated stale uncommitted-change references.

Only source/diff, conflict-marker and whitespace checks were performed for this merge. No builds, tests, dependency resolution, VM/service operations or provider actions were run; the combined tree remains unvalidated.

## Original native artifact and acceptance proposal (scope superseded above)

Upstream commit `9c8d672` added the original [porting proposal](native-porting-plan.md), based on current source `6f7b51e` and predecessor `bc1d3e0`. It maps selected VM/QMP, process/cleanup, SSH/evidence, artifact-inspection and scenario source/tests to proposed destinations, with P01–P13 dependencies, source/native exits and exact-target execution gates. CoreOS remains the proposed host path; bootc/Anaconda, the old account model, Updates and release publication/qualification machinery remain excluded.

The plan explicitly distinguishes application OCI from a bootable host image, public media from private provisioning, a QCOW2 deployment kit from a preinstalled image, and forwarded access from a real client route. It prioritizes fresh x86_64 installation/developer/persistence evidence and preserves independent aarch64 follow-up. Existing `soda-test` state and its backing disk must remain untouched.

Planning/documentation only. Reviewed source/documentation, diffs, local documentation link targets, plan anchors and whitespace. No port implementation, generated artifacts, builds, product tests, dependency resolution, VM/service/network/provider operations or publication ran. At that proposal's creation P01–P13 were not started. The coordination merge above transfers P07/P08 into core U08/U20 and narrows the remaining P scope; active P milestones are still unimplemented. Prior native results and the merged tree's unvalidated status are unchanged.

## Original source handoff (historical)

The M01–M14 entries below describe the original source-only handoff, when builds, tests, type checks, dependency resolution, installation and publication had not run. Their original “not run” statements are historical; current evidence is recorded above and in [local testing](local-testing.md).

## Baseline

- New backend and privileged integration: Go; no Rust subsystem without a concrete need. Retained Cockpit frontend: TypeScript/React.
- Go 1.26.7 (predecessor source baseline); HTMX 2.0.10 vendored from its npm distribution with license.
- Host candidate: Fedora CoreOS stable 44.20260817.3.2, reported for x86_64/aarch64 by the upstream stable stream metadata. Native compatibility remains unverified.
- Predecessor source: local `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. No changes are made in that repository; Updates remain excluded.

## M01 — source implemented

Go server/configuration, embedded canonical branding, HTMX source, initial templates, narrow process runner reused from predecessor, and authored configuration/server/process tests. Application features follow in subsequent milestones; this foundation is not a completed product.

Build: not run. Validation: not run.

## M02 — source implemented

SQLite via modernc.org/sqlite v1.58.0 (upstream module metadata), schema and concrete store, hashed session tokens and single-use OAuth state storage. Startup opens the persistent database. Tests authored, not run; go.sum/dependency resolution remains a later build input step.

## M03 — source implemented

Forgejo 15.0.7 API client, confidential OAuth setup CLI, persistent Forgejo container definition and operator-native installation instructions. Native API/schema source inspected; no live accounts or OAuth applications created.

## M04 — source implemented

Forgejo OAuth authorization code + S256, session rotation/logout, CSRF/origin checks, profile/public-key forms, operator user creation, and real provider/database wiring. Tests authored; not run. Browser and native OAuth behavior remain unverified.

## M05 — source implemented

Rocky 9.6 + mise project image recipe, systemd/OpenSSH initialization, native project-account setup, root-owned SSH keys and retained writable-rootfs strategy. Native image/account/SSH behavior remains unbuilt and unvalidated.

## M06 — source implemented

Fixed-operation root helper over a root:soda systemd Unix socket; dashboard client wired. Native Podman create-once/start/inspect/account commands, user-namespace mapping, retained project units and routed bridge IP readback. Configure a real route to the chosen project subnet before client SSH; bridge allocation alone is not remote reachability proof. Native helper/network/persistence tests authored, not executed.

## M07 — source implemented

Project discovery/create/detail and explicit join routes now reach the provider, native helper and database. Repository ownership is checked server-side; membership records follow native account success. Real IP inspection and non-ready errors are displayed. Authored a provider/native-double create-flow test, not run.

## M08 — source implemented

Shared mise install/shim path and native global config are wired into login profiles and SSH non-interactive environments. Project-owner installation instructions and authored two-user checks accompany ordinary Git/shared-file guidance. No private resource selection machinery. Build/validation not run.

## M09 — source implemented, high native risk

Nested project-local Podman service, owner-only shared-engine socket, Compose 1.6.0 provider and build/mount/database example. Parent user namespace, capabilities, fuse, SELinux and cgroup combination requires native proof; no privileged host shortcut or dormant fallback added. Build/validation not run.

## M10 — source implemented

Selected native CoreOS package layering for Cockpit, root-only PAM, loopback-first operator access and native /etc/cockpit/branding delivery. Shared frontend dependency closure/build source and licenses ported without old Projects/Updates entrypoints. Feature entrypoints follow in M11/M12. No builds or tests run.

## M11 — source implemented

Tailnet page/native state/store/components and focused tests ported. Native host refresh now updates Forgejo Git SSH advertisement while preserving configured browser/OAuth origins, inspects live environment and restarts the actual container unit when needed. No Tailnet enrollment or native execution performed. M14 additionally verifies the actual Git SSH listener before advertising a Tailnet address, avoiding unreachable clone guidance.

## M12 — source implemented

Runners page, native provider/lifecycle/helper/launcher code, service wiring and focused tests ported. Root-only authorization replaces the incompatible predecessor human-host-admin rule. Configured Forgejo origins now feed native registration and browser links. Provider-client source locks retained; no binary download, registration, build or validation performed.

## M13 — source implemented

Native build/staging, first-install and private Butane provisioning recipes authored; dashboard/proxy Quadlets, TLS activation, service identity/ownership, native extension package requests, provider-client checksum fetch and branding staging included. Routes require explicit deployment configuration; no project DNS/gateway added. Native staging tests authored. No build, package install, provisioning render, provider download, activation or validation executed.

## M14 — source implemented

Integrated authenticated navigation, visible HTMX error responses, pending native-operation states, basic accessibility and direct SSH/SCP/SFTP guidance that is withheld for stopped/unavailable endpoints. Added server-side compatible username validation, HTTPS/loopback configuration guards, provider/native journey and privilege test source, and the missing retained Cockpit process-test support. Removed predecessor-guessed Tailnet service URLs; actual listener inspection now guards Forgejo advertisement. Corrected native Forgejo SVG/favicon destinations and relocated theme imports from inspected upstream template paths without modifying canonical branding assets.

Authored explicit matching-native source/staging check entrypoint and the full later Alice/Bob, persistence, Tailnet and provider-runner journey. Architecture documentation now reflects implemented choices rather than describing them as unresolved. x/sys v0.47.0 source metadata was inspected; dependency resolution was not run.

**Source-complete, unbuilt, unvalidated.** No builds, compilation/type checks, tests, dependency installations, provisioning renders, native activations, enrollments, registrations or CI jobs were executed. M15–M18 remain held.

## Repository guidance update

Customized `AGENTS.md` from the predecessor's engineering guidance: requirement-versus-choice classification, human-maintainable design, coherent refactoring/reuse, source ownership, actual script side effects, scoped commit authorization and separate evidence reporting. Retained SodaOS's execution hold, project-local authority, private networking and persistent-container boundaries; excluded obsolete predecessor test/release commands and UI assumptions.

Documentation-only change. Reviewed the predecessor/current guidance, owning documentation, script source and full diff; local documentation links and whitespace checked. No product behavior changed and no builds, tests, dependency resolution or deployment operations ran.

## Compatible predecessor follow-up ports

Source reference remains `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`; that repository is unchanged.

- **Branding/configuration:** ported Forgejo browser checks, PNG renderer/pixel comparator and focused test source. Added fresh-evidence and explicit native-review boundaries; actual renderer verification is opt-in with the `branding` test tag. Adapted app metadata, native/accessibility theme choices and cache revalidation into a staged Forgejo environment source file, using the selected upstream environment-to-INI encoding. The disposable component sheet is not an appliance payload. No artwork regeneration, browser checks, tests, builds or native configuration changes ran.
- **Console/handbook:** adapted the native operator welcome, main-table and connected-uplink discovery, interactive-only hook and CLI delivery. It reads configured public origins without printing secrets or inventing reachable ports. Added command-double/staging test source. Reworked developer instructions for project-IP SSH, editors, separate Git credentials, real shared mise installations and project-local service ports; adapted the screenshot brief without adding fabricated captures. No native console, tests, account operations or screenshots were executed.
- **Project CLIs:** retained Tea 0.15.1 source lock/license/fetch behavior and adapted its native Makefile build into project-image inputs, with fetch/build-boundary test source. Added the predecessor's GitHub CLI 2.97.0 baseline through GitHub's signed RPM repository inside Rocky rather than copying a Fedora host package. Developer auth remains native and personal. Source/release metadata was inspected; no archives, binaries, RPMs, dependencies, builds or logins were fetched/executed. The reuse inventory and later validation guide separate these ports from the excluded workspace/release/Updates systems. Source review also tightened console origin parsing and removed stale predecessor RPM/page/renderer instructions and a broken link from the Cockpit asset guide. Formatting, diff/whitespace inspection and local documentation path checks are the only checks executed.

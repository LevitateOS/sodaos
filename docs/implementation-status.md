# Implementation handoff

## Current execution — 2026-09-06

**M15 x86_64 build/source checks executed successfully.** An isolated `soda-test` CoreOS KVM host has booted, received native extensions and Soda's first-install components, and serves the Forgejo installer and Cockpit. See [local testing](local-testing.md) for access, exact scope, discovered/fixed defects and logs. The working tree contains the real resolved Go metadata; no artifact publication or commit was made.

Dashboard OAuth/TLS activation and the M16–M17 product journeys remain pending. AArch64 M18 remains unverified. A first install recovered from the discovered copy/label defects is not a fresh-disk proof of the final installer. Nested Podman and direct client routing remain the highest native risks. Additional installations, network changes and provider/lifecycle operations still require named targets and explicit permission.

### Cockpit/Tailnet native correction

The first interactive Tailnet read exposed a missing SELinux PAM session transition: root authenticated successfully but its bridge remained in `cockpit_session_t`, where Tailscale socket access and stock systemd operations were denied. Restored the native Fedora Cockpit PAM stack while retaining the required UID-0 account gate. A new authenticated Cockpit WebSocket session now runs in the native operator context and successfully reads `tailscale status --json` and LocalAPI preferences. SELinux remains enforcing; no socket permission changes, daemon restart or Tailnet enrollment were performed. Native PAM account checks allow root and deny the existing non-operator `core` account.

Added a staging regression for the root-only gate and ordered SELinux session rules. The old staged config fails it; the corrected stage passes all five packaging checks, along with Go, TypeScript, 60 Cockpit tests and installed host checks. Existing Cockpit users must log out and back in to receive the correction. Changes remain uncommitted.

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

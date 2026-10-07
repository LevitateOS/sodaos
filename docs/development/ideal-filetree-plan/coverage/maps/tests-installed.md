# Tests installed

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-ad5b5504b727"></a>

## [tests/installed/host.sh](../../../../../tests/installed/host.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require explicitly selected installed native root before read-only host observations; declarations/fields: `SODA_NATIVE_VALIDATE` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/host.sh; manifest identical_prior confirms byte identity |
| 9–19, 34–42; prior detailed responsibility interval; lines 39–42: installed Tailscale Cockpit page file presence | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Observe enforcing CoreOS substrate, installed packages and native deployment state; declarations/fields: `soda-host-probes`, `host-content`, `host-deployments`; Observe native Cockpit configuration and retired custom runner page absence; declarations/fields: `cockpit.conf`; Installed Cockpit integration presence is checked for the Tailscale page assets. — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/host.sh; manifest identical_prior confirms byte identity; tests/installed/host.sh:39-42; current for-loop checks index.html and manifest.json |
| 20–25, 31–33, 43–49; prior detailed responsibility interval | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Observe native active services and canonical host filesystem ownership; declarations/fields: `systemctl`; Observe protected native host socket and dashboard state ownership; declarations/fields: `host.sock`; Observe activated service lifetime, restricted dashboard container identity and private config; declarations/fields: `phase`, `soda-dashboard` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/host.sh; manifest identical_prior confirms byte identity |
| 26–30; prior detailed responsibility interval | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Observe installed compiled command ownership/modes/SELinux labels; declarations/fields: `command` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/host.sh; manifest identical_prior confirms byte identity |
| 50–58; prior detailed responsibility interval | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Observe pre-activation Forgejo ports confined to native loopback; declarations/fields: `podman`; Observe requested activation phase and native listeners; explicitly no routed-client proof; declarations/fields: `host-listeners` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/host.sh; manifest identical_prior confirms byte identity |

<a id="coverage-d1389a68d05d"></a>

## [tests/installed/operator.sh](../../../../../tests/installed/operator.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require explicitly selected enforcing native root before retained observations; declarations/fields: `SODA_NATIVE_VALIDATE` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.sh; manifest identical_prior confirms byte identity |
| 6–7; prior detailed responsibility interval | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Observe current host Tailnet status and binary version; declarations/fields: `operator-tailscale` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.sh; manifest identical_prior confirms byte identity |
| 8–9, 13–16; prior detailed responsibility interval | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Observe delivered stock Cockpit/Forgejo branding files; Observe delivered noninteractive welcome-hook silence without login/TTY; declarations/fields: `quiet` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.sh; manifest identical_prior confirms byte identity |
| 10–12; prior detailed responsibility interval | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Assert obsolete custom Tailnet/runner/update Cockpit pages are absent; declarations/fields: `soda-tailscale`, `soda-runners`, `soda-updates` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.sh; manifest identical_prior confirms byte identity |

<a id="coverage-b2b8dc1c50a4"></a>

## [tests/installed/operator.ts](../../../../../tests/installed/operator.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35, 109–210; prior detailed responsibility interval | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Native Cockpit root administration driver declarations; 4 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.ts; manifest identical_prior confirms byte identity |
| 36–86; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require explicit native target/private input and test-owned browser lifetime — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.ts; manifest identical_prior confirms byte identity |
| 87–108; prior detailed responsibility interval | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Snapshot configured native origins before stock administration journey — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.ts; manifest identical_prior confirms byte identity |
| 211–231; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Exercise native logout and retire test browser/evidence without retaining credentials — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/operator.ts; manifest identical_prior confirms byte identity |

<a id="coverage-aa3d5b80afa0"></a>

## [tests/installed/project-foundation.sh](../../../../../tests/installed/project-foundation.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16, 62; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require scoped explicitly selected ordinary Project account and private disposable evidence before native fixture compilation; declarations/fields: `SODA_NATIVE_VALIDATE`, `SODA_PROJECT_FOUNDATION_APPROVED`; Report selected native account/fixture evidence scope only — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/project-foundation.sh; manifest identical_prior confirms byte identity |
| 17–61; prior detailed responsibility interval | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Declare minimal C/C++/CMake tool workloads for actual installed compilers; declarations/fields: `CMakeLists.txt`; Observe isolated-HOME native compilation/testing/debugger/trace/object tools; declarations/fields: `cmake`, `ctest`, `gdb`, `strace`, `readelf` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/project-foundation.sh; manifest identical_prior confirms byte identity |

<a id="coverage-1625ce3ec7a6"></a>

## [tests/installed/project-os.sh](../../../../../tests/installed/project-os.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require scoped ordinary selected Project before native observations; declarations/fields: `SODA_NATIVE_VALIDATE` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/project-os.sh; manifest identical_prior confirms byte identity |
| 6–20; prior detailed responsibility interval | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Observe installed headless userspace/tooling/terminfo substrate; declarations/fields: `rpm`, `infocmp` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/project-os.sh; manifest identical_prior confirms byte identity |
| 21–23; prior detailed responsibility interval | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Observe selected account home and protected authorized key file ownership; declarations/fields: `id`, `stat` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/project-os.sh; manifest identical_prior confirms byte identity |
| 24; prior detailed responsibility interval | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | State direct SSH client authentication/routing is separate proof — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/project-os.sh; manifest identical_prior confirms byte identity |

<a id="coverage-3355ddec1b89"></a>

## [tests/installed/service-ordering.sh](../../../../../tests/installed/service-ordering.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require explicitly selected installed native root before service-ordering observations; declarations/fields: `SODA_NATIVE_VALIDATE` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/service-ordering.sh; manifest identical_prior confirms byte identity |
| 6–31; prior detailed responsibility interval | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Observe socket/first-boot/image-import/service dependency order and native unit state; declarations/fields: `systemctl`, `soda-image-import.service` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/service-ordering.sh; manifest identical_prior confirms byte identity |

<a id="coverage-134ff9cdad90"></a>

## [tests/installed/shared-tools.sh](../../../../../tests/installed/shared-tools.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require explicit ordinary selected Projects and scoped routed native SSH/client fixture inputs; declarations/fields: `SODA_NATIVE_VALIDATE` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/shared-tools.sh; manifest identical_prior confirms byte identity |
| 19–45; prior detailed responsibility interval | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Observe shared executable root ownership, shared inode and refusal of ordinary writes; declarations/fields: `mise`, `stat`; Exercise selected shared file/tool/per-user alias and cross-member inode behavior; declarations/fields: `alias`, `stat` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/shared-tools.sh; manifest identical_prior confirms byte identity |
| 46–48; prior detailed responsibility interval | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Observe selected cross-Project state isolation and report scope — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/shared-tools.sh; manifest identical_prior confirms byte identity |

<a id="coverage-1f992a9a12b8"></a>

## [tests/installed/sodaspaces-cli.ts](../../../../../tests/installed/sodaspaces-cli.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–199; installed soda-spaces CLI qualification helper | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Drives the installed CLI in qualification and records outcomes for the outside acceptance harness; it consumes installed-build evidence rather than defining CLI behavior. — Current source inspected at tests/installed/sodaspaces-cli.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-2d5e3af8d830"></a>

## [tests/installed/sodaspaces-controls.ts](../../../../../tests/installed/sodaspaces-controls.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6, 24, 36–83; prior detailed responsibility interval | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Declarations/fixtures and integration for Installed project/terminal UI control helpers; prepareManagedTerminal — Installed project/terminal UI control helpers; declarations/fields: `prepareManagedTerminal`; newManagedTerminal — Installed project/terminal UI control helpers; declarations/fields: `newManagedTerminal` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-controls.ts; manifest identical_prior confirms byte identity |
| 7–23; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | loginNativeForgejo — Installed project/terminal UI control helpers; declarations/fields: `loginNativeForgejo` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-controls.ts; manifest identical_prior confirms byte identity |
| 25–35, 84–89; prior detailed responsibility interval | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | projectView — Installed project/terminal UI control helpers; declarations/fields: `projectView`; terminalMenu — Installed project/terminal UI control helpers; declarations/fields: `terminalMenu` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-controls.ts; manifest identical_prior confirms byte identity |

<a id="coverage-704d1bfb0ec6"></a>

## [tests/installed/sodaspaces-input.ts](../../../../../tests/installed/sodaspaces-input.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 24, 49–145; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declarations/fixtures and integration for Private journey/mutation scope parsing and exact terminal reservation matcher; 6 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-input.ts; manifest identical_prior confirms byte identity |
| 20–23; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | object — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `object` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-input.ts; manifest identical_prior confirms byte identity |
| 25–44; prior detailed responsibility interval | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | matchesTerminalReservation — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `matchesTerminalReservation` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-input.ts; manifest identical_prior confirms byte identity |
| 45–48; prior detailed responsibility interval | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | validID — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `validID` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/sodaspaces-input.ts; manifest identical_prior confirms byte identity |

<a id="coverage-5112ed1b4d4d"></a>

## [tests/installed/sodaspaces-matrix-native.ts](../../../../../tests/installed/sodaspaces-matrix-native.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–249; installed native matrix qualification helper | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Runs the native installed-build qualification matrix and emits evidence consumed by the release acceptance workflow. — Current source inspected at tests/installed/sodaspaces-matrix-native.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-09b51898e255"></a>
<a id="testsinstalledsodaspaces-workspace-journeyts-1"></a>

## [tests/installed/sodaspaces-workspace-journey.ts](../../../../../tests/installed/sodaspaces-workspace-journey.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–242; installed workspace journey qualification helper | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Drives installed workspace journeys and captures qualification evidence for the outside acceptance workflow. — Current source inspected at tests/installed/sodaspaces-workspace-journey.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-c845d92d9275"></a>

## [tests/installed/workloads.sh](../../../../../tests/installed/workloads.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Require explicitly approved selected Project/native mode before workload operation; declarations/fields: `SODA_NATIVE_VALIDATE` — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/workloads.sh; manifest identical_prior confirms byte identity |
| 10–34, 36; prior detailed responsibility interval | [P11](../../slices/projects.md#p11-nested-services-and-volumes) | retained | Start selected disposable workload once or enter separate read-only Check path; declarations/fields: `start`, `check`; Observe bounded database/HTTP readiness and native workload secret/volume data; declarations/fields: `podman`, `curl`; Preserve running workload/volume state; no implicit cleanup — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/workloads.sh; manifest identical_prior confirms byte identity |
| 35; prior detailed responsibility interval | [N02](../../slices/networking.md#n02-project-lan-access) | retained | State external routed native workload service ports require separate client proof — docs/development/ideal-filetree-plan/coverage/maps/tests-installed.md#tests/installed/workloads.sh; manifest identical_prior confirms byte identity |

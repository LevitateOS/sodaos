# Tests installed

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-ad5b5504b727"></a>

## [tests/installed/host.sh](../../../../../tests/installed/host.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–8 | Require explicitly selected installed native root before read-only host observations; declarations/fields: `SODA_NATIVE_VALIDATE` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 9–19 | Observe enforcing CoreOS substrate, installed packages and native deployment state; declarations/fields: `soda-host-probes`, `host-content`, `host-deployments` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 20–25 | Observe native active services and canonical host filesystem ownership; declarations/fields: `systemctl` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 26–30 | Observe installed compiled command ownership/modes/SELinux labels; declarations/fields: `command` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 31–33 | Observe protected native host socket and dashboard state ownership; declarations/fields: `host.sock` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 34–38 | Observe native Cockpit configuration and retired custom runner page absence; declarations/fields: `cockpit.conf` |
| Historical; no active owner / obsolete | 39–42 | Obsolete custom soda-tailscale Cockpit page presence assertions; declarations/fields: `soda-tailscale` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 43–49 | Observe activated service lifetime, restricted dashboard container identity and private config; declarations/fields: `phase`, `soda-dashboard` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 50–53 | Observe pre-activation Forgejo ports confined to native loopback; declarations/fields: `podman` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 54–58 | Observe requested activation phase and native listeners; explicitly no routed-client proof; declarations/fields: `host-listeners` |

<a id="coverage-d1389a68d05d"></a>

## [tests/installed/operator.sh](../../../../../tests/installed/operator.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–5 | Require explicitly selected enforcing native root before retained observations; declarations/fields: `SODA_NATIVE_VALIDATE` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 6–7 | Observe current host Tailnet status and binary version; declarations/fields: `operator-tailscale` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 8–9 | Observe delivered stock Cockpit/Forgejo branding files |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 10–12 | Assert obsolete custom Tailnet/runner/update Cockpit pages are absent; declarations/fields: `soda-tailscale`, `soda-runners`, `soda-updates` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 13–16 | Observe delivered noninteractive welcome-hook silence without login/TTY; declarations/fields: `quiet` |

<a id="coverage-b2b8dc1c50a4"></a>

## [tests/installed/operator.ts](../../../../../tests/installed/operator.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 1–9 | Native Cockpit root administration driver declarations |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 10–35 | Use stock Cockpit login and native package/frame navigation; declarations/fields: `loginOperator`, `openOperatorPackage` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 36–86 | Require explicit native target/private input and test-owned browser lifetime |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 87–108 | Snapshot configured native origins before stock administration journey |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 109–180 | Observe selected native root host identity/SELinux/service diagnostics and forbidden custom pages |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 181–210 | Visit stock native host administration pages read-only, preserving existing accounts/containers/files; declarations/fields: `openOperatorPackage` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 211–231 | Exercise native logout and retire test browser/evidence without retaining credentials |

<a id="coverage-aa3d5b80afa0"></a>

## [tests/installed/project-foundation.sh](../../../../../tests/installed/project-foundation.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–16 | Require scoped explicitly selected ordinary Project account and private disposable evidence before native fixture compilation; declarations/fields: `SODA_NATIVE_VALIDATE`, `SODA_PROJECT_FOUNDATION_APPROVED` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 17–50 | Declare minimal C/C++/CMake tool workloads for actual installed compilers; declarations/fields: `CMakeLists.txt` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 51–61 | Observe isolated-HOME native compilation/testing/debugger/trace/object tools; declarations/fields: `cmake`, `ctest`, `gdb`, `strace`, `readelf` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 62 | Report selected native account/fixture evidence scope only |

<a id="coverage-1625ce3ec7a6"></a>

## [tests/installed/project-os.sh](../../../../../tests/installed/project-os.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–5 | Require scoped ordinary selected Project before native observations; declarations/fields: `SODA_NATIVE_VALIDATE` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 6–20 | Observe installed headless userspace/tooling/terminfo substrate; declarations/fields: `rpm`, `infocmp` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 21–23 | Observe selected account home and protected authorized key file ownership; declarations/fields: `id`, `stat` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 24 | State direct SSH client authentication/routing is separate proof |

<a id="coverage-3355ddec1b89"></a>

## [tests/installed/service-ordering.sh](../../../../../tests/installed/service-ordering.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–5 | Require explicitly selected installed native root before service-ordering observations; declarations/fields: `SODA_NATIVE_VALIDATE` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 6–31 | Observe socket/first-boot/image-import/service dependency order and native unit state; declarations/fields: `systemctl`, `soda-image-import.service` |

<a id="coverage-134ff9cdad90"></a>

## [tests/installed/shared-tools.sh](../../../../../tests/installed/shared-tools.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–18 | Require explicit ordinary selected Projects and scoped routed native SSH/client fixture inputs; declarations/fields: `SODA_NATIVE_VALIDATE` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 19–39 | Observe shared executable root ownership, shared inode and refusal of ordinary writes; declarations/fields: `mise`, `stat` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 40–45 | Exercise selected shared file/tool/per-user alias and cross-member inode behavior; declarations/fields: `alias`, `stat` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 46–48 | Observe selected cross-Project state isolation and report scope |

<a id="coverage-1f992a9a12b8"></a>

## [tests/installed/sodaspaces-cli.ts](../../../../../tests/installed/sodaspaces-cli.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–24 | Parse observed native terminal context and declared CLI wire markers without generating expected semantics; declarations/fields: `objectContext`, `cliProtocolObservation` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 25–55 | Require exactly selected CLI/provider use, restricted prompts and observed native installed versions/account/terminfo; declarations/fields: `exerciseSelectedCLIs` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 56–120 | Drive declared CLI in existing browser terminal, observe streamed native bytes and preserve original attachment through resize/re-entry; declarations/fields: `exerciseSelectedCLIs` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 121–182 | Drive same declared CLI through exact pinned native SSH identity and observe interactive streamed protocol; declarations/fields: `exerciseSelectedCLIs` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 183–199 | Return bounded CLI protocol comparison evidence and selected scope only; declarations/fields: `exerciseSelectedCLIs` |

<a id="coverage-2d5e3af8d830"></a>

## [tests/installed/sodaspaces-controls.ts](../../../../../tests/installed/sodaspaces-controls.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1–6, 24, 36–37 | Declarations/fixtures and integration for Installed project/terminal UI control helpers |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 7–23 | loginNativeForgejo — Installed project/terminal UI control helpers; declarations/fields: `loginNativeForgejo` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 25–35 | projectView — Installed project/terminal UI control helpers; declarations/fields: `projectView` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 38–58 | prepareManagedTerminal — Installed project/terminal UI control helpers; declarations/fields: `prepareManagedTerminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 59–83 | newManagedTerminal — Installed project/terminal UI control helpers; declarations/fields: `newManagedTerminal` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 84–89 | terminalMenu — Installed project/terminal UI control helpers; declarations/fields: `terminalMenu` |

<a id="coverage-704d1bfb0ec6"></a>

## [tests/installed/sodaspaces-input.ts](../../../../../tests/installed/sodaspaces-input.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–3, 24 | Declarations/fixtures and integration for Private journey/mutation scope parsing and exact terminal reservation matcher |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 4–9 | JourneyUser — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `JourneyUser` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 10–19 | JourneyInput — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `JourneyInput` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 20–23 | object — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `object` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 25–44 | matchesTerminalReservation — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `matchesTerminalReservation` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 45–48 | validID — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `validID` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 49–101 | journeyInput — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `journeyInput` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 102–113 | ManagementRequest — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `ManagementRequest` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 114–145 | managementInput — Private journey/mutation scope parsing and exact terminal reservation matcher; declarations/fields: `managementInput` |

<a id="coverage-5112ed1b4d4d"></a>

## [tests/installed/sodaspaces-matrix-native.ts](../../../../../tests/installed/sodaspaces-matrix-native.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–9, 73 | Declarations/fixtures and integration for Scoped SSH/process/native terminal observation and browser-shell probe |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 10–15 | restrictedText — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `restrictedText` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 16 | quote — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `quote` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 17–49 | sshArgs — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `sshArgs` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 50–60 | matrixSSH — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `matrixSSH` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 61–72 | inspectMatrixProcess — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `inspectMatrixProcess` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 74–146 | inspectNativeTerminal — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `inspectNativeTerminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 147–249 | observeMatrixShell — Scoped SSH/process/native terminal observation and browser-shell probe; declarations/fields: `observeMatrixShell` |

<a id="coverage-09b51898e255"></a>

<a id="testsinstalledsodaspaces-workspace-journeyts-1"></a>

## [tests/installed/sodaspaces-workspace-journey.ts](../../../../../tests/installed/sodaspaces-workspace-journey.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–11, 36, 171 | Declarations/fixtures and integration for Selected installed first-use and matrix product journey driver |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 12–25 | nativeSpacesMount — Selected installed first-use and matrix product journey driver; declarations/fields: `nativeSpacesMount` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 26–35 | FirstUseEvidence — Selected installed first-use and matrix product journey driver; declarations/fields: `FirstUseEvidence` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 37–53 | Declare selected first-use target, permitted writes and evidence state; declarations/fields: `exerciseFirstUse` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 54–89 | Observe exactly one admitted native Create ID and exact later attachment identity; declarations/fields: `exerciseFirstUse` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 90–127 | Observe actual native repository/profile selection and one explicit Project Create; declarations/fields: `exerciseFirstUse` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 128–144 | Observe explicitly permitted browser-only Join and native response; declarations/fields: `exerciseFirstUse` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 145–151 | Create one explicitly named/measured native terminal; declarations/fields: `exerciseFirstUse` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 152–157 | Submit native shell probe and observe ordinary account/process/TTY facts; declarations/fields: `exerciseFirstUse` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 158–170 | Reload/re-enter exact existing native terminal without replacement; declarations/fields: `exerciseFirstUse` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 172–177 | MatrixSession — Selected installed first-use and matrix product journey driver; declarations/fields: `MatrixSession` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 178–185 | MatrixFacts — Selected installed first-use and matrix product journey driver; declarations/fields: `MatrixFacts` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 186–199 | MatrixEvidence — Selected installed first-use and matrix product journey driver; declarations/fields: `MatrixEvidence` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 200–210 | MatrixNative — Selected installed first-use and matrix product journey driver; declarations/fields: `MatrixNative` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 211–239 | Bind selected multi-actor/Project matrix and evidence observers; declarations/fields: `exerciseWorkspaceMatrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 240–292 | Observe bounded generation/Project admission, one reservation and original IDs for native Create/Attach; declarations/fields: `exerciseWorkspaceMatrix` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 293–307 | Select exact authorized existing session without replacement; declarations/fields: `exerciseWorkspaceMatrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 308–329 | Create six explicitly permitted native sessions across two selected Projects and retain identities; declarations/fields: `exerciseWorkspaceMatrix` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 330–350 | Observe competing native writer refusal without takeover or new session; declarations/fields: `exerciseWorkspaceMatrix` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 351–369 | Move/split native panes and retain all live renderer owners; declarations/fields: `exerciseWorkspaceMatrix` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 370–389 | Hide/select/reload exact existing sessions and preserve original native process facts; declarations/fields: `exerciseWorkspaceMatrix` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 390–393 | Invoke separately declared installed CLI protocol exercise only when explicitly selected; declarations/fields: `exerciseWorkspaceMatrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 394–441 | Explicitly confirm End on each original native ID and verify exact cleanup without affecting siblings; declarations/fields: `exerciseWorkspaceMatrix` |

<a id="coverage-c845d92d9275"></a>

## [tests/installed/workloads.sh](../../../../../tests/installed/workloads.sh)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–9 | Require explicitly approved selected Project/native mode before workload operation; declarations/fields: `SODA_NATIVE_VALIDATE` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 10–18 | Start selected disposable workload once or enter separate read-only Check path; declarations/fields: `start`, `check` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 19–34 | Observe bounded database/HTTP readiness and native workload secret/volume data; declarations/fields: `podman`, `curl` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 35 | State external routed native workload service ports require separate client proof |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 36 | Preserve running workload/volume state; no implicit cleanup |


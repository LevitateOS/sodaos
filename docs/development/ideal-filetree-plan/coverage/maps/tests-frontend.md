# Tests frontend

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-9d1e53fd14b7"></a>

<a id="testsfrontenddrawer-controlstestts-1"></a>

## [tests/frontend/drawer-controls.test.ts](../../../../../tests/frontend/drawer-controls.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–39, 66, 178, 265, 306, 485, 609, 624 | Declarations/fixtures and integration for Explicit native project controls and retained workspace/terminal ownership |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 40–55 | Test harness fixture — Explicit native project controls and retained workspace/terminal ownership; declarations/fields: `fixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 56–58 | Test harness refresh — Explicit native project controls and retained workspace/terminal ownership; declarations/fields: `refresh` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 59–64 | Test harness click — Explicit native project controls and retained workspace/terminal ownership; declarations/fields: `click` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 65 | Test harness writes — Explicit native project controls and retained workspace/terminal ownership; declarations/fields: `writes` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 67–86 | Assert project control mount is inert; refresh only reads and never owns a terminal; declarations/fields: `test: project control mount is inert; refresh only reads and never owns a terminal` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 87–104 | Assert project view tabs and app switches dispatch no reads or writes; declarations/fields: `test: project view tabs and app switches dispatch no reads or writes` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 105–110 | Assert hidden access view cannot remove a key through a synthetic click; declarations/fields: `test: hidden access view cannot remove a key through a synthetic click` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 111–126 | Assert compact-surface inert project controls cannot dispatch through a synthetic click; declarations/fields: `test: compact-surface inert project controls cannot dispatch through a synthetic click` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 127–145 | Assert create never implicitly joins, saves keys or starts; rapid clicks dispatch once; declarations/fields: `test: create never implicitly joins, saves keys or starts; rapid clicks dispatch once` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 146–162 | Assert managed Create submits the reviewed binding, while an explicit Off ignores the managed default; declarations/fields: `test: managed Create submits the reviewed binding, while an explicit Off ignores the managed default` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 163–177 | Assert network failure after Create keeps the project and offers network recovery, not recreation; declarations/fields: `test: network failure after Create keeps the project and offers network recovery, not recreation` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 179–203 | Assert shared Network panel requires current administration and explicit target confirmation; double activation sends once; declarations/fields: `test: shared Network panel requires current administration and explicit target confirmation; double activation sends once` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 204–223 | Assert members get only observed own-account network access, not mutation controls or automatic authentication; declarations/fields: `test: members get only observed own-account network access, not mutation controls or automatic authentication` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 224–247 | Assert network startup uncertainty is not shown as connected and never replays on refresh; declarations/fields: `test: network startup uncertainty is not shown as connected and never replays on refresh` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 248–264 | Assert legacy OS observation is explicit, read-only and never becomes a creation profile; declarations/fields: `test: legacy OS observation is explicit, read-only and never becomes a creation profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 266–305 | Assert OS text stays text, malformed receipts clear observations, and denial invalidates context; declarations/fields: `test: OS text stays text, malformed receipts clear observations, and denial invalidates context` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 307–318 | Assert Stop requires explicit shared-impact confirmation and Start is separate; declarations/fields: `test: Stop requires explicit shared-impact confirmation and Start is separate` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 319–325 | Assert stopped environment has explicit Start only, and no terminal; declarations/fields: `test: stopped environment has explicit Start only, and no terminal` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 326–335 | Assert saved-key removal truthfully does not dispatch a native apply; declarations/fields: `test: saved-key removal truthfully does not dispatch a native apply` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 336–349 | Assert review then explicit Apply confirms last-key removal; declarations/fields: `test: review then explicit Apply confirms last-key removal` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 350–378 | Assert an unconfirmed key apply requires another target preview, not permanent page lockout; declarations/fields: `test: an unconfirmed key apply requires another target preview, not permanent page lockout` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 379–415 | Assert own Forgejo key selection is explicit and does not install or silently save a profile key; declarations/fields: `test: own Forgejo key selection is explicit and does not install or silently save a profile key` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 416–424 | Assert private-key paste is refused before request dispatch; declarations/fields: `test: private-key paste is refused before request dispatch` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 425–433 | Assert nonadministrator has no lifecycle controls; nonmember joins separately; declarations/fields: `test: nonadministrator has no lifecycle controls; nonmember joins separately` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 434–444 | Assert browser-only Join is available without a public key and never imports saved keys implicitly; declarations/fields: `test: browser-only Join is available without a public key and never imports saved keys implicitly` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 445–451 | Assert external SSH at Join requires explicit saved-key selection; declarations/fields: `test: external SSH at Join requires explicit saved-key selection` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 452–470 | Assert unconfirmed Create keeps its notice without a permanent lock or another reservation; declarations/fields: `test: unconfirmed Create keeps its notice without a permanent lock or another reservation` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 471–484 | Assert stale project controls cannot refresh/replay on focus; declarations/fields: `test: stale project controls cannot refresh/replay on focus` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 486–502 | Assert ${kind} response cannot expose actions; declarations/fields: `test: ${kind} response cannot expose actions` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 503–509 | Assert repeated project refresh never acquires terminal ownership; declarations/fields: `test: repeated project refresh never acquires terminal ownership` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 510–529 | Assert closing during the initial native read cannot publish controls or dispatch a mutation; declarations/fields: `test: closing during the initial native read cannot publish controls or dispatch a mutation` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 530–547 | Assert unknown key-save response cannot claim a confirmed key; declarations/fields: `test: unknown key-save response cannot claim a confirmed key` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 548–554 | Assert hidden create and lifecycle actions cannot dispatch through their handlers; declarations/fields: `test: hidden create and lifecycle actions cannot dispatch through their handlers` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 555–562 | Assert copy uses own displayed login/IP without changing native access; declarations/fields: `test: copy uses own displayed login/IP without changing native access` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 563–592 | Assert reactive project view/Hide updates preserve draft identity and selection; declarations/fields: `test: reactive project view/Hide updates preserve draft identity and selection` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 593–608 | Assert hidden completed project reads never attach a terminal on showing details; declarations/fields: `test: hidden completed project reads never attach a terminal on showing details` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 610–623 | Assert render readiness after disposal cannot call terminal factory; reconnect remains retired; declarations/fields: `test: render readiness after disposal cannot call terminal factory; reconnect remains retired` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 625–630 | Assert unavailable/incomplete state refuses terminal: ${JSON.stringify(state)}; declarations/fields: `test: unavailable/incomplete state refuses terminal: ${JSON.stringify(state)}` |

<a id="coverage-ab64cfc46f94"></a>

## [tests/frontend/fixtures/drawer-fixture.ts](../../../../../tests/frontend/fixtures/drawer-fixture.ts)

Source/assertion inspection only; no suite or native scenario executed. Synthetic test/preview behavior only. Domain mapping identifies represented responsibilities, without claiming an active product service.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–56 | Browser fixture declarations, project state knobs and inert mount setup; declarations/fields: `createFixture`, `Call`, `State` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 57–68 | Synthetic reviewed network target/policy selection snapshot; declarations/fields: `network` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 69–85 | Capture fixture HTTP dispatch, native generation and override/refusal mechanics; declarations/fields: `fetchFixture` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 86–101 | Synthetic Create receipt preserving project outcome independently of optional network setup; declarations/fields: `fetchFixture` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 102 | Synthetic explicit human Join receipt; declarations/fields: `fetchFixture` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 103–107 | Synthetic explicit native Start/Stop receipt; declarations/fields: `fetchFixture` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 108–116 | Synthetic development key application/removal response distinction; declarations/fields: `fetchFixture` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 117–125 | Synthetic reviewed network mutation/readback receipts; declarations/fields: `fetchFixture` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 126–132 | Synthetic supported profile and observed Project readiness; declarations/fields: `fetchFixture` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 133–134 | Synthetic saved human development key metadata; declarations/fields: `fetchFixture` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 135–140 | Synthetic observed Project userspace version receipt; declarations/fields: `fetchFixture` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 141–142 | Synthetic observed native Project lifecycle receipt; declarations/fields: `fetchFixture` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 143–154 | Synthetic target key fingerprints and ordinary project SSH connection; declarations/fields: `fetchFixture` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 155–166 | Synthetic Project association/member/admin/execution observation projection; declarations/fields: `fetchFixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 167–219 | Install production project controls with test-owned transport, selectors, callbacks and global facade; declarations/fields: `createFixture` |

<a id="coverage-0d9022ac1da1"></a>

## [tests/frontend/fixtures/terminal-fixture.ts](../../../../../tests/frontend/fixtures/terminal-fixture.ts)

Source/assertion inspection only; no suite or native scenario executed. Synthetic test/preview behavior only. Domain mapping identifies represented responsibilities, without claiming an active product service.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–41 | Fixture state declarations, tracked test sockets/renderers and mount setup; declarations/fields: `createFixture`, `TerminalFixtureOptions` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 42–57 | Synthetic exact issued/native terminal metadata; declarations/fields: `metadata` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 58–78 | Track real observer retirement, deferred replies and locator events; declarations/fields: `createFixture` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 79–112 | Synthetic terminal reservation/lookup/explicit End HTTP state; declarations/fields: `fetchFixture` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 113–155 | Test-owned socket frame capture/open/message/close for interactive attachment; declarations/fields: `Socket` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 156–212 | Instrument terminal emulator parser/render/focus/write/disposal for browser assertions; declarations/fields: `Terminal` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 213–254 | Bind production mountTerminal to instrumented renderer and native-generation test transport; declarations/fields: `mountTerminal`, `transport`, `binding` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 255–287 | Expose exact test selectors, reply hooks and renderer scheduling; declarations/fields: `button`, `socket`, `term` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 288–300 | Emit test-owned ready frame and explicit invalid-locator fixture call; declarations/fields: `ready`, `mountInvalidLocator` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 301–309 | Publish typed browser fixture globals; declarations/fields: `createTerminalFixture`, `terminalFixture` |

<a id="coverage-197a19f57480"></a>

## [tests/frontend/fixtures/workspace-model.ts](../../../../../tests/frontend/fixtures/workspace-model.ts)

Source/assertion inspection only; no suite or native scenario executed. Synthetic test/preview behavior only. Domain mapping identifies represented responsibilities, without claiming an active product service.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–13 | Synthetic workspace model options, storage and fixed actor declarations; declarations/fields: `createWorkspaceModel` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 14–25 | Synthetic exact terminal metadata factory; declarations/fields: `metadata` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 26–68 | Synthetic two-Project authorized inventory and ended-session persistence; declarations/fields: `spaces` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 69–80 | Synthetic supported userspace profile and creation availability knobs; declarations/fields: `profile` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 81–90 | Record test requests and generation/actor/failure controls; declarations/fields: `calls`, `generation` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 91–100 | Synthetic collection admission/completeness/actor refusal; declarations/fields: `request` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 101–119 | Synthetic native repository discovery and existing reservation choices; declarations/fields: `request` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 120–124 | Synthetic current profile readiness and absent optional Tailnet choice; declarations/fields: `request` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 125–150 | Synthetic exact Project observation/Create, uncertain/incomplete/rejected outcomes; declarations/fields: `request` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 151–157 | Synthetic empty/unavailable development-key collection; declarations/fields: `request` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 158–163 | Synthetic explicit browser-only Join and failure outcomes; declarations/fields: `request` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 164–183 | Synthetic native terminal reserve/rename/explicit End state and socket retirement; declarations/fields: `request` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 184–193 | Synthetic observed Project status/admission; declarations/fields: `request` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 194–197 | Synthetic separately explicit native Start/Stop; declarations/fields: `request` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 198–204 | Synthetic ordinary Project login/IP/fingerprint connection projection; declarations/fields: `request` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 205–223 | Fixture WebSocket registration and bounded open callback instrumentation; declarations/fields: `Socket` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 224–238 | Synthetic generation-bound terminal input/attach admission; declarations/fields: `Socket.send` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 239–250 | Consume exactly issued reservation and create one exact native terminal; declarations/fields: `Socket.send` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 251–266 | Record attachment, native-ready delivery and detach without ending terminal; declarations/fields: `Socket.send`, `Socket.end`, `Socket.close` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 267–300 | Expose selected failure/pause/actor/model controls; declarations/fields: `createWorkspaceModel` |

<a id="coverage-09a2663e5b5a"></a>

## [tests/frontend/identity.test.ts](../../../../../tests/frontend/identity.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 1–5, 25, 53, 79 | Declarations/fixtures and integration for Identity connection/enrollment/grant/lease browser metadata boundaries |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 6–24 | Assert subscription metadata remains bound to owner and excludes credential fields; declarations/fields: `test: subscription metadata remains bound to owner and excludes credential fields` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 26–52 | Assert device enrollment only links to the selected provider authentication origin; declarations/fields: `test: device enrollment only links to the selected provider authentication origin` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 54–65 | Assert named delegation/grant stable identities; declarations/fields: `grantView` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 66–78 | Assert execution lease metadata hides credential/binding delivery; declarations/fields: `leaseView` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 80–177 | Assert identity widget keeps stable context and starts Codex with protected metadata-only request; declarations/fields: `test: identity widget keeps stable context and starts Codex with protected metadata-only request` |

<a id="coverage-5c4dbe25534a"></a>

## [tests/frontend/journey-input.test.ts](../../../../../tests/frontend/journey-input.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–3, 14, 28, 41, 51 | Declarations/fixtures and integration for Private installed-journey actor/target/action permission parsing |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 4 | Test harness user — Private installed-journey actor/target/action permission parsing; declarations/fields: `user` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 5–13 | Test harness input — Private installed-journey actor/target/action permission parsing; declarations/fields: `input` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 15–27 | Assert private journey input preserves two distinct actors and rejects ambiguous identities; declarations/fields: `test: private journey input preserves two distinct actors and rejects ambiguous identities` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 29–40 | Assert public-key inputs require explicit access mode and valid project logins; declarations/fields: `test: public-key inputs require explicit access mode and valid project logins` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 42–50 | Assert managed terminal input requires explicit create/End scope and cannot reuse old read-only inputs; declarations/fields: `test: managed terminal input requires explicit create/End scope and cannot reuse old read-only inputs` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 52–69 | Assert management input rejects extra authority fields and malformed container targets; declarations/fields: `test: management input rejects extra authority fields and malformed container targets` |

<a id="coverage-a85f845b1ff8"></a>

## [tests/frontend/native-project-controls.test.ts](../../../../../tests/frontend/native-project-controls.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–42, 69, 84, 104, 124, 148, 159, 169, 179, 190, 205, 241 | Declarations/fixtures and integration for Native project contribution explicit Create/Join/lifecycle/key/network actions |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 43–58 | Test harness fixture — Native project contribution explicit Create/Join/lifecycle/key/network actions; declarations/fields: `fixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 59–61 | Test harness refresh — Native project contribution explicit Create/Join/lifecycle/key/network actions; declarations/fields: `refresh` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 62–67 | Test harness click — Native project contribution explicit Create/Join/lifecycle/key/network actions; declarations/fields: `click` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 68 | Test harness writes — Native project contribution explicit Create/Join/lifecycle/key/network actions; declarations/fields: `writes` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 70–83 | Assert native project controls mount inertly and preserve Forgejo content; declarations/fields: `test: native project controls mount inertly and preserve Forgejo content` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 85–103 | Assert Create dispatches once with native generation and does not join or start; declarations/fields: `test: Create dispatches once with native generation and does not join or start` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 105–123 | Assert Create uses the reviewed network binding and keeps network failure separate; declarations/fields: `test: Create uses the reviewed network binding and keeps network failure separate` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 125–147 | Assert network changes require target confirmation and dispatch once; declarations/fields: `test: network changes require target confirmation and dispatch once` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 149–158 | Assert Stop requires shared-impact confirmation and never opens a terminal; declarations/fields: `test: Stop requires shared-impact confirmation and never opens a terminal` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 160–168 | Assert Join is explicit and leaves saved SSH keys out by default; declarations/fields: `test: Join is explicit and leaves saved SSH keys out by default` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 170–178 | Assert private key input is rejected before a native request; declarations/fields: `test: private key input is rejected before a native request` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 180–189 | Assert saved-key removal does not apply keys to the environment; declarations/fields: `test: saved-key removal does not apply keys to the environment` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 191–204 | Assert removing the last installed key needs reviewed explicit Apply; declarations/fields: `test: removing the last installed key needs reviewed explicit Apply` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 206–240 | Assert Forgejo public-key selection stays a read until explicit profile save; declarations/fields: `test: Forgejo public-key selection stays a read until explicit profile save` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 242–257 | Assert an unconfirmed Create does not retry a reservation; declarations/fields: `test: an unconfirmed Create does not retry a reservation` |

<a id="coverage-cf5e8a201965"></a>

## [tests/frontend/sodaspaces-http.test.ts](../../../../../tests/frontend/sodaspaces-http.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–7, 23 | Declarations/fixtures and integration for Bounded raw probe HTTP/TLS transport contract |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 8–22 | Assert probe HTTP decoding bounds headers/body and refuses ambiguous validators; declarations/fields: `test: probe HTTP decoding bounds headers/body and refuses ambiguous validators` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 24–109 | Assert raw HTTPS keeps constrained CA verification, exact paths and bounded responses; declarations/fields: `test: raw HTTPS keeps constrained CA verification, exact paths and bounded responses` |

<a id="coverage-452b81773833"></a>

## [tests/frontend/spaces-api.test.ts](../../../../../tests/frontend/spaces-api.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 1–48, 92–105 | Declarations/fixtures and integration for Authorized Spaces inventory and bounded native/factory projections |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 49–76 | Assert repository choices bind stable identity and distinguish existing reservations from Create; declarations/fields: `test: repository choices bind stable identity and distinguish existing reservations from Create` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 77–91 | Assert terminal metadata preserves exact native binding without retired lifetime fields; declarations/fields: `test: terminal metadata preserves exact native binding without retired lifetime fields` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 106–108 | Assert invalid terminal metadata ${JSON.stringify(delta)}; declarations/fields: `test: invalid terminal metadata ${JSON.stringify(delta)}` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 109–118 | Assert Spaces validates visible rows and does not turn incomplete into complete; declarations/fields: `test: Spaces validates visible rows and does not turn incomplete into complete` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 119–134 | Assert degraded collection cannot carry session metadata or elevation; declarations/fields: `test: degraded collection cannot carry session metadata or elevation` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 135–170 | Assert Spaces admits the factory authority verdict and renders its status; declarations/fields: `test: Spaces admits the factory authority verdict and renders its status` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 171–203 | Assert Spaces admits the factory control state and renders its status; declarations/fields: `test: Spaces admits the factory control state and renders its status` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 204–219 | Assert Spaces admits the bounded next cursor and factory incompleteness; declarations/fields: `test: Spaces admits the bounded next cursor and factory incompleteness` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 220–224 | Assert Factory command identities are idempotent ledger keys; declarations/fields: `test: Factory command identities are idempotent ledger keys` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 225–250 | Assert Spaces admits factory run rows and renders their status; declarations/fields: `test: Spaces admits factory run rows and renders their status` |

<a id="coverage-1b00ae83c417"></a>

## [tests/frontend/spaces-attention.test.ts](../../../../../tests/frontend/spaces-attention.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1–17 | Declarations/fixtures and integration for Bounded authorized workspace attention/lifecycle observations |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 18–29 | Assert attention is bounded lifecycle observation, never output semantics or cleanup inference; declarations/fields: `test: attention is bounded lifecycle observation, never output semantics or cleanup inference` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 30–42 | Assert terminal observations reject bad identity, generations, unknown reasons and transcript fields; declarations/fields: `test: terminal observations reject bad identity, generations, unknown reasons and transcript fields` |

<a id="coverage-747c939fc0ac"></a>

## [tests/frontend/tailnet.test.ts](../../../../../tests/frontend/tailnet.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–5, 44, 82 | Declarations/fixtures and integration for Observed native host Tailnet response projections |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 6–43 | Test harness snapshot — Observed native host Tailnet response projections; declarations/fields: `snapshot` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 45–81, 83–119 | Assert Tailnet projections reject malformed, unsafe and mixed-version responses; declarations/fields: `test: Tailnet projections reject malformed, unsafe and mixed-version responses` |

<a id="coverage-99652b7b5667"></a>

<a id="testsfrontendterminaltestts-1"></a>

## [tests/frontend/terminal.test.ts](../../../../../tests/frontend/terminal.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–61, 95, 100–101, 182, 303, 379, 477, 530, 542–551 | Declarations/fixtures and integration for Interactive terminal attachment, renderer, IO, retirement and lifetime controls |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 62–77 | Test harness fixture — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `fixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 78–82 | Test harness action — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `action` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 83–86 | opening — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `opening` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 87–90 | ready — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `ready` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 91–94 | end — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `end` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 96–97 | actions — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `actions` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 98–99 | inputFrames — Interactive terminal attachment, renderer, IO, retirement and lifetime controls; declarations/fields: `inputFrames` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 102–119 | Assert mounting is inert and does not own Forgejo markup or beforeunload; declarations/fields: `test: mounting is inert and does not own Forgejo markup or beforeunload` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 120–143 | Assert pre-issued exact locator before one explicit Create and correct native authority; declarations/fields: `test: server locator is published before Create; native generation and bounded Unicode IO` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 144–168 | Assert bounded Unicode input/output on established interactive attachment; declarations/fields: `test: server locator is published before Create; native generation and bounded Unicode IO` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 169–181 | Assert sub-URL deployments prefix API and socket URLs; declarations/fields: `test: sub-URL deployments prefix API and socket URLs` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 183–205 | Assert ${event} retires access without End or replay; declarations/fields: `test: ${event} retires access without End or replay` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 206–234 | Assert visibility changes preserve renderer and perform no lifetime operation; declarations/fields: `test: visibility changes preserve renderer and perform no lifetime operation` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 235–241 | Assert late readiness does not steal native focus; declarations/fields: `test: late readiness does not steal native focus` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 242–253 | Assert focus-triggered retirement cannot leave a late observer; declarations/fields: `test: focus-triggered retirement cannot leave a late observer` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 254–281 | Assert hidden controls and late renderer import cannot dispatch; declarations/fields: `test: hidden controls and late renderer import cannot dispatch` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 282–302 | Assert late reservation result cannot create a socket after retirement; declarations/fields: `test: late reservation result cannot create a socket after retirement` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 304–309 | Assert actual reservation authorization rejects before native transport: ${JSON.stringify(options)}; declarations/fields: `test: actual reservation authorization rejects before native transport: ${JSON.stringify(options)}` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 310–343 | Assert End is separately confirmed with native generation and survives attachment closing first; declarations/fields: `test: End is separately confirmed with native generation and survives attachment closing first` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 344–357 | Assert unconfirmed End keeps the locator, but does not prevent a later explicit action; declarations/fields: `test: unconfirmed End keeps the locator, but does not prevent a later explicit action` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 358–378 | Assert native focus and hidden views cannot forward input; declarations/fields: `test: native focus and hidden views cannot forward input` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 380–396 | Assert confirmed native absence/exit cannot reopen the same ID (${state}); declarations/fields: `test: confirmed native absence/exit cannot reopen the same ID (${state})` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 397–421 | Assert unavailable lookup is not absence; an explicit refresh may recover; declarations/fields: `test: unavailable lookup is not absence; an explicit refresh may recover` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 422–439 | Assert lost creation reply uses the issued ID, not another Create or request namespace; declarations/fields: `test: lost creation reply uses the issued ID, not another Create or request namespace` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 440–450 | Assert late open after hiding sends no Create, but keeps the pre-issued locator; declarations/fields: `test: late open after hiding sends no Create, but keeps the pre-issued locator` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 451–476 | Assert missing, retired pending and malformed locators refuse before mounting; declarations/fields: `test: missing, retired pending and malformed locators refuse before mounting` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 478–484 | Assert ${state} is an observation, not a permanent admission flag; declarations/fields: `test: ${state} is an observation, not a permanent admission flag` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 485–490 | Assert observed writer refuses takeover without creating work; declarations/fields: `test: observed writer refuses takeover without creating work` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 491–503 | Assert retired renderer cannot send through a successor attachment; declarations/fields: `test: retired renderer cannot send through a successor attachment` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 504–520 | Assert disposal and late socket callbacks close exactly once; declarations/fields: `test: disposal and late socket callbacks close exactly once` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 521–529 | Assert input and output queues stay bounded; declarations/fields: `test: input and output queues stay bounded` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 531–541 | Assert ${overload} overload detaches without replay; declarations/fields: `test: ${overload} overload detaches without replay` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 552–559 | Assert invalid frame: ${JSON.stringify(frame).slice(0, 70)}; declarations/fields: `test: invalid frame: ${JSON.stringify(frame).slice(0, 70)}` |

<a id="coverage-2f3c2de49d3b"></a>

## [tests/frontend/workspace-journey.test.ts](../../../../../tests/frontend/workspace-journey.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 1–23 | Declarations/fixtures and integration for Exact private matrix/terminal approval and native observer contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 24–48 | Test harness scope — Exact private matrix/terminal approval and native observer contracts; declarations/fields: `scope` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 49–113 | Assert matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope; declarations/fields: `test: matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 114–156 | Assert exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution; declarations/fields: `test: exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 157–169 | Assert reservation permission admits only the selected name and measured bounded geometry; declarations/fields: `test: reservation permission admits only the selected name and measured bounded geometry` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 170–182 | Assert CLI parser records observed protocol only, never generated agent semantics or a pass; declarations/fields: `test: CLI parser records observed protocol only, never generated agent semantics or a pass` |

<a id="coverage-7d206141c57e"></a>

<a id="testsfrontendworkspacetestts-1"></a>

## [tests/frontend/workspace.test.ts](../../../../../tests/frontend/workspace.test.ts)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1–14, 18–85, 147, 197–199, 261, 404, 611, 823, 893, 954–955, 1232–1235, 1307, 1403, 1430–1433, 1463, 1487–1492, 1576, 1594–1596, 1679, 1734–1735, 1750, 1791, 1820 | Declarations/fixtures and integration for Persistent workspace navigation and explicit project/terminal journey assertions |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 15–17 | textContents — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `textContents` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 86–126 | Test harness fixture — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `fixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 127–131 | Test harness openSession — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `openSession` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 132–135 | Test harness action — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `action` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 136–142 | Test harness create — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `create` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 143–146 | Test harness paneAction — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `paneAction` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 148–196 | Assert Open in drawer preserves the terminal when ${failure} prevents a safe handoff; declarations/fields: `test: Open in drawer preserves the terminal when ${failure} prevents a safe handoff` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 200–260 | Assert ${mode}/${theme}/${width}: resolved tokens, focus, menu and warning preserve native draft and terminal; declarations/fields: `test: ${mode}/${theme}/${width}: resolved tokens, focus, menu and warning preserve native draft and terminal` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 262–293 | Assert ${mode}: three exact sessions across two projects retain real xterm owners; declarations/fields: `test: ${mode}: three exact sessions across two projects retain real xterm owners` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 294–331 | Assert observed unread coalesces noisy hidden output, survives refresh, and clears only deliberate viewing; declarations/fields: `test: observed unread coalesces noisy hidden output, survives refresh, and clears only deliberate viewing` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 332–385 | Assert attention has stable authorized counts/order and exact navigation without creation or lifetime actions; declarations/fields: `test: attention has stable authorized counts/order and exact navigation without creation or lifetime actions` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 386–403 | Assert stale metadata and late transport generations cannot claim cleanup or replay unread; declarations/fields: `test: stale metadata and late transport generations cannot claim cleanup or replay unread` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 405–421 | Assert direct New reserves one default-named locator before creating; Rename is separate; declarations/fields: `test: direct New reserves one default-named locator before creating; Rename is separate` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 422–443 | Assert End confirmation defaults Cancel and unknown cleanup preserves exact locator; declarations/fields: `test: End confirmation defaults Cancel and unknown cleanup preserves exact locator` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 444–456 | Assert changed actor invalidates siblings and clears private navigation observations; declarations/fields: `test: changed actor invalidates siblings and clears private navigation observations` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 457–475 | Assert a terminal admission actor mismatch invalidates every sibling without creating a replacement; declarations/fields: `test: a terminal admission actor mismatch invalidates every sibling without creating a replacement` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 476–494 | Assert Hide preserves renderer/socket and showing through the shared navigation does not Return; declarations/fields: `test: Hide preserves renderer/socket and showing through the shared navigation does not Return` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 495–512 | Assert project drafts and independent terminal owners survive detail switching; declarations/fields: `test: project drafts and independent terminal owners survive detail switching` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 513–533 | Assert pending rename keeps original ID while another project is deliberately selected; declarations/fields: `test: pending rename keeps original ID while another project is deliberately selected` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 534–556 | Assert search and This page change navigation only; New defaults to selected original project; declarations/fields: `test: search and This page change navigation only; New defaults to selected original project` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 557–567 | Assert legacy pending and unknown IDs never select/create a session; declarations/fields: `test: legacy pending and unknown IDs never select/create a session` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 568–588 | Assert obsolete caches are ignored; native inventory still permits deliberate discovery; declarations/fields: `test: obsolete caches are ignored; native inventory still permits deliberate discovery` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 589–610 | Assert current arrangement restores the exact selected ID and owner key, never a replacement; declarations/fields: `test: current arrangement restores the exact selected ID and owner key, never a replacement` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 612–624 | Assert test; declarations/fields: `test: test` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 625–640 | Assert storage write failure does not disable the live workspace; declarations/fields: `test: storage write failure does not disable the live workspace` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 641–714 | Assert real xterm owners survive pane moves, keyboard divider, maximize, compact and consolidation without effects; declarations/fields: `test: real xterm owners survive pane moves, keyboard divider, maximize, compact and consolidation without effects` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 715–766 | Assert sidebar bounds, tab overflow, pointer reorder and edge split preserve owners without IO; declarations/fields: `test: sidebar bounds, tab overflow, pointer reorder and edge split preserve owners without IO` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 767–798 | Assert installed control paths use the actual emitted project, chooser and original-target End UI; declarations/fields: `test: installed control paths use the actual emitted project, chooser and original-target End UI` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 799–822 | Assert authorized collection reports an observed other writer without needing a failed attachment; declarations/fields: `test: authorized collection reports an observed other writer without needing a failed attachment` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 824–892 | Assert measurement subscriptions retire on ${retirement}, including queued callbacks; declarations/fields: `test: measurement subscriptions retire on ${retirement}, including queued callbacks` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 894–917 | chooseFirstRepository — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `chooseFirstRepository` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 918–953 | workspaceIntroGeometry — Persistent workspace navigation and explicit project/terminal journey assertions; declarations/fields: `workspaceIntroGeometry` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 956–1064 | Assert explicit empty welcome, native repository choice and one coherent Project Create; declarations/fields: `test: first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1065–1091 | Assert separate browser-only Join, real account projection and no terminal creation; declarations/fields: `test: first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1092–1101 | Assert one explicit native terminal creation after Join; declarations/fields: `test: first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 1102–1150 | Assert measured terminal layout/geometry and bounded native frame insets; declarations/fields: `test: first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1151–1206 | Assert exact tab/navigation/context-menu ownership without new IO/lifetime effects; declarations/fields: `test: first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1207–1231 | Assert exact native terminal identity survives re-entry without second Create or writes; declarations/fields: `test: first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1236–1279 | Assert welcome ${viewport.width}/${viewport.height}: keyboard action and short-screen content stay reachable; declarations/fields: `test: welcome ${viewport.width}/${viewport.height}: keyboard action and short-screen content stay reachable` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1280–1306 | Assert setup: pending creation keeps its selection and disables navigation without duplicate writes; declarations/fields: `test: setup: pending creation keeps its selection and disables navigation without duplicate writes` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1308–1329 | Assert first use: ${outcome} creation is inspected, not replayed; declarations/fields: `test: first use: ${outcome} creation is inspected, not replayed` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1330–1353 | Assert first use: keyboard selection, Back and Change retain the repository without writes; declarations/fields: `test: first use: keyboard selection, Back and Change retain the repository without writes` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1354–1402 | Assert first use: superseded search and retired actor cannot publish stale private choices; declarations/fields: `test: first use: superseded search and retired actor cannot publish stale private choices` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1404–1429 | Assert first use: stopped project Start is explicit and authorized (${administrator}); declarations/fields: `test: first use: stopped project Start is explicit and authorized (${administrator})` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 1434–1462 | Assert workspace intro ${viewport.width}/${viewport.height}: long identity and actions remain reachable; declarations/fields: `test: workspace intro ${viewport.width}/${viewport.height}: long identity and actions remain reachable` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1464–1486 | Assert project header: ${state} never claims running or stopped; declarations/fields: `test: project header: ${state} never claims running or stopped` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1493–1530 | Assert first use: ${code} Join explains recovery without replay; declarations/fields: `test: first use: ${code} Join explains recovery without replay` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1531–1561 | Assert first use: second-project setup cancellation preserves the live renderer, input target and layout; declarations/fields: `test: first use: second-project setup cancellation preserves the live renderer, input target and layout` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 1562–1575 | Assert first use: incomplete and failed inventories never render guessed welcome; declarations/fields: `test: first use: incomplete and failed inventories never render guessed welcome` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1577–1593 | Assert departure during authorization cannot dispatch a late collection read; declarations/fields: `test: departure during authorization cannot dispatch a late collection read` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 1597–1629 | Assert failed/pending inventory yields explicit read-only Retry without guessed project controls; declarations/fields: `test: coherence recovery ${theme}/${width}: retry, stopped and Join failure keep explicit actions` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1630–1648 | Assert native project choice/Create actions stay explicit in recovery presentation; declarations/fields: `test: coherence recovery ${theme}/${width}: retry, stopped and Join failure keep explicit actions` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1649–1660 | Assert stopped Project requires explicit authorized Start; declarations/fields: `test: coherence recovery ${theme}/${width}: retry, stopped and Join failure keep explicit actions` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1661–1678 | Assert Join failure preserves explicit recovery and no replay; declarations/fields: `test: coherence recovery ${theme}/${width}: retry, stopped and Join failure keep explicit actions` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 1680–1733 | Assert coherence short ${theme}: long names, bounded menus and original disconnected identity; declarations/fields: `test: coherence short ${theme}: long names, bounded menus and original disconnected identity` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1736–1749 | Assert native recovery: ${status} project reads never masquerade as new configuration; declarations/fields: `test: native recovery: ${status} project reads never masquerade as new configuration` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1751–1790 | Assert terminal handshake with a mismatched native generation is refused without attaching; declarations/fields: `test: terminal handshake with a mismatched native generation is refused without attaching` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 1792–1819 | Assert native CSS loaded last preserves full-width Spaces and quiet controls; declarations/fields: `test: native CSS loaded last preserves full-width Spaces and quiet controls` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 1821–1843 | Assert partial project list explains missing items and retries reads without replacing a terminal; declarations/fields: `test: partial project list explains missing items and retries reads without replacing a terminal` |


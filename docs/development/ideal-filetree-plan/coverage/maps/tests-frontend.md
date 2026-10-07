# Tests frontend

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-9d1e53fd14b7"></a>
<a id="testsfrontenddrawer-controlstestts-1"></a>

## [tests/frontend/drawer-controls.test.ts](../../../../../tests/frontend/drawer-controls.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; test imports and Playwright fixture setup | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Imports and installs the shared browser fixture harness consumed by the control behavior tests. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 6–25, 66–84, 99–116, 169–176; project control mount is inert test; unconfirmed Create recovery test; malformed response actions refusal tests; copy own connection display test | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Verifies refresh stays read-only and the project controls do not acquire terminal ownership.; 4 named units assigned here; remaining selectors preserve each duty — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 26–43, 177–206; project view tabs and app switches test; reactive project view draft identity test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks tab selection and view switching do not issue native operations.; Checks view/Hide updates preserve unsent key draft identity and selection. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 44–65, 144–161; hidden access view synthetic key removal test; inert controls synthetic click test; uncertain key-save response test | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Proves hidden key removal cannot dispatch through a synthetic click.; Proves inert key controls cannot dispatch writes.; Checks an unknown key-save result cannot be shown as confirmed and remains recoverable. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 85–98, 124–143; stale project control replay test; close during initial read test | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Checks retired/stale project controls cannot refresh or replay on focus.; Checks disposal during a pending native read cannot publish controls or dispatch a mutation. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 117–123, 207–223, 238–244; repeated refresh terminal ownership test; hidden completed reads terminal attachment test; unavailable/incomplete state terminal refusal tests | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Checks repeated project refresh never acquires terminal ownership.; Checks showing completed project details after a hidden read does not attach a terminal.; Checks unavailable or incomplete project state refuses terminal creation. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 162–168; hidden lifecycle actions test | [P05](../../slices/projects.md#p05-project-startstop) | retained | Checks hidden Create and lifecycle handlers cannot dispatch. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |
| 224–237; terminal render after disposal test | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Checks terminal render readiness after disposal and reconnect cannot revive a retired session. — Current source block and named test consumer inspected in tests/frontend/drawer-controls.test.ts. |

<a id="coverage-ab64cfc46f94"></a>

## [tests/frontend/fixtures/drawer-fixture.ts](../../../../../tests/frontend/fixtures/drawer-fixture.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56, 69–85, 167–219; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Browser fixture declarations, project state knobs and inert mount setup; declarations/fields: `createFixture`, `Call`, `State`; Capture fixture HTTP dispatch, native generation and override/refusal mechanics; declarations/fields: `fetchFixture`; Install production project controls with test-owned transport, selectors, callbacks and global facade; declarations/fields: `createFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |
| 57–68, 117–125; prior detailed responsibility interval | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Synthetic reviewed network target/policy selection snapshot; declarations/fields: `network`; Synthetic reviewed network mutation/readback receipts; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |
| 86–101, 155–166; prior detailed responsibility interval | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Synthetic Create receipt preserving project outcome independently of optional network setup; declarations/fields: `fetchFixture`; Synthetic Project association/member/admin/execution observation projection; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |
| 102; prior detailed responsibility interval | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Synthetic explicit human Join receipt; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |
| 103–107, 141–142; prior detailed responsibility interval | [P05](../../slices/projects.md#p05-project-startstop) | retained | Synthetic explicit native Start/Stop receipt; declarations/fields: `fetchFixture`; Synthetic observed native Project lifecycle receipt; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |
| 108–116, 133–134, 143–154; prior detailed responsibility interval | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Synthetic development key application/removal response distinction; declarations/fields: `fetchFixture`; Synthetic saved human development key metadata; declarations/fields: `fetchFixture`; Synthetic target key fingerprints and ordinary project SSH connection; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |
| 126–132, 135–140; prior detailed responsibility interval | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Synthetic supported profile and observed Project readiness; declarations/fields: `fetchFixture`; Synthetic observed Project userspace version receipt; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/drawer-fixture.ts; manifest identical_prior confirms byte identity |

<a id="coverage-0d9022ac1da1"></a>

## [tests/frontend/fixtures/terminal-fixture.ts](../../../../../tests/frontend/fixtures/terminal-fixture.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41, 58–78, 156–287, 301–309; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Fixture state declarations, tracked test sockets/renderers and mount setup; declarations/fields: `createFixture`, `TerminalFixtureOptions`; 6 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/terminal-fixture.ts; manifest identical_prior confirms byte identity |
| 42–57, 79–112; prior detailed responsibility interval | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Synthetic exact issued/native terminal metadata; declarations/fields: `metadata`; Synthetic terminal reservation/lookup/explicit End HTTP state; declarations/fields: `fetchFixture` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/terminal-fixture.ts; manifest identical_prior confirms byte identity |
| 113–155, 288–300; prior detailed responsibility interval | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Test-owned socket frame capture/open/message/close for interactive attachment; declarations/fields: `Socket`; Emit test-owned ready frame and explicit invalid-locator fixture call; declarations/fields: `ready`, `mountInvalidLocator` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/fixtures/terminal-fixture.ts; manifest identical_prior confirms byte identity |

<a id="coverage-197a19f57480"></a>

## [tests/frontend/fixtures/workspace-model.ts](../../../../../tests/frontend/fixtures/workspace-model.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–301; workspace-model test fixture | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Shared test-only workspace model and canned records consumed by frontend unit suites; no production producer is claimed. — Current source inspected at tests/frontend/fixtures/workspace-model.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-09a2663e5b5a"></a>

## [tests/frontend/identity.test.ts](../../../../../tests/frontend/identity.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25, 53–65, 79; prior detailed responsibility interval | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Declarations/fixtures and integration for Identity connection/enrollment/grant/lease browser metadata boundaries; Assert subscription metadata remains bound to owner and excludes credential fields; declarations/fields: `test: subscription metadata remains bound to owner and excludes credential fields`; Assert named delegation/grant stable identities; declarations/fields: `grantView` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/identity.test.ts; manifest identical_prior confirms byte identity |
| 26–52; prior detailed responsibility interval | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Assert device enrollment only links to the selected provider authentication origin; declarations/fields: `test: device enrollment only links to the selected provider authentication origin` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/identity.test.ts; manifest identical_prior confirms byte identity |
| 66–78; prior detailed responsibility interval | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Assert execution lease metadata hides credential/binding delivery; declarations/fields: `leaseView` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/identity.test.ts; manifest identical_prior confirms byte identity |
| 80–177; prior detailed responsibility interval | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Assert identity widget keeps stable context and starts Codex with protected metadata-only request; declarations/fields: `test: identity widget keeps stable context and starts Codex with protected metadata-only request` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/identity.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-5c4dbe25534a"></a>

## [tests/frontend/journey-input.test.ts](../../../../../tests/frontend/journey-input.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 14–69; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declarations/fixtures and integration for Private installed-journey actor/target/action permission parsing; 5 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/journey-input.test.ts; manifest identical_prior confirms byte identity |
| 4–13; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness user — Private installed-journey actor/target/action permission parsing; declarations/fields: `user`; Test harness input — Private installed-journey actor/target/action permission parsing; declarations/fields: `input` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/journey-input.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-a85f845b1ff8"></a>

## [tests/frontend/native-project-controls.test.ts](../../../../../tests/frontend/native-project-controls.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–42, 69, 84–104, 124, 148, 159, 169, 179, 190, 205, 241–257; prior detailed responsibility interval | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Declarations/fixtures and integration for Native project contribution explicit Create/Join/lifecycle/key/network actions; Assert Create dispatches once with native generation and does not join or start; declarations/fields: `test: Create dispatches once with native generation and does not join or start`; Assert an unconfirmed Create does not retry a reservation; declarations/fields: `test: an unconfirmed Create does not retry a reservation` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |
| 43–68; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness fixture — Native project contribution explicit Create/Join/lifecycle/key/network actions; declarations/fields: `fixture`; 4 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |
| 70–83; prior detailed responsibility interval | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Assert native project controls mount inertly and preserve Forgejo content; declarations/fields: `test: native project controls mount inertly and preserve Forgejo content` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |
| 105–123, 125–147; prior detailed responsibility interval | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Assert Create uses the reviewed network binding and keeps network failure separate; declarations/fields: `test: Create uses the reviewed network binding and keeps network failure separate`; Assert network changes require target confirmation and dispatch once; declarations/fields: `test: network changes require target confirmation and dispatch once` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |
| 149–158; prior detailed responsibility interval | [P05](../../slices/projects.md#p05-project-startstop) | retained | Assert Stop requires shared-impact confirmation and never opens a terminal; declarations/fields: `test: Stop requires shared-impact confirmation and never opens a terminal` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |
| 160–168; prior detailed responsibility interval | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Assert Join is explicit and leaves saved SSH keys out by default; declarations/fields: `test: Join is explicit and leaves saved SSH keys out by default` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |
| 170–178, 180–189, 191–204, 206–240; prior detailed responsibility interval | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Assert private key input is rejected before a native request; declarations/fields: `test: private key input is rejected before a native request`; 4 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/native-project-controls.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-cf5e8a201965"></a>

## [tests/frontend/sodaspaces-http.test.ts](../../../../../tests/frontend/sodaspaces-http.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Declarations/fixtures and integration for Bounded raw probe HTTP/TLS transport contract; Assert probe HTTP decoding bounds headers/body and refuses ambiguous validators; declarations/fields: `test: probe HTTP decoding bounds headers/body and refuses ambiguous validators` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/sodaspaces-http.test.ts; manifest identical_prior confirms byte identity |
| 24–109; prior detailed responsibility interval | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Assert raw HTTPS keeps constrained CA verification, exact paths and bounded responses; declarations/fields: `test: raw HTTPS keeps constrained CA verification, exact paths and bounded responses` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/sodaspaces-http.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-452b81773833"></a>

## [tests/frontend/spaces-api.test.ts](../../../../../tests/frontend/spaces-api.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48, 109–134, 204–219; test imports and shared Spaces response fixture; Spaces visible row completeness test; degraded collection session/elevation test; bounded cursor and factory incompleteness test | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Imports API decoders and defines bounded shared repository, environment, terminal and factory samples used by decoder tests.; 4 named units assigned here; remaining selectors preserve each duty — Current source block and named test consumer inspected in tests/frontend/spaces-api.test.ts. |
| 49–55, 61–76; repositoryChoices native identity, Create eligibility and reservation contract assertions | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Asserts stable native repository identity, Create eligibility, bounded cursor and invalid reservation/project combinations; readiness-field meaning is split to P02. — current source tests/frontend/spaces-api.test.ts:49-55,61-76; parser consumes P01 repository response contract |
| 77–108; terminal metadata native binding test; invalid terminal metadata tests | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Checks terminal metadata binds to the exact native environment/user/repository and excludes retired lifetime fields.; Rejects malformed or mismatched terminal metadata values. — Current source block and named test consumer inspected in tests/frontend/spaces-api.test.ts. |
| 135–203, 225–250; factory authority verdict test; factory control state test; factory run rows test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks Spaces admits only valid factory authority verdicts and renders status text.; Checks factory pause/dispatch/unsettled control state admission and status rendering.; Checks factory run row admission, bounded values, and status rendering. — Current source block and named test consumer inspected in tests/frontend/spaces-api.test.ts. |
| 220–224; factory command idempotency identity test | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Checks factory command identities are unique fixed-width ledger keys. — Current source block and named test consumer inspected in tests/frontend/spaces-api.test.ts. |
| 251–280; per-row Tailnet state test | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Checks each Spaces row retains only its own valid Tailnet observation without inheriting collection-level state. — Current source block and named test consumer inspected in tests/frontend/spaces-api.test.ts. |
| 56–60; repositoryChoices associated Project provisioned readiness assertion | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Asserts the saved provisioned flag survives parsing for an already-associated Project independently of native repository identity or Create eligibility. — current source tests/frontend/spaces-api.test.ts:56-60; exercises project.provisioned false projection |

<a id="coverage-1b00ae83c417"></a>

## [tests/frontend/spaces-attention.test.ts](../../../../../tests/frontend/spaces-attention.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–42; Spaces attention test suite | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks attention indicators and their workspace view state in the Spaces frontend. — Current source inspected at tests/frontend/spaces-attention.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-747c939fc0ac"></a>

## [tests/frontend/tailnet.test.ts](../../../../../tests/frontend/tailnet.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 44–119; prior detailed responsibility interval | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Declarations/fixtures and integration for Observed native host Tailnet response projections; Assert Tailnet projections reject malformed, unsafe and mixed-version responses; declarations/fields: `test: Tailnet projections reject malformed, unsafe and mixed-version responses` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/tailnet.test.ts; manifest identical_prior confirms byte identity |
| 6–43; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness snapshot — Observed native host Tailnet response projections; declarations/fields: `snapshot` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/tailnet.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-99652b7b5667"></a>
<a id="testsfrontendterminaltestts-1"></a>

## [tests/frontend/terminal.test.ts](../../../../../tests/frontend/terminal.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–208; Spaces terminal test suite | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Checks terminal presentation lifecycle, rendering and retirement behavior in the Spaces frontend. — Current source inspected at tests/frontend/terminal.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-2f3c2de49d3b"></a>

## [tests/frontend/workspace-journey.test.ts](../../../../../tests/frontend/workspace-journey.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 49–113, 170–182; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declarations/fixtures and integration for Exact private matrix/terminal approval and native observer contracts; Assert matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope; declarations/fields: `test: matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope`; Assert CLI parser records observed protocol only, never generated agent semantics or a pass; declarations/fields: `test: CLI parser records observed protocol only, never generated agent semantics or a pass` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/workspace-journey.test.ts; manifest identical_prior confirms byte identity |
| 24–48; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness scope — Exact private matrix/terminal approval and native observer contracts; declarations/fields: `scope` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/workspace-journey.test.ts; manifest identical_prior confirms byte identity |
| 114–169; prior detailed responsibility interval | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Assert exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution; declarations/fields: `test: exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution`; Assert reservation permission admits only the selected name and measured bounded geometry; declarations/fields: `test: reservation permission admits only the selected name and measured bounded geometry` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/workspace-journey.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-9e91797b5f1c"></a>

## [tests/frontend/workspace-setup.test.ts](../../../../../tests/frontend/workspace-setup.test.ts)

Current browser assertions split by repository setup/creation, persisted Project readiness, verified actor search publication, and the independent workspace navigation/lifecycle assertions.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; test imports and workspace browser fixture setup | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Loads the Playwright test harness, shared workspace driver and screenshot support used by the browser assertions. — current source tests/frontend/workspace-setup.test.ts:1-7; fixture/driver producers imported |
| 8–18, 169–182, 187–199, 250–285, 291–304; projectPressed selected-project observation helper; second-project setup preserves active terminal, input target and workspace layout; missing setup return target falls back to first remaining Project | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Reads the current selected Project state from the mounted Spaces workspace to assert navigation restoration.; Asserts opening and cancelling another Project setup does not disconnect or replace the existing workspace terminal, focus target, socket or saved layout.; Asserts navigation restoration selects the first remaining workspace Project only after the saved return target disappears. — Current named units/source consumers; retained normalized source evidence records each selector |
| 19–61, 66–132, 142–148, 183–186, 200–249, 286–290; pending creation keeps repository selection and suppresses duplicate navigation/writes; uncertain/incomplete/rejected creation request and inspection setup; keyboard repository choice, back/change and Create-link picker flow; superseded repository search result fixture and query-selection flow; existing repository setup fixture and stopped Project readiness state; setup cancellation restores the non-first return target without creating the temporary repository; second-project repository selection before cancelling setup | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Asserts the selected repository remains attached while explicit Create is pending, setup navigation stays disabled, and only one creation request is sent.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 62–65, 149–158; refreshed creation outcome readiness interpretation; stopped Project is represented as not ready after refresh | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Asserts uncertain creation resolves to Join only after status inspection, incomplete creation opens Project needs inspection, and rejected creation returns to eligible Create.; Changes the existing Project observation to stopped and asserts the UI reports Project not ready without silently repeating creation. — current source tests/frontend/workspace-setup.test.ts:62-65; ready/incomplete/rejected post-refresh views; current source tests/frontend/workspace-setup.test.ts:149-158; observed.running false and not-ready assertion |
| 133–141; retired native actor cannot publish private repository choices | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Changes the native user/session context while a search is pending and asserts stale private repository choices and setup UI are removed. — current source tests/frontend/workspace-setup.test.ts:133-141; actor change, refresh and no-choice assertions |
| 159–168; explicit authorized Project Start action | [P05](../../slices/projects.md#p05-project-startstop) | retained | Asserts Start is offered and mutates Project lifecycle only for an administrator, while non-administrators cannot start it. — current source tests/frontend/workspace-setup.test.ts:159-168; explicit Start click and authorization branch |

<a id="coverage-7d206141c57e"></a>
<a id="testsfrontendworkspacetestts-1"></a>

## [tests/frontend/workspace.test.ts](../../../../../tests/frontend/workspace.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–185; Spaces workspace test suite | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks workspace view state, selection, navigation and reactive presentation behavior. — Current source inspected at tests/frontend/workspace.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

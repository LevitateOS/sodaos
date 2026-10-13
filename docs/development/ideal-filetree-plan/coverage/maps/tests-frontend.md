# Tests frontend

## Committed native correction tests at ac22514b (2026-10-13)

Incremental from `5d281760`; each added path is retained in the existing test
tree, with no new framework. Historical complete-body selectors remain separate.

| Path / named units | Existing owner | Actual subject and limit |
| --- | --- | --- |
| `tests/frontend/declared-routes.test.ts`: declared new/saved request cases and drawer host navigation | G01/S02/H06 | Synthetic API/browser state with supplied public navigation; native route/history qualification remains separate. |
| `tests/frontend/fixtures/browser-zoom.ts`: ChromeZoom, browserZoomFixture, setZoom, close | H06 | Actual Chromium tabs.setZoomSettings/setZoom/getZoom, verified CSS viewport/DPR, disposable profile and exact cleanup. |
| `tests/frontend/host-lifecycle.test.ts`: event-listener census, matching callback retention, foreign-generation refusal | S02/S04/H06 | Real Soda component callbacks and fixture socket/terminal identity; no installed native process identity claim. |
| `tests/frontend/native-container-geometry.test.ts`: source CSS/server fixture, page/panel/settings cases | S05/H06 | Actual 1×/4× browser zoom, Fountain styles and transcribed HTML; three cases pass, with native-rendered creation/dialog/container conditions open. |
| `tests/frontend/page-state-inventory.md`: subject/owner/evidence rows and masking rules | S04/H06 | Verification inventory; prepared or skipped subjects are not performed proof. |
| `tests/frontend/workspace-owner.test.ts`: owner/entry/subroute manifest, mount-handle/removal and mirror assertions | G01/S02/H06 | Source/manifest/type checks; native registration/one-owner navigation is not established by text assertions. |
| `tests/frontend/zoom-geometry.test.ts`: applyZoom, dialog bounds and ten settings/keyboard/creation/dialog/compact cases | S04/S05/H06 | Actual 200%/400% per-tab zoom with reachable actions, focus/dismissal, scrolling and pane content; all ten pass locally. |

Changed settings/workspace drivers add optional page factories for the same
zoom fixture; ordinary callers retain their previous browser/viewport lifecycle.
One unchanged default caller per driver passed. Existing terminal/workspace/
panel/contract fixtures are joined to the supplied lifecycle and mandatory
renderer/socket assertions. Those changes preserve their existing slices and
do not qualify native or installed composition by themselves.

Scoped local geometry check at `2f95baaef4a8f1519f31348de2e6bdcfd0b9d3c6` (2026-10-12): existing settings/setup/dialog fixtures preserve local S05/S04 duties and H06 measurement/focus assertions. Fifteen settings, three setup, nine dialog and three empty-pane cases pass; reduced CSS viewport/allocation proxy and default colors only, with actual browser zoom/native/installed qualification open.

[Responsibility map index](README.md) · [Coverage scope](../README.md).

## [tests/frontend/empty-pane-geometry.test.ts](../../../../../tests/frontend/empty-pane-geometry.test.ts)

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; existing workspace fixture setup and viewport matrix | H06 | retained | Reuses the existing browser harness; no parallel test framework. |
| 16–88; empty-pane callback, pane/action/message containment and reachable-focus assertions | S05 | retained | Whole added body inspected at `2f95baaef4a8f1519f31348de2e6bdcfd0b9d3c6`; three real local-browser cases pass, without native shell, actual browser zoom or installed claims. |

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

<a id="coverage-6c0243e8e3fc"></a>

## [tests/frontend/factory-access-response.test.ts](../../../../../tests/frontend/factory-access-response.test.ts)

Delta duty inspection at `93fb3513`. Access-parser tests: exact permission/usage units without conversion, non-owner/missing-budget honesty, visible unknown codes, real call-unit allowances, and malformed-DTO refusal.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 46–109; owner permission/usage DTO parses with exact units and no conversion test; non-owner usage stays a status test; unknown reason and status codes stay visible test; planning allowances parse with real call units test; malformed permission DTO is refused test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks access DTO admission, unit honesty and refusal without partial use — Current source block and named test consumer inspected in tests/frontend/factory-access-response.test.ts. |

<a id="coverage-4a1dbb58d192"></a>

## [tests/frontend/factory-activity.test.ts](../../../../../tests/frontend/factory-activity.test.ts)

Delta duty inspection at `93fb3513`. Activity-parser tests: repository binding/stamps/safe links, item kind/target/link binding, and unknown-instead-of-invented timestamps.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 43–96; factory activity keeps repository binding, stamps and safe links test; factory activity items bind kind, target and exact native link test; activity timestamps stay unknown test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks activity payload admission and honest timestamps — Current source block and named test consumer inspected in tests/frontend/factory-activity.test.ts. |

<a id="coverage-12d21b318c35"></a>

## [tests/frontend/factory-progress-response.test.ts](../../../../../tests/frontend/factory-progress-response.test.ts)

Delta duty inspection at `93fb3513`. H-R/H-A/L06 parser tests: identity-bound overview/plan/allowance/effects/access/revision reads, foreign masking, and wrong-scope/bad-page refusal.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 59–223; request overview binds direction, counts, mappings and attention test; issue plan binds freshness, plan and comparisons test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks overview/plan identity binding — Current source block and named test consumer inspected in tests/frontend/factory-progress-response.test.ts. |
| 224–324; planning allowance binds bounded units and consumption test; lifecycle configs bind enablement, proof, actor and targets test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks allowance/config binding — Current source block and named test consumer inspected in tests/frontend/factory-progress-response.test.ts. |
| 325–524; lifecycle actions bind attempts, revocation and per-target results test; lifecycle actions admit unclaimed rows test; masked foreign choices stay visible test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks action results, unclaimed rows and masking — Current source block and named test consumer inspected in tests/frontend/factory-progress-response.test.ts. |
| 525–686; repository access binds standing, verdict and bounded planning listing test; repository access rejects wrong scope test; revision history binds before/after and affected work test; revision history rejects wrong scope test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks access/revision binding and refusal — Current source block and named test consumer inspected in tests/frontend/factory-progress-response.test.ts. |
| 473–544; lifecycle actions bind unequal identities and a distinct native target test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `93fb3513`: unequal-identity and distinct-target test — Current source block and named test consumer inspected in tests/frontend/factory-progress-response.test.ts. |
| 104–179; request overview normalizes omitted sparse by_state buckets to zero test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `31caf48a` (V/a9bf): Q-exact sparse payload with malformed-count rejections — Current source block and named test consumer inspected in tests/frontend/factory-progress-response.test.ts. |

<a id="coverage-e1cdbc7969dc"></a>

## [tests/frontend/factory-progress-view.test.ts](../../../../../tests/frontend/factory-progress-view.test.ts)

Delta duty inspection at `93fb3513`. Progress model/render tests: generations, overview counts, why-next, prerequisites, revision/allowance, request overview, recorded plan/choice, lifecycle gating, planning reference, access standing and revision history.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 84–260; generation records bind tasks to native issues test; overview counts factual states test; overview distinguishes missing, waiting and unavailable test; why-next needs no consultation test; prerequisites name waiting reasons test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks generation/overview/why-next/prerequisite models — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 261–425; revision and allowance stay attributable test; allowance shows bounded units test; request overview projects evidenced counts test; recorded plans name freshness test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks revision/allowance/overview/plan models — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 426–621; recorded choices explain selection, fairness and masked winners test; lifecycle effects gate on proof test; each optional effect gates on its own checkpoint test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks choice/lifecycle models — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 622–738; planning reference finds this request test; access standing states powers test; revision history composes before/after test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks planning-reference/standing/history models — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 548–608; lifecycle results name submitter, authorizer and distinct native target test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `93fb3513`: submitter/authorizer/target rendering test — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 810–823; overview scope feeds effective mappings without capabilities test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: capability-free mapping feed — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 824–847; overview scope feeds bound issues with read capabilities test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: capability-bound issue feed — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |
| 848–879; child contexts derive from scope with exact keys and runs test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: exact-key child derivation — Current source block and named test consumer inspected in tests/frontend/factory-progress-view.test.ts. |

<a id="coverage-0c23db822fa9"></a>

## [tests/frontend/factory-settings-controls.test.ts](../../../../../tests/frontend/factory-settings-controls.test.ts)

Delta duty inspection at `93fb3513`. Rendered settings-control regressions: activity links/preferences, authority/capacity, policy, preparation decisions/identity/retry, budget/sponsorship, late-reply fencing, execution usage honesty and load-interruption recovery.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 6–163; Factory activity preserves exact native links test; Factory activity rejects foreign links test; Factory settings render current authority test; Factory settings expose stale authority test; repository policy preserves canonical role selections test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks activity/settings/policy rendering — Current source block and named test consumer inspected in tests/frontend/factory-settings-controls.test.ts. |
| 164–830; preparation controls render API evidence test; preparation decisions copy reviewed facts test; initial source, acceptance, approval and role preparation test; preparation retry reuses admitted identities test; reload adopts role identity test; reaccepting requirements blocks preparation test | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Checks preparation decision/identity behavior — Current source block and named test consumer inspected in tests/frontend/factory-settings-controls.test.ts. |
| 831–1088; provider budget and sponsorship test; provider budget reports forbidden test; late replies cannot write into newer context test | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Checks budget/approval fencing — Current source block and named test consumer inspected in tests/frontend/factory-settings-controls.test.ts. |
| 1089–1223; Factory execution usage reports exact minutes test; Factory execution usage stays unavailable test; Factory settings reload after tab flap test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `93fb3513`: usage-honesty and reload tests — Current source block and named test consumer inspected in tests/frontend/factory-settings-controls.test.ts. |
| 1228–1295; Factory settings mount optional effects with authorizer/submitter context and retire on tab switch test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: effects mount context with tab-switch retirement — Current source block and named test consumer inspected in tests/frontend/factory-settings-controls.test.ts. |

<a id="coverage-481f0d5e01fd"></a>

## [tests/frontend/factory-work-controls.test.ts](../../../../../tests/frontend/factory-work-controls.test.ts)

Delta duty inspection at `93fb3513`. Real-engine work control tests: overview rendering, detail evidence with all six interventions, direction-revision allowance adjust with local refusal, per-checkpoint effect gating with executor-refusal mapping, and plan rendering.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 263–409; work overview renders direction test; work detail renders evidence links and runs all six interventions test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks overview/detail controls on a real engine — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. Superseded in part at `31caf48a`: the direction test was converted to the authoritative H-R render; see the superseding rows below. |
| 303–324; work overview renders the authoritative request overview on a real engine test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `31caf48a` (V/a9bf): converted from the removed direction test; authoritative H-R render — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. |
| 854–912; intent overview failures surface unavailable with no fallback reads test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `31caf48a` (V/a9bf): endpoint/parser failures surface unavailable without fallback reads — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. |
| 410–505; direction revision shows allowance reference test; allowance increase with blank effect test; allowance below recorded usage is refused locally test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks allowance adjust and local refusal — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. |
| 506–627; optional effects gate per checkpoint test; optional effects map unavailable executor refusal test; work plan renders recorded steps test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks effect gating, refusal mapping and plan rendering — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. |
| 598–679; optional effects link a journaled pull number and leave unclaimed rows text-only test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: journaled-PR links with text-only unclaimed rows — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. |
| 711–843; overview scope feeds derived plan, detail and revision mounts, then retires test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: scope-fed mounts with retirement — Current source block and named test consumer inspected in tests/frontend/factory-work-controls.test.ts. |

<a id="coverage-c5572d2bff36"></a>

## [tests/frontend/factory-work-response.test.ts](../../../../../tests/frontend/factory-work-response.test.ts)

Delta duty inspection at `93fb3513`. Work-parser tests: issue-detail and repository-summary identity binding, command-outcome attribution, and error/pending honesty.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 80–229; issue detail binds acceptance, readiness, checks and merge test; issue detail binds attempt, publication, review, closure and links test; repository summary binds queue, setup and authority test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks detail/summary identity binding — Current source block and named test consumer inspected in tests/frontend/factory-work-response.test.ts. |
| 230–388; pause and resume receipts bind the exact command test; run intervention outcomes bind the exact run test; project stop binds running state test; error envelopes yield detail or nothing test; command read-back refusals stay attributable test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks outcome attribution and refusal honesty — Current source block and named test consumer inspected in tests/frontend/factory-work-response.test.ts. |

<a id="coverage-1ffe2dfc9c57"></a>

## [tests/frontend/factory-work.test.ts](../../../../../tests/frontend/factory-work.test.ts)

Delta duty inspection at `93fb3513`. Read-failure missing/unavailable separation test.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 5–22; read failures separate missing from unavailable test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks failure classification — Current source block and named test consumer inspected in tests/frontend/factory-work.test.ts. |

<a id="coverage-8ba1cec4e3df"></a>

## [tests/frontend/factory-work-view.test.ts](../../../../../tests/frontend/factory-work-view.test.ts)

Delta duty inspection at `93fb3513`. Work-view tests: merge+closure delivery evidence, check/readiness verdicts, detail evidence models, capability-gate reasons, and intervention outcome separation.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 61–207; delivery needs both merge and closure evidence test; check and readiness verdicts map test; detail models carry evidence test; detail models carry attempt, review, staleness test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks delivery evidence and detail models — Current source block and named test consumer inspected in tests/frontend/factory-work-view.test.ts. |
| 208–304; capability gates name the recorded refusal reason test; intervention outcomes separate confirmed, pending, uncertain and failed test; interventions pair scope, impact and results test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Checks gating and intervention outcomes — Current source block and named test consumer inspected in tests/frontend/factory-work-view.test.ts. |
| 307–339; upfront capabilities map known flags and stay permissive when unknown test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `deb79186`: permissive capability mapping — Current source block and named test consumer inspected in tests/frontend/factory-work-view.test.ts. |

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

<a id="coverage-696a2b4e0f2b"></a>

## [tests/frontend/fixtures/factory-work-fixture.ts](../../../../../tests/frontend/fixtures/factory-work-fixture.ts)

Delta duty inspection at `93fb3513`. Work-view fixture: stable synthetic identities (intent/revision/task/run/project/environment), call/route shapes and view selectors for the work control tests.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–118; file scaffold; WorkCall; WorkRoute; INTENT; REVISION; TASK_A; TASK_B; RUN; PROJECT; ENVIRONMENT; WorkView | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current scaffold duty: file scaffold; 10 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-ab768aa1e036"></a>

## [tests/frontend/fixtures/intent-model.ts](../../../../../tests/frontend/fixtures/intent-model.ts)

Delta duty inspection at `93fb3513`. Scripted intent-store fixture for creation/review tests: allowance endpoints, head-GET behavior and saved/resumed detail reads consumed by the intent journey suites.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–153; file scaffold; createIntentModel | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

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
| 357–367; enableFactoryRun issue parameter | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Delta at `a835c02d`: fixture run issue becomes a caller parameter defaulting to '42'. Prior row retains its pinned scope. |

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

<a id="coverage-c0d90b2ec261"></a>

## [tests/frontend/project-environment-controls.test.ts](../../../../../tests/frontend/project-environment-controls.test.ts)

Delta duty inspection at `93fb3513`. Environment control tests: no-implicit-create/join/start, managed binding, network failure/recovery, shared-network administration, member read-only scope, startup uncertainty, legacy-OS read-only display, stop/start separation, readiness links and non-admin read-only state.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 6–204; create never implicitly joins test; managed Create submits reviewed binding test; network failure after Create test; shared Network requires administration test; members get only observed access test; network startup uncertainty test; legacy OS observation is explicit test; OS text stays text test; Stop requires confirmation test; stopped environment has Start only test | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Checks environment control behavior — Current source block and named test consumer inspected in tests/frontend/project-environment-controls.test.ts. |
| 205–226; overview readiness links summarize Access/Network test; non-administrator sees read-only state test | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Delta at `93fb3513`: readiness-link and read-only tests — Current source block and named test consumer inspected in tests/frontend/project-environment-controls.test.ts. |
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

<a id="coverage-3993a8d79cb4"></a>

## [tests/frontend/workspace-discovery.test.ts](../../../../../tests/frontend/workspace-discovery.test.ts)

Delta duty inspection at `93fb3513`. Discovery tests: empty-state distinctions, stable recent ordering, attention split, lifecycle grouping and honest unbound labels.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 17–111; empty discovery distinguishes loading, unavailable, filtered and truly empty test; recent keys order stable identities test; attention split keeps ordinary sessions out test; saved requests group by lifecycle test; unbound requests label their missing repository test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks discovery grouping and honesty — Current source block and named test consumer inspected in tests/frontend/workspace-discovery.test.ts. |

<a id="coverage-06d45a0053ed"></a>

## [tests/frontend/workspace-intent.test.ts](../../../../../tests/frontend/workspace-intent.test.ts)

Delta duty inspection at `93fb3513`. Intent-first journey tests: entry/department, draft save/resume, review/correction, challenges, unsupported honesty, failure preservation, provenance, votes, accepted-review/generation status, autonomous baseline, and the full independent-allowance lifecycle plus Back-focus behavior.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 351–577; intent-first entry stores department test; native generation failure retains acceptance test; save draft reports saved state test; department dialog validates test; unavailable proposal keeps objective editable test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Checks entry/department/draft behavior — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 446–488; review refusal retains acceptance and asks for reconsideration without retry test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `79b4a524` (U/8682): exact refusal message with acceptance retained, one accepted POST and no non-Factory mutation — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 578–785; saved draft resumes by request id test; declined direction resumes for revision test; saved inventory lists states test; saved inventory names resume actions test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Checks resume and inventory behavior — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 786–1050; review leads with walkthrough test; change something submits correction test; assumption challenges batch recommendations test; unsupported objectives show unknowns test; planning and judgment failures preserve draft test; human decisions record provenance test; idea cards keep votes test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Checks review/challenge/provenance behavior — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 1051–1187; accepted review shows authorization, generation progress and unsaved loss test; settled autonomous baseline advances test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Checks accepted-review and baseline behavior — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 218; generation fixture composite authorization_revision | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `2f382509` (U07/b619): fixture GET /generation emits the actual composite shape for ordinary and stale cases — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 1155–1159; accepted-review committed state step | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `2f382509` (U07/b619): accepted-review progress regression extended with the committed state step — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 1188–1416; explore initializes independent allowance test; allowance-only decrease preserves usage test; raising planning calls states effect test; lowering below used calls fails test; stale allowance retries once test; refused allowance blocks exploration test; untouched control adopts raised funding test; back-to-review focuses heading test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Checks independent-allowance lifecycle and Back focus — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 1418–1446; resumed accepted request exports its durable repository on intent-resumed test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `deb79186` (U08): durable-repository export on resume — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 1453–1542; review top-up funds an exhausted allowance without touching the draft test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `39d9ad6a` (U06/06e6): review top-up funds exhaustion, draft untouched — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |
| 1543–1557; unchanged raise posts nothing and asks nothing again test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `39d9ad6a` (U06/06e6): unchanged raise posts nothing — Current source block and named test consumer inspected in tests/frontend/workspace-intent.test.ts. |

<a id="coverage-2f3c2de49d3b"></a>

## [tests/frontend/workspace-journey.test.ts](../../../../../tests/frontend/workspace-journey.test.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 49–113, 170–182; prior detailed responsibility interval | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declarations/fixtures and integration for Exact private matrix/terminal approval and native observer contracts; Assert matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope; declarations/fields: `test: matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope`; Assert CLI parser records observed protocol only, never generated agent semantics or a pass; declarations/fields: `test: CLI parser records observed protocol only, never generated agent semantics or a pass` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/workspace-journey.test.ts; manifest identical_prior confirms byte identity |
| 24–48; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness scope — Exact private matrix/terminal approval and native observer contracts; declarations/fields: `scope` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/workspace-journey.test.ts; manifest identical_prior confirms byte identity |
| 114–169; prior detailed responsibility interval | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Assert exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution; declarations/fields: `test: exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution`; Assert reservation permission admits only the selected name and measured bounded geometry; declarations/fields: `test: reservation permission admits only the selected name and measured bounded geometry` — docs/development/ideal-filetree-plan/coverage/maps/tests-frontend.md#tests/frontend/workspace-journey.test.ts; manifest identical_prior confirms byte identity |

<a id="coverage-223e47bacb7e"></a>

## [tests/frontend/workspace-recovery.test.ts](../../../../../tests/frontend/workspace-recovery.test.ts)

Delta duty inspection at `93fb3513`. Recovery honesty tests: unguessed welcome, no late dispatch after departure, refused mismatched terminal handshake, partial-list explanation, and offline-vs-failure distinction with retry recovery.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 96–287; first use: incomplete and failed inventories never render guessed welcome test; departure during authorization cannot dispatch test; terminal handshake with mismatched generation is refused test; partial project list explains missing items test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks recovery honesty — Current source block and named test consumer inspected in tests/frontend/workspace-recovery.test.ts. |
| 288–317; offline refresh says offline instead of generic failure test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `93fb3513`: offline-distinction test — Current source block and named test consumer inspected in tests/frontend/workspace-recovery.test.ts. |

<a id="coverage-c2a26e926cdd"></a>

## [tests/frontend/workspace-setup-inventory-resume.test.ts](../../../../../tests/frontend/workspace-setup-inventory-resume.test.ts)

Delta duty inspection at `deb79186`. Inventory-click accepted-resume test: restores the committed overview with native links and performs no writes.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 6–46; first use: inventory-click accepted resume restores its overview with native links and no writes test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks write-free accepted resume — Current source block and named test consumer inspected in tests/frontend/workspace-setup-inventory-resume.test.ts. |
<a id="coverage-def1180aecb0"></a>

## [tests/frontend/workspace-setup-routes.test.ts](../../../../../tests/frontend/workspace-setup-routes.test.ts)

Delta duty inspection at `93fb3513`. Setup-route tests: request-id scoping, sub-path preservation, saved/resumed intent identity, and inventory-intent parsing with malformed rejection.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 13–77; request id reads only from app-root request routes test; request and new routes preserve sub-path test; intent identity reads saved detail test; inventory intents parse saved list test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks route scoping and identity reads — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 129–157; overview scope feed keeps valid entries and retires on malformed payloads test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186`: scope-feed admission with malformed retirement — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 158–167; scope equality compares values so refresh re-dispatch does not churn the root test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186`: value-equality churn guard — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 168–187; work children derive only from a live scope matching the retained identity test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186`: live-scope child gating — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 193–202; watchProjectA; watchEvent; watchSpace | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Delta at `a835c02d`: watch-request test fixtures — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 204–222; watch requests resolve to the exact live space run or stay unresolved test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `a835c02d`: exact live-run resolution with unresolved stays — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 224–248; watch request payloads admit exact bindings and ignore malformed events test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `a835c02d`: exact-binding admission with malformed-event rejection — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 250–270; Detail runs select the single live run and bind its owning project test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `a835c02d`: live-run selection with owning-project binding — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
| 99–122; new route matches only the exact admitted fresh-entry path test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `59b18572`: exact-boundary fresh-entry match — Current source block and named test consumer inspected in tests/frontend/workspace-setup-routes.test.ts. |
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
| 98–217; first use: keyboard selection and post-commit Change test; first use: committed inspection mounts overview test; first use: uncommitted review mounts no overview test; first use: picker Back preserves direction test; first use: picker summarizes authorization test; first use: resumed request suggests repository test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `93fb3513`: first-use inspection/picker tests. Prior rows retain their pinned scope. — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 455–503; setup: join states account test; setup: welcome footer tracks steps test | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Delta at `93fb3513`: join/footer creation tests. Prior rows retain their pinned scope. — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 504–553; setup: failed creation preserves request test; setup: intent entry receives session actor test | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `93fb3513`: accepted-request preservation and intent entry tests. Prior rows retain their pinned scope. — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 554–665; setup: saved draft resumes from request route test; setup: unknown request route test; setup: saving draft syncs route test; setup: new entry canonicalizes route test; setup: discovery groups saved drafts test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `93fb3513`: draft-route and discovery tests. Prior rows retain their pinned scope. — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 218–259; first use: resumed accepted request restores its overview with native links and no writes test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186`: write-free accepted-resume restore — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 260–308; first use: committed inspection mounts plan, detail and revision from the overview scope test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `deb79186`: scope-fed child mounts — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 309–371; setup: detail Watch binds the live run target into an observe-only watch test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `a835c02d`: live target bound into an observe-only watch — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 309–390; setup: detail Watch binds the live run target into an observe-only watch test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `8fbed0a7`: extended with reveal (visible watch surface with live output, no writes, leaves setup), return (saved request resumes with committed identity) and dedup (second Watch reveals the same live watch: one surface, one socket, no new status calls); replaces the prior hidden-surface assertions above, which stay as historical record — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 391–486; setup: Watch reveal clears only obstructing filters, hidden watch and hidden navigation test | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Delta at `8fbed0a7`: three obscuring-phase regression — search/attention obstruction cleared, hidden watch revealed without a second connection, compact-layout navigation restored — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 856–898; setup: direct /workspace/new load activates fresh entry in populated workspace test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `59b18572`: direct-load fresh entry — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 899–915; setup: retained owner follows native forward to /workspace/new without remount test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `59b18572`: native-forward retained presentation — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 916–975; setup: native New while resuming starts a blank entry and saves a new identity test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `39d9ad6a` (C/7657): blank entry with new saved identity — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 976–1090; setup: native New while viewing accepted starts blank and saves apart test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `71f79938` (C/63d2): accepted blank capture with separate save identity, watch preservation and intact re-resume — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |
| 1091–1152; setup: native New on idle Configure starts blank and saves apart test | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Delta at `71f79938` (C/71f7): idle-configure blank capture with separate save identity and intact re-resume — Current source block and named test consumer inspected in tests/frontend/workspace-setup.test.ts. |

<a id="coverage-7d206141c57e"></a>
<a id="testsfrontendworkspacetestts-1"></a>

## [tests/frontend/workspace.test.ts](../../../../../tests/frontend/workspace.test.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–185; Spaces workspace test suite | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | retained | Checks workspace view state, selection, navigation and reactive presentation behavior. — Current source inspected at tests/frontend/workspace.test.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="current-workspace-panel-owner-test"></a>

## [tests/frontend/workspace-panel.test.ts](../../../../../tests/frontend/workspace-panel.test.ts)

Selective current source at `0ace2383` (2026-10-11); historical body selectors above retain their original pins.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| native panel yields saved terminal restoration to the full Spaces page; complete file | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Native page mount retires the shared panel first; ordinary repository navigation preserves its terminal renderer and socket owner. |

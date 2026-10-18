# Visible experience and development qualification

[Remaining-work catalog](../factory-remaining-tasks.md#task-list) owns execution order. These packets preserve the original requirement authorities; the catalog's consolidated outcomes own current status; the closed checklist retains evidence and former leases. Implementation is stopped. The executable remainder refines each existing parent without adding counted outcomes. Source locators identify the current owner, not blanket write permission; use the catalog’s next-step disposition and actual prerequisite outputs.

<a id="r42"></a>
## R42 — Finish the persisted factory read model and API

**Work:** Implement and integrate. **Owner lane:** C. **Accepted inputs:** [R10](01-boundaries.md#r10).

**Current (wind-down):** Extend canonical read model with workflow producers and persisted role/verb feed. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Extend the one existing read model for actual phases, plans/tasks/lanes/checkpoints, review/acceptance costs, blocked owner, reassessment/stall/repair and exact app verification. Use producer-owned records and native links.

**Ralph absorption addition:** Present R32's communication and the actual
procedure as durable role-authored feed history. Keep author/role/run/assignment,
communication verb, recorded time/revision, linked issue/PR/plan/finding and
reported versus confirmed outcome distinct. Retain post revisions and related
question/reply history with stable identities, bounded ordered paging and honest
missing/truncated/stale state. Reuse canonical evidence owners and existing
visibility/authority checks; history must survive reload and service restart.
Exact storage/post schema follows the
[feed design](../../design/factory-feed-and-ralph-integration.md), not an invented
second task database.

**Additional acceptance:** All eight verbs and their role-specific formal
results have real producer-to-store-to-API coverage. Reads spend nothing and do
not mutate the workflow. Cross-account/repository isolation, ordering/revision
consistency and missing evidence are tested; a request, report or accepted
submission is never mislabeled as an observed downstream effect.

**Accept only when:** Read routes are bounded, authorized, secret-free and spend nothing; submitted/pending is not verified success. Each new producer supplies persisted data and integration tests; no fixture-only DTO or parallel dashboard.

**Executable remainder:** First define the combined-query/cursor contract against existing `factory_journey_readback.go` Store/API records; this preparation is ready. Consume R32's post/comment/revision output to implement subscription preferences and the combined feed; do not replace canonical workflow records. Build one authorized combined subscribed-feed query across projects with newest posts first and a deterministic unique tie-break, bounded keyset paging and rechecked visibility. Thread/comment reads retain their own provenance. Then add each workflow producer's reported/applied/verified projection as its receipt becomes available; full qualification waits for the listed consumers, not initial query implementation. Test equal timestamps, concurrent insert/page continuation, cross-project/account denial, unsubscribe/access revocation and restart persistence in PG/API. Subscription never grants access; forbidden entries and counts must not leak. Expose comment-to-plan/issue result/failure links without claiming submitted work succeeded.

**Principal source owners:** [internal/store/factory_journey_readback.go](../../../internal/store/factory_journey_readback.go), [internal/web/api/factory_journey_readback.go](../../../internal/web/api/factory_journey_readback.go), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go).

**Additional runtime/qualification gates:** [R13](02-prompt-to-pr.md#r13), [R17](02-prompt-to-pr.md#r17), [R24](03-review-merge-reassessment.md#r24), [R28](03-review-merge-reassessment.md#r28), [R29](03-review-merge-reassessment.md#r29), [R31](04-controls-repair-verification.md#r31), [R32](04-controls-repair-verification.md#r32), [R33](04-controls-repair-verification.md#r33), [R34](04-controls-repair-verification.md#r34), [R35](04-controls-repair-verification.md#r35), [R36](04-controls-repair-verification.md#r36), [R37](04-controls-repair-verification.md#r37), [R39](04-controls-repair-verification.md#r39), [R40](04-controls-repair-verification.md#r40). Source-only work cannot release these gates.

<a id="r43"></a>
## R43 — Complete Work/Spaces and current native frontend composition

**Work:** Implement and qualify. **Owner lane:** C. **Accepted inputs:** [R42](05-experience-and-development.md#r42).

**Current (wind-down):** Fresh Lit Work/Spaces feed and changed native joins; reuse suitable independent Ralph code/assets, seven roles and stable reading/drafts. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Render actual producer/read-model states, links, actionable waits and admitted intervention in existing Work/Spaces. Reuse native ownership/lifecycle/navigation/terminal work and requalify changed joins, geometry, accessibility, assets and attribution.

**Ralph absorption addition:** Absorb Ralph's useful capabilities and interaction
behavior into a fresh Lit/TypeScript interface in this same Work/Spaces experience.
The owner explicitly excluded React integration; Ralph's existing appearance
does not constrain Soda's design. Reuse suitable framework-independent code and
assets individually. First map
the complete intended procedure to role, permitted MCP action, feed component,
evidence condition and concrete reuse source. Use one actual avatar/typed
question/reply/history flow to define the new Lit components, their Soda
appearance and host/resource lifetime behavior. Preserve the seven established personas as
primary authors; select character/artwork assignments explicitly, with technical
specialties as a separate possible dimension. Include structured components for
direction, dependencies, investigation, task lanes/checkpoints, review/correction,
JEV decisions, waits, reassessment/repair and exact app verification. Reuse
Ralph's question, finding, knowledge, proposal and progress presentations and the
marketing film's artifact patterns where they fit real recorded work. Keep one
persistent Workspace owner and the existing terminal/resource lifetimes.

**Additional acceptance:** Real persisted role/verb entries, questions/replies,
evidence links and result confirmations work through both native entry orders,
navigation/reload/account changes and narrow/wide light/dark views. New updates
preserve reading position, expanded entries and drafts. Review/fix/merge and app
verification remain distinguishable. Required keyboard/focus behavior,
accessibility and any reused asset attribution survive integration. The source
reuse map accounts for retained independent code/assets, behavior implemented
in fresh Lit components and demonstrated new work. No React runtime or React
component integration is introduced.

**Accept only when:** Real persisted API-to-browser journeys pass both native entry orders, page/panel/history/reload/account changes and mandatory terminal/renderer/socket assertions. No injected links, absent-renderer success or reopening all 57 NFX tasks as new implementation.

**Executable remainder:** First select a concrete Ralph avatar/question/comment flow and suitable framework-independent assets with provenance, then design it in the existing Lit Work/Spaces owners (`sodaspaces-factory-work-journey-view.ts`, `sodaspaces-workspace.ts`). Project-only feed composition is superseded: the main view combines subscribed projects/feeds, newest posts on top. After R32/R42 supply actual records, implement subscriptions, persisted comment/reply composition, source discussion links and visible agent plan/issue outcomes. Then cover seven-role workflow cards and native joins. Use focused browser tests for reading position, drafts, keyboard/focus, account switch, late updates and pagination, plus changed native entry/terminal lifetime assertions. R03 gates native acceptance; no React integration, copied mock-only feed or reopening all NFX work.

**Principal source owners:** [frontend/spaces/soda-extension.ts](../../../frontend/spaces/soda-extension.ts), [frontend/spaces/sodaspaces-terminal-attachment.ts](../../../frontend/spaces/sodaspaces-terminal-attachment.ts), [docs/development/native-frontend-correction-tasks.md](../../../docs/development/native-frontend-correction-tasks.md).

**Additional runtime/qualification gates:** [R03](01-boundaries.md#r03). Source-only work cannot release these gates.

<a id="r44"></a>
## R44 — Requalify retained Project, provider, human and operator foundations

**Work:** Qualify and repair demonstrated gaps. **Owner lane:** A. **Accepted inputs:** [R03](01-boundaries.md#r03), [R10](01-boundaries.md#r10).

**Current (wind-down):** Current Project preparation/provider/broker/human/operator qualification. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Qualify current Project preparation, protected service identities/Unix mappings, broker custody, provider grants and human SSH/terminal/operator foundations before live factory dispatch. Inspect and repair demonstrated joins only. R41 later composes factory-specific intervention with these accepted foundations.

**Shared Filetree/library consumers:** Retain completed identity, SQL, HTTP/WS
and library cuts. The remaining L08/LA-G4 and Q1/Q2/Q3/Q9 witnesses use actual
admitted DSN/auth/TLS, transaction deadlines/cancellation, native Unix peer and
installed service identities through current Soda Identity/setup/host callers.
Start at `cmd/soda-identity/tests/http.rs`, `cmd/soda-setup/src/tests/postgres.rs`,
`lib/host/src/project/executor.rs` and the existing host websocket tests. Select
only unproved shipping/provider joins; package tests do not establish installed
behavior. R48 owns funded provider responses, and R52 binds unchanged native
proof to actual shipped identities where required.

**Accept only when:** Required setup is approved and inspectable; wrong peer/account/repository/grant and rotated credentials refuse. Background authority survives browser logout, native/operator identities remain separate, and existing human services remain usable. The new factory intervention/retirement interaction is accepted under R41 and the complete installed journey under R49; it is not a prerequisite for establishing this foundation. Preserve retained identity/tooling work and its current input bindings.

**Executable remainder:** Consume R03's native background authority and R10's scope binding. Start in existing `internal/project`, `lib/host/src/project/admission.rs` and `cmd/soda-identity` preparation/broker handlers; trace one approved preparation and account lease through launch/return. Qualify wrong peer/repository/grant, rotation/logout and human SSH/operator coexistence using retained source checks plus native cases. Release current preparation and credential custody to R11/R12. V03 normal managed Muse admission requires human device-account approval; TestOnly evidence does not close it. That input holds the normal-account case, not every local Project check, and R41 remains the later factory intervention composition.

**Principal source owners:** [internal/project](../../../internal/project), [lib/host/src/project](../../../lib/host/src/project), [cmd/soda-identity](../../../cmd/soda-identity), [docs/guides/develop.md](../../../docs/guides/develop.md), [docs/reference/credentials.md](../../../docs/reference/credentials.md).

<a id="r45"></a>
## R45 — Qualify current release-component composition and maintenance

**Work:** Qualify and repair demonstrated gaps. **Owner lane:** C. **Accepted inputs:** [R03](01-boundaries.md#r03), [R06](01-boundaries.md#r06), [R44](05-experience-and-development.md#r44).

**Current (wind-down):** Current component/package/controller/maintenance qualification; scoped RCX development acceptance retained. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind Payload v4, seven separate required image roles, including canlang-runtime, thin host/outer Docker, host-only compression/official-image compatibility, Fountain-owned lifecycle/durable intent and Soda package/controller declarations. Verify protected locks, startup ordering, closed admission, offline preparation and enabled-only maintenance.

**Shared Filetree/library consumers:** Carry C/D/H and L10/L17 native residuals
through the real component/installed owner: shipped curl/Bash/Podman and image/RPM
identity, resolver-inclusive bounded transport/TLS failure, and generated unit/
configuration identity. Reuse accepted source profiles and Butane/OCI/parser
fixtures. The native filesystem confinement cases start at
`lib/soda-release-deliver/src/buildx/filesystem.rs`,
`cmd/soda-forgejo-domain/src/marker.rs` and `lib/host/src/tailnet/files/mod.rs`;
actual supported-kernel openat2 behavior, symlink/escape refusal and preserved
prior data need native observations, not another parser conversion. R46 consumes
network-specific results; R52 supplies final shipping-byte witnesses. These are
aliases of existing gates, not extra prerequisite projects.

**Accept only when:** Current component/package/source identities and authenticated start/stop/snapshot/rollback readback match. Failure preserves prior valid bytes/data and stays closed when retirement is uncertain. Reuse accepted RCX source; no obsolete-plan restoration or replacement component architecture.

**Executable remainder:** Consume R03 package authority, R06 importer retirement and R44 preparation. Start at `lib/release-inputs/src/payload.rs::Role`, release maintenance and existing component drivers. Bind Payload v4's seven required image roles including canlang-runtime; qualify current package/controller enablement, offline preparation and same-input maintenance/rollback. Release exact component/lifecycle observations to R46/R51; direct-workerd replacement adds its own P27/P28 producer outputs before shipping. Existing RCX source/development evidence remains accepted. Missing/invalid bytes must preserve prior data and keep admission closed; test changed native joins rather than rebuilding the component architecture.

**Principal source owners:** [lib/release-inputs/src/payload.rs](../../../lib/release-inputs/src/payload.rs), [cmd/soda-project-maintain/src/release_extensions.rs](../../../cmd/soda-project-maintain/src/release_extensions.rs), [tools/release-assets](../../../tools/release-assets).

<a id="r46"></a>
## R46 — Qualify optional Tailnet and retained networking behavior

**Work:** Qualify; named external cases. **Owner lane:** C. **Accepted inputs:** [R45](05-experience-and-development.md#r45).

**Current (wind-down):** Current optional Tailnet/network qualification; actual external enrollment/routes remain separate. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Verify default absence, administrator-enabled optional content, required Soda controller, full stop when disabled and current LAN/private-origin/HTTPS behavior. Exercise actual enrollment/routes only with the specifically authorized provider input.

**Accept only when:** Native authority/failure/restart/disable cases pass on current bytes; lack of enrollment cannot be relabelled successful external access. Optional Tailnet/domain purchase is not required for core factory use. Missing provider input holds only the named external cases and remains explicit at final scope accounting.

**Executable remainder:** Consume R45's component identity/lifecycle. First qualify default absence and administrator enable/disable/restart through `lib/host/src/tailnet/{admission,lifecycle}.rs` and existing LAN/private-origin handlers. Then, only with the selected human-supplied client/enrollment/route/trust input, exercise actual external connectivity and revoked/disabled access. Release the precise local and external results separately to R55/R52. Existing native drivers cover authority/failure cases; inability to enroll is an explicit held external case, not success or a requirement to purchase Tailnet for core use. Share the same private-client fixture with Canlang P22 where applicable.

**Principal source owners:** [lib/host/src/tailnet](../../../lib/host/src/tailnet), [docs/guides/operator-setup.md](../../../docs/guides/operator-setup.md), [docs/architecture/networking.md](../../../docs/architecture/networking.md).

<a id="r47"></a>
## R47 — Run current composed PostgreSQL and serving-path race acceptance

**Work:** Integration qualification. **Owner lane:** Coordinator. **Accepted inputs:** [R11](02-prompt-to-pr.md#r11), [R13](02-prompt-to-pr.md#r13), [R17](02-prompt-to-pr.md#r17), [R20](03-review-merge-reassessment.md#r20), [R23](03-review-merge-reassessment.md#r23), [R26](03-review-merge-reassessment.md#r26), [R29](03-review-merge-reassessment.md#r29), [R32](04-controls-repair-verification.md#r32), [R33](04-controls-repair-verification.md#r33), [R34](04-controls-repair-verification.md#r34), [R35](04-controls-repair-verification.md#r35), [R37](04-controls-repair-verification.md#r37), [R39](04-controls-repair-verification.md#r39), [R40](04-controls-repair-verification.md#r40), [R41](04-controls-repair-verification.md#r41), [R42](05-experience-and-development.md#r42).

**Current (wind-down):** Current composed API/Start/MCP/PostgreSQL serving and race acceptance. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Compose real API/Start/MCP producers through shared Store transactions, native/host adapters and actual next-stage consumers. Exercise clean and correction routes, all D13 branches, stall repair, Test Pilot and intervention/crash prefixes.

**Accept only when:** No pre-injected accepted graph/merge authorization or manual phase invocation substitutes for serving progression. Relevant race/rollback/lost-reply cases pass on actual PostgreSQL with no hidden skips; compare real baseline failures, not summary claims. This adds cross-task composition to checks already required within each task.

**Executable remainder:** First triage recorded Deep CI 39139499605 against its actual failed jobs and current selectors: release-build source/CLI assertions, Tailnet navigation, PG topology and fixture/toolchain cases. Start at `.github/workflows/deep-ci.yml`, `tests/build/source_checks_test.go`, `tests/frontend/{tailnet,workspace-navigation}.test.ts`, `scripts/pg_runtime_test.go` and the failed package's existing PG fixture. Prepare only demonstrated repairs and focused reproductions; accepted wait-slot/schema fixes stay closed. This can start independently of the later composed proof. As upstream slices land, exercise API/Start/MCP → real PG → adapter → next consumer in existing tests, ending with clean/correction, D13, stall/Test Pilot and intervention routes. Reuse constituent evidence and run only missing cross-task/race cases with no hidden PG skips. The listed parent inputs gate final composition, not initial failure triage; required CI green does not erase Deep failures.

**Principal source owners:** [internal/store](../../../internal/store), [internal/factory/control](../../../internal/factory/control), [internal/web/api](../../../internal/web/api).

<a id="r48"></a>
## R48 — Qualify actual harness transport and first-response semantic behavior

**Work:** Native/provider qualification. **Owner lane:** A. **Accepted inputs:** [R12](02-prompt-to-pr.md#r12), [R13](02-prompt-to-pr.md#r13), [R16](02-prompt-to-pr.md#r16), [R22](03-review-merge-reassessment.md#r22), [R24](03-review-merge-reassessment.md#r24), [R29](03-review-merge-reassessment.md#r29), [R32](04-controls-repair-verification.md#r32), [R33](04-controls-repair-verification.md#r33), [R34](04-controls-repair-verification.md#r34), [R35](04-controls-repair-verification.md#r35), [R40](04-controls-repair-verification.md#r40), [R41](04-controls-repair-verification.md#r41).

**Current (wind-down):** Exact harness/provider transport and first-response semantic trials. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use retained drivers and bounded authorized accounts/fixtures to exercise actual Codex and Muse transport/role/native behavior and first-response semantic cases for decomposition, delivery decisions, reassessment and repair.

**Accept only when:** Exact CLI/model/account/reasoning/Contributor/config and prompt/result/usage bindings are retained. Broad roots/joins, tiny work, ambiguity, clean zero-call and correction/uncertainty cases meet contracts. A missing input names the precise case; mock or duplicated SDK-binary runs cannot qualify a real provider/harness.

**Executable remainder:** Consume the role consumers listed above, exact R07–R09 harnesses and R44 account custody. Extend retained transport/semantic drivers: group trials by actual harness/configuration and role, covering first-response decomposition, delivery decision, reassessment and repair, with tiny/broad/join/ambiguity cases. Clean clear remains zero-call. Release prompt/result/usage and native effect bindings to R49, within a finite human-approved provider budget. Normal Muse enrollment is V03's external input; no mock account or repeated SDK binary substitutes. Small-model manageability is desirable, not an acceptance gate; use stronger executor judgment for hard coherent work.

**Principal source owners:** [lib/host/src/terminal/factory](../../../lib/host/src/terminal/factory), [internal/factory/control](../../../internal/factory/control), [docs/development/testing.md](../../../docs/development/testing.md).

<a id="r49"></a>
## R49 — Demonstrate the complete development factory with real Work/Spaces

**Work:** Installed development qualification. **Owner lane:** Coordinator. **Accepted inputs:** [R39](04-controls-repair-verification.md#r39), [R40](04-controls-repair-verification.md#r40), [R43](05-experience-and-development.md#r43), [R44](05-experience-and-development.md#r44), [R45](05-experience-and-development.md#r45), [R47](05-experience-and-development.md#r47), [R48](05-experience-and-development.md#r48).

**Current (wind-down):** Complete real development factory plus persisted role feed question/reply/finding/result journey. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Ralph absorption addition:** Include the role-authored feed in this same
installed journey: follow actual issue/task/PR/review/correction/reassessment and
application-verification records, ask and answer a consequential question through
its authorized route, inspect progress and a finding, and distinguish a role's
done/result submission from Factory confirmation. Observe the agreed component
reuse and real persisted updates; illustrative movie data cannot stand in for
this acceptance. Retain the existing R49 inputs and installed proof boundaries.

**Deliver:** On one owned supported development assembly, run real prompt→semantic graph→selection→investigation/plan→native overlap/checkpoints→PR→review/correction→CI→merge/original closure→dependent→reassessment→Test Pilot, with actual Work/Spaces and human intervention.

**Accept only when:** Both clean and correction/Jev routes plus pre-completion stall recovery and restart are observed with exact identities/bytes and bounded usage. Retained human services/work survive. Development success is explicitly distinct from shipping-byte qualification.

**Executable remainder:** Consume R39/R40 completion, R43 combined feed, R44/R45 native foundations and R47/R48 composed/provider proof. Use one owned development assembly and the existing testing.md demonstration: prompt through graph/plan/overlap/checkpoint/PR/review/merge/dependant/reassessment/Test Pilot, then correction, stall and restart branches. Include two authorized subscribed projects in the one newest-first feed, a persisted user comment consumed into a linked plan/issue, question/reply, progress/finding and submitted-versus-verified outcome. Confirm denied cross-project visibility and reading/draft continuity through real Work/Spaces. Reuse earlier primitive proof, observe the actual joins and bounded usage; no fixture-injected accepted phases or shipping qualification claim.

**Principal source owners:** [docs/development/testing.md](../../../docs/development/testing.md), [internal/factory/control](../../../internal/factory/control), [frontend/spaces](../../../frontend/spaces).

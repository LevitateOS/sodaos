# Prompt to checked PR

[Executable task map](../factory-remaining-tasks.md#task-list) owns the current
deliverables and output dependencies. This packet preserves the parent contracts,
technical source owners, accepted evidence and additional native/installed gates.
The Rxx.y links below identify the execution units; parent input lists describe
full qualification and do not require waiting for whole parents before using
applicable outputs. Recorded baseline and landed slices retain their actual
scope. Implementation notes support the execution map rather than define a
second schedule. This documentation restructuring starts no runtime work.

<a id="r11"></a>
## R11 — Finish one atomic subject-aware dispatch and accounting path

**Execution tasks:** [R11.1](../factory-remaining-tasks.md#r11-1), [R11.2](../factory-remaining-tasks.md#r11-2).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01), [R10](01-boundaries.md#r10).

**Recorded baseline:** Live/native dispatch and atomic settlement qualification; source/common-boundary acceptance retained. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Extend the existing shared registration/assignment/reservation/run/view/result core for intent and completion subjects. Factor transaction-bound reservation internals; inspect actual lock ordering, authority, preparation, current head, capacity and withdrawal. Persist intent before any host call.

**Collection-maintenance admission:** Consume R10's Library Admin subject in this
same dispatch/accounting path. Admit an explicit maintenance request or configured
bounded trigger with finite queue/collection/operation scope and funding; search,
feed reads and item edits do not implicitly start another bot. Do not invent an
issue or refill a production attempt's allowance. Concurrent/replayed triggers
must identify one admitted operation/run or no run; R32 maintenance finish and
R35 incomplete settlement preserve committed item changes and actual charges.

**Accept only when:** Concurrent admission/replay produces one run/charge or no rows. Failed/interrupted provider work stays charged; retries cannot refill consumption. Provider calls happen outside short database transactions. Settlement and release are atomic and replay-safe; no nested self-committing reservation or separate intent engine.

**Dependency boundary:** R10 releases the exact typed contract before transaction changes. Real PostgreSQL admission/rollback work can proceed before native activation; R44's current Project/provider custody is required before a live launch.

**Implementation notes:** Start at `factory_dispatch_packet.go`, `dispatch_launch.go::executeFreshAttempt` and `dispatch_result.go::AccountSettledRun`. Against R10's defined envelopes, trace registration/reservation/run/result through the common transaction boundary; add only uncovered real-PG racing/replayed intent/completion cases. Release a once-charged persisted launch and replay-safe settlement to R12/R15/R35. Live proof waits for R44 preparation/custody and R07–R09/R30 activation. Failed start, unknown result and withdrawal retain charge/custody until observed retirement. Accepted common dispatch code is not a second engine to implement.

**Source-trace assessment (no code change):** admission is one transaction (`RecordDispatchPacket`: gate registration, queued-control/acceptance-head locks, attempt/authority/custody checks, row writes, attempt funding, limit recheck, single commit) committed before any host call (`executeFreshAttempt` records the packet, then `launchTail` observes within a 5s bound) — provider work runs outside the transaction by construction. Settlement (`AccountSettledRun`) is sequential, not one transaction, but each step is replay-guarded: usage first-write-wins, consume tolerates already-consumed, finish is `FOR UPDATE` freshness-checked with the concurrent loser refused, and double-settle is proven to settle nothing twice (`dispatch_accounting_test.go:652`). Same-issue races refuse `ErrAssignmentActive` across four test files; stale control, withdrawn acceptance, limits and one-session admission are proven atomic at the packet. Uncovered real-PG cases for the host run: (1) concurrent double `FinishAssignment` — exactly one terminal write wins with no duplicate allowance transition; (2) partial-settlement replay (consume-then-crash) completing without double charge. Both need the fixture; no source change precedes them.

**Host-PG race/replay proof (landed):** `dispatch_settlement_race_test.go` closes both cases against a disposable fixture — `TestSettlementConcurrentDoubleFinish` races eight concurrent settlements of one failed run (one finished write, one usage row, reservation consumed once, post-race replay silent with stable total) and `TestSettlementConsumeThenCrashReplay` replays usage-recorded plus reservation-consumed state through `AccountSettledRun` to the same single-charged finish. Sandbox: `go vet` clean, both tests PASS with `-race` over three runs. Live launch proof still waits for R44 preparation/custody and R07–R09/R30 activation.

**Principal source owners:** [internal/store/factory_dispatch_packet.go](../../../internal/store/factory_dispatch_packet.go), [internal/store/factory_grants.go](../../../internal/store/factory_grants.go), [internal/store/factory_intents.go](../../../internal/store/factory_intents.go), [internal/factory/control/dispatch_launch.go](../../../internal/factory/control/dispatch_launch.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R44](05-experience-and-development.md#r44), [R30](04-controls-repair-verification.md#r30). Source-only work cannot release these gates.

<a id="r12"></a>
## R12 — Connect the real semantic Product Lead planner

**Execution tasks:** [R12.1](../factory-remaining-tasks.md#r12-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R10](01-boundaries.md#r10), [R11](02-prompt-to-pr.md#r11).

**Recorded baseline:** Real semantic provider qualification; server inputs, plan/Generate UI through cf5887a0 accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect authenticated Generate to current reviewed draft, repository association, prepared Project, inspected source and bounded generative planning through the common dispatch path. Persist typed attention/failure/candidate results bound to exact inputs.

**Accept only when:** A repository-bound reviewed draft can plan without a fake issue or pre-accepted task graph; an unassociated draft launches nothing. Wrong owner/source/phase/grant refuses before effects. Actual broad roots/joins, tiny cohesive work and consequential ambiguity are exercised; no template substitution.

**Dependency boundary:** Consume R44's approved preparation and provider authority for live planning. R12 implements the shared planner/result consumer; R32 later composes its MCP finish controls and R48 qualifies actual provider first responses. Those later composition tasks cannot be counted as already complete here.

**Discussion input:** R32/R31 can supply an authorized selected post/comment
revision to the existing draft/planning input. Preserve its origin link and
request identity in the resulting plan or attention/failure result. The original
Generate path need not wait for comments; this added join does. Use existing
scope/grant/decision rules and once-charged dispatch, without automatic execution
of every comment or an extra mandatory comment approval.

**Implementation notes:** Consume R11's admitted intent-subject launch and R44's preparation/provider authority. Start at `StageIntentBoundLaunch` in `intent_bound.go` and `PlanIntentDecomposition` in `intent_planning.go`; Generate and draft/source staging already exist. Join actual semantic output to those owners: one bound draft, then tiny work, multiple roots/join and consequential ambiguity. Release a source/scope-bound candidate set or product attention to R13/R32. Focused planner/Generate checks cover changed joins; semantic acceptance needs a finite funded native trial. Wrong owner, unassociated draft or changed preparation launches nothing.

**Preserved pre-repository entry:** [Intent-first creation](../../product/spaces.md#intent-first-creation) precedes repository selection. The existing `apiIntentExplore` in `internal/web/api/factory_intent_explore.go` and saved-review tests already support owner-private bounded exploration; `frontend/spaces/sodaspaces-intent{,-walkthrough}.ts` presents it. “Unassociated draft launches nothing” here forbids repository-bound Product Lead/execution dispatch, not the accepted pre-association exploration. Next join the stored effective proposal/review revision to repository association and the existing R11/R44-admitted Generate path. Acceptance: corrections keep actual human/autonomous provenance; consequential unresolved assumptions block only dependent work; navigation/department choice never provisions or grants access; stale/cross-owner review refuses and GET/reload spends nothing. Preserve accepted source proofs rather than rebuilding this entry flow; R43/R49 consume the real UI journey.

**Principal source owners:** [internal/factory/control/intent_bound.go](../../../internal/factory/control/intent_bound.go), [internal/factory/control/intent_planning.go](../../../internal/factory/control/intent_planning.go), [internal/web/api/factory_intent_generation.go](../../../internal/web/api/factory_intent_generation.go), [internal/factory/decomposition_options.go](../../../internal/factory/decomposition_options.go).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R44](05-experience-and-development.md#r44), [R30](04-controls-repair-verification.md#r30). Source-only work cannot release these gates.

<a id="r13"></a>
## R13 — Adopt one chosen semantic graph with exact scope and provenance

**Execution tasks:** [R13.2](../factory-remaining-tasks.md#r13-2), [R13.1](../factory-remaining-tasks.md#r13-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R12](02-prompt-to-pr.md#r12).

**Recorded baseline:** Native chosen-graph adoption qualification; 3e4de021 source repair accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Wire single-candidate direct choice and complete-set funded multi-candidate choice to current accepted outcome authority and exact native graph adoption. Preserve existing task/delivered bindings and explicit human product decisions.

**Initial Git-issue ownership:** Product Lead supplies initial decomposition,
issue descriptions/acceptance and dependencies through R32's revisioned product
draft/finish tools. This existing Factory/native graph owner applies accepted
changes and returns actual issue IDs/effects. Reference the actual choice/authority
in R32 decision records and R42 feed/library projections; a decision log or draft
does not itself create an issue. Strategist's ongoing maintenance is R29/R37;
Library Admin never receives a generic Git issue-edit capability.

**Discussion-derived issues:** Consume R12/R32's origin-bound authorized request
through this same issue/graph owner. Persist links from created issues to the
source discussion; exact retry returns the existing operation/issues. Stale
comment revision or insufficient issue/scope authority yields a visible refusal
or attention result. Test lost reply and concurrent duplicate requests without
creating another issue or bypassing native conditional mutation.

**Accept only when:** One candidate causes no comparative call; multiple candidates use one bounded replayable choice, never first-pick or a second template rewrite. Scope, source, preparation and head changes invalidate adoption. Real issue/edge receipts are adopted once and exact graph readback matches the chosen tasks.

**Implementation notes:** Consume R12's complete candidate set, R04's graph operation and R05's readback. Start at `ChooseSingleCandidate`/`ChoosePlannedCandidate` and `GenerateAcceptedIntentFromChoice`. Qualify the repaired path with one candidate (zero comparison calls), then funded multiple choice and a lost native response. Release exact chosen issue/edge bindings. Existing source tests remain evidence; native witnesses show stale source/scope/preparation refusal and retry adoption of the original operation once. Do not create a second translator or rewrite the chosen graph.

**Principal source owners:** [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/factory/control/intent_planning.go](../../../internal/factory/control/intent_planning.go), [internal/store/factory_intent_generation.go](../../../internal/store/factory_intent_generation.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04), [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r14"></a>
## R14 — Finish current large-inventory selection and readiness qualification

**Execution tasks:** [R14.2](../factory-remaining-tasks.md#r14-2), [R14.1](../factory-remaining-tasks.md#r14-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Native current-inventory readiness qualification; cursor/restart/large-queue source proofs accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Audit current serving calls and close only remaining cursor, prerequisite truth, freshness and selection seams. Use actual per-pass bounds and authoritative eligibility; preserve independent component progress.

**Accept only when:** 256 candidates and later-page prerequisites make bounded progress through restart/concurrent insertion. One eligible item needs no comparative call. A join needs all required verified outcomes, and closed/reopened/unaccepted work cannot fabricate readiness.

**Implementation notes:** Consume R05's complete revision-bound snapshot. Start at `priority_dispatch.go`, `readiness.go::assessOne` and `factory_priority_inventory.go`. Run the retained 256-candidate/later-page fixture through actual snapshot/serving calls, interrupt between rounds, insert a candidate and reopen a prerequisite. Release fresh selected eligibility and prerequisite satisfaction to R15/R36. Repair only a failed native cursor/freshness join; bounded scan/restart source behavior is accepted. One eligible item spends zero comparison calls; partial inventory cannot release a join.

**Principal source owners:** [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go), [internal/factory/control/readiness.go](../../../internal/factory/control/readiness.go), [internal/store/factory_priority_inventory.go](../../../internal/store/factory_priority_inventory.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r15"></a>
## R15 — Connect selected investigation to immutable executable plans

**Execution tasks:** [R15.1](../factory-remaining-tasks.md#r15-1), [R15.2](../factory-remaining-tasks.md#r15-2).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R11](02-prompt-to-pr.md#r11), [R14](02-prompt-to-pr.md#r14).

**Recorded baseline:** Compose selected investigation and exact plan/head admission; journey and immutable old-revision Store proofs already pass. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Investigate only the freshly selected issue using current source/prerequisite evidence. Persist a concrete required-check plan and ordered task DAG with coverage, output dependencies, nonconflicting ownership and parallel lanes before coding.

**Work-in-progress integration:** Consume R32's Architect investigation and plan
draft records at exact revisions; expose inspected interfaces, evidence,
alternatives, ownership, dependencies and checks before finish. Bind recorded
technical decisions and knowledge references without granting authority from
their prose. Drafts remain inert; accepted executable plans are immutable and
material changes use the existing replacement/admission route. Release these
same records to R42 approach/investigation/lane components.

**Discussion input:** Relevant authorized comments may inform this selected
issue's plan through R31/R32 context. Bind the consumed discussion revision and
origin reference to the immutable plan; later edits do not silently mutate it.
Comments do not select an unrelated issue, confer write authority or invalidate
otherwise accepted old-plan evidence merely by existing.

**Accept only when:** No unselected issue receives a generative run; replay cannot double spend. Stale source/preparation/prerequisites cannot launch implementation. Old plan revisions survive with exact head/assignment bindings.

**Implementation notes:** Compose the accepted R11 dispatch and R14 selection interfaces with `planning.go` and `factory_planning_revisions.go::{SaveInvestigationPlan,AdmitPlanHead}`; this source slice need not await their native qualifications. Use one selected issue with independent outputs and a dependent join, explicit paths/checks and current source/preparation bindings. Release an immutable admitted plan/head and manifest to R16. Extend Store/journey proof at selection → investigation → implementation admission, including stale head/preparation and replay. Actual provider execution still consumes R07–R09/R44. Do not investigate the whole queue or replace old revisions.

**Source-complete status:** the composition is wired — `dispatch_attempt.go` persists the reported plan, advances the head and admits against live source/prerequisite/preparation bindings before any implementation child launches; stale, provisional, rebound or forged plans wait instead of launching, and the admitted plan stages into the implementation prompt within its phase bound. Store proofs cover immutable revisions, conflict survival, CAS head advance, atomic replace and binding-checked admission; journey proofs cover admit-once, advance-once, unresolved-cannot-launch, changed-source-withholds and covered-skip. Sandbox: pure binding test PASS, PG-gated admission tests skip by design. Remainder is host PG proof of the admission suite plus R07–R09/R44 provider execution.

**Host-PG proof (landed):** the full admission suite passes against a disposable fixture — all seven `planning_admission_test.go` journey cases (`TestSelectedInvestigationAdmitsOnce`, `TestCompletedInvestigationAdvancesOnce`, `TestUnresolvedPlanCannotLaunch`, `TestChangedSourceWithholdsAdvance`, `TestPlanningSkipsCoveredIssue`, `TestAdmitInvestigationPlanBindings`, `TestLivePlanBindingsCanonical`) plus the five `factory_planning_revisions_test.go` Store proofs (`TestSaveInvestigationPlanRoundTripAndRefusals`, `TestPlanRevisionImmutableSurvivesConflict`, `TestAdvancePlanHeadCAS`, `TestReplacePlanHeadAtomic`, `TestAdmitPlanHeadBindings`). No source change; the wired composition holds on real PG. Remainder is R07–R09/R44 provider execution only.

**Principal source owners:** [internal/factory/control/planning.go](../../../internal/factory/control/planning.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go), [internal/factory/control/dispatch_launch.go](../../../internal/factory/control/dispatch_launch.go).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R44](05-experience-and-development.md#r44). Source-only work cannot release these gates.

<a id="r16"></a>
## R16 — Connect real native task scheduling and dependency release

**Execution tasks:** [R16.1](../factory-remaining-tasks.md#r16-1).

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R15](02-prompt-to-pr.md#r15).

**Recorded baseline:** Connect native task events to Store and accepted dependency release. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Drive approved plan tasks through genuine native children under one admitted parent, with inherited identity and limits. Observe results, cancellation and retirement; release successors promptly from trusted accepted outputs.

**Accept only when:** Actual independent tasks overlap and a dependent starts while unrelated work runs. Lost start replies are looked up, not replayed. Failed/unknown outputs never release descendants or grant another lease; parent death requires explicit recovery.

**Implementation notes:** The progress loop now imports `FactoryTaskEvents` against the exact retained launch manifest and observed native invocation, folding each page through `ImportTaskEvents` and saving its receipt advances atomically. Persisted event references retain canonical digests: changed replay bytes and cross-page chain rewrites refuse without advancing the Store cursor. `TestProgressImportsTaskJournalPG` exercises real dispatch/Store admission, page replay, terminal cursor retention and replacement-plan refusal against disposable PostgreSQL; the native event source is scripted. The remaining producer must use supported native children inside the admitted parent, with actual overlap, prompt dependency release and observed whole-run retirement. The independent headless `TaskRunner` process pool does not satisfy D05 and must not be wired as that producer. Release trusted events to R17; unknown start/output cannot trigger another child or successor.

**Principal source owners:** [lib/host/src/factory/tasks.rs](../../../lib/host/src/factory/tasks.rs), [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [lib/host/src/terminal/factory/children.rs](../../../lib/host/src/terminal/factory/children.rs), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r17"></a>
## R17 — Produce trusted task receipts and early Git checkpoints

**Execution tasks:** [R17.1](../factory-remaining-tasks.md#r17-1).

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R16](02-prompt-to-pr.md#r16).

**Recorded baseline:** Production trusted task-receipt/check/Git-CAS/Store join. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect host-observed task output to protected exact-tree checks, serialized Git CAS checkpoint receipts and durable Store admission. Preserve worker files and unrelated dirty/index state through crash/lost reply.

**Accept only when:** Real checkpoint production calls the Store consumer. Dirty sentinel, wrong ancestry, mutated checks, forged journal and wrong invocation refuse. Crash before/after CAS recovers only the prepared commit; dependent tasks consume a trusted receipt, never a synthetic acknowledgment.

**Dependency boundary:** Establish the trusted host-observation/check/CAS/Store path here, using R30's authority for native qualification. R33 connects agent proposals to that existing owner; R17 must not require an R33 completion receipt to implement its own prerequisite, or duplicate the checkpoint engine.

**Implementation notes:** The trusted owner is now `lib/host/src/factory/checkpoint/`, joined to the Go progress loop by `control/task_checkpoint.go`. It snapshots only released literal ownership, runs fixed admitted checks in check-only exports through the existing Project systemd manager, fsyncs preparation before serialized Git CAS, and admits only the resulting private native journal. An authenticated matching `submit_task_output` proposal is required before execution; observation alone grants no checkpoint. A retained-VM fixture exercised actual Git/systemd task commits, independent output observed before another task integrates, verification without an empty commit, required final checks, exact replay, lost reply after an actual successful CAS, and retirement of a running check process group before parent credential return. That fixture uses a non-model parent and fake broker; it does **not** close native child or provider-custody qualification. Real PostgreSQL dispatch/proposal/receipt recovery also passes. Wrong ancestry, post-check mutation and genuine native-child composition still need the complete production journey. The independent headless `TaskRunner` remains unsuitable for this producer.

**Principal source owners:** [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [lib/host/src/factory/candidate.rs](../../../lib/host/src/factory/candidate.rs), [lib/host/src/factory/manifest.rs](../../../lib/host/src/factory/manifest.rs), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

**Additional runtime/qualification gates:** [R30](04-controls-repair-verification.md#r30). Source-only work cannot release these gates.

<a id="r18"></a>
## R18 — Gate final integration and publication on the complete checked plan

**Execution tasks:** [R18.1](../factory-remaining-tasks.md#r18-1).

**Work:** Implement and integrate. **Owner lane:** A. **Accepted inputs:** [R17](02-prompt-to-pr.md#r17).

**Recorded baseline:** Complete-plan checked-tree publication gate. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Join all required task outputs, checkpoint ancestry and declared final exact-tree checks before exporting one publishable candidate. Support verification-only/no-change work without fake commits or PRs.

**Accept only when:** Incomplete tasks, stale source, wrong ancestry or unchecked trees cannot publish. Exact final artifact/tree bindings survive lost replies and cancellation; final host evidence is independently attributable.

**Implementation notes:** Manifest-backed publication and first correction export now require the committed native final-candidate receipt for the exact run/container/invocation/manifest/candidate and every required integration check. Native finalization verifies task checkpoint ancestry and a clean tracked/index tree, refusing unexplained untracked files outside the reserved run home. The Go PostgreSQL journey proves completed task evidence alone still waits, native final evidence releases publication, and an ordinary run observation cannot erase the retained receipt or rerun completed integration checks. The retained-VM engine fixture covers real final checks and verification-only work; the genuine provider/native-child → finalization → PR journey remains open. No-change work creates no empty commit or PR.

**Principal source owners:** [lib/host/src/factory/candidate.rs](../../../lib/host/src/factory/candidate.rs), [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go).

<a id="r19"></a>
## R19 — Complete one attributable PR and publication recovery journey

**Execution tasks:** [R19.1](../factory-remaining-tasks.md#r19-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R18](02-prompt-to-pr.md#r18), [R20](03-review-merge-reassessment.md#r20).

**Recorded baseline:** Current attributable PR and publication recovery composition. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Compose current checkpoints with separate conditional ref and PR operations, exact receipt adoption, correction reuse and native completion. Preserve committed effects when later work fails.

**Accept only when:** One issue/candidate creates one attributable PR under lost replies, retry and restart. Matching remote names are not adopted as attribution; unexpected refs, wrong head/base and stale authority refuse.

**Implementation notes:** Consume R18's checkpoint and R20's fresh tuple. Start at `PublishPass`, `publication_reconcile.go` and `factory_publications.go`; use R04 receipts for ref publication and PR creation separately. Prove a normal candidate, lost reply after each effect, restart and correction reuse of the same PR. Release attributable current PR/operation binding to R21/R22/R25. Focused PG reconciliation plus native ref/PR readback rejects an unexpected matching branch or mismatched tuple without deleting human refs.

**Principal source owners:** [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/factory/control/publication_reconcile.go](../../../internal/factory/control/publication_reconcile.go), [internal/store/factory_publications.go](../../../internal/store/factory_publications.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04). Source-only work cannot release these gates.

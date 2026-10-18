# Prompt to checked PR

[Remaining-work catalog](../factory-remaining-tasks.md#task-list) owns execution order. These packets preserve the original requirement authorities; the existing central checklist alone owns live progress and writer leases. Exact source file subsets are frozen before allocation, so directories below are responsibility locators rather than blanket write leases.

<a id="r11"></a>
## R11 — Finish one atomic subject-aware dispatch and accounting path

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01), [R10](01-boundaries.md#r10).

**Current:** Issue dispatch and finite reservations exist. Intent dispatch is not serving; the 198 tx-bound reservation extraction is an unrun prerequisite candidate.

**Deliver:** Extend the existing shared registration/assignment/reservation/run/view/result core for intent and completion subjects. Factor transaction-bound reservation internals; inspect actual lock ordering, authority, preparation, current head, capacity and withdrawal. Persist intent before any host call.

**Accept only when:** Concurrent admission/replay produces one run/charge or no rows. Failed/interrupted provider work stays charged; retries cannot refill consumption. Provider calls happen outside short database transactions. Settlement and release are atomic and replay-safe; no nested self-committing reservation or separate intent engine.

**Dependency boundary:** R10 releases the exact typed contract before transaction changes. Real PostgreSQL admission/rollback work can proceed before native activation; R44's current Project/provider custody is required before a live launch.

**Principal source owners:** [internal/store/factory_dispatch_packet.go](../../../internal/store/factory_dispatch_packet.go), [internal/store/factory_grants.go](../../../internal/store/factory_grants.go), [internal/store/factory_intents.go](../../../internal/store/factory_intents.go), [internal/factory/control/dispatch_launch.go](../../../internal/factory/control/dispatch_launch.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R44](05-experience-and-development.md#r44), [R30](04-controls-repair-verification.md#r30). Source-only work cannot release these gates.

<a id="r12"></a>
## R12 — Connect the real semantic Product Lead planner

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R10](01-boundaries.md#r10), [R11](02-prompt-to-pr.md#r11).

**Current:** Planner/admission helpers exist, but Generate follows the accepted authored/template route and no real semantic producer serves it.

**Deliver:** Connect authenticated Generate to current reviewed draft, repository association, prepared Project, inspected source and bounded generative planning through the common dispatch path. Persist typed attention/failure/candidate results bound to exact inputs.

**Accept only when:** A repository-bound reviewed draft can plan without a fake issue or pre-accepted task graph; an unassociated draft launches nothing. Wrong owner/source/phase/grant refuses before effects. Actual broad roots/joins, tiny cohesive work and consequential ambiguity are exercised; no template substitution.

**Dependency boundary:** Consume R44's approved preparation and provider authority for live planning. R12 implements the shared planner/result consumer; R32 later composes its MCP finish controls and R48 qualifies actual provider first responses. Those later composition tasks cannot be counted as already complete here.

**Principal source owners:** [internal/factory/control/intent_bound.go](../../../internal/factory/control/intent_bound.go), [internal/factory/control/intent_planning.go](../../../internal/factory/control/intent_planning.go), [internal/web/api/factory_intent_generation.go](../../../internal/web/api/factory_intent_generation.go), [internal/factory/decomposition_options.go](../../../internal/factory/decomposition_options.go).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R44](05-experience-and-development.md#r44), [R30](04-controls-repair-verification.md#r30). Source-only work cannot release these gates.

<a id="r13"></a>
## R13 — Adopt one chosen semantic graph with exact scope and provenance

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R12](02-prompt-to-pr.md#r12).

**Current:** Chosen-plan and native graph adoption helpers exist; fixtures manually calling them do not prove the serving producer.

**Deliver:** Wire single-candidate direct choice and complete-set funded multi-candidate choice to current accepted outcome authority and exact native graph adoption. Preserve existing task/delivered bindings and explicit human product decisions.

**Accept only when:** One candidate causes no comparative call; multiple candidates use one bounded replayable choice, never first-pick or a second template rewrite. Scope, source, preparation and head changes invalidate adoption. Real issue/edge receipts are adopted once and exact graph readback matches the chosen tasks.

**Principal source owners:** [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/factory/control/intent_planning.go](../../../internal/factory/control/intent_planning.go), [internal/store/factory_intent_generation.go](../../../internal/store/factory_intent_generation.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04), [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r14"></a>
## R14 — Finish current large-inventory selection and readiness qualification

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Current:** Durable complete-set inventory, cursor, fairness and fresh selected admission are implemented; do not rebuild them from an old gap description.

**Deliver:** Audit current serving calls and close only remaining cursor, prerequisite truth, freshness and selection seams. Use actual per-pass bounds and authoritative eligibility; preserve independent component progress.

**Accept only when:** 256 candidates and later-page prerequisites make bounded progress through restart/concurrent insertion. One eligible item needs no comparative call. A join needs all required verified outcomes, and closed/reopened/unaccepted work cannot fabricate readiness.

**Principal source owners:** [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go), [internal/factory/control/readiness.go](../../../internal/factory/control/readiness.go), [internal/store/factory_priority_inventory.go](../../../internal/store/factory_priority_inventory.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r15"></a>
## R15 — Connect selected investigation to immutable executable plans

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R11](02-prompt-to-pr.md#r11), [R14](02-prompt-to-pr.md#r14).

**Current:** Immutable plan revisions and head admission exist; the actual selected generative investigation-to-plan journey is not accepted.

**Deliver:** Investigate only the freshly selected issue using current source/prerequisite evidence. Persist a concrete required-check plan and ordered task DAG with coverage, output dependencies, nonconflicting ownership and parallel lanes before coding.

**Accept only when:** No unselected issue receives a generative run; replay cannot double spend. Stale source/preparation/prerequisites cannot launch implementation. Old plan revisions survive with exact head/assignment bindings.

**Principal source owners:** [internal/factory/control/planning.go](../../../internal/factory/control/planning.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go), [internal/factory/control/dispatch_launch.go](../../../internal/factory/control/dispatch_launch.go).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R44](05-experience-and-development.md#r44). Source-only work cannot release these gates.

<a id="r16"></a>
## R16 — Connect real native task scheduling and dependency release

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R15](02-prompt-to-pr.md#r15).

**Current:** Task schemas/host scaffolding exist; real native event-to-Store progression is incomplete.

**Deliver:** Drive approved plan tasks through genuine native children under one admitted parent, with inherited identity and limits. Observe results, cancellation and retirement; release successors promptly from trusted accepted outputs.

**Accept only when:** Actual independent tasks overlap and a dependent starts while unrelated work runs. Lost start replies are looked up, not replayed. Failed/unknown outputs never release descendants or grant another lease; parent death requires explicit recovery.

**Principal source owners:** [lib/host/src/factory/tasks.rs](../../../lib/host/src/factory/tasks.rs), [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [lib/host/src/terminal/factory/children.rs](../../../lib/host/src/terminal/factory/children.rs), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r17"></a>
## R17 — Produce trusted task receipts and early Git checkpoints

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R16](02-prompt-to-pr.md#r16).

**Current:** Task-receipt CAS exists; SavePlanTaskReceipt has no production caller in the current search.

**Deliver:** Connect host-observed task output to protected exact-tree checks, serialized Git CAS checkpoint receipts and durable Store admission. Preserve worker files and unrelated dirty/index state through crash/lost reply.

**Accept only when:** Real checkpoint production calls the Store consumer. Dirty sentinel, wrong ancestry, mutated checks, forged journal and wrong invocation refuse. Crash before/after CAS recovers only the prepared commit; dependent tasks consume a trusted receipt, never a synthetic acknowledgment.

**Dependency boundary:** Establish the trusted host-observation/check/CAS/Store path here, using R30's authority for native qualification. R33 connects agent proposals to that existing owner; R17 must not require an R33 completion receipt to implement its own prerequisite, or duplicate the checkpoint engine.

**Principal source owners:** [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [lib/host/src/factory/candidate.rs](../../../lib/host/src/factory/candidate.rs), [lib/host/src/factory/manifest.rs](../../../lib/host/src/factory/manifest.rs), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

**Additional runtime/qualification gates:** [R30](04-controls-repair-verification.md#r30). Source-only work cannot release these gates.

<a id="r18"></a>
## R18 — Gate final integration and publication on the complete checked plan

**Work:** Implement and integrate. **Owner lane:** A. **Accepted inputs:** [R17](02-prompt-to-pr.md#r17).

**Current:** Candidate publication exists; complete-task/checkpoint/ancestry gates have not been qualified as one journey.

**Deliver:** Join all required task outputs, checkpoint ancestry and declared final exact-tree checks before exporting one publishable candidate. Support verification-only/no-change work without fake commits or PRs.

**Accept only when:** Incomplete tasks, stale source, wrong ancestry or unchecked trees cannot publish. Exact final artifact/tree bindings survive lost replies and cancellation; final host evidence is independently attributable.

**Principal source owners:** [lib/host/src/factory/candidate.rs](../../../lib/host/src/factory/candidate.rs), [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go).

<a id="r19"></a>
## R19 — Complete one attributable PR and publication recovery journey

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R18](02-prompt-to-pr.md#r18), [R20](03-review-merge-reassessment.md#r20).

**Current:** Conditional ref publication, PR creation and recovery primitives are real code.

**Deliver:** Compose current checkpoints with separate conditional ref and PR operations, exact receipt adoption, correction reuse and native completion. Preserve committed effects when later work fails.

**Accept only when:** One issue/candidate creates one attributable PR under lost replies, retry and restart. Matching remote names are not adopted as attribution; unexpected refs, wrong head/base and stale authority refuse.

**Principal source owners:** [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/factory/control/publication_reconcile.go](../../../internal/factory/control/publication_reconcile.go), [internal/store/factory_publications.go](../../../internal/store/factory_publications.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04). Source-only work cannot release these gates.

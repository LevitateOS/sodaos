# Role controls, repair and app verification

[Remaining-work catalog](../factory-remaining-tasks.md#task-list) owns execution order. These packets preserve the original requirement authorities; the existing central checklist alone owns live progress and writer leases. Exact source file subsets are frozen before allocation, so directories below are responsibility locators rather than blanket write leases.

<a id="r30"></a>
## R30 — Qualify one live Factory MCP authority and lifecycle

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R10](01-boundaries.md#r10).

**Current:** Catalog, schema, admission, staging and immutable submissions exist; ACTIVE/native qualification is deliberately held.

**Deliver:** Connect one server and fixed relay to exact role/subject/repository/revision/run/incarnation authority, trusted startup, reconnect/stop and receipt lifecycle. Activate only after the native boundaries actually pass.

**Ralph absorption addition:** Extend this same MCP authority to R32's typed role
communication. Derive author, role, run, assignment and subject from admission;
apply lifetime, replay, revocation and cross-scope refusal to communication as
well as formal results. Keep the existing Factory server/relay and profiles.
Native child/lane presentation does not grant the parent's mutation capability.

**Accept only when:** Cross-scope/expired/revoked/forbidden calls refuse server-side. Replay returns the same receipt, conflict refuses, restart/rebind cannot renew authority. No secrets in prompts/argv/logs or broad operator/extension socket exposure.

**Principal source owners:** [cmd/soda-dashboard/factory_mcp.go](../../../cmd/soda-dashboard/factory_mcp.go), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/mcp_admission.go](../../../internal/factory/control/mcp_admission.go), [internal/store/factory_mcp_capability.go](../../../internal/store/factory_mcp_capability.go), [lib/host/src/terminal/factory/mcp_relay.rs](../../../lib/host/src/terminal/factory/mcp_relay.rs).

<a id="r31"></a>
## R31 — Finish all four bounded MCP read controls

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30).

**Current:** Read handlers exist, but factory.evidence explicitly returns complete:false and required evidence coverage is missing.

**Deliver:** Complete factory.context, factory.evidence, factory.status and factory.receipt using bounded current canonical evidence, complete pagination, honest incompleteness and exact receipt recovery.

**Accept only when:** Each admitted profile receives its allowed current context without spending, leaking or granting authority. Missing pages cannot claim completeness. Reads need no native mutation effect; they still require live authenticated transport and correct readback.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control](../../../internal/factory/control), [internal/store/factory_mcp_submission.go](../../../internal/store/factory_mcp_submission.go).

<a id="r32"></a>
## R32 — Connect Product Lead and Architect finish controls to real consumers

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R12](02-prompt-to-pr.md#r12), [R13](02-prompt-to-pr.md#r13), [R15](02-prompt-to-pr.md#r15).

**Current:** Role/tool declarations and proposal storage exist; downstream completion is not proved by recording the proposal.

**Deliver:** Bind product-plan and issue-plan prompts/results to current common dispatch, semantic graph/manifest proposal adoption and immutable executable plans, with explicit product attention and observed terminal settlement.

**Ralph absorption addition:** Own the common typed communication producer for
all seven admitted roles: **ask, flag, learned, suggest, partial, blocked, stuck
and done**, following the
[verb blueprint](../../design/factory-feed-and-ralph-integration.md#adopted-verb-vocabulary).
Resolve the exact tool representation/schemas within Soda's existing MCP system,
reuse Ralph's useful structured payloads, and persist attributable submissions
and revisions through the existing domain/Store owners. Link canonical findings,
plans and evidence instead of creating duplicate truth. Done uses each role's
existing finish control; R33/R34/R40 retain ownership of their role-specific
downstream effects. Partial progress is nonterminal unless the role explicitly
finishes incomplete through R35. Ask/reply and suggest follow existing product
review/adoption authority; blocked/stuck are reports for deterministic assessment,
not caller-authorized readiness transitions. R42 consumes these records.

**Additional acceptance:** Exercise each verb with actual persisted readback and
the admitted role/run binding. Forged authors, wrong scope, changed-content
replay, expired/revoked capabilities and terminal-afterlife mutation refuse.
Question/answer, partial progress, known blocker and diagnosis-needed reports
remain distinct; no communication grants approval, resets allowances or turns an
agent's done report into verified completion. Update affected prompt/tool
contracts and consumers together when the extension representation is selected.

**Accept only when:** Real handler→controller→Store→permitted native effect→readback cases pass for both controls and all refusals. No-repository drafts launch nothing, model output grants no approval, and accepted finish cannot advance while the process remains uncertain.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r33"></a>
## R33 — Connect Builder task output and implementation finish to checkpoints

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R16](02-prompt-to-pr.md#r16), [R17](02-prompt-to-pr.md#r17), [R18](02-prompt-to-pr.md#r18).

**Current:** Output/finish schema and storage exist; proposal-only output cannot release dependants.

**Deliver:** Wire factory.submit_task_output and factory.finish_implementation through actual native observation, trusted checks/Git CAS/receipt and the common plan completion owner. Reuse the executor for singleton Fixer output.

**Accept only when:** Duplicate/racing proposals produce one checkpoint and no ownership rewrite. Child/handoff identity is forbidden for singleton Fixer; failed checks, mutation after release and unobserved retirement cannot advance. P8 is a real native composition.

**Principal source owners:** [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r34"></a>
## R34 — Connect Inspector and Fixer finish controls to native review/correction

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R22](03-review-merge-reassessment.md#r22), [R23](03-review-merge-reassessment.md#r23), [R24](03-review-merge-reassessment.md#r24), [R33](04-controls-repair-verification.md#r33).

**Current:** Profiles and finish proposal receipts exist; full permitted native effects and downstream acceptance are unqualified.

**Deliver:** Apply exact independent role prompts, canonical review coverage/dispositions and singleton Fixer output to factory.finish_review and factory.finish_correction. Use conditional native review/comment/resolution owners.

**Accept only when:** Clear is explicit and complete; nothing-found, incomplete and disputed reports keep their distinct semantics. No direct agent merge or authority grant. Native readback, task output and observed retirement gate progression.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/correction.go](../../../internal/factory/control/correction.go).

<a id="r35"></a>
## R35 — Finish incomplete, handoff and terminal settlement semantics

**Work:** Implement and integrate. **Owner lane:** A. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R11](02-prompt-to-pr.md#r11).

**Current:** Control/proposal scaffolding exists; terminal versus nonterminal lifecycle composition remains open.

**Deliver:** Wire factory.finish_incomplete and factory.request_handoff to the exact blocked owner/approved DAG successor and common stop/settlement path. Preserve pre-start hold/FactoryStart, receipt recovery and terminal capacity.

**Accept only when:** Blocked/nothing-found/finish are distinct. Submission is not observed termination; stop/submission/reconnect/restart races cannot release unknown children, transfer ownership arbitrarily or charge twice.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs), [lib/host/src/factory/stop.rs](../../../lib/host/src/factory/stop.rs).

<a id="r36"></a>
## R36 — Wire serving stall detection over complete component inventories

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R14](02-prompt-to-pr.md#r14), [R10](01-boundaries.md#r10).

**Current:** Classifier/helper exists; DetectRepairStalls has no production caller in the current search.

**Deliver:** Add the actual serving detector, complete component inventory and bound episode/fence. Separate active work, CI, quota, preparation, permission, pause and product attention from cycles/missing producers/unexplained no-ready work.

**Accept only when:** Ordinary waits make no investigator call; a wait in one component cannot hide repairable work in another. Partial inventory cannot prove deadlock. Ready work returns to selection and fully delivered outcomes enter verification.

**Principal source owners:** [internal/factory/control/stall_detect.go](../../../internal/factory/control/stall_detect.go), [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go).

<a id="r37"></a>
## R37 — Connect bounded Strategist repair, sponsorship and adoption

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R36](04-controls-repair-verification.md#r36), [R29](03-review-merge-reassessment.md#r29), [R11](02-prompt-to-pr.md#r11).

**Current:** Repair grants/types/accounting scaffolding exist; the complete serving repair loop is unfinished.

**Deliver:** Admit one bound repair episode under explicit sponsor and stable intent/repository/global accounting. Consume a checked Strategist proposal and finite Jev choice, then reuse protected serial graph adoption and fresh readback.

**Accept only when:** Restart, Retry, replacement issues and graph reshuffles cannot refill allowances; only material new evidence permits the allowed next cycle. No-change cannot clear a real stall; obsolete ready work retires the episode and committed prefixes remain fenced/reconciled.

**Principal source owners:** [internal/store/factory_repair_grants.go](../../../internal/store/factory_repair_grants.go), [internal/store/factory_reassessment_allowances.go](../../../internal/store/factory_reassessment_allowances.go), [internal/factory/control](../../../internal/factory/control).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09). Source-only work cannot release these gates.

<a id="r38"></a>
## R38 — Implement trusted Test Pilot manifests and artifact verification

**Work:** Implement and prove. **Owner lane:** A. **Accepted inputs:** [R10](01-boundaries.md#r10), [R13](02-prompt-to-pr.md#r13), [R18](02-prompt-to-pr.md#r18), [R27](03-review-merge-reassessment.md#r27).

**Current:** Profile/control scaffolding exists; manifest adoption, fixed trusted host-check production and exact-artifact verification remain missing or unqualified.

**Deliver:** Bind approved manifest/outcome coverage to accepted intent version and integrated tree/artifact/fixture. Launch Test Pilot under the readonly profile and existing bounded host operations, retaining trusted check and cleanup receipts.

**Accept only when:** P9 uses the actual approved fixture and current integrated bytes. Missing/stale/failed checks cannot pass; forged proposed evidence is not trusted check output. Lost replies and CLI-start ordering preserve identity and no deployment occurs.

**Principal source owners:** [internal/factory](../../../internal/factory), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store), [lib/host/src/factory/manifest.rs](../../../lib/host/src/factory/manifest.rs), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R30](04-controls-repair-verification.md#r30), [R44](05-experience-and-development.md#r44), [R45](05-experience-and-development.md#r45). Source-only work cannot release these gates.

<a id="r39"></a>
## R39 — Make app completion and delivery repair truthful and bounded

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R38](04-controls-repair-verification.md#r38), [R13](02-prompt-to-pr.md#r13), [R29](03-review-merge-reassessment.md#r29), [R37](04-controls-repair-verification.md#r37).

**Current:** An empty queue does not establish app completion; the full verification/repair subject and consumer chain is unfinished.

**Deliver:** Complete an intent only from all accepted delivered outcomes plus its exact verification receipts. Route real verification failure through the distinct bounded delivery-repair cause, stable intent ledger and protected graph owner.

**Accept only when:** Intent A can verify while unrelated B progresses. A missing fixture is a precise per-intent wait. Failure/Retry/revision cannot refill the two-run ceiling or manufacture an app intent for a manual issue; material scope questions return to the owner.

**Principal source owners:** [internal/factory](../../../internal/factory), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store).

<a id="r40"></a>
## R40 — Connect Strategist and Test Pilot finish controls

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R29](03-review-merge-reassessment.md#r29), [R37](04-controls-repair-verification.md#r37), [R38](04-controls-repair-verification.md#r38), [R39](04-controls-repair-verification.md#r39).

**Current:** factory.finish_reassessment and factory.finish_app_verification are catalogued; actual accepted downstream journeys remain unqualified.

**Deliver:** Connect typed role prompts/results and observed retirement to the reassessment/repair and exact-artifact verification/app completion owners. Finish the seven-profile instruction/context matrix.

**Accept only when:** Both controls have authenticated producer→durable admission→permitted effect→native/host readback→visible state, including stale/blocked/refused/replay cases. Tool completion grants no merge or deployment authority.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store).

<a id="r41"></a>
## R41 — Qualify intervention, withdrawal and interrupted-work recovery

**Work:** Integrate and qualify. **Owner lane:** A. **Accepted inputs:** [R06](01-boundaries.md#r06), [R11](02-prompt-to-pr.md#r11), [R16](02-prompt-to-pr.md#r16), [R26](03-review-merge-reassessment.md#r26), [R30](04-controls-repair-verification.md#r30), [R35](04-controls-repair-verification.md#r35), [R44](05-experience-and-development.md#r44).

**Current:** Cancellation, reconciliation and retained-effect mechanisms exist; full current composition is unaccepted.

**Deliver:** Compose pause/cancel/resume/Retry/takeover with exact native tasks, outstanding operations, MCP capability withdrawal and credential/resource return. Preserve finite consumption and human work.

**Accept only when:** Observe whole owned execution retirement before release or takeover; uncertain stop remains fenced. Restart adopts committed effects instead of duplicating them. Human terminals, checkouts, services and credentials remain distinct and usable.

**Principal source owners:** [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store), [lib/host/src/factory/stop.rs](../../../lib/host/src/factory/stop.rs), [lib/host/src/project/admission.rs](../../../lib/host/src/project/admission.rs).

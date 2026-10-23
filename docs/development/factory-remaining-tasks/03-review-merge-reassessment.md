# Current review, merge and reassessment

[Executable task map](../factory-remaining-tasks.md#task-list) owns the current
deliverables and output dependencies. This packet preserves the parent contracts,
technical source owners, accepted evidence and additional native/installed gates.
The Rxx.y links below identify the execution units; parent input lists describe
full qualification and do not require waiting for whole parents before using
applicable outputs. Recorded baseline and landed slices retain their actual
scope. Implementation notes support the execution map rather than define a
second schedule. This documentation restructuring starts no runtime work.

<a id="r20"></a>
## R20 — Wire fresh candidate/base tuples into every current consumer

**Execution tasks:** [R20.1](../factory-remaining-tasks.md#r20-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Native tuple/checkpoint qualification only; source conversion and classified historical BaseOID uses accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Qualify the accepted observer/consumer conversion on the current native tuple: outer-revision CAS, no churn and explicit freshness. Repair only a demonstrated missing join; retain historical PR-create/review records unchanged.

**Accept only when:** The candidate must match its trusted checkpoint; current target/diff base is never reconstructed from history. Stale refresh loses safely. Native outage cannot block committed-effect/cancel reconciliation; classify every remaining historical BaseOID use.

**Implementation notes:** Source conversion is accepted. Start at `review_cycle.go::checkCurrent`, `checks_pass.go::AssessPublicationChecks` and `merge_authorization.go::CheckMergeUnionReady`. With R05's snapshot and R17's checkpoint, qualify target-base advance and two racing observers through real adapters. Release exact candidate/current-base/native-revision tuple to R19/R21/R22/R25. Stale CAS loses, unchanged observation does not churn, and outage still permits committed-effect/cancel reconciliation. Reuse classified historical BaseOID uses; convert another source use only if it is demonstrably a current consumer.

**Principal source owners:** [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/checks_pass.go](../../../internal/factory/control/checks_pass.go), [internal/factory/control/merge_authorization.go](../../../internal/factory/control/merge_authorization.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05), [R17](02-prompt-to-pr.md#r17). Source-only work cannot release these gates.

<a id="r21"></a>
## R21 — Sponsor bounded merge-in correction when the base advances

**Execution tasks:** [R21.1](../factory-remaining-tasks.md#r21-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R19](02-prompt-to-pr.md#r19), [R20](03-review-merge-reassessment.md#r20), [R15](02-prompt-to-pr.md#r15).

**Recorded baseline:** Bounded base-advance merge-in correction sponsor. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use the existing bounded coder correction/root accounting to merge the current base into the candidate while preserving ancestry and the same PR. Resume current checks/review/acceptance only after verified new output.

**Accept only when:** Base advance launches at most the permitted correction and never silently rebases, forces, creates a new allowance or uses old acceptance. Uncertain predecessor effects remain fenced.

**Implementation notes:** Consume R19 PR, R20 changed base and R15 plan/allowance. Start at `publication_children.go::producePublicationChild` and `correctionPlanFor`. Add bounded base-advance routing to the existing coder/root accounting, then use R16/R17 to merge the new base into the candidate. Release a new checkpoint on the same PR for fresh checks/review. Test second base advance, exhaustion and uncertain predecessor in PG, then native ancestry. No rebase, force push, replenished root or stale acceptance.

**Principal source owners:** [internal/factory/control/publication_children.go](../../../internal/factory/control/publication_children.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/merge.go](../../../internal/factory/control/merge.go).

**Additional runtime/qualification gates:** [R16](02-prompt-to-pr.md#r16), [R17](02-prompt-to-pr.md#r17). Source-only work cannot release these gates.

<a id="r22"></a>
## R22 — Complete independent native reviews and full conversation evidence

**Execution tasks:** [R22.1](../factory-remaining-tasks.md#r22-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R19](02-prompt-to-pr.md#r19), [R20](03-review-merge-reassessment.md#r20).

**Recorded baseline:** Current independent native review and complete conversation journey. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Launch a fresh exact-candidate independent reviewer, gather every paged/outdated/human native thread and submit attributable anchored findings or a complete explicit clear report.

**Accept only when:** Renames/context/deletions and stale/cancelled submissions obey native atomicity. Hidden/truncated conversations cannot count as coverage; silence, approval or empty output is never clear. Actor/coder separation and immutable provenance hold.

**Implementation notes:** Consume R19/R20's exact tuple and R05's complete conversation. Start at `review_executor.go`, `review_cycle.go` and `internal/forgejo/review.go`. Run a reviewer distinct from the coder over renamed/deleted context and later-page human/outdated threads; submit attributable findings or explicit complete clear. Release report and native receipt to R23/R24. Existing review tests cover changed producers; R04/R08 native identity/atomicity witnesses cover submission. Silent, cancelled, truncated or wrong-candidate output never clears review.

**Principal source owners:** [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/review_executor.go](../../../internal/factory/control/review_executor.go), [internal/forgejo/review.go](../../../internal/forgejo/review.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04), [R05](01-boundaries.md#r05), [R08](01-boundaries.md#r08). Source-only work cannot release these gates.

<a id="r23"></a>
## R23 — Account for every correction and native thread resolution

**Execution tasks:** [R23.1](../factory-remaining-tasks.md#r23-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R22](03-review-merge-reassessment.md#r22).

**Recorded baseline:** Current finding/correction/disposition/native resolution composition. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Supply canonical current finding content to a distinct corrector. Admit candidate-bound evidence/dispositions for every item, retain carried/human history and reconcile only authorized native resolutions with exact readback.

**Accept only when:** Missing/duplicate/unknown/stale findings cannot disappear. Optional-only feedback makes no forced edit run; changed defects invalidate old evidence; unchanged real verification fixes are distinguished from unperformed code fixes.

**Implementation notes:** Consume R22 canonical findings and independent corrector identity. Start at `correction.go` and `delivery_acceptance.go::AdmitCorrectionReportCanonical`; retain immutable Store journals. Join every finding to a disposition/evidence record, then authorized native resolution with R05 readback after candidate change. Release complete current accounting to R24/R34. Focused missing/duplicate/unknown/stale cases and one native resolution cover the join. Optional-only comments require no edit run; carried unresolved human items cannot disappear on retry.

**Principal source owners:** [internal/factory/control/correction.go](../../../internal/factory/control/correction.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/store/factory_review_journals_test.go](../../../internal/store/factory_review_journals_test.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r24"></a>
## R24 — Connect clean-review and funded-Jev delivery acceptance producers

**Execution tasks:** [R24.1](../factory-remaining-tasks.md#r24-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R22](03-review-merge-reassessment.md#r22), [R23](03-review-merge-reassessment.md#r23), [R11](02-prompt-to-pr.md#r11).

**Recorded baseline:** Serving clean-review and finite funded Jev acceptance producers. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Produce durable explicit complete-clear acceptance with zero delivery calls, or a finite funded Jev routing/acceptance from complete current evidence. Connect both to the same authorization flow and bounded review/correction accounting.

**Dependency boundary:** Recording complete review/delivery acceptance can proceed while native CI is pending. R26 joins that acceptance with R25's exact checks and the remaining current authority gates before any new merge operation; acceptance alone does not grant merge authority.

**Serving progress:** `delivery_producer.go` now journals an exact settled independent clear review through `AdmitClearReview`, including its committed native review and observed candidate/merge-base tuple. Exact replay reuses the immutable acceptance; changed evidence refuses. After current policy checks pass, an undriven merge row binds this acceptance through `AuthorizeMerge` and retains the existing native approval, authority and submission gates. `TestCleanReviewServingAuthorizesMergeWithoutDeliveryCallsPG` passes against real PostgreSQL with `DeliveryCalls=0`, including pending checks, stale-policy assessment refresh and one native submission across replay. Its native review/check/merge boundaries are scripted; installed native qualification and the funded uncertain/correction producer remain open.

**Accept only when:** Real serving callers reach AdmitClearReview/AdmitJevAcceptance and AuthorizeMerge. DeliveryCalls=0 permits valid clean acceptance. Uncertainty waits; optional/disputed/repetitive feedback cannot buy endless reviews; all unchanged-head and cumulative limits survive restart.

**Implementation notes:** Consume R22 explicit complete report, R23 dispositions and R11 stable charging. Start at `AdmitClearReview`/`AdmitJevAcceptance` and `review_cycle.go::progressAfterPublish`. Wire the clean route with DeliveryCalls=0, then the existing Jev client/reservation owner for correction or uncertainty within a finite approved budget. Release durable tuple-bound acceptance to R26 while checks may still be pending. PG serving tests show one acceptance/charge across retry and bounded unchanged-head correction. Real Jev acceptance waits for provider input; uncertainty stays a wait.

**Principal source owners:** [internal/factory/control/delivery_acceptance.go](../../../internal/factory/control/delivery_acceptance.go), [internal/factory/control/merge_authorization.go](../../../internal/factory/control/merge_authorization.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go).

<a id="r25"></a>
## R25 — Evaluate every configured native check on the exact current tuple

**Execution tasks:** [R25.1](../factory-remaining-tasks.md#r25-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R19](02-prompt-to-pr.md#r19), [R20](03-review-merge-reassessment.md#r20).

**Recorded baseline:** Exact-tuple complete native check evidence. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect complete nonempty configured Actions/check evidence to current candidate and verified base, exposing failure versus infrastructure wait and refusing unaccepted check-policy edits.

**Accept only when:** Missing, skipped, cancelled, failed, stale or partial check sets cannot pass. Native separately managed runner capacity remains upstream; no substitute CI engine or agent success narrative.

**Implementation notes:** Consume R19/R20 tuple, R05 complete check inventory and R04 authority. Start at `internal/forgejo/checks.go` and `AssessPublicationChecks`. Drive the actual configured nonempty check set through the observer, including later pages and head change during polling. Release exact-tuple assessment to R26. Extend changed-join tests; native witnesses cover failed/skipped/cancelled/missing/stale/partial sets and capacity wait. Policy edits require existing authorization; agent prose does not substitute for check records.

**Principal source owners:** [internal/forgejo/checks.go](../../../internal/forgejo/checks.go), [internal/factory/control/checks_pass.go](../../../internal/factory/control/checks_pass.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04), [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r26"></a>
## R26 — Gate every merge path with current acceptance and authorization

**Execution tasks:** [R26.1](../factory-remaining-tasks.md#r26-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R20](03-review-merge-reassessment.md#r20), [R21](03-review-merge-reassessment.md#r21), [R24](03-review-merge-reassessment.md#r24), [R25](03-review-merge-reassessment.md#r25).

**Recorded baseline:** Join current acceptance and authorization at every new merge entry. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Apply clean/Jev union, tuple/source/policy/native approval/protection/check gates to every new merge entry. Serialize withdrawal, cancellation, replacement generations and resolution readback without losing already committed effects.

**Accept only when:** Real-PG plus native refusal/recovery cases cover every entrypoint. Missing acceptance cannot create a new operation; uncertain predecessor cannot be replaced; a committed lost response reconciles without re-merge or rejudgment.

**Implementation notes:** Consume R20 tuple, R21 corrected head when needed, R24 acceptance and R25 checks. Start at `AuthorizeMerge`, `merge.go` and `reconcileMerge`. Route every new operation through the same current union gate, separately from reconciliation of committed operations. Test withdrawal/cancel/replacement in real PG, then R04 native lost-response recovery. Release attributable merge/completion to R27. Missing acceptance creates no operation; unknown predecessor remains fenced and committed effects reconcile without re-merge or rejudgment.

**Principal source owners:** [internal/factory/control/merge.go](../../../internal/factory/control/merge.go), [internal/factory/control/merge_reconcile.go](../../../internal/factory/control/merge_reconcile.go), [internal/factory/control/merge_authorization.go](../../../internal/factory/control/merge_authorization.go), [internal/store/factory_merges.go](../../../internal/store/factory_merges.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04). Source-only work cannot release these gates.

<a id="r27"></a>
## R27 — Confirm original issue completion before releasing dependants

**Execution tasks:** [R27.2](../factory-remaining-tasks.md#r27-2), [R27.1](../factory-remaining-tasks.md#r27-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R26](03-review-merge-reassessment.md#r26), [R14](02-prompt-to-pr.md#r14).

**Recorded baseline:** Composed original-closure readiness release; evidence/authority/CAS/completion source checks accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind reachable attributable merge, native completion and the approved original issue outcome to durable completion/readiness. Preserve outcome truth and record each completion event exactly once.

**Accept only when:** A two-parent join waits for both original verified closures; reopened/stale closure cannot release it. Completion, event and root closure are atomic; committed effects survive restart and no old root silently reopens.

**Implementation notes:** Consume R26's attributable merge/native original-issue completion and R14 readiness. Start at `merge_evidence.go::completeMerge`, `factory_merges.go` and `prerequisites.go`. Compose accepted completion/CAS with a two-parent native join: merge without bookkeeping, later recovery, then reopen one prerequisite. Release one atomic completion/event/root closure to R28 and readiness only when both verified outcomes hold. Reuse source proof; add the serving/native recovery witness rather than a second completion engine.

**Principal source owners:** [internal/factory/control/merge_evidence.go](../../../internal/factory/control/merge_evidence.go), [internal/store/factory_merges.go](../../../internal/store/factory_merges.go), [internal/factory/control/prerequisites.go](../../../internal/factory/control/prerequisites.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r28"></a>
## R28 — Complete the post-completion Jev reassessment fence

**Execution tasks:** [R28.1](../factory-remaining-tasks.md#r28-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R27](03-review-merge-reassessment.md#r27), [R11](02-prompt-to-pr.md#r11).

**Recorded baseline:** Two-completion/two-Jev-decision serving proof; fence/drive/replay PG checks accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** After each verified completion, collect the full current repository inventory and make the bounded persisted Jev continue/reassess decision. Enforce the repository fence across selection, Retry and phase advancement.

**Accept only when:** Two completions require two decisions. Continue launches no investigator; reassess admits one bounded completion-subject run. All pages/manual/blocked/protected records are represented; unrelated repositories progress and no one-item shortcut bypasses the gate.

**Implementation notes:** Consume R27's durable completion event and R11's once-charged decision path. Start at `completion_drive.go::DriveCompletions`, `completion_decision.go::DecideReassessment` and Store reassessment records. Drive two actual completion events through full paged inventory collection and two separately persisted Jev decisions; continue must launch no Strategist, reassess exactly one completion-subject run. Release the sealed cycle/inventory and repository fence to R29. Reuse accepted PG replay/fence proof, adding the serving two-completion witness and funded real-client case only where unproved. Other repositories remain live; Retry cannot bypass the undecided fence.

**Principal source owners:** [internal/factory/reassessment.go](../../../internal/factory/reassessment.go), [internal/store/factory_reassessments.go](../../../internal/store/factory_reassessments.go), [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r29"></a>
## R29 — Complete protected reassessment adoption and prefix recovery

**Execution tasks:** [R29.1](../factory-remaining-tasks.md#r29-1).

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R28](03-review-merge-reassessment.md#r28), [R13](02-prompt-to-pr.md#r13).

**Recorded baseline:** The serving driver now connects a sealed completion inventory to one atomically admitted reassessment coder, its typed report, a separately funded proposal assessment, serial graph materialization and fresh no-change/full-adoption release. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Consume checked no-change or proposed revisions, assess through finite Jev policy and adopt serial exact native graph operations without replacing protected/manual/active/completed identities or accepted history.

**Ongoing Git-issue ownership:** Strategist owns in-scope issue-plan maintenance:
clarify idle tasks, split/combine them, add missing tasks, retire redundant tasks
and repair dependencies. Consume R40's exact diagnosis/draft/preview/finish and
R32's attributed decision references; Factory/native operations apply the graph.
Preserve protected manual/active/delivered/PR-linked work. Retiring a task uses
supersession and removes dispatch authority, not false delivered closure. Qualify
these changes with full outcome/dependency coverage, lost replies and partial
adoption; R42 shows proposed versus confirmed changes. Scope expansion returns
to the actual product owner. Generic labels/assignees/milestones or arbitrary
issue deletion are not silently added to this bounded contract.

**Accept only when:** Current scope/coverage/route/packet bounds refuse before effects. Lost replies and a later conflicting intent preserve the committed prefix and fence successor work. Only confirmed no-change/full adoption releases its event; Retry cannot abandon an activated head.

**Implementation notes:** `completion_proposal.go` consumes the exact sealed inventory and sole admitted coder report, runs deterministic adoption validation before the funded proposal call, and rechecks freshness before effects. `completion_adoption.go` uses the existing intent-generation owner; activation records the successor head and current graph operation in one transaction. Lost replies look up the same operation before comparing freshness, and only the journal's exact recorded effects are normalized when checking for unrelated drift. Protected/manual/active/delivered/PR-linked tasks retain their identities and prior acceptance history. Real PostgreSQL serving checks cover no-change release, non-contiguous native issue indexes, explicit retry after settled coder failure, edit/lost-reply adoption, local-only retirement and two-intent adoption across a controller restart. A later conflicting human head preserves the completed first intent, prevents activation of the second successor and keeps its fence. Native graph and model boundaries are scripted; installed native graph readback and provider execution remain qualification work. Never abandon an activated unmaterialized head.

**Principal source owners:** [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/store/factory_intent_generation.go](../../../internal/store/factory_intent_generation.go), [internal/store/factory_reassessments.go](../../../internal/store/factory_reassessments.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

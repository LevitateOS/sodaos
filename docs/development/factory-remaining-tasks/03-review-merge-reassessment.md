# Current review, merge and reassessment

[Remaining-work catalog](../factory-remaining-tasks.md#task-list) owns execution order. These packets preserve the original requirement authorities; the existing central checklist alone owns live progress and writer leases. Exact source file subsets are frozen before allocation, so directories below are responsibility locators rather than blanket write leases.

<a id="r20"></a>
## R20 — Wire fresh candidate/base tuples into every current consumer

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Current:** The eight-file tuple candidate has narrow real-PG evidence but is unapplied; the later 183B increment is incomplete and unrun.

**Deliver:** Persist actual observer snapshots with outer-revision CAS, no churn and explicit freshness. Convert all current diff/check/review/judgment/dispatch/merge consumers; retain historical PR-create/review records unchanged.

**Accept only when:** The candidate must match its trusted checkpoint; current target/diff base is never reconstructed from history. Stale refresh loses safely. Native outage cannot block committed-effect/cancel reconciliation; classify every remaining historical BaseOID use.

**Principal source owners:** [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/checks_pass.go](../../../internal/factory/control/checks_pass.go), [internal/factory/control/merge_authorization.go](../../../internal/factory/control/merge_authorization.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05), [R17](02-prompt-to-pr.md#r17). Source-only work cannot release these gates.

<a id="r21"></a>
## R21 — Sponsor bounded merge-in correction when the base advances

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R19](02-prompt-to-pr.md#r19), [R20](03-review-merge-reassessment.md#r20), [R15](02-prompt-to-pr.md#r15).

**Current:** Holding a base-ahead candidate exists as a draft; holding is not an actual correction sponsor.

**Deliver:** Use the existing bounded coder correction/root accounting to merge the current base into the candidate while preserving ancestry and the same PR. Resume current checks/review/acceptance only after verified new output.

**Accept only when:** Base advance launches at most the permitted correction and never silently rebases, forces, creates a new allowance or uses old acceptance. Uncertain predecessor effects remain fenced.

**Principal source owners:** [internal/factory/control/publication_children.go](../../../internal/factory/control/publication_children.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/merge.go](../../../internal/factory/control/merge.go).

**Additional runtime/qualification gates:** [R16](02-prompt-to-pr.md#r16), [R17](02-prompt-to-pr.md#r17). Source-only work cannot release these gates.

<a id="r22"></a>
## R22 — Complete independent native reviews and full conversation evidence

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R19](02-prompt-to-pr.md#r19), [R20](03-review-merge-reassessment.md#r20).

**Current:** Native inline anchoring, submission and finding records exist; the complete current review journey is not accepted.

**Deliver:** Launch a fresh exact-candidate independent reviewer, gather every paged/outdated/human native thread and submit attributable anchored findings or a complete explicit clear report.

**Accept only when:** Renames/context/deletions and stale/cancelled submissions obey native atomicity. Hidden/truncated conversations cannot count as coverage; silence, approval or empty output is never clear. Actor/coder separation and immutable provenance hold.

**Principal source owners:** [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/review_executor.go](../../../internal/factory/control/review_executor.go), [internal/forgejo/review.go](../../../internal/forgejo/review.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04), [R05](01-boundaries.md#r05), [R08](01-boundaries.md#r08). Source-only work cannot release these gates.

<a id="r23"></a>
## R23 — Account for every correction and native thread resolution

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R22](03-review-merge-reassessment.md#r22).

**Current:** Correction/disposition/history records and narrow repairs exist; complete current producer/consumer accounting remains open.

**Deliver:** Supply canonical current finding content to a distinct corrector. Admit candidate-bound evidence/dispositions for every item, retain carried/human history and reconcile only authorized native resolutions with exact readback.

**Accept only when:** Missing/duplicate/unknown/stale findings cannot disappear. Optional-only feedback makes no forced edit run; changed defects invalidate old evidence; unchanged real verification fixes are distinguished from unperformed code fixes.

**Principal source owners:** [internal/factory/control/correction.go](../../../internal/factory/control/correction.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/store/factory_review_journals_test.go](../../../internal/store/factory_review_journals_test.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r24"></a>
## R24 — Connect clean-review and funded-Jev delivery acceptance producers

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R22](03-review-merge-reassessment.md#r22), [R23](03-review-merge-reassessment.md#r23), [R11](02-prompt-to-pr.md#r11).

**Current:** Acceptance union/admission helpers exist with no production callers in the inspected source.

**Deliver:** Produce durable explicit complete-clear acceptance with zero delivery calls, or a finite funded Jev routing/acceptance from complete current evidence. Connect both to the same authorization flow and bounded review/correction accounting.

**Dependency boundary:** Recording complete review/delivery acceptance can proceed while native CI is pending. R26 joins that acceptance with R25's exact checks and the remaining current authority gates before any new merge operation; acceptance alone does not grant merge authority.

**Accept only when:** Real serving callers reach AdmitClearReview/AdmitJevAcceptance and AuthorizeMerge. DeliveryCalls=0 permits valid clean acceptance. Uncertainty waits; optional/disputed/repetitive feedback cannot buy endless reviews; all unchanged-head and cumulative limits survive restart.

**Principal source owners:** [internal/factory/control/delivery_acceptance.go](../../../internal/factory/control/delivery_acceptance.go), [internal/factory/control/merge_authorization.go](../../../internal/factory/control/merge_authorization.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go).

<a id="r25"></a>
## R25 — Evaluate every configured native check on the exact current tuple

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R19](02-prompt-to-pr.md#r19), [R20](03-review-merge-reassessment.md#r20).

**Current:** Check observation/policy code exists. This is original implementation-plan D13, not post-completion reassessment.

**Deliver:** Connect complete nonempty configured Actions/check evidence to current candidate and verified base, exposing failure versus infrastructure wait and refusing unaccepted check-policy edits.

**Accept only when:** Missing, skipped, cancelled, failed, stale or partial check sets cannot pass. Native separately managed runner capacity remains upstream; no substitute CI engine or agent success narrative.

**Principal source owners:** [internal/forgejo/checks.go](../../../internal/forgejo/checks.go), [internal/factory/control/checks_pass.go](../../../internal/factory/control/checks_pass.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04), [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r26"></a>
## R26 — Gate every merge path with current acceptance and authorization

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R20](03-review-merge-reassessment.md#r20), [R21](03-review-merge-reassessment.md#r21), [R24](03-review-merge-reassessment.md#r24), [R25](03-review-merge-reassessment.md#r25).

**Current:** Merge primitives/generations/recovery exist; the real acceptance producer and all current consumers remain incomplete.

**Deliver:** Apply clean/Jev union, tuple/source/policy/native approval/protection/check gates to every new merge entry. Serialize withdrawal, cancellation, replacement generations and resolution readback without losing already committed effects.

**Accept only when:** Real-PG plus native refusal/recovery cases cover every entrypoint. Missing acceptance cannot create a new operation; uncertain predecessor cannot be replaced; a committed lost response reconciles without re-merge or rejudgment.

**Principal source owners:** [internal/factory/control/merge.go](../../../internal/factory/control/merge.go), [internal/factory/control/merge_reconcile.go](../../../internal/factory/control/merge_reconcile.go), [internal/factory/control/merge_authorization.go](../../../internal/factory/control/merge_authorization.go), [internal/store/factory_merges.go](../../../internal/store/factory_merges.go).

**Additional runtime/qualification gates:** [R04](01-boundaries.md#r04). Source-only work cannot release these gates.

<a id="r27"></a>
## R27 — Confirm original issue completion before releasing dependants

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R26](03-review-merge-reassessment.md#r26), [R14](02-prompt-to-pr.md#r14).

**Current:** Completion/closure/prerequisite code exists; the full current serving chain remains unaccepted.

**Deliver:** Bind reachable attributable merge, native completion and the approved original issue outcome to durable completion/readiness. Preserve outcome truth and record each completion event exactly once.

**Accept only when:** A two-parent join waits for both original verified closures; reopened/stale closure cannot release it. Completion, event and root closure are atomic; committed effects survive restart and no old root silently reopens.

**Principal source owners:** [internal/factory/control/merge_evidence.go](../../../internal/factory/control/merge_evidence.go), [internal/store/factory_merges.go](../../../internal/store/factory_merges.go), [internal/factory/control/prerequisites.go](../../../internal/factory/control/prerequisites.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r28"></a>
## R28 — Complete the post-completion Jev reassessment fence

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R27](03-review-merge-reassessment.md#r27), [R11](02-prompt-to-pr.md#r11).

**Current:** Integrated source and narrow PG proofs exist; a stale failure receipt does not negate them. This is contracts D13.

**Deliver:** After each verified completion, collect the full current repository inventory and make the bounded persisted Jev continue/reassess decision. Enforce the repository fence across selection, Retry and phase advancement.

**Accept only when:** Two completions require two decisions. Continue launches no investigator; reassess admits one bounded completion-subject run. All pages/manual/blocked/protected records are represented; unrelated repositories progress and no one-item shortcut bypasses the gate.

**Principal source owners:** [internal/factory/reassessment.go](../../../internal/factory/reassessment.go), [internal/store/factory_reassessments.go](../../../internal/store/factory_reassessments.go), [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

<a id="r29"></a>
## R29 — Complete protected reassessment adoption and prefix recovery

**Work:** Integrate and qualify. **Owner lane:** B. **Accepted inputs:** [R28](03-review-merge-reassessment.md#r28), [R13](02-prompt-to-pr.md#r13).

**Current:** Substantial reassessment/history/adoption source and narrow fixes are present, with incomplete current composition.

**Deliver:** Consume checked no-change or proposed revisions, assess through finite Jev policy and adopt serial exact native graph operations without replacing protected/manual/active/completed identities or accepted history.

**Accept only when:** Current scope/coverage/route/packet bounds refuse before effects. Lost replies and a later conflicting intent preserve the committed prefix and fence successor work. Only confirmed no-change/full adoption releases its event; Retry cannot abandon an activated head.

**Principal source owners:** [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/store/factory_intent_generation.go](../../../internal/store/factory_intent_generation.go), [internal/store/factory_reassessments.go](../../../internal/store/factory_reassessments.go).

**Additional runtime/qualification gates:** [R05](01-boundaries.md#r05). Source-only work cannot release these gates.

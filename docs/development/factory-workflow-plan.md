# Complete the software factory workflow

Status: **planned, not implemented or qualified**. This plan follows the owner's
2026-10-16 request to plan all gaps in the prompt-to-delivery loop before further
implementation. It is based on read-only inspection of Soda
`6d0281e64102269fda3614c7bd77948c4845914e` and Fountain
`a4fd2221f16f181e5ec36c1b11d4181a936052e7`. No new runtime evidence is claimed.

This document owns the corrective implementation queue for that workflow. The
[existing factory plan](factory-implementation-plan.md) retains its broader
source history, appliance and installed-experience acceptance. For overlapping
workflow tasks, use this queue rather than executing both plans. Preserve already
completed work. This is separate from the completed RCX-2 release-component work.

Product behavior belongs in [the product overview](../product/overview.md),
interfaces in [factory interfaces](../architecture/factory-interfaces.md), and
existing operator inputs in [the factory reference](../reference/factory.md).
Implementation tasks below update those owners together with their code; this
plan does not claim the proposed interfaces already exist.

## Required outcome

1. A prompt produces coherent, PR-sized native issues with acceptance outcomes
   and real dependencies. Any number can be ready; two is an example, not a limit.
2. Jev selects from the complete eligible set. A single eligible issue needs no
   comparative selection call. Blocked issues cannot be selected for coding.
3. An agent investigates the selected issue against current source, writes a
   concrete implementation plan, then produces ordered tasks and parallel lanes.
4. Implementation releases tasks when their prerequisites finish, makes coherent
   commits throughout, integrates the result, and opens one PR for the issue.
5. A separate reviewer examines the candidate and posts native inline findings.
   A fresh correction agent addresses every finding with explicit evidence.
6. Jev chooses another independent review or readiness to merge. Review/fix can
   repeat, including another review on an unchanged head. A clean first review
   also reaches Jev's decision; reviewer approval alone does not bypass it.
7. A fresh recorded Jev acceptance plus deterministic authority, checks and native
   branch policy allow an exact-head merge. Confirmed merge and attributable issue
   completion unlock dependants, and the coordinator continues selecting work.

The first independent review is mandatory. A second review after corrections is
Jev's choice when native policy permits either path. Native required approvals,
blocking reviews and checks still apply; Jev cannot create an approval or waive
branch protection. Missing evidence, funds or authority produces a visible wait,
not an optimistic merge.

## Inspected baseline and gaps

Paths in this table are relative to Soda unless prefixed `Fountain:` (the canonical
`forgejo-ext` checkout). Evidence describes source, not newly executed tests.

| Area | Present behavior | Required correction and source evidence |
| --- | --- | --- |
| Prompt decomposition | Native task graphs, multiple roots, cycle checks and accepted-scope provenance already exist. | `internal/factory/intent_proposal.go` emits generic implementation/verification tasks; `decomposition_options.go` splits or combines templates. Generate semantic candidates with real feature boundaries. |
| Readiness and selection | `control/readiness.go`, `priority.go` and `priority_dispatch.go` queue independent roots, compare candidates with Jev, apply fairness and recheck admission. | Full collection restarts each pass while sharing a bounded RPC allowance. Prove the suspected large-queue starvation before selecting a fix. |
| Investigation and planning | `control/planning.go` selects a strategy and emits three fixed strings. | `factory/planning.go` has `Steps []string`, without inspected evidence, task dependencies, lanes or execution receipts. Replace the placeholder planning path. |
| Execution phases | Existing assignments, preparation, broker custody, attempt roots and run history support bounded work. | `dispatch_launch.go` treats non-root publication assignments as children requiring a PR; completed `result.go` requires a commit. Add explicit prepublication phase/result contracts. |
| Parallel work and commits | Project launchers start one headless CLI; repository session capacity is one. Publication already consumes an exact candidate. | Native subagent control/inheritance/cancellation is unqualified. `dispatch_prompt.go` asks for a final commit, without task checkpoints or integration evidence. |
| Reviews | Independent reviewer/coder roles and fresh run IDs exist. Exact-head conditional native review submission exists. | `factory/review_native.go`, `forgejo/review.go`, and Fountain `sdk/review.go` support body-only reviews. Findings are strings, without line anchors or native comment receipts. |
| Corrections | `control/correction.go` accepts a successful changed candidate. | No complete finding-to-fix accounting. `dispatch_prompt_stage.go` excludes prior review evidence from reviewer input. |
| Review/merge choice | `publication_children.go` deterministically reviews each new head, then corrects requested changes. | No funded Jev delivery decision. Child identity hashes publication/head/role, preventing a new review of the same head. Soda's merge observation requires exact-head approval unconditionally. |
| Completion | `merge_evidence.go`, `store/factory_merges.go`, and `prerequisites.go` verify merge plus original closure and queue reassessment. | Preserve this path and compose it with the new stages. Existing serving journeys start with a hand-authored task graph, so do not prove prompt decomposition. |
| Visibility | Existing Work/Spaces views expose factory work and persistent terminals. | Expose investigation, immutable plan, live lanes, checkpoint commits, findings, correction evidence and Jev's current decision without creating another workflow UI. |

Retain the serving coordinator's existing ten-second progress mechanism. Do not
add another scheduler, a duplicate issue database, or a generic job framework.
Keep factory domain/coordinator/store code in Go and privileged Project execution
in its existing Rust owners. Fountain supplies generic collaboration primitives;
it must not learn Soda's planning or Jev policy.

## Implementation design

### Semantic decomposition and selection

Use a bounded generative planning agent to turn the accepted objective, reviewed
product outcomes and available repository evidence into structured candidate
graphs. Jev evaluates supplied alternatives; it cannot inspect source or invent
an implementation plan. Preserve `IntentTask`, `IntentRevision`, native graph
writing, accepted revisions, stable task bindings and existing scope review.

Each candidate contains cohesive issue objectives, observable outcomes, covered
accepted ideas and dependencies on specific prerequisite outputs. Code validates
bounds, explicit outcome coverage, provenance and the dependency DAG. The planner
and Jev assess semantic cohesion, actual outcome coverage and necessary edges
from supplied evidence; string equality or keyword checks cannot establish those
properties. Representative real-provider evidence challenges that assessment.
Verification belongs with each deliverable unless a separate
integration issue has its own useful acceptance. A small prompt may legitimately
produce one issue. No fixed issue count or mandatory artificial parallelism.

Before repository association, retain a private editable product draft using
supplied evidence; do not start a generative Project run without an authorized
Project. That draft is not the final issue graph. Semantic decomposition runs
after authorized association and preparation, before native issues are created.
Its input is the reviewed product direction/draft revision and effective approved
outcomes, not an already accepted task graph: current accepted `IntentRevision`
requires repository and tasks, so requiring it here would be circular.

Extend the existing Project execution boundary with an explicit **intent-bound
planning subject**; do not manufacture an issue or grant publication authority.
Bind it to the exact draft/review receipt, owner, concrete repository/Project and
preparation, provider account, deadline, current grants and finite run allowance.
An associated repository alone is not execution permission. Candidate adoption
then produces the accepted graph revision through the existing scope rules; new
material product assumptions return to review, without approvals for each
technical task. Jev consultation reservations use the intent's independent allowance
head; legacy `IntentRevision.PlanningCalls` is not funding. Generative agent usage
needs its own existing provider/run accounting, not a relabelled classifier call.
Gate P4 settles the smallest usable execution path before implementation.

Separate **eligible for issue selection** from **eligible to implement**. A ready
accepted issue can enter Jev's selection without an implementation plan. The
selected issue then gets a source-bound investigation and plan. Coding requires
that fresh validated plan. Provisional planning for blocked work may remain
advisory, but cannot skip investigation against actual merged prerequisites.
Persist selection-to-investigation admission atomically under current authority
and capacity, with its phase and attempt identity. Unselected ready issues must
not incur generative investigation. Restart resumes the recorded choice without
duplicate spending; invalidation before planning or coding triggers reassessment.

For complete-set selection, first reproduce the source-derived starvation
concern: production refresh consumes readiness RPCs, the shared work cap is 156,
and collection restarts and scans again after advice. If confirmed, retain
revision-bound inventory progress across passes or share a correctly bound native
inventory snapshot, choosing the smaller proved change. Keep complete-set
comparison, finite work limits, invalidation on inventory change and a fresh
selected-issue admission check. Never truncate candidates to fit the budget.

### Investigated plan and execution records

Extend current factory records in place, with one canonical definition per
boundary. Replace obsolete experimental forms and callers together; no legacy
reader or compatibility migration is required for this unreleased implementation.

| Record | Required content and validation |
| --- | --- |
| Execution subject and phase | Intent planning or issue attempt; phase independent of account role; explicit parent/run identity, accepted-input and preparation binding. Keep `soda-coder`/`soda-reviewer` roles. |
| Investigation | Inspected source references at a known revision, findings, affected owners/interfaces, requirements and unresolved assumptions. Unresolved mandatory facts block implementation. |
| Immutable plan revision | Accepted scope/input digest, source/prerequisite/preparation identities, investigation, approach, alternatives considered, verification, tasks and lanes. Keep the exact revision consumed by each run; a mutable current pointer does not replace history. |
| Task | Unique ID, objective, expected output, acceptance coverage, dependencies, lane, exclusive paths/interfaces, checks and checkpoint expectation. Enforce bounded sizes, known references, a DAG and no concurrent conflicting ownership. |
| Task execution receipt | Attempt/plan/task/lane/run identities, state, consumed prerequisite output digests, actual start/completion, check results and produced commits. Only validated successful prerequisites release a dependent task. |
| Phase result | Planning returns an investigated plan; implementation/correction returns a candidate and evidence; review returns findings. A planning result cannot be interpreted as publishable code. |

One bounded run may perform investigation, approach selection and task planning
in order. Implementation starts only after its structured result is validated and
persisted. Do not add a separate model call for each heading merely to mirror the
workflow. Keep one issue attempt root and cumulative allowances across planning,
implementation, review, correction and repeated review; stages do not reset limits.
Intent planning before issue creation retains its separate explicitly funded scope.

Admission records a finite `MaxChildren`, bounded by the proved harness capability
and authorized run policy; the scheduler cannot exceed it even when more tasks are
ready. This is distinct from the single admitted repository session. Preserve the
existing active-minute/correction-cycle units. Record provider consumption in its
reported native units (such as input/output tokens), attributing every child to
its parent and counting each charge once; never add inclusive parent totals to
the same child totals. P1 must establish observation and enforcement of the
configured provider limit. A time/concurrency bound must not be described as a
monetary limit where the provider exposes no such enforcement.

Delivery judgments need a new finite **call subledger** under the attempt root:
the existing `AttemptAllowance` has active minutes and correction cycles, not
classifier credits. Its grant is part of the sponsor's admitted attempt terms,
with reserved/consumed counts and immutable request identities. Neither a phase
change nor another review refills it; missing or exhausted grants block further
judgments. Reuse the existing reservation transaction pattern and store owner.

Use a narrow native task executor inside one admitted issue run, if gate P1 proves
it. The interface must expose child start, observable identity/progress, join,
result and cancellation sufficiently to enforce the task DAG. It must release a
newly ready task without waiting for an unrelated slow lane. A prompt asking for
parallelism is not a scheduler or evidence of execution.

The Project-side parent executor owns Git integration and the index; the Go
coordinator validates and stores its receipts. Workers own disjoint
files or explicitly released interfaces; shared-file changes are serialized.
Checkpoint coherent verified outputs during implementation, staging only completed
tasks' released paths. Record the exact committed tree and the tree checked; a
check against a mixed dirty checkout cannot prove that checkpoint. Checks needing
the integrated tree run after the required writers release it. Verify commits
exist and are ancestors of the integrated candidate. Do not force meaningless
minimum commit counts. No final PR until all
required tasks and checks are complete on the integrated candidate.

Keep the repository's one-admitted-session policy. If pinned CLI native children
cannot meet the contract, implement the smallest necessary capability in the
existing Project-side executor after recording the failed assumption and evidence.
Do not silently serialize a parallel plan, create hidden broker leases, raise
repository capacity, or introduce a second execution service. Child identities,
credential scope, provider usage and cancellation must remain attributable to the
parent's finite admission. Preserve native CLI text output through the existing
terminal stream; screenshots are only for genuinely visual acceptance.

### Native inline findings and correction evidence

Extend Fountain's existing conditional `pull_request.review.submit` primitive,
using its native review/code-comment machinery. A review contains a bounded
summary and structured findings: stable key, severity, actionable/optional
classification, defect/evidence/required outcome, reviewed head/base, and a valid
diff path/side/line for code findings. Non-code findings remain identified summary
findings; never invent a line to make them appear inline.

Submission atomically creates the summary and native code comments under the
existing actor, revision, deadline and invalidation gates. Return an immutable
finding-key-to-native-comment-ID receipt. Validate anchors against the candidate
diff, including deleted lines. Replayed operation identities return the same
receipt; stale head/base refuses the entire operation without partial comments.

Extend the existing paged native snapshot with locations, original/current
commit, thread identity, content version and resolution/resolver evidence. Include
outdated unresolved findings. All pages must form a complete snapshot at one
native revision; hidden, omitted, truncated or unavailable required evidence
blocks acceptance. Add a narrow conditional thread-resolution operation
only where the current SDK lacks it, reusing native `MarkConversation` and existing
permissions. Bind resolution to exact PR/head/thread/content identity. An agent's
claim or the native resolved marker alone is not proof that a defect was fixed.

Expose generic, read-only native merge eligibility for the actual merge actor,
bound to PR/head/base and native revision. Reuse the native policy evaluator to
report required approvals, blocking reviews/requests, checks and other blockers;
current snapshots do not provide this contract. This observation guides review
and Jev choices, but grants no authority and never replaces rechecking native
policy during the conditional merge mutation.
An unresolved-thread blocker that can be cleared by accepting the supplied fix
evidence does not prevent that evidence from reaching Jev. Acceptance is then
conditional on the expected resolution and fresh policy readback. A required
approval or blocking change request is a separate native constraint.

Correction input includes the complete current finding-set digest and native
comment references. Output accounts for every finding exactly once:

- Actionable: fixed with candidate-bound code/check evidence, or unresolved/
  disputed with reasons. Unresolved actionable findings block acceptance.
- Optional: implemented with evidence or explicitly declined with reasons.

Reject missing, duplicate, unknown or stale dispositions. Distinguish reported
fixes from mechanically verified evidence (candidate ancestry, changed paths,
recorded checks and current native readback). Semantic sufficiency is assessed by
Jev or the next independent reviewer, not inferred from a changed commit alone.
Keep `fix reported`, `fix accepted` and `native thread resolved` distinct. A
mechanically complete proposed fix reaches Jev/review while its thread can still
be open. After semantic acceptance, journal resolution operations and confirm
native readback before merge. Expected resolution writes are reconciliation of
that accepted decision, not new finding content requiring another paid judgment;
unexpected content/policy/head changes invalidate it. Disputes use existing maintainer intervention rather
than an unbounded argument between agents.

Reviewer and correction work remain separate fresh runs. Re-review receives the
original findings and correction facts to verify, without the coding agent's
private conversation or a prewritten approval. Persist a review-cycle identity
independent of the candidate hash: a new review decision can create a fresh run
on the same head; replay of the same decision cannot create duplicate runs.

### Jev decision and one common merge gate

Add a delivery-review judgment purpose using the existing typed chooser,
`AdvisePurpose`, durable request/answer receipts and finite reservation machinery.
Jev supplies a choice; code supplies evidence, admissible actions and authority.
Follow the current [Choice contract](https://docs.typesafe.ai/primitives/choice)
and [HTTP API](https://docs.typesafe.ai/api), retaining the installed Go integration.

The durable input fingerprint binds publication/cycle, exact candidate/base,
accepted objective and policy revisions, original independent review, native
finding IDs/content versions, complete correction dispositions, correction diff,
tests/checks and native merge blockers. Reserve the consultation from the active
attempt's allowance before sending; do not borrow the pre-issue intent allowance.
Preserve consumed-call accounting and reuse unchanged recorded judgments.

| Observed state | Next action |
| --- | --- |
| No first independent review | Dispatch reviewer. |
| Actionable findings lack complete proposed fixes, are disputed, or required code checks fail | Dispatch bounded correction when fixable; otherwise expose existing intervention/wait. An open thread with complete proposed fix evidence can proceed to judgment. |
| Native policy requires another approval | Obtain the required independent review; Jev cannot waive this gate. |
| Evidence/authority is incomplete, checks are pending, or funds are exhausted | Wait with the exact reason; no merge acceptance. |
| Independent review and mechanically complete correction evidence is ready for semantic judgment, and native policy permits the path | Ask Jev to choose `review_again` or `ready_to_merge`. This also applies after a clean first review. The choice assesses correction sufficiency; incomplete evidence cannot be accepted. |
| Valid `review_again` | Persist the decision/cycle, dispatch a fresh reviewer, then reconsider after its result. |
| Valid `ready_to_merge` | Record candidate-bound Soda acceptance, reconcile accepted finding resolutions, then recheck all deterministic/native conditions before merge. |

Insufficient-confidence advice conservatively requests review within the remaining
allowance; otherwise waits. Missing/malformed/provider-failed answers cannot create
acceptance. Do not silently repeat a consumed request. Freeze confidence policy
and failure accounting in FW06C using existing conventions, with no invented
unbounded retry budget.

Replace Soda's mandatory fresh approval after every correction with two supported
evidence forms: independent approval of the candidate, or prior independent
findings plus complete candidate-bound correction evidence. **Both require a
fresh applicable Jev `ready_to_merge` decision.** This makes the requested choice
effective without pretending Jev granted a native approval.

Enforce this in the common merge admission used by `mergeOne`, `reconcileMerge`
and `observeMergeEvidence`, as well as normal publication progress. Preserve
ordered cancellation, native exact-head/base checks, deadlines and
`CheckPullMergeable(..., false)`. Invalidate acceptance on relevant head/base,
scope, policy, findings, check or correction-evidence change. Recovery must still
recognize a previously authorized merge that actually completed; it must not
launch a new mutation using stale acceptance or charge another decision merely
to observe an existing native receipt.

Persist the consumed decision ID, cycle and evidence fingerprint in immutable
`MergeWork`/merge intent. Superseding acceptance invalidates an already registered
but undriven merge using existing withdrawal ordering. Test invalidation between
intent persistence and submission, plus a lost response after actual native merge.

Confirmed merge, reachable result and original issue closure remain prerequisites
for releasing downstream work. A resolved comment, successful agent exit, closed
issue alone, or locally created merge commit cannot substitute for that evidence.

### Progress and product surfaces

Extend existing Work/Spaces read models with selected issue and selection evidence,
investigation/plan revision, dependency-ready/running/completed/blocked tasks,
lane/CLI status, checkpoint commits, PR/review links, finding dispositions, review
round and Jev decision. Expose precise waits and the next admissible action.
Use current persistent terminal streams and native Fountain review UI; no extra
workflow editor, per-task human approval queue or screenshot-based log transport.

## Early evidence gates

These gates precede broad orchestration changes and full builds. They run only
during subsequent authorized implementation. Use retained artifacts and existing
drivers in disposable native fixtures, with every write/cache/socket under
`/home`; preserve `sodaos-local` and unrelated release/Fountain work.

| Gate / task | Smallest decisive experiment | Required evidence and failure consequence |
| --- | --- | --- |
| P1 / FW02 | Inspect pinned Codex/Muse launchers and exact installed CLI capabilities; run two independent tiny tasks plus a dependent task within one admitted run. | Actual overlap; dependent release before the unrelated slow task finishes; child IDs/output/results; credential/model inheritance where applicable; serialized Git ownership; child cap and attributable usage/limit enforcement; cancellation/deadline stops all children. If unsupported, record exact interface/behavior and specify the narrow existing-executor change before FW06A. Do not claim both harnesses from evidence for one. |
| P2 / FW03 | Extend the existing native review fixture minimally to submit an added-line and a deleted-line finding, then resolve one exact thread. Read actor-specific merge blockers and exercise native merge against permissive and protected branches. | Actual native code comments and readback, stable replay receipt, no partial stale-head write, correct resolution permissions/readback. Native eligibility agrees with actual conditional merge behavior: corrected-head merge permitted when policy permits; missing required approval/status refuses. Resolve blocking-review-state assumptions before designing Jev transitions. |
| P3 / FW04 | Existing serving coordinator fixture with more than 156 unassigned ready issues and the real bounded refresh path. | Reproduce or disprove starvation across passes; complete inventory selection must eventually run without losing candidates or increasing the work cap. Current pure-selector 256-candidate coverage is insufficient. |
| P4 / FW05 | Exercise `Repository=0` product drafting without a Project launch; then the authorized repository-bound planner before issue creation, using reviewed draft outcomes rather than an accepted graph. Try representative broad and small prompts. | Concrete subject/funding/preparation/output contract without fake issue or circular accepted-graph prerequisite; broad output has cohesive feature issues and independent roots when justified; tiny output can stay one issue. A canned graph proves plumbing only. Record any required provider enrollment as an external gate, not passed semantic evidence. |

A failed assumption returns to the coordinator with the exact command/version,
observed output, affected contract and smallest proposed correction. Preserve the
failed evidence. Independent tasks may continue, but dependent broad changes wait
for the corrected boundary. Do not compensate with a full image build.

## Ordered implementation tasks

Every task below is unstarted in this plan. Dependencies mean **accepted output**,
not merely a started worker. Task-specific checks accompany each commit. Three
physical lanes execute this queue; the coordinator serializes shared contracts
and Git operations. Lane assignments may change only after explicit file/command
handoff. These implementation lanes are separate from lanes generated by the
factory for a user's issue.

Owner abbreviations: `F` = Soda `internal/factory`; `C` = `F/control`;
`S` = `internal/store`; `N` = Soda `internal/forgejo`; `P` = existing Rust
Project/host factory execution owners; `X` = Fountain canonical checkout.

| ID | Depends on | Lane | Deliverable and principal owners | Completion check |
| --- | --- | --- | --- | --- |
| FW01 | — | Coordinator | Refresh both Git states, live writers/commands and retained evidence; reconcile this baseline with later work. Establish central checklist, file reservations and representative acceptance cases. | Preserved changes attributed; no competing writer; exact current heads and task ownership recorded. No new worktree or Muse instance for an existing run. |
| FW02 | FW01 | B | P1 native executor proof in `lib/host/src/terminal/factory/{tcodex.rs,muse/,run.rs,output.rs}` and `cmd/soda-project-terminal/src/factory_roles/`. | P1 evidence for each supported harness; chosen control boundary and any required narrow extension stated. |
| FW03 | FW01 | C | P2 native inline/resolve/merge-policy proof in X's existing SDK/native-operation/review fixtures. | P2 receipts and stale/protected refusal evidence; reusable minimal implementation retained. |
| FW04 | FW01 | A | P3 complete-inventory reproduction in `C/priority_dispatch*`, `readiness_work*`, `progress*` fixtures. | Production-budget behavior established; smallest fix or reason no fix is needed recorded. |
| FW05 | FW01 | A | P4 semantic decomposition and pre-issue planner proof in `F/intent_proposal*`, `decomposition_options*`, `C/intent_*`, existing Project execution boundary. | Authorized intent-bound planner path and representative semantic outputs; funded-provider limitation explicitly separated if present. |
| FW06A | FW02 | Coordinator / A | Freeze execution subject/phase/result, immutable plan/task, selected-issue admission, child concurrency and usage contracts. Owners: `F/{assignment,assignment_result,result,planning}.go`, affected host/Project wire owners and owning guides. | Bounded validation, state transitions, exact checked-tree receipts, limits and authority agreed; unit contract tests; no dual legacy path. Release independently of native review work. |
| FW06B | FW05, FW06A | Coordinator / A | Freeze concrete intent-bound planning admission and funding on the common execution boundary; update product decomposition/selection wording. Owners: F intent/domain and existing intent/store/Project contracts. | Draft-before-association and authorized generation-before-accepted-graph are distinct; no circular admission; remove conflicting oldest-ready-first wording. |
| FW06C | FW03 | Coordinator / C | Freeze native findings/blocker-read/resolve, correction dispositions, cycle, judgment subledger and merge-acceptance contracts. Owners: `F/{review_native,judgment,assignment_result}.go`, existing store owners, X SDK and owning guides. | Native authority and failure/uncertainty policy explicit; both review evidence forms require Jev acceptance; replace unconditional correction re-review wording. Release independently of intent planning; serialize shared-file edits with FW06A. |
| FW07 | FW06A | A | Persist immutable investigated plans, current pointers and task evidence in existing PostgreSQL/store owners, primarily `S/factory_planning*`. | Original consumed plan survives replacement/restart; stale source/prerequisite/preparation bindings rejected; no new database/service. |
| FW08 | FW06B, FW11 | A | Implement intent-bound admission through the common execution path and semantic candidate production; API remains admission. Owners: `F/intent_proposal*`, `C/intent_*`, `internal/web/api/factory_intent_explore*`; coordinated P caller. | Input revision/authority/funds checked before run; semantic output assessed and structure validated; cancellation/exhaustion visible; no fake issue, accepted-graph prerequisite or publication permission. |
| FW09 | FW08 | A | Replace template splitting with candidate selection/adoption and real graph writing; preserve accepted scope and stable issue bindings. Owners: `F/decomposition_options*`, `C/intent_decomposition_judgment*`, `intent_generation*`. | Broad/small cases, uncovered outcomes, false dependencies, revision changes and replay covered; one native issue graph per accepted adoption. |
| FW10 | FW04 | A | Correct complete-inventory progress if P3 demonstrates failure; retain Jev/fairness/fresh admission. Owners: `C/priority_dispatch*`, `readiness_work*`, affected store owner if required. | More-than-cap serving fixture makes bounded progress; inventory mutation invalidates stale collection; no candidate truncation. If no defect, land only justified regression coverage. |
| FW11 | FW06A | B | Common phase/lineage admission and results, with selected-issue planning first; reuse attempt history, broker/preparation and remaining allowances. Owners: `C/dispatch_{attempt,launch,result}*`, `S/factory_{assignments,dispatch_packet,attempt_allowances}*`, P wire/role validation. FW08 adds its concrete intent subject through this same boundary. | Planning can finish without a commit or PR; cannot publish; no reset of allowance; phase/role/parent mismatch rejected. |
| FW12 | FW07, FW11 | A | Persist atomic selection-to-investigation admission, investigate and validate the plan before coding; split selection eligibility from implementation admission. Owners: `C/planning*`, readiness/dispatch join and `F/dispatch_prompt*`. | Only selected issue incurs investigation; restart reuses admitted phase; source inspection precedes planning; stale/provisional plans cannot launch code; changed inputs trigger reassessment; generic fixed steps removed. |
| FW13 | FW02, FW06A, FW11 | B | Implement the proved native child-execution boundary within one admitted run. Owners: P terminal factory, Project factory roles, host factory run/state; existing Go host/project transport as needed. | Real child IDs, results, text output and cancellation; broker/model/account inheritance; enforced child cap and attributable provider usage; no extra admitted repository sessions. |
| FW14 | FW07, FW13 | B | Execute ready tasks, persist receipts and release dependencies; enforce lane ownership, aggregate limits and sole Git integrator. Owners: existing P execution and `C`/`S` task integration. | Independent overlap; prompt downstream release; failed prerequisite blocks descendants; parent termination stops workers; task/commit evidence survives restart without duplicate writes. |
| FW15 | FW12, FW14 | A | Integrate coherent checkpoint commits and require completed plan/check evidence before existing PR publication. Owners: `F/dispatch_prompt*`, `C/publication*`, result admission. | Reported commits exist in candidate ancestry; incomplete tasks cannot publish; complete issue publishes exactly one PR despite lost replies/replay. |
| FW16 | FW03, FW06C | C | Complete generic conditional inline review primitive using existing X `sdk/review.go`, `services/nativeoperation/review.go`, `models/issues/review.go`. | Atomic native summary/code comments, valid diff anchors, permissions, exact head/base, immutable IDs and replay validated with existing native tests. |
| FW17 | FW16 | C | Complete paged native thread evidence, conditional resolution and actor-bound merge-eligibility reads using X `sdk/snapshot.go`, current readers and native review/policy services; wire N read adapters. | All pages agree on one revision; later-page/outdated unresolved findings retained; incomplete evidence blocks. Exact resolution readback and native policy blockers observable; read does not grant mutation authority. |
| FW18 | FW06C | C | Soda structured findings, receipts and immutable correction/review-cycle storage. Owners: `F/review_native*`, `N/review*`, existing publication/store records. | Stable native/factory identity mapping; bounded parsing; complete finding digest/dispositions; obsolete string-only callers replaced. |
| FW19 | FW11, FW17, FW18 | C | Correction result admission and accepted-fix/native-resolution reconciliation. Owners: `C/correction*`, correction prompts and N resolution adapter. | Every finding accounted for; incomplete/disputed evidence blocks; mechanically complete proposed fixes reach judgment before native resolution; accepted resolutions journal and read back without circular admission. |
| FW20 | FW11, FW18 | C | Durable review cycles and fresh independent re-review input. Owners: `C/publication_children*`, `review_cycle*`, `F/dispatch_prompt_stage*`. | New decision on same head creates a fresh reviewer; replay reuses its run; full outstanding finding/correction evidence supplied; distinct reviewer/corrector sessions retained. |
| FW21 | FW06C, FW18 | A | Funded Jev delivery choice, evidence fingerprint, decision persistence and uncertainty/failure policy against accepted disposition/cycle contracts. Owners: `F/judgment*`, `C/judgment_advice*`, existing chooser/store reservation owner. | Both choices exercised; clean review also requires judgment; attempt-root call subledger reserved/consumed; stale semantic evidence invalidates while expected resolution readback does not repeat the call. Integrate real correction/cycle paths in FW22. |
| FW22 | FW16, FW17, FW19, FW20, FW21 | C | Wire review/fix/Jev transitions and one common merge gate through progress, periodic merge and recovery. Owners: `C/publication_children*`, `merge*`, `N/merge_observation*`, `F/merge_operation*`. | Review-again blocks every merge path; accepted corrections reconcile resolutions before merge; immutable merge intent retains decision/cycle/fingerprint; superseded acceptance withdraws undriven intent; native blockers and completed-operation recovery remain correct. |
| FW23 | FW06A, FW06C | B | Extend existing plan/work API and frontend contracts/views; native comments and terminal streams remain primary. Owners: `internal/web/api/factory_plan_view*` and current Work/Spaces components. | Bounded API/component fixtures render phases, tasks/lanes, commits/PRs, findings/corrections and Jev/waits. Bind and prove actual persisted readback in FW25 after producers land; do not claim fixture UI proves the journey. |
| FW24 | FW12, FW14, FW19, FW20, FW22 | B | Integrate restart/cancellation/invalidation and finite allowances across the composed stages using existing fixtures. | No duplicate child/review/PR/merge, stale accepted evidence or orphan worker; restart observes receipts instead of redoing completed writes; shared attempt limits hold. |
| FW25 | FW09, FW10, FW15, FW22, FW23, FW24 | A | Extend the actual prompt-entry serving journey through the full graph and both Jev branches. Owners: existing API/controller/serving tests. | Full acceptance matrix below passes with deterministic adapters; graph is produced through real entrypoints, not injected as already accepted. |
| FW26 | FW25 | B + C | Run the smallest disposable native end-to-end factory journey using retained drivers and current built components. | Native CLI overlap/commits, actual PR/inline threads/correction, real funded judgment where available, both merge-policy outcomes and downstream release observed; exact external qualification limits recorded. |
| FW27 | FW25 | Independent coordinator review | Review released changes while FW26 runs; inspect native receipts as they arrive. Cover semantic decomposition, authority, every merge path, budgets and lane execution. Repairs return to file owner and repeat affected checks. | Source review can finish with external qualification gates recorded; native conclusions await actual evidence. No substantive unresolved source finding or active writer on reviewed files. |
| FW28 | FW26, FW27 | A / Coordinator | Complete owning docs, references and operator guidance; reconcile broader factory plan and remove absorbed completed planning material. Record focused local commit groups and remaining external gates. | User's complete loop traceable to code/evidence; native evidence independently reviewed; clean intended Git state; no push, publication, merge or protected-appliance mutation by this implementation session. External gates leave qualification open. |

### Parallel schedule and writer handoffs

These are scheduling bands, not barriers: start a task as soon as its listed
dependencies are accepted and its files are released. In particular, do not wait
for all of A's decomposition work before B/C proceed.

| Band | Lane A: intent, planning, judgment | Lane B: Project execution and integration | Lane C: native review and delivery |
| --- | --- | --- | --- |
| Establish boundaries | FW04 then FW05 | FW02 | FW03 |
| Independent contract releases | FW06A after P1; FW06B after P4 and FW06A. FW10 may proceed after FW04. | Consume FW06A without waiting for review/decomposition implementation. | FW06C after P2; native review work need not wait for P1/P4. |
| Main parallel work | FW07; FW08 → FW09 once FW11/FW06B land. Start FW12 once FW07/FW11 land; use available time for FW10. | FW11 → FW13 → FW14 as FW07 becomes available. FW23 can start from its contracts when B has capacity. | FW16; release FW18 early so A can implement FW21 while C completes FW17/FW19/FW20. |
| Integrate behavior | FW12 → FW15 when FW14 lands; FW21 starts after FW18, without waiting for completed correction orchestration. | Finish FW23 contract views; actual producer/readback integration is FW25. | FW19/FW20 → FW22 once FW21 lands. |
| Composed proof | FW25 when all its joins pass. | FW23/FW24, then FW26 with C. | Assist native FW26 after FW22; hand off delivery code for review. |
| Close | FW28 after FW26, independent FW27 and repairs; FW27 may run alongside FW26. | Release writers/resources and supply evidence. | Release writers/resources and supply evidence. |

The coordinator serializes shared-file writes for FW06A/FW06B/FW06C, while
releasing each contract separately as soon as its proof is accepted. Later edits require a named
handoff, particularly `assignment_result.go`, `dispatch_prompt*.go`,
`publication_children.go`, shared store/schema registration and `internal/web/server.go`.
The coordinator owns Git staging/commits and shared wiring; a worker must not
stage another lane's changes. A released interface includes schema, caller,
fixture and rejection behavior, not just a Go/Rust struct.

Within a lane, independent tasks may be reordered; task dependencies and file
reservations always win. Idle capacity may take a ready task from another lane
only after the previous owner releases its files, command handles and unpublished
work. No worktrees. Before any launch or resumption, inspect existing monitors and
coordinate with any active Fountain writer; this document grants no authority to
replace their work.

If implementation uses the previously requested Muse setup, preserve one
interactive coordinator with native subagents, `muse-spark-1.3-contributor`,
Contributor ON and XHIGH verified for every worker. Use the current
`muse-implementation` skill for launch/handoff. Retain the 25-minute Codex
same-chat supervision schedule, quiet during healthy progress; no recurring
five-minute Muse worker loop or external timer substitute. This planning change
does not launch a session or arm a timer.

### Reviewable commit groups

Commit small coherent verified slices as tasks finish, without waiting for the
entire group. These are grouping boundaries for later review, not authorization
to publish PRs. Cross-repository commits name their exact dependency revisions.

| Group | Tasks | Review focus |
| --- | --- | --- |
| Contracts and bounded proofs | FW02, FW04–FW05, FW06A–FW06C, FW07, FW11 | Native assumptions, authority, phase/result schema and durable plan identity. Fountain proof changes belong to its native group. |
| Prompt to native issue graph | FW08–FW10 | Semantic scope, finite planning and complete candidate selection. |
| Investigate and execute lanes | FW12–FW15 | Correct order, actual parallelism, shared allowance, checkpoints and publication gate. |
| Generic Fountain inline reviews | FW03, FW16–FW17 | Native atomicity, code anchors, permissions, snapshot and thread resolution. |
| Correction and independent rounds | FW18–FW20 | Finding completeness, fresh sessions and replay-safe cycle identity. |
| Jev delivery and merge admission | FW21–FW22 | Genuine choice, funds, stale evidence and coverage of every merge path. |
| Visible progress and composed acceptance | FW23–FW26 | Real read models, restart behavior, prompt-to-merge journey and native proof. |
| Independent repairs and documentation | FW01, FW27–FW28 | Baseline reconciliation, findings resolved, owning guides accurate, external gates precise. |

## Acceptance and verification

Use existing test drivers and the smallest affected packages. Database fixtures
use the documented disposable PostgreSQL setup (`SODA_PG_*`), not an old SQLite
design or the protected appliance. Keep test caches/output under `/home`. Native
proof uses x86_64 disposable Project/VM/Docker fixtures as appropriate; no full
ISO/outer-image build to establish a CLI or review API boundary.

| Acceptance | Existing test owner to extend |
| --- | --- |
| Real proposal entrypoint produces two independent roots, a join and a descendant; small prompt stays cohesive; scope/dependencies validated. | `TestProposalUsesReviewedCorrectionsAndRealDependencies`, `TestDecompositionCandidatesPreserveScopeAndBounds`, `TestIntentGraphSubmitsSelectedDecomposition`, API intent exploration tests. |
| Jev sees the full ready set, picks one, and selection stays live beyond the per-pass budget; fresh admission prevents stale launch. | `TestPriorityCompleteTournamentAndFiniteCalls`, `TestServingCoordinatorConsultsCompleteEligibleSetAndLaunchesFreshSelection`, production readiness-budget fixture. |
| Selection is persisted before investigation; unselected issues incur no planning run; restart preserves selection; immutable plan → valid DAG → implementation, with no stale/provisional launch. | `TestPlanningRecordsOnceAndFeedsFreshLaunch`, `TestPlanningProvisionalRecordCannotLaunch`, `TestServingCoordinatorRefreshesProvisionalPlanningForActualMergedOutput`, `TestBuildDispatchPrompt`, phase-result/store tests. |
| Independent lanes overlap; downstream work starts without unrelated-lane barrier; coherent commits appear before final integration; all children stop on cancellation. | Existing Codex script goldens, Muse supervisor-shape and Project factory-role tests, followed by P1/native executor fixture. Golden prompt text alone does not pass execution acceptance. |
| One final candidate/PR, complete tasks and ancestor-bound checkpoints; replay/lost reply cannot duplicate publication. | `TestAccountSettledRunResults`, `TestPublishPassLinksOneExactPR`, existing native publication recovery tests and `factory_attempt_allowances_test.go`. |
| Real inline comments on added/deleted lines; exact receipts; stale head, replay, unauthorized resolution, later-page and outdated findings behave correctly. Native merge-eligibility read agrees with protected mutation. | Fountain `sdk/review_test.go`, `services/nativeoperation/review_test.go`, native review/merge integration fixtures; Soda review adapter/cycle tests. |
| Separate correction agent accounts for all findings; unknown/missing/disputed/stale evidence prevents acceptance. | `C/correction_test.go`, `publication_children_test.go`, stage prompt/result tests. |
| Jev requests another review on both same and changed head; that new review can produce more findings; later Jev acceptance reaches merge. | `C/review_cycle_test.go`, `publication_children_test.go`, judgment funding/cache/failure tests. |
| Clean review needs Jev; complete proposed fixes can be judged before thread resolution; accepted resolution readback does not cause repeated paid calls. Every merge entrypoint requires applicable decision evidence; invalidation after intent persistence prevents submission. | `C/merge_test.go`, reconcile/evidence and lost-response tests, native exact-merge/protection fixtures. |
| One root merge does not unlock a two-parent join; both verified merges/closures do; join merge unlocks descendant and work continues automatically. | `TestServingAcceptedIntentCreatesPlansExecutesReviewsMergesAndUnblocks`, `TestMergeCompletionReleasesCodeDependant`, `TestCodePrereqRequiresAttributableOriginalClosure`, reopen tests. |
| Restart after each important side effect reuses receipts, consumes no invented allowance and never launches stale work. | Existing serving completion, publication/review lost-reply, cancellation and attempt-accounting fixtures. |
| Work/Spaces shows actual phases, lanes, comments, decisions and actionable waits with live CLI text. | Existing plan-view API, Work/Spaces frontend tests and bounded native installed-view check. |

The composed journey must start at the real prompt/proposal entrypoint, retain
the generated graph, execute the selected issue's investigated plan, publish and
review actual candidate code, fix findings, exercise `review_again`, then exercise
`ready_to_merge` and observe downstream release. Separate deterministic cases
cover permissive and approval-required native policy; neither can stand in for
the other. Include changed-head findings becoming stale and rechecking relevant
acceptance before applying the Jev decision.

Fixture responses establish orchestration, not semantic model quality. Native
CLI evidence establishes only the harness/version/account actually exercised.
Record source tests, native collaboration checks, actual funded-provider runs and
installed experience separately. If provider enrollment or an unpublished
component prevents final qualification, name the exact missing input and affected
acceptance; do not mark FW26/FW27 fully accepted or claim the product journey passed.
Unaffected implementation and independent review can still finish.

Completion requires the full requested behavior, relevant checks, independent
review, accurate owning docs and focused local commits. This task authorizes
planning only now. Later implementation remains subject to its actual instruction;
no push, publication or merge is implied by this document.

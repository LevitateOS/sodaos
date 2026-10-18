# Complete the software factory workflow

## Wind-down status — 18 October 2026

Implementation is stopped and remains unqualified as a complete factory. Use the [single remaining-outcome record](factory-remaining-tasks.md#consolidated-remaining-outcomes--execution-stopped) for current status and holds; the central checklist is closed evidence. Accepted R11/R12/R13/R14/R20 source work, R36 serving/event/isolation/order slices and R50 serving integration are preserved. Native/installed parents and R37/verification/feed remainder stay open; earlier helper/no-caller gap prose is historical and must not restart accepted work.

**Execution reorganization, 2026-10-17:** use
[Remaining factory work](factory-remaining-tasks.md) for the current consolidated
task boundaries, dependencies and three-lane order. The existing run's single
central checklist owns live progress and writer leases. The FW tables and lane
sequences below retain original packet requirements and design context; they are
not a second active queue. Preserve accepted work and evidence at its recorded
scope. This changes execution organization, not the frozen product contracts.

Historical planning status: **implementation was in progress; not qualified**. This plan follows the owner's
2026-10-16 request to plan all gaps in the prompt-to-delivery loop before further
implementation. It is based on read-only inspection of Soda
`6d0281e64102269fda3614c7bd77948c4845914e` and Fountain
`a4fd2221f16f181e5ec36c1b11d4181a936052e7`. No new runtime evidence is claimed.

The [frozen implementation contracts and decisive tests](factory-workflow-contracts.md)
record Codex's final design decisions, including concrete schemas, algorithms,
limits, failure/recovery rules and 64 acceptance cases, including
FW-REASSESS-002 (D13) and FW-REFINE-003 (D14–D17). Read them before assigning
any task. Spark implements those decisions; failed native assumptions return to
Codex with evidence rather than becoming worker-selected architectures.

The [remaining-work catalog](factory-remaining-tasks.md) now owns the consolidated
corrective execution order. This document retains its workflow design and original
packet requirements. The [existing factory plan](factory-implementation-plan.md)
retains its broader
source history, appliance and installed-experience acceptance. For overlapping
workflow tasks, use the consolidated catalog rather than executing both old plans.
Preserve already completed work. This is separate from the completed RCX-2
release-component work.

Product behavior belongs in [the product overview](../product/overview.md),
interfaces in [factory interfaces](../architecture/factory-interfaces.md), and
existing operator inputs in [the factory reference](../reference/factory.md).
Implementation tasks below update those owners together with their code; this
plan does not claim the proposed interfaces already exist.

## FW-REFINE-003: review, recovery and role controls

The 2026-10-16 refinement is now a complete implementation design, not a list of
open brainstorm items. D07–D09 replace mandatory clean-review judgment with the
clean/Jev acceptance union; D14 specifies one MCP implementation, seven profiles,
14 controls, instruction/task templates and production wiring; D15 specifies
bounded stall diagnosis/repair; D16 makes Test Pilot and app completion concrete.
D17 adds P6–P9 native gates and T41–T64. None is claimed implemented or qualified.

Preserve FW01–FW32 work and receipts. Read-only reconciliation for this amendment
observed Soda `0d40d3d3` and Fountain `2551d0aa2b`: completion reassessment, native
review/merge infrastructure and phase/task work are reusable; clean-review bypass,
MCP control admission, stall recovery and whole-app verification still need the
refinement tasks below. In particular, the current judgment-only merge gate must
be generalized, not deleted or rebuilt from scratch. Claimed worker completion
and helper fixtures do not replace production/native evidence.

The closed Muse run retains its historical completion/writer checklist.
FW33–FW50 remain additive requirements; this document owns their design and tests,
while the remaining-work catalog owns current dependencies and execution packets.
Preserve active work until a named safe handoff.
Any old acceptance wording amended below uses the new frozen contracts. Source
changes already committed under earlier wording are correction inputs, not a
reason to restart the run. No second Muse instance or worktree.

The monitor already reopens P1: its retained fixture launched separate per-task
CLI processes and generated its own checkpoint acknowledgement. That proves
neither native children nor trusted checkpoint release. The central checklist's
C6 audit also records the missing production `FactoryTaskCheckpoint` route and
unqualified native-child control route for the tested pins. FW33/FW34 and P8 must
produce the actual supported boundary; source helpers and historical “passed”
labels cannot release dependent native qualification. Retain those artifacts as
transport/process evidence, not as proof of the stronger contract.

The three decisions are:

1. An explicit, complete independent clear review proceeds to the same merge gate
   without a delivery Jev call. Proposed fixes, optional/disputed/repetitive
   feedback use bounded Jev routing. Uncertainty waits; it does not automatically
   buy a review. One extra discretionary review per unchanged head is the ceiling.
2. Deterministic classification distinguishes active/external waits from cycles,
   missing producers and unexplained no-ready work. One fenced repair episode
   uses complete evidence, a bounded Strategist and validated/judged graph changes.
   Retries and replacement issues cannot refill aggregate allowances.
3. One Factory MCP server in Soda exposes seven profiles: Product Lead, Architect,
   Builder, Inspector, Fixer, Strategist and Test Pilot. Run-scoped authentication,
   OS permissions, trusted native observations and durable verification connect
   every control to real progression. Explicit blocked, nothing-found and finish
   reports never substitute for observed completion.

Generative Product Lead begins after repository association; existing private
pre-repository product drafting remains. Whole-app verification binds an accepted
intent version and actual integrated artifact/checks. Empty queues do not mean a
finished app, and verification does not authorize deployment.

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
6. Explicit complete independent clear review skips delivery Jev. Otherwise Jev
   chooses useful further review or sufficient correction evidence within fixed
   bounds; uncertainty cannot launch a review. Native approval alone is not clear.
7. Fresh clean-review or Jev acceptance plus deterministic authority, checks and
   native branch policy allow an exact-head merge. Confirmed merge and attributable issue
   completion unlock dependants and atomically record a repository reassessment gate.
8. After **each** verified completion, Jev decides whether the entire current issue
   inventory needs reassessment. On `continue`, fresh selection resumes. On
   `reassess`, a bounded coding agent investigates and proposes revisions; scope,
   coverage, dependencies, protected work and native bounds are validated, Jev
   assesses the proposal, and confirmed adoption precedes the next selection.
   An accepted no-change report also resumes selection. Failures remain visible.

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
| Readiness and selection | `control/readiness.go`, `priority.go` and `priority_dispatch.go` queue independent roots, compare candidates with Jev, apply fairness and recheck admission. | Full collection restarts each pass while sharing a bounded RPC allowance. Prove suspected starvation, then implement the prescribed durable-cursor fix if reproduced. |
| Investigation and planning | `control/planning.go` selects a strategy and emits three fixed strings. | `factory/planning.go` has `Steps []string`, without inspected evidence, task dependencies, lanes or execution receipts. Replace the placeholder planning path. |
| Execution phases | Existing assignments, preparation, broker custody, attempt roots and run history support bounded work. | `dispatch_launch.go` treats non-root publication assignments as children requiring a PR; completed `result.go` requires a commit. Add explicit prepublication phase/result contracts. |
| Parallel work and commits | Project launchers start one headless CLI; repository session capacity is one. Publication already consumes an exact candidate. | Native subagent control/inheritance/cancellation is unqualified. `dispatch_prompt.go` asks for a final commit, without task checkpoints or integration evidence. |
| Reviews | Independent reviewer/coder roles and fresh run IDs exist. Exact-head conditional native review submission exists. | `factory/review_native.go`, `forgejo/review.go`, and Fountain `sdk/review.go` support body-only reviews. Findings are strings, without line anchors or native comment receipts. |
| Corrections | `control/correction.go` accepts a successful changed candidate. | No complete finding-to-fix accounting. `dispatch_prompt_stage.go` excludes prior review evidence from reviewer input. |
| Review/merge choice | `publication_children.go` deterministically reviews each new head, then corrects requested changes. | No funded Jev delivery decision. Child identity hashes publication/head/role, preventing a new review of the same head. Soda's merge observation requires exact-head approval unconditionally. |
| Completion | `merge_evidence.go`, `store/factory_merges.go`, and `prerequisites.go` verify merge plus original closure and queue readiness observation. | Preserve this path and compose it with the new stages. Existing serving journeys start with a hand-authored task graph, so do not prove prompt decomposition. |
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
Gate P4 verifies the intent-subject path fixed in
[D01–D02](factory-workflow-contracts.md#d01--execution-subjects-authority-and-admission).

Separate **eligible for issue selection** from **eligible to implement**. A ready
accepted issue can enter Jev's selection without an implementation plan. The
selected issue then gets a source-bound investigation and plan. Coding requires
that fresh validated plan. Provisional planning for blocked work may remain
advisory, but cannot skip investigation against actual merged prerequisites.
Persist selection-to-investigation admission atomically under current authority
and capacity, with its phase and attempt identity. Unselected ready issues do not incur per-issue generative investigation; D13
separately authorizes one post-completion inventory reassessment when Jev chooses it. Restart resumes the recorded choice without
duplicate spending; invalidation before planning or coding triggers reassessment.

For complete-set selection, first reproduce the source-derived starvation
concern: production refresh consumes readiness RPCs, the shared work cap is 156,
and collection restarts and scans again after advice. If confirmed, implement
the [D02 durable inventory cursor](factory-workflow-contracts.md#d02--decomposition-and-priority-algorithms),
with native/local snapshot guards and no second full native scan. Keep complete-set
comparison, finite work limits, invalidation on inventory change and a fresh
selected-issue admission check. Never truncate candidates to fit the budget.

### Investigated plan and execution records

Extend current factory records in place, with one canonical definition per
boundary. Replace obsolete experimental forms and callers together; no legacy
reader or compatibility migration is required for this unreleased implementation.

| Record | Required content and validation |
| --- | --- |
| Execution subject and phase | Intent planning, issue attempt or D13 completion reassessment; phase independent of account role; explicit parent/run identity, accepted-input and preparation binding. Keep `soda-coder`/`soda-reviewer` roles. |
| Investigation | Inspected source references at a known revision, findings, affected owners/interfaces, requirements and unresolved assumptions. Unresolved mandatory facts block implementation. |
| Immutable plan revision | Accepted scope/input digest, source/prerequisite/preparation identities, investigation, approach, alternatives considered, verification, tasks and lanes. Keep the exact revision consumed by each run; a mutable current pointer does not replace history. |
| Task | Unique ID, objective, expected output, acceptance coverage, dependencies, lane, exclusive paths/interfaces, checks and checkpoint expectation. Enforce bounded sizes, known references, a DAG and no concurrent conflicting ownership. |
| Task execution receipt | Attempt/plan/task/lane/run identities, state, consumed prerequisite output digests, actual start/completion, check results and produced commits. Only validated successful prerequisites release a dependent task. |
| Phase result | Planning returns an investigated plan; implementation/correction returns a candidate and evidence; review returns findings; reassessment returns a bounded proposal. A planning result cannot be interpreted as publishable code. |

One bounded run performs investigation, approach selection and task planning
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
configured time/concurrency limit. A time/concurrency bound must not be described as a
monetary limit where the provider exposes no such enforcement.

Delivery judgments need a new finite **call subledger** under the attempt root:
the existing `AttemptAllowance` has active minutes and correction cycles, not
classifier credits. Its grant is part of the sponsor's admitted attempt terms,
with reserved/consumed counts and immutable request identities. Neither a phase
change nor another review refills it; missing or exhausted grants block further
judgments. Reuse the existing reservation transaction pattern and store owner.

Use the [D05 native task executor](factory-workflow-contracts.md#d05--native-children-scheduling-and-checkpoints)
inside one admitted issue run after P1 proves its supported native boundary.
The interface must expose child start, observable identity/progress, join,
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
cannot meet the fixed contract, stop dependent work and return the exact failed
assumption to Codex. Spark cannot choose a replacement execution architecture.
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
unexpected content/policy/head changes invalidate it. Disputes reach the bounded
D08 choice before another pass; a demonstrated unresolved defect cannot be waived
and unresolved disagreement ultimately requires owner attention.

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
| Concrete defects lack fixes or required code checks fail | Dispatch bounded correction when fixable; otherwise name the wait. Complete proposed fixes can proceed to judgment before thread resolution. |
| Optional-only, disputed or repetitive findings | Jev assesses material risk and specific benefit before more coding/review; actionable disputes cannot be waived. |
| Native policy requires another approval | Obtain the required independent review; Jev cannot waive this gate. |
| Evidence/authority is incomplete, checks are pending, or funds are exhausted | Wait with the exact reason; no merge acceptance. |
| Explicit complete independent clear report, all carried actionable findings verified | Record clean-review acceptance without a delivery judgment; preserve checks/native policy/D13. |
| Complete proposed correction evidence without a later clear review, native policy permits the path | Ask Jev to choose `review_again` or `ready_to_merge`; incomplete evidence cannot be accepted. |
| Valid `review_again` | Persist the decision/cycle, dispatch a fresh reviewer, then reconsider after its result. |
| Valid `ready_to_merge` | Record candidate-bound Soda acceptance, reconcile accepted finding resolutions, then recheck all deterministic/native conditions before merge. |

Insufficient-confidence or tied advice waits with `judgment_attention`; it does
not purchase a review. A positive further-review decision must also fit total and
unchanged-head limits. Exact limits and thresholds are fixed in
[D04](factory-workflow-contracts.md#d04--fixed-limits-and-accounting) and
[D08](factory-workflow-contracts.md#d08--native-merge-eligibility-and-jev-policy).
Missing/malformed/provider-failed answers cannot create acceptance. Do not silently
repeat a consumed request. FW06C implements the frozen policy and failure accounting.

Replace Soda's mandatory fresh approval after every correction with two supported
evidence forms: explicit complete independent clear review, or prior independent
findings plus complete candidate-bound correction evidence accepted by Jev.
Persist the D07 `DeliveryAcceptance` union. Both use identical authority,
freshness, check, thread-resolution and native protection gates; neither invents
a native approval.

Enforce this in the common merge admission used by `mergeOne`, `reconcileMerge`
and `observeMergeEvidence`, as well as normal publication progress. Preserve
ordered cancellation, native exact-head/base checks, deadlines and
`CheckPullMergeable(..., false)`. Invalidate acceptance on relevant head/base,
scope, policy, findings, check or correction-evidence change. Recovery must still
recognize a previously authorized merge that actually completed; it must not
launch a new mutation using stale acceptance or charge another decision merely
to observe an existing native receipt.

Persist the acceptance kind/ID, originating receipt/decision, cycle and fingerprint in immutable
`MergeWork`/merge intent. Superseding acceptance invalidates an already registered
but undriven merge using existing withdrawal ordering. Test invalidation between
intent persistence and submission, plus a lost response after actual native merge.
Use [D09's bounded authorization generations](factory-workflow-contracts.md#d09--merge-authorization-generations-and-completion)
so a confirmed cancelled operation can be replaced under a new decision without
stranding the existing PR. Uncertain or committed effects cannot be replaced.

Confirmed merge, reachable result and original issue closure remain prerequisites
for releasing downstream work. A resolved comment, successful agent exit, closed
issue alone, or locally created merge commit cannot substitute for that evidence.
D13 then fences new issue admission until the completion's Jev decision and any
required proposal adoption are resolved.

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
| P1 / FW02 | Inspect pinned Codex/Muse launchers and exact installed CLI capabilities; run two independent tiny tasks plus a dependent task within one admitted run. | Prove D05 native control, overlap, prompt dependent release, profile/custody inheritance, checkpoint correctness, limits and retirement. If unsupported, stop dependent tasks and return evidence to Codex; no implementer-chosen fallback. Do not claim both harnesses from one proof. |
| P2 / FW03 | Extend the existing native review fixture minimally to submit an added-line and a deleted-line finding, then resolve one exact thread. Read actor-specific merge blockers and exercise native merge against permissive and protected branches. | Actual native code comments and readback, stable replay receipt, no partial stale-head write, correct resolution permissions/readback. Native eligibility agrees with actual conditional merge behavior: corrected-head merge permitted when policy permits; missing required approval/status refuses. Resolve blocking-review-state assumptions before designing Jev transitions. |
| P3 / FW04 | Existing serving coordinator fixture with more than 156 unassigned ready issues and the real bounded refresh path. | Reproduce or disprove starvation across passes; complete inventory selection must eventually run without losing candidates or increasing the work cap. Current pure-selector 256-candidate coverage is insufficient. |
| P5 / FW29 | Extend the existing native snapshot family with complete paged open-issue enumeration; use retained native graph fixtures for atomic stale/oversize refusal, preserved records/edges and exact receipt replay. | Native boundary facts precede FW31/FW32 consumers. Soda acceptance identity and adoption recovery are implemented/proved in FW32/T37, not prerequisites of their own implementation. |
| P4 / FW05 | Exercise `Repository=0` product drafting without a Project launch; then the authorized repository-bound planner before issue creation, using reviewed draft outcomes rather than an accepted graph. Try representative broad and small prompts. | Concrete subject/funding/preparation/output contract without fake issue or circular accepted-graph prerequisite; broad output has cohesive feature issues and independent roots when justified; tiny output can stay one issue. A canned graph proves plumbing only. Record any required provider enrollment as an external gate, not passed semantic evidence. |

A failed assumption returns to Codex with the exact command/version,
observed output and affected frozen contract. Preserve the
failed evidence. Independent tasks may continue, but dependent broad changes wait
for the corrected boundary. Do not compensate with a full image build.

## Ordered implementation tasks

The original FW packet definitions below are preserved for requirement traceability.
Execute their remaining portions through the [current catalog](factory-remaining-tasks.md#task-list),
not by replaying this historical order or its lane schedules.

The active run's single central checklist owns completion and writer status; this
plan defines dependencies and acceptance, not a second progress ledger. Preserve
accepted work and the running coordinator when incorporating FW-REASSESS-002.
Dependencies mean **accepted output**,
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
| FW04 | FW01 | A | P3 complete-inventory reproduction in `C/priority_dispatch*`, `readiness_work*`, `progress*` fixtures. | Production-budget behavior established; evidence accepts or challenges the prescribed D02 cursor fix. |
| FW05 | FW01 | A | P4 semantic decomposition and pre-issue planner proof in `F/intent_proposal*`, `decomposition_options*`, `C/intent_*`, existing Project execution boundary. | Authorized intent-bound planner path and representative semantic outputs; funded-provider limitation explicitly separated if present. |
| FW06A | FW02 | Coordinator / A | Implement frozen D01/D03–D05 execution subject/phase/result, plan/task, selection, profile and allowance types, including the D13 completion subject/reassess union and ReassessmentMinutes; update owning guides. Owners: `F/{assignment,assignment_result,result,planning,allowance,grants}.go`, host/Project wires. | T03/T11/T16 contract validation; exact tree/event/limit shapes; no dual legacy path. Release independently of native review work. |
| FW06B | FW05, FW06A | Coordinator / A | Implement frozen D01/D02 intent-subject, command, graph-output dependency and adoption contracts; update product wording. Owners: F intent/domain and existing intent/store/Project contracts. | T01/T02/T04 contract cases; draft authorization distinct from accepted graph; remove conflicting oldest-ready-first wording. |
| FW06C | FW03 | Coordinator / C | Implement frozen D04/D06–D09 findings/blocker/resolve, correction/cycle, call subledger and merge-generation contracts. Owners: `F/{review_native,judgment,assignment_result,publication,merge_operation,grants}.go`, store and X SDK. | T19/T21/T24/T28 contract cases; explicit resolve binding, limits and authority; replace unconditional re-review wording. Release independently; serialize shared files with FW06A. |
| FW07 | FW06A | A | Persist immutable investigated plans, current pointers and task evidence in existing PostgreSQL/store owners, primarily `S/factory_planning*`. | Original consumed plan survives replacement/restart; stale source/prerequisite/preparation bindings rejected; no new database/service. |
| FW08 | FW06B, FW11 | A | Implement intent-bound admission through the common execution path and semantic candidate production; API remains admission. Owners: `F/intent_proposal*`, `C/intent_*`, `internal/web/api/factory_intent_explore*`; coordinated P caller. | Input revision/authority/funds checked before run; semantic output assessed and structure validated; cancellation/exhaustion visible; no fake issue, accepted-graph prerequisite or publication permission. |
| FW09 | FW08 | A | Replace template splitting with candidate selection/adoption and real graph writing; preserve accepted scope and stable issue bindings. Owners: `F/decomposition_options*`, `C/intent_decomposition_judgment*`, `intent_generation*`. | Broad/small cases, uncovered outcomes, false dependencies, revision changes and replay covered; one native issue graph per accepted adoption. |
| FW10 | FW04 | A | Implement D02 durable inventory cursor if P3 demonstrates failure; retain Jev/fairness/fresh admission. Owners: `C/priority_dispatch*`, `readiness_work*`, `S/factory_priority_inventory.go`. | T08/T09: bounded progress, partial prerequisites, restart/invalidation; no second full scan or candidate truncation. If disproved, return evidence to Codex before removing the fix. |
| FW11 | FW06A | B | Common phase/lineage admission and results, with selected-issue planning first; reuse attempt history, broker/preparation and remaining allowances. Owners: `C/dispatch_{attempt,launch,result}*`, `S/factory_{assignments,dispatch_packet,attempt_allowances}*`, P wire/role validation. FW08 adds its concrete intent subject through this same boundary. | Planning can finish without a commit or PR; cannot publish; no reset of allowance; phase/role/parent mismatch rejected. |
| FW12 | FW07, FW11 | A | Persist atomic selection-to-investigation admission, investigate and validate the plan before coding; split selection eligibility from implementation admission. Owners: `C/planning*`, readiness/dispatch join and `F/dispatch_prompt*`. | Only selected issue incurs investigation; restart reuses admitted phase; source inspection precedes planning; stale/provisional plans cannot launch code; changed inputs trigger reassessment; generic fixed steps removed. |
| FW13 | FW02, FW06A, FW11 | B | Implement D05 manifest/native child-control and typed-event boundary within one admitted run. Owners: P terminal factory, Project factory roles, host run/state and existing Go host/project transport. | T12/T13/T16: native IDs/events/output/cancellation, exact profile/account inheritance, child cap; no extra admitted repository session. |
| FW14 | FW07, FW13 | B | Execute ready tasks, persist receipts and release dependencies; enforce lane ownership, aggregate limits and sole Git integrator. Owners: existing P execution and `C`/`S` task integration. | Independent overlap; prompt downstream release; failed prerequisite blocks descendants; parent termination stops workers; task/commit evidence survives restart without duplicate writes. |
| FW15 | FW12, FW14 | A | Integrate coherent checkpoint commits and require completed plan/check evidence before existing PR publication. Owners: `F/dispatch_prompt*`, `C/publication*`, result admission. | Reported commits exist in candidate ancestry; incomplete tasks cannot publish; complete issue publishes exactly one PR despite lost replies/replay. |
| FW16 | FW03, FW06C | C | Complete generic conditional inline review primitive using existing X `sdk/review.go`, `services/nativeoperation/review.go`, `models/issues/review.go`. | Atomic native summary/code comments, valid diff anchors, permissions, exact head/base, immutable IDs and replay validated with existing native tests. |
| FW17 | FW16 | C | Complete paged native thread evidence, conditional resolution and actor-bound merge-eligibility reads using X `sdk/snapshot.go`, current readers and native review/policy services; wire N read adapters. | All pages agree on one revision; later-page/outdated unresolved findings retained; incomplete evidence blocks. Exact resolution readback and native policy blockers observable; read does not grant mutation authority. |
| FW18 | FW06C | C | Soda structured findings, receipts and immutable correction/review-cycle storage. Owners: `F/review_native*`, `N/review*`, existing publication/store records. | Stable native/factory identity mapping; bounded parsing; complete finding digest/dispositions; obsolete string-only callers replaced. |
| FW19 | FW11, FW17, FW18 | C | Correction result admission and accepted-fix/native-resolution reconciliation. Owners: `C/correction*`, correction prompts and N resolution adapter. | Every finding accounted for; incomplete/disputed evidence blocks; mechanically complete proposed fixes reach judgment before native resolution; accepted resolutions journal and read back without circular admission. |
| FW20 | FW11, FW18 | C | Durable review cycles and fresh independent re-review input. Owners: `C/publication_children*`, `review_cycle*`, `F/dispatch_prompt_stage*`. | New decision on same head creates a fresh reviewer; replay reuses its run; full outstanding finding/correction evidence supplied; distinct reviewer/corrector sessions retained. |
| FW21 | FW06C, FW18 | A | Funded Jev delivery choice, evidence fingerprint, decision persistence and uncertainty/failure policy against accepted disposition/cycle contracts. Owners: `F/judgment*`, `C/judgment_advice*`, existing chooser/store reservation owner. | Both choices exercised; explicit clear review bypasses delivery judgment under D07; attempt-root call subledger reserved/consumed; stale semantic evidence invalidates while expected resolution readback does not repeat the call. Integrate real correction/cycle paths in FW22. |
| FW22 | FW16, FW17, FW19, FW20, FW21 | C | Implement D07–D09 transitions, common merge gate and bounded authorization generations. Owners: `C/publication_children*`, `merge*`, `N/merge_observation*`, `F/merge_operation*`, `S/factory_merges*`. | T25–T29: expected resolution reconciliation, every merge entrypoint gated, cancelled authorization replaceable on same PR, uncertain/committed predecessor never replaced. |
| FW29 | FW01 | C | P5 narrow native proof: extend generic repository_issues snapshot family and prove complete actor-bound pagination; reuse graph fixtures for frozen edges, packet bounds and receipt replay. Owners: X SDK/snapshot/native graph; N snapshot adapter. | T34 and T37 native packet facts established before consumers; Soda acceptance/adoption assertions remain FW32; small fixture, no full build. Return failure to Codex; preserve existing P1–P4 receipts. |
| FW30 | FW06A, FW06B | Coordinator / A | Freeze D13 domain/store/UI contracts in code: completion event/cycle/report, two-call/one-run ledger, Retry identity and repository generation; add schema registration after exclusive handoff. Owners: F reassessment records, grants, S schema/completion owners. | T33/T35 contracts; completed issue root cannot reopen; typed scope/coverage/limits; source callers updated together. |
| FW31 | FW10, FW22, FW29, FW30 | A | Persist completion/fence atomically; complete inventory cursor, funded Jev continue/reassess decision, freshness and dispatch gate in existing progress/priority/packet owners. | T33–T35/T39: exact-once per completion, full inventory, no subset election/global stall, no early admission, charged failure and explicit Retry. |
| FW32 | FW09, FW11, FW29, FW30 | B | Admit one completion-subject coder; validate full report and Jev proposal; stage and apply serial per-intent graph revisions, preserving protected bindings/acceptances and enforcing journal-aware recovery. Owners: C intent/adoption/dispatch, S intent generation, F result and reassess-only prompt bound across Go/Rust/CLI; shared N graph adapter by handoff. | T35–T39: no source/publication effects, no scope inflation, unchanged/delivered acceptance survives, partial prefix/restart/lost response safe, confirmed no-change/adoption releases. |
| FW23 | FW06A, FW06C, FW30 | B | Extend existing plan/work API and frontend contracts/views; native comments and terminal streams remain primary. Owners: `internal/web/api/factory_plan_view*` and current Work/Spaces components. | Bounded API/component fixtures render phases, tasks/lanes, commits/PRs, findings/corrections and Jev/waits, including D13 completion gates/proposals/adoption. Bind and prove actual persisted readback in FW25 after producers land; do not claim fixture UI proves the journey. |
| FW24 | FW12, FW14, FW19, FW20, FW22, FW31, FW32 | B | Integrate restart/cancellation/invalidation and finite allowances across the composed stages using existing fixtures. | No duplicate child/review/PR/merge, stale accepted evidence or orphan worker; restart observes receipts instead of redoing completed writes; shared attempt limits hold. |
| FW25 | FW09, FW10, FW15, FW22, FW23, FW24 | A | Extend the actual prompt-entry serving journey through the full graph, both delivery choices and all D13 reassessment branches. Owners: existing API/controller/serving tests. | Full acceptance matrix below passes with deterministic adapters; graph is produced through real entrypoints, not injected as already accepted. |
| FW26 | FW25 | B + C | Run the smallest disposable native end-to-end factory journey using retained drivers and current built components. | Native CLI overlap/commits, actual PR/inline threads/correction, real funded judgment where available, both merge-policy outcomes, D13 continue/revise/no-change branches and subsequent selection observed; exact external qualification limits recorded. |
| FW27 | FW25 | Independent coordinator review | Review released changes while FW26 runs; inspect native receipts as they arrive. Cover semantic decomposition, authority, every merge path, budgets and lane execution. Repairs return to file owner and repeat affected checks. | Source review can finish with external qualification gates recorded; native conclusions await actual evidence. No substantive unresolved source finding or active writer on reviewed files. |
| FW28 | FW26, FW27 | A / Coordinator | Complete owning docs, references and operator guidance; reconcile broader factory plan and remove absorbed completed planning material. Record focused local commit groups and remaining external gates. | User's complete loop traceable to code/evidence; native evidence independently reviewed; clean intended Git state; no push, publication, merge or protected-appliance mutation by this implementation session. External gates leave qualification open. |

### FW-REFINE-003 dependency-ordered tasks

Retain the baseline checklist and evidence. Before FW33, coordinator records the
current commits, command owners and which baseline prerequisites actually pass.
Unqualified P1/native assumptions remain open even if an earlier worker marked a
task done. P6/P7/P8 are small native gates, not permission for a broad rewrite.
FW35 can proceed independently of MCP; transport work cannot block a released
review correction. No new source implementation is claimed by this plan amendment.

| ID | Depends on accepted output | Lane | Deliverable / exact owners | Completion evidence |
| --- | --- | --- | --- | --- |
| FW33 | FW01; released P launcher files | B | P6 fixed stdio relay/private IPC directory/socket/trusted instruction-channel proof for exact admitted harnesses; smallest SDK integration in existing P + `internal/web/api` fixture. | T47; pin SDK/harness/config bytes, actual profile and commands; unsupported profiles stay unavailable. |
| FW34 | FW33 | B | P7 native caller/worker attenuation and OS permission proof; narrowly implement required P mount/config/credential protections. | T49/T50; native forbidden writes/calls fail, permitted checks/scratch work; no shared-home token assumption. |
| FW35 | FW18, FW20; released shared domain files | C | D06 complete finding digest, canonical dispositions and D07 DeliveryAcceptance union; exact clear-report validation and storage in F review/result/publication, C correction, S publication/review. | T41–T43 domain/store tests; no key-only authorization or rewritten defects. Release union to FW37 without waiting for transport. |
| FW36 | FW35, FW21 | C | D08 routing and finite unchanged-head discretionary counter in C review cycles/publication children/judgment and S allowance reservations. | T44; positive evidence needed to buy review, uncertainty waits, optional advice cannot force correction, all charges idempotent. |
| FW37 | FW36, FW17, FW22 | C | Apply union to conditional resolution, all merge registration/reconcile/progress paths and S merge generations; preserve already committed effects. | T45/T46 on real PG and retained native policy fixture; zero-delivery-call clean route and both variants through every serving entry. |
| FW38 | FW33, FW06A; shared schema/registration handoff | A | D14 canonical profile/tool DTOs, D16 verification-manifest proposal/catalog-reference contract, bounded listener, capability admission, immutable submissions and receipt recovery. Owners F/C/S + `internal/web/api`, construction in `internal/web`/`cmd/soda-dashboard`; use pinned SDK. | T47/T48/T51 handler→controller→PG; unsupported tools fail server-side; listener lifetime/restart reconcile; manifest contract released before Product Lead adoption. |
| FW39 | FW34, FW38, FW11 | B | Production fixed relay and capability delivery/config staging in P; protected instruction/profile/readonly execution binding; native parent/child stop and receipt identity. | T47–T51 through actual Project namespace/start marker; no secrets in prompt/argv/log or broad socket mount. |
| FW40 | FW38, FW39, FW09, FW12 | A | D14 common reads/incomplete completion + Product Lead/Architect prompts and typed result ingress; dispatch/settlement/store/view consumers. Include D16 manifest proposal validation, graph-choice coverage, accepted-intent adoption/digest persistence now; FW45 consumes it. Remove authoritative fenced-result ingress. | T51/T52/T54; real repository-bound graph/manifest and plan admission; no-repo draft launches nothing; blocked receipt names cause and observed retirement. |
| FW42 | FW35, FW39, FW14, FW15 | B | Builder prompt/output/handoff controls and trusted checkpoint integration; P8 genuine native overlap/receipt/release. Includes single-task Fixer checkpoint reuse with FW41 handoff. | T53 and repeated T12–T16; no synthetic acknowledgement or separate per-task CLIs; failed child/check/cancel cannot release descendants. |
| FW41 | FW35, FW38, FW39, FW42, FW19 | C | Inspector/Fixer profiles/prompts, typed reviews/dispositions and lifecycle consumers into native review/candidate operations; no direct MCP native credentials. | T42/T51/T52/T54 with real handler→conditional review/comment receipts; explicit clear not inferred from exit/approval. |
| FW43 | FW38, FW30, FW31; completion/progress/store handoff | A | D15 deterministic wait/stall detector, subject/episode/fence, explicit repair sponsor/limits and stable intent active-time aggregate. Owners F assignment/allowance, C progress/readiness/completion admission, S packet/accounting + policy command wiring. | T55–T57/T60 on real serving/store path; no model call for ordinary waits; retries/splits do not refill; unrelated repos progress. |
| FW44 | FW43, FW32, FW39, FW40 | A | Strategist reassess/repair prompts/control and result consumer; bounded typed proposal Jev request, D13 serial graph adoption/readback reuse and fresh release. | T52/T54/T57–T59; pre-completion deadlock recovery, unchanged evidence reuse, protected records and interrupted prefix preserved. |
| FW45 | FW39, FW40, FW42, FW44 | B | D16 delivery subject/manifest/outcome map, Test Pilot prompt/control, trusted check/artifact receipts, finite verification/repair admission and P9. F/C/S/P shared files require handoff from A. | T52/T54/T60–T62; exact version verified, empty queue not complete, missing fixture explicit, no deployment; native P9 gate. |
| FW46 | FW37, FW40, FW41, FW42, FW44, FW45 | C | Wire actual producers to existing intent/attempt/Work/Spaces API/frontend: seven profiles, recorded vs accepted receipts, review cost, blocked owner, stalls and app verification. | T63 and real-readmodel portions of T54; no fake UI status/parallel dashboard, reads spend nothing. Design work may start against released contracts earlier. |
| FW47 | FW37, FW40, FW41, FW42, FW44, FW45, FW46, FW24 | A | Extend existing serving journey/real PG fixtures, restart/cancel/authority/race matrix for full MCP ingress and all three refinements. | T41–T64 deterministic assertions at production entrypoints; no injected accepted graph/acceptance short circuit. Record relevant baseline failures separately. |
| FW48 | FW47; P6–P9 actually passed | B + C, coordinator assigns one fixture owner | Small disposable native journey using retained drivers, real native effects and exact pinned profiles; separately funded first-response semantic cases if authorized inputs exist. | T46/T64, actual A/B/C overlap, clean zero-call merge, correction choice, stall repair and integrated verification; precise external gates otherwise. |
| FW49 | FW47 | Independent coordinator review | Read production wiring, auth, budgets, all merge entries, protected graph adoption and result/checkpoint retirement. Review FW48 receipts when available. Return fixes to sole file owner. | No unresolved substantive source/design finding; native claims independently match receipts. Can run alongside FW48. |
| FW50 | FW48, FW49 | Coordinator | Reconcile owning docs, completion table and exact remaining external qualification gates; focused local commits and safe writer/resource release. | All applicable tests/evidence traced; incomplete native inputs explicitly keep qualification open. No push/publish/merge. |

### Refinement lanes and explicit handoffs

These are dependency releases, not synchronized waves. Initially C can implement
FW35–FW37 while B proves FW33–FW34; A prepares contract fixtures and starts FW38
as soon as P6 releases. B consumes FW38 to finish FW39. A then runs FW40 → FW43 →
FW44; C starts review adapter work after its contracts and completes FW41 after the
Fixer checkpoint release; B runs FW42, then FW45 after A's
shared delivery/adoption handoff. C integrates views in FW46; A owns composed
serving tests FW47. FW48 native qualification and FW49 independent review overlap.
Idle capacity may take a ready task only after the central checklist records its
new writer/file/command ownership.

The coordinator serializes shared edits to `assignment*.go`, `result.go`,
`dispatch_prompt*.go`, host/Project wire types, schema registration and
`internal/web/server.go`. A owns capability/submission/store admission; C receives
only review/correction consumer files. B owns P relay/permissions/checkpoint files;
Go bridge edits require a named handoff from A. FW43 may use released F/S allowance
files while B works in Rust. FW45 receives shared F/C/S delivery files from A only
after FW44 checkpoint/commands settle. FW46 receives readmodel producer contracts
and released API files, not ownership of active controller files. Ongoing FW24
recovery commands/writers must release overlapping dispatch/merge owners first.

Every release names commit/diff, source/test receipts, exact files, live processes,
remaining facts and next owner. The coordinator stages only released owned files
and makes early focused local commits: review semantics/digests; review routing;
merge union; transport proof; server admission/receipts; native permissions;
role adapters/checkpoints; stall classification/accounting; repair/adoption;
verification; real views/composed tests; independent fixes/docs. Do not bundle
unrelated identity, release, tools or Fountain work into these commits.

Final completeness requires each D14 wiring-table row and T41–T64, not merely a
tool appearing in discovery. Preserve the one Muse coordinator and exact native
worker profile. Do not add the retired five-minute loop. A supervision schedule
may be described as active only after its actual saved state is verified; an
unarmed/paused 25-minute relay is not active supervision.

### Baseline parallel schedule and writer handoffs

These are scheduling bands, not barriers: start a task as soon as its listed
dependencies are accepted and its files are released. In particular, do not wait
for all of A's decomposition work before B/C proceed.

| Band | Lane A: intent, planning, judgment | Lane B: Project execution and integration | Lane C: native review and delivery |
| --- | --- | --- | --- |
| Establish boundaries | FW04 then FW05 | FW02 | FW03; FW29 can follow now using released native owners. |
| Independent contract releases | FW06A after P1; FW06B after P4 and FW06A. FW10 may proceed after FW04; FW30 follows FW06A/FW06B. | Consume FW06A without waiting for review/decomposition implementation. | FW06C after P2; native review work need not wait for P1/P4. |
| Main parallel work | FW07; FW08 → FW09 once FW11/FW06B land. Start FW12 once FW07/FW11 land; use available time for FW10. | FW11 → FW13 → FW14 as FW07 becomes available. FW23 can start from its contracts when B has capacity. | FW16; release FW18 early so A can implement FW21 while C completes FW17/FW19/FW20. |
| Integrate behavior | FW12 → FW15 when FW14 lands; FW21 starts after FW18, without waiting for completed correction orchestration. | Finish FW23 contract views; actual producer/readback integration is FW25. | FW19/FW20 → FW22 once FW21 lands. |
| Post-completion integration | FW31 after FW10/FW22/FW29/FW30. | FW32 after FW09/FW11/FW29/FW30; release shared intent owners explicitly. | Release FW22 completion/store owners to FW31; assist only through named handoffs. |
| Composed proof | FW25 when all its joins pass. | FW23/FW24 after FW31/FW32, then FW26 with C. | Assist native FW26 after FW22; hand off delivery code for review. |
| Close | FW28 after FW26, independent FW27 and repairs; FW27 may run alongside FW26. | Release writers/resources and supply evidence. | Release writers/resources and supply evidence. |

The coordinator serializes shared-file writes for FW06A/FW06B/FW06C/FW30, while
releasing each contract separately as soon as its proof is accepted. Later edits require a named
handoff, particularly `assignment_result.go`, `dispatch_prompt*.go`,
`publication_children.go`, shared store/schema registration and `internal/web/server.go`. FW29 snapshot files overlap FW17; FW31 merge/packet
files overlap FW22/FW11; FW32 intent/adoption files overlap FW09/FW06B. A reservation
must name the releasing and receiving worker plus any live command before transfer.
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
five-minute Muse worker loop or external timer substitute. This amendment continues the existing run; do not launch another session or arm
a Muse worker timer. Codex sends the frozen amendment through the verified parent
TUI route and verifies acknowledgment plus central-checklist incorporation before
affected workers resume. Preserve safe checkpoints and existing commit groups.

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
| Post-completion reassessment | FW29–FW32 | Full repository inventory, completion fence, bounded Jev/agent process, preservation and recoverable issue revisions. Native enumeration is a separate Fountain commit. |
| Visible progress and composed acceptance | FW23–FW26 | Real read models, restart behavior, prompt-to-merge journey and native proof. |
| Independent repairs and documentation | FW01, FW27–FW28 | Baseline reconciliation, findings resolved, owning guides accurate, external gates precise. |

The FW-REASSESS-002 handoff adds four tasks without renumbering existing IDs.
FW29 is the next independent native gate; FW30 requires the shared contract release.
FW31 and FW32 then run in parallel after their exact dependencies, before composed
acceptance. Do not restart accepted tasks solely because this requirement was added.

## Acceptance and verification

The [D12 decisive test list](factory-workflow-contracts.md#d12--decisive-test-list-and-completion-mapping)
fixes T01–T40; [D17](factory-workflow-contracts.md#d17--refinement-proof-gates-and-decisive-cases)
adds T41–T64 and their task/suite mappings. The matrix below locates existing
entrypoints; it does not replace the more specific assertions in D12.

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
| Explicit clear review bypasses Jev; complete proposed fixes can be judged before thread resolution; expected resolution readback adds no paid calls. Every merge entrypoint requires applicable clean/Jev acceptance; invalidation after persistence prevents submission. | `C/merge_test.go`, reconcile/evidence and lost-response tests, native exact-merge/protection fixtures. |
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

# Factory workflow: frozen implementation contracts

Design decision owner: **Codex coordinator**, 2026-10-16. Status: design frozen;
implementation and native qualification remain pending. This is the implementation
contract and decisive test list for the [workflow plan](factory-workflow-plan.md),
not a description of already shipped interfaces. It will be absorbed into the
owning guides when implementation finishes.

Spark implements these decisions. It may choose local variable names, organize
small private helpers within the named owners, and correct mechanical compilation
errors. It must return evidence to Codex before changing an authority boundary,
record/wire contract, state transition, algorithm, limit, dependency, package
owner, native primitive or required check. A failed proof is a blocked assumption,
not permission to substitute an architecture. No design choice below is deferred
to an implementation worker.

## D01 — Execution subjects, authority and admission

Reuse `factory.Assignment`, `Run`, `AttemptAllowance`, reservations, broker custody
and `RecordDispatchPacket`. Add the following canonical types in
`internal/factory/assignment.go`; repository and Project stay on the assignment:

```go
type ExecutionSubject struct {
    Kind   string         // "issue", "intent", "completion"; exactly one payload
    Issue  *IssueSubject
    Intent *IntentSubject
    Completion *CompletionSubject // D13 completion/cycle/inventory binding
}
type IssueSubject struct {
    Issue      int64
    Acceptance string
}
type IntentSubject struct {
    IntentID    string
    RevisionID  string
    ScopeDigest string
    OwnerID     int64
}
```

Use snake_case JSON tags. Subject keys are `issue:<repository>:<issue>` and
`intent:<intent_id>`; the latter's owner is validated from the authoritative row,
not accepted from JSON. Extend existing assignment/allowance SQL with a subject
key and nullable issue column. Do not use zero as a fake native issue. Keep one
unfinished phase assignment per subject and one admitted repository session
across all three kinds. Retain one allowance root across revisions of the same active
attempt; immutable subject snapshots on individual assignments name exact inputs.

`Assignment.Phase` is one of `decompose`, `investigate`, `implement`, `review`,
`correct`, `reassess`. Only `decompose` accepts an intent subject; only
`reassess` accepts the D13 completion subject. The other phases require an issue. Only `review` uses the
reviewer account; all other phases use the coder account. Add `ParentAssignment`;
root has no parent, children keep `AttemptRoot`. `PublicationAssignment` is absent
until implementation establishes the publication owner. Do not infer phase from
role, parent equality or whether a PR happens to exist.

Add `plan-intent` to the current command ledger. Reuse the existing Generate work
UI action/route to submit `{command_id,intent,expected_revision,scope_digest}`;
respond 202 with the recorded planning state, and expose progress through the
existing generation GET. Same command/content replays; changed content conflicts.
Do not add another confirmation screen.

- Before repository association, save/display a private editable draft only.
- After association, the command can authorize planning of a **draft** whose
  effective ideas are accepted/edited and have valid attribution. Material
  undecided ideas cannot enter its effective subset. Tasks need not exist yet.
- `SavedReview` is presentation/provenance, never spending or execution authority.
- Admission checks current human repository authority, intent head/effective
  scope, repository policy, operator/environment/sponsorship grants, concrete
  Project/preparation/source, account availability and finite allowances.
- In the existing packet transaction, lock those inputs and record command,
  assignment/root allowance, reservation and run together. No external operation
  occurs inside the transaction. Launch afterward through the existing host path.
- Adoption saves the selected graph as a new accepted revision using current-head
  CAS. New material product assumptions go back to existing product review.

Planning phases carry no native publication/mutation capability. Capture the
checkout's tracked tree/index/HEAD before and after planning; unexpected source
mutation invalidates the result and preserves the changes for intervention.
Do not describe the shared coder account as a proven read-only OS sandbox.

The issue branch uses the same admission machinery, locking queued control and
acceptance. Selection records `investigate` before launching it. Completed
investigation starts a fresh `implement` child under that root; it does not
re-enter priority competition. Unselected issues incur no generative investigation.
Remove automatic `PlanningPass` strategy calls for unselected/blocked issues;
retained historical outlines can remain visible as non-executable evidence.
Advance settled phases before selecting new work in the existing progress pass.
Uniqueness on `(parent_assignment,parent_result_digest,next_phase)` makes automatic
advancement replay-safe. Persist a validated plan and current pointer atomically;
admit its implementation child through a separate recoverable packet transaction.

Replace every publication-based phase inference together: `dispatch_attempt.go`
must advance completed planning instead of requiring Retry; `dispatch_launch.go`
and `validateAttemptPacketTx` require a PR only for review/correction;
`recordDispatchPacketTx` charges correction only for `Phase=correct`;
`currentAttemptOwnerTx`, expiry and source-preparation custody follow explicit
attempt/phase lineage. Planning cannot be added only to the result parser.

Extend `control/lifecycle_retry.go`, `store/factory_explicit_retry.go`, pending
retry pickup, packet validation and run-view subject binding together. Explicit
Retry of a reconciled terminal failed/cancelled/exhausted intent plan creates one
fresh decomposition root after current owner/revision/scope/grant checks. An
investigation can likewise be retried before a PR exists. Replay returns the same
root; an ordinary second `plan-intent` command cannot escape a terminal attempt.
`decomposition_needs_attention` requires product revision/review, not an unchanged
automatic rerun. Ordinary input changes on an active root retain its consumption;
only the explicit authorized Retry command creates new admitted allowances.

## D02 — Decomposition and priority algorithms

The decomposition agent returns **one to three** candidate graphs in one bounded
run, with inspected source references, rationale and exact intent/scope/source/
preparation bindings. Keep at most `IntentRevision.MaxIssues` issues, with the
existing ceiling of 64. No artificial minimum issue count.

Replace bare `IntentTask.DependsOn []string` with dependencies containing
`task_key`, `outcome_indexes` (zero-based indexes into that prerequisite's immutable
outcomes) and `reason`. Update all callers/fixtures together. Project these records
to the same native dependency edges; do not create another dependency authority.
Code validates IDs, explicit outcome references/coverage, bounds and DAG. The
planner/Jev assess semantic completeness, cohesion and necessary dependencies.

One Jev Choice request includes every candidate plus `clarify`. Candidate
selection requires a valid answer and confidence **>= 0.50**; `clarify` or lower
confidence records `decomposition_needs_attention` without generating another
batch automatically. Use the intent allowance and existing global call ceiling.
Persist the result by producing assignment and the chosen candidate/judgment
digest in the accepted revision. Native generation writes that exact graph;
remove the second template split/combine and decomposition-triplet pass from this
path. Other existing judgment purposes retain their behavior. The development
procedure for a separately requested three-call advisory consultation is distinct.

Before both judgment and accepted-graph adoption, re-observe source/preparation
and effective scope against the planner input. Stale evidence is retained but
cannot be adopted. After confirmed settlement, queue fresh decomposition under
the same active root's remaining allowance; exhausted/closed roots require the
explicit Retry path. Do not erase the stale result or silently refill funding.

Prior canonical task keys/delivered bindings are input to revised decomposition.
Unchanged tasks retain keys, changed tasks use existing diff/adoption rules, and
models never choose native issue IDs. An invalid result or provider failure is
recorded; an explicit existing Retry can start a new reconciled run. Edits and
replay do not refill the original active-time allowance.

**Priority collection uses a durable cursor, not a second full native scan.**
Implement `internal/store/factory_priority_inventory.go` and extend
`control/priority_dispatch.go`; keep the existing serving loop and selector.

1. Obtain idle native revision `N`. In one repeatable-read transaction capture
   complete ordered membership and normalized local inputs together. Include
   accepted blocked controls that may have become ready, order by
   `(first_seen,repository,issue)`, and apply current terminal/retry rules. Keep
   the existing 4,096-item ceiling. Confirm idle `N` again before saving the scan;
   an insertion between membership and digest must not be able to disappear.
2. The local input digest covers
   acceptance/withdrawal, lifecycle, grants, preparation, assignment/retry/
   allowance and current capacity/connection-budget facts. Exclude assessment
   timestamps and observation counters, so refreshing a control cannot invalidate
   its own scan. Include time-dependent budget eligibility as evaluated now, not
   a ticking timestamp or continuously accruing raw usage counter. Exact remaining
   balances/deadlines are rechecked in the final selected packet admission.
3. Persist inventory ID, `N`, local digest, ordered subject references, next
   subject cursor, partial prerequisite observations, candidate/exclusion records
   and eventual judgment/selection. Resume across passes within the existing
   156-RPC, 52-evidence-read and time bounds; a large individual prerequisite set
   also needs resumable progress. No unbounded all-input reservation.
4. A changed native revision or local semantic digest invalidates observations.
   Retain spent judgment receipts. On restart, verify bindings before resuming.
5. At completion, recheck idle `N` and the full local digest, then use the existing
   complete tournament, fairness and uncertainty policy. One candidate skips the
   comparative call; zero waits. Never truncate the set to fit a pass.
6. After judgment, recheck `N`/local digest, then fresh selected-issue evidence
   whose native bracket also reports `N`.
   In a serializable packet transaction, compare the complete local input snapshot
   and atomically admit the selected investigation. Retry serialization at most
   three times with database revalidation only; never repeat the external call.

Persist bounded inventory progress in one current scan record plus candidate rows
keyed by `(scan_id,subject_key)`. Replace invalid observations; the existing
judgment ledger retains paid history. Continuously changing inputs can yield
`priority_snapshot_changing`; eventual stable inputs must make progress. Native
unavailability is `priority_incomplete`, not a subset election.

## D03 — Plan, result and storage schemas

Replace fixed `PlanningRecord.Steps` with immutable `InvestigationPlan` containing:

- `id`, `attempt_root`, `generation`, accepted scope/input digest, original source
  commit, current integration commit, prerequisite output digests and preparation;
- inspected `{path,commit,evidence}` references, findings and unresolved facts;
- approach, rejected alternatives with reasons, acceptance coverage;
- `tasks[]`, `lanes[]`, `checks[]` and the canonical semantic digest.

`PlanTask` contains `id`, `objective`, `outputs`, `acceptance_refs`, `depends_on`,
`lane`, `owned_paths`, `interface_writes`, `check_ids`. `PlanLane` contains `id`
and its ordered task IDs. Ownership entries are exact relative files or explicit
directory prefixes, never globs. Prefix overlap or matching interface-write keys
for potentially concurrent tasks is invalid; encode a dependency to serialize it.
Use the existing factory 32-hex ID generator for durable records. Plan-local
task/lane/check keys match `[a-z][a-z0-9_-]{0,31}`. Dependencies refer to task keys,
not ordinal positions. The tasks array must be topologically ordered, with each
dependency earlier in the array; every declared task is required. No unresolved
mandatory fact can enter an executable plan.

`CheckSpec` is `{id,argv,cwd,timeout_seconds,scope,required}`. `scope` is `task` or
`integration`; `cwd` is a clean relative path or `.`. Execute argv directly, not
through an implicit shell; an explicitly declared script interpreter is allowed.
Environment/secret custody stays with the prepared Project. Required integration
checks run on the final exact candidate even when component checks already passed.

Replace the process result with strict `{version:2,phase,status,summary,output}`.
`output` is the single typed payload for that phase: decomposition, plan,
implementation receipt, review report, correction report or D13 reassessment report.
Completed decomposition, investigation and reassessment forbid a publishable candidate. Non-completed results may retain
diagnostic references but cannot authorize effects. Keep the fenced-result
transport, reject unknown fields/trailing JSON, and bind phase/subject/run from the
recorded assignment. Remove obsolete `review_passed`/string-finding compatibility.

Persist immutable plan revisions in `factory_planning`, keyed by
`(attempt_root,plan_id)`, with current pointers in `factory_plan_heads`. Persist
task state/receipts in `factory_plan_tasks`, keyed by `(attempt_root,plan_id,task_id)`
with CAS revision. Task receipt contains native child identity, input/output
digests, actual event sequence, check/tree/commit evidence and terminal reason.
Store methods own transactions; no SQL in controllers or new repository layer.
Change unreleased schema/fixtures together; no legacy format reader or migration.

New planning files belong in `internal/factory/{planning,plan_task,execution_phase}.go`
and existing `control/planning*`, `store/factory_planning*` owners. Add small files
within those packages when needed; do not move ownership or create generic workflow
packages. The Go domain is canonical; Rust wire types mirror only the host boundary.

## D04 — Fixed limits and accounting

These are product defaults/bounds, not claims of model quality or provider prices.
Record defaults when admitting a new policy/attempt; never infer fresh allowance
from a missing field in a running record.

| Input or allowance | Frozen value |
| --- | --- |
| Attempt time / correction runs | Retain current defaults: 120 active minutes / 3 corrections; existing accounting semantics and explicit Retry remain. |
| Active native task children | `MaxChildren=3` by default; policy range 1–4; actual admission is the lesser of policy and qualified native capacity. Parent is separate; no nested child spawning. |
| Plan size | 64 tasks, 4 lanes, 32 ownership entries/task, 16 reusable checks/plan, 64 inspected source references. Total encoded result remains <=64 KiB. |
| Plan text / checks | Objective or evidence text <=4 KiB each; clean path <=1,024 bytes; argv <=64 elements and <=16 KiB total; check timeout 1–600 seconds and never beyond remaining run time. |
| Review runs | `ReviewRounds=6` by default, allowed 1–16 per attempt, including initial and policy-required reviews. Charge before launch by immutable assignment ID. |
| Delivery judgments | `DeliveryCalls=8` by default, allowed 0–16; zero cannot authorize automatic merge. Reserve against both attempt root and existing global judgment ceiling. |
| Agent prompt/result | Keep 64 KiB each except D13 reassess prompt <=320 KiB with complete bounded inventory inline; its result remains <=64 KiB. Never silently truncate a structured result or required evidence. |
| Execution manifest/events | Manifest <=64 KiB; journal <=512 events/8 MiB, reserving two terminal events; event <=16 KiB; checkpoint metadata <=64 KiB/task and <=64 tasks/run. Logs retain existing separate bounds. |
| Review / findings | 64 findings/report and at most 64 outstanding findings; summary <=4 KiB, combined prose/finding <=4 KiB, encoded native review payload <=48 KiB. |
| Native review receipt/history | Review-specific receipt cap 16 KiB; retain 1 MiB review-history cap, counting full encoded work plus reserved receipt capacity. Other operation receipt limits stay unchanged. |
| Jev delivery evidence | Encoded semantic state <=256 KiB, including complete current findings, corrections, relevant diff and checks. Excess is `delivery_evidence_too_large`, not selective omission. |
| Merge authorization history | At most 32 immutable generations per publication; reaching the cap is a visible intervention. Never create another PR to escape it. |
| Completion reassessment | D13: two Jev requests and one coder run per cycle; policy `ReassessmentMinutes` default 30, range 1–120; closed delivery allowance remains closed. |
| Classifier request | Existing 30-second timeout, no transport retries. |

Existing attempt active time includes review/native waits and is charged once for
the parent interval, not multiplied by children. Existing sponsorship/connection
session-time accounting is preserved. Children remain under the parent's lease,
deadline and process boundary. Provider token counts are supplementary usage:
record either inclusive parent totals or individual child totals, never both;
unavailable counters are explicitly unavailable, not zero. This design enforces
time/concurrency/call bounds and does not add or promise a monetary/token cap.

Extend `AttemptLimits`/`AttemptAllowance` and their store/command/UI owners for
review rounds, children and delivery calls. Consumption is immutable/idempotent by
run/request identity. Reserve calls in the same transaction as the existing global
budget. Failure after reservation stays charged. Only explicit authorized Retry
starts a new root; policy edits, review rounds and input edits do not reset it.

Preserve product role/model selection. Extend its recorded execution profile with
explicit reasoning effort and Contributor mode, replacing hard-coded launcher
flags. New omitted settings resolve to the current product defaults, `low` and
Contributor off, at policy creation; running snapshots are explicit. Contributor
is valid only for Muse. A qualified profile must support the exact chosen effort;
unsupported settings refuse. Children inherit the exact parent profile/account.
The separate **implementation crew** uses the user's requested Spark 1.3
Contributor/XHIGH setup, not those product defaults; its native proof fixtures
must record the actual profile tested.

## D05 — Native children, scheduling and checkpoints

The chosen architecture is **one native agent coordinator with native task
children inside one admitted Project run**. No replacement headless-process pool,
worktrees, extra broker leases or second execution service. P1 must prove a
supported control/event route for the exact pinned CLI; otherwise stop dependent
work and return the failed assumption to Codex. Spark must not design a fallback.

Stage an immutable execution manifest with the existing launch: exact serialized
Go plan bytes/digest, invocation, phase/run/deadline/profile and child limit. The
parent owns the ready queue; the Go progress loop observes it rather than starting
each task itself. Rust verifies the digest of the supplied bytes, not a separately
reconstructed cross-language serialization.
Add `execution_manifest` bytes to `FactoryLaunch` and its digest/explicit execution
phase to `FactoryRun`. The manifest is required for `implement` and absent for
the other phases, which remain single-agent runs. Keep the daemon's existing
1 MiB launch envelope; field caps above apply before staging. New wire parsing
replaces callers/fixtures together without an old-format branch.

The Rust Project executor translates the supported native route into typed
observations for the existing host boundary. Add
`FactoryTaskEvents(project,run,after_sequence,byte_limit)` alongside current
inspect/output at authenticated `POST /factory-task-events`: byte limit 1–24,320,
at most 64 whole events/response, each event
<=16 KiB, with a next-sequence/terminal cursor. Events bind parent run, native
invocation, plan digest, task key, sequence and previous-event digest. Types are
`child_start_intent`, `child_started`, `child_reported`, `checkpoint_prepared`,
`task_committed`, `task_failed`, `run_stopped`. A response too small for its next
event returns a bounded size error without advancing. Ordinary prose cannot
advance task state. Native control
commands are only start eligible task, observe/join child and cancel child; they
are internal to this admitted parent, not a public arbitrary-process API.
Add the route, backend method, strict request/response and Go host client in the
existing daemon/project/host owners. It is read-only, subject to existing run/
Project/incarnation checks, with the existing 64 KiB response cap. Journal
exhaustion stops admission and records a terminal reason; it cannot drop events.

Task states are `pending → running → output_ready → completed`, or terminal
`failed/cancelled`. A dependent becomes ready only after validated checkpoint/
output evidence completes all prerequisites. Choose ready tasks in plan order,
breaking ties by task key, respecting one active task per lane, ownership locks
and `MaxChildren`. Release a ready child immediately on a completion event; an
unrelated active lane is not a global barrier. One child owns a task until its
terminal receipt. No automatic new child after a failed/uncertain task; normal
agent fixes/check iterations occur within the original live child and deadline.
Once a task reports terminal failure or exact-tree verification fails, stop new
task admission, stop remaining children, preserve checkpoints and finish with the
concrete failure. Do not spend an undeclared replacement-child retry allowance.

The parent's Rust executor alone integrates Git; the model proposes task output,
and children cannot authorize ref/index updates. File ownership is scheduling/cooperation, not a
claim of OS isolation between same-role children. Reject unexplained changes and
unexpected index/HEAD mutation; do not reset or discard them to continue. Shared
files need released ownership. Inspect/checkpoint changes using literal pathspecs;
reject absolute paths, `..`, `.git`, `.soda-home` and symlink escapes.

For each coherent released task output:

1. Create a private Git index in protected checkpoint state from the current integration
   HEAD; stage only that task's released paths with `--literal-pathspecs`.
2. Prepare the candidate commit object using Git plumbing, fixed parent and
   recorded role author/message/time, without changing any live ref. Materialize
   its exact tree into a disposable **check-only export** under `/home`, excluding
   unrelated dirty lane files. Give it isolated Git metadata, a private index and
   detached HEAD equal to that candidate. A read-only object alternate may refer
   to immutable canonical objects; never share live index/refs/config or use
   `git worktree`. This export is never an agent editing checkout. Git-aware
   checks must see correct `git ls-files` and `git rev-parse HEAD` results.
3. Run required checks against the export, with caches/output under `/home`.
   Verify its tracked tree, HEAD and index remain identical afterward; a check
   that modifies tracked source cannot count as passing. Record tree OID, check
   specs/results and log digests. Durably write/fsync `checkpoint_prepared`
   containing old HEAD/index tree, candidate tree/OID and checks digest **before**
   compare-and-swap of the live integration ref. Release the run lock during
   checks. Reacquire it and revalidate invocation, deadline, stop tombstone,
   released ownership and expected HEAD/index before the prepared/CAS transition.
   Serialize that short transition with stop; cancellation wins if already recorded.
4. After CAS, update the ordinary index without replacing other workers' working
   files, then persist `task_committed`. On lost reply, HEAD equal to candidate
   means adopt; HEAD equal to old parent permits identical CAS; anything else is
   a conflict. After CAS but before index update, the recorded old index is a
   valid intermediate state: reconstruct the new index without touching files.
   Any unrelated index tree is a conflict. Never create a different commit to
   repair an uncertain reply.
5. Validate checkpoint ancestry and task output digests before releasing dependants.
   Final required integration checks bind the final candidate tree before PR
   publication. No arbitrary minimum commit count.

A verification-only task can complete on a checked unchanged tree without an
empty commit. Its `task_committed` event means accepted output/checkpoint evidence,
with an optional commit OID. An entirely unchanged issue ends with an explicit
no-change/intervention outcome, not a fabricated PR.

The live checkpoint path is `FactoryTaskCheckpoint` at authenticated
`POST /factory-task-checkpoint`, in the existing Rust factory owner and Go host
client. Its request contains Project/run/incarnation, plan digest, task key,
reported-output digest and expected integration HEAD; the host loads fixed
ownership/check specifications from the admitted manifest. It accepts no arbitrary
argv or claimed successful checks. Go invokes it after observing `child_reported`;
the Rust adapter delivers the accepted checkpoint receipt to the same native
parent before that parent releases dependants. P1 must prove this round trip.
Keep read-only event polling free of checkpoint mutations. Do not reuse settled-run
candidate inspection to authorize this live operation.

Serialize checkpoints per run while coding lanes continue concurrently. Go queues
reported tasks by their `child_reported` event sequence and forms the next request
only after the active checkpoint completes, using its accepted integration HEAD.
Rust permits at most one task in `preparing/checking/checked`; another task receives
`busy` without creating an operation or consuming a retry. A checkpoint failure
stops the run as above. Independent outputs must not race CAS from the same parent.

Make the operation resumable under identity `(run,plan_digest,task_key)`: persist
`preparing → checking → checked → committed` or `failed` in private checkpoint
state. Requests return bounded `pending`, `committed` or `failed` state within the
existing host request deadline. Existing executor process custody owns check
processes and observes their handles on subsequent calls; no second scheduler or
agent session. Duplicate identical requests observe the same operation; different
inputs conflict. Unknown check-process outcome fails the task instead of rerunning
it. Stop/deadline retires those processes with the run. Check results come from the
executor's process exit/output observation, never a file supplied by the model.

Keep authoritative task events and prepared checkpoint records in a bounded
per-run journal beside the existing private host receipts (`Factory.state_dir`),
under the existing run lock. Its directory and ancestors must remain outside
agent-writable checkout/run homes and inaccessible from the Project container;
validate service ownership, private permissions and absence of symlink redirection.
Use verified directory handles and no-follow journal/lock opens; existing mode-only
directory validation is insufficient. Use private files, atomic rename and file/
directory fsync; a prepared record must be durable before ref CAS. Agent-owned
`.soda-home` files are untrusted proposals/logs, never authoritative receipts.
The Rust executor records observed native lifecycle and independently verified
Git/check facts; the model cannot append accepted events or supply a successful
check receipt. Go stores verified imported receipts. Proof/test fixtures place
the private state under `/home`, with a distinct service owner. On daemon restart,
observe the same native parent/children and protected journal before any launch.
Import events uniquely by `(run,sequence)`; exact replay is a no-op, different
bytes, sequence gaps or wrong invocation refuse advancement. A lost child-start
reply is looked up by the recorded task-execution identity; if existence cannot
be established, stop/fence the parent instead of repeating the start. Native
parent/supervisor death requires confirmed child retirement and existing explicit
Retry, preserving completed checkpoints. No autonomous session reconstruction.
An unobservable live child fences the parent/connection. Whole-run cancellation
first closes admission, records the stop tombstone and stops the native process
boundary, then reconciles credentials; no child may outlive confirmed retirement.

If investigation or implementation discovers a mandatory design/scope gap, return
`blocked` with evidence and preserve checkpoint commits. Do not let a worker expand
the accepted task graph. Existing maintainer revision/Retry can obtain a new plan
against retained evidence; completed work is not silently replayed or erased.

## D06 — Native review and thread contracts

Fountain's SDK extension is generic:

```text
ReviewSubmitPayload += comments[] {key, path, side: old|new, line, body}
ReviewSubmitReceipt += summary_comment_id, comments[] {key, comment_id}
```

Keep the existing exact PR/actor/head/base/commit fields and final review events.
SDK keys are opaque client correlation keys; Soda severity/dispositions/Jev never
enter Fountain policy. Summary, native code comments and operation receipt commit
atomically. Do not adopt drafts or bypass conditional operations via ordinary API.

Anchors use native PR diff semantics: old side is the native merge-base tree,
new side the exact head. The host derives merge base from expected head/base and
returns it in the receipt; the client cannot substitute one. For renames, old
side uses old path and new side uses new path. Lines are one-based actual diff
anchors; context lines follow native rules. Invalid/binary-only anchors refuse;
non-line findings must be explicitly represented in the summary. Reject duplicate
`(path,side,line)` in one report; consolidate defects at a shared anchor.

Soda `Finding` adds stable 32-hex key, origin review/run, severity
`critical|major|minor`, `actionable`, defect, evidence, required outcome, fix kind
`code|verification`, and optional native location. Preserve keys through rounds;
new defects get new keys. Non-inline findings map to summary review/comment plus
their key. Complete encoded sizes, not just body strings, enforce D04 limits.

Native thread identity is the first comment in native `(created_unix,id)` order
within `(issue,review,path,signed-line)` grouping, matching native conversation
ordering. That root ID is stable evidence; deletion invalidates its mapping and
cannot silently transfer accepted resolution to another root. Replies are members,
not resolution targets. Add conditional
`pull_request.review.thread.resolve` with exact PR/head/base, root comment ID,
expected complete conversation digest and `resolved=true`. Digest covers member
IDs, bodies/content versions, locations and resolver state. Reuse native
`CanMarkConversation`/`MarkConversation`; resolve the anchored root, not an invented
thread table. An unexpected reply/edit/unresolve refuses stale resolution.

Add an explicit enrolled `Resolve` actor binding to repository policy, operation
kind validation, settings/CLI and adapter wiring. It uses the reviewer account
identity with its own operation authorization; review permission is not a reusable
resolve credential. Keep native authorization checks; no administrator bypass.

Extend existing snapshots with root/location/original-current-commit/resolver
fields. Page size stays 50. Observe every required page at one native revision,
including outdated unresolved threads. Hidden/truncated/mixed evidence blocks
acceptance. New unmapped code-review conversations require a fresh reviewer to
classify each as a keyed finding or informational with rationale before judgment.
Bind classification to root ID and complete semantic conversation digest; any
reply/edit invalidates it, including formerly informational threads. Human-origin
findings retain native actor/review/root provenance with a separate classifier-run
reference, not an invented agent origin. Carried/human findings reuse their native
mapping; only new agent-authored findings enter `ReviewSubmit.comments[]`.
Ordinary PR chat is not automatically a defect; native blocking reviews always
retain their independent effect. Human-required approvals cannot be manufactured
by the configured agent reviewer.

Owners: existing X SDK/review/snapshot, native-operation and native model/service
files; Soda `review_native.go`, `publication.go`, `internal/forgejo/review.go` and
snapshot adapters. Use a review-specific receipt bound; do not enlarge every
generic native receipt.

## D07 — Corrections and review cycles

Persist a review cycle with publication, monotonically increasing round number,
triggering decision/reason, exact head/base, input finding digest and fresh reviewer/
correction assignment IDs. IDs derive from publication, attempt root, round and
phase. New round on the same head is distinct; replay of a round is identical.

A report carries new findings plus explicit `verified|still_open` assessments for
every carried actionable finding, with evidence. It cannot silently omit them.
Corrections receive the whole current finding set; output has exactly one
`fixed|unresolved|disputed|declined_optional` disposition per supplied key and the
exact finding-set digest. Optional-only reviews go straight to Jev: optional items
are recorded as `declined_optional` unless implemented, and never force a coder
run. Actionable items cannot be declined by an agent.

Fixed code findings require relevant changed paths/commits and exact checked-tree
evidence. Verification-only findings may retain the same head when new passing
checks prove their required outcome. Unchanged code cannot claim to fix a code
defect. Missing/duplicate/unknown/stale dispositions refuse; disputes use existing
intervention. Every correction run consumes one existing correction cycle.

Mechanically complete proposed fixes can reach Jev while their native threads are
open. A reviewer may verify them in a new round, or Jev may accept the full set.
Persist accepted-fix identities and intended resolution operations before writes;
only then resolve/read back and proceed to merge. Native resolved state alone is
never semantic proof. A clean first review also requires Jev's decision.

## D08 — Native merge eligibility and Jev policy

Add an actor-bound read to existing native snapshot/background transport, using
the enrolled merge binding and deriving the actor from authenticated context.
Bind PR/head/base and native revision. Reuse the evaluator behind
`CheckPullMergeable(..., false)` and expose its first typed blocker in native order:
permission, lifecycle, mergeability/checking, method, status checks, approvals,
requested changes, outstanding review request, protected files, signature or
native dependency. Include relevant native review IDs. Refactor shared typed
reasons if needed; never parse human error messages or duplicate policy in Soda.

If the configured reviewer can supply a missing approval and has not done so,
dispatch one bounded review. If it already approved or another actor's approval/
blocking request is required, wait for that native actor. Avoid repeated agent
reviews that cannot satisfy policy. An open thread with a complete proposed fix
may reach semantic judgment, conditional on successful resolution afterward.

Use purpose `delivery-review-v1`, the current Go TypeSafe client and exactly
`review_again` / `ready_to_merge`. Request state includes every current finding,
proposed fix and relevant diff/check evidence. Criteria are fixed:

- `ready_to_merge`: supplied evidence adequately covers accepted outcomes and
  every actionable finding; another independent pass is unlikely to find a
  material defect. No missing evidence, unresolved dispute or requirement can
  be treated as satisfied.
- `review_again`: a further independent inspection is warranted to establish
  coverage or correction sufficiency from the supplied evidence.

Effective acceptance requires selected `ready_to_merge`, its probability
**>=0.80**, and returned confidence **>=0.80**. Otherwise effective action is
`review_again`, if another review fits the allowance, or `review_allowance_exhausted`.
Keep raw answer and effective action. These are conservative product thresholds,
not probabilities that the code is correct; [TypeSafe confidence describes the
answer distribution](https://docs.typesafe.ai/confidence). No paid request occurs
when deterministic evidence/authority/check prerequisites already block it.

Reuse `AdvisePurpose` and the existing journal. Delivery funding uses the attempt
subledger plus global budget in one transaction. Missing/malformed/provider-failed
answers give `judgment_unavailable`; an already reserved fingerprint without a
completed answer stays charged and is not automatically retried. Successful
unchanged decisions are reused. Explicit authorized Retry creates a new attempt;
clock ticks and incidental polling do not create new request identities.

Fingerprint schema/version, rubric/options, configured model, attempt/publication/
round, head/base, accepted scope/policy, reviews, semantic conversation content,
proposed fixes/diff, check outcomes/tree and native constraints. Sort set-like
collections; preserve ordered tasks/diff. Use Go's deterministic typed JSON plus
SHA-256; reject non-finite numbers. Exclude timestamps, global native revision and
poll counters. Keep raw observed revision separately for native mutation guards.

Expected resolution operations are named consequences of the acceptance. Their
exact confirmed effects may advance the native revision without buying another
judgment. Any unexpected reply/content/state, head/base, relevant policy/scope or
check-outcome change invalidates it. Pending resolution cannot become merge-ready
merely because a model selected the option.

## D09 — Merge authorization generations and completion

Add a current candidate verification tuple `{head,base,diff_base}` to the
publication cycle, with its own CAS revision. Derive `diff_base` from the actual
native merge-base. The immutable original `PRCreate` operation/receipt stays
historical. Current prompts, diff anchors, checks, reviews, Jev and merge admission
use the current tuple; historical review/operation entries validate against their
own admitted tuples. Update `publication.go`, `review_native.go`, check work,
`publication_children.go` and their store validation together; retaining
`PRCreate.BaseOID` as the permanent current base would create a stale-base loop.

Keep one `factory_merges` ledger per publication, with one current authorization
generation and bounded immutable prior-generation history. Do not keep the current
one-terminal-operation restriction, which would strand a PR after cancellation.
The ledger retains stable publication/repository/issue identity. Candidate, base,
authority, deadline and native-operation fields belong to each generation;
`UpdateMerge` validates append-only history and never rewrites a prior generation.

Each generation captures decision ID, review round, semantic fingerprint, exact
head/base/check evidence, authority, observed native revision and deadline. Its
native operation ID derives from publication plus generation; content is immutable.
In the registration transaction require current applicable `ready_to_merge`,
confirmed accepted resolutions and all current authority gates.

On superseded acceptance, close that generation's admission, cancel/reconcile its
native operation, and return the still-authorized attempt to review/judgment.
Create a new generation only after predecessor effect is proven `not_committed`
with confirmed cancellation, or deterministic no-effect refusal. Pending,
indeterminate or committed predecessor forbids replacement. Generation replacement
does not reset any attempt allowance. D04's history ceiling applies.

Every new mutation path—publication progress, `MergePass`, `mergeOne` and
reconciliation—uses the same decision gate. Recovery first looks up the recorded
operation: a merge already committed under earlier valid admission is observed
and adopted even if inputs changed later. It is never resubmitted or erased by
cancellation, and observation consumes no new Jev call.

Retain fast-forward-only native merge and current protection checks. A changed
base invalidates pending acceptance. Refresh the current tuple from native refs;
if the new base is not already an ancestor of the candidate, dispatch bounded
coder correction to **merge the new base into the candidate**, preserving prior
checkpoint ancestry. No automatic rebase/force-push. The new tuple, integration
diff, exact-tree checks and correction receipts then enter the normal review/Jev
path; native branch policy may require another approval. Conflicts that cannot be
resolved within the correction allowance become intervention. Confirm reachable
merged result, PR closure and original issue closing occurrence before dependency
release. In that same completion transaction retain readiness enqueue and record
the D13 completion event/scheduling fence. Dependency satisfaction does not bypass
post-completion reassessment before selecting the next issue.

## D10 — Visible state and failure routing

Expose phase/attempt, plan revision, tasks/lanes, verified checkpoint commits,
native PR/comments, correction dispositions, round, raw/effective Jev choice and
precise waits through current Work/Spaces read models. Show choices and supplied
criteria/evidence; do not invent a textual explanation attributed to Jev. Read
models never call a provider or mutate workflow state. D13 adds completion-gate,
reassessment proposal/adoption and explicit Retry state to these same read models.

Add bounded reason codes in existing error/DTO owners for invalid planner output,
decomposition attention, incomplete/changing priority inventory, unsupported
native task control, stale plan, task failure, checkpoint conflict, incomplete
review evidence, disputed findings, judgment unavailable/exhausted, review
allowance exhausted, oversized delivery evidence and native policy waits. Preserve
existing authority, capacity and lifecycle codes. Do not turn every wait into a
new approval workflow. Existing Retry, revise, pause/stop and native review actions
remain the recovery routes under their current authorization.

Text output uses current offset-based terminal streaming, labeled with parent/
child identity where native output permits. The typed task-event stream supplies
state; logs supply detail. No screenshot log relay, new timer or scheduler.

## D11 — Exact evidence-gate failure rules

| Gate | Required fact | Failed gate response |
| --- | --- | --- |
| P1 | Exact pinned CLI exposes supported native child start/observe/result/cancel and actual overlap; children inherit profile/custody and terminate with parent; live host-mediated checkpoint/acknowledgment and stop ordering work. | Preserve version, command, events and failure. Mark affected harness profile unqualified and stop FW11/FW13/FW14 consumers as applicable. No process-pool, extra lease, silent serial fallback or undocumented CLI command. Codex must revise a failed design assumption. |
| P2 | Conditional native code comments/receipts remain atomic; root identity and full snapshot match native UI; policy evaluator supports trustworthy actor-bound read. | Stop affected native/Soda delivery consumers. No ordinary API bypass, guessed blocker or parallel thread database. Return exact failing transaction/fixture to Codex. |
| P3 | Existing production bounds reproduce or disprove large-inventory starvation. | If reproduced, implement D02 cursor algorithm. If disproved by the real path, retain regression evidence and ask Codex to remove the unnecessary fix; do not invent another optimization. |
| P5 | Full actor-bound open-issue enumeration covers later pages; generic graph packets preserve frozen records/edges, refuse stale/oversized work atomically and replay the exact receipt. | Prove the narrow enumeration extension and existing graph boundary in FW29 before FW31/FW32 consumers. No ready-only inventory, ordinary API mutation bypass, guessed acceptance regeneration or silently chunked graph. Return failed native evidence to Codex. |
| P4 | Draft/no-repository launches nothing; authorized draft with repository can run without an accepted graph; actual broad/small outputs satisfy semantic acceptance. | Missing account/preparation is an explicit external gate. Wrong authority/subject behavior blocks implementation acceptance; semantic failure returns exact prompt/output for Codex's rubric/design correction. No template fallback. |

Proof tasks may implement the smallest chosen native primitive to test it. They
must not invent a replacement primitive on failure. All fixture work is disposable
and under `/home`; preserve canonical checkouts and protected state. Evidence for
one harness/profile is not qualification of another. No full image build is needed
to discover any of these facts.

## D12 — Decisive test list and completion mapping

These tests extend current suites. Names below identify required cases, not a new
test framework. `F/C/S/N/P/X` retain the plan's owner abbreviations. Cases with
native or funded evidence are separate from deterministic adapter tests.

| Case | Task / existing suite owner | Required assertion |
| --- | --- | --- |
| T01 Draft authority | FW08; intent API + S dispatch packet | `Repository=0` cannot launch; saved review alone cannot fund; valid reviewed draft can plan without fake issue/accepted task graph. |
| T02 Subject transaction | FW11; S dispatch/allowance | Wrong owner/revision/Project/grant refuses; intent/issue share repository slot; replay creates one run and rollback none; explicit intent/investigation Retry before PR works once, while ordinary command replay cannot refill a terminal root. |
| T03 Phase/result boundary | FW06A/FW11; F result + P role tests | Each valid phase/role/result combination; cross-phase payload, extra JSON and completed planning candidate refuse; automatic same-parent advancement is idempotent; only correction phase spends a correction cycle. |
| T04 Scope graph | FW09; intent/decomposition tests | Invalid IDs/outcome indexes, missing coverage, cycles, declined scope and false provenance refuse; prior task keys/delivered bindings retained correctly. |
| T05 Semantic examples | FW05/FW09; existing exploration fixture, then funded run | Existing tracker: independent CSV export and label filter; dashboard needs both; saved reports need dashboard. Preserve roots/join/descendant. A help-text typo stays one issue. A consequential unspecified access policy returns attention. |
| T06 Candidate choice | FW09; intent judgment/generation | All 1–3 candidates plus clarify supplied; confidence 0.50 boundary; no second triplet/template rewrite; exact graph adopted once; source/preparation/scope changed after planning prevents adoption and refresh preserves active-root consumption. |
| T07 Funded-call replay | FW09/FW21; S judgment allowance | Reserve both scoped/global budgets before request; failed/interrupted request remains charged; repeated fingerprint never resends; no funds means zero calls. |
| T08 Large queue progress | FW04/FW10; serving priority fixture | 256 candidates use actual per-pass limits, eventually complete, survive restart and all reach selection; no second full native scan. |
| T09 Inventory invalidation | FW10; priority/readiness | Insert during initial membership/digest capture cannot disappear; changed eligibility/grant/source/preparation/budget invalidates; ticking timestamps do not; selected evidence must match N; partial large prerequisite reads resume. |
| T10 Selected investigation | FW12; planning + S packet | Two ready issues produce one persisted investigation; no unselected prompt/planning run; crash/replay no duplicate spend; stale selected admission refuses. |
| T11 Immutable plans | FW07/FW12; F/S planning | Unique keys, DAG/coverage/ownership/limits; old plan survives new head pointer; stale source/prerequisite/preparation cannot launch implementation. |
| T12 Native overlap | FW02/FW13; P native fixture | Two independent children actually overlap; release a dependent while unrelated child remains active; exact profile/account inheritance and child cap. |
| T13 Task event recovery | FW14/FW24; P event + S task fixtures | Contiguous/hash-bound replay, duplicate event idempotency and byte bounds; wrong invocation/task/native ID refuses; agent-written/replaced run files cannot forge accepted events or check receipts and private journal ownership/symlink violations refuse; lost start reply is looked up, not replayed; native parent death stops children and requires explicit Retry. |
| T14 Checkpoint purity | FW14/FW15; P Git fixture | Dirty sentinel excluded; literal path guards; export's Git tracked list/HEAD match exact candidate; source-mutating check refuses; checked tree equals checkpoint; simultaneous independent outputs serialize checkpoints against successive heads while coding overlaps; descendants wait for verified output. |
| T15 Checkpoint lost reply | FW14/FW24; P Git fixture | Resumable checkpoint requests observe the same checks and reject changed inputs; cancellation during checks stays responsive and prevents later CAS; crash before/after ref CAS adopts identical prepared commit or retries identical CAS; crash before index update reconstructs only recorded intermediate state; foreign HEAD/index conflicts; preserve worker files. |
| T16 Cancellation/bounds | FW13/FW14/FW24; host/broker/allowance | Parent stop/deadline retires all children before credential release; uncertain stop stays fenced; children/reviews/calls never reset active/correction budget. |
| T17 Publication join | FW15; publication/lost-reply tests | Incomplete tasks or wrong ancestry cannot publish; verification-only task needs no empty commit; wholly unchanged issue makes no fake PR; final exact-tree checks required; one candidate produces one PR. |
| T18 Inline atomicity | FW03/FW16; X native review tests | Added/deleted/context/rename anchors; binary/invalid/duplicate anchor refuses; stale/cancelled submit writes no partial comments; replay keeps native IDs. |
| T19 Receipt bounds | FW16/FW18; F publication + X SDK | Maximum allowed report/receipt persists; 48/64 KiB and history caps reject above bounds; full work and reserved receipt count. |
| T20 Complete comments | FW17; X snapshot + N adapters | Later-page/outdated roots retained; mixed/hidden/truncated evidence blocks; new or edited/replied human conversation triggers classification, including formerly informational roots. |
| T21 Resolution authority | FW17/FW19; X native operation | Missing resolve binding, wrong actor/root/reply target, deleted root, changed content or concurrent reply refuse; native root derivation and exact replay/readback succeed. |
| T22 Finding accounting | FW18/FW19; correction/result | Missing/duplicate/unknown/stale/disputed actionable item cannot pass; optional-only report causes no correction run; same-head verification fix allowed, unchanged code fix refused. |
| T23 Independent rounds | FW20; publication children + review cycle | Fresh reviewer/corrector sessions; carried/human findings assessed without reposting duplicate comments; same-head second review is a fresh run, replay identical; newly found defect enters correction. |
| T24 Jev policy boundaries | FW21; judgment tests | Clean first approval still consults; ready requires both probability and confidence >=0.80; below/tie routes review; raw/effective choices retained; malformed/unavailable answer cannot merge. |
| T25 Semantic freshness | FW21/FW22; judgment/review | Relevant head/base/scope/policy/finding/check change invalidates; timestamp/native-revision-only changes do not; exact expected resolution readback adds no paid call. |
| T26 Native blocker parity | FW03/FW17/FW22; X eligibility/exact merge | Read agrees with permissive/protected mutation; missing configured approval dispatches once; already-approved/other-actor blocker waits without repeated agent reviews. |
| T27 All merge entries | FW22; merge/progress/reconcile | Missing/currently invalid Jev acceptance refuses every new operation path; accepted fixes resolve/read back before registration; native checks remain authoritative. |
| T28 Replacement authorization | FW22/FW24; S merge + controller | Persist intent → invalidate → confirmed cancel → new review/Jev → new generation merges same PR. Base advance → bounded merge-in correction → current tuple/checks/Jev → same-PR merge; historical PR/reviews unchanged. Uncertain predecessor forbids replacement. |
| T29 Committed lost response | FW22/FW24; merge evidence | Actual completed merge is adopted after lost reply/input change, with no resubmit/rejudgment; closure verification still required. |
| T30 Dependency release | FW25/FW31; serving/prerequisite tests | One parent merge cannot release two-parent join; both verified original closures do; join unlocks descendant; reopen/stale closure refuses; truth of readiness cannot bypass a pending D13 gate. |
| T31 Views | FW23/FW25; plan API + current frontend tests | Real persisted phases/tasks/lanes, commit/comment links and effective Jev waits render; read endpoints spend nothing; native CLI text streams without screenshot transport. |
| T32 Actual journey | FW25/FW26; existing serving/native drivers | Real prompt entry → graph → Jev selection → investigation/plan → native parallel tasks/checkpoints → PR → inline review → correction → another review → Jev acceptance → verified merge/unlock → D13 Jev gate → reassessment/no branch → next selection. Also clean-first-review and direct-after-fix acceptance variants. |
| T33 Completion atomicity and fence | FW30/FW31; merge/store/dispatch | Completion, old-root closure, event and readiness commit/rollback together; replay records once; two completions require two decisions; zero/one ready candidate, Retry and phase advancement cannot bypass; unrelated repositories progress. |
| T34 Full inventory | FW29/FW31; native snapshot + serving cursor | More than 50 open issues including blocked/manual/unaccepted/later-page items; PRs/closed excluded except explicit context; missing/hidden/page/size/revision failures hold; resumable scan does not omit a concurrent insertion. |
| T35 Routing, limits and identity | FW30/FW31/FW32; judgment/packet/phase | Continue makes no agent; reassess admits one coder completion root; wrong subject/role/payload/candidate refuses; reassess-only 320-KiB prompt and all 64-KiB results enforce byte boundaries across Go/Rust/CLI without argv transport; low-confidence/provider/unknown-response waits; cycle/global reservations charged once; old root stays closed; explicit Retry preserves history. |
| T36 Proposal scope and coverage | FW32; result/intent/adoption | Full coverage required; valid empty delta; required attention prevents adoption; scope expansion/manual/protected edits, false outcome references, cycles and packet 64-operation/48-KiB excess refuse before any prefix; accepted proposal judgment required. |
| T37 Frozen identity and history | FW29/FW32; graph/prerequisite/recovery | Completed/unchanged issue AND acceptance IDs survive revisions; current-binding existence works with older rows; completed dependency still satisfies; inflight publication not cancelled; human edits remain unaccepted; protected outgoing edges/routes unchanged; changed prerequisite acceptance rebinds idle dependents transitively or refuses a protected route. |
| T38 Partial adoption and restart | FW32/FW24; existing native graph lost-reply + store | Two intents: first commits, second changes externally; keep prefix, fence next dispatch, do not duplicate or rollback; resume exact operation after lost reply or crash between head CAS/packet; generic recovery cannot run a later step; expected prefix changes alone do not invalidate; successor Retry/continue cannot abandon an activated unmaterialized head. |
| T39 Freshness and release | FW31/FW32/FW25; serving/read models | Source/scope/authority changes invalidate before effects; accepted no-change and confirmed full adoption release only their event; pending later events still fence; fresh priority chooses next; visible decisions/proposals/waits spend nothing on reads. |
| T40 Reassessment journey and semantics | FW25/FW26; existing native drivers | Exercise continue, reassess-with-change and reassess-with-no-change through real completion entry. Deterministic fixtures prove transitions; separate first-response real Jev cases cover stable remaining plan, completion exposing a necessary dependency revision, and scope-expanding proposal refusal. Missing provider/native inputs leave this qualification open. |

Use existing source/package checks for changed owners and native x86_64 fixtures
for real boundaries. Real-provider semantic evidence must be labeled separately
from canned responses. Record exact versions/profiles, commands, assertions and
receipts. Stop broadening checks once the affected contract is established.

Native journeys may use deterministic judgment adapters to force both transition
branches. Label that evidence accordingly. Funded semantic qualification separately
uses a fixed satisfactory-correction case and a structurally complete but
semantically insufficient correction case; require acceptance and further review,
respectively. Preserve the first responses—no resampling until a preferred answer
appears. A failed semantic assumption returns to Codex; canned output cannot repair
or qualify it. Model/provider setup is a named external gate when unavailable.

Final acceptance requires all applicable cases, independent review of merge and
authority invariants, documentation and focused local commits. External missing
provider/component inputs leave their native qualification cases open. Spark must
report them precisely; tests of substitutes do not close those cases.

## D13 — Reassess current issues after every verified completion

This is the **FW-REASSESS-002** amendment, decided by Codex on 2026-10-16.
It adds to the running implementation; completed proofs, commits and assignments
remain valid within their original scope. The following choices are frozen.

### Trigger, scope and scheduling fence

A completion means Soda has verified the merged result, PR closure and original
issue-closing occurrence under D09. Agent success, a closed PR without merge,
manual issue closure and a polling tick are not completion. In the first
`UpdateMerge` transaction recording that completion, atomically close its issue
allowance, retain readiness enqueue, insert a unique `CompletionEvent`, and
increment the repository's completion generation. Rollback rolls back all four.
The event binds repository, issue, acceptance, publication, merge authorization,
merged commit and original closing occurrence. Its unique key is publication plus
original closing occurrence; observing the same effect through another recovery
path cannot create another event. Use existing 32-hex record IDs and immutable
completion evidence, not the coalescing readiness-root ID.

Process events by repository generation. Each event gets its own Jev decision,
even if multiple completions have identical remaining inventories or none remain.
Do not combine several events into one paid decision. Repository scheduling is
fenced while any event is unresolved. Dependency satisfaction remains truthful;
new issue investigation/implementation waits with `reassessment_pending`.
Readiness observation, already-running work, published-PR review/correction,
cancellation and native-effect reconciliation may finish. Reassessment waits
for the existing repository execution slot, without holding a broker lease while
waiting for Jev or evidence. Other repositories continue normally.

Enforce the pending-event/generation comparison in the final dispatch packet
transaction, including Retry and automatic investigation-to-implementation
advancement; an earlier priority check is insufficient. Include this generation
in the D02 semantic snapshot. A pending event for one repository excludes that
repository from selection; it must not turn the whole global tournament into a
wait. Continue observing recoverable effects instead of launching new work through
`dispatchAfterIntake`. Completion insertion and packet admission lock the same
repository control row; whichever commits first defines ordering.

“All current issues” means **all open non-PR native issues in that repository**,
including blocked, unaccepted and manually authored issues, plus exact delivered
prerequisites/accepted-graph bindings needed to interpret them and the completion.
It does not mean only ready controls or only the completed issue's intent. Other
repositories are outside the scope. Manual/unaccepted issues are read-only
context, not new spending or mutation authority.

Collect a complete stable native inventory, ordered by issue index, through the
extended existing paged read owner, with an idle native-revision bracket and a durable
cursor under the existing pass limits. Capture local accepted intent heads,
acceptance/control/plan/active-assignment state and grants in one repeatable-read
transaction;
confirm the native bracket before sealing. Reuse D02 cursor machinery where
appropriate, but give this full inventory a distinct purpose and membership.
Maximum 4,096 issues and 256 KiB complete encoded semantic evidence; missing
permissions/pages, excess size or unstable evidence visibly hold the gate. Never
truncate, summarize away required issue bodies/outcomes/dependencies, or confuse
an empty page with a complete empty inventory. Supplied evidence includes source
commit/preparation, completion/check results, accepted scope and task bindings,
issue title/body/dependencies, active status and exact semantic digests. Ordinary
observation timestamps do not invalidate it; material content/source changes do.

The current native snapshot has individual-issue reads but no repository issue
enumeration. Add the generic `repository_issues` family in Fountain's existing
`sdk/snapshot.go` / `services/nativeoperation/snapshot_read.go`, and the matching
Soda snapshot adapter. It selects open non-PR issues of exactly `RepositoryID`
under the bound actor's issue-read permission. Require this family alone, no
issue/pull/ref selectors; use existing limit 1–50 (default 50), decimal cursor as
exclusive issue-index lower bound. Return `RepositoryIssues` page with
`items[{issue_id,issue_index,visible}]`, `next_cursor`, `complete`, ascending indexes;
`complete` means fully enumerated through the final page, not merely a successful
query. Initial cursor is empty; a nonempty cursor must be a canonical positive
int64 decimal index, otherwise refuse. A nonterminal page has nonempty items and
`next_cursor` equal to the final returned index, strictly greater than the input.
A terminal page has `complete=true` and empty `next_cursor`. Read one extra row
after open/non-PR filtering to determine terminal status, without returning it.
An empty incomplete page is invalid. Denied enumeration refuses; unavailable/hidden items cannot imply an empty
or complete set. Do not expose hidden content. Fetch exact issue/dependency bodies
through existing individual families, under the same stable revision. The generic
read applies no factory readiness/acceptance filter. P5 proves this small native
extension before building its complete-inventory consumer.

### Decision, agent and finite funding

Use the existing Jev Choice adapter with purpose
`post-completion-reassessment-v1` and exactly `reassess` / `continue`. The rubric
asks whether the delivered change or current issue set warrants revising scope
allocation, decomposition, acceptance details or dependencies before selecting
another issue. Jev receives the evidence; it does not investigate source or write
a plan. A valid selected option with confidence >=0.50 is actionable. Low
confidence or malformed/unavailable response is `reassessment_judgment_attention`,
never an inferred `continue` or an automatic extra agent call. A `continue`
response releases this event after freshness checks, then ordinary fresh priority
selection runs. It launches no reassessment agent and edits no issues.

Each completion gets a separate bounded reassessment cycle, not a child of the
closed issue attempt. A cycle permits **two Jev requests and one coder run**:
initial decision and, only on `reassess`, proposal assessment. Reserve each request
against the cycle and existing global allowance atomically. Existing 30-second
request timeout/no transport retry applies; failed or indeterminate requests stay
charged. Cache by cycle ID, purpose/rubric/model and complete semantic evidence
fingerprint, including the distinct completion ID. Never treat an unknown request
outcome as an unused reservation after restart.

Add `RepositoryPolicy.ReassessmentMinutes`, default 30, integer range 1–120;
record the admitted value explicitly. Reassessment uses current sponsorship,
operator/environment authority, prepared Project, coder profile/account and the
existing time accounting; its active-time allowance begins at agent admission.
No funds are taken from or restored to the completed delivery root. There is no
correction run, native child spawning or autonomous replacement run for this
phase. A stale/invalid result, exhausted allowance or indeterminate provider result
holds the event. Existing explicit authorized Retry may create one successor cycle
for that same event after prior calls/runs/operations are reconciled. It requires
current authority and explicit expected prior-cycle identity; command replay
reuses the successor. No ordinary poll, input edit or policy edit refills a cycle.
A fresh cycle retains all earlier receipts and committed native effects.

Extend `ExecutionSubject` with exactly one matching payload for kind `completion`:
`CompletionSubject {completion_id, cycle_id, inventory_digest}`. Its subject key is
`completion:<completion_id>`. Add `reassess` phase, accepting only this subject and
the coder role. It uses a fresh attempt root and no `ParentAssignment` or
`PublicationAssignment`. The common repository slot/admission boundary still
applies. This is the explicit exception to investigating only the selected issue:
it reviews the inventory but does not implement any issue. Capture tracked
source/index/HEAD before and after as for other planning phases; mutation refuses
the result and preserves evidence. Do not claim an OS-enforced read-only sandbox.

Use the strict version-2 result union with a `ReassessmentReport` payload:

- `completion_id`, `cycle_id`, `inventory_digest`, `source_commit`, preparation
  digest and local input digest, all exact assignment bindings;
- `inspected_refs` using D03 source-reference shape/bounds, and a <=4 KiB summary;
- `retained_issue_indexes[]`, sorted/unique, and `intent_changes[]`, sorted by
  intent ID, at most 64, each containing `{intent_id, expected_revision,
  scope_digest, tasks, reason}`. `tasks` is the full replacement D02 graph, not
  executable implementation tasks; models cannot choose native issue IDs;
- `coverage[]`, at most 64 entries, each `{intent_id, outcome_id, task_keys,
  delivered_acceptance_ids}` accounting for affected accepted product outcomes.
  `outcome_id` names an effective accepted product-idea ID in that intent's exact
  expected revision, not a task-outcome ordinal. Proposed task keys must include
  that idea in `IdeaIDs`; a delivered acceptance must bind a historically accepted
  task covering that idea and have verified completion. Validate all affected
  effective ideas exactly once; D02 dependency outcome indexes retain their
  separate meaning;
- `attention[]`, at most 64 `{issue_index, reason, evidence_refs}` entries for
  necessary changes outside current authority or unresolved facts.

D03 text/path/reference limits and total 64 KiB result apply. For `reassess`
only, the existing prompt field permits **320 KiB**, containing the exact complete
<=256 KiB inventory inline plus bounded instructions. Other phases retain 64 KiB.
Update the Go Project validator, Rust factory/terminal records, both CLI launchers
and request/role validation consistently, binding this exception to the recorded
phase and completion subject. Use existing prompt stdin/file delivery, never
command-line arguments; check the enclosing encoded request cap before launch.
There is no new artifact transport or caller-supplied path/URL. Persist and hash
the exact prompt through the existing assignment custody. Every inventory issue must appear exactly once in
`retained_issue_indexes` or the deterministically derived changed/retired issue
set from the proposed graphs. Every referenced issue must belong to that sealed
inventory or its explicit delivered context. Missing coverage refuses. An empty
change set is valid, including after Jev requested investigation; agent inspection
may establish that no changes are needed. `attention` keeps the event held and
routes to current product/native issue actions; it does not silently drop work.

### Validation, adoption and recovery

The coder inspects the merged result, repository source and full issue inventory,
then proposes one coherent revision set. Deterministically validate bindings,
coverage, bounds, DAG, outcome references and current authority before another
paid call. Reuse existing intent/graph owners; do not introduce a workflow service
or direct agent issue-edit token. The permitted delta is changes to idle,
factory-generated tasks within their existing accepted product scope: clarify,
split/combine, add missing in-scope tasks, remove redundant tasks and revise
necessary dependencies. Preserve objective/accepted ideas/attribution/MaxIssues;
new product assumptions, scope expansion or changes to manually controlled work
require existing product review. Do not transfer outcomes between intents or
invent cross-intent dependency editing. External dependencies remain fixed.

**Protected and unchanged tasks retain native issue ID, acceptance ID, content,
state and outgoing dependency edges.** Protected means delivered, active, PR-linked,
manually changed since the recorded generation, or not factory-authorized. Keep
protected tasks unchanged in replacement graphs. A materially changed idle task
keeps its native issue ID where existing diff rules permit and gets a new exact
acceptance; a new task gets a new issue only through the native graph receipt.
Retired idle tasks receive the existing supersession treatment and lose dispatch
authority; do not delete/close issues to pretend completion. Account for retired
outcomes through retained/new tasks or exact verified delivery evidence.

Do not regenerate acceptance IDs for completed/unchanged bindings merely because
an intent revision advances: prerequisite completion compares exact acceptance
identity. “Unchanged” also requires identical prerequisite acceptance IDs and
selected outcome routes. Changing an idle prerequisite acceptance makes its idle
dependents materially affected transitively: recompute their exact acceptance
bindings in topological order, even if their prose/native edges are unchanged.
Refuse a proposal that would change a protected task's pinned prerequisite route;
retaining the native edge alone is insufficient. Copy frozen bindings without invoking `AdmitAcceptance`, which can
cancel active publication/merge work. A copied acceptance never blesses later
human edits: ordinary content-freshness checks still block that issue. Update
`GeneratedIntentCurrent` to test for **existence of a current
accepted-head binding**, not arbitrary `LIMIT 1` over historical bindings.
Invalidate current selection/investigation pointers only for materially affected
idle work and dependency closures. Preserve immutable historical plans/receipts.

Preflight **every** per-intent native packet before accepting any effects: current
SDK limit is 1–64 total creates + edits + dependency changes, encoded <=48 KiB.
A no-effect graph uses no native operation. Do not mistake 64 tasks for 64 changes,
chunk an oversized packet or rely on native closed-issue checks to protect edges.
The Soda validator freezes protected outgoing edges explicitly.

With no attention/refusal, ask Jev `post-completion-proposal-v1`, exactly
`accept` / `needs_attention`, over the original complete evidence, proposal and
validation report. Bound the **entire encoded request** to 384 KiB, including
state, criteria and envelope. The validation report is <=16 KiB with exact fields
`inventory_digest`, `proposal_digest`, `scope_digests` (intent/digest pairs),
`mutable_issue_indexes`, `protected_issue_indexes`, `coverage_digest`, and
`packet_digests` (intent/digest pairs); sorted unique lists, no freeform narrative.
Reuse the bounded Choice request path and reject size excess before reserving
the second call. The first complete request remains <=320 KiB with <=256 KiB
semantic evidence. No selective omission to meet either bound.
Acceptance requires selected `accept`, option probability
>=0.80 and confidence >=0.80. The rubric checks accepted-scope preservation,
complete outcome coverage, necessary dependencies and whether the revision
addresses the observed reason for reassessment. It also assesses no-change
reports. A rejected/uncertain proposal waits for intervention; no automatic
rewrite/rejudge loop. Raw answers and supplied criteria are visible, not invented
model explanations. Thresholds are product policy; native semantic cases must
exercise them without resampling preferred answers.

Stage the accepted proposal as immutable, inert data in the completion-cycle
journal, including exact successor revisions, native packet digests and readback
expectations. Do not advance all intent heads before native operations succeed.
While the repository fence remains, process intents in ID order: recheck all
unapplied inputs/authority; CAS and activate one successor through existing intent
adoption/generation; bind generic `RecoverIntentGraphs` pickup to this approved
journal step so it cannot run a later staged revision. A crash between head CAS
and packet creation resumes this same step. Reconcile its one immutable native
operation and exact
readback/adoption; then advance the durable cursor. Preserve unchanged/protected
bindings explicitly when adopting. Other product edits remain permitted but make
incompatible remaining effects stale. No external call inside a SQL transaction.

This is **not** a cross-intent atomic native transaction. A committed prefix stays
committed if later work fails; the journal records it, the fence stays closed,
and recovery first observes/reconciles that prefix. A lost reply never starts a
second operation. Unknown effect, withdrawn authority or incompatible human edit
cannot cause blind replay, rollback or acceptance of stale remaining effects.
Before effects, any unrelated semantic change invalidates the whole proposal;
after a prefix, normalize only the exact expected recorded effects, with every
other change still invalidating remaining work. Explicit Retry starts from the
actual observed state, preserving the committed prefix as history. A head that
was activated but not fully materialized cannot be abandoned: before any release,
each previously activated step must be confirmed adopted/read back or explicitly
superseded by a validated current-head replacement through existing product
revision/adoption. Reconciliation of a failed operation alone is not that proof.
A successor cycle's `continue` cannot release this residual state; it remains a
visible adoption wait until repaired, preserving already committed effects.

After all graph effects and local bindings are confirmed, recheck freshness and
mark this event released with its final inventory digest. A no-change accepted
report releases without graph mutations. If later completion events exist, process
them before selection. Otherwise invalidate the old priority snapshot, enqueue
fresh readiness, and select through D02. No reassessment result can directly pick
or launch the next implementation issue.

Persist this in existing factory store/control owners: `factory_completions` for
unique events/repository generation and `factory_reassessments` for immutable
cycles, bounded evidence, decision receipts, report and adoption cursor. Index
pending events by repository/generation. Keep native graph effects in existing
`factory_intent_generations`; reference them rather than duplicate their ledger.
Use cycle states `collecting`, `deciding`, `reassessing`, `assessing`, `applying`,
`released`, `attention`, `superseded`. Only durable sealed inventory advances to
`deciding`; valid continue can release; valid reassess admits the sole run; valid
report advances to `assessing`; accepted changed report enters `applying`, while
accepted no-change releases. Every release also checks residual prior adoption
and pending event ordering. Failures enter `attention` with the exact resumable
step/operation retained. Observation/reconciliation can finish recorded effects;
only explicit authorized Retry supersedes an unresolved terminal cycle.

Extend existing Work/Spaces/API read models with completion, gate decision,
assessment progress, exact proposal effects/receipts and the wait/retry reason.
Read endpoints do not spend money. No new dashboard, scheduler or recurring timer.

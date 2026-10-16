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
    Kind   string         // "issue" or "intent"; exactly one matching payload
    Issue  *IssueSubject
    Intent *IntentSubject
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
across both kinds. Retain one allowance root across revisions of the same active
attempt; immutable subject snapshots on individual assignments name exact inputs.

`Assignment.Phase` is one of `decompose`, `investigate`, `implement`, `review`,
`correct`. Only `decompose` accepts an intent subject. Only `review` uses the
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
implementation receipt, review report or correction report. Completed decomposition
and investigation forbid a publishable candidate. Non-completed results may retain
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
| Agent prompt/result | Keep existing 64 KiB each. Never silently truncate a structured result or required evidence. |
| Execution manifest/events | Manifest <=64 KiB; journal <=512 events/8 MiB, reserving two terminal events; event <=16 KiB; checkpoint metadata <=64 KiB/task and <=64 tasks/run. Logs retain existing separate bounds. |
| Review / findings | 64 findings/report and at most 64 outstanding findings; summary <=4 KiB, combined prose/finding <=4 KiB, encoded native review payload <=48 KiB. |
| Native review receipt/history | Review-specific receipt cap 16 KiB; retain 1 MiB review-history cap, counting full encoded work plus reserved receipt capacity. Other operation receipt limits stay unchanged. |
| Jev delivery evidence | Encoded semantic state <=256 KiB, including complete current findings, corrections, relevant diff and checks. Excess is `delivery_evidence_too_large`, not selective omission. |
| Merge authorization history | At most 32 immutable generations per publication; reaching the cap is a visible intervention. Never create another PR to escape it. |
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
release. Keep existing readiness enqueue in that completion transaction.

## D10 — Visible state and failure routing

Expose phase/attempt, plan revision, tasks/lanes, verified checkpoint commits,
native PR/comments, correction dispositions, round, raw/effective Jev choice and
precise waits through current Work/Spaces read models. Show choices and supplied
criteria/evidence; do not invent a textual explanation attributed to Jev. Read
models never call a provider or mutate workflow state.

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
| T30 Dependency release | FW25; serving/prerequisite tests | One parent merge cannot release two-parent join; both verified original closures do; join unlocks descendant; reopen/stale closure refuses. |
| T31 Views | FW23/FW25; plan API + current frontend tests | Real persisted phases/tasks/lanes, commit/comment links and effective Jev waits render; read endpoints spend nothing; native CLI text streams without screenshot transport. |
| T32 Actual journey | FW25/FW26; existing serving/native drivers | Real prompt entry → graph → Jev selection → investigation/plan → native parallel tasks/checkpoints → PR → inline review → correction → another review → Jev acceptance → verified merge/unlock. Also clean-first-review and direct-after-fix acceptance variants. |

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

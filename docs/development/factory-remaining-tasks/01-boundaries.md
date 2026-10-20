# Trustworthy boundaries

[Executable task map](../factory-remaining-tasks.md#task-list) owns the current
deliverables and output dependencies. This packet preserves the parent contracts,
technical source owners, accepted evidence and additional native/installed gates.
The Rxx.y links below identify the execution units; parent input lists describe
full qualification and do not require waiting for whole parents before using
applicable outputs. Recorded baseline and landed slices retain their actual
scope. Implementation notes support the execution map rather than define a
second schedule. This documentation restructuring starts no runtime work.

<a id="r01"></a>
## R01 — Accept the retained inputs and freeze the first work packets

**Accepted reference:** retained scope; no new execution task.

**Work:** Review. **Owner lane:** Coordinator. **Accepted inputs:** none.

**Current:** R01 retained-input/freeze and executor/lease handoff accepted. All run writers/commands are released; no active orphan49 worker or held slot remains. Original input manifests and scoped acceptance stay in the closed checklist.

**Deliver:** Review exact current preimages, patches and meaningful proposed checks. Accept useful slices or return precise defects. Reconcile orphan49, writer/resource custody and the protected dirty paths. Freeze exact files, shared contracts and next evidence for the first assignments.

**Accept only when:** Every retained draft has an explicit disposition and owner; no unreviewed patch is called integrated. Current identities, commands, source hashes and exclusive writers are recorded. Bind the actual executor and next file/resource lease: the human's direct Codex handoff permits that owner without waiting for an absent Muse coordinator. A Muse assignment still requires its supported continuation and actual receipt. Further design work is limited to an identified unresolved seam.

**Principal source owners:** [internal/store/factory_intents.go](../../../internal/store/factory_intents.go), [internal/factory/control](../../../internal/factory/control), [cmd/soda-project-maintain/src/command.rs](../../../cmd/soda-project-maintain/src/command.rs).

<a id="r02"></a>
## R02 — Repair the Fountain CI cleanup failure

**Accepted reference:** retained scope; no new execution task.

**Work:** Repair and qualify. **Owner lane:** C. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Current:** Focused CI cleanup repair 56a1c17bb0 independently accepted with pinned Go tests. Full current CI/native qualification remains separate in R03/R47/R52.

**Deliver:** Finish the smallest correct test/indexer lifetime repair. Observe actual worker retirement before close, preserve assertions and account for background population. Correct the blocked-handler witness and retain any async-to-sync coverage change explicitly.

**Accept only when:** A meaningful negative survives a shutdown timeout without authorizing Close; the retired positive is synchronized. Focused original tests and witness pass on exact candidate bytes with no leaked owned work. Diagnose the existing full-run result before a bounded rerun; do not infer harmless flakiness.

**Principal source owners:** `Fountain:modules/indexer/issues/indexer_test.go`, `Fountain:modules/queue/workerqueue.go`, `Fountain:modules/queue/workergroup.go`.

<a id="r03"></a>
## R03 — Qualify the current extension, SDK and background-authentication boundary

**Execution tasks:** [R03.1](../factory-remaining-tasks.md#r03-1).

**Work:** Qualify and repair gaps. **Owner lane:** C. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Native extension/SDK/background authority qualification; source and 29 real-PG cases accepted. Historical run closed; current user instructions govern execution. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind the actual SDK, host, Soda package and consumer revisions. Reuse valid independent-consumer/lifecycle/authentication evidence, then close changed install/replace/enable/disable, required-package, peer/installation/actor/token-generation and isolation cases.

**Accept only when:** An independently built current consumer uses supported capabilities; native package policy and unattended calls enforce exact authority after logout, rotation and restart. No direct forge database access or browser credential substitute.

**Implementation notes:** Start at `internal/forgejo/background.go` and `internal/web/server.go`, then the pinned Fountain SDK/extension lifecycle owner. Bind one independently built consumer to actual host/package revisions and exercise required-package installation/background authentication; then replacement, disable, logout, rotation and restart on those bytes. R02's focused repair is already available; blanket Fountain CI is not a prerequisite. Release exact supported authority/lifecycle behavior to R04/R44. Reuse retained source/29-PG-case evidence; run the missing native cases and repair only a demonstrated changed join. Missing fixture/package access holds that case, not a rewrite of the accepted SDK.

**Principal source owners:** `Fountain:sdk`, `Fountain:services/extensions`, [internal/web/server.go](../../../internal/web/server.go).

**Additional runtime/qualification gates:** [R02](01-boundaries.md#r02). Source-only work cannot release these gates.

<a id="r04"></a>
## R04 — Close native writer-domain and conditional-operation evidence

**Execution tasks:** [R04.1](../factory-remaining-tasks.md#r04-1).

**Work:** Qualify and repair gaps. **Owner lane:** C. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Complete native writer-domain and four conditional-operation recovery witnesses. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Check current participating ref/repository, authority, issue/review and Actions writers, callbacks and deferred completion against the existing reservation owner. Reuse earlier valid proofs; repair only uncovered callers. Cover all four conditional kinds and exact offline recovery.

**Accept only when:** Exact effects, native completion and cancellation remain distinct through contention, restart and lost replies. Ordinary work is retained, uncertain writers stay fenced, and no recovery path releases an owner without whole-domain quiescence. No ordinary API bypass or parallel operation ledger.

**Implementation notes:** Consume R03's installation/actor/token binding. Start with `internal/forgejo/merge.go` and the reservation protocol in testing.md: one real merge writer paused at guarded admission, competing ref/policy mutation, cancellation and controller death. Then apply the same recovery path to publication, review and issue-graph operations. Release attributable committed/not-observed/indeterminate receipts and quiescence observations to R05/R19/R26; uncertainty retains its fence. Reuse reservation tests, then exercise the four operation kinds and uncovered native writers. No new ledger or timeout-based release.

**Principal source owners:** `Fountain:services/extensions`, [internal/forgejo](../../../internal/forgejo), [internal/factory/control/publication.go](../../../internal/factory/control/publication.go), [internal/store/factory_merges.go](../../../internal/store/factory_merges.go).

**HB14 operator handoff finding (20 October 2026):** The public recovery
procedure can identify the native `admin native-operation status/recover`
subcommands, but the packaged appliance has no documented offline launch
invocation supplying their exact server image, user, configuration, database
network and retained mounts. `soda-forgejo-domain stop` removes the running
container; the server binary is `/app/gitea/gitea` in that image, not a host
`forgejo` executable. Complete this existing recovery handoff and observe it
with the whole writer domain inhibited. Preserve the same AppDataPath/offline
marker, database and repository storage; do not restart the web entrypoint
merely to obtain a shell. Update the operator guide and O08 with the supported
invocation after that witness. This is a missing execution context within R04,
not a new recovery authority or a substitute for its native-effect evidence.

**Additional runtime/qualification gates:** [R03](01-boundaries.md#r03). Source-only work cannot release these gates.

<a id="r05"></a>
## R05 — Qualify authoritative snapshots, pagination and native graph packets

**Execution tasks:** [R05.1](../factory-remaining-tasks.md#r05-1).

**Work:** Qualify and repair gaps. **Owner lane:** C. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Qualify native paged snapshots/graphs; source collectors, brackets and validation accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Prove native behavior and provenance, hidden and later-page evidence, atomic revision bracketing, complete issue/comment/review/check inventories and bounded conditional graph edit/create/dependency packets.

**Accept only when:** Edits/reverts, delete/recreate, hidden records and concurrent insertion cannot become false current evidence. Full paged conversations and graph receipts replay exactly; stale, oversized or protected edits refuse before partial effects.

**Implementation notes:** Consume R03 authority and R04's idle/revision/reservation observations. Start in `internal/forgejo/snapshot_repository_issues.go`, `snapshot_review_threads.go` and `issue_graph.go`. Qualify complete current issue/conversation/check pages, including insertion between pages and hidden records; then one bounded graph create/edit/dependency packet with lost-reply readback. Release sealed revision/provenance-bound inventory and exact graph receipts to R13/R14/R20. Existing collectors/brackets stay accepted; extend only changed caller tests and native witnesses. Incomplete inventory is a wait/error, never an empty successful graph.

**Principal source owners:** `Fountain:sdk`, [internal/forgejo](../../../internal/forgejo), [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go).

**Additional runtime/qualification gates:** [R03](01-boundaries.md#r03), [R04](01-boundaries.md#r04). Source-only work cannot release these gates.

<a id="r06"></a>
## R06 — Repair durable importer acquisition and process retirement

**Execution tasks:** [R06.1](../factory-remaining-tasks.md#r06-1).

**Work:** Implement and prove. **Owner lane:** A. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Native post-crash importer/descendant retirement; 7be597d7 source/cleanup accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Qualify the accepted acquisition-custody implementation through post-crash native consumers. Preserve read-only preview, exact invocation identity, pinned signal authority and fenced uncertain prefixes. Recover/rollback/boot/retry must consume the same custody state.

**Accept only when:** Real bounded native witnesses cover leader exit, surviving descendants, timeout, lost identity and interrupted persistence. No signal targets a reused identity; no success/retry releases custody while owned descendants may execute. Empty tests, numeric-PGID-only recovery and timeout inflation cannot pass.

**Implementation notes:** Start at `importer_custody.rs::{acquire,launched,reconcile,retired}` beside `command.rs::run_group_observed`. Source tests already exercise leader/descendant and persistence failure. Run the missing post-crash witness through real maintenance with a disposable importer and retained custody directory, then recovery/rollback/boot consumers of that state. Cover leader gone with descendant alive, unavailable identity and deadline expiry. Release observed whole-process retirement or fenced uncertainty to R41/R45. Change only a demonstrated faulty join; no second supervisor, PID-only adoption or replay of completed source repairs.

**Principal source owners:** [cmd/soda-project-maintain/src/command.rs](../../../cmd/soda-project-maintain/src/command.rs), [cmd/soda-project-maintain/src/release_extensions.rs](../../../cmd/soda-project-maintain/src/release_extensions.rs), [cmd/soda-project-maintain/src/release_operation.rs](../../../cmd/soda-project-maintain/src/release_operation.rs), [lib/host/src/project/admission.rs](../../../lib/host/src/project/admission.rs).

<a id="r07"></a>
## R07 — Prove supported native worker control on the installed harnesses

**Execution tasks:** [R07.1](../factory-remaining-tasks.md#r07-1).

**Work:** Native proof. **Owner lane:** A. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Reuse the landed native worker-control proof and release it to the missing isolation/transport/scheduling consumers. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Exercise supported child start/observe/result/cancel and real overlap on each admitted exact harness. Bind model/account/reasoning/Contributor, capacity, deadline, parent identity and terminal observations. Resolve unsupported behavior with a concrete Codex decision.

**Accept only when:** Two independent native children overlap; a dependent can proceed while unrelated work continues. Parent stop/death retires children before credential return. No separate process pool, extra provider lease, silent downgrade or serial substitute. The Muse run continuation-tool limitation is tracked separately from this product boundary.

**Implementation notes:** Reuse the installed harness behavior already exercised by the existing native child driver in `lib/host/src/terminal/factory/{codex,muse,children.rs}`; preparation needs no model call. Proof needs each exact binary/configuration plus a finite authorized account/model/reasoning/Contributor reservation, child limit, deadline and spend bound. For a changed or still-unproved boundary, exercise independent children and a dependent successor through `TaskRunner`, cancellation and parent-death retirement. Release native reference/result/control semantics to R08/R09/R16. Unsupported behavior returns a concrete capability decision to the owner; never substitute a process pool or serial run for required native overlap. Do not compare or gate on tool/dependency version numbers.

**Native-trial proof (landed):** `trial_overlap.rs` drives real children of both installed harnesses through `ChildTable`: simultaneous overlap with exact output, cancel and parent-stop retire clean; a dependent successor orders through `TaskRunner` journal receipts (predecessor `committed` before successor `start_intent`); a dependent proceeds while unrelated work continues. An abandoned child completes with exact output and no retire ever issued — retirement after death is observed downstream, no kill mechanism exists in this layer. Staged auth is destroyed with retried scratch removal. Remainder is release of reference/result/control semantics to R08/R09/R16 consumers.

**Principal source owners:** [lib/host/src/terminal/factory](../../../lib/host/src/terminal/factory), [lib/host/src/factory/tasks.rs](../../../lib/host/src/factory/tasks.rs), [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs).

<a id="r08"></a>
## R08 — Prove parent attenuation and OS-enforced role isolation

**Execution tasks:** [R08.1](../factory-remaining-tasks.md#r08-1).

**Work:** Native proof. **Owner lane:** A. **Accepted inputs:** [R07](01-boundaries.md#r07).

**Recorded baseline:** Native parent attenuation and OS-enforced role isolation. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Authenticate actual native request peers and attenuate parent authority to children. Protect credentials, journal, Git/config and authoritative source with real OS boundaries; permit only each role's intended edits, checks and scratch.

**Accept only when:** Real forbidden reads/writes and parent-only calls fail, while permitted work succeeds. Forged events/symlinks and post-cancellation calls refuse. Readonly profiles and Builder/Fixer restrictions are tested through their actual shells, not boolean/path fixtures.

**Implementation notes:** Consume R07's real parent/child references. Start at `lib/host/src/factory/identity.rs` and `cmd/soda-project-terminal/src/factory_roles`. Bind native peers to attenuated role mounts/credentials; drive forbidden read/write/parent-call attempts through actual shells alongside permitted work for each relevant role. Cover symlink escape, forged child events and post-stop reuse. Release verified peer-to-role mapping and OS restrictions to R09/R30. Existing isolation fixtures support the native negatives; a profile without enforcement stays unavailable rather than relying on prompts or caller role strings.

**Scoped hold:** admission logic and all runnable native negatives pass (12 permissions tests green, including worker attenuation, parent-completion refusal and forged-journal/symlink rejection). The remainder needs real OS role accounts, which do not exist on this host — operator provisioning, not a source change.

**Principal source owners:** [lib/host/src/factory/identity.rs](../../../lib/host/src/factory/identity.rs), [lib/host/src/terminal/factory](../../../lib/host/src/terminal/factory), [cmd/soda-project-terminal/src/factory_roles](../../../cmd/soda-project-terminal/src/factory_roles).

<a id="r09"></a>
## R09 — Prove supported instruction placement and private MCP transport

**Execution tasks:** [R09.1](../factory-remaining-tasks.md#r09-1).

**Work:** Native proof. **Owner lane:** A. **Accepted inputs:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08).

**Recorded baseline:** Installed CLI instruction/configuration and private MCP transport proof. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use supported trusted configuration/instruction channels for installed Codex and Muse harnesses, preserving base instructions. Prove actual CLI-to-stdio-relay-to-private-socket-to-controller discovery, admitted read/finish and receipt recovery.

**Dependency boundary:** Instruction/configuration inspection can proceed alongside R08. Actual private transport proof uses R08's accepted peer and OS boundary. This proves the supported transport against a bounded fixture; production MCP authority and lifecycle are composed in R30, so R09 does not depend on the later R30 acceptance.

**Accept only when:** Each admitted harness has real configuration/request-role evidence, fixed mounts and authority binding. Unsupported profiles stay unavailable. No public HTTP fallback, user-prompt authority substitute, copied provider secrets or claim based only on --help.

**Implementation notes:** Inspect supported instruction/configuration loading in `prompt_entry.rs` and the Codex/Muse adapters alongside R08. After R08 releases peer/role enforcement, exercise `mcp_relay.rs` through actual CLI discovery, one allowed read, formal submission and lost-response receipt recovery against a bounded controller fixture. Release the exact private transport contract to R30. Preserve base instructions/fixed mounts; wrong peer/profile/expired invocation refuses. Existing relay/P6 fixtures support diagnosis but do not establish an actual CLI round trip. R30 production activation is the later consumer, not a prerequisite cycle.

**Principal source owners:** [lib/host/src/terminal/factory/mcp_relay.rs](../../../lib/host/src/terminal/factory/mcp_relay.rs), [lib/host/src/terminal/factory/codex](../../../lib/host/src/terminal/factory/codex), [lib/host/src/terminal/factory/muse](../../../lib/host/src/terminal/factory/muse), [internal/web/api/factory_mcp_p6.go](../../../internal/web/api/factory_mcp_p6.go).

<a id="r10"></a>
## R10 — Complete current phase, subject and result parity

**Execution tasks:** [R10.1](../factory-remaining-tasks.md#r10-1), [R10.2](../factory-remaining-tasks.md#r10-2), [R10.3](../factory-remaining-tasks.md#r10-3).

**Work:** Audit and integrate. **Owner lane:** B. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Finish trusted ingress/consumer and D15/D16 subject wiring; accepted parity and terminal transport slices stay closed. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Trace every admitted phase through Go controller/Store, actual Rust host, CLI result ingress and settlement. Finish missing typed payloads/consumers together, retaining phase bounds and existing owner policy.

**Owner-requested Library Admin addition:** Define a typed collection-maintenance
subject and result for the eighth profile in the
[full design](../factory-remaining-tasks.md#full-worker-design-integration).
Bind objective, collection/operation grants, exact input revisions and allowance
without inventing a Git issue, delivery or code candidate. Carry the same identity
through Go dispatch/Store, actual host admission and finish/settlement mirrors.
Release the contract to R11/R30/R31/R32. Extend the existing positive/refusal
matrix for wrong profile, cross-collection scope, candidate-bearing maintenance
results and stale revisions; prior accepted production-role parity remains valid
at its original scope and does not qualify this new subject.

**Accept only when:** A complete positive/negative phase-role-subject-result matrix passes. Extra/cross-phase payloads, candidate-bearing planning results, stale identities and wrong manifests refuse. Actual host owners are inspected; absence in the operator CLI is not a claimed execution gap.

**Implementation notes:** The types already exist, including `internal/factory/verification.go::{DeliverySubject,DeliveryRepairSubject}`. Start at `assignment_result.go`, `dispatch_result.go::derivePhasedAttemptResult` and host `finish.rs`. Trace one repair and one verify result from trusted host ingress through Store settlement; close missing subject/cause/manifest binding there. Release the same envelope to R11/R32/R38/R40. Extend the existing parity matrix for these consumers: wrong subject/profile, extra candidate, stale run and manifest mismatch refuse before effect. Preserve accepted Go/Rust parity and terminal transport; coordinate changed mirrors through existing schema owners.

**Source-complete status:** the ingress-to-settlement path is traced and closed in source — host `finish_run` carries raw output only, control derives and binds (phase/subject/cycle/inventory/head, repair-chain settlement with unproven-to-blocked refusal, all matrix-green), store finishes with shape/freshness/immutability checks. Failure-digest linkage stays open by design (no producer equation; cause semantics belong to R37-R40) and verify/repair assignment production awaits R11 dispatch. Remainder is PG/native qualification of the matrix plus native consumers.

**PG matrix proof (landed):** `dispatch_settlement_phased_test.go` runs the refusal classes end to end on real PG — cross-phase and legacy-candidate results finish blocked-human with no recorded candidate and a stable single charge; a stale direct finish refuses with exactly one terminal success. Manifest-mismatch end to end still wants a verify-root staging helper; pure derivation refusals and the store freshness guard stay green. Native consumers remain open.

**Principal source owners:** [internal/factory/assignment.go](../../../internal/factory/assignment.go), [internal/factory/assignment_result.go](../../../internal/factory/assignment_result.go), [internal/factory/result.go](../../../internal/factory/result.go), [internal/project/factory.go](../../../internal/project/factory.go), [lib/host/src/factory/run.rs](../../../lib/host/src/factory/run.rs), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs), [lib/host/src/factory/receipt.rs](../../../lib/host/src/factory/receipt.rs).

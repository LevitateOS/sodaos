# Mechanism-deletion audit — Spaces, Runners and Tailnet

**Source baseline:** `7bd8f3b76359dcb0aa73c23ff4d7d4ca02526681`.
**Scope:** all 28 slices in the requested inventory, including their callers,
requirements and supporting tests. The line-count target was withdrawn; there is
no deletion quota or measured saving. A preliminary size estimate is recorded below.

This is a removal-focused audit, not a behavior-preserving refactoring review.
Some of the largest mechanisms exist because earlier requirements demanded them.
My recommendation is to **delete those requirements and their implementations**, not
move the implementations into smaller files. Where that changes product behavior,
the consequence is stated below. Recommendations are not implemented changes or
new native/retained-state permissions.

The [refactoring plan](refactoring-plan.md) remains the implementation-decision
owner. Feature requirements still belong to their existing guides until a change
is selected. This review explicitly revisits protections that earlier reviews
said to preserve; their presence in a guide or test is not independent evidence
that they are proportionate.

## Principal conclusion

The largest excess is concentrated in two places:

- **Spaces treats a developer's shell as an expiring browser authorization lease.**
  That requires a second lifetime owner, recurring renewal, a root watchdog,
  multiple deadlines, cleanup receipts and browser preservation of uncertainty.
- **Tailnet treats short-lived enrollment and disposable runtime state as a
  non-replayable transaction that must survive every ambiguous failure.** It
  records attempts before provider work, refuses recovery within that run, and
  detects why systemd is stopping so it can preserve an ephemeral identity.

Those policies buy guarantees that are considerably stronger than the underlying
product needs, while introducing concrete failure modes: lost work, exhausted
terminal capacity, and networking that cannot recover after a failed key request.

**Runners is not hiding an equivalent large subsystem.** Its clearest excesses are
whole-inventory failure propagation and universal typed confirmations. Inventing a
runner scheduler or provider framework to delete would misrepresent this tree.

## Preliminary size estimate

If all recommended policy changes are accepted, the rough estimate is
**1,000–2,000 net implementation lines removed**, with **about 1,500** as the working
estimate. This uses physical source lines, including comments and blanks, but
**excludes tests and documentation**. Most of the reduction would come from terminal
lifetime/receipt machinery and Tailnet enrollment/restart bookkeeping.

The estimate allows for replacement native lookup/admission/cleanup code and does
not add overlapping findings together. Some smaller changes, particularly partial
runner inventory reporting, may add code rather than remove it. This is a judgment
estimate from the inspected source, not a measured patch or completed per-block
sizing ledger. No runtime code has been deleted, and the estimate is not a target
or implementation approval.

## Recommended mechanism deletions

### D1. Delete browser-owned shell lifetime and its renewal supervisor

**Slices:** 1, 3, 5, 8. **Verdict: delete; deliberate product-policy change.**

[Web ownership](../internal/web/terminal_sessions.go), especially
`ownBrowserTerminal`, keeps a native owner connection alive independently of the
browser. Every 15 seconds it performs fresh repository/session checks and renews
a native heartbeat. A failed check or stream ends ownership. The same file
implements active/30-minute retained/two-hour Keep/session-bound twelve-hour
lifetimes. [The Python owner](../internal/host/project_terminal.py) adds a root
`guard`, a locked lease file, watchdog notifications, per-second tmux checks and
per-second unit inspections around systemd's own supervision.

**Why delete it:** browser access and process lifetime are different concerns.
Revoking a browser's authority must stop its IO; it need not destroy the user's
editor or build. Currently a provider observation failure, dashboard restart or
helper restart can end the shell, not merely detach the browser. Ordinary project
SSH accounts and independent workloads do not acquire this browser lifetime.
The lease regime is therefore neither general project revocation nor necessary
for tmux persistence.

**Removal boundary:** retire Keep/Return/retention deadlines, detached authorization
renewal, the extra long-lived owner stream and the root lease/watchdog loop whose
purpose is enforcing those deadlines. Let the project-local systemd unit supervise
the owned tmux server. Remove the corresponding browser controls, deadline metadata
and deadline-derived attention warnings. Do not replace them with another lease
manager or durable web session registry.

**Behavior after removal:** closing the browser, signing out, authentication expiry
or a web-service restart ends browser access, not shell work. The shell ends on
explicit End, its own exit, or project Stop. Continued browser attachment still
needs authorization and a bounded response to authority loss. Abandoned shells can
remain running; that is the real resource tradeoff, not a hidden implementation
detail. Capacity must count actual owned terminals/in-flight creation, not rely on
a promise that all shells disappear when a browser login ends.

**Keep:** original actor/project/account binding; fresh admission and attached-access
checks; bounded startup and transport; fixed commands; exact attach-only tmux;
non-root execution; native cgroup-scoped End. The account validator and PTY bridge
inside the Python file are not deleted with the guard. Stock systemd also does not
provide tmux screen restoration by itself. `tmux -D` alone disables empty-server
exit; fixed startup/configuration still needs to arrange normal exit when the
session finishes, rather than leaving another permanently empty server.

**Affected evidence:** `terminal_resume_test.go`, `terminal_ids_test.go`,
`tests/build/test_terminal.py`, and the terminal/workspace browser tests. Port
continuity and denial tests to the new lifetime; retire assertions that logout or
Keep must kill/extend a native shell. The owning requirement is
[terminal integration](terminal-integration.md).

### D2. Delete the terminal attempt/receipt ledger and permanent unknown-slot custody

**Slices:** 2, 5, 7. **Verdict: delete with the native lifetime/lookup change in D1.**

[The browser transport](../internal/web/terminal.go) accepts a creation `request_id`
and allocates another terminal ID. [The registry](../internal/web/terminal_sessions.go)
then supports a separate attempt lookup, two identity searches, five-minute
cleanup receipts and `unconfirmed` reservations. Only a native `closed` receipt
releases a live slot. A failed initial native connection or lost cleanup result
leaves an entry after its owner goroutine has returned; there is no later native
reconciliation or expiry that releases that entry. Sixty-four such entries consume
the global bound even if their native processes are gone.

The browser carries this distinction through `new`/`pending`/`existing` locators,
`inspectOutcome`, storage, restoration and attention. Receipt absence is deliberately
never sufficient to forget the locator, even after the receipt's own expiry.

**Why delete it:** a short-lived in-memory receipt is being made the only acceptable
proof of native lifetime. Once it is lost, the system cannot recover by observing
its actual owner. The resulting permanent quarantine is not useful preservation
of a process or file.

**Removal boundary:** allocate one terminal locator before sending Create; treat it
as an untrusted identifier, not authority. Use the same exact ID for Create, lookup,
attach and End. Delete the second request-ID namespace, `/terminal-attempts`,
receipt maps/expiry, and indefinite cleanup reservations. Obtain current existence
and lifecycle from the exact owned native unit/session on demand. Keep a bounded
in-flight creation exclusion and single-writer admission; those are not a cleanup
history service.

**Consequence:** Soda stops promising retrospective proof of how a vanished terminal
ended. It reports current native presence/absence or unavailable observation. A
failed observation must not become invented absence, successful cleanup or an
automatic replacement. This requires a real native lookup, not deleting receipt
checks while retaining the current web-only registry.

**Affected evidence:** `TestTerminalUnconfirmedCleanupKeepsSlotAndRefusesReplacement`,
`TestTerminalCorrelationLostLocatorAndCapacityReservations`, receipt-capacity cases,
`terminal.test.ts`, `workspace.test.ts` and `spaces-layout.test.ts`. Preserve tests
that an exact missing attachment creates nothing and concurrent creates cannot
adopt or overwrite an occupied target.

### D3. Delete permanent browser-wide uncertainty lockout

**Slices:** 9, 10, 19, 24. **Verdict: delete, independently of D1.**

In [project controls](../frontend/spaces/sodaspaces-project.ts), a dispatched
mutation with an uncertain response sets `uncertain`. `blocked` disables subsequent
mutations and `canRestore` refuses restoration. A successful `refresh()` does not
clear it. [The workspace](../frontend/spaces/sodaspaces-workspace.ts) combines
`canRestore` across all mounted project controls, propagating that prohibition to
restoration of the larger surface.

This applies to a saved-public-key request as well as native provisioning. It is
not confined to replaying the uncertain operation. Yet a full new page constructs
new controls without the flag: this is not a durable exactly-once boundary.

**Removal boundary:** delete the sticky `uncertain` admission flag and its
cross-component restoration veto. Keep a scoped outcome notice, in-flight duplicate
suppression, page/actor retirement and the existing authoritative preconditions for
the next explicit action. A fresh key preview must still supply its current revision;
a reserved project must still refuse another Create.

**Consequence:** users can inspect state and explicitly perform an independently
valid action without operator intervention or a full reload. An uncertain previous
write remains uncertain; refreshing a view does not certify that write. No automatic
replay, rollback or new operation-journal service is proposed.

**Affected evidence:** `tests/frontend/drawer-controls.test.ts` explicitly tests
unknown Create lockout. Keep the no-recreation assertion at the real API/store
boundary; stop applying it to every unrelated browser action. Runners already
separates a persistent outcome notice from newly refreshed observations.

### D4. Delete the durable at-most-one-key-attempt-per-run journal

**Slices:** 13, 14, 15, 18. **Verdict: delete; change the recovery policy.**

[`EnrollRun`](../internal/tailnet/enrollment.go) writes `project.ActiveRun` and an
`attempt-PROJECT-RUN.json` record **before requesting an OAuth token or auth key**.
Any subsequent attempt with the same run marker is refused. The marker
also survives a missing attempt file. Phases are `key-requested`, `submitted` and
`unconfirmed`; none enables recovery by another attempt.

The `retry` policy action in [policy.go](../internal/tailnet/policy.go) changes the
project revision but leaves `ActiveRun` intact. [`StartTailnet`](../internal/host/tailnet_companion.go)
requires retained daemon state after any attempt and enrolls only for phase `none`.
Thus a credential/provider failure can leave this run unable to request another
key even after credentials are corrected. The tests explicitly require this refusal.

**Why delete it:** an uncertain auth-key issuance is not an uncertain persistent
project creation. An unconsumed Soda key is single-use, ephemeral and requested
with a five-minute expiry. Refusing all same-run recovery is a disproportionate
response to possibly having issued an extra unused short-lived key. The journal
does not establish exactly-once enrollment; it establishes at-most-one attempt,
including attempts that never reached enrollment.

**Removal boundary:** remove `runAttempt`, `RunAttempt`, `ActiveRun`, persistent
attempt files and phase-dependent restart refusal. Use the existing native
lifecycle/lock to exclude concurrent enrollment. Observe an existing daemon/node
before another attempt; reuse an existing identity rather than forcing login.
Permit a bounded explicit retry after a failed attempt. No automatic replay loop
inside the HTTP transport and no browser-triggered mutation on GET/focus.

**Consequence:** a lost provider response may leave another unused key until expiry.
If key consumption already registered a device, key expiry does **not** revoke that
device; existing-node observation and native logout/ephemeral expiry still matter.
This recommendation does not permit a second companion in a conflicting namespace,
ignore a changed target, or claim immediate provider deletion.

**Affected evidence:** `TestRunEnrollmentHasOneAttemptAndNoSavedSecrets`,
`TestRunEnrollmentFencesBindingIdentityAndUncertainty` and runtime restart cases.
Retain credential secrecy, bounded requests, exact target/binding, single concurrent
consumer and passive-read tests; replace the permanent refusal assertions with
bounded explicit recovery. The owner is the
[Tailnet plan](tailnet-integration-plan.md).

### D5. Delete the same-run ephemeral-identity preservation protocol

**Slices:** 16, 17, 18. **Verdict: delete; deliberate continuity tradeoff.**

The companion has a file-backed node identity in run-scoped host storage.
[`StartTailnet`/`stopTailnetRun`](../internal/host/tailnet_companion.go) distinguish a
companion-only restart from project Stop/restart or Off so the former preserves the
node while the latter logs out. `parentStopQueued` parses systemd's job table because
`ActiveState` alone does not distinguish those intentions. Current-run records,
previous-run comparisons and retained-state requirements support that distinction.

**Why delete it:** each project needs its own network identity while running. That
does not require the same ephemeral identity to survive a companion failure. This
additional continuity guarantee creates a restart-reason detector and failure
recovery protocol around native supervisors.

**Removal boundary:** retire Soda's promise to preserve the node across a companion
restart, and the job-reason parser/state-resumption branches that enforce it. Use
one ephemeral node lifetime per companion activation, with native supervision and
bounded termination. The existing Stage-1 research identifies upstream
`--state=mem:` as a candidate; its actual namespace/DNS/shutdown behavior still needs
native proof. Do not replace the parser with a new restart controller.

**Consequence:** companion restart can change the node/address, interrupt SSH or
service connections and create another provider enrollment. Where preauthorization
is not configured, device approval may be needed again. Those are the costs of
this deletion. If stable identity across daemon failure is a product requirement,
this deletion must not be disguised as equivalent behavior.

**Keep:** separate appliance/project identities; trusted companion filesystem and
secret isolation; original project/CID and live namespace checks; bounded orderly
logout; DNS restoration; exact-owned resource cleanup. Completed disposable runtime
resources should not become a permanent custody archive, but removal of any
already-retained resources still needs its own authorization. Persistent project
roots are not disposable companions.

**Affected evidence:** companion/runtime/files tests, including parent-job parsing
and same-run state expectations. Native restart, DNS and client reconnection tests
would establish the replacement's behavior; this audit did not run them.

### D6. Delete Tailnet's second project-creation reservation transaction

**Slices:** 9, 15, 25. **Verdict: delete the cross-feature transaction.**

The web layer already reserves a unique project in
[`apiCreateEnvironment`](../internal/web/environments_api.go).
[`ProvisionProject`](../internal/tailnet/project_runtime.go) adds a second
`reservation-PROJECT.json`, holds the enrollment-policy lock around stopped-container
creation, and publishes another project policy before
[the host caller](../internal/host/daemon.go) may start the container.
Failure retains the reservation and blocks another attempt. The reservation has
no successful-run consumer that makes it necessary as persistent product state.

**Why delete it:** optional network configuration has been inserted into the core
persistent-project creation transaction. A network-policy publication failure can
leave a created container unstarted and the web project unprovisioned. Network
failure should not require project recovery or recreation.

**Removal boundary:** keep the real project reservation and preflight the reviewed
network choice, but complete native project provisioning independently. Apply the
reviewed network selection using the normal exact-project policy operation and
report its outcome separately. Delete the network-specific reservation file and
policy-lock-held create callback. A changed binding must still refuse the network
selection, not silently choose a different network.

**Consequence:** Create can succeed with networking Off/unconfirmed and require an
explicit network retry. It no longer promises atomic project-plus-network-policy
creation. It must never retry/recreate the persistent project to repair networking.

**Affected evidence:** `TestProjectRuntimeRetainsFailedReservationsAndClosesAdmission`,
managed Create handler/native tests and the Create form's result handling. Preserve
unique repository reservation, original CID/profile and no-root-replacement tests.

### D7. Delete the automatic archive of superseded OAuth credentials

**Slice:** 13. **Verdict: delete the archive, not restricted credential storage.**

Every Save/Rotate in [policy.go](../internal/tailnet/policy.go) publishes another
immutable `credential-REVISION.json`; `policy.json` points at the active one. Runtime
reads follow that pointer. There is no production rollback/history-selection
consumer for superseded credentials. `TestTailnetPolicyRotationCASAndSecretProjection`
requires the old secret file to remain after rotation.

**Why delete it:** the runtime has acquired a credential-history retention policy
without a product feature consuming that history. It retains additional secrets,
requires reference/publication machinery and makes normal rotation look like
custody of every historical input.

**Removal boundary:** use one atomically published restricted active configuration
containing the credential and associated policy, with the existing lock/revision
check and sanitized public projection. Do not replace a two-file publication with
another two-file non-atomic update. Backups belong to authorized maintenance, not
an ever-growing live credential archive.

**Consequence:** no automatic local rollback to an old credential. Operators can
supply a replacement or use an appropriately scoped backup. This does not revoke
provider credentials and does not authorize deleting existing credential files.
The change is to future runtime storage behavior; retained-secret migration is a
separate target action.

### D8. Delete exact-release vetoes from live Tailnet management

**Slices:** 12, 16, 19. **Verdict: delete the runtime veto, not version selection.**

[`Management.observe`/`verifyCLI`](../internal/tailnet/management.go) and
[`ProjectStatus`/`ProjectCLIRelease`/`CheckDaemonRelease`](../internal/tailnet/project_status.go)
refuse anything other than `ManagementCLIRelease = "1.102.4"`. This includes passive
status projection, independently of whether the required response fields are valid.
The constant's own comment says it is not an installer pin.

**Why delete it:** a source-review release string is being used as live operational
authority. A version change can hide healthy status and disable management without
identifying an incompatible capability or response. That is not a substitute for
selecting and testing shipped artifacts.

**Removal boundary:** remove the exact-string admission chain and standalone version
probes whose only purpose is that veto. Keep pinned build/image inputs, bounded
native parsing, required fields, fixed supported commands and compatibility tests.
Unsupported shapes/operations still fail at their actual boundary. Do not add an
unbounded compatibility matrix, silently upgrade packages, or claim an untested
release is validated.

**Consequence:** a different version is no longer rejected solely for its number;
Soda attempts the concrete supported contract. `TestTailnetHostRevisionIncludesNativeIdentityAndCLIIsPinned`
and project release tests need to distinguish useful identity/shape assertions
from the veto being retired.

This does **not** select blind deletion of companion isolation checks or image
identity validation. Exact `CreateCommand` equality and terminal/unit-property
attestation are narrower review candidates: remove assertions about irrelevant
spelling/drop-ins only while preserving the security properties currently established
through them. They are not an independently proven whole-subsystem deletion.

### D9. Delete disposable workspace compatibility and malformed-cache custody

**Slices:** 7, 27. **Verdict: delete; retire the old browser storage formats.**

[`migrateLayout`](../frontend/spaces/sodaspaces-layout.ts), V1 fallback and old
`soda-terminal:ACTOR:PROJECT` imports in
[the workspace](../frontend/spaces/sodaspaces-workspace.ts) retain historical
presentation formats. Invalid current storage sets `storageWritable=false` for the
component lifetime, preserving its bytes rather than allowing the working layout
to be saved. Those bytes contain presentation locators, not files, transcripts,
credentials or native authority.

**Removal boundary:** support the current layout only; remove V1 and singleton-key
imports. Treat corrupt/obsolete layout as a resettable cache instead of protected
historical state. Retain byte/depth/identity bounds for untrusted browser input and
never infer terminal authority or Create from stored values. With D2, recover actual
terminals from their authorized native inventory, not a preserved pending receipt.

**Consequence:** users of an old layout must reopen/rearrange terminals; an old or
corrupt layout's only-known pending locator may be lost. It must not End a terminal
or automatically create a replacement. This is why terminal lookup and cache
retirement belong together, rather than silently discarding uncertainty in the
existing web-only registry.

**Affected evidence:** V1/legacy/corrupt-storage cases in `spaces-layout.test.ts` and
`workspace.test.ts`. Port useful identity, no-Create-on-restore and live-owner tests.
The old singular HTTP endpoint is only a 410 stub, not a compatibility subsystem
worth presenting as a large deletion.

### D10. Delete browser reauthorization preflight/postflight choreography

**Slices:** 5, 19, 23, 24, 26. **Verdict: delete the duplicate request protocol.**

Project mutations fetch `/api/session` and `/api/forgejo/me` before the actual
request. Workspace mutation helpers fetch another session. Runners wraps requests
in a fresh session fetch; Tailnet does it both before and after a request. The
actual [API guard](../internal/web/api.go), feature authorization and
[post-I/O Tailnet guards](../internal/web/tailnet.go) already enforce the session,
expected actor, CSRF/origin and applicable authority at the real boundary.

**Why delete it:** these extra browser requests cannot atomically authorize the
later request or publication. They reproduce server admission as UI choreography,
add round trips and create additional pending/failed-operation branches.

**Removal boundary:** obtain the session/CSRF context at page entry and send it with
the actual guarded operation. Keep the original actor, immutable target, request
lifetime and logout/page-retirement handling. Refuse stale credentials through the
actual server response; reconnect explicitly instead of silently adopting a new
session. Retain `/api/session` for bootstrap and real login/logout flows. Do not
create a generic browser authorization service to replace the individual preflights.

**Consequence:** a stale click may send a request that the server rejects; it no
longer promises zero HTTP mutation requests merely because an earlier browser
preflight could have detected the problem. It must still cause zero unauthorized
native effects. Navigating after dispatch does not undo an action. Move tests of
that authority to real handlers, while retaining frontend actor/lifetime, late
response and secret-scrubbing tests.

## Smaller policy mechanisms to remove

These findings are real, but should not be inflated into large subsystem deletions.

| Finding | Evidence and deletion | Consequence / retained boundary |
| --- | --- | --- |
| **D11. Whole-runner-inventory failure propagation** | `Native.List` in [native.go](../internal/runners/native.go) discards every earlier row when one `runnerView` fails. Missing client-version output also fails the list. `TestListDoesNotPublishEarlierRowsWhenAnotherRunnerIsUnreadable` explicitly covers a partial second runner directory hiding the healthy first runner. Delete the global completeness prerequisite for displaying/managing unrelated valid runners. | Report an unreadable target as unavailable and keep other validated rows usable. Do not silently omit broken entries, invent available capacity, or allow destructive operations on an invalid descriptor. Aggregate counts may need an unknown/partial qualification; this improvement may add representation code rather than produce a large net deletion. |
| **D12. Typed-ID confirmation for every runner lifecycle action** | [The UI](../frontend/runners/soda-runners-page.ts) and [handler](../internal/web/runners.go) require the same typed `confirm_id` ceremony for Start, Stop, Restart and Remove. Delete it for routine lifecycle operations. | Keep named targets and shared-impact warnings for Stop/Restart; retain strong confirmation for destructive Remove. Typing an ID is not authorization. Preserve configured-operator, CSRF and exact native target checks, and disclose the existing boot-policy effects. |
| **D13. Helper-wide serialization of read-only observations** | [Daemon admission](../internal/host/daemon.go) serializes `/inspect`, `/profile`, `/os` and `/connection` with long mutations; `startCreated` holds admission during its readiness wait. `TestAdmissionRemainsGloballySerialized` specifically requires independent `/inspect` requests not to overlap. Delete read-side participation in that global gate. | Reads can observe a transition and report unavailable/partial state; the mutex never made observations atomic against native root/systemd anyway. Retain cancellation/bounds and serialization needed by shared creation/mutation resources. Do not replace it with a scheduler, per-project lock framework or speculative parallel mutation engine. |

## All 28 slices — individual disposition

“Retain” means no supported **whole-mechanism** deletion was found, not that every
line is ideal. “Simplify” is not counted as a major removal finding.

| # | Slice | Verdict and concrete boundary |
| --- | --- | --- |
| 1 | Terminal lifetime policy | **Delete D1.** Shell lifetime should not be renewed by browser authorization. Revoke IO without destroying work. |
| 2 | Terminal uncertainty accounting | **Delete D2.** One preallocated ID and current native lookup replace the attempt namespace/cleanup-history monopoly. Keep in-flight exclusion and one writer. |
| 3 | Native terminal supervisor | **Delete the D1 lease guard, not the whole Python program.** Account binding, privilege dropping, PTY behavior and exact native cleanup have separate purposes. |
| 4 | Terminal transport chain | **Retain the privilege-crossing bridge.** The browser cannot directly access the root helper or container PTY. Removing JSON/base64 alone would be a protocol rewrite, not elimination of that boundary. D1 does remove the separate owner stream. |
| 5 | Browser terminal state machine | **Delete D1/D2/D10 branches.** Keep renderer lifetime, bounded IO, readiness, exact attachment, stale-callback exclusion and attach-only reconnect. A networked terminal still needs connection state. |
| 6 | Workspace layout machinery | **Retain.** Multiple simultaneous cross-project panes are requested functionality. One project's tmux cannot replace cross-container composition without introducing another connection layer. Flat DOM ownership prevents Lit moves from disposing xterm/socket owners; a generic docking framework would add machinery. |
| 7 | Workspace storage/compatibility | **Delete D9 and D2 dependants.** Keep one current, bounded layout and explicit native identity lookup. Layout preferences are not retained project data. |
| 8 | Attention/refresh | **Delete deadline-derived attention with D1; retain basic unread/connection indication.** `sodaspaces-attention.ts` is a small projection of real events, not an agent-status inference engine. No case for deleting all refresh/attention functionality was established. |
| 9 | Project creation/profiles/lifecycle | **Delete D6 and D3 coupling; retain real provisioning identity.** One persistent project per repository, original profile/CID, readiness and same-root lifecycle are not optional-network transactions. Broad profile deletion would remove selected product behavior. |
| 10 | Membership/SSH-key management | **Retain native account/key integration; delete D3/D10 browser policy.** Forgejo profile keys are optional inputs, not authority to change project OpenSSH access. The managed writer's lock/revision/atomic publication protects actual key changes. Replacing everything with a Forgejo key import silently changes credential selection and revocation semantics. |
| 11 | Spaces collection/degraded reads | **Retain bounded authorized enumeration and partial-read semantics.** Membership/operator degraded reads serve persistent projects during provider trouble; they do not authorize a terminal. Repeated request-local reads can be simplified, but removing completeness/authority distinctions would hide failures or expose metadata. D13 addresses avoidable native read blocking. |
| 12 | Host Tailnet management adapter | **Delete D8; retain host control adaptation.** CLI/LocalAPI commands cover real status/login/preference callers. Field-specific updates preserve native settings. The host revision fences stale drafts/identity, not provider ACL authority. No equivalent whole upstream browser-management replacement was demonstrated. |
| 13 | Enrollment policy/credential lifecycle | **Delete D7 and D4's `ActiveRun`; retain one active policy and meaningful binding/revision checks.** Admission, new-project default and disconnect differ in effect. Collapsing them into one toggle would silently disconnect or retarget projects. The always-false `EnrollmentVerified` field is removable dead surface, not a subsystem. |
| 14 | Provider key-creation wrapper | **Retain a narrow bounded secret-handling adapter; remove D4-derived one-attempt policy.** The selected SDK performs one API call, but uses unbounded `io.ReadAll`; its OAuth convenience helper uses `context.Background()`. Wholesale substitution loses actual bounds. Single-use/ephemeral/tag/scope checks are meaningful; duplicate parsing, tag ordering and local-clock strictness are narrower simplification candidates, not justification to delete the broker. |
| 15 | Per-run enrollment journal | **Delete D4 and D6.** Persistent attempt prohibition and the second Create reservation provide blocked recovery, not a required user capability. |
| 16 | Companion architecture/incarnation checks | **Retain the companion and real namespace/CID verification; delete only D5/D8 policy dependants.** Host-root Unix credentials do not automatically become privileged inside the shifted user namespace. Moving OAuth into a developer-controlled project or using a host-socket shortcut is not an equivalent deletion. Exact recipe comparison may be simplified only with its isolation assertions accounted for. |
| 17 | Runtime filesystem/DNS ownership | **Delete D5 identity-preservation storage; retain restricted input/control ownership and DNS safety.** The companion and project share the actual resolver file. Ignoring inode/ownership conflicts can overwrite somebody else's DNS; memory node state does not eliminate resolver backup/restore. No Soda DNS server exists here to delete. |
| 18 | Start/Stop/restart orchestration | **Delete D4/D5 recovery protocol; retain native parent-child lifecycle.** `BindsTo`/ordering, bounded startup/stop and exact-owned termination still matter. A cancelled Podman client is not proof its exec stopped consuming a key. That observation cannot simply be removed with the journal. |
| 19 | Tailnet UI/cross-feature projections | **Delete D3/D8/D10 dependants; retain host/project separation, independent drafts and exposure confirmations.** Network status is not project readiness or proof of reachability. Removing those distinctions would misstate the product rather than remove an invented subsystem. |
| 20 | Runner command/dispatch layers | **Simplify, not a principal deletion.** `Coordinator` and `Operations` duplicate decoding/action switches, but the CLI and root socket remain real distinct callers with different admission/response needs. Consolidate a duplicated dispatcher only if it reduces code; do not rebuild an invoker hierarchy. The old runner subprocess hop is already gone and cannot be counted again. |
| 21 | Runner registration/stored state | **Retain.** Dedicated account, restricted token file, native Forgejo configuration and a systemd listener implement local capacity, which Forgejo's registration record does not install. The remaining one-value provider field is not a provider framework. No Soda scheduler, results store or multi-provider backend was found. |
| 22 | Runner inventory policy | **Delete D11.** A broken or unsupported saved runner must not suppress unrelated healthy local capacity. Per-target descriptor safety remains. |
| 23 | Runner lifecycle/uncertainty/confirmations | **Delete D12 and D10 preflights; retain outcome/readback separation.** This UI already allows refresh after uncertainty; it is not the sticky project lockout in D3. Start/Stop's boot-policy coupling is explicit existing behavior, not a separate desired-state engine to delete. |
| 24 | Native-page connection/session coordination | **Delete D10 and D3 propagation; retain bounded OAuth/session/logout coordination.** Inspected Forgejo 15.0.7 API middleware uses OAuth2/HTTP-signature/Basic (optionally reverse-proxy) authentication and calls `AuthShared` with no session store; forwarding the browser cookie is not a supported replacement. `login_contexts` makes logout win over a late consumed OAuth callback. Removing it without replacing that protection can resurrect a signed-out session. No borrowed Forgejo cookie or new authentication framework is proposed. |
| 25 | Privileged-helper admission/serialization | **Delete D13 and D6 lock coupling; retain the root socket boundary and real writer locks.** Global read serialization is broader than needed; deleting all serialization would expose shared native mutation races. No new scheduling subsystem. |
| 26 | Repeated validation/state models | **Delete D10's duplicate browser authorization protocol; simplify redundant in-process semantic assertions individually.** Browser HTTP, privileged Unix socket, provider/native output and project filesystem are different boundaries. Deleting all decoders or replacing them with TypeScript casts is not a valid whole-mechanism removal. |
| 27 | Pre-release compatibility | **Delete disposable UI compatibility through D9; retain the existing database upgrade path.** Retained development fixtures are real state, including v9 databases; this candidate adds v10. `migrations.go` is an append-only migration list and bounded checks, not a general migration platform. Resetting/squashing it without a selected preserved-state transition is not supported. Tiny old-field/410 stubs are not a large deletion opportunity. |
| 28 | Tests/fixtures/packaging/procedural burden | **Delete support for each retired mechanism with its source.** Port useful behavior tests; retire assertions about the deleted deadlines, phases, archive copies, legacy formats and universal lockouts. Keep real authorization, state preservation, native topology and browser asset coverage. No new evidence framework, full-suite ritual or unrelated architecture gate is proposed. |

## Evidence and limits

- Inspected source owners, real web/helper/CLI/browser callers, owning guides and
  relevant tests. The selected SDK was inspected at
  `tailscale.com/client/tailscale/v2@v2.10.1` in the local module cache, including
  `keys.go`, `client.go` and `oauth.go`; this is not a claim that its OAuth wrapper
  meets Soda's operation deadlines. Also inspected the retained Forgejo 15.0.7
  API middleware/authentication sources under
  `.artifacts/research/forgejo-15.0.7-notifications/`, matching the concrete
  [native API authentication finding](forgejo-frontend-integration.md#existing-native-interfaces).
  That was source inspection, not a new native login test or archive verification.
- Prior reviews and native receipts were treated as historical evidence, not fresh
  validation or an independent reviewer consensus. This audit is one assistant's
  assessment. It deliberately changes earlier recommendations about terminal
  lifetime, unknown-state custody and inventory failure policy.
- No product tests, builds, benchmarks, native lifecycle operations, provider key
  requests, deployment or retained-state changes were performed for this audit.
  Test names above are inspected coverage/contract evidence, not passing receipts.
- The first ten findings are mechanism-removal recommendations; D11–D13 are smaller
  policy findings. They overlap across slices. Neither source-file lengths nor
  test/doc deletions establish net implementation savings.
- For implementation, update the owning requirement rather than adding exceptions
  around it. Check the changed behavior with its existing focused test driver;
  native behavior needs applicable native evidence before delivery. This report
  creates no second implementation backlog or mandatory new test sequence.

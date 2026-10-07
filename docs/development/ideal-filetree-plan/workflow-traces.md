# Connected workflow traces

This chapter follows the requested workflows through callers, authority, stored
state, native effects, completion and cleanup. It adds three bounded correction
findings: **P04-F1**, **O02-F1** and **O05-F1**. Earlier source repairs remain
complete at their recorded scope; installed qualification remains separate.

The [authority and state audit](authority-and-state.md) at `88a40b5d` expands the
current grant, transaction, custody and retry questions. It reuses these workflow
findings and records two separate broker failure windows without advancing this
chapter's source pin or reopening completed structural/terminal-fencing work.

## Subject and evidence

| Input | Identity / scope |
| --- | --- |
| Inspected source | `916857990fe6a5934adcab6e7024685768745ece`, tree `0de194d1970718fe97c029e7ac86b234091a874d` |
| Application baseline | `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; subsequent baseline, requirements, coverage, caller-map and observation chapters are documentation changes |
| Contracts | [Workflow requirements](workflow-requirements.md), their linked product/operator guides, and [controlling guidance](review-assignments.md#guidance-conflicts-and-controlling-decisions) |
| Current mutable inputs | The 12 pre-existing dirty plan/review files and 39 dependency inputs retain their baseline bytes; identities are in the ignored receipt |
| Method | Three Luna medium primaries, reciprocal source challenges and coordinator traces of setup/persistence/operator enrollment; tests inspected, no tests/builds/native/provider/database/remote operations executed |
| Responsibility | H06 owns this document; Go domain/Store/frontend, Rust broker/privileged/native and operator responsibilities retain their existing owners |

The ignored `.artifacts/workflow-traces-20261007-91685799/` packet retains source
selectors, independent challenges and baseline hashes. A connected source trace
establishes sequencing and predicates, not native completion or behavior PASS.
Reuse applies only to matching code, contract and question. The full 80-slice
validity review remains pinned at `f7e9cf9d`; this bounded chapter does not advance
that review or regenerate the desired tree.

## Enrollment and authentication

**Human authentication — WF-A01/A02 and all browser mutations.** Browser actions
enter Forgejo's allowlisted extension proxy, then the dashboard's distinct
private Unix listener. [`ExtensionAuthority`](../../../internal/web/auth/extension.go)
requires one admitted SDK context, constrains extension/method and compares its
actor with `Native().CurrentActor`. The receiving
[`extension handler`](../../../internal/web/api/extension_native.go) independently
checks contribution/path, derives the Soda user from the native positive actor
ID, binds native session generation and revalidates current authority around
effects. Mutations require the configured origin; profile routes perform their
own corresponding checks. The public dashboard mux does not provide a Soda
browser-session authentication API. A socket connection, cookie or requested
actor ID does not substitute for native actor authority. In-repository fixtures
inject SDK contexts/callbacks; Fountain's actual producer/admission semantics
remain an external evidence gate.

**Provider enrollment — WF-A06.**
[`soda-identity.ts`](../../../frontend/spaces/soda-identity.ts) submits the explicit
provider/label/exposure confirmation through authenticated
[`identity API`](../../../internal/web/api/identity.go) → bounded Go Unix client →
broker admin socket → [`start_enrollment`](../../../cmd/soda-identity/src/enrollment.rs).
Broker owns a transient session bound to the authenticated owner, provider and
label. Codex/Muse adapters admit configured binary/version and isolated state,
then perform their separate native login protocols. Owner-bound polling captures
completed account metadata and credential bytes, encrypts the connection and
metadata-only event transactionally, zeroizes in-memory material and closes the
provider session. Browser receives verification URL/code/status, never credential
bytes. Cancel, finish failure and persistence failure close/clear the session and
do not report completion. Native subprocess termination, provider-version
behavior and crash cleanup are not proven by the source trace.

**Operator host enrollment — WF-C06.** Console `soda-install` admits root,
installed CoreOS and its installer lock, explicitly selects a live private
address and arms bounded dedicated service/socket state. Native SSH ForceCommand
[`receive`](../../../cmd/soda-install/src/enroll/receive.rs) refuses commands/TTY,
wrong destination/port, expiry and oversized key input; it sends only a normalized
public key to the private broker. Kernel root credentials and the dedicated
receiver cgroup admit that peer. The fixed broker service checks its own cgroup,
live address and window before
[`authorized_keys` mutation](../../../cmd/soda-install/src/enroll/keys/authorized_keys.rs).
Directory lock, bounded same-descriptor read, exact inode/content checks and
complete no-replace publication preserve existing/editor-owned keys. Append,
sync/readback, SELinux relabel and result receipt have distinct failure outcomes;
loss after mutation is uncertain, never proof that no key was imported.
[`arm_enrollment`](../../../cmd/soda-install/src/enroll/session.rs) joins exact
service/socket cleanup before returning; failed native stop preserves attempt
state. An imported receipt establishes key installation, not a successful SSH
session. Completed O07/L12 custody repairs remain preserved.

## Projects and execution

**Create and join — WF-A01/A02.** Current native actor/repository ownership →
[`apiCreateEnvironment`](../../../internal/web/api/environments_create.go) →
durable unique repository/Project reservation → host's admitted fixed profile and
exact native create → `Store.MarkReady`. Existing unready reservation is inspected
and reconciled, never blindly replaced. Failed host response retains provisioning
uncertainty; failed Store completion can leave a created native Project without a
saved ready result. Optional Tailnet application has its own result. Ready is a
provisioning observation, not membership, route, SSH or application health.
[`Join`](../../../internal/web/api/environments_join.go) rechecks native authority,
repository access, Ready, supported stable login and selected saved keys; host
account/key provisioning precedes SQL membership. Native or SQL failure can leave
partial native state; existing member login remains stable.

**Saved versus installed keys — WF-A02.** Saved-key deletion is authenticated
profile DELETE → [`apiRemoveDevelopmentKey`](../../../internal/web/auth/session.go)
→ owner-scoped `Store.RemoveKey`; it explicitly leaves Project access unchanged.
Installed-key review/apply instead requires current repository write authority,
exact reviewed fingerprint set, revision and `confirm_empty` for an empty set →
host/guest locked revision-checked replacement → returned installed observation.
That separate confirmation does not satisfy final saved-key deletion. **P04-F1**
below records the missing owner-selected confirmation; SSH session revocation is
not inferred from either saved preference changes or transport closure.

**Start/Stop — WF-A03.**
[`Lifecycle mutation`](../../../internal/web/api/lifecycle.go) requires current
operator/Project-administrator authority and explicit action. Stop gates/cancels
attached terminal peers → coordinator withdraws dispatch and stops recorded runs
→ marker-first maintenance hold → host stops the exact retained container/unit.
Failures preserve individual coordination/hold/native outcomes; cancellation of
the caller does not prove effects absent. Start uses that same retained Project
and `VerifyProjectStart`; it does not release maintenance hold or revive runs.
**P05-F1 remains open:** Start does not reopen its Project-stop dispatch cause.
Its correction must compose with independent pauses and grants, not reopen all
dispatch indiscriminately.

**Provider execution and settlement — WF-A07/B04/B05.** Current Go
coordinator/Store admission → host
[`factory launch`](../../../lib/host/src/factory/launch.rs) → persist exact run
receipt under its lock → broker Acquire/persist lease → native reserve → persist
binding → broker Register/attest → deliver credential → consume one-use start
marker → native start/wait → capture/stop/Return → typed Go settlement/SQL state.
Browser/API admin routes cannot access runtime credential delivery. Broker checks
current connection/grant/provider/repository, immutable execution identity,
deadline and native binding before decrypting. Native reservation happens before
broker registration: missing broker binding alone cannot prove no host child.
The host receipt, per-run lock and marker-first Stop tombstone govern that race;
unconfirmed exact-run stop remains uncertain and fences duplicate launch.

Return, End, expiry, revoke, startup retirement and Close record terminal
execution observation; SQL prevents reversal and admission rejects terminal or
changed-digest IDs. If credential return commits but terminal observation fails,
replay sees the missing lease and refuses as uncertain. Only explicitly unbound
`reconcile_lease` releases a broker reservation to pending. It is not native
termination evidence. **I06-F1 is source-corrected at `8a1e392b`**; the distinct
F07/F08 host reconciliation/retry gates remain. Retry must preserve prior attempt
attribution and obtain fresh admission/allowance. Native attestation, actual
provider capture and process death were not executed.

## Terminal sessions

**Reserve/create/attach — WF-A09/A10.**
[`Private extension routes`](../../../internal/web/api/extension_terminal.go)
bind current actor, Project, membership login, repository and session generation.
Reservation generates an exact ID without creating a shell. WebSocket upgrade
[`stream`](../../../internal/web/api/extension_terminal_stream.go) validates
origin/headers and handshake, claims that ID, optionally creates the exact session,
checks ready state and current authority again, then attaches through host to the
same native PTY/account. Client/control/native pumps and heartbeats own the
attachment lifetime; cancellation closes the handle and unregisters the peer.
Existing HTTP upgrade/pump ownership stays together; a successful library call
does not establish the native shell or actor contract.

**Detach/End/Stop.** Detach closes the attachment, leaving the session available.
Explicit authorized End targets the exact ID and native service/cgroup, then
removes only owned terminal files; subscription-owned sessions require broker
End. End does not stop the Project or delete persistent data. Project Stop
cancels peers before its separate lifecycle effect and does not claim every
session had an explicit End. Unknown effects require exact-object inspection.
S05-F1's current `pty_select` writefds correction and real-descriptor assertion
are present; that test was not run here. S04-F1 malformed environment admission
and H01-F3 blocking NativeAttach mutex remain separate open subjects.

## Setup, activation and persistence

**Bootstrap — WF-C03.** Root/path/origin/absent-output admission → same-descriptor
bounded protected token → native Forgejo admin check → unpredictable private
grant key → create/adopt complete protected PostgreSQL secret set → exclusive
dashboard configuration write → one bootstrap-token revoke → success or explicit
intervention. Before publication, cleanup removes only this attempt's created
secrets; adopted files remain. After publication, configuration remains and blocks
blind setup replay. Writes are sequenced, not a proven fsync/power-loss guarantee.
**O02-F1** records incorrect certainty in the post-publication revoke error.

**Activation — WF-C01/C04.**
[`Activate`](../../../cmd/soda-activate/src/activation.rs) admits root, selected
TLS/origin, native operator/listeners and expected key/socket/account identities
→ publishes owned config/Forgejo extension/proxy/drop-in state → writes activation
marker → reload/restart/start services → polls observed active units. Partial
mutations survive errors; marker exists before service success and prevents blind
rerun. Blocking systemctl calls mean the outer poll is not proof of an absolute
operation deadline. Marker/config publication is not working human login or SSH.
CFG01 native configuration evidence remains a scoped gate; no new global setup
redesign or implicit rollback is selected.

**Database lifetime — WF-C13/A08.** Dashboard protected config/key/DSN →
[`OpenEncrypted`](../../../internal/store/store.go) → matching grant-key marker →
transactional current schema → application/private extension listeners. Version
26 initializes only empty state, refuses unversioned nonempty/wrong-version state,
checks required columns/guards and commits. Broker opens the canonical mirrored
schema with its own client/cipher; connections, encrypted credentials, grants,
leases, immutable execution fences and metadata-only events retain their actual
transaction owners. No live public `Store.events` consumer is established; I10
history exposure/retention stays unresolved. Dashboard shutdown closes terminal
and coordinator consumers before HTTP drain and DB close. Source joins do not
prove native descendant termination, concurrent PG behavior or crash recovery.

**Roles, backups and restore — WF-C05.** Fixed init service → protected safe role
inputs → native readiness → quoted SQL/stdin close/wait/error join. Backup timer/
service → admitted keep/root → exclusive private staging → globals/custom dumps →
nonempty/header checks → file/directory sync → no-replace publication → retention
→ printed completed path. **O05-F1** identifies that retention can delete that
exact new path. Restore requires `--yes`, retained run and selected valid databases
→ globals → native database existence/create → exclusive archive stage → copy/
restore → exact-stage cleanup. Cleanup errors fail; partial database effects are
not rolled back. Caller quiescence and separate snapshots remain explicit, with
no whole-appliance backup/restore claim. Completed SQL failure joins and L12
publication/restore custody remain preserved.

## Release verification and delivery

**Candidate inspection — WF-C09.** Actual `soda-candidate-check` →
[`check_candidate`](../../../lib/soda-release-deliver/src/check.rs) admits exact
candidate/architecture/revisions and confined root → hashes archives → image
identity/content → selected rootfs members through delivery's shared bounded OCI
scanner. For gzip, decoded EOF/padding is consumed while `MultiGzDecoder` lives,
then compressed length/hash completes. **D05/L13 repair `d4f3c922` remains complete**;
build imports this scanner and its duplicate engine is retired. Old blanket
build/delivery equivalence proposals are not a current retirement prerequisite.
Producer records remain distinct from installed qualification; observation
corrections in [the reliability chapter](observation-reliability.md) gate only the
checks that use their affected mechanisms.

**Admission/sign/publish — WF-C10.** Public library APIs admit trust/candidate/
media/qualification → digest/repository/expiry permit and protected signer inputs
→ local signing/verification. Separate publisher locks ledger → saves pending
before immutable upload → verifies anonymous roundtrip/channel history → marks
complete/receipt. An uncertain effect remains pending, without blind replay.
**D09-E3** is a bounded exported `finalize` composition concern: its optional
channel helper omits referenced-release publication and hardcodes previous absent.
No production caller of `finalize` was found; it is not an observed active delivery
failure. No in-repository production CLI caller for Soda release sign/publish/
channel fetch was established. `soda-artifacts` actually exposes inspect-oci,
convert-butane, fetch-coreos and fetch-coreos-iso: upstream-input actions.

**Fetch/install/import — WF-C11.** Public fetch API → trust/channel/architecture
admission and locked highwater → digest-pinned signed offer → persisted monotonic
progression → complete artifact checks → verified receipt. Later content failure
can retain advanced highwater without success. Fetch grants no installation
authority. Explicit root/console `soda-install` admits live media/lock, authenticates
identity, confirms/re-inspects disk and records attempt before native write.
Payload/layout identity checks precede effects. Separate image-import admits the
fixed image set and local blob/metadata hashes → observed Podman existence/pull/
post-observation with cancellation/reaping. Installer/import adapters do not
duplicate compressed layer scanning; Podman is the native content consumer.
Actual signer, registry, Podman, disk and installed qualification remain unperformed.

## Three correction packets in the existing plan

These specifications belong to current tasks/lanes. They do not dispatch source
implementation or create a second schedule. Each has one accountable owner;
cross-owner consumers are handoffs, not competing ownership.

| Finding / owner | Prerequisites and scope | Acceptance on the actual chain; not executed here |
| --- | --- | --- |
| **P04-F1 / B**, Go/frontend handoff within [A01/A02](implementation-tasks.md#a01-project-creation-membership-and-access) | Owner-selected [final saved-key confirmation](../../product/projects.md#explicit-joining) is settled. Existing saved-key frontend, private auth DELETE and Store mutation only. Enforce final-row decision with the mutation, including concurrent removals; installed SSH state stays independently owned. Completed A01 structural work stays complete. | Nonfinal own-row removal; final unconfirmed removal makes no deletion; affirmative final confirmation succeeds; stale/concurrent removals cannot bypass the gate; cross-user denial; no native key/Host mutation. Extend actual saved-key route subjects, not the separate installed-key apply test. |
| **O02-F1 / C**, [C05/O02](implementation-tasks.md#c05-activation-and-setup) | Established requested/observed/confirmed distinction. Existing setup revoke outcome, operator guide and WF-C03 wording. Distinguish known rejection from unknown remote result; preserve published config and native inspect/revoke → explicit activate path. No automatic replay/rollback layer. | Existing native-HTTP fixture applies DELETE then loses/truncates response: nonzero, config retained, outcome unconfirmed, no token-state certainty/retry/leak. Preserve explicit status rejection, ordinary success and pre-publication retry-input behavior. |
| **O05-F1 / C**, [C06/O05](implementation-tasks.md#c06-postgresql-initialization-backup-and-restore) | WF-C05/O05 already require retaining the just-published run. Existing rotate victim selection only: exclude exact current publication, then remove eligible other runs to satisfy keep>=1. Preserve private staging/no-overwrite/sync and restore identity; no clock service/retention ledger. | Existing backup/restore driver seeds a valid later-stamped retained entry, runs real backup keep=1, verifies emitted path still contains new dumps/globals and restores that exact completed path; normal retention still honors keep. Publication refusal/cleanup tests remain relevant. |

Each consequential packet receives independent **Luna medium** review. Settled
diagnostic/plumbing implementation can use Luna low; mutation concurrency and
retention identity use medium where unresolved. The coordinator owns explicit
staging, manifests/locks and resource-heavy/native qualification after its actual
prerequisites. No new facade, compatibility obligation or recovery subsystem is
selected by this audit.

The historical reviews [P04](reviews/P04.md), [O02](reviews/O02.md),
[O05](reviews/O05.md) and [I06](reviews/I06.md) carry bounded current dispositions.
P05-F1, S04-F1, H01-F3, F07/F08, D09-E3, I10, CFG01, L10.N4 and REQ-Q1 keep their
canonical scopes and owners. Missing native configuration evidence blocks its
parser decision only; a source-corrected transition remains distinguishable from
its still-unperformed installed qualification.

# Authority and state transitions

Two Identity broker failure windows need bounded corrections: acquisition can
leave an unlinked reservation after partial persistence, and close can report
success after a failed lease observation. Earlier factory, saved-key and evidence
findings remain open under their existing IDs. Typed trust admission, raw-byte
authority and completed terminal fencing remain preserved.

Source: `88a40b5df6c146fa568e46bc97e6bcb39cab16e5`, tree
`22829c7af775066f65e9dcd4620e64d1ffdaa745`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 pre-existing
dirty guidance files and 39 dependency inputs retain their recorded bytes.
This extends [workflow requirements](workflow-requirements.md),
[connected traces](workflow-traces.md), [observation reliability](observation-reliability.md)
and [concurrency custody](concurrency-and-termination.md). The [current tasks](implementation-tasks.md)
and [lanes](implementation-lanes.md) remain the execution plan. This is source
inspection and documentation; no tests, builds, database/provider/VM/network
operations or native qualification ran.

The [independent challenge](independent-conclusions.md) at `24fc3ea7` refines
F08/F12 below from the current split/ported definitions: dispatch capture and
acceptance/publication capture remain separate duties, while merge equality
supports a conditional completion refusal rather than unrelated-result admission.
Historical review locations are context, not current source-identity evidence.

## Authority and transition coverage

A library's successful parse, a durable ID, a prior permission observation and
a native receipt have different authority. Check the authenticated actor,
current operation grant, admitted bytes, state transition and confirmed effect
at their defining owners. Atomic SQL capacity admission does not make a preceding
permission snapshot current; request cancellation does not erase a committed effect.
The [trust model](../../architecture/trust.md) owns those distinctions.

| Workflow / defining owners | Fresh trace and disposition |
| --- | --- |
| Native extension → Go authentication → product API | Exactly one SDK context/admission, Soda extension ID, contribution/method, current native actor, positive stable ID and session generation are checked. Product contribution/path and same-origin mutation checks are separate. Mutators revalidate current session/operation authority. Profile preferences/keys do not confer repository or background service authority. Actual Fountain producer/admission behavior remains its external gate |
| Repository policy, appliance capacity and provider sponsorship → dispatch packet | Current grants form a captured plan. Assignment/reservation/run/view and capacity checks commit together under row locks. **F07-F1** remains: packet admission does not compare the full current enabled/paused/revision tuple with that plan. Host/broker binding checks are distinct and cannot supply the missing factory authority comparison |
| Acceptance withdrawal → immutable publication/correction/review/merge | Original intent/operation IDs, CAS, lookup/adoption and fencing are meaningful retained duties. Registration versus withdrawal ordering and complete correction enumeration remain open. Review intent retention/production allowance wiring and merge reachability retain their existing gates; no native duplicate write is inferred from source alone |
| Browser provider enrollment → admin socket → Identity controller/provider/Store | Actor derives from authenticated session; grant use checks ready Project, membership and current repository authority. Broker consumes trusted service assertions and owns connection owner/provider/generation, grants and credential custody. Enrollment closes/clears sessions on failure; encrypted credential and metadata-only event persistence are transactional. Browser sees metadata/verification status, not credential bytes |
| Identity acquire/register → host validation/delivery → return/end/reconcile | Controller serialization and Store transaction exclusion remain. Register matches exact execution/lease, current grant/generation and native binding before credential delivery. Return validates captured bytes and native stop, atomically rotates Codex encrypted custody and deletes the lease; Muse has its own parallel-execution profile. Failed retirement keeps uncertainty/lease custody. **AUTH-I-ACQ-1** and **AUTH-I-CLOSE-1** are separate failure windows below |
| Bound execution terminal fence | I06-F1's current repair is present: successful return/end/expiry/retirement terminalizes bound execution; the schema prevents terminal reopening. Only reconciliation of an unbound reservation can return its execution to pending. New acquisition/close findings do not reopen this completed repair or impose unresolved Muse InvocationID policy |
| Saved key → installed Project key review/apply | Saved keys are owner-scoped and public-only. Go SSH admission refuses options/private material and fingerprints parsed key bytes. Installed-key apply requires current membership/write authority, reviewed saved fingerprints, native revision and explicit empty-set confirmation. Guest directory lock, original-byte revision, inode recheck and private replacement own publication. **P04-F1** final saved-key confirmation remains a separate missing product requirement |
| Root console → enrollment receiver/broker → authorized_keys append | Kernel root peer and dedicated cgroup/unit, live address/window and public-key admission precede mutation. Same-FD bounded inspection, directory lock, inode/content checks and no-replace publication preserve existing keys. Short append, sync/readback/relabel or lost post-mutation result is uncertain, not safe blind retry or proof of no imported key |
| Release trust → protected permit → snapshot/sign/verify → publication | Shared typed P-256 SPKI admits original DER and on-curve keys; roles keep distinct keys/fingerprints. Permit binds exact repository/digest/expiry. Skopeo signs a fresh private snapshot after payload/channel admission and independently verifies the signed copy. Publication records pending state before remote write; ambiguous outcome is observed/reconciled rather than blindly replaced |
| Installer local CA and payload/disk admission | Strict typed ECDSA verification hashes original certificate TBS; local-CA fingerprint uses original DER. MediaIdentity binds console/payload and OCI digests; confined content verification and exact disk recheck precede writes. These local consistency checks alone do not establish the external media/boot authenticity chain; that native contract remains a bounded qualification question |
| Acceptance observations → scrubbed evidence → final record | L01 wait/pump/writer/final-flush ordering refuses incomplete capture. Final evidence publication consumes those failures. **OBS-S01** still permits artifact-reference loss before structured validation; **H06-F1** can misreport an analyzer failure as PASS. Both keep their existing IDs and correction owners |

Defining entrypoints include [extension admission](../../../internal/web/auth/extension.go),
[product admission](../../../internal/web/api/extension_native.go),
[profile mutations](../../../internal/web/auth/session.go),
[installed keys](../../../internal/web/api/access_keys.go),
[dispatch packet](../../../internal/store/factory_dispatch_packet.go),
[Identity acquisition](../../../cmd/soda-identity/src/acquisition.rs),
[leases](../../../cmd/soda-identity/src/store_leases.rs),
[execution fences](../../../cmd/soda-identity/src/store_executions.rs),
[retirement](../../../cmd/soda-identity/src/retirement.rs),
[native Factory stop](../../../lib/host/src/factory/stop.rs),
[guest keys](../../../cmd/soda-project-terminal/src/keys.rs),
[enrollment keys](../../../cmd/soda-install/src/enroll/keys/authorized_keys.rs),
[trust admission](../../../lib/release-inputs/src/trust_key.rs) and
[release signing](../../../lib/soda-release-deliver/src/native/sign.rs).

## New broker failure windows

### AUTH-I-ACQ-1 — persisted reservation is not atomically linked

Controller acquire separately commits execution admission, lease reservation and
the observation attaching the lease ID. A committed reservation followed by failed
observe_execution leaves PENDING with an empty lease reference and a durable
unbound lease. Same-ID/same-digest acquisition can reserve again. Muse deliberately
permits concurrent leases for different executions, so the second reservation is
not stopped by the other-provider connection exclusion. Close follows only the
recorded lease; startup reconciliation can retire the orphan, while ordinary
sweep can retain a future-deadline READY unbound lease until expiry.

The counterexample concerns partial persistence and same-identity retry. Current
Factory launch acquires once and retains its failure receipt; its duplicate launch
returns that receipt. Terminal producers use fresh IDs, and Unix HTTP does not
retry POST. No automatic same-ID retry or duplicate native/provider execution was
established. This remains a conditional broker idempotence defect, not an
authorization bypass or credential disclosure.

Make reservation and execution linkage atomic or equivalently idempotent at the
current Controller/Store owner. Include ambiguous reserve COMMIT as well as failed
link observation. Preserve independent Muse executions; connection-wide exclusion
would impose the wrong contract. Do not infer that a failed reply means no lease.

### AUTH-I-CLOSE-1 — failed lease lookup becomes confirmed close

close_execution uses `if let Ok(lease) = store.lease(...)`; every error falls through.
It then clears lease_id and can successfully persist TERMINAL after a transient
lookup/connection error. Only established NotFound can justify the idempotent
absence branch. Other query/decode/deadline failures cannot establish absence or
retirement. A second Close sees no reference and cannot retry that lease through
this execution, while the durable lease may remain until reconciliation/expiry.

The terminal acquisition/registration fence still holds. Native Factory callers
usually have additional stop/capture/return checks; source does not establish a
surviving process or exposed credential. The supported consequence is false broker
close confirmation and stranded lease state, which can block Codex reservation.
Preserve reference/binding and surface uncertainty on non-NotFound observation
failure. Retain terminal fencing and retryable retirement without converting failed
observation into absence.

| Packet / single owner | Exact scope and prerequisites | Acceptance for later implementation |
| --- | --- | --- |
| **AUTH-I-ACQ-1 / A**, Identity/A05/L08 | acquisition.rs, Store execution/lease transaction and minimal schema/query joins. Choose atomic/idempotent reservation+link using existing mutex/Tx and uncertain-COMMIT policy | Inject committed and ambiguous reserve followed by failed link; same ID/digest returns/reconciles one reservation or explicit uncertainty, never a second lease/event. Changed digest/terminal still refuse; Close and reconciliation account for custody; distinct Muse IDs remain concurrent |
| **AUTH-I-CLOSE-1 / A**, Identity/A05 retirement | close_execution, Store lookup error classification and actual close callers. Established NotFound is distinct from database/decode/deadline failure | Failed lookup then successful Store reconnect cannot produce confirmed close or lose the lease reference; retry can retire that exact lease; terminal fence remains; genuine absent lease is idempotent. Native stop/return failure stays uncertain |

Both touch acquisition.rs and must use one writer. Luna medium implements/reviews
the consequential transaction and custody contracts; settled transfers can use
Luna low. Typed-row simplification **SIMP-I-PG-1** can share the owner but must
preserve transaction exclusion, deadline/cancel/discard and uncertain COMMIT.

## Existing findings remain distinct

The following IDs were freshly checked against current source/callers. Existing
review records retain their historical pins; this table specifies bounded reuse,
not a whole-slice validity refresh or reopened completed structural work.

| Existing ID / accountable owner | Current gap and bounded correction/acceptance handoff |
| --- | --- |
| **F07-F1 / B** | Compare full current factory authority at final packet admission, not only capacity maxima. Establish revision tuple/refuse-or-replan policy; changed grant/enabled/paused state between plan and packet cannot reach host launch |
| **F07-F2 / B** | Recorded prompt lacks contracted controller limits/actions/checks/blockers, dependency outcomes and approved repository instructions/template revision. Identify authoritative inputs; their actual bounded content participates in the recorded prompt digest |
| **F08-F1 / B** | Order withdrawal and registration at one serialization point. Preserve both duties: current WithdrawDispatch enumerates registrations before its separate gate close; acceptance withdrawal versus initial publication-operation registration has its own capture interleaving. Both race commit orders must refuse registration or include its immutable operation in cancellation/reconciliation; no native commit is claimed |
| **F08-F3 / B**, native file handoff to A | Delayed successful finish can overwrite stop-requested/uncertain receipt attribution. Define owner-confirmed settlement; retain stop provenance and custody uncertainty across both interleavings |
| **F09-F1 / B** | Published-parent correction operations escape open/fenced-only withdrawal enumeration. Include every retained correction identity in cancel/reconcile, preserving immutable intent and uncertain outcome |
| **F10-F1 / B** | Existing allowance helpers lack the production autonomous reviewer/correction path. Consume existing prerequisites and cumulative limits through the actual coordinator; seeded reviewer fixtures do not prove it |
| **F10-F2 / B** | Reviewer submission reconstructs a stable-ID intent with fresh head/deadline rather than retaining the original. Persist/lookup original intent before submit; uncertain retry cannot change it. Actual native G05 replay remains its gate |
| **F12-F1 / B** | Current Merge.Validate/completeMerge exact head and BaseTip equality can refuse a reachable attributable result after legitimate target advance; this is a conditional completion/profile mismatch, not demonstrated unrelated-result admission. Actual G07/native producer evidence is prerequisite; unrelated/stale/ambiguous result cannot release dependants. Removing equality alone is insufficient |
| **P04-F1 / B** | Require the owner's explicit confirmation for deleting the final saved key, at the actual frontend/API/Store deletion decision, including concurrent changes. Installed empty-set confirmation is separate; saved deletion cannot claim SSH revocation |
| **OBS-S01 / C** | Make redacted artifact-map conversion fallible: distinct transformed-key collisions cannot drop a digest and publish completed. Preserve existing schema/noncolliding/repeated-path behavior and leak-safe failure |
| **H06-F1 / C** | Distinguish analyzer/tool/parse/missing-input failure from valid clean output through the real script/hook. Failed observation cannot produce complexity PASS |

The [existing observation packets](observation-reliability.md#prerequisites-and-acceptance-for-later-authorized-repairs)
and [workflow findings](workflow-traces.md) retain their full prerequisites and
checks. CON-G01/G02 and OBS-W01 remain in the concurrency chapter rather than
duplicate state findings. O02-F1 setup token/config partial completion and O05-F1
backup retention remain separate workflow corrections.

## Completion and limits

Three Luna medium primaries and independent cross-challenges covered broker
authority/custody/transactions, Go factory/Store authority and release/evidence.
The coordinator traced profile/key/native enrollment and browser mutation paths,
and identified AUTH-I-CLOSE-1 for independent medium challenge. Review narrowed
AUTH-I-ACQ-1 to ledger reservations without an automatic retry, preserved I06-F1,
and separated installer media consistency from unestablished boot authenticity.

Source coverage, workflow tracing, independent challenge and exact existing-owner
handoffs are complete for the recorded boundaries/questions. Actual SDK conditional
mutation support, media trust bootstrap and native process/signer/provider behavior
retain bounded evidence gaps; they hold only dependent changes/qualification.
The full desired tree and historical 80-slice validity audit are not regenerated
or advanced. L02/L03 trust/entropy repairs and L01/L08/L09/L12 completions retain
their recorded scopes and historical receipts, without new runtime PASS claims.

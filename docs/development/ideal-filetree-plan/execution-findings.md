# Finding allocation within the existing execution plan

This is the detailed subtask allocation for the existing
[task list](implementation-tasks.md) and [lanes](implementation-lanes.md), not a
second queue. Each row has one accountable finding owner; cross-owner files use
the explicit physical handoffs below. Parent packets retain their completed
M/C/V receipts. New follow-ups never reopen those receipts or claim native proof.

Reconciled at `04286c48cf06943c03b9d59a53a2e9c91ecc9de5`, tree
`6a1238924212a64ebffff47cbd440ea589fa1184`, application source `0b073439`.
The [independent challenge](independent-conclusions.md) supplies 75 prior
canonical dispositions plus CA-ALGORITHM-PROFILE-1. All 76 are allocated below;
aliases are not extra findings. Existing 80-slice corrections and Q1–Q9 remain
in their parent task/review records, including findings outside this recent
challenge. No source implementation, build, test or native operation ran here.

## Selection order and writer rules

1. Repair observation failures, authority/custody, cancellation and resource
   admission. Fix the tooling used to judge a later change before relying on it.
2. Settle expensive integration mismatches with a bounded caller/upstream proof:
   operation identity, merge reachability, producer profiles and ownership. A
   failed proof holds that dependent replacement, not independent work.
3. Transfer settled callers and remove their last obsolete machinery. Preserve
   actual producer contracts and necessary policy; do not build more wrappers.

These priorities choose among ready tasks. A held correctness specification
does not block a disjoint settled cut. Same-file corrections precede or share
that file's simplification; do not run both writers concurrently. Rank 1 uses
Luna medium for transactions, secrecy, cancellation and custody; Luna low handles
settled settings/error joins. Rank 2 uses medium for decisions/proofs; rank 3
uses low after admission is settled. Each coherent packet gets one independent
review at the corresponding level. Coordinator alone integrates manifests,
locks, shared roots, payload/compile selectors and expensive checks.

Current paths below are defining surfaces, with existing caller/test descendants
in the linked owning audits. Targets remain those same owners unless a row
explicitly removes a bridge. Dispatch reserves the complete literal file list,
including module roots and fixtures. No global extraction barrier or new package
is selected. B supplies canonical schema changes and A the broker mirror, together.

## Rank 1: correctness and trustworthy evidence

All rows are pending follow-ups. Their exact prerequisites are required outputs,
not completion of an entire parent packet. Established profiles need no new
product decision; unspecified bounds and transaction designs stay explicit.

| Finding → existing subtask / accountable owner | Exact scope and retained duty | Prerequisites | Acceptance checks for later implementation |
| --- | --- | --- | --- |
| B03.C operator bound → B03.C / B | [operator.go](../../../internal/factory/control/operator.go), outer 4 KiB request admission before strictjson; retain principal/actions | Existing schema/size profile; independent of Q8 and SDK changes | Exact cap succeeds; cap+one and late read error refuse; valid prefix plus whitespace cannot hide trailing input |
| H06-F1, TEST-CARGO-RESULT-1, TEST-SOURCE-CLI-GATE-1 → C11.C-observers / C | [complexity gate](../../../scripts/check-complexity.sh), [Cargo helper](../../../tests/build/helpers.go), [Muse selector](../../../tests/build/muse_exec_test.go); tool outcomes and admitted optional CLI | Selected analyzer semantics; explicit retained Muse binary/effect selector. Remove personal fallback; no provider call needed for source tests | Tool/parse/missing-input failure never PASS; Cargo failure including empty stderr never cached success/stale target; ordinary source checks use deterministic subjects, optional CLI has explicit identity |
| OBS-G01 → C11.C-observers / B | [installed.go](../../../internal/acceptance/installed.go) and existing [process owner](../../../internal/acceptance/process.go); stdout/stderr retention, wait/drain/cleanup | C gives B an exclusive whole-file tooling handoff; select caller output ceilings/cleanup allowance; retain SSH/Git/psql classification | Bound before append; cap/read/wait/drain failure cannot become PASS or expected denial; descendant-held pipes/cancellation finish or transfer custody visibly within allowance |
| OBS-D01, OBS-S01 → C11.C-observers / C | [SSH readiness](../../../tools/acceptance/src/command/ssh.rs) and [finalization](../../../tools/acceptance/src/driver/finalization.rs); phase outcome and artifact-map cardinality | Completed Phase/pump/redaction owners; repeated-original-path policy | No start or successful readiness after phase expiry; bounded retry/cleanup. Distinct transformed-key collisions fail safely before completed publication; noncolliding digests retained |
| L16.G → L16.G / C | Secret collection before raw/trimmed/decoded/Ignition/line variants and evidence admission; retain matcher/redaction duties | Select aggregate count/raw/derived-byte profile before expansion; L01/L12 retained | Many individually valid files and repeated/derived inputs cannot exceed aggregate budget before refusal; exact limits pass; refusal is leak-safe and prevents capture/completed publication |
| OBS-W01, OBS-R01 → C08.C-evidence / C | [worker execution](../../../lib/soda-release-tools/src/worker/execution.rs), runtime release, [RecallLog](../../../lib/soda-release-image/src/recall.rs) | Exact-unit terminal confirmation/never-dispatched distinction; select diagnostic fragment/line/reason byte budget | Unconfirmed stop preserves runtime and primary/cleanup errors; confirmed completion permits removal. Newline-free/huge-line stderr stays bounded before copying while full attached log/error propagation remains |
| AUTH-I-ACQ-1, AUTH-I-CLOSE-1 → A05.C-custody / A | [acquisition.rs](../../../cmd/soda-identity/src/acquisition.rs), execution/lease Store transaction and lookup classification; preserve I06 fence | Select atomic/idempotent reserve+link within existing Store/Tx and uncertain-COMMIT contract; NotFound distinct from other errors | Committed/ambiguous reserve plus failed link cannot silently orphan/duplicate a lease on same-ID retry; changed digest/terminal refuse. Lookup failure never confirms close/loses custody; genuine absence remains idempotent |
| ID-CFG-01 → A05.C-bounds / A | [main.rs::load](../../../cmd/soda-identity/src/main.rs) settings read on one opened descriptor before unchanged 1 MiB strict decode | Existing settled cap/path policy; Luna low | Exact cap reaches decoder; cap+one/growth/read error fails before excess retention; no reopen or loosened decoder |
| RES-I-PROBE-OUTPUT-1 → A05.C-bounds / A | [providers/mod.rs](../../../cmd/soda-identity/src/providers/mod.rs) run_capture and actual pinned version callers | Select stdout cap and reader cleanup allowance; preserve binary pin/version equality | Cap+one/read error fails before comparison; matching incomplete prefix cannot pass; stalled pipe has bounded owned reader/child cleanup |
| RES-I-LEASE-ENUM-1, RES-GO-IDENTITY-CONNECTION-LIST-1 → A05.C-bounds / A | [lease Store](../../../cmd/soda-identity/src/store_leases.rs), [connection Store](../../../cmd/soda-identity/src/store_connections.rs), controllers/routes; B receives explicit list-consumer handoff | Supported inventory/page/retention profile and stable complete progress/error semantics; public list contract separate from internal revocation | Bound producer copies before serialization; every supported lease/connection considered or explicit continuation/refusal; incomplete scans never confirm revocation/reconciliation; Go body cap is not producer admission |
| CON-M01, CON-M02 → A07.C-muse / A | [muse/launch.rs](../../../lib/host/src/muse/launch.rs) child/supervisor/FD custody and listener workers/handles | Choose identity-safe signal/reap owner, finite worker budget and reclamation; same writer for both findings | Disconnect cancels owned child; no signal after identity release; supervisor/FD joins complete. Silent peers/churn stay within budget and completed handles reclaimed during uptime |
| CON-G01 → B03.C-lifetime / B | [dashboard shutdown](../../../cmd/soda-dashboard/main.go), coordinator Close and admitted mutators/DB/flock lifetime | Select stop-admission/drain/ownership-release sequence including Shutdown failure | New admission stops before ownership release; admitted mutation/host-broker work finishes or remains explicitly owned; second coordinator cannot overlap unresolved first-owner work |
| RES-GO-ACCEPTANCE-DEPENDANTS-1 → B03.C-lifetime / B | AcceptanceDependants, assessCascade and dispatch reconsideration population/visited state | Supported graph/retention/continuation profile; correctness cascade and best-effort dispatch remain distinct | Bound retained copies with deterministic sort/dedup and exhaustive eventual consideration; scan errors propagate in correctness path; no arbitrary truncation or omitted visited state |
| CON-G02 → B06.C-admission / B | [background_admission.go](../../../internal/forgejo/background_admission.go) bootstrap waiting and publication | Cancellable waiter under existing shared admission; retain per-dial peer, atomic pair publication and authenticated-401 policy | A blocked caller observes its own deadline/cancellation; successful admission remains shared and coherent; no duplicate client/admission owner |
| P04-F1 → A01.C-saved-key / B | Saved-key frontend/API/Store deletion; A01 retains native Project access scope | User's explicit final-saved-key confirmation requirement; current row state at mutation, B exclusive writer | Final deletion including concurrent removals requires explicit confirmation; nonfinal deletion works; cross-user denied. Saved preference deletion does not claim installed SSH revocation |
| F07-F1/F2, F08-F1/F3 → B03.C / B | Current dispatch packet/prompt; both dispatch capture/close and acceptance/publication capture; native stop-owned finish update | Q8 exact authority tuple/approved prompt sources/serialization contract. B04 releases shared publication files; A receives exclusive finish/launch/stop/receipt handoff | Current enabled/paused/revisions govern packet; required bounded prompt inputs/template participate in digest. Both commit orders refuse or capture exact IDs in both withdrawal paths. Delayed finish preserves stop attribution and confirmed custody facts |
| F09-F1, F10-F2 → B04.C / B | Correction operation registration/enumeration/cancel; review_cycle immutable original work | Q8 original-intent ownership/recovery specification; check whether native owner already persists full tuple. Q5 gates only required native capability | Every correction ID captured/fenced/reconciled; same-ID retry uses original admitted tuple before submit, including changed actor/head/deadline; no parallel ledger or uncertain-mutation replay |
| F10-F1 → B04.C / B | Existing coordinator reviewer/correction creation and cumulative allowance | Q3/Q8 exact role/harness/allowance inputs plus F07/F09/F11/F12 required outputs | Real production path creates distinct reviewer and bounded correction, consumes cumulative limits and terminates; seeded fixtures alone insufficient; no new service |
| P05-F1 → A02.C / A | Project Start/stop-cause decision; B performs Store/dispatch handoff | Q1 cause-specific grant/quiescence representation | Reopen only the authorized Project-stop cause, preserving independent holds, revisions and uncertain stop; unrelated withdrawals stay closed |
| O02-F1 → C05.C / C | Setup token/publication/revocation result and explicit follow-up | Established single revoke and published-config authority; native proof separate | Lost/failed revoke after publication preserves published state plus uncertainty; explicit inspect/revoke/activate choice, no success inference or automatic retry |
| O05-F1 → C06.C-rotation / C | Backup rotation excludes exact newly published file; retain staging/sync/no-overwrite/keep>=1 | Existing backup path/retention contract; Luna low | Later-stamped old entry cannot cause current publication deletion; legitimate older rotation and errors remain; no DB operation needed for selection regression |
| ENROLL-CLEANUP-RESULT-1 → C07.C-cleanup / C | [enroll/session.rs](../../../cmd/soda-install/src/enroll/session.rs) arm_enrollment primary operation/session.close result join | Existing EnrollUncertain and session custody; Luna low after settled error join | Primary failure plus failed close retains both; successful operation plus failed close fails; uncertainty never licenses retry; completed O07 write/address repairs preserved |
| ACC-SNAPSHOT-FILE-1 → C11.C-custody / C | [snapshot Entry](../../../tools/acceptance/src/project_state/files.rs) opened-file admission/hash branch | Same-open type/512 MiB cap/failure schema | Nonregular/FIFO/link refusal without blocking; fstat/hash/cap bind consumed FD, growth refuses; ordinary files hash correctly. No atomic-tree claim |
| CUST-C-BUTANE-CLEANUP-1 → C08.C-artifact-cleanup / C | [artifacts.rs](../../../lib/soda-release-tools/src/artifacts.rs) convert_butane/finish_butane_conversion/run_butane create/clone/wait/unlink ownership | Private exclusive output and sanitized primary/cleanup contract; shipped soda-artifacts caller | Clone/run/unlink/wait errors finish owned cleanup and report actual removal plus primary failure; private mode/existing-output refusal/success remain; no demonstrated descendant survivor |

## Rank 2: costly boundary and profile decisions

These rows produce a decision/proof before their dependent cut. A missing native
producer remains a hold, not an instruction to invent fields or another service.

| Finding → existing subtask / accountable owner | Exact scope | Prerequisites | Acceptance / disposition |
| --- | --- | --- | --- |
| F12-F1 → B05.C / B | Merge.Validate and completeMerge equality versus exact native result attribution/reachability | Q5 actual pinned G07 producer/result/base semantics; B07 adapter evidence | Equal and legitimately advanced target cases use authoritative attribution/reachability, actor/PR/head/base/closure; stale/unrelated/ambiguous refuse. Conditional completion refusal, not demonstrated stale admission; no blind equality deletion |
| SIMP-SDK-1 → B06.M-adapters / B | Current background transport/admission and exact sibling SDK client | Selected SDK per-dial peer pin, typed sanitized refusal, shared reader/publisher admission, push and same-FD bounded credentials | Only remove client machinery after every required capability/caller holds; SIMP-SNAPSHOT-1 independently proceeds; no alternate Soda read path |
| SIMP-I-JSON-1, REP-HOST-STRICT-1 → A05.M-adapters / A | Identity strict DTO admission and host alias binding, separately at actual readers | Per-reader alias/null/unknown/required/number profile; existing byte/duplicate/depth/raw-byte policy | Typed decode can remove remaps while rejecting duplicate/depth/trailing/malformed input and preserving supported producer semantics. Host work handed to A07 physical owner in same lane; no global exact-case rule |
| SIMP-GJSON-1 → B06.M-adapters / B | strictjson Token preflight/typed decode and actual DTO callers | Per-DTO sorted-remarshal alias winner/null/unknown rule; retain EqualFold HaveNodeKey/source bytes | Remove repeated RawMessage/map/serialize parsing after profile checks; preserve cap/read/UTF8/object/EOF/duplicate/depth/refusal rules; B03.C outer admission independent |
| REP-CFG-1 → C05.M-profiles / C | Command-local activation/welcome/access projections, with B dashboard whole-reader handoff | Exact consumed-field name/null/presence/duplicate profile; CFG01 separate | Positive operator i64, presence-based public_url refusal including null, origin/listener/banner and no-effect-on-invalid semantics remain; no repeated dynamic parsing or third shared config model |
| SIMP-PEM-1, CA-ALGORITHM-PROFILE-1 → C10.M-profiles / C | One bounded CERTIFICATE envelope; owning installation guide and exact CA producer/algorithm profile | Single-object/no-extra/profile settled before PEM cut; actual producers before any algorithm narrowing | Remove manual PEM/Base64 line mechanics while preserving 16 KiB admission, original DER/TBS/fingerprint and strict CA/signature authority. Current feature trim keeps all algorithms; Caddy fixture proves P-256, synthetic curves are not permanent obligations |
| SIMP-REL-WIRE-1, SIMP-REL-ORDERED-1, SIMP-INSTALL-OCI-1 → C10.M-profiles / C | Current build/image wire DTO joins, image-config versus Ignition trees, installer reachable OCI graph | Matching producer/readers; duplicate/order/integer/null/raw-number and native schema profiles; no third DTO allocation | Keep unowned values, actual update/comparison semantics and descriptor/path/type/digest/layer budgets; delete only matched bridge/tree duties with last consumers |
| L10.N4 → L10.N4 / C | Resolver-inclusive host provider deadline/Executor fit proof; A receives production caller only after proof | Existing credentials/caps/native marker/one-shot uncertain-operation contract | Owned deadline covers DNS/connect/read/write/cleanup; no detached resolver. Failed proof holds this replacement; setup N3 stays complete |
| CFG01 → L17 / C | Effective Forgejo config parser fit and deployment overrides | Existing 21-case native corpus; candidate continuation failure; admitted dependency and actual override evidence | All required native semantics/no-guess/marker hold before cutover; rust-ini failure remains held; CFG02 independent; no generic parser recreation |
| D09-E3 → C10.C / C | Exported finalize prior-state/retained-ledger helper and config-test callers | Q7 retained-history/current external caller contract or supported retirement decision | Correct actual retained helper or remove it after last-caller proof; no current active-pipeline failure or registry replay inferred |
| CUST-G-SOURCE-1 → B07.M-retirement / B | Config.Source only in internal/forgejo/publish/source.go; adjacent sourceRepository retained | L18/R02.targets current last-caller and retention decision | If dead, delete only method/tests. If retained, same-FD cap+one/read-error before bytes. Do not repair a discarded path or delete the whole publisher |

## Rank 3: straightforward cuts after settled contracts

| Finding → existing subtask / accountable owner | Exact scope | Prerequisites | Acceptance checks |
| --- | --- | --- | --- |
| SIMP-I-PG-1 → A05.M-adapters / A | pg_query/store_schema typed Row::try_get, params and Json<T> | Selected NULL/int4/JSONB/schema profile; existing Store/Tx custody | Generic Field/Row/string round trips disappear; real typed cases, exclusion/deadline/cancel/discard/join and ambiguous COMMIT remain |
| SIMP-SNAPSHOT-1 → B06.M-adapters / B | snapshot_transport::decodeSnapshotAnswer typed SDK values/duplicate Soda wire DTOs | Matching field tags/types/nulls at pinned SDK; no upstream transport gate | Remove marshal/decode/copied wire models; preserve revision bracket, completeness, digest, visibility and repository binding |
| SIMP-FD-1 → A03.M-adapters / A | project-terminal sys::open_at integer flags/mode into typed rustix | Current component/no-follow/errno/FD contract | Actual callers pass typed flags; NOFOLLOW/CLOEXEC/mode/errno/custody preserved; libc and rustix entries are one cut, not global libc retirement |
| REP-ACC-EVIDENCE-ROUNDTRIP-1 → C11.M-format / C | Acceptance compact serialize/parse and duplicate structured formatter | Corrected observation collision/resource outcomes; unsigned evidence consumer profile | Scrub/collision/leak/depth/size, sorted fields/LF, recomputed hashes and publication order hold; second parse/formatter disappear |
| REP-FMT-1, SIMP-REL-EMIT-1 → C10.M-format / C | Candidate/lab pretty formatters and controlled unsigned release emitters | Actual output consumers, shared helper last uses, deterministic field/order/integer/newline rules | Standard emission preserves values/current determinism; recompute actual blob/candidate hashes; signed inputs/fingerprints remain exact; no historical Go/Python byte gate |
| SIMP-ASSET-NODE-1 → C09.M-format / C | Provisioning document Node/PythonFormatter and sole base.json producer | Current private mutations/integer/output/Butane input contract | Value/standard emission retains private strings, inserted and unknown future keys, exact integers and custody; remove formatter/tree, no candidate/lab conflation |
| COST-GO-SQLITE-FIXTURE-1 → C11.M-dependencies / B | staged_seed helper/import and two control test callers | Existing test-support owner; retain acceptance SQLite | Production dashboard loses exact helper/driver edge; both native fixture flows work; no module deletion/new forwarding package |
| COST-HOST-BUILD-EDGE-1 → A07.M-dependencies / A | host iconfig native predicate/direct build edge | Current predicates/target graph; coordinator manifest/lock handoff; retain delivery payload | Host payload/OS/architecture/conflict refusals pass; only exact build dependency removed; no new model or measured size claim |
| COST-IDENTITY-AEAD-FEATURES-1 → A05.M-dependencies / A | Identity direct AES alloc feature request | Exact upstream gates/target graph; coordinator manifest/lock handoff; retain direct fallible RNG | Seal/open/AAD/key/nonce/malformed-ciphertext checks pass; only unused feature requests removed; no measured size claim |
| COST-INSTALLER-KEY-FEATURES-1, COST-BOOTSTRAP-SEAM-1 → C10.M-dependencies / C | Installer direct curve/RSA features; two snapshot execute calls/full RunnerProduction stub | Exact selected upstream decode/verify features; coordinator graph handoff; current algorithms and Runner/log/cancel contract | Focused Caddy/X509 signatures/fingerprints unchanged; unused requests trimmed. Pass existing Runner directly, bad revision refuses before effects; broad real Production bridge remains |

Humantime's settled CLI03 cut stays in C11.M-format/L14: remove redundant empty,
minus and bare-plus prechecks, retaining accepted leading-plus stripping, μ
normalization, zero refusal, 24-hour option cap and errors. It is a maintenance
cut of a completed adapter, not another duration-engine adoption.

## Retained, completed and optional dispositions

| Finding / parent and sole accountable owner | Scope, prerequisite and acceptance/disposition |
| --- | --- |
| D03-F1 / C08.C / C; D06-F2 / L01 / C | Completed concurrent pipe drain/error joins and bounded Phase child remain checked at their source identities. Preserve actual regressions through affected changes; later caller fixes/native qualification are separate |
| H01-F3 / A07.C / A; I06-F1 / A05.C / A | Completed owned WebSocket reader and broker terminal fence remain checked. Preserve existing lifecycle/terminal subjects; no new detached pump or reopening task |
| OBS-S02 / C11 / C | Withdrawn: retain disposition, remove any pending correction direction. A new demonstrated defect needs its own evidence |
| TEST-ACCEPTANCE-SSH-EMULATOR-1, TEST-REL-PRODUCTION-BOUNDARY-1 / C11.V / C | Evidence labels only: real orchestration uses substituted SSH/keygen/transfers or execute/capture/next. Preserve useful policy assertions; selected R04 actual native subject is required for native claims |
| TEST-RETIRED-PYTHON-OUTPUT-1, TEST-REL-ORACLE-1, TEST-RETIREMENT-GUARDS-1 / C11.M-tests / C | Supersede unsupported unsigned old-output equality and only six named historical-path/token guards. Actual emitter consumers, current Rust behavior, TestDependencyDirection and no-Python gate must remain before deletion; signed originals/raw fingerprints remain exact |
| JOIN-HOST-PROBE-PATH-1, JOIN-INVOKED-IDENTITY-1 / C11.C-observers / C | Fix five obsolete host.sh prefixes from current staged inventory; narrow native-support claims to stored-image/content checks. Source selectors/guide checks first; a future running-image claim requires actual selected container .Image comparison. No automatic installed run or new attestation system |
| JOIN-BROWSER-CACHE-1 / B08 / B | Conditional question, no stale token shown. Retain current manual-token/epoch distinction unless owning cache policy requires automatic invalidation; then check actual template/builder consumers before a change |
| SIMP-BROWSER-1 / A08.M / B | Parked ownership alternative. Reconsider only a concrete screen/attachment diff reducing duties/callsites; preserve renderer seam, auth/generation/visibility, detach-versus-End, queue release and fit bounds. No extra controller framework |

All 62 retain recommendations preserve selected engines and actual application
policy; none creates an implementation task simply for existing. The 11 simplify
rows map to the exact cuts above and CLI03; libc/rustix and PEM rows share cuts.
Optional L16 remains C-owned/deferred after L16.G and demonstrated value.
A34/C41 stay preserved optional seams; B27 is integrated and never replayed.

## Supersession and finish

Pending custom hash/curve/encoding/JSON/SSH/CA/PG/HTTP/WS/archive/CLI engine
decompositions in A03/A04/A07/C01/C02/C03/C05/C07/C10/C11 are explicitly
superseded by completed L02–L14 and these caller-policy cuts. Pending historical
unsigned byte-golden equivalence in C09/C10/C11 is superseded by actual consumer,
semantic and current deterministic-output checks. Required source bytes and
fingerprints are preserved. No completed move or test receipt is erased.

R02.inventory is complete at the current responsibility snapshot, with authored
documentation deltas counted separately. Keep selective current caller/owner
updates with each integration. Full R02.targets desired-tree/decomposition
regeneration waits for the replacement boundaries to settle; R02.joins and R04
retain actual target/shipping/installed scope. Missing one parser, provider or
native profile holds only its dependent action. This reconciliation completes
planning allocation and independent scope challenge, not code corrections or
qualification. Implementation still requires the matching scoped instruction.

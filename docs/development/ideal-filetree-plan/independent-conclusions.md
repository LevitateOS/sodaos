# Independent challenge of consequential conclusions

The selected engines remain supported, with targeted caller simplifications and
explicit correctness work. Independent challenge corrected five misplaced
finding references and narrowed one retained-compatibility assumption. Agreement
means the named conclusion follows from its contract and source evidence; it
does not establish complete workflows, minimal architecture or measured savings.

Source: `24fc3ea7ed60295ef4667248df2e167d106885bf`, tree
`104a68db1b86414e44692acef66fe55b34e04802`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`. The 12 already dirty
guidance documents and 39 dependency inputs retain their recorded bytes. This
is documentation upkeep, with no application edits, builds, tests, dependency
resolution, network, database, provider, VM or installed operations.
[Tasks](implementation-tasks.md) and [lanes](implementation-lanes.md) remain
the execution plan. This chapter refines evidence and prerequisites rather than
replacing that plan or regenerating the desired tree.

## Independent coverage and evidence rules

Two Luna medium reviewers challenged the coordinator's consequential synthesis:
the preceding Go/browser primary examined the 40 crypto/native entries, while
the preceding native/release primary examined the 33 Go/browser/support entries.
Each examined adequacy as well as defects, removal and compatibility decisions.
The coordinator reconciles their results; Luna low checks bookkeeping separately.
Reviewers are independent of the current synthesis, not of every historical
source inspection. Reuse of a reviewer's own earlier receipt is navigation and
same-question evidence, not an additional independent vote.

All **73 integration entries** in the [maintenance assessment](integration-maintenance-result.md)
have challenged recommendations: **62 retain, 11 simplify callers**. The
[selected-version caller maps](library-integrations/README.md) and
[API challenge](library-integrations/adapter-challenges.md) supply upstream and
consumer evidence. Support/tool/module-set entries are not extra engine
replacements. Eleven rows can share one implementation packet.

Regex discovery yielded **78 claim mentions**, not 78 defects or fresh body
reviews. `SIMP-REL-EMIT` and `SIMP-REL-WIRE` fold into their `-1` IDs; `RES-Go`
is a family mention covered by the two concrete Go resource records. The result
is **75 prior canonical claim dispositions**, plus the new CA profile question
below: **76 records**. Completed, withdrawn, dormant, conditional and evidence-held
records stay distinct from open defects. The grouped ledger below preserves
their contract, decisive evidence and limits; exact per-claim reviewer records
are in local ignored `.artifacts/consequential-challenge-20261007-24fc3ea7/`.

Earlier evidence is reused only where the relevant source unit, governing
contract, actual caller and review question still match. Changed or moved units
require current inspection. Factory files differ from the historical `f7e9cf9d`
reviews: current split/ported definitions and the application-`0b073439` authority
challenge supply their evidence, rather than presumed historical identity.
A manifest, selector count or filename match alone is insufficient. Current
counterexamples establish ordering/admission
problems; they do not establish that a native race, secret leak, unauthorized
mutation, exhaustion or installed mismatch occurred.

## Corrections applied to the current synthesis

| Integration | Challenged reference and correction | Evidence and limit |
| --- | --- | --- |
| Go standard crypto, GB-MAINT-14 | Remove Rust L02/L03 adoption references | Actual Go raw fingerprints, authority and fallible randomness remain in the [Go caller map](library-integrations/go-browser.md); Rust finding IDs do not establish Go defects |
| Go URL, GB-MAINT-15 | Remove OBS-R01 | OBS-R01 belongs to release-image RecallLog byte growth in the [resource audit](resource-bounds.md), not Go URL admission |
| xterm, GB-MAINT-18 | Remove OBS-G01 | OBS-G01 is Go installed-probe capture/process ownership in [observations](observation-reliability.md); the separate parked browser simplification remains |
| LitAnalyzer and gocyclo, GB-MAINT-23/30 | Move H06-F1 to gocyclo; correct both maintenance and adapter chapters | [check-complexity.sh](../../../scripts/check-complexity.sh) masks analyzer status and classifies stdout. Retaining gocyclo does not close that gate defect; no LitAnalyzer failure is established |
| Indirect Go module set, GB-MAINT-29 | Remove COST-GO-SQLITE-FIXTURE-1 from the whole set | The [dependency audit](dependency-and-architecture-cost.md) identifies one direct SQLite seeder edge; its direct integration row retains the finding. Module removal is not selected |

These are evidence-association corrections, not five new implementation defects.
They leave the library recommendations unchanged. No conclusion is accepted
merely because reviewers agreed on its label.

## Retained compatibility: local CA algorithm profile

**CA-ALGORITHM-PROFILE-1 / C, decision pending.** The
[installation guide](../../guides/installation.md) explicitly names RSA,
named-curve ECDSA and Ed25519 families, supported self-signatures, canonical DER,
strict parameters, CA/key-use constraints and original-DER fingerprint meaning.
It does not enumerate ECDSA curves. The recorded
[Caddy 2.10.2 fixture](../../../cmd/soda-install/src/x509/tests/fixtures/caddy-2.10.2-root.pem)
is P-256/ECDSA-SHA256. Synthetic tests for other implemented curves and key
families show implementation intent and regression coverage, not an external
producer requirement for every algorithm. Retired Go behavior does not create
a permanent compatibility obligation for this unreleased product.

[local_ca_fingerprint](../../../cmd/soda-install/src/setup/local_ca.rs) admits
the PEM, verifies the certificate and hashes the original DER. Keep its strict
signature, original TBS/DER and trust-authority rules. Host SSH/point P-384 and
P-521 consumers are separate from installer CA admission.

The existing COST-INSTALLER-KEY-FEATURES-1 cut preserves **all currently admitted
algorithms** while trimming excess direct feature requests. It can proceed on
that contract; it does not resolve the permanent algorithm profile. C owns the
separate decision: record intended actual producers and exact accepted
algorithms/parameters in the owning guide before any narrowing, then align
admission, producer fixtures, refusal tests and dependency features. Acceptance
must preserve fingerprints, original signed bytes and strict verification, and
must check host/release consumers before deleting a shared dependency. This
audit selects no algorithm deletion and adds no generic PKI subsystem.

## Supported adequacy claims and their limits

| Boundary challenged | Contract and decisive caller/API evidence | Supported conclusion; explicit limit |
| --- | --- | --- |
| Typed crypto, entropy and SSH | [Crypto profiles](library-integrations/crypto-profiles.md): typed constructors/verifiers, caller-selected key purpose and raw fingerprints; selected ssh-key handles OpenSSH forever-valid certificates | Retain the engines and fail on OS randomness errors. No synthetic entropy fallback, whole-workspace curve removal or native signer qualification follows |
| PostgreSQL and Unix HTTP | [Native maps](library-integrations/native-engines.md), Identity Store/Tx and four Unix client consumers: typed rows/params, real drivers/runtimes, absolute deadlines and connection disposition | Retain protocol engines; SIMP-I-PG-1 removes representation bridges. Store exclusion, cancellation/drain/discard/join and uncertain COMMIT remain. Source agreement is not live PG proof or a reason to add pooling |
| HTTP upgrade and WebSocket pump | [Concurrency audit](concurrency-and-termination.md), host upgrade/read-ahead and owned pump/session lifetime | Retain Hyper/tungstenite and the single upgrade/pump owner. Repaired detached reader stays repaired; no complete descendant termination or installed session proof |
| SDK typed snapshots versus transport | [API challenge](library-integrations/adapter-challenges.md), decodeSnapshotAnswer and sibling SDK at `c92db11c` | SIMP-SNAPSHOT-1 can remove the local DTO/JSON detour while retaining snapshot validation. SIMP-SDK-1 separately requires per-dial peer pinning, refusal/admission/push and credential custody; one does not gate the other |
| JSON and output profiles | [Representations](data-representations.md), actual strict decoders, current unsigned emitters and signed consumers | Upstream syntax plus required admission remains. Only matching producer contracts support consolidation; exact signed bytes, fingerprints, duplicate/depth and alias rules stay. Old pretty-print goldens alone do not impose compatibility |
| File, process and archive libraries | [Custody](file-and-process-custody.md), same-FD checks, Runner and current tar/gzip scanner consumers | Retain syscall/codec/staging engines. Bounds, confinement, complete EOF/trailer drain, publication and cleanup results remain application duties; RAII or collecting a body is not proof of completion |
| Go, browser and developer engines | [Go/browser map](library-integrations/go-browser.md), actual session/terminal owners and tool wrappers | Retain the selected engines; framework use does not fix lifecycle, authorization, queue or tool-error defects. SIMP-BROWSER-1 remains parked until it demonstrates fewer duties/state owners |

The established Go server/domain/coordinator/Store/client and Rust
system/privileged/runtime ownership remains controlled by
[GUIDANCE-01/03/04/05/06/08](review-assignments.md#guidance-conflicts-and-controlling-decisions).
Those choices and intentional all-page Forgejo presentation are product/language
constraints, not a technical proof of minimum cost. No new runtime, facade,
package, state owner or global compatibility layer is justified by this review.

## Canonical claim ledger

Each row links the owning current audit, which supplies scope, prerequisites and
acceptance. Grouping shares evidence; it does not merge owners or dispositions.
Unless stated otherwise, the result is source-supported and behavior is unrun.

| Canonical records / owning audit | Contract, decisive challenge and resulting scope | Material limit |
| --- | --- | --- |
| B03.C, OBS-G01, H06-F1 — [observations](observation-reliability.md) | LimitReader can create false EOF; installed capture grows before postcheck and lacks complete process/drain custody; complexity wrapper can classify analyzer failure as PASS. Repair their outer observation owners | Codec/tool adoption does not fix these wrappers; no exploit, descendant stall or tool-error run here |
| OBS-R01, L16.G — [resources](resource-bounds.md) | RecallLog line count does not bound retained fragment/reason bytes; per-file secret caps do not admit aggregate input before derived copies. Bound before growth while preserving evidence errors and redaction variants | No OOM or unbounded single-secret claim; Aho-Corasick remains optional |
| OBS-S01, OBS-W01, OBS-D01 — [observations](observation-reliability.md) | Redacted artifact paths can overwrite requested hashes; unit-stop failure precedes runtime release; readiness can accept SSH success after Phase expiry. Correct finalizer/worker/readiness joins | Hash loss is not a leak; worker loss is conditional on actual unit/removal outcomes; no native effect executed |
| OBS-S02 — [observations](observation-reliability.md) | Withdrawn in the owning audit; discovery must not resurrect it | No open defect or implementation task |
| D03-F1, D06-F2, H01-F3, I06-F1 — [concurrency](concurrency-and-termination.md) and [authority](authority-and-state.md) | Completed concurrent drain/error joins, bounded Phase child, owned WebSocket reader and broker terminal fence remain distinguished from newer caller problems | Reused source evidence does not qualify native builds, all callers or host process termination |
| CON-G01, CON-G02, CON-M01, CON-M02 — [concurrency](concurrency-and-termination.md) | Coordinator lock release precedes admitted-handler completion; SDK bootstrap mutex wait misses caller cancellation; Muse reap/signal custody and pre-admission workers/retained handles need their existing lifecycle owners | No second-process overlap, PID reuse, token corruption or exhaustion observed; no generic scheduler/process framework |
| AUTH-I-ACQ-1, AUTH-I-CLOSE-1 — [authority](authority-and-state.md) | Separate lease reserve/link can orphan a reservation on failed observation; close treats lookup errors as absence. Preserve distinct execution IDs, terminal fence and NotFound idempotence | Inspected callers do not routinely retry acquisition IDs; no credential leak or execution reopening established |
| P04-F1 — [authority](authority-and-state.md) | User requires explicit final saved-key confirmation at mutation time, including concurrent changes. Current deletion path lacks it | Saved preference deletion differs from installed Project empty-set confirmation and does not revoke installed SSH |
| F07-F1/F2, F09-F1, F10-F1/F2 — [authority](authority-and-state.md), historical [F07](reviews/F07.md), [F09](reviews/F09.md), [F10](reviews/F10.md) | Current dispatch packet/prompt split, published-parent correction enumeration, allowance callsites and review-cycle intent preserve the distinct authority/completeness/withdrawal/wiring/recovery gaps | Historical locations are not current selectors; no native launch/write or race replay. Missing wiring does not justify a new service |
| F08-F1/F3 — [authority](authority-and-state.md), historical [F08](reviews/F08.md) | [WithdrawDispatch/registerDispatchTx](../../../internal/store/factory_grants.go) still split capture from close; acceptance withdrawal versus [publication registration](../../../internal/store/factory_publications.go) has a separate capture interleaving. [finish_run](../../../lib/host/src/factory/finish.rs) checks stop before return, but [update_receipt](../../../lib/host/src/factory/launch.rs) can later overwrite stop-owned attribution | Preserve both existing-owner capture duties. No native effect or lost credential observed; later reconciliation is not ruled out. Confirmed settlement does not erase stop provenance |
| F12-F1 — [authority](authority-and-state.md), historical [F12](reviews/F12.md) | Current Merge.Validate and [completeMerge](../../../internal/factory/control/merge_evidence.go) require exact head equality, including BaseTip. A legitimate target advance can retain reachability while failing equality | Conditional completion refusal/profile mismatch, not unrelated-result admission. Actual G07/native attribution/reachability or explicit exact fast-forward contract remains prerequisite; deleting equality alone is insufficient |
| O02-F1, O05-F1, P05-F1 — [workflows](workflow-traces.md) | Post-publication revoke uncertainty, backup rotation selecting the new publication and Project Start not reopening stop-dispatch cause remain supported narrow workflow corrections | No provider/database/process execution; preserve uncertain operations without automatic replay |
| D09-E3 — [workflows](workflow-traces.md) and [D09](reviews/D09.md) | Exported finalize prior-state/retained-ledger question has no current active pipeline caller beyond definition/config test | Dormant helper correctness or removal decision, not demonstrated active publication failure |
| ID-CFG-01, ACC-SNAPSHOT-FILE-1 — [custody](file-and-process-custody.md) | Settings read precedes strict-decoder cap; snapshot checks metadata then reopens. Bound the consumed opened descriptor; preserve actual path and strict admission | Same-FD admission does not make a file/tree coherent under concurrent in-place writes; no privilege exploit claimed |
| CUST-G-SOURCE-1 — [custody](file-and-process-custody.md) | Source reads its bundle before checking size; no current production caller found. Decide retention with L18 first, then delete that method or repair its bounded same-file read | Adjacent sourceRepository remains live; repository-scoped caller absence is not blanket package deletion |
| ENROLL-CLEANUP-RESULT-1, CUST-C-BUTANE-CLEANUP-1 — [custody](file-and-process-custody.md) | session.close can mask primary EnrollUncertain; Butane unlink/clone/try_wait exits can miss owned finalization. Compose errors and finish existing custody | No child survivor/native enrollment effect established; cleanup failure never licenses uncertain mutation retry |
| RES-I-PROBE-OUTPUT-1, RES-I-LEASE-ENUM-1, RES-GO-IDENTITY-CONNECTION-LIST-1, RES-GO-ACCEPTANCE-DEPENDANTS-1 — [resources](resource-bounds.md) | Provider probe retains output before admission; lease/connection/graph producers materialize aggregates. Set supported bounds or continuation at the producer with complete progress/error semantics | Go downstream body cap cannot bound Rust producer allocation; silent truncation can falsify revocation/inventory/cascade. No arbitrary cap or observed exhaustion |
| SIMP-I-PG-1, SIMP-FD-1 — [API challenge](library-integrations/adapter-challenges.md) | Remove Field/Row/JSONB string bridges at Store's schema; replace project-terminal integer open flags with typed rustix flags | Preserve deadline/Tx/cancel uncertainty and NOFOLLOW/CLOEXEC/errno/custody. libc/rustix rows are one cut; no global libc deletion |
| SIMP-I-JSON-1, SIMP-GJSON-1, REP-HOST-STRICT-1, REP-CFG-1 — [API challenge](library-integrations/adapter-challenges.md) and [representations](data-representations.md) | Typed decode plus duplicate/depth admission can remove remapping; settle each actual reader's alias/null/unknown/number profile. Preserve EqualFold HaveNodeKey and source bytes | Profile-gated, not globally exact-case or experimental Go JSON v2; B03.C is independent outer EOF admission |
| SIMP-PEM-1 — [API challenge](library-integrations/adapter-challenges.md) | Typed PEM can remove manual body/line/Base64 mechanics after exact bounded single-certificate/no-extra-object admission is settled | DER/TBS/fingerprint, CA/signature/key role and actual Caddy producer remain; algorithm narrowing is the separate question above |
| REP-ACC-EVIDENCE-ROUNDTRIP-1, REP-FMT-1, SIMP-ASSET-NODE-1, SIMP-REL-EMIT-1 — [representations](data-representations.md) | Remove compact-parse/formatter detours in unsigned output; provisioning PythonFormatter consumes base.json. Preserve scrub/collision/depth/integer/unknown-key/output/hash/publication rules | Provisioning differs from candidate/lab formatting. Current semantic/deterministic output matters; retired Python/Go bytes are not permanent requirements |
| SIMP-REL-WIRE-1, SIMP-REL-ORDERED-1, SIMP-INSTALL-OCI-1 — [representations](data-representations.md) | Shared DTO/tree/OCI cuts need matched producer/readers and actual number/order/native profiles; preserve descriptor path/type/digest/layer rules | Conditional. Current OCI reachable graph is bounded per blob/aggregate; no arbitrary-directory unbounded claim or generic Value deletion |
| SIMP-SNAPSHOT-1, SIMP-SDK-1, SIMP-BROWSER-1 — [API challenge](library-integrations/adapter-challenges.md) | Local typed snapshot cut supported; SDK transport equivalence held on upstream capabilities; browser ownership remains parked pending demonstrated reduction | Keep snapshot completeness/revision/digest/visibility, credential/peer admission and terminal detach-versus-End/queue cleanup; no net browser deletion proven |
| COST-GO-SQLITE-FIXTURE-1, COST-HOST-BUILD-EDGE-1, COST-IDENTITY-AEAD-FEATURES-1, COST-INSTALLER-KEY-FEATURES-1, COST-BOOTSTRAP-SEAM-1 — [dependency cost](dependency-and-architecture-cost.md) | Exact seeder/test edge, host native-predicate dependency, direct AEAD/key features and two-call RunnerProduction stub are supported bounded cuts | Keep acceptance SQLite, delivery payload, fallible RNG, current CA algorithms/DER gates and real Production bridge. No measured artifact shrinkage or broad dependency removal |
| L10.N4, CFG01 — [workflows](workflow-traces.md) and [maintenance](integration-maintenance-result.md) | Host curl replacement holds on resolver-inclusive deadline/Executor/credentials/caps; INI parser fit holds on native continuation/deployment evidence | Each holds only its replacement. Setup ureq N3 and independent configuration/adoption work remain separate |
| TEST-ACCEPTANCE-SSH-EMULATOR-1, TEST-REL-PRODUCTION-BOUNDARY-1 — [tests](test-evidence.md) | Real orchestration with substituted SSH/keygen/transfer or execute/capture/next proves only chosen policy/CLI joins | Evidence-label limits, not parser/runtime defects or installed/native qualification |
| TEST-RETIRED-PYTHON-OUTPUT-1, TEST-REL-ORACLE-1, TEST-RETIREMENT-GUARDS-1 — [tests](test-evidence.md) | Challenge legacy output goldens per actual consumer; only six named historical-path/source-token guards are selected for retirement. Preserve current Rust behavioral assertions, TestDependencyDirection and no-Python gate | Raw signed inputs and fingerprints remain exact; Go source ownership tests do not certify Rust behavior; no blanket fixture deletion |
| TEST-SOURCE-CLI-GATE-1, TEST-CARGO-RESULT-1 — [tests](test-evidence.md) | Ambient/personal-home Muse selection enters ordinary source tests; failed Cargo Run with empty stderr can become cached success/stale output path. Pin/select optional CLI and bind helper outcome | Echo case intends zero spend; invalid-Meta-auth case does not establish an observed provider request. No stale executable run observed |
| JOIN-HOST-PROBE-PATH-1, JOIN-INVOKED-IDENTITY-1, JOIN-BROWSER-CACHE-1 — [operational joins](build-and-operational-joins.md) | Host stdin script has five obsolete prefixes unaffected by service rewrite; guide claims running image identity while probe inspects stored content; direct template cache tokens have a separate owner policy question | Source staging/guide mismatches supported, no installed failure. No stale browser token established; cache behavior remains conditional |

## Integration and completion limits

Keep the bounded cuts at their existing owners. Verify actual last callers,
retained policy, current producer fixtures and failure/cleanup joins before
deleting machinery. Preserve completed L01/trust/terminal repairs; this challenge
does not turn unresolved caller correctness into completed structural work.
Each later coherent implementation packet still needs focused checks and an
independent review with source/contract evidence, followed by an explicit-path
commit. The coordinator retains manifests, locks and expensive-check custody.

This completes the scoped source challenge and evidence reconciliation. It does
not rerun the historical 80-slice audit or demonstrate implemented corrections.
Normal-return PG tests without a DSN, Go/Bun source checks, fake native peers,
build recipes and installed observations remain different evidence classes.
Native qualification follows the named prerequisites and actual invoked bytes;
reviewer agreement alone cannot advance it.

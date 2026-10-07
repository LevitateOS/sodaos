# Implementation task list

Execute through the [parallel schedule and file ownership rules](implementation-lanes.md).
Unchecked entries are **deferred**, not implementation authorization; checked entries preserve their recorded completed scope. M is a
behavior-preserving move/split/consolidation, C a named correctness correction,
and V verification of the real implementation. A gated C stays blocked while
independent M/V planning can continue. Dependencies below are correctness or
shared-file dependencies; they are not whole-lane barriers.
Apply each dependency to its required boundary output for the selected M/C/V
subtask, not full-packet completion. SDK/native qualification uncertainty does
not block locally specified operation registration or settled extraction.

Each packet lead must account for every assigned duty in its linked slice records
and maps, including retained unchanged files, mixed responsibilities, tests,
fixtures and real build/install callers. The paths here locate the defining
owners; the current review allocations specify their exact leaves. Do not
implement just the named examples or repeat another packet's shared-file edits.
Current source is post-integration canonical (run 20261005; implementation STARTED — see ticked boxes), superseding the clean `de65ff68` planning snapshot. The `f7e9cf9d` source audit and `d7e565aa`/`0d8d3b8e` structural baselines remain the preserved historical identities. All 80 slices have one primary packet
below; shared duties route through their physical writer in the lane schedule.

## Current task state (2026-10-07)

Implementation is active under the owner's “go”, “commit early and often” and
subsequent all-tasks goal. Current reconciliation uses source `5794a7d9`;
`04286c48`/`0caf6b91` remain the pre-implementation planning identities. The
first twelve selected packets are committed, followed by readiness/artifact
identity, installed-probe joins, duration cleanup and typed SDK snapshot cuts.
Completed checks are source/development evidence; no installed, provider or
native-worker qualification is inferred. Seven pre-existing dirty guidance
files remain untouched. Current tracked paths total 2,564, including the new
H06 Cargo-helper regression file. The established responsibility snapshot and
historical validity reviews retain their original scope; affected owner deltas
are recorded selectively, with full R02.targets regeneration still pending.

The [finding allocation](execution-findings.md) is the detailed subtask section
of this same plan: 75 prior canonical dispositions plus the CA profile question,
each with one accountable owner, current scope, prerequisites and acceptance.
Existing 80-slice corrections outside that challenge remain in their rows and
Q gates below. Select ready correctness first, expensive boundary/profile proofs
second, then settled caller cuts. Same-file corrections and their simplification
have one writer; a held profile does not hold a disjoint task. Completed M/C/V
entries stay checked; the new suffixed follow-ups below are separate pending work.

The [library-adoption work](library-adoption.md) replaced the selected custom
hash/encoding/trust, PostgreSQL, HTTP/WebSocket, URL/IP/time, file/process,
archive/ELF/SVG and CLI/metadata engines. **L00–L09, L11–L15 and L18 are complete
at their recorded source scopes; L10.N3 is also complete.** L15's source is the
separately identified Fountain checkout. L00–L18 are subpackets of the existing
owners, not new slices. Their defining scope and acceptance remain in the
chapter; the detailed rows below retain exact revisions and evidence limits.
Completed conversions supersede pending decomposition of those old engines.
Application/lifecycle corrections and their Q gates remain separate.

| Remaining work | Current disposition and exact next boundary | Owner / task |
| --- | --- | --- |
| Current inventory / later target regeneration | Current source responsibility inventory/maps are complete at their recorded snapshot. Keep selective owner/caller deltas with each change; regenerate the full desired tree/decomposition after replacement boundaries settle. Existing task allocation is reconciled here. | Coordinator / R02.inventory complete; R02.targets/joins pending |
| Operator request bound | Completed in `a1fec662`: cap+one/read-error refusal before strict decoding, with focused production endpoint checks. Remaining B03 authority findings retain Q8. | B / B03.C partial |
| Aggregate secret inputs | Completed in `7334f36b`; candidate count/bytes are admitted before retained variants, with per-file staging and escaped-pattern admission separately bounded. | C / L16.G complete |
| Host provider HTTP | Held until resolver-inclusive cancellation and native Executor custody are proved; setup HTTPS is already converted. | C, then A handoff / L10.N4 |
| Effective configuration | Native corpus and candidate assessment are complete; CFG01 cutover is held on parser fit, deployment override evidence and dependency admission. CFG02 remains retained. | C / L17 |
| Optional matcher conversion | Consideration complete; adoption deferred. Reconsider only after L16.G and demonstrated value plus construction/streaming admission. | C / L16 |
| Other correctness and caller cuts | Follow the current [finding allocation](execution-findings.md), then remaining unchecked A/B/C scopes with precise Q gates. Parked A34/C41/browser seams are optional; B27 is integrated and is not replayed. | Named accountable finding owner; exclusive physical handoffs |
| Wider verification | R03.L covers completed adoption source checks. Broader R03 checks and selected R04 native/installed/provider qualification remain open at their own scopes. | Coordinator |

The coordinator's [R03.L source verification](implementation-lanes.md#coordinator-checklist)
is complete for the implemented adoption scopes at `d7eca882`. Fresh focused
evidence tests and locked offline all-target Rust workspace checks pass; final
unchanged-source packet receipts and independent reviews retain their scope.
The owning Go fixture row is reconciled in `93b50615`. L10.N4, CFG01,
R02.targets/joins and selected R04 qualification remain open; none is
reclassified by this source-verification result.

## A — Projects, identity and Spaces

### A00 Shared defining extraction

Lead A; supporting packet, no additional slice. Extract each host/guest/browser
monolith at the same time as its selected same-file correction or move; no
mandatory repository-wide extraction barrier. A owns host/guest extraction;
route the browser workspace extraction to B.

- [x] **A00.M** [run 20261005: DONE, integrated 56c095ed (6 in-place roots, whole-file; guest verified bound)] Bind the reviewed descendants of host `project`, `account`, `preparation`, `prepare`, `texec`, terminal/factory and guest PTY/protocol plus `frontend/spaces/sodaspaces-workspace.ts`. Keep one defining DTO/fixture/module root before handing descendants to other packets. Use [host](decomposition/host-runtime.md), [Project](decomposition/project-runtime.md) and [browser](decomposition/browser-and-design.md) allocations.
- [x] **A00.V** [run 20261005: DONE (real imports, test visibility, relative fixtures, bin+lib ownership; host 641 + guest 92+12)] Check real imports, private test visibility, relative fixtures and same-package binary ownership through R01. No new facade, duplicated state or size-only helper files.

### A01 Project creation, membership and access

Lead A; [P01](reviews/P01.md), [P02](reviews/P02.md), [P03](reviews/P03.md), [P04](reviews/P04.md). A00 for shared native files; B owns Go counterparts.

- [x] **A01.M** [run 20261005: DONE, integrated d4597974 (11 commits 1d9d38b2..31bf88cf: R100 wholesale host+terminal, account/ssh/project splits, account fold, OS move-follow, reap fix) + R01 859cfb2d (members, project-account bin, lock -account stanza, zerocopy pin kept); 5 conflicts resolved (2 take-deletion, 3 take-A-side)] Retain Go `internal/project`, Project API/Store/client owners; move native creation/profile/OS/connection/account/SSH duties to `lib/host/src/project/` and account/SSH descendants. Fold the existing account binary into `cmd/soda-project-terminal` with C/R01 producer selectors.
- [x] **A01.V** [run 20261005: DONE on canonical mixed tree (859cfb2d): TestAccount* 12/12 green (11 previously red), terminal 19+92+12+12, host 645/0, image 59/59, build 50/50, Go build clean, successors green, fmt clean, clippy 1 pre-existing (dbackend, byte-identical to base); native transport pumping + provider qualification separate] Preserve reservation reconciliation, immutable profile/login, explicit Join and exact admitted public-key revision/apply through existing real account/Project suites. Optional creation Tailnet intent is gate Q4, not new state.

- [x] **A01.C-saved-key** [2026-10-07: `ddd2f103`; real PostgreSQL concurrency/API and browser checks passed; independently reviewed] B is the accountable Go/frontend/Store owner for P04-F1 mutation-time final saved-key confirmation; native installed-key policy remains separate. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### A02 Preparation authority and Project lifecycle

Lead A; [P05](reviews/P05.md), [P08](reviews/P08.md), [P09](reviews/P09.md), [P12](reviews/P12.md). B01/B03 grant/Store handoffs; A00 native extraction.

- [ ] **A02.M** Keep Go preparation decisions/Store CAS/API and Rust `lib/host/src/{preparation,prepare,project}` lifecycle definitions with their current processes and distinct heads/markers.
- [ ] **A02.C** Specify and correct P05-F1 cause-specific Project-stop reopening and P09-F1 actual approval authority only after Q1/Q2 resolve. Preserve independent maintenance holds, other withdrawals and marker-first/CAS failure reporting.
- [ ] **A02.V** Exercise existing real start/stop/preparation/approval subjects, separate head revisions and failure paths; coordinate actual dispatch reopening with B03.

### A03 Guest role custody and checkout supervision

Lead A; [P06](reviews/P06.md), [P07](reviews/P07.md). Shared guest-package ownership with A07; no simultaneous module-root edits.

- [ ] **A03.M** Retain current guest role custody, checkout supervision, executable/install/hash selectors and actual compiled-helper subjects; reconcile any remaining package-fold duty against current source. Further custom SHA/JSON codec extraction is superseded by L03/L04, without changing child or checkout authority.
- [ ] **A03.C** Source repairs for P07-F1 capture retirement (`368e1842`) and P07-F2 preexisting checkout preservation (`823334ac`) have landed, with later custody changes. Reconcile the complete current supervision/caller chain before closing this combined correction entry; preserve those repairs rather than schedule them again. Broader acceptance remains open.
- [ ] **A03.V** Use actual compiled role-helper subjects for capture/cleanup and checkout failure, with exact requested binary selection; no Python predecessor recreation.

- [x] **A03.M-adapters** [2026-10-07: `fb15e33b`; typed FD/flag callers and actual filesystem checks passed; independently reviewed] A owns SIMP-FD-1 typed project-terminal flags with retained no-follow/descriptor/errno contracts. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### A04 Shared tools, nested services and volumes

Lead A; [P10](reviews/P10.md), [P11](reviews/P11.md). C owns system/Containerfile/compiler joins and the Muse-maintain command/stage implementation; A owns host tool-observation counterparts.

- [ ] **A04.M** Retain mise/profile/systemd/OCI/workload substrates and selected Project definitions; preserve native tool observation and Muse staging authority. Manual archive-header extraction is superseded by L13/CF-07; stream/FD/deadline/child custody remains in the current maintenance owner.
- [ ] **A04.V** Preserve actual script subjects, read-only readiness, ordered bus/tool staging and volume lifetime. Rebind existing source assertions with their true implementation owners.

### A05 Broker custody and execution state

Lead A; [I01](reviews/I01.md), [I02](reviews/I02.md), [I03](reviews/I03.md), [I04](reviews/I04.md), [I05](reviews/I05.md), [I06](reviews/I06.md), [I10](reviews/I10.md). C01 bounded parser work; B01/H02 schema/admin-reader handoffs.

- [x] **A05.M** [run 20261005: DONE, integrated 8c489d16 + R01 91c6d3e7] Consolidate broker/provider package ownership under `cmd/soda-identity`; keep one Controller/State/Store/Tx and reviewed enrollment/grant/acquisition/registration/retirement/store/wire descendants. Retain Go domain/client and canonical PostgreSQL schema with its existing mirror assertion.
- [x] **A05.C** [run 20261005: DONE, lane V closed on live disposable PG; overall acceptance CLOSED via CORR-C-001 (merged + real-tree proven)] Correct I06-F1 terminal execution fencing after successful admitted End/expiry/return using actual provider-specific retirement; preserve legitimate recovery before any returned binding. Do not add I05 terminal-Muse InvocationID enforcement or public history/retention from unresolved requirements (Q3/Q4).
- [x] **A05.V** [run 20261005: DONE, 25 unit + 10 integration executed, 0 skips] Exercise actual same-ID terminal fencing, current grants, encrypted custody and atomic append subjects; coordinate sponsor metadata and fixture rebinding with B01 before shadow-write retirement.

- [ ] **A05.C-custody** New A-owned AUTH-I-ACQ-1/AUTH-I-CLOSE-1 reserve/link and close-observation corrections; completed I06-F1 fence remains checked. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **A05.C-bounds** ID-CFG-01 startup is complete in `7900a0f2` with four actual settings checks; A retains RES-I-PROBE-OUTPUT-1 provider output and RES-I-LEASE-ENUM-1/RES-GO-IDENTITY-CONNECTION-LIST-1 producer bounds; unknown aggregate profiles gate only their own changes. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **A05.M-adapters** A owns SIMP-I-PG-1 and profile-gated SIMP-I-JSON-1/REP-HOST-STRICT-1 at their actual identity/host readers. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [x] **A05.M-dependencies** [2026-10-07: `5c63e6c5`; crypto checks passed, versions unchanged; independently reviewed] A supplies the exact Identity AEAD feature cut; coordinator owns manifest/lock integration and retains fallible RNG. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### A06 Provider adapters

Lead A; [I07](reviews/I07.md), [I08](reviews/I08.md). A05 provider-support/construction ownership must be fixed before independent child-module work.

- [x] **A06.M** [run 20261005: DONE, integrated db2b92d7 + R01 eb6b14d6, predecessor retired] Fold the actual Codex/Muse implementations into private `cmd/soda-identity/src/providers/{codex,muse}` descendants; preserve one shared support/type owner and direct imports.
- [x] **A06.V** [run 20261005: DONE, 41 unit + 10 integration on live PG, 0 skips] Preserve pinned constructors, sanitized protocol, stop-before-read/cleanup, serialized Codex execution and concurrent immutable Muse lease behavior. Existing adapter assertions are source tests; real provider calls remain separate qualification.

### A07 Native launch and interactive attachment

Lead A; [I09](reviews/I09.md), [S04](reviews/S04.md), [S05](reviews/S05.md). A00/A03 guest handoffs, C02 host transport and B03/B04 native run/export handoffs.
Compose/Muse-maintain command changes route to C; reserve their current module
roots once across their P/I/H duties, while A owns host/guest changes.

- [ ] **A07.M** Retain the R02/A34 host/companion binding and R02/C41 Muse argument, connection, caller and cleanup-order duties at current owners; optional caller/cleanup splits require a concrete owner benefit and current-adapter diff. Define remaining run/binding/lifecycle/output/artifact duties directly under existing host/guest owners; preserve distinct Codex/Muse launch and custody. Pending manual HTTP/WS frame/handshake/pump decomposition is superseded by L09. Guest/Compose JSON, encoding and randomness use L03/L04/L12 through the physical owners; no new processes or forwarding facades.
- [x] **A07.C** [run 20261005: LANDED 865f0e2f (H01-F3) + 9d5a9281 (S04-F1/S05-F1); H01-F3 reap repair LANDED (6edf58b7+31bf88cf, integrated d4597974) + kill-then-wait source-verified; native transport pumping unproven (helpers lack output reader), provider/native qualification separate] Correct S04-F1 C-string ABI and S05-F1 select writable-set handling; implement canonical H01-F3 pump mutex correction through C02's A-owned host handoff. Q3 gates harness/invocation-dependent enforcement only.
- [ ] **A07.V** Exercise actual PTY/relay/attachment and replacement transport subjects for concurrent input/output/close, reservation consumption, exact binding and honest retirement. L09 upgrade/read-ahead/wakeup/close ownership must be proven before its cutover and expensive native runs.

- [ ] **A07.C-muse** New A-owned CON-M01/CON-M02 signal/reap, supervisor joins and aggregate listener custody; completed H01-F3 remains checked. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [x] **A07.M-dependencies** [2026-10-07: `5c63e6c5`; iconfig checks passed, direct release-build edge removed; independently reviewed] A supplies COST-HOST-BUILD-EDGE-1 current iconfig predicate/direct dependency cut; delivery payload authority remains. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### A08 Spaces browser inventory and viewer

Lead A; [S01](reviews/S01.md), [S02](reviews/S02.md), [S03](reviews/S03.md), [S06](reviews/S06.md). B owns physical Go/browser files; A00 releases workspace descendants.

- [ ] **A08.M** Retain bounded Go inventory/output admission and explicit browser decoder/workspace/layout/view/CSS owners with one actor/epoch/slot definition. Retire the proven-unused widths predecessor after its actual assertions are rebound.
- [x] **A08.C** S01-F1 is already corrected in the current `spaceItem`→`spaceNetwork(row)` decoder and per-row Store/API observation. Current-source inspection and all 22 actual `spaces-api.test.ts` cases pass, including distinct row values and ignored collection state. The earlier audit finding remains historical; no new implementation patch was needed.
- [ ] **A08.V** Preserve stale-callback rejection, renderer/slot lifetime, restoration/layout and read-only Factory viewing through existing real browser subjects.

## B — Factory and Forgejo

### B01 Standing authority, sponsorship and accounting

Lead B; [F01](reviews/F01.md), [F02](reviews/F02.md), [F03](reviews/F03.md), [F04](reviews/F04.md). A05 broker-admin/custody and C02 PostgreSQL challenge; schema has one writer.

- [x] **B01.M** [run 20261005: DONE, integrated 60964bea + CORR-B-001 via 8d835377] Split current Go grants/authority/domain resources/sponsorship, policy/settings/status APIs and Store concerns inside the existing dashboard. Preserve canonical Go DDL/Rust mirror and real cross-package fixtures.
- [ ] **B01.C** Close the recorded exact specification gates, then correct F01-F1 effect/receipt atomicity, F03-F1 connection-wide rolling-window accounting, F04-F1 sponsor/beneficiary/Project attribution and F04-F2 broker-admin metadata reads. Complete F04-F2 before retiring production dashboard credential-shadow methods; preserve/rebind actual native/ST15/Store/web seed consumers to the reviewed single `internal/store/identity_fixture.go` owner with canonical sealing/atomic append.
- [x] **B01.V** [run 20261005: DONE on disposable PG per EVID-B-002] Use actual disposable PostgreSQL command/receipt/quota subjects and real broker-admin caller assertions; do not create another ledger or treat fixture writes as production broker ownership.

### B02 Accepted inputs and readiness

Lead B; [F05](reviews/F05.md), [F06](reviews/F06.md). B01 for shared Store leaves; B07 for the existing publication diagnostic owner.

- [x] **B02.M** [run 20261005: DONE, integrated 8d835377] Extract accepted evidence/initial/status and readiness/prerequisite/sweep/traversal/API concerns with their actual snapshots and tests.
- [x] **B02.C** [run 20261005: DONE; Q5/Q6 unknowns still gated] Correct F05-F1 recorded replay versus fresh stale-screen admission and F06-F2 diagnostic dependency at the existing `StatusError`/publication owner. Unknown native readiness/merge observations remain Q5/Q6 gates.
- [x] **B02.V** [run 20261005: DONE on disposable PG] Test the real replay and intake subjects, source ownership assertions and fresh versus persisted authority; no replacement poller or status writer.

### B03 Dispatch and intervention

Lead B; [F07](reviews/F07.md), [F08](reviews/F08.md). B01/B02, A02/A05/A07 boundary outputs; serialize current grants/dispatch/Store packet files before leaf handoff.

- [x] **B03.M** [run 20261005: DONE, integrated 1e2de63e (dispatch/assignment/store/lifecycle splits pure; Rust changes as specs to A)] Split actual Go dispatch/occupancy/recovery/packet/queue/settlement and tests; send Rust host run/receipt/launch/finish/stop and neutral model changes to A. Preserve same dashboard/host topology.
- [ ] **B03.C** Correct F07-F1 authority tuple freshness, F07-F2 recorded prompt context, F08-F1 registration/withdraw capture ordering, F08-F2 fresh retry attempt and F08-F3 stop-owned settlement. Resolve exact grant/quiescence/transaction specifications first; Q3 blocks undecided simultaneous provider-family selection. The independent operator 4 KiB limiter-EOF correction is complete in `a1fec662`, with cap+one/read-error admission and focused endpoint checks; it does not close the remaining authority findings.
- [x] **B03.V** [run 20261005: DONE on disposable PG (11 pkgs, -race 4/4, raw logs archived, 0F/0S)] Exercise actual authority/registration/withdraw/retry/stop races with existing real Store and coordinator subjects; preserve historical attribution and usage.

- [ ] **B03.C-lifetime** B owns CON-G01 dashboard admission/drain/flock/DB lifetime and RES-GO-ACCEPTANCE-DEPENDANTS-1 exhaustive traversal bounds; select supported graph/progress profile first. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### B04 Publication, review and correction loop

Lead B; [F09](reviews/F09.md), [F10](reviews/F10.md). B03 durable attempt/settlement/registration, B07 native adapters, A07 shared exports; Q3/Q5 apply to dependent loop work.

- [x] **B04.M** [run 20261005: DONE, integrated 6636851f (publication 5-way + 4-way + test splits pure, ZERO added lines)] Extract Go publication hooks/correction/review/allowance leaves and A-owned shared native takeover/output/artifacts, importing their direct defining owners.
- [ ] **B04.C** Correct F09-F1 common operation capture/cancel registration and F10-F2 immutable review operation retention before submission/replay; wire the established finite F10-F1 reviewer/correction loop after its exact prerequisite specification. No second scheduler or review service.
- [x] **B04.V** [run 20261005: DONE on disposable PG (13 pkgs, 27+8+12 targeted; 6 SKIP pre-existing native opt-in gate)] Verify the actual registered-operation replay/capture and finite allowance subjects. Qualify same-Project review/fix/new-head/fresh-review separately; primitive snapshots or mocks alone do not prove the loop.

### B05 Verification and merge completion

Lead B; [F11](reviews/F11.md), [F12](reviews/F12.md). B02/B04/B07 fresh candidate/evidence; Q5 actual native result reachability blocks the dependent C.

- [x] **B05.M** [run 20261005: DONE, integrated 51e2bf59 (check/merge splits P1-P4+T1-T4, 26 files pure; 230 decls token-preserved per Codex)] Split existing check assessment and merge/evidence/reconcile/withdraw concern owners; cached assessment remains observation, not admission authority.
- [ ] **B05.C** [run 20261005: HELD (Q5, zero correction edits)] Correct F12-F1 attributable result reachability only with the actual approved producer/SDK proof. Removing target-tip equality alone is not a complete correction.
- [x] **B05.V** [run 20261005: DONE (bounded rerun 02:54Z on de470b99 clean: real GOEXIT=0, 254 PASS/40 native-SKIP w/ 40/40 reasons/0 FAIL, 3 pkgs ok, fixture named+stopped; evidence/B05-FINAL-rerun.log; original filtered receipt withdrawn per CODEX-B05-EVIDENCE-1; B04-log identity resolved)] Preserve result/actor/target/bookkeeping checks, fresh policy/digest and stale/non-progress refusal through the real producer/caller subjects. Do not invent a native wire field or local merge engine.

### B06 Dashboard authority and native reads

Lead B; [G01](reviews/G01.md), [G02](reviews/G02.md), [G03](reviews/G03.md), [G09](reviews/G09.md). Q5 exact external SDK/source correspondence.

- [ ] **B06.M** Retain actual auth/extension/background admission, transport/snapshot/issue/pull reads and preferences within existing dashboard/extension processes. L15 completed SDK cap+one input admission in Fountain `c92db11c14`; meaningful per-dial peer/admission transport stays. No Go HTTP/JSON engine replacement or cross-checkout mutation follows from planning.
- [ ] **B06.C** Correct G01-F1 declared/admin scope mismatch only after exact SDK propagation evidence is retained. Cache-refresh/association/redaction hypotheses remain investigation tasks, not presumed fixes.
- [ ] **B06.V** Verify existing real Go authority/transport/preference subjects and exact dependency correspondence; no adjacent-checkout substitution or new request authority.

- [x] **B06.C-admission** [2026-10-07: `a2a0926d`; actual Unix-socket blocked-waiter and shared-admission checks passed; independently reviewed] B owns CON-G02 cancellable bootstrap wait independent of SDK replacement. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **B06.M-adapters** Local SIMP-SNAPSHOT-1 is complete in `5794a7d9`, independently reviewed with actual snapshot/bracket checks; broader SIMP-SDK-1 waits on exact upstream capabilities, SIMP-GJSON-1 on actual DTO profiles. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### B07 Native Forgejo effect adapters

Lead B; [G04](reviews/G04.md), [G05](reviews/G05.md), [G06](reviews/G06.md), [G07](reviews/G07.md). B06 transport; coordinate B04/B05 rather than duplicating their registration fixes.

- [x] **B07.M** Source placement and the named concern splits are complete: publication moved to `internal/forgejo/publish` in `33682cfc`, with observation/validation/push/receipt and actual test descendants; merge observation/completion are split through `6cb54cf9`. Current callers, shared Background/StatusError/DTO owners and architecture assertions use the defining destination; `internal/host/publish` is absent. This source reconciliation does not close B07.C/V or Q5/native qualification.
- [ ] **B07.C** Complete the native adapter handoffs for canonical F09-F1/F10-F2/F12-F1 at their recorded owners after Q5. SHA length and evaluation-count unknowns do not authorize expanding accepted native input.
- [ ] **B07.V** Exercise real source/client subjects and independently supplied pinned native producer evidence; durable coordinator registration stays with B04, not these adapters.

- [ ] **B07.M-retirement** B decides CUST-G-SOURCE-1 method retention with current last callers; delete the dead method or bound its same-file read if retained, preserving sourceRepository. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### B08 Full Soda Forgejo presentation

Lead B; [G08](reviews/G08.md). Physical writer C; C09/H05/H06 payload/build/stage/locales joins. Can start early on reserved presentation files.

- [ ] **B08.M** Retain all deliberate page overrides and Soda design under `frontend/forgejo`; implement the reviewed Go-template context/form/CSRF, CSS cascade and TS enhancement splits. Rebind all four payload-manifest importers together through R01.
- [ ] **B08.C** Correct G08-F1 stale validator expectations against the current retained design. Unreproduced switcher timing remains a bounded investigation; no stock-page replacement or recoloring.
- [ ] **B08.V** Use existing real template/native-form/source/browser tests with honest upstream parity limits; preserve installed destination keys and distinct extension payloads.

## C — Platform, release and verification

### C01 Parser and filesystem primitives

Lead C; [H03](reviews/H03.md), [H04](reviews/H04.md). Starts independently; A/B implement callers in their owned files; Q6 gates policy/equivalence changes.

- [ ] **C01.M** [run 20261005: PARTIAL — historical json->lib/json move done; L04 later retires that engine; 36 config fixtures Q4-deferred spec-only] Preserve that landed move and fixture evidence. Pending Rust lexer/parser/binder/emitter splits are superseded by L04/JSON01 and explicit profile adapters. Retain Go strict binding and caller-specific file/config policy; L12 rooted custody precedes temporary convenience, and L17 holds only the effective-config decision.
- [ ] **C01.C** Partial at the broader parser/filesystem scope: preserve the completed H03-F1 panic correction, L04 parser replacement/caller-profile checks and L12 confinement policy. H04-F1's unused `root_chain`, its exclusive test matrix and stale sys reference are already retired in `913de806`; no retirement task remains for that helper. Other linked findings and caller-specific admissibility gates remain open; historical parser equivalence is not an adoption requirement.
- [x] **C01.V** [run 20261005: DONE, 5/5 json (truncated/duplicate/garbage subjects)] Exercise real malformed/truncated input and file/error subjects. Preserve distinct duplicate/null/case policies rather than treating all parsers as equivalent.

### C02 Private IPC and PostgreSQL persistence

Lead C; [H01](reviews/H01.md), [H02](reviews/H02.md). A owns Rust host/broker; B owns Go clients/Store/schema. Prioritize bounded runtime assumptions early.

- [ ] **C02.M** Custom framing/driver splits remain superseded. L07/L08/L09 source adoption is complete; retained modules describe library adapters and actual route/peer/credential policy, exclusive broker State/transaction custody, PostgreSQL and canonical Go DDL/Rust mirror. Broader structural/native acceptance is not inferred from those source checks. Physical A/B handoffs remain explicit.
- [x] **C02.C** [run 20261005: DONE (H01-Q2/Q9 disposition closed, H01-F1/F2/F3 routed to A, H02-F1 docs)] Close H01-Q2 framing disposition (Q9), then route bounded H01-F1 correction or evidenced decoder retirement in current Rust broker `runtime.rs` to A05/A; B owns its Go callers. Route H01-F2 pipe drain and H01-F3 attachment mutex correction to A. Correct H02-F1 stale SQLite guidance without selecting a different database or reusing old concurrency proof.
- [ ] **C02.V** L08/L09 development checks cover replacement framing/pump budgets, retained listener authority, upgrade read-ahead, one WS owner/close/reap, full transaction exclusion and PG operation cancellation/reconnect. L00 proofs preceded cutover. Broader installed/native qualification remains pending and distinct from completed C02.C.

### C03 Host and Project network observations

Lead C; [N02](reviews/N02.md), [N03](reviews/N03.md). A physical host writer; C02 HTTP/private IPC fit.

- [ ] **C03.M** Retain host Tailnet controls/domain/fixtures and Go status/DTO/client/browser authority. Pending LocalAPI framing, custom IP and calendar extraction is superseded by L09/N2 and L10/L11 library adapters; preserve unavailable/unconfirmed routing and provider execution policy.
- [x] **C03.C** N03-F1 LocalAPI transfer engine replaced through L09/N2 in `6b18ee1b`. Complete Hyper framing preserves bounded responses, endpoint status/deadline/no-replay contracts and unavailable outcomes; actual socket/client oracles pass. Provider curl recipes remain with L10/N4 separately; no installed-provider qualification is claimed.
- [ ] **C03.V** Exercise current peers and complete-driver response fixtures for chunked/length/EOF framing, truncation, caps and deadline/cancellation. L11 covers purpose-specific URL/IP/time admission; successful parsing does not prove route availability.

### C04 Project Tailnet and Forgejo helper port

Lead C; [N04](reviews/N04.md), [N05](reviews/N05.md), [N06](reviews/N06.md), [N07](reviews/N07.md). A physical native writer; C03 interface fit and C08/R01 binary staging; Q4 optional semantics.

- [ ] **C04.M** Preserve policy/provider/enrollment/companion incarnation and explicit-retry behavior. The Forgejo Tailnet Rust helper and release selector are implemented under L11; retain that source completion. R02/A34 keeps current project/run-binding/enroll-run policy in `tcontrol.rs`; optional caller/test seams must consume current L09/L11 adapters, with host lifecycle handoffs to A07.M/V. No stale checkpoint replay or further generic-engine split.
- [ ] **C04.V** Preserve root/timeout/native Endpoint/listener admission, private SSH_DOMAIN-only replacement, conditional restart, browser-origin independence and exact installed helper identity. The Rust helper source/build selection is complete; installed listener/restart/helper qualification remains separate. No further language port follows from R02.

### C05 Activation and setup

Lead C; [N01](reviews/N01.md), [O02](reviews/O02.md), [O03](reviews/O03.md). C01 parser/shared secret fit and C06 maintenance helpers; one setup writer.

- [ ] **C05.M** Retain ordered activation/setup/origin/secret publication and section-exact credential maintenance. Pending generic HTTP, JSON, URL/IP/time, formatting and CA-parser splits are superseded by L10/L04/L11/L14/L06; effective Forgejo configuration follows the L17 semantic-fit and deployment-override gate; its native corpus is collected. Keep actual wizard/service/file policy within existing commands.
- [ ] **C05.C** Correct established short-write/publication/cleanup findings through existing owners and L12. L18/TMP02 removed the unused migrate temporary in `eaed66a9`, preserving credential scrub/inode/mode behavior; retain single revocation and honest failure reporting.
- [ ] **C05.V** Exercise actual setup publication/error subjects with private inputs, complete writes and correct cleanup precedence; no new onboarding authority.

- [ ] **C05.M-profiles** C owns REP-CFG-1 command-local projections with B whole-config reader handoff; CFG01 INI fit is separate. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### C06 PostgreSQL initialization, backup and restore

Lead C; [O01](reviews/O01.md), [O05](reviews/O05.md), [O06](reviews/O06.md). Can start independently; serialize shared maintenance/setup files with C05.

- [ ] **C06.M** Retain one `cmd/soda-pg-maintenance` library and three existing bin/install selectors, established scopes/permissions and caller-owned quiescence.
- [x] **C06.C** O01-F1/O06-F1 source correction landed in `f9db9062`: shared `delivery_exit` refuses failed stdin delivery even with child exit zero, preserves nonzero child errors and is called after one reap by init-roles/restore. The retained regression and L12 package receipts cover their recorded subjects; C06.V and native database qualification remain separate.
- [ ] **C06.V** Exercise real child/input/caller failure subjects and existing backup/restore contracts; database operations require explicitly disposable or owner-selected scope.

- [x] **C06.C-rotation** [2026-10-07: `cd555cfc`; keep-one/keep-two future-stamped publication regressions passed; independently reviewed] New C-owned O05-F1 exact-publication backup retention correction; completed O01/O06 stdin-delivery repairs remain checked. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### C07 Operator recovery and SSH enrollment

Lead C; [O04](reviews/O04.md), [O07](reviews/O07.md). Existing installer/private fixture ownership also intersects C05/C10.

- [ ] **C07.M** Retain recovery/enrollment/window/session/cleanup and temporary SSH lifetime. Pending SSH/mpint/Base64/X509 grammar decomposition is superseded by L05/L06 and their profile gates. L12 owns file/temp mechanics; L17 holds only CFG01 candidate fit and effective-configuration cutover, not ready enrollment corrections.
- [x] **C07.C** Named enrollment source corrections landed: `8524cb2c` refuses incomplete writes before publication (O07-F1), and `bc8b818b` revalidates the selected address throughout serve/session lifetime (O07-F2). Current owners retain both repairs and unknown-marker refusal before lift/start mutation. This closes the named source corrections; C07.V and native authentication/window qualification remain open.
- [ ] **C07.V** Exercise actual publication/address/window/cleanup and marker subjects; root/Cockpit authority and native authentication proof remain distinct.

- [x] **C07.C-cleanup** [2026-10-07: `1ac697fd`; actual primary/cleanup error joins passed; independently reviewed] New C-owned ENROLL-CLEANUP-RESULT-1 primary/close error join; completed O07 write/address repairs remain checked. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### C08 Build controllers and candidate production

Lead C; [D01](reviews/D01.md), [D03](reviews/D03.md). Start early on disjoint release files; consumes explicit binary/source tuples from A/C09/C11 through R01.

- [x] **C08.M** [run 20261005: DONE, integrated 26327094 + R01 526bae78 (4 lib moves pure, 5 splits pure, D03-E4 retired)] Move the four existing Rust release crates to `lib/soda-release-{build,deliver,image,tools}`; split controllers/compiler/staging/progress/oracle concerns. Retire only the evidenced dead build progress/clock mirror and definition-only helpers, keeping actual release-tools progress/image Runner.
- [x] **C08.C** [run 20261005: LANDED (D01-F1..F5, D03-F1..F3, D01-F2/F3, mixed-cmd discovery proven); D01-F4 repair LANDED (86bce854, integrated 74a6a1b7) + SOURCE PASS per review-010; systemd/progress-output qualification separate] Correct recorded architecture namespace, clean-inspection/signal/progress and pipe-drain defects; explicitly fix D01-F2 obsolete Go controller build/candidate selectors and executable stamp, D03-F2 deleted Go artifact compile selector, and D03-F3 missing acceptance remote companion before inventory. Fix discovery for mixed Go/Rust `cmd` and explicit package/bin compilation in the actual cutover.
- [x] **C08.V** [run 20261005: DONE (mirror counts matched, Go pins green, candidate-check no stale pins)] Verify real producer/worker subjects and exact tools/rootfs selectors without a full release first. `soda-candidate-check` is already Rust: rebind its paths, do not schedule another port.

- [ ] **C08.C-evidence** OBS-R01 is complete in `d3fc68a7`, including bounded diagnostics, single media-log attachment and propagated replay failures, with seven actual checks. C retains OBS-W01 exact-unit/runtime cleanup; completed D03 drain/discovery work remains checked. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [x] **C08.C-artifact-cleanup** [2026-10-07: `26071cfc`; actual Butane failures/private output and child-reaping checks passed; independently reviewed] C owns CUST-C-BUTANE-CLEANUP-1 soda-artifacts output/child finalization, separate from installer enrollment. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### C09 Inputs, authenticated media, assets and attribution

Lead C; [D02](reviews/D02.md), [D04](reviews/D04.md), [H05](reviews/H05.md). C08 compile/stage join, B08 presentation; Q7 version/attribution gates.

- [x] **C09.M** [run 20261005: DONE, integrated c0a19d5a + R01 d296d340 (7 bins consolidated into tools/release-assets, ~70 system/license moves, package-selector rebinds; selinux move accepted per placement.md:151); rootfs-server placement completed in d89f4563] Consolidate seven existing fetch/render/locales binaries into `tools/release-assets`; rebind all actual package selections, manifest/fixture/loader inputs and notices. Move system definitions to `system/host`, `system/containers`, `system/project` and licenses to their selected owners. The Go rootfs server already lives at `tools/soda-rootfs-server` with its test/service references rebound.
- [ ] **C09.C** [run 20261005: PARTIAL — D02-F1 DONE (byte-safe predicate + regression, proven-red pre-fix at url.rs:60); installer-version media reconciliation NOT done, gate Q7 REMAINS OPEN per RULING-C-002 §3, no finalized finding] Correct the recorded byte-safe HTTPS predicate and reconcile actual installer-version media handoff only with retained producer evidence. Preserve full Forgejo design and installed payload destinations.
- [x] **C09.V** [run 20261005: DONE (375/0 mirror + canonical post-merge, clippy 0, 7 bins live, Go FAIL sets identical base-vs-post, cargo-building Go + TS suites green post-join; cockpit playwright NOT run)] Check real input/media/asset/attribution subjects and every actual binary/importer; no new network acquisition or signing contract.

- [ ] **C09.M-format** C owns SIMP-ASSET-NODE-1 provisioning Node/formatter cut for base.json; completed package/asset placement remains checked. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### C10 Verification, delivery and native payload application

Lead C; [D05](reviews/D05.md), [D07](reviews/D07.md), [D08](reviews/D08.md), [D09](reviews/D09.md), [D10](reviews/D10.md), [D11](reviews/D11.md). C01/C09 and exact Q6/Q7 equivalence/native gates.

- [ ] **C10.M** Retain release trust/admission/signing/publication/distribution and native installer/import policy. Generic JSON/Base64/DER/tar/gzip and duplicate OCI scanner splits are superseded by L02/L04/L13; L13 has retired the duplicate scanner after trailer/budget and fixture checks. Delivery owns shared layer scanning with an acyclic build dependency; the four crates and public/native lifetimes remain.
- [ ] **C10.C** L13's decoded gzip EOF/trailer and bounded member/aggregate correction is source complete in `d4f3c922`, with actual build/delivery fixtures passing. Preserve the separately recorded exported-finalize publication/history correction at its real API and Q7 gates. Unknown console EOF or native equivalence stays explicit; library defaults do not decide it.
- [ ] **C10.V** Exercise actual trust/null/duplicate/error/confinement/compression/result subjects and selected import/install lifecycle. Signing, publication and distribution effects are separately selected operations.

- [ ] **C10.M-profiles** C owns SIMP-PEM-1, CA-ALGORITHM-PROFILE-1 and SIMP-REL-WIRE-1/ORDERED-1/INSTALL-OCI-1 decisions; settle actual producer/reader contracts before dependent removal. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **C10.M-format** C owns REP-FMT-1/SIMP-REL-EMIT-1 controlled unsigned output cuts after actual consumers and current determinism are established. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [x] **C10.M-dependencies** [2026-10-07: `4e30cd9c` crypto features and `ad14c1c5` direct Runner cut, independently reviewed; 115 actual installer tests and real-Runner invalid-revision check pass] C supplies installer feature-only trim preserving current algorithms and COST-BOOTSTRAP-SEAM-1 two-call Runner cut; coordinator owns graph changes. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

### C11 Verification infrastructure and developer tooling

Lead C; [D06](reviews/D06.md), [H06](reviews/H06.md). Local driver/gate corrections can start early; C08/C10 candidate binding precedes native qualification.

- [ ] **C11.M** Retain actual Rust/Go/Bun drivers, fixture and source/stage/installed subjects. Pending custom JSON/time/hash, CLI/duration and process emulator extraction is superseded by L01/L03/L04/L11/L12/L14. Retain the current evidence owner without another matcher split while optional L16 adoption is deferred. Preserve active no-Python policy and live private-file custody; L18/DEAD01 removed the orphan OCI fixture in `eaed66a9`; active fixture/private-file custody remains retained.
- [ ] **C11.C** Preserve completed L01 child/QMP deadlines, writer/error custody and slashless URL confidentiality, plus L12 read/output/cleanup bounds. L14 CLI/duration adoption is source complete; L16 consideration is complete and optional adoption deferred. L16.G completes the separate aggregate secret-collection bound in `7334f36b`. Preserve recorded analyzer/SSH-pin/platform/hash-field corrections. Assess reachable SQLite-native fixture/probe consumers separately; do not schedule a Python port or globally remove their dependency.
- [ ] **C11.V** Verify the selected driver/evidence/parser/process failures and actual stage/installed bindings after each bounded replacement. No-newline output, split secrets, absolute deadlines and fail-closed finalization precede trusting these observations. Source/build/installed evidence remains separate; R04 alone owns a selected native journey.

- [ ] **C11.C-observers** H06-F1 and TEST-CARGO-RESULT-1/TEST-SOURCE-CLI-GATE-1 are complete in `04ddbff1`; OBS-D01/OBS-S01 in `e2214713`; JOIN-HOST-PROBE-PATH-1/JOIN-INVOKED-IDENTITY-1 in `c5f23dd8`, each independently reviewed with focused source/development checks. OBS-G01 remains pending and B-accountable through an exclusive installed.go/process.go handoff from C. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [x] **C11.C-custody** [2026-10-07: `6ee41013`; actual snapshot checks passed, including metadata-only sparse file; independently reviewed] C owns ACC-SNAPSHOT-FILE-1 opened-file snapshot admission; no atomic filesystem snapshot claim. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **C11.M-format** CLI03 is complete in `86812b0b` with independent review and the actual duration grammar check. C retains REP-ACC-EVIDENCE-ROUNDTRIP-1, preserving evidence custody and duration admission. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **C11.M-dependencies** B owns COST-GO-SQLITE-FIXTURE-1 seeder/test-support relocation; preserve acceptance SQLite and current test consumers. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).
- [ ] **C11.M-tests** C owns the narrow TEST-RETIRED-PYTHON-OUTPUT-1/TEST-REL-ORACLE-1/TEST-RETIREMENT-GUARDS-1 consumer/oracle retirement; native-emulator evidence labels and current behavior tests remain. Scope, prerequisites and acceptance: [finding allocation](execution-findings.md).

## Selected library-adoption subpackets

These are the active-priority planning entries. Scope, callers, retained policy,
prerequisites and acceptance live in [the chapter](library-adoption.md#execution-packets);
the lane schedule supplies one physical writer and integration order. Checked entries
are complete only at their stated scope. Each library conversion carries its
affected build/offline qualification.

- [x] **L00** Coordinator: initial compiler/dependency/license/local cache/archive inventory, nine boundary/API probes and affected L02 package/caller offline checks complete. tokio-postgres deadline facade selected; production cutover/native/profile gates remain with their named packets. At that checkpoint L02 had not started; it is now complete. [Defined boundary](library-adoption.md#l00-admission-and-boundary-preparation).
- [x] **L01** C: repair acceptance deadlines and evidence completion/confidentiality. Phase fix `9fda53fe` preserved; CoreOS finalization `3873614f`, evidence/pump bounds and failure propagation `d80aec93`, QMP/VM ownership `4c87f5e9`. All 126 acceptance library tests and three binary compile checks pass. Local shell/socket evidence only; N13 library parser adoption remains in L11 and L16 matching stays separate. [Defined boundary](library-adoption.md#l01-deadline-and-evidence-repair).
- [x] **L02** C: complete in `743dde17`; shared strict on-curve P-256 SPKI trust adapter plus strict ECDSA DER on all four installer curves, retaining original DER/TBS and role boundaries. Trust foundation 24; image and delivery models 5 each; installer X.509 26 passed. No installed/native-worker qualification. [Defined boundary](library-adoption.md#l02-trust-key-and-signature-repair).
- [x] **L03** A: source scope complete in `a84447ff` (fail-closed entropy across ten packages) and `52eee7ee` (all seven SHA definitions and host NIST engines replaced). Fingerprints, input recipes, point admission and failure propagation remain with callers. Focused SHA tests, 319 host tests and affected offline builds passed; no installed/native-worker qualification. [Defined boundary](library-adoption.md#l03-hash-curve-and-randomness-owners).
- [x] **L04** C: source scope complete; CF-04 in `26493cf2`, caller-specific Serde transfers and engine retirement through `229e9cce`. Raw signed bytes, duplicate/alias/null/integer rules and required producer representations remain with callers. Final guest 179 unit + 33 integration checks and Acceptance 131 passed; affected offline graph builds without warnings, independent medium reviews closed and source/dependency census is zero. Dynamic trees use explicit 127-container admission and borrowed child capture; no installed/native-worker qualification. [Defined boundary](library-adoption.md#l04-json-and-base64-profiles).
- [x] **L05** A, with C installer handoff: source scope complete in `7b42671d`, using Luna medium. Typed upstream SSH formats retain raw/certificate allowlists, scalar/point/line policy, canonical raw fingerprints and exact signed certificate bytes. `22496cb3` selects published 0.7.0-rc.11 because 0.6.7 rejects the actual OpenSSH forever certificate. Final host 697 and installer 114 tests, independent review and locked offline native development build pass; installed/native-worker qualification remains separate. [Defined boundary](library-adoption.md#l05-ssh-formats).
- [x] **L06** C: source scope complete in `7d063f15`, using Luna medium, independently of L05. The actual pinned Caddy root closes the profile gate; x509-cert/der replace custom certificate/SPKI/extension/calendar parsing. One bounded public PEM, strict CA/critical/algorithm admission, original DER fingerprints and original-TBS verification remain local. All 114 installer tests, independent review and locked offline native development build pass; eight retained unrelated export warnings and installed/native-worker qualification remain separate. [Defined boundary](library-adoption.md#l06-local-ca-parsing).
- [x] **L07** B, with explicit A Rust handoff: complete in `d12bf6d3`. Native `$n` parameters cover Go and Rust product queries; both translators and Go's rebind-only wrappers are removed, while Rust's encoding/query helpers remain. Fresh PG17.11 fixture: 68 Go store tests, eight Rust broker/enrollment tests and the bytea unit test pass; architecture checks and affected builds pass. No SQLite seed, dependency, manifest, lock or schema change. The latent Rust transaction-lock gate remains with L08. [Defined boundary and limits](library-adoption.md#l07-native-sql-parameters).
- [x] **L08** A: source scope complete in `6b18ee1b`. tokio-postgres replaces wire/auth/query/DSN engines; synchronous Store/Tx retains schema/CAS/sealing/events and sanitized errors. One 30s operation deadline covers lock/connect/query/cancel/discard/join; one guard spans the whole transaction and swallowed errors cannot report commit. Fresh PG17.11 SCRAM Unix/TCP adapter checks4/4, actual broker7/enrollment1, affected client/HTTP checks and locked offline builds pass. Production-generated Unix DSN is checked; installed/customized HBA and native-worker qualification remain unverified. [Defined boundary](library-adoption.md#l08-postgresql-driver).
- [x] **L09** A, with C fixture handoff: source scope complete through `21387814`, using Luna medium. Four clients precede identity/host servers; Hyper owns framing, and tungstenite owns the coupled upgrade/pump with read-ahead, bounded output and joined session/child cleanup. Actual blocked-backend/admission/shutdown, short-write/Pong, 5s socket-write and 2s child-input expiry checks pass, alongside full affected caller suites and warning-free locked offline development builds. Installed/native-worker qualification remains unverified; broader A07/C02 acceptance remains pending. [Defined boundary and limits](library-adoption.md#l09-unix-http-and-websocket-engines).
- [ ] **L10** C: setup HTTPS transfer complete; host provider replacement held at its deadline/Executor boundary. [Defined boundary](library-adoption.md#l10-external-http-adapters).
- [x] **L10.N3** C: `719d1137` replaces setup framing with ureq; 21 local facade/credential/framing/cap/deadline tests pass. Blocking OS DNS remains unbounded as before; installed trust qualification is separate.
- [ ] **L10.N4** C: prove owned resolver-inclusive cancellation and native Executor custody before A receives host provider transfer. Preserve Basic/Bearer inputs, bounded bodies, native marker/test seam and one-shot unconfirmed key creation; acceptance and prerequisite are in the L10 boundary above.
- [x] **L11** A: source adoption complete through `4d0f1128`, `2d9066bd` and `380914ed`, using Luna low after focused Luna medium URL/deadline admission. Typed URL/percent, std IP and shared strict time adapters preserve raw literals, caller admission, bounded evidence and local/shared clock policy. 1,572 scoped tests and locked offline native development builds for 16 selected packages pass. Four unchanged opt-in PG tests and one existing doctest remain ignored; an unrelated release-assets staging integration lacks the prebuilt Muse artifact. Installed/native-worker qualification is separate. [Defined boundary](library-adoption.md#l11-url-ip-and-time-adapters).
- [x] **L12** C, with A host/guest/identity handoffs: source scope complete through `c5cca5e7`. Completed L01/L00 and relevant L03 profiles preceded Luna medium custody/cancellation and review; Luna low handled settled plumbing. Same-FD cap+1 and rooted admission precede tempfile adoption. Typed owned descriptors/CLOEXEC, live cancellation, bounded fair capture, required EOF, feeder/ticker joins and checked native cleanup preserve caller authority. 1,687 selected tests and locked offline native development builds for 20 packages pass. Installed systemd/container/VM and shipping/kernel qualification remain separate; N11/TMP02 were subsequently retired under L18; N4 remains held. [Scope, retained policy and acceptance evidence](library-adoption.md#l12-file-fd-and-process-ownership).
- [x] **L13** C: source scope complete through `46d5c4cd`, using Luna medium for EOF/budget/ownership and low for settled format transfers. Deterministic tar writers, delivery-owned shared bounded OCI scanning, complete gzip, bounded terminal extraction, shared purpose-specific ELF accessor and roxmltree/svgtypes parsing replace the selected engines. Actual pinned terminal outputs, canonical emblem, corrupted/truncated trailers, budgets, overlays and original signed fixtures pass; installed/shipping qualification remains separate. [Defined boundary and evidence](library-adoption.md#l13-archive-and-release-formats).
- [x] **L14** C: source scope complete through `c82125ee`, using Luna low for settled profiles and medium for metadata custody/independent CLI review. Clap/humantime/Rust formatting replace goflag/duration/fmtx engines; one bounded Cargo metadata inventory replaces TOML scanning and flows through compile/staging. Actual help/refusal exits, false/repeats/literal tails, default-feature admission and unchanged shipping selectors pass. L13/L14 together have 656 selected passing tests and ten affected locked offline development builds; native appliance/worker qualification stays separate. [Defined boundary and evidence](library-adoption.md#l14-cli-and-target-discovery).
- [x] **L15** B: SDK cap+one input admission complete in Fountain `c92db11c14`; exact-source, regression and retained credential-transport checks pass. [Defined boundary](library-adoption.md#l15-sdk-input-admission).
- [x] **L16** C: Luna medium consideration complete; retain the matcher and defer optional Aho-Corasick adoption. Reconsider only after L16.G, demonstrated value, exact dependency admission and construction/streaming proofs. No source cutover is claimed. [Decision, scope and acceptance](library-adoption.md#l16-evidence-matching).
- [x] **L16.G** C: completed in `7334f36b`, independently reviewed at Luna medium. One collector admits at most 16,384 nonempty candidates and 16 MiB before retaining raw/trimmed/Ignition variants; bounded per-file staging remains separate. Actual repeated-file, exact-bound, derived-line, near-limit key and trust tests pass. Collection refuses before evidence creation/capture; escaped-pattern admission remains separate. Optional matcher adoption stays deferred. [Profile and evidence](library-adoption.md#l16-evidence-matching).
- [ ] **L17** C: 21-case native corpus and exact candidate-source decision complete; rust-ini 0.21.3 fails continuation admission. CFG01 cutover stays held on semantic fit and deployment override/dependency admission; preserve no-guess and marker policy. CFG02 locale scanner is retained independently. [Defined boundary](library-adoption.md#l17-configuration-evidence).
- [x] **L18** C, with A/B source handoffs: N11/TMP02/DEAD01 retired in `eaed66a9` after current reference closure and retained-duty review. Forty existing tests, eight real-command byte/inode/mode fixtures and three locked offline development entrypoints pass. R02 parked-seam assessment is complete at this scope; full coverage/count regeneration and installed qualification remain separate. [Defined boundary](library-adoption.md#l18-dead-machinery-removal).

## Gates and readiness

The complete per-record open-decision/correction tables remain canonical. This
table routes the known consequential holds; it is not permission to ignore a
new material unknown found in a selected packet. A settled language choice is
different from a missing correction specification or native equivalence proof.

| Gate | Exact remaining fact/decision | Dependent task only |
| --- | --- | --- |
| Q1 | P05-F1 cause-specific withdrawal, grant and quiescence representation must be finalized without clearing other holds. | A02.C and its B03 lifecycle handoff |
| Q2 | P09-F1 actual native Project-admin/operator authority mapping; owner status alone is insufficient. | A02.C approval correction |
| Q3 | I05-Q4 terminal-Muse InvocationID applicability; I09/F07-Q4 harness/family selection and F10 loop allowance/harness questions. | A05/A07/B03/B04 dependent enforcement/loop actions; no provider default invented |
| Q4 | Optional P01/N05 creation intent retention, N04/N06 runtime/selection/retry applicability, I10 public history/retention scope. | Only the corresponding optional behavior, not relocation or established native policy |
| Q5 | Exact external Fountain/SDK scope propagation, producer bytes, review/merge/reachability/read correspondence and native qualification. Retain the exact approved dependency identity; no alternate Soda read path. Local F09/F10 operation registration is governed by Q8, not blanket-held by missing native qualification. | G01-F1 in B06.C; B02/B04/B05/B07 native-dependent actions only |
| Q6 | Caller-specific parser admissibility, link/confinement/case/null/error/compression policies and pure-helper equivalence; console diagnostic EOF remains explicit. | C01/C10 dependent correction/copy retirement only |
| Q7 | Actual authenticated media installer-version correspondence and distribution attribution requirements. | C09/C10 dependent media/distribution action only |
| Q8 | Exact outstanding effect/receipt, accounting, registration/recovery, fresh-attempt and finite-loop correction specifications in F01/F03/F07–F10. Source defect identification does not fill in transaction/allowance design. | Corresponding B01/B03/B04 C steps; settled mechanical allocations can proceed |
| Q9 | Custom chunk-decoder retention/extraction is superseded by L09 complete framing. Actual endpoint bounds, status, deadline and no-replay policy remain acceptance requirements; completed C02.C evidence is historical. | L09 relevant caller/server cutover with LA-G3; no hand-maintained framing engine |

Library cutover gates LA-G1–LA-G6 are defined once in
[library adoption](library-adoption.md#readiness-gates). They hold only their
named boundary; old domain Q gates are not global adoption barriers.

## Coverage and completion

| Catalog group | Primary packets | Slice count |
| --- | --- | ---: |
| Projects P01–P12 | A01–A04 | 12 |
| Identity I01–I10 | A05–A07 | 10 |
| Spaces S01–S06 | A07–A08 | 6 |
| Factory F01–F12 | B01–B05 | 12 |
| Forgejo G01–G09 | B06–B08 | 9 |
| Networking N01–N07 | C03–C05 | 7 |
| Operator O01–O07 | C05–C07 | 7 |
| Release D01–D11 | C08–C11 | 11 |
| Shared H01–H06 | C01–C02, C09, C11 | 6 |
| **Total** | **27 primary packets + A00 support; R00–R04 integration** | **80** |

A packet closes when all assigned current duties have retained/implemented/retired
dispositions, exact caller/fixture/build/install joins and appropriate performed
verification, with consequential independent challenge. Keep M/C/V and native
readiness separate. Owner-blocked C steps remain visibly open; do not mark their
whole workflow correct because a move or unit test passed. Update this task list
and the living plan after each merge through the maintenance workflow.

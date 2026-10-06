# Implementation task list

Execute through the [parallel schedule and file ownership rules](implementation-lanes.md).
All checkboxes are **deferred**, not implementation authorization. M is a
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

### A02 Preparation authority and Project lifecycle

Lead A; [P05](reviews/P05.md), [P08](reviews/P08.md), [P09](reviews/P09.md), [P12](reviews/P12.md). B01/B03 grant/Store handoffs; A00 native extraction.

- [ ] **A02.M** Keep Go preparation decisions/Store CAS/API and Rust `lib/host/src/{preparation,prepare,project}` lifecycle definitions with their current processes and distinct heads/markers.
- [ ] **A02.C** Specify and correct P05-F1 cause-specific Project-stop reopening and P09-F1 actual approval authority only after Q1/Q2 resolve. Preserve independent maintenance holds, other withdrawals and marker-first/CAS failure reporting.
- [ ] **A02.V** Exercise existing real start/stop/preparation/approval subjects, separate head revisions and failure paths; coordinate actual dispatch reopening with B03.

### A03 Guest role custody and checkout supervision

Lead A; [P06](reviews/P06.md), [P07](reviews/P07.md). Shared guest-package ownership with A07; no simultaneous module-root edits.

- [ ] **A03.M** Fold current `rust/soda-project-factory-roles` into `cmd/soda-project-terminal/src/factory_roles/` and its existing `project-factory-roles` binary; retain the compiled-helper oracle and install/hash selectors.
- [ ] **A03.C** Correct P07-F1/F2 exact child retirement on capture failure and preservation of preexisting checkout state at the recorded real supervision/caller owners.
- [ ] **A03.V** Use actual compiled role-helper subjects for capture/cleanup and checkout failure, with exact requested binary selection; no Python predecessor recreation.

### A04 Shared tools, nested services and volumes

Lead A; [P10](reviews/P10.md), [P11](reviews/P11.md). C owns system/Containerfile/compiler joins and the Muse-maintain command/stage implementation; A owns host tool-observation counterparts.

- [ ] **A04.M** Retain mise/profile/systemd/OCI/workload substrates; move Project definitions to `system/project` and reviewed native tool observation/`cmd/soda-muse-maintain/src/stage.rs` concerns without new services.
- [ ] **A04.V** Preserve actual script subjects, read-only readiness, ordered bus/tool staging and volume lifetime. Rebind existing source assertions with their true implementation owners.

### A05 Broker custody and execution state

Lead A; [I01](reviews/I01.md), [I02](reviews/I02.md), [I03](reviews/I03.md), [I04](reviews/I04.md), [I05](reviews/I05.md), [I06](reviews/I06.md), [I10](reviews/I10.md). C01 bounded parser work; B01/H02 schema/admin-reader handoffs.

- [x] **A05.M** [run 20261005: DONE, integrated 8c489d16 + R01 91c6d3e7] Consolidate broker/provider package ownership under `cmd/soda-identity`; keep one Controller/State/Store/Tx and reviewed enrollment/grant/acquisition/registration/retirement/store/wire descendants. Retain Go domain/client and canonical PostgreSQL schema with its existing mirror assertion.
- [x] **A05.C** [run 20261005: DONE, lane V closed on live disposable PG; overall acceptance CLOSED via CORR-C-001 (merged + real-tree proven)] Correct I06-F1 terminal execution fencing after successful admitted End/expiry/return using actual provider-specific retirement; preserve legitimate recovery before any returned binding. Do not add I05 terminal-Muse InvocationID enforcement or public history/retention from unresolved requirements (Q3/Q4).
- [x] **A05.V** [run 20261005: DONE, 25 unit + 10 integration executed, 0 skips] Exercise actual same-ID terminal fencing, current grants, encrypted custody and atomic append subjects; coordinate sponsor metadata and fixture rebinding with B01 before shadow-write retirement.

### A06 Provider adapters

Lead A; [I07](reviews/I07.md), [I08](reviews/I08.md). A05 provider-support/construction ownership must be fixed before independent child-module work.

- [x] **A06.M** [run 20261005: DONE, integrated db2b92d7 + R01 eb6b14d6, predecessor retired] Fold the actual Codex/Muse implementations into private `cmd/soda-identity/src/providers/{codex,muse}` descendants; preserve one shared support/type owner and direct imports.
- [x] **A06.V** [run 20261005: DONE, 41 unit + 10 integration on live PG, 0 skips] Preserve pinned constructors, sanitized protocol, stop-before-read/cleanup, serialized Codex execution and concurrent immutable Muse lease behavior. Existing adapter assertions are source tests; real provider calls remain separate qualification.

### A07 Native launch and interactive attachment

Lead A; [I09](reviews/I09.md), [S04](reviews/S04.md), [S05](reviews/S05.md). A00/A03 guest handoffs, C02 host transport and B03/B04 native run/export handoffs.
Compose/Muse-maintain command changes route to C; reserve their current module
roots once across their P/I/H duties, while A owns host/guest changes.

- [ ] **A07.M** Define shared run/native/binding/lifecycle/output/artifact units directly under `lib/host/src/terminal/factory/`; preserve distinct Codex/Muse launch/custody owners and the existing Service. Split actual guest terminal/PTY, attachment/pump and Compose/maintenance caller duties without new processes.
- [x] **A07.C** [run 20261005: LANDED 865f0e2f (H01-F3) + 9d5a9281 (S04-F1/S05-F1); H01-F3 reap repair LANDED (6edf58b7+31bf88cf, integrated d4597974) + kill-then-wait source-verified; native transport pumping unproven (helpers lack output reader), provider/native qualification separate] Correct S04-F1 C-string ABI and S05-F1 select writable-set handling; implement canonical H01-F3 pump mutex correction through C02's A-owned host handoff. Q3 gates harness/invocation-dependent enforcement only.
- [ ] **A07.V** Exercise real PTY/relay/pump subjects for concurrent input/output/close, reservation consumption, exact binding and honest retirement before expensive native runs.

### A08 Spaces browser inventory and viewer

Lead A; [S01](reviews/S01.md), [S02](reviews/S02.md), [S03](reviews/S03.md), [S06](reviews/S06.md). B owns physical Go/browser files; A00 releases workspace descendants.

- [ ] **A08.M** Retain bounded Go inventory/output admission and explicit browser decoder/workspace/layout/view/CSS owners with one actor/epoch/slot definition. Retire the proven-unused widths predecessor after its actual assertions are rebound.
- [ ] **A08.C** Correct S01-F1 omitted per-row `TailnetState` observation: the actual `spaceItem` decoder reads collection data instead of the row value. Restore bounded row decoding/display without inventing admission or incomplete-inventory policy.
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
- [ ] **B03.C** Correct F07-F1 authority tuple freshness, F07-F2 recorded prompt context, F08-F1 registration/withdraw capture ordering, F08-F2 fresh retry attempt and F08-F3 stop-owned settlement. Resolve exact grant/quiescence/transaction specifications first; Q3 blocks undecided simultaneous provider-family selection.
- [x] **B03.V** [run 20261005: DONE on disposable PG (11 pkgs, -race 4/4, raw logs archived, 0F/0S)] Exercise actual authority/registration/withdraw/retry/stop races with existing real Store and coordinator subjects; preserve historical attribution and usage.

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

- [ ] **B06.M** Split actual auth/extension/background admission, transport/snapshot/issue/pull reads and preferences within existing dashboard/extension processes.
- [ ] **B06.C** Correct G01-F1 declared/admin scope mismatch only after exact SDK propagation evidence is retained. Cache-refresh/association/redaction hypotheses remain investigation tasks, not presumed fixes.
- [ ] **B06.V** Verify existing real Go authority/transport/preference subjects and exact dependency correspondence; no adjacent-checkout substitution or new request authority.

### B07 Native Forgejo effect adapters

Lead B; [G04](reviews/G04.md), [G05](reviews/G05.md), [G06](reviews/G06.md), [G07](reviews/G07.md). B06 transport; coordinate B04/B05 rather than duplicating their registration fixes.

- [ ] **B07.M** Move `internal/host/publish` to `internal/forgejo/publish`; split actual publication/review/check/merge observer/intent/receipt concerns, preserving one Background/StatusError/DTO owner and architecture assertions.
- [ ] **B07.C** Complete the native adapter handoffs for canonical F09-F1/F10-F2/F12-F1 at their recorded owners after Q5. SHA length and evaluation-count unknowns do not authorize expanding accepted native input.
- [ ] **B07.V** Exercise real source/client subjects and independently supplied pinned native producer evidence; durable coordinator registration stays with B04, not these adapters.

### B08 Full Soda Forgejo presentation

Lead B; [G08](reviews/G08.md). Physical writer C; C09/H05/H06 payload/build/stage/locales joins. Can start early on reserved presentation files.

- [ ] **B08.M** Retain all deliberate page overrides and Soda design under `frontend/forgejo`; implement the reviewed Go-template context/form/CSRF, CSS cascade and TS enhancement splits. Rebind all four payload-manifest importers together through R01.
- [ ] **B08.C** Correct G08-F1 stale validator expectations against the current retained design. Unreproduced switcher timing remains a bounded investigation; no stock-page replacement or recoloring.
- [ ] **B08.V** Use existing real template/native-form/source/browser tests with honest upstream parity limits; preserve installed destination keys and distinct extension payloads.

## C — Platform, release and verification

### C01 Parser and filesystem primitives

Lead C; [H03](reviews/H03.md), [H04](reviews/H04.md). Starts independently; A/B implement callers in their owned files; Q6 gates policy/equivalence changes.

- [ ] **C01.M** [run 20261005: PARTIAL — json->lib/json done; 36 config fixtures Q4-deferred spec-only] Move shared Rust JSON to `lib/json`, retain Go strict binding and caller-specific filesystem/configuration owners; split actual private tests and all 36 config fixtures without new helper packages.
- [ ] **C01.C** [run 20261005: PARTIAL — H03-F1 done; other parser findings/root_chain open] Correct H03-F1 bounded surrogate panic and other established parser/error findings at their real callers. Correct confinement only after the record's allowed-link/admissibility gate; retire only proven-unused `root_chain` units.
- [x] **C01.V** [run 20261005: DONE, 5/5 json (truncated/duplicate/garbage subjects)] Exercise real malformed/truncated input and file/error subjects. Preserve distinct duplicate/null/case policies rather than treating all parsers as equivalent.

### C02 Private IPC and PostgreSQL persistence

Lead C; [H01](reviews/H01.md), [H02](reviews/H02.md). A owns Rust host/broker; B owns Go clients/Store/schema. Prioritize bounded runtime assumptions early.

- [ ] **C02.M** Rebind actual daemon/client/wire and transport fixture subjects; retain exclusive broker State/Store transaction custody, PostgreSQL, canonical Go DDL and existing Rust mirror.
- [x] **C02.C** [run 20261005: DONE (H01-Q2/Q9 disposition closed, H01-F1/F2/F3 routed to A, H02-F1 docs)] Close H01-Q2 framing disposition (Q9), then route bounded H01-F1 correction or evidenced decoder retirement in current Rust broker `runtime.rs` to A05/A; B owns its Go callers. Route H01-F2 pipe drain and H01-F3 attachment mutex correction to A. Correct H02-F1 stale SQLite guidance without selecting a different database or reusing old concurrency proof.
- [ ] **C02.V** Exercise the real frame/pipe/pump and existing disposable PostgreSQL subjects. Readiness of host output/input/expiry precedes expensive native qualification; no new service or store.

### C03 Host and Project network observations

Lead C; [N02](reviews/N02.md), [N03](reviews/N03.md). A physical host writer; C02 HTTP/private IPC fit.

- [ ] **C03.M** Move actual controls/domain/fixtures to `lib/host/src/tailnet`, retain Go status/DTO/client and browser authority, and preserve bounded connection observation including unverified routing.
- [ ] **C03.C** Correct N03-F1 LocalAPI transfer framing at the existing native client, preserving current response bounds/deadlines/unavailable results.
- [ ] **C03.V** Test actual chunked/length-framed producers and native client subjects; successful parsing does not prove route availability.

### C04 Project Tailnet and Forgejo helper port

Lead C; [N04](reviews/N04.md), [N05](reviews/N05.md), [N06](reviews/N06.md), [N07](reviews/N07.md). A physical native writer; C03 interface fit and C08/R01 binary staging; Q4 optional semantics.

- [ ] **C04.M** Preserve policy/provider/enrollment/companion incarnation and explicit-retry behavior; retire Go execution predecessors only after real Rust caller/assertion closure. Port `soda-forgejo-tailnet` and its exclusive Go rewrite into `cmd/soda-forgejo-tailnet/main.rs` plus `lib/host/src/tailnet/forgejo.rs`, an additional binary of the existing host package.
- [ ] **C04.V** Preserve root/timeout/native Endpoint/listener admission, private SSH_DOMAIN-only replacement, conditional restart, browser-origin independence and exact installed helper identity. This is the **one remaining Go→Rust implementation port**; no Rust→Go port is selected.

### C05 Activation and setup

Lead C; [N01](reviews/N01.md), [O02](reviews/O02.md), [O03](reviews/O03.md). C01 parser/shared secret fit and C06 maintenance helpers; one setup writer.

- [ ] **C05.M** Split reviewed activation/setup/console/config/trust concerns under `cmd/` and `system/host`, retaining ordered origin/config/secret publication and section-exact credential maintenance.
- [ ] **C05.C** Correct established successful-short-write findings and misleading unused-temp guidance at their existing owners; retain single revocation and current failure reporting.
- [ ] **C05.V** Exercise actual setup publication/error subjects with private inputs, complete writes and correct cleanup precedence; no new onboarding authority.

### C06 PostgreSQL initialization, backup and restore

Lead C; [O01](reviews/O01.md), [O05](reviews/O05.md), [O06](reviews/O06.md). Can start independently; serialize shared maintenance/setup files with C05.

- [ ] **C06.M** Retain one `cmd/soda-pg-maintenance` library and three existing bin/install selectors, established scopes/permissions and caller-owned quiescence.
- [ ] **C06.C** Correct O01-F1/O06-F1 failed stdin delivery combined with child exit zero, preserving nonzero child errors and one reap. No retry or whole-state backup subsystem.
- [ ] **C06.V** Exercise real child/input/caller failure subjects and existing backup/restore contracts; database operations require explicitly disposable or owner-selected scope.

### C07 Operator recovery and SSH enrollment

Lead C; [O04](reviews/O04.md), [O07](reviews/O07.md). Existing installer/private fixture ownership also intersects C05/C10.

- [ ] **C07.M** Split actual recovery/enrollment/window/session/key/cleanup concerns within existing commands and temporary SSH lifetime.
- [ ] **C07.C** Preserve unknown-marker failure before lift/start mutation; correct O07-F1 complete enrollment writes and O07-F2 ongoing selected-address binding at their existing owners.
- [ ] **C07.V** Exercise actual publication/address/window/cleanup and marker subjects; root/Cockpit authority and native authentication proof remain distinct.

### C08 Build controllers and candidate production

Lead C; [D01](reviews/D01.md), [D03](reviews/D03.md). Start early on disjoint release files; consumes explicit binary/source tuples from A/C09/C11 through R01.

- [x] **C08.M** [run 20261005: DONE, integrated 26327094 + R01 526bae78 (4 lib moves pure, 5 splits pure, D03-E4 retired)] Move the four existing Rust release crates to `lib/soda-release-{build,deliver,image,tools}`; split controllers/compiler/staging/progress/oracle concerns. Retire only the evidenced dead build progress/clock mirror and definition-only helpers, keeping actual release-tools progress/image Runner.
- [x] **C08.C** [run 20261005: LANDED (D01-F1..F5, D03-F1..F3, D01-F2/F3, mixed-cmd discovery proven); D01-F4 repair LANDED (86bce854, integrated 74a6a1b7) + SOURCE PASS per review-010; systemd/progress-output qualification separate] Correct recorded architecture namespace, clean-inspection/signal/progress and pipe-drain defects; explicitly fix D01-F2 obsolete Go controller build/candidate selectors and executable stamp, D03-F2 deleted Go artifact compile selector, and D03-F3 missing acceptance remote companion before inventory. Fix discovery for mixed Go/Rust `cmd` and explicit package/bin compilation in the actual cutover.
- [x] **C08.V** [run 20261005: DONE (mirror counts matched, Go pins green, candidate-check no stale pins)] Verify real producer/worker subjects and exact tools/rootfs selectors without a full release first. `soda-candidate-check` is already Rust: rebind its paths, do not schedule another port.

### C09 Inputs, authenticated media, assets and attribution

Lead C; [D02](reviews/D02.md), [D04](reviews/D04.md), [H05](reviews/H05.md). C08 compile/stage join, B08 presentation; Q7 version/attribution gates.

- [x] **C09.M** [run 20261005: DONE, integrated c0a19d5a + R01 d296d340 (7 bins consolidated into tools/release-assets, ~70 system/license moves, package-selector rebinds; selinux move accepted per placement.md:151; cmd/soda-rootfs-server Go-app move queued to B per placement.md:150)] Consolidate seven existing fetch/render/locales binaries into `tools/release-assets`; rebind all actual package selections, manifest/fixture/loader inputs and notices. Move system definitions to `system/host`, `system/containers`, `system/project` and licenses to their selected owners.
- [ ] **C09.C** [run 20261005: PARTIAL — D02-F1 DONE (byte-safe predicate + regression, proven-red pre-fix at url.rs:60); installer-version media reconciliation NOT done, gate Q7 REMAINS OPEN per RULING-C-002 §3, no finalized finding] Correct the recorded byte-safe HTTPS predicate and reconcile actual installer-version media handoff only with retained producer evidence. Preserve full Forgejo design and installed payload destinations.
- [x] **C09.V** [run 20261005: DONE (375/0 mirror + canonical post-merge, clippy 0, 7 bins live, Go FAIL sets identical base-vs-post, cargo-building Go + TS suites green post-join; cockpit playwright NOT run)] Check real input/media/asset/attribution subjects and every actual binary/importer; no new network acquisition or signing contract.

### C10 Verification, delivery and native payload application

Lead C; [D05](reviews/D05.md), [D07](reviews/D07.md), [D08](reviews/D08.md), [D09](reviews/D09.md), [D10](reviews/D10.md), [D11](reviews/D11.md). C01/C09 and exact Q6/Q7 equivalence/native gates.

- [ ] **C10.M** Split selected release verification/admission/signing/publication/distribution and native installer/import owners. Reuse recorded pure release-deliver payload/content and release-build OCI owners only after caller-specific equivalence; preserve quartet and public/native lifetimes.
- [ ] **C10.C** Correct recorded decoded gzip EOF validation and exported-finalize publication/history composition at the real current APIs. Do not resolve unknown console EOF policy or delete non-equivalent mirrors by assumption.
- [ ] **C10.V** Exercise actual trust/null/duplicate/error/confinement/compression/result subjects and selected import/install lifecycle. Signing, publication and distribution effects are separately selected operations.

### C11 Verification infrastructure and developer tooling

Lead C; [D06](reviews/D06.md), [H06](reviews/H06.md). Local driver/gate corrections can start early; C08/C10 candidate binding precedes native qualification.

- [ ] **C11.M** Rebind actual Rust/Go/Bun drivers, fixtures, source gates, hooks and remaining tools with their real subjects. Retain active no-Python policy/gate and live `PrivateFile`; retire only evidenced obsolete assertions/tooling/generated tracked artifacts via R02.
- [ ] **C11.C** Correct recorded timeout/platform/redaction/hash-field/transient-PID/SSH-pin and retired-page assumptions plus analyzer failure classification. Reconcile the current Go lifecycle observer's stale SQLite assumptions under PostgreSQL rather than treating it as a pending Python port.
- [ ] **C11.V** Validate actual driver/analyzer failure paths and stage/installed bindings. A source-only proof never becomes installed qualification; one owner runs any selected matching-native candidate journey through R04.

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
| Q9 | H01-Q2 retain-correct chunk support versus retirement of unused decoding; current producer framing and retained transport contract must decide the bounded disposition. | C02.C H01-F1 only; no broad HTTP compatibility project |

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

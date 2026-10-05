# Slice validity review records

The user started source review on 2026-10-05: connected identity brokering,
factory execution and persistence/private IPC first, then all 80 slices. Use
the [source/guidance baseline](../review-baseline.md), [single shared format](../review-format.md)
and [primary/challenger assignments](../review-assignments.md). Implementation
remains deferred. A record is created only when its actual review begins.

The primary record is the authority for its inspected scope, model, findings,
workflow analysis, challenge responses and target allocation. This index links
records without treating creation as completion. An open finding can be inspected
and challenged while its correction remains pending. Whole-audit completion
requires the [four completion dimensions](../review-format.md#completion-criteria)
across the full current catalog. Uninspected duties and unresolved allocations
remain explicit; source review is not installed behavioral proof.

## Completed source-audit coverage

Completed **2026-10-05** against source
`f7e9cf9db616f93de351dd44c9bb7456feb608b9` and the retained guidance manifest
`816989ab02e66c8c6803eb15547420e5ca8c6d58348a4817f55ed2f891dd8e7b`.
Every one of the **80 slices** has the four source-audit dimensions recorded
below: assigned responsibilities, complete established workflows, independent
challenge and exact target allocation. Each primary record contains its
intended model, actual inspection/delegated evidence, objections/responses,
findings, Go/Rust or applicable substrate ownership, hosting process and
caller/test/build/install/cutover duties. Record creation and path counts alone
are not the basis for completion.

The retained assignment ledger has **2,887 slice/path relationships across 1,716
distinct committed paths**. The catalog's other three paths are separately
accounted for: generated `installer` and `soda-candidate` binaries in the
[root inventory](../coverage/inventory/root-files.md), and the obsolete
`factory-os/Containerfile` with its actual recipe/caller disposition in the
[coverage ledger](../coverage/README.md). This accounts for all **1,719** tracked
paths without treating binary artifacts or the retired recipe as active slices.
For mixed files, completion covers each slice's assigned duties, not every
unrelated body in every reader's copy. Reused/delegated evidence is credited at
its actual inspected scope. The original catalog/structural evidence identities
remain separate from this source-validity review.

**Complete** below means source analysis and selected ownership are recorded
and independently challenged. Correctness findings remain open; unspecified
product or external producer behavior blocks only its named dependent correction.
The chosen owners/paths stay explicit. No tests, builds, provider calls, native
operations or installed qualification were performed in this audit. Source
audit completion does not certify that any slice works at runtime.

| Slice record | Primary / challenger | Assigned paths | Responsibilities | Workflows | Independent challenge | Exact targets |
| --- | --- | ---: | --- | --- | --- | --- |
| [D01](D01.md) | C / A | 28 | Complete | Complete | Complete | Complete |
| [D02](D02.md) | C / A | 46 | Complete | Complete | Complete | Complete |
| [D03](D03.md) | C / A | 69 | Complete | Complete | Complete | Complete |
| [D04](D04.md) | C / A | 23 | Complete | Complete | Complete | Complete |
| [D05](D05.md) | C / A | 42 | Complete | Complete | Complete | Complete |
| [D06](D06.md) | C / A | 67 | Complete | Complete | Complete | Complete |
| [D07](D07.md) | C / A | 14 | Complete | Complete | Complete | Complete |
| [D08](D08.md) | C / A | 9 | Complete | Complete | Complete | Complete |
| [D09](D09.md) | C / A | 5 | Complete | Complete | Complete | Complete |
| [D10](D10.md) | C / A | 10 | Complete | Complete | Complete | Complete |
| [D11](D11.md) | C / A | 29 | Complete | Complete | Complete | Complete |
| [F01](F01.md) | B / C | 12 | Complete | Complete | Complete | Complete |
| [F02](F02.md) | B / C | 8 | Complete | Complete | Complete | Complete |
| [F03](F03.md) | B / C | 21 | Complete | Complete | Complete | Complete |
| [F04](F04.md) | B / C | 11 | Complete | Complete | Complete | Complete |
| [F05](F05.md) | B / C | 20 | Complete | Complete | Complete | Complete |
| [F06](F06.md) | B / C | 30 | Complete | Complete | Complete | Complete |
| [F07](F07.md) | B / C | 50 | Complete | Complete | Complete | Complete |
| [F08](F08.md) | B / C | 72 | Complete | Complete | Complete | Complete |
| [F09](F09.md) | B / C | 35 | Complete | Complete | Complete | Complete |
| [F10](F10.md) | B / C | 15 | Complete | Complete | Complete | Complete |
| [F11](F11.md) | B / C | 24 | Complete | Complete | Complete | Complete |
| [F12](F12.md) | B / C | 19 | Complete | Complete | Complete | Complete |
| [G01](G01.md) | B / C | 45 | Complete | Complete | Complete | Complete |
| [G02](G02.md) | B / C | 11 | Complete | Complete | Complete | Complete |
| [G03](G03.md) | B / C | 20 | Complete | Complete | Complete | Complete |
| [G04](G04.md) | B / C | 18 | Complete | Complete | Complete | Complete |
| [G05](G05.md) | B / C | 9 | Complete | Complete | Complete | Complete |
| [G06](G06.md) | B / C | 9 | Complete | Complete | Complete | Complete |
| [G07](G07.md) | B / C | 7 | Complete | Complete | Complete | Complete |
| [G08](G08.md) | B / C | 403 | Complete | Complete | Complete | Complete |
| [G09](G09.md) | B / C | 5 | Complete | Complete | Complete | Complete |
| [H01](H01.md) | C / A | 61 | Complete | Complete | Complete | Complete |
| [H02](H02.md) | C / A | 25 | Complete | Complete | Complete | Complete |
| [H03](H03.md) | C / A | 82 | Complete | Complete | Complete | Complete |
| [H04](H04.md) | C / A | 80 | Complete | Complete | Complete | Complete |
| [H05](H05.md) | C / A | 160 | Complete | Complete | Complete | Complete |
| [H06](H06.md) | C / A | 286 | Complete | Complete | Complete | Complete |
| [I01](I01.md) | A / B | 20 | Complete | Complete | Complete | Complete |
| [I02](I02.md) | A / B | 26 | Complete | Complete | Complete | Complete |
| [I03](I03.md) | A / B | 31 | Complete | Complete | Complete | Complete |
| [I04](I04.md) | A / B | 36 | Complete | Complete | Complete | Complete |
| [I05](I05.md) | A / B | 33 | Complete | Complete | Complete | Complete |
| [I06](I06.md) | A / B | 27 | Complete | Complete | Complete | Complete |
| [I07](I07.md) | A / B | 4 | Complete | Complete | Complete | Complete |
| [I08](I08.md) | A / B | 9 | Complete | Complete | Complete | Complete |
| [I09](I09.md) | A / B | 40 | Complete | Complete | Complete | Complete |
| [I10](I10.md) | A / B | 7 | Complete | Complete | Complete | Complete |
| [N01](N01.md) | C / A | 24 | Complete | Complete | Complete | Complete |
| [N02](N02.md) | C / A | 16 | Complete | Complete | Complete | Complete |
| [N03](N03.md) | C / A | 34 | Complete | Complete | Complete | Complete |
| [N04](N04.md) | C / A | 25 | Complete | Complete | Complete | Complete |
| [N05](N05.md) | C / A | 24 | Complete | Complete | Complete | Complete |
| [N06](N06.md) | C / A | 25 | Complete | Complete | Complete | Complete |
| [N07](N07.md) | C / A | 9 | Complete | Complete | Complete | Complete |
| [O01](O01.md) | C / A | 14 | Complete | Complete | Complete | Complete |
| [O02](O02.md) | C / A | 14 | Complete | Complete | Complete | Complete |
| [O03](O03.md) | C / A | 4 | Complete | Complete | Complete | Complete |
| [O04](O04.md) | C / A | 19 | Complete | Complete | Complete | Complete |
| [O05](O05.md) | C / A | 7 | Complete | Complete | Complete | Complete |
| [O06](O06.md) | C / A | 5 | Complete | Complete | Complete | Complete |
| [O07](O07.md) | C / A | 9 | Complete | Complete | Complete | Complete |
| [P01](P01.md) | A / B | 52 | Complete | Complete | Complete | Complete |
| [P02](P02.md) | A / B | 45 | Complete | Complete | Complete | Complete |
| [P03](P03.md) | A / B | 57 | Complete | Complete | Complete | Complete |
| [P04](P04.md) | A / B | 51 | Complete | Complete | Complete | Complete |
| [P05](P05.md) | A / B | 29 | Complete | Complete | Complete | Complete |
| [P06](P06.md) | A / B | 12 | Complete | Complete | Complete | Complete |
| [P07](P07.md) | A / B | 52 | Complete | Complete | Complete | Complete |
| [P08](P08.md) | A / B | 14 | Complete | Complete | Complete | Complete |
| [P09](P09.md) | A / B | 14 | Complete | Complete | Complete | Complete |
| [P10](P10.md) | A / B | 17 | Complete | Complete | Complete | Complete |
| [P11](P11.md) | A / B | 16 | Complete | Complete | Complete | Complete |
| [P12](P12.md) | A / B | 26 | Complete | Complete | Complete | Complete |
| [S01](S01.md) | A / B | 19 | Complete | Complete | Complete | Complete |
| [S02](S02.md) | A / B | 19 | Complete | Complete | Complete | Complete |
| [S03](S03.md) | A / B | 11 | Complete | Complete | Complete | Complete |
| [S04](S04.md) | A / B | 45 | Complete | Complete | Complete | Complete |
| [S05](S05.md) | A / B | 33 | Complete | Complete | Complete | Complete |
| [S06](S06.md) | A / B | 33 | Complete | Complete | Complete | Complete |

## Shared-boundary queue

Detailed observations, independent requirements, source scopes, evidence limits,
objections and responses belong in the primary records. The coordinator links
them here to route all affected primaries and required independent challenges.
Source boundary exchanges and consequential challenges are complete at the pin.
Open corrections and exact owner/producer/equivalence questions remain in their
canonical records and block dependent instructions. The queue is not proof that
a workflow is correct.

| Boundary ID | Participating slices / workers | Canonical finding/workflow reference | Question and current disposition | Correction owner / remaining evidence |
| --- | --- | --- | --- | --- |
| BOUNDARY-01 | I04/I06/I09, F08, H01/H02; A/B/C | [I06-F1](I06.md#concrete-findings) | Broker lease completion releases execution to PENDING; Factory closure normally tombstones it separately. B challenged the source-supported contract mismatch; no duplicate native launch established. | A owns the broker end/expiry/return correction at the recorded existing owners; B/C retain Factory closure and transport/persistence boundaries. Native proof remains unperformed. |
| BOUNDARY-02 | H02, I03-I10, F01/F07/F08, H06; A/B/C | [H02-F1](H02.md#concrete-findings), [GUIDANCE-05](../review-assignments.md#guidance-conflicts-and-controlling-decisions) | Owner resolved backend choice on 2026-10-05: retain PostgreSQL; SQLite guidance is outdated. Historical SQLite proof remains historical; current PostgreSQL invariants still require review. | C/H02 owns the exact obsolete guidance correction and current PostgreSQL evidence limits; the backend decision is closed and no database operation is selected. |
| BOUNDARY-03 | F01/F02/F03/F04, H02; B/C | [F01-F1](F01.md#concrete-findings) | Settings effect and command receipt are distinct commits; an interrupted operation can leave an unfinished receipt. C independently supports the bounded source finding. | B owns the existing command/store receipt correction; C has challenged transaction allocation. Implementation and interruption evidence remain pending. |
| BOUNDARY-04 | F03/F04/F07, I01/I05/I09, H02; A/B/C | [F03-F1](F03.md#concrete-findings) | Repository/lifetime quota calculation differs from the shared connection rolling-window contract. C supports source mismatch; actual provider and admission boundaries remain part of A/B review. | B owns the exact quota correction; A has supplied actual provider/concurrency distinctions. Preserve the shared rolling-window contract. |
| BOUNDARY-05 | F01/F02/F04/F05/F07/F08, H02; B/C | [F08-F1](F08.md#concrete-findings) | Withdrawal captures registrations before independently closing its gate; an interleaving can escape capture. C supports that source consequence; no native launch claim follows. | B owns the specified registration/gate-capture ordering and settlement correction; C has challenged the actual transaction boundary. |
| BOUNDARY-06 | H01, I04-I06/I09, F08; A/B/C | [H01-F1](H01.md#concrete-findings) | Private HTTP chunk decoder interprets lengths as decimal. A challenged protocol evidence; current host emits Content-Length, so installed failure is unproven. | C owns framing correction in the existing private client; A/B have challenged caller consequences. Installed framing behavior is unverified. |
| BOUNDARY-07 | H01, P02/P05/P07, F07/F08, N03/N06; A/B/C | [H01-F2](H01.md#concrete-findings) | Native executor waits for process exit before draining piped output. A challenged the actual inspect caller and conditional blocking consequence; no live Podman failure reproduced. | C owns the executor pipe-drain correction; A/B have checked affected operations and confirmation semantics. Preserve actual process/deadline ownership. |
| BOUNDARY-08 | H03, P03/P06, I01/I07/I08; A/C | [H03-F1](H03.md#concrete-findings) | Truncated Unicode surrogate input can panic in the Rust JSON parser; A independently checked source and actual bounded-input callers. | C owns actual parser bounds/error correction and tests; A has challenged the bounded-input callers and retained error contract. |
| BOUNDARY-09 | F05/F06, G03, H02; B/C | [F05-F1](F05.md#concrete-findings) | Acceptance replay refreshes native evidence before retrieving the recorded decision. C confirms the bounded replay-order defect; current visibility and new admission checks remain separate. | B owns the replay/new-admission correction at existing coordinator owners; C has checked actual persisted authority. No native replay outcome is claimed. |
| BOUNDARY-10 | F06, G03, H06; B/C | [F06-F2](F06.md#concrete-findings) | Intake imports the publication executor solely to inspect a bounded diagnostic error; the current architecture test forbids that edge. C confirms dependency mismatch, without a native execution claim. | B owns diagnostic placement preserving decided behavior; C has checked the source-test dependency boundary. No executor/service port follows. |
| BOUNDARY-11 | F04, I01/I03/I05/I08, H02; A/B/C | [F04-F2](F04.md#concrete-findings) | Sponsorship reads dashboard identity rows despite the separate broker database contract; inspected production writes belong to the broker. C independently confirms the source-of-truth gap. Operator provisioning of broker config/key is established, so missing automatic setup is not itself a defect. Installed database topology remains unobserved. | B specifies correction using existing broker administration queries where adequate; A challenges grant/connection authority, C challenges custody and transaction boundaries. |
| BOUNDARY-12 | F08, I06/I09, H02; A/B/C | [F08-F3](F08.md#concrete-findings) | Successful native finish can overwrite a concurrent stop-owned disposition after its earlier stop check. C independently supports lost stop attribution; no surviving process or credential exposure is established. | B owns finish/stop settlement correction; A has checked broker closure and C persisted authority. Installed termination evidence remains absent. |
| BOUNDARY-13 | P05/P12, F07/F08, H02; A/B/C | [P05-F1](P05.md#concrete-findings) | Explicit Project Start leaves the stored Project-stop dispatch withdrawal closed despite the product contract saying Start clears that lifecycle gate. B independently supports the persisted authority and admission chain; no native start failure is established. | A/B specify the exact cause, grant and quiescence correction, preserving other withdrawals and the independently owned maintenance hold. |
| BOUNDARY-14 | S04/S05, H04; A/B/C | [S04-F1](S04.md#native-account-and-pty-child-scope-s04-f1) | Native PTY launch passes newly formatted Rust strings to execve without C-string terminators. B independently challenged the actual ABI/caller chain; conditional invalid native reads or launch failure are source-supported, with no runtime or disclosure claim. | A owns the specified existing PTY environment correction and real-subject assertion allocation. Preserve admitted account, environment and lifetime semantics. |
| BOUNDARY-15 | S04/S05; A/B | [S05-F1](S05.md#actual-native-relay-production-and-assertions-s05-f1) | Queued PTY writes supply the writable set as select's exceptional set, so ordinary writability does not trigger the real flush. B independently supports the syscall/caller finding; no interactive failure executed. | A owns the specified select-set correction and actual relay assertions in the existing Rust helper. The complete human/Factory lifetime traces are recorded. |
| BOUNDARY-16 | D03/H06; A/C/coordinator | [D03](D03.md#performed-evidence-and-limits), [retirement disposition](../decomposition/release-production.md#rustsoda-release-buildsrcprogressrs) | Coordinator inspection and A's independent source/caller census establish the release-build mirrored progress/clock/BuildExecution closure as obsolete. Six exclusive desired leaves are removed; active release-tools progress and release-image Runner remain. Source is untouched. | C has incorporated the supported retirement and exact allocation; the six obsolete leaves are excluded. Broader D03 findings and actual live executor ownership remain separately recorded. |
| BOUNDARY-17 | N03/N07, D03/H06; A/C/coordinator | [N07](N07.md#language-and-exact-target-allocation), [exact cutover](../decomposition/host-runtime.md#current-forgejo-tailnet-helper-allocation) | Current soda-forgejo-tailnet is a live privileged Go helper. The controlling Rust system policy applies. A independently challenged C's native duties, actual Endpoint/listener guard, current caller, package and retained tests. | Shared tree/placement now select cmd/soda-forgejo-tailnet/main.rs and lib/host/src/tailnet/forgejo.rs in existing soda-host Cargo package, preserving installed binary and private tests in that same module. Explicit package/binary compile selection and exclusive Go retirement are recorded; no new service/package or browser-origin change. |
| BOUNDARY-18 | H01/S05, S04; A/B/C | [H01-F3](H01.md#h01-f3-terminal-output-read-owns-the-mutex-needed-for-input-and-expiry) | Live terminal output performs a blocking child read while holding the mutex needed for input and expiry close. A, B and C independently inspected the actual pipe/pump chain and support the conditional quiet-child obstruction. No native stall was executed. | C owns same-host daemon websocket pump allocation; A owns terminal attachment counterpart and real input/lifetime assertions. Keep one native session and existing authority/protocol; no new service. |
| BOUNDARY-19 | F07/P07, F03/I09; A/B/C | [F07-F2](F07.md#concrete-findings) | The recorded coding prompt omits contracted policy, accepted dependency outcomes, approved repository instructions/setup/verification and template revision. C independently supports the actual input/assembly gap. A native CLI reading files does not establish the recorded context contract; no wrong generated code was observed. | B specifies the existing Go prompt/assignment correction using the owning approved context sources; A supplies preparation/provider boundaries. No new context collector or process is inferred. |
| BOUNDARY-20 | F08/F07/F03, H02; B/C | [F08-F2](F08.md#concrete-findings) | Retry reevaluates a queued issue without supplying a fresh dispatch attempt; the actual guard rejects the unchanged acceptance with historical assignment. C independently supports the producer/consumer gap. | B preserves attribution and usage while specifying an explicit fresh attempt with a new bounded allowance at the existing coordinator/store owners; do not erase history or replenish prior usage. |
| BOUNDARY-21 | F10/G05, F02/F05; B/C | [F10-F2](F10.md#concrete-findings), [G05](G05.md#concrete-findings) | The review producer reconstructs an immutable operation from current actor, revision and deadline instead of retaining the originally registered work and withdrawal/recovery capture. C independently supports the source mismatch; no duplicate native operation was executed. | B specifies the original operation registration/recovery correction at existing Go owners and challenges the native handoff. No new work ledger or review service follows. |
| BOUNDARY-22 | F12/G07, F09/F11; B/C | [F12-F1](F12.md#concrete-findings), [G07](G07.md#concrete-findings) | Exact target-tip equality rejects a legitimate later target descendant of the attributable merged result. C independently supports the bounded predicate defect; native producer bytes and successful reachability proof remain unretained. | B retains all result/actor/target/bookkeeping checks and requires actual native reachability proof. Removing equality alone is insufficient; no invented native wire field or local merge engine is selected. |
| BOUNDARY-23 | D06/D07/H06; A/C | [D06-F2](D06.md#concrete-findings) | A child phase's requested deadline becomes absent when its parent has no deadline, due to Option ordering. A independently supports the actual driver/VM caller consequence; no native hang was executed. | C corrects the existing phase deadline calculation and actual parent/child assertions, retaining cancellation and tighter parent deadlines. |
| BOUNDARY-24 | D06/D11, S02/S05/H06; A/C/coordinator | [D06 findings F3–F7](D06.md#concrete-findings) | Qualification assumptions differ from current producers: retired Cockpit page requirements, URL redaction, Linux vocabulary, passwordHash spelling and original terminal-shell identity. Independent challenges and exact source limits are recorded per finding. Synthetic fixtures do not establish producer correspondence; no installed failure or credential exposure is claimed. | C owns the existing driver/probe corrections and assertion allocation; A challenges terminal/input contracts. Preserve the established UI, platform, secret scope and terminal lifecycle; no substitute service or broader product contract. |
| BOUNDARY-25 | N03/H01/H03; A/C | [N03-F1](N03.md#concrete-findings) | The actual LocalAPI client passes HTTP transfer framing into JSON admission unchanged. Independent source/protocol review supports a conditional incompatibility with legitimate chunked status responses; it does not establish that every current response fails or an installed outcome. | C specifies framing normalization inside the existing native client and actual source-subject tests; preserve bounded admission and the existing host process. |
| BOUNDARY-26 | O07/H04/N01; A/C | [O07-F2](O07.md#concrete-findings) | The enrollment window checks its selected address at startup, then checks only armed state/time during its lifetime; the owning guide requires abort on address change. A independently challenged the live-address caller chain; no native change or key import was exercised. | C specifies ongoing address-binding checks at the existing window owner and actual lifecycle tests. Preserve the separate receiver admission; no second daemon or force unlock. |
| BOUNDARY-27 | D01/D03/H06; A/C/coordinator | [D01-F2](D01.md#concrete-findings) | The Rust candidate setup still builds the removed Go controller/wrapper paths and checks for a Go executable stamp. Root and A checked the actual call sites, absent tracked paths and existing Rust binary owners; its tests stop before that phase. | C specifies compilation and admission of the existing Rust release-tools binaries while retaining the current install identities and privilege boundary. Do not restore obsolete Go tools or add a build service. |
| BOUNDARY-28 | D01/H06; A/C/coordinator | [D01-F3](D01.md#concrete-findings) | Failed Git-status inspection can be treated as a clean source preflight before setup reaches its first privileged mutation. Root and A independently checked the actual branches and owning clean-source requirement. Downstream controller readmission is separate; no unclean image or setup effect was exercised. | C specifies explicit successful source inspection at the existing setup owner and actual failure-path tests. Preserve the canonical checkout and separate Forgejo source requirements. |
| BOUNDARY-29 | D01/D03/H06; A/C | [D01-F4](D01.md#concrete-findings) | The current controller records interruption but supplies a constant-false cancellation predicate to the synchronous worker attempt. A independently challenged the actual signal, dispatch and existing unit-stop chain. No surviving unit or native signal outcome was observed. | Correct signal propagation in the existing Rust controller/worker owners and retain exact-unit stop, reap and cleanup semantics; no new process or cancellation service. |
| BOUNDARY-30 | D01/H06; A/C | [D01-F5](D01.md#concrete-findings) | Padded START and DONE labels retain different leading spaces and fail to identify the same displayed phase. A independently challenged the actual producer/parser/caller chain; historical parity assertions can preserve this defect. | Normalize phase identity at the existing parser/renderer boundary while retaining the event wire, lock/cause/ticker ownership and real subject tests. No terminal or rendering behavior was executed. |
| BOUNDARY-31 | D03/D06/H06; A/C/coordinator | [D03-F3](D03.md#concrete-findings) | Current candidate shipping-tools compilation includes only the acceptance driver, whose actual payload loader requires its existing remote companion beside it. A independently confirms the current Rust producer/consumer gap; no candidate operation was executed. | Stage the existing same-package soda-acceptance-remote binary into artifacts/tools before recorded inventory at the existing Rust image compile owner, preserving target binding and executable identity. |
| BOUNDARY-32 | F07/F08/F09, I09/S06; A/B/C/coordinator | [F09 shared owner correction](F09.md#shared-native-model-owner-correction), [exact target](../decomposition/host-runtime.md#current-shared-factory-terminal-ownership) | Actual both-family run/output/artifact/native/lifecycle consumers contradict inherited Codex-only target ownership. Primary B and independent C accept the inspected shared owner correction; this is architectural allocation debt, not evidence of a native failure. | Shared Rust terminal/factory defining owners and direct imports replace misplaced Codex targets inside the same host crate/Service. Preserve distinct family policy, one test fixture owner and exact behavior; no sidecar/facade/duplicated DTO. |

## Evidence and remaining scope

The committed inventory remains 1,719 paths at its recorded catalog baseline;
the pinned source delta changes documentation only. Audit records and retained
guidance snapshots are separate from that earlier inventory/evidence scope.
The derived ignored artifact `assigned-files.json` under the retained input
packet's parent folder enumerates each slice's full assigned file set. It is
coverage bookkeeping, not a substitute for inspecting the detailed map units,
actual source, canonical requirements or tests' assumptions.

No implementation, test assertion, dependency, database, API, service or Git
state change follows from a finding or executable-looking correction. Questions
link exact blocked requirement/target/action IDs in the owning record. Findings
come before any later separately authorized implementation.

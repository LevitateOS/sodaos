# Host runtime

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

## Current library adoption and retained adapters

The [library adoption allocation](../library-adoption.md#finding-allocation)
supersedes pending extraction of the generic engines identified below. Existing
completed concern modules remain useful locations for Soda adapters; historical
source ranges and test maps remain provenance, not instructions to preserve an
engine or every Go/Python parser quirk. The
[execution packets](../library-adoption.md#execution-packets) own sequencing and
physical writers; [readiness gates](../library-adoption.md#readiness-gates) own the
small integration proofs before transport replacement.

| Finding and existing surface | Current target responsibility |
| --- | --- |
| [N1](../../../research/library-reuse-investigation.md#n1), [N2](../../../research/library-reuse-investigation.md#n2): `daemon/{http,response}.rs`, `gmux_server.rs`, `iclient.rs`, `tcontrol_native.rs` | Adopt complete Hyper HTTP engines inside the existing processes. Retain systemd listener custody, each socket's actual authority, route admission, request/response limits, status/secrecy mapping and bounded synchronous backend adapters. Preserve the existing Go library-backed clients. |
| [N6](../../../research/library-reuse-investigation.md#n6): `daemon/websocket.rs`, `dbackend.rs`, terminal attachment | Adopt tungstenite with one nonblocking protocol owner, bounded child-output queue and readiness wakeup. Retain route/Origin/query admission, session expiry, inflight/TerminalGate lifetime, child close/reap and shutdown. Handshake SHA-1/Base64 and handwritten frame state disappear with this engine. |
| [N4](../../../research/library-reuse-investigation.md#n4), [N7](../../../research/library-reuse-investigation.md#n7): `tcontrol_provider.rs`, native/provider URL helpers | Use the selected blocking HTTPS engine and URL/form/percent primitives. Retain credential scope/lifetime, neutral errors, no uncertain replay and raw lexical admission; generic curl recipes and authority/escape algorithms are superseded. |
| [N8](../../../research/library-reuse-investigation.md#n8), [N9](../../../research/library-reuse-investigation.md#n9): Tailnet address/time and terminal/Factory deadline codecs | Use std IP types and the selected time codec. Retain zone/mask, DNS/name and purpose-specific address policy, lease/deadline bounds, zero-time handling and original signed text; calendar/IP engines are superseded. |
| [CF-01](../../../research/library-reuse-investigation.md#cf-01)–[CF-04](../../../research/library-reuse-investigation.md#cf-04): SHA, NIST, SSH and Base64 helpers | Adopt RustCrypto, ssh-key and explicit Base64 profiles. Retain admitted algorithms/options, uncompressed curve gates, canonical public-key output, raw-byte digest recipes and managed-key revision/ownership policy. Do not extract separate crypto/SSH engines. |
| [N10](../../../research/library-reuse-investigation.md#n10), [N11](../../../research/library-reuse-investigation.md#n11): Muse packet/peer helpers and obsolete daemon peer adapter | Use typed descriptor/socket mechanics where they simplify the active owner. Retain exact stdio-FD admission, kernel peer/pidfd pinning, cgroup/account custody and cleanup. Delete the unused daemon peer duplicate; HTTP does not replace the Muse packet channel. |
| [JSON01](../../../research/library-reuse-investigation.md#json01): host and maintenance JSON scanners/binders | Adopt serde/serde_json with explicit caller DTO/Visitor profiles. Keep ordered aliases, null/byte-field admission, caps and canonical raw-byte boundaries; generic scanner/string/number state machines are superseded. |

Host HTTP relies on its root-owned filesystem/systemd socket admission; it does
not currently apply a production peer-UID gate. Muse's packet service has its own
kernel peer/pidfd checks. Keep these authority rules at their actual owners.
Before host HTTP/WebSocket cutover, prove bounded backend work, upgrade read-ahead
preservation into the std tungstenite stream and complete session/child cleanup.
Typed syscall or randomness changes elsewhere are not a prerequisite.

## Current Forgejo Tailnet helper allocation

[N07](../reviews/N07.md) and A's independent current source/caller challenge
select Rust for the existing privileged one-shot helper. Keep the installed
`/usr/libexec/soda/soda-forgejo-tailnet` and current
`tcontrol_native.rs:792–806` caller. Add this binary to the existing `soda-host`
Cargo package: `cmd/soda-forgejo-tailnet/main.rs` calls the defining native
owner `lib/host/src/tailnet/forgejo.rs`. It has no separate manifest or service.

That one native module owns the root admission, 90-second command deadline,
actual Tailnet Endpoint observation, bounded Podman state decoding, real
22/tcp binding to the selected IPv4 at port 2222, SSH_DOMAIN-only rewrite,
0600 temporary file/atomic rename, conditional restart and existing diagnostic
behavior. Reuse the actual Rust Endpoint observer and predicate with the
equivalent required fields. An observation-only hostname does not replace the
native listener guard. Retain both current Go test subjects as a private
`cfg(test)` descendant in this same `forgejo.rs`: actual bound listener and
credential-free refusal, browser/OAuth preservation and unchanged-content
behavior. No extra test file is required by their current size.

At cutover, release-image's existing `compile_soda_commands` must select
Cargo package `soda-host`, binary `soda-forgejo-tailnet`, and the same installed
destination explicitly; its current package-equals-binary loop is insufficient.
Go command enumeration must exclude Rust-only entrypoints after relocation.
Rebind real source/build tests and fixture paths. Retire the entire Go
`cmd/soda-forgejo-tailnet/main.go`/`main_test.go` and helper-exclusive
`internal/forgejo/tailnet.go`/`tailnet_test.go` only after their required behavior
and tests are bound to this successor. Keep live Go Tailnet DTO/status clients
and the remaining Forgejo client package. Source review does not establish
native listener, restart or installed parity; no implementation was performed.

## Current terminal predecessor retirement

At `f7e9cf9d`, `texec.rs:2367–2440` defines a private terminal-request predicate
and StreamTable that have only defining and own-test uses. Actual
`gmux_admission.rs:190–193` and the live TerminalGate/route owner retain current
request admission, stream limits and shutdown. A's actual source/caller census,
B's independent challenge and the coordinator's body inspection support
retiring exactly that predecessor closure from the target allocation.

Retire `stream_table_flows` (`4561–4584`) and the private-terminal predicate
assertions (`4586–4591`). Keep `private_request_matrix`'s identity assertions
(`4592` onward), current frame/metadata/identity/attachment tests and the live
gmux lifetime owner. Do not delete NativeAttach or terminal transport behavior.
See [S05](../reviews/S05.md) and [H01-F3](../reviews/H01.md#h01-f3-terminal-output-read-owns-the-mutex-needed-for-input-and-expiry)
for the separate live transport ownership defect. Source and tests remain
unchanged; this paragraph refines the desired allocation only.

## Current native Factory test allocation

The primary F08 reviewer inspected the whole current `pfactory.rs`. The
coordinator inspected the actual declaration boundaries, selected assertion
bodies and support seams; B challenged that exact partition against the full
source. Retain Rust tests as private `cfg(test)` descendants of the existing
`lib/host/src/factory` owner and exercise its real Factory/receipt/lock,
using the current injected native/broker seams. No substitute lifecycle or
production API exposed solely for tests is required.

| Exact target | Current source duties |
| --- | --- |
| `lib/host/src/factory/tests/mod.rs` | `cfg(test)`/module/import shell at 3195–3201 and closing brace at 6081; declare the private test modules once. |
| `lib/host/src/factory/tests/common.rs` | Existing counter/path/deadline/ID/run/launch/lease/binding fixtures at 3202–3322; test_factory/receipt_bytes at 3754–3766; fixed wire records/strings and dummy_factory at 3979–4064; wired_factory 4375–4392, drive_to_start 4612–4624, shared write_running 4979–4997, wired_factory_exec/preparation_target 5525–5550, sample_state 5806–5815 and preparation-state fixtures 5984–6022. Retain one defining fixture owner; these are not production decoders. |
| `lib/host/src/factory/tests/mocks.rs` | ExecCall alias and actual FakeExec/FakeTerminal/FakeBroker implementations, scripted seams and Rc delegation at 3323–3753. The 431-line cohesive seam is inspected explicitly; do not create a file per trait method. |
| `lib/host/src/factory/tests/wire.rs` | Validators/deadline/error assertions at 3767–3978 and open_factory/harness-pin assertions at 4276–4374. |
| `lib/host/src/factory/tests/receipt.rs` | Real receipt encoding, private file/mode, tombstone and decoder assertions at 4065–4275; retain one receipt implementation. |
| `lib/host/src/factory/tests/launch.rs` | Launch success/provider/duplicate/precheck/acquire/abandon cases at 4393–4611 and tombstone replay at 4791–4811. |
| `lib/host/src/factory/tests/finish.rs` | Failed-start, timeout and finish-state cases at 4625–4790, against real Factory transitions. |
| `lib/host/src/factory/tests/stop.rs` | Stop/closure/uncertainty cases and stop-local receipt helpers at 4812–4978 and 4998–5080, including stop-owned launch and pre-delivery regressions at 4882–4978. Import shared write_running from the one common fixture owner. |
| `lib/host/src/factory/inspect.rs` | Keep inspect assertions 5081–5150 as an inline private test descendant of this 69-line production owner; use existing common/mocks and the single real receipt fixture. |
| `lib/host/src/factory/tests/artifacts.rs` | Human takeover, output cursor and export cases at 5151–5524. |
| `lib/host/src/factory/tests/candidate.rs` | Exact native candidate script, clean/dirty/incarnation and decoded candidate-shape assertions at 5551–5805; these are candidate inspection, not terminal display tests. |
| `lib/host/src/factory/tests/confirmation.rs` | Factory confirmations/status mappings at 5816–5983 and preparation confirmations at 6023–6080; preserve P07's preparation dependency. |

Keep each test attribute with its full function. The fixture and mock modules
use only the internal visibility justified by actual callers. C's independent
allocation challenge remains in [F08](../reviews/F08.md); this is a concrete
candidate partition, not executed lifecycle or installed qualification.

## Current factory Muse and shared lifecycle allocation

The connected source review at `f7e9cf9db616f93de351dd44c9bb7456feb608b9`
accounts for the active `rust/soda-host/src/tmuse.rs` and `tfactory.rs`, which
the earlier physical split omitted. [I09](../reviews/I09.md) records provider
semantics; [I04](../reviews/I04.md), [I05](../reviews/I05.md) and
[I06](../reviews/I06.md) record related enrollment, lease and lifecycle duties.
Both remain Rust in the existing `soda-host` process and `lib/host` crate.
The module splits add no service, database or authority boundary.

`tmuse.rs` has 576 lines before its embedded test module and 1,218 total lines:

- Lines 1–148: path records/builders, Muse binding policy and guest locator →
  `lib/host/src/terminal/factory/muse/paths.rs`.
- Lines 149–250: fixed supervisor, setup/start gates and argv →
  `lib/host/src/terminal/factory/muse/commands.rs`.
- Lines 251–436: pinned harness verification, reservation and staging →
  `lib/host/src/terminal/factory/muse/reserve.rs`.
- Lines 437–474: opaque credential/prompt staging and marker order →
  `lib/host/src/terminal/factory/muse/start.rs`.
- Lines 475–576: wait/attest/stop/capture/live/output adapters →
  `lib/host/src/terminal/factory/muse/lifecycle.rs`.
- Test support 577–737 → `lib/host/src/terminal/factory/muse/tests/common.rs`;
  paths 739–796, commands 798–850, reserve 851–1060, start 1062–1139 and
  lifecycle 1141–1217 → respective `paths.rs`, `commands.rs`, `reserve.rs`,
  `start.rs` and `lifecycle.rs` in that same test directory. These current
  intervals include each case's actual attribute; verify_harness_matrix belongs
  with reserve from 851. The cfg(test)/module shell at 576 and closing shell at
  1218 travel once with tests/mod.rs; preserve one common fixture owner. A's
  primary inspection and B's independent actual seam/import challenge support
  this allocation, without executed provider or native qualification.

`tfactory.rs` has 379 lines and retains the shared `Service` implementation:

- Lines 22–25 and 37–74: path-resolver signature and binding checks
  → `lib/host/src/terminal/factory/binding.rs`.
- Neutral alias19–20 and cursor gate27–35
  → `lib/host/src/terminal/factory/output.rs`.
- Lines 75–263: supervised wait, attestation, stop, capture and live observation
  → `lib/host/src/terminal/factory/lifecycle.rs`.
- Lines 264–379: output binding and bounded cursor window
  → `lib/host/src/terminal/factory/output.rs`.

Wire these files through `terminal/factory/mod.rs`, `muse/mod.rs` and
`muse/tests/mod.rs` in the existing terminal module. Rebind the existing
`dbackend`/`pfactory` callers and family tests to these modules. Preserve one
existing output DTO definition and neutral `FactoryOutputSlice` alias once;
shared run/native/output helpers have the explicit defining owners below. Muse keeps its provider/scope/generation
checks and immutable borrowed credentials. Splitting files does not merge
family admission policy or alter native argv, marker order or broker return.

B independently checked both production allocations against actual source.
The owning records close the independent source-fit challenge and specify
complete caller/cutover instructions below. Target presence is not proof of
compilation or installed execution.

## Current shared Factory terminal ownership

The current 3,089-line `rust/soda-host/src/tcodex.rs` contains both-family
models and native mechanics used by `tfactory.rs`, `tmuse.rs`, `pfactory.rs`
and `dbackend.rs`. Its filename does not establish Codex ownership.
[F07](../reviews/F07.md), [F08](../reviews/F08.md) and
[F09](../reviews/F09.md) record the primary inspection and independent challenge;
the coordinator inspected the defining units and actual shared callers.
The following allocation supersedes the historical Codex ranges. All owners
remain Rust in the existing `soda-host` crate/Service and process; they add no
package, sidecar, API or provider policy.

| Exact defining owner under `lib/host/src/terminal/factory/` | Current complete source units at f7e9cf9d |
| --- | --- |
| `run.rs` | tcodex24–226: BOTH family/scope constants, fixed roles, prompt bound43–44, validators and complete FactoryRun declaration/spec/validate/decode. Exclude only output bounds45–50 and artifact constants51–59, whose single owners follow below. Decoder closes226; 219 is not a complete unit. |
| `native.rs` | tcodex unit name254–264; shell quoting/systemd escaping335–343; user bus/current euid/systemctl/systemd argv475–504; shared unit header543–554; harness install571–580; stage-file command582–589 and actual Service stage-file method1083–1103; UnitShow/parser/role-ID/sliced wait681–726 and existing Service env/Podman/systemctl/systemd/unit/invocation/role-ID methods729–833. Retain one Executor and Service, private current_euid/run_env and existing pub(crate) run_podman; no unnecessary access widening. |
| `binding.rs` | tfactory RunPathsFn22–25 and checked_binding_paths37–74; tcodex broker operation1444–1474. Preserve exact family admission differences and dispatch: Muse validate/stop and finish denial, Codex mutable finish. The alias/output-cursor gate are allocated to output.rs below rather than inherited binding ownership. |
| `lifecycle.rs` | tfactory75–263; tcodex retirement script374–404, await-inactive1149–1169, retire1171–1194, read-PID1196–1215 and container-existence1217–1227. The script uses only run_dir: accept that same string directly and remove the artificial FactoryCodexPaths shim, preserving exact script bytes, marker/kill/scan, native sequence, errors, deadlines and quiescence assertions. No family policy is consolidated. |
| `output.rs` | tfactory neutral alias19–20, cursor gate27–35 and native output264–379; tcodex output bounds45–50, ONE DTO325–333, read command602–609 and size parser1477–1485. Keep the existing actual DTO and neutral FactoryOutputSlice alias once, with direct provider/shared imports; no second DTO or facade. |
| `artifacts.rs` | tcodex constants51–59; takeover destination/source266–280; export comments/script406–434; takeover steps611–649; export argv651–677; native export1298–1339 and takeover comments/attribute/method1341–1442. These operations admit the recorded settled role/preparation/container/candidate, with no Codex discriminator. Keep the complete takeover closing brace1442. |
| `tests/run.rs` | tcodex run_validate_pins attribute1670 and whole case1671–1796, using the actual shared FactoryRun/strict decoder and existing factory_run fixture once. |
| `tests/artifacts.rs` | tcodex export attribute2839/case2840–2916 and takeover attribute2919/case2920–3025. The next attribute3027 belongs to the separate identity-operation matrix. Use actual shared Service/artifact subjects and the existing FakeExec fixture, never copied production behavior. |
| `tests/mod.rs` | One new private test-module declaration shell under existing terminal/factory/mod.rs. Do not duplicate the production Service or current Codex/Muse mock closures. |

Keep the actual defining test support at
`lib/host/src/terminal/codex/tests/common.rs`: constants1494–1501,
counter1503, deadline1505–1507/test_tmp1509–1514, RecordedCall1516,
FakeExec1518–1555, ok1557–1559/err1561–1563, make_service1565–1575,
inspect_json1577–1581 and factory_run1583–1598. The new shared tests import
only their required items/methods through bounded test-only
`pub(in crate::terminal)` module/item visibility. The synthetic make_service
fixture explicitly has Codex pins and empty Muse pins; it tests shared mechanics,
not Muse admission or installed qualification. Preserve the distinct Muse mock
closure and Codex-specific lease/run-dir/harness fixtures without merger/copy.
Existing mixed path, quote, unit-show, script-golden, output and callback cases
remain whole at their recorded Codex test destinations and directly import the
single shared subjects. Update retirement-builder calls to the same run_dir;
the export golden at1914 imports the actual shared script owner.

Rebind actual tcodex/tmuse/tfactory/pfactory/dbackend callers, module declarations,
private types/specs/helpers and fixture imports together. Shared consumers import
run/native/binding/lifecycle/output/artifacts directly; a Codex forwarding facade
must not preserve the misplaced ownership. Preserve family-specific .codex and
Muse paths, guest/auth policy, harness verification and supervisor/start gates.
Keep the existing Cargo library/binary and release/install identity. Source
inspection establishes this allocation and its preserved contract; it does not
establish compilation, native stop/return/export or installed provider behavior.

## Current Project operations adapter allocation

[P12](../reviews/P12.md) inspected all of the active
`rust/soda-host/src/pops.rs` (286 lines) and its actual
`tests/pops_oracle.rs` (870 lines), including duties owned by other Project
slices. Retain the thin `Ops` adapter, borrowed `Runtime` and current request
and result types in `lib/host/src/project/operations.rs`. Rebind `lib.rs`
and the existing `dbackend.rs` construction and operation calls. This remains
Rust in the existing host crate and process; it adds no facade or authority.

Split the actual integration assertions under
`lib/host/tests/project_operations/` as follows:

- `main.rs`: imports at 1–25 and adapter construction at 343–350.
- `common.rs`: constants at 26–48, mock at 50–110, fixtures at 112–188 and
  argv helpers at 352–386.
- `requests.rs`: request assertions at 190–341 and 853–870.
- `access_keys.rs`: key operations at 387–479 and 837–851.
- `accounts.rs`: account operations at 480–524.
- `preparation.rs`: preparation operations at 525–572, 673–734 and 788–835.
- `candidate.rs`: candidate collection at 573–672.
- `hold.rs`: hold confirmation at 735–787.

Use `main.rs` as the discoverable multi-file integration target, declaring
the sibling modules and importing the actual public host library API. The
[Cargo package layout](https://doc.rust-lang.org/cargo/guide/project-layout.html)
specifies this entrypoint for multi-file integration tests. Do not replace the
production subject with a copied implementation. The existing
`lib/host/src/factory/confirmation.rs` retains native hold confirmation;
the adapter calls that owner. B independently checked both whole source files
and the actual callers, supporting this adapter and test partition. The current
oracle uses a private `#[path]` copy despite the public library module: the
cutover must replace that duplicate subject with the actual library imports,
retaining the scripted executor and golden provenance. This source challenge
does not qualify native execution or complete hold concurrency and cutover
review in the owning record.

## Current broker-client integration test allocation

The complete current `rust/soda-host/tests/iclient_oracle.rs` contains broker
request/response/transport assertions, daemon-config goldens and one mixed case
checking validators reused by Config. These have distinct subjects. H01 owns
the transport assessment, H03/H04 the mapped codec/config responsibilities, and
P12's keyword-selected error-substring case is reassigned to H01. C's primary
actual source/declaration census and A's bounded independent defining-unit
challenge select the following exact destinations. They remain test owners in
the existing Rust host package, not additional processes or mock services.

| Exact retained target | Complete current defining units |
| --- | --- |
| `lib/host/tests/identity_transport/main.rs` | One integration root declaring common/requests/responses and importing the actual public host library's selected identity/client and shared types. Replace old #[path] production-module copies. No duplicate BrokerClient, Config, decoder or validator subject is retained. |
| `lib/host/tests/identity_transport/common.rs` | Source1–176 actual broker socket/capture/reply/deadline/request/lease fixtures, full_binding230–245, DELIVERY_JSON246 and sized_lease499–505. Rebind used imports; the old config #[path]/load_config imports move to their real config subject. Retain ONE FakeBroker/socket counter/helper owner; child visibility is bounded to this private suite. |
| `lib/host/tests/identity_transport/requests.rs` | Whole request/body/registration/return/deadline-encoding/HTML-escaping cases with attributes177–229 and248–444, importing actual BrokerClient/types and the single common fixture owner. |
| `lib/host/tests/identity_transport/responses.rs` | Whole error/substrings/body-cap/strict-decode/deadline/transport/framing cases with attributes445–498 and506–692. The substring case starts attr483/body484 and ends497; the sized_lease helper499–505 remains common. Do not misplace this broker protocol in terminal PTY tests. |
| `lib/host/src/config/tests.rs` | Config support694–705, including GOLDENS696, PROJECT_IMAGE697–698, TAILNET_IMAGE699–700 and golden702–704; complete config cases706–881 and whole reused-validator case attr883/body884–911. Import the actual Config and domain/factory/network/terminal/JSON subjects once; preserve fixture bytes, loader/overlay/error-order assertions and test-only provenance. |

Current `rust/soda-host/Cargo.toml` has no explicit test declarations; the
current oracle is auto-discovered. The new multi-file suite is discoverable at
`tests/identity_transport/main.rs` with target name `identity_transport`, as
specified by the [Cargo package layout](https://doc.rust-lang.org/cargo/guide/project-layout.html).
Rebind any actual explicit old test-name selection and source-path assertions
with that move; do not describe a nonexistent old manifest declaration. Ordinary
Cargo package test discovery remains in the same host package. Config tests
are private real-subject descendants, not another integration crate.

Keep the existing iconfig golden data at `lib/host/tests/data/iconfig/` and
rebind its loader paths relative to the retained Cargo package root. Existing
`terminal_transport.rs` keeps actual terminal transport duties; it does not own
the complete broker administration/request oracle. Each selected transport
concern is below400 current source lines after cohesive fixture extraction.
Source inspection and imported fake socket/HTTP fixtures do not prove native
identity-provider behavior, installed admission or test execution.

## internal/host/daemon.go

Observed size: 644 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: NewDaemon at 212; NewCompanionDaemon at 238; ServeHTTP at 570; LoadConfig at 55; validateRuntimeConfig at 104; validateMuseRuntime at 636; acquireAdmission at 257; validateNativeOperationRequest at 307; makeBodyDecoder at 329; dispatchCreate at 348; dispatchPrepare at 416; dispatchFactory at 457.

## internal/host/project/factory.go

Observed size: 837 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: OpenFactory at 56; HarnessPin at 76; Launch at 249; loadReceipt at 137; storeReceipt at 168; storeTombstone at 684; drive at 280; acquireRunLease at 346; consumeStart at 452; finishRun at 565; Stop at 611; stopTimedOut at 537; reconcileRunCredential at 712; Inspect at 743; Takeover at 776; recordStopOutcome at 810.

## internal/host/project/prepare.go

Observed size: 514 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Prepare at 187; prepareTargetReady at 68; mapPreparationState at 136; clonePreparationSource at 283; confirmPreparationHead at 313; verifyLauncherEnvironment at 342; resolvePreparationTools at 405; recordPreparationTools at 429; InspectPreparation at 442; StopPreparation at 456; HoldPreparation at 488.

## internal/host/publish/operation.go

Observed size: 769 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/publish/operation.go` — Conditional branch/PR submission, lookup/cancel and native outcome mapping.
- `internal/forgejo/publish/operation_observation.go` — Exact Git ref advertisements and bracketed observation.
- `internal/forgejo/publish/candidate_validation.go` — Existing candidate bundle and credential validation in temporary repository.
- `internal/forgejo/publish/operation_push.go` — Bounded credentialed Git push and observed-effect reconciliation.
- `internal/forgejo/publish/operation_receipts.go` — Strict branch/PR receipt decoding and expiry extraction.

Evidence: SubmitPublish at 194; SubmitPRCreate at 260; LookupOperation at 368; CancelOperation at 421; ObserveForPublish at 60; observeTips at 126; observeTip at 146; PrepareValidated at 529; verifyCandidateBundle at 558; validationVerdict at 576; PushBranch at 602; reconcilePush at 633; DecodePublishReceipt at 672; DecodePRCreateReceipt at 722; OperationNotAfter at 755.

## internal/host/publish/operation_test.go

Observed size: 574 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/publish/operation_test.go` — Exact intent mapping, submission/lookup/cancel and lost reply.
- `internal/forgejo/publish/operation_receipts_test.go` — Strict receipt scope and trailing-document refusal.
- `internal/forgejo/publish/operation_git_test.go` — Native advertisement and exact branch mutation cases.

Evidence: TestSubmitPublishMapsTerminalDispatchVerdicts at 84; TestSubmitPublishReconcilesLostReplies at 130; TestLookupAndCancelMapHonestly at 285; TestDecodePublishReceiptRefusesLookalikes at 323; TestDecodePRCreateReceiptRefusesLookalikes at 365; TestReceiptRejectsTrailingDocument at 561; TestPushBranchMovesExactlyItsTarget at 397; TestObserveTipParsesExactAdvertisement at 438.

## internal/host/tailnet/companion.go

Observed size: 730 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: validateCompanionRecord at 100; inspectCompanion at 133; companionCLI at 169; current at 196; saveCurrent at 218; consumeRunKey at 246; prepareCompanionState at 390; createFreshCompanion at 413; enrollCompanion at 482; StartTailnet at 522; StopTailnet at 571; logoutAndStopCompanion at 627; queueProjectTailnetDisable at 640; applyCompanionIdleState at 673; ObserveProjectTailnet at 705.

## internal/host/terminal/factory_codex_linux.go

Observed size: 539 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: FactoryCodexReserve at 111; FactoryCodexStart at 265; FactoryCodexWait at 311; factoryCodexSetup at 180; factoryCodexStage at 201; factoryCodexStageHost at 233; factoryUnitState at 54; factoryActiveInvocation at 62; factoryRoleIDs at 88; FactoryCodexValidate at 350; FactoryCodexStop at 379; FactoryCodexFinish at 465.

## internal/host/terminal/muse_execution_linux.go

Observed size: 439 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Start at 29; reserveExecution at 87; deliverExecution at 108; controlExecution at 137; stageConfig at 238; populateConfig at 251; copyNestedConfig at 385; museHostEnvironment at 414; stopExecution at 268; ValidateMuseBinding at 293; retireMount at 426.

## internal/host/terminal/muse_linux.go

Observed size: 436 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/host`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: resolveProject at 86; resolve at 161; projectAccount at 189; authorizedCaller at 231; RegisterNested at 265; registeredChild at 311; validateNested at 344; nestedNamespace at 359.

## internal/host/terminal_native_test.go

Observed size: 418 lines, including tests where embedded. Preserve the opt-in
native terminal marker/probe and journey assertions, rebinding its temporary
helper to the Rust host binary and actual native binding. The current fixture
constructs Go Daemon/Native/terminal.Service at 115-142 and cannot survive
executor deletion unchanged. Keep installed proof distinct from source tests.

- `internal/host/terminal_native_test.go` — Existing native terminal escape/control sequence probe.
- `internal/host/terminal_boundary_native_test.go` — Installed actor/terminal boundary journey.

Evidence: TestTerminalProbeNativeControlSequences at 38; TestInstalledTerminalBoundary at 48.

## rust/soda-host/src/account.rs

Observed size: 764 lines, including tests where embedded. Keep the 254-line production account/key implementation; its distinction between provisioning key normalization and revision-checked own-account replacement is established. Split the 509-line embedded test body into existing account provisioning cases and own-account key preview/apply/drift cases. Move Mock and fixture builders with account_tests.rs and share those existing helpers within the test module only; do not introduce a new fixture service.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/account/mod.rs`
- `lib/host/src/account/tests.rs`
- `lib/host/src/account/access_keys_tests.rs`

Current defining fit at f7: production1–254 stays `account/mod.rs`; cfg(test)255 and module256 define one test root. `account/tests.rs` owns the existing ED/Mock/config/deadline/ID/inspection fixtures257–337 and whole cases with attributes338–597. `access_keys_tests.rs` starts at the actual next #[test]598 and owns complete cases599–763; original764 closes the common test root. Share the original fixtures through test-only sibling imports; do not copy Runtime, decoding or confirmation behavior. Provisioning normalization and revision-checked replacement retain their distinct contracts. Root independently inspected the real provisioning subject, cases and defining seams; the complete key scenarios remain the primary reviewer's source duties, with no execution proof.

## rust/soda-host/src/domain.rs

Observed size: 709 lines, including tests where embedded. Separate already-existing pure domain records by concern inside the same host crate: profile/Create; account/AccessKeys; OS release/observation; and core project validators plus Environment/Connection in domain.rs. Move each type with its decoder/encoder/spec constants, retaining pure no-I/O validation and wire field order. The 532 lines before tests and the 177-line embedded test module are distinct; no new canonical DTOs should be introduced.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/domain/mod.rs`
- `lib/host/src/domain/profile.rs`
- `lib/host/src/domain/account.rs`
- `lib/host/src/domain/os.rs`
- `lib/host/src/domain/tests.rs`

Evidence: 1-6: pure mirrors of the established project Unix contract; 12-93,262-325: Unicode/ID/login/image/container validation, Environment/Connection -> domain.rs; 95-261: Profile specs/validation/encoding and Create -> profile.rs; 327-408: OsRelease/OsObservation and OS validators -> domain_os.rs; 411-531: Account/AccessKeys/AccessKeyState and their typed specs -> domain_account.rs; 533-709: profile/domain/account/OS wire and validator tests -> domain_tests.rs.

Open detail: The Go project domain remains the established owner; this physical split preserves the current Rust wire mirror and does not authorize a second competing definition or complete the host port.

## rust/soda-host/src/json.rs

Historical source size: 1661 lines, including tests where embedded. The completed host-local split below records the value, scanner, string, number and binder owners. [JSON01](../../../research/library-reuse-investigation.md#json01) supersedes preservation or further extraction of those generic engines: the target is serde/serde_json with explicit strict-request and tolerant native-observation DTO/Visitor adapters. Keep caller-specific duplicate/null/ordered-alias/byte-array/depth policies and canonical output needs; historical scanner/error-text equivalence alone is not an admission requirement. Retain meaningful caller assertions and update obsolete parser oracles with the selected policy.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/json/mod.rs`
- `lib/host/src/json/scan.rs`
- `lib/host/src/json/string.rs`
- `lib/host/src/json/number.rs`
- `lib/host/src/json/specs.rs`
- `lib/host/src/json/bind.rs`
- `lib/host/src/json/strict_tests.rs`
- `lib/host/src/json/binding_tests.rs`

Evidence: 23-74,662-805,1242-1306: Value/Error, decode_strict/decode_tolerant, Value accessors and output/tolerant map encoding -> json.rs; 76-190,191-465: Parser structural methods and run_machine object/array/value stack handling -> json_scan.rs; extract quote_byte with lexical diagnostics if needed to keep this file bounded; 467-585: hex4/parse_string, including surrogate substitution -> json_string.rs; 586-660,806-815: parse_number/parse_go_int64/parse_go_uint32 -> json_number.rs; 817-953: Spec/Kind/Bound/BoundMap and field access -> json_specs.rs; 954-1241: type_error, byte binding, bind_value/match_spec/bind_struct/bind_root -> json_bind.rs; 1308-1554: strict shapes/duplicates/scanner/errors/depth matrices -> json_strict_tests.rs; 1555-1661: binder fixtures, surrogate/output escaping and tolerant semantics -> json_binding_tests.rs.

Current target: `3bf7e75b` removes scanner/string/binder/specs engines and their exclusive tests. `json/mod.rs` and `json/number.rs` now contain only Serde admission/formatting and actual scalar/byte policy; request/native DTOs remain with callers. Completed module paths above are provenance; do not create a new shared minimal parser or keep a second engine to preserve historical diagnostics. The JSON01 profiles, not a blanket serde derive, preserve the actual distinct contracts.

## rust/soda-host/src/preparation.rs

Observed size: 1209 lines, including tests where embedded. This is the pure preparation record/validation surface, not the executor. Group the existing validators and Preparation/Prepare identity in preparation.rs; RequirementAcceptance/AdminApproval in preparation_decisions.rs; ApprovedSetup/digest in preparation_setup.rs; resolved-tool, inspect/stop/hold and state wire in preparation_state.rs; FactoryCandidate in preparation_candidate.rs. Split 452 lines of tests into validation/digest/candidate and exact decoding/state-output cases. JSON01 `df4b2cf5` replaced their Spec tables with concrete Serde visitors at these same record owners. Preserve the landed module responsibilities; signed raw-token, byte and nullable list adapters carry the distinct admission policies.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/preparation/mod.rs`
- `lib/host/src/preparation/decisions.rs`
- `lib/host/src/preparation/setup.rs`
- `lib/host/src/preparation/state.rs`
- `lib/host/src/preparation/candidate.rs`
- `lib/host/src/preparation/validation_tests.rs`
- `lib/host/src/preparation/wire_tests.rs`

Evidence: 38-101,207-315,390-437: role/phase/ID validators, Preparation and Prepare -> preparation.rs; 102-206: RequirementAcceptance/AdminApproval and validation -> preparation_decisions.rs; 316-388: ApprovedSetup allowed files/bundle and setup_digest_of -> preparation_setup.rs; 438-698: ResolvedTool, PrepareState/Inspect/Stop, HoldState/PrepareHold -> preparation_state.rs; 699-755: FactoryCandidate validation/from_value/from_map -> preparation_candidate.rs; 756-1044: current record fixtures and validator/setup-digest/candidate cases -> preparation_validation_tests.rs; 1045-1209: prep_json/prepare_json, exact decoding messages and omitempty state output -> preparation_wire_tests.rs.

Current JSON01 checkpoint `df4b2cf5` also transfers terminal stream/inspection, factory identity/receipt/request/run, and native preparation helper/state records. Their typed adapters retain alias timing, strict request preflight, output order, protected snapshots and original digest inputs. Final host engine retirement completed in `3bf7e75b`; the historical scanner allocation above is not a new split task. The remaining JSON duties are strict admission, caller DTO policy and required producer formatting over Serde.

## Current P06/P12 native preparation defining seams

Current P06 LauncherEvidence321–327 and quote_bytes_base64328–332 stay prepare/state.rs; its helper callers343/494 import one definition. role_exec/must_role_exec538–557 stay prepare/source.rs; launcher verification638–732 stays prepare/tools.rs. Existing Runtime/Executor/DTO owners remain single definitions.
P12 HoldState/PrepareHold621–698 from preparation.rs stay preparation/state.rs; mixed address_and_hold_validation975–1000 stays preparation/validation_tests.rs. Runtime hold859–881 from prepare.rs stays prepare/mod.rs orchestration; the whole mixed inspect_stop_hold_paths1665–1789 stays prepare/execution_tests.rs with its attribute and scripted Mock subject. Do not duplicate inspection/stop assertions into a separate hold test leaf. project_operations/hold.rs735–787 is a distinct already-selected integration subject.
These same-host concern/import allocations have underlying A/B source challenge; no native quiescence, concurrent restore or installed correctness proof follows.

## rust/soda-host/src/prepare.rs

Observed size: 1955 lines, including tests where embedded. Split 1,113 production lines at the existing execution phases: orchestrating prepare/inspect/stop/hold; clean role paths and ID maps; fixed helper exchange/approval; source clone/head binding and role execution; launcher/tool verification and recorded evidence; protected candidate snapshot reads; and native state decode. Keep the prepare sequence and its re-observation on error together and preserve fixed argv/body bytes. The 840-line test module has existing pure/path helpers, full preparation-chain cases, and protected candidate cases suitable for separate concern test files.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/prepare/mod.rs`
- `lib/host/src/prepare/paths.rs`
- `lib/host/src/prepare/helper.rs`
- `lib/host/src/prepare/source.rs`
- `lib/host/src/prepare/tools.rs`
- `lib/host/src/prepare/candidate.rs`
- `lib/host/src/prepare/state.rs`
- `lib/host/src/prepare/tests.rs`
- `lib/host/src/prepare/execution_tests.rs`
- `lib/host/src/prepare/candidate_tests.rs`

Evidence: 21-150: path_clean/path_join/prepare_id_map/preparation_paths/single_line/tool path policy -> prepare_paths.rs; 151-348,1055-1113: inspection Spec tables, LauncherEvidence, map_preparation_state -> prepare_state.rs; approval/saved-request tables move with their actual decoder instead; 351-425,484-537: factory_helper/prepare_container/approve_preparation and files encoding -> prepare_helper.rs; 426-483,820-881: inspect/prepare orchestration/stop/hold -> prepare.rs; 538-637: role_exec/must_role_exec/clone_preparation_source/confirm_preparation_head -> prepare_source.rs; 638-819: verify_launcher_environment/resolve_preparation_tools/record_preparation_tools -> prepare_tools.rs; 882-1054: prepare_candidate/candidate_protected_file/candidate_approved_files -> prepare_candidate.rs; 1114-1456: existing Mock/config/preparation/setup fixtures and pure/native admission cases -> prepare_tests.rs; 1457-1788: full-chain/error/blocked/inspect/stop/hold cases -> prepare_execution_tests.rs; 1789-1955: candidate_reuses_ready_source/rejects_bad_snapshots -> prepare_candidate_tests.rs.

## rust/soda-host/src/project.rs

Current P02 defining fit at f7 supersedes the historical selectors below: `project/profile.rs` owns PROFILE_INSPECT_FORMAT28–29, the whole resolve_profile comment285/function286–360 and apply_creation_profile1033–1062. `project/os.rs` owns observation comments647–648/function649–697 and the whole parser/helper closure1081–1275. `project/confirmation.rs` owns confirm_resolve_profile940–946 and OS confirmation/helper closure948–983, alongside its other established confirmations. Keep common go_arch comment807/function808–816 in `project/mod.rs`, importing that one definition in profile/create/confirmation. INSPECTION_SPECS30–73 and PROJECT_INSPECT_FORMAT75 retain one inspect owner and current crate-visible reexports for actual preparation/terminal consumers. Runtime270–273 and its common podman275–283 retain one mod owner. Existing test root1277–1278 keeps its Mock/config/profile/image/container fixtures once in `project/tests.rs`; the mixed address/profile/OS/key case with attribute2052 and complete function2053–2203 stays whole in `project/confirmation_tests.rs`, using bounded test-only imports. Root independently read these real source units and assertions. This target-fit challenge does not establish native profile/OS/provisioning proof.

Observed size: 1383 lines, including tests where embedded. Separate Native/Executor and fixed process deadline/output handling from project operations. Keep creation/network setup/start/readiness in project_create.rs; immutable profile resolution in project_profile.rs; container observation/isolation in project_inspect.rs; OS-release reading and its shlex parser in project_os.rs; host-key reading/fingerprint in project_connection.rs. project.rs retains Config/Runtime and the common Podman entry. Existing tests occupy 396 lines and can remain one concern test module initially. Do not combine domain types with privileged execution.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/project/mod.rs`
- `lib/host/src/project/executor.rs`
- `lib/host/src/project/profile.rs`
- `lib/host/src/project/create.rs`
- `lib/host/src/project/inspect.rs`
- `lib/host/src/project/os.rs`
- `lib/host/src/project/connection.rs`
- `lib/host/src/project/tests.rs`

Evidence: 78-119,215-230: Config/Runtime/podman -> project.rs; 120-213: Native Executor, bounded subprocess I/O/deadlines and exit_text -> project_executor.rs; 231-306,743-777: resolve_profile/apply_creation_profile -> project_profile.rs; 408-539,682-711: create/create_container/start_created/wait_project_ready, go_arch/go_dir -> project_create.rs; 307-407,540-593,712-742,778-790: inspect/project_container, label decoding and ID-map isolation -> project_inspect.rs; 594-644,791-986: observe_os, split_lines/shlex_posix and parse_os_release -> project_os.rs; 645-681: connection host key/fingerprint -> project_connection.rs; 987-1383: mock runtime/profile/create/inspect/OS/connection cases -> project_tests.rs.

Current package fact at f7: soda-host/Cargo.toml already defines both the soda_host library and soda-host binary. Retain this existing host process and its real library callers; proposed file splits do not establish installed daemon cutover or native qualification.

## rust/soda-host/src/ssh.rs

Historical source size: 1028 lines, including tests where embedded. The completed SSH split below is superseded as an engine-preservation target by [CF-03](../../../research/library-reuse-investigation.md#cf-03), with [CF-02](../../../research/library-reuse-investigation.md#cf-02) curve admission and [CF-04](../../../research/library-reuse-investigation.md#cf-04) Base64 profiles. ssh-key owns public-key/certificate/mpint format parsing and encoding; RustCrypto owns NIST point validation. Retain a small host adapter for the actual algorithm/options policy, exact uncompressed curve gate, canonical managed-key output and fingerprints. Format parsing does not establish certificate trust or authorize new algorithms. Deliberately resolve malformed/noncanonical input admission at the caller; do not retain general SSH machinery solely for predecessor quirks.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/ssh/mod.rs`
- `lib/host/src/ssh/base64.rs`
- `lib/host/src/ssh/mpint.rs`
- `lib/host/src/ssh/material.rs`
- `lib/host/src/ssh/certificate.rs`
- `lib/host/src/ssh/tests.rs`

Current target: `ssh/base64.rs`, `ssh/mpint.rs`, `ssh/material.rs` and
`ssh/certificate.rs` are existing custom-engine locations, not additional
extraction tasks. Remove or reduce them to the selected admission/output
adapters after caller transfer; keep account preview/apply and filesystem
publication at their existing owners. SHA-256 compression disappears through
[CF-01](../../../research/library-reuse-investigation.md#cf-01), while exact
canonical bytes, lower-case presentation and revision recipes remain local.

Evidence: 26-64,756-907: trim_ws, parse_public_key/parse_key_text/scan_options/parse_authorized_key and public marshal/fingerprint -> ssh.rs; 65-229: strict standard and Go-specific base64 decoders/encoders -> ssh_base64.rs; 231-412: read_string/read_u32/read_u64/put_string, Mpint parsing/comparison/marshalling -> ssh_mpint.rs; 436-471,489-594,698-755: ParsedKey/KeyMaterial, curve_for, ordinary key parsing/marshalling -> ssh_material.rs; 413-435,472-488,595-697: certificate algorithm names/material, parse_tuples/parse_cert/marshal_tuples -> ssh_certificate.rs; certificate arm of marshal_fields stays with certificate responsibility; 908-1028: base64, canonical/authorized-key and fingerprint fixtures -> ssh_tests.rs.

## rust/soda-host/src/tailnet_companion.rs

Current `f7e9cf9d` retirement: the private `#[allow(dead_code)] podman` method at 394–398 has no caller in the actual source census. Exclude precisely that attribute/method from the proposed execute module. Retain the enclosing Companion, `runtime_command` from 400, live `runtime_podman` and their secret-safe error handling. The production and independent reviewers challenged this scope in [N06](../reviews/N06.md); no whole-file retirement or additional process is implied.

Observed size: 2188 lines, including tests where embedded. The 1,111-line production companion module combines incarnation proof, process execution, state reconciliation/start, enrollment key consumption, confirmed logout/stop, and user-facing observation. Split these existing phases within the host crate while retaining one Companion executor and TailnetControl authority boundary. Keep repeated runtime/resolver/namespace checks at their current lifecycle points. Its 1,075-line test module needs the same concern grouping: existing fixture/mock scaffolding, identity/record cases, lifecycle cases, and view/action cases.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/companion/mod.rs`
- `lib/host/src/tailnet/companion/identity.rs`
- `lib/host/src/tailnet/companion/execute.rs`
- `lib/host/src/tailnet/companion/start.rs`
- `lib/host/src/tailnet/companion/enroll.rs`
- `lib/host/src/tailnet/companion/stop.rs`
- `lib/host/src/tailnet/companion/view.rs`
- `lib/host/src/tailnet/companion/tests.rs`
- `lib/host/src/tailnet/companion/identity_tests.rs`
- `lib/host/src/tailnet/companion/lifecycle_tests.rs`
- `lib/host/src/tailnet/companion/view_tests.rs`

Evidence: 33-89,133-360: CompanionRecord decode/recipe/exec validation, proc namespace identity and resolver match -> tailnet_companion_identity.rs; 90-132,370-393: TailnetControl/Companion, staged errors and freshness/idle helpers -> tailnet_companion.rs; 394-555,843-869: bounded native runtime execution/inspect/CLI plus wait_tailnet -> tailnet_companion_execute.rs; 556-624,671-773,811-842: start gating, admit run, prepare state/container activation/node wait, final recheck and start_tailnet -> tailnet_companion_start.rs; 774-810,938-980: enroll_companion/ensure_companion_enrolled and consume_run_key -> tailnet_companion_enroll.rs; 625-670,870-937: previous-run retirement/reconciliation and confirmed stop/logout/resolver restoration -> tailnet_companion_stop.rs; 981-1111: disable/start queue, stopped-project and companion/project status observation -> tailnet_companion_view.rs; 1112-1308: existing temp/run/view/binding/request helpers and MockExec/MockTailnet -> tailnet_companion_tests.rs; 1309-1605: record/recipe/proc/resolver/scanner/freshness identity cases -> tailnet_companion_identity_tests.rs; 1606-1955: native execution/start/wait/enrollment/stop/reconciliation cases -> tailnet_companion_lifecycle_tests.rs; 1956-2188: queued intent, stopped/observed status and project-status maps -> tailnet_companion_view_tests.rs.

## rust/soda-host/src/tailnet_domain.rs

Historical source size: 1538 lines, including tests where embedded. Completed DTO/status/native-observation concern modules remain. [N8](../../../research/library-reuse-investigation.md#n8) supersedes the IP grammar/printing engine; retain std IP types plus small DNS/zone/mask and Tailnet global-unicast policy. [N9](../../../research/library-reuse-investigation.md#n9) supersedes calendar arithmetic/RFC3339 parsing; retain the strict lexical/deadline and zero-time adapter. [JSON01](../../../research/library-reuse-investigation.md#json01) supplies native status/preferences DTO profiles. Keep project binding/status decisions pure, original signed text unchanged and existing concern tests around actual library-backed subjects; no runtime or SQL authority moves into these files.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/domain/mod.rs`
- `lib/host/src/tailnet/domain/time.rs`
- `lib/host/src/tailnet/domain/native.rs`
- `lib/host/src/tailnet/domain/addresses.rs`
- `lib/host/src/tailnet/domain/status.rs`
- `lib/host/src/tailnet/domain/node_tests.rs`
- `lib/host/src/tailnet/domain/status_tests.rs`
- `lib/host/src/tailnet/domain/address_tests.rs`

Evidence: 1-84: errors, reader types, RunTarget/RunBinding/ProjectRequest/ProjectView and ID validators -> tailnet_domain.rs; 85-219: time math/parse_rfc3339_nano/go_escape -> tailnet_time.rs; 220-454: case folding/native_object/field binders and SelfPeer/NativeStatus/NativePrefs decoding -> tailnet_native.rs; 455-775: DNS/address parsers/canonicalization/global-unicast validation and peer resolution -> tailnet_addresses.rs; 776-892: parse_project_status/validate_project_preferences/match_project_self/binding/project_status/project_has_node -> tailnet_status.rs; 893-1026: binding/status/prefs fixtures and has_node vectors/state table -> tailnet_node_tests.rs; 1027-1294: connected/fresh/non-running/unsafe/malformed/casefold/limit project-status cases -> tailnet_status_tests.rs; 1295-1538: address/DNS/simple-fold/ID/RFC3339/escaping vectors -> tailnet_address_tests.rs.

## rust/soda-host/src/tailnet_files.rs

Observed size: 874 lines, including tests where embedded. Separate protected runtime-root/open/locking from current-run records/subdirectory preparation, ephemeral secret-file and companion-ID custody, and resolver inode/text/network-device admission. Keep file-descriptor identity and exclusive lock checks at existing mutation boundaries. Production is 566 lines; tests are 307 and can remain together because their fixtures exercise these linked filesystem boundaries.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/files/mod.rs`
- `lib/host/src/tailnet/files/run.rs`
- `lib/host/src/tailnet/files/keys.rs`
- `lib/host/src/tailnet/files/resolver.rs`
- `lib/host/src/tailnet/files/tests.rs`

Evidence: 36-225: Root/RunFiles structs, protected root/project ownership, open and exclusive lock -> tailnet_files.rs; 233-342: run subdirectories and RunFiles::current/save_current/prepare -> tailnet_run_files.rs; 343-496: key create/retire/pending-key inode handling and companion ID records -> tailnet_keys.rs; 497-566: resolver generated path/inode/text/device conflict checks -> tailnet_resolver.rs; 567-874: existing TempDir/run fixtures, locks, symlink/mode/key-inode/resolver/explicit-retry cases -> tailnet_files_tests.rs.

## rust/soda-host/src/tailnet_runtime.rs

Observed size: 1121 lines, including tests where embedded. Split the 687-line production runtime into native proc identity reading, run snapshot/record encoding plus identity digest, orchestration/companion recipe, and project container isolation/inspection. Keep snapshot rereads and exact process identity comparison before admitting a run. Extract the 433-line test portion into run/process/recipe cases and container-isolation/inspection cases; do not broaden supported isolation mappings.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/tailnet/runtime/mod.rs`
- `lib/host/src/tailnet/runtime/process.rs`
- `lib/host/src/tailnet/runtime/wire.rs`
- `lib/host/src/tailnet/runtime/project.rs`
- `lib/host/src/tailnet/runtime/tests.rs`
- `lib/host/src/tailnet/runtime/project_tests.rs`

Evidence: 23-67,386-549: ProjectRun/runtime declarations, actual readers, admit/confirm/inspect/project_run and companion_create_args/recheck -> tailnet_runtime.rs; 74-210: proc stat/start/id-map/namespace/boot readers and process_run_identity -> tailnet_process.rs; 211-385: RUN_INSPECT specs, run decode/encode/assemble_project_run digest binding -> tailnet_run_wire.rs; 550-687: id_map/project_isolation/inspect_project/project_container/project_running -> tailnet_project.rs; 688-1014: synthetic proc readers, exact recipe/run wire/admission cases -> tailnet_runtime_tests.rs; 1015-1121: inspection executor/JSON fixture and argv/isolation gates -> tailnet_project_tests.rs.

## rust/soda-muse-maintain/src/main.rs

Observed size: 4075 lines, including tests where embedded. The file contains 2,892 lines before tests plus a 1,183-line embedded test module, spanning every maintenance phase. Preserve one ordered maintain operation: admit pinned public tool sources, bind exact project incarnation, optionally ensure system bus/stage tools, prepare public directory, then attach the restricted mount. Retain the completed flags, host-config/release DTO, native-file, process, project, staging and interface-admission concern owners. Generic JSON scanners, SHA rounds, IP parsers and tar-header assembly follow JSON01, CF-01, N8 and CF-07 library adoption rather than further extraction. Retain fixed scripts with their existing phase. Test files follow existing groups and shared TestDir/synthetic archive helpers move to test_support.rs only. Keep root-owned pinned-tool FD admission, exact digest/declared-size reads, deterministic archive metadata, bounded child/pipe writes and incarnation rechecks at the real consumers. Shared mechanics use the selected filesystem/process adapters; the old Go path/errno/quote emulation is historical, not a reason for another package.

- `cmd/soda-muse-maintain/src/main.rs`
- `cmd/soda-muse-maintain/src/options.rs`
- `cmd/soda-muse-maintain/src/config.rs`
- `cmd/soda-muse-maintain/src/config_wire.rs`
- `cmd/soda-muse-maintain/src/config_validation.rs`
- `cmd/soda-muse-maintain/src/filesystem.rs`
- `cmd/soda-muse-maintain/src/release.rs`
- `cmd/soda-muse-maintain/src/release_wire.rs`
- `cmd/soda-muse-maintain/src/release_validation.rs`
- `cmd/soda-muse-maintain/src/network.rs`
- `cmd/soda-muse-maintain/src/sha256.rs`
- `cmd/soda-muse-maintain/src/project.rs`
- `cmd/soda-muse-maintain/src/command.rs`
- `cmd/soda-muse-maintain/src/stage.rs`
- `cmd/soda-muse-maintain/src/archive.rs`
- `cmd/soda-muse-maintain/src/interface.rs`
- `cmd/soda-muse-maintain/src/interface_admission.rs`
- `cmd/soda-muse-maintain/src/test_support.rs`
- `cmd/soda-muse-maintain/src/options_tests.rs`
- `cmd/soda-muse-maintain/src/config_tests.rs`
- `cmd/soda-muse-maintain/src/network_tests.rs`
- `cmd/soda-muse-maintain/src/release_tests.rs`
- `cmd/soda-muse-maintain/src/project_tests.rs`
- `cmd/soda-muse-maintain/src/filesystem_tests.rs`
- `cmd/soda-muse-maintain/src/interface_tests.rs`
- `cmd/soda-muse-maintain/src/archive_tests.rs`

Evidence: 59-212: Options/default_tools/parse/usage/bool/project-ID flag policy -> options.rs; 66-91 and 268-279 -> main.rs run/maintain; 281-338: go_clean/go_base/go_dir and 366-513: path_error/go_errno/last_errno -> filesystem.rs with real pinned-file operations; 216-267: go_quoted/nonprint -> config_wire.rs with existing diagnostic formatting; 339-365: Config/load_config -> config.rs; 514-725: host decode/field slot/string setters with saved error semantics -> config_wire.rs; 1690-1790: current runtime validation order -> config_validation.rs; 726-792,904-1073,1074-1100: JsonParser basic methods, literal/number/skip scanning and UTF8 unit decoding -> json.rs; 793-903: parse_string/surrogate_tail -> json_string.rs; 1101-1230: apply_release_images, ReleasePayload/Image and confined payload file admission -> release.rs; 1231-1366,1481-1689: typed release parsing/image/string-list binding -> release_wire.rs; 1367-1480: payload/digest/architecture/reference validation -> release_validation.rs; 1791-1993: observed netip prefix/address/IPv4/IPv6 error-parity routines -> network.rs; 1994-2095: Tool FD ownership/load_tools/open_tool/trusted_tool/verify_native -> filesystem.rs; 2096-2238: hex/streaming Sha256 -> sha256.rs; 2239-2246,2347-2532: Observation, inspect/validate/decode/slot/wait/confirm project incarnation -> project.rs; 2247-2346: podman/podman_streamed/wait_output with O_CLOEXEC, drain and deadline precedence -> command.rs; 14-55,2533-2573: BUS/INSTALL/DESTINATIONS and ensure_system_bus/stage_tools -> stage.rs; 2574-2683: feed_archive/emit_archive/tar_header and existing deadline/CopyN/USTAR byte contract -> archive.rs; 22,2684-2841: prepare_interface/FdGuard/open_tree/mount_setattr/attach_project_mount; exact incarnation recheck before unshare/setns -> interface.rs; 2842-2892: public_socket_directory/validate_public_socket/validate_interface_directory -> interface_admission.rs; 2893-2937,3451-3498,3896-3946: existing TestDir and synthetic release/archive fixture helpers -> test_support.rs; keep release-specific helpers with release_tests if simpler; 2975-2991: quote vectors -> config_tests.rs; 3027-3041 and 2992-3026: path/errno vectors -> filesystem_tests.rs; 3042-3119: flags -> options_tests.rs; 3120-3245,3616-3665: config/error-order vectors -> config_tests.rs; 3246-3450: netip prefix/address matrix -> network_tests.rs; 3499-3615: payload admission/image conflict vectors -> release_tests.rs; 3666-3762: strict observation/isolation vectors -> project_tests.rs; 2938-2974,3763-3819: streaming SHA and root-owned pinned tool admission -> filesystem_tests.rs; 3820-3871: public launch socket/directory protection -> interface_tests.rs; 3872-3895,3947-4075: USTAR/install replacement/streaming/wait-output/short-read cases -> archive_tests.rs.

Current target: [JSON01](../../../research/library-reuse-investigation.md#json01) completed the host-config/release/observation caller transfer in `bf88640b`. Local typed Serde visitors retain their distinct admission and stream policies; `json.rs` and `json_string.rs` were deleted with their last callers. [CF-01](../../../research/library-reuse-investigation.md#cf-01) is complete in `52eee7ee`: sha2 owns hashing, with tool-FD/read custody and recipes retained. [N8](../../../research/library-reuse-investigation.md#n8) and [CF-07](../../../research/library-reuse-investigation.md#cf-07) replace address and archive mechanics while retaining size, metadata and deadline policies. The engine paths/ranges above remain historical allocation evidence, not pending extraction instructions.

## rust/soda-muse/src/main.rs

Observed size: 1637 lines, including tests where embedded. Split 1,233 production lines along the existing launcher/helper responsibilities. Keep command/native-action selection in main; shell input/validation in shell; seqpacket+FD passing/signal/resize forwarding in launch; credential-view admission/native exec/environment in execution; account marker/passwd checks in account; config copy/read in config; pinned-version/private clean subprocess in runtime. Path and diagnostic emulation, scalar JSON scanning and Base64 algorithms are superseded by their selected library/std adapters; retain only actual path admission and request/exit DTO/output policy. Split the 401-line test module into shell/wire/paths, FD/signal transport, and config/account/native-dispatch tests. Preserve one launch-only socket and broker confirmation gates.

- `cmd/soda-muse/src/main.rs`
- `cmd/soda-muse/src/shell.rs`
- `cmd/soda-muse/src/launch.rs`
- `cmd/soda-muse/src/execution.rs`
- `cmd/soda-muse/src/account.rs`
- `cmd/soda-muse/src/config.rs`
- `cmd/soda-muse/src/runtime.rs`
- `cmd/soda-muse/src/paths.rs`
- `cmd/soda-muse/src/launch_wire.rs`
- `cmd/soda-muse/src/launch_json.rs`
- `cmd/soda-muse/src/shell_tests.rs`
- `cmd/soda-muse/src/launch_tests.rs`
- `cmd/soda-muse/src/config_tests.rs`

Evidence: 14-79: main/run/native_action/metadata_action -> main.rs; 80-187: ShellRequest, shell_request/validation and path/text/argument admission -> shell.rs; 242-405: seqpacket_connect/send_with_fds/launch_shell/signal controls -> launch.rs; 406-547: execute/await_admission/execution_state/private runtime directory and fixed environment -> execution.rs; 548-631,661-664: Go path/error/rune quoting -> paths.rs; 632-660,665-734: account_for/account_entry/account_node/lookup_user -> account.rs; 735-831: check_runtime/make_private_dir/run_pristine -> runtime.rs; 832-954: copy_config tree/filter/file and read_config settings/trust view -> config.rs; 188-241,955-1079: shell request serialization/base64/json_string/LaunchExit object decode -> launch_wire.rs; 1080-1233: parse_json_string/parse_json_integer/skip_json_value -> launch_json.rs; 1234-1403: ShellRequest fixture/wire/validation/exit/base64/path/quote/environment cases -> shell_tests.rs; 1469-1617: FD passing/exit and signal forwarding -> launch_tests.rs; 1404-1468,1618-1637: config copy/read/account safety/native action errors -> config_tests.rs.

The completed Muse concern allocation above retains one packet/control owner.
[JSON01](../../../research/library-reuse-investigation.md#json01) and
[CF-04](../../../research/library-reuse-investigation.md#cf-04) completed the
launch JSON and Base64 transfers in `26493cf2`. `launch_json.rs` retains the
Go-compatible Serde formatter; `launch_wire.rs` retains typed request/exit
records and library Base64 use. The scalar/skip and Base64 engines are deleted.
[N10](../../../research/library-reuse-investigation.md#n10) supplies typed
SCM_RIGHTS/descriptor mechanics while preserving exact stdio count, pinned peer
identity, signal/resize forwarding and cleanup. The packet path remains separate
from HTTP. Historical extraction does not require another transport state machine.

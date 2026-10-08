# Verification and support

L11/N13 delegates valid evidence URL interpretation to url setters, omits malformed/non-UTF8 tokens and preserves the completed L01 streaming bounds and failure publication. Acceptance timestamp validation/formatting uses `lib/wire-time`; selected calendar and URL grammars are retired, while evidence/pump lifecycle remains Soda-owned.

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

Pending generic-engine splits below yield to the existing packets' scoped
[library adoption](../library-adoption.md#execution-packets). Completed moves,
historical source counts and domain/support owners remain recorded. Prioritize
the named correctness repairs and dependency admission before adoption; use
existing focused checks rather than another qualification framework.

CFG01 remains a precise configuration-selection gate: the native Forgejo
effective server.APP_DATA_PATH, including supported INI inputs and environment
precedence, must match without guessing. A Python ConfigParser oracle or the
locale catalog's values-ignored tokenizer cannot establish that contract.
Defer this parser selection only; it does not block unrelated JSON or CLI work.
See [finding allocation and readiness](../library-adoption.md#readiness-gates).

## internal/acceptance/developer_access_test.go

Observed size: 447 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/acceptance/developer_access_test.go` — Existing access input, user, UID, subnet and SSH options cases.
- `internal/acceptance/developer_access_request_test.go` — Private request/support file loading and refusal cases.
- `internal/acceptance/developer_access_journey_test.go` — Bound transport fixture and native access journey.

Evidence: TestValidateAccessUser at 58; TestAccessSSHOptions at 140; TestLoadAccessRequest at 276; TestRunDeveloperAccessValidation at 181; stubAccessTransport at 319; TestRunDeveloperAccessEndToEnd at 368.

## internal/acceptance/evidence.go

Observed size: 466 lines, including tests where embedded. Retire the obsolete
Evidence/Execute/Remote/Command closure and its tests. Its remaining production
use is PrivateFile, called by installed probes; preserve that real operation
in probe input support before removing the rest. The current caller census
retains Go StartCommand/Process in this same acceptance package for live
bounded installed probes; only the uncalled StartProcess adapter retires.
No new process package or replacement evidence facade is proposed.

- `internal/acceptance/installed.go` — Existing restricted regular-file input operation folded into its actual private-input caller; retain absolute-path/regular/restricted-mode and effective min(caller cap, 1 MiB) admission on the same opened descriptor; refuse final links/FIFOs, bound reads before growth, check read/close errors and retain existing probe tests.

Evidence: CreateEvidence at 30; WriteJSON at 188; PublishObservation at 241; Hashes at 217; scanEvidenceBytes at 277; CheckSecrets at 327; PrivateFile at 454; RedactError at 361; redactingWriter at 370; redactPendingSecrets at 409.

Disposition: Rust acceptance is the retained harness. Remove the obsolete
Go release-build hashing import with Evidence; do not keep legacy machinery
to retain its unit tests.

## internal/acceptance/personal_git.go

Observed size: 487 lines, including tests where embedded. Keep the Go probe's
orchestration, exported contracts and execution order. The embedded remote
Python program is an implementation predecessor: replace it with the Rust
user-scoped payload in the existing acceptance package, keeping the actual
SSH login, passphrase custody and key/agent lifetime. Merely moving its string
to personal_git_keys.go would not complete the Python cutover.

- `internal/acceptance/personal_git.go` — Explicit prepare/exercise admission and orchestration.
- `internal/acceptance/personal_git_keys.go` — Passphrase custody and exported public-key checks.
- `internal/acceptance/personal_git_transport.go` — Target decoding, SSH invocation and exact Git URL checks.
- `internal/acceptance/personal_git_exercise.go` — Per-user clone/commit/push exercise and outcome records.
- `tools/acceptance/src/personal_git.rs` — Native replacement for the existing remote key/agent operation, called through the existing ephemeral acceptance payload; no new installed service.

Evidence: RunPersonalGit at 451; runPersonalGit at 458; preparePassfile at 176; gitKeyUser at 213; fetchExportedKey at 238; loadGitTarget at 82; gitSSHBase at 126; validateGitURL at 267; exerciseCommand at 316; exerciseUser at 381; writeGitOutcomes at 405.

Open detail: Update the real Go SSH invocation, Rust remote dispatch and
personal_git_test.go together. Keep secret inputs off argv and output; target
payload binding and ephemeral staging must follow the existing acceptance
delivery rather than requiring a target compiler or installing another agent.

## rust/soda-acceptance/src/command.rs

Observed size: 550 lines, including tests where embedded. Retain pinned SSH argument construction and separate execution/evidence outcomes in the owners already extracted. PROC01 consolidates only equal std-backed bounded capture/drain mechanics; RED01 retains safe writer finalization. SSH pins, secret stdin, native cleanup and incomplete-pump custody stay with their callers. Keep cfg(test) tests private and attached to the real subject.

- `tools/acceptance/src/command/mod.rs`
- `tools/acceptance/src/command/ssh.rs`
- `tools/acceptance/src/command/execute.rs`
- `tools/acceptance/src/command/tests.rs`

Evidence: rust/soda-acceptance/src/command.rs:72-230 Remote, decode_remote, Remote::args and Remote::command; rust/soda-acceptance/src/command.rs:297-399 execute; rust/soda-acceptance/src/command.rs:402-550 tests for quoting, SSH pins, cancellation and outcome separation.

## rust/soda-acceptance/src/coreos.rs

Observed size: 503 lines, including tests where embedded. Keep the connected CoreOS resolution closure together; its bounded curl transport, stable stream and registry confirmation are one VM-base resolution operation. Extract the six existing unit tests. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/coreos.rs`
- `tools/acceptance/src/coreos/tests.rs`

Evidence: rust/soda-acceptance/src/coreos.rs:117-203 fetch_capped; rust/soda-acceptance/src/coreos.rs:261-284 resolve_stream_build; rust/soda-acceptance/src/coreos.rs:304-400 resolve_registry_digest and resolve_qemu; rust/soda-acceptance/src/coreos.rs:404-503 bounded-source and document tests.

RED01's immediate caller repair closes both redacting writers after ended
pump custody and propagates pump/close failures before consuming curl metadata.
The unterminated write-out line currently remains pending. This repair is
separate from later matcher adoption; native curl, HTTPS/redirect policy and
CoreOS resolution authority remain with this operation.

## rust/soda-acceptance/src/driver.rs

Observed size: 1047 lines, including tests where embedded. Retain private input collection, action dispatch, observation finalization and run in their extracted owners. CLI03 replaces the general flag/Go-duration engine with the admitted CLI/duration libraries and a small positive-timeout adapter. Preserve default/bounds, repeated files, bare/equals booleans, literal command tails and validation before evidence or side effects. Explicitly document the selected grammar/precision changes; parser overflow repair and real Phase deadlines precede a claim of bounded execution. Keep private cfg(test) descendants; dependency/toolchain admission follows the library chapter.

- `tools/acceptance/src/driver/mod.rs`
- `tools/acceptance/src/driver/options.rs`
- `tools/acceptance/src/driver/inputs.rs`
- `tools/acceptance/src/driver/actions.rs`
- `tools/acceptance/src/driver/finalization.rs`
- `tools/acceptance/src/driver/tests.rs`

Evidence: rust/soda-acceptance/src/driver.rs:62-392 parse_duration, parse_flags and parse_run_options; rust/soda-acceptance/src/driver.rs:393-434 secret and VM input collection; rust/soda-acceptance/src/driver.rs:489-715 execute_exec_or_native, execute_vm and execute_action; rust/soda-acceptance/src/driver.rs:717-830 observation finalization and run; rust/soda-acceptance/src/driver.rs:861-1047 duration, admission and cancelled-execution tests.

## rust/soda-acceptance/src/evidence.rs

Observed size: 909 lines at the historical decomposition baseline, including tests where embedded. Retain current confined evidence creation/publication and its outcome/custody state. L16's Luna medium consideration defers RED01 matcher adoption; retain current matching until aggregate input custody (L16.G), useful simplification and construction/streaming admission justify a later cutover. Completed L01 finalization and transformed-pattern/output/pending/tee bounds remain distinct from L16.G's open collection bound; L11's typed URL sanitizer is already adopted. Preserve longest-at-earliest-position semantics, escaped variants, sticky errors, binary bytes, final fail-closed leak scans and exclusive publication. A later library adapter uses a buffered whole-window search and withheld suffix, not LeftmostLongest overlapping-search APIs. Keep real private cfg(test) policy checks; sequential placeholder-rewrite quirks do not require a second engine. [Current decision and checks](../library-adoption.md#l16-evidence-matching).

- `tools/acceptance/src/evidence/mod.rs`
- `tools/acceptance/src/evidence/store.rs`
- `tools/acceptance/src/evidence/redaction.rs`
- `tools/acceptance/src/evidence/tests.rs`

Evidence: rust/soda-acceptance/src/evidence.rs:34-271 create_evidence and Evidence methods; rust/soda-acceptance/src/evidence.rs:341-368 scan_evidence_bytes; rust/soda-acceptance/src/evidence.rs:369-646 URL sanitization and RedactingWriter; rust/soda-acceptance/src/evidence.rs:648-909 retention, split-secret and byte-fidelity tests.

## rust/soda-acceptance/src/files.rs

Observed size: 696 lines, including tests where embedded. FS01 replaces repeated raw FD/syscall mechanics with the admitted typed rooted-file adapter, retaining one OwnedDir policy owner. Fix same-FD private-input admission and cap+1 reads before TMP01 adopts temporary allocation convenience. Preserve descriptor identity, symlink/type/ownership/mode/link rules, explicit handoff and exact-object cleanup; library Drop alone does not establish native cleanup. Keep private cfg(test) cases with the real file subject.

- `tools/acceptance/src/files/mod.rs`
- `tools/acceptance/src/files/owned_directory.rs`
- `tools/acceptance/src/files/inputs.rs`
- `tools/acceptance/src/files/temporary.rs`
- `tools/acceptance/src/files/tests.rs`

Evidence: rust/soda-acceptance/src/files.rs:111-370 OwnedDir and its fd-relative operations; rust/soda-acceptance/src/files.rs:372-437 private_file, hash_file and hash_at; rust/soda-acceptance/src/files.rs:438-522 fresh_directory, private_destination, TempDir and write_new; rust/soda-acceptance/src/files.rs:525-696 filesystem confinement and input tests.

## rust/soda-acceptance/src/host_probes.rs

Observed size: 939 lines, including tests where embedded. Retain the distinct installed probe commands and observation authority in their extracted modules. JSON01 replaces generic JSON emission; PROC01 consolidates equal bounded std process mechanics only. Keep private failure reporting and command-specific admission locally. These remain installed host observations, not another outside driver; retain private cfg(test) subjects and their proof limits.

- `tools/acceptance/src/host_probes/mod.rs`
- `tools/acceptance/src/host_probes/content.rs`
- `tools/acceptance/src/host_probes/listeners.rs`
- `tools/acceptance/src/host_probes/deployments.rs`
- `tools/acceptance/src/host_probes/tailnet.rs`
- `tools/acceptance/src/host_probes/forgejo.rs`
- `tools/acceptance/src/host_probes/tests.rs`

Evidence: rust/soda-acceptance/src/host_probes.rs:184-413 host_content and extension package checks; rust/soda-acceptance/src/host_probes.rs:415-678 listener parsing and host_listeners; rust/soda-acceptance/src/host_probes.rs:680-731 host_deployments and operator_tailscale; rust/soda-acceptance/src/host_probes.rs:733-819 Forgejo origins, tailnet and advertisement probes; rust/soda-acceptance/src/host_probes.rs:821-939 probe parser/summary tests.

## rust/soda-acceptance/src/jsonio.rs

Observed size: 511 lines, including tests where embedded. JSON01 replaces generic JSON parsing/emission through the acceptance record profile; N9 replaces calendar codecs with admitted time APIs and thin policy adapters. Retain actual bounded same-file reads, raw-byte hashes where consumed, record validation and current deterministic producer/consumer behavior. Historical exact-emission/date cases do not establish a requirement for every Go diagnostic or date quirk. Keep private cfg(test) cases attached to the replacement subject.

- `tools/acceptance/src/jsonio.rs`
- `tools/acceptance/src/timestamps.rs`
- `tools/acceptance/src/jsonio/tests.rs`
- `tools/acceptance/src/timestamps/tests.rs`

Evidence: rust/soda-acceptance/src/jsonio.rs:25-96 bounded regular JSON reads; rust/soda-acceptance/src/jsonio.rs:98-268 typed fields and compact/indented emission; rust/soda-acceptance/src/jsonio.rs:270-412 RFC3339 date conversion and validation; rust/soda-acceptance/src/jsonio.rs:422-509 byte-emission, date and strict-read tests.

## rust/soda-acceptance/src/native_phase.rs

Observed size: 654 lines, including tests where embedded. Retain the current native prepare/build/check admission and receipt sequence as one cohesive operation; extract its fake-runner fixtures and tests. The existing build phase admits candidate bytes and does not run a producer. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/native_phase.rs`
- `tools/acceptance/src/native_phase/tests.rs`

Evidence: rust/soda-acceptance/src/native_phase.rs:179-250 phase_request admission; rust/soda-acceptance/src/native_phase.rs:311-445 prepare_checkout, phase_command and run_phase; rust/soda-acceptance/src/native_phase.rs:557-588 explicit_phase_order_without_automatic_work; rust/soda-acceptance/src/native_phase.rs:448-654 native phase test module.

## rust/soda-acceptance/src/process.rs

Observed size: 739 lines, including tests where embedded. Retain Phase cancellation/deadline policy and the owned process-group wait/reap/stop lifecycle in their extracted owners. PROC01 consolidates equal std-backed byte drain/capture mechanics, without another async/process framework. N14 immediately repairs Phase::child so a finite child deadline remains effective when its parent has no deadline; RED01 repairs writer-close/error handoff. Preserve leader-unreaped/PGID ownership, resistant-descendant cleanup and incomplete capture custody; keep real private cfg(test) scenarios.

- `tools/acceptance/src/process/mod.rs`
- `tools/acceptance/src/process/phase.rs`
- `tools/acceptance/src/process/launch.rs`
- `tools/acceptance/src/process/owned_process.rs`
- `tools/acceptance/src/process/tests.rs`

Evidence: rust/soda-acceptance/src/process.rs:28-104 Phase and cancellation/deadline methods; rust/soda-acceptance/src/process.rs:144-282 process launch and stdout/stderr pumps; rust/soda-acceptance/src/process.rs:283-442 Linux non-reaping/group cleanup; rust/soda-acceptance/src/process.rs:452-605 Process methods and signal_group; rust/soda-acceptance/src/process.rs:608-739 cancellation and resistant-descendant tests.

## rust/soda-acceptance/src/project_state.rs

Observed size: 943 lines, including tests where embedded. Retain account/Git state collection and explicitly requested workload/database observations, with one run_snapshot shape and its existing gates. PROC01/FS01/JSON01 supply bounded capture, admitted filesystem and serialization mechanics to the extracted owners. SQLITE01 distinguishes actual fixture/probe consumers from product PostgreSQL persistence; it does not authorize blanket driver removal. Keep private cfg(test) cases and source/native evidence limits.

- `tools/acceptance/src/project_state/mod.rs`
- `tools/acceptance/src/project_state/command.rs`
- `tools/acceptance/src/project_state/files.rs`
- `tools/acceptance/src/project_state/snapshot.rs`
- `tools/acceptance/src/project_state/workloads.rs`
- `tools/acceptance/src/project_state/tests.rs`

Evidence: rust/soda-acceptance/src/project_state.rs:165-240 command capture and output_lines; rust/soda-acceptance/src/project_state.rs:242-472 Entry, sorted JSON and traversal; rust/soda-acceptance/src/project_state.rs:499-672 root/account/Git/shared-file observations; rust/soda-acceptance/src/project_state.rs:673-819 declared workload, volume and native database observations; rust/soda-acceptance/src/project_state.rs:822-943 entry, command, gate and network tests.

## rust/soda-acceptance/src/report.rs

Observed size: 665 lines, including tests where embedded. Retain observation authority, retained-file hash verification, handoff admission and incomplete/failed scope reporting in their extracted owners. JSON01 replaces generic observation parsing/emission with a record adapter; RED01 retains final leak refusal and publication custody. Preserve actual hashed bytes and update coupled readers/writers/cases together. Keep private cfg(test) subjects rather than a second serialization engine.

- `tools/acceptance/src/report/mod.rs`
- `tools/acceptance/src/report/observation.rs`
- `tools/acceptance/src/report/handoff.rs`
- `tools/acceptance/src/report/tests.rs`

Evidence: rust/soda-acceptance/src/report.rs:77-294 Observation wire fields and JSON conversion; rust/soda-acceptance/src/report.rs:296-381 evidence hashes and read_observation; rust/soda-acceptance/src/report.rs:383-500 handoff admission/rendering; rust/soda-acceptance/src/report.rs:503-665 round-trip and unsafe-reference tests.

## rust/soda-acceptance/src/trust.rs

Observed size: 506 lines, including tests where embedded. Separate bounded Ignition inline-data decoding from the existing fixture SSH host-key trust comparison. Keep all current encoding and key pin semantics. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/acceptance/src/trust/mod.rs`
- `tools/acceptance/src/trust/inline_data.rs`
- `tools/acceptance/src/trust/host_key.rs`
- `tools/acceptance/src/trust/tests.rs`

Evidence: rust/soda-acceptance/src/trust.rs:31-217 base64/data URI/gzip and IgnitionFile decoding; rust/soda-acceptance/src/trust.rs:223-332 Ignition parse and fixture pinned-host-key verification; rust/soda-acceptance/src/trust.rs:334-506 codec, bounded compression and trust-mismatch tests.

## rust/soda-acceptance/src/vm.rs

Observed size: 944 lines, including tests where embedded. Separate VM configuration/preflight, verified base receipt, owned work/launch preparation and the VM readiness/restart/shutdown lifecycle. Keep failure ownership and retained-work reporting. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

N14 repairs the retained std/serde QMP request-ID exchange in
`tools/acceptance/src/qmp.rs` with one absolute deadline covering connect,
buffered event processing and every read/write. Event storms and trickling
peers cannot extend that deadline. This and the Phase::child repair precede
bounded readiness/shutdown claims; no new QMP runtime is selected. The
[L01 packet](../library-adoption.md#execution-packets) owns the acceptance scope.

- `tools/acceptance/src/vm/mod.rs`
- `tools/acceptance/src/vm/config.rs`
- `tools/acceptance/src/vm/base.rs`
- `tools/acceptance/src/vm/launch.rs`
- `tools/acceptance/src/vm/lifecycle.rs`
- `tools/acceptance/src/vm/tests.rs`

Evidence: rust/soda-acceptance/src/vm.rs:30-257 VmConfig/RemoteConfig decoding and preflight; rust/soda-acceptance/src/vm.rs:259-341 VerifiedBase and launch receipt verification; rust/soda-acceptance/src/vm.rs:343-480 QEMU checks, work preparation and owned arguments; rust/soda-acceptance/src/vm.rs:483-758 Vm, launch_vm, readiness and shutdown; rust/soda-acceptance/src/vm.rs:761-944 config, pin, disjointness and QMP argument tests.

## rust/soda-factory/src/main.rs

Observed size: 593 lines, including tests where embedded. The operator CLI historically has 351 production lines and no coordinator or database. Retain its small real argument selector under CLI01; JSON01 and N2 replace generic envelope and Unix HTTP framing through admitted adapters with actual response bounds. Keep command admission and useful errors together, with the 240-line historical test allocation. No second factory runtime, database or CLI framework is selected for this small selector.

- `cmd/soda-factory/src/main.rs`
- `cmd/soda-factory/src/operator_tests.rs`

Evidence: 1-4: stated thin operator role and no database; 45-151: parse_args/dispatch for status, stop and reconcile; 153-351: sorted Go envelope encoding, send, read_response and response-size/chunk handling; 352-593: operator_server/request_parts, envelope/status/response/misuse tests -> operator_tests.rs.

## rust/soda-test-vm/src/main.rs

Observed size: 768 lines, including tests where embedded. Retain pidfile/liveness state, locked QEMU start and SSH/tunnel/console actions in the completed concern owners. PROC01 replaces raw exec/wait plumbing with standard APIs and deletes unused foreign-shell diagnostic emulation; PATH01 uses admitted native paths rather than a general Go path engine. CLI01 retains the small real verbs. Keep native/KVM gates, fixed management forwarding, status/argv contracts and private cfg(test) subjects; exact Bash text is not an independent operational requirement.

- `tools/test-vm/src/main.rs`
- `tools/test-vm/src/process.rs`
- `tools/test-vm/src/state.rs`
- `tools/test-vm/src/start.rs`
- `tools/test-vm/src/transport.rs`
- `tools/test-vm/src/tests.rs`

Evidence: rust/soda-test-vm/src/main.rs:132-363 process/access/exec diagnostics and signal handling; rust/soda-test-vm/src/main.rs:365-449 pidfile state, VM directory and SSH arguments; rust/soda-test-vm/src/main.rs:451-535 lock/native-input/QEMU start; rust/soda-test-vm/src/main.rs:537-624 status and transport action dispatch; rust/soda-test-vm/src/main.rs:652-768 pid/SSH/status diagnostics tests.

## rust/soda-test-vm/tests/cli.rs

Observed size: 777 lines, including tests where embedded. Group existing status/pidfile, native start/lock/QEMU argument, and exec/SSH/tunnel tests; retain CLI/default-action checks separately and share existing fake commands/VM fixtures.

- `tools/test-vm/tests/cli.rs`
- `tools/test-vm/tests/status.rs`
- `tools/test-vm/tests/start.rs`
- `tools/test-vm/tests/transport.rs`
- `tools/test-vm/tests/support/mod.rs`

Evidence: rust/soda-test-vm/tests/cli.rs:20-82 KVM gate, TempDir, command and fake VM helpers; rust/soda-test-vm/tests/cli.rs:84-169 usage/status/pidfile tests; rust/soda-test-vm/tests/cli.rs:170-566 start host/input/lock/QEMU/refusal tests; rust/soda-test-vm/tests/cli.rs:568-725 SSH failure and exact transport argument tests; rust/soda-test-vm/tests/cli.rs:727-777 working-directory and closed-stdout checks.

## scripts/forgejo_components_test.go

Observed size: 691 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `scripts/forgejo_components_test.go` — Native intro/empty/explore composition checks.
- `scripts/forgejo_theme_components_test.go` — Guest/theme-toggle singleton and route selection cases.
- `scripts/forgejo_form_components_test.go` — Original creation permission/form composition and adapter boundaries.
- `scripts/forgejo_template_fixture_test.go` — Shared native template dictionary/read/call fixtures.

Current `f7e9cf9d` allocation, checked against actual defining bodies by the primary reviewer and coordinator:

| Current lines | Exact target |
| --- | --- |
| 48–345, 548–625 | `forgejo_components_test.go`: intro/empty/Explore and the whole general-page composition case |
| 347–471 | `forgejo_theme_components_test.go`: singleton and header route cases |
| 473–546, 627–660 | `forgejo_form_components_test.go`: whole repository creation case and native form adapter case |
| 14–46, 662–691 | `forgejo_template_fixture_test.go`: one shared locale/context/dictionary and source/read/call fixture owner |

Redistribute imports by actual use within the same private Go test package. Keep the general-page function's opening and intro list at 548–574 with its remainder; grouping them with repository creation would split a function. Read the actual template/style owners at their new paths, preserving native gates, contexts and full Soda design coverage. Primary full assertion inspection and the coordinator's extraction/helper fit are distinct; no test or installed browser execution is claimed.

## tests/build/test_project_keys.py

Observed size: 416 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to the native owner and live tests
listed in [Python cutover closure](../port-assessment.md#python-cutover-closure).
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: test_project_keys.py:20-55 real source loader/setup; 56-142 basic preservation/publication; 143-300 writer/admission races; 301-397 uncertain publication/cleanup; 398-416 CLI/refusal.

## tests/build/test_terminal.py

Observed size: 817 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to the native owner and live tests
listed in [Python cutover closure](../port-assessment.md#python-cutover-closure).
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: test_terminal.py:1-5 limited proof contract; 22-25 real-source load; 28-151 TerminalProtocol; 154-203 managed fixture; 204-600 admission/supervision tests; 602-817 LocalTerminalProcess.

## tests/forgejo/component-boundaries.test.ts

Observed size: 510 lines, including tests where embedded. Split current nested component subtests by layout and control concerns; keep their existing native CSS fixture and authorization guard.

- `tests/forgejo/component-boundaries.test.ts` — Existing native form focus/icon/help/heading and settings-panel/table boundary assertions.
- `tests/forgejo/component-toolbar-boundaries.test.ts` — Existing repository toolbar/control size/button state and adjoining-input assertions.
- `tests/forgejo/component-layout-boundaries.test.ts` — Existing compact/empty/fluid/repository status/package/profile narrow-layout assertions.
- `tests/forgejo/fixtures/component-browser.ts` — Extract current guarded native origin, real palette/styles loading, sandboxed browser/page and render fixture.

Evidence: component-boundaries.test.ts:9-29 guarded fixture; 30-154 form/heading/focus/empty; 155-277 toolbar/button sizes; 278-359 table/fluid/status/settings; 360-448 primary states; 449-510 packages/cleanup.

Open detail: The shared stylesheet loader must include actual split canonical files and preserve existing style order. Do not broaden to installed-product proof or introduce new test contracts.

## tests/frontend/drawer-controls.test.ts

Observed size: 630 lines, including tests where embedded. Split regression cases along the production environment/access/control responsibilities using the existing drawer-fixture as the real browser entry.

- `tests/frontend/drawer-controls.test.ts` — Existing inert mount/visibility/tab/actor retirement, draft preservation and no terminal ownership tests.
- `tests/frontend/project-environment-controls.test.ts` — Existing explicit Create/network/OS/Start/Stop observations and confirmed administration controls.
- `tests/frontend/project-access-controls.test.ts` — Existing deliberate Join, key selection/review/Apply/save/refusal/copy tests.
- `tests/frontend/fixtures/project-controls-driver.ts` — Extract existing drawer browser/server fixture, refresh/click helpers and cleanup once.

Evidence: drawer-controls.test.ts:7-65 browser/server/driver; 67-126 inert controls; 127-325 Create/network/OS/lifecycle; 326-451 keys/Join; 452-629 uncertainty/retirement/hidden controls/identity.

Open detail: Fixture extraction must not repeat production logic or reset browser state between assertions that currently form one regression scenario.

## tests/frontend/terminal.test.ts

Observed size: 559 lines, including tests where embedded. Split existing protocol, retirement, explicit End and stream regressions; use the current terminal browser fixture rather than a new terminal model.

- `tests/frontend/terminal.test.ts` — Existing inert exact-locator mount, reserve/create/attach readiness, generation/binding and native observation contracts.
- `tests/frontend/terminal-retirement.test.ts` — Existing visibility/focus/actor/late import/reservation/renderer/socket retirement cases.
- `tests/frontend/terminal-end.test.ts` — Existing separately confirmed End and unconfirmed/absent native outcome handling.
- `tests/frontend/terminal-stream.test.ts` — Existing Unicode IO, bounded queues, overload and invalid-frame cases.
- `tests/frontend/fixtures/terminal-driver.ts` — Extract current terminal-fixture browser/server setup, action/open/ready/End helpers and cleanup once.

Evidence: terminal.test.ts:11-100 current fixture/helpers; 102-182 mount/readiness/subURL; 183-309 retirement/admission; 310-357 End; 358-520 identity/visibility/observation/retired callbacks; 521-559 bounds/frames.

Open detail: A single scenario spanning readiness and retirement stays intact; splitting file placement cannot weaken original actor/native-generation assertions.

## tests/frontend/workspace.test.ts

Observed size: 1843 lines, including tests where embedded. Split the many current regression groups by established workspace concern. Keep shared driver and actual emitted component fixture; do not copy the product state machine into new tests.

- `tests/frontend/workspace.test.ts` — Existing page/panel shared original-target Create/End and native drawer handoff/renderer continuity cases.
- `tests/frontend/workspace-attention.test.ts` — Existing unread coalescing, authorized stable attention order and stale-generation observations.
- `tests/frontend/workspace-navigation.test.ts` — Existing project/detail/search/This-page switching, actor invalidation, pending rename and Hide ownership cases.
- `tests/frontend/workspace-persistence.test.ts` — Existing exact arrangement restoration, obsolete/corrupt caches and failed storage cases.
- `tests/frontend/workspace-layout.test.ts` — Existing real xterm owner retention through splits/moves/dividers/overflow/maximize and measurement retirement.
- `tests/frontend/workspace-first-use.test.ts` — Existing complete selected-repository welcome/Create/Join/terminal/re-entry scenario.
- `tests/frontend/workspace-setup.test.ts` — Existing pending creation, repository keyboard search/back/change, superseded choice and second-project cancellation cases.
- `tests/frontend/workspace-recovery.test.ts` — Existing stopped/incomplete/failed/native-generation/private-context recovery and partial collection cases.
- `tests/frontend/workspace-responsive.test.ts` — Existing welcome/intro/coherence short viewport, theme/native CSS precedence and long-name menu/focus assertions.
- `tests/frontend/fixtures/workspace-driver.ts` — Extract current browser/server/fixture lifetime and real UI action helpers once, importing the existing workspace-fixture.

Evidence: workspace.test.ts:11-146 fixture/driver; 148-293 drawer/layout surfaces; 294-404 attention; 405-556 exact operations/navigation; 557-640 persistence; 641-766 layout; 767-893 installed controls/measurement; 894-955 first-use helpers; 956-1235 complete first use; 1236-1463 keyboard/setup/responsive; 1464-1596 state/Join/cancellation/inventory; 1597-1843 responsive/native recovery/generation/CSS/partial inventory.

Open detail: Some existing parameterized scenarios cover several concerns and must remain whole. Browser/server hooks and shared mutable fixture lifetime need isolation review when distributing test modules; do not assume parallel execution is safe.

## tests/installed/sodaspaces-workspace-journey.ts

Observed size: 441 lines, including tests where embedded. Separate the two already exported installed scenarios and their shared evidence records; retain their guarded native callers and independent local-fixture proof limits.

- `tests/installed/sodaspaces-workspace-journey.ts` — Existing workspace matrix scenario, exact native transport observation and bounded UI operations.
- `tests/installed/sodaspaces-first-use-journey.ts` — Existing selected-actor/repository Create, keyless Join, terminal and re-entry first-use scenario.
- `tests/installed/sodaspaces-journey-evidence.ts` — Move existing FirstUseEvidence/MatrixSession/MatrixFacts/MatrixEvidence/MatrixNative definitions and native mount identity check to one shared owner.

Evidence: sodaspaces-workspace-journey.ts:1-2 proof scope; 9-24 native mount/session check; 26-171 FirstUseEvidence/exerciseFirstUse; 172-210 matrix records; 211-441 exerciseWorkspaceMatrix.

Open detail: Update actual installed and fixture imports directly without alias modules. Splitting retained scenario source does not establish new installed qualification.

## tests/build/project_factory_roles_test.go

Current source at `f7e9cf9d`: 481 lines and 13 `TestRoles` functions. The
Python driver is already gone. These active Go integration assertions invoke
the compiled Rust helper using a test-owned factory and a recording Git
executable. Retain the real test subject, subprocess observations and fixed
request/refusal bytes in the existing `tests/build` package:

- `tests/build/project_factory_roles_fixture_test.go`: harness at 1–229.
- `tests/build/project_factory_roles_accounts_test.go`: accounts at 230–256.
- `tests/build/project_factory_roles_inputs_test.go`: approved inputs at 257–324.
- `tests/build/project_factory_roles_readiness_test.go`: readiness at 325–346.
- `tests/build/project_factory_roles_lifecycle_test.go`: start, stop,
  interruption, maintenance holds and process groups at 347–435.
- `tests/build/project_factory_roles_output_test.go`: bounded output and exit
  observations at 436–463.
- `tests/build/project_factory_roles_test.go`: malformed requests at 464–481.

The native implementation and its current private unit tests remain Rust in
`cmd/soda-project-terminal/src/factory_roles/`; its compiled
`src/bin/project-factory-roles.rs` entrypoint retains its installed identity.
Update the actual Cargo build selector in the Go fixture when folding the
helper crate; preserve the existing scratch-directory and Git fixtures.
Source assertions about scratch accounts, bundle argv or process groups do
not qualify installed native accounts, real Git verification or factory
execution. Root and A independently inspected the whole current file and all
13 case allocations, supporting this partition. The interactive-account case
writes a shell fixture only; its title alone does not establish extra-group
rejection. These source checks did not execute the suite. The old Python-driver
disposition is superseded.

## tests/build/helpers.go

At `f7e9cf9d`, retain the active Go support at `tests/build/helpers.go`, including
`CargoBinary` at 149–174 and `tail` at 191–196. Retire only the unused
exception-MRO `raisedAs` function at 180–189 and its obsolete comment at
176–179. Root and A independently inspected the source and actual caller
census: the function has no caller, and the former role-test consumer now
invokes the Rust binary. The whole helper file remains active. This is an
exact desired-allocation retirement, not an implementation deletion.

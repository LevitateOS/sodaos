# Project runtime

Current path reconciliation at `45ebf4c4` retains the
`cmd/soda-project-terminal` package, its `src/main.rs` executable, and its two
`src/bin` helpers with their actual modules and tests. There is no selected
`src/lib.rs`. Historical predecessor ranges and optional filename proposals
below are superseded by the [current target tree](../proposed-tree.md), without
changing the retained PTY, descriptor, credential or child-custody contracts.
Native and installed verification retains its existing tasks.

L11 implements the narrow timex lease adapter through `lib/wire-time`, with explicit preepoch expiry refusal and strict producer fixtures. The permissive Python ISO parser and its equivalence vectors are retired. Local PTY/process budgets use an Instant origin while wait, kill/reap and signal ownership stay here.

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

## Current library adoption and retained guest policy

The [library adoption allocation](../library-adoption.md#finding-allocation)
supersedes pending generic-engine extraction in the historical reviews below.
Keep completed guest modules, helper binaries and concern-specific custody
boundaries; reduce generic code through library adoption within those owners.
The [execution packets](../library-adoption.md#execution-packets) and
[readiness gates](../library-adoption.md#readiness-gates) own scheduling and
acceptance, with one physical writer for shared guest files.

| Finding and existing surface | Current target responsibility |
| --- | --- |
| [N9](../../../research/library-reuse-investigation.md#n9): `timex.rs`, lease/deadline callers and elapsed timers | Replace the Python-style calendar parser with the selected strict RFC3339 time adapter for the actual lease producer; use std Instant for local elapsed budgets. Retain raw signed deadline text, expiry/lifetime admission and explicit preepoch policy. |
| [N10](../../../research/library-reuse-investigation.md#n10): `socket.rs`, PTY/readiness helpers | Use typed Unix socket/FD/readiness mechanics where helpful. Retain Linux forkpty/login readiness, queue/frame bounds, signal/resize handling, exact tmux peer UID/PID and socket inode custody, and child close/reap. Keep one synchronous relay owner. |
| [CF-01](../../../research/library-reuse-investigation.md#cf-01), [CF-04](../../../research/library-reuse-investigation.md#cf-04): `sha.rs`, `factory_roles/sha.rs`, Base64 consumers | Replace hash rounds and encoding algorithms with selected libraries. Retain exact approved-file/NUL recipes, full-file program verification and raw managed-key revision bytes; keep the actual strict Base64 profile. |
| [JSON01](../../../research/library-reuse-investigation.md#json01): broker/subscription, binding records and Python-style emitters | Use caller-specific serde DTO/Visitor/output profiles. Retain exact dispatch, caps, parent/writer locks, binding comparison and meaningful producer semantics; generic JSON engines are superseded. |

Managed `key_lines.rs` and `project_account.rs::valid_key` are shallow key-shape
and file-policy gates, not full SSH engines. Retain them without inventing extra
cryptographic admission; the host's complete key-format boundary follows
[CF-03](../../../research/library-reuse-investigation.md#cf-03). HTTP/WebSocket
adoption belongs to the host; it does not replace this guest PTY/seqpacket
protocol or add a guest daemon. Complete syscall/randomness migrations are not
prerequisites to the narrow deadline or ownership repairs.

## Current Project account helper allocation

The current Rust helper is retained in the existing Project terminal package;
the separate guest account package folds into that owner. [P03](../reviews/P03.md)
records A's mapped source review and B's independent source challenge of this
package/test seam. This is a package consolidation, with the same guest helper
process and installed `/usr/libexec/soda/project-account` binary.

- `cmd/soda-project-terminal/src/bin/project-account.rs` receives the current
  `rust/soda-project-account/src/main.rs` bounded entrypoint.
- `cmd/soda-project-terminal/src/project_account.rs` owns current
  `account.rs:1–511` production request, configuration and provisioning.
- `cmd/soda-project-terminal/src/project_account_tests.rs` receives the current
  `account.rs:512–1068` test module. Declare it as a test child of the real
  account module and retain its private subject access; do not copy production
  provisioning or put these tests under the unrelated terminal account module.
- `cmd/soda-project-terminal/tests/project_account.rs` receives the actual
  `rust/soda-project-account/tests/binary.rs:1–410` integration tests.

Rebind the one library module and entrypoint imports, Cargo binary/test discovery,
compiled executable resolver and Go `tests/build/project_account_test.go`
`CargoBinary` selection together. The release-build compile selection and
`production.rs:900–915` fake Cargo recipe must select the requested binary,
because one package will contain multiple actual executables. Rebind package
specific oracle vectors to the retained compiled subject. Preserve Containerfile
installation and native account/refusal semantics. Scratch passwd/tool doubles
and scripted host adapters do not qualify installed account provisioning. Source
inspection and this target allocation authorize no source or manifest changes.

## Project factory roles helper consolidation

Current f7 Rust allocation (historical Python ranges below are not move selectors):
`factory_roles/mod.rs` owns main.rs1–204; the fixed binary owns thin main205–207.
`layout.rs` owns fsx.rs and ops_inspect.rs15–61; `accounts.rs` owns account.rs1–301.
`accounts_tests.rs` owns the actual account cfg/test302–414 as a private real-subject descendant.
Thin do_ensure ops_approve74–84 stays inputs.rs, calling ONE account::ensure_role; its attribute415/body416–429 stays inputs_tests.rs.
The Cargo binary is project-factory-roles (old package soda-project-factory-roles); consolidate into soda-project-terminal with explicit --bin project-factory-roles and preserve existing install destination.
`inputs.rs` owns ops_approve.rs1–367; `records.rs` owns ops_record.rs1–166 and ops_inspect.rs62–236;
`execution.rs` owns proc.rs1–379, ops_record.rs167–199 and ops_inspect.rs237–480.
Retain single private validation and approved-input digest-recipe owners for
validate.rs1–131 and sha.rs1–153, with shared imports for phase/group/barrier/Prestate
rather than duplicated state. CF-01 replaces SHA compression here with sha2; this
allocation preserves the canonical recipe, not a private hash engine.
Descendant tests retain actual private subjects: inputs_tests.rs368–725;
records_tests.rs200–368 from ops_record; execution_tests.rs proc380–420 plus inspect481–771;
validate_tests.rs132–220; sha_tests.rs154–191; tests.rs main211–398 owns one Scratch/recording-Git fixture.
The outer cfg(test) shells/attributes travel once with their corresponding modules.
Keep the actual compiled-helper oracle and active Go fixture assertions. B independently
challenged production and exact module seams; the final P06/P07/P12 source reviews
record the complete assigned assertion assessment and its native evidence limits.
Current `rust/soda-project-factory-roles/tests/oracle.rs`1–468 has the explicit
retained target `cmd/soda-project-terminal/tests/factory_roles_oracle.rs`.
It remains a distinct Rust integration suite executing the actual
project-factory-roles binary in the consolidated soda-project-terminal package,
with one compiled-subject resolver and its existing scratch/recording-Git
fixtures. Preserve its cohesive digest/reference/refusal/stop workflow assertions;
its length is not grounds to replace it with private copied subjects. Rebind
Cargo package/binary selection and vectors to that same executable. These are
source-reviewed assertions, not executed or installed qualification.
These allocations preserve the existing helper process, binary/install identity and Cargo owner.


The historical Python source review below described 800 lines. At `f7e9cf9d`
the helper is already Rust in `rust/soda-project-factory-roles`; the pending
work is consolidation into the existing project-terminal Cargo package, with
current per-duty allocation in [P06](../reviews/P06.md) and the connected
identity/factory review records. The line intervals below refer to the historical
predecessor and cannot be used as current Rust move selectors. Separate the
role/layout, approved input custody, receipt observation and process lifecycle
concerns without widening the privileged protocol or adding a daemon.

- `cmd/soda-project-terminal/src/bin/project-factory-roles.rs` — Thin entrypoint; compiled installation remains `/usr/libexec/soda/project-factory-roles`.
- `cmd/soda-project-terminal/src/factory_roles/mod.rs` — Bounded stdin, exact operation dispatch and lock scope.
- `cmd/soda-project-terminal/src/factory_roles/layout.rs` — Fixed paths, caps, identifiers, roles and root-owned file/layout admission (25-121).
- `cmd/soda-project-terminal/src/factory_roles/accounts.rs` — Fixed coder/reviewer lookup, creation and restricted homes (121-165).
- `cmd/soda-project-terminal/src/factory_roles/inputs.rs` — Approved digest, Git bundle, credential-reference custody and immutable snapshot creation (176-325).
- `cmd/soda-project-terminal/src/factory_roles/records.rs` — Verified receipts and log/phase/status/inspect observations (326-377, 541-623).
- `cmd/soda-project-terminal/src/factory_roles/execution.rs` — Role/environment execution, detached preparation, process Start/Stop, hold/release and confirmed ownership (378-540, 624-800).
- `cmd/soda-project-terminal/src/factory_roles/tests.rs` — Existing helper behavior tested against the actual native implementation; no imported Python subject.

Evidence: project-factory-roles:1-6 fixed contract; 25-121 fixed paths/admission/layout; 121-165 roles; 176-325 approved inputs/bundle/credential/snapshot; 326-377 verified records; 378-540 execution; 541-623 inspect; 624-773 process stop/hold/release; 774-800 main.

Open detail: Declare the two helper binaries and private shared library in the
existing project-terminal manifest. Install the compiled helper root-owned at
its existing path. At 0d8d3b8e, project_factory_roles_test.go drives the compiled
helper, and the ST15 native fixture already builds/stages/hashes it at
st15_demo_native_test.go:277-294,348-353,399-410. Keep those executing-byte
observations during consolidation; this source inspection is not installed proof.

## rust/soda-project-terminal/src/broker.rs

Observed size: 1342 lines, including tests where embedded. This is the fixed project-local identity execution helper inside project-terminal, distinct from the host soda-identity custody service. Split 1,118 lines before tests into existing JSON/wire helpers, reserve/private-profile preparation, stored profile/binding/unit-incarnation observation, protected credential seed/capture, pinned harness start, cgroup freeze/control, and confirmed finish/stop/model retirement. broker.rs retains dispatch plus bounded stdin/stdout, 45-second alarm, exclusive TERMINALS parent lock and fixed non-secret stderr. Preserve prepare cleanup precedence, freeze before capture, kill before stop and retirement only after confirmed stop. JSON01 replaces generic JSON/numeric machinery with explicit subscription DTO/comparison profiles; required binding semantics come from actual producers, not every Python dict/numeric quirk. Its 223-line existing test module remains together; no new subscription operation or separate daemon is proposed.

- `cmd/soda-project-terminal/src/broker.rs`
- `cmd/soda-project-terminal/src/subscription_wire.rs`
- `cmd/soda-project-terminal/src/subscription_prepare.rs`
- `cmd/soda-project-terminal/src/subscription_profile.rs`
- `cmd/soda-project-terminal/src/subscription_credentials.rs`
- `cmd/soda-project-terminal/src/subscription_start.rs`
- `cmd/soda-project-terminal/src/subscription_cgroup.rs`
- `cmd/soda-project-terminal/src/subscription_retire.rs`
- `cmd/soda-project-terminal/src/broker_tests.rs`

Evidence: 1-53,1011-1118: documented guest stdin/stdout protocol, subscription_dispatch, BrokerFail/read/write/alarm/broker_main; broker_inner owns exclusive parent lock across dispatch -> broker.rs; 60-240: json_equal/json_int/lease_with_binding/deadline_ok/result/decode_request/native_binding/profile_object -> subscription_wire.rs; 269-280,298-423: mount argv, subscription_prepare/provision, exact reserve/provision/cleanup order -> subscription_prepare.rs; 241-268,281-297,424-524,933-1010: paths/lease actor/execution/deadline, ProfileHit/subscription_profile, invocation/check_unit, lookup/resolve -> subscription_profile.rs; 525-600,773-800: no-follow owner changes/auth directory, subscription_seed/stage and capture -> subscription_credentials.rs; 601-689: root-owned harness digest, fixed respawn argv and subscription_start repeated binding/unit proof -> subscription_start.rs; 690-772: subscription_cgroup/kernel write/cgroup.events parse and confirmed freeze -> subscription_cgroup.rs; 801-932: bounded missing-ok removal/is_mount, subscription_retire and subscription_finish; preserve finally kill/stop/body-error precedence -> subscription_retire.rs; 1119-1342: numeric deep equality/coercion, binding/deadline/decode/result/argv/cgroup/mount/removal/dispatch cases -> broker_tests.rs.

## rust/soda-project-terminal/src/fs.rs

Observed size: 534 lines, including tests where embedded. Keep the used descriptor-relative protected record/filesystem boundary: regular-file UID/mode/link checks, bounded record read, create/replace/unlink, chmod/chown and full-fstat/read-up-to helpers. Production spans lines 1-269 with a small test-only predicate at 60-64; the embedded test module spans 270-534. Extract the retained tests and test-only predicate within the same crate.

- `cmd/soda-project-terminal/src/fs.rs`
- `cmd/soda-project-terminal/src/fs_tests.rs`

Evidence: 14-35,56-59,65-269: component checks, root_file/read_record/new_file and descriptor-relative operations; fstat_all at 238 and read_up_to at 247 remain actual used helpers. Retained test support and safety/file/record/create/wrapper/chown/chmod cases at 60-64,270-292,308-534 → fs_tests.rs.

Retire the unused `root_chain` function and its future-only commentary at
36-55, its exclusive `root_chain_matrix` at 293-307, and the obsolete reference
in `sys.rs:9`. Source review at `f7e9cf9d` found no production caller; `main.rs`
declares a private binary module, and actual terminal/key paths use their own
per-level directory checks. A independently challenged this scoped retirement.
These units have no desired-tree allocation. Keep the active `fs`/`sys` modules
and their actual consumers; source deletion remains a later implementation.

## rust/soda-project-terminal/src/keys.rs

Observed size: 614 lines, including tests where embedded. Split 442 lines before tests into canonical managed authorized-key line parsing, strict request/truthiness/duplicate handling, and the existing locked filesystem preview/replace plus fixed CLI failure/result boundary. Keep update and replace_inner together: stable directory lock, login-to-identity admission, preview hash, exclusive temporary creation, fsync, bytes+FileId reread before rename, parent fsync and result reread form one atomic publication path. Extract the 171-line existing tests without inventing a new SSH parser, update mode or validation policy.

- `cmd/soda-project-terminal/src/keys.rs`
- `cmd/soda-project-terminal/src/key_request.rs`
- `cmd/soda-project-terminal/src/key_lines.rs`
- `cmd/soda-project-terminal/src/keys_tests.rs`

Evidence: 1-37,39-112: fixed limits/key algorithm line shape, canonical_key_ok/canonical_lines -> key_lines.rs; 113-239: Python truthiness, decoded duplicate rejection, KeyRequest/decode_key_request/state_object -> key_request.rs; 240-380: protected keys directory, FileId/read_keys/candidate_name/update/replace_inner -> keys.rs; 381-442: bounded stdin/write/EPIPE/fixed-stderr key_main/key_inner -> keys.rs; retain with atomic operation boundary; 443-614: key shape/canonical lines/truthiness/request/candidate/state and unprivileged refusal -> keys_tests.rs.

## rust/soda-project-terminal/src/pty.rs

Observed size: 1002 lines, including tests where embedded. Split the 756-line production PTY implementation into public ready/output/closed orchestration, fork/login/resize/child termination, relay control/backpressure/deadline handling, and raw descriptor/readiness I/O. Keep relay_pty_session as the existing loop over apply/ingest/read/stop/flush helpers; do not add another session state machine. Preserve no terminal bytes in diagnostics, parent size gate before child exec, heartbeat/lifetime caps and existing EINTR behavior. Extract the 244-line tests as one module.

- `cmd/soda-project-terminal/src/pty.rs`
- `cmd/soda-project-terminal/src/pty_process.rs`
- `cmd/soda-project-terminal/src/pty_relay.rs`
- `cmd/soda-project-terminal/src/pty_io.rs`
- `cmd/soda-project-terminal/src/pty_tests.rs`

Current target: [N10](../../../research/library-reuse-investigation.md#n10)
replaces duplicated raw FD/nonblocking/readiness mechanics with typed helpers
only where the actual caller benefits. `pty_relay` keeps its existing synchronous
queue/heartbeat/deadline/cleanup policy and one session owner; host tungstenite
adoption does not introduce an async guest runtime. Keep the forkpty child gate
and definitive termination/reap boundary together.

Evidence: 23-54,115-150,621-655,693-756: stop signal handling, output frames, bounded closed flush and run_terminal orchestration -> pty.rs; 151-332: tmux attach argv, size, child wait/end and spawn_login_pty with fork/ready gate -> pty_process.rs; 333-443,519-620: apply_control_frame/ingest/read output/session_should_stop/take_control_input/flush queues/relay loop -> pty_relay.rs; 55-114,444-518,656-692: nonblocking/read/write/select/wait_readable -> pty_io.rs; 757-1002: exact frames, ingest/backpressure/deadlines, child lifecycle/resize/bounds cases -> pty_tests.rs.

## rust/soda-project-terminal/src/svc.rs

Observed size: 882 lines, including tests where embedded. Separate owned systemd unit field/argv/state/stop handling, cgroup filesystem/emptiness admission, tmux socket ownership/peer identity, and tmux subprocess control. Production is 665 lines; tests are 215 and can remain together while moved methods keep private access inside the same crate. Preserve stop confirmation with empty cgroup, no-follow descriptors, socket peer PID/UID/inode checks and no arbitrary control commands.

- `cmd/soda-project-terminal/src/svc.rs`
- `cmd/soda-project-terminal/src/cgroup.rs`
- `cmd/soda-project-terminal/src/socket.rs`
- `cmd/soda-project-terminal/src/tmux.rs`
- `cmd/soda-project-terminal/src/svc_tests.rs`

Current target: the socket adapter follows
[N10](../../../research/library-reuse-investigation.md#n10) with typed connect and
peer-credential mechanics, preserving the existing PID/UID/inode checks and
retry/deadline classification. Systemd/cgroup/tmux admission remains Soda policy;
these concern files are not targets for a new general process or session engine.

Evidence: 19-57,87-200: fixed unit naming/properties/argv, parse/verify service fields, state/confirmed stop -> svc.rs; 58-68,201-458: bounded statfs subprocess/read-only filesystem checks and cgroup_parent/events/empty -> cgroup.rs; shared existing wait/exit helpers stay crate-private to consumers; 15-18,459-604: SocketCheck/socket_identity_kinded/connect classification/exact socket_identity -> socket.rs; 69-86,605-665: infocmp/tmux fixed argv and tmux_control -> tmux.rs; 666-882: exact argv/fields/cgroup/socket/statfs/supervisor/tmux-negative vectors -> svc_tests.rs.

L04 completed guest JSON transfer in `1350250a`, with borrowed-child refinement
in `3e1504ac`. Binding/reservation/ready and control records now use concrete
Serde admission. `state_json.rs` retains only dynamic factory/subscription data,
ordinary last-wins dictionary position, raw numbers and 127-container admission;
`pyemit.rs` and factory emission keep the actual Python byte profiles. Grammar
and escaping belong to Serde. The existing executable, private-file, lease,
service and PTY authority boundaries remain; further parser/codec splits are
superseded, while the parked filesystem/socket/time work below remains open.

## rust/soda-project-terminal/src/term.rs

Observed size: 1821 lines, including tests where embedded. Split 1,363 lines before the embedded test module along already-defined terminal phases: held path/record helpers; account binding/reservation records and serialization; native status; confirmed finished-terminal collection; pinned-program reservation/create; exclusive writer attach/subscription lifetime; list/mutate/control dispatcher; and systemd post-start tmux preparation. Keep the parent TERMINALS shared/exclusive lock in control_terminal spanning its current operations, and keep the writer lock held throughout relay. Split 457 embedded-test lines into binding/reservation/status, exact argv/output/lifetime, and program-hash/native entry smoke groups. Do not move terminal transcript/bytes into persisted state or add lifecycle behavior.

- `cmd/soda-project-terminal/src/term.rs`
- `cmd/soda-project-terminal/src/term_paths.rs`
- `cmd/soda-project-terminal/src/term_binding.rs`
- `cmd/soda-project-terminal/src/term_status.rs`
- `cmd/soda-project-terminal/src/term_collect.rs`
- `cmd/soda-project-terminal/src/term_create.rs`
- `cmd/soda-project-terminal/src/term_attach.rs`
- `cmd/soda-project-terminal/src/term_prepare.rs`
- `cmd/soda-project-terminal/src/term_binding_tests.rs`
- `cmd/soda-project-terminal/src/term_protocol_tests.rs`
- `cmd/soda-project-terminal/src/term_native_tests.rs`

Evidence: 1066-1181: list_owned_terminals/mutate_terminal/control_terminal and parent lock scope -> term.rs; 33-146: terminal constants/path/checked_chain/list/fstatat/existence/output failure helpers -> term_paths.rs; 147-314,714-787: BindingRecord/Reservation validation/read/account match, JSON record/name emission -> term_binding.rs; 315-489,1243-1258: unit/state classification, ready parsing/writer/socket observation/status and ready_object -> term_status.rs; 490-657: terminal listing/stub retirement/screen and owned-file removal/collect_finished -> term_collect.rs; remove_owned_files is now crate-private for broker caller; 658-713,788-967: regular root-owned program hash/full-file verification and reserve/systemd_run/create -> term_create.rs; 968-1065: lease lifetime/subscription command and held-writer attach -> term_attach.rs; 1182-1242,1259-1363: tmux session argv/subscription setup/prepare/prepare_inner -> term_prepare.rs; 1364-1578: account/binding fixture, path/binding/account/reservation/status matrices -> term_binding_tests.rs; 1579-1736: exact systemd/tmux/subscription/lifetime/output vectors -> term_protocol_tests.rs; 1737-1821: full binary hash beyond 64KB, ready/native entry/tmux configuration cases -> term_native_tests.rs; src/main.rs:18-30 wires term/broker/keys into the fixed single argv-dispatched project-terminal binary; retain this crate and installed binary identity.

## rust/soda-project-terminal/src/timex.rs

Observed size: 403 lines, including tests where embedded. The 234-line production ISO deadline parser and its CPython compatibility vectors are superseded by [N9](../../../research/library-reuse-investigation.md#n9). The sole production parser caller reads a lease deadline produced as RFC3339 with fractional nanoseconds. Use the selected time codec with explicit lexical gates rather than extracting date/time/offset engines. Retain timezone/expiry/lifetime admission and raw signed text; arbitrary separators, basic dates and offset seconds have no demonstrated producer requirement. Resolve preepoch truncation versus floor explicitly, or reject preepoch expiry values consistently with live admission. Keep meaningful producer/boundary assertions, replacing obsolete grammar-equivalence vectors.

- `cmd/soda-project-terminal/src/timex.rs`
- `cmd/soda-project-terminal/src/timex_tests.rs`

Current target: `timex.rs` is a small library-backed lease/deadline adapter;
`timex_tests.rs` exercises that producer contract. Their completed placement
remains, while calendar arithmetic and general Python ISO parsing disappear.

Evidence: 1-31: exact CPython grammar and three documented deviations; 33-234: now_secs/civil date, parse_date/time/offset and parse_iso_deadline; 235-403: CPython accepted/rejected vectors, deviations, civil math and current-clock case -> timex_tests.rs.

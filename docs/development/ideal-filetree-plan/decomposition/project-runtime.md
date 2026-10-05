# Project runtime

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

## project-os/rootfs/usr/libexec/soda/project-factory-roles

Observed size: 800 lines, including tests where embedded. Replace this Python
helper in the existing project-terminal Cargo package. Separate its current
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
its existing path, and update project_factory_roles_test.go and the ST15 native
copy/hash at st15_demo_native_test.go:321-325,371-381 to stage/verify those real
executing bytes. The old Python source and import driver both retire.

## rust/soda-project-terminal/src/broker.rs

Observed size: 1342 lines, including tests where embedded. This is the fixed project-local identity execution helper inside project-terminal, distinct from the host soda-identity custody service. Split 1,118 lines before tests into existing JSON/wire helpers, reserve/private-profile preparation, stored profile/binding/unit-incarnation observation, protected credential seed/capture, pinned harness start, cgroup freeze/control, and confirmed finish/stop/model retirement. broker.rs retains dispatch plus bounded stdin/stdout, 45-second alarm, exclusive TERMINALS parent lock and fixed non-secret stderr. Preserve prepare cleanup precedence, freeze before capture, kill before stop, retirement only after confirmed stop, and Python-compatible numeric/dict semantics. Its 223-line existing test module remains together; no new subscription operation or separate daemon is proposed.

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

Observed size: 534 lines, including tests where embedded. Keep the cohesive descriptor-relative protected record/filesystem boundary: root-chain/regular-file UID/mode/link checks, bounded record read, create/replace/unlink, chmod/chown and added full-fstat/read-up-to helpers. Production spans lines 1-269 with a small test-only predicate at 60-64; the embedded test module spans 270-534. Extract those tests and the test-only predicate instead of adding another filesystem wrapper layer or new permission rule.

- `cmd/soda-project-terminal/src/fs.rs`
- `cmd/soda-project-terminal/src/fs_tests.rs`

Evidence: 14-59,65-269: component checks, root_chain/root_file/read_record/new_file and descriptor-relative operations; fstat_all at 238 and read_up_to at 247 remain reusable actual source helpers; 60-64,270-534: test-only s_isdir plus root-chain/safety/file/record/create/wrapper/chown/chmod cases -> fs_tests.rs.

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

Evidence: 23-54,115-150,621-655,693-756: stop signal handling, output frames, bounded closed flush and run_terminal orchestration -> pty.rs; 151-332: tmux attach argv, size, child wait/end and spawn_login_pty with fork/ready gate -> pty_process.rs; 333-443,519-620: apply_control_frame/ingest/read output/session_should_stop/take_control_input/flush queues/relay loop -> pty_relay.rs; 55-114,444-518,656-692: nonblocking/read/write/select/wait_readable -> pty_io.rs; 757-1002: exact frames, ingest/backpressure/deadlines, child lifecycle/resize/bounds cases -> pty_tests.rs.

## rust/soda-project-terminal/src/svc.rs

Observed size: 882 lines, including tests where embedded. Separate owned systemd unit field/argv/state/stop handling, cgroup filesystem/emptiness admission, tmux socket ownership/peer identity, and tmux subprocess control. Production is 665 lines; tests are 215 and can remain together while moved methods keep private access inside the same crate. Preserve stop confirmation with empty cgroup, no-follow descriptors, socket peer PID/UID/inode checks and no arbitrary control commands.

- `cmd/soda-project-terminal/src/svc.rs`
- `cmd/soda-project-terminal/src/cgroup.rs`
- `cmd/soda-project-terminal/src/socket.rs`
- `cmd/soda-project-terminal/src/tmux.rs`
- `cmd/soda-project-terminal/src/svc_tests.rs`

Evidence: 19-57,87-200: fixed unit naming/properties/argv, parse/verify service fields, state/confirmed stop -> svc.rs; 58-68,201-458: bounded statfs subprocess/read-only filesystem checks and cgroup_parent/events/empty -> cgroup.rs; shared existing wait/exit helpers stay crate-private to consumers; 15-18,459-604: SocketCheck/socket_identity_kinded/connect classification/exact socket_identity -> socket.rs; 69-86,605-665: infocmp/tmux fixed argv and tmux_control -> tmux.rs; 666-882: exact argv/fields/cgroup/socket/statfs/supervisor/tmux-negative vectors -> svc_tests.rs.

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

Observed size: 403 lines, including tests where embedded. The 234-line production ISO deadline parser is one cohesive grammar with integer date math and explicit documented CPython deviations. Extract the 167-line vectors instead of fragmenting date/time/offset into tiny files. Preserve required timezone, truncation toward zero, accepted offset forms, and the existing rejection of naive/week-date/unsplit inputs.

- `cmd/soda-project-terminal/src/timex.rs`
- `cmd/soda-project-terminal/src/timex_tests.rs`

Evidence: 1-31: exact CPython grammar and three documented deviations; 33-234: now_secs/civil date, parse_date/time/offset and parse_iso_deadline; 235-403: CPython accepted/rejected vectors, deviations, civil math and current-clock case -> timex_tests.rs.


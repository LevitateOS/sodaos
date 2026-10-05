# Host runtime

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

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

Evidence: 19-143: revision/key validation, canonicalize_account_keys/canonical_keys and access-key state decoding; 146-254: Runtime::account and Runtime::access_keys; preserve preview before apply and target incarnation checks; 255-598: Mock, fixture payloads, key normalization and account provisioning/rejection -> account_tests.rs; 599-764: access_keys_preview_observes_without_applying/apply_round_trips_preview_and_confirms_set/rejects_drift -> access_keys_tests.rs.

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

Observed size: 1661 lines, including tests where embedded. Split the current host-local codec into value/public decode/output encoding, structural scanner, string token decoding, number scanning, typed field specifications/BoundMap access, and binding/type-error handling. The production code is 1,307 lines before tests. run_machine is one 276-line structural state machine with object/array/value branches; retain that algorithm together after removing unrelated lexical and binder code rather than creating one file per state. Separate existing strict/scanner/depth/UTF8 tests from binder/escaping/tolerant tests. Keep strict request policy distinct from tolerant Podman decoding and from lib/json.

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

Open detail: Scanner visibility/imports must be resolved inside this crate during implementation. Different duplicate-key and escaping contracts preclude an unverified blanket replacement with the shared minimal parser.

## rust/soda-host/src/preparation.rs

Observed size: 1209 lines, including tests where embedded. This is the pure preparation record/validation surface, not the executor. Group the existing validators and Preparation/Prepare identity in preparation.rs; RequirementAcceptance/AdminApproval in preparation_decisions.rs; ApprovedSetup/digest in preparation_setup.rs; resolved-tool, inspect/stop/hold and state wire in preparation_state.rs; FactoryCandidate in preparation_candidate.rs. Split 452 lines of tests into validation/digest/candidate and exact decoding/state-output cases. Move typed Spec tables with their consuming records.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/preparation/mod.rs`
- `lib/host/src/preparation/decisions.rs`
- `lib/host/src/preparation/setup.rs`
- `lib/host/src/preparation/state.rs`
- `lib/host/src/preparation/candidate.rs`
- `lib/host/src/preparation/validation_tests.rs`
- `lib/host/src/preparation/wire_tests.rs`

Evidence: 38-101,207-315,390-437: role/phase/ID validators, Preparation and Prepare -> preparation.rs; 102-206: RequirementAcceptance/AdminApproval and validation -> preparation_decisions.rs; 316-388: ApprovedSetup allowed files/bundle and setup_digest_of -> preparation_setup.rs; 438-698: ResolvedTool, PrepareState/Inspect/Stop, HoldState/PrepareHold -> preparation_state.rs; 699-755: FactoryCandidate validation/from_value/from_map -> preparation_candidate.rs; 756-1044: current record fixtures and validator/setup-digest/candidate cases -> preparation_validation_tests.rs; 1045-1209: prep_json/prepare_json, exact decoding messages and omitempty state output -> preparation_wire_tests.rs.

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

Open detail: soda-host/Cargo.toml currently defines only a library and no Rust binary; these are physical splits of existing library code, not evidence of daemon cutover.

## rust/soda-host/src/ssh.rs

Observed size: 1028 lines, including tests where embedded. Separate authorized_keys text/options scanning and public parse/marshal/fingerprint entrypoints from standard/Go base64 semantics, SSH binary/mpint primitives, non-certificate key material, and certificate field/tuple handling. Preserve supported algorithms, NIST on-curve checks, canonical signed-minimal mpints, sorted certificate tuples, and multiline trailing-data policy. The longest existing algorithm is parse_key_fields at about 97 lines and remains cohesive; no new key algorithm or dependency is proposed.

Disposition: retained at `lib/host/` under the decided language policy; pending PR26 consumes this crate as the daemon foundation.

- `lib/host/src/ssh/mod.rs`
- `lib/host/src/ssh/base64.rs`
- `lib/host/src/ssh/mpint.rs`
- `lib/host/src/ssh/material.rs`
- `lib/host/src/ssh/certificate.rs`
- `lib/host/src/ssh/tests.rs`

Evidence: 26-64,756-907: trim_ws, parse_public_key/parse_key_text/scan_options/parse_authorized_key and public marshal/fingerprint -> ssh.rs; 65-229: strict standard and Go-specific base64 decoders/encoders -> ssh_base64.rs; 231-412: read_string/read_u32/read_u64/put_string, Mpint parsing/comparison/marshalling -> ssh_mpint.rs; 436-471,489-594,698-755: ParsedKey/KeyMaterial, curve_for, ordinary key parsing/marshalling -> ssh_material.rs; 413-435,472-488,595-697: certificate algorithm names/material, parse_tuples/parse_cert/marshal_tuples -> ssh_certificate.rs; certificate arm of marshal_fields stays with certificate responsibility; 908-1028: base64, canonical/authorized-key and fingerprint fixtures -> ssh_tests.rs.

## rust/soda-host/src/tailnet_companion.rs

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

Observed size: 1538 lines, including tests where embedded. Keep pure Tailnet DTO/error/ID definitions separate from RFC3339Nano validity, native status/preferences JSON binding with Go case folding, DNS/IP parsing/canonicalization, and project status/binding decisions. The 892-line production portion has clear existing seams. Split tests by has-node, project-status/native-binding, and address/DNS/time vectors; retain cohesive tables together rather than one file per case. None of these files gains runtime or SQL authority.

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

Observed size: 4075 lines, including tests where embedded. The file contains 2,892 lines before tests plus a 1,183-line embedded test module, spanning every maintenance phase. Preserve one ordered maintain operation: admit pinned public tool sources, bind exact project incarnation, optionally ensure system bus/stage tools, prepare public directory, then attach the restricted mount. Split current flags, host-config wire/validation, native tool files with their path/errno operations, existing config wire diagnostics, JSON lexical parser, release-payload read/decode/validation, network parsing, bounded command execution, project observation, archive/staging and interface mount/admission into concern files. Parser string decoding separates from scalar/structural scanner so neither lexical file must carry the unrelated release/host object bindings. Retain fixed scripts with their existing phase. Test files follow existing groups and shared TestDir/synthetic archive helpers move to test_support.rs only. Keep path/errno support with the real pinned-file operations in filesystem.rs (about 309 production lines); the existing Go quote renderer stays with config wire diagnostics rather than a tiny standalone module. These shared crate-private functions may be reused by current flags/network/interface callers without introducing a new package.

- `cmd/soda-muse-maintain/src/main.rs`
- `cmd/soda-muse-maintain/src/options.rs`
- `cmd/soda-muse-maintain/src/config.rs`
- `cmd/soda-muse-maintain/src/config_wire.rs`
- `cmd/soda-muse-maintain/src/config_validation.rs`
- `cmd/soda-muse-maintain/src/filesystem.rs`
- `cmd/soda-muse-maintain/src/json.rs`
- `cmd/soda-muse-maintain/src/json_string.rs`
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

Open detail: Host config uses duplicate-last-wins/null-no-op/trailing-data/error-priority behavior; release and observation decoders have stricter policies. These remain separate bindings over the existing scanner. Consolidation with other crates or replacing the parser requires further behavior comparison, outside this physical split plan.

## rust/soda-muse/src/main.rs

Observed size: 1637 lines, including tests where embedded. Split 1,233 production lines along the existing launcher/helper responsibilities. Keep command/native-action selection in main; shell input/validation in shell; seqpacket+FD passing/signal/resize forwarding in launch; credential-view admission/native exec/environment in execution; account marker/passwd checks in account; config copy/read in config; pinned-version/private clean subprocess in runtime; Go path/errno/quoting semantics in paths. Exact request/exit output codec stays distinct from scalar JSON token scanning. Split the 401-line test module into shell/wire/paths, FD/signal transport, and config/account/native-dispatch tests. Preserve one launch-only socket and broker confirmation gates.

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


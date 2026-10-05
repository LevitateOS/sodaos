# Identity brokering

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

## rust/identity-providers/src/codex.rs

Observed size: 702 lines, including tests where embedded. Separate the existing pinned-provider/process/enrollment lifecycle from app-server request/reply dispatch and notification parsing. Keep Config admission, environment filtering, binary digest and version checks together in codex_config.rs. codex.rs retains Provider and Session ownership, process stop before credential reading, and private tmpfs cleanup. Move the embedded tests without changing their fixtures. All files stay in the same provider crate; no Soda grant policy moves into this provider.

- `cmd/soda-identity/src/providers/codex/mod.rs`
- `cmd/soda-identity/src/providers/codex/protocol.rs`
- `cmd/soda-identity/src/providers/codex/config.rs`
- `cmd/soda-identity/src/providers/codex/tests.rs`

Evidence: 21-157: Config/Provider, new/start/create_session and split_env -> codex.rs and codex_config.rs; 161-242,304-397: Inner/Session, snapshot/cancel, device enrollment, finish/account/credential_file/stop/close -> codex.rs; 180-190,243-302,400-500: Message, send/call/protocol_error/read_loop/notify -> codex_protocol.rs; 502-565: environment/filter_env/validate_config/check_binary/check_version -> codex_config.rs; 566-702: environment/config fixtures, managed enrollment, cancellation, disabled-device tests -> codex_tests.rs.

## rust/identity-providers/src/muse.rs

Observed size: 561 lines, including tests where embedded. The production provider is 347 lines and owns one native enrollment lifecycle. Its process, prompt recognition, immutable subscription validation and private credential-file checks are cohesive; do not invent additional provider layers merely because tests bring the whole file to 561 lines. Extract the 213-line test module, keeping its real CLI-script fixtures and its existing environment/config/digest/version assertions.

- `cmd/soda-identity/src/providers/muse/mod.rs`
- `cmd/soda-identity/src/providers/muse/tests.rs`

Evidence: 22-163: Config/Provider/Inner/Session, native login process, finish and confirmed close; 165-242: bounded prompt read_loop/complete/scan_device_url and fixed environment; 244-346: pinned configuration/binary/version checks plus credential_valid/credential_file; 348-561: native_subscription_and_private_file, environment/device prompt, digest/version, presentation-private fixtures -> muse_tests.rs.

## rust/soda-identity-compose/src/main.rs

Observed size: 910 lines, including tests where embedded. Keep main as the explicit one-service registration sequence. Move existing flag decisions to options.rs; Compose override/up/ps/immutable-child selection to compose.rs; private tmpfs registration root, provisioned account marker and launch-socket registration to registration.rs; request/exit wire handling to launch_wire.rs; generic scalar/skip token routines to launch_json.rs. The production portion is 750 lines and tests 158. Preserve service opt-in, immutable Podman child attribution, launch-only socket access and current error/JSON behavior; do not add Compose orchestration features.

- `cmd/soda-identity-compose/src/main.rs`
- `cmd/soda-identity-compose/src/options.rs`
- `cmd/soda-identity-compose/src/compose.rs`
- `cmd/soda-identity-compose/src/registration.rs`
- `cmd/soda-identity-compose/src/launch_wire.rs`
- `cmd/soda-identity-compose/src/launch_json.rs`
- `cmd/soda-identity-compose/src/compose_tests.rs`

Evidence: 20-47: load options, attest account, create runtime root, launch one service and register -> main.rs; 13-18,49-165: Options and flag/validation/usage helpers -> options.rs; 245-287,316-345,697-750: launch_compose/write_override/select_compose_child and immutable ID/service attribution -> compose.rs; 166-244,288-315,626-696: private tmpfs root/random/mkdir, account marker and SOCK_SEQPACKET registration -> registration.rs; 346-470: exact string/request encoding and LaunchExit decode -> launch_wire.rs; 471-625: parse_json_string/parse_json_integer/skip_json_value -> launch_json.rs; 751-910: flags, exact override/wire response, runtime-root and Compose-child vectors -> compose_tests.rs.

## rust/soda-identity/src/control.rs

Observed size: 837 lines, including tests where embedded. The 837-line controller has no embedded tests; separate its already-existing broker concerns while retaining the single mutex-protected State and one Controller. Provider adapter traits/mapping are transport-neutral broker integration; enrollment and grants remain distinct from acquisition/execution admission, registration/credential delivery, and confirmed retirement/reconciliation. Keep all current lock/transaction and native runtime attestation sequencing. This is a file split within one broker crate, not independent controllers or a new authorization layer. Keep the small existing provider/runtime adapter block in control.rs; providers/mod.rs is the one root for the consolidated native provider implementations.

- `cmd/soda-identity/src/control.rs` — Provider/runtime traits and existing session adapters remain with controller construction, State, lock ownership and shared controller helpers.
- `cmd/soda-identity/src/enrollment.rs`
- `cmd/soda-identity/src/grants.rs`
- `cmd/soda-identity/src/acquisition.rs`
- `cmd/soda-identity/src/registration.rs`
- `cmd/soda-identity/src/retirement.rs`

Evidence: 13-85: provider/runtime traits, error mapping and Codex/Muse session adapters -> control.rs; 86-165,800-837: EnrollmentEntry/State/Controller, construction/lock/ownership, close/error join/random-ID/zeroize -> control.rs; 133-143,166-272: pending/start/read/cancel enrollment -> enrollment.rs; 273-298,705-739: grant listing/create plus connection/grant revoke -> grants.rs; retirement helpers remain called from retirement.rs; 298-501: lease list, acquire/admit/replay/reserve/get/close execution and reservation authorization -> acquisition.rs; 502-632: register/registration authority/execution binding/terminal attestation/release and return_lease -> registration.rs; 633-704,740-799: uncertain/end/finish/reconcile/sweep/reject/retire_connection -> retirement.rs.

## rust/soda-identity/src/http.rs

Observed size: 494 lines, including tests where embedded. Separate socket serving/shutdown/inflight ownership from bounded HTTP head/body framing and encoding, and from current admin/runtime route admission. Production is 464 lines and tests are 28. Keep runtime_allowed enforcement on the existing service socket path and POST/query/Origin checks intact; route files do not own broker custody or policy. Do not replace this with a new server framework.

- `cmd/soda-identity/src/http.rs`
- `cmd/soda-identity/src/http_wire.rs`
- `cmd/soda-identity/src/http_routes.rs`
- `cmd/soda-identity/src/http_tests.rs`

Evidence: 73-151: Server construction/serve/handle_connection, shutdown/inflight and dead-listener behavior -> http.rs; 152-283,431-464: Admission/HttpRequest/read_request/percent_decode/success_response/error_response -> http_wire.rs; 1-72,284-430: allowed wire fields/nested specs, dispatch/error mapping, admin and runtime routes -> http_routes.rs; 465-494: percent-path, error-body and success-envelope cases -> http_tests.rs.

## rust/soda-identity/src/pg.rs

Observed size: 512 lines, including tests where embedded. Separate PostgreSQL DSN parsing/percent decoding, connection/transport/authentication and message receipt, and row decoding/query/simple-command result handling. Keep the current upstream postgres-protocol framing/SCRAM use and one Client connection/buffer. Production is 475 lines and tests 35; no new pooling/retry/TLS support is proposed. Authentication phases already have authenticate and authenticate_scram functions and need no extra state machine.

- `cmd/soda-identity/src/pg.rs`
- `cmd/soda-identity/src/pg_dsn.rs`
- `cmd/soda-identity/src/pg_query.rs`
- `cmd/soda-identity/src/pg_tests.rs`

Evidence: 19-125: Dsn parse/percent_decode -> pg_dsn.rs; 126-146,187-327,456-475: Stream/Client/connect/send/receive/authenticate/authenticate_scram and backend error -> pg.rs; 147-186,328-455: Row scalar/bytea conversions and query/simple/command_count -> pg_query.rs; 476-512: DSN forms and command-tag row counts -> pg_tests.rs.

## rust/soda-identity/src/store.rs

Observed size: 798 lines, including tests where embedded. Split existing SQL locality by connection custody, grants, leases, execution records and audit events, retaining Store/Tx plus parameter binding/encoding/transaction primitives in store.rs. Schema bootstrap and grant-key admission belong together in store_schema.rs; preserve internal/store/schema.go as the existing authoritative schema and leave schema.rs a mechanically synchronized representation. Production is 772 lines and tests 24. Do not create separate databases, duplicate schema ownership, or alter transaction atomicity while moving methods.

- `cmd/soda-identity/src/store.rs`
- `cmd/soda-identity/src/store_schema.rs`
- `cmd/soda-identity/src/store_connections.rs`
- `cmd/soda-identity/src/store_grants.rs`
- `cmd/soda-identity/src/store_leases.rs`
- `cmd/soda-identity/src/store_executions.rs`
- `cmd/soda-identity/src/store_events.rs`
- `cmd/soda-identity/src/store_tests.rs`

Evidence: 13-117,576-587,704-710,729-772: placeholder bind, Store/Tx query/transaction, changed and Param encoding -> store.rs; 118-206,588-639: grant-key/schema admission/bootstrap, load_schema_version/required columns/trigger checks -> store_schema.rs; 207-333: encrypted save_connection/read credential/list/available/state -> store_connections.rs; 334-412: grant insert/list/read/revoke -> store_grants.rs; 413-488,544-555,668-703: reserve/register/return/forget lease and transactional maintain_credential -> store_leases.rs; 489-543: execution/admit_execution/observe_execution -> store_executions.rs; 556-575,640-667,711-728: events/append_event/lease_event -> store_events.rs; 773-798: placeholder and bytea encoding cases -> store_tests.rs.

## rust/soda-identity/src/strict.rs

Observed size: 402 lines, including tests where embedded. Keep the cohesive 314-line request codec: bounded UTF-8 object decode, decoded duplicate-key scanner, case remapping and known-field checking are one admission path. Extract the 86-line test module. The total only barely exceeds 400 because of tests; do not create multiple scanners or replace it with a different duplicate/case/null policy.

- `cmd/soda-identity/src/strict.rs`
- `cmd/soda-identity/src/strict_tests.rs`

Evidence: 1-86: decode/remap_case/check_known_fields and declared strict request semantics; 87-314: Scanner lexical handling/check_unique_keys/check_value recursion; 315-402: strict shape/size/casefold/decoded-Unicode duplicate vectors -> strict_tests.rs.

## rust/soda-identity/src/wire.rs

Observed size: 981 lines, including tests where embedded. Split the 833-line production wire mirror by existing record responsibility, keeping canonical field names/order/string integers/null semantics and provider Connection/Enrollment reuse. wire.rs owns Request/DeliveryWire and provider/constants; wire_time owns UnixTime/date formatting; wire_scalars owns current int/null/base64 serde routines; grant/acquisition and lease/binding/execution records each keep their validators/digest; wire_errors keeps the fixed failure taxonomy. The 147-line tests can remain a single exact-wire/time/digest module.

- `cmd/soda-identity/src/wire.rs`
- `cmd/soda-identity/src/wire_time.rs`
- `cmd/soda-identity/src/wire_scalars.rs`
- `cmd/soda-identity/src/wire_grants.rs`
- `cmd/soda-identity/src/wire_execution.rs`
- `cmd/soda-identity/src/wire_errors.rs`
- `cmd/soda-identity/src/wire_tests.rs`

Evidence: 1-23,578-631,655-661: existing provider record reuse/constants, Request including its derive/credential field and DeliveryWire including its derive -> wire.rs; preserve the single Connection/Enrollment definitions under providers/types.rs; 24-219: UnixTime/civil date math/parse_rfc3339_nano/format_rfc3339_nano -> wire_time.rs; scalar-codec documentation starts separately at221; 221-354,462-464,633-653: complete int-string/null/base64 serde modules, is_zero_i32 and complete base64_bytes_option module -> wire_scalars.rs; 356-410,747-775,817-832: Grant/GrantRequest/AcquireRequest including derive attributes, validation and acquisition_digest -> wire_grants.rs; 412-460,466-576,777-815: Binding/Lease/Execution/Event including derive attributes and binding/execution validation -> wire_execution.rs; 663-745: ErrorKind/Error including derive attributes, mapping and conversions -> wire_errors.rs; 834-981: cfg(test) descendant module for Go time/wire/base64/null/provider ID/acquisition digest vectors -> wire_tests.rs.

Open detail: Module imports and serde with/deserialize_with/skip_serializing_if paths must follow each complete codec declaration. Keep tests as a descendant of the owning wire module and preserve one Rust definition per existing record; moving codecs does not authorize a new wire shape or forwarding DTO.

## rust/soda-identity/tests/broker.rs

Observed size at 0d8d3b8e: 634 lines, including tests where embedded. This whole 634-line file is integration-test code, not 203 lines of production before the first #[test]. Extract only the existing environment-gated PostgreSQL/StubProvider/StubRuntime fixture support into tests/common/mod.rs; group persistent custody/fencing/revoke checks in broker.rs, enrollment-to-lease path in enrollment.rs, and admin/runtime HTTP plus dead-listener shutdown in http.rs. Keep the same real broker/store under test, fixture opt-in, native cleanup and skips; do not create a new test harness.

- `cmd/soda-identity/tests/broker.rs`
- `cmd/soda-identity/tests/enrollment.rs`
- `cmd/soda-identity/tests/http.rs`
- `cmd/soda-identity/tests/common/mod.rs`

Evidence at 0d8d3b8e: 1–203: existing PostgreSQL/StubProvider/StubRuntime fixture support -> tests/common/mod.rs; 204–237,305–458: persistent custody/delegation, Muse retirement, execution fencing, post-reconcile repeated closure and revoke checks -> tests/broker.rs; 238–304: enrollment lifecycle -> tests/enrollment.rs; 459–634: HTTP admission and dead-listener lifetime checks -> tests/http.rs. These remain source-inspected integration scenarios, not executed native proof.


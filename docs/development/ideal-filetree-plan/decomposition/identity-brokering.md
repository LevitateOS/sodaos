# Identity brokering

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

The [library-adoption allocation](../library-adoption.md#finding-allocation)
supersedes pending generic-engine preservation and extraction below. Historical
source sizes, line allocations and completed moves remain evidence; they are
not requirements to keep the extracted engines. Follow the existing packet
owners and [readiness gates](../library-adoption.md#readiness-gates), retaining
one Controller/State and provider-specific custody.

## rust/identity-providers/src/codex.rs

Observed size: 702 lines, including tests where embedded. Separate the existing pinned-provider/process/enrollment lifecycle from app-server request/reply dispatch and notification parsing. Keep Config admission, environment filtering, binary digest and version checks together in codex_config.rs. codex.rs retains Provider and Session ownership, process stop before credential reading, and private tmpfs cleanup. Move the embedded tests without changing their fixtures. All files stay in the same provider crate; no Soda grant policy moves into this provider.

- `cmd/soda-identity/src/providers/codex/mod.rs`
- `cmd/soda-identity/src/providers/codex/protocol.rs`
- `cmd/soda-identity/src/providers/codex/config.rs`
- `cmd/soda-identity/src/providers/codex/tests.rs`

Evidence: 21-157: Config/Provider, new/start/create_session and split_env -> codex.rs and codex_config.rs; 161-242,304-397: Inner/Session, snapshot/cancel, device enrollment, finish/account/credential_file/stop/close -> codex.rs; 180-190,243-302,400-500: Message, send/call/protocol_error/read_loop/notify -> codex_protocol.rs; 502-565: environment/filter_env/validate_config/check_binary/check_version -> codex_config.rs; 566-702: environment/config fixtures, managed enrollment, cancellation, disabled-device tests -> codex_tests.rs.

Completed provider extraction remains complete. JSON01/PROC01/FS01/TMP01
followups replace equal codec/capture/file/allocation mechanics only; pinned
provider admission, protocol dispatch, stop-before-credential-read and private
tmpfs custody remain with this provider. The native provider processes stay
distinct; adoption does not merge their execution or enrollment policies.

## rust/identity-providers/src/muse.rs

Observed size: 561 lines, including tests where embedded. The production provider is 347 lines and owns one native enrollment lifecycle. Its process, prompt recognition, immutable subscription validation and private credential-file checks are cohesive; do not invent additional provider layers merely because tests bring the whole file to 561 lines. Extract the 213-line test module, keeping its real CLI-script fixtures and its existing environment/config/digest/version assertions.

- `cmd/soda-identity/src/providers/muse/mod.rs`
- `cmd/soda-identity/src/providers/muse/tests.rs`

Evidence: 22-163: Config/Provider/Inner/Session, native login process, finish and confirmed close; 165-242: bounded prompt read_loop/complete/scan_device_url and fixed environment; 244-346: pinned configuration/binary/version checks plus credential_valid/credential_file; 348-561: native_subscription_and_private_file, environment/device prompt, digest/version, presentation-private fixtures -> muse_tests.rs.

## rust/soda-identity-compose/src/main.rs

Observed size: 910 lines, including tests where embedded. Keep main as the explicit one-service registration sequence. Retain the options, Compose child attribution, registration and launch-record owners already extracted. JSON01 replaces generic scalar/skip/string parsing and emission with serde-backed launch adapters; launch_json.rs is an engine retirement, not a future extraction target. RNG01/TMP01/FS01 supply randomness, allocation and rooted-file mechanics while registration retains its deliberate tmpfs/account traversal modes, immutable child binding, launch-only socket access and cleanup. Keep the small real option selector under CLI01. The historical production portion is 750 lines and tests 158; this adoption does not add Compose orchestration features.

- `cmd/soda-identity-compose/src/main.rs`
- `cmd/soda-identity-compose/src/options.rs`
- `cmd/soda-identity-compose/src/compose.rs`
- `cmd/soda-identity-compose/src/registration.rs`
- `cmd/soda-identity-compose/src/launch_wire.rs`
- `cmd/soda-identity-compose/src/compose_tests.rs`

Evidence at f7: 20–47 explicit sequence/load-options→main.rs; Options derive12 and struct13–18 plus complete flag/validation/usage49–165→options.rs; launch_compose245–287, write_override316–345 and child-attribution comments697/function698–750→compose.rs; root/random/mkdir166–244, account288–315 and complete registration626–694 (including its internal FD Guard/Drop)→registration.rs; string codec346–371, single NestedRegistration373–378, request/comment380–396 and exit-decode/comment397–469→launch_wire.rs; scalar/skip helpers471–624→launch_json.rs; cfg(test)751/module752 and all real cases753–909→compose_tests.rs, with original910 closing that test root. Keep one MUSE_LAUNCH_SOCKET9 in the common main owner, imported by compose and registration; TMPFS_MAGIC10 belongs registration. NestedRegistration and its fields use bounded parent-only imports for main/registration, with no duplicate DTO or public API. Tests import the actual moved subjects privately. [I09](../reviews/I09.md) records the independent defining-fit challenge and preserves canonical H03 parser corrections; Go-parity comments do not override the required wire semantics. No Compose, socket or test operation was run.

The historical launch_json allocation above records the completed seam only.
Its replacement follows JSON01's actual launch-record profile and coupled
caller/test closure; foreign error wording is not an independent requirement.

## rust/soda-identity/src/control.rs

Observed size: 837 lines, including tests where embedded. The 837-line controller has no embedded tests; separate its already-existing broker concerns while retaining the single mutex-protected State and one Controller. Provider adapter traits/mapping are transport-neutral broker integration; enrollment and grants remain distinct from acquisition/execution admission, registration/credential delivery, and confirmed retirement/reconciliation. Keep all current lock/transaction and native runtime attestation sequencing. This is a file split within one broker crate, not independent controllers or a new authorization layer. Keep the small existing provider/runtime adapter block in control.rs; providers/mod.rs is the one root for the consolidated native provider implementations.

- `cmd/soda-identity/src/control.rs` — Provider/runtime traits and existing session adapters remain with controller construction, State, lock ownership and shared controller helpers.
- `cmd/soda-identity/src/enrollment.rs`
- `cmd/soda-identity/src/grants.rs`
- `cmd/soda-identity/src/acquisition.rs`
- `cmd/soda-identity/src/registration.rs`
- `cmd/soda-identity/src/retirement.rs`

Evidence: 13-85: provider/runtime traits, error mapping and Codex/Muse session adapters -> control.rs; 86-165,800-837: EnrollmentEntry/State/Controller, construction/lock/ownership, close/error join/random-ID/zeroize -> control.rs; 133-143,166-272: pending/start/read/cancel enrollment -> enrollment.rs; 273-298,705-739: grant listing/create plus connection/grant revoke -> grants.rs; retirement helpers remain called from retirement.rs; 298-501: lease list, acquire/admit/replay/reserve/get/close execution and reservation authorization -> acquisition.rs; 502-632: register/registration authority/execution binding/terminal attestation/release and return_lease -> registration.rs; 633-704,740-799: uncertain/end/finish/reconcile/sweep/reject/retire_connection -> retirement.rs.

RNG01 replaces predictable entropy fallback with the admitted library and a
fail-closed result. Controller locking, grant/lease decisions, zeroization and
confirmed retirement remain here; a primitive replacement does not authorize
another controller or weaken native execution fencing.

## rust/soda-identity/src/http.rs

Observed size: 494 lines, including tests where embedded. Retain socket serving/shutdown/inflight and admin/runtime route ownership. N1 replaces generic HTTP parsing/framing/encoding through its admitted library adapter; http_wire.rs may retain only Soda admission glue after that cutover. Production was 464 lines and tests 28. Keep runtime_allowed enforcement on the existing service socket path and POST/query/Origin checks intact; route files do not own broker custody or policy. Prove the selected Unix transport, bounds and lifetime interface before protocol adoption, as specified in the library chapter.

- `cmd/soda-identity/src/http.rs`
- `cmd/soda-identity/src/http_wire.rs`
- `cmd/soda-identity/src/http_routes.rs`
- `cmd/soda-identity/src/http_tests.rs`

Evidence: 73-151: Server construction/serve/handle_connection, shutdown/inflight and dead-listener behavior -> http.rs; 152-283,431-464: Admission/HttpRequest/read_request/percent_decode/success_response/error_response -> http_wire.rs; 1-72,284-430: allowed wire fields/nested specs, dispatch/error mapping, admin and runtime routes -> http_routes.rs; 465-494: percent-path, error-body and success-envelope cases -> http_tests.rs.

## rust/soda-identity/src/pg.rs

Observed size: 512 lines, including tests where embedded. PG01 replaces the custom DSN, transport/authentication/message and row/query engines with the selected PostgreSQL driver. Keep only the small connection-policy/error adapter needed by Store; pg_dsn.rs and pg_query.rs retire with their actual callers instead of receiving further structural work. The historical production portion is 475 lines and tests 35. Complete SQL01 native-parameter cleanup before driver cutover. L00 proved the tokio-postgres deadline/cancel/discard/reconnect and whole-transaction guard interfaces locally; retain their small internal runtime facade behind the existing repository-facing boundary. A synchronous postgres-only migration is held because its blocking API provides no operation deadline hook. Actual DSN/auth/transport and Store/Tx acceptance still belongs to L08. Pooling, retries, an ORM and another database are outside this allocation.

- `cmd/soda-identity/src/pg.rs`
- `cmd/soda-identity/src/pg_tests.rs`

Evidence: 19-125: Dsn parse/percent_decode -> pg_dsn.rs; 126-146,187-327,456-475: Stream/Client/connect/send/receive/authenticate/authenticate_scram and backend error -> pg.rs; 147-186,328-455: Row scalar/bytea conversions and query/simple/command_count -> pg_query.rs; 476-512: DSN forms and command-tag row counts -> pg_tests.rs.

## rust/soda-identity/src/store.rs

Observed size: 798 lines, including tests where embedded. Retain SQL locality by connection custody, grants, leases, execution records and audit events. SQL01 deletes question-mark rewriting in favor of native PostgreSQL parameters; PG01 supplies typed binding/row operations and a driver transaction guard covering the complete transaction. Store retains domain atomicity, authority and error/affected-row interpretation. Preserve internal/store/schema.go as the authoritative schema and schema.rs as its mechanically synchronized representation. The historical production portion is 772 lines and tests 24. Controller locking currently serializes the traced production paths; do not claim a demonstrated live interleaving solely from the weaker public Store transaction API. No separate database or schema authority is introduced.

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

Observed size: 402 lines, including tests where embedded. JSON01 replaces the lexical/recursive duplicate scanner with serde-backed admission and small record/profile adapters. strict.rs retains the request object/full-consumption, decoded-duplicate, unknown-field, alias/null, size and depth policy; do not erase duplicates or choose alias precedence through a Value map before enforcing the profile. The historical request codec was 314 lines and its test module had 86 lines. Keep real policy cases with the replacement subject; exact Go diagnostics and scanner offsets do not justify retaining another lexer.

- `cmd/soda-identity/src/strict.rs`
- `cmd/soda-identity/src/strict_tests.rs`

Evidence: 1-86: decode/remap_case/check_known_fields and declared strict request semantics; 87-314: Scanner lexical handling/check_unique_keys/check_value recursion; 315-402: strict shape/size/casefold/decoded-Unicode duplicate vectors -> strict_tests.rs.

## rust/soda-identity/src/wire.rs

Observed size: 981 lines, including tests where embedded. Retain the record responsibilities already extracted from the historical 833-line production mirror and 147-line tests: Request/DeliveryWire, shared provider types, grant/acquisition and lease/binding/execution validators/digests, and fixed error taxonomy. JSON01 retains small integer/null/bytes adapters for admitted records; selected Base64 and N9 time libraries replace generic codec/calendar engines in wire_scalars/wire_time after their actual profiles are fixed. Preserve field semantics and the acquisition-digest contract, rather than assuming every incidental Go formatting quirk is required. Library types do not create forwarding DTOs or duplicate provider definitions.

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

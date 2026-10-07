# Identity broker protocol and policy

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
Selected source selectors below include the later typed-PG, direct-host-DTO and
Identity HTTP decoder follow-ups; this is a focused refresh, not a full map census.
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-475e779cff71"></a>

## [cmd/soda-identity/src/acquisition.rs](../../../../../cmd/soda-identity/src/acquisition.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 67–103, 192–216; current module/import/attribute shell; declaration replay_acquisition; declaration authorize_reservation | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/acquisition.rs into its current native target.; replay_acquisition: implement the current identity enrollment and owner consent duty in acquisition.rs.; authorize_reservation: implement the current identity enrollment and owner consent duty in acquisition.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 9–65, 105–141; declaration leases; declaration acquire; declaration admit_execution; declaration reserve_execution_lease; declaration get_execution | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Acquisition links the lease, reserved event and execution mapping through one Store transaction. Uncertain COMMIT is re-read rather than automatically replayed. — Current named units/source consumers; retained normalized source evidence records each selector |
| 143–190; declaration close_execution | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | close_execution preserves the terminal execution fence and retains the lease association on non-NotFound close-read failures, returning Uncertain. — current source cmd/soda-identity/src/acquisition.rs; current function inspected |

<a id="coverage-3e4294f35465"></a>
<a id="rustsoda-identitysrccontrolrs-1"></a>
<a id="coverage-17ec4969c422"></a>

## [cmd/soda-identity/src/control.rs](../../../../../cmd/soda-identity/src/control.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 15–21, 25–45, 55–57, 67–150, 165–203; current module/import/attribute shell; declaration EnrollmentSession; declaration snapshot; declaration Provider; declaration start; declaration Runtime; declaration validate; declaration stop; declaration map_provider_error; declaration CodexProvider; declaration MuseProvider; declaration CodexSession; declaration MuseSession; declaration EnrollmentEntry; fields owner, label, provider_id, session, result; declaration State; fields store, providers, runtime, enrollments; declaration Controller; fields state; declaration new; declaration lock; declaration owned; declaration connections; declaration available; declaration join_errors; declaration new_id; declaration new_id_with; declaration zeroize; declaration tests; declaration identity_id_fails_closed_after_partial_entropy_write | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/control.rs into its current native target.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 11–14, 22–24, 46–54, 58–66, 151–164; declaration finish; declaration close | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | finish: implement the current completion, revocation, and reconciliation duty in control.rs.; close: implement the current completion, revocation, and reconciliation duty in control.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-386bd3f38897"></a>

## [cmd/soda-identity/src/crypto.rs](../../../../../cmd/soda-identity/src/crypto.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; current module/import/attribute shell | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/crypto.rs into its current native target. — current source cmd/soda-identity/src/crypto.rs; Cargo target and callers |
| 8–135; declaration KEY_BINDING; declaration GRANT_KEY_ERROR; declaration GrantCipher; fields cipher; declaration new; declaration seal; declaration seal_with_random; declaration open; declaration nonce_size; declaration tests; declaration rejects_short_keys; declaration round_trip_with_binding; declaration nist_vector; declaration seal_fails_closed_after_partial_entropy_write; declaration hex | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | KEY_BINDING: implement the current encrypted credential custody and cryptographic binding duty in crypto.rs.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d877157ea310"></a>

## [cmd/soda-identity/src/enrollment.rs](../../../../../cmd/soda-identity/src/enrollment.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123; current module/import/attribute shell; declaration pending_enrollment; declaration start_enrollment; declaration enrollment; declaration cancel_enrollment | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/enrollment.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b79445f71583"></a>

## [cmd/soda-identity/src/grants.rs](../../../../../cmd/soda-identity/src/grants.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; current module/import/attribute shell; declaration grants; declaration create_grant; declaration revoke; declaration revoke_grant | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/grants.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-72e250c9050e"></a>
<a id="rustsoda-identitysrchttprs-1"></a>
<a id="coverage-05773aac6313"></a>

## [cmd/soda-identity/src/http.rs](../../../../../cmd/soda-identity/src/http.rs)

Current source responsibilities were split by present listener/adapter, wire representation, or product dispatch unit; spans cover the full current file extent.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–308; HTTP module imports and listener/service shell lines 1-21; MAX_CONNECTIONS accepted-listener bound; MAX_BACKENDS and BODY_TIMEOUT request admission limits; Server data and constructor; Server::serve Unix listener and Tokio runtime; serve_runtime listener admission, in-flight tracking, and shutdown joins; InflightGuard and Drop request accounting; serve_connection Hyper HTTP/1 connection adapter; respond Hyper request framing, bounded body read, and dispatch handoff; wait_shutdown listener/task wait helper | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Wire Hyper HTTP/1, Tokio Unix listener, task lifecycle, HTTP router and Controller into the identity Unix-socket service.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8b23aa8070a6"></a>

## [cmd/soda-identity/src/http_routes.rs](../../../../../cmd/soda-identity/src/http_routes.rs)

Current source responsibilities were split by present listener/adapter, wire representation, or product dispatch unit; spans cover the full current file extent.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27, 32–56, 94–107, 154–156; imports/module shell, body/header limits, HTTP method/query/origin admission, route handoff and error mapping, route/runtime selector boundaries and unknown-path refusal | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Wire the Unix HTTP adapter to Controller and response adapters; preserve admission and error mapping. — Current route/controller callers inspected |
| 28–31; dispatch bounded HTTP request admission | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Decode the bounded request directly into the exact HTTP wire DTO after UTF-8, decoded-duplicate, depth and EOF checks; DTOs reject unknown exact-case fields. Settings retains its separate profile-aware decoder. — `strict::decode_typed` consumes the same `MAX_BODY` admission before Controller dispatch |
| 58, 64–75; route /connections owner listing; enrollment start/read/cancel | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | List existing owner connections and manage owner enrollment. — Current named route selectors inspected |
| 59, 76–84; route /available; route /grants and /grant/create | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Derive actor/Project connection availability and list/create owner grants. — Controller caller and scoped Store operation retained |
| 60–63, 85–93, 132–146, 150–153; revoke, grant revoke, lease end, return, reconcile and execution close routes | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Preserve existing revocation, credential-return and terminal lifecycle actions. — Current route/controller callers inspected |
| 89–91, 109–114, 147–149; lease list, acquire and execution lookup | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | List leases, admit executions and retrieve execution state. — Current route/controller callers inspected |
| 115–131; runtime register/reject binding and private credential delivery | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Register or reject the existing binding contract and retain private credential delivery. — Controller.register/reject callers inspected |

<a id="coverage-a1637e610af2"></a>

## [cmd/soda-identity/src/http_tests.rs](../../../../../cmd/soda-identity/src/http_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 12–49; current module/import/attribute shell; declaration error_envelope_preserves_status_headers_and_lf; declaration success_envelope_preserves_cache_policy_and_lf | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private HTTP adapter module/test wiring; codec syntax assertions retain H03.; Asserts error-response status, headers and final LF emitted by the private HTTP response adapter.; Asserts success-response cache policy and final LF emitted by the private HTTP response adapter. — Current named units/source consumers; retained normalized source evidence records each selector |
| 4–11; declaration percent_paths_decode | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | percent_paths_decode: implement the current wire decoding, encoding, and representation conversion duty in http_tests.rs. — current source cmd/soda-identity/src/http_tests.rs; lines 4-11; module/caller wiring inspected |

<a id="coverage-5d0d8c064484"></a>

## [cmd/soda-identity/src/http_wire.rs](../../../../../cmd/soda-identity/src/http_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 47–73; current module/import/attribute shell; declaration MAX_BODY; declaration MAX_HEADER; declaration HttpRequest; fields method, path, query, origin; declaration success_response; declaration error_response; declaration tests | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private HTTP adapter module/test wiring; codec syntax assertions retain H03.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 18–46; declaration percent_decode; declaration valid_percent_escapes | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | percent_decode: implement the current wire decoding, encoding, and representation conversion duty in http_wire.rs.; Rejects malformed percent escapes as a path-decoding syntax contract. — current source cmd/soda-identity/src/http_wire.rs; lines 18-27; module/caller wiring inspected; Independent current-body/caller responsibility challenge: http_wire.rs:28-46; percent_decode uses it; http.rs:245 invokes the decoder |

<a id="coverage-69d26a2117d8"></a>

## [cmd/soda-identity/src/lib.rs](../../../../../cmd/soda-identity/src/lib.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 8–9, 18–19; current module/import/attribute shell; declaration acquisition; declaration enrollment; declaration grants; declaration providers; declaration registration | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/lib.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 6; declaration control | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | control: implement the current execution admission and lease fencing duty in lib.rs. — current source cmd/soda-identity/src/lib.rs; lines 6-6; module/caller wiring inspected |
| 7; declaration crypto | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | crypto: implement the current encrypted credential custody and cryptographic binding duty in lib.rs. — current source cmd/soda-identity/src/lib.rs; lines 7-7; module/caller wiring inspected |
| 10; declaration http | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | http: implement the current Unix-socket service and process lifetime duty in lib.rs. — current source cmd/soda-identity/src/lib.rs; lines 10-10; module/caller wiring inspected |
| 11–12, 30–36; declaration http_routes; declaration http_wire; declaration strict; declaration wire; declaration wire_errors; declaration wire_execution; declaration wire_grants; declaration wire_scalars; declaration wire_time | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | http_routes: implement the current wire decoding, encoding, and representation conversion duty in lib.rs.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 13–17, 22–29; declaration pg; declaration pg_dsn; declaration pg_query; declaration pg_tests; declaration schema; declaration store; declaration store_connections; declaration store_events; declaration store_executions; declaration store_grants; declaration store_leases; declaration store_schema | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | pg: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in lib.rs.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20; declaration retirement | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | retirement: implement the current completion, revocation, and reconciliation duty in lib.rs. — current source cmd/soda-identity/src/lib.rs; lines 20-20; module/caller wiring inspected |
| 21; declaration runtime | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | runtime: implement the current native binding and private delivery duty in lib.rs. — current source cmd/soda-identity/src/lib.rs; lines 21-21; module/caller wiring inspected |

<a id="coverage-749b2e8df0a8"></a>
<a id="rustsoda-identitysrcpgrs-1"></a>
<a id="coverage-8d9410b54095"></a>

## [cmd/soda-identity/src/pg.rs](../../../../../cmd/soda-identity/src/pg.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; current module/import/attribute shell | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/pg.rs into its current native target. — current source cmd/soda-identity/src/pg.rs; Cargo target and callers |
| 6–25; declaration Connection; fields client, driver; declaration connect; declaration pg_error | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Connection: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in pg.rs.; connect: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in pg.rs.; pg_error: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in pg.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-80ecd53dca3e"></a>

## [cmd/soda-identity/src/pg_dsn.rs](../../../../../cmd/soda-identity/src/pg_dsn.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37; current module/import/attribute shell; declaration Dsn; declaration parse | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/pg_dsn.rs into its current native target.; Dsn: implement the current identity enrollment and owner consent duty in pg_dsn.rs.; parse: implement the current identity enrollment and owner consent duty in pg_dsn.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-adc2bf00b6bd"></a>

## [cmd/soda-identity/src/pg_query.rs](../../../../../cmd/soda-identity/src/pg_query.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; current module/import/attribute shell | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/pg_query.rs into its current native target. — current source cmd/soda-identity/src/pg_query.rs; Cargo target and callers |
| 8–40; declaration DRIVER_JOIN_BUDGET; declaration Outcome; declaration bounded | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | The bounded driver join owns operation timeout, query cancellation, response draining and error mapping; the generic Field/Row conversion layer has been removed. — Current named units/source consumers; actual Store callsites inspected |

<a id="coverage-008763f67f4b"></a>

## [cmd/soda-identity/src/pg_tests.rs](../../../../../cmd/soda-identity/src/pg_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; current module/import/attribute shell; declaration TEST_LOCK; declaration dsn_policy_keeps_uri_no_tls_and_socket_support | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/pg_tests.rs into its current native target.; TEST_LOCK: implement the current identity enrollment and owner consent duty in pg_tests.rs.; dsn_policy_keeps_uri_no_tls_and_socket_support: implement the current identity enrollment and owner consent duty in pg_tests.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 26–232, 233–238; declaration integration_store; declaration integration_store_for_dsn; declaration store_authenticates_over_unix_and_loopback_tcp; declaration store_cancellation_discards_session_and_next_operation_reconnects; declaration swallowed_transaction_errors_cannot_commit_or_return_success; declaration typed_postgres_rows_and_transaction_guard_match_store_contract; declaration schema_integer_rejects_values_outside_postgres_integer_range | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Real driver checks cover Unix/TCP authentication, canceled-session discard/reconnect, transaction failure containment, typed nullable/scalar/JSONB rows and borrowed parameters; local schema-range checks retain int4 bounds. — Current source `cmd/soda-identity/src/pg_tests.rs`; 15 actual PostgreSQL/broker checks pass |

<a id="coverage-7a9f29a78915"></a>

## [cmd/soda-identity/src/registration.rs](../../../../../cmd/soda-identity/src/registration.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 50–64, 82–116; current module/import/attribute shell; declaration observe_terminal; declaration registration; declaration registration_authority | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/registration.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 9–49, 65–81, 117–139; declaration register; declaration registration_execution; declaration observe_execution_binding; declaration release_execution_lease; declaration return_lease | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | register: implement the current execution admission and lease fencing duty in registration.rs.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-aee4e5e339a8"></a>

## [cmd/soda-identity/src/retirement.rs](../../../../../cmd/soda-identity/src/retirement.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31, 124–133; current module/import/attribute shell; declaration uncertain; declaration end; declaration reject | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/retirement.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 32–40; declaration end_lease | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | end_lease: implement the current execution admission and lease fencing duty in retirement.rs. — current source cmd/soda-identity/src/retirement.rs; lines 32-40; module/caller wiring inspected |
| 41–123; declaration finish_lease; declaration reconcile_lease; declaration reconcile; declaration sweep; declaration sweep_lease | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | finish_lease: implement the current completion, revocation, and reconciliation duty in retirement.rs.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 134–146; declaration retire_connection | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | retire_connection: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in retirement.rs. — current source cmd/soda-identity/src/retirement.rs; lines 134-146; module/caller wiring inspected |

<a id="coverage-e6620e3beafc"></a>
<a id="coverage-0276b9fb3e6c"></a>

## [cmd/soda-identity/src/runtime.rs](../../../../../cmd/soda-identity/src/runtime.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 276–286; current module/import/attribute shell; declaration legal_chunked_response_is_accepted_by_host_client | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/runtime.rs into its current native target.; legal_chunked_response_is_accepted_by_host_client: implement the current identity enrollment and owner consent duty in runtime.rs. — current source cmd/soda-identity/src/runtime.rs; Cargo target and callers; current source cmd/soda-identity/src/runtime.rs; lines 276-286; module/caller wiring inspected |
| 8–67, 93–99; declaration CALL_TIMEOUT; declaration DEFAULT_LIMIT; declaration FINISH_LIMIT; declaration HostClient; fields socket; declaration new; declaration call; declaration default_limit | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | CALL_TIMEOUT: implement the current Unix-socket service and process lifetime duty in runtime.rs.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 68–75, 100–102, 111–257, 267–275; declaration validate; declaration tests; declaration lease; declaration stub; declaration rand_suffix; declaration delegates_both_kinds_to_host; declaration finish_rejects_null_and_oversized_bodies | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | validate: implement the current native binding and private delivery duty in runtime.rs.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 76–92, 103–110; declaration stop; declaration finish | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | stop: implement the current completion, revocation, and reconciliation duty in runtime.rs.; finish: implement the current completion, revocation, and reconciliation duty in runtime.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 258–266; declaration refuses_unbound_lease_without_host_call | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | refuses_unbound_lease_without_host_call: implement the current execution admission and lease fencing duty in runtime.rs. — current source cmd/soda-identity/src/runtime.rs; lines 258-266; module/caller wiring inspected |

<a id="coverage-fac98c2013f2"></a>
<a id="rustsoda-identitysrcstrictrs-1"></a>
<a id="coverage-012423b63d27"></a>

## [cmd/soda-identity/src/strict.rs](../../../../../cmd/soda-identity/src/strict.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 103–210; current module/import/attribute shell; declaration UniqueSeed; fields depth, top; declaration UniqueVisitor; recursive duplicate/depth scan | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Keep the shared bounded decoded-key duplicate/depth preflight used by both readers. — Current decoder callers inspected |
| 11–23, 25–50, 103–228; declaration MAX_DOCUMENT; declaration decode_typed; declaration decode; declaration remap_case; declaration check_known_fields; declaration check_unique_keys; test-module binding | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | HTTP uses bounded duplicate/depth preflight followed by direct typed DTO decoding from original UTF-8 input; Settings retains its distinct case-folding projection. Both preserve byte caps, recursive duplicates/depth and trailing-input refusal; no shared alias policy inferred. — Current HTTP and Settings callers inspected |

<a id="coverage-8d525f22e8b2"></a>

## [cmd/soda-identity/src/strict_tests.rs](../../../../../cmd/soda-identity/src/strict_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 66–140; current module/import shell, retained Settings envelope and generic profile tests | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Preserve the Settings profile's existing case-fold, alias-collision and null behavior. — Existing generic decoder regression selectors retained |
| 24–64; `typed_http_request_preserves_wire_codecs_and_rejects_unknowns` | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Cover canonical HTTP DTO codecs and exact unknown/duplicate/depth/EOF/UTF-8 refusal. — Included in corrected five-test strict selector; passed |

<a id="coverage-8cb18c62534a"></a>
<a id="rustsoda-identitysrcwirers-1"></a>
<a id="coverage-e65d2c87fcc0"></a>

## [cmd/soda-identity/src/wire.rs](../../../../../cmd/soda-identity/src/wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 33–87; current module/import/attribute shell; exact-field HTTP `Request` with `deny_unknown_fields` and unchanged scalar codecs | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Request remains the single private HTTP DTO; it now owns exact-name unknown-field refusal without changing its known-field codecs. — Current HTTP decoder caller inspected |
| 8–32, 89–99; protocol constants, provider_valid, DeliveryWire and test-module binding | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Preserve current wire constants and delivery/base64 representation. — Current named units/source consumers inspected |

<a id="coverage-91f231710f16"></a>

## [cmd/soda-identity/src/wire_errors.rs](../../../../../cmd/soda-identity/src/wire_errors.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; current module/import/attribute shell; declaration ErrorKind; declaration Error; fields kind, message; declaration denied; declaration busy; declaration stale; declaration uncertain; declaration not_found; declaration internal; declaration kind; declaration is_denied; declaration is_not_found; declaration fmt; declaration from | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire_errors.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1bebd650016a"></a>

## [cmd/soda-identity/src/wire_execution.rs](../../../../../cmd/soda-identity/src/wire_execution.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–71, 173–268; current module/import/attribute shell; exact-field Binding (`deny_unknown_fields`), Event and validators | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Binding is the shared retained DTO and now refuses unknown fields while preserving known scalar codecs. — Current HTTP and persisted-record consumers inspected |
| 72–172; declaration Lease; fields repository_id, provider_id, id, connection_id, generation, actor_id, project_id, execution_id, kind, role, deadline, grant_id, grant_revision, binding; declaration Execution; fields binding, kind, execution_id, digest, state, lease_id | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Lease and Execution retain the current execution admission/fencing wire contract. — current source fields inspected |

<a id="coverage-5b28468b9b2c"></a>

## [cmd/soda-identity/src/wire_grants.rs](../../../../../cmd/soda-identity/src/wire_grants.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–63, 110–141; module/import shell, Grant and strict GrantRequest (`deny_unknown_fields`) with unchanged validators/codecs | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | GrantRequest now owns exact HTTP nested-field refusal while preserving current scalar codecs. — Current wire and HTTP callers inspected |
| 64–109; strict AcquireRequest (`deny_unknown_fields`) and existing fields | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | AcquireRequest now owns exact HTTP nested-field refusal; deadline, integer-string and null behavior remain unchanged. — Current wire and HTTP callers inspected |
| 142–155; declaration acquisition_digest | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | acquisition_digest: implement the current wire decoding, encoding, and representation conversion duty in wire_grants.rs. — Current source inspected |

<a id="coverage-de94b64e2662"></a>

## [cmd/soda-identity/src/wire_scalars.rs](../../../../../cmd/soda-identity/src/wire_scalars.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 73–81, 113–122; current module/import/attribute shell; declaration go_std; declaration is_zero_i32; declaration base64_bytes_option | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire_scalars.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 6–72, 82–112, 123–136; declaration i64_string; declaration serialize; declaration deserialize; declaration i64_string_omitted; declaration null_tolerant; declaration string; declaration boolean; declaration integer; declaration integer32; declaration time; declaration is_zero; declaration base64_bytes; declaration encode; declaration decode | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | i64_string: implement the current wire decoding, encoding, and representation conversion duty in wire_scalars.rs.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e7c4dc6747ef"></a>

## [cmd/soda-identity/src/wire_tests.rs](../../../../../cmd/soda-identity/src/wire_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2, 62–95; current module/import/attribute shell; declaration unix_time_arithmetic_and_formatting_are_checked | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire_tests.rs into its current native target.; unix_time_arithmetic_and_formatting_are_checked: implement the current identity enrollment and owner consent duty in wire_tests.rs. — current source cmd/soda-identity/src/wire_tests.rs; Cargo target and callers; current source cmd/soda-identity/src/wire_tests.rs; lines 62-95; module/caller wiring inspected |
| 3–61, 96–203; declaration go_time_vectors_round_trip; declaration timestamps_reject_malformed_input; declaration lease_wire_shape_matches_go; declaration base64_matches_go_byte_form; declaration null_scalars_match_go_noop; declaration provider_ids_match_go; declaration digest_matches_go_acquisition | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | go_time_vectors_round_trip: implement the current wire decoding, encoding, and representation conversion duty in wire_tests.rs.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6b49519dc9ab"></a>

## [cmd/soda-identity/src/wire_time.rs](../../../../../cmd/soda-identity/src/wire_time.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 46–61; current module/import/attribute shell; declaration serialize; declaration deserialize | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire_time.rs into its current native target.; serialize: implement the current identity enrollment and owner consent duty in wire_time.rs.; deserialize: implement the current identity enrollment and owner consent duty in wire_time.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 4–45, 62–68; declaration UnixTime; fields sec, nanos; declaration now; declaration add_hours; declaration as_system_time; declaration parse_rfc3339_nano; declaration format_rfc3339_nano | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | UnixTime: implement the current wire decoding, encoding, and representation conversion duty in wire_time.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-11786437dd58"></a>

## [cmd/soda-identity/tests/broker.rs](../../../../../cmd/soda-identity/tests/broker.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; current module/import shell and test helpers | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Wire existing real PostgreSQL broker fixtures. — Current target and helper callers inspected |
| 24–59; `store_round_trip` | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Preserve encrypted credential persistence, owner listing and grant visibility. — Current PostgreSQL test inspected |
| 60–96, 609–632; `connection_listing_refuses_oversized_rows_and_aggregate_output`, `connection_lists_refuse_relational_json_owner_and_state_mismatch` | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Real PG cases cover exact response bounds, oversized rows/aggregate and relational-to-JSON identity refusal. — `37dc05fd`; included in the 20-case bound suite |
| 97–131, 132–274, 331–491, 549–608; keyset reconciliation during page deletion and retained lease close/revoke/recovery/malformed-selector regressions | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Preserve I06 fence and complete authority scans; page deletion cannot hide remaining leases and malformed selectors refuse retirement. — Actual PostgreSQL cases in `37dc05fd`; no native/provider qualification |
| 275–330; `acquire_reservation_and_execution_link_roll_back_together` | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Real PostgreSQL regression verifies atomic rollback of lease, reserved event and execution link. — Current test inspected |
| 492–548; `grant_revoke_preserves_ungranted_and_other_grant_leases` | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Retain grant-scoped lease behavior under revocation. — Current PostgreSQL test inspected |

<a id="coverage-56faffb6cb45"></a>

## [cmd/soda-identity/tests/common/mod.rs](../../../../../cmd/soda-identity/tests/common/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43, 93–113, 121–143, 157–166, 175–204, 211–264; fixture module, PostgreSQL setup/teardown, synthetic connection/subscription, provider/runtime stubs and constructors | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | The broker integration fixture uses stub provider/runtime implementations; these tests do not qualify native provider/runtime execution. — Current named units/source consumers; retained normalized source evidence records each selector |
| 114–120, 144–155; declaration store; declaration connection | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | store and connection helpers exercise the native PostgreSQL query/transaction fixture setup. — current source cmd/soda-identity/tests/common/mod.rs; current helper spans inspected |
| 167–173, 205–209; declaration finish; declaration close | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Stub session/runtime retirement helpers used by broker integration tests. — current source cmd/soda-identity/tests/common/mod.rs; current helper spans inspected |

<a id="coverage-fd728676de4e"></a>

## [cmd/soda-identity/tests/enrollment.rs](../../../../../cmd/soda-identity/tests/enrollment.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; current module/import/attribute shell; declaration common | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/tests/enrollment.rs into its current native target.; common: implement the current identity enrollment and owner consent duty in enrollment.rs. — current source cmd/soda-identity/tests/enrollment.rs; Cargo target and callers; current source cmd/soda-identity/tests/enrollment.rs; lines 3-7; module/caller wiring inspected |
| 8–73; declaration enrollment_to_lease_lifecycle | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | enrollment_to_lease_lifecycle: implement the current execution admission and lease fencing duty in enrollment.rs. — current source cmd/soda-identity/tests/enrollment.rs; lines 8-73; module/caller wiring inspected |

<a id="coverage-4d018cd89538"></a>

## [cmd/soda-identity/tests/http.rs](../../../../../cmd/soda-identity/tests/http.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 174–225, 237–418; module/import shell, dead-listener and remaining shutdown/join integration subjects | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Existing server lifecycle and backend-join integration subjects remain in the actual listener test owner. — Current selectors inspected |
| 9–173, 226–236; `http_admission_matches_go` and shutdown integration setup | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Real server/PG fixture checks canonical typed request admission and exact nested/root unknown, duplicate and depth refusal. — Three actual Unix HTTP checks passed; no fixture skips |
| 247–254, 278–316; declaration finish; declaration close | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | finish: implement the current completion, revocation, and reconciliation duty in http.rs.; close: implement the current completion, revocation, and reconciliation duty in http.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

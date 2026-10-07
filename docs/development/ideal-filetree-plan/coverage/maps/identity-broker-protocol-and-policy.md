# Identity broker protocol and policy

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-475e779cff71"></a>

## [cmd/soda-identity/src/acquisition.rs](../../../../../cmd/soda-identity/src/acquisition.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8, 70–107, 188–212; current module/import/attribute shell; declaration replay_acquisition; declaration authorize_reservation | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/acquisition.rs into its current native target.; replay_acquisition: implement the current identity enrollment and owner consent duty in acquisition.rs.; authorize_reservation: implement the current identity enrollment and owner consent duty in acquisition.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 9–69, 108–149; declaration leases; declaration acquire; declaration admit_execution; declaration reserve_execution_lease; declaration get_execution | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | leases: implement the current execution admission and lease fencing duty in acquisition.rs.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 150–187; declaration close_execution | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | close_execution: implement the current completion, revocation, and reconciliation duty in acquisition.rs. — current source cmd/soda-identity/src/acquisition.rs; lines 150-187; module/caller wiring inspected |

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
| 1–18, 73–81, 86–110, 156–169, 216–218; HTTP router imports and module shell; MAX_BODY and HEADER_TIMEOUT HTTP admission limits; dispatch HTTP method/query/origin admission; dispatch handoff to product route selector; dispatch success and ErrorKind-to-HTTP response mapping; route boundary comment and signature; route runtime-listener gate and runtime selector handoff; route_runtime adapter signature; route_runtime unknown endpoint refusal | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Wire the Unix HTTP adapter to Controller, strict wire decoder, request and response adapters.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 19–72, 82–85; REQUEST_FIELDS, GRANT_FIELDS, ACQUIRE_FIELDS, BINDING_FIELDS, NESTED identity wire allowlists; dispatch strict request decode using wire allowlists | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Define accepted top-level and nested JSON identity request field shapes.; Decode the bounded JSON request into the identity wire Request representation. — strict::decode in dispatch consumes top-level and nested allowlists before Controller dispatch.; strict::decode consumes REQUEST_FIELDS and NESTED before product route selection. |
| 111–114, 121–133; route /connections owner connection listing branch; route /enrollment/start and /enrollment/read branches; route /enrollment/cancel branch | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | List the owner’s existing identity connections.; Start an owner enrollment and retrieve its enrollment state.; Cancel an owner enrollment. — Current named units/source consumers; retained normalized source evidence records each selector |
| 115–117, 134–144; route /available delegated connection availability branch; route /grants and /grant/create branches | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Derive which owned or delegated connections are available to the requested Project.; List and create owner-controlled delegation grants. — Calls Controller.available(owner_id, project_id), whose control implementation delegates to the actor/Project scoped availability operation.; Calls Controller.grants and Controller.create_grant; missing grant input is denied. |
| 118–120, 145–148, 152–155, 194–208, 212–215; route /revoke connection retirement branch; route /grant/revoke branch; route /lease/end branch; route_runtime /return credential return branch; route_runtime /reconcile-lease branch; route_runtime /execution/close branch | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Revoke an owned identity connection and return no response body.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 149–151, 170–176, 209–211; route /leases listing branch; route_runtime /acquire execution admission branch; route_runtime /execution/get branch | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | List leases for the owner and connection.; Admit an execution request and produce its lease.; Retrieve execution admission state. — Current named units/source consumers; retained normalized source evidence records each selector |
| 177–193; route_runtime /register native binding and credential delivery branch; route_runtime /reject binding branch | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Register an attested binding and serialize lease plus private credential delivery.; Reject a binding under the identity binding contract. — Calls Controller.register and encodes DeliveryWire.; Calls Controller.reject. |

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
| 1–49, 92–145; current module/import/attribute shell; declaration DRIVER_JOIN_BUDGET; declaration Outcome; declaration bounded; declaration Field; declaration field; declaration nullable_text; declaration text; declaration nullable_integer; declaration integer; declaration nullable_bytea; declaration bytea; declaration nullable_boolean; declaration boolean | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/pg_query.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 50–91; declaration Row; fields fields; declaration from_pg | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Row: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in pg_query.rs.; from_pg: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in pg_query.rs. — current source cmd/soda-identity/src/pg_query.rs; lines 50-55; module/caller wiring inspected; current source cmd/soda-identity/src/pg_query.rs; lines 56-91; module/caller wiring inspected |

<a id="coverage-008763f67f4b"></a>

## [cmd/soda-identity/src/pg_tests.rs](../../../../../cmd/soda-identity/src/pg_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; current module/import/attribute shell; declaration TEST_LOCK; declaration dsn_policy_keeps_uri_no_tls_and_socket_support | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/pg_tests.rs into its current native target.; TEST_LOCK: implement the current identity enrollment and owner consent duty in pg_tests.rs.; dsn_policy_keeps_uri_no_tls_and_socket_support: implement the current identity enrollment and owner consent duty in pg_tests.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 26–199; declaration integration_store; declaration integration_store_for_dsn; declaration store_authenticates_over_unix_and_loopback_tcp; declaration store_cancellation_discards_session_and_next_operation_reconnects; declaration swallowed_transaction_errors_cannot_commit_or_return_success; declaration typed_nullable_rows_and_transaction_guard_match_store_contract | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | integration_store: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in pg_tests.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

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
| 1–10, 91–198; current module/import/attribute shell; declaration UniqueSeed; fields depth, top; declaration Value; declaration deserialize; declaration UniqueVisitor; fields depth, top; declaration expecting; declaration visit_bool; declaration visit_i64; declaration visit_u64; declaration visit_f64; declaration visit_str; declaration visit_string; declaration visit_unit; declaration visit_none; declaration visit_seq; declaration visit_map | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/strict.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 11–90, 199–216; declaration MAX_DOCUMENT; declaration decode; declaration remap_case; declaration check_known_fields; declaration check_unique_keys; declaration strict_tests | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | MAX_DOCUMENT: implement the current wire decoding, encoding, and representation conversion duty in strict.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8d525f22e8b2"></a>

## [cmd/soda-identity/src/strict_tests.rs](../../../../../cmd/soda-identity/src/strict_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 59–94; current module/import/attribute shell; declaration Envelope; fields command_id, wire_type, target; declaration FIELDS; declaration limits_match_go; declaration case_fold_matches_go_fallback | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/strict_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20–58, 95–98; declaration decode_envelope; declaration strict_vectors_match_go; declaration unicode_keys_compare_decoded | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | decode_envelope: implement the current wire decoding, encoding, and representation conversion duty in strict_tests.rs.; strict_vectors_match_go: implement the current wire decoding, encoding, and representation conversion duty in strict_tests.rs.; unicode_keys_compare_decoded: implement the current wire decoding, encoding, and representation conversion duty in strict_tests.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8cb18c62534a"></a>
<a id="rustsoda-identitysrcwirers-1"></a>
<a id="coverage-e65d2c87fcc0"></a>

## [cmd/soda-identity/src/wire.rs](../../../../../cmd/soda-identity/src/wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 34–88; current module/import/attribute shell; declaration Request; fields provider_id, owner_id, id, label, project_id, kind, execution_id, grant, acquire, binding, credential | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire.rs into its current native target.; Request: implement the current identity enrollment and owner consent duty in wire.rs. — current source cmd/soda-identity/src/wire.rs; Cargo target and callers; current source cmd/soda-identity/src/wire.rs; lines 34-88; module/caller wiring inspected |
| 8–33, 89–98; declaration CODEX; declaration MUSE; declaration READY; declaration REAUTH; declaration REVOKED; declaration FACTORY; declaration TERMINAL; declaration EXECUTION_PENDING; declaration EXECUTION_LIVE; declaration EXECUTION_TERMINAL; declaration provider_valid; declaration DeliveryWire; fields lease, credential; declaration tests | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | CODEX: implement the current wire decoding, encoding, and representation conversion duty in wire.rs.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

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
| 1–70, 172–267; current module/import/attribute shell; declaration Binding; fields child_id, uid, gid, scope, credential_root, invocation_id, kind, id, project, login, generation; declaration Event; fields id, time, action, owner_id, actor_id, connection_id, lease_id, project_id, grant_id, execution_id, kind, generation; declaration validate | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire_execution.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 71–171; declaration Lease; fields repository_id, provider_id, id, connection_id, generation, actor_id, project_id, execution_id, kind, role, deadline, grant_id, grant_revision, binding; declaration Execution; fields binding, kind, execution_id, digest, state, lease_id | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Lease: implement the current execution admission and lease fencing duty in wire_execution.rs.; Execution: implement the current execution admission and lease fencing duty in wire_execution.rs. — current source cmd/soda-identity/src/wire_execution.rs; lines 71-139; module/caller wiring inspected; current source cmd/soda-identity/src/wire_execution.rs; lines 140-171; module/caller wiring inspected |

<a id="coverage-5b28468b9b2c"></a>

## [cmd/soda-identity/src/wire_grants.rs](../../../../../cmd/soda-identity/src/wire_grants.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62, 109–139; current module/import/attribute shell; declaration Grant; fields id, connection_id, user_id, project_id, revision, revoked; declaration GrantRequest; fields connection_id, user_id, project_id, confirm_subscription, confirm_credential_exposure; declaration validate | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/src/wire_grants.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 63–108; declaration AcquireRequest; fields repository_id, provider_id, execution_id, actor_id, connection_id, project_id, kind, deadline, role | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | AcquireRequest: implement the current execution admission and lease fencing duty in wire_grants.rs. — current source cmd/soda-identity/src/wire_grants.rs; lines 63-108; module/caller wiring inspected |
| 140–153; declaration acquisition_digest | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | acquisition_digest: implement the current wire decoding, encoding, and representation conversion duty in wire_grants.rs. — current source cmd/soda-identity/src/wire_grants.rs; lines 140-153; module/caller wiring inspected |

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
| 1–13; current module/import/attribute shell; declaration common | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/tests/broker.rs into its current native target.; common: implement the current identity enrollment and owner consent duty in broker.rs. — current source cmd/soda-identity/tests/broker.rs; Cargo target and callers; current source cmd/soda-identity/tests/broker.rs; lines 3-13; module/caller wiring inspected |
| 14–47; declaration store_round_trip | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | store_round_trip: implement the current encrypted credential custody and cryptographic binding duty in broker.rs. — current source cmd/soda-identity/tests/broker.rs; lines 14-47; module/caller wiring inspected |
| 48–291; declaration muse_lease_returns_by_forget; declaration close_execution_fences_late_registration; declaration end_lease_fences_same_id_reacquire; declaration reconcile_unbound_preserves_recovery; declaration close_after_reconcile_is_idempotent; declaration revoke_retires_live_leases | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | muse_lease_returns_by_forget: implement the current completion, revocation, and reconciliation duty in broker.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-56faffb6cb45"></a>

## [cmd/soda-identity/tests/common/mod.rs](../../../../../cmd/soda-identity/tests/common/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113, 120–139, 153–162, 171–200, 207–260; current module/import/attribute shell; declaration ADMIN_OPERATION_BUDGET; declaration ADMIN_CLEANUP_BUDGET; declaration super_dsn; declaration Ephemeral; fields dsn, super_dsn, name; declaration admin_command; declaration create; declaration drop; declaration hex; declaration fixture_key; declaration subscription; declaration StubSession; fields snapshot, connection, credential; declaration snapshot; declaration StubProvider; fields enrollment, connection, credential; declaration start; declaration StubRuntime; fields credential, calls; declaration validate; declaration stop; declaration stub_provider; declaration controller; declaration binding | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/tests/common/mod.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 114–119, 140–152; declaration store; declaration connection | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | store: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in mod.rs.; connection: implement the current native PostgreSQL query, transaction, schema, and parameter handling duty in mod.rs. — current source cmd/soda-identity/tests/common/mod.rs; lines 114-119; module/caller wiring inspected; current source cmd/soda-identity/tests/common/mod.rs; lines 140-152; module/caller wiring inspected |
| 163–170, 201–206; declaration finish; declaration close | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | finish: implement the current completion, revocation, and reconciliation duty in mod.rs.; close: implement the current completion, revocation, and reconciliation duty in mod.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

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
| 1–8, 42–185, 198–207, 216–238, 278–379; current module/import/attribute shell; declaration common; declaration Guard; fields shutdown; declaration drop; declaration dead_listener_fails_fast; declaration BlockingProvider; fields entered, release, finished; declaration Session; declaration snapshot; declaration start; declaration EmptyRuntime; declaration validate; declaration stop; declaration ShutdownOnDrop | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Imports and module declarations wire cmd/soda-identity/tests/http.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 9–41, 186–197; declaration http_admission_matches_go; declaration shutdown_joins_admitted_provider_work_without_blocking_http_runtime | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | http_admission_matches_go: implement the current wire decoding, encoding, and representation conversion duty in http.rs.; shutdown_joins_admitted_provider_work_without_blocking_http_runtime: implement the current wire decoding, encoding, and representation conversion duty in http.rs. — current source cmd/soda-identity/tests/http.rs; lines 9-41; module/caller wiring inspected; current source cmd/soda-identity/tests/http.rs; lines 186-197; module/caller wiring inspected |
| 208–215, 239–277; declaration finish; declaration close | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | finish: implement the current completion, revocation, and reconciliation duty in http.rs.; close: implement the current completion, revocation, and reconciliation duty in http.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

# Host service admission

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-eae04619505c"></a>
<a id="coverage-13e8650a494c"></a>

## [lib/host/GMUX_PATCHES.md](../../../../../lib/host/GMUX_PATCHES.md)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–188; whole file lines 1-188 | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Host daemon integration patch/contract notes documenting the current backend-to-route seam. — current file lib/host/GMUX_PATCHES.md; tracked in assigned native-runtime input inventory |

<a id="coverage-de1ed5cd138c"></a>

## [lib/host/src/daemon/admission.rs](../../../../../lib/host/src/daemon/admission.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–273; current module/import/attribute shell; declaration BODY_LIMIT_DEFAULT; declaration BODY_LIMIT_LARGE; declaration BODY_LIMIT_IDENTITY; declaration NATIVE_CLEAN_PATHS; declaration ADMITTED_MUTATION_PATHS; declaration IDENTITY_ACTIONS; declaration TAILNET_ACTIONS; declaration TERMINAL_STREAM_CAP; declaration TERMINAL_REQUEST_LIMIT; declaration TERMINAL_FRAME_LIMIT; declaration RequestHead; fields method, path, has_query, escaped, origin_present, upgrade_websocket, ws_key, websocket_request; declaration post; declaration NativeRejection; fields status, message; declaration validate_native_request; declaration is_admitted_mutation_path; declaration body_limit_for; declaration valid_identity_request; declaration validate_tailnet_request; declaration valid_terminal_request; declaration AdmissionGate; fields held; declaration new; declaration acquire; declaration try_acquire; declaration AdmissionGuard; fields _guard; declaration TerminalGate; fields live; declaration try_register; declaration live; declaration TerminalSlot; fields live; declaration drop | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/daemon/admission.rs into its current native target.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-07743b7c23e0"></a>

## [lib/host/src/daemon/http.rs](../../../../../lib/host/src/daemon/http.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; current module/import/attribute shell; declaration MAX_HEADER; declaration HEADER_TIMEOUT; declaration BODY_TIMEOUT; declaration request_head; declaration read_body; declaration percent_decode; declaration valid_percent_escapes | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/daemon/http.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1285f0c9cc29"></a>

## [lib/host/src/daemon/response.rs](../../../../../lib/host/src/daemon/response.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50; current module/import/attribute shell; declaration HttpResponse; declaration response; declaration error_response; declaration not_found_response; declaration json_response; declaration tailnet_json_response; declaration tailnet_response | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/daemon/response.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f38bae4e0df3"></a>
<a id="coverage-af898077b2d3"></a>

## [lib/host/src/daemon/routes.rs](../../../../../lib/host/src/daemon/routes.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–173; current module/import/attribute shell; declaration DaemonConfig; fields image, tailnet_management, terminal_available, identity_available; declaration all_enabled; declaration RouteOutcome; fields response, session, slot; declaration into_response; declaration dispatch; declaration dispatch_native | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/daemon/routes.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 174–206; declaration dispatch_identity | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | dispatch_identity: implement the current provider execution integration duty in routes.rs. — current source lib/host/src/daemon/routes.rs; lines 174-206; module/caller wiring inspected |
| 207–254; declaration dispatch_tailnet; declaration tailnet_unavailable | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | dispatch_tailnet: implement the current host Tailnet control duty in routes.rs.; tailnet_unavailable: implement the current host Tailnet control duty in routes.rs. — current source lib/host/src/daemon/routes.rs; lines 207-248; module/caller wiring inspected; current source lib/host/src/daemon/routes.rs; lines 249-254; module/caller wiring inspected |
| 255–297; declaration dispatch_terminal | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | dispatch_terminal: implement the current interactive terminal attachment duty in routes.rs. — current source lib/host/src/daemon/routes.rs; lines 255-297; module/caller wiring inspected |

<a id="coverage-0445c185179e"></a>

## [lib/host/src/daemon/websocket.rs](../../../../../lib/host/src/daemon/websocket.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; current module/import/attribute shell; declaration websocket_upgrade_response | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/daemon/websocket.rs into its current native target.; websocket_upgrade_response: implement the current Unix-socket service and process lifetime duty in websocket.rs. — current source lib/host/src/daemon/websocket.rs; Cargo target and callers; current source lib/host/src/daemon/websocket.rs; lines 9-14; module/caller wiring inspected |

<a id="coverage-bb1fa657a288"></a>

## [lib/host/src/domain/mod.rs](../../../../../lib/host/src/domain/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; current module/import/attribute shell; declaration account; declaration os; declaration profile; declaration tests; declaration ROCKY_HEADLESS; declaration GO_CC; declaration GO_CF; declaration in_ranges; declaration is_hex_lower; declaration valid_version; declaration valid_id; declaration valid_login; declaration valid_image_ref; declaration valid_container_id; declaration Environment; fields image, profile, id, ip, running; declaration encode_into; declaration encode; declaration Connection; fields environment, host_key, fingerprint | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/domain/mod.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d0e261f49e80"></a>

## [lib/host/src/domain/os.rs](../../../../../lib/host/src/domain/os.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–83; current module/import/attribute shell; declaration OsRelease; fields id, version, name; declaration encode_into; declaration OsObservation; fields environment, release, unavailable; declaration encode; declaration valid_os_id; declaration valid_os_version; declaration valid_os_release | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/domain/os.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f9b619b2cfa1"></a>

## [lib/host/src/domain/profile.rs](../../../../../lib/host/src/domain/profile.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–203; current module/import/attribute shell; declaration Profile; fields id, distribution, version, interface, architecture, image, revision; declaration deserialize; declaration ProfileVisitor; declaration Value; declaration expecting; declaration visit_map; declaration validate; declaration decode; declaration encode_into; declaration encode; declaration decode_profile; declaration Create; fields profile, id, owner; declaration CreateVisitor | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/domain/profile.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4a2153fc5316"></a>

## [lib/host/src/domain/tests.rs](../../../../../lib/host/src/domain/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–181; current module/import/attribute shell; declaration sample_profile; declaration validators_match_go_regexps; declaration profile_validation_matches; declaration profile_wire_round_trip; declaration create_decode_and_validate; declaration environment_omits_empty_like_go; declaration os_release_validation_matches_go; declaration account_dto_strict_shape; declaration signed_json_integer_fields_keep_negative_zero_and_reject_other_number_tokens; declaration access_keys_dto_strict_shape; declaration access_key_state_encodes_struct_order | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/domain/tests.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7768dd1418a0"></a>
<a id="coverage-b14091b1a833"></a>

## [lib/host/src/gmux_admission.rs](../../../../../lib/host/src/gmux_admission.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; current module/import/attribute shell; declaration admission | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/gmux_admission.rs into its current native target.; admission: implement the current Unix-socket service and process lifetime duty in gmux_admission.rs. — current source lib/host/src/gmux_admission.rs; Cargo target and callers; current source lib/host/src/gmux_admission.rs; lines 6-16; module/caller wiring inspected |

<a id="coverage-1c0dfc31b87c"></a>
<a id="coverage-53e1884ced2a"></a>

## [lib/host/src/gmux_backend.rs](../../../../../lib/host/src/gmux_backend.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–67, 138–141; current module/import/attribute shell; declaration BackendError; declaration TerminalSession; fields id; declaration ExecBackend; declaration StubBackend | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/gmux_backend.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 68, 70–71, 142–144, 148–153; declaration profile; declaration inspect; declaration observe_os | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | profile: implement the current Project profile and readiness duty in gmux_backend.rs.; inspect: implement the current Project profile and readiness duty in gmux_backend.rs.; observe_os: implement the current Project profile and readiness duty in gmux_backend.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 69, 145–147; declaration create | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | create: implement the current Project repository association and creation duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 69-69; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 145-147; module/caller wiring inspected |
| 72, 74, 154–156, 160–162; declaration connection; declaration access_keys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | connection: implement the current Project SSH key access duty in gmux_backend.rs.; access_keys: implement the current Project SSH key access duty in gmux_backend.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 73, 157–159; declaration lifecycle | [P05](../../slices/projects.md#p05-project-startstop) | retained | lifecycle: implement the current Project lifecycle duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 73-73; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 157-159; module/caller wiring inspected |
| 75–77, 163–165; declaration account | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | account: implement the current human Project membership and accounts duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 75-77; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 163-165; module/caller wiring inspected |
| 78–81, 166–177; declaration prepare; declaration prepare_candidate; declaration inspect_preparation; declaration stop_preparation | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | prepare: implement the current Project preparation and checkout allocation duty in gmux_backend.rs.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 82–84, 178–180; declaration hold_preparation | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | hold_preparation: implement the current Project maintenance holds duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 82-84; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 178-180; module/caller wiring inspected |
| 85, 181–183; declaration factory_launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | factory_launch: implement the current factory assignment and dispatch duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 85-85; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 181-183; module/caller wiring inspected |
| 86–88, 184–192; declaration factory_inspect; declaration factory_stop; declaration factory_takeover | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | factory_inspect: implement the current factory run lifecycle and intervention duty in gmux_backend.rs.; factory_stop: implement the current factory run lifecycle and intervention duty in gmux_backend.rs.; factory_takeover: implement the current factory run lifecycle and intervention duty in gmux_backend.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 89–92, 193–195; declaration factory_output | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | factory_output: implement the current factory activity presentation duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 89-92; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 193-195; module/caller wiring inspected |
| 93, 98–111, 196–198, 205–210; declaration factory_harness; declaration identity_launch; declaration identity_action | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | factory_harness: implement the current provider execution integration duty in gmux_backend.rs.; identity_launch: implement the current provider execution integration duty in gmux_backend.rs.; identity_action: implement the current provider execution integration duty in gmux_backend.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 94, 199–201; declaration factory_export | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | factory_export: implement the current publication progression duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 94-94; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 199-201; module/caller wiring inspected |
| 95–97, 202–204; declaration factory_candidate_inspect | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | factory_candidate_inspect: implement the current candidate verification assessment duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 95-97; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 202-204; module/caller wiring inspected |
| 112–119, 211–213; declaration tailnet | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | tailnet: implement the current host Tailnet control duty in gmux_backend.rs. — current source lib/host/src/gmux_backend.rs; lines 112-119; module/caller wiring inspected; current source lib/host/src/gmux_backend.rs; lines 211-213; module/caller wiring inspected |
| 120–137, 214–225; declaration terminal_accept; declaration pump_terminal | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | terminal_accept: implement the current interactive terminal attachment duty in gmux_backend.rs.; pump_terminal: implement the current interactive terminal attachment duty in gmux_backend.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5a3b4fac473f"></a>

## [lib/host/src/gmux_routes.rs](../../../../../lib/host/src/gmux_routes.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–66; current module/import/attribute shell; declaration response; declaration routes; declaration websocket; declaration ROUTE_TABLE | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/gmux_routes.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f015ec6c6d39"></a>

## [lib/host/src/gmux_server.rs](../../../../../lib/host/src/gmux_server.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–348; current module/import/attribute shell; declaration http; declaration Server; fields backend, config, gate, terminal_gate, shutdown, inflight; declaration new; declaration shutdown; declaration inflight; declaration serve; declaration handle_connection; declaration Inflight; declaration drop; declaration PumpInflight; declaration wait_shutdown; declaration systemd_listener; declaration bind_listener | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/gmux_server.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6a5d4983a71e"></a>

## [lib/host/src/iconfig/tests.rs](../../../../../lib/host/src/iconfig/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–231; current module/import/attribute shell; declaration GOLDENS; declaration PROJECT_IMAGE; declaration TAILNET_IMAGE; declaration golden; declaration golden_valid_configs; declaration golden_decode_errors; declaration golden_go_framing_parity; declaration golden_release_overlay; declaration golden_validators; declaration MUSE; declaration IDENTITY; declaration STAGED; declaration TAILNET; declaration NATIVE; declaration golden_validation_order; declaration crate_helpers_match_go_validators; declaration configured_muse_socket_requires_clean_absolute_spelling | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/iconfig/tests.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f3d098ee8060"></a>

## [lib/host/src/json/mod.rs](../../../../../lib/host/src/json/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–318; current module/import/attribute shell; declaration number; declaration strict_tests; declaration MAXIMUM_REQUEST_BYTES; declaration Error; declaration fmt; declaration err; declaration SignedInteger; declaration from; declaration deserialize; declaration BytesField; declaration V; declaration Value; declaration expecting; declaration visit_str; declaration visit_string; declaration visit_unit; declaration visit_seq; declaration decode_strict_as; declaration Root; declaration visit_map; declaration UniqueSeed; declaration UniqueVisitor; declaration visit_bool; declaration visit_i64; declaration visit_u64; declaration visit_f64; declaration visit_borrowed_str; declaration visit_none; declaration decode_tolerant_as; declaration quote; declaration GoFormatter; declaration write_string_fragment | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/json/mod.rs into its current native target.; 47 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-de2b0af6125b"></a>

## [lib/host/src/json/number.rs](../../../../../lib/host/src/json/number.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; current module/import/attribute shell; declaration parse_go_int64; declaration parse_go_uint32 | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/json/number.rs into its current native target.; parse_go_int64: implement the current Unix-socket service and process lifetime duty in number.rs.; parse_go_uint32: implement the current Unix-socket service and process lifetime duty in number.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-63ebbc17e0da"></a>

## [lib/host/src/json/strict_tests.rs](../../../../../lib/host/src/json/strict_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; current module/import/attribute shell; declaration AnyObject; declaration strict_admission_rejects_decoded_duplicates_even_in_ignored_values; declaration strict_depth_matches_accepted_empty_container_boundary_and_scalar_leaf; declaration strict_typed_path_preserves_sorted_root_and_nested_source_alias_order; declaration strict_caps_utf8_and_go_string_emission; declaration signed_and_unsigned_number_adapters_keep_distinct_minus_zero_rules | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/json/strict_tests.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0efc6465765f"></a>

## [lib/host/src/net.rs](../../../../../lib/host/src/net.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; current module/import/attribute shell; declaration parse_addr; declaration Prefix; fields addr, bits; declaration parse_prefix; declaration contains; declaration admit_ip; declaration tests; declaration admission_matches_go_cases | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/net.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-70bdbbe6110d"></a>

## [lib/host/src/nist.rs](../../../../../lib/host/src/nist.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–162; current module/import/attribute shell; declaration Curve; fields name, coord_len; declaration P256; declaration P384; declaration P521; declaration decode_point; declaration tests; declaration unhex; declaration generator_points_parse_and_remain_uncompressed; declaration invalid_and_infinity_points_are_rejected | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/nist.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7607fa05d7e6"></a>

## [lib/host/src/sha256.rs](../../../../../lib/host/src/sha256.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–45; current module/import/attribute shell; declaration hex_lower; declaration HEX; declaration digest; declaration tests; declaration fips_vectors | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/sha256.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-401a73ec277c"></a>
<a id="coverage-49d209301d73"></a>

## [lib/host/tests/gmux_smoke.rs](../../../../../lib/host/tests/gmux_smoke.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–195, 315–476, 511–573, 599–733, 793–1407; current module/import/attribute shell; declaration head; declaration ResponseView; declaration status; declaration headers_and_body; declaration body_bytes; declaration header_value; declaration status_of; declaration text_of; declaration body_of; declaration dispatch_stub; declaration ScriptBackend; fields result, session, seen_bodies, seen_image, pumped, pumped_message, blocked_calls, hold_pump; declaration ReleaseBlockedCalls; declaration drop; declaration StopSmokeServer; declaration err; declaration ok; declaration replay; declaration native_clean_paths_reject_query_and_escapes; declaration native_method_check_runs_after_clean_path_check; declaration mutation_gate_covers_exactly_the_go_paths; declaration body_limits_match_go; declaration terminal_constants_match_go; declaration identity_validator_matches_go; declaration tailnet_validator_matches_go; declaration terminal_validator_matches_go; declaration mutation_gate_serializes; declaration route_table_has_every_go_route; declaration every_post_route_dispatches_through_the_stub; declaration native_error_mapping_matches_go; declaration native_unknown_path_is_404_and_terminal_prefix_is_405; declaration native_success_envelope_matches_go_encoder; declaration over_limit_bodies_fail_like_go_maxbytesreader; declaration identity_errors_collapse_to_409_like_go; declaration identity_rejects_bad_shape_unknown_actions_and_missing_runtime; declaration tailnet_error_mapping_matches_go; declaration tailnet_success_has_no_trailing_newline_and_disabled_is_503; declaration terminal_head; declaration terminal_upgrade_holds_slot_and_renders_101; declaration terminal_rejections_match_go; declaration stub_pump_reports_unimplemented; declaration read_all; declaration read_http_head; declaration socket_path; declaration server_serves_stub_routes_and_parser_rejections; declaration server_runs_terminal_upgrade_and_pump; declaration server_keeps_upgrade_inflight_until_pump_shutdown_join; declaration server_bounds_backend_work_and_joins_it_during_shutdown; declaration systemd_listener_refuses_without_activation | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/gmux_smoke.rs into its current native target.; 58 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 196–198, 202–207; declaration profile; declaration inspect; declaration observe_os | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | profile: implement the current Project profile and readiness duty in gmux_smoke.rs.; inspect: implement the current Project profile and readiness duty in gmux_smoke.rs.; observe_os: implement the current Project profile and readiness duty in gmux_smoke.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 199–201, 734–792; declaration create; declaration create_and_mutations_take_the_gate_reads_do_not | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | create: implement the current Project repository association and creation duty in gmux_smoke.rs.; create_and_mutations_take_the_gate_reads_do_not: implement the current Project repository association and creation duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 199-201; module/caller wiring inspected; current source lib/host/tests/gmux_smoke.rs; lines 734-792; module/caller wiring inspected |
| 208–210, 214–216; declaration connection; declaration access_keys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | connection: implement the current Project SSH key access duty in gmux_smoke.rs.; access_keys: implement the current Project SSH key access duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 208-210; module/caller wiring inspected; current source lib/host/tests/gmux_smoke.rs; lines 214-216; module/caller wiring inspected |
| 211–213; declaration lifecycle | [P05](../../slices/projects.md#p05-project-startstop) | retained | lifecycle: implement the current Project lifecycle duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 211-213; module/caller wiring inspected |
| 217–219; declaration account | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | account: implement the current human Project membership and accounts duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 217-219; module/caller wiring inspected |
| 220–231; declaration prepare; declaration prepare_candidate; declaration inspect_preparation; declaration stop_preparation | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | prepare: implement the current Project preparation and checkout allocation duty in gmux_smoke.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 232–234; declaration hold_preparation | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | hold_preparation: implement the current Project maintenance holds duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 232-234; module/caller wiring inspected |
| 235–237; declaration factory_launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | factory_launch: implement the current factory assignment and dispatch duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 235-237; module/caller wiring inspected |
| 238–246; declaration factory_inspect; declaration factory_stop; declaration factory_takeover | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | factory_inspect: implement the current factory run lifecycle and intervention duty in gmux_smoke.rs.; factory_stop: implement the current factory run lifecycle and intervention duty in gmux_smoke.rs.; factory_takeover: implement the current factory run lifecycle and intervention duty in gmux_smoke.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 247–249; declaration factory_output | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | factory_output: implement the current factory activity presentation duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 247-249; module/caller wiring inspected |
| 250–253, 260–269, 574–598; declaration factory_harness; declaration identity_launch; declaration identity_action; declaration bodies_and_harness_image_pass_through_verbatim | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | factory_harness: implement the current provider execution integration duty in gmux_smoke.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 254–256; declaration factory_export | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | factory_export: implement the current publication progression duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 254-256; module/caller wiring inspected |
| 257–259; declaration factory_candidate_inspect | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | factory_candidate_inspect: implement the current candidate verification assessment duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 257-259; module/caller wiring inspected |
| 270–276; declaration tailnet | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | tailnet: implement the current host Tailnet control duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 270-276; module/caller wiring inspected |
| 277–314, 494–510; declaration terminal_accept; declaration pump_terminal; declaration terminal_gate_caps_and_releases | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | terminal_accept: implement the current interactive terminal attachment duty in gmux_smoke.rs.; pump_terminal: implement the current interactive terminal attachment duty in gmux_smoke.rs.; terminal_gate_caps_and_releases: implement the current interactive terminal attachment duty in gmux_smoke.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 477–493; declaration mutation_gate_blocking_acquire_hands_off | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | mutation_gate_blocking_acquire_hands_off: implement the current execution admission and lease fencing duty in gmux_smoke.rs. — current source lib/host/tests/gmux_smoke.rs; lines 477-493; module/caller wiring inspected |

<a id="coverage-e05415c51742"></a>

## [lib/host/tests/identity_transport/common.rs](../../../../../lib/host/tests/identity_transport/common.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–184; current module/import/attribute shell; declaration SOCK_COUNTER; declaration sock_path; declaration read_http_request; declaration find_crlf2; declaration FakeBroker; fields path, stop, handle, captured; declaration start; declaration request; declaration request_count; declaration drop; declaration json_reply; declaration error_reply; declaration deadline; declaration full_request; declaration LEASE_JSON; declaration full_binding; declaration DELIVERY_JSON; declaration sized_lease | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/identity_transport/common.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b836be6ff4ae"></a>

## [lib/host/tests/identity_transport/requests.rs](../../../../../lib/host/tests/identity_transport/requests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; current module/import/attribute shell; declaration acquire_body_matches_go_oracle; declaration acquire_omits_zero_repository_and_role; declaration register_full_binding_matches_go_oracle; declaration register_minimal_binding_matches_go_oracle; declaration reconcile_end_available_bodies_match_go_oracle; declaration return_execution_bodies_match_go_oracle; declaration execution_is_terminal_matches_factory; declaration deadline_format_matches_go_time_json; declaration html_escaping_matches_go_encoder | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/identity_transport/requests.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-01ec285664e6"></a>

## [lib/host/tests/identity_transport/responses.rs](../../../../../lib/host/tests/identity_transport/responses.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; current module/import/attribute shell; declaration error_codes_map_like_go; declaration substring_contract_holds; declaration response_limit_is_512kib; declaration LIMIT; declaration strict_response_decode; declaration deadline_and_transport_failures; declaration framing_edge_cases_match_go_transport | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/identity_transport/responses.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

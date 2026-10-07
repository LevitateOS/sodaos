# Host terminals

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-e60c9c0aa51e"></a>

## [lib/host/src/terminal/agent.rs](../../../../../lib/host/src/terminal/agent.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–138; current module/import/attribute shell; declaration EndIdentityHook; declaration Service; fields exec, codex_harness, codex_harness_sha256, codex_harness_version, muse_harness, muse_harness_sha256, muse_harness_version; declaration inspect_argv; declaration container_exists_argv; declaration agent_argv; declaration podman; declaration podman_owned; declaration project_container; declaration factory_project_container | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/agent.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-438ae1e01b75"></a>

## [lib/host/src/terminal/broker.rs](../../../../../lib/host/src/terminal/broker.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–160; current module/import/attribute shell; declaration valid_identity_request; declaration IdentityBroker; declaration acquire; declaration register; declaration reconcile_lease; declaration rand_id; declaration rand_id_with; declaration entropy_tests; declaration id_generation_rejects_partial_entropy_output; declaration identity_launch; declaration IdentityRoute; declaration identity_route; declaration identity_action | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/broker.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5a5895b7fc9a"></a>

## [lib/host/src/terminal/core.rs](../../../../../lib/host/src/terminal/core.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; current module/import/attribute shell; declaration FRAME_LIMIT; declaration TERMINAL_LIMIT; declaration STREAM_LIMIT; declaration BROKER_RESPONSE_LIMIT; declaration CREDENTIAL_LIMIT; declaration ERR_DENIED; declaration ERR_STALE; declaration ERR_UNCERTAIN; declaration ERR_NOT_FOUND; declaration ERR_INVALID_IDENTITY_OPERATION; declaration ERR_IDENTITY_UNCONFIRMED; declaration IDENTITY_LAUNCH_PATH; declaration PROVIDER_CODEX; declaration PROVIDER_MUSE; declaration KIND_FACTORY; declaration KIND_TERMINAL; declaration SCOPE_MUSE_PROJECT; declaration err_denied; declaration err_stale; declaration err_uncertain; declaration valid_terminal_id; declaration now_unix | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/core.rs into its current native target.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b46ee877136d"></a>

## [lib/host/src/terminal/identity.rs](../../../../../lib/host/src/terminal/identity.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–422; current module/import/attribute shell; declaration IdentityRequest; fields action, delivery, login, scope, cols, rows, source_hash, container, harness_sha256; declaration encode; declaration terminal_reservation; declaration terminal_binding; declaration terminal_lease; declaration terminal_preparation; declaration terminal_prepared; declaration identity_call; declaration prepare_identity; declaration identity_action; declaration identity; declaration managed_end; declaration verify_identity_harness; declaration identity_target; declaration identity_stop_target; declaration identity_stage; declaration identity_result | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/identity.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0c81e922a34d"></a>

## [lib/host/src/terminal/inspect.rs](../../../../../lib/host/src/terminal/inspect.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–241; current module/import/attribute shell; declaration TERMINAL_INSPECT; declaration TerminalInspection; fields id, running, project, owner, privileged, userns, uid_map, gid_map; declaration Mappings; fields uid_map, gid_map; declaration deserialize; declaration MappingsVisitor; declaration Value; declaration expecting; declaration visit_map; declaration InspectionVisitor; declaration decode; declaration terminal_id_map; declaration terminal_isolation; declaration parse_go_int; declaration parse_go_uint; declaration terminal_target_ready; declaration exit_code_of | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/inspect.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-be827ee309d9"></a>

## [lib/host/src/terminal/lease.rs](../../../../../lib/host/src/terminal/lease.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–591; current module/import/attribute shell; declaration Binding; fields child_id, uid, gid, scope, credential_root, invocation_id, kind, id, project, login, generation; declaration BindingWire; fields child_id, uid, gid, scope, credential_root, invocation_id, kind, id, project, login, generation; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration parse_string_i64; declaration Lease; fields repository_id, provider_id, id, connection_id, generation, actor_id, project_id, execution_id, kind, role, deadline_raw, deadline, grant_id, grant_revision, binding; declaration LeaseWire; fields repository_id, provider_id, id, connection_id, generation, actor_id, project_id, execution_id, kind, role, deadline, grant_id, grant_revision, binding, repository_id_seen, actor_id_seen, deadline_seen; declaration binding_from_wire; declaration lease_from_wire; declaration validate; declaration encode_into; declaration encode; declaration decode; declaration Delivery; fields lease, credential; declaration DeliveryWire; fields lease, credential; declaration AcquireRequest; fields repository_id, provider_id, execution_id, actor_id, connection_id, project_id, kind, deadline_secs, deadline_nanos, role | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/lease.rs into its current native target.; 35 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7d03172d524c"></a>
<a id="coverage-d09d964505a5"></a>

## [lib/host/src/terminal/mod.rs](../../../../../lib/host/src/terminal/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; current module/import/attribute shell; declaration factory; declaration core; declaration text; declaration wire; declaration time; declaration lease; declaration inspect; declaration agent; declaration identity; declaration transfer; declaration native; declaration stream; declaration broker; declaration tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/mod.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0f360a8a7b8e"></a>

## [lib/host/src/terminal/native.rs](../../../../../lib/host/src/terminal/native.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–417; current module/import/attribute shell; declaration clean_path; declaration is_clean_absolute_path; declaration stream_identity_harness; declaration agent_program_path; declaration agent_program_hash; declaration native_argv; declaration parse_output_line; declaration NativeAttach; fields child, stdin, reader, closed; declaration from_child_for_test; declaration attach; declaration input_frame; declaration take_reader; declaration output_frame; declaration close; declaration drop | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/native.rs into its current native target.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5ad98e3a5ccf"></a>

## [lib/host/src/terminal/stream.rs](../../../../../lib/host/src/terminal/stream.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106; current module/import/attribute shell; declaration TerminalStart; fields connection_id, project_id, actor_id, login, scope, cols, rows; declaration deserialize; declaration StartVisitor; declaration Value; declaration expecting; declaration visit_map; declaration decode | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/stream.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e2a17f8d7597"></a>

## [lib/host/src/terminal/tests.rs](../../../../../lib/host/src/terminal/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2293; current module/import/attribute shell; declaration PID; declaration TID; declaration CID; declaration TMP_COUNTER; declaration ENV_LOCK; declaration deadline; declaration test_tmp; declaration RecordedCall; declaration FakeExec; fields calls, script; declaration new; declaration calls; declaration argvs; declaration run; declaration ok; declaration inspect_json; declaration terminal_name_matrix; declaration id_predicates; declaration base_request; declaration request_valid_matrix; declaration strict_b64_vectors; declaration frame_input_matrix; declaration frame_output_matrix; declaration meta_state; declaration metadata_output_matrix; declaration frame_encode_goldens; declaration request_encode_golden; declaration golden_binding; declaration golden_lease; declaration identity_encode_goldens; declaration strict_decode_matrix; declaration credential_valid_vectors; declaration string_i64_vectors; declaration rfc3339_vectors; declaration id_map_matrix; declaration inspection; declaration isolation_matrix; declaration exit_code_pins; declaration argv_vectors; declaration native_path_components_and_clean_admission; declaration make_service; declaration project_container_flows; declaration factory_project_container_flows; declaration live_lease; declaration identity_call_flows; declaration terminal_lease_matrix; declaration identity_result_matrix; declaration identity_dispatch_matrix; declaration managed_end_matrix; declaration euid; declaration with_agent_env; declaration agent_program_hash_matrix; declaration write_harness; declaration verify_identity_harness_matrix; declaration stream_identity_harness_flows; declaration preparing_lease; declaration prepare_identity_flows; declaration identity_start_flow; declaration private_request_matrix; declaration terminal_start_decode; declaration zero_deadline_encode_pin; declaration output_line_and_attach_pins; declaration rand_id_shape; declaration FakeBroker; fields acquire_result, register_result, reconciled, acquires; declaration acquire; declaration register; declaration reconcile_lease; declaration launch_input; declaration identity_launch_flows; declaration identity_route_matrix; declaration piped_file; declaration take_reader_detaches_output; declaration quiet_output_never_blocks_input; declaration attach_with_child; declaration proc_stat; declaration assert_pid_reaped; declaration close_reaps_eof_exited_child; declaration close_reaps_killed_child | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/tests.rs into its current native target.; 78 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9e5578752867"></a>

## [lib/host/src/terminal/text.rs](../../../../../lib/host/src/terminal/text.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–70; current module/import/attribute shell; declaration GO_CC; declaration GO_CF; declaration in_ranges; declaration valid_terminal_name; declaration terminal_dimensions; declaration strict_b64_decode; declaration contains_crlf | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/text.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-810ccc2dff79"></a>

## [lib/host/src/terminal/time.rs](../../../../../lib/host/src/terminal/time.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4; current module/import/attribute shell; declaration parse_rfc3339 | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/time.rs into its current native target.; parse_rfc3339: implement the current interactive terminal attachment duty in time.rs. — current source lib/host/src/terminal/time.rs; Cargo target and callers; current source lib/host/src/terminal/time.rs; lines 2-4; module/caller wiring inspected |

<a id="coverage-10c728506987"></a>

## [lib/host/src/terminal/transfer.rs](../../../../../lib/host/src/terminal/transfer.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; current module/import/attribute shell; declaration tar_producer_argv; declaration tar_consumer_argv | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/transfer.rs into its current native target.; tar_producer_argv: implement the current interactive terminal attachment duty in transfer.rs.; tar_consumer_argv: implement the current interactive terminal attachment duty in transfer.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-22a4641d914f"></a>

## [lib/host/src/terminal/wire.rs](../../../../../lib/host/src/terminal/wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–582; current module/import/attribute shell; declaration TerminalRequest; fields action, id, project, login, identity, cols, rows, expires, name, scope; declaration TerminalRequestWire; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration TerminalState; fields id, name, created_at, ready, attached, state; declaration TerminalStateWire; declaration TerminalFrame; fields frame_type, data, cols, rows, reason, terminals; declaration TerminalFrameWire; declaration encode_into; declaration encode; declaration decode; declaration valid_terminal_actor; declaration valid_terminal_window; declaration valid_terminal_scope; declaration valid_list_request; declaration valid_sized_terminal_action; declaration valid_idle_terminal_action; declaration valid_terminal_action; declaration valid; declaration valid_typed_input; declaration valid_resize_input; declaration valid_idle_input; declaration input_valid; declaration output_valid; declaration valid_terminal_state_value; declaration valid_terminal_item_flags; declaration valid_terminal_item; declaration valid_metadata_output; declaration valid_output_data; declaration valid_closed_reason; declaration valid_closed_output; declaration json_valid; declaration credential_valid | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire lib/host/src/terminal/wire.rs into its current native target.; 51 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

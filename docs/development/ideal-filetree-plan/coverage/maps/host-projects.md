# Host projects

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-da2fe5ff23b3"></a>

## [lib/host/src/account/access_keys_tests.rs](../../../../../lib/host/src/account/access_keys_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; current module/import/attribute shell; declaration access_keys_preview_observes_without_applying; declaration access_keys_apply_round_trips_preview_and_confirms_set; declaration access_keys_rejects_drift_and_bad_requests | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire lib/host/src/account/access_keys_tests.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7417f4912a73"></a>
<a id="rustsoda-hostsrcaccountrs-1"></a>
<a id="coverage-4c84103a53c6"></a>

## [lib/host/src/account/mod.rs](../../../../../lib/host/src/account/mod.rs)

Current P03 selectors in this section verified; unaffected prior intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 101–160; GuestPrivilegeRequest; ProjectPrivilegeConfirmation; observe_project_access | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | The read-only observation binds status to the selected running private-userns container and uses the existing deadline/capped executor; typed guest status requires the echoed login/identity and boolean. — Current P03 source selectors. |
| 1–100, 161–412; account and access-key functions/tests | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Existing account and access-key responsibilities; prior selector intervals are historical hints pending R02. |

<a id="coverage-05efeb0fc8c5"></a>

## [lib/host/src/account/tests.rs](../../../../../lib/host/src/account/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–340; current module/import/attribute shell; declaration ED; declaration MockCall; declaration Mock; fields calls, script; declaration new; declaration run; declaration deadline; declaration test_config; declaration pid; declaration inspect_payload; declaration container_payload; declaration key_revision_shape; declaration account_keys_drop_comments_but_keep_newline; declaration development_keys_require_canonical_unique_lines; declaration access_key_state_decode_uses_plain_json_semantics; declaration agent_program_path_matches_go; declaration account_provisions_with_exact_body_and_argv; declaration account_rejects_bad_requests_and_unconfirmed_helpers | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire lib/host/src/account/tests.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-743d93e291d8"></a>

## [lib/host/src/domain/account.rs](../../../../../lib/host/src/domain/account.rs)

Current ProjectAccess DTO declarations verified; prior account/access-key intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 17–40; ProjectAccessRequest; ProjectAccessStatus | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Direct typed, required-field request/status boundary for the read-only native privilege observation. — Current source types. |
| 1–16, 42–204; current module/import/attribute shell; declaration Account; fields project, login, identity, keys; declaration deserialize; declaration AccountVisitor; declaration Value; declaration expecting; declaration visit_map; declaration decode; declaration AccessKeys; fields project, login, identity, revision, keys, apply; declaration AccessKeysVisitor; declaration AccessKeyState; fields revision, keys; declaration encode | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Existing account/access-key IPC shapes; earlier intervals are historical hints pending R02. |

<a id="coverage-4263fb98abf4"></a>
<a id="coverage-f89f906a2051"></a>

## [lib/host/src/pops.rs](../../../../../lib/host/src/pops.rs)

Current P03 selectors in this section verified; unaffected prior intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 228–238; Ops.project_access | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Read-only project access observation calls the guest confirmation path and returns the typed bound status. — Current caller. |
| 1–227; current module/import/attribute shell; declaration AccessKeysReq; declaration decode; declaration AccountReq; declaration PrepareReq; declaration PrepareCandidateReq; declaration InspectPreparationReq; declaration StopPreparationReq; declaration HoldPreparationReq; declaration Ops; fields exec, config; declaration runtime; declaration access_keys; declaration account; declaration prepare; declaration prepare_candidate; declaration inspect_preparation; declaration stop_preparation; declaration hold_preparation | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Existing operation declarations; earlier intervals are historical hints pending R02. |

<a id="coverage-6501cae87408"></a>

## [lib/host/src/project/confirmation.rs](../../../../../lib/host/src/project/confirmation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106; current module/import/attribute shell; declaration valid_address; declaration lifecycle_action_confirmed; declaration confirm_lifecycle; declaration confirm_resolve_profile; declaration valid_os_environment; declaration valid_os_observation; declaration confirm_observe_os; declaration confirm_access_keys | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Imports and module declarations wire lib/host/src/project/confirmation.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5157fa6253b3"></a>

## [lib/host/src/project/connection.rs](../../../../../lib/host/src/project/connection.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; current module/import/attribute shell; declaration connection | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Imports and module declarations wire lib/host/src/project/connection.rs into its current native target.; connection: implement the current identity grants and connection availability duty in connection.rs. — current source lib/host/src/project/connection.rs; Cargo target and callers; current source lib/host/src/project/connection.rs; lines 7-41; module/caller wiring inspected |

<a id="coverage-e7df196519d6"></a>

## [lib/host/src/project/create.rs](../../../../../lib/host/src/project/create.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–155; current module/import/attribute shell; declaration go_dir; declaration create; declaration create_container; declaration start_created; declaration wait_project_ready | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Imports and module declarations wire lib/host/src/project/create.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0550a5e516be"></a>

## [lib/host/src/project/executor.rs](../../../../../lib/host/src/project/executor.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–1003; current module/import/attribute shell; declaration Executor; declaration run; declaration run_with_capture_grace; declaration is_host_native; declaration Native; declaration NativeStatusOnly; declaration execute; declaration execute_with_waiter; declaration refusal_text; declaration completion_text; declaration wait_text; declaration Transfer; declaration Cleanup; declaration CLEANUP_GRACE; declaration retire_group; declaration cleanup_child; declaration wait_exit; declaration set_nonblocking; declaration poll_ready; declaration Poll; declaration poll_once; declaration wait_ready; declaration drain_pipe; declaration pump_stdin; declaration exit_text; declaration tests; declaration HITS_USR1; declaration HITS_USR2; declaration sigusr1_hit; declaration sigusr2_hit; declaration arm_sig; declaration hits; declaration MaskGuard; declaration unblock; declaration drop; declaration PipeGuard; declaration new; declaration read; declaration SenderGuard; declaration spawn; declaration join_assert; declaration poll_once_reports_interruption; declaration wait_ready_recomputes_budget_on_interruption; declaration Fixture; fields dir; declaration FIXTURE_SEQ; declaration fresh; declaration script; declaration path; declaration reap_bounded; declaration ReapGuard; fields seen; declaration echild_never_signals_unpinned_group; declaration intact_custody_wait_error_cleans_up | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Imports and module declarations wire lib/host/src/project/executor.rs into its current native target.; 67 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-98d18c97f88c"></a>

## [lib/host/src/project/inspect.rs](../../../../../lib/host/src/project/inspect.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–414; current module/import/attribute shell; declaration GoStringList; declaration deserialize; declaration ListVisitor; declaration Value; declaration expecting; declaration visit_unit; declaration visit_seq; declaration ProjectIdMappings; fields uid_map, gid_map; declaration MappingsVisitor; declaration visit_map; declaration ProjectContainerInspection; fields id, running, project, owner, privileged, userns, mappings; declaration InspectionVisitor; declaration decode_project_container_inspection; declaration ImageConfig; fields labels; declaration ContainerState; fields running; declaration Network; fields ip_address; declaration NetworkSettings; fields networks; declaration ContainerInspect; fields image, config, state, network_settings; declaration ObjectVisitor; declaration ProjectInspectArray; declaration ArrayVisitor; declaration PROJECT_INSPECT_FORMAT; declaration inspect; declaration project_container; declaration project_id_map | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Imports and module declarations wire lib/host/src/project/inspect.rs into its current native target.; 41 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6695c792cf9e"></a>
<a id="rustsoda-hostsrcprojectrs-1"></a>
<a id="coverage-6cb2f9090807"></a>

## [lib/host/src/project/mod.rs](../../../../../lib/host/src/project/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31, 33, 35, 57–1018; current module/import/attribute shell; declaration confirmation; declaration create; declaration inspect; declaration tests; declaration Config; fields muse_socket, image, network, subnet, bridge; declaration Lifecycle; fields project, action; declaration deserialize; declaration LifecycleVisitor; declaration Value; declaration expecting; declaration visit_map; declaration decode; declaration LifecycleState; fields environment, boot_enabled; declaration encode; declaration PROJECT_UNIT_PATH; declaration Runtime; fields exec, config; declaration podman; declaration lifecycle; declaration read_project_unit; declaration apply_lifecycle_action; declaration go_arch; declaration valid_lifecycle_action; declaration parse_unit_show_properties; declaration validate_unit_properties; declaration ERR; declaration verify_lifecycle_outcome; declaration retained_tests; declaration inspect_observes_identity_and_profile; declaration inspect_rejects_mismatch_and_bad_network; declaration project_container_binds_isolated_identity; declaration format_inspect; declaration unit_show; declaration unit_show_parsing_matrix; declaration lifecycle_start_stop_inspect_flows; declaration lifecycle_rejects_identity_change_and_bad_outcome; declaration lifecycle_facade_confirmations; declaration address_profile_os_key_confirmations; declaration native_run_drains_large_dual_streams_intact; declaration native_run_failure_shape_carries_full_stderr; declaration native_run_deadline_kills_promptly | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Imports and module declarations wire lib/host/src/project/mod.rs into its current native target.; 41 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 32; declaration connection | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | connection: implement the current identity grants and connection availability duty in mod.rs. — current source lib/host/src/project/mod.rs; lines 32-32; module/caller wiring inspected |
| 34; declaration executor | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | executor: implement the current execution admission and lease fencing duty in mod.rs. — current source lib/host/src/project/mod.rs; lines 34-34; module/caller wiring inspected |
| 36–56; declaration os; declaration profile | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | os: implement the current Project profile and readiness duty in mod.rs.; profile: implement the current Project profile and readiness duty in mod.rs. — current source lib/host/src/project/mod.rs; lines 36-36; module/caller wiring inspected; current source lib/host/src/project/mod.rs; lines 37-56; module/caller wiring inspected |

<a id="coverage-04b4a714c784"></a>

## [lib/host/src/project/os.rs](../../../../../lib/host/src/project/os.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–257; current module/import/attribute shell; declaration observe_os; declaration is_python_space; declaration split_lines; declaration shlex_posix; declaration valid_release_id; declaration valid_release_version; declaration parse_os_release | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Imports and module declarations wire lib/host/src/project/os.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-18e8714a2c32"></a>

## [lib/host/src/project/profile.rs](../../../../../lib/host/src/project/profile.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; current module/import/attribute shell; declaration ImageInspection; fields id, architecture, os, labels; declaration deserialize; declaration ImageVisitor; declaration Value; declaration expecting; declaration visit_map; declaration PROFILE_INSPECT_FORMAT; declaration resolve_profile; declaration apply_creation_profile | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Imports and module declarations wire lib/host/src/project/profile.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b99259e40113"></a>

## [lib/host/src/project/tests.rs](../../../../../lib/host/src/project/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–262; current module/import/attribute shell; declaration Mock; fields calls, script; declaration new; declaration run; declaration deadline; declaration test_config; declaration sample_profile; declaration container_inspect; declaration image_inspect; declaration resolve_profile_accepts_native_image; declaration resolve_profile_refuses_foreign_or_invalid; declaration create_validates_before_exec; declaration os_release_parsing_mirrors_python_cases; declaration observe_os_reports_unavailable_without_starting; declaration connection_returns_ed25519_material | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Imports and module declarations wire lib/host/src/project/tests.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-54c9445c6cd9"></a>

## [lib/host/src/ssh/base64.rs](../../../../../lib/host/src/ssh/base64.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123; current module/import/attribute shell; declaration go_std; declaration source_offset; declaration go_error_offset; declaration source_len_for_incomplete_quantum; declaration b64_decode; declaration b64_decode_go; declaration b64_corrupt; declaration b64_encode; declaration b64_encode_raw | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Imports and module declarations wire lib/host/src/ssh/base64.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3548ba8a28b9"></a>

## [lib/host/src/ssh/mod.rs](../../../../../lib/host/src/ssh/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–183; current module/import/attribute shell; declaration base64; declaration tests; declaration ParsedKey; fields key_type, blob; declaration ALGO_RSA; declaration ALGO_DSS; declaration ALGO_ECDSA256; declaration ALGO_ECDSA384; declaration ALGO_ECDSA521; declaration ALGO_SKECDSA; declaration ALGO_ED25519; declaration ALGO_SKED25519; declaration check_key_data; declaration bit_len; declaration less_than; declaration parse_wire; declaration parse_public_key; declaration parse_authorized_key; declaration marshal_authorized_key; declaration fingerprint_sha256 | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Imports and module declarations wire lib/host/src/ssh/mod.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-504f86fdb522"></a>

## [lib/host/src/ssh/tests.rs](../../../../../lib/host/src/ssh/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–401; current module/import/attribute shell; declaration put_string; declaration read_string; declaration push_u32; declaration push_u64; declaration certificate_blob; declaration string_field; declaration mpint_field; declaration rsa_blob; declaration dsa_blob; declaration ecdsa_fields; declaration ordinary_ecdsa_point; declaration certificate_families; declaration RSA; declaration ECDSA256; declaration ECDSA384; declaration ECDSA521; declaration ED; declaration DSA; declaration CERT; declaration canonical; declaration ordinary_p256_point; declaration all_types_round_trip_canonically; declaration real_forever_certificate_preserves_raw_validity_and_signed_wire; declaration redundant_leading_zero_mpint_is_rejected; declaration all_certificate_families_parse_without_trust_or_validity_checks; declaration rsa_sha2_certificate_alias_is_refused_when_library_normalizes_it; declaration sk_types_parse; declaration sk_ecdsa_p256_validates_point_and_curve_id; declaration ecdsa_algorithm_rejects_wrong_curve_id; declaration comments_and_whitespace_tolerated_but_not_canonical; declaration options_are_refused_and_multiline_key_selection_stays_bounded; declaration invalid_keys_rejected; declaration rsa_exponent_and_modulus_bounds_remain_host_policy; declaration dsa_width_and_positive_in_range_parameters_remain_host_policy; declaration base64_vectors; declaration fingerprint_shape | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Imports and module declarations wire lib/host/src/ssh/tests.rs into its current native target.; 37 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8dde55d47076"></a>

## [lib/host/tests/project_operations/access_keys.rs](../../../../../lib/host/tests/project_operations/access_keys.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–114; current module/import/attribute shell; declaration access_keys_observe_matches_golden; declaration access_keys_apply_rechecks_revision; declaration access_keys_rejects_revision_drift; declaration access_keys_validates_before_exec; declaration oracle_access_key_state_empty_encoding | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/project_operations/access_keys.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-596349a89488"></a>

## [lib/host/tests/project_operations/accounts.rs](../../../../../lib/host/tests/project_operations/accounts.rs)

Current P03 selectors in this section verified; unaffected prior intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 53–114; project_access_uses_bound_container_and_confirms_false; project_access_refuses_unconfirmed_or_invalid_observations | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Read-only route tests cover bound running-container selection, confirmed false, and refusal on unconfirmed/invalid observations. — Current source selectors. |
| 1–52; account_provisions_login; account_refuses_stopped_and_root | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Existing account-operation tests. — Current source selectors. |

<a id="coverage-aea250dcca40"></a>

## [lib/host/tests/project_operations/candidate.rs](../../../../../lib/host/tests/project_operations/candidate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; current module/import/attribute shell; declaration candidate_body; declaration prepare_candidate_reuses_protected_snapshot; declaration SRC; declaration NEW; declaration prepare_candidate_rejects_unready_source | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/project_operations/candidate.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a97b7645f11c"></a>

## [lib/host/tests/project_operations/common.rs](../../../../../lib/host/tests/project_operations/common.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–198; current module/import/attribute shell; declaration PID; declaration FID; declaration REV; declaration COMMIT; declaration ED; declaration GO_ACCESS_KEY_STATE; declaration GO_ACCESS_KEY_STATE_EMPTY; declaration GO_HOLD_TRUE; declaration GO_HOLD_FALSE; declaration GO_PREPARE_FULL; declaration GO_PREPARE_READY; declaration GO_PREPARE_STOPPED; declaration GO_PREPARE_MISSING; declaration GO_KEYS_BODY_APPLY; declaration GO_ACCOUNT_BODY; declaration GO_HELPER_INSPECT; declaration MockCall; declaration Mock; fields calls, script; declaration new; declaration run; declaration deadline; declaration test_config; declaration ops; declaration format_inspect; declaration container_inspect; declaration helper_state; declaration approve_response; declaration fixture_files; declaration fixture_digest; declaration prepare_body; declaration binding_argv; declaration helper_argv; declaration helper_ops | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/project_operations/common.rs into its current native target.; 34 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5c1dc323176c"></a>

## [lib/host/tests/project_operations/hold.rs](../../../../../lib/host/tests/project_operations/hold.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; current module/import/attribute shell; declaration hold_preparation_confirms_marker | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/project_operations/hold.rs into its current native target.; hold_preparation_confirms_marker: implement the current Unix-socket service and process lifetime duty in hold.rs. — current source lib/host/tests/project_operations/hold.rs; Cargo target and callers; current source lib/host/tests/project_operations/hold.rs; lines 6-55; module/caller wiring inspected |

<a id="coverage-9b81f04f97ec"></a>

## [lib/host/tests/project_operations/main.rs](../../../../../lib/host/tests/project_operations/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–32; current module/import/attribute shell; declaration access_keys; declaration accounts; declaration candidate; declaration common; declaration hold; declaration preparation; declaration requests; declaration ops_constructor_mirrors_runtime_fields | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/project_operations/main.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f2316f6d4c80"></a>

## [lib/host/tests/project_operations/requests.rs](../../../../../lib/host/tests/project_operations/requests.rs)

Current P03 selectors in this section verified; unaffected prior intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 63–83; project_access_request_has_exact_bound_identity_shape | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Canonical required-field request decoding test. — Current source selector. |
| 1–139; current module/import/attribute shell; declaration access_keys_req_strict_shape; declaration account_req_strict_shape; declaration prepare_req_strict_shape; declaration prepare_candidate_req_strict_shape; declaration inspect_stop_hold_req_shapes; declaration oversize_body_rejected_before_shape | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Existing request-shape checks. — Current source selectors. |

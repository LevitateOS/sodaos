# Host tailnet control

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-73caae56d5bb"></a>

## [lib/host/src/tailnet/domain/address_tests.rs](../../../../../lib/host/src/tailnet/domain/address_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–244; current module/import/attribute shell; declaration status_peer_address_rules; declaration status_dns_name_rules; declaration dns_trims_nel_and_fold_covers_simple_fold_orbits; declaration id_matchers; declaration rfc3339_vectors; declaration escape_vectors | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/address_tests.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1e6d716a3d4d"></a>

## [lib/host/src/tailnet/domain/addresses.rs](../../../../../lib/host/src/tailnet/domain/addresses.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–152; current module/import/attribute shell; declaration valid_label; declaration canonical_magic_dns_name; declaration ParsedAddr; fields addr, zone; declaration parse_addr; declaration canonical; declaration is_global_unicast; declaration check_addresses; declaration peer_view; declaration resolve_project_peer | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/addresses.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0882b695e4d5"></a>

## [lib/host/src/tailnet/domain/mod.rs](../../../../../lib/host/src/tailnet/domain/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–107; current module/import/attribute shell; declaration ERR_INVALID; declaration ERR_CONFLICT; declaration ERR_UNSUPPORTED; declaration ERR_UNCONFIRMED; declaration ERR_UNAVAILABLE; declaration RESPONSE_LIMIT; declaration ReadFile; declaration LinkFile; declaration StatFile; declaration RunTarget; fields project, container, run; declaration RunBinding; fields enabled, admission, tailnet, tags; declaration ProjectRequest; fields project, action, revision, binding, confirm_id; declaration ProjectView; fields available_binding, available_network, addresses, dns_name, saved, project, revision, binding, enabled, state, outcome; declaration valid_project_id; declaration valid_container_id; declaration valid_image_id; declaration unavailable; declaration time; declaration native; declaration addresses; declaration status; declaration node_tests; declaration status_tests; declaration address_tests | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/mod.rs into its current native target.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0cc7e0f498fc"></a>

## [lib/host/src/tailnet/domain/native.rs](../../../../../lib/host/src/tailnet/domain/native.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–418; current module/import/attribute shell; declaration lower_char; declaration fold_char; declaration fold_eq; declaration SelfPeer; fields id, dns_name, ips, tags, online, expired; declaration StringList; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_unit; declaration visit_seq; declaration PeerWire; fields id, dns_name, ips, tags, online, expired; declaration visit_map; declaration merge_peer; declaration NativeStatus; fields backend_state, have_node_key, tailnet, peer; declaration StatusWire; fields value, required_state, key_seen, exact_key_seen, exact_key_null, key_value_seen; declaration TailnetWire; fields name; declaration decode_native_status; declaration NativePrefs; fields want_running, corp_dns, route_all, run_ssh, exit_node_id, exit_node_ip, advertise_routes; declaration PrefsWire; fields value, required; declaration decode_native_prefs | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/native.rs into its current native target.; 41 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-82a6761d7157"></a>

## [lib/host/src/tailnet/domain/node_tests.rs](../../../../../lib/host/src/tailnet/domain/node_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–130; current module/import/attribute shell; declaration binding; declaration status_doc; declaration prefs_doc; declaration has_node_vectors; declaration has_node_state_table | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/node_tests.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-493cbbffb5ff"></a>

## [lib/host/src/tailnet/domain/status.rs](../../../../../lib/host/src/tailnet/domain/status.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; current module/import/attribute shell; declaration parse_project_status; declaration validate_project_preferences; declaration match_project_self; declaration match_project_binding; declaration project_status; declaration project_has_node | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/status.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cb6aa6e01196"></a>

## [lib/host/src/tailnet/domain/status_tests.rs](../../../../../lib/host/src/tailnet/domain/status_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–273; current module/import/attribute shell; declaration status_fresh_daemon_omits_node_key; declaration status_connected_and_unsafe_prefs; declaration status_binding_mismatch_is_unconfirmed; declaration status_non_running_outcomes; declaration status_malformed_inputs; declaration status_case_fold_binding; declaration status_response_limit | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/status_tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-142859ccc02f"></a>

## [lib/host/src/tailnet/domain/time.rs](../../../../../lib/host/src/tailnet/domain/time.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11; current module/import/attribute shell; declaration parse_rfc3339_nano; declaration go_escape | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/domain/time.rs into its current native target.; parse_rfc3339_nano: implement the current host Tailnet control duty in time.rs.; go_escape: implement the current host Tailnet control duty in time.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-06a95215dee3"></a>

## [lib/host/src/tailnet/files/keys.rs](../../../../../lib/host/src/tailnet/files/keys.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; current module/import/attribute shell; declaration write_run_key; declaration retire_run_key; declaration pending_run_key_matches; declaration open_pending_run_key; declaration retire_pending_run_key; declaration write_companion_id; declaration read_companion_id_file; declaration read_companion_id | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/files/keys.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-497b2d343a1c"></a>

## [lib/host/src/tailnet/files/mod.rs](../../../../../lib/host/src/tailnet/files/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–239; current module/import/attribute shell; declaration RUNTIME_RECORD_NOT_FOUND; declaration Root; fields path; declaration RunFiles; fields root, lock, uid, gid; declaration join; declaration lstat; declaration stat; declaration root_directory; declaration runtime_file; declaration owned_run_dir; declaration same_file; declaration mkdir_700; declaration chown_path; declaration open_runtime_root; declaration ensure_runtime_project_dir; declaration acquire_exclusive; declaration lock_runtime_project; declaration open_runtime_project_owned; declaration open_runtime_project; declaration run; declaration keys; declaration resolver; declaration tests | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/files/mod.rs into its current native target.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-62b4feb57f3b"></a>

## [lib/host/src/tailnet/files/resolver.rs](../../../../../lib/host/src/tailnet/files/resolver.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–74; current module/import/attribute shell; declaration fresh_tailscale_conflict; declaration validate_resolver_text; declaration resolver_path; declaration validate_resolver_inode; declaration validate_run_resolver | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/files/resolver.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-57e8376f3da1"></a>

## [lib/host/src/tailnet/files/run.rs](../../../../../lib/host/src/tailnet/files/run.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–122; current module/import/attribute shell; declaration prepare_run_subdir; declaration current; declaration save_current; declaration prepare | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/files/run.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8854a087c798"></a>

## [lib/host/src/tailnet/files/tests.rs](../../../../../lib/host/src/tailnet/files/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–304; current module/import/attribute shell; declaration COUNTER; declaration TempDir; fields path; declaration create; declaration path; declaration drop; declaration runtime_test_root; declaration euid; declaration egid; declaration file_run; declaration shifted_run; declaration open; declaration run_files_exclusive_secret_retirement_and_independent_locks; declaration runtime_root_refuses_symlink_without_writing_through_it; declaration run_files_refuse_symlinks_modes_and_changed_key_inode; declaration resolver_requires_original_inode_and_no_conflicting_manager; declaration completed_key_input_can_be_retired_for_explicit_retry | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/files/tests.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8395cbee2fe6"></a>

## [lib/host/src/tailnet/forgejo.rs](../../../../../lib/host/src/tailnet/forgejo.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–1501; current module/import/attribute shell; declaration DEADLINE_SECS; declaration PODMAN; declaration SYSTEMCTL; declaration TAILSCALE_CLI; declaration FORGEJO_CONTAINER; declaration FORGEJO_ENV; declaration SSH_DOMAIN_KEY; declaration run; declaration inspect_forgejo; declaration PodBinding; fields host_ip, host_port; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration HostConfig; fields port_bindings; declaration PodConfig; fields env; declaration PodState; fields running; declaration PodInspect; fields host_config, config, state; declaration published_state; declaration ssh_bound_to_ip; declaration valid_endpoint_name; declaration update_ssh_domain; declaration rewrite_ssh_domain; declaration write_forgejo_env; declaration TEMP_SEQ; declaration stage_forgejo_env; declaration write_full; declaration close_owned; declaration restart_forgejo_if_needed; declaration tests; declaration Mock; fields calls, inspect, restart_err; declaration new; declaration argv; declaration TEST_SEQ; declaration TestEnv; fields dir; declaration fresh; declaration path; declaration drop; declaration PUBLISHED; declaration refresh_requires_actual_tailnet_listener; declaration published_state_rejects_malformed; declaration inspection_validates_every_port_binding; declaration inspection_validates_every_env_item; declaration duplicate_and_case_merge; declaration sequential_alias_null_semantics; declaration bound_to_ip_matrix; declaration ssh_domain_env_cases; declaration tailnet_preserves_browser_origin; declaration rewrite_matrix; declaration endpoint_name_matrix; declaration invalid_endpoint_refused; declaration written_env_is_private; declaration checked_close_reports_errors; declaration zero_write_fails_without_spinning; declaration restart_conditions_and_argv; declaration status_only_reports_real_status; declaration status_only_reports_real_signal; declaration status_only_argv_passthrough; declaration status_only_spawn_failure; declaration expired_deadline_refuses_without_launch; declaration live_timeout_cleans_up_child_and_group; declaration large_transfers_complete_exactly; declaration native_real_error_shapes; declaration descendant_pipes_cannot_wedge_completion; declaration run_command_completion_bounded_by_grace; declaration descendant_stdin_hold_cannot_wedge_completion; declaration stub_cli; declaration endpoint_cli_reports_real_status; declaration endpoint_cli_parses_real_output; declaration endpoint_cli_ignores_stdout_on_failure; declaration inspect_failure_sanitized; declaration root_gate_refused | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/forgejo.rs into its current native target.; 95 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a35ca45673c8"></a>

## [lib/host/src/tailnet/runtime/mod.rs](../../../../../lib/host/src/tailnet/runtime/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; current module/import/attribute shell; declaration TAILNET_RUN_INSPECT; declaration ProjectRun; fields target, pid, started, userns, netns, uid, gid, resolver; declaration ProcessIdentity; fields start, userns, netns, boot, uid, gid; declaration ProjectRunInspect; fields id, running, pid, started, resolver; declaration ProjectInspection; fields id, running, project, owner, privileged, userns, uid_map, gid_map; declaration unavailable; declaration process; declaration wire; declaration real_read; declaration real_link; declaration admit_project_run_snapshot; declaration confirm_project_run_identity; declaration inspect_project_run; declaration project_run; declaration companion_create_args; declaration recheck_project_run; declaration project; declaration tests; declaration project_tests | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/runtime/mod.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-52b1675a59f5"></a>

## [lib/host/src/tailnet/runtime/process.rs](../../../../../lib/host/src/tailnet/runtime/process.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–144; current module/import/attribute shell; declaration split_fields; declaration parse_stat_fields; declaration parse_start_time_inner; declaration parse_process_start_time; declaration read_id_map_inner; declaration read_process_id_map; declaration parse_namespace_link; declaration read_namespace_inner; declaration read_process_namespace; declaration read_boot_id_inner; declaration read_system_boot_id; declaration process_run_identity | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/runtime/process.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fbfbc68d8b0c"></a>

## [lib/host/src/tailnet/runtime/project.rs](../../../../../lib/host/src/tailnet/runtime/project.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–234; current module/import/attribute shell; declaration id_map; declaration project_isolation; declaration MappingWire; fields uid_map, gid_map; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration InspectionWire; fields id, running, project, owner, privileged, userns, mappings; declaration inspect_project; declaration project_container; declaration project_running | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/runtime/project.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-946b44b1f784"></a>

## [lib/host/src/tailnet/runtime/project_tests.rs](../../../../../lib/host/src/tailnet/runtime/project_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; current module/import/attribute shell; declaration FakeExec; fields out, err, seen; declaration run; declaration inspection_json; declaration inspect_project_argv_and_gates | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/runtime/project_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9a6629737f59"></a>

## [lib/host/src/tailnet/runtime/tests.rs](../../../../../lib/host/src/tailnet/runtime/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–327; current module/import/attribute shell; declaration stat_bytes; declaration mode_read; declaration mode_link; declaration process_identity_matrix; declaration process_identity_rejects_non_root_pid; declaration fixture_run; declaration companion_recipe_keeps_fixed_namespaces; declaration stat_fields_reject_bad_shapes; declaration namespace_link_shapes; declaration id_map_edges; declaration marshal_and_decode_round_trip; declaration decode_run_inspect_binds_container; declaration admit_snapshot_rejects_before_touching_proc | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/runtime/tests.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c8bd3f1e425e"></a>

## [lib/host/src/tailnet/runtime/wire.rs](../../../../../lib/host/src/tailnet/runtime/wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–309; current module/import/attribute shell; declaration RunInspectWire; fields id, running, pid, started, resolver; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration decode_project_run_inspect; declaration marshal_project_run; declaration RunTargetWire; fields project, container, run; declaration ProjectRunWire; fields target, pid, started, userns, netns, uid, gid, resolver; declaration decode_project_run; declaration assemble_project_run | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/runtime/wire.rs into its current native target.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-db67f334b66a"></a>
<a id="coverage-e59785a5aad8"></a>

## [lib/host/src/tcontrol.rs](../../../../../lib/host/src/tcontrol.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–144, 163–167, 197–256; current module/import/attribute shell; declaration NativeExec; declaration Options; fields state_dir, uid, runtime, socket, cli, libexec, curl; declaration default; declaration Control; fields exec, policy, socket, cli, libexec, curl, local_stub, provider_stub; declaration new; declaration new_project_control; declaration local_round_trip; declaration transport; declaration provider; declaration observe; declaration host_action | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tcontrol.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 145–162, 168–182, 257–267; declaration check_credential; declaration lock_policy; declaration settings; declaration enrollment | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | check_credential: protected operator credential/policy admission.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 183–196, 268–278, 317–325; declaration options; declaration project | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | options: project selection options.; project: project selection actions. — Current named units/source consumers; retained normalized source evidence records each selector |
| 279–305, 326–346; declaration run_binding; declaration enroll_run | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | run_binding: exact companion run admission and key handoff.; enroll_run: exact companion run admission and key handoff. — Current named units/source consumers; retained normalized source evidence records each selector |
| 306–316, 347–352; declaration cli_status; declaration cli_endpoint; declaration capped | [N07](../../slices/networking.md#n07-git-endpoint-advertisement) | retained | cli_status: native git endpoint status advertisement.; cli_endpoint: native git endpoint status advertisement.; capped: native git endpoint status advertisement. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-53259cb2f4f9"></a>
<a id="coverage-e18bd9cbad60"></a>

## [lib/host/src/tcontrol_enroll.rs](../../../../../lib/host/src/tcontrol_enroll.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; current module/import/attribute shell | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tcontrol_enroll.rs into its current native target. — current source lib/host/src/tcontrol_enroll.rs; Cargo target and callers |
| 17–151; declaration Provider; fields exec, curl; declaration token; declaration key; declaration check_credential; declaration project_key | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Provider: restricted provider credential and key interface.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 152–214; declaration admit_enroll_run; declaration enroll_run | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | admit_enroll_run: companion run authority admission and issued key handoff.; enroll_run: companion run authority admission and issued key handoff. — current source lib/host/src/tcontrol_enroll.rs; lines 152-173; module/caller wiring inspected; exact byte-identical source map rust/soda-host/src/tcontrol_enroll.rs; exact byte-identical prior map docs/development/ideal-filetree-plan/coverage/maps/host-tailnet-control.md at the same current line; current source lib/host/src/tcontrol_enroll.rs; lines 174-214; module/caller wiring inspected; exact byte-identical source map rust/soda-host/src/tcontrol_enroll.rs; exact byte-identical prior map docs/development/ideal-filetree-plan/coverage/maps/host-tailnet-control.md at the same current line |

<a id="coverage-b3f689740403"></a>
<a id="coverage-9fdabeddca28"></a>

## [lib/host/src/tcontrol_native.rs](../../../../../lib/host/src/tcontrol_native.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–878, 907–978, 996–1115; current module/import/attribute shell; declaration HOST_SOCKET; declaration DEFAULT_CLI; declaration DEFAULT_LIBEXEC; declaration RESPONSE_LIMIT; declaration LOCAL_TIMEOUT; declaration UP_TIMEOUT; declaration Transport; declaration RoundTrip; declaration finish_local; declaration local_request; declaration fold_char; declaration fold_eq; declaration NativePeer; fields id, dns_name, ips, online, exit_node_option, expired; declaration NativeStatus; fields backend_state, have_node_key, tailnet_name, tailnet_magic_dns, has_tailnet, self_peer, peers, health_count, auth_url; declaration GoStringList; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_unit; declaration visit_seq; declaration PeerWire; fields id, dns_name, ips, online, exit_node_option, expired; declaration visit_map; declaration from; declaration merge_peer; declaration TailnetWire; fields name, magic; declaration NativeStatusWire; fields value, exact_backend, exact_key_seen, exact_key, folded_key_seen; declaration decode_native_status; declaration NativePrefs; fields want_running, exit_node_id, exit_node_ip, allow_lan, advertise_routes; declaration NativePrefsWire; fields value, required; declaration decode_native_prefs; declaration round_trip; declaration fetch_native_status; declaration fetch_native_prefs; declaration populate_host_preferences; declaration compute_host_revision; declaration observe; declaration COMPLETION_GRACE; declaration run_command; declaration decode_up_notifications; declaration NotificationWire; fields error; declaration split_json_values; declaration socket_arg; declaration execute_signin; declaration execute_logout; declaration find_available_exit_node; declaration execute_exit_node; declaration verify_exit_node; declaration verify_host_action_outcome; declaration readback_host_action; declaration CliTailnetWire; fields magic; declaration CliPeerWire; fields dns, expired, ips; declaration CliStatusWire; fields backend, auth, tailnet, peer | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tcontrol_native.rs into its current native target.; 93 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 879–906, 979–995, 1116–1194; declaration execute_advertise_exit_node; declaration execute_refresh_forgejo; declaration CliStatus; fields backend_state, identity, ipv4, magic_dns_enabled, expired, auth_pending; declaration CliEndpoint; fields identity, ipv4; declaration parse_cli_status; declaration cli_status; declaration cli_endpoint | [N07](../../slices/networking.md#n07-git-endpoint-advertisement) | retained | execute_advertise_exit_node: implement the current Git endpoint advertisement duty in tcontrol_native.rs.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 1195–1223; declaration RunStatus; fields enabled, admission, tailnet, tags, addresses, dns_name; declaration validate | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | RunStatus: implement the current Project companion lifecycle duty in tcontrol_native.rs.; validate: implement the current Project companion lifecycle duty in tcontrol_native.rs. — current source lib/host/src/tcontrol_native.rs; lines 1195-1206; module/caller wiring inspected; current source lib/host/src/tcontrol_native.rs; lines 1207-1223; module/caller wiring inspected |

<a id="coverage-9b9220ebf3f1"></a>
<a id="coverage-7e9229d23367"></a>

## [lib/host/src/tcontrol_policy.rs](../../../../../lib/host/src/tcontrol_policy.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27, 66–93, 108–185, 213–265, 317–576, 964–983; current module/import/attribute shell; declaration deserialize; declaration CredentialVisitor; declaration Value; declaration expecting; declaration visit_map; declaration PolicyVisitor; declaration EntryVisitor; declaration new_revision; declaration new_revision_with; declaration PolicyLock; fields dir; declaration fd; declaration dir_owned; declaration file_owned; declaration last_errno; declaration acquire_exclusive; declaration cstr; declaration EnsureErr; declaration current_uid; declaration PolicyStore; fields state_dir, uid, runtime, sync_hook; declaration new; declaration create_dir; declaration ensure_dir; declaration lock; declaration open_owned; declaration read; declaration sync_dir; declaration publish; declaration entropy_tests; declaration signed_policy_version_keeps_negative_zero_and_rejects_fraction; declaration revision_generation_propagates_partial_entropy_failure | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tcontrol_policy.rs into its current native target.; 39 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 28–65, 94–107, 186–201, 266–295, 577–782; declaration POLICY_VERSION; declaration PROJECT_VERSION; declaration POLICY_FILE_LIMIT; declaration Credential; fields client_id, secret; declaration fmt; declaration zero_secret; declaration EnrollmentPolicy; fields version, revision, binding, tailnet, tags, preauthorized, admission, default, credential; declaration decode_enrollment_policy; declaration encode_enrollment_policy; declaration load; declaration view; declaration enrollment; declaration check_enrollment; declaration apply_save_or_rotate; declaration apply_default; declaration apply_disable; declaration update | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | POLICY_VERSION: implement the current Project Tailnet enrollment policy duty in tcontrol_policy.rs.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 202–212, 296–316, 783–928; declaration ProjectPolicyEntry; fields version, revision, project, container, binding, enabled; declaration decode_project_entry; declaration encode_project_entry; declaration load_project; declaration validate_project_binding; declaration mutate_project; declaration build_project_view; declaration project | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | ProjectPolicyEntry: implement the current Project Tailnet selection duty in tcontrol_policy.rs.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 929–963; declaration run_binding | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | run_binding: implement the current Project companion lifecycle duty in tcontrol_policy.rs. — current source lib/host/src/tcontrol_policy.rs; lines 929-963; module/caller wiring inspected |

<a id="coverage-21b3658e2add"></a>
<a id="coverage-bc0cb2e3d5c2"></a>

## [lib/host/src/tcontrol_provider.rs](../../../../../lib/host/src/tcontrol_provider.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 213–298, 338–446; current module/import/attribute shell; declaration ExpirationWire; declaration deserialize; declaration seconds; declaration TokenWire; fields access_token, token_type, expires_in; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration parse_rfc3339_nanos; declaration system_nanos; declaration KeyBindingWire; fields reusable, ephemeral, preauthorized, tags; declaration KeyCreateDeviceWire; fields create; declaration KeyCreateCapabilitiesWire; fields devices; declaration KeyResponseWire; fields id, key, created, expires, capabilities, invalid, revoked; declaration fold_ascii_go | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tcontrol_provider.rs into its current native target.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20–212, 299–337, 447–508; declaration TOKEN_URL; declaration API_HOST; declaration PROJECT_KEY_LIFETIME_SECS; declaration KEY_DESCRIPTION; declaration DEFAULT_CURL; declaration ProviderRequest; fields client_id, client_secret, tags, tailnet, preauthorized, token; declaration fmt; declaration ProviderTransport; declaration provider_failed; declaration config_escape; declaration token_form_body; declaration key_create_body; declaration token_config; declaration key_config; declaration run_curl; declaration fetch_token; declaration create_key; declaration validate_token; declaration validate_key; declaration ZERO_NANOS | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | TOKEN_URL: implement the current Project Tailnet enrollment policy duty in tcontrol_provider.rs.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3acfbb55921b"></a>
<a id="coverage-3adc9135c7d9"></a>

## [lib/host/tests/tcontrol_oracle.rs](../../../../../lib/host/tests/tcontrol_oracle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–304, 354–420, 494–774, 878–912, 1034–1043, 1106–1204, 1378–1966, 2140–2200, 2297–2312, 2362–2460, 2522–2648, 2761–2818, 2901–2916; current module/import/attribute shell; declaration json; declaration project; declaration sha256; declaration tailnet_companion; declaration tailnet_domain; declaration tcontrol; declaration tcontrol_enroll; declaration tcontrol_native; declaration tcontrol_policy; declaration tcontrol_provider; declaration tcontrol_wire; declaration SCRATCH_COUNTER; declaration scratch; declaration soon; declaration policy_store; declaration accept_check; declaration wire_revision_and_hex_patterns; declaration wire_tag_client_and_network_patterns; declaration wire_magic_dns_vectors; declaration wire_address_vectors; declaration wire_prefix_exit_ip_and_first_ipv4; declaration wire_peer_view; declaration wire_host_request_decode; declaration wire_host_request_validate; declaration enrollment_fixture; declaration wire_authentication_url; declaration host_fixture; declaration enrollment_view_fixture; declaration wire_view_validators; declaration wire_project_view_validator; declaration wire_encoder_goldens; declaration policy_does_not_convert_or_delete_retained_credentials; declaration PID; declaration inspect_req; declaration runtime_seed; declaration policy_project_runtime_selection_and_bindings; declaration NATIVE_STATUS; declaration NATIVE_PREFS; declaration GO_REVISION; declaration GO_FRESH_REVISION; declaration GO_HOSTVIEW; declaration fixture_transport; declaration FakeRun; declaration FakeExec; fields f; declaration run; declaration boom_exec; declaration native_observe_fixture_matches_go_goldens; declaration native_fresh_daemon_omits_false_node_key; declaration native_node_key_optional_but_strict; declaration native_unavailable_kinds; declaration native_revision_tracks_identity_without_release_veto; declaration native_sha256_matches_reference; declaration native_signin_reauth_preserves_prefs; declaration native_exit_node_and_logout_confirm; declaration native_offline_exit_conflicts_and_retained_id_stays_unconfirmed; declaration native_up_notifications_and_command_bounds; declaration native_local_request_over_real_socket; declaration native_run_status_validate; declaration native_cli_client_vectors; declaration provider_rfc3339_matches_go_vectors; declaration rfc3339; declaration control_options; declaration GO_SETTINGS; declaration GO_OPTIONS; declaration control_settings_and_options_match_go_bytes; declaration control_host_action_cycle; declaration extract_revision; declaration seed_enrollment; declaration enroll_credential_and_key_guards; declaration control_project_and_binding_through_trait; declaration control_enroll_run_serial_explicit_requests_without_journal; declaration control_cli_passthrough | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/tests/tcontrol_oracle.rs into its current native target.; 73 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 305–340, 421–471, 775–877, 913–1033, 1044–1105, 1205–1377, 1967–2139, 2313–2361, 2461–2521, 2819–2900; declaration wire_enrollment_request_decode; declaration wire_enrollment_request_validate; declaration policy_revision_format; declaration policy_reads_and_checks_are_non_mutating; declaration policy_unsupported_default_has_no_effects; declaration policy_rotation_cas_and_secret_projection; declaration policy_concurrency_and_cancelled_waiter; declaration policy_refuses_unsafe_and_ambiguous_state; declaration policy_publication_failure_neither_rolls_back_nor_replays; declaration policy_project_disabled_until_runtime_exists; declaration policy_project_closed_admission_has_no_reservation; declaration policy_project_enable_cas_and_no_implicit_retarget; declaration policy_project_missing_is_off_while_malformed_fails_safely; declaration provider_request_bodies_match_go_recipes; declaration provider_curl_recipe_keeps_secrets_out_of_argv; declaration provider_token_vectors; declaration token_ok; declaration key_ok; declaration provider_fixture; declaration control_enrollment_save_check_rotate_disable; declaration control_failed_enrollment_can_be_explicitly_retried | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | wire_enrollment_request_decode: implement the current Project Tailnet enrollment policy duty in tcontrol_oracle.rs.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 341–353, 472–493; declaration wire_project_selection_decode; declaration wire_project_selection_validate | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | wire_project_selection_decode: implement the current Project Tailnet selection duty in tcontrol_oracle.rs.; wire_project_selection_validate: implement the current Project Tailnet selection duty in tcontrol_oracle.rs. — current source lib/host/tests/tcontrol_oracle.rs; lines 341-353; module/caller wiring inspected; current source lib/host/tests/tcontrol_oracle.rs; lines 472-493; module/caller wiring inspected |
| 2201–2296, 2649–2760; declaration key_response; declaration provider_key_vectors; declaration control_enroll_run_fences_identity_and_uncertainty | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | key_response: implement the current Project companion lifecycle duty in tcontrol_oracle.rs.; provider_key_vectors: implement the current Project companion lifecycle duty in tcontrol_oracle.rs.; control_enroll_run_fences_identity_and_uncertainty: implement the current Project companion lifecycle duty in tcontrol_oracle.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

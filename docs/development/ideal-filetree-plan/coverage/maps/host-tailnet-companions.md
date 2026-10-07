# Host tailnet companions

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-d5503873ae2d"></a>

## [lib/host/src/tailnet/companion/enroll.rs](../../../../../lib/host/src/tailnet/companion/enroll.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–90; current module/import/attribute shell; declaration enroll_companion; declaration ensure_companion_enrolled; declaration consume_run_key | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/enroll.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3f11f50150ef"></a>

## [lib/host/src/tailnet/companion/execute.rs](../../../../../lib/host/src/tailnet/companion/execute.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–204; current module/import/attribute shell; declaration runtime_command; declaration runtime_podman; declaration run_native_command; declaration inspect_companion; declaration companion_cli; declaration wait_tailnet | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/execute.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8c75d2fac032"></a>

## [lib/host/src/tailnet/companion/identity.rs](../../../../../lib/host/src/tailnet/companion/identity.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–345; current module/import/attribute shell; declaration CompanionRecord; fields id, image, command, running, pid, started, execs; declaration CompanionWire; fields id, image, command, running, pid, started, execs; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration decode_companion_record; declaration companion_identity_matches; declaration companion_command_matches; declaration companion_execs_valid; declaration validate_companion_record; declaration ProcessIdentity; fields userns, netns, uid, gid; declaration parse_stat_fields; declaration parse_process_start; declaration read_process_id_map; declaration read_process_namespace; declaration read_system_boot_id; declaration process_run_identity; declaration match_companion_namespaces; declaration stat_metadata; declaration companion_resolver | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/identity.rs into its current native target.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a7c55f870d8b"></a>

## [lib/host/src/tailnet/companion/identity_tests.rs](../../../../../lib/host/src/tailnet/companion/identity_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–298; current module/import/attribute shell; declaration resolver_uses_actual_inode_rather_than_generated_metadata; declaration record_requires_immutable_cid_recipe_and_running_incarnation; declaration exec_identity_and_command_matchers; declaration inspect_decode_is_strict; declaration preparation_stages_preserve_typed_causes; declaration run_freshness_and_idle_state; declaration namespaces_match_only_exact_incarnation | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/identity_tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f6938e4c5a79"></a>

## [lib/host/src/tailnet/companion/lifecycle_tests.rs](../../../../../lib/host/src/tailnet/companion/lifecycle_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–358; current module/import/attribute shell; declaration wait_tailnet_output_matrix; declaration native_commands_never_leak_diagnostics; declaration start_skipped_when_image_empty; declaration should_start_tailnet_policy_gates; declaration companion_cli_rejects_empty_args_and_missing_state; declaration logout_and_stop_reports_unconfirmed_logout; declaration retire_previous_companion_guards_identity; declaration reconcile_previous_run_freshness; declaration prepare_and_finalize_stage_fast_failures; declaration stop_tailnet_validates_and_needs_state | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/lifecycle_tests.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6b639f322c34"></a>

## [lib/host/src/tailnet/companion/mod.rs](../../../../../lib/host/src/tailnet/companion/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–122; current module/import/attribute shell; declaration RUNTIME_ROOT; declaration COMPANION_INSPECT; declaration identity; declaration TailnetControl; declaration project; declaration run_binding; declaration enroll_run; declaration Companion; fields exec, tailnet, image, enabled_check; declaration preparation_error; declaration stage_error; declaration arg_refs; declaration companion_still_running; declaration is_companion_run_fresh; declaration apply_companion_idle_state; declaration execute; declaration start; declaration enroll; declaration stop; declaration view; declaration tests; declaration identity_tests; declaration lifecycle_tests; declaration view_tests | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/mod.rs into its current native target.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b223a7358c3d"></a>

## [lib/host/src/tailnet/companion/start.rs](../../../../../lib/host/src/tailnet/companion/start.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–222; current module/import/attribute shell; declaration project_tailnet_enabled; declaration should_start_tailnet; declaration wait_project_runtime; declaration admit_tailnet_run; declaration prepare_companion_state; declaration create_fresh_companion; declaration start_companion_if_stopped; declaration activate_companion_container; declaration wait_companion_node; declaration finalize_companion_run; declaration start_tailnet | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/start.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d5d17258d6b1"></a>

## [lib/host/src/tailnet/companion/stop.rs](../../../../../lib/host/src/tailnet/companion/stop.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–132; current module/import/attribute shell; declaration retire_previous_companion; declaration reconcile_previous_run; declaration stop_tailnet; declaration confirm_stopped_resolver; declaration stop_tailnet_run; declaration logout_and_stop_companion | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/stop.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-380e3b93715e"></a>

## [lib/host/src/tailnet/companion/tests.rs](../../../../../lib/host/src/tailnet/companion/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–190; current module/import/attribute shell; declaration TEMP_COUNTER; declaration temp_dir; declaration project_id; declaration file_run; declaration test_view; declaration test_binding; declaration test_request; declaration project_inspect_json; declaration MockExec; fields calls, handler, native; declaration with_handler; declaration calls; declaration run; declaration is_host_native; declaration MockTailnet; fields project_fn, binding_fn, enroll_targets, enroll_err; declaration inert; declaration project; declaration run_binding; declaration enroll_run; declaration deadline; declaration companion_with; declaration image_id | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/tests.rs into its current native target.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f23a69c9edd6"></a>

## [lib/host/src/tailnet/companion/view.rs](../../../../../lib/host/src/tailnet/companion/view.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–144; current module/import/attribute shell; declaration queue_project_tailnet_disable; declaration mark_stopped_project; declaration queue_project_tailnet_start; declaration observe_companion_status; declaration observe_project_tailnet | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/view.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-83ec389004a1"></a>

## [lib/host/src/tailnet/companion/view_tests.rs](../../../../../lib/host/src/tailnet/companion/view_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–240; current module/import/attribute shell; declaration disable_flow_marks_runtime_unconfirmed; declaration queue_start_gates_on_action_and_systemd; declaration mark_stopped_distinguishes_confirmed_stop; declaration observe_passthrough_and_fast_paths; declaration observe_companion_status_fails_closed; declaration project_status_maps_observations; declaration project_has_node_reports_presence; declaration confirm_stopped_resolver_ignores_gone_parent | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Imports and module declarations wire lib/host/src/tailnet/companion/view_tests.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-24fb3f5f96ea"></a>
<a id="rustsoda-hostsrctailnet_companionrs-1"></a>

Former source `rust/soda-host/src/tailnet_companion.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-1b953fb230a2"></a>
<a id="rustsoda-hostsrctailnet_domainrs-1"></a>

Former source `rust/soda-host/src/tailnet_domain.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-7be209e96237"></a>
<a id="rustsoda-hostsrctailnet_filesrs-1"></a>

Former source `rust/soda-host/src/tailnet_files.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-c86ee8e29e87"></a>
<a id="rustsoda-hostsrctailnet_runtimers-1"></a>

Former source `rust/soda-host/src/tailnet_runtime.rs`; consult its pinned earlier Git source and the current coverage disposition.

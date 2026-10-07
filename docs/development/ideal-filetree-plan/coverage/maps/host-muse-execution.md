# Host muse execution

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-481016c83564"></a>

## [lib/host/src/muse/args.rs](../../../../../lib/host/src/muse/args.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–83; current module/import/attribute shell; declaration muse_auth_override; declaration muse_value_flag; declaration muse_positional; declaration muse_provider_arguments; declaration muse_arguments | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/args.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-03ca3b94870d"></a>

## [lib/host/src/muse/argv.rs](../../../../../lib/host/src/muse/argv.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–116; current module/import/attribute shell; declaration muse_command_argv; declaration unit_active_argv; declaration unit_invocation_argv | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/argv.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8b3f9e8a45f8"></a>

## [lib/host/src/muse/codec.rs](../../../../../lib/host/src/muse/codec.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–44; current module/import/attribute shell; declaration muse_command_exit; declaration JsonPacket; declaration split_json_object | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/codec.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4654e7768206"></a>

## [lib/host/src/muse/config.rs](../../../../../lib/host/src/muse/config.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; current module/import/attribute shell; declaration decode_config_view | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/config.rs into its current native target.; decode_config_view: implement the current provider execution integration duty in config.rs. — current source lib/host/src/muse/config.rs; Cargo target and callers; current source lib/host/src/muse/config.rs; lines 7-14; module/caller wiring inspected |

<a id="coverage-98eece53868d"></a>

## [lib/host/src/muse/connection.rs](../../../../../lib/host/src/muse/connection.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; current module/import/attribute shell; declaration MuseConnection; fields id, provider_id, state; declaration muse_connection_authorized; declaration select_muse_connection | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/connection.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2731e2aa3ab0"></a>

## [lib/host/src/muse/execution.rs](../../../../../lib/host/src/muse/execution.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–252; current module/import/attribute shell; declaration verify_guest_binary; declaration prepare_execution; declaration reserve_execution; declaration deliver_execution; declaration control_execution | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/execution.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-faa954fb9446"></a>

## [lib/host/src/muse/inspect.rs](../../../../../lib/host/src/muse/inspect.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101; current module/import/attribute shell; declaration MuseInspection; fields id, pid, project, running, privileged, userns; declaration deserialize; declaration InspectionVisitor; declaration Value; declaration expecting; declaration visit_map; declaration muse_peer_alive; declaration sleep_until | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/inspect.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a77ceec81163"></a>

## [lib/host/src/muse/launch.rs](../../../../../lib/host/src/muse/launch.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–29, 88–314; current module/import/attribute shell; declaration MuseLaunch; fields runtime; declaration new; declaration serve_one; declaration serve_connection; declaration shell; declaration shell_inner; declaration finish_unconfirmed; declaration control_loop | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/launch.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 30–87; declaration serve | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | serve: implement the current Unix-socket service and process lifetime duty in launch.rs. — current source lib/host/src/muse/launch.rs; lines 30-87; module/caller wiring inspected |

<a id="coverage-f0ade161f531"></a>

## [lib/host/src/muse/mod.rs](../../../../../lib/host/src/muse/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 27–55, 64–95; current module/import/attribute shell; declaration args; declaration connection; declaration runtime_types; declaration validate; declaration argv; declaration execution; declaration inspect; declaration launch; declaration nested; declaration observe; declaration operate; declaration ops; declaration request; declaration resolve; declaration socket; declaration spawn; declaration stage; declaration stop; declaration tests | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/mod.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20–26, 56–59; declaration wire; declaration codec | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | wire: implement the current wire decoding, encoding, and representation conversion duty in mod.rs.; codec: implement the current wire decoding, encoding, and representation conversion duty in mod.rs. — current source lib/host/src/muse/mod.rs; lines 20-26; module/caller wiring inspected; current source lib/host/src/muse/mod.rs; lines 56-59; module/caller wiring inspected |
| 60–63; declaration config | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | config: implement the current configuration and safe filesystem primitives duty in mod.rs. — current source lib/host/src/muse/mod.rs; lines 60-63; module/caller wiring inspected |

<a id="coverage-c6924a75def1"></a>

## [lib/host/src/muse/nested.rs](../../../../../lib/host/src/muse/nested.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–252; current module/import/attribute shell; declaration register_nested; declaration registration_authority; declaration registered_child; declaration validate_nested; declaration nested_namespace; declaration actor_account; declaration AccountWire; fields username, home_dir, json; declaration nested_caller | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/nested.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8bd149422338"></a>

## [lib/host/src/muse/observe.rs](../../../../../lib/host/src/muse/observe.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–90; current module/import/attribute shell; declaration podman; declaration guest; declaration guest_refs; declaration inspect | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/observe.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ecaca1312fab"></a>

## [lib/host/src/muse/operate.rs](../../../../../lib/host/src/muse/operate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–200; current module/import/attribute shell; declaration muse_operation; declaration project_operation; declaration cleanup_execution_state; declaration invoke_state; declaration retire_mount; declaration retire_execution_files | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/operate.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5a4f1faaf7de"></a>

## [lib/host/src/muse/ops.rs](../../../../../lib/host/src/muse/ops.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; current module/import/attribute shell; declaration muse_resize; declaration state_container; declaration prepare_muse_listener_dir | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/ops.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4722a92bdd49"></a>

## [lib/host/src/muse/request.rs](../../../../../lib/host/src/muse/request.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; current module/import/attribute shell; declaration MuseRequest; fields request, files; declaration muse_request_from_fd; declaration muse_descriptors_valid | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/request.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b321df63ff35"></a>

## [lib/host/src/muse/resolve.rs](../../../../../lib/host/src/muse/resolve.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–195; current module/import/attribute shell; declaration resolve_project; declaration kernel_caller; declaration registered_caller; declaration resolve; declaration resolve_inner; declaration project_account; declaration project_actor; declaration authorized_caller | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/resolve.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a93797f2cd0c"></a>

## [lib/host/src/muse/runtime_types.rs](../../../../../lib/host/src/muse/runtime_types.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101; current module/import/attribute shell; declaration MusePeer; fields pid, uid, gid, pidfd; declaration MuseCaller; fields project, container, login, home, namespace, child, registration, actor, uid, gid, project_pid, nested_pid, muse_allowed; declaration MuseNested; fields parent, project, child, namespace, registration, actor, pid, muse; declaration MuseExecution; fields caller, request, lease, binding, path, unit; declaration MuseHooks; declaration acquire; declaration attach; declaration end; declaration authorize; declaration nested_authorize; declaration select; declaration MuseRuntime; fields exec, hooks, binary_version, binary_sha256, nested; declaration new | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/runtime_types.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-915ae580508e"></a>

## [lib/host/src/muse/socket.rs](../../../../../lib/host/src/muse/socket.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–46; current module/import/attribute shell; declaration SO_PEERPIDFD; declaration muse_peer_from_fd | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/socket.rs into its current native target.; SO_PEERPIDFD: implement the current provider execution integration duty in socket.rs.; muse_peer_from_fd: implement the current provider execution integration duty in socket.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eb5d3264c4a1"></a>

## [lib/host/src/muse/spawn.rs](../../../../../lib/host/src/muse/spawn.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; current module/import/attribute shell; declaration spawn_execution | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/spawn.rs into its current native target.; spawn_execution: implement the current provider execution integration duty in spawn.rs. — current source lib/host/src/muse/spawn.rs; Cargo target and callers; current source lib/host/src/muse/spawn.rs; lines 4-24; module/caller wiring inspected |

<a id="coverage-52992bfb00e9"></a>

## [lib/host/src/muse/stage.rs](../../../../../lib/host/src/muse/stage.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–234; current module/import/attribute shell; declaration stage; declaration stage_files; declaration stage_config; declaration populate_config; declaration auth_mount_target; declaration copy_nested_config | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/stage.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ab20334836a2"></a>

## [lib/host/src/muse/stop.rs](../../../../../lib/host/src/muse/stop.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; current module/import/attribute shell; declaration stop_execution; declaration validate_muse_binding; declaration await_unit | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/stop.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-81f2cd1ead4f"></a>

## [lib/host/src/muse/tests.rs](../../../../../lib/host/src/muse/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2291; current module/import/attribute shell; declaration PID; declaration TID; declaration IID; declaration CID; declaration PIN; declaration TMP_COUNTER; declaration ENV_LOCK; declaration deadline; declaration test_tmp; declaration RecordedCall; declaration FakeExec; fields calls, script; declaration new; declaration calls; declaration run; declaration ok; declaration err; declaration FakeHooks; fields select_out, acquire_out, attach_out, end_calls, end_out, authorize_out, nested_out, selects, acquires, attaches; declaration acquire; declaration attach; declaration end; declaration authorize; declaration nested_authorize; declaration select; declaration runtime; declaration caller; declaration lease_fixture; declaration launch_request_validate_matrix; declaration launch_decode_matrix; declaration muse_arguments_matrix; declaration connection_directory_matrix; declaration cgroup_matrix; declaration uidmap_matrix; declaration passwd_and_modes_matrix; declaration registration_and_child_matrix; declaration readonly_mount_matrix; declaration elf_header; declaration elf_matrix; declaration signal_and_root_matrix; declaration muse_command_goldens; declaration inspect_flows; declaration actor_account_flows; declaration reserve_execution_flows; declaration stage_flows; declaration deliver_execution_flows; declaration stop_execution_flows; declaration retire_mount_matrix; declaration validate_and_ops_matrix; declaration state_container_matrix; declaration config_view_matrix; declaration control_execution_flows; declaration open_pty_pair; declaration pty_size; declaration open_pipe_pair; declaration verify_guest_binary_flows; declaration seqpacket_pair; declaration send_with_fds; declaration peer_attestation; declaration accepted_muse_connection_is_cloexec; declaration request_parsing_matrix; declaration rejected_received_rights_are_closed; declaration truncated_rights_are_rejected_and_closed; declaration oversized_request_datagram_is_rejected; declaration request_validation_failure_closes_received_rights; declaration command_exit_matrix; declaration json_split_matrix; declaration control_loop_matrix; declaration serve_shutdown_and_listener_setup; declaration muse_host_environment_filters_meta_key | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/tests.rs into its current native target.; 70 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3ca060fa3dfb"></a>

## [lib/host/src/muse/validate.rs](../../../../../lib/host/src/muse/validate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–538; current module/import/attribute shell; declaration MUSE_INSPECT; declaration MUSE_CHILD_INSPECT; declaration muse_project_cgroup; declaration muse_mapped_uid; declaration muse_passwd_valid; declaration muse_account_node; declaration muse_account_modes; declaration muse_registration_valid; declaration muse_child_pid; declaration ChildInspection; fields id, pid, running; declaration deserialize; declaration ChildVisitor; declaration Value; declaration expecting; declaration visit_map; declaration SoftString; declaration SoftStringVisitor; declaration visit_str; declaration visit_string; declaration visit_unit; declaration visit_bool; declaration visit_i64; declaration visit_u64; declaration visit_f64; declaration visit_seq; declaration SoftBool; declaration SoftBoolVisitor; declaration Mount; fields source, destination, rw; declaration MountVisitor; declaration MountItem; declaration ItemVisitor; declaration muse_readonly_mount; declaration muse_elf; declaration host_go_arch; declaration muse_signal; declaration muse_project_credential_root; declaration muse_delivery_valid; declaration muse_host_environment | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/validate.rs into its current native target.; 71 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f83664f68888"></a>

## [lib/host/src/muse/wire.rs](../../../../../lib/host/src/muse/wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–372; current module/import/attribute shell; declaration MUSE_LAUNCH_SOCKET; declaration MUSE_CONNECTION_SETTING; declaration NestedRegistration; fields child_id, actor_id, registration_id, muse; declaration deserialize; declaration RegistrationVisitor; declaration Value; declaration expecting; declaration visit_map; declaration LaunchRequest; fields home, register, config_home, term, connection_id, cwd, args, tty, cols, rows; declaration RequestVisitor; declaration LaunchControl; fields signal, cols, rows; declaration ControlVisitor; declaration LaunchExit; fields code, error; declaration denied; declaration cleanup_unconfirmed; declaration encode; declaration encode_line; declaration launch_text; declaration launch_absolute_path; declaration launch_arguments_valid; declaration validate; declaration registration_valid; declaration decode | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse/wire.rs into its current native target.; 33 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a546243813d1"></a>
<a id="coverage-32590f33ca8d"></a>

## [lib/host/src/muse_serve.rs](../../../../../lib/host/src/muse_serve.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–167; current module/import/attribute shell; declaration CLEANUP_SECS; declaration WAIT_POLL_MS; declaration LISTEN_BACKLOG; declaration start; declaration muse; declaration finish_start; declaration spawn_detached; declaration wait_bounded | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/src/muse_serve.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 168–196; declaration open_muse_listener | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | open_muse_listener: implement the current Unix-socket service and process lifetime duty in muse_serve.rs. — current source lib/host/src/muse_serve.rs; lines 168-196; module/caller wiring inspected |

<a id="coverage-9325ccc40087"></a>

## [lib/host/src/terminal/factory/muse/commands.rs](../../../../../lib/host/src/terminal/factory/muse/commands.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113; current module/import/attribute shell; declaration factory_muse_supervisor; declaration muse_setup_script; declaration muse_start_gate_script; declaration muse_exec_argv | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/commands.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-500ebd7e3075"></a>

## [lib/host/src/terminal/factory/muse/lifecycle.rs](../../../../../lib/host/src/terminal/factory/muse/lifecycle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; current module/import/attribute shell; declaration factory_muse_wait; declaration factory_muse_validate; declaration factory_muse_stop; declaration factory_muse_stop_unbound; declaration factory_muse_capture; declaration factory_muse_live; declaration factory_muse_output | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/lifecycle.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2266a92a8645"></a>

## [lib/host/src/terminal/factory/muse/mod.rs](../../../../../lib/host/src/terminal/factory/muse/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; current module/import/attribute shell; declaration commands; declaration lifecycle; declaration paths; declaration reserve; declaration start; declaration tests | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/mod.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e50183514c01"></a>

## [lib/host/src/terminal/factory/muse/paths.rs](../../../../../lib/host/src/terminal/factory/muse/paths.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; current module/import/attribute shell; declaration FactoryMusePaths; fields checkout, run_dir, home, muse_config, prompt, marker, started, stop, pid_file, output, stdout, auth, credential, guest; declaration factory_muse_run_paths; declaration factory_muse_paths; declaration factory_muse_binding; declaration factory_muse_guest | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/paths.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b7c56ea98d34"></a>

## [lib/host/src/terminal/factory/muse/reserve.rs](../../../../../lib/host/src/terminal/factory/muse/reserve.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 38–193; current module/import/attribute shell; declaration factory_muse_reserve; declaration factory_muse_setup; declaration factory_muse_stage | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/reserve.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 18–37; declaration verify_muse_harness | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | verify_muse_harness: implement the current candidate verification assessment duty in reserve.rs. — current source lib/host/src/terminal/factory/muse/reserve.rs; lines 18-37; module/caller wiring inspected |

<a id="coverage-3a1e9c1ca59e"></a>

## [lib/host/src/terminal/factory/muse/start.rs](../../../../../lib/host/src/terminal/factory/muse/start.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50; current module/import/attribute shell; declaration factory_muse_start | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/start.rs into its current native target.; factory_muse_start: implement the current factory run lifecycle and intervention duty in start.rs. — current source lib/host/src/terminal/factory/muse/start.rs; Cargo target and callers; current source lib/host/src/terminal/factory/muse/start.rs; lines 14-50; module/caller wiring inspected |

<a id="coverage-66b127a2df7d"></a>

## [lib/host/src/terminal/factory/muse/tests/commands.rs](../../../../../lib/host/src/terminal/factory/muse/tests/commands.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; current module/import/attribute shell; declaration supervisor_shape; declaration setup_and_gate_scripts | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/tests/commands.rs into its current native target.; supervisor_shape: implement the current factory run lifecycle and intervention duty in commands.rs.; setup_and_gate_scripts: implement the current factory run lifecycle and intervention duty in commands.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4f14b9c493e9"></a>

## [lib/host/src/terminal/factory/muse/tests/common.rs](../../../../../lib/host/src/terminal/factory/muse/tests/common.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–163; current module/import/attribute shell; declaration PID; declaration RID; declaration IID; declaration CID; declaration PREP; declaration ROLE; declaration COMMIT; declaration PIN; declaration TMP_COUNTER; declaration deadline; declaration test_tmp; declaration RecordedCall; declaration FakeExec; fields calls, script; declaration new; declaration calls; declaration run; declaration ok; declaration err; declaration make_service; declaration write_harness; declaration reserve_harness; declaration inspect_json; declaration muse_run; declaration muse_run_dir; declaration muse_lease; declaration euid | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/tests/common.rs into its current native target.; 27 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6b0201c44247"></a>

## [lib/host/src/terminal/factory/muse/tests/lifecycle.rs](../../../../../lib/host/src/terminal/factory/muse/tests/lifecycle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; current module/import/attribute shell; declaration wait_output_live; declaration stop_and_capture; declaration finish_denied_for_muse | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/tests/lifecycle.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d41472a0b623"></a>

## [lib/host/src/terminal/factory/muse/tests/mod.rs](../../../../../lib/host/src/terminal/factory/muse/tests/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6; declaration commands; declaration common; declaration lifecycle; declaration paths; declaration reserve; declaration start | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | commands: implement the current factory run lifecycle and intervention duty in mod.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0579d77d2d53"></a>

## [lib/host/src/terminal/factory/muse/tests/paths.rs](../../../../../lib/host/src/terminal/factory/muse/tests/paths.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–63; current module/import/attribute shell; declaration paths_and_guest; declaration binding_gates | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/tests/paths.rs into its current native target.; paths_and_guest: implement the current factory run lifecycle and intervention duty in paths.rs.; binding_gates: implement the current factory run lifecycle and intervention duty in paths.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-309ab10a17dd"></a>

## [lib/host/src/terminal/factory/muse/tests/reserve.rs](../../../../../lib/host/src/terminal/factory/muse/tests/reserve.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 53–220; current module/import/attribute shell; declaration reserve_denial_pins; declaration reserve_success_argv_sequence; declaration reserve_stage_missing_guest | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/tests/reserve.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 11–52; declaration verify_harness_matrix | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | verify_harness_matrix: implement the current candidate verification assessment duty in reserve.rs. — current source lib/host/src/terminal/factory/muse/tests/reserve.rs; lines 11-52; module/caller wiring inspected |

<a id="coverage-df10841a3114"></a>

## [lib/host/src/terminal/factory/muse/tests/start.rs](../../../../../lib/host/src/terminal/factory/muse/tests/start.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–84; current module/import/attribute shell; declaration start_flows | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/muse/tests/start.rs into its current native target.; start_flows: implement the current factory run lifecycle and intervention duty in start.rs. — current source lib/host/src/terminal/factory/muse/tests/start.rs; Cargo target and callers; current source lib/host/src/terminal/factory/muse/tests/start.rs; lines 7-84; module/caller wiring inspected |

<a id="coverage-2ec56c0c7f39"></a>
<a id="coverage-27b47c7ed802"></a>

## [lib/host/tests/muse_serve_oracle.rs](../../../../../lib/host/tests/muse_serve_oracle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–944; current module/import/attribute shell; declaration account; declaration domain; declaration json; declaration muse; declaration muse_serve; declaration net; declaration nist; declaration project; declaration sha256; declaration ssh; declaration terminal; declaration PID; declaration TID; declaration IID; declaration CID; declaration CHID; declaration RID; declaration PIN; declaration deadline; declaration RecordedCall; declaration FakeExec; fields calls, script; declaration new; declaration calls; declaration run; declaration ok; declaration err; declaration FakeHooks; fields end_calls; declaration acquire; declaration attach; declaration end; declaration authorize; declaration nested_authorize; declaration select; declaration runtime; declaration lease_fixture; declaration delivery_fixture; declaration valid_launch_request; declaration self_pid; declaration dead_peer; declaration live_peer; declaration TMP_COUNTER; declaration work_tmp; declaration sock_path; declaration cleanup; declaration seqpacket_connect; declaration close_fd; declaration devnull; declaration send_with_fds; declaration recv_line; declaration start_rejects_invalid_request_without_touching_exec; declaration start_denies_dead_peer_without_touching_exec; declaration start_denies_live_peer_outside_any_project; declaration muse_validate_ok_echoes_delivery; declaration muse_validate_stale_on_invocation_mismatch; declaration muse_stop_ok_without_custody_return; declaration muse_start_action_denied_like_go; declaration muse_finish_action_denied_like_go; declaration muse_unknown_action_denied_after_invocation_check; declaration muse_malformed_delivery_never_calls_out; declaration open_listener_binds_seqpacket_world_writable; declaration open_listener_refuses_nonempty_dir; declaration open_listener_reports_empty_before_occupied; declaration open_listener_rejects_overlong_path; declaration spawn_existing_serve; declaration launch_wire; declaration register_wire; declaration DENIED_LINE; declaration serve_loopback_register_denied; declaration serve_loopback_launch_denied; declaration serve_loopback_garbage_denied; declaration serve_loopback_never_echoes_request_bytes; declaration serve_loopback_shutdown_clean_without_connections; declaration oracle_launch_exit_wire_bytes; declaration oracle_launch_request_go_shapes; declaration oracle_muse_arguments_vectors; declaration oracle_muse_command_argv_project_and_nested | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire lib/host/tests/muse_serve_oracle.rs into its current native target.; 78 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-5ff369a7e67e"></a>

Former source `rust/soda-host/src/muse.rs`; consult its pinned earlier Git source and the current coverage disposition.

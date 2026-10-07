# Host factory runs

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-8ab7a036a168"></a>

## [lib/host/src/factory/artifacts.rs](../../../../../lib/host/src/factory/artifacts.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–111; current module/import/attribute shell; declaration output; declaration export | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/artifacts.rs into its current native target.; output: implement the current factory run lifecycle and intervention duty in artifacts.rs.; export: implement the current factory run lifecycle and intervention duty in artifacts.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9b9f55cdd22f"></a>

## [lib/host/src/factory/candidate.rs](../../../../../lib/host/src/factory/candidate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 107–120; current module/import/attribute shell; declaration run_reason | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/candidate.rs into its current native target.; run_reason: implement the current factory run lifecycle and intervention duty in candidate.rs. — current source lib/host/src/factory/candidate.rs; Cargo target and callers; current source lib/host/src/factory/candidate.rs; lines 107-120; module/caller wiring inspected |
| 16–106, 121–152; declaration inspect_candidate; declaration CANDIDATE_INSPECT_SCRIPT | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | inspect_candidate: implement the current candidate verification assessment duty in candidate.rs.; CANDIDATE_INSPECT_SCRIPT: implement the current candidate verification assessment duty in candidate.rs. — current source lib/host/src/factory/candidate.rs; lines 16-106; module/caller wiring inspected; current source lib/host/src/factory/candidate.rs; lines 121-152; module/caller wiring inspected |

<a id="coverage-811394982f6b"></a>

## [lib/host/src/factory/confirmation.rs](../../../../../lib/host/src/factory/confirmation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20, 34–113, 131–146, 163–229; current module/import/attribute shell; declaration confirm_factory_inspect; declaration confirm_factory_harness; declaration confirm_factory_stop; declaration confirm_factory_output; declaration confirm_factory_export; declaration confirm_factory_takeover; declaration confirm_prepare; declaration confirm_inspect_preparation; declaration confirm_stop_preparation; declaration confirm_hold_preparation; declaration factory_not_found_status; declaration factory_output_status_error; declaration factory_export_status_error | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/confirmation.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 21–33; declaration confirm_factory_launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | confirm_factory_launch: implement the current factory assignment and dispatch duty in confirmation.rs. — current source lib/host/src/factory/confirmation.rs; lines 21-33; module/caller wiring inspected |
| 114–130, 147–162, 230–236; declaration confirm_factory_candidate_inspect; declaration confirm_prepare_candidate; declaration factory_candidate_status_error | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | confirm_factory_candidate_inspect: implement the current candidate verification assessment duty in confirmation.rs.; confirm_prepare_candidate: implement the current candidate verification assessment duty in confirmation.rs.; factory_candidate_status_error: implement the current candidate verification assessment duty in confirmation.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-21f1c77fdb69"></a>

## [lib/host/src/factory/deadline.rs](../../../../../lib/host/src/factory/deadline.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; current module/import/attribute shell; declaration NANOS_PER_SEC; declaration parse_deadline; declaration system_nanos_now; declaration deadline_is_zero; declaration seconds_until; declaration drive_deadline; declaration min_instant; declaration cleanup_deadline; declaration live_deadline | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/deadline.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a5ebaf3fcf74"></a>

## [lib/host/src/factory/finish.rs](../../../../../lib/host/src/factory/finish.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–99, 113–284; current module/import/attribute shell; declaration fail_run; declaration abandon_run; declaration refresh_stopped; declaration stop_failed_start; declaration stop_timed_out; declaration finish_run | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/finish.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 100–112; declaration launch_yielded | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | launch_yielded: implement the current factory assignment and dispatch duty in finish.rs. — current source lib/host/src/factory/finish.rs; lines 100-112; module/caller wiring inspected |

<a id="coverage-a5ff812d1119"></a>

## [lib/host/src/factory/identity.rs](../../../../../lib/host/src/factory/identity.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–322; current module/import/attribute shell; declaration Binding; fields child_id, uid, gid, scope, credential_root, invocation_id, kind, id, project, login, generation; declaration encode_into; declaration deserialize; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map; declaration Lease; fields repository_id, provider_id, id, connection_id, generation, actor_id, project_id, execution_id, kind, role, deadline, grant_id, grant_revision, binding; declaration with_binding; declaration LeaseVisitor; declaration AcquireRequest; fields provider_id, execution_id, actor_id, connection_id, project_id, kind, deadline, role | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/identity.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c048055141bf"></a>

## [lib/host/src/factory/inspect.rs](../../../../../lib/host/src/factory/inspect.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–163; current module/import/attribute shell; declaration inspect; declaration takeover; declaration tests; declaration inspect_matrix; declaration inspect_running_probes_liveness | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/inspect.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cfbf25f6de38"></a>

## [lib/host/src/factory/launch.rs](../../../../../lib/host/src/factory/launch.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16, 55–209; current module/import/attribute shell; declaration drive; declaration consume_start; declaration update_receipt | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/launch.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 17–54; declaration launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | launch: implement the current factory assignment and dispatch duty in launch.rs. — current source lib/host/src/factory/launch.rs; lines 17-54; module/caller wiring inspected |

<a id="coverage-6010490fcf3d"></a>

## [lib/host/src/factory/mod.rs](../../../../../lib/host/src/factory/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–34, 36–40, 42–230; current module/import/attribute shell; declaration artifacts; declaration confirmation; declaration deadline; declaration finish; declaration identity; declaration inspect; declaration receipt; declaration requests; declaration run; declaration state; declaration stop; declaration tests; declaration FactoryTerminal; declaration harness_family; declaration harness_version; declaration harness_sha256; declaration reserve; declaration start; declaration wait; declaration stop_unbound; declaration capture; declaration live; declaration output; declaration takeover_copy; declaration export_bundle; declaration FactoryBroker; declaration acquire; declaration register; declaration return_lease; declaration reconcile_lease; declaration execution_is_terminal; declaration close_execution; declaration Secret; declaration drop; declaration RunLock; fields _file; declaration Factory; fields exec, terminal, broker, state_dir; declaration open_factory; declaration harness_pin | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/mod.rs into its current native target.; 40 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 35; declaration candidate | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | candidate: implement the current candidate verification assessment duty in mod.rs. — current source lib/host/src/factory/mod.rs; lines 35-35; module/caller wiring inspected |
| 41; declaration launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | launch: implement the current factory assignment and dispatch duty in mod.rs. — current source lib/host/src/factory/mod.rs; lines 41-41; module/caller wiring inspected |

<a id="coverage-4fdd7235902d"></a>

## [lib/host/src/factory/receipt.rs](../../../../../lib/host/src/factory/receipt.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–435; current module/import/attribute shell; declaration FactoryReceipt; fields run, lease, binding, exit_code, generation, phase, started, delivered, credential_returned, output, retirement, reason; declaration validate; declaration lease_id; declaration encode; declaration deserialize; declaration ReceiptVisitor; declaration Value; declaration expecting; declaration visit_map; declaration receipt_terminal; declaration receipt_stop_owned; declaration truncate_output; declaration receipt_state; declaration receipt_path; declaration lock_path; declaration lock_run; declaration load_receipt; declaration store_receipt; declaration store_tombstone; declaration write_receipt_file | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/receipt.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ad4c4fbba447"></a>

## [lib/host/src/factory/requests.rs](../../../../../lib/host/src/factory/requests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–278, 288–359, 370–384; current module/import/attribute shell; declaration FactoryInspect; fields project, id; declaration decode; declaration validate; declaration FactoryStop; fields project, id; declaration FactoryTakeover; fields project, id, member; declaration TakeoverResult; fields id, project, member, destination, reused; declaration encode; declaration FactoryOutput; fields project, id, offset, limit; declaration FactoryOutputState; fields exit_code, live, terminal, truncated, gap, id, project, phase, container, unit, invocation, total, offset, next, data, reason; declaration FactoryExport; fields project, id, role, preparation, candidate; declaration FactoryExportState; fields id, project, phase, container, candidate, bundle; declaration deserialize; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/requests.rs into its current native target.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 279–287, 360–369; declaration FactoryCandidateInspect; fields project, id; declaration FactoryCandidateState; fields id, project, container, candidate, dirty | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | FactoryCandidateInspect: implement the current candidate verification assessment duty in requests.rs.; FactoryCandidateState: implement the current candidate verification assessment duty in requests.rs. — current source lib/host/src/factory/requests.rs; lines 279-287; module/caller wiring inspected; current source lib/host/src/factory/requests.rs; lines 360-369; module/caller wiring inspected |

<a id="coverage-11d4c69c78e9"></a>

## [lib/host/src/factory/run.rs](../../../../../lib/host/src/factory/run.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–305, 313–363, 366–426; current module/import/attribute shell; declaration FACTORY_STATE_ROOT; declaration FACTORY_HARNESS_CODEX; declaration FACTORY_HARNESS_MUSE; declaration valid_harness_family; declaration FACTORY_APPROVED; declaration FACTORY_RUNNING; declaration FACTORY_COMPLETED; declaration FACTORY_FAILED; declaration FACTORY_STOPPED; declaration FACTORY_UNCERTAIN; declaration MAX_FACTORY_PROMPT; declaration MAX_FACTORY_OUTPUT; declaration MAX_FACTORY_OUTPUT_READ; declaration MAX_FACTORY_OUTPUT_WINDOW; declaration MAX_FACTORY_OUTPUT_OFFSET; declaration MAX_FACTORY_EXPORT_BUNDLE; declaration IDENTITY_FACTORY; declaration EXECUTION_TERMINAL; declaration ERR_RUN_NOT_FOUND; declaration ERR_RUN_STALE; declaration FACTORY_CLEANUP_SECS; declaration FACTORY_LIVE_SECS; declaration FACTORY_DEADLINE_BOUND_NANOS; declaration valid_factory_phase; declaration valid_factory_run_id; declaration valid_harness_version; declaration factory_unit_name; declaration factory_run_paths; declaration takeover_destination; declaration takeover_source; declaration FactoryError; declaration message; declaration msg; declaration fmt; declaration FactoryRun; fields deadline, actor, id, project, role, preparation, harness, harness_vers, model, assignment, source_commit, connection; declaration validate; declaration encode_into; declaration deserialize; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map; declaration PromptBytes; declaration BytesVisitor; declaration visit_str; declaration visit_string; declaration visit_seq; declaration decode | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/run.rs into its current native target.; 57 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 306–312, 364–365; declaration FactoryLaunch; fields run, prompt, harness_sha256; declaration LaunchVisitor | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | FactoryLaunch: implement the current factory assignment and dispatch duty in run.rs.; LaunchVisitor: implement the current factory assignment and dispatch duty in run.rs. — current source lib/host/src/factory/run.rs; lines 306-312; module/caller wiring inspected; current source lib/host/src/factory/run.rs; lines 364-365; module/caller wiring inspected |

<a id="coverage-3c4667b3e310"></a>

## [lib/host/src/factory/state.rs](../../../../../lib/host/src/factory/state.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–140; current module/import/attribute shell; declaration OutputSlice; fields data, total, offset, truncated, gap; declaration FactoryState; fields exit_code, generation, uid, gid, credential_returned, live, delivered, id, project, role, phase, container, unit, invocation, login, lease_id, output, retirement, reason; declaration encode; declaration FactoryHarnessPin; fields harness, version, sha256, image; declaration validate | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/state.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-31beaee9d560"></a>

## [lib/host/src/factory/stop.rs](../../../../../lib/host/src/factory/stop.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–212; current module/import/attribute shell; declaration stop; declaration reconcile_run_credential; declaration record_stop_outcome | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/stop.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2c1d935b3acb"></a>

## [lib/host/src/factory/tests/artifacts.rs](../../../../../lib/host/src/factory/tests/artifacts.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–379; current module/import/attribute shell; declaration takeover_success_and_refusals; declaration output_without_binding_reports_phase_only; declaration output_slice_reports_binding_and_cursors; declaration export_success_denied_and_bounds | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/artifacts.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3cb19779a2b8"></a>

## [lib/host/src/factory/tests/candidate.rs](../../../../../lib/host/src/factory/tests/candidate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; current module/import/attribute shell | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/candidate.rs into its current native target. — current source lib/host/src/factory/tests/candidate.rs; Cargo target and callers |
| 8–261; declaration candidate_script_pins_cutover_shape; declaration inspect_candidate_reports_clean_and_dirty; declaration inspect_candidate_incarnation_matrix_is_stale; declaration inspect_candidate_output_shapes | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | candidate_script_pins_cutover_shape: implement the current candidate verification assessment duty in candidate.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-65fab656e0a9"></a>

## [lib/host/src/factory/tests/common.rs](../../../../../lib/host/src/factory/tests/common.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–92, 101–363; current module/import/attribute shell; declaration TAG_COUNTER; declaration test_state_dir; declaration deadline; declaration civil_from_days; declaration deadline_text; declaration run_id; declaration project_id; declaration prep_id; declaration container_id; declaration sample_run; declaration sample_lease; declaration sample_binding; declaration TestFactory; declaration test_factory; declaration receipt_bytes; declaration fixed_run; declaration fixed_run_json; declaration fixed_binding; declaration fixed_binding_json; declaration fixed_lease; declaration fixed_lease_json; declaration dummy_factory; declaration wired_factory; declaration drive_to_start; declaration write_running; declaration wired_factory_exec; declaration preparation_target; declaration sample_state; declaration sample_preparation; declaration sample_prepare_state | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/common.rs into its current native target.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 93–100; declaration sample_launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | sample_launch: implement the current factory assignment and dispatch duty in common.rs. — current source lib/host/src/factory/tests/common.rs; lines 93-100; module/caller wiring inspected |

<a id="coverage-da4bbfe7f41a"></a>

## [lib/host/src/factory/tests/confirmation.rs](../../../../../lib/host/src/factory/tests/confirmation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–232; current module/import/attribute shell; declaration factory_facade_confirmations; declaration factory_status_mappings; declaration prepare_facade_confirmations | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/confirmation.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eae9ace83203"></a>

## [lib/host/src/factory/tests/finish.rs](../../../../../lib/host/src/factory/tests/finish.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 89–169; current module/import/attribute shell; declaration launch_timed_out_stop_failure_is_uncertain; declaration launch_finish_maps_exit_codes; declaration launch_finish_uncertainty_matrix | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/finish.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 5–88; declaration launch_failed_start_retires_cleanly; declaration launch_failed_start_uncertainty_matrix; declaration launch_timed_out_wait_records_deadline | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | launch_failed_start_retires_cleanly: implement the current factory assignment and dispatch duty in finish.rs.; launch_failed_start_uncertainty_matrix: implement the current factory assignment and dispatch duty in finish.rs.; launch_timed_out_wait_records_deadline: implement the current factory assignment and dispatch duty in finish.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-16b47ecc683a"></a>

## [lib/host/src/factory/tests/launch.rs](../../../../../lib/host/src/factory/tests/launch.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 225–244; current module/import/attribute shell; declaration launch_on_tombstoned_id_returns_stop | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/launch.rs into its current native target.; launch_on_tombstoned_id_returns_stop: implement the current factory run lifecycle and intervention duty in launch.rs. — current source lib/host/src/factory/tests/launch.rs; Cargo target and callers; current source lib/host/src/factory/tests/launch.rs; lines 225-244; module/caller wiring inspected |
| 8–99, 136–224; declaration launch_success_drives_to_completed; declaration launch_muse_harness_acquires_muse_provider; declaration launch_duplicate_returns_recorded_state; declaration launch_acquire_refusal_matrix; declaration launch_abandon_paths_record_refusals | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | launch_success_drives_to_completed: implement the current factory assignment and dispatch duty in launch.rs.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 100–135; declaration launch_prechecks_reject_before_any_seam | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | launch_prechecks_reject_before_any_seam: implement the current candidate verification assessment duty in launch.rs. — current source lib/host/src/factory/tests/launch.rs; lines 100-135; module/caller wiring inspected |

<a id="coverage-96fb6a5e1ba7"></a>

## [lib/host/src/factory/tests/mocks.rs](../../../../../lib/host/src/factory/tests/mocks.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–435; current module/import/attribute shell; declaration ExecCall; declaration FakeExec; fields calls, script; declaration new; declaration run; declaration TakeoverCall; declaration ExportCall; declaration FakeTerminal; fields version, sha256, reserve, start, wait, stop, stop_unbound, capture, live, output, takeover, export, reserve_calls, takeover_calls, export_calls, output_calls; declaration pop; declaration harness_family; declaration harness_version; declaration harness_sha256; declaration reserve; declaration start; declaration wait; declaration stop; declaration stop_unbound; declaration capture; declaration live; declaration output; declaration takeover_copy; declaration export_bundle; declaration FakeBroker; fields acquire, register, returns, terminal, close, acquire_calls, reconcile_calls, close_calls; declaration acquire; declaration register; declaration return_lease; declaration reconcile_lease; declaration execution_is_terminal; declaration close_execution; declaration script_success | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/mocks.rs into its current native target.; 53 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8f32bab0f4db"></a>

## [lib/host/src/factory/tests/mod.rs](../../../../../lib/host/src/factory/tests/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1, 4–6, 8–11; declaration artifacts; declaration common; declaration confirmation; declaration finish; declaration mocks; declaration receipt; declaration stop; declaration wire | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | artifacts: implement the current factory run lifecycle and intervention duty in mod.rs.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 2–3; declaration candidate | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | candidate: implement the current candidate verification assessment duty in mod.rs. — current source lib/host/src/factory/tests/mod.rs; lines 2-3; module/caller wiring inspected |
| 7; declaration launch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | launch: implement the current factory assignment and dispatch duty in mod.rs. — current source lib/host/src/factory/tests/mod.rs; lines 7-7; module/caller wiring inspected |

<a id="coverage-f9d429ff0e48"></a>

## [lib/host/src/factory/tests/receipt.rs](../../../../../lib/host/src/factory/tests/receipt.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–238; current module/import/attribute shell; declaration signed_factory_numbers_preserve_negative_zero_and_nullable_exit_code; declaration receipt_bytes_match_go_marshal; declaration receipt_decode_matrix | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/receipt.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b96def31f310"></a>

## [lib/host/src/factory/tests/stop.rs](../../../../../lib/host/src/factory/tests/stop.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; current module/import/attribute shell; declaration stop_req; declaration stop_before_start_writes_exact_tombstone; declaration stop_before_start_close_failure_retries; declaration stop_approved_run_uses_unbound_stop; declaration launch_path_respects_stop_owned_uncertain; declaration write_approved_with_lease; declaration stop_before_delivery_confirms_despite_native_failure; declaration stop_close_retries_before_uncertain; declaration stop_running_run_reconciles_custody; declaration stop_converges_when_launch_returned_first; declaration stop_uncertain_when_custody_unsettled; declaration stop_terminal_receipt_is_final | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/stop.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-85ce88d502c4"></a>

## [lib/host/src/factory/tests/wire.rs](../../../../../lib/host/src/factory/tests/wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–320; current module/import/attribute shell; declaration validators_match_go; declaration run_and_launch_validation_pins_every_error; declaration deadline_parser_uses_strict_shared_wire_shape; declaration error_messages_are_exact; declaration open_factory_matrix; declaration harness_pin_shape_matches_go | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/factory/tests/wire.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-84ecb8a320b0"></a>

## [lib/host/src/terminal/factory/artifacts.rs](../../../../../lib/host/src/terminal/factory/artifacts.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 15–277; current module/import/attribute shell; declaration MAX_FACTORY_EXPORT_BUNDLE; declaration TAKEOVER_DIR_NAME; declaration ERR_FACTORY_EXPORT_BOUNDS; declaration takeover_destination; declaration takeover_source; declaration FACTORY_EXPORT_SCRIPT; declaration takeover_steps; declaration export_argv; declaration factory_export_bundle; declaration factory_takeover_copy | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/artifacts.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 13–14; declaration ERR_FACTORY_EXPORT_CANDIDATE | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | ERR_FACTORY_EXPORT_CANDIDATE: implement the current candidate verification assessment duty in artifacts.rs. — current source lib/host/src/terminal/factory/artifacts.rs; lines 13-14; module/caller wiring inspected |

<a id="coverage-7b6ea0b2de37"></a>

## [lib/host/src/terminal/factory/binding.rs](../../../../../lib/host/src/terminal/factory/binding.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 53–82; current module/import/attribute shell; declaration RunPathsFn; declaration factory_identity_operation | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/binding.rs into its current native target.; RunPathsFn: implement the current factory run lifecycle and intervention duty in binding.rs.; factory_identity_operation: implement the current factory run lifecycle and intervention duty in binding.rs. — Current named units/source consumers; retained normalized source evidence records each selector |
| 18–52; declaration checked_binding_paths | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | checked_binding_paths: implement the current candidate verification assessment duty in binding.rs. — current source lib/host/src/terminal/factory/binding.rs; lines 18-52; module/caller wiring inspected |

<a id="coverage-c6143b5c508d"></a>

## [lib/host/src/terminal/factory/codex/tests.rs](../../../../../lib/host/src/terminal/factory/codex/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–610; current module/import/attribute shell; declaration PID; declaration RID; declaration IID; declaration CID; declaration PREP; declaration ROLE; declaration COMMIT; declaration PIN; declaration TMP_COUNTER; declaration deadline; declaration test_tmp; declaration RecordedCall; declaration FakeExec; fields calls, script; declaration new; declaration calls; declaration run; declaration ok; declaration err; declaration make_service; declaration inspect_json; declaration factory_run; declaration run_dir; declaration factory_lease; declaration write_harness; declaration host_digest; declaration domain_predicates; declaration run_validate_pins; declaration path_vectors; declaration quote_vectors; declaration GOLDEN_SUPERVISOR; declaration GOLDEN_RETIRE; declaration script_goldens; declaration binding_matrix; declaration unit_show_vectors; declaration euid; declaration reserve_harness; declaration reserve_denial_pins | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/codex/tests.rs into its current native target.; 38 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cf7870b517c2"></a>

## [lib/host/src/terminal/factory/lifecycle.rs](../../../../../lib/host/src/terminal/factory/lifecycle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–302; current module/import/attribute shell; declaration factory_retire; declaration factory_await_inactive; declaration factory_read_pid; declaration factory_container_exists; declaration factory_wait_result; declaration factory_attest_live; declaration factory_stop_confirmed; declaration factory_stop_unbound_confirmed; declaration factory_capture_valid; declaration factory_live_scoped | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/lifecycle.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-172cd0039f63"></a>

## [lib/host/src/terminal/factory/native.rs](../../../../../lib/host/src/terminal/factory/native.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–273; current module/import/attribute shell; declaration factory_unit_name; declaration factory_unit_name_or_denied; declaration systemd_escape; declaration shell_quote; declaration factory_user_bus; declaration current_euid; declaration factory_systemctl_argv; declaration factory_systemd_run_argv; declaration reserve_run_argv; declaration harness_install_script; declaration stage_file_command; declaration FactoryUnitShow; fields active, invocation; declaration parse_factory_unit_show; declaration factory_role_id; declaration sleep_until; declaration run_env; declaration run_podman; declaration factory_systemctl; declaration factory_systemd_run; declaration factory_unit_state; declaration factory_active_invocation; declaration factory_role_ids; declaration factory_stage_file | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/native.rs into its current native target.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ff92ce8249d5"></a>

## [lib/host/src/terminal/factory/output.rs](../../../../../lib/host/src/terminal/factory/output.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30, 41–178; current module/import/attribute shell; declaration MAX_FACTORY_OUTPUT_READ; declaration MAX_FACTORY_OUTPUT_WINDOW; declaration MAX_FACTORY_OUTPUT_OFFSET; declaration FactoryCodexOutputSlice; fields data, total, offset, truncated, gap; declaration FactoryOutputSlice; declaration output_read_command; declaration factory_output_size; declaration factory_output_stdout; declaration factory_output_window | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/output.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 31–40; declaration check_output_range | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | check_output_range: implement the current candidate verification assessment duty in output.rs. — current source lib/host/src/terminal/factory/output.rs; lines 31-40; module/caller wiring inspected |

<a id="coverage-11664b5943f3"></a>

## [lib/host/src/terminal/factory/run.rs](../../../../../lib/host/src/terminal/factory/run.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–266; current module/import/attribute shell; declaration FACTORY_SCOPE_CODEX; declaration FACTORY_HARNESS_CODEX; declaration FACTORY_SCOPE_MUSE; declaration FACTORY_HARNESS_MUSE; declaration valid_harness_family; declaration ROLE_CODER; declaration ROLE_REVIEWER; declaration MAX_FACTORY_PROMPT; declaration valid_factory_role; declaration valid_preparation_id; declaration valid_digest; declaration valid_commit; declaration valid_factory_run_id; declaration valid_harness_version; declaration FactoryRun; fields deadline_raw, actor, id, project, role, preparation, harness, harness_vers, model, assignment, source_commit, connection; declaration FactoryRunWire; fields deadline, actor, id, project, role, preparation, harness, harness_version, model, assignment, source_commit, connection, deadline_seen; declaration deserialize; declaration V; declaration Value; declaration expecting; declaration visit_map; declaration validate; declaration decode | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/run.rs into its current native target.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-93157a0eb7c1"></a>

## [lib/host/src/terminal/factory/tests/artifacts.rs](../../../../../lib/host/src/terminal/factory/tests/artifacts.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–197; current module/import/attribute shell; declaration export_flows; declaration takeover_flows | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/tests/artifacts.rs into its current native target.; export_flows: implement the current factory run lifecycle and intervention duty in artifacts.rs.; takeover_flows: implement the current factory run lifecycle and intervention duty in artifacts.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a5ee0fc58dff"></a>

## [lib/host/src/terminal/factory/tests/mod.rs](../../../../../lib/host/src/terminal/factory/tests/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4; declaration artifacts; declaration run | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | artifacts: implement the current factory run lifecycle and intervention duty in mod.rs.; run: implement the current factory run lifecycle and intervention duty in mod.rs. — current source lib/host/src/terminal/factory/tests/mod.rs; lines 1-2; module/caller wiring inspected; current source lib/host/src/terminal/factory/tests/mod.rs; lines 3-4; module/caller wiring inspected |

<a id="coverage-c25786ca9afa"></a>

## [lib/host/src/terminal/factory/tests/run.rs](../../../../../lib/host/src/terminal/factory/tests/run.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–836; current module/import/attribute shell; declaration reserve_success_argv_sequence; declaration reserve_stage_and_failure_paths; declaration reserve_unattested_unit_is_stopped; declaration start_flows; declaration wait_flows; declaration validate_flows; declaration stop_flows; declaration capture_and_finish_flows; declaration stop_unbound_flows; declaration live_matrix; declaration output_flows; declaration factory_identity_operation_matrix | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/tests/run.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-ec8078a0d212"></a>

Former source `rust/soda-host/src/pfactory.rs`; consult its pinned earlier Git source and the current coverage disposition.

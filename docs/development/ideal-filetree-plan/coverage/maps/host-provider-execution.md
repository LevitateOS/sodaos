# Host provider execution

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-d3a966f6d3cc"></a>
<a id="coverage-08d3fedc31ef"></a>

## [lib/host/src/terminal/factory/mod.rs](../../../../../lib/host/src/terminal/factory/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; declaration artifacts; declaration binding; declaration codex; declaration lifecycle; declaration muse; declaration native; declaration output; declaration run; declaration tcodex; declaration tests | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | artifacts: implement the current factory run lifecycle and intervention duty in mod.rs.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-60a9454258b1"></a>
<a id="coverage-125be2eb23c3"></a>

## [lib/host/src/terminal/factory/tcodex.rs](../../../../../lib/host/src/terminal/factory/tcodex.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–602; current module/import/attribute shell; declaration factory_run_paths; declaration factory_codex_guest; declaration FactoryCodexPaths; fields checkout, run_dir, home, codex, prompt, marker, started, stop, pid_file, output, stdout, auth, guest; declaration factory_codex_paths; declaration factory_supervisor; declaration factory_codex_binding; declaration reserve_exec_argv; declaration codex_setup_script; declaration start_gate_script; declaration factory_codex_reserve; declaration factory_codex_setup; declaration factory_codex_stage; declaration factory_codex_stage_host; declaration HOST_GUEST; declaration factory_codex_start; declaration factory_codex_wait; declaration factory_codex_validate; declaration factory_codex_stop; declaration factory_codex_capture; declaration factory_codex_finish; declaration factory_codex_stop_unbound; declaration factory_codex_live; declaration factory_codex_output | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Imports and module declarations wire lib/host/src/terminal/factory/tcodex.rs into its current native target.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3e7d28103e6d"></a>

## [lib/host/tests/tcodex_ops_oracle.rs](../../../../../lib/host/tests/tcodex_ops_oracle.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–659; current module/import/attribute shell; declaration PID; declaration RID; declaration IID; declaration CID; declaration PREP; declaration ROLE; declaration UNIT; declaration RUN_DIR; declaration deadline; declaration ok; declaration err; declaration inspect_json; declaration factory_lease; declaration make_service; declaration RecordedCall; declaration FakeExec; fields calls, script; declaration new; declaration calls; declaration run; declaration assert_env_systemctl; declaration assert_podman; declaration fixed_paths_match_go_layout; declaration validate_ok_argv_golden; declaration validate_denies_each_gate; declaration stop_ok_argv_golden; declaration stop_is_idempotent_without_container; declaration stop_uncertain_matrix; declaration finish_ok_returns_credential_bytes; declaration finish_runs_capture_only_after_confirmed_stop; declaration unit_show_parse_oracle; declaration adapter_dispatch_calls_each_callback | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/tcodex_ops_oracle.rs into its current native target.; 32 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-ec73e0d18b70"></a>

Former source `lib/host/src/terminal/factory/tfactory.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-15bd15f9cf85"></a>

Former source `lib/host/src/terminal/factory/tmuse.rs`; consult its pinned earlier Git source and the current coverage disposition.

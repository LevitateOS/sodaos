# Soda muse

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-5e4c061fa57f"></a>

## [cmd/soda-muse/src/account.rs](../../../../../cmd/soda-muse/src/account.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–118; current module/import/attribute shell; declaration account_for; declaration AccountRecord; fields uid, gid, username, name, home_dir; declaration account_entry; declaration account_node; declaration lookup_user | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/account.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a7f66aca43e7"></a>

## [cmd/soda-muse/src/config.rs](../../../../../cmd/soda-muse/src/config.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–235; current module/import/attribute shell; declaration copy_config; declaration copy_config_file; declaration copy_config_file_paths; declaration read_limited; declaration tests; declaration walker_keeps_native_names_and_leaf_link_policy; declaration walker_retains_auth_directory_descent_failure; declaration config_destination_symlink_is_rejected; declaration bounded_read_uses_open_inode_and_detects_growth; declaration read_config | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/config.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-58d38bf8e80f"></a>

## [cmd/soda-muse/src/config_tests.rs](../../../../../cmd/soda-muse/src/config_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; current module/import/attribute shell; declaration copy_config_skips_credentials_and_copies_tree; declaration read_config_contract_shapes; declaration account_markers_reject_unsafe_nodes; declaration native_dispatch_errors_match_go | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/config_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-78e7f99aa257"></a>

## [cmd/soda-muse/src/execution.rs](../../../../../cmd/soda-muse/src/execution.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147; current module/import/attribute shell; declaration execute; declaration await_admission; declaration execution_state; declaration mkdir_mode; declaration muse_environment; declaration env_lossy_opt; declaration muse_environment_with | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/execution.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-510066f88d4a"></a>

## [cmd/soda-muse/src/launch.rs](../../../../../cmd/soda-muse/src/launch.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–174; current module/import/attribute shell; declaration seqpacket_connect; declaration send_with_fds; declaration launch_shell; declaration Guard; declaration drop; declaration NEVER; declaration forward_signals; declaration controls_loop | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/launch.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-872e0d1da647"></a>

## [cmd/soda-muse/src/launch_json.rs](../../../../../cmd/soda-muse/src/launch_json.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–40; current module/import/attribute shell; declaration GoFormatter; declaration write_string_fragment; declaration serialize_go | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/launch_json.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5ce2f02d19aa"></a>

## [cmd/soda-muse/src/launch_tests.rs](../../../../../cmd/soda-muse/src/launch_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–148; current module/import/attribute shell; declaration launch_shell_passes_fds_and_maps_exit; declaration Guard; declaration drop; declaration controls_forward_signal_number | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/launch_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-663d6a1eb1c4"></a>

## [cmd/soda-muse/src/launch_wire.rs](../../../../../cmd/soda-muse/src/launch_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–95; current module/import/attribute shell; declaration ShellLaunchRequest; fields home, config_home, term, connection_id, cwd, args, tty, cols, rows; declaration shell_request_json; declaration base64_encode; declaration LaunchExit; fields code, error; declaration deserialize; declaration ExitVisitor; declaration Value; declaration expecting; declaration visit_map; declaration parse_launch_exit | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/launch_wire.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3cff7d6b1f2d"></a>

## [cmd/soda-muse/src/main.rs](../../../../../cmd/soda-muse/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; current module/import/attribute shell; declaration account; declaration config; declaration config_tests; declaration execution; declaration launch; declaration launch_json; declaration launch_tests; declaration launch_wire; declaration paths; declaration runtime; declaration shell; declaration shell_tests; declaration MUSE_LAUNCH_SOCKET; declaration MUSE_NATIVE; declaration main; declaration run | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/main.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 56–92; declaration native_action; declaration metadata_action | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | native_action: implement the current identity grants and connection availability duty in main.rs.; metadata_action: implement the current identity grants and connection availability duty in main.rs. — current source cmd/soda-muse/src/main.rs; lines 56-77; module/caller wiring inspected; current source cmd/soda-muse/src/main.rs; lines 78-92; module/caller wiring inspected |

<a id="coverage-6dfa6a67e80d"></a>

## [cmd/soda-muse/src/paths.rs](../../../../../cmd/soda-muse/src/paths.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; current module/import/attribute shell; declaration go_base; declaration go_join; declaration is_clean_absolute_path; declaration errno_str; declaration go_strerror; declaration path_error; declaration go_quote_rune | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/paths.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-932bc9546e4f"></a>

## [cmd/soda-muse/src/runtime.rs](../../../../../cmd/soda-muse/src/runtime.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; current module/import/attribute shell; declaration check_runtime; declaration Cleanup; declaration drop; declaration make_private_dir; declaration run_pristine | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/runtime.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0d981a1c951b"></a>

## [cmd/soda-muse/src/shell.rs](../../../../../cmd/soda-muse/src/shell.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–110; current module/import/attribute shell; declaration ShellRequest; fields cwd, args, home, connection_id, config_home, term, tty, cols, rows; declaration shell_request; declaration env_lossy; declaration validate_shell; declaration config_paths_valid; declaration launch_sizes_valid; declaration launch_absolute_path; declaration launch_text; declaration launch_arguments_valid | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/shell.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5345bcb7bc2f"></a>

## [cmd/soda-muse/src/shell_tests.rs](../../../../../cmd/soda-muse/src/shell_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–210; current module/import/attribute shell; declaration shell_fixture; declaration shell_request_wire_matches_go; declaration shell_request_keeps_go_html_safe_escapes; declaration shell_validation_matches_go; declaration launch_exit_parsing_matches_go_unmarshal; declaration launch_exit_rejects_malformed_unicode_boundary; declaration base64_matches_standard_vectors; declaration native_path_join_and_root_admission; declaration go_quote_matches_invalid_byte_cases; declaration execution_id_rules_match_go; declaration muse_environment_orders_fixed_then_passthrough | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-muse/src/shell_tests.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-4288d87834c0"></a>
<a id="rustsoda-musesrcmainrs-1"></a>

Former source `rust/soda-muse/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

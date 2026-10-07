# Soda project terminal

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-7fae7f2262df"></a>

## [cmd/soda-project-terminal/Cargo.toml](../../../../../cmd/soda-project-terminal/Cargo.toml)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8; package metadata lines 1-8 | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Package identity and crate-level purpose for the Project terminal command package. — current Cargo package metadata; package source modules in cmd/soda-project-terminal/src |
| 9–20; [[bin]] project-terminal lines 9-12; [[bin]] project-account lines 13-16; [[bin]] project-factory-roles lines 17-20 | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Register the project-terminal binary target and its source path.; Register the project-account binary target and its source path.; Register the project-factory-roles binary target and its source path. — Current named units/source consumers; retained normalized source evidence records each selector |
| 21–25, 27; serde, serde_json, base64, sha2 dependency declarations lines 21-25; soda-wire-time dependency line 27 | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Wire encoding, JSON representation and hashing dependency edges for Project command protocols.; Shared wire-time protocol dependency used by Project command interfaces. — current command source imports and uses these crates for JSON/wire representations, base64 and digest duties; cmd/soda-project-terminal source imports soda_wire_time protocol types |
| 26, 28; libc dependency line 26; rustix dependency line 28 | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Native libc interface dependency for Project command process and filesystem primitives.; Rust native filesystem, networking and event primitives dependency. — current Project terminal source calls libc process/native interfaces; current Project terminal source imports rustix for filesystem and event operations |

<a id="coverage-72a45b5a5c78"></a>

## [cmd/soda-project-terminal/src/b64.rs](../../../../../cmd/soda-project-terminal/src/b64.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–52; current module/import/attribute shell; declaration python_validate; declaration decode; declaration encode; declaration tests; declaration vectors | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/b64.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7d6131a1c268"></a>

## [cmd/soda-project-terminal/src/bin/project-factory-roles.rs](../../../../../cmd/soda-project-terminal/src/bin/project-factory-roles.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; current module/import/attribute shell; declaration pyemit; declaration state_json; declaration factory_roles; declaration main | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/bin/project-factory-roles.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c32042d9cd17"></a>

## [cmd/soda-project-terminal/src/broker.rs](../../../../../cmd/soda-project-terminal/src/broker.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39, 167–169; current module/import/attribute shell; declaration broker_tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/broker.rs into its current native target.; Exercise the named native behavior in broker.rs. — current source cmd/soda-project-terminal/src/broker.rs; Cargo target and callers; current source cmd/soda-project-terminal/src/broker.rs; lines 167-169; module/caller wiring inspected |
| 40–98; declaration REQUEST_LIMIT; declaration CREDENTIAL_LIMIT; declaration HORIZON_SECS; declaration BROKER_ALARM_SECS; declaration subscription_dispatch | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | REQUEST_LIMIT: implement the current provider execution integration duty in broker.rs.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 99–166; declaration BrokerFail; declaration read_stdin; declaration write_stdout_all; declaration write_stderr_all; declaration broker_inner; declaration broker_main | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | BrokerFail: implement the current Unix-socket service and process lifetime duty in broker.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5017c5a98ed3"></a>

## [cmd/soda-project-terminal/src/broker_tests.rs](../../../../../cmd/soda-project-terminal/src/broker_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–229; current module/import/attribute shell; declaration parse; declaration equality_matrix; declaration int_conversion_matrix; declaration lease_binding_edit; declaration deadline_window; declaration request_decode_matrix; declaration result_shapes_exact; declaration argv_constructors_exact; declaration cgroup_events_matrix; declaration mount_probe_matrix; declaration TEST_SEQ; declaration remove_model_matrix; declaration dispatch_rejects_garbage_without_touching_fs | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/broker_tests.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-24a209e4ab4c"></a>

## [cmd/soda-project-terminal/src/cgroup.rs](../../../../../cmd/soda-project-terminal/src/cgroup.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–273; current module/import/attribute shell; declaration stat_fs_argv; declaration cstring; declaration wait_status_timeout; declaration exit_ok; declaration run_stat_fs; declaration fstatvfs_readonly; declaration cgroup_parent; declaration parse_cgroup_populated; declaration cgroup_empty | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/cgroup.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2b40e9853695"></a>

## [cmd/soda-project-terminal/src/fs.rs](../../../../../cmd/soda-project-terminal/src/fs.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 82–107, 258–260; current module/import/attribute shell; declaration read_record_text; declaration read_record_text_at; declaration read_record_value; declaration fs_tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/fs.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 14–81, 108–257; declaration cstr; declaration check_component; declaration cvt; declaration s_isreg; declaration s_islnk; declaration stat_is_safe; declaration root_file; declaration read_record; declaration new_file; declaration mkdir_at; declaration fchmod; declaration unlink_at; declaration rmdir_at; declaration chown_path; declaration chmod_path; declaration replace_at; declaration fsync_file; declaration fstat_uid_mode; declaration fstat_all; declaration read_up_to | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | cstr: implement the current configuration and safe filesystem primitives duty in fs.rs.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ab6afd7d8c36"></a>

## [cmd/soda-project-terminal/src/fs_tests.rs](../../../../../cmd/soda-project-terminal/src/fs_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–246; current module/import/attribute shell; declaration TEST_SEQ; declaration test_dir; declaration dir_fd; declaration am_root; declaration fake_stat; declaration safety_predicate_matrix; declaration root_file_failures; declaration record_size_and_json_limits; declaration read_record_public_paths; declaration new_file_create_and_collision; declaration wrapper_matrix; declaration path_chown_chmod_matrix; declaration stat_all_and_single_read | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/fs_tests.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-48a56c3785a9"></a>

## [cmd/soda-project-terminal/src/key_lines.rs](../../../../../cmd/soda-project-terminal/src/key_lines.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 9–29, 54–79; current module/import/attribute shell; declaration STDIN_LIMIT; declaration is_word; declaration word_dash_plus; declaration sk_plus; declaration canonical_lines | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/key_lines.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 5–8, 30–53; declaration KEY_FILE_LIMIT; declaration KEY_COUNT_LIMIT; declaration canonical_key_ok | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | KEY_FILE_LIMIT: implement the current Project SSH key access duty in key_lines.rs.; KEY_COUNT_LIMIT: implement the current Project SSH key access duty in key_lines.rs.; canonical_key_ok: implement the current Project SSH key access duty in key_lines.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f45ce2f53eb4"></a>

## [cmd/soda-project-terminal/src/key_request.rs](../../../../../cmd/soda-project-terminal/src/key_request.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–116, 172–183; current module/import/attribute shell; declaration RevisionValue; declaration from_raw; declaration as_string; declaration is_truthy; declaration check_unique_raw; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map; declaration ArrayVisitor; declaration visit_seq; declaration state_object | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/key_request.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 117–171; declaration KeyRequest; fields login, identity, apply, revision, keys, desired; declaration KeyRequestWire; fields login, identity, apply, revision, keys; declaration decode_key_request | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | KeyRequest: implement the current Project SSH key access duty in key_request.rs.; KeyRequestWire: implement the current Project SSH key access duty in key_request.rs.; decode_key_request: implement the current Project SSH key access duty in key_request.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-98edb4aae566"></a>

## [cmd/soda-project-terminal/src/keys.rs](../../../../../cmd/soda-project-terminal/src/keys.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–34; current module/import/attribute shell | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/keys.rs into its current native target. — current source cmd/soda-project-terminal/src/keys.rs; Cargo target and callers |
| 35–240; declaration keys_directory; declaration FileId; fields dev, ino, mtime_sec, mtime_nsec; declaration file_id; declaration read_keys; declaration candidate_name; declaration update; declaration replace_inner; declaration KeyFail; declaration read_stdin; declaration write_stdout_all; declaration write_stderr_all; declaration key_main; declaration key_inner; declaration keys_tests | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | keys_directory: implement the current Project SSH key access duty in keys.rs.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-40cf94c4eb0d"></a>

## [cmd/soda-project-terminal/src/keys_tests.rs](../../../../../cmd/soda-project-terminal/src/keys_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 50–200; current module/import/attribute shell; declaration canonical_lines_matrix; declaration truthiness_matrix; declaration request; declaration decode_matrix; declaration candidate_matrix; declaration state_shape_exact; declaration update_refuses_unprivileged | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/keys_tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 5–49; declaration KEY_A; declaration KEY_B; declaration KEY_SK; declaration key_line_matrix | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | KEY_A: implement the current Project SSH key access duty in keys_tests.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-79eceb11e8e7"></a>
<a id="coverage-8b71c1af882c"></a>

## [cmd/soda-project-terminal/src/main.rs](../../../../../cmd/soda-project-terminal/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 21, 27–30, 33–41, 45–51, 53–55, 81–102; current module/import/attribute shell; declaration cgroup; declaration pty; declaration pty_io; declaration pty_process; declaration pty_relay; declaration socket; declaration state_json; declaration subscription_cgroup; declaration subscription_credentials; declaration subscription_prepare; declaration subscription_profile; declaration subscription_retire; declaration subscription_start; declaration subscription_wire; declaration term_attach; declaration term_binding; declaration term_collect; declaration term_create; declaration term_paths; declaration term_prepare; declaration term_status; declaration tmux; declaration write_stdout_all; declaration is_epipe; declaration emit_closed | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/main.rs into its current native target.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 18, 26, 42, 44, 56–80, 103–222; declaration account; declaration proto; declaration svc; declaration term; declaration Route; declaration route; declaration control_main; declaration run; declaration main; declaration tests; declaration argv; declaration control10; declaration route_matrix | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | account: implement the current human terminal lifecycle duty in main.rs.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 19, 31–32, 52; declaration b64; declaration pyemit; declaration sha; declaration timex | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | b64: implement the current wire decoding, encoding, and representation conversion duty in main.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20; declaration broker | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | broker: implement the current provider execution integration duty in main.rs. — current source cmd/soda-project-terminal/src/main.rs; lines 20-20; module/caller wiring inspected |
| 22, 43; declaration fs; declaration sys | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | fs: implement the current configuration and safe filesystem primitives duty in main.rs.; sys: implement the current configuration and safe filesystem primitives duty in main.rs. — current source cmd/soda-project-terminal/src/main.rs; lines 22-22; module/caller wiring inspected; current source cmd/soda-project-terminal/src/main.rs; lines 43-43; module/caller wiring inspected |
| 23–25; declaration key_lines; declaration key_request; declaration keys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | key_lines: implement the current Project SSH key access duty in main.rs.; key_request: implement the current Project SSH key access duty in main.rs.; keys: implement the current Project SSH key access duty in main.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f61ccc84403d"></a>
<a id="coverage-512c080899f7"></a>

## [cmd/soda-project-terminal/src/proto.rs](../../../../../cmd/soda-project-terminal/src/proto.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–125, 259–304; current module/import/attribute shell; declaration FRAME_LIMIT; declaration QUEUE_LIMIT; declaration HEARTBEAT_SECONDS; declaration dimensions; declaration ControlFrame; declaration FrameFields; declaration deserialize; declaration FieldsVisitor; declaration Value; declaration expecting; declaration visit_map; declaration has_fields; declaration field; declaration decode_frame; declaration is_cc_or_cf; declaration frames | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/proto.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 126–258, 305–352; declaration valid_name; declaration ControlArgs; fields action, identifier, login, identity, cols, rows, seconds, source_hash, name, scope; declaration parse_control_argv; declaration valid_identifier; declaration require_terminal_target; declaration valid_scope; declaration require_creation_scope; declaration tests; declaration argv; declaration names; declaration argv_matrix | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | valid_name: implement the current human terminal lifecycle duty in proto.rs.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-940c84b6b0ad"></a>

## [cmd/soda-project-terminal/src/pty.rs](../../../../../cmd/soda-project-terminal/src/pty.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–196; current module/import/attribute shell; declaration STOPPED; declaration stop_handler; declaration install_stop_handlers; declaration restore_handlers; declaration ready_line; declaration output_line; declaration closed_line; declaration flush_closed; declaration run_terminal; declaration pty_tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/pty.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6d82ccf1c0ee"></a>

## [cmd/soda-project-terminal/src/pty_io.rs](../../../../../cmd/soda-project-terminal/src/pty_io.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–174; current module/import/attribute shell; declaration set_nonblocking; declaration is_eintr; declaration read_fd; declaration write_fd; declaration write_all_blocking; declaration pty_select; declaration wait_readable | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/pty_io.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-815ea692d15e"></a>

## [cmd/soda-project-terminal/src/pty_process.rs](../../../../../cmd/soda-project-terminal/src/pty_process.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–200; current module/import/attribute shell; declaration tmux_attach_argv; declaration cstring; declaration env_entries; declaration set_size; declaration child_exited; declaration end_child; declaration spawn_login_pty | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/pty_process.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-32c4a1a1cc86"></a>

## [cmd/soda-project-terminal/src/pty_relay.rs](../../../../../cmd/soda-project-terminal/src/pty_relay.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–220; current module/import/attribute shell; declaration apply_control_frame; declaration ingest_control_bytes; declaration read_pty_output; declaration session_should_stop; declaration take_control_input; declaration flush_pty_queues; declaration relay_pty_session | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/pty_relay.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3e286e66d0d5"></a>

## [cmd/soda-project-terminal/src/pty_tests.rs](../../../../../cmd/soda-project-terminal/src/pty_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–298; current module/import/attribute shell; declaration login; declaration line_bytes_exact; declaration ingest_matrix; declaration queue_backpressure; declaration stop_matrix; declaration child_lifecycle; declaration end_child_terms; declaration pty_output_lines; declaration resize_needs_tty; declaration bounds_rejected; declaration env_entries_are_nul_terminated; declaration select_reports_writable_fds | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/pty_tests.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-de61328eca27"></a>

## [cmd/soda-project-terminal/src/pyemit.rs](../../../../../cmd/soda-project-terminal/src/pyemit.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–189, 204–241; current module/import/attribute shell; declaration dumps; declaration line; declaration dumps_default; declaration as_int; declaration tests; declaration obj; declaration emit_matches_cpython; declaration emit_default_separators; declaration emit_short_escapes; declaration ordinary_dictionary_duplicates_keep_last_value_and_first_position; declaration int_shapes; declaration dumps_serde; declaration dumps_default_serde; declaration line_serde; declaration dumps_with_profile; declaration PythonFormatter; fields spaced; declaration write_string_fragment; declaration begin_array_value; declaration begin_object_value; declaration serde_output_tests; declaration Reply; fields value, items; declaration python_ascii_and_separator_profiles | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/pyemit.rs into its current native target.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 190–203; declaration begin_object_key | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | begin_object_key: implement the current Project SSH key access duty in pyemit.rs. — current source cmd/soda-project-terminal/src/pyemit.rs; lines 190-203; module/caller wiring inspected |

<a id="coverage-70744bbef77a"></a>

## [cmd/soda-project-terminal/src/sha.rs](../../../../../cmd/soda-project-terminal/src/sha.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; current module/import/attribute shell; declaration digest; declaration hex; declaration HEX; declaration hex_digest; declaration tests; declaration nist_vectors_and_streaming_partitions | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/sha.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d139d7c24567"></a>

## [cmd/soda-project-terminal/src/socket.rs](../../../../../cmd/soda-project-terminal/src/socket.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–111; current module/import/attribute shell; declaration s_issock; declaration SocketCheck; declaration socket_identity_kinded; declaration classify_connect_err; declaration socket_identity | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/socket.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-85487b942a1d"></a>

## [cmd/soda-project-terminal/src/state_json.rs](../../../../../cmd/soda-project-terminal/src/state_json.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–240; current module/import/attribute shell; declaration StateValue; declaration parse; declaration from_raw; declaration ObjectVisitor; fields depth; declaration Value; declaration expecting; declaration visit_map; declaration ArrayVisitor; fields depth; declaration visit_seq; declaration collapse_dictionaries; declaration get; declaration object; declaration as_str; declaration as_bool; declaration as_integer; declaration serialize; declaration tests; declaration python_dictionary_order_and_raw_numbers_survive_decode_and_emit; declaration dynamic_tree_matches_serde_container_depth_limit | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/state_json.rs into its current native target.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c926569168a6"></a>

## [cmd/soda-project-terminal/src/subscription_cgroup.rs](../../../../../cmd/soda-project-terminal/src/subscription_cgroup.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; current module/import/attribute shell; declaration subscription_cgroup; declaration subscription_kernel_write; declaration parse_cgroup_events; declaration subscription_freeze | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_cgroup.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9bae8cada29e"></a>

## [cmd/soda-project-terminal/src/subscription_credentials.rs](../../../../../cmd/soda-project-terminal/src/subscription_credentials.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–119; current module/import/attribute shell; declaration fchownat_no_follow; declaration is_regular; declaration subscription_auth_directory; declaration subscription_seed; declaration subscription_stage; declaration subscription_capture | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_credentials.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a4c715ba902b"></a>

## [cmd/soda-project-terminal/src/subscription_prepare.rs](../../../../../cmd/soda-project-terminal/src/subscription_prepare.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–153; current module/import/attribute shell; declaration mount_argv; declaration subscription_prepare; declaration subscription_provision | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_prepare.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9c4e154db657"></a>

## [cmd/soda-project-terminal/src/subscription_profile.rs](../../../../../cmd/soda-project-terminal/src/subscription_profile.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–236; current module/import/attribute shell; declaration subscription_path; declaration path_missing; declaration lease_execution_id; declaration lease_actor_id; declaration profile_deadline; declaration live_deadline; declaration ProfileHit; fields profile, account; declaration subscription_profile; declaration subscription_invocation; declaration subscription_check_unit; declaration subscription_lookup; declaration subscription_resolve | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_profile.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ec0475f429fc"></a>

## [cmd/soda-project-terminal/src/subscription_retire.rs](../../../../../cmd/soda-project-terminal/src/subscription_retire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; current module/import/attribute shell; declaration unlink_missing_ok; declaration rmdir_missing_ok; declaration subscription_remove_model; declaration is_mount; declaration subscription_retire; declaration subscription_finish | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_retire.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1d3e047b8fb8"></a>

## [cmd/soda-project-terminal/src/subscription_start.rs](../../../../../cmd/soda-project-terminal/src/subscription_start.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101; current module/import/attribute shell; declaration subscription_harness_digest; declaration respawn_argv; declaration subscription_start | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_start.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b39592e6ed98"></a>

## [cmd/soda-project-terminal/src/subscription_wire.rs](../../../../../cmd/soda-project-terminal/src/subscription_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–186; current module/import/attribute shell; declaration last; declaration number_f64; declaration number_equal; declaration json_equal; declaration json_int; declaration LO; declaration HI; declaration lease_with_binding; declaration deadline_ok; declaration result_object; declaration empty_result; declaration decode_request; declaration native_binding; declaration profile_object | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/subscription_wire.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f5e782728857"></a>

## [cmd/soda-project-terminal/src/svc.rs](../../../../../cmd/soda-project-terminal/src/svc.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 168–170; current module/import/attribute shell; declaration svc_tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/svc.rs into its current native target.; Exercise the named native behavior in svc.rs. — current source cmd/soda-project-terminal/src/svc.rs; Cargo target and callers; current source cmd/soda-project-terminal/src/svc.rs; lines 168-170; module/caller wiring inspected |
| 14–167; declaration unit_name; declaration service_properties; declaration systemctl_show_argv; declaration systemctl_stop_argv; declaration invocation_show_argv; declaration parse_service_fields; declaration verify_loaded_unit; declaration service_state; declaration stop_service | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | unit_name: implement the current human terminal lifecycle duty in svc.rs.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-abc4f964290d"></a>

## [cmd/soda-project-terminal/src/svc_tests.rs](../../../../../cmd/soda-project-terminal/src/svc_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–218; current module/import/attribute shell; declaration sample; declaration show_fixture; declaration argv_constructors_exact; declaration service_fields_matrix; declaration cgroup_events_matrix; declaration socket_negative_paths; declaration stat_fs_detects_tmpfs; declaration supervisor_smoke; declaration tmux_control_fails_closed | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/svc_tests.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-81618bc6fbfe"></a>

## [cmd/soda-project-terminal/src/sys.rs](../../../../../cmd/soda-project-terminal/src/sys.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–482; current module/import/attribute shell; declaration monotonic; declaration ORIGIN; declaration now_secs; declaration spawn_argv; declaration wait_timeout; declaration run_checked; declaration run_output; declaration OUTPUT_LIMIT; declaration DRAIN_GRACE; declaration set_pipe_nonblocking; declaration check_component; declaration open_child_dir; declaration open_root; declaration open_at; declaration flock_exclusive_nb; declaration flock_blocking; declaration flock_shared; declaration flock_exclusive; declaration read_uuid; declaration tests; declaration TEST_SEQ; declaration test_dir; declaration argv; declaration clocks; declaration run_checked_matrix; declaration run_checked_timeout_kills; declaration run_output_matrix; declaration run_output_large_no_deadlock; declaration run_output_drains_both_pipes_in_bounded_passes; declaration run_output_fails_promptly_at_per_stream_bound; declaration run_output_requires_pipe_eof_after_child_exit; declaration lock_contention_two_fds; declaration open_walk_and_component_rule; declaration opened_fds_are_cloexec; declaration uuid_shape | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/sys.rs into its current native target.; 36 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-caeae0b041b2"></a>

## [cmd/soda-project-terminal/src/term.rs](../../../../../cmd/soda-project-terminal/src/term.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–103, 147–157; current module/import/attribute shell; declaration list_owned_terminals; declaration mutate_terminal; declaration term_binding_tests; declaration term_protocol_tests; declaration term_native_tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 104–146; declaration control_terminal | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | control_terminal: implement the current human terminal lifecycle duty in term.rs. — current source cmd/soda-project-terminal/src/term.rs; lines 104-146; module/caller wiring inspected |

<a id="coverage-e3fea4bcbc8b"></a>

## [cmd/soda-project-terminal/src/term_attach.rs](../../../../../cmd/soda-project-terminal/src/term_attach.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; current module/import/attribute shell; declaration lifetime_argv; declaration subscription_lifetime; declaration subscription_command; declaration attach_terminal; declaration attach_inner | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_attach.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-dc28c2568b25"></a>

## [cmd/soda-project-terminal/src/term_binding.rs](../../../../../cmd/soda-project-terminal/src/term_binding.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–280; current module/import/attribute shell; declaration BindingRecord; fields login, uid, gid, home, shell, identity, cols, rows, created_at; declaration Reservation; fields expires, scope; declaration RawObject; declaration raw; declaration deserialize; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map; declaration validate_binding; declaration binding_matches_account; declaration binding_record; declaration validate_reservation; declaration read_reservation; declaration permit_live; declaration binding_object; declaration reservation_object; declaration write_name | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_binding.rs into its current native target.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c5f191cfb6c8"></a>

## [cmd/soda-project-terminal/src/term_binding_tests.rs](../../../../../cmd/soda-project-terminal/src/term_binding_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–237; current module/import/attribute shell; declaration sample; declaration binding_doc; declaration good_account; declaration parse; declaration path_matrix; declaration binding_matrix; declaration account_match_matrix; declaration reservation_matrix; declaration missing_reservation_is_none; declaration classify_matrix | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_binding_tests.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eeeb9518e768"></a>

## [cmd/soda-project-terminal/src/term_collect.rs](../../../../../cmd/soda-project-terminal/src/term_collect.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–176; current module/import/attribute shell; declaration terminal_directories; declaration read_single_byte; declaration stub_retire; declaration remove_screen_contents; declaration remove_owned_files; declaration collect_finished | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_collect.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b726ece2f2b9"></a>

## [cmd/soda-project-terminal/src/term_create.rs](../../../../../cmd/soda-project-terminal/src/term_create.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–260; current module/import/attribute shell; declaration program_stat_ok; declaration file_sha256_hex; declaration verify_program; declaration reserve_terminal; declaration systemd_run_argv; declaration create_terminal | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_create.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-deb8e7dc538c"></a>

## [cmd/soda-project-terminal/src/term_native_tests.rs](../../../../../cmd/soda-project-terminal/src/term_native_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; current module/import/attribute shell; declaration program_hash_streams_past_64k; declaration ready_matrix; declaration entry_point_smoke; declaration tmux_config_exact | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_native_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-04a279e97653"></a>

## [cmd/soda-project-terminal/src/term_paths.rs](../../../../../cmd/soda-project-terminal/src/term_paths.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–117; current module/import/attribute shell; declaration TERMINALS; declaration PROGRAM; declaration s_isreg; declaration s_issock; declaration TMUX_CONFIG; declaration terminal_path; declaration checked_chain; declaration list_dir_names; declaration fstatat; declaration record_exists; declaration write_stdout_best_effort; declaration closed_launch_failed | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_paths.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5ef5ad0dd60f"></a>

## [cmd/soda-project-terminal/src/term_prepare.rs](../../../../../cmd/soda-project-terminal/src/term_prepare.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–187; current module/import/attribute shell; declaration tmux_new_session_args; declaration subscription_session; declaration prepare; declaration prepare_inner | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_prepare.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c3a2ce5ed82f"></a>

## [cmd/soda-project-terminal/src/term_protocol_tests.rs](../../../../../cmd/soda-project-terminal/src/term_protocol_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–170; current module/import/attribute shell; declaration systemd_argv_exact; declaration tmux_session_argv_exact; declaration subscription_command_exact; declaration lifetime_rule; declaration emission_bytes_exact | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_protocol_tests.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ecd43734f522"></a>

## [cmd/soda-project-terminal/src/term_status.rs](../../../../../cmd/soda-project-terminal/src/term_status.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–225; current module/import/attribute shell; declaration UnitClass; declaration classify_bare_state; declaration status_object; declaration ReadyFields; declaration deserialize; declaration ReadyVisitor; declaration Value; declaration expecting; declaration visit_map; declaration parse_ready; declaration writer_attached; declaration observe_ready_terminal; declaration classify_unit_state; declaration terminal_status; declaration ready_object | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/term_status.rs into its current native target.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-edb75ef0cd82"></a>

## [cmd/soda-project-terminal/src/timex.rs](../../../../../cmd/soda-project-terminal/src/timex.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 19–21; current module/import/attribute shell; declaration timex_tests | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/timex.rs into its current native target.; Exercise the named native behavior in timex.rs. — current source cmd/soda-project-terminal/src/timex.rs; Cargo target and callers; current source cmd/soda-project-terminal/src/timex.rs; lines 19-21; module/caller wiring inspected |
| 4–18; declaration now_secs; declaration parse_iso_deadline | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | now_secs: implement the current wire decoding, encoding, and representation conversion duty in timex.rs.; parse_iso_deadline: implement the current wire decoding, encoding, and representation conversion duty in timex.rs. — current source cmd/soda-project-terminal/src/timex.rs; lines 4-13; module/caller wiring inspected; current source cmd/soda-project-terminal/src/timex.rs; lines 14-18; module/caller wiring inspected |

<a id="coverage-78b85c1d4ee5"></a>

## [cmd/soda-project-terminal/src/timex_tests.rs](../../../../../cmd/soda-project-terminal/src/timex_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; current module/import/attribute shell; declaration strict_wire_deadlines_decode_to_unix_seconds; declaration strict_wire_deadlines_reject_permissive_iso_shapes; declaration now_secs_tracks_system_wall_clock | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/timex_tests.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-39def8063498"></a>

## [cmd/soda-project-terminal/src/tmux.rs](../../../../../cmd/soda-project-terminal/src/tmux.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–83; current module/import/attribute shell; declaration infocmp_argv; declaration tmux_argv; declaration tmux_control | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/tmux.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fc3bf1566148"></a>
<a id="coverage-c501e6df156b"></a>

## [cmd/soda-project-terminal/tests/cli.rs](../../../../../cmd/soda-project-terminal/tests/cli.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; current module/import/attribute shell | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Imports and module declarations wire cmd/soda-project-terminal/tests/cli.rs into its current native target. — current source cmd/soda-project-terminal/tests/cli.rs; Cargo target and callers |
| 8–133; declaration bin_path; declaration run_stdin; declaration CLOSED; declaration prepare_bogus_id_closed; declaration prepare_missing_locator_closed; declaration bad_argv_closed; declaration control_bad_identifier_closed; declaration KEYS_ERR; declaration keys_invalid_stdin; declaration keys_duplicate_field_rejected; declaration keys_wrong_shape_rejected; declaration keys_valid_shape_unconfirmed_without_identity; declaration BROKER_ERR | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | bin_path: one project-native terminal helper with managed sessions, keys and subscription operations.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 134–168; declaration broker_invalid_stdin; declaration broker_oversize_rejected; declaration broker_unknown_action_without_delivery; declaration broker_missing_action_rejected | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | broker_invalid_stdin: source assertion of provider execution integration.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-89b47e00f108"></a>
<a id="rustsoda-project-terminalsrcbrokerrs-1"></a>

Former source `rust/soda-project-terminal/src/broker.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2e3a2d057922"></a>
<a id="rustsoda-project-terminalsrcfsrs-1"></a>

Former source `rust/soda-project-terminal/src/fs.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2f067f5acd60"></a>
<a id="rustsoda-project-terminalsrckeysrs-1"></a>

Former source `rust/soda-project-terminal/src/keys.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-649478827042"></a>
<a id="rustsoda-project-terminalsrcptyrs-1"></a>

Former source `rust/soda-project-terminal/src/pty.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-33d09196cd88"></a>
<a id="rustsoda-project-terminalsrcsvcrs-1"></a>

Former source `rust/soda-project-terminal/src/svc.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-ad4c5aae1e49"></a>
<a id="rustsoda-project-terminalsrctermrs-1"></a>

Former source `rust/soda-project-terminal/src/term.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-1f5242f69516"></a>
<a id="rustsoda-project-terminalsrctimexrs-1"></a>

Former source `rust/soda-project-terminal/src/timex.rs`; consult its pinned earlier Git source and the current coverage disposition.

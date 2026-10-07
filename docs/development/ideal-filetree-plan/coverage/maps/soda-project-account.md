# Soda project account

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-7d631ec1ed65"></a>

## [cmd/soda-project-terminal/src/account.rs](../../../../../cmd/soda-project-terminal/src/account.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–304; current module/import/attribute shell; declaration s_isreg; declaration Account; fields pw_name, pw_uid, pw_gid, pw_dir, pw_shell; declaration valid_login; declaration login_shell_ok; declaration marker_matches; declaration account_for; declaration lookup_passwd; declaration account_binding; declaration user_environment; declaration become_user; declaration tests; declaration sample; declaration login_matrix; declaration shell_matrix; declaration marker_matrix; declaration binding_and_env_exact; declaration account_for_refuses_unprivileged; declaration become_user_fails_unprivileged | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/account.rs into its current native target.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6644fd357fe6"></a>

## [cmd/soda-project-terminal/src/bin/project-account.rs](../../../../../cmd/soda-project-terminal/src/bin/project-account.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–158; current module/import/attribute shell; declaration project_account; declaration REQUEST_LIMIT; declaration FAILURE_LINE; declaration read_request; declaration response_line; declaration config_from_env; declaration fail; declaration run; declaration main; declaration tests; declaration response_bytes_match_print_json_dumps; declaration failure_line_matches_sys_exit; declaration test_root_parses_env | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/bin/project-account.rs into its current native target.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e856ffffedda"></a>

## [cmd/soda-project-terminal/src/project_account.rs](../../../../../cmd/soda-project-terminal/src/project_account.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106, 174–195, 235–331, 360–526; current module/import/attribute shell; declaration Request; fields login, identity, admin, keys; declaration Response; fields login, identity; declaration RunCommand; declaration SyncFile; declaration Config; fields accounts, keys, passwd_file, expect_uid, expect_gid, require_root, fail_sync, run_command, sync_file; declaration production; declaration test_root; declaration valid_login; declaration RequestFields; declaration deserialize; declaration FieldsVisitor; declaration Value; declaration expecting; declaration visit_map; declaration run_command_inherit; declaration useradd_argv; declaration usermod_argv; declaration cstring; declaration open_nofollow; declaration ReadError; declaration owned_contents; declaration lock_exclusive_nb; declaration lexists; declaration system_home; declaration test_passwd_home; declaration lookup_home; declaration provision; declaration project_account_tests | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/project_account.rs into its current native target.; 29 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 23–31, 46–62, 107–173, 332–359; declaration is_py_space; declaration valid_key_head; declaration valid_key_body; declaration valid_key; declaration sync_one; declaration create_file; field Request.keys; field Config.keys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | is_py_space: parse, preserve, or synchronize the saved Project SSH authorized-key data.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 196–234; declaration parse_request | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Decode the exact project-account helper request shape. — current source cmd/soda-project-terminal/src/project_account.rs; lines 196-234; module/caller wiring inspected |

<a id="coverage-950e69227ce4"></a>

## [cmd/soda-project-terminal/src/project_account_tests.rs](../../../../../cmd/soda-project-terminal/src/project_account_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88, 159–306, 368–563; current module/import/attribute shell; declaration TEST_SEQ; declaration test_root; declaration request; declaration direct_config; declaration login_matrix; declaration key_length_counts_characters_not_bytes; declaration parse_doc; declaration request_shape_matrix; declaration request_scalar_matrix; declaration mode_of; declaration provisions_locked_home_marker_shared_empty_keyfile; declaration occupied_inputs_and_unassociated_users_refuse; declaration key_symlink_is_not_followed; declaration existing_account_binding_matrix; declaration lock_contention_refuses_before_effects; declaration fsync_order_holds_lock_and_orders_durably; declaration failed_sync_releases_lock_and_keeps_partial_files; declaration unsafe_directories_refuse; declaration unprivileged_production_gate | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire cmd/soda-project-terminal/src/project_account_tests.rs into its current native target.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 89–158, 307–367; declaration key_oracle; declaration selected_keys_rerun_is_idempotent; declaration join_never_applies_keys_and_preserves_drift | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Exercise saved SSH public-key parsing and normalization.; Exercise idempotent synchronization of selected saved SSH keys.; Exercise the boundary between membership and native SSH key application. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-397a7869f303"></a>
<a id="coverage-a02d25ff6350"></a>

## [cmd/soda-project-terminal/tests/project_account.rs](../../../../../cmd/soda-project-terminal/tests/project_account.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–178, 219–271, 286–410; current module/import/attribute shell; declaration FAILURE_LINE; declaration bin_path; declaration TEST_SEQ; declaration Env; fields root; declaration setup; declaration run; declaration commands; declaration mode; declaration read; declaration doc; declaration provisions_locked_home_marker_shared_empty_keyfile; declaration retry_never_erases_existing_keys; declaration join_never_applies_keys_and_preserves_drift; declaration occupied_inputs_and_unassociated_users_refuse; declaration validation_refuses_before_native_effects; declaration lock_contention_refuses_before_effects; declaration failed_sync_releases_lock_and_keeps_partial_files; declaration stdin_size_boundary_is_exact; declaration malformed_stdin_refuses; declaration argv_is_ignored_like_the_python | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Imports and module declarations wire cmd/soda-project-terminal/tests/project_account.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 179–218, 272–285; declaration selected_keys_admin_rerun_is_idempotent; declaration key_symlink_is_not_followed | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | selected_keys_admin_rerun_is_idempotent: source assertion of development ssh access.; key_symlink_is_not_followed: source assertion of development ssh access. — current source cmd/soda-project-terminal/tests/project_account.rs; lines 179-218; module/caller wiring inspected; exact byte-identical source map rust/soda-project-account/tests/binary.rs; exact byte-identical prior map docs/development/ideal-filetree-plan/coverage/maps/soda-project-account.md at the same current line; current source cmd/soda-project-terminal/tests/project_account.rs; lines 272-285; module/caller wiring inspected; exact byte-identical source map rust/soda-project-account/tests/binary.rs; exact byte-identical prior map docs/development/ideal-filetree-plan/coverage/maps/soda-project-account.md at the same current line |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-8955aff292da"></a>

Former source `rust/soda-project-account/src/account.rs`; consult its pinned earlier Git source and the current coverage disposition.

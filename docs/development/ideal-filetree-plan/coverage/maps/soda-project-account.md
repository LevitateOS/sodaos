# Soda project account

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Historical responsibility accounting at `519b76bd` (2026-10-07).
The account/SSH selector sections below are refreshed at `e0e2413f` (2026-10-09);
other rows retain their stated source and evidence scopes.
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

Current declaration and caller allocation inspected at `e0e2413f`; earlier body and test evidence retains its recorded scope.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31, 33–49, 51–55, 57–67, 69–73, 75–85, 87–92, 94–99, 101–158, 160–162, 164–176, 178–231; module/import scaffold; project_account; REQUEST_LIMIT; STATUS_REQUEST_LIMIT; FAILURE_LINE; STATUS_FAILURE_LINE; read_request_from; read_request; response_line; privilege_response_line; config_from_env; fail; fail_status; run; main; reader_tests; request_reader_consumes_at_most_one_refusal_byte_over_limit; tests; response_bytes_match_print_json_dumps; failure_line_matches_sys_exit; test_root_parses_env | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | retained |

<a id="coverage-e856ffffedda"></a>

## [cmd/soda-project-terminal/src/project_account.rs](../../../../../cmd/soda-project-terminal/src/project_account.rs)

Current declaration and caller allocation inspected at `e0e2413f`; earlier body and test evidence retains its recorded scope.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 38–53, 252–260, 262–327, 329–470; PrivilegeRequest; PrivilegeResponse; parse_privilege_request; open_accounts_directory; read_account_marker; confirm_account_marker; NativeAccountIDs; lookup_native_account; lookup_wheel_gid; native_group_limit; lookup_native_groups; checked_native_group_count; is_project_administrator; project_privilege_status | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | The separate `PrivilegeRequest`/status path requires exact fields and independently observes current native passwd/NSS groups after binding the exact managed marker; root or current wheel membership is the only positive guest result. Source scope is covered by the 39 compiled guest checks; no installed/native qualification is claimed. |
| 1–20, 22–27, 29, 31–36, 55–58, 60–65, 67–76, 78–110, 112–122, 471–481, 483–498, 500–505, 507–509, 511–527, 529–532, 536–566, 611–613, 615–646, 648–660, 662–667, 669–759, 761–762; file scaffold; Request (excluding keys); Response; RunCommand; SyncFile; Config (excluding keys); Config::production; Config::test_root; valid_login; run_command_inherit; useradd_argv; usermod_argv; cstring; open_nofollow; ReadError; owned_contents; lexists; system_home; test_passwd_home; lookup_home; provision; project_account_tests module wiring | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Account request, provisioning, identity/home lookup and local marker/file admission. P04 initial-key handling consumes the shared custody helpers. Complete helpers retain one primary owner; the saved-key fields have separate P04 selectors. |
| 28, 66, 124–188, 568–573, 575–592, 594–609; Request.keys; Config.keys; is_py_space; valid_key_head; valid_key_body; valid_key; sync_one; create_file; lock_exclusive_nb | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | SSH key input/validation, managed-key directory locking and the existing durable-file helpers. P03 provision uses create_file/sync_one for both account marker and initial keyfile; those callers are cross-duty evidence. Field and helper spans correct the old offsets without creating a second owner. |
| 190–212, 214–250; RequestFields and visitor; parse_request | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Typed provisioning-request decoding; existing allocation retained. |

<a id="coverage-950e69227ce4"></a>

## [cmd/soda-project-terminal/src/project_account_tests.rs](../../../../../cmd/soda-project-terminal/src/project_account_tests.rs)

Current declaration and caller allocation inspected at `e0e2413f`; earlier body and test evidence retains its recorded scope.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 221–359; privilege_request_is_exact_and_bounded_by_identity_fields; write_status_marker; privilege_marker_is_bound_to_open_managed_directory_and_exact_identity; privilege_marker_refuses_unsafe_accounts_directory; privilege_decision_and_group_count_fail_closed; current_process_passwd_and_groups_are_resolved_through_nss | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current tests cover exact status input, bounded open-marker binding, unsafe directory refusal, root/wheel classification, and real current-process NSS lookup. These selectors are part of the 25 guest unit checks; guest source is not installed-system qualification. |
| 1–63, 64–90, 173–176, 178–220, 403–406, 408–449, 490–510, 511–530, 547–572, 573–594, 595–663, 664–680, 681–699, 700–709; file scaffold; TEST_SEQ; test_root; request; direct_config; login_matrix; parse_doc; request_shape_matrix; mode_of; provisions_locked_home_marker_shared_empty_keyfile; join_never_applies_keys_and_preserves_drift; occupied_inputs_and_unassociated_users_refuse; existing_account_binding_matrix; lock_contention_refuses_before_effects; fsync_order_holds_lock_and_orders_durably; failed_sync_releases_lock_and_keeps_partial_files; unsafe_directories_refuse; unprivileged_production_gate | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Complete provisioning/admission fixtures and tests have one primary account owner; combined tests include H03 request decoding and P04 key preservation, lock, initial-file and durability assertions. Source helper ownership remains separately assigned above. |
| 91–160, 161–171, 450–489, 531–546; key_oracle; key_length_counts_characters_not_bytes; selected_keys_rerun_is_idempotent; key_symlink_is_not_followed | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | SSH key parsing/length, selected-key rerun and keyfile symlink refusal tests. |
| 362–402; request_scalar_matrix | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Provisioning-request scalar decoder cases. |

<a id="coverage-397a7869f303"></a>
<a id="coverage-a02d25ff6350"></a>

## [cmd/soda-project-terminal/tests/project_account.rs](../../../../../cmd/soda-project-terminal/tests/project_account.rs)

Current declaration and caller allocation inspected at `e0e2413f`; earlier body and test evidence retains its recorded scope.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 409–418, 420–464, 467–518, 521–533; unsupported_argument_tail_refuses_before_provisioning; native_account_uid; wheel_group_resolves_through_nss; status_mode_reads_current_native_nss_without_provisioning; status_mode_refuses_oversized_request_before_account_or_nss_effects | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current integration checks cover unsupported argument tails, current NSS observation, no mutating command, and over-cap refusal before account/NSS effects. — Current selectors. |
| 1–35, 36–136, 137–160, 162–182, 236–252, 253–275, 290–309, 335–351; file scaffold; FAILURE_LINE; STATUS_FAILURE_LINE; bin_path; TEST_SEQ; Env and methods; doc; provisions_locked_home_marker_shared_empty_keyfile; join_never_applies_keys_and_preserves_drift; occupied_inputs_and_unassociated_users_refuse; validation_refuses_before_native_effects; failed_sync_releases_lock_and_keeps_partial_files | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Actual compiled-helper fixture and complete account provisioning/refusal tests. Shared fixture and combined tests have one primary owner; initial keyfile, drift and sync/lock assertions are P04 cross-duty evidence. Native commands use the existing test-root seam, so this is not installed privilege-boundary qualification. |
| 183–222, 223–235, 276–289, 310–334; selected_keys_admin_rerun_is_idempotent; retry_never_erases_existing_keys; key_symlink_is_not_followed; lock_contention_refuses_before_effects | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Selected-key rerun, existing-key preservation, symlink and key-directory lock refusals. |
| 352–387, 388–407; stdin_size_boundary_is_exact; malformed_stdin_refuses | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Exact bounded stdin admission and malformed provisioning-request refusal; key-bearing inputs are parser fixtures, not a separate key-operation owner. |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-8955aff292da"></a>

Former source `rust/soda-project-account/src/account.rs`; consult its pinned earlier Git source and the current coverage disposition.

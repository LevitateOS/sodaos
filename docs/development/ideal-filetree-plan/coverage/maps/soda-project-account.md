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

Current status-mode declarations and tests verified; unaffected provisioning intervals are historical selector hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 26–48, 69–73, 101–158, 164–176, 178–231; current module/import/attribute shell; declaration project_account; declaration REQUEST_LIMIT; declaration STATUS_REQUEST_LIMIT; declaration read_request_from; declaration privilege_response_line; declaration fail_status; declaration run; declaration reader_tests; declaration request_reader_consumes_at_most_one_refusal_byte_over_limit; declaration tests; declaration response_bytes_match_print_json_dumps; declaration failure_line_matches_sys_exit; declaration test_root_parses_env | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | No-argument provisioning stays intact; exact `--status` selects the bounded read-only status path and all other arguments refuse before input/effects. The separate reader regression proves cap+one refusal. Guest implementation/tests are source-verified and included in the 39 compiled guest checks; no installed-container qualification is claimed. |

<a id="coverage-e856ffffedda"></a>

## [cmd/soda-project-terminal/src/project_account.rs](../../../../../cmd/soda-project-terminal/src/project_account.rs)

Current Project privilege status declarations and NSS implementation verified; prior provisioning intervals are historical selector hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 38–53, 252–260, 262–327, 329–470; PrivilegeRequest; PrivilegeResponse; parse_privilege_request; open_accounts_directory; read_account_marker; confirm_account_marker; NativeAccountIDs; lookup_native_account; lookup_wheel_gid; native_group_limit; lookup_native_groups; checked_native_group_count; is_project_administrator; project_privilege_status | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | The separate `PrivilegeRequest`/status path requires exact fields and independently observes current native passwd/NSS groups after binding the exact managed marker; root or current wheel membership is the only positive guest result. Source scope is covered by the 39 compiled guest checks; no installed/native qualification is claimed. |
| 672–762; provision | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current existing account-provisioning entry point; its behavior is unchanged by the separate status mode. — Current source selector. |
| Previously mapped P03 provisioning/custody helpers (current spans pending R02); RequestFields; parse_request; provision; owned_contents; lock_exclusive_nb; create_file; sync_one | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Existing account provisioning and managed-file custody remain allocated; refresh current intervals under R02. |
| 27, 65, 125–188, 568–595; Request.keys; Config.keys; is_py_space; valid_key_head; valid_key_body; valid_key; sync_one; create_file | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | is_py_space: parse, preserve, or synchronize the saved Project SSH authorized-key data.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 190–212, 214–250; RequestFields; parse_request | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Decode the existing project-account provisioning request. — Current source selectors. |

<a id="coverage-950e69227ce4"></a>

## [cmd/soda-project-terminal/src/project_account_tests.rs](../../../../../cmd/soda-project-terminal/src/project_account_tests.rs)

Current privilege-status tests verified; unaffected provisioning intervals are historical selector hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 221–359; privilege_request_is_exact_and_bounded_by_identity_fields; write_status_marker; privilege_marker_is_bound_to_open_managed_directory_and_exact_identity; privilege_marker_refuses_unsafe_accounts_directory; privilege_decision_and_group_count_fail_closed; current_process_passwd_and_groups_are_resolved_through_nss | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current tests cover exact status input, bounded open-marker binding, unsafe directory refusal, root/wheel classification, and real current-process NSS lookup. These selectors are part of the 25 guest unit checks; guest source is not installed-system qualification. |
| 88–172, 450–509, 531–546; key_oracle; key_length_counts_characters_not_bytes; selected_keys_rerun_is_idempotent; join_never_applies_keys_and_preserves_drift; key_symlink_is_not_followed | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Exercise saved SSH public-key parsing and normalization.; Exercise idempotent synchronization of selected saved SSH keys.; Exercise the boundary between membership and native SSH key application. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-397a7869f303"></a>
<a id="coverage-a02d25ff6350"></a>

## [cmd/soda-project-terminal/tests/project_account.rs](../../../../../cmd/soda-project-terminal/tests/project_account.rs)

Current read-only status and unsupported-argument integration selectors verified; earlier provisioning intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 409–418, 420–464, 467–518, 521–533; unsupported_argument_tail_refuses_before_provisioning; native_account_uid; wheel_group_resolves_through_nss; status_mode_reads_current_native_nss_without_provisioning; status_mode_refuses_oversized_request_before_account_or_nss_effects | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current integration checks cover unsupported argument tails, current NSS observation, no mutating command, and over-cap refusal before account/NSS effects. — Current selectors. |
| Prior P03 provisioning fixture and custody selectors (current spans pending R02); provisions_locked_home_marker_shared_empty_keyfile; retry_never_erases_existing_keys; occupied_inputs_and_unassociated_users_refuse; validation_refuses_before_native_effects; lock_contention_refuses_before_effects; failed_sync_releases_lock_and_keeps_partial_files; stdin_size_boundary_is_exact; malformed_stdin_refuses | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Existing provisioning behavior remains in the integration test target; selector intervals await R02 refresh. |
| 184–223, 277–290; selected_keys_admin_rerun_is_idempotent; key_symlink_is_not_followed | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Existing selected-key rerun and symlink-refusal tests remain assigned to SSH access. — Current source selectors. |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-8955aff292da"></a>

Former source `rust/soda-project-account/src/account.rs`; consult its pinned earlier Git source and the current coverage disposition.

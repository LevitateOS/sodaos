# Soda project account

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-8955aff292da"></a>

## [rust/soda-project-account/src/account.rs](../../../../../rust/soda-project-account/src/account.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1–20 | Human account request/response/native configuration; declarations/fields: `Request`, `Response`, `Config`, `production`, `test_root`, `valid_login`, `is_py_space` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 21 | Human account request/response/native configuration; declaration/member Request; declarations/fields: `Request` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 22 | Human account request/response/native configuration; declaration/member Request.login; declarations/fields: `Request.login` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 23 | Human account request/response/native configuration; declaration/member Request.identity; declarations/fields: `Request.identity` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 24 | Human account request/response/native configuration; declaration/member Request.admin; declarations/fields: `Request.admin` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 25–29 | Human account request/response/native configuration; declaration/member Request.keys; declarations/fields: `Request.keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 30 | Human account request/response/native configuration; declaration/member Response; declarations/fields: `Response` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 31 | Human account request/response/native configuration; declaration/member Response.login; declarations/fields: `Response.login` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 32–35 | Human account request/response/native configuration; declaration/member Response.identity; declarations/fields: `Response.identity` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 36–37 | Human account request/response/native configuration; declaration/member RunCommand; declarations/fields: `RunCommand` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 38–42 | Human account request/response/native configuration; declaration/member SyncFile; declarations/fields: `SyncFile` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 43 | Human account request/response/native configuration; declaration/member Config; declarations/fields: `Config` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 44 | Human account request/response/native configuration; declaration/member Config.accounts; declarations/fields: `Config.accounts` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 45–46 | Human account request/response/native configuration; declaration/member Config.keys; declarations/fields: `Config.keys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 47 | Human account request/response/native configuration; declaration/member Config.passwd_file; declarations/fields: `Config.passwd_file` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 48 | Human account request/response/native configuration; declaration/member Config.expect_uid; declarations/fields: `Config.expect_uid` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 49 | Human account request/response/native configuration; declaration/member Config.expect_gid; declarations/fields: `Config.expect_gid` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 50–51 | Human account request/response/native configuration; declaration/member Config.require_root; declarations/fields: `Config.require_root` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 52 | Human account request/response/native configuration; declaration/member Config.fail_sync; declarations/fields: `Config.fail_sync` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 53 | Human account request/response/native configuration; declaration/member Config.run_command; declarations/fields: `Config.run_command` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 54–59 | Human account request/response/native configuration; declaration/member Config.sync_file; declarations/fields: `Config.sync_file` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 60–75 | Human account request/response/native configuration; declaration/member production; declarations/fields: `production` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 76–92 | Human account request/response/native configuration; declaration/member test_root; declarations/fields: `test_root` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 93–103 | Human account request/response/native configuration; declaration/member valid_login; declarations/fields: `valid_login` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 104–113 | Human account request/response/native configuration; declaration/member is_py_space; declarations/fields: `is_py_space` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 114–136 | Initial selected-key shape admission; declarations/fields: `valid_key_head` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 137–155 | Initial selected-key shape admission; declaration/member valid_key_body; declarations/fields: `valid_key_body` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 156–170 | Initial selected-key shape admission; declaration/member valid_key; declarations/fields: `valid_key` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 171–220 | Account request decoding and native account commands; declarations/fields: `parse_request` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 221–232 | Account request decoding and native account commands; declaration/member run_command_inherit; declarations/fields: `run_command_inherit` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 233–249 | Account request decoding and native account commands; declaration/member useradd_argv; declarations/fields: `useradd_argv` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 250–256 | Account request decoding and native account commands; declaration/member usermod_argv; declarations/fields: `usermod_argv` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 257–262 | No-follow owned read/create/sync/lock primitives; declarations/fields: `cstring` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 263–278 | No-follow owned read/create/sync/lock primitives; declaration/member open_nofollow; declarations/fields: `open_nofollow` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 279–285 | No-follow owned read/create/sync/lock primitives; declaration/member ReadError; declarations/fields: `ReadError` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 286–317 | No-follow owned read/create/sync/lock primitives; declaration/member owned_contents; declarations/fields: `owned_contents` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 318–326 | No-follow owned read/create/sync/lock primitives; declaration/member sync_one; declarations/fields: `sync_one` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 327–345 | No-follow owned read/create/sync/lock primitives; declaration/member create_file; declarations/fields: `create_file` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 346–360 | No-follow owned read/create/sync/lock primitives; declaration/member lock_exclusive_nb; declarations/fields: `lock_exclusive_nb` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 361–364 | No-follow owned read/create/sync/lock primitives; declaration/member lexists; declarations/fields: `lexists` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 365–399 | Native home/account association lookup; declarations/fields: `system_home` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 400–411 | Native home/account association lookup; declaration/member test_passwd_home; declarations/fields: `test_passwd_home` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 412–421 | Native home/account association lookup; declaration/member lookup_home; declarations/fields: `lookup_home` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 422–425 | Project-local root account association admission; declarations/fields: `provision` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 426–437 | Exclusive managed-key directory lock/admission |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 438–479 | Account/key marker association and native locked account creation |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 480–495 | Initial selected authorized-key materialization; drift requires explicit maintenance |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 496–511 | Shared home link and explicit wheel administrator account |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 512–517, 684–688, 873–892, 909–934 | Native account fixture assertions; declarations/fields: `tests`, `parse_doc`, `occupied_inputs_and_unassociated_users_refuse`, `existing_account_binding_matrix` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 518–521 | Native account fixture assertions; declaration/member TEST_SEQ; declarations/fields: `TEST_SEQ` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 522–536 | Native account fixture assertions; declaration/member test_root; declarations/fields: `test_root` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 537–547 | Native account fixture assertions; declaration/member request; declarations/fields: `request` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 548–574 | Native account fixture assertions; declaration/member direct_config; declarations/fields: `direct_config` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 575–601 | Native account fixture assertions; declaration/member login_matrix; declarations/fields: `login_matrix` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 602–671, 812–851, 893–908 | Source assertion of Development SSH access; declarations/fields: `key_oracle`, `selected_keys_rerun_is_idempotent`, `key_symlink_is_not_followed` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 672–683 | Source assertion of Development SSH access; declaration/member key_length_counts_characters_not_bytes; declarations/fields: `key_length_counts_characters_not_bytes` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 689–724 | Native account fixture assertions; declaration/member request_shape_matrix; declarations/fields: `request_shape_matrix` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 725–764 | Native account fixture assertions; declaration/member request_scalar_matrix; declarations/fields: `request_scalar_matrix` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 765–769 | Native account fixture assertions; declaration/member mode_of; declarations/fields: `mode_of` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 770–811 | Native account fixture assertions; declaration/member provisions_locked_home_marker_shared_empty_keyfile; declarations/fields: `provisions_locked_home_marker_shared_empty_keyfile` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 852–872 | Source assertion of Human membership and accounts; declarations/fields: `join_never_applies_keys_and_preserves_drift` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 935–956 | Native account fixture assertions; declaration/member lock_contention_refuses_before_effects; declarations/fields: `lock_contention_refuses_before_effects` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 957–1025 | Native account fixture assertions; declaration/member fsync_order_holds_lock_and_orders_durably; declarations/fields: `fsync_order_holds_lock_and_orders_durably` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1026–1042 | Native account fixture assertions; declaration/member failed_sync_releases_lock_and_keeps_partial_files; declarations/fields: `failed_sync_releases_lock_and_keeps_partial_files` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1043–1061 | Native account fixture assertions; declaration/member unsafe_directories_refuse; declarations/fields: `unsafe_directories_refuse` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1062–1068 | Native account fixture assertions; declaration/member unprivileged_production_gate; declarations/fields: `unprivileged_production_gate` |

<a id="coverage-397a7869f303"></a>

## [rust/soda-project-account/tests/binary.rs](../../../../../rust/soda-project-account/tests/binary.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1–12, 220–232, 250–272, 287–306 | Project-local locked human account association and administrator membership; declarations/fields: `FAILURE_LINE`, `bin_path`, `TEST_SEQ`, `Env`, `setup`, `run`, `commands`, `mode`, `read`, `doc`, `provisions_locked_home_marker_shared_empty_keyfile`, `retry_never_erases_existing_keys`, `occupied_inputs_and_unassociated_users_refuse`, `validation_refuses_before_native_effects` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 13–15 | Project-local locked human account association and administrator membership; declaration/member FAILURE_LINE; declarations/fields: `FAILURE_LINE` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 16–27 | Project-local locked human account association and administrator membership; declaration/member bin_path; declarations/fields: `bin_path` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 28–31 | Project-local locked human account association and administrator membership; declaration/member TEST_SEQ; declarations/fields: `TEST_SEQ` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 32 | Project-local locked human account association and administrator membership; declaration/member Env; declarations/fields: `Env` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 33–36 | Project-local locked human account association and administrator membership; declaration/member Env.root; declarations/fields: `Env.root` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 37–74 | Project-local locked human account association and administrator membership; declaration/member setup; declarations/fields: `setup` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 75–111 | Project-local locked human account association and administrator membership; declaration/member run; declarations/fields: `run` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 112–123 | Project-local locked human account association and administrator membership; declaration/member commands; declarations/fields: `commands` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 124–127 | Project-local locked human account association and administrator membership; declaration/member mode; declarations/fields: `mode` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 128–132 | Project-local locked human account association and administrator membership; declaration/member read; declarations/fields: `read` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 133–158 | Project-local locked human account association and administrator membership; declaration/member doc; declarations/fields: `doc` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 159–179 | Project-local locked human account association and administrator membership; declaration/member provisions_locked_home_marker_shared_empty_keyfile; declarations/fields: `provisions_locked_home_marker_shared_empty_keyfile` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 180–219, 273–286 | Source assertion of Development SSH access; declarations/fields: `selected_keys_admin_rerun_is_idempotent`, `key_symlink_is_not_followed` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 233–249 | Source assertion of Human membership and accounts; declarations/fields: `join_never_applies_keys_and_preserves_drift` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 307–331 | Project-local locked human account association and administrator membership; declaration/member lock_contention_refuses_before_effects; declarations/fields: `lock_contention_refuses_before_effects` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 332–348 | Project-local locked human account association and administrator membership; declaration/member failed_sync_releases_lock_and_keeps_partial_files; declarations/fields: `failed_sync_releases_lock_and_keeps_partial_files` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 349–384 | Project-local locked human account association and administrator membership; declaration/member stdin_size_boundary_is_exact; declarations/fields: `stdin_size_boundary_is_exact` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 385–404 | Project-local locked human account association and administrator membership; declaration/member malformed_stdin_refuses; declarations/fields: `malformed_stdin_refuses` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 405–410 | Project-local locked human account association and administrator membership; declaration/member argv_is_ignored_like_the_python; declarations/fields: `argv_is_ignored_like_the_python` |


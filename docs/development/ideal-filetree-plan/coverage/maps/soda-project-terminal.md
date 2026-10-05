# Soda project terminal

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-89b47e00f108"></a>

<a id="rustsoda-project-terminalsrcbrokerrs-1"></a>

## [rust/soda-project-terminal/src/broker.rs](../../../../../rust/soda-project-terminal/src/broker.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–46 | Existing Codex execution helper and exact subscription files; declarations/fields: `REQUEST_LIMIT`, `CREDENTIAL_LIMIT`, `HORIZON_SECS`, `BROKER_ALARM_SECS` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 47–48 | Existing Codex execution helper and exact subscription files; declaration/member REQUEST_LIMIT; declarations/fields: `REQUEST_LIMIT` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 49–50 | Existing Codex execution helper and exact subscription files; declaration/member CREDENTIAL_LIMIT; declarations/fields: `CREDENTIAL_LIMIT` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 51–52 | Existing Codex execution helper and exact subscription files; declaration/member HORIZON_SECS; declarations/fields: `HORIZON_SECS` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 53–59 | Existing Codex execution helper and exact subscription files; declaration/member BROKER_ALARM_SECS; declarations/fields: `BROKER_ALARM_SECS` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 60–63 | JSON numeric/equality and wire conversion; declarations/fields: `last` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 64–69 | JSON numeric/equality and wire conversion; declaration/member number_f64; declarations/fields: `number_f64` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 70–84 | JSON numeric/equality and wire conversion; declaration/member number_equal; declarations/fields: `number_equal` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 85–119 | JSON numeric/equality and wire conversion; declaration/member json_equal; declarations/fields: `json_equal` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 120–127 | JSON numeric/equality and wire conversion; declaration/member json_int; declarations/fields: `json_int` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 128 | JSON numeric/equality and wire conversion; declaration/member LO; declarations/fields: `LO` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 129–155 | JSON numeric/equality and wire conversion; declaration/member HI; declarations/fields: `HI` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 156–171 | Native lease binding edit; declarations/fields: `lease_with_binding` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 172–177 | Lease live deadline predicate; declarations/fields: `deadline_ok` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 178–188 | Private result/request JSON; declarations/fields: `result_object` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 189–194 | Private result/request JSON; declaration/member empty_result; declarations/fields: `empty_result` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 195–203 | Private result/request JSON; declaration/member decode_request; declarations/fields: `decode_request` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 204–219 | Native binding/profile metadata construction; declarations/fields: `native_binding` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 220–240 | Native binding/profile metadata construction; declaration/member profile_object; declarations/fields: `profile_object` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 241–246 | Exact subscription path/actor/execution validation; declarations/fields: `subscription_path` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 247–254 | Exact subscription path/actor/execution validation; declaration/member path_missing; declarations/fields: `path_missing` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 255–261 | Exact subscription path/actor/execution validation; declaration/member lease_execution_id; declarations/fields: `lease_execution_id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 262–268 | Exact subscription path/actor/execution validation; declaration/member lease_actor_id; declarations/fields: `lease_actor_id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 269–280 | Exact subscription path/actor/execution validation; declaration/member mount_argv; declarations/fields: `mount_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 281–287 | Exact subscription path/actor/execution validation; declaration/member profile_deadline; declarations/fields: `profile_deadline` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 288–297 | Exact subscription path/actor/execution validation; declaration/member live_deadline; declarations/fields: `live_deadline` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 298–377 | Native subscription reservation/provisioning; declarations/fields: `subscription_prepare` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 378–423 | Native subscription reservation/provisioning; declaration/member subscription_provision; declarations/fields: `subscription_provision` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 424–431 | Native subscription reservation/provisioning; declaration/member ProfileHit; declarations/fields: `ProfileHit` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 432–467 | Exact private profile/unit/native binding attestation; declarations/fields: `subscription_profile` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 468–479 | Exact private profile/unit/native binding attestation; declaration/member subscription_invocation; declarations/fields: `subscription_invocation` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 480–524 | Exact private profile/unit/native binding attestation; declaration/member subscription_check_unit; declarations/fields: `subscription_check_unit` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 525–542 | No-follow file owner helper; declarations/fields: `fchownat_no_follow` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 543–548 | No-follow file owner helper; declaration/member is_regular; declarations/fields: `is_regular` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 549–562 | Private auth directory and credential/harness staging; declarations/fields: `subscription_auth_directory` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 563–581 | Private auth directory and credential/harness staging; declaration/member subscription_seed; declarations/fields: `subscription_seed` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 582–600 | Private auth directory and credential/harness staging; declaration/member subscription_stage; declarations/fields: `subscription_stage` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 601–618 | Private auth directory and credential/harness staging; declaration/member subscription_harness_digest; declarations/fields: `subscription_harness_digest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 619–634 | Native harness respawn/start; declarations/fields: `respawn_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 635–689 | Native harness respawn/start; declaration/member subscription_start; declarations/fields: `subscription_start` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 690–702 | Exact cgroup freeze and final auth capture; declarations/fields: `subscription_cgroup` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 703–729 | Exact cgroup freeze and final auth capture; declaration/member subscription_kernel_write; declarations/fields: `subscription_kernel_write` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 730–750 | Exact cgroup freeze and final auth capture; declaration/member parse_cgroup_events; declarations/fields: `parse_cgroup_events` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 751–772 | Exact cgroup freeze and final auth capture; declaration/member subscription_freeze; declarations/fields: `subscription_freeze` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 773–800 | Exact cgroup freeze and final auth capture; declaration/member subscription_capture; declarations/fields: `subscription_capture` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 801–808 | Private auth/model cleanup and native boundary retirement; declarations/fields: `unlink_missing_ok` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 809–819 | Private auth/model cleanup and native boundary retirement; declaration/member rmdir_missing_ok; declarations/fields: `rmdir_missing_ok` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 820–836 | Private auth/model cleanup and native boundary retirement; declaration/member subscription_remove_model; declarations/fields: `subscription_remove_model` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 837–862 | Private auth/model cleanup and native boundary retirement; declaration/member is_mount; declarations/fields: `is_mount` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 863–883 | Private auth/model cleanup and native boundary retirement; declaration/member subscription_retire; declarations/fields: `subscription_retire` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 884–932 | Private auth/model cleanup and native boundary retirement; declaration/member subscription_finish; declarations/fields: `subscription_finish` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 933–981 | Existing subscription lookup/resolve/operation dispatch; declarations/fields: `subscription_lookup` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 982–1010 | Existing subscription lookup/resolve/operation dispatch; declaration/member subscription_resolve; declarations/fields: `subscription_resolve` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1011–1050 | Existing subscription lookup/resolve/operation dispatch; declaration/member subscription_dispatch; declarations/fields: `subscription_dispatch` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1051–1055 | Bounded helper stdin/output entrypoint; declarations/fields: `BrokerFail` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1056–1065 | Bounded helper stdin/output entrypoint; declaration/member read_stdin; declarations/fields: `read_stdin` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1066–1074 | Bounded helper stdin/output entrypoint; declaration/member write_stdout_all; declarations/fields: `write_stdout_all` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1075–1091 | Bounded helper stdin/output entrypoint; declaration/member write_stderr_all; declarations/fields: `write_stderr_all` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1092–1104 | Bounded helper stdin/output entrypoint; declaration/member broker_inner; declarations/fields: `broker_inner` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1105–1119 | Bounded helper stdin/output entrypoint; declaration/member broker_main; declarations/fields: `broker_main` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1120–1122, 1212–1230, 1311–1313, 1333–1342 | Native subscription helper source assertions; declarations/fields: `tests`, `request_decode_matrix`, `TEST_SEQ`, `dispatch_rejects_garbage_without_touching_fs` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1123–1127 | Native subscription helper source assertions; declaration/member parse; declarations/fields: `parse` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1128–1161 | Source assertion of Encoding and parsing; declarations/fields: `equality_matrix` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1162–1183 | Source assertion of Encoding and parsing; declaration/member int_conversion_matrix; declarations/fields: `int_conversion_matrix` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 1184–1202, 1304–1310 | Source assertion of Native binding and private delivery; declarations/fields: `lease_binding_edit`, `mount_probe_matrix` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 1203–1211 | Source assertion of Execution admission and lease fencing; declarations/fields: `deadline_window` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1231–1262 | Native subscription helper source assertions; declaration/member result_shapes_exact; declarations/fields: `result_shapes_exact` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1263–1287 | Source assertion of Provider execution integration; declarations/fields: `argv_constructors_exact` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1288–1303, 1314–1332 | Source assertion of Completion, revocation and reconciliation; declarations/fields: `cgroup_events_matrix`, `remove_model_matrix` |

<a id="coverage-2e3a2d057922"></a>

<a id="rustsoda-project-terminalsrcfsrs-1"></a>

## [rust/soda-project-terminal/src/fs.rs](../../../../../rust/soda-project-terminal/src/fs.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–13 | No-follow bounded private filesystem primitives; declarations/fields: `cstr`, `check_component`, `cvt`, `root_chain`, `s_isreg`, `s_isdir`, `s_islnk`, `stat_is_safe`, `root_file`, `read_record_inner`, `read_record`, `new_file`, `mkdir_at`, `fchmod`, `unlink_at`, `rmdir_at`, `chown_path`, `chmod_path`, `replace_at`, `fsync_file`, `fstat_uid_mode`, `fstat_all`, `read_up_to`, `tests`, `TEST_SEQ`, `test_dir`, `dir_fd`, `am_root`, `root_chain_matrix`, `fake_stat`, `safety_predicate_matrix`, `root_file_failures`, `record_size_and_json_limits`, `read_record_public_paths`, `new_file_create_and_collision`, `wrapper_matrix`, `path_chown_chmod_matrix`, `stat_all_and_single_read` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 14–19 | No-follow bounded private filesystem primitives; declaration/member cstr; declarations/fields: `cstr` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 20–29 | No-follow bounded private filesystem primitives; declaration/member check_component; declarations/fields: `check_component` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 30–44 | No-follow bounded private filesystem primitives; declaration/member cvt; declarations/fields: `cvt` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 45–55 | No-follow bounded private filesystem primitives; declaration/member root_chain; declarations/fields: `root_chain` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 56–60 | No-follow bounded private filesystem primitives; declaration/member s_isreg; declarations/fields: `s_isreg` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 61–64 | No-follow bounded private filesystem primitives; declaration/member s_isdir; declarations/fields: `s_isdir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 65–70 | No-follow bounded private filesystem primitives; declaration/member s_islnk; declarations/fields: `s_islnk` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 71–90 | No-follow bounded private filesystem primitives; declaration/member stat_is_safe; declarations/fields: `stat_is_safe` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 91–105 | No-follow bounded private filesystem primitives; declaration/member root_file; declarations/fields: `root_file` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 106–121 | No-follow bounded private filesystem primitives; declaration/member read_record_inner; declarations/fields: `read_record_inner` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 122–131 | No-follow bounded private filesystem primitives; declaration/member read_record; declarations/fields: `read_record` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 132–147 | No-follow bounded private filesystem primitives; declaration/member new_file; declarations/fields: `new_file` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 148–154 | No-follow bounded private filesystem primitives; declaration/member mkdir_at; declarations/fields: `mkdir_at` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 155–159 | No-follow bounded private filesystem primitives; declaration/member fchmod; declarations/fields: `fchmod` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 160–166 | No-follow bounded private filesystem primitives; declaration/member unlink_at; declarations/fields: `unlink_at` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 167–173 | No-follow bounded private filesystem primitives; declaration/member rmdir_at; declarations/fields: `rmdir_at` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 174–187 | No-follow bounded private filesystem primitives; declaration/member chown_path; declarations/fields: `chown_path` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 188–203 | No-follow bounded private filesystem primitives; declaration/member chmod_path; declarations/fields: `chmod_path` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 204–212 | No-follow bounded private filesystem primitives; declaration/member replace_at; declarations/fields: `replace_at` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 213–228 | No-follow bounded private filesystem primitives; declaration/member fsync_file; declarations/fields: `fsync_file` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 229–237 | No-follow bounded private filesystem primitives; declaration/member fstat_uid_mode; declarations/fields: `fstat_uid_mode` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 238–246 | No-follow bounded private filesystem primitives; declaration/member fstat_all; declarations/fields: `fstat_all` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 247–270 | No-follow bounded private filesystem primitives; declaration/member read_up_to; declarations/fields: `read_up_to` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 271–273 | No-follow bounded private filesystem primitives; declaration/member tests; declarations/fields: `tests` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 274–276 | No-follow bounded private filesystem primitives; declaration/member TEST_SEQ; declarations/fields: `TEST_SEQ` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 277–283 | No-follow bounded private filesystem primitives; declaration/member test_dir; declarations/fields: `test_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 284–287 | No-follow bounded private filesystem primitives; declaration/member dir_fd; declarations/fields: `dir_fd` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 288–292 | No-follow bounded private filesystem primitives; declaration/member am_root; declarations/fields: `am_root` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 293–307 | No-follow bounded private filesystem primitives; declaration/member root_chain_matrix; declarations/fields: `root_chain_matrix` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 308–316 | No-follow bounded private filesystem primitives; declaration/member fake_stat; declarations/fields: `fake_stat` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 317–333 | No-follow bounded private filesystem primitives; declaration/member safety_predicate_matrix; declarations/fields: `safety_predicate_matrix` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 334–376 | No-follow bounded private filesystem primitives; declaration/member root_file_failures; declarations/fields: `root_file_failures` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 377–411 | No-follow bounded private filesystem primitives; declaration/member record_size_and_json_limits; declarations/fields: `record_size_and_json_limits` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 412–427 | No-follow bounded private filesystem primitives; declaration/member read_record_public_paths; declarations/fields: `read_record_public_paths` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 428–450 | No-follow bounded private filesystem primitives; declaration/member new_file_create_and_collision; declarations/fields: `new_file_create_and_collision` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 451–488 | No-follow bounded private filesystem primitives; declaration/member wrapper_matrix; declarations/fields: `wrapper_matrix` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 489–517 | No-follow bounded private filesystem primitives; declaration/member path_chown_chmod_matrix; declarations/fields: `path_chown_chmod_matrix` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 518–534 | No-follow bounded private filesystem primitives; declaration/member stat_all_and_single_read; declarations/fields: `stat_all_and_single_read` |

<a id="coverage-2f067f5acd60"></a>

<a id="rustsoda-project-terminalsrckeysrs-1"></a>

## [rust/soda-project-terminal/src/keys.rs](../../../../../rust/soda-project-terminal/src/keys.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1–32, 494–520 | Explicit locked managed authorized-key inspection/update with revision fencing; declarations/fields: `STDIN_LIMIT`, `KEY_FILE_LIMIT`, `KEY_COUNT_LIMIT`, `is_word`, `word_dash_plus`, `sk_plus`, `canonical_key_ok`, `canonical_lines`, `json_truthy`, `reject_duplicates`, `KeyRequest`, `decode_key_request`, `state_object`, `keys_directory`, `FileId`, `file_id`, `read_keys`, `candidate_name`, `update`, `replace_inner`, `KeyFail`, `read_stdin`, `write_stdout_all`, `write_stderr_all`, `key_main`, `key_inner`, `tests`, `KEY_A`, `KEY_B`, `KEY_SK`, `canonical_lines_matrix` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 33–34 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member STDIN_LIMIT; declarations/fields: `STDIN_LIMIT` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 35–36 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KEY_FILE_LIMIT; declarations/fields: `KEY_FILE_LIMIT` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 37–38 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KEY_COUNT_LIMIT; declarations/fields: `KEY_COUNT_LIMIT` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 39–43 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member is_word; declarations/fields: `is_word` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 44–48 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member word_dash_plus; declarations/fields: `word_dash_plus` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 49–59 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member sk_plus; declarations/fields: `sk_plus` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 60–83 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member canonical_key_ok; declarations/fields: `canonical_key_ok` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 84–112 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member canonical_lines; declarations/fields: `canonical_lines` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 113–130 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member json_truthy; declarations/fields: `json_truthy` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 131–158 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member reject_duplicates; declarations/fields: `reject_duplicates` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 159 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest; declarations/fields: `KeyRequest` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 160 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest.login; declarations/fields: `KeyRequest.login` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 161 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest.identity; declarations/fields: `KeyRequest.identity` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 162 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest.apply; declarations/fields: `KeyRequest.apply` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 163 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest.revision; declarations/fields: `KeyRequest.revision` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 164 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest.keys; declarations/fields: `KeyRequest.keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 165–168 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyRequest.desired; declarations/fields: `KeyRequest.desired` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 169–227 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member decode_key_request; declarations/fields: `decode_key_request` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 228–239 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member state_object; declarations/fields: `state_object` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 240–256 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member keys_directory; declarations/fields: `keys_directory` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 257 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member FileId; declarations/fields: `FileId` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 258 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member FileId.dev; declarations/fields: `FileId.dev` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 259 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member FileId.ino; declarations/fields: `FileId.ino` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 260 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member FileId.mtime_sec; declarations/fields: `FileId.mtime_sec` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 261–263 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member FileId.mtime_nsec; declarations/fields: `FileId.mtime_nsec` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 264–275 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member file_id; declarations/fields: `file_id` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 276–293 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member read_keys; declarations/fields: `read_keys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 294–310 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member candidate_name; declarations/fields: `candidate_name` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 311–353 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member update; declarations/fields: `update` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 354–380 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member replace_inner; declarations/fields: `replace_inner` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 381–386 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KeyFail; declarations/fields: `KeyFail` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 387–396 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member read_stdin; declarations/fields: `read_stdin` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 397–406 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member write_stdout_all; declarations/fields: `write_stdout_all` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 407–424 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member write_stderr_all; declarations/fields: `write_stderr_all` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 425–435 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member key_main; declarations/fields: `key_main` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 436–443 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member key_inner; declarations/fields: `key_inner` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 444–446 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member tests; declarations/fields: `tests` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 447 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KEY_A; declarations/fields: `KEY_A` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 448 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KEY_B; declarations/fields: `KEY_B` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 449–452 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member KEY_SK; declarations/fields: `KEY_SK` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 453–493 | Source assertion of Development SSH access; declarations/fields: `key_line_matrix` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 521–539 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member truthiness_matrix; declarations/fields: `truthiness_matrix` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 540–545 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member request; declarations/fields: `request` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 546–582 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member decode_matrix; declarations/fields: `decode_matrix` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 583–593 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member candidate_matrix; declarations/fields: `candidate_matrix` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 594–606 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member state_shape_exact; declarations/fields: `state_shape_exact` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 607–614 | Explicit locked managed authorized-key inspection/update with revision fencing; declaration/member update_refuses_unprivileged; declarations/fields: `update_refuses_unprivileged` |

<a id="coverage-79eceb11e8e7"></a>

## [rust/soda-project-terminal/src/main.rs](../../../../../rust/soda-project-terminal/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1–17 | One native helper argv/closed-response entrypoint |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 18 | Native human terminal account module; declarations/fields: `account` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 19 | Base64 module; declarations/fields: `b64` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 20 | Native subscription helper module; declarations/fields: `broker` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 21 | Filesystem module; declarations/fields: `fs` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 22 | Managed developer key module; declarations/fields: `keys` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 23 | Terminal scope module; declarations/fields: `proto` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 24 | PTY module; declarations/fields: `pty` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 25–26 | Wire/hash modules; declarations/fields: `pyemit`, `sha` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 27 | Native unit module; declarations/fields: `svc` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 28 | System primitives module; declarations/fields: `sys` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 29 | Managed terminal module; declarations/fields: `term` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 30–33 | Timestamp module; declarations/fields: `timex` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 34–57 | Native helper argv routing; declarations/fields: `Route`, `route` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 58–79 | Attachment output/closed-frame transport; declarations/fields: `write_stdout_all`, `is_epipe`, `emit_closed` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 80–136 | Managed terminal control operation; declarations/fields: `control_main` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 137–140 | Single helper dispatch; declarations/fields: `run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 141 | Subscription operation dispatch |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 142 | Developer key operation dispatch |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 143–152 | Managed terminal operation dispatch; declarations/fields: `main` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 153–196 | Argv routing source assertion; declarations/fields: `tests`, `argv`, `control10`, `route_matrix` |

<a id="coverage-f61ccc84403d"></a>

## [rust/soda-project-terminal/src/proto.rs](../../../../../rust/soda-project-terminal/src/proto.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–91 | Interactive dimensions/frame contracts; declarations/fields: `FRAME_LIMIT`, `QUEUE_LIMIT`, `HEARTBEAT_SECONDS`, `dimensions`, `ControlFrame`, `decode_frame`, `is_cc_or_cf` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 92–214 | Managed terminal names, control argv and scope contracts; declarations/fields: `valid_name`, `ControlArgs`, `parse_control_argv`, `valid_identifier`, `require_terminal_target`, `valid_scope`, `require_creation_scope`, `tests`, `argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 215–225, 268–314 | Terminal protocol source assertions; declarations/fields: `names`, `argv_matrix` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 226–267 | Source assertion of Interactive attachment; declarations/fields: `frames` |

<a id="coverage-649478827042"></a>

<a id="rustsoda-project-terminalsrcptyrs-1"></a>

## [rust/soda-project-terminal/src/pty.rs](../../../../../rust/soda-project-terminal/src/pty.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–23 | Native terminal PTY transport and bounded IO; declarations/fields: `STOPPED`, `stop_handler`, `install_stop_handlers`, `restore_handlers`, `set_nonblocking`, `is_eintr`, `read_fd`, `write_fd`, `write_all_blocking`, `ready_line`, `output_line`, `closed_line`, `tmux_attach_argv`, `cstring`, `set_size` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 24–25 | Native terminal PTY transport and bounded IO; declaration/member STOPPED; declarations/fields: `STOPPED` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 26–29 | Native terminal PTY transport and bounded IO; declaration/member stop_handler; declarations/fields: `stop_handler` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 30–46 | Native terminal PTY transport and bounded IO; declaration/member install_stop_handlers; declarations/fields: `install_stop_handlers` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 47–54 | Native terminal PTY transport and bounded IO; declaration/member restore_handlers; declarations/fields: `restore_handlers` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 55–70 | Native terminal PTY transport and bounded IO; declaration/member set_nonblocking; declarations/fields: `set_nonblocking` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 71–74 | Native terminal PTY transport and bounded IO; declaration/member is_eintr; declarations/fields: `is_eintr` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 75–88 | Native terminal PTY transport and bounded IO; declaration/member read_fd; declarations/fields: `read_fd` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 89–102 | Native terminal PTY transport and bounded IO; declaration/member write_fd; declarations/fields: `write_fd` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 103–114 | Native terminal PTY transport and bounded IO; declaration/member write_all_blocking; declarations/fields: `write_all_blocking` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 115–122 | Native terminal PTY transport and bounded IO; declaration/member ready_line; declarations/fields: `ready_line` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 123–136 | Native terminal PTY transport and bounded IO; declaration/member output_line; declarations/fields: `output_line` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 137–150 | Native terminal PTY transport and bounded IO; declaration/member closed_line; declarations/fields: `closed_line` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 151–163 | Native terminal PTY transport and bounded IO; declaration/member tmux_attach_argv; declarations/fields: `tmux_attach_argv` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 164–169 | Native terminal PTY transport and bounded IO; declaration/member cstring; declarations/fields: `cstring` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 170–184 | Native terminal PTY transport and bounded IO; declaration/member set_size; declarations/fields: `set_size` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 185–208 | Owned native child/PTY lifecycle; declarations/fields: `child_exited` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 209–241 | Owned native child/PTY lifecycle; declaration/member end_child; declarations/fields: `end_child` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 242–308 | Spawn admitted human login shell; declarations/fields: `spawn_login_pty` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 309–332 | Attachment IO/resize/heartbeat and bounded control; declarations/fields: `apply_control_frame`, `ingest_control_bytes`, `read_pty_output`, `session_should_stop`, `pty_select`, `take_control_input`, `flush_pty_queues`, `relay_pty_session`, `flush_closed`, `wait_readable`, `run_terminal`, `tests` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 333–360 | Attachment IO/resize/heartbeat and bounded control; declaration/member apply_control_frame; declarations/fields: `apply_control_frame` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 361–390 | Attachment IO/resize/heartbeat and bounded control; declaration/member ingest_control_bytes; declarations/fields: `ingest_control_bytes` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 391–416 | Attachment IO/resize/heartbeat and bounded control; declaration/member read_pty_output; declarations/fields: `read_pty_output` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 417–443 | Attachment IO/resize/heartbeat and bounded control; declaration/member session_should_stop; declarations/fields: `session_should_stop` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 444–518 | Attachment IO/resize/heartbeat and bounded control; declaration/member pty_select; declarations/fields: `pty_select` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 519–543 | Attachment IO/resize/heartbeat and bounded control; declaration/member take_control_input; declarations/fields: `take_control_input` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 544–561 | Attachment IO/resize/heartbeat and bounded control; declaration/member flush_pty_queues; declarations/fields: `flush_pty_queues` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 562–620 | Attachment IO/resize/heartbeat and bounded control; declaration/member relay_pty_session; declarations/fields: `relay_pty_session` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 621–655 | Attachment IO/resize/heartbeat and bounded control; declaration/member flush_closed; declarations/fields: `flush_closed` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 656–692 | Attachment IO/resize/heartbeat and bounded control; declaration/member wait_readable; declarations/fields: `wait_readable` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 693–757 | Attachment IO/resize/heartbeat and bounded control; declaration/member run_terminal; declarations/fields: `run_terminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 758–760 | Attachment IO/resize/heartbeat and bounded control; declaration/member tests; declarations/fields: `tests` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 761–771 | PTY source assertions; declarations/fields: `login` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 772–804 | PTY source assertions; declaration/member line_bytes_exact; declarations/fields: `line_bytes_exact` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 805–840 | PTY source assertions; declaration/member ingest_matrix; declarations/fields: `ingest_matrix` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 841–853 | PTY source assertions; declaration/member queue_backpressure; declarations/fields: `queue_backpressure` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 854–886 | PTY source assertions; declaration/member stop_matrix; declarations/fields: `stop_matrix` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 887–914 | PTY source assertions; declaration/member child_lifecycle; declarations/fields: `child_lifecycle` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 915–937 | PTY source assertions; declaration/member end_child_terms; declarations/fields: `end_child_terms` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 938–965 | PTY source assertions; declaration/member pty_output_lines; declarations/fields: `pty_output_lines` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 966–992 | PTY source assertions; declaration/member resize_needs_tty; declarations/fields: `resize_needs_tty` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 993–1002 | PTY source assertions; declaration/member bounds_rejected; declarations/fields: `bounds_rejected` |

<a id="coverage-33d09196cd88"></a>

<a id="rustsoda-project-terminalsrcsvcrs-1"></a>

## [rust/soda-project-terminal/src/svc.rs](../../../../../rust/soda-project-terminal/src/svc.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1–14 | Managed systemd session units and native invocation admission; declarations/fields: `s_issock`, `unit_name`, `service_properties`, `systemctl_show_argv`, `systemctl_stop_argv`, `invocation_show_argv`, `stat_fs_argv`, `infocmp_argv`, `tmux_argv`, `parse_service_fields`, `verify_loaded_unit`, `service_state`, `stop_service`, `cstring`, `wait_status_timeout`, `exit_ok` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 15–18 | Managed systemd session units and native invocation admission; declaration/member s_issock; declarations/fields: `s_issock` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 19–22 | Managed systemd session units and native invocation admission; declaration/member unit_name; declarations/fields: `unit_name` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 23–27 | Managed systemd session units and native invocation admission; declaration/member service_properties; declarations/fields: `service_properties` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 28–37 | Managed systemd session units and native invocation admission; declaration/member systemctl_show_argv; declarations/fields: `systemctl_show_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 38–46 | Managed systemd session units and native invocation admission; declaration/member systemctl_stop_argv; declarations/fields: `systemctl_stop_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 47–57 | Managed systemd session units and native invocation admission; declaration/member invocation_show_argv; declarations/fields: `invocation_show_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 58–68 | Managed systemd session units and native invocation admission; declaration/member stat_fs_argv; declarations/fields: `stat_fs_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 69–73 | Managed systemd session units and native invocation admission; declaration/member infocmp_argv; declarations/fields: `infocmp_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 74–86 | Managed systemd session units and native invocation admission; declaration/member tmux_argv; declarations/fields: `tmux_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 87–120 | Managed systemd session units and native invocation admission; declaration/member parse_service_fields; declarations/fields: `parse_service_fields` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 121–154 | Managed systemd session units and native invocation admission; declaration/member verify_loaded_unit; declarations/fields: `verify_loaded_unit` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 155–186 | Managed systemd session units and native invocation admission; declaration/member service_state; declarations/fields: `service_state` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 187–200 | Managed systemd session units and native invocation admission; declaration/member stop_service; declarations/fields: `stop_service` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 201–205 | Managed systemd session units and native invocation admission; declaration/member cstring; declarations/fields: `cstring` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 206–236 | Managed systemd session units and native invocation admission; declaration/member wait_status_timeout; declarations/fields: `wait_status_timeout` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 237–242 | Managed systemd session units and native invocation admission; declaration/member exit_ok; declarations/fields: `exit_ok` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 243–362 | Native protected read-only mount/file attestation used by subscription helper; declarations/fields: `run_stat_fs` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 363–372 | Native protected read-only mount/file attestation used by subscription helper; declaration/member fstatvfs_readonly; declarations/fields: `fstatvfs_readonly` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 373–392 | Native protected read-only mount/file attestation used by subscription helper; declaration/member cgroup_parent; declarations/fields: `cgroup_parent` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 393–415 | Native protected read-only mount/file attestation used by subscription helper; declaration/member parse_cgroup_populated; declarations/fields: `parse_cgroup_populated` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 416–458 | Native protected read-only mount/file attestation used by subscription helper; declaration/member cgroup_empty; declarations/fields: `cgroup_empty` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 459–463 | Native protected read-only mount/file attestation used by subscription helper; declaration/member SocketCheck; declarations/fields: `SocketCheck` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 464–586 | Managed terminal local socket/unit identity; declarations/fields: `socket_identity_kinded` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 587–597 | Managed terminal local socket/unit identity; declaration/member classify_connect_err; declarations/fields: `classify_connect_err` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 598–604 | Managed terminal local socket/unit identity; declaration/member socket_identity; declarations/fields: `socket_identity` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 605–664 | Native tmux control attachment; declarations/fields: `tmux_control` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 665–666, 753–792, 804–837 | Unit/session source assertions; declarations/fields: `tests`, `sample`, `show_fixture`, `service_fields_matrix`, `socket_negative_paths` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 667–670 | Unit/session source assertions; declaration/member tests; declarations/fields: `tests` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 671–680 | Unit/session source assertions; declaration/member sample; declarations/fields: `sample` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 681–690 | Unit/session source assertions; declaration/member show_fixture; declarations/fields: `show_fixture` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 691–752 | Source assertion of Provider execution integration; declarations/fields: `argv_constructors_exact` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 793–803 | Source assertion of Completion, revocation and reconciliation; declarations/fields: `cgroup_events_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 838–856 | Unit/session source assertions; declaration/member stat_fs_detects_tmpfs; declarations/fields: `stat_fs_detects_tmpfs` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 857–868 | Unit/session source assertions; declaration/member supervisor_smoke; declarations/fields: `supervisor_smoke` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 869–882 | Unit/session source assertions; declaration/member tmux_control_fails_closed; declarations/fields: `tmux_control_fails_closed` |

<a id="coverage-ad4c5aae1e49"></a>

<a id="rustsoda-project-terminalsrctermrs-1"></a>

## [rust/soda-project-terminal/src/term.rs](../../../../../rust/soda-project-terminal/src/term.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1–32 | Human managed-session scope, metadata, native systemd/tmux lifetime; declarations/fields: `TERMINALS`, `PROGRAM`, `s_isreg`, `s_issock`, `TMUX_CONFIG`, `terminal_path`, `checked_chain`, `list_dir_names`, `fstatat`, `record_exists`, `write_stdout_best_effort`, `closed_launch_failed`, `BindingRecord`, `Reservation`, `validate_binding`, `binding_matches_account`, `binding_record`, `validate_reservation`, `read_reservation`, `permit_live`, `UnitClass`, `classify_bare_state`, `status_object`, `status_state`, `parse_ready`, `writer_attached`, `observe_ready_terminal`, `classify_unit_state`, `terminal_status`, `terminal_directories`, `read_single_byte`, `stub_retire`, `remove_screen_contents`, `remove_owned_files`, `collect_finished`, `program_stat_ok`, `file_sha256_hex`, `verify_program`, `binding_object`, `reservation_object`, `write_name`, `reserve_terminal`, `systemd_run_argv`, `create_terminal`, `lifetime_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 33 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member TERMINALS; declarations/fields: `TERMINALS` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 34–35 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member PROGRAM; declarations/fields: `PROGRAM` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 36–39 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member s_isreg; declarations/fields: `s_isreg` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 40–43 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member s_issock; declarations/fields: `s_issock` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 44–54 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member TMUX_CONFIG; declarations/fields: `TMUX_CONFIG` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 55–63 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member terminal_path; declarations/fields: `terminal_path` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 64–78 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member checked_chain; declarations/fields: `checked_chain` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 79–94 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member list_dir_names; declarations/fields: `list_dir_names` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 95–112 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member fstatat; declarations/fields: `fstatat` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 113–120 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member record_exists; declarations/fields: `record_exists` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 121–137 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member write_stdout_best_effort; declarations/fields: `write_stdout_best_effort` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 138–146 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member closed_launch_failed; declarations/fields: `closed_launch_failed` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 147 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord; declarations/fields: `BindingRecord` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 148 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.login; declarations/fields: `BindingRecord.login` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 149 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.uid; declarations/fields: `BindingRecord.uid` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 150 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.gid; declarations/fields: `BindingRecord.gid` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 151 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.home; declarations/fields: `BindingRecord.home` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 152 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.shell; declarations/fields: `BindingRecord.shell` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 153 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.identity; declarations/fields: `BindingRecord.identity` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 154 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.cols; declarations/fields: `BindingRecord.cols` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 155 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.rows; declarations/fields: `BindingRecord.rows` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 156–159 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member BindingRecord.created_at; declarations/fields: `BindingRecord.created_at` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 160 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member Reservation; declarations/fields: `Reservation` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 161 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member Reservation.expires; declarations/fields: `Reservation.expires` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 162–168 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member Reservation.scope; declarations/fields: `Reservation.scope` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 169–235 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member validate_binding; declarations/fields: `validate_binding` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 236–244 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member binding_matches_account; declarations/fields: `binding_matches_account` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 245–267 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member binding_record; declarations/fields: `binding_record` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 268–296 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member validate_reservation; declarations/fields: `validate_reservation` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 297–303 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member read_reservation; declarations/fields: `read_reservation` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 304–314 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member permit_live; declarations/fields: `permit_live` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 315–323 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member UnitClass; declarations/fields: `UnitClass` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 324–348 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member classify_bare_state; declarations/fields: `classify_bare_state` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 349–369 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member status_object; declarations/fields: `status_object` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 370–377 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member status_state; declarations/fields: `status_state` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 378–399 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member parse_ready; declarations/fields: `parse_ready` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 400–405 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member writer_attached; declarations/fields: `writer_attached` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 406–416 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member observe_ready_terminal; declarations/fields: `observe_ready_terminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 417–439 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member classify_unit_state; declarations/fields: `classify_unit_state` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 440–489 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member terminal_status; declarations/fields: `terminal_status` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 490–510 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member terminal_directories; declarations/fields: `terminal_directories` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 511–528 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member read_single_byte; declarations/fields: `read_single_byte` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 529–551 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member stub_retire; declarations/fields: `stub_retire` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 552–570 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member remove_screen_contents; declarations/fields: `remove_screen_contents` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 571–612 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member remove_owned_files; declarations/fields: `remove_owned_files` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 613–657 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member collect_finished; declarations/fields: `collect_finished` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 658–663 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member program_stat_ok; declarations/fields: `program_stat_ok` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 664–689 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member file_sha256_hex; declarations/fields: `file_sha256_hex` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 690–713 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member verify_program; declarations/fields: `verify_program` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 714–744 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member binding_object; declarations/fields: `binding_object` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 745–754 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member reservation_object; declarations/fields: `reservation_object` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 755–787 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member write_name; declarations/fields: `write_name` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 788–868 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member reserve_terminal; declarations/fields: `reserve_terminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 869–923 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member systemd_run_argv; declarations/fields: `systemd_run_argv` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 924–967 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member create_terminal; declarations/fields: `create_terminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 968–976 | Human managed-session scope, metadata, native systemd/tmux lifetime; declaration/member lifetime_argv; declarations/fields: `lifetime_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 977–993 | Subscription native session lifetime/pin selection; declarations/fields: `subscription_lifetime` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 994–1012 | Subscription native session lifetime/pin selection; declaration/member subscription_command; declarations/fields: `subscription_command` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1013–1029 | Interactive attachment into admitted managed terminal; declarations/fields: `attach_terminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1030–1065 | Interactive attachment into admitted managed terminal; declaration/member attach_inner; declarations/fields: `attach_inner` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1066–1087 | Interactive attachment into admitted managed terminal; declaration/member list_owned_terminals; declarations/fields: `list_owned_terminals` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1088–1100 | Interactive attachment into admitted managed terminal; declaration/member mutate_terminal; declarations/fields: `mutate_terminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1101–1132 | Human terminal managed session inspection/control; declarations/fields: `control_terminal`, `tmux_new_session_args` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1133–1181 | Human terminal managed session inspection/control; declaration/member control_terminal; declarations/fields: `control_terminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1182–1203 | Human terminal managed session inspection/control; declaration/member tmux_new_session_args; declarations/fields: `tmux_new_session_args` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1204–1242 | Execution-aware subscription session branch; declarations/fields: `subscription_session` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1243–1258 | Execution-aware subscription session branch; declaration/member ready_object; declarations/fields: `ready_object` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1259–1268 | Managed terminal preparation entrypoint; declarations/fields: `prepare` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1269–1359 | Managed terminal preparation entrypoint; declaration/member prepare_inner; declarations/fields: `prepare_inner` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1360–1364, 1693–1707 | Human terminal source assertions; declarations/fields: `tests`, `sample`, `binding_doc`, `good_account`, `parse`, `path_matrix`, `binding_matrix`, `account_match_matrix`, `reservation_matrix`, `missing_reservation_is_none`, `classify_matrix`, `systemd_argv_exact`, `tmux_session_argv_exact`, `lifetime_rule` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1365–1367 | Human terminal source assertions; declaration/member tests; declarations/fields: `tests` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1368–1377 | Human terminal source assertions; declaration/member sample; declarations/fields: `sample` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1378–1383 | Human terminal source assertions; declaration/member binding_doc; declarations/fields: `binding_doc` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1384–1387 | Human terminal source assertions; declaration/member good_account; declarations/fields: `good_account` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1388–1392 | Human terminal source assertions; declaration/member parse; declarations/fields: `parse` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1393–1405 | Human terminal source assertions; declaration/member path_matrix; declarations/fields: `path_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1406–1484 | Human terminal source assertions; declaration/member binding_matrix; declarations/fields: `binding_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1485–1504 | Human terminal source assertions; declaration/member account_match_matrix; declarations/fields: `account_match_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1505–1527 | Human terminal source assertions; declaration/member reservation_matrix; declarations/fields: `reservation_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1528–1538 | Human terminal source assertions; declaration/member missing_reservation_is_none; declarations/fields: `missing_reservation_is_none` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1539–1579 | Human terminal source assertions; declaration/member classify_matrix; declarations/fields: `classify_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1580–1642 | Human terminal source assertions; declaration/member systemd_argv_exact; declarations/fields: `systemd_argv_exact` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1643–1673 | Human terminal source assertions; declaration/member tmux_session_argv_exact; declarations/fields: `tmux_session_argv_exact` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1674–1692 | Source assertion of Provider execution integration; declarations/fields: `subscription_command_exact` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1708–1737 | Human terminal source assertions; declaration/member emission_bytes_exact; declarations/fields: `emission_bytes_exact` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1738–1762 | Human terminal source assertions; declaration/member program_hash_streams_past_64k; declarations/fields: `program_hash_streams_past_64k` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1763–1786 | Human terminal source assertions; declaration/member ready_matrix; declarations/fields: `ready_matrix` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1787–1814 | Human terminal source assertions; declaration/member entry_point_smoke; declarations/fields: `entry_point_smoke` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1815–1821 | Human terminal source assertions; declaration/member tmux_config_exact; declarations/fields: `tmux_config_exact` |

<a id="coverage-1f5242f69516"></a>

<a id="rustsoda-project-terminalsrctimexrs-1"></a>

## [rust/soda-project-terminal/src/timex.rs](../../../../../rust/soda-project-terminal/src/timex.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–32 | Go/Python-compatible deadline and timestamp parsing; declarations/fields: `now_secs`, `days_from_civil`, `is_leap_year`, `days_in_month`, `two_digits`, `parse_date`, `parse_time_offset`, `parse_offset`, `parse_iso_deadline`, `tests`, `vectors_match_cpython`, `rejections_match_cpython`, `documented_deviations`, `civil_math_spot_checks`, `now_secs_matches_system` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 33–40 | Go/Python-compatible deadline and timestamp parsing; declaration/member now_secs; declarations/fields: `now_secs` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 41–50 | Go/Python-compatible deadline and timestamp parsing; declaration/member days_from_civil; declarations/fields: `days_from_civil` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 51–54 | Go/Python-compatible deadline and timestamp parsing; declaration/member is_leap_year; declarations/fields: `is_leap_year` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 55–64 | Go/Python-compatible deadline and timestamp parsing; declaration/member days_in_month; declarations/fields: `days_in_month` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 65–72 | Go/Python-compatible deadline and timestamp parsing; declaration/member two_digits; declarations/fields: `two_digits` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 73–108 | Go/Python-compatible deadline and timestamp parsing; declaration/member parse_date; declarations/fields: `parse_date` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 109–164 | Go/Python-compatible deadline and timestamp parsing; declaration/member parse_time_offset; declarations/fields: `parse_time_offset` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 165–217 | Go/Python-compatible deadline and timestamp parsing; declaration/member parse_offset; declarations/fields: `parse_offset` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 218–235 | Go/Python-compatible deadline and timestamp parsing; declaration/member parse_iso_deadline; declarations/fields: `parse_iso_deadline` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 236–239 | Go/Python-compatible deadline and timestamp parsing; declaration/member tests; declarations/fields: `tests` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 240–329 | Go/Python-compatible deadline and timestamp parsing; declaration/member vectors_match_cpython; declarations/fields: `vectors_match_cpython` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 330–366 | Go/Python-compatible deadline and timestamp parsing; declaration/member rejections_match_cpython; declarations/fields: `rejections_match_cpython` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 367–379 | Go/Python-compatible deadline and timestamp parsing; declaration/member documented_deviations; declarations/fields: `documented_deviations` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 380–395 | Go/Python-compatible deadline and timestamp parsing; declaration/member civil_math_spot_checks; declarations/fields: `civil_math_spot_checks` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 396–403 | Go/Python-compatible deadline and timestamp parsing; declaration/member now_secs_matches_system; declarations/fields: `now_secs_matches_system` |

<a id="coverage-fc3bf1566148"></a>

## [rust/soda-project-terminal/tests/cli.rs](../../../../../rust/soda-project-terminal/tests/cli.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1–134 | One Project-native terminal helper with managed sessions, keys and subscription operations; declarations/fields: `bin_path`, `run_stdin`, `CLOSED`, `prepare_bogus_id_closed`, `prepare_missing_locator_closed`, `bad_argv_closed`, `control_bad_identifier_closed`, `KEYS_ERR`, `keys_invalid_stdin`, `keys_duplicate_field_rejected`, `keys_wrong_shape_rejected`, `keys_valid_shape_unconfirmed_without_identity`, `BROKER_ERR` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 135–168 | Source assertion of Provider execution integration; declarations/fields: `broker_invalid_stdin`, `broker_oversize_rejected`, `broker_unknown_action_without_delivery`, `broker_missing_action_rejected` |


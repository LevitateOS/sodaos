# Soda muse

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-4288d87834c0"></a>

<a id="rustsoda-musesrcmainrs-1"></a>

## [rust/soda-muse/src/main.rs](../../../../../rust/soda-muse/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–10 | Native Muse Project CLI and explicit launch/exec/control entrypoint; declarations/fields: `MUSE_LAUNCH_SOCKET`, `MUSE_NATIVE`, `main`, `run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 11 | Native Muse Project CLI and explicit launch/exec/control entrypoint; declaration/member MUSE_LAUNCH_SOCKET; declarations/fields: `MUSE_LAUNCH_SOCKET` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 12–13 | Native Muse Project CLI and explicit launch/exec/control entrypoint; declaration/member MUSE_NATIVE; declarations/fields: `MUSE_NATIVE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 14–21 | Native Muse Project CLI and explicit launch/exec/control entrypoint; declaration/member main; declarations/fields: `main` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 22–41 | Native Muse Project CLI and explicit launch/exec/control entrypoint; declaration/member run; declarations/fields: `run` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 42–63 | Connection metadata selection action; declarations/fields: `native_action` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 64–79 | Connection metadata selection action; declaration/member metadata_action; declarations/fields: `metadata_action` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 80 | Allowed native invocation request; declarations/fields: `ShellRequest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 81 | Allowed native invocation request; declaration/member ShellRequest.cwd; declarations/fields: `ShellRequest.cwd` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 82 | Allowed native invocation request; declaration/member ShellRequest.args; declarations/fields: `ShellRequest.args` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 83 | Allowed native invocation request; declaration/member ShellRequest.home; declarations/fields: `ShellRequest.home` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 84 | Allowed native invocation request; declaration/member ShellRequest.connection_id; declarations/fields: `ShellRequest.connection_id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 85 | Allowed native invocation request; declaration/member ShellRequest.config_home; declarations/fields: `ShellRequest.config_home` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 86 | Allowed native invocation request; declaration/member ShellRequest.term; declarations/fields: `ShellRequest.term` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 87 | Allowed native invocation request; declaration/member ShellRequest.tty; declarations/fields: `ShellRequest.tty` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 88 | Allowed native invocation request; declaration/member ShellRequest.cols; declarations/fields: `ShellRequest.cols` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 89–91 | Allowed native invocation request; declaration/member ShellRequest.rows; declarations/fields: `ShellRequest.rows` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 92–130 | Allowed native invocation request; declaration/member shell_request; declarations/fields: `shell_request` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 131–136 | Allowed native invocation request; declaration/member env_lossy; declarations/fields: `env_lossy` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 137–151 | Allowed native invocation request; declaration/member validate_shell; declarations/fields: `validate_shell` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 152–155 | Allowed native invocation request; declaration/member config_paths_valid; declarations/fields: `config_paths_valid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 156–160 | Allowed native invocation request; declaration/member launch_sizes_valid; declarations/fields: `launch_sizes_valid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 161–167 | Allowed native invocation request; declaration/member launch_absolute_path; declarations/fields: `launch_absolute_path` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 168–171 | Allowed native invocation request; declaration/member launch_text; declarations/fields: `launch_text` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 172–187 | Allowed native invocation request; declaration/member launch_arguments_valid; declarations/fields: `launch_arguments_valid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 188–241 | Allowed native invocation request; declaration/member shell_request_json; declarations/fields: `shell_request_json` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 242–271 | Bounded public launch socket descriptor exchange; declarations/fields: `seqpacket_connect` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 272–300 | Bounded public launch socket descriptor exchange; declaration/member send_with_fds; declarations/fields: `send_with_fds` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 301 | Explicit native launch/control and execution integration; declarations/fields: `launch_shell` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 302–303, 361–362 | Explicit native launch/control and execution integration; declaration/member Guard; declarations/fields: `Guard` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 304–327, 363–405 | Explicit native launch/control and execution integration; declaration/member drop; declarations/fields: `drop` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 328–345 | Explicit native launch/control and execution integration; declaration/member NEVER; declarations/fields: `NEVER` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 346–359 | Explicit native launch/control and execution integration; declaration/member forward_signals; declarations/fields: `forward_signals` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 360 | Explicit native launch/control and execution integration; declaration/member controls_loop; declarations/fields: `controls_loop` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 406–471 | Explicit native launch/control and execution integration; declaration/member execute; declarations/fields: `execute` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 472–487 | Explicit native launch/control and execution integration; declaration/member await_admission; declarations/fields: `await_admission` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 488–503 | Execution live deadline/fence observation; declarations/fields: `execution_state` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 504–514 | Execution live deadline/fence observation; declaration/member mkdir_mode; declarations/fields: `mkdir_mode` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 515–518 | Execution live deadline/fence observation; declaration/member muse_environment; declarations/fields: `muse_environment` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 519–522 | Execution live deadline/fence observation; declaration/member env_lossy_opt; declarations/fields: `env_lossy_opt` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 523–547 | Execution live deadline/fence observation; declaration/member muse_environment_with; declarations/fields: `muse_environment_with` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 548–562 | Execution live deadline/fence observation; declaration/member go_base; declarations/fields: `go_base` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 563–594 | Execution live deadline/fence observation; declaration/member go_clean; declarations/fields: `go_clean` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 595–599 | Execution live deadline/fence observation; declaration/member go_join; declarations/fields: `go_join` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 600–603 | Execution live deadline/fence observation; declaration/member errno_str; declarations/fields: `errno_str` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 604–621 | Execution live deadline/fence observation; declaration/member go_strerror; declarations/fields: `go_strerror` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 622–631 | Execution live deadline/fence observation; declaration/member go_quote_rune; declarations/fields: `go_quote_rune` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 632–660 | Native account and execution scope attestation; declarations/fields: `account_for` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 661–664 | Native account and execution scope attestation; declaration/member path_error; declarations/fields: `path_error` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 665–691 | Native account and execution scope attestation; declaration/member account_entry; declarations/fields: `account_entry` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 692–708 | Native account and execution scope attestation; declaration/member account_node; declarations/fields: `account_node` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 709–734 | Native account and execution scope attestation; declaration/member lookup_user; declarations/fields: `lookup_user` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 735–736 | Native account and execution scope attestation; declaration/member check_runtime; declarations/fields: `check_runtime` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 737–738 | Native account and execution scope attestation; declaration/member Cleanup; declarations/fields: `Cleanup` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 739–756 | Native account and execution scope attestation; declaration/member drop; declarations/fields: `drop` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 757–778 | Native account and execution scope attestation; declaration/member make_private_dir; declarations/fields: `make_private_dir` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 779–831 | Native account and execution scope attestation; declaration/member run_pristine; declarations/fields: `run_pristine` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 832–843 | Bound private credential/config files; declarations/fields: `copy_config` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 844–880 | Bound private credential/config files; declaration/member copy_config_walk; declarations/fields: `copy_config_walk` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 881–894 | Bound private credential/config files; declaration/member copy_config_children; declarations/fields: `copy_config_children` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 895–916 | Bound private credential/config files; declaration/member copy_config_file; declarations/fields: `copy_config_file` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 917–954 | Bound private credential/config files; declaration/member read_config; declarations/fields: `read_config` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 955 | Strict request/result wire decoding and encoding; declarations/fields: `base64_encode` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 956–980 | Strict request/result wire decoding and encoding; declaration/member ALPHABET; declarations/fields: `ALPHABET` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 981–1007 | Strict request/result wire decoding and encoding; declaration/member json_string; declarations/fields: `json_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1008–1079 | Strict request/result wire decoding and encoding; declaration/member parse_launch_exit; declarations/fields: `parse_launch_exit` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1080–1131 | Strict request/result wire decoding and encoding; declaration/member parse_json_string; declarations/fields: `parse_json_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1132–1154 | Strict request/result wire decoding and encoding; declaration/member parse_json_integer; declarations/fields: `parse_json_integer` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1155–1234 | Strict request/result wire decoding and encoding; declaration/member skip_json_value; declarations/fields: `skip_json_value` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1235–1238 | Strict request/result wire decoding and encoding; declaration/member tests; declarations/fields: `tests` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1239–1253 | Native Muse CLI source assertions; declarations/fields: `shell_fixture` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1254–1276 | Native Muse CLI source assertions; declaration/member shell_request_wire_matches_go; declarations/fields: `shell_request_wire_matches_go` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1277–1301 | Native Muse CLI source assertions; declaration/member shell_validation_matches_go; declarations/fields: `shell_validation_matches_go` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1302–1331 | Native Muse CLI source assertions; declaration/member launch_exit_parsing_matches_go_unmarshal; declarations/fields: `launch_exit_parsing_matches_go_unmarshal` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1332–1347 | Native Muse CLI source assertions; declaration/member base64_matches_standard_vectors; declarations/fields: `base64_matches_standard_vectors` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1348–1366 | Native Muse CLI source assertions; declaration/member go_clean_matches_filepath_cases; declarations/fields: `go_clean_matches_filepath_cases` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1367–1373 | Native Muse CLI source assertions; declaration/member go_quote_matches_invalid_byte_cases; declarations/fields: `go_quote_matches_invalid_byte_cases` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1374–1381 | Native Muse CLI source assertions; declaration/member execution_id_rules_match_go; declarations/fields: `execution_id_rules_match_go` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1382–1404 | Native Muse CLI source assertions; declaration/member muse_environment_orders_fixed_then_passthrough; declarations/fields: `muse_environment_orders_fixed_then_passthrough` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1405–1445 | Native Muse CLI source assertions; declaration/member copy_config_skips_credentials_and_copies_tree; declarations/fields: `copy_config_skips_credentials_and_copies_tree` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1446–1456 | Native Muse CLI source assertions; declaration/member read_config_contract_shapes; declarations/fields: `read_config_contract_shapes` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1457–1469 | Native Muse CLI source assertions; declaration/member account_markers_reject_unsafe_nodes; declarations/fields: `account_markers_reject_unsafe_nodes` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1470–1484 | Native Muse CLI source assertions; declaration/member launch_shell_passes_fds_and_maps_exit; declarations/fields: `launch_shell_passes_fds_and_maps_exit` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1485–1486 | Native Muse CLI source assertions; declaration/member Guard; declarations/fields: `Guard` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1487–1559 | Native Muse CLI source assertions; declaration/member drop; declarations/fields: `drop` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1560–1618 | Native Muse CLI source assertions; declaration/member controls_forward_signal_number; declarations/fields: `controls_forward_signal_number` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1619–1637 | Native Muse CLI source assertions; declaration/member native_dispatch_errors_match_go; declarations/fields: `native_dispatch_errors_match_go` |


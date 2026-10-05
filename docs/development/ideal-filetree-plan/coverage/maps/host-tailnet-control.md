# Host tailnet control

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-db67f334b66a"></a>

## [rust/soda-host/src/tcontrol.rs](../../../../../rust/soda-host/src/tcontrol.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–144 | Tailnet control composition and native host options; declarations/fields: `Options`, `default`, `Control`, `new`, `new_project_control`, `local_round_trip`, `transport`, `provider` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 145–162 | Protected operator credential/policy admission; declarations/fields: `check_credential`, `lock_policy` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 163–167 | Native host observation; declarations/fields: `observe` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 168–182 | Standing enrollment policy settings; declarations/fields: `settings` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 183–196 | Project selection options; declarations/fields: `options` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 197–256 | Explicit host sign-in/sign-out/preferences; declarations/fields: `host_action` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 257–267 | Operator enrollment policy operations; declarations/fields: `enrollment` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 268–278 | Project selection actions; declarations/fields: `project` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 279–305 | Exact companion run admission and key handoff; declarations/fields: `run_binding`, `enroll_run` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 306–352 | Native Git endpoint status advertisement; declarations/fields: `cli_status`, `cli_endpoint`, `project`, `run_binding`, `enroll_run`, `capped` |

<a id="coverage-53259cb2f4f9"></a>

## [rust/soda-host/src/tcontrol_enroll.rs](../../../../../rust/soda-host/src/tcontrol_enroll.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–151 | Restricted provider credential and key interface; declarations/fields: `Provider`, `token`, `key`, `check_credential`, `project_key` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 152–214 | Companion run authority admission and issued key handoff; declarations/fields: `admit_enroll_run`, `enroll_run` |

<a id="coverage-b3f689740403"></a>

## [rust/soda-host/src/tcontrol_native.rs](../../../../../rust/soda-host/src/tcontrol_native.rs)

 Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–19 | Bounded private local HTTP transport; declarations/fields: `HOST_SOCKET`, `DEFAULT_CLI`, `DEFAULT_LIBEXEC`, `RESPONSE_LIMIT`, `LOCAL_TIMEOUT`, `UP_TIMEOUT`, `finish_local`, `local_request` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 20 | Bounded private local HTTP transport; declaration/member HOST_SOCKET; declarations/fields: `HOST_SOCKET` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 21 | Bounded private local HTTP transport; declaration/member DEFAULT_CLI; declarations/fields: `DEFAULT_CLI` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 22–23 | Bounded private local HTTP transport; declaration/member DEFAULT_LIBEXEC; declarations/fields: `DEFAULT_LIBEXEC` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 24–25 | Bounded private local HTTP transport; declaration/member RESPONSE_LIMIT; declarations/fields: `RESPONSE_LIMIT` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 26–27 | Bounded private local HTTP transport; declaration/member LOCAL_TIMEOUT; declarations/fields: `LOCAL_TIMEOUT` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 28–32 | Bounded private local HTTP transport; declaration/member UP_TIMEOUT; declarations/fields: `UP_TIMEOUT` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 33–37 | Bounded private local HTTP transport; declaration/member Transport; declarations/fields: `Transport` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 38–41 | Bounded private local HTTP transport; declaration/member RoundTrip; declarations/fields: `RoundTrip` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 42–49 | Bounded private local HTTP transport; declaration/member finish_local; declarations/fields: `finish_local` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 50–110 | Bounded private local HTTP transport; declaration/member local_request; declarations/fields: `local_request` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 111–117 | Native Tailscale status JSON binders; declarations/fields: `fold_char`, `fold_eq`, `native_object`, `bound_values`, `bind_string_into`, `bind_bool_into`, `bind_list_into`, `NativePeer`, `bind_native_peer_into`, `NativeStatus`, `decode_native_status`, `NativePrefs`, `decode_native_prefs` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 118–126 | Native Tailscale status JSON binders; declaration/member fold_char; declarations/fields: `fold_char` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 127–147 | Native Tailscale status JSON binders; declaration/member fold_eq; declarations/fields: `fold_eq` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 148–181 | Native Tailscale status JSON binders; declaration/member native_object; declarations/fields: `native_object` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 182–189 | Native Tailscale status JSON binders; declaration/member bound_values; declarations/fields: `bound_values` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 190–204 | Native Tailscale status JSON binders; declaration/member bind_string_into; declarations/fields: `bind_string_into` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 205–215 | Native Tailscale status JSON binders; declaration/member bind_bool_into; declarations/fields: `bind_bool_into` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 216–241 | Native Tailscale status JSON binders; declaration/member bind_list_into; declarations/fields: `bind_list_into` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 242 | Native Tailscale status JSON binders; declaration/member NativePeer; declarations/fields: `NativePeer` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 243 | Native Tailscale status JSON binders; declaration/member NativePeer.id; declarations/fields: `NativePeer.id` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 244 | Native Tailscale status JSON binders; declaration/member NativePeer.dns_name; declarations/fields: `NativePeer.dns_name` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 245 | Native Tailscale status JSON binders; declaration/member NativePeer.ips; declarations/fields: `NativePeer.ips` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 246 | Native Tailscale status JSON binders; declaration/member NativePeer.online; declarations/fields: `NativePeer.online` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 247 | Native Tailscale status JSON binders; declaration/member NativePeer.exit_node_option; declarations/fields: `NativePeer.exit_node_option` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 248–250 | Native Tailscale status JSON binders; declaration/member NativePeer.expired; declarations/fields: `NativePeer.expired` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 251–261 | Native Tailscale status JSON binders; declaration/member bind_native_peer_into; declarations/fields: `bind_native_peer_into` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 262 | Native Tailscale status JSON binders; declaration/member NativeStatus; declarations/fields: `NativeStatus` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 263 | Native Tailscale status JSON binders; declaration/member NativeStatus.backend_state; declarations/fields: `NativeStatus.backend_state` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 264 | Native Tailscale status JSON binders; declaration/member NativeStatus.have_node_key; declarations/fields: `NativeStatus.have_node_key` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 265 | Native Tailscale status JSON binders; declaration/member NativeStatus.tailnet_name; declarations/fields: `NativeStatus.tailnet_name` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 266 | Native Tailscale status JSON binders; declaration/member NativeStatus.tailnet_magic_dns; declarations/fields: `NativeStatus.tailnet_magic_dns` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 267 | Native Tailscale status JSON binders; declaration/member NativeStatus.has_tailnet; declarations/fields: `NativeStatus.has_tailnet` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 268 | Native Tailscale status JSON binders; declaration/member NativeStatus.self_peer; declarations/fields: `NativeStatus.self_peer` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 269 | Native Tailscale status JSON binders; declaration/member NativeStatus.peers; declarations/fields: `NativeStatus.peers` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 270 | Native Tailscale status JSON binders; declaration/member NativeStatus.health_count; declarations/fields: `NativeStatus.health_count` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 271–273 | Native Tailscale status JSON binders; declaration/member NativeStatus.auth_url; declarations/fields: `NativeStatus.auth_url` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 274–349 | Native Tailscale status JSON binders; declaration/member decode_native_status; declarations/fields: `decode_native_status` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 350 | Native Tailscale status JSON binders; declaration/member NativePrefs; declarations/fields: `NativePrefs` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 351 | Native Tailscale status JSON binders; declaration/member NativePrefs.want_running; declarations/fields: `NativePrefs.want_running` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 352 | Native Tailscale status JSON binders; declaration/member NativePrefs.exit_node_id; declarations/fields: `NativePrefs.exit_node_id` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 353 | Native Tailscale status JSON binders; declaration/member NativePrefs.exit_node_ip; declarations/fields: `NativePrefs.exit_node_ip` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 354 | Native Tailscale status JSON binders; declaration/member NativePrefs.allow_lan; declarations/fields: `NativePrefs.allow_lan` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 355–357 | Native Tailscale status JSON binders; declaration/member NativePrefs.advertise_routes; declarations/fields: `NativePrefs.advertise_routes` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 358–394 | Native Tailscale status JSON binders; declaration/member decode_native_prefs; declarations/fields: `decode_native_prefs` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 395–396 | Host interface/session observation and derived preferences; declarations/fields: `round_trip`, `fetch_native_status`, `fetch_native_prefs`, `populate_host_preferences`, `compute_host_revision`, `observe`, `run_command`, `decode_up_notifications`, `split_json_values`, `socket_arg` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 397–407 | Host interface/session observation and derived preferences; declaration/member round_trip; declarations/fields: `round_trip` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 408–428 | Host interface/session observation and derived preferences; declaration/member fetch_native_status; declarations/fields: `fetch_native_status` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 429–443 | Host interface/session observation and derived preferences; declaration/member fetch_native_prefs; declarations/fields: `fetch_native_prefs` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 444–467 | Host interface/session observation and derived preferences; declaration/member populate_host_preferences; declarations/fields: `populate_host_preferences` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 468–508 | Host interface/session observation and derived preferences; declaration/member compute_host_revision; declarations/fields: `compute_host_revision` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 509–564 | Host interface/session observation and derived preferences; declaration/member observe; declarations/fields: `observe` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 565–578 | Host interface/session observation and derived preferences; declaration/member run_command; declarations/fields: `run_command` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 579–611 | Host interface/session observation and derived preferences; declaration/member decode_up_notifications; declarations/fields: `decode_up_notifications` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 612–698 | Host interface/session observation and derived preferences; declaration/member split_json_values; declarations/fields: `split_json_values` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 699–702 | Host interface/session observation and derived preferences; declaration/member socket_arg; declarations/fields: `socket_arg` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 703–735 | Native sign-in/logout/exit-node actions; declarations/fields: `execute_signin` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 736–740 | Native sign-in/logout/exit-node actions; declaration/member execute_logout; declarations/fields: `execute_logout` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 741–753 | Native sign-in/logout/exit-node actions; declaration/member find_available_exit_node; declarations/fields: `find_available_exit_node` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 754–777 | Native sign-in/logout/exit-node actions; declaration/member execute_exit_node; declarations/fields: `execute_exit_node` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 778–791 | Git endpoint advertisement and Forgejo refresh; declarations/fields: `execute_advertise_exit_node` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 792–805 | Git endpoint advertisement and Forgejo refresh; declaration/member execute_refresh_forgejo; declarations/fields: `execute_refresh_forgejo` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 806–818 | Confirmed host request readback; declarations/fields: `verify_exit_node` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 819–836 | Confirmed host request readback; declaration/member verify_host_action_outcome; declarations/fields: `verify_host_action_outcome` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 837–870 | Confirmed host request readback; declaration/member readback_host_action; declarations/fields: `readback_host_action` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 871–878 | Native endpoint client and advertisement reads; declarations/fields: `CliStatus`, `CliEndpoint`, `last_field`, `tolerant_string`, `tolerant_bool`, `tolerant_object`, `tolerant_str_list`, `parse_cli_status`, `cli_status`, `cli_endpoint` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 879 | Native endpoint client and advertisement reads; declaration/member CliStatus; declarations/fields: `CliStatus` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 880 | Native endpoint client and advertisement reads; declaration/member CliStatus.backend_state; declarations/fields: `CliStatus.backend_state` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 881 | Native endpoint client and advertisement reads; declaration/member CliStatus.identity; declarations/fields: `CliStatus.identity` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 882 | Native endpoint client and advertisement reads; declaration/member CliStatus.ipv4; declarations/fields: `CliStatus.ipv4` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 883 | Native endpoint client and advertisement reads; declaration/member CliStatus.magic_dns_enabled; declarations/fields: `CliStatus.magic_dns_enabled` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 884 | Native endpoint client and advertisement reads; declaration/member CliStatus.expired; declarations/fields: `CliStatus.expired` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 885–889 | Native endpoint client and advertisement reads; declaration/member CliStatus.auth_pending; declarations/fields: `CliStatus.auth_pending` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 890 | Native endpoint client and advertisement reads; declaration/member CliEndpoint; declarations/fields: `CliEndpoint` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 891 | Native endpoint client and advertisement reads; declaration/member CliEndpoint.identity; declarations/fields: `CliEndpoint.identity` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 892–894 | Native endpoint client and advertisement reads; declaration/member CliEndpoint.ipv4; declarations/fields: `CliEndpoint.ipv4` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 895–902 | Native endpoint client and advertisement reads; declaration/member last_field; declarations/fields: `last_field` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 903–910 | Native endpoint client and advertisement reads; declaration/member tolerant_string; declarations/fields: `tolerant_string` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 911–918 | Native endpoint client and advertisement reads; declaration/member tolerant_bool; declarations/fields: `tolerant_bool` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 919–926 | Native endpoint client and advertisement reads; declaration/member tolerant_object; declarations/fields: `tolerant_object` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 927–944 | Native endpoint client and advertisement reads; declaration/member tolerant_str_list; declarations/fields: `tolerant_str_list` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 945–980 | Native endpoint client and advertisement reads; declaration/member parse_cli_status; declarations/fields: `parse_cli_status` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 981–998 | Native endpoint client and advertisement reads; declaration/member cli_status; declarations/fields: `cli_status` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 999–1029 | Native endpoint client and advertisement reads; declaration/member cli_endpoint; declarations/fields: `cli_endpoint` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1030 | Companion run status view; declarations/fields: `RunStatus` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1031 | Companion run status view; declaration/member RunStatus.enabled; declarations/fields: `RunStatus.enabled` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1032 | Companion run status view; declaration/member RunStatus.admission; declarations/fields: `RunStatus.admission` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1033 | Companion run status view; declaration/member RunStatus.tailnet; declarations/fields: `RunStatus.tailnet` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1034 | Companion run status view; declaration/member RunStatus.tags; declarations/fields: `RunStatus.tags` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1035 | Companion run status view; declaration/member RunStatus.addresses; declarations/fields: `RunStatus.addresses` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1036–1040 | Companion run status view; declaration/member RunStatus.dns_name; declarations/fields: `RunStatus.dns_name` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 1041–1057 | Companion run status view; declaration/member validate; declarations/fields: `validate` |

<a id="coverage-9b9220ebf3f1"></a>

## [rust/soda-host/src/tcontrol_policy.rs](../../../../../rust/soda-host/src/tcontrol_policy.rs)

 Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–24 | Protected enrollment provider credential and policy data; declarations/fields: `POLICY_VERSION`, `PROJECT_VERSION`, `POLICY_FILE_LIMIT`, `Credential`, `fmt`, `zero_secret`, `EnrollmentPolicy` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 25 | Protected enrollment provider credential and policy data; declaration/member POLICY_VERSION; declarations/fields: `POLICY_VERSION` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 26 | Protected enrollment provider credential and policy data; declaration/member PROJECT_VERSION; declarations/fields: `PROJECT_VERSION` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 27–30 | Protected enrollment provider credential and policy data; declaration/member POLICY_FILE_LIMIT; declarations/fields: `POLICY_FILE_LIMIT` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 31 | Protected enrollment provider credential and policy data; declaration/member Credential; declarations/fields: `Credential` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 32 | Protected enrollment provider credential and policy data; declaration/member Credential.client_id; declarations/fields: `Credential.client_id` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 33–36 | Protected enrollment provider credential and policy data; declaration/member Credential.secret; declarations/fields: `Credential.secret` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 37–45, 66–82 | Protected enrollment provider credential and policy data; declaration/member fmt; declarations/fields: `fmt` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 46–52 | Protected enrollment provider credential and policy data; declaration/member zero_secret; declarations/fields: `zero_secret` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 53 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy; declarations/fields: `EnrollmentPolicy` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 54 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.version; declarations/fields: `EnrollmentPolicy.version` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 55 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.revision; declarations/fields: `EnrollmentPolicy.revision` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 56 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.binding; declarations/fields: `EnrollmentPolicy.binding` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 57 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.tailnet; declarations/fields: `EnrollmentPolicy.tailnet` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 58 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.tags; declarations/fields: `EnrollmentPolicy.tags` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 59 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.preauthorized; declarations/fields: `EnrollmentPolicy.preauthorized` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 60 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.admission; declarations/fields: `EnrollmentPolicy.admission` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 61 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.default; declarations/fields: `EnrollmentPolicy.default` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 62–65 | Protected enrollment provider credential and policy data; declaration/member EnrollmentPolicy.credential; declarations/fields: `EnrollmentPolicy.credential` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 83 | Project selection entry data; declarations/fields: `ProjectPolicyEntry` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 84 | Project selection entry data; declaration/member ProjectPolicyEntry.version; declarations/fields: `ProjectPolicyEntry.version` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 85 | Project selection entry data; declaration/member ProjectPolicyEntry.revision; declarations/fields: `ProjectPolicyEntry.revision` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 86 | Project selection entry data; declaration/member ProjectPolicyEntry.project; declarations/fields: `ProjectPolicyEntry.project` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 87 | Project selection entry data; declaration/member ProjectPolicyEntry.container; declarations/fields: `ProjectPolicyEntry.container` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 88 | Project selection entry data; declaration/member ProjectPolicyEntry.binding; declarations/fields: `ProjectPolicyEntry.binding` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 89–93 | Project selection entry data; declaration/member ProjectPolicyEntry.enabled; declarations/fields: `ProjectPolicyEntry.enabled` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 94–104 | Project selection entry data; declaration/member CREDENTIAL_SPECS; declarations/fields: `CREDENTIAL_SPECS` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 105–113 | Project selection entry data; declaration/member ENROLLMENT_POLICY_SPECS; declarations/fields: `ENROLLMENT_POLICY_SPECS` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 114–147 | Credential/policy decode and validation; declarations/fields: `PROJECT_POLICY_SPECS`, `decode_policy_file`, `decode_enrollment_policy`, `encode_enrollment_policy` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 148–174 | Credential/policy decode and validation; declaration/member PROJECT_POLICY_SPECS; declarations/fields: `PROJECT_POLICY_SPECS` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 175–185 | Credential/policy decode and validation; declaration/member decode_policy_file; declarations/fields: `decode_policy_file` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 186–204 | Credential/policy decode and validation; declaration/member decode_enrollment_policy; declarations/fields: `decode_enrollment_policy` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 205–230 | Credential/policy decode and validation; declaration/member encode_enrollment_policy; declarations/fields: `encode_enrollment_policy` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 231–242 | Project entry decode; declarations/fields: `decode_project_entry` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 243–260 | Project entry decode; declaration/member encode_project_entry; declarations/fields: `encode_project_entry` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 261–295 | Canonical revision encoding; declarations/fields: `new_revision` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 296 | Exclusive protected file/lock/read/write primitives; declarations/fields: `PolicyLock` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 297–300 | Exclusive protected file/lock/read/write primitives; declaration/member PolicyLock.dir; declarations/fields: `PolicyLock.dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 301–305 | Exclusive protected file/lock/read/write primitives; declaration/member fd; declarations/fields: `fd` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 306–309 | Exclusive protected file/lock/read/write primitives; declaration/member dir_owned; declarations/fields: `dir_owned` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 310–317 | Exclusive protected file/lock/read/write primitives; declaration/member file_owned; declarations/fields: `file_owned` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 318–323 | Exclusive protected file/lock/read/write primitives; declaration/member last_errno; declarations/fields: `last_errno` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 324–347 | Exclusive protected file/lock/read/write primitives; declaration/member acquire_exclusive; declarations/fields: `acquire_exclusive` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 348–351 | Exclusive protected file/lock/read/write primitives; declaration/member cstr; declarations/fields: `cstr` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 352–357 | Exclusive protected file/lock/read/write primitives; declaration/member EnsureErr; declarations/fields: `EnsureErr` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 358–364 | Exclusive protected file/lock/read/write primitives; declaration/member current_uid; declarations/fields: `current_uid` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 365 | Exclusive protected file/lock/read/write primitives; declaration/member PolicyStore; declarations/fields: `PolicyStore` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 366 | Exclusive protected file/lock/read/write primitives; declaration/member PolicyStore.state_dir; declarations/fields: `PolicyStore.state_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 367 | Exclusive protected file/lock/read/write primitives; declaration/member PolicyStore.uid; declarations/fields: `PolicyStore.uid` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 368–370 | Exclusive protected file/lock/read/write primitives; declaration/member PolicyStore.runtime; declarations/fields: `PolicyStore.runtime` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 371–374 | Exclusive protected file/lock/read/write primitives; declaration/member PolicyStore.sync_hook; declarations/fields: `PolicyStore.sync_hook` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 375–383 | Exclusive protected file/lock/read/write primitives; declaration/member new; declarations/fields: `new` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 384–409 | Exclusive protected file/lock/read/write primitives; declaration/member create_dir; declarations/fields: `create_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 410–425 | Exclusive protected file/lock/read/write primitives; declaration/member ensure_dir; declarations/fields: `ensure_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 426–443 | Exclusive protected file/lock/read/write primitives; declaration/member lock; declarations/fields: `lock` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 444–469 | Exclusive protected file/lock/read/write primitives; declaration/member open_owned; declarations/fields: `open_owned` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 470–485 | Exclusive protected file/lock/read/write primitives; declaration/member read; declarations/fields: `read` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 486–494 | Exclusive protected file/lock/read/write primitives; declaration/member sync_dir; declarations/fields: `sync_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 495–535 | Exclusive protected file/lock/read/write primitives; declaration/member publish; declarations/fields: `publish` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 536–569 | Standing policy load/view and credential validity; declarations/fields: `load` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 570–587 | Standing policy load/view and credential validity; declaration/member view; declarations/fields: `view` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 588–601 | Standing policy load/view and credential validity; declaration/member enrollment; declarations/fields: `enrollment` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 602–624 | Standing policy load/view and credential validity; declaration/member check_enrollment; declarations/fields: `check_enrollment` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 625–660 | Standing policy load/view and credential validity; declaration/member apply_save_or_rotate; declarations/fields: `apply_save_or_rotate` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 661–676 | Standing policy load/view and credential validity; declaration/member apply_default; declarations/fields: `apply_default` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 677–688 | Standing policy load/view and credential validity; declaration/member apply_disable; declarations/fields: `apply_disable` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 689–741 | Standing policy load/view and credential validity; declaration/member update; declarations/fields: `update` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 742–771 | Project selection entry load, mutation and confirmed view; declarations/fields: `load_project` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 772–789 | Project selection entry load, mutation and confirmed view; declaration/member validate_project_binding; declarations/fields: `validate_project_binding` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 790–814 | Project selection entry load, mutation and confirmed view; declaration/member mutate_project; declarations/fields: `mutate_project` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 815–855 | Project selection entry load, mutation and confirmed view; declaration/member build_project_view; declarations/fields: `build_project_view` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 856–887 | Project selection entry load, mutation and confirmed view; declaration/member project; declarations/fields: `project` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 888–921 | Current policy/selection fenced companion run binding; declarations/fields: `run_binding` |

<a id="coverage-21b3658e2add"></a>

## [rust/soda-host/src/tcontrol_provider.rs](../../../../../rust/soda-host/src/tcontrol_provider.rs)

 Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–17 | Restricted Tailscale provider request/error and credential interface; declarations/fields: `TOKEN_URL`, `API_HOST`, `PROJECT_KEY_LIFETIME_SECS`, `KEY_DESCRIPTION`, `DEFAULT_CURL`, `ProviderRequest`, `fmt`, `provider_failed`, `config_escape`, `query_escape`, `token_form_body`, `key_create_body`, `token_config`, `key_config`, `run_curl`, `fetch_token`, `create_key`, `last_str`, `expires_in`, `validate_token` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 18 | Restricted Tailscale provider request/error and credential interface; declaration/member TOKEN_URL; declarations/fields: `TOKEN_URL` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 19 | Restricted Tailscale provider request/error and credential interface; declaration/member API_HOST; declarations/fields: `API_HOST` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 20 | Restricted Tailscale provider request/error and credential interface; declaration/member PROJECT_KEY_LIFETIME_SECS; declarations/fields: `PROJECT_KEY_LIFETIME_SECS` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 21 | Restricted Tailscale provider request/error and credential interface; declaration/member KEY_DESCRIPTION; declarations/fields: `KEY_DESCRIPTION` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 22–24 | Restricted Tailscale provider request/error and credential interface; declaration/member DEFAULT_CURL; declarations/fields: `DEFAULT_CURL` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 25–39 | Restricted Tailscale provider request/error and credential interface; declaration/member ProviderRequest; declarations/fields: `ProviderRequest` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 40–68 | Restricted Tailscale provider request/error and credential interface; declaration/member fmt; declarations/fields: `fmt` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 69–71 | Restricted Tailscale provider request/error and credential interface; declaration/member ProviderTransport; declarations/fields: `ProviderTransport` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 72–79 | Restricted Tailscale provider request/error and credential interface; declaration/member provider_failed; declarations/fields: `provider_failed` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 80–93 | Restricted Tailscale provider request/error and credential interface; declaration/member config_escape; declarations/fields: `config_escape` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 94–109 | Restricted Tailscale provider request/error and credential interface; declaration/member query_escape; declarations/fields: `query_escape` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 110–118 | Restricted Tailscale provider request/error and credential interface; declaration/member token_form_body; declarations/fields: `token_form_body` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 119–134 | Restricted Tailscale provider request/error and credential interface; declaration/member key_create_body; declarations/fields: `key_create_body` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 135–143 | Restricted Tailscale provider request/error and credential interface; declaration/member token_config; declarations/fields: `token_config` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 144–153 | Restricted Tailscale provider request/error and credential interface; declaration/member key_config; declarations/fields: `key_config` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 154–193 | Restricted Tailscale provider request/error and credential interface; declaration/member run_curl; declarations/fields: `run_curl` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 194–208 | Restricted Tailscale provider request/error and credential interface; declaration/member fetch_token; declarations/fields: `fetch_token` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 209–225 | Restricted Tailscale provider request/error and credential interface; declaration/member create_key; declarations/fields: `create_key` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 226–239 | Restricted Tailscale provider request/error and credential interface; declaration/member last_str; declarations/fields: `last_str` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 240–265 | Restricted Tailscale provider request/error and credential interface; declaration/member expires_in; declarations/fields: `expires_in` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 266–301 | Restricted Tailscale provider request/error and credential interface; declaration/member validate_token; declarations/fields: `validate_token` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 302–303 | Provider timestamp/expiration parsing; declarations/fields: `digits`, `days_since_year_one`, `parse_rfc3339_nanos`, `system_nanos` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 304–318 | Provider timestamp/expiration parsing; declaration/member digits; declarations/fields: `digits` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 319–332 | Provider timestamp/expiration parsing; declaration/member days_since_year_one; declarations/fields: `days_since_year_one` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 333–431 | Provider timestamp/expiration parsing; declaration/member parse_rfc3339_nanos; declarations/fields: `parse_rfc3339_nanos` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 432–441 | Provider timestamp/expiration parsing; declaration/member system_nanos; declarations/fields: `system_nanos` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 442–443 | Restricted key request admission and issuance; declarations/fields: `exact_field`, `fold_char`, `fold_eq`, `fold_field`, `require_str`, `require_object`, `require_bool`, `optional_bool`, `validate_key`, `ZERO_NANOS` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 444–449 | Restricted key request admission and issuance; declaration/member exact_field; declarations/fields: `exact_field` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 450–458 | Restricted key request admission and issuance; declaration/member fold_char; declarations/fields: `fold_char` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 459–479 | Restricted key request admission and issuance; declaration/member fold_eq; declarations/fields: `fold_eq` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 480–487 | Restricted key request admission and issuance; declaration/member fold_field; declarations/fields: `fold_field` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 488–494 | Restricted key request admission and issuance; declaration/member require_str; declarations/fields: `require_str` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 495–504 | Restricted key request admission and issuance; declaration/member require_object; declarations/fields: `require_object` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 505–511 | Restricted key request admission and issuance; declaration/member require_bool; declarations/fields: `require_bool` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 512–524 | Restricted key request admission and issuance; declaration/member optional_bool; declarations/fields: `optional_bool` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 525–578 | Restricted key request admission and issuance; declaration/member validate_key; declarations/fields: `validate_key` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 579–604 | Restricted key request admission and issuance; declaration/member ZERO_NANOS; declarations/fields: `ZERO_NANOS` |

<a id="coverage-3acfbb55921b"></a>

## [rust/soda-host/tests/tcontrol_oracle.rs](../../../../../rust/soda-host/tests/tcontrol_oracle.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–11, 355–407, 495–535, 1031–1032, 1103–1111, 1375, 1541–1592, 2177–2200, 2522–2527, 2650–2761, 2902–2916 | Pure host Tailnet, policy and Project selection compatibility oracle; declarations/fields: `json`, `project`, `sha256`, `tailnet_companion`, `tailnet_domain`, `tcontrol`, `tcontrol_enroll`, `tcontrol_native`, `tcontrol_policy`, `tcontrol_provider`, `tcontrol_wire`, `SCRATCH_COUNTER`, `scratch`, `soon`, `policy_store`, `accept_check`, `wire_revision_and_hex_patterns`, `wire_tag_client_and_network_patterns`, `wire_magic_dns_vectors`, `wire_address_vectors`, `wire_prefix_exit_ip_and_first_ipv4`, `wire_peer_view`, `wire_host_request_decode`, `wire_host_request_validate`, `wire_authentication_url`, `PID`, `runtime_seed`, `NATIVE_STATUS`, `native_signin_reauth_preserves_prefs`, `rfc3339`, `extract_revision`, `control_enroll_run_fences_identity_and_uncertainty`, `control_cli_passthrough` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 12–14 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member json; declarations/fields: `json` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 15–17 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member project; declarations/fields: `project` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 18–20 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member sha256; declarations/fields: `sha256` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 21–23 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tailnet_companion; declarations/fields: `tailnet_companion` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 24–28 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tailnet_domain; declarations/fields: `tailnet_domain` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 29–30 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tcontrol; declarations/fields: `tcontrol` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 31–32 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tcontrol_enroll; declarations/fields: `tcontrol_enroll` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 33–34 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tcontrol_native; declarations/fields: `tcontrol_native` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 35–36 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tcontrol_policy; declarations/fields: `tcontrol_policy` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 37–38 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tcontrol_provider; declarations/fields: `tcontrol_provider` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 39–44 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member tcontrol_wire; declarations/fields: `tcontrol_wire` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 45–47 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member SCRATCH_COUNTER; declarations/fields: `SCRATCH_COUNTER` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 48–62 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member scratch; declarations/fields: `scratch` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 63–66 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member soon; declarations/fields: `soon` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 67–75 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member policy_store; declarations/fields: `policy_store` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 76–82 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member accept_check; declarations/fields: `accept_check` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 83–96 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_revision_and_hex_patterns; declarations/fields: `wire_revision_and_hex_patterns` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 97–135 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_tag_client_and_network_patterns; declarations/fields: `wire_tag_client_and_network_patterns` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 136–168 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_magic_dns_vectors; declarations/fields: `wire_magic_dns_vectors` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 169–204 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_address_vectors; declarations/fields: `wire_address_vectors` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 205–245 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_prefix_exit_ip_and_first_ipv4; declarations/fields: `wire_prefix_exit_ip_and_first_ipv4` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 246–266 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_peer_view; declarations/fields: `wire_peer_view` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 267–305 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_host_request_decode; declarations/fields: `wire_host_request_decode` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 306–341, 422–472, 776–784, 911–946, 1042–1102, 1203–1250, 2462–2521, 2820–2901 | Source assertion of Project enrollment policy; declarations/fields: `wire_enrollment_request_decode`, `wire_enrollment_request_validate`, `policy_revision_format`, `policy_concurrency_and_cancelled_waiter`, `policy_project_disabled_until_runtime_exists`, `policy_project_closed_admission_has_no_reservation`, `control_enrollment_save_check_rotate_disable`, `control_failed_enrollment_can_be_explicitly_retried` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 342–354, 473–494 | Source assertion of Project Tailnet selection; declarations/fields: `wire_project_selection_decode`, `wire_project_selection_validate` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 408–421 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member enrollment_fixture; declarations/fields: `enrollment_fixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 536–561 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member host_fixture; declarations/fields: `host_fixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 562–576 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member enrollment_view_fixture; declarations/fields: `enrollment_view_fixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 577–663 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_view_validators; declarations/fields: `wire_view_validators` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 664–717 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_project_view_validator; declarations/fields: `wire_project_view_validator` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 718–775 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member wire_encoder_goldens; declarations/fields: `wire_encoder_goldens` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 785–797 | Source assertion of Project enrollment policy; declaration/member policy_reads_and_checks_are_non_mutating; declarations/fields: `policy_reads_and_checks_are_non_mutating` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 798–815 | Source assertion of Project enrollment policy; declaration/member policy_unsupported_default_has_no_effects; declarations/fields: `policy_unsupported_default_has_no_effects` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 816–875 | Source assertion of Project enrollment policy; declaration/member policy_rotation_cas_and_secret_projection; declarations/fields: `policy_rotation_cas_and_secret_projection` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 876–910, 2542–2598 | Source assertion of Encrypted credential custody; declarations/fields: `policy_does_not_convert_or_delete_retained_credentials`, `enroll_credential_and_key_guards` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 947–999 | Source assertion of Project enrollment policy; declaration/member policy_refuses_unsafe_and_ambiguous_state; declarations/fields: `policy_refuses_unsafe_and_ambiguous_state` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1000–1030 | Source assertion of Project enrollment policy; declaration/member policy_publication_failure_neither_rolls_back_nor_replays; declarations/fields: `policy_publication_failure_neither_rolls_back_nor_replays` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1033–1041 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member inspect_req; declarations/fields: `inspect_req` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 1112–1202, 2599–2649 | Source assertion of Native binding and private delivery; declarations/fields: `policy_project_runtime_selection_and_bindings`, `control_project_and_binding_through_trait` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1251–1298 | Source assertion of Project enrollment policy; declaration/member policy_project_enable_cas_and_no_implicit_retarget; declarations/fields: `policy_project_enable_cas_and_no_implicit_retarget` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1299–1374 | Source assertion of Project enrollment policy; declaration/member policy_project_missing_is_off_while_malformed_fails_safely; declarations/fields: `policy_project_missing_is_off_while_malformed_fails_safely` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1376–1379 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member NATIVE_PREFS; declarations/fields: `NATIVE_PREFS` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1380 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member GO_REVISION; declarations/fields: `GO_REVISION` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1381 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member GO_FRESH_REVISION; declarations/fields: `GO_FRESH_REVISION` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1382–1383 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member GO_HOSTVIEW; declarations/fields: `GO_HOSTVIEW` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1384–1397 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member fixture_transport; declarations/fields: `fixture_transport` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1398–1399 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member FakeRun; declarations/fields: `FakeRun` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1400 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member FakeExec; declarations/fields: `FakeExec` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1401–1404 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member FakeExec.f; declarations/fields: `FakeExec.f` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1405–1415 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member run; declarations/fields: `run` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1416–1422 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member boom_exec; declarations/fields: `boom_exec` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1423–1447 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_observe_fixture_matches_go_goldens; declarations/fields: `native_observe_fixture_matches_go_goldens` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1448–1464 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_fresh_daemon_omits_false_node_key; declarations/fields: `native_fresh_daemon_omits_false_node_key` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1465–1482 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_node_key_optional_but_strict; declarations/fields: `native_node_key_optional_but_strict` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 1483–1520 | Source assertion of Delegation and connection availability; declarations/fields: `native_unavailable_kinds` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 1521–1532 | Source assertion of Execution admission and lease fencing; declarations/fields: `native_revision_tracks_identity_without_release_veto` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1533–1540, 2138–2176 | Source assertion of Encoding and parsing; declarations/fields: `native_sha256_matches_reference`, `provider_rfc3339_matches_go_vectors` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1593–1697 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_exit_node_and_logout_confirm; declarations/fields: `native_exit_node_and_logout_confirm` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1698–1755 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_offline_exit_conflicts_and_retained_id_stays_unconfirmed; declarations/fields: `native_offline_exit_conflicts_and_retained_id_stays_unconfirmed` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1756–1783 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_up_notifications_and_command_bounds; declarations/fields: `native_up_notifications_and_command_bounds` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1784–1844 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_local_request_over_real_socket; declarations/fields: `native_local_request_over_real_socket` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1845–1878 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_run_status_validate; declarations/fields: `native_run_status_validate` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1879–1964 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member native_cli_client_vectors; declarations/fields: `native_cli_client_vectors` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1965–1980 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member provider_request_bodies_match_go_recipes; declarations/fields: `provider_request_bodies_match_go_recipes` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1981–2046 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member provider_curl_recipe_keeps_secrets_out_of_argv; declarations/fields: `provider_curl_recipe_keeps_secrets_out_of_argv` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2047–2137 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member provider_token_vectors; declarations/fields: `provider_token_vectors` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2201–2207 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member key_response; declarations/fields: `key_response` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2208–2296 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member provider_key_vectors; declarations/fields: `provider_key_vectors` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2297–2312 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member control_options; declarations/fields: `control_options` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2313–2319 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member token_ok; declarations/fields: `token_ok` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2320–2328 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member key_ok; declarations/fields: `key_ok` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2329–2361 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member provider_fixture; declarations/fields: `provider_fixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2362 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member GO_SETTINGS; declarations/fields: `GO_SETTINGS` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2363–2366 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member GO_OPTIONS; declarations/fields: `GO_OPTIONS` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2367–2386 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member control_settings_and_options_match_go_bytes; declarations/fields: `control_settings_and_options_match_go_bytes` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2387–2461 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member control_host_action_cycle; declarations/fields: `control_host_action_cycle` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2528–2541 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member seed_enrollment; declarations/fields: `seed_enrollment` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 2762–2819 | Pure host Tailnet, policy and Project selection compatibility oracle; declaration/member control_enroll_run_serial_explicit_requests_without_journal; declarations/fields: `control_enroll_run_serial_explicit_requests_without_journal` |


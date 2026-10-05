# Soda identity compose

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-c23f516b60ba"></a>

<a id="rustsoda-identity-composesrcmainrs-1"></a>

## [rust/soda-identity-compose/src/main.rs](../../../../../rust/soda-identity-compose/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–8 | Explicit native nested Compose execution registration CLI; declarations/fields: `MUSE_LAUNCH_SOCKET`, `TMPFS_MAGIC`, `Options`, `main`, `run`, `load_options`, `parse_options`, `print_usage`, `flag_error`, `parse_bool_flag`, `valid_options`, `go_base` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 9 | Explicit native nested Compose execution registration CLI; declaration/member MUSE_LAUNCH_SOCKET; declarations/fields: `MUSE_LAUNCH_SOCKET` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 10–12 | Explicit native nested Compose execution registration CLI; declaration/member TMPFS_MAGIC; declarations/fields: `TMPFS_MAGIC` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 13 | Explicit native nested Compose execution registration CLI; declaration/member Options; declarations/fields: `Options` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 14 | Explicit native nested Compose execution registration CLI; declaration/member Options.login; declarations/fields: `Options.login` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 15 | Explicit native nested Compose execution registration CLI; declaration/member Options.service; declarations/fields: `Options.service` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 16 | Explicit native nested Compose execution registration CLI; declaration/member Options.file; declarations/fields: `Options.file` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 17–19 | Explicit native nested Compose execution registration CLI; declaration/member Options.muse; declarations/fields: `Options.muse` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 20–26 | Explicit native nested Compose execution registration CLI; declaration/member main; declarations/fields: `main` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 27–39 | Explicit native nested Compose execution registration CLI; declaration/member run; declarations/fields: `run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 40–48 | Explicit native nested Compose execution registration CLI; declaration/member load_options; declarations/fields: `load_options` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 49–130 | Explicit native nested Compose execution registration CLI; declaration/member parse_options; declarations/fields: `parse_options` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 131–142 | Explicit native nested Compose execution registration CLI; declaration/member print_usage; declarations/fields: `print_usage` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 143–148 | Explicit native nested Compose execution registration CLI; declaration/member flag_error; declarations/fields: `flag_error` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 149–156 | Explicit native nested Compose execution registration CLI; declaration/member parse_bool_flag; declarations/fields: `parse_bool_flag` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 157–165 | Explicit native nested Compose execution registration CLI; declaration/member valid_options; declarations/fields: `valid_options` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 166–179 | Explicit native nested Compose execution registration CLI; declaration/member go_base; declarations/fields: `go_base` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 180–201 | Protected registration root and Project/local actor authority; declarations/fields: `registration_root` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 202–207 | Random id/private directory mechanics; declarations/fields: `read_random` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 208–216 | Random id/private directory mechanics; declaration/member hex_encode; declarations/fields: `hex_encode` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 217–244 | Random id/private directory mechanics; declaration/member mkdir_p; declarations/fields: `mkdir_p` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 245–287 | Native Compose launch and allowed nested service selection; declarations/fields: `launch_compose` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 288–315 | Native Compose launch and allowed nested service selection; declaration/member account; declarations/fields: `account` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 316–347 | Explicit private delivery mount override; declarations/fields: `write_override` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 348–372 | Private nested request and result JSON encoding/decoding; declarations/fields: `json_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 373 | Private nested request and result JSON encoding/decoding; declaration/member NestedRegistration; declarations/fields: `NestedRegistration` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 374 | Private nested request and result JSON encoding/decoding; declaration/member NestedRegistration.child_id; declarations/fields: `NestedRegistration.child_id` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 375 | Private nested request and result JSON encoding/decoding; declaration/member NestedRegistration.actor_id; declarations/fields: `NestedRegistration.actor_id` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 376 | Private nested request and result JSON encoding/decoding; declaration/member NestedRegistration.registration_id; declarations/fields: `NestedRegistration.registration_id` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 377–381 | Private nested request and result JSON encoding/decoding; declaration/member NestedRegistration.muse; declarations/fields: `NestedRegistration.muse` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 382–398 | Private nested request and result JSON encoding/decoding; declaration/member launch_request_json; declarations/fields: `launch_request_json` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 399–470 | Private nested request and result JSON encoding/decoding; declaration/member parse_launch_exit; declarations/fields: `parse_launch_exit` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 471–523 | Private nested request and result JSON encoding/decoding; declaration/member parse_json_string; declarations/fields: `parse_json_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 524–546 | Private nested request and result JSON encoding/decoding; declaration/member parse_json_integer; declarations/fields: `parse_json_integer` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 547–625 | Private nested request and result JSON encoding/decoding; declaration/member skip_json_value; declarations/fields: `skip_json_value` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 626–630 | Nested registration native socket exchange; declarations/fields: `register` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 631–632 | Nested registration native socket exchange; declaration/member Guard; declarations/fields: `Guard` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 633–697 | Nested registration native socket exchange; declaration/member drop; declarations/fields: `drop` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 698–725 | Immutable Project child container identity selection; declarations/fields: `select_compose_child` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 726–729 | Immutable Project child container identity selection; declaration/member compose_container_handle; declarations/fields: `compose_container_handle` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 730–742 | Immutable Project child container identity selection; declaration/member compose_observed_child; declarations/fields: `compose_observed_child` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 743–746 | Immutable Project child container identity selection; declaration/member immutable_compose_id; declarations/fields: `immutable_compose_id` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 747–751 | Immutable Project child container identity selection; declaration/member is_hex; declarations/fields: `is_hex` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 752–755 | Nested Compose source assertions; declarations/fields: `tests` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 756–771 | Nested Compose source assertions; declaration/member selects_only_requested_service_within_project; declarations/fields: `selects_only_requested_service_within_project` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 772–794 | Nested Compose source assertions; declaration/member rejects_ambiguous_or_unconfirmed_identity; declarations/fields: `rejects_ambiguous_or_unconfirmed_identity` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 795–808 | Nested Compose source assertions; declaration/member resolves_upstream_short_handle_to_immutable_identity; declarations/fields: `resolves_upstream_short_handle_to_immutable_identity` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 809–828 | Nested Compose source assertions; declaration/member compose_muse_mounts_are_explicit; declarations/fields: `compose_muse_mounts_are_explicit` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 829–838 | Nested Compose source assertions; declaration/member override_escapes_like_go_encoding_json; declarations/fields: `override_escapes_like_go_encoding_json` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 839–846 | Nested Compose source assertions; declaration/member go_base_matches_filepath_semantics; declarations/fields: `go_base_matches_filepath_semantics` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 847–855 | Nested Compose source assertions; declaration/member bool_flag_values_match_go; declarations/fields: `bool_flag_values_match_go` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 856–872 | Nested Compose source assertions; declaration/member launch_request_wire_matches_go; declarations/fields: `launch_request_wire_matches_go` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 873–910 | Nested Compose source assertions; declaration/member launch_exit_parsing_matches_go_unmarshal; declarations/fields: `launch_exit_parsing_matches_go_unmarshal` |


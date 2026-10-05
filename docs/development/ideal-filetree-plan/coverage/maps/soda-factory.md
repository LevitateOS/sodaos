# Soda factory

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-b25e1fd0f0dd"></a>

<a id="rustsoda-factorysrcmainrs-1"></a>

## [rust/soda-factory/src/main.rs](../../../../../rust/soda-factory/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–10 | Explicit factory operator status/stop/reconcile CLI; declarations/fields: `USAGE`, `OPERATOR_PATH`, `OPERATOR_HOST`, `RESPONSE_LIMIT`, `CLIENT_TIMEOUT`, `main`, `run`, `ParsedArgs`, `print_usage`, `parse_args`, `dispatch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 11–12 | Explicit factory operator status/stop/reconcile CLI; declaration/member USAGE; declarations/fields: `USAGE` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 13 | Explicit factory operator status/stop/reconcile CLI; declaration/member OPERATOR_PATH; declarations/fields: `OPERATOR_PATH` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 14 | Explicit factory operator status/stop/reconcile CLI; declaration/member OPERATOR_HOST; declarations/fields: `OPERATOR_HOST` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 15 | Explicit factory operator status/stop/reconcile CLI; declaration/member RESPONSE_LIMIT; declarations/fields: `RESPONSE_LIMIT` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 16–17 | Explicit factory operator status/stop/reconcile CLI; declaration/member CLIENT_TIMEOUT; declarations/fields: `CLIENT_TIMEOUT` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 18–27 | Explicit factory operator status/stop/reconcile CLI; declaration/member main; declarations/fields: `main` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 28–32 | Explicit factory operator status/stop/reconcile CLI; declaration/member run; declarations/fields: `run` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 33 | Explicit factory operator status/stop/reconcile CLI; declaration/member ParsedArgs; declarations/fields: `ParsedArgs` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 34 | Explicit factory operator status/stop/reconcile CLI; declaration/member ParsedArgs.socket; declarations/fields: `ParsedArgs.socket` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 35 | Explicit factory operator status/stop/reconcile CLI; declaration/member ParsedArgs.command; declarations/fields: `ParsedArgs.command` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 36–38 | Explicit factory operator status/stop/reconcile CLI; declaration/member ParsedArgs.positional; declarations/fields: `ParsedArgs.positional` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 39–44 | Explicit factory operator status/stop/reconcile CLI; declaration/member print_usage; declarations/fields: `print_usage` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 45–106 | Explicit factory operator status/stop/reconcile CLI; declaration/member parse_args; declarations/fields: `parse_args` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 107–152 | Explicit factory operator status/stop/reconcile CLI; declaration/member dispatch; declarations/fields: `dispatch` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 153–168 | Strict factory request JSON encoding; declarations/fields: `encode_envelope` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 169–189 | Strict factory request JSON encoding; declaration/member push_json_string; declarations/fields: `push_json_string` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 190–227 | Bounded private host socket transport; declarations/fields: `send` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 228–275 | Bounded private host socket transport; declaration/member read_response; declarations/fields: `read_response` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 276–290 | Bounded private host socket transport; declaration/member read_exact_limited; declarations/fields: `read_exact_limited` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 291–308 | Bounded private host socket transport; declaration/member read_to_end_limited; declarations/fields: `read_to_end_limited` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 309–331 | Bounded private host socket transport; declaration/member read_chunked; declarations/fields: `read_chunked` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 332–352 | Bounded private host socket transport; declaration/member read_line; declarations/fields: `read_line` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 353–359 | Bounded private host socket transport; declaration/member tests; declarations/fields: `tests` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 360–361 | Bounded private host socket transport; declaration/member TEST_SOCKET_SEQ; declarations/fields: `TEST_SOCKET_SEQ` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 362–367 | Factory CLI source assertions; declarations/fields: `args` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 368–427 | Factory CLI source assertions; declaration/member operator_server; declarations/fields: `operator_server` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 428–439 | Factory CLI source assertions; declaration/member request_parts; declarations/fields: `request_parts` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 440–461 | Factory CLI source assertions; declaration/member status_envelope_matches_go_client_bytes; declarations/fields: `status_envelope_matches_go_client_bytes` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 462–493 | Factory CLI source assertions; declaration/member stop_and_reconcile_envelopes_carry_sorted_keys; declarations/fields: `stop_and_reconcile_envelopes_carry_sorted_keys` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 494–524 | Factory CLI source assertions; declaration/member responses_map_like_the_go_client; declarations/fields: `responses_map_like_the_go_client` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 525–537 | Factory CLI source assertions; declaration/member success_appends_missing_trailing_newline; declarations/fields: `success_appends_missing_trailing_newline` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 538–547 | Factory CLI source assertions; declaration/member oversized_response_is_refused; declarations/fields: `oversized_response_is_refused` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 548–558 | Factory CLI source assertions; declaration/member unreachable_socket_maps_to_unavailable; declarations/fields: `unreachable_socket_maps_to_unavailable` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 559–585 | Factory CLI source assertions; declaration/member local_misuse_is_rejected; declarations/fields: `local_misuse_is_rejected` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 586–593 | Factory CLI source assertions; declaration/member envelope_escapes_match_go_encoding_json; declarations/fields: `envelope_escapes_match_go_encoding_json` |


# Soda identity compose

Current path navigation reconciled at `45ebf4c4` (2026-10-09); historical symbol/body selectors remain pinned to `519b76bd` unless a narrow current selector is explicitly stated.

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-4c615fef74df"></a>

## [cmd/soda-identity-compose/src/compose.rs](../../../../../cmd/soda-identity-compose/src/compose.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–125; current module/import/attribute shell; declaration launch_compose; declaration write_override; declaration select_compose_child; declaration compose_container_handle; declaration compose_observed_child; declaration immutable_compose_id; declaration is_hex | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-identity-compose/src/compose.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-95e15f5ddbe5"></a>

## [cmd/soda-identity-compose/src/compose_tests.rs](../../../../../cmd/soda-identity-compose/src/compose_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–207; current module/import/attribute shell; declaration selects_only_requested_service_within_project; declaration rejects_ambiguous_or_unconfirmed_identity; declaration resolves_upstream_short_handle_to_immutable_identity; declaration compose_muse_mounts_are_explicit; declaration override_escapes_like_go_encoding_json; declaration bool_flag_values_match_go; declaration launch_request_wire_matches_go; declaration launch_exit_preserves_scalar_and_duplicate_admission; declaration launch_exit_rejects_escape_split_inside_multibyte_char; declaration launch_exit_ignores_unknown_nested_response_fields; declaration launch_exit_skips_deep_unknown_fields_within_packet_bound | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-identity-compose/src/compose_tests.rs into its current native target.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-08b2c8f80476"></a>

## Former source `cmd/soda-identity-compose/src/launch_json.rs` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.
Launch serialization and parsing are allocated to [compose.rs](../../../../../cmd/soda-identity-compose/src/compose.rs) and [launch_wire.rs](../../../../../cmd/soda-identity-compose/src/launch_wire.rs); no body equivalence is implied.

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–160; current module/import/attribute shell; declaration RegistrationDto; fields child_id, actor_id, registration_id, muse; declaration LaunchRequestDto; fields register, connection_id, cwd, args, tty, cols, rows; declaration ComposeOverrideDto; fields services; declaration ComposeServiceDto; fields volumes; declaration GoHtmlFormatter; declaration write_string_fragment; declaration to_go_json; declaration json_string; declaration launch_request_json; declaration compose_override_json; declaration LaunchExit; fields code, error; declaration deserialize; declaration ExitVisitor; declaration Value; declaration expecting; declaration visit_map; declaration parse_launch_exit | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-identity-compose/src/launch_json.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a20a30b34aef"></a>

## [cmd/soda-identity-compose/src/launch_wire.rs](../../../../../cmd/soda-identity-compose/src/launch_wire.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; current module/import/attribute shell; declaration NestedRegistration; fields child_id, actor_id, registration_id, muse; declaration json_string; declaration launch_request_json; declaration parse_launch_exit | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-identity-compose/src/launch_wire.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-66a215d39adb"></a>

## [cmd/soda-identity-compose/src/main.rs](../../../../../cmd/soda-identity-compose/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–49; current module/import/attribute shell; declaration compose; declaration launch_json; declaration launch_wire; declaration options; declaration registration; declaration MUSE_LAUNCH_SOCKET; declaration main; declaration run; declaration load_options; declaration compose_tests | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-identity-compose/src/main.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a83479ff886e"></a>

## [cmd/soda-identity-compose/src/options.rs](../../../../../cmd/soda-identity-compose/src/options.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; declaration Options; fields login, service, file, muse; declaration parse_options; declaration print_usage; declaration flag_error; declaration parse_bool_flag; declaration valid_options | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Options: implement the current provider execution integration duty in options.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e4abb3f24cdc"></a>

## [cmd/soda-identity-compose/src/registration.rs](../../../../../cmd/soda-identity-compose/src/registration.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–150; current module/import/attribute shell; declaration TMPFS_MAGIC; declaration registration_root; declaration read_random; declaration hex_encode; declaration mkdir_p; declaration account; declaration register; declaration Guard; declaration drop | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Imports and module declarations wire cmd/soda-identity-compose/src/registration.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-c23f516b60ba"></a>
<a id="rustsoda-identity-composesrcmainrs-1"></a>

Former source `rust/soda-identity-compose/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

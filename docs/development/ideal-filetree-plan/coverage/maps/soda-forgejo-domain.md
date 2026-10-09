# Soda forgejo domain

Current path navigation reconciled at `45ebf4c4` (2026-10-09); historical symbol/body selectors remain pinned to `519b76bd` unless a narrow current selector is explicitly stated.

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-fd0c98989412"></a>

## [cmd/soda-forgejo-domain/src/cli.rs](../../../../../cmd/soda-forgejo-domain/src/cli.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; declaration DESCRIPTION; declaration usage; declaration help_text; declaration parse_args | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | DESCRIPTION: implement the current Forgejo host administration duty in cli.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5863d40181c9"></a>

## Former source `cmd/soda-forgejo-domain/src/config.rs` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.
Marker tests now live in [cmd/soda-forgejo-domain/src/tests/marker.rs](../../../../../cmd/soda-forgejo-domain/src/tests/marker.rs); former INI-emulation tests were retired.
Marker/path parsing duties now live in [cmd/soda-forgejo-domain/src/marker.rs](../../../../../cmd/soda-forgejo-domain/src/marker.rs); former local INI emulation was retired.

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–270; current module/import/attribute shell; declaration IniFile; fields defaults, sections; declaration parse_ini; declaration interpolate; declaration interpolate_depth; declaration ini_get; declaration app_data_path; declaration marker_path | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/config.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-dce78a1d2ffa"></a>

## [cmd/soda-forgejo-domain/src/domain.rs](../../../../../cmd/soda-forgejo-domain/src/domain.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–180; current module/import/attribute shell; declaration dispatch; declaration unit_active; declaration unit_masked; declaration container_present; declaration cmd_stop; declaration cmd_inhibit; declaration cmd_status; declaration cmd_lift; declaration cmd_start | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/domain.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-240e582db55f"></a>

## [cmd/soda-forgejo-domain/src/main.rs](../../../../../cmd/soda-forgejo-domain/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–60; current module/import/attribute shell; declaration cli; declaration config; declaration domain; declaration system; declaration tests; declaration main | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/main.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9a74ab06d8ec"></a>

## [cmd/soda-forgejo-domain/src/system.rs](../../../../../cmd/soda-forgejo-domain/src/system.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–69; current module/import/attribute shell; declaration UNIT; declaration CONTAINER; declaration MARKER_NAME; declaration STOP_TIMEOUT; declaration Paths; fields env_file, app_ini, data_root; declaration production; declaration Sys; declaration run; declaration elapsed; declaration sleep; declaration RealSys; declaration ORIGIN | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/system.rs into its current native target.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0917c9e32b21"></a>

## [cmd/soda-forgejo-domain/src/tests/cli.rs](../../../../../cmd/soda-forgejo-domain/src/tests/cli.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–28; current module/import/attribute shell; declaration cli_parsing_matches_argparse_verbs | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/tests/cli.rs into its current native target.; cli_parsing_matches_argparse_verbs: implement the current Forgejo host administration duty in cli.rs. — current source cmd/soda-forgejo-domain/src/tests/cli.rs; Cargo target and callers; current source cmd/soda-forgejo-domain/src/tests/cli.rs; lines 3-28; module/caller wiring inspected |

<a id="coverage-8c008d5b9cea"></a>

## Former source `cmd/soda-forgejo-domain/src/tests/config.rs` (body snapshot `519b76bd`)

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–133; current module/import/attribute shell; declaration bare_app_name_status_reports_clear_error; declaration bare_app_name_start_has_no_crash; declaration valid_app_ini_still_resolves; declaration parsed_but_missing_app_data_path_reports_clear_error; declaration ini_shapes_match_configparser; declaration marker_mapping_rejects_outside_volume_and_symlinks | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/tests/config.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2145f2d377f6"></a>

## [cmd/soda-forgejo-domain/src/tests/domain.rs](../../../../../cmd/soda-forgejo-domain/src/tests/domain.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–161; current module/import/attribute shell; declaration stop_verifies_quiescence_before_returning; declaration stop_reports_survivors_after_timeout; declaration inhibit_requires_quiescence_then_masks_and_marks; declaration status_reads_all_four_signals; declaration lift_removes_marker_and_unmasks; declaration start_refuses_inhibited_and_masked; declaration lift_refuses_unknown_marker_before_unmask; declaration start_refuses_unknown_marker_before_start | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/tests/domain.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e70d1c663d76"></a>

## [cmd/soda-forgejo-domain/src/tests/fixtures.rs](../../../../../cmd/soda-forgejo-domain/src/tests/fixtures.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–112; current module/import/attribute shell; declaration TEST_SEQ; declaration FakeSys; fields calls, active, enabled_out, podman_out, podman_code, start_code, now_values, sleeps; declaration new; declaration run; declaration elapsed; declaration sleep; declaration Fixture; fields temp, paths; declaration fixture; declaration valid_ini; declaration run_verb | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Imports and module declarations wire cmd/soda-forgejo-domain/src/tests/fixtures.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-16cc065bc9ea"></a>

## [cmd/soda-forgejo-domain/src/tests/mod.rs](../../../../../cmd/soda-forgejo-domain/src/tests/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4; declaration cli; declaration config; declaration domain; declaration fixtures | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | cli: implement the current Forgejo host administration duty in mod.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-91dc6070452d"></a>
<a id="rustsoda-forgejo-domainsrcmainrs-1"></a>

Former source `rust/soda-forgejo-domain/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.


## Current path reconciliation at `45ebf4c4`

This section reconciles current path ownership only. Existing numeric/body selectors above remain pinned to `519b76bd` unless a row explicitly gives a narrow current inspection.


<a id="r02-current-path-cmd-soda-forgejo-domain-src-marker-rs"></a>

### [cmd/soda-forgejo-domain/src/marker.rs](../../../../../cmd/soda-forgejo-domain/src/marker.rs)

O04: lines 1–189; parser 16–32, MarkerLocation 37–110, open helpers 113–151, marker_path 153–189.

<a id="r02-current-path-cmd-soda-forgejo-domain-src-tests-marker-rs"></a>

### [cmd/soda-forgejo-domain/src/tests/marker.rs](../../../../../cmd/soda-forgejo-domain/src/tests/marker.rs)

O04: lines 1–100, marker-domain current tests.

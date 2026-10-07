# Soda console welcome

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-1992de264921"></a>

## [cmd/soda-console-welcome/src/config.rs](../../../../../cmd/soda-console-welcome/src/config.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; render_config; read_config; opened config capped at 64 KiB + 1 before lossy UTF-8 and one Value parse; present listen must be a string, absent listen keeps default, forgejo_url must be a string | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Bounded direct Value reader preserves duplicate-last behavior, tolerant unknown fields and current display/refusal projection; exact-cap and cap+one behavior retained |

<a id="coverage-7e40f75adbbf"></a>

## [cmd/soda-console-welcome/src/main.rs](../../../../../cmd/soda-console-welcome/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21; current module/import/attribute shell; declaration config; declaration origin; declaration tests; declaration welcome; declaration main | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Imports and module declarations wire cmd/soda-console-welcome/src/main.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fcb24e804cc8"></a>

## [cmd/soda-console-welcome/src/origin.rs](../../../../../cmd/soda-console-welcome/src/origin.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–97; current module/import/attribute shell; declaration valid_listen; declaration is_loopback_ipv4; declaration valid_origin; declaration valid_percent_escapes | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Imports and module declarations wire cmd/soda-console-welcome/src/origin.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0e859202c0c5"></a>

## [cmd/soda-console-welcome/src/tests.rs](../../../../../cmd/soda-console-welcome/src/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; config_value_keeps_last_duplicate_and_unknown_values; string_escapes_decode_like_python; config_reader_accepts_exact_cap_and_rejects_cap_plus_one; listen_shapes; origin_shapes | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Five current welcome/config checks, including bounded reads, duplicate handling and rendering-input projection |

<a id="coverage-a90a4f3ebb4c"></a>

## [cmd/soda-console-welcome/src/welcome.rs](../../../../../cmd/soda-console-welcome/src/welcome.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–168; current module/import/attribute shell; declaration run; declaration capture; declaration have_command; declaration hostname | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Imports and module declarations wire cmd/soda-console-welcome/src/welcome.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-d4e4d1750a74"></a>
<a id="rustsoda-console-welcomesrcmainrs-1"></a>

Former source `rust/soda-console-welcome/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

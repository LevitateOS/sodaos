# Soda activate

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-24fa2a3f9ddf"></a>

## [cmd/soda-activate/src/activation.rs](../../../../../cmd/soda-activate/src/activation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–300; activation admission and effect sequencing; one opened dashboard file capped at 64 KiB + 1 and parsed once into Value; consumed typed fields and presence-based public_url refusal retained | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | One bounded Value projection preserves operator positive-i64 admission, exact consumed keys, null/presence semantics and no-effect-on-invalid input. Current tests: 13 activation checks, including exact cap/overflow and refusal before effects |

<a id="coverage-8df5ebf145e4"></a>

## [cmd/soda-activate/src/cli.rs](../../../../../cmd/soda-activate/src/cli.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113; declaration CliArgs; fields bind_ip, certificate, private_key, local_tls; declaration parse_args | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | CliArgs: implement the current private origins, TLS, and activation duty in cli.rs.; parse_args: implement the current private origins, TLS, and activation duty in cli.rs. — current source cmd/soda-activate/src/cli.rs; lines 1-10; module/caller wiring inspected; current source cmd/soda-activate/src/cli.rs; lines 11-113; module/caller wiring inspected |

<a id="coverage-3dca059c795a"></a>

## [cmd/soda-activate/src/forgejo_env.rs](../../../../../cmd/soda-activate/src/forgejo_env.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–103; current module/import/attribute shell; declaration rewrite_forgejo_env | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/forgejo_env.rs into its current native target.; rewrite_forgejo_env: implement the current private origins, TLS, and activation duty in forgejo_env.rs. — current source cmd/soda-activate/src/forgejo_env.rs; Cargo target and callers; current source cmd/soda-activate/src/forgejo_env.rs; lines 8-103; module/caller wiring inspected |

<a id="coverage-f9e3327fb2fc"></a>

## [cmd/soda-activate/src/main.rs](../../../../../cmd/soda-activate/src/main.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; current module/import/attribute shell; declaration activation; declaration cli; declaration forgejo_env; declaration origin; declaration system; declaration tests; declaration USAGE; declaration help_text; declaration main | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/main.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a9606b2e4b4e"></a>

## [cmd/soda-activate/src/origin.rs](../../../../../cmd/soda-activate/src/origin.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–239; current module/import/attribute shell; declaration v4_in_net; declaration v4_is_private; declaration NETS; declaration v4_is_global; declaration v6_in_net; declaration v6_is_private; declaration EXCEPTIONS; declaration activate_rejects_ip; declaration OriginParts; fields hostname, port, port_present; declaration split_origin; declaration check_browser_origin; declaration origin_host_port; declaration raw_authority; declaration host_to_string; declaration valid_percent_escapes | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/origin.rs into its current native target.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2b3cd99821b0"></a>

## [cmd/soda-activate/src/system.rs](../../../../../cmd/soda-activate/src/system.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–117; current module/import/attribute shell; declaration Paths; fields root, var_lib, containers_systemd; declaration production; declaration ActivateError; declaration usage; declaration runtime; declaration Sys; declaration euid; declaration lookup_user; declaration chown; declaration run; declaration elapsed; declaration sleep; declaration RealSys; declaration MAX_NSS_BYTES; declaration ORIGIN | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/system.rs into its current native target.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a6150f59078f"></a>

## [cmd/soda-activate/src/tests/activation.rs](../../../../../cmd/soda-activate/src/tests/activation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–356; first_activation_derives_provider_from_forgejo_origin; tls_mode_rejects_missing_or_mixed_inputs_before_effects; empty_identity_socket_uses_standard_admin_socket; reports_units_that_never_become_active; operator_identity_encodings; dashboard_duplicate_fields_are_last_wins_and_unknown_fields_are_ignored; public_url_null_is_present_and_trailing_values_are_rejected_before_mutation; dashboard_input_cap_accepts_exact_limit_and_refuses_overflow_before_effects; refuses_without_root_before_effects | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Current activation/projection and pre-effect refusal regressions; included in 13 actual REP-CFG-1 checks |

<a id="coverage-83cb6f3d7b0f"></a>

## [cmd/soda-activate/src/tests/cli.rs](../../../../../cmd/soda-activate/src/tests/cli.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; current module/import/attribute shell; declaration cli_parsing_matches_argparse | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/tests/cli.rs into its current native target.; cli_parsing_matches_argparse: implement the current private origins, TLS, and activation duty in cli.rs. — current source cmd/soda-activate/src/tests/cli.rs; Cargo target and callers; current source cmd/soda-activate/src/tests/cli.rs; lines 3-33; module/caller wiring inspected |

<a id="coverage-76255fa46cd1"></a>

## [cmd/soda-activate/src/tests/fixtures.rs](../../../../../cmd/soda-activate/src/tests/fixtures.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; current module/import/attribute shell; declaration TEST_SEQ; declaration FakeSys; fields euid, user, calls, chowns, probe_code, now_values, sleeps; declaration new; declaration euid; declaration lookup_user; declaration chown; declaration run; declaration elapsed; declaration sleep; declaration Fixture; fields temp, paths; declaration fixture; declaration dashboard; declaration cli; declaration env_map | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/tests/fixtures.rs into its current native target.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-153f31c510a3"></a>

## [cmd/soda-activate/src/tests/mod.rs](../../../../../cmd/soda-activate/src/tests/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4; declaration activation; declaration cli; declaration fixtures; declaration origin | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | activation: implement the current private origins, TLS, and activation duty in mod.rs.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7ca2c5817666"></a>

## [cmd/soda-activate/src/tests/origin.rs](../../../../../cmd/soda-activate/src/tests/origin.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–107; current module/import/attribute shell; declaration ip_classifier_matches_cpython_oracle; declaration browser_origin_checks_match_activate_rules; declaration origin_port_presence_and_ipv6_host_are_retained | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Imports and module declarations wire cmd/soda-activate/src/tests/origin.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-e5789a937347"></a>
<a id="rustsoda-activatesrcmainrs-1"></a>

Former source `rust/soda-activate/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

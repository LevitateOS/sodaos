# Soda factory

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-d147fbf65cbb"></a>

## [cmd/soda-factory/src/main.rs](../../../../../cmd/soda-factory/src/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–242; lines 1–6: use std and attached body; lines 7–7: use std and attached body; lines 8–8: use std and attached body; lines 9–9: use std and attached body; lines 10–12: const USAGE and attached body; lines 13–13: const OPERATOR_PATH and attached body; lines 14–14: const OPERATOR_HOST and attached body; lines 15–15: const RESPONSE_LIMIT and attached body; lines 16–16: const CLIENT_TIMEOUT and attached body; lines 17–26: fn main and attached body; lines 27–31: fn run and attached body; lines 32–37: struct ParsedArgs and attached body; lines 38–43: fn print_usage and attached body; lines 44–105: fn parse_args and attached body; lines 106–150: fn dispatch and attached body; lines 151–151: fn encode_envelope and attached body; lines 152–152: use serde and attached body; lines 153–153: use serde_json and attached body; lines 154–154: use std and attached body; lines 155–155: use std and attached body; lines 156–157: struct GoFormatter and attached body; lines 158–158: impl Formatter and attached body; lines 159–181: fn write_string_fragment and attached body; lines 182–203: fn write_char_escape and attached body; lines 204–241: fn send and attached body; lines 242–242: mod operator_tests and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ed0d743acaf4"></a>

## [cmd/soda-factory/src/operator_tests.rs](../../../../../cmd/soda-factory/src/operator_tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–247; lines 1–1: use super and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: static TEST_SOCKET_SEQ and attached body; lines 9–15: fn args and attached body; lines 16–81: fn operator_server and attached body; lines 82–94: fn request_parts and attached body; lines 95–117: fn status_envelope_matches_go_client_bytes and attached body; lines 118–149: fn stop_and_reconcile_envelopes_carry_sorted_keys and attached body; lines 150–180: fn responses_map_like_the_go_client and attached body; lines 181–193: fn success_appends_missing_trailing_newline and attached body; lines 194–202: fn oversized_response_is_refused and attached body; lines 203–213: fn unreachable_socket_maps_to_unavailable and attached body; lines 214–240: fn local_misuse_is_rejected and attached body; lines 241–247: fn envelope_escapes_match_go_encoding_json and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-b25e1fd0f0dd"></a>
<a id="rustsoda-factorysrcmainrs-1"></a>

Former source `rust/soda-factory/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

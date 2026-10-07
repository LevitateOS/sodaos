# Soda release deliver verification

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-3c0c572bbe9b"></a>

## [lib/soda-release-deliver/tests/oracle/artifacts.rs](../../../../../lib/soda-release-deliver/tests/oracle/artifacts.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–297; lines 1–1: use base64 and attached body; lines 2–2: use serde_json and attached body; lines 3–3: use soda_release_deliver and attached body; lines 4–4: use soda_release_deliver and attached body; lines 5–5: use soda_release_deliver and attached body; lines 6–6: use soda_release_deliver and attached body; lines 7–7: use soda_release_deliver and attached body; lines 8–8: use soda_release_deliver and attached body; lines 9–9: use soda_release_deliver and attached body; lines 10–16: use super and attached body; lines 17–46: fn write_document_digest_is_reproducible_and_content_round_trips and attached body; lines 47–69: fn copy_as_dir and attached body; lines 70–140: fn oci_inspection_matches_oracle and attached body; lines 141–180: fn strict_decode_edges_match_oracle and attached body; lines 181–196: fn payload_load_matches_owner and attached body; lines 197–235: fn check_candidate_binds_before_archives and attached body; lines 236–297: fn admit_qualification_accepts_bound_evidence and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use base64 in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cf243faeb13f"></a>

## [lib/soda-release-deliver/tests/oracle/fetch_state.rs](../../../../../lib/soda-release-deliver/tests/oracle/fetch_state.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; lines 1–1: use soda_release_deliver and attached body; lines 2–2: use soda_release_deliver and attached body; lines 3–3: use soda_release_deliver and attached body; lines 4–4: use soda_release_deliver and attached body; lines 5–8: use super and attached body; lines 9–29: fn state_and_ledger_init_round_trip and attached body; lines 30–35: struct ScriptRunner and attached body; lines 36–37: impl Runner and attached body; lines 38–64: fn run and attached body; lines 65–81: fn fetch_rejects_bad_runner_output and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use soda_release_deliver in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5eee23e9b5ab"></a>

## [lib/soda-release-deliver/tests/oracle/main.rs](../../../../../lib/soda-release-deliver/tests/oracle/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–367; lines 1–9: use serde_json and attached body; lines 10–10: use soda_release_deliver and attached body; lines 11–14: use soda_release_deliver and attached body; lines 15–15: use soda_release_deliver and attached body; lines 16–16: use soda_release_deliver and attached body; lines 17–17: use soda_release_deliver and attached body; lines 18–19: mod artifacts and attached body; lines 20–20: mod fetch_state and attached body; lines 21–22: const GOLDENS and attached body; lines 23–26: fn goldens and attached body; lines 27–33: fn golden_str and attached body; lines 34–47: fn golden_result and attached body; lines 48–52: fn decode_trust and attached body; lines 53–58: fn decode_payload and attached body; lines 59–64: fn decode_candidate and attached body; lines 65–69: fn decode_release and attached body; lines 70–76: fn decode_channel and attached body; lines 77–106: fn fixtures_round_trip_byte_identical and attached body; lines 107–191: fn validation_battery_matches_oracle and attached body; lines 192–199: fn golden_now and attached body; lines 200–240: fn channel_progression_matches_oracle and attached body; lines 241–276: fn release_admission_matches_oracle and attached body; lines 277–302: fn merge_policy_matches_oracle_semantically and attached body; lines 303–319: fn merge_policy_preserves_arbitrary_object_order_and_number_tokens and attached body; lines 320–328: fn dom_sort and attached body; lines 329–341: fn json_escape and attached body; lines 342–347: fn hex_sha256 and attached body; lines 348–360: fn temp_dir and attached body; lines 361–363: fn private_dir and attached body; lines 364–367: use std and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde_json in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-5b77f05522c5"></a>
<a id="rustsoda-release-delivertestsoraclers-1"></a>

Former source `lib/soda-release-deliver/tests/oracle.rs`; consult its pinned earlier Git source and the current coverage disposition.

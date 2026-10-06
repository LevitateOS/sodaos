# Soda release deliver verification

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 @HEAD `d5012d10` (C08): oracle.rs verified byte-identical move; intervals kept.

<a id="coverage-5b77f05522c5"></a>

<a id="rustsoda-release-delivertestsoraclers-1"></a>

## [lib/soda-release-deliver/tests/oracle.rs](../../../../../lib/soda-release-deliver/tests/oracle.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1–23 | Captured native Go release fixtures; declarations/fields: `GOLDENS`, `goldens`, `golden_str`, `golden_result`, `decode_trust`, `decode_payload`, `decode_candidate`, `decode_release`, `decode_channel` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 24–25 | Captured native Go release fixtures; declaration/member GOLDENS; declarations/fields: `GOLDENS` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 26–29 | Captured native Go release fixtures; declaration/member goldens; declarations/fields: `goldens` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 30–36 | Captured native Go release fixtures; declaration/member golden_str; declarations/fields: `golden_str` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 37–50 | Captured native Go release fixtures; declaration/member golden_result; declarations/fields: `golden_result` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 51–56 | Captured native Go release fixtures; declaration/member decode_trust; declarations/fields: `decode_trust` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 57–63 | Captured native Go release fixtures; declaration/member decode_payload; declarations/fields: `decode_payload` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 64–70 | Captured native Go release fixtures; declaration/member decode_candidate; declarations/fields: `decode_candidate` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 71–76 | Captured native Go release fixtures; declaration/member decode_release; declarations/fields: `decode_release` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 77–83 | Captured native Go release fixtures; declaration/member decode_channel; declarations/fields: `decode_channel` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 84–101 | Round-trip byte oracle; declarations/fields: `fixtures_round_trip_byte_identical` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 102–186 | Trust/candidate/release schema validation oracle; declarations/fields: `validation_battery_matches_oracle` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 187–195 | Trust/candidate/release schema validation oracle; declaration/member golden_now; declarations/fields: `golden_now` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 196–236 | Channel monotonic progression oracle; declarations/fields: `channel_progression_matches_oracle` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 237–273 | Release admission oracle; declarations/fields: `release_admission_matches_oracle` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 274–299 | Native verification-copy policy oracle; declarations/fields: `merge_policy_matches_oracle_semantically` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 300–313 | Canonical JSON shape helper; declarations/fields: `dom_sort` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 314–335 | Immutable document digest oracle; declarations/fields: `write_document_digests_match_oracle` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 336–365 | OCI directory fixture and identity oracle; declarations/fields: `copy_as_dir` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 366–431 | OCI directory fixture and identity oracle; declaration/member oci_inspection_matches_oracle; declarations/fields: `oci_inspection_matches_oracle` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 432–447 | Strict JSON oracle; declarations/fields: `strict_decode_edges_match_oracle` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 448–463 | Payload immutable load oracle; declarations/fields: `payload_load_matches_owner` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 464–475 | Highwater initialized-state round trip; declarations/fields: `state_and_ledger_init_round_trip` |
| [D09](../../slices/release-and-installation.md#d09-publication-and-effect-observation) / active | 476–486 | Publication initialized-ledger round trip |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 487–525 | Candidate byte binding before archive inspection; declarations/fields: `check_candidate_binds_before_archives` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 526–588 | Exact qualification evidence admission; declarations/fields: `admit_qualification_accepts_bound_evidence` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 589–601 | Test JSON/hash helpers; declarations/fields: `json_escape` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 602–607 | Test JSON/hash helpers; declaration/member hex_sha256; declarations/fields: `hex_sha256` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 608–620 | Private verification transport fixtures and fetch refusal oracle; declarations/fields: `temp_dir` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 621–628 | Private verification transport fixtures and fetch refusal oracle; declaration/member private_dir; declarations/fields: `private_dir` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 629 | Private verification transport fixtures and fetch refusal oracle; declaration/member ScriptRunner; declarations/fields: `ScriptRunner` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 630 | Private verification transport fixtures and fetch refusal oracle; declaration/member ScriptRunner.calls; declarations/fields: `ScriptRunner.calls` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 631 | Private verification transport fixtures and fetch refusal oracle; declaration/member ScriptRunner.inspect_raw; declarations/fields: `ScriptRunner.inspect_raw` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 632 | Private verification transport fixtures and fetch refusal oracle; declaration/member ScriptRunner.copy_manifest; declarations/fields: `ScriptRunner.copy_manifest` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 633–636 | Private verification transport fixtures and fetch refusal oracle; declaration/member ScriptRunner.copy_config; declarations/fields: `ScriptRunner.copy_config` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 637–663 | Private verification transport fixtures and fetch refusal oracle; declaration/member run; declarations/fields: `run` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 664–680 | Source assertion of Verified distribution consumption; declarations/fields: `fetch_rejects_bad_runner_output` |


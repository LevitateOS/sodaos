# Soda release tools verification

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-a9e3578f5f3a"></a>

## [lib/soda-release-tools/tests/cli/main.rs](../../../../../lib/soda-release-tools/tests/cli/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; lines 1–7: use std and attached body; lines 8–8: use std and attached body; lines 9–9: use std and attached body; lines 10–10: use std and attached body; lines 11–11: use std and attached body; lines 12–13: mod soda_artifacts and attached body; lines 14–14: mod soda_build and attached body; lines 15–15: mod soda_candidate and attached body; lines 16–17: static COUNTER and attached body; lines 18–21: struct TempDir and attached body; lines 22–23: impl TempDir and attached body; lines 24–31: fn new and attached body; lines 32–33: impl Drop and attached body; lines 34–37: fn drop and attached body; lines 38–41: fn build_bin and attached body; lines 42–45: fn candidate_bin and attached body; lines 46–49: fn artifacts_bin and attached body; lines 50–62: fn run and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-06628416cfcc"></a>

## [lib/soda-release-tools/tests/cli/soda_artifacts.rs](../../../../../lib/soda-release-tools/tests/cli/soda_artifacts.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–7: use super and attached body; lines 8–49: fn artifacts_dispatch_matches_go and attached body; lines 50–61: fn artifacts_generated_help_exits_before_work and attached body; lines 62–99: fn artifacts_butane_refusals_match_go and attached body; lines 100–120: fn artifacts_inspect_validates_archive_shape and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5a1076890198"></a>

## [lib/soda-release-tools/tests/cli/soda_build.rs](../../../../../lib/soda-release-tools/tests/cli/soda_build.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–168; lines 1–3: use super and attached body; lines 4–20: fn build_help_is_generated_and_safe and attached body; lines 21–43: fn build_flag_errors_refuse_without_running and attached body; lines 44–93: fn build_admission_matrix_preserves_release_boundaries and attached body; lines 94–130: fn build_bool_and_scalar_forms_preserve_contract and attached body; lines 131–168: fn build_deep_refusal_matches_go and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d3980f923fd5"></a>

## [lib/soda-release-tools/tests/cli/soda_candidate.rs](../../../../../lib/soda-release-tools/tests/cli/soda_candidate.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–154; lines 1–3: use super and attached body; lines 4–20: fn candidate_help_is_generated_and_safe and attached body; lines 21–69: fn candidate_flag_errors_refuse_before_admission and attached body; lines 70–154: fn candidate_validation_preserves_admission_boundaries and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-9093b495ba8c"></a>
<a id="rustsoda-release-toolstestsclirs-1"></a>

Former source `lib/soda-release-tools/tests/cli.rs`; consult its pinned earlier Git source and the current coverage disposition.

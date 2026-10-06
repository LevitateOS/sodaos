# Soda release tools verification

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 @HEAD `d5012d10` (C08): cli.rs verified byte-identical move; intervals kept.

<a id="coverage-9093b495ba8c"></a>

<a id="rustsoda-release-toolstestsclirs-1"></a>

## [lib/soda-release-tools/tests/cli.rs](../../../../../lib/soda-release-tools/tests/cli.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 1–13 | CLI usage/refusal fixtures and builder admission oracle; declarations/fields: `COUNTER`, `TempDir`, `new`, `drop`, `build_bin`, `candidate_bin`, `artifacts_bin`, `run`, `go_build_usage`, `build_help_matches_go`, `build_flag_errors_match_go`, `build_admission_matrix_matches_go`, `build_bool_forms_match_go`, `go_candidate_usage`, `candidate_help_matches_go`, `candidate_flag_errors_match_go`, `candidate_validation_matches_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 14–15 | CLI usage/refusal fixtures and builder admission oracle; declaration/member COUNTER; declarations/fields: `COUNTER` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 16 | CLI usage/refusal fixtures and builder admission oracle; declaration/member TempDir; declarations/fields: `TempDir` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 17–20 | CLI usage/refusal fixtures and builder admission oracle; declaration/member TempDir.path; declarations/fields: `TempDir.path` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 21–30 | CLI usage/refusal fixtures and builder admission oracle; declaration/member new; declarations/fields: `new` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 31–35 | CLI usage/refusal fixtures and builder admission oracle; declaration/member drop; declarations/fields: `drop` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 36–39 | CLI usage/refusal fixtures and builder admission oracle; declaration/member build_bin; declarations/fields: `build_bin` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 40–43 | CLI usage/refusal fixtures and builder admission oracle; declaration/member candidate_bin; declarations/fields: `candidate_bin` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 44–47 | CLI usage/refusal fixtures and builder admission oracle; declaration/member artifacts_bin; declarations/fields: `artifacts_bin` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 48–60 | CLI usage/refusal fixtures and builder admission oracle; declaration/member run; declarations/fields: `run` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 61–67 | CLI usage/refusal fixtures and builder admission oracle; declaration/member go_build_usage; declarations/fields: `go_build_usage` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 68–82 | CLI usage/refusal fixtures and builder admission oracle; declaration/member build_help_matches_go; declarations/fields: `build_help_matches_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 83–104 | CLI usage/refusal fixtures and builder admission oracle; declaration/member build_flag_errors_match_go; declarations/fields: `build_flag_errors_match_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 105–155 | CLI usage/refusal fixtures and builder admission oracle; declaration/member build_admission_matrix_matches_go; declarations/fields: `build_admission_matrix_matches_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 156–176 | CLI usage/refusal fixtures and builder admission oracle; declaration/member build_bool_forms_match_go; declarations/fields: `build_bool_forms_match_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 177–182 | CLI usage/refusal fixtures and builder admission oracle; declaration/member go_candidate_usage; declarations/fields: `go_candidate_usage` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 183–197 | CLI usage/refusal fixtures and builder admission oracle; declaration/member candidate_help_matches_go; declarations/fields: `candidate_help_matches_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 198–240 | CLI usage/refusal fixtures and builder admission oracle; declaration/member candidate_flag_errors_match_go; declarations/fields: `candidate_flag_errors_match_go` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 241–327 | CLI usage/refusal fixtures and builder admission oracle; declaration/member candidate_validation_matches_go; declarations/fields: `candidate_validation_matches_go` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 328–373 | Artifacts dispatch/refusal oracle; declarations/fields: `artifacts_dispatch_matches_go` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 374–411 | Butane conversion refusal oracle; declarations/fields: `artifacts_butane_refusals_match_go` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 412–434 | OCI archive shape refusal oracle; declarations/fields: `artifacts_inspect_validates_archive_shape` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 435–472 | Deep builder admission refusal oracle; declarations/fields: `build_deep_refusal_matches_go` |


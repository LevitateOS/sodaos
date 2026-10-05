# Soda release build verification

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-944b552bf221"></a>

<a id="rustsoda-release-buildtestsoraclers-1"></a>

## [rust/soda-release-build/tests/oracle.rs](../../../../../rust/soda-release-build/tests/oracle.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–8 | Captured Go vectors and native production fixtures; declarations/fields: `oracle_vectors`, `FIXTURE_REVISION`, `data_path`, `scratch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 9–32 | Captured Go vectors and native production fixtures; declaration/member oracle_vectors; declarations/fields: `oracle_vectors` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 33–34 | Captured Go vectors and native production fixtures; declaration/member FIXTURE_REVISION; declarations/fields: `FIXTURE_REVISION` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 35–40 | Captured Go vectors and native production fixtures; declaration/member data_path; declarations/fields: `data_path` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 41–55 | Captured Go vectors and native production fixtures; declaration/member scratch; declarations/fields: `scratch` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 56–67 | OCI archive/layout/content identity oracle assertions; declarations/fields: `oracle_go_fixture_identity` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 68–101 | OCI archive/layout/content identity oracle assertions; declaration/member oracle_go_fixture_rejections; declarations/fields: `oracle_go_fixture_rejections` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 102–129 | OCI archive/layout/content identity oracle assertions; declaration/member oracle_go_fixture_content; declarations/fields: `oracle_go_fixture_content` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 130–156 | OCI archive/layout/content identity oracle assertions; declaration/member oracle_go_layout; declarations/fields: `oracle_go_layout` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 157–192 | Pinned live-input/base record byte oracle; declarations/fields: `oracle_live_inputs` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 193–224 | Pinned live-input/base record byte oracle; declaration/member oracle_live_inputs_bytes; declarations/fields: `oracle_live_inputs_bytes` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 225–236 | Pinned live-input/base record byte oracle; declaration/member oracle_verified_base_bytes; declarations/fields: `oracle_verified_base_bytes` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 237–249 | Pinned live-input/base record byte oracle; declaration/member oracle_resolved_inputs_bytes; declarations/fields: `oracle_resolved_inputs_bytes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 250–261 | Native fork build argv/toolchain/production oracle; declarations/fields: `oracle_forgejo_argv_and_script` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 262–274 | Native fork build argv/toolchain/production oracle; declaration/member oracle_forgejo_toolchain_bytes; declarations/fields: `oracle_forgejo_toolchain_bytes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 275–285 | Native fork build argv/toolchain/production oracle; declaration/member fixture_elf; declarations/fields: `fixture_elf` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 286–432 | Native fork build argv/toolchain/production oracle; declaration/member oracle_production_sequence; declarations/fields: `oracle_production_sequence` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 433–462 | Progress/exit-code byte oracle; declarations/fields: `oracle_progress_bytes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 463–491 | Progress/exit-code byte oracle; declaration/member oracle_exit_codes; declarations/fields: `oracle_exit_codes` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 492–516 | Encoding and strict JSON oracle; declarations/fields: `oracle_misc_vectors` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 517–541 | Encoding and strict JSON oracle; declaration/member oracle_read_json_strictness; declarations/fields: `oracle_read_json_strictness` |


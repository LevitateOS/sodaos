# Soda release build verification

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: 1 section, all rows verified; 2 progress-oracle rows retired (tests removed post-audit), H03 tail spans corrected.

<a id="coverage-944b552bf221"></a>

<a id="rustsoda-release-buildtestsoraclers-1"></a>

## [lib/soda-release-build/tests/oracle.rs](../../../../../lib/soda-release-build/tests/oracle.rs)

Re-audit @HEAD: import-reorder drift verified row-by-row; `oracle_progress_bytes`/`oracle_exit_codes` retired by C08 commit 3aaec1c8 (evidenced-dead build progress/clock mirror D03-E4; no successor in this file); `oracle_misc_vectors` + `oracle_read_json_strictness` cover the tail.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–8 | Captured Go vectors and native production fixtures; declarations/fields: `oracle_vectors`, `FIXTURE_REVISION`, `data_path`, `scratch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 9–28 | Captured Go vectors and native production fixtures; declaration/member oracle_vectors; declarations/fields: `oracle_vectors` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 28–29 | Captured Go vectors and native production fixtures; declaration/member FIXTURE_REVISION; declarations/fields: `FIXTURE_REVISION` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 30–35 | Captured Go vectors and native production fixtures; declaration/member data_path; declarations/fields: `data_path` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 36–50 | Captured Go vectors and native production fixtures; declaration/member scratch; declarations/fields: `scratch` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 51–62 | OCI archive/layout/content identity oracle assertions; declarations/fields: `oracle_go_fixture_identity` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 63–96 | OCI archive/layout/content identity oracle assertions; declaration/member oracle_go_fixture_rejections; declarations/fields: `oracle_go_fixture_rejections` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 97–124 | OCI archive/layout/content identity oracle assertions; declaration/member oracle_go_fixture_content; declarations/fields: `oracle_go_fixture_content` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 125–151 | OCI archive/layout/content identity oracle assertions; declaration/member oracle_go_layout; declarations/fields: `oracle_go_layout` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 152–187 | Pinned live-input/base record byte oracle; declarations/fields: `oracle_live_inputs` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 188–219 | Pinned live-input/base record byte oracle; declaration/member oracle_live_inputs_bytes; declarations/fields: `oracle_live_inputs_bytes` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 220–231 | Pinned live-input/base record byte oracle; declaration/member oracle_verified_base_bytes; declarations/fields: `oracle_verified_base_bytes` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 232–244 | Pinned live-input/base record byte oracle; declaration/member oracle_resolved_inputs_bytes; declarations/fields: `oracle_resolved_inputs_bytes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 245–256 | Native fork build argv/toolchain/production oracle; declarations/fields: `oracle_forgejo_argv_and_script` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 257–269 | Native fork build argv/toolchain/production oracle; declaration/member oracle_forgejo_toolchain_bytes; declarations/fields: `oracle_forgejo_toolchain_bytes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 270–280 | Native fork build argv/toolchain/production oracle; declaration/member fixture_elf; declarations/fields: `fixture_elf` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 281–426 | Native fork build argv/toolchain/production oracle; declaration/member oracle_production_sequence; declarations/fields: `oracle_production_sequence` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 427–449 | Encoding and strict JSON oracle; declarations/fields: `oracle_misc_vectors` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 451–476 | Encoding and strict JSON oracle; declaration/member oracle_read_json_strictness; declarations/fields: `oracle_read_json_strictness` |


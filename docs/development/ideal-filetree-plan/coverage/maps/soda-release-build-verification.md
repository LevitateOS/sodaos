# Soda release build verification

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-944b552bf221"></a>
<a id="rustsoda-release-buildtestsoraclers-1"></a>
<a id="coverage-c2d198be3994"></a>

## [lib/soda-release-build/tests/oracle.rs](../../../../../lib/soda-release-build/tests/oracle.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–44; lines 1–9: oracle_vectors and attached behavior; attached module comments and attributes; lines 18–19: mod inputs; lines 20–21: mod oci; lines 22–22: mod production; lines 23–24: FIXTURE_REVISION and attached behavior; lines 25–30: data_path and attached behavior; lines 31–44: scratch and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Captured Go vectors and native production fixtures; declaration/member oracle_vectors Adjacent comments and attributes explain this same authored responsibility.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 45–79; lines 45–79: fn oracle_live_inputs | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Source-level oracle/test assertions for pinned live-input/base record byte oracle — lib/soda-release-build/tests/oracle.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-edaf27b977d3"></a>

## [lib/soda-release-build/tests/oracle/inputs.rs](../../../../../lib/soda-release-build/tests/oracle/inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–119; oracle_live_inputs_write_and_read; oracle_forgejo_argv_and_script; oracle_forgejo_toolchain_bytes; oracle_misc_vectors; oracle_read_json_strictness | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Live inputs retain actual write/read/refusal checks; obsolete VerifiedBase/resolved-input formatter goldens are removed; Forgejo toolchain and non-emitter vectors remain |

<a id="coverage-99e034d4c9ba"></a>

## [lib/soda-release-build/tests/oracle/oci.rs](../../../../../lib/soda-release-build/tests/oracle/oci.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; lines 1–4: use super and attached body; lines 5–5: use soda_release_build and attached body; lines 6–6: use soda_release_build and attached body; lines 7–9: use std and attached body; lines 10–21: fn oracle_go_fixture_identity and attached body; lines 22–55: fn oracle_go_fixture_rejections and attached body; lines 56–83: fn oracle_go_fixture_content and attached body; lines 84–109: fn oracle_go_layout and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e3be2bc21d28"></a>

## [lib/soda-release-build/tests/oracle/production.rs](../../../../../lib/soda-release-build/tests/oracle/production.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–198; fixture_elf; oracle_production_sequence and current app-inputs semantic assertion | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Full production sequence remains; app-inputs are parsed from emitted bytes and checked for four resolved records, config digests, final LF and private 0600 mode, without a stale Go-format hash vector |

<a id="coverage-b77fc5491484"></a>

## [lib/soda-release-build/tests/oracle_vectors.rs](../../../../../lib/soda-release-build/tests/oracle_vectors.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–182; retained OCI/archive, command/script, error and progress vectors; retired live/verified/resolved/app-input serializer byte constants removed | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Remaining constants serve their actual reader/admission or command consumers; no release-build producer formatter oracle is assigned here |

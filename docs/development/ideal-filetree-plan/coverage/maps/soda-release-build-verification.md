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
| 1–148; lines 1–4: use super and attached body; lines 5–5: use soda_build_tools and attached body; lines 6–6: use soda_release_build and attached body; lines 7–7: use soda_release_build and attached body; lines 8–10: use soda_release_build and attached body; lines 11–11: use soda_release_build and attached body; lines 12–12: use soda_release_build and attached body; lines 13–13: use soda_release_build and attached body; lines 14–14: use soda_release_build and attached body; lines 15–15: use soda_release_build and attached body; lines 16–16: use soda_release_build and attached body; lines 17–19: use std and attached body; lines 20–51: fn oracle_live_inputs_bytes and attached body; lines 52–63: fn oracle_verified_base_bytes and attached body; lines 64–74: fn oracle_resolved_inputs_bytes and attached body; lines 75–86: fn oracle_forgejo_argv_and_script and attached body; lines 87–100: fn oracle_forgejo_toolchain_bytes and attached body; lines 101–125: fn oracle_misc_vectors and attached body; lines 126–148: fn oracle_read_json_strictness and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

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
| 1–170; lines 1–4: use super and attached body; lines 5–5: use soda_release_build and attached body; lines 6–6: use soda_release_build and attached body; lines 7–7: use soda_release_build and attached body; lines 8–8: use std and attached body; lines 9–9: use std and attached body; lines 10–21: fn fixture_elf and attached body; lines 22–170: fn oracle_production_sequence and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b77fc5491484"></a>

## [lib/soda-release-build/tests/oracle_vectors.rs](../../../../../lib/soda-release-build/tests/oracle_vectors.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; lines 1–10: const OCI_MANIFEST and attached body; lines 11–12: const OCI_CONFIG and attached body; lines 13–13: const OCI_ARCH and attached body; lines 14–14: const OCI_REVISION and attached body; lines 15–15: const OCI_SOURCE and attached body; lines 16–16: const OCI_BASENAME and attached body; lines 17–18: const OCI_BASEDIGEST and attached body; lines 19–19: const OCI_ERR_ARCH and attached body; lines 20–20: const OCI_ERR_REVISION and attached body; lines 21–21: const OCI_ERR_TAMPERED and attached body; lines 22–23: const OCI_FIXTURE_HASH and attached body; lines 24–24: const OCI_ERR_MISSING and attached body; lines 25–25: const LAYOUT_IMAGES and attached body; lines 26–26: const LAYOUT_FILES and attached body; lines 27–27: const LAYOUT_BYTES and attached body; lines 28–29: const LAYOUT_INDEXHASH and attached body; lines 30–30: const LAYOUT_ERR_ARCH and attached body; lines 31–62: const LIVE_INPUTS_JSON and attached body; lines 63–63: const LIVE_ERR_TAILNET and attached body; lines 64–64: const LIVE_ERR_RELEASE and attached body; lines 65–65: const LIVE_ERR_CONTAINER and attached body; lines 66–74: const VERIFIED_BASE_JSON and attached body; lines 75–82: const RESOLVED_INPUTS_JSON and attached body; lines 83–99: const FORGEJO_ARGV and attached body; lines 100–125: const FORGEJO_SCRIPT and attached body; lines 126–134: const FORGEJO_TOOLCHAIN_JSON and attached body; lines 135–204: const PRODUCTION_SEQUENCE and attached body; lines 205–227: const APP_INPUTS_JSON and attached body; lines 228–242: const PROGRESS_BYTES and attached body; lines 243–244: const EXIT_NIL and attached body; lines 245–245: const EXIT_7 and attached body; lines 246–246: const EXIT_SIGTERM and attached body; lines 247–247: const EXIT_CANCELED and attached body; lines 248–248: const EXIT_GENERIC and attached body; lines 249–249: const MISC_OCI_ARCH and attached body; lines 250–250: const MISC_HTTPS_HTTP and attached body; lines 251–251: const MISC_TAILNET and attached body; lines 252–252: const READ_UNKNOWN_FIELD and attached body; lines 253–253: const READ_TRAILING and attached body; lines 254–254: const READ_OK_MODE and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for const OCI_MANIFEST in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 40 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

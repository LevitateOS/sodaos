# Soda image import

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-2cbea9d84c63"></a>

## [cmd/soda-image-import/src/context.rs](../../../../../cmd/soda-image-import/src/context.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–8: use super and attached body; lines 9–40: fn run and attached body; lines 41–47: fn admit and attached body; lines 48–53: static CANCELLED and attached body; lines 54–65: fn install_cancel_handlers and attached body; lines 66–69: struct ImportCtx and attached body; lines 70–71: impl ImportCtx and attached body; lines 72–81: fn check and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6eb3ec79c9bb"></a>

## [cmd/soda-image-import/src/import.rs](../../../../../cmd/soda-image-import/src/import.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–171; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–9: use super and attached body; lines 10–32: fn bind_image_revisions and attached body; lines 33–52: fn match_layout_identities and attached body; lines 53–67: fn verify_content and attached body; lines 68–76: enum PodmanOutcome and attached body; lines 77–111: fn import_missing_image and attached body; lines 112–127: fn import_images and attached body; lines 128–160: fn run_podman and attached body; lines 161–171: fn native_import and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-de7ec15b18db"></a>

## [cmd/soda-image-import/src/json.rs](../../../../../cmd/soda-image-import/src/json.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–34; lines 1–1: use serde and attached body; lines 2–2: use serde_json and attached body; lines 3–10: fn parse_json and attached body; lines 11–14: fn json_valid and attached body; lines 15–22: fn raw_string and attached body; lines 23–34: fn raw_int and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d8de478c4f3b"></a>

## [cmd/soda-image-import/src/main.rs](../../../../../cmd/soda-image-import/src/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; lines 1–11: use std and attached body; lines 12–12: use std and attached body; lines 13–13: use std and attached body; lines 14–14: use std and attached body; lines 15–15: use std and attached body; lines 16–17: mod context and attached body; lines 18–18: mod import and attached body; lines 19–19: mod json and attached body; lines 20–20: mod oci and attached body; lines 21–21: mod payload and attached body; lines 22–22: mod platform and attached body; lines 23–23: mod sha256 and attached body; lines 24–25: use context and attached body; lines 26–26: use import and attached body; lines 27–27: use json and attached body; lines 28–28: use oci and attached body; lines 29–29: use payload and attached body; lines 30–36: use platform and attached body; lines 37–37: fn geteuid and attached body; lines 38–39: fn signal and attached body; lines 40–41: const RELEASE_PATH and attached body; lines 42–42: const IMAGES_PATH and attached body; lines 43–44: const PODMAN and attached body; lines 45–45: const IMPORT_TIMEOUT and attached body; lines 46–46: const SIGINT and attached body; lines 47–47: const SIGTERM and attached body; lines 48–55: const NAMES and attached body; lines 56–61: fn main and attached body; lines 62–62: mod tests and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 29 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-28b8faa7be6f"></a>

## [cmd/soda-image-import/src/oci/inspection.rs](../../../../../cmd/soda-image-import/src/oci/inspection.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–112; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use super and attached body; lines 5–8: use super and attached body; lines 9–11: use super and attached body; lines 12–56: fn inspect_oci_image and attached body; lines 57–112: fn inspect_oci_layout and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a9104b89fdb8"></a>

## [cmd/soda-image-import/src/oci/layout.rs](../../../../../cmd/soda-image-import/src/oci/layout.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–205; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–7: use crate and attached body; lines 8–8: use sha2 and attached body; lines 9–10: use super and attached body; lines 11–11: use super and attached body; lines 12–14: use super and attached body; lines 15–20: struct LayoutLoader and attached body; lines 21–22: impl LayoutLoader and attached body; lines 23–35: fn load and attached body; lines 36–52: fn fetch and attached body; lines 53–63: fn open_layout_root and attached body; lines 64–90: fn read_layout_blob and attached body; lines 91–137: fn read_blob_bytes and attached body; lines 138–153: fn validate_oci_layers and attached body; lines 154–172: fn validate_oci_rootfs and attached body; lines 173–205: fn validate_oci_attribution and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e9676646a8c8"></a>

## [cmd/soda-image-import/src/oci/metadata.rs](../../../../../cmd/soda-image-import/src/oci/metadata.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–196; lines 1–1: use crate and attached body; lines 2–2: use serde and attached body; lines 3–3: use serde and attached body; lines 4–4: use serde_json and attached body; lines 5–8: use std and attached body; lines 9–18: struct OciImage and attached body; lines 19–26: struct OciDescriptor and attached body; lines 27–30: struct OciBlobData and attached body; lines 31–35: macro_rules! raw_record and attached body; lines 36–36: fn deserialize and attached body; lines 37–37: struct V and attached body; lines 38–38: fn expecting and attached body; lines 39–58: fn visit_map and attached body; lines 59–67: fn string_list and attached body; lines 68–79: fn string_map and attached body; lines 80–90: fn descriptor and attached body; lines 91–105: fn descriptors and attached body; lines 106–109: struct OciManifestData and attached body; lines 110–127: fn parse_oci_manifest and attached body; lines 128–134: struct OciConfigData and attached body; lines 135–167: fn parse_oci_config and attached body; lines 168–196: fn read_oci_index and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7bec728dcfe5"></a>

## [cmd/soda-image-import/src/oci/mod.rs](../../../../../cmd/soda-image-import/src/oci/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–6; lines 1–1: mod inspection and attached body; lines 2–2: mod layout and attached body; lines 3–3: mod metadata and attached body; lines 4–5: use inspection and attached body; lines 6–6: use metadata and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod inspection in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod layout in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod metadata in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-da54f845ff4e"></a>

## [cmd/soda-image-import/src/payload.rs](../../../../../cmd/soda-image-import/src/payload.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–330; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–7: use serde and attached body; lines 8–8: use serde and attached body; lines 9–9: use serde_json and attached body; lines 10–11: use super and attached body; lines 12–17: use super and attached body; lines 18–25: struct ImageBinding and attached body; lines 26–39: struct Payload and attached body; lines 40–47: struct ImageBindingWire and attached body; lines 48–48: fn deserialize and attached body; lines 49–50: struct V and attached body; lines 51–51: type Value and attached body; lines 52–54: fn expecting and attached body; lines 55–85: fn visit_map and attached body; lines 86–100: struct PayloadWire and attached body; lines 101–101: fn deserialize and attached body; lines 102–103: struct V and attached body; lines 104–104: type Value and attached body; lines 105–107: fn expecting and attached body; lines 108–176: fn visit_map and attached body; lines 177–186: fn decode_image_binding and attached body; lines 187–190: fn decode_images and attached body; lines 191–192: struct ImageMap and attached body; lines 193–193: fn deserialize and attached body; lines 194–195: struct V and attached body; lines 196–196: type Value and attached body; lines 197–199: fn expecting and attached body; lines 200–219: fn visit_map and attached body; lines 220–245: fn decode_payload and attached body; lines 246–247: impl Payload and attached body; lines 248–253: fn valid_identity and attached body; lines 254–265: fn valid_base and attached body; lines 266–288: fn valid_images and attached body; lines 289–302: fn validate and attached body; lines 303–309: fn load and attached body; lines 310–330: fn read_bounded_json and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 40 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b57c4b40ffd2"></a>

## [cmd/soda-image-import/src/platform.rs](../../../../../cmd/soda-image-import/src/platform.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–84; lines 1–9: fn oci_architecture and attached body; lines 10–24: fn require_native and attached body; lines 25–28: fn is_lower_hex and attached body; lines 29–32: fn is_digest and attached body; lines 33–36: fn is_revision and attached body; lines 37–42: fn is_prefixed_digest and attached body; lines 43–54: fn is_coreos_version and attached body; lines 55–84: fn valid_repository_prefix and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for fn oci_architecture in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b5530597c99c"></a>

## [cmd/soda-image-import/src/sha256.rs](../../../../../cmd/soda-image-import/src/sha256.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; lines 1–4: use sha2 and attached body; lines 5–5: use sha2 and attached body; lines 6–7: fn hex_lower and attached body; lines 8–17: const HEX and attached body; lines 18–20: fn sha256_hex and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use sha2 in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a75d6d475d6d"></a>

## [cmd/soda-image-import/src/tests/fixtures.rs](../../../../../cmd/soda-image-import/src/tests/fixtures.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–215; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–4: use crate and attached body; lines 5–6: static TEST_SEQ and attached body; lines 7–17: fn test_root and attached body; lines 18–21: fn repeat and attached body; lines 22–30: fn test_ctx and attached body; lines 31–35: struct LayoutImage and attached body; lines 36–45: fn write_blob and attached body; lines 46–116: fn build_layout and attached body; lines 117–133: fn full_fixture and attached body; lines 134–167: fn test_payload and attached body; lines 168–215: fn payload_json_for and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use std in the current fixture-or-asset source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-047e09840043"></a>

## [cmd/soda-image-import/src/tests/import.rs](../../../../../cmd/soda-image-import/src/tests/import.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–217; lines 1–1: use super and attached body; lines 2–4: use super and attached body; lines 5–62: fn content_imports_exact_local_references and attached body; lines 63–109: fn content_refuses_whole_layout_before_any_import and attached body; lines 110–170: fn native_failures_remain_unconfirmed_without_replay and attached body; lines 171–174: fn fake_podman and attached body; lines 175–180: use std and attached body; lines 181–217: fn runner_maps_exits_and_kills_on_deadline and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b5ff7f333c1c"></a>

## [cmd/soda-image-import/src/tests/mod.rs](../../../../../cmd/soda-image-import/src/tests/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; lines 1–1: mod fixtures and attached body; lines 2–2: mod import and attached body; lines 3–3: mod oci and attached body; lines 4–4: mod payload and attached body; lines 5–5: mod primitives and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for mod fixtures in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-12ed5be34568"></a>

## [cmd/soda-image-import/src/tests/oci.rs](../../../../../cmd/soda-image-import/src/tests/oci.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–162; lines 1–1: use super and attached body; lines 2–4: use super and attached body; lines 5–34: fn layout_preserves_identities_and_counts and attached body; lines 35–108: fn mutate_layout and attached body; lines 109–114: fn set_field and attached body; lines 115–122: fn set_annotation and attached body; lines 123–126: fn set_size and attached body; lines 127–134: fn set_urls and attached body; lines 135–140: fn set_media and attached body; lines 141–162: fn layout_refuses_substitution and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3e9d649dc64d"></a>

## [cmd/soda-image-import/src/tests/payload.rs](../../../../../cmd/soda-image-import/src/tests/payload.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–8: fn decode and attached body; lines 9–60: fn payload_decode_is_strict_like_disallow_unknown_fields and attached body; lines 61–108: fn payload_validation_rejects_go_test_mutations and attached body; lines 109–146: fn payload_load_enforces_regular_bounded_input and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1db0b51eeb96"></a>

## [cmd/soda-image-import/src/tests/primitives.rs](../../../../../cmd/soda-image-import/src/tests/primitives.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use crate and attached body; lines 4–6: use sha2 and attached body; lines 7–14: fn admission_requires_root_and_no_arguments and attached body; lines 15–48: fn identifier_shapes_match_go_regexps and attached body; lines 49–84: fn sha256_matches_fips_vectors_streamed_and_oneshot and attached body; lines 85–93: fn native_platform_matches_go_checks and attached body; lines 94–102: fn json_validity_matches_single_value_rule and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-0d1bc1fb1f1d"></a>
<a id="rustsoda-image-importsrcmainrs-1"></a>

Former source `rust/soda-image-import/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

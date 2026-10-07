# Soda release deliver implementation

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-cb1363ce762f"></a>

## [lib/soda-release-deliver/src/admission.rs](../../../../../lib/soda-release-deliver/src/admission.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–290; lines 1–3: use serde and attached body; lines 4–4: use std and attached body; lines 5–6: use sha2 and attached body; lines 7–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–16: use crate and attached body; lines 17–21: const QUALIFICATION_SCOPE and attached body; lines 22–29: struct EvidenceCheck and attached body; lines 30–35: impl EvidenceCheck and attached body; lines 36–62: struct QualificationEvidence and attached body; lines 63–64: impl QualificationEvidence and attached body; lines 65–71: fn required_qualification_checks and attached body; lines 72–82: fn admit_release_identity and attached body; lines 83–88: fn sha256_hex and attached body; lines 89–95: fn decode_qualification_evidence and attached body; lines 96–113: fn admit_evidence_shape and attached body; lines 114–134: fn admit_evidence_check and attached body; lines 135–145: fn admit_evidence_checks and attached body; lines 146–159: fn load_admitted_candidate and attached body; lines 160–177: fn admit_evidence_candidate and attached body; lines 178–199: fn admit_evidence_media and attached body; lines 200–227: fn admit_qualification and attached body; lines 228–228: mod tests and attached body; lines 229–231: use super and attached body; lines 232–250: fn release_identity_rules and attached body; lines 251–290: fn evidence_shape_and_checks and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1ce663b5b2fe"></a>

## [lib/soda-release-deliver/src/buildx/filesystem.rs](../../../../../lib/soda-release-deliver/src/buildx/filesystem.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–370; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use sha2 and attached body; lines 6–7: use crate and attached body; lines 8–13: fn os_error and attached body; lines 14–16: struct Root and attached body; lines 17–18: impl Root and attached body; lines 19–22: fn open and attached body; lines 23–27: fn fd and attached body; lines 28–44: fn open_dir_fd and attached body; lines 45–64: fn check_confined and attached body; lines 65–68: fn c_string and attached body; lines 69–78: fn fstatat_no_follow and attached body; lines 79–87: fn fstat_fd and attached body; lines 88–93: fn is_regular and attached body; lines 94–127: fn open_confined_regular and attached body; lines 128–158: fn read_layout_entry and attached body; lines 159–165: use std and attached body; lines 166–196: fn read_at and attached body; lines 197–208: use std and attached body; lines 209–215: fn hash_at and attached body; lines 216–217: struct HashWriter and attached body; lines 218–219: impl std and attached body; lines 220–223: fn write and attached body; lines 224–227: fn flush and attached body; lines 228–244: fn clean_path and attached body; lines 245–257: fn fresh_directory and attached body; lines 258–265: use std and attached body; lines 266–276: fn private_destination and attached body; lines 277–287: use std and attached body; lines 288–288: fn write_new and attached body; lines 289–295: use std and attached body; lines 296–309: use std and attached body; lines 310–327: fn decode_build_json and attached body; lines 328–358: fn read_json_at and attached body; lines 359–370: use std and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 37 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c776df4ebf01"></a>

## [lib/soda-release-deliver/src/buildx/mod.rs](../../../../../lib/soda-release-deliver/src/buildx/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–108; lines 1–7: use serde and attached body; lines 8–9: use crate and attached body; lines 10–11: use soda_build_tools and attached body; lines 12–13: mod filesystem and attached body; lines 14–18: use filesystem and attached body; lines 19–21: const FORGEJO_COMPILER_IMAGE and attached body; lines 22–22: const FORGEJO_BUN_VERSION and attached body; lines 23–24: const FORGEJO_BUN_SHA256 and attached body; lines 25–25: const FORGEJO_UPSTREAM_BASE and attached body; lines 26–30: const FORGEJO_COMPAT_TOKEN and attached body; lines 31–46: struct Image and attached body; lines 47–52: impl Image and attached body; lines 53–60: struct ForgejoToolchain and attached body; lines 61–62: impl ForgejoToolchain and attached body; lines 63–74: fn validate and attached body; lines 75–93: fn valid_forgejo_apk_list and attached body; lines 94–107: fn has_forgejo_native_build_tools and attached body; lines 108–108: mod tests and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-441468063b86"></a>

## [lib/soda-release-deliver/src/buildx/tests.rs](../../../../../lib/soda-release-deliver/src/buildx/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; lines 1–3: use super and attached body; lines 4–39: fn toolchain_validation_matches_go and attached body; lines 40–55: fn file_helpers_match_go_errors and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn toolchain_validation_matches_go in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn file_helpers_match_go_errors in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a100fd477b8d"></a>

## [lib/soda-release-deliver/src/check.rs](../../../../../lib/soda-release-deliver/src/check.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39; lines 1–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–9: use crate and attached body; lines 10–31: fn check_candidate and attached body; lines 32–32: mod tests and attached body; lines 33–35: use super and attached body; lines 36–39: fn missing_candidate_dir_errors and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d236306c7263"></a>

## [lib/soda-release-deliver/src/content.rs](../../../../../lib/soda-release-deliver/src/content.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–69; lines 1–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–30: fn bind_image_revisions and attached body; lines 31–47: fn match_layout_identities and attached body; lines 48–59: fn verify_content and attached body; lines 60–60: mod tests and attached body; lines 61–63: use super and attached body; lines 64–69: fn content_path_rules and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0de803ef2a57"></a>

## [lib/soda-release-deliver/src/document.rs](../../../../../lib/soda-release-deliver/src/document.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–391; lines 1–3: use serde and attached body; lines 4–4: use serde and attached body; lines 5–5: use serde_json and attached body; lines 6–6: use std and attached body; lines 7–7: use tar and attached body; lines 8–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–12: const MANIFEST_TYPE and attached body; lines 13–16: const LAYER_TYPE and attached body; lines 17–24: struct Descriptor and attached body; lines 25–31: fn emit_descriptor and attached body; lines 32–42: fn put_document_blob and attached body; lines 43–79: fn document_layer and attached body; lines 80–92: fn write_document_index and attached body; lines 93–121: fn write_document_blobs and attached body; lines 122–123: fn set_mode and attached body; lines 124–129: use std and attached body; lines 130–139: fn write_document and attached body; lines 140–140: fn marshal_go_pretty and attached body; lines 141–141: struct GoFormatter and attached body; lines 142–142: impl Formatter and attached body; lines 143–145: fn begin_array and attached body; lines 146–148: fn end_array and attached body; lines 149–155: fn begin_array_value and attached body; lines 156–158: fn end_array_value and attached body; lines 159–161: fn begin_object and attached body; lines 162–164: fn end_object and attached body; lines 165–171: fn begin_object_key and attached body; lines 172–174: fn begin_object_value and attached body; lines 175–177: fn end_object_value and attached body; lines 178–200: fn write_string_fragment and attached body; lines 201–220: fn write_char_escape and attached body; lines 221–231: fn read_file and attached body; lines 232–235: fn read_json and attached body; lines 236–254: fn decode_document_manifest and attached body; lines 255–264: struct OciManifestDoc and attached body; lines 265–276: fn read_document_blob and attached body; lines 277–291: fn read_document_record and attached body; lines 292–304: use std and attached body; lines 305–315: fn read_document and attached body; lines 316–316: mod tests and attached body; lines 317–317: use super and attached body; lines 318–318: use crate and attached body; lines 319–319: use std and attached body; lines 320–334: fn temp_dir and attached body; lines 335–362: fn document_round_trip_and_bounds and attached body; lines 363–391: fn document_layer_is_deterministic_ustar_at_block_boundaries and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 47 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-195bd684346c"></a>

## [lib/soda-release-deliver/src/fetch/mod.rs](../../../../../lib/soda-release-deliver/src/fetch/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–114; lines 1–3: use serde and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–11: mod state and attached body; lines 12–12: mod verification and attached body; lines 13–14: use state and attached body; lines 15–15: use verification and attached body; lines 16–39: fn discover and attached body; lines 40–47: fn init_state and attached body; lines 48–58: fn fetch_document and attached body; lines 59–68: fn admit_fetch_request and attached body; lines 69–91: fn complete_fetch and attached body; lines 92–113: fn fetch and attached body; lines 114–114: mod tests and attached body | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6882fab32a6b"></a>

## [lib/soda-release-deliver/src/fetch/state.rs](../../../../../lib/soda-release-deliver/src/fetch/state.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use serde and attached body; lines 6–10: struct StateLock and attached body; lines 11–12: impl std and attached body; lines 13–16: fn fmt and attached body; lines 17–18: impl Drop and attached body; lines 19–24: fn drop and attached body; lines 25–28: fn lock_state and attached body; lines 29–42: use std and attached body; lines 43–54: fn create_temp and attached body; lines 55–62: fn save_state and attached body; lines 63–75: use std and attached body | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-655e47785659"></a>

## [lib/soda-release-deliver/src/fetch/tests.rs](../../../../../lib/soda-release-deliver/src/fetch/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–46; lines 1–1: use super and attached body; lines 2–12: fn private_dir and attached body; lines 13–18: use std and attached body; lines 19–22: fn state_lock_serializes_and_saves and attached body; lines 23–41: use std and attached body; lines 42–46: fn fetch_request_admission and attached body | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-797b62aba0dc"></a>

## [lib/soda-release-deliver/src/fetch/verification.rs](../../../../../lib/soda-release-deliver/src/fetch/verification.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–344; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use serde and attached body; lines 5–5: use serde and attached body; lines 6–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–14: use super and attached body; lines 15–27: fn decode_raw and attached body; lines 28–32: struct ImageMetadata and attached body; lines 33–33: fn deserialize and attached body; lines 34–35: struct V and attached body; lines 36–36: type Value and attached body; lines 37–39: fn expecting and attached body; lines 40–63: fn visit_map and attached body; lines 64–67: struct ImageManifest and attached body; lines 68–68: fn deserialize and attached body; lines 69–70: struct V and attached body; lines 71–71: type Value and attached body; lines 72–74: fn expecting and attached body; lines 75–93: fn visit_map and attached body; lines 94–97: struct ImageConfig and attached body; lines 98–98: fn deserialize and attached body; lines 99–100: struct V and attached body; lines 101–101: type Value and attached body; lines 102–104: fn expecting and attached body; lines 105–121: fn visit_map and attached body; lines 122–138: fn resolve_verification_architectures and attached body; lines 139–149: fn check_verified_image_metadata and attached body; lines 150–161: fn check_verified_image_manifest and attached body; lines 162–162: mod json_slot_tests and attached body; lines 163–165: use super and attached body; lines 166–186: fn fetch_metadata_uses_final_exact_values_before_typed_conversion and attached body; lines 187–205: fn verify_release_image_copy and attached body; lines 206–217: fn build_expected_release_configs and attached body; lines 218–256: fn verify_release_images and attached body; lines 257–261: struct ReleaseIdentityTracker and attached body; lines 262–263: impl ReleaseIdentityTracker and attached body; lines 264–277: fn check and attached body; lines 278–304: fn verify_architecture_release and attached body; lines 305–344: fn verify_releases and attached body | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 44 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fe28f76842f6"></a>
<a id="coverage-2bd1953b7092"></a>

## [lib/soda-release-deliver/src/finalize.rs](../../../../../lib/soda-release-deliver/src/finalize.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–190, 274–364; lines 16–28: fn decode_raw; attached module comments and attributes; lines 29–32: struct IndexDocument; lines 59–62: struct ManifestDescriptor; lines 94–190: struct Config, impl Config, fn admit_publication_inputs, fn admit_config, fn load_config, fn load_trust, fn load_signer, fn document_digest; lines 274–296: fn prepare_and_sign; lines 297–318: fn finalize; lines 319–319: mod tests; lines 323–336: fn publication_index_uses_final_exact_values_before_typed_conversion; lines 337–364: fn config_round_trip_and_admission | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Current fn decode_raw and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 9 named units assigned here; remaining selectors preserve each duty — lib/soda-release-deliver/src/finalize.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 191–227; lines 191–227: fn sign_digest, fn bind_channel_releases, fn channel_repository | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Native exact release digest signing — lib/soda-release-deliver/src/finalize.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 228–273; lines 228–261: fn publish_channel_last; lines 262–273: fn write_final_receipt | [D09](../../slices/release-and-installation.md#d09-publication-and-effect-observation) | retained | Release publication followed by channel publication; Final observed publication receipt — lib/soda-release-deliver/src/finalize.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-154077b76116"></a>

## [lib/soda-release-deliver/src/import.rs](../../../../../lib/soda-release-deliver/src/import.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; lines 1–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–8: use crate and attached body; lines 9–15: enum RunOutcome and attached body; lines 16–16: trait EngineRunner and attached body; lines 17–18: fn run and attached body; lines 19–55: fn import_missing_image and attached body; lines 56–63: fn import_images and attached body; lines 64–65: struct NativeEngine and attached body; lines 66–67: impl EngineRunner and attached body; lines 68–80: fn run and attached body; lines 81–85: fn native_import and attached body; lines 86–86: mod tests and attached body; lines 87–87: use super and attached body; lines 88–88: use std and attached body; lines 89–89: use std and attached body; lines 90–93: struct ScriptEngine and attached body; lines 94–95: impl EngineRunner and attached body; lines 96–103: fn run and attached body; lines 104–112: fn image and attached body; lines 113–169: fn import_exit_code_paths and attached body | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d8e1c8d53cad"></a>

## [lib/soda-release-deliver/src/json_serde.rs](../../../../../lib/soda-release-deliver/src/json_serde.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–229; lines 1–1: use serde and attached body; lines 2–2: use serde and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–8: use crate and attached body; lines 9–9: struct DecodeError and attached body; lines 10–14: const MAX_STRICT_BYTES and attached body; lines 15–29: fn strict and attached body; lines 30–32: struct RootSeed and attached body; lines 33–33: type Value and attached body; lines 34–34: fn deserialize and attached body; lines 35–36: struct RootVisitor and attached body; lines 37–37: type Value and attached body; lines 38–40: fn expecting and attached body; lines 41–55: fn visit_map and attached body; lines 56–58: struct RawObject and attached body; lines 59–59: fn deserialize and attached body; lines 60–61: struct V and attached body; lines 62–62: type Value and attached body; lines 63–65: fn expecting and attached body; lines 66–76: fn visit_map and attached body; lines 77–79: struct RawArray and attached body; lines 80–80: fn deserialize and attached body; lines 81–82: struct V and attached body; lines 83–83: type Value and attached body; lines 84–86: fn expecting and attached body; lines 87–97: fn visit_seq and attached body; lines 98–129: fn check_raw_value and attached body; lines 130–137: fn null_default and attached body; lines 138–139: fn null_i64 and attached body; lines 140–148: use serde and attached body; lines 149–150: fn null_u64 and attached body; lines 151–166: use serde and attached body; lines 167–170: fn null_u64_map and attached body; lines 171–191: use serde and attached body; lines 192–201: fn parse_integer and attached body; lines 202–202: mod tests and attached body; lines 203–203: use super and attached body; lines 204–206: use serde and attached body; lines 207–211: struct RawDocument and attached body; lines 212–229: fn strict_raw_values_preserve_numbers_and_reject_duplicates_and_depth and attached body | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Shared bounded strict JSON decoding, duplicate-key/depth validation, null/default handling, and exact integer conversion used by candidate admission, signing, publication ledger, model, and OCI callers. — current shared adapter lib/soda-release-deliver/src/json_serde.rs; callers in native/policy.rs, native/sign.rs, publish/ledger.rs, model/*, oci/schema.rs; current source inspected |

<a id="coverage-4fb86a3dd4f1"></a>
<a id="coverage-54ee85a788cb"></a>

## [lib/soda-release-deliver/src/lib.rs](../../../../../lib/soda-release-deliver/src/lib.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 14, 16, 18–19, 23, 25–89; lines 9–10: mod admission; attached module comments and attributes; lines 11–11: mod buildx; lines 14–14: mod document; lines 16–16: mod finalize; lines 18–18: mod json_serde; lines 19–19: mod model; lines 23–23: mod prepare | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Module wiring: admission; Release admission and preparation Adjacent comments and attributes explain this same authored responsibility.; 7 named units assigned here; remaining selectors preserve each duty — lib/soda-release-deliver/src/lib.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 12–13, 21–22; lines 12–12: mod check; lines 13–13: mod content; lines 21–21: mod oci; lines 22–22: mod payload | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Module wiring: check; Artifact verification; 4 named units assigned here; remaining selectors preserve each duty — lib/soda-release-deliver/src/lib.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 15, 20; lines 15–15: mod fetch; lines 20–20: mod native | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Module wiring: fetch; Verified distribution consumption; Module wiring: native; Verified distribution consumption — lib/soda-release-deliver/src/lib.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 17; lines 17–17: mod import | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Module wiring: import; Host installation and payload application — lib/soda-release-deliver/src/lib.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 24; lines 24–24: mod publish | [D09](../../slices/release-and-installation.md#d09-publication-and-effect-observation) | retained | Module wiring: publish; Publication and effect observation — lib/soda-release-deliver/src/lib.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-13abbd18c3e0"></a>

## [lib/soda-release-deliver/src/model/candidate.rs](../../../../../lib/soda-release-deliver/src/model/candidate.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–191; lines 1–1: use serde and attached body; lines 2–2: use std and attached body; lines 3–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–15: use crate and attached body; lines 16–45: struct Candidate and attached body; lines 46–55: fn valid_candidate_host and attached body; lines 56–63: fn valid_candidate_provenance and attached body; lines 64–71: fn valid_candidate_source and attached body; lines 72–80: fn valid_candidate_build and attached body; lines 81–104: fn valid_candidate_content and attached body; lines 105–124: fn required_candidate_content and attached body; lines 125–155: fn path_clean and attached body; lines 156–163: fn valid_candidate_asset_name and attached body; lines 164–178: fn known_candidate_content_name and attached body; lines 179–180: impl Candidate and attached body; lines 181–191: fn validate and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-063b6641e62b"></a>

## [lib/soda-release-deliver/src/model/channel.rs](../../../../../lib/soda-release-deliver/src/model/channel.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–228; lines 1–1: use serde and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–14: use super and attached body; lines 15–31: struct Channel and attached body; lines 32–45: fn nil_if_empty and attached body; lines 46–56: struct Seen and attached body; lines 57–70: struct Highwater and attached body; lines 71–72: impl Channel and attached body; lines 73–76: impl Seen and attached body; lines 77–82: fn empty_state and attached body; lines 83–86: fn valid_highwater_maps and attached body; lines 87–99: fn valid_highwater_channels and attached body; lines 100–112: fn valid_highwater_serials and attached body; lines 113–114: impl Highwater and attached body; lines 115–124: fn validate and attached body; lines 125–135: fn validate_release_reference and attached body; lines 136–151: fn validate_channel_releases and attached body; lines 152–169: fn validate_channel_identity and attached body; lines 170–187: fn validate_channel_timing and attached body; lines 188–199: fn validate_channel_progression and attached body; lines 200–228: fn admit_channel and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9ebbc2fa6300"></a>

## [lib/soda-release-deliver/src/model/mod.rs](../../../../../lib/soda-release-deliver/src/model/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; lines 1–3: use serde and attached body; lines 4–5: use crate and attached body; lines 6–7: mod trust and attached body; lines 8–8: use trust and attached body; lines 9–10: mod candidate and attached body; lines 11–11: use candidate and attached body; lines 12–12: use candidate and attached body; lines 13–14: mod release and attached body; lines 15–15: use release and attached body; lines 16–16: use release and attached body; lines 17–18: mod channel and attached body; lines 19–26: use channel and attached body; lines 27–38: struct Permit and attached body; lines 39–40: impl Permit and attached body; lines 41–54: fn validate and attached body; lines 55–55: mod tests and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-91fcbd720161"></a>

## [lib/soda-release-deliver/src/model/release.rs](../../../../../lib/soda-release-deliver/src/model/release.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–415; whole file: release and media binding data model, json decoding, provenance/evidence admission and release identity checks for delivery preparation | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Release and media binding data model, JSON decoding, provenance/evidence admission and release identity checks for delivery preparation. — lib/soda-release-deliver/src/model/release.rs; consumed by release preparation/finalization |

<a id="coverage-e734d538a9aa"></a>

## [lib/soda-release-deliver/src/model/tests.rs](../../../../../lib/soda-release-deliver/src/model/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–23: fn full_content and attached body; lines 24–49: fn candidate_content_rules and attached body; lines 50–61: fn path_clean_matches_go and attached body; lines 62–82: fn trust_reference_shapes and attached body; lines 83–107: fn trust_key_admission_preserves_cross_role_separation and attached body; lines 108–124: fn producer_trust_fixture_keeps_its_raw_der_fingerprint and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7e6295e02021"></a>

## [lib/soda-release-deliver/src/model/trust.rs](../../../../../lib/soda-release-deliver/src/model/trust.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–108; lines 1–1: use serde and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–12: use crate and attached body; lines 13–30: struct Trust and attached body; lines 31–38: fn valid_trust_timing and attached body; lines 39–49: fn valid_trust_envelope and attached body; lines 50–53: fn parse_trust_public_key and attached body; lines 54–67: fn admit_trust_role_keys and attached body; lines 68–69: impl Trust and attached body; lines 70–84: fn validate and attached body; lines 85–98: fn role and attached body; lines 99–108: fn reference and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-90df2ad9f827"></a>

## [lib/soda-release-deliver/src/native/mod.rs](../../../../../lib/soda-release-deliver/src/native/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–217; lines 1–3: use std and attached body; lines 4–5: use serde and attached body; lines 6–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–14: const TOOL_LOCK and attached body; lines 15–19: struct ToolLock and attached body; lines 20–20: trait Runner and attached body; lines 21–25: fn run and attached body; lines 26–28: struct Native and attached body; lines 29–30: impl Runner and attached body; lines 31–50: fn run and attached body; lines 51–61: fn check_native and attached body; lines 62–82: fn owned_private_regular and attached body; lines 83–92: fn private_file and attached body; lines 93–97: fn write_json and attached body; lines 98–99: mod policy and attached body; lines 100–100: use policy and attached body; lines 101–101: use policy and attached body; lines 102–102: use policy and attached body; lines 103–127: fn admit_verify_source and attached body; lines 128–156: fn verify_copy and attached body; lines 157–164: fn verify_copied_manifest and attached body; lines 165–166: mod sign and attached body; lines 167–169: use sign and attached body; lines 170–170: mod tests and attached body; lines 171–171: use super and attached body; lines 172–175: struct MockRunner and attached body; lines 176–177: impl Runner and attached body; lines 178–183: fn run and attached body; lines 184–198: fn check_native_pins_version and attached body; lines 199–206: fn private_file_rules and attached body; lines 207–217: use std and attached body | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 34 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9f60c4c21b96"></a>

## [lib/soda-release-deliver/src/native/policy.rs](../../../../../lib/soda-release-deliver/src/native/policy.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–348; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use base64 and attached body; lines 6–6: use serde and attached body; lines 7–7: use serde and attached body; lines 8–8: use serde and attached body; lines 9–9: use std and attached body; lines 10–12: use std and attached body; lines 13–22: struct Requirement and attached body; lines 23–30: struct PolicyDocument and attached body; lines 31–37: enum ScopeValue and attached body; lines 38–45: enum PolicyValue and attached body; lines 46–48: struct PolicyPairs and attached body; lines 49–49: fn deserialize and attached body; lines 50–51: struct V and attached body; lines 52–52: type Value and attached body; lines 53–55: fn expecting and attached body; lines 56–66: fn visit_map and attached body; lines 67–69: struct PolicyItems and attached body; lines 70–70: fn deserialize and attached body; lines 71–72: struct V and attached body; lines 73–73: type Value and attached body; lines 74–76: fn expecting and attached body; lines 77–87: fn visit_seq and attached body; lines 88–89: impl PolicyValue and attached body; lines 90–121: fn from_raw and attached body; lines 122–128: fn is_null and attached body; lines 129–132: fn deserialize and attached body; lines 133–134: impl Serialize and attached body; lines 135–157: fn serialize and attached body; lines 158–164: fn simple_rule and attached body; lines 165–184: fn requirement and attached body; lines 185–193: fn policy_for_publish and attached body; lines 194–197: fn registry_config_for_publish and attached body; lines 198–213: fn policy_for and attached body; lines 214–226: fn local_policy and attached body; lines 227–240: fn soda_trust_repos and attached body; lines 241–247: fn soda_override_exists and attached body; lines 248–268: fn apply_soda_trust and attached body; lines 269–276: struct PolicyInput and attached body; lines 277–299: fn merge_policy and attached body; lines 300–312: struct MergedPolicy and attached body; lines 313–317: fn write_registry_config and attached body; lines 318–321: fn registry_config and attached body; lines 322–348: use std and attached body | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 47 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6bd2b49fec62"></a>

## [lib/soda-release-deliver/src/native/sign.rs](../../../../../lib/soda-release-deliver/src/native/sign.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; lines 1–1: use serde and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–13: use super and attached body; lines 14–19: struct SecretFiles and attached body; lines 20–21: impl SecretFiles and attached body; lines 22–33: fn admit_sign_inputs and attached body; lines 34–59: fn snapshot_sign_source and attached body; lines 60–76: fn admit_signed_payload and attached body; lines 77–121: fn emit_signed_directory and attached body; lines 122–135: fn sign and attached body | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d146d1379967"></a>

## [lib/soda-release-deliver/src/oci/archive.rs](../../../../../lib/soda-release-deliver/src/oci/archive.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–111; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–8: use super and attached body; lines 9–13: use super and attached body; lines 14–31: fn open_oci_archive and attached body; lines 32–41: fn is_valid_oci_regular_entry and attached body; lines 42–99: fn read_oci_archive_entries and attached body; lines 100–111: fn inspect_archive_index and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e14b7a71acd5"></a>

## [lib/soda-release-deliver/src/oci/layers.rs](../../../../../lib/soda-release-deliver/src/oci/layers.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–582; whole file: oci layer byte reader: bounded decompression/tar scanning, whiteout and ancestor replacement handling, member content digests and layer budgets | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | OCI layer byte reader: bounded decompression/tar scanning, whiteout and ancestor replacement handling, member content digests and layer budgets. — lib/soda-release-deliver/src/oci/layers.rs; consumed by delivery artifact inspection and verification |

<a id="coverage-b85f0a6ea5ed"></a>

## [lib/soda-release-deliver/src/oci/mod.rs](../../../../../lib/soda-release-deliver/src/oci/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–320; lines 1–6: use std and attached body; lines 7–7: use std and attached body; lines 8–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–15: const MANIFEST_TYPE and attached body; lines 16–16: const CONFIG_TYPE and attached body; lines 17–17: const INDEX_TYPE and attached body; lines 18–18: const LAYER_TAR and attached body; lines 19–19: const LAYER_GZIP and attached body; lines 20–23: const LAYER_ZSTD and attached body; lines 24–30: struct OciLayout and attached body; lines 31–35: struct Blob and attached body; lines 36–37: mod schema and attached body; lines 38–41: use schema and attached body; lines 42–43: mod archive and attached body; lines 44–46: use archive and attached body; lines 47–51: fn inspect_oci and attached body; lines 52–53: mod layers and attached body; lines 54–54: use layers and attached body; lines 55–59: use layers and attached body; lines 60–77: fn inspect_content_manifest and attached body; lines 78–87: fn inspect_oci_content and attached body; lines 88–94: use std and attached body; lines 95–96: type LayerScan and attached body; lines 97–138: fn scan_oci_archive_layers and attached body; lines 139–160: fn validate_oci_layout_inputs and attached body; lines 161–194: fn inspect_oci_layout and attached body; lines 195–200: struct LayoutLoader and attached body; lines 201–202: impl LayoutLoader and attached body; lines 203–209: fn load and attached body; lines 210–225: fn fetch and attached body; lines 226–253: fn inspect_image and attached body; lines 254–292: fn inspect_layout_images and attached body; lines 293–293: mod tests and attached body; lines 294–296: use super and attached body; lines 297–310: fn member_path_rules and attached body; lines 311–320: fn non_archive_refused and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 38 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9d8a4a85817d"></a>

## [lib/soda-release-deliver/src/oci/schema.rs](../../../../../lib/soda-release-deliver/src/oci/schema.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–598; whole file: oci index, manifest, config and descriptor decoding plus digest, media-type, layer and rootfs shape checks | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | OCI index, manifest, config and descriptor decoding plus digest, media-type, layer and rootfs shape checks. — lib/soda-release-deliver/src/oci/schema.rs; consumed by delivery OCI archive verifier |

<a id="coverage-3cb1c1b2ee1f"></a>
<a id="rustsoda-release-deliversrcpayloadrs-1"></a>
<a id="coverage-f2e054fb68a8"></a>

## [lib/soda-release-deliver/src/payload.rs](../../../../../lib/soda-release-deliver/src/payload.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–396; lines 10–11: PATH and attached behavior; attached module comments and attributes; lines 12–12: IMAGES_PATH and attached behavior; lines 13–24: NAMES and attached behavior; lines 32–42: fn decode_raw_default; lines 43–67: fn decode_images; lines 68–69: struct RawI64; lines 250–259: is_core_os_version and attached behavior; lines 260–263: is_digest_ref and attached behavior; lines 266–274: valid_identity and attached behavior; lines 275–293: valid_base and attached behavior; lines 294–319: valid_images and attached behavior; lines 320–343: validate and attached behavior; lines 344–379: valid_repository_prefix and attached behavior; lines 380–395: load and attached behavior; lines 396–396: tests and attached behavior | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Immutable payload identity and declared appliance image contracts; declaration/member PATH Adjacent comments and attributes explain this same authored responsibility.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0dec78b56389"></a>

## [lib/soda-release-deliver/src/payload/tests.rs](../../../../../lib/soda-release-deliver/src/payload/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–67; lines 1–3: use super and attached body; lines 4–19: fn repository_prefix_shapes and attached body; lines 20–39: fn payload_identity_errors_match_go and attached body; lines 40–55: fn payload_raw_slots_keep_only_the_last_exact_value and attached body; lines 56–67: fn images_map_decodes_only_the_last_duplicate_entry and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b84106523e8c"></a>
<a id="coverage-41b08e368ee7"></a>

## [lib/soda-release-deliver/src/prepare.rs](../../../../../lib/soda-release-deliver/src/prepare.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88, 229–288; lines 14–88: struct Qualification, fn hash_release_provenance, fn read_media_binding, fn build_release_document, fn prepare; attached module comments and attributes; lines 249–249: mod tests | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Qualified exact release input admission and immutable metadata preparation Adjacent comments and attributes explain this same authored responsibility.; Source-level oracle/test assertions for final immutable release tag/document binding — lib/soda-release-deliver/src/prepare.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 89–228; lines 89–228: fn candidate_image_content, fn candidate_content_with_embedded_inventory, fn inspect_candidate_image, fn verify_candidate_image, fn candidate_image_inputs, fn verify_candidate_images | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Candidate artifact and image content verification — lib/soda-release-deliver/src/prepare.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-39d3d2edaab1"></a>

## [lib/soda-release-deliver/src/publish/channel.rs](../../../../../lib/soda-release-deliver/src/publish/channel.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–195; lines 1–1: use std and attached body; lines 2–3: use serde and attached body; lines 4–4: use serde and attached body; lines 5–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–12: use super and attached body; lines 13–25: fn decode_raw and attached body; lines 26–30: struct TagList and attached body; lines 31–31: fn deserialize and attached body; lines 32–33: struct V and attached body; lines 34–34: type Value and attached body; lines 35–37: fn expecting and attached body; lines 38–61: fn visit_map and attached body; lines 62–62: mod json_slot_tests and attached body; lines 63–65: use super and attached body; lines 66–78: fn tag_list_uses_final_exact_values_before_typed_conversion and attached body; lines 79–97: fn validate_channel_history and attached body; lines 98–115: fn admit_channel_offer and attached body; lines 116–129: fn channel_tag_exists and attached body; lines 130–155: fn verify_previous_channel and attached body; lines 156–179: fn promote_channel and attached body; lines 180–195: use std and attached body | [D09](../../slices/release-and-installation.md#d09-publication-and-effect-observation) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3145622fe22b"></a>

## [lib/soda-release-deliver/src/publish/ledger.rs](../../../../../lib/soda-release-deliver/src/publish/ledger.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; lines 1–1: use serde and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–10: use crate and attached body; lines 11–22: struct Ledger and attached body; lines 23–24: impl Ledger and attached body; lines 25–37: fn validate and attached body; lines 38–54: fn init_ledger and attached body; lines 55–62: fn valid_ledger_phase and attached body | [D09](../../slices/release-and-installation.md#d09-publication-and-effect-observation) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b740f028d131"></a>

## [lib/soda-release-deliver/src/publish/mod.rs](../../../../../lib/soda-release-deliver/src/publish/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–313; lines 1–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–13: mod ledger and attached body; lines 14–14: use ledger and attached body; lines 15–47: fn upload and attached body; lines 48–50: fn policy_for_upload and attached body; lines 51–54: fn registry_config_for_upload and attached body; lines 55–79: fn observe_published and attached body; lines 80–98: fn admit_publish_ledger and attached body; lines 99–102: fn should_observe and attached body; lines 103–131: fn observe_only and attached body; lines 132–133: mod channel and attached body; lines 134–136: use channel and attached body; lines 137–169: fn admit_signed and attached body; lines 170–180: fn commit_immutable and attached body; lines 181–205: use std and attached body; lines 206–230: fn finalize_publication and attached body; lines 231–263: fn publish and attached body; lines 264–264: mod tests and attached body; lines 265–265: use super and attached body; lines 266–267: use super and attached body; lines 268–268: use crate and attached body; lines 269–277: fn trust and attached body; lines 278–300: fn ledger_phases and attached body; lines 301–313: fn channel_history_rules and attached body | [D09](../../slices/release-and-installation.md#d09-publication-and-effect-observation) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-74bad50280a1"></a>
<a id="rustsoda-release-deliversrcbuildxrs-1"></a>

Former source `lib/soda-release-deliver/src/buildx.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-452620fb57f9"></a>
<a id="rustsoda-release-deliversrcfetchrs-1"></a>

Former source `lib/soda-release-deliver/src/fetch.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-9cca680cf321"></a>
<a id="rustsoda-release-deliversrcjsonxrs-1"></a>

Former source `lib/soda-release-deliver/src/jsonx.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-be197ef17c55"></a>
<a id="rustsoda-release-deliversrcmodelrs-1"></a>

Former source `lib/soda-release-deliver/src/model.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-fc774294f0d5"></a>
<a id="rustsoda-release-deliversrcnativers-1"></a>

Former source `lib/soda-release-deliver/src/native.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-a060f97fea2b"></a>
<a id="rustsoda-release-deliversrcocirs-1"></a>

Former source `lib/soda-release-deliver/src/oci.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-b4303a0369b0"></a>
<a id="rustsoda-release-deliversrcpublishrs-1"></a>

Former source `lib/soda-release-deliver/src/publish.rs`; consult its pinned earlier Git source and the current coverage disposition.

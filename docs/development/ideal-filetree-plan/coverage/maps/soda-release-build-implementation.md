# Soda release build implementation

Current path navigation reconciled at `45ebf4c4` (2026-10-09); historical symbol/body selectors remain pinned to `519b76bd` unless a narrow current selector is explicitly stated.

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-47d643575161"></a>

## [lib/release-inputs/src/elf.rs](../../../../../lib/release-inputs/src/elf.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–45; lines 1–5: const HEADER_LEN and attached body; lines 6–9: struct Elf64Le and attached body; lines 10–21: fn elf64_le_header and attached body; lines 22–22: mod tests and attached body; lines 23–25: use super and attached body; lines 26–45: fn header_fields_require_a_complete_elf64_little_endian_prefix and attached body | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Declaration block for const HEADER_LEN in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f0e02079188c"></a>

## [lib/release-inputs/src/lib.rs](../../../../../lib/release-inputs/src/lib.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; lines 1–7: mod elf and attached body; lines 8–10: mod reader and attached body; lines 11–13: mod trust_key and attached body; lines 14–15: mod tests and attached body; lines 16–20: fn package_metadata_present and attached body | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Declaration block for mod elf in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7979fa8447df"></a>

## [lib/release-inputs/src/reader.rs](../../../../../lib/release-inputs/src/reader.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–86; lines 1–13: mod forgejo and attached body; lines 14–14: mod muse and attached body; lines 15–15: mod settings and attached body; lines 16–16: mod signature and attached body; lines 17–17: mod stream and attached body; lines 18–21: mod url and attached body; lines 22–22: struct Error and attached body; lines 23–24: impl std and attached body; lines 25–28: fn fmt and attached body; lines 29–32: impl std and attached body; lines 33–41: fn oci_architecture and attached body; lines 42–46: fn is_digest and attached body; lines 47–49: fn is_revision and attached body; lines 50–55: fn non_empty_digits and attached body; lines 56–56: mod tests and attached body; lines 57–59: use super and attached body; lines 60–72: fn architecture_names and attached body; lines 73–86: fn digest_and_revision_shapes and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for mod forgejo in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c7d5144480cc"></a>

## [lib/release-inputs/src/reader/forgejo.rs](../../../../../lib/release-inputs/src/reader/forgejo.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–159; lines 1–8: use super and attached body; lines 9–12: const FORGEJO_COMPILER_IMAGE and attached body; lines 13–15: const FORGEJO_BUN_VERSION and attached body; lines 16–19: const FORGEJO_BUN_SHA256 and attached body; lines 20–22: const FORGEJO_UPSTREAM_BASE and attached body; lines 23–26: const FORGEJO_COMPAT_TOKEN and attached body; lines 27–37: fn forgejo_version_stamp and attached body; lines 38–60: fn valid_apk_list and attached body; lines 61–73: fn has_native_build_tools and attached body; lines 74–78: struct ForgejoToolchain and attached body; lines 79–80: impl ForgejoToolchain and attached body; lines 81–92: fn validate and attached body; lines 93–93: mod tests and attached body; lines 94–94: use super and attached body; lines 95–104: fn toolchain and attached body; lines 105–117: fn version_stamp_carries_short_revision and attached body; lines 118–135: fn apk_list_requires_sorted_unique_names and attached body; lines 136–159: fn toolchain_validation_pins_image_and_native_tools and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fb2238fbfc33"></a>

## [lib/release-inputs/src/reader/muse.rs](../../../../../lib/release-inputs/src/reader/muse.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; lines 1–7: use super and attached body; lines 8–15: struct MuseArtifact and attached body; lines 16–35: fn valid_muse_version and attached body; lines 36–48: fn valid_muse_artifact and attached body; lines 49–49: mod tests and attached body; lines 50–50: use super and attached body; lines 51–60: fn artifact and attached body; lines 61–65: fn accepts_pinned_release and attached body; lines 66–96: fn rejects_bad_pins and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a93399baa385"></a>

## [lib/release-inputs/src/reader/settings.rs](../../../../../lib/release-inputs/src/reader/settings.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–158; lines 1–7: use super and attached body; lines 8–12: use std and attached body; lines 13–31: fn single_setting and attached body; lines 32–36: fn recipe_base and attached body; lines 37–41: fn unit_image and attached body; lines 42–55: fn valid_command_name and attached body; lines 56–80: fn soda_commands and attached body; lines 81–81: mod tests and attached body; lines 82–82: use super and attached body; lines 83–83: use std and attached body; lines 84–85: static NEXT_ID and attached body; lines 86–98: fn scratch_dir and attached body; lines 99–127: fn single_setting_reads_missing_and_duplicate and attached body; lines 128–158: fn soda_commands_inventories_sorted_commands and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-554e197a31e4"></a>

## [lib/release-inputs/src/reader/signature.rs](../../../../../lib/release-inputs/src/reader/signature.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; lines 1–16: const GNUPG_FAILURES and attached body; lines 17–29: fn valid_sig_match and attached body; lines 30–49: fn valid_signature and attached body; lines 50–50: mod tests and attached body; lines 51–51: use super and attached body; lines 52–55: const SIGNER and attached body; lines 56–62: fn accepts_matching_validsig and attached body; lines 63–68: fn accepts_subkey_validsig_in_long_form and attached body; lines 69–81: fn rejects_failures_and_mismatches and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for const GNUPG_FAILURES in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8e9ce09b49cb"></a>

## [lib/release-inputs/src/reader/stream.rs](../../../../../lib/release-inputs/src/reader/stream.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–311; lines 1–7: use super and attached body; lines 8–8: use super and attached body; lines 9–17: struct CoreOSImage and attached body; lines 18–34: fn valid_release and attached body; lines 35–64: fn valid_stream_images and attached body; lines 65–80: fn stream_release_url and attached body; lines 81–88: struct TailnetInputs and attached body; lines 89–99: fn valid_tailnet_version and attached body; lines 100–110: fn valid_tailnet_base and attached body; lines 111–123: fn valid_tailnet_inputs and attached body; lines 124–131: struct ResolvedCoreOS and attached body; lines 132–167: fn valid_resolved_coreos and attached body; lines 168–168: mod tests and attached body; lines 169–169: use super and attached body; lines 170–178: fn iso and attached body; lines 179–187: fn qemu and attached body; lines 188–202: fn resolved and attached body; lines 203–236: fn stream_images_require_https_triples and attached body; lines 237–252: fn release_url_derives_from_stream_endpoint and attached body; lines 253–280: fn tailnet_pins_require_version_checksum_and_base and attached body; lines 281–311: fn resolved_coreos_requires_metadata_and_pinned_base and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e746cd56f93a"></a>

## [lib/release-inputs/src/reader/url.rs](../../../../../lib/release-inputs/src/reader/url.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–104; lines 1–4: use url and attached body; lines 5–31: fn https_url and attached body; lines 32–52: fn valid_escapes and attached body; lines 53–53: mod tests and attached body; lines 54–56: use super and attached body; lines 57–75: fn admits_production_metadata_urls and attached body; lines 76–99: fn rejects_unsafe_public_inputs and attached body; lines 100–104: fn numeric_hosts_use_whatwg_ip_classification_for_admission and attached body | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Declaration block for use url in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b68884685c60"></a>

## [lib/release-inputs/src/trust_key.rs](../../../../../lib/release-inputs/src/trust_key.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–133; lines 1–5: use p256 and attached body; lines 6–7: const MAX_PEM_BYTES and attached body; lines 8–8: const BEGIN and attached body; lines 9–15: const END and attached body; lines 16–63: fn parse_p256_public_key and attached body; lines 64–64: mod tests and attached body; lines 65–65: use super and attached body; lines 66–66: use p256 and attached body; lines 67–70: const GENERATOR and attached body; lines 71–77: fn admits_p256_and_returns_the_decoded_spki and attached body; lines 78–100: fn rejects_off_curve_and_non_single_pem_envelopes and attached body; lines 101–133: fn rejects_non_p256_or_noncanonical_spki_encodings and attached body | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Declaration block for use p256 in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cc678946c824"></a>

## [lib/soda-release-build/src/confined_files.rs](../../../../../lib/soda-release-build/src/confined_files.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–190; lines 1–4: use crate and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–11: use std and attached body; lines 12–16: struct Root and attached body; lines 17–24: struct FileMeta and attached body; lines 25–32: fn stat_to_meta and attached body; lines 33–35: impl Root and attached body; lines 36–50: fn open and attached body; lines 51–52: fn resolve_parent and attached body; lines 53–89: use std and attached body; lines 90–111: fn lstat and attached body; lines 112–123: fn open_file and attached body; lines 124–134: fn open_unchanged and attached body; lines 135–148: fn open_at and attached body; lines 149–165: fn open_at_fd and attached body; lines 166–169: fn is_regular_meta and attached body; lines 170–171: fn same_file_meta and attached body; lines 172–176: use std and attached body; lines 177–190: fn hash_at and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0ee161fbce5f"></a>
<a id="rustsoda-release-buildsrccoreosrs-1"></a>
<a id="coverage-96d8cad69d60"></a>

## [lib/soda-release-build/src/coreos.rs](../../../../../lib/soda-release-build/src/coreos.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–500; whole file: pinned CoreOS admission, signature/checksum verification and download/decompression; verified-base JSON is standard pretty Serde with caller LF and mode 0600 | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Verified-base output preserves typed fields and existing writer custody |

<a id="coverage-78a0bcc6f698"></a>

## [lib/soda-release-build/src/coreos/process.rs](../../../../../lib/soda-release-build/src/coreos/process.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; lines 1–5: use crate and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–10: use std and attached body; lines 11–53: fn run_bounded and attached body; lines 54–74: use std and attached body; lines 75–114: use std and attached body; lines 115–129: fn set_nonblocking and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-23c7088e0101"></a>

## [lib/soda-release-build/src/coreos_iso.rs](../../../../../lib/soda-release-build/src/coreos_iso.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; whole file: verified ISO acquisition and verified-base receipt output; standard pretty Serde, final LF, exclusive mode 0600 writer | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Receipt fields and custody remain with the ISO acquisition owner |

<a id="coverage-67290639f2af"></a>

## [lib/soda-release-build/src/coreos_registry.rs](../../../../../lib/soda-release-build/src/coreos_registry.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–196; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use serde and attached body; lines 9–9: use serde and attached body; lines 10–10: use std and attached body; lines 11–11: use std and attached body; lines 12–12: use std and attached body; lines 13–14: const COREOS_CONTAINER_REPO and attached body; lines 15–15: const COREOS_CONTAINER_TAG and attached body; lines 16–86: fn resolve_registry_digests_with and attached body; lines 87–90: struct ManifestIndex and attached body; lines 91–95: struct ManifestRecord and attached body; lines 96–100: struct ManifestPlatform and attached body; lines 101–101: fn deserialize and attached body; lines 102–103: struct V and attached body; lines 104–104: type Value and attached body; lines 105–107: fn expecting and attached body; lines 108–130: fn visit_map and attached body; lines 131–131: fn deserialize and attached body; lines 132–133: struct V and attached body; lines 134–134: type Value and attached body; lines 135–137: fn expecting and attached body; lines 138–168: fn visit_map and attached body; lines 169–169: fn deserialize and attached body; lines 170–171: struct V and attached body; lines 172–172: type Value and attached body; lines 173–175: fn expecting and attached body; lines 176–196: fn visit_map and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8edea7777a05"></a>
<a id="rustsoda-release-buildsrccoreos_streamrs-1"></a>
<a id="coverage-940ce722ac2c"></a>

## [lib/soda-release-build/src/coreos_stream.rs](../../../../../lib/soda-release-build/src/coreos_stream.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–425; whole file: coreos stream and tailnet live-input document parsing, resolution and locked input serialization | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | CoreOS stream and Tailnet live-input document parsing, resolution and locked input serialization. — lib/soda-release-build/src/coreos_stream.rs; consumed by release build input resolver |

<a id="coverage-efc01c53b2bb"></a>

## [lib/soda-release-build/src/elf.rs](../../../../../lib/soda-release-build/src/elf.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–70; lines 1–4: use crate and attached body; lines 5–5: use soda_build_tools and attached body; lines 6–6: use std and attached body; lines 7–9: use std and attached body; lines 10–27: fn inspect_elf and attached body; lines 28–34: fn is_native_executable and attached body; lines 35–35: mod tests and attached body; lines 36–36: use super and attached body; lines 37–48: fn fixture_elf and attached body; lines 49–70: fn native_elf_header_profile and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-495c86a56818"></a>
<a id="rustsoda-release-buildsrcfilesrs-1"></a>
<a id="coverage-a877363f9a56"></a>

## [lib/soda-release-build/src/files.rs](../../../../../lib/soda-release-build/src/files.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–196; File strict deserializer, integer/duplicate policy, architecture/digest/revision admission, hash and private/exclusive filesystem primitives | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | File remains a reader; its unused serializer is retired. Existing reader and custody tests remain |

<a id="coverage-07ec951e4416"></a>

## [lib/soda-release-build/src/files/tests.rs](../../../../../lib/soda-release-build/src/files/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–167; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–4: use std and attached body; lines 5–11: fn scratch and attached body; lines 12–13: static COUNT and attached body; lines 14–18: fn unique and attached body; lines 19–33: fn validators_match_go and attached body; lines 34–54: fn hash_file_refuses_symlinks and attached body; lines 55–89: fn fresh_and_private_directories and attached body; lines 90–101: fn write_new_is_exclusive and attached body; lines 102–124: fn read_json_strict_round_trip and attached body; lines 125–139: fn file_duplicate_fields_validate_only_the_last_match and attached body; lines 140–154: fn file_mode_keeps_go_integer_token_edges and attached body; lines 155–167: fn root_rejects_symlink_escape and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7d78a9120c73"></a>

## [lib/soda-release-build/src/forgejo.rs](../../../../../lib/soda-release-build/src/forgejo.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–337; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use serde and attached body; lines 8–8: use serde and attached body; lines 9–9: use serde and attached body; lines 10–10: use std and attached body; lines 11–18: use soda_build_tools and attached body; lines 19–30: fn forgejo_version_stamp and attached body; lines 31–34: struct ForgejoToolchain and attached body; lines 35–36: impl ForgejoToolchain and attached body; lines 37–45: fn validate and attached body; lines 47–49: ForgejoToolchain::marshal using standard Serde pretty JSON; lines 51–52: impl Serialize and attached body; lines 53–61: fn serialize and attached body; lines 62–62: fn deserialize and attached body; lines 63–64: struct ToolchainVisitor and attached body; lines 65–65: type Value and attached body; lines 66–68: fn expecting and attached body; lines 69–104: fn visit_map and attached body; lines 105–122: fn record_forgejo_toolchain and attached body; lines 123–133: fn build_forgejo_binary and attached body; lines 134–140: fn validate_forgejo_build and attached body; lines 141–193: fn forgejo_build_args and attached body; lines 194–203: fn inspect_forgejo_build and attached body; lines 204–216: fn stage_fork_binary and attached body; lines 217–217: mod tests and attached body; lines 218–218: use super and attached body; lines 219–221: use crate and attached body; lines 222–234: fn oracle_version_stamp_vectors and attached body; lines 235–272: fn oracle_toolchain_provenance_vectors and attached body; lines 273–337: fn oracle_stage_fork_binary_wiring and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 33 named units assigned here; remaining selectors preserve each duty; three existing stage/writer/readback checks pass — Current named units/source consumers |

<a id="coverage-94cc6ab4739c"></a>

## [lib/soda-release-build/src/http.rs](../../../../../lib/soda-release-build/src/http.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–339; lines 1–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use std and attached body; lines 12–14: use std and attached body; lines 15–20: struct HttpResponse and attached body; lines 21–23: impl HttpResponse and attached body; lines 24–27: fn status_line and attached body; lines 28–29: impl std and attached body; lines 30–39: fn fmt and attached body; lines 40–40: trait HttpTransport and attached body; lines 41–49: fn get and attached body; lines 50–50: struct UreqTransport and attached body; lines 51–52: impl HttpTransport and attached body; lines 53–78: fn get and attached body; lines 79–87: fn ureq_response and attached body; lines 88–93: fn is_redirect and attached body; lines 94–102: fn split_url and attached body; lines 103–135: fn resolve_location and attached body; lines 136–163: fn get_follow and attached body; lines 164–197: fn fetch_capped_json and attached body; lines 198–207: fn fetch_capped_text and attached body; lines 208–208: mod tests and attached body; lines 209–209: use super and attached body; lines 210–210: use std and attached body; lines 211–213: use std and attached body; lines 214–217: struct Stub and attached body; lines 218–219: impl Stub and attached body; lines 220–234: fn new and attached body; lines 235–236: impl HttpTransport and attached body; lines 237–262: fn get and attached body; lines 263–275: fn reason and attached body; lines 276–320: fn oracle_redirect_policy_vectors and attached body; lines 321–339: fn oracle_location_resolution_vectors and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 33 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Retired source: `lib/soda-release-build/src/json_emit.rs`

Removed in `5186c2eb` after its last callers selected standard Serde at concrete record boundaries. This retirement is specific to release-build; delivery signed-record, release-image ordered/raw and Acceptance report serializers retain their own profiles.

<a id="coverage-f954a3ba4108"></a>

## [lib/soda-release-build/src/json_input.rs](../../../../../lib/soda-release-build/src/json_input.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use serde and attached body; lines 7–7: use serde and attached body; lines 8–8: use std and attached body; lines 9–13: use std and attached body; lines 14–23: fn parse_integer_token and attached body; lines 24–35: fn read_json and attached body; lines 36–39: fn read_json_at and attached body; lines 40–62: const LIMIT and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-33ce1bf6ebad"></a>
<a id="coverage-4963a8a36cc2"></a>

## [lib/soda-release-build/src/lib.rs](../../../../../lib/soda-release-build/src/lib.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 10, 15, 17, 19, 23–27, 30–254; modules confined_files, elf, forgejo, json_input, production and release-build helpers; crate Error and pipeline functions | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Module wiring and common build/pipeline operations; `json_emit` is removed and no release-build generic formatter is allocated |
| 11–14, 20, 28; modules coreos, coreos_iso, coreos_registry, coreos_stream, live_inputs and tailnet_inputs | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Pinned input acquisition and reader ownership |
| 16; module files | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Configuration and filesystem primitives |
| 18; module http | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private IPC and service lifetime |
| 21–22; modules oci and oci_layout | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Artifact verification |

<a id="coverage-3bf5a40bd502"></a>

## [lib/soda-release-build/src/live_inputs.rs](../../../../../lib/soda-release-build/src/live_inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–331; Tailnet/CoreOS input DTOs and reader policy; ResolvedCoreOS sorts maps before Serde; write_live_inputs validates then emits standard pretty JSON plus caller LF through exclusive mode 0644 writer | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Actual reader/refusal tests remain; the generic Go formatter and LiveInputs marshal shim are retired |

<a id="coverage-b01b99e67b72"></a>
<a id="rustsoda-release-buildsrcocirs-1"></a>
<a id="coverage-9f9af3cb3e11"></a>

## Former source `lib/soda-release-build/src/oci.rs` (body snapshot `519b76bd`)

Current canonical OCI archive implementation is under [lib/soda-release-deliver/src/oci/](../../../../../lib/soda-release-deliver/src/oci/mod.rs): module, [archive](../../../../../lib/soda-release-deliver/src/oci/archive.rs), [layers](../../../../../lib/soda-release-deliver/src/oci/layers.rs), and [schema](../../../../../lib/soda-release-deliver/src/oci/schema.rs).

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–253; OCI archive identity/content admission, raw descriptor decoding and Image output; compact standard Serde preserves declared field order, with newline owned by CLI caller | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Existing identity, admission and archive tests remain; retired Go escaping has no external exact-byte contract |

<a id="coverage-1511215e39c6"></a>

## Former source `lib/soda-release-build/src/oci/archive.rs` (body snapshot `519b76bd`)

Current canonical OCI archive reader: [lib/soda-release-deliver/src/oci/archive.rs](../../../../../lib/soda-release-deliver/src/oci/archive.rs).

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–224; lines 1–4: use super and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use std and attached body; lines 8–8: use std and attached body; lines 9–9: use std and attached body; lines 10–10: use std and attached body; lines 11–27: fn open_oci_archive and attached body; lines 28–37: fn check_tar_entry_name and attached body; lines 38–47: fn check_tar_directory and attached body; lines 48–57: fn is_valid_oci_regular_entry and attached body; lines 58–71: fn record_tar_entry and attached body; lines 72–76: fn copy_oci_blob and attached body; lines 77–122: use sha2 and attached body; lines 123–132: fn is_json_bytes and attached body; lines 133–158: fn read_oci_blob and attached body; lines 159–165: fn entry_raw_name and attached body; lines 166–188: fn read_archive_blob_entry and attached body; lines 189–224: fn read_oci_archive_entries and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7445f5823747"></a>

## Former source `lib/soda-release-build/src/oci/content.rs` (body snapshot `519b76bd`)

Current content/layer handling: [archive.rs](../../../../../lib/soda-release-deliver/src/oci/archive.rs) and [layers.rs](../../../../../lib/soda-release-deliver/src/oci/layers.rs).

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–238; lines 1–4: use super and attached body; lines 5–5: use super and attached body; lines 6–6: use crate and attached body; lines 7–7: use sha2 and attached body; lines 8–11: use soda_release_deliver and attached body; lines 12–12: use std and attached body; lines 13–15: use std and attached body; lines 16–16: type LayerScans and attached body; lines 17–54: fn layer_archive_indexes and attached body; lines 55–62: struct HashReader and attached body; lines 63–70: fn new and attached body; lines 71–77: fn hex and attached body; lines 78–94: fn read and attached body; lines 95–99: use sha2 and attached body; lines 100–109: fn scan_layer_reader and attached body; lines 110–117: fn scan_archive_layer and attached body; lines 118–172: fn scan_archive_layer_with_budget and attached body; lines 173–205: fn scan_oci_archive_layers and attached body; lines 206–238: fn resolve_oci_members and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3bd66e4590dd"></a>

## Former source `lib/soda-release-build/src/oci/manifest.rs` (body snapshot `519b76bd`)

Current OCI schema/layer owners: [schema.rs](../../../../../lib/soda-release-deliver/src/oci/schema.rs) and [layers.rs](../../../../../lib/soda-release-deliver/src/oci/layers.rs).

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–318; lines 1–7: use super and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use serde and attached body; lines 11–11: use serde and attached body; lines 12–12: use std and attached body; lines 13–16: type RawJson and attached body; lines 17–19: struct IntegerToken and attached body; lines 20–28: fn deserialize and attached body; lines 29–32: struct LayoutRecord and attached body; lines 33–37: struct IndexRecord and attached body; lines 38–43: struct ManifestRecord and attached body; lines 44–49: struct ConfigRecord and attached body; lines 50–53: struct RootfsRecord and attached body; lines 54–56: struct ConfigSection and attached body; lines 57–60: macro_rules! serde_record and attached body; lines 61–63: fn deserialize and attached body; lines 64–64: type Value and attached body; lines 65–65: fn expecting and attached body; lines 66–96: fn visit_map and attached body; lines 97–119: fn read_oci_index and attached body; lines 120–124: struct OciManifest and attached body; lines 125–145: fn parse_oci_manifest and attached body; lines 146–166: fn fetch_oci_blob and attached body; lines 167–181: fn validate_oci_layers and attached body; lines 182–197: fn validate_oci_rootfs and attached body; lines 198–225: fn validate_oci_attribution and attached body; lines 226–278: fn inspect_oci_config and attached body; lines 279–302: fn inspect_oci_image and attached body; lines 303–303: mod tests and attached body; lines 304–306: use super and attached body; lines 307–318: fn schema_version_keeps_go_integer_tokens and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 32 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-89b047cdc223"></a>

## Former source `lib/soda-release-build/src/oci/tests.rs` (body snapshot `519b76bd`)

Old build-side OCI fixtures/tests were retired; current production parser is under [delivery OCI archive](../../../../../lib/soda-release-deliver/src/oci/archive.rs) and [layers](../../../../../lib/soda-release-deliver/src/oci/layers.rs), without asserting fixture equivalence.

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–397; OCI archive identity, content and scanner tests; old exact Go-escape output test removed | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Actual OCI input admission and archive behavior remain tested; CLI identity semantics are checked by the release-tools caller test |

<a id="coverage-7f8ee5523495"></a>

## Former source `lib/soda-release-build/src/oci_layout.rs` (body snapshot `519b76bd`)

The old layout inspector was retired. Current OCI archive/schema parsing lives at [archive.rs](../../../../../lib/soda-release-deliver/src/oci/archive.rs) and [schema.rs](../../../../../lib/soda-release-deliver/src/oci/schema.rs); not a claim that the inspector was replaced one-for-one.

Historical locator and selector/body evidence are pinned to `519b76bd`; these selectors do not describe a current path or current body.

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–361; lines 1–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use std and attached body; lines 12–16: use std and attached body; lines 17–21: struct OciLayout and attached body; lines 22–23: fn same_opened and attached body; lines 24–26: use std and attached body; lines 27–46: fn validate_oci_layout_inputs and attached body; lines 47–57: fn open_oci_layout_root and attached body; lines 58–113: fn inspect_oci_layout and attached body; lines 114–138: fn load_layout_entry and attached body; lines 139–139: mod tests and attached body; lines 140–140: use super and attached body; lines 141–141: use crate and attached body; lines 142–142: use crate and attached body; lines 143–256: fn layout_fixture and attached body; lines 257–281: fn oracle_layout_identities_and_counts and attached body; lines 282–361: fn oracle_layout_refuses_substitution and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9258418e94ee"></a>
<a id="rustsoda-release-buildsrcproductionrs-1"></a>
<a id="coverage-c5999e83f1b9"></a>

## [lib/soda-release-build/src/production.rs](../../../../../lib/soda-release-build/src/production.rs)

Current cohesive production adapter body inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; whole file: native image production adapter interface and image export implementation | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Defines the production inputs and execution hooks, validates native build context, exports the four frozen input images, builds the exact Forgejo fork binary, stages that binary, and returns the image archive digest. It does not sign, publish, or install the candidate. — lib/soda-release-build/src/production.rs:1-109; module docs plus Production::validate/images/build_forgejo_binary/stage_fork_binary; consumer lib/soda-release-image/src/build.rs and release pipeline |

<a id="coverage-8bca6aadcbf5"></a>

## [lib/soda-release-build/src/production/tests.rs](../../../../../lib/soda-release-build/src/production/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–328; lines 1–1: use super and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use std and attached body; lines 8–10: use std and attached body; lines 11–147: fn production_fixture and attached body; lines 148–212: fn oracle_production_sequence and attached body; lines 213–232: fn oracle_production_failure_stops and attached body; lines 233–268: fn oracle_production_refusals and attached body; lines 269–283: fn oracle_asset_destinations_refuse_early and attached body; lines 284–319: fn oracle_compile_recipes and attached body; lines 320–328: fn oracle_image_repo_parsing and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-99b094ad7919"></a>

## [lib/soda-release-build/src/production_assets.rs](../../../../../lib/soda-release-build/src/production_assets.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–167; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use std and attached body; lines 7–9: impl Production and attached body; lines 10–36: fn assets and attached body; lines 37–167: fn asset_steps and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0d1c0b41caeb"></a>

## [lib/soda-release-build/src/production_compile.rs](../../../../../lib/soda-release-build/src/production_compile.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–137; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use serde and attached body; lines 7–7: use serde and attached body; lines 8–8: use std and attached body; lines 9–13: impl Production and attached body; lines 14–47: fn compile_rust and attached body; lines 48–67: fn compile and attached body; lines 68–84: fn require_pinned_bun and attached body; lines 85–102: fn dependencies and attached body; lines 103–108: struct PackageManifest and attached body; lines 109–109: fn deserialize and attached body; lines 110–111: struct V and attached body; lines 112–112: type Value and attached body; lines 113–115: fn expecting and attached body; lines 116–137: fn visit_map and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ec912acb4c19"></a>

## [lib/soda-release-build/src/production_images.rs](../../../../../lib/soda-release-build/src/production_images.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–275; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use soda_build_tools and attached body; lines 10–10: use std and attached body; lines 11–13: use std and attached body; lines 14–14: type PullFn and attached body; lines 15–15: type BuildFn and attached body; lines 16–19: type ExportFn and attached body; lines 20–35: fn lexical_rel and attached body; lines 36–37: impl Production and attached body; lines 38–80: fn build_image and attached body; lines 81–112: fn export_image_archive and attached body; lines 113–130: fn resolve_rocky_base and attached body; lines 131–158: fn export_app_images and attached body; lines 159–173: fn export_forgejo_image and attached body; lines 174–182: fn export_proxy_image and attached body; lines 183–205: fn export_tailnet_image and attached body; lines 206–223: fn export_extension_image and attached body; lines 224–246: fn export_images and attached body; lines 247–275: macro_rules! exporting and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-03c616be8c09"></a>

## [lib/soda-release-build/src/production_inputs.rs](../../../../../lib/soda-release-build/src/production_inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–177; resolved image DTO, pull/digest admission and live settings; app-inputs record uses standard pretty Serde, caller LF and private 0600 output | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Fresh app-input provenance hash is derived from emitted bytes; production checks retain resolved records and config digests without a stale Go-emitter hash fixture |

<a id="coverage-f10ae67c9613"></a>

## [lib/soda-release-build/src/tailnet_inputs.rs](../../../../../lib/soda-release-build/src/tailnet_inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–234; lines 1–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use serde and attached body; lines 11–14: use serde and attached body; lines 15–17: fn resolve_tailnet_inputs and attached body; lines 18–42: fn resolve_tailnet_inputs_with and attached body; lines 43–74: fn latest_tailnet_release and attached body; lines 75–104: fn parse_tailnet_release_at and attached body; lines 105–109: fn latest_tailnet_release_with and attached body; lines 110–145: fn latest_tailnet_base_tag_with and attached body; lines 146–150: struct TagDocument and attached body; lines 151–151: fn deserialize and attached body; lines 152–153: struct DocVisitor and attached body; lines 154–154: type Value and attached body; lines 155–157: fn expecting and attached body; lines 158–181: fn visit_map and attached body; lines 182–187: struct TagRecord and attached body; lines 188–188: fn deserialize and attached body; lines 189–190: struct RecordVisitor and attached body; lines 191–191: type Value and attached body; lines 192–194: fn expecting and attached body; lines 195–217: fn visit_map and attached body; lines 218–218: mod tests and attached body; lines 219–221: use super and attached body; lines 222–234: fn oracle_tailnet_release_selection and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 28 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b5c8027c5016"></a>

## [lib/soda-release-build/src/test_support.rs](../../../../../lib/soda-release-build/src/test_support.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–108; lines 1–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use std and attached body; lines 9–13: const FIXTURE_REVISION and attached body; lines 14–73: fn fixture_oci_bytes and attached body; lines 74–108: fn fixture_live_inputs and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-ea777fc0e833"></a>
<a id="rustsoda-release-buildsrcjson_gors-1"></a>

Former source `lib/soda-release-build/src/json_go.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-876126f7558f"></a>
<a id="rustsoda-release-buildsrcprogressrs-1"></a>

Former source `rust/soda-release-build/src/progress.rs`; consult its pinned earlier Git source and the current coverage disposition.


## Current path reconciliation at `45ebf4c4`

This section reconciles current path ownership only. Existing numeric/body selectors above remain pinned to `519b76bd` unless a row explicitly gives a narrow current inspection.


<a id="r02-current-path-lib-release-inputs-src-reader-image-rs"></a>

### [lib/release-inputs/src/reader/image.rs](../../../../../lib/release-inputs/src/reader/image.rs)

D07 canonical Image type and codec; D02 remains reader parent shell.

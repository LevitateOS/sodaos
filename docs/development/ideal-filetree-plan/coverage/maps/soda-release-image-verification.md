# Soda release image verification

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-16cef9a7838a"></a>
<a id="rustsoda-release-imagetestsoraclers-1"></a>
<a id="coverage-587a4579ef3d"></a>

## [lib/soda-release-image/tests/oracle.rs](../../../../../lib/soda-release-image/tests/oracle.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; lines 9–12: b64 and attached behavior; attached module comments and attributes; lines 13–16: check_ok and attached behavior; lines 17–22: check_err and attached behavior; lines 23–24: mod host; lines 25–26: mod media; lines 27–27: mod staging | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Complete native candidate image assembly and authenticated live media; declaration/member b64 Adjacent comments and attributes explain this same authored responsibility.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6c5b8d4c3deb"></a>

## [lib/soda-release-image/tests/oracle/host.rs](../../../../../lib/soda-release-image/tests/oracle/host.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–193; lines 1–3: use super and attached body; lines 4–73: fn oracle_local_quadlet and attached body; lines 74–74: fn oracle_live_ignition and attached body; lines 75–75: use flate2 and attached body; lines 76–76: use flate2 and attached body; lines 77–106: use std and attached body; lines 107–124: fn oracle_package_inputs and attached body; lines 125–157: fn oracle_rpm_inventory and attached body; lines 158–193: fn oracle_candidate_live_config and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-92e513159821"></a>

## [lib/soda-release-image/tests/oracle/media.rs](../../../../../lib/soda-release-image/tests/oracle/media.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–256; lines 1–1: use super and attached body; lines 2–4: use soda_release_image and attached body; lines 5–96: fn oracle_media_base_url and attached body; lines 97–154: fn oracle_media_compression and attached body; lines 155–222: fn oracle_media_log_events and attached body; lines 223–256: fn oracle_rootfs_chunks and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c26997a18b5a"></a>

## [lib/soda-release-image/tests/oracle/staging.rs](../../../../../lib/soda-release-image/tests/oracle/staging.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–221; lines 1–3: use super and attached body; lines 4–29: fn oracle_extension_asset_names and attached body; lines 30–52: fn oracle_qualify_reason and attached body; lines 53–54: struct Stub and attached body; lines 55–55: impl soda_release_image and attached body; lines 56–58: fn source and attached body; lines 59–61: fn forgejo_source and attached body; lines 62–64: fn forgejo_revision and attached body; lines 65–67: fn native and attached body; lines 68–70: fn out and attached body; lines 71–73: fn arch and attached body; lines 74–76: fn revision and attached body; lines 77–79: fn live_inputs and attached body; lines 80–87: fn execute and attached body; lines 88–95: fn capture and attached body; lines 96–98: fn next and attached body; lines 99–101: fn resolve_inputs and attached body; lines 102–104: fn dependencies and attached body; lines 105–107: fn compile and attached body; lines 108–115: fn compile_rust and attached body; lines 116–118: fn stage_fork_binary and attached body; lines 119–121: fn assets and attached body; lines 122–130: fn images and attached body; lines 131–138: fn inspect_oci and attached body; lines 139–146: fn verify_content and attached body; lines 147–149: fn resolve_core_os and attached body; lines 150–155: fn read_live_inputs and attached body; lines 156–158: fn check_native and attached body; lines 159–170: fn sign_media and attached body; lines 171–180: fn verify_copy and attached body; lines 181–190: fn write_document and attached body; lines 191–221: fn oracle_stage_layout and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 32 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

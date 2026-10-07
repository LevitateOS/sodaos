# Soda release image implementation

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting updated selectively after `274621ce` (2026-10-08); other rows retain their recorded snapshot.
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-0387b9fa196a"></a>
<a id="rustsoda-release-imagesrcbuildrs-1"></a>
<a id="coverage-702496be1cd8"></a>

## [lib/soda-release-image/src/build.rs](../../../../../lib/soda-release-image/src/build.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–316, 324–326, 330–453; lines 31–34: PINNED_GO_VERSION and attached behavior; attached module comments and attributes; lines 35–48: ProductionInputs and attached behavior; lines 49–63: build and attached behavior; lines 71–103: extract_build_snapshot and attached behavior; lines 104–129: setup_build_workspace and attached behavior; lines 130–159: prepare_build_production and attached behavior; lines 160–196: execute_build_production and attached behavior; lines 197–214: finalize_build and attached behavior; lines 215–241: run_build and attached behavior; lines 242–254: run_build_inner and attached behavior; lines 259–259: impl Production; lines 260–262: source and attached behavior; lines 263–265: forgejo_source and attached behavior; lines 266–268: forgejo_revision and attached behavior; lines 269–271: native and attached behavior; lines 272–274: out and attached behavior; lines 275–277: arch and attached behavior; lines 278–280: revision and attached behavior; lines 281–283: live_inputs and attached behavior; lines 284–286: execute and attached behavior; lines 287–289: capture and attached behavior; lines 290–292: next and attached behavior; lines 293–295: resolve_inputs and attached behavior; lines 296–298: dependencies and attached behavior; lines 299–301: compile and attached behavior; lines 302–304: compile_rust and attached behavior; lines 305–307: stage_fork_binary and attached behavior; lines 308–310: assets and attached behavior; lines 311–313: images and attached behavior; lines 314–316: inspect_oci and attached behavior; lines 324–326: resolve_core_os and attached behavior; lines 330–332: check_native and attached behavior; lines 333–344: sign_media and attached behavior; lines 345–354: verify_copy and attached behavior; lines 355–453: write_document and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Native candidate build runner/cancellation/logging; declaration/member PINNED_GO_VERSION Adjacent comments and attributes explain this same authored responsibility.; 35 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 317–323; lines 317–323: verify_content and attached behavior | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | OCI/content verification adapter; declaration/member verify_content — lib/soda-release-image/src/build.rs:317-323; current named unit matched to maintained semantic map and release consumer |
| 327–329; lines 327–329: read_live_inputs and attached behavior | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Pinned CoreOS/live inputs adapter; declaration/member read_live_inputs — lib/soda-release-image/src/build.rs:327-329; current named unit matched to maintained semantic map and release consumer |

<a id="coverage-8143346f6f85"></a>

## [lib/soda-release-image/src/build_candidate.rs](../../../../../lib/soda-release-image/src/build_candidate.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; lines 1–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–15: use crate and attached body; lines 16–77: fn build_host_candidate and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn build_host_candidate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-01d2f08990d5"></a>

## [lib/soda-release-image/src/build_compile.rs](../../../../../lib/soda-release-image/src/build_compile.rs)

Current module body and direct consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–853; runtime command inventory and command discovery/compile selection; tool-files inventory JSON producer; Rust rootfs tool selector and compile adapter; shipping-tool inventory validation and compilation orchestration; embedded build-selector and tool-inventory tests | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Selects cmd runtime commands from the shipping inventory, validates Rust package/bin ownership, and compiles each Go or Rust command to rootfs/usr/libexec/soda; RUNTIME_COMMANDS preserves Rust binaries moved out of cmd/.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4b62897f4b79"></a>

## [lib/soda-release-image/src/build_context/mod.rs](../../../../../lib/soda-release-image/src/build_context/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–149; lines 1–5: use std and attached body; lines 6–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–60: fn freeze_base_image_config and attached body; lines 61–85: fn freeze_image_config and attached body; lines 86–128: fn prepare_build_host_context and attached body; lines 129–148: fn link_prepared_assets and attached body; lines 149–149: mod tests and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-835711281ef2"></a>

## [lib/soda-release-image/src/build_context/tests.rs](../../../../../lib/soda-release-image/src/build_context/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; lines 1–1: use super and attached body; lines 2–4: use std and attached body; lines 5–23: fn freezing_image_config_preserves_raw_duplicates_order_and_numeric_tokens and attached body; lines 24–25: fn oracle_link_prepared_assets_runs_no_commands and attached body; lines 26–29: struct Stub and attached body; lines 30–30: impl Production and attached body; lines 31–33: fn source and attached body; lines 34–36: fn forgejo_source and attached body; lines 37–39: fn forgejo_revision and attached body; lines 40–42: fn native and attached body; lines 43–45: fn out and attached body; lines 46–48: fn arch and attached body; lines 49–51: fn revision and attached body; lines 52–54: fn live_inputs and attached body; lines 55–57: fn execute and attached body; lines 58–60: fn capture and attached body; lines 61–63: fn next and attached body; lines 64–66: fn resolve_inputs and attached body; lines 67–69: fn dependencies and attached body; lines 70–72: fn compile and attached body; lines 73–75: fn compile_rust and attached body; lines 76–78: fn stage_fork_binary and attached body; lines 79–81: fn assets and attached body; lines 82–84: fn images and attached body; lines 85–87: fn inspect_oci and attached body; lines 88–94: fn verify_content and attached body; lines 95–97: fn resolve_core_os and attached body; lines 98–100: fn read_live_inputs and attached body; lines 101–103: fn check_native and attached body; lines 104–115: fn sign_media and attached body; lines 116–125: fn verify_copy and attached body; lines 126–146: fn write_document and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 32 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b4ab4b684b17"></a>

## [lib/soda-release-image/src/build_media.rs](../../../../../lib/soda-release-image/src/build_media.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–314; lines 1–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–13: use crate and attached body; lines 14–20: struct MediaTools and attached body; lines 21–21: const BUTANE_IMAGE and attached body; lines 22–38: fn prepare_build_media and attached body; lines 39–54: fn finish_build_media and attached body; lines 55–108: fn verify_butane_image and attached body; lines 109–130: fn admit_media_tools and attached body; lines 131–180: fn prepare_media_inputs and attached body; lines 181–181: mod tests and attached body; lines 182–184: use super and attached body; lines 185–186: fn oracle_media_boundary_skips_without_target and attached body; lines 187–187: struct Stub and attached body; lines 188–188: impl Production and attached body; lines 189–191: fn source and attached body; lines 192–194: fn forgejo_source and attached body; lines 195–197: fn forgejo_revision and attached body; lines 198–200: fn native and attached body; lines 201–203: fn out and attached body; lines 204–206: fn arch and attached body; lines 207–209: fn revision and attached body; lines 210–212: fn live_inputs and attached body; lines 213–215: fn execute and attached body; lines 216–218: fn capture and attached body; lines 219–221: fn next and attached body; lines 222–224: fn resolve_inputs and attached body; lines 225–227: fn dependencies and attached body; lines 228–230: fn compile and attached body; lines 231–233: fn compile_rust and attached body; lines 234–236: fn stage_fork_binary and attached body; lines 237–239: fn assets and attached body; lines 240–246: fn images and attached body; lines 247–249: fn inspect_oci and attached body; lines 250–256: fn verify_content and attached body; lines 257–259: fn resolve_core_os and attached body; lines 260–262: fn read_live_inputs and attached body; lines 263–265: fn check_native and attached body; lines 266–277: fn sign_media and attached body; lines 278–287: fn verify_copy and attached body; lines 288–314: fn write_document and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 46 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e7e08b83c482"></a>

## [lib/soda-release-image/src/build_runner.rs](../../../../../lib/soda-release-image/src/build_runner.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–374; lines 16–20: Cancel and attached behavior; attached module comments and attributes; lines 24–27: cancel and attached behavior; lines 28–36: is_cancelled and attached behavior; lines 37–39: SharedFile and attached behavior; lines 40–43: wrap and attached behavior; lines 44–45: impl Write; lines 46–48: write and attached behavior; lines 49–57: flush and attached behavior; lines 58–63: Runner and attached behavior; lines 70–77: execute and attached behavior; lines 78–82: capture and attached behavior; lines 83–89: open_log and attached behavior; lines 128–134: reason and attached behavior; lines 135–139: LogCloser and attached behavior; lines 140–151: close and attached behavior; lines 152–178: build_environment_pairs and attached behavior; lines 179–190: run_build_command and attached behavior; lines 191–215: fn run_build_command_with_budget; lines 216–216: const CAPTURE_LIMIT; lines 217–311: const DRAIN_GRACE; lines 312–316: fn bounded_operation_deadline; lines 317–318: struct ChildGuard; lines 321–321: type Target; lines 322–325: fn deref; lines 328–331: fn deref_mut; lines 332–333: impl Drop; lines 334–338: fn drop; lines 339–349: fn set_nonblocking; lines 350–354: enum PipeRead; lines 355–373: fn read_available; lines 374–374: mod tests | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Native candidate build runner/cancellation/logging; declaration/member Cancel Adjacent comments and attributes explain this same authored responsibility.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-87a991f781e1"></a>

## [lib/soda-release-image/src/build_runner/tests.rs](../../../../../lib/soda-release-image/src/build_runner/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–238; lines 4–19: fn metadata_deadline_is_scoped_to_the_metadata_operation; attached module comments and attributes; lines 20–43: fn bounded_build_operation_expires_while_output_keeps_arriving; lines 44–100: oracle_build_command_capture_environment_and_failure and attached behavior; lines 101–122: run_build_command_drains_saturated_pipes and attached behavior; lines 123–146: run_build_command_cancel_kills_and_reports and attached behavior; lines 147–162: fn cloned_cancellation_reaches_a_running_command; lines 163–164: struct CancellingWriter; lines 167–170: fn write; lines 171–177: fn flush; lines 178–194: fn run_build_command_streams_before_child_exit; lines 195–210: fn run_build_command_bounds_capture; lines 211–228: fn descendant_pipe_drain_deadline_closes_owned_fds; lines 229–229: fn output_read_error_is_returned; lines 230–230: struct FailingReader; lines 232–238: fn read | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current fn metadata_deadline_is_scoped_to_the_metadata_operation and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7adbc5baf32b"></a>

## [lib/soda-release-image/src/build_source.rs](../../../../../lib/soda-release-image/src/build_source.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; verify_checkout_source and attached module contract | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Validate that the controller-selected source is a canonical clean checkout; source admission belongs to the release build request boundary. — Current source inspected at lib/soda-release-image/src/build_source.rs; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-88e9feae83ed"></a>

## [lib/soda-release-image/src/complete.rs](../../../../../lib/soda-release-image/src/complete.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–288; lines 1–3: use std and attached body; lines 4–4: use std and attached body; lines 5–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–42: fn stage_presentation and attached body; lines 43–52: fn validate_complete_payload and attached body; lines 53–76: fn write_complete_quadlets and attached body; lines 77–96: fn verify_public_branding_and_motd and attached body; lines 97–115: fn write_factory_defaults and attached body; lines 116–148: fn configure_complete_systemd and attached body; lines 149–155: fn bind_extension_install_image and attached body; lines 156–165: const PLACEHOLDER and attached body; lines 166–187: fn write_release_metadata_and_normalize and attached body; lines 188–195: use std and attached body; lines 196–219: fn complete and attached body; lines 220–220: mod tests and attached body; lines 221–223: use super and attached body; lines 224–234: fn oracle_complete_refuses_before_payload and attached body; lines 235–255: fn oracle_extension_binding_refuses_ambiguity and attached body; lines 256–288: fn quadlets_read_new_layout and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-45ac36eaf0bb"></a>

## [lib/soda-release-image/src/compression.rs](../../../../../lib/soda-release-image/src/compression.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147; ImageConfig admission and mutation; ordered raw-number/duplicate-preserving parse/serialization; standard pretty output; exact and folded option selection; both exact option pairs update in place | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Fresh image-config output uses standard Serde over the retained OrderedValue representation; duplicate/order/raw-number semantics and sorted output remain. GoFormatter and compact/indent compatibility helpers are retired in `274621ce`; 86 actual checks include duplicate-mutation coverage. |

<a id="coverage-206d9eff3ea0"></a>

## [lib/soda-release-image/src/error.rs](../../../../../lib/soda-release-image/src/error.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35; lines 1–5: struct Error and attached body; lines 6–7: impl Error and attached body; lines 8–11: fn msg and attached body; lines 12–13: impl std and attached body; lines 14–17: fn fmt and attached body; lines 18–19: impl std and attached body; lines 20–21: impl From and attached body; lines 22–28: fn from and attached body; lines 29–35: fn join_close and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for struct Error in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9d3f8dd26c13"></a>

## [lib/soda-release-image/src/events.rs](../../../../../lib/soda-release-image/src/events.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–149; MediaEventWriter line construction and direct Serde event output; EventRecord fields; bounded public-window event test | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Event JSONL output uses standard Serde and preserves its current fields/newline and bounded-window behavior. No repository consumer pins Go float spellings; the old byte-only float formatter is retired in `274621ce`. |

<a id="coverage-07ef85b82f0f"></a>

## [lib/soda-release-image/src/extension.rs](../../../../../lib/soda-release-image/src/extension.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–91; lines 1–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–12: use crate and attached body; lines 13–23: fn stage_extension_assets and attached body; lines 24–35: fn extension_asset_inventory and attached body; lines 36–42: fn safe_extension_asset_name and attached body; lines 43–57: fn link_extension_asset and attached body; lines 58–58: mod tests and attached body; lines 59–61: use super and attached body; lines 62–91: fn oracle_asset_stage_keeps_recorded_output_and_refuses_unsafe and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7be2e731223b"></a>

## [lib/soda-release-image/src/files.rs](../../../../../lib/soda-release-image/src/files.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; lines 1–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: use serde and attached body; lines 9–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–19: fn owned_write and attached body; lines 20–26: fn hash_bytes and attached body; lines 27–56: fn public_files and attached body; lines 57–64: struct MediaFile and attached body; lines 65–76: fn media_file and attached body; lines 77–77: mod tests and attached body; lines 78–80: use super and attached body; lines 81–100: fn oracle_public_files_enforce_modes and attached body | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-35029f5fe87a"></a>

## [lib/soda-release-image/src/foreign.rs](../../../../../lib/soda-release-image/src/foreign.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–47, 50–63, 104–111; lines 21–29: struct PackagingInputs; attached module comments and attributes; lines 50–51: fn dependencies; lines 52–53: fn compile; lines 54–55: fn compile_rust; lines 58–59: fn assets; lines 60–63: fn images | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current struct PackagingInputs and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 6 named units assigned here; remaining selectors preserve each duty — lib/soda-release-image/src/foreign.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 48–49, 73–77; lines 48–49: fn resolve_inputs; lines 73–74: fn resolve_core_os; lines 75–77: fn read_live_inputs | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Pinned input acquisition adapter: resolve_inputs; Pinned input acquisition adapter: resolve_core_os; Pinned input acquisition adapter: read_live_inputs — lib/soda-release-image/src/foreign.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 64–72, 78–79; lines 64–66: fn inspect_oci; lines 67–72: fn verify_content; lines 78–79: fn check_native | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Artifact verification adapter: inspect_oci; Artifact verification adapter: verify_content; Artifact verification adapter: check_native — lib/soda-release-image/src/foreign.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 80–90; lines 80–90: fn sign_media | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Signing custody adapter: sign_media — lib/soda-release-image/src/foreign.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 91–99; lines 91–99: fn verify_copy | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Verified distribution consumption adapter: verify_copy — lib/soda-release-image/src/foreign.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 100–103; lines 100–103: fn write_document | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Release admission and preparation adapter: write_document — lib/soda-release-image/src/foreign.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-62ddbc3ba015"></a>

## [lib/soda-release-image/src/forgejo.rs](../../../../../lib/soda-release-image/src/forgejo.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–173; lines 1–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–12: use crate and attached body; lines 13–48: fn extract_forgejo_snapshot and attached body; lines 49–49: mod tests and attached body; lines 50–52: use super and attached body; lines 53–54: fn oracle_forgejo_snapshot_refuses_bad_revision and attached body; lines 55–55: struct Stub and attached body; lines 56–56: impl Production and attached body; lines 57–59: fn source and attached body; lines 60–62: fn forgejo_source and attached body; lines 63–65: fn forgejo_revision and attached body; lines 66–68: fn native and attached body; lines 69–71: fn out and attached body; lines 72–74: fn arch and attached body; lines 75–77: fn revision and attached body; lines 78–80: fn live_inputs and attached body; lines 81–83: fn execute and attached body; lines 84–86: fn capture and attached body; lines 87–89: fn next and attached body; lines 90–92: fn resolve_inputs and attached body; lines 93–95: fn dependencies and attached body; lines 96–98: fn compile and attached body; lines 99–101: fn compile_rust and attached body; lines 102–104: fn stage_fork_binary and attached body; lines 105–107: fn assets and attached body; lines 108–114: fn images and attached body; lines 115–117: fn inspect_oci and attached body; lines 118–124: fn verify_content and attached body; lines 125–127: fn resolve_core_os and attached body; lines 128–130: fn read_live_inputs and attached body; lines 131–133: fn check_native and attached body; lines 134–145: fn sign_media and attached body; lines 146–155: fn verify_copy and attached body; lines 156–173: fn write_document and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 37 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6ec9c0721258"></a>
<a id="coverage-9c45c3c490a8"></a>

## [lib/soda-release-image/src/host.rs](../../../../../lib/soda-release-image/src/host.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–196, 250–268; lines 15–160: fn read_host_image_id, fn inspect_host_identity, fn observe_host_packages, fn record_host_packages, fn export_host_archive, fn build_host_image; attached module comments and attributes; build_host_image SCOPE passed to rootfs build | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Native host image architecture/package provenance Adjacent comments and attributes explain this same authored responsibility.; Selects the complete-local-payload scope used for host image assembly. — lib/soda-release-image/src/host.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; Current build_host_image/verify_built_host call sites show the selector controls image assembly versus verification. |
| 197–249; lines 197–209: fn verify_built_host; verify_built_host SCOPE passed to content identity inspection | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Immutable produced host image readback; Selects the complete-local-payload scope used when inspecting the assembled host image content. — lib/soda-release-image/src/host.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; Current build_host_image/verify_built_host call sites show the selector controls image assembly versus verification. |

<a id="coverage-d052e96d3c04"></a>

## [lib/soda-release-image/src/ignition.rs](../../../../../lib/soda-release-image/src/ignition.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–357; lines 1–3: use std and attached body; lines 4–5: use base64 and attached body; lines 6–6: use flate2 and attached body; lines 7–7: use serde and attached body; lines 8–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–13: const LIVE_IGNITION_LIMIT and attached body; lines 14–18: const LIVE_IGNITION_COMPRESSED_LIMIT and attached body; lines 19–35: fn verify_live_ignition and attached body; lines 36–94: fn live_ignition_bytes and attached body; lines 95–97: const CANDIDATE_INSTALLER_BINARY and attached body; lines 98–113: struct MediaIdentity and attached body; lines 114–115: impl MediaIdentity and attached body; lines 116–128: fn validate and attached body; lines 129–141: fn valid_content and attached body; lines 142–185: fn candidate_live_config and attached body; lines 186–189: struct InlineContents and attached body; lines 190–195: struct InlineFile and attached body; lines 196–200: struct MaskUnit and attached body; lines 201–207: struct ConsoleUnit and attached body; lines 208–212: enum Unit and attached body; lines 213–216: struct IgnitionVersion and attached body; lines 217–220: struct Storage and attached body; lines 221–224: struct Systemd and attached body; lines 225–240: struct IgnitionDocument and attached body; lines 241–296: const LIVE_DATA and attached body; lines 297–297: mod tests and attached body; lines 298–298: use super and attached body; lines 299–299: use flate2 and attached body; lines 300–300: use flate2 and attached body; lines 301–301: use std and attached body; lines 302–309: fn gzip_bytes and attached body; lines 310–337: fn oracle_live_ignition_readback_ignores_nulls and attached body; lines 338–338: fn live_ignition_equality_keeps_object_order_duplicates_and_number_tokens and attached body; lines 339–357: fn wrapped and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 36 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a9419e4b62dd"></a>

## [lib/soda-release-image/src/inspect.rs](../../../../../lib/soda-release-image/src/inspect.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–231; lines 1–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–10: use crate and attached body; lines 11–11: type InspectRun and attached body; lines 12–33: fn inspect_complete_payload and attached body; lines 34–67: fn inspect_complete_content and attached body; lines 68–82: fn inspect_complete_quadlets and attached body; lines 83–101: use std and attached body; lines 102–123: fn inspect_complete_extension_unit and attached body; lines 124–145: fn inspect_complete_service_files and attached body; lines 146–147: fn inspect_complete_inventory and attached body; lines 148–161: const PATH and attached body; lines 162–201: fn inspect_complete and attached body; lines 202–202: mod tests and attached body; lines 203–205: use super and attached body; lines 206–231: fn oracle_quadlet_generator_binding_checks and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eafc06048d6d"></a>

## [lib/soda-release-image/src/jsonio.rs](../../../../../lib/soda-release-image/src/jsonio.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–117; OrderedMap defaults and visitation; SortedPairs serialization; case-insensitive record decoding; field aliases and duplicate-field refusal; bounded numeric/string decoders | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Shared parsing/ordered-map duties remain for live consumers. The GoFormatter, compact/indent emitters, Go float formatting and byte-golden tests were removed in `274621ce`; output-site-specific profiles are recorded at the actual producers. |

<a id="coverage-9f6c9a85cf9b"></a>
<a id="coverage-35865cdb81e1"></a>

## [lib/soda-release-image/src/layout.rs](../../../../../lib/soda-release-image/src/layout.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75, 97–110; lines 15–75: fn valid_stage_layout, fn verify_archive_digests, fn copy_staged_archives, fn chmod_staged_files; attached module comments and attributes | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Native candidate shared OCI image layout assembly Adjacent comments and attributes explain this same authored responsibility. — lib/soda-release-image/src/layout.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 76–96; lines 76–96: fn stage_images | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Verify generated layout immutable identities — lib/soda-release-image/src/layout.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-aaa9527d7af8"></a>

## [lib/soda-release-image/src/lib.rs](../../../../../lib/soda-release-image/src/lib.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; lines 1–16: mod build and attached body; lines 17–17: mod build_candidate and attached body; lines 18–18: mod build_compile and attached body; lines 19–19: mod build_context and attached body; lines 20–20: mod build_media and attached body; lines 21–21: mod build_runner and attached body; lines 22–22: mod build_source and attached body; lines 23–23: mod complete and attached body; lines 24–24: mod compression and attached body; lines 25–25: mod error and attached body; lines 26–26: mod events and attached body; lines 27–27: mod extension and attached body; lines 28–28: mod files and attached body; lines 29–29: mod foreign and attached body; lines 30–30: mod forgejo and attached body; lines 31–31: mod host and attached body; lines 32–32: mod ignition and attached body; lines 33–33: mod inspect and attached body; lines 34–34: mod jsonio and attached body; lines 35–35: mod layout and attached body; lines 36–36: mod media and attached body; lines 37–37: mod media_assembler and attached body; lines 38–38: mod media_authentication and attached body; lines 39–39: mod media_container and attached body; lines 40–40: mod media_installer and attached body; lines 41–41: mod model and attached body; lines 42–42: mod ordered_json and attached body; lines 43–43: mod packages and attached body; lines 44–44: mod payload_stage and attached body; lines 45–45: mod prepare and attached body; lines 46–46: mod quadlet and attached body; lines 47–47: mod recall and attached body; lines 48–48: mod record and attached body; lines 49–49: mod request and attached body; lines 50–50: mod rootfs and attached body; lines 51–51: mod sys and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for mod build in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 36 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6104704208c3"></a>
<a id="rustsoda-release-imagesrcmediars-1"></a>
<a id="coverage-7c5a1bea2b62"></a>

## [lib/soda-release-image/src/media.rs](../../../../../lib/soda-release-image/src/media.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101, 148–229; lines 34–35: RunFn and attached behavior; attached module comments and attributes; lines 36–38: NativeFn and attached behavior; lines 39–54: MediaLock and attached behavior; lines 55–60: MediaAuthority and attached behavior; lines 61–71: parse and attached behavior; lines 72–101: Media and attached behavior; lines 148–163: fn admit_media_authority; lines 164–190: admit_media_candidate and attached behavior; lines 191–205: admit_media_compression and attached behavior; lines 206–229: admit_media_inputs and attached behavior | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Authenticated media sources, inputs and assembly output contract; declaration/member RunFn Adjacent comments and attributes explain this same authored responsibility.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 102–137; lines 102–137: fn media_base_url | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Pinned native media assembler/ISO input acquisition — lib/soda-release-image/src/media.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 138–147; lines 138–147: MediaInputs and attached behavior | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Native exact digest media signing; declaration/member MediaInputs — lib/soda-release-image/src/media.rs:138-147; current named unit matched to maintained semantic map and release consumer |
| 230–350; lines 230–276: seal_media and attached behavior; lines 277–349: assemble_media and attached behavior; lines 350–350: tests and attached behavior | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Immutable emitted media readback verification; declaration/member seal_media; Immutable emitted media readback verification; declaration/member assemble_media; Immutable emitted media readback verification; declaration/member tests — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b62ef7c4e563"></a>

## [lib/soda-release-image/src/media/tests.rs](../../../../../lib/soda-release-image/src/media/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–54; lines 1–3: use super and attached body; lines 4–31: fn oracle_media_url_has_no_credentials_or_mutable_query and attached body; lines 32–54: fn oracle_assembler_pin_requires_buildroot_selection and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn oracle_media_url_has_no_credentials_or_mutable_query in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn oracle_assembler_pin_requires_buildroot_selection in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7bf6866f2c39"></a>

## [lib/soda-release-image/src/media_assembler.rs](../../../../../lib/soda-release-image/src/media_assembler.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–227; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–13: use crate and attached body; lines 14–14: const ASSEMBLER_IMAGE and attached body; lines 15–15: const ASSEMBLER_CONFIG_BRANCH and attached body; lines 16–74: fn fetch_assembler_config and attached body; lines 75–95: fn pin_assembler_build_args and attached body; lines 96–131: fn verify_assembler_layers and attached body; lines 132–159: fn wrap_assembler_image and attached body; lines 160–197: const PREFIX and attached body; lines 198–213: fn prepare_assembler and attached body; lines 214–227: fn builder_id and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cc64c78b014e"></a>

## [lib/soda-release-image/src/media_authentication.rs](../../../../../lib/soda-release-image/src/media_authentication.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; lines 1–1: use crate and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–49: fn sign_media_input and attached body; lines 50–85: fn collect_media_inventory and attached body; lines 86–101: fn verify_media_inventory and attached body; lines 102–146: fn authenticate_packaging_inputs and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b2754ab52522"></a>

## [lib/soda-release-image/src/media_container.rs](../../../../../lib/soda-release-image/src/media_container.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–193; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–43: fn stop_packaging_container and attached body; lines 44–93: fn build_media_container and attached body; lines 94–111: fn assemble_native_media and attached body; lines 112–118: struct MediaMetaImage and attached body; lines 119–124: struct MediaMeta and attached body; lines 125–139: struct MediaMetaWire and attached body; lines 140–141: impl MediaMeta and attached body; lines 142–149: fn parse and attached body; lines 150–174: fn verify_meta_images and attached body; lines 175–193: fn verify_build_meta and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2803233c50af"></a>

## [lib/soda-release-image/src/media_installer.rs](../../../../../lib/soda-release-image/src/media_installer.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–224; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–25: fn setup_media_rootfs and attached body; lines 26–60: fn verify_customized_iso and attached body; lines 61–121: fn customize_installer_iso and attached body; lines 122–191: fn verify_media_readback and attached body; lines 192–224: fn prepare_and_verify_media and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cc36266cdbdc"></a>
<a id="rustsoda-release-imagesrcmodelrs-1"></a>
<a id="coverage-f0e1bffd065b"></a>

## [lib/soda-release-image/src/model.rs](../../../../../lib/soda-release-image/src/model.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 11–13, 15–112; lines 8–9: mod candidate; attached module comments and attributes; lines 11–11: mod live_inputs; lines 12–12: mod payload; lines 13–13: mod trust; lines 25–26: DELIVER_PATH and attached behavior; lines 27–28: DELIVER_IMAGES_PATH and attached behavior; lines 29–36: NAMES and attached behavior; lines 37–39: FORGEJO_COMPILER_IMAGE and attached behavior; lines 40–40: SCHEMA_VERSION and attached behavior; lines 41–41: SODA_SOURCE and attached behavior; lines 42–45: is_revision and attached behavior; lines 46–49: is_digest and attached behavior; lines 50–55: oci_architecture and attached behavior; lines 56–63: prefixed_digest and attached behavior; lines 64–66: deliver_hash and attached behavior; lines 67–74: is_dotted_numbers and attached behavior; lines 75–78: is_coreos_release and attached behavior; lines 79–112: valid_repository_prefix and attached behavior | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Current mod candidate and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 10, 113–123; lines 10–10: images and attached behavior; lines 113–122: content_get and attached behavior; lines 123–123: mod tests | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Immutable OCI image/payload identity contract; declaration/member Payload.images; Immutable candidate/content provenance validation; declaration/member content_get; Source-level oracle/test assertions for candidate/input source vectors — Current named units/source consumers; retained normalized source evidence records each selector |
| 14; lines 14–14: url and attached behavior | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member CoreOSImage.url — lib/soda-release-image/src/model.rs:14-14; current named unit matched to maintained semantic map and release consumer |

<a id="coverage-7b95c5dad56b"></a>

## [lib/soda-release-image/src/model/candidate.rs](../../../../../lib/soda-release-image/src/model/candidate.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–283; lines 1–1: use serde and attached body; lines 2–2: use serde and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–17: use super and attached body; lines 18–23: struct ForgejoToolchain and attached body; lines 24–25: impl ForgejoToolchain and attached body; lines 26–28: fn parse and attached body; lines 29–48: fn validate and attached body; lines 49–77: struct CandidateWire and attached body; lines 78–101: fn valid_forgejo_apk_list and attached body; lines 102–113: fn has_forgejo_native_build_tools and attached body; lines 114–127: struct Candidate and attached body; lines 128–129: impl Serialize and attached body; lines 130–149: fn serialize and attached body; lines 150–151: impl Candidate and attached body; lines 152–168: fn parse and attached body; lines 169–185: fn validate and attached body; lines 186–199: fn valid_candidate_host and attached body; lines 200–211: fn valid_candidate_provenance and attached body; lines 212–222: fn valid_candidate_source and attached body; lines 223–229: fn valid_candidate_build and attached body; lines 230–241: const REQUIRED_CANDIDATE_CONTENT and attached body; lines 242–271: fn valid_candidate_content and attached body; lines 272–283: fn valid_candidate_asset_name and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a74d9fb161c6"></a>

## [lib/soda-release-image/src/model/images.rs](../../../../../lib/soda-release-image/src/model/images.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–9: use serde and attached body; lines 10–25: struct Image and attached body; lines 26–27: impl Image and attached body; lines 28–43: fn parse and attached body; lines 44–48: struct ProducedImage and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-25e0a610e831"></a>

## [lib/soda-release-image/src/model/live_inputs.rs](../../../../../lib/soda-release-image/src/model/live_inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–208; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–3: use serde and attached body; lines 4–11: use super and attached body; lines 12–21: struct CoreOSImage and attached body; lines 22–23: impl CoreOSImage and attached body; lines 24–36: fn parse and attached body; lines 37–53: struct ResolvedCoreOSWire and attached body; lines 54–60: struct ResolvedCoreOS and attached body; lines 61–62: impl ResolvedCoreOS and attached body; lines 63–72: fn parse and attached body; lines 73–79: fn container_ref and attached body; lines 80–84: struct TailnetInputs and attached body; lines 85–86: impl TailnetInputs and attached body; lines 87–101: fn parse and attached body; lines 102–105: struct LiveInputs and attached body; lines 106–107: impl LiveInputs and attached body; lines 108–118: fn parse and attached body; lines 119–129: fn deserialize and attached body; lines 130–133: fn find_arch and attached body; lines 134–163: fn valid_stream_images and attached body; lines 164–180: fn valid_tailnet_inputs and attached body; lines 181–203: fn valid_resolved_core_os and attached body; lines 204–208: fn valid_live_inputs and attached body | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b7fa69413b37"></a>

## [lib/soda-release-image/src/model/payload.rs](../../../../../lib/soda-release-image/src/model/payload.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–200; lines 1–1: use serde and attached body; lines 2–2: use serde and attached body; lines 3–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–17: use super and attached body; lines 18–36: struct PayloadImage and attached body; lines 37–67: struct PayloadWire and attached body; lines 68–81: struct Payload and attached body; lines 82–83: impl Serialize and attached body; lines 84–100: fn serialize and attached body; lines 101–102: impl Payload and attached body; lines 103–119: fn parse and attached body; lines 120–127: fn image and attached body; lines 128–134: fn valid_identity and attached body; lines 135–148: fn valid_base and attached body; lines 149–173: fn valid_images and attached body; lines 174–193: fn validate and attached body; lines 194–200: fn load and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2a4a1d3cf5cb"></a>

## [lib/soda-release-image/src/model/tests.rs](../../../../../lib/soda-release-image/src/model/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; lines 1–3: use super and attached body; lines 4–14: fn oracle_prefix_and_digest_shapes and attached body; lines 15–43: fn url_parser_preserves_metadata_policy_fields and attached body; lines 44–53: fn oracle_payload_identity_validation and attached body; lines 54–78: fn trust_key_admission_preserves_cross_role_separation and attached body; lines 79–96: fn producer_trust_fixture_keeps_its_raw_der_fingerprint and attached body; lines 97–108: fn image_model_uses_exact_then_first_folded_raw_member and attached body; lines 109–120: fn release_dtos_defer_nested_alias_conversion_and_keep_minus_zero and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-53bef357aa10"></a>

## [lib/soda-release-image/src/model/trust.rs](../../../../../lib/soda-release-image/src/model/trust.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–154; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–11: use super and attached body; lines 12–23: struct Trust and attached body; lines 24–44: struct TrustWire and attached body; lines 45–46: impl Trust and attached body; lines 47–59: fn parse and attached body; lines 60–67: fn role_keys and attached body; lines 68–75: fn minimum and attached body; lines 76–100: fn validate and attached body; lines 101–117: fn admit_trust_role_keys and attached body; lines 118–127: fn parse_trust_public_key and attached body; lines 128–136: struct Permit and attached body; lines 137–140: struct SecretFiles and attached body; lines 141–142: impl SecretFiles and attached body; lines 143–154: fn parse and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-385919e62165"></a>

## [lib/soda-release-image/src/model/url.rs](../../../../../lib/soda-release-image/src/model/url.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–103; lines 1–3: use url and attached body; lines 4–15: struct UrlParts and attached body; lines 16–52: fn parse_url and attached body; lines 53–74: fn valid_escapes and attached body; lines 75–88: fn https_url and attached body; lines 89–103: fn is_loopback_addr and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use url in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f2b5b6a6ca53"></a>

## [lib/soda-release-image/src/ordered_json.rs](../../../../../lib/soda-release-image/src/ordered_json.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–240; lines 1–4: use serde and attached body; lines 5–5: use serde and attached body; lines 6–6: use serde and attached body; lines 7–7: use serde_json and attached body; lines 8–11: use crate and attached body; lines 12–19: enum OrderedValue and attached body; lines 20–21: impl OrderedValue and attached body; lines 22–25: fn parse and attached body; lines 26–34: fn from_raw and attached body; lines 35–38: struct ObjectVisitor and attached body; lines 39–39: type Value and attached body; lines 40–42: fn expecting and attached body; lines 43–68: fn visit_map and attached body; lines 69–72: struct ArrayVisitor and attached body; lines 73–73: type Value and attached body; lines 74–76: fn expecting and attached body; lines 77–113: fn visit_seq and attached body; lines 114–120: fn object and attached body; lines 121–127: fn object_mut and attached body; lines 128–137: fn set_all_exact and attached body; lines 138–145: fn last_exact and attached body; lines 146–158: fn first_exact_then_folded and attached body; lines 159–166: fn string_or_empty and attached body; lines 167–184: fn prune_null_members and attached body; lines 185–186: impl Serialize and attached body; lines 187–214: fn serialize and attached body; lines 215–215: mod depth_tests and attached body; lines 216–218: use super and attached body; lines 219–240: fn ordered_values_match_serde_container_depth_limit and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 29 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d69d07417410"></a>

## [lib/soda-release-image/src/packages.rs](../../../../../lib/soda-release-image/src/packages.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–192; lines 1–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–6: const TAILSCALE_REPO_URL and attached body; lines 7–11: const INSTALL_PREFIX and attached body; lines 12–59: fn package_inputs and attached body; lines 60–78: fn parse_install_packages and attached body; lines 79–92: fn is_package_name and attached body; lines 93–106: fn valid_rpm_inventory and attached body; lines 107–141: fn is_rpm_line and attached body; lines 142–142: mod tests and attached body; lines 143–145: use super and attached body; lines 146–174: fn oracle_package_input_failures and attached body; lines 175–192: fn oracle_rpm_inventory_shape and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6170dd468d15"></a>
<a id="coverage-2efb1cca3994"></a>

## [lib/soda-release-image/src/payload_stage.rs](../../../../../lib/soda-release-image/src/payload_stage.rs)

Current module body and direct consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–495; whole file: candidate payload file staging and content inventory construction; artifact verification helpers retain d05 duties where separately selected | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Candidate payload file staging and content inventory construction; artifact verification helpers retain D05 duties where separately selected. — lib/soda-release-image/src/payload_stage.rs:1-495; module purpose and current direct consumer inspected |

<a id="coverage-c74a73ba4b6a"></a>
<a id="rustsoda-release-imagesrcpreparers-1"></a>
<a id="coverage-08d76b98c233"></a>

## [lib/soda-release-image/src/prepare.rs](../../../../../lib/soda-release-image/src/prepare.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–377; lines 20–26: Base and attached behavior; attached module comments and attributes; lines 27–40: image and attached behavior; lines 41–49: load_base and attached behavior; lines 50–57: load_base_from_file and attached behavior; lines 58–67: base_from_resolved and attached behavior; lines 68–74: PreparedWriter and attached behavior; lines 75–81: write and attached behavior; lines 82–97: copy_file and attached behavior; lines 98–113: write_base_files and attached behavior; lines 114–128: stage_symlinks_and_extras and attached behavior; lines 129–139: stage_rootfs_files and attached behavior; lines 140–148: write_build_record and attached behavior; lines 149–185: struct BuildRecord; lines 186–258: rootfs_file_map and attached behavior; lines 259–268: load_base_inputs and attached behavior; lines 269–279: load_base_inputs_resolved and attached behavior; lines 280–296: finish_base_inputs and attached behavior; lines 297–309: prepare and attached behavior; lines 310–321: prepare_resolved and attached behavior; lines 322–345: finish_prepare and attached behavior; lines 346–376: inventory and attached behavior; lines 377–377: tests and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Complete native candidate image assembly and authenticated live media; declaration/member Base Adjacent comments and attributes explain this same authored responsibility.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-69c8eba2882c"></a>

## [lib/soda-release-image/src/prepare/tests.rs](../../../../../lib/soda-release-image/src/prepare/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–125; lines 1–3: use super and attached body; lines 4–5: fn oracle_base_inputs_require_exact_revision and attached body; lines 6–6: struct Stub and attached body; lines 7–7: impl Production and attached body; lines 8–10: fn source and attached body; lines 11–13: fn forgejo_source and attached body; lines 14–16: fn forgejo_revision and attached body; lines 17–19: fn native and attached body; lines 20–22: fn out and attached body; lines 23–25: fn arch and attached body; lines 26–28: fn revision and attached body; lines 29–31: fn live_inputs and attached body; lines 32–34: fn execute and attached body; lines 35–37: fn capture and attached body; lines 38–40: fn next and attached body; lines 41–43: fn resolve_inputs and attached body; lines 44–46: fn dependencies and attached body; lines 47–49: fn compile and attached body; lines 50–52: fn compile_rust and attached body; lines 53–55: fn stage_fork_binary and attached body; lines 56–58: fn assets and attached body; lines 59–64: fn images and attached body; lines 65–67: fn inspect_oci and attached body; lines 68–74: fn verify_content and attached body; lines 75–77: fn resolve_core_os and attached body; lines 78–80: fn read_live_inputs and attached body; lines 81–83: fn check_native and attached body; lines 84–95: fn sign_media and attached body; lines 96–105: fn verify_copy and attached body; lines 106–125: fn write_document and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0ab8b93d3e88"></a>

## [lib/soda-release-image/src/quadlet.rs](../../../../../lib/soda-release-image/src/quadlet.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–69; lines 1–3: use crate and attached body; lines 4–35: fn rewrite_quadlet_line and attached body; lines 36–52: fn local_quadlet and attached body; lines 53–53: mod tests and attached body; lines 54–56: use super and attached body; lines 57–69: fn oracle_quadlet_rewrite_and_refusals and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2c97e43a26d7"></a>

## [lib/soda-release-image/src/recall.rs](../../../../../lib/soda-release-image/src/recall.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–131; lines 1–9: use crate and attached body; lines 10–14: struct RecallLog and attached body; lines 15–16: impl RecallLog and attached body; lines 17–19: fn new and attached body; lines 20–32: fn write_bytes and attached body; lines 33–41: use std and attached body; lines 42–42: fn attach and attached body; lines 43–52: use std and attached body; lines 53–64: fn reason and attached body; lines 65–77: fn qualify_reason and attached body; lines 78–87: fn failure_reason and attached body; lines 88–90: fn is_tool_failure and attached body; lines 91–92: impl std and attached body; lines 93–95: fn write and attached body; lines 96–105: fn flush and attached body; lines 106–106: mod tests and attached body; lines 107–109: use super and attached body; lines 110–121: fn oracle_reason_skips_markers and attached body; lines 122–131: fn oracle_failure_reason_prefers_local_error_over_stale_ring and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-affcebd9c2d5"></a>

## [lib/soda-release-image/src/record.rs](../../../../../lib/soda-release-image/src/record.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–267; lines 1–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–15: use crate and attached body; lines 16–101: fn candidate_content and attached body; lines 102–109: fn content_inventory_bytes and attached body; lines 110–125: fn write_content_inventory and attached body; lines 126–183: fn record_build_result and attached body; lines 184–207: fn record_candidate and attached body; lines 208–230: fn record_candidate_inputs and attached body; lines 231–249: fn verify_embedded_content_inventory and attached body; lines 250–250: mod tests and attached body; lines 251–253: use super and attached body; lines 254–267: fn oracle_candidate_inputs_require_exact_fork_revision and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ad348e05d9a8"></a>
<a id="coverage-8509ff884a2a"></a>

## [lib/soda-release-image/src/request.rs](../../../../../lib/soda-release-image/src/request.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–217; lines 10–118: struct Request, struct Result, impl Request, fn validate_development_target, fn validate_media_inputs, fn validate_target, fn wants_media, fn purpose, fn requested_target; attached module comments and attributes | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Explicit builder target/source/output/media admission contract Adjacent comments and attributes explain this same authored responsibility. — lib/soda-release-image/src/request.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 218–238; lines 218–238: fn oracle_result_omits_empty_media_fields | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Source-level oracle/test assertions for source assertion of authenticated installation media — lib/soda-release-image/src/request.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-980f31af825d"></a>

## [lib/soda-release-image/src/rootfs.rs](../../../../../lib/soda-release-image/src/rootfs.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–107; lines 1–3: use std and attached body; lines 4–4: use std and attached body; lines 5–6: use sha2 and attached body; lines 7–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–13: const CHUNK_LEN and attached body; lines 14–34: fn verify_rootfs_chunks and attached body; lines 35–66: fn match_rootfs_chunk and attached body; lines 67–67: mod tests and attached body; lines 68–70: use super and attached body; lines 71–107: fn oracle_rootfs_chunks_match_and_mismatch and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3270fc4fae13"></a>
<a id="coverage-fff6b3143c3a"></a>

## [lib/soda-release-image/src/sys.rs](../../../../../lib/soda-release-image/src/sys.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–266, 289–301, 316–967; lines 29–32: fn is_false; attached module comments and attributes; lines 61–74: fn is_clean_path; lines 75–78: fn is_clean_abs; lines 237–242: fn read_json_build_text; lines 243–243: fn read_json_deliver_text; lines 260–261: fn read_limited; lines 289–289: mod json_tests; lines 293–301: fn file_mode_keeps_integer_token_and_minus_zero_policy; lines 323–329: struct CargoMetadata; lines 330–339: struct CargoPackageMetadata; lines 340–347: struct CargoTargetMetadata; lines 348–357: struct ShippingPackage; lines 358–362: struct ShippingInventory; lines 363–382: fn active_default_features; lines 383–449: fn parse_shipping_inventory; lines 450–451: impl ShippingInventory; lines 452–464: fn package_at; lines 465–483: fn require_bin; lines 484–499: fn contains_bin_at; lines 500–506: fn require_bin_at; lines 507–520: fn require_package_bin; lines 521–524: fn package_name_for_manifest; lines 525–530: fn has_bins_at; lines 531–536: fn package_name_at; lines 537–543: fn commands; lines 544–589: fn select_commands; lines 590–602: fn has_go_sources; lines 657–737: fn cargo_metadata_binds_opaque_ids_to_canonical_manifests_and_default_features; lines 738–787: fn actual_workspace_shipping_commands_preserve_the_selected_inventory; lines 788–794: fn bounded_read_caps_growth_and_keeps_open_inode; lines 821–848: fn deliver_text_rejects_links_fifos_and_cap_plus_one; lines 862–876: fn native_path_helpers_and_clean_admission; lines 915–946: fn inventory_rejects_duplicate_opaque_ids | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current fn is_false and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 33 named units assigned here; remaining selectors preserve each duty — lib/soda-release-image/src/sys.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 267–288; lines 267–288: fn refused, fn private_file | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Strict bounded JSON input — lib/soda-release-image/src/sys.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 302–315; lines 302–315: fn require_native | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Matching native architecture/source admission — lib/soda-release-image/src/sys.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

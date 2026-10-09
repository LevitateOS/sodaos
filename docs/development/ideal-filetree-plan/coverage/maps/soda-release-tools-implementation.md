# Soda release tools implementation

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-416616ed7f21"></a>

## [lib/soda-release-tools/build.rs](../../../../../lib/soda-release-tools/build.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–59; lines 1–3: module docs/imports; 5–18: fn git; 20–26: fn watch_git_input; 28–38: fn watch_tracked_files; 40–59: fn main | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Build-time Soda revision/dirty stamp; read-only Git calls use `--no-optional-locks`; Cargo watches resolved HEAD/index/packed-refs/refs/current symbolic ref plus each repository-tracked path. Git failure conservatively yields empty revision and modified=1. Ten actual Cargo fixture cases cover cached clean/dirty and ref changes plus missing-Git behavior; both affected native-bin builds and a production stamp observation at `c63c707f` are recorded in the execution finding. New untracked paths are visible to status only after another watched input reruns the build script; public `soda-candidate` admission separately calls fresh `check_clean_tree`. |

<a id="coverage-8e670c258300"></a>
<a id="rustsoda-release-toolssrcartifactsrs-1"></a>
<a id="coverage-bfc7f26e0fe7"></a>

## [lib/soda-release-tools/src/artifacts.rs](../../../../../lib/soda-release-tools/src/artifacts.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–208, 283–314, 335–395; lines 10–12: USAGE and attached behavior; attached module comments and attributes; lines 13–13: const HELP_REQUESTED; lines 14–73: fn command; lines 74–79: fn help_text; lines 80–87: ArtifactFlags and attached behavior; lines 88–137: parse_artifact_flags and attached behavior; lines 138–160: look_path and attached behavior; lines 161–170: is_executable_file and attached behavior; lines 171–191: private_destination and attached behavior; lines 192–208: fresh_directory and attached behavior; lines 283–294: fn admit_coreos_fetch; lines 295–314: admit_coreos_iso_fetch and attached behavior; lines 335–357: fn run_coreos_artifact; lines 358–365: run_artifact_action and attached behavior; lines 366–377: run and attached behavior; lines 378–394: main and attached behavior; lines 395–395: tests and attached behavior | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Artifact acquisition CLI/flags; declaration/member USAGE Adjacent comments and attributes explain this same authored responsibility.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 209–282; lines 209–216: fn open_private_butane; lines 217–247: run_butane and attached behavior; lines 248–263: finish_butane_conversion and attached behavior; lines 264–282: convert_butane and attached behavior | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Explicit native Butane input conversion; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 315–334; lines 315–327: fn open_oci_archive; lines 328–334: inspect_artifact_oci and attached behavior | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | OCI archive identity inspection; OCI archive identity inspection; declaration/member inspect_artifact_oci — lib/soda-release-tools/src/artifacts.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; lib/soda-release-tools/src/artifacts.rs:328-334; current named unit matched to maintained semantic map and release consumer |

<a id="coverage-3f048bb99508"></a>

## [lib/soda-release-tools/src/artifacts/tests.rs](../../../../../lib/soda-release-tools/src/artifacts/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–461; artifact CLI flags, private output/Butane lifecycle, OCI refusals and inspect_positive_returns_image_identity_json | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | CLI OCI output retains all seven image-identity fields and no trailing newline through semantic JSON assertions; no Go escape byte contract |

<a id="coverage-fd3c671bdaf1"></a>
<a id="rustsoda-release-toolssrcbuild_clirs-1"></a>
<a id="coverage-0d2781457ed6"></a>

## [lib/soda-release-tools/src/build_cli.rs](../../../../../lib/soda-release-tools/src/build_cli.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–430; lines 15–23: fn value; attached module comments and attributes; lines 24–36: fn boolean; lines 37–97: fn command; lines 98–102: BuildFlags and attached behavior; lines 103–171: parse_build_flags and attached behavior; lines 172–183: admit_worker_build and attached behavior; lines 184–201: admit_parent_dispatch and attached behavior; lines 202–209: admit_build_dispatch and attached behavior; lines 210–213: usage and attached behavior; lines 214–227: sanitize_build_env and attached behavior; lines 228–234: watch_build_signals and attached behavior; lines 235–242: progress_title and attached behavior; lines 243–256: controller_revision and attached behavior; lines 257–275: forgejo_git_output and attached behavior; lines 276–290: validate_forgejo_checkout_root and attached behavior; lines 291–311: forgejo_checkout_revision and attached behavior; lines 312–321: bind_forgejo_source and attached behavior; lines 322–332: bind_build_source and attached behavior; lines 333–344: print_build_artifacts and attached behavior; lines 345–359: run_parent_build and attached behavior; lines 360–370: map_worker_error and attached behavior; lines 371–420: run and attached behavior; lines 421–429: main and attached behavior; lines 430–430: tests and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Current fn value and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3fe6f918e31c"></a>
<a id="rustsoda-release-toolssrccandidaters-1"></a>
<a id="coverage-2f2b482ff150"></a>

## [lib/soda-release-tools/src/build_cli/tests.rs](../../../../../lib/soda-release-tools/src/build_cli/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–187; live_inputs_admission_boundary test | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Exercise build CLI admission of live-input requests and refusal before dispatch. — Current source inspected at lib/soda-release-tools/src/build_cli/tests.rs; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-3384431c4595"></a>

## [lib/soda-release-tools/src/build_spec.rs](../../../../../lib/soda-release-tools/src/build_spec.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–267; lines 1–7: use url and attached body; lines 8–24: struct Request and attached body; lines 25–38: struct ImageResult and attached body; lines 39–40: impl Request and attached body; lines 41–43: fn wants_media and attached body; lines 44–51: fn purpose and attached body; lines 52–59: fn requested_target and attached body; lines 60–72: fn validate_development_target and attached body; lines 73–90: fn validate_media_inputs and attached body; lines 91–98: fn validate_target and attached body; lines 99–139: fn media_base_url and attached body; lines 140–160: fn valid_percent_escapes and attached body; lines 161–161: mod tests and attached body; lines 162–162: use super and attached body; lines 163–177: fn dev_request and attached body; lines 178–214: fn target_validation_matrix and attached body; lines 215–257: fn media_url_boundaries and attached body; lines 258–267: fn purpose_and_requested_target and attached body | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Declaration block for use url in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-912dc65fa693"></a>

## [lib/soda-release-tools/src/candidate/mod.rs](../../../../../lib/soda-release-tools/src/candidate/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; lines 7–8: mod options; attached module comments and attributes; lines 14–16: ExitError and attached behavior; lines 30–62: run_candidate and attached behavior; lines 63–66: CandidateError and attached behavior; lines 76–87: main and attached behavior; lines 88–88: tests and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol Adjacent comments and attributes explain this same authored responsibility.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-745e2c4c5995"></a>

## [lib/soda-release-tools/src/candidate/options.rs](../../../../../lib/soda-release-tools/src/candidate/options.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–344; lines 9–20: Options and attached behavior; attached module comments and attributes; lines 21–29: fn value; lines 30–79: fn command; lines 80–80: ARCH_DEFAULT and attached behavior; lines 81–84: usage and attached behavior; lines 85–95: native_arch and attached behavior; lines 96–102: validate_mode_flag and attached behavior; lines 103–109: validate_arch_flag and attached behavior; lines 110–168: parse_options and attached behavior; lines 169–176: valid_out_leaf and attached behavior; lines 177–196: validate_common_paths and attached behavior; lines 197–203: validate_candidate and attached behavior; lines 204–213: validate_media and attached behavior; lines 214–222: validate_resolved and attached behavior; lines 223–229: check_checkout_root and attached behavior; lines 230–240: dirty_files and attached behavior; lines 241–271: check_clean_tree and attached behavior; lines 272–288: check_fresh_out and attached behavior; lines 289–298: preflight and attached behavior; lines 299–312: monotonic_ns and attached behavior; lines 313–322: describe and attached behavior; lines 323–339: resolve_options and attached behavior; lines 340–344: ready_run and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol; declaration/member Options Adjacent comments and attributes explain this same authored responsibility.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a02a976d734b"></a>

## [lib/soda-release-tools/src/candidate/tests.rs](../../../../../lib/soda-release-tools/src/candidate/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; lines 2–16: base_options and attached behavior; attached module comments and attributes; lines 17–24: arch_flag_admits_only_x86_64 and attached behavior; lines 25–46: fn parser_uses_double_dash_equals_and_last_scalar_value; lines 47–65: resolved_boundaries and attached behavior; lines 66–86: out_leaf_follows_worker_name_rule and attached behavior; lines 87–93: dirty_files_skips_blanks and attached behavior; lines 94–118: resolve_options_defers_worker_admission_to_controller and attached behavior; lines 119–135: check_fresh_out_matrix and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol; declaration/member base_options Adjacent comments and attributes explain this same authored responsibility.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ceee3984d945"></a>

## [lib/soda-release-tools/src/candidate_controller.rs](../../../../../lib/soda-release-tools/src/candidate_controller.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–214; lines 1–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–8: use url and attached body; lines 9–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–24: static CHILD_PID and attached body; lines 25–35: fn controller_env and attached body; lines 36–46: fn controller_argv and attached body; lines 47–53: struct ControllerRun and attached body; lines 54–55: impl Drop and attached body; lines 56–64: fn drop and attached body; lines 65–108: fn start_controller_run and attached body; lines 109–110: impl ControllerRun and attached body; lines 111–134: fn relay_output and attached body; lines 135–145: fn wait_exit and attached body; lines 146–153: fn wait and attached body; lines 154–163: fn maybe_serve_fixture and attached body; lines 164–174: fn file_built_rootfs and attached body; lines 175–195: fn url_hostname and attached body; lines 196–214: fn valid_percent_escapes and attached body | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-aff95eb37055"></a>

## [lib/soda-release-tools/src/candidate_display/events.rs](../../../../../lib/soda-release-tools/src/candidate_display/events.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; lines 1–11: Event and attached behavior; lines 12–24: parse_event and attached behavior; lines 25–37: parse_start_event and attached behavior; lines 38–53: parse_done_event and attached behavior; lines 54–65: apply_duration_part and attached behavior; lines 66–78: parse_artifact_event and attached behavior; lines 79–96: host_artifact_path and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol; declaration/member Event; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9cc191a395af"></a>

## [lib/soda-release-tools/src/candidate_display/mod.rs](../../../../../lib/soda-release-tools/src/candidate_display/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–394; lines 9–10: mod events; attached module comments and attributes; lines 15–20: Phase and attached behavior; lines 21–22: SPINNER and attached behavior; lines 23–23: LOG_VIEWPORT and attached behavior; lines 24–24: CLOSED_PHASE_KEEP and attached behavior; lines 25–39: RendererInner and attached behavior; lines 45–49: TickerHandle and attached behavior; lines 50–51: impl Drop; lines 52–58: fn drop; lines 61–79: new and attached behavior; lines 80–83: set_out_dir and attached behavior; lines 84–140: feed and attached behavior; lines 141–145: note and attached behavior; lines 146–165: note_locked and attached behavior; lines 166–190: draw_locked and attached behavior; lines 191–206: close_phase and attached behavior; lines 207–215: phase_mark and attached behavior; lines 216–258: build_lines and attached behavior; lines 259–271: phase_line and attached behavior; lines 275–299: start_ticker and attached behavior; lines 300–304: stop_ticker and attached behavior; lines 305–324: finish and attached behavior; lines 325–349: print_why_panel_locked and attached behavior; lines 350–357: exit_meaning and attached behavior; lines 358–362: wall_duration and attached behavior; lines 363–366: wall_since and attached behavior; lines 367–374: truncate and attached behavior; lines 375–381: is_terminal and attached behavior; lines 382–393: term_width and attached behavior; lines 394–394: tests and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol Adjacent comments and attributes explain this same authored responsibility.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1b03bde431f1"></a>

## [lib/soda-release-tools/src/candidate_display/tests.rs](../../../../../lib/soda-release-tools/src/candidate_display/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–240; host_artifact_path_leaves_foreign_paths_alone test | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Verify candidate display preserves an artifact path outside the candidate root rather than claiming it as generated output. — Current source inspected at lib/soda-release-tools/src/candidate_display/tests.rs; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-a7c0c26ef802"></a>

## [lib/soda-release-tools/src/candidate_fixture.rs](../../../../../lib/soda-release-tools/src/candidate_fixture.rs)

Current module body and direct consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–432; whole file: test/development rootfs fixture http server for candidate setup; it serves caller-owned fixture trees and does not create a product service | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test/development rootfs fixture HTTP server for candidate setup; it serves caller-owned fixture trees and does not create a product service. — lib/soda-release-tools/src/candidate_fixture.rs:1-432; module purpose and current direct consumer inspected |

<a id="coverage-885ca64a646f"></a>

## [lib/soda-release-tools/src/candidate_fixture/tests.rs](../../../../../lib/soda-release-tools/src/candidate_fixture/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–197; lines 1–1: use super and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: static COUNTER and attached body; lines 6–18: fn temp_dir and attached body; lines 19–37: fn fixture_wanted_matrix and attached body; lines 38–50: fn fixture_addr_matrix and attached body; lines 51–75: fn http_get and attached body; lines 76–95: fn serve_and_file_rootfs and attached body; lines 96–106: fn serve_fixture_busy_port_is_not_an_error and attached body; lines 107–128: fn fixture_stop_closes_partial_request_and_drop_joins_server and attached body; lines 129–142: fn fixture_refuses_symlink_escape and attached body; lines 143–174: fn fixture_stop_joins_active_large_file_stream and attached body; lines 175–190: fn copy_file_refuses_occupied_pickup and attached body; lines 191–197: fn copy_built_rootfs_needs_an_image and attached body | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-11a70dac03f3"></a>
<a id="coverage-df6303a40839"></a>

## [lib/soda-release-tools/src/candidate_hints.rs](../../../../../lib/soda-release-tools/src/candidate_hints.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–91; lines 1–49: struct Hint, const HINT_CATALOG, fn failure_hint; attached module comments and attributes; lines 50–50: mod tests; lines 54–91: fn known_signatures_map_to_fixes | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol Adjacent comments and attributes explain this same authored responsibility.; Source-level oracle/test assertions for native builder admission, controllers and restricted worker protocol; Current-source audit correction: `known_signatures_map_to_fixes` asserts sandbox/cache/worker provisioning diagnostic hints. It does not exercise signing custody. Root inspected the complete current body for D08's delegated review; retain the test with its D01 producer. Frozen catalog assignment remains historical. — lib/soda-release-tools/src/candidate_hints.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-be5efdb13453"></a>

## [lib/soda-release-tools/src/candidate_prompts/defaults.rs](../../../../../lib/soda-release-tools/src/candidate_prompts/defaults.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–90; lines 3–4: FIXTURE_ROOTFS_URL and attached behavior; attached module comments and attributes; lines 5–6: STANDARD_CONTROLLER_PATHS and attached behavior; lines 7–8: STANDARD_WORKER_CONFIG_PATHS and attached behavior; lines 9–13: MODE_OPTIONS and attached behavior; lines 14–21: mode_label and attached behavior; lines 22–30: mode_index and attached behavior; lines 31–42: default_path and attached behavior; lines 43–46: file_exists and attached behavior; lines 47–62: suggest_out and attached behavior; lines 63–90: controller_args and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol; declaration/member FIXTURE_ROOTFS_URL Adjacent comments and attributes explain this same authored responsibility.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-02ef2862da4f"></a>
<a id="rustsoda-release-toolssrccandidate_promptsrs-1"></a>
<a id="coverage-a6945127e197"></a>

## [lib/soda-release-tools/src/candidate_prompts/mod.rs](../../../../../lib/soda-release-tools/src/candidate_prompts/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–349; lines 7–8: mod defaults; attached module comments and attributes; lines 11–18: Prompter and attached behavior; lines 19–25: new and attached behavior; lines 26–37: with_exists and attached behavior; lines 38–58: line and attached behavior; lines 59–74: choice and attached behavior; lines 75–83: edit_mode and attached behavior; lines 84–88: edit_out and attached behavior; lines 89–94: edit_controller and attached behavior; lines 95–100: edit_worker_config and attached behavior; lines 101–105: edit_rootfs_url and attached behavior; lines 106–111: edit_fast_compress and attached behavior; lines 112–119: edit_repo_prefix and attached behavior; lines 120–132: ask_absolute and attached behavior; lines 133–145: ask_non_empty and attached behavior; lines 146–176: ask_out and attached behavior; lines 177–191: ask_fast and attached behavior; lines 192–199: show_fast_compress and attached behavior; lines 200–220: render_overview and attached behavior; lines 221–251: overview and attached behavior; lines 252–256: exists_as_fn and attached behavior; lines 257–272: edit_field and attached behavior; lines 273–291: overview_rows and attached behavior; lines 292–310: out_status and attached behavior; lines 311–314: parent_dir_exists and attached behavior; lines 315–318: path_absent and attached behavior; lines 319–338: default_overview and attached behavior; lines 339–348: prompter_overview and attached behavior; lines 349–349: tests and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol Adjacent comments and attributes explain this same authored responsibility.; 29 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-68dcd9af1971"></a>

## [lib/soda-release-tools/src/candidate_prompts/tests.rs](../../../../../lib/soda-release-tools/src/candidate_prompts/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–203; lines 6–9: Shared and attached behavior; attached module comments and attributes; lines 10–11: impl Write; lines 12–15: write and attached behavior; lines 16–20: flush and attached behavior; lines 21–40: scripted and attached behavior; lines 41–62: scripted_exists and attached behavior; lines 63–73: fresh_options and attached behavior; lines 74–89: controller_args_cover_modes and attached behavior; lines 90–96: overview_starts_when_valid and attached behavior; lines 97–105: overview_edits_field and attached behavior; lines 106–115: overview_blocks_bad_start_without_losing_answers and attached behavior; lines 116–138: overview_prefills_standard_paths and attached behavior; lines 139–156: overview_defaults_fixture_rootfs_url and attached behavior; lines 157–175: mode_switch_restores_fixture_url and attached behavior; lines 176–194: ask_out_rejects_then_accepts and attached behavior; lines 195–203: suggested_out_follows_worker_rule and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Native builder admission, controllers and restricted worker protocol; declaration/member Shared Adjacent comments and attributes explain this same authored responsibility.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-91fab23c2a2b"></a>

## [lib/soda-release-tools/src/check_cli.rs](../../../../../lib/soda-release-tools/src/check_cli.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–220; lines 1–6: use clap and attached body; lines 7–8: const HELP_REQUESTED and attached body; lines 9–45: fn command and attached body; lines 46–51: fn help_text and attached body; lines 52–57: struct CheckFlags and attached body; lines 58–97: fn parse_check_flags and attached body; lines 98–115: fn run and attached body; lines 116–128: fn main and attached body; lines 129–129: mod tests and attached body; lines 130–130: use super and attached body; lines 131–134: fn parse_list and attached body; lines 135–140: fn run_list and attached body; lines 141–205: fn flag_parse_matrix and attached body; lines 206–220: fn nonexistent_candidate_dir_errors and attached body | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declaration block for use clap in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-89361860814d"></a>

## [lib/soda-release-tools/src/digest.rs](../../../../../lib/soda-release-tools/src/digest.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; lines 1–4: use sha2 and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–11: fn is_revision and attached body; lines 12–15: fn is_digest and attached body; lines 16–21: fn is_signer and attached body; lines 22–27: fn oci_architecture and attached body; lines 28–37: fn require_native and attached body; lines 38–59: fn hash_file and attached body; lines 60–64: fn eval_symlinks and attached body; lines 65–65: mod tests and attached body; lines 66–68: use super and attached body; lines 69–80: fn revision_and_digest_shapes and attached body; lines 81–89: fn native_arch_admission and attached body | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Declaration block for use sha2 in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-341f9a7ce004"></a>

## [lib/soda-release-tools/src/exitcode.rs](../../../../../lib/soda-release-tools/src/exitcode.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–97; lines 1–6: use std and attached body; lines 7–7: struct Interrupted and attached body; lines 8–9: impl std and attached body; lines 10–13: fn fmt and attached body; lines 14–15: impl Interrupted and attached body; lines 16–19: fn exit_code and attached body; lines 20–21: static INTERRUPT_CODE and attached body; lines 22–25: fn note_interrupt and attached body; lines 26–35: fn take_interrupt and attached body; lines 36–40: fn has_interrupt and attached body; lines 41–44: static INTERRUPT_TEST_LOCK and attached body; lines 45–50: fn interrupt_test_lock and attached body; lines 51–74: fn build_exit_code and attached body; lines 75–80: enum ToolError and attached body; lines 81–82: impl std and attached body; lines 83–91: fn fmt and attached body; lines 92–93: impl ToolError and attached body; lines 94–97: fn msg and attached body | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e4598a553a79"></a>

## [lib/soda-release-tools/src/lib.rs](../../../../../lib/soda-release-tools/src/lib.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–22; lines 1–8: mod artifacts and attached body; lines 9–9: mod build_cli and attached body; lines 10–10: mod build_spec and attached body; lines 11–11: mod candidate and attached body; lines 12–12: mod candidate_controller and attached body; lines 13–13: mod candidate_display and attached body; lines 14–14: mod candidate_fixture and attached body; lines 15–15: mod candidate_hints and attached body; lines 16–16: mod candidate_prompts and attached body; lines 17–17: mod check_cli and attached body; lines 18–18: mod digest and attached body; lines 19–19: mod exitcode and attached body; lines 20–20: mod pipeline and attached body; lines 21–21: mod progress and attached body; lines 22–22: mod worker and attached body | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Declaration block for mod artifacts in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0092d269fe11"></a>

## [lib/soda-release-tools/src/pipeline.rs](../../../../../lib/soda-release-tools/src/pipeline.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–174, 251–304, 309–347, 440–497, 542–1068; lines 55–74: image_request and attached behavior; attached module comments and attributes; lines 75–90: image_result and attached behavior; lines 91–94: build_err and attached behavior; lines 95–105: deliver_err and attached behavior; lines 106–117: image_of and attached behavior; lines 118–125: produced_of and attached behavior; lines 126–134: coreos_image_of and attached behavior; lines 135–140: sorted_pairs and attached behavior; lines 141–149: sorted_images and attached behavior; lines 150–159: resolved_coreos_of and attached behavior; lines 160–167: tailnet_of and attached behavior; lines 168–174: live_inputs_of and attached behavior; lines 251–255: RealProduction and attached behavior; lines 256–259: new and attached behavior; lines 260–261: impl ImageProduction; lines 262–264: source and attached behavior; lines 265–268: forgejo_source and attached behavior; lines 269–272: forgejo_revision and attached behavior; lines 273–276: native and attached behavior; lines 277–280: out and attached behavior; lines 281–284: arch and attached behavior; lines 285–288: revision and attached behavior; lines 289–292: live_inputs and attached behavior; lines 293–296: execute and attached behavior; lines 297–300: capture and attached behavior; lines 309–312: fn dependencies; lines 313–316: compile and attached behavior; lines 317–322: compile_rust and attached behavior; lines 323–326: stage_fork_binary and attached behavior; lines 327–332: assets and attached behavior; lines 333–347: images and attached behavior; lines 440–441: impl ImageProgress; lines 442–444: fn phase; lines 449–452: end_phase and attached behavior; lines 453–456: end and attached behavior; lines 457–460: create_log and attached behavior; lines 461–475: note_reason and attached behavior; lines 476–478: CURRENT_RUNNER and attached behavior; lines 479–497: with_current_runner and attached behavior; lines 542–546: struct CancellationWatcher; lines 547–548: impl CancellationWatcher; lines 549–565: fn start; lines 566–567: impl Drop; lines 568–578: fn drop; lines 579–591: fn run_worker_stage; lines 592–592: mod tests; lines 597–616: spec_request and attached behavior; lines 617–635: request_mapping_round_trip and attached behavior; lines 636–668: result_mapping_drops_checks and attached behavior; lines 669–688: test_progress and attached behavior; lines 689–715: progress_adapter_call_through and attached behavior; lines 716–751: fixture_inner and attached behavior; lines 752–768: accessors_passthrough and attached behavior; lines 769–783: hooks_call_through and attached behavior; lines 784–823: hook_errors_map and attached behavior; lines 824–893: build_model_conversions and attached behavior; lines 894–965: deliver_model_conversions and attached behavior; lines 966–1005: deliver_refusals_map and attached behavior; lines 1006–1039: factory_wires_inputs_and_runner_hooks and attached behavior; lines 1040–1051: cancel_seeds_from_interrupt_and_restores and attached behavior; lines 1052–1068: fn late_interrupt_reaches_live_build_cancellation | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Native production interface and existing candidate result conversion; declaration/member image_request Adjacent comments and attributes explain this same authored responsibility.; 61 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 175–211, 348–368, 381–389; lines 175–211: fn deliver_payload_of; lines 348–358: fn inspect_oci; lines 359–368: verify_content and attached behavior; lines 381–389: fn check_native | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Immutable payload byte contract conversion; 4 named units assigned here; remaining selectors preserve each duty — lib/soda-release-tools/src/pipeline.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; lib/soda-release-tools/src/pipeline.rs:359-368; current named unit matched to maintained semantic map and release consumer |
| 212–224, 430–439; lines 212–224: fn deliver_trust_of; lines 430–439: fn write_document | [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) | retained | Release trust contract conversion; Immutable metadata document adapter — lib/soda-release-tools/src/pipeline.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 225–250, 390–414; lines 225–234: fn deliver_permit_of; lines 235–250: deliver_secrets_of and attached behavior; lines 390–414: fn sign_media | [D08](../../slices/release-and-installation.md#d08-signing-custody) | retained | Explicit signing permit/protected signer contract conversion; Explicit signing permit/protected signer contract conversion; declaration/member deliver_secrets_of; Native exact digest signing adapter — lib/soda-release-tools/src/pipeline.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; lib/soda-release-tools/src/pipeline.rs:235-250; current named unit matched to maintained semantic map and release consumer |
| 305–308, 369–380; lines 305–308: fn resolve_inputs; lines 369–374: fn resolve_core_os; lines 375–380: read_live_inputs and attached behavior | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Pinned live input adapter; Pinned CoreOS/live-input adapters; Pinned CoreOS/live-input adapters; declaration/member read_live_inputs — lib/soda-release-tools/src/pipeline.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; lib/soda-release-tools/src/pipeline.rs:375-380; current named unit matched to maintained semantic map and release consumer |
| 415–429; lines 415–429: fn verify_copy | [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) | retained | Native authenticated verification/copy adapter — lib/soda-release-tools/src/pipeline.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 498–541; lines 498–533: fn make_production; lines 534–541: seeded_cancel and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Admit isolated exact native production configuration; Admit isolated exact native production configuration; declaration/member seeded_cancel — lib/soda-release-tools/src/pipeline.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; lib/soda-release-tools/src/pipeline.rs:534-541; current named unit matched to maintained semantic map and release consumer |

<a id="coverage-bd0120a8fe6d"></a>
<a id="rustsoda-release-toolssrcprogressrs-1"></a>
<a id="coverage-497f713e837f"></a>

## [lib/soda-release-tools/src/progress.rs](../../../../../lib/soda-release-tools/src/progress.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–320; lines 11–22: monotonic and attached behavior; attached module comments and attributes; lines 23–28: format_duration and attached behavior; lines 29–36: failed_text and attached behavior; lines 37–46: finish_kind and attached behavior; lines 47–63: BuildProgress and attached behavior; lines 64–66: new and attached behavior; lines 67–103: new_with_clock and attached behavior; lines 104–107: capture and attached behavior; lines 108–123: note_reason and attached behavior; lines 124–149: emit and attached behavior; lines 150–167: create_log and attached behavior; lines 168–179: phase and attached behavior; lines 180–189: kind_for and attached behavior; lines 190–193: end_phase and attached behavior; lines 194–214: end_phase_opt and attached behavior; lines 215–225: next and attached behavior; lines 226–229: end and attached behavior; lines 230–250: end_opt and attached behavior; lines 251–275: write_section_summary and attached behavior; lines 276–319: finish and attached behavior; lines 320–320: tests and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Builder progress events and retained log output; declaration/member monotonic Adjacent comments and attributes explain this same authored responsibility.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-979c2fcb215a"></a>
<a id="rustsoda-release-toolssrcworkerrs-1"></a>
<a id="coverage-eb0374de0b7e"></a>

## [lib/soda-release-tools/src/progress/tests.rs](../../../../../lib/soda-release-tools/src/progress/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–87; lines 4–10: EnvGuard and attached behavior; attached module comments and attributes; lines 11–16: set and attached behavior; lines 17–18: impl Drop; lines 19–27: drop and attached behavior; lines 28–35: fixed_clock and attached behavior; lines 36–37: ENV_LOCK and attached behavior; lines 38–53: fresh_progress and attached behavior; lines 54–69: phase_lifecycle_emits_wire_format and attached behavior; lines 70–82: failure_carries_reason and attached behavior; lines 83–87: bad_inherited_origin_refused and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Builder progress events and retained log output; declaration/member EnvGuard Adjacent comments and attributes explain this same authored responsibility.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3124257f8df7"></a>

## [lib/soda-release-tools/src/worker/config.rs](../../../../../lib/soda-release-tools/src/worker/config.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–370; lines 10–11: WORKER_SOURCE and attached behavior; attached module comments and attributes; lines 12–12: WORKER_HOME and attached behavior; lines 13–13: WORKER_RUNTIME and attached behavior; lines 14–14: WORKER_TOOLS and attached behavior; lines 15–15: WORKER_FORGEJO and attached behavior; lines 16–18: PINNED_GO_ROOT and attached behavior; lines 19–31: WorkerConfig and attached behavior; lines 32–41: Worker and attached behavior; lines 42–47: euid and attached behavior; lines 48–68: lookup_user and attached behavior; lines 69–76: build_worker_identity and attached behavior; lines 77–80: worker_runtime_ids and attached behavior; lines 81–95: go_clean and attached behavior; lines 96–99: is_abs_clean and attached behavior; lines 100–111: valid_worker_task_paths and attached behavior; lines 112–119: parent_dir and attached behavior; lines 120–129: worker_executable_matches and attached behavior; lines 130–134: eval_strict and attached behavior; lines 135–147: valid_worker_path and attached behavior; lines 148–171: trusted_executable and attached behavior; lines 172–179: admit_loaded_worker_config and attached behavior; lines 180–195: worker_config_paths and attached behavior; lines 196–213: admit_storage_root and attached behavior; lines 214–235: private_file and attached behavior; lines 236–239: struct ConfigFields; lines 240–243: fn deserialize; lines 244–245: struct FieldsVisitor; lines 246–246: type Value; lines 247–250: fn expecting; lines 251–265: fn visit_map; lines 266–277: strict_string_field and attached behavior; lines 278–279: decode_worker_config and attached behavior; lines 280–319: KNOWN and attached behavior; lines 320–330: read_worker_config and attached behavior; lines 331–351: load_worker_config and attached behavior; lines 352–352: mod tests; lines 356–370: fn configuration_checks_every_known_occurrence_and_rejects_unknown_fields | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Restricted controller worker identity/source/mount/input/environment/lease admission; declaration/member WORKER_SOURCE Adjacent comments and attributes explain this same authored responsibility.; 37 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f49506b845c3"></a>

## [lib/soda-release-tools/src/worker/execution.rs](../../../../../lib/soda-release-tools/src/worker/execution.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–400; lines 15–31: valid_worker_identity and attached behavior; attached module comments and attributes; lines 32–52: append_bind_paths and attached behavior; lines 53–67: allowed_worker_env_key and attached behavior; lines 68–87: append_worker_env and attached behavior; lines 88–120: worker_argv and attached behavior; lines 121–126: enum PipeRead; lines 127–128: struct ChildGuard; lines 131–131: type Target; lines 132–135: fn deref; lines 138–141: fn deref_mut; lines 142–143: impl Drop; lines 144–148: fn drop; lines 149–157: fn set_nonblocking; lines 158–174: fn read_available; lines 175–212: fn stop_worker_unit; lines 213–216: WorkerError and attached behavior; lines 217–218: impl From; lines 219–229: from and attached behavior; lines 230–400: run_worker and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Restricted controller worker identity/source/mount/input/environment/lease admission; declaration/member valid_worker_identity Adjacent comments and attributes explain this same authored responsibility.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-43c5ab132d32"></a>

## [lib/soda-release-tools/src/worker/mod.rs](../../../../../lib/soda-release-tools/src/worker/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; lines 1–8: mod config, mod execution, mod runtime; attached module comments and attributes; lines 15–15: tests and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Restricted controller worker identity/source/mount/input/environment/lease admission Adjacent comments and attributes explain this same authored responsibility.; Exact worker admission and native build handoff; declaration/member tests — lib/soda-release-tools/src/worker/mod.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source; lib/soda-release-tools/src/worker/mod.rs:15-15; current named unit matched to maintained semantic map and release consumer |

<a id="coverage-af92c4cfec63"></a>

## [lib/soda-release-tools/src/worker/runtime.rs](../../../../../lib/soda-release-tools/src/worker/runtime.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–511; resolve_live_inputs | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Resolve the controller-supplied live release inputs for worker execution; this is input-source selection, not candidate construction. — Current source inspected at lib/soda-release-tools/src/worker/runtime.rs; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-2fd37711a534"></a>

## [lib/soda-release-tools/src/worker/tests.rs](../../../../../lib/soda-release-tools/src/worker/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–634; worker_forwards_controller_live_inputs test | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Verify the release worker forwards controller-selected live inputs into the build request. — Current source inspected at lib/soda-release-tools/src/worker/tests.rs; concrete renderer/build/test consumer is named in the selector and description. |

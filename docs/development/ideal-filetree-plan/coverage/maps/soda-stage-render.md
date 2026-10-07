# Soda stage render

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-d04d16b4f1e0"></a>
<a id="coverage-7c05b7dc805e"></a>

## [tools/release-assets/src/render/mod.rs](../../../../../tools/release-assets/src/render/mod.rs)

Current helpers and module consumers inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; renderer crate documentation and compatibility envelope | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Documents the three build-time render/stage binaries and their source-tree invocation contract. — tools/release-assets/src/render/mod.rs:1-15; Cargo binary entrypoints consume each module |
| 16; provisioning renderer module export | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Exports the Butane/provisioning media renderer used to produce authenticated installation configuration. — tools/release-assets/src/render/mod.rs:16; consumer tools/release-assets/src/bin/render-provisioning.rs |
| 17; staging renderer module export | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Exports staged host/image content generation used by candidate production. — tools/release-assets/src/render/mod.rs:17; consumer tools/release-assets/src/bin/stage.rs and candidate build pipeline |
| 18; terminal logo renderer module export | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Exports the canonical terminal emblem renderer that generates branding output. — tools/release-assets/src/render/mod.rs:18; consumer tools/release-assets/src/bin/render-terminal-logo.rs |
| 19–50; path imports and source root/file mode helpers | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Locates the source checkout from the branding payload manifest and applies explicit Unix file modes required by staging. — tools/release-assets/src/render/mod.rs:20-50; source_root checks frontend/forgejo/payload.json, chmod writes mode |
| 51–88; long-form CLI flag tokenization helper; SHA-256 lowercase hexadecimal encoder and its tests module | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Splits `--long` and `--long=value` tokens for compatibility with the source argparse scripts.; Encodes SHA-256 bytes as lowercase hexadecimal for render/staging output. — tools/release-assets/src/render/mod.rs:52-73; consumed by render binary parsers; tools/release-assets/src/render/mod.rs:75-88; tests module is source assertion of helper behavior |

<a id="coverage-abe4dadfc228"></a>

## [tools/release-assets/src/render/provisioning/document.rs](../../../../../tools/release-assets/src/render/provisioning/document.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–67; `files_mut` (10–26), `push_file` (28–31), `clear_files` (33–36), `file_entry` (38–44), `public_config` (46–67) | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Build a JSON Value from the live base document and branding SVG, retaining the single document owner. Object access uses `Value::as_object_mut` at the actual mutation sites; no forwarding formatter/value helpers remain. — Current renderer caller inspected |

<a id="coverage-90df13df2cca"></a>
<a id="rustsoda-stage-rendersrcprovisioningrs-1"></a>
<a id="coverage-fb2d056df18b"></a>

## [tools/release-assets/src/render/provisioning/mod.rs](../../../../../tools/release-assets/src/render/provisioning/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–86; lines 1–8: mod document; attached module comments and attributes; lines 9–9: mod private_files; lines 10–12: mod render; lines 13–13: tests and attached behavior; lines 27–41: ProvKind and attached behavior; lines 42–59: name and attached behavior; lines 60–65: ProvError and attached behavior; lines 66–71: new and attached behavior; lines 72–75: value and attached behavior; lines 76–86: io and attached behavior | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Current mod document and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8317f13a19e2"></a>

## [tools/release-assets/src/render/provisioning/private_files.rs](../../../../../tools/release-assets/src/render/provisioning/private_files.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–319; lines 11–21: read_text and attached behavior; attached module comments and attributes; lines 22–48: fn read_open_text; lines 49–68: regular and attached behavior; lines 78–97: is_label and attached behavior; lines 98–103: is_appliance_hostname and attached behavior; lines 104–111: is_fixture_hostname and attached behavior; lines 112–122: absolute_lexical and attached behavior; lines 123–143: derive_host_public and attached behavior; lines 144–244: const PUBLIC_LIMIT; lines 245–259: fn set_nonblocking; lines 260–272: is_real_parent and attached behavior; lines 273–288: write_exclusive and attached behavior | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Public bootstrap plus private per-instance operator input renderer; declaration/member read_text Adjacent comments and attributes explain this same authored responsibility.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c078f6dcbfb8"></a>

## [tools/release-assets/src/render/provisioning/render.rs](../../../../../tools/release-assets/src/render/provisioning/render.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101; `RenderInputs` (11–19), `render` (23–101) | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Merge the public base/SVG document with admitted private inputs and emit standard pretty JSON; the final exclusive writer retains mode 0600 and appends LF. — Current producer and final writer inspected |

<a id="coverage-3669d4c2bc10"></a>

## [tools/release-assets/src/render/provisioning/tests.rs](../../../../../tools/release-assets/src/render/provisioning/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–149; lines 6–6: fn regular_input_reads_admitted_open_file_with_cap_and_private_policy; attached module comments and attributes; lines 52–76: fn hostnames_follow_the_script_regexes; lines 77–88: dump_matches_json_indent_two and attached behavior; lines 89–97: dump_keeps_number_literals_and_key_order and attached behavior; lines 98–103: fn dump_preserves_valid_exponent_outside_machine_float_range; lines 104–115: fn document_edits_use_last_exact_member_and_keep_duplicate_pairs_and_raw_numbers; lines 116–149: fn provisioning_dynamic_depth_limit_keeps_safe_error_classification | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Current fn regular_input_reads_admitted_open_file_with_cap_and_private_policy and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bb512ffa23de"></a>

## [tools/release-assets/src/render/stage/branding.rs](../../../../../tools/release-assets/src/render/stage/branding.rs)

Current body and actual staging consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; current block: canonical forgejo favicon png frames are encoded into the ico container used by staged branding | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Canonical Forgejo favicon PNG frames are encoded into the ICO container used by staged branding. — tools/release-assets/src/render/stage/branding.rs:[[1, 24]]; consumed by tools/release-assets/src/render/stage/mod.rs and stage command |

<a id="coverage-cb8c81e41847"></a>

## [tools/release-assets/src/render/stage/files.rs](../../../../../tools/release-assets/src/render/stage/files.rs)

Current body and actual staging consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; current block: build-context platform admission for staging | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Build-context platform admission for staging. — tools/release-assets/src/render/stage/files.rs:[[1, 17]]; consumed by tools/release-assets/src/render/stage/mod.rs and stage command |
| 18–141; current block: fresh-directory copy, recursive tree normalization, and protected file-mode utilities used by staging | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Fresh-directory copy, recursive tree normalization, and protected file-mode utilities used by staging. — tools/release-assets/src/render/stage/files.rs:[[18, 141]]; consumed by tools/release-assets/src/render/stage/mod.rs and stage command |

<a id="coverage-020d78224a6c"></a>

## [tools/release-assets/src/render/stage/mod.rs](../../../../../tools/release-assets/src/render/stage/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–261; lines 1–8: mod branding; attached module comments and attributes; lines 9–9: mod files; lines 10–12: mod payload; lines 13–13: mod tests; lines 28–33: StageError and attached behavior; lines 34–36: refusal and attached behavior; lines 37–45: failure and attached behavior; lines 46–65: check_contexts and attached behavior; lines 66–261: run and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current mod branding and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f9c661056f1c"></a>

## [tools/release-assets/src/render/stage/payload.rs](../../../../../tools/release-assets/src/render/stage/payload.rs)

Current body and actual staging consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38, 165–170; payload manifest source lookup and source-root resolution | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Reads staged Forgejo payload entries and resolves source versus @build paths to concrete staging inputs. — Current payload_source/load payload code feeds the stage assembler. |
| 39–164; locked terminal asset lock decoding | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Decodes lock-selected terminal asset paths and SHA-256 values for downstream pinned-byte admission. — Current lock deserialization and digest map construction consumed by payload staging. |

<a id="coverage-e40feaacfc57"></a>

## [tools/release-assets/src/render/stage/tests.rs](../../../../../tools/release-assets/src/render/stage/tests.rs)

Current body and actual staging consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 58–69; current block: source assertions for payload manifest selection and staging source roots | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Source assertions for payload manifest selection and staging source roots. — tools/release-assets/src/render/stage/tests.rs:[[1, 37], [58, 70]]; consumed by tools/release-assets/src/render/stage/mod.rs and stage command |
| 38–57; current block: source assertion that staged favicon bytes preserve the expected ico container layout | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Source assertion that staged favicon bytes preserve the expected ICO container layout. — tools/release-assets/src/render/stage/tests.rs:[[38, 57]]; consumed by tools/release-assets/src/render/stage/mod.rs and stage command |

<a id="coverage-79756d7c3d50"></a>

## [tools/release-assets/src/render/terminal_logo/geometry.rs](../../../../../tools/release-assets/src/render/terminal_logo/geometry.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–138; lines 1–67: polygons and attached behavior; lines 68–75: fn finite; lines 76–83: fn require_ring; lines 84–94: inside and attached behavior; lines 95–138: render_layers and attached behavior | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member polygons; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9e2a88b24f24"></a>

## [tools/release-assets/src/render/terminal_logo/mod.rs](../../../../../tools/release-assets/src/render/terminal_logo/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–73; lines 1–3: mod geometry; attached module comments and attributes; lines 4–6: mod svg; lines 7–7: tests and attached behavior; lines 18–42: render_svg and attached behavior; lines 43–73: run and attached behavior | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current mod geometry and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f3cec19cbd4a"></a>

## [tools/release-assets/src/render/terminal_logo/svg.rs](../../../../../tools/release-assets/src/render/terminal_logo/svg.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; lines 1–3: Attrs and attached behavior; lines 4–5: const SVG_NAMESPACE; lines 6–6: const MAX_DOCUMENT_BYTES; lines 7–7: const MAX_DOCUMENT_NODES; lines 8–19: fn attributes; lines 20–25: fn is_svg_element; lines 26–41: fn only_text_or_comments; lines 42–89: parse_svg and attached behavior; lines 90–96: attr and attached behavior | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Attrs; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5b94cb4d3d7f"></a>

## [tools/release-assets/src/render/terminal_logo/tests.rs](../../../../../tools/release-assets/src/render/terminal_logo/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–126; lines 2–9: document and attached behavior; attached module comments and attributes; lines 10–28: fn svg_path_parser_accepts_valid_numbers_and_expands_moveto_pairs; lines 29–48: fn path_geometry_requires_absolute_finite_closed_rings; lines 49–79: svg_gate_rejects_changed_syntax and attached behavior; lines 80–110: fn xml_gate_checks_resolved_namespaces_and_rejects_dtd_text_and_transforms; lines 111–126: tiny_emblem_renders_both_layers and attached behavior | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member document Adjacent comments and attributes explain this same authored responsibility.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e4edbcf809fe"></a>

## [tools/release-assets/src/render/tests.rs](../../../../../tools/release-assets/src/render/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; lines 1–3: use super and attached body; lines 4–17: fn flags_match_argparse_long_forms and attached body; lines 18–23: fn sha256_matches_hashlib and attached body | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn flags_match_argparse_long_forms in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn sha256_matches_hashlib in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ee049481a41e"></a>

## [tools/release-assets/tests/render_provisioning.rs](../../../../../tools/release-assets/tests/render_provisioning.rs)

Current declaration selectors; prior map used only as owner candidate evidence; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–482; provisioning renderer tests and live `prov-root` fixture | [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) | retained | Tests current document values, deterministic rendering, host-key exclusion from argv, private output mode/LF and rejection behavior. The four unused formatter oracles are retired; six renderer checks pass. — Current test source and fixture loader inspected |

<a id="coverage-8c08675160c9"></a>

## [tools/release-assets/tests/render_staging.rs](../../../../../tools/release-assets/tests/render_staging.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–176; lines 1–6: mod render_support; attached module comments and attributes; lines 15–74: stage_help_and_usage and attached behavior; lines 75–156: stage_refuses_bad_contexts_before_touching_the_tree and attached behavior; lines 157–176: stage_reports_a_missing_checkout_root and attached behavior | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current mod render_support and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3a7641dee59c"></a>

## [tools/release-assets/tests/render_support/mod.rs](../../../../../tools/release-assets/tests/render_support/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113; shared render-suite temporary roots, process launch/output capture, repository/fixture lookup, copy and mode helpers | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Reusable test harness consumed by release staging, provisioning, and terminal-logo suites: creates and cleans restricted temporary roots, locates suite binaries/fixtures, copies trees, runs subprocesses with environment, captures results, and reads file modes. It supplies harness mechanics; each suite owns its fixture content and domain assertions. — current source tools/release-assets/tests/render_support/mod.rs; imports at staging/provisioning/logo test modules inspected |

<a id="coverage-1e74a9e55014"></a>

## [tools/release-assets/tests/render_terminal_logo.rs](../../../../../tools/release-assets/tests/render_terminal_logo.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; test imports, shared renderer harness, and logo_root fixture; logo help, canonical output and stale-output tests; logo source admission gate tests | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Builds a temporary branding tree with the canonical SVG source for terminal-logo renderer tests.; Checks terminal logo command help and byte-identical canonical branding outputs, including stale output detection.; Renderer contract assertions pin refusal of altered emblem geometry, viewBox, layer colors, and fill rule before branding output is emitted. — Current named units/source consumers; retained normalized source evidence records each selector |

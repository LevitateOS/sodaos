# Soda stage render

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: provisioning/stage/logo/cli splits re-mapped (18 files); C09 regression test rowed. All rows machine-verified against current bytes.

<a id="coverage-d04d16b4f1e0"></a>

## [tools/release-assets/src/render/mod.rs](../../../../../tools/release-assets/src/render/mod.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; no drift.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–14 | Staging renderer package wiring |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 15 | Provisioning render module wiring |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 16 | Existing outputs staging wiring; declarations/fields: `provisioning` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 17 | Canonical terminal emblem renderer wiring; declarations/fields: `stage` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 18–77 | Source root/private file mode utilities; declarations/fields: `terminal_logo`, `source_root`, `chmod`, `split_flag` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 78–115 | Go/Python flag splitting helpers and source vectors; declarations/fields: `sha256_hex`, `tests`, `flags_match_argparse_long_forms`, `sha256_matches_hashlib` |

<a id="coverage-90df13df2cca"></a>

<a id="rustsoda-stage-rendersrcprovisioningrs-1"></a>

## [tools/release-assets/src/render/provisioning/mod.rs](../../../../../tools/release-assets/src/render/provisioning/mod.rs)

Re-audit @HEAD: C09 consolidation + split: `provisioning.rs` is `provisioning/` with 5 files; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–10 | Public bootstrap plus private per-instance operator input renderer; declarations/fields: `tests`, `ProvKind`, `name`, `ProvError`, `ProvError.kind`, `ProvError.detail`, `new`, `value`, `io` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 12–22 | Public bootstrap plus private per-instance operator input renderer; declaration/member tests; declarations/fields: `tests` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 26–39 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvKind; declarations/fields: `ProvKind` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 42–57 | Public bootstrap plus private per-instance operator input renderer; declaration/member name; declarations/fields: `name` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 59–63 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvError; declarations/fields: `ProvError` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 61 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvError.kind; declarations/fields: `ProvError.kind` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 62 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvError.detail; declarations/fields: `ProvError.detail` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 66–71 | Public bootstrap plus private per-instance operator input renderer; declaration/member new; declarations/fields: `new` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 73–75 | Public bootstrap plus private per-instance operator input renderer; declaration/member value; declarations/fields: `value` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 77–86 | Public bootstrap plus private per-instance operator input renderer; declaration/member io; declarations/fields: `io` |

## [tools/release-assets/src/render/provisioning/document.rs](../../../../../tools/release-assets/src/render/provisioning/document.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–8 | Public bootstrap plus private per-instance operator input renderer; declarations/fields: `escape_python_into`, `indent_into`, `dump_python`, `emit_python`, `object_mut`, `get_mut`, `files_mut`, `not_a_list`, `push_file`, `clear_files`, `str_value`, `file_entry`, `public_config` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 15–40 | Public bootstrap plus private per-instance operator input renderer; declaration/member escape_python_into; declarations/fields: `escape_python_into` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 42–46 | Public bootstrap plus private per-instance operator input renderer; declaration/member indent_into; declarations/fields: `indent_into` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 52–56 | Public bootstrap plus private per-instance operator input renderer; declaration/member dump_python; declarations/fields: `dump_python` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 58–102 | Public bootstrap plus private per-instance operator input renderer; declaration/member emit_python; declarations/fields: `emit_python` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 104–109 | Public bootstrap plus private per-instance operator input renderer; declaration/member object_mut; declarations/fields: `object_mut` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 111–117 | Public bootstrap plus private per-instance operator input renderer; declaration/member get_mut; declarations/fields: `get_mut` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 122–139 | Public bootstrap plus private per-instance operator input renderer; declaration/member files_mut; declarations/fields: `files_mut` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 141–143 | Public bootstrap plus private per-instance operator input renderer; declaration/member not_a_list; declarations/fields: `not_a_list` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 145–153 | Public bootstrap plus private per-instance operator input renderer; declaration/member push_file; declarations/fields: `push_file` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 155–163 | Public bootstrap plus private per-instance operator input renderer; declaration/member clear_files; declarations/fields: `clear_files` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 165–167 | Public bootstrap plus private per-instance operator input renderer; declaration/member str_value; declarations/fields: `str_value` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 169–178 | Public bootstrap plus private per-instance operator input renderer; declaration/member file_entry; declarations/fields: `file_entry` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 182–201 | Public bootstrap plus private per-instance operator input renderer; declaration/member public_config; declarations/fields: `public_config` |

## [tools/release-assets/src/render/provisioning/private_files.rs](../../../../../tools/release-assets/src/render/provisioning/private_files.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–8 | Public bootstrap plus private per-instance operator input renderer; declarations/fields: `read_text`, `regular`, `is_label`, `is_appliance_hostname`, `is_fixture_hostname`, `absolute_lexical`, `derive_host_public`, `is_real_parent`, `write_exclusive` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 10–19 | Public bootstrap plus private per-instance operator input renderer; declaration/member read_text; declarations/fields: `read_text` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 23–39 | Public bootstrap plus private per-instance operator input renderer; declaration/member regular; declarations/fields: `regular` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 41–56 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_label; declarations/fields: `is_label` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 60–63 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_appliance_hostname; declarations/fields: `is_appliance_hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 66–70 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_fixture_hostname; declarations/fields: `is_fixture_hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 74–81 | Public bootstrap plus private per-instance operator input renderer; declaration/member absolute_lexical; declarations/fields: `absolute_lexical` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 85–142 | Public bootstrap plus private per-instance operator input renderer; declaration/member derive_host_public; declarations/fields: `derive_host_public` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 148–160 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_real_parent; declarations/fields: `is_real_parent` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 162–207 | Public bootstrap plus private per-instance operator input renderer; declaration/member write_exclusive; declarations/fields: `write_exclusive` |

## [tools/release-assets/src/render/provisioning/render.rs](../../../../../tools/release-assets/src/render/provisioning/render.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–13 | Public bootstrap plus private per-instance operator input renderer; declarations/fields: `RenderInputs`, `RenderInputs.operator_key`, `RenderInputs.password_hash`, `RenderInputs.out`, `RenderInputs.hostname`, `RenderInputs.host_key`, `RenderInputs.bootstrap`, `RenderInputs.product_hostname`, `render` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 15–23 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs; declarations/fields: `RenderInputs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 16 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.operator_key; declarations/fields: `RenderInputs.operator_key` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 17 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.password_hash; declarations/fields: `RenderInputs.password_hash` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 18 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.out; declarations/fields: `RenderInputs.out` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 19 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.hostname; declarations/fields: `RenderInputs.hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 20 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.host_key; declarations/fields: `RenderInputs.host_key` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 21 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.bootstrap; declarations/fields: `RenderInputs.bootstrap` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 22 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.product_hostname; declarations/fields: `RenderInputs.product_hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 27–107 | Public bootstrap plus private per-instance operator input renderer; declaration/member render; declarations/fields: `render` |

## [tools/release-assets/src/render/provisioning/tests.rs](../../../../../tools/release-assets/src/render/provisioning/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–3 | Public bootstrap plus private per-instance operator input renderer; declarations/fields: `hostnames_follow_the_script_regexes`, `dump_keeps_number_literals_and_key_order`, `dump_matches_json_indent_two` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 5–28 | Source assertion of Authenticated installation media; declarations/fields: `hostnames_follow_the_script_regexes` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 30–40 | Public bootstrap plus private per-instance operator input renderer; declaration/member dump_matches_json_indent_two; declarations/fields: `dump_matches_json_indent_two` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 42–49 | Public bootstrap plus private per-instance operator input renderer; declaration/member dump_keeps_number_literals_and_key_order; declarations/fields: `dump_keeps_number_literals_and_key_order` |

## [tools/release-assets/src/render/stage/mod.rs](../../../../../tools/release-assets/src/render/stage/mod.rs)

Re-audit @HEAD: C09 consolidation + split: `stage.rs` is `stage/` with 5 files; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–10 | Existing native build outputs staging admission/errors; declarations/fields: `tests`, `StageError`, `refusal`, `failure`, `check_contexts`, `run` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 12–23 | Staging source assertions; declarations/fields: `tests` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 28–31 | Existing native build outputs staging admission/errors; declaration/member StageError; declarations/fields: `StageError` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 34–36 | Existing native build outputs staging admission/errors; declaration/member refusal; declarations/fields: `refusal` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 38–41 | Existing native build outputs staging admission/errors; declaration/member failure; declarations/fields: `failure` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 46–63 | Bound existing source/build payload roots and staging operation; declaration/member check_contexts; declarations/fields: `check_contexts` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 66–261 | Bound existing source/build payload roots and staging operation; declaration/member run; declarations/fields: `run` |

## [tools/release-assets/src/render/stage/branding.rs](../../../../../tools/release-assets/src/render/stage/branding.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1 | Existing native build outputs staging admission/errors; declarations/fields: `favicon_bytes` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 3–24 | Native favicon byte container rendering; declarations/fields: `favicon_bytes` |

## [tools/release-assets/src/render/stage/files.rs](../../../../../tools/release-assets/src/render/stage/files.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–7 | Existing native build outputs staging admission/errors; declarations/fields: `check_platform`, `is_real_dir`, `copy`, `copy_tree`, `normalize_tree`, `mkdir_leaf`, `mkdir_fresh`, `read_text` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 12–17 | Existing native build outputs staging admission/errors; declaration/member check_platform; declarations/fields: `check_platform` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 19–27 | Fresh-directory protected copying and permissions; declarations/fields: `is_real_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 29–64 | Fresh-directory protected copying and permissions; declaration/member copy; declarations/fields: `copy` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 69–95 | Fresh-directory protected copying and permissions; declaration/member copy_tree; declarations/fields: `copy_tree` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 100–117 | Fresh-directory protected copying and permissions; declaration/member normalize_tree; declarations/fields: `normalize_tree` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 121–124 | Fresh-directory protected copying and permissions; declaration/member mkdir_leaf; declarations/fields: `mkdir_leaf` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 128–136 | Fresh-directory protected copying and permissions; declaration/member mkdir_fresh; declarations/fields: `mkdir_fresh` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 138–141 | Fresh-directory protected copying and permissions; declaration/member read_text; declarations/fields: `read_text` |

## [tools/release-assets/src/render/stage/payload.rs](../../../../../tools/release-assets/src/render/stage/payload.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–9 | Existing native build outputs staging admission/errors; declarations/fields: `payload_entries`, `locked_terminal_assets`, `payload_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 11–37 | Declared native payload source mapping; declarations/fields: `payload_entries` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 39–78 | Locked browser terminal asset integrity pin admission; declarations/fields: `locked_terminal_assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 80–85 | Bound existing source/build payload roots and staging operation; declarations/fields: `payload_source` |

## [tools/release-assets/src/render/stage/tests.rs](../../../../../tools/release-assets/src/render/stage/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1 | Existing native build outputs staging admission/errors; declarations/fields: `favicon_container_matches_struct_layout`, `payload_source_roots_build_tokens` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 3–21 | Source assertion of Branding, avatars and attribution; declarations/fields: `favicon_container_matches_struct_layout` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 23–35 | Source assertion of Candidate production; declarations/fields: `payload_source_roots_build_tokens` |

## [tools/release-assets/src/render/terminal_logo/mod.rs](../../../../../tools/release-assets/src/render/terminal_logo/mod.rs)

Re-audit @HEAD: C09 consolidation + split: `terminal_logo.rs` is `terminal_logo/` with 4 files; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1–12 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declarations/fields: `tests`, `render_svg`, `run` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 14–23 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tests; declarations/fields: `tests` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 27–56 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member render_svg; declarations/fields: `render_svg` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 59–79 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member run; declarations/fields: `run` |

## [tools/release-assets/src/render/terminal_logo/geometry.rs](../../../../../tools/release-assets/src/render/terminal_logo/geometry.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declarations/fields: `Token`, `tokenize`, `operand`, `polygons`, `inside`, `render_layers` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 3–7 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Token; declarations/fields: `Token` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 11–42 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tokenize; declarations/fields: `tokenize` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 44–51 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member operand; declarations/fields: `operand` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 55–109 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member polygons; declarations/fields: `polygons` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 111–120 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member inside; declarations/fields: `inside` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 122–164 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member render_layers; declarations/fields: `render_layers` |

## [tools/release-assets/src/render/terminal_logo/svg.rs](../../../../../tools/release-assets/src/render/terminal_logo/svg.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declarations/fields: `Attrs`, `Cursor`, `Cursor.bytes`, `Cursor.pos`, `new`, `eof`, `starts_with`, `consume`, `skip_ws`, `skip_comment`, `skip_pi`, `skip_prolog`, `parse_name`, `decode_entities`, `parse_attrs`, `skip_text_element`, `tag_boundary`, `parse_svg`, `attr` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 3 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Attrs; declarations/fields: `Attrs` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 5–8 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Cursor; declarations/fields: `Cursor` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 6 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Cursor.bytes; declarations/fields: `Cursor.bytes` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 7 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Cursor.pos; declarations/fields: `Cursor.pos` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 11–16 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member new; declarations/fields: `new` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 18–20 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member eof; declarations/fields: `eof` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 22–24 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member starts_with; declarations/fields: `starts_with` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 26–33 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member consume; declarations/fields: `consume` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 35–39 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_ws; declarations/fields: `skip_ws` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 41–51 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_comment; declarations/fields: `skip_comment` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 53–63 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_pi; declarations/fields: `skip_pi` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 65–76 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_prolog; declarations/fields: `skip_prolog` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 78–94 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member parse_name; declarations/fields: `parse_name` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 96–130 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member decode_entities; declarations/fields: `decode_entities` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 133–174 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member parse_attrs; declarations/fields: `parse_attrs` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 178–198 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_text_element; declarations/fields: `skip_text_element` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 200–206 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tag_boundary; declarations/fields: `tag_boundary` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 210–280 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member parse_svg; declarations/fields: `parse_svg` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 282–287 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member attr; declarations/fields: `attr` |

## [tools/release-assets/src/render/terminal_logo/tests.rs](../../../../../tools/release-assets/src/render/terminal_logo/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declarations/fields: `document`, `tokenizer_matches_findall_shapes`, `polygon_gates_match_the_script`, `svg_gate_rejects_changed_syntax`, `tiny_emblem_renders_both_layers` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 3–7 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member document; declarations/fields: `document` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 9–27 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tokenizer_matches_findall_shapes; declarations/fields: `tokenizer_matches_findall_shapes` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 29–59 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member polygon_gates_match_the_script; declarations/fields: `polygon_gates_match_the_script` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 61–90 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member svg_gate_rejects_changed_syntax; declarations/fields: `svg_gate_rejects_changed_syntax` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 92–108 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tiny_emblem_renders_both_layers; declarations/fields: `tiny_emblem_renders_both_layers` |

## [tools/release-assets/tests/render_staging.rs](../../../../../tools/release-assets/tests/render_staging.rs)

Re-audit @HEAD: C09 consolidation: `soda-stage-render/tests/cli.rs` split into `render_staging.rs` + `render_provisioning.rs` + `render_terminal_logo.rs` + `render_support`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–10 | Synthetic-root native staging CLI fixtures and byte assertions; declarations/fields: `stage_help_and_usage`, `stage_refuses_bad_contexts_before_touching_the_tree`, `stage_reports_a_missing_checkout_root` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 14–72 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_help_and_usage; declarations/fields: `stage_help_and_usage` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 74–154 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_refuses_bad_contexts_before_touching_the_tree; declarations/fields: `stage_refuses_bad_contexts_before_touching_the_tree` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 156–176 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_reports_a_missing_checkout_root; declarations/fields: `stage_reports_a_missing_checkout_root` |

## [tools/release-assets/tests/render_provisioning.rs](../../../../../tools/release-assets/tests/render_provisioning.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–16 | Synthetic-root native staging CLI fixtures and byte assertions; declarations/fields: `provisioning_inputs`, `fixture_root`, `provisioning_help_and_usage`, `provisioning_matches_the_baked_goldens`, `provisioning_host_key_never_enters_argv`, `provisioning_rejections_match_the_script` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 20–24 | Private-input provisioning render fixture setup/assertions; declarations/fields: `provisioning_inputs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 26–30 | Private-input provisioning render fixture setup/assertions; declaration/member fixture_root; declarations/fields: `fixture_root` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 32–114 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_help_and_usage; declarations/fields: `provisioning_help_and_usage` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 116–160 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_matches_the_baked_goldens; declarations/fields: `provisioning_matches_the_baked_goldens` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 162–215 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_host_key_never_enters_argv; declarations/fields: `provisioning_host_key_never_enters_argv` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 217–402 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_rejections_match_the_script; declarations/fields: `provisioning_rejections_match_the_script` |

## [tools/release-assets/tests/render_terminal_logo.rs](../../../../../tools/release-assets/tests/render_terminal_logo.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–11 | Synthetic-root native staging CLI fixtures and byte assertions; declarations/fields: `logo_root`, `logo_help_and_usage`, `logo_renders_the_canonical_emblem_byte_for_byte`, `logo_gates_match_the_script` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 15–31 | Canonical emblem byte rendering and shape gate source tests; declarations/fields: `logo_root` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 33–47 | Canonical emblem byte rendering and shape gate source tests; declaration/member logo_help_and_usage; declarations/fields: `logo_help_and_usage` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 49–70 | Canonical emblem byte rendering and shape gate source tests; declaration/member logo_renders_the_canonical_emblem_byte_for_byte; declarations/fields: `logo_renders_the_canonical_emblem_byte_for_byte` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 72–109 | Canonical emblem byte rendering and shape gate source tests; declaration/member logo_gates_match_the_script; declarations/fields: `logo_gates_match_the_script` |

## [tools/release-assets/tests/render_support/mod.rs](../../../../../tools/release-assets/tests/render_support/mod.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–8 | Synthetic-root native staging CLI fixtures and byte assertions; declarations/fields: `COUNTER`, `TempDir`, `TempDir.path`, `new`, `sub`, `file`, `drop`, `fixtures`, `repo_root`, `copy_dir`, `stage_bin`, `provisioning_bin`, `logo_bin`, `run`, `run_env`, `mode_of` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 10 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member COUNTER; declarations/fields: `COUNTER` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 12–14 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member TempDir; declarations/fields: `TempDir` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 13 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member TempDir.path; declarations/fields: `TempDir.path` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 17–27 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member new; declarations/fields: `new` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 29–34 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member sub; declarations/fields: `sub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 36–42 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member file; declarations/fields: `file` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 45–48 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member drop; declarations/fields: `drop` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 50–52 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member fixtures; declarations/fields: `fixtures` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 54–60 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member repo_root; declarations/fields: `repo_root` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 62–74 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member copy_dir; declarations/fields: `copy_dir` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 76–78 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_bin; declarations/fields: `stage_bin` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 80–82 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member provisioning_bin; declarations/fields: `provisioning_bin` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 84–86 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member logo_bin; declarations/fields: `logo_bin` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 88–90 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member run; declarations/fields: `run` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 92–109 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member run_env; declarations/fields: `run_env` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 111–113 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member mode_of; declarations/fields: `mode_of` |

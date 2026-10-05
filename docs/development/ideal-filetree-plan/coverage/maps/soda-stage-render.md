# Soda stage render

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-d04d16b4f1e0"></a>

## [rust/soda-stage-render/src/lib.rs](../../../../../rust/soda-stage-render/src/lib.rs)

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

## [rust/soda-stage-render/src/provisioning.rs](../../../../../rust/soda-stage-render/src/provisioning.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–17, 595–606 | Public bootstrap plus private per-instance operator input renderer; declarations/fields: `ProvKind`, `name`, `ProvError`, `new`, `value`, `io`, `read_text`, `regular`, `is_label`, `is_appliance_hostname`, `is_fixture_hostname`, `escape_python_into`, `indent_into`, `dump_python`, `emit_python`, `object_mut`, `get_mut`, `files_mut`, `not_a_list`, `push_file`, `clear_files`, `str_value`, `file_entry`, `public_config`, `absolute_lexical`, `derive_host_public`, `is_real_parent`, `write_exclusive`, `RenderInputs`, `render`, `tests`, `dump_matches_json_indent_two` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 18–32 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvKind; declarations/fields: `ProvKind` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 33–50 | Public bootstrap plus private per-instance operator input renderer; declaration/member name; declarations/fields: `name` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 51 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvError; declarations/fields: `ProvError` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 52 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvError.kind; declarations/fields: `ProvError.kind` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 53–56 | Public bootstrap plus private per-instance operator input renderer; declaration/member ProvError.detail; declarations/fields: `ProvError.detail` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 57–63 | Public bootstrap plus private per-instance operator input renderer; declaration/member new; declarations/fields: `new` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 64–67 | Public bootstrap plus private per-instance operator input renderer; declaration/member value; declarations/fields: `value` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 68–78 | Public bootstrap plus private per-instance operator input renderer; declaration/member io; declarations/fields: `io` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 79–91 | Public bootstrap plus private per-instance operator input renderer; declaration/member read_text; declarations/fields: `read_text` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 92–109 | Public bootstrap plus private per-instance operator input renderer; declaration/member regular; declarations/fields: `regular` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 110–128 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_label; declarations/fields: `is_label` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 129–134 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_appliance_hostname; declarations/fields: `is_appliance_hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 135–145 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_fixture_hostname; declarations/fields: `is_fixture_hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 146–172 | Public bootstrap plus private per-instance operator input renderer; declaration/member escape_python_into; declarations/fields: `escape_python_into` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 173–182 | Public bootstrap plus private per-instance operator input renderer; declaration/member indent_into; declarations/fields: `indent_into` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 183–188 | Public bootstrap plus private per-instance operator input renderer; declaration/member dump_python; declarations/fields: `dump_python` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 189–234 | Public bootstrap plus private per-instance operator input renderer; declaration/member emit_python; declarations/fields: `emit_python` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 235–241 | Public bootstrap plus private per-instance operator input renderer; declaration/member object_mut; declarations/fields: `object_mut` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 242–252 | Public bootstrap plus private per-instance operator input renderer; declaration/member get_mut; declarations/fields: `get_mut` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 253–271 | Public bootstrap plus private per-instance operator input renderer; declaration/member files_mut; declarations/fields: `files_mut` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 272–275 | Public bootstrap plus private per-instance operator input renderer; declaration/member not_a_list; declarations/fields: `not_a_list` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 276–285 | Public bootstrap plus private per-instance operator input renderer; declaration/member push_file; declarations/fields: `push_file` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 286–295 | Public bootstrap plus private per-instance operator input renderer; declaration/member clear_files; declarations/fields: `clear_files` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 296–299 | Public bootstrap plus private per-instance operator input renderer; declaration/member str_value; declarations/fields: `str_value` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 300–312 | Public bootstrap plus private per-instance operator input renderer; declaration/member file_entry; declarations/fields: `file_entry` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 313–335 | Public bootstrap plus private per-instance operator input renderer; declaration/member public_config; declarations/fields: `public_config` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 336–346 | Public bootstrap plus private per-instance operator input renderer; declaration/member absolute_lexical; declarations/fields: `absolute_lexical` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 347–409 | Public bootstrap plus private per-instance operator input renderer; declaration/member derive_host_public; declarations/fields: `derive_host_public` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 410–423 | Public bootstrap plus private per-instance operator input renderer; declaration/member is_real_parent; declarations/fields: `is_real_parent` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 424–470 | Public bootstrap plus private per-instance operator input renderer; declaration/member write_exclusive; declarations/fields: `write_exclusive` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 471 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs; declarations/fields: `RenderInputs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 472 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.operator_key; declarations/fields: `RenderInputs.operator_key` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 473 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.password_hash; declarations/fields: `RenderInputs.password_hash` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 474 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.out; declarations/fields: `RenderInputs.out` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 475 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.hostname; declarations/fields: `RenderInputs.hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 476 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.host_key; declarations/fields: `RenderInputs.host_key` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 477 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.bootstrap; declarations/fields: `RenderInputs.bootstrap` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 478–482 | Public bootstrap plus private per-instance operator input renderer; declaration/member RenderInputs.product_hostname; declarations/fields: `RenderInputs.product_hostname` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 483–565 | Public bootstrap plus private per-instance operator input renderer; declaration/member render; declarations/fields: `render` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 566–569 | Public bootstrap plus private per-instance operator input renderer; declaration/member tests; declarations/fields: `tests` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 570–594 | Source assertion of Authenticated installation media; declarations/fields: `hostnames_follow_the_script_regexes` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 607–614 | Public bootstrap plus private per-instance operator input renderer; declaration/member dump_keeps_number_literals_and_key_order; declarations/fields: `dump_keeps_number_literals_and_key_order` |

<a id="coverage-982645f89792"></a>

<a id="rustsoda-stage-rendersrcstagers-1"></a>

## [rust/soda-stage-render/src/stage.rs](../../../../../rust/soda-stage-render/src/stage.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–17 | Existing native build outputs staging admission/errors; declarations/fields: `StageError`, `refusal`, `failure`, `check_platform` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 18–23 | Existing native build outputs staging admission/errors; declaration/member StageError; declarations/fields: `StageError` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 24–27 | Existing native build outputs staging admission/errors; declaration/member refusal; declarations/fields: `refusal` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 28–35 | Existing native build outputs staging admission/errors; declaration/member failure; declarations/fields: `failure` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 36–42 | Existing native build outputs staging admission/errors; declaration/member check_platform; declarations/fields: `check_platform` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 43–52 | Fresh-directory protected copying and permissions; declarations/fields: `is_real_dir` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 53–87 | Fresh-directory protected copying and permissions; declaration/member copy; declarations/fields: `copy` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 88–118 | Fresh-directory protected copying and permissions; declaration/member copy_tree; declarations/fields: `copy_tree` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 119–139 | Fresh-directory protected copying and permissions; declaration/member normalize_tree; declarations/fields: `normalize_tree` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 140–146 | Fresh-directory protected copying and permissions; declaration/member mkdir_leaf; declarations/fields: `mkdir_leaf` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 147–156 | Fresh-directory protected copying and permissions; declaration/member mkdir_fresh; declarations/fields: `mkdir_fresh` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 157–161 | Fresh-directory protected copying and permissions; declaration/member read_text; declarations/fields: `read_text` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 162–189 | Declared native payload source mapping; declarations/fields: `payload_entries` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 190–230 | Locked browser terminal asset integrity pin admission; declarations/fields: `locked_terminal_assets` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 231–253 | Native favicon byte container rendering; declarations/fields: `favicon_bytes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 254–263 | Bound existing source/build payload roots and staging operation; declarations/fields: `payload_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 264–283 | Bound existing source/build payload roots and staging operation; declaration/member check_contexts; declarations/fields: `check_contexts` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 284–481 | Bound existing source/build payload roots and staging operation; declaration/member run; declarations/fields: `run` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 482–485 | Staging source assertions; declarations/fields: `tests` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 486–505 | Source assertion of Branding, avatars and attribution; declarations/fields: `favicon_container_matches_struct_layout` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 506–518 | Source assertion of Candidate production; declarations/fields: `payload_source_roots_build_tokens` |

<a id="coverage-1d7b5e90f301"></a>

<a id="rustsoda-stage-rendersrcterminal_logors-1"></a>

## [rust/soda-stage-render/src/terminal_logo.rs](../../../../../rust/soda-stage-render/src/terminal_logo.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1–13 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declarations/fields: `Token`, `tokenize`, `operand`, `polygons`, `inside`, `render_layers`, `Cursor`, `new`, `eof`, `starts_with`, `consume`, `skip_ws`, `skip_comment`, `skip_pi`, `skip_prolog`, `parse_name`, `decode_entities`, `parse_attrs`, `skip_text_element`, `tag_boundary`, `parse_svg`, `attr`, `render_svg`, `run`, `tests`, `document`, `tokenizer_matches_findall_shapes`, `polygon_gates_match_the_script`, `svg_gate_rejects_changed_syntax`, `tiny_emblem_renders_both_layers` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 14–20 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Token; declarations/fields: `Token` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 21–53 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tokenize; declarations/fields: `tokenize` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 54–64 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member operand; declarations/fields: `operand` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 65–120 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member polygons; declarations/fields: `polygons` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 121–131 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member inside; declarations/fields: `inside` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 132–175 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member render_layers; declarations/fields: `render_layers` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 176–177 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Attrs; declarations/fields: `Attrs` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 178 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Cursor; declarations/fields: `Cursor` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 179 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Cursor.bytes; declarations/fields: `Cursor.bytes` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 180–183 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member Cursor.pos; declarations/fields: `Cursor.pos` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 184–190 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member new; declarations/fields: `new` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 191–194 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member eof; declarations/fields: `eof` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 195–198 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member starts_with; declarations/fields: `starts_with` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 199–207 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member consume; declarations/fields: `consume` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 208–213 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_ws; declarations/fields: `skip_ws` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 214–225 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_comment; declarations/fields: `skip_comment` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 226–237 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_pi; declarations/fields: `skip_pi` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 238–250 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_prolog; declarations/fields: `skip_prolog` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 251–268 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member parse_name; declarations/fields: `parse_name` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 269–305 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member decode_entities; declarations/fields: `decode_entities` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 306–350 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member parse_attrs; declarations/fields: `parse_attrs` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 351–372 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member skip_text_element; declarations/fields: `skip_text_element` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 373–382 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tag_boundary; declarations/fields: `tag_boundary` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 383–454 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member parse_svg; declarations/fields: `parse_svg` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 455–463 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member attr; declarations/fields: `attr` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 464–495 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member render_svg; declarations/fields: `render_svg` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 496–518 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member run; declarations/fields: `run` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 519–521 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tests; declarations/fields: `tests` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 522–528 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member document; declarations/fields: `document` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 529–548 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tokenizer_matches_findall_shapes; declarations/fields: `tokenizer_matches_findall_shapes` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 549–580 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member polygon_gates_match_the_script; declarations/fields: `polygon_gates_match_the_script` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 581–613 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member svg_gate_rejects_changed_syntax; declarations/fields: `svg_gate_rejects_changed_syntax` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 614–630 | Canonical polygon emblem to terminal mark rendering and strict source shape gate; declaration/member tiny_emblem_renders_both_layers; declarations/fields: `tiny_emblem_renders_both_layers` |

<a id="coverage-a740ec3e9a23"></a>

<a id="rustsoda-stage-rendertestsclirs-1"></a>

## [rust/soda-stage-render/tests/cli.rs](../../../../../rust/soda-stage-render/tests/cli.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–19 | Synthetic-root native staging CLI fixtures and byte assertions; declarations/fields: `COUNTER`, `TempDir`, `new`, `sub`, `file`, `drop`, `fixtures`, `repo_root`, `copy_dir`, `stage_bin`, `provisioning_bin`, `logo_bin`, `run`, `run_env`, `mode_of`, `stage_help_and_usage`, `stage_refuses_bad_contexts_before_touching_the_tree`, `stage_reports_a_missing_checkout_root` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 20–21 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member COUNTER; declarations/fields: `COUNTER` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 22 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member TempDir; declarations/fields: `TempDir` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 23–26 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member TempDir.path; declarations/fields: `TempDir.path` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 27–38 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member new; declarations/fields: `new` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 39–45 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member sub; declarations/fields: `sub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 46–54 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member file; declarations/fields: `file` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 55–59 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member drop; declarations/fields: `drop` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 60–63 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member fixtures; declarations/fields: `fixtures` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 64–71 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member repo_root; declarations/fields: `repo_root` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 72–85 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member copy_dir; declarations/fields: `copy_dir` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 86–89 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_bin; declarations/fields: `stage_bin` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 90–93 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member provisioning_bin; declarations/fields: `provisioning_bin` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 94–97 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member logo_bin; declarations/fields: `logo_bin` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 98–101 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member run; declarations/fields: `run` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 102–115 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member run_env; declarations/fields: `run_env` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 116–122 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member mode_of; declarations/fields: `mode_of` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 123–182 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_help_and_usage; declarations/fields: `stage_help_and_usage` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 183–264 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_refuses_bad_contexts_before_touching_the_tree; declarations/fields: `stage_refuses_bad_contexts_before_touching_the_tree` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 265–287 | Synthetic-root native staging CLI fixtures and byte assertions; declaration/member stage_reports_a_missing_checkout_root; declarations/fields: `stage_reports_a_missing_checkout_root` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 288–293 | Private-input provisioning render fixture setup/assertions; declarations/fields: `provisioning_inputs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 294–300 | Private-input provisioning render fixture setup/assertions; declaration/member fixture_root; declarations/fields: `fixture_root` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 301–384 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_help_and_usage; declarations/fields: `provisioning_help_and_usage` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 385–430 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_matches_the_baked_goldens; declarations/fields: `provisioning_matches_the_baked_goldens` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 431–485 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_host_key_never_enters_argv; declarations/fields: `provisioning_host_key_never_enters_argv` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 486–673 | Private-input provisioning render fixture setup/assertions; declaration/member provisioning_rejections_match_the_script; declarations/fields: `provisioning_rejections_match_the_script` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 674–692 | Canonical emblem byte rendering and shape gate source tests; declarations/fields: `logo_root` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 693–708 | Canonical emblem byte rendering and shape gate source tests; declaration/member logo_help_and_usage; declarations/fields: `logo_help_and_usage` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 709–731 | Canonical emblem byte rendering and shape gate source tests; declaration/member logo_renders_the_canonical_emblem_byte_for_byte; declarations/fields: `logo_renders_the_canonical_emblem_byte_for_byte` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 732–768 | Canonical emblem byte rendering and shape gate source tests; declaration/member logo_gates_match_the_script; declarations/fields: `logo_gates_match_the_script` |


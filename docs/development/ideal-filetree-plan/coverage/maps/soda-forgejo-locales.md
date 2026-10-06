# Soda forgejo locales

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: merge test extraction + locales_cli 4-way split re-mapped (7 files). All rows machine-verified against current bytes.

<a id="coverage-09c1f2dd0505"></a>

## [tools/release-assets/src/locales/mod.rs](../../../../../tools/release-assets/src/locales/mod.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; no drift.

Public module wiring is mapped separately; module exposure does not create a new process boundary.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–12, 18–37 | Native locale module/tool wiring; declarations/fields: `split_flag` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 13–17 | Module wiring: locales; Native Forgejo presentation; declarations/fields: `locales` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 38–47 | Locale asset digest helper; declarations/fields: `sha256_hex` |

<a id="coverage-621f066f7532"></a>

<a id="rustsoda-forgejo-localessrclocalesrs-1"></a>

## [tools/release-assets/src/locales/merge.rs](../../../../../tools/release-assets/src/locales/merge.rs)

Re-audit @HEAD: C09 consolidation: inline tests extracted to `locales/merge/tests.rs`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–16 | Native locale parsing, merge and Soda namespace preservation; declarations/fields: `SOURCE_PREFIX`, `MAX_NATIVE`, `FETCH_TIMEOUT`, `DEFAULT_ADDITIONS`, `Error`, `usage`, `runtime`, `message`, `exit_code`, `python_space`, `strip`, `rstrip`, `indent_of`, `Ini`, `Ini.sections`, `Ini.options`, `has_section`, `sections`, `parse_ini`, `section_header`, `merge`, `read_capped`, `load_lock`, `fetch_locked`, `read_native_file`, `Native`, `run`, `tests` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 19 | Native locale parsing, merge and Soda namespace preservation; declaration/member SOURCE_PREFIX; declarations/fields: `SOURCE_PREFIX` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 21 | Native locale parsing, merge and Soda namespace preservation; declaration/member MAX_NATIVE; declarations/fields: `MAX_NATIVE` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 23 | Native locale parsing, merge and Soda namespace preservation; declaration/member FETCH_TIMEOUT; declarations/fields: `FETCH_TIMEOUT` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 25 | Native locale parsing, merge and Soda namespace preservation; declaration/member DEFAULT_ADDITIONS; declarations/fields: `DEFAULT_ADDITIONS` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 29–33 | Native locale parsing, merge and Soda namespace preservation; declaration/member Error; declarations/fields: `Error` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 36–38 | Native locale parsing, merge and Soda namespace preservation; declaration/member usage; declarations/fields: `usage` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 40–42 | Native locale parsing, merge and Soda namespace preservation; declaration/member runtime; declarations/fields: `runtime` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 44–48 | Native locale parsing, merge and Soda namespace preservation; declaration/member message; declarations/fields: `message` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 50–56 | Native locale parsing, merge and Soda namespace preservation; declaration/member exit_code; declarations/fields: `exit_code` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 62–78 | Native locale parsing, merge and Soda namespace preservation; declaration/member python_space; declarations/fields: `python_space` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 80–82 | Native locale parsing, merge and Soda namespace preservation; declaration/member strip; declarations/fields: `strip` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 84–86 | Native locale parsing, merge and Soda namespace preservation; declaration/member rstrip; declarations/fields: `rstrip` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 88–90 | Native locale parsing, merge and Soda namespace preservation; declaration/member indent_of; declarations/fields: `indent_of` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 93–96 | Native locale parsing, merge and Soda namespace preservation; declaration/member Ini; declarations/fields: `Ini` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 94 | Native locale parsing, merge and Soda namespace preservation; declaration/member Ini.sections; declarations/fields: `Ini.sections` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 95 | Native locale parsing, merge and Soda namespace preservation; declaration/member Ini.options; declarations/fields: `Ini.options` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 99–101 | Native locale parsing, merge and Soda namespace preservation; declaration/member has_section; declarations/fields: `has_section` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 103–110 | Native locale parsing, merge and Soda namespace preservation; declaration/member sections; declarations/fields: `sections` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 115–168 | Native locale parsing, merge and Soda namespace preservation; declaration/member parse_ini; declarations/fields: `parse_ini` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 173–182 | Native locale parsing, merge and Soda namespace preservation; declaration/member section_header; declarations/fields: `section_header` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 186–200 | Native locale parsing, merge and Soda namespace preservation; declaration/member merge; declarations/fields: `merge` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 204–211 | Native locale parsing, merge and Soda namespace preservation; declaration/member read_capped; declarations/fields: `read_capped` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 214–231 | Native locale parsing, merge and Soda namespace preservation; declaration/member load_lock; declarations/fields: `load_lock` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 235–262 | Pinned upstream locale input acquisition; declarations/fields: `fetch_locked` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 264–275 | Pinned upstream locale input acquisition; declaration/member read_native_file; declarations/fields: `read_native_file` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 278–281 | Pinned upstream locale input acquisition; declaration/member Native; declarations/fields: `Native` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 285–318 | Locale merge/materialization CLI implementation; declarations/fields: `run` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 320–321 | Locale merge/materialization CLI implementation; declaration/member tests; declarations/fields: `tests` |

## [tools/release-assets/src/locales/merge/tests.rs](../../../../../tools/release-assets/src/locales/merge/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1 | Native locale parsing, merge and Soda namespace preservation; declarations/fields: `NATIVE`, `EXTRA`, `sections_of`, `strip_set_matches_python_over_all_chars`, `merge_is_byte_exact`, `merge_rstrips_python_whitespace`, `merge_rejects_incomplete_native`, `merge_rejects_non_soda_additions`, `merge_rejects_native_soda_collision`, `merge_rejects_duplicate_addition_keys`, `trailing_text_after_section_is_ignored`, `section_header_matches_greedily_to_last_bracket`, `indented_section_after_value_is_a_continuation`, `colon_is_not_a_delimiter`, `bare_words_are_rejected`, `strict_duplicates_are_rejected`, `option_names_are_case_sensitive`, `options_before_any_section_are_rejected`, `comments_and_blank_lines_end_continuations`, `option_name_stops_at_first_equals_and_rstrips`, `empty_option_name_is_rejected`, `colons_are_allowed_inside_option_names`, `default_section_stays_out_of_sections`, `continuation_ignores_indent_changes_until_blank`, `values_may_hold_percent_comment_and_crlf_bytes`, `lone_carriage_returns_do_not_split_lines`, `same_option_in_two_sections_is_allowed`, `serve_once`, `fetch_case`, `locked_fetch_accepts_exact_bytes`, `locked_fetch_refuses_changed_bytes`, `locked_fetch_refuses_oversize_bodies`, `locked_fetch_accepts_exactly_one_mib`, `locked_fetch_refuses_error_status` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 3 | Locale merge/materialization CLI implementation; declaration/member NATIVE; declarations/fields: `NATIVE` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 4 | Locale merge/materialization CLI implementation; declaration/member EXTRA; declarations/fields: `EXTRA` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 6–13 | Locale merge/materialization CLI implementation; declaration/member sections_of; declarations/fields: `sections_of` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 15–35, 37–40, 42–50, 52–56, 58–62, 64–69, 71–75, 77–80, 82–85, 87–91, 93–96, 98–101, 103–108, 110–114, 116–120, 122–126, 128–132, 134–138, 140–144, 146–151, 153–159, 161–165, 167–171, 173–177, 181–198, 200–205, 207–212, 214–219, 221–229, 231–237, 239–246 | Native locale source assertions; declarations/fields: `strip_set_matches_python_over_all_chars`, `merge_is_byte_exact`, `merge_rstrips_python_whitespace`, `merge_rejects_incomplete_native`, `merge_rejects_non_soda_additions`, `merge_rejects_native_soda_collision`, `merge_rejects_duplicate_addition_keys`, `trailing_text_after_section_is_ignored`, `section_header_matches_greedily_to_last_bracket`, `indented_section_after_value_is_a_continuation`, `colon_is_not_a_delimiter`, `bare_words_are_rejected`, `strict_duplicates_are_rejected`, `option_names_are_case_sensitive`, `options_before_any_section_are_rejected`, `comments_and_blank_lines_end_continuations`, `option_name_stops_at_first_equals_and_rstrips`, `empty_option_name_is_rejected`, `colons_are_allowed_inside_option_names`, `default_section_stays_out_of_sections`, `continuation_ignores_indent_changes_until_blank`, `values_may_hold_percent_comment_and_crlf_bytes`, `lone_carriage_returns_do_not_split_lines`, `same_option_in_two_sections_is_allowed`, `serve_once`, `fetch_case`, `locked_fetch_accepts_exact_bytes`, `locked_fetch_refuses_changed_bytes`, `locked_fetch_refuses_oversize_bodies`, `locked_fetch_accepts_exactly_one_mib`, `locked_fetch_refuses_error_status` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 15–35 | Native locale source assertions; declaration/member strip_set_matches_python_over_all_chars; declarations/fields: `strip_set_matches_python_over_all_chars` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 37–40 | Native locale source assertions; declaration/member merge_is_byte_exact; declarations/fields: `merge_is_byte_exact` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 42–50 | Native locale source assertions; declaration/member merge_rstrips_python_whitespace; declarations/fields: `merge_rstrips_python_whitespace` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 52–56 | Native locale source assertions; declaration/member merge_rejects_incomplete_native; declarations/fields: `merge_rejects_incomplete_native` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 58–62 | Native locale source assertions; declaration/member merge_rejects_non_soda_additions; declarations/fields: `merge_rejects_non_soda_additions` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 64–69 | Native locale source assertions; declaration/member merge_rejects_native_soda_collision; declarations/fields: `merge_rejects_native_soda_collision` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 71–75 | Native locale source assertions; declaration/member merge_rejects_duplicate_addition_keys; declarations/fields: `merge_rejects_duplicate_addition_keys` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 77–80 | Native locale source assertions; declaration/member trailing_text_after_section_is_ignored; declarations/fields: `trailing_text_after_section_is_ignored` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 82–85 | Native locale source assertions; declaration/member section_header_matches_greedily_to_last_bracket; declarations/fields: `section_header_matches_greedily_to_last_bracket` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 87–91 | Native locale source assertions; declaration/member indented_section_after_value_is_a_continuation; declarations/fields: `indented_section_after_value_is_a_continuation` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 93–96 | Native locale source assertions; declaration/member colon_is_not_a_delimiter; declarations/fields: `colon_is_not_a_delimiter` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 98–101 | Native locale source assertions; declaration/member bare_words_are_rejected; declarations/fields: `bare_words_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 103–108 | Native locale source assertions; declaration/member strict_duplicates_are_rejected; declarations/fields: `strict_duplicates_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 110–114 | Native locale source assertions; declaration/member option_names_are_case_sensitive; declarations/fields: `option_names_are_case_sensitive` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 116–120 | Native locale source assertions; declaration/member options_before_any_section_are_rejected; declarations/fields: `options_before_any_section_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 122–126 | Native locale source assertions; declaration/member comments_and_blank_lines_end_continuations; declarations/fields: `comments_and_blank_lines_end_continuations` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 128–132 | Native locale source assertions; declaration/member option_name_stops_at_first_equals_and_rstrips; declarations/fields: `option_name_stops_at_first_equals_and_rstrips` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 134–138 | Native locale source assertions; declaration/member empty_option_name_is_rejected; declarations/fields: `empty_option_name_is_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 140–144 | Native locale source assertions; declaration/member colons_are_allowed_inside_option_names; declarations/fields: `colons_are_allowed_inside_option_names` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 146–151 | Native locale source assertions; declaration/member default_section_stays_out_of_sections; declarations/fields: `default_section_stays_out_of_sections` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 153–159 | Native locale source assertions; declaration/member continuation_ignores_indent_changes_until_blank; declarations/fields: `continuation_ignores_indent_changes_until_blank` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 161–165 | Native locale source assertions; declaration/member values_may_hold_percent_comment_and_crlf_bytes; declarations/fields: `values_may_hold_percent_comment_and_crlf_bytes` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 167–171 | Native locale source assertions; declaration/member lone_carriage_returns_do_not_split_lines; declarations/fields: `lone_carriage_returns_do_not_split_lines` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 173–177 | Native locale source assertions; declaration/member same_option_in_two_sections_is_allowed; declarations/fields: `same_option_in_two_sections_is_allowed` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 181–198 | Native locale source assertions; declaration/member serve_once; declarations/fields: `serve_once` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 200–205 | Native locale source assertions; declaration/member fetch_case; declarations/fields: `fetch_case` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 207–212 | Native locale source assertions; declaration/member locked_fetch_accepts_exact_bytes; declarations/fields: `locked_fetch_accepts_exact_bytes` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 214–219 | Native locale source assertions; declaration/member locked_fetch_refuses_changed_bytes; declarations/fields: `locked_fetch_refuses_changed_bytes` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 221–229 | Native locale source assertions; declaration/member locked_fetch_refuses_oversize_bodies; declarations/fields: `locked_fetch_refuses_oversize_bodies` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 231–237 | Native locale source assertions; declaration/member locked_fetch_accepts_exactly_one_mib; declarations/fields: `locked_fetch_accepts_exactly_one_mib` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 239–246 | Native locale source assertions; declaration/member locked_fetch_refuses_error_status; declarations/fields: `locked_fetch_refuses_error_status` |

## [tools/release-assets/tests/locales_cli.rs](../../../../../tools/release-assets/tests/locales_cli.rs)

Re-audit @HEAD: C09 consolidation: `locales_cli.rs` split into focused CLI test files + `locales_support`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–8 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declarations/fields: `native_or_lock_is_required`, `native_and_lock_are_mutually_exclusive`, `out_is_required`, `stray_arguments_are_rejected`, `unknown_flags_are_rejected`, `help_exits_zero` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 10–20 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member native_or_lock_is_required; declarations/fields: `native_or_lock_is_required` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 22–32 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member native_and_lock_are_mutually_exclusive; declarations/fields: `native_and_lock_are_mutually_exclusive` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 34–44 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member out_is_required; declarations/fields: `out_is_required` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 46–53 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member stray_arguments_are_rejected; declarations/fields: `stray_arguments_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 55–65 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member unknown_flags_are_rejected; declarations/fields: `unknown_flags_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 67–77 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member help_exits_zero; declarations/fields: `help_exits_zero` |

## [tools/release-assets/tests/locales_locked_input.rs](../../../../../tools/release-assets/tests/locales_locked_input.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–8 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declarations/fields: `lock_with_unexpected_source_is_refused_without_fetch`, `lock_document_problems_fail_without_fetch` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 10–37 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member lock_with_unexpected_source_is_refused_without_fetch; declarations/fields: `lock_with_unexpected_source_is_refused_without_fetch` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 39–86 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member lock_document_problems_fail_without_fetch; declarations/fields: `lock_document_problems_fail_without_fetch` |

## [tools/release-assets/tests/locales_native_merge.rs](../../../../../tools/release-assets/tests/locales_native_merge.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–10 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declarations/fields: `native_merge_is_byte_exact_and_silent`, `native_merge_supports_equals_flags`, `default_additions_resolve_against_the_working_directory`, `native_over_one_mib_is_refused`, `incomplete_native_catalog_is_rejected`, `duplicate_addition_keys_are_rejected`, `non_soda_addition_namespaces_are_rejected`, `native_soda_collisions_are_rejected`, `existing_output_is_never_overwritten` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 12–41 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member native_merge_is_byte_exact_and_silent; declarations/fields: `native_merge_is_byte_exact_and_silent` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 43–67 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member native_merge_supports_equals_flags; declarations/fields: `native_merge_supports_equals_flags` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 69–94 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member default_additions_resolve_against_the_working_directory; declarations/fields: `default_additions_resolve_against_the_working_directory` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 96–122 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member native_over_one_mib_is_refused; declarations/fields: `native_over_one_mib_is_refused` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 124–148 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member incomplete_native_catalog_is_rejected; declarations/fields: `incomplete_native_catalog_is_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 150–174 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member duplicate_addition_keys_are_rejected; declarations/fields: `duplicate_addition_keys_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 176–200 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member non_soda_addition_namespaces_are_rejected; declarations/fields: `non_soda_addition_namespaces_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 202–229 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member native_soda_collisions_are_rejected; declarations/fields: `native_soda_collisions_are_rejected` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 231–255 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member existing_output_is_never_overwritten; declarations/fields: `existing_output_is_never_overwritten` |

## [tools/release-assets/tests/locales_support/mod.rs](../../../../../tools/release-assets/tests/locales_support/mod.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–8 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declarations/fields: `COUNTER`, `NATIVE`, `EXTRA`, `TempDir`, `TempDir.path`, `new`, `file`, `drop`, `bin`, `run`, `status_of` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 10 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member COUNTER; declarations/fields: `COUNTER` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 12 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member NATIVE; declarations/fields: `NATIVE` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 13 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member EXTRA; declarations/fields: `EXTRA` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 15–17 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member TempDir; declarations/fields: `TempDir` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 16 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member TempDir.path; declarations/fields: `TempDir.path` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 20–30 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member new; declarations/fields: `new` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 32–40 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member file; declarations/fields: `file` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 43–46 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member drop; declarations/fields: `drop` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 48–50 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member bin; declarations/fields: `bin` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 52–58 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member run; declarations/fields: `run` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 60–62 | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member status_of; declarations/fields: `status_of` |

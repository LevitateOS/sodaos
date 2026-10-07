# Soda forgejo locales

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-621f066f7532"></a>
<a id="rustsoda-forgejo-localessrclocalesrs-1"></a>
<a id="coverage-7e12a394da95"></a>

## [tools/release-assets/src/locales/merge.rs](../../../../../tools/release-assets/src/locales/merge.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–284, 335–371; lines 23–24: SOURCE_PREFIX and attached behavior; attached module comments and attributes; lines 25–26: MAX_NATIVE and attached behavior; lines 27–28: FETCH_TIMEOUT and attached behavior; lines 29–33: DEFAULT_ADDITIONS and attached behavior; lines 34–39: Error and attached behavior; lines 40–42: usage and attached behavior; lines 43–46: runtime and attached behavior; lines 47–52: message and attached behavior; lines 53–65: exit_code and attached behavior; lines 66–82: python_space and attached behavior; lines 83–86: strip and attached behavior; lines 87–90: rstrip and attached behavior; lines 91–96: indent_of and attached behavior; lines 97–102: Ini and attached behavior; lines 103–105: has_section and attached behavior; lines 106–118: sections and attached behavior; lines 119–176: parse_ini and attached behavior; lines 177–189: section_header and attached behavior; lines 190–207: merge and attached behavior; lines 208–217: read_capped and attached behavior; lines 218–241: load_lock and attached behavior; lines 242–248: struct LocaleLock; lines 249–252: fn deserialize; lines 253–254: struct LockVisitor; lines 255–255: type Value; lines 256–258: fn expecting; lines 259–284: fn visit_map; lines 335–363: fn run; lines 371–371: tests and attached behavior | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Native locale parsing, merge and Soda namespace preservation; declaration/member SOURCE_PREFIX Adjacent comments and attributes explain this same authored responsibility.; 29 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 285–334; fetch_locked pinned locale input acquisition; lines 313–327: read_native_file and attached behavior; lines 328–334: Native and attached behavior | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Fetches the exact locale archive named by the lock data and enforces the locked digest before locale merge consumes it.; Pinned upstream locale input acquisition; declaration/member read_native_file; Pinned upstream locale input acquisition; declaration/member Native — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b09a011a2b70"></a>

## [tools/release-assets/src/locales/merge/tests.rs](../../../../../tools/release-assets/src/locales/merge/tests.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–262; lines 4–17: fn locale_lock_raw_slots_take_the_last_string_and_ignore_large_unknown_numbers; attached module comments and attributes; lines 18–19: NATIVE and attached behavior; lines 20–20: EXTRA and attached behavior; lines 21–31: sections_of and attached behavior; lines 32–53: strip_set_matches_python_over_all_chars and attached behavior; lines 54–58: merge_is_byte_exact and attached behavior; lines 59–68: merge_rstrips_python_whitespace and attached behavior; lines 69–74: merge_rejects_incomplete_native and attached behavior; lines 75–80: merge_rejects_non_soda_additions and attached behavior; lines 81–87: merge_rejects_native_soda_collision and attached behavior; lines 88–93: merge_rejects_duplicate_addition_keys and attached behavior; lines 94–98: trailing_text_after_section_is_ignored and attached behavior; lines 99–103: section_header_matches_greedily_to_last_bracket and attached behavior; lines 104–109: indented_section_after_value_is_a_continuation and attached behavior; lines 110–114: colon_is_not_a_delimiter and attached behavior; lines 115–119: bare_words_are_rejected and attached behavior; lines 120–126: strict_duplicates_are_rejected and attached behavior; lines 127–132: option_names_are_case_sensitive and attached behavior; lines 133–138: options_before_any_section_are_rejected and attached behavior; lines 139–144: comments_and_blank_lines_end_continuations and attached behavior; lines 145–150: option_name_stops_at_first_equals_and_rstrips and attached behavior; lines 151–156: empty_option_name_is_rejected and attached behavior; lines 157–162: colons_are_allowed_inside_option_names and attached behavior; lines 163–169: default_section_stays_out_of_sections and attached behavior; lines 170–177: continuation_ignores_indent_changes_until_blank and attached behavior; lines 178–183: values_may_hold_percent_comment_and_crlf_bytes and attached behavior; lines 184–189: lone_carriage_returns_do_not_split_lines and attached behavior; lines 190–196: same_option_in_two_sections_is_allowed and attached behavior; lines 197–197: serve_once and attached behavior; lines 215–223: fetch_case and attached behavior; lines 224–230: locked_fetch_accepts_exact_bytes and attached behavior; lines 231–237: locked_fetch_refuses_changed_bytes and attached behavior; lines 238–247: locked_fetch_refuses_oversize_bodies and attached behavior; lines 248–255: locked_fetch_accepts_exactly_one_mib and attached behavior; lines 256–262: locked_fetch_refuses_error_status and attached behavior | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current fn locale_lock_raw_slots_take_the_last_string_and_ignore_large_unknown_numbers and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 35 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-09c1f2dd0505"></a>
<a id="coverage-da6059a88aa5"></a>

## [tools/release-assets/src/locales/mod.rs](../../../../../tools/release-assets/src/locales/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 39–47; lines 1–17: mod merge; attached module comments and attributes; lines 18–37: fn split_flag | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current mod merge and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; Native locale module/tool wiring — tools/release-assets/src/locales/mod.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |
| 38; lines 38–38: fn sha256_hex | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Locale asset digest helper — tools/release-assets/src/locales/mod.rs; current symbols and attached bodies inspected; current consumer is the module/pipeline named in source |

<a id="coverage-a1961c640aa6"></a>

## [tools/release-assets/tests/locales_cli.rs](../../../../../tools/release-assets/tests/locales_cli.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; lines 1–6: mod locales_support; attached module comments and attributes; lines 11–22: native_or_lock_is_required and attached behavior; lines 23–34: native_and_lock_are_mutually_exclusive and attached behavior; lines 35–46: out_is_required and attached behavior; lines 47–55: stray_arguments_are_rejected and attached behavior; lines 56–67: unknown_flags_are_rejected and attached behavior; lines 68–77: help_exits_zero and attached behavior | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current mod locales_support and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bf4c6aedfcc0"></a>

## [tools/release-assets/tests/locales_locked_input.rs](../../../../../tools/release-assets/tests/locales_locked_input.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–86; lines 1–6: mod locales_support; attached module comments and attributes; lines 11–39: lock_with_unexpected_source_is_refused_without_fetch and attached behavior; lines 40–86: lock_document_problems_fail_without_fetch and attached behavior | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current mod locales_support and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member lock_with_unexpected_source_is_refused_without_fetch; Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member lock_document_problems_fail_without_fetch — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-39a33e83e9e5"></a>

## [tools/release-assets/tests/locales_native_merge.rs](../../../../../tools/release-assets/tests/locales_native_merge.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–255; lines 1–6: mod locales_support; attached module comments and attributes; lines 13–43: native_merge_is_byte_exact_and_silent and attached behavior; lines 44–69: native_merge_supports_equals_flags and attached behavior; lines 70–96: default_additions_resolve_against_the_working_directory and attached behavior; lines 97–124: native_over_one_mib_is_refused and attached behavior; lines 125–150: incomplete_native_catalog_is_rejected and attached behavior; lines 151–176: duplicate_addition_keys_are_rejected and attached behavior; lines 177–202: non_soda_addition_namespaces_are_rejected and attached behavior; lines 203–231: native_soda_collisions_are_rejected and attached behavior; lines 232–255: existing_output_is_never_overwritten and attached behavior | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current mod locales_support and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a1adb33805c3"></a>

## [tools/release-assets/tests/locales_support/mod.rs](../../../../../tools/release-assets/tests/locales_support/mod.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; lines 9–10: COUNTER and attached behavior; attached module comments and attributes; lines 11–12: NATIVE and attached behavior; lines 13–13: EXTRA and attached behavior; lines 14–19: TempDir and attached behavior; lines 20–30: new and attached behavior; lines 31–40: file and attached behavior; lines 41–42: impl Drop; lines 43–46: drop and attached behavior; lines 47–50: bin and attached behavior; lines 51–58: run and attached behavior; lines 59–62: status_of and attached behavior | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Native Forgejo locale augmentation preserving Soda-owned message namespace; declaration/member COUNTER Adjacent comments and attributes explain this same authored responsibility.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

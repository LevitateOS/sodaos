# Soda asset fetchers

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: muse/tea test extractions re-mapped (4 files). All rows machine-verified against current bytes.

<a id="coverage-820aaa3fbcac"></a>

<a id="rustsoda-asset-fetcherssrcmusers-1"></a>

## [tools/release-assets/src/fetch/muse.rs](../../../../../tools/release-assets/src/fetch/muse.rs)

Re-audit @HEAD: C09 consolidation: inline tests extracted to `fetch/muse/tests.rs`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1–10 | Pinned Muse, Tea and terminal asset acquisition; declarations/fields: `DOWNLOAD_BASE`, `USER_AGENT`, `TIMEOUT`, `MANIFEST_LIMIT`, `TEMP_COUNTER`, `query_escape`, `download_url`, `manifest_string`, `unknown_field`, `load_release`, `current_digest`, `create_temp`, `stage`, `fetch_muse`, `tests` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 12 | Pinned Muse, Tea and terminal asset acquisition; declaration/member DOWNLOAD_BASE; declarations/fields: `DOWNLOAD_BASE` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 15 | Pinned Muse, Tea and terminal asset acquisition; declaration/member USER_AGENT; declarations/fields: `USER_AGENT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 16 | Pinned Muse, Tea and terminal asset acquisition; declaration/member TIMEOUT; declarations/fields: `TIMEOUT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 18 | Pinned Muse, Tea and terminal asset acquisition; declaration/member MANIFEST_LIMIT; declarations/fields: `MANIFEST_LIMIT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 20 | Pinned Muse, Tea and terminal asset acquisition; declaration/member TEMP_COUNTER; declarations/fields: `TEMP_COUNTER` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 24–36 | Pinned Muse, Tea and terminal asset acquisition; declaration/member query_escape; declarations/fields: `query_escape` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 40–46 | Pinned Muse, Tea and terminal asset acquisition; declaration/member download_url; declarations/fields: `download_url` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 50–56 | Pinned Muse, Tea and terminal asset acquisition; declaration/member manifest_string; declarations/fields: `manifest_string` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 58–65 | Pinned Muse, Tea and terminal asset acquisition; declaration/member unknown_field; declarations/fields: `unknown_field` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 68–118 | Pinned Muse, Tea and terminal asset acquisition; declaration/member load_release; declarations/fields: `load_release` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 122–129 | Pinned Muse, Tea and terminal asset acquisition; declaration/member current_digest; declarations/fields: `current_digest` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 131–147 | Pinned Muse, Tea and terminal asset acquisition; declaration/member create_temp; declarations/fields: `create_temp` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 151–199 | Pinned Muse, Tea and terminal asset acquisition; declaration/member stage; declarations/fields: `stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 201–221 | Pinned Muse, Tea and terminal asset acquisition; declaration/member fetch_muse; declarations/fields: `fetch_muse` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 223–224 | Pinned Muse, Tea and terminal asset acquisition; declaration/member tests; declarations/fields: `tests` |

## [tools/release-assets/src/fetch/muse/tests.rs](../../../../../tools/release-assets/src/fetch/muse/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1–3 | Pinned Muse, Tea and terminal asset acquisition; declarations/fields: `VERSION`, `FILE`, `manifest_text`, `write_manifest`, `Fixture`, `Fixture.server`, `Fixture.base`, `Fixture.manifest`, `Fixture.payload`, `Fixture.scratch`, `start`, `mode`, `query_escape_matches_go`, `tampered_bytes_are_rejected_before_publishing`, `verified_bytes_stage_executable_with_pinned_query`, `matching_destination_is_kept_and_chmodded_without_fetch`, `stale_destination_is_replaced`, `non_200_status_fails_the_download`, `relative_destination_is_refused_first`, `unknown_architecture_is_refused`, `invalid_pins_are_refused_without_fetch`, `unknown_manifest_fields_are_decode_errors`, `oversized_manifest_is_refused`, `transport_errors_carry_the_download_prefix`, `missing_manifest_is_an_error` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 5 | Pinned Muse, Tea and terminal asset acquisition; declaration/member VERSION; declarations/fields: `VERSION` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 6 | Pinned Muse, Tea and terminal asset acquisition; declaration/member FILE; declarations/fields: `FILE` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 8–14 | Pinned Muse, Tea and terminal asset acquisition; declaration/member manifest_text; declarations/fields: `manifest_text` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 16–20 | Pinned Muse, Tea and terminal asset acquisition; declaration/member write_manifest; declarations/fields: `write_manifest` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 22–28 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture; declarations/fields: `Fixture` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 23 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.server; declarations/fields: `Fixture.server` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 24 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.base; declarations/fields: `Fixture.base` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 25 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.manifest; declarations/fields: `Fixture.manifest` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 26 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.payload; declarations/fields: `Fixture.payload` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 27 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.scratch; declarations/fields: `Fixture.scratch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 33–49 | Pinned Muse, Tea and terminal asset acquisition; declaration/member start; declarations/fields: `start` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 51–55 | Pinned Muse, Tea and terminal asset acquisition; declaration/member mode; declarations/fields: `mode` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 57–68 | Pinned Muse, Tea and terminal asset acquisition; declaration/member query_escape_matches_go; declarations/fields: `query_escape_matches_go` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 70–85 | Pinned Muse, Tea and terminal asset acquisition; declaration/member tampered_bytes_are_rejected_before_publishing; declarations/fields: `tampered_bytes_are_rejected_before_publishing` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 87–102 | Pinned Muse, Tea and terminal asset acquisition; declaration/member verified_bytes_stage_executable_with_pinned_query; declarations/fields: `verified_bytes_stage_executable_with_pinned_query` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 104–115 | Pinned Muse, Tea and terminal asset acquisition; declaration/member matching_destination_is_kept_and_chmodded_without_fetch; declarations/fields: `matching_destination_is_kept_and_chmodded_without_fetch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 117–126 | Pinned Muse, Tea and terminal asset acquisition; declaration/member stale_destination_is_replaced; declarations/fields: `stale_destination_is_replaced` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 128–143 | Pinned Muse, Tea and terminal asset acquisition; declaration/member non_200_status_fails_the_download; declarations/fields: `non_200_status_fails_the_download` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 145–159 | Pinned Muse, Tea and terminal asset acquisition; declaration/member relative_destination_is_refused_first; declarations/fields: `relative_destination_is_refused_first` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 161–171 | Pinned Muse, Tea and terminal asset acquisition; declaration/member unknown_architecture_is_refused; declarations/fields: `unknown_architecture_is_refused` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 173–219 | Pinned Muse, Tea and terminal asset acquisition; declaration/member invalid_pins_are_refused_without_fetch; declarations/fields: `invalid_pins_are_refused_without_fetch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 221–253 | Pinned Muse, Tea and terminal asset acquisition; declaration/member unknown_manifest_fields_are_decode_errors; declarations/fields: `unknown_manifest_fields_are_decode_errors` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 255–272 | Pinned Muse, Tea and terminal asset acquisition; declaration/member oversized_manifest_is_refused; declarations/fields: `oversized_manifest_is_refused` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 274–285 | Pinned Muse, Tea and terminal asset acquisition; declaration/member transport_errors_carry_the_download_prefix; declarations/fields: `transport_errors_carry_the_download_prefix` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 287–295 | Pinned Muse, Tea and terminal asset acquisition; declaration/member missing_manifest_is_an_error; declarations/fields: `missing_manifest_is_an_error` |

## [tools/release-assets/src/fetch/tea.rs](../../../../../tools/release-assets/src/fetch/tea.rs)

Re-audit @HEAD: C09 consolidation: inline tests extracted to `fetch/tea/tests.rs`; rows re-mapped declaration-by-declaration to current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1–4 | Pinned Muse, Tea and terminal asset acquisition; declarations/fields: `RELEASES_API`, `DL_BASE`, `LICENSE_BASE`, `USER_AGENT`, `TIMEOUT`, `BINARY_LIMIT`, `META_LIMIT`, `Endpoints`, `Endpoints.releases_api`, `Endpoints.dl_base`, `Endpoints.license_base`, `production`, `default_out`, `download`, `valid_tag`, `latest_tag`, `checksums_for`, `valid_elf64`, `fetch`, `tests` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 6 | Pinned Muse, Tea and terminal asset acquisition; declaration/member RELEASES_API; declarations/fields: `RELEASES_API` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 7 | Pinned Muse, Tea and terminal asset acquisition; declaration/member DL_BASE; declarations/fields: `DL_BASE` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 8 | Pinned Muse, Tea and terminal asset acquisition; declaration/member LICENSE_BASE; declarations/fields: `LICENSE_BASE` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 9 | Pinned Muse, Tea and terminal asset acquisition; declaration/member USER_AGENT; declarations/fields: `USER_AGENT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 10 | Pinned Muse, Tea and terminal asset acquisition; declaration/member TIMEOUT; declarations/fields: `TIMEOUT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 11 | Pinned Muse, Tea and terminal asset acquisition; declaration/member BINARY_LIMIT; declarations/fields: `BINARY_LIMIT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 12 | Pinned Muse, Tea and terminal asset acquisition; declaration/member META_LIMIT; declarations/fields: `META_LIMIT` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 16–20 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Endpoints; declarations/fields: `Endpoints` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 17 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Endpoints.releases_api; declarations/fields: `Endpoints.releases_api` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 18 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Endpoints.dl_base; declarations/fields: `Endpoints.dl_base` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 19 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Endpoints.license_base; declarations/fields: `Endpoints.license_base` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 23–30 | Pinned Muse, Tea and terminal asset acquisition; declaration/member production; declarations/fields: `production` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 35–41 | Pinned Muse, Tea and terminal asset acquisition; declaration/member default_out; declarations/fields: `default_out` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 43–53 | Pinned Muse, Tea and terminal asset acquisition; declaration/member download; declarations/fields: `download` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 56–73 | Pinned Muse, Tea and terminal asset acquisition; declaration/member valid_tag; declarations/fields: `valid_tag` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 75–89 | Pinned Muse, Tea and terminal asset acquisition; declaration/member latest_tag; declarations/fields: `latest_tag` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 93–102 | Pinned Muse, Tea and terminal asset acquisition; declaration/member checksums_for; declarations/fields: `checksums_for` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 104–109 | Pinned Muse, Tea and terminal asset acquisition; declaration/member valid_elf64; declarations/fields: `valid_elf64` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 112–162 | Pinned Muse, Tea and terminal asset acquisition; declaration/member fetch; declarations/fields: `fetch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 164–165 | Pinned Muse, Tea and terminal asset acquisition; declaration/member tests; declarations/fields: `tests` |

## [tools/release-assets/src/fetch/tea/tests.rs](../../../../../tools/release-assets/src/fetch/tea/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1–3 | Pinned Muse, Tea and terminal asset acquisition; declarations/fields: `VERSION`, `TAG`, `elf_body`, `Fixture`, `Fixture.server`, `Fixture.endpoints`, `Fixture.binary`, `Fixture.license`, `start`, `plain`, `mode`, `tag_shape_matches_version_tags_only`, `checksums_take_the_last_match_and_ignore_odd_lines`, `x86_64_stages_exact_bytes_license_modes_and_message`, `unsupported_architecture_is_refused_without_fetch`, `corrupt_download_leaves_no_stage`, `wrong_architecture_binary_leaves_no_stage`, `non_version_tag_leaves_no_stage`, `empty_license_leaves_no_stage`, `checksums_without_the_archive_leave_no_stage`, `shared_output_dir_stages_alongside_existing_files`, `existing_tea_outputs_are_refused_without_fetch`, `dangling_symlink_outputs_are_refused_without_fetch`, `http_errors_fail_without_a_stage`, `default_out_is_repo_shaped` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 5 | Pinned Muse, Tea and terminal asset acquisition; declaration/member VERSION; declarations/fields: `VERSION` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 6 | Pinned Muse, Tea and terminal asset acquisition; declaration/member TAG; declarations/fields: `TAG` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 8–13 | Pinned Muse, Tea and terminal asset acquisition; declaration/member elf_body; declarations/fields: `elf_body` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 15–20 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture; declarations/fields: `Fixture` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 16 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.server; declarations/fields: `Fixture.server` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 17 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.endpoints; declarations/fields: `Fixture.endpoints` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 18 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.binary; declarations/fields: `Fixture.binary` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 19 | Pinned Muse, Tea and terminal asset acquisition; declaration/member Fixture.license; declarations/fields: `Fixture.license` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 23–51 | Pinned Muse, Tea and terminal asset acquisition; declaration/member start; declarations/fields: `start` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 53–56 | Pinned Muse, Tea and terminal asset acquisition; declaration/member plain; declarations/fields: `plain` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 58–62 | Pinned Muse, Tea and terminal asset acquisition; declaration/member mode; declarations/fields: `mode` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 64–83 | Pinned Muse, Tea and terminal asset acquisition; declaration/member tag_shape_matches_version_tags_only; declarations/fields: `tag_shape_matches_version_tags_only` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 85–94 | Pinned Muse, Tea and terminal asset acquisition; declaration/member checksums_take_the_last_match_and_ignore_odd_lines; declarations/fields: `checksums_take_the_last_match_and_ignore_odd_lines` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 96–123 | Pinned Muse, Tea and terminal asset acquisition; declaration/member x86_64_stages_exact_bytes_license_modes_and_message; declarations/fields: `x86_64_stages_exact_bytes_license_modes_and_message` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 125–136 | Pinned Muse, Tea and terminal asset acquisition; declaration/member unsupported_architecture_is_refused_without_fetch; declarations/fields: `unsupported_architecture_is_refused_without_fetch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 138–178 | Pinned Muse, Tea and terminal asset acquisition; declaration/member corrupt_download_leaves_no_stage; declarations/fields: `corrupt_download_leaves_no_stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 180–217 | Pinned Muse, Tea and terminal asset acquisition; declaration/member wrong_architecture_binary_leaves_no_stage; declarations/fields: `wrong_architecture_binary_leaves_no_stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 219–240 | Pinned Muse, Tea and terminal asset acquisition; declaration/member non_version_tag_leaves_no_stage; declarations/fields: `non_version_tag_leaves_no_stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 242–276 | Pinned Muse, Tea and terminal asset acquisition; declaration/member empty_license_leaves_no_stage; declarations/fields: `empty_license_leaves_no_stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 278–302 | Pinned Muse, Tea and terminal asset acquisition; declaration/member checksums_without_the_archive_leave_no_stage; declarations/fields: `checksums_without_the_archive_leave_no_stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 304–318 | Pinned Muse, Tea and terminal asset acquisition; declaration/member shared_output_dir_stages_alongside_existing_files; declarations/fields: `shared_output_dir_stages_alongside_existing_files` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 320–336 | Pinned Muse, Tea and terminal asset acquisition; declaration/member existing_tea_outputs_are_refused_without_fetch; declarations/fields: `existing_tea_outputs_are_refused_without_fetch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 338–351 | Pinned Muse, Tea and terminal asset acquisition; declaration/member dangling_symlink_outputs_are_refused_without_fetch; declarations/fields: `dangling_symlink_outputs_are_refused_without_fetch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 353–368 | Pinned Muse, Tea and terminal asset acquisition; declaration/member http_errors_fail_without_a_stage; declarations/fields: `http_errors_fail_without_a_stage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 370–379 | Pinned Muse, Tea and terminal asset acquisition; declaration/member default_out_is_repo_shaped; declarations/fields: `default_out_is_repo_shaped` |

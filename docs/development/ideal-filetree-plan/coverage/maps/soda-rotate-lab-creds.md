# Soda rotate lab creds

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-09519f0e60d0"></a>

## [tools/lab-credentials/src/fixture_authority.rs](../../../../../tools/lab-credentials/src/fixture_authority.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–179; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–9: use crate and attached body; lines 10–12: use crate and attached body; lines 13–18: fn env_or and attached body; lines 19–29: fn current_umask and attached body; lines 30–35: fn write_staged and attached body; lines 36–44: fn is_executable and attached body; lines 45–48: fn command_v and attached body; lines 49–65: fn command_v_in and attached body; lines 66–74: fn random_hex_passphrase and attached body; lines 75–85: fn unix_now and attached body; lines 86–92: fn read_staged and attached body; lines 93–97: fn stage_file and attached body; lines 98–179: fn rotate_fixture_authority and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current fixture-or-asset source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9a982a902f85"></a>

## [tools/lab-credentials/src/inventory.rs](../../../../../tools/lab-credentials/src/inventory.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–171; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–10: use crate and attached body; lines 11–21: fn current_pwd and attached body; lines 22–29: fn note and attached body; lines 30–85: fn stat_line and attached body; lines 86–100: fn first_live_inputs and attached body; lines 101–171: fn inventory and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-899975a9ed04"></a>

## [tools/lab-credentials/src/main.rs](../../../../../tools/lab-credentials/src/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; lines 1–17: mod fixture_authority and attached body; lines 18–18: mod inventory and attached body; lines 19–19: mod process and attached body; lines 20–21: mod runbooks and attached body; lines 22–22: mod tests and attached body; lines 23–24: use std and attached body; lines 25–25: use std and attached body; lines 26–26: use std and attached body; lines 27–28: use crate and attached body; lines 29–29: use crate and attached body; lines 30–32: use crate and attached body; lines 33–34: const FAIL_PREFIX and attached body; lines 35–35: const PREFIX_DEFAULT and attached body; lines 36–36: const AUTHORITY and attached body; lines 37–37: const WORKER_USER and attached body; lines 38–43: use libc and attached body; lines 44–47: enum Exit and attached body; lines 48–51: fn fail and attached body; lines 52–102: fn run_rotate and attached body; lines 103–135: fn main and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for mod fixture_authority in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2045c9b07c14"></a>

## [tools/lab-credentials/src/process.rs](../../../../../tools/lab-credentials/src/process.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–330; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–18: fn status_code and attached body; lines 19–23: fn flush_stdout and attached body; lines 24–30: fn stripped and attached body; lines 31–34: fn stripped_string and attached body; lines 35–42: fn spawn_diag and attached body; lines 43–52: enum Captured and attached body; lines 53–80: fn capture and attached body; lines 81–85: fn run and attached body; lines 86–88: fn run_stdout_null and attached body; lines 89–125: fn run_with_io and attached body; lines 126–129: struct EnsureAsciiPretty and attached body; lines 130–131: impl serde_json and attached body; lines 132–148: fn write_string_fragment and attached body; lines 149–157: fn begin_array and attached body; lines 158–169: fn end_array and attached body; lines 170–177: fn begin_array_value and attached body; lines 178–185: fn end_array_value and attached body; lines 186–194: fn begin_object and attached body; lines 195–206: fn end_object and attached body; lines 207–214: fn begin_object_key and attached body; lines 215–221: fn begin_object_value and attached body; lines 222–230: fn end_object_value and attached body; lines 231–232: impl EnsureAsciiPretty and attached body; lines 233–242: fn write_indent and attached body; lines 243–256: fn pretty_json and attached body; lines 257–268: struct TrustRecord and attached body; lines 269–276: struct TrustKeys and attached body; lines 277–284: struct MinimumSequence and attached body; lines 285–291: struct ConfigRecord and attached body; lines 292–297: struct ConfigKeys and attached body; lines 298–321: fn trust_json and attached body; lines 322–330: fn config_json and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 36 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e9c1b7722425"></a>

## [tools/lab-credentials/src/runbooks.rs](../../../../../tools/lab-credentials/src/runbooks.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11; lines 1–3: const HELP_HEADER and attached body; lines 4–5: const RUNBOOK_FIXTURE and attached body; lines 6–7: const RUNBOOK_CLOUDFLARED and attached body; lines 8–9: const RUNBOOK_RUNNER and attached body; lines 10–11: const RUNBOOK_LAB_VM and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for const HELP_HEADER in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-80e40d1b2e10"></a>

## [tools/lab-credentials/src/tests.rs](../../../../../tools/lab-credentials/src/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–115; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–9: use crate and attached body; lines 10–25: fn trust_json_matches_python_dump_with_trailing_newline and attached body; lines 26–32: fn config_json_matches_python_dump_with_trailing_newline and attached body; lines 33–39: fn note_pads_tag_to_seven and attached body; lines 40–66: fn stat_line_reports_modes_without_content and attached body; lines 67–87: fn first_live_inputs_takes_sorted_first_match and attached body; lines 88–88: fn first_live_inputs_skips_dangling_symlinks and attached body; lines 89–108: use std and attached body; lines 109–115: fn passphrase_is_64_lowercase_hex_without_newline and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bd823b8df855"></a>

## [tools/lab-credentials/tests/cli.rs](../../../../../tools/lab-credentials/tests/cli.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–334; lines 1–6: use std and attached body; lines 7–7: use std and attached body; lines 8–8: use std and attached body; lines 9–9: use std and attached body; lines 10–10: use std and attached body; lines 11–12: static COUNTER and attached body; lines 13–16: struct TempDir and attached body; lines 17–18: impl TempDir and attached body; lines 19–27: fn new and attached body; lines 28–29: impl Drop and attached body; lines 30–33: fn drop and attached body; lines 34–40: fn bin and attached body; lines 41–60: fn cmd and attached body; lines 61–62: const HELP_HEADER and attached body; lines 63–64: const RUNBOOK_FIXTURE and attached body; lines 65–66: const RUNBOOK_CLOUDFLARED and attached body; lines 67–68: const RUNBOOK_RUNNER and attached body; lines 69–72: const RUNBOOK_LAB_VM and attached body; lines 73–88: fn help_spellings_print_header_usage_and_classes and attached body; lines 89–114: fn inventory_default_and_explicit_match_and_stay_structured and attached body; lines 115–132: fn inventory_ignores_extra_arguments and attached body; lines 133–149: fn inventory_reports_live_inputs_when_present and attached body; lines 150–182: fn rotate_runbooks_are_exact_with_or_without_execute and attached body; lines 183–209: fn unknown_command_and_class_fail_cleanly and attached body; lines 210–230: fn execute_requires_acknowledgement and attached body; lines 231–249: fn execute_requires_skopeo and attached body; lines 250–250: fn closed_stdout_dies_by_sigpipe_like_shell and attached body; lines 251–251: use std and attached body; lines 252–252: use std and attached body; lines 253–255: use std and attached body; lines 256–256: fn pipe and attached body; lines 257–283: fn close and attached body; lines 284–312: fn unset_pwd_user_and_home_crash_like_set_u and attached body; lines 313–334: fn missing_stat_stays_silent_like_redirected_shell and attached body | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 34 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-7e2657b0b728"></a>
<a id="rustsoda-rotate-lab-credssrcmainrs-1"></a>

Former source `rust/soda-rotate-lab-creds/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

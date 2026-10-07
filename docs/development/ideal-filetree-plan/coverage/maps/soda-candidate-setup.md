# Soda candidate setup

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-26c315f6bf17"></a>

## [tools/candidate-setup/src/config.rs](../../../../../tools/candidate-setup/src/config.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–252; lines 1–1: use std and attached body; lines 2–3: use serde and attached body; lines 4–6: use serde_json and attached body; lines 7–10: struct EnsureAsciiPretty and attached body; lines 11–12: impl Formatter and attached body; lines 13–29: fn write_string_fragment and attached body; lines 30–38: fn begin_array and attached body; lines 39–50: fn end_array and attached body; lines 51–58: fn begin_array_value and attached body; lines 59–66: fn end_array_value and attached body; lines 67–75: fn begin_object and attached body; lines 76–87: fn end_object and attached body; lines 88–95: fn begin_object_key and attached body; lines 96–102: fn begin_object_value and attached body; lines 103–111: fn end_object_value and attached body; lines 112–113: impl EnsureAsciiPretty and attached body; lines 114–123: fn write_indent and attached body; lines 124–138: fn pretty_json and attached body; lines 139–152: struct WorkerRecord and attached body; lines 153–164: struct TrustRecord and attached body; lines 165–172: struct TrustKeys and attached body; lines 173–180: struct MinimumSequence and attached body; lines 181–187: struct ConfigRecord and attached body; lines 188–194: struct ConfigKeys and attached body; lines 195–219: fn worker_json and attached body; lines 220–243: fn trust_json and attached body; lines 244–252: fn config_json and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 27 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6a8d8629ba24"></a>

## [tools/candidate-setup/src/controller.rs](../../../../../tools/candidate-setup/src/controller.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–103; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use super and attached body; lines 6–6: use super and attached body; lines 7–9: use super and attached body; lines 10–15: struct ControllerPaths and attached body; lines 16–30: fn controller_cargo_argv and attached body; lines 31–103: fn admit_controller and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-1bef5f9c9e02"></a>

## [tools/candidate-setup/src/fixture_authority.rs](../../../../../tools/candidate-setup/src/fixture_authority.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–177; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use super and attached body; lines 6–6: use super and attached body; lines 7–7: use super and attached body; lines 8–11: use super and attached body; lines 12–20: fn random_hex_passphrase and attached body; lines 21–30: fn unix_now and attached body; lines 31–37: fn read_staged and attached body; lines 38–45: fn stage_file and attached body; lines 46–177: fn admit_fixture_authority and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current fixture-or-asset source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3143d6ca837c"></a>

## [tools/candidate-setup/src/main.rs](../../../../../tools/candidate-setup/src/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–245; lines 1–14: use std and attached body; lines 15–15: use std and attached body; lines 16–16: use std and attached body; lines 17–17: use std and attached body; lines 18–18: use std and attached body; lines 19–19: use std and attached body; lines 20–20: use std and attached body; lines 21–22: mod config and attached body; lines 23–23: mod controller and attached body; lines 24–24: mod fixture_authority and attached body; lines 25–25: mod preflight and attached body; lines 26–26: mod process and attached body; lines 27–27: mod selinux and attached body; lines 28–28: mod storage and attached body; lines 29–29: mod worker_caches and attached body; lines 30–30: mod worker_tools and attached body; lines 31–32: use self and attached body; lines 33–34: const FAIL_PREFIX and attached body; lines 35–35: const PREFIX_DEFAULT and attached body; lines 36–36: const STORAGE_ROOT_DEFAULT and attached body; lines 37–37: const ROOTFS_DIR and attached body; lines 38–38: const ADMITTED and attached body; lines 39–39: const PINNED_GO and attached body; lines 40–40: const WRAPPER and attached body; lines 41–41: const TOOLS and attached body; lines 42–42: const WORKER_POLICY_SRC and attached body; lines 43–43: const AUTHORITY and attached body; lines 44–44: const LEGACY_HOME and attached body; lines 45–45: const LEGACY_RUN and attached body; lines 46–46: const WORKER_USER and attached body; lines 47–54: use libc and attached body; lines 55–58: enum Exit and attached body; lines 59–64: fn fail and attached body; lines 65–73: fn env_or and attached body; lines 74–81: fn current_pwd and attached body; lines 82–85: fn is_dir and attached body; lines 86–91: fn is_file and attached body; lines 92–98: fn stripped and attached body; lines 99–102: fn stripped_string and attached body; lines 103–113: fn current_umask and attached body; lines 114–119: fn write_staged and attached body; lines 120–128: fn is_executable and attached body; lines 129–132: fn command_v and attached body; lines 133–146: fn command_v_in and attached body; lines 147–216: fn run_setup and attached body; lines 217–244: fn main and attached body; lines 245–245: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 47 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-484b9bde0c82"></a>

## [tools/candidate-setup/src/preflight.rs](../../../../../tools/candidate-setup/src/preflight.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–194; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use super and attached body; lines 5–5: use super and attached body; lines 6–9: use super and attached body; lines 10–13: use super and attached body; lines 14–29: struct Preflight and attached body; lines 30–43: fn bridge_ip and attached body; lines 44–171: fn preflight and attached body; lines 172–194: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-152b2a306a99"></a>

## [tools/candidate-setup/src/process.rs](../../../../../tools/candidate-setup/src/process.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–227; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use super and attached body; lines 6–18: fn status_code and attached body; lines 19–21: fn flush_stdout and attached body; lines 22–29: fn spawn_diag and attached body; lines 30–39: enum Captured and attached body; lines 40–68: fn capture and attached body; lines 69–76: fn git_tree_clean and attached body; lines 77–81: fn run and attached body; lines 82–86: fn run_stdout_null and attached body; lines 87–89: fn run_stderr_null and attached body; lines 90–144: fn run_with_io and attached body; lines 145–150: fn run_piped_stdin and attached body; lines 151–195: fn pipe2 and attached body; lines 196–204: fn ls_nonempty and attached body; lines 205–214: fn id_un and attached body; lines 215–227: fn run_in_dir and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-86ab95cf1fcd"></a>

## [tools/candidate-setup/src/selinux.rs](../../../../../tools/candidate-setup/src/selinux.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–114; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–6: use super and attached body; lines 7–22: fn fcontext_add_or_modify and attached body; lines 23–30: fn selinux_has_type and attached body; lines 31–114: fn install_worker_selinux and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fdc4f28a212e"></a>

## [tools/candidate-setup/src/storage.rs](../../../../../tools/candidate-setup/src/storage.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–95; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–7: use super and attached body; lines 8–16: fn units_have_active and attached body; lines 17–41: fn refuse_active_build and attached body; lines 42–48: struct Storage and attached body; lines 49–85: fn migrate_candidate_home and attached body; lines 86–95: fn prepare_scratch and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-910902b8a66c"></a>

## [tools/candidate-setup/src/tests.rs](../../../../../tools/candidate-setup/src/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–221; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–6: use super and attached body; lines 7–7: use super and attached body; lines 8–8: use super and attached body; lines 9–9: use super and attached body; lines 10–10: use super and attached body; lines 11–11: use super and attached body; lines 12–14: use super and attached body; lines 15–31: fn worker_json_matches_python_dump_without_trailing_newline and attached body; lines 32–47: fn trust_json_matches_python_dump_with_trailing_newline and attached body; lines 48–54: fn config_json_matches_python_dump_with_trailing_newline and attached body; lines 55–78: fn units_have_active_matches_awk_third_field and attached body; lines 79–92: fn bridge_ip_takes_first_line_fourth_field_before_slash and attached body; lines 93–104: fn env_or_falls_back_on_unset_or_empty and attached body; lines 105–114: fn stripped_removes_only_trailing_newlines and attached body; lines 115–133: fn command_v_searches_path_for_executables and attached body; lines 134–143: fn pipe2_reports_last_nonzero_by_position and attached body; lines 144–154: fn write_staged_matches_redirection_bytes_and_mode and attached body; lines 155–167: fn passphrase_is_64_lowercase_hex_without_newline and attached body; lines 168–181: fn run_in_dir_maps_missing_directory_to_fail_message and attached body; lines 182–194: fn git_tree_clean_requires_successful_empty_status and attached body; lines 195–215: fn controller_build_selects_rust_release_tools and attached body; lines 216–221: fn worker_policy_input_resolves_in_checkout and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fbe652fc4d26"></a>

## [tools/candidate-setup/src/worker_caches.rs](../../../../../tools/candidate-setup/src/worker_caches.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–5: use super and attached body; lines 6–11: struct WorkerCaches and attached body; lines 12–169: fn warm_worker_caches and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for struct WorkerCaches in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn warm_worker_caches in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e336471439c7"></a>

## [tools/candidate-setup/src/worker_tools.rs](../../../../../tools/candidate-setup/src/worker_tools.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–152; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use super and attached body; lines 4–8: use super and attached body; lines 9–14: struct WorkerTools and attached body; lines 15–152: fn provision_worker_tools and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for struct WorkerTools in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn provision_worker_tools in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-dc7b52788205"></a>
<a id="rustsoda-candidate-setuptestsclirs-1"></a>
<a id="coverage-0bd65d814669"></a>

## [tools/candidate-setup/tests/cli.rs](../../../../../tools/candidate-setup/tests/cli.rs)

Current coherent source duties matched to live consumer and prior semantic unit context

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–335; lines 11–12: mod support; attached module comments and attributes; lines 18–20: LEASE_SERIAL and attached behavior; lines 21–39: fails_outside_repo_root and attached behavior; lines 40–59: requires_forgejo_source and attached behavior; lines 60–80: rejects_relative_forgejo and attached behavior; lines 81–109: rejects_dirty_forgejo and attached behavior; lines 110–151: reports_missing_tool and attached behavior; lines 152–168: missing_ip_stays_silent and attached behavior; lines 169–200: full_fake_bin and attached behavior; lines 201–212: clean_repo and attached behavior; lines 213–237: missing_go_pin_exits_silently and attached behavior; lines 238–273: refuses_held_lease and attached behavior; lines 274–304: lists_units_quietly and attached behavior; lines 305–335: refuses_active_build and attached behavior | [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) | retained | Current mod support and its attached implementation body; current module purpose and consumer determine this responsibility. Adjacent comments and attributes explain this same authored responsibility.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-43f09efdd565"></a>

## [tools/candidate-setup/tests/support/mod.rs](../../../../../tools/candidate-setup/tests/support/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–136; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: static COUNTER and attached body; lines 9–12: struct TempDir and attached body; lines 13–14: impl TempDir and attached body; lines 15–22: fn new and attached body; lines 23–24: impl Drop and attached body; lines 25–28: fn drop and attached body; lines 29–36: fn bin and attached body; lines 37–55: fn cmd and attached body; lines 56–61: fn write_fake and attached body; lines 62–78: fn real_tool and attached body; lines 79–81: fn fake_ip and attached body; lines 82–114: fn git_init and attached body; lines 115–119: fn flock and attached body; lines 120–122: struct HeldLock and attached body; lines 123–130: fn hold_flock_lock and attached body; lines 131–136: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-9bdcb2a9512b"></a>
<a id="rustsoda-candidate-setupsrcmainrs-1"></a>

Former source `rust/soda-candidate-setup/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

# Soda test vm

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-355ef73ed0f9"></a>

## [tools/test-vm/src/main.rs](../../../../../tools/test-vm/src/main.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; lines 1–18: use std and attached body; lines 19–19: use std and attached body; lines 20–20: use std and attached body; lines 21–21: use std and attached body; lines 22–22: use std and attached body; lines 23–24: mod process and attached body; lines 25–25: mod start and attached body; lines 26–26: mod transport and attached body; lines 27–28: use self and attached body; lines 29–29: mod state and attached body; lines 30–31: const FAIL_PREFIX and attached body; lines 32–32: const QEMU_DEFAULT and attached body; lines 33–33: const SSH_PORT and attached body; lines 34–41: use libc and attached body; lines 42–46: enum Exit and attached body; lines 47–50: fn refuse and attached body; lines 51–63: fn status_code and attached body; lines 64–68: fn flush_stdout and attached body; lines 69–80: fn env_or and attached body; lines 81–92: fn current_pwd and attached body; lines 93–99: fn stripped and attached body; lines 100–103: fn stripped_string and attached body; lines 104–128: fn main and attached body; lines 129–129: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ba4048fa343c"></a>

## [tools/test-vm/src/process.rs](../../../../../tools/test-vm/src/process.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–179; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–7: use std and attached body; lines 8–8: use std and attached body; lines 9–10: use super and attached body; lines 11–14: fn os_error and attached body; lines 15–21: fn access and attached body; lines 22–25: fn is_file and attached body; lines 26–31: enum Captured and attached body; lines 32–56: fn capture and attached body; lines 57–57: fn capture_stdout and attached body; lines 58–58: const CAP and attached body; lines 59–123: const PIPE_GRACE and attached body; lines 124–148: fn run and attached body; lines 149–163: fn exec_replace and attached body; lines 164–164: mod capture_tests and attached body; lines 165–167: use super and attached body; lines 168–172: fn infinite_output_fails_at_capture_bound and attached body; lines 173–179: fn descendant_pipe_is_a_capture_failure and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-83bf10015223"></a>

## [tools/test-vm/src/start.rs](../../../../../tools/test-vm/src/start.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–95; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use super and attached body; lines 5–5: use super and attached body; lines 6–9: use super and attached body; lines 10–95: fn action_start and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn action_start in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d9f1ecd4382c"></a>

## [tools/test-vm/src/state.rs](../../../../../tools/test-vm/src/state.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–93; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–7: use super and attached body; lines 8–14: fn pid_display and attached body; lines 15–50: fn pid_alive and attached body; lines 51–62: fn vm_running and attached body; lines 63–69: fn vm_dir and attached body; lines 70–86: fn ssh_args and attached body; lines 87–93: fn uname_is and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-782d305657ec"></a>

## [tools/test-vm/src/tests.rs](../../../../../tools/test-vm/src/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–54; lines 1–1: use super and attached body; lines 2–4: use super and attached body; lines 5–11: fn pid_display_keeps_spaces_drops_newlines and attached body; lines 12–27: fn pid_alive_matches_kill_zero_semantics and attached body; lines 28–49: fn ssh_args_pin_key_paths_and_options and attached body; lines 50–54: fn stripped_drops_all_trailing_newlines and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-82eadd5db227"></a>

## [tools/test-vm/src/transport.rs](../../../../../tools/test-vm/src/transport.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–95; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–4: use super and attached body; lines 5–5: use super and attached body; lines 6–6: use super and attached body; lines 7–27: fn action_status and attached body; lines 28–35: fn action_ssh and attached body; lines 36–58: fn action_tunnel and attached body; lines 59–71: fn action_console and attached body; lines 72–95: fn run_vm and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b3bce85fd039"></a>

## [tools/test-vm/tests/cli.rs](../../../../../tools/test-vm/tests/cli.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–74; lines 1–8: use std and attached body; lines 9–10: mod start and attached body; lines 11–11: mod status and attached body; lines 12–13: mod support and attached body; lines 14–14: mod transport and attached body; lines 15–18: use self and attached body; lines 19–19: fn faccessat and attached body; lines 20–23: fn flock and attached body; lines 24–43: fn stale_pwd_falls_back_to_working_directory and attached body; lines 44–44: fn closed_stdout_dies_by_sigpipe_like_shell and attached body; lines 45–45: use std and attached body; lines 46–46: use std and attached body; lines 47–49: use std and attached body; lines 50–50: fn pipe and attached body; lines 51–74: fn close and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c00d8671b1c0"></a>

## [tools/test-vm/tests/start.rs](../../../../../tools/test-vm/tests/start.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–394; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use super and attached body; lines 6–8: use super and attached body; lines 9–56: fn start_refuses_without_kvm_or_with_bad_qemu and attached body; lines 57–88: fn start_reports_missing_files_in_order and attached body; lines 89–91: struct HeldLock and attached body; lines 92–93: impl HeldLock and attached body; lines 94–100: fn new and attached body; lines 101–105: use std and attached body; lines 106–107: impl Drop and attached body; lines 108–108: fn drop and attached body; lines 109–116: use std and attached body; lines 117–153: fn start_refuses_held_lock_and_running_vm and attached body; lines 154–165: fn write_fake_qemu and attached body; lines 166–232: fn start_runs_qemu_with_exact_argv_and_releases_lock and attached body; lines 233–240: use std and attached body; lines 241–261: fn start_propagates_qemu_status_silently and attached body; lines 262–312: fn start_reports_qemu_killed_by_signal and attached body; lines 313–337: fn unreadable_pidfile_reports_and_aborts and attached body; lines 338–373: fn unreadable_pidfile_aborts_start_before_qemu and attached body; lines 374–394: fn missing_uname_keeps_native_error_and_refuses and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bdf1af16dfde"></a>

## [tools/test-vm/tests/status.rs](../../../../../tools/test-vm/tests/status.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; lines 1–1: use std and attached body; lines 2–5: use super and attached body; lines 6–14: fn unknown_action_prints_usage_and_exits_two and attached body; lines 15–25: fn empty_action_defaults_to_status and attached body; lines 26–46: fn status_reports_not_running_without_pidfile and attached body; lines 47–70: fn status_rejects_unparseable_pidfiles_silently and attached body; lines 71–89: fn status_reports_live_pid_and_keeps_spacing and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ec1e35636cb3"></a>

## [tools/test-vm/tests/support/mod.rs](../../../../../tools/test-vm/tests/support/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–73; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–6: use std and attached body; lines 7–8: use super and attached body; lines 9–10: static COUNTER and attached body; lines 11–12: fn kvm_accessible and attached body; lines 13–13: use std and attached body; lines 14–19: use std and attached body; lines 20–23: struct TempDir and attached body; lines 24–25: impl TempDir and attached body; lines 26–33: fn new and attached body; lines 34–35: impl Drop and attached body; lines 36–39: fn drop and attached body; lines 40–46: fn bin and attached body; lines 47–53: fn cmd and attached body; lines 54–62: fn write_fake and attached body; lines 63–71: fn vm_fixture and attached body; lines 72–73: const USAGE and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cdd1ccce23cd"></a>

## [tools/test-vm/tests/transport.rs](../../../../../tools/test-vm/tests/transport.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–166; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–6: use super and attached body; lines 7–71: fn ssh_exec_failures_keep_native_errors_and_shell_statuses and attached body; lines 72–166: fn ssh_and_tunnels_exec_with_exact_argv and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current test-or-fixture source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-6c17b7a12499"></a>
<a id="rustsoda-test-vmsrcmainrs-1"></a>

Former source `rust/soda-test-vm/src/main.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-31d4f7093d97"></a>
<a id="rustsoda-test-vmtestsclirs-1"></a>

Former source `rust/soda-test-vm/tests/cli.rs`; consult its pinned earlier Git source and the current coverage disposition.

# Acceptance native qualification

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-8b92dccc109c"></a>

## [tools/acceptance/build.rs](../../../../../tools/acceptance/build.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30; lines 1–16: fn git_output and attached body; lines 17–30: fn main and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for fn git_output in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn main in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — tools/acceptance/build.rs:1-16; current source declaration and body; tools/acceptance/build.rs:17-30; current source declaration and body |

<a id="coverage-461a9efac9ba"></a>

## [tools/acceptance/src/bin/soda-acceptance-remote.rs](../../../../../tools/acceptance/src/bin/soda-acceptance-remote.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; lines 1–4: use soda_acceptance and attached body; lines 5–28: fn native_phase and attached body; lines 29–53: fn cockpit_account and attached body; lines 54–66: fn project_state and attached body; lines 67–85: fn main and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use soda_acceptance in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5d24c2bf4d43"></a>

## [tools/acceptance/src/bin/soda-host-probes.rs](../../../../../tools/acceptance/src/bin/soda-host-probes.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39; lines 1–2: use std and attached body; lines 3–4: use soda_acceptance and attached body; lines 5–10: fn read_stdin and attached body; lines 11–39: fn main and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-bb3d363712da"></a>

## [tools/acceptance/src/cockpit.rs](../../../../../tools/acceptance/src/cockpit.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–285; lines 1–27: use std and attached body; lines 28–36: struct Conversation and attached body; lines 37–43: type PamStart and attached body; lines 44–44: type PamAcctMgmt and attached body; lines 45–48: type PamEnd and attached body; lines 49–54: enum CockpitKind and attached body; lines 55–57: impl CockpitKind and attached body; lines 58–67: fn name and attached body; lines 68–73: struct CockpitFailure and attached body; lines 74–75: impl CockpitFailure and attached body; lines 76–82: fn assertion and attached body; lines 83–84: impl std and attached body; lines 85–93: fn fmt and attached body; lines 94–98: impl std and attached body; lines 99–104: fn check_gate and attached body; lines 105–130: fn account_uid and attached body; lines 131–136: struct Pam and attached body; lines 137–139: impl Pam and attached body; lines 140–186: fn load and attached body; lines 187–225: fn account_phase and attached body; lines 226–251: fn run_cockpit and attached body; lines 252–252: mod tests and attached body; lines 253–255: use super and attached body; lines 256–272: fn gate_requires_root_and_matching_validation and attached body; lines 273–279: fn root_account_resolves_to_uid_zero and attached body; lines 280–285: fn pam_phases_need_a_native_root_target and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a2ffbbf74961"></a>

## [tools/acceptance/src/command/execute.rs](../../../../../tools/acceptance/src/command/execute.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–132; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–7: use super and attached body; lines 8–14: fn valid_command_label and attached body; lines 15–18: fn redact_opt and attached body; lines 19–30: fn close_shared and attached body; lines 31–132: fn execute and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3971b4c8cca0"></a>

## [tools/acceptance/src/command/mod.rs](../../../../../tools/acceptance/src/command/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–84; lines 1–8: use crate and attached body; lines 9–10: mod execute and attached body; lines 11–11: mod ssh and attached body; lines 12–13: use self and attached body; lines 14–17: use self and attached body; lines 18–28: enum StdinSpec and attached body; lines 29–43: struct CommandSpec and attached body; lines 44–58: struct CommandResult and attached body; lines 59–64: fn quote and attached body; lines 65–70: fn look_path and attached body; lines 71–83: use std and attached body; lines 84–84: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-eb4e2aaa0a64"></a>

## [tools/acceptance/src/command/ssh.rs](../../../../../tools/acceptance/src/command/ssh.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–242; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use serde_json and attached body; lines 6–6: use std and attached body; lines 7–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–14: use super and attached body; lines 15–32: struct Remote and attached body; lines 33–71: fn decode_remote and attached body; lines 72–80: fn valid_ssh_user and attached body; lines 81–92: fn valid_ssh_host and attached body; lines 93–96: fn valid_ssh_port and attached body; lines 97–102: fn trusted_known_hosts and attached body; lines 103–108: use std and attached body; lines 109–111: impl Remote and attached body; lines 112–149: fn args and attached body; lines 150–179: fn command and attached body; lines 180–195: fn wait_ready and attached body; lines 196–196: mod json_tests and attached body; lines 197–197: use super and attached body; lines 198–200: use serde_json and attached body; lines 201–215: fn remote_record_uses_last_exact_fields_before_type_decoding and attached body; lines 216–242: fn ssh_true and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a27e16b01ba9"></a>

## [tools/acceptance/src/command/tests.rs](../../../../../tools/acceptance/src/command/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–185; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–6: use super and attached body; lines 7–19: fn fixture_evidence and attached body; lines 20–21: fn fresh_id and attached body; lines 22–22: use std and attached body; lines 23–27: static NEXT and attached body; lines 28–43: fn literal_remote_arguments and attached body; lines 44–51: fn pinned_ssh_options and attached body; lines 52–86: use std and attached body; lines 87–122: fn command_and_evidence_failures_are_separate and attached body; lines 123–151: fn output_limit_failure_survives_nonzero_native_exit and attached body; lines 152–167: fn cancelled_command_is_not_denial_or_success and attached body; lines 168–181: fn secret_evidence and attached body; lines 182–185: fn fixture_evidence_pair and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0ba66b4bcdf7"></a>
<a id="rustsoda-acceptancesrccoreosrs-1"></a>
<a id="coverage-c2ec809eb8f3"></a>

## [tools/acceptance/src/coreos.rs](../../../../../tools/acceptance/src/coreos.rs)

Current module body and direct consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–482; whole file: acceptance-driver coreos stream resolution for qualification guest setup; the acceptance tool owns this test input acquisition path | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Acceptance-driver CoreOS stream resolution for qualification guest setup; the acceptance tool owns this test input acquisition path. — tools/acceptance/src/coreos.rs:1-482; module purpose and current direct consumer inspected |

<a id="coverage-8052f08e357b"></a>

## [tools/acceptance/src/coreos/tests.rs](../../../../../tools/acceptance/src/coreos/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–181; lines 1–1: use super and attached body; lines 2–23: fn stream_doc and attached body; lines 24–47: fn stream_build_parses_release_and_triples and attached body; lines 48–65: fn registry_endpoint_gates and attached body; lines 66–74: fn status_text_prefers_last_response and attached body; lines 75–112: fn fetch_capture_flushes_newline_free_metadata_before_parsing and attached body; lines 113–158: fn fetch_capture_rejects_pump_and_both_close_failures and attached body; lines 159–171: fn fetch_gate_rejects_plain_http_without_network and attached body; lines 172–181: fn resolve_gate_rejects_plain_http_stream and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7024b150e511"></a>

## [tools/acceptance/src/driver/actions.rs](../../../../../tools/acceptance/src/driver/actions.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–251; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–14: use super and attached body; lines 15–24: fn payload_bytes and attached body; lines 25–91: fn execute_exec_or_native and attached body; lines 92–120: fn execute_probe_ssh and attached body; lines 121–163: fn run_vm_phase and attached body; lines 164–216: fn execute_vm and attached body; lines 217–251: fn execute_action and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d1ba0a3ec7f1"></a>

## [tools/acceptance/src/driver/finalization.rs](../../../../../tools/acceptance/src/driver/finalization.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–94; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–14: fn read_build_settings and attached body; lines 15–35: fn redact_observation_strings and attached body; lines 36–45: fn combine_messages and attached body; lines 46–94: fn finalize_observation and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e3ed9e3a7214"></a>

## [tools/acceptance/src/driver/inputs.rs](../../../../../tools/acceptance/src/driver/inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; lines 1–1: use crate and attached body; lines 2–2: use crate and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–7: use super and attached body; lines 8–24: fn read_secret_files and attached body; lines 25–38: fn load_vm_secrets and attached body; lines 39–51: fn collect_all_secrets and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-01eb791cef47"></a>

## [tools/acceptance/src/driver/mod.rs](../../../../../tools/acceptance/src/driver/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–224; lines 1–10: use std and attached body; lines 11–11: use std and attached body; lines 12–13: use crate and attached body; lines 14–14: use crate and attached body; lines 15–15: use crate and attached body; lines 16–16: use crate and attached body; lines 17–17: use crate and attached body; lines 18–18: use crate and attached body; lines 19–19: use crate and attached body; lines 20–21: mod actions and attached body; lines 22–22: mod finalization and attached body; lines 23–23: mod inputs and attached body; lines 24–24: mod options and attached body; lines 25–26: use actions and attached body; lines 27–27: use clap and attached body; lines 28–28: use finalization and attached body; lines 29–30: use inputs and attached body; lines 31–31: use options and attached body; lines 32–34: use options and attached body; lines 35–44: static SIGNAL_PHASE and attached body; lines 45–54: fn install_signal_forwarding and attached body; lines 55–67: fn client_platform and attached body; lines 68–110: fn init_observation and attached body; lines 111–145: fn run and attached body; lines 146–199: fn report and attached body; lines 200–209: fn top_level_command and attached body; lines 210–223: fn requested_top_level_help and attached body; lines 224–224: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 28 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4c429b396fec"></a>

## [tools/acceptance/src/driver/options.rs](../../../../../tools/acceptance/src/driver/options.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–241; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–6: const HELP_REQUESTED and attached body; lines 7–9: use clap and attached body; lines 10–21: fn parse_duration and attached body; lines 22–41: struct RunOptions and attached body; lines 42–53: fn valid_target and attached body; lines 54–72: fn validate_common_options and attached body; lines 73–105: fn validate_action_options and attached body; lines 106–113: fn value and attached body; lines 114–125: fn boolean and attached body; lines 126–167: fn run_command and attached body; lines 168–171: fn help_text and attached body; lines 172–241: fn parse_run_options and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d28357d7692f"></a>

## [tools/acceptance/src/driver/tests.rs](../../../../../tools/acceptance/src/driver/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–232; lines 1–1: use super and attached body; lines 2–4: use crate and attached body; lines 5–22: fn durations_use_bounded_humantime_grammar and attached body; lines 23–61: fn clap_schema_keeps_repeats_overrides_and_literal_exec_tail and attached body; lines 62–85: fn generated_help_is_clean_and_does_not_consume_exec_tail_help and attached body; lines 86–149: fn option_validation_matches_go_messages and attached body; lines 150–171: fn run_args and attached body; lines 172–183: fn invalid_actions_do_not_create_evidence and attached body; lines 184–214: fn cancelled_execution_records_failure and attached body; lines 215–232: fn evidence_failure_records_failed_evidence_and_outcome and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3e9a4ed3964c"></a>

## [tools/acceptance/src/error.rs](../../../../../tools/acceptance/src/error.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; lines 1–11: use std and attached body; lines 12–15: use std and attached body; lines 16–27: enum Error and attached body; lines 28–30: impl Error and attached body; lines 31–39: fn msg and attached body; lines 40–47: fn redacted and attached body; lines 48–56: fn wrap and attached body; lines 57–76: fn join and attached body; lines 77–85: fn is_cancelled and attached body; lines 86–93: fn io_kind and attached body; lines 94–95: impl fmt and attached body; lines 96–103: fn fmt and attached body; lines 104–105: impl std and attached body; lines 106–115: fn source and attached body; lines 116–117: impl From and attached body; lines 118–121: fn from and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f669e8b885ac"></a>

## [tools/acceptance/src/evidence/mod.rs](../../../../../tools/acceptance/src/evidence/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; lines 1–12: use crate and attached body; lines 13–14: mod redaction and attached body; lines 15–15: mod store and attached body; lines 16–17: use redaction and attached body; lines 18–20: use store and attached body; lines 21–23: const EVIDENCE_LIMIT and attached body; lines 24–27: struct Evidence and attached body; lines 28–34: fn contains_slice and attached body; lines 35–40: fn longest_secret and attached body; lines 41–41: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-13f7389564f2"></a>

## [tools/acceptance/src/evidence/redaction.rs](../../../../../tools/acceptance/src/evidence/redaction.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–410; whole file: secret redaction primitives for qualification evidence, including bounded pattern matching and safe replacement | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Secret redaction primitives for qualification evidence, including bounded pattern matching and safe replacement. — tools/acceptance/src/evidence/redaction.rs; consumed by evidence store writers |

<a id="coverage-470f166c09bb"></a>

## [tools/acceptance/src/evidence/store.rs](../../../../../tools/acceptance/src/evidence/store.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–609; whole file: private qualification evidence store: path admission, bounded json serialization, secret scrubbing, atomic writes and evidence rechecks | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Private qualification evidence store: path admission, bounded JSON serialization, secret scrubbing, atomic writes and evidence rechecks. — tools/acceptance/src/evidence/store.rs; consumed by acceptance phase/report writers |

<a id="coverage-8d627ab39ed4"></a>

## [tools/acceptance/src/evidence/tests.rs](../../../../../tools/acceptance/src/evidence/tests.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–431; whole file: source assertions for qualification evidence storage, redaction and secret exclusion | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Source assertions for qualification evidence storage, redaction and secret exclusion. — tools/acceptance/src/evidence/tests.rs; test module for evidence store/redaction |

<a id="coverage-9c810ac3076f"></a>

## [tools/acceptance/src/files/inputs.rs](../../../../../tools/acceptance/src/files/inputs.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–5: use std and attached body; lines 6–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–13: use super and attached body; lines 14–53: fn private_file and attached body; lines 54–61: fn read_limited and attached body; lines 62–72: fn hash_file and attached body; lines 73–84: fn hash_at and attached body; lines 85–93: fn fstat_of and attached body; lines 94–106: fn hash_reader and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-62d2501cb951"></a>

## [tools/acceptance/src/files/mod.rs](../../../../../tools/acceptance/src/files/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–111; lines 1–8: use std and attached body; lines 9–9: use std and attached body; lines 10–10: use std and attached body; lines 11–12: use crate and attached body; lines 13–14: mod inputs and attached body; lines 15–15: mod owned_directory and attached body; lines 16–16: mod temporary and attached body; lines 17–18: use inputs and attached body; lines 19–19: use owned_directory and attached body; lines 20–22: use temporary and attached body; lines 23–24: const PRIVATE_FILE_LIMIT and attached body; lines 25–29: const JSON_LIMIT and attached body; lines 30–54: fn lexical_clean and attached body; lines 55–66: fn c_string and attached body; lines 67–86: struct FileAttr and attached body; lines 87–88: impl FileAttr and attached body; lines 89–105: fn from_stat and attached body; lines 106–110: fn same_file and attached body; lines 111–111: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7a719e96bfd3"></a>

## [tools/acceptance/src/files/owned_directory.rs](../../../../../tools/acceptance/src/files/owned_directory.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–287; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–6: use crate and attached body; lines 7–13: use super and attached body; lines 14–19: struct OwnedDir and attached body; lines 20–22: impl OwnedDir and attached body; lines 23–41: fn open and attached body; lines 42–44: fn path and attached body; lines 45–53: fn single_component and attached body; lines 54–70: fn open_child_dir and attached body; lines 71–81: fn traverse and attached body; lines 82–91: fn split_parent and attached body; lines 92–112: fn lstat_at and attached body; lines 113–130: fn open_file_at and attached body; lines 131–144: fn sub_dir and attached body; lines 145–155: fn mkdir_at and attached body; lines 156–174: fn create_new_at and attached body; lines 175–193: fn link_at and attached body; lines 194–206: fn duplicate and attached body; lines 207–212: fn walk_files and attached body; lines 213–287: fn walk_recursion and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9910a9e9984e"></a>

## [tools/acceptance/src/files/temporary.rs](../../../../../tools/acceptance/src/files/temporary.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–4: use std and attached body; lines 5–6: use crate and attached body; lines 7–11: use super and attached body; lines 12–30: fn fresh_directory and attached body; lines 31–51: fn private_destination and attached body; lines 52–54: struct TempDir and attached body; lines 55–57: impl TempDir and attached body; lines 58–58: fn new and attached body; lines 59–67: use std and attached body; lines 68–72: fn join and attached body; lines 73–78: fn path and attached body; lines 79–79: mod tests and attached body; lines 80–80: use super and attached body; lines 81–83: use std and attached body; lines 84–100: fn private_tempdir_is_created_and_removed_by_owner and attached body; lines 101–109: fn write_new and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3fd42d2536cf"></a>

## [tools/acceptance/src/files/tests.rs](../../../../../tools/acceptance/src/files/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–290; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use super and attached body; lines 5–11: fn temp_dir and attached body; lines 12–13: fn rand_suffix and attached body; lines 14–14: use std and attached body; lines 15–22: use std and attached body; lines 23–38: fn lexical_clean_matches_go and attached body; lines 39–76: fn private_file_matrix and attached body; lines 77–105: fn bounded_private_read_observes_growth_on_open_inode and attached body; lines 106–123: fn hash_file_vectors and attached body; lines 124–170: fn owned_dir_confines_and_links and attached body; lines 171–213: fn fresh_directory_and_destination_gates and attached body; lines 214–230: fn fixture_dir_fds and attached body; lines 231–258: fn walk_lstat_error_closes_stream and attached body; lines 259–271: fn walk_all and attached body; lines 272–290: fn walk_repeated_scans_agree and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d2a7a97ef622"></a>

## [tools/acceptance/src/host_probes/content.rs](../../../../../tools/acceptance/src/host_probes/content.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–238; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–5: use crate and attached body; lines 6–10: use super and attached body; lines 11–158: fn host_content and attached body; lines 159–238: fn check_extension_package and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7dadc7dd4617"></a>

## [tools/acceptance/src/host_probes/deployments.rs](../../../../../tools/acceptance/src/host_probes/deployments.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; lines 1–1: use crate and attached body; lines 2–6: use super and attached body; lines 7–34: fn host_deployments and attached body; lines 35–56: fn operator_tailscale and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-3bd732ee857c"></a>

## [tools/acceptance/src/host_probes/forgejo.rs](../../../../../tools/acceptance/src/host_probes/forgejo.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–72; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–8: use super and attached body; lines 9–29: fn forgejo_origins and attached body; lines 30–72: fn forgejo_advertisement and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4cf71e1e50f5"></a>

## [tools/acceptance/src/host_probes/listeners.rs](../../../../../tools/acceptance/src/host_probes/listeners.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–261; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–8: use super and attached body; lines 9–14: struct Listener and attached body; lines 15–38: fn parse_listeners and attached body; lines 39–62: fn bound and attached body; lines 63–103: fn published and attached body; lines 104–118: fn is_private_bind and attached body; lines 119–164: fn split_origin and attached body; lines 165–180: fn parse_env_file and attached body; lines 181–196: fn host_listeners and attached body; lines 197–261: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d6e38dc69577"></a>

## [tools/acceptance/src/host_probes/mod.rs](../../../../../tools/acceptance/src/host_probes/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–145; lines 1–13: use std and attached body; lines 14–14: use std and attached body; lines 15–16: use crate and attached body; lines 17–18: use crate and attached body; lines 19–20: mod content and attached body; lines 21–21: mod deployments and attached body; lines 22–22: mod forgejo and attached body; lines 23–23: mod listeners and attached body; lines 24–24: mod tailnet and attached body; lines 25–26: use content and attached body; lines 27–27: use deployments and attached body; lines 28–28: use forgejo and attached body; lines 29–29: use listeners and attached body; lines 30–32: use tailnet and attached body; lines 33–36: use listeners and attached body; lines 37–42: enum ProbeFailure and attached body; lines 43–44: impl ProbeFailure and attached body; lines 45–47: fn exit and attached body; lines 48–52: fn failed and attached body; lines 53–54: impl std and attached body; lines 55–61: fn fmt and attached body; lines 62–66: impl std and attached body; lines 67–71: fn dumps and attached body; lines 72–75: fn read_file and attached body; lines 76–80: fn read_text and attached body; lines 81–84: fn parse_json and attached body; lines 85–94: fn get_str and attached body; lines 95–107: fn is_truthy and attached body; lines 108–120: fn run_output and attached body; lines 121–136: fn sha256_file and attached body; lines 137–144: fn podman and attached body; lines 145–145: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 32 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-81ea9de56f2a"></a>

## [tools/acceptance/src/host_probes/tailnet.rs](../../../../../tools/acceptance/src/host_probes/tailnet.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; lines 1–1: use crate and attached body; lines 2–6: use super and attached body; lines 7–25: fn forgejo_tailnet and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn forgejo_tailnet in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cc44f5019969"></a>

## [tools/acceptance/src/host_probes/tests.rs](../../../../../tools/acceptance/src/host_probes/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–119; lines 1–3: use super and attached body; lines 4–22: fn dumps_keeps_order_and_separators and attached body; lines 23–49: fn origins_split_hosts_and_ports and attached body; lines 50–65: fn private_binds_match_plain_lan_shapes and attached body; lines 66–84: fn listeners_parse_and_bind and attached body; lines 85–95: fn env_files_skip_empty_and_reject_bare_words and attached body; lines 96–113: fn tailscale_summaries_shape and attached body; lines 114–119: fn deployments_summarize_in_order and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-311e50e7c9c0"></a>

## [tools/acceptance/src/jsonio.rs](../../../../../tools/acceptance/src/jsonio.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–235; lines 1–6: use std and attached body; lines 7–7: use std and attached body; lines 8–8: use std and attached body; lines 9–10: use serde and attached body; lines 11–11: use serde_json and attached body; lines 12–12: use serde_json and attached body; lines 13–14: use crate and attached body; lines 15–15: use crate and attached body; lines 16–16: use crate and attached body; lines 17–21: use crate and attached body; lines 22–49: fn read_json_file and attached body; lines 50–67: fn read_json_at and attached body; lines 68–69: fn fstat_attr and attached body; lines 70–90: use std and attached body; lines 91–91: struct GoFormatter and attached body; lines 92–93: impl Formatter and attached body; lines 94–113: fn write_string_fragment and attached body; lines 114–115: struct GoPrettyFormatter and attached body; lines 116–117: impl Default and attached body; lines 118–121: fn default and attached body; lines 122–123: impl Formatter and attached body; lines 124–129: fn write_string_fragment and attached body; lines 130–136: fn begin_array and attached body; lines 137–142: fn end_array and attached body; lines 143–148: fn begin_array_value and attached body; lines 149–154: fn end_array_value and attached body; lines 155–160: fn begin_object and attached body; lines 161–166: fn end_object and attached body; lines 167–172: fn begin_object_key and attached body; lines 173–178: fn end_object_key and attached body; lines 179–184: fn begin_object_value and attached body; lines 185–194: fn end_object_value and attached body; lines 195–204: fn write_compact and attached body; lines 205–215: fn write_indent and attached body; lines 216–235: fn escape_go and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 35 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b66cf5f8ed1c"></a>

## [tools/acceptance/src/jsonio/tests.rs](../../../../../tools/acceptance/src/jsonio/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–45; lines 1–1: use super and attached body; lines 2–4: use serde_json and attached body; lines 5–29: fn serde_formatter_matches_go_compact_and_indent and attached body; lines 30–45: fn bounded_reader_rejects_suffix_and_hashes_original_bytes and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ac1355894030"></a>

## [tools/acceptance/src/lib.rs](../../../../../tools/acceptance/src/lib.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–32; lines 1–11: mod cockpit and attached body; lines 12–12: mod command and attached body; lines 13–13: mod coreos and attached body; lines 14–14: mod driver and attached body; lines 15–15: mod error and attached body; lines 16–16: mod evidence and attached body; lines 17–17: mod files and attached body; lines 18–18: mod host_probes and attached body; lines 19–19: mod jsonio and attached body; lines 20–20: mod native_phase and attached body; lines 21–21: mod probe and attached body; lines 22–22: mod process and attached body; lines 23–23: mod project_state and attached body; lines 24–24: mod provisioning and attached body; lines 25–25: mod qmp and attached body; lines 26–26: mod remote and attached body; lines 27–27: mod report and attached body; lines 28–28: mod sha256 and attached body; lines 29–29: mod structured and attached body; lines 30–30: mod timestamps and attached body; lines 31–31: mod trust and attached body; lines 32–32: mod vm and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for mod cockpit in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 22 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-37af525019c4"></a>

## [tools/acceptance/src/main.rs](../../../../../tools/acceptance/src/main.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; lines 1–2: use soda_acceptance and attached body; lines 3–3: use soda_acceptance and attached body; lines 4–13: fn main and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use soda_acceptance in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn main in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7ce9c4bb4790"></a>
<a id="rustsoda-acceptancesrcnative_phasers-1"></a>
<a id="coverage-4b029a61e034"></a>

## [tools/acceptance/src/native_phase.rs](../../../../../tools/acceptance/src/native_phase.rs)

Current module body and direct consumer inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–469; whole file: native qualification phase orchestration and phase-bound evidence collection | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Native qualification phase orchestration and phase-bound evidence collection. — tools/acceptance/src/native_phase.rs:1-469; module purpose and current direct consumer inspected |

<a id="coverage-7a58d8f0c78a"></a>

## [tools/acceptance/src/native_phase/tests.rs](../../../../../tools/acceptance/src/native_phase/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–219; lines 1–1: use super and attached body; lines 2–4: use std and attached body; lines 5–22: fn phase_request_checks_every_duplicate_occurrence_before_last_wins and attached body; lines 23–24: const REVISION and attached body; lines 25–30: struct FakeRunner and attached body; lines 31–32: impl Runner and attached body; lines 33–39: fn status and attached body; lines 40–48: fn output and attached body; lines 49–55: struct Harness and attached body; lines 56–57: impl Harness and attached body; lines 58–78: fn new and attached body; lines 79–92: fn request and attached body; lines 93–96: fn invoke and attached body; lines 97–101: fn calls and attached body; lines 102–103: impl Drop and attached body; lines 104–107: fn drop and attached body; lines 108–109: fn fresh_id and attached body; lines 110–110: use std and attached body; lines 111–113: static NEXT and attached body; lines 114–122: fn seed_candidate and attached body; lines 123–156: fn explicit_phase_order_without_automatic_work and attached body; lines 157–173: fn unknown_phase_target_arch_and_revision_fail_before_creation and attached body; lines 174–189: fn occupied_prepare_and_phase_replay_do_not_run_commands and attached body; lines 190–204: fn missing_prerequisite_stale_binding_and_dirty_source and attached body; lines 205–213: fn build_without_candidate_retains_started_without_completion and attached body; lines 214–219: fn remote_dispatch_has_no_runtime_or_publication_phases and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 26 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-80e1ee60e352"></a>

## [tools/acceptance/src/probe.rs](../../../../../tools/acceptance/src/probe.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–284; lines 1–13: use std and attached body; lines 14–15: use crate and attached body; lines 16–16: use crate and attached body; lines 17–19: use crate and attached body; lines 20–21: const EXCHANGE_TIMEOUT and attached body; lines 22–25: const OUTPUT_LIMIT and attached body; lines 26–44: fn parse_server_host_key and attached body; lines 45–52: fn probe_ssh_key and attached body; lines 53–120: use std and attached body; lines 121–148: fn run_exchange and attached body; lines 149–157: use std and attached body; lines 158–158: mod tests and attached body; lines 159–159: use super and attached body; lines 160–167: fn fixture_dir and attached body; lines 168–179: fn remote_for and attached body; lines 180–205: fn pinned_hosts and attached body; lines 206–233: fn endpoint_and_pin_gates and attached body; lines 234–260: use std and attached body; lines 261–271: fn cancelled_phase_wins and attached body; lines 272–284: fn fingerprint_lines_parse and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 20 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-96ecb97ea8e4"></a>

## [tools/acceptance/src/process/launch.rs](../../../../../tools/acceptance/src/process/launch.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–246; lines 1–3: use std and attached body; lines 4–5: use std and attached body; lines 6–7: use std and attached body; lines 8–8: use std and attached body; lines 9–9: use std and attached body; lines 10–11: use std and attached body; lines 12–12: use std and attached body; lines 13–14: use super and attached body; lines 15–15: use super and attached body; lines 16–16: use crate and attached body; lines 17–17: use crate and attached body; lines 18–23: use crate and attached body; lines 24–24: trait PumpSink and attached body; lines 25–25: fn pump_write and attached body; lines 26–27: fn pump_cancelled and attached body; lines 28–29: impl PumpSink and attached body; lines 30–34: fn pump_write and attached body; lines 35–44: fn pump_cancelled and attached body; lines 45–55: fn start_process and attached body; lines 56–65: fn start_inner and attached body; lines 66–77: fn start_inner and attached body; lines 78–83: fn start_inner_linux and attached body; lines 84–139: use crate and attached body; lines 140–145: fn lock and attached body; lines 146–164: fn spawn_pump and attached body; lines 165–196: fn pump_blocking and attached body; lines 197–233: fn pump_phased and attached body; lines 234–246: fn set_nonblocking and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 28 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-dd26f73e522d"></a>

## [tools/acceptance/src/process/mod.rs](../../../../../tools/acceptance/src/process/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50; lines 1–10: use std and attached body; lines 11–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–15: mod launch and attached body; lines 16–16: mod owned_process and attached body; lines 17–17: mod phase and attached body; lines 18–18: mod raw and attached body; lines 19–20: use self and attached body; lines 21–21: use self and attached body; lines 22–22: use self and attached body; lines 23–25: use self and attached body; lines 26–29: type SharedWriter and attached body; lines 30–37: struct ProcessOutcome and attached body; lines 38–40: impl ProcessOutcome and attached body; lines 41–49: fn combined and attached body; lines 50–50: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f1467b7d2a0f"></a>

## [tools/acceptance/src/process/owned_process.rs](../../../../../tools/acceptance/src/process/owned_process.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–379; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use super and attached body; lines 6–6: use super and attached body; lines 7–7: use crate and attached body; lines 8–14: struct ProcessState and attached body; lines 15–24: struct Process and attached body; lines 25–26: impl Process and attached body; lines 27–43: fn new and attached body; lines 44–81: fn reaper_main and attached body; lines 82–107: fn wait_owned_exit and attached body; lines 108–147: fn reap_leader and attached body; lines 148–186: fn signal_name and attached body; lines 187–203: fn reap_owned_children and attached body; lines 204–205: impl std and attached body; lines 206–211: fn fmt and attached body; lines 212–214: impl Process and attached body; lines 215–219: fn pid and attached body; lines 220–224: fn is_done and attached body; lines 225–229: fn outcome and attached body; lines 230–258: fn wait and attached body; lines 259–275: fn wait_done_timeout and attached body; lines 276–290: fn signal and attached body; lines 291–301: fn stop and attached body; lines 302–338: fn stop_once and attached body; lines 339–343: fn stop_once and attached body; lines 344–362: fn join_pumps and attached body; lines 363–374: fn signal_group and attached body; lines 375–379: fn signal_group and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 30 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4471186f928d"></a>

## [tools/acceptance/src/process/phase.rs](../../../../../tools/acceptance/src/process/phase.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–15: struct PhaseInner and attached body; lines 16–19: struct Phase and attached body; lines 20–21: impl Phase and attached body; lines 22–32: fn new and attached body; lines 33–37: fn background and attached body; lines 38–43: fn timeout and attached body; lines 44–53: fn child and attached body; lines 54–58: fn cancel and attached body; lines 59–64: fn is_cancelled and attached body; lines 65–70: fn expired and attached body; lines 71–81: fn check and attached body; lines 82–85: fn deadline and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-ac7bbb3758e2"></a>

## [tools/acceptance/src/process/raw.rs](../../../../../tools/acceptance/src/process/raw.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–93; lines 1–7: use std and attached body; lines 8–9: use super and attached body; lines 10–10: use super and attached body; lines 11–11: use super and attached body; lines 12–12: use crate and attached body; lines 13–15: use crate and attached body; lines 16–25: struct RawState and attached body; lines 26–28: struct RawCapture and attached body; lines 29–30: impl RawCapture and attached body; lines 31–43: fn new and attached body; lines 44–51: fn take and attached body; lines 52–55: fn cancelled and attached body; lines 56–57: impl PumpSink and attached body; lines 58–66: fn pump_write and attached body; lines 67–77: fn pump_cancelled and attached body; lines 78–93: fn start_raw_process and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4d21b4521efa"></a>

## [tools/acceptance/src/process/tests.rs](../../../../../tools/acceptance/src/process/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–395; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–4: use crate and attached body; lines 5–11: fn discard_pair and attached body; lines 12–24: fn shell_command and attached body; lines 25–33: fn cancellation_before_start and attached body; lines 34–48: fn child_cancel_stays_local_but_parent_flows_down and attached body; lines 49–62: fn child_of_unbounded_parent_has_an_effective_timeout and attached body; lines 63–74: fn child_deadline_never_extends_a_bounded_parent and attached body; lines 75–87: fn owned_process_wait_and_cleanup and attached body; lines 88–106: fn pump_write_failure_is_retained_across_joins and attached body; lines 107–118: fn pump_panics_are_joined_and_cached and attached body; lines 119–119: fn redacting_pumps_stop_at_phase_deadline_with_incomplete_capture and attached body; lines 120–158: use std and attached body; lines 159–182: fn raw_capture_keeps_bounded_bytes_and_discards_stderr and attached body; lines 183–217: fn phased_pumps_exit_on_deadline_without_detaching and attached body; lines 218–222: struct DeadPipe and attached body; lines 223–223: impl std and attached body; lines 224–230: fn read and attached body; lines 231–231: impl std and attached body; lines 232–239: fn as_raw_fd and attached body; lines 240–245: struct ProbeSink and attached body; lines 246–246: impl launch and attached body; lines 247–250: fn pump_write and attached body; lines 251–258: fn pump_cancelled and attached body; lines 259–288: fn phased_pump_setup_failure_fails_without_fallback and attached body; lines 289–326: fn stop_is_safe_at_any_lifecycle_point and attached body; lines 327–395: fn leader_exit_and_resistant_descendants and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 28 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-87f5bf242c56"></a>

## [tools/acceptance/src/project_state/command.rs](../../../../../tools/acceptance/src/project_state/command.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–128; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–8: use super and attached body; lines 9–29: const MAX_OUTPUT and attached body; lines 30–98: fn command_with_timeout and attached body; lines 99–99: fn map_launch and attached body; lines 100–106: use std and attached body; lines 107–116: fn failed_command and attached body; lines 117–122: fn command and attached body; lines 123–128: fn output_lines and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cbb7165f1ca4"></a>

## [tools/acceptance/src/project_state/files.rs](../../../../../tools/acceptance/src/project_state/files.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–148; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–3: use std and attached body; lines 4–5: use crate and attached body; lines 6–7: use crate and attached body; lines 8–12: use super and attached body; lines 13–23: enum EntryBody and attached body; lines 24–29: struct Entry and attached body; lines 30–32: impl Entry and attached body; lines 33–33: fn snapshot and attached body; lines 34–80: use std and attached body; lines 81–97: fn json and attached body; lines 98–102: fn dumps_sorted and attached body; lines 103–115: fn list_files and attached body; lines 116–125: fn has_git_part and attached body; lines 126–148: fn walk_sorted and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7eb472acb8b7"></a>

## [tools/acceptance/src/project_state/mod.rs](../../../../../tools/acceptance/src/project_state/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–233; lines 1–13: use std and attached body; lines 14–14: use std and attached body; lines 15–16: use crate and attached body; lines 17–18: mod command and attached body; lines 19–19: mod files and attached body; lines 20–20: mod snapshot and attached body; lines 21–21: mod workloads and attached body; lines 22–23: use command and attached body; lines 24–24: use files and attached body; lines 25–28: use snapshot and attached body; lines 29–32: use workloads and attached body; lines 33–52: enum SnapshotKind and attached body; lines 53–55: impl SnapshotKind and attached body; lines 56–73: fn name and attached body; lines 74–79: struct SnapshotFailure and attached body; lines 80–81: impl SnapshotFailure and attached body; lines 82–87: fn runtime and attached body; lines 88–94: fn assertion and attached body; lines 95–101: fn bare and attached body; lines 102–103: fn io and attached body; lines 104–114: use std and attached body; lines 115–116: impl std and attached body; lines 117–125: fn fmt and attached body; lines 126–129: impl std and attached body; lines 130–134: fn obj and attached body; lines 135–139: fn s and attached body; lines 140–146: fn n and attached body; lines 147–157: fn set and attached body; lines 158–173: fn sub_mut and attached body; lines 174–199: fn snapshot_ssh_files and attached body; lines 200–218: fn glob_prefix and attached body; lines 219–224: fn arg_list and attached body; lines 225–232: fn check_snapshot_gate and attached body; lines 233–233: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 34 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-197cbcba8c41"></a>

## [tools/acceptance/src/project_state/snapshot.rs](../../../../../tools/acceptance/src/project_state/snapshot.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–196; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–6: use super and attached body; lines 7–7: use super and attached body; lines 8–13: use super and attached body; lines 14–196: fn run_snapshot and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8a7151d0de18"></a>

## [tools/acceptance/src/project_state/tests.rs](../../../../../tools/acceptance/src/project_state/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–380; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–4: use crate and attached body; lines 5–8: use crate and attached body; lines 9–9: fn entry_hashes_file_and_reports_mode and attached body; lines 10–36: use std and attached body; lines 37–47: fn entry_reports_size_without_hashing and attached body; lines 48–56: fn entry_rejects_special_files and attached body; lines 57–77: fn entry_fails_closed_on_links_and_missing_paths and attached body; lines 78–101: fn ssh_files_export_hashes_sizes_and_reject_links and attached body; lines 102–118: fn snapshot_gate_requires_root_container and attached body; lines 119–136: fn snapshot_command_reports_failures and attached body; lines 137–177: fn assert_descendant_retired and attached body; lines 178–207: fn snapshot_command_timeout_retires_group_and_readers and attached body; lines 208–239: fn snapshot_command_escaped_pipes_cancel_at_deadline and attached body; lines 240–270: fn snapshot_command_stderr_only_escape_cancels_at_deadline and attached body; lines 271–299: fn snapshot_command_stdout_only_escape_cancels_at_deadline and attached body; lines 300–321: fn snapshot_command_timeout_with_escaped_pipes_stays_bounded and attached body; lines 322–342: fn snapshot_command_exit_path_retires_inherited_pipes and attached body; lines 343–357: fn dumps_sorted_matches_python_separators and attached body; lines 358–380: fn project_network_check_is_strict and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-044f70405ce5"></a>

## [tools/acceptance/src/project_state/workloads.rs](../../../../../tools/acceptance/src/project_state/workloads.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–174; lines 1–1: use std and attached body; lines 2–3: use crate and attached body; lines 4–11: use super and attached body; lines 12–25: fn check_project_ip and attached body; lines 26–126: fn snapshot_workloads and attached body; lines 127–174: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-9ec170a1c6f3"></a>

## [tools/acceptance/src/provisioning.rs](../../../../../tools/acceptance/src/provisioning.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–152; lines 1–8: use crate and attached body; lines 9–10: use crate and attached body; lines 11–11: use crate and attached body; lines 12–15: use crate and attached body; lines 16–48: fn provisioning_secrets and attached body; lines 49–52: fn single_document_message and attached body; lines 53–84: fn is_single_complete_v3 and attached body; lines 85–90: fn is_private_host_key_path and attached body; lines 91–91: mod tests and attached body; lines 92–92: use super and attached body; lines 93–96: fn write_input and attached body; lines 97–102: use std and attached body; lines 103–126: fn secrets_cover_hashes_and_host_key_material and attached body; lines 127–152: fn single_document_gate and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8aaad60abf07"></a>

## [tools/acceptance/src/qmp.rs](../../../../../tools/acceptance/src/qmp.rs)

Current cohesive body inspected; single defining duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–1046; whole file: qemu machine protocol client used by native guest qualification to negotiate capabilities, issue commands and decode bounded responses | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | QEMU Machine Protocol client used by native guest qualification to negotiate capabilities, issue commands and decode bounded responses. — tools/acceptance/src/qmp.rs; consumed by acceptance VM driver |

<a id="coverage-fee0c60a868a"></a>

## [tools/acceptance/src/remote.rs](../../../../../tools/acceptance/src/remote.rs)

Inherited prior inventory row; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–250; lines 1–10: use serde_json and attached body; lines 11–11: use std and attached body; lines 12–13: use crate and attached body; lines 14–14: use crate and attached body; lines 15–15: use crate and attached body; lines 16–18: use crate and attached body; lines 19–32: struct RemoteRequest and attached body; lines 33–65: fn decode_remote_request and attached body; lines 66–99: fn native_phase and attached body; lines 100–100: mod json_tests and attached body; lines 101–101: use super and attached body; lines 102–104: use serde_json and attached body; lines 105–118: fn native_request_uses_last_exact_string_and_ignores_overwritten_raw_number and attached body; lines 119–119: mod tests and attached body; lines 120–120: use super and attached body; lines 121–121: use std and attached body; lines 122–129: fn fixture_dir and attached body; lines 130–133: fn write_request and attached body; lines 134–157: struct Request and attached body; lines 158–163: use std and attached body; lines 164–189: fn native_request_binding_rejects_mismatch and attached body; lines 190–211: fn remote_unknown_phase_fails_before_ssh and attached body; lines 212–217: fn native_delivery_runs_payload_from_tempfile and attached body; lines 218–250: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use serde_json in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fd846af112f2"></a>

## [tools/acceptance/src/report/handoff.rs](../../../../../tools/acceptance/src/report/handoff.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–220; lines 1–1: use std and attached body; lines 2–3: use serde and attached body; lines 4–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–13: use super and attached body; lines 14–38: fn hashes and attached body; lines 39–49: fn observation_artifacts_valid and attached body; lines 50–72: fn observation_files_match and attached body; lines 73–90: fn split_record and attached body; lines 91–99: fn read_observation and attached body; lines 100–129: fn admit_handoff_observation and attached body; lines 130–142: fn append_missing_owners and attached body; lines 143–145: fn handoff_description and attached body; lines 146–187: struct Description and attached body; lines 188–220: fn handoff and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-42aa4956e00d"></a>

## [tools/acceptance/src/report/mod.rs](../../../../../tools/acceptance/src/report/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–73; lines 1–9: use crate and attached body; lines 10–11: mod handoff and attached body; lines 12–12: mod observation and attached body; lines 13–14: use handoff and attached body; lines 15–18: use observation and attached body; lines 19–25: fn oci_architecture and attached body; lines 26–41: fn require_native and attached body; lines 42–46: fn digest and attached body; lines 47–51: fn revision and attached body; lines 52–72: fn valid_owner and attached body; lines 73–73: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5e8eb2f79c4f"></a>

## [tools/acceptance/src/report/observation.rs](../../../../../tools/acceptance/src/report/observation.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–283; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use serde and attached body; lines 5–5: use serde and attached body; lines 6–6: use serde_json and attached body; lines 7–8: use crate and attached body; lines 9–14: use crate and attached body; lines 15–54: struct Observation and attached body; lines 55–58: struct ObservationSlots and attached body; lines 59–62: fn deserialize and attached body; lines 63–64: struct SlotsVisitor and attached body; lines 65–65: type Value and attached body; lines 66–68: fn expecting and attached body; lines 69–82: fn visit_map and attached body; lines 83–91: fn last and attached body; lines 92–100: fn optional_string and attached body; lines 101–115: fn optional_string_list and attached body; lines 116–130: fn optional_string_map and attached body; lines 131–144: fn optional_timestamp and attached body; lines 145–168: const OBSERVATION_FIELDS and attached body; lines 169–219: fn observation_from_json and attached body; lines 220–259: struct ObservationRecord and attached body; lines 260–283: fn observation_json and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4edc08a153a3"></a>

## [tools/acceptance/src/report/tests.rs](../../../../../tools/acceptance/src/report/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–167; lines 1–1: use std and attached body; lines 2–3: use super and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–8: use crate and attached body; lines 9–29: fn owner_and_shape_gates and attached body; lines 30–73: fn observation_json_round_trip_keeps_nulls and attached body; lines 74–134: fn handoff_preserves_missing_and_failed_scopes and attached body; lines 135–167: fn handoff_rejects_unsafe_references and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5a3da0711f22"></a>

## [tools/acceptance/src/sha256.rs](../../../../../tools/acceptance/src/sha256.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; lines 1–3: use sha2 and attached body; lines 4–9: fn digest and attached body; lines 10–10: fn hex_lower and attached body; lines 11–20: const HEX and attached body; lines 21–21: mod tests and attached body; lines 22–24: use super and attached body; lines 25–55: fn nist_vectors_and_streaming_partitions and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use sha2 in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-c7d991c5768f"></a>

## [tools/acceptance/src/structured.rs](../../../../../tools/acceptance/src/structured.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–355; lines 1–5: use serde and attached body; lines 6–6: use serde and attached body; lines 7–7: use serde and attached body; lines 8–8: use serde_json and attached body; lines 9–12: use serde_json and attached body; lines 13–13: const MAX_CONTAINERS and attached body; lines 14–17: struct RawEntries and attached body; lines 18–18: fn deserialize and attached body; lines 19–20: struct EntriesVisitor and attached body; lines 21–21: type Value and attached body; lines 22–24: fn expecting and attached body; lines 25–35: fn visit_map and attached body; lines 36–39: struct RawItems and attached body; lines 40–40: fn deserialize and attached body; lines 41–42: struct ItemsVisitor and attached body; lines 43–43: type Value and attached body; lines 44–46: fn expecting and attached body; lines 47–61: fn visit_seq and attached body; lines 62–69: enum Value and attached body; lines 70–73: impl Value and attached body; lines 74–80: fn parse and attached body; lines 81–83: fn from_raw and attached body; lines 84–131: fn from_raw_at and attached body; lines 132–141: fn get and attached body; lines 142–148: fn as_str and attached body; lines 149–157: fn as_integer and attached body; lines 158–185: fn validate_depth and attached body; lines 186–187: impl Serialize and attached body; lines 188–192: fn serialize and attached body; lines 193–194: struct NestedValue and attached body; lines 195–196: impl Serialize and attached body; lines 197–200: fn serialize and attached body; lines 201–228: fn serialize_value and attached body; lines 229–229: struct PythonFormatter and attached body; lines 230–231: impl Formatter and attached body; lines 232–256: fn write_string_fragment and attached body; lines 257–267: fn begin_array_value and attached body; lines 268–278: fn begin_object_key and attached body; lines 279–286: fn begin_object_value and attached body; lines 287–288: struct Sorted and attached body; lines 289–290: impl Serialize and attached body; lines 291–314: fn serialize and attached body; lines 315–324: fn write_python and attached body; lines 325–332: fn write_python_sorted and attached body; lines 333–333: mod tests and attached body; lines 334–336: use super and attached body; lines 337–348: fn dynamic_tree_keeps_duplicates_order_and_raw_numbers and attached body; lines 349–355: fn dynamic_tree_uses_pinned_serde_container_depth and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 48 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d3d5cda9d8d3"></a>

## [tools/acceptance/src/timestamps.rs](../../../../../tools/acceptance/src/timestamps.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; lines 1–3: use crate and attached body; lines 4–12: fn now_rfc3339_nano and attached body; lines 13–19: fn validate_rfc3339 and attached body; lines 20–20: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-7b28c5a87442"></a>

## [tools/acceptance/src/timestamps/tests.rs](../../../../../tools/acceptance/src/timestamps/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38; lines 1–3: use super and attached body; lines 4–14: fn current_timestamp_uses_canonical_shared_format and attached body; lines 15–38: fn timestamp_validation_uses_strict_wire_profile and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn current_timestamp_uses_canonical_shared_format in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for fn timestamp_validation_uses_strict_wire_profile in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-5e8c9bdf66be"></a>

## [tools/acceptance/src/trust/host_key.rs](../../../../../tools/acceptance/src/trust/host_key.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; lines 1–1: use crate and attached body; lines 2–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–11: use super and attached body; lines 12–15: struct IgnitionError and attached body; lines 16–19: fn parse_ignition and attached body; lines 20–26: fn is_fixture_trust_file and attached body; lines 27–29: fn trim_space and attached body; lines 30–43: fn run_ssh_keygen and attached body; lines 44–58: fn known_hosts_query and attached body; lines 59–98: fn verify_fixture_trust and attached body; lines 99–103: fn verify_pinned_host_key and attached body; lines 104–124: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 14 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a67ed7b89d29"></a>

## [tools/acceptance/src/trust/inline_data.rs](../../../../../tools/acceptance/src/trust/inline_data.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–164; lines 1–1: use crate and attached body; lines 2–2: use base64 and attached body; lines 3–3: use base64 and attached body; lines 4–4: use base64 and attached body; lines 5–5: use percent_encoding and attached body; lines 6–9: use crate and attached body; lines 10–10: const INLINE_GZIP_LIMIT and attached body; lines 11–14: const INLINE_GZIP_COMPRESSED_LIMIT and attached body; lines 15–28: fn decode_base64 and attached body; lines 29–34: fn encode_base64 and attached body; lines 35–40: fn percent_decode and attached body; lines 41–62: fn valid_percent_escapes and attached body; lines 63–78: fn decode_data_uri and attached body; lines 79–96: fn gunzip_bounded and attached body; lines 97–112: fn inline_data and attached body; lines 113–124: struct IgnitionFile and attached body; lines 125–164: fn decode_ignition_files and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use crate in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 17 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6e5671d8d1e4"></a>

## [tools/acceptance/src/trust/mod.rs](../../../../../tools/acceptance/src/trust/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; lines 1–12: mod host_key and attached body; lines 13–13: mod inline_data and attached body; lines 14–16: use host_key and attached body; lines 17–17: use inline_data and attached body; lines 18–23: use inline_data and attached body; lines 24–24: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for mod host_key in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod inline_data in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; Declaration block for mod tests in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-619d612bc09b"></a>

## [tools/acceptance/src/trust/tests.rs](../../../../../tools/acceptance/src/trust/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–186; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–5: use crate and attached body; lines 6–27: fn base64_round_trip_and_rejections and attached body; lines 28–65: fn data_uri_matrix and attached body; lines 66–77: fn inline_data_compression_gates and attached body; lines 78–110: fn gzip_bounds_match_go and attached body; lines 111–123: fn key_fixture and attached body; lines 124–134: fn trust_fixture and attached body; lines 135–148: use std and attached body; lines 149–162: fn fixture_trust_happy_and_mismatch and attached body; lines 163–168: fn fixture_trust_rejects_shape_violations and attached body; lines 169–186: use std and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-f321737713ad"></a>

## [tools/acceptance/src/vm/base.rs](../../../../../tools/acceptance/src/vm/base.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; lines 1–1: use serde and attached body; lines 2–2: use serde_json and attached body; lines 3–3: use std and attached body; lines 4–5: use super and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–13: use crate and attached body; lines 14–25: struct VerifiedBase and attached body; lines 26–53: fn decode_verified_base and attached body; lines 54–71: fn verify_launch_base_image and attached body; lines 72–88: use std and attached body; lines 89–97: fn publish_launch_fixture and attached body; lines 98–124: struct Fixture and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use serde in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-6433f87757b3"></a>

## [tools/acceptance/src/vm/config.rs](../../../../../tools/acceptance/src/vm/config.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–265; lines 1–1: use std and attached body; lines 2–3: use serde_json and attached body; lines 4–4: use std and attached body; lines 5–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–14: use crate and attached body; lines 15–39: struct VmConfig and attached body; lines 40–51: struct RemoteConfig and attached body; lines 52–53: impl RemoteConfig and attached body; lines 54–66: fn remote and attached body; lines 67–132: fn decode_vm_config and attached body; lines 133–145: fn valid_vm_name and attached body; lines 146–151: fn valid_signer and attached body; lines 152–165: fn validate_vm_config_identity and attached body; lines 166–181: fn rel_path and attached body; lines 182–191: fn disjoint_vm_work and attached body; lines 192–221: fn validate_vm_paths and attached body; lines 222–230: fn validate_vm_ssh_and_trust and attached body; lines 231–234: fn validate_vm_inputs and attached body; lines 235–241: use std and attached body; lines 242–257: fn check_vm_host_tools_and_kvm and attached body; lines 258–265: fn preflight and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 25 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d9afe47a83e8"></a>

## [tools/acceptance/src/vm/launch.rs](../../../../../tools/acceptance/src/vm/launch.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–162; lines 1–1: use serde_json and attached body; lines 2–2: use std and attached body; lines 3–4: use super and attached body; lines 5–5: use super and attached body; lines 6–6: use crate and attached body; lines 7–7: use crate and attached body; lines 8–8: use crate and attached body; lines 9–9: use crate and attached body; lines 10–10: use crate and attached body; lines 11–31: fn verify_qemu_commands and attached body; lines 32–80: fn verify_base_image_format and attached body; lines 81–113: fn prepare_vm_work_directory and attached body; lines 114–117: impl VmConfig and attached body; lines 118–156: fn args and attached body; lines 157–162: struct LaunchFailure and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use serde_json in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 15 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d3012b40f9b4"></a>

## [tools/acceptance/src/vm/lifecycle.rs](../../../../../tools/acceptance/src/vm/lifecycle.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–302; lines 1–1: use std and attached body; lines 2–2: use std and attached body; lines 3–4: use crate and attached body; lines 5–6: use super and attached body; lines 7–7: use super and attached body; lines 8–10: use super and attached body; lines 11–11: use crate and attached body; lines 12–12: use crate and attached body; lines 13–13: use crate and attached body; lines 14–14: use crate and attached body; lines 15–17: use crate and attached body; lines 18–31: struct Vm and attached body; lines 32–84: fn launch_vm and attached body; lines 85–100: fn open_boot_logs and attached body; lines 101–126: fn wait_qemu_ready and attached body; lines 127–160: fn wait_guest_ssh and attached body; lines 161–184: fn start and attached body; lines 185–188: fn restart and attached body; lines 189–217: fn power_down and attached body; lines 218–230: fn close_outputs and attached body; lines 231–246: fn wait and attached body; lines 247–281: fn close_once and attached body; lines 282–302: fn close and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use std in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 23 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-fb6ac2317bc8"></a>

## [tools/acceptance/src/vm/mod.rs](../../../../../tools/acceptance/src/vm/mod.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; lines 1–12: mod base and attached body; lines 13–13: mod config and attached body; lines 14–14: mod launch and attached body; lines 15–15: mod lifecycle and attached body; lines 16–17: use self and attached body; lines 18–18: use self and attached body; lines 19–19: use self and attached body; lines 20–22: use self and attached body; lines 23–23: mod tests and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for mod base in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a05a6c3efc29"></a>

## [tools/acceptance/src/vm/tests.rs](../../../../../tools/acceptance/src/vm/tests.rs)

Current path and release/build/install consumer trace; bounded selector; current Rust declaration/body intervals

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–236; lines 1–1: use super and attached body; lines 2–2: use super and attached body; lines 3–3: use crate and attached body; lines 4–4: use crate and attached body; lines 5–5: use crate and attached body; lines 6–6: use crate and attached body; lines 7–30: fn fixture_config and attached body; lines 31–62: fn vm_arguments_retain_disk_and_native_isolation and attached body; lines 63–75: fn name_and_signer_shapes and attached body; lines 76–84: fn relative_paths_match_go_rel_cases and attached body; lines 85–105: fn vm_config_decode_rejects_unknown_and_mistyped and attached body; lines 106–156: fn preflight_rejects_bad_identity_and_paths and attached body; lines 157–181: fn close_replays_first_outcome and attached body; lines 182–182: fn close_joins_expired_capture_before_closing_and_replays_error and attached body; lines 183–183: use std and attached body; lines 184–184: use std and attached body; lines 185–186: use crate and attached body; lines 187–187: use crate and attached body; lines 188–236: use crate and attached body | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Declaration block for use super in the current Rust source source; all authored source is accounted by these blocks and module imports, attributes, and file-level dispatch context.; 19 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-1d83b9f95dbe"></a>
<a id="rustsoda-acceptancesrcdriverrs-1"></a>

Former source `rust/soda-acceptance/src/driver.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-a39dfb562912"></a>
<a id="rustsoda-acceptancesrchost_probesrs-1"></a>

Former source `rust/soda-acceptance/src/host_probes.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-373953680721"></a>
<a id="rustsoda-acceptancesrcproject_staters-1"></a>

Former source `rust/soda-acceptance/src/project_state.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-51e42a7e881e"></a>
<a id="rustsoda-acceptancesrctrustrs-1"></a>

Former source `rust/soda-acceptance/src/trust.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-42c2f9adc933"></a>
<a id="rustsoda-acceptancesrcvmrs-1"></a>

Former source `rust/soda-acceptance/src/vm.rs`; consult its pinned earlier Git source and the current coverage disposition.

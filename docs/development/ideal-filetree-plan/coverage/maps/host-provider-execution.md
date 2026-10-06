# Host provider execution

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: 4 sections rebound rust→lib and verified clean (tcodex/tfactory/tmuse + ops oracle); factory/mod.rs wiring section added. All rows machine-verified against current bytes.

<a id="coverage-60a9454258b1"></a>

## [lib/host/src/terminal/factory/tcodex.rs](../../../../../lib/host/src/terminal/factory/tcodex.rs)

Re-audit @HEAD: moved `rust/soda-host/src/*.rs` → `lib/host/...` (A00/A01); every row verified declaration-by-declaration against current bytes.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–26 | Fixed supported factory harness and run/pin admission; declarations/fields: `FACTORY_SCOPE_CODEX`, `FACTORY_HARNESS_CODEX`, `FACTORY_SCOPE_MUSE`, `FACTORY_HARNESS_MUSE`, `valid_harness_family`, `ROLE_CODER`, `ROLE_REVIEWER`, `MAX_FACTORY_PROMPT`, `MAX_FACTORY_OUTPUT_READ`, `MAX_FACTORY_OUTPUT_WINDOW`, `MAX_FACTORY_OUTPUT_OFFSET`, `MAX_FACTORY_EXPORT_BUNDLE`, `TAKEOVER_DIR_NAME`, `ERR_FACTORY_EXPORT_CANDIDATE`, `ERR_FACTORY_EXPORT_BOUNDS`, `valid_factory_role`, `valid_preparation_id`, `valid_digest`, `valid_commit`, `valid_factory_run_id`, `valid_harness_version`, `FactoryRun`, `FACTORY_RUN_SPECS`, `validate`, `decode` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 27–28 | Fixed supported factory harness and run/pin admission; declaration/member FACTORY_SCOPE_CODEX; declarations/fields: `FACTORY_SCOPE_CODEX` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 29–30 | Fixed supported factory harness and run/pin admission; declaration/member FACTORY_HARNESS_CODEX; declarations/fields: `FACTORY_HARNESS_CODEX` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 31–33 | Fixed supported factory harness and run/pin admission; declaration/member FACTORY_SCOPE_MUSE; declarations/fields: `FACTORY_SCOPE_MUSE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 34–36 | Fixed supported factory harness and run/pin admission; declaration/member FACTORY_HARNESS_MUSE; declarations/fields: `FACTORY_HARNESS_MUSE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 37–40 | Fixed supported factory harness and run/pin admission; declaration/member valid_harness_family; declarations/fields: `valid_harness_family` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 41 | Fixed supported factory harness and run/pin admission; declaration/member ROLE_CODER; declarations/fields: `ROLE_CODER` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 42–43 | Fixed supported factory harness and run/pin admission; declaration/member ROLE_REVIEWER; declarations/fields: `ROLE_REVIEWER` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 44–45 | Fixed supported factory harness and run/pin admission; declaration/member MAX_FACTORY_PROMPT; declarations/fields: `MAX_FACTORY_PROMPT` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 46–47 | Fixed supported factory harness and run/pin admission; declaration/member MAX_FACTORY_OUTPUT_READ; declarations/fields: `MAX_FACTORY_OUTPUT_READ` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 48–49 | Fixed supported factory harness and run/pin admission; declaration/member MAX_FACTORY_OUTPUT_WINDOW; declarations/fields: `MAX_FACTORY_OUTPUT_WINDOW` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 50–51 | Fixed supported factory harness and run/pin admission; declaration/member MAX_FACTORY_OUTPUT_OFFSET; declarations/fields: `MAX_FACTORY_OUTPUT_OFFSET` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 52–53 | Fixed supported factory harness and run/pin admission; declaration/member MAX_FACTORY_EXPORT_BUNDLE; declarations/fields: `MAX_FACTORY_EXPORT_BUNDLE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 54–56 | Fixed supported factory harness and run/pin admission; declaration/member TAKEOVER_DIR_NAME; declarations/fields: `TAKEOVER_DIR_NAME` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 57–58 | Fixed supported factory harness and run/pin admission; declaration/member ERR_FACTORY_EXPORT_CANDIDATE; declarations/fields: `ERR_FACTORY_EXPORT_CANDIDATE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 59–61 | Fixed supported factory harness and run/pin admission; declaration/member ERR_FACTORY_EXPORT_BOUNDS; declarations/fields: `ERR_FACTORY_EXPORT_BOUNDS` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 62–66 | Fixed supported factory harness and run/pin admission; declaration/member valid_factory_role; declarations/fields: `valid_factory_role` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 67–71 | Fixed supported factory harness and run/pin admission; declaration/member valid_preparation_id; declarations/fields: `valid_preparation_id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 72–76 | Fixed supported factory harness and run/pin admission; declaration/member valid_digest; declarations/fields: `valid_digest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 77–81 | Fixed supported factory harness and run/pin admission; declaration/member valid_commit; declarations/fields: `valid_commit` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 82–86 | Fixed supported factory harness and run/pin admission; declaration/member valid_factory_run_id; declarations/fields: `valid_factory_run_id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 87–98 | Fixed supported factory harness and run/pin admission; declaration/member valid_harness_version; declarations/fields: `valid_harness_version` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 99 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun; declarations/fields: `FactoryRun` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 100 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.deadline_raw; declarations/fields: `FactoryRun.deadline_raw` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 101 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.actor; declarations/fields: `FactoryRun.actor` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 102 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.id; declarations/fields: `FactoryRun.id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 103 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.project; declarations/fields: `FactoryRun.project` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 104 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.role; declarations/fields: `FactoryRun.role` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 105 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.preparation; declarations/fields: `FactoryRun.preparation` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 106 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.harness; declarations/fields: `FactoryRun.harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 107 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.harness_vers; declarations/fields: `FactoryRun.harness_vers` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 108 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.model; declarations/fields: `FactoryRun.model` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 109 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.assignment; declarations/fields: `FactoryRun.assignment` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 110 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.source_commit; declarations/fields: `FactoryRun.source_commit` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 111–113 | Fixed supported factory harness and run/pin admission; declaration/member FactoryRun.connection; declarations/fields: `FactoryRun.connection` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 114–166 | Fixed supported factory harness and run/pin admission; declaration/member FACTORY_RUN_SPECS; declarations/fields: `FACTORY_RUN_SPECS` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 167–203 | Fixed supported factory harness and run/pin admission; declaration/member validate; declarations/fields: `validate` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 204–229 | Fixed supported factory harness and run/pin admission; declaration/member decode; declarations/fields: `decode` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 230–246 | Codex native execution paths and unit argv; declarations/fields: `factory_run_paths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 247–254 | Codex native execution paths and unit argv; declaration/member factory_codex_guest; declarations/fields: `factory_codex_guest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 255–261 | Codex native execution paths and unit argv; declaration/member factory_unit_name; declarations/fields: `factory_unit_name` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 262–266 | Codex native execution paths and unit argv; declaration/member factory_unit_name_or_denied; declarations/fields: `factory_unit_name_or_denied` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 267–274 | Human takeover path predicates; declarations/fields: `takeover_destination` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 275–285 | Human takeover path predicates; declaration/member takeover_source; declarations/fields: `takeover_source` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 286 | Codex execution files/supervisor and reserve scripts; declarations/fields: `FactoryCodexPaths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 287 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.checkout; declarations/fields: `FactoryCodexPaths.checkout` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 288 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.run_dir; declarations/fields: `FactoryCodexPaths.run_dir` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 289 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.home; declarations/fields: `FactoryCodexPaths.home` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 290 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.codex; declarations/fields: `FactoryCodexPaths.codex` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 291 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.prompt; declarations/fields: `FactoryCodexPaths.prompt` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 292 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.marker; declarations/fields: `FactoryCodexPaths.marker` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 293 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.started; declarations/fields: `FactoryCodexPaths.started` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 294 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.stop; declarations/fields: `FactoryCodexPaths.stop` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 295 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.pid_file; declarations/fields: `FactoryCodexPaths.pid_file` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 296 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.output; declarations/fields: `FactoryCodexPaths.output` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 297 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.stdout; declarations/fields: `FactoryCodexPaths.stdout` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 298 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.auth; declarations/fields: `FactoryCodexPaths.auth` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 299–302 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexPaths.guest; declarations/fields: `FactoryCodexPaths.guest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 303–326 | Codex execution files/supervisor and reserve scripts; declaration/member factory_codex_paths; declarations/fields: `factory_codex_paths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 327 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexOutputSlice; declarations/fields: `FactoryCodexOutputSlice` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 328 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexOutputSlice.data; declarations/fields: `FactoryCodexOutputSlice.data` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 329 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexOutputSlice.total; declarations/fields: `FactoryCodexOutputSlice.total` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 330 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexOutputSlice.offset; declarations/fields: `FactoryCodexOutputSlice.offset` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 331 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexOutputSlice.truncated; declarations/fields: `FactoryCodexOutputSlice.truncated` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 332–335 | Codex execution files/supervisor and reserve scripts; declaration/member FactoryCodexOutputSlice.gap; declarations/fields: `FactoryCodexOutputSlice.gap` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 336–340 | Codex execution files/supervisor and reserve scripts; declaration/member systemd_escape; declarations/fields: `systemd_escape` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 341–346 | Codex execution files/supervisor and reserve scripts; declaration/member shell_quote; declarations/fields: `shell_quote` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 347–375 | Codex execution files/supervisor and reserve scripts; declaration/member factory_supervisor; declarations/fields: `factory_supervisor` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 376–407 | Codex native retirement script; declarations/fields: `factory_retire` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 408–439 | Codex native retirement script; declaration/member FACTORY_EXPORT_SCRIPT; declarations/fields: `FACTORY_EXPORT_SCRIPT` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 440–476 | Lease native binding attestation; declarations/fields: `factory_codex_binding` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 477–480 | Factory native user-bus/unit reservation and staging argv; declarations/fields: `factory_user_bus` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 481–485 | Factory native user-bus/unit reservation and staging argv; declaration/member current_euid; declarations/fields: `current_euid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 486–496 | Factory native user-bus/unit reservation and staging argv; declaration/member factory_systemctl_argv; declarations/fields: `factory_systemctl_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 497–507 | Factory native user-bus/unit reservation and staging argv; declaration/member factory_systemd_run_argv; declarations/fields: `factory_systemd_run_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 508–544 | Factory native user-bus/unit reservation and staging argv; declaration/member reserve_exec_argv; declarations/fields: `reserve_exec_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 545–557 | Factory native user-bus/unit reservation and staging argv; declaration/member reserve_run_argv; declarations/fields: `reserve_run_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 558–572 | Factory native user-bus/unit reservation and staging argv; declaration/member codex_setup_script; declarations/fields: `codex_setup_script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 573–583 | Factory native user-bus/unit reservation and staging argv; declaration/member harness_install_script; declarations/fields: `harness_install_script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 584–592 | Factory native user-bus/unit reservation and staging argv; declaration/member stage_file_command; declarations/fields: `stage_file_command` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 593–603 | Factory native user-bus/unit reservation and staging argv; declaration/member start_gate_script; declarations/fields: `start_gate_script` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 604–612 | Bounded factory output read command; declarations/fields: `output_read_command` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 613–652 | Explicit human takeover native copy; declarations/fields: `takeover_steps` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 653–683 | Exact candidate bundle export argv; declarations/fields: `export_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 684 | Native unit observations and role ids; declarations/fields: `FactoryUnitShow` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 685 | Native unit observations and role ids; declaration/member FactoryUnitShow.active; declarations/fields: `FactoryUnitShow.active` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 686–689 | Native unit observations and role ids; declaration/member FactoryUnitShow.invocation; declarations/fields: `FactoryUnitShow.invocation` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 690–703 | Native unit observations and role ids; declaration/member parse_factory_unit_show; declarations/fields: `parse_factory_unit_show` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 704–712 | Native unit observations and role ids; declaration/member factory_role_id; declarations/fields: `factory_role_id` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 713–729 | Native unit observations and role ids; declaration/member sleep_until; declarations/fields: `sleep_until` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 730–734 | Native unit observations and role ids; declaration/member run_env; declarations/fields: `run_env` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 735–745 | Native unit observations and role ids; declaration/member run_podman; declarations/fields: `run_podman` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 746–750 | Native unit observations and role ids; declaration/member factory_systemctl; declarations/fields: `factory_systemctl` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 751–760 | Native unit observations and role ids; declaration/member factory_systemd_run; declarations/fields: `factory_systemd_run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 761–776 | Native unit observations and role ids; declaration/member factory_unit_state; declarations/fields: `factory_unit_state` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 777–800 | Native unit observations and role ids; declaration/member factory_active_invocation; declarations/fields: `factory_active_invocation` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 801–837 | Native unit observations and role ids; declaration/member factory_role_ids; declarations/fields: `factory_role_ids` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 838–893 | Codex native reserve/setup/start integration; declarations/fields: `factory_codex_reserve` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 894–937 | Codex native reserve/setup/start integration; declaration/member factory_codex_setup; declarations/fields: `factory_codex_setup` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 938–994 | Private auth/harness staging; declarations/fields: `factory_codex_stage` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 995 | Host harness staging and native start; declarations/fields: `factory_codex_stage_host` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 996–1047 | Host harness staging and native start; declaration/member HOST_GUEST; declarations/fields: `HOST_GUEST` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1048–1083 | Host harness staging and native start; declaration/member factory_codex_start; declarations/fields: `factory_codex_start` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 1084–1107 | Bound private file stage; declarations/fields: `factory_stage_file` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1108–1129 | Native run completion wait; declarations/fields: `factory_codex_wait` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 1130–1137 | Exact native Codex binding validation; declarations/fields: `factory_codex_validate` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1138–1149 | Stop/retire/capture/finish/unbound cleanup; declarations/fields: `factory_codex_stop` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1150–1171 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_await_inactive; declarations/fields: `factory_await_inactive` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1172–1196 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_retire; declarations/fields: `factory_retire` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1197–1217 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_read_pid; declarations/fields: `factory_read_pid` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1218–1231 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_container_exists; declarations/fields: `factory_container_exists` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1232–1246 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_codex_capture; declarations/fields: `factory_codex_capture` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1247–1257 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_codex_finish; declarations/fields: `factory_codex_finish` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 1258–1274 | Stop/retire/capture/finish/unbound cleanup; declaration/member factory_codex_stop_unbound; declarations/fields: `factory_codex_stop_unbound` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1275–1279 | Native run liveness observation; declarations/fields: `factory_codex_live` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 1280–1299 | Bounded native run output; declarations/fields: `factory_codex_output` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1300–1344 | Exact recorded candidate Git bundle export; declarations/fields: `factory_export_bundle` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1345–1446 | Explicit human takeover copy; declarations/fields: `factory_takeover_copy` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1447–1478 | Broker-native factory callback dispatch; declarations/fields: `factory_identity_operation` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 1479–1488 | Output byte size parsing; declarations/fields: `factory_output_size` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1489–1494, 1856–1865, 1955–1987, 2711–2737, 3029–3090 | Codex native operation source fixtures/assertions; declarations/fields: `tests`, `GOLDEN_SUPERVISOR`, `unit_show_vectors`, `live_matrix`, `factory_identity_operation_matrix` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1495 | Codex native operation source fixtures/assertions; declaration/member PID; declarations/fields: `PID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1496 | Codex native operation source fixtures/assertions; declaration/member RID; declarations/fields: `RID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1497 | Codex native operation source fixtures/assertions; declaration/member IID; declarations/fields: `IID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1498 | Codex native operation source fixtures/assertions; declaration/member CID; declarations/fields: `CID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1499 | Codex native operation source fixtures/assertions; declaration/member PREP; declarations/fields: `PREP` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1500 | Codex native operation source fixtures/assertions; declaration/member ROLE; declarations/fields: `ROLE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1501 | Codex native operation source fixtures/assertions; declaration/member COMMIT; declarations/fields: `COMMIT` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1502–1503 | Codex native operation source fixtures/assertions; declaration/member PIN; declarations/fields: `PIN` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1504–1505 | Codex native operation source fixtures/assertions; declaration/member TMP_COUNTER; declarations/fields: `TMP_COUNTER` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1506–1509 | Codex native operation source fixtures/assertions; declaration/member deadline; declarations/fields: `deadline` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1510–1516 | Codex native operation source fixtures/assertions; declaration/member test_tmp; declarations/fields: `test_tmp` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1517–1518 | Codex native operation source fixtures/assertions; declaration/member RecordedCall; declarations/fields: `RecordedCall` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1519 | Codex native operation source fixtures/assertions; declaration/member FakeExec; declarations/fields: `FakeExec` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1520 | Codex native operation source fixtures/assertions; declaration/member FakeExec.calls; declarations/fields: `FakeExec.calls` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1521–1524 | Codex native operation source fixtures/assertions; declaration/member FakeExec.script; declarations/fields: `FakeExec.script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1525–1531 | Codex native operation source fixtures/assertions; declaration/member new; declarations/fields: `new` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1532–1537 | Codex native operation source fixtures/assertions; declaration/member calls; declarations/fields: `calls` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1538–1557 | Codex native operation source fixtures/assertions; declaration/member run; declarations/fields: `run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1558–1561 | Codex native operation source fixtures/assertions; declaration/member ok; declarations/fields: `ok` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1562–1565 | Codex native operation source fixtures/assertions; declaration/member err; declarations/fields: `err` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1566–1577 | Codex native operation source fixtures/assertions; declaration/member make_service; declarations/fields: `make_service` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1578–1583 | Codex native operation source fixtures/assertions; declaration/member inspect_json; declarations/fields: `inspect_json` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1584–1600 | Codex native operation source fixtures/assertions; declaration/member factory_run; declarations/fields: `factory_run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1601–1604 | Codex native operation source fixtures/assertions; declaration/member run_dir; declarations/fields: `run_dir` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1605–1631 | Codex native operation source fixtures/assertions; declaration/member factory_lease; declarations/fields: `factory_lease` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1632–1643 | Codex native operation source fixtures/assertions; declaration/member write_harness; declarations/fields: `write_harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1644–1650 | Codex native operation source fixtures/assertions; declaration/member host_digest; declarations/fields: `host_digest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1651–1671 | Codex native operation source fixtures/assertions; declaration/member domain_predicates; declarations/fields: `domain_predicates` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1672–1799 | Codex native operation source fixtures/assertions; declaration/member run_validate_pins; declarations/fields: `run_validate_pins` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1800–1846 | Codex native operation source fixtures/assertions; declaration/member path_vectors; declarations/fields: `path_vectors` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1847–1855 | Source assertion of Encoding and parsing; declarations/fields: `quote_vectors` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1866–1892 | Codex native operation source fixtures/assertions; declaration/member GOLDEN_RETIRE; declarations/fields: `GOLDEN_RETIRE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1893–1920 | Codex native operation source fixtures/assertions; declaration/member script_goldens; declarations/fields: `script_goldens` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 1921–1954 | Source assertion of Native binding and private delivery; declarations/fields: `binding_matrix` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1988–1991 | Codex native operation source fixtures/assertions; declaration/member euid; declarations/fields: `euid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1992–1999 | Codex native operation source fixtures/assertions; declaration/member reserve_harness; declarations/fields: `reserve_harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2000–2080 | Codex native operation source fixtures/assertions; declaration/member reserve_denial_pins; declarations/fields: `reserve_denial_pins` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2081–2159 | Codex native operation source fixtures/assertions; declaration/member reserve_success_argv_sequence; declarations/fields: `reserve_success_argv_sequence` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2160–2257 | Codex native operation source fixtures/assertions; declaration/member reserve_stage_and_failure_paths; declarations/fields: `reserve_stage_and_failure_paths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2258–2297 | Codex native operation source fixtures/assertions; declaration/member reserve_unattested_unit_is_stopped; declarations/fields: `reserve_unattested_unit_is_stopped` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2298–2399 | Codex native operation source fixtures/assertions; declaration/member start_flows; declarations/fields: `start_flows` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2400–2463 | Codex native operation source fixtures/assertions; declaration/member wait_flows; declarations/fields: `wait_flows` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2464–2522 | Codex native operation source fixtures/assertions; declaration/member validate_flows; declarations/fields: `validate_flows` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 2523–2611 | Codex native operation source fixtures/assertions; declaration/member stop_flows; declarations/fields: `stop_flows` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 2612–2662 | Source assertion of Completion, revocation and reconciliation; declarations/fields: `capture_and_finish_flows` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 2663–2710 | Source assertion of Completion, revocation and reconciliation; declaration/member stop_unbound_flows; declarations/fields: `stop_unbound_flows` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 2738–2840 | Source assertion of Factory activity presentation; declarations/fields: `output_flows` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 2841–2920 | Source assertion of Publication progression; declarations/fields: `export_flows` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 2921–3028 | Source assertion of Run lifecycle and intervention; declarations/fields: `takeover_flows` |

<a id="coverage-ec73e0d18b70"></a>

## [lib/host/src/terminal/factory/tfactory.rs](../../../../../lib/host/src/terminal/factory/tfactory.rs)

Re-audit @HEAD: moved `rust/soda-host/src/*.rs` → `lib/host/...` (A00/A01); every row verified declaration-by-declaration against current bytes.



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–128 | Shared native provider factory execution paths; declarations/fields: `check_output_range`, `checked_binding_paths`, `factory_wait_result` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 129–155 | Native factory binding attestation; declarations/fields: `factory_attest_live` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 156–242 | Bound/unbound stop and credential capture; declarations/fields: `factory_stop_confirmed`, `factory_stop_unbound_confirmed`, `factory_capture_valid` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 243–266 | Native factory run liveness; declarations/fields: `factory_live_scoped` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 267–380 | Bound output incarnation, size and bounded byte reads; declarations/fields: `factory_output_stdout`, `factory_output_window` |

<a id="coverage-15bd15f9cf85"></a>

## [lib/host/src/terminal/factory/tmuse.rs](../../../../../lib/host/src/terminal/factory/tmuse.rs)

Re-audit @HEAD: moved `rust/soda-host/src/*.rs` → `lib/host/...` (A00/A01); every row verified declaration-by-declaration against current bytes.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–30 | Muse native factory paths, supervisor and allowed argv; declarations/fields: `FactoryMusePaths`, `factory_muse_run_paths`, `factory_muse_paths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 31 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths; declarations/fields: `FactoryMusePaths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 32 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.checkout; declarations/fields: `FactoryMusePaths.checkout` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 33 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.run_dir; declarations/fields: `FactoryMusePaths.run_dir` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 34 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.home; declarations/fields: `FactoryMusePaths.home` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 35 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.muse_config; declarations/fields: `FactoryMusePaths.muse_config` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 36 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.prompt; declarations/fields: `FactoryMusePaths.prompt` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 37 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.marker; declarations/fields: `FactoryMusePaths.marker` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 38 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.started; declarations/fields: `FactoryMusePaths.started` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 39 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.stop; declarations/fields: `FactoryMusePaths.stop` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 40 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.pid_file; declarations/fields: `FactoryMusePaths.pid_file` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 41 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.output; declarations/fields: `FactoryMusePaths.output` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 42 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.stdout; declarations/fields: `FactoryMusePaths.stdout` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 43 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.auth; declarations/fields: `FactoryMusePaths.auth` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 44 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.credential; declarations/fields: `FactoryMusePaths.credential` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 45–48 | Muse native factory paths, supervisor and allowed argv; declaration/member FactoryMusePaths.guest; declarations/fields: `FactoryMusePaths.guest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 49–66 | Muse native factory paths, supervisor and allowed argv; declaration/member factory_muse_run_paths; declarations/fields: `factory_muse_run_paths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 67–97 | Muse native factory paths, supervisor and allowed argv; declaration/member factory_muse_paths; declarations/fields: `factory_muse_paths` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 98–134 | Exact Muse lease binding attestation; declarations/fields: `factory_muse_binding` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 135–149 | Muse native public tool/supervisor setup; declarations/fields: `factory_muse_guest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 150–181 | Muse native public tool/supervisor setup; declaration/member factory_muse_supervisor; declarations/fields: `factory_muse_supervisor` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 182–197 | Muse native public tool/supervisor setup; declaration/member muse_setup_script; declarations/fields: `muse_setup_script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 198–211 | Muse native public tool/supervisor setup; declaration/member muse_start_gate_script; declarations/fields: `muse_start_gate_script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 212–256 | Muse native public tool/supervisor setup; declaration/member muse_exec_argv; declarations/fields: `muse_exec_argv` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 257–276 | Muse native public tool/supervisor setup; declaration/member verify_muse_harness; declarations/fields: `verify_muse_harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 277–332 | Muse native public tool/supervisor setup; declaration/member factory_muse_reserve; declarations/fields: `factory_muse_reserve` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 333–376 | Reserve/setup operation; declarations/fields: `factory_muse_setup` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 377–437 | Private subscription delivery; declarations/fields: `factory_muse_stage` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 438–475 | Native Muse start; declarations/fields: `factory_muse_start` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 476–496 | Native wait; declarations/fields: `factory_muse_wait` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 497–503 | Exact native binding callback; declarations/fields: `factory_muse_validate` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 504–516 | Native stop/unbound retirement/subscription finish; declarations/fields: `factory_muse_stop` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 517–536 | Native stop/unbound retirement/subscription finish; declaration/member factory_muse_stop_unbound; declarations/fields: `factory_muse_stop_unbound` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 537–551 | Native stop/unbound retirement/subscription finish; declaration/member factory_muse_capture; declarations/fields: `factory_muse_capture` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 552–556 | Native liveness; declarations/fields: `factory_muse_live` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 557–577 | Bounded native output; declarations/fields: `factory_muse_output` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 578–583, 800–837, 895–946, 1177–1202 | Muse factory source assertions; declarations/fields: `tests`, `supervisor_shape`, `reserve_denial_pins`, `stop_and_capture` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 584 | Muse factory source assertions; declaration/member PID; declarations/fields: `PID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 585 | Muse factory source assertions; declaration/member RID; declarations/fields: `RID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 586 | Muse factory source assertions; declaration/member IID; declarations/fields: `IID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 587 | Muse factory source assertions; declaration/member CID; declarations/fields: `CID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 588 | Muse factory source assertions; declaration/member PREP; declarations/fields: `PREP` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 589 | Muse factory source assertions; declaration/member ROLE; declarations/fields: `ROLE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 590 | Muse factory source assertions; declaration/member COMMIT; declarations/fields: `COMMIT` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 591–592 | Muse factory source assertions; declaration/member PIN; declarations/fields: `PIN` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 593–594 | Muse factory source assertions; declaration/member TMP_COUNTER; declarations/fields: `TMP_COUNTER` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 595–598 | Muse factory source assertions; declaration/member deadline; declarations/fields: `deadline` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 599–605 | Muse factory source assertions; declaration/member test_tmp; declarations/fields: `test_tmp` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 606–607 | Muse factory source assertions; declaration/member RecordedCall; declarations/fields: `RecordedCall` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 608 | Muse factory source assertions; declaration/member FakeExec; declarations/fields: `FakeExec` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 609 | Muse factory source assertions; declaration/member FakeExec.calls; declarations/fields: `FakeExec.calls` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 610–613 | Muse factory source assertions; declaration/member FakeExec.script; declarations/fields: `FakeExec.script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 614–620 | Muse factory source assertions; declaration/member new; declarations/fields: `new` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 621–626 | Muse factory source assertions; declaration/member calls; declarations/fields: `calls` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 627–646 | Muse factory source assertions; declaration/member run; declarations/fields: `run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 647–650 | Muse factory source assertions; declaration/member ok; declarations/fields: `ok` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 651–654 | Muse factory source assertions; declaration/member err; declarations/fields: `err` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 655–666 | Muse factory source assertions; declaration/member make_service; declarations/fields: `make_service` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 667–674 | Muse factory source assertions; declaration/member write_harness; declarations/fields: `write_harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 675–681 | Muse factory source assertions; declaration/member reserve_harness; declarations/fields: `reserve_harness` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 682–687 | Muse factory source assertions; declaration/member inspect_json; declarations/fields: `inspect_json` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 688–704 | Muse factory source assertions; declaration/member muse_run; declarations/fields: `muse_run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 705–708 | Muse factory source assertions; declaration/member muse_run_dir; declarations/fields: `muse_run_dir` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 709–735 | Muse factory source assertions; declaration/member muse_lease; declarations/fields: `muse_lease` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 736–740 | Muse factory source assertions; declaration/member euid; declarations/fields: `euid` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 741–768 | Muse factory source assertions; declaration/member paths_and_guest; declarations/fields: `paths_and_guest` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 769–799 | Source assertion of Native binding and private delivery; declarations/fields: `binding_gates` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 838–852 | Muse factory source assertions; declaration/member setup_and_gate_scripts; declarations/fields: `setup_and_gate_scripts` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 853–894 | Source assertion of Provider execution integration; declarations/fields: `verify_harness_matrix` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 947–1010 | Muse factory source assertions; declaration/member reserve_success_argv_sequence; declarations/fields: `reserve_success_argv_sequence` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1011–1063 | Muse factory source assertions; declaration/member reserve_stage_missing_guest; declarations/fields: `reserve_stage_missing_guest` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1064–1142 | Muse factory source assertions; declaration/member start_flows; declarations/fields: `start_flows` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 1143–1176 | Source assertion of Factory activity presentation; declarations/fields: `wait_output_live` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1203–1219 | Muse factory source assertions; declaration/member finish_denied_for_muse; declarations/fields: `finish_denied_for_muse` |

<a id="coverage-d3a966f6d3cc"></a>

## [lib/host/src/terminal/factory/mod.rs](../../../../../lib/host/src/terminal/factory/mod.rs)

Re-audit @HEAD: new module wiring from the A00/A01 move (no audit predecessor); 3 re-export declarations.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–3 | Factory provider module wiring; declarations/fields: `tcodex`, `tfactory`, `tmuse` |

## [lib/host/tests/tcodex_ops_oracle.rs](../../../../../lib/host/tests/tcodex_ops_oracle.rs)

Scripted Executor oracle of native command sequences; does not run real containers/systemd/provider execution. Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–56, 523–556 | Scripted Codex native reserve/start/retire/export operation oracle; declarations/fields: `PID`, `RID`, `IID`, `CID`, `PREP`, `ROLE`, `UNIT`, `RUN_DIR`, `deadline`, `ok`, `err`, `inspect_json`, `factory_lease`, `make_service`, `FakeExec`, `new`, `calls`, `run`, `assert_env_systemctl`, `assert_podman`, `fixed_paths_match_go_layout`, `validate_ok_argv_golden`, `validate_denies_each_gate`, `stop_ok_argv_golden`, `stop_is_idempotent_without_container`, `stop_uncertain_matrix`, `finish_runs_capture_only_after_confirmed_stop` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 57 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member PID; declarations/fields: `PID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 58 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member RID; declarations/fields: `RID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 59 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member IID; declarations/fields: `IID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 60 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member CID; declarations/fields: `CID` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 61 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member PREP; declarations/fields: `PREP` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 62 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member ROLE; declarations/fields: `ROLE` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 63 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member UNIT; declarations/fields: `UNIT` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 64–65 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member RUN_DIR; declarations/fields: `RUN_DIR` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 66–69 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member deadline; declarations/fields: `deadline` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 70–73 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member ok; declarations/fields: `ok` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 74–77 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member err; declarations/fields: `err` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 78–83 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member inspect_json; declarations/fields: `inspect_json` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 84–110 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member factory_lease; declarations/fields: `factory_lease` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 111–122 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member make_service; declarations/fields: `make_service` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 123–124 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member RecordedCall; declarations/fields: `RecordedCall` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 125 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member FakeExec; declarations/fields: `FakeExec` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 126 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member FakeExec.calls; declarations/fields: `FakeExec.calls` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 127–130 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member FakeExec.script; declarations/fields: `FakeExec.script` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 131–137 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member new; declarations/fields: `new` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 138–143 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member calls; declarations/fields: `calls` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 144–165 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member run; declarations/fields: `run` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 166–179 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member assert_env_systemctl; declarations/fields: `assert_env_systemctl` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 180–189 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member assert_podman; declarations/fields: `assert_podman` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 190–205 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member fixed_paths_match_go_layout; declarations/fields: `fixed_paths_match_go_layout` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 206–247 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member validate_ok_argv_golden; declarations/fields: `validate_ok_argv_golden` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 248–352 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member validate_denies_each_gate; declarations/fields: `validate_denies_each_gate` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 353–403 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member stop_ok_argv_golden; declarations/fields: `stop_ok_argv_golden` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 404–425 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member stop_is_idempotent_without_container; declarations/fields: `stop_is_idempotent_without_container` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 426–491 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member stop_uncertain_matrix; declarations/fields: `stop_uncertain_matrix` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 492–522 | Source assertion of Encrypted credential custody; declarations/fields: `finish_ok_returns_credential_bytes` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 557–609 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member unit_show_parse_oracle; declarations/fields: `unit_show_parse_oracle` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 610–657 | Scripted Codex native reserve/start/retire/export operation oracle; declaration/member adapter_dispatch_calls_each_callback; declarations/fields: `adapter_dispatch_calls_each_callback` |


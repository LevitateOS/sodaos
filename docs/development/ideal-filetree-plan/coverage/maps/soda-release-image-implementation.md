# Soda release image implementation

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 re-audit COMPLETE @HEAD: build.rs split re-mapped (build+compile+runner+tests+source, doubles in place); foreign/host/layout/media/model/payload/prepare/request/sys verified; CORR-C-001/C-004 + C09-paths tests rowed. GAP (pre-existing): build_media.rs/complete.rs never interval-mapped; leaf inventory covers them.

<a id="coverage-0387b9fa196a"></a>

<a id="rustsoda-release-imagesrcbuildrs-1"></a>

## [lib/soda-release-image/src/build.rs](../../../../../lib/soda-release-image/src/build.rs)

Re-audit @HEAD: pre-C08 `build.rs` (1447 lines) split into `build.rs` + `build_compile.rs` + `build_runner.rs` + `build_runner/tests.rs` + `build_source.rs`; rows re-mapped declaration-by-declaration to current bytes. Test-local `RunnerProduction`/`Stub` doubles re-mapped in place. `build_media.rs`/`complete.rs` content was never interval-mapped (pre-existing audit gap; leaf inventory covers them).

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–28 | Native candidate build runner/cancellation/logging; declarations/fields: `PINNED_GO_VERSION`, `ProductionInputs`, `ProductionInputs.source`, `ProductionInputs.forgejo_source`, `ProductionInputs.forgejo_revision`, `ProductionInputs.native`, `ProductionInputs.out`, `ProductionInputs.arch`, `ProductionInputs.revision`, `ProductionInputs.live_inputs`, `build`, `init_build_directories`, `extract_build_snapshot`, `setup_build_workspace`, `freeze_base_image_config`, `prepare_build_host_context`, `prepare_build_production`, `build_host_candidate`, `execute_build_production`, `finalize_build`, `run_build`, `run_build_inner`, `RunnerProduction`, `RunnerProduction.runner`, `RunnerProduction.inputs`, `source`, `forgejo_source`, `forgejo_revision`, `native`, `out`, `arch`, `revision`, `live_inputs`, `execute`, `capture`, `next`, `resolve_inputs`, `dependencies`, `compile`, `compile_rust`, `stage_fork_binary`, `assets`, `images`, `inspect_oci`, `verify_content`, `resolve_core_os`, `read_live_inputs`, `check_native`, `sign_media`, `verify_copy`, `write_document`, `build_host_candidate_with_progress`, `link_prepared_assets`, `tests`, `oracle_link_prepared_assets_runs_no_commands`, `Stub`, `Stub.source`, `Stub.native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 33 | Native candidate build runner/cancellation/logging; declaration/member PINNED_GO_VERSION; declarations/fields: `PINNED_GO_VERSION` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 36–46 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs; declarations/fields: `ProductionInputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 38 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.source; declarations/fields: `ProductionInputs.source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 39 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.forgejo_source; declarations/fields: `ProductionInputs.forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 40 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.forgejo_revision; declarations/fields: `ProductionInputs.forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 41 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.native; declarations/fields: `ProductionInputs.native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 42 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.out; declarations/fields: `ProductionInputs.out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 43 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.arch; declarations/fields: `ProductionInputs.arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 44 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.revision; declarations/fields: `ProductionInputs.revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 45 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.live_inputs; declarations/fields: `ProductionInputs.live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 51–69 | Native candidate build runner/cancellation/logging; declaration/member build; declarations/fields: `build` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 71–76 | Fresh build workspace and native snapshot extraction; declarations/fields: `init_build_directories` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 78–109 | Fresh build workspace and native snapshot extraction; declaration/member extract_build_snapshot; declarations/fields: `extract_build_snapshot` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 111–131 | Fresh build workspace and native snapshot extraction; declaration/member setup_build_workspace; declarations/fields: `setup_build_workspace` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 133–203 | Frozen native base image identity acquisition; declarations/fields: `freeze_base_image_config` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 205–240 | Prepare/compile/stage shipping tools and candidate image; declarations/fields: `prepare_build_host_context` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 245–274 | Prepare/compile/stage shipping tools and candidate image; declaration/member prepare_build_production; declarations/fields: `prepare_build_production` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 280–339 | Prepare/compile/stage shipping tools and candidate image; declaration/member build_host_candidate; declarations/fields: `build_host_candidate` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 341–375 | Prepare/compile/stage shipping tools and candidate image; declaration/member execute_build_production; declarations/fields: `execute_build_production` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 377–393 | Prepare/compile/stage shipping tools and candidate image; declaration/member finalize_build; declarations/fields: `finalize_build` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 395–420 | Prepare/compile/stage shipping tools and candidate image; declaration/member run_build; declarations/fields: `run_build` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 422–626 | Prepare/compile/stage shipping tools and candidate image; declaration/member run_build_inner; declarations/fields: `run_build_inner` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 434–438 | Real native production operation adapter; declarations/fields: `RunnerProduction` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 435 | Real native production operation adapter; declaration/member RunnerProduction.runner; declarations/fields: `RunnerProduction.runner` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 436 | Real native production operation adapter; declaration/member RunnerProduction.inputs; declarations/fields: `RunnerProduction.inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 439–441 | Real native production operation adapter; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 442–444 | Real native production operation adapter; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 445–447 | Real native production operation adapter; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 448–450 | Real native production operation adapter; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 451–453 | Real native production operation adapter; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 454–456 | Real native production operation adapter; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 457–459 | Real native production operation adapter; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 460–462 | Real native production operation adapter; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 463–465 | Real native production operation adapter; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 466–468 | Real native production operation adapter; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 469–471 | Real native production operation adapter; declaration/member next; declarations/fields: `next` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 472–474 | Pinned live-input resolution adapter; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 475–477 | Native dependencies/compile/fork/assets/images adapter; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 478–480 | Native dependencies/compile/fork/assets/images adapter; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 481–483 | Native dependencies/compile/fork/assets/images adapter; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 484–486 | Native dependencies/compile/fork/assets/images adapter; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 487–489 | Native dependencies/compile/fork/assets/images adapter; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 490–492 | Native dependencies/compile/fork/assets/images adapter; declaration/member images; declarations/fields: `images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 493–495 | OCI/content verification adapter; declarations/fields: `inspect_oci` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 496 | OCI/content verification adapter; declaration/member verify_content; declarations/fields: `verify_content` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 503–505 | Pinned CoreOS/live inputs adapter; declarations/fields: `resolve_core_os` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 506–508 | Pinned CoreOS/live inputs adapter; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 509–511 | Native candidate check adapter; declarations/fields: `check_native` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 512 | Media signing adapter; declarations/fields: `sign_media` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 524 | Verified external copy adapter; declarations/fields: `verify_copy` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 534–626 | Immutable document output adapter; declarations/fields: `write_document` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 628–686, 693–710 | Candidate/media orchestration and native build environment; declarations/fields: `build_host_candidate_with_progress`, `build_environment_pairs`, `resolve_build_tool`, `look_path`, `run_build_command`, `link_prepared_assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 628–686 | Candidate/media orchestration and native build environment; declaration/member build_host_candidate_with_progress; declarations/fields: `build_host_candidate_with_progress` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 693–710 | Candidate/media orchestration and native build environment; declaration/member link_prepared_assets; declarations/fields: `link_prepared_assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 712–836 | Native build source fixtures/assertions; declarations/fields: `tests` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 716–717 | Native build source fixtures/assertions; declaration/member oracle_link_prepared_assets_runs_no_commands; declarations/fields: `oracle_link_prepared_assets_runs_no_commands` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 719–723 | Native build source fixtures/assertions; declaration/member Stub; declarations/fields: `Stub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 720 | Native build source fixtures/assertions; declaration/member Stub.source; declarations/fields: `Stub.source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 721 | Native build source fixtures/assertions; declaration/member Stub.native; declarations/fields: `Stub.native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 724–726 | Native build source fixtures/assertions; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 727–729 | Native build source fixtures/assertions; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 730–732 | Native build source fixtures/assertions; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 733–735 | Native build source fixtures/assertions; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 736–738 | Native build source fixtures/assertions; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 739–741 | Native build source fixtures/assertions; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 742–744 | Native build source fixtures/assertions; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 745–747 | Native build source fixtures/assertions; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 748–750 | Native build source fixtures/assertions; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 751–753 | Native build source fixtures/assertions; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 754–756 | Native build source fixtures/assertions; declaration/member next; declarations/fields: `next` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 757–759 | Native build source fixtures/assertions; declaration/member resolve_inputs; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 760–762 | Native build source fixtures/assertions; declaration/member dependencies; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 763–765 | Native build source fixtures/assertions; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 766–768 | Native build source fixtures/assertions; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 769–771 | Native build source fixtures/assertions; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 772–774 | Native build source fixtures/assertions; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 775–777 | Native build source fixtures/assertions; declaration/member images; declarations/fields: `images` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 778–780 | Native build source fixtures/assertions; declaration/member inspect_oci; declarations/fields: `inspect_oci` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 781 | Native build source fixtures/assertions; declaration/member verify_content; declarations/fields: `verify_content` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 788–790 | Native build source fixtures/assertions; declaration/member resolve_core_os; declarations/fields: `resolve_core_os` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 791–793 | Native build source fixtures/assertions; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 794–796 | Native build source fixtures/assertions; declaration/member check_native; declarations/fields: `check_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 797 | Native build source fixtures/assertions; declaration/member sign_media; declarations/fields: `sign_media` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 809 | Native build source fixtures/assertions; declaration/member verify_copy; declarations/fields: `verify_copy` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 819–836 | Native build source fixtures/assertions; declaration/member write_document; declarations/fields: `write_document` |

## [lib/soda-release-image/src/build_compile.rs](../../../../../lib/soda-release-image/src/build_compile.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–8 | Native candidate build runner/cancellation/logging; declarations/fields: `compile_soda_commands`, `record_tool_files`, `RUST_TOOLS`, `compile_rust_tools`, `compile_shipping_tools`, `shipping_tools_use_rust_recipes_and_ship_remote_companion`, `soda_commands_follow_actual_go_rust_owners`, `cmd1_compile_skips_terminal_identity_and_tools_ship_it_once` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 10–51 | Prepare/compile/stage shipping tools and candidate image; declaration/member compile_soda_commands; declarations/fields: `compile_soda_commands` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 53–102 | Prepare/compile/stage shipping tools and candidate image; declaration/member record_tool_files; declarations/fields: `record_tool_files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 107–158 | Prepare/compile/stage shipping tools and candidate image; declaration/member RUST_TOOLS; declarations/fields: `RUST_TOOLS` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 160–167 | Prepare/compile/stage shipping tools and candidate image; declaration/member compile_rust_tools; declarations/fields: `compile_rust_tools` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 169–212 | Prepare/compile/stage shipping tools and candidate image; declaration/member compile_shipping_tools; declarations/fields: `compile_shipping_tools` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 335–385 | Prepare/compile/stage shipping tools and candidate image; declaration/member shipping_tools_use_rust_recipes_and_ship_remote_companion; declarations/fields: `shipping_tools_use_rust_recipes_and_ship_remote_companion` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 387–431 | Prepare/compile/stage shipping tools and candidate image; declaration/member soda_commands_follow_actual_go_rust_owners; declarations/fields: `soda_commands_follow_actual_go_rust_owners` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 433–490 | Prepare/compile/stage shipping tools and candidate image; declaration/member cmd1_compile_skips_terminal_identity_and_tools_ship_it_once; declarations/fields: `cmd1_compile_skips_terminal_identity_and_tools_ship_it_once` |

## [lib/soda-release-image/src/build_runner.rs](../../../../../lib/soda-release-image/src/build_runner.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–12 | Native candidate build runner/cancellation/logging; declarations/fields: `Cancel`, `Cancel.flag`, `new`, `cancel`, `is_cancelled`, `SharedFile`, `wrap`, `write`, `flush`, `Runner`, `Runner.cancel`, `Runner.log`, `execute`, `capture`, `open_log`, `reason`, `LogCloser`, `LogCloser.files`, `close`, `build_environment_pairs`, `resolve_build_tool`, `look_path`, `run_build_command`, `drain_pipe` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 15–18 | Native candidate build runner/cancellation/logging; declaration/member Cancel; declarations/fields: `Cancel` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 17 | Native candidate build runner/cancellation/logging; declaration/member Cancel.flag; declarations/fields: `Cancel.flag` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 21–23 | Native candidate build runner/cancellation/logging; declaration/member new; declarations/fields: `new` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 25–27 | Native candidate build runner/cancellation/logging; declaration/member cancel; declarations/fields: `cancel` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 29–32 | Native candidate build runner/cancellation/logging; declaration/member is_cancelled; declarations/fields: `is_cancelled` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 36–37 | Native candidate build runner/cancellation/logging; declaration/member SharedFile; declarations/fields: `SharedFile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 40–43 | Native candidate build runner/cancellation/logging; declaration/member wrap; declarations/fields: `wrap` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 46–48 | Native candidate build runner/cancellation/logging; declaration/member write; declarations/fields: `write` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 50–53 | Native candidate build runner/cancellation/logging; declaration/member flush; declarations/fields: `flush` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 57–61 | Native candidate build runner/cancellation/logging; declaration/member Runner; declarations/fields: `Runner` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 59 | Native candidate build runner/cancellation/logging; declaration/member Runner.cancel; declarations/fields: `Runner.cancel` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 60 | Native candidate build runner/cancellation/logging; declaration/member Runner.log; declarations/fields: `Runner.log` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 71–77 | Native candidate build runner/cancellation/logging; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 79–82 | Native candidate build runner/cancellation/logging; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 84–127 | Native candidate build runner/cancellation/logging; declaration/member open_log; declarations/fields: `open_log` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 129–132 | Native candidate build runner/cancellation/logging; declaration/member reason; declarations/fields: `reason` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 135–137 | Native candidate build runner/cancellation/logging; declaration/member LogCloser; declarations/fields: `LogCloser` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 136 | Native candidate build runner/cancellation/logging; declaration/member LogCloser.files; declarations/fields: `LogCloser.files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 140–147 | Native candidate build runner/cancellation/logging; declaration/member close; declarations/fields: `close` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 152–178, 184–191, 193–205, 207–279 | Candidate/media orchestration and native build environment; declarations/fields: `build_host_candidate_with_progress`, `build_environment_pairs`, `resolve_build_tool`, `look_path`, `run_build_command`, `link_prepared_assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 152–178 | Candidate/media orchestration and native build environment; declaration/member build_environment_pairs; declarations/fields: `build_environment_pairs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 184–191 | Candidate/media orchestration and native build environment; declaration/member resolve_build_tool; declarations/fields: `resolve_build_tool` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 193–205 | Candidate/media orchestration and native build environment; declaration/member look_path; declarations/fields: `look_path` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 207–279 | Candidate/media orchestration and native build environment; declaration/member run_build_command; declarations/fields: `run_build_command` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 281–291 | Candidate/media orchestration and native build environment; declaration/member drain_pipe; declarations/fields: `drain_pipe` |

## [lib/soda-release-image/src/build_runner/tests.rs](../../../../../lib/soda-release-image/src/build_runner/tests.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1 | Native candidate build runner/cancellation/logging; declarations/fields: `oracle_build_command_capture_environment_and_failure`, `run_build_command_drains_saturated_pipes`, `run_build_command_cancel_kills_and_reports` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 3–58 | Native build source fixtures/assertions; declaration/member oracle_build_command_capture_environment_and_failure; declarations/fields: `oracle_build_command_capture_environment_and_failure` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 60–80 | Native build source fixtures/assertions; declaration/member run_build_command_drains_saturated_pipes; declarations/fields: `run_build_command_drains_saturated_pipes` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 82–104 | Native build source fixtures/assertions; declaration/member run_build_command_cancel_kills_and_reports; declarations/fields: `run_build_command_cancel_kills_and_reports` |

## [lib/soda-release-image/src/build_source.rs](../../../../../lib/soda-release-image/src/build_source.rs)

Re-audit @HEAD: split sibling; spans re-audited against current bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–9 | Native candidate build runner/cancellation/logging; declarations/fields: `verify_checkout_source`, `verify_committed_revision`, `verify_compiler`, `admit_build_output`, `admit_build_inputs` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 11–33 | Clean exact checkout and source/compiler/output admission; declarations/fields: `verify_checkout_source` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 35–71 | Clean exact checkout and source/compiler/output admission; declaration/member verify_committed_revision; declarations/fields: `verify_committed_revision` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 73–81 | Clean exact checkout and source/compiler/output admission; declaration/member verify_compiler; declarations/fields: `verify_compiler` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 83–99 | Clean exact checkout and source/compiler/output admission; declaration/member admit_build_output; declarations/fields: `admit_build_output` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 101–121 | Clean exact checkout and source/compiler/output admission; declaration/member admit_build_inputs; declarations/fields: `admit_build_inputs` |

## [lib/soda-release-image/src/foreign.rs](../../../../../lib/soda-release-image/src/foreign.rs)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–37, 46–47, 94–101 | Complete native candidate image assembly and authenticated live media; declarations/fields: `Production`, `source`, `forgejo_source`, `forgejo_revision`, `native`, `out`, `arch`, `revision`, `live_inputs`, `execute`, `capture`, `next`, `stage_fork_binary`, `Progress`, `phase`, `end_phase`, `end`, `create_log`, `note_reason` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 38–39 | Pinned input acquisition adapter: resolve_inputs; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 40–41 | Candidate production adapter: dependencies; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 42–43 | Candidate production adapter: compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 44–45 | Candidate production adapter: compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 48–49 | Candidate production adapter: assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 50–53 | Candidate production adapter: images; declarations/fields: `images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 54–56 | Artifact verification adapter: inspect_oci; declarations/fields: `inspect_oci` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 57–62 | Artifact verification adapter: verify_content; declarations/fields: `verify_content` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 63–64 | Pinned input acquisition adapter: resolve_core_os; declarations/fields: `resolve_core_os` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 65–67 | Pinned input acquisition adapter: read_live_inputs; declarations/fields: `read_live_inputs` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 68–69 | Artifact verification adapter: check_native; declarations/fields: `check_native` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 70–80 | Signing custody adapter: sign_media; declarations/fields: `sign_media` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 81–89 | Verified distribution consumption adapter: verify_copy; declarations/fields: `verify_copy` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 90–93 | Release admission and preparation adapter: write_document; declarations/fields: `write_document` |

<a id="coverage-6ec9c0721258"></a>

## [lib/soda-release-image/src/host.rs](../../../../../lib/soda-release-image/src/host.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–196 | Native host image architecture/package provenance; declarations/fields: `read_host_image_id`, `inspect_host_identity`, `observe_host_packages`, `record_host_packages`, `export_host_archive`, `build_host_image`, `SCOPE` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 197–268 | Immutable produced host image readback; declarations/fields: `verify_built_host`, `SCOPE`, `tests`, `oracle_host_image_id_shape` |

<a id="coverage-9f6c9a85cf9b"></a>

## [lib/soda-release-image/src/layout.rs](../../../../../lib/soda-release-image/src/layout.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–75 | Native candidate shared OCI image layout assembly; declarations/fields: `valid_stage_layout`, `verify_archive_digests`, `copy_staged_archives`, `chmod_staged_files` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 76–110 | Verify generated layout immutable identities; declarations/fields: `stage_images`, `tests`, `oracle_layout_staging_refuses_before_copies` |

<a id="coverage-6104704208c3"></a>

<a id="rustsoda-release-imagesrcmediars-1"></a>

## [lib/soda-release-image/src/media.rs](../../../../../lib/soda-release-image/src/media.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; no drift.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1–19 | Authenticated media sources, inputs and assembly output contract; declarations/fields: `MediaLock`, `to_json`, `MediaAuthority`, `parse`, `Media` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 20–21 | Authenticated media sources, inputs and assembly output contract; declaration/member RunFn; declarations/fields: `RunFn` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 22–24 | Authenticated media sources, inputs and assembly output contract; declaration/member NativeFn; declarations/fields: `NativeFn` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 25 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaLock; declarations/fields: `MediaLock` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 26 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaLock.assembler; declarations/fields: `MediaLock.assembler` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 27 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaLock.config; declarations/fields: `MediaLock.config` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 28 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaLock.architecture; declarations/fields: `MediaLock.architecture` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 29–32 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaLock.installer; declarations/fields: `MediaLock.installer` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 33–56, 95–144 | Authenticated media sources, inputs and assembly output contract; declaration/member to_json; declarations/fields: `to_json` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 57 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaAuthority; declarations/fields: `MediaAuthority` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 58 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaAuthority.trust; declarations/fields: `MediaAuthority.trust` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 59–62 | Authenticated media sources, inputs and assembly output contract; declaration/member MediaAuthority.keys; declarations/fields: `MediaAuthority.keys` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 63–76 | Authenticated media sources, inputs and assembly output contract; declaration/member parse; declarations/fields: `parse` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 77 | Authenticated media sources, inputs and assembly output contract; declaration/member Media; declarations/fields: `Media` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 78 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.scope; declarations/fields: `Media.scope` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 79 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.revision; declarations/fields: `Media.revision` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 80 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.architecture; declarations/fields: `Media.architecture` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 81 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.host_manifest; declarations/fields: `Media.host_manifest` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 82 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.payload_sha256; declarations/fields: `Media.payload_sha256` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 83 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.console_sha256; declarations/fields: `Media.console_sha256` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 84 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.assembler_import_commit; declarations/fields: `Media.assembler_import_commit` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 85 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.rootfs_url; declarations/fields: `Media.rootfs_url` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 86 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.iso; declarations/fields: `Media.iso` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 87 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.rootfs; declarations/fields: `Media.rootfs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 88 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.tools; declarations/fields: `Media.tools` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 89 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.compression_mode; declarations/fields: `Media.compression_mode` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 90 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.rootfs_filesystem; declarations/fields: `Media.rootfs_filesystem` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 91–94 | Authenticated media sources, inputs and assembly output contract; declaration/member Media.rootfs_options; declarations/fields: `Media.rootfs_options` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 145–183 | Pinned native media assembler/ISO input acquisition; declarations/fields: `media_base_url` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 184 | Pinned native media assembler/ISO input acquisition; declaration/member ASSEMBLER_IMAGE; declarations/fields: `ASSEMBLER_IMAGE` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 185–186 | Pinned native media assembler/ISO input acquisition; declaration/member ASSEMBLER_CONFIG_BRANCH; declarations/fields: `ASSEMBLER_CONFIG_BRANCH` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 187–245 | Pinned native media assembler/ISO input acquisition; declaration/member fetch_assembler_config; declarations/fields: `fetch_assembler_config` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 246–266 | Pinned native media assembler/ISO input acquisition; declaration/member pin_assembler_build_args; declarations/fields: `pin_assembler_build_args` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 267–302 | Pinned native media assembler/ISO input acquisition; declaration/member verify_assembler_layers; declarations/fields: `verify_assembler_layers` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 303–329 | Pinned native media assembler/ISO input acquisition; declaration/member wrap_assembler_image; declarations/fields: `wrap_assembler_image` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 330–368 | Pinned native media assembler/ISO input acquisition; declaration/member PREFIX; declarations/fields: `PREFIX` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 369–384 | Pinned native media assembler/ISO input acquisition; declaration/member prepare_assembler; declarations/fields: `prepare_assembler` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 385–398 | Pinned native media assembler/ISO input acquisition; declaration/member builder_id; declarations/fields: `builder_id` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 399–439 | Native exact digest media signing; declarations/fields: `sign_media_input` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 440 | Native exact digest media signing; declaration/member MediaInputs; declarations/fields: `MediaInputs` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 441 | Native exact digest media signing; declaration/member MediaInputs.authority; declarations/fields: `MediaInputs.authority` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 442 | Native exact digest media signing; declaration/member MediaInputs.trust; declarations/fields: `MediaInputs.trust` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 443 | Native exact digest media signing; declaration/member MediaInputs.candidate; declarations/fields: `MediaInputs.candidate` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 444 | Native exact digest media signing; declaration/member MediaInputs.payload; declarations/fields: `MediaInputs.payload` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 445 | Native exact digest media signing; declaration/member MediaInputs.observed; declarations/fields: `MediaInputs.observed` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 446 | Native exact digest media signing; declaration/member MediaInputs.archive; declarations/fields: `MediaInputs.archive` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 447 | Native exact digest media signing; declaration/member MediaInputs.filesystem; declarations/fields: `MediaInputs.filesystem` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 448–450 | Native exact digest media signing; declaration/member MediaInputs.fsoptions; declarations/fields: `MediaInputs.fsoptions` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 451–466 | Bound authenticated media preparation/assembly; declarations/fields: `admit_media_authority` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 467–493 | Bound authenticated media preparation/assembly; declaration/member admit_media_candidate; declarations/fields: `admit_media_candidate` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 494–508 | Bound authenticated media preparation/assembly; declaration/member admit_media_compression; declarations/fields: `admit_media_compression` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 509–532 | Bound authenticated media preparation/assembly; declaration/member admit_media_inputs; declarations/fields: `admit_media_inputs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 533–568 | Bound authenticated media preparation/assembly; declaration/member collect_media_inventory; declarations/fields: `collect_media_inventory` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 569–584 | Bound authenticated media preparation/assembly; declaration/member verify_media_inventory; declarations/fields: `verify_media_inventory` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 585–639 | Bound authenticated media preparation/assembly; declaration/member authenticate_packaging_inputs; declarations/fields: `authenticate_packaging_inputs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 640–673 | Bound authenticated media preparation/assembly; declaration/member stop_packaging_container; declarations/fields: `stop_packaging_container` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 674–723 | Bound authenticated media preparation/assembly; declaration/member build_media_container; declarations/fields: `build_media_container` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 724–740 | Bound authenticated media preparation/assembly; declaration/member assemble_native_media; declarations/fields: `assemble_native_media` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 741 | Bound authenticated media preparation/assembly; declaration/member MediaMetaImage; declarations/fields: `MediaMetaImage` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 742 | Bound authenticated media preparation/assembly; declaration/member MediaMetaImage.path; declarations/fields: `MediaMetaImage.path` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 743 | Bound authenticated media preparation/assembly; declaration/member MediaMetaImage.sha256; declarations/fields: `MediaMetaImage.sha256` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 744–747 | Bound authenticated media preparation/assembly; declaration/member MediaMetaImage.size; declarations/fields: `MediaMetaImage.size` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 748 | Bound authenticated media preparation/assembly; declaration/member MediaMeta; declarations/fields: `MediaMeta` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 749 | Bound authenticated media preparation/assembly; declaration/member MediaMeta.ostree_commit; declarations/fields: `MediaMeta.ostree_commit` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 750–753 | Bound authenticated media preparation/assembly; declaration/member MediaMeta.images; declarations/fields: `MediaMeta.images` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 754–774 | Bound authenticated media preparation/assembly; declaration/member parse; declarations/fields: `parse` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 775–799 | Bound authenticated media preparation/assembly; declaration/member verify_meta_images; declarations/fields: `verify_meta_images` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 800–819 | Bound authenticated media preparation/assembly; declaration/member verify_build_meta; declarations/fields: `verify_build_meta` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 820–836 | Bound authenticated media preparation/assembly; declaration/member setup_media_rootfs; declarations/fields: `setup_media_rootfs` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 837–871 | Bound authenticated media preparation/assembly; declaration/member verify_customized_iso; declarations/fields: `verify_customized_iso` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 872–932 | Bound authenticated media preparation/assembly; declaration/member customize_installer_iso; declarations/fields: `customize_installer_iso` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 933–1002, 1181–1204 | Immutable emitted media readback verification; declarations/fields: `verify_media_readback`, `oracle_assembler_pin_requires_buildroot_selection` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1003–1035 | Immutable emitted media readback verification; declaration/member prepare_and_verify_media; declarations/fields: `prepare_and_verify_media` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1036–1082 | Immutable emitted media readback verification; declaration/member seal_media; declarations/fields: `seal_media` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1083–1154 | Immutable emitted media readback verification; declaration/member assemble_media; declarations/fields: `assemble_media` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1155–1158 | Immutable emitted media readback verification; declaration/member tests; declarations/fields: `tests` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 1159–1180 | Source assertion of Authenticated installation media; declarations/fields: `oracle_media_url_has_no_credentials_or_mutable_query` |

<a id="coverage-cc36266cdbdc"></a>

<a id="rustsoda-release-imagesrcmodelrs-1"></a>

## [lib/soda-release-image/src/model.rs](../../../../../lib/soda-release-image/src/model.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–11 | Exact source revision/digest and URL parsing helpers; declarations/fields: `DELIVER_PATH`, `DELIVER_IMAGES_PATH`, `NAMES`, `FORGEJO_COMPILER_IMAGE`, `SCHEMA_VERSION`, `SODA_SOURCE`, `is_revision`, `is_digest`, `oci_architecture`, `prefixed_digest`, `deliver_hash`, `is_dotted_numbers`, `is_coreos_release`, `valid_repository_prefix`, `UrlParts`, `parse_url`, `https_url`, `is_loopback_addr`, `parse_ipv4`, `parse_ipv6`, `parse_v6_side` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 12 | Exact source revision/digest and URL parsing helpers; declaration/member DELIVER_PATH; declarations/fields: `DELIVER_PATH` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 13–14 | Exact source revision/digest and URL parsing helpers; declaration/member DELIVER_IMAGES_PATH; declarations/fields: `DELIVER_IMAGES_PATH` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 15–22 | Exact source revision/digest and URL parsing helpers; declaration/member NAMES; declarations/fields: `NAMES` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 23–25 | Exact source revision/digest and URL parsing helpers; declaration/member FORGEJO_COMPILER_IMAGE; declarations/fields: `FORGEJO_COMPILER_IMAGE` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 26 | Exact source revision/digest and URL parsing helpers; declaration/member SCHEMA_VERSION; declarations/fields: `SCHEMA_VERSION` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 27–28 | Exact source revision/digest and URL parsing helpers; declaration/member SODA_SOURCE; declarations/fields: `SODA_SOURCE` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 29–32 | Exact source revision/digest and URL parsing helpers; declaration/member is_revision; declarations/fields: `is_revision` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 33–36 | Exact source revision/digest and URL parsing helpers; declaration/member is_digest; declarations/fields: `is_digest` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 37–41 | Exact source revision/digest and URL parsing helpers; declaration/member oci_architecture; declarations/fields: `oci_architecture` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 42–49 | Exact source revision/digest and URL parsing helpers; declaration/member prefixed_digest; declarations/fields: `prefixed_digest` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 50–53 | Exact source revision/digest and URL parsing helpers; declaration/member deliver_hash; declarations/fields: `deliver_hash` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 54–61 | Exact source revision/digest and URL parsing helpers; declaration/member is_dotted_numbers; declarations/fields: `is_dotted_numbers` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 62–65 | Exact source revision/digest and URL parsing helpers; declaration/member is_coreos_release; declarations/fields: `is_coreos_release` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 66–104 | Exact source revision/digest and URL parsing helpers; declaration/member valid_repository_prefix; declarations/fields: `valid_repository_prefix` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 105 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts; declarations/fields: `UrlParts` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 106 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.scheme; declarations/fields: `UrlParts.scheme` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 107 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.host; declarations/fields: `UrlParts.host` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 108 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.hostname; declarations/fields: `UrlParts.hostname` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 109 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.user; declarations/fields: `UrlParts.user` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 110 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.raw_query; declarations/fields: `UrlParts.raw_query` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 111 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.force_query; declarations/fields: `UrlParts.force_query` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 112–116 | Exact source revision/digest and URL parsing helpers; declaration/member UrlParts.fragment; declarations/fields: `UrlParts.fragment` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 117–187 | Exact source revision/digest and URL parsing helpers; declaration/member parse_url; declarations/fields: `parse_url` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 188–204 | Exact source revision/digest and URL parsing helpers; declaration/member https_url; declarations/fields: `https_url` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 205–214 | Exact source revision/digest and URL parsing helpers; declaration/member is_loopback_addr; declarations/fields: `is_loopback_addr` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 215–229 | Exact source revision/digest and URL parsing helpers; declaration/member parse_ipv4; declarations/fields: `parse_ipv4` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 230–266 | Exact source revision/digest and URL parsing helpers; declaration/member parse_ipv6; declarations/fields: `parse_ipv6` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 267–290 | Exact source revision/digest and URL parsing helpers; declaration/member parse_v6_side; declarations/fields: `parse_v6_side` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 291 | Immutable OCI image/payload identity contract; declarations/fields: `Image` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 292 | Immutable OCI image/payload identity contract; declaration/member Image.manifest; declarations/fields: `Image.manifest` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 293 | Immutable OCI image/payload identity contract; declaration/member Image.config; declarations/fields: `Image.config` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 294 | Immutable OCI image/payload identity contract; declaration/member Image.architecture; declarations/fields: `Image.architecture` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 295 | Immutable OCI image/payload identity contract; declaration/member Image.revision; declarations/fields: `Image.revision` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 296 | Immutable OCI image/payload identity contract; declaration/member Image.source; declarations/fields: `Image.source` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 297 | Immutable OCI image/payload identity contract; declaration/member Image.base_name; declarations/fields: `Image.base_name` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 298–301 | Immutable OCI image/payload identity contract; declaration/member Image.base_digest; declarations/fields: `Image.base_digest` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 302–325, 374–383, 420–466 | Immutable OCI image/payload identity contract; declaration/member parse; declarations/fields: `parse` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 326–354, 384–403, 475–530 | Immutable OCI image/payload identity contract; declaration/member to_json; declarations/fields: `to_json` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 355 | Immutable OCI image/payload identity contract; declaration/member ProducedImage; declarations/fields: `ProducedImage` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 356 | Immutable OCI image/payload identity contract; declaration/member ProducedImage.manifest; declarations/fields: `ProducedImage.manifest` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 357 | Immutable OCI image/payload identity contract; declaration/member ProducedImage.config; declarations/fields: `ProducedImage.config` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 358–365 | Immutable OCI image/payload identity contract; declaration/member ProducedImage.archive_sha256; declarations/fields: `ProducedImage.archive_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 366 | Immutable OCI image/payload identity contract; declaration/member PayloadImage; declarations/fields: `PayloadImage` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 367 | Immutable OCI image/payload identity contract; declaration/member PayloadImage.reference; declarations/fields: `PayloadImage.reference` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 368 | Immutable OCI image/payload identity contract; declaration/member PayloadImage.config; declarations/fields: `PayloadImage.config` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 369 | Immutable OCI image/payload identity contract; declaration/member PayloadImage.manifest; declarations/fields: `PayloadImage.manifest` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 370–373 | Immutable OCI image/payload identity contract; declaration/member PayloadImage.archive_sha256; declarations/fields: `PayloadImage.archive_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 404 | Immutable OCI image/payload identity contract; declaration/member Payload; declarations/fields: `Payload` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 405 | Immutable OCI image/payload identity contract; declaration/member Payload.format; declarations/fields: `Payload.format` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 406 | Immutable OCI image/payload identity contract; declaration/member Payload.id; declarations/fields: `Payload.id` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 407 | Immutable OCI image/payload identity contract; declaration/member Payload.revision; declarations/fields: `Payload.revision` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 408 | Immutable OCI image/payload identity contract; declaration/member Payload.architecture; declarations/fields: `Payload.architecture` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 409 | Immutable OCI image/payload identity contract; declaration/member Payload.core_os; declarations/fields: `Payload.core_os` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 410 | Immutable OCI image/payload identity contract; declaration/member Payload.base; declarations/fields: `Payload.base` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 411 | Immutable OCI image/payload identity contract; declaration/member Payload.repository_prefix; declarations/fields: `Payload.repository_prefix` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 412 | Immutable OCI image/payload identity contract; declaration/member Payload.schema; declarations/fields: `Payload.schema` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 413 | Immutable OCI image/payload identity contract; declaration/member Payload.presentation_sha256; declarations/fields: `Payload.presentation_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 414 | Immutable OCI image/payload identity contract; declaration/member Payload.host_packages_sha256; declarations/fields: `Payload.host_packages_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 415 | Immutable OCI image/payload identity contract; declaration/member Payload.images; declarations/fields: `Payload.images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 416–419 | Immutable OCI image/payload identity contract; declaration/member Payload.upgrade_from; declarations/fields: `Payload.upgrade_from` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 467–474 | Immutable OCI image/payload identity contract; declaration/member image; declarations/fields: `image` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 531–537 | Immutable OCI image/payload identity contract; declaration/member valid_identity; declarations/fields: `valid_identity` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 538–551 | Immutable OCI image/payload identity contract; declaration/member valid_base; declarations/fields: `valid_base` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 552–576 | Immutable OCI image/payload identity contract; declaration/member valid_images; declarations/fields: `valid_images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 577–595 | Immutable OCI image/payload identity contract; declaration/member validate; declarations/fields: `validate` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 596–608 | Immutable OCI image/payload identity contract; declaration/member load; declarations/fields: `load` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 609 | Native Forgejo toolchain provenance; declarations/fields: `ForgejoToolchain` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 610 | Native Forgejo toolchain provenance; declaration/member ForgejoToolchain.compiler_image; declarations/fields: `ForgejoToolchain.compiler_image` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 611–614 | Native Forgejo toolchain provenance; declaration/member ForgejoToolchain.apk_packages; declarations/fields: `ForgejoToolchain.apk_packages` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 615–629 | Native Forgejo toolchain provenance; declaration/member parse; declarations/fields: `parse` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 630–647 | Native Forgejo toolchain provenance; declaration/member to_json; declarations/fields: `to_json` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 648–660 | Native Forgejo toolchain provenance; declaration/member validate; declarations/fields: `validate` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 661–684 | Native Forgejo toolchain provenance; declaration/member valid_forgejo_apk_list; declarations/fields: `valid_forgejo_apk_list` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 685–695 | Native Forgejo toolchain provenance; declaration/member has_forgejo_native_build_tools; declarations/fields: `has_forgejo_native_build_tools` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 696 | Immutable candidate/content provenance validation; declarations/fields: `Candidate` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 697 | Immutable candidate/content provenance validation; declaration/member Candidate.format; declarations/fields: `Candidate.format` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 698 | Immutable candidate/content provenance validation; declaration/member Candidate.host; declarations/fields: `Candidate.host` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 699 | Immutable candidate/content provenance validation; declaration/member Candidate.host_reference; declarations/fields: `Candidate.host_reference` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 700 | Immutable candidate/content provenance validation; declaration/member Candidate.host_archive_sha256; declarations/fields: `Candidate.host_archive_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 701 | Immutable candidate/content provenance validation; declaration/member Candidate.payload_sha256; declarations/fields: `Candidate.payload_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 702 | Immutable candidate/content provenance validation; declaration/member Candidate.migration; declarations/fields: `Candidate.migration` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 703 | Immutable candidate/content provenance validation; declaration/member Candidate.notes; declarations/fields: `Candidate.notes` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 704 | Immutable candidate/content provenance validation; declaration/member Candidate.forgejo_revision; declarations/fields: `Candidate.forgejo_revision` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 705 | Immutable candidate/content provenance validation; declaration/member Candidate.forgejo_source_sha256; declarations/fields: `Candidate.forgejo_source_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 706 | Immutable candidate/content provenance validation; declaration/member Candidate.forgejo_toolchain; declarations/fields: `Candidate.forgejo_toolchain` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 707 | Immutable candidate/content provenance validation; declaration/member Candidate.architecture; declarations/fields: `Candidate.architecture` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 708–711 | Immutable candidate/content provenance validation; declaration/member Candidate.content_sha256; declarations/fields: `Candidate.content_sha256` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 712–757 | Immutable candidate/content provenance validation; declaration/member parse; declarations/fields: `parse` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 758–811 | Immutable candidate/content provenance validation; declaration/member to_json; declarations/fields: `to_json` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 812–828 | Immutable candidate/content provenance validation; declaration/member validate; declarations/fields: `validate` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 829–842 | Immutable candidate/content provenance validation; declaration/member valid_candidate_host; declarations/fields: `valid_candidate_host` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 843–854 | Immutable candidate/content provenance validation; declaration/member valid_candidate_provenance; declarations/fields: `valid_candidate_provenance` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 855–865 | Immutable candidate/content provenance validation; declaration/member valid_candidate_source; declarations/fields: `valid_candidate_source` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 866–872 | Immutable candidate/content provenance validation; declaration/member valid_candidate_build; declarations/fields: `valid_candidate_build` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 873–884 | Immutable candidate/content provenance validation; declaration/member REQUIRED_CANDIDATE_CONTENT; declarations/fields: `REQUIRED_CANDIDATE_CONTENT` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 885–892 | Immutable candidate/content provenance validation; declaration/member content_get; declarations/fields: `content_get` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 893–922 | Immutable candidate/content provenance validation; declaration/member valid_candidate_content; declarations/fields: `valid_candidate_content` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 923–939 | Immutable candidate/content provenance validation; declaration/member valid_candidate_asset_name; declarations/fields: `valid_candidate_asset_name` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 940 | Role release trust and public-key admission; declarations/fields: `Trust` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 941 | Role release trust and public-key admission; declaration/member Trust.format; declarations/fields: `Trust.format` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 942 | Role release trust and public-key admission; declaration/member Trust.prefix; declarations/fields: `Trust.prefix` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 943 | Role release trust and public-key admission; declaration/member Trust.epoch; declarations/fields: `Trust.epoch` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 944 | Role release trust and public-key admission; declaration/member Trust.keys; declarations/fields: `Trust.keys` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 945 | Role release trust and public-key admission; declaration/member Trust.not_before; declarations/fields: `Trust.not_before` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 946 | Role release trust and public-key admission; declaration/member Trust.max_age_seconds; declarations/fields: `Trust.max_age_seconds` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 947 | Role release trust and public-key admission; declaration/member Trust.clock_skew_seconds; declarations/fields: `Trust.clock_skew_seconds` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 948–951 | Role release trust and public-key admission; declaration/member Trust.minimum_sequence; declarations/fields: `Trust.minimum_sequence` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 952–1012 | Role release trust and public-key admission; declaration/member parse; declarations/fields: `parse` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1013–1020 | Role release trust and public-key admission; declaration/member role_keys; declarations/fields: `role_keys` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1021–1028 | Role release trust and public-key admission; declaration/member minimum; declarations/fields: `minimum` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1029–1053 | Role release trust and public-key admission; declaration/member validate; declarations/fields: `validate` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1054–1071 | Role release trust and public-key admission; declaration/member admit_trust_role_keys; declarations/fields: `admit_trust_role_keys` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1072–1096 | Role release trust and public-key admission; declaration/member parse_trust_public_key; declarations/fields: `parse_trust_public_key` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1097–1117 | Role release trust and public-key admission; declaration/member read_der_length; declarations/fields: `read_der_length` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 1118–1176 | Role release trust and public-key admission; declaration/member is_p256_spki; declarations/fields: `is_p256_spki` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1177 | Explicit media signing permit and protected secret paths; declarations/fields: `Permit` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1178 | Explicit media signing permit and protected secret paths; declaration/member Permit.format; declarations/fields: `Permit.format` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1179 | Explicit media signing permit and protected secret paths; declaration/member Permit.repository; declarations/fields: `Permit.repository` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1180 | Explicit media signing permit and protected secret paths; declaration/member Permit.digest; declarations/fields: `Permit.digest` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1181 | Explicit media signing permit and protected secret paths; declaration/member Permit.previous; declarations/fields: `Permit.previous` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1182–1185 | Explicit media signing permit and protected secret paths; declaration/member Permit.expires; declarations/fields: `Permit.expires` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1186–1209, 1229–1244 | Explicit media signing permit and protected secret paths; declaration/member to_json; declarations/fields: `to_json` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1210 | Explicit media signing permit and protected secret paths; declaration/member SecretFiles; declarations/fields: `SecretFiles` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1211 | Explicit media signing permit and protected secret paths; declaration/member SecretFiles.key; declarations/fields: `SecretFiles.key` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1212–1215 | Explicit media signing permit and protected secret paths; declaration/member SecretFiles.passphrase; declarations/fields: `SecretFiles.passphrase` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 1216–1228 | Explicit media signing permit and protected secret paths; declaration/member parse; declarations/fields: `parse` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1245 | Pinned CoreOS/Tailnet live-input contracts and validators; declarations/fields: `CoreOSImage` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1246 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member CoreOSImage.url; declarations/fields: `CoreOSImage.url` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1247 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member CoreOSImage.signature_url; declarations/fields: `CoreOSImage.signature_url` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1248 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member CoreOSImage.sha256; declarations/fields: `CoreOSImage.sha256` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1249–1252 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member CoreOSImage.uncompressed_sha256; declarations/fields: `CoreOSImage.uncompressed_sha256` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1253–1267, 1277–1299, 1313–1326, 1333–1348 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member parse; declarations/fields: `parse` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1268 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member ResolvedCoreOS; declarations/fields: `ResolvedCoreOS` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1269 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member ResolvedCoreOS.release; declarations/fields: `ResolvedCoreOS.release` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1270 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member ResolvedCoreOS.metadata_url; declarations/fields: `ResolvedCoreOS.metadata_url` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1271 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member ResolvedCoreOS.container; declarations/fields: `ResolvedCoreOS.container` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1272 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member ResolvedCoreOS.iso; declarations/fields: `ResolvedCoreOS.iso` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1273–1276 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member ResolvedCoreOS.qemu; declarations/fields: `ResolvedCoreOS.qemu` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1300–1305 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member container_ref; declarations/fields: `container_ref` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1306 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member TailnetInputs; declarations/fields: `TailnetInputs` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1307 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member TailnetInputs.version; declarations/fields: `TailnetInputs.version` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1308 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member TailnetInputs.sha256; declarations/fields: `TailnetInputs.sha256` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1309–1312 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member TailnetInputs.base; declarations/fields: `TailnetInputs.base` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1327 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member LiveInputs; declarations/fields: `LiveInputs` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1328 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member LiveInputs.core_os; declarations/fields: `LiveInputs.core_os` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1329–1332 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member LiveInputs.tailnet; declarations/fields: `LiveInputs.tailnet` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1349–1352 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member find_arch; declarations/fields: `find_arch` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1353–1382 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member valid_stream_images; declarations/fields: `valid_stream_images` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1383–1399 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member valid_tailnet_inputs; declarations/fields: `valid_tailnet_inputs` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1400–1422 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member valid_resolved_core_os; declarations/fields: `valid_resolved_core_os` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1423–1428 | Pinned CoreOS/Tailnet live-input contracts and validators; declaration/member valid_live_inputs; declarations/fields: `valid_live_inputs` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1429–1432 | Candidate/input source vectors; declarations/fields: `tests` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1433–1443 | Candidate/input source vectors; declaration/member oracle_prefix_and_digest_shapes; declarations/fields: `oracle_prefix_and_digest_shapes` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1444–1466 | Candidate/input source vectors; declaration/member oracle_url_parser_matches_go_probe; declarations/fields: `oracle_url_parser_matches_go_probe` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1467–1475 | Candidate/input source vectors; declaration/member oracle_payload_identity_validation; declarations/fields: `oracle_payload_identity_validation` |

<a id="coverage-6170dd468d15"></a>

## [lib/soda-release-image/src/payload_stage.rs](../../../../../lib/soda-release-image/src/payload_stage.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; C09-paths regression tests rowed.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–113 | Native candidate payload/image metadata staging; declarations/fields: `link_candidate_commands`, `stage_candidate_forgejo`, `stage_extension_package`, `record_candidate_images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 114–278 | Staged immutable candidate input/content inspection; declarations/fields: `inspect_candidate_forgejo`, `inspect_candidate_files`, `inspect_extension_assets`, `inspect_packaged_file`, `seal_candidate_payload` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 279–406 | Complete candidate metadata/content staging; declarations/fields: `complete_candidate`, `tests`, `oracle_record_candidate_images_binds_references`, `extension_package_reads_new_layout`, `cmd1_staging_links_resolve_without_terminal_identity` |

<a id="coverage-c74a73ba4b6a"></a>

<a id="rustsoda-release-imagesrcpreparers-1"></a>

## [lib/soda-release-image/src/prepare.rs](../../../../../lib/soda-release-image/src/prepare.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; no drift.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–19 | Complete native candidate image assembly and authenticated live media; declarations/fields: `Base`, `image`, `load_base`, `load_base_from_file`, `base_from_resolved`, `PreparedWriter`, `write`, `copy_file`, `write_base_files`, `stage_symlinks_and_extras`, `stage_rootfs_files`, `write_build_record`, `rootfs_file_map`, `load_base_inputs`, `load_base_inputs_resolved`, `finish_base_inputs`, `prepare`, `prepare_resolved`, `finish_prepare`, `inventory`, `tests`, `oracle_base_inputs_require_exact_revision`, `Stub`, `source`, `forgejo_source`, `forgejo_revision`, `native`, `out`, `arch`, `revision`, `live_inputs`, `execute`, `capture`, `next`, `resolve_inputs`, `dependencies`, `compile`, `compile_rust`, `stage_fork_binary`, `assets`, `images`, `inspect_oci`, `verify_content`, `resolve_core_os`, `read_live_inputs`, `check_native`, `sign_media`, `verify_copy`, `write_document` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 20 | Complete native candidate image assembly and authenticated live media; declaration/member Base; declarations/fields: `Base` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 21 | Complete native candidate image assembly and authenticated live media; declaration/member Base.release; declarations/fields: `Base.release` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 22 | Complete native candidate image assembly and authenticated live media; declaration/member Base.metadata_url; declarations/fields: `Base.metadata_url` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 23–26 | Complete native candidate image assembly and authenticated live media; declaration/member Base.images; declarations/fields: `Base.images` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 27–40 | Complete native candidate image assembly and authenticated live media; declaration/member image; declarations/fields: `image` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 41–49 | Complete native candidate image assembly and authenticated live media; declaration/member load_base; declarations/fields: `load_base` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 50–58 | Complete native candidate image assembly and authenticated live media; declaration/member load_base_from_file; declarations/fields: `load_base_from_file` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 59–68 | Complete native candidate image assembly and authenticated live media; declaration/member base_from_resolved; declarations/fields: `base_from_resolved` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 69 | Complete native candidate image assembly and authenticated live media; declaration/member PreparedWriter; declarations/fields: `PreparedWriter` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 70 | Complete native candidate image assembly and authenticated live media; declaration/member PreparedWriter.source; declarations/fields: `PreparedWriter.source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 71–74 | Complete native candidate image assembly and authenticated live media; declaration/member PreparedWriter.out; declarations/fields: `PreparedWriter.out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 75–82 | Complete native candidate image assembly and authenticated live media; declaration/member write; declarations/fields: `write` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 83–98 | Complete native candidate image assembly and authenticated live media; declaration/member copy_file; declarations/fields: `copy_file` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 99–114 | Complete native candidate image assembly and authenticated live media; declaration/member write_base_files; declarations/fields: `write_base_files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 115–129 | Complete native candidate image assembly and authenticated live media; declaration/member stage_symlinks_and_extras; declarations/fields: `stage_symlinks_and_extras` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 130–140 | Complete native candidate image assembly and authenticated live media; declaration/member stage_rootfs_files; declarations/fields: `stage_rootfs_files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 141–179 | Complete native candidate image assembly and authenticated live media; declaration/member write_build_record; declarations/fields: `write_build_record` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 180–252 | Complete native candidate image assembly and authenticated live media; declaration/member rootfs_file_map; declarations/fields: `rootfs_file_map` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 253–262 | Complete native candidate image assembly and authenticated live media; declaration/member load_base_inputs; declarations/fields: `load_base_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 263–273 | Complete native candidate image assembly and authenticated live media; declaration/member load_base_inputs_resolved; declarations/fields: `load_base_inputs_resolved` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 274–289 | Complete native candidate image assembly and authenticated live media; declaration/member finish_base_inputs; declarations/fields: `finish_base_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 290–302 | Complete native candidate image assembly and authenticated live media; declaration/member prepare; declarations/fields: `prepare` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 303–315 | Complete native candidate image assembly and authenticated live media; declaration/member prepare_resolved; declarations/fields: `prepare_resolved` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 316–338 | Complete native candidate image assembly and authenticated live media; declaration/member finish_prepare; declarations/fields: `finish_prepare` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 339–375 | Complete native candidate image assembly and authenticated live media; declaration/member inventory; declarations/fields: `inventory` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 376–379 | Complete native candidate image assembly and authenticated live media; declaration/member tests; declarations/fields: `tests` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 380–381 | Complete native candidate image assembly and authenticated live media; declaration/member oracle_base_inputs_require_exact_revision; declarations/fields: `oracle_base_inputs_require_exact_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 382–383 | Complete native candidate image assembly and authenticated live media; declaration/member Stub; declarations/fields: `Stub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 384–386 | Complete native candidate image assembly and authenticated live media; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 387–389 | Complete native candidate image assembly and authenticated live media; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 390–392 | Complete native candidate image assembly and authenticated live media; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 393–395 | Complete native candidate image assembly and authenticated live media; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 396–398 | Complete native candidate image assembly and authenticated live media; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 399–401 | Complete native candidate image assembly and authenticated live media; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 402–404 | Complete native candidate image assembly and authenticated live media; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 405–407 | Complete native candidate image assembly and authenticated live media; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 408–410 | Complete native candidate image assembly and authenticated live media; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 411–413 | Complete native candidate image assembly and authenticated live media; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 414–416 | Complete native candidate image assembly and authenticated live media; declaration/member next; declarations/fields: `next` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 417–419 | Complete native candidate image assembly and authenticated live media; declaration/member resolve_inputs; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 420–422 | Complete native candidate image assembly and authenticated live media; declaration/member dependencies; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 423–425 | Complete native candidate image assembly and authenticated live media; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 426–428 | Complete native candidate image assembly and authenticated live media; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 429–431 | Complete native candidate image assembly and authenticated live media; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 432–434 | Complete native candidate image assembly and authenticated live media; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 435–441 | Complete native candidate image assembly and authenticated live media; declaration/member images; declarations/fields: `images` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 442–444 | Complete native candidate image assembly and authenticated live media; declaration/member inspect_oci; declarations/fields: `inspect_oci` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 445–451 | Complete native candidate image assembly and authenticated live media; declaration/member verify_content; declarations/fields: `verify_content` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 452–454 | Complete native candidate image assembly and authenticated live media; declaration/member resolve_core_os; declarations/fields: `resolve_core_os` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 455–457 | Complete native candidate image assembly and authenticated live media; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 458–460 | Complete native candidate image assembly and authenticated live media; declaration/member check_native; declarations/fields: `check_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 461–472 | Complete native candidate image assembly and authenticated live media; declaration/member sign_media; declarations/fields: `sign_media` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 473–482 | Complete native candidate image assembly and authenticated live media; declaration/member verify_copy; declarations/fields: `verify_copy` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 483–499 | Complete native candidate image assembly and authenticated live media; declaration/member write_document; declarations/fields: `write_document` |

<a id="coverage-ad348e05d9a8"></a>

## [lib/soda-release-image/src/request.rs](../../../../../lib/soda-release-image/src/request.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 1–265 | Explicit builder target/source/output/media admission contract; declarations/fields: `Request`, `Result`, `to_json`, `validate_development_target`, `validate_media_inputs`, `validate_target`, `wants_media`, `purpose`, `requested_target`, `tests`, `oracle_development_target_admission` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 266–286 | Source assertion of Authenticated installation media; declarations/fields: `oracle_result_omits_empty_media_fields` |

<a id="coverage-3270fc4fae13"></a>

## [lib/soda-release-image/src/sys.rs](../../../../../lib/soda-release-image/src/sys.rs)

Re-audit @HEAD: every row verified declaration-by-declaration against current bytes; CORR-C-001/C-004 command-discovery additions rowed.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–220 | Builder confined filesystem/hash/process primitives; declarations/fields: `File`, `to_json`, `parse`, `clean_path`, `is_abs`, `join`, `dir_name`, `base_name`, `rel_path`, `to_slash`, `components`, `hash_file`, `hex_sha256`, `hex_bytes`, `fresh_directory`, `write_new`, `read_bounded` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 221–261 | Strict bounded JSON input; declarations/fields: `read_json_build`, `read_json_deliver`, `refused`, `private_file` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 262–273 | Matching native architecture/source admission; declarations/fields: `require_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 274–462 | Native shipping command inventory; declarations/fields: `soda_commands`, `is_soda_command`, `mkdir_all`, `create_dir`, `walk`, `tests`, `oracle_clean_path_matches_go`, `oracle_hash_file_refuses_symlink`, `oracle_write_new_refuses_overwrite`, `is_rust_command`, `rust_command_follows_manifest_presence`, `cmd1_discovery_skips_folded_terminal_crate`, `require_native_accepts_matching_x86_64_linux` |


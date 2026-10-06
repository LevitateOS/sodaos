# Soda release image implementation

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 @HEAD `d5012d10` (C08+CORRs+DELTA): 5 headings verified byte-identical (foreign, host, layout, model, request); build/media/payload_stage/prepare/sys STALE drift (splits + CORR-C-001/004/005 + DELTA-C-001) — bannered.

<a id="coverage-0387b9fa196a"></a>

<a id="rustsoda-release-imagesrcbuildrs-1"></a>

## [lib/soda-release-image/src/build.rs](../../../../../lib/soda-release-image/src/build.rs)

> R02 STALE: `lib/soda-release-image/src/build.rs` differs from the audited `rust/soda-release-image/src/build.rs` blob (C08 split / CORR repairs) — intervals pending re-audit.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–31 | Native candidate build runner/cancellation/logging; declarations/fields: `PINNED_GO_VERSION`, `Cancel`, `new`, `cancel`, `is_cancelled`, `SharedFile`, `wrap`, `write`, `flush`, `Runner`, `execute`, `capture`, `open_log`, `reason`, `LogCloser`, `close`, `ProductionInputs`, `build` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 32–35 | Native candidate build runner/cancellation/logging; declaration/member PINNED_GO_VERSION; declarations/fields: `PINNED_GO_VERSION` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 36 | Native candidate build runner/cancellation/logging; declaration/member Cancel; declarations/fields: `Cancel` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 37–40 | Native candidate build runner/cancellation/logging; declaration/member Cancel.flag; declarations/fields: `Cancel.flag` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 41–44, 84–90 | Native candidate build runner/cancellation/logging; declaration/member new; declarations/fields: `new` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 45–48 | Native candidate build runner/cancellation/logging; declaration/member cancel; declarations/fields: `cancel` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 49–56 | Native candidate build runner/cancellation/logging; declaration/member is_cancelled; declarations/fields: `is_cancelled` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 57–59 | Native candidate build runner/cancellation/logging; declaration/member SharedFile; declarations/fields: `SharedFile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 60–65 | Native candidate build runner/cancellation/logging; declaration/member wrap; declarations/fields: `wrap` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 66–69 | Native candidate build runner/cancellation/logging; declaration/member write; declarations/fields: `write` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 70–77 | Native candidate build runner/cancellation/logging; declaration/member flush; declarations/fields: `flush` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 78 | Native candidate build runner/cancellation/logging; declaration/member Runner; declarations/fields: `Runner` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 79 | Native candidate build runner/cancellation/logging; declaration/member Runner.cancel; declarations/fields: `Runner.cancel` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 80–83 | Native candidate build runner/cancellation/logging; declaration/member Runner.log; declarations/fields: `Runner.log` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 91–98 | Native candidate build runner/cancellation/logging; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 99–103 | Native candidate build runner/cancellation/logging; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 104–148 | Native candidate build runner/cancellation/logging; declaration/member open_log; declarations/fields: `open_log` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 149–154 | Native candidate build runner/cancellation/logging; declaration/member reason; declarations/fields: `reason` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 155 | Native candidate build runner/cancellation/logging; declaration/member LogCloser; declarations/fields: `LogCloser` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 156–159 | Native candidate build runner/cancellation/logging; declaration/member LogCloser.files; declarations/fields: `LogCloser.files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 160–170 | Native candidate build runner/cancellation/logging; declaration/member close; declarations/fields: `close` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 171 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs; declarations/fields: `ProductionInputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 172 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.source; declarations/fields: `ProductionInputs.source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 173 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.forgejo_source; declarations/fields: `ProductionInputs.forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 174 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.forgejo_revision; declarations/fields: `ProductionInputs.forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 175 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.native; declarations/fields: `ProductionInputs.native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 176 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.out; declarations/fields: `ProductionInputs.out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 177 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.arch; declarations/fields: `ProductionInputs.arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 178 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.revision; declarations/fields: `ProductionInputs.revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 179–184 | Native candidate build runner/cancellation/logging; declaration/member ProductionInputs.live_inputs; declarations/fields: `ProductionInputs.live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 185–204 | Native candidate build runner/cancellation/logging; declaration/member build; declarations/fields: `build` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 205–228 | Clean exact checkout and source/compiler/output admission; declarations/fields: `verify_checkout_source` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 229–266 | Clean exact checkout and source/compiler/output admission; declaration/member verify_committed_revision; declarations/fields: `verify_committed_revision` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 267–276 | Clean exact checkout and source/compiler/output admission; declaration/member verify_compiler; declarations/fields: `verify_compiler` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 277–294 | Clean exact checkout and source/compiler/output admission; declaration/member admit_build_output; declarations/fields: `admit_build_output` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 295–316 | Clean exact checkout and source/compiler/output admission; declaration/member admit_build_inputs; declarations/fields: `admit_build_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 317–323 | Fresh build workspace and native snapshot extraction; declarations/fields: `init_build_directories` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 324–356 | Fresh build workspace and native snapshot extraction; declaration/member extract_build_snapshot; declarations/fields: `extract_build_snapshot` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 357–378 | Fresh build workspace and native snapshot extraction; declaration/member setup_build_workspace; declarations/fields: `setup_build_workspace` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 379–450 | Frozen native base image identity acquisition; declarations/fields: `freeze_base_image_config` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 451–490 | Prepare/compile/stage shipping tools and candidate image; declarations/fields: `prepare_build_host_context` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 491–521 | Prepare/compile/stage shipping tools and candidate image; declaration/member prepare_build_production; declarations/fields: `prepare_build_production` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 522–554 | Prepare/compile/stage shipping tools and candidate image; declaration/member compile_soda_commands; declarations/fields: `compile_soda_commands` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 555–608 | Prepare/compile/stage shipping tools and candidate image; declaration/member record_tool_files; declarations/fields: `record_tool_files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 609–661 | Prepare/compile/stage shipping tools and candidate image; declaration/member RUST_TOOLS; declarations/fields: `RUST_TOOLS` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 662–670 | Prepare/compile/stage shipping tools and candidate image; declaration/member compile_rust_tools; declarations/fields: `compile_rust_tools` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 671–711 | Prepare/compile/stage shipping tools and candidate image; declaration/member compile_shipping_tools; declarations/fields: `compile_shipping_tools` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 712–772 | Prepare/compile/stage shipping tools and candidate image; declaration/member build_host_candidate; declarations/fields: `build_host_candidate` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 773–808 | Prepare/compile/stage shipping tools and candidate image; declaration/member execute_build_production; declarations/fields: `execute_build_production` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 809–826 | Prepare/compile/stage shipping tools and candidate image; declaration/member finalize_build; declarations/fields: `finalize_build` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 827–853 | Prepare/compile/stage shipping tools and candidate image; declaration/member run_build; declarations/fields: `run_build` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 854–865 | Prepare/compile/stage shipping tools and candidate image; declaration/member run_build_inner; declarations/fields: `run_build_inner` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 866 | Real native production operation adapter; declarations/fields: `RunnerProduction` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 867 | Real native production operation adapter; declaration/member RunnerProduction.runner; declarations/fields: `RunnerProduction.runner` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 868–870 | Real native production operation adapter; declaration/member RunnerProduction.inputs; declarations/fields: `RunnerProduction.inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 871–873 | Real native production operation adapter; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 874–876 | Real native production operation adapter; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 877–879 | Real native production operation adapter; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 880–882 | Real native production operation adapter; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 883–885 | Real native production operation adapter; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 886–888 | Real native production operation adapter; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 889–891 | Real native production operation adapter; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 892–894 | Real native production operation adapter; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 895–897 | Real native production operation adapter; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 898–900 | Real native production operation adapter; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 901–903 | Real native production operation adapter; declaration/member next; declarations/fields: `next` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 904–906 | Pinned live-input resolution adapter; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 907–909 | Native dependencies/compile/fork/assets/images adapter; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 910–912 | Native dependencies/compile/fork/assets/images adapter; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 913–915 | Native dependencies/compile/fork/assets/images adapter; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 916–918 | Native dependencies/compile/fork/assets/images adapter; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 919–921 | Native dependencies/compile/fork/assets/images adapter; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 922–924 | Native dependencies/compile/fork/assets/images adapter; declaration/member images; declarations/fields: `images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 925–927 | OCI/content verification adapter; declarations/fields: `inspect_oci` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 928–934 | OCI/content verification adapter; declaration/member verify_content; declarations/fields: `verify_content` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 935–937 | Pinned CoreOS/live inputs adapter; declarations/fields: `resolve_core_os` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 938–940 | Pinned CoreOS/live inputs adapter; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 941–943 | Native candidate check adapter; declarations/fields: `check_native` |
| [D08](../../slices/release-and-installation.md#d08-signing-custody) / active | 944–955 | Media signing adapter; declarations/fields: `sign_media` |
| [D10](../../slices/release-and-installation.md#d10-verified-distribution-consumption) / active | 956–965 | Verified external copy adapter; declarations/fields: `verify_copy` |
| [D07](../../slices/release-and-installation.md#d07-release-admission-and-preparation) / active | 966–973 | Immutable document output adapter; declarations/fields: `write_document` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 974–1059 | Candidate/media orchestration and native build environment; declarations/fields: `build_host_candidate_with_progress`, `build_environment_pairs`, `resolve_build_tool`, `look_path`, `run_build_command`, `link_prepared_assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1060–1122 | Candidate/media orchestration and native build environment; declaration/member build_host_candidate_with_progress; declarations/fields: `build_host_candidate_with_progress` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1123–1154 | Candidate/media orchestration and native build environment; declaration/member build_environment_pairs; declarations/fields: `build_environment_pairs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1155–1163 | Candidate/media orchestration and native build environment; declaration/member resolve_build_tool; declarations/fields: `resolve_build_tool` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1164–1177 | Candidate/media orchestration and native build environment; declaration/member look_path; declarations/fields: `look_path` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1178–1246 | Candidate/media orchestration and native build environment; declaration/member run_build_command; declarations/fields: `run_build_command` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1247–1266 | Candidate/media orchestration and native build environment; declaration/member link_prepared_assets; declarations/fields: `link_prepared_assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1267–1270 | Native build source fixtures/assertions; declarations/fields: `tests` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1271–1327 | Native build source fixtures/assertions; declaration/member oracle_build_command_capture_environment_and_failure; declarations/fields: `oracle_build_command_capture_environment_and_failure` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1328–1329 | Native build source fixtures/assertions; declaration/member oracle_link_prepared_assets_runs_no_commands; declarations/fields: `oracle_link_prepared_assets_runs_no_commands` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1330 | Native build source fixtures/assertions; declaration/member Stub; declarations/fields: `Stub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1331 | Native build source fixtures/assertions; declaration/member Stub.source; declarations/fields: `Stub.source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1332–1334 | Native build source fixtures/assertions; declaration/member Stub.native; declarations/fields: `Stub.native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1335–1337 | Native build source fixtures/assertions; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1338–1340 | Native build source fixtures/assertions; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1341–1343 | Native build source fixtures/assertions; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1344–1346 | Native build source fixtures/assertions; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1347–1349 | Native build source fixtures/assertions; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1350–1352 | Native build source fixtures/assertions; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1353–1355 | Native build source fixtures/assertions; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1356–1358 | Native build source fixtures/assertions; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1359–1361 | Native build source fixtures/assertions; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1362–1364 | Native build source fixtures/assertions; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1365–1367 | Native build source fixtures/assertions; declaration/member next; declarations/fields: `next` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1368–1370 | Native build source fixtures/assertions; declaration/member resolve_inputs; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1371–1373 | Native build source fixtures/assertions; declaration/member dependencies; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1374–1376 | Native build source fixtures/assertions; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1377–1379 | Native build source fixtures/assertions; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1380–1382 | Native build source fixtures/assertions; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1383–1385 | Native build source fixtures/assertions; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1386–1388 | Native build source fixtures/assertions; declaration/member images; declarations/fields: `images` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1389–1391 | Native build source fixtures/assertions; declaration/member inspect_oci; declarations/fields: `inspect_oci` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1392–1398 | Native build source fixtures/assertions; declaration/member verify_content; declarations/fields: `verify_content` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1399–1401 | Native build source fixtures/assertions; declaration/member resolve_core_os; declarations/fields: `resolve_core_os` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1402–1404 | Native build source fixtures/assertions; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1405–1407 | Native build source fixtures/assertions; declaration/member check_native; declarations/fields: `check_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1408–1419 | Native build source fixtures/assertions; declaration/member sign_media; declarations/fields: `sign_media` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1420–1429 | Native build source fixtures/assertions; declaration/member verify_copy; declarations/fields: `verify_copy` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1430–1447 | Native build source fixtures/assertions; declaration/member write_document; declarations/fields: `write_document` |

<a id="coverage-06edbfe39c61"></a>

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

> R02 STALE: `lib/soda-release-image/src/media.rs` differs from the audited `rust/soda-release-image/src/media.rs` blob (C08 split / CORR repairs) — intervals pending re-audit.

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

> R02 STALE: `lib/soda-release-image/src/payload_stage.rs` differs from the audited `rust/soda-release-image/src/payload_stage.rs` blob (C08 split / CORR repairs) — intervals pending re-audit.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–110 | Native candidate payload/image metadata staging; declarations/fields: `link_candidate_commands`, `stage_candidate_forgejo`, `stage_extension_package`, `record_candidate_images` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 111–275 | Staged immutable candidate input/content inspection; declarations/fields: `inspect_candidate_forgejo`, `inspect_candidate_files`, `inspect_extension_assets`, `inspect_packaged_file`, `seal_candidate_payload` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 276–335 | Complete candidate metadata/content staging; declarations/fields: `complete_candidate`, `tests`, `oracle_record_candidate_images_binds_references` |

<a id="coverage-c74a73ba4b6a"></a>

<a id="rustsoda-release-imagesrcpreparers-1"></a>

## [lib/soda-release-image/src/prepare.rs](../../../../../lib/soda-release-image/src/prepare.rs)

> R02 STALE: `lib/soda-release-image/src/prepare.rs` differs from the audited `rust/soda-release-image/src/prepare.rs` blob (C08 split / CORR repairs) — intervals pending re-audit.

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
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 99–119 | Complete native candidate image assembly and authenticated live media; declaration/member write_base_files; declarations/fields: `write_base_files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 120–134 | Complete native candidate image assembly and authenticated live media; declaration/member stage_symlinks_and_extras; declarations/fields: `stage_symlinks_and_extras` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 135–145 | Complete native candidate image assembly and authenticated live media; declaration/member stage_rootfs_files; declarations/fields: `stage_rootfs_files` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 146–184 | Complete native candidate image assembly and authenticated live media; declaration/member write_build_record; declarations/fields: `write_build_record` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 185–254 | Complete native candidate image assembly and authenticated live media; declaration/member rootfs_file_map; declarations/fields: `rootfs_file_map` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 255–264 | Complete native candidate image assembly and authenticated live media; declaration/member load_base_inputs; declarations/fields: `load_base_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 265–275 | Complete native candidate image assembly and authenticated live media; declaration/member load_base_inputs_resolved; declarations/fields: `load_base_inputs_resolved` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 276–291 | Complete native candidate image assembly and authenticated live media; declaration/member finish_base_inputs; declarations/fields: `finish_base_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 292–304 | Complete native candidate image assembly and authenticated live media; declaration/member prepare; declarations/fields: `prepare` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 305–317 | Complete native candidate image assembly and authenticated live media; declaration/member prepare_resolved; declarations/fields: `prepare_resolved` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 318–340 | Complete native candidate image assembly and authenticated live media; declaration/member finish_prepare; declarations/fields: `finish_prepare` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 341–377 | Complete native candidate image assembly and authenticated live media; declaration/member inventory; declarations/fields: `inventory` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 378–381 | Complete native candidate image assembly and authenticated live media; declaration/member tests; declarations/fields: `tests` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 382–383 | Complete native candidate image assembly and authenticated live media; declaration/member oracle_base_inputs_require_exact_revision; declarations/fields: `oracle_base_inputs_require_exact_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 384–385 | Complete native candidate image assembly and authenticated live media; declaration/member Stub; declarations/fields: `Stub` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 386–388 | Complete native candidate image assembly and authenticated live media; declaration/member source; declarations/fields: `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 389–391 | Complete native candidate image assembly and authenticated live media; declaration/member forgejo_source; declarations/fields: `forgejo_source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 392–394 | Complete native candidate image assembly and authenticated live media; declaration/member forgejo_revision; declarations/fields: `forgejo_revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 395–397 | Complete native candidate image assembly and authenticated live media; declaration/member native; declarations/fields: `native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 398–400 | Complete native candidate image assembly and authenticated live media; declaration/member out; declarations/fields: `out` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 401–403 | Complete native candidate image assembly and authenticated live media; declaration/member arch; declarations/fields: `arch` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 404–406 | Complete native candidate image assembly and authenticated live media; declaration/member revision; declarations/fields: `revision` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 407–409 | Complete native candidate image assembly and authenticated live media; declaration/member live_inputs; declarations/fields: `live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 410–412 | Complete native candidate image assembly and authenticated live media; declaration/member execute; declarations/fields: `execute` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 413–415 | Complete native candidate image assembly and authenticated live media; declaration/member capture; declarations/fields: `capture` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 416–418 | Complete native candidate image assembly and authenticated live media; declaration/member next; declarations/fields: `next` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 419–421 | Complete native candidate image assembly and authenticated live media; declaration/member resolve_inputs; declarations/fields: `resolve_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 422–424 | Complete native candidate image assembly and authenticated live media; declaration/member dependencies; declarations/fields: `dependencies` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 425–427 | Complete native candidate image assembly and authenticated live media; declaration/member compile; declarations/fields: `compile` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 428–430 | Complete native candidate image assembly and authenticated live media; declaration/member compile_rust; declarations/fields: `compile_rust` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 431–433 | Complete native candidate image assembly and authenticated live media; declaration/member stage_fork_binary; declarations/fields: `stage_fork_binary` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 434–436 | Complete native candidate image assembly and authenticated live media; declaration/member assets; declarations/fields: `assets` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 437–443 | Complete native candidate image assembly and authenticated live media; declaration/member images; declarations/fields: `images` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 444–446 | Complete native candidate image assembly and authenticated live media; declaration/member inspect_oci; declarations/fields: `inspect_oci` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 447–453 | Complete native candidate image assembly and authenticated live media; declaration/member verify_content; declarations/fields: `verify_content` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 454–456 | Complete native candidate image assembly and authenticated live media; declaration/member resolve_core_os; declarations/fields: `resolve_core_os` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 457–459 | Complete native candidate image assembly and authenticated live media; declaration/member read_live_inputs; declarations/fields: `read_live_inputs` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 460–462 | Complete native candidate image assembly and authenticated live media; declaration/member check_native; declarations/fields: `check_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 463–474 | Complete native candidate image assembly and authenticated live media; declaration/member sign_media; declarations/fields: `sign_media` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 475–484 | Complete native candidate image assembly and authenticated live media; declaration/member verify_copy; declarations/fields: `verify_copy` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 485–501 | Complete native candidate image assembly and authenticated live media; declaration/member write_document; declarations/fields: `write_document` |

<a id="coverage-ad348e05d9a8"></a>

## [lib/soda-release-image/src/request.rs](../../../../../lib/soda-release-image/src/request.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 1–265 | Explicit builder target/source/output/media admission contract; declarations/fields: `Request`, `Result`, `to_json`, `validate_development_target`, `validate_media_inputs`, `validate_target`, `wants_media`, `purpose`, `requested_target`, `tests`, `oracle_development_target_admission` |
| [D04](../../slices/release-and-installation.md#d04-authenticated-installation-media) / active | 266–286 | Source assertion of Authenticated installation media; declarations/fields: `oracle_result_omits_empty_media_fields` |

<a id="coverage-3270fc4fae13"></a>

## [lib/soda-release-image/src/sys.rs](../../../../../lib/soda-release-image/src/sys.rs)

> R02 STALE: `lib/soda-release-image/src/sys.rs` differs from the audited `rust/soda-release-image/src/sys.rs` blob (C08 split / CORR repairs) — intervals pending re-audit.

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–220 | Builder confined filesystem/hash/process primitives; declarations/fields: `File`, `to_json`, `parse`, `clean_path`, `is_abs`, `join`, `dir_name`, `base_name`, `rel_path`, `to_slash`, `components`, `hash_file`, `hex_sha256`, `hex_bytes`, `fresh_directory`, `write_new`, `read_bounded` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 221–261 | Strict bounded JSON input; declarations/fields: `read_json_build`, `read_json_deliver`, `refused`, `private_file` |
| [D01](../../slices/release-and-installation.md#d01-builder-admission-and-controllers) / active | 262–271 | Matching native architecture/source admission; declarations/fields: `require_native` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 272–391 | Native shipping command inventory; declarations/fields: `soda_commands`, `is_soda_command`, `mkdir_all`, `create_dir`, `walk`, `tests`, `oracle_clean_path_matches_go`, `oracle_hash_file_refuses_symlink`, `oracle_write_new_refuses_overwrite` |


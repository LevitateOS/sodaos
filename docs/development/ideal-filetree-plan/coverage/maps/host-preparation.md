# Host preparation

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-8a335b4517a9"></a>

## [lib/host/src/preparation/candidate.rs](../../../../../lib/host/src/preparation/candidate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–84; current module/import/attribute shell; declaration FactoryCandidate; fields preparation, source_preparation, bundle; declaration validate; declaration decode; declaration deserialize; declaration CandidateVisitor; declaration Value; declaration expecting; declaration visit_map | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/candidate.rs into its current native target.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d930b1fed167"></a>

## [lib/host/src/preparation/decisions.rs](../../../../../lib/host/src/preparation/decisions.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–86; current module/import/attribute shell; declaration RequirementAcceptance; fields id, revision, approver, source_commit, digest; declaration validate; declaration AdminApproval; fields id, revision, approver, effects_digest; declaration deserialize; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/decisions.rs into its current native target.; 10 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a09d39e50be6"></a>

## [lib/host/src/preparation/mod.rs](../../../../../lib/host/src/preparation/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 16–24, 113–146, 162–189, 197–261; current module/import/attribute shell; declaration candidate; declaration decisions; declaration state; declaration GoStringList; declaration from; declaration deserialize; declaration StringListVisitor; declaration Value; declaration expecting; declaration visit_seq; declaration validate; declaration decode; declaration ObjectVisitor; declaration visit_map; declaration validation_tests; declaration wire_tests | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/mod.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 15, 25–49, 61–112, 147–161, 190–196; declaration setup; declaration ROLE_CODER; declaration ROLE_REVIEWER; declaration FACTORY_DIR; declaration FACTORY_PREPARATIONS_DIR; declaration FACTORY_CREDENTIALS_DIR; declaration FACTORY_HOLD_FILE; declaration FACTORY_HELPER; declaration FACTORY_SETUP_ENTRY; declaration FACTORY_CHECK_ENTRY; declaration MAX_APPROVED_FILES; declaration MAX_APPROVED_FILE_SIZE; declaration MAX_APPROVED_TOTAL; declaration MAX_SOURCE_BUNDLE; declaration MAX_PREPARE_TOOLS; declaration PREPARE_APPROVED; declaration PREPARE_WAITING; declaration PREPARE_RUNNING; declaration PREPARE_READY; declaration PREPARE_FAILED; declaration PREPARE_STOPPED; declaration PREPARE_INTERRUPTED; declaration is_hex_lower; declaration valid_preparation_id; declaration valid_decision_id; declaration valid_digest; declaration valid_commit; declaration valid_approved_name; declaration valid_tool_name; declaration Preparation; fields id, project, role, revision, requirements, approval, source_commit, setup_digest, tools, credential; declaration Prepare; fields preparation, setup | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | setup: implement the current Project preparation and checkout allocation duty in mod.rs.; 31 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 50–60; declaration valid_factory_role; declaration valid_prepare_phase | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | valid_factory_role: implement the current factory role accounts duty in mod.rs.; valid_prepare_phase: implement the current factory role accounts duty in mod.rs. — current source lib/host/src/preparation/mod.rs; lines 50-53; module/caller wiring inspected; current source lib/host/src/preparation/mod.rs; lines 54-60; module/caller wiring inspected |

<a id="coverage-216c87ef567d"></a>

## [lib/host/src/preparation/setup.rs](../../../../../lib/host/src/preparation/setup.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–159; current module/import/attribute shell; declaration ApprovedSetup; fields files, bundle; declaration GoBytes; declaration deserialize; declaration BytesVisitor; declaration Value; declaration expecting; declaration visit_str; declaration visit_string; declaration visit_unit; declaration visit_seq; declaration validate; declaration SetupVisitor; declaration visit_map; declaration setup_digest_of | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/setup.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-731a0d4a961a"></a>

## [lib/host/src/preparation/state.rs](../../../../../lib/host/src/preparation/state.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–223; current module/import/attribute shell; declaration ResolvedTool; fields name, path, version; declaration encode_into; declaration deserialize; declaration ObjectVisitor; declaration Value; declaration expecting; declaration visit_map; declaration PrepareState; fields id, project, role, phase, container, source_commit, setup_digest, tools, missing, setup_exit, check_exit, output, ready, stopped, retirement; declaration encode; declaration PrepareInspect; fields project, id; declaration validate; declaration decode; declaration PrepareStop; fields project, id; declaration HoldState; fields active, revision; declaration PrepareHold; fields project, hold, revision | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/state.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-a71054c7ab14"></a>

## [lib/host/src/preparation/validation_tests.rs](../../../../../lib/host/src/preparation/validation_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–287; current module/import/attribute shell; declaration DIGEST; declaration COMMIT; declaration EFFECTS; declaration pid; declaration fid; declaration did; declaration requirements; declaration approval; declaration preparation; declaration setup; declaration validators_match_go_shapes; declaration setup_digest_matches_reference; declaration preparation_validation_branches; declaration approved_setup_validation_branches; declaration prepare_validates_digest_binding; declaration address_and_hold_validation; declaration candidate_validation_branches | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/validation_tests.rs into its current native target.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-2e8d713cc268"></a>

## [lib/host/src/preparation/wire_tests.rs](../../../../../lib/host/src/preparation/wire_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; current module/import/attribute shell; declaration prep_json; declaration prepare_json; declaration prepare_decode_round_trip; declaration preparation_tool_list_null_elements_decode_as_empty_strings; declaration preparation_signed_revision_fields_accept_negative_zero; declaration prepare_rejects_malformed_and_unknown_fields; declaration state_encodes_honor_omitempty | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/preparation/wire_tests.rs into its current native target.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-4932f0950347"></a>

## [lib/host/src/prepare/candidate.rs](../../../../../lib/host/src/prepare/candidate.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–243; current module/import/attribute shell; declaration SavedRequest; fields id, role, setup_digest, source_commit, credential; declaration deserialize; declaration SavedVisitor; declaration Value; declaration expecting; declaration visit_map; declaration prepare_candidate; declaration candidate_protected_file; declaration candidate_approved_files; declaration ERR | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/candidate.rs into its current native target.; 11 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-23cd27e44f63"></a>

## [lib/host/src/prepare/candidate_tests.rs](../../../../../lib/host/src/prepare/candidate_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–176; current module/import/attribute shell; declaration candidate_reuses_ready_source; declaration candidate_rejects_bad_snapshots | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/candidate_tests.rs into its current native target.; candidate_reuses_ready_source: implement the current Unix-socket service and process lifetime duty in candidate_tests.rs.; candidate_rejects_bad_snapshots: implement the current Unix-socket service and process lifetime duty in candidate_tests.rs. — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-64d452e9eb19"></a>

## [lib/host/src/prepare/execution_tests.rs](../../../../../lib/host/src/prepare/execution_tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–328; current module/import/attribute shell; declaration prepare_runs_the_full_chain; declaration prepare_short_circuits_and_recovers; declaration prepare_skips_start_when_blocked; declaration inspect_stop_hold_paths | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/execution_tests.rs into its current native target.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-d789d9bfb44f"></a>

## [lib/host/src/prepare/helper.rs](../../../../../lib/host/src/prepare/helper.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–213; current module/import/attribute shell; declaration FACTORY_INSPECT_LIMIT; declaration encode_files_object; declaration ApprovalResponse; fields approved, repeated, checkout, credential_file; declaration deserialize; declaration ApprovalVisitor; declaration Value; declaration expecting; declaration visit_map; declaration factory_helper; declaration prepare_container; declaration approve_preparation; declaration ERR | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/helper.rs into its current native target.; 13 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-e801e0fa60a4"></a>

## [lib/host/src/prepare/mod.rs](../../../../../lib/host/src/prepare/mod.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–118, 233–236; current module/import/attribute shell; declaration candidate; declaration helper; declaration paths; declaration source; declaration state; declaration tools; declaration StopResponse; fields stopped, retirement, known; declaration deserialize; declaration StopVisitor; declaration Value; declaration expecting; declaration visit_map; declaration HoldResponse; fields hold; declaration HoldVisitor; declaration candidate_tests; declaration execution_tests | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/mod.rs into its current native target.; 21 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 119–212, 237–238; declaration inspect_preparation_state; declaration prepare; declaration inspect_preparation; declaration stop_preparation; declaration ERR; declaration tests | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | inspect_preparation_state: implement the current Project preparation and checkout allocation duty in mod.rs.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 213–232; declaration hold_preparation | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | hold_preparation: implement the current Project maintenance holds duty in mod.rs. — current source lib/host/src/prepare/mod.rs; lines 213-232; module/caller wiring inspected |

<a id="coverage-d0d2d93c2ebb"></a>

## [lib/host/src/prepare/paths.rs](../../../../../lib/host/src/prepare/paths.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; current module/import/attribute shell; declaration path_join; declaration prepare_id_map; declaration preparation_paths; declaration single_line; declaration valid_resolved_tool_path; declaration is_clean_absolute_path | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/paths.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8ec1682d9395"></a>

## [lib/host/src/prepare/source.rs](../../../../../lib/host/src/prepare/source.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; current module/import/attribute shell; declaration role_exec; declaration must_role_exec; declaration clone_preparation_source; declaration confirm_preparation_head; declaration single_line_trimmed | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/source.rs into its current native target.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-cc6bca0c7bbd"></a>

## [lib/host/src/prepare/state.rs](../../../../../lib/host/src/prepare/state.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–248; current module/import/attribute shell; declaration HelperInspection; fields known, phase, role, setup_digest, source_commit, tools, missing, stopped, ready, setup_exit, check_exit, setup_log, check_log, hold, verified; declaration deserialize; declaration InspectionVisitor; declaration Value; declaration expecting; declaration visit_map; declaration LauncherEvidence; fields uid, login, groups, refusal; declaration EvidenceVisitor; declaration quote_bytes_base64; declaration map_preparation_state; declaration ERR | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/state.rs into its current native target.; 16 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-8b52ac36c99c"></a>

## [lib/host/src/prepare/tests.rs](../../../../../lib/host/src/prepare/tests.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–330; current module/import/attribute shell; declaration DIGEST; declaration COMMIT; declaration EFFECTS; declaration MockCall; declaration Mock; fields calls, script; declaration new; declaration run; declaration deadline; declaration test_config; declaration pid; declaration fid; declaration container_payload; declaration observation; declaration preparation; declaration setup; declaration approve_response; declaration path_join_uses_native_components_and_tool_admission_is_strict; declaration idmap_requires_shifted_usable_range; declaration single_line_rules; declaration tool_paths_must_be_clean_and_pinned; declaration preparation_paths_layout; declaration map_state_validates_and_assembles; declaration prepare_container_binds_isolated_target | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/tests.rs into its current native target.; 24 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-b70fcc9a1c56"></a>

## [lib/host/src/prepare/tools.rs](../../../../../lib/host/src/prepare/tools.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–192; current module/import/attribute shell; declaration verify_launcher_environment; declaration resolve_preparation_tools; declaration record_preparation_tools | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/src/prepare/tools.rs into its current native target.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-0ba179ebb6ea"></a>

## [lib/host/tests/project_operations/preparation.rs](../../../../../lib/host/tests/project_operations/preparation.rs)

current line spans and declarations inspected; prior intervals reused only for byte-identical source, otherwise prior units serve as symbol-level allocation evidence

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–166; current module/import/attribute shell; declaration prepare_ready_short_circuit_matches_golden; declaration prepare_validates_before_exec; declaration inspect_preparation_matches_golden; declaration stop_preparation_confirms_retirement; declaration oracle_prepare_state_full_encoding; declaration oracle_prepare_state_missing_encoding | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Imports and module declarations wire lib/host/tests/project_operations/preparation.rs into its current native target.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-bebecbb8e821"></a>
<a id="rustsoda-hostsrcpreparationrs-1"></a>

Former source `rust/soda-host/src/preparation/mod.rs`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-00ec6f2cd3fa"></a>
<a id="rustsoda-hostsrcpreparers-1"></a>

Former source `rust/soda-host/src/prepare/mod.rs`; consult its pinned earlier Git source and the current coverage disposition.

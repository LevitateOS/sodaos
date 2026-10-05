# Host preparation

Current P06/P12 target note: host LauncherEvidence/base64 helper321–332→prepare/state.rs, role_exec538–557→prepare/source.rs, launcher638–732→prepare/tools.rs. Pure HoldState/PrepareHold621–698→preparation/state.rs; Runtime hold859–881→prepare/mod.rs; mixed native hold/inspect/stop assertions stay ONE execution_tests.rs function. These are concern destinations, not authority transfers or source edits.

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-bebecbb8e821"></a>

<a id="rustsoda-hostsrcpreparationrs-1"></a>

## [rust/soda-host/src/preparation.rs](../../../../../rust/soda-host/src/preparation.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–12 | Preparation identifiers and byte bounds; declarations/fields: `ROLE_CODER`, `ROLE_REVIEWER`, `FACTORY_DIR`, `FACTORY_PREPARATIONS_DIR`, `FACTORY_CREDENTIALS_DIR`, `FACTORY_HOLD_FILE`, `FACTORY_HELPER`, `FACTORY_SETUP_ENTRY`, `FACTORY_CHECK_ENTRY`, `MAX_APPROVED_FILES`, `MAX_APPROVED_FILE_SIZE`, `MAX_APPROVED_TOTAL`, `MAX_SOURCE_BUNDLE`, `MAX_PREPARE_TOOLS`, `PREPARE_APPROVED`, `PREPARE_WAITING`, `PREPARE_RUNNING`, `PREPARE_READY`, `PREPARE_FAILED`, `PREPARE_STOPPED`, `PREPARE_INTERRUPTED` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 13 | Preparation identifiers and byte bounds; declaration/member ROLE_CODER; declarations/fields: `ROLE_CODER` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 14–15 | Preparation identifiers and byte bounds; declaration/member ROLE_REVIEWER; declarations/fields: `ROLE_REVIEWER` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 16 | Preparation identifiers and byte bounds; declaration/member FACTORY_DIR; declarations/fields: `FACTORY_DIR` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 17 | Preparation identifiers and byte bounds; declaration/member FACTORY_PREPARATIONS_DIR; declarations/fields: `FACTORY_PREPARATIONS_DIR` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 18 | Preparation identifiers and byte bounds; declaration/member FACTORY_CREDENTIALS_DIR; declarations/fields: `FACTORY_CREDENTIALS_DIR` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 19 | Preparation identifiers and byte bounds; declaration/member FACTORY_HOLD_FILE; declarations/fields: `FACTORY_HOLD_FILE` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 20 | Preparation identifiers and byte bounds; declaration/member FACTORY_HELPER; declarations/fields: `FACTORY_HELPER` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 21 | Preparation identifiers and byte bounds; declaration/member FACTORY_SETUP_ENTRY; declarations/fields: `FACTORY_SETUP_ENTRY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 22–23 | Preparation identifiers and byte bounds; declaration/member FACTORY_CHECK_ENTRY; declarations/fields: `FACTORY_CHECK_ENTRY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 24 | Preparation identifiers and byte bounds; declaration/member MAX_APPROVED_FILES; declarations/fields: `MAX_APPROVED_FILES` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 25 | Preparation identifiers and byte bounds; declaration/member MAX_APPROVED_FILE_SIZE; declarations/fields: `MAX_APPROVED_FILE_SIZE` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 26 | Preparation identifiers and byte bounds; declaration/member MAX_APPROVED_TOTAL; declarations/fields: `MAX_APPROVED_TOTAL` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 27 | Preparation identifiers and byte bounds; declaration/member MAX_SOURCE_BUNDLE; declarations/fields: `MAX_SOURCE_BUNDLE` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 28–29 | Preparation identifiers and byte bounds; declaration/member MAX_PREPARE_TOOLS; declarations/fields: `MAX_PREPARE_TOOLS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 30 | Preparation identifiers and byte bounds; declaration/member PREPARE_APPROVED; declarations/fields: `PREPARE_APPROVED` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 31 | Preparation identifiers and byte bounds; declaration/member PREPARE_WAITING; declarations/fields: `PREPARE_WAITING` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 32 | Preparation identifiers and byte bounds; declaration/member PREPARE_RUNNING; declarations/fields: `PREPARE_RUNNING` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 33 | Preparation identifiers and byte bounds; declaration/member PREPARE_READY; declarations/fields: `PREPARE_READY` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 34 | Preparation identifiers and byte bounds; declaration/member PREPARE_FAILED; declarations/fields: `PREPARE_FAILED` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 35 | Preparation identifiers and byte bounds; declaration/member PREPARE_STOPPED; declarations/fields: `PREPARE_STOPPED` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 36–37 | Preparation identifiers and byte bounds; declaration/member PREPARE_INTERRUPTED; declarations/fields: `PREPARE_INTERRUPTED` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 38–41 | Fixed factory-role admission; declarations/fields: `valid_factory_role` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 42–48 | Fixed factory-role admission; declaration/member valid_prepare_phase; declarations/fields: `valid_prepare_phase` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 49–55 | Exact input digest/id validators; declarations/fields: `is_hex_lower` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 56–60 | Exact input digest/id validators; declaration/member valid_preparation_id; declarations/fields: `valid_preparation_id` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 61–65 | Exact input digest/id validators; declaration/member valid_decision_id; declarations/fields: `valid_decision_id` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 66–70 | Exact input digest/id validators; declaration/member valid_digest; declarations/fields: `valid_digest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 71–75 | Exact input digest/id validators; declaration/member valid_commit; declarations/fields: `valid_commit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 76–90 | Exact input digest/id validators; declaration/member valid_approved_name; declarations/fields: `valid_approved_name` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 91–101 | Exact input digest/id validators; declaration/member valid_tool_name; declarations/fields: `valid_tool_name` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 102 | Maintainer requirement acceptance reference; declarations/fields: `RequirementAcceptance` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 103 | Maintainer requirement acceptance reference; declaration/member RequirementAcceptance.id; declarations/fields: `RequirementAcceptance.id` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 104 | Maintainer requirement acceptance reference; declaration/member RequirementAcceptance.revision; declarations/fields: `RequirementAcceptance.revision` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 105 | Maintainer requirement acceptance reference; declaration/member RequirementAcceptance.approver; declarations/fields: `RequirementAcceptance.approver` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 106 | Maintainer requirement acceptance reference; declaration/member RequirementAcceptance.source_commit; declarations/fields: `RequirementAcceptance.source_commit` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 107–109 | Maintainer requirement acceptance reference; declaration/member RequirementAcceptance.digest; declarations/fields: `RequirementAcceptance.digest` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 110–133 | Maintainer requirement acceptance reference; declaration/member REQUIREMENT_ACCEPTANCE_SPECS; declarations/fields: `REQUIREMENT_ACCEPTANCE_SPECS` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 134–145 | Maintainer requirement acceptance reference; declaration/member validate; declarations/fields: `validate` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 146–157 | Maintainer requirement acceptance reference; declaration/member from_map; declarations/fields: `from_map` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 158 | Administrator exact-effects approval reference; declarations/fields: `AdminApproval` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 159 | Administrator exact-effects approval reference; declaration/member AdminApproval.id; declarations/fields: `AdminApproval.id` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 160 | Administrator exact-effects approval reference; declaration/member AdminApproval.revision; declarations/fields: `AdminApproval.revision` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 161 | Administrator exact-effects approval reference; declaration/member AdminApproval.approver; declarations/fields: `AdminApproval.approver` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 162–164 | Administrator exact-effects approval reference; declaration/member AdminApproval.effects_digest; declarations/fields: `AdminApproval.effects_digest` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 165–184 | Administrator exact-effects approval reference; declaration/member ADMIN_APPROVAL_SPECS; declarations/fields: `ADMIN_APPROVAL_SPECS` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 185–195 | Administrator exact-effects approval reference; declaration/member validate; declarations/fields: `validate` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 196–206 | Administrator exact-effects approval reference; declaration/member from_map; declarations/fields: `from_map` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 207 | Immutable preparation identity and materialized inputs; declarations/fields: `Preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 208 | Immutable preparation identity and materialized inputs; declaration/member Preparation.id; declarations/fields: `Preparation.id` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 209 | Immutable preparation identity and materialized inputs; declaration/member Preparation.project; declarations/fields: `Preparation.project` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 210 | Immutable preparation identity and materialized inputs; declaration/member Preparation.role; declarations/fields: `Preparation.role` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 211 | Immutable preparation identity and materialized inputs; declaration/member Preparation.revision; declarations/fields: `Preparation.revision` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 212 | Immutable preparation identity and materialized inputs; declaration/member Preparation.requirements; declarations/fields: `Preparation.requirements` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 213 | Immutable preparation identity and materialized inputs; declaration/member Preparation.approval; declarations/fields: `Preparation.approval` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 214 | Immutable preparation identity and materialized inputs; declaration/member Preparation.source_commit; declarations/fields: `Preparation.source_commit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 215 | Immutable preparation identity and materialized inputs; declaration/member Preparation.setup_digest; declarations/fields: `Preparation.setup_digest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 216 | Immutable preparation identity and materialized inputs; declaration/member Preparation.tools; declarations/fields: `Preparation.tools` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 217–219 | Immutable preparation identity and materialized inputs; declaration/member Preparation.credential; declarations/fields: `Preparation.credential` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 220–271 | Immutable preparation identity and materialized inputs; declaration/member PREPARATION_SPECS; declarations/fields: `PREPARATION_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 272–298 | Immutable preparation identity and materialized inputs; declaration/member validate; declarations/fields: `validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 299–315 | Immutable preparation identity and materialized inputs; declaration/member from_map; declarations/fields: `from_map` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 316 | Approved setup snapshot and native preparation request; declarations/fields: `ApprovedSetup` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 317 | Approved setup snapshot and native preparation request; declaration/member ApprovedSetup.files; declarations/fields: `ApprovedSetup.files` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 318–320 | Approved setup snapshot and native preparation request; declaration/member ApprovedSetup.bundle; declarations/fields: `ApprovedSetup.bundle` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 321–332 | Approved setup snapshot and native preparation request; declaration/member APPROVED_SETUP_SPECS; declarations/fields: `APPROVED_SETUP_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 333–366, 415–423 | Approved setup snapshot and native preparation request; declaration/member validate; declarations/fields: `validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 367–376, 429–437 | Approved setup snapshot and native preparation request; declaration/member from_map; declarations/fields: `from_map` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 377–389 | Approved setup snapshot and native preparation request; declaration/member setup_digest_of; declarations/fields: `setup_digest_of` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 390 | Approved setup snapshot and native preparation request; declaration/member Prepare; declarations/fields: `Prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 391 | Approved setup snapshot and native preparation request; declaration/member Prepare.preparation; declarations/fields: `Prepare.preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 392–394 | Approved setup snapshot and native preparation request; declaration/member Prepare.setup; declarations/fields: `Prepare.setup` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 395–414 | Approved setup snapshot and native preparation request; declaration/member PREPARE_SPECS; declarations/fields: `PREPARE_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 424–428 | Approved setup snapshot and native preparation request; declaration/member from_value; declarations/fields: `from_value` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 438 | Observed tool availability contract; declarations/fields: `ResolvedTool` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 439 | Observed tool availability contract; declaration/member ResolvedTool.name; declarations/fields: `ResolvedTool.name` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 440 | Observed tool availability contract; declaration/member ResolvedTool.path; declarations/fields: `ResolvedTool.path` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 441–443 | Observed tool availability contract; declaration/member ResolvedTool.version; declarations/fields: `ResolvedTool.version` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 444–459 | Observed tool availability contract; declaration/member RESOLVED_TOOL_SPECS; declarations/fields: `RESOLVED_TOOL_SPECS` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 460–467 | Observed tool availability contract; declaration/member from_map; declarations/fields: `from_map` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 468–479 | Observed tool availability contract; declaration/member encode_into; declarations/fields: `encode_into` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 480 | Native preparation state/inspection/stop contracts; declarations/fields: `PrepareState` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 481 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.id; declarations/fields: `PrepareState.id` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 482 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.project; declarations/fields: `PrepareState.project` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 483 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.role; declarations/fields: `PrepareState.role` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 484 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.phase; declarations/fields: `PrepareState.phase` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 485 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.container; declarations/fields: `PrepareState.container` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 486 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.source_commit; declarations/fields: `PrepareState.source_commit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 487 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.setup_digest; declarations/fields: `PrepareState.setup_digest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 488 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.tools; declarations/fields: `PrepareState.tools` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 489 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.missing; declarations/fields: `PrepareState.missing` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 490 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.setup_exit; declarations/fields: `PrepareState.setup_exit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 491 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.check_exit; declarations/fields: `PrepareState.check_exit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 492 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.output; declarations/fields: `PrepareState.output` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 493 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.ready; declarations/fields: `PrepareState.ready` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 494 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.stopped; declarations/fields: `PrepareState.stopped` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 495–499 | Native preparation state/inspection/stop contracts; declaration/member PrepareState.retirement; declarations/fields: `PrepareState.retirement` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 500–554 | Native preparation state/inspection/stop contracts; declaration/member encode; declarations/fields: `encode` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 555 | Native preparation state/inspection/stop contracts; declaration/member PrepareInspect; declarations/fields: `PrepareInspect` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 556 | Native preparation state/inspection/stop contracts; declaration/member PrepareInspect.project; declarations/fields: `PrepareInspect.project` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 557–559 | Native preparation state/inspection/stop contracts; declaration/member PrepareInspect.id; declarations/fields: `PrepareInspect.id` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 560–571 | Native preparation state/inspection/stop contracts; declaration/member PREPARE_INSPECT_SPECS; declarations/fields: `PREPARE_INSPECT_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 572–578, 600–606 | Native preparation state/inspection/stop contracts; declaration/member validate; declarations/fields: `validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 579–584, 607–611 | Native preparation state/inspection/stop contracts; declaration/member from_value; declarations/fields: `from_value` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 585–593, 612–620 | Native preparation state/inspection/stop contracts; declaration/member from_map; declarations/fields: `from_map` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 594 | Native preparation state/inspection/stop contracts; declaration/member PrepareStop; declarations/fields: `PrepareStop` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 595 | Native preparation state/inspection/stop contracts; declaration/member PrepareStop.project; declarations/fields: `PrepareStop.project` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 596–599 | Native preparation state/inspection/stop contracts; declaration/member PrepareStop.id; declarations/fields: `PrepareStop.id` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 621 | Revision-fenced hold state/request; declarations/fields: `HoldState` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 622 | Revision-fenced hold state/request; declaration/member HoldState.active; declarations/fields: `HoldState.active` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 623–626 | Revision-fenced hold state/request; declaration/member HoldState.revision; declarations/fields: `HoldState.revision` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 627–634 | Revision-fenced hold state/request; declaration/member encode; declarations/fields: `encode` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 635–642, 689–698 | Revision-fenced hold state/request; declaration/member from_map; declarations/fields: `from_map` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 643–654 | Revision-fenced hold state/request; declaration/member HOLD_STATE_SPECS; declarations/fields: `HOLD_STATE_SPECS` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 655 | Revision-fenced hold state/request; declaration/member PrepareHold; declarations/fields: `PrepareHold` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 656 | Revision-fenced hold state/request; declaration/member PrepareHold.project; declarations/fields: `PrepareHold.project` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 657 | Revision-fenced hold state/request; declaration/member PrepareHold.hold; declarations/fields: `PrepareHold.hold` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 658–660 | Revision-fenced hold state/request; declaration/member PrepareHold.revision; declarations/fields: `PrepareHold.revision` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 661–676 | Revision-fenced hold state/request; declaration/member PREPARE_HOLD_SPECS; declarations/fields: `PREPARE_HOLD_SPECS` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 677–683 | Revision-fenced hold state/request; declaration/member validate; declarations/fields: `validate` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 684–688 | Revision-fenced hold state/request; declaration/member from_value; declarations/fields: `from_value` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 699 | Review/fix candidate checkout preparation contract; declarations/fields: `FactoryCandidate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 700 | Review/fix candidate checkout preparation contract; declaration/member FactoryCandidate.preparation; declarations/fields: `FactoryCandidate.preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 701 | Review/fix candidate checkout preparation contract; declaration/member FactoryCandidate.source_preparation; declarations/fields: `FactoryCandidate.source_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 702–704 | Review/fix candidate checkout preparation contract; declaration/member FactoryCandidate.bundle; declarations/fields: `FactoryCandidate.bundle` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 705–724 | Review/fix candidate checkout preparation contract; declaration/member FACTORY_CANDIDATE_SPECS; declarations/fields: `FACTORY_CANDIDATE_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 725–740 | Review/fix candidate checkout preparation contract; declaration/member validate; declarations/fields: `validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 741–746 | Review/fix candidate checkout preparation contract; declaration/member from_value; declarations/fields: `from_value` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 747–756 | Review/fix candidate checkout preparation contract; declaration/member from_map; declarations/fields: `from_map` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 757–759 | Review/fix candidate checkout preparation contract; declaration/member tests; declarations/fields: `tests` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 760, 906–956, 1045–1064, 1085–1161 | Preparation/decision/hold source assertions; declarations/fields: `DIGEST`, `approved_setup_validation_branches`, `prep_json`, `decode_messages_match_go_exactly` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 761 | Preparation/decision/hold source assertions; declaration/member COMMIT; declarations/fields: `COMMIT` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 762–763 | Preparation/decision/hold source assertions; declaration/member EFFECTS; declarations/fields: `EFFECTS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 764–767 | Preparation/decision/hold source assertions; declaration/member pid; declarations/fields: `pid` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 768–771 | Preparation/decision/hold source assertions; declaration/member fid; declarations/fields: `fid` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 772–775 | Preparation/decision/hold source assertions; declaration/member did; declarations/fields: `did` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 776–785 | Preparation/decision/hold source assertions; declaration/member requirements; declarations/fields: `requirements` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 786–794 | Preparation/decision/hold source assertions; declaration/member approval; declarations/fields: `approval` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 795–809 | Preparation/decision/hold source assertions; declaration/member preparation; declarations/fields: `preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 810–822 | Preparation/decision/hold source assertions; declaration/member setup; declarations/fields: `setup` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 823–865 | Preparation/decision/hold source assertions; declaration/member validators_match_go_shapes; declarations/fields: `validators_match_go_shapes` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 866–870 | Preparation/decision/hold source assertions; declaration/member setup_digest_matches_reference; declarations/fields: `setup_digest_matches_reference` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 871–905, 957–975, 1076–1084 | Source assertion of Checkout allocation and preparation; declarations/fields: `preparation_validation_branches`, `prepare_validates_digest_binding`, `prepare_decode_round_trip` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 976–1001 | Source assertion of Maintenance holds; declarations/fields: `address_and_hold_validation` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 1002–1044 | Source assertion of Candidate verification assessment; declarations/fields: `candidate_validation_branches` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1065–1075 | Preparation/decision/hold source assertions; declaration/member prepare_json; declarations/fields: `prepare_json` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1162–1209 | Preparation/decision/hold source assertions; declaration/member state_encodes_honor_omitempty; declarations/fields: `state_encodes_honor_omitempty` |

<a id="coverage-00ec6f2cd3fa"></a>

<a id="rustsoda-hostsrcpreparers-1"></a>

## [rust/soda-host/src/prepare.rs](../../../../../rust/soda-host/src/prepare.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–17 | Protected exact checkout paths and file checks; declarations/fields: `FACTORY_INSPECT_LIMIT`, `path_is_abs`, `path_clean`, `path_join` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 18–20 | Protected exact checkout paths and file checks; declaration/member FACTORY_INSPECT_LIMIT; declarations/fields: `FACTORY_INSPECT_LIMIT` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 21–25 | Protected exact checkout paths and file checks; declaration/member path_is_abs; declarations/fields: `path_is_abs` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 26–80 | Protected exact checkout paths and file checks; declaration/member path_clean; declarations/fields: `path_clean` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 81–89 | Protected exact checkout paths and file checks; declaration/member path_join; declarations/fields: `path_join` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 90–119 | Native user namespace/id-map admission; declarations/fields: `prepare_id_map` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 120–128 | Preparation layout and approved file/tool predicates; declarations/fields: `preparation_paths` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 129–140 | Preparation layout and approved file/tool predicates; declaration/member single_line; declarations/fields: `single_line` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 141–147 | Preparation layout and approved file/tool predicates; declaration/member valid_resolved_tool_path; declarations/fields: `valid_resolved_tool_path` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 148–166 | Preparation layout and approved file/tool predicates; declaration/member APPROVE_RESPONSE_SPECS; declarations/fields: `APPROVE_RESPONSE_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 167–181 | Preparation layout and approved file/tool predicates; declaration/member STOP_RESPONSE_SPECS; declarations/fields: `STOP_RESPONSE_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 182–190 | Preparation layout and approved file/tool predicates; declaration/member HOLD_RESPONSE_SPECS; declarations/fields: `HOLD_RESPONSE_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 191–201 | Preparation layout and approved file/tool predicates; declaration/member INSPECTION_HOLD_SPECS; declarations/fields: `INSPECTION_HOLD_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 202–220 | Preparation layout and approved file/tool predicates; declaration/member INSPECTION_VERIFIED_SPECS; declarations/fields: `INSPECTION_VERIFIED_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 221–295 | Preparation layout and approved file/tool predicates; declaration/member HELPER_INSPECTION_SPECS; declarations/fields: `HELPER_INSPECTION_SPECS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 296–320 | Preparation layout and approved file/tool predicates; declaration/member SAVED_REQUEST_SPECS; declarations/fields: `SAVED_REQUEST_SPECS` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 321 | Fixed-role launcher environment evidence; declarations/fields: `LauncherEvidence` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 322 | Fixed-role launcher environment evidence; declaration/member LauncherEvidence.uid; declarations/fields: `LauncherEvidence.uid` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 323 | Fixed-role launcher environment evidence; declaration/member LauncherEvidence.login; declarations/fields: `LauncherEvidence.login` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 324 | Fixed-role launcher environment evidence; declaration/member LauncherEvidence.groups; declarations/fields: `LauncherEvidence.groups` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 325–327 | Fixed-role launcher environment evidence; declaration/member LauncherEvidence.refusal; declarations/fields: `LauncherEvidence.refusal` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 328–332 | Fixed-role launcher environment evidence; declaration/member quote_bytes_base64; declarations/fields: `quote_bytes_base64` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 333–350 | Approved file base64 emission; declarations/fields: `encode_files_object` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 351–366 | Project-local helper protocol; declarations/fields: `factory_helper` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 367–425 | Target Project container admission; declarations/fields: `prepare_container` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 426–443 | Inspect/prepare/approved snapshot materialization; declarations/fields: `inspect_preparation_state` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 444–483 | Inspect/prepare/approved snapshot materialization; declaration/member prepare; declarations/fields: `prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 484–508 | Inspect/prepare/approved snapshot materialization; declaration/member approve_preparation; declarations/fields: `approve_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 509–537 | Inspect/prepare/approved snapshot materialization; declaration/member ERR; declarations/fields: `ERR` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 538–551 | Fixed-role execution; declarations/fields: `role_exec` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 552–557 | Fixed-role execution; declaration/member must_role_exec; declarations/fields: `must_role_exec` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 558–614 | Role checkout clone and exact Git head; declarations/fields: `clone_preparation_source` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 615–637 | Role checkout clone and exact Git head; declaration/member confirm_preparation_head; declarations/fields: `confirm_preparation_head` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 638–732 | Role launcher native environment; declarations/fields: `verify_launcher_environment` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 733–782 | Native tools observation; declarations/fields: `resolve_preparation_tools` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 783–819 | Record/inspect/stop preparation evidence; declarations/fields: `record_preparation_tools` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 820–830 | Record/inspect/stop preparation evidence; declaration/member inspect_preparation; declarations/fields: `inspect_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 831–841 | Record/inspect/stop preparation evidence; declaration/member stop_preparation; declarations/fields: `stop_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 842–858 | Record/inspect/stop preparation evidence; declaration/member ERR; declarations/fields: `ERR` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 859–881 | Maintenance hold native marker operation; declarations/fields: `hold_preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 882–918 | Protected candidate preparation and snapshot reuse; declarations/fields: `prepare_candidate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 919–968 | Protected candidate preparation and snapshot reuse; declaration/member candidate_protected_file; declarations/fields: `candidate_protected_file` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 969–1003 | Protected candidate preparation and snapshot reuse; declaration/member candidate_approved_files; declarations/fields: `candidate_approved_files` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1004–1054, 1056–1106 | Protected candidate preparation and snapshot reuse; declaration/member ERR; declarations/fields: `ERR` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1055 | Protected candidate preparation and snapshot reuse; declaration/member map_preparation_state; declarations/fields: `map_preparation_state` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1107–1114 | Protected candidate preparation and snapshot reuse; declaration/member single_line_trimmed; declarations/fields: `single_line_trimmed` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1115–1120 | Protected candidate preparation and snapshot reuse; declaration/member tests; declarations/fields: `tests` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1121 | Protected candidate preparation and snapshot reuse; declaration/member DIGEST; declarations/fields: `DIGEST` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1122 | Protected candidate preparation and snapshot reuse; declaration/member COMMIT; declarations/fields: `COMMIT` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1123–1124 | Protected candidate preparation and snapshot reuse; declaration/member EFFECTS; declarations/fields: `EFFECTS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1125–1126 | Protected candidate preparation and snapshot reuse; declaration/member MockCall; declarations/fields: `MockCall` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1127, 1344–1399, 1457–1465 | Native preparation source fixtures and assertions; declarations/fields: `Mock`, `map_state_validates_and_assembles`, `approve_response` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1128 | Native preparation source fixtures and assertions; declaration/member Mock.calls; declarations/fields: `Mock.calls` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1129–1132 | Native preparation source fixtures and assertions; declaration/member Mock.script; declarations/fields: `Mock.script` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1133–1141 | Native preparation source fixtures and assertions; declaration/member new; declarations/fields: `new` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1142–1160 | Native preparation source fixtures and assertions; declaration/member run; declarations/fields: `run` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1161–1164 | Native preparation source fixtures and assertions; declaration/member deadline; declarations/fields: `deadline` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1165–1174 | Native preparation source fixtures and assertions; declaration/member test_config; declarations/fields: `test_config` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1175–1178 | Native preparation source fixtures and assertions; declaration/member pid; declarations/fields: `pid` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1179–1182 | Native preparation source fixtures and assertions; declaration/member fid; declarations/fields: `fid` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1183–1190 | Native preparation source fixtures and assertions; declaration/member container_payload; declarations/fields: `container_payload` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1191–1202 | Native preparation source fixtures and assertions; declaration/member observation; declarations/fields: `observation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1203–1228 | Native preparation source fixtures and assertions; declaration/member preparation; declarations/fields: `preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1229–1241 | Native preparation source fixtures and assertions; declaration/member setup; declarations/fields: `setup` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1242–1278 | Native preparation source fixtures and assertions; declaration/member path_clean_matches_go_vectors; declarations/fields: `path_clean_matches_go_vectors` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1279–1304 | Native preparation source fixtures and assertions; declaration/member idmap_requires_shifted_usable_range; declarations/fields: `idmap_requires_shifted_usable_range` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1305–1316 | Native preparation source fixtures and assertions; declaration/member single_line_rules; declarations/fields: `single_line_rules` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1317–1331 | Native preparation source fixtures and assertions; declaration/member tool_paths_must_be_clean_and_pinned; declarations/fields: `tool_paths_must_be_clean_and_pinned` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1332–1343, 1400–1456, 1466–1541 | Source assertion of Checkout allocation and preparation; declarations/fields: `preparation_paths_layout`, `prepare_container_binds_isolated_target`, `prepare_runs_the_full_chain` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1542–1610 | Source assertion of Checkout allocation and preparation; declaration/member prepare_short_circuits_and_recovers; declarations/fields: `prepare_short_circuits_and_recovers` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1611–1664 | Source assertion of Checkout allocation and preparation; declaration/member prepare_skips_start_when_blocked; declarations/fields: `prepare_skips_start_when_blocked` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 1665–1789 | Source assertion of Maintenance holds; declarations/fields: `inspect_stop_hold_paths` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 1790–1886 | Source assertion of Candidate verification assessment; declarations/fields: `candidate_reuses_ready_source` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 1887–1955 | Source assertion of Candidate verification assessment; declaration/member candidate_rejects_bad_snapshots; declarations/fields: `candidate_rejects_bad_snapshots` |

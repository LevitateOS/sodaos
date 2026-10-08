# Backend project

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-92d8c9add75c"></a>

## [internal/project/factory.go](../../../../../internal/project/factory.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–47, 62–166; whole file; (declaration group); harnessVersion; ValidHarnessVersion; ValidHarnessFamily; FactoryRun; FactoryRun.Validate; FactoryPromptDigest; FactoryLaunch; FactoryLaunch.Validate | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 48–61, 167–192, 216–247; ValidFactoryPhase, factoryRunID, ValidFactoryRunID, FactoryState, FactoryUnitName, FactoryInspect, FactoryInspect.Validate, FactoryStop, FactoryStop.Validate | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Factory run lifecycle identity, state or native inspection/stop contract; declarations/fields: `ValidFactoryPhase`, `factoryRunID`, `ValidFactoryRunID`, `FactoryState`, `FactoryUnitName`, `FactoryInspect`, `FactoryInspect.Validate`, `FactoryStop`, `FactoryStop.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 193–205; FactoryRunPaths | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Assigned run checkout and input path derivation; declarations/fields: `FactoryRunPaths` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 206–215; FactoryCodexGuest | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Native provider consumer identity for factory execution; declarations/fields: `FactoryCodexGuest` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-25c9b65fc824"></a>

## [internal/project/factory_candidate.go](../../../../../internal/project/factory_candidate.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–57; file scaffold; FactoryCandidate; Validate; FactoryCandidateInspect; FactoryCandidateState | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-49b6c4ca4634"></a>

## [internal/project/factory_candidate_test.go](../../../../../internal/project/factory_candidate_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; file scaffold; TestFactoryCandidateRequiresFreshBoundedPreparation | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryCandidateRequiresFreshBoundedPreparation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d5bcb7f11505"></a>

## [internal/project/factory_export.go](../../../../../internal/project/factory_export.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–58; file scaffold; MaxFactoryExportBundle; FactoryExport; Validate; FactoryExportState | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8832390ea3b5"></a>

## [internal/project/factory_export_test.go](../../../../../internal/project/factory_export_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30; file scaffold; TestFactoryExportValidates | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryExportValidates — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-39acd1000d1c"></a>

## [internal/project/factory_harness.go](../../../../../internal/project/factory_harness.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; file scaffold; FactoryHarnessPin; Validate | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: FactoryHarnessPin; Current declaration duty: Validate — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7092c9c4be5f"></a>

## [internal/project/factory_harness_test.go](../../../../../internal/project/factory_harness_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–34; file scaffold; TestFactoryHarnessPinValidate | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryHarnessPinValidate — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d77de6717cbc"></a>

## [internal/project/factory_output.go](../../../../../internal/project/factory_output.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; file scaffold; FactoryOutput; Validate; FactoryOutputState | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a7bbc9546f76"></a>

## [internal/project/factory_output_test.go](../../../../../internal/project/factory_output_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; file scaffold; TestFactoryOutputValidation | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryOutputValidation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-57f958e846a3"></a>

## [internal/project/factory_test.go](../../../../../internal/project/factory_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–68; whole file; factoryRunFixture; TestFactoryRunValidation; TestFactoryLaunchBindsPromptDigest | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 69–93; TestFactoryRunPathsAreFixed | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Factory role checkout/run path assertions; declarations/fields: `TestFactoryRunPathsAreFixed` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 94–109; TestFactoryPhasesAndAddresses | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Factory lifecycle protocol assertions; declarations/fields: `TestFactoryPhasesAndAddresses` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-775c7088065e"></a>

## [internal/project/grants.go](../../../../../internal/project/grants.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37; whole file; EnvironmentGrant; EnvironmentGrant.Validate | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Record, DTO or interface contract EnvironmentGrant for Repository association and creation; declarations/fields: `EnvironmentGrant`; EnvironmentGrant.Validate — Repository association and creation; declarations/fields: `EnvironmentGrant.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 38–67; RequirementDecision, RequirementDecision.Validate, RequirementDecision.Ref | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Exact preparation requirement decisions; declarations/fields: `RequirementDecision`, `RequirementDecision.Validate`, `RequirementDecision.Ref` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 68–93; ApprovalDecision, ApprovalDecision.Validate, ApprovalDecision.Ref | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Privileged preparation approval decisions; declarations/fields: `ApprovalDecision`, `ApprovalDecision.Validate`, `ApprovalDecision.Ref` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-b15c54c23df1"></a>

## [internal/project/grants_test.go](../../../../../internal/project/grants_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35; whole file; grantTestProfile; TestEnvironmentGrantValidation | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Fixture/protocol support grantTestProfile; declarations/fields: `grantTestProfile`; Assertions TestEnvironmentGrantValidation: %s: grant accepted; declarations/fields: `TestEnvironmentGrantValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 36–54; TestRequirementDecisionValidation | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Preparation requirement shape assertions; declarations/fields: `TestRequirementDecisionValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 55–73; TestApprovalDecisionValidation | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Privileged effect approval shape assertions; declarations/fields: `TestApprovalDecisionValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-16a64cb32054"></a>

## [internal/project/preparation.go](../../../../../internal/project/preparation.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16, 27–117, 169–173, 176–177, 179–285, 293–330, 336–344, 360–378; whole file; (declaration group); ValidPreparePhase; ValidPreparationID; ValidDecisionID; ValidDigest; ValidCommit; ValidApprovedName; ValidToolName; Preparation; Preparation.Validate; ApprovedSetup, ApprovedSetup.Validate, SetupDigestOf; Prepare; Prepare.Validate; PrepareState; PrepareInspect; PrepareInspect.Validate; PrepareStop; PrepareStop.Validate; StoredPreparation; StoredPreparation.Validate | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 21 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 17–26; RoleCoder, RoleReviewer | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Fixed coder/reviewer factory role identities; declarations/fields: `RoleCoder`, `RoleReviewer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 118–134, 174; RequirementAcceptance, RequirementAcceptance.Validate; Preparation.Requirements | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Preparation requirement acceptance protocol; declarations/fields: `RequirementAcceptance`, `RequirementAcceptance.Validate`; Reference to independent exact requirement acceptance; declarations/fields: `Preparation.Requirements` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 135–150, 175; AdminApproval, AdminApproval.Validate; Preparation.Approval | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Approved exact setup/effect input protocol; declarations/fields: `AdminApproval`, `AdminApproval.Validate`; Reference to independent privileged effects approval; declarations/fields: `Preparation.Approval` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 151–168; LifecycleGrant, LifecycleGrant.Validate | [P05](../../slices/projects.md#p05-project-startstop) | retained | Standing Project lifecycle execution grant; declarations/fields: `LifecycleGrant`, `LifecycleGrant.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 178, 286–292; Preparation.Tools; ResolvedTool | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Prepared required tool references; declarations/fields: `Preparation.Tools`; Prepared pinned tool identity projection; declarations/fields: `ResolvedTool` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 331–335, 345–359, 379–390; HoldState, PrepareHold, PrepareHold.Validate, MaintenanceHold, MaintenanceHold.Validate | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Preparation admission hold state and command protocol; declarations/fields: `HoldState`, `PrepareHold`, `PrepareHold.Validate`, `MaintenanceHold`, `MaintenanceHold.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-19da41ea7b45"></a>

## [internal/project/preparation_test.go](../../../../../internal/project/preparation_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–32, 44–127, 148–157; whole file; (declaration group); testAcceptance; testApproval; testPreparation; TestPreparationRequiresBothApprovals; TestPreparationRejectsUntrustedIdentities; TestApprovedSetupRequiresFixedEntrypoints; TestPreparePhasesAreKnown; TestStoredPreparationBindsStateIdentity | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 33–43; TestFactoryRolesAreFixed | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Fixed role validation assertions; declarations/fields: `TestFactoryRolesAreFixed` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 128–137; TestLifecycleGrantAndHoldValidate | [P05](../../slices/projects.md#p05-project-startstop) | retained | Lifecycle grant/hold assertions partitioned below; declarations/fields: `TestLifecycleGrantAndHoldValidate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 138–147; TestLifecycleGrantAndHoldValidate | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Maintenance hold shape assertions inside mixed lifecycle/hold test; declarations/fields: `TestLifecycleGrantAndHoldValidate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-2e47833e185a"></a>

## [internal/project/profile.go](../../../../../internal/project/profile.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; file scaffold; RockyHeadless; Profile; Validate; Decode | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cf4f5504efe8"></a>

## [internal/project/project.go](../../../../../internal/project/project.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 18–23; whole file; (declaration group); ValidID | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Declared identifiers/bounds (declaration group) for Repository association and creation; declarations/fields: `(declaration group)`; ValidID — Repository association and creation; declarations/fields: `ValidID` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 15, 24–26; loginName; ValidLogin | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Human Linux Project login shape; declarations/fields: `loginName`; Canonical human account login validation; declarations/fields: `ValidLogin` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 16–17, 27–30; imageID, containerID; ValidImageRef, ValidContainerID | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Pinned native image/container identity shapes; declarations/fields: `imageID`, `containerID`; Pinned runtime/image identity validation; declarations/fields: `ValidImageRef`, `ValidContainerID` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-69db73c3e604"></a>

## [internal/project/project_test.go](../../../../../internal/project/project_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; file scaffold; validProfile; TestProfileValidation; TestCreateValidation; TestWireShapeFrozen; TestIdentityHelpers | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Existing validation and test support. — Current source declarations. |
| 91–105; TestProjectAccessRequestValidation | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Validates the canonical Project privilege observation request. — Current source selector. |

<a id="coverage-7a4270c54f54"></a>

## [internal/project/takeover.go](../../../../../internal/project/takeover.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–78; file scaffold; TakeoverDirName; FactoryTakeover; Validate; TakeoverDestination; TakeoverResult; TakeoverExcluded; TakeoverSource | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-166572481011"></a>

## [internal/project/takeover_test.go](../../../../../internal/project/takeover_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–76; file scaffold; TestFactoryTakeoverValidates; TestTakeoverDestinationDerivesMemberPath; TestTakeoverResultBindsDestination; TestTakeoverExclusions; TestTakeoverSourcePinsFixedLayout | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b58cd1f97f0b"></a>

## [internal/project/types.go](../../../../../internal/project/types.go)

Current changed DTO section verified; earlier unaffected intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 7–20; Create; Create.Validate | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Repository association and creation contract; prior interval is a historical hint pending R02. |
| 22–28; Account | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Human membership/account creation contract. — Current source selector. |
| 30–51; ProjectAccessRequest; ProjectAccessStatus | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Canonical required-field request/status DTOs; the status has a required administrator observation. — Current source types and callers. |
| 27, 63–68, 82–96; Account.Keys; Connection, AccessKeys, AccessKeyState | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Explicit authorized-key input and native development SSH contract; earlier intervals are historical hints pending R02. |
| 53–61, 98–110; Environment, OSRelease, OSObservation | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Observed native runtime and Project OS DTO; earlier intervals are historical hints pending R02. |
| 59; Environment.IP | [N02](../../slices/networking.md#n02-project-lan-access) | retained | Observed native Project LAN IP field; prior selector is a historical hint pending R02. |
| 70–80; Lifecycle, LifecycleState | [P05](../../slices/projects.md#p05-project-startstop) | retained | Project Start/Stop contract; prior interval is a historical hint pending R02. |

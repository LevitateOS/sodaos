# Backend project

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-92d8c9add75c"></a>

## [internal/project/factory.go](../../../../../internal/project/factory.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 16–47, 74–84 | Declared identifiers/bounds (declaration group) for Assignment and dispatch; declarations/fields: `(declaration group)` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 48–61, 167–192, 216–247 | Factory run lifecycle identity, state or native inspection/stop contract; declarations/fields: `ValidFactoryPhase`, `factoryRunID`, `ValidFactoryRunID`, `FactoryState`, `FactoryUnitName`, `FactoryInspect`, `FactoryInspect.Validate`, `FactoryStop`, `FactoryStop.Validate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 62–66 | Declared identifiers/bounds harnessVersion for Assignment and dispatch; declarations/fields: `harnessVersion` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 67–69 | ValidHarnessVersion — Assignment and dispatch; declarations/fields: `ValidHarnessVersion` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 70–73 | ValidHarnessFamily — Assignment and dispatch; declarations/fields: `ValidHarnessFamily` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 85–99 | Record, DTO or interface contract FactoryRun for Assignment and dispatch; declarations/fields: `FactoryRun` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 100–135 | FactoryRun.Validate — Assignment and dispatch; declarations/fields: `FactoryRun.Validate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 136–143 | FactoryPromptDigest — Assignment and dispatch; declarations/fields: `FactoryPromptDigest` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 144–149 | Record, DTO or interface contract FactoryLaunch for Assignment and dispatch; declarations/fields: `FactoryLaunch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 150–166 | FactoryLaunch.Validate — Assignment and dispatch; declarations/fields: `FactoryLaunch.Validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 193–205 | Assigned run checkout and input path derivation; declarations/fields: `FactoryRunPaths` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 206–215 | Native provider consumer identity for factory execution; declarations/fields: `FactoryCodexGuest` |

<a id="coverage-57f958e846a3"></a>

## [internal/project/factory_test.go](../../../../../internal/project/factory_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–8 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 9–24 | Fixture/protocol support factoryRunFixture; declarations/fields: `factoryRunFixture` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 25–50 | Assertions TestFactoryRunValidation: invalid %s admitted; declarations/fields: `TestFactoryRunValidation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 51–68 | Assertions TestFactoryLaunchBindsPromptDigest: changed prompt reused assignment digest; declarations/fields: `TestFactoryLaunchBindsPromptDigest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 69–93 | Factory role checkout/run path assertions; declarations/fields: `TestFactoryRunPathsAreFixed` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 94–109 | Factory lifecycle protocol assertions; declarations/fields: `TestFactoryPhasesAndAddresses` |

<a id="coverage-775c7088065e"></a>

## [internal/project/grants.go](../../../../../internal/project/grants.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 14–22 | Record, DTO or interface contract EnvironmentGrant for Repository association and creation; declarations/fields: `EnvironmentGrant` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 23–37 | EnvironmentGrant.Validate — Repository association and creation; declarations/fields: `EnvironmentGrant.Validate` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 38–67 | Exact preparation requirement decisions; declarations/fields: `RequirementDecision`, `RequirementDecision.Validate`, `RequirementDecision.Ref` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 68–93 | Privileged preparation approval decisions; declarations/fields: `ApprovalDecision`, `ApprovalDecision.Validate`, `ApprovalDecision.Ref` |

<a id="coverage-b15c54c23df1"></a>

## [internal/project/grants_test.go](../../../../../internal/project/grants_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–7 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 8–14 | Fixture/protocol support grantTestProfile; declarations/fields: `grantTestProfile` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 15–35 | Assertions TestEnvironmentGrantValidation: %s: grant accepted; declarations/fields: `TestEnvironmentGrantValidation` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 36–54 | Preparation requirement shape assertions; declarations/fields: `TestRequirementDecisionValidation` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 55–73 | Privileged effect approval shape assertions; declarations/fields: `TestApprovalDecisionValidation` |

<a id="coverage-16a64cb32054"></a>

## [internal/project/preparation.go](../../../../../internal/project/preparation.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 17–26 | Fixed coder/reviewer factory role identities; declarations/fields: `RoleCoder`, `RoleReviewer` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 27–76, 86–95 | Declared identifiers/bounds (declaration group) for Checkout allocation and preparation; declarations/fields: `(declaration group)` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 77–85 | ValidPreparePhase — Checkout allocation and preparation; declarations/fields: `ValidPreparePhase` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 96–98 | ValidPreparationID — Checkout allocation and preparation; declarations/fields: `ValidPreparationID` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 99–101 | ValidDecisionID — Checkout allocation and preparation; declarations/fields: `ValidDecisionID` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 102–104 | ValidDigest — Checkout allocation and preparation; declarations/fields: `ValidDigest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 105–108 | ValidCommit — Checkout allocation and preparation; declarations/fields: `ValidCommit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 109–113 | ValidApprovedName — Checkout allocation and preparation; declarations/fields: `ValidApprovedName` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 114–117 | ValidToolName — Checkout allocation and preparation; declarations/fields: `ValidToolName` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 118–134 | Preparation requirement acceptance protocol; declarations/fields: `RequirementAcceptance`, `RequirementAcceptance.Validate` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 135–150 | Approved exact setup/effect input protocol; declarations/fields: `AdminApproval`, `AdminApproval.Validate` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 151–168 | Standing Project lifecycle execution grant; declarations/fields: `LifecycleGrant`, `LifecycleGrant.Validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 169–173, 176–177, 179–181 | Record, DTO or interface contract Preparation for Checkout allocation and preparation; declarations/fields: `Preparation` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 174 | Reference to independent exact requirement acceptance; declarations/fields: `Preparation.Requirements` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 175 | Reference to independent privileged effects approval; declarations/fields: `Preparation.Approval` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 178 | Prepared required tool references; declarations/fields: `Preparation.Tools` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 182–211 | Preparation.Validate — Checkout allocation and preparation; declarations/fields: `Preparation.Validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 212–247, 269–285 | Bounded exact approved input materialization contract; separate P09 decision is prerequisite; declarations/fields: `ApprovedSetup`, `ApprovedSetup.Validate`, `SetupDigestOf` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 248–252 | Record, DTO or interface contract Prepare for Checkout allocation and preparation; declarations/fields: `Prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 253–268 | Prepare.Validate — Checkout allocation and preparation; declarations/fields: `Prepare.Validate` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 286–292 | Prepared pinned tool identity projection; declarations/fields: `ResolvedTool` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 293–311 | Record, DTO or interface contract PrepareState for Checkout allocation and preparation; declarations/fields: `PrepareState` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 312–316 | Record, DTO or interface contract PrepareInspect for Checkout allocation and preparation; declarations/fields: `PrepareInspect` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 317–324 | PrepareInspect.Validate — Checkout allocation and preparation; declarations/fields: `PrepareInspect.Validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 325–330 | Record, DTO or interface contract PrepareStop for Checkout allocation and preparation; declarations/fields: `PrepareStop` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 331–335, 345–359, 379–390 | Preparation admission hold state and command protocol; declarations/fields: `HoldState`, `PrepareHold`, `PrepareHold.Validate`, `MaintenanceHold`, `MaintenanceHold.Validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 336–344 | PrepareStop.Validate — Checkout allocation and preparation; declarations/fields: `PrepareStop.Validate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 360–364 | Record, DTO or interface contract StoredPreparation for Checkout allocation and preparation; declarations/fields: `StoredPreparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 365–378 | StoredPreparation.Validate — Checkout allocation and preparation; declarations/fields: `StoredPreparation.Validate` |

<a id="coverage-19da41ea7b45"></a>

## [internal/project/preparation_test.go](../../../../../internal/project/preparation_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–7 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 8–16 | Fixture/protocol support (declaration group); declarations/fields: `(declaration group)` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 17–20 | Fixture/protocol support testAcceptance; declarations/fields: `testAcceptance` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 21–24 | Fixture/protocol support testApproval; declarations/fields: `testApproval` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 25–32 | Fixture/protocol support testPreparation; declarations/fields: `testPreparation` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 33–43 | Fixed role validation assertions; declarations/fields: `TestFactoryRolesAreFixed` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 44–62 | Assertions TestPreparationRequiresBothApprovals: valid preparation rejected: %v; declarations/fields: `TestPreparationRequiresBothApprovals` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 63–84 | Assertions TestPreparationRejectsUntrustedIdentities: case %d accepted; declarations/fields: `TestPreparationRejectsUntrustedIdentities` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 85–116 | Exact approved setup byte/path/materialization input assertions; declarations/fields: `TestApprovedSetupRequiresFixedEntrypoints` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 117–127 | Assertions TestPreparePhasesAreKnown: known phase rejected: %q; declarations/fields: `TestPreparePhasesAreKnown` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 128–137 | Lifecycle grant/hold assertions partitioned below; declarations/fields: `TestLifecycleGrantAndHoldValidate` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 138–147 | Maintenance hold shape assertions inside mixed lifecycle/hold test; declarations/fields: `TestLifecycleGrantAndHoldValidate` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 148–157 | Assertions TestStoredPreparationBindsStateIdentity: valid record rejected: %v; declarations/fields: `TestStoredPreparationBindsStateIdentity` |

<a id="coverage-cf4f5504efe8"></a>

## [internal/project/project.go](../../../../../internal/project/project.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 13–14, 18–20 | Declared identifiers/bounds (declaration group) for Repository association and creation; declarations/fields: `(declaration group)` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 15 | Human Linux Project login shape; declarations/fields: `loginName` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 16–17 | Pinned native image/container identity shapes; declarations/fields: `imageID`, `containerID` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 21–23 | ValidID — Repository association and creation; declarations/fields: `ValidID` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 24–26 | Canonical human account login validation; declarations/fields: `ValidLogin` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 27–30 | Pinned runtime/image identity validation; declarations/fields: `ValidImageRef`, `ValidContainerID` |

<a id="coverage-b58cd1f97f0b"></a>

## [internal/project/types.go](../../../../../internal/project/types.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–8 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 9–14 | Record, DTO or interface contract Create for Repository association and creation; declarations/fields: `Create` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 15–22 | Create.Validate — Repository association and creation; declarations/fields: `Create.Validate` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 23–26, 28–31 | Human membership/account creation contract; declarations/fields: `Account` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 27 | Explicit authorized-key input for new human account; declarations/fields: `Account.Keys` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 32–35, 37–40, 76–87 | Observed native runtime and Project OS DTO; declarations/fields: `Environment`, `OSRelease`, `OSObservation` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 36 | Observed native Project LAN IP field; declarations/fields: `Environment.IP` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 41–47, 60–75 | Own development SSH address/key contract; declarations/fields: `Connection`, `AccessKeys`, `AccessKeyState` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 48–59 | Project Start/Stop contract; declarations/fields: `Lifecycle`, `LifecycleState` |


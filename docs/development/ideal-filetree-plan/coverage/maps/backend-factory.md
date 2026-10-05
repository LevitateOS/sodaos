# Backend factory

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-edd20d1e6636"></a>

<a id="internalfactoryassignmentgo-1"></a>

## [internal/factory/assignment.go](../../../../../internal/factory/assignment.go)

Committed 018740f4 source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–23 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 24–32, 42–51 | Declared identifiers/bounds (declaration group) for Assignment and dispatch; declarations/fields: `(declaration group)` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 33–35 | Declared identifiers/bounds MaxDispatchAttempts for Assignment and dispatch; declarations/fields: `MaxDispatchAttempts` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 36–41 | ValidAssignmentStage — Assignment and dispatch; declarations/fields: `ValidAssignmentStage` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 52–66 | validAssignReason — Assignment and dispatch; declarations/fields: `validAssignReason` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 67–97 | Record, DTO or interface contract Assignment for Assignment and dispatch; declarations/fields: `Assignment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 98–173 | Assignment.Validate — Assignment and dispatch; declarations/fields: `Assignment.Validate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 174–183 | ValidPreparationRef — Assignment and dispatch; declarations/fields: `ValidPreparationRef` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 184–196 | Record, DTO or interface contract AssignmentResult for Assignment and dispatch; declarations/fields: `AssignmentResult` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 197–210 | AssignmentResult.Validate — Assignment and dispatch; declarations/fields: `AssignmentResult.Validate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 211–219 | AssignmentResult.ToResult — Assignment and dispatch; declarations/fields: `AssignmentResult.ToResult` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 220–231 | ResultFromHarness — Assignment and dispatch; declarations/fields: `ResultFromHarness` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 232–242 | ResultSynthesized — Assignment and dispatch; declarations/fields: `ResultSynthesized` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 243–247 | Declared identifiers/bounds ResultFence for Assignment and dispatch; declarations/fields: `ResultFence` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 248–284 | Select the last result-json opener; try candidate closing fences until strict bounded JSON and Result validation admit a coding result, preserving nested diff/code fences in quoted summary text; declarations/fields: `ParseHarnessResult` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 285–291 | Held/consumed/released capacity reservation state constants; declarations/fields: `(declaration group)` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 292–357 | Reservation and confirmed accounting record invariants; declarations/fields: `ValidReservationState`, `Reservation`, `Reservation.Validate`, `Usage`, `Usage.Validate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 358–366 | Record, DTO or interface contract PromptSource for Assignment and dispatch; declarations/fields: `PromptSource` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 367–386 | Record, DTO or interface contract PromptInputs for Assignment and dispatch; declarations/fields: `PromptInputs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 387–395 | fenceCollision — Assignment and dispatch; declarations/fields: `fenceCollision` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 396–440 | BuildDispatchPrompt — Assignment and dispatch; declarations/fields: `BuildDispatchPrompt` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 441–450 | writePromptSection — Assignment and dispatch; declarations/fields: `writePromptSection` |

<a id="coverage-4895a64c32cc"></a>

## [internal/factory/assignment_test.go](../../../../../internal/factory/assignment_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 12–29 | Fixture/protocol support testPrompt; declarations/fields: `testPrompt` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 30–46 | Fixture/protocol support testAssignment; declarations/fields: `testAssignment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 47–86 | Assertions TestAssignmentValidate: valid assignment refused: %v; declarations/fields: `TestAssignmentValidate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 87–113 | Assertions TestAssignmentFinishRequiresResult: finished assignment without result accepted; declarations/fields: `TestAssignmentFinishRequiresResult` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 114–133 | Assertions TestAssignmentResultExtendsRetainedValidation: reported result refused: %v; declarations/fields: `TestAssignmentResultExtendsRetainedValidation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 134–174 | Source assertions for result fence selection, strict result validation and rejection cases, including a completed result whose summary contains nested diff fences; declarations/fields: `TestParseHarnessResult` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 175–218 | Assertions TestBuildDispatchPrompt: prompt lacks %q; declarations/fields: `TestBuildDispatchPrompt` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 219–236 | Reservation/usage record validation assertions; declarations/fields: `TestReservationAndUsageValidate` |

<a id="coverage-c86c75410883"></a>

<a id="internalfactorychecksgo-1"></a>

## [internal/factory/checks.go](../../../../../internal/factory/checks.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 21–36, 47–63 | Declared identifiers/bounds (declaration group) for Candidate verification assessment; declarations/fields: `(declaration group)` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 37–46 | ValidCheckVerdict — Candidate verification assessment; declarations/fields: `ValidCheckVerdict` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 64–79 | validCheckReason — Candidate verification assessment; declarations/fields: `validCheckReason` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 80–115 | CheckResolution — Candidate verification assessment; declarations/fields: `CheckResolution` |
| [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) / active | 116–122, 192–248 | Native CI evidence DTO and integrity bounds; verdict remains F11; declarations/fields: `MaxObservedCheckContext`, `MaxObservedCheckState`, `ObservedCheck`, `ObservedCheck.Validate`, `ObservedChecks`, `ObservedChecks.Validate` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 123–135 | Record, DTO or interface contract CheckTarget for Candidate verification assessment; declarations/fields: `CheckTarget` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 136–150 | CheckTarget.Validate — Candidate verification assessment; declarations/fields: `CheckTarget.Validate` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 151–160 | ChecksDigest — Candidate verification assessment; declarations/fields: `ChecksDigest` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 161–168 | Record, DTO or interface contract AdoptedChecks for Candidate verification assessment; declarations/fields: `AdoptedChecks` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 169–191 | AdoptedChecks.Validate — Candidate verification assessment; declarations/fields: `AdoptedChecks.Validate` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 249–253 | Declared identifiers/bounds SnapshotPageBound for Candidate verification assessment; declarations/fields: `SnapshotPageBound` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 254–259 | Record, DTO or interface contract CheckResult for Candidate verification assessment; declarations/fields: `CheckResult` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 260–274 | CheckResult.Validate — Candidate verification assessment; declarations/fields: `CheckResult.Validate` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 275–278 | Declared identifiers/bounds CheckStateSuccess for Candidate verification assessment; declarations/fields: `CheckStateSuccess` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 279–303 | checkStateSeverity — Candidate verification assessment; declarations/fields: `checkStateSeverity` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 304–326 | Record, DTO or interface contract CheckAssessment for Candidate verification assessment; declarations/fields: `CheckAssessment` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 327–405 | CheckAssessment.Validate — Candidate verification assessment; declarations/fields: `CheckAssessment.Validate` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 406–480 | VerifyChecks — Candidate verification assessment; declarations/fields: `VerifyChecks` |

<a id="coverage-dac3ba29ba1f"></a>

<a id="internalfactorygrantsgo-1"></a>

## [internal/factory/grants.go](../../../../../internal/factory/grants.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 19, 24–26, 119–128, 305–308 | Declared identifiers/bounds (declaration group) for Repository factory policy; declarations/fields: `(declaration group)` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 20–21 | Native ref publish and PR creation operation kind identifiers; declarations/fields: `OpRefPublish`, `OpPRCreate` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 22 | Native review submission operation kind; declarations/fields: `OpReviewSubmit` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 23 | Conditional native merge operation kind; declarations/fields: `OpMerge` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 27–35 | ValidOperationKind — Repository factory policy; declarations/fields: `ValidOperationKind` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 36–37 | Established fast-forward-only merge eligibility method; declarations/fields: `MergeFastForward` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 38 | Declared identifiers/bounds MergeFastForward for Repository factory policy; declarations/fields: `MergeFastForward` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 39–42 | Shared immutable settings command ledger envelope; declarations/fields: `SettingsCommandType` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 43 | Specific command discriminator for separately owned authority; declarations/fields: `CommandPolicy` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 44 | Specific command discriminator for separately owned authority; declarations/fields: `CommandOperatorGrant` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 45 | Specific command discriminator for separately owned authority; declarations/fields: `CommandCapacity` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 46 | Specific command discriminator for separately owned authority; declarations/fields: `CommandSponsorship` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 47 | Specific command discriminator for separately owned authority; declarations/fields: `CommandEnvironmentGrant` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 48 | Specific command discriminator for separately owned authority; declarations/fields: `CommandRequirement` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 49 | Specific command discriminator for separately owned authority; declarations/fields: `CommandApproval` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 50 | Specific command discriminator for separately owned authority; declarations/fields: `CommandReopen` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 51–52 | Specific command discriminator for separately owned authority; declarations/fields: `CommandAcceptance`, `CommandWithdrawal` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 53–80 | Factory lifecycle/shared command digest and replay discriminator; declarations/fields: `CommandPause`, `CommandResume`, `CommandRetry`, `CommandTakeover`, `SettingsCommandType`, `SettingsDigest` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 81–85 | SettingsDigest — Repository factory policy; declarations/fields: `SettingsDigest` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 86–91 | Record, DTO or interface contract ActorBindingRef for Repository factory policy; declarations/fields: `ActorBindingRef` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 92–101 | ActorBindingRef.Validate — Repository factory policy; declarations/fields: `ActorBindingRef.Validate` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 102–106 | Record, DTO or interface contract RoleSelection for Repository factory policy; declarations/fields: `RoleSelection` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 107–118 | RoleSelection.Validate — Repository factory policy; declarations/fields: `RoleSelection.Validate` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 129–142 | ValidTargetBranch — Repository factory policy; declarations/fields: `ValidTargetBranch` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 143–159 | Record, DTO or interface contract RepositoryPolicy for Repository factory policy; declarations/fields: `RepositoryPolicy` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 160–198 | RepositoryPolicy.Validate — Repository factory policy; declarations/fields: `RepositoryPolicy.Validate` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 199–217 | Appliance concurrent run and accounting capacity; declarations/fields: `Capacity`, `Capacity.Validate` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 218–239 | Repository execution permission granted by configured operator; declarations/fields: `OperatorGrant`, `OperatorGrant.Validate` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 240–278 | Connection-owner sponsorship and pinned delegation reference; declarations/fields: `Sponsorship`, `Sponsorship.Validate` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 279 | Dispatch envelope binds independently owned authority revisions; declarations/fields: `AuthorityRef` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 280 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.RequirementsID` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 281 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.ApprovalID` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 282 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.Policy` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 283 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.Operator` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 284 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.Capacity` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 285 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.Sponsorship` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 286 | Authority revision/reference, not a replacement decision owner; declarations/fields: `AuthorityRef.Environment` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 287–288 | Record, DTO or interface contract AuthorityRef for Repository factory policy; declarations/fields: `AuthorityRef` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 289–304 | Composite dispatch missing-authority reasons; consumers retain independent grant owners; declarations/fields: `MissingPolicy`, `MissingOperatorGrant`, `MissingCapacity`, `MissingSponsorship`, `MissingEnvironment`, `MissingPreparation`, `MissingDispatch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 309–318, 333–409 | Composite dispatch admission consumes separate grant owners; declarations/fields: `EffectiveAuthority`, `EvaluateAuthority`, `DispatchRegistration`, `DispatchRegistration.Validate` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 319–332 | Record, DTO or interface contract AuthorityInput for Repository factory policy; declarations/fields: `AuthorityInput` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 410–414 | Declared identifiers/bounds MaxCapturedDispatch for Repository factory policy; declarations/fields: `MaxCapturedDispatch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 415–438 | Dispatch withdrawal captures registered execution identities; declarations/fields: `Withdrawal`, `Withdrawal.Validate` |

<a id="coverage-97b8a6e35eaa"></a>

## [internal/factory/grants_test.go](../../../../../internal/factory/grants_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 11–28 | Fixture/protocol support grantTestPolicy; declarations/fields: `grantTestPolicy` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 29–56 | Assertions TestRepositoryPolicyValidation: %s: policy accepted; declarations/fields: `TestRepositoryPolicyValidation` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 57–63 | Capacity bound validation assertions in mixed grant test; declarations/fields: `TestGrantValidation` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 64–69 | Bounded operator execution grant validation assertions; declarations/fields: `TestGrantValidation` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 70–82 | Connection sponsorship/fixed allowed role validation assertions; declarations/fields: `TestGrantValidation` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 83–104 | Assertions TestSettingsCommandDigest: settings command accepted the operator digest; declarations/fields: `TestSettingsCommandDigest` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 105–143 | Composite dispatch authority assertions; declarations/fields: `TestEvaluateAuthority` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 144–153 | Withdrawal identity capture assertions; declarations/fields: `TestWithdrawalValidation` |

<a id="coverage-169582ce45a4"></a>

<a id="internalfactorymergego-1"></a>

## [internal/factory/merge.go](../../../../../internal/factory/merge.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1–21 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 22–30, 46–55 | Declared identifiers/bounds (declaration group) for Merge eligibility and completion; declarations/fields: `(declaration group)` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 31–41 | ValidMergeStage — Merge eligibility and completion; declarations/fields: `ValidMergeStage` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 42–45 | Declared identifiers/bounds MaxMergeAttempts for Merge eligibility and completion; declarations/fields: `MaxMergeAttempts` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 56–68 | validMergeReason — Merge eligibility and completion; declarations/fields: `validMergeReason` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 69–95 | MergeResolution — Merge eligibility and completion; declarations/fields: `MergeResolution` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 96–123 | Record, DTO or interface contract MergeOperation for Merge eligibility and completion; declarations/fields: `MergeOperation` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 124–168 | MergeOperation.Validate — Merge eligibility and completion; declarations/fields: `MergeOperation.Validate` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 169–203 | MergeOperation.validateLinks — Merge eligibility and completion; declarations/fields: `MergeOperation.validateLinks` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 204–212 | MergeOperationID — Merge eligibility and completion; declarations/fields: `MergeOperationID` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 213–250 | Record, DTO or interface contract Merge for Merge eligibility and completion; declarations/fields: `Merge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 251–347 | Merge.Validate — Merge eligibility and completion; declarations/fields: `Merge.Validate` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 348–371 | Record, DTO or interface contract MergeWork for Merge eligibility and completion; declarations/fields: `MergeWork` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 372–392 | Record, DTO or interface contract MergeIntent for Merge eligibility and completion; declarations/fields: `MergeIntent` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 393–404 | MergeWork.Intent — Merge eligibility and completion; declarations/fields: `MergeWork.Intent` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 405–417 | MergeIntent.Apply — Merge eligibility and completion; declarations/fields: `MergeIntent.Apply` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 418–434 | MergeWork.ValidateTarget — Merge eligibility and completion; declarations/fields: `MergeWork.ValidateTarget` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 435–436 | MergeIntent.Validate — Merge eligibility and completion; declarations/fields: `MergeIntent.Validate` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 437–470 | MergeIntent.validate — Merge eligibility and completion; declarations/fields: `MergeIntent.validate` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 471–478 | MergeWork.ValidateObservation — Merge eligibility and completion; declarations/fields: `MergeWork.ValidateObservation` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 479–489 | MergeWork.Validate — Merge eligibility and completion; declarations/fields: `MergeWork.Validate` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 490–520 | VerifyMergeCheckEvidence — Merge eligibility and completion; declarations/fields: `VerifyMergeCheckEvidence` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 521–570 | Conditional native merge observation/receipt DTO; declarations/fields: `MergeObservation`, `MergeConfirmation`, `MergeOutcome` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 571–579 | Record, DTO or interface contract MergeWithdrawal for Merge eligibility and completion; declarations/fields: `MergeWithdrawal` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 580–586 | MergeAuthRevision — Merge eligibility and completion; declarations/fields: `MergeAuthRevision` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 587–592 | MergeTargetChanged — Merge eligibility and completion; declarations/fields: `MergeTargetChanged` |

<a id="coverage-d2c88094d9be"></a>

<a id="internalfactorypublicationgo-1"></a>

## [internal/factory/publication.go](../../../../../internal/factory/publication.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1–21 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 22–30, 46–59, 76–83, 96–104, 117–123 | Declared identifiers/bounds (declaration group) for Publication progression; declarations/fields: `(declaration group)` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 31–41 | ValidPublicationStage — Publication progression; declarations/fields: `ValidPublicationStage` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 42–45 | Declared identifiers/bounds MaxPublicationAttempts for Publication progression; declarations/fields: `MaxPublicationAttempts` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 60–75 | validPublishReason — Publication progression; declarations/fields: `validPublishReason` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 84–95 | ValidOpEffect — Publication progression; declarations/fields: `ValidOpEffect` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 105–116 | ValidOpCancellation — Publication progression; declarations/fields: `ValidOpCancellation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 124–135 | ValidOpCompletion — Publication progression; declarations/fields: `ValidOpCompletion` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 136–143 | Declared identifiers/bounds MaxPublicationReceipt for Publication progression; declarations/fields: `MaxPublicationReceipt` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 144–170 | Record, DTO or interface contract PublicationOperation for Publication progression; declarations/fields: `PublicationOperation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 171–215 | PublicationOperation.Validate — Publication progression; declarations/fields: `PublicationOperation.Validate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 216–251 | PublicationOperation.validateLinks — Publication progression; declarations/fields: `PublicationOperation.validateLinks` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 252–269 | ValidPublicationOperationID — Publication progression; declarations/fields: `ValidPublicationOperationID` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 270–280 | PublicationOperationID — Publication progression; declarations/fields: `PublicationOperationID` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 281–289 | PublicationBranch — Publication progression; declarations/fields: `PublicationBranch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 290–324 | Record, DTO or interface contract Publication for Publication progression; declarations/fields: `Publication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 325–479 | Publication.Validate — Publication progression; declarations/fields: `Publication.Validate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 480–506 | Record, DTO or interface contract PublicationWork for Publication progression; declarations/fields: `PublicationWork` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 507–510 | Declared identifiers/bounds MaxPublicationBody for Publication progression; declarations/fields: `MaxPublicationBody` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 511–529 | Record, DTO or interface contract PublicationIntent for Publication progression; declarations/fields: `PublicationIntent` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 530–541 | PublicationWork.Intent — Publication progression; declarations/fields: `PublicationWork.Intent` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 542–552 | PublicationIntent.Apply — Publication progression; declarations/fields: `PublicationIntent.Apply` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 553–554 | PublicationIntent.Validate — Publication progression; declarations/fields: `PublicationIntent.Validate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 555–593 | PublicationIntent.validate — Publication progression; declarations/fields: `PublicationIntent.validate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 594–613 | PublicationWork.ValidateObservation — Publication progression; declarations/fields: `PublicationWork.ValidateObservation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 614–622 | PublicationWork.Validate — Publication progression; declarations/fields: `PublicationWork.Validate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 623–628 | PRTitleFor — Publication progression; declarations/fields: `PRTitleFor` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 629–642 | PRBodyFor — Publication progression; declarations/fields: `PRBodyFor` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 643–652, 693–716 | Native publication observation or conditional effect receipt; declarations/fields: `PublicationObservation`, `BranchOutcome`, `PRCreationOutcome` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 653–662 | Record, DTO or interface contract PublicationWithdrawal for Publication progression; declarations/fields: `PublicationWithdrawal` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 663–679 | Record, DTO or interface contract OperationOutcome for Publication progression; declarations/fields: `OperationOutcome` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 680–681 | Record, DTO or interface contract PublicationRefusal for Publication progression; declarations/fields: `PublicationRefusal` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 682–686 | PublicationRefusal.Error — Publication progression; declarations/fields: `PublicationRefusal.Error` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 687–688 | Record, DTO or interface contract PublicationWait for Publication progression; declarations/fields: `PublicationWait` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 689–692 | PublicationWait.Error — Publication progression; declarations/fields: `PublicationWait.Error` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 717–723 | PublishBranchName — Publication progression; declarations/fields: `PublishBranchName` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 724–729 | AuthRevisionFor — Publication progression; declarations/fields: `AuthRevisionFor` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 730–736 | RefHead — Publication progression; declarations/fields: `RefHead` |

<a id="coverage-9ebf8fcb5ebd"></a>

## [internal/factory/review_native.go](../../../../../internal/factory/review_native.go)

Committed 018740f4 source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 12–32 | Record, DTO or interface contract ReviewWork for Independent review and correction; declarations/fields: `ReviewWork` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 33–44 | ReviewWork.ValidateTarget — Independent review and correction; declarations/fields: `ReviewWork.ValidateTarget` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 45–65 | ReviewWork.Validate — Independent review and correction; declarations/fields: `ReviewWork.Validate` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 66–80, 157–169 | Native exact-candidate review observation/effect receipt; declarations/fields: `ReviewObservation`, `ReviewOutcome` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 81–87 | ReviewAuthRevision — Independent review and correction; declarations/fields: `ReviewAuthRevision` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 88–91 | Declared identifiers/bounds ReviewReportFence for Independent review and correction; declarations/fields: `ReviewReportFence` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 92–100 | Record, DTO or interface contract ReviewReport for Independent review and correction; declarations/fields: `ReviewReport` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 101–119 | ReviewReport.Validate — Independent review and correction; declarations/fields: `ReviewReport.Validate` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 120–156 | Select the last review-json opener; try candidate closing fences until strict bounded JSON and ReviewReport validation admit an independent reviewer verdict, preserving nested diff/code fences in quoted body text; declarations/fields: `ParseReviewReport` |

<a id="coverage-7be34e721d87"></a>

## [internal/factory/review_role_test.go](../../../../../internal/factory/review_role_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 1–9 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 10–20 | Reviewer cannot publish candidate assertions; declarations/fields: `TestST10ReviewerCannotBecomePublication` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 21–59 | Source assertions for approve/request-changes review parsing, rejected malformed/unknown fields and stable auth revision, including an approve body containing nested diff fences; declarations/fields: `TestParseReviewReport` |


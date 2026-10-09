# Backend factory

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-9177563dcdfb"></a>

## [internal/factory/acceptance.go](../../../../../internal/factory/acceptance.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–172; file scaffold; MaxAcceptedSources; SelectedSource; Validate; AcceptedPrerequisite; Acceptance; decimalNativeID; validSourceList; InitialAcceptanceID | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9f17c997dcad"></a>

## [internal/factory/acceptance_test.go](../../../../../internal/factory/acceptance_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; file scaffold; acceptanceFixture; TestAcceptanceValidateAcceptsFullDecision; TestAcceptanceValidateRejectsBadShape; TestInitialAcceptanceIDIsStable | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5fc5c16d0f9b"></a>

## [internal/factory/allowance.go](../../../../../internal/factory/allowance.go)

At `165d0d03`, this canonical domain owner distinguishes terminal `Closed` from active clock/slot custody and rejects reactivation or correction consumption on a closed root. Store and coordinator owners implement retirement and native outcome joins; the domain test alone does not prove those workflows.

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–146; file scaffold; AttemptLimits; DefaultAttemptLimits; Validate; EffectiveAttemptLimits; AttemptAllowance; RemainingSeconds; Checkpoint; ConsumeCorrection | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a3d80b07dd64"></a>

## [internal/factory/allowance_test.go](../../../../../internal/factory/allowance_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; file scaffold; allowanceFixture; TestAttemptLimitsDefaultsAndOwnerValues; TestAttemptAllowanceExcludesOnlyConfirmedQueuedIntervals; TestCorrectionConsumptionChargesFailuresAndReplays; TestAttemptTimeExhaustionStopsCorrections; TestClosedAttemptAllowanceCanStopButCannotResumeOrConsumeCorrection | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-edd20d1e6636"></a>
<a id="internalfactoryassignmentgo-1"></a>

## [internal/factory/assignment.go](../../../../../internal/factory/assignment.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–93, 170–172; file scaffold; MaxDispatchAttempts; ValidAssignmentStage; validAssignReason; Assignment; ValidPreparationRef | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 94–169; Validate | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: Validate — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0ba2bd46a4e5"></a>

## [internal/factory/assignment_resources.go](../../../../../internal/factory/assignment_resources.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; file scaffold; ValidReservationState; Reservation; Validate; Usage | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-83bd8db19543"></a>

## [internal/factory/assignment_result.go](../../../../../internal/factory/assignment_result.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113; file scaffold; AssignmentResult; Validate; ToResult; ResultFromHarness; ResultSynthesized; ResultFence; ParseHarnessResult | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4895a64c32cc"></a>

## [internal/factory/assignment_test.go](../../../../../internal/factory/assignment_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–218; whole file; testPrompt; testAssignment; TestAssignmentValidate; TestAssignmentFinishRequiresResult; TestAssignmentResultExtendsRetainedValidation; TestParseHarnessResult; TestBuildDispatchPrompt | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 8 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 219–236; TestReservationAndUsageValidate | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Reservation/usage record validation assertions; declarations/fields: `TestReservationAndUsageValidate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-c86c75410883"></a>
<a id="internalfactorychecksgo-1"></a>

## [internal/factory/checks.go](../../../../../internal/factory/checks.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–36; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 37–182; ValidCheckVerdict; validCheckReason; CheckResolution; CheckTarget; Validate; ChecksDigest; AdoptedChecks | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: ValidCheckVerdict; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7b04c61ad7bb"></a>

## [internal/factory/checks_assessment.go](../../../../../internal/factory/checks_assessment.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–187; file scaffold; CheckAssessment; Validate; VerifyChecks | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6e991d3e95b0"></a>

## [internal/factory/checks_observation.go](../../../../../internal/factory/checks_observation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–122; file scaffold; MaxObservedCheckContext; MaxObservedCheckState; ObservedCheck; Validate; ObservedChecks; SnapshotPageBound; CheckResult; CheckStateSuccess; checkStateSeverity | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-77241e7820a2"></a>

## [internal/factory/checks_test.go](../../../../../internal/factory/checks_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–253; file scaffold; checkTargetFixture; adoptedChecksFixture; checkPolicyFixture; observedChecksFixture; verifyCheckVerdict; TestVerifyChecksPass; TestVerifyChecksWaitOnPendingOrMissing; TestVerifyChecksFailClosed; TestVerifyChecksWorstStateWins; TestVerifyChecksRefusals; TestVerifyChecksRejectsMalformedInputs; TestChecksDigestIgnoresOrder; TestCheckAssessmentValidation | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-938acb4cc9c3"></a>

## [internal/factory/dispatch_prompt.go](../../../../../internal/factory/dispatch_prompt.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–258; file scaffold; PromptSource; PromptInputs; fenceCollision; BuildDispatchPrompt; writeRepositoryContext; writePromptSection | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Binds bounded repository context to the exact prompt scope and renders selected setup/check instructions, files and candidate diff into the digest-covered assignment. |

<a id="r02-current-path-internal-factory-dispatch-prompt-stage-go"></a>

## [internal/factory/dispatch_prompt_stage.go](../../../../../internal/factory/dispatch_prompt_stage.go)

Current source and its `BuildDispatchPrompt` caller inspected at `ff2670d0`; existing assignment and dispatch ownership remains in place.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| File scaffold; promptStage; validatePromptStage; writePromptStage; admitPromptText | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Selects coding, review or correction prompt policy, validates exact candidate/base and consolidated evidence, renders stage findings, and admits aggregate text before expansion. `BuildDispatchPrompt` invokes these duties and binds approved repository context into the final bounded prompt. [F07 source/development evidence](../../reviews/F07.md#current-prompt-and-preparation-observation-source-cuts) remains distinct from native/provider qualification. |

<a id="r02-current-path-internal-factory-dispatch-prompt-stage-test-go"></a>

## [internal/factory/dispatch_prompt_stage_test.go](../../../../../internal/factory/dispatch_prompt_stage_test.go)

Current source inspected at `ff2670d0`; this selective addition does not renew historical body-audit evidence.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| File scaffold; stagePromptInputs; stageRepositoryContext; stageCorrectionInputs; TestBuildDispatchPromptReviewerGetsOnlyExactCandidateReview; TestBuildDispatchPromptCorrectionBindsConsolidatedEvidence; TestBuildDispatchPromptBindsRepositoryContextScope; TestBuildDispatchPromptRefusesOversizedAggregateStageEvidence | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Exercises production prompt construction for role-specific report contracts, fence neutralization, exact evidence/context scope and aggregate input refusal. Helpers supply typed fixture inputs; they do not prove native preparation or provider execution. |

<a id="coverage-fb603e59c85e"></a>

## [internal/factory/effective_authority.go](../../../../../internal/factory/effective_authority.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–63; file scaffold; EvaluateAuthority | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: EvaluateAuthority — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dac3ba29ba1f"></a>
<a id="internalfactorygrantsgo-1"></a>

## [internal/factory/grants.go](../../../../../internal/factory/grants.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–26, 269–278, 293–299; file scaffold; EffectiveAuthority; DispatchRegistration | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: EffectiveAuthority; Current declaration duty: DispatchRegistration — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 27–62, 77–91, 102–106, 129–159, 239–268, 279–292, 308–312; ValidOperationKind; MergeFastForward; SettingsDigest; ActorBindingRef; RoleSelection; ValidTargetBranch; RepositoryPolicy; AuthorityRef; AuthorityInput; MaxCapturedDispatch | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Current declaration duty: ValidOperationKind; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 63–76, 92–101, 107–128, 160–198, 206–217, 226–238, 300–307, 313–355; SettingsCommandType; Validate; Withdrawal | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current declaration duty: SettingsCommandType; Current declaration duty: Validate; Current declaration duty: Withdrawal — `Withdrawal.Validate` covers the complete bounded active-cause and captured-ID checks. P05's verified-start consumer clears only its own cause while retaining this validated withdrawal receipt. |
| 199–205; Capacity | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: Capacity — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 218–225; OperatorGrant | [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) | retained | Current declaration duty: OperatorGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-97b8a6e35eaa"></a>

## [internal/factory/grants_test.go](../../../../../internal/factory/grants_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56, 83–104; whole file; grantTestPolicy; TestRepositoryPolicyValidation; TestSettingsCommandDigest | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 57–63; TestGrantValidation | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Capacity bound validation assertions in mixed grant test; declarations/fields: `TestGrantValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 64–69; TestGrantValidation | [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) | retained | Bounded operator execution grant validation assertions; declarations/fields: `TestGrantValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 70–82; TestGrantValidation | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Connection sponsorship/fixed allowed role validation assertions; declarations/fields: `TestGrantValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 105–143; TestEvaluateAuthority | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Composite dispatch authority assertions; declarations/fields: `TestEvaluateAuthority` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 144–153; TestWithdrawalValidation | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Withdrawal identity capture assertions; declarations/fields: `TestWithdrawalValidation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-ad9ecd63bd81"></a>

## [internal/factory/lifecycle.go](../../../../../internal/factory/lifecycle.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–186; file scaffold; ValidRepositoryAction; ValidRunAction; RunStopOutcome; PauseReceipt; Validate; ResumeReceipt; RetryDecision; TakeoverRecord; StartVerification | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1c46556ca423"></a>

## [internal/factory/lifecycle_test.go](../../../../../internal/factory/lifecycle_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–128; file scaffold; TestControlActionsValidate; TestLifecycleCommandsRideSettingsLedger; pauseFixture; TestPauseReceiptKeepsActionAndEffectSeparate; TestRetryDecisionStaysQueued; TestResumeReceiptValidates; TestTakeoverRecordDerivesMemberDestination; TestStartVerificationValidates | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-169582ce45a4"></a>
<a id="internalfactorymergego-1"></a>

## [internal/factory/merge.go](../../../../../internal/factory/merge.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–248; ValidMergeStage; MaxMergeAttempts; validMergeReason; MergeResolution; Merge; Validate; MergeWithdrawal; MergeAuthRevision; MergeTargetChanged | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: ValidMergeStage; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f8345b7d2b05"></a>

## [internal/factory/merge_evidence.go](../../../../../internal/factory/merge_evidence.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–87; file scaffold; VerifyMergeCheckEvidence; MergeObservation; MergeConfirmation; MergeOutcome | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3582b850fb14"></a>

## [internal/factory/merge_operation.go](../../../../../internal/factory/merge_operation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–268; file scaffold; MergeOperation; Validate; validateLinks; MergeOperationID; MergeWork; MergeIntent; Intent; Apply; ValidateTarget; validate; ValidateObservation | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-90709be52019"></a>

## [internal/factory/merge_test.go](../../../../../internal/factory/merge_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–277; file scaffold; testMerge; TestMergeValidatesOpen; TestMergeRejectsMalformed; testMergeIntent; testMergeWork; TestMergeIntentRoundTrip; TestMergeOperationRejectsMalformed; testMergedMerge; TestMergeMergedRequiresConfirmedCompletion; TestMergeTerminalStages; testMergePolicy; testMergeAssessment; TestVerifyMergeCheckEvidence; TestMergeTargetChanged | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d2c88094d9be"></a>
<a id="internalfactorypublicationgo-1"></a>

## [internal/factory/publication.go](../../../../../internal/factory/publication.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–301, 357–383, 407–428; ValidPublicationStage; MaxPublicationAttempts; validPublishReason; ValidOpCompletion; MaxPublicationReceipt; Publication (base fields and non-review validation); Validate; PublicationWithdrawal; OperationOutcome; PublishBranchName; AuthRevisionFor; RefHead | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Publication validation includes its full terminal-state checks; withdrawal/outcome records and shared publication helpers retain their existing F09 duties. The review-history field and its admission policy are assigned to F10 below. |
| 96–107; MaxPublicationReviewOperations; MaxPublicationReviewBytes; 119; Publication.ReviewOperations; 235; review-history validation call; 302–344; Publication.CanAppendReviewOperation; Publication.validateReviewOperations | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Review admission validates linked-publication state, unique run operations, and the bounded retained-history/receipt budget before work is persisted or submitted. The `Publication` struct remains principally F09; these review-history fields and validation responsibilities are F10. |
| 349–355, 385–405; PublicationObservation; BranchOutcome; PRCreationOutcome | [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) | retained | Native publication observations and branch/PR result records retain the Forgejo adapter contract. |

<a id="coverage-3a69dd5d491d"></a>

## [internal/factory/publication_correction.go](../../../../../internal/factory/publication_correction.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; file scaffold; CorrectionOps | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: CorrectionOps — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7344a0b53630"></a>

## [internal/factory/publication_intent.go](../../../../../internal/factory/publication_intent.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–173; file scaffold; PublicationWork; MaxPublicationBody; PublicationIntent; Intent; Apply; Validate; validate; ValidateObservation; PRTitleFor; PRBodyFor | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b57ede4500a5"></a>

## [internal/factory/publication_operation.go](../../../../../internal/factory/publication_operation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–195; file scaffold; ValidOpEffect; ValidOpCancellation; PublicationOperation; Validate; validateLinks; ValidPublicationOperationID; PublicationOperationID; PublicationBranch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2badb8c8f0e3"></a>

## [internal/factory/publication_refusal.go](../../../../../internal/factory/publication_refusal.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; file scaffold; PublicationRefusal; Error; PublicationWait | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-58e068698188"></a>

## [internal/factory/publication_test.go](../../../../../internal/factory/publication_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–115, 199–395; file scaffold; testPublicationRun; testPublication; TestPublicationValidatesOpen; TestPublicationRejectsMalformed; testPublicationIntent; publishedPublication; TestPublicationValidatesPublished; TestPublicationRejectsLookalikeLinks; TestPublicationValidatesTerminalStages; TestPublicationOperationRejectsMalformed; TestPublicationOperationIDShape; TestPublicationBranchShape; TestPublicationWorkValidates; TestPublicationKeepsCommittedPRWhileCompletionWaits; TestPublicationWithdrawalCannotGuessPendingEffect | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration and fixture duties for the remaining publication tests; the review-history admission regression is assigned separately to F10. |
| 116–197; TestPublicationBoundsReviewHistoryAdmission | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Exercises record and byte bounds, receipt reservation, and persistence of admitted bounded review history. |

<a id="coverage-cb771a4abcf6"></a>

## [internal/factory/readiness.go](../../../../../internal/factory/readiness.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–348; file scaffold; ValidReadiness; validBlockerCode; BlockerResolution; Blocker; validBlockerDetail; Validate; PrereqSatisfaction; IssueControl; MaxControlBlockers; DependenceRef; FingerprintPrereq; FingerprintInput; Fingerprint; AuthorityFingerprint | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e4b1825c789f"></a>

## [internal/factory/readiness_test.go](../../../../../internal/factory/readiness_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–166; file scaffold; validControl; TestIssueControlValidate; TestBlockerResolutionsCoverEveryCode; fingerprintFixture; TestFingerprintStableOverOrdering; TestFingerprintChangesOnSemanticInputs; TestAuthorityFingerprintBindsGrants | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1eb698b03d61"></a>

## [internal/factory/result.go](../../../../../internal/factory/result.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35; file scaffold; Result; ResultSchema; Validate | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9ebf8fcb5ebd"></a>

## [internal/factory/review_native.go](../../../../../internal/factory/review_native.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65, 81–156; whole file; ReviewWork; ReviewWork.ValidateTarget; ReviewWork.Validate; ReviewAuthRevision; ReviewReportFence; ReviewReport; ReviewReport.Validate; ParseReviewReport | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 9 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 66–80, 157–169; ReviewObservation, ReviewOutcome | [G05](../../slices/forgejo-integration.md#g05-native-review-submission) | retained | Native exact-candidate review observation/effect receipt; declarations/fields: `ReviewObservation`, `ReviewOutcome` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-7be34e721d87"></a>

## [internal/factory/review_role_test.go](../../../../../internal/factory/review_role_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 21–59; whole file; TestParseReviewReport | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Source assertions for approve/request-changes review parsing, rejected malformed/unknown fields and stable auth revision, including an approve body containing nested diff fences; declarations/fields: `TestParseReviewReport` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 10–20; TestST10ReviewerCannotBecomePublication | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Reviewer cannot publish candidate assertions; declarations/fields: `TestST10ReviewerCannotBecomePublication` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-689f99dfa2ef"></a>

## [internal/factory/run.go](../../../../../internal/factory/run.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–115; file scaffold; digest; ValidDigest; Run; Validate; validateDeadline; validateProvenance; validateOutcome; validateCredentials; validateUnleasedCredential; validateIdentityBinding | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-63ede0c9c0b1"></a>

## [internal/factory/run_test.go](../../../../../internal/factory/run_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; file scaffold; testRun; TestRunRequiresExecutionAddress; TestCredentialAuthorityRequiresExactRunBinding; TestReconciledRunIsTerminal; TestCommandDigestBindsPayload | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c52b07211d2a"></a>

## [internal/factory/sponsorship.go](../../../../../internal/factory/sponsorship.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–47; file scaffold; Sponsorship; Validate | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: Sponsorship; Current declaration duty: Validate — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ae7171a7e557"></a>

## [internal/factory/types.go](../../../../../internal/factory/types.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–117; file scaffold; Outcome; NewID; ValidID; ValidCommit; ValidProjectID; validOutcome; Command; Validate; CommandDigest | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-395b1e08e543"></a>

## [internal/factory/views.go](../../../../../internal/factory/views.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–44; file scaffold; viewAttempt; ValidRunViewAttempt; RunView; Validate | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d68aa9c3bd4e"></a>

## [internal/factory/views_test.go](../../../../../internal/factory/views_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–26; file scaffold; TestRunViewValidation | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRunViewValidation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

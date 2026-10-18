# Factory control admission and dispatch

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

Selective readiness source delta at `07ab24bd` (production integration
`25ab7a48`): persisted readiness batches now belong to the current coordinator
and Store pages. The old sweep, recursive visitor, issue-list bridges and their
exclusive test machinery are retired. Historical map/body pins remain historical.

<a id="coverage-3f4cc6abd51f"></a>

## [internal/factory/control/assumption_advice.go](../../../../../internal/factory/control/assumption_advice.go)

Delta duty inspection at `93fb3513`. One judgment per pending assumption idea (S04): confident verdicts interrupt, uncertain/default proceed; decided ideas never re-judged. Consumes the judgment journal via AdvisePurpose.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; file scaffold; assumptionRubric; assumptionOptions; assumptionEvidence; JudgeAssumptions; settleChoice | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-8f8c25d59e33"></a>

## [internal/factory/control/assumption_advice_test.go](../../../../../internal/factory/control/assumption_advice_test.go)

Delta duty inspection at `93fb3513`. Pending-only, uncertainty, no-question-default, targeted-correction and exhaustion tests with injected choosers.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–197; file scaffold; assumptionChooser; Select; draftAssumptionFixture; TestJudgeAssumptionsChallengesOnlyPendingAssumptions; TestJudgeAssumptionsDefaultVerdictSettlesNonMaterial; TestJudgeAssumptionsSettlesNoQuestionDefault; TestJudgeAssumptionsTargetedCorrectionHoldsOneBranch; TestJudgeAssumptionsExhaustionAndMismatchFail | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 8 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-a40e1115bfda"></a>

## [internal/factory/control/consistency_advice.go](../../../../../internal/factory/control/consistency_advice.go)

Delta duty inspection at `93fb3513`. Coordinator consistency verdicts (S06): deterministic mismatch classes over proposal effects; incidental vocabulary consults clean; code findings skip judgment; genuine semantic overlap propagates the verdict.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–108; file scaffold; consistencyRubric; consistencyOptions; consistencyDetails; consistencyEvidence; JudgeConsistency; codeCheckAdvice | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 6 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-464604b9d617"></a>

## [internal/factory/control/consistency_advice_test.go](../../../../../internal/factory/control/consistency_advice_test.go)

Delta duty inspection at `93fb3513`. Clean-walkthrough, mismatch-class, incidental-vocabulary, semantic-overlap and code-findings tests.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–144; file scaffold; consistencyFixture; TestJudgeConsistencyConfirmsCleanWalkthrough; TestJudgeConsistencyNamesMismatchClasses; TestJudgeConsistencyIncidentalVocabularyConsultsClean; TestJudgeConsistencySemanticOverlapJudgmentNamesMismatch; TestJudgeConsistencyCodeFindingsSkipJudgment | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 6 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-7c806f10a610"></a>

## [internal/factory/control/consolidation_advice.go](../../../../../internal/factory/control/consolidation_advice.go)

Delta duty inspection at `93fb3513`. Coordinator consolidation verdicts (S05): compares actual alternatives, journals advice/failures, returns one coherent proposal; refuses unsupported content.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; file scaffold; consolidationRubric; consolidationEvidence; JudgeConsolidation | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 3 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-f22b9c8fee59"></a>

## [internal/factory/control/consolidation_advice_test.go](../../../../../internal/factory/control/consolidation_advice_test.go)

Delta duty inspection at `93fb3513`. Coherent-recommendation and unsupported-content refusal tests.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; file scaffold; TestJudgeConsolidationRecommendsOneCoherentProposal; TestJudgeConsolidationRefusesUnsupportedContent | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 2 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-cb1a904d77e2"></a>

## [internal/factory/control/context_bundle_advice.go](../../../../../internal/factory/control/context_bundle_advice.go)

R31 context-bundle selection (added `479e2cd2`, eligibility threaded `f491ebfa`, fingerprint shared `9c5e90de`). Fixed Choice rubric, eligibility-gated options, scope-bound state and funded selection over code-enforced eligibility, plus exact fingerprint recomputation for receipt lookup.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–103; file scaffold; contextBundleRubric; contextBundleOptions; contextBundleScope; contextBundleState; contextBundleStateFor; ContextBundleFingerprint; SelectContextBundle | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-cb2b115e88f3"></a>

## [internal/factory/control/context_bundle_advice_test.go](../../../../../internal/factory/control/context_bundle_advice_test.go)

R31 selection tests (added `479e2cd2`, eligibility and unoffered-hold cases `f491ebfa`, fingerprint binding `9c5e90de`). Funded serve/hold/exhaustion cases with injected choosers plus pure option/state/fingerprint units.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–171; file scaffold; bundleChooser; Select; contextBundleFixture; onceReserve; TestSelectContextBundleServesConfidentCurrent; TestSelectContextBundleServesConfidentAttempt; TestSelectContextBundleHoldsLowConfidence; TestSelectContextBundleHoldsUnofferedLineage; TestSelectContextBundleHoldsWhenChooserDown; TestSelectContextBundleExhaustedHoldsUnfunded; TestContextBundleFingerprintBindsEligibility; TestContextBundleOptionsOfferCurrentAndHold; TestContextBundleStateBindsScope | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 14 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-cb3c226f99a4"></a>

## [internal/factory/control/context_bundle_prepare.go](../../../../../internal/factory/control/context_bundle_prepare.go)

R31 preparation-time selection (added `f6835c02`). Rebuilds the staged capability from recorded preparation, joins parent content for eligibility and records one funded Choice per fingerprint from the priority call budget; best-effort, never fails the launch.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; file scaffold; PrepareContextBundleSelection | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-cb4d337a10b5"></a>

## [internal/factory/control/context_bundle_prepare_test.go](../../../../../internal/factory/control/context_bundle_prepare_test.go)

R31 preparation tests (added `f6835c02`). Funded-choice record with receipt reuse, chooser-down hold and nil-input refusal.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–68; file scaffold; TestPrepareContextBundleSelectionRecordsFundedChoice; TestPrepareContextBundleSelectionHoldsWhenChooserDown; TestPrepareContextBundleSelectionRefusesWithoutInputs | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-b7b1cd2777fd"></a>

## [internal/factory/control/dispatch_preparation.go](../../../../../internal/factory/control/dispatch_preparation.go)

Selective current source delta at `81bcdea0` (2026-10-11); historical body selectors below retain their recorded pins.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| File scaffold; Coordinator.prepareRetryCoder | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Accepted explicit Retry owns a bounded fresh coder preparation before run admission. Native setup copies the protected original source bundle; lost replies adopt the same identity and deadline. |
| Coordinator.reconcileRetryPreparations; retryPreparationInvalidated; Coordinator.retireRetryCoder | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Rejected, expired or invalidated setup retires through the existing native stop receipt without renewing the prior attempt allowance. Temporary Store/context errors preserve valid custody. |

<a id="coverage-ca852e847367"></a>

## [internal/factory/control/dispatch_preparation_test.go](../../../../../internal/factory/control/dispatch_preparation_test.go)

Selective current source delta at `81bcdea0` (2026-10-11); historical body selectors below retain their recorded pins.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| Whole current file; consumedCoderRetry and actual Retry regression subjects | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Real PostgreSQL dispatch regressions cover consumed checkout replacement, lost native reply adoption, rejected packet retirement, early planning invalidation, current sponsorship authority and custody retention across temporary Store/context failures. At `c9de3b26`, TestHoldReleaseWakesExistingRetryAfterProjectStop proves normal Stop/Start and hold release resume the same accepted Retry without a native revision change; unchanged authority performs no native observation. Native Rust source-bundle checks remain at their Rust owner; installed continuation is separate. |

<a id="coverage-bd92c58c05f9"></a>
<a id="internalfactorycontrolacceptancego-1"></a>

## [internal/factory/control/acceptance.go](../../../../../internal/factory/control/acceptance.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 219–265; file scaffold; replayAdmittedAcceptance; verifyAcceptanceVisibility | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: replayAdmittedAcceptance; Current declaration duty: verifyAcceptanceVisibility — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 38–218, 266–269; AcceptanceRefusal; Error; refuseAcceptance; AcceptanceIssueView; AcceptanceComment; AcceptanceEdge; AcceptanceEvidence; AcceptanceSource; AcceptanceReceipt; AdmitAcceptance; replayAcceptance; mustIssueIndex | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current declaration duty: AcceptanceRefusal; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-be7f8b799d47"></a>

## [internal/factory/control/acceptance_evidence.go](../../../../../internal/factory/control/acceptance_evidence.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106; file scaffold; readAcceptanceEvidence; verifyAcceptance | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: readAcceptanceEvidence; Current declaration duty: verifyAcceptance — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e9994ac10e3a"></a>

## [internal/factory/control/acceptance_initial.go](../../../../../internal/factory/control/acceptance_initial.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–87; file scaffold; AdmitInitialAcceptance | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: AdmitInitialAcceptance — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2bd3fe2cbf7a"></a>

## [internal/factory/control/acceptance_initial_test.go](../../../../../internal/factory/control/acceptance_initial_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–118; file scaffold; creationEvidence; creationHarness; TestAdmitInitialAcceptanceVerifiesCreation; TestAdmitInitialAcceptanceRefusesIneligible; TestAdmitInitialAcceptanceCannotOverwrite | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-89ded702d3fa"></a>

## [internal/factory/control/acceptance_status.go](../../../../../internal/factory/control/acceptance_status.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 26–219; file scaffold; AcceptanceValidity; AcceptanceStatus; acceptanceStatus; assessAcceptance; WithdrawAcceptance; replayWithdrawal | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–25; WithdrawalReceipt | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: WithdrawalReceipt — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6d369eeead3c"></a>

## [internal/factory/control/acceptance_status_test.go](../../../../../internal/factory/control/acceptance_status_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–157; file scaffold; TestAcceptanceStatusAssessesValidity; TestWithdrawAcceptanceLatchesAndReplays | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestAcceptanceStatusAssessesValidity; Current declaration duty: TestWithdrawAcceptanceLatchesAndReplays — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9c1eda10c1a7"></a>
<a id="internalfactorycontrolacceptance_testgo-1"></a>

## [internal/factory/control/acceptance_test.go](../../../../../internal/factory/control/acceptance_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 117–150; file scaffold; TestAdmitAcceptanceReplaysAfterNativeRevisionAdvance; TestAdmitAcceptanceReplayEnforcesCurrentVisibility | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestAdmitAcceptanceReplaysAfterNativeRevisionAdvance; Current declaration duty: TestAdmitAcceptanceReplayEnforcesCurrentVisibility — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 14–116, 151–254; stubAcceptanceSource; ReadAcceptanceEvidence; acceptanceDigests; acceptanceDecision; acceptanceEvidence; acceptanceHarness; acceptanceRefusal; TestAdmitAcceptanceVerifiesAndRecords; TestAdmitAcceptanceRefusesWithoutRecording; TestAdmitAcceptanceGuardsCodeRoutes | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current declaration duty: stubAcceptanceSource; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ad27b0983b18"></a>
<a id="internalfactorycontrolchecks_native_testgo-1"></a>

## [internal/factory/control/checks_native_test.go](../../../../../internal/factory/control/checks_native_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–163; file scaffold; TestNativeCheckPass; TestNativeCheckPending; TestNativeCheckMissing; TestNativeCheckFailed; TestNativeCheckCancelled; TestNativeCheckSkipped; TestNativeCheckUnknownState; TestNativeCheckLatestWins; TestNativeCheckPolicyEmpty | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2e44c5f3e47f"></a>

## [internal/factory/control/checks_pass.go](../../../../../internal/factory/control/checks_pass.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 47–57; file scaffold; CheckReport; checkError | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: CheckReport; Current declaration duty: checkError — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–20, 58–68; CheckObserver; checkWait | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: CheckObserver; Current declaration duty: checkWait — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 21–30, 69–146; CheckLink; AssessPublicationChecks; assessPublicationChecksOne | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: CheckLink; Current declaration duty: AssessPublicationChecks; Current declaration duty: assessPublicationChecksOne — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–46; CheckWait; CheckError | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: CheckWait; Current declaration duty: CheckError — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-47a07560d0e9"></a>

## [internal/factory/control/checks_pass_test.go](../../../../../internal/factory/control/checks_pass_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38; file scaffold; fakeCheckObserver; ObserveChecks; happyCheckObserver | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 39–82, 97–126; checkSeed; TestAssessPublicationChecksRecordsPass; TestAssessPublicationChecksWaitsAndErrors | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: checkSeed; Current declaration duty: TestAssessPublicationChecksRecordsPass; Current declaration duty: TestAssessPublicationChecksWaitsAndErrors — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 83–96; TestAssessPublicationChecksRecordsFailure | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: TestAssessPublicationChecksRecordsFailure — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2d089605b314"></a>

## [internal/factory/control/coordinator.go](../../../../../internal/factory/control/coordinator.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37, 39–41, 44–49, 52–67, 69–70, 80–178; whole file; (declaration group); HostFactory; BrokerExecution; Coordinator; NewCoordinator; Coordinator.Start; Coordinator.Close; Coordinator.Status; redactRun | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 38, 43, 73, 79; HostFactory.FactoryLaunch; HostFactory.FactoryHarness; Coordinator.DispatchReads, Coordinator.queue | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Native assignment launch contract; declarations/fields: `HostFactory.FactoryLaunch`; Pinned factory role harness observation contract; declarations/fields: `HostFactory.FactoryHarness`; Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.DispatchReads`, `Coordinator.queue` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 42, 74; HostFactory.FactoryExport; Coordinator.Publication | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Native candidate export contract; declarations/fields: `HostFactory.FactoryExport`; Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Publication` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 50; BrokerExecution.GetExecution | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Broker execution fence metadata read contract; coordinator never acquires credentials; declarations/fields: `BrokerExecution.GetExecution` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 51; BrokerExecution.CloseExecution | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Broker terminal execution close contract; declarations/fields: `BrokerExecution.CloseExecution` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 68; Coordinator.Store | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Store` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 71; Coordinator.AcceptanceReads | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.AcceptanceReads` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 72; Coordinator.Readiness | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Native revision/snapshot observer used to bracket readiness evidence; the obsolete traversal field was removed at `07ab24bd`. |
| 75; Coordinator.Reviews | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Reviews` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 76; Coordinator.Checks | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Checks` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 77; Coordinator.Merges | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Merges` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-78a6e82dd54b"></a>

## [internal/factory/control/coordinator_test.go](../../../../../internal/factory/control/coordinator_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–108, 127–400; file scaffold; stubHost; FactoryLaunch; FactoryStop; FactoryInspect; FactoryTakeover; FactoryExport; FactoryHarness; stubBroker; GetExecution; CloseExecution; coordinatorFixture; recordRun; stopCommand; TestStopSettlesAndReplays; TestStopReplaysDurableOutcome; TestStopUnknownRunRecordsNothing; TestStopFencesUncertainRetirement; TestReconcileSettlesCompletedRun; TestReconcileFencesBrokerFailure; TestStartTakesExclusiveOwnership; restartCoordinatorAfterCrash; settledStopBroker; TestRestartedStopReportsEstablishedOutcome; TestRestartedStopPreservesUncertainty; TestRestartedReconcileReportsSettledRuns | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 26 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 109–126; TestStatusRedactsCredentialPaths | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: TestStatusRedactsCredentialPaths — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ab11884907b9"></a>

## [internal/factory/control/correction.go](../../../../../internal/factory/control/correction.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 40–50; file scaffold; CorrectionReport; correctionError | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: CorrectionReport; Current declaration duty: correctionError — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 14–22, 51–319, 338–352; CorrectionLink; correctionWait; PublishCorrection; reconcileRecordedCorrection; linkCorrection | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: CorrectionLink; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 23–39, 320–337, 353–398; CorrectionWait; CorrectionError; adoptCorrectionOutcome; cancelRecordedCorrection; fenceCorrection; storeCorrection; correctionCallError | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: CorrectionWait; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dbb20c18e5c8"></a>

## [internal/factory/control/correction_test.go](../../../../../internal/factory/control/correction_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; file scaffold | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–35, 40–131; correctionRun; TestPublishCorrectionAdvancesHead; TestPublishCorrectionRefusesWithoutChange; dbMustPublication | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: correctionRun; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 36–39; correctionOutput | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: correctionOutput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b188ad73a4ab"></a>

## [internal/factory/control/cycles.go](../../../../../internal/factory/control/cycles.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 21–69, 93–103; file scaffold; findPrereqCycle; cyclePath | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: findPrereqCycle; Current declaration duty: cyclePath — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–20, 70–92; MaxCycleNodes; headDecision; prereqEndpoints | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: MaxCycleNodes; Current declaration duty: headDecision; Current declaration duty: prereqEndpoints — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6e200bbe2221"></a>
<a id="internalfactorycontroldispatchgo-1"></a>

## [internal/factory/control/dispatch.go](../../../../../internal/factory/control/dispatch.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–286; file scaffold; DispatchHost; DispatchBroker; DispatchComment; DispatchIssue; DispatchInputs; DispatchReads; DispatchWait; DispatchLaunch; DispatchError; DispatchReport; newDispatchReport; DispatchDeps; dispatchReady; RecoverDispatch; DispatchPass; dispatchDeps; Dispatch; dispatchAfterIntake | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current dispatch contract and bounded queue pass. `Coordinator.Dispatch` invokes the durable readiness drain after its ordinary pass; see the F06 current drain owner in `readiness_work.go`. The old `assessDispatchDependants` helper was retired at `07ab24bd`. |

<a id="coverage-cb30171eac23"></a>

## [internal/factory/control/dispatch_accounting_test.go](../../../../../internal/factory/control/dispatch_accounting_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–332; file scaffold; TestDispatchAccountingNeverRefreshes; TestDispatchInterruptionReleasesOnlyUnused; TestDispatchApprovedReceiptReleasesWhenBrokerUnknown; TestDispatchApprovedReceiptFencesWhenBrokerHolds; TestDispatchFencedLaunchStaysHeld; TestAccountSettledRunResults | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-81f3f2c15426"></a>
<a id="internalfactorycontroldispatch_attemptgo-1"></a>

<a id="coverage-d2a1c401e5b6"></a>

## [internal/factory/control/dispatch_advance_stale_test.go](../../../../../internal/factory/control/dispatch_advance_stale_test.go)

R15 stale-source advancement wait (added `bb08926a`). Stale source cannot launch implementation.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; file scaffold; TestStaleSourceCannotLaunchImplementation | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

## [internal/factory/control/dispatch_attempt.go](../../../../../internal/factory/control/dispatch_attempt.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–106, 119–193; file scaffold; attemptPlan; dispatchOne; planWait; waitFor; admissionWait; planAttempt | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 107–118; refreshOccupancy | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: refreshOccupancy — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-factory-control-dispatch-context"></a>

## [internal/factory/control/dispatch_context.go](../../../../../internal/factory/control/dispatch_context.go)

Current source inspected; selects accepted repository references and reads context from the bound native preparation before prompt construction.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–81; repositoryPathReference; readRepositoryContext; acceptedRepositoryPathReferences | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Bounds accepted path references, rejects overflow before native I/O, applies the attempt deadline and validates exact prepared-source/base/candidate response bindings. |

<a id="coverage-8f05676f603c"></a>

## [internal/factory/control/dispatch_concurrency_test.go](../../../../../internal/factory/control/dispatch_concurrency_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123; file scaffold; rendezvousReads; ReadDispatchInputs; mutexDispatchHost; FactoryLaunch; FactoryInspect; FactoryHarness; TestConcurrentDispatchPassesPreserveCapacityAndBudget | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="r02-current-path-internal-factory-control-dispatch-expiry-test-go"></a>

## [internal/factory/control/dispatch_expiry_test.go](../../../../../internal/factory/control/dispatch_expiry_test.go)

Current source inspected at `4b829d67`; these current responsibilities do not renew historical body-audit evidence.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| File scaffold; TestDispatchClosesExpiredReportedAttemptBeforeEarlyReturn | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Production Dispatch closes an expired reported root before its early return, preserving the result and recorded usage without another host launch. Fixture setup expires the recorded root in disposable PostgreSQL; [F10 source/development evidence](../../reviews/F10.md) remains distinct from native/installed qualification. |
| TestExplicitRetryCreatesFreshRootFromSettledReviewerChildAfterExpiry | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Uses production closure, explicit retry and dispatch to create one fresh root after a settled reviewer child; verifies command replay and preserved prior usage. The `fa395bff` [F07 source/development cut](../../reviews/F07.md) retains its scoped PostgreSQL evidence; host effects are substituted and full autonomous/native/provider qualification remains open. |

<a id="coverage-1cd1aa56ae9c"></a>

## [internal/factory/control/dispatch_fixture_test.go](../../../../../internal/factory/control/dispatch_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–267; file scaffold; dispatchDigest; fakeDispatchHost; FactoryLaunch; FactoryInspect; FactoryHarness; FactoryStop; FactoryTakeover; FactoryExport; fakeDispatchBroker; GetExecution; CloseExecution; fakeDispatchReads; ReadDispatchInputs; dispatchFixture; dispatchTestDB; dispatchSeed; accept; queue; queueAt; deps; waitReason; fakeAcceptanceSource; ReadAcceptanceEvidence | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 24 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5e0d1520bb7a"></a>

## [internal/factory/control/dispatch_inputs.go](../../../../../internal/factory/control/dispatch_inputs.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; file scaffold; readAttemptInputs; promptSections | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: readAttemptInputs; Current declaration duty: promptSections — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-158f3efbef96"></a>

## [internal/factory/control/dispatch_launch.go](../../../../../internal/factory/control/dispatch_launch.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–215; file scaffold; executeFreshAttempt; retryAttempt; launchTail; ensureHeldLaunch; settleUnusedLaunch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 593–602; R31 preparation-time bundle selection call site | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current declaration duty: best-effort PrepareContextBundleSelection beside capability staging — Current complete declaration/method or file span inspected at HEAD (`f6835c02`); historical pins retain their scope. |

<a id="coverage-9deac349c45b"></a>

## [internal/factory/control/dispatch_occupancy.go](../../../../../internal/factory/control/dispatch_occupancy.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; file scaffold; passOccupancy; snapshotOccupancy; hold; heldTotal; heldRepo; heldConn | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7685c91de81c"></a>

## [internal/factory/control/dispatch_recovery.go](../../../../../internal/factory/control/dispatch_recovery.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–314; file scaffold; recoverAssigned; recoverOne; dispatchHistoryRuns; activeHistoryRun; confirmUnused; hostRunNotFound; isHostNotFound; ensureHeld; recoverSettled; recoverMissingRun; recoverUnused; retryOrFinish; finishTerminal; recoverOutput | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e0c2db6ee404"></a>

## [internal/factory/control/dispatch_recovery_test.go](../../../../../internal/factory/control/dispatch_recovery_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–506, 622–743; file scaffold; TestRecoveryConsumesSettledRunWithoutHook; TestCompletionTriggersDependantReassessment; TestHostNotFoundMatcherPinsHostSentinel; TestIntakeTriggersAutomaticDispatch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 507–621; TestReportedAssignmentReadinessRootDrainsOnDispatch | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Actual reported-assignment completion emits durable readiness work and the normal Dispatch pass drains it; selected PostgreSQL fixture proof only, with aggregate trigger-graph limits still open. |

<a id="coverage-90596fafa0a5"></a>

## [internal/factory/control/dispatch_registration.go](../../../../../internal/factory/control/dispatch_registration.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; file scaffold; RegisterDispatch; ReopenDispatch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: RegisterDispatch; Current declaration duty: ReopenDispatch — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f11a77d022a8"></a>

## [internal/factory/control/dispatch_result.go](../../../../../internal/factory/control/dispatch_result.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–112; file scaffold; finishFromRun; recordConfirmedUsage; deriveAttemptResult; boundSummary; AccountSettledRun | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-00541ac014dc"></a>

<a id="coverage-d2b2d512f6c7"></a>

## [internal/factory/control/dispatch_terminal_phase_test.go](../../../../../internal/factory/control/dispatch_terminal_phase_test.go)

R10 phased terminal finishes (added `6d6d937a`). Terminal finishes bind their typed phased result; pre-phase assignments keep the legacy harness report contract.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–169; file scaffold; TestFinishTerminalBindsPhasedResult; TestFinishTerminalKeepsPrePhaseLegacyResult | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 3 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

## [internal/factory/control/dispatch_selection.go](../../../../../internal/factory/control/dispatch_selection.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–142; file scaffold; selectSponsorship; checkLimits; selectPreparation; checkHarness | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dc71af56a96e"></a>
<a id="internalfactorycontroldispatch_testgo-1"></a>

## [internal/factory/control/dispatch_test.go](../../../../../internal/factory/control/dispatch_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–108; file scaffold; TestDispatchPassLaunchesOldestWithinShortLimit; TestDispatchPassWithoutDepsReports | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestDispatchPassLaunchesOldestWithinShortLimit; Current declaration duty: TestDispatchPassWithoutDepsReports — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-32e414a25920"></a>

## [internal/factory/control/dispatch_wait_test.go](../../../../../internal/factory/control/dispatch_wait_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–298; file scaffold; TestDispatchWaitReasons; TestDispatchNeverFallsBackToAnotherConnection; TestDispatchWithdrawnGateRecordsNothing | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 420–480; TestDispatchRejectsAcceptanceWithdrawalAfterPlanning | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Actual Store withdrawal commits at the post-planning host boundary; the current packet admission guard then prevents assignment, run/view persistence and host launch. This does not cover withdrawal after packet commit. |

<a id="coverage-7d56dba1ba23"></a>

## [internal/factory/control/grant_authority.go](../../../../../internal/factory/control/grant_authority.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–67; file scaffold; EffectiveAuthority | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Current scaffold duty: file scaffold; Current declaration duty: EffectiveAuthority — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 68–94; preparationReadiness | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current declaration duty: preparationReadiness — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-eeebcb0b9b39"></a>
<a id="internalfactorycontrolgrantsgo-1"></a>

## [internal/factory/control/grants.go](../../../../../internal/factory/control/grants.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16, 76–99; file scaffold; ApplySponsorship | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Current scaffold duty: file scaffold; Current declaration duty: ApplySponsorship — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 17–21; ErrIneffectiveAuthority | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: ErrIneffectiveAuthority — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 22–37, 112–187; GrantReceipt; grantTarget; applyGrant; replayGrant | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current declaration duty: GrantReceipt; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 38–53; ApplyPolicy | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Current declaration duty: ApplyPolicy — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 54–64; ApplyCapacity | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: ApplyCapacity — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 65–75; ApplyOperatorGrant | [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) | retained | Current declaration duty: ApplyOperatorGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 100–111; ApplyEnvironmentGrant | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: ApplyEnvironmentGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-86d92eb75373"></a>

## [internal/factory/control/grants_test.go](../../../../../internal/factory/control/grants_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–119; whole file; grantPolicy; grantProfile; grantProject; grantFullAuthority; TestApplyPolicyReplaysAndClosesDispatch | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 120–146; TestSponsorshipWithdrawalKeepsDispatchWithActiveSibling | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Sponsorship withdrawal assertions; declarations/fields: `TestSponsorshipWithdrawalKeepsDispatchWithActiveSibling` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 147–163; TestReopenRequiresEffectiveGrants | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Effective admission before dispatch reopens assertions; declarations/fields: `TestReopenRequiresEffectiveGrants` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 164–177; TestRequirementAndApprovalDecisions | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Preparation decision chain assertions partitioned below; declarations/fields: `TestRequirementAndApprovalDecisions` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 178–191; TestRequirementAndApprovalDecisions | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Approval must reference current exact requirement inside mixed decision test; declarations/fields: `TestRequirementAndApprovalDecisions` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-bd575823b151"></a>

## [internal/factory/control/intent_closed_packets_test.go](../../../../../internal/factory/control/intent_closed_packets_test.go)

Delta duty inspection at `93fb3513`. Closed-history packet tests (I07): fixture-refused closed edits, tampered-history refusal, and r1/r2/r3 wire-golden packet checks against testdata/closed-history goldens. Native receipts stay shape references, never proof.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–355; file scaffold; updateWire; closedKeyA; closedKeyB; closedKeyC; closedKeyD; closedHistoryRevisions; closeNativeIssues; TestGeneratedGraphFixtureRefusesClosedEditAtomically; TestIntentGraphDeltaRefusesTamperedClosedHistory; wireNativeIssue; wireCase; wireManifest; dumpWireNative; checkWireGolden; wirePacketKeys; TestIntentGraphClosedHistoryWirePackets | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 15 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-ce41bcacf751"></a>

## [internal/factory/control/intent_decomposition.go](../../../../../internal/factory/control/intent_decomposition.go)

Delta duty inspection at `93fb3513`. Scoped generation preparation: accepted-scope task subset, durable packet preparation with delivered-stable binding, and receipt-issue mapping for adoption. Changed keys recreate once delivered; delivered records never enter packets.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–286; file scaffold; scopedIntentTasks; prepareIntentGraph; deliveredStable; intentReceiptIssues | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-e3b94a1a20dd"></a>

## [internal/factory/control/intent_decomposition_judgment.go](../../../../../internal/factory/control/intent_decomposition_judgment.go)

Delta duty inspection at `93fb3513`. Decomposition advice (S07): triplet-question advice with cached agreement, and RecomposeDecomposition validation of scoped split/combine alternatives (cycles, outcome coverage, bounds). Selected alternatives persist before native submission.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–145; file scaffold; ErrIntentDecompositionReview; decompositionQuestions; decompositionChoices; decompositionEvidence; judgeDecompositionTriplet; AdviseIntentDecomposition; decompositionOptionsRubric; RecomposeDecomposition | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 8 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 33–71, 77, 79, 161; effectiveDecompositionScope | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `46848018` (S07/2f50): effective-scope projection restricts triplet/options evidence to decided task-bound choices, accepted-or-corrected global assumptions, declined rejection evidence and all tasks; pending excluded, original revision kept for CAS/authority/native guards, veto/floor/filter/budgets unchanged. Prior rows retain their pinned scope. |

<a id="coverage-a9c657dda442"></a>

## [internal/factory/control/intent_decomposition_judgment_test.go](../../../../../internal/factory/control/intent_decomposition_judgment_test.go)

Delta duty inspection at `93fb3513`. Recompose scope-selection, ambiguity-challenge and triplet-cache/disagreement tests.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–141; file scaffold; decompositionChooser; Select; recomposeChooser; Select; multiOutcomeAccepted; TestRecomposeDecompositionKeepsConfidentGraphs; TestRecomposeDecompositionSelectsScopedAlternative; TestRecomposeDecompositionChallengesProductAmbiguity; TestIntentDecompositionFreshTripletCachesAndDisagreementRefuses | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 123–176; TestEffectiveDecompositionScopeProjectsEvidence | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `46848018` (S07/2f50): pure projection regression — Current source block and named test consumer inspected in internal/factory/control/intent_decomposition_judgment_test.go. |
| 177–236; TestRecomposeDecompositionJudgesEffectiveScope | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Delta at `46848018` (S07/2f50): recompose judges effective scope — Current source block and named test consumer inspected in internal/factory/control/intent_decomposition_judgment_test.go. |

<a id="coverage-31eda1f08272"></a>

<a id="coverage-d2c3e623a7d8"></a>

## [internal/factory/control/intent_generation_binding_test.go](../../../../../internal/factory/control/intent_generation_binding_test.go)

Accepted handback-034 recorded-plan binding repair (added `3e4de021`). Generation requires the original plan binding, rejects tasks outside the recorded candidate and rechecks the plan after native adoption reads.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–125; file scaffold; TestGenerationRequiresOriginalPlanBinding; TestGenerationRejectsTasksOutsideRecordedCandidate; TestGenerationRechecksPlanAfterNativeAdoptionReads | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d2d4f734b8e9"></a>

## [internal/factory/control/intent_generation_mechanics_test.go](../../../../../internal/factory/control/intent_generation_mechanics_test.go)

Graph-mechanics test helper isolating graph mechanics from planning (native graph fixture isolates them from Forgejo); production entry points always use the real recorded-plan validator.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; file scaffold; generateGraphMechanics | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d2e5a845c9f0"></a>

## [internal/factory/control/intent_generation_recovery.go](../../../../../internal/factory/control/intent_generation_recovery.go)

Intent graph recovery pass entry point.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; RecoverIntentGraphs | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

## [internal/factory/control/intent_generation.go](../../../../../internal/factory/control/intent_generation.go)

Delta duty inspection at `93fb3513`. Accepted-intent generation (I05/I07): request-scoped native issue-graph submission binding submitted tasks to one receipt, adoption of children from attributable receipts with exact native evidence, current-accepted gating, and generation cancellation. Consumed inline by the L07 ConsumeGeneratedIssues hook after adoption.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–547; file scaffold; IssueGraphExecutor; IntentGraphPayload; intentTaskBody; generatedDecisionID; GenerateAcceptedIntent; currentAcceptedIntent; adoptIntentGraph; generatedEdges; generatedTitleOccurrence; intentTaskTitle; cancelIntentGeneration | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 11 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 268–272; ConsumeGeneratedIssues hook in GenerateAcceptedIntent | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Delta at `93fb3513`: nil-guarded new-issue effect after adoption (55ec); never gates default delivery. |

<a id="coverage-709de2c68600"></a>

## [internal/factory/control/intent_generation_test.go](../../../../../internal/factory/control/intent_generation_test.go)

Delta duty inspection at `93fb3513`. Payload binding, generate/adopt, recovery, interrupted-response, split/consolidation, stale-cancel, long-proposal, correction, decline-scope, decomposition-submit, delivered-delta and effective-scope tests.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–797; file scaffold; generatedGraphFixture; CheckIntentOwner; SubmitGraph; LookupGraph; CancelGraph; ReadGeneratedIssue; ReadAcceptanceEvidence; acceptedGenerationFixture; TestIntentGraphPayloadBindsProductDirection; TestIntentGraphGeneratesAndAdoptsOnceWithNativeProvenance; TestIntentGraphServingOwnerRecoversDurableAcceptedDirection; TestIntentGraphInterruptedNativeResponseReusesStoredOperation; TestIntentGraphReviewedSplitAndConsolidationPreserveNativeHistory; generatedGraphChooser; Select; TestIntentGraphStaleDurablePacketCancelsBeforeNativeEffects; TestIntentGraphLongProposalRetainsAcceptedSemanticsWithinPacket; TestIntentGraphBodyIncludesReviewedChoicesAndCorrections; TestIntentGraphDeclinedAssumptionDoesNotConstrainNativeScope; recomposeGenerationChooser; Select; TestIntentGraphSubmitsSelectedDecomposition; TestIntentGraphDeltaRetainsDeliveredRecords; TestIntentGraphEffectiveScopeExcludesPendingAndDepartment | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 25 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-d2f6b956d0a1"></a>

## [internal/factory/control/intent_plan_choice.go](../../../../../internal/factory/control/intent_plan_choice.go)

R13 recorded-plan choice binding (added `0a96b22e`). Resolves the accepted plan choice from the recorded plan: single-candidate needs no comparative call, funded multi-choice, pending/missing/predecessor/attention/failed/stale-source refusals with source rechecks.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–373; file scaffold; ErrPlanChoiceStale; ErrPlanChoiceAttention; ErrPlanChoiceStore; GenerateAcceptedIntentFromRecordedPlan; ResolveAcceptedPlanChoice; admitAcceptedChoiceHead; bindRecordedPlanInput; bindAcceptedScope; bindRecordedScopeCoverage; settlePlanChoiceStatus; recheckPlanChoiceSource; recheckPlanChoiceAfterComparison | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 12 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d307ca67e1b2"></a>

## [internal/factory/control/intent_plan_choice_test.go](../../../../../internal/factory/control/intent_plan_choice_test.go)

R13 recorded-plan choice tests (added `0a96b22e`). Single/multi/pending/missing/predecessor/attention/failed/stale-source choice cases with persisted settled planner assignments and real PostgreSQL.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–734; file scaffold; TestResolveAcceptedPlanChoiceSingleNeedsNoComparativeCall; TestResolveAcceptedPlanChoiceMultiUsesOneFundedChoice; TestResolveAcceptedPlanChoicePendingWaits; TestResolveAcceptedPlanChoiceMissingRefuses; TestResolveAcceptedPlanChoiceWithoutPredecessorRefuses; TestResolveAcceptedPlanChoiceAttentionRefuses; TestResolveAcceptedPlanChoiceFailedRefuses; TestResolveAcceptedPlanChoiceStaleSourceRefuses | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-e25bcf361055"></a>

## [internal/factory/control/inventory_test.go](../../../../../internal/factory/control/inventory_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–76; file scaffold; settleFixture; recordSettledHistory; TestReconcileFindsUnresolvedBehindSettledHistory; TestProjectRunsFindsUnresolvedBehindSettledHistory | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7e6f72a09417"></a>

## [internal/factory/control/judgment_advice.go](../../../../../internal/factory/control/judgment_advice.go)

Delta duty inspection at `93fb3513`. Seven typed JudgmentPurposes with durable completed-result reuse (S02): reservation-before-call, semantic-fingerprint cache, refusal/unavailable records; failed/interrupted stays charged without repeat.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; file scaffold; PurposeAdvice; AdvisePurpose | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 2 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-6e9905b27dba"></a>

## [internal/factory/control/judgment_advice_test.go](../../../../../internal/factory/control/judgment_advice_test.go)

Delta duty inspection at `93fb3513`. Result-cache, charged-failure and reserve-refusal tests with injected choosers.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–92; file scaffold; purposeChooser; Select; purposeAdvice; TestAdvisePurposeCachesCompletedResult; TestAdvisePurposeFailureStaysChargedWithoutRepeat; TestAdvisePurposeReserveRefusalStaysUnavailable | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 6 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-f7a105fa9718"></a>
<a id="internalfactorycontrollifecyclego-1"></a>

## [internal/factory/control/lifecycle.go](../../../../../internal/factory/control/lifecycle.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147, 193–238; file scaffold; ErrPendingRuns; Error; PauseRepository; replayPause; withdrawAndStopRuns; projectRuns; ResumeRepository; replayResume | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 148–192; StopProject; ClearProjectStopAfterStart | [P05](../../slices/projects.md#p05-project-startstop) | retained | Start clears only the Project-stop cause after validated host verification and unsettled-run checks; independent dispatch causes remain closed. Source subcut `8d9485af`; broader P09 approval authority remains open. |

<a id="coverage-137e01e08905"></a>

## [internal/factory/control/lifecycle_callbacks_test.go](../../../../../internal/factory/control/lifecycle_callbacks_test.go)

Delta duty inspection at `93fb3513`. Owner-callback tests proving the three completion hooks fire: PR reviewer request, verified-merge note and generated-issue effects through enabled configs.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–138; file scaffold; callbackLifecycleExecutor; enableCallbackConfig; TestLifecycleCallbackPR; TestLifecycleCallbackMerge; TestLifecycleCallbackGeneration | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 5 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-c26d76bb5064"></a>

## [internal/factory/control/lifecycle_capabilities.go](../../../../../internal/factory/control/lifecycle_capabilities.go)

Delta duty inspection at `93fb3513`. Native capability registrar (L06): installation-bound markers per supported lifecycle kind from an authenticated handshake; failed/unattributable answers clear every marker so new admission fails closed. Every new-effect producer must pass RequireCurrentCapability; reconciliation by identity is exempt.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; file scaffold; CapabilityHandshakeSource; ReconcileLifecycleCapabilities; RequireCurrentCapability | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 3 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-17610ce74657"></a>

## [internal/factory/control/lifecycle_capabilities_test.go](../../../../../internal/factory/control/lifecycle_capabilities_test.go)

Delta duty inspection at `93fb3513`. Registrar reconciliation and marker-replacement tests with scripted handshakes.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–150; file scaffold; scriptHandshake; Handshake; lifecycleTestConfig; TestReconcileLifecycleCapabilities; TestReconcileReplacements | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 5 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-5521507409ca"></a>

## [internal/factory/control/lifecycle_consume.go](../../../../../internal/factory/control/lifecycle_consume.go)

Delta duty inspection at `93fb3513`. Shared inline event consumer (L07): gates on enabled config plus proved capability, persists action identity before submit, reconciles uncertainty by operation identity, retries once after definitive no-effect, drains unsettled actions bounded, and journals unavailable-executor outcomes. Called inline from generation adoption, publication and merge completion; every error path leaves default state untouched.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–579; file scaffold; LifecycleExecutor; LifecycleEventInput; lifecycleNotAfter; lifecycleCallTimeout; withLifecycleTimeout; LifecycleNewIssueKeys; LifecycleActionID; LifecycleOperationID; ClassifyLifecycleOutcome; lifecycleAttempt; lifecycleRetryAllowed; claimLifecycleAction; ReconcileLifecycleAction; ErrLifecycleExecutorUnavailable; journalUnavailableExecutor; ConsumeLifecycleEvent; ConsumeGeneratedIssues; ConsumePublishedPR; ConsumeVerifiedMerge; lifecycleDrainLimit; ReconcileUnsettledLifecycle | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 21 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
| 275–338, 394–469, 476–547, 558–572; ReconcileLifecycleAction; ConsumeLifecycleEvent; ConsumeGeneratedIssues; ConsumePublishedPR; ConsumeVerifiedMerge; ReconcileUnsettledLifecycle | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Delta at `93fb3513`: executor field/construction predates the delta (C06ef seam; coordinator.go:88, server.go:91-92, configuration-gated). Production hooks and PublishPass/MergePass drain hosting integrated at 55ec; all nil-guarded and non-gating. Source wiring is not installed credential/configuration proof (Q08). Corrects OCR-1 per addendum. |
| 57, 136–184, 423–440, 549–572; lifecycleCallTimeout; lifecycleAttempt; ConsumeLifecycleEvent native calls; drain pass | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Delta at `93fb3513`: first attempt runs inline with up to five 60s-bounded native calls; default-unaffected reasoning covers state, not time (OCR-5 latency question for L/Q). |

<a id="coverage-0b8d4f4b5b55"></a>

## [internal/factory/control/lifecycle_consume_test.go](../../../../../internal/factory/control/lifecycle_consume_test.go)

Delta duty inspection at `93fb3513`. Consumer, owner-helper, restart-reconcile and unsettled-drain tests with a fake executor.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–509; file scaffold; consumeFakeExecutor; EnrolledActor; ObserveLifecycleRevision; ResolveNoteTarget; SubmitLifecycle; LookupLifecycle; consumeOutcome; enableConsumeConfig; consumeIssueInput; TestLifecycleNewIssueKeys; TestConsumeLifecycleEvent; TestConsumeOwnerHelpers; TestReconcileAfterRestart; TestReconcileUnsettledDrain | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 14 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-54d733b9515a"></a>

## [internal/factory/control/lifecycle_envelope_join_test.go](../../../../../internal/factory/control/lifecycle_envelope_join_test.go)

Delta duty inspection at `93fb3513`. Envelope/payload join acceptance on the real hooked passes: submit visibility and visible-result checks per envelope case.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–133; file scaffold; joinEnvelopeCases; checkJoinSubmit; checkJoinVisible; TestLifecycleEnvelopeJoin | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 4 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
<a id="coverage-478b12aafffc"></a>

## [internal/factory/control/lifecycle_retry.go](../../../../../internal/factory/control/lifecycle_retry.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77, 107–116; file scaffold; RetryRun; replayRetry | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: RetryRun; Current declaration duty: replayRetry — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 78–106; onlyDispatchClosed; retryQueueReason | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: onlyDispatchClosed; Current declaration duty: retryQueueReason — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e71b842c6faf"></a>

## [internal/factory/control/lifecycle_retry_test.go](../../../../../internal/factory/control/lifecycle_retry_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; file scaffold | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–41; TestRetryStaysQueuedAfterReconciledRun | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: TestRetryStaysQueuedAfterReconciledRun — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2618d5eea339"></a>

## [internal/factory/control/lifecycle_takeover.go](../../../../../internal/factory/control/lifecycle_takeover.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 24–134; file scaffold; TakeoverRun; replayTakeover; VerifyProjectStart | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–23, 135–154; ErrTakeoverFailed; checkStartRun | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: ErrTakeoverFailed; Current declaration duty: checkStartRun — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-59e304d80751"></a>

## [internal/factory/control/lifecycle_takeover_test.go](../../../../../internal/factory/control/lifecycle_takeover_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–122; file scaffold; TestTakeoverCopiesOnlyReconciledRuns; TestStopProjectScopesWithdrawalAndRuns; TestVerifyProjectStartClassifiesRuns | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5b8742504bab"></a>

## [internal/factory/control/lifecycle_test.go](../../../../../internal/factory/control/lifecycle_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–178; file scaffold; lifecycleProject; lifecycleProjectFixture; settleStubs; readyPreparations; TestPauseWithdrawsBeforeStopping; TestPauseKeepsUncertainRunsFenced; TestResumeReopensOnlyAfterSettledRuns; TestResumeRefusesWithdrawnGrants | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 180–206; TestResumePreservesProjectStopUntilVerifiedStart; 208–224; TestStartKeepsProjectStopWhileRunIsUnsettled | [P05](../../slices/projects.md#p05-project-startstop) | retained | These regressions retain `project_stop` across ordinary Resume and unsettled runs, then exercise the verified-start boundary. `8d9485af` source/owned-PG scope; this does not establish native Project-admin approval authority or native-provider qualification. |

<a id="coverage-db65166769d0"></a>

## [internal/factory/control/operator.go](../../../../../internal/factory/control/operator.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–145; file scaffold; OperatorPath; principalKey; WithOperatorPrincipal; operatorPrincipal; OperatorRequest; OperatorHandler; serveOperator; serveStatus; serveStop; serveReconcile; writeCommandError; writeOperator | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e90f6ccf90f7"></a>

## [internal/factory/control/operator_test.go](../../../../../internal/factory/control/operator_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; file scaffold; operatorRequest; TestOperatorRequiresPrincipalAndEnvelope; TestOperatorStatusAndStopRoundTrip; TestOperatorReplaysAndConflicts | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f357328db6e5"></a>

## [internal/factory/control/planning.go](../../../../../internal/factory/control/planning.go)

Delta duty inspection at `93fb3513`. Planning pass (S08): provisional missing outputs, unchanged-input reuse, freshness/uncertainty records and actual-prerequisite-output refresh before coding; charged failure records on planning-call failure.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–258; file scaffold; ProductJudgment; PlanningJournal; PlanningJudgments; PlanningPass; planningInput; planningRubric; planningChoices; planIssue; planningCallFailed; planningSteps; planningCurrent; planningRecord | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 12 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-d318db78f2c3"></a>

## [internal/factory/control/planning_test.go](../../../../../internal/factory/control/planning_test.go)

Planning record tests: records once and feeds fresh launches; provisional records cannot launch.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113; file scaffold; TestPlanningRecordsOnceAndFeedsFreshLaunch; TestPlanningProvisionalRecordCannotLaunch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 3 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-66978fcec35a"></a>

## [internal/factory/control/postgres_fixture_external_test.go](../../../../../internal/factory/control/postgres_fixture_external_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; file scaffold; postgresFixture | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: postgresFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f0db5acc95db"></a>

## [internal/factory/control/postgres_fixture_test.go](../../../../../internal/factory/control/postgres_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; file scaffold; postgresFixture | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: postgresFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-25379eba5810"></a>

## [internal/factory/control/preparation_decisions.go](../../../../../internal/factory/control/preparation_decisions.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–160; file scaffold; DecisionReceipt; AdmitRequirement; AdmitApproval; admitDecision; replayDecision; holdActive | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-46f296c4814d"></a>

## [internal/factory/control/prerequisites.go](../../../../../internal/factory/control/prerequisites.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; file scaffold | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 16–190, 207–269; prereqAssessment; assessPrerequisites; assessEndpointError; endpointDecision; endpointEvidence; assessCodePrereq; assessResultPrereq; endpointAcceptanceValid | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: prereqAssessment; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 191–206; codePrereqCompleted | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: codePrereqCompleted — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b6c91fe80256"></a>

## [internal/factory/control/priority.go](../../../../../internal/factory/control/priority.go)

Delta duty inspection at `93fb3513`. Staged-tournament priority selection (S09): complete eligible inventory, finite upfront calls, uncertain/oldest handling, recorded fairness and fresh exact admission guards inside the priority journal.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–154; file scaffold; PriorityJournal; PriorityChooser; PriorityJudgments; ErrPriorityUnavailable; selectPriority; priorityCallCount | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-d329ec89a3d4"></a>

## [internal/factory/control/priority_dispatch.go](../../../../../internal/factory/control/priority_dispatch.go)

Priority inventory dispatch pass: revision observation, candidate refresh and eligibility, scan-state collection, snapshot confirmation and per-subject observation.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–397; file scaffold; PrioritySnapshotChanging; dispatchPriorityPass; priorityObserveRevision; priorityRefreshCandidate; priorityMemberEligible; dispatchPriorityInventoryPass; priorityScanState; observePriorityInventory; observePrioritySubject; prioritySelection; collectPriorityCandidates; confirmPrioritySnapshot | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 12 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d43afd90b4e5"></a>

## [internal/factory/control/priority_test.go](../../../../../internal/factory/control/priority_test.go)

Priority tournament tests: complete tournament with finite calls, uncertainty aging with single candidate, dispatch selection with stale-set refusal.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147; file scaffold; TestPriorityCompleteTournamentAndFiniteCalls; TestPriorityUncertaintyAgeAndSingleCandidate; TestPriorityDispatchSelectsAndRefusesStaleSet | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d44b0ea1c5f6"></a>

## [internal/factory/control/priority_typesafe.go](../../../../../internal/factory/control/priority_typesafe.go)

TypeSafe Jev priority client (server-only secret file, never retries; the finite journal reservation owns count and attribution). Serves both the PriorityChooser and the ProductJudgment interfaces over one bounded Choice call.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–141; file scaffold; TypeSafePriority; Choose; admitPriorityCandidates; readPriorityKeyFile; postPriorityRequest; decodePriorityResponse; Select | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d45c1fb2d6a7"></a>

## [internal/factory/control/priority_typesafe_test.go](../../../../../internal/factory/control/priority_typesafe_test.go)

TypeSafe client tests: bounded Choice with partial-answer rejection.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–58; file scaffold; TestTypeSafePriorityUsesBoundedChoiceAndRejectsPartialAnswer | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current scaffold duty: file scaffold; 2 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-6a1f26af267b"></a>

## [internal/factory/control/publication_authority.go](../../../../../internal/factory/control/publication_authority.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–56; file scaffold; publicationAuthority | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; Current declaration duty: publicationAuthority — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 57–70; publicationWork | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: publicationWork — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2567dd235cff"></a>

## [internal/factory/control/publication_effect_test.go](../../../../../internal/factory/control/publication_effect_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–145; file scaffold; TestPublishPassReconcilesLostSubmitReply; TestPublishPassReplaysUnobservedSubmitWithByteIdenticalIntent; TestPublishPassKeepsBranchCommittedPRFailedPartial; TestPublishPassDoesNotRetryTerminalBranchRefusal; TestPublishPassPendingPushIsBoundedPerPass; TestPublishPassFencesIndeterminateEffect | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5aa144fbeec2"></a>

## [internal/factory/control/publication_export.go](../../../../../internal/factory/control/publication_export.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; exportCandidate; decodeExportBundle; hostRunStale; isHostStale | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-05fa0a71c9b7"></a>

## [internal/factory/control/publication_fixture_test.go](../../../../../internal/factory/control/publication_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–156, 177–181, 244–260, 304–311; file scaffold; publicationTestDB; fakePublishHost; FactoryLaunch; FactoryStop; FactoryInspect; FactoryTakeover; FactoryHarness; FactoryExport; fakePublisher; ObservePublication; SubmitPublish; PushBranch; SubmitPRCreate; LookupOp; CancelOp; AdoptBranch; AdoptPRCreation; record; publicationTestOutcome; wire; pendingOutcome; committedOutcome; refusedOutcome; pass | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 25 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 157–176, 182–243, 261–303; publishFixture; publishSeed; finishReported; happyPublisher; publication | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: publishFixture; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-63afabb0ac8f"></a>

## [internal/factory/control/publication_reconcile.go](../../../../../internal/factory/control/publication_reconcile.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 232–235; file scaffold; terminalPublicationEffect | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; Current declaration duty: terminalPublicationEffect — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–231, 236–264; reconcilePublication; adoptPublicationReceipts | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: reconcilePublication; Current declaration duty: adoptPublicationReceipts — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-27f7f394740f"></a>

## [internal/factory/control/publication_recovery_test.go](../../../../../internal/factory/control/publication_recovery_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 41–115; file scaffold; TestPublishPassLostPRReplyRetainsLateCommittedLinkAndCompletion; TestPublishPassRecoversCommittedPRWithoutExport; TestPublishPassFailsUnrecoverableExport; TestPublishPassWaitsWithoutExecutor | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–40; TestPublishPassWithdrawalFailureStaysOpen | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: TestPublishPassWithdrawalFailureStaysOpen — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-33d1ca5d2bd1"></a>

## [internal/factory/control/publication_withdraw.go](../../../../../internal/factory/control/publication_withdraw.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–57, 124–157; file scaffold; cancelPublications; adoptObserved; operationOutcomeOf; storePublication | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 58–123, 158–171; publishAfterDispatch; withdrawPublication; finishPublication | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: publishAfterDispatch; Current declaration duty: withdrawPublication; Current declaration duty: finishPublication. The `3766b3c3` repair propagates the readiness caller deadline through publication handoff while retaining a 10-minute cap; its cancellation/custody regression exercises expiry at this boundary. |

<a id="coverage-e04e60fdaf35"></a>

## [internal/factory/control/q05_first_save_allowance_test.go](../../../../../internal/factory/control/q05_first_save_allowance_test.go)

Delta duty inspection at `93fb3513`. Q05 first-save allowance producer/explore boundary driver: funded-explore charging and error behavior at the control seam.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–70; file scaffold; TestFirstSaveAllowanceProducerExploreBoundary | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |
<a id="coverage-ddeb03ec3eda"></a>

## [internal/factory/control/readiness.go](../../../../../internal/factory/control/readiness.go)

Current intake/assessment responsibilities and selective durable-drain delta at `07ab24bd`; per-issue deadline correction is `7763c62e`.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–127; scaffold, ReadinessObservation, IntakeHint/Validate, assessOutcome, ObserveIssueEvent, readinessDeliverySourceID | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Authenticated intake, duplicate delivery handling, durable source creation and selected-delivery completion before acknowledgement. An incomplete critical delivery remains unrecorded for its existing retry; aggregate actual-trigger behavior remains B03.C-lifetime. |
| 128–303; assessOne, issueAssessor, obscured/obscuredVerdict, acceptanceBlocker, authorityBlockers, record, carrySatisfaction | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | One assessment's evidence and persistence. `7763c62e` keeps prerequisite/cycle reads and final recording inside its existing two-minute context, with the tighter status-read child preserved. Focused actual PG17 regression, blocker and retry checks pass; this does not bound the whole cascade. |
| 304–321; readinessVerdict | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Ordered blocker classification for authorization, blocked and queued outcomes. |

<a id="r02-current-path-internal-factory-control-readiness-work-go"></a>

## [internal/factory/control/readiness_work.go](../../../../../internal/factory/control/readiness_work.go)

Current durable production drain, source integration `25ab7a48`.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–328; pass budget, evidence/page/assessment/selection counters, local drain exclusion, source selection/checkpoint/complete/defer and bounded graph processing | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Production continuation over Store `ReadinessWork` nodes. The intake regression demonstrates authenticated delivery yield/resume through Dispatch; the 2026-10 selected deadline join also covers reported assignment → durable root → normal Dispatch and caller-expiry/custody, while existing merge-completion and publication/dispatch handoff subjects run in that proof. The retirement packet reused 26 passing Store/schema/control cases and passed the corrected intake/cycle/fairness selectors. These selected joins do not prove aggregate physical RPC bounds through bootstrap/actor admission or mutable-keyset restart; schema 39 removes the unused persisted `RootChanged` marker, while the invocation-local result remains. |

<a id="coverage-f7aaecb325be"></a>

## [internal/factory/control/readiness_fixture_test.go](../../../../../internal/factory/control/readiness_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16, 39–58, 87–104; file scaffold; totalCalls; fakeObserver; ObserveNativeRevision; readinessHint | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current fixtures keep only native revision observation; list-page fakes and issue-list adapters were removed at `07ab24bd`. |
| 17–38, 60–85, 91–121; fakeEvidenceSource; ReadAcceptanceEvidence; readinessView; readinessEvidence; readinessDecision; readinessCoordinator; mustAdmit; setupCodePrereq | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: fake evidence and accepted prerequisite setup helpers for readiness assessment tests. |

<a id="coverage-574e43afaa56"></a>

## [internal/factory/control/readiness_prerequisite_test.go](../../../../../internal/factory/control/readiness_prerequisite_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 35–51; file scaffold; setupResultPrereq | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current scaffold duty: file scaffold; Current declaration duty: setupResultPrereq — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–34, 52–125; TestClosureAloneCannotSatisfyCodeOutcome; TestResultPrereqClosureOccurrence; TestResultPrereqNeedsAcceptedResolution | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: TestClosureAloneCannotSatisfyCodeOutcome; Current declaration duty: TestResultPrereqClosureOccurrence; Current declaration duty: TestResultPrereqNeedsAcceptedResolution — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5f62fc9c60ac"></a>

## [internal/factory/control/readiness_sweep_test.go](../../../../../internal/factory/control/readiness_sweep_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–59; file scaffold; TestFindPrereqCycle; TestFindPrereqCycleExceedsBound | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current retained tests cover bounded cycle detection and its over-limit result. Sweep tests were retired at `07ab24bd`; the two cycle selectors passed in the corrected three-selector control batch. |
| 70–202; TestAcceptanceHeadMutationBehindSavedCursorRestartsDelivery | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Actual PG17 critical-intake regression: a head mutation behind the saved cursor is reread on redelivery; the blocked late control and assessed node are retained, then source deletion and acknowledgement complete together. Native/provider revision boundaries are fake in-memory fixtures; aggregate bounds and full trigger-graph behavior remain open. |

<a id="coverage-8d8ccde235b9"></a>
<a id="internalfactorycontrolreadiness_testgo-1"></a>

## [internal/factory/control/readiness_test.go](../../../../../internal/factory/control/readiness_test.go)

Current production-intake fixtures and deadline regression at `7763c62e`.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–321; scaffold, eleven intake/readiness tests, acceptanceEvidenceDeadline, deadlineRecordingAcceptanceSource and its ReadAcceptanceEvidence | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Production coordinator intake, creation/adoption, duplicate suppression, unchanged-root retry, pull/visibility refusal and readiness classification. The new `TestObserveIssueEventBoundsPrerequisiteEvidenceRead` reproduces unbounded endpoint evidence before the repair and passes afterward; two existing blocker/retry tests pass, and the affected regression passes after review's timing hardening. Actual PostgreSQL is used; evidence comes from the existing fake source. All owned clusters stopped before deletion; durable continuation and installed/native qualification remain open. |

<a id="coverage-5fe3a2cf91af"></a>

## [internal/factory/control/readiness_visibility_test.go](../../../../../internal/factory/control/readiness_visibility_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11; file scaffold | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–172; TestHiddenEndpointBlocksWithoutLeaking; TestHiddenIssueDeniesAuthorization; TestIncompleteEvidenceBlocks; TestLongerCycleBlocks; TestWithdrawnEndpointInvalidatesDependent; TestEditedEndpointInvalidatesDependent; TestMissingAuthorityStaysUnauthorized | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: TestHiddenEndpointBlocksWithoutLeaking; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d46d20c3e7b8"></a>

## [internal/factory/control/repair_admit.go](../../../../../internal/factory/control/repair_admit.go)

R36/R37 stall-episode repair admission (added `cada7aaa`). Admits one bound repair charge per classified stall episode; refuses without a stall or past the admitted generation.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; file scaffold; repairEpisodeChargeID; AdmitStallEpisodeRun | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 2 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d47e31d4f8c9"></a>

## [internal/factory/control/repair_admit_test.go](../../../../../internal/factory/control/repair_admit_test.go)

R36/R37 repair admission tests (added `cada7aaa`). One bound charge admitted; refusal without stall and refusal past the admitted generation.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–141; file scaffold; TestAdmitStallEpisodeRunAdmitsOneBoundCharge; TestAdmitStallEpisodeRunRefusesWithoutStall; TestAdmitStallEpisodeRunRefusesFutureGeneration | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 4 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-4188f8fdfa55"></a>

## [internal/factory/control/serving_intent_journey_test.go](../../../../../internal/factory/control/serving_intent_journey_test.go)

Delta duty inspection at `93fb3513`. Serving accept→plan→execute→review→merge→unblock journey test with scripted graph and chooser.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–114; file scaffold; journeyGraph; SubmitGraph; journeyChooser; Select; TestServingAcceptedIntentCreatesPlansExecutesReviewsMergesAndUnblocks | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here — Current complete declaration/method or file span inspected at `93fb3513`; historical pins retain their scope. |

<a id="coverage-d48f42e509d0"></a>

## [internal/factory/control/serving_planning_priority_test.go](../../../../../internal/factory/control/serving_planning_priority_test.go)

Serving planning-priority tests: complete eligible set consulted with fresh selection launched; provisional planning refreshed for actual merged output.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–109; file scaffold; TestServingCoordinatorConsultsCompleteEligibleSetAndLaunchesFreshSelection; TestServingCoordinatorRefreshesProvisionalPlanningForActualMergedOutput | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 3 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-19d2f4e17a6a"></a>

## [internal/factory/control/settle.go](../../../../../internal/factory/control/settle.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–103, 122–299, 313–369; whole file; settleRun | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Package/import scaffolding; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 104–106; whole file | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Invoke readiness progression from Reconcile; readiness authority remains F06 — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 107–109, 300–301; whole file | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Invoke dispatch progression; assignment authority remains F07; Settled-run accounting and dependant assessment — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 110–113, 302–306; whole file | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Invoke publication progression; Publication after successful accounting; Correction after settlement — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 114–117; whole file | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Invoke check progression — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 118–121; whole file | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Invoke merge progression — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 307–312; whole file | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Independent reviewer progression — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-b913f84dada5"></a>

## [internal/factory/control/settle_test.go](../../../../../internal/factory/control/settle_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–78; whole file; TestSettleRunRetriesTransientBrokerClose; TestSettleRunRefusesUnconfirmedBrokerClose | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Package/import scaffolding; Transient broker-close recovery on second attempt, Cancelled outcome and stored reconciliation; declarations/fields: `TestSettleRunRetriesTransientBrokerClose`; Three failed broker closes leave an uncertain receipt; declarations/fields: `TestSettleRunRefusesUnconfirmedBrokerClose` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-974af78e2fa5"></a>

## [internal/factory/control/st15_demo_broker_test.go](../../../../../internal/factory/control/st15_demo_broker_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; file scaffold; st15RepoRoot; st15BuildBroker; st15BuildHost; st15BuildFactoryRoles; st15TmpfsRoot; st15FakeCodex; st15FakeMuse; st15WaitSocket | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fa44836e6ebb"></a>

## [internal/factory/control/st15_demo_coding_test.go](../../../../../internal/factory/control/st15_demo_coding_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21, 31–64, 138–189, 226–233; file scaffold; runForIssue; pollRunForIssue; stallForBrowserDiag; tailLines; mustViewAttempt | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 22–30, 65–93; providerGate; runBrowser | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: providerGate; Current declaration duty: runBrowser — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 94–137; dispatchA | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: dispatchA — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 190–225; publishA | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: publishA — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2237e93ea142"></a>

## [internal/factory/control/st15_demo_completion_test.go](../../../../../internal/factory/control/st15_demo_completion_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18, 150–209; file scaffold; proveRetention | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: proveRetention — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 19–34; ciPass | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current declaration duty: ciPass — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 35–149; mergeAndDependants | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: mergeAndDependants — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-436adc9aad96"></a>

## [internal/factory/control/st15_demo_coordinator_test.go](../../../../../internal/factory/control/st15_demo_coordinator_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101; file scaffold; wireCoordinator; postIntake; issueOpenedHint; commentHint | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b574e43536c2"></a>

## [internal/factory/control/st15_demo_project_test.go](../../../../../internal/factory/control/st15_demo_project_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 85–235; file scaffold; prepareRoles; seedHumanWork | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: prepareRoles; Current declaration duty: seedHumanWork — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 20–84; setupProject | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: setupProject — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a55f8c764882"></a>

## [internal/factory/control/st15_demo_stack_test.go](../../../../../internal/factory/control/st15_demo_stack_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–312; file scaffold; setupHostStack | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; Current declaration duty: setupHostStack — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 313–325; openFactoryStore | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Current declaration duty: openFactoryStore — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d49a53f61ae1"></a>

## [internal/factory/control/stall_serve.go](../../../../../internal/factory/control/stall_serve.go)

R36 D15 detection serving (added `9eb5f96f`). Serves one bounded stall round admitting a bounded episode per classified stall.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–65; file scaffold; stallEpisodeActiveMinutes; ServeStallRound | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 2 named units assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-d4ab64a72bf2"></a>

## [internal/factory/control/stall_serve_test.go](../../../../../internal/factory/control/stall_serve_test.go)

R36 stall serving tests (added `9eb5f96f`). Bounded episode admitted per stall round.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–83; file scaffold; TestServeStallRoundAdmitsBoundedEpisode | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold; 1 named unit assigned here — Current complete declaration/method or file span inspected at HEAD; historical pins retain their scope. |

<a id="coverage-4de07ed759a2"></a>

## [internal/factory/control/traversal.go](../../../../../internal/factory/control/traversal.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–64; file scaffold; DispatchQueueCursor, NewDispatchQueueCursor, DispatchQueueCursor.start, DispatchQueueCursor.advance, DispatchQueueCursor.reset | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Bounded dispatch queue cursor mechanics and fair rotation remain. All process-local sweep/page/repository cursor state was removed at `07ab24bd`. |

<a id="coverage-3f65bf95377c"></a>

## [internal/factory/control/traversal_test.go](../../../../../internal/factory/control/traversal_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| TestDispatchPassReachesRunnableBehindWaitingPrefix | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | At `c9de3b26`, actual dispatch continuation reaches runnable work after both queued and accepted authority-blocked waiting prefixes. The corrected retirement selector passes; per-fixture source headers are removed only through the fixture SQL connection because queue rotation, not source admission, is under test. |

<a id="current-progress-owner"></a>

## [internal/factory/control/progress.go](../../../../../internal/factory/control/progress.go)

Selective current source at `c9de3b26` (2026-10-11); historical body selectors above retain their original pins.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| progressPass; startProgress; runProgress; progressInterval; progressPassKey | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Same Coordinator serialization, owner-lock lifetime and cancellation/join; no second service. |
| refreshReadiness | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Authoritative native revision refresh uses the existing assessment and bounded evidence budget. Accepted authority-blocked controls compare the current Store authority fingerprint; a changed fingerprint refreshes assessment despite an unchanged native revision, while an unchanged fingerprint consumes no native RPC. |
| advanceProgress | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Bounded unsettled-run inspection/settlement, ordinary Dispatch and owned readiness drain; immutable attempt allowance remains at the Store. |

<a id="current-progress-tests"></a>

## [internal/factory/control/progress_test.go](../../../../../internal/factory/control/progress_test.go)

Selective current source at `610a2df2` (2026-10-11); historical body selectors above retain their original pins.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| TestCoordinatorWakePreservesLiveCLIAndCloseJoinsInspection | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Live CLI remains supervised; exhausted inspection budget refuses work; Close cancels and joins. |
| TestCoordinatorWakeResumesReadinessWithoutRedelivery | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Durable readiness completes without another delivery. |
| TestCoordinatorWakesForCodingDelayedChecksAndReviewer; progressHost; progressInputs; progressRevision | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Normal producer reaches delayed native CI and a distinct exact-candidate reviewer without duplicate publication. |
| TestCoordinatorWakeRetriesStaleReviewPreparationAdmission; preparationAdmissionRaceInputs and ReadDispatchInputs | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | A normal Store assessment changes control after planning; authoritative stale admission has no native effect and preserves Published/root/time/cycles. The same Coordinator's next timer wake admits and launches one exact reviewer without redelivery/restart. Actual PostgreSQL17 race regression is RED before/GREEN after `610a2df2`; host/provider effects remain substituted. |
| TestCoordinatorWakeClosesExpiredRootWithoutMoreRuns; progressLiveHost; file scaffolding | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Finite original root closes without replenishment or additional launches/publications. |

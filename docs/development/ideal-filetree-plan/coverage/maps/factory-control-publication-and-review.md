# Factory control publication and review

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-491df64a7c65"></a>
<a id="internalfactorycontrolmergego-1"></a>

## [internal/factory/control/merge.go](../../../../../internal/factory/control/merge.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–185; file scaffold; MergeExecutor; mergePassLimit; MergeLink; MergeWait; MergeError; MergeReport; MergePass; mergeError; mergeWait; mergeOne; mergeCallError; storeMerge; finishMerge | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fcde0088ef72"></a>

## [internal/factory/control/merge_effect_test.go](../../../../../internal/factory/control/merge_effect_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–199; file scaffold; TestMergePassReconcilesLostSubmitReply; TestMergePassWaitsForNativeCompletion; TestMergePassFencesUnconfirmedCompletion; TestMergePassFencesIndeterminateEffect; TestMergePassWithdrawsBeforeSubmit; TestMergePassWithdrawalAfterCommitConfirmsCompletion; TestMergeCompletionReleasesCodeDependant | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-26f25db87a30"></a>

## [internal/factory/control/merge_evidence.go](../../../../../internal/factory/control/merge_evidence.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–136, 151–188; file scaffold; observeMergeEvidence; mergeAuthority; completeMerge | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 137–150; mergeWork | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: mergeWork — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bbbc155fd9cd"></a>

## [internal/factory/control/merge_fixture_test.go](../../../../../internal/factory/control/merge_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–195; file scaffold; fakeMerger; ObserveMerge; SubmitMerge; LookupOp; CancelOp; AdoptMerge; ObserveCompletion; happyMerger; pendingMerger; mergeFixture; mergeSeed; wire; pass; merge | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b897a247ace9"></a>
<a id="internalfactorycontrolmerge_native_testgo-1"></a>

## [internal/factory/control/merge_native_test.go](../../../../../internal/factory/control/merge_native_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38; file scaffold; nativeMergePR | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; Current declaration duty: nativeMergePR — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 39–79; nativeMergeAPI | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current declaration duty: nativeMergeAPI — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 80–106; nativeMergePublish | [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) | retained | Current declaration duty: nativeMergePublish — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 107–131, 181–214; TestNativeMergeWire; TestNativeMergeConfirmationSnapshot | [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) | retained | Current declaration duty: TestNativeMergeWire; Current declaration duty: TestNativeMergeConfirmationSnapshot — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 132–180; TestNativeMergeReviewSnapshot | [G05](../../slices/forgejo-integration.md#g05-native-review-submission) | retained | Current declaration duty: TestNativeMergeReviewSnapshot — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6f2f0070ca92"></a>

## [internal/factory/control/merge_reconcile.go](../../../../../internal/factory/control/merge_reconcile.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 168–214; file scaffold; adoptMergeReceipt; adoptMergeObserved; mergeOperationOutcomeOf | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–167; reconcileMerge | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: reconcileMerge — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-eedaccbbd850"></a>
<a id="internalfactorycontrolmerge_testgo-1"></a>

## [internal/factory/control/merge_test.go](../../../../../internal/factory/control/merge_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–263; file scaffold; TestMergePassMergesOneExactPR; TestMergePassSkipsRowWithoutCurrentPass; TestMergePassWaitsForStaleCheckEvidence; TestMergePassRefusesFailedCheckEvidence; TestMergePassWaitsForPendingChecks; TestMergePassRefusesRegressedChecks; TestMergePassWaitsForApproval; TestMergePassRefusesChangeRequest; TestMergePassRequiresCurrentAuthority | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-31f3fa2e3a45"></a>

## [internal/factory/control/merge_withdraw.go](../../../../../internal/factory/control/merge_withdraw.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–89; file scaffold; cancelRepositoryMerges; cancelAcceptanceMerges; cancelMerges; withdrawMerge | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0a96a43913b6"></a>
<a id="internalfactorycontrolpublicationgo-1"></a>

## [internal/factory/control/publication.go](../../../../../internal/factory/control/publication.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–187; file scaffold; PublicationExecutor; publishPassLimit; PublishLink; PublishWait; PublishError; PublishReport; PublishPass; publicationError; publicationWait; publishOne; assignmentForPublication; completePublication; publicationCallError; cancelRepositoryPublications; cancelAcceptancePublications; exportTerminal | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; `publishAfterSettle` retired in `6cc4771c`; retained named duties assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="r02-current-path-internal-factory-control-publication-children-go"></a>

## [internal/factory/control/publication_children.go](../../../../../internal/factory/control/publication_children.go)

Current source addition at `b38a37b4`; existing coordinator and Store owners remain in place.

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| File scaffold; publicationChildLimit; producePublishedChildren; producePublicationChild; currentReviewReport; currentChildCheckAssessment; prepareReviewCandidate; candidatePreparationStateMatches; candidatePreparationIdentityMatches; observeCandidatePreparation; executePublicationChild; existingMatchesPublicationChild; publicationChildID; publicationPreparationID; fenceChildPublication | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Produces exact-head review/correction children, invokes canonical preparation/dispatch admission and fences failed or exhausted continuation. Store retains authority, deadline and retirement state ownership. Source/development evidence is recorded in [F07](../../reviews/F07.md#current-prompt-and-preparation-observation-source-cuts); full exhaustion/intervention and native qualification remain open. |
| buildPublicationChildPrompt | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Uses canonical recorded prompt inputs and the role-specific output contract; bounded repository material remains pending. |

## [internal/factory/control/publication_children_test.go](../../../../../internal/factory/control/publication_children_test.go)

| Current named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| File scaffold; producerFlowHost and methods; publishedChildProducerSeed; settleProducedChild; TestPublishPassProducesExactReviewerChildAndReplays; TestPublishPassProducesCorrectionThenFreshReviewForChangedHead | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Uses the production controller and Store with substituted host effects; proves exact child admission, replay and changed-head review, not native/provider execution. |
| TestPauseKeepsRunningCandidatePreparationUntilStopReceiptConfirms | [F08](../../slices/factory-coordination.md#f08-execution-lifecycle) | retained | Confirmed stop releases custody; uncertain and failed/uncertain preparation receipts retain it and refuse further native effects. |

<a id="coverage-f09-correction-reconcile"></a>

## [internal/factory/control/correction.go](../../../../../internal/factory/control/correction.go)

Correction registration and recovery have mixed F09/F10 responsibilities. F09 owns the shared gate, cancellation, and publication recovery joins; F10 retains correction creation and the correction chain.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| F09 overlay at `0352f4ba`: correction gate recheck, recorded-operation adoption, and cancellation integration; F10: `PublishCorrection` and correction-chain progression/recovery | [F09](../../slices/factory-coordination.md#f09-publication-progression); [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | F09's bounded Store/coordinator subcut serializes registration and withdrawal, and covers cancellation/recovery joins. F10 remains responsible for creating and advancing the correction chain. The `0352f4ba` packet passed 31 actual PG17 checks (six Store, 25 coordinator), with no fixture skips; two initial fixture failures remain recorded. This does not close broader B04.C or claim native producer qualification. |

<a id="coverage-f09-correction-tests"></a>

## [internal/factory/control/correction_test.go](../../../../../internal/factory/control/correction_test.go)

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| At `0352f4ba`: TestPublishCorrectionAdvancesHead; TestPublishCorrectionRefusesWithoutChange (F10); TestWithdrawalCancelsRecordedCorrectionAndKeepsPublishedHead; TestWithdrawalWaitsForCommittedCorrectionCompletion (F09/F10 join) | [F09](../../slices/factory-coordination.md#f09-publication-progression); [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | The first two regressions exercise correction-chain behavior; the withdrawal cases exercise cancellation and committed-completion recovery across both owners. All scoped checks ran against disposable PG17; native-provider behavior remains unqualified. |

<a id="coverage-f09-publication-reconcile"></a>

## [internal/factory/control/publication_reconcile.go](../../../../../internal/factory/control/publication_reconcile.go)

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| F09 published-stage reconciliation of persisted correction operations, as added at `0352f4ba` | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | This bounded path reloads the assignment/run/gate and reconciles the recorded operation; pending completion remains recoverable. It does not transfer F10's correction-chain creation/progression responsibility. |

<a id="coverage-f09-publication-withdraw"></a>

## [internal/factory/control/publication_withdraw.go](../../../../../internal/factory/control/publication_withdraw.go)

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| F09 withdrawal gate, recorded-operation cancellation and completion-pending handling, as added at `0352f4ba` | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Withdrawal materializes/uses the shared gate, records exact cancellation and preserves committed-but-incomplete state for later recovery. Correction-chain creation remains F10. Broader B04.C remains open for other registration/review duties. |

<a id="coverage-cd3691f7a87a"></a>

## [internal/factory/control/publication_hooks_test.go](../../../../../internal/factory/control/publication_hooks_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–80; whole file; publicationDispatchHost; publicationDispatchHost.FactoryStop; publicationDispatchHost.FactoryExport; TestPublicationDispatchHandoffAccountsAndPublishesRecordedResult | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 81–140; TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope, TestPublicationReplacementAcceptanceCancelsPredecessorAndReplaysReceipt | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Accepted requirement invalidation coordinates conditional publication cancellation; declarations/fields: `TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope`, `TestPublicationReplacementAcceptanceCancelsPredecessorAndReplaysReceipt` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| Current named tests: TestPublicationGrantChangeReportsNativePendingSeparatelyFromDispatch, TestPublicationPauseReceiptPreservesPendingNativeCancellation, TestReconcileSettlesOnlyAndReplaysWithoutAdvancement | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Lifecycle receipts preserve independent native effect state. At `6cc4771c`, the obsolete Reconcile-publication expectation is superseded by actual settlement, replay and no dispatch/publication/review/check/merge effects. |

<a id="coverage-518ebb905ac6"></a>
<a id="internalfactorycontrolpublication_native_testgo-1"></a>

## [internal/factory/control/publication_native_test.go](../../../../../internal/factory/control/publication_native_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; file scaffold; TestNativePublishCandidate; TestNativePublishCorrection; TestNativePublishWrongRefs; TestNativePublishExtraRefs; nativeLookalike; ObservePublication | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1ac2ec9fa4dc"></a>
<a id="internalfactorycontrolpublication_testgo-1"></a>

## [internal/factory/control/publication_test.go](../../../../../internal/factory/control/publication_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–71; file scaffold; TestPublishPassLinksOneExactPR; TestPublishPassRefusesOccupiedTarget; TestPublishPassRejectsChangedAcceptedSource; TestPublishPassRequiresAcceptanceAndRefsAtSameRevision | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0423e2dfe6c9"></a>

## [internal/factory/control/review_cycle.go](../../../../../internal/factory/control/review_cycle.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 106–252, 254–308, 340–350; reviewCyclePassLimit; Coordinator.SubmitReviewForRun; sameReviewOperationOutcome; Coordinator.reconcileRecordedReviews | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Immutable review work and operation outcomes are recorded before native submit behind the dispatch gate. Retries look up only the saved operation; reconciliation retains uncertain/cancelled/committed outcomes. `f8e14d39` source/owned-PG scope; the autonomous F10 loop and native producer qualification remain open. `reviewAfterSettle` is retired in `6cc4771c`; bounded PublishPass child consumption owns that handoff. |
| 24–77; Coordinator.CheckPass, Coordinator.checkCurrent | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current exact-candidate check assessment progression; declarations/fields: `Coordinator.CheckPass`, `Coordinator.checkCurrent` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 78–104; Coordinator.progressAfterPublish | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Post-publication review/check/merge progression routing; branches partitioned below; declarations/fields: `Coordinator.progressAfterPublish` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| Coordinator.correctAfterSettle (removed) | [F09](../../slices/factory-coordination.md#f09-publication-progression) | obsolete | Retired in `6cc4771c`; shared settlement performs accounting only. Existing automatic PublishPass consumes the stored correction result. |

<a id="coverage-3379ae9f7b2d"></a>

## [internal/factory/control/review_cycle_test.go](../../../../../internal/factory/control/review_cycle_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 115–529; fakeReviewer; fakeReviewer.ObserveReview; fakeReviewer.SubmitReview; fakeReviewer.LookupOp; fakeReviewer.CancelOp; fakeReviewer.AdoptReview; reviewRunSeed; TestSubmitReviewForRunUnavailable; TestSubmitReviewForRunSubmitsExactHead; TestSubmitReviewForRunRefusesStaleHead; TestPublishPassSubmitsSettledReviewChild; TestReviewRetryUsesSavedWorkAfterHeadAndPolicyChange; TestWithdrawalCancelsRecordedReviewOperation; TestReviewRegistrationRefusesClosedDispatchBeforeSubmit | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Existing cases plus the `f8e14d39` regressions cover saved-tuple retry after head/policy change, cancellation, and refusal before submit when dispatch is closed. They establish the bounded F10-F2 subcut, not the autonomous review/correction loop. At `6cc4771c`, Stop captures the result without native submission; the actual subsequent PublishPass submits it. |
| 14–69; TestCheckPassUnavailableWithoutAssessor, TestCheckPassAssessesPublishedOnce, TestCheckPassReassessesFailedHead | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Check assessment progression assertions; declarations/fields: `TestCheckPassUnavailableWithoutAssessor`, `TestCheckPassAssessesPublishedOnce`, `TestCheckPassReassessesFailedHead` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 70–114; TestCheckPassOpensMergeOnFreshPass | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Fresh passing check assessment opens merge eligibility; declarations/fields: `TestCheckPassOpensMergeOnFreshPass` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 530–571; TestRetryTruthfulReasons | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Retry reason/state assertions; declarations/fields: `TestRetryTruthfulReasons` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-baa86fd5d2e1"></a>

## [internal/factory/control/review_executor.go](../../../../../internal/factory/control/review_executor.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; file scaffold; ReviewExecutor | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; Current declaration duty: ReviewExecutor — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0ff4beb1e778"></a>

## [internal/factory/control/review_native_primitive_test.go](../../../../../internal/factory/control/review_native_primitive_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; file scaffold | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 24–43, 84–320; nativeST10Config; nativeReviewConfig; nativeReviewObserved; TestNativeReviewPrimitive; TestNativeReviewStaleHeadDirect; TestNativeReviewRoleBinding | [G05](../../slices/forgejo-integration.md#g05-native-review-submission) | retained | Current declaration duty: nativeST10Config; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 44–83; TestNativeReviewWire | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | Current declaration duty: TestNativeReviewWire — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c211cd31000f"></a>

## [internal/factory/control/st15_demo_review_test.go](../../../../../internal/factory/control/st15_demo_review_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18, 39–84, 145–178; file scaffold; reviewLeg1; ciFail; requireContentStatus; reviewLeg2 | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current review/CI/merge-completion journey duties; the former direct reviewer-preparation helper was removed in favor of the existing production child producer. |
| 19–38, 85–144, 179–185; reviewLeg; correctA; st15CheckLink | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Drives and validates the current candidate through production reviewer/correction child assignments and recorded check links. |


## Current path reconciliation at `45ebf4c4`

This section reconciles current path ownership only. Existing numeric/body selectors above remain pinned to `519b76bd` unless a row explicitly gives a narrow current inspection.

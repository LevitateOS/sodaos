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
| 1–187; file scaffold; PublicationExecutor; publishPassLimit; PublishLink; PublishWait; PublishError; PublishReport; PublishPass; publicationError; publicationWait; publishAfterSettle; publishOne; assignmentForPublication; completePublication; publicationCallError; cancelRepositoryPublications; cancelAcceptancePublications; exportTerminal | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 18 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cd3691f7a87a"></a>

## [internal/factory/control/publication_hooks_test.go](../../../../../internal/factory/control/publication_hooks_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–80; whole file; publicationDispatchHost; publicationDispatchHost.FactoryStop; publicationDispatchHost.FactoryExport; TestPublicationDispatchHandoffAccountsAndPublishesRecordedResult | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 81–140; TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope, TestPublicationReplacementAcceptanceCancelsPredecessorAndReplaysReceipt | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Accepted requirement invalidation coordinates conditional publication cancellation; declarations/fields: `TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope`, `TestPublicationReplacementAcceptanceCancelsPredecessorAndReplaysReceipt` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 141–215; TestPublicationGrantChangeReportsNativePendingSeparatelyFromDispatch, TestPublicationPauseReceiptPreservesPendingNativeCancellation, TestPublicationReconcileReceiptPersistsExactLink | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Factory lifecycle preserves independent native publication effect state; declarations/fields: `TestPublicationGrantChangeReportsNativePendingSeparatelyFromDispatch`, `TestPublicationPauseReceiptPreservesPendingNativeCancellation`, `TestPublicationReconcileReceiptPersistsExactLink` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

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
| 1–23, 105–186, 210–220; whole file; reviewCyclePassLimit; Coordinator.SubmitReviewForRun; Coordinator.reviewAfterSettle | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 24–77; Coordinator.CheckPass, Coordinator.checkCurrent | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current exact-candidate check assessment progression; declarations/fields: `Coordinator.CheckPass`, `Coordinator.checkCurrent` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 78–104; Coordinator.progressAfterPublish | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Post-publication review/check/merge progression routing; branches partitioned below; declarations/fields: `Coordinator.progressAfterPublish` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 187–209; Coordinator.correctAfterSettle | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Correction publication progression after run retirement; declarations/fields: `Coordinator.correctAfterSettle` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-3379ae9f7b2d"></a>

## [internal/factory/control/review_cycle_test.go](../../../../../internal/factory/control/review_cycle_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 115–283; whole file; fakeReviewer; fakeReviewer.ObserveReview; fakeReviewer.SubmitReview; fakeReviewer.LookupOp; fakeReviewer.CancelOp; fakeReviewer.AdoptReview; reviewRunSeed; TestSubmitReviewForRunUnavailable; TestSubmitReviewForRunSubmitsExactHead; TestSubmitReviewForRunRefusesStaleHead; TestStopSubmitsSettledReview | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 12 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 14–69; TestCheckPassUnavailableWithoutAssessor, TestCheckPassAssessesPublishedOnce, TestCheckPassReassessesFailedHead | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Check assessment progression assertions; declarations/fields: `TestCheckPassUnavailableWithoutAssessor`, `TestCheckPassAssessesPublishedOnce`, `TestCheckPassReassessesFailedHead` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 70–114; TestCheckPassOpensMergeOnFreshPass | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Fresh passing check assessment opens merge eligibility; declarations/fields: `TestCheckPassOpensMergeOnFreshPass` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 284–325; TestRetryTruthfulReasons | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Retry reason/state assertions; declarations/fields: `TestRetryTruthfulReasons` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

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
| 1–92, 115–164, 211–244; file scaffold; prepareReviewer; reviewLeg1; ciFail; requireContentStatus; reviewLeg2 | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 93–114, 165–210, 245–252; reviewLeg; correctA; st15CheckLink | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: reviewLeg; Current declaration duty: correctA; Current declaration duty: st15CheckLink — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

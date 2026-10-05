# Factory control publication and review

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-491df64a7c65"></a>

<a id="internalfactorycontrolmergego-1"></a>

## [internal/factory/control/merge.go](../../../../../internal/factory/control/merge.go)

Committed 018740f4 source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 21–31 | Record, DTO or interface contract MergeExecutor for Merge eligibility and completion; declarations/fields: `MergeExecutor` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 32–34 | Declared identifiers/bounds mergePassLimit for Merge eligibility and completion; declarations/fields: `mergePassLimit` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 35–46 | Record, DTO or interface contract MergeLink for Merge eligibility and completion; declarations/fields: `MergeLink` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 47–54 | Record, DTO or interface contract MergeWait for Merge eligibility and completion; declarations/fields: `MergeWait` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 55–63 | Record, DTO or interface contract MergeError for Merge eligibility and completion; declarations/fields: `MergeError` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 64–75 | Record, DTO or interface contract MergeReport for Merge eligibility and completion; declarations/fields: `MergeReport` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 76–95 | Merge lifecycle pass reconciles recorded rows before selecting further published candidates; availability and bounded store reads belong to the same merge operation; declarations/fields: `Coordinator.MergePass` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 96–103 | Open a merge row only when existing checkCurrent confirms the exact current head/base passing assessment; consume F11 assessment without creating a second verdict owner; declarations/fields: `Coordinator.MergePass`, `Coordinator.checkCurrent` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 104–108 | After eligibility succeeds, create/reconcile one merge row and return merge progress report; declarations/fields: `Coordinator.MergePass`, `Coordinator.mergeOne` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 109–111 | mergeError — Merge eligibility and completion; declarations/fields: `mergeError` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 112–115 | mergeWait — Merge eligibility and completion; declarations/fields: `mergeWait` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 116–153 | Coordinator.mergeOne — Merge eligibility and completion; declarations/fields: `Coordinator.mergeOne` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 154–315 | Coordinator.reconcileMerge — Merge eligibility and completion; declarations/fields: `Coordinator.reconcileMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 316–387 | Coordinator.observeMergeEvidence — Merge eligibility and completion; declarations/fields: `Coordinator.observeMergeEvidence` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 388–432 | Coordinator.mergeAuthority — Merge eligibility and completion; declarations/fields: `Coordinator.mergeAuthority` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 433–442 | Coordinator.mergeWork — Merge eligibility and completion; declarations/fields: `Coordinator.mergeWork` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 443–466 | Coordinator.adoptMergeReceipt — Merge eligibility and completion; declarations/fields: `Coordinator.adoptMergeReceipt` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 467–505 | Coordinator.completeMerge — Merge eligibility and completion; declarations/fields: `Coordinator.completeMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 506–518 | Coordinator.mergeCallError — Merge eligibility and completion; declarations/fields: `Coordinator.mergeCallError` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 519–522 | Coordinator.cancelRepositoryMerges — Merge eligibility and completion; declarations/fields: `Coordinator.cancelRepositoryMerges` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 523–526 | Coordinator.cancelAcceptanceMerges — Merge eligibility and completion; declarations/fields: `Coordinator.cancelAcceptanceMerges` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 527–562 | Coordinator.cancelMerges — Merge eligibility and completion; declarations/fields: `Coordinator.cancelMerges` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 563–599 | Coordinator.withdrawMerge — Merge eligibility and completion; declarations/fields: `Coordinator.withdrawMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 600–622 | Coordinator.adoptMergeObserved — Merge eligibility and completion; declarations/fields: `Coordinator.adoptMergeObserved` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 623–627 | mergeOperationOutcomeOf — Merge eligibility and completion; declarations/fields: `mergeOperationOutcomeOf` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 628–636 | Coordinator.storeMerge — Merge eligibility and completion; declarations/fields: `Coordinator.storeMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 637–650 | Coordinator.finishMerge — Merge eligibility and completion; declarations/fields: `Coordinator.finishMerge` |

<a id="coverage-b897a247ace9"></a>

<a id="internalfactorycontrolmerge_native_testgo-1"></a>

## [internal/factory/control/merge_native_test.go](../../../../../internal/factory/control/merge_native_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim. New nativeMergeOpenRow directly seeds records for stale-head/base, withdrawal and authority-change reconciliation scenarios; these scenarios do not exercise MergePass opening eligibility. Production check assessment and publication fixture helpers retain distinct F11/F09 ownership. Native REST issue/dependency seeding is explicit staged fixture setup, not production authorization or proof that upstream behavior ran.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1–34 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 35–40 | Fixture/protocol support nativeST12Config; declarations/fields: `nativeST12Config` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 41–69 | Fixture/protocol support loadNativeST12: native fixture requires explicit repository and merger inputs; declarations/fields: `loadNativeST12` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 70–89 | Fixture/protocol support nativeMergeReceipt: ST12_RECEIPT_DIR is required to retain native proof; declarations/fields: `nativeMergeReceipt` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 90–123, 130–141 | Native background admission/protocol fixture assertions; declarations/fields: `nativeMergeDrain`, `nativeMergeIdle`, `nativeMergeBusyError` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 124–129 | Fixture/protocol support nativeMergeStaleRefusal; declarations/fields: `nativeMergeStaleRefusal` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 142–160 | Fixture/protocol support nativeMergeCall: native %s unanswered: %v; declarations/fields: `nativeMergeCall` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 161–199, 539–557 | Separate native review binding/submission driver; declarations/fields: `nativeMergeSubmitReviewCommitted`, `nativeMergeApprove` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 200–266, 347–371, 421–459, 1050–1063, 1069–1078, 1093–1101 | Conditional native merge effect/receipt fixture; declarations/fields: `nativeMergeSubmitMergeCommitted`, `nativeMergeSubmitRawCommitted`, `TestNativeMergeWire`, `TestNativeMergeConfirmationSnapshot`, `dropOnceMerger.SubmitMerge`, `countMerger.SubmitMerge`, `lagOnceMerger.SubmitMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 267–278 | Fixture/protocol support nativeMergePR; declarations/fields: `nativeMergePR` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 279–319 | Opt-in native fixture REST transport for explicit method/path/body calls, including native issue creation; this helper does not own production read or issue-mutation policy; declarations/fields: `nativeMergeAPI` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 320–346, 753–768 | Native candidate publication preparation/effect driver; declarations/fields: `nativeMergePublish`, `nativeMergeGit` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 372–420 | Native review projection fixture; declarations/fields: `TestNativeMergeReviewSnapshot` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 460–465, 470–476 | Fixture/protocol support nativeMergeSetup; declarations/fields: `nativeMergeSetup` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 466–469 | Native fixture sets the existing repository policy's independent reviewer binding used by subsequent merge scenarios; declarations/fields: `nativeMergeSetup`, `policy.Review` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 477–495 | Native fixture confirms its creator's code-write authority and creates an upstream issue as intake input for this merge scenario; upstream native mutation is fixture setup; declarations/fields: `nativeMergeAssignment` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 496–510 | Native fixture reads exact issue evidence and admits its factory accepted-requirements decision through production AdmitAcceptance, retaining the acceptance receipt; declarations/fields: `nativeMergeAssignment`, `Coordinator.AdmitAcceptance` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 511–525 | Native fixture constructs the recorded coder assignment with effective authority, requirement/approval heads, preparation and exact prompt/run references; declarations/fields: `nativeMergeAssignment`, `factory.Assignment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 526–527 | Native fixture registers one dispatch packet atomically with assignment, reservation, run and run-view references; dependency records retain their F03/F08 authorities; declarations/fields: `nativeMergeAssignment`, `Store.RecordDispatchPacket` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 528–529 | Native fixture marks the stopped run reconciled/succeeded and stores its final summary for publication setup; declarations/fields: `nativeMergeAssignment`, `Store.SaveFactoryRun` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 530 | Native fixture consumes the assignment's held capacity reservation after the stopped run; declarations/fields: `nativeMergeAssignment`, `Store.ConsumeReservation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 531–538 | Native fixture records the coder assignment's completed candidate result, finishes it and returns the assignment; declarations/fields: `nativeMergeAssignment`, `Store.FinishAssignment` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 558–580 | Native fixture drives production check observation, verifies exact candidate results against adopted repository policy, and records the assessment consumed by merge eligibility; declarations/fields: `nativeMergeAssess` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 581–603 | Native test fixture directly seeds an open Merge row for subsequent reconciliation gates; it deliberately bypasses MergePass row-opening eligibility; declarations/fields: `nativeMergeOpenRow` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 604–630, 689–752, 794–1043, 1112–1429 | Factory merge eligibility/progression exercised by opt-in native fixture; declarations/fields: `nativeMergeDrive`, `TestNativeMergeFullPass`, `nativeMergeFullPassAttempt`, `TestNativeMergeStaleHead`, `TestNativeMergeStaleBase`, `TestNativeMergeWithdrawBeforeSubmit`, `TestNativeMergeAuthorityChanged`, `nativeMergeAuthorityAttempt`, `TestNativeMergeCancelBeforeSubmit`, `TestNativeMergeCancelAfterCommit`, `TestNativeMergeLostReply`, `nativeMergeLostReplyAttempt`, `TestNativeMergeCommittedIncomplete`, `nativeMergeCommittedIncompleteAttempt`, `TestNativeMergeDependantRunnable`, `nativeMergeDependantAttempt`, `TestNativeMergeProtectionRefuses`, `nativeMergeProtectionAttempt` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 631–658 | Native fixture drives publication progression until its candidate publication leaves open/fenced; declarations/fields: `nativeMergePublishDrive` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 659–680 | Native publication fixture seeds a candidate assignment, drives publication and reseeds only after a certain stale native refusal; declarations/fields: `nativeMergeSeedPublication` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 681–688 | Staged native issue dependency fixture inserts a blocked-by-blocker edge for readiness and dependant-release scenarios; declarations/fields: `nativeMergeInsertEdge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 769–793 | Fixture/protocol support nativeMergeAdvanceBranch: native fixture git clone failed: %v; declarations/fields: `nativeMergeAdvanceBranch` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1044–1049 | Fixture/protocol support dropOnceMerger; declarations/fields: `dropOnceMerger` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1064–1068 | Fixture/protocol support countMerger; declarations/fields: `countMerger` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1079–1084 | Fixture/protocol support lagOnceMerger; declarations/fields: `lagOnceMerger` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1085–1092 | Fixture/protocol support lagOnceMerger.lag; declarations/fields: `lagOnceMerger.lag` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1102–1111 | Fixture/protocol support lagOnceMerger.LookupOp; declarations/fields: `lagOnceMerger.LookupOp` |

<a id="coverage-eedaccbbd850"></a>

<a id="internalfactorycontrolmerge_testgo-1"></a>

## [internal/factory/control/merge_test.go](../../../../../internal/factory/control/merge_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. New opening-gate test uses the existing database fixture and fakeMerger. Stale/failed stored-evidence tests open a row with pendingMerger before regressing evidence, so their scope is reconciliation of an already-open row.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 12–27 | Fixture/protocol support fakeMerger; declarations/fields: `fakeMerger` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 28–32 | Fixture/protocol support fakeMerger.ObserveMerge; declarations/fields: `fakeMerger.ObserveMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 33–43 | Fixture/protocol support fakeMerger.SubmitMerge; declarations/fields: `fakeMerger.SubmitMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 44–55 | Fixture/protocol support fakeMerger.LookupOp; declarations/fields: `fakeMerger.LookupOp` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 56–71 | Fixture/protocol support fakeMerger.CancelOp; declarations/fields: `fakeMerger.CancelOp` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 72–75 | Fixture/protocol support fakeMerger.AdoptMerge; declarations/fields: `fakeMerger.AdoptMerge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 76–80 | Fixture/protocol support fakeMerger.ObserveCompletion; declarations/fields: `fakeMerger.ObserveCompletion` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 81–114 | Fixture/protocol support happyMerger; declarations/fields: `happyMerger` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 115–129 | Fake native observation fixture clears live checks while leaving an already-open merge row available for evidence-regression reconciliation assertions; declarations/fields: `pendingMerger` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 130–134 | Fixture/protocol support mergeFixture; declarations/fields: `mergeFixture` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 135–174 | Fixture/protocol support mergeSeed: merge seed publication not published: %+v; declarations/fields: `mergeSeed` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 175–179 | Fixture/protocol support mergeFixture.wire; declarations/fields: `mergeFixture.wire` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 180–188 | Fixture/protocol support mergeFixture.pass: merge pass: %+v; declarations/fields: `mergeFixture.pass` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 189–197 | Fixture/protocol support mergeFixture.merge; declarations/fields: `mergeFixture.merge` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 198–225 | Assertions TestMergePassMergesOneExactPR: merge linkage: %+v %+v; declarations/fields: `TestMergePassMergesOneExactPR` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 226–260 | Assertions TestMergePassReconcilesLostSubmitReply: lost reply misadvanced: %+v %+v; declarations/fields: `TestMergePassReconcilesLostSubmitReply` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 261–293 | Source test asserts failed stored assessment opens no merge row, then a recorded passing assessment admits and completes the same publication using the fake merger; declarations/fields: `TestMergePassSkipsRowWithoutCurrentPass` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 294–332 | Assertions TestMergePassWaitsForStaleCheckEvidence: stale evidence misadvanced: %+v %+v; declarations/fields: `TestMergePassWaitsForStaleCheckEvidence` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 333–362 | Assertions TestMergePassRefusesFailedCheckEvidence: failed checks merged: %+v; declarations/fields: `TestMergePassRefusesFailedCheckEvidence` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 363–392 | Assertions TestMergePassWaitsForPendingChecks: pending checks misadvanced: %+v %+v; declarations/fields: `TestMergePassWaitsForPendingChecks` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 393–412 | Assertions TestMergePassRefusesRegressedChecks: regressed checks merged: %+v; declarations/fields: `TestMergePassRefusesRegressedChecks` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 413–436 | Assertions TestMergePassWaitsForNativeCompletion: incomplete merge finished: %+v %+v; declarations/fields: `TestMergePassWaitsForNativeCompletion` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 437–450 | Assertions TestMergePassFencesUnconfirmedCompletion: unconfirmed completion merged: %+v; declarations/fields: `TestMergePassFencesUnconfirmedCompletion` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 451–464 | Assertions TestMergePassFencesIndeterminateEffect: indeterminate effect unfinished: %+v; declarations/fields: `TestMergePassFencesIndeterminateEffect` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 465–489 | Assertions TestMergePassWaitsForApproval: unapproved candidate misadvanced: %+v %+v; declarations/fields: `TestMergePassWaitsForApproval` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 490–503 | Assertions TestMergePassRefusesChangeRequest: rejected candidate merged: %+v; declarations/fields: `TestMergePassRefusesChangeRequest` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 504–538 | Assertions TestMergePassRequiresCurrentAuthority: seed merge unsettled: %+v; declarations/fields: `TestMergePassRequiresCurrentAuthority` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 539–560 | Assertions TestMergePassWithdrawsBeforeSubmit: seed observed despite busy native: %+v; declarations/fields: `TestMergePassWithdrawsBeforeSubmit` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 561–594 | Assertions TestMergePassWithdrawalAfterCommitConfirmsCompletion: seed merge never submitted; declarations/fields: `TestMergePassWithdrawalAfterCommitConfirmsCompletion` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 595–640 | Assertions TestMergeCompletionReleasesCodeDependant: dependant runnable before completion: %+v; declarations/fields: `TestMergeCompletionReleasesCodeDependant` |

<a id="coverage-0a96a43913b6"></a>

<a id="internalfactorycontrolpublicationgo-1"></a>

## [internal/factory/control/publication.go](../../../../../internal/factory/control/publication.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1–22 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 23–35 | Record, DTO or interface contract PublicationExecutor for Publication progression; declarations/fields: `PublicationExecutor` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 36–38 | Declared identifiers/bounds publishPassLimit for Publication progression; declarations/fields: `publishPassLimit` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 39–48 | Record, DTO or interface contract PublishLink for Publication progression; declarations/fields: `PublishLink` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 49–56 | Record, DTO or interface contract PublishWait for Publication progression; declarations/fields: `PublishWait` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 57–65 | Record, DTO or interface contract PublishError for Publication progression; declarations/fields: `PublishError` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 66–77 | Record, DTO or interface contract PublishReport for Publication progression; declarations/fields: `PublishReport` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 78–102 | Coordinator.PublishPass — Publication progression; declarations/fields: `Coordinator.PublishPass` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 103–105 | publicationError — Publication progression; declarations/fields: `publicationError` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 106–109 | publicationWait — Publication progression; declarations/fields: `publicationWait` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 110–118 | Coordinator.publishAfterSettle — Publication progression; declarations/fields: `Coordinator.publishAfterSettle` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 119–147 | Coordinator.publishOne — Publication progression; declarations/fields: `Coordinator.publishOne` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 148–162 | Coordinator.assignmentForPublication — Publication progression; declarations/fields: `Coordinator.assignmentForPublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 163–381 | Coordinator.reconcilePublication — Publication progression; declarations/fields: `Coordinator.reconcilePublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 382–387 | terminalPublicationEffect — Publication progression; declarations/fields: `terminalPublicationEffect` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 388–432 | Coordinator.publicationAuthority — Publication progression; declarations/fields: `Coordinator.publicationAuthority` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 433–447 | Coordinator.publicationWork — Publication progression; declarations/fields: `Coordinator.publicationWork` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 448–477 | Coordinator.adoptPublicationReceipts — Publication progression; declarations/fields: `Coordinator.adoptPublicationReceipts` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 478–485 | Coordinator.completePublication — Publication progression; declarations/fields: `Coordinator.completePublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 486–498 | Coordinator.publicationCallError — Publication progression; declarations/fields: `Coordinator.publicationCallError` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 499–502 | Coordinator.cancelRepositoryPublications — Publication progression; declarations/fields: `Coordinator.cancelRepositoryPublications` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 503–506 | Coordinator.cancelAcceptancePublications — Publication progression; declarations/fields: `Coordinator.cancelAcceptancePublications` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 507–553 | Coordinator.cancelPublications — Publication progression; declarations/fields: `Coordinator.cancelPublications` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 554–575 | Coordinator.publishAfterDispatch — Publication progression; declarations/fields: `Coordinator.publishAfterDispatch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 576–619 | Coordinator.withdrawPublication — Publication progression; declarations/fields: `Coordinator.withdrawPublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 620–641 | Coordinator.adoptObserved — Publication progression; declarations/fields: `Coordinator.adoptObserved` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 642–645 | operationOutcomeOf — Publication progression; declarations/fields: `operationOutcomeOf` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 646–653 | Coordinator.storePublication — Publication progression; declarations/fields: `Coordinator.storePublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 654–668 | Coordinator.finishPublication — Publication progression; declarations/fields: `Coordinator.finishPublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 669–670 | Record, DTO or interface contract exportTerminal for Publication progression; declarations/fields: `exportTerminal` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 671–687 | Coordinator.exportCandidate — Publication progression; declarations/fields: `Coordinator.exportCandidate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 688–698 | decodeExportBundle — Publication progression; declarations/fields: `decodeExportBundle` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 699–700 | Declared identifiers/bounds hostRunStale for Publication progression; declarations/fields: `hostRunStale` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 701 | isHostStale — Publication progression; declarations/fields: `isHostStale` |

<a id="coverage-cd3691f7a87a"></a>

## [internal/factory/control/publication_hooks_test.go](../../../../../internal/factory/control/publication_hooks_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 15–21 | Fixture/protocol support publicationDispatchHost; declarations/fields: `publicationDispatchHost` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 22–29 | Fixture/protocol support publicationDispatchHost.FactoryStop; declarations/fields: `publicationDispatchHost.FactoryStop` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 30–33 | Fixture/protocol support publicationDispatchHost.FactoryExport; declarations/fields: `publicationDispatchHost.FactoryExport` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 34–80 | Assertions TestPublicationDispatchHandoffAccountsAndPublishesRecordedResult: dispatch: %+v; declarations/fields: `TestPublicationDispatchHandoffAccountsAndPublishesRecordedResult` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 81–140 | Accepted requirement invalidation coordinates conditional publication cancellation; declarations/fields: `TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope`, `TestPublicationReplacementAcceptanceCancelsPredecessorAndReplaysReceipt` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 141–215 | Factory lifecycle preserves independent native publication effect state; declarations/fields: `TestPublicationGrantChangeReportsNativePendingSeparatelyFromDispatch`, `TestPublicationPauseReceiptPreservesPendingNativeCancellation`, `TestPublicationReconcileReceiptPersistsExactLink` |

<a id="coverage-518ebb905ac6"></a>

<a id="internalfactorycontrolpublication_native_testgo-1"></a>

## [internal/factory/control/publication_native_test.go](../../../../../internal/factory/control/publication_native_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1–37 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 38–52 | Fixture/protocol support nativeST09Config; declarations/fields: `nativeST09Config` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 53–74 | Fixture/protocol support loadNativeST09: native fixture requires explicit repository and publisher inputs; declarations/fields: `loadNativeST09` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 75–80 | Fixture/protocol support nativeMust; declarations/fields: `nativeMust` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 81–86 | Fixture/protocol support nativeSecret; declarations/fields: `nativeSecret` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 87–89 | Fixture/protocol support nativeRepoURL; declarations/fields: `nativeRepoURL` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 90–119 | Native authoritative read fixture or observation projection; declarations/fields: `nativeAPI` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 120–182, 528–535 | Native candidate publication preparation/effect driver; declarations/fields: `nativeGit`, `nativeGitOK`, `nativeCandidate`, `nativeNewCandidate`, `nativeAppendCandidate`, `nativeTip`, `nativeLostReplies.SubmitPublish` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 183–184 | Fixture/protocol support nativeHost; declarations/fields: `nativeHost` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 185–187 | Fixture/protocol support nativeHost.FactoryLaunch; declarations/fields: `nativeHost.FactoryLaunch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 188–190 | Fixture/protocol support nativeHost.FactoryStop; declarations/fields: `nativeHost.FactoryStop` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 191–193 | Fixture/protocol support nativeHost.FactoryInspect; declarations/fields: `nativeHost.FactoryInspect` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 194–196 | Fixture/protocol support nativeHost.FactoryHarness; declarations/fields: `nativeHost.FactoryHarness` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 197–199 | Fixture/protocol support nativeHost.FactoryTakeover; declarations/fields: `nativeHost.FactoryTakeover` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 200–203 | Fixture/protocol support nativeHost.FactoryExport; declarations/fields: `nativeHost.FactoryExport` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 204–205 | Fixture/protocol support nativeBroker; declarations/fields: `nativeBroker` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 206–208 | Fixture/protocol support nativeBroker.GetExecution; declarations/fields: `nativeBroker.GetExecution` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 209–212 | Fixture/protocol support nativeBroker.CloseExecution; declarations/fields: `nativeBroker.CloseExecution` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 213–222 | Fixture/protocol support nativeFixture; declarations/fields: `nativeFixture` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 223–259, 267–349, 365–399, 430–476, 519–521, 562–660, 683–701 | Factory publication progression exercised by opt-in native fixture; declarations/fields: `nativeSeed`, `nativeFixture.assignment`, `nativeFixture.drive`, `TestNativePublishCandidate`, `TestNativePublishCorrection`, `TestNativePublishWrongRefs`, `TestNativePublishExtraRefs`, `TestNativePublishLostReplies`, `TestNativePublishWithdrawal`, `TestNativePublishBranchCommittedPRFailed` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 260–266 | Native background admission/protocol fixture assertions; declarations/fields: `nativeFixture.wire` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 350–364 | Fixture/protocol support nativeReceipt: ST09_RECEIPT_DIR is required to retain native proof; declarations/fields: `nativeReceipt` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 400–405 | Fixture/protocol support nativeFixture.work; declarations/fields: `nativeFixture.work` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 406–412 | Fixture/protocol support nativeFixture.observe; declarations/fields: `nativeFixture.observe` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 413–429 | Fixture/protocol support nativeFixture.terminal: native operation did not reach a terminal effect; declarations/fields: `nativeFixture.terminal` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 477–518 | Fixture/protocol support nativeUnexpectedRefs: operation not registered: %+v; declarations/fields: `nativeUnexpectedRefs` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 522–527 | Fixture/protocol support nativeLostReplies; declarations/fields: `nativeLostReplies` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 536–546 | Fixture/protocol support nativeLostReplies.PushBranch; declarations/fields: `nativeLostReplies.PushBranch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 547–561 | Fixture/protocol support nativeLostReplies.SubmitPRCreate; declarations/fields: `nativeLostReplies.SubmitPRCreate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 661–667 | Fixture/protocol support nativeLookalike; declarations/fields: `nativeLookalike` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 668–682 | Fixture/protocol support nativeLookalike.ObservePublication: lookalike fixture has no native PR; declarations/fields: `nativeLookalike.ObservePublication` |

<a id="coverage-1ac2ec9fa4dc"></a>

<a id="internalfactorycontrolpublication_testgo-1"></a>

## [internal/factory/control/publication_test.go](../../../../../internal/factory/control/publication_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1–19 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 20–25 | Fixture/protocol support publicationTestDB; declarations/fields: `publicationTestDB` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 26–32 | Fixture/protocol support fakePublishHost; declarations/fields: `fakePublishHost` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 33–36 | Fixture/protocol support fakePublishHost.FactoryLaunch; declarations/fields: `fakePublishHost.FactoryLaunch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 37–40 | Fixture/protocol support fakePublishHost.FactoryStop; declarations/fields: `fakePublishHost.FactoryStop` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 41–44 | Fixture/protocol support fakePublishHost.FactoryInspect; declarations/fields: `fakePublishHost.FactoryInspect` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 45–48 | Fixture/protocol support fakePublishHost.FactoryTakeover; declarations/fields: `fakePublishHost.FactoryTakeover` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 49–52 | Fixture/protocol support fakePublishHost.FactoryHarness; declarations/fields: `fakePublishHost.FactoryHarness` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 53–67 | Fixture/protocol support fakePublishHost.FactoryExport; declarations/fields: `fakePublishHost.FactoryExport` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 68–86 | Fixture/protocol support fakePublisher; declarations/fields: `fakePublisher` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 87–91 | Fixture/protocol support fakePublisher.ObservePublication; declarations/fields: `fakePublisher.ObservePublication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 92–96 | Fixture/protocol support fakePublisher.SubmitPublish; declarations/fields: `fakePublisher.SubmitPublish` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 97–101 | Fixture/protocol support fakePublisher.PushBranch; declarations/fields: `fakePublisher.PushBranch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 102–106 | Fixture/protocol support fakePublisher.SubmitPRCreate; declarations/fields: `fakePublisher.SubmitPRCreate` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 107–118 | Fixture/protocol support fakePublisher.LookupOp; declarations/fields: `fakePublisher.LookupOp` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 119–134 | Fixture/protocol support fakePublisher.CancelOp; declarations/fields: `fakePublisher.CancelOp` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 135–138 | Fixture/protocol support fakePublisher.AdoptBranch; declarations/fields: `fakePublisher.AdoptBranch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 139–142 | Fixture/protocol support fakePublisher.AdoptPRCreation; declarations/fields: `fakePublisher.AdoptPRCreation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 143–151 | Fixture/protocol support fakePublisher.record; declarations/fields: `fakePublisher.record` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 152–157 | Fixture/protocol support publicationTestOutcome; declarations/fields: `publicationTestOutcome` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 158–166 | Fixture/protocol support publishFixture; declarations/fields: `publishFixture` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 167–177 | Fixture/protocol support publishSeed; declarations/fields: `publishSeed` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 178–182 | Fixture/protocol support publishFixture.wire; declarations/fields: `publishFixture.wire` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 183–244 | Fixture/protocol support publishFixture.finishReported: fixture authority: %+v %v; declarations/fields: `publishFixture.finishReported` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 245–248 | Fixture/protocol support pendingOutcome; declarations/fields: `pendingOutcome` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 249–255 | Fixture/protocol support committedOutcome; declarations/fields: `committedOutcome` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 256–261 | Fixture/protocol support refusedOutcome; declarations/fields: `refusedOutcome` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 262–295 | Fixture/protocol support happyPublisher; declarations/fields: `happyPublisher` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 296–304 | Fixture/protocol support publishFixture.publication; declarations/fields: `publishFixture.publication` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 305–313 | Fixture/protocol support publishFixture.pass: publication pass: %+v; declarations/fields: `publishFixture.pass` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 314–332 | Assertions TestPublishPassLinksOneExactPR: PR linkage: %+v %+v; declarations/fields: `TestPublishPassLinksOneExactPR` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 333–355 | Assertions TestPublishPassReconcilesLostSubmitReply: lost reply: %+v; declarations/fields: `TestPublishPassReconcilesLostSubmitReply` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 356–397 | Assertions TestPublishPassReplaysUnobservedSubmitWithByteIdenticalIntent: lost reply: %+v; declarations/fields: `TestPublishPassReplaysUnobservedSubmitWithByteIdenticalIntent` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 398–412 | Assertions TestPublishPassRefusesOccupiedTarget: occupied branch adopted: %+v; declarations/fields: `TestPublishPassRefusesOccupiedTarget` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 413–427 | Assertions TestPublishPassRejectsChangedAcceptedSource: changed source authorized publication: %+v; declarations/fields: `TestPublishPassRejectsChangedAcceptedSource` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 428–441 | Assertions TestPublishPassRequiresAcceptanceAndRefsAtSameRevision: cross-revision snapshot submitted: %+v; declarations/fields: `TestPublishPassRequiresAcceptanceAndRefsAtSameRevision` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 442–461 | Assertions TestPublishPassKeepsBranchCommittedPRFailedPartial: partial outcome lost: %+v; declarations/fields: `TestPublishPassKeepsBranchCommittedPRFailedPartial` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 462–477 | Assertions TestPublishPassDoesNotRetryTerminalBranchRefusal: terminal operation replaced: %+v; declarations/fields: `TestPublishPassDoesNotRetryTerminalBranchRefusal` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 478–493 | Assertions TestPublishPassPendingPushIsBoundedPerPass: pending push recursively repeated; declarations/fields: `TestPublishPassPendingPushIsBoundedPerPass` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 494–511 | Assertions TestPublishPassFencesIndeterminateEffect: uncertain effect not fenced: %+v; declarations/fields: `TestPublishPassFencesIndeterminateEffect` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 512–540 | Assertions TestPublishPassWithdrawalFailureStaysOpen: unknown cancellation marked finished: %+v; declarations/fields: `TestPublishPassWithdrawalFailureStaysOpen` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 541–576 | Assertions TestPublishPassLostPRReplyRetainsLateCommittedLinkAndCompletion: lost PR reply: %+v; declarations/fields: `TestPublishPassLostPRReplyRetainsLateCommittedLinkAndCompletion` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 577–596 | Assertions TestPublishPassRecoversCommittedPRWithoutExport: lost PR reply: %+v; declarations/fields: `TestPublishPassRecoversCommittedPRWithoutExport` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 597–608 | Assertions TestPublishPassFailsUnrecoverableExport: missing bundle advanced: %+v; declarations/fields: `TestPublishPassFailsUnrecoverableExport` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 609–615 | Assertions TestPublishPassWaitsWithoutExecutor: report: %+v; declarations/fields: `TestPublishPassWaitsWithoutExecutor` |

<a id="coverage-0423e2dfe6c9"></a>

## [internal/factory/control/review_cycle.go](../../../../../internal/factory/control/review_cycle.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 15–23 | Declared identifiers/bounds reviewCyclePassLimit for Independent review and correction; declarations/fields: `reviewCyclePassLimit` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 24–77 | Current exact-candidate check assessment progression; declarations/fields: `Coordinator.CheckPass`, `Coordinator.checkCurrent` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 78–104 | Post-publication review/check/merge progression routing; branches partitioned below; declarations/fields: `Coordinator.progressAfterPublish` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 105–186 | Coordinator.SubmitReviewForRun — Independent review and correction; declarations/fields: `Coordinator.SubmitReviewForRun` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 187–209 | Correction publication progression after run retirement; declarations/fields: `Coordinator.correctAfterSettle` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 210–220 | Coordinator.reviewAfterSettle — Independent review and correction; declarations/fields: `Coordinator.reviewAfterSettle` |

<a id="coverage-3379ae9f7b2d"></a>

## [internal/factory/control/review_cycle_test.go](../../../../../internal/factory/control/review_cycle_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 14–69 | Check assessment progression assertions; declarations/fields: `TestCheckPassUnavailableWithoutAssessor`, `TestCheckPassAssessesPublishedOnce`, `TestCheckPassReassessesFailedHead` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 70–114 | Fresh passing check assessment opens merge eligibility; declarations/fields: `TestCheckPassOpensMergeOnFreshPass` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 115–121 | Fixture/protocol support fakeReviewer; declarations/fields: `fakeReviewer` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 122–126 | Fixture/protocol support fakeReviewer.ObserveReview; declarations/fields: `fakeReviewer.ObserveReview` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 127–130 | Fixture/protocol support fakeReviewer.SubmitReview; declarations/fields: `fakeReviewer.SubmitReview` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 131–134 | Fixture/protocol support fakeReviewer.LookupOp; declarations/fields: `fakeReviewer.LookupOp` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 135–138 | Fixture/protocol support fakeReviewer.CancelOp; declarations/fields: `fakeReviewer.CancelOp` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 139–142 | Fixture/protocol support fakeReviewer.AdoptReview; declarations/fields: `fakeReviewer.AdoptReview` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 143–185 | Fixture/protocol support reviewRunSeed; declarations/fields: `reviewRunSeed` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 186–193 | Assertions TestSubmitReviewForRunUnavailable: unwired review submission succeeded; declarations/fields: `TestSubmitReviewForRunUnavailable` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 194–222 | Assertions TestSubmitReviewForRunSubmitsExactHead: adopted review: %+v; declarations/fields: `TestSubmitReviewForRunSubmitsExactHead` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 223–236 | Assertions TestSubmitReviewForRunRefusesStaleHead: stale review submitted; declarations/fields: `TestSubmitReviewForRunRefusesStaleHead` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 237–283 | Assertions TestStopSubmitsSettledReview: review stop: %+v %v; declarations/fields: `TestStopSubmitsSettledReview` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 284–325 | Retry reason/state assertions; declarations/fields: `TestRetryTruthfulReasons` |

<a id="coverage-0ff4beb1e778"></a>

## [internal/factory/control/review_native_primitive_test.go](../../../../../internal/factory/control/review_native_primitive_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 1–23 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 24–30 | Fixture/protocol support nativeST10Config; declarations/fields: `nativeST10Config` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 31–43 | Fixture/protocol support nativeReviewConfig: native review needs separate enrolled actor; declarations/fields: `nativeReviewConfig` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 44–83 | Native background admission/protocol fixture assertions; declarations/fields: `TestNativeReviewWire` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 84–102 | Fixture/protocol support nativeReviewObserved: native review observation did not settle; declarations/fields: `nativeReviewObserved` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 103–320 | Separate native review binding/submission driver; declarations/fields: `TestNativeReviewPrimitive`, `TestNativeReviewStaleHeadDirect`, `TestNativeReviewRoleBinding` |


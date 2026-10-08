# Factory coordination

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

## internal/factory/assignment.go

Observed size: 442 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/assignment.go` — Canonical assignment identity, stage and validation.
- `internal/factory/assignment_result.go` — Reported and synthesized harness results.
- `internal/factory/assignment_resources.go` — Canonical reservation and usage records.
- `internal/factory/dispatch_prompt.go` — Accepted input sections and bounded prompt construction.

Evidence: Assignment at 67; ValidAssignmentStage at 36; AssignmentResult at 184; ParseHarnessResult at 248; Reservation at 297; Usage at 326; PromptInputs at 359; BuildDispatchPrompt at 388.

## internal/factory/checks.go

Observed size: 480 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/checks.go` — Check verdicts, resolution reasons and target/adopted definitions.
- `internal/factory/checks_observation.go` — Observed context records and severity classification.
- `internal/factory/checks_assessment.go` — Canonical assessment validation and exact evidence verification.

Evidence: CheckTarget at 123; AdoptedChecks at 161; ObservedChecks at 212; checkStateSeverity at 279; CheckAssessment at 304; VerifyChecks at 406.

## internal/factory/control/acceptance.go

Observed size: 584 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/acceptance.go` — Explicit acceptance admission, immutable command replay and receipts.
- `internal/factory/control/acceptance_evidence.go` — Read and verify native acceptance evidence.
- `internal/factory/control/acceptance_initial.go` — Initial creator-authorized acceptance without overwrite.
- `internal/factory/control/acceptance_status.go` — Current validity assessment and explicit withdrawal.

Evidence: AdmitAcceptance at 125; replayAcceptance at 191; readAcceptanceEvidence at 218; verifyAcceptance at 246; AdmitInitialAcceptance at 322; acceptanceStatus at 410; assessAcceptance at 470; WithdrawAcceptance at 525.

## internal/factory/control/acceptance_test.go

Observed size: 472 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/acceptance_test.go` — Explicit admission and code-route refusal cases.
- `internal/factory/control/acceptance_initial_test.go` — Initial authority, non-overwrite and creator eligibility cases.
- `internal/factory/control/acceptance_status_test.go` — Current validity and withdrawal/replay cases.

Evidence: TestAdmitAcceptanceVerifiesAndRecords at 71; TestAdmitAcceptanceGuardsCodeRoutes at 180; TestAdmitInitialAcceptanceVerifiesCreation at 240; TestAdmitInitialAcceptanceCannotOverwrite at 312; TestAcceptanceStatusAssessesValidity at 329; TestWithdrawAcceptanceLatchesAndReplays at 441.

## internal/factory/control/checks_native_test.go

Observed size: 565 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/checks_native_test.go` — Native verdict, context and policy scenarios.
- `internal/factory/control/checks_native_fixture_test.go` — Shared exact native check fixture and status transport.
- `internal/factory/control/checks_native_stale_test.go` — Head/base/definition and workflow snapshot cases.

Evidence: TestNativeCheckPass at 283; TestNativeCheckUnknownState at 375; TestNativeCheckPolicyEmpty at 466; nativeCheckPublish at 115; nativePostStatus at 170; nativeVerifyChecks at 242; TestNativeCheckStaleHead at 409; TestNativeCheckStaleBase at 483; TestNativeCheckWorkflowPendingSnapshot at 504.

## internal/factory/control/dispatch.go

Current f7 declaration and comment boundaries were checked by B and the
coordinator against the actual 742-line source. Retain the same Go control
package and Coordinator/Store/host/broker subjects; rebind imports per leaf.

| Desired leaf in `internal/factory/control/` | Current complete concern allocation |
| --- | --- |
| `dispatch.go` | 1–252 and 702–742: interfaces, input records, reasons/reports, pass dependencies/entrypoints and coordinator hooks; definitions remain single-owned. |
| `dispatch_occupancy.go` | 253–299: passOccupancy, snapshot and all held-capacity methods. |
| `dispatch_recovery.go` | 300–600: recoverAssigned through recoverOutput, including history, host miss sentinel, custody fence, retry/finish and held-reservation recovery. |
| `dispatch_result.go` | 601–701: finishFromRun, confirmed usage, result derivation/summary and AccountSettledRun. |

These are source-fit allocations. Moving definitions does not resolve workflow
defects or prove execution. The later F07-F1 source correction remains at the
existing Store packet owner, with `factory_dispatch_authority_test.go` and the
controller wait test; the other F07/F08 findings retain their separate scope
and prerequisites in [the execution allocation](../execution-findings.md).

Observed size: 742 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/dispatch.go` — Dispatch dependency records, pass entry and coordinator hooks.
- `internal/factory/control/dispatch_occupancy.go` — Current held capacity and sponsorship occupancy.
- `internal/factory/control/dispatch_recovery.go` — Existing assignment recovery without launching fresh work.
- `internal/factory/control/dispatch_result.go` — Terminal outcome, exact harness result and usage accounting.

Evidence: DispatchDeps at 176; DispatchPass at 212; dispatchDeps at 703; passOccupancy at 256; snapshotOccupancy at 262; heldConn at 290; recoverAssigned at 306; recoverOne at 324; recoverSettled at 438; recoverUnused at 511; finishFromRun at 603; deriveAttemptResult at 638; AccountSettledRun at 672.

## internal/factory/control/dispatch_attempt.go

Current f7 source is 620 lines. Its selected same-package leaves retain complete
declarations and attached comments: `dispatch_attempt.go`1–195 (attemptPlan,
dispatchOne, planWait/admissionWait, refreshOccupancy and planAttempt),
`dispatch_selection.go`196–327 (sponsorship/limits/preparation/harness),
`dispatch_inputs.go`328–417 (accepted reads and promptSections) and
`dispatch_launch.go`418–620 (fresh/retry launch, launchTail, ensureHeldLaunch and
unused settlement). Imports rebind to actual use; no new authority or process.

Observed size: 620 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/dispatch_attempt.go` — Attempt planning and existing readiness refusals.
- `internal/factory/control/dispatch_selection.go` — Sponsorship, resource limits, preparation and harness selection.
- `internal/factory/control/dispatch_inputs.go` — Current accepted native input reads and prompt sections.
- `internal/factory/control/dispatch_launch.go` — Fresh/retry execution, launch receipts and unused launch settlement.

Evidence: attemptPlan at 19; planAttempt at 121; selectSponsorship at 200; checkLimits at 228; selectPreparation at 277; checkHarness at 313; readAttemptInputs at 332; promptSections at 403; executeFreshAttempt at 424; retryAttempt at 500; launchTail at 565; settleUnusedLaunch at 603.

## internal/factory/control/dispatch_test.go

Current f7 source is 1,236 lines. B's full primary assertion reads and the
coordinator's independent declaration/seam, actual shared fixture and selected
production/case reads support the exact allocation below. Independent seam
review does not duplicate every assertion's adequacy or prove native execution.
Retain the real same-package Coordinator/Store tests, one fake/seed definition,
the existing PostgreSQL fixture and the existing deterministic admission barrier.
Distribute source imports1–19 by actual use; attached comments move with cases.

| Desired leaf in `internal/factory/control/` | Current source units |
| --- | --- |
| `dispatch_test.go` | 247–337 and 1060–1066: queue ordering/short visit limit and missing dependencies. |
| `dispatch_fixture_test.go` | 21–246, 338–346 and 965–978: shared digest/host/broker/read/seed fixtures, waitReason and fakeAcceptanceSource; one defining owner. |
| `dispatch_accounting_test.go` | 347–573 and 838–928: accounting, interrupt/approved/fenced effects and settled results. |
| `dispatch_wait_test.go` | 574–837 and 1107–1127: wait reasons, explicit connection refusal and closed dispatch gate. |
| `dispatch_recovery_test.go` | 929–964, 979–1059 and 1067–1106: settled recovery, dependant reassessment, host miss sentinel and intake-triggered dispatch. |
| `dispatch_concurrency_test.go` | 1128–1236: rendezvousReads, mutexDispatchHost and the actual concurrent capacity/budget case. These concern-local fakes stay together. |

Observed size: 1236 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/dispatch_test.go` — Current queue ordering, short limit and required deps cases.
- `internal/factory/control/dispatch_fixture_test.go` — Shared host/broker/read fakes and canonical seed fixture.
- `internal/factory/control/dispatch_accounting_test.go` — Existing interrupt/fence/resource accounting cases.
- `internal/factory/control/dispatch_wait_test.go` — Wait reasons and explicit provider selection refusals.
- `internal/factory/control/dispatch_recovery_test.go` — Settled recovery, dependant reassessment and automatic intake.
- `internal/factory/control/dispatch_concurrency_test.go` — Existing simultaneous dispatch capacity/budget case.

Evidence: TestDispatchPassLaunchesOldestWithinShortLimit at 247; TestDispatchPassWithoutDepsReports at 1060; fakeDispatchHost at 26; fakeDispatchBroker at 61; dispatchSeed at 113; TestDispatchAccountingNeverRefreshes at 347; TestDispatchFencedLaunchStaysHeld at 544; TestAccountSettledRunResults at 838; TestDispatchWaitReasons at 574; TestDispatchNeverFallsBackToAnotherConnection at 787; TestRecoveryConsumesSettledRunWithoutHook at 929; TestCompletionTriggersDependantReassessment at 979; TestIntakeTriggersAutomaticDispatch at 1067; TestConcurrentDispatchPassesPreserveCapacityAndBudget at 1179.

## internal/factory/control/grants.go

Observed size: 502 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/grants.go` — Standing policy/capacity/operator/sponsorship command admission and replay.
- `internal/factory/control/preparation_decisions.go` — Requirement and approval admission with exact replay.
- `internal/factory/control/grant_authority.go` — Effective authority and preparation readiness reads.
- `internal/factory/control/dispatch_registration.go` — Explicit dispatch registration and reopening.

Evidence: ApplyPolicy at 49; ApplySponsorship at 87; applyGrant at 123; AdmitRequirement at 203; AdmitApproval at 221; admitDecision at 246; EffectiveAuthority at 339; preparationReadiness at 392; RegisterDispatch at 423; ReopenDispatch at 445.

## internal/factory/control/lifecycle.go

Observed size: 487 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/lifecycle.go` — Repository pause/resume and tracked run stop.
- `internal/factory/control/lifecycle_retry.go` — Explicit retry and replay.
- `internal/factory/control/lifecycle_takeover.go` — Human takeover and project start checks.

Current F08 test allocation retains the existing Go package and actual
Coordinator/Store calls: `lifecycle_test.go:1–179` keeps shared fixtures plus
pause/resume assertions; retry assertions at `180–211` move to
`internal/factory/control/lifecycle_retry_test.go`; takeover assertions at
`212–251` move to `internal/factory/control/lifecycle_takeover_test.go`.
Keep fixture definitions once in `lifecycle_test.go`, with current cross-file
consumers; no copied state machine or alternate native test subject.

Evidence: PauseRepository at 33; ResumeRepository at 168; StopProject at 155; RetryRun at 251; replayRetry at 342; TakeoverRun at 357; VerifyProjectStart at 437.

## internal/factory/control/merge.go

Current size at f7: 650 production lines. Keep the same Go package, exported contracts and execution order; move complete concern definitions with their comments and actual imports.

- `internal/factory/control/merge.go` — Pass orchestration, report and persistence helpers.
- `internal/factory/control/merge_reconcile.go` — Existing conditional-operation reconciliation and evidence adoption.
- `internal/factory/control/merge_evidence.go` — Exact snapshot, authority and completion observation.
- `internal/factory/control/merge_withdraw.go` — Repository/acceptance cancellation and withdrawal ordering.

Current defining fit: merge.go retains interface/report/pass/open-row1–150, common error classification506–518 and CAS/finish628–650. merge_reconcile.go owns reconciliation comments151–153 plus complete function154–307, receipt adoption443–462 and observation/outcome conversion600–627. merge_evidence.go owns evidence comments309–315/function316–384, authority386–431, work433–441 and completion comments463–466/function467–504. merge_withdraw.go owns complete cancellation wrappers, bounded cancellation and withdrawal519–599. Blank separators travel with their adjoining units. The coordinator independently inspected the whole current file and actual callers; [F12](../reviews/F12.md) records the primary and native-boundary challenges. Preserve check-before-open, lookup/adoption before fresh write authority, immutable intent persistence before submit, the gate recheck, same-identity cancellation and separate native completion. F12-F1 requires genuine result-reachability proof at the existing native boundary; deleting the current equality alone cannot establish it. No native protocol or merge engine is invented by this split.

## internal/factory/control/merge_native_test.go

Observed size: 1408 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/merge_native_test.go` — Exact native wire, review and confirmation snapshots.
- `internal/factory/control/merge_native_fixture_test.go` — Native config/receipts and committed operation transport.
- `internal/factory/control/merge_native_setup_test.go` — Canonical assignment, approval, assessment and native merge drive.
- `internal/factory/control/merge_native_stale_test.go` — Current authority/head/base/withdrawal refusal cases.
- `internal/factory/control/merge_native_effect_test.go` — Cancel/lost-reply/incomplete-effect and protection cases.
- `internal/factory/control/merge_native_completion_test.go` — Complete merge and dependant readiness journeys.

Evidence: TestNativeMergeWire at 347; TestNativeMergeReviewSnapshot at 372; TestNativeMergeConfirmationSnapshot at 421; loadNativeST12 at 41; nativeMergeSubmitMergeCommitted at 200; nativeMergeAPI at 279; nativeMergeAssignment at 477; nativeMergeApprove at 539; nativeMergeDrive at 581; TestNativeMergeStaleHead at 771; TestNativeMergeStaleBase at 805; TestNativeMergeAuthorityChanged at 857; TestNativeMergeCancelAfterCommit at 988; TestNativeMergeLostReply at 1091; TestNativeMergeProtectionRefuses at 1350; TestNativeMergeFullPass at 666; TestNativeMergeDependantRunnable at 1254.

## internal/factory/control/merge_test.go

Observed size: 571 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/merge_test.go` — Exact merge pass, current check/review/authority cases.
- `internal/factory/control/merge_fixture_test.go` — Existing merger fake and canonical seed.
- `internal/factory/control/merge_effect_test.go` — Lost reply, completion, withdrawal and dependant cases.

Evidence: TestMergePassMergesOneExactPR at 180; TestMergePassRequiresCurrentAuthority at 435; happyMerger at 81; mergeSeed at 117; TestMergePassReconcilesLostSubmitReply at 208; TestMergePassWithdrawalAfterCommitConfirmsCompletion at 492; TestMergeCompletionReleasesCodeDependant at 526.

## internal/factory/control/publication.go

Observed size: 701 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/publication.go` — Pass orchestration, assignment lookup and persistence hooks.
- `internal/factory/control/publication_reconcile.go` — Existing branch/PR effects and receipt reconciliation.
- `internal/factory/control/publication_authority.go` — Current authority and exact native work construction.
- `internal/factory/control/publication_withdraw.go` — Cancellation, withdrawal and late observed-effect adoption.
- `internal/factory/control/publication_export.go` — Confirmed candidate export and bounded bundle decoding.

Evidence: PublishPass at 78; publishOne at 119; assignmentForPublication at 148; reconcilePublication at 163; adoptPublicationReceipts at 448; publicationAuthority at 388; publicationWork at 433; cancelPublications at 507; withdrawPublication at 576; adoptObserved at 620; exportCandidate at 671; decodeExportBundle at 688.

## internal/factory/control/publication_native_test.go

Observed size: 701 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/publication_native_test.go` — Native publish, correction and unexpected-ref scenarios.
- `internal/factory/control/publication_native_fixture_test.go` — Existing native config, Git/API and seed fixture.
- `internal/factory/control/publication_native_effect_test.go` — Lost branch/PR reply, withdrawal and partial committed effects.

Evidence: TestNativePublishCandidate at 365; TestNativePublishCorrection at 430; TestNativePublishWrongRefs at 519; loadNativeST09 at 53; nativeAPI at 90; nativeSeed at 223; TestNativePublishLostReplies at 562; TestNativePublishWithdrawal at 615; TestNativePublishBranchCommittedPRFailed at 683.

## internal/factory/control/publication_test.go

Observed size: 615 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/publication_test.go` — Exact PR link and source/ref refusal cases.
- `internal/factory/control/publication_fixture_test.go` — Existing publication fakes, native outcomes and seed.
- `internal/factory/control/publication_effect_test.go` — Lost reply, immutable replay and partial/fenced effects.
- `internal/factory/control/publication_recovery_test.go` — Late committed links, export recovery and unavailable executor.

Evidence: TestPublishPassLinksOneExactPR at 314; TestPublishPassRejectsChangedAcceptedSource at 413; publicationTestDB at 20; publishSeed at 167; happyPublisher at 262; TestPublishPassReconcilesLostSubmitReply at 333; TestPublishPassReplaysUnobservedSubmitWithByteIdenticalIntent at 356; TestPublishPassFencesIndeterminateEffect at 494; TestPublishPassLostPRReplyRetainsLateCommittedLinkAndCompletion at 541; TestPublishPassRecoversCommittedPRWithoutExport at 577; TestPublishPassWaitsWithoutExecutor at 609.

## internal/factory/control/readiness_test.go

Observed size: 625 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/readiness_test.go` — Native event intake, duplicate observation and explicit adoption.
- `internal/factory/control/readiness_fixture_test.go` — Existing exact native-read fakes and seed inputs.
- `internal/factory/control/readiness_prerequisite_test.go` — Accepted code/result prerequisite outcome cases.
- `internal/factory/control/readiness_visibility_test.go` — Hidden/incomplete/cycle/withdrawn native evidence cases.

Evidence: TestObserveIssueEventCreationQueues at 152; TestObserveIssueEventDuplicateSuppresses at 178; TestEditedCreationNeedsExplicitAdoption at 258; readinessCoordinator at 103; readinessDecision at 95; readinessHint at 111; TestClosureAloneCannotSatisfyCodeOutcome at 296; TestResultPrereqNeedsAcceptedResolution at 382; TestHiddenEndpointBlocksWithoutLeaking at 412; TestLongerCycleBlocks at 471; TestWithdrawnEndpointInvalidatesDependent at 505.

## internal/factory/control/st15_demo_journey_test.go

Observed size: 694 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/st15_demo_journey_test.go` — One composed demo entry and final receipt.
- `internal/factory/control/st15_demo_coding_test.go` — Original coding dispatch, native publication and browser observations.
- `internal/factory/control/st15_demo_review_test.go` — Independent reviewer/fix/new-head/fresh-review sequence.
- `internal/factory/control/st15_demo_completion_test.go` — Native CI, merge/dependant and retention proof.

Evidence: TestST15ComposedDemo at 664; writeFinalReceipt at 651; dispatchA at 99; publishA at 193; runBrowser at 71; prepareReviewer at 238; reviewLeg1 at 326; correctA at 376; reviewLeg2 at 422; ciPass at 458; mergeAndDependants at 474; proveRetention at 590.

## internal/factory/control/st15_demo_native_test.go

Observed size at 0d8d3b8e: 637 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/st15_demo_native_test.go` — Existing fixture identity/configuration, receipts and command helpers.
- `internal/factory/control/st15_demo_stack_test.go` — Explicit task fixture host/broker/runtime stack setup and cleanup.
- `internal/factory/control/st15_demo_broker_test.go` — Current fixture broker build, tmpfs and socket wait.

Evidence at 0d8d3b8e: loadST15 at 64; st15Receipt at 91; st15Podman at 167; setupHostStack at 340; openFactoryStore at 625; st15BuildBroker at 247; st15TmpfsRoot at 295; st15WaitSocket at 326. Broker file logging and kill/wait/file-close cleanup at 579–598 stay with st15_demo_stack_test.go.

## internal/factory/control/st15_demo_seed_test.go

Observed size: 454 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/control/st15_demo_seed_test.go` — Native repository, source and issue/comment setup.
- `internal/factory/control/st15_demo_project_test.go` — Existing retained Project, role preparation and human work setup.
- `internal/factory/control/st15_demo_coordinator_test.go` — Original coordinator construction and native intake hints.

Evidence: seedRepository at 128; createIssue at 386; insertEdge at 430; setupProject at 63; prepareRoles at 169; seedHumanWork at 272; wireCoordinator at 323; postIntake at 361; issueOpenedHint at 436.

## internal/factory/grants.go

Observed size: 438 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/grants.go` — Standing grant identity, capacity and operator records.
- `internal/factory/sponsorship.go` — Provider sponsorship records and allowance validation.
- `internal/factory/effective_authority.go` — Evaluation of separately recorded authority inputs.

Evidence: SettingsCommandType at 63; SettingsDigest at 77; ValidTargetBranch at 129; Sponsorship at 240; EvaluateAuthority at 333.

## internal/factory/merge.go

Observed size: 592 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/merge.go` — Canonical merge stage, identity and aggregate record validation.
- `internal/factory/merge_operation.go` — Immutable conditional operation, work and native intent.
- `internal/factory/merge_evidence.go` — Exact check evidence and native observation validation.

Evidence: Merge at 213; ValidMergeStage at 31; MergeOperationID at 204; MergeIntent at 372; MergeWork at 348; VerifyMergeCheckEvidence at 490; MergeTargetChanged at 587.

## internal/factory/publication.go

Observed size: 736 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/factory/publication.go` — Canonical publication aggregate, identity and outcome.
- `internal/factory/publication_operation.go` — Native effect/cancellation/completion and exact operation record.
- `internal/factory/publication_intent.go` — Immutable branch and PR creation intent binding.
- `internal/factory/publication_refusal.go` — Typed refusal and transport error classification.

Evidence: Publication at 290; ValidPublicationStage at 31; ValidOpEffect at 84; ValidOpCancellation at 105; PublicationOperationID at 270; PublicationWork at 480; PRTitleFor at 623; PRBodyFor at 629; Error at 682.

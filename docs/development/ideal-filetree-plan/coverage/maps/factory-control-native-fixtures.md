# Factory control native fixtures

Current path navigation reconciled at `45ebf4c4` (2026-10-09); historical symbol/body selectors remain pinned to `519b76bd` unless a narrow current selector is explicitly stated.

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-0ab37a6f56c4"></a>

## [internal/factory/control/checks_native_fixture_test.go](../../../../../internal/factory/control/checks_native_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–101, 126–268; file scaffold; loadNativeST11; nativeCheckReceipt; nativeCheckAssessor; nativeCheckDB; nativeCheckPolicy; nativeCheckAdopted; nativeCheckPR; target; nativePostStatusOnce; nativePostStatus; nativeListedStatus; nativeListStatuses; nativeWaitStatusContext; nativeObserveChecks; nativeVerifyChecks; nativeEnsureWorkflow | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 102–125; nativeCheckPublish | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: nativeCheckPublish — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-aeb3679360d4"></a>

## [internal/factory/control/checks_native_stale_test.go](../../../../../internal/factory/control/checks_native_stale_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 114–150; file scaffold; TestNativeCheckWorkflowSeed | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestNativeCheckWorkflowSeed — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–113; TestNativeCheckStaleHead; TestNativeCheckDefinitionsChanged; TestNativeCheckStaleBase; TestNativeCheckWorkflowPendingSnapshot | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: TestNativeCheckStaleHead; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-096ddbe19f25"></a>

## [internal/factory/control/merge_native_completion_test.go](../../../../../internal/factory/control/merge_native_completion_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; file scaffold; TestNativeMergeFullPass; nativeMergeFullPassAttempt; TestNativeMergeDependantRunnable | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 97–173; nativeMergeDependantAttempt | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: nativeMergeDependantAttempt — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c3c98c8a302d"></a>

## [internal/factory/control/merge_native_effect_test.go](../../../../../internal/factory/control/merge_native_effect_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–393; file scaffold; TestNativeMergeCancelBeforeSubmit; TestNativeMergeCancelAfterCommit; dropOnceMerger; SubmitMerge; countMerger; lagOnceMerger; lag; LookupOp; TestNativeMergeLostReply; nativeMergeLostReplyAttempt; TestNativeMergeCommittedIncomplete; nativeMergeCommittedIncompleteAttempt; TestNativeMergeProtectionRefuses; nativeMergeProtectionAttempt | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bf3e274a4e59"></a>

## [internal/factory/control/merge_native_fixture_test.go](../../../../../internal/factory/control/merge_native_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–252; file scaffold; nativeST12Config; loadNativeST12; nativeMergeReceipt; nativeMergeDrain; nativeMergeIdle; nativeMergeStaleRefusal; nativeMergeBusyError; nativeMergeCall; nativeMergeSubmitReviewCommitted; nativeMergeSubmitMergeCommitted; nativeMergeSubmitRawCommitted | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-864e51425fd6"></a>

## [internal/factory/control/merge_native_setup_test.go](../../../../../internal/factory/control/merge_native_setup_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–42, 105–196, 247–291; file scaffold; nativeMergeSetup; nativeMergeApprove; nativeMergeAssess; nativeMergeOpenRow; nativeMergeDrive; nativeMergeInsertEdge; nativeMergeGit; nativeMergeAdvanceBranch | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 43–104, 197–246; nativeMergeAssignment; nativeMergePublishDrive; nativeMergeSeedPublication | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: nativeMergeAssignment; Current declaration duty: nativeMergePublishDrive; Current declaration duty: nativeMergeSeedPublication — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-49510d085a32"></a>

## [internal/factory/control/merge_native_stale_test.go](../../../../../internal/factory/control/merge_native_stale_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–177; file scaffold; TestNativeMergeStaleHead; TestNativeMergeStaleBase; TestNativeMergeWithdrawBeforeSubmit; TestNativeMergeAuthorityChanged; nativeMergeAuthorityAttempt | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-42fc52e73321"></a>

## [internal/factory/control/publication_native_effect_test.go](../../../../../internal/factory/control/publication_native_effect_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 58–97; file scaffold; nativeLostReplies; SubmitPublish; PushBranch; SubmitPRCreate | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–57, 98–215; nativeUnexpectedRefs; TestNativePublishLostReplies; TestNativePublishWithdrawal; TestNativePublishBranchCommittedPRFailed | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: nativeUnexpectedRefs; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8ffeedd75ed6"></a>

## [internal/factory/control/publication_native_fixture_test.go](../../../../../internal/factory/control/publication_native_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–259, 343–357, 364–383; file scaffold; nativeST09Config; loadNativeST09; nativeMust; nativeSecret; nativeRepoURL; nativeAPI; nativeGit; nativeGitOK; nativeCandidate; nativeNewCandidate; nativeAppendCandidate; nativeTip; nativeHost; FactoryLaunch; FactoryStop; FactoryInspect; FactoryHarness; FactoryTakeover; FactoryExport; nativeBroker; GetExecution; CloseExecution; nativeFixture; nativeSeed; wire; nativeReceipt; observe; terminal | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 29 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 260–342, 358–363; assignment; drive; work | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: assignment; Current declaration duty: drive; Current declaration duty: work — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b068616b36c2"></a>

## [internal/factory/control/st15_demo_accept_test.go](../../../../../internal/factory/control/st15_demo_accept_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38, 125–286, 357; whole file; st15Fixture.readEvidence; st15Fixture.admitDecision; st15Fixture.admitPLeg; st15Fixture.admitALeg; st15Fixture.admitBCLegs; _ | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 7 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 39–124, 287–322; st15Fixture.observeControl, blockerCodes, requireBlocker, st15Fixture.seedIssues, st15Fixture.proveClosureNotOutcome | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Readiness and prerequisite semantics in composed demo; declarations/fields: `st15Fixture.observeControl`, `blockerCodes`, `requireBlocker`, `st15Fixture.seedIssues`, `st15Fixture.proveClosureNotOutcome` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 323–356; st15Fixture.proveGrantWithdrawal, st15Fixture.activateAuthority | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Dispatch authority and grant-withdrawal composed demo assertions; declarations/fields: `st15Fixture.proveGrantWithdrawal`, `st15Fixture.activateAuthority` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-4074e0c7742b"></a>
<a id="internalfactorycontrolst15_demo_journey_testgo-1"></a>

## [internal/factory/control/st15_demo_journey_test.go](../../../../../internal/factory/control/st15_demo_journey_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; file scaffold | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 8–51; writeFinalReceipt; TestST15ComposedDemo | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current declaration duty: writeFinalReceipt; Current declaration duty: TestST15ComposedDemo — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-10c77ae3f1f1"></a>
<a id="internalfactorycontrolst15_demo_native_testgo-1"></a>

## [internal/factory/control/st15_demo_native_test.go](../../../../../internal/factory/control/st15_demo_native_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30; file scaffold | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–215; st15Image; st15Config; loadST15; st15ReceiptDir; st15Receipt; st15Fixture; st15Check; check; st15Podman; st15Pexec; st15Git; st15SHA256; st15RandHex | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current declaration duty: st15Image; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-00cb8f9b9b02"></a>

## [internal/factory/control/st15_demo_runs_test.go](../../../../../internal/factory/control/st15_demo_runs_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 67–108; st15Fixture.stopSettled | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Retires a recorded production run through the coordinator stop path; uncertain receipts retry within the bounded cleanup window. Direct synthetic stop/settlement helpers were removed. |
| 1–66, 109–126; st15Fixture.publishChildRun; st15Fixture.settleDispatched | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Discovers and binds reviewer/correction children through PublishPass and settles the actual recorded assignment/run. The former direct launch helper was removed. |
| 127–140; st15Fixture.submitReview | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | retained | Submits the recorded reviewer result through the existing review path; the separate prompt-construction helper was removed. |
| 141–153; st15Fixture.seedCIStatus | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Explicit native fixture status publication. |

<a id="coverage-d50be1d675ed"></a>
<a id="internalfactorycontrolst15_demo_seed_testgo-1"></a>

## [internal/factory/control/st15_demo_seed_test.go](../../../../../internal/factory/control/st15_demo_seed_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; file scaffold | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–26; st09; api | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current declaration duty: st09; Current declaration duty: api — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 27–84; git; repoURL; seedRepository | [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) | retained | Current declaration duty: git; Current declaration duty: repoURL; Current declaration duty: seedRepository — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 85–113, 134–137; st15Issue; createIssue; commentIssue; insertEdge | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Current declaration duty: st15Issue; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 114–133; readIssue; readComment | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Current declaration duty: readIssue; Current declaration duty: readComment — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9da153c94ff8"></a>

## [internal/factory/control/staged_seed_test.go](../../../../../internal/factory/control/staged_seed_test.go)

Selective physical-owner transfer at `c84d242c`; unchanged SQL and fixture
semantics move out of the production Store package into existing `control_test`.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; file scaffold | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Test-only database/sql driver and fixture support; acceptance SQLite remains independent |
| 17–61; seedStagedDependencyEdge | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Same bound SQL/schema/optional-column fixture write used by the existing ST12/ST15 callers. Both callers compile; no native flow execution claimed |

<a id="r02-current-path-internal-factory-control-merge-native-deadline-test-go"></a>

### [internal/factory/control/merge_native_deadline_test.go](../../../../../internal/factory/control/merge_native_deadline_test.go)

F12: `fixedDeadlineContext` lines 21–26 and `TestNativeMergeCallContext*` lines 28–165, covering deadline propagation, late result rejection and elapsed-deadline behavior. F09: `TestNativeAPIContextCancelsStalledHTTPRequest` lines 167–215, testing the current `nativeAPIContext` caller defined in `publication_native_fixture_test.go`. Imports at lines 1–19 are shared test scaffold without a separately assigned behavior. This is a mixed-file split, not a single-owner assignment.

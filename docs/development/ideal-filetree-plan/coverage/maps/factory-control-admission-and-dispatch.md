# Factory control admission and dispatch

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-bd92c58c05f9"></a>

<a id="internalfactorycontrolacceptancego-1"></a>

## [internal/factory/control/acceptance.go](../../../../../internal/factory/control/acceptance.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 16–37 | Declared identifiers/bounds (declaration group) for Accepted requirements and invalidation; declarations/fields: `(declaration group)` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 38–41 | Record, DTO or interface contract AcceptanceRefusal for Accepted requirements and invalidation; declarations/fields: `AcceptanceRefusal` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 42–43 | AcceptanceRefusal.Error — Accepted requirements and invalidation; declarations/fields: `AcceptanceRefusal.Error` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 44–48 | refuseAcceptance — Accepted requirements and invalidation; declarations/fields: `refuseAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 49–64 | Record, DTO or interface contract AcceptanceIssueView for Accepted requirements and invalidation; declarations/fields: `AcceptanceIssueView` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 65–72 | Record, DTO or interface contract AcceptanceComment for Accepted requirements and invalidation; declarations/fields: `AcceptanceComment` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 73–81 | Record, DTO or interface contract AcceptanceEdge for Accepted requirements and invalidation; declarations/fields: `AcceptanceEdge` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 82–92 | Record, DTO or interface contract AcceptanceEvidence for Accepted requirements and invalidation; declarations/fields: `AcceptanceEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 93–98 | Record, DTO or interface contract AcceptanceSource for Accepted requirements and invalidation; declarations/fields: `AcceptanceSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 99–108 | Record, DTO or interface contract AcceptanceReceipt for Accepted requirements and invalidation; declarations/fields: `AcceptanceReceipt` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 109–124 | Record, DTO or interface contract WithdrawalReceipt for Accepted requirements and invalidation; declarations/fields: `WithdrawalReceipt` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 125–190 | Coordinator.AdmitAcceptance — Accepted requirements and invalidation; declarations/fields: `Coordinator.AdmitAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 191–210 | replayAcceptance — Accepted requirements and invalidation; declarations/fields: `replayAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 211–217 | mustIssueIndex — Accepted requirements and invalidation; declarations/fields: `mustIssueIndex` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 218–245 | Coordinator.readAcceptanceEvidence — Accepted requirements and invalidation; declarations/fields: `Coordinator.readAcceptanceEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 246–321 | Coordinator.verifyAcceptance — Accepted requirements and invalidation; declarations/fields: `Coordinator.verifyAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 322–390 | Coordinator.AdmitInitialAcceptance — Accepted requirements and invalidation; declarations/fields: `Coordinator.AdmitInitialAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 391–404 | Record, DTO or interface contract AcceptanceValidity for Accepted requirements and invalidation; declarations/fields: `AcceptanceValidity` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 405–409 | Coordinator.AcceptanceStatus — Accepted requirements and invalidation; declarations/fields: `Coordinator.AcceptanceStatus` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 410–469 | Coordinator.acceptanceStatus — Accepted requirements and invalidation; declarations/fields: `Coordinator.acceptanceStatus` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 470–524 | assessAcceptance — Accepted requirements and invalidation; declarations/fields: `assessAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 525–565 | Coordinator.WithdrawAcceptance — Accepted requirements and invalidation; declarations/fields: `Coordinator.WithdrawAcceptance` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 566–584 | replayWithdrawal — Accepted requirements and invalidation; declarations/fields: `replayWithdrawal` |

<a id="coverage-9c1eda10c1a7"></a>

<a id="internalfactorycontrolacceptance_testgo-1"></a>

## [internal/factory/control/acceptance_test.go](../../../../../internal/factory/control/acceptance_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 14–19 | Fixture/protocol support stubAcceptanceSource; declarations/fields: `stubAcceptanceSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 20–24 | Fixture/protocol support stubAcceptanceSource.ReadAcceptanceEvidence; declarations/fields: `stubAcceptanceSource.ReadAcceptanceEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 25–28 | Fixture/protocol support acceptanceDigests; declarations/fields: `acceptanceDigests` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 29–39 | Fixture/protocol support acceptanceDecision; declarations/fields: `acceptanceDecision` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 40–53 | Fixture/protocol support acceptanceEvidence; declarations/fields: `acceptanceEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 54–61 | Fixture/protocol support acceptanceHarness; declarations/fields: `acceptanceHarness` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 62–70 | Fixture/protocol support acceptanceRefusal: expected refusal, got %v; declarations/fields: `acceptanceRefusal` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 71–116 | Assertions TestAdmitAcceptanceVerifiesAndRecords: receipt: %+v; declarations/fields: `TestAdmitAcceptanceVerifiesAndRecords` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 117–179 | Assertions TestAdmitAcceptanceRefusesWithoutRecording: reason %q; declarations/fields: `TestAdmitAcceptanceRefusesWithoutRecording` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 180–221 | Assertions TestAdmitAcceptanceGuardsCodeRoutes: reason %q; declarations/fields: `TestAdmitAcceptanceGuardsCodeRoutes` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 222–230 | Fixture/protocol support creationEvidence; declarations/fields: `creationEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 231–239 | Fixture/protocol support creationHarness; declarations/fields: `creationHarness` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 240–259 | Assertions TestAdmitInitialAcceptanceVerifiesCreation: decision: %+v; declarations/fields: `TestAdmitInitialAcceptanceVerifiesCreation` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 260–311 | Assertions TestAdmitInitialAcceptanceRefusesIneligible: reason %q; declarations/fields: `TestAdmitInitialAcceptanceRefusesIneligible` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 312–328 | Assertions TestAdmitInitialAcceptanceCannotOverwrite: overwrite: %v; declarations/fields: `TestAdmitInitialAcceptanceCannotOverwrite` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 329–440 | Assertions TestAcceptanceStatusAssessesValidity: status: %+v %v; declarations/fields: `TestAcceptanceStatusAssessesValidity` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 441–472 | Assertions TestWithdrawAcceptanceLatchesAndReplays: replay: %+v %v; declarations/fields: `TestWithdrawAcceptanceLatchesAndReplays` |

<a id="coverage-ad27b0983b18"></a>

<a id="internalfactorycontrolchecks_native_testgo-1"></a>

## [internal/factory/control/checks_native_test.go](../../../../../internal/factory/control/checks_native_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 1–35 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 36–52 | Fixture/protocol support loadNativeST11: native fixture requires explicit repository and assessor inputs; declarations/fields: `loadNativeST11` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 53–67 | Fixture/protocol support nativeCheckReceipt: ST11_RECEIPT_DIR is required to retain native proof; declarations/fields: `nativeCheckReceipt` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 68–72 | Fixture/protocol support nativeCheckAssessor; declarations/fields: `nativeCheckAssessor` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 73–78 | Fixture/protocol support nativeCheckDB; declarations/fields: `nativeCheckDB` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 79–101 | Fixture/protocol support nativeCheckPolicy; declarations/fields: `nativeCheckPolicy` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 102–105 | Fixture/protocol support nativeCheckAdopted; declarations/fields: `nativeCheckAdopted` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 106–114 | Fixture/protocol support nativeCheckPR; declarations/fields: `nativeCheckPR` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 115–138 | Native candidate publication preparation/effect driver; declarations/fields: `nativeCheckPublish` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 139–145 | Fixture/protocol support nativeCheckPR.target; declarations/fields: `nativeCheckPR.target` |
| [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) / active | 146–197, 203–241, 263–282 | Native CI observation or explicit fixture check-status/workflow seed; declarations/fields: `nativePostStatusOnce`, `nativePostStatus`, `nativeListStatuses`, `nativeWaitStatusContext`, `nativeObserveChecks`, `nativeEnsureWorkflow` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 198–202 | Native authoritative read fixture or observation projection; declarations/fields: `nativeListedStatus` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 242–262 | Fixture/protocol support nativeVerifyChecks: stored check assessment differs from its verdict; declarations/fields: `nativeVerifyChecks` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 283–565 | Factory candidate check verdict exercised by opt-in native fixture; declarations/fields: `TestNativeCheckPass`, `TestNativeCheckPending`, `TestNativeCheckMissing`, `TestNativeCheckFailed`, `TestNativeCheckCancelled`, `TestNativeCheckSkipped`, `TestNativeCheckUnknownState`, `TestNativeCheckLatestWins`, `TestNativeCheckStaleHead`, `TestNativeCheckDefinitionsChanged`, `TestNativeCheckPolicyEmpty`, `TestNativeCheckStaleBase`, `TestNativeCheckWorkflowPendingSnapshot`, `TestNativeCheckWorkflowSeed` |

<a id="coverage-2d089605b314"></a>

## [internal/factory/control/coordinator.go](../../../../../internal/factory/control/coordinator.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–22 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 23–36 | Declared identifiers/bounds (declaration group) for Run lifecycle and intervention; declarations/fields: `(declaration group)` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 37, 39–41, 44–48 | Record, DTO or interface contract HostFactory for Run lifecycle and intervention; declarations/fields: `HostFactory` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 38 | Native assignment launch contract; declarations/fields: `HostFactory.FactoryLaunch` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 42 | Native candidate export contract; declarations/fields: `HostFactory.FactoryExport` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 43 | Pinned factory role harness observation contract; declarations/fields: `HostFactory.FactoryHarness` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 49, 52–66 | Record, DTO or interface contract BrokerExecution for Run lifecycle and intervention; declarations/fields: `BrokerExecution` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 50 | Broker execution fence metadata read contract; coordinator never acquires credentials; declarations/fields: `BrokerExecution.GetExecution` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 51 | Broker terminal execution close contract; declarations/fields: `BrokerExecution.CloseExecution` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 67, 69–70, 80–82 | Record, DTO or interface contract Coordinator for Run lifecycle and intervention; declarations/fields: `Coordinator` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 68 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Store` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 71 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.AcceptanceReads` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 72, 78 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Readiness`, `Coordinator.traversal` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 73, 79 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.DispatchReads`, `Coordinator.queue` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 74 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Publication` |
| [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) / active | 75 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Reviews` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 76 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Checks` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 77 | Coordinator dependency/read/effect contract for independent slice; declarations/fields: `Coordinator.Merges` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 83–89 | NewCoordinator — Run lifecycle and intervention; declarations/fields: `NewCoordinator` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 90–134 | Coordinator.Start — Run lifecycle and intervention; declarations/fields: `Coordinator.Start` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 135–146 | Coordinator.Close — Run lifecycle and intervention; declarations/fields: `Coordinator.Close` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 147–169 | Coordinator.Status — Run lifecycle and intervention; declarations/fields: `Coordinator.Status` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 170–178 | redactRun — Run lifecycle and intervention; declarations/fields: `redactRun` |

<a id="coverage-6e200bbe2221"></a>

<a id="internalfactorycontroldispatchgo-1"></a>

## [internal/factory/control/dispatch.go](../../../../../internal/factory/control/dispatch.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 19–27 | Record, DTO or interface contract DispatchHost for Assignment and dispatch; declarations/fields: `DispatchHost` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 28–34 | Record, DTO or interface contract DispatchBroker for Assignment and dispatch; declarations/fields: `DispatchBroker` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 35–45 | Record, DTO or interface contract DispatchComment for Assignment and dispatch; declarations/fields: `DispatchComment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 46–62 | Record, DTO or interface contract DispatchIssue for Assignment and dispatch; declarations/fields: `DispatchIssue` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 63–75 | Record, DTO or interface contract DispatchInputs for Assignment and dispatch; declarations/fields: `DispatchInputs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 76–83 | Record, DTO or interface contract DispatchReads for Assignment and dispatch; declarations/fields: `DispatchReads` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 84–118, 158–175 | Declared identifiers/bounds (declaration group) for Assignment and dispatch; declarations/fields: `(declaration group)` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 119–126 | Record, DTO or interface contract DispatchWait for Assignment and dispatch; declarations/fields: `DispatchWait` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 127–135 | Record, DTO or interface contract DispatchLaunch for Assignment and dispatch; declarations/fields: `DispatchLaunch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 136–146 | Record, DTO or interface contract DispatchError for Assignment and dispatch; declarations/fields: `DispatchError` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 147–153 | Record, DTO or interface contract DispatchReport for Assignment and dispatch; declarations/fields: `DispatchReport` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 154–157 | newDispatchReport — Assignment and dispatch; declarations/fields: `newDispatchReport` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 176–187 | Record, DTO or interface contract DispatchDeps for Assignment and dispatch; declarations/fields: `DispatchDeps` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 188–196 | dispatchReady — Assignment and dispatch; declarations/fields: `dispatchReady` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 197–211, 306–408, 438–530, 593–602 | Assignment recovery consumes separate native/broker lifecycle facts; declarations/fields: `RecoverDispatch`, `recoverAssigned`, `recoverOne`, `dispatchHistoryRuns`, `activeHistoryRun`, `confirmUnused`, `recoverSettled`, `recoverMissingRun`, `recoverUnused`, `recoverOutput` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 212–255 | DispatchPass — Assignment and dispatch; declarations/fields: `DispatchPass` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 256–305, 623–637 | Shared current-pass occupancy and confirmed usage accounting; declarations/fields: `passOccupancy`, `snapshotOccupancy`, `passOccupancy.hold`, `passOccupancy.heldTotal`, `passOccupancy.heldRepo`, `passOccupancy.heldConn`, `recordConfirmedUsage` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 409–410 | Declared identifiers/bounds hostRunNotFound for Assignment and dispatch; declarations/fields: `hostRunNotFound` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 411–416 | isHostNotFound — Assignment and dispatch; declarations/fields: `isHostNotFound` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 417–437 | Recover reservation accounting from durable assignment/native facts; declarations/fields: `ensureHeld` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 531–558 | retryOrFinish — Assignment and dispatch; declarations/fields: `retryOrFinish` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 559–592 | finishTerminal — Assignment and dispatch; declarations/fields: `finishTerminal` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 603–622 | finishFromRun — Assignment and dispatch; declarations/fields: `finishFromRun` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 638–654 | deriveAttemptResult — Assignment and dispatch; declarations/fields: `deriveAttemptResult` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 655–671 | boundSummary — Assignment and dispatch; declarations/fields: `boundSummary` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 672–702 | AccountSettledRun — Assignment and dispatch; declarations/fields: `AccountSettledRun` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 703–713 | Coordinator.dispatchDeps — Assignment and dispatch; declarations/fields: `Coordinator.dispatchDeps` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 714–722 | Coordinator.Dispatch — Assignment and dispatch; declarations/fields: `Coordinator.Dispatch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 723–732 | Coordinator.dispatchAfterIntake — Assignment and dispatch; declarations/fields: `Coordinator.dispatchAfterIntake` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 733–742 | Readiness reassessment after dispatch completion; declarations/fields: `Coordinator.assessDispatchDependants` |

<a id="coverage-81f3f2c15426"></a>

<a id="internalfactorycontroldispatch_attemptgo-1"></a>

## [internal/factory/control/dispatch_attempt.go](../../../../../internal/factory/control/dispatch_attempt.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 19–39 | Record, DTO or interface contract attemptPlan for Assignment and dispatch; declarations/fields: `attemptPlan` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 40–77 | dispatchOne — Assignment and dispatch; declarations/fields: `dispatchOne` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 78–82 | Record, DTO or interface contract planWait for Assignment and dispatch; declarations/fields: `planWait` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 83–87 | waitFor — Assignment and dispatch; declarations/fields: `waitFor` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 88–108 | admissionWait — Assignment and dispatch; declarations/fields: `admissionWait` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 109–120, 228–276 | Admission capacity/budget accounting; declarations/fields: `refreshOccupancy`, `checkLimits` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 121–199 | planAttempt — Assignment and dispatch; declarations/fields: `planAttempt` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 200–227 | Select the authority-bound active sponsorship; no connection fallback; declarations/fields: `selectSponsorship` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 277–312 | Assignment selects existing accepted/approved preparation; does not own preparation records; declarations/fields: `selectPreparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 313–331 | checkHarness — Assignment and dispatch; declarations/fields: `checkHarness` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 332–402 | readAttemptInputs — Assignment and dispatch; declarations/fields: `readAttemptInputs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 403–423 | promptSections — Assignment and dispatch; declarations/fields: `promptSections` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 424–499 | executeFreshAttempt — Assignment and dispatch; declarations/fields: `executeFreshAttempt` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 500–564 | retryAttempt — Assignment and dispatch; declarations/fields: `retryAttempt` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 565–590 | launchTail — Assignment and dispatch; declarations/fields: `launchTail` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 591–602 | ensureHeldLaunch — Assignment and dispatch; declarations/fields: `ensureHeldLaunch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 603–620 | Fence or settle unused native launch; declarations/fields: `settleUnusedLaunch` |

<a id="coverage-dc71af56a96e"></a>

<a id="internalfactorycontroldispatch_testgo-1"></a>

## [internal/factory/control/dispatch_test.go](../../../../../internal/factory/control/dispatch_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 21–25 | Fixture/protocol support dispatchDigest; declarations/fields: `dispatchDigest` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 26–32 | Fixture/protocol support fakeDispatchHost; declarations/fields: `fakeDispatchHost` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 33–37 | Fixture/protocol support fakeDispatchHost.FactoryLaunch; declarations/fields: `fakeDispatchHost.FactoryLaunch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 38–44 | Fixture/protocol support fakeDispatchHost.FactoryInspect; declarations/fields: `fakeDispatchHost.FactoryInspect` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 45–48 | Fixture/protocol support fakeDispatchHost.FactoryHarness; declarations/fields: `fakeDispatchHost.FactoryHarness` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 49–52 | Fixture/protocol support fakeDispatchHost.FactoryStop; declarations/fields: `fakeDispatchHost.FactoryStop` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 53–56 | Fixture/protocol support fakeDispatchHost.FactoryTakeover; declarations/fields: `fakeDispatchHost.FactoryTakeover` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 57–60 | Fixture/protocol support fakeDispatchHost.FactoryExport; declarations/fields: `fakeDispatchHost.FactoryExport` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 61–64 | Fixture/protocol support fakeDispatchBroker; declarations/fields: `fakeDispatchBroker` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 65–71 | Fixture/protocol support fakeDispatchBroker.GetExecution; declarations/fields: `fakeDispatchBroker.GetExecution` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 72–75 | Fixture/protocol support fakeDispatchBroker.CloseExecution; declarations/fields: `fakeDispatchBroker.CloseExecution` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 76–81 | Fixture/protocol support fakeDispatchReads; declarations/fields: `fakeDispatchReads` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 82–93 | Fixture/protocol support fakeDispatchReads.ReadDispatchInputs; declarations/fields: `fakeDispatchReads.ReadDispatchInputs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 94–106 | Fixture/protocol support dispatchFixture; declarations/fields: `dispatchFixture` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 107–112 | Fixture/protocol support dispatchTestDB; declarations/fields: `dispatchTestDB` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 113–197 | Fixture/protocol support dispatchSeed; declarations/fields: `dispatchSeed` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 198–220 | Fixture/protocol support dispatchFixture.accept; declarations/fields: `dispatchFixture.accept` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 221–225 | Fixture/protocol support dispatchFixture.queue; declarations/fields: `dispatchFixture.queue` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 226–237 | Fixture/protocol support dispatchFixture.queueAt; declarations/fields: `dispatchFixture.queueAt` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 238–246 | Fixture/protocol support dispatchFixture.deps; declarations/fields: `dispatchFixture.deps` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 247–337 | Assertions TestDispatchPassLaunchesOldestWithinShortLimit: launched = %+v; declarations/fields: `TestDispatchPassLaunchesOldestWithinShortLimit` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 338–346 | Fixture/protocol support waitReason; declarations/fields: `waitReason` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 347–432, 1179–1236 | Dispatch capacity and charged usage assertions; declarations/fields: `TestDispatchAccountingNeverRefreshes`, `TestConcurrentDispatchPassesPreserveCapacityAndBudget` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 433–573, 929–964 | Dispatch recovery/lifecycle uncertainty assertions; declarations/fields: `TestDispatchInterruptionReleasesOnlyUnused`, `TestDispatchApprovedReceiptReleasesWhenBrokerUnknown`, `TestDispatchApprovedReceiptFencesWhenBrokerHolds`, `TestDispatchFencedLaunchStaysHeld`, `TestRecoveryConsumesSettledRunWithoutHook` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 574–786 | Assertions TestDispatchWaitReasons: launched = %+v; declarations/fields: `TestDispatchWaitReasons` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 787–837 | Assertions TestDispatchNeverFallsBackToAnotherConnection: first pass = %+v %+v; declarations/fields: `TestDispatchNeverFallsBackToAnotherConnection` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 838–928 | Assertions TestAccountSettledRunResults: launched = %+v; declarations/fields: `TestAccountSettledRunResults` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 965–969 | Fixture/protocol support fakeAcceptanceSource; declarations/fields: `fakeAcceptanceSource` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 970–978 | Fixture/protocol support fakeAcceptanceSource.ReadAcceptanceEvidence; declarations/fields: `fakeAcceptanceSource.ReadAcceptanceEvidence` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 979–1047, 1067–1106 | Intake/readiness trigger and completion reassessment assertions; declarations/fields: `TestCompletionTriggersDependantReassessment`, `TestIntakeTriggersAutomaticDispatch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1048–1059 | Assertions TestHostNotFoundMatcherPinsHostSentinel: host sentinel not recognized; declarations/fields: `TestHostNotFoundMatcherPinsHostSentinel` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1060–1066 | Assertions TestDispatchPassWithoutDepsReports: report = %+v; declarations/fields: `TestDispatchPassWithoutDepsReports` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1107–1132 | Assertions TestDispatchWithdrawnGateRecordsNothing: report = %+v %+v %+v; declarations/fields: `TestDispatchWithdrawnGateRecordsNothing` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1133–1139 | Fixture/protocol support rendezvousReads; declarations/fields: `rendezvousReads` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1140–1157 | Fixture/protocol support rendezvousReads.ReadDispatchInputs; declarations/fields: `rendezvousReads.ReadDispatchInputs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1158–1163 | Fixture/protocol support mutexDispatchHost; declarations/fields: `mutexDispatchHost` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1164–1170 | Fixture/protocol support mutexDispatchHost.FactoryLaunch; declarations/fields: `mutexDispatchHost.FactoryLaunch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1171–1174 | Fixture/protocol support mutexDispatchHost.FactoryInspect; declarations/fields: `mutexDispatchHost.FactoryInspect` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1175–1178 | Fixture/protocol support mutexDispatchHost.FactoryHarness; declarations/fields: `mutexDispatchHost.FactoryHarness` |

<a id="coverage-eeebcb0b9b39"></a>

<a id="internalfactorycontrolgrantsgo-1"></a>

## [internal/factory/control/grants.go](../../../../../internal/factory/control/grants.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 17–21, 339–391, 423–444 | Composite dispatch authority evaluation/registration; declarations/fields: `ErrIneffectiveAuthority`, `Coordinator.EffectiveAuthority`, `Coordinator.RegisterDispatch` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 22–48, 123–202, 246–325 | Shared immutable factory command receipt/replay mechanics; supplied grant/decision callbacks retain their policy owners; declarations/fields: `GrantReceipt`, `DecisionReceipt`, `grantTarget`, `Coordinator.applyGrant`, `replayGrant`, `Coordinator.admitDecision`, `replayDecision` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 49–64 | Coordinator.ApplyPolicy — Repository factory policy; declarations/fields: `Coordinator.ApplyPolicy` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 65–75 | Operator-admitted capacity mutation; declarations/fields: `Coordinator.ApplyCapacity` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 76–86 | Operator repository execution grant mutation; declarations/fields: `Coordinator.ApplyOperatorGrant` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 87–110 | Connection-owner sponsorship mutation; declarations/fields: `Coordinator.ApplySponsorship` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 111–122 | Repository-owner standing environment creation grant; declarations/fields: `Coordinator.ApplyEnvironmentGrant` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 203–220 | Exact preparation requirement acceptance command; declarations/fields: `Coordinator.AdmitRequirement` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 221–245 | Privileged effects approval command; declarations/fields: `Coordinator.AdmitApproval` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 326–338 | Current preparation admission hold observation; declarations/fields: `Coordinator.holdActive` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 392–422 | Existing preparation readiness projection; declarations/fields: `Coordinator.preparationReadiness` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 445–502 | Explicitly reopening a withdrawn dispatch gate; declarations/fields: `Coordinator.ReopenDispatch` |

<a id="coverage-86d92eb75373"></a>

## [internal/factory/control/grants_test.go](../../../../../internal/factory/control/grants_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 14–31 | Fixture/protocol support grantPolicy; declarations/fields: `grantPolicy` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 32–38 | Fixture/protocol support grantProfile; declarations/fields: `grantProfile` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 39–49 | Fixture/protocol support grantProject; declarations/fields: `grantProject` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 50–76 | Fixture/protocol support grantFullAuthority; declarations/fields: `grantFullAuthority` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 77–119 | Assertions TestApplyPolicyReplaysAndClosesDispatch: authority: %+v %v; declarations/fields: `TestApplyPolicyReplaysAndClosesDispatch` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 120–146 | Sponsorship withdrawal assertions; declarations/fields: `TestSponsorshipWithdrawalKeepsDispatchWithActiveSibling` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 147–163 | Effective admission before dispatch reopens assertions; declarations/fields: `TestReopenRequiresEffectiveGrants` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 164–177 | Preparation decision chain assertions partitioned below; declarations/fields: `TestRequirementAndApprovalDecisions` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 178–191 | Approval must reference current exact requirement inside mixed decision test; declarations/fields: `TestRequirementAndApprovalDecisions` |

<a id="coverage-f7a105fa9718"></a>

<a id="internalfactorycontrollifecyclego-1"></a>

## [internal/factory/control/lifecycle.go](../../../../../internal/factory/control/lifecycle.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 18–21 | Record, DTO or interface contract ErrPendingRuns for Run lifecycle and intervention; declarations/fields: `ErrPendingRuns` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 22–26 | ErrPendingRuns.Error — Run lifecycle and intervention; declarations/fields: `ErrPendingRuns.Error` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 27–32 | Declared identifiers/bounds ErrTakeoverFailed for Run lifecycle and intervention; declarations/fields: `ErrTakeoverFailed` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 33–72 | Coordinator.PauseRepository — Run lifecycle and intervention; declarations/fields: `Coordinator.PauseRepository` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 73–86 | replayPause — Run lifecycle and intervention; declarations/fields: `replayPause` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 87–123 | Coordinator.withdrawAndStopRuns — Run lifecycle and intervention; declarations/fields: `Coordinator.withdrawAndStopRuns` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 124–154 | Coordinator.projectRuns — Run lifecycle and intervention; declarations/fields: `Coordinator.projectRuns` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 155–167, 437–487 | Project Start/Stop coordination over existing factory run state; declarations/fields: `Coordinator.StopProject`, `Coordinator.VerifyProjectStart`, `Coordinator.checkStartRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 168–235 | Coordinator.ResumeRepository — Run lifecycle and intervention; declarations/fields: `Coordinator.ResumeRepository` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 236–250 | replayResume — Run lifecycle and intervention; declarations/fields: `replayResume` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 251–312 | Coordinator.RetryRun — Run lifecycle and intervention; declarations/fields: `Coordinator.RetryRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 313–320 | onlyDispatchClosed — Run lifecycle and intervention; declarations/fields: `onlyDispatchClosed` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 321–341 | Coordinator.retryQueueReason — Run lifecycle and intervention; declarations/fields: `Coordinator.retryQueueReason` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 342–356 | replayRetry — Run lifecycle and intervention; declarations/fields: `replayRetry` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 357–419 | Coordinator.TakeoverRun — Run lifecycle and intervention; declarations/fields: `Coordinator.TakeoverRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 420–436 | replayTakeover — Run lifecycle and intervention; declarations/fields: `replayTakeover` |

<a id="coverage-5b8742504bab"></a>

## [internal/factory/control/lifecycle_test.go](../../../../../internal/factory/control/lifecycle_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 15–16 | Fixture/protocol support lifecycleProject; declarations/fields: `lifecycleProject` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 17–27 | Fixture/protocol support lifecycleProjectFixture; declarations/fields: `lifecycleProjectFixture` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 28–40 | Fixture/protocol support settleStubs; declarations/fields: `settleStubs` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 41–61 | Fixture/protocol support readyPreparations; declarations/fields: `readyPreparations` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 62–108 | Assertions TestPauseWithdrawsBeforeStopping: run stopped before dispatch closed; declarations/fields: `TestPauseWithdrawsBeforeStopping` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 109–131 | Assertions TestPauseKeepsUncertainRunsFenced: uncertain run reported settled: %+v; declarations/fields: `TestPauseKeepsUncertainRunsFenced` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 132–169 | Assertions TestResumeReopensOnlyAfterSettledRuns: settled resume refused; declarations/fields: `TestResumeReopensOnlyAfterSettledRuns` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 170–179 | Assertions TestResumeRefusesWithdrawnGrants: resume without grants: %v; declarations/fields: `TestResumeRefusesWithdrawnGrants` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 180–211 | Assertions TestRetryStaysQueuedAfterReconciledRun: retry admitted for an unsettled run; declarations/fields: `TestRetryStaysQueuedAfterReconciledRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 212–251 | Assertions TestTakeoverCopiesOnlyReconciledRuns: takeover finished for an unsettled run; declarations/fields: `TestTakeoverCopiesOnlyReconciledRuns` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 252–321 | Project lifecycle/factory coordination assertions; declarations/fields: `TestStopProjectScopesWithdrawalAndRuns`, `TestVerifyProjectStartClassifiesRuns` |

<a id="coverage-8d8ccde235b9"></a>

<a id="internalfactorycontrolreadiness_testgo-1"></a>

## [internal/factory/control/readiness_test.go](../../../../../internal/factory/control/readiness_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 14–19 | Fixture/protocol support (declaration group); declarations/fields: `(declaration group)` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 20–25 | Fixture/protocol support fakeEvidenceSource; declarations/fields: `fakeEvidenceSource` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 26–41 | Fixture/protocol support fakeEvidenceSource.ReadAcceptanceEvidence; declarations/fields: `fakeEvidenceSource.ReadAcceptanceEvidence` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 42–49 | Fixture/protocol support fakeEvidenceSource.totalCalls; declarations/fields: `fakeEvidenceSource.totalCalls` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 50–57 | Fixture/protocol support fakeObserver; declarations/fields: `fakeObserver` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 58–62 | Fixture/protocol support fakeIssuePage; declarations/fields: `fakeIssuePage` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 63–69 | Fixture/protocol support fakeObserver.ObserveNativeRevision; declarations/fields: `fakeObserver.ObserveNativeRevision` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 70–83 | Fixture/protocol support fakeObserver.ListRepositoryIssues; declarations/fields: `fakeObserver.ListRepositoryIssues` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 84–90 | Fixture/protocol support readinessView; declarations/fields: `readinessView` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 91–94 | Fixture/protocol support readinessEvidence; declarations/fields: `readinessEvidence` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 95–102 | Fixture/protocol support readinessDecision; declarations/fields: `readinessDecision` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 103–110 | Fixture/protocol support readinessCoordinator; declarations/fields: `readinessCoordinator` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 111–114 | Fixture/protocol support readinessHint; declarations/fields: `readinessHint` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 115–121 | Fixture/protocol support mustAdmit; declarations/fields: `mustAdmit` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 122–130 | Fixture/protocol support findBlocker; declarations/fields: `findBlocker` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 131–151 | Assertions TestIntakeHintValidate: valid hint refused:; declarations/fields: `TestIntakeHintValidate` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 152–177 | Assertions TestObserveIssueEventCreationQueues: creation intake failed:; declarations/fields: `TestObserveIssueEventCreationQueues` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 178–198 | Assertions TestObserveIssueEventDuplicateSuppresses: first intake failed:; declarations/fields: `TestObserveIssueEventDuplicateSuppresses` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 199–233 | Assertions TestObserveIssueEventUnchangedBlockerSuppresses: blocked intake failed:; declarations/fields: `TestObserveIssueEventUnchangedBlockerSuppresses` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 234–257 | Assertions TestCreationWithoutAuthorityWaitsForAdoption: outsider creation intake failed:; declarations/fields: `TestCreationWithoutAuthorityWaitsForAdoption` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 258–279 | Assertions TestEditedCreationNeedsExplicitAdoption: edited creation not awaiting adoption:; declarations/fields: `TestEditedCreationNeedsExplicitAdoption` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 280–295 | Fixture/protocol support setupCodePrereq; declarations/fields: `setupCodePrereq` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 296–319 | Assertions TestClosureAloneCannotSatisfyCodeOutcome: open code prerequisite not blocked:; declarations/fields: `TestClosureAloneCannotSatisfyCodeOutcome` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 320–336 | Fixture/protocol support setupResultPrereq; declarations/fields: `setupResultPrereq` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 337–381 | Assertions TestResultPrereqClosureOccurrence: open result prerequisite not blocked:; declarations/fields: `TestResultPrereqClosureOccurrence` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 382–411 | Assertions TestResultPrereqNeedsAcceptedResolution: resolutionless result prerequisite queued:; declarations/fields: `TestResultPrereqNeedsAcceptedResolution` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 412–433 | Assertions TestHiddenEndpointBlocksWithoutLeaking: hidden endpoint not blocked:; declarations/fields: `TestHiddenEndpointBlocksWithoutLeaking` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 434–451 | Assertions TestHiddenIssueDeniesAuthorization: hidden issue not inaccessible:; declarations/fields: `TestHiddenIssueDeniesAuthorization` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 452–470 | Assertions TestIncompleteEvidenceBlocks: over-bound issue not blocked:; declarations/fields: `TestIncompleteEvidenceBlocks` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 471–504 | Assertions TestLongerCycleBlocks: cycle member not blocked:; declarations/fields: `TestLongerCycleBlocks` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 505–528 | Assertions TestWithdrawnEndpointInvalidatesDependent: withdrawn endpoint route authorized:; declarations/fields: `TestWithdrawnEndpointInvalidatesDependent` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 529–549 | Assertions TestEditedEndpointInvalidatesDependent: edited endpoint route authorized:; declarations/fields: `TestEditedEndpointInvalidatesDependent` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 550–569 | Assertions TestPullHintSkips: pull hint recorded:; declarations/fields: `TestPullHintSkips` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 570–581 | Assertions TestObserveWithoutReadsRefuses: unwired observation assessed; declarations/fields: `TestObserveWithoutReadsRefuses` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 582–605 | Assertions TestMissingAuthorityStaysUnauthorized: grantless issue authorized:; declarations/fields: `TestMissingAuthorityStaysUnauthorized` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 606–625 | Assertions TestReadinessVerdictPrecedence: empty verdict wrong:; declarations/fields: `TestReadinessVerdictPrecedence` |

<a id="coverage-4de07ed759a2"></a>

## [internal/factory/control/traversal.go](../../../../../internal/factory/control/traversal.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 18–25 | Record, DTO or interface contract traversalState for Issue intake and readiness; declarations/fields: `traversalState` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 26–36 | traversalState.nextSweepPage — Issue intake and readiness; declarations/fields: `traversalState.nextSweepPage` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 37–51 | traversalState.advanceSweepPage — Issue intake and readiness; declarations/fields: `traversalState.advanceSweepPage` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 52–63 | traversalState.nextSweepRepo — Issue intake and readiness; declarations/fields: `traversalState.nextSweepRepo` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 64–76 | traversalState.advanceSweepRepo — Issue intake and readiness; declarations/fields: `traversalState.advanceSweepRepo` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 77–126 | Bounded dispatch queue cursor mechanics; declarations/fields: `DispatchQueueCursor`, `NewDispatchQueueCursor`, `DispatchQueueCursor.start`, `DispatchQueueCursor.advance`, `DispatchQueueCursor.reset` |

<a id="coverage-3f65bf95377c"></a>

## [internal/factory/control/traversal_test.go](../../../../../internal/factory/control/traversal_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 16–38 | Assertions TestReadinessSweepReachesDeepPages: repeated sweeps never reached page 5: %v; declarations/fields: `TestReadinessSweepReachesDeepPages` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 39–65 | Assertions TestReadinessAllReachesDeepRepositories: repeated sweeps never reached repository 33: %v; declarations/fields: `TestReadinessAllReachesDeepRepositories` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 66–103 | Dispatch queue continuation beyond waiting prefix assertions; declarations/fields: `TestDispatchPassReachesRunnableBehindWaitingPrefix` |


# Backend store

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-f8171fb08abf"></a>

<a id="internalstorefactory_assignmentsgo-1"></a>

## [internal/store/factory_assignments.go](../../../../../internal/store/factory_assignments.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 16–21 | Declared identifiers/bounds ErrAssignmentActive for Assignment and dispatch; declarations/fields: `ErrAssignmentActive` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 22–32 | Declared identifiers/bounds (declaration group) for Assignment and dispatch; declarations/fields: `(declaration group)` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 33–35 | Declared identifiers/bounds MaxQueuedDispatch for Assignment and dispatch; declarations/fields: `MaxQueuedDispatch` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 36–50, 486–504 | Atomic capacity, planned reservation and charged usage enforcement; declarations/fields: `storeAssignedLimit`, `Store.ActiveRunCounts` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 51–129 | Store.RecordDispatchPacket — Assignment and dispatch; declarations/fields: `Store.RecordDispatchPacket` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 130–209 | Atomic held capacity/repository/connection/planned-versus-confirmed usage enforcement; reads independently owned policy/operator/sponsorship limits; declarations/fields: `checkAdmissionTx` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 210 | checkAdmissionTx — Assignment and dispatch; declarations/fields: `checkAdmissionTx` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 211–224 | admissionChanged — Assignment and dispatch; declarations/fields: `admissionChanged` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 225–231 | cappedCountTx — Assignment and dispatch; declarations/fields: `cappedCountTx` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 232–243 | dispatchPacketError — Assignment and dispatch; declarations/fields: `dispatchPacketError` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 244–255 | Store.Assignment — Assignment and dispatch; declarations/fields: `Store.Assignment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 256–267 | Store.AssignmentByRun — Assignment and dispatch; declarations/fields: `Store.AssignmentByRun` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 268–297 | Store.IssueAssignments — Assignment and dispatch; declarations/fields: `Store.IssueAssignments` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 298–400 | Store.RecordRetryPacket — Assignment and dispatch; declarations/fields: `Store.RecordRetryPacket` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 401–429 | Store.FinishAssignment — Assignment and dispatch; declarations/fields: `Store.FinishAssignment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 430–456 | Store.AssignedAssignments — Assignment and dispatch; declarations/fields: `Store.AssignedAssignments` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 457–485 | Oldest-first issue dispatch queue selection; declarations/fields: `Store.QueuedControls` |

<a id="coverage-18dff481a157"></a>

<a id="internalstorefactory_dispatch_testgo-1"></a>

## [internal/store/factory_dispatch_test.go](../../../../../internal/store/factory_dispatch_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 16–25 | Fixture/protocol support dispatchStoreFixture; declarations/fields: `dispatchStoreFixture` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 26–42, 203–378, 439–459 | Atomic admission, reservation transitions and confirmed accounting assertions; declarations/fields: `seedDispatchLimits`, `TestRecordDispatchPacketEnforcesLimits`, `TestRecordRetryPacketReholdsAndRefusesLimits`, `TestReservationTransitions`, `TestRunUsageFirstWriteWins`, `TestActiveRunCountsSkipAttributed` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 43–46 | Fixture/protocol support dispatchTestRegistration; declarations/fields: `dispatchTestRegistration` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 47–62 | Fixture/protocol support dispatchTestPrompt; declarations/fields: `dispatchTestPrompt` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 63–87 | Fixture/protocol support dispatchTestPacket; declarations/fields: `dispatchTestPacket` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 88–122 | Assertions TestRecordDispatchPacket: packet refused: %v; declarations/fields: `TestRecordDispatchPacket` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 123–126 | Fixture/protocol support isAssignmentActive; declarations/fields: `isAssignmentActive` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 127–137 | Fixture/protocol support retryTestRun; declarations/fields: `retryTestRun` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 138–202 | Assertions TestRecordRetryPacketBoundsAttemptsAndFinishes: attempt not recorded: %+v %v; declarations/fields: `TestRecordRetryPacketBoundsAttemptsAndFinishes` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 379–394 | Fixture/protocol support queuedTestControl; declarations/fields: `queuedTestControl` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 395–438 | Assertions TestQueuedControlsOldestFirst: queued count = %d; declarations/fields: `TestQueuedControlsOldestFirst` |

<a id="coverage-f41e1a04a434"></a>

## [internal/store/factory_grants.go](../../../../../internal/store/factory_grants.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 14–17 | Declared identifiers/bounds ErrStaleRevision for Repository factory policy; declarations/fields: `ErrStaleRevision` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 18–20 | Declared identifiers/bounds ErrDispatchClosed for Repository factory policy; declarations/fields: `ErrDispatchClosed` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 21–24 | Declared identifiers/bounds ErrDispatchConflict for Repository factory policy; declarations/fields: `ErrDispatchConflict` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 25–55 | Generic revisioned JSONB row/CAS mechanics, semantic wrappers retain authority; declarations/fields: `Store.saveRevisionedGrant`, `Store.loadGrant` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 56–65 | Store.SaveRepositoryPolicy — Repository factory policy; declarations/fields: `Store.SaveRepositoryPolicy` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 66–71 | Store.RepositoryPolicy — Repository factory policy; declarations/fields: `Store.RepositoryPolicy` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 72–87 | Capacity record persistence; declarations/fields: `Store.SaveCapacity`, `Store.Capacity` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 88–103 | Operator execution grant persistence; declarations/fields: `Store.SaveOperatorGrant`, `Store.OperatorGrant` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 104–164 | Connection sponsorship persistence; declarations/fields: `Store.SaveSponsorship`, `Store.Sponsorship`, `Store.Sponsorships` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 165–182, 247–321 | Dispatch gate withdrawal/reopening persistence; declarations/fields: `Store.DispatchState`, `Store.WithdrawDispatch`, `Store.ReopenDispatch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 183–246 | Immutable dispatch registration persistence; declarations/fields: `Store.RegisterDispatch`, `registerDispatchTx` |

<a id="coverage-1998756bdb5e"></a>

## [internal/store/factory_grants_test.go](../../../../../internal/store/factory_grants_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 13–14 | Fixture/protocol support grantTestTime; declarations/fields: `grantTestTime` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 15–20 | Fixture/protocol support grantStoreFixture; declarations/fields: `grantStoreFixture` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 21–38 | Fixture/protocol support grantTestPolicy; declarations/fields: `grantTestPolicy` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 39–62 | Assertions TestPolicySaveIsCAS: policy: %+v %v; declarations/fields: `TestPolicySaveIsCAS` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 63–74 | Separate capacity/operator execution records assertions partitioned below; declarations/fields: `TestCapacityAndOperatorGrant` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 75–84 | Operator execution grant revision assertions inside capacity/grant test; declarations/fields: `TestCapacityAndOperatorGrant` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 85–108 | Connection sponsorship revision/visibility persistence assertions; declarations/fields: `TestSponsorships` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 109–177 | Withdrawal ordering and immutable command replay assertions; declarations/fields: `TestDispatchWithdrawalOrdering`, `TestSettingsCommandReplay` |

<a id="coverage-957843a7f1f0"></a>

## [internal/store/factory_inventory.go](../../../../../internal/store/factory_inventory.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–19 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 20–46 | Store.FactoryUnsettledRuns — Run lifecycle and intervention; declarations/fields: `Store.FactoryUnsettledRuns` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 47–74 | Store.ProjectFactoryRuns — Run lifecycle and intervention; declarations/fields: `Store.ProjectFactoryRuns` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 75–103 | Store.ProjectUnsettledRuns — Run lifecycle and intervention; declarations/fields: `Store.ProjectUnsettledRuns` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 104–138 | Association pagination for authorized Spaces inventory; declarations/fields: `Store.SpaceProjectsAfter` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 139–174 | Stable dispatch queue continuation; declarations/fields: `Store.QueuedControlsAfter` |

<a id="coverage-27fd6cbe7d53"></a>

## [internal/store/factory_inventory_test.go](../../../../../internal/store/factory_inventory_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 12–17 | Fixture/protocol support inventoryFixture; declarations/fields: `inventoryFixture` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 18–23 | Fixture/protocol support inventoryRun; declarations/fields: `inventoryRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 24–37 | Fixture/protocol support recordInventoryRun; declarations/fields: `recordInventoryRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 38–56 | Assertions TestFactoryUnsettledRunsSkipsSettledHistory: unsettled listing missed live work: %d runs %v; declarations/fields: `TestFactoryUnsettledRunsSkipsSettledHistory` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 57–79 | Assertions TestProjectRunListsScopeToProject: project history wrong: %d runs %v; declarations/fields: `TestProjectRunListsScopeToProject` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 80–109 | Association pagination assertions; declarations/fields: `TestSpaceProjectsAfterPages` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 110–133 | Dispatch queue continuation assertions; declarations/fields: `TestQueuedControlsAfterContinues` |

<a id="coverage-98fd029563b9"></a>

<a id="internalstorefactory_publications_testgo-1"></a>

## [internal/store/factory_publications_test.go](../../../../../internal/store/factory_publications_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 14–48 | Fixture/protocol support publicationStoreFixture; declarations/fields: `publicationStoreFixture` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 49–70 | Fixture/protocol support publicationTestRecord; declarations/fields: `publicationTestRecord` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 71–92 | Assertions TestRecordPublicationRoundTrip: record: %v; declarations/fields: `TestRecordPublicationRoundTrip` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 93–125 | Assertions TestUpdatePublicationComparesAndSwaps: record: %v; declarations/fields: `TestUpdatePublicationComparesAndSwaps` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 126–140 | Fixture/protocol support finishedPublishableAssignment; declarations/fields: `finishedPublishableAssignment` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 141–164 | Fixture/protocol support recordFinishedAssignment; declarations/fields: `recordFinishedAssignment` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 165–193 | Assertions TestPublishableAssignmentsSelectsReportedCompleted: list: %v; declarations/fields: `TestPublishableAssignmentsSelectsReportedCompleted` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 194–235 | Assertions TestOutstandingPublicationsListsOpenAndFenced: record open: %v; declarations/fields: `TestOutstandingPublicationsListsOpenAndFenced` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 236–243 | Fixture/protocol support publicationStoreIntent; declarations/fields: `publicationStoreIntent` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 244–293 | Assertions TestPublicationRegistrationPersistsImmutableIntent: immutable change: %v; declarations/fields: `TestPublicationRegistrationPersistsImmutableIntent` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 294–334 | Assertions TestWithdrawalOrdersPublicationRegistrationAndKeepsReconciliation: reconcile after withdrawal: %v; declarations/fields: `TestWithdrawalOrdersPublicationRegistrationAndKeepsReconciliation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 335–356 | Assertions TestPublicationReceiptAndTerminalOutcomeCannotBeReplaced: old verdict moved to new identity: %v; declarations/fields: `TestPublicationReceiptAndTerminalOutcomeCannotBeReplaced` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 357–369 | Fixture/protocol support seedPublicationAssignment; declarations/fields: `seedPublicationAssignment` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 370–435 | Assertions TestPublicationRegistrationRejectsAuthorityChangesAfterPreflight: operation escaped changed authority: %v; declarations/fields: `TestPublicationRegistrationRejectsAuthorityChangesAfterPreflight` |

<a id="coverage-698246430cc4"></a>

## [internal/store/identity.go](../../../../../internal/store/identity.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 11–33 | Encrypted identity connection credential persistence used by current compatibility/qualification fixtures; declarations/fields: `identityBinding`, `Store.IdentitySaveConnection` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 34–65 | Shared-schema identity metadata reads and fixture delegation writes; declarations/fields: `Store.IdentityConnection`, `Store.IdentitySaveGrant`, `Store.IdentityGrant` |

<a id="coverage-fc987cc9b8c7"></a>

## [internal/store/identity_events.go](../../../../../internal/store/identity_events.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state. Go append is reached through fixture identity writers; native Rust broker appends audit transactionally. No current browser/HTTP audit-read feature is demonstrated.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 11–22 | Atomic domain write+audit transaction mechanics; declarations/fields: `Store.identityAtomic` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 23–43 | appendIdentityEvent — Identity audit history; declarations/fields: `appendIdentityEvent` |

<a id="coverage-322a6adfc409"></a>

## [internal/store/identity_test.go](../../../../../internal/store/identity_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 11–50 | Credential custody validation/round-trip fixture assertions; declarations/fields: `TestIdentityConnectionRoundTripAndValidation` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 51–67 | Assertions TestIdentityGrantRoundTrip: saved grant unreadable; declarations/fields: `TestIdentityGrantRoundTrip` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 68–113 | Append-only credential-free audit/rollback assertions; declarations/fields: `TestIdentityAuditTrailIsImmutableAndCredentialFree`, `TestIdentityAuditFailureRollsBackSave` |

<a id="coverage-27c964ad743e"></a>

## [internal/store/issue_controls.go](../../../../../internal/store/issue_controls.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 18–20 | Declared identifiers/bounds MaxIntakeDeliveries for Issue intake and readiness; declarations/fields: `MaxIntakeDeliveries` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 21–26 | Declared identifiers/bounds MaxIssueControls for Issue intake and readiness; declarations/fields: `MaxIssueControls` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 27–69 | Store.RecordIssueAssessment — Issue intake and readiness; declarations/fields: `Store.RecordIssueAssessment` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 70–82 | Store.IssueControl — Issue intake and readiness; declarations/fields: `Store.IssueControl` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 83–109 | Store.IssueControls — Issue intake and readiness; declarations/fields: `Store.IssueControls` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 110–119 | Store.IntakeDeliverySeen — Issue intake and readiness; declarations/fields: `Store.IntakeDeliverySeen` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 120–143 | Store.RecordIntakeDelivery — Issue intake and readiness; declarations/fields: `Store.RecordIntakeDelivery` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 144–192 | Store.AcceptanceDependants — Issue intake and readiness; declarations/fields: `Store.AcceptanceDependants` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 193–199 | Store.ReadinessSweepRevision — Issue intake and readiness; declarations/fields: `Store.ReadinessSweepRevision` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 200–214 | Store.SaveReadinessSweep — Issue intake and readiness; declarations/fields: `Store.SaveReadinessSweep` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 215–234 | Repository policy inventory used by readiness sweeps; declarations/fields: `Store.FactoryPolicies` |

<a id="coverage-a4970692e99c"></a>

## [internal/store/preparation.go](../../../../../internal/store/preparation.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 15–53 | Standing Project lifecycle grant CAS; declarations/fields: `Store.SaveLifecycleGrant`, `Store.LifecycleGrant` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 54–94 | Preparation admission hold CAS; declarations/fields: `Store.SaveMaintenanceHold`, `Store.MaintenanceHold` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 95–128 | Store.AdmitPreparation — Checkout allocation and preparation; declarations/fields: `Store.AdmitPreparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 129–154 | Store.ObservePreparation — Checkout allocation and preparation; declarations/fields: `Store.ObservePreparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 155–165 | Store.Preparation — Checkout allocation and preparation; declarations/fields: `Store.Preparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 166–185 | Store.ProjectPreparations — Checkout allocation and preparation; declarations/fields: `Store.ProjectPreparations` |

<a id="coverage-6439ad7b3573"></a>

## [internal/store/preparation_test.go](../../../../../internal/store/preparation_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 12–18 | Fixture/protocol support (declaration group); declarations/fields: `(declaration group)` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 19–32 | Fixture/protocol support prepTestStore; declarations/fields: `prepTestStore` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 33–41 | Fixture/protocol support prepTestPreparation; declarations/fields: `prepTestPreparation` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 42–63 | Lifecycle grant revision assertions; declarations/fields: `TestLifecycleGrantRevisionRejectsStaleWriters` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 64–83 | Persistent preparation hold assertions; declarations/fields: `TestMaintenanceHoldPersistsWithCAS` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 84–100 | Assertions TestAdmitPreparationIsIdempotentForExactInputs: admit: %v %v; declarations/fields: `TestAdmitPreparationIsIdempotentForExactInputs` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 101–128 | Assertions TestPreparationRefsAreImmutable: requirement reference mutation accepted; declarations/fields: `TestPreparationRefsAreImmutable` |

<a id="coverage-0fb359162dd4"></a>

## [internal/store/project_grants.go](../../../../../internal/store/project_grants.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 15–24 | Store.SaveEnvironmentGrant — Repository association and creation; declarations/fields: `Store.SaveEnvironmentGrant` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 25–30 | Store.EnvironmentGrant — Repository association and creation; declarations/fields: `Store.EnvironmentGrant` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 31–64 | Requirement acceptance chain/head persistence; declarations/fields: `Store.AdmitRequirementDecision`, `Store.RequirementDecision`, `Store.RequirementHead`, `Store.RequirementDepth` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 65–100 | Privileged effect approval chain/head persistence; declarations/fields: `Store.AdmitApprovalDecision`, `Store.ApprovalDecision`, `Store.ApprovalHead`, `Store.ApprovalDepth` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 101–160 | Shared immutable decision transaction/CAS mechanics; typed callers own decision semantics; declarations/fields: `Store.admitProjectDecision` |

<a id="coverage-c4b8f4ed9428"></a>

## [internal/store/project_grants_test.go](../../../../../internal/store/project_grants_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 12–22 | Fixture/protocol support grantProjectFixture; declarations/fields: `grantProjectFixture` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 23–44 | Assertions TestEnvironmentGrantIsCAS: stale environment grant saved; declarations/fields: `TestEnvironmentGrantIsCAS` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 45–84 | Requirement acceptance predecessor/replay assertions; declarations/fields: `TestRequirementDecisionChain` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 85–106 | Approval predecessor/replay assertions; declarations/fields: `TestApprovalDecisionChain` |

<a id="coverage-a0df3af0f947"></a>

## [internal/store/schema.go](../../../../../internal/store/schema.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state. Shared physical declarations/SQL rows/constructor lines have separate named-field units; this is coupled storage/projection, not competing decision ownership. Table/field/guard units identify domain record invariants; schema format/init/query mechanics remain H02. Identity event storage stays I10 even where Go writes are fixture-only.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 1–9 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 10–16 | Declared identifiers/bounds schemaFormatVersion for Storage mechanics; declarations/fields: `schemaFormatVersion` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 17–19, 73–76, 155–158 | Declared identifiers/bounds schemaStatements for Storage mechanics; declarations/fields: `schemaStatements` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 20 | Native actor mirror fields id/login and id primary-key/check, separate from local display preference; declarations/fields: `users.id`, `users.login`, `PRIMARY KEY(id)`, `CHECK(id>0)` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 20 | Soda-local name preference field and default, separate from native actor id/login; declarations/fields: `users.name`, `name TEXT NOT NULL DEFAULT ''` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 21 | Persistence schema for keys; own saved development SSH key records; declarations/fields: `keys` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 22 | Observed native Project LAN IP field/default, separate from repository association and readiness; declarations/fields: `projects.ip`, `ip TEXT NOT NULL DEFAULT ''` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 22 | Repository association fields and relation constraints, separate from profile/readiness and LAN address; declarations/fields: `projects.id`, `projects.name`, `projects.repository_id`, `projects.owner_id`, `projects.repository`, `UNIQUE(repository_id)`, `REFERENCES users(id)` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 22 | Immutable selected creation profile and confirmed readiness fields/constraints; declarations/fields: `projects.creation_profile`, `projects.ready`, `octet_length(creation_profile::text)<=1024` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 23 | Persistence schema for memberships; human Project account mapping; declarations/fields: `memberships` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 24 | Persistence schema for grant_key_check; encryption master-key binding; declarations/fields: `grant_key_check` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 25–30 | Factory run ledger and unresolved-run index; declarations/fields: `factory_runs`, `factory_unsettled_runs` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 31–35 | Immutable shared admitted command/replay ledger; declarations/fields: `factory_commands` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 36 | Persistence schema for factory_policies; declarations/fields: `factory_policies` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 37 | Persistence schema for factory_capacity; declarations/fields: `factory_capacity` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 38 | Persistence schema for factory_operator_grants; declarations/fields: `factory_operator_grants` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 39 | Persistence schema for factory_sponsorships; declarations/fields: `factory_sponsorships` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 40 | Persistence schema for factory_dispatch; withdrawal/reopening gate; declarations/fields: `factory_dispatch` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 41 | Persistence schema for factory_dispatch_regs; declarations/fields: `factory_dispatch_regs` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 42 | Persistence schema for factory_dispatch_regs repository index; declarations/fields: `factory_dispatch_regs repository index` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 43 | Persistence schema for factory_takeovers; declarations/fields: `factory_takeovers` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 44 | Persistence schema for factory_run_views; exact display locator; declarations/fields: `factory_run_views` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 45 | Persistence schema for factory_assignments; declarations/fields: `factory_assignments` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 46 | Persistence schema for unfinished assignment uniqueness; declarations/fields: `unfinished assignment uniqueness` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 47 | Persistence schema for factory_reservations; declarations/fields: `factory_reservations` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 48 | Persistence schema for factory_usage; declarations/fields: `factory_usage` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 49 | Persistence schema for issue_acceptance_decisions; declarations/fields: `issue_acceptance_decisions` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 50 | Persistence schema for issue_acceptance_heads; declarations/fields: `issue_acceptance_heads` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 51 | Persistence schema for issue_acceptance_withdrawals; declarations/fields: `issue_acceptance_withdrawals` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 52 | Persistence schema for issue_controls; declarations/fields: `issue_controls` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 53 | Persistence schema for intake_deliveries; declarations/fields: `intake_deliveries` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 54 | Persistence schema for factory_publications; declarations/fields: `factory_publications` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 55 | Persistence schema for factory_check_assessments; declarations/fields: `factory_check_assessments` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 56 | Persistence schema for factory_merges; declarations/fields: `factory_merges` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 57 | Persistence schema for factory_readiness_sweeps; declarations/fields: `factory_readiness_sweeps` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 58 | Persistence schema for project_environment_grants; declarations/fields: `project_environment_grants` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 59 | Persistence schema for project_requirement_decisions; declarations/fields: `project_requirement_decisions` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 60 | Persistence schema for project_requirement_heads; declarations/fields: `project_requirement_heads` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 61 | Persistence schema for project_approval_decisions; declarations/fields: `project_approval_decisions` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 62 | Persistence schema for project_approval_heads; declarations/fields: `project_approval_heads` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 63 | Persistence schema for identity_connections.credential; encrypted custody; declarations/fields: `identity_connections.credential` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 63 | Connection availability/generation metadata shares encrypted connection row; declarations/fields: `identity_connections.owner_id`, `identity_connections.generation`, `identity_connections.state`, `identity_connections.data` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 64 | Persistence schema for identity_grants; declarations/fields: `identity_grants` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 65 | Persistence schema for identity_grant_recipient uniqueness; declarations/fields: `identity_grant_recipient uniqueness` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 66 | Persistence schema for identity_leases; declarations/fields: `identity_leases` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 67 | Persistence schema for identity_events immutable audit; declarations/fields: `identity_events immutable audit` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 68 | Persistence schema for project_lifecycle_grants; declarations/fields: `project_lifecycle_grants` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 69 | Persistence schema for project_maintenance; declarations/fields: `project_maintenance` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 70 | Persistence schema for project_preparations; declarations/fields: `project_preparations` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 71 | Persistence schema for project_preparations_project index; declarations/fields: `project_preparations_project index` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 72 | Persistence schema for identity_executions; declarations/fields: `identity_executions` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 77–83, 134, 249–252 | Immutable selected native creation profile guard/trigger verification; declarations/fields: `soda_guard_creation_profile`, `immutable_creation_profile`, `verifyImmutableCreationProfile` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 84–113 | Immutable factory run identity/terminal progression and command receipt guards; declarations/fields: `soda_guard_run_binding`, `soda_guard_command` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 114–123 | Immutable exact preparation role/decision reference guard; declarations/fields: `soda_guard_preparation_refs` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 124–133 | Immutable execution digest and terminal admission fence guard; declarations/fields: `soda_guard_execution_identity` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 135 | Persistence schema for factory_run_binding_immutable trigger; declarations/fields: `factory_run_binding_immutable trigger` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 136 | Persistence schema for factory_command_immutable trigger; declarations/fields: `factory_command_immutable trigger` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 137 | Persistence schema for factory_dispatch registration immutable update; declarations/fields: `factory_dispatch registration immutable update` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 138 | Persistence schema for factory_dispatch registration immutable delete; declarations/fields: `factory_dispatch registration immutable delete` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 139 | Persistence schema for factory_takeover immutable update; declarations/fields: `factory_takeover immutable update` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 140 | Persistence schema for factory_takeover immutable delete; declarations/fields: `factory_takeover immutable delete` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 141 | Persistence schema for factory_run_view immutable update; declarations/fields: `factory_run_view immutable update` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 142 | Persistence schema for factory_run_view immutable delete; declarations/fields: `factory_run_view immutable delete` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 143 | Persistence schema for issue_acceptance decision immutable update; declarations/fields: `issue_acceptance decision immutable update` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 144 | Persistence schema for issue_acceptance decision immutable delete; declarations/fields: `issue_acceptance decision immutable delete` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 145 | Persistence schema for issue_acceptance withdrawal immutable update; declarations/fields: `issue_acceptance withdrawal immutable update` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 146 | Persistence schema for issue_acceptance withdrawal immutable delete; declarations/fields: `issue_acceptance withdrawal immutable delete` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 147 | Persistence schema for project_requirement decision immutable update; declarations/fields: `project_requirement decision immutable update` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 148 | Persistence schema for project_requirement decision immutable delete; declarations/fields: `project_requirement decision immutable delete` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 149 | Persistence schema for project_approval decision immutable update; declarations/fields: `project_approval decision immutable update` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 150 | Persistence schema for project_approval decision immutable delete; declarations/fields: `project_approval decision immutable delete` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 151 | Persistence schema for identity_event immutable update; declarations/fields: `identity_event immutable update` |
| [I10](../../slices/identity-brokering.md#i10-identity-audit-history) / active | 152 | Persistence schema for identity_event immutable delete; declarations/fields: `identity_event immutable delete` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 153 | Persistence schema for preparation_refs_immutable trigger; declarations/fields: `preparation_refs_immutable trigger` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 154 | Persistence schema for identity_execution_immutable trigger; declarations/fields: `identity_execution_immutable trigger` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 159–160 | SchemaVersion — Storage mechanics; declarations/fields: `SchemaVersion` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 161–168 | schemaPresent — Storage mechanics; declarations/fields: `schemaPresent` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 169–179 | refuseUnversionedDatabase — Storage mechanics; declarations/fields: `refuseUnversionedDatabase` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 180–191 | storedSchemaVersion — Storage mechanics; declarations/fields: `storedSchemaVersion` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 192–193, 226–237 | verifyRequiredColumns — Storage mechanics; declarations/fields: `verifyRequiredColumns` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 194 | Required native actor mirror field selectors in shared users read; declarations/fields: `users.id`, `users.login` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 194 | Required local preference selector in shared users schema verification; declarations/fields: `users.name` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 195 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `keys` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 196 | Required native LAN IP selector in shared Project schema verification; declarations/fields: `projects.ip` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 196 | Required repository association field selectors in shared Project schema verification; declarations/fields: `projects.id`, `projects.name`, `projects.repository_id`, `projects.owner_id`, `projects.repository` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 196 | Required immutable profile/readiness selectors in shared Project schema verification; declarations/fields: `projects.creation_profile`, `projects.ready` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 197 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `memberships` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 198 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `grant_key_check` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 199 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `project_lifecycle_grants` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 200 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `project_maintenance` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 201 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `project_preparations` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 202 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `identity_executions` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 203–204, 209, 214 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_runs`, `factory_commands`, `factory_dispatch`, `factory_takeovers` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 205 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_policies` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 206, 212–213 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_capacity`, `factory_reservations`, `factory_usage` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 207 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_operator_grants` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 208 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_sponsorships` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 210–211 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_dispatch_regs`, `factory_assignments` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 215–217 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `issue_acceptance_decisions`, `issue_acceptance_heads`, `issue_acceptance_withdrawals` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 218 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `project_environment_grants` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 219–220 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `project_requirement_decisions`, `project_requirement_heads` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 221–222 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `project_approval_decisions`, `project_approval_heads` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 223 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_run_views` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 224 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_publications` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 225 | Required current domain table/column group; declarations/fields: `verifyRequiredColumns`, `factory_merges` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 238–248 | verifyTrigger — Storage mechanics; declarations/fields: `verifyTrigger` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 253–256 | Verify immutable exact preparation decision references; declarations/fields: `verifyImmutablePreparationRefs` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 257–260 | Verify immutable execution fence and terminal state; declarations/fields: `verifyImmutableExecutionIdentity` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 261–271 | loadSchemaVersion — Storage mechanics; declarations/fields: `loadSchemaVersion` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 272–306 | initializeSchema — Storage mechanics; declarations/fields: `initializeSchema` |

<a id="coverage-b26d414d0eb2"></a>

## [internal/store/store.go](../../../../../internal/store/store.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state. Shared physical declarations/SQL rows/constructor lines have separate named-field units; this is coupled storage/projection, not competing decision ownership.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 19–21 | Declared identifiers/bounds ErrNotFound for Storage mechanics; declarations/fields: `ErrNotFound` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 22–23 | Immutable command identity conflict sentinel; declarations/fields: `ErrCommandConflict` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 24–27 | Record, DTO or interface contract Store for Storage mechanics; declarations/fields: `Store` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 28–29, 31, 43–53, 181–184, 186, 189–190, 192–194 | Native admitted actor mirror; local display name retains its separate G09 owner; declarations/fields: `User`, `Session`, `Store.UpsertUser`, `Store.User` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 30 | Native actor login and persisted local name share one Go field declaration; declarations/fields: `User.Login` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 30 | Soda-local profile display name distinct from mirrored login; declarations/fields: `User.Name` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 32–35, 203–233 | Own saved development public key persistence; declarations/fields: `Key`, `Store.AddKey`, `Store.RemoveKey`, `Store.Keys` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 36, 38–39, 42, 234–238, 247–249, 255–257, 262–264, 266–273, 294–297 | Repository association persistence/row assembly; profile, readiness and LAN fields have separate named selectors; declarations/fields: `Project`, `Store.CreateProject`, `scanProject`, `projectColumns`, `Store.Project`, `Store.ProjectByRepository` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 37 | Immutable selected native creation profile DTO reference; declarations/fields: `Project.Profile` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 40 | Native observed LAN IP field in shared Go declaration; declarations/fields: `Project.IP` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 40 | Repository label field in shared Go declaration; declarations/fields: `Project.Repository` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 41 | Native Project confirmed provisioning readiness; declarations/fields: `Project.Ready` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 54–82 | bind — Storage mechanics; declarations/fields: `bind` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 83–86 | Store.exec — Storage mechanics; declarations/fields: `Store.exec` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 87–90 | Store.query — Storage mechanics; declarations/fields: `Store.query` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 91–97 | Store.queryRow — Storage mechanics; declarations/fields: `Store.queryRow` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 98–99 | Record, DTO or interface contract tx for Storage mechanics; declarations/fields: `tx` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 100–107 | Store.begin — Storage mechanics; declarations/fields: `Store.begin` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 108–109 | tx.Commit — Storage mechanics; declarations/fields: `tx.Commit` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 110–111 | tx.Rollback — Storage mechanics; declarations/fields: `tx.Rollback` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 112–115 | tx.exec — Storage mechanics; declarations/fields: `tx.exec` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 116–119 | tx.query — Storage mechanics; declarations/fields: `tx.query` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 120–126 | tx.queryRow — Storage mechanics; declarations/fields: `tx.queryRow` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 127–136 | sameJSONDocument — Storage mechanics; declarations/fields: `sameJSONDocument` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 137–140 | Open — Storage mechanics; declarations/fields: `Open` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 141–148 | Production encryption key binding before schema initialization; declarations/fields: `OpenEncrypted` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 149–154 | Small PostgreSQL connection pool initialization; declarations/fields: `configureStore` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 155 | Bind provisioned credential encryption key to existing stored check; declarations/fields: `checkGrantKey` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 156–158 | Create/refuse current schema according to database format; declarations/fields: `initializeSchema` |
| [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) / active | 159–161 | Initialize encryption key check only after current schema admission; declarations/fields: `initializeGrantKey` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 162–168 | configureStore — Storage mechanics; declarations/fields: `configureStore` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 169–178 | open — Storage mechanics; declarations/fields: `open` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 179–180 | Store.Close — Storage mechanics; declarations/fields: `Store.Close` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 185 | Native actor id/login INSERT and conflict update of login only; declarations/fields: `users.id`, `users.login`, `ON CONFLICT(id) DO UPDATE SET login=excluded.login` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 185 | Initial local name INSERT; conflict branch preserves existing display-name preference; declarations/fields: `users.name`, `u.Name` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 187–188 | Return from admitted native actor mirror write; declarations/fields: `Store.UpsertUser` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 191 | Native actor id/login selectors in shared User read; declarations/fields: `users.id`, `users.login`, `User.ID`, `User.Login` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 191 | Local display-name preference selector in shared User read; declarations/fields: `users.name`, `User.Name` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 195–202 | Soda-local display-name preference persistence; declarations/fields: `Store.RenameProfile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 239–245 | Validate and encode immutable selected native creation profile; declarations/fields: `Project.Profile`, `Profile.Validate`, `json.Marshal(p.Profile)` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 246 | Create repository association fields in composite INSERT; declarations/fields: `projects.id`, `projects.name`, `projects.repository_id`, `projects.owner_id`, `projects.repository` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 246 | Persist immutable selected native creation profile in composite INSERT; declarations/fields: `projects.creation_profile`, `string(raw)` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 250, 252–254 | Confirmed native Project runtime readiness persistence; declarations/fields: `Store.MarkReady` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 251 | Persist observed native LAN IP in shared readiness UPDATE; declarations/fields: `projects.ip`, `ip` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 251 | Mark confirmed native provisioning readiness in shared UPDATE; declarations/fields: `projects.ready`, `ready=TRUE` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 258 | Decode native LAN IP selector from shared Project row; declarations/fields: `Project.IP` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 258 | Decode repository association fields from shared Project row; declarations/fields: `Project.ID`, `Project.Name`, `Project.RepositoryID`, `Project.OwnerID`, `Project.Repository` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 258 | Decode current readiness and immutable profile selectors from shared Project row; declarations/fields: `Project.Ready`, `profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 259–261 | Decode immutable selected creation profile from nullable JSON; declarations/fields: `Project.Profile`, `project.Decode` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 265 | Shared SELECT native LAN IP column; declarations/fields: `ip` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 265 | Shared SELECT association column group; declarations/fields: `id`, `name`, `repository_id`, `owner_id`, `repository` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 265 | Shared SELECT immutable profile/readiness column group; declarations/fields: `ready`, `creation_profile` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 274–293 | Bounded association scan for authorized inventory; caller authorizes each row; declarations/fields: `Store.SpaceProjects` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 298–307 | Human Project membership/account persistence; declarations/fields: `Store.Join`, `Store.MemberLogin` |

<a id="coverage-f4cfe40ce089"></a>

## [internal/store/store_test.go](../../../../../internal/store/store_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 12–68 | Bounded Project memberships and persistent own account assertions; declarations/fields: `TestMembersListingIsBounded`, `TestPersistenceAndMembership` |
| [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) / active | 69–84 | Assertions TestSetupSocketDSNParses: host %q, want socket directory; declarations/fields: `TestSetupSocketDSNParses` |


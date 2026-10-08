# Backend store

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-fa3146f813f8"></a>

## [internal/store/corruption.go](../../../../../internal/store/corruption.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; file scaffold | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–45; InjectCorruptFactoryRun; DropFactoryRunViews | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: InjectCorruptFactoryRun; Current declaration duty: DropFactoryRunViews — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-01de9d0ef58e"></a>

## [internal/store/ephemeral.go](../../../../../internal/store/ephemeral.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–104; file scaffold; TestFixtureDSN; OpenEphemeral; createEphemeralDatabase; ephemeralName | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-28fa99a4c0e2"></a>

## [internal/store/factory.go](../../../../../internal/store/factory.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–181; file scaffold; RecordFactoryRun; FactoryRun; SaveFactoryRun; FactoryRuns; RecordFactoryCommand; FactoryCommand; UnfinishedCommands; FinishFactoryCommand | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f8171fb08abf"></a>
<a id="internalstorefactory_assignmentsgo-1"></a>

## [internal/store/factory_assignments.go](../../../../../internal/store/factory_assignments.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 15–115; file scaffold; Assignment; AssignmentByRun; IssueAssignments; FinishAssignment; AssignedAssignments | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–14; storeAssignedLimit | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: storeAssignedLimit — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-082686001ffc"></a>

## [internal/store/factory_checks.go](../../../../../internal/store/factory_checks.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–52; file scaffold; RecordCheckAssessment; CheckAssessment | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: RecordCheckAssessment; Current declaration duty: CheckAssessment — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4e45ba771259"></a>

## [internal/store/factory_checks_test.go](../../../../../internal/store/factory_checks_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 39–92; file scaffold; TestRecordCheckAssessmentRoundTrip; TestRecordCheckAssessmentRejectsMalformed | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRecordCheckAssessmentRoundTrip; Current declaration duty: TestRecordCheckAssessmentRejectsMalformed — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–38; checkAssessmentFixture | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current declaration duty: checkAssessmentFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-59e7c8af2798"></a>

## [internal/store/factory_dispatch_packet.go](../../../../../internal/store/factory_dispatch_packet.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123, 205–236; file scaffold; ErrAssignmentActive; RecordDispatchPacket; admissionChanged; cappedCountTx; dispatchPacketError | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 124–204; checkAdmissionTx | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: checkAdmissionTx — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4c931bde530b"></a>

## [internal/store/factory_dispatch_queue.go](../../../../../internal/store/factory_dispatch_queue.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–66; file scaffold; MaxQueuedDispatch; QueuedControls; ActiveRunCounts | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-224426eeb0d0"></a>

## [internal/store/factory_dispatch_queue_test.go](../../../../../internal/store/factory_dispatch_queue_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–57, 88–168; file scaffold; TestReservationTransitions; queuedTestControl; TestQueuedControlsOldestFirst; TestActiveRunCountsSkipAttributed | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 58–87; TestRunUsageFirstWriteWins | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: TestRunUsageFirstWriteWins — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-18dff481a157"></a>
<a id="internalstorefactory_dispatch_testgo-1"></a>

## [internal/store/factory_dispatch_test.go](../../../../../internal/store/factory_dispatch_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25, 43–126; file scaffold; dispatchStoreFixture; dispatchTestRegistration; dispatchTestPrompt; dispatchTestPacket; TestRecordDispatchPacket; isAssignmentActive | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 26–42, 127–181; seedDispatchLimits; TestRecordDispatchPacketEnforcesLimits | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: seedDispatchLimits; Current declaration duty: TestRecordDispatchPacketEnforcesLimits — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f41e1a04a434"></a>

## [internal/store/factory_grants.go](../../../../../internal/store/factory_grants.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; file scaffold | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–25, 57–72; ErrStaleRevision; ErrDispatchClosed; ErrDispatchConflict; SaveRepositoryPolicy; RepositoryPolicy | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Current declaration duty: ErrStaleRevision; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 26–56; saveRevisionedGrant; loadGrant | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current declaration duty: saveRevisionedGrant; Current declaration duty: loadGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 73–88; SaveCapacity; Capacity | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: SaveCapacity; Current declaration duty: Capacity — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 89–104; SaveOperatorGrant; OperatorGrant | [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) | retained | Current declaration duty: SaveOperatorGrant; Current declaration duty: OperatorGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 105–165; SaveSponsorship; Sponsorship; Sponsorships | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Current declaration duty: SaveSponsorship; Current declaration duty: Sponsorship; Current declaration duty: Sponsorships — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 166–183, 248–322, 377–457; DispatchState; WithdrawDispatch; ReopenDispatch; ClearProjectStop; clearDispatchCauses | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention); [P05](../../slices/projects.md#p05-project-startstop) | retained | The shared row-lock/CAS path composes active causes while preserving the first-closure receipt. F08 Resume clears non-Project causes; P05 verified Start clears only `project_stop`. `8d9485af` and its owned Store checks prove both causes can compose and clear independently. |
| 184–247; RegisterDispatch; registerDispatchTx | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: RegisterDispatch; Current declaration duty: registerDispatchTx — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1998756bdb5e"></a>

## [internal/store/factory_grants_test.go](../../../../../internal/store/factory_grants_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; whole file; grantTestTime; grantStoreFixture; grantTestPolicy; TestPolicySaveIsCAS | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 63–74; TestCapacityAndOperatorGrant | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Separate capacity/operator execution records assertions partitioned below; declarations/fields: `TestCapacityAndOperatorGrant` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 75–84; TestCapacityAndOperatorGrant | [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) | retained | Operator execution grant revision assertions inside capacity/grant test; declarations/fields: `TestCapacityAndOperatorGrant` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 85–108; TestSponsorships | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Connection sponsorship revision/visibility persistence assertions; declarations/fields: `TestSponsorships` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 109–154, 213–235; TestDispatchWithdrawalOrdering, TestSettingsCommandReplay | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Withdrawal ordering and immutable command replay assertions; declarations/fields: `TestDispatchWithdrawalOrdering`, `TestSettingsCommandReplay` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 155–212; TestDispatchCausesComposeAndClearIndependently; TestProjectStopCauseComposesAfterExistingPause | [P05](../../slices/projects.md#p05-project-startstop) | retained | Store regressions prove active causes compose without replacing the first-closure receipt, and verified Start clears only the Project-stop cause. Source/test scope `8d9485af`; owned PostgreSQL checks passed. |

<a id="coverage-957843a7f1f0"></a>

## [internal/store/factory_inventory.go](../../../../../internal/store/factory_inventory.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 139–174; file scaffold; QueuedControlsAfter | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: QueuedControlsAfter — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 20–103; FactoryUnsettledRuns; ProjectFactoryRuns; ProjectUnsettledRuns | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current declaration duty: FactoryUnsettledRuns; Current declaration duty: ProjectFactoryRuns; Current declaration duty: ProjectUnsettledRuns — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 104–138; SpaceProjectsAfter | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: SpaceProjectsAfter — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-27fd6cbe7d53"></a>

## [internal/store/factory_inventory_test.go](../../../../../internal/store/factory_inventory_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; whole file; inventoryFixture; inventoryRun; recordInventoryRun; TestFactoryUnsettledRunsSkipsSettledHistory; TestProjectRunListsScopeToProject | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 80–109; TestSpaceProjectsAfterPages | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Association pagination assertions; declarations/fields: `TestSpaceProjectsAfterPages` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 110–133; TestQueuedControlsAfterContinues | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Dispatch queue continuation assertions; declarations/fields: `TestQueuedControlsAfterContinues` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-713894fb4d96"></a>

## [internal/store/factory_lifecycle.go](../../../../../internal/store/factory_lifecycle.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 19–58; file scaffold; RecordTakeover; Takeover | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: RecordTakeover; Current declaration duty: Takeover — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 14–18; ErrTakeoverConflict | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: ErrTakeoverConflict — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9ea27a6ad525"></a>

## [internal/store/factory_lifecycle_test.go](../../../../../internal/store/factory_lifecycle_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–70; file scaffold; takeoverStore; takeoverFixture; TestTakeoverRecordReplays; TestTakeoverUnknownIsNotFound; TestTakeoverRecordValidates | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bcb338cb890a"></a>

## [internal/store/factory_merges.go](../../../../../internal/store/factory_merges.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 16–18, 24–42, 145–235, 265–286; file scaffold; storeMergeableLimit; RecordMerge; mergeRebaseAllowed; mergeOperationUpdateAllowed; OutstandingMerges; OpenMerges; MergeForIssue; IssueMergeCompletion | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–15, 19–23, 43–120, 236–264; storeMergeLimit; ErrMergeConflict; MergeByPublication; UpdateMerge; MergeablePublications | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: storeMergeLimit; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 121–144; mergeUpdateAllowed | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: mergeUpdateAllowed — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cb38cac4b7e9"></a>

## [internal/store/factory_merges_test.go](../../../../../internal/store/factory_merges_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 39–49; file scaffold; mergeTestIntent | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold; Current declaration duty: mergeTestIntent — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–38, 72–122; mergeTestRecord; seedMergeAssignment; TestUpdateMergeRegistersOnce; TestUpdateMergeGateRefusesWithdrawnDispatch | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: mergeTestRecord; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 50–71, 123–152; TestRecordMergeRoundTrip; TestMergeListings | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current declaration duty: TestRecordMergeRoundTrip; Current declaration duty: TestMergeListings — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-62af5f6eb272"></a>

## [internal/store/factory_publication_intent_test.go](../../../../../internal/store/factory_publication_intent_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 125–190; file scaffold; TestPublicationRegistrationRejectsAuthorityChangesAfterPreflight | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestPublicationRegistrationRejectsAuthorityChangesAfterPreflight — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–124; TestPublicationRegistrationPersistsImmutableIntent; TestWithdrawalOrdersPublicationRegistrationAndKeepsReconciliation; TestPublicationReceiptAndTerminalOutcomeCannotBeReplaced | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: TestPublicationRegistrationPersistsImmutableIntent; Current declaration duty: TestWithdrawalOrdersPublicationRegistrationAndKeepsReconciliation; Current declaration duty: TestPublicationReceiptAndTerminalOutcomeCannotBeReplaced — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0d33802d4366"></a>

## [internal/store/factory_publications.go](../../../../../internal/store/factory_publications.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 16–18, 24–42, 155–233; file scaffold; storePublishableLimit; RecordPublication; publicationOperationUpdateAllowed; OutstandingPublications; OpenPublications | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–15, 19–23, 43–154, 234–263; storePublicationLimit; ErrPublicationConflict; PublicationByAssignment; UpdatePublication; publicationUpdateAllowed; PublishableAssignments | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: storePublicationLimit; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-98fd029563b9"></a>
<a id="internalstorefactory_publications_testgo-1"></a>

## [internal/store/factory_publications_test.go](../../../../../internal/store/factory_publications_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; file scaffold; publicationStoreFixture; publicationTestRecord; TestRecordPublicationRoundTrip; TestUpdatePublicationComparesAndSwaps; finishedPublishableAssignment; recordFinishedAssignment; TestPublishableAssignmentsSelectsReportedCompleted; TestOutstandingPublicationsListsOpenAndFenced; publicationStoreIntent; seedPublicationAssignment | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8b646d5da10a"></a>

## [internal/store/factory_reservations.go](../../../../../internal/store/factory_reservations.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 16–143; file scaffold; Reservation; HeldReservations; transitionReservation; ConsumeReservation; ReleaseReservation; ReholdReservation; RecordRunUsage; RunUsage | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–15, 144–149; MaxHeldReservations; UsageTotal | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: MaxHeldReservations; Current declaration duty: UsageTotal — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a7c97b7afae2"></a>

## [internal/store/factory_retry_packet.go](../../../../../internal/store/factory_retry_packet.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–118; file scaffold; RecordRetryPacket | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: RecordRetryPacket — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-17bbf6ce34a1"></a>

## [internal/store/factory_retry_packet_test.go](../../../../../internal/store/factory_retry_packet_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–131; file scaffold; retryTestRun; TestRecordRetryPacketBoundsAttemptsAndFinishes; TestRecordRetryPacketReholdsAndRefusesLimits | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e2d89894d729"></a>

## [internal/store/factory_review_role_test.go](../../../../../internal/store/factory_review_role_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11; file scaffold | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–35; TestST10PublishableAssignmentsExcludeReviewer | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: TestST10PublishableAssignmentsExcludeReviewer — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-65b68647f1f5"></a>

## [internal/store/factory_test.go](../../../../../internal/store/factory_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 60–83; file scaffold; factoryFixture; factoryRun; TestFactoryRunsListNewestFirst | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 24–59, 84–111; TestFactoryRunIdentityOnlyAdvances; TestFactoryCommandIdentityConflictsOnChangedPayload | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: TestFactoryRunIdentityOnlyAdvances; Current declaration duty: TestFactoryCommandIdentityConflictsOnChangedPayload — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ac52ec4ff146"></a>

## [internal/store/factory_views.go](../../../../../internal/store/factory_views.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 17–73; file scaffold; RecordFactoryRunView; FactoryRunView; FactoryRunViews | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–16; ErrRunViewConflict | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: ErrRunViewConflict — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9d3d2af8b0b0"></a>

## [internal/store/factory_views_test.go](../../../../../internal/store/factory_views_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–76; file scaffold; TestFactoryRunViewRecordReplays; TestFactoryRunViewRequiresItsRun; TestFactoryRunViewsListNewestFirst | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c4a690b69098"></a>

## [internal/store/grants.go](../../../../../internal/store/grants.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; file scaffold; ErrGrantKey; grantCipher; newGrantCipher; seal; open; keyBinding; checkGrantKey; rejectUnkeyedIdentityCredentials; validateGrantKey; initializeGrantKey | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e51b4b0cbb6f"></a>

## [internal/store/grants_test.go](../../../../../internal/store/grants_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–57; file scaffold; TestIdentityEncryptionKeyMustSurviveRestart | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestIdentityEncryptionKeyMustSurviveRestart — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c225f2b12d4f"></a>

## [internal/store/identity_fixture.go](../../../../../internal/store/identity_fixture.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; file scaffold; seedIdentityInsert; SeedIdentityConnection; SeedIdentityGrant; identityBinding; identityAtomic; appendIdentityEvent | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-322a6adfc409"></a>

## [internal/store/identity_test.go](../../../../../internal/store/identity_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 60–83; file scaffold; TestIdentityGrantRoundTrip | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestIdentityGrantRoundTrip — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–59; TestIdentityConnectionRoundTripAndValidation | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Current declaration duty: TestIdentityConnectionRoundTripAndValidation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 84–130; TestIdentityAuditTrailIsImmutableAndCredentialFree; TestIdentityAuditFailureRollsBackSave | [I10](../../slices/identity-brokering.md#i10-identity-audit-history) | retained | Current declaration duty: TestIdentityAuditTrailIsImmutableAndCredentialFree; Current declaration duty: TestIdentityAuditFailureRollsBackSave — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3b5c8b9b53f4"></a>

## [internal/store/issue_acceptances.go](../../../../../internal/store/issue_acceptances.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147; file scaffold; AdmitAcceptanceDecision; AcceptanceDecision; AcceptanceHead; AcceptanceDepth; WithdrawAcceptanceDecision; AcceptanceWithdrawn | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b99dec7916ab"></a>

## [internal/store/issue_acceptances_test.go](../../../../../internal/store/issue_acceptances_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102; file scaffold; acceptanceFixture; TestAdmitAcceptanceChainsPredecessors; TestWithdrawAcceptanceLatchesHead | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-27c964ad743e"></a>

## [internal/store/issue_controls.go](../../../../../internal/store/issue_controls.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 22–205, 208–229; RecordIssueAssessment, IssueControl(s), intake delivery, AcceptanceDependants and readiness sweep state | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | `AcceptanceDependants` uses stable 64-row keyset pages, closes each cursor before decision reads and propagates late errors. Total-visited, recursion and query bounds remain open under RES-GO-ACCEPTANCE-DEPENDANTS-1 |
| 231–252; FactoryPolicies | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Repository factory policy listing for readiness sweeps |

<a id="coverage-aaa37b27b70e"></a>

## [internal/store/issue_controls_test.go](../../../../../internal/store/issue_controls_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–250; current intake, readiness and accepted-dependant fixtures; TestAcceptanceDependants; TestVisitAcceptanceDependantsPagesAndPropagatesLateFailure; TestReadinessSweepState and related Store tests | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Store page/retry packet passed 38 actual PostgreSQL checks (5 + 33); exact test cluster stopped; no live DB snapshot. Total traversal/resource profile remains open |

<a id="coverage-cad8b9caa61e"></a>

## [internal/store/members.go](../../../../../internal/store/members.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–28; file scaffold; Member; Members | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Current scaffold duty: file scaffold; Current declaration duty: Member; Current declaration duty: Members — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d6b2594c3e40"></a>

## [internal/store/observe.go](../../../../../internal/store/observe.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 25–70; file scaffold; ReadSchemaVersion; OpenObserve | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold; Current declaration duty: ReadSchemaVersion; Current declaration duty: OpenObserve — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 10–24; openReadOnly | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: openReadOnly — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 71–95; IntegrityCheck | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: IntegrityCheck — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a53d5628d8a4"></a>

## [internal/store/observe_test.go](../../../../../internal/store/observe_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–8; file scaffold | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 9–71; TestReadSchemaVersionAndOpenObserve | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Current declaration duty: TestReadSchemaVersionAndOpenObserve — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-278c771f6a09"></a>

## [internal/store/postgres_fixture_test.go](../../../../../internal/store/postgres_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; file scaffold; postgresFixture | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold; Current declaration duty: postgresFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a4970692e99c"></a>

## [internal/store/preparation.go](../../../../../internal/store/preparation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 95–185; file scaffold; AdmitPreparation; ObservePreparation; Preparation; ProjectPreparations | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–53; SaveLifecycleGrant; LifecycleGrant | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: SaveLifecycleGrant; Current declaration duty: LifecycleGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 54–94; SaveMaintenanceHold; MaintenanceHold | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Current declaration duty: SaveMaintenanceHold; Current declaration duty: MaintenanceHold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6439ad7b3573"></a>

## [internal/store/preparation_test.go](../../../../../internal/store/preparation_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41, 84–128; file scaffold; prepTestStore; prepTestPreparation; TestAdmitPreparationIsIdempotentForExactInputs; TestPreparationRefsAreImmutable | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 42–63; TestLifecycleGrantRevisionRejectsStaleWriters | [P05](../../slices/projects.md#p05-project-startstop) | retained | Current declaration duty: TestLifecycleGrantRevisionRejectsStaleWriters — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 64–83; TestMaintenanceHoldPersistsWithCAS | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Current declaration duty: TestMaintenanceHoldPersistsWithCAS — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0fb359162dd4"></a>

## [internal/store/project_grants.go](../../../../../internal/store/project_grants.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; file scaffold | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–30; SaveEnvironmentGrant; EnvironmentGrant | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: SaveEnvironmentGrant; Current declaration duty: EnvironmentGrant — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–64; AdmitRequirementDecision; RequirementDecision; RequirementHead; RequirementDepth | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Current declaration duty: AdmitRequirementDecision; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 65–100; AdmitApprovalDecision; ApprovalDecision; ApprovalHead; ApprovalDepth | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Current declaration duty: AdmitApprovalDecision; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 101–160; admitProjectDecision | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current declaration duty: admitProjectDecision — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c4b8f4ed9428"></a>

## [internal/store/project_grants_test.go](../../../../../internal/store/project_grants_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–44; whole file; grantProjectFixture; TestEnvironmentGrantIsCAS | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Fixture/protocol support grantProjectFixture; declarations/fields: `grantProjectFixture`; Assertions TestEnvironmentGrantIsCAS: stale environment grant saved; declarations/fields: `TestEnvironmentGrantIsCAS` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 45–84; TestRequirementDecisionChain | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Requirement acceptance predecessor/replay assertions; declarations/fields: `TestRequirementDecisionChain` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 85–106; TestApprovalDecisionChain | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Approval predecessor/replay assertions; declarations/fields: `TestApprovalDecisionChain` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-2d6fbf5e7c8a"></a>

## [internal/store/project_profile_test.go](../../../../../internal/store/project_profile_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; TestProjectWithoutCreationProfileAndImmutableCreation | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestProjectWithoutCreationProfileAndImmutableCreation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a0df3af0f947"></a>

## [internal/store/schema.go](../../../../../internal/store/schema.go)

Current canonical Go schema source and Rust literal mirror inspected by SQL statement; domain tables/columns and their domain-specific guards follow the owning slice; generic guard implementation, schema version/admission, indexes and verification mechanics remain H02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 30, 42, 46, 65, 71, 73–76, 155–248, 261–306; file scaffold; schemaFormatVersion; schema_version PostgreSQL table/query/trigger declaration; factory_unsettled_runs PostgreSQL table/query/trigger declaration; factory_dispatch_regs_repository PostgreSQL table/query/trigger declaration; factory_unfinished_assignment PostgreSQL table/query/trigger declaration; identity_grant_recipient PostgreSQL table/query/trigger declaration; project_preparations_project PostgreSQL table/query/trigger declaration; as PostgreSQL table/query/trigger declaration; SchemaVersion; schemaPresent; refuseUnversionedDatabase; storedSchemaVersion; verifyRequiredColumns; verifyTrigger; loadSchemaVersion; initializeSchema | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Package/import, DDL array declaration/closing syntax and schema-version documentation shell for canonical PostgreSQL schema; domain literals have separate owners.; 18 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 20; users.id/login actor mirror | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | users.id/login actor mirror: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 20-20; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 11-11; schema.rs drift test enforces equality |
| 20; users.name display-name preference | [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) | retained | users.name display-name preference: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 20-20; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 11-11; schema.rs drift test enforces equality |
| 21; keys PostgreSQL table/query/trigger declaration | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | keys PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 21-21; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 12-12; schema.rs drift test enforces equality |
| 22, 77–83, 134, 249–252; projects readiness and creation_profile fields; function soda_guard_creation_profile; immutable_creation_profile domain immutability trigger; verifyImmutableCreationProfile | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | projects readiness and creation_profile fields: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 22; projects repository association and owner fields | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | projects repository association and owner fields: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 22-22; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 13-13; schema.rs drift test enforces equality |
| 22; projects.ip LAN observation field | [N02](../../slices/networking.md#n02-project-lan-access) | retained | projects.ip LAN observation field: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 22-22; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 13-13; schema.rs drift test enforces equality |
| 23; memberships PostgreSQL table/query/trigger declaration | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | memberships PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 23-23; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 14-14; schema.rs drift test enforces equality |
| 24, 63; grant_key_check PostgreSQL table/query/trigger declaration; identity_connections PostgreSQL table/query/trigger declaration | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | grant_key_check PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; identity_connections PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 24-24; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 15-15; schema.rs drift test enforces equality; current Go schemaStatements literal internal/store/schema.go lines 63-63; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 54-54; schema.rs drift test enforces equality |
| 25–29, 31–35, 43, 84–113, 135–136, 139–140; factory_runs PostgreSQL table/query/trigger declaration; factory_commands PostgreSQL table/query/trigger declaration; factory_takeovers PostgreSQL table/query/trigger declaration; function soda_guard_run_binding; function soda_guard_command; factory_run_binding_immutable domain immutability trigger; factory_command_immutable domain immutability trigger; factory_takeover_immutable_update domain immutability trigger; factory_takeover_immutable_delete domain immutability trigger | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | factory_runs PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 9 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 36; factory_policies PostgreSQL table/query/trigger declaration | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | factory_policies PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 36-36; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 27-27; schema.rs drift test enforces equality |
| 37, 47–48; factory_capacity PostgreSQL table/query/trigger declaration; factory_reservations PostgreSQL table/query/trigger declaration; factory_usage PostgreSQL table/query/trigger declaration | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | factory_capacity PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; factory_reservations PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; factory_usage PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — Current named units/source consumers; retained normalized source evidence records each selector |
| 38; factory_operator_grants PostgreSQL table/query/trigger declaration | [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) | retained | factory_operator_grants PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 38-38; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 29-29; schema.rs drift test enforces equality |
| 39; factory_sponsorships PostgreSQL table/query/trigger declaration | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | factory_sponsorships PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 39-39; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 30-30; schema.rs drift test enforces equality |
| 40–41, 45, 137–138; factory_dispatch PostgreSQL table/query/trigger declaration; factory_dispatch_regs PostgreSQL table/query/trigger declaration; factory_assignments PostgreSQL table/query/trigger declaration; factory_dispatch_reg_immutable_update domain immutability trigger; factory_dispatch_reg_immutable_delete domain immutability trigger | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | factory_dispatch PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 44, 141–142; factory_run_views PostgreSQL table/query/trigger declaration; factory_run_view_immutable_update domain immutability trigger; factory_run_view_immutable_delete domain immutability trigger | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | factory_run_views PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; factory_run_view_immutable_update: enforce the owning domain slice’s immutable-state policy for its protected database row.; factory_run_view_immutable_delete: enforce the owning domain slice’s immutable-state policy for its protected database row. — Current named units/source consumers; retained normalized source evidence records each selector |
| 49–51, 143–146; issue_acceptance_decisions PostgreSQL table/query/trigger declaration; issue_acceptance_heads PostgreSQL table/query/trigger declaration; issue_acceptance_withdrawals PostgreSQL table/query/trigger declaration; issue_acceptance_decision_immutable_update domain immutability trigger; issue_acceptance_decision_immutable_delete domain immutability trigger; issue_acceptance_withdrawal_immutable_update domain immutability trigger; issue_acceptance_withdrawal_immutable_delete domain immutability trigger | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | issue_acceptance_decisions PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 52–53, 57; issue_controls PostgreSQL table/query/trigger declaration; intake_deliveries PostgreSQL table/query/trigger declaration; factory_readiness_sweeps PostgreSQL table/query/trigger declaration | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | issue_controls PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; intake_deliveries PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; factory_readiness_sweeps PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — Current named units/source consumers; retained normalized source evidence records each selector |
| 54; factory_publications PostgreSQL table/query/trigger declaration | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | factory_publications PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 54-54; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 45-45; schema.rs drift test enforces equality |
| 55; factory_check_assessments PostgreSQL table/query/trigger declaration | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | retained | factory_check_assessments PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 55-55; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 46-46; schema.rs drift test enforces equality |
| 56; factory_merges PostgreSQL table/query/trigger declaration | [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) | retained | factory_merges PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 56-56; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 47-47; schema.rs drift test enforces equality |
| 58; project_environment_grants PostgreSQL table/query/trigger declaration | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | project_environment_grants PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 58-58; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 49-49; schema.rs drift test enforces equality |
| 59–60, 147–148; project_requirement_decisions PostgreSQL table/query/trigger declaration; project_requirement_heads PostgreSQL table/query/trigger declaration; project_requirement_decision_immutable_update domain immutability trigger; project_requirement_decision_immutable_delete domain immutability trigger | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | project_requirement_decisions PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 61–62, 149–150; project_approval_decisions PostgreSQL table/query/trigger declaration; project_approval_heads PostgreSQL table/query/trigger declaration; project_approval_decision_immutable_update domain immutability trigger; project_approval_decision_immutable_delete domain immutability trigger | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | project_approval_decisions PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 64; identity_grants PostgreSQL table/query/trigger declaration | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | identity_grants PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 64-64; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 55-55; schema.rs drift test enforces equality |
| 66, 72, 124–133, 154, 257–260; identity_leases PostgreSQL table/query/trigger declaration; identity_executions PostgreSQL table/query/trigger declaration; function soda_guard_execution_identity; identity_execution_immutable domain immutability trigger; verifyImmutableExecutionIdentity | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | identity_leases PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 5 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 67, 151–152; identity_events PostgreSQL table/query/trigger declaration; identity_events_immutable_update domain immutability trigger; identity_events_immutable_delete domain immutability trigger | [I10](../../slices/identity-brokering.md#i10-identity-audit-history) | retained | identity_events PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; identity_events_immutable_update: enforce the owning domain slice’s immutable-state policy for its protected database row.; identity_events_immutable_delete: enforce the owning domain slice’s immutable-state policy for its protected database row. — Current named units/source consumers; retained normalized source evidence records each selector |
| 68; project_lifecycle_grants PostgreSQL table/query/trigger declaration | [P05](../../slices/projects.md#p05-project-startstop) | retained | project_lifecycle_grants PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 68-68; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 59-59; schema.rs drift test enforces equality |
| 69; project_maintenance PostgreSQL table/query/trigger declaration | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | project_maintenance PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal. — current Go schemaStatements literal internal/store/schema.go lines 69-69; byte-identical Rust mirror cmd/soda-identity/src/schema.rs lines 60-60; schema.rs drift test enforces equality |
| 70, 114–123, 153, 253–256; project_preparations PostgreSQL table/query/trigger declaration; function soda_guard_preparation_refs; preparation_refs_immutable domain immutability trigger; verifyImmutablePreparationRefs | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | project_preparations PostgreSQL table/query/trigger declaration: owns this domain schema row/field responsibility in canonical PostgreSQL DDL; Rust schema.rs mirrors this literal.; 4 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |

<a id="coverage-24ed293e59b9"></a>

## [internal/store/schema_test.go](../../../../../internal/store/schema_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–142; file scaffold; TestOpenCreatesOnlyTheCurrentSchema; legacyFingerprint; TestOpenRejectsOldSchemaWithoutMutation; TestOpenRejectsUnversionedDataWithoutMutation | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b26d414d0eb2"></a>

## [internal/store/store.go](../../../../../internal/store/store.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20, 23–26, 53–66, 75–106; file scaffold; ErrNotFound; Store; sameJSONDocument; Open; configureStore; open; Close | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 21–22; ErrCommandConflict | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current declaration duty: ErrCommandConflict — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 27–30, 42–52, 107–120; User; Session; UpsertUser | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: User; Current declaration duty: Session; Current declaration duty: UpsertUser — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–34, 129–159; Key; AddKey; RemoveKey; Keys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: Key; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 35–41, 160–175, 181–199, 220–223; Project; CreateProject; scanProject; projectColumns; ProjectByRepository | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: Project; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 67–74; OpenEncrypted | [I02](../../slices/identity-brokering.md#i02-encrypted-credential-custody) | retained | Current declaration duty: OpenEncrypted — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 121–128; RenameProfile | [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) | retained | Current declaration duty: RenameProfile — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 176–180; MarkReady | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: MarkReady — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 200–219; SpaceProjects | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current declaration duty: SpaceProjects — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 224–233; Join; MemberLogin | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: Join; Current declaration duty: MemberLogin — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f4cfe40ce089"></a>

## [internal/store/store_test.go](../../../../../internal/store/store_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11, 69–84; whole file; TestSetupSocketDSNParses | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Assertions TestSetupSocketDSNParses: host %q, want socket directory; declarations/fields: `TestSetupSocketDSNParses` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 12–68; TestMembersListingIsBounded, TestPersistenceAndMembership | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Bounded Project memberships and persistent own account assertions; declarations/fields: `TestMembersListingIsBounded`, `TestPersistenceAndMembership` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-698246430cc4"></a>

Former source `internal/store/identity.go`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-fc987cc9b8c7"></a>

Former source `internal/store/identity_events.go`; consult its pinned earlier Git source and the current coverage disposition.

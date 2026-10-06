# Server and state

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

[Library adoption](../library-adoption.md#finding-allocation) supersedes only
pending generic-engine preservation. Retain the Go domain/Store/API/client
owners and completed concern splits below. SQL01 removes PostgreSQL placeholder
translation; JSON02 and the transport findings consolidate bounded mechanics
around Go's existing standard libraries. Native authority, transaction/CAS,
per-dial peer admission and operation binding remain Soda policy. Historical
sizes and review evidence are unchanged.

## Current repository search predecessor

At `f7e9cf9d`, P01's actual source/caller census and B's independent challenge
establish that `Client.SearchOwnedRepositories` (`repositories.go:40–63`), its
three exclusive validators (`19–38`), and `RepositoryPageLimit` (`14`) have
only defining or exclusive test uses. Native SDK discovery owns current search.
Retire precisely those members and the whole exclusive
`internal/forgejo/repositories_test.go`; the desired tree excludes that test
file. Remove imports made unused by that scoped retirement.

Keep Go `internal/forgejo/repositories.go` as the single defining owner of
`RepositoryPageSize` (`12`) and `repositoryPart` (`65–67`): current API search
pagination (`internal/web/api/repositories.go:125,130`) and native observation
(`internal/forgejo/observe.go:172,188`) consume them. Retain those callers and
their real tests. See [P01's source and allocation record](../reviews/P01.md).
This is a supported target disposition, with no source removal or native
search execution performed.

## internal/forgejo/background.go

Observed size: 450 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/background.go` — Existing background service/client API methods and callback contract.
- `internal/forgejo/background_admission.go` — Serialized bootstrap/admission custody and peer verification.
- `internal/forgejo/background_transport.go` — Bounded POST transport and exact operation lookup validation.

Evidence: ServiceBackground at 35; NewServiceBackground at 49; admissionForCall at 299; bootstrap at 313; verifiedPeer at 430; checkBackgroundRecord at 241; checkBackgroundLookup at 251; post at 383.

The pending split remains about admission and callback custody. Keep standard
net/http and encoding/json under JSON02; do not wholesale replace the Soda
wrapper with the SDK client, whose bootstrap-only peer check does not supply
Soda's per-dial verification/rebind behavior. A bounded decoder must distinguish
oversized input from EOF after a valid prefix. Follow the library chapter's
caller-specific adapter and readiness gates instead of adding another codec.

## internal/forgejo/merge.go

Observed size: 459 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/merge.go` — Conditional merge submit/lookup/cancel adapter.
- `internal/forgejo/merge_observation.go` — Native actor, target and complete check evidence mapping.
- `internal/forgejo/merge_completion.go` — Receipt adoption and confirmed native issue outcome.

Evidence: NewMerger at 30; SubmitMerge at 247; LookupOp at 276; CancelOp at 296; ObserveMerge at 59; matchMergeTarget at 94; matchMergeChecks at 170; AdoptMerge at 374; ObserveCompletion at 402; matchMergeConfirmation at 427.

## internal/forgejo/snapshot.go

Observed size: 716 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/forgejo/snapshot.go` — Revision-bound snapshot aggregate and bracketed read.
- `internal/forgejo/snapshot_request.go` — Evidence-family identifiers and exact request validation.
- `internal/forgejo/snapshot_issue.go` — Issue, comment and dependency DTOs with existing validators.
- `internal/forgejo/snapshot_pull.go` — Pull, review, check and ref DTOs with existing validators.

Evidence: NativeSnapshot at 383; BracketedRead at 412; ValidateRequest at 124; ContentDigest at 204; validIssue at 516; validCommentPage at 538; validDependencyPage at 559; validPull at 577; validReviewPage at 596; validCheckSet at 614; validRefs at 635.

## Current dashboard identity predecessor and fixture allocation

At `f7e9cf9d`, `internal/store/identity.go` is not the broker runtime.
Its two save methods have only test callers; its two metadata readers have
one production consumer, `internal/web/api/factory_settings.go:468–485`.
That consumer must use the existing broker administration client under the
separate broker database contract; see [F04-F2](../reviews/F04.md#concrete-findings).
Installed database identities have not been inspected.

The desired tree excludes `internal/store/identity.go` and
`internal/store/identity_events.go` after that final production reader cutover.
Retain their necessary compatibility fixtures at the exact Go target
`internal/store/identity_fixture.go`: purpose-scoped `SeedIdentityConnection`
and `SeedIdentityGrant`, with one private defining owner for the existing
seal binding, transaction and credential-free event append. Existing
`ephemeral.go` and `corruption.go` provide the actual cross-package fixture
precedent. This ordinary Go source placement does not mean test-only
compilation or prove that fixture code is absent from a shipped binary.

Rebind `internal/store/identity_test.go`, `internal/store/grants_test.go`,
`internal/identity/client/broker_compat_test.go:212`,
`internal/factory/control/st15_demo_native_test.go:476` and
`internal/web/factory_settings_test.go:218–221` to that real fixture subject.
The corrected sponsorship test must exercise the actual broker metadata
caller. Retain canonical Go PostgreSQL DDL, its generated Rust mirror,
the current Go domain/client and grant-key custody, and the existing Rust
broker store. Do not remove identity tables or introduce another store,
process, schema authority or test framework. [I03](../reviews/I03.md) and
[I10](../reviews/I10.md#exact-fixture-successor-allocation-independently-challenged)
record the full caller census and B's independent source challenge.
This is a deferred correction plan; no source deletion or database change
has been performed or authorized.

SQL01 applies native PostgreSQL parameters to surviving Soda Store callers
without moving their schema or transaction authority. Fixture/probe SQLite
uses are a separate SQLITE01 consumer disposition: staged Forgejo fixture
imports can register a driver in compiled binaries, and the lifecycle probe
remains reachable. They are not evidence for SQLite product persistence or
permission to remove modernc globally before their callers are handled.

## internal/store/factory_assignments.go

Observed size: 504 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/store/factory_assignments.go` — Canonical assignment reads, queries and finishing.
- `internal/store/factory_dispatch_packet.go` — Atomic first dispatch packet and current admission/capacity check.
- `internal/store/factory_retry_packet.go` — Atomic retry packet under the same transaction owner.
- `internal/store/factory_dispatch_queue.go` — Queued controls and active run counts.

Evidence: Assignment at 51; IssueAssignments at 268; FinishAssignment at 401; RecordDispatchPacket at 51; checkAdmissionTx at 130; RecordRetryPacket at 298; QueuedControls at 457; ActiveRunCounts at 486.

## internal/store/factory_dispatch_test.go

Observed size: 459 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/store/factory_dispatch_test.go` — Dispatch packet, existing fixture and limit enforcement.
- `internal/store/factory_retry_packet_test.go` — Retry packet bounds and resource reholding.
- `internal/store/factory_dispatch_queue_test.go` — Reservation/usage/queue/current-count scenarios.

Evidence: dispatchStoreFixture at 16; TestRecordDispatchPacket at 88; TestRecordDispatchPacketEnforcesLimits at 203; TestRecordRetryPacketBoundsAttemptsAndFinishes at 138; TestRecordRetryPacketReholdsAndRefusesLimits at 259; TestReservationTransitions at 304; TestRunUsageFirstWriteWins at 349; TestQueuedControlsOldestFirst at 395.

## internal/store/factory_publications_test.go

Observed size: 435 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/store/factory_publications_test.go` — Record round-trip, CAS and selectable assignment/record cases.
- `internal/store/factory_publication_intent_test.go` — Immutable intent/receipt and withdrawal/authority registration ordering.

Evidence: TestRecordPublicationRoundTrip at 71; TestUpdatePublicationComparesAndSwaps at 93; TestPublishableAssignmentsSelectsReportedCompleted at 165; TestPublicationRegistrationPersistsImmutableIntent at 244; TestWithdrawalOrdersPublicationRegistrationAndKeepsReconciliation at 294; TestPublicationRegistrationRejectsAuthorityChangesAfterPreflight at 370.

## internal/tailnet/control.go

Observed size: 651 lines, including tests where embedded.

Disposition: retire native Go Control execution and its server-only tests at
host cutover. The existing protected policy/provider/enrollment/host-action
behavior belongs to `lib/host/src/tailnet/control/`, with its actual tests.
Retain Go request/response validators and read-only clients; this source is
not a new Go split owner in the target.

Evidence: NewControl at 40; NewProjectControl at 53; RunNative at 117; RoundTrip at 84; request at 130; checkCredential at 629; fetchNativeStatus at 264; populateHostPreferences at 296; observe at 356; HostAction at 427; executeSignin at 490; executeExitNode at 544; verifyHostActionOutcome at 591.

## internal/tailnet/control_test.go

Observed size: 495 lines, including tests where embedded.

Disposition: retire native Go Control execution and its server-only tests at
host cutover. The existing protected policy/provider/enrollment/host-action
behavior belongs to `lib/host/src/tailnet/control/`, with its actual tests.
Retain Go request/response validators and read-only clients; this source is
not a new Go split owner in the target.

Evidence: TestTailnetHostProjectionAndPassiveReads at 51; TestTailnetHostNodeKeyOptionalButStrict at 97; TestTailnetInitialLoginUsesBoundedUpNotReset at 228; TestTailnetCredentialCheckIsScopedBoundedAndNoRegistration at 255; TestTailnetOfflineExitNodeAndUnconfirmedClear at 392; TestTailnetBoundedOutputChild at 481; TestTailnetCommandOutputBounds at 488.

## internal/web/api/extension_terminal.go

Observed size: 411 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/extension_terminal.go` — Existing extension terminal route/session operations.
- `internal/web/api/extension_terminal_authority.go` — Original actor/account/membership and contribution binding.
- `internal/web/api/extension_terminal_stream.go` — Exact generation handshake, websocket attach and control pump.

Evidence: ExtensionHandler at 24; extensionTerminalSession at 243; extensionTerminalAccount at 87; extensionTerminalCurrent at 138; extensionTerminalAttach at 296; validNativeTerminalHandshake at 353; pumpExtensionControls at 387.

## internal/web/api/factory_settings.go

Observed size: 499 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/factory_settings.go` — Common native actor/repository/operator admission and errors.
- `internal/web/api/factory_status.go` — Current factory and preparation status projection.
- `internal/web/api/factory_policy.go` — Policy, capacity and operator grant routes.
- `internal/web/api/factory_sponsorship.go` — Environment/provider grant routes and broker sponsorship check.

Evidence: factoryRepository at 39; factoryOperator at 76; settingsCommandID at 86; apiFactoryStatus at 128; factoryPreparationStatus at 195; apiFactoryPolicy at 260; apiFactoryCapacity at 314; apiFactoryOperatorGrant at 349; apiFactoryEnvironmentGrant at 382; apiFactorySponsorship at 434; checkSponsorshipBroker at 467.

## internal/web/api/issue_acceptances.go

Observed size: 479 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/issue_acceptances.go` — Explicit acceptance and withdrawal routes.
- `internal/web/api/issue_acceptance_evidence.go` — Authorized native acceptance snapshots and refusal mapping.
- `internal/web/api/factory_issue_view.go` — Current issue/check/merge response projection.

Evidence: apiIssueAcceptances at 159; apiIssueWithdrawal at 247; ReadAcceptanceEvidence at 32; mapAcceptanceSnapshotError at 97; checksViewDTO at 326; mergeViewDTO at 358; apiFactoryIssue at 372.

## internal/web/api/spaces.go

Observed size: 404 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/api/spaces.go` — Inventory pagination, session checks and Spaces response entry.
- `internal/web/api/spaces_inspection.go` — Current per-project native/terminal/Tailnet/run projection.
- `internal/web/api/spaces_authority.go` — Existing authority and allowed-control projection.

Evidence: inspectSpaces at 284; apiSpaces at 361; parseSpacesCursor at 391; inspectSpaceNative at 78; inspectSpaceTerminals at 91; inspectSpaceTailnet at 118; inspectSpaceFactoryRuns at 186; resolveSpaceAuthority at 69; inspectSpaceAuthority at 223; inspectSpaceControl at 245.

## internal/web/factory_views_test.go

Observed size: 439 lines, including tests where embedded. Keep the same Go package, exported contracts and execution order; move existing concern definitions together.

- `internal/web/factory_views_test.go` — Current run status and Spaces inventory response cases.
- `internal/web/factory_output_fixture_test.go` — Existing output proxy and handshake fixtures.
- `internal/web/factory_output_stream_test.go` — Bound output/status/EOF and stale/unauthorized/input refusal cases.

Evidence: TestFactoryRunStatusPendingLiveAndExcerpt at 63; TestSpacesShowsFactoryRuns at 168; factoryOutputProxyFixture at 206; dialFactoryOutput at 252; factoryOutputHandshakeFor at 290; TestFactoryOutputStreamDeliversStatusOutputAndEOF at 294; TestFactoryOutputStreamRefusesStaleHandshake at 377; TestFactoryOutputStreamRequiresWrite at 419.

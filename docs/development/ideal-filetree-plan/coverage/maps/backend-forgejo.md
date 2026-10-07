# Backend forgejo

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-9e5fb8076312"></a>
<a id="internalforgejobackgroundgo-1"></a>

## [internal/forgejo/background.go](../../../../../internal/forgejo/background.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; file scaffold | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 19–44, 84–166; ServiceBackground; backgroundBootstrap; NewServiceBackground; SubmitOperation; GetOperation; CancelOperation; PublishPushEnv | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | The SDK BackgroundClient methods are implemented directly by ServiceBackground; snapshot readers and publishers continue sharing the same lazy admission. The redundant forwarding adapter is removed. — Current SDK interface and production call sites inspected |
| 45–82; ReadNativeRevision; ReadSnapshot | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | ServiceBackground remains the typed SDK client for revision/snapshot reads, preserving per-dial peer verification and same-open bounded credential handling. — Current source and caller paths inspected |

<a id="coverage-d7a9a3149384"></a>

## [internal/forgejo/background_admission.go](../../../../../internal/forgejo/background_admission.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–165; file scaffold; maxBackgroundBodyBytes; validBackgroundAdmission; validBackgroundOperationID; backgroundSecret; checkBackgroundRecord; admissionForCall; bootstrap; verifiedPeer | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b3264fdcee80"></a>

## [internal/forgejo/background_test.go](../../../../../internal/forgejo/background_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–319; file scaffold; scriptedBackgroundServer; current; handler; serveOp; TestServiceBackgroundVerifiesPeerBeforeSendingCredential; TestServiceBackgroundRejectsForeignNestedRecord; shortSocketPath; serveScriptedBackground; TestServiceBackgroundSharesOneAdmission; TestServiceBackgroundRebindsAfterRevocation; TestServiceBackgroundMapsDispatchStatuses | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fc3a0ed46e1c"></a>

## [internal/forgejo/background_transport.go](../../../../../internal/forgejo/background_transport.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–121; file scaffold; maxBackgroundStatusBody; checkBackgroundLookup; call; post; readStatusBody | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7757b0beb8a1"></a>

## [internal/forgejo/checks.go](../../../../../internal/forgejo/checks.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–173; file scaffold; CheckAssessor; NewCheckAssessor; checkActor; ObserveChecks; checkReadError; matchCheckTarget | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-da866174645d"></a>

## [internal/forgejo/checks_test.go](../../../../../internal/forgejo/checks_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–141; file scaffold; checkTargetForObserve; checkSnapshotFixture; TestMatchCheckTargetLatestWins; TestMatchCheckTargetStaleTipsPassThrough; TestMatchCheckTargetHidden; TestMatchCheckTargetRefusals; TestCheckReadErrorMapsBusy; TestObserveChecksRejectsInvalidTarget | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-27d6a3048980"></a>

## [internal/forgejo/client.go](../../../../../internal/forgejo/client.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–112; whole file; Client; User; Repository; RepositoryPermissions; New; transportError; decodeResponse; Client.request; Client.Current | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 113–115; Client.RevokeCurrentToken | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | unresolved | Historical Go bootstrap-token revocation after durable operator configuration; current Rust soda-setup owns the live operation; declarations/fields: `Client.RevokeCurrentToken` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-07949652966e"></a>

## [internal/forgejo/client_test.go](../../../../../internal/forgejo/client_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; file scaffold; TestCurrent; TestNoCredentialRedirect | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestCurrent; Current declaration duty: TestNoCredentialRedirect — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dae7780022bd"></a>

## [internal/forgejo/errors.go](../../../../../internal/forgejo/errors.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; file scaffold; HTTPError; Error | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; Current declaration duty: HTTPError; Current declaration duty: Error — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c54cbd1c2364"></a>

## [internal/forgejo/errors_test.go](../../../../../internal/forgejo/errors_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–69; file scaffold; TestResponseBoundary; TestNativeStatusIsTypedAndSanitized; TestTransportCancellation | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-75ed324c2b45"></a>
<a id="internalforgejomergego-1"></a>

## [internal/forgejo/merge.go](../../../../../internal/forgejo/merge.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; file scaffold | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 21–219; Merger; NewMerger; checkActor; ObserveMerge; mergeReadError; mergeIntent; SubmitMerge; LookupOp; CancelOp; mergeCallError; mergeOperationOutcome | [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) | retained | Current declaration duty: Merger; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-47942fe788eb"></a>

## [internal/forgejo/merge_completion.go](../../../../../internal/forgejo/merge_completion.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–117; file scaffold; mergeReceipt; AdoptMerge; ObserveCompletion; matchMergeConfirmation | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-745684ba8493"></a>

## [internal/forgejo/merge_observation.go](../../../../../internal/forgejo/merge_observation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–144; file scaffold; matchMergeTarget; matchMergeChecks | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; Current declaration duty: matchMergeTarget; Current declaration duty: matchMergeChecks — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6d31a60c6341"></a>

## [internal/forgejo/merge_test.go](../../../../../internal/forgejo/merge_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–275; file scaffold; mergeTestWork; mergeTestOutcome; TestMergerReceipt; TestMergerIntentFastForwardOnly; mergeTestSnapshots; TestMergerMatchTarget; TestMergerMatchConfirmation; TestMergerLookupAfterCredentialLoss | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3130a4345f2f"></a>

## [internal/forgejo/observe.go](../../../../../internal/forgejo/observe.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–71, 120–213; whole file; ObservationIssuePageSize; ServiceObserver; NewServiceObserver; ServiceObserver.Credential; ServiceObserver.SnapshotReader; ServiceObserver.ensureClient; serviceSnapshotReader; serviceSnapshotReader.ReadNativeRevision; serviceSnapshotReader.ReadSnapshot; ServiceObserver.ListIssuesPage; Client.RepositoryByID; ListedIssue; Client.ListRepositoryIssuesPage | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 14 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 69–119; ServiceObserver.sharedBackgroundLocked, ServiceObserver.ShareBackground, ServiceObserver.ensureClient, ServiceObserver.ensureActor | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | The observer caches or shares the ServiceBackground directly as the SDK BackgroundClient, preserving lazy bootstrap and one reader/publisher admission. — Current shared transport callers inspected |

<a id="coverage-d3e08163e098"></a>

## [internal/forgejo/observe_test.go](../../../../../internal/forgejo/observe_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–198; file scaffold; observationCredential; observationREST; TestServiceObserverListsIssuesOldestFirst; TestServiceObserverListBounds; fakeBackgroundServer; handler; serveBackgroundSocket; TestServiceObserverBootstrapsOnce; TestServiceObserverBootstrapFailure | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e3cb6caf989e"></a>

## [internal/forgejo/own_keys.go](../../../../../internal/forgejo/own_keys.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; file scaffold; OwnPublicKey; validOwnPublicKey; OwnPublicKeys | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-11a8549e6c95"></a>

## [internal/forgejo/ownership.go](../../../../../internal/forgejo/ownership.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–28; file scaffold; OrganizationOwner | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; Current declaration duty: OrganizationOwner — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-42302a38efd2"></a>

## [internal/forgejo/publish.go](../../../../../internal/forgejo/publish.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–345; file scaffold; Publisher; NewPublisher; ObservePublication; SubmitPublish; PushBranch; SubmitPRCreate; LookupOp; CancelOp; AdoptBranch; AdoptPRCreation; publicationOutcomeMatches; publisherConfig; checkActor; cachedLogin; protectedSecrets; mapValidationError; flattenOutcome; flattenError | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 19 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6988678fcfe6"></a>

## [internal/forgejo/publish/candidate_validation.go](../../../../../internal/forgejo/publish/candidate_validation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–90; file scaffold; ValidatedRepo; PrepareValidated; verifyCandidateBundle; validationVerdict; Close | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3715f19b3d9f"></a>

## [internal/forgejo/publish/credentials.go](../../../../../internal/forgejo/publish/credentials.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; file scaffold; checkCredentials; credentialScanResult; scanCredentials; credentialObjectHeader; credentialObjectSizeAllowed; scanCredentialObject | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f7761249f452"></a>

## [internal/forgejo/publish/credentials_test.go](../../../../../internal/forgejo/publish/credentials_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–116; file scaffold; candidateGit; candidateBundle; TestPublisherRejectsCredentialHistoryBeforeAuthorization; TestPublisherAcceptsCleanCandidateWithCredentialDenylist; TestPublisherRejectsCompressedOversizedDecodedObject; TestCredentialScanRejectsMismatchedContentIdentity; TestCredentialScanRejectsDecodedAggregateLimit | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-53053d2a3f30"></a>

## [internal/forgejo/publish/git.go](../../../../../internal/forgejo/publish/git.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–64; file scaffold; repository; initialize; run; command; limitedOutput; Write | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3a8182478af6"></a>

## [internal/forgejo/publish/operation_git_test.go](../../../../../internal/forgejo/publish/operation_git_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–150; file scaffold; TestPushBranchMovesExactlyItsTarget; TestObserveTipParsesExactAdvertisement; TestObserveForPublishRefusesBeforeAnyCall; servePublicationGit | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f143f20babd3"></a>

## [internal/forgejo/publish/operation_observation.go](../../../../../internal/forgejo/publish/operation_observation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–124; file scaffold; ObserveForPublish; validPublishRef; bracketRevision; mapRevisionError; observeTips; observeTip | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bd63d08e7edc"></a>

## [internal/forgejo/publish/operation_push.go](../../../../../internal/forgejo/publish/operation_push.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–86; file scaffold; runWithEnv; PushBranch; reconcilePush | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-94f681cb54a1"></a>

## [internal/forgejo/publish/operation_receipts.go](../../../../../internal/forgejo/publish/operation_receipts.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–114; file scaffold; PublishReceipt; DecodePublishReceipt; PRCreateReceipt; DecodePRCreateReceipt; OperationNotAfter | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0ee2c51a20d6"></a>

## [internal/forgejo/publish/operation_receipts_test.go](../../../../../internal/forgejo/publish/operation_receipts_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; file scaffold; TestDecodePublishReceiptRefusesLookalikes; TestDecodePRCreateReceiptRefusesLookalikes; TestReceiptRejectsTrailingDocument | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-aad5e21707ed"></a>

## [internal/forgejo/publish/publish.go](../../../../../internal/forgejo/publish/publish.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–167; file scaffold; Config; Request; Validate; validateRemote; ValidateCandidate; prepare; privateInputs; protected; validateCandidate; localFixture | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6fa02b532af7"></a>

## [internal/forgejo/publish/publish_test.go](../../../../../internal/forgejo/publish/publish_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; file scaffold; candidateFixture; TestPublisherValidatesCandidate; TestPublisherRejectsMismatchedBundle | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a45a5490b11a"></a>

## [internal/forgejo/publish/review_role_test.go](../../../../../internal/forgejo/publish/review_role_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–20; file scaffold; TestST10PublisherRejectsReviewerBundle | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestST10PublisherRejectsReviewerBundle — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4f88991f9166"></a>

## [internal/forgejo/publish/source.go](../../../../../internal/forgejo/publish/source.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7; file scaffold; 8–23 declaration sourceRepository | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | `Config.Source` and `Config.BranchRevision` were removed in `8dbc0989` after whole-workspace reference closure. `sourceRepository` remains the repository source used by `ObserveForPublish`; three real Git/loopback publisher tests cover the retained observation path. |

<a id="coverage-dc86ad39fbe5"></a>

## [internal/forgejo/publish_test.go](../../../../../internal/forgejo/publish_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–238; file scaffold; publishWorkFixture; publishTestPublisher; TestPublisherResolvesPerRepositoryRemote; TestPublisherSubmitsThroughSharedTransport; TestPublisherRefusesInvalidBundle; TestPublisherAdoptsExactReceipts; TestPublisherCanObserveWithoutPreviousObservation; TestPublisherAdoptionRejectsForeignOperationMetadata | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bc43f54bfa20"></a>

## [internal/forgejo/repositories.go](../../../../../internal/forgejo/repositories.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; file scaffold; RepositoryPageSize; repositoryPart | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; Current declaration duty: RepositoryPageSize; Current declaration duty: repositoryPart — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-81c9ae4c456c"></a>

## [internal/forgejo/review.go](../../../../../internal/forgejo/review.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–290; file scaffold; Reviewer; NewReviewer; checkActor; ObserveReview; reviewReadError; matchReviewTarget; reviewIntent; SubmitReview; LookupOp; CancelOp; reviewCallError; reviewOperationOutcome; reviewReceipt; AdoptReview | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-109ffe706a88"></a>

## [internal/forgejo/review_test.go](../../../../../internal/forgejo/review_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–197; file scaffold; reviewTestWork; reviewTestOutcome; TestReviewerReceipt; TestReviewerIntentBodyOnly; TestReviewerLookupAfterCredentialLoss; TestReviewerActorMismatch; TestReviewerExactNativeTarget; TestReviewerLostSubmitReply | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ab7ca91464f5"></a>
<a id="internalforgejosnapshotgo-1"></a>

## [internal/forgejo/snapshot.go](../../../../../internal/forgejo/snapshot.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; file scaffold | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 26–214; NativeSnapshot; SnapshotReader; BracketedRead; ValidateSnapshot; HasHiddenEvidence | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Current declaration duty: NativeSnapshot; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-301a967e8bc2"></a>

## [internal/forgejo/snapshot_issue.go](../../../../../internal/forgejo/snapshot_issue.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–153; file scaffold; LifecycleEvent; CreationProvenance; IssueEvidence; CommentEvidence; CommentPage; DependencyEvidence; DependencyPage; validIssue; validCommentPage; validDependencyPage | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bac834cfe1cb"></a>

## [internal/forgejo/snapshot_pull.go](../../../../../internal/forgejo/snapshot_pull.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–168; file scaffold; PullEvidence; ReviewEvidence; ReviewPage; CheckEvidence; CheckSet; RefEvidence; validPull; validReviewPage; validCheckSet; validRefs | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4c851bafeaf0"></a>

## [internal/forgejo/snapshot_request.go](../../../../../internal/forgejo/snapshot_request.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–189; file scaffold; SnapshotFamily; validFamily; SnapshotRequest; decimalID; fullOID; fullBranchRef; ValidateRequest; ContentDigest | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-653e7ba714cd"></a>

## [internal/forgejo/snapshot_test.go](../../../../../internal/forgejo/snapshot_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–187; file scaffold; fakeSnapshotReader; ReadNativeRevision; ReadSnapshot; snapshotTestRequest; snapshotTestSnapshot; idleRevision; TestBracketedReadAcceptsEqualIdleBracket; TestBracketedReadRefusesInterveningChange; TestBracketedReadRefusesMissingPage; TestBracketedReadReturnsHiddenRecordsForAuthorization; TestBracketedReadRefusesDigestMismatch; TestValidateRequestBoundsFamilies; TestContentDigestIsStable | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c73dda68dae6"></a>

## [internal/forgejo/snapshot_transport.go](../../../../../internal/forgejo/snapshot_transport.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; file scaffold; BackgroundSnapshotReader; ReadNativeRevision; ReadSnapshot; decodeSnapshotAnswer | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b982934975cb"></a>

## [internal/forgejo/snapshot_transport_test.go](../../../../../internal/forgejo/snapshot_transport_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–177; file scaffold; fakeBackgroundClient; ReadNativeRevision; ReadSnapshot; SubmitOperation; GetOperation; CancelOperation; wireSnapshotAnswer; TestBackgroundSnapshotReaderMapsWireSnapshot; TestBackgroundSnapshotReaderRefusesUnusableTransport; TestBracketedReadBindsRevisionFromBracket; TestValidateRequestRequiresFamilySelectors | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

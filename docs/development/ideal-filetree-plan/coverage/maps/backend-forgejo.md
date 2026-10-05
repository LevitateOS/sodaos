# Backend forgejo

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-9e5fb8076312"></a>

<a id="internalforgejobackgroundgo-1"></a>

## [internal/forgejo/background.go](../../../../../internal/forgejo/background.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 21–24 | Declared identifiers/bounds maxBackgroundBodyBytes for Background service admission; declarations/fields: `maxBackgroundBodyBytes` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 25–34 | Declared identifiers/bounds maxBackgroundStatusBody for Background service admission; declarations/fields: `maxBackgroundStatusBody` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 35–48 | Record, DTO or interface contract ServiceBackground for Background service admission; declarations/fields: `ServiceBackground` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 49–54 | NewServiceBackground — Background service admission; declarations/fields: `NewServiceBackground` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 55–94, 184–191 | Authoritative native snapshot/revision operation via shared admission; declarations/fields: `ServiceBackground.ReadNativeRevision`, `ServiceBackground.ReadSnapshot`, `backgroundClientAdapter.ReadNativeRevision`, `backgroundClientAdapter.ReadSnapshot` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 95–121 | ServiceBackground.SubmitOperation — Background service admission; declarations/fields: `ServiceBackground.SubmitOperation` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 122–139 | ServiceBackground.GetOperation — Background service admission; declarations/fields: `ServiceBackground.GetOperation` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 140–159 | ServiceBackground.CancelOperation — Background service admission; declarations/fields: `ServiceBackground.CancelOperation` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 160–179 | ServiceBackground.PublishPushEnv — Background service admission; declarations/fields: `ServiceBackground.PublishPushEnv` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 180–183 | Record, DTO or interface contract backgroundClientAdapter for Background service admission; declarations/fields: `backgroundClientAdapter` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 192–195 | backgroundClientAdapter.SubmitOperation — Background service admission; declarations/fields: `backgroundClientAdapter.SubmitOperation` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 196–199 | backgroundClientAdapter.GetOperation — Background service admission; declarations/fields: `backgroundClientAdapter.GetOperation` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 200–203 | backgroundClientAdapter.CancelOperation — Background service admission; declarations/fields: `backgroundClientAdapter.CancelOperation` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 204–216 | validBackgroundAdmission — Background service admission; declarations/fields: `validBackgroundAdmission` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 217–229 | validBackgroundOperationID — Background service admission; declarations/fields: `validBackgroundOperationID` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 230–240 | backgroundSecret — Background service admission; declarations/fields: `backgroundSecret` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 241–250 | checkBackgroundRecord — Background service admission; declarations/fields: `checkBackgroundRecord` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 251–272 | checkBackgroundLookup — Background service admission; declarations/fields: `checkBackgroundLookup` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 273–298 | ServiceBackground.call — Background service admission; declarations/fields: `ServiceBackground.call` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 299–312 | ServiceBackground.admissionForCall — Background service admission; declarations/fields: `ServiceBackground.admissionForCall` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 313–382 | ServiceBackground.bootstrap — Background service admission; declarations/fields: `ServiceBackground.bootstrap` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 383–429 | ServiceBackground.post — Background service admission; declarations/fields: `ServiceBackground.post` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 430–442 | ServiceBackground.verifiedPeer — Background service admission; declarations/fields: `ServiceBackground.verifiedPeer` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 443–450 | readStatusBody — Background service admission; declarations/fields: `readStatusBody` |

<a id="coverage-27d6a3048980"></a>

## [internal/forgejo/client.go](../../../../../internal/forgejo/client.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state. Historical-purpose attribution does not claim this Go helper is currently called; its unit lifecycle is unknown.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 15–18 | Record, DTO or interface contract Client for Authoritative native reads; declarations/fields: `Client` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 19–24 | Record, DTO or interface contract User for Authoritative native reads; declarations/fields: `User` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 25–33 | Record, DTO or interface contract Repository for Authoritative native reads; declarations/fields: `Repository` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 34–39 | Record, DTO or interface contract RepositoryPermissions for Authoritative native reads; declarations/fields: `RepositoryPermissions` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 40–43 | New — Authoritative native reads; declarations/fields: `New` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 44–51 | transportError — Authoritative native reads; declarations/fields: `transportError` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 52–75 | decodeResponse — Authoritative native reads; declarations/fields: `decodeResponse` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 76–103 | Client.request — Authoritative native reads; declarations/fields: `Client.request` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 104–112 | Client.Current — Authoritative native reads; declarations/fields: `Client.Current` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / unknown | 113–115 | Historical Go bootstrap-token revocation after durable operator configuration; current Rust soda-setup owns the live operation; declarations/fields: `Client.RevokeCurrentToken` |

<a id="coverage-75ed324c2b45"></a>

<a id="internalforgejomergego-1"></a>

## [internal/forgejo/merge.go](../../../../../internal/forgejo/merge.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 1–23 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 24–29 | Record, DTO or interface contract Merger for Conditional native merge; declarations/fields: `Merger` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 30–33 | NewMerger — Conditional native merge; declarations/fields: `NewMerger` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 34–58 | Merger.checkActor — Conditional native merge; declarations/fields: `Merger.checkActor` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 59–83 | Merger.ObserveMerge — Conditional native merge; declarations/fields: `Merger.ObserveMerge` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 84–93 | mergeReadError — Conditional native merge; declarations/fields: `mergeReadError` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 94–134 | Native merge target identity, revision, branch and review evidence integrity; declarations/fields: `matchMergeTarget` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 135–155 | Exact-head independent official approval/changes-requested eligibility decision; declarations/fields: `matchMergeTarget` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 156–163 | Return verified native merge observation with mapped G06 check evidence; declarations/fields: `matchMergeTarget` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 164–169 | Exact-candidate approval/refusal policy within native merge matcher; native identity parsing remains G07 in partition below; declarations/fields: `matchMergeTarget` |
| [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) / active | 170–228 | Map latest native CI evidence inside merge bracket; shares G06 semantics with CheckAssessor; declarations/fields: `matchMergeChecks` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 229–246 | mergeIntent — Conditional native merge; declarations/fields: `mergeIntent` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 247–275 | Merger.SubmitMerge — Conditional native merge; declarations/fields: `Merger.SubmitMerge` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 276–295 | Merger.LookupOp — Conditional native merge; declarations/fields: `Merger.LookupOp` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 296–310 | Merger.CancelOp — Conditional native merge; declarations/fields: `Merger.CancelOp` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 311–330 | mergeCallError — Conditional native merge; declarations/fields: `mergeCallError` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 331–361 | mergeOperationOutcome — Conditional native merge; declarations/fields: `mergeOperationOutcome` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 362–373 | Record, DTO or interface contract mergeReceipt for Conditional native merge; declarations/fields: `mergeReceipt` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 374–401 | Merger.AdoptMerge — Conditional native merge; declarations/fields: `Merger.AdoptMerge` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 402–426 | Merger.ObserveCompletion — Conditional native merge; declarations/fields: `Merger.ObserveCompletion` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 427–459 | matchMergeConfirmation — Conditional native merge; declarations/fields: `matchMergeConfirmation` |

<a id="coverage-3130a4345f2f"></a>

## [internal/forgejo/observe.go](../../../../../internal/forgejo/observe.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 16–23 | Declared identifiers/bounds ObservationIssuePageSize for Authoritative native reads; declarations/fields: `ObservationIssuePageSize` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 24–38 | Record, DTO or interface contract ServiceObserver for Authoritative native reads; declarations/fields: `ServiceObserver` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 39–44 | NewServiceObserver — Authoritative native reads; declarations/fields: `NewServiceObserver` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 45–51 | ServiceObserver.Credential — Authoritative native reads; declarations/fields: `ServiceObserver.Credential` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 52–55 | ServiceObserver.SnapshotReader — Authoritative native reads; declarations/fields: `ServiceObserver.SnapshotReader` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 56–71 | ServiceObserver.ensureClient — Authoritative native reads; declarations/fields: `ServiceObserver.ensureClient` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 72–119 | Shared native service admission and actor credential binding; declarations/fields: `ServiceObserver.sharedBackgroundLocked`, `ServiceObserver.ShareBackground`, `ServiceObserver.ensureActor` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 120–123 | Record, DTO or interface contract serviceSnapshotReader for Authoritative native reads; declarations/fields: `serviceSnapshotReader` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 124–131 | serviceSnapshotReader.ReadNativeRevision — Authoritative native reads; declarations/fields: `serviceSnapshotReader.ReadNativeRevision` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 132–147 | serviceSnapshotReader.ReadSnapshot — Authoritative native reads; declarations/fields: `serviceSnapshotReader.ReadSnapshot` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 148–163 | ServiceObserver.ListIssuesPage — Authoritative native reads; declarations/fields: `ServiceObserver.ListIssuesPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 164–179 | Client.RepositoryByID — Authoritative native reads; declarations/fields: `Client.RepositoryByID` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 180–186 | Record, DTO or interface contract ListedIssue for Authoritative native reads; declarations/fields: `ListedIssue` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 187–213 | Client.ListRepositoryIssuesPage — Authoritative native reads; declarations/fields: `Client.ListRepositoryIssuesPage` |

<a id="coverage-ab7ca91464f5"></a>

<a id="internalforgejosnapshotgo-1"></a>

## [internal/forgejo/snapshot.go](../../../../../internal/forgejo/snapshot.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 1–29 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 30–31 | Record, DTO or interface contract SnapshotFamily for Authoritative native reads; declarations/fields: `SnapshotFamily` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 32–41, 52–73 | Declared identifiers/bounds (declaration group) for Authoritative native reads; declarations/fields: `(declaration group)` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 42–51 | validFamily — Authoritative native reads; declarations/fields: `validFamily` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 74–85 | Record, DTO or interface contract SnapshotRequest for Authoritative native reads; declarations/fields: `SnapshotRequest` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 86–98 | decimalID — Authoritative native reads; declarations/fields: `decimalID` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 99–111 | fullOID — Authoritative native reads; declarations/fields: `fullOID` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 112–123 | fullBranchRef — Authoritative native reads; declarations/fields: `fullBranchRef` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 124–203 | ValidateRequest — Authoritative native reads; declarations/fields: `ValidateRequest` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 204–209 | ContentDigest — Authoritative native reads; declarations/fields: `ContentDigest` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 210–220 | Record, DTO or interface contract LifecycleEvent for Authoritative native reads; declarations/fields: `LifecycleEvent` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 221–229 | Record, DTO or interface contract CreationProvenance for Authoritative native reads; declarations/fields: `CreationProvenance` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 230–252 | Record, DTO or interface contract IssueEvidence for Authoritative native reads; declarations/fields: `IssueEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 253–270 | Record, DTO or interface contract CommentEvidence for Authoritative native reads; declarations/fields: `CommentEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 271–280 | Record, DTO or interface contract CommentPage for Authoritative native reads; declarations/fields: `CommentPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 281–292 | Record, DTO or interface contract DependencyEvidence for Authoritative native reads; declarations/fields: `DependencyEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 293–300 | Record, DTO or interface contract DependencyPage for Authoritative native reads; declarations/fields: `DependencyPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 301–322 | Record, DTO or interface contract PullEvidence for Authoritative native reads; declarations/fields: `PullEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 323–340 | Record, DTO or interface contract ReviewEvidence for Authoritative native reads; declarations/fields: `ReviewEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 341–348 | Record, DTO or interface contract ReviewPage for Authoritative native reads; declarations/fields: `ReviewPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 349–363 | Record, DTO or interface contract CheckEvidence for Authoritative native reads; declarations/fields: `CheckEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 364–371 | Record, DTO or interface contract CheckSet for Authoritative native reads; declarations/fields: `CheckSet` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 372–382 | Record, DTO or interface contract RefEvidence for Authoritative native reads; declarations/fields: `RefEvidence` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 383–399 | Record, DTO or interface contract NativeSnapshot for Authoritative native reads; declarations/fields: `NativeSnapshot` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 400–411 | Record, DTO or interface contract SnapshotReader for Authoritative native reads; declarations/fields: `SnapshotReader` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 412–453 | BracketedRead — Authoritative native reads; declarations/fields: `BracketedRead` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 454–515 | ValidateSnapshot — Authoritative native reads; declarations/fields: `ValidateSnapshot` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 516–537 | validIssue — Authoritative native reads; declarations/fields: `validIssue` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 538–558 | validCommentPage — Authoritative native reads; declarations/fields: `validCommentPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 559–576 | validDependencyPage — Authoritative native reads; declarations/fields: `validDependencyPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 577–595 | validPull — Authoritative native reads; declarations/fields: `validPull` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 596–613 | validReviewPage — Authoritative native reads; declarations/fields: `validReviewPage` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 614–634 | validCheckSet — Authoritative native reads; declarations/fields: `validCheckSet` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 635–663 | validRefs — Authoritative native reads; declarations/fields: `validRefs` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 664–716 | HasHiddenEvidence — Authoritative native reads; declarations/fields: `HasHiddenEvidence` |


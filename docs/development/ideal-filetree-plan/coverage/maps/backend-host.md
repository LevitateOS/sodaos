# Backend host

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-2a842406ecbb"></a>

## [internal/host/client.go](../../../../../internal/host/client.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 17–21 | Record, DTO or interface contract (declaration group) for Private IPC and service lifetime; declarations/fields: `(declaration group)` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 22–25 | nativeHTTPError.Error — Private IPC and service lifetime; declarations/fields: `nativeHTTPError.Error` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 26–31 | NewClient — Private IPC and service lifetime; declarations/fields: `NewClient` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 32–41 | decodeNativeResponseLimit — Private IPC and service lifetime; declarations/fields: `decodeNativeResponseLimit` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 42–55 | readNativeResponseLimit — Private IPC and service lifetime; declarations/fields: `readNativeResponseLimit` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 56–59 | Client.call — Private IPC and service lifetime; declarations/fields: `Client.call` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 60–77 | Client.callLimit — Private IPC and service lifetime; declarations/fields: `Client.callLimit` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 78–86 | Project creation native operation; declarations/fields: `Client.Create` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 87–95, 116–119 | Observed Project runtime identity/address; declarations/fields: `Client.Inspect`, `validAddress` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 96–106 | Human Project account creation native operation; declarations/fields: `Client.Join` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 107–115 | Own native development connection observation; declarations/fields: `Client.Connection` |

<a id="coverage-0889ccbfcca1"></a>

## [internal/host/factory_client.go](../../../../../internal/host/factory_client.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 13–17 | Declared identifiers/bounds ErrRunNotFound for Run lifecycle and intervention; declarations/fields: `ErrRunNotFound` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 18–21 | Declared identifiers/bounds ErrRunStale for Run lifecycle and intervention; declarations/fields: `ErrRunStale` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 22–23 | Declared identifiers/bounds factoryResponseLimit for Run lifecycle and intervention; declarations/fields: `factoryResponseLimit` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 24–31 | factoryNotFound — Run lifecycle and intervention; declarations/fields: `factoryNotFound` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 32–47, 89–96 | Bounded read-only factory output protocol; declarations/fields: `factoryOutputError`, `Client.FactoryOutput` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 48–51 | factoryStateConfirmed — Run lifecycle and intervention; declarations/fields: `factoryStateConfirmed` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 52–60, 70–78 | Native factory assignment launch or pinned harness observation; declarations/fields: `Client.FactoryLaunch`, `Client.FactoryHarness` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 61–69 | Client.FactoryInspect — Run lifecycle and intervention; declarations/fields: `Client.FactoryInspect` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 79–88 | Client.FactoryStop — Run lifecycle and intervention; declarations/fields: `Client.FactoryStop` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 97–130 | Exact factory candidate export protocol; declarations/fields: `factoryExportResponseLimit`, `Client.FactoryExport`, `factoryExportError` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 131–138 | Client.FactoryTakeover — Run lifecycle and intervention; declarations/fields: `Client.FactoryTakeover` |

<a id="coverage-ff42bb734720"></a>

## [internal/host/identity.go](../../../../../internal/host/identity.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 1–8 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 9–15 | Client.IdentityLaunch — Provider execution integration; declarations/fields: `Client.IdentityLaunch` |
| [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) / active | 16–19, 24–29 | Attested native execution binding; declarations/fields: `Client.IdentityValidate`, `Client.IdentityStart` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 20–23, 30–34 | Native credential retirement acknowledgement; declarations/fields: `Client.IdentityStop`, `Client.IdentityFinish` |

<a id="coverage-2b1ef0c1a495"></a>

## [internal/host/prepare.go](../../../../../internal/host/prepare.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 12–13 | Declared identifiers/bounds prepareResponseLimit for Checkout allocation and preparation; declarations/fields: `prepareResponseLimit` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 14–19 | preparationIdentityConfirmed — Checkout allocation and preparation; declarations/fields: `preparationIdentityConfirmed` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 20–28 | Client.Prepare — Checkout allocation and preparation; declarations/fields: `Client.Prepare` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 29–38 | Client.InspectPreparation — Checkout allocation and preparation; declarations/fields: `Client.InspectPreparation` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 39–48 | Client.StopPreparation — Checkout allocation and preparation; declarations/fields: `Client.StopPreparation` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 49–56 | Native preparation admission hold synchronization; declarations/fields: `Client.HoldPreparation` |

<a id="coverage-f4c2e2fdefc5"></a>

<a id="internalhostpublishoperationgo-1"></a>

## [internal/host/publish/operation.go](../../../../../internal/host/publish/operation.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 1–24 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 25–39 | Record, DTO or interface contract BackgroundOperations for Candidate publication and PR creation; declarations/fields: `BackgroundOperations` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 40–41 | Record, DTO or interface contract Refusal for Candidate publication and PR creation; declarations/fields: `Refusal` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 42–46 | Refusal.Error — Candidate publication and PR creation; declarations/fields: `Refusal.Error` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 47–48 | Record, DTO or interface contract Wait for Candidate publication and PR creation; declarations/fields: `Wait` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 49–52 | Wait.Error — Candidate publication and PR creation; declarations/fields: `Wait.Error` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 53–59 | Declared identifiers/bounds operationLifetime for Candidate publication and PR creation; declarations/fields: `operationLifetime` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 60–91 | Config.ObserveForPublish — Candidate publication and PR creation; declarations/fields: `Config.ObserveForPublish` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 92–99 | validPublishRef — Candidate publication and PR creation; declarations/fields: `validPublishRef` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 100–113 | bracketRevision — Candidate publication and PR creation; declarations/fields: `bracketRevision` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 114–125 | mapRevisionError — Candidate publication and PR creation; declarations/fields: `mapRevisionError` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 126–145 | Config.observeTips — Candidate publication and PR creation; declarations/fields: `Config.observeTips` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 146–172 | observeTip — Candidate publication and PR creation; declarations/fields: `observeTip` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 173–193 | Record, DTO or interface contract PublishOpIntent for Candidate publication and PR creation; declarations/fields: `PublishOpIntent` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 194–208 | Config.SubmitPublish — Candidate publication and PR creation; declarations/fields: `Config.SubmitPublish` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 209–241 | publishPayload — Candidate publication and PR creation; declarations/fields: `publishPayload` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 242–259 | Record, DTO or interface contract PRCreateOpIntent for Candidate publication and PR creation; declarations/fields: `PRCreateOpIntent` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 260–274 | Config.SubmitPRCreate — Candidate publication and PR creation; declarations/fields: `Config.SubmitPRCreate` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 275–302 | prCreatePayload — Candidate publication and PR creation; declarations/fields: `prCreatePayload` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 303–325 | submitOperation — Candidate publication and PR creation; declarations/fields: `submitOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 326–350 | mapSubmitError — Candidate publication and PR creation; declarations/fields: `mapSubmitError` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 351–360 | reconcileAfterSubmitError — Candidate publication and PR creation; declarations/fields: `reconcileAfterSubmitError` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 361–362 | Record, DTO or interface contract Adopted for Candidate publication and PR creation; declarations/fields: `Adopted` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 363–367 | Adopted.Error — Candidate publication and PR creation; declarations/fields: `Adopted.Error` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 368–377 | Config.LookupOperation — Candidate publication and PR creation; declarations/fields: `Config.LookupOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 378–401 | lookupOperation — Candidate publication and PR creation; declarations/fields: `lookupOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 402–420 | mapLookupError — Candidate publication and PR creation; declarations/fields: `mapLookupError` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 421–446 | Config.CancelOperation — Candidate publication and PR creation; declarations/fields: `Config.CancelOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 447–499 | mapRecord — Candidate publication and PR creation; declarations/fields: `mapRecord` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 500–518 | repository.runWithEnv — Candidate publication and PR creation; declarations/fields: `repository.runWithEnv` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 519–528 | Record, DTO or interface contract ValidatedRepo for Candidate publication and PR creation; declarations/fields: `ValidatedRepo` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 529–557 | Config.PrepareValidated — Candidate publication and PR creation; declarations/fields: `Config.PrepareValidated` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 558–575 | verifyCandidateBundle — Candidate publication and PR creation; declarations/fields: `verifyCandidateBundle` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 576–589 | validationVerdict — Candidate publication and PR creation; declarations/fields: `validationVerdict` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 590–601 | ValidatedRepo.Close — Candidate publication and PR creation; declarations/fields: `ValidatedRepo.Close` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 602–632 | ValidatedRepo.PushBranch — Candidate publication and PR creation; declarations/fields: `ValidatedRepo.PushBranch` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 633–657 | ValidatedRepo.reconcilePush — Candidate publication and PR creation; declarations/fields: `ValidatedRepo.reconcilePush` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 658–671 | Record, DTO or interface contract PublishReceipt for Candidate publication and PR creation; declarations/fields: `PublishReceipt` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 672–706 | DecodePublishReceipt — Candidate publication and PR creation; declarations/fields: `DecodePublishReceipt` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 707–721 | Record, DTO or interface contract PRCreateReceipt for Candidate publication and PR creation; declarations/fields: `PRCreateReceipt` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 722–754 | DecodePRCreateReceipt — Candidate publication and PR creation; declarations/fields: `DecodePRCreateReceipt` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 755–761 | OperationNotAfter — Candidate publication and PR creation; declarations/fields: `OperationNotAfter` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 762–766 | Record, DTO or interface contract StatusError for Candidate publication and PR creation; declarations/fields: `StatusError` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 767–769 | StatusError.Error — Candidate publication and PR creation; declarations/fields: `StatusError.Error` |

<a id="coverage-f06918b949ca"></a>

<a id="internalhostpublishoperation_testgo-1"></a>

## [internal/host/publish/operation_test.go](../../../../../internal/host/publish/operation_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 21–36 | Fixture/protocol support fakeBackgroundOps; declarations/fields: `fakeBackgroundOps` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 37–43 | Fixture/protocol support fakeBackgroundOps.ReadNativeRevision; declarations/fields: `fakeBackgroundOps.ReadNativeRevision` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 44–52 | Fixture/protocol support fakeBackgroundOps.SubmitOperation; declarations/fields: `fakeBackgroundOps.SubmitOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 53–59 | Fixture/protocol support fakeBackgroundOps.GetOperation; declarations/fields: `fakeBackgroundOps.GetOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 60–66 | Fixture/protocol support fakeBackgroundOps.CancelOperation; declarations/fields: `fakeBackgroundOps.CancelOperation` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 67–74 | Fixture/protocol support fakeBackgroundOps.PublishPushEnv; declarations/fields: `fakeBackgroundOps.PublishPushEnv` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 75–83 | Fixture/protocol support pendingRecord; declarations/fields: `pendingRecord` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 84–129 | Assertions TestSubmitPublishMapsTerminalDispatchVerdicts: status %d accepted; declarations/fields: `TestSubmitPublishMapsTerminalDispatchVerdicts` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 130–172 | Assertions TestSubmitPublishReconcilesLostReplies: lost reply: %v; declarations/fields: `TestSubmitPublishReconcilesLostReplies` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 173–205 | Assertions TestSubmitPublishAdoptsTerminalReplay: replay: %v; declarations/fields: `TestSubmitPublishAdoptsTerminalReplay` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 206–248 | Assertions TestSubmitPublishRejectsMalformedIntents: case %s accepted; declarations/fields: `TestSubmitPublishRejectsMalformedIntents` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 249–284 | Assertions TestSubmitPRCreateBuildsExactIntent: submit: %v; declarations/fields: `TestSubmitPRCreateBuildsExactIntent` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 285–322 | Assertions TestLookupAndCancelMapHonestly: not observed: %+v %v; declarations/fields: `TestLookupAndCancelMapHonestly` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 323–364 | Assertions TestDecodePublishReceiptRefusesLookalikes: decode: %v; declarations/fields: `TestDecodePublishReceiptRefusesLookalikes` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 365–396 | Assertions TestDecodePRCreateReceiptRefusesLookalikes: decode: %v; declarations/fields: `TestDecodePRCreateReceiptRefusesLookalikes` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 397–437 | Assertions TestPushBranchMovesExactlyItsTarget: bare: %v %s; declarations/fields: `TestPushBranchMovesExactlyItsTarget` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 438–492 | Assertions TestObserveTipParsesExactAdvertisement: bare: %v %s; declarations/fields: `TestObserveTipParsesExactAdvertisement` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 493–517 | Assertions TestObserveForPublishRefusesBeforeAnyCall: malformed refs accepted; declarations/fields: `TestObserveForPublishRefusesBeforeAnyCall` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 518–531 | Fixture/protocol support servePublicationGit; declarations/fields: `servePublicationGit` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 532–560 | Assertions TestPublicationRecordRejectsForeignScope: foreign record adopted; declarations/fields: `TestPublicationRecordRejectsForeignScope` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 561–574 | Assertions TestReceiptRejectsTrailingDocument: trailing branch document adopted; declarations/fields: `TestReceiptRejectsTrailingDocument` |

<a id="coverage-a409688841a5"></a>

## [internal/host/tailnet.go](../../../../../internal/host/tailnet.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 11–31 | Client.tailnetCall — Host Tailnet control; declarations/fields: `Client.tailnetCall` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 32–40 | Client.TailnetSettings — Host Tailnet control; declarations/fields: `Client.TailnetSettings` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 41–52 | Client.TailnetHost — Host Tailnet control; declarations/fields: `Client.TailnetHost` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 53–107 | Project enrollment policy protocol; declarations/fields: `admitEnrollmentSave`, `admitEnrollmentSaveRotate`, `admitEnrollmentAction`, `Client.TailnetEnrollment` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 108–161 | Project selection/policy observation protocol; declarations/fields: `Client.TailnetOptions`, `Client.TailnetPolicy`, `tailnetProjectConfirmed`, `Client.TailnetProject` |

<a id="coverage-9544bc76c23b"></a>

## [internal/host/terminal.go](../../../../../internal/host/terminal.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 1–21 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 22–28, 238–245 | Declared identifiers/bounds (declaration group) for Human terminal lifecycle; declarations/fields: `(declaration group)` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 29–41 | Record, DTO or interface contract TerminalRequest for Human terminal lifecycle; declarations/fields: `TerminalRequest` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 42–50 | Record, DTO or interface contract TerminalState for Human terminal lifecycle; declarations/fields: `TerminalState` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 51–59, 72–75, 96–99, 124–158, 185–206, 216–237, 246–257 | Interactive terminal attachment/stream wire contract; declarations/fields: `TerminalFrame`, `terminalDimensions`, `validSizedTerminalAction`, `validTypedInput`, `validResizeInput`, `validIdleInput`, `TerminalFrame.InputValid`, `validOutputShape`, `validMetadataOutput`, `validOutputData`, `validClosedOutput`, `TerminalFrame.OutputValid`, `Write` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 60–71 | ValidTerminalName — Human terminal lifecycle; declarations/fields: `ValidTerminalName` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 76–79 | validTerminalActor — Human terminal lifecycle; declarations/fields: `validTerminalActor` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 80–83 | validTerminalWindow — Human terminal lifecycle; declarations/fields: `validTerminalWindow` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 84–91 | validTerminalScope — Human terminal lifecycle; declarations/fields: `validTerminalScope` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 92–95 | validListRequest — Human terminal lifecycle; declarations/fields: `validListRequest` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 100–103 | validIdleTerminalAction — Human terminal lifecycle; declarations/fields: `validIdleTerminalAction` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 104–119 | validTerminalAction — Human terminal lifecycle; declarations/fields: `validTerminalAction` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 120–123 | TerminalRequest.Valid — Human terminal lifecycle; declarations/fields: `TerminalRequest.Valid` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 159–167 | validTerminalState — Human terminal lifecycle; declarations/fields: `validTerminalState` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 168–174 | validTerminalItemFlags — Human terminal lifecycle; declarations/fields: `validTerminalItemFlags` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 175–184 | validTerminalItem — Human terminal lifecycle; declarations/fields: `validTerminalItem` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 207–215 | validClosedReason — Human terminal lifecycle; declarations/fields: `validClosedReason` |

<a id="coverage-3f3e67152bb6"></a>

## [internal/host/terminal_client.go](../../../../../internal/host/terminal_client.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 16–17 | Record, DTO or interface contract Terminal for Interactive attachment; declarations/fields: `Terminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 18–38 | Client.OpenTerminal — Interactive attachment; declarations/fields: `Client.OpenTerminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 39–48 | Terminal.Send — Interactive attachment; declarations/fields: `Terminal.Send` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 49–56 | Terminal.Receive — Interactive attachment; declarations/fields: `Terminal.Receive` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 57–60 | Terminal.Close — Interactive attachment; declarations/fields: `Terminal.Close` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 61–95 | Human terminal inventory/lifecycle metadata; declarations/fields: `terminalMetadata`, `terminalTargetUnchanged`, `Client.TerminalStates` |


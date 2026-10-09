# Backend host

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-f4c2e2fdefc5"></a>
<a id="internalhostpublishoperationgo-1"></a>
<a id="coverage-96f0eeb0f412"></a>

## [internal/forgejo/publish/operation.go](../../../../../internal/forgejo/publish/operation.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–389; file scaffold; StatusError; Error; BackgroundOperations; Refusal; Wait; operationLifetime; PublishOpIntent; SubmitPublish; publishPayload; PRCreateOpIntent; SubmitPRCreate; prCreatePayload; submitOperation; mapSubmitError; reconcileAfterSubmitError; Adopted; LookupOperation; lookupOperation; mapLookupError; CancelOperation; mapRecord | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 25 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f06918b949ca"></a>
<a id="internalhostpublishoperation_testgo-1"></a>
<a id="coverage-4429ae29f03c"></a>

## [internal/forgejo/publish/operation_test.go](../../../../../internal/forgejo/publish/operation_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–345; file scaffold; fakeBackgroundOps; ReadNativeRevision; SubmitOperation; GetOperation; CancelOperation; PublishPushEnv; pendingRecord; TestSubmitPublishMapsTerminalDispatchVerdicts; TestSubmitPublishReconcilesLostReplies; TestSubmitPublishAdoptsTerminalReplay; TestSubmitPublishRejectsMalformedIntents; TestSubmitPRCreateBuildsExactIntent; TestLookupAndCancelMapHonestly; TestPublicationRecordRejectsForeignScope | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-572f9203c88c"></a>

## [internal/host/access_keys.go](../../../../../internal/host/access_keys.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–54; file scaffold; keyRevision; AccessKeys; canonicalKeys | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2a842406ecbb"></a>

## [internal/host/client.go](../../../../../internal/host/client.go)

Current selectors are maintained here; earlier unaffected intervals are historical hints pending R02.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; package/import and shared bounded HTTP response helpers | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Shared bounded HTTP response helpers. Earlier unchanged selectors are historical hints pending R02. |
| 78–86; Client.Create | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Project creation native operation; prior selector interval is a historical hint pending R02. |
| 87–95; Client.Inspect | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Observed Project runtime identity/address; prior selector interval is a historical hint pending R02. |
| 96–106; Client.Join | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Human Project account creation native operation; prior selector interval is a historical hint pending R02. |
| 107–119; Client.ProjectAccess | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Read-only Project privilege observation; 4 KiB reply cap, required administrator value, and project/login/identity echo validation. — Current source method and caller. |
| 121–128; Client.Connection; 130–133; validAddress | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Own native development connection observation and its address validator. — Current selectors verified. |

<a id="coverage-1c74cc0c7a7e"></a>

## [internal/host/factory_candidate.go](../../../../../internal/host/factory_candidate.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38; file scaffold; PrepareCandidate; FactoryInspectCandidate | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: PrepareCandidate; Current declaration duty: FactoryInspectCandidate — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d6bb7e900e71"></a>

## [internal/host/factory_candidate_test.go](../../../../../internal/host/factory_candidate_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–79; file scaffold; TestFactoryInspectCandidateClientBindsObservation; TestPrepareCandidateClientBindsExactSource | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryInspectCandidateClientBindsObservation; Current declaration duty: TestPrepareCandidateClientBindsExactSource — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0889ccbfcca1"></a>

## [internal/host/factory_client.go](../../../../../internal/host/factory_client.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31, 48–51, 61–69, 79–88, 131–138; whole file; ErrRunNotFound; ErrRunStale; factoryResponseLimit; factoryNotFound; factoryStateConfirmed; Client.FactoryInspect; Client.FactoryStop; Client.FactoryTakeover | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 9 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 32–47, 89–96; factoryOutputError, Client.FactoryOutput | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Bounded read-only factory output protocol; declarations/fields: `factoryOutputError`, `Client.FactoryOutput` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 52–60, 70–78; Client.FactoryLaunch, Client.FactoryHarness | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Native factory assignment launch or pinned harness observation; declarations/fields: `Client.FactoryLaunch`, `Client.FactoryHarness` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 97–130; factoryExportResponseLimit, Client.FactoryExport, factoryExportError | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Exact factory candidate export protocol; declarations/fields: `factoryExportResponseLimit`, `Client.FactoryExport`, `factoryExportError` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-41c6f09f058d"></a>

## [internal/host/factory_export_test.go](../../../../../internal/host/factory_export_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–44; file scaffold; TestFactoryExportClientPreservesConfirmedRefusals | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryExportClientPreservesConfirmedRefusals — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3d416738d438"></a>

## [internal/host/factory_harness_test.go](../../../../../internal/host/factory_harness_test.go)

Current source at `35ce794a`; tests exercise the production host client's request and response admission.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–70; file scaffold; TestFactoryHarnessRequestsSelectedFamily; TestFactoryHarnessRejectsUnsupportedFamilyBeforeIO; TestFactoryHarnessRejectsWrongFamilyResponse | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Selected family reaches the native request; unsupported selection makes no request; a valid pin for another family refuses. Actual Go client checks pass; no installed host or provider execution is claimed. |

<a id="coverage-ff42bb734720"></a>

## [internal/host/identity.go](../../../../../internal/host/identity.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; whole file; Client.IdentityLaunch | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Client.IdentityLaunch — Provider execution integration; declarations/fields: `Client.IdentityLaunch` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 16–19, 24–29; Client.IdentityValidate, Client.IdentityStart | [I05](../../slices/identity-brokering.md#i05-native-binding-and-private-delivery) | retained | Attested native execution binding; declarations/fields: `Client.IdentityValidate`, `Client.IdentityStart` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 20–23, 30–34; Client.IdentityStop, Client.IdentityFinish | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Native credential retirement acknowledgement; declarations/fields: `Client.IdentityStop`, `Client.IdentityFinish` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-f375656ed2d8"></a>

## [internal/host/lifecycle.go](../../../../../internal/host/lifecycle.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38; file scaffold; lifecycleActionConfirmed; lifecycleOutcomeConfirmed; Lifecycle | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-026142bb43ff"></a>

## [internal/host/os.go](../../../../../internal/host/os.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–63; file scaffold; validOSRelease; validOSEnvironment; validOSObservation; ObserveOS | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2b1ef0c1a495"></a>

## [internal/host/prepare.go](../../../../../internal/host/prepare.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; whole file; prepareResponseLimit; preparationIdentityConfirmed; Client.Prepare; Client.InspectPreparation; Client.StopPreparation | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 49–56; Client.HoldPreparation | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Native preparation admission hold synchronization; declarations/fields: `Client.HoldPreparation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-cc94f369637c"></a>

## [internal/host/prepare_test.go](../../../../../internal/host/prepare_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37; file scaffold; TestPrepareClientConfirmsIdentity; TestStopClientRequiresConfirmedStop; roundTripFunc; RoundTrip; jsonResponse | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold and shared fixture helpers; existing preparation/stop client checks. — Current source declarations and callers. |
| 38–79; TestProjectAccessClientRequiresBoundCompleteObservation | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Confirms matching binding, explicit false, and refusal on missing/null/mismatch/malformed/oversized/error responses. — Current client caller regression. |

<a id="coverage-84fde3dff37f"></a>

## [internal/host/profiles.go](../../../../../internal/host/profiles.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; file scaffold; ResolveProfile | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: ResolveProfile — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a409688841a5"></a>

## [internal/host/tailnet.go](../../../../../internal/host/tailnet.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–52; whole file; Client.tailnetCall; Client.TailnetSettings; Client.TailnetHost | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 53–107; admitEnrollmentSave, admitEnrollmentSaveRotate, admitEnrollmentAction, Client.TailnetEnrollment | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Project enrollment policy protocol; declarations/fields: `admitEnrollmentSave`, `admitEnrollmentSaveRotate`, `admitEnrollmentAction`, `Client.TailnetEnrollment` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 108–161; Client.TailnetOptions, Client.TailnetPolicy, tailnetProjectConfirmed, Client.TailnetProject | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Project selection/policy observation protocol; declarations/fields: `Client.TailnetOptions`, `Client.TailnetPolicy`, `tailnetProjectConfirmed`, `Client.TailnetProject` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-92428800368a"></a>

## [internal/host/tailnet_test.go](../../../../../internal/host/tailnet_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39; file scaffold; tailnetRoundTrip; RoundTrip; TestTailnetClientRejectsMalformedAndSecretBearingErrorResponses | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9544bc76c23b"></a>

## [internal/host/terminal.go](../../../../../internal/host/terminal.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50, 60–71, 76–95, 100–123, 159–184, 207–215, 238–245; whole file; (declaration group); TerminalRequest; TerminalState; ValidTerminalName; validTerminalActor; validTerminalWindow; validTerminalScope; validListRequest; validIdleTerminalAction; validTerminalAction; TerminalRequest.Valid; validTerminalState; validTerminalItemFlags; validTerminalItem; validClosedReason | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 16 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 51–59, 72–75, 96–99, 124–158, 185–206, 216–237, 246–257; TerminalFrame, terminalDimensions, validSizedTerminalAction, validTypedInput, validResizeInput, validIdleInput, TerminalFrame.InputValid, validOutputShape, validMetadataOutput, validOutputData, validClosedOutput, TerminalFrame.OutputValid | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Interactive terminal attachment/stream wire contract; declarations/fields: `TerminalFrame`, `terminalDimensions`, `validSizedTerminalAction`, `validTypedInput`, `validResizeInput`, `validIdleInput`, `TerminalFrame.InputValid`, `validOutputShape`, `validMetadataOutput`, `validOutputData`, `validClosedOutput`, `TerminalFrame.OutputValid`, `Write` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-3f3e67152bb6"></a>

## [internal/host/terminal_client.go](../../../../../internal/host/terminal_client.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–60; whole file; Terminal; Client.OpenTerminal; Terminal.Send; Terminal.Receive; Terminal.Close | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 61–95; terminalMetadata, terminalTargetUnchanged, Client.TerminalStates | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Human terminal inventory/lifecycle metadata; declarations/fields: `terminalMetadata`, `terminalTargetUnchanged`, `Client.TerminalStates` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

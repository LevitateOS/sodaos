# Go services, browser libraries and developer consumers

[Map index](README.md). Source `f8b22b0b` (2026-10-07); documentation changes are an explicit later delta.

Each entry distinguishes direct upstream API use from the retained application surface and its reverse callers. The tables preserve source selectors and composition targets. `Types` fields include upstream types used and Soda types exposed within the stated module scope; they do not imply that every listed type is a public Soda API. A source compiled with the product can contain test-only helpers: those uses are identified separately. Test source inspection is not test execution.

Selective source deltas: `5794a7d9` replaces SDK snapshot wire projections with
matching upstream typed values; `c84d242c` moves the staged SQLite writer into
existing factory `control_test` support. The dashboard's actual dependency list
now contains no modernc packages. `8b10ab14` retires the obsolete acceptance
lifecycle observer and its exclusive tests/dispatcher; SQLite remains only for
the two staged fixture callers. Earlier numeric
selectors and findings keep their original source scope.

`469f47f5` replaces the live private-input reopen/read bridge with same-FD
bounded admission in installed.go. `dbdb0615` removes uncalled Go command,
evidence and worker machinery and its exclusive tests; the current Rust
harness remains canonical. Go StartCommand/Process remains in acceptance
for live bounded installed probes, with its original cleanup regression.
No library selection or module changes accompany these cuts.

Numeric selectors refer to the pinned source. Multiple duties in one file remain separate responsibilities in the [coverage maps](../coverage/maps/README.md); this integration map does not transfer their defining owner.

The [adapter challenge](adapter-challenges.md) at `2dc3bce9` selects SDK snapshot
wire-value reuse, records upstream security prerequisites for the mirrored SDK
transport, and challenges Go JSON normalization with explicit alias policy.
Browser callback-bag consolidation remains a parked reassessment until a net
deletion is demonstrated. Developer/type-only entries are not product adapters.

| Integration | Selected version |
| --- | --- |
| [forgejo.org/extension-sdk](#integration-1) | v0.0.0 -&gt; local ../forgejo-ext/sdk @ c92db11c14b773c9cc20ccfa4b853b4c017e8717 |
| [github.com/coder/websocket](#integration-2) | v1.8.15 |
| [github.com/dicebear/dicebear-go/v10](#integration-3) | v10.7.0 |
| [golang.org/x/crypto/ssh](#integration-4) | golang.org/x/crypto v0.55.0 |
| [golang.org/x/oauth2](#integration-5) | golang.org/x/oauth2 v0.34.0 |
| [golang.org/x/sys/unix](#integration-6) | golang.org/x/sys v0.47.0 |
| [tailscale.com/client/tailscale/v2](#integration-7) | v2.10.1 |
| [modernc.org/sqlite](#integration-8) | v1.58.0 |
| [github.com/jackc/pgx/v5](#integration-9) | v5.10.0 |
| [github.com/stretchr/testify](#integration-10) | v1.12.1 |
| [Go standard library net/http](#integration-11) | Go 1.26.7 module language/toolchain floor (go.mod) |
| [Go standard library encoding/json](#integration-12) | Go 1.26.7 module language/toolchain floor (go.mod) |
| [Go standard library encoding/base64](#integration-13) | Go 1.26.7 module language/toolchain floor (go.mod) |
| [Go standard library crypto/*](#integration-14) | Go 1.26.7 module language/toolchain floor (go.mod) |
| [Go standard library net/url](#integration-15) | Go 1.26.7 module language/toolchain floor (go.mod) |
| [Go standard library time](#integration-16) | Go 1.26.7 module language/toolchain floor (go.mod) |
| [lit](#integration-17) | 3.3.3 |
| [@xterm/xterm](#integration-18) | 6.0.0 |
| [@xterm/addon-fit](#integration-19) | 0.11.0 |
| [jsdom](#integration-20) | 30.0.1 |
| [playwright](#integration-21) | 1.63.0 |
| [typescript](#integration-22) | root 7.0.2; tools/lit-check workspace 5.9.3 |
| [lit-analyzer](#integration-23) | 2.0.3 |
| [oxfmt](#integration-24) | 0.68.0 |
| [oxlint](#integration-25) | 1.81.0 |
| [@types/bun](#integration-26) | 1.4.2 |
| [@types/node](#integration-27) | 24.10.1 |
| [@types/jsdom](#integration-28) | 30.0.0 |
| [Go module dependency-only / indirect set](#integration-29) | selected versions as go.mod/go.sum |
| [github.com/fzipp/gocyclo](#integration-30) | v0.6.0 |
| [github.com/kisielk/errcheck](#integration-31) | v1.9.0 |
| [honnef.co/go/tools/cmd/staticcheck](#integration-32) | v0.8.1 |
| [mvdan.cc/gofumpt](#integration-33) | v0.9.1 |

<a id="integration-1"></a>


> Line-level profile selectors are historical evidence from the recorded source snapshot. A moved or retired path is shown as a literal locator; current ownership is navigated through the [coverage maps](../coverage/maps/README.md).

## forgejo.org/extension-sdk

**Selected:** v0.0.0 -&gt; local ../forgejo-ext/sdk @ c92db11c14b773c9cc20ccfa4b853b4c017e8717. **Upstream capability used:** Forgejo extension host SDK: sibling-local typed Actor/Repository/PublicKey/Authority and contribution/policy contract DTOs/interfaces; private callback clients; browser extension and browserless background admission; bounded snapshot reads and conditional native operations; payload validators.

**Dependency admission:** go.mod require forgejo.org/extension-sdk v0.0.0; replace directive ../forgejo-ext/sdk; upstream sibling go.mod module name and clean Git identity.

**Primary-source evidence:** Inspected selected local SDK source: contract.go, native.go, background.go, snapshot.go, publish.go, merge.go, prcreate.go, review.go, runtime.go; selected APIs below are direct imports, not inferred from nominal release version. Selected Git tree id: 1791fdbb1adc97e86b14eda36475d89147531439; worktree and index clean, branch `codex/fountain-extensions` one commit ahead of its recorded upstream. Sibling SDK go.mod SHA-256: 446dc47271e222dfb6548f6dc32f9b7cb72b8169f307beb3c8263196542844e5.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [cmd/soda-extension/main.go](../../../../cmd/soda-extension/main.go) | ContributionAuthorizer; ContributionDecision; ContributionRequest; DataDir; Serve; extensions.Serve; extensions.Application; extensions.DataDir | **role:** Starts the current extension process and registers its HTTP/policy/contribution application.; **public types:** contributionAuthorizer; extensions.Application; extensions.ContributionAuthorizer; extensions.ContributionDecision; extensions.ContributionRequest; **conversions:** Soda contribution allow/deny mapping in allowedContribution; environment data dir read from SDK.; **production consumers:** [cmd/soda-extension/main.go](../../../../cmd/soda-extension/main.go) | **role:** Process entrypoint or command adapter.; **chain:** cmd/soda-extension/main.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** allowedContribution; contributionAuthorizer; run |
| [internal/forgejo/background.go](../../../../internal/forgejo/background.go) | BackgroundCancelPath; BackgroundGetPath; BackgroundRevisionPath; BackgroundSnapshotPath; BackgroundSubmitPath; CredentialFile; NativeRevisionObservation; NativeSnapshot; OperationIntent; OperationLookup; OperationRecord; PublishPushEnv; SnapshotRequest; ValidateSnapshotRequest | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/background.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** CancelOperation; GetOperation; PublishPushEnv; ReadNativeRevision; ReadSnapshot; SubmitOperation |
| [internal/forgejo/background_admission.go](../../../../internal/forgejo/background_admission.go) | BackgroundBootstrapPath; CredentialFile; OperationRecord; PeerCredential | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/background_admission.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** backgroundSecret; bootstrap; checkBackgroundRecord; verifiedPeer |
| [internal/forgejo/background_transport.go](../../../../internal/forgejo/background_transport.go) | AdmissionHeader; BackgroundOutcomeNotObserved; OperationLookup | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/background_transport.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** checkBackgroundLookup; post |
| [internal/forgejo/checks.go](../../../../internal/forgejo/checks.go) | CredentialFile; SnapshotRequest | **role:** Checks evidence through the shared snapshot reader and converts bounded native observations into factory check inputs.; **public types:** CheckAssessor; extensions.CredentialFile; **conversions:** local factory check request -&gt; bounded snapshot families -&gt; domain check result.; **production consumers:** [internal/forgejo/checks.go](../../../../internal/forgejo/checks.go) | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/checks.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** ObserveChecks |
| [internal/forgejo/merge.go](../../../../internal/forgejo/merge.go) | BackgroundOutcomeNotObserved; BackgroundOutcomeOperationsUnavailable; CredentialFile; MergeMethodFastForwardOnly; MergePayload; OperationIntent; OperationRecord; ValidateMergePayload | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/merge.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** LookupOp; ObserveMerge; SubmitMerge; mergeIntent; mergeOperationOutcome |
| [internal/forgejo/merge_completion.go](../../../../internal/forgejo/merge_completion.go) | CredentialFile | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/merge_completion.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** ObserveCompletion |
| [internal/forgejo/observe.go](../../../../internal/forgejo/observe.go) | BackgroundClient; CredentialFile; NativeRevisionObservation | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/observe.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Credential; ReadNativeRevision; ReadSnapshot; ServiceObserver; ensureClient |
| [internal/forgejo/publish.go](../../../../internal/forgejo/publish.go) | CredentialFile | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/publish.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** SubmitPRCreate; SubmitPublish |
| [internal/forgejo/publish/operation.go](../../../../internal/forgejo/publish/operation.go) | BackgroundOutcomeNotObserved; BackgroundOutcomeOperationsUnavailable; CredentialFile; NativeRevisionObservation; OperationIntent; OperationLookup; OperationRecord; PRCreatePayload; PublishCorrection; PublishPayload; ValidatePRCreatePayload; ValidatePublishPayload | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/publish/operation.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** BackgroundOperations; PRCreateOpIntent; PublishOpIntent; SubmitPRCreate; SubmitPublish; lookupOperation; mapRecord; prCreatePayload; publishPayload; submitOperation |
| [internal/forgejo/publish/operation_push.go](../../../../internal/forgejo/publish/operation_push.go) | FormatPublishRefspec | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/publish/operation_push.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** PushBranch |
| [internal/forgejo/publish/operation_receipts.go](../../../../internal/forgejo/publish/operation_receipts.go) | PublishExpectedOldAbsent | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/publish/operation_receipts.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** DecodePublishReceipt |
| [internal/forgejo/review.go](../../../../internal/forgejo/review.go) | BackgroundOutcomeNotObserved; BackgroundOutcomeOperationsUnavailable; CredentialFile; OperationIntent; OperationRecord; ReviewSubmitPayload; ValidateReviewSubmitPayload | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/review.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** LookupOp; ObserveReview; SubmitReview; reviewIntent; reviewOperationOutcome |
| [internal/forgejo/snapshot.go](../../../../internal/forgejo/snapshot.go) | CredentialFile; NativeRevisionObservation | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/snapshot.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** BracketedRead; SnapshotReader |
| [internal/forgejo/snapshot_transport.go](../../../../internal/forgejo/snapshot_transport.go) | BackgroundClient; CredentialFile; NativeRevisionObservation; NativeSnapshot; SnapshotRequest | none | **role:** Forgejo SDK/client adapter and domain conversion.; **chain:** internal/forgejo/snapshot_transport.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** BackgroundSnapshotReader; ReadNativeRevision; ReadSnapshot; decodeSnapshotAnswer |
| [internal/web/api/dispatch_inputs.go](../../../../internal/web/api/dispatch_inputs.go) | CredentialFile | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/dispatch_inputs.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** DispatchSnapshotSource |
| [internal/web/api/environment_authority.go](../../../../internal/web/api/environment_authority.go) | Authority; ErrRepositoryNotVisible | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/environment_authority.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** nativeVisibleRepository |
| [internal/web/api/extension.go](../../../../internal/web/api/extension.go) | AdmissionHeader; ContextHeader; SessionGenerationHeader | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/extension.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Extension; rewriteExtensionRequest |
| [internal/web/api/extension_native.go](../../../../internal/web/api/extension_native.go) | Authority; Contribution; SessionGenerationHeader | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/extension_native.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** extensionActor; extensionAdminPageContribution; extensionAuthorityCurrent; extensionGenerationCurrent; extensionPageContribution; extensionProductActor; extensionProductAuthority; extensionProductContribution; extensionWorkspacePanelContribution; logExtensionDenial; requestExtensionAuthority |
| [internal/web/api/extension_terminal_authority.go](../../../../internal/web/api/extension_terminal_authority.go) | Authority; SessionGenerationHeader | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/extension_terminal_authority.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** extensionTerminalGeneration; extensionTerminalIdentity; nativeTerminalActor; nativeTerminalContribution; nativeTerminalSessionActor; sameNativeTerminalAuthority |
| [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go) | Authority | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/factory_output.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** factoryViewerIdentity |
| [internal/web/api/issue_acceptance_evidence.go](../../../../internal/web/api/issue_acceptance_evidence.go) | CredentialFile | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/issue_acceptance_evidence.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** AcceptanceSnapshotSource |
| [internal/web/api/repositories.go](../../../../internal/web/api/repositories.go) | Actor; Authority; Repository; RepositoryPage; SearchOwnedRepositories | **role:** Owned-repository search, actor ownership validation and paging endpoint.; **public types:** extensions.Actor; extensions.Authority; extensions.Repository; extensions.RepositoryPage; repositoryChoice; repositoryPage; **conversions:** SDK repository IDs/owner/name -&gt; repositoryChoice; store Project association -&gt; Project readiness DTO.; **production consumers:** [internal/web/api/repositories.go](../../../../internal/web/api/repositories.go) | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/repositories.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** apiNativeRepositories; collectNativeRepositoryChoices |
| [internal/web/auth/errors.go](../../../../internal/web/auth/errors.go) | ErrRepositoryNotVisible | none | **role:** Web authorization/session facade.; **chain:** internal/web/auth/errors.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** ProviderError |
| [internal/web/auth/extension.go](../../../../internal/web/auth/extension.go) | AdmissionHeader; Authority; ContextHeader; Contribution; RequestContext; SessionGenerationHeader | **role:** Parses SDK authority from the private extension request and verifies extension/method/actor against Soda session.; **public types:** RequestContext; extensions.Authority; extensions.Contribution; **conversions:** SDK Actor.ID decimal string -&gt; Soda int64; SDK Contribution -&gt; Soda resource/action checks; extension headers -&gt; verified request context.; **production consumers:** [internal/web/auth/extension.go](../../../../internal/web/auth/extension.go); [internal/web/auth/extension_service.go](../../../../internal/web/auth/extension_service.go); [internal/web/auth/errors.go](../../../../internal/web/auth/errors.go) | **role:** Web authorization/session facade.; **chain:** internal/web/auth/extension.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** ExtensionAuthority; ExtensionContribution |
| [internal/web/auth/extension_service.go](../../../../internal/web/auth/extension_service.go) | Authority; SessionGenerationHeader | none | **role:** Web authorization/session facade.; **chain:** internal/web/auth/extension_service.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** extensionIdentity; extensionRequest; extensionSessionGenerationMatches; serveExtensionResource; serveExtensionSession |
| [internal/web/extension.go](../../../../internal/web/extension.go) | Application; PolicyDecision; PolicyForgejoUsername; PolicyHandler; PolicyRequest | **role:** Composes the SDK application and maps Linux username policy requests to Soda host-side account constraints.; **public types:** Application; extensions.Application; extensions.PolicyDecision; extensions.PolicyHandler; extensions.PolicyRequest; **conversions:** SDK username policy request -&gt; Linux login allow/deny response.; **production consumers:** [internal/web/extension.go](../../../../internal/web/extension.go) | **role:** Web server composition or feature adapter.; **chain:** internal/web/extension.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Extension; forgejoUsernamePolicy |
| internal/web/api/extension.go, extension_native.go, extension_terminal_authority.go, environment_authority.go | Authority; NativeClient; Contribution; RequestContext; SessionGenerationHeader; ContextHeader; AdmissionHeader; ErrRepositoryNotVisible | **role:** Extension bridge strips/forwards authority headers; native product handlers admit exact actor/contribution/session and call SDK NativeClient for repository/actor/key data.; **public types:** API; extensionTerminalIdentity; extensions.Actor; extensions.Authority; extensions.Contribution; extensions.PublicKey; extensions.Repository; nativeActor; **conversions:** Actor.ID and Repository.ID decimal strings -&gt; typed native IDs; SDK response/visibility error -&gt; HTTP status; contribution tuples -&gt; endpoint-specific authorization.; **production consumers:** [internal/web/api/extension.go](../../../../internal/web/api/extension.go); [internal/web/api/extension_native.go](../../../../internal/web/api/extension_native.go); [internal/web/api/extension_terminal_authority.go](../../../../internal/web/api/extension_terminal_authority.go); [internal/web/api/environment_authority.go](../../../../internal/web/api/environment_authority.go) | none |
| internal/forgejo/snapshot.go, snapshot_transport.go, observe.go, background.go | BackgroundClient; SnapshotRequest; NativeSnapshot; CredentialFile; ValidateSnapshotRequest; NativeRevisionObservation | **role:** Maps bounded SDK snapshots and revision observations into the Soda Forgejo evidence read model, brackets reads and controls long-lived service background admission.; **public types:** BackgroundSnapshotReader; NativeRevisionObservation; NativeSnapshot; SnapshotReader; SnapshotRequest; extensions.BackgroundClient; extensions.CredentialFile; extensions.NativeRevisionObservation; extensions.NativeSnapshot; extensions.SnapshotRequest; **conversions:** SDK NativeSnapshot -&gt; Soda NativeSnapshot (per-family completeness/visibility/locators); SDK revisions -&gt; local revision bracket; credential file path -&gt; typed restricted credential handle.; **production consumers:** [internal/forgejo/snapshot.go](../../../../internal/forgejo/snapshot.go); [internal/forgejo/snapshot_transport.go](../../../../internal/forgejo/snapshot_transport.go); [internal/forgejo/observe.go](../../../../internal/forgejo/observe.go); [internal/forgejo/background.go](../../../../internal/forgejo/background.go) | none |
| internal/forgejo/background_admission.go, background_transport.go, background.go | PeerCredential; BackgroundBootstrapPath; BackgroundRevisionPath; BackgroundSnapshotPath; BackgroundSubmitPath; BackgroundGetPath; BackgroundCancelPath; OperationIntent; OperationRecord; OperationLookup; PublishPushEnv | **role:** Bridges installed service callback/bootstrap to the SDK background runtime and validates operation observations before domain callers adopt them.; **public types:** ServiceBackground; backgroundClientAdapter; extensions.BackgroundAdmissionDelivery; extensions.OperationIntent; extensions.OperationLookup; extensions.OperationRecord; **conversions:** Soda operation requests -&gt; SDK typed intents; SDK operation outcome/lookup -&gt; local errors and factory.OperationOutcome; publish operation binding -&gt; GIT_CONFIG_COUNT/KEY/VALUE environment.; **production consumers:** [internal/forgejo/background_admission.go](../../../../internal/forgejo/background_admission.go); [internal/forgejo/background_transport.go](../../../../internal/forgejo/background_transport.go); [internal/forgejo/background.go](../../../../internal/forgejo/background.go) | none |
| internal/forgejo/publish/operation.go, operation_push.go, operation_receipts.go | OperationIntent; PublishPayload; PRCreatePayload; ValidatePublishPayload; ValidatePRCreatePayload; FormatPublishRefspec; PublishExpectedOldAbsent | **role:** Translates validated candidate and correction records into exact conditional branch/PR operations; validates SDK intent early and interprets returned receipts.; **public types:** BackgroundOperations; extensions.OperationIntent; extensions.PRCreatePayload; extensions.PublishCorrection; extensions.PublishPayload; publishOperation; **conversions:** factory publication work -&gt; SDK operation payload and JSON RawMessage; exact ref tuple -&gt; Git refspec; SDK operation record -&gt; Soda publication result.; **production consumers:** [internal/forgejo/publish/operation.go](../../../../internal/forgejo/publish/operation.go); [internal/forgejo/publish/operation_push.go](../../../../internal/forgejo/publish/operation_push.go); [internal/forgejo/publish/operation_receipts.go](../../../../internal/forgejo/publish/operation_receipts.go) | none |
| internal/forgejo/merge.go, review.go, publish.go, merge_completion.go | MergePayload; ReviewSubmitPayload; OperationIntent; OperationRecord; CredentialFile; ValidateMergePayload; ValidateReviewSubmitPayload | **role:** Maps factory merge/review/publication work through SDK validated intents and restricted native credential files.; **public types:** Merger; Publisher; Reviewer; extensions.CredentialFile; extensions.MergePayload; extensions.OperationIntent; extensions.OperationRecord; extensions.ReviewSubmitPayload; **conversions:** factory.MergeWork/ReviewWork -&gt; typed SDK payloads; SDK outcomes -&gt; factory.OperationOutcome; local token file path -&gt; CredentialFile.; **production consumers:** [internal/forgejo/merge.go](../../../../internal/forgejo/merge.go); [internal/forgejo/review.go](../../../../internal/forgejo/review.go); [internal/forgejo/publish.go](../../../../internal/forgejo/publish.go); [internal/forgejo/merge_completion.go](../../../../internal/forgejo/merge_completion.go) | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [cmd/soda-extension/main_test.go](../../../../cmd/soda-extension/main_test.go) | Contribution; ContributionRequest | Process entrypoint or command adapter. |
| [internal/factory/control/merge_native_fixture_test.go](../../../../internal/factory/control/merge_native_fixture_test.go) | CredentialFile; NativeRevisionObservation; OperationIntent; OperationRecord | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_test.go](../../../../internal/factory/control/merge_native_test.go) | CredentialFile; MergeMethodFastForwardOnly; MergePayload; NativeSnapshot; SnapshotRequest | Factory coordinator composition or runtime service. |
| [internal/factory/control/review_native_primitive_test.go](../../../../internal/factory/control/review_native_primitive_test.go) | CredentialFile; NativeRevisionObservation; OperationIntent; PublishPayload; ReviewSubmitEventRequestChanges; ReviewSubmitPayload; SnapshotRequest | Factory coordinator composition or runtime service. |
| [internal/forgejo/background_test.go](../../../../internal/forgejo/background_test.go) | AdmissionHeader; BackgroundBootstrapPath; BackgroundCancelPath; BackgroundGetPath; BackgroundOutcomeNotObserved; BackgroundRevisionPath; BackgroundSubmitPath; CredentialFile; OperationIntent; OperationLookup; OperationRecord; PublishOperationHeader | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge_test.go](../../../../internal/forgejo/merge_test.go) | MergeMethodFastForwardOnly; MergePayload; OperationRecord; ValidateMergePayload | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/observe_test.go](../../../../internal/forgejo/observe_test.go) | AdmissionHeader; BackgroundBootstrapPath; BackgroundRevisionPath; BackgroundSnapshotPath; CredentialFile; SnapshotRequest | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_git_test.go](../../../../internal/forgejo/publish/operation_git_test.go) | NativeRevisionObservation | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_receipts_test.go](../../../../internal/forgejo/publish/operation_receipts_test.go) | PublishExpectedOldAbsent | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_test.go](../../../../internal/forgejo/publish/operation_test.go) | BackgroundOutcomeNotObserved; CredentialFile; NativeRevisionObservation; OperationIntent; OperationLookup; OperationRecord; PRCreatePayload; PublishExpectedOldAbsent; PublishPayload | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish_test.go](../../../../internal/forgejo/publish_test.go) | OperationRecord | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/review_test.go](../../../../internal/forgejo/review_test.go) | BackgroundSubmitPath; OperationRecord | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/snapshot_test.go](../../../../internal/forgejo/snapshot_test.go) | CredentialFile; NativeRevisionObservation | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/snapshot_transport_test.go](../../../../internal/forgejo/snapshot_transport_test.go) | CredentialFile; NativeRevisionObservation; NativeSnapshot; OperationIntent; OperationLookup; OperationRecord; SnapshotCheck; SnapshotCheckSet; SnapshotCreationProvenance; SnapshotDependency; SnapshotDependencyPage; SnapshotIssue; SnapshotLifecycleEvent; SnapshotRef; SnapshotRequest | Forgejo SDK/client adapter and domain conversion. |
| [internal/web/api/dispatch_inputs_test.go](../../../../internal/web/api/dispatch_inputs_test.go) | CredentialFile; NativeRevisionObservation | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_test.go](../../../../internal/web/api/extension_test.go) | AdmissionHeader; Authority; ContextHeader; Contribution | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_views_test.go](../../../../internal/web/api/factory_views_test.go) | Authority | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/issue_acceptances_test.go](../../../../internal/web/api/issue_acceptances_test.go) | CredentialFile; NativeRevisionObservation | Product HTTP/WebSocket handler or adapter. |
| [internal/web/auth/errors_test.go](../../../../internal/web/auth/errors_test.go) | ErrRepositoryNotVisible | Web authorization/session facade. |
| [internal/web/auth/extension_test.go](../../../../internal/web/auth/extension_test.go) | AdmissionHeader; CallbackEnv; ContextHeader; Contribution | Web authorization/session facade. |
| [internal/web/environment_authority_test.go](../../../../internal/web/environment_authority_test.go) | CallbackRequest; CallbackResponse; OperationOrganizationOwner; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/environment_os_test.go](../../../../internal/web/environment_os_test.go) | CallbackRequest; CallbackResponse; ContextHeader; Repository | Web server composition or feature adapter. |
| [internal/web/environment_read_publication_test.go](../../../../internal/web/environment_read_publication_test.go) | CallbackRequest; CallbackResponse; ContextHeader; OperationOrganizationOwner; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/execution_access_test.go](../../../../internal/web/execution_access_test.go) | CallbackRequest; CallbackResponse; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/extension_terminal_test.go](../../../../internal/web/extension_terminal_test.go) | Actor; AdmissionHeader; Authority; CallbackRequest; CallbackResponse; ContextHeader; Contribution; NativeCallbackPath; OperationCurrentActor; OperationRepository; Repository; ServiceCallbackEnv; SessionGenerationHeader | Web server composition or feature adapter. |
| [internal/web/extension_test.go](../../../../internal/web/extension_test.go) | LoadManifest; PolicyForgejoUsername; PolicyRequest | Web server composition or feature adapter. |
| [internal/web/factory_lifecycle_test.go](../../../../internal/web/factory_lifecycle_test.go) | CallbackRequest; CallbackResponse; Repository | Web server composition or feature adapter. |
| [internal/web/factory_output_fixture_test.go](../../../../internal/web/factory_output_fixture_test.go) | ServiceCallbackEnv; SessionGenerationHeader | Web server composition or feature adapter. |
| [internal/web/factory_output_stream_test.go](../../../../internal/web/factory_output_stream_test.go) | SessionGenerationHeader | Web server composition or feature adapter. |
| [internal/web/factory_views_test.go](../../../../internal/web/factory_views_test.go) | CallbackRequest; CallbackResponse; Repository | Web server composition or feature adapter. |
| [internal/web/forgejo_keys_test.go](../../../../internal/web/forgejo_keys_test.go) | CallbackRequest; CallbackResponse; OperationPublicSSHKeys; PublicKey | Web server composition or feature adapter. |
| [internal/web/identity_native_test.go](../../../../internal/web/identity_native_test.go) | CallbackRequest; CallbackResponse; ContextHeader; OperationRepository; Repository; SessionGenerationHeader | Web server composition or feature adapter. |
| [internal/web/issue_acceptances_test.go](../../../../internal/web/issue_acceptances_test.go) | CallbackRequest; CallbackResponse; Repository | Web server composition or feature adapter. |
| [internal/web/lifecycle_access_keys_test.go](../../../../internal/web/lifecycle_access_keys_test.go) | SessionGenerationHeader | Web server composition or feature adapter. |
| [internal/web/mutation_admission_test.go](../../../../internal/web/mutation_admission_test.go) | CallbackRequest; CallbackResponse; ContextHeader; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/project_profiles_test.go](../../../../internal/web/project_profiles_test.go) | CallbackRequest; CallbackResponse; ContextHeader; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/repositories_test.go](../../../../internal/web/repositories_test.go) | CallbackRequest; CallbackResponse; ContextHeader; OperationOwnedRepositories; Repository; RepositoryPage | Web server composition or feature adapter. |
| [internal/web/repository_access_test.go](../../../../internal/web/repository_access_test.go) | CallbackRequest; CallbackResponse; OperationOwnedRepositories; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/repository_settings_test.go](../../../../internal/web/repository_settings_test.go) | CallbackRequest; CallbackResponse; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/retired_frontend_test.go](../../../../internal/web/retired_frontend_test.go) | Contribution | Web server composition or feature adapter. |
| [internal/web/spaces_inventory_test.go](../../../../internal/web/spaces_inventory_test.go) | CallbackRequest; CallbackResponse | Web server composition or feature adapter. |
| [internal/web/spaces_test.go](../../../../internal/web/spaces_test.go) | Actor; CallbackRequest; CallbackResponse; ContextHeader; Contribution; OperationRepository; Repository | Web server composition or feature adapter. |
| [internal/web/tailnet_test.go](../../../../internal/web/tailnet_test.go) | CallbackRequest; CallbackResponse; ContextHeader; OperationOrganizationOwner; OperationRepository; Repository; SessionGenerationHeader | Web server composition or feature adapter. |
| [internal/web/test_helpers_test.go](../../../../internal/web/test_helpers_test.go) | Actor; AdmissionHeader; Authority; CallbackRequest; CallbackResponse; ContextHeader; Contribution; NativeCallbackPath; OperationCurrentActor; OperationOrganizationOwner; OperationOwnedRepositories; OperationPublicSSHKeys; OperationRepository; PublicKey; Repository; RepositoryPage; ServiceCallbackEnv; SessionGenerationHeader | Web server composition or feature adapter. |

**Transfer / predecessor evidence:** **item:** Old direct browser callback/extension API shapes replaced by selected local SDK Authority, NativeClient and BackgroundClient contracts.; **evidence:** Current imports and selected SDK source; no compatibility aliases observed in listed production consumers.

<a id="integration-2"></a>

## github.com/coder/websocket

**Selected:** v1.8.15. **Upstream capability used:** WebSocket accept/dial, frame reads/writes, contexts, control/close and read limits.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; direct calls in host and web/api protocol adapters; application frames are Soda-owned JSON types.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/host/terminal.go](../../../../internal/host/terminal.go) | Conn; MessageText | none | **role:** Typed native host client or privileged development executor.; **chain:** internal/host/terminal.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Write |
| [internal/host/terminal_client.go](../../../../internal/host/terminal_client.go) | Conn; Dial; DialOptions; MessageText | none | **role:** Typed native host client or privileged development executor.; **chain:** internal/host/terminal_client.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** OpenTerminal; Receive |
| [internal/web/api/extension_terminal_stream.go](../../../../internal/web/api/extension_terminal_stream.go) | Accept; AcceptOptions; Conn | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/extension_terminal_stream.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** extensionTerminalAttach |
| [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go) | Accept; AcceptOptions; Conn; MessageText; StatusPolicyViolation | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/factory_output.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** factoryOutputAttach; factoryRejectViewerInput; readFactoryOutputHandshake; refuseFactoryOutput; writeFactoryFrame |
| [internal/web/api/terminal_registry.go](../../../../internal/web/api/terminal_registry.go) | Conn; MessageText; StatusPolicyViolation | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/terminal_registry.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** pumpNativeToExtension; pumpTerminalInput; readTerminalHandshake; refuseTerminal |
| internal/host/terminal.go + terminal_client.go | websocket.Dial; Read; Write; SetReadLimit; CloseNow; MessageText | **role:** Typed host facade and private helper connection; web/api handles browser-facing stream separately.; **public types:** Client; Terminal; TerminalFrame; TerminalRequest; TerminalState; host.Terminal; **conversions:** HTTP/WebSocket frames -&gt; Soda JSON terminal frames; context, limits and close outcomes -&gt; host errors. | none |
| internal/web/api/extension_terminal_stream.go + factory_output.go + terminal_registry.go | websocket.Accept; Read; Write; Close; CloseStatus | **role:** Accepts authorized browser attachment/output streams and transports bounded status/output or terminal frames. Public browser routes are registered in internal/web/api/extension_terminal.go and internal/web/api/extension_native.go.; **public types:** TerminalFrame; factoryOutputFrame; factoryStatusFrame; terminalRegistry; **conversions:** Soda frame structs -&gt; JSON WebSocket frames; coder CloseError/status -&gt; endpoint lifecycle outcomes. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/web/api/factory_output_test.go](../../../../internal/web/api/factory_output_test.go) | Accept; AcceptOptions; Conn; Dial | Product HTTP/WebSocket handler or adapter. |
| [internal/web/extension_terminal_test.go](../../../../internal/web/extension_terminal_test.go) | Conn; Dial; DialOptions; MessageText | Web server composition or feature adapter. |
| [internal/web/factory_output_fixture_test.go](../../../../internal/web/factory_output_fixture_test.go) | Conn; Dial; DialOptions; MessageText | Web server composition or feature adapter. |
| [internal/web/factory_output_stream_test.go](../../../../internal/web/factory_output_stream_test.go) | CloseError; Dial; DialOptions; MessageText; StatusPolicyViolation | Web server composition or feature adapter. |
| [internal/web/spaces_inventory_test.go](../../../../internal/web/spaces_inventory_test.go) | Accept; MessageText | Web server composition or feature adapter. |
| [internal/web/spaces_test.go](../../../../internal/web/spaces_test.go) | Accept; MessageText | Web server composition or feature adapter. |
| [internal/web/terminal_test.go](../../../../internal/web/terminal_test.go) | Accept; Conn; MessageText | Web server composition or feature adapter. |

<a id="integration-3"></a>

## github.com/dicebear/dicebear-go/v10

**Selected:** v10.7.0. **Upstream capability used:** Style parsing and deterministic avatar SVG generation.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; direct calls NewStyle/NewAvatar in internal/avatar and tools/soda-avatars.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/avatar/avatar.go](../../../../internal/avatar/avatar.go) | NewAvatar; NewStyle; Style; dicebear.NewStyle; dicebear.NewAvatar | **role:** Canonical SVG avatar renderer; parses one embedded style definition once, generates deterministic seeded SVG and returns application render output.; **public types:** Renderer; SVG; avatar.Renderer; avatar.SVG; **conversions:** Caller avatar identity/seed -&gt; Dicebear style options -&gt; SVG bytes/data URI. | **role:** Avatar rendering facade.; **chain:** internal/avatar/avatar.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Render; style |
| [tools/soda-avatars/main.go](../../../../tools/soda-avatars/main.go) | NewAvatar; NewStyle; Style; dicebear.NewStyle; dicebear.NewAvatar | **role:** Standalone developer asset catalog command enumerates style variants/colors and writes avatar sheet outputs. | **role:** Developer/support command.; **chain:** tools/soda-avatars/main.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** catalogAvatarSheet; colorSection; generate; variantSection |
| internal/web/avatars.go; internal/web/server.go; cmd/soda-dashboard/main.go | avatar.SVG; avatar.New | **role:** Dashboard avatar HTTP endpoint and server composition consume the shared avatar facade; dashboard executable injects it.; **public types:** HTTP avatar response; **conversions:** avatar SVG -&gt; HTTP image response; query parameter is validated before rendering. | none |

<a id="integration-4"></a>

## golang.org/x/crypto/ssh

**Selected:** golang.org/x/crypto v0.55.0. **Upstream capability used:** SSH public-key parsing, canonical authorized-key formatting and fingerprinting.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; calls are limited to access-key request/host/auth conversions.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/host/access_keys.go](../../../../internal/host/access_keys.go) | MarshalAuthorizedKey; ParseAuthorizedKey; ssh.ParsePublicKey; ssh.MarshalAuthorizedKey; ssh.FingerprintSHA256 | **role:** Validates canonical development public keys before host project account installation.; **public types:** project.AccessKey; **conversions:** authorized_keys string -&gt; ssh.PublicKey -&gt; canonical fingerprint/authorized-key bytes. | **role:** Typed native host client or privileged development executor.; **chain:** internal/host/access_keys.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** canonicalKeys |
| [internal/web/api/access_keys.go](../../../../internal/web/api/access_keys.go) | FingerprintSHA256; ParseAuthorizedKey | none | **role:** Product HTTP/WebSocket handler or adapter.; **chain:** internal/web/api/access_keys.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** nativeAccessFingerprints |
| [internal/web/auth/development_key.go](../../../../internal/web/auth/development_key.go) | FingerprintSHA256; MarshalAuthorizedKey; ParseAuthorizedKey | none | **role:** Web authorization/session facade.; **chain:** internal/web/auth/development_key.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** NormalizeDevelopmentKey |
| internal/web/api/access_keys.go; internal/web/auth/development_key.go; internal/web/auth/session.go; internal/web/auth/extension_service.go; internal/web/auth/forgejo_keys.go | ssh.ParsePublicKey; ssh.MarshalAuthorizedKey; ssh.FingerprintSHA256 | **role:** HTTP key management, development-key registration and extension public-key synchronization consume the parsed canonical key/fingerprint facade.; **public types:** project.AccessKey; web auth key result; **conversions:** HTTP key string -&gt; parsed public key/fingerprint -&gt; canonical storage/host request.; **production consumers:** [internal/web/api/access_keys.go](../../../../internal/web/api/access_keys.go); [internal/web/auth/development_key.go](../../../../internal/web/auth/development_key.go); [internal/web/auth/session.go](../../../../internal/web/auth/session.go); [internal/web/auth/extension_service.go](../../../../internal/web/auth/extension_service.go); [internal/web/auth/forgejo_keys.go](../../../../internal/web/auth/forgejo_keys.go); [internal/host/access_keys.go](../../../../internal/host/access_keys.go); [internal/web/api/environments_join.go](../../../../internal/web/api/environments_join.go) | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/web/forgejo_keys_test.go](../../../../internal/web/forgejo_keys_test.go) | FingerprintSHA256; MarshalAuthorizedKey; NewPublicKey | Web server composition or feature adapter. |
| [internal/web/lifecycle_access_keys_test.go](../../../../internal/web/lifecycle_access_keys_test.go) | FingerprintSHA256; MarshalAuthorizedKey; NewPublicKey | Web server composition or feature adapter. |

<a id="integration-5"></a>

## golang.org/x/oauth2

**Selected:** golang.org/x/oauth2 v0.34.0. **Upstream capability used:** OAuth2 token exchange via clientcredentials for Tailscale enrollment.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; direct Config.Token and OAuth HTTP client context in Tailnet enrollment; not a generalized web OAuth server.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| `internal/tailnet/control.go` (`../../../../internal/tailnet/control.go`; historical source locator) | AuthStyleInHeader; Config.Token; HTTPClient; clientcredentials.Config | none | **role:** Tailnet control/provider integration.; **chain:** internal/tailnet/control.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** checkCredential |
| `internal/tailnet/enrollment.go` (`../../../../internal/tailnet/enrollment.go`; historical source locator) | AuthStyleInHeader; Config.Token; HTTPClient; Token; clientcredentials.Config; oauth2.HTTPClient; clientcredentials.Config.Token | **role:** Exchanges restricted Tailscale OAuth client credentials for one operation-scoped bearer token.; **public types:** AuthKey; Enrollment; oauth2.Token; **conversions:** stored client credentials/policy tags -&gt; token request; token is checked then used by the key-specific SDK request and cleared. | **role:** Tailnet control/provider integration.; **chain:** internal/tailnet/enrollment.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** tokenFromEnrollment |

<a id="integration-6"></a>

## golang.org/x/sys/unix

**Selected:** golang.org/x/sys v0.47.0. **Upstream capability used:** Linux/Unix syscalls for locking, process/wait, fd/socket and policy file operations.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; direct unix API calls in Linux build-tagged adapters and host processes.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [cmd/soda-dashboard/operator_linux.go](../../../../cmd/soda-dashboard/operator_linux.go) | GetsockoptUcred; SOL_SOCKET; SO_PEERCRED | none | **role:** Process entrypoint or command adapter.; **chain:** cmd/soda-dashboard/operator_linux.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** operatorPeerOK |
| [internal/acceptance/process_wait_linux.go](../../../../internal/acceptance/process_wait_linux.go) | ECHILD; EINTR; P_PID; Siginfo; WEXITED; WNOWAIT; Wait4; WaitStatus; Waitid | none | **role:** Developer acceptance/evidence support integration.; **chain:** internal/acceptance/process_wait_linux.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** reapOwnedChildren; waitOwnedExit |
| [internal/factory/control/coordinator.go](../../../../internal/factory/control/coordinator.go) | LOCK_EX | none | **role:** Factory coordinator composition or runtime service.; **chain:** internal/factory/control/coordinator.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Start |
| [internal/filelock/filelock.go](../../../../internal/filelock/filelock.go) | EINTR; EWOULDBLOCK; Flock; LOCK_NB; unix.Flock | **role:** Cross-process advisory lock wrapper used for shared policy files.; **public types:** filelock.Lock | **role:** Cross-process lock adapter.; **chain:** internal/filelock/filelock.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** Acquire |
| `internal/tailnet/policy.go` (`../../../../internal/tailnet/policy.go`; historical source locator) | LOCK_EX | none | **role:** Tailnet control/provider integration.; **chain:** internal/tailnet/policy.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** acquirePolicyLock |
| cmd/soda-dashboard/operator_linux.go + internal/factory/control/coordinator.go + internal/acceptance/process_wait_linux.go + internal/tailnet/policy.go | Unix fd/process/socket/flock APIs | **role:** Linux operator IPC/process supervision, process wait, and secure policy-file operations.; **public types:** Linux-only helpers; **conversions:** OS error/credential/fd state -&gt; local error/typed admission. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/filelock/filelock_test.go](../../../../internal/filelock/filelock_test.go) | EBADF; Flock; LOCK_EX; LOCK_NB; LOCK_SH | Cross-process lock adapter. |
| [tests/build/project_account_test.go](../../../../tests/build/project_account_test.go) | Flock; LOCK_EX; LOCK_NB; LOCK_UN | Test/developer helper. |

<a id="integration-7"></a>

## tailscale.com/client/tailscale/v2

**Selected:** v2.10.1. **Upstream capability used:** Typed Tailscale API client for ephemeral auth-key create and key response fields.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; internal/tailnet/enrollment.go uses ts.Client/Keys().CreateAuthKey and validates result; custom transport constrains endpoint/redirect/body.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| `internal/tailnet/enrollment.go` (`../../../../internal/tailnet/enrollment.go`; historical source locator) | Client; CreateKeyRequest; Key; ts.Client; Client.Keys; Keys.CreateAuthKey | **role:** Creates an ephemeral preauthorized project auth key after one-time OAuth token exchange.; **public types:** AuthKey; Enrollment; ts.CreateKeyRequest; ts.Key; **conversions:** Soda policy/Tags -&gt; SDK capabilities; SDK returned key checked for identity/capabilities/lifetime, then secret is cleared after copying. | **role:** Tailnet control/provider integration.; **chain:** internal/tailnet/enrollment.go imports and invokes the listed upstream API; caller composition is described in the chapter.; **entrypoints:** createProjectAuthKey; validAuthKeyCapabilities; validAuthKeyIdentity; validAuthKeyLifetime; validCreatedAuthKey |
| internal/tailnet/control.go; internal/host/tailnet.go | TailnetOptions; ProjectTailnet operation facade | **role:** Soda tailnet policy and typed host client expose project enrollment options and one native run-key operation; credential creation is internal to tailnet Control.; **public types:** Client; ProjectOptions; ProjectTailnet; project.TailnetSelection; tailnet.ProjectOptions; **conversions:** Saved tailnet policy -&gt; validated SDK CreateKeyRequest; key bytes -&gt; restricted companion input. | none |
| internal/web/api/tailnet.go; internal/web/api/environments_create.go; cmd/soda-tailnet/main.go | apiTailnetOptions; project create preflight; status display command | **role:** Browser settings/project-create path and host CLI consume Soda Tailnet APIs/status; they do not call the Tailscale client package directly. | none |

<a id="integration-8"></a>

## modernc.org/sqlite

**Selected:** v1.58.0. **Upstream capability used:** SQLite database/sql driver registration. Current Soda use is the test-only staged fixture writer; the legacy dashboard reader was retired in `8b10ab14`.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** v1.58.0 selected in go.mod. Current source has the SQLite import in internal/factory/control/staged_seed_test.go. seedStagedDependencyEdge has two tracked factory test callers; the former acceptance reader/fixtures are absent.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/factory/control/staged_seed_test.go](../../../../internal/factory/control/staged_seed_test.go) | blank import registers database/sql driver "sqlite"; seedStagedDependencyEdge; blank import driver registration | **role:** Private test-package helper for writing one staged Forgejo SQLite fixture edge.; **test helper:** seedStagedDependencyEdge; **conversions:** test fixture edge arguments -&gt; external staged SQLite issue_dependency row; **test callers:** internal/factory/control/st15_demo_seed_test.go::(*st15Fixture).insertEdge; internal/factory/control/merge_native_setup_test.go::nativeMergeInsertEdge | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/factory/control/st15_demo_seed_test.go](../../../../internal/factory/control/st15_demo_seed_test.go) | (*st15Fixture).insertEdge -&gt; seedStagedDependencyEdge | Current test-only caller of staged fixture writer. |
| [internal/factory/control/merge_native_setup_test.go](../../../../internal/factory/control/merge_native_setup_test.go) | nativeMergeInsertEdge -&gt; seedStagedDependencyEdge | Current test-only caller of staged fixture writer. |

**Transfer / predecessor evidence:** No product SQLite persistence path identified. The staged writer is test-only. The former lifecycle-state temporary-copy reader had no current installed journey consumer and was retired with its exclusive tests/CLI selector; current persistence qualification remains separate.

**developer consumers:** No current acceptance SQLite consumer. The two control test fixture callers above retain the selected module.

<a id="integration-9"></a>

## github.com/jackc/pgx/v5

**Selected:** v5.10.0. **Upstream capability used:** PostgreSQL database/sql driver and pgconn typed errors.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below. Root go.mod SHA-256 dcc2232e5710147b5b7859960887c6b51c30428b0002f0b9e6e673fc483ced4a; go.sum SHA-256 8bee84792c2aca81c0a8398a24b74355794a2622b3d198113f95cbc1ad5f5d1e.

**Primary-source evidence:** Selected module version in go.mod indirect block despite direct imports; store.Open calls sql.Open("pgx") through stdlib driver; dispatch packet inspects pgconn errors.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/store/store.go](../../../../internal/store/store.go) | stdlib blank-import registers database/sql driver name "pgx"; sql.Open("pgx", dsn); blank import github.com/jackc/pgx/v5/stdlib; Store; Open; OpenEncrypted | **role:** Canonical production database/sql Store facade registers pgx and opens primary database pools.; **public types:** store.Store; **conversions:** PostgreSQL DSN -&gt; database/sql DB pool; pgx stdlib driver adapts to database/sql.; **composition:** cmd/soda-dashboard/main.go::openDashboard -&gt; store.OpenEncrypted -&gt; internal/web/server.go::New -&gt; internal/factory/control/coordinator.go::NewCoordinator; web.New also injects Store into API/auth. | **role:** PostgreSQL driver registration and Store.Open adapter.; **chain:** cmd/soda-dashboard main opens Store; internal/web.Server injects Store into API and coordinator consumers.; **entrypoints:** Store.open |
| [internal/store/factory_dispatch_packet.go](../../../../internal/store/factory_dispatch_packet.go) | pgconn.PgError; dispatchPacketError | **role:** Production adapter maps PostgreSQL constraint error metadata to Store dispatch errors.; **public types:** store.ErrAssignmentActive; **conversions:** pgconn.PgError constraint metadata -&gt; Soda store error.; **composition:** factory coordinator dispatch -&gt; Store transaction writer -&gt; dispatchPacketError | **role:** Maps the selected pgx driver error shape to PostgreSQL dispatch constraint/error handling.; **chain:** Store transaction writer -&gt; factory coordinator dispatch admission.; **entrypoints:** dispatchPacketError |
| cmd/soda-dashboard/main.go; internal/web/server.go; internal/factory/control/coordinator.go | openDashboard; web.New; control.NewCoordinator | none | **role:** Concrete production composition and Store injection root.; **chain:** cmd/soda-dashboard/main.go::openDashboard -&gt; store.OpenEncrypted -&gt; internal/web/server.go::New -&gt; internal/factory/control/coordinator.go::NewCoordinator; web.New also injects Store into API/auth.; **entrypoints:** openDashboard; New; NewCoordinator |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/store/store_test.go](../../../../internal/store/store_test.go) | TestSetupSocketDSNParses; pgx.ParseConfig@71 | Test-only DSN parser assertion. This file does not use pgx.Connect, pgx.Conn, or pgx.Tx. |

**store facade consumers:** **path:** [cmd/soda-dashboard/main.go](../../../../cmd/soda-dashboard/main.go); **symbols:** openDashboard -&gt; store.OpenEncrypted; **role:** Production Store facade caller; **path:** [internal/factory/control/acceptance.go](../../../../internal/factory/control/acceptance.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/acceptance_evidence.go](../../../../internal/factory/control/acceptance_evidence.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/acceptance_initial.go](../../../../internal/factory/control/acceptance_initial.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/acceptance_status.go](../../../../internal/factory/control/acceptance_status.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/checks_pass.go](../../../../internal/factory/control/checks_pass.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/coordinator.go](../../../../internal/factory/control/coordinator.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/correction.go](../../../../internal/factory/control/correction.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/cycles.go](../../../../internal/factory/control/cycles.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch.go](../../../../internal/factory/control/dispatch.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_attempt.go](../../../../internal/factory/control/dispatch_attempt.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_launch.go](../../../../internal/factory/control/dispatch_launch.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_occupancy.go](../../../../internal/factory/control/dispatch_occupancy.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_recovery.go](../../../../internal/factory/control/dispatch_recovery.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_registration.go](../../../../internal/factory/control/dispatch_registration.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_result.go](../../../../internal/factory/control/dispatch_result.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/dispatch_selection.go](../../../../internal/factory/control/dispatch_selection.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/grant_authority.go](../../../../internal/factory/control/grant_authority.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/grants.go](../../../../internal/factory/control/grants.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/lifecycle.go](../../../../internal/factory/control/lifecycle.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/lifecycle_retry.go](../../../../internal/factory/control/lifecycle_retry.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/lifecycle_takeover.go](../../../../internal/factory/control/lifecycle_takeover.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/merge.go](../../../../internal/factory/control/merge.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/merge_evidence.go](../../../../internal/factory/control/merge_evidence.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/merge_reconcile.go](../../../../internal/factory/control/merge_reconcile.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/merge_withdraw.go](../../../../internal/factory/control/merge_withdraw.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/preparation_decisions.go](../../../../internal/factory/control/preparation_decisions.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/prerequisites.go](../../../../internal/factory/control/prerequisites.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/publication.go](../../../../internal/factory/control/publication.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/publication_authority.go](../../../../internal/factory/control/publication_authority.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/publication_reconcile.go](../../../../internal/factory/control/publication_reconcile.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/publication_withdraw.go](../../../../internal/factory/control/publication_withdraw.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/readiness.go](../../../../internal/factory/control/readiness.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/readiness_sweep.go](../../../../internal/factory/control/readiness_sweep.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/review_cycle.go](../../../../internal/factory/control/review_cycle.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/factory/control/settle.go](../../../../internal/factory/control/settle.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/web/api/access_keys.go](../../../../internal/web/api/access_keys.go); **symbols:** Keys; MemberLogin; **role:** Production Store facade caller; **path:** [internal/web/api/api.go](../../../../internal/web/api/api.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/web/api/environment_authority.go](../../../../internal/web/api/environment_authority.go); **symbols:** MemberLogin; **role:** Production Store facade caller; **path:** [internal/web/api/environments_api.go](../../../../internal/web/api/environments_api.go); **symbols:** MemberLogin; Members; Project; ProjectByRepository; **role:** Production Store facade caller; **path:** [internal/web/api/environments_create.go](../../../../internal/web/api/environments_create.go); **symbols:** CreateProject; MarkReady; ProjectByRepository; **role:** Production Store facade caller; **path:** [internal/web/api/environments_join.go](../../../../internal/web/api/environments_join.go); **symbols:** Join; Keys; MemberLogin; **role:** Production Store facade caller; **path:** [internal/web/api/environments_preparation.go](../../../../internal/web/api/environments_preparation.go); **symbols:** ApprovalHead; MaintenanceHold; ProjectPreparations; RequirementHead; SaveMaintenanceHold; **role:** Production Store facade caller; **path:** [internal/web/api/extension_native.go](../../../../internal/web/api/extension_native.go); **symbols:** UpsertUser; User; **role:** Production Store facade caller; **path:** [internal/web/api/extension_terminal.go](../../../../internal/web/api/extension_terminal.go); **symbols:** MemberLogin; Project; **role:** Production Store facade caller; **path:** [internal/web/api/extension_terminal_authority.go](../../../../internal/web/api/extension_terminal_authority.go); **symbols:** MemberLogin; Project; **role:** Production Store facade caller; **path:** [internal/web/api/factory_assignments.go](../../../../internal/web/api/factory_assignments.go); **symbols:** IssueAssignments; PublicationByAssignment; Reservation; **role:** Production Store facade caller; **path:** [internal/web/api/factory_issue_view.go](../../../../internal/web/api/factory_issue_view.go); **symbols:** CheckAssessment; IssueControl; MergeForIssue; **role:** Production Store facade caller; **path:** [internal/web/api/factory_lifecycle.go](../../../../internal/web/api/factory_lifecycle.go); **symbols:** FactoryCommand; FactoryRun; MemberLogin; Project; **role:** Production Store facade caller; **path:** [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go); **symbols:** FactoryRun; Project; **role:** Production Store facade caller; **path:** [internal/web/api/factory_status.go](../../../../internal/web/api/factory_status.go); **symbols:** ApprovalHead; Capacity; EnvironmentGrant; IssueControls; OperatorGrant; ProjectByRepository; ProjectPreparations; RepositoryPolicy; RequirementHead; Sponsorships; **role:** Production Store facade caller; **path:** [internal/web/api/factory_views.go](../../../../internal/web/api/factory_views.go); **symbols:** FactoryRunView; **role:** Production Store facade caller; **path:** [internal/web/api/identity_grants.go](../../../../internal/web/api/identity_grants.go); **symbols:** MemberLogin; **role:** Production Store facade caller; **path:** [internal/web/api/identity_launch.go](../../../../internal/web/api/identity_launch.go); **symbols:** MemberLogin; **role:** Production Store facade caller; **path:** [internal/web/api/lifecycle.go](../../../../internal/web/api/lifecycle.go); **symbols:** MaintenanceHold; SaveMaintenanceHold; **role:** Production Store facade caller; **path:** [internal/web/api/preparation_decisions.go](../../../../internal/web/api/preparation_decisions.go); **symbols:** ApprovalHead; RequirementHead; **role:** Production Store facade caller; **path:** [internal/web/api/provisioning.go](../../../../internal/web/api/provisioning.go); **symbols:** MarkReady; **role:** Production Store facade caller; **path:** [internal/web/api/repositories.go](../../../../internal/web/api/repositories.go); **symbols:** ProjectByRepository; **role:** Production Store facade caller; **path:** [internal/web/api/spaces_authority.go](../../../../internal/web/api/spaces_authority.go); **symbols:** DispatchState; RepositoryPolicy; **role:** Production Store facade caller; **path:** [internal/web/api/spaces_inventory.go](../../../../internal/web/api/spaces_inventory.go); **symbols:** FactoryRunViews; FactoryRuns; FactoryUnsettledRuns; SpaceProjectsAfter; **role:** Production Store facade caller; **path:** [internal/web/api/tailnet.go](../../../../internal/web/api/tailnet.go); **symbols:** MemberLogin; **role:** Production Store facade caller; **path:** [internal/web/auth/extension_service.go](../../../../internal/web/auth/extension_service.go); **symbols:** UpsertUser; User; **role:** Production Store facade caller; **path:** [internal/web/auth/service.go](../../../../internal/web/auth/service.go); **symbols:** Store facade/database consumer; **role:** Production Store facade caller; **path:** [internal/web/auth/session.go](../../../../internal/web/auth/session.go); **symbols:** AddKey; Keys; RemoveKey; RenameProfile; **role:** Production Store facade caller; **path:** [internal/web/server.go](../../../../internal/web/server.go); **symbols:** New -&gt; auth.New, api.New, control.NewCoordinator; passes Store; **role:** Production Store facade caller

**unreached or test support:** **path:** [internal/store/observe.go](../../../../internal/store/observe.go); **symbols:** OpenObserve; ReadSchemaVersion; **role:** Compiled direct pgx driver user; tracked callers are internal/store/observe_test.go and internal/store/schema_test.go, with no current product source caller.; **path:** [internal/store/ephemeral.go](../../../../internal/store/ephemeral.go); **symbols:** createEphemeralDatabase; sql.Open("pgx", superDSN); **role:** Test fixture create/drop helper; called by *_test.go only.; **path:** [internal/store/corruption.go](../../../../internal/store/corruption.go); **symbols:** InjectCorruptFactoryRun; InjectCorruptFactoryOutput; InjectCorruptFactoryStatus; **role:** Source marks helpers test-only; no product source caller.; **path:** [internal/store/identity_fixture.go](../../../../internal/store/identity_fixture.go); **symbols:** SeedIdentityConnection; SeedIdentityGrant; **role:** Source marks helpers test-only; no product source caller.

**sql open driver users:** **path:** [internal/store/store.go](../../../../internal/store/store.go); **symbols:** Open -&gt; sql.Open("pgx", dsn); **role:** Production primary Store database open.; **path:** [internal/store/observe.go](../../../../internal/store/observe.go); **symbols:** openReadOnly -&gt; sql.Open("pgx", dsn); OpenObserve; ReadSchemaVersion; **role:** Compiled API with current callers only in tracked tests; no current product source reachability.; **path:** [internal/store/ephemeral.go](../../../../internal/store/ephemeral.go); **symbols:** createEphemeralDatabase -&gt; sql.Open("pgx", superDSN); **role:** Test-only ephemeral database fixture.; **path:** [internal/store/corruption.go](../../../../internal/store/corruption.go); **symbols:** sql.Open("pgx", dsn); **role:** Test-only corruption injection helpers.

**store facade consumer note:** Complete tracked production Go source file set found by Store facade type/open/member callsites; excludes *_test.go and test-only helpers explicitly identified below.

<a id="integration-10"></a>

## github.com/stretchr/testify

**Selected:** v1.12.1. **Upstream capability used:** Test assertions only; no production imports.

**Dependency admission:** go.mod selected module requirement; actual import sites listed below.

**Primary-source evidence:** Selected module version in go.mod; direct testify/require uses in tests and *_test.go files only.

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [cmd/soda-tailnet/command_test.go](../../../../cmd/soda-tailnet/command_test.go) | Contains; ErrorIs; NoError; NotContains; True | Test assertions only; no production imports. |
| [internal/filelock/filelock_test.go](../../../../internal/filelock/filelock_test.go) | ErrorIs; NoError | Test assertions only; no production imports. |
| [internal/strictjson/decode_test.go](../../../../internal/strictjson/decode_test.go) | Equal; Error; ErrorContains; NoError | Test assertions only; no production imports. |
| [internal/tailnet/tailnet_test.go](../../../../internal/tailnet/tailnet_test.go) | Empty; Equal; ErrorAs; ErrorContains; ErrorIs; Less; NoError; True | Test assertions only; no production imports. |
| [internal/web/environment_read_publication_test.go](../../../../internal/web/environment_read_publication_test.go) | Contains; Equal; False; Len; Nil; NoError; NotContains; True | Test assertions only; no production imports. |
| [internal/web/execution_access_test.go](../../../../internal/web/execution_access_test.go) | Contains; Empty; Equal; ErrorIs; False; NoError; True | Test assertions only; no production imports. |
| [scripts/render_forgejo_branding_test.go](../../../../scripts/render_forgejo_branding_test.go) | Contains; Equal; Error; NoError | Test assertions only; no production imports. |
| [scripts/render_forgejo_native_test.go](../../../../scripts/render_forgejo_native_test.go) | NoErrorf | Test assertions only; no production imports. |
| [tools/png-equal/main_test.go](../../../../tools/png-equal/main_test.go) | False; NoError; True | Test assertions only; no production imports. |
| [tools/soda-rootfs-server/main_test.go](../../../../tools/soda-rootfs-server/main_test.go) | Contains; Empty; Equal; Len; NoError; NotContains; NotEqual; NotRegexp; True | Test assertions only; no production imports. |

<a id="integration-11"></a>

## Go standard library net/http

**Selected:** Go 1.26.7 module language/toolchain floor (go.mod). **Upstream capability used:** HTTP client/server, middleware, callback transports and request/response framing used by product routes and native/local clients.

**Dependency admission:** go.mod `go 1.26.7`; no external module version applies.

**Primary-source evidence:** Standard-library direct import inventory; consumers use net/http APIs for in-process handlers, private Unix transports, HTTPS/TLS API clients and timeouts.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [cmd/soda-dashboard/extension.go](../../../../cmd/soda-dashboard/extension.go) | Handler; Server | none | **role:** Process entrypoint or command adapter.; **entrypoints:** extensionListener; **chain:** Process entrypoint or command adapter. |
| [cmd/soda-dashboard/main.go](../../../../cmd/soda-dashboard/main.go) | ErrServerClosed; Server | none | **role:** Process entrypoint or command adapter.; **entrypoints:** run; serveExtensionSocket; serveOperatorSocket; **chain:** Process entrypoint or command adapter. |
| [cmd/soda-dashboard/operator.go](../../../../cmd/soda-dashboard/operator.go) | HandlerFunc; Request; ResponseWriter; Server | none | **role:** Process entrypoint or command adapter.; **entrypoints:** operatorHTTPServer; **chain:** Process entrypoint or command adapter. |
| [internal/acceptance/service_https.go](../../../../internal/acceptance/service_https.go) | Client; ErrUseLastResponse; Request; Transport | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** httpsClient; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_access.go](../../../../internal/acceptance/workload_access.go) | Client; Request; Transport | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** fetchAccessURL; **chain:** Developer acceptance/evidence support integration. |
| [internal/factory/control/operator.go](../../../../internal/factory/control/operator.go) | Error; Handler; MethodPost; NewServeMux; Request; ResponseWriter; StatusAccepted; StatusBadRequest; StatusConflict; StatusInternalServerError; StatusMethodNotAllowed; StatusNotFound | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** OperatorHandler; serveOperator; serveReconcile; serveStatus; serveStop; writeCommandError; **chain:** Factory coordinator composition or runtime service. |
| [internal/forgejo/background_admission.go](../../../../internal/forgejo/background_admission.go) | Client; MethodPost; NewRequestWithContext; StatusOK; Transport | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** bootstrap; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/background_transport.go](../../../../internal/forgejo/background_transport.go) | Client; MethodPost; NewRequestWithContext; StatusOK; StatusUnauthorized; Transport | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** call; post; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/checks.go](../../../../internal/forgejo/checks.go) | StatusServiceUnavailable | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** checkReadError; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/client.go](../../../../internal/forgejo/client.go) | Client; ErrUseLastResponse; NewRequestWithContext; Request | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** New; request; transportError; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/host/client.go](../../../../internal/host/client.go) | Client; ErrUseLastResponse; NewRequestWithContext; Request; Response; Transport | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** NewClient; callLimit; **chain:** Typed native host client or privileged development executor. |
| [internal/host/factory_candidate.go](../../../../internal/host/factory_candidate.go) | StatusConflict | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** FactoryInspectCandidate; **chain:** Typed native host client or privileged development executor. |
| [internal/host/factory_client.go](../../../../internal/host/factory_client.go) | StatusConflict; StatusNotFound; StatusRequestEntityTooLarge; StatusUnprocessableEntity | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** factoryExportError; factoryNotFound; factoryOutputError; **chain:** Typed native host client or privileged development executor. |
| [internal/identity/client/client.go](../../../../internal/identity/client/client.go) | Client; ErrUseLastResponse; NewRequestWithContext; Request; StatusOK; Transport | none | **role:** Direct package consumer.; **entrypoints:** New; call; **chain:** Direct package consumer. |
| `internal/tailnet/control.go` (`../../../../internal/tailnet/control.go`; historical source locator) | Client; DefaultTransport; NewRequestWithContext; Request; Response; RoundTripper; Transport | none | **role:** Tailnet control/provider integration.; **entrypoints:** NewControl; request; **chain:** Tailnet control/provider integration. |
| `internal/tailnet/enrollment.go` (`../../../../internal/tailnet/enrollment.go`; historical source locator) | Client; DefaultTransport; Request; Response; RoundTripper | none | **role:** Tailnet control/provider integration.; **entrypoints:** createProjectAuthKey; **chain:** Tailnet control/provider integration. |
| [internal/web/api/access_keys.go](../../../../internal/web/api/access_keys.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** accessKeysEnvironment; apiAccessKeys; applyAccessKeyMutation; authorizeAccessKeysMember; nativeAccessFingerprints; savedAccessKeyMaterial; validAccessKeyConfirmation; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/api.go](../../../../internal/web/api/api.go) | Request | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** New; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environment_authority.go](../../../../internal/web/api/environment_authority.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** authorizeEnvironmentRead; environmentAdministrator; executionRepository; nativeVisibleRepository; readEnvironmentAuthority; reportExecutionAuthorityError; repositoryExecutionAllowed; visibleRepository; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environment_os.go](../../../../internal/web/api/environment_os.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiEnvironmentOS; confirmOSSession; rejectOSQuery; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environments_api.go](../../../../internal/web/api/environments_api.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** EnvironmentDTO; apiConnection; apiEnvironment; apiEnvironmentMembers; apiEnvironments; environmentListQuery; environmentProfileMismatch; listedEnvironments; loadEnvironment; observedEnvironment; parseEnvironmentsListQuery; requireListedSession; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environments_create.go](../../../../internal/web/api/environments_create.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiCreateEnvironment; applyCreatedEnvironmentTailnet; checkTailnetPreflight; lookupReservation; parseCreateEnvironmentInput; precheckProfileAndTailnet; provisionAndSaveProject; reconcileCreate; reconfirmRepositoryAndSession; resolveNativeProfile; verifyCurrentSessionMatch; verifyRepositoryOwner; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environments_join.go](../../../../internal/web/api/environments_join.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** admitNewJoin; apiJoinEnvironment; currentJoinSession; joinPublicKeys; persistEnvironmentJoin; reportJoinPersist; validJoinSSHSelection; writeJoinLogin; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environments_preparation.go](../../../../internal/web/api/environments_preparation.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiPreparation; apiPreparationHold; inspectSpacePreparation; preparationHoldState; preparationItemDTO; preparationItems; summarizeSpacePreparation; syncPreparationHold; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension.go](../../../../internal/web/api/extension.go) | Handler; HandlerFunc; Header; MaxBytesReader; MethodDelete; MethodGet; MethodPatch; MethodPost; MethodPut; NotFound; Request; Response; ResponseWriter; StatusBadGateway; StatusForbidden; StatusServiceUnavailable; StatusSwitchingProtocols; Transport | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** Extension; extensionDevelopmentKeyRoute; extensionEnvironmentActionRoute; extensionEnvironmentIdentityRoute; extensionEnvironmentPreparationPath; extensionEnvironmentRoute; extensionFactoryOutputStreamRoute; extensionFactoryRoute; extensionIdentityActionRoute; extensionIdentityConnectionsRoute; extensionIdentityEnrollmentsRoute; extensionMeRoute; extensionProductRoute; extensionRepositoryRoute; extensionSessionRoute; extensionTailnetSettingsRoute; extensionTerminalSessionRoute; extensionTerminalStreamRoute; rewriteExtensionRequest; terminalControlMethod; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_native.go](../../../../internal/web/api/extension_native.go) | HandlerFunc; MaxBytesReader; MethodGet; MethodHead; MethodPost; MethodPut; Request; ResponseWriter; ServeMux; StatusConflict; StatusForbidden; StatusServiceUnavailable | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** extensionGenerationCurrent; extensionProductActor; extensionProductAuthority; extensionProductMutation; extensionProductUser; extensionProtected; registerExtensionProductRoutes; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal.go](../../../../internal/web/api/extension_terminal.go) | Handler; HandlerFunc; MaxBytesReader; MethodGet; MethodPost; NewServeMux; NotFound; Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** ExtensionHandler; extensionReserveTerminal; extensionTerminalOperation; extensionTerminalSessionMethod; privateTerminalStreamRoute; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal_authority.go](../../../../internal/web/api/extension_terminal_authority.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** extensionTerminalAccount; extensionTerminalCurrent; extensionTerminalGeneration; extensionTerminalOrigin; extensionTerminalRepository; nativeTerminalActor; nativeTerminalContribution; nativeTerminalMember; nativeTerminalMembershipCurrent; nativeTerminalScope; nativeTerminalSessionActor; sameNativeTerminalAuthority; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal_stream.go](../../../../internal/web/api/extension_terminal_stream.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** registerExtensionTerminalPeer; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_assignments.go](../../../../internal/web/api/factory_assignments.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryAssignment; currentIssueAssignment; factoryAssignmentDTO; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_intake.go](../../../../internal/web/api/factory_intake.go) | MethodPost; Request; ResponseWriter; StatusBadRequest; StatusMethodNotAllowed; StatusOK; StatusRequestEntityTooLarge; StatusServiceUnavailable; StatusUnauthorized | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** ServeHTTP; parseIntakeHint; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_issue_view.go](../../../../internal/web/api/factory_issue_view.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryIssue; checksViewDTO; mergeViewDTO; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_lifecycle.go](../../../../internal/web/api/factory_lifecycle.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryActions; apiFactoryCommand; apiFactoryRunActions; commandRepository; factoryActionRun; lifecycleControlError; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go) | MethodGet; Request; ResponseWriter; StatusForbidden; StatusMethodNotAllowed | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** checkFactoryOutputHeaders; privateFactoryOutputRoute; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_policy.go](../../../../internal/web/api/factory_policy.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryCapacity; apiFactoryOperatorGrant; apiFactoryPolicy; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_settings.go](../../../../internal/web/api/factory_settings.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** factoryCommandError; factoryOperator; factoryOwner; factoryPrincipal; factoryRepository; resolve; settingsCommandID; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_sponsorship.go](../../../../internal/web/api/factory_sponsorship.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryEnvironmentGrant; apiFactorySponsorship; checkSponsorshipBroker; sponsorshipView; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_status.go](../../../../internal/web/api/factory_status.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryStatus; factoryPreparationStatus; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_views.go](../../../../internal/web/api/factory_views.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryRun; factoryHostTerminal; factoryRunBindingDTO; factoryRunLiveDTO; factoryRunRecordDTO; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/identity.go](../../../../internal/web/api/identity.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiIdentityCancelEnrollment; apiIdentityConnections; apiIdentityEndLease; apiIdentityEnrollment; apiIdentityGrants; apiIdentityLeases; apiIdentityRevoke; apiIdentityRevokeGrant; apiIdentityStartEnrollment; identityAdmission; identityError; identityMutation; identityResult; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/identity_grants.go](../../../../internal/web/api/identity_grants.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiIdentityAvailable; apiIdentityCreateGrant; identityNamedMember; identityProjectMember; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/identity_launch.go](../../../../internal/web/api/identity_launch.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiIdentityLaunch; identityLaunchInput; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/issue_acceptances.go](../../../../internal/web/api/issue_acceptances.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** acceptanceDecision; acceptanceDecisionError; acceptanceRefusalError; acceptanceRefusalMessage; acceptanceSources; apiIssueAcceptances; apiIssueWithdrawal; factoryIssue; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/lifecycle.go](../../../../internal/web/api/lifecycle.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** acquireLifecycleStop; apiLifecycle; apiLifecycleStart; apiLifecycleStop; authorizeLifecycleOperator; checkLifecycleSession; decodeLifecycleRequest; handleLifecycleMutation; lifecycleStopHold; loadLifecycleEnvironment; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/operator.go](../../../../internal/web/api/operator.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** authorizeOperator; operatorAuthorization; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/preparation_decisions.go](../../../../internal/web/api/preparation_decisions.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiPreparationAcceptances; apiPreparationActionApprove; apiPreparationActionHold; apiPreparationActionInspect; apiPreparationActions; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/project_profiles.go](../../../../internal/web/api/project_profiles.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiProjectProfiles; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/repositories.go](../../../../internal/web/api/repositories.go) | Request; ResponseWriter; StatusBadRequest; StatusForbidden; StatusOK; StatusServiceUnavailable | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiNativeRepositories; apiRepositories; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/spaces.go](../../../../internal/web/api/spaces.go) | Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiSpaces; appendSpaceRow; inspectSpaces; parseSpacesCursor; verifySpacesSession; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/spaces_authority.go](../../../../internal/web/api/spaces_authority.go) | Request | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** inspectSpaceAuthority; inspectSpaceControl; resolveSpaceAuthority; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/spaces_inspection.go](../../../../internal/web/api/spaces_inspection.go) | Request | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** inspectSpaceFactoryRuns; inspectSpaceNative; inspectSpaceRow; inspectSpaceTailnet; inspectSpaceTerminals; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/tailnet.go](../../../../internal/web/api/tailnet.go) | MethodGet; MethodPost; Request; ResponseWriter | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** authorizeProjectTailnet; parseProjectTailnetMutation; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/terminal_registry.go](../../../../internal/web/api/terminal_registry.go) | MethodGet; Request; ResponseWriter; StatusBadRequest; StatusForbidden; StatusMethodNotAllowed; StatusOK | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** checkTerminalRequestHeaders; parseTerminalSessionAction; terminalMetadata; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/auth/errors.go](../../../../internal/web/auth/errors.go) | ResponseWriter; StatusConflict; StatusForbidden; StatusNotFound; StatusServiceUnavailable; StatusTooManyRequests; StatusUnauthorized | none | **role:** Web authorization/session facade.; **entrypoints:** ProviderError; **chain:** Web authorization/session facade. |
| [internal/web/auth/extension.go](../../../../internal/web/auth/extension.go) | Request | none | **role:** Web authorization/session facade.; **entrypoints:** ExtensionAuthority; ExtensionContribution; **chain:** Web authorization/session facade. |
| [internal/web/auth/extension_service.go](../../../../internal/web/auth/extension_service.go) | Handler; HandlerFunc; MaxBytesReader; MethodDelete; MethodGet; MethodPatch; MethodPost; NotFound; Request; ResponseWriter; StatusBadRequest; StatusConflict; StatusForbidden; StatusOK; StatusServiceUnavailable; StatusUnsupportedMediaType | none | **role:** Web authorization/session facade.; **entrypoints:** ExtensionHandler; allowExtensionMethod; allowExtensionNoQuery; apiExtensionForgejoKeys; extensionIdentity; extensionMutation; extensionUser; revalidateExtensionAuthority; serveExtension; serveExtensionAccount; serveExtensionDevelopmentKeys; serveExtensionResource; writeExtensionSession; **chain:** Web authorization/session facade. |
| [internal/web/auth/http.go](../../../../internal/web/auth/http.go) | MaxBytesError; Request; ResponseWriter; StatusBadRequest; StatusInternalServerError; StatusMethodNotAllowed; StatusRequestEntityTooLarge | none | **role:** Web authorization/session facade.; **entrypoints:** AllowAPIMethod; DecodeAPIObject; JSONResponse; **chain:** Web authorization/session facade. |
| [internal/web/auth/session.go](../../../../internal/web/auth/session.go) | MethodGet; MethodPost; Request; ResponseWriter; StatusBadRequest; StatusOK; StatusServiceUnavailable | none | **role:** Web authorization/session facade.; **entrypoints:** apiKeys; preferences; **chain:** Web authorization/session facade. |
| [internal/web/avatars.go](../../../../internal/web/avatars.go) | MethodGet; MethodHead; Request; ResponseWriter; ServeContent; StatusBadRequest; StatusMethodNotAllowed; StatusNotFound; StatusServiceUnavailable | none | **role:** Web server composition or feature adapter.; **entrypoints:** ServeHTTP; admitAvatarQuery; admitAvatarRoute; avatarError; **chain:** Web server composition or feature adapter. |
| [internal/web/extension.go](../../../../internal/web/extension.go) | Handler | none | **role:** Web server composition or feature adapter.; **entrypoints:** Extension; ExtensionHandler; forgejoUsernamePolicy; **chain:** Web server composition or feature adapter. |
| [internal/web/server.go](../../../../internal/web/server.go) | Error; NewServeMux; NotFound; Redirect; Request; ResponseWriter; ServeMux; StatusNotFound; StatusSeeOther; StatusServiceUnavailable | none | **role:** Web server composition or feature adapter.; **entrypoints:** New; ServeHTTP; forgejoHome; **chain:** Web server composition or feature adapter. |
| [tools/soda-rootfs-server/main.go](../../../../tools/soda-rootfs-server/main.go) | Error; HandlerFunc; ListenAndServe; MethodGet; MethodHead; NotFound; Request; ResponseWriter; StatusNotImplemented; StatusOK; server | none | **role:** Developer/support command.; **entrypoints:** run; serveRootfs; **chain:** Developer/support command. |
| source consumers listed below | none | **role:** Soda-owned functions/types define policy; standard package supplies primitive transport, serialization, encoding, time or cryptographic operations. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [cmd/soda-dashboard/extension_test.go](../../../../cmd/soda-dashboard/extension_test.go) | NotFoundHandler | Process entrypoint or command adapter. |
| [cmd/soda-dashboard/operator_linux_test.go](../../../../cmd/soda-dashboard/operator_linux_test.go) | Client; MethodPost; NewRequest; Response; StatusOK; Transport | Process entrypoint or command adapter. |
| [internal/acceptance/service_https_test.go](../../../../internal/acceptance/service_https_test.go) | HandlerFunc; Redirect; Request; ResponseWriter; StatusFound; StatusNotFound; StatusOK | Developer acceptance/evidence support integration. |
| [internal/avatar/avatar_test.go](../../../../internal/avatar/avatar_test.go) | DefaultTransport; Request; Response | Avatar rendering facade. |
| [internal/factory/control/checks_native_fixture_test.go](../../../../internal/factory/control/checks_native_fixture_test.go) | Client; ErrUseLastResponse; MethodGet; MethodPost; NewRequestWithContext; Request; StatusCreated; StatusOK; StatusServiceUnavailable | Factory coordinator composition or runtime service. |
| [internal/factory/control/checks_native_stale_test.go](../../../../internal/factory/control/checks_native_stale_test.go) | MethodGet | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_completion_test.go](../../../../internal/factory/control/merge_native_completion_test.go) | MethodGet | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_effect_test.go](../../../../internal/factory/control/merge_native_effect_test.go) | MethodDelete; MethodGet; MethodPost | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_fixture_test.go](../../../../internal/factory/control/merge_native_fixture_test.go) | StatusServiceUnavailable | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_setup_test.go](../../../../internal/factory/control/merge_native_setup_test.go) | MethodPost | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_test.go](../../../../internal/factory/control/merge_native_test.go) | Client; ErrUseLastResponse; MethodGet; MethodPost; NewRequestWithContext; Request; StatusServiceUnavailable | Factory coordinator composition or runtime service. |
| [internal/factory/control/operator_test.go](../../../../internal/factory/control/operator_test.go) | MethodGet; MethodPost; StatusAccepted; StatusBadRequest; StatusConflict; StatusInternalServerError; StatusMethodNotAllowed; StatusNotFound; StatusOK | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_fixture_test.go](../../../../internal/factory/control/publication_native_fixture_test.go) | Client; ErrUseLastResponse; MethodPost; NewRequestWithContext; Request; followRedirects | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_test.go](../../../../internal/factory/control/publication_native_test.go) | MethodGet; MethodPost | Factory coordinator composition or runtime service. |
| [internal/factory/control/review_native_primitive_test.go](../../../../internal/factory/control/review_native_primitive_test.go) | MethodGet; MethodPost; StatusForbidden | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_coordinator_test.go](../../../../internal/factory/control/st15_demo_coordinator_test.go) | MethodPost; StatusOK | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_review_test.go](../../../../internal/factory/control/st15_demo_review_test.go) | DefaultClient; MethodGet; NewRequestWithContext; StatusNotFound; StatusOK | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_runs_test.go](../../../../internal/factory/control/st15_demo_runs_test.go) | MethodPost | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_seed_test.go](../../../../internal/factory/control/st15_demo_seed_test.go) | MethodGet; MethodPost | Factory coordinator composition or runtime service. |
| [internal/forgejo/background_test.go](../../../../internal/forgejo/background_test.go) | Error; Handler; HandlerFunc; NotFound; Request; ResponseWriter; Server; StatusBadRequest; StatusConflict; StatusUnauthorized | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/checks_test.go](../../../../internal/forgejo/checks_test.go) | StatusInternalServerError; StatusServiceUnavailable | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/client_test.go](../../../../internal/forgejo/client_test.go) | HandlerFunc; Redirect; Request; ResponseWriter; StatusFound | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/errors_test.go](../../../../internal/forgejo/errors_test.go) | HandlerFunc; Request; ResponseWriter | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/observe_test.go](../../../../internal/forgejo/observe_test.go) | Error; Handler; HandlerFunc; NotFound; Request; ResponseWriter; Server; StatusBadRequest; StatusForbidden | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/review_test.go](../../../../internal/forgejo/review_test.go) | Error; HandlerFunc; Request; ResponseWriter; Server; StatusInternalServerError | Forgejo SDK/client adapter and domain conversion. |
| [internal/host/factory_candidate_test.go](../../../../internal/host/factory_candidate_test.go) | Client; Request; Response; StatusConflict; StatusNotFound | Typed native host client or privileged development executor. |
| [internal/host/factory_export_test.go](../../../../internal/host/factory_export_test.go) | Header; MethodPost; Request; Response; StatusConflict; StatusInternalServerError; StatusNotFound; StatusRequestEntityTooLarge; StatusText; StatusUnprocessableEntity | Typed native host client or privileged development executor. |
| [internal/host/prepare_test.go](../../../../internal/host/prepare_test.go) | Client; Header; Request; Response | Typed native host client or privileged development executor. |
| [internal/host/tailnet_test.go](../../../../internal/host/tailnet_test.go) | Client; Header; Request; Response | Typed native host client or privileged development executor. |
| [internal/identity/client/broker_compat_test.go](../../../../internal/identity/client/broker_compat_test.go) | Error; NewServeMux; Request; ResponseWriter; Server; StatusForbidden | Direct package consumer. |
| [internal/identity/client/client_test.go](../../../../internal/identity/client/client_test.go) | HandlerFunc; Request; Response; ResponseWriter; RoundTripper; StatusForbidden | Direct package consumer. |
| `internal/tailnet/control_test.go` (`../../../../internal/tailnet/control_test.go`; historical source locator) | Client; Header; Request; Response | Tailnet control/provider integration. |
| `internal/tailnet/enrollment_recovery_test.go` (`../../../../internal/tailnet/enrollment_recovery_test.go`; historical source locator) | Client; Request; Response | Tailnet control/provider integration. |
| `internal/tailnet/enrollment_test.go` (`../../../../internal/tailnet/enrollment_test.go`; historical source locator) | Client; Request; Response | Tailnet control/provider integration. |
| [internal/web/api/environments_join_test.go](../../../../internal/web/api/environments_join_test.go) | MethodPost; StatusUnprocessableEntity | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_test.go](../../../../internal/web/api/extension_test.go) | Header; MethodDelete; MethodGet; MethodPost; MethodPut; Response; StatusForbidden; StatusNotFound | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_intake_test.go](../../../../internal/web/api/factory_intake_test.go) | MethodGet; MethodPost; Request; StatusBadRequest; StatusMethodNotAllowed; StatusOK; StatusRequestEntityTooLarge; StatusServiceUnavailable; StatusUnauthorized | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_output_test.go](../../../../internal/web/api/factory_output_test.go) | Client; HandlerFunc; Header; Request; Response; ResponseWriter; StatusNotFound; StatusOK | Product HTTP/WebSocket handler or adapter. |
| [internal/web/auth/errors_test.go](../../../../internal/web/auth/errors_test.go) | StatusConflict; StatusNotFound | Web authorization/session facade. |
| [internal/web/auth/extension_test.go](../../../../internal/web/auth/extension_test.go) | MethodPatch; StatusForbidden | Web authorization/session facade. |
| [internal/web/avatars_browser_test.go](../../../../internal/web/avatars_browser_test.go) | Error; NewServeMux; Request; ResponseWriter | Web server composition or feature adapter. |
| [internal/web/avatars_test.go](../../../../internal/web/avatars_test.go) | MethodGet; MethodHead; StatusNotModified; StatusOK | Web server composition or feature adapter. |
| [internal/web/browser_join_test.go](../../../../internal/web/browser_join_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/environment_authority_test.go](../../../../internal/web/environment_authority_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/environment_os_test.go](../../../../internal/web/environment_os_test.go) | Client; DefaultTransport; HandlerFunc; Request; Response; ResponseWriter | Web server composition or feature adapter. |
| [internal/web/environment_read_publication_test.go](../../../../internal/web/environment_read_publication_test.go) | Client; MethodGet; MethodPost; Request; Response; StatusOK; StatusUnauthorized | Web server composition or feature adapter. |
| [internal/web/environments_api_test.go](../../../../internal/web/environments_api_test.go) | Client; DefaultTransport; HandlerFunc; Request; Response; ResponseWriter | Web server composition or feature adapter. |
| [internal/web/execution_access_test.go](../../../../internal/web/execution_access_test.go) | Client; Header; Request; Response; StatusForbidden; StatusOK | Web server composition or feature adapter. |
| [internal/web/extension_terminal_test.go](../../../../internal/web/extension_terminal_test.go) | HandlerFunc; Header; MethodGet; MethodPost; NewRequestWithContext; Request; Response; ResponseWriter; StatusCreated; StatusForbidden; StatusOK | Web server composition or feature adapter. |
| [internal/web/factory_intake_route_test.go](../../../../internal/web/factory_intake_route_test.go) | MethodPost; StatusOK; StatusServiceUnavailable; StatusUnauthorized | Web server composition or feature adapter. |
| [internal/web/factory_lifecycle_test.go](../../../../internal/web/factory_lifecycle_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/factory_output_fixture_test.go](../../../../internal/web/factory_output_fixture_test.go) | Client; MethodGet; Request; Response | Web server composition or feature adapter. |
| [internal/web/factory_output_stream_test.go](../../../../internal/web/factory_output_stream_test.go) | MethodGet; StatusForbidden | Web server composition or feature adapter. |
| [internal/web/factory_settings_test.go](../../../../internal/web/factory_settings_test.go) | Client; Header; MethodGet; MethodPost; MethodPut; Request; Response | Web server composition or feature adapter. |
| [internal/web/factory_views_test.go](../../../../internal/web/factory_views_test.go) | Client; Header; MethodGet; Request; Response | Web server composition or feature adapter. |
| [internal/web/identity_native_test.go](../../../../internal/web/identity_native_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/join_boundary_test.go](../../../../internal/web/join_boundary_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/lifecycle_access_keys_test.go](../../../../internal/web/lifecycle_access_keys_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/mutation_admission_test.go](../../../../internal/web/mutation_admission_test.go) | Client; Request; Response | Web server composition or feature adapter. |
| [internal/web/preparation_test.go](../../../../internal/web/preparation_test.go) | Client; Header; MethodGet; MethodPost; NoBody; Request; Response | Web server composition or feature adapter. |
| [internal/web/project_profiles_test.go](../../../../internal/web/project_profiles_test.go) | Client; Request; Response | Web server composition or feature adapter. |
| [internal/web/provisioning_lifetime_test.go](../../../../internal/web/provisioning_lifetime_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/repositories_test.go](../../../../internal/web/repositories_test.go) | Header | Web server composition or feature adapter. |
| [internal/web/repository_access_test.go](../../../../internal/web/repository_access_test.go) | Client; Header; Request; Response | Web server composition or feature adapter. |
| [internal/web/repository_settings_test.go](../../../../internal/web/repository_settings_test.go) | Client; Request; Response | Web server composition or feature adapter. |
| [internal/web/spaces_inventory_test.go](../../../../internal/web/spaces_inventory_test.go) | Client; HandlerFunc; Request; ResponseWriter; Transport | Web server composition or feature adapter. |
| [internal/web/spaces_test.go](../../../../internal/web/spaces_test.go) | Client; HandlerFunc; Header; MethodGet; Request; Response; ResponseWriter; StatusBadRequest; StatusForbidden; StatusNotFound; StatusServiceUnavailable; StatusUnauthorized; Transport | Web server composition or feature adapter. |
| [internal/web/tailnet_test.go](../../../../internal/web/tailnet_test.go) | Header; Request; Response; ResponseWriter | Web server composition or feature adapter. |
| [internal/web/terminal_test.go](../../../../internal/web/terminal_test.go) | Client; HandlerFunc; NotFound; Request; ResponseWriter; Transport | Web server composition or feature adapter. |
| [internal/web/test_helpers_test.go](../../../../internal/web/test_helpers_test.go) | HandlerFunc; Header; NewRequestWithContext; Request; Response; ResponseWriter; StatusForbidden; StatusNotFound | Web server composition or feature adapter. |
| [scripts/forgejo_presentation_test.go](../../../../scripts/forgejo_presentation_test.go) | Client; StatusOK | Test/developer helper. |
| [scripts/wire_contracts_test.go](../../../../scripts/wire_contracts_test.go) | HandlerFunc; Request; ResponseWriter; Server; StatusOK | Test/developer helper. |
| [tests/build/avatar_integration_test.go](../../../../tests/build/avatar_integration_test.go) | Client; HandlerFunc; NewRequest; Request; ResponseWriter | Test/developer helper. |
| [tools/soda-rootfs-server/main_test.go](../../../../tools/soda-rootfs-server/main_test.go) | Client; HandlerFunc; Header; MethodGet; MethodHead; NewRequest; Server; StatusNotFound; StatusOK; server | Developer/support command. |

<a id="integration-12"></a>

## Go standard library encoding/json

**Selected:** Go 1.26.7 module language/toolchain floor (go.mod). **Upstream capability used:** JSON request/response encoding and typed boundary decoding.

**Dependency admission:** go.mod `go 1.26.7`; no external module version applies.

**Primary-source evidence:** Standard-library direct import inventory; domain wrappers remain Soda-owned types and strictjson handles bounded single-object policy where called.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/acceptance/developer_access.go](../../../../internal/acceptance/developer_access.go) | RawMessage; Unmarshal | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** decodeAccessRequest; requestString; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/developer_access_transfer.go](../../../../internal/acceptance/developer_access_transfer.go) | MarshalIndent | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** writeAccessResults; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/developer_access_users.go](../../../../internal/acceptance/developer_access_users.go) | RawMessage; Unmarshal | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** checkAccessUserAdmin; loadAccessBrowser; validateAccessUser; validateAccessUsers; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/personal_git.go](../../../../internal/acceptance/personal_git.go) | MarshalIndent; RawMessage; Unmarshal | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** loadExerciseRepo; loadGitTarget; parseGitTarget; writeGitOutcomes; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_access.go](../../../../internal/acceptance/workload_access.go) | MarshalIndent; RawMessage; Unmarshal | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** finishAccessResults; loadAccessTarget; parseAccessTarget; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_exec.go](../../../../internal/acceptance/workload_exec.go) | Unmarshal | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** loadExecTarget; **chain:** Developer acceptance/evidence support integration. |
| [internal/config/config.go](../../../../internal/config/config.go) | NewDecoder | none | **role:** Direct package consumer.; **entrypoints:** decodeConfig; **chain:** Direct package consumer. |
| [internal/factory/assignment_result.go](../../../../internal/factory/assignment_result.go) | NewDecoder | none | **role:** Direct package consumer.; **entrypoints:** ParseHarnessResult; **chain:** Direct package consumer. |
| [internal/factory/control/acceptance.go](../../../../internal/factory/control/acceptance.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** AdmitAcceptance; replayAcceptance; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/acceptance_status.go](../../../../internal/factory/control/acceptance_status.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** WithdrawAcceptance; replayWithdrawal; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_registration.go](../../../../internal/factory/control/dispatch_registration.go) | Marshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** ReopenDispatch; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/grants.go](../../../../internal/factory/control/grants.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** applyGrant; replayGrant; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/lifecycle.go](../../../../internal/factory/control/lifecycle.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** PauseRepository; ResumeRepository; replayPause; replayResume; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/lifecycle_retry.go](../../../../internal/factory/control/lifecycle_retry.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** RetryRun; replayRetry; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/lifecycle_takeover.go](../../../../internal/factory/control/lifecycle_takeover.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** TakeoverRun; replayTakeover; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/operator.go](../../../../internal/factory/control/operator.go) | NewEncoder | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** writeCommandError; writeOperator; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/preparation_decisions.go](../../../../internal/factory/control/preparation_decisions.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** admitDecision; replayDecision; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/settle.go](../../../../internal/factory/control/settle.go) | Marshal; Unmarshal | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** Reconcile; Stop; finishAbandonedCommand; replayReconcile; replayStop; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/review_native.go](../../../../internal/factory/review_native.go) | NewDecoder | none | **role:** Direct package consumer.; **entrypoints:** ParseReviewReport; **chain:** Direct package consumer. |
| [internal/forgejo/background_admission.go](../../../../internal/forgejo/background_admission.go) | Marshal; NewDecoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** bootstrap; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/background_transport.go](../../../../internal/forgejo/background_transport.go) | Marshal; NewDecoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** call; post; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/client.go](../../../../internal/forgejo/client.go) | NewDecoder; NewEncoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** decodeResponse; request; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge.go](../../../../internal/forgejo/merge.go) | Marshal | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** mergeIntent; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge_completion.go](../../../../internal/forgejo/merge_completion.go) | NewDecoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** AdoptMerge; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation.go](../../../../internal/forgejo/publish/operation.go) | Marshal; RawMessage | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** prCreatePayload; publishPayload; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_receipts.go](../../../../internal/forgejo/publish/operation_receipts.go) | NewDecoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** DecodePRCreateReceipt; DecodePublishReceipt; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/review.go](../../../../internal/forgejo/review.go) | Marshal; NewDecoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** AdoptReview; reviewIntent; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/snapshot_transport.go](../../../../internal/forgejo/snapshot_transport.go) | Marshal; NewDecoder | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** decodeSnapshotAnswer; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/host/client.go](../../../../internal/host/client.go) | NewEncoder; Unmarshal | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** callLimit; decodeNativeResponseLimit; **chain:** Typed native host client or privileged development executor. |
| [internal/host/terminal.go](../../../../internal/host/terminal.go) | Marshal | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** Write; **chain:** Typed native host client or privileged development executor. |
| [internal/host/terminal_client.go](../../../../internal/host/terminal_client.go) | Marshal | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** OpenTerminal; **chain:** Typed native host client or privileged development executor. |
| [internal/identity/client/client.go](../../../../internal/identity/client/client.go) | NewDecoder; NewEncoder | none | **role:** Direct package consumer.; **entrypoints:** call; decodeResponse; **chain:** Direct package consumer. |
| [internal/identity/types.go](../../../../internal/identity/types.go) | Valid | none | **role:** Direct package consumer.; **entrypoints:** CredentialValid; **chain:** Direct package consumer. |
| [internal/store/factory.go](../../../../internal/store/factory.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** FactoryRun; FactoryRuns; RecordFactoryRun; SaveFactoryRun; **chain:** Persistent store/driver integration. |
| [internal/store/factory_assignments.go](../../../../internal/store/factory_assignments.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** AssignedAssignments; Assignment; AssignmentByRun; FinishAssignment; IssueAssignments; **chain:** Persistent store/driver integration. |
| [internal/store/factory_checks.go](../../../../internal/store/factory_checks.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** CheckAssessment; RecordCheckAssessment; **chain:** Persistent store/driver integration. |
| [internal/store/factory_dispatch_packet.go](../../../../internal/store/factory_dispatch_packet.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** RecordDispatchPacket; checkAdmissionTx; **chain:** Persistent store/driver integration. |
| [internal/store/factory_dispatch_queue.go](../../../../internal/store/factory_dispatch_queue.go) | Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** QueuedControls; **chain:** Persistent store/driver integration. |
| [internal/store/factory_grants.go](../../../../internal/store/factory_grants.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** DispatchState; SaveSponsorship; Sponsorship; Sponsorships; WithdrawDispatch; loadGrant; registerDispatchTx; saveRevisionedGrant; **chain:** Persistent store/driver integration. |
| [internal/store/factory_inventory.go](../../../../internal/store/factory_inventory.go) | Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** FactoryUnsettledRuns; ProjectFactoryRuns; ProjectUnsettledRuns; QueuedControlsAfter; **chain:** Persistent store/driver integration. |
| [internal/store/factory_lifecycle.go](../../../../internal/store/factory_lifecycle.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** RecordTakeover; Takeover; **chain:** Persistent store/driver integration. |
| [internal/store/factory_merges.go](../../../../internal/store/factory_merges.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** IssueMergeCompletion; MergeByPublication; MergeForIssue; MergeablePublications; OpenMerges; OutstandingMerges; RecordMerge; UpdateMerge; **chain:** Persistent store/driver integration. |
| [internal/store/factory_publications.go](../../../../internal/store/factory_publications.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** OpenPublications; OutstandingPublications; PublicationByAssignment; PublishableAssignments; RecordPublication; UpdatePublication; **chain:** Persistent store/driver integration. |
| [internal/store/factory_reservations.go](../../../../internal/store/factory_reservations.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** HeldReservations; RecordRunUsage; ReholdReservation; Reservation; RunUsage; **chain:** Persistent store/driver integration. |
| [internal/store/factory_retry_packet.go](../../../../internal/store/factory_retry_packet.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** RecordRetryPacket; **chain:** Persistent store/driver integration. |
| [internal/store/identity_fixture.go](../../../../internal/store/identity_fixture.go) | Marshal | none | **role:** Persistent store/driver integration.; **entrypoints:** SeedIdentityConnection; SeedIdentityGrant; appendIdentityEvent; **chain:** Persistent store/driver integration. |
| [internal/store/issue_acceptances.go](../../../../internal/store/issue_acceptances.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** AcceptanceDecision; AdmitAcceptanceDecision; **chain:** Persistent store/driver integration. |
| [internal/store/issue_controls.go](../../../../internal/store/issue_controls.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** FactoryPolicies; IssueControl; IssueControls; RecordIssueAssessment; **chain:** Persistent store/driver integration. |
| [internal/store/preparation.go](../../../../internal/store/preparation.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** AdmitPreparation; LifecycleGrant; MaintenanceHold; ObservePreparation; Preparation; ProjectPreparations; SaveLifecycleGrant; SaveMaintenanceHold; **chain:** Persistent store/driver integration. |
| [internal/store/project_grants.go](../../../../internal/store/project_grants.go) | Marshal; Unmarshal | none | **role:** Persistent store/driver integration.; **entrypoints:** ApprovalDecision; RequirementDecision; admitProjectDecision; **chain:** Persistent store/driver integration. |
| [internal/store/store.go](../../../../internal/store/store.go) | Marshal | none | **role:** Persistent store/driver integration.; **entrypoints:** CreateProject; **chain:** Persistent store/driver integration. |
| [internal/strictjson/decode.go](../../../../internal/strictjson/decode.go) | Decoder; Delim; Marshal; NewDecoder; RawMessage | none | **role:** Direct package consumer.; **entrypoints:** Decode; decodeFields; decodeUniqueObject; finishObject; rejectDuplicateKeys; rejectObjectDuplicates; requireObject; **chain:** Direct package consumer. |
| `internal/tailnet/control.go` (`../../../../internal/tailnet/control.go`; historical source locator) | Marshal; NewDecoder; RawMessage; Unmarshal | none | **role:** Tailnet control/provider integration.; **entrypoints:** computeHostRevision; decodeUpNotifications; nativeObject; request; **chain:** Tailnet control/provider integration. |
| `internal/tailnet/enrollment.go` (`../../../../internal/tailnet/enrollment.go`; historical source locator) | RawMessage | none | **role:** Tailnet control/provider integration.; **entrypoints:** validKeyCreateCapabilities; **chain:** Tailnet control/provider integration. |
| `internal/tailnet/policy.go` (`../../../../internal/tailnet/policy.go`; historical source locator) | Marshal | none | **role:** Tailnet control/provider integration.; **entrypoints:** publish; **chain:** Tailnet control/provider integration. |
| [internal/tailnet/tailnet.go](../../../../internal/tailnet/tailnet.go) | Unmarshal | none | **role:** Tailnet control/provider integration.; **entrypoints:** parseStatus; **chain:** Tailnet control/provider integration. |
| [internal/web/api/factory_intake.go](../../../../internal/web/api/factory_intake.go) | Unmarshal | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** parseIntakeHint; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go) | Marshal | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** writeFactoryFrame; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/spaces.go](../../../../internal/web/api/spaces.go) | Marshal | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** appendSpaceRow; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/terminal_registry.go](../../../../internal/web/api/terminal_registry.go) | Marshal | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** pumpNativeToExtension; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/auth/http.go](../../../../internal/web/auth/http.go) | Marshal | none | **role:** Web authorization/session facade.; **entrypoints:** JSONResponse; **chain:** Web authorization/session facade. |
| [tools/soda-avatars/main.go](../../../../tools/soda-avatars/main.go) | RawMessage; Unmarshal | none | **role:** Developer/support command.; **entrypoints:** generate; **chain:** Developer/support command. |
| source consumers listed below | none | **role:** Soda-owned functions/types define policy; standard package supplies primitive transport, serialization, encoding, time or cryptographic operations. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/acceptance/developer_access_journey_test.go](../../../../internal/acceptance/developer_access_journey_test.go) | Unmarshal | Developer acceptance/evidence support integration. |
| [internal/acceptance/developer_access_request_test.go](../../../../internal/acceptance/developer_access_request_test.go) | Marshal | Developer acceptance/evidence support integration. |
| [internal/acceptance/developer_access_test.go](../../../../internal/acceptance/developer_access_test.go) | RawMessage | Developer acceptance/evidence support integration. |
| [internal/acceptance/personal_git_test.go](../../../../internal/acceptance/personal_git_test.go) | MarshalIndent; RawMessage; Unmarshal | Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_access_test.go](../../../../internal/acceptance/workload_access_test.go) | MarshalIndent | Developer acceptance/evidence support integration. |
| [internal/avatar/avatar_test.go](../../../../internal/avatar/avatar_test.go) | RawMessage; Unmarshal | Avatar rendering facade. |
| [internal/config/background_test.go](../../../../internal/config/background_test.go) | Marshal | Direct package consumer. |
| [internal/config/intake_test.go](../../../../internal/config/intake_test.go) | Marshal | Direct package consumer. |
| [internal/config/load_test.go](../../../../internal/config/load_test.go) | Marshal | Direct package consumer. |
| [internal/config/review_credential_test.go](../../../../internal/config/review_credential_test.go) | Marshal | Direct package consumer. |
| [internal/factory/control/checks_native_fixture_test.go](../../../../internal/factory/control/checks_native_fixture_test.go) | Marshal; MarshalIndent; NewDecoder; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_fixture_test.go](../../../../internal/factory/control/merge_native_fixture_test.go) | MarshalIndent; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_test.go](../../../../internal/factory/control/merge_native_test.go) | Marshal; NewDecoder; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/operator_test.go](../../../../internal/factory/control/operator_test.go) | Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_effect_test.go](../../../../internal/factory/control/publication_effect_test.go) | Marshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_fixture_test.go](../../../../internal/factory/control/publication_native_fixture_test.go) | Marshal; MarshalIndent; NewDecoder; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/review_native_primitive_test.go](../../../../internal/factory/control/review_native_primitive_test.go) | Marshal; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_coding_test.go](../../../../internal/factory/control/st15_demo_coding_test.go) | MarshalIndent | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_completion_test.go](../../../../internal/factory/control/st15_demo_completion_test.go) | Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_coordinator_test.go](../../../../internal/factory/control/st15_demo_coordinator_test.go) | Marshal; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_native_test.go](../../../../internal/factory/control/st15_demo_native_test.go) | MarshalIndent; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_stack_test.go](../../../../internal/factory/control/st15_demo_stack_test.go) | Marshal; Unmarshal | Factory coordinator composition or runtime service. |
| [internal/factory/grants_test.go](../../../../internal/factory/grants_test.go) | Marshal | Direct package consumer. |
| [internal/forgejo/background_test.go](../../../../internal/forgejo/background_test.go) | NewDecoder; NewEncoder | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge_test.go](../../../../internal/forgejo/merge_test.go) | Marshal; Unmarshal | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/observe_test.go](../../../../internal/forgejo/observe_test.go) | NewDecoder; NewEncoder | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_receipts_test.go](../../../../internal/forgejo/publish/operation_receipts_test.go) | Marshal | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_test.go](../../../../internal/forgejo/publish/operation_test.go) | Unmarshal | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/review_test.go](../../../../internal/forgejo/review_test.go) | Marshal; Unmarshal | Forgejo SDK/client adapter and domain conversion. |
| [internal/host/prepare_test.go](../../../../internal/host/prepare_test.go) | Marshal | Typed native host client or privileged development executor. |
| [internal/identity/client/broker_compat_test.go](../../../../internal/identity/client/broker_compat_test.go) | Marshal; NewDecoder; NewEncoder | Direct package consumer. |
| [internal/project/project_test.go](../../../../internal/project/project_test.go) | Marshal; Unmarshal | Direct package consumer. |
| [internal/store/identity_test.go](../../../../internal/store/identity_test.go) | Unmarshal | Persistent store/driver integration. |
| `internal/tailnet/control_test.go` (`../../../../internal/tailnet/control_test.go`; historical source locator) | Marshal; Unmarshal | Tailnet control/provider integration. |
| `internal/tailnet/enrollment_test.go` (`../../../../internal/tailnet/enrollment_test.go`; historical source locator) | Marshal; NewDecoder; Unmarshal | Tailnet control/provider integration. |
| `internal/tailnet/policy_test.go` (`../../../../internal/tailnet/policy_test.go`; historical source locator) | Marshal; Unmarshal | Tailnet control/provider integration. |
| `internal/tailnet/project_runtime_test.go` (`../../../../internal/tailnet/project_runtime_test.go`; historical source locator) | Marshal | Tailnet control/provider integration. |
| [internal/web/api/factory_intake_test.go](../../../../internal/web/api/factory_intake_test.go) | Unmarshal | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_output_test.go](../../../../internal/web/api/factory_output_test.go) | Marshal; Unmarshal | Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_views_test.go](../../../../internal/web/api/factory_views_test.go) | Marshal; Unmarshal | Product HTTP/WebSocket handler or adapter. |
| [internal/web/browser_join_test.go](../../../../internal/web/browser_join_test.go) | NewDecoder | Web server composition or feature adapter. |
| [internal/web/environment_authority_test.go](../../../../internal/web/environment_authority_test.go) | Unmarshal | Web server composition or feature adapter. |
| [internal/web/environment_os_test.go](../../../../internal/web/environment_os_test.go) | NewDecoder; NewEncoder | Web server composition or feature adapter. |
| [internal/web/environment_read_publication_test.go](../../../../internal/web/environment_read_publication_test.go) | NewDecoder; NewEncoder; RawMessage; Unmarshal | Web server composition or feature adapter. |
| [internal/web/environments_api_test.go](../../../../internal/web/environments_api_test.go) | NewDecoder; NewEncoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/execution_access_test.go](../../../../internal/web/execution_access_test.go) | Unmarshal | Web server composition or feature adapter. |
| [internal/web/extension_terminal_test.go](../../../../internal/web/extension_terminal_test.go) | Marshal; NewDecoder; NewEncoder | Web server composition or feature adapter. |
| [internal/web/factory_intake_route_test.go](../../../../internal/web/factory_intake_route_test.go) | Unmarshal | Web server composition or feature adapter. |
| [internal/web/factory_lifecycle_test.go](../../../../internal/web/factory_lifecycle_test.go) | NewDecoder; NewEncoder | Web server composition or feature adapter. |
| [internal/web/factory_output_fixture_test.go](../../../../internal/web/factory_output_fixture_test.go) | Marshal; NewDecoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/factory_settings_test.go](../../../../internal/web/factory_settings_test.go) | Unmarshal | Web server composition or feature adapter. |
| [internal/web/factory_views_test.go](../../../../internal/web/factory_views_test.go) | NewEncoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/identity_native_test.go](../../../../internal/web/identity_native_test.go) | NewDecoder | Web server composition or feature adapter. |
| [internal/web/lifecycle_access_keys_test.go](../../../../internal/web/lifecycle_access_keys_test.go) | Marshal; NewDecoder | Web server composition or feature adapter. |
| [internal/web/mutation_admission_test.go](../../../../internal/web/mutation_admission_test.go) | NewDecoder; NewEncoder | Web server composition or feature adapter. |
| [internal/web/preparation_test.go](../../../../internal/web/preparation_test.go) | NewDecoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/project_profiles_test.go](../../../../internal/web/project_profiles_test.go) | Marshal | Web server composition or feature adapter. |
| [internal/web/provisioning_lifetime_test.go](../../../../internal/web/provisioning_lifetime_test.go) | Marshal; NewDecoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/repository_access_test.go](../../../../internal/web/repository_access_test.go) | NewDecoder | Web server composition or feature adapter. |
| [internal/web/spaces_inventory_test.go](../../../../internal/web/spaces_inventory_test.go) | NewDecoder; NewEncoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/spaces_test.go](../../../../internal/web/spaces_test.go) | NewDecoder; NewEncoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/tailnet_test.go](../../../../internal/web/tailnet_test.go) | Marshal; NewDecoder; NewEncoder | Web server composition or feature adapter. |
| [internal/web/terminal_test.go](../../../../internal/web/terminal_test.go) | Marshal; NewDecoder; NewEncoder; Unmarshal | Web server composition or feature adapter. |
| [internal/web/test_helpers_test.go](../../../../internal/web/test_helpers_test.go) | Marshal; NewDecoder; NewEncoder | Web server composition or feature adapter. |
| [scripts/forgejo_form_layout_test.go](../../../../scripts/forgejo_form_layout_test.go) | Unmarshal | Test/developer helper. |
| [scripts/forgejo_soda_settings_test.go](../../../../scripts/forgejo_soda_settings_test.go) | Unmarshal | Test/developer helper. |
| [scripts/system_formats_test.go](../../../../scripts/system_formats_test.go) | Marshal; Unmarshal | Test/developer helper. |
| [scripts/wire_contracts_test.go](../../../../scripts/wire_contracts_test.go) | Unmarshal | Test/developer helper. |
| [tests/build/avatar_integration_test.go](../../../../tests/build/avatar_integration_test.go) | Marshal; Unmarshal | Test/developer helper. |
| [tests/build/helpers.go](../../../../tests/build/helpers.go) | Unmarshal | Test/developer helper. |
| [tests/build/native_support_test.go](../../../../tests/build/native_support_test.go) | Unmarshal | Test/developer helper. |
| [tests/build/project_account_test.go](../../../../tests/build/project_account_test.go) | Marshal | Test/developer helper. |
| [tests/build/project_factory_roles_fixture_test.go](../../../../tests/build/project_factory_roles_fixture_test.go) | Marshal; Unmarshal | Test/developer helper. |
| [tests/build/project_factory_roles_inputs_test.go](../../../../tests/build/project_factory_roles_inputs_test.go) | Unmarshal | Test/developer helper. |
| [tests/build/sodaspaces_test.go](../../../../tests/build/sodaspaces_test.go) | Marshal; Unmarshal | Test/developer helper. |
| [tests/build/source_checks_test.go](../../../../tests/build/source_checks_test.go) | Unmarshal | Test/developer helper. |
| [tests/build/workload_probe_test.go](../../../../tests/build/workload_probe_test.go) | Unmarshal | Test/developer helper. |

<a id="integration-13"></a>

## Go standard library encoding/base64

**Selected:** Go 1.26.7 module language/toolchain floor (go.mod). **Upstream capability used:** Binary credential/output/frame and token payload encodings.

**Dependency admission:** go.mod `go 1.26.7`; no external module version applies.

**Primary-source evidence:** Standard-library direct import inventory; each listed caller uses std encoding/base64 directly.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/acceptance/personal_git.go](../../../../internal/acceptance/personal_git.go) | RawURLEncoding | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** tokenPassphrase; **chain:** Developer acceptance/evidence support integration. |
| [internal/config/config.go](../../../../internal/config/config.go) | StdEncoding | none | **role:** Direct package consumer.; **entrypoints:** GrantKey; **chain:** Direct package consumer. |
| [internal/factory/control/publication_export.go](../../../../internal/factory/control/publication_export.go) | StdEncoding | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** decodeExportBundle; **chain:** Factory coordinator composition or runtime service. |
| [internal/host/terminal.go](../../../../internal/host/terminal.go) | StdEncoding | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** validOutputData; validTypedInput; **chain:** Typed native host client or privileged development executor. |
| source consumers listed below | none | **role:** Soda-owned functions/types define policy; standard package supplies primitive transport, serialization, encoding, time or cryptographic operations. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/config/grant_key_test.go](../../../../internal/config/grant_key_test.go) | StdEncoding | Direct package consumer. |
| [internal/factory/control/publication_fixture_test.go](../../../../internal/factory/control/publication_fixture_test.go) | StdEncoding | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_fixture_test.go](../../../../internal/factory/control/publication_native_fixture_test.go) | StdEncoding | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_review_test.go](../../../../internal/factory/control/st15_demo_review_test.go) | StdEncoding | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_stack_test.go](../../../../internal/factory/control/st15_demo_stack_test.go) | StdEncoding | Factory coordinator composition or runtime service. |
| [internal/identity/client/broker_compat_test.go](../../../../internal/identity/client/broker_compat_test.go) | StdEncoding | Direct package consumer. |
| [internal/web/api/factory_output_test.go](../../../../internal/web/api/factory_output_test.go) | StdEncoding | Product HTTP/WebSocket handler or adapter. |
| [tests/build/project_factory_roles_fixture_test.go](../../../../tests/build/project_factory_roles_fixture_test.go) | StdEncoding | Test/developer helper. |
| [tests/build/project_factory_roles_inputs_test.go](../../../../tests/build/project_factory_roles_inputs_test.go) | StdEncoding | Test/developer helper. |

<a id="integration-14"></a>

## Go standard library crypto/*

**Selected:** Go 1.26.7 module language/toolchain floor (go.mod). **Upstream capability used:** Standard cryptographic primitives (hash, HMAC, random, subtle compare, TLS, etc.) directly referenced by app code.

**Dependency admission:** go.mod `go 1.26.7`; no external module version applies.

**Primary-source evidence:** Direct source imports only; no transitive package algorithms are attributed to Soda callers.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/acceptance/developer_access.go](../../../../internal/acceptance/developer_access.go) | New | none | **standard subpackage:** crypto/sha256; **role:** Developer acceptance/evidence support integration.; **entrypoints:** developerAccessDigest; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/installed.go](../../../../internal/acceptance/installed.go) | Reader | none | **standard subpackage:** crypto/rand; **role:** Developer acceptance/evidence support integration.; **entrypoints:** uuidHex; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/personal_git.go](../../../../internal/acceptance/personal_git.go) | Reader | none | **standard subpackage:** crypto/rand; **role:** Developer acceptance/evidence support integration.; **entrypoints:** tokenPassphrase; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/service_https.go](../../../../internal/acceptance/service_https.go) | Config; VersionTLS12; CertPool; NewCertPool | none | **standard subpackage:** crypto/tls; crypto/x509; **role:** Developer acceptance/evidence support integration.; **entrypoints:** CheckServiceHTTPS; httpsClient; **chain:** Developer acceptance/evidence support integration. |
| [internal/factory/acceptance.go](../../../../internal/factory/acceptance.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** InitialAcceptanceID; **chain:** Direct package consumer. |
| [internal/factory/assignment.go](../../../../internal/factory/assignment.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** Validate; **chain:** Direct package consumer. |
| [internal/factory/checks.go](../../../../internal/factory/checks.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** ChecksDigest; **chain:** Direct package consumer. |
| [internal/factory/grants.go](../../../../internal/factory/grants.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** SettingsDigest; **chain:** Direct package consumer. |
| [internal/factory/readiness.go](../../../../internal/factory/readiness.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** AuthorityFingerprint; Fingerprint; **chain:** Direct package consumer. |
| [internal/factory/types.go](../../../../internal/factory/types.go) | Read; Sum256 | none | **standard subpackage:** crypto/rand; crypto/sha256; **role:** Direct package consumer.; **entrypoints:** CommandDigest; NewID; **chain:** Direct package consumer. |
| [internal/forgejo/publish/credentials.go](../../../../internal/forgejo/publish/credentials.go) | New | none | **standard subpackage:** crypto/sha1; **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** scanCredentialObject; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/snapshot_request.go](../../../../internal/forgejo/snapshot_request.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** ContentDigest; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/identity/types.go](../../../../internal/identity/types.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** AcquisitionDigest; **chain:** Direct package consumer. |
| [internal/project/factory.go](../../../../internal/project/factory.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** FactoryPromptDigest; **chain:** Direct package consumer. |
| [internal/project/preparation.go](../../../../internal/project/preparation.go) | New | none | **standard subpackage:** crypto/sha256; **role:** Direct package consumer.; **entrypoints:** SetupDigestOf; **chain:** Direct package consumer. |
| [internal/store/ephemeral.go](../../../../internal/store/ephemeral.go) | Read | none | **standard subpackage:** crypto/rand; **role:** Persistent store/driver integration.; **entrypoints:** ephemeralName; **chain:** Persistent store/driver integration. |
| [internal/store/grants.go](../../../../internal/store/grants.go) | NewCipher; AEAD; NewGCM; Read | none | **standard subpackage:** crypto/aes; crypto/cipher; crypto/rand; **role:** Persistent store/driver integration.; **entrypoints:** newGrantCipher; seal; **chain:** Persistent store/driver integration. |
| `internal/tailnet/control.go` (`../../../../internal/tailnet/control.go`; historical source locator) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Tailnet control/provider integration.; **entrypoints:** computeHostRevision; **chain:** Tailnet control/provider integration. |
| `internal/tailnet/policy.go` (`../../../../internal/tailnet/policy.go`; historical source locator) | Read | none | **standard subpackage:** crypto/rand; **role:** Tailnet control/provider integration.; **entrypoints:** newRevision; **chain:** Tailnet control/provider integration. |
| [internal/web/api/environments_create.go](../../../../internal/web/api/environments_create.go) | Read | none | **standard subpackage:** crypto/rand; **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiCreateEnvironment; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal_authority.go](../../../../internal/web/api/extension_terminal_authority.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** nativeTerminalScope; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_intake.go](../../../../internal/web/api/factory_intake.go) | New; ConstantTimeCompare | none | **standard subpackage:** crypto/hmac; crypto/sha256; crypto/subtle; **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** verifyIntakeSignature; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/terminal_registry.go](../../../../internal/web/api/terminal_registry.go) | Read; Sum256 | none | **standard subpackage:** crypto/rand; crypto/sha256; **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** TerminalCreationScope; newTerminalID; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/avatars.go](../../../../internal/web/avatars.go) | Sum256 | none | **standard subpackage:** crypto/sha256; **role:** Web server composition or feature adapter.; **entrypoints:** ServeHTTP; **chain:** Web server composition or feature adapter. |
| [tools/soda-avatars/main.go](../../../../tools/soda-avatars/main.go) | Sum | none | **standard subpackage:** crypto/md5; **role:** Developer/support command.; **entrypoints:** robotSamples; **chain:** Developer/support command. |
| source consumers listed below | none | **role:** Soda-owned functions/types define policy; standard package supplies primitive transport, serialization, encoding, time or cryptographic operations. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [internal/avatar/avatar_test.go](../../../../internal/avatar/avatar_test.go) | Sum256 | Avatar rendering facade. |
| [internal/factory/assignment_test.go](../../../../internal/factory/assignment_test.go) | Sum256 | Direct package consumer. |
| [internal/factory/control/dispatch_fixture_test.go](../../../../internal/factory/control/dispatch_fixture_test.go) | Sum256 | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_setup_test.go](../../../../internal/factory/control/merge_native_setup_test.go) | Sum256 | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_fixture_test.go](../../../../internal/factory/control/publication_native_fixture_test.go) | Sum256 | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_coordinator_test.go](../../../../internal/factory/control/st15_demo_coordinator_test.go) | New; Read | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_native_test.go](../../../../internal/factory/control/st15_demo_native_test.go) | Read; Sum256 | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_stack_test.go](../../../../internal/factory/control/st15_demo_stack_test.go) | Read; Sum256 | Factory coordinator composition or runtime service. |
| [internal/forgejo/publish/credentials_test.go](../../../../internal/forgejo/publish/credentials_test.go) | Sum | Forgejo SDK/client adapter and domain conversion. |
| [internal/identity/client/broker_compat_test.go](../../../../internal/identity/client/broker_compat_test.go) | Read; Sum256 | Direct package consumer. |
| [internal/store/factory_dispatch_test.go](../../../../internal/store/factory_dispatch_test.go) | Sum256 | Persistent store/driver integration. |
| [internal/web/api/factory_intake_test.go](../../../../internal/web/api/factory_intake_test.go) | New | Product HTTP/WebSocket handler or adapter. |
| [internal/web/factory_intake_route_test.go](../../../../internal/web/factory_intake_route_test.go) | New | Web server composition or feature adapter. |
| [internal/web/forgejo_keys_test.go](../../../../internal/web/forgejo_keys_test.go) | NewKeyFromSeed | Web server composition or feature adapter. |
| [internal/web/lifecycle_access_keys_test.go](../../../../internal/web/lifecycle_access_keys_test.go) | GenerateKey; Reader | Web server composition or feature adapter. |
| [scripts/forgejo_admin_details_test.go](../../../../scripts/forgejo_admin_details_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_admin_monitoring_test.go](../../../../scripts/forgejo_admin_monitoring_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_admin_org_test.go](../../../../scripts/forgejo_admin_org_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_auth_test.go](../../../../scripts/forgejo_auth_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_federated_auth_test.go](../../../../scripts/forgejo_federated_auth_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_insights_test.go](../../../../scripts/forgejo_insights_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_org_details_test.go](../../../../scripts/forgejo_org_details_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_org_home_test.go](../../../../scripts/forgejo_org_home_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_repository_code_test.go](../../../../scripts/forgejo_repository_code_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_repository_content_test.go](../../../../scripts/forgejo_repository_content_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_repository_settings_details_test.go](../../../../scripts/forgejo_repository_settings_details_test.go) | Sum256 | Test/developer helper. |
| [scripts/forgejo_setup_test.go](../../../../scripts/forgejo_setup_test.go) | Sum256 | Test/developer helper. |
| [tests/build/project_factory_roles_fixture_test.go](../../../../tests/build/project_factory_roles_fixture_test.go) | New | Test/developer helper. |
| [tests/build/sodaspaces_test.go](../../../../tests/build/sodaspaces_test.go) | Sum256 | Test/developer helper. |

<a id="integration-15"></a>

## Go standard library net/url

**Selected:** Go 1.26.7 module language/toolchain floor (go.mod). **Upstream capability used:** URL/query construction, callback path encoding, and origin parsing.

**Dependency admission:** go.mod `go 1.26.7`; no external module version applies.

**Primary-source evidence:** Direct source imports only; HTTP-specific route validation and typed native query adapters stay in local callers.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [internal/acceptance/personal_git.go](../../../../internal/acceptance/personal_git.go) | Parse; URL | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** validateGitURL; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/service_https.go](../../../../internal/acceptance/service_https.go) | Parse; URL | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** HTTPSOrigin; httpsClient; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_access.go](../../../../internal/acceptance/workload_access.go) | URL | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** fetchAccessURL; **chain:** Developer acceptance/evidence support integration. |
| [internal/config/config.go](../../../../internal/config/config.go) | Parse; URL | none | **role:** Direct package consumer.; **entrypoints:** BaseURL; **chain:** Direct package consumer. |
| [internal/forgejo/observe.go](../../../../internal/forgejo/observe.go) | PathEscape; Values | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** ListRepositoryIssuesPage; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/ownership.go](../../../../internal/forgejo/ownership.go) | PathEscape | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** OrganizationOwner; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish.go](../../../../internal/forgejo/publish.go) | PathEscape | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** publisherConfig; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/publish.go](../../../../internal/forgejo/publish/publish.go) | Parse | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** validateRemote; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/store/ephemeral.go](../../../../internal/store/ephemeral.go) | Parse; URL; UserPassword; Values | none | **role:** Persistent store/driver integration.; **entrypoints:** TestFixtureDSN; createEphemeralDatabase; **chain:** Persistent store/driver integration. |
| `internal/tailnet/control.go` (`../../../../internal/tailnet/control.go`; historical source locator) | Parse; URL; Values | none | **role:** Tailnet control/provider integration.; **entrypoints:** authenticationURL; checkCredential; **chain:** Tailnet control/provider integration. |
| `internal/tailnet/enrollment.go` (`../../../../internal/tailnet/enrollment.go`; historical source locator) | URL; Values | none | **role:** Tailnet control/provider integration.; **entrypoints:** tokenFromEnrollment; **chain:** Tailnet control/provider integration. |
| [internal/web/api/environments_api.go](../../../../internal/web/api/environments_api.go) | ParseQuery; Values | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** parseEnvironmentsListQuery; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/repositories.go](../../../../internal/web/api/repositories.go) | ParseQuery; Values | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** parseNativeRepositoryQuery; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/auth/forgejo_keys.go](../../../../internal/web/auth/forgejo_keys.go) | ParseQuery | none | **role:** Web authorization/session facade.; **entrypoints:** parseForgejoKeysPage; **chain:** Web authorization/session facade. |
| [internal/web/avatars.go](../../../../internal/web/avatars.go) | ParseQuery; Values | none | **role:** Web server composition or feature adapter.; **entrypoints:** admitAvatarQuery; **chain:** Web server composition or feature adapter. |
| [internal/web/server.go](../../../../internal/web/server.go) | Parse; URL | none | **role:** Web server composition or feature adapter.; **entrypoints:** forgejoHome; **chain:** Web server composition or feature adapter. |
| [tools/soda-rootfs-server/main.go](../../../../tools/soda-rootfs-server/main.go) | Parse; PathUnescape | none | **role:** Developer/support command.; **entrypoints:** requestPath; **chain:** Developer/support command. |
| source consumers listed below | none | **role:** Soda-owned functions/types define policy; standard package supplies primitive transport, serialization, encoding, time or cryptographic operations. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/forgejo_onboarding_test.go](../../../../scripts/forgejo_onboarding_test.go) | Parse | Test/developer helper. |

<a id="integration-16"></a>

## Go standard library time

**Selected:** Go 1.26.7 module language/toolchain floor (go.mod). **Upstream capability used:** Deadlines, timestamps, leases, expiry, retry and duration calculations.

**Dependency admission:** go.mod `go 1.26.7`; no external module version applies.

**Primary-source evidence:** Direct source imports only; domain owner retains meaning of each deadline/clock field.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [cmd/soda-dashboard/extension.go](../../../../cmd/soda-dashboard/extension.go) | Second | none | **role:** Process entrypoint or command adapter.; **entrypoints:** extensionListener; **chain:** Process entrypoint or command adapter. |
| [cmd/soda-dashboard/main.go](../../../../cmd/soda-dashboard/main.go) | Second | none | **role:** Process entrypoint or command adapter.; **entrypoints:** run; **chain:** Process entrypoint or command adapter. |
| [cmd/soda-dashboard/operator.go](../../../../cmd/soda-dashboard/operator.go) | Minute; Second | none | **role:** Process entrypoint or command adapter.; **entrypoints:** operatorHTTPServer; **chain:** Process entrypoint or command adapter. |
| [cmd/soda-tailnet/command.go](../../../../cmd/soda-tailnet/command.go) | Second | none | **role:** Process entrypoint or command adapter.; **entrypoints:** execute; **chain:** Process entrypoint or command adapter. |
| [internal/acceptance/developer_access_session.go](../../../../internal/acceptance/developer_access_session.go) | Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** accessChecked; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/developer_access_transfer.go](../../../../internal/acceptance/developer_access_transfer.go) | Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** checkAccessSudo; checkCrossUserDenial; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/installed.go](../../../../internal/acceptance/installed.go) | Duration | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** Error; RestrictUmask; Unwrap; fail; failDetail; failParen; machineArch; ownedByCaller; privateDir; privateFile; runBounded; runBoundedDirEnv; runBoundedEnv; uuidHex; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/personal_git.go](../../../../internal/acceptance/personal_git.go) | Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** gitChecked; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/process.go](../../../../internal/acceptance/process.go) | After; Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** StartCommand; Stop; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/service_https.go](../../../../internal/acceptance/service_https.go) | Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** httpsClient; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_access.go](../../../../internal/acceptance/workload_access.go) | Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** accessRemote; clientPSQL; fetchAccessURL; **chain:** Developer acceptance/evidence support integration. |
| [internal/acceptance/workload_exec.go](../../../../internal/acceptance/workload_exec.go) | Second | none | **role:** Developer acceptance/evidence support integration.; **entrypoints:** checkMemberDenied; execDifferentUID; **chain:** Developer acceptance/evidence support integration. |
| [internal/factory/allowance.go](../../../../internal/factory/allowance.go) | Minute; Time | none | **role:** Direct package consumer.; **entrypoints:** Validate; **chain:** Direct package consumer. |
| [internal/factory/control/acceptance.go](../../../../internal/factory/control/acceptance.go) | Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** AdmitAcceptance; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/acceptance_initial.go](../../../../internal/factory/control/acceptance_initial.go) | Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** AdmitInitialAcceptance; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/acceptance_status.go](../../../../internal/factory/control/acceptance_status.go) | Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** WithdrawAcceptance; acceptanceStatus; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/checks_pass.go](../../../../internal/factory/control/checks_pass.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** assessPublicationChecksOne; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/coordinator.go](../../../../internal/factory/control/coordinator.go) | Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** Start; Status; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/correction.go](../../../../internal/factory/control/correction.go) | Minute; Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** PublishCorrection; adoptCorrectionOutcome; cancelRecordedCorrection; fenceCorrection; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_attempt.go](../../../../internal/factory/control/dispatch_attempt.go) | Time | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** admissionWait; dispatchOne; planAttempt; refreshOccupancy; waitFor; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_launch.go](../../../../internal/factory/control/dispatch_launch.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** executeFreshAttempt; retryAttempt; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_recovery.go](../../../../internal/factory/control/dispatch_recovery.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** finishTerminal; recoverSettled; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_registration.go](../../../../internal/factory/control/dispatch_registration.go) | Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** RegisterDispatch; ReopenDispatch; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_result.go](../../../../internal/factory/control/dispatch_result.go) | Minute; Nanosecond; Time | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** recordConfirmedUsage; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_selection.go](../../../../internal/factory/control/dispatch_selection.go) | Duration; Minute; Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** checkLimits; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/grant_authority.go](../../../../internal/factory/control/grant_authority.go) | Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** EffectiveAuthority; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/grants.go](../../../../internal/factory/control/grants.go) | Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** applyGrant; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/lifecycle.go](../../../../internal/factory/control/lifecycle.go) | Minute; Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** PauseRepository; ResumeRepository; StopProject; withdrawAndStopRuns; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/lifecycle_retry.go](../../../../internal/factory/control/lifecycle_retry.go) | Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** RetryRun; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/lifecycle_takeover.go](../../../../internal/factory/control/lifecycle_takeover.go) | Minute; Now; RFC3339 | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** TakeoverRun; VerifyProjectStart; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/merge.go](../../../../internal/factory/control/merge.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** finishMerge; mergeOne; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_evidence.go](../../../../internal/factory/control/merge_evidence.go) | Minute; Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** observeMergeEvidence; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_reconcile.go](../../../../internal/factory/control/merge_reconcile.go) | Now; Time | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** reconcileMerge; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_withdraw.go](../../../../internal/factory/control/merge_withdraw.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** withdrawMerge; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/preparation_decisions.go](../../../../internal/factory/control/preparation_decisions.go) | Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** admitDecision; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/publication.go](../../../../internal/factory/control/publication.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** publishOne; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_reconcile.go](../../../../internal/factory/control/publication_reconcile.go) | Minute; Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** reconcilePublication; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_withdraw.go](../../../../internal/factory/control/publication_withdraw.go) | Minute; Now; Time | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** finishPublication; publishAfterDispatch; withdrawPublication; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/readiness.go](../../../../internal/factory/control/readiness.go) | Minute; Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** ObserveIssueEvent; assessOne; record; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/readiness_sweep.go](../../../../internal/factory/control/readiness_sweep.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** ReconcileReadiness; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/review_cycle.go](../../../../internal/factory/control/review_cycle.go) | Now | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** SubmitReviewForRun; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/control/settle.go](../../../../internal/factory/control/settle.go) | After; Minute; Now; Second | none | **role:** Factory coordinator composition or runtime service.; **entrypoints:** Reconcile; Stop; finishAbandonedCommand; reconcileRuns; settleAbandonedCommand; settleRun; **chain:** Factory coordinator composition or runtime service. |
| [internal/factory/run.go](../../../../internal/factory/run.go) | Hour; Time | none | **role:** Direct package consumer.; **entrypoints:** validateDeadline; **chain:** Direct package consumer. |
| [internal/filelock/filelock.go](../../../../internal/filelock/filelock.go) | Millisecond; NewTicker | none | **role:** Cross-process lock adapter.; **entrypoints:** Acquire; **chain:** Cross-process lock adapter. |
| [internal/forgejo/client.go](../../../../internal/forgejo/client.go) | Second | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** New; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge.go](../../../../internal/forgejo/merge.go) | Now | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** SubmitMerge; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge_completion.go](../../../../internal/forgejo/merge_completion.go) | Now | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** matchMergeConfirmation; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/merge_observation.go](../../../../internal/forgejo/merge_observation.go) | Now | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** matchMergeTarget; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish.go](../../../../internal/forgejo/publish.go) | Second | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** checkActor; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation.go](../../../../internal/forgejo/publish/operation.go) | Minute; Now | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** prCreatePayload; publishPayload; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_observation.go](../../../../internal/forgejo/publish/operation_observation.go) | Now | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** ObserveForPublish; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_receipts.go](../../../../internal/forgejo/publish/operation_receipts.go) | Time | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** DecodePRCreateReceipt; DecodePublishReceipt; OperationNotAfter; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/review.go](../../../../internal/forgejo/review.go) | Now | none | **role:** Forgejo SDK/client adapter and domain conversion.; **entrypoints:** ObserveReview; SubmitReview; **chain:** Forgejo SDK/client adapter and domain conversion. |
| [internal/host/client.go](../../../../internal/host/client.go) | Minute | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** NewClient; **chain:** Typed native host client or privileged development executor. |
| [internal/host/terminal.go](../../../../internal/host/terminal.go) | Hour; Second; Time | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** Write; validTerminalWindow; **chain:** Typed native host client or privileged development executor. |
| [internal/host/terminal_client.go](../../../../internal/host/terminal_client.go) | Now; Second | none | **role:** Typed native host client or privileged development executor.; **entrypoints:** OpenTerminal; TerminalStates; **chain:** Typed native host client or privileged development executor. |
| [internal/identity/client/client.go](../../../../internal/identity/client/client.go) | Second | none | **role:** Direct package consumer.; **entrypoints:** New; **chain:** Direct package consumer. |
| [internal/identity/event.go](../../../../internal/identity/event.go) | Time | none | **role:** Direct package consumer.; **entrypoints:** Time; **chain:** Direct package consumer. |
| [internal/identity/types.go](../../../../internal/identity/types.go) | Hour; Time | none | **role:** Direct package consumer.; **entrypoints:** Validate; **chain:** Direct package consumer. |
| [internal/project/factory.go](../../../../internal/project/factory.go) | Time | none | **role:** Direct package consumer.; **entrypoints:** FactoryCodexGuest; FactoryPromptDigest; FactoryRunPaths; FactoryUnitName; ValidFactoryPhase; ValidFactoryRunID; ValidHarnessFamily; ValidHarnessVersion; Validate; **chain:** Direct package consumer. |
| [internal/store/factory.go](../../../../internal/store/factory.go) | RFC3339Nano; Time | none | **role:** Persistent store/driver integration.; **entrypoints:** FinishFactoryCommand; RecordFactoryCommand; **chain:** Persistent store/driver integration. |
| [internal/store/identity_fixture.go](../../../../internal/store/identity_fixture.go) | Now | none | **role:** Persistent store/driver integration.; **entrypoints:** appendIdentityEvent; **chain:** Persistent store/driver integration. |
| [internal/store/issue_controls.go](../../../../internal/store/issue_controls.go) | Time | none | **role:** Persistent store/driver integration.; **entrypoints:** AcceptanceDependants; FactoryPolicies; IntakeDeliverySeen; IssueControl; IssueControls; ReadinessSweepRevision; RecordIntakeDelivery; RecordIssueAssessment; SaveReadinessSweep; **chain:** Persistent store/driver integration. |
| [internal/factory/control/staged_seed_test.go](../../../../internal/factory/control/staged_seed_test.go) | Now | none | **role:** Test-only staged Forgejo fixture support.; **entrypoints:** seedStagedDependencyEdge; **chain:** Test-only staged Forgejo fixture support. |
| `internal/tailnet/control.go` (`../../../../internal/tailnet/control.go`; historical source locator) | Now; Second | none | **role:** Tailnet control/provider integration.; **entrypoints:** NewControl; RunNative; checkCredential; executeSignin; **chain:** Tailnet control/provider integration. |
| `internal/tailnet/enrollment.go` (`../../../../internal/tailnet/enrollment.go`; historical source locator) | Minute; Now; Second; Time | none | **role:** Tailnet control/provider integration.; **entrypoints:** EnrollRun; createProjectAuthKey; projectKey; tokenFromEnrollment; validAuthKeyLifetime; **chain:** Tailnet control/provider integration. |
| [internal/web/api/environment_os.go](../../../../internal/web/api/environment_os.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiEnvironmentOS; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/environments_create.go](../../../../internal/web/api/environments_create.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** resolveNativeProfile; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension.go](../../../../internal/web/api/extension.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** Extension; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal.go](../../../../internal/web/api/extension_terminal.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** extensionTerminalOperation; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal_authority.go](../../../../internal/web/api/extension_terminal_authority.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** extensionTerminalCurrent; extensionTerminalRepository; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/extension_terminal_stream.go](../../../../internal/web/api/extension_terminal_stream.go) | Hour; NewTicker; Now; Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** extensionOpenHostTerminal; extensionTerminalHeartbeat; pumpExtensionControls; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go) | Millisecond; NewTicker; Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** factoryOutputHeartbeat; factoryViewerCurrent; pumpFactoryOutput; readFactoryOutputHandshake; writeFactoryFrame; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/factory_views.go](../../../../internal/web/api/factory_views.go) | Time | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiFactoryRun; factoryHostTerminal; factoryRunBindingDTO; factoryRunLiveDTO; factoryRunRecordDTO; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/identity_launch.go](../../../../internal/web/api/identity_launch.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiIdentityLaunch; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/project_profiles.go](../../../../internal/web/api/project_profiles.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiProjectProfiles; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/provisioning.go](../../../../internal/web/api/provisioning.go) | Duration; Minute; Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** operationContext; provisioningConfirmed; reconcileProvisioning; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/repositories.go](../../../../internal/web/api/repositories.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiNativeRepositories; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/spaces.go](../../../../internal/web/api/spaces.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** apiSpaces; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/spaces_inspection.go](../../../../internal/web/api/spaces_inspection.go) | Millisecond; Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** inspectSpaceRow; inspectSpaceTailnet; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/api/terminal_registry.go](../../../../internal/web/api/terminal_registry.go) | Second | none | **role:** Product HTTP/WebSocket handler or adapter.; **entrypoints:** pumpNativeToExtension; readTerminalHandshake; **chain:** Product HTTP/WebSocket handler or adapter. |
| [internal/web/auth/extension.go](../../../../internal/web/auth/extension.go) | Second | none | **role:** Web authorization/session facade.; **entrypoints:** ExtensionAuthority; **chain:** Web authorization/session facade. |
| [internal/web/auth/extension_service.go](../../../../internal/web/auth/extension_service.go) | Second | none | **role:** Web authorization/session facade.; **entrypoints:** apiExtensionForgejoKeys; **chain:** Web authorization/session facade. |
| [internal/web/avatars.go](../../../../internal/web/avatars.go) | Time | none | **role:** Web server composition or feature adapter.; **entrypoints:** ServeHTTP; **chain:** Web server composition or feature adapter. |
| source consumers listed below | none | **role:** Soda-owned functions/types define policy; standard package supplies primitive transport, serialization, encoding, time or cryptographic operations. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [cmd/soda-dashboard/operator_linux_test.go](../../../../cmd/soda-dashboard/operator_linux_test.go) | Second | Process entrypoint or command adapter. |
| [internal/acceptance/process_linux_test.go](../../../../internal/acceptance/process_linux_test.go) | Hour; Millisecond; Now; Second; Sleep | Developer acceptance/evidence support integration. |
| [internal/factory/allowance_test.go](../../../../internal/factory/allowance_test.go) | Hour; Minute; Second; Time; Unix | Direct package consumer. |
| [internal/factory/control/checks_native_fixture_test.go](../../../../internal/factory/control/checks_native_fixture_test.go) | Millisecond; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/coordinator_test.go](../../../../internal/factory/control/coordinator_test.go) | Hour; Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/correction_test.go](../../../../internal/factory/control/correction_test.go) | Hour; Minute; Now; Second | Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_accounting_test.go](../../../../internal/factory/control/dispatch_accounting_test.go) | Hour; Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_concurrency_test.go](../../../../internal/factory/control/dispatch_concurrency_test.go) | Hour; Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_fixture_test.go](../../../../internal/factory/control/dispatch_fixture_test.go) | Now; Time | Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_recovery_test.go](../../../../internal/factory/control/dispatch_recovery_test.go) | Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_test.go](../../../../internal/factory/control/dispatch_test.go) | Hour; Minute; Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/dispatch_wait_test.go](../../../../internal/factory/control/dispatch_wait_test.go) | Hour; Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_fixture_test.go](../../../../internal/factory/control/merge_fixture_test.go) | Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_effect_test.go](../../../../internal/factory/control/merge_native_effect_test.go) | Millisecond; Minute; Now; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_fixture_test.go](../../../../internal/factory/control/merge_native_fixture_test.go) | Duration; Millisecond; Minute; Now; Second; Since; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_setup_test.go](../../../../internal/factory/control/merge_native_setup_test.go) | Duration; Hour; Millisecond; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/merge_native_test.go](../../../../internal/factory/control/merge_native_test.go) | Millisecond; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/operator_test.go](../../../../internal/factory/control/operator_test.go) | Now | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_fixture_test.go](../../../../internal/factory/control/publication_fixture_test.go) | Minute; Now; Second | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_effect_test.go](../../../../internal/factory/control/publication_native_effect_test.go) | Millisecond; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/publication_native_fixture_test.go](../../../../internal/factory/control/publication_native_fixture_test.go) | Hour; Millisecond; Minute; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/review_cycle_test.go](../../../../internal/factory/control/review_cycle_test.go) | Minute; Now; Second | Factory coordinator composition or runtime service. |
| [internal/factory/control/review_native_primitive_test.go](../../../../internal/factory/control/review_native_primitive_test.go) | Millisecond; Minute; Now; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_broker_test.go](../../../../internal/factory/control/st15_demo_broker_test.go) | Millisecond; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_coding_test.go](../../../../internal/factory/control/st15_demo_coding_test.go) | Duration; Millisecond; Minute; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_completion_test.go](../../../../internal/factory/control/st15_demo_completion_test.go) | After; Millisecond; Minute; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_project_test.go](../../../../internal/factory/control/st15_demo_project_test.go) | Minute; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_review_test.go](../../../../internal/factory/control/st15_demo_review_test.go) | Minute; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_runs_test.go](../../../../internal/factory/control/st15_demo_runs_test.go) | Minute; Nanosecond; Now; Second; Since; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/st15_demo_stack_test.go](../../../../internal/factory/control/st15_demo_stack_test.go) | Duration; Millisecond; Now; Second; Sleep | Factory coordinator composition or runtime service. |
| [internal/factory/control/traversal_test.go](../../../../internal/factory/control/traversal_test.go) | Duration; Hour; Now; Second | Factory coordinator composition or runtime service. |
| [internal/factory/merge_test.go](../../../../internal/factory/merge_test.go) | none | Direct package consumer. |
| [internal/factory/publication_test.go](../../../../internal/factory/publication_test.go) | Unix | Direct package consumer. |
| [internal/factory/run_test.go](../../../../internal/factory/run_test.go) | Hour; Now | Direct package consumer. |
| [internal/filelock/filelock_test.go](../../../../internal/filelock/filelock_test.go) | After; Millisecond; Second | Cross-process lock adapter. |
| [internal/forgejo/merge_test.go](../../../../internal/forgejo/merge_test.go) | Minute; Now | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/operation_test.go](../../../../internal/forgejo/publish/operation_test.go) | Hour; Minute; Now | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish/publish_test.go](../../../../internal/forgejo/publish/publish_test.go) | Hour; Now | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/publish_test.go](../../../../internal/forgejo/publish_test.go) | Hour; Now | Forgejo SDK/client adapter and domain conversion. |
| [internal/forgejo/review_test.go](../../../../internal/forgejo/review_test.go) | Minute; Now | Forgejo SDK/client adapter and domain conversion. |
| [internal/identity/client/broker_compat_test.go](../../../../internal/identity/client/broker_compat_test.go) | Hour; Millisecond; Now; Second; Sleep | Direct package consumer. |
| [internal/identity/types_test.go](../../../../internal/identity/types_test.go) | Hour; Now | Direct package consumer. |
| [internal/project/factory_test.go](../../../../internal/project/factory_test.go) | Hour; Now; Time | Direct package consumer. |
| [internal/store/factory_dispatch_queue_test.go](../../../../internal/store/factory_dispatch_queue_test.go) | Date; Hour; UTC | Persistent store/driver integration. |
| [internal/store/factory_dispatch_test.go](../../../../internal/store/factory_dispatch_test.go) | Date; Minute; Time; UTC | Persistent store/driver integration. |
| [internal/store/factory_grants_test.go](../../../../internal/store/factory_grants_test.go) | Date; Time; UTC | Persistent store/driver integration. |
| [internal/store/factory_inventory_test.go](../../../../internal/store/factory_inventory_test.go) | Date; Duration; Hour; Second; Time; UTC; Unix | Persistent store/driver integration. |
| [internal/store/factory_publications_test.go](../../../../internal/store/factory_publications_test.go) | Now; Second; Time | Persistent store/driver integration. |
| [internal/store/factory_retry_packet_test.go](../../../../internal/store/factory_retry_packet_test.go) | Date; Hour; UTC | Persistent store/driver integration. |
| [internal/store/factory_review_role_test.go](../../../../internal/store/factory_review_role_test.go) | Now; Second | Persistent store/driver integration. |
| [internal/store/factory_test.go](../../../../internal/store/factory_test.go) | Date; Hour; Time; UTC | Persistent store/driver integration. |
| [internal/store/factory_views_test.go](../../../../internal/store/factory_views_test.go) | Duration; Second | Persistent store/driver integration. |
| [internal/store/issue_controls_test.go](../../../../internal/store/issue_controls_test.go) | Hour; Now; Unix | Persistent store/driver integration. |
| `internal/tailnet/control_test.go` (`../../../../internal/tailnet/control_test.go`; historical source locator) | Millisecond; Second | Tailnet control/provider integration. |
| `internal/tailnet/enrollment_test.go` (`../../../../internal/tailnet/enrollment_test.go`; historical source locator) | Hour; Millisecond; Minute; Now; Sleep | Tailnet control/provider integration. |
| `internal/tailnet/policy_test.go` (`../../../../internal/tailnet/policy_test.go`; historical source locator) | Millisecond | Tailnet control/provider integration. |
| [internal/tailnet/tailnet_test.go](../../../../internal/tailnet/tailnet_test.go) | Millisecond; Now; Second; Since | Tailnet control/provider integration. |
| [internal/web/api/factory_output_test.go](../../../../internal/web/api/factory_output_test.go) | After; Second | Product HTTP/WebSocket handler or adapter. |
| [internal/web/extension_terminal_test.go](../../../../internal/web/extension_terminal_test.go) | Second | Web server composition or feature adapter. |
| [internal/web/factory_lifecycle_test.go](../../../../internal/web/factory_lifecycle_test.go) | Hour; Now | Web server composition or feature adapter. |
| [internal/web/factory_output_fixture_test.go](../../../../internal/web/factory_output_fixture_test.go) | Second | Web server composition or feature adapter. |
| [internal/web/factory_output_stream_test.go](../../../../internal/web/factory_output_stream_test.go) | Second | Web server composition or feature adapter. |
| [internal/web/factory_views_test.go](../../../../internal/web/factory_views_test.go) | Hour; Now | Web server composition or feature adapter. |
| [internal/web/join_boundary_test.go](../../../../internal/web/join_boundary_test.go) | After; Second | Web server composition or feature adapter. |
| [internal/web/provisioning_lifetime_test.go](../../../../internal/web/provisioning_lifetime_test.go) | Millisecond; Sleep | Web server composition or feature adapter. |
| [internal/web/spaces_inventory_test.go](../../../../internal/web/spaces_inventory_test.go) | Date; Hour; Now; Time; UTC | Web server composition or feature adapter. |
| [internal/web/spaces_test.go](../../../../internal/web/spaces_test.go) | After; Second | Web server composition or feature adapter. |
| [internal/web/terminal_test.go](../../../../internal/web/terminal_test.go) | Hour; Minute; Now; Time | Web server composition or feature adapter. |
| [scripts/forgejo_migrate_test.go](../../../../scripts/forgejo_migrate_test.go) | Minute | Test/developer helper. |
| [scripts/forgejo_presentation_test.go](../../../../scripts/forgejo_presentation_test.go) | Second | Test/developer helper. |
| [scripts/pg_backup_test.go](../../../../scripts/pg_backup_test.go) | Millisecond; Minute; Sleep | Test/developer helper. |
| [scripts/system_formats_test.go](../../../../scripts/system_formats_test.go) | Minute | Test/developer helper. |
| [scripts/wire_contracts_test.go](../../../../scripts/wire_contracts_test.go) | After; Minute; Second | Test/developer helper. |
| [tests/build/avatar_integration_test.go](../../../../tests/build/avatar_integration_test.go) | After; Millisecond; Second; Sleep | Test/developer helper. |
| [tests/build/complexity_scope_test.go](../../../../tests/build/complexity_scope_test.go) | Second | Test/developer helper. |
| [tests/build/helpers.go](../../../../tests/build/helpers.go) | Duration | Test/developer helper. |
| [tests/build/muse_exec_test.go](../../../../tests/build/muse_exec_test.go) | Now; Second; Since | Test/developer helper. |
| [tests/build/native_support_test.go](../../../../tests/build/native_support_test.go) | Second | Test/developer helper. |
| [tests/build/operator_probe_test.go](../../../../tests/build/operator_probe_test.go) | Second | Test/developer helper. |
| [tests/build/project_factory_roles_fixture_test.go](../../../../tests/build/project_factory_roles_fixture_test.go) | Duration; Millisecond; Now; Sleep | Test/developer helper. |
| [tests/build/project_factory_roles_lifecycle_test.go](../../../../tests/build/project_factory_roles_lifecycle_test.go) | Millisecond; Now; Second; Sleep | Test/developer helper. |
| [tests/build/project_factory_roles_output_test.go](../../../../tests/build/project_factory_roles_output_test.go) | Second | Test/developer helper. |
| [tests/build/sodaspaces_test.go](../../../../tests/build/sodaspaces_test.go) | Second | Test/developer helper. |
| [tests/build/source_checks_test.go](../../../../tests/build/source_checks_test.go) | Second | Test/developer helper. |
| [tests/build/workload_probe_test.go](../../../../tests/build/workload_probe_test.go) | Second | Test/developer helper. |
| [tools/soda-installed-probes/cockpit_account_test.go](../../../../tools/soda-installed-probes/cockpit_account_test.go) | Second | Developer/support command. |
| [tools/soda-installed-probes/installed_probes_test.go](../../../../tools/soda-installed-probes/installed_probes_test.go) | Duration | Developer/support command. |
| [tools/soda-installed-probes/project_state_test.go](../../../../tools/soda-installed-probes/project_state_test.go) | Second | Developer/support command. |
| [tools/soda-rootfs-server/main_test.go](../../../../tools/soda-rootfs-server/main_test.go) | Second | Developer/support command. |

<a id="integration-17"></a>

## lit

**Selected:** 3.3.3. **Upstream capability used:** LitElement reactive custom elements, html TemplateResult templates, directives, render and nothing sentinel.

**Dependency admission:** package.json dependencies.lit = 3.3.3; root Bun lock resolves exact package. package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Current imports in assets/branding/forgejo/lit.ts, frontend/spaces and frontend/tailnet; scripts/build-forgejo.ts bundles shared runtime for Forgejo modules, scripts/build-soda-extension.ts bundles extension entries; outputs staged to different host payloads.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [frontend/spaces/sodaspaces-workspace-view.ts](../../../../frontend/spaces/sodaspaces-workspace-view.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** renderMenu; renderWelcome; renderWelcomeSteps; renderWorkspaceIntro; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace.ts](../../../../frontend/spaces/sodaspaces-workspace.ts) | LitElement; html; repeat | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** SodaSpaces; mountSodaspaces; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-repository-picker-view.ts](../../../../frontend/spaces/sodaspaces-repository-picker-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** renderRepositoryPicker; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-factory.ts](../../../../frontend/spaces/sodaspaces-workspace-factory.ts) | TemplateResult | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** FactoryAdmission; FactoryCommandInput; FactoryDisplayInput; FactorySectionInput; FactoryWatchContextInput; FactoryWatchMutation; displayFactoryWatch; displayFactoryWatches; factoryRowDisabled; factorySection; factoryWatchContext; factoryWatchLimit; factoryWatching; onFactoryCommand; toggleFactoryWatch; unwatchRun; watchRun; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-network.ts](../../../../frontend/spaces/sodaspaces-network.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** renderNetwork; renderNetworkSelection; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-project-settings-view.ts](../../../../frontend/spaces/sodaspaces-project-settings-view.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** Lifecycle; SettingsViewInput; View; renderAccessView; renderEnvironmentView; renderNetworkView; renderRepositoryContext; renderViewTabs; views; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-inventory.ts](../../../../frontend/spaces/sodaspaces-workspace-inventory.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** ApiInput; InventoryInput; LiveReading; MoreProjectsInput; ReadSpacesInput; api; appendSpacesCollection; applySlotMetadata; applySpacesCollection; beginRefreshRead; clearConfigureAfterCreate; collectionStatus; existingSlotMetadata; finishRefreshSuccess; invalidateSlot; leaveJourneyManagement; live; loadMore; pageJourneyActive; readSpacesResponse; refresh; refreshBlocked; refreshFailed; refreshJourneyManagement; refreshLive; refreshSlots; renderMoreProjects; restoreIfNeeded; slotLost; syncPageJourney; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-setup.ts](../../../../frontend/spaces/sodaspaces-workspace-setup.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** SetupActions; SetupHost; SetupReading; SetupReturn; applyRepositorySearch; beginSetup; cancelSetup; changeRepository; configureProject; failRepositorySearch; focusSetupPanel; onRepositoryQuery; renderSetup; renderSetupBody; repositorySearchPath; searchAdmitted; searchCursor; searchRepositories; setupIntroAction; setupIntroDescription; setupIntroHeading; setupIntroHelper; setupIntroKind; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-layout.ts](../../../../frontend/spaces/sodaspaces-workspace-layout.ts) | Area; DividerArea; Minimum; Pane; Projection; Split; WorkspaceLayout; html; pane: Pane | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** ConsolidateInput; LayoutInput; MaximizedInput; MinimumInput; adjustDivider; arrange; canSplit; compact; consolidatePanes; dividerKey; dividerPointerMove; dragDivider; emptyPaneMinimum; focusPane; keyDivider; loadLayout; move; paneMinimum; persist; projection; renderPaneDivider; resizeSidebar; setSidebar; sidebarKey; split; splitBlocked; splitFits; toggleMaximizedPane; toggleSidebar; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-environment-view.ts](../../../../frontend/spaces/sodaspaces-environment-view.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** EnvironmentCommands; EnvironmentPresentation; renderEnvironment; renderOSObservation; renderProjectOS; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-session-navigation-view.ts](../../../../frontend/spaces/sodaspaces-session-navigation-view.ts) | TemplateResult; html; repeat | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** NavigationSession; SessionTab; renderProjectNavigation; renderSessionTab; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/soda-identity.ts](../../../../frontend/spaces/soda-identity.ts) | LitElement; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** SodaIdentity; mountIdentity; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-toolbar-view.ts](../../../../frontend/spaces/sodaspaces-workspace-toolbar-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** disableNewTerminal; hideNewTerminal; newTerminalButtonClass; newTerminalButtonLabel; renderBackButton; renderNativeManagementOption; renderOpenInDrawerOption; renderProjectSettingsButton; renderSpacesLink; renderToggleSidebarOption; renderToolbarProject; sessionsButtonHidden; sessionsButtonLabel; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-measurement.ts](../../../../frontend/spaces/sodaspaces-workspace-measurement.ts) | LitElement; ReactiveController | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** WorkspaceMeasurement; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-factory-navigation-view.ts](../../../../frontend/spaces/sodaspaces-factory-navigation-view.ts) | TemplateResult; html; repeat | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** FactoryRunRow; FactoryWatchSlot; renderFactoryRuns; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-project-view.ts](../../../../frontend/spaces/sodaspaces-project-view.ts) | TemplateResult; html; nothing | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** Connection; KeyCommands; KeyPresentation; renderConnection; renderForgejoKeys; renderKeys; renderProjectStatus; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-navigation.ts](../../../../frontend/spaces/sodaspaces-workspace-navigation.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** FilteredSpacesInput; NavReading; ProjectRowsInput; attentionRows; filteredSpaces; projectRows; rowAttention; rowName; rows; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-project.ts](../../../../frontend/spaces/sodaspaces-project.ts) | LitElement; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** ProjectContext; SodaProjectControls; mountProjectControls; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-terminal-dialog-view.ts](../../../../frontend/spaces/sodaspaces-terminal-dialog-view.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** CreationPresentation; renderCreation; renderRename; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-factory-view.ts](../../../../frontend/spaces/sodaspaces-factory-view.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** FactoryWatchCommands; FactoryWatchPresentation; renderFactoryWatch; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-shell-view.ts](../../../../frontend/spaces/sodaspaces-workspace-shell-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** busyAttr; hideCanvas; hideManagement; hideNavigation; hidePaneChrome; hideSidebarDivider; hideWorkspaceBody; renderFirstTerminalIntro; renderInventoryRecovery; renderSetupHeading; renderSetupTopbar; renderStatusBanners; workspaceBlocked; workspaceBodyClass; workspaceBodyStyle; workspaceClasses; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-workspace-pane-view.ts](../../../../frontend/spaces/sodaspaces-workspace-pane-view.ts) | !input.canSplit('below'); !input.canSplit('right'); !input.canSplit(axis; Area; LayoutEntry; Pane; Split; WorkspaceLayout; axis; html; moveTab; pane.key); panes; renderPaneLayoutExtras(input); repeat; splitPane | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** PaneChromeInput; beforeTabName; dropEdgeStyle; emptyPaneAttached; filterOverflowTabs; moveBeforeTab; moveTargetName; moveToPane; onDropEdgeDrop; onDropEdgeOver; onFocusPaneChange; onTabDragEnd; onTabDragStart; onTabDrop; onTabListDragOver; onTabListDrop; overflowTabLabel; paneActions; paneAriaOwns; paneChrome; paneSession; paneTabKeys; renderDropEdges; renderEmptyPane; renderFocusedPaneActions; renderMoveMenu; renderPaneLayoutExtras; renderPaneMenu; renderPaneSplitHelp; renderPaneSwitcher; renderTabOverflow; sessionRow; sessionTabProps; showMoveMenu; showPaneSwitcher; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-project-journey-view.ts](../../../../frontend/spaces/sodaspaces-project-journey-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** JourneyViewInput; renderJourney; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-terminal-view.ts](../../../../frontend/spaces/sodaspaces-terminal-view.ts) | TemplateResult; html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** TerminalCommands; TerminalPresentation; TerminalViewInput; renderTerminal; terminalCommands; terminalPresentation; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-terminal.ts](../../../../frontend/spaces/sodaspaces-terminal.ts) | LitElement | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** Renderer; SodaTerminal; TerminalContext; TerminalLocator; TerminalView; mountTerminal; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/spaces/sodaspaces-factory.ts](../../../../frontend/spaces/sodaspaces-factory.ts) | LitElement | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** FactoryWatchContext; SodaFactoryWatch; mountFactoryWatch; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/tailnet/soda-tailnet-host-view.ts](../../../../frontend/tailnet/soda-tailnet-host-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** ExitChoice; HostViewInput; renderHostSection; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/tailnet/soda-tailnet-enrollment-view.ts](../../../../frontend/tailnet/soda-tailnet-enrollment-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** EnrollmentViewInput; renderEnrollmentSection; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/tailnet/soda-tailnet-confirmation-view.ts](../../../../frontend/tailnet/soda-tailnet-confirmation-view.ts) | html | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** ConfirmationViewInput; renderPending; renderReconnect; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [frontend/tailnet/soda-tailnet-page.ts](../../../../frontend/tailnet/soda-tailnet-page.ts) | LitElement; html; this.message | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** mountTailnetPage; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| [assets/branding/forgejo/lit.ts](../../../../assets/branding/forgejo/lit.ts) | repeat | none | **role:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload.; **entrypoints:** repeat; **chain:** Forgejo branding or Soda Lit UI component/template; build output enters the corresponding browser payload. |
| assets/branding/forgejo/lit.ts; frontend/** | LitElement; html; TemplateResult; repeat; nothing; render; classMap as applicable | **role:** Forgejo branding wrappers and browser custom elements; Spaces/Tailnet component trees and typed presentational helpers.; **public types:** LitElement; TemplateResult; ReactiveController via local type imports; **conversions:** Soda model/read state -&gt; Lit render templates; no Go DTO runtime mapping. | none |
| [scripts/build-forgejo.ts](../../../../scripts/build-forgejo.ts) | Bun.build bare-lit resolution and runtime-entry handling | **role:** Builds shared Lit runtime and relative module imports into Forgejo public branding payload. | none |
| [scripts/build-soda-extension.ts](../../../../scripts/build-soda-extension.ts) | Bun.build extension entry bundling | **role:** Bundles Soda extension entry modules and shared chunks with Lit included in package graph. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [tests/forgejo/lit-build.test.ts](../../../../tests/forgejo/lit-build.test.ts) | classMap | Test/analyzer fixture or test-only Lit smoke component. |
| [tests/forgejo/fixtures/lit-smoke.ts](../../../../tests/forgejo/fixtures/lit-smoke.ts) | LitElement; css; html; nothing; render; repeat | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/positive.ts](../../../../tools/lit-check/fixtures/positive.ts) | LitElement; TemplateResult; html; repeat | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/property.ts](../../../../tools/lit-check/fixtures/property.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/unknownProperty.ts](../../../../tools/lit-check/fixtures/unknownProperty.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/boolean.ts](../../../../tools/lit-check/fixtures/boolean.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/event.ts](../../../../tools/lit-check/fixtures/event.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/unknownEvent.ts](../../../../tools/lit-check/fixtures/unknownEvent.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/customProperty.ts](../../../../tools/lit-check/fixtures/customProperty.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/directive.ts](../../../../tools/lit-check/fixtures/directive.ts) | html; repeat | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/unclosed.ts](../../../../tools/lit-check/fixtures/unclosed.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/nullable.ts](../../../../tools/lit-check/fixtures/nullable.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |
| [tools/lit-check/fixtures/aria.ts](../../../../tools/lit-check/fixtures/aria.ts) | html | Test/analyzer fixture or test-only Lit smoke component. |

**Transfer / predecessor evidence:** **item:** Cockpit custom React/PatternFly/Zustand/Vite frontend; **evidence:** Current worktree uses Lit, and docs/development/typescript.md states the old Cockpit workspace/build is retired.

<a id="integration-18"></a>

## @xterm/xterm

**Selected:** 6.0.0. **Upstream capability used:** Terminal emulator API for open/write/resize, parser/data/render events, addons, dispose and option types.

**Dependency admission:** tools/release-assets/terminal-assets.lock.json selected registry package version/integrity/files; package.json devDependency and bun.lock also pin declarations. package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Upstream source use is represented by selected package lock (xterm.mjs, xterm.css, LICENSE with SHA-256) and Soda wrapper imports/types. Runtime file is externally resolved by the extension bundler and copied from locked vendor directory.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [frontend/spaces/terminal-vendor.d.ts](../../../../frontend/spaces/terminal-vendor.d.ts) | Terminal | none | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** Terminal; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-factory-screen.ts](../../../../frontend/spaces/sodaspaces-factory-screen.ts) | ITerminalAddon | none | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** FactoryFit; FactoryScreenInput; awaitScreen; clearScreen; openScreen; resize; screenReady; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-terminal-screen.ts](../../../../frontend/spaces/sodaspaces-terminal-screen.ts) | ITerminalAddon; TerminalView | **role:** Fits screen, writes geometry/resize and disposes terminal through narrowed wrapper. | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** ScreenInput; TerminalFit; awaitScreen; clearScreen; openTerminal; resize; screenReady; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-terminal.ts](../../../../frontend/spaces/sodaspaces-terminal.ts) | ITerminalAddon; ITerminalInitOnlyOptions; ITerminalOptions; Terminal; dynamic import ./soda-terminal/xterm.mjs | **role:** Owns TerminalView narrowed API and Renderer construction; calls open/write/loadAddon/focus/dispose via lifecycle screen helpers.; **public types:** TerminalView; Renderer; TerminalContext; TerminalLocator; **conversions:** TypeScript caller boundary narrows upstream Terminal to the methods used; locked module constructor becomes renderer injection. | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** Renderer; SodaTerminal; TerminalContext; TerminalLocator; TerminalView; mountTerminal; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-factory.ts](../../../../frontend/spaces/sodaspaces-factory.ts) | ITerminalAddon | none | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** FactoryWatchContext; SodaFactoryWatch; mountFactoryWatch; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| tools/release-assets/src/bin/soda-fetch-terminal.rs; tools/release-assets/terminal-assets.lock.json | locked registry package/version/integrity/member/hash selection | none | **role:** Build-time asset fetch command copies only manifest-locked module, CSS and license; no runtime import.; **chain:** scripts/build:forgejo fetches into the ignored terminal vendor directory; scripts/build-soda-extension.ts copies the selected files into system/containers/extension asset output.; **entrypoints:** soda-fetch-terminal |
| [scripts/build-soda-extension.ts](../../../../scripts/build-soda-extension.ts) | copyLockedTerminal; buildScripts | none | **role:** Externalizes xterm/addon relative imports and copies locked terminal module assets into the browser package.; **chain:** Manifest pages/panels -&gt; Soda extension entry bundles -&gt; locked terminal asset copy -&gt; extension files.json/staging.; **entrypoints:** buildSodaExtensionAssets |
| frontend/spaces/sodaspaces-factory.ts, sodaspaces-factory-screen.ts | ITerminalAddon type | **role:** Factory output view uses the same terminal rendering API with a read-only interaction policy. | none |
| scripts/build-soda-extension.ts; tools/release-assets/src/bin/soda-fetch-terminal.rs | locked package extraction/copy and external module resolution | **role:** Fetches/verifies the locked distribution, copies selected module/CSS/license and leaves relative module import external. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [tests/frontend/fixtures/terminal-fixture.ts](../../../../tests/frontend/fixtures/terminal-fixture.ts) | ITerminalInitOnlyOptions; ITerminalOptions | Test fixture/assertion or developer-only tool caller. |

<a id="integration-19"></a>

## @xterm/addon-fit

**Selected:** 0.11.0. **Upstream capability used:** FitAddon API supplies fit() as a terminal addon and CSS for terminal rendering.

**Dependency admission:** tools/release-assets/terminal-assets.lock.json selected registry package version/integrity/files; package.json devDependency and bun.lock pin declarations. package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Current selected package lock records addon-fit.mjs and LICENSE hashes; dynamic relative import is externalized and the exact asset is copied into the extension package.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [frontend/spaces/terminal-vendor.d.ts](../../../../frontend/spaces/terminal-vendor.d.ts) | FitAddon | none | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** FitAddon; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-factory-screen.ts](../../../../frontend/spaces/sodaspaces-factory-screen.ts) | FitAddon | none | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** FactoryFit; FactoryScreenInput; awaitScreen; clearScreen; openScreen; resize; screenReady; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-terminal-screen.ts](../../../../frontend/spaces/sodaspaces-terminal-screen.ts) | FitAddon; FitAddon fit | **role:** Uses addon fit result to derive cols/rows and publishes resize frames. | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** ScreenInput; TerminalFit; awaitScreen; clearScreen; openTerminal; resize; screenReady; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-terminal.ts](../../../../frontend/spaces/sodaspaces-terminal.ts) | FitAddon; fit | **role:** Narrow Renderer addon constructor and call fit after terminal screen geometry is ready.; **public types:** Renderer; **conversions:** Upstream FitAddon is constrained to Pick&lt;FitAddon, fit&gt; & ITerminalAddon. | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** Renderer; SodaTerminal; TerminalContext; TerminalLocator; TerminalView; mountTerminal; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| [frontend/spaces/sodaspaces-factory.ts](../../../../frontend/spaces/sodaspaces-factory.ts) | FitAddon | none | **role:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import.; **entrypoints:** FactoryWatchContext; SodaFactoryWatch; mountFactoryWatch; **chain:** Compile-time type contract; production runtime is the locked copied .mjs asset loaded by dynamic relative import. |
| tools/release-assets/src/bin/soda-fetch-terminal.rs; tools/release-assets/terminal-assets.lock.json | locked registry package/version/integrity/member/hash selection | none | **role:** Build-time asset fetch command copies only manifest-locked module, CSS and license; no runtime import.; **chain:** scripts/build:forgejo fetches into the ignored terminal vendor directory; scripts/build-soda-extension.ts copies the selected files into system/containers/extension asset output.; **entrypoints:** soda-fetch-terminal |
| [scripts/build-soda-extension.ts](../../../../scripts/build-soda-extension.ts) | copyLockedTerminal; buildScripts | none | **role:** Externalizes xterm/addon relative imports and copies locked terminal module assets into the browser package.; **chain:** Manifest pages/panels -&gt; Soda extension entry bundles -&gt; locked terminal asset copy -&gt; extension files.json/staging.; **entrypoints:** buildSodaExtensionAssets |
| frontend/spaces/sodaspaces-factory.ts, sodaspaces-factory-screen.ts | FitAddon type | **role:** Factory read-only output screen shares measured terminal presentation. | none |
| scripts/build-soda-extension.ts; tools/release-assets/src/bin/soda-fetch-terminal.rs | locked package extraction/copy and external module resolution | **role:** Fetches/verifies/copies the selected addon module and license. | none |

<a id="integration-20"></a>

## jsdom

**Selected:** 30.0.1. **Upstream capability used:** A DOM implementation for source-level component/action tests under Bun.

**Dependency admission:** package.json devDependencies.jsdom package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Direct test imports instantiate JSDOM and exercise browser code in a local DOM; no browser payload import.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| tests/frontend/workspace-factory.test.ts; tests/forgejo/repository-actions.test.ts | JSDOM | **role:** Runs focused DOM behavior tests without Chromium. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [tests/forgejo/repository-actions.test.ts](../../../../tests/forgejo/repository-actions.test.ts) | JSDOM | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/workspace-factory.test.ts](../../../../tests/frontend/workspace-factory.test.ts) | JSDOM | Test fixture or browser/source assertion; never included in runtime payload. |

<a id="integration-21"></a>

## playwright

**Selected:** 1.63.0. **Upstream capability used:** Chromium browser automation with Page/Locator/WebSocket APIs for browser, presentation and opt-in installed journeys.

**Dependency admission:** package.json devDependencies.playwright package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Direct imports in tests and developer screenshot/branding scripts; does not ship in Forgejo or extension assets.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| tests/frontend/**; tests/forgejo/**; tests/installed/**; scripts/check-forgejo-branding.ts; scripts/screenshot.ts | chromium.launch; Page; Locator; Browser; WebSocket | **role:** Owns browser automation fixtures and journeys; installed suites are explicitly gated, local suites use fixtures. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [tests/installed/sodaspaces-cli.ts](../../../../tests/installed/sodaspaces-cli.ts) | Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/native-browser.ts](../../../../tests/installed/native-browser.ts) | Browser; BrowserType | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/sodaspaces-first-use-journey.ts](../../../../tests/installed/sodaspaces-first-use-journey.ts) | Page; WebSocket | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/sodaspaces-controls.ts](../../../../tests/installed/sodaspaces-controls.ts) | Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/sodaspaces-workspace-journey.ts](../../../../tests/installed/sodaspaces-workspace-journey.ts) | Page; WebSocket | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/operator.ts](../../../../tests/installed/operator.ts) | Page; chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/sodaspaces-journey-evidence.ts](../../../../tests/installed/sodaspaces-journey-evidence.ts) | Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/installed/sodaspaces-matrix-native.ts](../../../../tests/installed/sodaspaces-matrix-native.ts) | Page; WebSocket | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/explore-overflow.test.ts](../../../../tests/forgejo/explore-overflow.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/repository-switcher.test.ts](../../../../tests/forgejo/repository-switcher.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/repository-container.test.ts](../../../../tests/forgejo/repository-container.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/cockpit-branding.test.ts](../../../../tests/forgejo/cockpit-branding.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/lit-runtime.test.ts](../../../../tests/forgejo/lit-runtime.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/notification-preview.test.ts](../../../../tests/forgejo/notification-preview.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/milestones-layout.test.ts](../../../../tests/forgejo/milestones-layout.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/work-item-lists.test.ts](../../../../tests/forgejo/work-item-lists.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/identity.test.ts](../../../../tests/frontend/identity.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/native-project-controls.test.ts](../../../../tests/frontend/native-project-controls.test.ts) | chromium; type Browser; type Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/spaces-preview.test.ts](../../../../tests/frontend/spaces-preview.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/workspace-first-use.test.ts](../../../../tests/frontend/workspace-first-use.test.ts) | Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/persistent-panel.test.ts](../../../../tests/frontend/persistent-panel.test.ts) | chromium; type Browser | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/workspace-setup.test.ts](../../../../tests/frontend/workspace-setup.test.ts) | Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/profile-browser.test.ts](../../../../tests/forgejo/presentation/profile-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/settings-browser.test.ts](../../../../tests/forgejo/presentation/settings-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/gallery.test.ts](../../../../tests/forgejo/presentation/gallery.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/home-background-browser.test.ts](../../../../tests/forgejo/presentation/home-background-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/login-station-browser.test.ts](../../../../tests/forgejo/presentation/login-station-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/notification-layout-browser.test.ts](../../../../tests/forgejo/presentation/notification-layout-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/refinement-browser.test.ts](../../../../tests/forgejo/presentation/refinement-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/dashboard-sidebar-browser.test.ts](../../../../tests/forgejo/presentation/dashboard-sidebar-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/form-browser.test.ts](../../../../tests/forgejo/presentation/form-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/repository-actionbar-browser.test.ts](../../../../tests/forgejo/presentation/repository-actionbar-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/repository-settings-browser.test.ts](../../../../tests/forgejo/presentation/repository-settings-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/presentation/migration-browser.test.ts](../../../../tests/forgejo/presentation/migration-browser.test.ts) | chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/forgejo/fixtures/component-browser.ts](../../../../tests/forgejo/fixtures/component-browser.ts) | Page; chromium | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/fixtures/project-controls-driver.ts](../../../../tests/frontend/fixtures/project-controls-driver.ts) | chromium; type Browser; type Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/fixtures/terminal-driver.ts](../../../../tests/frontend/fixtures/terminal-driver.ts) | chromium; type Browser; type Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/fixtures/workspace-driver.ts](../../../../tests/frontend/fixtures/workspace-driver.ts) | chromium; type Browser; type Locator; type Page | Test fixture or browser/source assertion; never included in runtime payload. |
| [scripts/screenshot.ts](../../../../scripts/screenshot.ts) | Page; chromium | Developer/test tooling; not product runtime. |
| [scripts/check-forgejo-branding.ts](../../../../scripts/check-forgejo-branding.ts) | chromium; type Locator; type Page | Developer/test tooling; not product runtime. |

<a id="integration-22"></a>

## typescript

**Selected:** root 7.0.2; tools/lit-check workspace 5.9.3. **Upstream capability used:** Static type checking and analyzer compiler API (create program/source-file/type-checker diagnostics).

**Dependency admission:** root package.json devDependencies.typescript plus tools/lit-check/package.json devDependencies.typescript; root scripts invoke root tsc while checker workspace resolves its own compiler. package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Current typecheck command invokes root TypeScript 7 for root/browser/tests/tools configs; tools/lit-check/check.ts imports workspace TypeScript 5.9.3 for Lit analyzer integration. Neither compiler ships in browser payload.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| scripts/typecheck invocations; tools/lit-check/check.ts | tsc; createProgram/diagnostics via TypeScript API | **role:** Compile-time/type-analysis tooling only. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| package.json; scripts/check-lit.ts; tools/lit-check/tsconfig.json | tsc CLI; typescript compiler API | Root typecheck commands use TS 7.0.2; Lit checker workspace compiler API resolves TS 5.9.3. Developer-only. |
| [tools/lit-check/check.ts](../../../../tools/lit-check/check.ts) | ts | Developer/test tooling; not product runtime. |

<a id="integration-23"></a>

## lit-analyzer

**Selected:** 2.0.3. **Upstream capability used:** Analyze Lit tagged-template bindings against component properties/events; checker maps diagnostics to source and fails on selected errors.

**Dependency admission:** tools/lit-check/package.json devDependencies.lit-analyzer package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** tools/lit-check/check.ts imports LitAnalyzer/DefaultLitAnalyzerContext/makeConfig; scripts/check-lit.ts invokes the workspace CLI; fixtures validate checker behavior. Development-only.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| tools/lit-check/check.ts; scripts/check-lit.ts; tools/lit-check/fixtures/** | LitAnalyzer; DefaultLitAnalyzerContext; makeConfig | **role:** Development analyzer integration and fixtures; checks actual-source Lit bindings. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [tools/lit-check/check.ts](../../../../tools/lit-check/check.ts) | DefaultLitAnalyzerContext; LitAnalyzer; LitDiagnostic; makeConfig | Developer/test tooling; not product runtime. |

<a id="integration-24"></a>

## oxfmt

**Selected:** 0.68.0. **Upstream capability used:** TypeScript formatting CLI used by repository checks.

**Dependency admission:** package.json devDependencies.oxfmt; scripts/check-oxfmt.sh package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Root script invokes tool on selected/tracked TS files; no runtime module consumer or browser payload.

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/check-oxfmt.sh](../../../../scripts/check-oxfmt.sh) | oxfmt CLI path invocation | Repository formatting wrapper; developer tooling only. |

<a id="integration-25"></a>

## oxlint

**Selected:** 1.81.0. **Upstream capability used:** Correctness lint and browser-payload complexity CLI.

**Dependency admission:** package.json devDependencies.oxlint; scripts/check-oxlint.sh and check-ts-complexity.sh package.json SHA-256 30b065d8d8dd8af288a91bdab9b10cb20202cda3d1e447435505a60fb15f775f; bun.lock SHA-256 e14b67506a3d0c497c4a19a31289feec3a66574613944eb0b019132916ce0b86.

**Primary-source evidence:** Root scripts call binary with controlled scopes; no runtime library import.

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/check-oxlint.sh](../../../../scripts/check-oxlint.sh) | oxlint CLI path invocation | Repository correctness-lint wrapper; developer tooling only. |
| [scripts/check-ts-complexity.sh](../../../../scripts/check-ts-complexity.sh) | oxlint complexity CLI path invocation | Browser payload complexity wrapper; developer tooling only. |

<a id="integration-26"></a>

## @types/bun

**Selected:** 1.4.2. **Upstream capability used:** Type declarations for Bun tool/test APIs.

**Dependency admission:** package.json devDependencies.@types/bun

**Primary-source evidence:** TypeScript config includes Bun types for scripts/tests; erased from emitted JS.

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| tsconfig.json; tsconfig.tests.json | Bun ambient declarations | Compile-time types only. |

<a id="integration-27"></a>

## @types/node

**Selected:** 24.10.1. **Upstream capability used:** Type declarations for selected node: compatibility APIs executed under Bun (path, crypto, fs/promises, util, test).

**Dependency admission:** package.json devDependencies.@types/node

**Primary-source evidence:** TypeScript configurations resolve node types; actual runtime is Bun, not Node process.

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| tsconfig.json; tsconfig.tests.json | node: ambient declarations | Compile-time types only for selected Bun-compatible node: imports. |

<a id="integration-28"></a>

## @types/jsdom

**Selected:** 30.0.0. **Upstream capability used:** Type declarations for jsdom test fixtures.

**Dependency admission:** package.json devDependencies.@types/jsdom

**Primary-source evidence:** Test TypeScript only; no production/runtime browser API.

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [tests/forgejo/repository-actions.test.ts](../../../../tests/forgejo/repository-actions.test.ts) | JSDOM | Test fixture or browser/source assertion; never included in runtime payload. |
| [tests/frontend/workspace-factory.test.ts](../../../../tests/frontend/workspace-factory.test.ts) | JSDOM | Test fixture or browser/source assertion; never included in runtime payload. |
| [tsconfig.tests.json](../../../../tsconfig.tests.json) | jsdom ambient declarations | Compile-time type declarations only. |

<a id="integration-29"></a>

## Go module dependency-only / indirect set

**Selected:** selected versions as go.mod/go.sum. **Upstream capability used:** Transitive APIs needed by selected direct modules/tool commands; not an independently used Soda application integration unless a direct import is listed above.

**Dependency admission:** Current go.mod indirect requirements and unchanged selected-module evidence. pgx/v5 is a direct API integration despite its indirect marker; four explicit Go tool modules have separate developer-consumer entries.

**Primary-source evidence:** Current module graph manifests only; no transitive implementation audit performed.

**dependency only modules:** **module:** github.com/BurntSushi/toml; **requirement:** v1.4.1-0.20240526193622-a339e1f7089c; **selected:** v1.4.1-0.20240526193622-a339e1f7089c; **module:** github.com/dicebear/schema; **requirement:** v1.5.1; **selected:** v1.5.1; **module:** github.com/dustin/go-humanize; **requirement:** v1.0.1; **selected:** v1.0.1; **module:** github.com/fatih/color; **requirement:** v1.13.0; **selected:** v1.13.0; **module:** github.com/golang/protobuf; **requirement:** v1.5.4; **selected:** v1.5.4; **module:** github.com/google/go-cmp; **requirement:** v0.7.0; **selected:** v0.7.0; **module:** github.com/google/uuid; **requirement:** v1.6.0; **selected:** v1.6.0; **module:** github.com/hashicorp/go-hclog; **requirement:** v1.6.3; **selected:** v1.6.3; **module:** github.com/hashicorp/go-plugin; **requirement:** v1.8.0; **selected:** v1.8.0; **module:** github.com/hashicorp/yamux; **requirement:** v0.1.2; **selected:** v0.1.2; **module:** github.com/jackc/pgpassfile; **requirement:** v1.0.0; **selected:** v1.0.0; **module:** github.com/jackc/pgservicefile; **requirement:** v0.0.0-20240606120523-5a60cdf6a761; **selected:** v0.0.0-20240606120523-5a60cdf6a761; **module:** github.com/jackc/puddle/v2; **requirement:** v2.2.2; **selected:** v2.2.2; **module:** github.com/mattn/go-colorable; **requirement:** v0.1.12; **selected:** v0.1.12; **module:** github.com/mattn/go-isatty; **requirement:** v0.0.24; **selected:** v0.0.24; **module:** github.com/ncruces/go-strftime; **requirement:** v1.0.0; **selected:** v1.0.0; **module:** github.com/oklog/run; **requirement:** v1.1.0; **selected:** v1.1.0; **module:** github.com/remyoudompheng/bigfft; **requirement:** v0.0.0-20230129092748-24d4a6f8daec; **selected:** v0.0.0-20230129092748-24d4a6f8daec; **module:** github.com/santhosh-tekuri/jsonschema/v6; **requirement:** v6.0.2; **selected:** v6.0.2; **module:** github.com/tailscale/hujson; **requirement:** v0.0.0-20220506213045-af5ed07155e5; **selected:** v0.0.0-20220506213045-af5ed07155e5; **module:** go.yaml.in/yaml/v3; **requirement:** v3.0.5; **selected:** v3.0.5; **module:** golang.org/x/exp/typeparams; **requirement:** v0.0.0-20231108232855-2478ac86f678; **selected:** v0.0.0-20231108232855-2478ac86f678; **module:** golang.org/x/mod; **requirement:** v0.38.0; **selected:** v0.38.0; **module:** golang.org/x/net; **requirement:** v0.57.0; **selected:** v0.57.0; **module:** golang.org/x/sync; **requirement:** v0.22.0; **selected:** v0.22.0; **module:** golang.org/x/text; **requirement:** v0.41.0; **selected:** v0.41.0; **module:** golang.org/x/tools; **requirement:** v0.48.0; **selected:** v0.48.0; **module:** google.golang.org/genproto/googleapis/rpc; **requirement:** v0.0.0-20231106174013-bbf56f31fb17; **selected:** v0.0.0-20231106174013-bbf56f31fb17; **module:** google.golang.org/grpc; **requirement:** v1.61.0; **selected:** v1.61.0; **module:** google.golang.org/protobuf; **requirement:** v1.36.6; **selected:** v1.36.6; **module:** modernc.org/libc; **requirement:** v1.75.6; **selected:** v1.75.6; **module:** modernc.org/mathutil; **requirement:** v1.7.1; **selected:** v1.7.1; **module:** modernc.org/memory; **requirement:** v1.12.1; **selected:** v1.12.1

**Evidence limits / remaining questions:** These 33 modules support the selected dependency graph. No Soda-authored direct API imports were found in current tracked Go source; their transitive source bodies were not audited. This disposition is not retirement/deletion evidence.

<a id="integration-30"></a>

## github.com/fzipp/gocyclo

**Selected:** v0.6.0. **Upstream capability used:** Complexity score CLI for shipping Go production source.

**Dependency admission:** go.mod tool block declares github.com/fzipp/gocyclo; pinned module version in go.mod require block.

**Primary-source evidence:** go tool gocyclo; `-over 9` excludes scripts/tests per wrapper.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [scripts/check-complexity.sh](../../../../scripts/check-complexity.sh) | Go tool CLI wrapper | **role:** Development quality gate only; not a Soda product/runtime dependency. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/check-complexity.sh](../../../../scripts/check-complexity.sh) | tool invocation | Developer-only Go source check; no production import. |

<a id="integration-31"></a>

## github.com/kisielk/errcheck

**Selected:** v1.9.0. **Upstream capability used:** Unchecked Go error static analysis.

**Dependency admission:** go.mod tool block declares github.com/kisielk/errcheck; pinned module version in go.mod require block.

**Primary-source evidence:** Builds/runs errcheck against Linux Go packages.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [scripts/check-errcheck.sh](../../../../scripts/check-errcheck.sh) | Go tool CLI wrapper | **role:** Development quality gate only; not a Soda product/runtime dependency. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/check-errcheck.sh](../../../../scripts/check-errcheck.sh) | tool invocation | Developer-only Go source check; no production import. |

<a id="integration-32"></a>

## honnef.co/go/tools/cmd/staticcheck

**Selected:** v0.8.1. **Upstream capability used:** Static analysis for Linux Go source.

**Dependency admission:** go.mod tool block declares honnef.co/go/tools/cmd/staticcheck; pinned module version in go.mod require block.

**Primary-source evidence:** Builds/runs staticcheck under Linux target from the pinned module tool.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [scripts/check-staticcheck.sh](../../../../scripts/check-staticcheck.sh) | Go tool CLI wrapper | **role:** Development quality gate only; not a Soda product/runtime dependency. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/check-staticcheck.sh](../../../../scripts/check-staticcheck.sh) | tool invocation | Developer-only Go source check; no production import. |

<a id="integration-33"></a>

## mvdan.cc/gofumpt

**Selected:** v0.9.1. **Upstream capability used:** Go formatter CLI.

**Dependency admission:** go.mod tool block declares mvdan.cc/gofumpt; pinned module version in go.mod require block.

**Primary-source evidence:** go tool gofumpt wrapper checks/rewrites selected Go sources.

| Current source | Function / upstream API selectors | Remaining types, conversions and policy | Reverse callers / hosting targets |
| --- | --- | --- | --- |
| [scripts/check-gofumpt.sh](../../../../scripts/check-gofumpt.sh) | Go tool CLI wrapper | **role:** Development quality gate only; not a Soda product/runtime dependency. | none |

### Test and fixture uses

| Test/helper source | Named selectors | Purpose |
| --- | --- | --- |
| [scripts/check-gofumpt.sh](../../../../scripts/check-gofumpt.sh) | tool invocation | Developer-only Go source check; no production import. |

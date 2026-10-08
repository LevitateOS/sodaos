# Backend web api

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-e071a157de6e"></a>

## [internal/web/api/access_keys.go](../../../../../internal/web/api/access_keys.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–133; file scaffold; accessKeyRevision; accessKeysEnvironment; authorizeAccessKeysMember; savedAccessKeyMaterial; validAccessKeyConfirmation; applyAccessKeyMutation; nativeAccessFingerprints; apiAccessKeys | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-02c4cfae64a5"></a>

## [internal/web/api/api.go](../../../../../internal/web/api/api.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 24, 27, 36, 43–46, 48–49; whole file; API; API.Auth; TerminalPeer; New | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 20–21; API.Config, API.Store | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Configured service dependencies and restricted input references; declarations/fields: `API.Config`, `API.Store` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 22; API.Forgejo | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Authoritative native read client dependency; declarations/fields: `API.Forgejo` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 23; API.Host | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Private host IPC dependency; declarations/fields: `API.Host` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 25; API.Identity | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Native identity metadata/admission client dependency; declarations/fields: `API.Identity` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 26; API.Coordinator | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Exclusive factory coordination dependency; declarations/fields: `API.Coordinator` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 28, 31–35, 37–42; API.terminalMu; API.TerminalPeers, API.TerminalStopping, API.terminalClosed, API.terminalWG; TerminalPeer | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Human terminal registry mutex; declarations/fields: `API.terminalMu`; Human terminal lifetime/cancellation registry; declarations/fields: `API.TerminalPeers`, `API.TerminalStopping`, `API.terminalClosed`, `API.terminalWG`; Tracked native terminal operation/stream lifetime and cancellation; declarations/fields: `TerminalPeer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 29, 47; API.SpacesSlots | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Bounded concurrent authorized inventory observations; declarations/fields: `API.SpacesSlots`; Spaces and repository lookup admission slots share constructor expression; declarations/fields: `API.SpacesSlots` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 30, 47; API.RepositorySlots | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Bounded concurrent repository-choice observations; declarations/fields: `API.RepositorySlots`; Repository lookup admission slot in shared constructor; declarations/fields: `API.RepositorySlots` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-bfff581c5c93"></a>

## [internal/web/api/dispatch_inputs.go](../../../../../internal/web/api/dispatch_inputs.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; file scaffold; DispatchSnapshotSource | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; Current declaration duty: DispatchSnapshotSource — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 28–90; ReadDispatchInputs | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: ReadDispatchInputs — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bd8a28fe765f"></a>

## [internal/web/api/dispatch_inputs_test.go](../../../../../internal/web/api/dispatch_inputs_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–32, 71–101; file scaffold; dispatchReaderFake; ReadNativeRevision; ReadSnapshot; TestDispatchSnapshotSourceMissingTipReadsEmpty; TestDispatchSnapshotSourceMapsTransportFailures | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 33–70; dispatchTestSnapshot; TestDispatchSnapshotSourceMapsInputs | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: dispatchTestSnapshot; Current declaration duty: TestDispatchSnapshotSourceMapsInputs — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d5401ed1f9f4"></a>

## [internal/web/api/environment_authority.go](../../../../../internal/web/api/environment_authority.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27, 60–81, 93–141; whole file; (declaration group); repositoryAccess; repositoryExecutionAllowed; API.executionRepository; reportExecutionAuthorityError; environmentReader; errEnvironmentReadStore; API.readEnvironmentAuthority; API.authorizeEnvironmentRead | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 28–59; API.visibleRepository, API.nativeVisibleRepository | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Current native repository evidence projection; callers own product admission; declarations/fields: `API.visibleRepository`, `API.nativeVisibleRepository` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 82–92; API.environmentAdministrator | [P05](../../slices/projects.md#p05-project-startstop) | retained | Project lifecycle/preparation administrator authority from current native repository; declarations/fields: `API.environmentAdministrator` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-fabe43ed74eb"></a>

## [internal/web/api/environment_os.go](../../../../../internal/web/api/environment_os.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; file scaffold; rejectOSQuery; confirmOSSession; apiEnvironmentOS | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e46927017b7b"></a>

## [internal/web/api/environments_api.go](../../../../../internal/web/api/environments_api.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14, 16–20, 24–33, 35–53, 62–69, 100–112, 127–131, 140–147, 154, 159; whole file; EnvironmentView; repositoryContextView; EnvironmentDTO; p.ID, p.Name, p.RepositoryID, p.Repository, p.OwnerID; environmentListQuery; parseEnvironmentsListQuery; API.requireListedSession; API.loadEnvironment; apiEnvironment; reconcileProvisioning; EnvironmentView.ID, EnvironmentView.Name, EnvironmentView.RepositoryID, EnvironmentView.Repository, EnvironmentView.OwnerID; EnvironmentDTO(p).ID, EnvironmentDTO(p).Name, EnvironmentDTO(p).RepositoryID, EnvironmentDTO(p).Repository, EnvironmentDTO(p).OwnerID | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 13 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 15, 21, 32, 113–126, 136–139, 151, 154–156, 159–161; EnvironmentView.Profile; EnvironmentView.Provisioned; p.Profile, p.Ready; environmentProfileMismatch, observedEnvironment, API.apiEnvironment; EnvironmentView.Profile, EnvironmentView.Provisioned; observed, native_unavailable; EnvironmentDTO(p).Profile, EnvironmentDTO(p).Provisioned, observed, nativeErr != nil | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Immutable selected native creation profile in association DTO; declarations/fields: `EnvironmentView.Profile`; 7 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 22–23, 34, 54–61, 70–99; EnvironmentView, EnvironmentDTO, listedEnvironments, API.apiEnvironments | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Bounded authorized association list presentation; declarations/fields: `EnvironmentView`, `EnvironmentDTO`, `listedEnvironments`, `API.apiEnvironments` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 132–135, 152–153, 157–159, 162–197; authorizeEnvironmentRead; authority_unavailable, execution_allowed; login; environment_administrator; reader.authorityUnavailable, reader.executionAllowed, reader.login, reader.administrator; API.apiEnvironmentMembers | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Authorize own membership/current repository visibility; declarations/fields: `authorizeEnvironmentRead`; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 148–150; requireListedSession | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Fresh native session recheck before publishing detail; declarations/fields: `requireListedSession` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 159; observed.IP | [N02](../../slices/networking.md#n02-project-lan-access) | retained | Native observed LAN IP carried in compound runtime observation reference; declarations/fields: `observed.IP` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 198–226; API.apiConnection | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Own development SSH connection observation; declarations/fields: `API.apiConnection` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-bcac6b63d5d6"></a>

## [internal/web/api/environments_create.go](../../../../../internal/web/api/environments_create.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–73, 121–185, 205–266; whole file; apiError; createEnvironmentInput; (declaration group); parseCreateEnvironmentInput; API.verifyRepositoryOwner; API.lookupReservation; API.reconfirmRepositoryAndSession; API.provisionAndSaveProject; API.reconcileCreate; API.apiCreateEnvironment | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 11 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 74–90, 186–204; API.checkTailnetPreflight, API.applyCreatedEnvironmentTailnet | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Explicit creation-time original Tailnet selection; declarations/fields: `API.checkTailnetPreflight`, `API.applyCreatedEnvironmentTailnet` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 91–113; API.resolveNativeProfile, API.precheckProfileAndTailnet | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Installed runtime profile preflight before reservation; declarations/fields: `API.resolveNativeProfile`, `API.precheckProfileAndTailnet` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 114–120; API.verifyCurrentSessionMatch | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Fresh native session admission recheck; declarations/fields: `API.verifyCurrentSessionMatch` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-6e18ba4f16af"></a>

## [internal/web/api/environments_join.go](../../../../../internal/web/api/environments_join.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 37–142; whole file; API.persistEnvironmentJoin; (declaration group); API.writeJoinLogin; API.reportJoinPersist; API.admitNewJoin; API.apiJoinEnvironment; API.currentJoinSession | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 8 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 13–36; validJoinSSHSelection, API.joinPublicKeys, errTooManyJoinKeys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Explicit own saved-or-none SSH key choice for new human account; declarations/fields: `validJoinSSHSelection`, `API.joinPublicKeys`, `errTooManyJoinKeys` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-b7201369526a"></a>

## [internal/web/api/environments_join_test.go](../../../../../internal/web/api/environments_join_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; file scaffold; TestJoinRejectsNonProjectLoginBeforeProvisioning | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestJoinRejectsNonProjectLoginBeforeProvisioning — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6a3b63c40fb7"></a>

## [internal/web/api/environments_preparation.go](../../../../../internal/web/api/environments_preparation.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–91, 104–116; whole file; preparationItemView; environmentPreparationView; preparationItemDTO; API.apiPreparation; API.preparationItems | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 92–103, 117–190; API.preparationHoldState, preparationHoldRequest, API.apiPreparationHold, API.syncPreparationHold | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Preparation maintenance hold synchronization/observation; declarations/fields: `API.preparationHoldState`, `preparationHoldRequest`, `API.apiPreparationHold`, `API.syncPreparationHold` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 191–239; spacePreparationRole, spacePreparationView, summarizeSpacePreparation, API.inspectSpacePreparation | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Readiness summary in authorized Spaces row; declarations/fields: `spacePreparationRole`, `spacePreparationView`, `summarizeSpacePreparation`, `API.inspectSpacePreparation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-e3e1ee1f3d8c"></a>

## [internal/web/api/extension.go](../../../../../internal/web/api/extension.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–102, 110–212, 232–254, 269–327, 379–400; file scaffold; Extension; boundExtensionResponse; extensionCleanRoute; extensionSessionRoute; extensionMeRoute; extensionDevelopmentKeyRoute; extensionQueryRoute; extensionProductRoute; extensionProductID; extensionRepositoryRoute; extensionEnvironmentRoute; extensionEnvironmentPreparationPath; extensionEnvironmentIdentityPath; extensionEnvironmentIdentityRoute; extensionIdentityRoute; extensionIdentityActionRoute; extensionIdentityConnectionsRoute; extensionIdentityEnrollmentsRoute; rewriteExtensionRequest | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 20 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 103–109, 346–353, 359–378; extensionRoute; extensionTerminalStreamRoute; extensionTerminalRoute; extensionTerminalSessionRoute; terminalControlMethod | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: extensionRoute; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 213–231; extensionFactoryRoute | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: extensionFactoryRoute — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 255–268, 328–345; extensionEnvironmentActionRoute; extensionSettingsRoute; extensionTailnetSettingsRoute | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current declaration duty: extensionEnvironmentActionRoute; Current declaration duty: extensionSettingsRoute; Current declaration duty: extensionTailnetSettingsRoute — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 354–358; extensionFactoryOutputStreamRoute | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: extensionFactoryOutputStreamRoute — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-54abe9eb2253"></a>

## [internal/web/api/extension_native.go](../../../../../internal/web/api/extension_native.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–134, 142–178, 233–236; file scaffold; extensionAuthorityKey; requestExtensionAuthority; extensionProtected; extensionProductAuthority; logExtensionDenial; errorString; extensionGenerationCurrent; extensionProductActor; extensionProductMutation; extensionProductUser; extensionPageContribution; extensionAdminPageContribution; extensionWorkspacePanelContribution; extensionMutationOrigin; extensionSessionCurrent; extensionAuthorityCurrent; extensionActor | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 18 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 135–141, 179–232; extensionProductContribution; registerExtensionProductRoutes | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current declaration duty: extensionProductContribution; Current declaration duty: registerExtensionProductRoutes — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e7cbee456996"></a>
<a id="internalwebapiextension_terminalgo-1"></a>

## [internal/web/api/extension_terminal.go](../../../../../internal/web/api/extension_terminal.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18, 40–44, 170–179; file scaffold; privateTerminalStreamRoute; extensionTerminalStream | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; Current declaration duty: privateTerminalStreamRoute; Current declaration duty: extensionTerminalStream — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 19–39; ExtensionHandler | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: ExtensionHandler — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 45–169; extensionTerminalOperation; extensionTerminalStates; extensionTerminalView; extensionReserveTerminal; extensionTerminalSession; extensionTerminalSessionMethod | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Current declaration duty: extensionTerminalOperation; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bdaff5e13582"></a>

## [internal/web/api/extension_terminal_authority.go](../../../../../internal/web/api/extension_terminal_authority.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–126; file scaffold; extensionTerminalIdentity; nativeTerminalScope; extensionTerminalGeneration; extensionTerminalOrigin; nativeTerminalContribution; nativeTerminalMember; nativeTerminalActor; extensionTerminalAccount; sameNativeTerminalAuthority; nativeTerminalMembershipCurrent; extensionTerminalRepository; extensionTerminalCurrent; nativeTerminalSessionActor | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-30326f476fb7"></a>

## [internal/web/api/extension_terminal_stream.go](../../../../../internal/web/api/extension_terminal_stream.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; file scaffold; extensionTerminalAttach; registerExtensionTerminalPeer; validNativeTerminalHandshake; extensionClaimTerminal; extensionOpenHostTerminal; pumpExtensionControls; extensionTerminalHeartbeat | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5dac9632c419"></a>

## [internal/web/api/extension_test.go](../../../../../internal/web/api/extension_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–170, 192–213; whole file; TestExtensionResponseIsBounded; TestExtensionRejectsUnknownRouteAndUnverifiedActor; TestExtensionAuthorityDenialLogsCause; TestExtensionProductAuthorityDenialLogsCause; TestExtensionProductRouteAllowlist; TestExtensionProductContributionScopes; TestExtensionTransportDoesNotForwardBrowserCredentials | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 8 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 171–191, 214–238; TestNativeTerminalContributionScopes, TestExtensionTerminalUpgradeForwardsOnlyProtocolAndAuthority | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Native terminal WebSocket header/authority proxy admission assertions; declarations/fields: `TestNativeTerminalContributionScopes`, `TestExtensionTerminalUpgradeForwardsOnlyProtocolAndAuthority` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-b018d5c05a94"></a>

## [internal/web/api/factory_assignments.go](../../../../../internal/web/api/factory_assignments.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30, 63–64, 67–113, 136–187; whole file; factoryAssignmentResult; factoryAssignmentView; factoryAssignmentDTO; currentIssueAssignment; API.apiFactoryAssignment | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 31–40, 65, 114–119; factoryAssignmentReservation; factoryAssignmentView.Reservation; factoryAssignmentDTO | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Assignment reservation read model; declarations/fields: `factoryAssignmentReservation`; Assignment view projects independently owned reservation accounting; declarations/fields: `factoryAssignmentView.Reservation`; Project assignment reservation accounting in composite assignment DTO; declarations/fields: `factoryAssignmentDTO` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 41–62, 66, 120–135; factoryAssignmentPublication; factoryAssignmentView.Publication; factoryAssignmentDTO | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Assignment publication exact-link/effect read model; declarations/fields: `factoryAssignmentPublication`; Assignment view projects independently owned publication progression; declarations/fields: `factoryAssignmentView.Publication`; Project exact publication link/effect/completion in composite assignment DTO; declarations/fields: `factoryAssignmentDTO` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-d0227b7d5d72"></a>

## [internal/web/api/factory_assignments_test.go](../../../../../internal/web/api/factory_assignments_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–66; whole file; TestFactoryAssignmentDTORendersInputsAndResult; TestCurrentIssueAssignmentSelectsLatest | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Assertions TestFactoryAssignmentDTORendersInputsAndResult: view = %+v; declarations/fields: `TestFactoryAssignmentDTORendersInputsAndResult`; Assertions TestCurrentIssueAssignmentSelectsLatest: empty selection accepted; declarations/fields: `TestCurrentIssueAssignmentSelectsLatest` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 67–77; TestFactoryAssignmentDTOPreservesIncompletePublication | [F09](../../slices/factory-coordination.md#f09-publication-progression) | retained | Publication committed effect/completion visibility assertions; declarations/fields: `TestFactoryAssignmentDTOPreservesIncompletePublication` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-366bca9b9ecc"></a>

## [internal/web/api/factory_intake.go](../../../../../internal/web/api/factory_intake.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–219; file scaffold; IntakeCoordinator; IntakeHandler; intakeHeader; intakeOutcome; ServeHTTP; verifyIntakeSignature; intakePermissions; intakeRepository; intakeSender; intakeIssue; intakeIssueEvent; intakeCommentEvent; parseIntakeHint | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0ed9d77a0a9e"></a>

## [internal/web/api/factory_intake_test.go](../../../../../internal/web/api/factory_intake_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–179; whole file; stubIntakeCoordinator; stubIntakeCoordinator.ObserveIssueEvent; signIntake; intakeRequest; TestFactoryIntakeOpenedCarriesCreatorAuthority; TestFactoryIntakeHints; TestFactoryIntakeRejectsForgedDeliveries; TestFactoryIntakeFailures | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 9 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 180–195; TestServiceReadinessSourceNilGuards | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Unwired evidence source refusal assertions; declarations/fields: `TestServiceReadinessSourceNilGuards` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-451d790c97ec"></a>

## [internal/web/api/factory_issue_view.go](../../../../../internal/web/api/factory_issue_view.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–153; file scaffold; factoryIssueView; readinessView; checkResultView; checksView; checksViewDTO; mergeView; mergeViewDTO; apiFactoryIssue | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-04c2577d784a"></a>

## [internal/web/api/factory_lifecycle.go](../../../../../internal/web/api/factory_lifecycle.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–264; file scaffold; lifecycleControlError; factoryActionsRequest; apiFactoryActions; factoryRunActionsRequest; factoryActionRun; apiFactoryRunActions; commandRepository; apiFactoryCommand | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a081deaca2cc"></a>

## [internal/web/api/factory_output.go](../../../../../internal/web/api/factory_output.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 22–349; privateFactoryOutputRoute; checkFactoryOutputHeaders; factoryViewerIdentity; factoryViewerAccount; factoryViewerCurrent; factoryOutputHandshake; readFactoryOutputHandshake; validFactoryOutputHandshake; factoryStatusFrame; factoryOutputFrame; factoryClosedFrame; factoryOutputStatus; writeFactoryFrame; refuseFactoryOutput; factoryOutputStream; factoryOutputAttach; factoryRejectViewerInput; pumpFactoryOutput; pumpFactorySlice; sameFactoryStatus; statusWithoutExit; closeFactorySlice; factoryOutputHeartbeat | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: privateFactoryOutputRoute; 23 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-91eebb3b28d8"></a>

## [internal/web/api/factory_output_test.go](../../../../../internal/web/api/factory_output_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–132; file scaffold; scriptedFactoryOutput; RoundTrip; factoryOutputTestPair; readFactoryTestFrame; factoryOutputTestBytes | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test imports and module shell for factory output stream tests.; 6 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 133–217; TestFactoryOutputFrameFitsProxyWireLimit; TestPumpFactorySliceDrainsTerminalOutputBeforeEOF; TestPumpFactorySliceEndsEmptyTerminalRun | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Assert the maximum S06 output frame fits the proxy wire budget.; Assert S06 streams all terminal factory output slices before emitting EOF.; Assert S06 emits status and EOF for a terminal run with no output. — The assertion exercises factory output frame/pump behavior through the viewer protocol; it checks neither human terminal attachment nor F07 assignment/lifecycle authority. |

<a id="coverage-13cbc1878eb4"></a>

## [internal/web/api/factory_policy.go](../../../../../internal/web/api/factory_policy.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–139; file scaffold; factoryPolicyRequest; apiFactoryPolicy; factoryCapacityRequest; apiFactoryCapacity; factoryOperatorGrantRequest; apiFactoryOperatorGrant | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e83d1593855f"></a>

## [internal/web/api/factory_readiness.go](../../../../../internal/web/api/factory_readiness.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 19–21, 23, 25–30, 37–42, 61–79; whole file; ServiceReadinessSource; (declaration group); NewServiceReadinessSource; ServiceReadinessSource.ObserveNativeRevision; ServiceReadinessSource.ListRepositoryIssues | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 16, 22, 31–33, 43–51; ServiceReadinessSource.Evidence; control.AcceptanceSource; AcceptanceSnapshotSource; ServiceReadinessSource.ReadAcceptanceEvidence | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Acceptance evidence reader contract; declarations/fields: `ServiceReadinessSource.Evidence`; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 17, 24, 34–36, 52–60; ServiceReadinessSource.Dispatch; control.DispatchReads; DispatchSnapshotSource; ServiceReadinessSource.ReadDispatchInputs | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Exact dispatch input reader contract; declarations/fields: `ServiceReadinessSource.Dispatch`; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 18; ServiceReadinessSource.Observer | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Shared authoritative native observer; declarations/fields: `ServiceReadinessSource.Observer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-619247c30ebc"></a>
<a id="internalwebapifactory_settingsgo-1"></a>

## [internal/web/api/factory_settings.go](../../../../../internal/web/api/factory_settings.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 17–74, 85–103; factoryPrincipal; factoryCommandError; factoryRepository; factoryOwner; settingsCommandID; actorRefRequest; resolve | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Current declaration duty: factoryPrincipal; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 75–84; factoryOperator | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: factoryOperator — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0736ad08057a"></a>

## [internal/web/api/factory_sponsorship.go](../../../../../internal/web/api/factory_sponsorship.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–182; file scaffold; sponsorshipView; factorySponsorshipRequest; apiFactorySponsorship; checkSponsorshipBroker; factoryEnvironmentGrantRequest; apiFactoryEnvironmentGrant | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6104e8c3c023"></a>

## [internal/web/api/factory_status.go](../../../../../internal/web/api/factory_status.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–140; file scaffold; factoryQueueView; factoryStatusView; factoryPreparationView; apiFactoryStatus; factoryPreparationStatus | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-e1f392b97869"></a>

## [internal/web/api/factory_views.go](../../../../../internal/web/api/factory_views.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147; file scaffold; factoryRunOutputExcerpt; factoryRunRecord; factoryRunRecordDTO; factoryRunBinding; factoryRunBindingDTO; factoryRunLive; factoryHostTerminal; factoryRunLiveDTO; factoryRunStatus; apiFactoryRun | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Wires the factory status viewer adapter and its source imports.; 11 named units assigned here; remaining selectors preserve each duty — Independent current-body review: factory_views.go:23-46 defines/maps viewer DTO fields; apiFactoryRun composes them at115-147 after the called authorization handler admits the read.; GET /api/factory/runs/{runID} in extension_native.go calls apiFactoryRun; it delegates current write-authority admission to factoryActionRun, then reads FactoryRunView and Host.FactoryInspect and renders the status projection. This is S06 presentation; Called authorization handlers, storage and host components retain their defining admission and lifecycle duties. |

<a id="coverage-104efcde52d9"></a>

## [internal/web/api/factory_views_test.go](../../../../../internal/web/api/factory_views_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–126; file scaffold; TestFactoryHostTerminalMapping; TestFactoryRunLiveExcerptIsBounded; TestFactoryRunBindingRendersDecimalIDs; TestFactoryOutputHandshakeBindsRunRepositoryAndGeneration; frameKeys; TestFactoryFramesUseFixedKeySets; TestSameFactoryStatusComparesExitByValue | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Test module shell for factory viewer projection and output protocol assertions.; Assert factory viewer status, handshake binding, or output frame presentation behavior. — Imports and package shell support only the factory viewer/status/handshake/frame tests in this file.; The named assertion calls factory view/handshake/frame helpers and checks the S06 factory viewer contract; it does not test assignment or run lifecycle mutation. |

<a id="coverage-98a92690bd62"></a>

## [internal/web/api/identity.go](../../../../../internal/web/api/identity.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 18–20, 27–28, 73–101, 135–141; whole file; IdentityClient; API.apiIdentityStartEnrollment; API.apiIdentityEnrollment; API.apiIdentityCancelEnrollment | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 16–17, 21–23, 62–72, 102–112, 149–155; IdentityClient.Connections, IdentityClient.Available; IdentityClient.Grants, IdentityClient.CreateGrant, IdentityClient.RevokeGrant; API.apiIdentityConnections, API.apiIdentityGrants, API.apiIdentityRevokeGrant | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Owned connections and delegated availability API contract; declarations/fields: `IdentityClient.Connections`, `IdentityClient.Available`; Delegation grant API contract; declarations/fields: `IdentityClient.Grants`, `IdentityClient.CreateGrant`, `IdentityClient.RevokeGrant`; Owned connection/delegation metadata read or grant revocation; declarations/fields: `API.apiIdentityConnections`, `API.apiIdentityGrants`, `API.apiIdentityRevokeGrant` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 24, 113–123; IdentityClient.Leases; API.apiIdentityLeases | [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) | retained | Lease metadata API contract; declarations/fields: `IdentityClient.Leases`; Own lease/fence metadata inventory; declarations/fields: `API.apiIdentityLeases` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 25–26, 142–148, 156–161; IdentityClient.EndLease, IdentityClient.Revoke; API.apiIdentityRevoke, API.apiIdentityEndLease | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Explicit connection/lease retirement API contract; declarations/fields: `IdentityClient.EndLease`, `IdentityClient.Revoke`; Explicit credential/lease retirement; declarations/fields: `API.apiIdentityRevoke`, `API.apiIdentityEndLease` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 29–61, 124–134; identityError, API.identityAdmission, API.identityResult, API.identityMutation | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current native actor admission/error envelope shared by identity operations; declarations/fields: `identityError`, `API.identityAdmission`, `API.identityResult`, `API.identityMutation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-02658d9b3176"></a>

## [internal/web/api/identity_grants.go](../../../../../internal/web/api/identity_grants.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; file scaffold; identityProjectMember; identityNamedMember; apiIdentityCreateGrant; apiIdentityAvailable | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-829992f02825"></a>

## [internal/web/api/identity_launch.go](../../../../../internal/web/api/identity_launch.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–55; file scaffold; apiIdentityLaunch; codexLaunchInput | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; Current declaration duty: apiIdentityLaunch; Current declaration duty: codexLaunchInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 56–66; identityLaunchInput | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: identityLaunchInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a21fcee1ae10"></a>

## [internal/web/api/issue_acceptance_evidence.go](../../../../../internal/web/api/issue_acceptance_evidence.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–105; file scaffold; AcceptanceSnapshotSource; ReadAcceptanceEvidence; mapAcceptanceSnapshotError | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-aa4c2fa22d23"></a>
<a id="internalwebapiissue_acceptancesgo-1"></a>

## [internal/web/api/issue_acceptances.go](../../../../../internal/web/api/issue_acceptances.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 16–240; factoryIssue; acceptanceSourceRequest; acceptancePrerequisiteRequest; issueAcceptanceRequest; apiIssueAcceptances; acceptanceDecision; acceptanceSources; issueWithdrawalRequest; apiIssueWithdrawal; acceptanceDecisionError; acceptanceRefusalError; acceptanceRefusalMessage | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Current declaration duty: factoryIssue; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-906ea1eaceef"></a>

## [internal/web/api/issue_acceptances_test.go](../../../../../internal/web/api/issue_acceptances_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–157; file scaffold; stubAcceptanceReader; ReadNativeRevision; ReadSnapshot; acceptanceWireSnapshot; idleAcceptanceObservation; TestAcceptanceSnapshotSourceBracketsSelection; TestAcceptanceSnapshotSourceSkipsUnselectedComments; TestAcceptanceSnapshotSourceMapsBracketFailures | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9276e67551ee"></a>

## [internal/web/api/lifecycle.go](../../../../../internal/web/api/lifecycle.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–207, 229–271; whole file; API.loadLifecycleEnvironment; decodeLifecycleRequest; API.authorizeLifecycleOperator; API.checkLifecycleSession; API.acquireLifecycleMutation; API.handleLifecycleMutation; API.apiLifecycle; lifecycleStopControlView; lifecycleStopView; API.apiLifecycleStop; lifecycleStartView; API.apiLifecycleStart | [P05](../../slices/projects.md#p05-project-startstop) | retained | Start invokes verified/quiescent Project-stop clearing through the coordinator; it does not clear unrelated dispatch causes. `8d9485af` closes this P05 source subcut. The lifecycle operator mapping does not establish P09's current native Project-admin privilege; that approval question remains open. |
| 208–228; API.lifecycleStopHold | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Project Stop synchronizes admission hold without owning hold policy; declarations/fields: `API.lifecycleStopHold` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-aba296f85b8d"></a>

## [internal/web/api/operator.go](../../../../../internal/web/api/operator.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–39; file scaffold; errOperatorRequired; operatorAuthorization; authorizeOperator | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-01a96515da16"></a>

## [internal/web/api/preparation_decisions.go](../../../../../internal/web/api/preparation_decisions.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12, 57–67, 80–100, 149–171; whole file; preparationActionRequest; API.apiPreparationActions; API.apiPreparationActionApprove | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Approval currently checks the lifecycle operator mapping and session, then admits the decision; it has no current native Project-admin privilege attestation. Q2 remains open: repository/project ownership alone does not prove the required Project privilege. |
| 13–56; preparationAcceptanceRequest, API.apiPreparationAcceptances | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Exact requirement acceptance route/input; declarations/fields: `preparationAcceptanceRequest`, `API.apiPreparationAcceptances` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 68–79, 125–148; preparationInspectView, API.apiPreparationActionInspect | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Existing preparation/decision-reference observation; declarations/fields: `preparationInspectView`, `API.apiPreparationActionInspect` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 101–124; API.apiPreparationActionHold | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Preparation admission maintenance hold command; declarations/fields: `API.apiPreparationActionHold` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-97b2e061ed54"></a>

## [internal/web/api/project_profiles.go](../../../../../internal/web/api/project_profiles.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; apiProjectProfiles | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current scaffold duty: file scaffold; Current declaration duty: apiProjectProfiles — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a38203d6ba5a"></a>

## [internal/web/api/provisioning.go](../../../../../internal/web/api/provisioning.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–26, 33–69; whole file; (declaration group); operationContext; provisioningConfirmed; API.reconcileProvisioning | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 27–32; stopOperationTimeout, startOperationTimeout | [P05](../../slices/projects.md#p05-project-startstop) | retained | Native Stop/Start bounded operation deadlines; declarations/fields: `stopOperationTimeout`, `startOperationTimeout` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-2fd68a992172"></a>

## [internal/web/api/repositories.go](../../../../../internal/web/api/repositories.go)

Current declarations split at native authority/search admission, Project reservation association, and readiness field boundaries; callers and store/native adapter behavior inspected.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 34–68, 77–82, 86–133, 139–147; package/import scaffold and native API dependencies; repositoryChoice native identity fields ID, Owner, Name; isValidSearchTerm and native repository query parsing/field allowlist; collectNativeRepositoryChoices native response identity and owner checks; append completed native repository choices; repository choice error mapping: native/provider branch; handleRepositorySessionFailure native context freshness and search timeout; apiRepositories human-owner discovery note, verified extension authority gate and handler handoff; apiNativeRepositories query admission and repository search slot; apiNativeRepositories owned-repository native search and page bounds; apiNativeRepositories verified context recheck before publishing results; response Items and NextCursor native search envelope | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Wires this endpoint to verified extension authority, native Forgejo repository search and the Project store lookup.; 12 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 24–28, 31–33, 69–76, 83–85, 134–138; repositoryChoice CanCreate and Project reservation fields; repositoryProject ID; errStoreReservation storage lookup failure sentinel; collectNativeRepositoryChoices Project reservation lookup and Create eligibility; repositoryProject.ID projection from store association; repository choice error mapping: reservation failure branch; apiNativeRepositories Project association join and reservation error dispatch | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Represents whether the native repository has no stored Project association and, if reserved, the associated Project identity.; 7 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 29–30, 76; repositoryProject Provisioned; repositoryProject.Provisioned projection from project.Ready | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Carries the saved Project readiness state; it is not the repository reservation/Create eligibility bit.; Projects the current saved readiness flag for the already-associated Project into its distinct readiness field. — current source internal/web/api/repositories.go:29-30; project.Ready is copied into response field at line 76; current source internal/web/api/repositories.go:76; project.Ready is distinct from CanCreate and reservation existence |

<a id="coverage-1b43436f7ee8"></a>
<a id="internalwebapispacesgo-1"></a>

## [internal/web/api/spaces.go](../../../../../internal/web/api/spaces.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–176; file scaffold; SpaceView; SpacesView; SpacesActor; appendSpaceRow; inspectSpaces; verifySpacesSession; apiSpaces; parseSpacesCursor | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-334651498050"></a>

## [internal/web/api/spaces_authority.go](../../../../../internal/web/api/spaces_authority.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; file scaffold; spaceAuthorityView; resolveSpaceAuthority; inspectSpaceAuthority; spaceControlView | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 52–78; inspectSpaceControl | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: inspectSpaceControl — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dc3c9248f0e9"></a>

## [internal/web/api/spaces_inspection.go](../../../../../internal/web/api/spaces_inspection.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 27–39; file scaffold; inspectSpaceNative | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current scaffold duty: file scaffold; Current declaration duty: inspectSpaceNative — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–26, 85–170; SpaceFactoryRun; inspectSpaceRow; inspectSpaceFactoryRuns | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: SpaceFactoryRun; Current declaration duty: inspectSpaceRow; Current declaration duty: inspectSpaceFactoryRuns — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 40–66; inspectSpaceTerminals | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: inspectSpaceTerminals — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 67–84; inspectSpaceTailnet | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current declaration duty: inspectSpaceTailnet — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4626544a8429"></a>

## [internal/web/api/spaces_inventory.go](../../../../../internal/web/api/spaces_inventory.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19, 116–119, 141–143; file scaffold; spacesRunCap; runsKnown | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Current scaffold duty: file scaffold; Current declaration duty: spacesRunCap; Current declaration duty: runsKnown — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 20–115, 120–140; spacesInventory; spacesAssociationPage; spacesRunPage; loadSpacesInventory; mergeSpacesRuns; spacesRunUsage; newSpacesRunUsage; factoryComplete | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: spacesInventory; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-69a9d21426ed"></a>

## [internal/web/api/tailnet.go](../../../../../internal/web/api/tailnet.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; whole file; tailnetQuery; API.tailnetSession; tailnetError; API.apiTailnetSettings; API.apiTailnetHost | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 86–113; API.apiTailnetEnrollment | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Operator Project enrollment policy mutation; declarations/fields: `API.apiTailnetEnrollment` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 114–246; API.apiTailnetOptions, API.authorizeProjectTailnet, parseProjectTailnetMutation, API.admitProjectTailnetDispatch, API.confirmProjectTailnetDispatch, API.admitProjectTailnetRequest, API.apiProjectTailnet | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Own Project original network selection and authorized observation; declarations/fields: `API.apiTailnetOptions`, `API.authorizeProjectTailnet`, `parseProjectTailnetMutation`, `API.admitProjectTailnetDispatch`, `API.confirmProjectTailnetDispatch`, `API.admitProjectTailnetRequest`, `API.apiProjectTailnet` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-9a59f58be692"></a>

## [internal/web/api/terminal_registry.go](../../../../../internal/web/api/terminal_registry.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23, 50–53, 99–212; whole file; validTerminalGeometry; validTerminalOrigin; validSecFetchHeaders; checkTerminalRequestHeaders; terminalHandshake; readTerminalHandshake; validHandshakeDimensions; validHandshakeAction; validHandshakeRepository; refuseTerminal; pumpTerminalInput; pumpNativeToExtension | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 13 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 24–49, 54–98, 213–241; TerminalID, newTerminalID, TerminalCreationScope, TerminalView, terminalDTO, reservedTerminal, terminalMetadata, terminalSessionMutation, parseTerminalSessionAction, validTerminalSessionMutation, API.CloseTerminals, API.dropTerminalPeer | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Human terminal identity/metadata and view-bound lifecycle tracking; declarations/fields: `TerminalID`, `newTerminalID`, `TerminalCreationScope`, `TerminalView`, `terminalDTO`, `reservedTerminal`, `terminalMetadata`, `terminalSessionMutation`, `parseTerminalSessionAction`, `validTerminalSessionMutation`, `API.CloseTerminals`, `API.dropTerminalPeer`, `API.unregisterTerminalPeer`, `peerIDInUse` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

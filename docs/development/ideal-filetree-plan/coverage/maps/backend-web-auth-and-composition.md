# Backend web auth and composition

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-11ada2746090"></a>

## [internal/web/auth/auth.go](../../../../../internal/web/auth/auth.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; file scaffold; PositiveID | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: PositiveID — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-808fc3f35b57"></a>

## [internal/web/auth/development_key.go](../../../../../internal/web/auth/development_key.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; file scaffold; NormalizeDevelopmentKey | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: NormalizeDevelopmentKey — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-063f3b84de74"></a>

## [internal/web/auth/errors.go](../../../../../internal/web/auth/errors.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; file scaffold; ProviderError | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: ProviderError — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-46db262256e3"></a>

## [internal/web/auth/errors_test.go](../../../../../internal/web/auth/errors_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31; file scaffold; TestProviderErrorMapsNativeInvisibleRepositoryToNotFound; TestProviderErrorMapsChangedNativeSessionToConflict | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestProviderErrorMapsNativeInvisibleRepositoryToNotFound; Current declaration duty: TestProviderErrorMapsChangedNativeSessionToConflict — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-03f190e9b260"></a>

## [internal/web/auth/extension.go](../../../../../internal/web/auth/extension.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; ExtensionAuthority | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: ExtensionAuthority — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 44–60; ExtensionContribution | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current declaration duty: ExtensionContribution — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-163b612535f1"></a>

## [internal/web/auth/extension_service.go](../../../../../internal/web/auth/extension_service.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96, 124–157, 163–205, 240–281; whole file; extensionRequest; sessionUserView; Service.ExtensionHandler; Service.serveExtension; Service.serveExtensionSession; Service.serveExtensionResource; Service.serveExtensionAccount; allowExtensionNoQuery; validExtensionPath; allowExtensionMethod; Service.extensionRequest; extensionIdentity; Service.extensionUser; extensionSessionGenerationMatches; writeExtensionSession; revalidateExtensionAuthority; Service.extensionMutation; validExtensionMutationOrigin | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 19 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 97–105; Service.serveExtensionPreferences | [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) | retained | Local preference route; current native authority supplied by G01; declarations/fields: `Service.serveExtensionPreferences` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 106–123, 158–162, 206–239; Service.serveExtensionDevelopmentKeys, Service.serveExtensionDevelopmentKeyRemoval, extensionDevelopmentKeyPath, Service.apiExtensionForgejoKeys | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Own development public key review/register/remove route; declarations/fields: `Service.serveExtensionDevelopmentKeys`, `Service.serveExtensionDevelopmentKeyRemoval`, `extensionDevelopmentKeyPath`, `Service.apiExtensionForgejoKeys` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-44f117739a51"></a>

## [internal/web/auth/extension_test.go](../../../../../internal/web/auth/extension_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; file scaffold; TestExtensionContribution; TestExtensionAuthorityFailsWithoutLiveHost; TestExtensionServiceRejectsUnverifiedMutation | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4dab55ad2f94"></a>

## [internal/web/auth/forgejo_keys.go](../../../../../internal/web/auth/forgejo_keys.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; file scaffold; parseForgejoKeysPage; profileKeyView | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: parseForgejoKeysPage; Current declaration duty: profileKeyView — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6ca8d39f3990"></a>

## [internal/web/auth/http.go](../../../../../internal/web/auth/http.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–22, 38–78; file scaffold; APIBodyLimit; apiError; JSONError; AllowAPIMethod; DecodeAPIObject | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 23–37; JSONResponse | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: JSONResponse — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c22615f9253a"></a>

## [internal/web/auth/postgres_fixture_test.go](../../../../../internal/web/auth/postgres_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; file scaffold; postgresFixture | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: postgresFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3b216923983e"></a>

## [internal/web/auth/service.go](../../../../../internal/web/auth/service.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–26; file scaffold; Service; New; ValidRepositoryPart | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-53e51f45434b"></a>

## [internal/web/auth/session.go](../../../../../internal/web/auth/session.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 40–101; whole file; developmentKeyView; Service.apiKeys; Service.apiRemoveDevelopmentKey | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 11–39; Service.preferences | [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) | retained | Read/update only current actor’s Soda-local display name; declarations/fields: `Service.preferences` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-9bff23d4b1a8"></a>

## [internal/web/avatars.go](../../../../../internal/web/avatars.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–141; file scaffold; avatarPrefix; avatarHandler; parseAvatarSize; parseAvatarQuery; admitAvatarRoute; admitAvatarQuery; parseAvatarRequest; ServeHTTP; avatarError | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3e4fd2b0206c"></a>

## [internal/web/avatars_browser_test.go](../../../../../internal/web/avatars_browser_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–40; file scaffold; TestAvatarBrowserRendering | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestAvatarBrowserRendering — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a25a9340514e"></a>

## [internal/web/avatars_test.go](../../../../../internal/web/avatars_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–149; file scaffold; avatarURL; TestAvatarPublicRoute; TestAvatarRejectsInvalidInputBeforeRendering; TestAvatarPathBoundaries; TestAvatarFailureIsBoundedAndUncached; TestAvatarNormalizesProviderHash | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0ae8ee7295cc"></a>

## [internal/web/browser_join_test.go](../../../../../internal/web/browser_join_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; file scaffold; TestBrowserOnlyJoinAndExplicitSavedSSHChoice | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestBrowserOnlyJoinAndExplicitSavedSSHChoice — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6c2fe25b2fdc"></a>

## [internal/web/environment_authority_test.go](../../../../../internal/web/environment_authority_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; file scaffold | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 16–126; TestEnvironmentAuthorityUsesCurrentNativeIdentity | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current declaration duty: TestEnvironmentAuthorityUsesCurrentNativeIdentity — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fe71141392e8"></a>

## [internal/web/environment_os_test.go](../../../../../internal/web/environment_os_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–14; file scaffold | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 15–104; TestOSObservationUsesExistingReadAuthorityAndGenerationWins | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestOSObservationUsesExistingReadAuthorityAndGenerationWins — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-28b27935ae98"></a>

## [internal/web/environment_read_publication_test.go](../../../../../internal/web/environment_read_publication_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–182; file scaffold; TestEnvironmentReadPublicationRechecksNativeAuthority | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestEnvironmentReadPublicationRechecksNativeAuthority — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-583eeb0c64e0"></a>

## [internal/web/environments_api_test.go](../../../../../internal/web/environments_api_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; file scaffold | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 17–138; TestJSONEnvironmentReservationAndExplicitJoins; TestIncompleteEnvironmentStillInspected | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestJSONEnvironmentReservationAndExplicitJoins; Current declaration duty: TestIncompleteEnvironmentStillInspected — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9a525e607702"></a>

## [internal/web/execution_access_test.go](../../../../../internal/web/execution_access_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–61; file scaffold; TestProjectExecutionRequiresCurrentNativeWritePermission | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestProjectExecutionRequiresCurrentNativeWritePermission — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0c778f5bceef"></a>

## [internal/web/extension.go](../../../../../internal/web/extension.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–41; file scaffold; Extension; forgejoUsernamePolicy | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: Extension; Current declaration duty: forgejoUsernamePolicy — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 42; ExtensionHandler | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: ExtensionHandler — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-da17df2e7719"></a>

## [internal/web/extension_terminal_test.go](../../../../../internal/web/extension_terminal_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–227; file scaffold; nativeTerminalGeneration; nativeTerminalContext; nativeTerminalHeaders; nativeTerminalFixtureDir; nativeTerminalCallback; nativeTerminalProxyFixture; nativeTerminalRequest; TestNativeTerminalReserveCreateAttachAndAuthority | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ee12d700b73f"></a>

## [internal/web/extension_test.go](../../../../../internal/web/extension_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–49; file scaffold; TestExtensionUsernamePolicy | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestExtensionUsernamePolicy — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1b87e76a2f2a"></a>

## [internal/web/factory_checks_view_test.go](../../../../../internal/web/factory_checks_view_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–94; file scaffold; TestFactoryIssueChecksView | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestFactoryIssueChecksView — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7ecd563f96d0"></a>

## [internal/web/factory_intake_route_test.go](../../../../../internal/web/factory_intake_route_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123; file scaffold; intakeRouteSource; ReadAcceptanceEvidence; intakeRouteServer; TestFactoryIntakeRouteServesVerifiedDeliveries; TestFactoryIntakeRouteRefusesUnauthenticated; TestFactoryIntakeRouteUnavailableWithoutSecret | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8097e371e5c6"></a>

## [internal/web/factory_lifecycle_test.go](../../../../../internal/web/factory_lifecycle_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–260; whole file; lifecycleWebFixture; lifecycleWebRun; lifecycleWebGrants; lifecycleWebPreparations; lifecycleAction; TestFactoryPauseAndResumeJourney; TestFactoryActionsRequireCurrentCodeWrite; TestFactoryRunStopAndRetry; TestFactoryTakeoverRequiresMembershipAndSettledRun | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 10 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 261–321; TestLifecycleStopAndStartCoordinate | [P05](../../slices/projects.md#p05-project-startstop) | retained | Lifecycle stop/start coordinates withdrawal and run reconciliation; uncertain start retains the Project-stop cause. |
| 323–343; TestLifecycleStartClearsProjectStopAfterQuiescence | [P05](../../slices/projects.md#p05-project-startstop) | retained | Regression verifies only a confirmed, quiescent Project start clears the Project-stop cause. `8d9485af` source/owned-PG scope; this fixture does not qualify native Project-admin approval or native provider behavior. |
| 345–366; TestSpacesShowsFactoryControl | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Authorized inventory current factory control projection; declarations/fields: `TestSpacesShowsFactoryControl` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-73e07b3f1e4d"></a>

## [internal/web/factory_output_fixture_test.go](../../../../../internal/web/factory_output_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–113; file scaffold; factoryOutputScript; factoryOutputProxyFixture; dialFactoryOutput; readFactoryFrame; factoryOutputHandshakeFor | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cccc6ad5b57b"></a>

## [internal/web/factory_output_stream_test.go](../../../../../internal/web/factory_output_stream_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–160; file scaffold; TestFactoryOutputStreamDeliversStatusOutputAndEOF; TestFactoryOutputStreamRejectsInput; TestFactoryOutputStreamRefusesStaleHandshake; TestFactoryOutputStreamClosesOnUnknownAndStale; TestFactoryOutputStreamRequiresWrite | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Test module shell for the factory output viewer stream protocol.; Assert the factory viewer output stream handshake, read-only frames, bounded output, or close behavior. — Imports and package shell support only factory output stream integration assertions.; The named test calls the factory output proxy and scripted host fixture and asserts S06 stream behavior; F07 assignment/lifecycle is exercised only as the caller authority source, not the tested duty. |

<a id="coverage-a4f7afdd4d39"></a>

## [internal/web/factory_settings_test.go](../../../../../internal/web/factory_settings_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–19; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 20–176; factorySettingsProject; factorySettingsServer; factoryEncryptedServer; factoryPolicyBody; decodeBody; assertNoSecrets; TestFactoryStatusShowsMissingAuthority; TestFactoryPolicyJourney; TestFactoryPolicyPauseWithdrawsDispatch | [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) | retained | Current declaration duty: factorySettingsProject; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 177–197; TestFactoryCapacityIsOperatorOnly | [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) | retained | Current declaration duty: TestFactoryCapacityIsOperatorOnly — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 198–213; TestFactoryEnvironmentGrantIsOwnerOnly | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current declaration duty: TestFactoryEnvironmentGrantIsOwnerOnly — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 214–287; TestFactorySponsorshipNeedsConnectionOwner | [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) | retained | Current declaration duty: TestFactorySponsorshipNeedsConnectionOwner — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 288–319; TestPreparationAcceptanceJourney | [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) | retained | Current declaration duty: TestPreparationAcceptanceJourney — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 320–363; TestPreparationApprovalJourney | [P09](../../slices/projects.md#p09-privileged-preparation-approval) | retained | Current declaration duty: TestPreparationApprovalJourney — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-900b9381263e"></a>
<a id="internalwebfactory_views_testgo-1"></a>

## [internal/web/factory_views_test.go](../../../../../internal/web/factory_views_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17; file scaffold | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–193; factoryViewsFixture; factoryViewsRun; factoryViewsHostResponse; readFactoryRun; TestFactoryRunStatusPendingLiveAndExcerpt; TestFactoryRunStatusBindingAndRefusals; TestSpacesShowsFactoryRuns | [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) | retained | Current declaration duty: factoryViewsFixture; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-749730919373"></a>

## [internal/web/forgejo_keys_test.go](../../../../../internal/web/forgejo_keys_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–71; file scaffold; TestOwnForgejoKeyReviewDoesNotSaveOrInstallAndIsBounded | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestOwnForgejoKeyReviewDoesNotSaveOrInstallAndIsBounded — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 72–86; TestForgejoKeyPickerRejectsGlobalQueries | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestForgejoKeyPickerRejectsGlobalQueries — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2b1235840ff6"></a>

## [internal/web/identity_native_test.go](../../../../../internal/web/identity_native_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35, 46–91; whole file; nativeIdentityFake; nativeIdentityFake.Connections; nativeIdentityFake.StartEnrollment; TestNativeIdentityActorAndEnrollmentTrust | [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 36–40, 92–154; nativeIdentityFake.CreateGrant, TestNativeIdentityGrantRequiresConfirmationsMembersAndWrite | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Grant confirmation, member/permission and recipient authority assertions; declarations/fields: `nativeIdentityFake.CreateGrant`, `TestNativeIdentityGrantRequiresConfirmationsMembersAndWrite` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 41–45; nativeIdentityFake.EndLease | [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) | retained | Explicit lease retirement fixture; declarations/fields: `nativeIdentityFake.EndLease` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 155–201; TestNativeIdentityLaunchCompensatesGenerationChange | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Native provider execution compensates generation loss assertions; declarations/fields: `TestNativeIdentityLaunchCompensatesGenerationChange` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-d020fb36e63b"></a>

## [internal/web/identity_stub_test.go](../../../../../internal/web/identity_stub_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–100; file scaffold; errStubIdentity; stubIdentityClient; Connections; Grants; Available; StartEnrollment; Enrollment; CancelEnrollment; CreateGrant; RevokeGrant; Leases; EndLease; Revoke | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-55c0101d2dbd"></a>

## [internal/web/issue_acceptances_test.go](../../../../../internal/web/issue_acceptances_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–181; file scaffold; issueWebSource; ReadAcceptanceEvidence; issueWebEvidence; issueWebFixture; issueAcceptanceBody; TestIssueAcceptanceJourney; TestIssueAcceptanceRefusals | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-df76478dc399"></a>

## [internal/web/join_boundary_test.go](../../../../../internal/web/join_boundary_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15; file scaffold | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 16–123; TestConcurrentNewJoinsAndResultPersistenceFailure; TestConnectionRemainsOwnMembershipOnlyDuringProviderFailure | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestConcurrentNewJoinsAndResultPersistenceFailure; Current declaration duty: TestConnectionRemainsOwnMembershipOnlyDuringProviderFailure — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d6aac70bcf52"></a>

## [internal/web/lifecycle_access_keys_test.go](../../../../../internal/web/lifecycle_access_keys_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–73, 93–206; whole file; managementWebFixture; TestSavedKeyRemovalIsOwnOnlyAndNeverNativeRevocation; TestOperatorCannotManageAnotherAccountsKeysAndGenerationStillApplies; TestAccessKeyPreviewApplyAndLastKeyConfirmation | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 74–92; TestLifecycleAuthorizationAndExplicitStop | [P05](../../slices/projects.md#p05-project-startstop) | retained | Project Start/Stop admission/confirmation assertions; declarations/fields: `TestLifecycleAuthorizationAndExplicitStop` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-90d15aaf6b3d"></a>

## [internal/web/mutation_admission_test.go](../../../../../internal/web/mutation_admission_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–22, 212–224, 226–262; file scaffold; mutationAdmissionBody; Read; TestMutationAdmissionAfterDecode | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Post-decode mutation admission rechecks current authority before calling a helper; the body wrapper drives this regression. |
| 23–145; TestMutationAdmissionAfterNativeCallbackIO | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current native callback I/O admission assertions; declarations/fields: `TestMutationAdmissionAfterNativeCallbackIO` — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 147–210; TestStartAdmissionRefusesOverlappingStopAndKeepsPeer | [P05](../../slices/projects.md#p05-project-startstop) | retained | Concurrent Start/Stop admission refuses the second mutation while preserving the existing human terminal peer. This fixture exercises lifecycle serialization, not native Project-admin approval or provider qualification. |

<a id="coverage-fcf0c3fe234a"></a>

## [internal/web/postgres_fixture_test.go](../../../../../internal/web/postgres_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24; file scaffold; postgresFixture | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: postgresFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f526f87d1223"></a>

## [internal/web/preparation_test.go](../../../../../internal/web/preparation_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–62; whole file; preparationProject; preparationFixture; seedPreparationRecord | [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 63–127; TestPreparationReadShowsHoldAndRoleReadiness, TestPreparationHoldRequiresAdministrator, TestPreparationHoldAndReleaseSyncMarker | [P12](../../slices/projects.md#p12-maintenance-holds) | retained | Persistent/native admission hold administrator and synchronization assertions; declarations/fields: `TestPreparationReadShowsHoldAndRoleReadiness`, `TestPreparationHoldRequiresAdministrator`, `TestPreparationHoldAndReleaseSyncMarker` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 128–153; TestSpacesRowSummarizesPreparation | [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) | retained | Authorized inventory preparation summary assertions; declarations/fields: `TestSpacesRowSummarizesPreparation` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-b3fafca16620"></a>

## [internal/web/project_profiles_test.go](../../../../../internal/web/project_profiles_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; file scaffold; testCreationProfile; profileTestResponse | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: testCreationProfile; Current declaration duty: profileTestResponse — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 26–99; TestProfileReadKeepsOwnerAndRequestGuards; TestProfilePreflightNeverReservesOnUnavailableOrAuthorityChange | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestProfileReadKeepsOwnerAndRequestGuards; Current declaration duty: TestProfilePreflightNeverReservesOnUnavailableOrAuthorityChange — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fd401fc7afc0"></a>

## [internal/web/provisioning_lifetime_test.go](../../../../../internal/web/provisioning_lifetime_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18, 29–37, 88–143; file scaffold; jsonNativeResponse; inspectStub; roundTrip; errNativeGone; errTestNativeGone; Error; lifecycleInspectBody; decodeDetail | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 19–28, 38–87, 144–304; seedUnreadyProject; TestCreateCancellationStillRecordsResult; TestInterruptedCreationReconcilesFromHostEvidence | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: seedUnreadyProject; Current declaration duty: TestCreateCancellationStillRecordsResult; Current declaration duty: TestInterruptedCreationReconcilesFromHostEvidence — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-85f36d6a66c5"></a>

## [internal/web/repositories_test.go](../../../../../internal/web/repositories_test.go)

Current test cases split by native actor/session/search admission, Project reservation eligibility, and the saved readiness assertion in the mixed table test.

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13, 28–68, 75–159; test package/imports and API test harness imports; native owned-repository callback fixture; request and native actor/session fixture; native authority, ownership and provider failure assertions; table-test closure; TestRepositoryPickerNativeGenerationChangeWinsPublication; TestRepositoryPickerNativePageBoundary; TestRepositoryPickerQueryAndAdmission | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Defines the authenticated native repository endpoint test source and imports the callback/request fixture types used by the following assertions.; 8 named units assigned here; remaining selectors preserve each duty — Current named units/source consumers; retained normalized source evidence records each selector |
| 14–21, 26–27, 69–74; TestRepositoryPickerAuthorityAndReservations Project association fixture setup; repository Create eligibility and Project reservation response assertions | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Creates associated Project rows for ready/incomplete cases and establishes the repository reservation state used to distinguish existing from creatable repositories.; Asserts an unassociated native repository is can_create with project null, while an existing association is non-creatable and carries the stored Project ID. — current source internal/web/repositories_test.go:14-21,26-27; Store.CreateProject fixture; current source internal/web/repositories_test.go:69-74; new versus existing reservation response checks |
| 22–25, 72; TestRepositoryPickerAuthorityAndReservations ready-state fixture marking; associated Project provisioned response assertion | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Marks only the ready case with Store.MarkReady so the response readiness projection is independently observable from reservation existence.; Checks that the response provisioned value tracks the separately marked ready state, rather than reservation presence. — current source internal/web/repositories_test.go:22-25; MarkReady fixture producer; current source internal/web/repositories_test.go:72; formatted expected provisioned value is kind == ready |

<a id="coverage-87051c671e04"></a>

## [internal/web/repository_access_test.go](../../../../../internal/web/repository_access_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–88; file scaffold; TestRepositoryDenialBlocksDiscoveryDirectReadsAndNewAccounts | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRepositoryDenialBlocksDiscoveryDirectReadsAndNewAccounts — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 89–174; TestRepositoryLookupAndJoinRecheckNativeAccess | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestRepositoryLookupAndJoinRecheckNativeAccess — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7022145dc344"></a>

## [internal/web/repository_settings_test.go](../../../../../internal/web/repository_settings_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–13; file scaffold | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 14–53; TestRepositorySettingsProtectedContext | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestRepositorySettingsProtectedContext — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3e1475a5bd57"></a>

## [internal/web/retired_frontend_test.go](../../../../../internal/web/retired_frontend_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; file scaffold | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–29; TestProductAPIsRequireNativeExtensionRoute | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestProductAPIsRequireNativeExtensionRoute — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-158eb27e417c"></a>

## [internal/web/server.go](../../../../../internal/web/server.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–38, 40–44, 68–69, 73–78, 81–100, 159–163, 172–180, 195–204; whole file; Server; New; host.NewClient; Server.forgejoHome; Server.SetHost; sodaPublicProbe; canonicalSodaPath; Server.ServeHTTP | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 9 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 39, 153–158; forgejo.New; Server.SetForgejo | [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) | retained | Construct configured native authoritative read client; declarations/fields: `forgejo.New`; Inject native read client consistently into dependent handlers; declarations/fields: `Server.SetForgejo` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 45–46, 181–184, 205–222; auth.New, api.New; Server.ServeHTTP | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Wire native extension admission and established product handlers; declarations/fields: `auth.New`, `api.New`; Native-extension private admission/router; public avatar/intake branches are separately partitioned; declarations/fields: `Server.ServeHTTP` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 47–48; identityclient.New | [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) | retained | Wire native identity metadata/admission client; declarations/fields: `identityclient.New` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 49–50, 134–152; control.NewCoordinator; coordinatorLockPath, Server.StartCoordinator, Server.CloseCoordinator | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Wire sole factory coordinator to shared store/host/broker; declarations/fields: `control.NewCoordinator`; Exclusive factory coordinator process/ledger lifetime; declarations/fields: `coordinatorLockPath`, `Server.StartCoordinator`, `Server.CloseCoordinator` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 51–55; forgejo.NewServiceBackground, ServiceObserver.ShareBackground | [G02](../../slices/forgejo-integration.md#g02-background-service-admission) | retained | Share one background service admission across native consumers; declarations/fields: `forgejo.NewServiceBackground`, `ServiceObserver.ShareBackground` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 56, 58, 70–72, 120–133; api.NewServiceReadinessSource; Coordinator.Readiness; IntakeHandler.ServeHTTP; loadIntakeSecret | [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) | retained | Construct shared native factory read-source adapter; declarations/fields: `api.NewServiceReadinessSource`; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 57; Coordinator.AcceptanceReads | [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) | retained | Connect acceptance verifier to authoritative evidence; declarations/fields: `Coordinator.AcceptanceReads` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 59; Coordinator.DispatchReads | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Connect dispatch to exact accepted prompt/source reads; declarations/fields: `Coordinator.DispatchReads` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 60–61, 106–119; forgejo.NewPublisher; defaultFactoryPublicationRoot, publicationRoot | [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) | retained | Connect conditional publication effects to private publisher; declarations/fields: `forgejo.NewPublisher`; Private publication bundle/workspace root; declarations/fields: `defaultFactoryPublicationRoot`, `publicationRoot` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 62–64; forgejo.NewReviewer | [G05](../../slices/forgejo-integration.md#g05-native-review-submission) | retained | Conditionally connect separate native reviewer; declarations/fields: `forgejo.NewReviewer` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 65–66; forgejo.NewMerger | [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) | retained | Conditionally connect separate native merger; declarations/fields: `forgejo.NewMerger` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 67; forgejo.NewCheckAssessor | [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) | retained | Connect native CI evidence observer using configured merge actor; declarations/fields: `forgejo.NewCheckAssessor` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 79–80, 164–171, 185–194; avatarHandler; publicAvatarPath, canonicalAvatarPath; Server.ServeHTTP | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Public fixed native avatar routes; declarations/fields: `avatarHandler`; Canonical versioned public avatar route; declarations/fields: `publicAvatarPath`, `canonicalAvatarPath`; Public versioned avatar path handling rejects aliases before routing; declarations/fields: `Server.ServeHTTP` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 101–105; Server.CloseTerminals | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Close human terminal operations/streams on service shutdown; declarations/fields: `Server.CloseTerminals` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-a6358cf5b564"></a>

## [internal/web/server_test.go](../../../../../internal/web/server_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; whole file | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 11–46; TestRootOnlyRedirectsToConfiguredForgejo, TestNoNativeDestinationDoesNotLoopOrRender | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Configured native private browser origin routing assertions; declarations/fields: `TestRootOnlyRedirectsToConfiguredForgejo`, `TestNoNativeDestinationDoesNotLoopOrRender` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 47–69; TestCoordinatorLockLivesInFactoryState, TestCoordinatorSharesBackendStoreAndClients | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Exclusive factory coordinator wiring/lifetime assertions; declarations/fields: `TestCoordinatorLockLivesInFactoryState`, `TestCoordinatorSharesBackendStoreAndClients` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-27df89429e2d"></a>

## [internal/web/spaces_inventory_test.go](../../../../../internal/web/spaces_inventory_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27, 108–135, 199–241; file scaffold; TestSpacesLiveWorkSurvivesHistoryCap; TestSpacesProjectCursorReachesOmittedProject | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestSpacesLiveWorkSurvivesHistoryCap; Current declaration duty: TestSpacesProjectCursorReachesOmittedProject — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 28–76, 97–107; inventoryFixture; readSpacesPath; inventoryProject | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: inventoryFixture; Current declaration duty: readSpacesPath; Current declaration duty: inventoryProject — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 77–96, 136–198; inventoryRun; recordInventoryRun; TestSpacesFailedRunReadIsIncomplete; TestSpacesFailedViewReadIsIncomplete; TestSpacesEmptyReadsStayEmpty | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | retained | Current declaration duty: inventoryRun; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-878dc1cb96a0"></a>

## [internal/web/spaces_test.go](../../../../../internal/web/spaces_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–22, 62–80, 101–209; file scaffold; spacesCallback; TestSpacesDeniedUnavailableAndDegradedOwnMembership; TestSpacesNativeActorDenialDoesNotPublishCollection; TestSpacesBoundsAreIncompleteNotCompleteEmpty; TestSpacesResponseByteLimitAndOversizedStoreLabel; TestSpacesAdmissionActorAndQueryBounds | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 23–61; spacesFixture | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current declaration duty: spacesFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 81–100, 210–247; spacesAPI; readSpaces; TestSpacesSlowInspectionCannotPublishAfterNativeRevocation | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: spacesAPI; Current declaration duty: readSpaces; Current declaration duty: TestSpacesSlowInspectionCannotPublishAfterNativeRevocation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f52b58155a77"></a>

## [internal/web/tailnet_test.go](../../../../../internal/web/tailnet_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–17, 143–221; whole file; tailnetOffSettings; TestTailnetOperatorAdmissionAndNativeFailureSecrecy; TestTailnetMutationStrictFieldsAndRequestGuards | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 18–142, 250–363; TestManagedCreateReviewsPolicyWithoutChangingLegacyDefaults, TestTailnetProjectPrivacyAndCurrentOwnership, TestTailnetCreationOptionsRequireCurrentHumanOwner | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Explicit original Project network selection and current owner/member privacy assertions; declarations/fields: `TestManagedCreateReviewsPolicyWithoutChangingLegacyDefaults`, `TestTailnetProjectPrivacyAndCurrentOwnership`, `TestTailnetCreationOptionsRequireCurrentHumanOwner` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 222–249; TestTailnetEnrollmentNeverEchoesInputAndRejectsEndpointOverride | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Enrollment credential secrecy/policy override refusal assertions; declarations/fields: `TestTailnetEnrollmentNeverEchoesInputAndRejectsEndpointOverride` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-ae47785f0daa"></a>

## [internal/web/terminal_test.go](../../../../../internal/web/terminal_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–231; file scaffold; nativeCreationPermit; terminalNativeFixture; count; terminalWebFixture | [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d1a52d7e0cf1"></a>

## [internal/web/test_helpers_test.go](../../../../../internal/web/test_helpers_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24, 37–46, 145–172; file scaffold; roundTrip; RoundTrip; apiTestRequest; nativeProductGeneration; nativeProductHeaders; nativeTestActorID; nativeProductHeadersForActor | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 25–36, 47–144, 173–248; apiTestServer; nativeProductProxy; nativeProductProxyForActor; nativeProductProxyForActorWithCallback; nativeProductProxyForActorWithCallbacks; nativeAPIServe; nativeAPIServeWithCallback; nativeAPIServeWithOptions; nativeProductRequest; stubbedHostWebFixture | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: apiTestServer; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4fe4615bff88"></a>

## [internal/web/testdata/avatar-browser.ts](../../../../../internal/web/testdata/avatar-browser.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; file scaffold; assert | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current scaffold duty: file scaffold; Current method duty: assert — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

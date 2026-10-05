# Backend web auth and composition

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-163b612535f1"></a>

## [internal/web/auth/extension_service.go](../../../../../internal/web/auth/extension_service.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 16–21 | Record, DTO or interface contract extensionRequest for Browser authority and contributions; declarations/fields: `extensionRequest` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 22–29 | Record, DTO or interface contract sessionUserView for Browser authority and contributions; declarations/fields: `sessionUserView` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 30–33 | Service.ExtensionHandler — Browser authority and contributions; declarations/fields: `Service.ExtensionHandler` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 34–54 | Service.serveExtension — Browser authority and contributions; declarations/fields: `Service.serveExtension` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 55–65 | Service.serveExtensionSession — Browser authority and contributions; declarations/fields: `Service.serveExtensionSession` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 66–81 | Service.serveExtensionResource — Browser authority and contributions; declarations/fields: `Service.serveExtensionResource` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 82–96 | Service.serveExtensionAccount — Browser authority and contributions; declarations/fields: `Service.serveExtensionAccount` |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 97–105 | Local preference route; current native authority supplied by G01; declarations/fields: `Service.serveExtensionPreferences` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 106–123, 158–162, 206–239 | Own development public key review/register/remove route; declarations/fields: `Service.serveExtensionDevelopmentKeys`, `Service.serveExtensionDevelopmentKeyRemoval`, `extensionDevelopmentKeyPath`, `Service.apiExtensionForgejoKeys` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 124–131 | allowExtensionNoQuery — Browser authority and contributions; declarations/fields: `allowExtensionNoQuery` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 132–136 | validExtensionPath — Browser authority and contributions; declarations/fields: `validExtensionPath` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 137–157 | allowExtensionMethod — Browser authority and contributions; declarations/fields: `allowExtensionMethod` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 163–170 | Service.extensionRequest — Browser authority and contributions; declarations/fields: `Service.extensionRequest` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 171–184 | extensionIdentity — Browser authority and contributions; declarations/fields: `extensionIdentity` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 185–200 | Service.extensionUser — Browser authority and contributions; declarations/fields: `Service.extensionUser` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 201–205 | extensionSessionGenerationMatches — Browser authority and contributions; declarations/fields: `extensionSessionGenerationMatches` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 240–253 | writeExtensionSession — Browser authority and contributions; declarations/fields: `writeExtensionSession` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 254–261 | revalidateExtensionAuthority — Browser authority and contributions; declarations/fields: `revalidateExtensionAuthority` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 262–275 | Service.extensionMutation — Browser authority and contributions; declarations/fields: `Service.extensionMutation` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 276–281 | validExtensionMutationOrigin — Browser authority and contributions; declarations/fields: `validExtensionMutationOrigin` |

<a id="coverage-53e51f45434b"></a>

## [internal/web/auth/session.go](../../../../../internal/web/auth/session.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G09](../../slices/forgejo-integration.md#g09-local-profile-preferences) / active | 11–39 | Read/update only current actor’s Soda-local display name; declarations/fields: `Service.preferences` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 40–45 | Record, DTO or interface contract developmentKeyView for Development SSH access; declarations/fields: `developmentKeyView` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 46–77 | Service.apiKeys — Development SSH access; declarations/fields: `Service.apiKeys` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 78–101 | Service.apiRemoveDevelopmentKey — Development SSH access; declarations/fields: `Service.apiRemoveDevelopmentKey` |

<a id="coverage-8097e371e5c6"></a>

## [internal/web/factory_lifecycle_test.go](../../../../../internal/web/factory_lifecycle_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–21 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 22–56 | Fixture/protocol support lifecycleWebFixture; declarations/fields: `lifecycleWebFixture` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 57–66 | Fixture/protocol support lifecycleWebRun; declarations/fields: `lifecycleWebRun` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 67–100 | Fixture/protocol support lifecycleWebGrants; declarations/fields: `lifecycleWebGrants` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 101–120 | Fixture/protocol support lifecycleWebPreparations; declarations/fields: `lifecycleWebPreparations` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 121–127 | Fixture/protocol support lifecycleAction; declarations/fields: `lifecycleAction` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 128–166 | Assertions TestFactoryPauseAndResumeJourney: pause: %+v; declarations/fields: `TestFactoryPauseAndResumeJourney` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 167–184 | Assertions TestFactoryActionsRequireCurrentCodeWrite; declarations/fields: `TestFactoryActionsRequireCurrentCodeWrite` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 185–230 | Assertions TestFactoryRunStopAndRetry: stop: %+v; declarations/fields: `TestFactoryRunStopAndRetry` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 231–260 | Assertions TestFactoryTakeoverRequiresMembershipAndSettledRun: takeover: %+v; declarations/fields: `TestFactoryTakeoverRequiresMembershipAndSettledRun` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 261–318 | Project lifecycle coordinates dispatch withdrawal/run reconciliation before Stop/Start; declarations/fields: `TestLifecycleStopAndStartCoordinate` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 319–340 | Authorized inventory current factory control projection; declarations/fields: `TestSpacesShowsFactoryControl` |

<a id="coverage-a4f7afdd4d39"></a>

## [internal/web/factory_settings_test.go](../../../../../internal/web/factory_settings_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–19 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 20–21 | Fixture/protocol support factorySettingsProject; declarations/fields: `factorySettingsProject` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 22–33 | Fixture/protocol support factorySettingsServer; declarations/fields: `factorySettingsServer` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 34–44 | Fixture/protocol support factoryEncryptedServer; declarations/fields: `factoryEncryptedServer` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 45–54 | Fixture/protocol support factoryPolicyBody; declarations/fields: `factoryPolicyBody` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 55–61 | Fixture/protocol support decodeBody: body %q: %v; declarations/fields: `decodeBody` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 62–71 | Fixture/protocol support assertNoSecrets: response leaks %q: %s; declarations/fields: `assertNoSecrets` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 72–95 | Assertions TestFactoryStatusShowsMissingAuthority: empty authority: %+v; declarations/fields: `TestFactoryStatusShowsMissingAuthority` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 96–142 | Assertions TestFactoryPolicyJourney: receipt: %+v; declarations/fields: `TestFactoryPolicyJourney` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 143–176 | Assertions TestFactoryPolicyPauseWithdrawsDispatch: pause receipt: %+v; declarations/fields: `TestFactoryPolicyPauseWithdrawsDispatch` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 177–197 | Operator-only appliance capacity route assertions; declarations/fields: `TestFactoryCapacityIsOperatorOnly` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 198–213 | Repository-owner creation grant route assertions; declarations/fields: `TestFactoryEnvironmentGrantIsOwnerOnly` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 214–259 | Connection-owner/grant/generation sponsorship route assertions; declarations/fields: `TestFactorySponsorshipNeedsConnectionOwner` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 260–291 | Exact preparation requirement acceptance predecessor assertions; declarations/fields: `TestPreparationAcceptanceJourney` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 292–335 | Administrator approval exact requirement/effect assertions; declarations/fields: `TestPreparationApprovalJourney` |

<a id="coverage-900b9381263e"></a>

<a id="internalwebfactory_views_testgo-1"></a>

## [internal/web/factory_views_test.go](../../../../../internal/web/factory_views_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 1–22 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 23–35 | Fixture/protocol support factoryViewsFixture; declarations/fields: `factoryViewsFixture` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 36–45 | Fixture/protocol support factoryViewsRun; declarations/fields: `factoryViewsRun` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 46–49 | Fixture/protocol support factoryViewsHostResponse; declarations/fields: `factoryViewsHostResponse` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 50–62 | Fixture/protocol support readFactoryRun; declarations/fields: `readFactoryRun` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 63–116 | Assertions TestFactoryRunStatusPendingLiveAndExcerpt: pending status: %d; declarations/fields: `TestFactoryRunStatusPendingLiveAndExcerpt` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 117–167 | Assertions TestFactoryRunStatusBindingAndRefusals: bound status: %d; declarations/fields: `TestFactoryRunStatusBindingAndRefusals` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 168–199 | Assertions TestSpacesShowsFactoryRuns: items: %d; declarations/fields: `TestSpacesShowsFactoryRuns` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 200–205 | Fixture/protocol support factoryOutputScript; declarations/fields: `factoryOutputScript` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 206–251 | Fixture/protocol support factoryOutputProxyFixture; declarations/fields: `factoryOutputProxyFixture` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 252–274 | Fixture/protocol support dialFactoryOutput: %v: %s; declarations/fields: `dialFactoryOutput` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 275–289 | Fixture/protocol support readFactoryFrame; declarations/fields: `readFactoryFrame` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 290–293 | Fixture/protocol support factoryOutputHandshakeFor; declarations/fields: `factoryOutputHandshakeFor` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 294–353 | Assertions TestFactoryOutputStreamDeliversStatusOutputAndEOF: status: %v; declarations/fields: `TestFactoryOutputStreamDeliversStatusOutputAndEOF` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 354–376 | Assertions TestFactoryOutputStreamRejectsInput: status: %v; declarations/fields: `TestFactoryOutputStreamRejectsInput` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 377–396 | Assertions TestFactoryOutputStreamRefusesStaleHandshake: stale handshake attached; declarations/fields: `TestFactoryOutputStreamRefusesStaleHandshake` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 397–418 | Assertions TestFactoryOutputStreamClosesOnUnknownAndStale: close: %v; declarations/fields: `TestFactoryOutputStreamClosesOnUnknownAndStale` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 419–439 | Assertions TestFactoryOutputStreamRequiresWrite: read-only viewer attached; declarations/fields: `TestFactoryOutputStreamRequiresWrite` |

<a id="coverage-2b1235840ff6"></a>

## [internal/web/identity_native_test.go](../../../../../internal/web/identity_native_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result. Opt-in native/composed fixture may require explicit credentials, shared native services and retained evidence destinations; no current runtime/qualification claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 18–25 | Fixture/protocol support nativeIdentityFake; declarations/fields: `nativeIdentityFake` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 26–30 | Fixture/protocol support nativeIdentityFake.Connections; declarations/fields: `nativeIdentityFake.Connections` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 31–35 | Fixture/protocol support nativeIdentityFake.StartEnrollment; declarations/fields: `nativeIdentityFake.StartEnrollment` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 36–40, 92–154 | Grant confirmation, member/permission and recipient authority assertions; declarations/fields: `nativeIdentityFake.CreateGrant`, `TestNativeIdentityGrantRequiresConfirmationsMembersAndWrite` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 41–45 | Explicit lease retirement fixture; declarations/fields: `nativeIdentityFake.EndLease` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 46–91 | Assertions TestNativeIdentityActorAndEnrollmentTrust: native actor was not bound to identity metadata; declarations/fields: `TestNativeIdentityActorAndEnrollmentTrust` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 155–201 | Native provider execution compensates generation loss assertions; declarations/fields: `TestNativeIdentityLaunchCompensatesGenerationChange` |

<a id="coverage-d6aac70bcf52"></a>

## [internal/web/lifecycle_access_keys_test.go](../../../../../internal/web/lifecycle_access_keys_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 19–73 | Fixture/protocol support managementWebFixture: untrusted lifecycle target; declarations/fields: `managementWebFixture` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 74–92 | Project Start/Stop admission/confirmation assertions; declarations/fields: `TestLifecycleAuthorizationAndExplicitStop` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 93–118 | Assertions TestSavedKeyRemovalIsOwnOnlyAndNeverNativeRevocation: preference deletion claimed revocation; declarations/fields: `TestSavedKeyRemovalIsOwnOnlyAndNeverNativeRevocation` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 119–155 | Assertions TestOperatorCannotManageAnotherAccountsKeysAndGenerationStillApplies: operator bypassed own membership; declarations/fields: `TestOperatorCannotManageAnotherAccountsKeysAndGenerationStillApplies` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 156–206 | Assertions TestAccessKeyPreviewApplyAndLastKeyConfirmation: stale/caller-selected key set reached native; declarations/fields: `TestAccessKeyPreviewApplyAndLastKeyConfirmation` |

<a id="coverage-f526f87d1223"></a>

## [internal/web/preparation_test.go](../../../../../internal/web/preparation_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 19–20 | Fixture/protocol support preparationProject; declarations/fields: `preparationProject` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 21–44 | Fixture/protocol support preparationFixture; declarations/fields: `preparationFixture` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 45–62 | Fixture/protocol support seedPreparationRecord; declarations/fields: `seedPreparationRecord` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 63–127 | Persistent/native admission hold administrator and synchronization assertions; declarations/fields: `TestPreparationReadShowsHoldAndRoleReadiness`, `TestPreparationHoldRequiresAdministrator`, `TestPreparationHoldAndReleaseSyncMarker` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 128–153 | Authorized inventory preparation summary assertions; declarations/fields: `TestSpacesRowSummarizesPreparation` |

<a id="coverage-158eb27e417c"></a>

## [internal/web/server.go](../../../../../internal/web/server.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–25 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 26–37 | Record, DTO or interface contract Server for Private IPC and service lifetime; declarations/fields: `Server` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 38, 41–44, 68–69, 73–78, 81–88 | New — Private IPC and service lifetime; declarations/fields: `New` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 39 | Construct configured native authoritative read client; declarations/fields: `forgejo.New` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 40 | Construct private host IPC client; declarations/fields: `host.NewClient` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 45–46 | Wire native extension admission and established product handlers; declarations/fields: `auth.New`, `api.New` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 47–48 | Wire native identity metadata/admission client; declarations/fields: `identityclient.New` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 49–50 | Wire sole factory coordinator to shared store/host/broker; declarations/fields: `control.NewCoordinator` |
| [G02](../../slices/forgejo-integration.md#g02-background-service-admission) / active | 51–55 | Share one background service admission across native consumers; declarations/fields: `forgejo.NewServiceBackground`, `ServiceObserver.ShareBackground` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 56 | Construct shared native factory read-source adapter; declarations/fields: `api.NewServiceReadinessSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 57 | Connect acceptance verifier to authoritative evidence; declarations/fields: `Coordinator.AcceptanceReads` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 58 | Connect readiness reconciliation to authoritative issue observations; declarations/fields: `Coordinator.Readiness` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 59 | Connect dispatch to exact accepted prompt/source reads; declarations/fields: `Coordinator.DispatchReads` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 60–61 | Connect conditional publication effects to private publisher; declarations/fields: `forgejo.NewPublisher` |
| [G05](../../slices/forgejo-integration.md#g05-native-review-submission) / active | 62–64 | Conditionally connect separate native reviewer; declarations/fields: `forgejo.NewReviewer` |
| [G07](../../slices/forgejo-integration.md#g07-conditional-native-merge) / active | 65–66 | Conditionally connect separate native merger; declarations/fields: `forgejo.NewMerger` |
| [G06](../../slices/forgejo-integration.md#g06-native-ci-observation) / active | 67 | Connect native CI evidence observer using configured merge actor; declarations/fields: `forgejo.NewCheckAssessor` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 70–72 | Authenticated factory webhook intake route; declarations/fields: `IntakeHandler.ServeHTTP` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 79–80 | Public fixed native avatar routes; declarations/fields: `avatarHandler` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 89–100 | Server.forgejoHome — Private IPC and service lifetime; declarations/fields: `Server.forgejoHome` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 101–105 | Close human terminal operations/streams on service shutdown; declarations/fields: `Server.CloseTerminals` |
| [G04](../../slices/forgejo-integration.md#g04-candidate-publication-and-pr-creation) / active | 106–119 | Private publication bundle/workspace root; declarations/fields: `defaultFactoryPublicationRoot`, `publicationRoot` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 120–133 | Configured native webhook delivery authentication secret; declarations/fields: `loadIntakeSecret` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 134–152 | Exclusive factory coordinator process/ledger lifetime; declarations/fields: `coordinatorLockPath`, `Server.StartCoordinator`, `Server.CloseCoordinator` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 153–158 | Inject native read client consistently into dependent handlers; declarations/fields: `Server.SetForgejo` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 159–163 | Inject private host IPC client consistently into dependent handlers; declarations/fields: `Server.SetHost` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 164–171 | Canonical versioned public avatar route; declarations/fields: `publicAvatarPath`, `canonicalAvatarPath` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 172–175 | sodaPublicProbe — Private IPC and service lifetime; declarations/fields: `sodaPublicProbe` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 176–180 | canonicalSodaPath — Private IPC and service lifetime; declarations/fields: `canonicalSodaPath` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 181–184, 205–222 | Native-extension private admission/router; public avatar/intake branches are separately partitioned; declarations/fields: `Server.ServeHTTP` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 185–194 | Public versioned avatar path handling rejects aliases before routing; declarations/fields: `Server.ServeHTTP` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 195–204 | Direct backend health/root probes preserve service routing boundary; declarations/fields: `Server.ServeHTTP` |

<a id="coverage-a6358cf5b564"></a>

## [internal/web/server_test.go](../../../../../internal/web/server_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 11–46 | Configured native private browser origin routing assertions; declarations/fields: `TestRootOnlyRedirectsToConfiguredForgejo`, `TestNoNativeDestinationDoesNotLoopOrRender` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 47–69 | Exclusive factory coordinator wiring/lifetime assertions; declarations/fields: `TestCoordinatorLockLivesInFactoryState`, `TestCoordinatorSharesBackendStoreAndClients` |

<a id="coverage-f52b58155a77"></a>

## [internal/web/tailnet_test.go](../../../../../internal/web/tailnet_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 18–142, 250–363 | Explicit original Project network selection and current owner/member privacy assertions; declarations/fields: `TestManagedCreateReviewsPolicyWithoutChangingLegacyDefaults`, `TestTailnetProjectPrivacyAndCurrentOwnership`, `TestTailnetCreationOptionsRequireCurrentHumanOwner` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 143–146 | Fixture/protocol support tailnetOffSettings; declarations/fields: `tailnetOffSettings` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 147–186 | Assertions TestTailnetOperatorAdmissionAndNativeFailureSecrecy: unexpected native dispatch; declarations/fields: `TestTailnetOperatorAdmissionAndNativeFailureSecrecy` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 187–221 | Assertions TestTailnetMutationStrictFieldsAndRequestGuards; declarations/fields: `TestTailnetMutationStrictFieldsAndRequestGuards` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 222–249 | Enrollment credential secrecy/policy override refusal assertions; declarations/fields: `TestTailnetEnrollmentNeverEchoesInputAndRejectsEndpointOverride` |


# Developer scripts

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-3dcf4de704f5"></a>

## [scripts/build-soda-extension.ts](../../../../../scripts/build-soda-extension.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–123, 139–182; native extension asset build and inventory | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Check source entry/style graph against the extension manifest, bundle browser JS/CSS, copy lock-hash-verified terminal assets, and enumerate generated browser payload files. — Current source inspected at scripts/build-soda-extension.ts; concrete renderer/build/test consumer is named in the selector and description. |
| 124–138; copyNotices | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Copy font and Lit notices for the staged branding/font/browser payload. — Current source inspected at scripts/build-soda-extension.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-e6723e4eae8c"></a>

## [scripts/check-forgejo-branding.ts](../../../../../scripts/check-forgejo-branding.ts)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43, 63, 74, 84–185; prior detailed responsibility interval | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Declarations/fixtures and integration for Observe native Forgejo contrast, controls, focus and image presentation; 4 named units assigned here; remaining selectors preserve each duty — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/check-forgejo-branding.ts; manifest identical_prior confirms byte identity |
| 44–62, 64–73; prior detailed responsibility interval | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | contrast — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `contrast`; colors — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `colors` — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/check-forgejo-branding.ts; manifest identical_prior confirms byte identity |
| 75–83; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | visit — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `visit` — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/check-forgejo-branding.ts; manifest identical_prior confirms byte identity |

<a id="coverage-fd92209a2c50"></a>

## [scripts/check-native.sh](../../../../../scripts/check-native.sh)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–15, 32–34; prior detailed responsibility interval | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Require matching native substrate, pinned tools and clean exact revision before qualification; declarations/fields: `revision`, `GOTOOLCHAIN`; Run source suite and guard unchanged revision; explicitly distinguish installed qualification; declarations/fields: `check:source`, `revision` — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/check-native.sh; manifest identical_prior confirms byte identity |
| 16–31; prior detailed responsibility interval | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Require candidate artifact identities and choose artifact-local verifier path; declarations/fields: `candidate`, `artifacts`, `verifier`; Legacy sealed-stage verifier branch invokes its artifact-local verify command; declarations/fields: `verifier`; Bind both checkouts and invoke current Rust soda-candidate-check over candidate archives; declarations/fields: `forgejo_revision`, `soda-candidate-check` — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/check-native.sh; manifest identical_prior confirms byte identity |

<a id="coverage-439506db437a"></a>

## [scripts/console_welcome_test.go](../../../../../scripts/console_welcome_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–47; file scaffold; consoleFixture | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: consoleFixture — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 48–96, 118–136; TestConsoleUsesConfiguredOriginsAndNativeUplinks; TestConsoleUsesConfiguredDashboardPort; TestConsoleDoesNotRenderNonOperatorOrUnsafeOrigins | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Current declaration duty: TestConsoleUsesConfiguredOriginsAndNativeUplinks; Current declaration duty: TestConsoleUsesConfiguredDashboardPort; Current declaration duty: TestConsoleDoesNotRenderNonOperatorOrUnsafeOrigins — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 97–117, 137–165; TestConsoleShowsForgejoInstallerBeforeSetup; TestConsoleHookKeepsNoninteractiveSSHQuiet; TestConsoleWelcomeInstallWiring | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current declaration duty: TestConsoleShowsForgejoInstallerBeforeSetup; Current declaration duty: TestConsoleHookKeepsNoninteractiveSSHQuiet; Current declaration duty: TestConsoleWelcomeInstallWiring — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b79767bad4b6"></a>

## [scripts/fixtures/portcontracts/cli_surface.json](../../../../../scripts/fixtures/portcontracts/cli_surface.json)

Exact byte-identical prior detailed map units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–16; prior detailed responsibility interval | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Frozen factory operator lifecycle/status CLI bytes — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 17–32; prior detailed responsibility interval | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Frozen identity-compose CLI surface bytes — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 33–41; prior detailed responsibility interval | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Frozen image-import CLI refusal bytes — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 42–50; prior detailed responsibility interval | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Frozen native host Tailnet CLI surface bytes — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 51–58; prior detailed responsibility interval | [N07](../../slices/networking.md#n07-git-endpoint-advertisement) | retained | Frozen Git endpoint advertisement helper CLI bytes — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 59–68, 76–77; prior detailed responsibility interval | [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) | retained | Frozen operator bootstrap CLI refusal bytes; Frozen operator-token/bootstrap configuration flags — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 69–75; prior detailed responsibility interval | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Frozen private-origin/setup flags checked through current Rust source adapter — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |
| 78–82; prior detailed responsibility interval | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Frozen first-boot database provisioning/config flag closure — docs/development/ideal-filetree-plan/coverage/maps/developer-scripts.md#scripts/fixtures/portcontracts/cli_surface.json; manifest identical_prior confirms byte identity |

<a id="coverage-b23dac91d26d"></a>

## [scripts/fixtures/portcontracts/systemd_wiring.json](../../../../../scripts/fixtures/portcontracts/systemd_wiring.json)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5; installed project-init path contract | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Frozen expected installed executable path for project-init; checked by systemd port-contract tests. — Static checked-in JSON fixture; consumer scripts/system_formats_test.go decodes this fixture. |
| 6–40; host socket/service credential and startup contracts; identity private IPC socket/service access contracts | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Fixture asserts host daemon socket/service startup and credential wiring.; Fixture asserts identity socket/service path and access contracts. — Static fixture consumed by systemd_wiring contract test in scripts/system_formats_test.go. |
| 41–47; native Project Tailnet companion unit contract | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Fixture asserts Tailnet companion unit wiring for the native Project service. — Static fixture consumed by systemd_wiring contract test in scripts/system_formats_test.go. |
| 48–53; Muse provider runtime binding/maintenance unit paths | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Fixture asserts provider runtime unit path and maintenance binding. — Static fixture consumed by systemd_wiring contract test in scripts/system_formats_test.go. |
| 54–61; PostgreSQL provisioning executable and secret wiring | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Fixture asserts PG provisioning service executable and secret path wiring. — Static fixture consumed by systemd_wiring contract test in scripts/system_formats_test.go. |

<a id="coverage-74588ab35864"></a>

## [scripts/forgejo_account_details_test.go](../../../../../scripts/forgejo_account_details_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–248; file scaffold; forgejoAccountDetailFixture; forgejoAccountDetailFixtures; TestForgejoAccountDetailOverridesRetainUpstreamAttribution; TestForgejoOAuthApplicationListKeepsNativeFormsAndActions; TestForgejoAccountDetailsPreserveNativeSecurityAndActions; forgejoAccountDisabledFeatures; Contains; TestForgejoAccountKeysKeepNativeCapabilityGates; TestForgejoAccountSecurityKeepsRequiredTwoFactorBoundary; TestForgejoAccountDetailsCSSIsScoped; readForgejoAccountDetailTemplate | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-65d87d7782c0"></a>

## [scripts/forgejo_account_settings_test.go](../../../../../scripts/forgejo_account_settings_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254; file scaffold; TestForgejoPersonalSettingsNavigationGates; TestForgejoAccountSettingsLayoutComposesNativeSeams; forgejoCleanupFixtureType; Name; TestForgejoPersonalAdaptersKeepNativeRootContext; TestForgejoPersonalActionsUseScopedEmptyStatesAndWarning; forgejoSettingsFixtureLocale; Tr; TestForgejoPersonalProviderSectionAbsentWithoutProviders; TestForgejoOAuthHeadingScopedByCallerNotNativeSettingsFlag; forgejoSettingsFixtureStrings; Join; TestForgejoOAuthEditorPreservesOtherCallers; TrSize; TestForgejoAvatarSourceChoices | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d9dd24828acb"></a>

## [scripts/forgejo_admin_details_test.go](../../../../../scripts/forgejo_admin_details_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–264; file scaffold; TestForgejoAdminDetailsOverridesMatchStock1509; TestForgejoAdminAuthEditParityRejectsMutatedLeaf; TestForgejoAdminDetailsKeepNativeActionsAndBranches; TestForgejoAdminDetailsCSSIsScopedAndResponsive | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8a72114fd120"></a>

## [scripts/forgejo_admin_monitoring_test.go](../../../../../scripts/forgejo_admin_monitoring_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–111; file scaffold; TestForgejoAdminMonitoringStockParityAndParse; forgejoMonitoringLocale; Tr; forgejoMonitoringContext; forgejoMonitoringFunctions; TestForgejoAdminMonitoringPreservesNativeWarningsAndEmptyNotice | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3297f4dac301"></a>

## [scripts/forgejo_admin_org_test.go](../../../../../scripts/forgejo_admin_org_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–186; file scaffold; TestForgejoAdminOrganizationOverridesRetain1509Source; TestForgejoAdminOrganizationCompositionBoundaries; TestForgejoAdminOrganizationStylesStayFamilyScoped; TestForgejoSettingsComponentKeepsNativeBoundaries | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-ae95041f54bb"></a>

## [scripts/forgejo_auth_test.go](../../../../../scripts/forgejo_auth_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–69; file scaffold; TestForgejoSecondaryAuthPagesOnlyAddPresentationRoot; TestForgejoSecondaryAuthStylesRemainPageScoped | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoSecondaryAuthPagesOnlyAddPresentationRoot; Current declaration duty: TestForgejoSecondaryAuthStylesRemainPageScoped — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a626fb9865cc"></a>

## [scripts/forgejo_cargo_test.go](../../../../../scripts/forgejo_cargo_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77; file scaffold; forgejoCargoLocale; Lang; forgejoCargoStrings; Split; Tr; forgejoCargoContext; TestForgejoCargoComposesNativeIndexBranches | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-317949102f9e"></a>

## [scripts/forgejo_code_search_test.go](../../../../../scripts/forgejo_code_search_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33; file scaffold; TestForgejoCodeSearchStylesHaveOneScopedOwner | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoCodeSearchStylesHaveOneScopedOwner — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9aec2f464928"></a>
<a id="scriptsforgejo_components_testgo-1"></a>

## [scripts/forgejo_components_test.go](../../../../../scripts/forgejo_components_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9; file scaffold | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 10–386; TestForgejoPageIntroComposition; TestForgejoEmptyContentComposition; TestForgejoExploreNavbarDelegatesNativePolicyAndOverflow; TestForgejoExplorePagesComposeNativeControlsWithOriginalContext; TestForgejoExploreEmptyOwnsOnlyVisibleActions; TestForgejoPagesComposeSharedPresentationWithNativeBoundaries | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current declaration duty: TestForgejoPageIntroComposition; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-32be88b4ac62"></a>

## [scripts/forgejo_federated_auth_test.go](../../../../../scripts/forgejo_federated_auth_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–313; file scaffold; forgejo1509FederatedTemplateHashes; forgejoFederatedPresentationEdits; TestForgejoFederatedAuthOverridesMatchStock1509ApartFromPresentation; TestForgejoFederatedAuthPreservesNativeSecurityAndDelegation; TestForgejoFederatedAuthOverridesParseWithNativeSeams; TestForgejoFederatedAuthGuestToggleFollowsNativeRouteFlags; TestForgejoFederatedAuthNativeAccountLinkBranches; TestForgejoFederatedAuthGrantErrorKeepsRepositoryHeaderBranch; TestForgejoFederatedAuthCSSIsScopedAndAttributed; readForgejoFederatedTemplate | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-59e7d59502f0"></a>

## [scripts/forgejo_form_components_test.go](../../../../../scripts/forgejo_form_components_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–120; file scaffold; TestForgejoRepositoryCreationKeepsNativePermissionBranches; TestForgejoNativeFormAdapterSelectsMainFormsOnly | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoRepositoryCreationKeepsNativePermissionBranches; Current declaration duty: TestForgejoNativeFormAdapterSelectsMainFormsOnly — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a15e5235cd87"></a>

## [scripts/forgejo_form_layout_test.go](../../../../../scripts/forgejo_form_layout_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–85; file scaffold; readForgejoTemplateForUpstreamParity; reverseFormPresentationDeltas; stripSodaPClasses; expandAuthLeafInvocations; expandAuthLeafInvocationsWithReader; reconstructAuthLeaf | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8774427c828a"></a>

## [scripts/forgejo_home_redesign_test.go](../../../../../scripts/forgejo_home_redesign_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–96; file scaffold; writeHomePreviews; previewUnwritable; TestForgejoHomeRedesign | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-52ec8e64c7bf"></a>

## [scripts/forgejo_insights_test.go](../../../../../scripts/forgejo_insights_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–231; file scaffold; TestForgejoInsightsOverridesMatchStock1509; forgejoInsightsPermission; CanRead; TestForgejoActivityDelegatesNativeInsightBranches; forgejoGraphRef; TestForgejoGraphPreservesNativeModesRefsAndContent; TestForgejoInsightsCSSIsPageScoped | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3abc7a522e59"></a>

## [scripts/forgejo_migrate_test.go](../../../../../scripts/forgejo_migrate_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; file scaffold; forgejoMigrateBinary | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: forgejoMigrateBinary — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 49–122; TestForgejoMigrateScrubsDatabasePasswdOnly; TestForgejoMigrateSkipsMissingConfig; TestForgejoMigrateUnitWiring | [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) | retained | Current declaration duty: TestForgejoMigrateScrubsDatabasePasswdOnly; Current declaration duty: TestForgejoMigrateSkipsMissingConfig; Current declaration duty: TestForgejoMigrateUnitWiring — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-262aa15c24db"></a>

## [scripts/forgejo_native_pages_test.go](../../../../../scripts/forgejo_native_pages_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–116; file scaffold; dashboardFixtureUser; ShortName; renderForgejoDashboard; TestForgejoDashboardKeepsNativeContent; renderForgejoAdminDashboard; TestForgejoAdminDashboardKeepsNativeOperations | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cbe3cc0e0143"></a>

## [scripts/forgejo_notification_preview_test.go](../../../../../scripts/forgejo_notification_preview_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–131; file scaffold; previewRequestContext; FormBool; previewTemplateContext; previewDates; TimeSince; previewNotification; Link; TestForgejoNotificationPreviewRendering; TestForgejoNotificationPreviewSignedInHook | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-90389805883c"></a>

## [scripts/forgejo_onboarding_test.go](../../../../../scripts/forgejo_onboarding_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–258; file scaffold; TestForgejoOnboardingPreservesNativeRoutesAndFields; TestForgejoOnboardingMigrationProvidersUseSharedFormLayout; TestForgejoOnboardingMigratingPreservesNativeRuntimeHooks; migrationChooserService; String; TestForgejoMigrationChooserRendersOnlyAvailableSourcesAndPreservesContext; entryNames; readForgejoAssetFile | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-332bba8daad1"></a>

## [scripts/forgejo_org_details_test.go](../../../../../scripts/forgejo_org_details_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–64; file scaffold; TestForgejoOrgDetailsSourceParityAndParse | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoOrgDetailsSourceParityAndParse — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-461abefdaaac"></a>

## [scripts/forgejo_org_home_test.go](../../../../../scripts/forgejo_org_home_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–63; file scaffold; TestForgejoOrganizationHomeSourceParity; TestForgejoOrganizationHomeNativeCreationAndVisibility | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoOrganizationHomeSourceParity; Current declaration duty: TestForgejoOrganizationHomeNativeCreationAndVisibility — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2a9c9779444c"></a>

## [scripts/forgejo_org_projects_test.go](../../../../../scripts/forgejo_org_projects_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–32; file scaffold; TestForgejoOrgProjectsPreserveNativeContextsAndProjectPartials | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoOrgProjectsPreserveNativeContextsAndProjectPartials — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-44a7026dfa67"></a>

## [scripts/forgejo_owner_code_test.go](../../../../../scripts/forgejo_owner_code_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–57; file scaffold; TestForgejoOwnerCodePreservesNativeSearchContext | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoOwnerCodePreservesNativeSearchContext — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a3b972a9c9a7"></a>

## [scripts/forgejo_packages_test.go](../../../../../scripts/forgejo_packages_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–240; file scaffold; TestForgejoCodeSearchKeepsNativeSearchContext; TestForgejoPackagesOwnerPagesRetainOrganizationAndUserBranches; TestForgejoPackagesNativeActionsAndProtocolPartialsRemain; TestForgejoPackagesStylesStayScoped; TestForgejoPackagesCleanupRulesKeepNativeActionsAndData | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-2107d34ed930"></a>

## [scripts/forgejo_presentation_test.go](../../../../../scripts/forgejo_presentation_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–190; file scaffold; TestForgejoPresentationGallery | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoPresentationGallery — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fa74dd9f7ba4"></a>

## [scripts/forgejo_profiles_test.go](../../../../../scripts/forgejo_profiles_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–189; file scaffold; forgejoProfilesLocale; Tr; forgejoProfilesContext; TestForgejoProfilesKeepNativePageWorkflows; TestForgejoProfilesComposeNativeBranchesWithOriginalContext; TestForgejoProfilesHeaderComposesNativeRoutes; TestForgejoSharedProfileCardCallersOptInExplicitly; TestForgejoProfilesStylesStayInsideProfileRoot | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-54d03e5651d9"></a>

## [scripts/forgejo_project_board_test.go](../../../../../scripts/forgejo_project_board_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–48; file scaffold; TestForgejoProjectBoardKeepsNativeBehaviorHooks; TestForgejoProjectBoardStylesDoNotOwnDragState | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoProjectBoardKeepsNativeBehaviorHooks; Current declaration duty: TestForgejoProjectBoardStylesDoNotOwnDragState — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-0c77efe96439"></a>

## [scripts/forgejo_repository_code_test.go](../../../../../scripts/forgejo_repository_code_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–186; file scaffold; forgejoCSSImport; inlineForgejoCSSImports; TestForgejoCodeAndWorkflowOverridesKeepNativeBodies; TestForgejoCodeEditorsKeepExplicitFormAndNativeRoot; TestForgejoCodeAndWorkflowStylesUsePositiveFamilyMarkers | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1aa2d000b613"></a>

## [scripts/forgejo_repository_content_test.go](../../../../../scripts/forgejo_repository_content_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–136; file scaffold; TestForgejoRepositoryContentOverridesRetain1509Source; TestForgejoRepositoryContentKeepsNativeGatesAndPartials; TestForgejoWikiSearchFragmentRetains1509Structure; TestForgejoReleaseTagHeaderRetainsNativePolicy; TestForgejoRepositoryContentStylesStayScoped | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-57e51435f7a2"></a>

## [scripts/forgejo_repository_general_settings_test.go](../../../../../scripts/forgejo_repository_general_settings_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–143; file scaffold; readForgejoTemplateClosure; TestForgejoRepositoryGeneralSettingsUseOpenSections; TestForgejoRepositoryUnitsRetainNativeControlContracts | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c2383f874299"></a>

## [scripts/forgejo_repository_issues_test.go](../../../../../scripts/forgejo_repository_issues_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–131; file scaffold; TestForgejoRepositoryIssuePagesKeepNativeWorkflows; TestForgejoPullFragmentsKeepNativeHooks; TestForgejoRepositoryIssueStylesStayScoped | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-dba35a061750"></a>

## [scripts/forgejo_repository_settings_collections_test.go](../../../../../scripts/forgejo_repository_settings_collections_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–142; file scaffold; readRepositorySettingsCollectionTemplate; TestForgejoRepositorySettingsCollectionsUseOpenSettingsComposition; TestForgejoRepositorySettingsCollectionsRetainNativeInteractionContracts; TestForgejoRepositorySettingsCollectionsUseSharedEmptyAndNoticeRoles | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-909cc8d5aabf"></a>

## [scripts/forgejo_repository_settings_details_test.go](../../../../../scripts/forgejo_repository_settings_details_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–147; file scaffold; TestForgejoRepositorySettingsLeavesUseSharedShell; TestForgejoWebhookPartialsRetain1509Source; TestForgejoRepositorySettingsDetailStylesStayScoped; TestForgejoSharedRunnerStylesStayWithSharedPartial; TestForgejoSharedRunnerDetailsRetain1509Source | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9de5e138dc80"></a>

## [scripts/forgejo_repository_settings_navigation_test.go](../../../../../scripts/forgejo_repository_settings_navigation_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–90; file scaffold; forgejoSettingsRepository; UnitEnabled; forgejoSettingsPermission; CanRead; TestForgejoRepositorySettingsNavigationMatchesNativeGates | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1bb567d3aadb"></a>

## [scripts/forgejo_repository_test.go](../../../../../scripts/forgejo_repository_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–87; file scaffold; TestForgejoRepositoryHeaderKeepsNativeAuthority; TestForgejoRepositorySettingsUseSharedLayout; TestForgejoRepositoryStylesStayScopedToNativePages | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-48ddfc5c3faa"></a>

## [scripts/forgejo_setup_test.go](../../../../../scripts/forgejo_setup_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–290; file scaffold; forgejoSetupLocale; Tr; TrString; forgejoSetupContext; TestForgejoSetupOverridesMatchStock1509; TestForgejoSetupKeepsNativeFieldAndHookContract; TestForgejoSetupRendersNativeConditionalBranches; TestForgejoPostInstallKeepsNativeRedirectFallback; TestForgejoSetupStylesStayPageScoped | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b5780c3969df"></a>

## [scripts/forgejo_shared_projects_test.go](../../../../../scripts/forgejo_shared_projects_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–129; file scaffold; forgejoSharedProjectsLocale; Tr; PrettyNumber; forgejoSharedProjectsContext; TestForgejoSharedProjectsPreserveNativeOwnershipAndForms; TestForgejoSharedProjectsHaveOneStyleOwner; TestForgejoSharedProjectsFormExecutesNewAndEditBranches | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b129cf70d90b"></a>

## [scripts/forgejo_soda_settings_test.go](../../../../../scripts/forgejo_soda_settings_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; file scaffold; TestSodaNavigationComesFromForgejoExtensionManifest | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestSodaNavigationComesFromForgejoExtensionManifest — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-941a3631a08d"></a>

## [scripts/forgejo_status_test.go](../../../../../scripts/forgejo_status_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–95; file scaffold; TestForgejoStatusWrappersPreserveNativeDiagnostics; TestForgejoStatus404RendersDefaultAndEscapedCustomPrompt; TestForgejoStatus500RetainsUpstreamPanicFallback | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5019511e4210"></a>

## [scripts/forgejo_template_fixture_test.go](../../../../../scripts/forgejo_template_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–75; file scaffold; forgejoTemplateLocale; Tr; TrN; forgejoTemplateContext; forgejoTemplateDict; readForgejoTemplate; forgejoTemplateCallPattern; templateCalls; requireForgejoTemplateCalls | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8c66b485b217"></a>

## [scripts/forgejo_theme_components_test.go](../../../../../scripts/forgejo_theme_components_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; file scaffold; TestForgejoThemeToggleIsSingletonAtEachPlacement; TestForgejoHeaderLoadsGuestThemeScriptOnlyForToggleRoutes | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoThemeToggleIsSingletonAtEachPlacement; Current declaration duty: TestForgejoHeaderLoadsGuestThemeScriptOnlyForToggleRoutes — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d983c2bc981e"></a>

## [scripts/pg_backup_test.go](../../../../../scripts/pg_backup_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–22, 109–174, 191–208, 265–274; TestPostgresBackupRoundTrip | [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) | retained | Backup/restore fixture declarations and command constants; declarations/fields: `TestPostgresBackupRoundTrip`; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 23–108; pgMaintenanceBinary, pgFixtureStart, pgExec, pgQuery | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Compile Rust maintenance commands, start disposable native PG fixture, execute bounded SQL observations; declarations/fields: `pgMaintenanceBinary`, `pgFixtureStart`, `pgExec`, `pgQuery` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 175–190, 259–264; TestPostgresBackupRoundTrip | [O06](../../slices/operator-administration.md#o06-database-restore) | retained | Destroy fixture tables then invoke shipped restore --yes and verify exact recovered rows; declarations/fields: `TestPostgresBackupRoundTrip`; Assert restore refuses without explicit --yes; declarations/fields: `TestPostgresBackupRoundTrip` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 209–258; TestPostgresBackupRoundTrip | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Assert bare-cluster initialization, role/database idempotency, quoted passwords and missing-secret refusal; declarations/fields: `TestPostgresBackupRoundTrip` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-11c762b0da49"></a>

## [scripts/pg_runtime_test.go](../../../../../scripts/pg_runtime_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30, 187–202; file scaffold; readRuntimeFile; requireContains; TestFixtureMatchesProductImage | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–103; TestPostgresUnitTopology; TestForgejoUnitPostgresWiring; TestPostgresProvisionUnitWiring; TestDashboardUnitDatabaseWiring | [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) | retained | Current declaration duty: TestPostgresUnitTopology; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 104–186; TestPostgresBackupScheduleAndStaging | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestPostgresBackupScheduleAndStaging — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b4808466efce"></a>

## [scripts/preview-spaces.ts](../../../../../scripts/preview-spaces.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–314; Spaces preview developer command | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Developer preview harness that starts and tears down a local Spaces preview; it consumes the built app and is not installed product behavior. — Current source inspected at scripts/preview-spaces.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-1c32651191ee"></a>

## [scripts/render_forgejo_branding_test.go](../../../../../scripts/render_forgejo_branding_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–51; file scaffold; TestForgejoBrandingPNGPlacements; TestForgejoBrandingRendererRejectsUnknownArguments | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoBrandingPNGPlacements; Current declaration duty: TestForgejoBrandingRendererRejectsUnknownArguments — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8c633e7cc6b5"></a>

## [scripts/render_forgejo_native_test.go](../../../../../scripts/render_forgejo_native_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–18; file scaffold; TestForgejoBrandingMatchesSVGMaster | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestForgejoBrandingMatchesSVGMaster — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-daa5e4e83890"></a>

## [scripts/screenshot.ts](../../../../../scripts/screenshot.ts)

Changed current body; prior ownership not reused as exact selectors

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–346; screenshot capture developer command | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Developer screenshot harness that opens the local preview, captures requested views, and manages browser/output lifecycle; consumer is the visual review workflow. — Current source inspected at scripts/screenshot.ts; concrete renderer/build/test consumer is named in the selector and description. |

<a id="coverage-8b6923c976fa"></a>

## [scripts/sodaspaces_templates_test.go](../../../../../scripts/sodaspaces_templates_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; file scaffold | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 13–57, 90–105; TestForgejoFooterKeepsLiveFeatures; TestForgejoFooterUsesPrefixedNativeAssets | [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) | retained | Current declaration duty: TestForgejoFooterKeepsLiveFeatures; Current declaration duty: TestForgejoFooterUsesPrefixedNativeAssets — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 58–89; TestSodaspacesRequestLoggingOmitsQueries | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Current declaration duty: TestSodaspacesRequestLoggingOmitsQueries — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-6da9e7c7704f"></a>

## [scripts/system_formats_test.go](../../../../../scripts/system_formats_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–50, 165–267; file scaffold; buildGoPortBinary; configVector; portedSourcePin; TestCLISurface | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 51–114; TestDashboardConfigVectors; TestDashboardConfigLimits | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current declaration duty: TestDashboardConfigVectors; Current declaration duty: TestDashboardConfigLimits — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 115–164; TestSystemdWiring | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Current declaration duty: TestSystemdWiring — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c3c155cde21c"></a>

## [scripts/terminal_branding_test.go](../../../../../scripts/terminal_branding_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–52; file scaffold; TestTerminalBrandingMatchesCanonicalSymbol; TestFastfetchRendersTerminalBranding | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestTerminalBrandingMatchesCanonicalSymbol; Current declaration duty: TestFastfetchRendersTerminalBranding — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-aba474b526fe"></a>

## [scripts/wire_contracts_test.go](../../../../../scripts/wire_contracts_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–49, 106–209; file scaffold; portContractFixture; vectorEnvelope; strictVector; operatorWire; buildRustPortBinary; buildRustPortBinaryAs; capturedRequest; wireStubServer; runPortBinary | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 50–105; TestStrictjsonWireVectors; TestStrictjsonWireLimits | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Current declaration duty: TestStrictjsonWireVectors; Current declaration duty: TestStrictjsonWireLimits — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 210–301; TestOperatorWireBytes | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Current declaration duty: TestOperatorWireBytes — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

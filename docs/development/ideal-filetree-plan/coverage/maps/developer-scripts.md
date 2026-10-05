# Developer scripts

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-3dcf4de704f5"></a>

## [scripts/build-soda-extension.ts](../../../../../scripts/build-soda-extension.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–25 | Native extension entry/style graph and source path declarations; declarations/fields: `entries`, `styles`, `source` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 26–48 | Assert style use and extension manifest entry closure; declarations/fields: `checkStylesUsedByEntries`, `checkManifestEntries` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 49–104 | Compile native extension split script graph and bundled stylesheet graph; declarations/fields: `buildScripts`, `buildStyles` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 105–123 | Copy and verify already-acquired locked terminal renderer files; declarations/fields: `copyLockedTerminal` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 124–138 | Stage exact native extension icon/Lit/font notices; declarations/fields: `copyNotices` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 139–182 | Inventory complete native browser payload and expose build command; declarations/fields: `filesUnder`, `buildSodaExtensionAssets` |

<a id="coverage-e6723e4eae8c"></a>

## [scripts/check-forgejo-branding.ts](../../../../../scripts/check-forgejo-branding.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–43, 63, 74, 84, 110 | Declarations/fixtures and integration for Observe native Forgejo contrast, controls, focus and image presentation |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 44–62 | contrast — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `contrast` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 64–73 | colors — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `colors` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 75–83 | visit — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `visit` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 85–109 | checkButtons — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `checkButtons` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 111–123 | checkFocusAndImages — Observe native Forgejo contrast, controls, focus and image presentation; declarations/fields: `checkFocusAndImages` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 124–185 | Observe native Forgejo contrast, controls, focus and image presentation |

<a id="coverage-fd92209a2c50"></a>

## [scripts/check-native.sh](../../../../../scripts/check-native.sh)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–15 | Require matching native substrate, pinned tools and clean exact revision before qualification; declarations/fields: `revision`, `GOTOOLCHAIN` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 16–21 | Require candidate artifact identities and choose artifact-local verifier path; declarations/fields: `candidate`, `artifacts`, `verifier` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 22–25 | Legacy sealed-stage verifier branch invokes its artifact-local verify command; declarations/fields: `verifier` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 26–31 | Bind both checkouts and invoke current Rust soda-candidate-check over candidate archives; declarations/fields: `forgejo_revision`, `soda-candidate-check` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 32–34 | Run source suite and guard unchanged revision; explicitly distinguish installed qualification; declarations/fields: `check:source`, `revision` |

<a id="coverage-439506db437a"></a>

## [scripts/console_welcome_test.go](../../../../../scripts/console_welcome_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 1–12, 47, 71, 96, 117, 136, 144 | Declarations/fixtures and integration for Console guidance and quiet installed welcome-hook contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 13–46 | Test harness consoleFixture — Console guidance and quiet installed welcome-hook contracts; declarations/fields: `consoleFixture` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 48–70 | Assert Console Uses Configured Origins And Native Uplinks; declarations/fields: `TestConsoleUsesConfiguredOriginsAndNativeUplinks` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 72–95 | Assert Console Uses Configured Dashboard Port; declarations/fields: `TestConsoleUsesConfiguredDashboardPort` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 97–116 | Assert Console Shows Forgejo Installer Before Setup; declarations/fields: `TestConsoleShowsForgejoInstallerBeforeSetup` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 118–135 | Assert Console Does Not Render Non Operator Or Unsafe Origins; declarations/fields: `TestConsoleDoesNotRenderNonOperatorOrUnsafeOrigins` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 137–143 | Assert Console Hook Keeps Noninteractive SSHQuiet; declarations/fields: `TestConsoleHookKeepsNoninteractiveSSHQuiet` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 145–165 | Assert Console Welcome Install Wiring; declarations/fields: `TestConsoleWelcomeInstallWiring` |

<a id="coverage-b79767bad4b6"></a>

## [scripts/fixtures/portcontracts/cli_surface.json](../../../../../scripts/fixtures/portcontracts/cli_surface.json)

Tracked canonical assertion/reference data with current test consumers; no generator provenance assumed.Historical Go source strings remain active oracle inputs through portedSourcePin's current Rust mapping, not active Go runtime owners.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 1–16 | Frozen factory operator lifecycle/status CLI bytes |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 17–32 | Frozen identity-compose CLI surface bytes |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 33–41 | Frozen image-import CLI refusal bytes |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 42–50 | Frozen native host Tailnet CLI surface bytes |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 51–58 | Frozen Git endpoint advertisement helper CLI bytes |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 59–68 | Frozen operator bootstrap CLI refusal bytes |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 69–75 | Frozen private-origin/setup flags checked through current Rust source adapter |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 76–77 | Frozen operator-token/bootstrap configuration flags |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 78–82 | Frozen first-boot database provisioning/config flag closure |

<a id="coverage-b23dac91d26d"></a>

## [scripts/fixtures/portcontracts/systemd_wiring.json](../../../../../scripts/fixtures/portcontracts/systemd_wiring.json)

Tracked canonical assertion/reference data with current test consumers; no generator provenance assumed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–5 | Frozen staged project-init executable path |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 6–18 | Frozen host socket/service credential and startup contracts |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 19–40 | Frozen identity private IPC socket/service access contracts |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 41–47 | Frozen native Project Tailnet companion units |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 48–53 | Frozen Muse provider runtime binding/maintenance unit paths |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 54–61 | Frozen PG provisioning unit executable/secret wiring |

<a id="coverage-9aec2f464928"></a>

<a id="scriptsforgejo_components_testgo-1"></a>

## [scripts/forgejo_components_test.go](../../../../../scripts/forgejo_components_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–17, 21, 28–32, 47, 115, 175, 203, 265, 346, 420, 472, 547, 626, 661, 671–673, 681 | Declarations/fixtures and integration for Native Forgejo components template/control contracts |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 18–20 | Tr — Native Forgejo components template/control contracts; declarations/fields: `Tr` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 22–27 | TrN — Native Forgejo components template/control contracts; declarations/fields: `TrN` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 33–46 | forgejoTemplateDict — Native Forgejo components template/control contracts; declarations/fields: `forgejoTemplateDict` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 48–114 | Assert Forgejo Page Intro Composition; declarations/fields: `TestForgejoPageIntroComposition` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 116–174 | Assert Forgejo Empty Content Composition; declarations/fields: `TestForgejoEmptyContentComposition` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 176–202 | Assert Forgejo Explore Navbar Delegates Native Policy And Overflow; declarations/fields: `TestForgejoExploreNavbarDelegatesNativePolicyAndOverflow` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 204–264 | Assert Forgejo Explore Pages Compose Native Controls With Original Context; declarations/fields: `TestForgejoExplorePagesComposeNativeControlsWithOriginalContext` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 266–345 | Assert Forgejo Explore Empty Owns Only Visible Actions; declarations/fields: `TestForgejoExploreEmptyOwnsOnlyVisibleActions` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 347–419 | Assert Forgejo Theme Toggle Is Singleton At Each Placement; declarations/fields: `TestForgejoThemeToggleIsSingletonAtEachPlacement` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 421–471 | Assert Forgejo Header Loads Guest Theme Script Only For Toggle Routes; declarations/fields: `TestForgejoHeaderLoadsGuestThemeScriptOnlyForToggleRoutes` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 473–546 | Assert Forgejo Repository Creation Keeps Native Permission Branches; declarations/fields: `TestForgejoRepositoryCreationKeepsNativePermissionBranches` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 548–625 | Assert Forgejo Pages Compose Shared Presentation With Native Boundaries; declarations/fields: `TestForgejoPagesComposeSharedPresentationWithNativeBoundaries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 627–660 | Assert Forgejo Native Form Adapter Selects Main Forms Only; declarations/fields: `TestForgejoNativeFormAdapterSelectsMainFormsOnly` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 662–670 | readForgejoTemplate — Native Forgejo components template/control contracts; declarations/fields: `readForgejoTemplate` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 674–680 | templateCalls — Native Forgejo components template/control contracts; declarations/fields: `templateCalls` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 682–691 | requireForgejoTemplateCalls — Native Forgejo components template/control contracts; declarations/fields: `requireForgejoTemplateCalls` |

<a id="coverage-3abc7a522e59"></a>

## [scripts/forgejo_migrate_test.go](../../../../../scripts/forgejo_migrate_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 1–22, 46–48, 87, 95 | Declarations/fixtures and integration for Existing Forgejo database-secret configuration migration |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 23–45 | Test harness forgejoMigrateBinary — Existing Forgejo database-secret configuration migration; declarations/fields: `forgejoMigrateBinary` |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 49–86 | Assert Forgejo Migrate Scrubs Database Passwd Only; declarations/fields: `TestForgejoMigrateScrubsDatabasePasswdOnly` |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 88–94 | Assert Forgejo Migrate Skips Missing Config; declarations/fields: `TestForgejoMigrateSkipsMissingConfig` |
| [O03](../../slices/operator-administration.md#o03-existing-install-credential-maintenance) / active | 96–122 | Assert Forgejo Migrate Unit Wiring; declarations/fields: `TestForgejoMigrateUnitWiring` |

<a id="coverage-d983c2bc981e"></a>

## [scripts/pg_backup_test.go](../../../../../scripts/pg_backup_test.go)

Source/assertion inspection only; no suite or native scenario executed. Native integration driver compiles shipped Rust commands and uses disposable pinned Postgres fixture conditional on native prerequisites. Source coverage only; no successful backup/restore execution claim.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 1–22 | Backup/restore fixture declarations and command constants; declarations/fields: `TestPostgresBackupRoundTrip` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 23–108 | Compile Rust maintenance commands, start disposable native PG fixture, execute bounded SQL observations; declarations/fields: `pgMaintenanceBinary`, `pgFixtureStart`, `pgExec`, `pgQuery` |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 109–119 | Seed real two-database state for round-trip assertions; declarations/fields: `TestPostgresBackupRoundTrip` |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 120–174 | Assert complete private role/database dumps and secret-free bounded backup progress; declarations/fields: `TestPostgresBackupRoundTrip` |
| [O06](../../slices/operator-administration.md#o06-database-restore) / active | 175–190 | Destroy fixture tables then invoke shipped restore --yes and verify exact recovered rows; declarations/fields: `TestPostgresBackupRoundTrip` |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 191–208 | Assert retention keeps newest complete backups and preserves progress files; declarations/fields: `TestPostgresBackupRoundTrip` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 209–258 | Assert bare-cluster initialization, role/database idempotency, quoted passwords and missing-secret refusal; declarations/fields: `TestPostgresBackupRoundTrip` |
| [O06](../../slices/operator-administration.md#o06-database-restore) / active | 259–264 | Assert restore refuses without explicit --yes; declarations/fields: `TestPostgresBackupRoundTrip` |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 265–274 | Assert invalid zero retention refuses backup; declarations/fields: `TestPostgresBackupRoundTrip` |

<a id="coverage-11c762b0da49"></a>

## [scripts/pg_runtime_test.go](../../../../../scripts/pg_runtime_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 1–12 | Postgres source topology test declarations |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 13–29 | Read shipping source and assert literal wiring; declarations/fields: `readRuntimeFile`, `requireContains` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 30–59 | Assert PG service data/role secret ownership and provisioning dependency; declarations/fields: `TestPostgresUnitTopology` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 60–77 | Assert Forgejo Postgres startup/secret wiring; declarations/fields: `TestForgejoUnitPostgresWiring` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 78–94 | Assert installed first-boot provisioning service and staged command; declarations/fields: `TestPostgresProvisionUnitWiring` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 95–102 | Assert dashboard native database/secret wiring; declarations/fields: `TestDashboardUnitDatabaseWiring` |
| [O01](../../slices/operator-administration.md#o01-first-boot-database-provisioning) / active | 103–124 | Assert role initialization before native database consumers; declarations/fields: `TestPostgresBackupScheduleAndStaging` |
| [O05](../../slices/operator-administration.md#o05-database-backup-and-retention) / active | 125–152 | Assert root private backup schedule, native timer and retention/temp ownership; declarations/fields: `TestPostgresBackupScheduleAndStaging` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 153–185 | Assert compiled PG provision/backup/restore payload staging and obsolete shell omission; declarations/fields: `TestPostgresBackupScheduleAndStaging` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 186–202 | Assert disposable native PG fixture uses same pinned database image as product; declarations/fields: `TestFixtureMatchesProductImage` |

<a id="coverage-b4808466efce"></a>

## [scripts/preview-spaces.ts](../../../../../scripts/preview-spaces.ts)

Synthetic test/preview behavior only. Domain mapping identifies represented responsibilities, without claiming an active product service.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–28 | Declare restricted local preview server/scenario options; declarations/fields: `startSpacesPreview`, `Fault`, `Review`, `SocketData`, `json` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 29–66 | Compile development client and create/reset disposable synthetic workspace models; declarations/fields: `buildClient`, `reset` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 67–93 | Simulate bounded fixture terminal output and selected shell command replies; declarations/fields: `startSpacesPreview` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 94–192 | Serve constrained loopback fixture assets and delegate synthetic product API to shared model; declarations/fields: `fetch` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 193–205 | Simulate native extension generation renewal for browser authority tests; declarations/fields: `fetch` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 206–264 | Simulate terminal socket attachment/input/close/error lifecycle; declarations/fields: `websocket` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 265–314 | Return restricted preview control facade and development CLI lifetime; declarations/fields: `startSpacesPreview` |

<a id="coverage-daa5e4e83890"></a>

## [scripts/screenshot.ts](../../../../../scripts/screenshot.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–37, 65–66, 100 | Declarations/fixtures and integration for Authorized browser screenshot capture and source-matching review checks |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 38–64 | Set reviewed native theme and wait finite visual transitions for exact presentation captures; declarations/fields: `setCaptureTheme` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 67–99 | Capture restricted local Spaces component fixture to private selected output directory; declarations/fields: `captureSpacesComponent` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 101–249 | Parse restricted capture/login options and open test browser/profile/output lifetime; declarations/fields: `main` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 250–277 | Verify native presentation revision/cache registry and exact delivered stylesheet bytes before capture; declarations/fields: `main` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 278–340 | Apply selected review theme/local CSS and capture bounded screenshots/landmarks; retire browser lifetime; declarations/fields: `main` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 341–346 | Authorized browser screenshot capture and source-matching review checks |

<a id="coverage-8b6923c976fa"></a>

## [scripts/sodaspaces_templates_test.go](../../../../../scripts/sodaspaces_templates_test.go)

Source/assertion inspection only; no suite or native scenario executed. Native service diagnostic configuration belongs to existing O04 host/service administration: appliance/config/forgejo.env:15–19 selects native console diagnostics and a bounded query-free access template. Test renders a synthetic request only; native journal visibility/retention and installed operation remain unverified and unspecified here.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 1–12, 57, 89 | Declarations/fixtures and integration for Native footer contributions and request-log redaction assertions |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 13–56 | Assert Forgejo Footer Keeps Live Features; declarations/fields: `TestForgejoFooterKeepsLiveFeatures` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 58–88 | Assert native Forgejo console diagnostic configuration excludes query/referrer data and emits only method, escaped path and status; declarations/fields: `TestSodaspacesRequestLoggingOmitsQueries` |
| [G08](../../slices/forgejo-integration.md#g08-native-forgejo-presentation) / active | 90–105 | Assert Forgejo Footer Uses Prefixed Native Assets; declarations/fields: `TestForgejoFooterUsesPrefixedNativeAssets` |

<a id="coverage-6da9e7c7704f"></a>

## [scripts/system_formats_test.go](../../../../../scripts/system_formats_test.go)

Source/assertion inspection only; no suite or native scenario executed. Historical Go source strings remain active oracle inputs through portedSourcePin's current Rust mapping, not active Go runtime owners.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 1–24, 41–50, 103, 114, 160–164, 180 | Declarations/fixtures and integration for Shared config vectors, installed service wiring and port CLI fixture adapter |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 25–40 | Test harness buildGoPortBinary — Shared config vectors, installed service wiring and port CLI fixture adapter; declarations/fields: `buildGoPortBinary` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 51–102 | Assert Dashboard Config Vectors; declarations/fields: `TestDashboardConfigVectors` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 104–113 | Assert Dashboard Config Limits; declarations/fields: `TestDashboardConfigLimits` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 115–159 | Assert Systemd Wiring; declarations/fields: `TestSystemdWiring` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 165–179 | Test harness portedSourcePin — Shared config vectors, installed service wiring and port CLI fixture adapter; declarations/fields: `portedSourcePin` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 181–263 | Assert CLISurface; declarations/fields: `TestCLISurface` |

<a id="coverage-aba474b526fe"></a>

## [scripts/wire_contracts_test.go](../../../../../scripts/wire_contracts_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–23, 34–49, 77, 105–131, 151–155, 182, 201 | Declarations/fixtures and integration for Strict JSON and factory command transport byte contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 24–33 | Test harness portContractFixture — Strict JSON and factory command transport byte contracts; declarations/fields: `portContractFixture` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 50–76 | Assert Strictjson Wire Vectors; declarations/fields: `TestStrictjsonWireVectors` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 78–104 | Assert Strictjson Wire Limits; declarations/fields: `TestStrictjsonWireLimits` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 132–150 | Test harness buildRustPortBinary — Strict JSON and factory command transport byte contracts; declarations/fields: `buildRustPortBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 156–181 | Test harness wireStubServer — Strict JSON and factory command transport byte contracts; declarations/fields: `wireStubServer` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 183–200 | Test harness runPortBinary — Strict JSON and factory command transport byte contracts; declarations/fields: `runPortBinary` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 202–293 | Assert Operator Wire Bytes; declarations/fields: `TestOperatorWireBytes` |


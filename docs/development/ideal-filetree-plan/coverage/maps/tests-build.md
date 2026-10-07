# Tests build

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-0bc00438cba7"></a>

## [tests/build/avatar_integration_test.go](../../../../../tests/build/avatar_integration_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–30, 84–105; file scaffold; activateBinary; runActivate; caddyBinary; adaptCaddy | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 31–75, 106–303; TestAvatarHelpReportsPrivateActivationSurface; TestAvatarMissingBindIPRejected; TestAvatarTLSModesRejectedBeforeEffects; TestAvatarUnknownFlagsRejected; TestAvatarNonrootRefusedWithoutEffects; TestAvatarLocalTLSUsesNativeIssuerWithoutAutomaticClientTrust; TestAvatarProductionRoutesWithTestOwnedUpstreams | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Current declaration duty: TestAvatarHelpReportsPrivateActivationSurface; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 76–83; TestAvatarNoAvatarRuntimeCommand | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current declaration duty: TestAvatarNoAvatarRuntimeCommand — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8119412e1db7"></a>

## [tests/build/candidate_gate_test.go](../../../../../tests/build/candidate_gate_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–24, 31–69; whole file; TestCandidateCheckUsesTheRustValidatorOwner; TestCandidateNoIndependentPythonCandidateRules; TestCandidateNativeGuardsStillSurroundTheOwnerCheck | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Declarations/fixtures and integration for Exact Rust candidate-validator ownership and surrounding native guards; 4 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 25–30; splitNative | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness splitNative — Exact Rust candidate-validator ownership and surrounding native guards; declarations/fields: `splitNative` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-15900411bd9c"></a>

## [tests/build/complexity_scope_test.go](../../../../../tests/build/complexity_scope_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–43; file scaffold; runGate; TestComplexityToolsFilesPassAsShipping; TestComplexityScriptsStillOutOfScope; TestComplexityPrecommitRoutesStagedToolsFiles | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-59cd051bc129"></a>

## [tests/build/forgejo_domain_test.go](../../../../../tests/build/forgejo_domain_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–9, 14, 19–48; whole file; TestForgejoDomainHelpReportsRecoveryVerbs; TestForgejoDomainMissingVerbRejected; TestForgejoDomainInvalidVerbRejected; TestForgejoDomainNonrootRefusedWithoutEffects | [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) | retained | Declarations/fixtures and integration for Privileged whole Forgejo-domain recovery helper surface/refusals; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 10–13, 15–18; domainBinary; runDomain | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness domainBinary — Privileged whole Forgejo-domain recovery helper surface/refusals; declarations/fields: `domainBinary`; Test harness runDomain — Privileged whole Forgejo-domain recovery helper surface/refusals; declarations/fields: `runDomain` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-79a49e2b7383"></a>

## [tests/build/forgejo_payload_test.go](../../../../../tests/build/forgejo_payload_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25, 97–105; file scaffold; payloadFiles; extensionPages | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: payloadFiles; Current declaration duty: extensionPages — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 26–96; TestForgejoPayloadExactSourcesTemplateClosureAndNotices | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestForgejoPayloadExactSourcesTemplateClosureAndNotices — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 106–123; TestForgejoPayloadExtensionPagesDeclaredInNativePackage; TestForgejoPayloadOperatorSettingsUseSeparatePackage | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestForgejoPayloadExtensionPagesDeclaredInNativePackage; Current declaration duty: TestForgejoPayloadOperatorSettingsUseSeparatePackage — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-494c395f5566"></a>

## [tests/build/helpers.go](../../../../../tests/build/helpers.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–181; file scaffold; RepoRoot; Check; Require; ReadFile; ReadJSON; TempDir; WriteFile; ProcResult; RunOpt; Run; SetEnv; CargoBinary; tail | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 14 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="cargo-helper-regressions"></a>

## [tests/build/helpers_test.go](../../../../../tests/build/helpers_test.go)

Selective implementation delta at `04ddbff1`; whole file is H06-owned and retained.
`writeFakeCargo`, `cargoTestEnv`, `TestCargoBinaryHelperProcess`,
`runCargoBinaryHelper`, `TestCargoBinaryEmptyStderrFailureIsNotCachedAsSuccess`,
`TestCargoBinaryUsesAndKeysSelectedTargetDirectory` and `countLines` exercise the
production Cargo helper's process outcome, caching and selected target path.
The actual subprocess checks passed; no native shipping identity is inferred.

<a id="coverage-3695dc8b875f"></a>

## [tests/build/muse_exec_test.go](../../../../../tests/build/muse_exec_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21, 33, 70–124; whole file; TestMuseExecStreamContract; TestMuseExecRejectsBadAuthFile | [I08](../../slices/identity-brokering.md#i08-muse-adapter) | retained | Declarations/fixtures and integration for Muse headless exec stream and staged-auth refusal protocol; Assert Muse Exec Stream Contract; declarations/fields: `TestMuseExecStreamContract`; Assert Muse Exec Rejects Bad Auth File; declarations/fields: `TestMuseExecRejectsBadAuthFile` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 22–32, 34–69; museBinary; runMuseExec | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Test harness museBinary — Muse headless exec stream and staged-auth refusal protocol; declarations/fields: `museBinary`; Test harness runMuseExec — Muse headless exec stream and staged-auth refusal protocol; declarations/fields: `runMuseExec` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-ef6ef7c8d23b"></a>

## [tests/build/native_support_test.go](../../../../../tests/build/native_support_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–72; file scaffold; renderBinary; provisionFixture; newProvisionFixture; renderTo; readButane; butaneFile; fileMode | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 73–101, 120–154, 171–192; TestProvisioningMinimalAndExtensionProfilesAreDistinct; TestProvisioningHostKeyContentsNeverBecomeProcessArguments; TestProvisioningPublicBootstrapHasNoIdentityOrAutomaticReboot; TestProvisioningInstalledSystemDefaultsToPasswordSSH | [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) | retained | Current declaration duty: TestProvisioningMinimalAndExtensionProfilesAreDistinct; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 102–119; TestProvisioningPrivateModesAndPlaintextAreRejectedBeforeOutput | [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) | retained | Current declaration duty: TestProvisioningPrivateModesAndPlaintextAreRejectedBeforeOutput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 155–170; TestProvisioningInstalledConsoleWelcomeRunsBeforeLogin | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Current declaration duty: TestProvisioningInstalledConsoleWelcomeRunsBeforeLogin — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 193–216; TestOutsideSupportToolsAreNotApplianceCommands | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestOutsideSupportToolsAreNotApplianceCommands — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 217–225; TestOutsideBootBindsBrowserServicesWithoutQuadletEnable | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Current declaration duty: TestOutsideBootBindsBrowserServicesWithoutQuadletEnable — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-8a2a508f90bf"></a>

## [tests/build/operator_probe_test.go](../../../../../tests/build/operator_probe_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–42; file scaffold; TestOperatorProbeMissingFailedNoisyAndQuietHooks | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestOperatorProbeMissingFailedNoisyAndQuietHooks — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-89796f4162bc"></a>

## [tests/build/project_account_test.go](../../../../../tests/build/project_account_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–135; file scaffold; accountFailure; accountBinary; accountEnv; accountUseraddDouble; accountUsermodDouble; newAccountEnv; run; commands; read; mode; accountDoc; requireAccountFailure | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 136–154, 191–214, 226–335; TestAccountOnlyProvisionsLockedHomeMarkerSharedAndEmptyKeyfile; TestAccountJoinIsNotKeyApplyAndPreservesDrift; TestAccountOccupiedInputsAndUnassociatedUsersRefuseBeforeCommands; TestAccountValidationBeforeNativeEffects; TestAccountLockContentionRefusesBeforeObservationOrCommands; TestAccountLockCoversFileDurabilityAndAccountCommands; TestAccountFailedProvisioningReleasesLockWithoutRemovingPartialFiles; TestAccountStdinContractBounds | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Current declaration duty: TestAccountOnlyProvisionsLockedHomeMarkerSharedAndEmptyKeyfile; 8 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 155–190, 215–225, 336–357; TestAccountSelectedKeysAndCreationOwnerPrivilegeRemainReal; TestAccountOnlyRetryNeverErasesExistingKeys; TestAccountKeySymlinkIsNotFollowed; TestAccountNativePasswordSSHPolicyIsNotRelaxed | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: TestAccountSelectedKeysAndCreationOwnerPrivilegeRemainReal; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-69be5372dbab"></a>

## [tests/build/project_factory_roles_accounts_test.go](../../../../../tests/build/project_factory_roles_accounts_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35; file scaffold; TestRolesEnsureProvisionsLockedRolesWithoutExtraGroups; TestRolesEnsureRefusesInteractiveOrGroupedAccounts | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRolesEnsureProvisionsLockedRolesWithoutExtraGroups; Current declaration duty: TestRolesEnsureRefusesInteractiveOrGroupedAccounts — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5f0d96ed072b"></a>

## [tests/build/project_factory_roles_fixture_test.go](../../../../../tests/build/project_factory_roles_fixture_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–228; file scaffold; rolesEnv; rolesBinary; rolesSetup; rolesRun; rolesOK; rolesRefused; rolesDigest; rolesMarshal; rolesApprove; rolesFixture; rolesRecord; rolesOp; rolesMode; rolesWaitFile; rolesDeadPGID; rolesGroupAlive | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 17 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-622b2b629b31"></a>

## [tests/build/project_factory_roles_inputs_test.go](../../../../../tests/build/project_factory_roles_inputs_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–78; file scaffold; TestRolesApproveWritesProtectedSnapshotAndVerifiesBundle; TestRolesApproveRejectsUntrustedInputsBeforeEffects | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRolesApproveWritesProtectedSnapshotAndVerifiesBundle; Current declaration duty: TestRolesApproveRejectsUntrustedInputsBeforeEffects — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-9c227c180184"></a>

## [tests/build/project_factory_roles_lifecycle_test.go](../../../../../tests/build/project_factory_roles_lifecycle_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–98; file scaffold; TestRolesStartSpawnsSupervisorAndReportsRunning; TestRolesDeadSupervisorReportsInterrupted; TestRolesStopBarsUnknownIdentityAndRetiresKnown; TestRolesHoldDeniesApproveAndReleaseNeedsRevisionAndQuiescence; TestRolesSupervisorLivenessTracksProcessGroup | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d7b1f1e79903"></a>

## [tests/build/project_factory_roles_output_test.go](../../../../../tests/build/project_factory_roles_output_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–36; file scaffold; TestRolesRunAsRoleCapturesBoundedOutputAndExit | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRolesRunAsRoleCapturesBoundedOutputAndExit — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-4ae86c9a22af"></a>

## [tests/build/project_factory_roles_readiness_test.go](../../../../../tests/build/project_factory_roles_readiness_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–27; file scaffold; TestRolesRecordReportsWaitingAndFailedPhases; TestRolesLauncherRefusalFailsWithoutStart | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestRolesRecordReportsWaitingAndFailedPhases; Current declaration duty: TestRolesLauncherRefusalFailsWithoutStart — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-bca40bd658e0"></a>
<a id="testsbuildproject_factory_roles_testgo-1"></a>

## [tests/build/project_factory_roles_test.go](../../../../../tests/build/project_factory_roles_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11; file scaffold | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–29; TestRolesRejectsMalformedRequests | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Current declaration duty: TestRolesRejectsMalformedRequests — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3d25499b3001"></a>

## [tests/build/project_foundation_test.go](../../../../../tests/build/project_foundation_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–11; file scaffold | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 12–31; TestFoundationRecipeDeclaresNativeDevelopmentFoundation | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: TestFoundationRecipeDeclaresNativeDevelopmentFoundation — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 32–43; TestFoundationRecipeStagesCompiledProjectHelpers | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestFoundationRecipeStagesCompiledProjectHelpers — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 44–67; TestFoundationInstalledProbeRequiresScopeBeforeWrites | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Current declaration duty: TestFoundationInstalledProbeRequiresScopeBeforeWrites — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-5d2f753c73ed"></a>

## [tests/build/project_runtime_test.go](../../../../../tests/build/project_runtime_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–37; file scaffold; parseUnit | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: parseUnit — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 38–47; TestRuntimeCreationImageIdentityHasRecipeAndOSGuard | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Current declaration duty: TestRuntimeCreationImageIdentityHasRecipeAndOSGuard — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 48–66; TestRuntimeServiceUsesLocalEngineAndActivationFD | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Current declaration duty: TestRuntimeServiceUsesLocalEngineAndActivationFD — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 67–104, 112–132; TestRuntimeSocketIsProjectAdminOnly; TestRuntimeOnlyNetworkSysctlSubtreeIsRebound; TestRuntimeComposeDatabaseImageIsDigestPinned; TestRuntimeComposeUsesNativeSecretNotPasswordArgv | [P11](../../slices/projects.md#p11-nested-services-and-volumes) | retained | Current declaration duty: TestRuntimeSocketIsProjectAdminOnly; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 105–111; TestRuntimeSharedToolsUsesLiveIsolationAddress | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Current declaration duty: TestRuntimeSharedToolsUsesLiveIsolationAddress — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-1c79b22c4553"></a>

## [tests/build/proxy_image_test.go](../../../../../tests/build/proxy_image_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–21; file scaffold; TestProxyImageIsDigestPinned | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: TestProxyImageIsDigestPinned — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-d8e4f05bf26a"></a>

## [tests/build/sodaspaces_test.go](../../../../../tests/build/sodaspaces_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35, 83–125; file scaffold; sodaspacesFiles; testVMBinary; stageBinary; copyTree; chmodTree; checkMode | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 7 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 36–55; TestSodaspacesSpacesEntryIsPackagedByTheExtension | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Current declaration duty: TestSodaspacesSpacesEntryIsPackagedByTheExtension — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 56–67; TestSodaspacesVMWebTunnelUsesOnlyTheNativeBrowserOrigin | [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) | retained | Current declaration duty: TestSodaspacesVMWebTunnelUsesOnlyTheNativeBrowserOrigin — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 68–82; TestSodaspacesAccessProbeRejectsPrivateBadRequestBeforeNativeCommands | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Current declaration duty: TestSodaspacesAccessProbeRejectsPrivateBadRequestBeforeNativeCommands — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 126–378; TestSodaspacesActualStageRecipeWithSyntheticBuildInputs | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestSodaspacesActualStageRecipeWithSyntheticBuildInputs — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-cc7cc5efe638"></a>

## [tests/build/source_checks_test.go](../../../../../tests/build/source_checks_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–172; whole file; gateStub; newSourceFixture; run; commands; commandNames; equalCommands; TestSourceCommandsWorkWithoutAStageOrGitCheckout; TestSourceEachFailureStopsRemainingSuites | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Declarations/fixtures and integration for Source suite sequencing/failure-stop and candidate gate integration; 9 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 173–198; TestSourceNativeGateStillSurroundsSharedSourceChecks | [D05](../../slices/release-and-installation.md#d05-artifact-verification) | retained | Assert Source Native Gate Still Surrounds Shared Source Checks; declarations/fields: `TestSourceNativeGateStillSurroundsSharedSourceChecks` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-5375d7279fb9"></a>

## [tests/build/tailnet_image_test.go](../../../../../tests/build/tailnet_image_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10; file scaffold | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 11–17, 28–34; TestTailnetNoStoredPins; TestTailnetBuildWiresLiveInputsWithoutLock | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Current declaration duty: TestTailnetNoStoredPins; Current declaration duty: TestTailnetBuildWiresLiveInputsWithoutLock — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 18–27, 35–39; TestTailnetRecipeFloatsOnBuildArgs; TestTailnetObservedVersionsRecordedWithoutGate | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestTailnetRecipeFloatsOnBuildArgs; Current declaration duty: TestTailnetObservedVersionsRecordedWithoutGate — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-67c6456606a2"></a>

## [tests/build/terminal_assets_test.go](../../../../../tests/build/terminal_assets_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–23; file scaffold; packageScripts | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: packageScripts — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 24–54; TestTerminalAssetsBrowserBuildPreparesLockedRenderer | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Current declaration duty: TestTerminalAssetsBrowserBuildPreparesLockedRenderer — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 55–81; TestTerminalAssetsShippingLockHasOnlyExactLocalRendererFiles | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | Current declaration duty: TestTerminalAssetsShippingLockHasOnlyExactLocalRendererFiles — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-751f619a131a"></a>

## [tests/build/workload_probe_test.go](../../../../../tests/build/workload_probe_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–156; file scaffold; fakeCommand; invokeWorkloads; countCalls; hasWord; TestWorkloadStartOnceThenReadiness; TestWorkloadCheckNeverStartsOrBuilds; TestWorkloadFailedReadinessIsBoundedAndNotSuccess; TestWorkloadInvalidModeNeverInvokesNativeCommands | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 9 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

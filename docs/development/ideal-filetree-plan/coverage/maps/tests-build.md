# Tests build

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-0bc00438cba7"></a>

## [tests/build/avatar_integration_test.go](../../../../../tests/build/avatar_integration_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 1–20, 25, 30, 39, 45, 59, 65, 75, 83, 94, 105, 139 | Declarations/fixtures and integration for Private activation/issuer/routes assertions using test-owned native upstreams |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 21–24 | Test harness activateBinary — Private activation/issuer/routes assertions using test-owned native upstreams; declarations/fields: `activateBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 26–29 | Test harness runActivate — Private activation/issuer/routes assertions using test-owned native upstreams; declarations/fields: `runActivate` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 31–38 | Assert Avatar Help Reports Private Activation Surface; declarations/fields: `TestAvatarHelpReportsPrivateActivationSurface` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 40–44 | Assert Avatar Missing Bind IPRejected; declarations/fields: `TestAvatarMissingBindIPRejected` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 46–58 | Assert Avatar TLSModes Rejected Before Effects; declarations/fields: `TestAvatarTLSModesRejectedBeforeEffects` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 60–64 | Assert Avatar Unknown Flags Rejected; declarations/fields: `TestAvatarUnknownFlagsRejected` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 66–74 | Assert Avatar Nonroot Refused Without Effects; declarations/fields: `TestAvatarNonrootRefusedWithoutEffects` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 76–82 | Assert Avatar No Avatar Runtime Command; declarations/fields: `TestAvatarNoAvatarRuntimeCommand` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 84–93 | Test harness caddyBinary — Private activation/issuer/routes assertions using test-owned native upstreams; declarations/fields: `caddyBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 95–104 | Test harness adaptCaddy — Private activation/issuer/routes assertions using test-owned native upstreams; declarations/fields: `adaptCaddy` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 106–138 | Assert Avatar Local TLSUses Native Issuer Without Automatic Client Trust; declarations/fields: `TestAvatarLocalTLSUsesNativeIssuerWithoutAutomaticClientTrust` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 140–303 | Assert Avatar Production Routes With Test Owned Upstreams; declarations/fields: `TestAvatarProductionRoutesWithTestOwnedUpstreams` |

<a id="coverage-8119412e1db7"></a>

## [tests/build/candidate_gate_test.go](../../../../../tests/build/candidate_gate_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 1–24, 31, 47, 57 | Declarations/fixtures and integration for Exact Rust candidate-validator ownership and surrounding native guards |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 25–30 | Test harness splitNative — Exact Rust candidate-validator ownership and surrounding native guards; declarations/fields: `splitNative` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 32–46 | Assert Candidate Check Uses The Rust Validator Owner; declarations/fields: `TestCandidateCheckUsesTheRustValidatorOwner` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 48–56 | Assert Candidate No Independent Python Candidate Rules; declarations/fields: `TestCandidateNoIndependentPythonCandidateRules` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 58–69 | Assert Candidate Native Guards Still Surround The Owner Check; declarations/fields: `TestCandidateNativeGuardsStillSurroundTheOwnerCheck` |

<a id="coverage-59cd051bc129"></a>

## [tests/build/forgejo_domain_test.go](../../../../../tests/build/forgejo_domain_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 1–9, 14, 19, 25, 31, 38 | Declarations/fixtures and integration for Privileged whole Forgejo-domain recovery helper surface/refusals |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 10–13 | Test harness domainBinary — Privileged whole Forgejo-domain recovery helper surface/refusals; declarations/fields: `domainBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 15–18 | Test harness runDomain — Privileged whole Forgejo-domain recovery helper surface/refusals; declarations/fields: `runDomain` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 20–24 | Assert Forgejo Domain Help Reports Recovery Verbs; declarations/fields: `TestForgejoDomainHelpReportsRecoveryVerbs` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 26–30 | Assert Forgejo Domain Missing Verb Rejected; declarations/fields: `TestForgejoDomainMissingVerbRejected` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 32–37 | Assert Forgejo Domain Invalid Verb Rejected; declarations/fields: `TestForgejoDomainInvalidVerbRejected` |
| [O04](../../slices/operator-administration.md#o04-native-host-administration-and-updates) / active | 39–48 | Assert Forgejo Domain Nonroot Refused Without Effects; declarations/fields: `TestForgejoDomainNonrootRefusedWithoutEffects` |

<a id="coverage-79a49e2b7383"></a>

## [tests/build/forgejo_payload_test.go](../../../../../tests/build/forgejo_payload_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–12, 25, 96, 105, 113 | Declarations/fixtures and integration for Canonical Forgejo payload/template/license closure and native extension declarations |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 13–24 | Test harness payloadFiles — Canonical Forgejo payload/template/license closure and native extension declarations; declarations/fields: `payloadFiles` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 26–95 | Assert Forgejo Payload Exact Sources Template Closure And Notices; declarations/fields: `TestForgejoPayloadExactSourcesTemplateClosureAndNotices` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 97–104 | Test harness extensionPages — Canonical Forgejo payload/template/license closure and native extension declarations; declarations/fields: `extensionPages` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 106–112 | Assert Forgejo Payload Extension Pages Declared In Native Package; declarations/fields: `TestForgejoPayloadExtensionPagesDeclaredInNativePackage` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 114–123 | Assert Forgejo Payload Operator Settings Use Separate Package; declarations/fields: `TestForgejoPayloadOperatorSettingsUseSeparatePackage` |

<a id="coverage-3695dc8b875f"></a>

## [tests/build/muse_exec_test.go](../../../../../tests/build/muse_exec_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 1–21, 33, 70, 90 | Declarations/fixtures and integration for Muse headless exec stream and staged-auth refusal protocol |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 22–32 | Test harness museBinary — Muse headless exec stream and staged-auth refusal protocol; declarations/fields: `museBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 34–69 | Test harness runMuseExec — Muse headless exec stream and staged-auth refusal protocol; declarations/fields: `runMuseExec` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 71–89 | Assert Muse Exec Stream Contract; declarations/fields: `TestMuseExecStreamContract` |
| [I08](../../slices/identity-brokering.md#i08-muse-adapter) / active | 91–124 | Assert Muse Exec Rejects Bad Auth File; declarations/fields: `TestMuseExecRejectsBadAuthFile` |

<a id="coverage-ef6ef7c8d23b"></a>

## [tests/build/native_support_test.go](../../../../../tests/build/native_support_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 1–12, 17–23, 34, 42, 51, 65, 72, 101, 119, 143, 154, 170, 192, 216 | Declarations/fixtures and integration for Installed provisioning payload/support command wiring contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 13–16 | Test harness renderBinary — Installed provisioning payload/support command wiring contracts; declarations/fields: `renderBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 24–33 | Test harness newProvisionFixture — Installed provisioning payload/support command wiring contracts; declarations/fields: `newProvisionFixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 35–41 | Test harness renderTo — Installed provisioning payload/support command wiring contracts; declarations/fields: `renderTo` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 43–50 | Test harness readButane — Installed provisioning payload/support command wiring contracts; declarations/fields: `readButane` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 52–64 | Test harness butaneFile — Installed provisioning payload/support command wiring contracts; declarations/fields: `butaneFile` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 66–71 | Test harness fileMode — Installed provisioning payload/support command wiring contracts; declarations/fields: `fileMode` |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 73–100 | Assert Provisioning Minimal And Extension Profiles Are Distinct; declarations/fields: `TestProvisioningMinimalAndExtensionProfilesAreDistinct` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 102–118 | Assert Provisioning Private Modes And Plaintext Are Rejected Before Output; declarations/fields: `TestProvisioningPrivateModesAndPlaintextAreRejectedBeforeOutput` |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 120–142 | Assert Provisioning Host Key Contents Never Become Process Arguments; declarations/fields: `TestProvisioningHostKeyContentsNeverBecomeProcessArguments` |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 144–153 | Assert Provisioning Public Bootstrap Has No Identity Or Automatic Reboot; declarations/fields: `TestProvisioningPublicBootstrapHasNoIdentityOrAutomaticReboot` |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 155–169 | Assert Provisioning Installed Console Welcome Runs Before Login; declarations/fields: `TestProvisioningInstalledConsoleWelcomeRunsBeforeLogin` |
| [D11](../../slices/release-and-installation.md#d11-host-installation-and-payload-application) / active | 171–191 | Assert Provisioning Installed System Defaults To Password SSH; declarations/fields: `TestProvisioningInstalledSystemDefaultsToPasswordSSH` |
| [D06](../../slices/release-and-installation.md#d06-installed-qualification) / active | 193–202 | Assert outside-only acceptance/VM tools are excluded from appliance commands; declarations/fields: `TestOutsideSupportToolsAreNotApplianceCommands` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 203–215 | Assert production export uses local OCI archives and no implicit publication; declarations/fields: `TestOutsideSupportToolsAreNotApplianceCommands` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 217–225 | Assert Outside Boot Binds Browser Services Without Quadlet Enable; declarations/fields: `TestOutsideBootBindsBrowserServicesWithoutQuadletEnable` |

<a id="coverage-89796f4162bc"></a>

## [tests/build/project_account_test.go](../../../../../tests/build/project_account_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1–26, 31–51, 69–70, 90, 104, 111, 118, 128, 135, 154, 181, 190, 201, 214, 225, 245, 264, 283, 297, 335 | Declarations/fixtures and integration for Project human-account provisioning and explicit SSH key application |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 27–30 | Test harness accountBinary — Project human-account provisioning and explicit SSH key application; declarations/fields: `accountBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 52–68 | Test harness newAccountEnv — Project human-account provisioning and explicit SSH key application; declarations/fields: `newAccountEnv` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 71–89 | Test harness run — Project human-account provisioning and explicit SSH key application; declarations/fields: `run` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 91–103 | Test harness commands — Project human-account provisioning and explicit SSH key application; declarations/fields: `commands` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 105–110 | Test harness read — Project human-account provisioning and explicit SSH key application; declarations/fields: `read` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 112–117 | Test harness mode — Project human-account provisioning and explicit SSH key application; declarations/fields: `mode` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 119–127 | Test harness accountDoc — Project human-account provisioning and explicit SSH key application; declarations/fields: `accountDoc` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 129–134 | Test harness requireAccountFailure — Project human-account provisioning and explicit SSH key application; declarations/fields: `requireAccountFailure` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 136–153 | Assert Account Only Provisions Locked Home Marker Shared And Empty Keyfile; declarations/fields: `TestAccountOnlyProvisionsLockedHomeMarkerSharedAndEmptyKeyfile` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 155–180 | Assert Account Selected Keys And Creation Owner Privilege Remain Real; declarations/fields: `TestAccountSelectedKeysAndCreationOwnerPrivilegeRemainReal` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 182–189 | Assert Account Only Retry Never Erases Existing Keys; declarations/fields: `TestAccountOnlyRetryNeverErasesExistingKeys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 191–200 | Assert Account Join Is Not Key Apply And Preserves Drift; declarations/fields: `TestAccountJoinIsNotKeyApplyAndPreservesDrift` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 202–213 | Assert Account Occupied Inputs And Unassociated Users Refuse Before Commands; declarations/fields: `TestAccountOccupiedInputsAndUnassociatedUsersRefuseBeforeCommands` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 215–224 | Assert Account Key Symlink Is Not Followed; declarations/fields: `TestAccountKeySymlinkIsNotFollowed` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 226–244 | Assert Account Validation Before Native Effects; declarations/fields: `TestAccountValidationBeforeNativeEffects` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 246–263 | Assert Account Lock Contention Refuses Before Observation Or Commands; declarations/fields: `TestAccountLockContentionRefusesBeforeObservationOrCommands` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 265–282 | Assert Account Lock Covers File Durability And Account Commands; declarations/fields: `TestAccountLockCoversFileDurabilityAndAccountCommands` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 284–296 | Assert Account Failed Provisioning Releases Lock Without Removing Partial Files; declarations/fields: `TestAccountFailedProvisioningReleasesLockWithoutRemovingPartialFiles` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 298–334 | Assert Account Stdin Contract Bounds; declarations/fields: `TestAccountStdinContractBounds` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 336–357 | Assert Account Native Password SSHPolicy Is Not Relaxed; declarations/fields: `TestAccountNativePasswordSSHPolicyIsNotRelaxed` |

<a id="coverage-bca40bd658e0"></a>

<a id="testsbuildproject_factory_roles_testgo-1"></a>

## [tests/build/project_factory_roles_test.go](../../../../../tests/build/project_factory_roles_test.go)

Source/assertion inspection only; no suite or native scenario executed. project-factory-roles record/start/inspect/stop supervise checkout preparation (P07); run-as-role output belongs factory run execution (F08). Privileged administrator approval decision records are P09, distinct from native approved-snapshot materialization.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 1–38, 43, 53–54, 74–75, 85–86, 94–95, 110, 117, 130, 137, 148, 157, 164, 176–178, 189–190, 229, 248, 256, 287, 324, 395, 411, 435, 463 | Declarations/fixtures and integration for Project role accounts, preparation snapshot, supervision and hold contracts |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 39–42 | Test harness rolesBinary — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 44–52 | Test harness rolesSetup — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesSetup` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 55–73 | Test harness rolesRun — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesRun` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 76–84 | Test harness rolesOK — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesOK` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 87–93 | Test harness rolesRefused — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesRefused` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 96–109 | Test harness rolesDigest — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesDigest` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 111–116 | Test harness rolesMarshal — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesMarshal` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 118–129 | Test harness rolesApprove — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesApprove` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 131–136 | Test harness rolesFixture — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesFixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 138–147 | Test harness rolesRecord — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesRecord` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 149–156 | Test harness rolesOp — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesOp` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 158–163 | Test harness rolesMode — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesMode` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 165–175 | Test harness rolesWaitFile — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesWaitFile` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 179–188 | Test harness rolesDeadPGID — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesDeadPGID` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 191–228 | Test harness rolesGroupAlive — Project role accounts, preparation snapshot, supervision and hold contracts; declarations/fields: `rolesGroupAlive` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 230–247 | Assert Roles Ensure Provisions Locked Roles Without Extra Groups; declarations/fields: `TestRolesEnsureProvisionsLockedRolesWithoutExtraGroups` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 249–255 | Assert Roles Ensure Refuses Interactive Or Grouped Accounts; declarations/fields: `TestRolesEnsureRefusesInteractiveOrGroupedAccounts` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 257–286 | Assert Roles Approve Writes Protected Snapshot And Verifies Bundle; declarations/fields: `TestRolesApproveWritesProtectedSnapshotAndVerifiesBundle` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 288–323 | Assert Roles Approve Rejects Untrusted Inputs Before Effects; declarations/fields: `TestRolesApproveRejectsUntrustedInputsBeforeEffects` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 325–335 | Assert Roles Record Reports Waiting And Failed Phases; declarations/fields: `TestRolesRecordReportsWaitingAndFailedPhases` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 336, 346, 369, 382 | Declarations/fixtures and integration for Project role accounts, preparation snapshot, supervision and hold contracts |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 337–345 | Assert Roles Launcher Refusal Fails Without Start; declarations/fields: `TestRolesLauncherRefusalFailsWithoutStart` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 347–368 | Assert Roles Start Spawns Supervisor And Reports Running; declarations/fields: `TestRolesStartSpawnsSupervisorAndReportsRunning` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 370–381 | Assert Roles Dead Supervisor Reports Interrupted; declarations/fields: `TestRolesDeadSupervisorReportsInterrupted` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 383–394 | Assert Roles Stop Bars Unknown Identity And Retires Known; declarations/fields: `TestRolesStopBarsUnknownIdentityAndRetiresKnown` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 396–410 | Assert Roles Hold Denies Approve And Release Needs Revision And Quiescence; declarations/fields: `TestRolesHoldDeniesApproveAndReleaseNeedsRevisionAndQuiescence` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 412–434 | Assert Roles Supervisor Liveness Tracks Process Group; declarations/fields: `TestRolesSupervisorLivenessTracksProcessGroup` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 436–462 | Assert Roles Run As Role Captures Bounded Output And Exit; declarations/fields: `TestRolesRunAsRoleCapturesBoundedOutputAndExit` |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 464–481 | Assert Roles Rejects Malformed Requests; declarations/fields: `TestRolesRejectsMalformedRequests` |

<a id="coverage-3d25499b3001"></a>

## [tests/build/project_foundation_test.go](../../../../../tests/build/project_foundation_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1–11, 31, 43 | Declarations/fixtures and integration for Project userspace foundation/compiled helpers and scoped native tool driver |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 12–30 | Assert Foundation Recipe Declares Native Development Foundation; declarations/fields: `TestFoundationRecipeDeclaresNativeDevelopmentFoundation` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 32–42 | Assert Foundation Recipe Stages Compiled Project Helpers; declarations/fields: `TestFoundationRecipeStagesCompiledProjectHelpers` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 44–67 | Assert Foundation Installed Probe Requires Scope Before Writes; declarations/fields: `TestFoundationInstalledProbeRequiresScopeBeforeWrites` |

<a id="coverage-5d2f753c73ed"></a>

## [tests/build/project_runtime_test.go](../../../../../tests/build/project_runtime_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 1–11, 37, 47, 66, 90, 104, 111, 124 | Declarations/fixtures and integration for Project runtime/service/socket/secret/tool wiring |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 12–36 | Test harness parseUnit — Project runtime/service/socket/secret/tool wiring; declarations/fields: `parseUnit` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 38–46 | Assert Runtime Creation Image Identity Has Recipe And OSGuard; declarations/fields: `TestRuntimeCreationImageIdentityHasRecipeAndOSGuard` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 48–65 | Assert Runtime Service Uses Local Engine And Activation FD; declarations/fields: `TestRuntimeServiceUsesLocalEngineAndActivationFD` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 67–89 | Assert Runtime Socket Is Project Admin Only; declarations/fields: `TestRuntimeSocketIsProjectAdminOnly` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 91–103 | Assert Runtime Only Network Sysctl Subtree Is Rebound; declarations/fields: `TestRuntimeOnlyNetworkSysctlSubtreeIsRebound` |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 105–110 | Assert Runtime Shared Tools Uses Live Isolation Address; declarations/fields: `TestRuntimeSharedToolsUsesLiveIsolationAddress` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 112–123 | Assert Runtime Compose Database Image Is Digest Pinned; declarations/fields: `TestRuntimeComposeDatabaseImageIsDigestPinned` |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 125–132 | Assert Runtime Compose Uses Native Secret Not Password Argv; declarations/fields: `TestRuntimeComposeUsesNativeSecretNotPasswordArgv` |

<a id="coverage-d8e4f05bf26a"></a>

## [tests/build/sodaspaces_test.go](../../../../../tests/build/sodaspaces_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 1–25, 30, 35, 55, 67, 82, 101, 116, 125 | Declarations/fixtures and integration for Real staging recipe against synthetic build inputs and native origin/access guards |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 26–29 | Test harness testVMBinary — Real staging recipe against synthetic build inputs and native origin/access guards; declarations/fields: `testVMBinary` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 31–34 | Test harness stageBinary — Real staging recipe against synthetic build inputs and native origin/access guards; declarations/fields: `stageBinary` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 36–54 | Assert Sodaspaces Spaces Entry Is Packaged By The Extension; declarations/fields: `TestSodaspacesSpacesEntryIsPackagedByTheExtension` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 56–66 | Assert Sodaspaces VMWeb Tunnel Uses Only The Native Browser Origin; declarations/fields: `TestSodaspacesVMWebTunnelUsesOnlyTheNativeBrowserOrigin` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 68–81 | Assert Sodaspaces Access Probe Rejects Private Bad Request Before Native Commands; declarations/fields: `TestSodaspacesAccessProbeRejectsPrivateBadRequestBeforeNativeCommands` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 83–100 | Test harness copyTree — Real staging recipe against synthetic build inputs and native origin/access guards; declarations/fields: `copyTree` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 102–115 | Test harness chmodTree — Real staging recipe against synthetic build inputs and native origin/access guards; declarations/fields: `chmodTree` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 117–124 | Test harness checkMode — Real staging recipe against synthetic build inputs and native origin/access guards; declarations/fields: `checkMode` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 126–374 | Assert Sodaspaces Actual Stage Recipe With Synthetic Build Inputs; declarations/fields: `TestSodaspacesActualStageRecipeWithSyntheticBuildInputs` |

<a id="coverage-cc7cc5efe638"></a>

## [tests/build/source_checks_test.go](../../../../../tests/build/source_checks_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 1–30, 38–55, 77–83, 93, 116, 124, 141, 158, 172 | Declarations/fixtures and integration for Source suite sequencing/failure-stop and candidate gate integration |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 31–37 | Test harness gateStub — Source suite sequencing/failure-stop and candidate gate integration; declarations/fields: `gateStub` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 56–76 | Test harness newSourceFixture — Source suite sequencing/failure-stop and candidate gate integration; declarations/fields: `newSourceFixture` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 84–92 | Test harness run — Source suite sequencing/failure-stop and candidate gate integration; declarations/fields: `run` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 94–115 | Test harness commands — Source suite sequencing/failure-stop and candidate gate integration; declarations/fields: `commands` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 117–123 | Test harness commandNames — Source suite sequencing/failure-stop and candidate gate integration; declarations/fields: `commandNames` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 125–140 | Test harness equalCommands — Source suite sequencing/failure-stop and candidate gate integration; declarations/fields: `equalCommands` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 142–157 | Assert Source Commands Work Without AStage Or Git Checkout; declarations/fields: `TestSourceCommandsWorkWithoutAStageOrGitCheckout` |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 159–171 | Assert Source Each Failure Stops Remaining Suites; declarations/fields: `TestSourceEachFailureStopsRemainingSuites` |
| [D05](../../slices/release-and-installation.md#d05-artifact-verification) / active | 173–198 | Assert Source Native Gate Still Surrounds Shared Source Checks; declarations/fields: `TestSourceNativeGateStillSurroundsSharedSourceChecks` |

<a id="coverage-5375d7279fb9"></a>

## [tests/build/tailnet_image_test.go](../../../../../tests/build/tailnet_image_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1–10, 17, 27, 34 | Declarations/fixtures and integration for Live Tailnet build input resolution and observed-version recording |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 11–16 | Assert Tailnet No Stored Pins; declarations/fields: `TestTailnetNoStoredPins` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 18–26 | Assert Tailnet Recipe Floats On Build Args; declarations/fields: `TestTailnetRecipeFloatsOnBuildArgs` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 28–33 | Assert Tailnet Build Wires Live Inputs Without Lock; declarations/fields: `TestTailnetBuildWiresLiveInputsWithoutLock` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 35–39 | Assert Tailnet Observed Versions Recorded Without Gate; declarations/fields: `TestTailnetObservedVersionsRecordedWithoutGate` |

<a id="coverage-67c6456606a2"></a>

## [tests/build/terminal_assets_test.go](../../../../../tests/build/terminal_assets_test.go)

Source/assertion inspection only; no suite or native scenario executed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 1–8, 23, 54 | Declarations/fixtures and integration for Exact local terminal asset lock, preparation and shipping checks |
| [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) / active | 9–22 | Test harness packageScripts — Exact local terminal asset lock, preparation and shipping checks; declarations/fields: `packageScripts` |
| [D03](../../slices/release-and-installation.md#d03-candidate-production) / active | 24–53 | Assert Terminal Assets Browser Build Prepares Locked Renderer; declarations/fields: `TestTerminalAssetsBrowserBuildPreparesLockedRenderer` |
| [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) / active | 55–81 | Assert Terminal Assets Shipping Lock Has Only Exact Local Renderer Files; declarations/fields: `TestTerminalAssetsShippingLockHasOnlyExactLocalRendererFiles` |


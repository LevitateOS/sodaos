# Backend tailnet

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-bc084df91e04"></a>
<a id="internaltailnetcontrolgo-1"></a>

## [internal/tailnet/control.go](../../../../../internal/tailnet/control.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–77, 102–457, 473–628; whole file; (declaration group); Control; NewControl; NewProjectControl; boundedBody; boundedBody.Read; boundedOutput; boundedOutput.Write; controlCommand; RunNative; Control.request; nativePeer; nativeStatus; nativePrefs; addresses; peerView; haveNodeKeyOmitted; requiredNativeField; nativeObject; validateNativeBackendState; Control.fetchNativeStatus; Control.fetchNativePrefs; populateHostPreferences; applySelfPeer; populatePeers; computeHostRevision; Control.observe; validTailscaleAuthURL; authenticationURL; Control.Settings; Control.HostAction; Control.executeHostAction; Control.executeSignin; decodeUpNotifications; Control.executeLogout; findAvailableExitNode; Control.executeExitNode; Control.executeAdvertiseExitNode; Control.executeRefreshForgejo; Control.readbackHostAction; verifyHostActionOutcome; verifyExitNode; boolString; Control.run | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 45 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 78–101, 458–472, 629–643; boundedProviderTransport, validOAuthTokenRequest, boundedProviderTransport.RoundTrip, Control.lockPolicy, Control.checkCredential, Control.Enrollment | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Enrollment credential provider boundary and policy lock/use; declarations/fields: `boundedProviderTransport`, `validOAuthTokenRequest`, `boundedProviderTransport.RoundTrip`, `Control.lockPolicy`, `Control.checkCredential`, `Control.Enrollment` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 644–651; Control.Options, Control.Project | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Project enrollment policy selection interface; declarations/fields: `Control.Options`, `Control.Project` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-64d467a68567"></a>
<a id="internaltailnetcontrol_testgo-1"></a>

## [internal/tailnet/control_test.go](../../../../../internal/tailnet/control_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–254, 301–495; whole file; managementRoundTrip; managementRoundTrip.RoundTrip; controlResponse; nativeStatusFixture; nativePrefsFixture; controlFixture; TestTailnetHostProjectionAndPassiveReads; TestTailnetFreshDaemonOmitsFalseNodeKey; TestTailnetHostNodeKeyOptionalButStrict; TestTailnetHostUnavailableIsNotDisconnected; TestTailnetHostReauthPreservesPrefsAndValidatesURL; TestTailnetHostActionsConfirmAndSeparateFailedReadback; TestTailnetInitialLoginUsesBoundedUpNotReset; TestTailnetHostRevisionIncludesNativeIdentityWithoutReleaseVeto; TestTailnetExitSelectionPreservesRoutesAndLogoutHidesAuthURL; TestTailnetOfflineExitNodeAndUnconfirmedClear; TestTailnetInitialLoginNotificationsAndTimeoutKeepNativeAuthentication; TestTailnetBoundedOutputChild; TestTailnetCommandOutputBounds | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 20 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 255–300; TestTailnetCredentialCheckIsScopedBoundedAndNoRegistration | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Scoped bounded OAuth credential check has no registration effects; declarations/fields: `TestTailnetCredentialCheckIsScopedBoundedAndNoRegistration` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-72f1c1b425da"></a>

## [internal/tailnet/control_types.go](../../../../../internal/tailnet/control_types.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25, 30–57, 72–73, 76–154; whole file; (declaration group); validRevision; HostPreferences; Peer; HostView; SettingsView.Host, SettingsView.HostUnavailable; HostRequest; hasExtraHostFields; validateHostSigninAction; validateHostConfirmedAction; validateHostExitNode; validateHostExitNodeAction; validateHostAdvertiseAction; HostRequest.Validate; HostResult | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 16 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 26–29, 229–278, 281–288; ValidProject, ProjectOptions, ProjectRequest, validateProjectMutation, ProjectRequest.Validate, ProjectView | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Per-Project original network/binding selection DTO; declarations/fields: `ValidProject`, `ProjectOptions`, `ProjectRequest`, `validateProjectMutation`, `ProjectRequest.Validate`, `ProjectView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 58–71, 74–75, 155–228; EnrollmentView, SettingsView, EnrollmentRequest, validateEnrollmentTags, validateEnrollmentPolicy, validateEnrollmentMutation, hasEnrollmentPayload, validateEnrollmentToggle, EnrollmentRequest.Validate, EnrollmentResult | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Project enrollment policy DTO/validation; declarations/fields: `EnrollmentView`, `SettingsView`, `EnrollmentRequest`, `validateEnrollmentTags`, `validateEnrollmentPolicy`, `validateEnrollmentMutation`, `hasEnrollmentPayload`, `validateEnrollmentToggle`, `EnrollmentRequest.Validate`, `EnrollmentResult` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 279–280; ProjectView.Addresses, ProjectView.DNSName | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Observed native companion addresses and DNS identity; declarations/fields: `ProjectView.Addresses`, `ProjectView.DNSName` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-4562c1e954df"></a>

## [internal/tailnet/control_validation.go](../../../../../internal/tailnet/control_validation.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–4, 40–97, 108–128; whole file; validHostTailnet; validHostInventory; validHostBackendState; validHostPeers; validHostExitNode; HostView.Validate; HostResult.Validate | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 8 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 5–39, 98–107, 129–135; validEnrollmentIdentity, emptyEnrollmentBinding, idleEnrollmentFlags, EnrollmentView.Validate, SettingsView.Validate, EnrollmentResult.Validate | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Enrollment policy observation integrity; declarations/fields: `validEnrollmentIdentity`, `emptyEnrollmentBinding`, `idleEnrollmentFlags`, `EnrollmentView.Validate`, `SettingsView.Validate`, `EnrollmentResult.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 136–207; validProjectOptionsIdentity, ProjectOptions.Validate, validProjectAvailability, validProjectIdentity, validProjectPersistence, validDisconnectedProjectState, validConnectedProjectState, validProjectRuntime, ProjectView.Validate | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Project policy/original binding observation integrity; declarations/fields: `validProjectOptionsIdentity`, `ProjectOptions.Validate`, `validProjectAvailability`, `validProjectIdentity`, `validProjectPersistence`, `validDisconnectedProjectState`, `validConnectedProjectState`, `validProjectRuntime`, `ProjectView.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-5d74b2a27b1e"></a>

## [internal/tailnet/enrollment.go](../../../../../internal/tailnet/enrollment.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–26, 36–182; whole file; projectKeyLifetime; authKeyPattern; keyTransport; validKeyURL; validKeyRequest; keyResponseBody; validKeyCreateCapabilities; keyTransport.RoundTrip; tokenFromEnrollment; validAuthKeyIdentity; validAuthKeyCapabilities; validAuthKeyLifetime; validCreatedAuthKey; Control.createProjectAuthKey; Control.projectKey | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 16 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 27–35, 183–252; RunTarget, RunTarget.valid, admitEnrollRun, consumeEnrollKey, confirmEnrollSubmission, Control.enrollLocked, Control.EnrollRun | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Per-incarnation one-use native companion enrollment; consumes separately owned enrollment policy; declarations/fields: `RunTarget`, `RunTarget.valid`, `admitEnrollRun`, `consumeEnrollKey`, `confirmEnrollSubmission`, `Control.enrollLocked`, `Control.EnrollRun` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-fa785a431b37"></a>

## [internal/tailnet/enrollment_recovery_test.go](../../../../../internal/tailnet/enrollment_recovery_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–12; whole file | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 13–59; TestFailedEnrollmentCanBeExplicitlyRetried | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Explicit same-incarnation native companion enrollment retry assertions; declarations/fields: `TestFailedEnrollmentCanBeExplicitlyRetried` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-35824d4c64a9"></a>

## [internal/tailnet/enrollment_test.go](../../../../../internal/tailnet/enrollment_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–152; whole file; keyFixture; keyResponse; TestProjectKeyUsesSelectedSDKAndExactPolicy; TestProjectKeyRefusesUnsafeOrAmbiguousProviderResult | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 5 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 153–309; runFixture, TestRunEnrollmentSerializesConsumersWithoutJournal, TestRunEnrollmentSerializesPolicyAndCancelsWaitingOff, TestRunEnrollmentFencesBindingIdentityAndUncertainty | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Native companion enrollment serialization/fence fixture assertions; declarations/fields: `runFixture`, `TestRunEnrollmentSerializesConsumersWithoutJournal`, `TestRunEnrollmentSerializesPolicyAndCancelsWaitingOff`, `TestRunEnrollmentFencesBindingIdentityAndUncertainty` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-181b94f8f7cb"></a>

## [internal/tailnet/policy.go](../../../../../internal/tailnet/policy.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–235; file scaffold; policyStore; newRevision; owned; openOwnedLibDir; policyParent; createPolicyDir; ensurePolicyDir; acquirePolicyLock; lock; read; refuseUnownedPolicy; writePolicyFile; syncPolicy; publish | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-f781e065cc03"></a>

## [internal/tailnet/policy_enrollment.go](../../../../../internal/tailnet/policy_enrollment.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–211; file scaffold; enrollmentPolicy; credential; load; view; enrollment; validateUpdateRequest; checkEnrollment; lockAndLoad; canRotateEnrollment; applySaveOrRotate; applyDefault; applyDisable; applyPolicyMutation; update | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 16 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-002214a2576a"></a>

## [internal/tailnet/policy_project.go](../../../../../internal/tailnet/policy_project.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–167; file scaffold; projectPolicy; validLoadedProject; loadProject; validateProjectRequest; lockAndLoadProject; bindingMismatch; validateProjectBinding; mutateProject; initialProjectStateAndOutcome; buildProjectView; project | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-af5f21b95c6e"></a>

## [internal/tailnet/policy_test.go](../../../../../internal/tailnet/policy_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–280, 309–326; whole file; policyFixture; enrollmentInput; acceptedCredential; TestTailnetPolicyReadsAndChecksAreNonMutating; TestTailnetUnsupportedDefaultHasNoFilesOrProviderEffects; TestTailnetPolicyRotationCASAndSecretProjection; TestTailnetPolicyDoesNotConvertOrDeleteRetainedCredentials; TestTailnetPolicyConcurrencyAndCancelledWaiter; TestTailnetPolicyRefusesUnsafeAndAmbiguousState; TestTailnetPublicationFailureDoesNotRollbackOrReplay; TestTailnetInputRefusesCredentialEndpointAndPolicyOverrides | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 12 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 281–308; TestTailnetProjectPolicyDisabledUntilRuntimeExists | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Per-Project original binding policy assertions; declarations/fields: `TestTailnetProjectPolicyDisabledUntilRuntimeExists` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-7e77ec8acead"></a>

## [internal/tailnet/project_runtime.go](../../../../../internal/tailnet/project_runtime.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–31; whole file; ProjectSelection; ProjectSelection.Validate | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; Record, DTO or interface contract ProjectSelection for Project Tailnet selection; declarations/fields: `ProjectSelection`; ProjectSelection.Validate — Project Tailnet selection; declarations/fields: `ProjectSelection.Validate` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 32–71; RunBinding, Control.RunBinding | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Native companion original network and incarnation binding; declarations/fields: `RunBinding`, `Control.RunBinding` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-be321241d74e"></a>

## [internal/tailnet/project_runtime_test.go](../../../../../internal/tailnet/project_runtime_test.go)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–150; whole file; runtimePolicy; TestProjectRuntimeSelectionAndIndependentOriginalBindings; TestProjectRuntimeClosedAdmissionHasNoReservation; TestProjectRuntimeEnableCASAndNoImplicitRetarget; TestProjectPolicyMissingIsOffWhileMalformedFailsSafely | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility; 6 named units assigned here; remaining selectors preserve each duty — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 151–241; TestProjectHasNodeUsesCurrentStateAndOptionalNodeKey, TestProjectStatusFreshDaemonOmitsFalseNodeKey, TestProjectStatusRequiresExactNetworkTagsAndNativePreferences | [N06](../../slices/networking.md#n06-project-companion-lifecycle) | retained | Native companion status and original binding assertions; declarations/fields: `TestProjectHasNodeUsesCurrentStateAndOptionalNodeKey`, `TestProjectStatusFreshDaemonOmitsFalseNodeKey`, `TestProjectStatusRequiresExactNetworkTagsAndNativePreferences` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |

<a id="coverage-d1e1e5018762"></a>

## [internal/tailnet/project_status.go](../../../../../internal/tailnet/project_status.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–136; file scaffold; projectNativeStatus; projectNativePrefs; parseProjectStatus; invalidProjectPrefs; validateProjectPreferences; matchProjectSelf; matchProjectBinding; resolveProjectPeer; ProjectStatus; ProjectHasNode | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 11 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c7881f0d96fd"></a>

## [internal/tailnet/tailnet.go](../../../../../internal/tailnet/tailnet.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–175; file scaffold; DefaultCLI; Status; Endpoint; Client; Options; New; URLHost; statusDocument; parseStatus; CanonicalMagicDNSName; validLabel; invalidLabelCharacter | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-28913ee1a931"></a>

## [internal/tailnet/tailnet_test.go](../../../../../internal/tailnet/tailnet_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–119; file scaffold; statusClient; TestClientReadsCanonicalMagicDNSIdentity; TestClientReadsTailnetEndpoint; TestEndpointDoesNotAdvertiseUnavailableAccess; TestStatusRejectsInvalidMagicDNSIdentity; TestStatusReportsUnavailableCLI; TestEndpointUsesIPv4WhenMagicDNSIsDisabled; TestStatusRejectsMalformedOutputAndPreservesAuthPending; TestStatusCommandFailureAndCancellation | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 10 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

# Backend tailnet

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-bc084df91e04"></a>

<a id="internaltailnetcontrolgo-1"></a>

## [internal/tailnet/control.go](../../../../../internal/tailnet/control.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–26 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 27–31 | Declared identifiers/bounds (declaration group) for Host Tailnet control; declarations/fields: `(declaration group)` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 32–39 | Record, DTO or interface contract Control for Host Tailnet control; declarations/fields: `Control` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 40–52 | NewControl — Host Tailnet control; declarations/fields: `NewControl` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 53–60 | NewProjectControl — Host Tailnet control; declarations/fields: `NewProjectControl` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 61–65 | Record, DTO or interface contract boundedBody for Host Tailnet control; declarations/fields: `boundedBody` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 66–77 | boundedBody.Read — Host Tailnet control; declarations/fields: `boundedBody.Read` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 78–101, 458–472, 629–643 | Enrollment credential provider boundary and policy lock/use; declarations/fields: `boundedProviderTransport`, `validOAuthTokenRequest`, `boundedProviderTransport.RoundTrip`, `Control.lockPolicy`, `Control.checkCredential`, `Control.Enrollment` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 102–103 | Record, DTO or interface contract boundedOutput for Host Tailnet control; declarations/fields: `boundedOutput` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 104–110 | boundedOutput.Write — Host Tailnet control; declarations/fields: `boundedOutput.Write` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 111–116 | controlCommand — Host Tailnet control; declarations/fields: `controlCommand` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 117–129 | RunNative — Host Tailnet control; declarations/fields: `RunNative` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 130–151 | Control.request — Host Tailnet control; declarations/fields: `Control.request` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 152–159 | Record, DTO or interface contract nativePeer for Host Tailnet control; declarations/fields: `nativePeer` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 160–171 | Record, DTO or interface contract nativeStatus for Host Tailnet control; declarations/fields: `nativeStatus` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 172–179 | Record, DTO or interface contract nativePrefs for Host Tailnet control; declarations/fields: `nativePrefs` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 180–194 | addresses — Host Tailnet control; declarations/fields: `addresses` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 195–210 | peerView — Host Tailnet control; declarations/fields: `peerView` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 211–225 | haveNodeKeyOmitted — Host Tailnet control; declarations/fields: `haveNodeKeyOmitted` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 226–233 | requiredNativeField — Host Tailnet control; declarations/fields: `requiredNativeField` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 234–249 | nativeObject — Host Tailnet control; declarations/fields: `nativeObject` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 250–263 | validateNativeBackendState — Host Tailnet control; declarations/fields: `validateNativeBackendState` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 264–278 | Control.fetchNativeStatus — Host Tailnet control; declarations/fields: `Control.fetchNativeStatus` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 279–295 | Control.fetchNativePrefs — Host Tailnet control; declarations/fields: `Control.fetchNativePrefs` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 296–313 | populateHostPreferences — Host Tailnet control; declarations/fields: `populateHostPreferences` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 314–325 | applySelfPeer — Host Tailnet control; declarations/fields: `applySelfPeer` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 326–338 | populatePeers — Host Tailnet control; declarations/fields: `populatePeers` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 339–355 | computeHostRevision — Host Tailnet control; declarations/fields: `computeHostRevision` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 356–396 | Control.observe — Host Tailnet control; declarations/fields: `Control.observe` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 397–400 | validTailscaleAuthURL — Host Tailnet control; declarations/fields: `validTailscaleAuthURL` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 401–411 | authenticationURL — Host Tailnet control; declarations/fields: `authenticationURL` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 412–426 | Control.Settings — Host Tailnet control; declarations/fields: `Control.Settings` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 427–457 | Control.HostAction — Host Tailnet control; declarations/fields: `Control.HostAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 473–489 | Control.executeHostAction — Host Tailnet control; declarations/fields: `Control.executeHostAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 490–507 | Control.executeSignin — Host Tailnet control; declarations/fields: `Control.executeSignin` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 508–525 | decodeUpNotifications — Host Tailnet control; declarations/fields: `decodeUpNotifications` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 526–530 | Control.executeLogout — Host Tailnet control; declarations/fields: `Control.executeLogout` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 531–543 | findAvailableExitNode — Host Tailnet control; declarations/fields: `findAvailableExitNode` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 544–556 | Control.executeExitNode — Host Tailnet control; declarations/fields: `Control.executeExitNode` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 557–560 | Control.executeAdvertiseExitNode — Host Tailnet control; declarations/fields: `Control.executeAdvertiseExitNode` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 561–565 | Control.executeRefreshForgejo — Host Tailnet control; declarations/fields: `Control.executeRefreshForgejo` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 566–590 | Control.readbackHostAction — Host Tailnet control; declarations/fields: `Control.readbackHostAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 591–605 | verifyHostActionOutcome — Host Tailnet control; declarations/fields: `verifyHostActionOutcome` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 606–616 | verifyExitNode — Host Tailnet control; declarations/fields: `verifyExitNode` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 617–623 | boolString — Host Tailnet control; declarations/fields: `boolString` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 624–628 | Control.run — Host Tailnet control; declarations/fields: `Control.run` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 644–651 | Project enrollment policy selection interface; declarations/fields: `Control.Options`, `Control.Project` |

<a id="coverage-64d467a68567"></a>

<a id="internaltailnetcontrol_testgo-1"></a>

## [internal/tailnet/control_test.go](../../../../../internal/tailnet/control_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 18–19 | Fixture/protocol support managementRoundTrip; declarations/fields: `managementRoundTrip` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 20 | Fixture/protocol support managementRoundTrip.RoundTrip; declarations/fields: `managementRoundTrip.RoundTrip` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 21–23 | Fixture/protocol support controlResponse; declarations/fields: `controlResponse` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 24–26 | Fixture/protocol support nativeStatusFixture; declarations/fields: `nativeStatusFixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 27–29 | Fixture/protocol support nativePrefsFixture; declarations/fields: `nativePrefsFixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 30–50 | Fixture/protocol support controlFixture: unexpected native mutation; declarations/fields: `controlFixture` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 51–68 | Assertions TestTailnetHostProjectionAndPassiveReads: private native state projected; declarations/fields: `TestTailnetHostProjectionAndPassiveReads` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 69–96 | Assertions TestTailnetFreshDaemonOmitsFalseNodeKey: fresh observation dispatched a mutation; declarations/fields: `TestTailnetFreshDaemonOmitsFalseNodeKey` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 97–115 | Assertions TestTailnetHostNodeKeyOptionalButStrict: running without a valid true node key was accepted; declarations/fields: `TestTailnetHostNodeKeyOptionalButStrict` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 116–145 | Assertions TestTailnetHostUnavailableIsNotDisconnected; declarations/fields: `TestTailnetHostUnavailableIsNotDisconnected` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 146–182 | Assertions TestTailnetHostReauthPreservesPrefsAndValidatesURL: unrelated prefs rewritten; declarations/fields: `TestTailnetHostReauthPreservesPrefsAndValidatesURL` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 183–227 | Assertions TestTailnetHostActionsConfirmAndSeparateFailedReadback: unexpected command; declarations/fields: `TestTailnetHostActionsConfirmAndSeparateFailedReadback` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 228–254 | Assertions TestTailnetInitialLoginUsesBoundedUpNotReset: unbounded CLI; declarations/fields: `TestTailnetInitialLoginUsesBoundedUpNotReset` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 255–300 | Scoped bounded OAuth credential check has no registration effects; declarations/fields: `TestTailnetCredentialCheckIsScopedBoundedAndNoRegistration` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 301–331 | Assertions TestTailnetHostRevisionIncludesNativeIdentityWithoutReleaseVeto: replacement did not retire revision; declarations/fields: `TestTailnetHostRevisionIncludesNativeIdentityWithoutReleaseVeto` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 332–391 | Assertions TestTailnetExitSelectionPreservesRoutesAndLogoutHidesAuthURL: unexpected logout; declarations/fields: `TestTailnetExitSelectionPreservesRoutesAndLogoutHidesAuthURL` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 392–432 | Assertions TestTailnetOfflineExitNodeAndUnconfirmedClear; declarations/fields: `TestTailnetOfflineExitNodeAndUnconfirmedClear` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 433–480 | Assertions TestTailnetInitialLoginNotificationsAndTimeoutKeepNativeAuthentication: unexpected command; declarations/fields: `TestTailnetInitialLoginNotificationsAndTimeoutKeepNativeAuthentication` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 481–487 | Assertions TestTailnetBoundedOutputChild; declarations/fields: `TestTailnetBoundedOutputChild` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 488–495 | Assertions TestTailnetCommandOutputBounds: oversized command output accepted; declarations/fields: `TestTailnetCommandOutputBounds` |

<a id="coverage-72f1c1b425da"></a>

## [internal/tailnet/control_types.go](../../../../../internal/tailnet/control_types.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 11–24 | Declared identifiers/bounds (declaration group) for Host Tailnet control; declarations/fields: `(declaration group)` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 25 | validRevision — Host Tailnet control; declarations/fields: `validRevision` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 26–29, 229–278, 281–288 | Per-Project original network/binding selection DTO; declarations/fields: `ValidProject`, `ProjectOptions`, `ProjectRequest`, `validateProjectMutation`, `ProjectRequest.Validate`, `ProjectView` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 30–36 | Record, DTO or interface contract HostPreferences for Host Tailnet control; declarations/fields: `HostPreferences` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 37–44 | Record, DTO or interface contract Peer for Host Tailnet control; declarations/fields: `Peer` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 45–57 | Record, DTO or interface contract HostView for Host Tailnet control; declarations/fields: `HostView` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 58–71, 74–75, 155–228 | Project enrollment policy DTO/validation; declarations/fields: `EnrollmentView`, `SettingsView`, `EnrollmentRequest`, `validateEnrollmentTags`, `validateEnrollmentPolicy`, `validateEnrollmentMutation`, `hasEnrollmentPayload`, `validateEnrollmentToggle`, `EnrollmentRequest.Validate`, `EnrollmentResult` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 72–73 | Host Tailnet observation/unavailability in aggregate settings; declarations/fields: `SettingsView.Host`, `SettingsView.HostUnavailable` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 76–84 | Record, DTO or interface contract HostRequest for Host Tailnet control; declarations/fields: `HostRequest` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 85–88 | hasExtraHostFields — Host Tailnet control; declarations/fields: `hasExtraHostFields` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 89–95 | validateHostSigninAction — Host Tailnet control; declarations/fields: `validateHostSigninAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 96–102 | validateHostConfirmedAction — Host Tailnet control; declarations/fields: `validateHostConfirmedAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 103–116 | validateHostExitNode — Host Tailnet control; declarations/fields: `validateHostExitNode` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 117–123 | validateHostExitNodeAction — Host Tailnet control; declarations/fields: `validateHostExitNodeAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 124–130 | validateHostAdvertiseAction — Host Tailnet control; declarations/fields: `validateHostAdvertiseAction` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 131–148 | HostRequest.Validate — Host Tailnet control; declarations/fields: `HostRequest.Validate` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 149–154 | Record, DTO or interface contract HostResult for Host Tailnet control; declarations/fields: `HostResult` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 279–280 | Observed native companion addresses and DNS identity; declarations/fields: `ProjectView.Addresses`, `ProjectView.DNSName` |

<a id="coverage-4562c1e954df"></a>

## [internal/tailnet/control_validation.go](../../../../../internal/tailnet/control_validation.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–4 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 5–39, 98–107, 129–135 | Enrollment policy observation integrity; declarations/fields: `validEnrollmentIdentity`, `emptyEnrollmentBinding`, `idleEnrollmentFlags`, `EnrollmentView.Validate`, `SettingsView.Validate`, `EnrollmentResult.Validate` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 40–46 | validHostTailnet — Host Tailnet control; declarations/fields: `validHostTailnet` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 47–50 | validHostInventory — Host Tailnet control; declarations/fields: `validHostInventory` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 51–58 | validHostBackendState — Host Tailnet control; declarations/fields: `validHostBackendState` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 59–75 | validHostPeers — Host Tailnet control; declarations/fields: `validHostPeers` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 76–87 | validHostExitNode — Host Tailnet control; declarations/fields: `validHostExitNode` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 88–97 | HostView.Validate — Host Tailnet control; declarations/fields: `HostView.Validate` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 108–128 | HostResult.Validate — Host Tailnet control; declarations/fields: `HostResult.Validate` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 136–207 | Project policy/original binding observation integrity; declarations/fields: `validProjectOptionsIdentity`, `ProjectOptions.Validate`, `validProjectAvailability`, `validProjectIdentity`, `validProjectPersistence`, `validDisconnectedProjectState`, `validConnectedProjectState`, `validProjectRuntime`, `ProjectView.Validate` |

<a id="coverage-5d74b2a27b1e"></a>

## [internal/tailnet/enrollment.go](../../../../../internal/tailnet/enrollment.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–20 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 21–22 | Declared identifiers/bounds projectKeyLifetime for Project enrollment policy; declarations/fields: `projectKeyLifetime` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 23–26 | Declared identifiers/bounds authKeyPattern for Project enrollment policy; declarations/fields: `authKeyPattern` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 27–35, 183–252 | Per-incarnation one-use native companion enrollment; consumes separately owned enrollment policy; declarations/fields: `RunTarget`, `RunTarget.valid`, `admitEnrollRun`, `consumeEnrollKey`, `confirmEnrollSubmission`, `Control.enrollLocked`, `Control.EnrollRun` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 36–41 | Record, DTO or interface contract keyTransport for Project enrollment policy; declarations/fields: `keyTransport` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 42–45 | validKeyURL — Project enrollment policy; declarations/fields: `validKeyURL` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 46–49 | validKeyRequest — Project enrollment policy; declarations/fields: `validKeyRequest` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 50–65 | keyResponseBody — Project enrollment policy; declarations/fields: `keyResponseBody` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 66–82 | validKeyCreateCapabilities — Project enrollment policy; declarations/fields: `validKeyCreateCapabilities` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 83–105 | keyTransport.RoundTrip — Project enrollment policy; declarations/fields: `keyTransport.RoundTrip` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 106–115 | tokenFromEnrollment — Project enrollment policy; declarations/fields: `tokenFromEnrollment` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 116–119 | validAuthKeyIdentity — Project enrollment policy; declarations/fields: `validAuthKeyIdentity` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 120–124 | validAuthKeyCapabilities — Project enrollment policy; declarations/fields: `validAuthKeyCapabilities` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 125–128 | validAuthKeyLifetime — Project enrollment policy; declarations/fields: `validAuthKeyLifetime` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 129–132 | validCreatedAuthKey — Project enrollment policy; declarations/fields: `validCreatedAuthKey` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 133–157 | Control.createProjectAuthKey — Project enrollment policy; declarations/fields: `Control.createProjectAuthKey` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 158–182 | Control.projectKey — Project enrollment policy; declarations/fields: `Control.projectKey` |

<a id="coverage-fa785a431b37"></a>

## [internal/tailnet/enrollment_recovery_test.go](../../../../../internal/tailnet/enrollment_recovery_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 13–59 | Explicit same-incarnation native companion enrollment retry assertions; declarations/fields: `TestFailedEnrollmentCanBeExplicitlyRetried` |

<a id="coverage-35824d4c64a9"></a>

## [internal/tailnet/enrollment_test.go](../../../../../internal/tailnet/enrollment_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 18–59 | Fixture/protocol support keyFixture: token has no operation deadline; declarations/fields: `keyFixture` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 60–63 | Fixture/protocol support keyResponse; declarations/fields: `keyResponse` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 64–88 | Assertions TestProjectKeyUsesSelectedSDKAndExactPolicy: key creation failed; declarations/fields: `TestProjectKeyUsesSelectedSDKAndExactPolicy` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 89–152 | Assertions TestProjectKeyRefusesUnsafeOrAmbiguousProviderResult: unsafe key or diagnostic accepted; declarations/fields: `TestProjectKeyRefusesUnsafeOrAmbiguousProviderResult` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 153–309 | Native companion enrollment serialization/fence fixture assertions; declarations/fields: `runFixture`, `TestRunEnrollmentSerializesConsumersWithoutJournal`, `TestRunEnrollmentSerializesPolicyAndCancelsWaitingOff`, `TestRunEnrollmentFencesBindingIdentityAndUncertainty` |

<a id="coverage-af5f21b95c6e"></a>

## [internal/tailnet/policy_test.go](../../../../../internal/tailnet/policy_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 16–20 | Fixture/protocol support policyFixture; declarations/fields: `policyFixture` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 21–24 | Fixture/protocol support enrollmentInput; declarations/fields: `enrollmentInput` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 25 | Fixture/protocol support acceptedCredential; declarations/fields: `acceptedCredential` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 26–42 | Assertions TestTailnetPolicyReadsAndChecksAreNonMutating: observation/check wrote state; declarations/fields: `TestTailnetPolicyReadsAndChecksAreNonMutating` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 43–54 | Assertions TestTailnetUnsupportedDefaultHasNoFilesOrProviderEffects: provider called; declarations/fields: `TestTailnetUnsupportedDefaultHasNoFilesOrProviderEffects` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 55–112 | Assertions TestTailnetPolicyRotationCASAndSecretProjection: stale save; declarations/fields: `TestTailnetPolicyRotationCASAndSecretProjection` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 113–160 | Assertions TestTailnetPolicyDoesNotConvertOrDeleteRetainedCredentials: retained secret altered; declarations/fields: `TestTailnetPolicyDoesNotConvertOrDeleteRetainedCredentials` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 161–204 | Assertions TestTailnetPolicyConcurrencyAndCancelledWaiter: cancelled waiter acquired lock; declarations/fields: `TestTailnetPolicyConcurrencyAndCancelledWaiter` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 205–260 | Assertions TestTailnetPolicyRefusesUnsafeAndAmbiguousState: fixture decode; declarations/fields: `TestTailnetPolicyRefusesUnsafeAndAmbiguousState` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 261–280 | Assertions TestTailnetPublicationFailureDoesNotRollbackOrReplay: publication was rolled back; declarations/fields: `TestTailnetPublicationFailureDoesNotRollbackOrReplay` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 281–308 | Per-Project original binding policy assertions; declarations/fields: `TestTailnetProjectPolicyDisabledUntilRuntimeExists` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 309–326 | Assertions TestTailnetInputRefusesCredentialEndpointAndPolicyOverrides: invalid credential/policy accepted; declarations/fields: `TestTailnetInputRefusesCredentialEndpointAndPolicyOverrides` |

<a id="coverage-7e77ec8acead"></a>

## [internal/tailnet/project_runtime.go](../../../../../internal/tailnet/project_runtime.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 1–11 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 12–17 | Record, DTO or interface contract ProjectSelection for Project Tailnet selection; declarations/fields: `ProjectSelection` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 18–31 | ProjectSelection.Validate — Project Tailnet selection; declarations/fields: `ProjectSelection.Validate` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 32–71 | Native companion original network and incarnation binding; declarations/fields: `RunBinding`, `Control.RunBinding` |

<a id="coverage-be321241d74e"></a>

## [internal/tailnet/project_runtime_test.go](../../../../../internal/tailnet/project_runtime_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 13–22 | Fixture/protocol support runtimePolicy; declarations/fields: `runtimePolicy` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 23–63 | Assertions TestProjectRuntimeSelectionAndIndependentOriginalBindings: default enrolled old project; declarations/fields: `TestProjectRuntimeSelectionAndIndependentOriginalBindings` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 64–82 | Assertions TestProjectRuntimeClosedAdmissionHasNoReservation: closed admission enabled project; declarations/fields: `TestProjectRuntimeClosedAdmissionHasNoReservation` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 83–113 | Assertions TestProjectRuntimeEnableCASAndNoImplicitRetarget: stale enable; declarations/fields: `TestProjectRuntimeEnableCASAndNoImplicitRetarget` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 114–150 | Assertions TestProjectPolicyMissingIsOffWhileMalformedFailsSafely: missing project policy is not Off; declarations/fields: `TestProjectPolicyMissingIsOffWhileMalformedFailsSafely` |
| [N06](../../slices/networking.md#n06-project-companion-lifecycle) / active | 151–241 | Native companion status and original binding assertions; declarations/fields: `TestProjectHasNodeUsesCurrentStateAndOptionalNodeKey`, `TestProjectStatusFreshDaemonOmitsFalseNodeKey`, `TestProjectStatusRequiresExactNetworkTagsAndNativePreferences` |


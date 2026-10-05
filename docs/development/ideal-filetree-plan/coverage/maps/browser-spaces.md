# Browser spaces

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-f710ab93aa2b"></a>

## [frontend/spaces/soda-identity-response.ts](../../../../../frontend/spaces/soda-identity-response.ts)

Browser projections only: connection availability, enrollment, grants and leases remain separate broker records.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–2, 36–44, 111–114 | Bounded representation parsing and encoding; declarations/fields: `text`, `identityID`, `items` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 3, 13–19, 45–48, 67–86 | Provider enrollment, owner consent and connection selection; declarations/fields: `ProviderID`, `Enrollment`, `provider`, `enrollmentView`, `enrollmentOrigin` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 4–12, 20–26, 49–66, 87–97 | Named delegation and derived connection availability; declarations/fields: `Connection`, `Grant`, `connectionView`, `availableConnectionView`, `grantView` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 27–35, 98–110 | Execution lease identity and admission projection; declarations/fields: `Lease`, `leaseView` |

<a id="coverage-896e9d27756b"></a>

## [frontend/spaces/soda-identity.ts](../../../../../frontend/spaces/soda-identity.ts)

Component context/busy/lifetime/render composition stays S02; specific decisions belong to Identity children. Browser credential UI does not own encrypted bytes.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1–17, 25–29, 36–125, 146–150, 305–344 | Mounted workspace/component lifetime, navigation and focus; declarations/fields: `properties`, `busy`, `message`, `actor`, `transport`, `project`, `retired`, `contextSerial` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 18–19, 21, 23–24, 30–31, 33, 35, 126–135, 139–143, 176–193 | Named delegation and derived connection availability; declarations/fields: `connections`, `selected`, `grants`, `available`, `load`, `loadSelected`, `delegate` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 20, 32, 136–138, 151–164, 170–175, 237–252 | Provider enrollment, owner consent and connection selection; declarations/fields: `enrollment`, `load`, `connect`, `cancel`, `enrollmentView` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 22, 34, 144–145 | Execution lease identity and admission projection; declarations/fields: `leases`, `loadSelected` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 165–169, 194–205 | Connection/grant revocation and lease closure; declarations/fields: `disconnect`, `revokeGrant`, `endLease` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 206–236 | Native provider launch integration; declarations/fields: `launch`, `startCodex` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 253–275 | Connection/grant/lease controls compose existing child actions; declarations/fields: `selectedView` |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 276–304 | Project availability and native Codex launch presentation; declarations/fields: `projectView` |

<a id="coverage-e6523693d393"></a>

<a id="frontendspacessodaspaces-apits-1"></a>

## [frontend/spaces/sodaspaces-api.ts](../../../../../frontend/spaces/sodaspaces-api.ts)

 Mixed Detail/Space fields, validators and composition branches are mapped individually; aggregate representation is not a new authority owner.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–47, 851–858 | Bounded representation parsing and encoding |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 48–82, 135–180 | Creation profile and current runtime/OS readiness; declarations/fields: `CreationProfile`, `creationProfile`, `OSObservation`, `osObservation` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 83–96 | Repository selection, association and explicit creation; declarations/fields: `Environment`, `environmentResponse` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 97, 99–101, 103, 105–109, 113–114, 116–118, 125, 127–129, 131, 133–134 | Joined member/account/execution projection; declarations/fields: `Detail`, `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 98 | Readiness field in mixed Detail DTO; declarations/fields: `Detail.environment` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 102 | Native observation availability field; declarations/fields: `Detail.native_unavailable` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 104 | Current native runtime observation field; declarations/fields: `Detail.observed` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 110–111 | Exact Project/repository association admission; declarations/fields: `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 112 | Provisioning/readiness admission; declarations/fields: `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 115 | Native observation availability admission; declarations/fields: `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 119–124 | Current native running-state observation admission; declarations/fields: `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 126 | Composed association/readiness projection; P01 decoder dependency; declarations/fields: `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 130 | Native availability projection; declarations/fields: `detailResponse` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 132 | Native observation projection; declarations/fields: `detailResponse` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 181–235 | Development public keys and verified SSH connection; declarations/fields: `SavedKey`, `ProfileKeys`, `KeyPreview` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 236–298 | Exact terminal metadata, identity and state parsing; declarations/fields: `TerminalMetadata`, `terminalResponse` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 299–317 | Factory authority, intervention and run DTO projections; declarations/fields: `FactoryAuthority`, `FactoryControl`, `FactoryRun` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 318, 331 | Inventory composition of authority/control/run/network projections; declarations/fields: `Space`, `spaceFactoryRuns` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 319 | Host Tailnet observation in mixed Space DTO; declarations/fields: `Space.tailnet_state` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 320–322 | Factory projections in mixed Space DTO; declarations/fields: `Space.factory_authority`, `Space.factory_control`, `Space.factory_runs` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 323 | Repository identity in mixed Space DTO; P02 readiness dependency; declarations/fields: `Space.environment` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 324–327 | Human login and repository execution-authority fields; declarations/fields: `Space.login`, `Space.execution_allowed` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 328–329 | Native availability/runtime observation fields; declarations/fields: `Space.native_unavailable`, `Space.observed` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 330 | Native human terminal metadata field; declarations/fields: `Space.terminals` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 332–356 | Factory authority/intervention text and command identity; declarations/fields: `factoryAuthorityText`, `factoryControlText`, `factoryCommandId` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 357–361 | Bounded host Tailnet state projection; declarations/fields: `spaceNetwork` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 362–448 | Factory authority/intervention/run binding and outcome admission; declarations/fields: `spaceFactoryAuthority`, `spaceFactoryControl`, `spaceFactoryRuns` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 449–703 | Exact factory run/status/output/cursor projections; declarations/fields: `factoryRunStatusResponse`, `factoryStatusFrame`, `factoryOutputFrame` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 704–724, 730, 734–745, 754–756, 763–764, 771–802 | Authorized project/session inventory and completeness; declarations/fields: `spacesResponse` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 725–727 | Terminal collection shape and native count bound; declarations/fields: `spaceTerminals` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 728–729 | Member/repository authority gates on visible terminal collection; declarations/fields: `spaceTerminals` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 731–733 | Exact actor/Project/login binding of native terminal items; declarations/fields: `spaceTerminals` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 746–753 | Repository/Project binding and unique reservation identity; declarations/fields: `spaceItem` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 757 | Shared Detail authorization projection; P02 observations dependency; declarations/fields: `spaceItem` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 758 | Native terminal projections composed into inventory; declarations/fields: `spaceItem` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 759 | Host Tailnet projection composed into inventory; declarations/fields: `spaceItem` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 760–762 | Factory authority/control/run projection composition; declarations/fields: `spaceItem` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 765 | Tailnet field projection; declarations/fields: `spaceItem` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 766–768 | Factory fields projected into inventory; declarations/fields: `spaceItem` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 769 | Repository identity and reservation projection; declarations/fields: `spaceItem` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 770 | Native terminal fields projected into inventory; declarations/fields: `spaceItem` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 803–850 | Owned-repository picker and explicit Create admission projection; declarations/fields: `repositoryChoices` |

<a id="coverage-e99452619350"></a>

## [frontend/spaces/sodaspaces-environment-view.ts](../../../../../frontend/spaces/sodaspaces-environment-view.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–4, 70–74, 98–118, 139–171, 192–194 | Repository selection, association and explicit creation; declarations/fields: `osReleaseLine`, `EnvironmentPresentation`, `EnvironmentCommands`, `onSSHChange`, `joinAdmission`, `environmentBusy`, `environmentActions` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 5–69, 75–97 | Creation profile and current runtime/OS readiness; declarations/fields: `onProjectOSChange`, `createdProjectOS`, `projectOSOptionLabel`, `projectOSHelp`, `projectOSPicker`, `renderProjectOS`, `observedOSBody`, `renderOSObservation` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 119–138, 195–204 | Explicit human Join and member account presentation; declarations/fields: `joinKeys`, `renderEnvironment` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 172–181 | Explicit Create presentation; declarations/fields: `environmentActions` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 182–191 | Explicit Join presentation; declarations/fields: `environmentActions` |

<a id="coverage-f42521383923"></a>

<a id="frontendspacessodaspaces-factoryts-1"></a>

## [frontend/spaces/sodaspaces-factory.ts](../../../../../frontend/spaces/sodaspaces-factory.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 1–16 | Read-only factory watch imports and component declaration scaffolding |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 17–90, 120–188, 236–240, 305–318, 397–411, 436–467, 478–529 | Factory watch state and read-only presentation controls; declarations/fields: `FactoryWatchContext`, `renderer`, `SodaFactoryWatch`, `constructor`, `createRenderRoot`, `render`, `viewDisabled`, `canWatchView`, `watchTitle`, `watchPresentation`, `menuKey`, `watchCommands`, `watchFromControls`, `stopFromControls`, `hideFromControls`, `closeMenu`, `statusText`, `blockedWatch`, `endedRun`, `onPeerClosed`, `bindPeer`, `attachPeer`, `screenReady`, `setVisible`, `stop`, `started`, `optionalMatches`, `admitWatchIds`, `admitWatchContext`, `mountFactoryWatch` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 91–119, 189–218, 468–477 | Factory watch view binding, lifetime and disposal; no run stop/lease return; declarations/fields: `configure`, `disconnectedCallback`, `live`, `closeSocket`, `clearScreen`, `detach`, `invalidate`, `dispose` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 219–235, 319–333 | Authorized exact-run status observation and authority loss; declarations/fields: `json`, `authorityLost`, `inspectRun` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 241–304 | Read-only output renderer and view geometry; declarations/fields: `fitReady`, `screenPresent`, `canFit`, `resize`, `requiredToken`, `terminalTheme`, `awaitScreen`, `openScreen` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 334–396, 412–435 | Read-only factory attachment and bounded cursor/frame delivery; declarations/fields: `attachPayload`, `onPeerOpen`, `acceptStatus`, `acceptOutput`, `dispatchFrame`, `detachClosed`, `onPeerMessage`, `watch` |

<a id="coverage-51b813d093e0"></a>

## [frontend/spaces/sodaspaces-project-view.ts](../../../../../frontend/spaces/sodaspaces-project-view.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1–216 | Development public keys and verified SSH connection; declarations/fields: `Connection`, `renderConnection`, `KeyPresentation`, `KeyCommands`, `keyChanges`, `renderKeys`, `renderForgejoKeys` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 217–233 | Creation profile and current runtime/OS readiness; declarations/fields: `renderProjectStatus` |

<a id="coverage-75264836dfef"></a>

<a id="frontendspacessodaspaces-projectts-1"></a>

## [frontend/spaces/sodaspaces-project.ts](../../../../../frontend/spaces/sodaspaces-project.ts)

Generic component lifetime, tabs and render/request composition stays S02; its child requests are domain-owned. Static/instance fields mirror those same concerns.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1–49, 54–59, 118–122, 167–174, 193–198, 229–234, 238–240, 254, 265–266, 282–283, 285, 290–296, 299–363, 373–378, 389–409, 427–429, 441–478, 568–591, 603–626, 638–658, 699–711, 780–791, 799–816, 848–866, 890–897, 910–989, 1059–1071, 1081–1109, 1135–1165, 1180–1193, 1207–1230, 1280–1287, 1297–1304, 1315–1345, 1463–1552, 1560–1561, 1618–1621, 1644–1674, 1681–1687, 1864–1895 | Mounted workspace/component lifetime, navigation and focus; declarations/fields: `ProjectContext`, `rejected`, `views`, `View`, `RefreshPrior`, `viewTabLabel`, `lifecycleCaption`, `SodaProjectControls`, `properties`, `presentation`, `status`, `outcome`, `busy`, `stale`, `selected`, `binding`, `readController`, `lifetime`, `epoch`, `disposed`, `mutationPending`, `outcomeNeedsAttention`, `canRestore`, `constructor`, `createRenderRoot`, `configure`, `disconnectedCallback`, `blocked`, `running`, `busyAttr`, `active`, `select`, `shouldFocusTab`, `command`, `setPresentation`, `journeyBlocked`, `renderJourney`, `journeyUnavailableHeading`, `journeyUnavailableDescription`, `journeyUnavailableAction`, `renderJourneyUnavailable`, `executionDenied`, `journeyExistingKind`, `journeyExistingHeading`, `journeyExistingDescription`, `journeyExistingAction`, `journeyExistingHelper`, `journeyExistingFlags`, `renderJourneyExisting`, `renderStaleReloadNote`, `createDisabled`, `configureStatusHidden`, `pendingTone`, `shouldRenderJourney`, `sectionAriaBusy`, `sessionCaption`, `boundEnvironmentId`, `render`, `renderViewTabs`, `onReloadPage`, `reloadAdmitted`, `renderEnvironmentView`, `renderAccessView`, `copyBlocked`, `noteCopy`, `reset`, `invalidate`, `refreshBlocked`, `beginRefreshRead`, `refresh`, `admitCollection`, `loadEnvironmentCollection`, `refreshEmpty`, `tailnetOptionsFailed`, `environmentStatus`, `refreshExisting`, `refreshOptionalDetails`, `refreshErrorStatus`, `refreshFailed`, `finishRefresh`, `mutate`, `dispatchMutation`, `completeMutation`, `checkMutationResult`, `checkDeleteMutation`, `mutateReason`, `mutateErrorMessage`, `applyMutateError`, `finishMutation`, `dispose`, `mountProjectControls` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 50–53, 130–166, 223–225, 244–246, 275, 287, 592–602, 630–633, 898–900, 1305–1309, 1362–1382, 1583–1594, 1750–1770 | Explicit Project Start/Stop confirmation and outcome; declarations/fields: `Lifecycle`, `renderLifecycle`, `confirmLifecycle`, `lifecycle`, `stopConfirmed`, `journeyStartButton`, `journeyStopped`, `onStopConfirmed`, `environmentNeedsAdminStart`, `refreshLifecycleIfAdmin`, `checkLifecycleMutation`, `changeLifecycle` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 60–67, 123–129, 214–222, 226–228, 241–243, 247–249, 272–274, 276–281, 286, 288, 379–388, 887–889, 901–906, 990–1000, 1053–1058, 1072–1080, 1346–1361, 1383–1428, 1605–1617, 1622–1631, 1771–1863 | Development public keys and verified SSH connection; declarations/fields: `viewFromTabKey`, `publicKeyToken`, `saved`, `keyPreview`, `profileKeys`, `connection`, `draft`, `emptyConfirmed`, `tabKey`, `onUseSavedKeys`, `onDraft`, `onConfirmEmpty`, `renderForgejoKeyReview`, `selectForgejoKey`, `copyConnection`, `refreshSavedKeysIfStandard`, `refreshSavedKeys`, `shouldLoadConnection`, `admitConnectionPayload`, `validIPv4`, `refreshConnectionIfJoined`, `checkAccessKeysMutation`, `checkSaveKeyMutation`, `saveKey`, `profileKeysPageAdmitted`, `reviewProfileKeys`, `profileKeysFailed`, `reviewKeys`, `applyKeys` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 68–102 | Bounded representation parsing and encoding; declarations/fields: `sodaFetchInit`, `sodaErrorCode`, `mutationObject` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 103–113, 199–210, 235–237, 267–270, 284, 410–421, 659–664, 712–779, 796–798, 834–847, 1194–1206, 1235–1250, 1562–1578, 1675–1680 | Repository selection, association and explicit creation; declarations/fields: `admitRepositoryPart`, `repository`, `repositoryName`, `repositoryURL`, `environment`, `canCreate`, `createProject`, `configureReady`, `requestRepositoryChange`, `renderCreateAction`, `renderConfigureFeedback`, `renderJourneyConfigure`, `boundRepositoryId`, `renderRepositoryContext`, `applyRepositoryLabels`, `refreshCreateOptions`, `checkCreateMutation`, `noteUnconfirmedCreation` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 114–117, 211–213, 250–253, 271, 289, 297–298, 364–372, 430–440, 479–564, 634–637, 792–795, 1036–1052, 1166–1179, 1288–1296, 1579–1582, 1632–1643 | Explicit human Join and member account presentation; declarations/fields: `admitProjectLogin`, `detail`, `useSavedKeys`, `joinFailed`, `joinNeedsCheck`, `canJoin`, `journeyJoinReady`, `joinFailedHelper`, `joinFailedFeedback`, `renderJourneyJoinFailed`, `joinAction`, `joinFeedback`, `joinRefreshAction`, `renderJourneyJoin`, `journeyAccountReady`, `projectAccountCaption`, `joinEnvironment`, `refreshAccount`, `applyJoinRecovery`, `checkJoinMutation`, `joinErrorMessage` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 175, 177–178, 255, 257–258, 422–426, 668–698, 867–880, 907–909, 1001–1035, 1251–1279, 1688–1696 | Project enrollment policy/secret and admission controls; declarations/fields: `networkReview`, `networkOptions`, `networkEnabled`, `createTailnetBody`, `onJourneyNetworkEnabled`, `onCreateNetworkEnabled`, `onClearNetworkReview`, `renderConfigureNetwork`, `renderNetworkReviewNotice`, `renderCreateNetworkSelection`, `networkSummaryText`, `renderNetworkSummary`, `onNetworkConfirmed`, `renderTailnetSSH`, `renderNetworkView`, `networkReviewNeeded`, `networkEnabledAfterRefresh`, `refreshTailnetOptions`, `networkChangeBlocked` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 176, 179–180, 256, 259–260, 1429–1462, 1595–1604, 1697–1715 | Project Tailnet selection and revision-bound intent; declarations/fields: `network`, `networkConfirmed`, `networkNotice`, `shouldLoadProjectNetwork`, `refreshProjectNetworkIfEligible`, `refreshProjectNetwork`, `projectNetworkFailed`, `checkTailnetMutation`, `networkEnableBlocked`, `changeNetwork` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 181–192, 261–264, 565–567, 627–629, 665–667, 881–886, 1231–1234, 1310–1314, 1716–1720, 1724–1749 | Creation profile and current runtime/OS readiness; declarations/fields: `observedOS`, `osStatus`, `profiles`, `selectedProfile`, `journeyRuntimeUnavailable`, `journeyIncomplete`, `onSelectProfile`, `renderObservedOS`, `selectedCreateProfile`, `applyEnvironmentStatus`, `applyOSError`, `inspectOS` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 817–833 | Provider enrollment, owner consent and connection selection; declarations/fields: `journeyIdentityHidden`, `renderIdentity`, `identityPresentationReady` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1110–1134, 1721–1723 | Native extension context and browser contribution admission; declarations/fields: `requestHeaders`, `admitHttpFailure`, `api`, `authLost` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1553 | Mutation response branch; Repository selection, association and explicit creation; declarations/fields: `checkMutationResult` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1554 | Mutation response branch; Explicit human Join and member account presentation; declarations/fields: `checkMutationResult` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1555 | Mutation response branch; Explicit Project Start/Stop confirmation and outcome; declarations/fields: `checkMutationResult` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 1556 | Mutation response branch; Project Tailnet selection and revision-bound intent; declarations/fields: `checkMutationResult` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 1557–1559 | Mutation response branch; Development public keys and verified SSH connection; declarations/fields: `checkMutationResult` |

<a id="coverage-3b2eeb224758"></a>

## [frontend/spaces/sodaspaces-terminal-view.ts](../../../../../frontend/spaces/sodaspaces-terminal-view.ts)

Readonly rendering projections compose attach and End/Rename commands; native lifecycle authority remains S04.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–29, 69–111 | Interactive attachment, writer/input gating and rendering; declarations/fields: `TerminalPresentation`, `TerminalCommands`, `renderTerminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 30–68 | Human terminal Create/Rename/End request and outcome; declarations/fields: `actions`, `confirmation` |

<a id="coverage-df201745844a"></a>

<a id="frontendspacessodaspaces-terminalts-1"></a>

## [frontend/spaces/sodaspaces-terminal.ts](../../../../../frontend/spaces/sodaspaces-terminal.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–23, 28–35, 56–123, 160–165, 169–219, 225–228, 274–362, 372–378, 441–455, 509–611, 629–657, 674–699, 715–880, 899–907, 937–942 | Interactive attachment, writer/input gating and rendering; declarations/fields: `TerminalView`, `TerminalContext`, `SodaTerminal`, `constructor`, `viewDisabled`, `canConnectView`, `contextLabels`, `terminalPresentation`, `menuKey`, `terminalCommands`, `connectFromControls`, `closeMenu`, `live`, `publish`, `abortActionIfStale`, `closeSocket`, `clearScreen`, `detach`, `invalidate`, `json`, `authorityLost`, `actionCurrent`, `finishAction`, `resize`, `blockedConnect`, `hiddenConnect`, `canStartConnect`, `detachUnready`, `hiddenWhileAttaching`, `cancelIfHidden`, `inspectExisting`, `awaitScreen`, `requiredToken`, `terminalTheme`, `interceptControlFocus`, `openTerminal`, `attachPayload`, `onPeerOpen`, `acceptReady`, `acceptOutput`, `dispatchFrame`, `onPeerMessage`, `onPeerClosed`, `bindPeer`, `attachPeer`, `connect`, `shouldFocusScreen`, `screenReady`, `setVisible`, `focus`, `started`, `restore`, `disconnect`, `mountTerminal` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 24–27, 36–55, 124–126, 157–159, 166–168, 220–224, 239–273, 363–371, 379–428, 612–628, 661–673, 700–714, 881–898, 929–936, 944–964 | Human terminal Create/Rename/End request and outcome; declarations/fields: `Renderer`, `TerminalLocator`, `renderer`, `createRenderRoot`, `render`, `canEndView`, `endConfirmedTerminal`, `confirmEnd`, `cancelEnd`, `remember`, `confirmNativeEnd`, `endUnconfirmed`, `endBlocked`, `endTerminal`, `send`, `sendInput`, `reserveCreate`, `refreshAttachedMetadata`, `observe`, `setName`, `open`, `admitLocator` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 127–156, 229–238, 908–928, 943 | Mounted workspace/component lifetime, navigation and focus; declarations/fields: `configure`, `disconnectedCallback`, `workspaceCommand`, `dispose`, `admitMountContext` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 429–440, 456–508, 658–660 | Disposable views, layout, geometry and exact locators; declarations/fields: `canFit`, `geometryScreen`, `paneMinimum`, `publishMinimum`, `measureMinimum`, `measureIfCurrent` |

<a id="coverage-1e87c626d0ae"></a>

<a id="frontendspacessodaspaces-workspace-viewts-1"></a>

## [frontend/spaces/sodaspaces-workspace-view.ts](../../../../../frontend/spaces/sodaspaces-workspace-view.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1–81, 101–102, 193–200 | Mounted workspace/component lifetime, navigation and focus; declarations/fields: `renderWelcome`, `renderWorkspaceIntro`, `renderWelcomeSteps`, `search`, `renderMenu` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 82–100, 103–192 | Repository selection, association and explicit creation; declarations/fields: `RepositoryPickerView`, `RepositoryPickerActions`, `onRepositorySearch`, `onRepositoryQuery`, `repositoryBadge`, `continueCaption`, `emptyRepositories`, `repositoryChoice`, `repositoryResults`, `pickerLocked`, `repositoryPages`, `renderRepositoryPicker` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 201–244 | Disposable views, layout, geometry and exact locators; declarations/fields: `SessionTab`, `renderSessionTab` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 245–319 | Authorized project/session inventory and completeness; declarations/fields: `NavigationSession`, `renderProjectNavigation` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 320–374 | Factory run/output/watch presentation; declarations/fields: `FactoryRunRow`, `FactoryWatchSlot`, `renderFactoryRuns` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 375–473 | Human terminal Create/Rename/End request and outcome; declarations/fields: `renderRename`, `CreationPresentation`, `renderCreation` |

<a id="coverage-b7c59695b1e9"></a>

<a id="frontendspacessodaspaces-workspacets-1"></a>

## [frontend/spaces/sodaspaces-workspace.ts](../../../../../frontend/spaces/sodaspaces-workspace.ts)

Browser-owned copies/locators only; native terminal lifecycle and factory control authority remain elsewhere.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 1–69, 111–112, 143–186, 188, 200–205, 212–223, 236–238, 244–245, 248–251, 255–257, 259, 261, 265–267, 286–290, 292–293, 296, 298, 300, 302–304, 307–309, 314, 316–339, 343–415, 431–470, 481–495, 499–505, 517–526, 536–590, 598–629, 657–695, 699–714, 723–773, 797–804, 837–1015, 1020–1048, 1052–1079, 1253–1259, 1265–1289, 1296–1302, 1362–1385, 1445–1447, 1484–1560, 1700–1702, 1763–1767, 1782–1791, 1816–1846, 1935–2047, 2068–2080, 2087–2101, 2110–2118, 2182–2194, 2206–2210, 2217–2222, 2227–2236, 2280–2286, 2335–2337, 2352–2393, 2422–2429, 2455–2462, 2471–2509, 2543–2561, 2602–2604, 2612–2619, 2639–2692, 2699–2717, 2771–2780, 2856–2875, 2907–2916, 2931–2944, 2965–2968, 3044–3095 | Mounted workspace/component lifetime, navigation and focus; declarations/fields: `WorkspaceContext`, `validName`, `WorkspaceMeasurement`, `constructor`, `hostUpdated`, `hostDisconnected`, `retire`, `properties`, `status`, `busy`, `view`, `stale`, `search`, `thisPage`, `openingDrawer`, `key`, `name`, `reconnectRequired`, `binding`, `factory`, `slots`, `setupReturn`, `managementMode`, `request`, `epoch`, `disposed`, `actor`, `surfaceVisible`, `lifetime`, `width`, `height`, `invoker`, `createRenderRoot`, `activeSurface`, `selectedSpace`, `welcomeScreen`, `pageReadyForFirstTerminal`, `spaceReadyForFirstTerminal`, `projectRunnable`, `firstTerminal`, `inventoryRecovery`, `workspaceIntro`, `projectState`, `projectStatus`, `selected`, `compact`, `projection`, `configure`, `projectEventFromHost`, `pageProjectEvent`, `onProjectObserved`, `onProjectOperation`, `disconnectedCallback`, `updated`, `publishMinimum`, `measuredCell`, `measure`, `workspaceBlocked`, `navigationVisible`, `busyAttr`, `workspaceClasses`, `navigationEscape`, `setSearchFromEvent`, `setThisPageFromEvent`, `renderNativeToolbar`, `renderPageToolbar`, `hideListStatus`, `retryLoading`, `renderStatusBanners`, `hideWorkspaceBody`, `workspaceBodyClass`, `workspaceBodyStyle`, `navAriaLabel`, `hideNavigation`, `renderProjectsHeading`, `hideTerminalSearch`, `renderThisPageFilter`, `renderCreateAnotherProject`, `emptyFilterMessage`, `hideCanvas`, `inventoryHelper`, `renderInventoryRecovery`, `renderFirstTerminalIntro`, `hideManagement`, `renderWelcomeFooter`, `renderNavigation`, `renderCanvas`, `renderWorkspaceFrame`, `render`, `connectURL`, `sessionsButtonHidden`, `sessionsButtonLabel`, `toolbarProjectTitle`, `renderToolbarProject`, `hideNewTerminal`, `disableNewTerminal`, `newTerminalButtonClass`, `newTerminalButtonLabel`, `renderProjectSettingsButton`, `hideBackButton`, `backButtonLabel`, `renderBackButton`, `renderOpenInDrawerOption`, `nativeManagementTarget`, `renderNativeManagementOption`, `renderToolbarMenu`, `renderToolbar`, `searchAdmitted`, `searchCursor`, `searchRepositories`, `selectProject`, `visibleSlot`, `markViewed`, `projectName`, `metadataRowName`, `draftRowName`, `unnamedRow`, `rowName`, `rowMatchesQuery`, `rowTerminalId`, `rowDescription`, `rowDisabled`, `selectRow`, `rowNavItem`, `pageProjectNav`, `projectSubtitle`, `sessionRow`, `showMoveMenu`, `renderMoveMenu`, `dropEdgeStyle`, `onDropEdgeOver`, `onDropEdgeDrop`, `renderDropEdges`, `workspaceKey`, `workspaceClick`, `menuFocusOut`, `closeMenus`, `rememberFocus`, `restoreFocus`, `showSessions`, `back`, `defaultTerminalName`, `pageBlocksNewTerminal`, `newTerminalBlocked`, `newTerminal`, `live`, `api`, `applySlotMetadata`, `existingSlotMetadata`, `restoreIfNeeded`, `pageJourneyActive`, `leaveJourneyManagement`, `syncPageJourney`, `renderMoreProjects`, `drawerOpenBlocked`, `drawerStillCurrent`, `assignDrawer`, `openDrawerNavigation`, `openInDrawer`, `slotName`, `openSavedBlocked`, `openSavedSpace`, `shouldFocusOpened`, `restoreOpenedSlot`, `openSaved`, `identity`, `observationAdmitted`, `staleObservation`, `noteUnread`, `commandAdmitted`, `onTerminalCommand`, `addSlot`, `slotHidden`, `displaySlot`, `display`, `rectangle`, `capture`, `releasePointer`, `createAdmitted`, `createSpaceReady`, `openExistingBlocked`, `openExisting`, `defaultManagementMode`, `setVisible`, `invalidate`, `canRestore`, `dispose` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 70, 95–105, 268, 1576–1677 | Factory run/output/watch presentation; declarations/fields: `TerminalFactory`, `FactoryWatch`, `factoryWatchLimit`, `watches`, `factoryWatching`, `factoryRowDisabled`, `factorySection`, `toggleFactoryWatch`, `watchRun`, `unwatchRun`, `factoryWatchContext`, `displayFactoryWatch`, `displayFactoryWatches`, `onFactoryCommand` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 71–84, 90–94, 206–211, 246–247, 294–295, 299, 301, 305–306, 310–313, 315, 416–430, 506–516, 527–535, 630–635, 719–722, 774–796, 1016–1019, 1303–1361, 1678–1699, 1703–1762, 1768–1781, 1792–1815, 1847–1881, 2313–2334, 2338–2351, 2394–2421, 2463–2470, 2510–2542, 2620–2634, 2693–2698, 2718–2770, 2781–2855, 2917–2930 | Disposable views, layout, geometry and exact locators; declarations/fields: `Slot`, `PaneSession`, `layout`, `storageNotice`, `restored`, `storageLoaded`, `storageKey`, `canvasSize`, `workspaceWidth`, `cell`, `measurement`, `maximized`, `dragged`, `choosingPane`, `lastMinimum`, `locator`, `paneMinimum`, `tabHeight`, `geometryChanged`, `applyGeometry`, `recordCanvasGeometry`, `renderDrawerProjectionTabs`, `hideSidebarDivider`, `renderPaneDivider`, `hidePaneChrome`, `renderToggleSidebarOption`, `toggleSidebar`, `toggleMaximizedPane`, `consolidatePanes`, `showPaneSwitcher`, `onFocusPaneChange`, `renderPaneSwitcher`, `renderPaneLayoutExtras`, `renderPaneSplitHelp`, `renderPaneMenu`, `paneActions`, `paneTabKeys`, `paneSession`, `paneAriaOwns`, `onTabListDragOver`, `onTabListDrop`, `onTabDragStart`, `onTabDragEnd`, `onTabDrop`, `sessionTabProps`, `renderFocusedPaneActions`, `filterOverflowTabs`, `overflowTabLabel`, `renderTabOverflow`, `moveTargetName`, `moveToPane`, `moveBeforeTab`, `beforeTabName`, `emptyPaneAttached`, `renderEmptyPane`, `paneChrome`, `loadLayout`, `persist`, `drawerEntry`, `canRestorePane`, `restoreSelectedPane`, `restoreLocators`, `applyOpenLayout`, `onTerminalLocator`, `commitTerminalLocator`, `geometrySize`, `onTerminalGeometry`, `applySlotGeometry`, `arrange`, `focusPane`, `splitBlocked`, `emptyPaneMinimum`, `splitFits`, `canSplit`, `split`, `move`, `dragDivider`, `keyDivider`, `adjustDivider`, `resizeSidebar`, `sidebarKey`, `setSidebar`, `tabKey`, `hideSlot`, `existingOrNewEntry` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 85–89, 187, 191, 197–199, 233–235, 239–243, 258, 260, 262–264, 278, 297, 471–480, 696–698, 715–718, 1049–1051, 1386–1444, 1448–1483, 1561–1575, 2102–2109, 2119–2169, 2195–2205, 2223–2226, 2237–2279, 2287–2312, 2969–2976, 3096–3125 | Authorized project/session inventory and completeness; declarations/fields: `Row`, `SodaSpaces`, `complete`, `spaces`, `attentionOnly`, `nextAfter`, `observedAt`, `now`, `attentionTimer`, `available`, `pollAttention`, `shouldRefreshAttention`, `hideAttentionFilters`, `renderEmptySpaces`, `renderSpacesLink`, `rowAttentionReady`, `rowAttentionBlocked`, `rowAttention`, `attentionRows`, `nextAttention`, `filteredSpaces`, `rows`, `projectRows`, `readSpacesResponse`, `refreshBlocked`, `beginRefreshRead`, `collectionStatus`, `applySpacesCollection`, `appendSpacesCollection`, `refreshSlots`, `refreshJourneyManagement`, `refreshFailed`, `finishRefreshSuccess`, `refreshLive`, `refresh`, `loadMore`, `refreshMountedProject`, `mountSodaspaces` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 106–110, 224–232, 252–254, 291, 805–836, 1882–1934, 2048–2067, 2081–2086, 2605–2611, 2876–2906, 3000–3043 | Human terminal Create/Rename/End request and outcome; declarations/fields: `Creation`, `creation`, `creating`, `editing`, `renaming`, `onRenameDraft`, `onRenameCancel`, `onRenameSave`, `renameDisabled`, `renderRenameDialog`, `renderCreationDialog`, `creationEligible`, `creationExplanation`, `creationContext`, `creationDisabled`, `onCreationEnvironment`, `onCreationName`, `cancelCreation`, `creationForm`, `creationSpace`, `pickCreationSpace`, `focusCreationDialog`, `beginRename`, `openCreatedSlot`, `createTerminal`, `renameAdmitted`, `applyRenamedMetadata`, `rename` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 113–123, 189–190, 192–196, 269–277, 279–285, 340–342, 496–498, 591–597, 636–656, 1080–1252, 1260–1264, 1290–1295, 2211–2216, 2945–2964, 2977–2999 | Repository selection, association and explicit creation; declarations/fields: `drawerRepositoryPart`, `setup`, `project`, `repositoryQuery`, `repositoryResult`, `repositoryChoice`, `repositoryBusy`, `repositoryError`, `projects`, `repositoryRequest`, `repositoryCursors`, `setupScreen`, `onProjectChangeRepository`, `renderSetupHeading`, `setupBackDisabled`, `renderSetupTopbar`, `repositoryPrefix`, `clearRepositorySearch`, `setupPanelClass`, `setupIntroKind`, `setupIntroHeading`, `setupIntroDescription`, `setupIntroAction`, `setupIntroHelper`, `renderSetupBody`, `onRepositoryQuery`, `setupUnavailableHeading`, `setupUnavailableDescription`, `setupUnavailableAction`, `setupUnavailableHelper`, `renderSetupUnavailable`, `renderSetup`, `focusSetup`, `beginSetup`, `changeRepository`, `cancelSetup`, `applyRepositorySearch`, `failRepositorySearch`, `repositorySearchPath`, `configureProject`, `clearConfigureAfterCreate`, `managementAdmitted`, `mountProject`, `focusConfigureIfNeeded`, `showManagement`, `presentManagement` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 124–142 | Native extension context and browser contribution admission; declarations/fields: `sodaWorkspaceInit` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 2170–2181, 2562–2601, 2635–2638 | Interactive attachment, writer/input gating and rendering; declarations/fields: `slotLost`, `invalidateSlot`, `applyTerminalObservation`, `onTerminalObservation`, `onTerminalMetadata`, `onTerminalFocusIn` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 2430–2432 | Exact confirmed terminal End binding; declarations/fields: `confirmedEnd` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 2433–2443 | Remove ended terminal from current browser inventory; declarations/fields: `confirmedEnd` |
| [S03](../../slices/spaces-and-terminals.md#s03-views-layout-and-restoration) / active | 2444–2447 | Forget exact locator and persist browser layout; declarations/fields: `confirmedEnd` |
| [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) / active | 2448–2454 | Dispose mounted owner after confirmed native End; declarations/fields: `confirmedEnd` |


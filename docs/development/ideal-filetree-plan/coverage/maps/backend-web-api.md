# Backend web api

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-02c4cfae64a5"></a>

## [internal/web/api/api.go](../../../../../internal/web/api/api.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state. Shared physical declarations/SQL rows/constructor lines have separate named-field units; this is coupled storage/projection, not competing decision ownership.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 19, 27, 36 | Record, DTO or interface contract API for Browser authority and contributions; declarations/fields: `API` |
| [H04](../../slices/shared-supporting-slices.md#h04-configuration-and-filesystem-primitives) / active | 20–21 | Configured service dependencies and restricted input references; declarations/fields: `API.Config`, `API.Store` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 22 | Authoritative native read client dependency; declarations/fields: `API.Forgejo` |
| [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) / active | 23 | Private host IPC dependency; declarations/fields: `API.Host` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 24 | Current native actor/session admission dependency; declarations/fields: `API.Auth` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 25 | Native identity metadata/admission client dependency; declarations/fields: `API.Identity` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 26 | Exclusive factory coordination dependency; declarations/fields: `API.Coordinator` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 28 | Human terminal registry mutex; declarations/fields: `API.terminalMu` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 29 | Bounded concurrent authorized inventory observations; declarations/fields: `API.SpacesSlots` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 30 | Bounded concurrent repository-choice observations; declarations/fields: `API.RepositorySlots` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 31–35 | Human terminal lifetime/cancellation registry; declarations/fields: `API.TerminalPeers`, `API.TerminalStopping`, `API.terminalClosed`, `API.terminalWG` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 37–42 | Tracked native terminal operation/stream lifetime and cancellation; declarations/fields: `TerminalPeer` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 43 | Record, DTO or interface contract TerminalPeer for Browser authority and contributions; declarations/fields: `TerminalPeer` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 44–46, 48–49 | New — Browser authority and contributions; declarations/fields: `New` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 47 | Repository lookup admission slot in shared constructor; declarations/fields: `API.RepositorySlots` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 47 | Spaces and repository lookup admission slots share constructor expression; declarations/fields: `API.SpacesSlots` |

<a id="coverage-d5401ed1f9f4"></a>

## [internal/web/api/environment_authority.go](../../../../../internal/web/api/environment_authority.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 16–21 | Declared identifiers/bounds (declaration group) for Human membership and accounts; declarations/fields: `(declaration group)` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 22–27 | Record, DTO or interface contract repositoryAccess for Human membership and accounts; declarations/fields: `repositoryAccess` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 28–59 | Current native repository evidence projection; callers own product admission; declarations/fields: `API.visibleRepository`, `API.nativeVisibleRepository` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 60–63 | repositoryExecutionAllowed — Human membership and accounts; declarations/fields: `repositoryExecutionAllowed` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 64–71 | API.executionRepository — Human membership and accounts; declarations/fields: `API.executionRepository` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 72–81 | reportExecutionAuthorityError — Human membership and accounts; declarations/fields: `reportExecutionAuthorityError` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 82–92 | Project lifecycle/preparation administrator authority from current native repository; declarations/fields: `API.environmentAdministrator` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 93–100 | Record, DTO or interface contract environmentReader for Human membership and accounts; declarations/fields: `environmentReader` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 101–102 | Declared identifiers/bounds errEnvironmentReadStore for Human membership and accounts; declarations/fields: `errEnvironmentReadStore` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 103–132 | API.readEnvironmentAuthority — Human membership and accounts; declarations/fields: `API.readEnvironmentAuthority` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 133–141 | API.authorizeEnvironmentRead — Human membership and accounts; declarations/fields: `API.authorizeEnvironmentRead` |

<a id="coverage-e46927017b7b"></a>

## [internal/web/api/environments_api.go](../../../../../internal/web/api/environments_api.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state. Shared physical declarations/SQL rows/constructor lines have separate named-field units; this is coupled storage/projection, not competing decision ownership.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–13 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 14, 16–20 | Association/immutable selected creation profile DTO; declarations/fields: `EnvironmentView` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 15 | Immutable selected native creation profile in association DTO; declarations/fields: `EnvironmentView.Profile` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 21 | Confirmed native provisioning result; declarations/fields: `EnvironmentView.Provisioned` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 22–23, 34, 54–61, 70–99 | Bounded authorized association list presentation; declarations/fields: `EnvironmentView`, `EnvironmentDTO`, `listedEnvironments`, `API.apiEnvironments` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 24–30 | Record, DTO or interface contract repositoryContextView for Repository association and creation; declarations/fields: `repositoryContextView` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 31, 33 | Composite association DTO rendering retains native-ready source below; declarations/fields: `EnvironmentDTO` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 32 | Repository association selectors in EnvironmentDTO constructor; declarations/fields: `p.ID`, `p.Name`, `p.RepositoryID`, `p.Repository`, `p.OwnerID` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 32 | Immutable selected profile and confirmed readiness selectors in EnvironmentDTO constructor; declarations/fields: `p.Profile`, `p.Ready` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 35–39 | environmentListQuery — Repository association and creation; declarations/fields: `environmentListQuery` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 40–53 | parseEnvironmentsListQuery — Repository association and creation; declarations/fields: `parseEnvironmentsListQuery` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 62–69 | API.requireListedSession — Repository association and creation; declarations/fields: `API.requireListedSession` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 100–112 | API.loadEnvironment — Repository association and creation; declarations/fields: `API.loadEnvironment` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 113–126, 136–139, 151, 160–161 | Confirmed runtime profile/state observation; declarations/fields: `environmentProfileMismatch`, `observedEnvironment`, `API.apiEnvironment` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 127–131 | Load retained Project association; declarations/fields: `apiEnvironment` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 132–135 | Authorize own membership/current repository visibility; declarations/fields: `authorizeEnvironmentRead` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 140–147 | Complete retained creation reservation only from confirmed native evidence; declarations/fields: `reconcileProvisioning` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 148–150 | Fresh native session recheck before publishing detail; declarations/fields: `requireListedSession` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 152–153 | Membership/write permission and uncertainty fields; declarations/fields: `authority_unavailable`, `execution_allowed` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 154 | Composite association detail reference; embedded profile/readiness fields listed separately; declarations/fields: `EnvironmentView.ID`, `EnvironmentView.Name`, `EnvironmentView.RepositoryID`, `EnvironmentView.Repository`, `EnvironmentView.OwnerID` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 154 | Embedded immutable profile/confirmed provisioning readiness in association detail reference; declarations/fields: `EnvironmentView.Profile`, `EnvironmentView.Provisioned` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 155–156 | Observed native runtime and unavailability fields; declarations/fields: `observed`, `native_unavailable` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 157 | Own mapped human Linux login; declarations/fields: `login` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 158 | Current Project administrator authority projection in environment read model; declarations/fields: `environment_administrator` |
| [N02](../../slices/networking.md#n02-project-lan-access) / active | 159 | Native observed LAN IP carried in compound runtime observation reference; declarations/fields: `observed.IP` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 159 | Association field selectors in compound detail constructor; declarations/fields: `EnvironmentDTO(p).ID`, `EnvironmentDTO(p).Name`, `EnvironmentDTO(p).RepositoryID`, `EnvironmentDTO(p).Repository`, `EnvironmentDTO(p).OwnerID` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 159 | Immutable profile/readiness/native runtime observation selectors in compound detail constructor; declarations/fields: `EnvironmentDTO(p).Profile`, `EnvironmentDTO(p).Provisioned`, `observed`, `nativeErr != nil` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 159 | Own membership/login/current administrator/write-authority selectors in compound detail constructor; declarations/fields: `reader.authorityUnavailable`, `reader.executionAllowed`, `reader.login`, `reader.administrator` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 162–197 | Bounded authorized human membership inventory; declarations/fields: `API.apiEnvironmentMembers` |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 198–226 | Own development SSH connection observation; declarations/fields: `API.apiConnection` |

<a id="coverage-bcac6b63d5d6"></a>

## [internal/web/api/environments_create.go](../../../../../internal/web/api/environments_create.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 17–21 | Record, DTO or interface contract apiError for Repository association and creation; declarations/fields: `apiError` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 22–27 | Record, DTO or interface contract createEnvironmentInput for Repository association and creation; declarations/fields: `createEnvironmentInput` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 28–32 | Declared identifiers/bounds (declaration group) for Repository association and creation; declarations/fields: `(declaration group)` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 33–45 | parseCreateEnvironmentInput — Repository association and creation; declarations/fields: `parseCreateEnvironmentInput` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 46–61 | API.verifyRepositoryOwner — Repository association and creation; declarations/fields: `API.verifyRepositoryOwner` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 62–73 | API.lookupReservation — Repository association and creation; declarations/fields: `API.lookupReservation` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 74–90, 186–204 | Explicit creation-time original Tailnet selection; declarations/fields: `API.checkTailnetPreflight`, `API.applyCreatedEnvironmentTailnet` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 91–113 | Installed runtime profile preflight before reservation; declarations/fields: `API.resolveNativeProfile`, `API.precheckProfileAndTailnet` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 114–120 | Fresh native session admission recheck; declarations/fields: `API.verifyCurrentSessionMatch` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 121–137 | API.reconfirmRepositoryAndSession — Repository association and creation; declarations/fields: `API.reconfirmRepositoryAndSession` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 138–163 | API.provisionAndSaveProject — Repository association and creation; declarations/fields: `API.provisionAndSaveProject` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 164–185 | API.reconcileCreate — Repository association and creation; declarations/fields: `API.reconcileCreate` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 205–266 | API.apiCreateEnvironment — Repository association and creation; declarations/fields: `API.apiCreateEnvironment` |

<a id="coverage-6e18ba4f16af"></a>

## [internal/web/api/environments_join.go](../../../../../internal/web/api/environments_join.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 13–36 | Explicit own saved-or-none SSH key choice for new human account; declarations/fields: `validJoinSSHSelection`, `API.joinPublicKeys`, `errTooManyJoinKeys` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 37–49 | API.persistEnvironmentJoin — Human membership and accounts; declarations/fields: `API.persistEnvironmentJoin` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 50–55 | Declared identifiers/bounds (declaration group) for Human membership and accounts; declarations/fields: `(declaration group)` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 56–61 | API.writeJoinLogin — Human membership and accounts; declarations/fields: `API.writeJoinLogin` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 62–74 | API.reportJoinPersist — Human membership and accounts; declarations/fields: `API.reportJoinPersist` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 75–98 | API.admitNewJoin — Human membership and accounts; declarations/fields: `API.admitNewJoin` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 99–139 | API.apiJoinEnvironment — Human membership and accounts; declarations/fields: `API.apiJoinEnvironment` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 140–142 | API.currentJoinSession — Human membership and accounts; declarations/fields: `API.currentJoinSession` |

<a id="coverage-6a3b63c40fb7"></a>

## [internal/web/api/environments_preparation.go](../../../../../internal/web/api/environments_preparation.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 17–30 | Record, DTO or interface contract preparationItemView for Checkout allocation and preparation; declarations/fields: `preparationItemView` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 31–39 | Record, DTO or interface contract environmentPreparationView for Checkout allocation and preparation; declarations/fields: `environmentPreparationView` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 40–49 | preparationItemDTO — Checkout allocation and preparation; declarations/fields: `preparationItemDTO` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 50–91 | API.apiPreparation — Checkout allocation and preparation; declarations/fields: `API.apiPreparation` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 92–103, 117–190 | Preparation maintenance hold synchronization/observation; declarations/fields: `API.preparationHoldState`, `preparationHoldRequest`, `API.apiPreparationHold`, `API.syncPreparationHold` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 104–116 | API.preparationItems — Checkout allocation and preparation; declarations/fields: `API.preparationItems` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 191–239 | Readiness summary in authorized Spaces row; declarations/fields: `spacePreparationRole`, `spacePreparationView`, `summarizeSpacePreparation`, `API.inspectSpacePreparation` |

<a id="coverage-e7cbee456996"></a>

<a id="internalwebapiextension_terminalgo-1"></a>

## [internal/web/api/extension_terminal.go](../../../../../internal/web/api/extension_terminal.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–23 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 24–44, 62–77, 82–125, 201–204 | Live native actor/contribution admission for terminal calls; declarations/fields: `API.ExtensionHandler`, `extensionTerminalGeneration`, `extensionTerminalOrigin`, `nativeTerminalContribution`, `API.nativeTerminalActor`, `API.extensionTerminalAccount`, `sameNativeTerminalAuthority`, `nativeTerminalSessionActor` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 45–49 | privateTerminalStreamRoute — Interactive attachment; declarations/fields: `privateTerminalStreamRoute` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 50–56 | Record, DTO or interface contract extensionTerminalIdentity for Interactive attachment; declarations/fields: `extensionTerminalIdentity` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 57–61, 156–200, 205–284, 371–386 | Human terminal reserve/create/list/rename/end lifecycle; declarations/fields: `nativeTerminalScope`, `API.extensionTerminalOperation`, `API.extensionTerminalStates`, `extensionTerminalView`, `API.extensionReserveTerminal`, `API.extensionTerminalSession`, `extensionTerminalSessionMethod`, `API.extensionOpenHostTerminal` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 78–81, 126–137 | Current own account/repository admission prerequisite for terminal operation; declarations/fields: `nativeTerminalMember`, `nativeTerminalMembershipCurrent`, `API.extensionTerminalRepository` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 138–155 | API.extensionTerminalCurrent — Interactive attachment; declarations/fields: `API.extensionTerminalCurrent` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 285–295 | API.extensionTerminalStream — Interactive attachment; declarations/fields: `API.extensionTerminalStream` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 296–337 | API.extensionTerminalAttach — Interactive attachment; declarations/fields: `API.extensionTerminalAttach` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 338–352 | API.registerExtensionTerminalPeer — Interactive attachment; declarations/fields: `API.registerExtensionTerminalPeer` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 353–357 | validNativeTerminalHandshake — Interactive attachment; declarations/fields: `validNativeTerminalHandshake` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 358–370 | API.extensionClaimTerminal — Interactive attachment; declarations/fields: `API.extensionClaimTerminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 387–406 | API.pumpExtensionControls — Interactive attachment; declarations/fields: `API.pumpExtensionControls` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 407–411 | API.extensionTerminalHeartbeat — Interactive attachment; declarations/fields: `API.extensionTerminalHeartbeat` |

<a id="coverage-5dac9632c419"></a>

## [internal/web/api/extension_test.go](../../../../../internal/web/api/extension_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 17–27 | Assertions TestExtensionResponseIsBounded: expected bounded response rejection, got %v; declarations/fields: `TestExtensionResponseIsBounded` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 28–49 | Assertions TestExtensionRejectsUnknownRouteAndUnverifiedActor: %s %s: got %d, want %d; declarations/fields: `TestExtensionRejectsUnknownRouteAndUnverifiedActor` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 50–70 | Assertions TestExtensionAuthorityDenialLogsCause: got %d, want 403; declarations/fields: `TestExtensionAuthorityDenialLogsCause` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 71–91 | Assertions TestExtensionProductAuthorityDenialLogsCause: bare request admitted; declarations/fields: `TestExtensionProductAuthorityDenialLogsCause` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 92–150 | Assertions TestExtensionProductRouteAllowlist: extensionRoute(%s %s) = %t, want %t; declarations/fields: `TestExtensionProductRouteAllowlist` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 151–170 | Assertions TestExtensionProductContributionScopes: extensionProductContribution(%q, %+v) = %t, want %t; declarations/fields: `TestExtensionProductContributionScopes` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 171–191, 214–238 | Native terminal WebSocket header/authority proxy admission assertions; declarations/fields: `TestNativeTerminalContributionScopes`, `TestExtensionTerminalUpgradeForwardsOnlyProtocolAndAuthority` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 192–213 | Assertions TestExtensionTransportDoesNotForwardBrowserCredentials: private service path = %q; declarations/fields: `TestExtensionTransportDoesNotForwardBrowserCredentials` |

<a id="coverage-b018d5c05a94"></a>

## [internal/web/api/factory_assignments.go](../../../../../internal/web/api/factory_assignments.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–16 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 17–30 | Record, DTO or interface contract factoryAssignmentResult for Assignment and dispatch; declarations/fields: `factoryAssignmentResult` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 31–40 | Assignment reservation read model; declarations/fields: `factoryAssignmentReservation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 41–62 | Assignment publication exact-link/effect read model; declarations/fields: `factoryAssignmentPublication` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 63–64, 67–93 | Record, DTO or interface contract factoryAssignmentView for Assignment and dispatch; declarations/fields: `factoryAssignmentView` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 65 | Assignment view projects independently owned reservation accounting; declarations/fields: `factoryAssignmentView.Reservation` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 66 | Assignment view projects independently owned publication progression; declarations/fields: `factoryAssignmentView.Publication` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 94–113, 136–140 | factoryAssignmentDTO — Assignment and dispatch; declarations/fields: `factoryAssignmentDTO` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 114–119 | Project assignment reservation accounting in composite assignment DTO; declarations/fields: `factoryAssignmentDTO` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 120–135 | Project exact publication link/effect/completion in composite assignment DTO; declarations/fields: `factoryAssignmentDTO` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 141–151 | currentIssueAssignment — Assignment and dispatch; declarations/fields: `currentIssueAssignment` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 152–187 | API.apiFactoryAssignment — Assignment and dispatch; declarations/fields: `API.apiFactoryAssignment` |

<a id="coverage-d0227b7d5d72"></a>

## [internal/web/api/factory_assignments_test.go](../../../../../internal/web/api/factory_assignments_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 1–10 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 11–54 | Assertions TestFactoryAssignmentDTORendersInputsAndResult: view = %+v; declarations/fields: `TestFactoryAssignmentDTORendersInputsAndResult` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 55–66 | Assertions TestCurrentIssueAssignmentSelectsLatest: empty selection accepted; declarations/fields: `TestCurrentIssueAssignmentSelectsLatest` |
| [F09](../../slices/factory-coordination.md#f09-publication-progression) / active | 67–77 | Publication committed effect/completion visibility assertions; declarations/fields: `TestFactoryAssignmentDTOPreservesIncompletePublication` |

<a id="coverage-0ed9d77a0a9e"></a>

## [internal/web/api/factory_intake_test.go](../../../../../internal/web/api/factory_intake_test.go)

Test/fixture assertions inspected as source only; no test, build, native driver or browser run performed. A defined scenario is not a passing qualification result.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 19–26 | Fixture/protocol support stubIntakeCoordinator; declarations/fields: `stubIntakeCoordinator` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 27–32 | Fixture/protocol support stubIntakeCoordinator.ObserveIssueEvent; declarations/fields: `stubIntakeCoordinator.ObserveIssueEvent` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 33–38 | Fixture/protocol support signIntake; declarations/fields: `signIntake` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 39–47 | Fixture/protocol support intakeRequest; declarations/fields: `intakeRequest` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 48–76 | Assertions TestFactoryIntakeOpenedCarriesCreatorAuthority: opened intake refused:; declarations/fields: `TestFactoryIntakeOpenedCarriesCreatorAuthority` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 77–109 | Assertions TestFactoryIntakeHints: edited hint wrong:; declarations/fields: `TestFactoryIntakeHints` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 110–129 | Assertions TestFactoryIntakeRejectsForgedDeliveries: forged delivery assessed:; declarations/fields: `TestFactoryIntakeRejectsForgedDeliveries` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 130–179 | Assertions TestFactoryIntakeFailures: malformed hint accepted:; declarations/fields: `TestFactoryIntakeFailures` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 180–195 | Unwired evidence source refusal assertions; declarations/fields: `TestServiceReadinessSourceNilGuards` |

<a id="coverage-e83d1593855f"></a>

## [internal/web/api/factory_readiness.go](../../../../../internal/web/api/factory_readiness.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 15, 19–20 | Record, DTO or interface contract ServiceReadinessSource for Issue intake and readiness; declarations/fields: `ServiceReadinessSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 16 | Acceptance evidence reader contract; declarations/fields: `ServiceReadinessSource.Evidence` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 17 | Exact dispatch input reader contract; declarations/fields: `ServiceReadinessSource.Dispatch` |
| [G03](../../slices/forgejo-integration.md#g03-authoritative-native-reads) / active | 18 | Shared authoritative native observer; declarations/fields: `ServiceReadinessSource.Observer` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 21, 23, 25–28 | Declared identifiers/bounds (declaration group) for Issue intake and readiness; declarations/fields: `(declaration group)` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 22 | Compile-time acceptance evidence contract; declarations/fields: `control.AcceptanceSource` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 24 | Compile-time exact dispatch inputs contract; declarations/fields: `control.DispatchReads` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 29–30, 37–42 | NewServiceReadinessSource — Issue intake and readiness; declarations/fields: `NewServiceReadinessSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 31–33 | Wire exact selected native acceptance reader; declarations/fields: `AcceptanceSnapshotSource` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 34–36 | Wire exact accepted native dispatch reader; declarations/fields: `DispatchSnapshotSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 43–51 | Map bracketed native evidence for accepted requirement verification; declarations/fields: `ServiceReadinessSource.ReadAcceptanceEvidence` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 52–60 | Map exact current accepted-source prompt/branch inputs; declarations/fields: `ServiceReadinessSource.ReadDispatchInputs` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 61–73 | ServiceReadinessSource.ObserveNativeRevision — Issue intake and readiness; declarations/fields: `ServiceReadinessSource.ObserveNativeRevision` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 74–79 | ServiceReadinessSource.ListRepositoryIssues — Issue intake and readiness; declarations/fields: `ServiceReadinessSource.ListRepositoryIssues` |

<a id="coverage-619247c30ebc"></a>

<a id="internalwebapifactory_settingsgo-1"></a>

## [internal/web/api/factory_settings.go](../../../../../internal/web/api/factory_settings.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 1–17 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 18–21 | factoryPrincipal — Repository factory policy; declarations/fields: `factoryPrincipal` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 22–38 | factoryCommandError — Repository factory policy; declarations/fields: `factoryCommandError` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 39–53 | API.factoryRepository — Repository factory policy; declarations/fields: `API.factoryRepository` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 54–75 | API.factoryOwner — Repository factory policy; declarations/fields: `API.factoryOwner` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 76–85, 307–341 | Configured operator-only capacity mutation; declarations/fields: `API.factoryOperator`, `factoryCapacityRequest`, `API.apiFactoryCapacity` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 86–93 | settingsCommandID — Repository factory policy; declarations/fields: `settingsCommandID` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 94–98 | Current issue dispatch queue summary; declarations/fields: `factoryQueueView` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 99, 108–110 | Record, DTO or interface contract factoryStatusView for Repository factory policy; declarations/fields: `factoryStatusView` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 100 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.Policy` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 101 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.OperatorGrant` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 102 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.Appliance` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 103 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.Sponsorships` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 104 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.EnvironmentGrant` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 105 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.Preparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 106–107 | Composite status projects separate authority state; declarations/fields: `factoryStatusView.Queue`, `factoryStatusView.Effective` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 111–112, 115–119, 195–231 | Preparation readiness/decision-reference read model; declarations/fields: `factoryPreparationView`, `API.factoryPreparationStatus` |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 113 | Requirement acceptance reference in preparation status; declarations/fields: `factoryPreparationView.Requirements` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 114 | Administrator approval reference in preparation status; declarations/fields: `factoryPreparationView.Approval` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 120–127, 423–499 | Connection-owner delegation/generation sponsorship admission; declarations/fields: `sponsorshipView`, `factorySponsorshipRequest`, `API.apiFactorySponsorship`, `API.checkSponsorshipBroker` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 128–144, 192–194 | API.apiFactoryStatus — Repository factory policy; declarations/fields: `API.apiFactoryStatus` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 145–150 | Aggregate factory status branch projects operator execution grant; declarations/fields: `apiFactoryStatus` |
| [F03](../../slices/factory-coordination.md#f03-capacity-reservations-and-accounting) / active | 151–156 | Aggregate factory status branch projects appliance capacity; declarations/fields: `apiFactoryStatus` |
| [F04](../../slices/factory-coordination.md#f04-connection-sponsorship) / active | 157–164 | Aggregate factory status branch projects sponsorships and owner-only private allowance; declarations/fields: `apiFactoryStatus` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 165–170 | Aggregate factory status branch projects standing environment creation permission; declarations/fields: `apiFactoryStatus` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 171–175 | Aggregate factory status branch projects current preparation readiness; declarations/fields: `apiFactoryStatus` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 176–181 | Aggregate factory status branch projects effective composite dispatch admission; declarations/fields: `apiFactoryStatus` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 182–191 | Aggregate factory status branch projects current queued dispatch items; declarations/fields: `apiFactoryStatus` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 232–236 | Record, DTO or interface contract actorRefRequest for Repository factory policy; declarations/fields: `actorRefRequest` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 237–243 | actorRefRequest.resolve — Repository factory policy; declarations/fields: `actorRefRequest.resolve` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 244–259 | Record, DTO or interface contract factoryPolicyRequest for Repository factory policy; declarations/fields: `factoryPolicyRequest` |
| [F01](../../slices/factory-coordination.md#f01-repository-factory-policy) / active | 260–306 | API.apiFactoryPolicy — Repository factory policy; declarations/fields: `API.apiFactoryPolicy` |
| [F02](../../slices/factory-coordination.md#f02-operator-execution-grants) / active | 342–373 | Operator repository execution permission route/DTO; declarations/fields: `factoryOperatorGrantRequest`, `API.apiFactoryOperatorGrant` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 374–422 | Current repository-owner standing environment grant mutation; declarations/fields: `factoryEnvironmentGrantRequest`, `API.apiFactoryEnvironmentGrant` |

<a id="coverage-98a92690bd62"></a>

## [internal/web/api/identity.go](../../../../../internal/web/api/identity.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 1–14 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 15, 18–20, 27–28 | Record, DTO or interface contract IdentityClient for Enrollment and owner consent; declarations/fields: `IdentityClient` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 16–17 | Owned connections and delegated availability API contract; declarations/fields: `IdentityClient.Connections`, `IdentityClient.Available` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 21–23 | Delegation grant API contract; declarations/fields: `IdentityClient.Grants`, `IdentityClient.CreateGrant`, `IdentityClient.RevokeGrant` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 24 | Lease metadata API contract; declarations/fields: `IdentityClient.Leases` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 25–26 | Explicit connection/lease retirement API contract; declarations/fields: `IdentityClient.EndLease`, `IdentityClient.Revoke` |
| [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) / active | 29–61, 124–134 | Current native actor admission/error envelope shared by identity operations; declarations/fields: `identityError`, `API.identityAdmission`, `API.identityResult`, `API.identityMutation` |
| [I03](../../slices/identity-brokering.md#i03-delegation-and-connection-availability) / active | 62–72, 102–112, 149–155 | Owned connection/delegation metadata read or grant revocation; declarations/fields: `API.apiIdentityConnections`, `API.apiIdentityGrants`, `API.apiIdentityRevokeGrant` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 73–93 | API.apiIdentityStartEnrollment — Enrollment and owner consent; declarations/fields: `API.apiIdentityStartEnrollment` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 94–101 | API.apiIdentityEnrollment — Enrollment and owner consent; declarations/fields: `API.apiIdentityEnrollment` |
| [I04](../../slices/identity-brokering.md#i04-execution-admission-and-lease-fencing) / active | 113–123 | Own lease/fence metadata inventory; declarations/fields: `API.apiIdentityLeases` |
| [I01](../../slices/identity-brokering.md#i01-enrollment-and-owner-consent) / active | 135–141 | API.apiIdentityCancelEnrollment — Enrollment and owner consent; declarations/fields: `API.apiIdentityCancelEnrollment` |
| [I06](../../slices/identity-brokering.md#i06-completion-revocation-and-reconciliation) / active | 142–148, 156–161 | Explicit credential/lease retirement; declarations/fields: `API.apiIdentityRevoke`, `API.apiIdentityEndLease` |

<a id="coverage-aa4c2fa22d23"></a>

<a id="internalwebapiissue_acceptancesgo-1"></a>

## [internal/web/api/issue_acceptances.go](../../../../../internal/web/api/issue_acceptances.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 1–21 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 22–31 | Record, DTO or interface contract AcceptanceSnapshotSource for Accepted requirements and invalidation; declarations/fields: `AcceptanceSnapshotSource` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 32–96 | AcceptanceSnapshotSource.ReadAcceptanceEvidence — Accepted requirements and invalidation; declarations/fields: `AcceptanceSnapshotSource.ReadAcceptanceEvidence` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 97–113 | mapAcceptanceSnapshotError — Accepted requirements and invalidation; declarations/fields: `mapAcceptanceSnapshotError` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 114–126 | API.factoryIssue — Accepted requirements and invalidation; declarations/fields: `API.factoryIssue` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 127–132 | Record, DTO or interface contract acceptanceSourceRequest for Accepted requirements and invalidation; declarations/fields: `acceptanceSourceRequest` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 133–141 | Record, DTO or interface contract acceptancePrerequisiteRequest for Accepted requirements and invalidation; declarations/fields: `acceptancePrerequisiteRequest` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 142–158 | Record, DTO or interface contract issueAcceptanceRequest for Accepted requirements and invalidation; declarations/fields: `issueAcceptanceRequest` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 159–190 | API.apiIssueAcceptances — Accepted requirements and invalidation; declarations/fields: `API.apiIssueAcceptances` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 191–225 | acceptanceDecision — Accepted requirements and invalidation; declarations/fields: `acceptanceDecision` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 226–238 | acceptanceSources — Accepted requirements and invalidation; declarations/fields: `acceptanceSources` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 239–246 | Record, DTO or interface contract issueWithdrawalRequest for Accepted requirements and invalidation; declarations/fields: `issueWithdrawalRequest` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 247–275 | API.apiIssueWithdrawal — Accepted requirements and invalidation; declarations/fields: `API.apiIssueWithdrawal` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 276, 278–279, 283–298, 372–384, 391–401, 414–419 | Current issue readiness composite read model; independent acceptance/check/merge state is separately owned; declarations/fields: `factoryIssueView`, `readinessView`, `API.apiFactoryIssue` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 277 | Current accepted requirement validity projection; declarations/fields: `factoryIssueView.Status` |
| [F06](../../slices/factory-coordination.md#f06-issue-intake-and-readiness) / active | 280 | Durable readiness verdict projection; declarations/fields: `factoryIssueView.Readiness` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 281 | Exact-candidate verification assessment projection; declarations/fields: `factoryIssueView.Checks` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 282 | Merge eligibility/completion projection; declarations/fields: `factoryIssueView.Merge` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 299–342 | Recorded exact-candidate check verdict read model; declarations/fields: `checkResultView`, `checksView`, `checksViewDTO` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 343–371 | Recorded conditional merge/completion read model; declarations/fields: `mergeView`, `mergeViewDTO` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 385–390 | Read and publish current acceptance validity; declarations/fields: `Coordinator.AcceptanceStatus` |
| [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) / active | 402–407 | Read and project recorded exact-candidate check assessment; declarations/fields: `Store.CheckAssessment` |
| [F12](../../slices/factory-coordination.md#f12-merge-eligibility-and-completion) / active | 408–413 | Read and project recorded conditional merge progression; declarations/fields: `Store.MergeForIssue` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 420–429 | acceptanceDecisionError — Accepted requirements and invalidation; declarations/fields: `acceptanceDecisionError` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 430–439 | acceptanceRefusalError — Accepted requirements and invalidation; declarations/fields: `acceptanceRefusalError` |
| [F05](../../slices/factory-coordination.md#f05-accepted-requirements-and-invalidation) / active | 440–479 | acceptanceRefusalMessage — Accepted requirements and invalidation; declarations/fields: `acceptanceRefusalMessage` |

<a id="coverage-9276e67551ee"></a>

## [internal/web/api/lifecycle.go](../../../../../internal/web/api/lifecycle.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 1–15 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 16–23 | API.loadLifecycleEnvironment — Project Start/Stop; declarations/fields: `API.loadLifecycleEnvironment` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 24–28 | Record, DTO or interface contract lifecycleRequest for Project Start/Stop; declarations/fields: `lifecycleRequest` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 29–40 | decodeLifecycleRequest — Project Start/Stop; declarations/fields: `decodeLifecycleRequest` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 41–61 | API.authorizeLifecycleOperator — Project Start/Stop; declarations/fields: `API.authorizeLifecycleOperator` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 62–69 | API.checkLifecycleSession — Project Start/Stop; declarations/fields: `API.checkLifecycleSession` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 70–94 | API.acquireLifecycleStop — Project Start/Stop; declarations/fields: `API.acquireLifecycleStop` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 95–121 | API.handleLifecycleMutation — Project Start/Stop; declarations/fields: `API.handleLifecycleMutation` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 122–162 | API.apiLifecycle — Project Start/Stop; declarations/fields: `API.apiLifecycle` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 163–169 | Record, DTO or interface contract lifecycleStopControlView for Project Start/Stop; declarations/fields: `lifecycleStopControlView` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 170–179 | Record, DTO or interface contract lifecycleStopView for Project Start/Stop; declarations/fields: `lifecycleStopView` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 180–207 | API.apiLifecycleStop — Project Start/Stop; declarations/fields: `API.apiLifecycleStop` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 208–228 | Project Stop synchronizes admission hold without owning hold policy; declarations/fields: `API.lifecycleStopHold` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 229–237 | Record, DTO or interface contract lifecycleStartView for Project Start/Stop; declarations/fields: `lifecycleStartView` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 238–271 | API.apiLifecycleStart — Project Start/Stop; declarations/fields: `API.apiLifecycleStart` |

<a id="coverage-01a96515da16"></a>

## [internal/web/api/preparation_decisions.go](../../../../../internal/web/api/preparation_decisions.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P08](../../slices/projects.md#p08-preparation-requirements-acceptance) / active | 13–56 | Exact requirement acceptance route/input; declarations/fields: `preparationAcceptanceRequest`, `API.apiPreparationAcceptances` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 57–67 | Record, DTO or interface contract preparationActionRequest for Privileged preparation approval; declarations/fields: `preparationActionRequest` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 68–79, 125–148 | Existing preparation/decision-reference observation; declarations/fields: `preparationInspectView`, `API.apiPreparationActionInspect` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 80–100 | API.apiPreparationActions — Privileged preparation approval; declarations/fields: `API.apiPreparationActions` |
| [P12](../../slices/projects.md#p12-maintenance-holds) / active | 101–124 | Preparation admission maintenance hold command; declarations/fields: `API.apiPreparationActionHold` |
| [P09](../../slices/projects.md#p09-privileged-preparation-approval) / active | 149–171 | API.apiPreparationActionApprove — Privileged preparation approval; declarations/fields: `API.apiPreparationActionApprove` |

<a id="coverage-a38203d6ba5a"></a>

## [internal/web/api/provisioning.go](../../../../../internal/web/api/provisioning.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 1–22 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 23–26, 33–41 | Declared identifiers/bounds (declaration group) for Repository association and creation; declarations/fields: `(declaration group)` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 27–32 | Native Stop/Start bounded operation deadlines; declarations/fields: `stopOperationTimeout`, `startOperationTimeout` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 42–49 | operationContext — Repository association and creation; declarations/fields: `operationContext` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 50–57 | provisioningConfirmed — Repository association and creation; declarations/fields: `provisioningConfirmed` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 58–69 | API.reconcileProvisioning — Repository association and creation; declarations/fields: `API.reconcileProvisioning` |

<a id="coverage-1b43436f7ee8"></a>

<a id="internalwebapispacesgo-1"></a>

## [internal/web/api/spaces.go](../../../../../internal/web/api/spaces.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 1–18 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 19, 33–37 | Record, DTO or interface contract SpaceView for Authorized inventory; declarations/fields: `SpaceView` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 20 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.TailnetState` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 21 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.Environment` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 22–23, 25 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.Login`, `SpaceView.ExecutionAllowed`, `SpaceView.AuthorityUnavailable` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 24 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.Administrator` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 26–27 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.NativeUnavailable`, `SpaceView.Observed` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 28 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.Terminals` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 29 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.Preparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 30 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.FactoryAuthority` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 31 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.FactoryControl` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 32 | Spaces row projects independent source-of-truth state; declarations/fields: `SpaceView.FactoryRuns` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 38–49, 186–222 | Recorded factory run activity read model and completeness bounds; declarations/fields: `SpaceFactoryRun`, `inspectSpaceFactoryRuns` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 50–55, 223–237 | Composite effective factory dispatch authority projection; declarations/fields: `spaceAuthorityView`, `API.inspectSpaceAuthority` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 56–63 | Record, DTO or interface contract SpacesView for Authorized inventory; declarations/fields: `SpacesView` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 64–68 | Record, DTO or interface contract SpacesActor for Authorized inventory; declarations/fields: `SpacesActor` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 69–77 | Current own membership/native repository authority before row disclosure; declarations/fields: `API.resolveSpaceAuthority` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 78–90 | Observe native runtime/readiness against immutable creation profile; declarations/fields: `API.inspectSpaceNative` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 91–117 | Observe own human terminal lifecycle metadata under current membership/native authority; declarations/fields: `API.inspectSpaceTerminals` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 118–135 | Observe own original Project network selection without projecting peer secrets; declarations/fields: `API.inspectSpaceTailnet` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 136–156, 164–165, 178–185 | API.inspectSpaceRow — Authorized inventory; declarations/fields: `API.inspectSpaceRow` |
| [P01](../../slices/projects.md#p01-repository-association-and-creation) / active | 157 | Project association projection in Spaces row; declarations/fields: `inspectSpaceRow`, `EnvironmentDTO` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 158–159 | Current own human account/write-authority projection; declarations/fields: `inspectSpaceRow` |
| [P05](../../slices/projects.md#p05-project-startstop) / active | 160 | Current Project administrator projection; declarations/fields: `inspectSpaceRow` |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 161 | Membership/native authority uncertainty projection; declarations/fields: `inspectSpaceRow` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 162 | Human terminal metadata collection initialization; declarations/fields: `inspectSpaceRow` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 163 | Factory recorded activity collection initialization; declarations/fields: `inspectSpaceRow` |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 166–168 | Observe runtime state against selected creation profile; declarations/fields: `inspectSpaceNative` |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 169–171 | Read current own terminal metadata with global bound; declarations/fields: `inspectSpaceTerminals` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 172 | Read Project network selection; declarations/fields: `inspectSpaceTailnet` |
| [P07](../../slices/projects.md#p07-checkout-allocation-and-preparation) / active | 173–174 | Read durable role checkout preparation readiness; declarations/fields: `inspectSpacePreparation` |
| [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) / active | 175 | Read effective dispatch admission; declarations/fields: `inspectSpaceAuthority` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 176 | Read intervention/withdrawal/unsettled run state; declarations/fields: `inspectSpaceControl` |
| [S06](../../slices/spaces-and-terminals.md#s06-factory-activity-presentation) / active | 177 | Project recorded run activity while preserving incompleteness; declarations/fields: `inspectSpaceFactoryRuns` |
| [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) / active | 238–272 | Current intervention/withdrawal/unsettled run read model; declarations/fields: `spaceControlView`, `API.inspectSpaceControl` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 273–283 | appendSpaceRow — Authorized inventory; declarations/fields: `appendSpaceRow` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 284–342 | API.inspectSpaces — Authorized inventory; declarations/fields: `API.inspectSpaces` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 343–360 | API.verifySpacesSession — Authorized inventory; declarations/fields: `API.verifySpacesSession` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 361–390 | API.apiSpaces — Authorized inventory; declarations/fields: `API.apiSpaces` |
| [S01](../../slices/spaces-and-terminals.md#s01-authorized-inventory) / active | 391–404 | parseSpacesCursor — Authorized inventory; declarations/fields: `parseSpacesCursor` |

<a id="coverage-69a9d21426ed"></a>

## [internal/web/api/tailnet.go](../../../../../internal/web/api/tailnet.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–12 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 13–21 | tailnetQuery — Host Tailnet control; declarations/fields: `tailnetQuery` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 22–29 | API.tailnetSession — Host Tailnet control; declarations/fields: `API.tailnetSession` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 30–44 | tailnetError — Host Tailnet control; declarations/fields: `tailnetError` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 45–59 | API.apiTailnetSettings — Host Tailnet control; declarations/fields: `API.apiTailnetSettings` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 60–85 | API.apiTailnetHost — Host Tailnet control; declarations/fields: `API.apiTailnetHost` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 86–113 | Operator Project enrollment policy mutation; declarations/fields: `API.apiTailnetEnrollment` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 114–246 | Own Project original network selection and authorized observation; declarations/fields: `API.apiTailnetOptions`, `API.authorizeProjectTailnet`, `parseProjectTailnetMutation`, `API.admitProjectTailnetDispatch`, `API.confirmProjectTailnetDispatch`, `API.admitProjectTailnetRequest`, `API.apiProjectTailnet` |

<a id="coverage-9a59f58be692"></a>

## [internal/web/api/terminal_registry.go](../../../../../internal/web/api/terminal_registry.go)

Committed ddd8715b source map. Cohesive functions keep one owner; a dependency/reference does not confer authority over the referenced state.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 1–23 | Package/import/build-tag/embed/comment scaffolding for this file’s primary responsibility |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 24–49, 54–98, 213–241 | Human terminal identity/metadata and view-bound lifecycle tracking; declarations/fields: `TerminalID`, `newTerminalID`, `TerminalCreationScope`, `TerminalView`, `terminalDTO`, `reservedTerminal`, `terminalMetadata`, `terminalSessionMutation`, `parseTerminalSessionAction`, `validTerminalSessionMutation`, `API.CloseTerminals`, `API.dropTerminalPeer`, `API.unregisterTerminalPeer`, `peerIDInUse` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 50–53 | validTerminalGeometry — Interactive attachment; declarations/fields: `validTerminalGeometry` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 99–104 | validTerminalOrigin — Interactive attachment; declarations/fields: `validTerminalOrigin` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 105–118 | validSecFetchHeaders — Interactive attachment; declarations/fields: `validSecFetchHeaders` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 119–132 | checkTerminalRequestHeaders — Interactive attachment; declarations/fields: `checkTerminalRequestHeaders` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 133–142 | Record, DTO or interface contract terminalHandshake for Interactive attachment; declarations/fields: `terminalHandshake` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 143–156 | readTerminalHandshake — Interactive attachment; declarations/fields: `readTerminalHandshake` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 157–160 | validHandshakeDimensions — Interactive attachment; declarations/fields: `validHandshakeDimensions` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 161–167 | validHandshakeAction — Interactive attachment; declarations/fields: `validHandshakeAction` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 168–172 | validHandshakeRepository — Interactive attachment; declarations/fields: `validHandshakeRepository` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 173–176 | refuseTerminal — Interactive attachment; declarations/fields: `refuseTerminal` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 177–195 | pumpTerminalInput — Interactive attachment; declarations/fields: `pumpTerminalInput` |
| [S05](../../slices/spaces-and-terminals.md#s05-interactive-attachment) / active | 196–212 | pumpNativeToExtension — Interactive attachment; declarations/fields: `pumpNativeToExtension` |


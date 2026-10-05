# Browser and design

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

## appliance/forgejo/templates/admin/auth/edit.tmpl

Observed size: 449 lines, including tests where embedded. Separate existing provider-specific form sections; preserve the native entry name, actions, input IDs/names and common form owner.

- `frontend/forgejo/templates/admin/auth/edit.tmpl` — Native edit entry, common source identity/status fields, one form, CSRF, submit/delete and tips/modal wiring.
- `frontend/forgejo/templates/admin/auth/edit_ldap.tmpl` — Existing LDAP/direct-LDAP bind, search, group, attributes and sync fields.
- `frontend/forgejo/templates/admin/auth/edit_smtp.tmpl` — Existing SMTP source fields.
- `frontend/forgejo/templates/admin/auth/edit_pam.tmpl` — Existing PAM source fields.
- `frontend/forgejo/templates/admin/auth/edit_oauth.tmpl` — Existing OAuth2 provider URLs, scopes, claims, tenant and quota mapping fields.

Evidence: admin/auth/edit.tmpl:10-23 common form; 24-183 LDAP/DLDAP; 184-245 SMTP; 246-265 PAM; 266-402 OAuth2; 404-449 shared status/actions/tips/modal.

Open detail: Go template invocations do not inherit lexical $cfg variables. Pass the same admitted native context and provider config explicitly; update the actual Forgejo payload inventory and existing source/browser contracts.

## appliance/forgejo/templates/repo/settings/options.tmpl

Observed size: 808 lines, including tests where embedded. The 808-line entry owns several already distinct native forms. Extract complete sections and confirmation markup, leaving native authorization branches and form ownership intact.

- `frontend/forgejo/templates/repo/settings/options.tmpl` — Native entry, layout and conditional composition of existing settings sections.
- `frontend/forgejo/templates/repo/settings/options_repository.tmpl` — Repository name, description, visibility/template fields and avatar form.
- `frontend/forgejo/templates/repo/settings/options_federation.tmpl` — Existing FederationEnabled-gated following-repositories form.
- `frontend/forgejo/templates/repo/settings/options_mirrors.tmpl` — Existing pull/push mirror conditions, forms and local mirror-selection variables.
- `frontend/forgejo/templates/repo/settings/options_trust.tmpl` — Existing signature trust-model radios and help.
- `frontend/forgejo/templates/repo/settings/options_maintenance.tmpl` — Existing fsck/health, code/issue/statistics indexer administration forms.
- `frontend/forgejo/templates/repo/settings/options_danger.tmpl` — Existing conversion, transfer, archive, delete and wiki action controls.
- `frontend/forgejo/templates/repo/settings/options_modals.tmpl` — Existing conversion/transfer/delete/wiki/archive confirmation modals and push mirror sync include.

Current `f7e9cf9d` allocation, source-challenged by the primary reviewer and coordinator:

| Current lines | Exact target concern |
| --- | --- |
| 1–2, 575–576 | Entry owns layout head, opening/closing wrapper and footer; compose sections in their current order with the same root context |
| 3–70 | `options_repository.tmpl`: repository and avatar forms |
| 72–94 | `options_federation.tmpl`: whole gated federation form |
| 96–348 | `options_mirrors.tmpl`: whole mirror section, including its local variables and forms |
| 350–393 | `options_trust.tmpl`: whole signature trust section |
| 394–463 | `options_maintenance.tmpl`: whole `.IsAdmin` branch and maintenance section |
| 465–574 | `options_danger.tmpl`: whole `.Permission.IsOwner` branch and action controls |
| 578–808 | `options_modals.tmpl`: whole owner branch, confirmation forms and push-mirror include |

Keep mirror variables inside their owning child: Go template invocations do not inherit lexical variables. Preserve authorization branches, native entry, modal IDs, POST action values and CSRF tokens. Update the actual payload inventory and tests to consume the complete child-template closure. These splits preserve every deliberately overridden page and its current design.

## assets/branding/forgejo/components-toolbar.css

Observed size: 476 lines, including tests where embedded. Split by existing selector ownership; canonical styles remain assets rather than becoming a second frontend theme.

- `assets/branding/forgejo/components-toolbar.css` — Preserved public stylesheet entry, composing the canonical concern sheets in reviewed cascade order.
- `assets/branding/forgejo/components-toolbar-layout.css` — Toolbar shell, native search/action inputs, disclosure controls and their responsive rules.
- `assets/branding/forgejo/components-navigation.css` — Native context switcher, jump menu and overflow-menu tabs, including active/responsive states.
- `assets/branding/forgejo/components-repository-toolbar.css` — Native repository compact-size button variants and joined clone/folder controls.

Current `f7e9cf9d` allocation, source-challenged by the primary reviewer and coordinator:

| Current lines | Exact target concern |
| --- | --- |
| 1–2 | Public entry imports layout, navigation, then repository-toolbar in that order |
| 3–79, 368–392 | `components-toolbar-layout.css`: toolbar/search bases and their generic responsive declarations |
| 80–247, 394–417 | `components-navigation.css`: context switcher, tabs and their responsive declarations |
| 248–366, 421–476 | `components-repository-toolbar.css`: native controls and repository variants |

Distribute the existing 700px media shell at 367/419 into balanced wrappers around its owning declarations. Keep navigation responsive rules after navigation bases: putting the entire media block in the earlier layout sheet would let later base rules override the current narrow context-switcher/menu behavior. Preserve current selectors, values, breakpoints and winning declarations.

Retain the installed public URL and stage every relative import through the real payload inventory. `component-boundaries.test.ts:16–27` strips imports; rebind it to the defining stylesheet closure in this same cascade order. Existing repository-code marker tests must likewise follow their real owners. Source fit does not establish rendered equivalence; retain the existing browser checks for the later cutover.

## assets/branding/forgejo/repository-code.css

Observed size: 465 lines, including tests where embedded. Split the current browser, editing/compare and native Actions selectors, preserving upstream editor/polling/dispatch ownership.

- `assets/branding/forgejo/repository-code.css` — Preserved public stylesheet entry composing existing code-workflow sheets.
- `assets/branding/forgejo/repository-code-browser.css` — Finder/file listing, branch/tag/commit history, quickstart/forks/people and landing sidebar/README presentation.
- `assets/branding/forgejo/repository-code-editing.css` — Existing file headers, editor chrome, branch picker and compare/diff framing.
- `assets/branding/forgejo/repository-code-actions.css` — Existing native Actions lists/filter/dispatch and the shared narrow-screen editor, branch-picker and Actions overrides that follow the base sheets.

Current `f7e9cf9d` allocation: browser owns 3–194, 288–324 and 401–465; editing owns 195–287; Actions and shared responsive overrides own 325–400. Keep the complete media block in that last concern. The public entry imports browser, editing, then Actions so shared narrow overrides follow both bases. Preserve the browser sheet's existing sidebar media block and selector specificity. The primary reviewer and coordinator checked the actual selector bodies; independent final allocation review remains recorded in G08.

Retain relative imports, native selectors and installed payload paths. Rebind existing fixtures to the actual defining stylesheet closure. This allocation changes ownership only; rendered equivalence remains a later cutover check.

## docs/design/spaces/preview.css

Observed size: 1025 lines, including tests where embedded. The owning README identifies this as superseded 8fe3468 history rather than a current product/design candidate. Do not convert an obsolete fake state machine into a newly maintained module graph.

- **Retirement disposition:** no implementation leaf or replacement module graph in the ideal target; retain historical attribution/evidence separately.

Evidence: docs/design/spaces/README.md:64-70 superseded mockup and historical retention; preview.ts:1-2 design-only/no API/socket/auth/storage/shell; review.ts:19-27 current historical source loading.

Open detail: Retire the superseded preview/review/compiler group coherently when implementation is authorized. Update retained [tsconfig.tools.json:3](../../../design/spaces/tsconfig.tools.json#L3) to remove `review.ts` while keeping `render-sheets.ts`; its old capture driver and historical README must not imply an active design candidate. Source deletion is deferred.

## docs/design/spaces/preview.ts

Observed size: 706 lines, including tests where embedded. The owning README identifies this as superseded 8fe3468 history rather than a current product/design candidate. Do not convert an obsolete fake state machine into a newly maintained module graph.

- **Retirement disposition:** no implementation leaf or replacement module graph in the ideal target; retain historical attribution/evidence separately.

Evidence: docs/design/spaces/README.md:64-70 superseded mockup and historical retention; preview.ts:1-2 design-only/no API/socket/auth/storage/shell; review.ts:19-27 current historical source loading.

Open detail: Retire the superseded preview/review/compiler group coherently when implementation is authorized. Update retained [tsconfig.tools.json:3](../../../design/spaces/tsconfig.tools.json#L3) to remove `review.ts` while keeping `render-sheets.ts`; its old capture driver and historical README must not imply an active design candidate. Source deletion is deferred.

## frontend/spaces/sodaspaces-api.ts

Observed size: 858 lines, including tests where embedded. Move each existing response contract to its current product concern. Keep one definition per DTO and import the actual owner directly.

- `frontend/spaces/sodaspaces-api.ts` — Bounded JSON/object primitives, common identifier admission and SodaRequestError; no forwarding exports for moved contracts.
- `frontend/spaces/sodaspaces-project-response.ts` — Canonical browser CreationProfile/Environment/Detail/OSObservation types and their existing decoders.
- `frontend/spaces/sodaspaces-keys-response.ts` — Saved/profile key and preview response types/decoders.
- `frontend/spaces/sodaspaces-terminal-response.ts` — Terminal identity/metadata types and exact binding/state response admission.
- `frontend/spaces/sodaspaces-factory-response.ts` — Existing factory authority/control/run types, status/detail admission and bounded display text.
- `frontend/spaces/sodaspaces-factory-stream-response.ts` — Existing factory status/output frame admission, cursor accounting and closed reason.
- `frontend/spaces/sodaspaces-inventory-response.ts` — Canonical Space composition (Environment/Detail plus TerminalMetadata and FactoryAuthority/FactoryControl/FactoryRun references), spaceNetwork and existing collection admission; observed inventory and incomplete-page facts remain here.
- `frontend/spaces/sodaspaces-repository-response.ts` — Existing repository-choice/path/pagination admission.

Evidence: sodaspaces-api.ts:4-47 common JSON/object/identifier primitives -> sodaspaces-api.ts; 48-185 profile/environment/OS definitions and admission -> project-response; 186-239 key definitions/preview admission -> keys-response; 240-298 terminal metadata/terminalResponse -> terminal-response.; sodaspaces-api.ts:299-317 FactoryAuthority/FactoryControl/FactoryRun definitions, 332-356 factory display/control command functions, 362-650 factory-field/run/detail admission -> factory-response; 651-703 status/output stream frames -> factory-stream-response.; sodaspaces-api.ts:318-331 Space interface and 357-361 spaceNetwork -> inventory-response; 704-814 actual Space/collection admission -> inventory-response. Space imports Environment/Detail, TerminalMetadata and the canonical Factory records directly; no duplicate DTOs.; sodaspaces-api.ts:815-858 repository choices/path/pagination and request error; repository contracts -> repository-response, SodaRequestError -> original common API owner..

Open detail: Space belongs to the collection/inventory owner and composes the canonical project, terminal and factory records through direct imports. Factory-field admission is reused by the Space decoder rather than recreated. Update every real caller, including installed journeys and Tailnet common JSON imports; final module visibility/signatures require implementation review.

## frontend/spaces/sodaspaces-factory.ts

Observed size: 529 lines, including tests where embedded. Extract screen setup and pure presentation from the existing read-only watcher without creating another run controller or input-capable transport.

- `frontend/spaces/sodaspaces-factory.ts` — One immutable factory run binding, generation/lifetime, read-only socket watch and retirement owner; public mount remains here.
- `frontend/spaces/sodaspaces-factory-screen.ts` — Existing disabled-input xterm construction, font/theme/fit and OSC behavior.
- `frontend/spaces/sodaspaces-factory-view.ts` — Reuse existing stateless view owner; absorb watch presentation/title/status and menu projection from the class.

Evidence: sodaspaces-factory.ts:40-42 read-only ownership contract; 123-188 view/menu projection; 236-304 status/font/theme/xterm; 305-475 current watch/socket/lifetime; 478-529 mount admission.

Open detail: Keep generation/cursor/retirement in the same owner. Screen helpers receive the existing owner callbacks/resources; no generic terminal/factory lifecycle abstraction is implied.

## frontend/spaces/sodaspaces-project.ts

Observed size: 1895 lines, including tests where embedded. Separate the current journey/settings projections and existing request/read/mutation capabilities. Keep original-target identity, epoch cancellation and dispatch admission in the same component; reuse environment/project/network view modules.

- `frontend/spaces/sodaspaces-project.ts` — SodaProjectControls state, original native binding, epoch/lifetime gate, Lit setup/render coordination and public mount.
- `frontend/spaces/sodaspaces-project-request.ts` — Existing native transport request preparation, error-code admission and authorization-loss callback.
- `frontend/spaces/sodaspaces-project-journey-view.ts` — Existing welcome/unavailable/join-failed/join/existing/configure journey projections.
- `frontend/spaces/sodaspaces-project-settings-view.ts` — Existing repository identity/tab/environment/access/network view composition, reusing existing concern views.
- `frontend/spaces/sodaspaces-project-refresh.ts` — Existing account/environment/profile/detail refresh sequencing and admitted read results.
- `frontend/spaces/sodaspaces-project-connection.ts` — Existing joined-account SSH connection observation and deliberate copy action.
- `frontend/spaces/sodaspaces-project-mutations.ts` — Existing dispatch, result checks, uncertain outcome explanations and completion observation.
- `frontend/spaces/sodaspaces-project-network.ts` — Existing Tailnet option/observation/change branches and confirmation admission.
- `frontend/spaces/sodaspaces-project-access.ts` — Existing explicit keyless/keyed Join, profile/saved key review, save and Apply actions.
- `frontend/spaces/sodaspaces-project-runtime.ts` — Existing OS read, observed runtime state, explicit Start/Stop and shared-impact confirmation.

Evidence: sodaspaces-project.ts:1-37 existing view/response imports; 172-347 component state/configure; 373-427 epoch/command/create gate; 430-780 journey; 802-1034 settings views; 1036-1080 join/copy; 1110-1494 reads; 1495-1687 mutations; 1688-1863 network/OS/lifecycle/keys; 1864-1895 disposal/mount.

Open detail: These are module boundaries, not permission for separate mutable controllers, new policy, or parallel request models. Exact exported helper signatures and post-extraction line sizes still need implementation review.

## frontend/spaces/sodaspaces-terminal.ts

Observed size: 964 lines, including tests where embedded. Separate existing renderer, attachment protocol and explicit native End operation while retaining one original actor and imperative lifetime owner.

- `frontend/spaces/sodaspaces-terminal.ts` — SodaTerminal original account/binding/generation, component lifecycle, visibility and public exact-locator mount.
- `frontend/spaces/sodaspaces-terminal-attachment.ts` — Existing metadata inspect, reservation/create/attach handshake, native-generation frame dispatch and socket callbacks.
- `frontend/spaces/sodaspaces-terminal-screen.ts` — Existing xterm/font/theme/input/fit/geometry/minimum-size and renderer readiness code.
- `frontend/spaces/sodaspaces-terminal-actions.ts` — Existing explicit End confirmation, target-preserving HTTP action and unconfirmed-outcome handling.
- `frontend/spaces/sodaspaces-terminal-view.ts` — Reuse existing stateless view; absorb terminal presentation labels/menu projections rather than making another rendered terminal owner.

Evidence: sodaspaces-terminal.ts:54-55 immutable account/no rendering effects; 127-152 configure; 157-273 presentation/confirmation; 274-404 generation/detach/End; 405-508 send/geometry; 550-856 metadata/renderer/reservation/socket; 869-964 visibility/dispose/mount.

Open detail: There must still be one generation and one socket/screen owner. Preserve bounded queues, readiness gating, input focus and no replay; helper extraction must not duplicate those state machines.

## frontend/spaces/sodaspaces-workspace-view.ts

Observed size: 473 lines, including tests where embedded. Extract complete stateless Lit view functions by existing screen concern; keep view data as presentation-only projections.

- `frontend/spaces/sodaspaces-workspace-view.ts` — Welcome and workspace introduction, plus existing menu primitive.
- `frontend/spaces/sodaspaces-repository-picker-view.ts` — Existing repository search, choice, empty result and pagination presentation.
- `frontend/spaces/sodaspaces-session-navigation-view.ts` — Existing session tabs and project/session navigation rows.
- `frontend/spaces/sodaspaces-factory-navigation-view.ts` — Existing factory run rows and watched-run navigation presentation.
- `frontend/spaces/sodaspaces-terminal-dialog-view.ts` — Existing Rename/New terminal form presentation and callbacks.

Evidence: sodaspaces-workspace-view.ts:7-81 welcome/intro; 82-192 repository picker; 193-200 menu; 201-319 tabs/project navigation; 320-374 factory rows; 375-473 rename/create.

Open detail: Update importing owners directly; do not keep a forwarding export barrel solely to preserve old source imports.

## frontend/spaces/sodaspaces-workspace.ts

Observed size: 3125 lines, including tests where embedded. Split existing presentation and resource concerns while preserving one SodaSpaces component, one actor/generation/lifetime gate, and flat live terminal hosts. New/Rename and locator restoration form one current terminal concern; the mounted project-controls cache remains with the component that owns its actor and retirement.

- `frontend/spaces/sodaspaces-workspace.ts` — Single SodaSpaces actor/epoch/lifetime/availability and component-state owner, constructor/createRenderRoot and derived-state getters/projections, public mount, markViewed/live admission, invalidation/disposal, and existing mounted project-controls cache/presentation/event delegation. Keep managementAdmitted/mountProject/refreshMountedProject/showManagement/presentManagement together here; stateless helpers receive admitted current projections.
- `frontend/spaces/sodaspaces-workspace-types.ts` — Move current WorkspaceContext, Slot, Row, PaneSession, FactoryWatch and Creation records to their single shared definition.
- `frontend/spaces/sodaspaces-workspace-measurement.ts` — Existing WorkspaceMeasurement subscriptions and owner-triggered geometry/minimum measurements.
- `frontend/spaces/sodaspaces-workspace-shell-view.ts` — Current workspace frame, status banners, canvas/navigation visibility and first-terminal introductions.
- `frontend/spaces/sodaspaces-workspace-toolbar-view.ts` — Current native/page toolbar, project label, sidebar/back/drawer/management disclosures.
- `frontend/spaces/sodaspaces-workspace-setup.ts` — Existing repository search/cursors, welcome/configure journey, selected repository and cancel/back flow.
- `frontend/spaces/sodaspaces-workspace-navigation.ts` — Current authorized project/session rows, query filters, unread/attention ordering and explicit selectProject; original owner remains responsible for project selection, back and management admission.
- `frontend/spaces/sodaspaces-workspace-factory.ts` — Existing factory watch selection, immutable watch context, display and command routing.
- `frontend/spaces/sodaspaces-workspace-pane-view.ts` — Current pane chrome, keyed tabs, overflow/move/drop targets, pane switcher/menu/layout-help projections. Receive original-owner focus/split/maximize/consolidate callbacks; do not own layout state.
- `frontend/spaces/sodaspaces-workspace-focus.ts` — Current workspace key/click/menu/focus handling and session-switcher focus restoration.
- `frontend/spaces/sodaspaces-workspace-inventory.ts` — Existing bounded collection refresh/pagination, incomplete facts, slot metadata reconciliation and journey synchronization.
- `frontend/spaces/sodaspaces-workspace-layout.ts` — Existing projection/split/move/divider/sidebar keyboard operations, toggleSidebar/toggleMaximizedPane/consolidatePanes, and load/persist calls into the existing pure layout owner. Original owner performs admitted state/focus/menu updates.
- `frontend/spaces/sodaspaces-workspace-drawer.ts` — Current native-navigation handoff, exact original entry, safe repository path and stale-request checks.
- `frontend/spaces/sodaspaces-workspace-terminal-hosts.ts` — Existing visibleSlot projection, stable flat DOM host mounting, exact locator/metadata/observation routing and geometry display; original-owner epoch/lifetime callbacks admit all observations.
- `frontend/spaces/sodaspaces-workspace-terminals.ts` — One existing terminal concern: exact saved/existing locator restoration and selection, creation eligibility/context/form admission, deliberate New and Rename, destination-pane admission and uncertain result handling. Reuse existing dialog presentation functions; no invented session inventory or second actor/lifetime owner.

Evidence: sodaspaces-workspace.ts:60-104 and 106-110 current shared records -> workspace-types; factoryWatchLimit105 -> workspace-factory; validName111 -> workspace-terminals; drawerRepositoryPart113-122 -> workspace-drawer; sodaWorkspaceInit124-139 remains one original workspace owner, passed through existing callbacks; 140-183 existing WorkspaceMeasurement -> workspace-measurement; 185-186 flat-host/no render-effects contract retained; 261-315 original binding/actor/epoch/lifetime/slots/watch/project cache state -> workspace.ts.; sodaspaces-workspace.ts:316-430 constructor/createRenderRoot, activeSurface/selectedSpace/setupScreen/welcomeScreen, pageReadyForFirstTerminal/spaceReadyForFirstTerminal/projectRunnable/firstTerminal/inventoryRecovery/workspaceIntro, projectState/projectStatus, selected/compact/projection/locator/paneMinimum/tabHeight remain original-owner construction and derived-state projections. Stateless view/layout helpers receive these currently admitted values; no additional module or state owner.; sodaspaces-workspace.ts:431-561 configure/measure/project event callbacks -> original owner plus existing measurement concern; 562-934 shell projections -> shell-view; 935-1078 toolbar projections -> toolbar-view; 1080-1295 repository search/setup/configure flow -> setup.; sodaspaces-workspace.ts:1296-1302 selectProject -> navigation through original-owner project/back/management callbacks; 1303-1315 toggleSidebar/toggleMaximizedPane/consolidatePanes -> layout; 1316-1361 showPaneSwitcher/onFocusPaneChange/renderPaneSwitcher/renderPaneLayoutExtras/renderPaneSplitHelp/renderPaneMenu/paneActions -> pane-view through original-owner layout/focus callbacks.; sodaspaces-workspace.ts:1362-1371 visibleSlot -> terminal-hosts; 1372-1385 markViewed remains original-owner epoch/readRequested/unread retirement; 1386-1574 row attention/navigation -> navigation; 1576-1677 watches -> factory; 1678-1880 pane chrome -> pane-view.; sodaspaces-workspace.ts:1882-1934 creationEligible/creationExplanation/creationContext/creationDisabled/onCreationEnvironment/onCreationName/cancelCreation/creationForm -> terminals; existing renderCreation receives presentation data and original-owner draft/create/focus callbacks. 1935-2037 key/click/menu/focus/back -> focus with original-owner state updates.; sodaspaces-workspace.ts:2038-2098 defaultTerminalName/creationSpace/pickCreationSpace/pageBlocksNewTerminal/newTerminalBlocked/focusCreationDialog/newTerminal -> terminals; 2099-2101 live remains original owner; 2102-2312 bounded collection reads/paging/reconciliation -> inventory.; sodaspaces-workspace.ts:2313-2334 loadLayout/persist -> layout; 2335-2392 native handoff -> drawer; 2394-2497 exact restoration/open/confirmed End -> terminals; 2499-2717 stable host mount/locator/observation/metadata/focus/geometry -> terminal-hosts; 2718-2855 arrange/focus/split/move/divider/sidebar/tab operations -> layout with original-owner admitted state updates.; sodaspaces-workspace.ts:2856-2944 createAdmitted/createSpaceReady/openCreatedSlot/createTerminal/openExistingBlocked/existingOrNewEntry/openExisting -> terminals; 2945-2999 managementAdmitted/mountProject/defaultManagementMode/refreshMountedProject/focusConfigureIfNeeded/showManagement/presentManagement remain original owner; 3000-3043 renameAdmitted/applyRenamedMetadata/rename -> terminals; 3044-3125 lifecycle/disposal/mount remain original owner..

Open detail: Private methods currently read and write the original class state: binding/actor/epoch/lifetime, slots/watches/projects/layout, creation/creating/renaming, selection/view and focus. File targets are proposed responsibility owners, not complete helper APIs. Extract stateless views and the existing measurement controller first; each later helper must identify the exact snapshot/read inputs, original-owner admission callbacks and returned observations/state changes, preserving current checks before and after await. Keep markViewed/live and mounted-project cache/lifecycle in the original owner. Do not implement mixins, forwarding barrels, duplicate controllers, a generic request bus, new restoration policy or alternative navigation. Concrete signatures and final line sizes require implementation review.

## frontend/tailnet/soda-tailnet-page.ts

Observed size: 836 lines, including tests where embedded. Separate already distinct native observation, host/enrollment actions and pure Lit sections; keep private secrets/drafts and authorization retirement in the original page owner.

- `frontend/tailnet/soda-tailnet-page.ts` — Single configured native transport, original authorization lifetime, pending confirmation, draft ownership, component mount and retirement.
- `frontend/tailnet/soda-tailnet-observation.ts` — Existing bounded request/refresh and admitted native settings reconciliation.
- `frontend/tailnet/soda-tailnet-actions.ts` — Existing explicit host/enrollment mutation selection, payload/result admission and unconfirmed outcome messaging.
- `frontend/tailnet/soda-tailnet-host-view.ts` — Existing status/auth link, exit-node preference choices, peer list and host controls.
- `frontend/tailnet/soda-tailnet-enrollment-view.ts` — Existing enrollment summary/form/admission presentation.
- `frontend/tailnet/soda-tailnet-confirmation-view.ts` — Existing pending confirmation/reconnect projection and consequence labels.

Evidence: soda-tailnet-page.ts:7-74 current helpers/types; 76-190 transport/state/lifetime; 191-286 observation; 287-457 mutations/confirmation; 458-558 handlers; 559-581 confirmation; 582-697 host view; 698-809 enrollment view; 810-836 composition/mount.

Open detail: All scope/revision/native authorization behavior remains existing. Do not create a second settings store, add refresh retries, or move secrets into presentation modules.

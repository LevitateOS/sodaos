# Spaces and terminals

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## S01 Authorized inventory

- **Validity review:** [S01 record](../reviews/S01.md) — actual reviewed scope, findings and pending completion dimensions.

Build the current actor's bounded Spaces read model from Project membership, native observations and recorded factory facts.

- **Entrypoints:** GET /api/spaces; apiSpaces; loadSpacesInventory; inspectSpaceAuthority/native/terminals/factory.
- **Owned data:** Transient authorized rows; next cursor/completeness/degraded flags; observed current-writer metadata.
- **Authority:** Verified native actor; membership/repository authority; uncertainty cannot disclose session metadata or grant elevation.
- **Dependencies:** [P01](projects.md#p01-repository-association-and-creation); [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [P07](projects.md#p07-checkout-allocation-and-preparation); [S04](#s04-human-terminal-lifecycle); [S06](#s06-factory-activity-presentation); Factory stored views; Native session authority.
- **Source files:** [internal/web/api/spaces.go:69](../../../../internal/web/api/spaces.go#L69); [internal/web/api/spaces.go:358](../../../../internal/web/api/spaces.go#L358); [internal/web/api/spaces_inventory.go:42](../../../../internal/web/api/spaces_inventory.go#L42).
- **Tests:** [tests/frontend/spaces-api.test.ts:109](../../../../tests/frontend/spaces-api.test.ts#L109) — Source-only parser assertions: incomplete remains incomplete; mismatched/duplicate rows rejected; [tests/frontend/spaces-api.test.ts:119](../../../../tests/frontend/spaces-api.test.ts#L119) — Source-only parser assertions: degraded authority rows cannot retain terminal metadata or administrator elevation.
- **Unclear boundaries:** Owns the read model, not stored Project, terminal or factory truth. Recorded factory summaries are not live-process proof. Inventory failure/incompleteness must not become guessed empty/new-product state.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## S02 Workspace lifetime and navigation

Retain one Soda workspace and flat execution-view owners inside Fountain's persistent host while native pages and detail selections navigate.

- **Entrypoints:** prepareExtensionMount; SodaSpaces mount/disconnect/dispose; generic Fountain persistent panel contribution.
- **Owned data:** Mounted owner references; workspace generation/lifetime; current navigation/detail selection.
- **Authority:** Fountain supplies verified extension context; Soda owns decided workspace/session interactions; actor change retires private views.
- **Dependencies:** [S01](#s01-authorized-inventory); [S03](#s03-views-layout-and-restoration); [S04](#s04-human-terminal-lifecycle); [S05](#s05-interactive-attachment); [S06](#s06-factory-activity-presentation); Fountain generic persistent host; Lit.
- **Source files:** [frontend/spaces/soda-extension.ts:28](../../../../frontend/spaces/soda-extension.ts#L28); [frontend/spaces/sodaspaces-workspace.ts:187](../../../../frontend/spaces/sodaspaces-workspace.ts#L187); [docs/product/spaces.md:14](../../../product/spaces.md#L14).
- **Tests:** [tests/frontend/persistent-panel.test.ts:81](../../../../tests/frontend/persistent-panel.test.ts#L81) — Source-only browser fixture assertions: same Lit/xterm owner, socket and exact target retained through native navigation.
- **Unclear boundaries:** Owns browser mount/navigation lifetime, not native session survival or lease termination. S03 owns saved arrangement; S05 owns attachments. Mounting/navigation must not silently Create, Join, Start or retarget an existing execution.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

- **Validity review:** [S02 audit record](../reviews/S02.md).

## S03 Views, layout and restoration

Validity review: [S03 record](../reviews/S03.md). Its recorded scope and completion dimensions govern audit status.

Manage panes/tabs/splits and disposable actor-scoped session locators while preserving exact view owners and original execution targets.

- **Entrypoints:** selectTab/hideTab/forgetEntry/splitPane/moveTab/resizeSplit; parseLayout/serializeLayout; restoreLocators.
- **Owned data:** Disposable layout tree/selection/ratios; owner keys and existing-session locators; actor-scoped browser storage.
- **Authority:** Current workspace actor; restoration accepts validated exact locators and does not supply native authority.
- **Dependencies:** [S02](#s02-workspace-lifetime-and-navigation); [S01](#s01-authorized-inventory); [S04](#s04-human-terminal-lifecycle); [S05](#s05-interactive-attachment); Browser session storage.
- **Source files:** [frontend/spaces/sodaspaces-layout.ts:74](../../../../frontend/spaces/sodaspaces-layout.ts#L74); [frontend/spaces/sodaspaces-layout.ts:326](../../../../frontend/spaces/sodaspaces-layout.ts#L326); [frontend/spaces/sodaspaces-workspace.ts:2206](../../../../frontend/spaces/sodaspaces-workspace.ts#L2206).
- **Tests:** [tests/frontend/spaces-layout.test.ts:27](../../../../tests/frontend/spaces-layout.test.ts#L27) — Source-only layout assertions: locator promotion preserves owner/selection; serialized state carries no effects; [tests/frontend/spaces-layout.test.ts:39](../../../../tests/frontend/spaces-layout.test.ts#L39) — Source-only assertions: Hide retains locator; Forget removes it; unsent New cannot restore as execution.
- **Unclear boundaries:** Owns presentation arrangement and locators only. Shells, leases and process lifetime remain elsewhere. Corrupt/obsolete cache refusal must not create a replacement native session or imply native absence.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## S04 Human terminal lifecycle

Validity review: [S04 record](../reviews/S04.md). Its recorded scope and completion dimensions govern audit status.

Reserve an exact managed terminal ID, explicitly create/inspect/rename/end its native session, and preserve it independently of a browser attachment.

- **Entrypoints:** Terminal reservation; terminal-sessions mutation; Host.TerminalCreate/Inspect/End; project-terminal helper.
- **Owned data:** Managed terminal ID/name/account/native-incarnation metadata; native tmux/systemd supervised process lifetime.
- **Authority:** Current verified native actor and stable P03 membership; explicit targeted End; member terminal differs from factory role execution.
- **Dependencies:** [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [S05](#s05-interactive-attachment); systemd; tmux; Podman; I09 where a provider consumer uses the terminal.
- **Source files:** [internal/web/api/extension_terminal.go:209](../../../../internal/web/api/extension_terminal.go#L209); [rust/soda-project-terminal/src/term.rs:869](../../../../rust/soda-project-terminal/src/term.rs#L869); [docs/reference/terminal.md:38](../../../reference/terminal.md#L38).
- **Tests:** [tests/frontend/terminal.test.ts:310](../../../../tests/frontend/terminal.test.ts#L310) — Source-only browser fixture assertions: separate End confirmation, one original-ID request, native generation and confirmed cleanup before locator removal.
- **Unclear boundaries:** Owns human managed session lifecycle, not every native Project process. End does not stop Project or delete files. Browser detachment and hidden views belong to S05/S03; factory role views cannot inherit human terminal End authority.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## S05 Interactive attachment

Validity review: [S05 record](../reviews/S05.md). Its recorded scope and completion dimensions govern audit status.

Attach an authorized browser to the exact existing native terminal and transport bounded input/output, controls and liveness checks.

- **Entrypoints:** Terminal WebSocket stream; extensionTerminalCurrent/open native attachment/pump controls; terminal input/output pumps.
- **Owned data:** Transient peer/writer claim; socket/generation/heartbeat; bounded IO queues and renderer attachment state.
- **Authority:** Native actor/repository/membership continuity throughout stream; focused visible view controls input; expired access detaches.
- **Dependencies:** [S04](#s04-human-terminal-lifecycle); [S02](#s02-workspace-lifetime-and-navigation); [S03](#s03-views-layout-and-restoration); Native extension session authority; WebSocket; PTY/tmux transport.
- **Source files:** [internal/web/api/extension_terminal.go:138](../../../../internal/web/api/extension_terminal.go#L138); [internal/web/api/extension_terminal.go:285](../../../../internal/web/api/extension_terminal.go#L285); [internal/web/api/terminal_registry.go:177](../../../../internal/web/api/terminal_registry.go#L177); [frontend/spaces/sodaspaces-terminal.ts](../../../../frontend/spaces/sodaspaces-terminal.ts).
- **Tests:** [tests/frontend/terminal.test.ts:358](../../../../tests/frontend/terminal.test.ts#L358) — Source-only browser fixture assertions: native focus and hidden view do not forward input; focused visible view may send; [internal/web/api/extension_test.go:214](../../../../internal/web/api/extension_test.go#L214) — Source-only transport test definitions: terminal upgrade forwards only protocol and verified authority, not browser credentials.
- **Unclear boundaries:** Owns attachment lifetime and writer access, not native session creation/End. Exact reattachment differs from replacement execution. Protocol/renderer tests remain separate from installed WebSocket-to-native user behavior.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

## S06 Factory activity presentation

Validity review: [S06 record](../reviews/S06.md). Its recorded scope and completion dimensions govern audit status.

Present authorized factory run/status/output and route explicit factory controls while retaining factory ownership of runs and attempts.

- **Entrypoints:** Spaces factory rows; factory status/output routes; SodaFactoryWatch mount/dispose; existing pause/cancel/takeover controls.
- **Owned data:** Transient run display/frame/cursor state; view locator and exact run/process binding; observed control availability.
- **Authority:** Current factory/repository role authority; read-only role output view; controls use factory authorization rather than human terminal input.
- **Dependencies:** [S01](#s01-authorized-inventory); [S02](#s02-workspace-lifetime-and-navigation); [S03](#s03-views-layout-and-restoration); Factory coordinator/run ledger; Native host factory status/output.
- **Source files:** [internal/web/api/spaces.go:35](../../../../internal/web/api/spaces.go#L35); [internal/web/api/factory_output.go](../../../../internal/web/api/factory_output.go); [frontend/spaces/sodaspaces-factory.ts:43](../../../../frontend/spaces/sodaspaces-factory.ts#L43); [docs/product/spaces.md:56](../../../product/spaces.md#L56).
- **Tests:** [tests/frontend/factory-view.test.ts:26](../../../../tests/frontend/factory-view.test.ts#L26) — Source-only parser assertions: status bound to exact run/process; mismatched identity/shape refused; [tests/frontend/factory-view.test.ts:54](../../../../tests/frontend/factory-view.test.ts#L54) — Source-only output assertions: exact bytes and server cursors; malformed cursor/frame rejected.
- **Unclear boundaries:** Owns presentation state only; factory owns durable execution/control state. A status display is not native liveness proof. Hide or lost view does not End, revoke lease or cancel attempt; factory controls remain distinct from S04.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.

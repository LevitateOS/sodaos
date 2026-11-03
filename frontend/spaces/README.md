# Shared Spaces browser source

This is authored TypeScript and CSS for the Soda Forgejo extension package, not a
Forgejo public-assets directory. Fountain owns browser identity, navigation and
the lifetime of the native page and persistent workspace-panel contributions.

`soda-factory-feed-host.ts` supplies canonical feed identity, catalogue and native
navigation. Its optional application-origin inventory read now accepts the shared
feed's cancellation signal. The shared component owns its captured runtime
generation, request retirement and polling; Fountain still owns account admission.

- `sodaspaces-workspace.ts` owns the shared page and panel workspace, locators,
  layout, original-target commands, stable terminal hosts and navigation checks.
- `sodaspaces-project.ts` owns project requests, synchronous admission and key
  drafts.
- `sodaspaces-project-work.ts` resolves the native Project/account into canonical
  Factory scope. A same-binding inventory refresh updates the selected verification
  runs while preserving that scope, reviewed issue/input and unchanged pending
  decisions. Initial discovery uses the latest run selection even if inventory
  changes while identity reads are pending. Removing execution admission removes
  those runs from the view; refresh itself never queues verification or repair.
  It and `soda-place-app.ts` import the shared `factory/work/verification-retry.ts`
  control directly. Native inventory owns discovery; the shared control owns
  retry/repair requests, receipt validation and read-only progress polling.
  Both hosts supply authenticated command scope for durable pending recovery;
  App keeps the shared control mounted with an empty run list so an older saved
  request can be re-admitted by fresh canonical run metadata.
  Project Work and request places also use the shared verification-allowance
  control and pass canonical scope plus authenticated command identity; the
  host no longer defines a separate limits widget or response contract.
  Factory settings also compose the shared repair-funding control with explicit
  canonical transport/repository and authenticated command scope; grant/history
  presentation and exact pending changes are owned by `frontend/factory/`.
- `sodaspaces-factory-settings.ts` accepts a native repository locator from the
  Project binding and resolves it before loading Factory controls. Policy,
  grants, sponsorship, activity and repair use the canonical Factory repository;
  preparation keeps its native environment ID. Context changes abort outstanding
  reads, and resuming hidden controls resolves the repository again. Policy actor
  IDs and token references stay strings, including IDs above the browser's safe
  integer range. This does not convert other workspace or repository-place callers.
  Both settings surfaces compose shared `factory/work/activity.ts`, supplying
  canonical transport/repository, current activity-edit capability and authenticated
  command scope. `soda-factory-links.ts` validates exact native activity links;
  shared presentation and scoped preference recovery stay with Factory.
- `soda-place-view.ts` resolves native repository locators before mounting the
  Knowledge, issues, App and request views. Their Factory components receive a
  separate canonical `factoryRepository`; inventory selection, application
  Project associations and native issue URLs retain the native locator. Pending
  discovery is cancelled on hiding/navigation and retried on resume. Run views
  must echo the exact canonical repository rather than borrowing the caller's ID.
- `soda-place-issues.ts` mounts the shared issue list and issue/Factory-facts
  composition beneath the native header. Its fresh metadata join and scoped
  transport come from `soda-factory-issue-host.ts`; no second issue model exists.
  `soda-places.ts` owns `/feed/work/issues[/{item}]` and the reserved pulls tree.
  Factory links open the issue's Factory section. Existing acceptance/context/
  completion controls mount only after the current primary issue read succeeds
  and are removed on read/access loss. Code-read page admission remains required.
  The same shared composition now supplies the ordinary issue-comment composer
  with fresh canonical account custody. It uses the existing human-comment API;
  draft/pending recovery and rendering stay in `frontend/factory/issues/`.
  Shared PR and issue-creation routes explicitly offer the native page until
  their shared views are implemented. Original native issue/PR URLs stay explicit.
- `soda-place-request.ts` uses the resolved canonical actor/repository for request
  records and provenance. Native picker rows resolve before repository acceptance;
  another repository's link comes from authenticated canonical identity discovery,
  never a native inventory query using a local ID. Request drafts include host
  scope and actor.
- `soda-place-knowledge.ts` composes shared `factory/library/library.ts` with
  canonical transport/repository and the selected item. It validates source-post
  event scope before navigating to Fountain's feed route. Library reads,
  search/history presentation and retired-response guards stay with Factory.
- `sodaspaces-workspace-setup.ts` keeps native picker/Project references separate
  from canonical request/overview references. The workspace verifies Factory
  account discovery against its native inventory actor before mounting the
  composer. Preparation and watch navigation resolve canonical IDs back to native
  Projects. Both workspace and request-page composers use host-scoped drafts;
  canonical actor IDs do not replace native terminal or Project custody.
- `sodaspaces-terminal.ts` owns the original account binding, issued locator,
  xterm renderer and attachment lifetime.
- `sodaspaces-factory.ts` adapts the shared `frontend/factory/watch/` implementation:
  it resolves native Project repositories before canonical Factory run reads,
  supplies the native attachment/socket and adapts Fountain lifecycle validation.
  The Fountain watch context explicitly
  keeps `nativeRepositoryId` for its native read-only output stream handshake;
  closing a pending view prevents late metadata reads or stream attachment.
  A reconnect resolves metadata again, while hiding keeps the attached renderer.
- `*-view.ts` modules provide stateless typed presentation. They do not own
  permission, request or resource authority.
- `sodaspaces-layout.ts` and `sodaspaces-api.ts` define pure layout and bounded
  JSON contracts.
- `sodaspaces-attention.ts` derives bounded lifecycle observations; mounted
  refreshes never create or renew terminals.
- `soda-spaces-entry.ts` is the one shared Workspace owner entry: the spaces
  page and workspace panel declare the same owner and entry, and the host
  mounts a single retained instance driven through its MountHandle.
  Browser fixtures import that same entry directly; `sodaspaces-page.ts`
  remains for the review fixture's page presentation.

`system/containers/extension/extension.json` declares the browser entries.
`scripts/build-soda-extension.ts` checks that declaration against the entries and
their styles, bundles the extension modules, and adds the locked terminal assets
and required notices to the independently installed package. The Forgejo image
payload in `frontend/forgejo/payload.json` owns host branding and
template assets; it does not publish a duplicate copy of the extension UI.

The preview builds and serves the same extension asset package. Generated browser
and terminal assets stay under ignored `.artifacts/`. The Forgejo branding modules
keep their own public Lit bundle, separate from the extension package runtime.

Rendering never creates a terminal, changes native lifetime, provisions access or
owns native forms. Controls retain the original target and bootstrap binding;
server operations authorize access. The [terminal guide](../../docs/reference/terminal.md)
owns native lifetime and exact lookup. Flat terminal hosts are not moved between
keyed template parents, and view updates retain unsent drafts and renderer/socket
identity.

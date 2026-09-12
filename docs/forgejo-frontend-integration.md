# Forgejo customization contract

The [12 September source audit](forgejo-extension-audit.md) maps the current
extension implementation, native-session/hosting limits and a reproduced runner
admission defect. It recommends how to extend the existing owners without a plugin
framework; it is not a new execution plan or installed acceptance receipt.
The [Forgejo extension status](forgejo-extension-status.md) now owns this agent's
follow-up implementation order and progress, independently of the Tailnet workstream.

Use stock Forgejo's frontend throughout; Soda adds the **Sodaspaces** repository
**button and side drawer**, not a new repository tab. See the [short plan](sodaspaces-plan.md)
and [handoff](implementation-status.md). The read-only hook/drawer/context caller
is authored, alongside Go/proxy/config/cookie and backend actor/return handling.
The isolated x86_64 authenticated native-page → Soda browser journey now passes,
including real tab/BFCache transitions. Native x86_64 stage/export checks and a
browser run using exported UI/branding and the built dashboard image also passed at
`ee8091a`. Explicit create/key/join/connection controls subsequently passed bounded
native access proof at `bdbce8e`, followed by separately approved retained cutover,
private-page browser and existing-account SSH/PTY checks. Exact evidence and limits
remain in the handoff, not a general release/installation acceptance.

**Current workspace:** the selected non-modal aside preserves native Forgejo pages.
The shared Lit workspace now supports multiple managed terminals, finite native
reattachment and page/drawer navigation; source/native evidence and remaining CLI
acceptance are in the [handoff](implementation-status.md). Open in drawer has local
coverage and no deployment claim. Modal/backdrop and pre-reattachment descriptions
below are historical, not current behavior to preserve. The six new feature plans
extend this same boundary. The later [native page integration plan](forgejo-soda-pages-plan.md)
now renders the three existing Soda page bodies within Forgejo's own dashboard
and real native shell. Steps 1–4 include connection/logout, normal navigation and
old-URL bookmark bridges; their Go shells are removed. Historical Go HTML placement
below is not the current source contract. The protected APIs retain authority;
no cookies or native authentication are borrowed. Step 5 adds whole-module-graph
cache versioning, duplicate-entry retirement checks and real BFCache/cache evidence.
Its remaining acceptance limits are in the leading handoff; no delivery follows.
The fixed native login-entry constraint and non-atomic logout limit still apply.

## Verified source surface

The selected [Soda settings pages](sodaspaces-plan.md#settings-pages-and-os-selection)
extend repository settings navigation with Sodaspaces and AI automation, and add a
global operator-only Sodarunners destination. The current local
[`repo/settings/navbar.tmpl`](../appliance/forgejo/templates/repo/settings/navbar.tmpl)
already preserves native Actions runners/secrets/variables links. Add Soda links
through the versioned customization mechanism without replacing those workflows.
The three current views use Forgejo's existing dashboard handler and the same
protected Soda namespace/API/Lit boundary. A template link grants no authority.
Repository settings keeps its stable-ID native link and native gates; repository
AI settings remains unimplemented. The shared connection/logout contract uses
schema v9. See the leading handoff for the source candidate, explicit shared-file
handoff status and pending native/provider proof. Public v15 customization documentation was
reviewed; the attempted exact-tag upstream navbar fetch was unavailable, so this
note claims local override inspection rather than fresh upstream source verification.

The inspected **15.0.7** source establishes:

- `modules/templates/base.go::AssetFS` layers `<CustomPath>/templates/` ahead of
  built-ins; absent overrides fall back to native templates.
- `templates/base/head_navbar.tmpl` calls empty `custom/extra_links.tmpl`.
- `templates/repo/header.tmpl` calls empty `custom/extra_tabs.tmpl` with repository
  context. Native permission/unit gates remain upstream-owned.
- Header/footer/body hooks also exist. Prefer dedicated hooks; copy an entire
  template only for a concrete unmet requirement, not the unchanged reference tree.
- The root image/wrapper defaults CustomPath to `/data/gitea`. Soda mounts
  `/var/lib/soda/forgejo` as `/data`; the candidate override location is therefore
  `/var/lib/soda/forgejo/gitea/templates/`. Verify effective configuration and
  reload/restart requirements before any rollout.

Staging supplies native themes/logos and the four Sodaspaces hook/assets under
`gitea/templates/custom/` and `gitea/public/assets/`. This is authored delivery,
not an installed appliance result. The read-only four hook/assets rendered from an
actual exported stage in the isolated fixture; new access controls still need that
updated-payload browser/access run. Forgejo's customized Fomantic subset has native
accessibility/initialization adaptations; it is not a standalone Soda component kit.
Preserve native markup, scripts, form behavior and branding. The source reference
is retained under `.artifacts/research/h01-0f43b9f/v15.0.7/forgejo/`; inspect the exact
selected version/configuration when changing an override.

## Shared presentation components

The existing overrides use the [Soda component contract](../appliance/forgejo/README.md#presentation-component-contract):
small Go template partials for intros, empty content and guest theme controls;
explicit CSS classes for page shells, toolbars, form sections and native list rows.
`custom/header.tmpl` loads component CSS before page-specific styles. Native partials
receive their original context; caller templates retain permission decisions,
translations, form fields and notification replacement hooks. Components accept
plain presentation values, with normal template escaping and native asset prefixes.
The [component audit](forgejo-components-audit.md) records ownership and remaining
validation limits. The page shell is not an embeddable drawer root. Explore tabs
and context menus delegate to native partials; scoped CSS only adapts their visuals.
The guest preference script loads only on anonymous routes with a guest toggle.

This extraction is live-mounted only in the existing local 15.0.7 preview.
Production staging/verifier now has an exact full-presentation inventory; an actual
new candidate build and delivery remain pending. Shared presentation does
not add handlers, authentication or permissions. The later user-requested
[Lit scaffold](lit.md) adds a shared runtime and browser build integration, with
no component ports or eager loading. Use Lit selectively for Soda-owned interactions; do not
migrate native forms/lists wholesale or assume their CSS/scripts cross a shadow root.

These are documented customization mechanisms, **not supported extension APIs**:
[Forgejo's documentation](https://forgejo.org/docs/latest/admin/advanced/customization/)
explicitly calls custom resources/page modifications **unsupported** and warns that
updates may break them without warning. This is stronger than merely lacking a
cross-version compatibility guarantee. Local passing checks establish compatibility
with the tested version, not upstream support or a server-side plugin SDK. Review
the overridden templates, native partial/script boundaries, CSS adapters and local
browser behavior against the exact candidate version before an upgrade.
The [Lit shadow DOM documentation](https://lit.dev/docs/components/shadow-dom/)
explains the CSS and DOM boundaries that any future Lit component must account for.

## Notification bell quick-view investigation

The [implementation plan/evidence](notification-preview-plan.md) sequences compact
native rendering, bell/HTMX integration, focused validation and separately authorized
delivery. Preview source and focused fixture tests are implemented. The authorized
local template reload and signed-in populated/empty rendering checks passed;
remaining native cases, user acceptance and production delivery are tracked there.

Source reviewed at **15.0.7**, tag commit
`d4de9eb2a87c26b402fdd0259e079957f8cd2b4b`. The following records the original
investigation and candidate; current implementation evidence is linked above.
Neither source inspection nor fixture tests are authenticated browser proof.

### Existing native interfaces

- JSON `GET /api/v1/notifications?status-types=unread&limit=5` supplies the data,
  but the API authentication group has OAuth2/token, HTTP signature, Basic and
  optional reverse-proxy methods—not browser-session authentication. It calls
  `AuthShared` with a nil session store. `reqToken` requires an authenticated actor;
  scoped tokens need notification read access. Same-origin cookies alone do not
  make this API usable from a native page. Do not expose a token, copy the session
  to Soda, enable reverse-proxy auth or add a Go proxy to bypass that boundary.
- Native `GET /notifications?div-only=true&q=unread&page=1&perPage=5` uses the
  ordinary signed-in web middleware and renders `user/notification/notification_div`
  with the current user's notification data. Forgejo's own notification JavaScript
  already calls this route to refresh the full-page list. This is an intentional
  HTML-fragment interface, not scraping a full page for data.
- Native `GET /notifications/new` returns only the unread count. Existing
  event-source/polling logic owns the navbar badge; do not create another poller.
- Forgejo already loads HTMX through `features/common-global.js` → `htmx.js`.
  Native redirect handling recognizes `HX-Request` and sends `204` plus
  `HX-Redirect`, avoiding insertion of a login page into the fragment target.

### Recommended candidate: native fragment plus existing HTMX

Use the documented navbar/custom hooks for the bell panel, plus a compact presentation
branch in the already overridden notification fragment. A presentation-only query
flag such as `soda-preview=true` can select that branch; it must not affect native
identity, permissions or notification queries. Source wiring exposes the web context
as `ctx.Context` (`NewTemplateContextForWeb`), whose `FormBool` can read the flag;
the exact template expression still needs a focused rendering check. Preserve the
existing full-page branch for all ordinary/native refresh requests.

On opening, HTMX requests the native fragment with that flag, and swaps the compact
server-rendered markup into one dedicated panel. Keep native links, escaping and
issue icons. Do not parse rows from a full page, fetch JSON through Soda, query the
upstream database, add an upstream executable patch or load a second HTMX copy.
Small local JS may handle opening/closing, focus, in-flight cancellation and stale
responses. The presentation flag is not an authentication or trust signal.

The preview must not reuse full-page IDs (`notification_div`, `notification_table`,
`notification_<id>`) or full-page initialization hooks. Native count updates replace
`#notification_div` using the current page's query; inserting that ID in a dropdown
would let the full-page updater replace it accidentally. Preview content also must
not mount another `role=main`/`.soda-page` shell. Loading the preview issues no
notification-status POST; visiting a native issue link retains Forgejo's own read
behavior. Keep “View all notifications” and the ordinary bell URL as fallback.

### Semantics and remaining checks

The native unread query includes **unread plus pinned**, ordered by notification
`updated_unix DESC`. Its pagination count counts unread only. `perPage=5` therefore
means five native inbox entries, not necessarily five strictly unread entries.
Recommended initial behavior is to match the native inbox and visibly label pinned
entries. Strictly unread-only is a separate decision: filtering the returned batch
can produce fewer than five (or no visible rows despite unread items on later pages).
Do not call that “all caught up” or claim it is the latest five unread notifications.
Do not implement an unbounded page scan or change the upstream query in templates.

Read-only anonymous requests to the existing local preview returned API `401`,
native fragment `303` to `/user/login`, and native fragment with `HX-Request: true`
`204` with `HX-Redirect` to `/user/login`. No login/session was created deliberately,
no existing credentials were used, and no notification or repository was mutated.
Authenticated populated/empty/pinned output, request-flag rendering, native-session
expiry/account switching, desktop/mobile bell behavior, Escape/outside click/focus,
rapid reopen/late responses, failure fallback, subpaths, and coexistence with the
full notifications page remain to test during implementation. No UI code, dependency,
service reload or deployment was changed for this investigation.

Source references (all at the selected tag):
[API middleware](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/shared/middleware.go),
[API routes](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/api.go),
[native routes](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/web.go),
[notification handler](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/user/notification.go),
[ordering](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/models/activities/notification_list.go),
[native updates](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/web_src/js/features/notification.js),
[HTMX](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/web_src/js/htmx.js),
[template context](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/services/context/context.go),
[redirects](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/services/context/base.go).

## Minimal button, drawer and loading candidate

Inspected 15.0.7 provides these concrete primitives:

| Need | Existing source / bounded addition |
| --- | --- |
| Header action | `repo/header.tmpl` has `.repo-header .repo-buttons` and native `ui compact small basic button` styling, but no dedicated hook in that row. Render Soda's hidden button through `custom/footer`, then move only that button into the row. If the row is absent (for example broken/creating repositories), do not mount. No tab or full header override is needed for this candidate. The selector/layout is version-sensitive and needs browser tests. |
| Drawer | Native `<dialog>` styling in `web_src/css/modules/dialog.css` and usage in `templates/repo/branch/list.tmpl`. Use browser `showModal()`/`close()` with scoped right-edge/full-height CSS and a scrolling content region. This is an overlay drawer, not a page-layout rewrite. Fomantic's selected component list does **not** include `sidebar`. |
| Loading | `web_src/css/modules/animations.css` defines `is-loading` and `loading-icon-2px`. Apply to the content/spinner or pending action, not the dialog: it disables pointer events and forces relative positioning. Add actual disabled controls, `aria-busy` and a text status; keep Close usable. No full-page loading screen or fabricated percentage. |
| Forms/status | Native button/form styles and theme variables. Soda controls only its own elements and uses text/value assignment for returned data, not injected provider HTML. |
| Copy SSH | `web_src/js/features/clipboard.js` delegates `[data-clipboard-target]` clicks and reads the input value, so a readonly SSH-command input can use the existing copy/feedback behavior. Display the current host-key fingerprint too. |

`custom/header` can load the scoped CSS; `custom/footer` can emit repository-ID
context/button/dialog plus an external script. Custom files are served by native
`/assets/*` (`modules/public/public.go`, `routers/web/web.go`). Proposed UI payload:
two small custom hook templates plus `sodaspaces.css`/`sodaspaces.js`; no Node build,
new library or upstream source edit. Asset paths must respect `AppSubUrl` and real
staging/configuration. Preserve any operator customizations rather than overwrite them.

Use the browser dialog API, not an assumed exported Forgejo JS API. Native
`modules/modal.ts` and the legacy `.show-modal`/Fomantic handler are different paths;
the latter's initializer binds existing elements. Clipboard is delegated. Test mount
ordering/repeated opens, Escape/backdrop/Close/focus return, narrow screens, native
navigation and upgrades. Keep the content wrapper filling the dialog so native
backdrop-click handling does not treat an empty internal area as an outside click.

### Source checks

With the existing pinned Go/Node tools and prepared root Bun workspace dependencies, local
checks are `go test -mod=readonly ./scripts` and
`bun run test:frontend`. The latter uses actual Soda markup
and script with jsdom/API/dialog doubles. No new frontend dependency or build is
needed. The full `scripts/check-native.sh ARCH` still requires a clean revision and
actual native stage. Browser focus, styling, cookies and stock hook rendering need
the separately approved [native journey](native-validation.md#read-only-sodaspaces-browser-probe).

### Controls and states

The current source implements the read states, explicit authentication and the
create/key/join/SSH controls below. Handler/DOM tests cover action-time identity,
duplicate/pending/stale requests and honest failures. Native mutation/connection
proof is recorded separately from the earlier read-only evidence.

Always show repository/environment context, the actual Soda acting identity and Close.

| State | Minimal content / control |
| --- | --- |
| Checking | Native spinner and “Checking environment…”; no mutation until state is known |
| Authentication needed or identity mismatch | Explicit Soda sign-in/re-authentication; never silently act as another identity |
| No environment | Human owner gets **Create shared environment**; others see why unavailable. Backend enforces ownership; no org-owned creation claim |
| Existing environment, not joined | Browser-only **Join** creates the real account without SSH keys; saved external SSH keys require explicit selection. Access also offers own Forgejo public-key review and explicit Save. No private-key upload or outbound Git setup |
| Member, usable observation | Project login, current IP, readonly SSH command + **Copy**, public host-key fingerprint; no claim that a displayed address proves client routing |
| Pending action | “Creating…” / “Saving key…” / “Adding you…” with duplicate submits disabled, not an invented progress/job system |
| Stopped, incomplete, denied or unavailable | Honest state; **Refresh status** for a safe reread. No implicit start/repair, automatic mutation retry or second creation over a reservation |

Closing synchronously invalidates stale reads/results, including before the browser's
queued close event; it does not undo an in-flight native mutation. Reopen requires
explicit full-page reload, never remounting to evade an ended terminal or uncertain
operation. Existing join is not later key propagation. The shell now mounts the
complete management content: Start/Stop and reviewed own-key Apply are explicit
source-implemented operations, still requiring scoped native persistence/revocation
proof at each new delivery target. That proof now passed on the isolated x86_64
fixture; see the [remaining-work list](sodaspaces-plan.md#remaining-work--ordered). Destruction requires a separate scope decision; restart controls, resource
charts, member administration and a browser IDE are not part of this slice. The existing-account terminal now has a protected backend and
self-contained mounted component with bounded isolated native-page proof, not a prerequisite
for SSH. Its [mount/dispose contract](terminal-integration.md) is now used by the
native dialog shell, loading the complete content and local terminal styles. Opening
the drawer never opens a terminal. Local layout/renderer tests remain distinct from
the separately recorded passed native OAuth/helper journeys in the handoff.

## Integration limits

A template cannot directly call Soda's database/backend. An OAuth grant is not a
native web session, and different ports do not isolate cookies. Preserve actual
actor/repository identity, origin/CSRF and native WebAuthn/session boundaries; the
[existing API](dashboard-api.md) is not proof of authenticated embedding.

The source foundation now uses `forgejo_url` as the sole browser origin and mounts
Soda API/login/callback paths at `/-/soda/`. Caddy forwards that prefix unchanged;
Go rejects unprefixed aliases, encoded paths and canonicalization redirects. Soda
uses its own newly named path-scoped cookies, CSRF token, exact Forgejo Origin and
same-origin fetch metadata. Native Forgejo cookies/CSRF are not substitutes, and
`fetch('/api/...')` still targets Forgejo rather than Soda. `public_url` and its
setup flag are removed; URL configuration remains origin-only. No permissive CORS.

Repository-scoped reads and new-join authorization now check the acting grant's
fresh subject, consent and native visibility. Schema v5 also binds OAuth finalization
and Soda logout to one cancellable login context. These backend fixes are locally
tested; the read-only native integration passed separately, not an installed migration.

Backend guards now require `X-Soda-Expected-User-ID` except for session bootstrap;
OAuth state binds optional repository/expected-user IDs and callback checks the
fresh subject before saving Soda state. Repository return uses the existing acting-
grant `RepositoryByID` lookup and a locally constructed native URL, never a supplied
redirect URL. Inspected 15.0.7 `repo.GetByID` checks acting-user repository access.

The header is a consistency guard, not proof of the live native browser session.
Native-page context capture, stale-tab handling and Caddy routing passed the bounded
isolated read-only journey; follow the [API caller boundary](dashboard-api.md#native-page-and-stale-tab-boundary).
The mutation-time path subsequently passed bounded native create/key/join/Copy/SSH
on the fresh fixture. Separately approved retained cutover now serves the native
integration with one Forgejo browser origin and schema v5; native private-page and
existing-account access checks passed. See the [cutover evidence](implementation-history.md#approved-retained-cutover).

The public [robot-avatar route](avatars.md) also uses this namespace, without
sessions or grants; it does not confer authenticated drawer access.

Further source facts informing the [implementation sequence](sodaspaces-plan.md):

- `routers/common/auth.go` supplies `.IsSigned` and `.SignedUserID`;
  `services/context/repo.go` sets `.Repository`, including its stable `.ID`.
  Use escaped template data attributes and keep IDs as strings in JavaScript.
  `templates/base/footer.tmpl` calls the custom footer after the native `index.js`
  tag; that ordering is source evidence, not proof of browser initialization.
- Native templates expose `AppSubUrl`, `AssetUrlPrefix` and `AssetVersion`; an
  operator asset prefix does not establish that a CDN serves Soda's files. The
  selected `appliance/config/forgejo.env` already sets `STATIC_CACHE_TIME=0`;
  upstream `modules/httpcache/httpcache.go` uses private max-age=0/must-revalidate.
  Preserve local custom-asset delivery and test actual revalidation, rather than
  introduce a new asset build/version pipeline just for this drawer.
- The inspected router tree has no `/-/soda/` route. Forgejo does own
  `/-/fetch-redirect` (`routers/init.go`) and development-only `/-/demo` routes
  (`routers/web/web.go`); do not proxy the whole `/-/` namespace. This inspection
  is not Caddy path-normalization or native routing proof.
- `routers/api/v1/user/app.go::UpdateOauth2Application` calls
  `GenerateClientSecret` after updating the application. Native
  `routers/web/user/setting/oauth2_common.go::EditSave` updates callbacks without
  that call; regeneration is a separate handler. `models/auth/oauth2.go` preserves
  the application's identity during that update. Use the application's actual
  owner/native settings for the planned callback transition, not a blind API PATCH.
  No OAuth application or credential was changed during inspection.
- `internal/nativebuild/bundle.go` now admits only the two Soda template paths and
  required ancestors, and requires both hooks/assets with readable modes. It does
  not admit arbitrary templates or Forgejo data. First-install preflight now refuses
  occupied hook/asset destinations before writes, including symlinks/special files;
  source fixtures exercise the actual staging/preflight logic. No generic merge,
  existing-target upgrade or permission to replace customizations was added.

If stock configuration/templates/assets/APIs cannot meet the requirement,
explain the concrete constraint and return for a decision. Do not fork Forgejo,
ship a custom executable, scrape/relay HTML, borrow credentials or introduce a
replacement frontend/framework. Keep native Git/LFS/package and administrator flows.

Overrides/assets require [matching notices and source obligations](licensing.md).
No image upgrade, ingress change, service restart or provider mutation is authorized
by this guide. All retained project state, private credentials and evidence stay intact.

## September Forgejo visual redesign

The owner selected an image-light interface: no decorative robots or routine dashboard/profile photography. The complete override set now inherits neutral light/dark surfaces, red actions, square controls, Barlow Condensed headings, Barlow body text and IBM Plex Mono controls. Text-led intros, restrained functional empty-state icons, mobile-first creation/dashboard layouts and symbol-only navigation replace the older presentation. Native forms, permissions, translations, Git status/diff colors, avatars, organization logos and repository content retain their owners.

[The independent redesign status](forgejo-redesign-status.md) records the source changes, browser fixtures and remaining installed verification. Fastfetch ASCII branding is a separate follow-up. Older route-specific art and visual receipts are historical, not the current design contract.

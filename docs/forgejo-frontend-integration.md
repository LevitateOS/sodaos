# Forgejo customization contract

Use stock Forgejo's frontend throughout; Soda adds the **Sodaspaces** repository
**button and side drawer**, not a new repository tab. See the [short plan](sodaspaces-plan.md)
and [handoff](implementation-status.md). The drawer and authenticated native-page
→ Soda connection remain unimplemented. Existing branded pages have a local preview;
the drawer findings below remain source inspection only.

## Verified source surface

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

Staging already supplies native themes/logos under `gitea/public/assets`, **not** a
new template delivery/render path. Forgejo's customized Fomantic subset has native
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
Production staging/verifier delivery is still pending. Shared presentation does
not add handlers, authentication, permissions, a frontend build or a Lit dependency.
Use Lit selectively when a new self-contained interaction warrants it; do not
migrate native forms/lists wholesale or assume their CSS/scripts cross a shadow root.

These are official customization mechanisms, with version-sensitive compatibility:
[Forgejo's documentation](https://forgejo.org/docs/latest/admin/advanced/customization/)
explicitly does not guarantee template/custom-resource compatibility across upgrades.
Review the overridden templates, native partial/script boundaries, CSS adapters and
local browser behavior against the exact candidate version before an upgrade.
The [Lit shadow DOM documentation](https://lit.dev/docs/components/shadow-dom/)
explains the CSS and DOM boundaries that any future Lit component must account for.

## Notification bell quick-view investigation

The [implementation plan](notification-preview-plan.md) sequences compact native
rendering, bell/HTMX integration, focused validation and separately authorized delivery.
No preview is implemented yet.

Source reviewed at **15.0.7**, tag commit
`d4de9eb2a87c26b402fdd0259e079957f8cd2b4b`. This is an investigation and candidate,
not an implemented dropdown or an authenticated browser proof.

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

Use supported navbar/custom hooks for the bell panel, plus a compact presentation
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

### Controls and states

Always show repository/environment context, the actual Soda acting identity and Close.

| State | Minimal content / control |
| --- | --- |
| Checking | Native spinner and “Checking environment…”; no mutation until state is known |
| Authentication needed or identity mismatch | Explicit Soda sign-in/re-authentication; never silently act as another identity |
| No environment | Human owner gets **Create shared environment**; others see why unavailable. Backend enforces ownership; no org-owned creation claim |
| Existing environment, not joined | Registered development-key summary; when none exist, a public-key textarea and **Save public key**, then **Add me**. No private-key upload or Git-key selector |
| Member, usable observation | Project login, current IP, readonly SSH command + **Copy**, public host-key fingerprint; no claim that a displayed address proves client routing |
| Pending action | “Creating…” / “Saving key…” / “Adding you…” with duplicate submits disabled, not an invented progress/job system |
| Stopped, incomplete, denied or unavailable | Honest state; **Refresh status** for a safe reread. No implicit start/repair, automatic mutation retry or second creation over a reservation |

Closing aborts/discards stale reads, not a promise to undo an in-flight native
mutation. Reopen by reading actual state. Existing join is not later key propagation.
Start/stop/restart/delete, resource charts, member administration and a browser IDE
are not needed here. The separately requested existing-account terminal remains a
follow-up, not a prerequisite for SSH access.

## Integration limits

A template cannot directly call Soda's database/backend. An OAuth grant is not a
native web session, and different ports do not isolate cookies. Preserve actual
actor/repository identity, origin/CSRF and native WebAuthn/session boundaries; the
[existing API](dashboard-api.md) is not proof of authenticated embedding.

Concrete current constraint: `internal/web/api.go` requires a Soda session cookie,
Soda CSRF token, exact `PublicURL` Origin and same-origin fetch metadata for writes;
`proxy.Caddyfile` exposes Soda and Forgejo on separate origins. Native Forgejo CSRF
or `fetch('/api/...')` is not a substitute. A fixed same-origin Soda API/OAuth namespace
through existing Caddy is a candidate to verify, not an implemented route: callback,
cookie/path, configured-origin and native-route collision contracts need review.
`config.BaseURL` currently permits origins only, so simply putting a subpath into
`public_url` is not valid. Do not weaken checks or add permissive CORS to hide this.

Further source facts informing the [implementation sequence](sodaspaces-plan.md):

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
- `internal/nativebuild/bundle.go` currently admits Forgejo public assets, not
  custom template directories. The planned hooks need a narrow staging/verifier
  change with tests, not a blanket allowance for Forgejo's writable data.

If supported configuration/templates/assets/APIs cannot meet the requirement,
explain the concrete constraint and return for a decision. Do not fork Forgejo,
ship a custom executable, scrape/relay HTML, borrow credentials or introduce a
replacement frontend/framework. Keep native Git/LFS/package and administrator flows.

Overrides/assets require [matching notices and source obligations](licensing.md).
No image upgrade, ingress change, service restart or provider mutation is authorized
by this guide. All retained project state, private credentials and evidence stay intact.

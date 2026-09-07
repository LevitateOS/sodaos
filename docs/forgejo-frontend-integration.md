# Forgejo customization contract

Use stock Forgejo's frontend throughout; Soda adds the **Sodaspaces** repository
**button and side drawer**, not a new repository tab. See the [short plan](sodaspaces-plan.md)
and [handoff](implementation-status.md). The read-only hook/drawer/context caller
is authored, alongside Go/proxy/config/cookie and backend actor/return handling.
Complete authenticated native-page → Soda browser proof remains pending; source
findings and DOM doubles are not that proof.

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

Staging supplies native themes/logos and the four Sodaspaces hook/assets under
`gitea/templates/custom/` and `gitea/public/assets/`. This is authored delivery,
not an installed/rendered native result. Forgejo's customized Fomantic subset has native
accessibility/initialization adaptations; it is not a standalone Soda component kit.
Preserve native markup, scripts, form behavior and branding. The source reference
is retained under `.artifacts/research/h01-0f43b9f/v15.0.7/forgejo/`; inspect the exact
selected version/configuration when changing an override.

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

With the existing pinned Go/Node tools and prepared Cockpit dependencies, local
checks are `go test -mod=readonly ./scripts` and
`node --test tests/frontend/sodaspaces.test.mjs`. The latter uses actual Soda markup
and script with jsdom/API/dialog doubles. No new frontend dependency or build is
needed. The full `scripts/check-native.sh ARCH` still requires a clean revision and
actual native stage. Browser focus, styling, cookies and stock hook rendering need
the separately approved [native journey](native-validation.md#read-only-sodaspaces-browser-probe).

### Controls and states

The current source implements the read-only states and explicit authentication;
create/key/join/SSH controls in the eventual table below are not implemented.

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
tested, not a native-page integration or installed migration.

Backend guards now require `X-Soda-Expected-User-ID` except for session bootstrap;
OAuth state binds optional repository/expected-user IDs and callback checks the
fresh subject before saving Soda state. Repository return uses the existing acting-
grant `RepositoryByID` lookup and a locally constructed native URL, never a supplied
redirect URL. Inspected 15.0.7 `repo.GetByID` checks acting-user repository access.

The header is a consistency guard, not proof of the live native browser session.
Native-page context capture and stale-tab handling are authored; the real browser
round trip remains pending; follow the [API caller boundary](dashboard-api.md#native-page-and-stale-tab-boundary).
The Caddy recipe has not been exercised or deployed; source HTTP tests do not prove
native route matching or cookie behavior. The installed guest retains historical
separate origins/configuration and schema v3.

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

If supported configuration/templates/assets/APIs cannot meet the requirement,
explain the concrete constraint and return for a decision. Do not fork Forgejo,
ship a custom executable, scrape/relay HTML, borrow credentials or introduce a
replacement frontend/framework. Keep native Git/LFS/package and administrator flows.

Overrides/assets require [matching notices and source obligations](licensing.md).
No image upgrade, ingress change, service restart or provider mutation is authorized
by this guide. All retained project state, private credentials and evidence stay intact.

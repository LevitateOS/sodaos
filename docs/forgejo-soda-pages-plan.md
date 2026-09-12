# Native Soda page integration contracts

This guide owns the native page host, connection/logout, navigation and browser
acceptance contracts. Product scope belongs to [Sodaspaces](sodaspaces-plan.md);
runner operation semantics belong to [Runners](runners-port.md).

The [combined plan](native-pages-runners-plan.md) owns the delivered native-page/
Runners coordination and runner retirement. The separately selected
[Tailnet plan](tailnet-integration-plan.md) owns its host/project feature and Cockpit
retirement, reusing the shared page contracts here. Installed state and permissions
belong to the [handoff](implementation-status.md).
Earlier implementation steps, editor reservations and intermediate results remain
in Git and [execution history](implementation-history.md), not a second work queue.

<a id="implementation-lanes-and-handoff"></a>
Historical lane links resolve here. Their original text is available with
`git show 3f80010:docs/forgejo-soda-pages-plan.md`, not as current editing restrictions.

## 1. Result the user should see

- Forgejo renders the surrounding document, native header, avatar/profile menu,
  notifications and account links on every integrated Soda view. Profile,
  password, account settings and Forgejo administration continue to use native
  Forgejo pages and handlers.
- The existing native navigation gains **Spaces** and **Runners**. Runners is
  visible for the configured Soda operator after identity/bootstrap verification;
  Forgejo site-administrator status is not substituted for that authority.
- Opening a Soda view reuses a valid matching Soda session. When connection is
  needed on a controlled initial entry, it starts the normal Forgejo OAuth flow
  automatically and returns to the selected view. First authorization can still
  require Forgejo's consent screen. Ordinary subsequent connections can return
  without another consent click while the Forgejo session and grant remain valid.
- One profile-menu Sign out action attempts to end the current Soda login context
  and then uses Forgejo's existing sign-out action. Partial failure is visible.
  This must not be described as atomic cross-system or all-device logout.
- Spaces retains the existing multi-session workspace and companion repository
  drawer. Runners and repository settings retain their current controls and
  server authorization. No duplicate account display, header or password form.
- Desktop and mobile views use the existing theme, spacing and menu behavior.
  Moving between native pages, full Spaces and the drawer preserves exact terminal
  identities and the documented retain/Return/End behavior.

## 2. Verified foundation and limits

| Fact inspected | Consequence |
| --- | --- |
| Forgejo 15.0.7 `routers/web/home.go` passes authenticated home requests to `user.Dashboard`; `routers/web/user/home.go` renders `user/dashboard/dashboard`. | The existing dashboard hosts the selected Soda views without a new backend route. |
| Our `appliance/forgejo/templates/user/dashboard/dashboard.tmpl` already invokes native `base/head` and `base/footer`. The upstream navbar owns avatar/profile links, native logout and notification context. | Keep those native template calls and their original context; replace only the selected dashboard body with Soda content. |
| Forgejo 15.0.7 exposes `ctx.Context.FormString`; the existing notification override uses the equivalent `FormBool` path. | A bounded presentation selector can choose a Soda view. It must not choose permissions, accounts or native operations. |
| The drawer already mounts the shared workspace inside Forgejo through `custom/footer.tmpl`. | Full-page and drawer integration share the workspace owner; the verification contract is below. |
| Forgejo 15.0.7 `AuthorizeOAuth` automatically redirects confidential clients with an existing user grant; Soda's setup creates a confidential client. | Automatic connection is possible through normal OAuth, with first consent and failure paths retained. |
| Current Soda callback handling can retire an old context and cancel terminals. Native logout does not revoke the OAuth grant. | Repeated OAuth is not a harmless session probe. Navigation integration alone cannot establish shared revocation. |
| In 15.0.7, protected account routes use Go's cross-origin request protection, not hidden CSRF fields. The native logout route is outside that middleware. | Preserve each native route's actual behavior; Soda's coordinated cancellation still needs its own protections. Do not invent a native logout-token contract. |

Exact upstream references: [home handler](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/home.go),
[dashboard handler](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/user/home.go),
[request accessors](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/services/context/base.go#L172),
[navbar](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/templates/base/head_navbar.tmpl),
[OAuth authorization](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/auth/oauth.go#L524).
Public source copies and SHA-256 records from this investigation are retained at
`.artifacts/forgejo-pages-plan-9ncxei8z/`.

Forgejo [documents template customization](https://forgejo.org/docs/v15.0/contributor/customization/),
but [does not guarantee compatibility across upgrades](https://forgejo.org/docs/v15.0/admin/advanced/customization/).
This is not a stable backend plugin SDK. The [architecture's frontend boundary](architecture.md#frontend-and-session-boundary)
owns the supported-integration and no-fork requirements.

## 3. Native page host and ownership

Use **one query-selected presentation of Forgejo's existing global dashboard**.

| Soda view | Native destination | Content owner |
| --- | --- | --- |
| Spaces | `/?soda-view=spaces` | Shared `SodaSpaces` component in full-page mode |
| Runners | `/?soda-view=runners` | Runner Lit component |
| Repository Spaces settings | `/?soda-view=repository-spaces&repository_id=123` | Shared project controls, authorized for that stable repository ID |

**Planned extension:** Tailnet adds an operator-only `/?soda-view=tailnet` under the
[Tailnet plan](tailnet-integration-plan.md). It is not admitted by current source.
Implement its fixed bookmark, enumerated OAuth destination/schema constraint,
settings-link visibility, mount/title, selector validation and fixture consumers as
one paired change; no generic return URL or second login coordinator. Project
Network controls reuse the existing authorized project settings/workspace owner.

**Entry constraint found in the real browser:** a raw root query does not force
native authentication. In particular, Forgejo's remember-me redirect can discard
that query; an anonymous home configured to redirect elsewhere also bypasses this
template. Use native `/user/login?redirect_to=<encoded fixed rendering destination>`
when a native session must be established. Normal fixture login and already-signed
return use that upstream endpoint. Use fixed entry bridges and OAuth returns;
do not advertise a raw rendering query as a universally reliable signed-out bookmark. This is not solved by the dashboard body template.

Construct these under the configured native origin and actual `AppSubUrl`. Do not
create `/spaces` or `/settings/runners` handlers inside Forgejo, proxy a fake native
page, or hide an arbitrary native page beneath a client-rendered overlay. With no
recognized Soda selector, the existing dashboard remains unchanged. Preserve native login,
account activation, password-change and 2FA gates before dashboard rendering.

The repository settings view deliberately uses this same dashboard host. Its
protected repository API admits a visible repository; individual operations have
their own authority. Hosting it exclusively under Forgejo's repository-admin
settings route, protected by upstream
[`reqRepoAdmin`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/web.go#L1107),
could wrongly exclude existing members or a configured operator.
Keep the existing repository-settings navigation entry, make it point to this
view, and display a repository heading/back-link resolved by the protected API.
Do not manufacture Forgejo repository template context in the global dashboard.
The existing repository drawer remains available from the native repository page.

Treat query values as untrusted locators. Accept only the three explicit view
names and, for repository settings, one canonical positive repository ID. Invalid,
duplicate or incompatible parameters must mount no controls or private metadata.
Do not use a caller-supplied return URL. A generic native HTML response can be 200
while the protected Soda API returns 401/403; a visible empty shell is not access.

Forgejo owns the outer document and its scripts. Soda owns only the content mount,
its view selection, its API/OAuth adapter and local environment/runner operations.
Native CSRF remains native; Soda APIs retain their separate CSRF, origin, actor,
grant and operation-specific checks. A template's signed-user ID is a consistency
input, not authentication for Soda's backend.

The existing dashboard handler still performs its normal feed/organization reads.
It also supplies the initial Dashboard document title. Set the final Spaces,
Runners or repository title in the already-required entry module before mounting;
the initial/no-JavaScript title remains a documented limit of this candidate.
Do not copy the global `base/head` template just to change that title. Show a useful
JavaScript-required message in the selected body while native navigation stays usable.

## 4. Connection and logout contract

### Automatic connection

Use a small shared integration module for these concrete callers, not a new
application router, user store or general authentication framework.

1. Read the native page actor and inspect the existing Soda session once on entry.
   A valid matching session is reused. Do not rotate it or interrupt its terminals.
2. With missing authentication on an initial full-page Soda entry, start one
   transaction-bound OAuth attempt. Reuse the current fixed destinations and
   repository/expected-actor binding; callback returns to the native view above.
3. First consent remains in Forgejo. On repeat connection, its confidential-client
   grant can skip that screen. Verify the granted scopes; prior consent is not
   proof of newly requested scopes.
4. Cancellation, denied/missing scopes, ambiguous/stale cookies, actor mismatch,
   superseded callbacks and provider/storage errors produce a stable native-shell
   error with Retry/Back. No automatic loop, account switching or mutation replay.
5. Automatic navigation is confined to entry before editable content is available.
   Do not reconnect on focus, polling, generic API errors or passive drawer restore.
   On an existing native page with a draft or active workspace, retain its native
   departure warning and an explicit connection/retry action when needed.

The header can discover an operator from an existing matching Soda session without
starting OAuth on every ordinary Forgejo page. A newly signed-in user may need to
enter Spaces once before the operator-only Runners link is revealed. The old
protected Runners URL must also remain a working direct entry into automatic
connection. Do not solve discoverability with a site-admin check or a new public
operator inventory.

### One normal sign-out action

Intercept only the native navbar's exact sign-out activation, synchronously in
capture phase, and prevent duplicate activation. Complete Soda cancellation first,
then permit one activation of the original native link. Forgejo's existing
`linkAction` continues to own its POST, error handling and redirect. Preserve the
native route's actual cookie/origin behavior; Soda cancellation needs the separate
CSRF protection below. Do not invent a native logout-token exchange.
An async bubble listener is insufficient because the native POST can already run.
Preserve the original native-only action when Forgejo's JavaScript works but the
Soda coordinator cannot complete; test keyboard activation too. The upstream anchor
has an empty `href` and requires JavaScript for its POST, so it is not a working
JavaScript-disabled logout fallback. This plan does not add a replacement logout form.

Reuse `apiLogout`, `EndLoginContext`, the existing terminal lock and context-bound
stream cancellation. Cover an OAuth attempt that has started before any Soda
session exists: a session-bootstrap 401 does not prove there is nothing to cancel.
`FinishOAuth` clears the pending state marker, so cancellation using only that
marker can miss a newly completed callback. The existing login context retains
the latest OAuth-cookie hash through callback completion, under the
[schema-v9 contract](dashboard-credentials.md#schema-v9-pending-oauth-cancellation).
That hash grants no API authority and replaces no single-use OAuth checks.

For this pending-context case, a no-store same-origin cancellation bootstrap and
POST must bind CSRF to the unique OAuth cookie, resolve only its existing context,
and accept no caller-selected actor/context. Require the configured Origin,
fetch-metadata/method/body protections and unambiguous cookies. Delete only that
context and cancel its streams under the existing lock. Prove callback-before,
callback-after, cookie rotation and late Set-Cookie races before relying on it.
Do not turn cancellation into an anonymous context-creation endpoint.

Suppress automatic reconnect while this browser's sign-out is in progress and
after explicit cancellation. Notify mounted Soda views to retire their current
requests, clear credential fields and stop attachment attempts. Cross-tab notices
contain no credentials and are only UI signals, never backend authorization.
Do not adopt a later actor while finishing an earlier actor's logout.

If Soda cannot confirm cancellation, offer retry and an explicit native-only
sign-out escape. If Forgejo fails after Soda succeeds, retain a visible partial
outcome through its native error behavior. Never report overall success merely
because one request succeeded or because the page navigated.

Forgejo's [native logout handler](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/auth/auth.go#L370)
and [event endpoint](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/events/events.go#L93)
provide `logout/here` versus `elsewhere` observations. However, the
[stock consumer](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/web_src/js/features/notification.js#L90)
uses an optional SharedWorker path and immediately navigates; it exposes no general
document event. Do not depend on that event for the required Soda POST or build a
second notification transport. It is a possible additional invalidation signal
only if the exact integration is verified without duplicating native ownership.

**Limit:** this is coordinated browser-menu logout, not atomic backend revocation.
Native-only sign-out, JavaScript disabled, offline/suspended/closed tabs, missed
events and independent session expiry cannot guarantee immediate Soda revocation.
The existing backend context/grant/expiry checks remain mandatory. Do not revoke
the entire OAuth application grant just to imitate browser logout. If immediate
revocation through every native logout path is required, the present mechanism
does not establish that guarantee; report that limitation, not a completed SSO claim.
A cancellation request can resolve only cookies already supplied by that browser:
a brand-new login whose first cookie has not arrived is not yet identifiable.
Late cookies for an already cancelled context cannot restore its authority.

## 5. Integration verification

These are acceptance contracts for affected behavior, not a sequence to repeat
with every edit. Reuse `TestNativeConnectionFixture` and the existing page/Forgejo/
workspace consumers. [TypeScript](typescript.md) owns local commands and fixture
prerequisites; [native validation](native-validation.md) owns installed journeys.
Runner processes/jobs and terminal lifetime proof stay with the
[runner native](runners-native-validation.md) and [terminal](terminal-integration.md)
guides, not synthetic operation peers.

| Boundary | Required behavior |
| --- | --- |
| Native host | Actual Forgejo HTML/header/footer, native account gates, CSP and assets; one main landmark and workspace mount. Suppress only the selected full page's drawer; retain ordinary dashboard/footer behavior. Invalid repository locators expose no controls/private metadata. |
| Connection/logout | First/repeat consent, decline/missing scopes, actor mismatch, concurrent attempts, pending/completed callback cancellation and both partial logout outcomes. Preserve native drafts and existing matching sessions. |
| Page/drawer lifetime | Full Spaces → repository drawer → full Spaces preserves exact session selection, finite retention/Return and independently named End. Real Back/BFCache revalidates the original actor; stale/duplicate owners and late responses ignoring abort cannot revive departed work or replay mutations. |
| Navigation/bookmarks | Native navigation and fixed legacy entry bridges agree; signed-in/out bookmarks preserve expected actor and repository bindings. No arbitrary return URLs, second login harness or automatic lifecycle effects. |
| Assets/upgrade | Changed entries and transitive imports respect the actual configured cache policy. Distinguish fresh-browser checks, predecessor asset revalidation and a genuinely open predecessor document/backend transition. Preserve the final CSP, staging and notices contracts. |
| Presentation | Native dark/light themes, narrow/wide layout, scroll/focus/keyboard/profile-menu behavior and unsaved forms remain usable. Use the existing screenshot guide; rendered/synthetic terminal content is not native editor/process proof. |

The fixture must fail for missing required native host/assets or a failed consumer,
not silently substitute handwritten HTML or skip the affected integration case.
Record source/browser results separately from installed proof in the handoff.

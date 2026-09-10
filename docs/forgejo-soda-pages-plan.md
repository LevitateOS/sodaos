# Integrate Soda pages into native Forgejo

**Status, 10 September 2026: steps 1–4 are implemented with local stock-Forgejo
browser proof. Steps 5–6 remain.** The original plan used source baseline `f3efebc`;
step 1 follows planning commit `a995f9e`. The three native hosts now mount the existing management components.
Normal navigation reaches those hosts; old Go URLs are fixed bookmark bridges
and the duplicate shells are removed. No appliance delivery is claimed.

**Whole-plan review at `97a2d5e`: changes required.** The
[source-backed review](forgejo-soda-pages-review.md) reproduced a late-entry OAuth
navigation bug, permanently stale full-page BFCache restoration and missed Go
asset-version assertions; it also identifies the old Runners navigation label.
Earlier step receipts remain historical evidence, not a waiver of these defects.
These corrective source changes can proceed locally without a Linux builder.

The user wants Spaces and Runners to feel like parts of SodaOS's Forgejo interface:
the same real header, navigation, profile menu, login and account settings. This
plan also covers the existing repository Spaces settings page. It supersedes the
previous recommendation to keep these pages in independent Soda-rendered HTML
shells. That earlier implementation and its test/deployment evidence remain facts;
they are not evidence that the requested integration is complete.

The [Sodaspaces plan](sodaspaces-plan.md) owns product scope. This document owns
only this integration sequence. [Runners](runners-port.md) retains its separate
native/provider parity and Cockpit-retirement obligations. Services, AI automation,
new environment profiles and additional runner functionality are not added here.

## Implementation lanes and handoff

**This is the single ownership boundary for this plan and
[runners-port](runners-port.md#implementation-lane-boundary).** They are separate
implementation lanes, not two agents completing the same runner page. The Soda-pages lane has implemented steps 2–4 with local authentication/page-body
browser proof. Shared files remain reserved through the source handoff below; runner work
must not independently edit that implementation.

| Work | Sole implementation owner | Other lane's responsibility |
| --- | --- | --- |
| Native document/mount, all three page-body ports, CSS/layout, navigation, old GET entry bridges and standalone-shell deletion | Soda-pages steps 1/3/4 | Runners supplies existing operation invariants; does not rebuild or retain a competing shell. |
| Shared session bootstrap, automatic OAuth entry/returns, coordinated logout, context/schema migration, cross-tab retirement and wiring existing sign-out controls | Soda-pages step 2 | Runners consumes the resulting authenticated actor/session and retirement contract; does not add an alternative connection/logout mechanism or migration. |
| Runner endpoint authorization, strict request/response contract, registration secrets, list/create/start/stop/restart/remove effects and partial outcomes | Runners step 3 | Soda-pages preserves these behaviors while moving the existing controls; reports runner defects to that owner rather than changing native semantics. |
| Native runner lock, CLI/root bridge, accounts/state, service/launcher/client and unsupported-provider refusal | Runners steps 3–6 | Soda-pages does not change these owners or exercise provider/lifecycle effects for shell acceptance. |
| Browser host/entry/logout/navigation/restoration fixtures, including porting existing runner browser cases and secret/no-replay assertions | Soda-pages steps 4–5 | Runners reuses these cases and evidence; adds only missing runner-operation cases after the browser-file handoff, not a second host/auth fixture. |
| Runner backend/CLI/socket regressions and opt-in installed registration/job/lifecycle/overlap/preservation journeys | Runners steps 3–5 | Soda-pages uses existing runner controls and bounded synthetic operation responses; shell proof is not provider parity. |
| Page assets/hooks/CSP/notices and their canonical build/stage assertions | Soda-pages steps 4–5 | Runners consumes the packaged page; owns only runner executable/service/client and Cockpit-specific payload assertions. |
| Native-shell/schema delivery | Soda-pages step 6 | Runners consumes the exact delivered revision/schema and shell evidence. |
| Paired runner management delivery, obsolete helper retirement, provider parity and later Cockpit runner presentation removal | Runners steps 4–7 | Soda-pages retains Cockpit and never marks these exits complete from a UI delivery. |

**Shared-file rule:** until the Soda-pages source handoff, it is the sole editor of
`frontend/runners/soda-runners-page.ts`, `soda-settings.css`,
`internal/web/settings_page.go`, the retiring runner HTML template,
`tests/frontend/runners.test.ts`, native navigation modules/templates and common
page/browser/build fixtures. Its step-2 assignment also owns shared OAuth/session/
logout/store changes and their tests. `internal/web/runners.go` and
`internal/web/runners_test.go` contain shared authorization and old page fixtures:
reserve these files too until an explicit file-level commit handoff. That reservation
permits integration edits, not changing runner operation semantics. Runners can
proceed now in `internal/runners/`, its CLI/host adapters, new focused runner API
test files using existing seams, and separate installed runner cases. Changes to a
shared authorization function or common file must be handed to its active owner.
Do not split, duplicate or wrap production code just to avoid a file collision.
If an operation fix needs a reserved UI/test/build file, record the exact requested
change and serialize that edit after an explicit commit handoff. Common documentation
and payload files likewise have one editor per change; neither agent overwrites
uncommitted work from the other lane.

**Bounded installed-driver editor handoff (after `9629c8e`):** Runners now owns
`tests/installed/sodaspaces.ts` and `tests/frontend/sodaspaces-probe.test.ts` for
integrating `loadRunnerInput`/`exerciseRunners` through the existing driver and its
embedded one-shot guard. Soda-pages will not edit these files concurrently. This
supersedes the blanket reservation only for those two files, not the outstanding
step-5 acceptance exit or the remaining UI/auth/payload files. The driver requires
runner argument/phase wiring, native-view entry and coordinated-cancellation guard
updates; it is an integration baseline, not installed-ready proof. See the leading
handoff for the exact dependencies. Any shared-file expansion needs another explicit
handoff; no separate authentication harness or execution grant follows.

**Source handoff:** Soda-pages identifies a committed revision, fixed entry URLs,
actor/session and retirement behavior, migrated browser callers, emitted payload
and actual checks/limits. Runners then extends the handed-off controls only for
runner-operation gaps. Native/provider fixture preparation and backend work need
not wait; final native-shell runner UI parity uses this handoff, not the retired
Go shell. No duplicate standalone shell or second login/browser test harness.

**Delivery handoff:** use one exact candidate manifest and one maintenance executor
per target/window. Separate deliveries are sequential: Soda-pages owns shell/schema
changes, then Runners owns the remaining paired management changes. If one approved
candidate combines them, explicitly name one executor and cite its common build,
schema rehearsal, backup and delivery receipt from both plans; do not run either
common phase twice. Each lane still owns its distinct acceptance assertions.
Later revisions or changed target state require fresh applicable checks/backups;
a shared receipt is not blanket evidence or rollback. Neither lane may deploy the
other's unreviewed pending artifacts or borrow provider, reboot or cleanup authority.

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
| Forgejo 15.0.7 `routers/web/home.go` passes authenticated home requests to `user.Dashboard`; `routers/web/user/home.go` renders `user/dashboard/dashboard`. | An additional presentation of the existing dashboard is a concrete host candidate. A new backend route is not required for that candidate. |
| Our `appliance/forgejo/templates/user/dashboard/dashboard.tmpl` already invokes native `base/head` and `base/footer`. The upstream navbar owns avatar/profile links, native logout and notification context. | Keep those native template calls and their original context; replace only the selected dashboard body with Soda content. |
| Forgejo 15.0.7 exposes `ctx.Context.FormString`; the existing notification override uses the equivalent `FormBool` path. | A bounded presentation selector can choose a Soda view. It must not choose permissions, accounts or native operations. |
| The drawer already mounts the shared workspace inside Forgejo through `custom/footer.tmpl`. | Native embedding is implemented. The new full-page hosting, navigation and lifecycle still need real-browser proof. |
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
This is not a stable backend plugin SDK. No fork, custom Forgejo executable,
borrowed native cookies, direct Forgejo database access, iframe, scraped/relayed
header or copied native authentication implementation is part of this plan.

## 3. Proposed page host and ownership

Use **one query-selected presentation of Forgejo's existing global dashboard**.
Step 1 implements the following rendering destinations with minimal bodies:

| Soda view | Proposed native destination | Content owner |
| --- | --- | --- |
| Spaces | `/?soda-view=spaces` | Existing `SodaSpaces` component in full-page mode |
| Runners | `/?soda-view=runners` | Existing runner Lit component |
| Repository Spaces settings | `/?soda-view=repository-spaces&repository_id=123` | Existing shared project controls, authorized for that stable repository ID |

The content-owner column is the later migration target. Step 1 renders a heading
and a Lit navigation link to the existing protected page, with no inventory,
repository metadata or mutation controls.

**Entry constraint found in the real browser:** a raw root query does not force
native authentication. In particular, Forgejo's remember-me redirect can discard
that query; an anonymous home configured to redirect elsewhere also bypasses this
template. Use native `/user/login?redirect_to=<encoded fixed rendering destination>`
when a native session must be established. Normal fixture login and already-signed
return through that upstream endpoint passed. Steps 2/4 must use fixed entry bridges
and OAuth returns; they must not advertise a raw rendering query as a universally
reliable signed-out bookmark. This is not solved by the dashboard body template.

Construct these under the configured native origin and actual `AppSubUrl`. Do not
create `/spaces` or `/settings/runners` handlers inside Forgejo, proxy a fake native
page, or hide an arbitrary native page beneath a client-rendered overlay. With no
recognized Soda selector, the existing dashboard remains unchanged. Preserve native login,
account activation, password-change and 2FA gates before dashboard rendering.

The repository settings view deliberately uses this same dashboard host. Its
current Go handler admits a visible repository; individual operations then have
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
The source review identified a concrete race: `FinishOAuth` clears the pending
state marker, so cancellation using only that marker can miss a newly completed
callback. The proposed minimal extension retains the latest OAuth-cookie hash on
the existing login context through callback completion, with an append-only schema
migration. It grants no API authority and replaces no single-use OAuth checks.

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

## 5. Implementation sequence and exits

### Step 1 — prove the native page host

**Implemented with local browser proof.** Evidence:
`.artifacts/native-page-host-h4eo83uh/`. Stock Forgejo 15.0.7 runs as an aarch64
container on this development machine; Chrome used the authorized local fixture
account. Actual server HTML, scripts and styles were used without response mocks.
The existing preview received canonical public assets and a template reload;
prior public files were backed up, and its data/configuration were preserved.

The signed root-dashboard branch accepts exactly one recognized selector and
canonical positive int64 repository IDs. Invalid/incompatible repository locators
produce feedback without a mount. Unknown/duplicate view selectors retain the
ordinary dashboard HTML. The footer suppresses only the selected view's drawer;
native header/profile/notifications and ordinary dashboard/drawer callers remain.
One minimal Lit mount sets the final document title and links to the current
protected page. Normal navigation, OAuth returns and full controls are unchanged.

Passing local checks cover the native profile/settings and notification menus,
all three views at 1440px/390px, guest exclusion, native login/return and ordinary
logout, protected-account cross-origin refusal, unchanged default dashboard HTML,
invalid IDs and the real workspace/Lit/xterm imports plus measured local renderer
output/disposal. This opens no native terminal or Soda operation. Mandatory
activation/password-change/2FA gates remain before rendering in the inspected
upstream Home handler; fixture account flags/2FA were not mutated or separately
exercised. The preview has no Soda backend: its existing operator-discovery session
request returns 404, so this does not prove Soda login, operator admission or OAuth.

The complete local frontend suites (315 passes, 34 independently gated skips),
strict TypeScript/Lit checks, Go template package and four payload tests passed;
the two separately enabled real-Forgejo browser tests also passed. The aggregate
`check:source` command is not passing on this macOS checkout: unchanged installer
code references a Linux-only `commandRunner`, and existing tests reject macOS
temporary-path symlinks. Logs retain those failures and corrected browser-probe
assumptions. The [handoff](implementation-status.md) owns exact evidence/limits.

The implementation contract for this step was:

Use the exact selected Forgejo version and an authorized local fixture. Add the
bounded dashboard view branch and mount a minimal Soda body between the untouched
native header/footer. Verify query selection, native actor context, anonymous login
return, mandatory account gates, profile menu, notification behavior and unchanged
ordinary dashboard. Confirm the native document's actual CSP/asset paths can load
the existing Lit/xterm modules and required measured terminal styles without
weakening protections across all Forgejo pages. Do not copy the old Go page CSP.
Verify absent/unknown selectors keep the original dashboard, valid selectors have
one main landmark and one workspace mount, and invalid/duplicate repository IDs
mount no controls. Suppress only the hidden drawer mount/module on these full-page
views; preserve the other native/custom footer behavior. Resolve guest entry through
the existing native login and fixed Soda OAuth return, including deployments whose
anonymous home redirects to Explore; do not assume root always enforces login.

**Exit:** real Forgejo HTML and browser behavior establish the host for all three
view selectors, with no private data in unauthenticated/denied states. Record any
route, context or CSP constraint before proceeding. A synthetic header fixture or
an attractive screenshot is not enough. If this host fails, revisit this precise
candidate instead of silently restoring separate shells or forking Forgejo.

### Step 2 — implement the shared connection and sign-out flow

**Implemented with local handler/store, emitted-browser and real Forgejo proof.**
The shared `soda-connection` module reuses matching sessions and makes one marked
initial-entry attempt. Failures remain in the native host with explicit Retry/Back;
focus, polling and generic API failures never start OAuth. Unsigned legacy entries
now go through native login with a fixed destination. Authenticated legacy bodies
remain until step 3 and final bridges/shell deletion in step 4.

Schema v9 retains the latest OAuth-cookie hash on its existing login context.
Same-origin cancellation bootstrap/POST handles pending and completed callbacks,
checks the expected actor when a session exists, and shares the terminal lock and
context cancellation. The short-lived OAuth cookie remains through callback success;
confirmed cancellation expires both Soda cookies. Callback/rotation failures do not
restore revoked database authority, even if a response sets a cookie late.

Native menu capture permits one original Forgejo activation after cancellation.
Existing Soda controls use the same coordinator; until their bodies move into
Forgejo, their completed Soda sign-out offers a native continuation because their
old document has no native menu to activate. Local storage carries only retirement
signals; pagehide aborts undispatched logout continuations. No native notification
transport or replacement Forgejo logout implementation was added.

Real proof uses the authorized local fixture account, fixture-owned OAuth apps,
fresh retained Soda databases and a browser-scoped loopback TLS proxy to stock
Forgejo 15.0.7. Consent/decline, repeat consent reuse, all three hosts preserving a
session, keyboard menu logout and both partial outcomes passed. Logout failures
were injected at the browser transport; provider authentication remained real.
See the leading handoff for final checks and evidence. This is not appliance delivery
or native runner/terminal process proof.

The documented non-atomic limit remains: a cancellation request can resolve only
cookies already supplied by that browser. A brand-new login response whose first
cookie has not arrived is not an identifiable context for this endpoint. Late cookies
for a context already cancelled cannot authenticate. Blocked storage/missed events
also cannot provide cross-tab revocation; backend authorization remains mandatory.

The implemented step follows this contract:
Implement the contract above in the existing API/store/OAuth owners and one narrow
native integration module. Preserve existing session lifetime and terminal
correlation. Return only to the fixed native views. Adapt the profile-menu action
through the original native link, and integrate the existing Soda sign-out controls
with the same coordinator rather than leaving contradictory logout buttons.

**Exit:** first consent, repeat automatic return, decline/failure/mismatch,
concurrent attempts, pending-callback cancellation and both partial logout outcomes
pass real-handler tests and a real Forgejo browser journey. Native drafts and
valid live Soda sessions are not discarded by automatic connection.

### Step 3 — port the three existing page bodies

**Implemented with local source/browser proof.** Native entry connects once and
mounts the existing Spaces, runner or repository-controls component with the
original native actor. Spaces measures the real navbar/footer and viewport; no
second drawer workspace mounts on its native full page. Settings styles are scoped
to the content mount and inherit the native theme. Existing Go entries remain
available until step 4; only their content wrapper needed a layout class here.

Repository headings/links use the existing protected collection's owner/name,
with canonical path-component validation. That API now rechecks the original
session after provider I/O, matching the retiring page's freshness boundary.
Runner operation semantics and the workspace/session/terminal owners are unchanged.

The real local Forgejo fixture reaches all three components, enabled operator
controls, token clearing on pagehide, current repository links, native keyboard
logout and 1440px/390px layouts. Runner list observations are synthetic and read-only;
no runner registration or lifecycle action occurred. Existing synthetic workspace
journeys retain exact sessions, finite retain/Return and independently named End.
This is not new native terminal process or runner/provider proof. The complete
navigation switch and browser-caller retirement were subsequently completed in
step 4; cached-payload acceptance remains step 5.


| Surface | Work and preservation |
| --- | --- |
| Spaces | Adapt `frontend/spaces/sodaspaces-page.ts` to the native mount and original native actor. Retain `SodaSpaces`, layout/session modules and terminal owners. Measure available height below the actual native header; full page and drawer must not mount two workspace owners simultaneously. |
| Runners | Mount `frontend/runners/soda-runners-page.ts` within the native body. Keep configured-operator admission, Forgejo-only behavior, original-target confirmations, token clearing, request generations and unconfirmed-operation messages. |
| Repository Spaces settings | Adapt `frontend/spaces/soda-repository-spaces.ts` and shared project controls. Preserve `visibleRepository`, fresh session and per-operation authority; move any required server-derived heading/context from the retired HTML into an existing protected API response, adding only the missing fields. |

Use the existing palette and component styles. Remove standalone header/body
selectors from `frontend/spaces/sodaspaces-workspace.css` and
`frontend/runners/soda-settings.css`; scope page layout to its explicit mount.
Forgejo owns the body's classes, so the old Go-shell selectors cannot be reused
unchanged. Match native theme changes,
dropdown stacking, scroll behavior, mobile menu, focus and accessibility landmarks.
Keep one primary main landmark. Native profile/settings links remain ordinary links.

**Exit:** each actual native page reaches its existing controls with the same
authorization and failure behavior. Native header/profile/notification components
remain functional. Full Spaces ↔ repository drawer preserves the exact selected
terminal, finite retention and independently named End behavior.

### Step 4 — switch navigation and retire the duplicate shells

**Implemented with local checks.** Native navigation and drawer links now use the
fixed native views. Old GET URLs are no-store native-login bridges with query,
origin, cookie-ambiguity and canonical-ID guards. Their Go templates, renderer and
standalone bootstrap callers are removed; protected APIs remain the authority.
No-JavaScript feedback returns to the native dashboard instead of looping through
the old entry. The native OAuth return contract is unchanged.

The existing page orchestrator now uses `TestNativeConnectionFixture`, a fresh
canonical public payload and its own native login for all three browser consumers.
Operation responses remain synthetic; native documents/assets and authentication
are real. The complete page/drawer/page journey preserves exact session selection,
finite retention and Continue working, followed by two independently named Ends.
Migrated runner tests preserve confirmations, secret clearing, retirement, late
responses, uncertain outcomes and non-operator refusal. Go checks retain protected
repository context and populated Spaces/session publication races.

The contract followed for this step was:

**Sequencing revision after step 3:** migrate affected callers, tests and served
assets in the same change as their production owners. Step 4 must leave a working,
tested source candidate; step 5 is the remaining integrated acceptance work, not a
place to defer broken tests or packaging. Scope and lane ownership are unchanged.

Update `custom/extra_links.tmpl`, `soda-settings-link.ts`, repository settings links,
Open in Spaces/Open in drawer and default app links together. Fixed native OAuth
returns already exist from step 2: verify their destination and actor bindings
against the new entry bridges rather than rebuilding that flow.
Keep the native four links and their upstream gates; label the local capacity
destination Runners and keep provider-owned Forgejo Actions navigation intact.

Keep old `/-/soda/spaces`, `/-/soda/settings/runners` and repository Spaces GET
addresses as bounded redirects/entry bridges for bookmarks. Validate IDs and use
fixed destinations; never forward arbitrary return/query values. Preserve any
existing authorization checks until their protected API replacement is in place.
Audit old expected-actor and destination bindings before changing callback behavior.

Remove `internal/web/templates/spaces.html`, `runners.html` and
`repository-spaces.html` after their behavior has moved. Reduce their Go handlers
to the necessary entry behavior and delete the page renderer only when it has no
remaining caller. Retain the Go API service, SQLite data, encrypted grants, existing
configuration/service names and native integration. Do not create a replacement SPA.

Port the affected Go page tests to redirect/entry tests as their HTML handlers
retire. Preserve authorization, protected repository context, expected-actor,
CSRF and current-session race assertions in their surviving owners. Move
`scripts/test-spaces-page.ts` and its three browser consumers to the existing
native-host fixture before removing the HTML producers they consume. Preserve
runner form/confirmation, secret-clearing, uncertainty and no-replay cases; do not
add a second host/auth harness or change runner operation semantics.

Update affected navigation, repository settings and workspace/drawer tests with
the link changes. Verify the complete full Spaces → repository drawer → full Spaces
journey preserves the exact selected session, finite retain/Return and independently
named End. The step-3 component regressions are supporting evidence, not proof of
the newly switched navigation. Cover signed-in and signed-out bookmarks and ensure
entry/reconnect never replays an operation.

Update affected build, payload, staging/preview, asset/CSP assertions and notices
alongside shell/bootstrap removal. Check transitive imports and remaining callers
before deleting obsolete served assets; preserve retained historical artifacts.
Run the affected Go/browser/workspace suites, strict TypeScript/Lit and payload
checks before committing. Update current guides with the source behavior changed
here, keeping installed/historical behavior separately labelled.

**Exit:** normal navigation and old bookmarks reach the native shell; no duplicate
header/account UI or stale full-page bootstrap remains. Migrated test callers and
affected packaging checks pass, including the complete page/drawer handoff.
No runner/provider/root state changes occur as a consequence of opening or redirecting a page.

### Step 5 — validate the complete source and packaged candidate

**In progress:** whole-graph cache versioning, real cached URL hits, native BFCache
runner secret clearing, duplicate-entry protection and additional repository denial/
actor cases now have local evidence. The leading handoff records the passing focused
checks and remaining supported-Linux aggregate, complete visual and combined-history
acceptance gaps. This is not a completed step-5 exit or delivery/source-ownership
handoff.

**Delivery cache check:** the local stock asset response currently uses a six-hour
private cache. Version changed entry modules and their transitive Soda imports as
part of this packaging step; a fresh-browser source check alone is not evidence
that an already-open/cached client has received the new logout/retirement code.


Use step 4's migrated tests and packaged source as the baseline. Reuse the passing
step-2 authentication/store and step-3 component evidence for unchanged behavior;
rerun affected checks for the final candidate and add only missing coverage. Keep
one existing test harness and record each remaining gap against its owner:

- Exercise real Back/BFCache, duplicate/stale mounts and late responses ignoring
  abort across the final native navigation. The step-3 synthetic interrupted-entry
  checks do not establish actual browser-history restoration by themselves.
- Complete dark/light, narrow-screen, scroll/focus, keyboard/profile-menu and
  screenshot review under the actual native shell. Preserve unsaved native forms,
  exact terminal identities and runner secret/uncertainty boundaries.
- Close uncovered authority combinations: operator without site-admin, site-admin
  without operator, owner/member, missing/private repository, actor switch and
  unavailable provider. Reuse existing consent, missing-scope and logout-failure
  tests where applicable; verify native-only escape and JavaScript-disabled limits.
- Prove cached-client delivery of changed entry modules and transitive imports,
  plus the final payload/staging/CSP contract. Missing required real host/asset
  evidence must fail the relevant integration gate; synthetic tests remain useful
  without being labelled native-session or native-process proof.

No navigation or reconnect may replay a mutation, create a terminal, start a
project or register a runner. Native runner lifecycle/provider scenarios remain
in the Runners lane. Test migration and ordinary asset cleanup belong to step 4,
not to this acceptance backlog.

Run the applicable focused tests, `bun run typecheck`, the existing page/Forgejo/
workspace suites, affected Go race tests, payload/staging checks and
`bun run check:source` on a supported prepared development environment. Use
[screenshot capture](screenshot-capture.md) for actual Forgejo visual review;
screenshots complement behavior tests and do not prove native process safety.

Finish consistency review of architecture/API/credential/design/Lit/runner and
validation guides to describe the final source candidate, not an assumed delivery;
retain dated historical evidence. Publish the exact local checks and remaining native assumptions in the handoff.

**Exit:** all three pages and the drawer form one verified native UI journey, with
the first-consent and logout limits above explicitly represented. Source completion
does not claim appliance deployment or real runner/provider parity.

### Step 6 — deliver the affected components with preserved state

After step 5 source acceptance, identify the exact committed candidate and the
explicitly authorized target/actions. Prepare the normal exact-revision native
build/check/export and affected-component delivery recipe under the [single-executor delivery handoff](#implementation-lanes-and-handoff).
This step owns shell/schema delivery only, not paired runner-management rollout,
obsolete helper retirement, provider jobs or Cockpit removal. Account for any
runner artifacts already present in the candidate: their activation needs the
Runners lane's reviewed compatibility scope, not incidental inclusion. Deliver the
compatible backend image, template/assets and any required login-context schema
change together. Rehearse populated-state migration
and preserve configuration, credentials, memberships, project roots, runner accounts,
work files, units and all later writes. This UI change does not call for rebuilding
project roots, installing another identity provider or upgrading Forgejo.

Use only targets/actions covered by the applicable execution authorization; this
plan creates no deployment, provider registration/job, reboot or cleanup grant.
Validate the real native header → connection → page/drawer journey on the authorized
fixture before a separately approved retained-target rollout. Record x86_64 and
aarch64 evidence independently; do not make one architecture a barrier to the other.

**Exit:** the delivered target matches the tested native-shell candidate and its
preserved-state baseline. Cockpit Runners remains until the separate
[runner parity and retirement gate](runners-port.md#implementation-and-completion-gate)
passes. Tailnet remains in Cockpit. This plan does not reopen GitHub runner support.

## 6. Completion claim

Call this integration complete only when Spaces, Runners and repository Spaces
settings actually render within Forgejo's native shell, a normal initial page entry
does not require the old manual Connect step after initial consent, the normal profile-menu
logout has tested coordinated behavior, and existing native workflows, authority,
drafts and terminal/runner lifetimes survive navigation. Report source completion
and each delivered target separately. Do not use “seamless” to imply an atomic
shared backend session or to conceal an unresolved acceptance failure.

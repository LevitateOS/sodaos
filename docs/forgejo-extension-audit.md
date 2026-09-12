# Forgejo extension source audit

**12 September 2026 · Soda baseline `5c2f92a` · stock Forgejo 15.0.7.**
This is an audit and implementation map, not a replacement proposal or a new
execution plan. [Forgejo remains selected](architecture.md#forge-selection).
The [customization guide](forgejo-frontend-integration.md),
[native page guide](forgejo-soda-pages-plan.md) and [API guide](dashboard-api.md)
remain the contract owners. No production code was changed by this audit.

## Verdict

**Soda can feel native without becoming a Forgejo plugin. Much of that integration
already exists.** The browser receives Forgejo's actual document, account gates,
navigation, theme and ordinary forms. Soda contributes selected page bodies and a
repository workspace drawer. Its own Go service authorizes Soda operations.

The distinction matters: these are documented template/asset customization
mechanisms that upstream explicitly calls **unsupported**, not a stable server
extension API. They do not create new native handlers or a shared server session.
Keeping Forgejo does not remove those limitations.

The main actionable defect found is a **runner mutation admission gap after body
decoding**. Native-session parity and dashboard hosting are architectural limits;
navigation/presentation and customization size are improvement opportunities.

## Findings

### Runner admission gap

**Confirmed · medium priority · fix recommended before reusing this operator pattern.**

In [`internal/web/runners.go`](../internal/web/runners.go),
`operatorAuthorization` checks the original Soda session after provider I/O
(line 40). Both mutation handlers then decode the request body. They call
`RunnerCreate` (line 85) or `RunnerAction` (line 145) without another current-session
check after that potentially blocking read.

An audit-only test paused the first body read, completed real routed Soda logout,
confirmed the original session was absent from the real test store, then resumed
the request. For **create, start, stop, restart and remove**, the response was 200
and the synthetic helper received one dispatch. The five unchanged-session controls
also returned 200/one dispatch, as expected. The five regression expectations
(401/no dispatch after logout) failed on pinned Go 1.26.7.

This is **not anonymous access or an operator-role bypass**: the request had already
passed operator, actor, Origin and CSRF checks. It is an already-authorized request
continuing after its session was retired, before privileged dispatch. No real
runner was created, registered, started, stopped or removed.

**Recommended correction:** retain the early operator gate, then use the existing
[`requireCurrentSession`](../internal/web/api.go) check on the original request
session after decoding/validation, immediately before each helper dispatch. Add
focused logout/context/CSRF-change and cancellation cases beside the runner tests,
following the existing [mutation admission tests](../internal/web/mutation_admission_test.go).
Do not move the only authorization check after decoding, invent another session
system, or promise that logout rolls back an operation already admitted to the
helper. This audit did not implement the correction.

### Session parity has a hard limit

**Existing, documented boundary—not a newly discovered authentication bypass.**

The native actor in the DOM and `X-Soda-Expected-User-ID` detect a mismatched Soda
actor; they do not authenticate Forgejo's live browser session. Soda's separate
OAuth grant and path-scoped session are its authority. Checking the grant's current
user is not checking whether the native browser cookie has just been logged out.

[`soda-connection.ts`](../frontend/spaces/soda-connection.ts) coordinates the normal
logout path: cancel pending OAuth or the captured Soda session, retire controls,
then activate the original native logout action. Retained upstream
`routers/web/auth/auth.go::SignOut` destroys Forgejo's session, not Soda's. Native-only
logout, missed browser events and unavailable JavaScript therefore cannot be called
atomic shared logout. The existing [API contract](dashboard-api.md#native-page-connection-and-coordinated-logout)
is appropriately explicit about this.

Keep the existing connection/retirement owners rather than creating feature-local
login code. If true native session/revocation equivalence becomes a requirement,
that needs an explicit architecture decision and an upstream integration mechanism
not established by this audit. Templates, an iframe or a forged actor header cannot
supply it.

### Dashboard hosting is not a new native route

[`dashboard.tmpl`](../appliance/forgejo/templates/user/dashboard/dashboard.tmpl)
selects `spaces`, `runners` or `repository-spaces` only after Forgejo's existing
`Home` → `Dashboard` execution. The inspected native handler still fetches dashboard
feeds and, when enabled, heatmap data before Soda's body is rendered. A native
handler failure there can prevent the Soda page from rendering too.

The document initially has the native dashboard title; the template calls
`base/head` before selecting the Soda body, and
[`soda-native-page.ts`](../assets/branding/forgejo/soda-native-page.ts) changes
`document.title` later. Repository Spaces settings similarly uses the global
Forgejo shell, not the native repository-settings handler/sidebar. It already
shows API-verified repository context and links back to native settings and the
drawer; adding a duplicate repository authority model would be a regression.

This hosting is a practical no-fork compromise. Do not describe query-selected
bodies as independently routed, server-rendered Soda pages. No latency benchmark
or fresh installed rendering test was performed here.

### Navigation and language still expose presentation seams

**Source-visible polish, lower priority than the admission defect.**

- [`custom/extra_links.tmpl`](../appliance/forgejo/templates/custom/extra_links.tmpl)
  renders Spaces with a plain `item` class;
  [`soda-settings-link.ts`](../assets/branding/forgejo/soda-settings-link.ts) creates
  Runners the same way. Neither sets a view-specific active state or `aria-current`.
  Add those presentation cues from the validated selected view, without treating
  selection as authority or changing operator-only visibility.
- New page titles, connection messages and feature controls contain literal English
  while ordinary native templates use `ctx.Locale.Tr`. For locale parity, use the
  existing custom locale/template boundary for Soda strings and pass presentation
  values to Lit; do not invent a second general translation framework. Full
  multilingual acceptance was not tested.

### Customization surface is larger than the extension itself

The current tree contains **253 `.tmpl` files, 240 outside `custom/`**.
[`custom/header.tmpl`](../appliance/forgejo/templates/custom/header.tmpl) emits
**52 unconditional stylesheet links**, including terminal styles. These counts
include Soda's wider Forgejo branding, not just the three extension pages; they
are not counts of demonstrated regressions.

Dedicated hooks, native partials and the shared presentation components reduce
copying, but whole-template replacements remain version-coupled. As affected areas
change, prefer the smallest hook/partial adaptation and review the exact upstream
version's forms, gates and initialization behavior. Route-appropriate stylesheet
loading or consolidation is worth measuring; this audit did not establish a
performance failure or justify a new asset pipeline.

The customization guide also retains clearly marked historical modal/two-hook
candidates beside the current non-modal Lit implementation. Condensing those into
history would make the current extension recipe easier to follow. This is a
documentation recommendation, not an additional release gate.

## What is already implemented

```text
One Forgejo HTTPS origin
  ordinary routes / ?soda-view=… → stock Forgejo → real document and account gates
                                                  └─ Soda Lit body / drawer
  /-/soda/*                     → Soda Go → session + actor + operation authority
                                              └─ private host helper → Project OS / runners
```

| Concern | Actual source owner and behavior |
| --- | --- |
| Origin and routing | [`proxy.Caddyfile`](../appliance/config/proxy.Caddyfile) forwards only `/-/soda/*` to Soda; ordinary native routes stay with Forgejo. This is not a second frontend origin. |
| Page hosting | [`dashboard.tmpl`](../appliance/forgejo/templates/user/dashboard/dashboard.tmpl) validates fixed view selectors and canonical repository IDs, preserves the ordinary dashboard branch, and passes escaped context to the browser. |
| Entry and connection | [`soda-native-page.ts`](../assets/branding/forgejo/soda-native-page.ts) and [`soda-connection.ts`](../frontend/spaces/soda-connection.ts) perform bounded entry-only OAuth when needed, compare the expected actor, load the selected body and retire stale work. Existing confidential-client consent can be reused; initial consent remains native. |
| Repository workspace | [`sodaspaces.ts`](../frontend/spaces/sodaspaces.ts), [`sodaspaces-workspace.ts`](../frontend/spaces/sodaspaces-workspace.ts) and shared project/terminal owners provide the non-modal drawer and page bodies without replacing native navigation or duplicating the terminal runtime. |
| Authority | [`api.go`](../internal/web/api.go), [`provider.go`](../internal/web/provider.go) and [`environment_authority.go`](../internal/web/environment_authority.go) own expected actors, Soda sessions, CSRF/Origin, acting-user grants and operation-specific repository authority. Visibility is not administration. |
| Appliance operator | [`runners.go`](../internal/web/runners.go) uses configured `OperatorID`, not Forgejo site-admin status. The browser link is a hint; protected handlers authorize operations, subject to the admission defect above. |
| Build and delivery | [`build-forgejo.ts`](../scripts/build-forgejo.ts) builds one shared Lit runtime and versioned public-relative module graph from [`forgejo-payload.json`](../internal/nativebuild/forgejo-payload.json). [`stage.py`](../scripts/stage.py) and [`bundle.go`](../internal/nativebuild/bundle.go) stage/verify the reviewed payload and terminal assets rather than a separate hand-maintained deployment list. |

## How to build the next native-feeling extension

These are recommendations derived from the current implementation, not a plugin SDK
or a newly approved feature plan.

1. **Use native functionality when the data is native.** The notification preview
   is the concrete example: the existing native `/notifications` fragment renderer,
   a presentation-only template branch and Forgejo's own HTMX. Its ordinary links,
   badge updates, session gates and full-page fallback remain native. See
   [`notification-preview.ts`](../assets/branding/forgejo/notification-preview.ts)
   and the [notification plan](notification-preview-plan.md). Do not replace that
   with duplicated Soda notification APIs or HTML scraping.
2. **Use the Go service for Soda-owned operations.** Put explicit typed endpoints
   in the existing web owner; reuse session/provider/helper contracts. Decide the
   actual role per action—member, repository owner or Soda operator—not a generic
   “administrator” bit. Treat the runner finding as a warning against copying an
   entry-only authorization check across later I/O.
3. **Mount a body, not another application shell.** For a full Soda page, follow
   the [native page host](forgejo-soda-pages-plan.md): fixed validated selection,
   bounded OAuth return, escaped stable IDs and the shared entry coordinator. For
   repository work, reuse the drawer/context integration. Keep native links and
   forms operating normally; there is no reason to add an iframe or client router.
4. **Share feature controllers between placements.** Follow the existing Lit
   workspace/project owners, their disposal and page/BFCache retirement behavior,
   native theme tokens and explicitly loaded component styles. Native CSS and
   private Forgejo JavaScript are not a universal shadow-DOM/component SDK; follow
   the [Lit guide](lit.md). Opening a page is not permission to start a project or
   replay a mutation.
5. **Wire assets through the existing build and payload.** Add the source/public
   destination in the canonical inventory and keep the presentation epoch coherent
   across HTML entries and transitive imports. The emitted-module checks already
   detect unresolved/duplicate runtime imports and unstaged source. No extension
   registry, secondary bundler or runtime plugin loader is needed.
6. **Check the affected contracts, not a new universal matrix.** Extend the existing
   Go authorization/failure tests, template render tests and emitted-browser tests
   for the feature, including ordinary-page preservation and late-response/departure
   cases. The [source-check prerequisites](typescript.md#local-source-checks) and
   [native validation guide](native-validation.md) distinguish local fixtures from
   separately authorized installed behavior. Neither architecture needs an
   unrelated sibling-build or customer-migration gate for this audit.

**Deployment scope caution:** configuration is origin-only. The drawer explicitly
refuses a nonempty `subUrl`, and several API callers use root-relative paths. Asset
import tests with a prefix do **not** establish end-to-end subpath support. Nothing
in this approach requires a purchased domain; private HTTPS/LAN access remains an
operator setup concern, not a reason to change the integration architecture.

## Evidence and remaining scope

The [execution receipt](implementation-history.md#forgejo-extension-source-audit)
records the exact checks. The pinned baseline checks passed: 35 selected top-level
web tests, 7 top-level template tests and 54 frontend/Forgejo checks (including
emitted Chromium tests), plus the emitted asset build. The
separate runner regression probe deliberately exposed five failures and is retained
with its five passing controls; **the audit is not an all-green security receipt**.

Evidence: `.artifacts/forgejo-native-audit-XdXOQq/`. The previously retained eight-file
Forgejo 15.0.7 source receipt was checksum-verified and reused, not treated as a
whole-upstream or all-template audit. No fresh appliance/native-provider acceptance,
visual/accessibility audit, full frontend/typecheck suite, deployment, real credential
use, fixture lifecycle or cleanup was performed. The admission defect remains
unfixed; [current work and permissions](implementation-status.md) are unchanged by
these recommendations.

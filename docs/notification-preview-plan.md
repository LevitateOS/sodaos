# Notification bell preview implementation plan

Status: **implemented and activated in the local preview; production delivery pending**.
Based on the [15.0.7 investigation](forgejo-frontend-integration.md#notification-bell-quick-view-investigation).
This is a focused native Forgejo customization, independent of the unfinished
Sodaspaces drawer/authentication work. No downstream Forgejo build or Go API adapter.
Implementation does not authorize login, fixture mutations, service reloads or deployment.

## Implementation evidence

The compact fragment, signed-in footer hook, scoped CSS and small lifecycle script
are authored. The real bundle has no `window.htmx`; implementation uses its existing
declarative attributes and DOM lifecycle/custom events instead. Native template-context
shape and both rendering branches are covered by focused Go tests. Browser tests load
the real local Forgejo bundle with browser-only signed-in markup/response fixtures;
they do not prove authenticated native fragment rendering. Checks passed for both bells,
loading/empty/error/retry, delayed-response rejection, keyboard/modified clicks,
Escape/outside/focus dismissal, mobile containment, HTMX redirects and no-JS/no-HTMX/
no-popover fallback. Explore/milestone and guest-theme regressions also passed.

After explicit activation/testing authorization, template reload succeeded. Real
signed-in populated (three unread) and empty inboxes rendered correctly, proving the
native context-expression seam. Desktop/mobile panel containment, Escape/focus,
full-page coexistence, ordinary fragment refresh and “View all notifications” passed;
unread counts stayed 3/3 and 0/0. Screenshots were inspected. Focused suites passed
again. A pre-existing 3px signed-in navbar overflow at 320px remains; the popup adds
no overflow. See the [handoff](implementation-history.md#notification-bell-quick-view-local-activation).

Remaining: actual pinned rows, live badge-event changes, account switching/expiry,
native read-on-navigation and user acceptance. Real notification status changes need
their own fixture scope. Production staging/delivery remains pending. The sequence
below remains the feature contract, not a claim that every validation step ran.

## Experience and scope

- An ordinary bell click opens a compact, anchored panel on desktop or mobile.
- Show up to **five newest native inbox entries: unread plus pinned**, ordered by
  Forgejo's notification update time. Identify pinned entries visibly. This is the
  proposed default matching the existing inbox, not strictly unread-only filtering.
  Confirm a different semantic requirement before implementation, not afterward.
- Each entry shows the native issue/pull/repository icon, repository, title, native
  destination and notification update time. Use server-rendered escaping/helpers;
  the full notification page currently sometimes displays issue update time instead.
- A persistent **“View all notifications”** link opens the native notifications page.
- Include loading, empty and failure states. Empty copy must describe returned inbox
  entries, not claim there are no unread items when a request failed or was redirected.
- Opening/refreshing the preview does not mark anything read. Clicking an entry
  retains normal Forgejo navigation/read behavior; no custom mark-read request.
- Preserve the existing badge and its native updater. No second polling loop,
  notification store, new subscription, mark-all-read action or background preload.
- No-JavaScript and modified clicks retain the ordinary bell link. Guests receive
  no preview controls or requests. No second HTMX library or frontend build.

## 1. Prove the compact native rendering seam

**Source:** `appliance/forgejo/templates/user/notification/notification_div.tmpl`,
new `custom/soda/notification_preview.tmpl`, focused Go template tests under `scripts/`.

1. Verify the exact request-flag expression through the selected Forgejo template
   context. The source-backed candidate is `ctx.Context.FormBool`, not a new global
   function or invented template data. Test it before building the interaction.
2. Use a presentation flag (`soda-preview=true`) on the existing native request:
   `/notifications?div-only=true&soda-preview=true&q=unread&page=1&perPage=5`.
   Respect `AppSubUrl`. The flag changes rendering only; Forgejo's signed-in handler
   still owns the user, query, permissions, sorting and loaded data.
3. Branch to the compact partial only for the intended fragment preview request.
   Preserve the existing full-page fragment for ordinary requests and native refreshes.
   Do not scrape/extract rows from the existing full-page response.
4. Render dedicated preview markup without `notification_div`, `notification_table`,
   `notification_<id>`, full-page `.soda-page` or `role=main`. Native full-page refresh
   code must never discover or replace the popup. Keep row links as native `.Link`
   destinations without copying full-page action/initialization hooks.
5. Keep loading/error markup outside the returned list where needed. Localize labels
   with existing Forgejo keys where suitable; document any remaining custom English
   strings rather than inventing untranslated locale keys.

**Exit:** focused template tests cover preview/non-preview branches, zero/one/five
entries, pinned/issue/pull/repository variants, escaping, subpath links and absence
of conflicting IDs. Verify native rendering under separately applicable execution
scope; stubs alone do not prove the real context expression. If the seam fails,
revisit supported rendering hooks—do not add an upstream patch or cookie proxy.

## 2. Enhance the existing desktop and mobile bells

**Source:** official `custom/footer.tmpl` hook (new; preserve other hook content),
new `assets/branding/forgejo/notification-preview.js` and `notification-preview.css`,
scoped stylesheet reference in `custom/header.tmpl`.

1. Render one signed-in panel and load its small enhancement through supported hooks.
   Locate the two stock bell anchors within the navbar using the reviewed 15.0.7
   structure; require their native notifications destinations. Do not replace the
   navbar or intercept arbitrary links to `/notifications` elsewhere.
2. Enhance only after required browser/HTMX facilities are ready. Preserve anchors,
   badge nodes, modified-click behavior and native destination fallback. Remove or
   suppress the native bell tooltip while the panel is open without affecting others.
3. Prefer a native popover with explicit close control, constrained width/height and
   a scrollable list. Anchor it to the active bell; verify positioning support in the
   selected browsers and use a small bounded positioning fallback if necessary.
   No new component framework. Both bells control the same panel, never duplicates.
4. Supply an accessible name, expanded/control state, keyboard activation, Escape,
   outside-click dismissal and focus restoration. This is a list of navigation links,
   not an ARIA application menu requiring arrow-key navigation. Announce loading/error
   state without repeatedly announcing the entire list or trapping normal focus.
5. On opening, use Forgejo's existing HTMX to load the compact fragment into the
   dedicated target. Verify actual HTMX availability and event initialization order;
   do not assume an exported global without checking the bundled implementation.
6. Keep one in-flight request. Repeated opening/closing must not duplicate requests
   or let an old response populate a later opening. Use existing HTMX abort/lifecycle
   hooks with small local state as needed, not a generalized request framework.
   Clear preview contents on close and refresh on reopening; no persistent cache.
7. Handle errors with concise feedback, an explicit retry and the native fallback
   link. Respect native HTMX login redirects. Do not inject a followed login page,
   retain old private entries after expiry, or log notification bodies/credentials.

**Exit:** the panel opens predictably, fits narrow screens, loads only on demand and
leaves native badge/full-page notification behavior untouched. Missing support or
failed initialization leaves a working ordinary bell link.

## 3. Validate the feature and native coexistence

**Source:** focused tests under `scripts/` and `tests/forgejo/`; use existing tools.

- Template/source checks: branch selection, escaped content, trusted native link
  generation, asset subpaths, signed-in gating and no duplicated full-page IDs/hooks.
- Browser fixtures: loading/empty/error/retry, delayed or out-of-order responses,
  close/reopen, duplicate clicks, modified-click fallback and keyboard/focus behavior.
- Read-only native-page checks under authorized login scope: desktop/mobile bells,
  real populated/empty/pinned output where existing fixtures permit, and coexistence
  with `/notifications`, its native fragment refresh and the unchanged badge updater.
- Check light/dark themes and widths around 320, 390, 768 and 1440px; constrain the
  popup within the viewport and verify no horizontal overflow or hidden focused links.
- Confirm ordinary page/fragment requests still render the full notification view.
  Ensure requests use the native origin/subpath, no API token or Soda backend.
- Cover expired-session HTMX redirects and late responses with isolated fixtures;
  real account switching/login/logout needs its own scope. Do not log credentials,
  borrow a personal browser profile or cache private content in local storage.
- Record network evidence that opening/closing performs only the intended fragment
  GET, apart from Forgejo's existing background traffic. No status/purge POST.
  Clicking a real notification may mark it read through native navigation; test that
  only with explicit exact-fixture mutation scope, or keep it unexecuted and reported.
- Run focused Go/JS/browser checks and the existing Explore/milestone layout
  regressions under applicable local test authorization. No dependency upgrades.

**Exit:** record actual results, failures and unexecuted cases separately in the
handoff. No passing claim based solely on authored tests or synthetic markup.

## 4. Review, delivery and handoff

- Keep coherent commits: compact rendering and tests; panel/assets and interaction
  tests; validation findings and guides. Update the customization README and handoff.
- Review the real preview with the user, including the unread-plus-pinned behavior,
  before treating the interaction as accepted.
- Existing local source mounts can deliver CSS/JS; new/changed templates require an
  explicitly authorized template reload. Do not restart services or mutate fixtures
  merely to demonstrate the feature. Bump asset cache versions deliberately.
- Production template/asset delivery is already pending in the broader integration
  plan. This feature does not silently complete it: inventory these files for the
  product-owned staging/verifier work, then test and roll out only in separately
  authorized delivery scope. No appliance install, upstream upgrade or ingress change.
- Leave native `/notifications` usable throughout. A failure of the preview is not a
  reason to weaken auth, replace native notifications or block normal navigation.

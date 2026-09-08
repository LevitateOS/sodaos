# Current handoff

## Source versus installed state

| Area | Current state |
| --- | --- |
| Selected frontend | Stock Forgejo native pages plus planned **Sodaspaces** repository button/right environment drawer (no new tab) |
| Soda UI source | Native Forgejo template/asset overrides with a shared presentation system in the local preview; drawer pending. Standalone React/duplicate forge adapters removed in `752079e`; original Go/HTMX frontend removed in `9f3baa7` |
| Retained backend | `cmd/soda-dashboard`, Go API/OAuth, schema-v3 SQLite/encrypted grants, real create/join/access integration and restricted helper/project OS |
| Retained operator frontend | Separate Cockpit React/PatternFly Tailnet/Runners, backing native logic/dependencies/tests |
| Installed affected components | Last recorded `8b823db` dashboard/helper/runner companion/default new-project image; stock Forgejo 15.0.7. Historical React `/app/` preview and HTMX defaults remain installed |
| Acceptance | Only historical bounded **U08** native x86_64 first-product proof accepted (`a12b741`). U01 architecture acceptance was withdrawn; no Sodaspaces/final-product/aarch64 acceptance |

Source removal is **not deployment**. No native image/stage build or installed
retest of the removal commits occurred. The current source has no usable Soda
browser controls until Sodaspaces is connected. Root and completed OAuth redirect
only to configured Forgejo, not to caller-selected or historical stored paths.
The old OAuth return-path column/default remains unused without a schema migration.
New consent requests read user/repository/organization scopes, not administrator
expansion; actual existing grants remain intact. Redirecting is not native-session
transfer or cross-origin authorization.

Keep [API](dashboard-api.md), [credential migration](dashboard-credentials.md),
[architecture](architecture.md) and [current work](sodaspaces-plan.md) authoritative.
Acting grants/current native ownership from `fed66cb` remain in retained callers;
no setup-token, stale-creator or copied-permission fallback was restored.

## Expanded native Forgejo branding

Broad supported stock 15.0.7 header/layout overrides now cover repository pages
and settings, account settings, administrator and organization views, and nine
secondary authentication wrappers. Three coordinated task teams added 183 template
override files from the `a81bfae` baseline, including 141 during the final authoring
sprint; the source now contains 201 overrides/helpers. The added detailed families
include code/edit/diff/history, issues/pulls/milestones, releases/wiki/projects,
Actions/runners/webhooks, storage and access lists, profiles/packages/imports,
administrator monitoring, account/security, organization/team, federated auth,
and setup pages. These are template-file counts, including shared partials, not
independently exercised workflows. The shared native leaf forms/lists/scripts,
permissions and handlers remain upstream-owned. The page-marker adapter reaches
whole native pages from their shared header/helper; it is not a widget root.
Common settings sidebars/cards have one CSS owner. Principal native forms use a
positive structural adapter; nested dialog/table/row-action and settings search
forms keep native sizing. Guest theme routes share one presentation gate.
Six new settings illustrations from the separate art task are mapped by native
page flags, preserving its asset/provenance commits and earlier artwork.

Local source tests passed with the existing offline Go toolchain and readonly
modules (`go test -count=1 -mod=readonly ./scripts -run TestForgejo`). These check
native template composition, permission seams, exact-stock recovery hashes,
escaping, theme placement, artwork selection and CSS boundaries. Stock preview
`reload-templates` succeeded. Initial browser checks covered all 11 account
sidebar pages at desktop/390px, all 16 administrator sidebar pages at 390px,
and 12 repository sections at desktop/390px. One narrow native stacktrace overflow
and clipped-popup risks were found and corrected. After combining all three teams,
the full focused Go suite and all seven guest-theme JavaScript tests passed.
Final browser checks exercised 27 distinct pages at 390px and 1654px, plus all ten
native migration-provider forms at 390px, with no page-level horizontal overflow
after correcting the direct system-notices table. The native file editor mounted
CodeMirror and retained its commit form; a desktop release page was visually
checked. Browser viewport overrides were reset. The owner code-search URL
redirected to the profile under the existing configuration, so that route remains
source-tested only. Registration/recovery remain truthfully disabled. Setup,
MFA, activation, consent, POSTs, populated packages/organization teams/project
boards, Actions dispatch and fullscreen logs were not executed or fabricated.
Guest light/dark navigation was checked earlier and returned to light; no native
account preference, fixture, provider, native stage or deployed VM was changed.
All task commits were collected onto `main`; production staging and the Sodaspaces
drawer remain separate unfinished integration work.

## Shared Forgejo presentation components

The local stock 15.0.7 preview uses small Go template partials for page intros,
empty content and the guest theme button. The component audit separates page
shell/tokens, intro, toolbar/native controls, forms, list/pagination, empty feedback
and guest theme/shell ownership. Page files retain only their specific metadata
and layout. Explore now delegates navigation, visibility and overflow to native
`explore/navbar`; context-switcher CSS has an explicit wrapper. All list callers
use the same wrapper contract. Search selectors cannot reach nested dialog buttons;
filled actions use the selected theme's action colors. Login no longer decorates
Forgejo's loader pseudo-element. Guest theme listeners load only on anonymous
routes with a guest toggle.

[Composition contract](../appliance/forgejo/README.md#presentation-component-contract)
and [full audit](forgejo-components-audit.md). `.soda-page` is a full-page shell,
not a root for the future drawer. Home/login keep distinct content layouts; the
native dashboard Vue widget, milestone cards and notification row actions retain
bounded page adapters. The original extraction is recorded in `ce4129c`.

Native forms, permission gates, translations, asset prefixes and notification
replacement hooks remain upstream-owned. Exact embedded 15.0.7 source comparison
found no unexplained behavior divergence in issues, milestones, notifications,
subscriptions or organization creation. The documented historical research mirror
is absent on this machine; the audit used `forgejo embedded view` from the existing
stock preview. No Lit dependency or frontend build was added. The authenticated
drawer and production staging remain pending.

Executed for this audit on the development machine:

- `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -mod=readonly ./scripts -run 'TestForgejo'`
  passed with Go 1.27.1 darwin/arm64. Tests render real Soda partials/callers with
  native seams stubbed, including creation permission branches, guest route gates,
  singleton placement, escaping, subpaths and Explore delegation. This is not a
  locked native toolchain or a full Go/native build check.
- `node --test tests/forgejo/login-theme.test.mjs`: all seven tests passed, now
  including actual head timing before the toggle exists and a no-toggle case.
- Reloaded templates only in existing `sodaos-local-forgejo`. All twelve signed-in
  route variants rendered at 1654px and 390px without horizontal overflow; shared
  intro artwork loaded and search inputs measured 44px. Native Explore overflow
  opened its Organizations menu item and navigated successfully. The syntax dialog
  opened/closed normally; its nested Cancel button matches zero toolbar rules even
  before Forgejo reparents the dialog. The personal context menu stayed within 390px.
- Notification bulk action uses dark action blue with white text; list wrappers and
  twenty visible native rows were verified without submitting actions. Expanded
  native repository initialization/advanced controls fit at 390px; organization
  fieldsets and required input remained intact.
- Guest home/login and all three directories rendered at measured 480px in both
  light/dark modes without overflow. Each page had one toggle (40px; login 44px),
  persisted the choice across navigation and kept native `data-theme`. Login's
  idle submit has no authored `::after`. Unrelated password recovery loaded no guest
  script or theme attribute. Both browser viewport overrides were reset afterward.
- All sixteen linked component/page CSS responses matched source bytes. Changed
  guide file links and `git diff --check` passed.

No fixture records, native account preferences, providers, dependencies, appliance
stage or deployed VM were changed. The guest local choice was restored. Native
notification POST/live replacement, creation POST/server-error and transient
loading/disabled journeys were not newly exercised; source contracts were reviewed.
Signed-in light appearance and organization/team context menus were not exercised
(the local account has no organization context). Template overrides still require
exact-version review and browser checks on upgrades. This is bounded local preview
validation, not production or final-product acceptance. Custom introductory copy
remains English pending the existing i18n work.

## Local New Organization preview

Official `org/create.tmpl` now uses the Soda shell and existing organization
workshop artwork. Native fields, defaults, visibility values, permission checkbox,
error flags and POST action are retained. Desktop cards match the repository form
with 24px padding/gaps and aligned labels. Local reload and Chrome checks covered
desktop, 390px layout without overflow, visibility selection and required name/
40-character constraint. The form was reset afterward; no organization was created.
Submission, server errors and light appearance were not newly exercised. Custom
intro remains English; no appliance deployment.

## Local New Repository preview

Official `repo/create.tmpl` now has the Soda shell, existing repository-folder
artwork and scoped form cards/styles. All native create-helper/basic/template/
initialization/advanced partials, permission gates and form action remain intact.
Local template reload and Chrome inspection verified expanded initialization and
advanced controls, required name/length constraints, asset loading and a 390px
layout without horizontal overflow. No repository was created; submission,
server-side error paths, template selection and light appearance were not newly
exercised. Custom intro is English. No appliance deployment.

## Local Notifications preview

Official stock 15.0.7 notification partial and subscriptions wrapper now use the
Soda shell, dedicated generated inbox artwork, separate consistent toolbar and
rounded list/empty state. Native notification IDs, sequence hooks, forms, data
attributes, status conditions and pagination remain unchanged. Existing fixtures
supply 40 unread notifications; no new data was seeded. Local Chrome verified
populated/read/empty views, mark-read then unread restoration, subscriptions shell,
and a 390px layout without horizontal overflow. Two fixture status checks were
restored to their original states. PNG alpha and template reload were verified;
`git diff --check` passed. Light appearance, pin/bulk actions and watching filters
were not newly exercised. Custom intro remains English; no appliance deployment.

## Local Milestones preview

The official dashboard milestones override now uses Soda's separate toolbar,
repository filter panel and progress cards, reusing the checklist illustration.
Native milestone data, filtering, rendered content, dates and pagination remain
upstream-owned. Local reload and browser checks covered the populated 8% fixture,
closed empty state, keyword no-match and 390px layout without horizontal overflow.
Deadline/overdue, tracked time, org context, light appearance and pagination were
not newly exercised. No new fixtures or deployment; custom intro is English.

## Local global Issues preview

The official 15.0.7 dashboard Issues template now has Soda styling and new
checklist artwork. Native query/filter/count/context and shared issue-list logic
remain; Pull requests now shares the same layout with its own heading and icons. Local reload succeeded.
Chrome exercised six populated issues, type switching, closed/no-match empty
states, oldest sorting and 390px layout without horizontal overflow. Pull requests
retains its native list partial. Light appearance, org context and pagination were
not newly exercised. No appliance deployment; custom intro copy remains English.

The Pull requests list now shares the Soda list layout and dedicated collaboration
artwork. Local Chrome checks covered reviewed-by filtering, open/closed/merged
fixture rows and review summaries, no-match search and 390px layout without
horizontal overflow. Native review filters, query state and permissions are
unchanged. No new fixtures or deployment; light mode/pagination not newly checked.

## Local signed-in dashboard preview

The personal home feed now has an official Soda dashboard template override,
new generated workbench artwork, responsive feed/sidebar layout and branded
native empty guide. Native account/org navigation, alerts, heatmap, activity
partial/pagination and Vue repository/organization controls remain composed
upstream partials. Shared shell styles match the explorer; Forgejo account theme
state remains authoritative. Custom dashboard copy is English for now.

The local stock 15.0.7 preview reloaded successfully. Chrome verified empty feed,
repository/organization tab switching, light and auto/dark themes, appearance
navigation and 390px layout without horizontal overflow. Account theme was
restored to `forgejo-auto`. Existing Alice authenticated HTTP rendering returned
200 with populated activity and native repo-list markup; its populated layout was
not visually checked. Organization/team contexts, heatmap and feed pagination
were not newly exercised. No new fixture data or appliance deployment.

The dashboard context switcher now uses a compact Soda trigger and rounded
menu with palette colors, monospace caption, active state and keyboard focus.
Its stock markup and context links are unchanged. Local Chrome inspection
verified the open menu, ArrowDown expansion and personal-context navigation;
organization switching was not newly exercised. CSS-only change and local
template reload; no deployment.

## Forgejo illustration consistency

Reviewed all nine page illustrations together and regenerated the repository
explorer, Users and Issues assets using Home/Dashboard as style references.
The replacements align robot proportions, paper materials and scene balance;
canonical logos remain unchanged. PNG alpha and the cream contact sheet were
inspected, then all three pages were visually checked in local Chrome dark mode
after template reload. Asset query versions refresh cached images. No appliance
deployment or backend changes. Prompts and decisions are recorded in
`assets/branding/forgejo/art-consistency-review.md`.

## Local repository explorer preview

The stock Forgejo 15.0.7 local Docker preview now uses a Soda repository-explorer
wrapper and scoped CSS with the approved folder artwork. Native search, list,
filters, sorting, pagination and main navigation remain upstream partials. Guests
share the login/home theme preference; signed-in pages use the same Soda palette
with light/dark selection inherited from the native account color-scheme.
Local browser checks exercised matching and empty searches, alphabetical sorting,
not-archived filtering, light/dark switching and a 480 CSS-pixel narrow layout
without horizontal overflow. The preview now contains 21 public repositories with sample descriptions/topics;
browser checks confirmed 20 rows on page one and one on page two. Language
variants and a fresh signed-in journey were not exercised. This is local
preview evidence only; appliance staging/deployment remains unchanged.

The Users and Organizations directories now share the Soda shell with distinct
headings and separate newly generated papercraft artwork for individual contributors
and a shared organization workshop. Browser inspection confirmed the page-specific
asset references after local template reload. Native user-list privacy/email
conditions, search, sorting and pagination remain upstream-owned. Local browser
checks covered people search, organization sorting/empty state, light/dark
appearance and 480 CSS-pixel layouts without horizontal overflow. No sample
organizations were added, so populated organization rows remain unexercised.

Signed-in explorer parity now includes the Soda logo/palette, themed native account
menus and a shortcut to native appearance settings. Local Chrome checks using the
existing Vince session covered light and auto/dark themes, profile/admin links,
390 CSS-pixel mobile navigation without horizontal overflow, search and page two.
Vince's original `forgejo-auto` preference was restored after verification. Native
navbar permission conditions are unchanged; private-repository authorization and
additional theme families were not newly tested. No appliance deployment occurred.

Branded empty states now cover the three explorer directories through a shared
presentation partial, used only when native result collections are empty. Native
populated lists and visibility/creation authorization are unchanged. Local browser
checks verified repository reset, people search clearing, organization empty
states for guests and the signed-in administrator, dark desktop and light narrow
layout. Empty state copy remains English; no translations or deployments occurred.

## Accepted native evidence

U08 covers the named infra client → isolated `soda-test` journey, **not** a fresh
appliance install, whole-host upgrade, new UI, release or aarch64 proof:

- `f233a4a`: actual existing-container stop/start and guest reboot preservation.
- `c96c108`: exact-image fresh project and different-UID/default-user/PTY/nested
  exec, SQL and Bob's engine denial. The create-time SYS_PTRACE correction was
  exercised; this does not authorize a privileged parent or unrestricted host socket.
- `8b823db`: full native build/seal/aggregate check; backed-up copied populated-v3
  startup rehearsal and affected-component rollout. Go, Cockpit 60 tests, then-
  dashboard 21 tests, 30 build-fixture and nine staging tests passed at that scope.
  Subsequent corrected operator-probe coverage brought build fixtures to 31.
- After rollout: independent operator/Alice/Bob OAuth/navigation/logout, current
  connection authorization/public keys, root Cockpit/PAM and existing-nobody denial,
  direct own-key SSH/PTY/SCP/SFTP, cross-project/sudo boundaries, retained HTTP/SQL,
  separate personal Git agents/remote refs and shared executable identity checks.
  Four environments and declared state survived; probe files were explicit additions.

Earlier lifecycle evidence was reused **only for unchanged mechanisms**, not claimed
as a new `8b823db` reboot. An image-layer-ID equality probe failed; retained content/
mode/owner/link/capability comparisons found only Tea binary content changed while
runtime configuration/dependency content was unchanged. Native Tea version checked.
An existing-output browser observation failed before a new exclusive observation
path was used. Keep these failures, not just successful retries.

The corrected operator script **fails** on missing
`/etc/profile.d/soda-console-welcome.sh`; the earlier apparent success is invalid.
Tailscale was `NeedsLogin`; zero runners/listeners/capacity were observed. These are
not enrollment or provider-job proof. Service/package observations do not certify
interactive console, visual branding or complete operator journeys.

### Evidence locations

These are retained references, not commands to rerun or evidence revalidated by
this documentation cleanup. Private directories/files remain restricted.

| Location | Retained purpose |
| --- | --- |
| `.artifacts/logs/u08-closure-*` | Build/check exit records, `rollout-8b823db`, host/byte binding, corrected operator failure, browser/Cockpit/PAM, developer/exec/workload/client reads and four-project/image comparisons |
| `.artifacts/logs/u08-completion-*`, `.artifacts/logs/u08-ptrace-*` | Earlier lifecycle, runtime diagnostics, fresh-project exec/Git/workload evidence and failures |
| `.artifacts/test-vm/u08-8417a90/` | Original user/project inputs, bindings and observations |
| `.artifacts/test-vm/u08-completion-952f3b3/`, `.artifacts/test-vm/u08-completion-c96c108/` | Completion inputs, before/after snapshots, connection/Git/client evidence |
| `.artifacts/test-vm/u08-closure-8b823db/` | Merged-candidate browser observations |
| Guest `/var/lib/soda/u08-completion-8b823db/` | Private consistent DB/config/key/helper/runner/unit/prior-image backups and rehearsal |
| `.artifacts/retained-native-u08-c96c108/` | Earlier artifacts moved intact before the merged build |
| `.artifacts/retained-worktree-builds/` | Retained builds; `retention.json` maps old worktree/log paths |
| `.artifacts/retired-dashboard-88fc21f/` | Ignored React outputs/dependency cache moved out of source, not erased |

Preserve project IDs `p4a1ba7562c740b1419169fb5`, `pa7cfcfd898ce306d6b23836b`,
`ped30b9d6932974b14feb2278`, `p7b41edaf83f10a6fd7e579bf`, all identities, keys,
roots, dirty checkouts, workloads/volumes, private inputs and backups. Old backups
predate later writes and are not lossless rollback. The live VM overlay depends on
its retained base. See [local access and retention](local-testing.md).

## Latest source checks

| Revision | Actually executed; not installed acceptance |
| --- | --- |
| `752079e` React removal | Full Go suite; web/Forgejo/nativebuild races; 31 Python build fixtures; Cockpit types and 60 tests; shell/document/whitespace checks. Logs `.artifacts/research/react-removal-88fc21f/` |
| `9f3baa7` HTMX removal | Full Go suite; web/store/Forgejo races; 31 Python build fixtures; shell/document/whitespace/caller checks. Initial OAuth schema/scope test failures and final passes retained in `.artifacts/research/htmx-removal-752079e/`. Cockpit not retested in that slice |

Go checks used cached 1.26.7, readonly modules and disabled resolution; some results
were cached. No dependencies, images, native stage or VM state were changed in
these removals. HTTP join/failure coverage was adapted to the retained JSON API,
not discarded with the HTML forms. Retired browser scripts remain in Git at their
matching revision and must not be run against the new API-only source.

## Minimal UI source inspection

After `c65aa37`, inspected retained Forgejo 15.0.7 header/footer hooks, native
`<dialog>` styles/browser usage, loading CSS, clipboard delegation and selected
Fomantic components. Recorded the [bounded candidate and control states](forgejo-frontend-integration.md#minimal-button-drawer-and-loading-candidate):
footer-hook markup plus moving our own button into the existing repository action
row, scoped right-aligned dialog CSS, content-only native spinner and native copy
controls. No full header override, new tab or library is needed for this candidate.
The row selector/initialization, browser layout/accessibility and authenticated
connection remain untested. Current Soda Origin/CSRF checks and two-origin proxy
prevent simply wiring native-page fetches; a same-origin namespace remains a
candidate requiring actual callback/cookie/config/route review, not a selected API.

Only source inspection/documentation/link/whitespace checks ran in this slice.
No production source, payload, dependency, browser/native test or installed state
changed. This is not a working drawer or a claim that the whole integration is
four static files. The existing-account terminal remains a separate follow-up.

## Implementation planning

After `64aad1f`, expanded the existing [Sodaspaces plan](sodaspaces-plan.md), not a
parallel roadmap: same-origin API/OAuth contract, repository-scoped reads, small
read-only hook/drawer delivery, explicit access actions, native-browser validation
and separately approved rehearsal/cutover. The candidate uses Go/JSON plus vanilla
JavaScript and native `<dialog>`, with no added HTMX or frontend build. Fixed
`/-/soda/` routing, single-origin configuration, scoped cookies, repository return
context and stable-ID API changes are **planned only**; current source is unchanged.

Inspected actual Go/config/setup/staging callers and retained Forgejo 15.0.7 routing
and OAuth application handlers. Native callback editing need not rotate the secret;
the upstream API PATCH does. Recorded that distinction and the template allowlist
gap in the integration guide. Source hashes and documentation checks are retained
in `.artifacts/research/sodaspaces-plan-64aad1f/`. Only source inspection and
Markdown/link/whitespace checks ran; no product tests, build, browser, dependency,
provider, native or private-state actions occurred. The terminal stays separate.

## Test ownership clarification

Following `81bacca`, removed broad upstream-regression requirements from the
Sodaspaces plan and native-validation guide. Tests for removed standalone frontends
and duplicate forge adapters were already deleted in `752079e`/`9f3baa7`; no further
upstream-only test files were identified in the current tracked inventory. Retained
Forgejo client/credential/branding tests exercise Soda-owned code, as do native
project Git/access and Cockpit checks. Keep those and narrowly targeted integration
smoke checks; do not recreate upstream business-logic/conformance suites.

This change is documentation-only: source/test inventory inspection and Markdown
link/whitespace checks, no product test execution or installed-state changes.

## Login design font assets

Downloaded the website's exact Fontsource 5.3.0 Latin WOFF2 selection into
`assets/branding/fonts/`: Fraunces variable 100–900, Barlow 400/600 and IBM Plex
Mono 400/500, all normal style. Added relative-URL font-face CSS, original family
OFL licenses and package/file provenance. Existing `assets/branding/theme/palette.css`
remains the shared color source, unchanged. These are source assets only; no
Forgejo template, running preview, native staging or deployment was changed.

Verified published archive SHA-512 integrity, font signatures, local CSS paths and
file SHA-256 values. No build, font-rendering/browser test or native validation ran.

## Local branded login preview

With user authorization, added `appliance/forgejo/templates/user/auth/signin.tmpl`
and the custom header CSS hook, plus `assets/branding/forgejo/login.css` and the
approved original papercraft PNG. The login shell uses the website's local fonts,
canonical logo and unchanged shared palette. Native `signin_inner`, head/footer
and scripts remain upstream-owned. This is the light login design; responsive CSS
hides the illustration below 900px. The Sodaspaces drawer remains unimplemented.

Recreated only `sodaos-local-forgejo` on Docker Desktop to bind source directories
read-only, retaining `sodaos-local-forgejo_data` and port 3300. An initial mount failed
because nested mountpoint directories were absent beneath a read-only parent;
created those empty local mountpoints and startup succeeded. The existing other
preview and appliance VM were untouched. No appliance stage/install changes.

Checks: login HTML and all sampled CSS/font/palette/logo/image URLs returned 200;
Alice's native form sign-in succeeded; wrong-password submission rendered the
native error inside the new shell. Image alpha data was verified. An exploratory
foreign-Origin rejection assertion failed (HTTP 200, also with cross-site fetch
metadata), so these probes do not establish CSRF protection; no native middleware
was changed. Browser automation was blocked by the user's password-manager panel;
the user inspected the preview and reported it looked good. Automated mobile,
keyboard, provider/passkey, account-link and CAPTCHA browser checks remain unrun.
Source whitespace checks passed. No full build, test suite or native acceptance.

## Login viewport correction

Removed Forgejo's inherited 80px wrapper bottom padding and first-section margin
on the login page. The flex layout now reserves the footer's actual height instead
of assuming a fixed footer size; artwork height and compact spacing adapt to shorter
viewports. Content may still scroll when genuinely taller than the available space.
Bumped the login CSS URL and reloaded templates only in the local preview.

Browser measurements confirmed document height and footer bottom equal viewport
height at 1654×970, 1366×768 and 390×844; mobile width was also exactly 390px.
Restored the browser viewport afterward. Initial measurements used cached CSS;
the versioned stylesheet loaded the correction. Whitespace checks passed.

## Login theme toggle

Added a single borderless sun/moon button at the top right, shared-palette dark
colors and the canonical dark logo. The guest preference follows system appearance
until explicitly selected, persists in origin/subpath-scoped localStorage, syncs
across tabs and tolerates blocked storage. A head script initializes appearance;
Forgejo's native theme attribute and authenticated account preference are unchanged.
No authentication/provider or appliance deployment changes.

Six Node state tests passed (system changes, explicit choice, toggle/persistence,
blocked storage, storage events and invalid/subpath values). Browser checks confirmed
system dark initial appearance, switching to light, correct next-action labels,
persistence after reload and no desktop vertical overflow in dark mode. The user's
existing “Soda dashboard” wording edit was preserved separately from this commit.

## Public homepage and texture removal

Removed the experimental paper texture asset and CSS references, restoring the
smooth login button. Added the native `home.tmpl` override and scoped `home.css`
for the public homepage: Soda welcome copy, approved papercraft artwork, sign-in
and repository exploration links, shared guest theme toggle and native footer.
The authenticated dashboard is unchanged. No account/authentication handlers,
provider configuration, appliance staging or deployed VM were changed.

Browser checks covered light/dark desktop appearance, shared theme on navigation
to login, native repository-explore and login destinations, and 390px mobile layout
with no horizontal overflow. Desktop homepage height matched the 970px viewport.
Verified login's computed background contains only its gradient, no texture.
Native public HTML/assets served successfully; whitespace checks passed. No full
build or native validation ran. Source is live-mounted only in the local preview.

## Remaining work and permission boundary

- Implement/prove the supported native button/drawer/authenticated Soda connection,
  then explicit create/join/key/connection controls and existing-account terminal.
  The verified template hook alone is not this integration. Stop if it needs a fork.
- Rehearse exact candidate/config/grants/populated-state preservation before a
  separately approved cutover; finish fresh/populated native product proof and
  independent native aarch64 validation. No current UI/final product is accepted.
- Complete console delivery/interactive proof, Tailnet and both providers' real
  runner journeys, intended-client routes, native branding and package/tool closure.
- Close [support-tool validation gaps](native-support.md#remaining-validation) and
  [actual-artifact licensing/source obligations](licensing.md). Optional media and
  incomplete outside helper ports are not product gates.

Local source builds/tests were authorized on this development machine. Existing
fixture/reboot grants have been used; there is no new permission for deployment,
restart, fixture creation, provider mutation, routing, destructive cleanup or other
targets. Check actual liveness/addresses only within the applicable scope; recorded
routes and agents are not promises of present availability. No new native actions
or evidence inspection occurred during this documentation cleanup.

## Documentation history

The old M/U/P roadmaps, dashboard inventory, 179-group forge audit and detailed
native audit are removed from active documentation, not from Git. Their complete
text and the chronological 2,076-line handoff remain at `9f3baa7`, for example:
`git show 9f3baa7:docs/implementation-status.md`. The native audit's remaining checks
are condensed into the support guide, not declared resolved. Original source/
license findings and all private evidence survive; old milestone labels in tool
arguments/evidence remain valid identifiers, not active roadmap assignments.

This cleanup changes Markdown/links only. Performed documentation link/anchor,
retired-reference and diff-whitespace checks; no build, product test, dependency
resolution, generated provisioning, service/provider/network action or data cleanup.
Checks covered 55 Markdown files, 218 local relative links and 17 Markdown anchors
with no errors; six retired documents have no active references. Logs:
`.artifacts/research/docs-cleanup-9f3baa7/`.

Local PR fixture follow-up (2026-09-08): added seven user-requested PRs through
native APIs within alice/activity-workbench. Browser confirmed 8 open/2 closed,
review summaries and conflict indicator; API confirmed a native draft and
non-mergeable conflicting PR. Added a new generated collaboration image selected
only for Pull requests. No deployment or non-fixture repository changes.

Milestone artwork/fixture follow-up (2026-09-08): dedicated generated steps/flag
illustration now replaces the reused checklist image. Prompt and provenance:
`assets/branding/forgejo/milestones-art-prompt.md`. User-authorized native API writes
added ten milestones and thirty linked issues within the existing three local
fixture repositories. Browser confirmed 9 open / 2 closed milestones, 0/25/33/50/75/100%
progress examples, overdue/upcoming/no-deadline states and empty milestone content.
The ignored one-shot execution record is `.artifacts/local-forgejo/seed-milestone-fixtures.py`;
do not blindly rerun it. No non-fixture repository writes or deployment.

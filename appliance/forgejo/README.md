# Soda Forgejo presentation overrides

Targets stock Forgejo **15.0.7**. The login wrapper is adapted from the embedded
`templates/user/auth/signin.tmpl`; upstream project: https://codeberg.org/forgejo/forgejo.
Forgejo's GPL-3.0-or-later terms apply to the adapted template; see
[licensing obligations](../../docs/licensing.md). Its original embedded wrapper SHA-256:
`b19374666dc4bff231aa49d60d2459cfebf565028edf67b439b3d093fc3a7547`.

Soda changes the wrapper into a branded story/form layout and adds a scoped
stylesheet via the official `custom/header` hook. The existing
`user/auth/signin_inner` template is invoked unchanged, including alerts, internal
sign-in, account-link mode, password settings, CAPTCHA, providers, WebAuthn errors,
registration and password-recovery controls. Native head/footer and scripts remain.
No authentication handlers, form fields or security middleware are replaced.

## Notification bell preview

Source now includes a signed-in `custom/footer.tmpl` panel, a compact branch in
`user/notification/notification_div.tmpl`, `custom/soda/notification_preview.tmpl`
and `notification-preview.{css,js}` under the branding asset directory. It requests
Forgejo's own session-authenticated HTML fragment using its existing HTMX bundle,
not a Soda API or a JSON-to-HTML proxy. The preview displays up to five native inbox
entries (unread plus pinned), with a permanent “View all notifications” link.

Both desktop/mobile bells keep their native links/badges; only ordinary clicks
are enhanced after HTMX initializes. The script uses DOM events because the bundle
does not expose `window.htmx`. Preview IDs/styles are deliberately separate from
native full-page notification replacement hooks. Keep that boundary on upgrades.

Focused Go/browser checks pass, and the user-authorized local template reload is
complete. Real signed-in populated and empty inboxes, both bells, mobile containment,
Escape/focus and full-page coexistence were checked without changing unread counts.
Production delivery remains separately pending. See the
[plan/evidence](../../docs/notification-preview-plan.md) for exact limits, including
unexercised native pinned/event cases and a pre-existing 320px navbar overflow.
Do not mark entries read or change fixture data merely to preview this feature.

## Asset mapping

| Source | Forgejo custom destination |
| --- | --- |
| `templates/` | `/data/gitea/templates/` |
| `assets/branding/forgejo/` (repository root) | `/data/gitea/public/assets/soda/forgejo/` |
| `assets/branding/fonts/` | `/data/gitea/public/assets/soda/fonts/` |
| `assets/branding/theme/` | `/data/gitea/public/assets/soda/theme/` |
| `assets/branding/source/` | `/data/gitea/public/assets/soda/source/` |

The preview's `.artifacts/local-forgejo/compose.yaml` binds these source directories
read-only; Docker volume `sodaos-local-forgejo_data` retains its users/repositories.
CSS and artwork changes need a browser refresh; template changes need:

```sh
docker exec --user git sodaos-local-forgejo forgejo manager reload-templates
```

The stylesheets use `AssetUrlPrefix` through the template and relative CSS imports.
Guest pages use the shared light/dark palette; signed-in pages derive their
appearance from Forgejo's native account theme. Below 900px the login illustration
is hidden to prioritize signing in. The English
brand copy is authored here; native form labels retain Forgejo localization.
No password-manager replacement, fake theme switch or unsupported sign-in option
is added. Forgejo's native footer retains language/license access.

The papercraft PNG is the user-approved original generated illustration, copied
unchanged from `.artifacts/design/login-papercraft-v1.png`. Canonical logo artwork
and the shared palette are reused unchanged. Fonts retain their family OFL notices.

## Scope and checks

Only the isolated local preview was updated. Appliance staging does **not** yet
ship these overrides/assets; template allowlisting, conflict refusal, full notice
packaging and separately authorized deployment remain required before rollout.
See the implementation handoff for actual checks and remaining validation.

## Presentation component contract

Soda pages compose a small, opt-in presentation vocabulary around native Forgejo
templates. `.soda-page` establishes the shared palette, typography, navigation and
footer treatment; include `data-signed="true"` or `"false"` so account and guest
theme selection remain distinct. `.soda-page-container` supplies the common content
width. Where native pages share only a header or an empty helper hook,
`.soda-page-marker[data-signed]` opts the enclosing `.page-content` and its
navbar/footer into that same full-page shell without copying every leaf template.
This marker is also forbidden inside widgets or the future drawer. Page
stylesheets should contain only layout or presentation specific to that page.

The partials under `templates/custom/soda/` accept fixed presentation data:

- `page_intro` requires `TitleID` and `Title`; `Eyebrow`, `Description`, `Artwork`
  and `Class` are optional. `Artwork` is a filename below the Soda Forgejo asset
  directory and is decorative.
- `empty_content` accepts `Title`, `Icon`, `Eyebrow`, `Description` and `TitleID`.
  Provide a title or descriptive message. The caller decides that the native result is empty, owns the
  `.soda-empty` wrapper and renders any permitted actions.
- `theme_toggle` accepts an optional `Class`. It preserves the single hidden
  `#soda-theme-toggle` hook used by the guest-theme script.
- `guest_theme` accepts the native `Page` context and a fixed `Placement` of
  `head` or `navbar`. It shares the anonymous route gate between the script and
  navbar toggle; native sign-in/home/setup wrappers own their local toggle.

Prefer open sections over enclosing cards. Use gaps for section separation;
reserve borders for controls, data rows, alerts and surfaces that need a boundary.
Removing a decorative divider must preserve its spacing.

Repository issue/PR, milestone, project, release, wiki and file-editor forms
share layout and control presentation in `components-forms.css`. Page styles
should retain only specialized editor behavior; avoid reintroducing per-page
widths, header cards or independent form-control scales.

Keep spacing compact: shared panels use 18px desktop / 16px narrow insets,
general list rows use 16px vertical padding, and section gaps generally use
16–24px. Milestone, issue and PR rows share 24px vertical padding.
Use the shared components before adding page-specific spacing; preserve readable
type and native control targets.

Each responsibility has one CSS owner:

| Owner | Contract |
| --- | --- |
| `components-buttons.css` | Shared button type, shape, neutral surfaces, native primary palette binding and joined-group edges. Native colored, loading, disabled and overflow behavior remains upstream. |
| `components.css` | Full-page shell, semantic colors, shared dimensions, native navbar/footer and page focus. `.soda-page-container` owns content width. |
| `components-intro.css` | `.soda-page-intro` heading, copy, artwork and compact variant. |
| `components-toolbar.css` | `.soda-toolbar` composition; independent `.soda-tabs`, explicit toolbar actions, and bounded native search/dropdown adapters. `.soda-context-switcher` wraps the unchanged native dashboard navbar. |
| `components-forms.css` | `.soda-form.ui.form` fields, labels, help, control states, actions and `.soda-form-section` fieldsets; a positive structural adapter for principal native settings/auth forms. Nested table/row/dialog action/search forms retain native sizing unless they explicitly opt in. Personal password and key-add forms keep `ignore-dirty` and panel hooks while opting into `soda-p-form`. One native CSS nesting block owns both callers. |
| `components-settings.css` | Shared personal/repository settings shell, compact grouped navigation, page gutters, open sections, 40px body inset and inventory action placement. Organization/administrator callers keep their native sidebar. Panel padding excludes native tables; nested row/dialog forms are not cards. |
| `components-list.css` | `.soda-list` wraps a direct native list; row spacing/metadata and native pagination. Shared work-item appearance covers native `#issue-list` and `.soda-milestone-row`. The list itself never carries `.soda-list`. |
| `components-empty.css` | `.soda-empty` presentation/actions plus adapters for native dashboard and issue-search feedback. |
| `components-guest.css` | Guest semantic theme, shared home/login shell and self-contained theme toggle. |

`.soda-toolbar-action--primary` explicitly selects a filled toolbar action.
Explore navigation delegates to native `explore/navbar`, including its overflow
behavior and visibility gates. Native menus, search forms and field markup stay
upstream-owned; these are CSS adapters, not replacement interactive controls.

Family adapters also have explicit owners. `repository.css` owns repository
identity/navigation and ordinary content width; native `.fluid` views retain their
wider canvas. Code, issues, releases/wiki and settings details belong to their
named family files. `projects.css` owns shared project lists and boards across
repository, user and organization contexts. `packages.css` and `code-search.css`
each cover their own feature. `admin.css` and `organization.css` own their
respective shells; their detail files own specialized content. Webhooks,
configuration lists, quota disclosure, runner details and moderation each have a
named owner. `.soda-profile-card-context` opts native profile-card callers into
one shared style owner, independently of profile page layout.
Do not restore the retired `admin-org.css` or `workflow-details.css` aggregators.

Shared form controls use low-specificity type exclusions so native focus and
validation states can win. Principal forms that are nested outside the stable
settings boundary opt in explicitly with `.soda-form`; do not widen the adapter
to all descendant forms. Lists, code canvases, board scrollers and native tables
need their own structure; a generic card or form renderer would obscure it.

`.soda-page` is a **full-page opt-in**, not a widget primitive: putting it inside a
repository drawer would restyle the surrounding navbar/footer. A future drawer
must use its own local root and native or explicitly scoped semantic variables.
Do not add page classes merely to borrow another page's styling. See the
[component audit](../../docs/forgejo-components-audit.md) for the reviewed boundaries.

Callers retain native handlers, context, permission gates, translations, IDs,
forms and scripts. The partials do not accept arbitrary template names, raw HTML or
caller-provided HTML slots. The public marketing hero and branded login remain
distinct layouts; they share guest theme and shell rules while keeping their content layouts separate.

## Guest theme preference

The guest icon button changes the branded public page appearance, including
home, login, explorer, repository, profile and secondary authentication pages.
It initially follows the system color preference. An explicit light/dark choice is stored under
`soda.login.theme:<AppSubUrl or />` in this origin's localStorage. Other tabs sync
through storage events. Clearing the value restores system following; invalid
values are ignored. Blocked storage still permits toggling for the current page.
The head script uses the same anonymous branded-route predicate as the toggle,
and applies the choice before guest content paints. Without JavaScript,
the light layout remains usable and the inactive toggle stays hidden.

Use a separate `data-soda-login-theme` attribute: Forgejo's `data-theme`, theme CSS,
and authenticated account setting remain authoritative for native pages. The
button never submits an account preference or changes authentication cookies.
Its accessible label describes the next action; a focus ring appears for keyboard
use although the resting button has no border. Tests: `node --test
tests/forgejo/login-theme.test.mjs` from the repository root.

## Public homepage

`templates/home.tmpl` replaces the stock public landing content with a Soda welcome
page. `home.css` is scoped to `.soda-home` and uses a dedicated collaboration papercraft asset with the shared
fonts, logos and palette. Sign-in and repository exploration use native routes;
the registration link follows the native `ShowRegistrationButton` context.
The stock head/footer are retained; the public page has its own visible header and
remains separate from the authenticated dashboard composition.

The homepage and login share the existing guest theme key and head script. Native
signed-in account themes remain unchanged. The legacy login-oriented key/attribute
names are retained to preserve already saved guest choices.

`home-papercraft.png` was generated from scratch with the built-in image generator:
two paper robots, each with a laptop, connected to one shared computer on a curved
cobalt workbench. Prompt direction: Friendly Workbench, folded matte cardstock,
cream/navy/mint palette, isolated composition, transparent background, no text.
The login illustration is unchanged.

## Translation scaffold

[i18n/](i18n/README.md) reserves Soda-only locale additions. It is not mounted or
staged: complete native catalogs must be preserved when preparing Forgejo's
replacement locale files. Translation of the custom pages remains future work.

`explore-papercraft.png` is a new standalone repository-explorer illustration,
created with the built-in image generator. Prompt direction: three layered mint,
navy and cobalt cardstock repository folders, cream code braces, and a small cream
paper robot peeking around the edge; matte papercraft, isolated transparent
background. Used by the repository explorer intro.

## Repository explorer

`templates/explore/repos.tmpl` adds the Soda intro and artwork around the unchanged
15.0.7 explore navigation, repository search/list and pagination partials.
`explore.css` scopes presentation to this wrapper. Native main navigation,
authentication links, permission-dependent controls and repository data remain
upstream-owned. The custom extra-links hook adds a guest theme button only here.
Guests share the homepage/login theme state; signed-in users use the same Soda palette and logo, with light/dark selection
following Forgejo’s native account color-scheme. New introductory copy remains English for now.

Local browser checks covered matching/empty search, alphabetical sorting, the
not-archived filter, guest light/dark switching, and a narrow 480 CSS-pixel viewport
without horizontal overflow. At the initial implementation, the preview had one repository; later pagination
evidence is recorded below. Populated language variants remain unexercised. Signed-in
navigation was preserved by source inspection, not a new authenticated journey.

### People and organizations

Forgejo 15.0.7 renders both directories with `explore/users.tmpl`. The override
selects introductory copy using `PageIsExploreOrganizations`, uses separate generated Users and Organizations
papercraft artwork, and retains native search, user list and pagination.
Avatar/profile links and email/visibility conditions remain upstream-owned.
The guest theme hook now covers all three designed explorer pages.

The local preview now contains 21 public repositories owned by Alice. Browser
verification observed 20 rows on page one and one on page two, working people
search, organization sort navigation and its empty state, and no horizontal
overflow on the people/organization pages at 480 CSS pixels. No sample
organizations were created; populated organization rows and authenticated
visibility behavior were not newly exercised.

`users-papercraft.png` depicts three distinct robot contributors; `orgs-papercraft.png`
depicts a shared workshop and project handoff. Both were newly generated with the
built-in image tool, retaining PNG alpha. Exact prompts are recorded in
`assets/branding/forgejo/directory-art-prompts.md`. Local template reload and browser
inspection confirmed each directory references its own asset.

### Signed-in explorer parity

The same wrapper, artwork, spacing, typography, repository rows and Soda palette
are now used before and after sign-in. Signed-in light/dark colors use CSS
`light-dark()` with Forgejo's inherited `color-scheme`; the guest storage value
never overrides the account setting. The Soda logo follows the bundled native
light/dark/auto theme families. Account menus, notifications, create actions and
administrator links are still generated entirely by the stock navbar. A paintbrush
shortcut opens native appearance settings; it does not write a separate preference.

Local Chrome verification with the existing Vince session covered native light
and auto/dark appearance, the appearance shortcut, profile/admin menu links,
mobile menu at 390 CSS pixels without horizontal overflow, repository search and
page-two navigation. The original `forgejo-auto` account preference was restored.
No backend/authentication or permissions code was modified. Additional theme
families and private-repository authorization were not newly tested.

### Branded explorer empty states

The explorer wrappers render `custom/explore_empty.tmpl` only when the native
result collection is empty. Populated lists remain stock partials. A compact
native icon, mint accent, Fraunces heading and Soda action style cover repository,
people and organization directories. Search terms remain template-escaped.
Wording distinguishes a keyword with no matches from records not visible in the
current view, without asserting hidden records do not exist. Reset links clear
search/filter/page state. Creation links follow stock navbar conditions (signed
in for repositories; `CanCreateOrganization` for organizations); native handlers
remain authoritative for creation permissions and limits. New copy remains English.

Local browser checks covered signed-in repository no-match/reset (20 rows restored),
empty organizations with the native creation link, guest people search/clear
(3 users restored), guest organization empty state without creation controls,
dark desktop and light 480 CSS-pixel layout without horizontal overflow.

### Signed-in home dashboard

`user/dashboard/dashboard.tmpl` wraps the native account/context navigation,
alerts, heatmap, activity/guide and Vue repository list. The new workbench hero,
feed card, sidebar and discovery link use `dashboard.css`; the shared page system
supplies the shell, navbar and footer. Native account themes remain authoritative.
The appearance shortcut uses the upstream `PageIsNews` flag.
Organization contexts retain stock navigation and receive an organization greeting.
New custom copy is English pending the existing i18n follow-up.

`dashboard-papercraft.png` is a newly generated supplemental asset; its exact
prompt is in `assets/branding/forgejo/dashboard-art-prompt.md`.

Local stock 15.0.7 template reload succeeded. Chrome checks covered the Vince
empty feed, sidebar tab switching, light and auto/dark colors, native appearance
shortcut, and 390px mobile layout without horizontal overflow. The original
`forgejo-auto` preference was restored. An authenticated HTTP check as the existing
Alice fixture returned the populated native activity feed and repository sidebar
with the new wrapper (200); populated activity was not visually checked. No new
records were created. Organization/team contexts, heatmap and feed pagination were
retained by composition but not newly exercised. No appliance deployment occurred.

### Global Issues overview

`user/dashboard/issues.tmpl` adapts the stock 15.0.7 template with a Soda intro,
new checklist artwork, and unified issue panel. Native search syntax, type/sort
links, open/closed counts, account/org navigation and shared issue list remain
upstream-owned. The shared template now also applies the design to Pull requests, with its own
heading, introduction and status icons.
Shared page, toolbar and list components own the presentation; the former
PR-only `issues.css` adapter was removed. Account theme state remains native. The two custom intro strings remain English. Artwork provenance
and exact prompt are in `assets/branding/forgejo/issues-art-prompt.md`.

Local template reload and Chrome checks covered populated dark-mode rows,
created-by/in-your-repositories switching (2/6 open issues), closed empty results,
keyword no-match, oldest sorting, and 390px populated/empty layouts without
horizontal overflow. Pull requests was subsequently brought into the same styling. Light-mode visual inspection, organization context and pagination
were not newly exercised; this fixture list has fewer than one page of issues.

### Global Pull requests overview

Pull requests uses the same list layout as Issues, with a dedicated collaboration
illustration (`pulls-papercraft.png`; exact prompt in `pulls-art-prompt.md`).
Its native review-requested/reviewed-by filters, branch information, review counts,
merged/closed icons, query state and shared list remain intact. New introduction
copy remains English. Local reload and Chrome checks covered reviewed-by filtering,
one open request, two closed requests (including one merged), review summaries,
no-match search and 390px layout with no horizontal overflow. No new fixture data
or deployment. Light appearance and pagination were not newly exercised.

Additional user-authorized local PR fixtures #9–15 in alice/activity-workbench
include a native draft, review requested from Vince, approved and changes-requested
reviews, assignments, labels, milestone/task progress, multilingual/long titles and
a real one-file merge conflict. Local browser confirmed 8 open / 2 closed requests
and the native waiting-review/approval/change-request/conflict summaries. Native API
confirmed draft=true for #9 and mergeable=false for #14. No application rules or
existing non-fixture repositories changed. The ignored one-shot execution record is
`.artifacts/local-forgejo/seed-pull-fixtures.py` (not safe to blindly rerun).

### Global Milestones overview

`user/dashboard/milestones.tmpl` adapts stock Forgejo 15.0.7 with the shared Soda
intro/toolbar and existing checklist artwork, a repository filter panel and separate
milestone cards. Native repository selection, count/state/search/sort links,
progress, deadlines, tracked time, rendered descriptions and pagination are retained.
Empty results now show the existing translated no-results guidance. Custom intro
copy is English. Local template reload and Chrome checks covered the populated
8% fixture, closed empty state, keyword no-match and 390px layout without horizontal
overflow. Due/overdue dates, tracked time, organization context, light appearance
and pagination were not newly exercised. No new fixtures or appliance deployment.

Milestone artwork/fixture follow-up (2026-09-08): dedicated generated steps/flag
illustration now replaces the reused checklist image. Prompt and provenance:
`assets/branding/forgejo/milestones-art-prompt.md`. User-authorized native API writes
added ten milestones and thirty linked issues within the existing three local
fixture repositories. Browser confirmed 9 open / 2 closed milestones, 0/25/33/50/75/100%
progress examples, overdue/upcoming/no-deadline states and empty milestone content.
The ignored one-shot execution record is `.artifacts/local-forgejo/seed-milestone-fixtures.py`;
do not blindly rerun it. No non-fixture repository writes or deployment.

### Notifications

`user/notification/notification_div.tmpl` and `notification_subscriptions.tmpl`
customize the stock 15.0.7 wrappers. Keep notification IDs, sequence/data hooks
and native forms intact: Forgejo refreshes the notification partial after actions.
`notifications.css` supplies notification metadata, visible row actions and
responsive row placement; shared toolbar/list/empty files own the common visuals. Dedicated artwork provenance lives
in `assets/branding/forgejo/notifications-art-prompt.md`. Existing local fixtures
were used for read/unread and empty-state checks; original states were restored.

### New Repository

`repo/create.tmpl` adapts the native 15.0.7 wrapper and composes the unchanged
creation partials and permission gates. `components-forms.css` owns fieldsets,
inputs, dropdowns, advanced disclosure and submit actions; `create.css` only
adapts template-unit spacing and the native template search. Reuses the repository-folder
artwork. Local UI checks exercised expanded options and narrow layout; no
repository creation or appliance deployment was performed.

## Broad native page families

The stock 15.0.7 repository header/settings seam now reaches 65 full-page templates;
account settings reaches 23; administrator and organization seams reach 58 (33 admin,
8 organization pages, 17 organization settings). Nine secondary authentication
wrappers add registration, recovery/reset/change-password, activation, TOTP/scratch,
WebAuthn and prohibited-login presentation. These counts describe source composition,
not 155 independently exercised browser journeys. Native leaf templates remain
upstream wherever a shared seam suffices; unchanged override copies are not shipped.

The three-task expansion added 183 override files relative to `a81bfae`, reaching
201 template overrides/helpers at `82379b4`. File counts include shared fragments. Detailed
adapters cover code browsing/editing/diffs, issue and pull workflows, milestones,
releases/wiki/projects, Actions and runners, repository settings and webhooks,
profiles/packages/migration, administrator monitoring, account and organization
details, federated authentication and setup. Native source provenance and exact
stock recovery checks accompany the adapted files. Full setup, consent and native
workflow execution are not implied by source composition or local rendering.

The follow-up [component audit](../../docs/forgejo-components-audit.md) removes
dead and competing adapters and records the current owner map. The shared header
registers the scoped feature files; existing page, form, toolbar, tab, list and
empty-state primitives remain the common presentation owners. Shared runner,
webhook, configuration and moderation leaves have their own cross-context files.
Native menus may overflow their
containers; repository action rows wrap on mobile without clipping dropdowns.
Forms retain native handler URLs, security fields, permissions, state, scripts and
semantic danger controls. Narrow auth grids explicitly clear native percentage
padding before constraining width. Personal settings no longer render artwork or an introductory hero. Previously
selected artwork remains attributed in `assets/branding/forgejo/settings-art-prompts.md`.

`custom/soda/guest_theme` owns one presentation gate for the head script and native
navbar toggle. Repository context uses `.Repository`, organization context uses
`.Org`, profile/overview context uses `.ContextUser`, and secondary auth uses native
`.Link`. Setup uses `PageIsInstall` with its own body toggle and no duplicate navbar
toggle. Stock 15.0.7 constructs that Link
from `AppSubURL` plus the escaped URL path (without query state). Prohibited login
retains `PageIsSignIn`, so its override supplies the body toggle, just as login does.
Account themes remain native. Source parity hashes for the adapted wrappers and
focused component/render tests live under `scripts/forgejo_*_test.go`.

The expanded baseline is mounted in the existing isolated local preview. Audit
changes from another worktree can be reviewed using the documented `--local-css`
capture mode without reloading server templates. These overrides do not stage or
deploy to the appliance or implement the Sodaspaces drawer. Current browser
evidence and remaining limitations are recorded in the implementation handoff.

## Presentation contract

The native frontend uses one registry, `templates/custom/header.tmpl`, and no
runtime component framework. `components.css` owns semantic tokens and explicit
presentation roles; `components-forms.css`, `components-settings.css`, intro,
list, empty and toolbar sheets own their respective reusable structures.
Family styles compose those structures and adapt specialized native canvases.

Use `soda-p-editor-container` for the shared repository editor width,
`soda-p-form` on principal native forms, `soda-p-form-host` on an existing
wrapper whose direct child is an unchanged native form partial, `soda-p-heading`
or `soda-p-title` on headings, `soda-p-section` on open sections, and
`soda-p-gap` on blank native dividers. Retain native classes and all form/script
attributes. Existing `soda-form`, `soda-toolbar`, `soda-list` and `soda-empty`
contracts remain valid; do not add a second implementation of them.

Spacing uses 4/8/12/16/24/32px. Inputs retain an 8px radius; buttons use a 6px
radius and 600-weight 14px labels. Standard, compact and icon actions all use
44px targets, including native mini/tiny/small classes. Internal edges
of joined repository controls are square even when a native modal lies between
the action and count. Page and section typography use shared font tokens with
the existing Fraunces, Barlow and IBM Plex Mono families. Ordinary content is
1120px; wide data views remain 1440px or native fluid canvases. Narrow columns
use the shared 240px token and stack at 900px where the native layout allows it.

Open sections are the default. Keep boundaries for alerts, dialogs, code/data
canvases and independently selectable items. Removing a blank divider preserves
24px section spacing. Primary submissions and destructive/ambiguous actions
retain text. Familiar icon utilities keep localized accessible names and native
hover/focus tooltips. Artwork belongs in introductions and empty content.

The centralized role `!important` declarations defeat native attached-segment
borders, surfaces and enforced utility padding. They are scoped to explicit
roles and exclude tables. Native settings leaves not overridden here still use
the narrow main-content form adapter; compact row/search/modal forms are excluded.
Sign-in retains its native partial with a bounded form adapter. Do not replace
these with selectors matching every descendant form or segment on a page.

`tests/forgejo/presentation/inventory.json` accounts for all 229 overrides and
records local and embedded-15.0.7 callers, compositions, roles and required
states. It is test-only, not routing configuration. Role-only source hashes
complement the existing exact upstream-body tests; they do not constitute visual
verification. The reference gallery renders production intro/empty partials,
the actual stylesheet registry and minimal native markup fixtures into ignored
`.artifacts/forgejo-presentation/`:

```sh
SODA_FORGEJO_GALLERY=1 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  go test -mod=readonly ./scripts -run TestForgejoPresentationGallery -count=1
SODA_FORGEJO_LAYOUT_ORIGIN=http://localhost:3300 \
  node --test tests/forgejo/presentation/*.test.mjs
```

Gallery evidence remains separate from native route evidence and does not prove
permissions, providers, form submissions, editor plugins or unavailable admin,
organization and setup states.

Ordinary repository body containers have zero padding. `repository.css` owns
that rule, and the repository header supplies the gap below navigation. Family
styles must not add another top inset to the same container. Fluid or explicitly
padded native canvases retain their separate gutter contract. The native
`repository-container.test.mjs` regression compares Code, Projects, Issues and
Releases at desktop/mobile widths so family-specific padding cannot silently
return on those routes.


## Personal settings structural contract

Personal settings use the native `user/settings/layout_head` / `layout_footer`
seam with one identity/navigation row and one page title. The account avatar,
display name and username come from `SignedUser`. `navbar.tmpl` retains every
native destination and capability gate, including mandatory enrollment hiding
ordinary navigation. Personal, Access & integrations, and Resources use Soda
locale keys; native controls and warnings retain native translations.

`components-settings.css` owns the shared shell, menus, open sections with headings above their bodies
and the 900px transition. `account-settings.css` only owns the personal identity row. `account-details.css` owns the portrait/editor grid,
preference rows and credential/editor compositions. Shared presentation owns
fonts, spacing and control roles. `soda-p-form` explicitly styles nested native
principal forms; `soda-p-control` and `soda-p-compact` retain their presentation
roles with the same shared 44px action size, preserving native behavior classes. Native enforced container width
and max-width require two documented `!important` rules here to prevent double
gutters; the shell uses 1120px usable content plus 24px desktop / 16px mobile gutters.

Profile keeps one identity/address/privacy form and a separate sibling avatar
form. Account keeps independent email/password/deletion handlers, with password
on its native page and a Security link to `account#password`. Appearance keeps
all four saves. Token selection, key verification, WebAuthn, runner configuration,
quota and webhook dispatch remain native. Child titles move into the shell.
Shared runner, secret, variable and webhook adapters use the explicit
`SettingsPresentation` input for personal/repository callers. Cleanup and OAuth
remain personal-only adapters. Each rebinds native root context and suppresses
only the opted-in duplicate heading. Other callers retain their headers and data.

`personal-settings.js` only enhances navigation, disclosure visibility, the avatar
modal and focus. The avatar link progressively opens one native `<dialog>` with
the existing form moved into it. Without enhancement or on server errors the
inline editor remains accessible; no form or authoritative alert is duplicated.
Avatar overlay and dialog styles are scoped in `account-details.css`.
There is one DOM copy of every destination. Without JavaScript the grouped links
and optional editors remain expanded. Ambiguous server errors open every editor;
fragment links reveal the addressed editor. No submission interception, autosave,
account preference writes or substitute authentication state is introduced.

The test-only presentation inventory records all overrides/native callers and
reviewed structural changes. `settings-native-contracts.json` records native
15.0.7 form controls and capability conditions, separately from presentation
hashes. Existing native gate tests and explicit shared-root rendering checks
remain required. The gallery uses production navigation and the actual CSS
registry with minimal native form, preference, security and inventory fixtures.
It is component evidence, not native-route acceptance.

See `i18n/README.md` for complete English catalog generation and the scoped local
activation. Generated catalogs, gallery and screenshots stay in ignored
`.artifacts/`. No appliance staging or deployment is part of this redesign.


Pronoun editing, its privacy control and public-profile display are omitted from
Soda presentation. Personal/admin forms carry existing native values in hidden
fields solely to avoid resetting data on unrelated saves. Forgejo's database,
API and native locale catalogs remain upstream-owned. The attributed
`shared/user/profile_big_avatar` override changes only that display fragment.


The selected C — Tonal design uses tinted primary surfaces with blue text and
fine blue borders, transparent neutral actions, and 14px horizontal padding with
no decorative shadow. Button-local tokens preserve the canonical page palette.
`components-buttons.css` owns appearance and dimensions for introductory/empty
actions, principal forms, repository toolbars and native basic buttons. Bounded
`!important` height, padding and font declarations defeat native mini/tiny/small,
compact and utility size rules; page-family CSS must not redefine those sizes.
Native explicit danger/success variants and loading/disabled behavior remain
intact. Soda danger actions use red text with a tinted hover surface.

Icons, joined count labels, native overflow triggers and adjoining single-value
inputs share the 44px height. Principal-form single-value fields also align at
44px; textareas and multiple-selection controls retain growing content areas.
Native joined internal edges stay square. Explore's desktop tab allowance is
448px to accommodate the 44px overflow trigger; native tab measurement and menu
behavior remain intact. The 320px header retains the canonical 128px logo with
44px mobile actions and reduced gaps instead of undersized targets.


Empty states share `custom/soda/empty_content` and `components-empty.css`.
Use `soda-empty--page` for an empty destination (centered icon/message with a
bounded breathing space), and `soda-empty--compact` for a section inside a working
page (smaller icon and left-aligned heading). Existing `soda-empty--inset` callers
use the compact composition. These are open surfaces without enclosing cards or
dividers. Native callers own empty conditions, translations and permitted actions;
never add a create action to a naturally empty state such as Blocked users.
Blocked-user lists and deploy-key sections now select these roles explicitly.


Heading placement follows one rule: page titles precede the page, section headings
precede their content, and introductory descriptions stay beneath the heading.
Personal settings no longer use a left-hand explanation column or the obsolete
`soda-settings-explained` role. `.soda-settings-section` owns a single-column
composition with a 12px heading/body gap and 32px between sections at every width.
Inventory actions remain beside headings on desktop and stack below on mobile.
Content grids (portrait/editor, checklists and native label/value data) are not
section-title columns and retain their task-specific arrangements.

Every personal/repository settings destination uses one 40px inline-start inset on
`.user-setting-content` or `.repo-setting-content` inside `.soda-settings-shell`. Section headings and fieldset legends return 40px toward
the outer edge. This covers native header/segment pairs, inventories and editors
without requiring the personal section-body wrapper. Nested bodies never add
another inset. `components-settings.css` owns this layout; shared form styles own
only the vertical heading spacing and must not reset its inline alignment.
The native browser check enumerates every permitted settings-menu destination,
verifying the inset, heading alignment and overflow instead of sampling three pages.
Shared native section headers, including nested headers, and form legends use
the same 24px heading token at every width. Disclosure summaries use the shared
button typography. Icon inputs reserve 40px on the icon side; ordinary field
padding must not override native icon clearance.


Static guidance uses native `ui info message` / `ui warning message` with the
`soda-notice` role. This opt-in keeps notices visible inside forms without exposing
inactive native validation messages. Recovery and key-loss guidance precedes the
relevant action. Account deletion retains its native danger message.

Empty access-token, webhook, membership, repository and cleanup-rule inventories
use the shared empty content partial only in the native empty branch. A title is
optional when the translated explanatory sentence already supplies the message.
Populated rows, permissions and actions remain native. Personal webhooks use a
real toolbar instead of an empty heading. Inventory actions accompany section
headings, stacking below on mobile; page-level actions use a left-aligned toolbar.
Submissions follow their fields and guidance, including all four Appearance saves
and Cargo/Chef actions. Native absolutely positioned header actions are returned
to normal flow within the shared settings shell to prevent narrow-screen overlap.

Key, WebAuthn, authorized/owned OAuth and token repository-selection inventories
use the same compact empty-state role. State the absence explicitly; when native
copy only explains the inventory, use the existing localized “No results” title
above that guidance. Empty authorized OAuth content must not
claim that access has already been granted. Personal Actions secrets, variables
and runners opt in through explicit context adapters, also used by repository settings;
organization/administrator callers retain their native empty branches. Runner
setup's last-chance credential guidance is a static warning in the opted-in settings shell. An empty personal cleanup preview omits its blank
heading and table, while populated and nonpersonal previews remain native.
Personal webhook event groups use open fieldsets with their native legends and
controls, without nested decorative boxes.

Cargo's pinned English description is split into its context and consequence
sentences, shown as separate info/warning notices before submission. Other locales
(or unexpected sentence structures) retain the complete translated warning; no
translation text is discarded or replaced by a guessed split. Existing locale
catalogs and activation remain unchanged.


## Repository settings structural contract

Repository settings compose the native repository header with the same
`.soda-settings-shell` used by personal settings. The native identity and unit
navigation remain above one task title and compact destination menus. There is
no repository settings sidebar or repeated attached task heading. Repository,
Access & integrations, and the conditional Actions group retain every native
link and visibility gate. The mobile disclosure exposes all permitted groups in
one expansion; JavaScript only enhances ordinary navigation and focus.

`components-settings.css` owns the 1120px usable canvas, 24px desktop / 16px
mobile gutters, 40px body inset, 24px section type and common action rows.
`repository-settings-details.css` owns only repository-specific composition and
technical data. Generic repository container rules have lower specificity so
this explicit shell wins at every breakpoint; ordinary repository pages retain
their zero-padding contract. File inputs use the shared principal-form width
constraint, including native inline avatar fields. All actions retain the
selected C — Tonal 44px dimensions.

General settings use open Basic, Avatar, Federation, Mirrors, Signing,
Administrator and Danger sections under their existing gates. Units retain one
native POST form and all four native anchors/save controls. Branch/tag
protection, collaborators, deploy keys, webhooks, Actions and LFS inventories
use shared sections, actions, notices and empty states. Child editors receive
one task title and a parent link. Technical tables use
`.soda-settings-table-scroll` where their content needs horizontal scrolling;
code viewers, native dropdowns, provider dispatch and dialog structures remain
upstream-owned. LFS totals stay visible in inventory summaries.

Ordinary repository-name and collaborator/team search fields no longer take
initial focus away from the new page heading. Tag creation is an inventory
landing page; its dedicated edit state retains autofocus. Deliberate child/panel
focus, submitted values, validation, dirty-form behavior, IDs, permissions,
methods/actions and destructive confirmations are preserved. No fields are
submitted, credentials generated or repository state changed by the enhancement.

`repository-settings-native-contracts.json` records exact 15.0.7 controls,
conditions and script hooks for the 22 settings leaves and four unit partials.
The Go navigation test compares all 256 feature/permission combinations with the
exact native navbar. Non-opted organization/administrator shared-partial bodies
retain explicit native-parity checks. The test-only inventory covers 235
production overrides/helpers and both local and native callers.

The separate `repository-settings-{light,dark}.html` gallery uses the production
registry, native SVG assets and the runner editor with minimal native form/inventory
fixtures. `repository-settings-browser.test.mjs` checks seven widths including
900/899px, no-JavaScript fallback, focus, long labels, gutters, file input and
table containment. Gallery output and component captures are ignored artifacts,
not native repository screenshots. Native owner-only, provider, credential,
mirror and mutation coverage must be recorded separately in the handoff.


Milestone list items use `custom/soda/milestone_row` for both global and repository
callers. `components-list.css` owns the appearance shared with native issue and
PR rows: uniform transparent backgrounds, 24px vertical padding, one separator
between complete items, 20px sans-serif linked titles, 13px metadata and matching
progress tracks. `milestones.css` owns compact Markdown previews, aligned native
deadline/progress columns and stacked mobile metadata. Keyboard focus reveals the full
preview when it contains links; the detail page keeps its full native Markdown.
Repository edit/close/reopen/delete actions retain native permission/archive
gates and handlers. Global items remain read-only. Page heading, filters,
navigation, pagination and empty states are outside this partial and unchanged.
Native project collections also reuse `milestone-card`; their existing styles
are explicitly excluded from the new row role to avoid changing those lists.

The native `shared/issuelist` template stays unmodified. Its repository, dashboard,
milestone-content and notification-subscription callers all receive the same row
contract through native `#issue-list`. Status colors, label colors, selection
checkboxes, commit-status popups, assignees, comments, review state and native
links remain intact. Ordinary collections keep their existing 16px rows/18px
titles. The PR-only spacing adapter and repository list corner adapter are removed.
`work-item-lists.test.mjs` exercises isolated native markup samples with the actual
registry, long labels/branches, keyboard selection and unrelated-list protection.
Its component gallery under `.artifacts/work-item-consistency/components/` is
separate from real-route captures and their verification status.

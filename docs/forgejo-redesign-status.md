# Forgejo redesign status

Updated 12 September 2026 in `/Users/vince/Projects/sodaos`, branch `main`.

## Current workspace

The owner moved ongoing work to the canonical `/Users/vince/Projects/sodaos`
checkout on `main`. The redesign through `bcf0fb1` was merged in `b81cab4`.
The deleted worktree is no longer used for development or local presentation mounts.
The six approved subway backgrounds are now source assets and are installed on
localhost:3300.

## Settings Sodaspaces destination

The repository settings Sodaspaces entry is now a direct navigation link, replacing
a triggerless dropdown that remained absolutely positioned and open on desktop.
Its signed-in gate and repository-ID destination are unchanged. The production
settings gallery now includes a signed-in repository so this entry is exercised.
Navigation checks pass across ten widths in both themes, plus three no-JavaScript
widths; native gates, inventory and TypeScript checks pass. The updated navbar
and `components-settings.css?v=18-direct-link` are live on localhost:3300, with
previous files and a component capture in `.artifacts/forgejo-settings-sodaspaces/`.

## Repository action bar refinement

Watch, star and fork now pair their native action and separately linked count in
one continuous 1px frame. Counts use aligned tabular numerals; controls share 16px
icons, 12px mono labels, 44px targets and quieter edge colors. RSS is a matching
square on wide screens and a labeled row in the compact menu. Sodaspaces keeps
its native insertion point and gets a matching red outlined control. Below the
existing 1000px container threshold, the same native actions occupy aligned rows
in a bounded 288px disclosure. Native forms, counters, permissions, workspace
behavior and the fork modal remain unchanged.

`repository.css?v=21-actionbar` is delivered on localhost:3300. Previous CSS/header
files and light/dark captures are under `.artifacts/forgejo-repo-actionbar/`.
The read-only live test passes at 320, 640, 720, 800, 960, 1000, 1100 and 1440px
in both themes, checking joined edges, count links, target sizes, menu containment
and Escape dismissal. Seven existing disclosure tests, branding/inventory checks
and changed-test TypeScript checks also pass. No watch, star, fork or workspace
mutation was performed.

## Quieter button borders

Forgejo's shared primary and secondary actions now use 1px borders, including
homepage, login, forms and repository controls. Migration choices and notification
retry/footer actions follow the same lighter treatment. Focus outlines and target
sizes retain their existing values. Only these three stylesheets and their cache
versions were updated on localhost:3300; previous files are retained in
`.artifacts/forgejo-thin-buttons/before-delivery/`. A read-only browser review
passed 166 visible-action checks across six routes at 320, 800 and 1440px in both
themes; primary focus outlines and minimum target size also pass. The migration
Git option retains its separate 6px red accent. Source inventory, branding and
changed-test TypeScript checks pass.

## Dashboard repository browser refinement

The native Vue sidebar uses understated mono tabs, unboxed counts and red selected
underlines. One thin closed frame encloses search, filters and results; the native
attached results wrapper has no separate border. Search has one divider and a
transparent filter button. Selected filters explicitly override Fomantic's important
corner radius; the filter boundary also overrides Vue's important zero-width rule.
Empty results do not double the bottom divider. Repository links retain 48px targets,
and the discover callout uses a smaller condensed heading. The sidebar stays
full-width below the feed through 1199px, then moves into a 344px desktop column.
Native Vue search, filters, overflow menus and tabs retain their behavior.

Only `dashboard.css` and its `v=22-simple-sidebar` header link were delivered to the
existing localhost:3300 mount. Prior files remain under
`.artifacts/forgejo-sidebar-refinement/before-simplification/`. The focused browser
check covers 320, 640, 720, 800, 960, 1199 and 1440px in both themes, including
search, fork filtering, empty results, organization tabs, dropdown bounds, long
names and 44px minimum targets. It now explicitly checks the reported single-fork
state: zero-radius transparent selection, four one-pixel outer edges and no nested
list borders. The existing screenshot session receives controlled repository GET
responses; no native data is changed. Captures, including `simple-forks-*`, are in
the same artifact directory. Branding, template inventory and changed-test TypeScript
checks pass. This receipt covers the sidebar, not every dashboard workflow.

## Subway welcome background and canonical local delivery

Six distinct WebPs replace the website's soda-bar photographs: mobile, tablet and
desktop, each with day and artificially lit night variants. Approved PNG masters,
export hashes and prompt summaries live in `assets/branding/forgejo/backgrounds/`.
Only the active crop downloads; the six WebPs total 1,244,550 bytes. The welcome
page retains its opaque centered frame and shared footer divider. Right anchoring
preserves the vending machine, portrait desktop windows use the tablet composition,
and mobile has space below the frame to reveal the foreground. The header requests
`home.css?v=18-subway` to avoid stale cached presentation.

The local container now mounts presentation files from
`.artifacts/forgejo-delivery/2026-09-12-subway/`, using that directory's
`compose.override.json` with `.artifacts/local-forgejo/compose.yaml`. Before the
move, 253 templates and 108 available public files were copied and verified.
A fresh stopped-state private backup of `/data/gitea` and `/data/git` is retained in
`private-backup/` under the delivery directory. The same named data volume,
loopback port and application configuration remain. Account, repository and issue
counts match the backup.

Fresh-browser checks exposed assets already missing after the worktree deletion.
Missing runtime modules were rebuilt from the deployed historical revision;
all 19 surviving modules matched that build byte-for-byte. Missing styles, symbols,
locked terminal dependencies and native-theme import bridges were also restored.
The exact revisions and paths are recorded in `restored-runtime-assets.json`;
this recovery does not upgrade the running JavaScript to newer source behavior.

Verification: all 14 live homepage theme/viewport combinations pass, including
320–2560px widths and a tall 1716×1975 window, with opposite system preferences.
Checks cover one matching image request, no failed resources or page errors,
no horizontal overflow, an opaque centered frame and one footer divider.
Both themes were also reviewed visually. All six served images and live home CSS
match source bytes; the six retired image URLs return 404. Five branding/inventory
checks, changed-test TypeScript checks and the preview build pass. Logs and the
mount/data preservation receipt are retained in the canonical delivery directory.
These checks cover the homepage and asset recovery, not authenticated workflows.

## Half-width desktop working layout

The owner explicitly requires browsing Forgejo beside a terminal. The standing
[responsive workspace contract](forgejo-frontend-integration.md#responsive-workspace-contract)
covers all page families at 640, 720, 800 and 960 CSS pixels in both themes.
Creation forms and editor metadata now stack through 1100px; homepage feature
rows stack below 1024px. Repository actions retain their existing container-based
compact disclosure, and settings keep their existing compact navigation.

A subsequent front-page-only pass tightens the 640–960px composition: paired
primary/secondary actions, a stronger headline, shorter reading lines, aligned
24px content gutters and compact numbered feature rows. Split panes use 32px
outer vertical spacing instead of the former 35svh trailing area; mobile retains
a modest 96px photographic margin. Full-width desktop keeps its two-column hero.
The live header now requests `home.css?v=20-compact-home`. All 22 guest homepage
cases also check action columns, aligned content edges, 44px action targets and
unclipped labels. Light/dark captures at 390, 720, 960 and 1440px, source/type
checks and prior delivery files are in `.artifacts/forgejo-home-split/`.

Only `components.css`, `form-pages.css`, `home.css` and their versioned links were
updated on the canonical local delivery mounts. Prior files are retained under
`.artifacts/forgejo-half-width/before-delivery/`; native runtime and data are unchanged.

A read-only audit using the existing screenshot fixture passed at four split widths
in explicit light/dark themes: dashboard, exploration, public profile, repository
code/issues/pulls/commits/releases, creation, migration, organization creation and
personal settings/appearance. There were 120 navigations, including eight expected
signed-in redirects from the login URL. No page or checked content overflow was
found. The separate guest homepage test passes 22 theme/viewport combinations.
Production component, editor, administration/form reference and settings navigation
checks pass across split widths, mobile and full desktop, including keyboard and
no-JavaScript settings access. Source branding/inventory, changed-test TypeScript
checks and preview build pass. These are representative page-family checks, not
an assertion that every authenticated or operator workflow was exercised. Evidence
and before/after 800px captures are in `.artifacts/forgejo-half-width/`.

## Desktop login station

The owner-selected original text-only train photograph now fills only the left
half of the sign-in page at widths of 1200px and above. It shows an approaching
train with pronounced motion blur, sharp brutalist station architecture and a
small soda machine down the platform. The native form stays on the right. Below
1200px the compact login remains and the photograph is not requested. Both themes
use the approved artificially lit scene, with theme-aware opaque text plates.
No animation, image filter or reference-photo-derived candidate is installed.

The PNG master, optimized WebP, prompt summary and hashes live in
`assets/branding/forgejo/login-station/`. Only the WebP enters the native payload.
Delivery changed this new asset, `login.css` and its `v=17-station` header link on
the existing canonical local mounts. Native authentication markup and JavaScript
were preserved. Previous CSS/header are retained under
`.artifacts/forgejo-login-station/before-delivery/`.

All 14 live theme/viewport cases pass at 320, 640, 960, 1199, 1200, 1440 and 1920px:
left-half geometry, no page overflow, desktop-only download, readable native
fields and unchanged form method/password type. Both desktop themes were reviewed
visually. Branding/inventory, test typecheck and preview build pass. Evidence is
in `.artifacts/forgejo-login-station/`; no credentials or form submission were used.

## Notifications completion pass

The prior broad pass missed the bell preview's older palette/typography and the
notification family's selected-tab treatment. This focused pass covers unread/read
inboxes, subscriptions, watched repositories, and the bell preview's populated,
pinned, loading, empty and failure/retry states. It uses the shared neutral palette,
inverse tabs/header, framed visible row controls, condensed headings, technical
metadata and a red View all notifications action. Single dividers separate rows
and controls. Narrow rows move timestamps/actions below the title through 1100px.

Only `notifications.css`, `notification-preview.css` and their versioned header
links were delivered to the canonical localhost:3300 mounts. Native templates,
HTMX lifecycle, notification IDs/status forms, destinations and runtime JS remain
unchanged. Prior files and screenshots are retained in
`.artifacts/forgejo-notification-refinement/`.

Checks pass: 40 native route/theme/viewport combinations at 320, 640, 800, 960 and
1440px using the existing screenshot session; populated/pinned row layout from
production-rendered template fixtures; preview containment/design at 14 width/theme
combinations, plus existing loading/error/retry/stale-response, keyboard/focus,
outside click, redirect and progressive fallback cases. The real fixture inbox
is empty; populated visual/lifecycle evidence is explicitly synthetic. No native
notification state was changed. Focused Go rendering, source branding/inventory,
test TypeScript and preview build pass. The opt-in populated fixture is generated
with `SODA_FORGEJO_NOTIFICATION_GALLERY=1 go test ./scripts -run '^TestForgejoNotificationPreview'`;
`SODA_FORGEJO_NOTIFICATION_REVIEW=1` enables its read-only live page checks.

## Scope completed

The full inventory of **253 Forgejo 15.0.7 overrides** uses the redesigned shared presentation owners. This includes authentication and onboarding, dashboard and exploration, public profiles and organizations, repositories and code views, issues and pull requests, milestones/projects/wiki/releases, Actions, packages, notifications, personal/repository/organization settings, moderation, administration and status pages. Most native templates inherit the design without unnecessary markup edits; explicit template edits remove obsolete presentation inputs and refine shared composition without replacing native control bodies.

- White/near-black surfaces, red filled actions, square controls and no decorative shadows. Barlow Condensed 800 headings, Barlow body text and IBM Plex Mono controls. Input/control borders and text roles meet the checked contrast thresholds.
- Symbol-only navigation and sign-in branding: red outer plate, opposite-color middle plate, transparent core. Explicit light/dark and automatic theme selection retain native ownership. Updated SVG staging and generated favicon, logo PNG and black Apple touch icon.
- Retired all robot/papercraft rendering and 24 image delivery entries. Retired source images and prompt sheets are now removed; their provenance remains in Git history. Avatars, organization logos, repository content and native functional icons remain.
- Text-only intros, restrained empty states, mobile-first creation/dashboard layouts, neutral information panels and readable selected buttons. Existing native status/diff/syntax colors remain separately owned.
- The shared semantic palette also supplies the existing Soda-owned page/island consumers and Cockpit. Their native layout and behavior were not reimplemented.
- Versioned entry styles and transitive font/palette imports; added the licensed Barlow Condensed font to the canonical payload. Native JavaScript graph revision and authorization/integration contracts are unchanged.

## Second-pass refinement

The second pass audits the full override inventory and shared stylesheet cascade, with targeted browser review of the affected components in both themes.

- Removed unused artwork arguments, variables and layout rules across page intros, migration, wiki and creation templates. Optional intro title IDs no longer produce empty attributes. Native permission gates and form bodies remain intact; the retired project-artwork-only gate is explicitly distinguished from the preserved edit/new title branch in the contract test.
- Text buttons keep a minimum 44px target and grow for wrapped labels. Settings/sidebar labels wrap, narrow repository settings headings sit above navigation, and joined/icon controls retain their established dimensions.
- Migration cards no longer reserve a vacant artwork column. Provider guidance remains visible on mobile, and Git-card hover text retains readable contrast. Mobile sign-in places the form directly after its introduction.
- Scoped public-profile and avatar-editor portrait rules to their owning pages. Fixed an empty-state SVG whose padding consumed its drawable area. Standardized technical typography, long runner values, heading wrapping and organization spacing; adjacent form dividers have a single owner.
- Fixed a delivery omission: `form-pages.css` was referenced by the header but absent from the canonical payload. The staged preview now includes it. A new manifest check verifies every header stylesheet, and browser checks reject missing resources.
- Added production-template migration and sign-in review pages plus a component reference for long labels, field validation, administration, creation forms, runner metadata and disclosures. Synthetic contexts and native-shaped inner controls are identified as fixtures.

## Brutalist personality pass

The shared presentation now uses uppercase condensed headings, inverse intro/sidebar labels, 1px framed red and neutral actions, inverse selected controls with red navigation markers, and unframed page introductions. Migration provider cards use 1px outlines and display typography. Text inputs and code/data canvases keep their quieter borders; no decorative imagery or shadows were added. The same semantic colors invert these treatments in dark mode.

Candidate build and five affected browser cases pass (component boundaries, refinement, gallery and settings with/without JavaScript). Mobile/desktop captures are in `.artifacts/forgejo-refinement/captures/personality-*`; logs use the `personality-` prefix. This remains isolated preview evidence.

## Verification

- `go test ./scripts -count=1`: native template parse, original-body recovery, permission/form/context contracts and brand PNG checks pass. Pinned upstream/native contract hashes were preserved; only reviewed presentation deltas and local inventory hashes changed.
- `bun test tests/forgejo/*.test.ts tests/forgejo/presentation/*.test.ts`: 35 pass, 28 opt-in browser/native-fixture cases skipped. Inventory covers all 253 overrides. New checks cover retired artwork, symbol geometry/staging and theme text/control contrast.
- Seven opt-in browser cases pass with the complete candidate CSS, native Forgejo CSS and rendered production partials: component boundaries, work-item lists, milestones, gallery, repository settings, JavaScript-disabled settings navigation and the new refinement review. Gallery tested at 320/390/900/1440px in both themes; settings at ten widths from 320–1440px. The new refinement review exercises migration, sign-in and shared components at 320/390/768/1440px, including resource delivery, wrapping, focus, contrast and page-specific avatar sizes. Explicit modes also tested against the opposite system preference; automatic switching selects the matching surface and logo.
- Full TypeScript/Lit checks pass; final changed-test typecheck passes. `build:forgejo` and `build:preview` pass. PNG regeneration comparison passes.
- Native-build source tests ran; two tests rejected macOS's symlinked temporary-directory ancestry. Both pass unchanged with `TMPDIR=/private/tmp`; the remaining native-build cases passed in the initial run. No native Linux build or installation is claimed.
- Second-pass evidence: `.artifacts/forgejo-refinement/` contains the initial source snapshot, final Go/payload/source/browser/typecheck logs and final desktop/mobile captures. `git diff --check` passes. Focused canonical-payload native-build checks pass with `TMPDIR=/private/tmp`.
- First-pass evidence: `.artifacts/forgejo-redesign/` holds logs, review scope, the incoming-template snapshot and desktop/mobile captures. The isolated component review server is at `http://localhost:8140/`; `/gallery-light.html`, `/gallery-dark.html`, `/repository-settings-light.html`, `/repository-settings-dark.html`, `/home-light.html` and `/home-dark.html` expose the candidate assets. Second-pass pages are `/refinement-light.html`, `/refinement-dark.html`, `/migration-light.html`, `/migration-dark.html`, `/signin-light.html` and `/signin-dark.html`.

## Website parity and local delivery

The website and Forgejo now share their canonical logo geometry, light/dark palette,
font families, inverse labels, uppercase actions and red selected-state
markers. The standalone Forgejo front page now uses the shared action owner too.
The website retains photography/marketing scale; handbook content retains its
sentence-case reading layout. Run the website's `bun run check:brand-parity
../sodaos` to compare the actual sources (no duplicate manifest).

On 12 September the owner selected **localhost:3300** for delivery. The Mac
container `sodaos-local-forgejo` was updated using its existing observed
`sodaos-local-forgejo_data` volume, distinct from the builder fixture mentioned in
the shared handoff. Its unchanged runtime JavaScript and original data remain.
A stopped, consistent copy of `/data/gitea` and `/data/git` is retained privately
under `.artifacts/forgejo-delivery/2026-09-12-brutalist/private-backup/`. Existing
account/repository counts match. New read-only presentation mounts are defined by
that delivery directory's `compose.override.json`, layered on the original local
compose file. The local default/available theme configuration and application
name now match `appliance/config/forgejo.env` (Soda Auto and Soda OS), preserving
the native and accessibility theme alternatives. Screenshot capture follows the
selected theme family instead of replacing Soda with stock Forgejo colors. The canonical sodaos checkout was not edited.

Actual served CSS and 28 public route/theme/viewport combinations pass on the
updated local instance. Website build, Storybook build, typecheck, unit checks,
132 browser combinations and 163 internal links pass. Delivery logs and the
preservation receipt are in the same ignored delivery directory. Component and
repository-settings keyboard/no-JavaScript evidence remains in the refinement logs.

## Retired image cleanup

Removed 34 papercraft images, 29 obsolete prompt/review documents, the old artwork
preview and two unused Forgejo wordmark masters. All 11 legacy SVG entries are
removed from the Forgejo delivery manifest. Nine shared masters remain only for
the still-active Cockpit/installer consumers, outside Forgejo's payload.

The active localhost:3300 public tree was cleaned of its 24 stale papercraft PNGs
and 11 legacy SVGs. The generated preview also prunes unlisted top-level Soda
images on rebuild. No uploaded images, provider/interface icons, database or
private backup was removed. Historical source-contract fixtures intentionally
retain old filenames to verify native template preservation; these are not assets.

All 35 retired local image URLs return 404; the six current images match source
bytes and native image aliases/icons still load. Public native browser checks,
source contracts, Go checks, packaging checks and website parity pass.
See `.artifacts/forgejo-image-cleanup/` for exact removal lists and receipts.

## Initial framed public welcome page (superseded photography)

The initial framed front page placed its native header, welcome content and footer
inside one centered, opaque 2px frame over the website’s soda-bar photography.
Mobile keeps 4% gutters; tablet uses 7%, desktop at least 12%, with a 1280px
maximum panel. Short pages center vertically; taller content scrolls naturally.
The footer owns the single shared divider. Native content, registration gates,
routes and theme controls remain unchanged.

Six optimized WebPs are copied byte-for-byte from website commit `4ff11a5`:
mobile/tablet/desktop, each in day/night. The website retains the PNG masters and
prompts. The canonical payload stages them; only the selected background downloads
on the welcome page. Other Forgejo pages retain their existing layouts.

Installed on the existing localhost:3300 presentation mounts. Only `home.css`,
its versioned header link and the six backgrounds were delivered; previous CSS
and header are saved in `.artifacts/forgejo-home-background/before`. The local
container restarted to reload its template; data, credentials and configuration
were preserved.

The preview build, affected Go templates, branding/presentation inventory and test
type checks pass. Twelve browser combinations cover both themes at 320, 390, 768,
1024, 1440 and 2560px, including opposite system preferences, one background
request, an opaque centered frame, and a joined footer with no doubled divider.
All 28 existing live public route/theme/viewport checks pass; the live theme toggle
was also reviewed visually. An optional embedded-upstream inventory check remains
skipped because its local export is absent. Logs are in `/tmp/forgejo-home-*.log`.

## Remaining boundaries

The redesign is installed on the selected **local** Forgejo instance. No remote
Soda OS server was deployed. Public native pages and the selected component
compositions were checked; this is not a claim that every authenticated route was
individually exercised. No real provider/account actions were submitted. The website parity commit is
`822ce09`; both repositories retain their own normal commit history. Cockpit's
complete native screens were not captured in this Forgejo pass.

Fastfetch ASCII branding is updated to the same canonical symbol. The native
stage delivers a themed fastfetch preset and a separate plain-text MOTD; see
[terminal identity](../assets/branding/terminal/README.md). Local fastfetch render
and staging-source checks pass; no native appliance installation was performed.

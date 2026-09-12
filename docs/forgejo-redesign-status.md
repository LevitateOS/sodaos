# Forgejo redesign status

Updated 12 September 2026 in `/Users/vince/Projects/sodaos-forgejo-redesign`, branch `codex/forgejo-redesign`.

## Current workspace

The owner moved ongoing work to the canonical `/Users/vince/Projects/sodaos`
checkout on `main`. The deleted worktree is no longer the development workspace.
The committed redesign through `bcf0fb1` is merged here, including the framed
welcome page. The six uninstalled subway concepts are preserved under
`.artifacts/subway-backgrounds/`; they remain previews. The running local
container still uses surviving presentation mounts under the former directory;
moving those runtime mounts is separate from this source recovery.

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

The shared presentation now uses uppercase condensed headings, inverse intro/sidebar labels, 2px framed red and neutral actions, inverse selected controls with red navigation markers, and 3px structural rules at page introductions/settings bars. Migration provider cards use firmer outlines and display typography. Text inputs and code/data canvases keep their quieter borders; no decorative imagery or shadows were added. The same semantic colors invert these treatments in dark mode.

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
font families, inverse labels, 2px framed uppercase actions and red selected-state
markers. The standalone Forgejo front page now uses the shared action owner too.
The website retains photography/marketing scale; handbook content retains its
sentence-case reading layout. Run the website's `bun run check:brand-parity
../sodaos-forgejo-redesign` to compare the actual sources (no duplicate manifest).

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

## Framed public welcome page

The public front page now places its native header, welcome content and footer
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

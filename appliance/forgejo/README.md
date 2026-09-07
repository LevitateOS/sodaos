# Soda Forgejo login override

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

The stylesheet uses `AssetUrlPrefix` through the template and relative CSS imports.
Login uses the shared light/dark palette; other pages retain their native
theme. Below 900px the illustration is hidden to prioritize signing in. The English
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

## Guest theme preference

The top-right icon button changes login appearance only. Initially follow the
system color preference. An explicit light/dark choice is stored under
`soda.login.theme:<AppSubUrl or />` in this origin's localStorage. Other tabs sync
through storage events. Clearing the value restores system following; invalid
values are ignored. Blocked storage still permits toggling for the current page.
The head script applies the choice before login content paints. Without JavaScript,
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
The stock head/footer are retained; the public page has its own visible header.
The authenticated dashboard template is not overridden.

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
feed card, sidebar and discovery link use `dashboard.css`; shared navbar/footer
rules in `explore.css` also cover `.soda-dashboard`. Native account themes remain
authoritative. The appearance shortcut uses the upstream `PageIsNews` flag.
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
upstream-owned. The shared template keeps Pull requests outside the new styles.
`issues.css` reuses the dashboard shell and context switcher; account theme state
remains native. The two custom intro strings remain English. Artwork provenance
and exact prompt are in `assets/branding/forgejo/issues-art-prompt.md`.

Local template reload and Chrome checks covered populated dark-mode rows,
created-by/in-your-repositories switching (2/6 open issues), closed empty results,
keyword no-match, oldest sorting, and 390px populated/empty layouts without
horizontal overflow. Pull requests still renders its native list without the
Issues styling. Light-mode visual inspection, organization context and pagination
were not newly exercised; this fixture list has fewer than one page of issues.

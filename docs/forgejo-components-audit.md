# Forgejo presentation component audit

The expanded system had real ownership overlaps and CSS regressions. The small
presentation vocabulary is still appropriate; the page expansion needed tighter
boundaries around it. This audit fixes those boundaries without adding Lit, a
form schema, a generic card renderer or a new component framework.

Reviewed from the first audit at `a81bfae` through the expanded baseline `82379b4`:
201 template overrides/helpers, all 38 linked stylesheets, guest-theme behavior,
native callers and the focused compatibility tests against stock Forgejo
**15.0.7**. Repository, settings/admin/org, and authentication/content families were
reviewed independently, followed by a shared-cascade review. After cleanup there
are 201 template files, four Soda partials and 45 explicitly registered CSS files.
The extra CSS files separate existing feature owners; they are not 45 new
primitives. Every CSS file has exactly one registry entry and every entry exists.

These counts describe audit candidate `919bb97`. Canonical main also retains the
newer notification preview: combined source has 203 template files, five Soda
partials and 46 registered CSS files. The [merge handoff](implementation-history.md#expanded-component-audit-merged-into-canonical-main)
records merged-tree tests and explicitly separates them from template activation.

Backend authority, canonical artwork, Cockpit's separate frontend and the
unimplemented Sodaspaces drawer are outside this presentation audit. Native
account-theme adapters and their separate staging path remain intentionally
separate from these preview overrides.

## Findings and corrections

| Finding | Correction |
| --- | --- |
| Shared input type exclusions had higher specificity than focus/error selectors. Native invalid inputs, textareas and dropdowns lost their error borders. | Use `:where()` for the shared control exclusions. Explicit native field/control error states win, including while focused. Browser tests reproduced the old failure and pass in both native light/dark themes. |
| Primary actions inside forms/repositories were blue, while the same native actions in settings headers remained orange. Several page files repeated button colors. | Bind the four native primary-action color variables on the button elements in the shell. Native sizing, basic/semantic colors, loading and disabled behavior remain intact; remove duplicate form/repository/profile/status color rules. |
| Shared settings cards applied panel padding to actual `<table class="ui attached segment">` elements, including administrator notices and repository LFS. | Split surface treatment from padding and exclude tables from panel padding. The native-table regression test observed 22px before the fix versus the native 14px. |
| The repository shell capped every direct container at 1120px, overriding native fluid code/diff/blame canvases and the intended 1440px pull-files layout. | The common width applies only to non-fluid containers. Organization containers follow the same boundary. Browser tests cover ordinary, fluid and pull-files widths and narrow-page overflow. |
| `repository.css` still owned old code, issue, release, wiki, project, Actions and settings leaf rules. Several overruled the newer family files, including code-summary spacing and issue-row typography. | Keep repository identity/navigation, ordinary width and bounded native action shape in the shell. Move actual leaves to their existing positive family markers; remove superseded declarations and duplicate container decisions. |
| Project lists and boards were spread across package, profile and repository files. Repository project cards combined grid gap with a second row margin; creation could acquire an extra outer card around its fieldset. | `projects.css` owns shared project lists/forms/boards across repository, user and organization callers. One row-spacing decision and the existing semantic form section remain. |
| `packages.css` also owned owner/global code search. Package cleanup put overflow on a wrapper whose native `.ui.table` display prevented a usable scroll box. | Separate `code-search.css`; give the package preview wrapper a bounded block scroll box while retaining its native table contents. |
| `admin-org.css` mixed two unrelated shells, monitoring output and organization home/team content. Broad organization rules also competed with newer leaf rules. | Separate `admin.css` and `organization.css`. Monitoring, home, team/member lists and README content remain in their named detail files. |
| `workflow-details.css` grouped webhook providers, secrets, variables, blocked users, quota statistics and abuse reporting. It also repeated card and focus decisions. | Delete the aggregator. Webhooks, configuration lists, quota disclosure and moderation have explicit owners. Shared settings CSS owns attached-card appearance and action-heading composition; shell CSS owns focus. |
| The appearance theme and OAuth application creation forms are nested outside the principal-settings-form adapter. Their controls differed from adjacent principal forms. | Add `.soda-form` to those two native forms. Keep the structural adapter narrow so search, row-action and dialog forms retain native sizing. |
| Repository-context error pages let the native navigation expand the implicit grid track. The 390px regression measured a 711px page. | Give the status grid a `minmax(0, 1fr)` track so native navigation can fit and overflow into its own menu. Plain and repository-context errors retain one status-card owner. |
| The same native profile card looked different on profile and package pages because its styles depended on the profile page class. | Give its existing profile/code/packages/projects callers an explicit `.soda-profile-card-context` marker. One profile-card style owner applies in each context without borrowing a page class. |
| The migrating-page wrapper hard-coded signed-in state even though the selected upstream route can serve public migrating repositories anonymously. | Derive `data-signed` from native `.IsSigned`, preserving account versus guest appearance authority. |
| The copied `finalize_openid` override has no caller in the selected upstream source. | Remove the unused override and its hash-test entry. Reachable OpenID/link-account wrappers retain their native flags, security fields and local theme-toggle placement. |

The prior audit's corrections remain intact: small intro/empty/toggle partials,
native Explore navigation, one list wrapper contract, dialog-safe search selectors,
theme-aware action colors, one guest palette owner, and native login loader/error
behavior. The later Explore overflow-width and milestone flex/grid fixes are
preserved and remain covered by their browser regressions.

## Ownership after the audit

The [composition contract](../appliance/forgejo/README.md#presentation-component-contract)
defines the exact inputs. The shared vocabulary remains small:

| Primitive or adapter | Responsibility and limit |
| --- | --- |
| Page shell | Semantic tokens, fonts, navbar/footer, content width, native action colors and focus. `.soda-page` and `.soda-page-marker` are full-page opt-ins, forbidden as drawer/widget roots. |
| Page intro | Fixed escaped heading/copy/artwork inputs. No route selection, permissions or arbitrary HTML slots. |
| Toolbar/tabs | Layout plus bounded native search, menu and context-switcher adapters. Native scripts own interaction and visibility. |
| Form section/controls | Semantic fieldsets, labels/help, control sizes and states within explicit principal forms. Native fields, validation, POSTs and dialogs remain upstream-owned. |
| Settings frame | Native side navigation and attached-card appearance across four settings families. Does not own tables' padding, feature lists or quota behavior. |
| List/empty content | Direct native row-list composition, metadata, pagination and empty presentation. Native callers determine emptiness and permitted actions. |
| Guest theme | Palette, toggle and a shared anonymous route/placement gate. Signed-in appearance remains Forgejo's account preference. |

Feature-specific structures stay explicit: code/diff canvases, native project
columns, release content, webhook event fieldsets, quota disclosures and moderation
rows are not variants of a universal card. Home's marketing composition, the main
sign-in story/form layout, secondary auth and federated consent/linking layouts
also retain distinct wrappers while sharing tokens and applicable controls.

All scoped CSS remains in one visible header registry. This has a loading cost;
conditional bundles or a second route-to-asset dispatch table are not needed for
the current boundary fixes. An upstream upgrade still requires exact-version
review of selectors, templates and script integration. CSS scope and hash tests
cannot establish compatibility with a different Forgejo release.

## Validation and limits

The [historical audit record](implementation-history.md#expanded-component-audit) records
executed tests and captures. New browser regressions load stock native CSS plus
the complete candidate cascade around small native markup contracts. They test
presentation collisions, not server permission enforcement or successful POSTs.
Focused Go tests check native composition, restored stock hashes and retained
security/interaction hooks; native seams are stubbed where appropriate.

The local preview binds the main checkout, not this isolated worktree. Candidate
screenshots use `scripts/screenshot.ts --local-css` and the existing authorized
non-admin fixture profile. Only CSS is substituted in that browser; server HTML
and scripts remain native and unchanged. The form/profile marker additions,
migrating guest flag and removed unused override are source-checked, not newly
rendered server changes. Missing quota/Actions routes produced native error pages;
disabled global code search redirected to Explore repositories. These are not
quota, Actions or code-search workflow evidence.
Administrator, owner-only settings, populated project boards, package cleanup and
external authentication/provider workflows were not newly exercised end to end.

No fixture records, account preferences, credentials, dependencies, services,
provider resources, production stage or installed appliance were changed. This is
bounded source and local presentation validation, not release acceptance.

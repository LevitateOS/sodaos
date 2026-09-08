# Soda–Forgejo local presentation review

Candidate **2026-09-08.3**, Forgejo **15.0.7**. Local preview only; full visual
acceptance remains open until the verification gaps below are resolved.

## Review entry points

- [Working repository preview](http://localhost:3300/bob/activity-field-notes)
- [Grouped screenshots and evidence](../.artifacts/forgejo-presentation/review.html)
- [Light reference gallery](../.artifacts/forgejo-presentation/gallery-light.html)
- [Dark reference gallery](../.artifacts/forgejo-presentation/gallery-dark.html)
- [Native capture manifest](../.artifacts/forgejo-presentation/review.json)
- [206-template coverage inventory](../tests/forgejo/presentation/inventory.json)
- [Shared component contract](../appliance/forgejo/README.md#presentation-contract)

Generated galleries, captures and evidence remain ignored local artifacts. They
are not served by the appliance or shipped as mock Forgejo pages.

## Implementation and evidence

The shared system owns spacing, fonts, controls, open sections and editor widths.
Repository, account, organization, package, administrator and focused-task styles
use that system while retaining native editor, table, menu and polling structures.
125 templates now select explicit presentation roles; all 206 current overrides
and helpers are accounted for, including unchanged specialized/native callers.
No backend, dependency, authentication or permission changes were made.

Focused Go source checks pass. The 30 Node checks pass with the local opt-in,
including the repaired open-settings baseline, native-table isolation, form
focus/errors, primary anchor contrast, native semantic/disabled/loading button
states, joined controls, native Explore overflow/navigation, notification
lifecycle, milestone layout and inventory coverage. Gallery compositions pass
light/dark at 1440, 900, 390 and 320px; native Explore also covers intermediate
widths and 320px. Component fixtures are not native-route evidence.

There are **36 accepted native captures**: dashboard, Explore repositories,
repository overview, issues, milestone detail, account profile settings, new
issue, contributor profile and sign-in; each has light/dark desktop/mobile views.
The capture helper rejects unexpected HTTP status, redirects, status pages,
missing landmarks, stale revision/stylesheets and browser errors. JSON sidecars
record what was actually checked. A deliberately signed-out settings capture was
rejected, proving that a login page cannot silently count as settings coverage.

These are viewport captures. They do not prove every offscreen field, scroll
state, permission branch or interaction. Captures use existing sessions and
browser-only theme selection; no account preference was saved.

## Required verification still outstanding

| Family/state | Missing prerequisite or coverage |
| --- | --- |
| Repository editors/settings | Existing authorized owner access for milestone, release, project, wiki and file creation/editing; protected settings and archived/destructive variants |
| Organizations | Existing organization/team/invitation fixture and appropriate membership; public Explore currently has no organization fixture |
| Administration/operations | Existing administrator session for inventories, queues, runners, providers, configuration, monitoring and quota |
| Setup/authentication | First-run setup and completion states; activation, recovery, MFA, consent and provider-specific states beyond the captured sign-in form |
| Packages and specialized workspaces | Populated package/cleanup, board, Actions/log, diff/editor and relevant permission/interaction states not exercised by these native captures |
| Form behavior | Native submission, validation responses, uploads and provider effects were not exercised; component states and preserved source hooks do not substitute for these journeys |

No fixtures/resources, permissions, destructive actions or appliance deployment
were used to manufacture coverage. These prerequisites remain explicit; the
local presentation candidate does not establish full-product or visual acceptance.

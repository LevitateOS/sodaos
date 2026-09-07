# Unified Soda dashboard inventory

**Official Forgejo template overrides are selected. Only bounded U08 is accepted
(1/20); U01's architecture acceptance is withdrawn.** Follow the
[leading plan](dashboard-implementation-plan.md), [workflow inventory](forgejo-api-coverage.md)
and [handoff](implementation-status.md). The former mandatory all-React/API and
source-built Forgejo sequence has been removed, not retained as a fallback.

## Interface and authority

Forgejo renders its own developer, administrator and authentication workflows.
Use supported extension templates, whole-template overrides only where necessary,
custom themes/assets and native configuration for coordinated Soda presentation.
Keep native handlers, scripts, forms, sessions, permissions and business rules.
No iframe, scraped fragments, new auth/grant authority or downstream executable.

Soda's existing React/TypeScript + PatternFly + Vite+ + Zustand frontend and Go API
remain for its environment/access extension. Do not remove working stock-API clients
or implemented screens without tracing their callers and proving replacements.
Exact shell/navigation, session/OAuth/origin integration and template compatibility
still need U01 review. A visually shared shell does not establish shared cookies.
No React SSR, production Node service, Tailwind, TanStack or new framework is selected.

| Area | Owner | Soda responsibility |
| --- | --- | --- |
| Identity, authentication, native account/security/roles | Forgejo | Preserve native flows and integrate Soda OAuth/session/navigation safely |
| Repositories, Git, collaboration, Actions, packages and administration | Forgejo | Customize native presentation; use stock APIs only for actual Soda callers |
| Persistent environments, associations and memberships | Soda with native Linux/Podman | Real provisioning and stable-ID/current-native-authority checks |
| Development-access keys and project accounts | Soda with OpenSSH/native account tools | Explicit join-time installation; no personal private-key upload |
| Host administration, Tailnet and local runners | Stock Cockpit/native services | Preserve separate operator boundary, backing logic and tests |

Forgejo site administrators, Soda operator, project administrators and host root
are not interchangeable. Native Git SSH/GPG keys and development-access keys remain
distinct. No provider-role inventory, password store, clone/index backend, scheduler
or generalized synchronization/recovery subsystem is introduced.

## Page families and implementation owners

The [single source register](forgejo-api-coverage.md) retains all 179 action groups
and conditional behaviors. Missing API coverage no longer means a missing frontend
feature: preserve/test the existing rendered workflow rather than implement an API.

| Family | Required native/customized outcome | Owner |
| --- | --- | --- |
| Login/onboarding/consent/recovery | Native password/MFA/WebAuthn/mail/IdP steps, real native eligibility and safe Soda return/session behavior | U04/U05/U16 |
| Profile/security/keys/applications | Native profile, visibility, email, credentials, OAuth apps/grants and tokens; separate Soda preferences/development keys | U05 |
| Repository discovery/create/content | Native repositories/templates/README/tree/raw/archive/LFS, with legitimate environment selection/entry | U06 |
| Code/ref/history/copy | Native blame, comparison, commits, refs, edits, forks, imports/tasks and normal Git | U09 |
| Issues/boards | Native templates, comments/assets/history, labels/milestones, time/dependencies and personal/org/repository boards | U10 |
| Pull requests | Native conversation/reviews/ranges/resolve/viewed/checks/protection/merge and scheduling | U11 |
| Repository/org settings | Native access/protection/teams/invitations/hooks/clients/units and ownership/lifecycle actions | U12 |
| Work/search/activity | Native notification inbox, profiles, search, feeds and graphs | U13 |
| Actions | Native workflow/jobs/logs/artifacts/trust/run controls and provider configuration; local capacity stays Cockpit | U14 |
| Releases/wiki/packages | Native publishing/assets/history/settings and existing package protocols | U15 |
| Site administration | Native People/MFA reset/auth sources/configuration/maintenance/quota/moderation/runner operations | U16 |
| Complete navigation/help/errors | Shared presentation, safe links, native-enabled/disabled behavior and real full-product rehearsal | U02/U17/U19 |

Customization must not break native scripts/forms, omit conditional features or
turn cosmetic controls into authority. Use native operation results and native
security checks; no copied policy or invented stronger mutation guarantees.

## Soda environments

Preserve repository association, one eligible environment, real create/explicit
join/account-key provisioning, own-login/current-IP/public-host-key connection
information, actual membership and honest partial/stopped/unavailable states.
Developers connect directly as `user@project-ip` using ordinary SSH/SCP/SFTP/Git.
The existing human-owned creation subset does not authorize stale authority after
native transfers. Preserve Linux state while checking current native authority.

U07 owns the requested browser terminal into the current user's **existing**
project-local account/home. Review transport, native target binding, CSRF/origin,
PTY/encoding/backpressure, session lifetime/disconnect and secret/audit behavior
with U02/U03/U04. No implicit create/join/start, host-root/shared-admin shell or
browser-selected container/UID/host flags. Missing membership keeps explicit join;
ordinary direct SSH remains supported. U08's earlier scope is unchanged.

No managed private tool/service branches, IDE/catalog, environment deletion/rebuild,
updater, backup platform or general recovery UI is added. E01–E03 remain unselected.

## Dependencies and packaging

Actual pins belong in `dashboard/package.json`, `dashboard/pnpm-lock.yaml`, Cockpit's
manifest/lock, `go.mod`/`go.sum` and appliance/project recipes. Keep them unchanged
unless a concrete supported caller needs an approved change.

- Preserve existing React/router/Markdown/Zustand/PatternFly dependencies and
  Vite+/TypeScript/browser-test tools for Soda's real frontend.
- Keep Go/net/http, SQLite and current native integration; no new auth library or
  custom Forgejo API protocol is required by the template approach.
- Use the stock Forgejo image and supported configuration/custom assets through
  current build/stage/install callers; do not compile Forgejo or add a patch lock.
- Keep Cockpit's frontend/native bridge separate, with both Tailnet/Runners pages,
  dependency notices and focused tests. Preserve canonical assets and project CLIs.
- Deliver actual license/font/source notices for shipped material; see
  [licensing notes](licensing.md). A package's MIT metadata is not blanket font or
  transitive compliance. Do not remove notices just because the architecture changed.

## Existing evidence and remaining work

Recorded affected components are installed at `8b823db`; React preview is `/app/`
and default Soda routes remain HTMX. Forgejo remains stock 15.0.7. Four environments
and all existing login/data/evidence stay untouched by source cleanup.

The three inspected authority defects in legacy admin-token substitution, native
account creation rules and cached environment ownership remain source work. Native
replacement pages can retire obsolete callers after proof, not silently excuse
active vulnerabilities or remove legitimate environment checks.

U01 still needs the concrete supported template/integration review. U02 packages
customization, U03/U04 integrate secure native/Soda behavior, feature owners verify
native page compatibility, U17 rehearses complete coverage, U18 performs separately
approved cutover, and U20 proves both native architectures and retained services.
No downstream fork is an alternative implementation lane.

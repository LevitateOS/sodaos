# Unified Soda dashboard plan

## Status and decisions

Planning inventory following the user's React/dashboard direction. This is not an implemented migration, a frozen dependency lockfile or authorization to change the running appliance. The current dashboard remains Go + HTMX; the earlier architecture/milestone descriptions of that frontend describe the existing implementation. The page scope below is proposed for review. The [multi-milestone implementation plan](dashboard-implementation-plan.md) assigns the work, API/data/build boundaries, tests and acceptance criteria; it also separates conditional environment extensions from the core migration.

Selected constraints:

- One Soda dashboard is the custom frontend for upstream Forgejo plus Soda's development-environment extension, including both developer and administrator views.
- TypeScript + client-rendered React, built with Vite+; no SSR or React server.
- PatternFly components, layout and design tokens, with small Soda CSS overrides. No Tailwind.
- Zustand for application state and request state; ordinary browser `fetch`. No TanStack packages.
- Go is the browser-facing API, a bounded adapter to Forgejo, and the backend for Soda's environment/access extension—not a replacement forge backend.
- Forgejo is upstream and retains its existing business rules, data and administration. Soda owns only its additional development-environment responsibilities and the required integration/session state.
- First make complete workflows work; then improve presentation, performance and conveniences. Authorization, data preservation, error handling and basic accessibility belong in the working version.

```text
Browser: React + PatternFly + Zustand
                 |
          same-origin Go API
            /           \
       Forgejo       Soda database
                     + fixed-operation native helper
                     + persistent project environments
```

Go serves the production frontend assets behind the existing Caddy proxy. Node and pnpm are build/development tools, not new appliance services. Keep Cockpit as a separate operator application; do not import its privileged native bridge into the dashboard.

## Upstream ownership and extension boundary

**Soda extends Forgejo; it does not absorb Forgejo's backend responsibilities. Owning a frontend screen does not transfer ownership of the underlying feature.**

| Area | Owner | Soda's role |
| --- | --- | --- |
| Human identity, account profile, passwords, MFA, organizations, teams and native roles | Forgejo | Authenticate through Forgejo and present its supported user/admin operations |
| Repositories, Git permissions, issues, reviews, releases, packages and CI scheduling/results | Forgejo | Present the upstream capability through a bounded API adapter, without recreating its rules |
| Forgejo site administration | Forgejo | Build administrator views over supported upstream interfaces; Forgejo must authorize the acting administrator |
| Persistent isolated development environments and their Podman resources | Soda integration, using native Linux/Podman mechanisms | Provision, inspect and preserve environments; associate them with upstream repositories |
| Project-local accounts, development-access public keys and join-time SSH setup | Soda integration, using native account tools/OpenSSH | Turn a Forgejo-authenticated person's join into usable development access |
| Browser/API integration | Soda | Serve React, maintain its secure session, protect requests and compose environment details with authorized upstream responses |

Do not add a shadow Forgejo user/repository inventory, copied permission hierarchy, replacement collaboration engine or direct access to Forgejo's database/storage. Soda's database holds extension-specific relationships and settings, development-access data and protected session/integration state. Existing Soda profile fields are not justification for taking ownership of Forgejo account fields.

Forgejo Git SSH keys remain Forgejo-owned. Soda manages keys for development-environment access and their native installation; that does not imply replacing Forgejo's key registry or automatically synchronizing later key changes across environments.

Use supported upstream interfaces and preserve upstream authorization and operation results. If an interface is missing, investigate the upstream capability, retain a native fallback or propose an upstream improvement; do not silently implement a competing subsystem or fork the Forgejo backend to manufacture frontend parity.

The unified frontend includes three distinct administrative contexts: **Forgejo administration**, **Soda environment/access administration**, and links to **native host administration in Cockpit**. They are not a new shared superuser role. Keeping advanced Forgejo screens native while APIs are investigated is an interim coverage decision, not a decision to exclude administration from the Soda frontend.

## Authentication and authorization

**One Forgejo login can authenticate both Forgejo and Soda.** Reuse OAuth authorization code + S256 PKCE, not shared browser cookies or a new Soda password form.

1. The browser enters Soda's Go-owned login route and is redirected to Forgejo.
2. Forgejo owns password entry, first-login password changes, MFA and consent. If its browser session and consent are still valid, another password prompt normally is not needed; this is not a promise of a redirect-free login.
3. Go exchanges the authorization code and resolves the stable Forgejo user ID.
4. Go creates a secure, HttpOnly Soda session linked to that identity.
5. React calls the same-origin Go API using that session. Go uses the corresponding user's authorized Forgejo access for repository/collaboration requests and applies Soda's own rules to environment operations.

### Existing implementation versus required work

`internal/web/auth.go` currently requests `read:user`, looks up the user and establishes the Soda session. `internal/forgejo/client.go` currently returns only the access token from the code exchange, and it is not retained for later user API calls. Repository listing/lookup currently uses the restricted operator credential with separate ownership filtering. That is not the general authorization model for the expanded frontend.

The migration needs:

- Scopes matched to the actual selected Forgejo operations, without asking every developer for administrative access.
- Protected server-side access/refresh-token storage and expiry/refresh handling. Tokens must never be put in React props, Zustand, browser storage, URLs or diagnostic output.
- API session discovery, JSON error responses and clear unauthenticated/forbidden states rather than HTMX redirects or HTML returned to JSON callers.
- CSRF/origin protection for **all** state-changing API methods, including POST, PUT, PATCH and DELETE. The existing form-only POST guard cannot simply be reused unchanged.
- Clearing user-specific Zustand data on sign-out/account change; only non-sensitive UI preferences may be persisted locally.
- No automatic retry of a write after an ambiguous failure. Show native failures without duplicating repository or environment creation.
- No fallback from a denied/expired user credential to the operator token.

A Forgejo-authenticated user is not automatically the configured Soda operator, a project administrator or host root. Forgejo administrator views require the acting user's actual upstream authority and appropriate scopes, not a Soda-maintained copy of administrative roles or automatic use of the bootstrap credential. The current operator-gated People endpoint is existing bootstrap behavior, not the general authority model for the expanded Forgejo administrator UI.

The existing human-owner environment rule and explicit join remain in force. Joining an environment does not grant Forgejo repository permissions. Git SSH keys and Soda development-access keys remain distinct; this plan does not promise later cross-environment key/revocation synchronization.

Cockpit still uses native operator authentication, and project SSH still uses OpenSSH/public keys. Signing out of Soda ends its session; it must not be presented as guaranteed global Forgejo logout or termination of existing SSH sessions.

## Page inventory

This inventories intended page families and their tabs/forms, not one bespoke React component or route for every action. Reuse detail pages for create/edit modes and use dialogs where appropriate.

**Delivery:** **First** = first complete repository-to-environment workflow; **Next** = subsequent functional coverage; **Native/check** = link to native Forgejo/Cockpit until the specific interface and scope are established. Later functional coverage is not merely visual polish.

**Authority:** **F** = Forgejo; **S** = Soda; **F/S** = composed view that preserves both authorities.

### Application and account

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Sign-in entry / session expired | Begin Forgejo login, explain failure/expiry; no Soda password form | F/S | First |
| Projects | Repository discovery with environment status and authorized actions; filters/search/pagination | F/S | First |
| My work / overview | My repositories/environments, assigned issues and requested reviews | F/S | Next |
| Global search | Repositories, issues and pull requests within native visibility | F | Next |
| Notifications | Inbox, unread filters, mark read and links to subjects | F | Next |
| User profile | Native user identity and visible repositories/activity | F | Next |
| My profile / preferences | Forgejo identity, existing Soda profile fields, appearance/preferences | F/S | First |
| Development-access keys | Register/list public SSH keys used at Soda join time; explain installation scope | S | First |
| Git keys | Forgejo SSH keys; GPG-key management as the next extension | F | First / Next |
| Account security | Password, email/security verification, MFA/passkeys, recovery, authorized applications and personal API tokens | F | Native/check |
| About / help | Version information, connection/key guidance and native-tool links | S | First |
| Error states | Forbidden, not found, upstream unavailable, expired session, field validation, empty/loading/pending states | F/S | First |

### Repository and code

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Create repository | Name, description, visibility, initialization; no implicit environment provisioning | F | First |
| Import repository | Native supported migration/import choices and honest progress/failure | F | Next |
| Repository overview | README, metadata, clone URLs and environment entry point | F/S | First |
| File browser / file viewer | Tree, ref selection, file contents and raw/download links | F | First |
| File history / blame | File commits and line attribution | F | Next |
| File create/edit/upload | Native commit operation, target branch and validation; simple editor first | F | Next |
| Commit list | Ref-specific history with pagination | F | Next |
| Commit detail | Metadata, changed files and diff | F | Next |
| Branches and tags | Lists and supported native operations, respecting protections | F | Next |
| Compare | Base/head selection, commits and changed files | F | Next |
| Fork repository | Native fork form and result; no implied Soda environment copy | F | Next |

Repository identifiers remain Forgejo-owned. Do not mirror the repository inventory or permissions into Soda's database merely to serve these screens. A repository can exist without a Soda environment.

### Issues and pull requests

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Issue list | Search, filters, labels, milestones, assignees and pagination | F | Next |
| New issue | Title/body, metadata and supported attachments/templates | F | Next |
| Issue detail | Conversation, edit, close/reopen, assignments, labels and milestones | F | Next |
| Labels | List/create/edit and native permission checks | F | Next |
| Milestones | List/detail/create/edit and associated issues/PRs | F | Next |
| Pull request list | Filters, review state and pagination | F | Next |
| New pull request | Base/head comparison and submission | F | Next |
| Pull request detail | Conversation, commits, changed files and checks tabs | F | Next |
| Review and merge | Inline comments, approve/request changes, merge controls and native rejection/conflict results; part of PR detail | F | Next |
| Issue boards | Forgejo's issue-project/board UI, distinctly named from Soda environments | F | Native/check |

Start with standard Markdown and safe rendering; do not silently claim exact parity with every native template, mention, attachment or Markdown extension. Inventory those details against the pinned provider when implementing each feature.

### Automation and other collaboration

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Actions overview | Workflow/run lists and status | F | Next |
| Run detail | Run/jobs/check information exposed by supported interfaces | F | Next |
| Job logs / artifacts / run controls | Log streaming, downloads, cancellation and reruns require separate capability verification; native links meanwhile | F | Native/check |
| Releases | List and release detail/downloads | F | Next |
| Create/edit release | Tags, notes and release assets | F | Next |
| Wiki | Index, page, editor and revision history | F | Next |
| Packages | Owner/package list, version detail and files/install guidance | F | Next |
| Advanced activity/graphs | Native views not covered by the initial code/commit screens | F | Native/check |

Forgejo still owns CI scheduling and results. These are provider views, not a Soda CI engine. Host runner registration/capacity remains in Cockpit.

### Repository and organization administration

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Repository general settings | Description, visibility, default branch and enabled native features | F | Next |
| Repository access | Collaborators and native permissions | F | Next |
| Branch/tag protection | Native protection rules and validation | F | Next |
| Deploy keys | Repository-scoped native keys | F | Next |
| Webhooks | Configuration and supported delivery/test views | F | Next |
| Actions configuration | Native secrets, variables and relevant settings; do not expose stored secret values | F | Next |
| Organization directory/create/profile | Native organization discovery, creation and overview | F | Next |
| Organization members / teams | Lists, team detail, membership and native repository access | F | Next |
| Organization settings | Profile, native hooks and Actions settings | F | Next |
| Destructive/ownership settings | Repository or organization deletion, transfer, rename and archive workflows with linked environments need an explicit scope decision | F/S | Native/check |

Displaying native organization/team functionality does not implement an organization-to-Linux-project-administrator mapping. The existing ordinary human-owner environment rule remains the first-version rule. Do not introduce copied roles, membership synchronization or automatic environment deletion.

### Forgejo site administration

Administrator functionality is part of the unified frontend, not just developer repository screens. The adapter delegates upstream-owned operations rather than implementing a second administration backend. The exact controls on each page still need a capability/authorization audit against the selected Forgejo version.

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Administration overview | Available upstream admin functions and reported instance information, alongside separately labeled Soda environment information | F/S | Next |
| People | Authoritative Forgejo user inventory with Soda environment associations where needed; not the local Soda profile table presented as all users | F/S | First |
| Add/edit person | Forgejo account creation and supported account fields/status; account edits do not automatically rename Linux users or revoke existing SSH sessions | F | First / Next |
| Organization/repository administration | Upstream instance-wide administrative views and supported operations, supplementing the scoped settings above | F | Next |
| Instance webhooks | Native hook configuration and supported inspection/test operations | F | Next |
| Provider runner administration | Forgejo registration/runner/job views; local service execution/capacity stays in Cockpit | F | Next |
| Native quota administration | Forgejo's own quota rules/groups where supported; not a new Soda environment quota system | F | Next |
| Authentication/configuration/maintenance | Upstream settings, authentication configuration and supported maintenance tasks; API gaps use native views meanwhile | F | Native/check |
| Destructive/account-remapping actions | Upstream deletion/rename operations with Soda associations need an explicit lifecycle decision; no implicit environment deprovisioning | F/S | Native/check |

The existing scoped operator bootstrap remains the first-install path. Adding administrator screens neither exposes an unfinished installer nor grants arbitrary host commands. Source/schema coverage alone does not authorize executing live account, runner or maintenance operations.

### Soda environments and operator pages

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Create environment | Pick an eligible repository, show owner rule, invoke real provisioning and report its result | F/S | First |
| Environment overview | Actual native state, environment identity, associated repository and connection readiness | F/S | First |
| Members / join | Existing membership, explicit Add me, missing-key guidance and real account provisioning | S | First |
| Connect | Observed project IP, own login, SSH/SCP/SFTP/editor guidance and route/host-key caveats; may be a detail tab | S | First |
| Incomplete/unavailable environment | Honest native failure and operator guidance; not a destructive recreate button | S | First |
| Environment/access administration | Soda-specific environment status, association, membership and development-access views/actions, with the existing project/operator boundaries | S | First |
| Operator tools entry | Role-appropriate administration navigation and native fallback links; links do not grant access | F/S | First |
| Host administration | Stock Cockpit services/logs/network/storage and retained Tailnet/Runners pages | Native host | Keep native |

No first-version web terminal, IDE, tool/service catalog, managed toolchain branches, environment deletion/rebuild, updater, backup platform or generalized recovery UI is added by this inventory. Normal native development tools remain the workflow inside projects. Native operator setup remains the installer/bootstrap path, not a new publicly exposed setup wizard.

### API coverage evidence and limits

Reviewed the locally saved schema obtained from installed Forgejo **15.0.7**, `.artifacts/downloads/forgejo-swagger.json`, plus the existing Go authentication/provider source. This is source/schema inspection, not execution of all the proposed operations.

The schema contains repository/content/commit/branch, issues/PR/review, release/wiki, notifications, organization/team, package, user-key/settings and selected Actions endpoints. Its `/admin` surface also includes users, organizations, emails, hooks, runners/jobs, cron tasks, unadopted repositories and native quotas. This does not establish complete site-configuration, authentication-source, issue-board, Actions-log/artifact or account-security UI coverage. Absence there is not proof that no other native interface exists. Verify each feature against the pinned upstream before replacing its native page; do not scrape HTML, borrow operator cookies or invent unsupported API calls to fake parity.

## Direct dependency inventory

This is the proposed **direct** dependency list for the dashboard work, not an invented list of every transitive package or OS RPM. The real frontend lockfile and Go module metadata must record the resolved closure. New package versions are not yet selected or installed; verify compatibility/licensing and pin them before implementation builds.

Most of the requested stack already appears in `cockpit/package.json`. Reuse that baseline where appropriate without coupling dashboard code to Cockpit privileges or incidentally upgrading the existing operator pages.

### Browser dependencies

| Dependency | Purpose | Existing shared baseline / selection |
| --- | --- | --- |
| `react` | UI | 18.3.1 |
| `react-dom` | Browser rendering | 18.3.1 |
| `react-router-dom` | Client-only URL routing, not SSR/framework mode | New pin to verify |
| `zustand` | Feature stores, request status and UI state | 5.0.15 |
| `@patternfly/react-core` | Forms, layout, navigation, feedback and other UI components | 6.6.1 |
| `@patternfly/react-icons` | Icons | 6.6.1 |
| `@patternfly/react-table` | Repository/issue/member tables | 6.6.1 |
| `@patternfly/patternfly` | Base CSS, design tokens and assets | 6.6.1 |
| `react-markdown` | README/issues/wiki rendering without enabling raw HTML | New pin to verify |
| `remark-gfm` | Standard GitHub-flavored Markdown features | New pin to verify |
| `react-diff-view` | Commit/PR diff and review presentation; add when implementing that feature | Candidate pin/API to verify |

`fetch`, `AbortController`, `URL` and browser file/clipboard APIs need no package. Use explicit feature-owned Zustand stores and a small HTTP client. Handle cancellation/stale responses, pagination, errors and post-write reloads deliberately; do not build a generic query/cache framework inside Zustand. No TanStack Query or Router, Axios, Redux or second state system.

A syntax-highlighting package such as `refractor` is a **polish candidate**, not required for the first working code viewer. A plain text editor is sufficient initially; no Monaco/CodeMirror dependency or browser IDE is selected by this plan.

### Build, types and testing

| Dependency/tool | Purpose | Existing shared baseline / selection |
| --- | --- | --- |
| `vite-plus` | Dev server, production bundling, lint/format/test tooling | 0.3.0 |
| `@vitejs/plugin-react` | React development integration/Fast Refresh | New compatible pin to verify |
| `typescript` | Type checking | 7.0.2 |
| `@types/react` | React types | 18.3.13 |
| `@types/react-dom` | DOM renderer types | 18.3.1 |
| `@types/node` | Build/test Node APIs | 24.10.1 |
| `@testing-library/dom` | DOM queries/test support | 10.4.1 |
| `@testing-library/react` | Component tests | 16.3.3 |
| `@testing-library/user-event` | User interaction tests | 14.6.7 |
| `jsdom` | DOM test environment | 30.0.1 |
| `playwright` | Real browser checks, including actual Forgejo OAuth | 1.63.0 |
| Node | Build/development runtime only | 24.20.0 |
| pnpm | Package manager and resolved lockfile | 11.25.0 |

Use the test tooling exposed by Vite+ rather than adding a parallel Jest toolchain. Do not add standalone Vite, ESLint or Prettier merely to duplicate Vite+ responsibilities. No Sass dependency is needed just for dashboard CSS; existing Cockpit Sass/licensing/build dependencies remain owned by its build and are not removed by this plan.

### Go and appliance dependencies

| Dependency/tool | Purpose | Selection |
| --- | --- | --- |
| Go standard library | HTTP/JSON, static assets, cryptography, sessions and database interfaces | Existing Go 1.26.7 |
| `modernc.org/sqlite` | Existing Soda database | Existing v1.58.0 |
| `golang.org/x/crypto` | Existing public-key/crypto integration | Existing v0.55.0 |
| `golang.org/x/sys` | Existing Linux/native integration | Existing v0.47.0 |
| `github.com/stretchr/testify` | Existing Go test assertions | Existing v1.12.1; test-only use |
| `golang.org/x/oauth2` | Standard OAuth exchange/refresh support for the expanded user-token lifecycle | Proposed addition; compatible pin to verify |
| Forgejo | Identity, repositories and collaboration | Existing 15.0.7 service |
| Caddy | Existing HTTPS entry point | Keep current image/configuration baseline |
| Podman/systemd/native helper | Existing application and project runtime | Keep current host/source baseline |
| Cockpit, Tailscale and runner integrations | Existing operator functionality | Retained, not dashboard JS dependencies |

The immutable host, project OS, Git/OpenSSH/mise/Tea/GitHub CLI and native workload/provider dependencies remain governed by their current image recipes and lockfiles. This frontend migration does not add or upgrade those systems. No Node production service, additional database, Redis, GraphQL, Go web framework, ORM or new authentication provider is required.

## Implementation order and working criteria

This is a summary; follow [U01–U20 and conditional E01–E03](dashboard-implementation-plan.md) for the detailed sequence. All milestones remain unstarted until implemented; the plan itself is not runtime evidence.

1. **Foundation:** new React source/build entry, PatternFly shell, browser router, same-origin JSON contract and Go asset delivery. Preserve canonical branding. Do not remove the working HTMX routes before their replacements are connected.
2. **Authentication:** reuse Forgejo login, add the required per-user API credential lifecycle and test secure sessions, expired/denied access and CSRF on every write method. Preserve the separate operator/host boundary.
3. **First end-to-end slice:** sign in, create/list a real Forgejo repository, open its README/files, register the needed public keys, create its Soda environment, explicitly join and connect by ordinary SSH. Include operator People/onboarding. No fake success from seeded JSON or a database row alone.
4. **Functional expansion:** finish code history/comparison, issues and PRs, then the remaining mapped repository/organization/collaboration and Forgejo administrator pages in small complete workflows. Build views and supported API integration, not replacement upstream business logic. Keep native links wherever coverage is not yet verified.
5. **Make it good:** improve layout, responsiveness, keyboard efficiency, syntax/diff presentation, performance and request caching based on observed problems. Do not defer security, preservation of work or basic usable error/empty states to this stage.

Existing native gaps still matter: project subnet routing, actual two-user SSH/shared-tools/workload use, nested Podman and persistence have not been fully proved. A React rewrite does not close those gaps. Builds, browser/native testing and VM changes require the applicable explicit scope; this planning work runs none of them.

# Unified Soda dashboard plan

## Current decision — official template overrides

The user selected native Forgejo rendering through its **official template
extensions/overrides**, not a complete React/JSON replacement. See the
[leading decision record](dashboard-implementation-plan.md#selected-direction--official-template-overrides-partial-u01-invalidation).
U01's architecture acceptance is withdrawn; only bounded U08 is accepted (1/20).
Retain the inventory and useful source/license work. The prior all-React/no-SSR/
no-native-HTML constraints and mandatory headless patches below are superseded;
page ownership and implementation assignments still need reconciliation. This
records the choice, not implemented overrides or deployment.

## Historical status and decisions

This is the selected page-family/dependency inventory, not completion evidence or a frozen dependency lockfile. The React/API/schema-v3 preview is installed at `8b823db` on retained `soda-test`; default routes remain HTMX. U01 is accepted for planning/contract readiness and U08 for bounded native x86_64 first-product proof (2/20), not complete frontend/cutover acceptance. The [implementation plan](dashboard-implementation-plan.md) now incorporates H01's **179 action groups**, revising dependencies and feature ownership without restarting the milestones. The [single action register](forgejo-api-coverage.md) owns concrete interface/authority findings; the [handoff](implementation-status.md) owns actual source/build/installed evidence. Conditional environment extensions remain unselected.

**Coordination:** the [implementation plan](dashboard-implementation-plan.md#coordination-with-native-support-porting) leads core frontend/backend, production environments and U08/U20 product acceptance. The [native support plan](native-porting-plan.md) is subordinate and covers outside tooling/operator integrations, not a parallel product roadmap. This inventory defines page families, not another execution sequence.

Selected constraints:

- One Soda dashboard is the complete user-facing frontend for upstream Forgejo plus Soda's development-environment extension, including every developer and administrator workflow. **No Forgejo frontend escape hatch, even temporarily:** no native-page links, embedded/re-skinned upstream HTML, or missing-API scope waiver. Missing interfaces require integration work, not missing features.
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

The [headless architecture revision plan](forgejo-architecture-revision-plan.md)
now defines the proposed workflow-first integration and maintenance model for
these requirements. This document supplies page families; the [single action
register](forgejo-api-coverage.md) supplies action/contract evidence and the leading
U plan supplies implementation order. Do not create a parallel inventory or restart
the milestones.

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

Use supported upstream interfaces and preserve upstream authorization and operation results. If an interface is missing, investigate supported interfaces in newer releases and propose the concrete upstream API work required. A bounded Forgejo patch or other backend integration needs an explicit design/maintenance decision before implementation; it is not selected automatically. Do not implement a competing subsystem or expose Forgejo's frontend as a substitute.

The unified frontend includes three distinct administrative contexts: **Forgejo administration**, **Soda environment/access administration**, and links to **native host administration in Cockpit**. They are not a new shared superuser role. All Forgejo administrative screens belong in Soda. Only the separately selected host-operator Cockpit application remains native; that exception does not apply to Forgejo.

## Authentication and authorization

**Required experience:** sign-in, first-password change, MFA/recovery, consent and account security must also stay in Soda's interface, with Forgejo retaining identity/password authority. No second password store, borrowed cookies or weakened MFA is authorized. The existing redirect-based flow below is a working baseline, **not an exception** to the Soda-only requirement. U04/U05/U16 must review the native authentication/challenge design early, alongside the source-build/compatibility work—not after the other screens. U17 verifies complete coverage and U18 performs the preserved transition; do not break working login or merely hide the provider behind a proxy.

1. The browser enters Soda's Go-owned login route and is redirected to Forgejo.
2. Forgejo owns password entry, first-login password changes, MFA and consent. If its browser session and consent are still valid, another password prompt normally is not needed; this is not a promise of a redirect-free login.
3. Go exchanges the authorization code and resolves the stable Forgejo user ID.
4. Go creates a secure, HttpOnly Soda session linked to that identity.
5. React calls the same-origin Go API using that session. Go uses the corresponding user's authorized Forgejo access for repository/collaboration requests and applies Soda's own rules to environment operations.

### Existing implementation versus required work

The pre-React baseline requested `read:user` and discarded the identity token. The current grant implementation requests user/repository consent (administrator consent separately), verifies actual scopes through native introspection and encrypts per-session access/refresh grants. Its installed subset at `8b823db` and later local source checks are recorded in the handoff; neither is complete headless authentication proof. New repository/account/admin APIs use only acting-user grants. Retained legacy repository handlers still use the restricted operator credential; they are not a fallback for new APIs. Fix their active authority defects under U04/U06 before their later U18 removal.

The source supplies JSON session/logout, Soda-local preferences/keys, all-unsafe-method guards, protected grants and connected first-workflow React/API routes. The [H01 workflow audit](forgejo-api-coverage.md) now identifies source-backed auth/security/admin gaps, including native password/MFA API gates and WebAuthn origins. U01's shared-native challenge/session integration review is complete; U04 still must implement each native operation/DTO and prove its security behavior before exposure. The full migration must implement and verify all of the following contracts; actual evidence is recorded in the handoff:

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

**Delivery:** **First** = first complete repository-to-environment workflow; **Next** = subsequent functional coverage; **Integration required** = a required Soda workflow whose upstream interface needs investigation/implementation. It is not deferred or satisfied by a Forgejo link. Later functional coverage is not merely visual polish.

**Authority:** **F** = Forgejo; **S** = Soda; **F/S** = composed view that preserves both authorities.

### Application and account

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Sign-in entry / session expired | Complete native-backed sign-in/challenges in Soda and explain failure/expiry; no independent Soda password verification/store | F/S | First; headless authentication integration required |
| Projects | Repository discovery with environment status and authorized actions; filters/search/pagination | F/S | First |
| My work / overview | My repositories/environments, assigned issues and requested reviews | F/S | Next |
| Global search | Repositories, issues and pull requests within native visibility | F | Next |
| Notifications | Inbox, unread filters, mark read and links to subjects | F | Next |
| User profile | Native user identity and visible repositories/activity | F | Next |
| My profile / preferences | Forgejo identity, existing Soda profile fields, appearance/preferences | F/S | First |
| Development-access keys | Register/list public SSH keys used at Soda join time; explain installation scope | S | First |
| Git keys | Forgejo SSH keys; GPG-key management as the next extension | F | First / Next |
| Account security | Password, email/security verification, MFA/passkeys, recovery, authorized applications and personal API tokens | F | Integration required |
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
| Issue boards | Forgejo's issue-project/board functionality, distinctly named from Soda environments | F | Integration required |

Start with standard Markdown and safe rendering; do not silently claim exact parity with every native template, mention, attachment or Markdown extension. Inventory those details against the pinned provider when implementing each feature.

### Automation and other collaboration

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Actions overview | Workflow/run lists and status | F | Next |
| Run detail | Run/jobs/check information exposed by supported interfaces | F | Next |
| Job logs / artifacts / run controls | Log streaming, downloads, cancellation and reruns require complete Soda integration | F | Integration required |
| Releases | List and release detail/downloads | F | Next |
| Create/edit release | Tags, notes and release assets | F | Next |
| Wiki | Index, page, editor and revision history | F | Next |
| Packages | Owner/package list, version detail and files/install guidance | F | Next |
| Advanced activity/graphs | Activity and graphs beyond the initial code/commit screens | F | Integration required |

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
| Destructive/ownership settings | Repository rename/transfer/archive/delete and organization rename/delete remain required; linked Soda records/access must follow current native ownership/authorization without stale creator authority; preserve Linux state, not a competing lifecycle policy or automatic environment mutation | F/S | Integration required |

Displaying native organization/team functionality does not implement an organization-to-Linux-project-administrator mapping. The existing ordinary human-owner environment rule remains the first-version rule. Do not introduce copied roles, membership synchronization or automatic environment deletion.

### Forgejo site administration

Administrator functionality is part of the unified frontend, not just developer repository screens. The adapter delegates upstream-owned operations rather than implementing a second administration backend. H01 now records the native action/authority gaps; U16 implements complete contracts/views with early U04/U05 security review, not another broad audit after collaboration is finished.

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Administration overview | Available upstream admin functions and reported instance information, alongside separately labeled Soda environment information | F/S | Next |
| People | Authoritative Forgejo user inventory with Soda environment associations where needed; not the local Soda profile table presented as all users | F/S | First |
| Add/edit person | Forgejo account creation and supported account fields/status; account edits do not automatically rename Linux users or revoke existing SSH sessions | F | First / Next |
| Organization/repository administration | Upstream instance-wide administrative views and supported operations, supplementing the scoped settings above | F | Next |
| Instance webhooks | Native hook configuration and supported inspection/test operations | F | Next |
| Provider runner administration | Forgejo registration/runner/job views; local service execution/capacity stays in Cockpit | F | Next |
| Native quota administration | Forgejo's own quota rules/groups where supported; not a new Soda environment quota system | F | Next |
| Authentication/configuration/maintenance | Upstream settings, authentication configuration and supported maintenance tasks; API gaps require backend integration | F | Integration required |
| Native identity lifecycle actions | Required upstream deletion/rename/security operations need explicit handling of Soda associations; no generalized account remapping or implicit environment deprovisioning | F/S | Integration required |

The existing scoped operator bootstrap remains the first-install path. Adding administrator screens neither exposes an unfinished installer nor grants arbitrary host commands. Source/schema coverage alone does not authorize executing live account, runner or maintenance operations.

### Soda environments and operator pages

| Page or surface | Contents | Authority | Delivery |
| --- | --- | --- | --- |
| Create environment | Pick an eligible repository, show owner rule, invoke real provisioning and report its result | F/S | First |
| Environment overview | Actual native state, environment identity, associated repository and connection readiness | F/S | First |
| Members / join | Existing membership, explicit Add me, missing-key guidance and real account provisioning | S | First |
| Connect | Observed project IP, own login, SSH/SCP/SFTP/editor guidance and route/host-key caveats; may be a detail tab | S | First |
| Workspace terminal | After choosing a project to work in, open a browser terminal into the signed-in user's existing project-local workspace, as that user | S | U07 design/implementation; U17/U20 complete coverage/proof |
| Incomplete/unavailable environment | Honest native failure and operator guidance; not a destructive recreate button | S | First |
| Environment/access administration | Soda-specific environment status, association, membership and development-access views/actions, with the existing project/operator boundaries | S | First |
| Operator tools entry | Soda administration navigation and the separately retained host-operator Cockpit entry; links do not grant access | F/S | First |
| Host administration | Stock Cockpit services/logs/network/storage and retained Tailnet/Runners pages | Native host | Keep native |

**New requirement — browser workspace terminal:** after selecting a project to
work in, a user whose workspace already exists there can access its terminal in
the browser. “Workspace” means their existing project-local Linux account/home
inside the persistent shared environment, not a new per-user container or a host
account. Enforce access to the selected project and the user's own account on the
server; do not grant a host-root or shared administrator shell. Selecting a
project/opening its terminal must not silently create a workspace or join the
project. Missing membership/workspace keeps the existing explicit join flow.
The revised implementation plan assigns terminal design/implementation/tests to
**U07**, with U02/U03/U04 build/security review and U17/U20 final coverage. Transport,
authentication, lifetime/disconnect and stopped-project behavior still require
review; no mechanism or automatic start is selected. Ordinary SSH/SCP/SFTP remain
supported. U08's accepted earlier scope is unchanged. This assignment is not an
implemented feature or authorization to execute commands.

No browser IDE, tool/service catalog, managed toolchain branches, environment deletion/rebuild, updater, backup platform or generalized recovery UI is added by this inventory. Normal native development tools remain the workflow inside projects. Native operator setup remains the installer/bootstrap path, not a new publicly exposed setup wizard.

### API coverage evidence and limits

The [H01 register](forgejo-api-coverage.md) supersedes the earlier schema-only inventory with full v15.0.7 source-surface tracing and targeted v16.0.3 comparison. It distinguishes suitable stock APIs, bounded Soda adaptation, required native additions and existing-adapter/UI-only work. V16 supplies human Actions jobs/logs/artifacts/cancel interfaces, but neither inspected version closes blame/net diff or complete authentication/review/board/admin coverage. This is source evidence, not installed conformance, baseline selection or patch approval. Consult the exact action/actor/configuration contract before implementation; do not scrape, borrow cookies or invent unsupported APIs.

## Direct dependency inventory

This is a **direct** dependency inventory with selected and proposed entries, not every transitive package or OS RPM. `dashboard/package.json`, its real `pnpm-lock.yaml`, Go metadata and native recipes own actual pins. React Router/Markdown dependencies are resolved and the implemented frontend subset has local build/test evidence; full transitive/license and feature acceptance remain U01/U02 work. Proposed additions below are not requirements to install unused packages or upgrade existing pins.

Most of the requested stack already appears in `cockpit/package.json`. Reuse that baseline where appropriate without coupling dashboard code to Cockpit privileges or incidentally upgrading the existing operator pages.

### Browser dependencies

| Dependency | Purpose | Existing shared baseline / selection |
| --- | --- | --- |
| `react` | UI | 18.3.1 |
| `react-dom` | Browser rendering | 18.3.1 |
| `react-router-dom` | Client-only URL routing, not SSR/framework mode | 7.18.3 selected/resolved; subset locally built/tested, full acceptance pending |
| `zustand` | Feature stores, request status and UI state | 5.0.15 |
| `@patternfly/react-core` | Forms, layout, navigation, feedback and other UI components | 6.6.1 |
| `@patternfly/react-icons` | Icons | 6.6.1 shared baseline; not currently a direct dashboard dependency |
| `@patternfly/react-table` | Repository/issue/member tables | 6.6.1 shared baseline; not currently a direct dashboard dependency |
| `@patternfly/patternfly` | Base CSS, design tokens and assets | 6.6.1 |
| `react-markdown` | README/issues/wiki rendering without enabling raw HTML | 10.1.0 resolved; subset locally built/tested, full content/license acceptance pending |
| `remark-gfm` | Standard GitHub-flavored Markdown features | 4.0.1 resolved; subset locally built/tested, full content/license acceptance pending |
| `react-diff-view` | Commit/PR diff and review presentation; add when implementing that feature | Candidate pin/API to verify |

`fetch`, `AbortController`, `URL` and browser file/clipboard APIs need no package. Use explicit feature-owned Zustand stores and a small HTTP client. Handle cancellation/stale responses, pagination, errors and post-write reloads deliberately; do not build a generic query/cache framework inside Zustand. No TanStack Query or Router, Axios, Redux or second state system.

A syntax-highlighting package such as `refractor` is a **polish candidate**, not required for the first working code viewer. A plain text editor is sufficient initially; no Monaco/CodeMirror dependency or browser IDE is selected by this plan.

### Build, types and testing

| Dependency/tool | Purpose | Existing shared baseline / selection |
| --- | --- | --- |
| `vite-plus` | Dev server, production bundling, lint/format/test tooling | 0.3.0 |
| `@vitejs/plugin-react` | React development integration/Fast Refresh | Unselected candidate; verify an actual need/compatible pin before adding |
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
| `golang.org/x/oauth2` | Possible standard OAuth helper | Unselected; existing Go exchange/refresh is implemented, so require a concrete need before adding |
| Forgejo | Native identity, repositories, collaboration and administration | Existing 15.0.7 service; U01 reviews supported source baseline, U02 owns reviewed native source/patch build |
| Caddy | Existing HTTPS entry point | Keep current image/configuration baseline |
| Podman/systemd/native helper | Existing application and project runtime | Keep current host/source baseline |
| Cockpit, Tailscale and runner integrations | Existing operator functionality | Retained, not dashboard JS dependencies |

The host, project OS, Git/OpenSSH/mise/Tea/GitHub CLI and native workload dependencies remain governed by current recipes/lockfiles; no incidental upgrade is selected. The reviewed Forgejo source-build work must account for its own native Go/frontend/runtime pins, GPL/source/notices and exact image identity through existing packaging—not assume Soda's toolchain or Swagger's MIT license covers its distribution. No Node production service, additional database, Redis, GraphQL, Go web framework, ORM or second authentication authority is selected.

## Implementation order and working criteria

Follow the [post-H01 U01–U20 execution order](dashboard-implementation-plan.md#6-milestone-map-and-execution-order), not a fresh foundation-first restart. It prioritizes baseline/contracts/maintenance and early native authentication design; source-build/compatibility and first blame/net-diff plus authentication proof; then feature-owned complete workflows, including U07 terminal and early U16 administration. Stock APIs and existing Soda adapters remain reusable. U17 proves complete candidate coverage/update behavior, U18 performs the preserved default/ingress cutover, U19 measures whole-product improvement and U20 owns final installed acceptance.

U08 remains accepted for recorded native x86_64 two-developer/direct-access/shared-resource/workload/lifecycle proof, not all configurations, headless authentication, terminal, final revision or aarch64. E01–E03/media remain unselected. Local builds/tests are already authorized within the recorded scope; deployment, real provider mutations and infrastructure/lifecycle actions remain separately scoped. This planning revision performs none of them.

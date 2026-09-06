# Unified Soda frontend and backend implementation plan

**Status: implementation in progress; 0/20 core milestones meet full acceptance.** The React preview, backed-up migration and first two-developer provisioning journey have installed evidence on `soda-test`. Substantial feature implementation and native verification remain—not just polish. The current snapshot below supersedes earlier source-only/pending statements for the specifically verified work; it does not relax the detailed acceptance criteria. See [native evidence](implementation-status.md#native-first-developer-fixtures-and-complete-core-build) and [implemented API contracts](dashboard-api.md).

**Goal:** one usable React frontend for upstream Forgejo and Soda's development-environment extension, including developer and administrator workflows. First make real workflows work; then make them good. Security, accessibility basics, truthful failures and preservation of work are part of “works.”

**Read with:** [page/dependency inventory](dashboard-plan.md), [architecture and authority boundaries](architecture.md), [deferred scope](deferred.md), [actual execution evidence](implementation-status.md), [installation](installation.md) and [native validation](native-validation.md).

This is the leading implementation sequence for the **core product**: frontend, Go API/session/data integration, production native environment/access mechanisms and their product acceptance. The [native support porting plan](native-porting-plan.md) covers outside VM/SSH/evidence/artifact tools and retained host-operator integrations, under the [coordination contract below](#coordination-with-native-support-porting). **If the plans conflict, this plan wins.** Native support must not create a second core implementation, test suite or readiness gate.

The [initial M01–M18 plan](implementation-plan.md) remains historical context for the existing Go + HTMX/native implementation; its completed source work is reused, not implemented again. U08/U20 revisit its still-unverified native requirements. Milestone numbers here do not inherit earlier PASS records.

## Current execution snapshot — approved routed-access follow-up

The source/provisioning baseline is `cde7ebb`; the subsequent approved tunnel and direct-access evidence below extends that baseline without changing installed application bytes.

### What is actually built and installed

- **Location:** this workspace is on x86_64 infra (`linux-infra.dimensionlab.net`), with the existing `soda-test` VM accessible through the pinned local SSH tooling. Builder SSH-to-self is not a blocker.
- **Dashboard:** candidate `35df189`, schema v3 encrypted grants, installed React preview at `/app/`. Default browser routes remain HTMX; **U18 has not happened**.
- **Core payload:** full x86_64 build/stage/seal succeeded at `8417a90`; only its matching host helper and new-project image were installed, not the entire core payload. Existing project containers are not replaced on image updates.
- **Checks:** Cockpit type checks and 60 tests; dashboard type checks and 21 tests; 12 build-fixture and 9 staging tests passed. The aggregate native check stopped on private-directory fixture assumptions. `eca7673` corrected those fixtures and the full pinned Go suite passed afterward; **the complete aggregate entrypoint was not rerun successfully**.
- **Migration/authentication:** private consistent backups, isolated schema 1→3 rehearsal, missing/wrong-key and missing-asset startup refusals, operator OAuth/consent/session/navigation/logout checks. The rehearsal preserved one user/two sessions but had no existing projects/keys/memberships; it does not prove populated-state migration or a live rollback.
- **Real developer state:** Alice and Bob have native accounts, completed first-password change/OAuth, registered development public keys, created private repositories and persistent environments, and explicitly joined. Bob also joined Alice's environment without project-administrator rights. Native collaboration, private-repository visibility and administrator/owner denials were checked separately from Linux membership.
- **Connection evidence:** authenticated connection authorization and independent public host-key verification passed. After explicit routing approval, infra now reaches project IPs through a project-only Layer-3 SSH tunnel. Alice in her project and Bob in both projects passed direct SSH, interactive PTY, bidirectional SCP/SFTP, non-host-root UID mapping and expected sudo boundaries. Alice's public-key authentication to Bob's unjoined project was denied. Personal SSH configuration/agent forwarding is disabled in the test. Earlier home/public-key/Tea/gh observations used operator execution; shared tools/workloads are not yet proven through the client. The browser fixture run used explicit resumption after selector failures following successful writes; it was not a clean first-install run.

### Milestone status and remaining work

“Locally checked” means the implemented subset has source/build/test evidence, not that all milestone cases or installed workflows passed. “Installed subset” likewise does not mean milestone acceptance.

| Milestone | Current progress | Still required |
| --- | --- | --- |
| **U01 — Audit** | Partial source-backed endpoint/scope/authority inventory | Complete every page/sub-action mapping, dependency/license closure and remaining native capability audit. |
| **U02 — React foundation** | React shell, real lockfile, assets and packaging built; preview navigation installed | Finish asset/header/route/permission and development-arrangement acceptance coverage, local branding/notices and retained-service regression checks. |
| **U03 — API/migration** | JSON/security foundation and encrypted schema v3 locally checked; native migration subset exercised | Complete populated-state preservation, incompatible/failing migration and installed API/security failure cases. |
| **U04 — SSO/grants** | Session-bound encrypted grants and refresh source locally checked; real operator/developer login, consent and logout | Installed expiry/rotation/concurrent refresh, replay, multi-session isolation and logout-race/security coverage. |
| **U05 — Accounts/keys/People** | Connected profile/Git-key/People/development-key source; actual two-user onboarding and non-admin denial | Remaining account/key/UI cases and full Forgejo-admin/non-Soda-operator versus Soda-operator authority matrix. |
| **U06 — Repository basics** | Discovery/create/tree/README source locally checked; actual private creation and collaboration visibility | Native pagination/ref/empty/binary/large-file/download cases and fuller private/collaborator/security coverage. |
| **U07 — Environments** | Actual two-project creation, explicit account/key provisioning, memberships and public connection inspection | Partial native/DB failure, invalid/missing-key, stopped/unavailable, forged-target and cross-project cases; usable client access belongs to U08. |
| **U08 — First product proof** | Build/migration, first developer provisioning and routed SSH/PTY/SCP/SFTP verified; owner/member sudo and unjoined-project authentication checked | Personal Git clone/commit/push; remaining host-engine/isolation checks; truly shared files/mise installs; nested HTTP/database workload and bind mounts; existing-container stop/start and VM reboot persistence. |
| **U09 — Code/history/writes** | Connected history/refs/compare/file-write/fork/basic-import source locally checked | Blame and fuller diff/import coverage; real Git readback, stale/protected writes, forks and import failure journeys. |
| **U10 — Issues** | Connected issues/comments/labels/milestones/reactions/subscriptions/bounded attachments locally checked | Structured native templates, remaining comment attachment/reaction detail, expanded failure cases and real two-user collaboration proof. |
| **U11 — Pull requests** | Connected revision-bound review/merge and PR inspection source locally checked | Existing inline threads, old-side positions, team reviewers and real reviewer/merger/conflict/check-failure/stale-head journeys. |
| **U12 — Settings/orgs/teams** | Connected settings/access/protections/hooks/org/team subsets locally checked; native direct collaboration exercised | Advanced protection/team-policy forms and remaining sub-actions; full owner/collaborator/team/admin matrix; webhook API-gap disposition. |
| **U13 — Work/search/notifications** | Connected and locally checked; installed operator notification reads | Real multi-user visibility/update/activity journeys, pagination, rapid query/account switching and advanced coverage decisions. |
| **U14 — Actions** | Runs/tasks/workflow discovery/dispatch/configuration source locally checked | Approved real runner/run/dispatch/secret-variable journeys, expiry/large-output cases; job/step/log/artifact/cancel/rerun human-interface gap disposition. |
| **U15 — Releases/wiki/packages** | Release/assets subset connected and locally checked | **Implement wiki index/page/editor/history and package owner/version/file/install-guidance views**; release-specific UI tests and native release/asset/wiki/package permission/conflict/transfer proof. |
| **U16 — Administration/security** | Initial People capability only; wider work pending | Expanded account editing, administration overview, instance org/repo/hooks, supported quota/maintenance/provider-runner views; account-security/interface audit and complete admin permission tests. |
| **U17 — Coverage closure** | Partial coverage register and identified upstream limitations | Reconcile every inventory row; implement supported missing interactions; obtain explicit dispositions for unavoidable native dependencies. Known gaps include webhook PATCH, Actions web-only interactions and release empty-field PATCH semantics. |
| **U18 — SPA cutover** | Not performed; preview and legacy UI coexist | Complete U08 and agree U17 boundary; rehearse backed-up deployment, move React to `/`, preserve bookmarks/state, remove replaced HTMX paths/assets and document lossless rollback limits. |
| **U19 — Polish** | Baseline UI/error/security handling exists; dedicated acceptance pending | Real-task usability, keyboard/focus/accessibility, responsive forms/tables/diffs, measured bundle/render/request performance and before/after evidence. |
| **U20 — Final acceptance** | Earlier build/installed subsets provide reusable evidence only | Final-revision full/race/UI/browser/packaging regressions; fresh installation and controlled populated-state upgrade; complete developer/collaboration/admin journeys; independent native x86_64 and aarch64 proof and final handoff. |

### Next execution and decisions

1. **Finish U08 from the retained fixtures.** The user approved the private tunnel, and direct project-IP access from infra now works. The route is runtime-only: `10.89.0.0/24` through `tun8417`, with interface-specific firewall restrictions. No LAN/Tailnet routes or global forwarding sysctls changed. This does not route the user's laptop automatically.
2. Next exercise personal Git, shared tools and nested workloads through that client path. Obtain explicit permission before fixture stop/start and reboot of **only `soda-test`**; preserve and compare existing state. Retain tunnel/evidence resources for these tests; exact teardown instructions are in the handoff.
3. **Continue independent source work:** U15 wiki/packages, U16 administration, missing U09–U14 interactions and focused tests. Routing is not a blocker to this work, and checkpoints are not completion.
4. Approve an exact disposable runner/repository before real Actions dispatch. Select approved native aarch64 and fresh-install targets for U20; neither has acceptance evidence yet.
5. Close U17 with explicit upstream-gap decisions before U18 cutover, then complete U19/U20. The current preview rollout is not authorization to discard data or perform final cutover.

**Preserve current state:** `u08-alice-8417` and `u08-bob-8417`, their repositories, project writable roots, accounts, keys and memberships are real retained resources. Private fixture inputs/bindings are under `.artifacts/test-vm/u08-8417a90/`; detailed backup/evidence locations are in the handoff. The pre-migration backup predates these writes and is **not a lossless rollback now**. Do not rerun bootstrap, recreate fixtures, replace containers or restore the old DB as a repair shortcut.

**Not selected:** E01–E03 (image profiles, lifecycle controls, resource limits) remain conditional. ISO/QCOW2 delivery is separate optional support work; **no installable SodaOS ISO has been built**, and ISO delivery is not a prerequisite for completing this dashboard plan.

## 1. Governing decisions

1. **Forgejo is upstream.** Its users, account fields, organizations, teams, permissions, repositories, collaboration, CI and administration remain upstream-owned. Use supported interfaces; do not fork its backend, access its database directly or build a competing forge/permissions database.
2. **Soda is an extension.** Its backend owns the integration that makes persistent development environments, project-local accounts, public development-access keys, memberships and connection information usable. Linux/OpenSSH/Podman remain the native mechanisms.
3. **Frontend:** TypeScript, client-rendered React, PatternFly, Vite+, Zustand and ordinary `fetch`. React Router is a browser routing library only. No SSR, JavaScript production server, Tailwind or TanStack.
4. **Backend:** retain Go, `net/http`, SQLite and the fixed-operation Unix-socket host helper. The browser-facing process stays unprivileged. A page in Soda does not move that page's business rules out of Forgejo.
5. **Authentication:** Forgejo OAuth code flow with S256 PKCE, Go-owned secure sessions and protected server-side user credentials. Never use the bootstrap operator token to rescue an unauthorized developer request.
6. **Administration:** build Forgejo administrator views as upstream-authorized clients, Soda environment/access views under their own narrow rules, and retain host administration in Cockpit. These are not a new universal administrator role.
7. **Persistence:** keep existing project IDs, memberships, writable roots, accounts, homes, tools, service data and SSH host keys. A frontend migration must not recreate environments.
8. **Initial environment scope:** retain one Rocky-based environment per repository, the human-owner administrator rule and explicit join for every person, including the creator. OS choice and other extensions have explicit conditional tracks below.
9. **Provider gaps remain visible.** A documented native fallback is an interim integration, not completed custom-screen parity. Unsupported APIs do not justify replacing the upstream subsystem.

## 2. Baseline and changes actually needed

Baseline source inspected at `6f7b51e`; this dashboard plan was committed in `55ce5cb` and then coordinated with the native support proposal from `9c8d672`. Those are documentation changes, not implemented U/E milestones. Recheck the merged HEAD and local work before implementation.

| Existing source | Reuse | Required change |
| --- | --- | --- |
| `internal/web/`, `cmd/soda-dashboard/` | Server lifecycle, configuration, security/session concepts and existing product flows | JSON API, React static delivery and eventual removal of HTMX/template callers |
| `internal/forgejo/client.go` | Selected upstream, bounded HTTP transport, bootstrap and repository calls | Per-session user credentials, typed errors and the additional supported developer/admin operations |
| `internal/store/store.go` | Stable identity links, public development keys, project/membership records and hashed sessions | Ordered migrations, protected OAuth credential lifecycle and a small amount of explicit environment result metadata |
| `internal/host/` | Fixed `create`, `inspect`, `account` operations and Unix-socket client | Needed non-secret inspection details and tested API integration; new controls only in an approved extension |
| `project-os/`, `soda-project@.service` | Persistent Rocky userspace, native account setup, shared mise paths and existing-container startup | Fix concrete native defects discovered during proof; do not replace the runtime speculatively |
| `cockpit/` | Existing React/PatternFly/Vite+/Zustand dependency baseline and applicable UI patterns | Preserve separate build, privileged bridge and Tailnet/Runners behavior |
| `assets/` | Canonical artwork, palette, attribution and native branding | Correct imports/bundling in the dashboard, not new artwork |
| `scripts/`, `appliance/`, `tests/` | Native packaging, activation, staged checks and opt-in installed journeys | Include frontend assets and schema/key migration needs; expand tests without adding automatic live execution |

Important current limitations:

- OAuth requests only `read:user`; its access token is used for identification and not retained. Refresh credentials are not handled.
- Repository reads currently use the operator credential with filtering. That must not become the general Forgejo frontend API.
- Mutation protection is form/POST-specific, and failures/expired sessions return HTML or HTMX redirects.
- Soda's current People list is its local profile table, not the full upstream user inventory.
- The host helper accepts one configured image; project records have no per-environment image choice. The database enforces one environment per repository.
- Native environment status exposes ID, IP and a running boolean. There is no dashboard start/stop API, propagated key rotation or general recovery controller.
- Earlier source checks and operator browser evidence do not validate the merged tree, real project routing, two-user SSH, nested workloads or full persistence.

## 3. Scope and decision register

### Included in the core U milestones

- Replace the dashboard's presentation with the selected React stack while keeping existing native/operator functionality.
- Real Forgejo-backed user and administrator screens for the supported page inventory, not mocked repositories or a second identity system.
- Safe per-user provider access, environment creation/join/status, development key registration, direct-IP connection guidance and basic environment administrator visibility.
- Ordered data/config migration, packaged static delivery, regression coverage, native proof, final cutover and subsequent UX/performance improvements.
- Explicit investigation and disposition of every native/API-gap page, including administrator and account-security screens.

### Decisions with safe defaults

| Decision | Default while unresolved | Decision point |
| --- | --- | --- |
| Exact new package versions | Reuse existing shared pins; verify new packages without incidental upgrades | U01/U02 |
| Forgejo OAuth scopes, refresh behavior and admin capabilities | Only verified minimum scopes for the implemented operations; no admin-token fallback | U01/U04/U16 |
| API gaps: boards, logs/artifacts, account security, site configuration | Keep native links, record the missing interaction and seek a supported/upstream solution | Owning milestone, closed out in U17 |
| Existing sessions without user credentials | Explicit reauthentication for the new API; never mint credentials from local identity rows | U03/U04 |
| Legacy Soda display-name data | Preserve it without writing it into Forgejo or treating it as upstream identity | U03/U05 |
| Rocky versus Fedora selection | Rocky only until the E01 scope and release/image contract are approved | E01 after U08 |
| Browser start/stop/restart | Retain native operator controls until E02 is approved | E02 after U08 |
| Environment resource limits/usage | No new quota/admission subsystem; basic caps only if E03 is approved | E03 after U08 |
| Org-owned environments, multiple environments per repository | Keep the current owner rule and uniqueness constraint | Separate product decision |
| Key rotation/offboarding, account remapping, destructive lifecycle and recovery | Remain deferred; do not imply that provider edits revoke Linux access | Separate product decision |

Choosing to plan an extension is not choosing to ship it. Approval must update this register and the relevant [deferred boundary](deferred.md) before its implementation. Native Forgejo quota/admin features are not authorization to introduce Soda environment quotas or Linux account-remapping machinery.

Not included: host UI replacement, a new forge/CI engine, a web IDE/terminal, a tool/service marketplace, private toolchain/service branches, project DNS or an SSH gateway, unrestricted Podman access, an ISO/bootc/updater platform, or a generic reconciliation/backup system. Existing native development tools and host Cockpit remain usable.

## 4. Architecture, source ownership and dependencies

```text
React browser application
  /api/session, /api/forgejo/..., /api/environments/...
                         |
                    Go dashboard
              /           |            \
    upstream Forgejo   Soda SQLite   fixed-operation Unix socket
                                          |
                                     root native helper
                                          |
                              existing Podman environments
```

Proposed source organization; these are paths to create/adapt, not existing implemented modules:

```text
dashboard/
  package.json, pnpm-lock.yaml, vite.config.ts, tsconfig.json, index.html
  src/app/                 router, shell, session startup and error boundaries
  src/api/                 small fetch client and explicit TypeScript API contracts
  src/components/          genuinely reused PatternFly compositions
  src/features/            accounts, repositories, environments, issues, pulls,
                           actions, organizations, administration, releases, wiki, packages
  src/styles/              canonical palette integration and minimal overrides
  tests/                   shared browser/DOM test setup
  dist/                    ignored generated assets
internal/web/              HTTP routes, session integration, explicit handlers/DTOs
internal/forgejo/          supported upstream HTTP operations and error handling
internal/store/            concrete Soda queries and ordered migrations
internal/host/             fixed native protocol, client and root helper
cmd/soda-dashboard/        application composition, configuration and shutdown
appliance/, scripts/       packaging, activation and explicit build/check entrypoints
tests/installed/            authorized browser and real native product journeys
```

Keep each feature's components, request actions, types and tests together. Split files by real responsibility; do not build generic resource frameworks, separate microservices per page, a permissions engine or a utility package full of unrelated branches. Do not import Cockpit's privileged transport into `dashboard/`.

### Dependency work

The full direct-dependency inventory and existing versions remain in [dashboard planning](dashboard-plan.md#direct-dependency-inventory); manifests/lockfiles are authoritative once implemented. Do not repeat independent version rules here.

- **U02:** React/DOM, browser React Router, Zustand, PatternFly core/icons/table/base styles, TypeScript, Vite+, compatible React development integration, type definitions and existing DOM/browser test tools.
- **U04:** evaluate the proposed `golang.org/x/oauth2` addition for standard exchange/refresh support; retain existing Go/SQLite/crypto/native dependencies.
- **U06:** safe Markdown rendering and GFM support for actual README behavior.
- **U09/U11:** evaluate/pin the proposed diff-view dependency when it has real callers.
- **U19:** add syntax highlighting or other performance dependencies only for demonstrated needs. No speculative editor dependency.
- Use a dedicated `dashboard/` manifest/lockfile, consistent with the existing separate `cockpit/` build. No workspace-wide restructuring is required to start.
- Node/pnpm are development/build tools only. No Axios, TanStack, Tailwind, Redux, Node server, new database, ORM, Go web framework or GraphQL layer is selected.
- Review licenses, transitive closure, native tooling architectures and generated notices. Do not fabricate lockfiles/checksums or claim compatibility from version strings alone.

### Coordination with native support porting

The [P plan](native-porting-plan.md) supplies outside helpers. It is subordinate to this core plan and does not redefine product scope or move runtime ownership merely because code runs on Linux. In particular, production `internal/host/` and `project-os/` remain core, not P-plan infrastructure.

| Shared area / contract | Core owner | Outside support owner and limit |
| --- | --- | --- |
| React, Go routes/DTOs, provider authority, OAuth/session/CSRF and Soda migrations | U01–U19 as assigned | No P implementation of these; consume existing app/test entrypoints |
| Production project helper, account/key setup, shared tools, workload runtime and persistent roots | U07/U08; policy/image/control additions only under approved E01–E03 | P02/P03 offer VM/SSH/observation primitives, not project APIs, image-policy decisions or alternate runtimes |
| Application asset payload and build order | U02 | P04 inspects/bundles the specified output; does not create a second frontend build or missing-asset substitute |
| Application config, secret/key provisioning, bootstrap, migrations and cutover | U03/U04/U18 | P05 transports private inputs/invokes approved existing steps, never copies OAuth/bootstrap or edits Soda/Forgejo databases |
| `scripts/`, `appliance/`, shared config and staging checks | U02/U03/U04/U18 own app payload/semantics | P04/P05 own outer artifact format/integrity and CoreOS provisioning transport; agree changed interfaces before either edits them |
| Product `tests/installed/dashboard.mjs`, project/shared-tools/workloads checks and fixtures | Owning U feature, integrated U08/U20 | P tools may invoke these exact entrypoints and retain evidence; do not copy browser flows or scenario catalogs into `internal/acceptance/` |
| Host/service substrate checks, native Cockpit/Tailnet/local runners, console and native branding delivery | Preserve compatibility in U02/U18 and consume results in U20 | P06 host observations and P11 outside integrations; no Forgejo Actions/admin UI or dashboard redesign |
| Exact-candidate/architecture evidence | U08/U20 own product assertions and overall readiness | P12/P13 hand off artifact/host/operator evidence, without independently qualifying the product |
| Optional ISO/QCOW2 delivery | No core prerequisite; current CoreOS installation remains usable | Conditional P09/P10 only after explicit delivery selection; no bootc/Anaconda/updater decision implied |

**Single implementation, reusable evidence:** name the owning milestone, affected paths and input/output contract before shared-file work. Keep existing build/install/test entrypoints and extend them coherently, instead of adding parallel builders, bootstrap scripts or product suites. Cross-plan callers pass exact revision, artifact identity, target/client and private input references; logs retain observations, not a second authoritative product database.

**Overlap disposition:** former P07 developer/workload and P08 persistence work now belongs exclusively to U08/U20, with its useful techniques retained in the [core proof detail](#core-owned-native-proof-detail). P06 is limited to installed host/service facts; application browser/auth assertions stay U04–U08. P11 owns host-operator/companion behavior, not developer fixture/authentication logic. P12/P13 hand off their own observations rather than duplicate U20 acceptance.

**Ordering without a second gate:** core U01–U07 source and U08 execution can use the existing authorized scripts/VM tools. P02–P06 can later supply safer reusable fixtures/artifact inputs; the entire P plan is not a dependency. U18/U20 do not wait for optional media or P12 completion, and P12/P13 do not wait for a U20 product verdict to report support results. U20 consumes available relevant evidence and records any missing architecture/integration proof. Unavailable routing or permission remains a real U08 blocker, not permission to substitute forwarded access.

Defects follow their owner: a QMP/SSH/logging/bundling bug goes to P; a core auth/config/schema/project runtime defect goes to its U/E milestone. A cross-boundary fix coordinates both callers. Neither plan may weaken acceptance, change a core contract unilaterally or claim an old result validates changed bytes. The [native validation guide](native-validation.md) is shared operational guidance, not another roadmap.

## 5. Shared API, state and migration contracts

### HTTP and route boundary

- Reserve explicit `/api/...` routes for JSON. Register supported Forgejo operations individually under `/api/forgejo/...`; this is not a catch-all proxy accepting arbitrary upstream URLs, credentials or verbs.
- Keep `/login` and `/oauth/callback` as Go-owned browser redirects so the installed OAuth callback does not need an unnecessary upstream application replacement.
- `GET /api/session` returns the current identity, narrowly useful action hints, CSRF token and public configured links; `POST /api/session/logout` ends the Soda session. Provider credentials never appear in responses.
- Resource responses use explicit typed DTOs; lists include items and reliable pagination metadata. Preserve native permission/validation outcomes rather than returning successful empty lists on upstream failure.
- Errors carry a stable application code, safe message, optional field errors and a non-secret diagnostic reference. Distinguish unauthenticated, forbidden, missing, conflict, oversized/invalid input, unavailable dependency and incomplete/unknown native result.
- Require appropriate content types, bounded bodies, explicit input fields and CSRF/origin validation on POST/PUT/PATCH/DELETE. Safe GETs do not mutate provider/native state.
- API expiration returns JSON 401, not login HTML hidden inside a successful fetch. The frontend deliberately begins top-level OAuth navigation when needed.
- Treat provider IDs as opaque identifiers in the browser; choose a lossless encoding for Go int64 IDs at the DTO boundary. Repository refs/file paths must survive slashes, spaces and Unicode without confusing them with route segments or host filesystem paths.
- Requests have bounded timeouts and cancellation. Refreshing a token is not permission to replay an ambiguous non-idempotent write. A browser disconnect is not proof that native creation rolled back.
- Keep provider transport errors typed and sanitized; never expose raw response bodies, full container inspection, passwords, token-bearing URLs or project file contents in diagnostics.

### Representative API families

These are proposed **Soda** contracts to implement, not claims that these exact paths exist in Forgejo. U01 maps each operation to the actual selected upstream interface.

| API family | Responsibility | Milestones |
| --- | --- | --- |
| `/api/session`, `/api/session/logout` | Session discovery and local sign-out | U03/U04 |
| `/api/me/preferences`, `/api/me/development-keys` | Soda-only preferences and public development-key registration/listing | U05 |
| `/api/forgejo/me/...`, `/api/forgejo/users/...` | Native profile/key/account capabilities and visible user information | U05/U13/U16 |
| `/api/forgejo/repositories`, `/api/forgejo/repos/{owner}/{repo}/...` | Discovery, creation, code, history, collaboration and repository settings | U06/U09–U12/U14/U15 |
| `/api/environments`, `/api/environments/{id}` | Soda discovery, create and observed detail | U07 |
| `/api/environments/{id}/join`, `/members`, `/connection` | Native join, allowed membership information and usable connection details | U07 |
| `/api/forgejo/notifications`, `/search/...` | Upstream notifications/search; no local index of Forgejo data | U13 |
| `/api/forgejo/orgs/...`, `/teams/...` | Native organizations/teams and scoped administration | U12 |
| `/api/forgejo/admin/...` | Upstream-authorized site administration, not host-root delegation | U05/U16 |
| `/api/environments/{id}/start`, `/stop`, `/restart` | Fixed existing-environment controls, only if approved | E02 |

For environment creation, accept a repository selection and resolve its canonical ID/owner through Forgejo. Do not accept caller-chosen Linux privileges, owner IDs, native names, mounts, Podman flags or unrestricted image references. Source-backed OS profile selection is added only by E01.

### Zustand and frontend state

- Store session display state and feature-owned data/loading/error state; use a small `fetch` wrapper for cookies, CSRF, parsing and typed errors.
- Keep filters, pagination and selected refs in the URL where they are navigable state. Keep local form drafts local; do not persist passwords, tokens, private repository data or key material in browser storage.
- Abort superseded reads and ignore late results after a route, user or session change. Clear all user-specific stores and drafts at sign-out/reauthentication.
- After a successful mutation, update or reload the specific affected view. Do not add a generic query/cache framework inside Zustand.
- Show empty, pending, denied, failed and unavailable states from real results. Keep consequential actions disabled while pending; do not report optimistic environment provisioning, joining or merging as complete.
- Native fallback links must leave the SPA correctly and use configured public origins, not internal provider endpoints or guessed ports.

### Data and credential migration

- Replace the current startup-only `CREATE TABLE IF NOT EXISTS` approach with small ordered, transactional migrations that recognize the existing version-1 database and reject unsupported newer schemas safely.
- Preserve identity IDs, existing projects, key records, memberships and native container associations. Do not rebuild the database or automatically convert its `users` table into an authoritative Forgejo user directory.
- Add session-bound provider credential storage: access/refresh material, actual expiry/granted scopes and the session association needed for safe deletion/rotation. Use authenticated encryption with an operator-managed key outside the database, restricted file access and standard cryptography—not a new vault framework. Missing/wrong keys must fail closed, without a plaintext fallback or regeneration on each boot.
- Bind credential records to their session/user so one user cannot acquire another's grant. Coordinate concurrent refresh for the same grant; logout must win over an in-flight refresh and prevent session resurrection.
- A legacy session lacks the required user grant. Preserve its product records but require reauthentication before new Forgejo-backed API use. Do not manufacture a grant from the operator token.
- Preserve legacy Soda profile values; do not silently push them to Forgejo. Native account fields are fetched/changed upstream. Retain only explicitly Soda-specific preferences/annotations going forward.
- Extend environment result metadata only where U07 needs it: distinguish retained/incomplete provisioning from live observed state. No generic job tables, mirrored repository ACLs or private-resource inventory.
- Preserve the one-repository/one-environment constraint unless a separate scope decision changes it. Do not guess an existing environment's original OS/image from the current global default.
- An explicit operator migration must provide any new configuration/key file without rerunning first-install OAuth bootstrap. Schema/key migration and rollback compatibility are documented and tested on controlled fixtures before a live rollout.

## 6. Milestone map and execution order

The map specifies the full target sequence, not completed work. Current partial source work is recorded in the status above and implementation handoff; no milestone inherits PASS from an authored test. Dependencies below are source/integration dependencies unless a milestone explicitly requires installed evidence. Execute tests only under the applicable authorization.

| ID | Milestone | Depends on | Main outcome |
| --- | --- | --- | --- |
| U01 | Capability, authority and baseline audit | Existing source | Concrete API/scope map, migration decisions and dependency research |
| U02 | React foundation and early asset packaging | U01 | Installable SPA shell, isolated from Cockpit and legacy routes |
| U03 | JSON API and database migration foundation | U01 | Shared security/error contracts and safe persistent-state evolution |
| U04 | Forgejo SSO and per-session provider access | U02/U03 | Authenticated React API with no operator-token substitution |
| U05 | Account, keys and initial administrator onboarding | U04 | Real profiles/key views and upstream-backed People/create-person flow |
| U06 | Repository discovery, creation and basic code browsing | U04/U05 | A real repository workflow entirely from the new frontend |
| U07 | Environment creation, joining and connection UI/API | U05/U06 | Existing native integration exposed safely through React |
| U08 | First installed product proof | U02–U07 plus named target permissions | Two-user repository-to-SSH/shared-workload/persistence evidence |
| U09 | Code history, comparison and native repository writes | U06 | Commits/branches/tags/file edits/import/fork workflows |
| U10 | Issues, labels and milestones | U06 | Real issue collaboration |
| U11 | Pull requests, reviews and merge | U09/U10 | Native-authorized review and merge workflows |
| U12 | Repository settings and organizations/teams | U06/U09 | Upstream-owned scoped administration |
| U13 | My work, search, notifications and user profiles | U09–U12 | Coherent cross-repository navigation |
| U14 | Actions and automation configuration | U11/U12 | Provider-owned workflow/run views and supported actions |
| U15 | Releases, wiki and packages | U09/U10 | Remaining core collaboration/artifact views |
| U16 | Forgejo site administration and account-security coverage | U05/U12/U14/U15 | Complete administrator workstream, not just developer screens |
| U17 | Page/API coverage and upstream-gap closure | U09–U16 | Every inventory row has implemented coverage or an explicit unresolved disposition |
| U18 | Default SPA cutover and legacy removal | U08/U17 | One production dashboard, migrated safely |
| U19 | Make it good | U18 | Measured UX, accessibility and performance improvements |
| U20 | Final installed verification and handoff | U19 plus target permissions | Fresh-install/upgrade evidence, architecture-specific results and honest release scope |
| E01 | Rocky/Fedora creation-time image profiles | Approval + U08 | Optional second supported userspace, not an in-place distro switch |
| E02 | Existing-environment lifecycle controls | Approval + U08 | Optional scoped start/stop/restart UI and helper operations |
| E03 | Basic environment resource limits and usage | Approval + U08 | Optional native caps and observation, not a scheduler/quota product |

**Recommended serial path:** U01 → U02/U03 → U04 → U05 → U06 → U07 → U08, then functional expansion, cutover, polish and final proof. Do not wait until every forge screen exists to test the core environment.

**Parallel work:** U02 and U03 have separate owners; U09/U10 and later U12 can advance independently once their contracts are stable; U13–U16 can be divided by feature after shared APIs exist. Existing native risk investigation can begin early with separate permission, but U08's React journey is not satisfied by an old HTMX test. If native execution is unavailable, source work can continue with the verification gap recorded; cutover/readiness claims remain gated. Conditional extensions do not block core source delivery.

## 7. Detailed core milestones

### U01 — Capability, authority and baseline audit

**Primary files:** existing `internal/web`, `internal/forgejo`, `internal/store`, `internal/host`, dependency manifests, service/build recipes; new `docs/forgejo-api-coverage.md` and `docs/dashboard-api.md`.

- Freeze the implementation baseline and record unrelated working-tree changes to preserve. Read native/operator guides before planning target actions.
- Map each page/action in the inventory to the selected Forgejo version's actual endpoint, verb, credential/scopes, native authority, pagination, errors and fallback. Include private/collaborator repository visibility, user creation, admin functions, OAuth refresh/consent, files/downloads and API gaps.
- Use the saved installed 15.0.7 schema as a starting point, not proof of every runtime behavior. Inspect the matching upstream source/docs where the schema is incomplete; retain minimal public contract fixtures and provenance, never real credentials or private content. Record each row's owner, endpoint/verb, scopes, authority, request/result shape, failure cases, implementation milestone and source/native evidence.
- Record the authority matrix: ordinary user, human repository owner, Forgejo administrator without Soda operator status, configured Soda operator, and native root. Keep trusted-team environment discovery/join separate from native Git visibility.
- Set the new dependency pins through compatibility/license research and later authorized resolution. Decide the credential migration and old-session behavior before implementation; do not change the Forgejo service version as a shortcut.
- Confirm which conditional E tracks are approved. Unselected ones remain absent from implementation and schema.

**Acceptance:** every inventory family has an owner/milestone; foundational API/auth assumptions have source references or explicit verification tasks. No unsupported endpoint, new role system or silently mandatory optional feature is in the contract. Review only at this stage is not runtime proof.

### U02 — React foundation and early asset packaging

**Primary files:** new `dashboard/`, `.gitignore`, `.containerignore`, `internal/web/server.go`, `internal/config`, `appliance/dashboard.Containerfile`, `scripts/build-native.sh`, `scripts/stage.py`, `scripts/check-native.sh`, packaging tests.

- Create the client-only app, explicit router, PatternFly shell, navigation/error boundaries, basic help/empty/loading states and feature-store conventions. Import canonical branding; keep fonts/assets local and preserve notices.
- Use `dashboard/dist` as ignored build output. Stage a validated frontend bundle into the native artifact tree and `/usr/local/share/soda/dashboard`, and copy it into the dashboard image. Serve it from Go as read-only files; keep API tests independent of generated assets through explicit filesystem injection/test fixtures.
- Validate production assets and required configuration before applying persistent-state migrations, rather than serving a fake index when the bundle is missing. Serve correct MIME/cache headers; hashed assets may be cached, while the entry document revalidates. Missing assets and unknown API routes must not return SPA HTML.
- Mount the preview SPA under `/app/` while legacy routes remain available. Keep the router/build base explicit. U18 changes the production base to `/`; do not infer authorization or backend targets from browser path fragments.
- Provide a loopback-only development arrangement with API/OAuth forwarding and trusted HTTPS where real login is tested. No permissive production CORS, cookie weakening, exposed dev server or secrets in `VITE_*` variables.
- Make build/staging include the frontend from the first installable slice, not as a final packaging afterthought. Preserve Cockpit assets/PAM/branding, project CLIs and existing native artifact paths. Add an explicit dashboard-only build path for later UI iteration so it need not rebuild unrelated project images.

**Tests/acceptance:** direct-link navigation, missing assets, API-versus-SPA routing, production asset startup validation, local fonts/branding and staged file permissions. Under build authorization, the native image serves the shell without Node; Cockpit and the old dashboard still operate independently.

### U03 — JSON API and database migration foundation

**Primary files:** `internal/web` handlers/DTOs/security, `internal/store` migrations/queries, `internal/config`, `dashboard/src/api`, Go/DOM contract tests, `docs/dashboard-api.md`.

- Implement explicit JSON decoding, request limits, safe typed errors and a consistent status contract. Protect every unsafe HTTP method; do not wrap JSON handlers in the old `ParseForm`-only POST path.
- Implement session discovery/logout contracts and shared request plumbing. Keep authenticated data out of shared/public caches and preserve security headers with tested PatternFly-compatible styling—not blanket script/CSP bypasses.
- Introduce migrations recognizing schema v1, additive session-credential storage and only the required application metadata. Define the restricted external encryption-key configuration and fail-closed behavior; use standard cryptography.
- Preserve legacy IDs/data and test old/new database fixtures. Make schema upgrades transactional and stop safely on incompatible versions/errors. Document how a consistent pre-migration copy and matching key are taken on an approved target; never blindly copy a live WAL database as a backup.
- Define legacy session reauthentication, profile ownership and rollback compatibility. Do not drop old tables/fields or reinterpret Forgejo account data during foundation work.

**Tests/acceptance:** v1-to-current preservation, clean initialization, repeat startup, migration failure/newer-schema refusal, wrong/missing key behavior, JSON errors/content types, CSRF/origin checks on all write verbs, field/body limits and API expiration. Real grants remain U04 work; an empty credential table is not authenticated integration.

### U04 — Forgejo SSO and per-session provider access

**Primary files:** `internal/web/auth.go`, `internal/forgejo` OAuth/transport, `internal/store` credential/session queries, `cmd/soda-setup` only where contract changes require it, config/activation source, `dashboard/src/app` and auth tests.

- Extend the existing code/S256 flow with the U01-verified scopes and token response. Preserve state binding, single use, session rotation and the configured callback/origins.
- Keep login return destinations local and validated; store the preview/final destination with the OAuth transaction rather than trusting arbitrary redirect URLs. Do not require another password authority or changes to Forgejo cookies.
- Store grants per Soda session, refresh against the supported native endpoint, persist rotation atomically and coordinate concurrent refresh. Distinguish insufficient scope/authorization from expired credentials; require consent/reauthentication when appropriate.
- Separate a user-authorized provider client from operator bootstrap usage. Credentials are explicit internal inputs, never browser-supplied tokens, a `sudo` impersonation header or an operator fallback.
- Populate React session display/action hints from trusted information, but enforce permissions on the actual operation. Native Forgejo denials remain authoritative even if a previously rendered button was enabled.
- Clear server grants/session and user-specific stores on logout; prevent in-flight requests from repopulating a signed-out account's data. Explain local sign-out versus native Forgejo/SSH session lifetime.

**Tests/acceptance:** real browser sign-in/remembered consent when authorized; stale/replayed state, scope denial, token expiry/refresh rotation, multi-user isolation, concurrent refresh, logout races and no credential leakage. An old session is prompted to reauthenticate without losing environments. A denied request never reaches a bootstrap-token retry.

### U05 — Profiles, public keys and initial administrator onboarding

**Primary files:** `dashboard/src/features/accounts` and `administration`, `internal/web/people.go`/new API handlers, `internal/forgejo` user/key/admin calls, `internal/store` extension preferences/keys, existing auth/journey tests.

- Build My profile/preferences, development-key registration/listing, Forgejo Git-key views and native account-security links. Identity/account fields come from Forgejo; preserve existing Soda-only values with explicit ownership instead of synchronizing two profiles.
- Validate development public keys using the existing canonical parser and retain duplicate-key behavior. Never request/generate a user's private key in the browser/backend or claim that adding a key updates existing project accounts.
- Implement the initial administrator People/create-person flow through actual upstream admin authorization. List upstream users, optionally joined with permitted Soda associations—not local profile rows as the full user directory.
- Do not impose Soda's Linux username restrictions on general Forgejo account creation. Forgejo validates its accounts; unsupported Linux names are explained when that person requests a Soda environment account, without an automatic remapping subsystem.
- Support Forgejo's first-password-change/onboarding behavior without persisting passwords or user-creation drafts in Zustand. Retain the explicit setup/bootstrap path; a configured Soda operator ID does not confer arbitrary Forgejo or host privileges.
- Include ordinary-user denial, Forgejo-admin/non-Soda-operator and Soda-operator/host-root distinctions in navigation and tests. Further administrator controls are U16, not excluded from the product.

**Tests/acceptance:** authorized admin creates an actual native account, that person completes native onboarding and signs into Soda; non-admin direct API attempts fail. Key syntax/options/private-key inputs are rejected, own-user data is scoped, and Forgejo Git keys do not silently become installed project keys.

### U06 — Repository discovery, creation and basic code browsing

**Primary files:** repository feature, explicit `internal/web` repo handlers, `internal/forgejo` repository/content calls, Markdown components, API and browser tests.

- Build Projects/repository discovery, create repository, repository overview/README and ref-aware file tree/viewer. Use the upstream visibility endpoint verified in U01, including pagination and collaborators where supported; do not confuse the old owner-only environment picker with all accessible repositories.
- Let the provider create/validate repositories. Return its stable identity and real clone URLs. Repository creation does not automatically provision an environment or create a project record.
- Add safe standard Markdown with relative links resolved for the selected repository/ref. No raw-HTML/script execution or unreviewed remote content loading. Clearly delimit unsupported native rendering extensions.
- Handle empty repositories, binary/large files and downloads without unbounded buffering. Do not serve untrusted repository HTML as active content on Soda's authenticated origin or proxy arbitrary URLs with provider credentials.
- Compose authorized repository data with Soda environment summaries. If a user can see a Soda environment under the existing trusted-team policy but cannot read its private Forgejo repository, do not use the operator token to fill in the missing native content.

**Tests/acceptance:** list across pages, create a real repository, browse a non-default/slash-containing ref and README/files, preserve private/collaborator permissions, handle empty/large/binary content, and reject cross-user/URL/path injection. The original environment-picker owner rule still applies only to environment creation.

### U07 — Environment creation, joining and connection integration

**Primary files:** environment feature, `internal/web/projects.go` and JSON handlers, `internal/store`, `internal/host/client.go`/`daemon.go`, narrow project-native changes if needed, installed tests.

- Port existing discovery/create/detail/join flows to JSON/React while reusing native implementations and preserving IDs. Resolve owner, native target and granted project-local privilege server-side; retain one environment per repository and explicit creator join.
- Keep UI creation/join pending until the actual operation completes. Retain a reserved/incomplete record after a native failure; expose safe results and read-only inspection even when provisioning did not finish. Do not turn a lost HTTP response into permission to recreate the container.
- Separate product provisioning state from live observed running/ready/IP information. A cached address or `ready` row is not proof of SSH reachability. Add only bounded status/failure metadata, not a durable workflow/reconciliation engine.
- Add allowed member/own-membership views and project/operator administration context. Identity/key installation reaches the existing fixed `/account` operation; membership follows native success, and joining does not alter Git permissions.
- Provide a fixed, non-secret inspection path for public SSH host keys/fingerprints and connection details if needed. Never read private host-key files or return full container inspection/environment data.
- Build Connect with actual user/IP, SSH/SCP/SFTP/editor guidance, stopped/unavailable warnings and native host-key verification. Expose reported state honestly; routing is operator configuration, not a new browser-controlled host-network API.
- Show existing key-installation limitations. No rotation/offboarding, arbitrary command endpoint, image picker or start/stop buttons unless the relevant scope is separately approved.

**Tests/acceptance:** owner/non-owner/organization selection, forged native IDs/privileges, missing/invalid keys, explicit Alice/Bob joins, partial native/DB failures, stopped/unavailable endpoints and cross-project isolation. Fake native command tests establish handler behavior only; actual usable access is U08.

### U08 — First installed product proof

**Primary files:** `tests/installed`, build/staging checks, `project-os`/native source for concrete corrections, `docs/local-testing.md`, `docs/native-validation.md`, execution handoff.

**Requires explicit builder/VM/client/resources and permission for each mutating test.** Prefer the recorded isolated test setup; do not change builder infrastructure, erase disks or infer network/reboot authority from a build permit.

- Build/check the actual merged implementation and install the matching dashboard assets/config/schema on an approved target. Verify the unprivileged process, restricted socket/credential permissions, trusted TLS and preserved Cockpit services.
- Create approved test identities/repositories through the new administrator/developer UI. Both users sign in, register public development keys, and explicitly join the owner's environment. Native Git credentials remain personal and separate.
- From an actual routed developer client, verify the displayed SSH host key and exercise interactive SSH, a command, SCP and SFTP. Browser tunnels to the VM do not substitute for project reachability.
- Verify the owner's project-local sudo boundary, the ordinary member's lack of that privilege and absence of human host accounts/host-engine access. Use a second named project to verify independent native state.
- Prove shared files and the same installed mise tool path for both users; build/start the ordinary nested workload example and reach its HTTP/database ports with real data and bind mounts.
- With separate permission, stop/start the existing project and reboot only the approved test host. Confirm container identity, accounts, host keys, homes, installed tools, shared files and service data survive.
- Investigate user namespaces, cgroups, seccomp, capabilities and SELinux precisely. The current nested candidate's `label=disable` and extra capabilities are limitations to evaluate, not a claim of fully confined hostile-tenant isolation. Do not solve failures with unrestricted/privileged host access or an unapproved VM fallback.

**Acceptance:** record source revision, target/client, commands, safe evidence and remaining limitations. Any unavailable routing, workload or persistence step stays unverified. Correct concrete blockers and rerun affected checks; no source-only or screenshot-only substitute makes this milestone verified. P tooling/host observations may be reused, but U08 owns the product tests and their interpretation.

#### Core-owned native proof detail

These concrete techniques were retained from the incoming P07/P08 proposal and are now owned here and reused by U20. Extend the existing `tests/installed/` and bounded workload/Git fixtures; do not create a parallel Go product-scenario runner. Predecessor `product_scenarios.go`, `project_scenarios.go`, `fixtures.go` and `preservation.go`/`preservation.sh` are references for selective test reuse at `bc1d3e0`, not imported product requirements.

- Exercise populated repository discovery/selection, owner-only environment creation, native first-password change and both explicit joins through the actual frontend. Missing keys, denied provider/native operations and partial provisioning must not appear joined/ready. Do not seed sessions or Linux accounts to bypass onboarding.
- Verify project public host keys through U07's trusted HTTPS connection information or an explicitly trusted native operator channel. From the named routed client, exercise interactive SSH, exact-output commands and bidirectional SCP/SFTP. QEMU remapping, `ProxyJump` or appliance-local SSH may support management but do not establish the claimed direct project-IP path.
- Check positive authorized operations and specific permission denials. A transport/lookup failure is not proof of denied access or absent host accounts. The second project must have separate roots/host keys and scoped runtime authority; project-local UID numbers may legitimately coincide across environments.
- Use personal home checkouts and independently authorized native Git credentials for real clone/commit/push/readback. Soda login, project join and Git authorization are distinct. Tea/gh delivery/version observations may come from P11; personal API authentication/credential isolation, if exercised, uses the core's user fixtures and separately approved provider access, never a P credential broker.
- Prove both users consume the same canonical shared mise installation/files with the intended ownership/permissions and executable resolution, including noninteractive SSH. Equal version/path strings or duplicate installations/downloads are insufficient. The ordinary member can execute but cannot replace the administrator-owned shared installation.
- Exercise the actual project-local workload engine/socket boundary: image build, source bind-mount edit, changed HTTP response and a committed PostgreSQL fixture value. Bob and the actual client reach the intended service ports. Compose parsing, a listed container or an unrelated host engine is insufficient.
- Before authorized lifecycle changes, capture bounded secret-safe observations of container identity, account/provider association, groups/ownership, public host-key fingerprints, homes, dirty/untracked Git work, shared files/tools, system configuration and service data. Fail on incomplete/failed snapshots instead of comparing empty outputs. Do not export shadow files, password hashes or private keys.
- Stop/start the existing project, reconnect as both users and compare state; repeat around the separately authorized appliance reboot. Confirm a changed boot ID only to prove that a reboot occurred; exclude volatile PIDs/timestamps from persistence equality. The second project and the same writable roots must remain intact.
- Follow the documented native workload restart path. If a normal explicit workload start is needed after reboot, record it and verify retained data; do not imply automatic service resurrection or rebuild/delete volumes to get a passing query.

### U09 — Code history, comparison and native repository writes

**Primary files:** repository code/history/edit features, `internal/forgejo` content/git/ref/import/fork operations, explicit web handlers and contract tests.

- Implement commit list/detail, file history/blame where supported, branches/tags, comparison and native fork/import screens. Handle refs and pagination without a local clone or second Git index.
- Add simple file create/edit/upload with native commit metadata and conflict/precondition handling. Use upstream update/version requirements so stale forms do not overwrite newer work silently.
- Respect branch protection, native permissions and import/fork capabilities. Never run arbitrary imported repository code in the Go service or provision/copy environments as a fork side effect.
- Add diff presentation needed for commits/comparison and subsequent reviews. Large/binary changes get an explicit bounded view or native link; preserve meaningful provider errors.

**Tests/acceptance:** history/ref correctness, stale file edit, protected branch denial, unsafe content/downloads, failed import and permitted fork. Confirm results in native Forgejo and ordinary Git, not only the React state. Unsupported blame/import details remain explicit U17 coverage items.

### U10 — Issues, labels and milestones

**Primary files:** issues feature, reusable Markdown/form components, native issue/comment/label/milestone adapters and web handlers.

- Implement list/filter/pagination, create and detail/conversation flows; native assignments, state transitions, labels and milestones; supported attachments/reactions/subscriptions as recorded in the inventory audit.
- Let Forgejo own issue numbering, authorship, notification behavior and permissions. No local issue/comment store or Soda-maintained assignment authority.
- Respect supported native templates/field constraints without inventing another issue schema or interpreting project files as backend instructions. Render input safely and bound uploads.

**Tests/acceptance:** two users create/comment/assign/close/reopen through authorized native operations; native IDs and results agree with Forgejo. Exercise missing/private repositories, oversized/unsafe attachments, stale metadata and denied edits. Changing issues must not trigger environment operations.

### U11 — Pull requests, reviews and merge

**Primary files:** pull request/review features, shared diffs, native pull/review/merge adapters and web handlers.

- Implement PR list/create/detail, conversation, commits/files/checks tabs, review requests, inline comments and approve/request-changes actions supported upstream.
- Bind review positions to the exact upstream revision/diff model. A refreshed branch must not silently attach a comment or approval to the wrong revision.
- Delegate mergeability, checks, protections, approvals and merge execution to Forgejo. Handle stale heads, conflicts and native rejection without simulating a successful merge.
- Reload authoritative results after mutation. Do not apply repository migrations, promote tools/services or clean up a development environment when a PR merges.

**Tests/acceptance:** real branch/PR/review/merge flow, restricted reviewer/merger, outdated diff/head, failed checks and native conflicts. Verify native commit/PR state. No Go Git engine, CI scheduler or merge-triggered environment mutation appears.

### U12 — Repository settings and organizations/teams

**Primary files:** repository settings and organization/team features; native settings/access/protection/key/hook/org/team operations; web authorization tests.

- Implement general settings, collaborators, branch/tag protection, deploy keys, webhooks and supported metadata/settings. Sensitive values are write-only or redacted as upstream defines them.
- Implement organization directory/create/profile, members/teams and scoped settings using Forgejo-owned entities/permissions. Do not mirror membership lists into a Soda authorization database.
- Separate repository and organization administration from Soda environment administration. An organization-owned repository may be browsable/administered while environment creation remains unsupported under the current human-owner rule.
- Keep deletion/rename/transfer and linked-environment effects behind the decision register; do not automatically reassign Linux accounts or delete containers when native ownership changes.

**Tests/acceptance:** owner/collaborator/team/admin permission matrix, protected operations, unsafe hook/secret handling and denied direct API access. Provider-authorized settings change in Forgejo; no unrelated native host/project privileges change.

### U13 — My work, search, notifications and user profiles

**Primary files:** app overview/search/notification/profile features, provider query adapters and safe composition handlers.

- Implement assigned/review-requested work, visible repositories/environments, native user profiles/activity and notification inbox/read state.
- Use native search/filter/pagination; no new search index or event store. Keep Soda-only environment information distinctly sourced.
- Put navigable filters in URLs, handle partial dependency failure per view and cancel superseded searches. Never reuse one user's cached private results for another session.

**Tests/acceptance:** permission-filtered cross-repository results, native notification updates, deep links, independent empty/error states, pagination and rapid account/query changes. Native activity/graph gaps remain explicitly mapped to U17.

### U14 — Actions and automation configuration

**Primary files:** actions feature, native workflow/run/task/status/variable/secret adapters, bounded read/dispatch handlers, browser/native-provider tests.

- Implement workflow/run listings, supported run/job detail, checks and permitted dispatch/configuration actions. Forgejo remains scheduler and record keeper.
- Implement supported repository/organization Actions variables/secrets with no readback of secret values. Do not place workflow credentials in browser persistence, logs or the Soda database as a second CI secret store.
- Verify actual human-authorized interfaces for logs, artifacts, cancellation and reruns. The saved REST schema alone does not establish these; do not borrow runner/admin cookies or proxy a runner-only protocol to manufacture coverage.
- Scope polling/streaming to the viewed run, bound retention/buffering and close it on navigation/logout. Keep local runner service registration/capacity in Cockpit.

**Tests/acceptance:** real provider run/status/configuration/dispatch cases only with approved runners/repositories. Test secret redaction, unauthorized cross-repo run access, expired credentials, failed dispatch and large outputs. Unsupported interactions are native fallbacks with precise U17 tasks, not fake buttons.

### U15 — Releases, wiki and packages

**Primary files:** release/wiki/package features, supported provider adapters, shared safe Markdown/download/upload components.

- Implement release list/detail/create/edit and assets using native tags/releases; wiki index/page/editor/history; package owner/version/file views and native install guidance.
- Delegate persistence, permissions and package/release semantics to Forgejo. No Soda artifact store, registry or package format.
- Stream authorized binary data through a bounded supported path or use verified public native links. Do not forward bearer credentials to arbitrary redirects, allow path traversal or buffer unlimited uploads in Go.
- Handle stale wiki edits, duplicate tags/versions, private artifacts and deleted native objects with truthful results. Native package publishing protocols remain native developer tools unless explicitly in the coverage map.

**Tests/acceptance:** real release/asset and wiki revision operations, package visibility/download metadata, conflict/permission cases, cancellation and malformed/oversized uploads. Verify upstream state and that failed transfers do not become successful local records.

### U16 — Forgejo site administration and account-security coverage

**Primary files:** administration/account-security features, explicit native admin adapters, role/session handling, administrator browser tests and coverage documentation.

- Extend U05 into administration overview, account editing, instance organization/repository views, instance hooks, provider runner/job views and supported native quota/maintenance operations.
- Use actual Forgejo administrator authority and verified scopes for the acting session. Recheck on the operation; a stale menu hint is not authority. A Forgejo admin who is not the configured Soda operator gets no host/root or extra Soda privilege.
- Treat provider quotas, cron tasks and runner registrations as upstream features; do not introduce a Soda scheduler, quota engine or local runner coordinator in the dashboard.
- Audit native account settings, email verification, applications/tokens, MFA/passkeys/recovery, auth-source configuration and site settings. Keep credential-entry/reauthentication flows native where required; a custom account-security form is not mandatory if no supported interface exists.
- Distinguish harmless account metadata edits from rename/disable/delete effects on Soda sessions and existing Linux accounts. Do not promise deprovisioning/offboarding synchronization or implement it accidentally.
- Preserve the native operator bootstrap and Cockpit paths. Do not provide a generic configuration-file editor, host command endpoint or catch-all `/admin` proxy.

**Tests/acceptance:** admin, non-admin, downgraded admin, insufficient-scope and non-Soda-operator cases; actual upstream admin results; no credential exposure or cross-project host control. Source inspection of a maintenance API is not permission to execute it on an arbitrary instance.

### U17 — Page/API coverage and upstream-gap closure

**Primary files:** `docs/forgejo-api-coverage.md`, inventory, affected feature/adapters/tests and native fallback navigation.

- Reconcile every inventory row and sub-action with implementation, tests and observed capability. Use distinct labels: implemented, source-tested, installed-verified, native fallback, blocked upstream, or deferred by an explicit decision.
- Investigate outstanding boards, logs/artifacts/run controls, advanced graphs, account-security and site-configuration interfaces against the pinned upstream. If a supported path exists, implement it in its owning feature rather than leaving an avoidable permanent external link.
- Where it does not exist, record the exact gap, native fallback, upstream proposal/reference and effect on delivery. Ask for an explicit disposition: accept a documented native dependency for this version, wait for upstream capability, or revise the feature scope. Do not silently call partial parity complete.
- Revisit destructive/ownership actions only through the decision register. Preserve existing work and explicitly state which native operations do not propagate to Linux access.

**Acceptance:** no inventory item disappears, no placeholder is reported as a working page, and administrator coverage is as explicit as developer coverage. Final cutover uses an agreed coverage boundary; “full frontend” must not be claimed for still-unreplaced interactions.

### U18 — Default SPA cutover and legacy removal

**Primary files:** `internal/web`, `dashboard` router/build base, config/activation/build/staging source, installed browser checks and operator documentation.

- Rehearse on a migrated database/config copy and then an approved target. Preserve configured HTTPS origins and OAuth application/callback, secret ownership, database path and all project identities/state.
- Move the SPA from `/app/` to the default browser routes. Keep legacy GET bookmarks useful through explicit redirects (including old project IDs to environment detail); keep `/login` and callback contracts stable. Never redirect/replay old POST mutations as a migration strategy.
- Remove HTMX-only routes, templates, script/license payload and headers only after their consumers are replaced and tests updated. Retain attribution required for any still-used material. Remove old operator-token repository/admin UI paths coherently; do not delete bootstrap credentials that still have legitimate setup/native callers.
- Retire temporary preview/config branches after the transition. One frontend and one API implementation remain, not a permanent dual-stack framework.
- Document a bounded dashboard-only deployment/migration procedure rather than rerunning the first installer or enabling an updater. Package matching frontend/backend assets together.
- State rollback compatibility precisely: retain a consistent pre-change DB/config/key set and prior application artifact; an old binary is not assumed compatible with every new schema. No rollback may discard unrelated Forgejo changes or project writable state.

**Tests/acceptance:** existing IDs/keys/memberships survive, legacy bookmarks work, sessions reauthenticate as documented, missing assets/API paths fail correctly and provider/native fallback URLs remain valid. Cutover requires U08 proof and an agreed U17 gap list, not just successful bundling.

### U19 — Make it good

**Primary files:** dashboard components/styles/stores, API hot paths only where measured, usability/browser tests, branding documentation.

- Review real developer/admin tasks for navigation, information density, keyboard/focus flow, responsive tables/forms and legible diffs. Apply PatternFly and canonical tokens consistently rather than wrapping every component in a competing design system.
- Improve accessibility beyond the baseline: focus restoration, screen-reader announcements, error association, contrast, reduced motion and practical small-screen behavior.
- Measure bundle size, large repository/diff behavior and request/polling cost. Use route-level code splitting, bounded rendering, lazy expensive views and targeted state refresh; add a dependency only if the measured problem warrants it.
- Refine empty/failure/provisioning guidance, connection copy actions and admin confirmations. Keep private data/capabilities scoped to the active session.

**Tests/acceptance:** documented before/after evidence for actual bottlenecks and user tasks, no accessibility/security regression and no new framework/state system introduced for hypothetical scale. Polish does not retroactively excuse unsafe or nonfunctional earlier milestones.

### U20 — Final installed verification and handoff

**Primary files:** native/source/browser test entrypoints, packaging/configuration tests, operator/developer documentation, final coverage and implementation status.

- Execute the approved source suite, focused race tests, DOM/store tests, real browser workflows and packaging checks against the final revision. Keep all retained Cockpit, PAM, branding, console, provider-CLI and runner test source in the regression set.
- Exercise a clean first installation on a separately approved fresh target and an upgrade of controlled existing state. The recovered early VM installation is not evidence of a clean final installer. Verify static assets, dependency notices, migrations, secret permissions, callback origins, service identity and the fixed socket boundary.
- Repeat the core-owned U08 entrypoints/proof detail and implemented collaboration/admin journeys with real provider state. Consume P06/P11/P12/P13 host/operator/artifact observations by exact revision/bytes when available; do not create or count a second P product suite. Optional P09/P10 media needs its own delivery proof only if selected, not as a prerequisite for verifying the existing installation path. Separately authorize runner jobs, native maintenance, network changes, destructive test fixtures and reboots; missing permissions stay explicit, not silently passed.
- Build and verify x86_64 and aarch64 independently on matching native hardware. The unavailable sibling does not block useful work, but cross-compilation or browser-only tests do not establish its native runtime compatibility.
- Document supported/verified architectures, supported image profiles, remaining upstream-native screens, credential/session limits, trusted-team isolation assumptions, access lifecycle limits and operational instructions. Include honest current routing/SELinux/workload/persistence evidence.
- Record changes and checks by revision; preserve clean Git history and unrelated work. Publication, provider enrollment on other instances, physical installation and an OS/component updater remain separate authorization/scope decisions.

**Acceptance:** an operator can install/migrate the approved product and a developer can complete its documented workflows without hidden manual integration or fake data. Every held item is visible; neither a generated bundle nor a list of authored tests is called release readiness.

## 8. Conditional Soda extension milestones

These are concrete plans for discussed additions, **not approved baseline requirements**. Do not create dormant flags, tables or helper methods for them before selection. An accepted extension must be included in U17/U20 coverage and preserve all upstream/native boundaries.

### E01 — Rocky/Fedora creation-time image profiles

**Gate:** explicit OS-choice approval, a supported Fedora release/image candidate and U08's baseline native proof. **Files:** `project-os` recipes/rootfs, native build/staging scripts, `internal/host` config/create protocol, `internal/store` migration, environment UI/API, image-specific installed tests.

- Define a small operator-approved profile list with stable IDs, distro/release, native architecture and immutable resolved image identity. Select prepared Soda-compatible OCI images, not arbitrary browser-supplied registry URLs/flags.
- Reuse common project account/SSH/shared-tool behavior, with separate package/build steps where Rocky and Fedora differ. Test package names, Python packaging, systemd/OpenSSH, mise/CLIs, storage/cgroups/SELinux and nested workloads; a different `FROM` line is not compatibility proof.
- Add optional per-environment profile/creation-image metadata through an additive migration. Preserve the existing default for new requests that use the old contract, and inspect existing native image identity rather than pretending old projects were created from the current default.
- Allow profile selection only for creation. A stopped environment restarts unchanged; selecting Fedora never replaces a Rocky writable root or changes the host kernel. OS migration/update remains separate work.
- Expose each profile only for architectures with the required build/validation evidence. No unsupported profile appears selectable; one unavailable profile/architecture need not block the working baseline.

**Acceptance:** create separate approved Rocky and Fedora environments and pass the same account/SSH/shared-workload/persistence/isolation tests. Invalid profiles never reach arbitrary image execution. Existing environments keep their original identity/data after changing the default for future creations.

### E02 — Existing-environment lifecycle controls

**Gate:** explicit approval of allowed actors and supported start/stop/restart operations plus U08. **Files:** fixed host protocol/daemon, existing systemd unit integration, web/API authorization, environment UI and native lifecycle tests.

- Resolve the environment and actor from trusted state. Add only named operations targeting that existing labeled container/unit; never expose arbitrary unit names or Podman commands.
- Preserve rootfs/accounts/host keys/service data. Reuse normal native startup rather than container replacement. Clearly warn that stopping/restarting disconnects users and affects shared services.
- Report already-running/stopped states, timeouts and failures without optimistic success or automatic destructive repair. Handle native owner/association mismatches conservatively, not through automatic remapping.
- Keep operator host reboot/network/service administration outside these project controls.

**Acceptance:** approved owner/operator actions work, unauthorized/cross-project attempts fail, stopped projects remain stopped until an explicit/defined native start, and all persistent state survives. No delete/rebuild/auto-idle feature is smuggled in.

### E03 — Basic resource limits and usage

**Gate:** explicit resource-policy scope and native cgroup/storage investigation after U08. **Files:** host config/create/inspect boundary, environment metadata where necessary, project/admin views and native cap tests.

- Select a small set of native CPU, memory and PID limits and decide who may choose/change them. Apply validated bounds at the whole-environment boundary and prove how nested workloads inherit them.
- Expose bounded CPU/memory/process/disk-usage observations using native data, without returning full host/container internals. Distinguish usage reporting from enforceable limits.
- Investigate filesystem/storage support before promising disk quotas. Do not infer that a generic Podman size flag safely caps the current writable-root/nested-volume layout.
- Do not add a scheduler, admission/quota database, billing model, automatic shutdown/deletion or a resource-pressure repair controller.

**Acceptance:** invalid/unapproved values are rejected, effective native caps match requested policy, a busy environment does not bypass those caps via nesting, and existing projects are not silently resized or deleted. Any unsupported disk enforcement is explicitly omitted from the claim.

### Other gaps remain decisions, not hidden milestones

Key rotation/removal propagation, member offboarding, provider disablement/session termination coordination, organization-to-project-admin mapping, identity renames, multiple environments per repository, destructive lifecycle, image replacement and backup/restore each need their own scoped policy and preservation tests before implementation. Native Forgejo can still own its corresponding account/repository operations; Soda must not claim those automatically reconcile Linux access or live environment state.

## 9. Inventory coverage cross-reference

Use this table with the detailed [page inventory](dashboard-plan.md#page-inventory). It assigns work, not completion status.

| Inventory family | Owning milestones |
| --- | --- |
| Shell/help/error states, sign-in/session expiry | U02–U05, refined U19 |
| Projects, repository create/overview/files, environment summaries | U06/U07 |
| My profile/preferences, development keys, Git/GPG keys | U05; extended native account capabilities U16 |
| My work, global search, notifications, user profiles/activity | U13; advanced gaps U17 |
| Import/fork, file edit/upload/history/blame, commits, branches/tags/compare | U09 |
| Issue list/create/detail, comments/attachments, labels/milestones | U10 |
| Pull requests, conversations/checks, reviews and merges | U11 |
| Repository settings/access/protection/deploy keys/hooks | U12 |
| Organizations, teams, membership and scoped settings | U12; native Actions configuration U14 |
| Actions runs/jobs/workflows/dispatch, variables/secrets | U14; missing logs/artifact/control APIs resolved or explicitly held U17 |
| Releases/assets, wiki/revisions, packages/files | U15 |
| Forgejo administrator overview/People/account edit/instance orgs/repos | U05/U16 |
| Instance hooks/provider runners/jobs/native quotas/maintenance | U16 |
| Account security, applications/tokens, MFA/recovery, native site/auth configuration | U05 native entry points, U16 audit/implementation, U17 disposition |
| Issue boards, advanced graphs and unresolved upstream interactions | U17 with the relevant feature owner |
| Environment create/detail/members/join/connect/incomplete state/admin views | U07/U08; optional control/image/resource work E01–E03 |
| Destructive/ownership/account-remapping controls | Decision register; U17 must record their explicit exclusion or newly approved scope |
| Host administration/Tailnet/local runner capacity | Retained outside integration P11; preserve packaging U02/U18 and consume evidence in U20 |
| Default routes, old bookmarks and removal of HTMX | U18 |
| Responsive/accessibility/performance improvements | Baseline throughout, focused U19 |
| Complete artifact/data migration and installed architecture evidence | U02/U03/U08/U18/U20 |

## 10. Definition of done, execution and handoff

### Per-feature completion

A feature is **source-complete** only when its frontend, typed API, real provider/native caller, necessary persistence/configuration/build wiring and focused tests are connected. A mocked success path, disabled placeholder or manual Linux checklist is not an implemented product operation.

Track evidence separately:

1. **Planned** — this document and contracts only.
2. **Source-complete** — production callers and authored tests exist; no execution implied.
3. **Source-tested/built** — named revision and actual checks/artifacts recorded.
4. **Installed-verified** — named native target/client and real end-to-end outcomes recorded.
5. **Blocked/native fallback/deferred** — exact reason and impact retained; not counted as custom-screen parity.

Each milestone handoff records changed paths, authority/data effects, upstream contract evidence, tests authored versus executed, actual commands/targets, unresolved questions and the next milestone. Shared-file changes name their U/P owner and agreed interface; support evidence is linked by exact revision/artifact/target rather than duplicated or used as an independent product verdict. Keep coherent commits; do not amend unrelated history or sweep other work into them. No milestone is checked off by writing this plan.

### Verification matrix to maintain

- **Go/provider/API:** status/error contracts, source-backed native payloads, malformed input, pagination, timeouts, body limits, private/cross-user access and no credential fallback.
- **Auth/data:** OAuth state/PKCE, consent/scopes, refresh/logout concurrency, encrypted storage, upgrade/old-session behavior and no product-state loss.
- **React:** real routing, forms and keyboard interaction, loading/error/empty states, mutation results, stale-response/account isolation and safe content rendering.
- **Installed:** trusted TLS, actual Forgejo login/permissions, genuine repository/collaboration/admin changes, real environment account/SSH/shared-workload/persistence behavior.
- **Packaging:** frontend/backend match, complete local assets/licenses, missing-bundle failures, service UID/permissions, unchanged native operator integrations and clean first-install plus existing-state migration.
- **Architectures/profiles:** independently recorded native x86_64/aarch64 and each approved userspace profile; no inference from emulation or a sibling's result.

Only run the applicable checks when authorized. This planning request does not authorize dependency installation, compilation, product tests, live account/repository changes, provider runner jobs, routing/firewall changes, deployment, removal or reboot. Keep secrets and private inputs in restricted channels and ignored private locations; never include them in commits, argv, logs or screenshots. Preserve the existing VM backing image and all unrelated builder infrastructure.

**Current implementation:** continue U01's full action/dependency audit, verify/correct U02–U07 source and complete outstanding Markdown/download/pagination/auth/native tests before claiming the U08 workflow. Protected per-session grants and first-workflow API/React callers now exist in unbuilt source; they are not a completed migration or installed proof. Continue the full U01–U20 sequence rather than scaffolding fake inventory pages. See the handoff for unfinished work and execution gates.

# Unified Soda frontend and backend implementation plan

**Status: official Forgejo template overrides selected; U01 partly invalidated and its acceptance withdrawn as described below. Only bounded U08 is accepted (1/20 core milestones).** This revision incorporates the **179-action-group H01 source audit** committed at `c832901`. It changes remaining work, dependencies and acceptance criteria—not installed bytes, prior evidence or execution permissions. U09 remains incomplete. H01 source coverage is recorded. H02 source preparation is implemented at `c9a9be0`, with 16.0.3 locked only as a development candidate; the first read contract and native-auth/build integration designs have completed U01 source review. Native image/extensions, executed authentication feasibility, full artifact/license closure and deployment acceptance remain open.

## Selected direction — official template overrides; partial U01 invalidation

**Hard boundary: forking Forgejo is a failure path.** If meeting a requirement
appears to require maintaining modified Forgejo source or a downstream custom
executable, the chosen approach has failed and needs architectural reconsideration
with the user. Stop the affected approach; do not normalize it as a small patch,
bounded API extension or routine next milestone. This overrides every historical
patch/build permission or proposal below. Supported template overrides, extension
points, themes/assets, configuration, native workflows/protocols and existing
APIs/integrations are the intended lanes. Never bypass native security to avoid a
fork. Separately scoped upstream contributions are possible, but are not permission
to ship or depend on an unmerged downstream fork. No requirement is silently
removed: bring the concrete conflict back for a design decision.

**The user selected Forgejo's official template-override method.** Retain upstream
Forgejo's server-rendered pages, handlers, authentication and business rules;
customize the shell/navigation/presentation through supported template overrides
and assets. This replaces the requirement to recreate every Forgejo workflow in
client-rendered React using JSON APIs. Native-rendered Forgejo pages are now the
selected integration, not a forbidden fallback. No iframe, HTML scraping/fragment
injection, backend fork or source rebuild is selected by this decision.

Forgejo's inspected `modules/templates/base.go::AssetFS` layers
`<CustomPath>/templates/` ahead of built-in templates. The mechanism is upstream
provided; the exact shared templates, asset/script dependencies, configured paths,
reload/restart requirements and upgrade compatibility still need implementation
review. Supported overrides do not mean arbitrary template changes are safe or
that maintenance disappears. No live template/configuration change is authorized
or claimed here, and the installed Forgejo version is not changed.

**U01 is partially invalidated and its overall acceptance at `542de21` is
withdrawn.** The invalidated part is the selected frontend/integration architecture
and its conclusion that missing JSON interfaces require a source-built, patched
Forgejo. The derived mandatory headless-auth transport, read API patches and
U02 native Forgejo build sequence are superseded as the default implementation
path. Do not continue them merely because their drafts say “reviewed/accepted.”

**Retain:** the H01 source/API findings and workflow inventory, authority/security
findings (including the three unfixed Soda defects), licensing work and original
Apache-2.0 grant, source-preparer code and honestly scoped evidence. These do not
become false or need deleting; the preparer may simply be unnecessary for the
selected deployment. Missing JSON endpoints no longer automatically block features
that the native rendered interface already supplies. Read/parser findings remain
source findings, not a requirement to implement new APIs for this approach.

Soda still owns its persistent development-environment/access integration. Keep
its existing React/Go source while reviewing how that UI and native Forgejo pages
share branding/navigation and secure session/origin boundaries; no wholesale
rewrite or assumed shared authentication cookie is selected. Preserve native Git,
SSH/LFS/package protocols, CSRF/authorization, separate Cockpit operator authority,
working login and all four environments. Native rendering is allowed for Forgejo;
this is not a choice to add React SSR or a production Node service.

**Next:** reopen only U01's affected architecture/dependency/acceptance sections
around the selected override mechanism, then reconcile the U02–U20 implementation
assignments. Do not repeat H01, reopen the user's choice between templates and a
separate frontend, or silently replace the milestones with a new roadmap.

**Precedence:** the remaining detailed plan below records the previous headless/
React implementation sequence and retained evidence. Where it requires all-React,
no Forgejo HTML/SSR, new native auth/read APIs or a mandatory source-built Forgejo,
it is **superseded and not an execution instruction**. Those sections still need
coherent replanning; this decision record does not claim that work is complete.

**Goal:** a coherent Soda-branded product using native Forgejo rendering through
official template overrides, plus Soda's persistent development environments.
Preserve complete workflows, native authority, security, accessibility and data.

**Read with:** [page/dependency inventory](dashboard-plan.md), [single workflow register](forgejo-api-coverage.md), [architecture](architecture.md), [headless architecture review](forgejo-architecture-revision-plan.md), [headless implementation details](forgejo-headless-implementation-plan.md), [deferred scope](deferred.md), [actual evidence](implementation-status.md), [installation](installation.md) and [native validation](native-validation.md).

This is the leading **core product** plan: React, Go API/session/data integration, maintained Forgejo interfaces, production native environment/access mechanisms and product acceptance. The [native support plan](native-porting-plan.md) is subordinate under the [coordination contract](#coordination-with-native-support-porting). H01–H08 are work packages inside existing U owners, not another roadmap or milestone count. The [initial M01–M18 plan](implementation-plan.md) and earlier execution checklists are historical. Reuse their implementation and accurately scoped evidence; do not restart the codebase or inherit their PASS labels.

## Current execution snapshot — U08 completion run

This heading is retained for incoming evidence links. The current planning baseline is the post-H01 tree, not a request to resume U08 execution.

### What is actually built and installed

- Last recorded affected-component installation on retained `soda-test`: **`8b823db` dashboard, helper, runner companion and default new-project image**. React is preview at `/app/`; default routes remain HTMX. Forgejo's separate browser origin remains exposed. Neither Soda-only authentication nor U18 cutover has happened.
- Exact `8b823db` native build/seal/check passed: Go, 60 Cockpit tests, 21 dashboard tests, 30 Python build tests and 9 staging tests. A subsequent operator-probe correction brought Python build tests to 31 passing. These counts describe those revisions, not every later source change.
- Later U09/Soda-navigation source at `1d74a08`/`ad42223` passed local Go and TypeScript checks, 31 dashboard tests and dashboard builds; it was **not deployed**. H01 performed source/schema/document checks, not native builds or workflow tests.
- Four persistent environments remain. The fourth, `p7b41edaf83f10a6fd7e579bf`, was created from exact `c96c108` without rootfs/unit patches. Earlier roots retain their recorded identities and corrections; rebuilding the default image does not replace them.
- U08 combines `f233a4a` lifecycle evidence for explicitly unchanged mechanisms, `c96c108` fresh/different-UID exec proof and `8b823db` preserved rollout/regressions. It is not a final-image reboot, fresh appliance installation, headless-authentication proof, U20 acceptance or aarch64 evidence.
- The console welcome hook is missing; corrected `operator.sh` fails honestly. P11/U20 retain its delivery/interactive proof. Tailscale `NeedsLogin` and zero local runners are observations, not enrollment/job acceptance.

See the [U08 acceptance handoff](implementation-status.md#u08-accepted--bounded-native-x86_64-first-product-proof) and [reconciliation](#u08-closure-reconciliation--merged-candidate). The recorded infra-only route does not establish laptop/LAN/Tailnet reachability. Existing native workload starts after lifecycle events were explicit; automatic workload resurrection was not proved.

### Milestone status and remaining work

| Milestone | Reusable progress | Remaining acceptance responsibility |
| --- | --- | --- |
| **U01** | Retained H01 findings, authority/license work and source preparation | **Acceptance withdrawn:** revise affected architecture/dependencies for the selected official template overrides; prior headless/build requirements are not current execution instructions |
| U02 | Built React shell, real lockfile, static packaging and installed preview; verified Forgejo development source preparer | Complete assets/dependency notices/dev-route coverage; integrate and verify the reviewed native Forgejo source/patch build |
| U03 | Protected JSON, typed bounded provider errors, encrypted schema-v3 grants; migration/preservation subsets | Minimal extension compatibility, complete populated upgrade/refusal tests and coordinated config/asset/schema consumers |
| U04 | Real redirect OAuth and session-bound grants; refresh/security source | Native-backed Soda login/challenges/consent/logout; threat model, expiry/replay/concurrency/revocation and installed security proof |
| U05 | Profile/People/Git-key/development-key subsets and actual two-user onboarding | Complete self-account/security/key/application/email flows inside Soda; full admin/operator distinction with U16 |
| U06 | Repository discovery/create/tree/README/content subsets; native private creation | Complete template/init choices, permission/pagination/ref/content/download and safe-rendering matrix |
| U07 | Real create/explicit join/account-key provisioning/membership/connection subsets | Failure/forged-target/stopped cases; **browser terminal into the existing own workspace**, including design and native proof |
| **U08** | **Accepted in the bounded scope above** | No reopened U08 criterion; new terminal/headless work and final repetition belong to their owners and U20 |
| U09 | Connected history/ref/file/copy subsets; immutable comparison and race/error corrections locally checked | Native blame/net diff, remaining code/ref/copy/search-related contracts and complete native Git/write/fork/import proof |
| U10 | Issues/comments/labels/milestones/reaction/subscription subsets | Templates/timeline/assets/timers/dependencies, native lock/content history and **repo/org/personal issue boards** |
| U11 | Revision-bound review/merge and inspection subsets | Existing/outdated/old-side/team workflows, native resolve/viewed/merge-panel/range state and real Git/check/conflict proof |
| U12 | Settings/access/protections/hooks/organizations/teams subsets | Existing advanced form fields, remaining native settings/hook/invitation/client operations and linked-resource lifecycle handling |
| U13 | Work/search/notifications/profile subsets | Full native search/activity/graph coverage, visibility/paging/account races and real multi-user updates |
| U14 | Runs/tasks/dispatch/configuration subsets | Baseline-specific stock Actions reuse; workflow/steps/attempts/rerun/trust/configuration/runner gaps and approved real provider proof |
| U15 | Releases/assets subset | Release clear semantics; complete wiki/history/search and package metadata/settings/protocol-backed views |
| U16 | Initial People capability and source audit | Complete site administration, native account/MFA detail/reset, auth sources/config/maintenance/moderation and authority tests |
| U17 | Single source register and supplemental navigation guard | Reconcile every action's native/adapter/browser evidence, update/rebase compatibility and full Soda-only candidate coverage |
| U18 | Not performed | Approved preserved SPA/default-route/ingress cutover and coherent legacy removal |
| U19 | Basic usability/error handling | Measured real-task accessibility, navigation, responsive and performance improvements |
| U20 | Earlier scoped evidence only | Final matching-revision source/browser/native regressions, fresh install/populated upgrade, full operator/provider/terminal and independent native architectures |

### Next execution and decisions

1. **U01 review is complete:** implement the reviewed source/build/read/auth boundaries below, reusing H01 and the existing preparer. Do not restart the audit or ask for replacement Forgejo policy decisions. Engineering selection is not approval to upgrade the retained installation.
2. **Start U04/U05/U16 authentication design now**, alongside U02's source-build and U03's compatibility design. Establish native challenge/consent/security feasibility before broad UI expansion. Do not wait until U16 to discover that ordinary grants cannot complete onboarding.
3. After the relevant reviews, implement the build/compatibility spine and first U09 blame/net-diff vertical slice, while proving the reviewed authentication path. Continue independent stock-API/UI fixes without duplicating native features. See the [execution order](#6-milestone-map-and-execution-order).
4. Complete feature-owned action batches, including early administrator work; verify the first patch update/rebase procedure before allowing a large unmaintained patch set to accumulate. U17 reconciles delivered work, not a late implementation bucket.
5. Only after complete candidate coverage and rehearsal: separately approved U18 cutover, U19 measured improvement and U20 final installed proof. Select exact provider/fixture/import/rollout and native aarch64/fresh-install scopes when needed.

**Preserve:** the VM, all four environments, accounts, repositories, memberships, public/private key inputs, installed tools, dirty checkouts, workloads/volumes, backups and failed evidence. Previous fixture/reboot approvals were used. No fifth environment, reset/reseed, replacement root, provider job/enrollment, host-network change, reboot or cleanup is implied. Never reboot infra. Old pre-change backups are not lossless rollback after later writes.

**Not selected:** E01–E03 and optional ISO/QCOW2/media. There is no installable SodaOS ISO; media is not a core prerequisite. Local builds and automated tests are already authorized within the recorded development scope; this documentation revision neither revokes that permission nor authorizes installation or live provider mutations.

## Architecture revision under review — first-class headless integration

H01 establishes that stock REST coverage is insufficient, but also that much useful native API and Soda code already exists. The implementation strategy is **suitable stock interfaces plus narrowly reviewed Forgejo-side additions sharing native functionality**, consumed by explicit Go adapters and React views. It is not a replacement forge, generic proxy or codebase rewrite.

The audit's decisive changes to this plan are:

- **Authentication is an early architectural gate.** Native middleware rejects forced-password-change and required-but-missing-MFA accounts. WebAuthn JSON is session/origin-bound; Basic rejects security-key users. Password-to-token, borrowed-cookie and administrator-token shortcuts cannot satisfy this requirement.
- **Version choice matters, but is not a solution by itself.** Inspected v16.0.3 adds human Actions jobs/logs/artifacts/cancel APIs. It still lacks blame/net comparison and complete headless authentication, review/board/admin and Actions semantics. Its targeted audit is not support/security/migration qualification.
- **A missing screen is not always a missing endpoint.** Reuse reviews/replies, issue templates/assets/timers/dependencies, file/ref/copy/protection/team/release/package/quota APIs. Advanced protection and team-policy fields already have Soda adapters. Byte authorization, read scopes and partial-update translation are adapter work.
- **Authority/semantics gaps need native changes, not Soda elevation.** Examples include repository Actions owner-only REST versus admin-capable web settings, webhook secret/events and release-note clearing. A JSON handler, Swagger entry, version response or cleanly applying patch proves none of these contracts.
- **Coverage is broader than code and collaboration.** Native authentication/security, boards, review state, activity/graphs, settings/invitations/hooks, wiki history, packages and site administration have named feature owners below.

Use the register's action classes to choose the work: **1** suitable stock API → ordinary typed client/UI; **2** existing interface → bounded adapter/composition; **3** missing native semantics/authority/interface → reviewed native addition/extension; **4** existing Soda adapter fields → missing UI controls only. Mixed rows require their specific sub-actions, not a blanket “whole family needs a patch” decision.

The [headless work package](forgejo-headless-implementation-plan.md) supplies H02–H08 implementation detail. This plan determines product order and acceptance. No concrete patch, wire protocol, dependency upgrade, maintainer or live configuration change is selected merely by writing either document.

## 1. Governing decisions

1. **Forgejo owns identity/password verification and storage, permissions, Git, repositories, collaboration, CI and administration.** Shared native services—not copied rules—must back both its existing web callers and new APIs. Soda never reads the provider database/filesystem directly or calls a privileged internal/runner protocol as a human API.
2. **Soda owns presentation and its extension.** Keep additional preferences, development-access public keys, environment associations/memberships and secure browser sessions. No second password/role inventory, clone/index backend, scheduler or secret registry.
3. **Retain the chosen stack:** TypeScript/React, PatternFly, Vite+, Zustand, browser routing and `fetch`; Go/`net/http`, SQLite and the bounded native helper. No SSR, Node production service, Tailwind, TanStack, Rust introduction or replacement framework.
4. **Every Forgejo workflow stays in Soda**, including sign-in, first-password change, MFA/security keys, recovery, consent, mail-linked account actions and administration. Preserve configured upstream IdP flows and native callback validation; integration/origin conflicts require a concrete adapter/native fix, not a new Soda authentication policy or a Forgejo-page fallback. Ordinary Git/SSH/LFS/package protocols and operator Cockpit remain separate selected boundaries.
5. **Native actors are distinct:** repository owner/admin/writer/reader, organization owner/team member, site administrator, Soda operator, project administrator and host root. Scopes are not roles. Resolve native resources and authorize each operation server-side; no Sudo/impersonation/operator-token fallback.
6. **Keep installed login working until its replacement passes.** Existing OAuth redirects, HTMX pages and direct Forgejo browser ingress are transitional gaps, not final exceptions. Do not close the origin first and strand users or enrolled security keys.
7. **Preserve native project integration.** One Rocky environment per eligible human-owned repository, explicit join including the creator, existing project-local account/home and normal existing-container startup. Native Git authorization remains separate from joining.
8. **Follow native authority; preserve Linux state.** Forgejo owns rename/transfer/delete/security rules. Soda must follow current native ownership/authorization, not preserve stale creator authority or offer competing succession policies. Keep distinct native admin roles and legitimate Soda memberships; these actions do not automatically remap Linux users, delete environments or revoke SSH sessions. General reconciliation/offboarding remains deferred.
9. **Conditional upstream functionality remains in scope when enabled.** Do not disable registration, mail, quotas, federation, moderation, hooks or repository units to shrink the frontend requirement. A truthful native-disabled state is not a missing-interface waiver.
10. **Evidence is specific.** Source tests, exact native builds and installed workflows are different. Neither a mock, schema/metadata advertisement, screenshot nor old sibling-architecture result certifies current behavior.

## 2. Baseline and changes actually needed

Use the post-audit source and [handoff](implementation-status.md), not the original pre-React assumptions. Recheck HEAD and unrelated working-tree changes before each batch.

| Existing source | Retain | Remaining change |
| --- | --- | --- |
| `dashboard/src/`, manifest/lockfile and Vite config | Connected flat feature files, PatternFly shell, session/race guards, safe Markdown and focused tests | Complete workflows and controls; split files only for real responsibilities, not a forced folder rewrite |
| `internal/web/`, `cmd/soda-dashboard/` | Explicit protected JSON routes/DTOs, unprivileged server and static preview | Native auth/feature adapters, bounded bytes, complete errors/read scopes; remove replaced legacy paths in U18 |
| `internal/forgejo/` | Acting-user stock clients, sanitized complete bounded responses, OAuth exchange/refresh | Reviewed feature contracts, minimal compatibility and native-backed missing operations |
| `internal/store/` | Stable identities/associations, ordered migrations, hashed sessions, schema-v3 authenticated-encrypted grants | Only required additive Soda metadata and proven upgrade/refusal/concurrency behavior; no copied provider inventory |
| `internal/config/`, setup/activation | Explicit origins, restricted credential files, strict parsing and legitimate bootstrap | Coordinated auth/build/rollout contracts across every consumer, including `soda-runners` |
| `internal/host/`, `project-os/`, `soda-project@.service` | Fixed create/inspect/account boundary, persistent roots, shared tools and validated U08 mechanisms | Remaining U07 failures and reviewed own-workspace terminal; concrete native corrections only |
| Forgejo service/build input | Standalone container, native data/runtime/protocols; currently pulled 15.0.7 image | Reviewed source/patch build feeding existing artifact consumers; no live change yet |
| `cockpit/`, canonical `assets/`, project CLIs | Separate privileged bridge, Tailnet/Runners backing logic/tests, artwork/licenses and native tools | Preserve packaging and operator compatibility; no dashboard takeover |
| `scripts/`, `appliance/`, `tests/` | Existing build/check/stage/install and core-owned installed entrypoints | Extend one delivery/test path for Forgejo contracts, preservation and final browser boundary |

Do not recreate already implemented grants/migrations or call the current personal-only fork/basic import form full native parity. Present source coverage and remaining native work separately.

## 3. Scope and decision register

### Included in the core U milestones

All 179 audited action groups and their sub-actions, Soda environment create/join/access integration, the requested browser workspace terminal, complete packaging/migration/security coverage, preserved cutover and final native proof. The [single register](forgejo-api-coverage.md) owns action evidence; [section 9](#9-inventory-coverage-cross-reference) assigns implementation responsibility without duplicating endpoint inventories.

### Decisions with safe defaults

| Decision | Required review / safe behavior while open | Owner and timing |
| --- | --- | --- |
| Supported Forgejo source baseline | Review support/security horizon, release/migration notes, native build/runtime dependencies and v16 stock reuse versus maintained backports. Keep installed 15.0.7 untouched; do not invent a lock from research hashes. | U01/U02/U03 before H02 implementation |
| Patch/security/update ownership | Forgejo owns upstream security rules/fixes. Soda maintains its own adapters/carried patches, consumes upstream fixes and verifies compatibility/source/notices/rebase/retirement. Record concrete review responsibility for that work, not a new owner of Forgejo policy or an invented blocker asking the user to take it over. No automatic updater or unsupported freeze. | U01/U02 with feature owners before adopting patches; U17 verifies H07 |
| Concrete native contracts | Specify actor/resource binding, scope, request/response fields, omission/clear semantics, errors, paging, native work/time/output/concurrency/cancellation limits and shared callers. Review API/web authority discrepancies individually. | Owning U feature with U03/U04 before each addition |
| Authentication/challenge/consent feasibility | Review the full native sequence, pre-authentication authority, abuse protections, grant/logout/revocation and sensitive reauthentication. Keep current login until isolated replacement proof. | U04/U05/U16 early H05 gate |
| WebAuthn, mail and external IdP origins | Resolve RP-ID/origin/enrolled-key compatibility, native-generated links/callbacks and external interactions against the Soda-only requirement. Do not reset keys, change domains or silently allow a frontend escape. | U04/U05/U16 with U18 ingress owner, before auth implementation/cutover |
| Consent and current sessions | Minimum per-operation read/write/category scopes, including package and distinct administrator consent; explicit reauthentication for missing grants. Never mint user authority from Soda rows. | U03/U04; features supply exact requirements |
| Linked identity/resource lifecycle | Specify native rename/transfer/archive/delete/disable/moderation and account-security effects on existing Soda associations, sessions and project access; preserve roots and avoid stale-owner/name-reuse escalation. Native workflows remain required, not silently excluded. | U05/U07/U12/U16 with U03/U04 before affected actions are enabled |
| Browser workspace terminal | Select one transport/native account-binding design, terminal lifetime, disconnect/expiry/revocation behavior, output/input bounds and stopped/unavailable behavior. No implicit create/join/start, host shell or shared-root login. | **U07** with U02/U03/U04 security/build review; U17/U20 coverage |
| New packages | Keep current manifest/lock pins. Add only researched, licensed dependencies with real callers; Forgejo build tools follow its own selected source contract. | U01/U02 and requesting feature |
| E01–E03, org-owned/multiple environments, generalized access lifecycle/recovery | Keep current uniqueness/owner/runtime rules; no dormant schemas/helpers or implicit selection. Native provider quotas are not Soda environment quotas. | Separate scoped product decision; conditional tracks below |

A decision blocks its affected contract or acceptance, not every independent stock-API fix. Record the concrete unresolved choice and reviewer; do not replace decisions with invented defaults, a generic policy engine or a gap waiver. Measured bounds must make the real workflow usable; today's subset byte caps are not a permanent parity exemption.

**Terminal placement:** U07 owns the requested terminal into a signed-in user's **existing project-local account/home** in the selected persistent environment. It is not U09, a browser IDE, a new per-user container or an E02 lifecycle control. This plan assigns design/implementation/testing responsibility; it does not select the transport or authorize native shell execution. U08 remains accepted for its earlier scope.

Not added: host UI replacement, forge/CI engine, tool/service marketplace, managed private toolchain branches, project DNS/custom SSH gateway, unrestricted host Podman access, new VM backend, generalized recovery/deletion, installer media or an update platform. See [deferred scope](deferred.md).

## 4. Architecture, source ownership and dependencies

```text
React browser application (Soda origin; no provider credentials)
                         |
                    Go dashboard
           /             |                 \
 explicit Forgejo     Soda SQLite       bounded native socket
 interfaces                              |
           |                         host helper
 Forgejo-owned auth,                     |
 services and state              existing project/account
```

The required native additions live **inside Forgejo**, sharing its authorization/computation with existing web callers before rendering. Do not serialize complete HTML contexts, expose arbitrary internal methods, add a privileged sidecar or let Soda access native storage.

### Concrete source and build ownership

Retain `dashboard/src/`, `internal/{web,forgejo,store,host}/` and existing tests. The [headless work package](forgejo-headless-implementation-plan.md#1-concrete-source-and-delivery-arrangement) distinguishes current source preparation from remaining native build/interface work:

- U01/U02: `appliance/forgejo/README.md` and sole authoritative `source.lock.json` now exist for a verified 16.0.3 **development candidate**, not deployment selection. Ordered native patches, `Containerfile`, full build input identities, notices/corresponding source and upgrade/retirement procedure remain pending.
- U02: `tools/soda-forgejo-source/` and `internal/forgejobuild/` implement bounded retrieval/extraction/patch/receipt preparation, with Go tests in the existing aggregate. `scripts/build-forgejo.sh` and its **existing** `scripts/build-native.sh` integration remain unimplemented. Extend tests with actual build-boundary failures; do not add a duplicate Python source preparer. Native feature tests must accompany future Forgejo patches; current Soda tests are not those native tests.
- U02/U03 and artifact owners: retain `forgejo.iid`, `images/forgejo.oci`, staging, sealing and install consumers. Replace the reviewed Forgejo input only; Caddy and project roots are not incidentally upgraded.
- U03/U04: minimal explicit extension compatibility in the existing client/config/session boundary; the owning account/repository/collaboration/admin features supply typed native operations and corresponding Soda callers/tests.
- Full downloaded source, generated assets and build/test output go in fresh ignored `.artifacts/` attempts, not a copied tracked upstream tree or overwritten accepted stage.

Forgejo's actual build includes native Go and frontend/embedded assets, migrations and runtime/entrypoint/Git requirements. Its build pins may differ from Soda's. Root GPL/GPL-3.0-or-later source obligations must be reviewed; the Swagger MIT license is not the distribution license. Preserve required source and notices, rather than assuming an equivalent Go-only binary.

### Dependency work

Manifests/lockfiles/image recipes—not this plan—own exact versions. Complete their transitive/license/native-architecture review in U01/U02. Existing React routing, Markdown and Go OAuth implementations are reusable; neither an OAuth library, diff renderer nor terminal emulator package is automatically selected. Add a small dependency only after a concrete caller and compatibility/security/license review. Keep separate dashboard/Cockpit builds; Node/pnpm remain build/development tools. No generic frontend resource/cache framework or speculative directory reorganization.

### Coordination with native support porting

Production `internal/host/`, `project-os/` **and Forgejo integration** are core, not outside infrastructure merely because they run on Linux.

| Shared area / contract | Core owner | Outside support owner and limit |
| --- | --- | --- |
| React, Go routes/DTOs, native provider authority, auth/session/CSRF and Soda schema | U01–U19 as assigned | Consume existing app/tests; no alternate handlers, password/role model or provider database access |
| Forgejo baseline, patches, native feature tests and source-build semantics | U01/U02/U03 plus owning U04–U16 features | P04 inspects/bundles specified identities; no P-owned fork, API implementation, update mechanism or conformance verdict |
| Project helper/accounts/keys/tools/runtime/persistent roots and own-workspace terminal | U07; U08 retained proof/U20 final integration; approved E tracks only | P02/P03 supply VM/SSH/observation primitives, not account/terminal policy or an alternate runtime |
| Dashboard/Forgejo payload and build order | U02 | P04 packages specified output; no second build or fake missing-asset substitute |
| Config, secret/key provisioning, bootstrap, both owners' migrations and cutover | U03/U04/U18; Forgejo owns its own migrations | P05 transports approved inputs/steps; no copied bootstrap logic or direct database edits |
| Shared `scripts/`, `appliance/`, identity/staging/install checks | U02/U03/U04/U18 own application semantics | P04/P05 own outer integrity/provisioning transport; agree changed input/output contracts first |
| Core `tests/installed/` scenarios and fixtures | Owning U feature, integrated U08/U17/U20 | Invoke exact entrypoints and retain results; no copied browser/product suite in `internal/acceptance/` |
| Host substrate, Cockpit/PAM/Tailnet/local runners, console/branding/CLIs | Preserve U02/U18; final consumption U20 | P06 observations/P11 retained integrations, not Forgejo Actions/admin or developer workspace UI |
| Exact-candidate/architecture evidence | U08/U17/U20 own relevant product assertions | P12/P13 hand off evidence, not independent product readiness |
| Optional ISO/QCOW2 | Not a core prerequisite | Conditional P09/P10 only after explicit selection |

Name the owning milestone, affected paths and caller contract before shared-file work. Retain exact Soda revision, Forgejo source/patch/image identity, architecture, target/client and private input references. Support results are reusable observations, not another product authority.

Former P07/P08 product/workload/persistence scenarios belong solely to the [core proof detail](#core-owned-native-proof-detail) and U20. Core work can use existing authorized entrypoints without finishing the P plan; optional media/P12 are not readiness barriers. A transport/artifact defect goes to P, a provider/auth/schema/runtime/terminal defect to its U owner. Cross-boundary corrections update both callers, without weakening acceptance or relabeling old results.

## 5. Shared API, state and migration contracts

### HTTP and route boundary

- Explicit `/api/...` routes and typed operations, not an arbitrary URL/method/header/token proxy. `docs/dashboard-api.md` records implemented Soda contracts; the audit records actual native interfaces. Proposed headless auth/extension/terminal routes need review before they are asserted to exist.
- Go owns secure browser sessions/request protection. Current `/login` and `/oauth/callback` keep working during transition, but **permanent Forgejo redirects are not the target contract**. U04 provides Soda-owned login/challenge/consent presentation; U18 changes installed origins/routes only after proof.
- Require bounded typed bodies/content types and CSRF/origin protection on every unsafe method, including authentication entrypoints as appropriate before a normal session. API expiry is JSON 401, not HTML in a successful fetch; users reauthenticate inside Soda once U04 is implemented.
- Browser IDs are lossless canonical strings for native int64 identities. Refs/paths preserve slash/space/Unicode semantics, are bound to authorized repositories and cannot become host paths. Native result identity must match the requested operation.
- Distinguish unauthenticated, forbidden, hidden/missing, native-disabled, conflict, validation, oversized, malformed, incompatible interface, unavailable and uncertain mutation outcomes. No successful empty fallback, raw provider errors, inferred permissions or automatic replay after ambiguous writes/refreshes.
- Safe reads need minimum native read scopes; mutations require actual write/category scopes and native actor/resource gates. Repository deletion, package operations and administrator actions do not inherit repository-write authority. UI hints never authorize the operation.
- Native additions specify real computation/input/time/output/concurrency budgets and cancellation/reaping, not just HTTP response size. Blame pagination cannot bound whole-file computation by itself. Binary/large/limited results must be useful and truthful, never a Forgejo-page escape.
- Authorized attachment/avatar/raw/archive/log/artifact/package bytes use reviewed same-origin adapters or native protocols as appropriate. No arbitrary redirect credential forwarding, active untrusted HTML on the authenticated origin or browser navigation to a native frontend. Audit provider/mail/Markdown/error URLs as well as explicit TSX links.

### Minimal extension compatibility — H03

Keep required contract revisions beside concrete feature clients, using a small read-only native interface description reviewed with the first additions. No generic discovery/plugin system, copied permissions, durable capability table, permanent stock/patched selector or arbitrary fallback dispatch.

Metadata is neither authority nor execution proof. Reject malformed/missing/breaking contracts for the affected operation; distinguish access denial and network failure. Define finite freshness/invalidation if cached. A missing blame contract must not disable unrelated working stock reads or environment access; graceful development unavailability still blocks final coverage. Resolve pre-authentication compatibility/bootstrap with U04 so checking compatibility does not itself require an unavailable ordinary grant. Test exact built behavior, not only `/version` or advertised feature names.

### Zustand and frontend state

Keep a small cookie/CSRF/error-aware fetch client and explicit feature-owned stores. Navigable refs/filters/pages belong in URLs; drafts stay local to their actor/target. No tokens/passwords/private content in browser persistence. Abort superseded reads and reject late route/session/account results, including uploads, downloads and terminal traffic. Logout clears private state; refresh must not resurrect it. Mutations report native completion or uncertainty and reload only affected state. Keyboard/focus/error/empty/loading behavior is required with the feature, not deferred to U19.

### Data, credential and deployment preservation

- Retain ordered Soda migrations and schema-v3 AES-GCM session/provider-bound grants with a restricted external key. Wrong/missing keys fail closed without regeneration/plaintext fallback. Logout wins refresh races; old sessions without grants reauthenticate without losing product records.
- Forgejo owns any native auth/challenge/session data and its migrations. Do not add a Soda password, provider-role or permission database to complete headless auth. Review native grant/session revocation separately from deleting Soda's credential copy and from Linux/SSH access.
- Preserve stable IDs, Soda-only profile values, keys, memberships and environment associations; only add metadata needed by an actual extension operation. No generic jobs/reconciliation, image selector or inferred creation-image history.
- Test populated old/current/newer-schema refusal, migration failure, repeat startup and irreversible changes separately for Soda and Forgejo. Back up consistently through each owner's supported mechanism, with matching restricted config/keys/artifacts—not a blind live-WAL file copy or direct provider edits.
- Validate assets/config/compatibility at the appropriate pre-mutation boundaries. Coordinate all strict config consumers, including dashboard/setup/runner companions, instead of loosening parsing or inventing duplicate configs.
- Rehearse exact rollout ordering and supported mixed-version behavior on restricted copies. Old binaries/backups are not automatic lossless rollback after new writes. No first-install/bootstrap replay, replaced project roots or broad service activation as a dashboard upgrade shortcut.

## 6. Milestone map and execution order

**Numbers identify responsibility, not a requirement to accept U01 through U20 serially.** A prerequisite below means the named reviewed contract/working deliverable, not completion of every case in the referenced milestone. This permits early authentication/admin work and avoids making U04 wait for a complete U16 or U17 wait for an already completed U18. Only U08's historically bounded acceptance is retained; broader uncompleted milestones do not inherit it.

| ID | Milestone | Required inputs | Main remaining outcome |
| --- | --- | --- | --- |
| U01 | Audit, baseline and maintainership decisions | Recorded H01 and existing source | Reviewed contract/security/lifecycle/build decisions with concrete feature owners |
| U02 | React assets and native Forgejo build spine | U01 baseline/build/license decisions | One exact source/patch-to-image path plus complete SPA packaging |
| U03 | API, compatibility and migration foundation | U01 contracts; U02 artifact/config inputs | Safe typed APIs, H03 compatibility and preservation/refusal behavior |
| U04 | Native-backed Soda authentication and grants | Early U01/H05 security review; U02/U03 native interfaces | Full Soda login/challenge/consent/session lifecycle, not just redirect OAuth |
| U05 | Self-account/security/keys and onboarding | U04 challenge/reauth contracts; relevant stock/native account APIs | Complete account/security flows and initial People/onboarding integration |
| U06 | Repository discovery/create/content | U03/U04 acting-user contract | Complete native repository basics/content and template/init choices |
| U07 | Environments, direct connection and workspace terminal | U04/U05 identity/access, U06 repository selection; reviewed terminal design | Real provisioning/failure coverage and existing-user terminal |
| U08 | First installed product proof — **accepted** | Historical U02–U07 deliverables and scoped native execution | Retained bounded x86_64 result; no new dependency on later terminal/headless work |
| U09 | Code history/comparison/native writes and copies | U06; U02/U03 first native contracts; early H05 feasibility review | Blame/net diff first, then complete audited code/ref/copy/native Git workflows |
| U10 | Issues, time/dependencies and boards | U06; relevant native/template/content contracts | Complete native issue collaboration and issue-project boards |
| U11 | Pull requests, reviews and merge state | U09 diff/ref and U10 conversation contracts | Complete native review/resolve/viewed/merge controls and proof |
| U12 | Repository/org/team/hook administration | U06/ref contracts; lifecycle and native authority review | Full scoped settings/invitations/hooks and advanced existing forms |
| U13 | Work/search/notifications/profiles/activity | Stable resource routing; native search/activity contracts | Complete permission-filtered cross-resource navigation/graphs |
| U14 | Actions and provider configuration | Selected baseline; U11 checks/trust and U12 scoped authority contracts | Stock human APIs plus missing workflow/run/configuration semantics |
| U15 | Releases/wiki/packages | U09 refs/content; native package scopes/protocols | Complete native artifact/documentation/settings workflows |
| U16 | Complete Forgejo site administration | Early U04/U05 security/People contracts; shared U12/U14/U15 operations as needed | Admin native interfaces/views, redaction and distinct-actor proof |
| U17 | Coverage, update compatibility and browser closure | Delivered U02–U07/U09–U16 actions; H07 update demonstration | Complete candidate coverage/rehearsed ingress, not a late gap backlog |
| U18 | Default SPA cutover and legacy removal | Accepted U08; U17 candidate proof; current backups/rollout scope | Preserved live default-route/auth/ingress transition |
| U19 | Make it good | Working complete workflows; U18 default integration for final review | Measured usability/accessibility/performance improvement |
| U20 | Final native verification and handoff | U17–U19 candidate; exact fresh/upgrade/architecture/provider scopes | Final revision/architecture-specific complete product proof |

**Execution from the current tree:**

1. **Use the accepted U01 review:** the baseline/build/license, first read and native-auth/ownership integration dispositions below are ready for implementation. U07 terminal design remains separate. Review actual patch diffs and new feature-specific contracts, not a second inventory, authentication proposal or source preparer.
2. **Build the shared native boundary:** U02/H02 source/patch build and U03/H03 compatibility/migration tests. Preserve usable stock APIs. Implement U09/H04 blame/net comparison as the first full read slice; prove the reviewed authentication path in parallel, not after all screens.
3. **Finish early account/environment integration:** U04/U05 end-to-end onboarding/security, U06 content and U07 error/terminal work. Begin U16 security-sensitive contracts/People detail now; it is not a post-collaboration afterthought. U08 is not replayed.
4. **Deliver feature-owned batches:** remaining U09 and U10–U16, following their concrete shared contracts. Stock-API forms/adapters can progress independently: for example advanced protection/team fields, read-scope fixes and authorized attachment bytes. Native additions follow native tests → API → Soda adapter → React, each with bounded errors/authority proof.
5. **Verify maintenance early and repeatedly:** H07 begins with the first native slice/update candidate and grows with every patch. U17 keeps the single register current, then closes complete candidate workflows and end-state ingress rehearsal. It does not implement everyone's postponed gaps.
6. **Deliver safely:** U18 approved preserved cutover, U19 measured whole-product review, U20 final fresh/upgrade/native-architecture/operator proof. Basic accessibility/security and measured native resource limits run throughout, not only here.

H01 maps to U01/U17; H02 to U01/U02/U03; H03 to U03/U04/U17; H04 to U09; H05 to U04/U05/U16; H06 to remaining feature-owned work; H07 to U01/U02/U03/U17/U20; H08 to U17/U18/U20 and feature owners. These labels add no milestone count or P-owned gate. Unselected E tracks do not block core delivery.

## 7. Detailed core milestones

Every feature below must close its assigned [audit actions](#9-inventory-coverage-cross-reference), not only its named examples. Required evidence has three layers: **N** native semantics/authority and real outcome, **A** typed bounded Soda adapter/security tests, **B** browser interaction/navigation/actor isolation. H01 names test starting points; it did not execute them.

### U01 — Capability, authority and baseline audit

**Files:** single coverage register, this plan/headless review, dependency manifests/locks, native service/build/config source.

**Execution rule: close existing work; do not restart R&D.** H01 at `c832901`, source preparation at `c9a9be0` and the review drafts at `2cf9127` are reusable inputs, not tasks to repeat. This checklist replaces broad instructions to audit/research/design these subjects again. The single 179-group register remains authoritative; no second inventory or decision-tracking system.

#### Completed inputs — reuse without repeating

- [H01 coverage register](forgejo-api-coverage.md): 179 action groups, selected-source authority/configuration/API gaps, v15/v16 comparison and U ownership. Do not enumerate the routes or rediscover the missing APIs again.
- [Forgejo source guide](../appliance/forgejo/README.md): support-horizon comparison, recommendation to reuse v16's human Actions APIs, independently verified development source lock, working preparer and passing preparation tests. Do not recreate the lock/preparer or repeat archive retrieval as a U01 gate. Development selection is not deployment approval.
- [Authentication delegation contract](forgejo-authentication-design.md): reviewed exact native helper/session/gate/consent/logout and ownership lookup boundaries, transport bindings, threat model and implementation tests. No replacement policy or new proposal is required.
- [Read-contract draft](forgejo-read-contracts.md): blame/net-diff semantics, immutable identity and authority requirements, parser/cancellation findings and required tests. Complete the missing fields/budgets here; do not rediscover the same native limitations.

#### Closure checklist — completed review and implementation handoff

| Step | Review disposition | Implementation responsibility, not U01 proof |
| --- | --- | --- |
| 1. Baseline | Accept the existing locked 16.0.3 engineering input and supported-stable direction; reuse completed release/migration/support review. Installed 15.0.7 stays untouched. | U02 resolves real build/platform pins; U03/U17/U20 verify migration/compatibility and refresh stale support facts before adoption. |
| 2. Build/license | Accept the source guide's native recipe/input/identity/source-notice/refusal contract. Root Apache-2.0 LICENSE/NOTICE cover original Soda only. Exact PatternFly CSS license, 33 font-source bindings and 13 named Go-module notices reviewed; full artifact/inherited-rights closure is not claimed. | U02 implements the single build/delivery path and fails missing source/notices or unresolved rights; native/embedded/transitive compliance must match actual shipped bytes. Soda owns its integration/patch review, Forgejo upstream owns its rules/fixes. |
| 3. Authentication | Accept shared native helper/gate extraction plus a client/browser-bound native session transport; retain native policy/storage/rotation, no bearer/cookie/admin substitution. Exact current consent/logout effects and required concurrency/security tests are recorded in H05. | U04 owns native carrier/operation DTOs/session/security implementation; U05/U16 own native account/admin operations and tests. No advertised auth capability until native proof exists. |
| 4. First read/compatibility | Accept revision-1 routes/DTOs/bytes/modes/predicates/errors/budgets and bounded descriptor contract. Require shared raw-byte/path/parser corrections; preserve native five-second cancellation grace in a corrected 15-second operation budget. | U02 packaged limits, U03 descriptor/consumer, U09 parser/endpoint/adapter/UI/native tests. Source review is not parser, cancellation or packaged-runtime proof. |
| 5. Associations | Accept current native stable-ID ownership checks, including the stock organization `is_owner` result—not `is_admin` or a copied role hierarchy. Preserve Linux accounts/data and separate existing Linux access from browser authorization. | U07 with U04/U05/U12/U16 removes stale-owner reliance and implements lifecycle handling/tests. The three explicit source defects below remain unfixed. |
| 6. Closure | Existing 179 groups remain assigned once; first contracts and build/auth review dispositions have concrete owners and test handoffs. No further U01 policy decision or broad audit is required. | Actual native patches, feature-specific contracts, full builds/compliance and installed acceptance stay with the existing implementation milestones. |

#### Closure results — execution after `d5b5065`

Historical delta results; the completed checklist and current U01 disposition
below supersede then-open questions and absent-license observations.

- **Release-note gap closed:** complete 16.0.0–16.0.3 notes recovered/length-and-blob verified and read; concrete upgrade constraints recorded in the existing source guide. Retain 16.0.3 as the single implementation input following the stable line. No repeat retrieval/API comparison is needed; deployment and human support commitment remain unapproved.
- **License/build disposition advanced:** three missing npm license declarations resolved from exact integrity-verified tarballs; direct frontend metadata and upstream notice-generation behavior inspected; corresponding-source/notice delivery contract recorded. The tolerated Go-license collection failures, unbound generated notices, missing PatternFly notice/font coverage and absent SodaOS top-level license are explicit—not a blanket license PASS. Actual tagged/native closure matching and artifact assembly remain U02; unresolved compatibility must be reviewed before distribution.
- **First read contract remains a candidate; auth overreach corrected:** retain the read contract/resource/compatibility work. The H05 draft's mandatory new auth state machine/schema and invented account-attempt thresholds are withdrawn as selected requirements; extend native operations instead. A bounded local Git/worker experiment supports the Linux limit-launcher primitive; it does not prove Forgejo, packaged Alpine, worst-case behavior or aarch64. Native security review, implementation and contract tests are still required, not another proposal or inventory.
- **Origin policy resolved at design level:** exact browser origins are per-installation operator inputs; retain an existing native RP-ID and reject an incompatible origin change rather than reset keys. A single global production hostname is not a U01 prerequisite. Exact retained-target values/credentials must be checked before U17/U18 configuration changes.
- **Association correction:** current native ownership/authorization, including organization ownership, must drive conforming integration rather than a competing Soda succession policy. Stored `Project.OwnerID` is not sufficient ongoing authority after transfer. Preserve Linux state and legitimate Soda memberships; do not promote every admin role or infer deletion from a failed lookup.

**User clarification now governs:** Forgejo owns its policies and upstream fixes; Soda extends/delegates and maintains its own code/patches. Preserve native external-IdP/logout/transfer behavior rather than asking the user to redesign it. The user selected **Apache-2.0 for original SodaOS code**; retain all third-party licenses, including Forgejo's terms. The original-code license/notice boundary is now authored; complete per-artifact dependency/inherited-rights compliance remains U02 delivery work, not an unanswered license-selection question.

**Historical U01 disposition — acceptance withdrawn by the template-override decision above:** the following records the earlier review, not current acceptance. The checklist above and the closing reviews in the source guide, H05 and read contract record the actual dispositions. This is not native-auth/parser conformance, a security certification, full license clearance, U02 image completion or installation approval. U01 and bounded U08 are now accepted (2/20). Native tests and source corrections remain mandatory under their named owners.

**Continue from this review:** implement/review the conforming interfaces and the specific corrections below. Technical gap-filling can proceed in parallel; do not spend another research pass on facts awaiting a decision. Before any new investigation, name the unresolved question, the existing evidence it cannot answer and the smallest expected result. Reopen settled research only for changed source/configuration, stale decision-critical information or contradictory evidence, and record that reason. Preserve earlier evidence and failures.

U01 must not offload unresolved first-contract/security architecture behind a vague “later” label. Conversely, native patches, complete image builds, full conformance matrices and installed journeys are implementation/validation work under their existing U owners—not prerequisites for repeating or accepting this planning milestone. Future feature-specific refinements stay with those owners. E/media work remains unselected.

**Exit:** the audit is reconciled with this plan, first native contracts/build/auth designs have concrete review dispositions, upstream versus Soda maintenance responsibilities are explicit and later feature-specific implementation/review/tests have accountable owners. U01 acceptance is planning/contract readiness, not native product conformance; H01 alone does not pass it.

#### Explicit authority corrections — implementation still pending

These are concrete corrections to active source, not permission to retain the behavior until U18. U18 removes superseded legacy code after parity; it is not the milestone for first fixing its authority defects.

| Correction / source | Implementation owner | Required proof |
| --- | --- | --- |
| Remove shared admin-token substitution in `internal/web/projects.go::{projects,createProject}` and `internal/web/people.go::createPerson`. Delegate through the acting user's native grant; use native site-admin authority for account administration, not Soda operator identity. Confine privileged credentials to actual authorized setup/operator tasks. | U04 with U05/U06/U16 | Each normal request uses its actor's grant; denial/expired/revoked grants never fall back to `AdminTokenFile`; a native admin who is not the Soda operator can administer, and operator status alone cannot. Cover both JSON and still-active HTMX handlers. |
| Remove `createPerson`'s Linux username/`root` restrictions and independent 12-character password policy from Forgejo account creation. Retain malformed/bounded request protections; Forgejo validates its accounts. Linux eligibility is checked only when explicitly provisioning a project account. | U05/U16 | Native-valid non-Linux usernames/passwords are not rejected by Soda-specific account rules; native rejections are preserved; unsafe project logins still cannot reach provisioning. |
| Replace ongoing authority derived only from saved `Project.OwnerID` in `apiEnvironment`/`apiEnvironmentMembers` with current trusted native ownership/authorization and stable-ID binding. Keep the association, memberships and roots; use supported native interfaces for organization ownership, not a copied role inventory. | U07 with U04/U12 | Old cached owners gain no continued authority from that row; current native-authorized actors can perform the intended operation; unrelated admin roles do not elevate. Cover transfer/name reuse/denial/unavailability and preserve accounts/homes/keys/workloads. |

The newer JSON account-administration handler already delegates to the acting user's grant, but that does not close the active legacy paths. These findings are not an exhaustive whole-codebase audit. Preserve CSRF, protected sessions, scope/identity binding, native-helper restrictions and legitimate Soda environment membership enforcement.

### U02 — React foundation and native Forgejo asset/build packaging

**Files:** `dashboard/`, static server/config, `appliance/forgejo/` (planned), `scripts/build-{dashboard,forgejo,native}.sh`, staging/sealing/install/check consumers and build/packaging tests.

- Retain the functioning client-only shell, explicit router, local branding, independent Cockpit build and dashboard-only iteration path. Complete local asset/font/notices, MIME/cache/security-header, unknown API/missing asset and startup validation coverage. No production Node or fake index fallback.
- Keep preview/base paths explicit until U18; trusted loopback development must not require permissive production CORS, insecure cookies, exposed dev servers or secrets in `VITE_*` variables.
- Implement H02's reviewed source lock/retrieval/ordered patches and exact native build, preserving upstream assets/features/runtime/data/entrypoint/protocols and corresponding source. Keep patches cohesive with native tests/schema/provenance/retirement.
- Refuse bad hashes, unsafe extraction, wrong bases, reordered/partial patches, stale/occupied outputs, missing assets/notices and wrong architectures. Wire native Forgejo tests and exact source/patch/image identities into existing aggregate build/check consumers; Caddy is unchanged.
- Test both native architectures independently when available. Artifact identity, successful compilation and installed conformance remain distinct.

**Exit:** a repeatable, identity-bound dashboard/Forgejo build through the existing delivery path, actual native build/check evidence, complete asset/license behavior and preserved Cockpit/project-tool packaging. Record resolved inputs honestly; bit-for-bit reproducibility is not inferred. No second build/release platform.

### U03 — JSON API, compatibility and database migration foundation

**Files:** `internal/{web,forgejo,store,config}/`, startup/config consumers, `dashboard/src/api.ts`, API/migration/build tests and implemented contract docs.

- Complete typed input/ID/error/CSRF/content-type/size/read-scope contracts across existing handlers; preserve complete bounded provider responses, cancellation and no uncertain replay.
- Implement H03 only for concrete reviewed extensions: required revisions, native-authenticated configured provider, finite freshness and correct incompatible/unavailable/denied behavior. Resolve the pre-auth compatibility boundary with U04; no global outage for an unrelated absent feature.
- Retain and test encrypted per-session grants, external key checks and ordered migrations. Cover populated prior/current fixtures, newer schema, partial failure, restart and missing/wrong key; never regenerate keys or infer user authority from legacy sessions.
- Specify consistent current backup/rehearsal, native-versus-Soda schema ownership, rollout order and lossless rollback limits. Update strict config consumers together, including runner companion changes; do not use first-install as migration.

**Exit:** executed adapter/migration/compatibility tests with positive, malformed, unauthorized, changed-provider and preservation failures; matched assets/config cannot silently corrupt or reinterpret existing data. Native metadata advertisement alone does not pass compatibility.

### U04 — Native-backed Soda authentication and per-session provider access

**Files:** native auth patches/tests, `internal/web/auth.go`, `internal/forgejo/` auth/transport, `internal/store/` sessions/grants, setup/config only as required, Soda authentication views and real browser tests.

- **First review H05:** explicit native next-step/challenge/verification/consent/grant/session sequence for password sign-in, forced password change, TOTP/scratch recovery, WebAuthn, registration/activation/recovery and external-method/link callbacks. Do not publish arbitrary web JSON as bearer APIs or bypass native must-change/MFA gates.
- Keep password verification/storage, enrolled credentials, counters/challenge authority, native policy/rate limits and revocation in Forgejo. Define transaction binding, one-use/expiry/replay, enumeration resistance, login-CSRF, session fixation/rotation, allowed return locations and secret-safe failures.
- Resolve WebAuthn RP-ID/allowed origins and existing credential compatibility; distinguish inspected second-factor security keys from unproved discoverable/passwordless passkeys. Resolve native mail/IdP/callback implications without changing live origins or destroying enrollment.
- Implement complete Soda login/challenge/consent/reauth presentation using reviewed native interfaces. Preserve standard code/PKCE/exchange/refresh behavior where applicable; there is no required new OAuth library or password-to-token shortcut.
- Retain session-bound encrypted grants with actual scopes/expiry, serialized native refresh and logout-winning races. Define Soda logout, native session/grant revocation and other sessions explicitly; none implies Linux/SSH deprovisioning.
- Include package/user/repository/issue/org/notification/admin scopes per operation and fresh consent inside Soda. No operator-token, Basic/security-key workaround, reverse-proxy impersonation or copied provider cookies.

**Exit:** native/API/browser proof of complete sign-in/onboarding/challenge/consent and failure/recovery paths, expiry/rotation/concurrency/replay, multi-session isolation/logout and sensitive reauthentication. No Forgejo-page hop or leaked secret. If native headless feasibility is unresolved, stop the affected architecture—not the security check. Current OAuth remains operational until isolated replacement proof; final origin cutover is U18.

### U05 — Profiles, account security, public keys and onboarding

**Files:** account/profile/key/security/application views, native account interfaces/patches, Go adapters, Soda-only preference/development-key queries and onboarding tests.

- Complete own profile/visibility/rename/avatar-source/blocking/storage/quota and preference flows using native fields/configuration; retain legitimate Soda-only values without synchronizing a second profile authority.
- Separate development-access keys from native Git SSH/GPG/signing verification. Reuse existing CRUD/challenge APIs; add reviewed missing SSH signing proof. Reject private keys/unsafe options; no generated personal private keys or promised later project-key propagation.
- Complete email primary/verification/resend/preferences, current-password change, TOTP/security-key enrollment/removal/recovery, external-login/OpenID links, account deletion and security status over U04's native challenge/reauth contracts.
- Distinguish personal API-token metadata from secret creation/deletion auth, owned OAuth clients from authorized-app grants, and real secret rotation from delete/recreate. Secrets are request-local/show-once, never persisted in client stores/logs or as a second Soda inventory.
- Keep initial People/create-person integration shared with U16: actual native admin authority, native username validation and full Soda-only first-password/MFA onboarding. Linux username eligibility is checked only on explicit environment join; no automatic remapping.
- Implement native lifecycle delegation and stable-ID association handling before enabling rename/delete/security mutations against linked records. Preserve upstream rules and explain Linux access limitations truthfully; do not invent an alternative succession/offboarding policy.

**Exit:** real native self-account/key/security/application and newly created-user journeys, admin/non-admin/non-Soda-operator distinctions, conditional settings, sensitive reauth and secret/race/denial cases. All steps remain in Soda; changing native Git keys does not silently modify existing project accounts.

### U06 — Repository discovery, creation and basic code browsing

**Files:** repository/content/Markdown views, native repository/content clients, explicit web handlers and byte/permission/browser tests.

- Complete visible repository discovery, create/init/template-generation choices, overview/README/tree/ref/file content and genuine clone URLs. Return native stable identities; repository creation never implicitly provisions an environment.
- Preserve native private/collaborator/team visibility and pagination. Compose permitted Soda environment summaries without using an operator token to fill hidden repository content.
- Provide safe Markdown/native markup adaptation, relative object links, avatars, raw/archive and LFS-aware file views. Keep bytes authorized, bounded and inert; no active repository HTML, arbitrary URL proxy or provider frontend navigation.
- Cover empty/binary/large/invalid-encoding content and native limits truthfully. Review necessary usable transfer limits rather than calling today's small subset cap full parity.

**Exit:** real native multi-page/private/read/collaborator/create/template/ref/content/download cases plus path/URL/CSRF/cross-account failures and browser deep links. Environment creation's human-owner rule does not become a restriction on general repository browsing.

### U07 — Environment creation, joining, connection and workspace terminal

**Files:** environment views, `internal/web/environments_api.go`, `internal/store/`, `internal/host/`, minimal reviewed project-native terminal integration and existing installed tests.

- Preserve canonical repository/project IDs and one eligible repository/one environment. Resolve owner, native container/account and privilege server-side. Every person explicitly joins; native account/key provisioning precedes membership success and remains separate from Git authorization.
- Finish partial native/DB failure, invalid/missing key, unavailable/stopped, forged privilege/target and retained/incomplete state cases. Do not retry uncertain creation, replace roots or turn a status row into a reachability claim.
- Keep bounded connection inspection for own login/current IP/public SSH host keys. Direct SSH/SCP/SFTP and editor guidance use trusted pins and actual routed clients; no full inspection/private keys or browser-controlled host networking.
- **Design then implement the requested terminal:** select an existing project and enter as the current user's existing project-local account/home, never host root or a shared administrator login. A fixed host-side target/account operation may open that session; it must not expose caller-selected UIDs/containers, arbitrary host commands or Podman flags. Inside the authorized own shell, ordinary native project commands remain ordinary Linux operations.
- Review transport origin/CSRF/authentication, PTY resize/encoding, input/output backpressure and limits, connection lifetime, logout/expiry/revocation, disconnect/process behavior and audit/secret handling. Do not record terminal contents or leak session material in URLs/logs. Specify what happens to shell children without killing unrelated shared workloads.
- Missing workspace/membership retains explicit join; stopped/unavailable state is truthful and never an implicit create/join/start. E02 start/stop controls remain unselected. No general recovery, IDE, private-key upload or automatic offboarding.

**Exit:** executed adapter/native/browser access and failure matrix, including owner/member/nonmember/cross-project/expired/forged terminal requests; shell identity/home and persistent changes agree with the existing account, and session termination follows the reviewed policy. Direct-IP access remains separately supported. Fake helper tests or SSH access alone do not prove browser terminal delivery; U08's accepted earlier scope is unchanged.

### U08 — First installed product proof

**Accepted for the recorded bounded scope, not a new execution checklist.** Keep core-owned `tests/installed/`, runtime corrections and private evidence. Later headless/terminal changes are tested under U04/U07/U17/U20 rather than retroactively adding requirements to this acceptance.

#### U08 completion execution plan — baseline `0d4c4eb`

This historical heading is retained for evidence links. The original staged checklist was executed and refined through `952f3b3`, `935dbdf`, `f233a4a`, separately approved `c96c108` and merged closure `8b823db`; its full text remains in Git history at `c832901`. It is not permission to repeat fixture creation, stop/start or reboot.

The run established actual administrator/two-user onboarding, personal keys and Git, explicit Linux joins, direct routed client access, shared installed tools/files, default bridge HTTP/PostgreSQL, concrete namespace/socket/startup fixes and retained-state lifecycle comparisons. Early readiness/exec/snapshot/operator failures remain in the evidence. Exactly four environments are retained; no fifth is required to preserve this verdict.

#### U08 closure reconciliation — merged candidate

| Criterion | Accepted evidence / boundary |
| --- | --- |
| Exact build and affected rollout | Clean native `8b823db` build/seal/full check, restricted consistent populated-v3 backup and isolated rehearsal; exact dashboard/helper/runner/default-image binding, not whole-host upgrade |
| Browser/security/substrate | Real independent operator/Alice/Bob OAuth/navigation/logout and connection authorization; trusted TLS, asset 404/API 401, full `host.sh`, root Cockpit/PAM and non-root denial |
| Onboarding/provisioning | Native account creation/first-password change, development keys, private repositories and explicit Alice/Bob joins; creation/onboarding evidence reused only for unchanged handlers |
| Routed access/authority | Actual infra→project-IP SSH/PTY/SCP/SFTP with independently pinned public host keys; owner sudo/member and cross-project denials; no human host accounts/unrestricted host engine |
| Native Git/shared resources | Personal encrypted Git credentials/agents, genuine commits/remote refs, same shared executable installation/ownership and shared files; not a cache-only claim |
| Workloads/different-UID exec | `c96c108` exact fresh creation, project-owned namespaces/default seccomp, bridge image build/bind HTTP/committed SQL; default-root/UID-999/PTY exec passed and Bob's engine access denied; read-only readiness resumed without rebuild/reseed |
| Lifecycle/preservation | Real `f233a4a` existing-project stop/start and **only `soda-test`** reboot; trusted reconnection/own-agent unlock/native workload starts and stable state compared. Reused for explicitly unchanged mechanisms, not a new c96c108/8b823db reboot |
| Rebuilt-image delta | Failed strict layer-ID comparison retained; content/mode/owner/link/capability audit found only rebuilt Tea content changed, with native immutable-image version proof; runtime layers/config unchanged |
| Four retained environments | `8b823db` regressions preserved declared roots/records; only intentional access-probe additions in the newest root. Boot ID unchanged during closure |
| Operator/provider limits | Package-provider and stale strict-config runner fixes verified. Corrected console probe **fails** for missing hook; P11/U20 delivery remains. Zero runners/`NeedsLogin` are not provider/enrollment acceptance |

Evidence lives under `.artifacts/logs/u08-{completion,ptrace,closure}-*`, retained native stages and restricted fixture/backup locations in the [handoff](implementation-status.md#u08-accepted--bounded-native-x86_64-first-product-proof). No failed result was erased or an old backup declared current rollback. Trusted-team extra namespaced capabilities/`label=disable` and cgroup limits are not hostile-tenant confinement. U20 owns final-image lifecycle, fresh/upgrade, full operator/provider and native aarch64 proof.

#### Core-owned native proof detail

These requirements remain with U08/U20, not a duplicate P07/P08 suite. Extend existing `tests/installed/` and bounded fixtures; predecessor techniques are references, not another product runner.

- Use real frontend/native onboarding, repository selection and explicit joins. No seeded sessions/accounts or successful rows substituting for provisioning. In U20 use the completed Soda-only authentication path.
- Resolve current project addresses and independently trust public host keys. Exercise interactive/exact-output SSH and bidirectional SCP/SFTP from the named routed client. QEMU forwards, ProxyJump and appliance-local access do not prove that direct path.
- Check authorized operations and specific native denials; transport failures are not denial evidence. Independent projects retain separate roots/host keys/authority even if local UID numbers coincide.
- Use personal home checkouts and each user's own native Git credentials for clone/commit/push/readback. Do not borrow tokens/agents; retain restricted passphrases needed to unlock after lifecycle events.
- Prove the same canonical shared mise installation/files and intended ownership, including noninteractive resolution. Version/path strings or duplicate downloads alone are insufficient; ordinary members cannot replace admin-owned tools.
- Exercise the project-local engine boundary with real image build, bind-mount HTTP edits and committed PostgreSQL values reached by both members and the client. Compose parsing or an unrelated host engine is insufficient.
- Before separately authorized lifecycle changes, capture complete bounded allowlisted identity/account/group/key/home/dirty-work/tool/config/workload/volume/data observations. Failed/incomplete snapshots fail the comparison; never export shadow/password/private-key data.
- Stop/start the same existing project, then repeat only around an explicitly approved appliance reboot. Compare stable state, not volatile IPs/PIDs/device numbers; observe independent retained roots. No replacement, pruning, `down -v` or reseeding.
- Record actual native workload restart behavior and verify exact committed data. An explicit start of the same container/volume is not automatic resurrection. U20 adds final own-workspace terminal and complete collaboration/admin checks without replacing these direct-access assertions.

### U09 — Code history, comparison and native repository writes

**Files:** existing history/file-editor/repository-copy views and Go clients/handlers/tests; reviewed native Git read/write interface patches and core installed Git fixtures.

- Reuse immutable SHA binding, exact result validation, minimum read scopes, complete per-commit comparison lists, byte bounds and route/account draft guards already locally checked. Do not call concatenated per-commit files a net diff or invent pagination.
- **First native slice H04:** share native blame/ignore-revs and comparison engines with existing web callers; expose explicit typed attribution/original paths/commit identities and direct-versus-merge-base/net-file/hunk semantics. Bind full base/head/merge-base identities and authorize every disclosed resource; no Soda clone/index, temporary PR or copied Git algorithm.
- Complete history/verification/diffs and branch/tag create/rename/delete/**native deleted-branch history/restore**; guessed-SHA branch creation is not native restore. Include notes and native cherry-pick/revert editor workflows. U13 owns search implementation; integrate its repository/commit search in U09 navigation.
- Complete native create/edit/delete/upload/rename/multi-file/patch commit choices with provider SHA/precondition/protection/branch authority and exact result checks. Preserve same-target drafts and reject stale/cross-account responses; no uncertain mutation replay.
- Complete native permitted personal/organization forks, ahead/behind/sync and supported import/migration destinations/options. Current personal fork/basic synchronous HTTPS import is a subset, not the entire contract. Native task progress/retry needs its own interface; a timeout proves neither cancellation nor cleanup.
- Resolve native computation/time/output/concurrency/cancellation budgets and usable large/binary behavior. Copy/import credentials stay request-local and absent from storage/logs/redirects; never widen native migration network policy or provision/copy environments as a side effect.

**Exit:** all audited code/ref/write/copy workflows have N/A/B proof, including private/read/write/protected/invalid/stale/malformed/ambiguous paths, blame and net comparison against ordinary native Git. Missing native interfaces block U09, not merely a U17 note. Repository transfer/archive/deletion settings stay U12; workspace terminal stays U07.

#### U09 completion plan — existing source to installed acceptance

This supersedes the earlier A–H checklist's narrow personal-copy/one-time-import scope and stale “missing DOM suites” statements. H01 is complete as source discovery; first native contracts, remaining implementation and installed acceptance are not.

1. Review selected baseline, first blame/net-diff wire/authority/budget contracts and early H05 feasibility; establish U02/U03 native build/compatibility inputs. Author native tests with each shared-service/API change.
2. Deliver that read slice through Forgejo → explicit Go adapter → React, retaining existing working reads/writes. Test rename/delete/binary/invalid encoding/large/ignore-revs/divergent history/ref movement/cancellation and mismatch; no incomplete result masquerades as complete.
3. Complete remaining refs/restore/notes/file/patch/cherry-pick/copy/task/sync actions in bounded batches; reuse native APIs before adding interfaces. Coordinate U13 search and U12 authority/settings contracts, not entire milestone acceptance.
4. Run already authorized focused and aggregate local suites as applicable, including native contract tests after the reviewed additions exist. Build clean exact Soda/Forgejo source/patch/image candidates through existing entrypoints, preserving prior artifacts. No new project image/runtime change is implied by U09.
5. With exact rollout scope, take consistent current native/Soda/config/key/artifact backups, rehearse supported pairing/migrations and deploy matching affected components. This may now include Forgejo; the old dashboard-only rollout assumption is no longer sufficient. Keep preview/login/protocols and retained services usable; no U18 cutover by implication.
6. With separately approved repository/import actions, verify real two-user React outcomes through supported native reads and personal Git in new checkout directories inside existing projects. Preserve original environment/membership associations, roots and workloads; no fifth environment or lifecycle operation.
7. Reconcile **all** U09 actions and N/A/B evidence by exact revision/bytes; retain failures, fix defects and rerun affected cases before accepting the milestone. U17 integrates that evidence; it does not supply missing U09 functionality later.

**Installed fixture proposal, not permission:** retain Alice/Bob and four environments. Propose Alice's private repository-only `u09-code-<candidate>`, Bob's fork, Alice's import and a separately named failed-import target if needed. Native read→write collaboration phases and fixture-only branch protections must not touch U08 repositories. Use independently authorized personal Git, verify exact refs/bytes, and demonstrate stale concurrent edits preserving the newer commit and the other user's draft. Approve a small HTTPS import source actually reachable by Forgejo under existing TLS/network policy; browser reachability is insufficient. Broader native org/task/copy cases need their own exact fixtures/actions rather than silently extending these names. Retain resulting partial resources, failure logs and checkouts; respect native failure semantics without adding Soda retries or cleanup.

### U10 — Issues, labels, milestones, time and boards

**Files:** issue/conversation/metadata/activity/board views, existing stock clients/handlers and reviewed lock/history/board native patches/tests.

- Complete issue filters/state/assignments/ref/due dates, labels/milestones, Markdown/YAML templates and configured contact/blank-issue choices. Reuse native template/config/validation APIs; do not invent an issue schema or render repository text as executable instructions.
- Reuse native timeline/comments, reactions, subscriptions/ignored state, issue/comment asset CRUD, timers/tracked time, dependencies, pin/reorder and deletion. Correct Soda read-scope overreach and replace attachment `native_url` with a bounded same-origin authorized byte adapter; the native byte interface already exists.
- Add reviewed native lock/reason and edited-content-history/detail/soft-delete interfaces, retaining native author/writer/admin gates and audit semantics.
- Implement **repository, organization and personal issue projects/boards**: cards, columns/default/order, CRUD/open/close, issue association/move/reorder under native unit/owner/card visibility. These are Forgejo issue projects, not Soda environments; no local board store or migration/F3 interface repurposed as human CRUD.

**Exit:** real two-user issue/template/timeline/assets/time/dependency/board journeys with native identity/order/state, positive and forbidden/stale/oversized/hidden/unit-disabled cases, inert bytes and accessible browser interactions. Native mutations do not call the environment helper.

### U11 — Pull requests, reviews and merge

**Files:** pull/review/diff/check views, existing native review/merge clients and missing-state patches/tests; real Git fixtures.

- Complete list/create/retarget/state/maintainer-edit/templates/draft workflows; obtain configured native draft prefixes/metadata rather than hardcoding an alternate policy.
- Reuse existing native review/reply/pending/submit/delete, old/new-side positions, individual/team requests and dismiss/undismiss APIs. Preserve native conversation identity, outdated context, pending-review visibility and exact commit/diff binding; a reply API already exists.
- Add native resolve/unresolve, viewed/unviewed/changed-since-viewed and complete merge blocker/options/pending auto-merge state. A local viewed flag or `mergeable` boolean is not equivalent. Forgejo owns protections/checks/approvals and calculations, not Soda.
- Complete commit-range/context inspection, update/rebase, merge/squash/manual choices, schedule/cancel auto-merge and permitted branch deletion. Use SHA-bound operations and refresh actual results; no automatic replay or merge-triggered environment promotion/cleanup. U14 owns real Actions/trust state.

**Exit:** native two-user author/reviewer/merger/team/pending/outdated/old-side/resolved/viewed/stale-head/conflict/check-failure and merge-state tests, plus adapter/DOM races and real Git result verification. No copied merge engine, misleading eligibility panel or provider-page escape.

### U12 — Repository settings and organizations/teams

**Files:** settings/access/protection/hooks/org/team/invitation/client views, existing clients/handlers and precise native settings/hook/authority patches.

- Complete native settings/units/merge options/topics/avatar/subscription/collaborators/protection/deploy keys/mirrors/Git hooks/LFS and unadopted-repository workflows. Existing transport does not imply complete fields; keep code-executing Git hooks, site-admin health/index and native network/config gates explicit.
- Expose **already implemented** advanced protection and team-policy adapter fields (RS06/OR04) before inventing native APIs. Reuse native organization/team/member/repository-assignment/label/activity/block/quota operations without mirrored roles.
- Add missing units/settings/fork-detach/transfer-cancel/federation/LFS administration and organization invitation/OAuth-client/rotation/grant contracts. Email/token invitations are not the same workflow as adding an existing user to a team.
- Complete hook handler-specific creation, signing-secret keep/replace/clear, package/Actions events, explicit tests/delivery inspection/replay and default/system kinds. Reuse native handler logic; preserve omission semantics and redact secrets/auth headers/URLs/payloads. A delivery replay is an explicit network mutation.
- Implement native rename/transfer/accept/reject/archive/delete workflows with conforming linked-resource handling, retaining native user/organization ownership predicates and scopes. Do not retain stale creator authority, silently rename/deprovision projects or turn site/org/repository administration into blanket project/root authority. U14 owns Actions configuration authority parity; U16 reuses instance operations.

**Exit:** full reader/writer/repo-admin/owner/org-owner/team/site-admin matrix and revocation, partial-update/clear/conflict/secret/network-effect tests, with native outcomes and preserved Soda/project associations. Unresolved lifecycle decisions block affected acceptance, not become an exclusion.

### U13 — My work, search, notifications and activity

**Files:** work/search/notification/profile/activity/graph views, native query clients and reviewed native search/graph/statistic interfaces.

- Complete assigned/review-requested work, repositories/orgs/teams, user profile/follow/star/watch/activity and notification read/unread/pinned/filter/native subject navigation. Keep Soda environment information separately sourced.
- Reuse native repository/issue/PR/user/org/topic search and heatmaps/feeds. Add bounded native code/commit search, multi-ref graph and period/contributor/code-frequency/recent-commit activity where missing; share repository search navigation with U09.
- Native filtering/paging/visibility stays native; no fetch-all inventory, Soda index/event store, generated fake pagination or cross-account cache. Handle independent dependency failures and route/query/account cancellation.

**Exit:** actual multi-user private/public/collaborator search/activity/notification results, native updates, correct paging/deep links and late-response isolation; graphs/search do not disclose hidden commits or require a provider frontend.

### U14 — Actions and automation configuration

**Files:** Actions/workflow/job/log/artifact/configuration/provider-runner views, explicit native clients and missing-semantic/authority patches/tests.

- Apply the selected baseline decision: **v16 already has human jobs/logs/artifacts/cancel APIs**; do not blindly recreate/backport all v15 gaps. Verify exact native token/resource/unit/write predicates, byte Range/archive/attempt/expiry semantics and deliver complete run/job/step/attempt/progress/diagnostic views. Task-token `/actions/run` is not human authority.
- Reuse native dispatch/input validation and completed-run deletion. Add missing workflow input/default/options/enabled metadata, enable/disable, full/native selected-job rerun and PR trust/approval controls. Re-dispatch is not rerun; file scanning/editing is not workflow state.
- Resolve repository API owner-only versus web-admin configuration/runner authority natively, without owner-token substitution. Preserve correct user/org/repo/site scope and distinct site administration.
- Complete secrets/variables and metadata rename/keep-value behavior, including instance variables with U16; no secret-value readback/storage. Complete provider runner registration/list/read/delete/token/jobs and missing name/description/credential/token reset. Labels are agent-reported, not invented edit fields.
- Bound polling/log streaming/download buffers and lifetime; stop on navigation/logout and distinguish absent/expired/unexecuted output from empty success. Forgejo owns scheduling/workflows/results; host local-capacity services remain Cockpit/P11.

**Exit:** native human reader/writer/admin/owner/unit/expiry/large-output proof and separately approved real workflow/run/dispatch/cancel/rerun/trust/configuration/runner actions. No fake job, runner-only protocol proxy, native-page fallback or replacement scheduler. Provider mutation scopes are explicit, not inferred from local test permission.

### U15 — Releases, wiki and packages

**Files:** release/wiki/package views, existing release/content/byte clients and reviewed native clear/wiki/package extensions/tests.

- Complete native releases/drafts/tags/publish/delete/assets and authorized byte transfers. Extend actual release-note clear semantics shared with native web edit; do not assume all empty fields are valid or delete/recreate a release to clear notes.
- Reuse current wiki/index/sidebar/footer/revision-list/create/edit/rename/delete APIs and native wiki-branch normalization. Add historical content/raw/diff/restore-content/search and full-wiki deletion contracts. The wiki is a separate native Git repository; main-repo endpoints are not its history API.
- Preserve **native wiki last-write behavior**: neither inspected web nor API has edit CAS. Do not fabricate a stale-write guarantee; any stronger native precondition is a separately reviewed semantic change.
- Reuse package inventory/version/files/link/unlink/delete/quota and format-specific publish/download protocols with package consent. Complete native descriptor/properties/counts/install context, owner cleanup-rule CRUD/preview/run and native Cargo/Chef settings interfaces. No Soda registry, index, artifact/cleanup engine or secret store.
- Handle draft/private/hidden/expired/duplicate/quota/native-unit restrictions, safe filenames and complete bounded transfer/cancellation; copyable tool commands must not embed real secrets or send browsers to Forgejo pages.

**Exit:** real release/asset/wiki revision/search/package metadata/protocol/settings outcomes and negative authority/byte/native-write-semantics tests. Explicit cleanup/deletion/maintenance fixtures need their own scope; missing UI or missing native fields are not conflated.

### U16 — Forgejo site administration and security-sensitive administration

**Files:** complete admin views, shared People/account/configuration/hook/runner/package clients and reviewed native admin interfaces/tests. Begin design and independent slices alongside U04/U05, not after all U15 work.

- Complete native People/status filters/editable fields/MFA status and explicit **other-user MFA reset**, email activation/avatar/keys, organizations/repositories/unadopted resources and native flags. Reuse available APIs; missing filter/detail fields require native contracts, not fetching all users or applying self-account authority.
- Complete native overview/runtime/health, notices, queues/process/stacktrace/diagnostics, selected configuration/auth-source/OAuth-client operations, cron versus special branch/tag synchronization and explicit mail/cache/conditional DB self-check. Redact secret-bearing detail; expose only native UI-supported bounded settings/actions, never arbitrary app.ini/filesystem/shell/internal-method access.
- Reuse shared hooks/default hooks, provider runners/jobs, quotas and package inventory/cleanup contracts with U12/U14/U15. Native quotas/cron/queues are upstream functions, not a Soda scheduler or environment quota service.
- Implement conditional moderation/abuse/federation user and admin interactions with their real actors/config gates. A native “report abuse” action is not automatically site-admin-only because its review screen is administrative.
- Complete U04/U05 sensitive reauth/auth-source effects and reviewed native rename/delete/disable/security consequences for linked Soda records/sessions. Preserve Linux access/roots and distinguish site admin, Soda operator, project administrator and Cockpit root.

**Exit:** complete native ordinary-user/site-admin/downgraded/insufficient-scope/non-Soda-operator matrix; actual permitted outcomes, redaction and sensitive-confirmation/denial tests. Maintenance/reset/delete effects execute only against exact approved native fixtures. No new universal administrator role or Forgejo UI escape.

### U17 — Page/API coverage, update verification and browser closure

**Files:** single coverage register, feature-owned native/adapter/browser tests, compatibility/update fixtures and auth/content/ingress integration checks.

- Track each action's concrete contract, configuration, authority, implementation and N/A/B evidence throughout delivery. Route defects back to the owner; do not defer boards/graphs/auth/admin implementation to this milestone or create a second readiness database.
- Close every required action, including conditional-enabled behavior, origin/mail/IdP decisions, linked-resource effects and UI05 terminal/environment boundaries. The TSX `forgejo_url` guard is supplemental: inspect redirects, errors/help, native URLs, Markdown/mail/notification links, authorized bytes and actual browser requests.
- **Demonstrate H07:** on a reviewed upstream update or contract-changing candidate, review support/security/migration and touched native services, rebase patches, rerun native authority/semantics and Soda consumers, check supported rollout pairs and preservation. Clean patch application/version advertisement is insufficient. Deliberately adopt equivalent stock APIs and retire patches with equivalence tests when applicable; no speculative duplicate backends or automatic updater.
- Verify complete Soda-only workflows and the proposed API/Git/SSH/LFS/package/Cockpit listener/proxy separation on the matching approved candidate/rehearsal target. Frontend restrictions must not break native protocols or rely on hiding navigation/User-Agent tricks.

**Exit:** all registered requirements have delivered evidence, no unresolved class-3 contract/authority/lifecycle gap or unavailable placeholder is counted complete, and update/compatibility/rehearsed end-state browser tests pass. **U17 proves the candidate; U18 changes the retained installation.** Do not require live ingress closure before its login replacement is proved, or make U17 depend circularly on completed U18. No missing-API scope waiver.

### U18 — Default SPA cutover and legacy removal

**Files:** Go/router/build base, `appliance/config/proxy.Caddyfile`, explicit native auth/listener/origin config as reviewed, build/stage/install consumers and browser/operator documentation.

- Require accepted U08 plus U17's complete candidate/ingress/update proof. Inventory exact installed bytes and current populated state; rehearse native/Soda/config/key migration and rollout order, then obtain the exact retained-target cutover scope.
- Move React from `/app/` to default browser routes and apply the reviewed Soda-only authentication/content/direct-ingress boundary. Preserve valid native API/Git/SSH/LFS/package and separate operator Cockpit access. Origins, RP-ID and callbacks are reviewed contracts, not assumed unchanged or reconstructed from old ports.
- Preserve stable environment/repository/user links and legacy GET bookmarks through explicit Soda redirects. Never redirect/replay old POST mutations. Keep working login until the replacement passes and preserve enrolled credentials; no bootstrap/first-install replay or environment replacement.
- Remove replaced HTMX routes/templates/assets and operator-token frontend callers coherently. Remove license payloads only when their material is no longer included; retain actual setup/native consumers and required attribution. Retire temporary preview/dual-stack branches, not a permanent frontend selector.
- Deploy matching strict-config companions/assets/native interface versions together; document interruption/re-authentication and irreversible changes. Backups and prior binaries are useful evidence, not permission to discard subsequent writes as rollback.

**Exit:** retained installation serves all required frontend workflows only through Soda, no browser escape through auth/content/direct origin, real protocols/Cockpit still work, old bookmarks and persistent IDs/data survive, and exact installed security/byte/preservation checks pass. A successful bundle or partial preview is not cutover acceptance.

### U19 — Make it good

**Files:** actual feature components/styles/stores, measured API/native hot paths and usability/browser evidence.

- Review real developer/admin/terminal tasks for navigation, density, forms/tables/diffs, responsive layouts, keyboard/focus, announcements, contrast and reduced motion. Apply PatternFly/canonical branding without another design system.
- Measure bundle/render/request/polling/large-diff/log/terminal behavior; use targeted route splitting, bounded rendering and state refresh for observed problems. Do not add a cache framework or speculative editor.
- Improve clear failure/recovery-without-recreation guidance, confirmations and connection/tool instructions. Keep private data session-bound and all Forgejo workflows in Soda.

**Exit:** before/after measurements and real-task accessibility/usability evidence, no security/authority/retained-service regressions. Basic functionality/accessibility and native resource safety were required earlier, not postponed here.

### U20 — Final installed verification and handoff

**Files:** existing core/source/native/browser/packaging tests and actual installation/operator/developer/coverage handoff.

- Execute final-revision Go/native Forgejo contract/race, both frontend suites, browser, asset/build/packaging/compatibility tests. Preserve Cockpit/PAM/Tailnet/Runners/console/branding/project-CLI tests and resolve the missing console-hook delivery with P11 evidence.
- Prove a clean installation on a separately approved fresh target and a controlled populated-state upgrade with exact Soda/Forgejo source/patch/image/config identities, native/Soda migration/credential/origin behavior and current-state preservation. The retained early VM is not a clean final install.
- Repeat the [core proof](#core-owned-native-proof-detail), full Soda-only onboarding/security/developer/collaboration/admin/provider journeys and U07 terminal with real native outcomes. Account for lifecycle/logout/reconnect, unchanged roots/host keys/dirty work/shared tools/HTTP/committed SQL; explicitly observe workload restart behavior.
- Build and execute independently on matching native x86_64 and aarch64 hardware. An unavailable sibling does not block useful work, but full two-architecture acceptance cannot be inferred from cross-compilation/emulation or browser-only tests.
- Consume exact P06/P11/P12/P13 host/operator/artifact evidence, not a second P product suite. Optional media adds proof only if selected. Separately scope provider jobs/enrollment/maintenance, fresh resources, routing, destructive fixtures and reboots; missing required execution stays incomplete, not PASS.
- Record an honest handoff of verified architecture/configuration, complete coverage, source/patch/license/security-update ownership, install/migration instructions, credential/access lifecycle and trusted-team/runtime/network limits. No “remaining upstream screens” exception to the full frontend.

**Exit:** all required final source/native/browser/upgrade/operator matrices pass on both native x86_64 and aarch64, and real operators/developers can complete the documented product without missing integration or provider-page fallback. Useful one-architecture results remain valid partial evidence, not complete U20 acceptance. Retained failures/limits stay explicit. Publication, upstream submission, other-instance enrollment and an updater are separate actions, not implied by acceptance.

## 8. Conditional Soda extension milestones

E01–E03 remain **unselected**, outside the 20 core milestone count. Do not create dormant flags/tables/helper operations. Selection must update the product/deferred boundary and add relevant U17/U20 proof. Native Forgejo settings/quotas and the already requested U07 terminal do not select these extensions.

### E01 — Rocky/Fedora creation-time image profiles

**Gate:** explicit OS-choice/release/image approval and U08 baseline proof. **Files:** project image/rootfs, native build/stage/create protocol, narrow metadata migration and environment UI/tests.

- Define a small approved immutable profile set with native architecture/distro/release/image identity; no caller-selected registry URLs/flags.
- Reuse account/SSH/shared-tool behavior and verify each distro's package/Python/systemd/mise/CLI/storage/cgroup/SELinux/nested-runtime differences natively. Changing `FROM` is not compatibility proof.
- Add only needed creation metadata; preserve old defaults/identities without pretending retained projects used today's image. Choice applies only to new creation, never in-place distro switching or stopped-root replacement.

**Exit:** approved Rocky/Fedora fixtures pass the same real access/shared-workload/persistence/authority matrix on their claimed architectures; invalid/unsupported choices are refused and existing environments remain unchanged.

### E02 — Existing-environment lifecycle controls

**Gate:** explicit actors/start/stop/restart scope and U08 proof. **Files:** fixed helper/unit operations, server authorization, environment UI and native tests.

- Resolve only the existing trusted labeled environment/unit; no arbitrary Podman/systemd target. Preserve accounts/rootfs/host keys/tools/data and normal existing-container startup.
- Warn about shared service/session disruption, including terminals. Report already-stopped/running/timeouts honestly; no recreation, automatic remapping/repair or host reboot/network control.

**Exit:** permitted operations and precise cross-project/unauthorized denials, retained persistent state and documented explicit start behavior. No delete/rebuild/auto-idle feature.

### E03 — Basic resource limits and usage

**Gate:** explicit policy and native cgroup/storage investigation after U08. **Files:** bounded create/inspect/config metadata, environment/admin views and cap tests.

- Select native CPU/memory/PID limits and permitted actors; verify effective inheritance by nested workloads. Usage observation is not enforceable quota.
- Bound non-secret native usage output. Investigate actual writable-root/volume storage before promising disk enforcement; no inferred generic flag support.
- No scheduler/admission/billing/automatic shutdown/deletion or pressure-repair subsystem; existing projects are not silently resized.

**Exit:** native effective limits match approved policy and cannot be bypassed by ordinary nesting; invalid choices fail, persistent state survives and unsupported disk enforcement is not claimed.

### Other gaps remain decisions, not hidden milestones

General key propagation/offboarding/identity remapping/reconciliation/recovery, org-to-project administration, multiple environments, project deletion/image replacement and backup platforms remain deferred. This does **not** defer Forgejo's own required rename/delete/security interfaces or normal authorization/preservation. Resolve their exact Soda association effects under the owning core milestone without silently building those larger subsystems.

## 9. Inventory coverage cross-reference

The [179-group register](forgejo-api-coverage.md) remains the **single action/contract/evidence inventory**. This table assigns a primary implementation owner for each existing group; named collaborators provide shared contracts and integrate their views. Do not copy endpoint details/status into another synchronized register. Ranges below cover every H01 group exactly once; action splits/additions must keep ownership current, not freeze the count as a scope cap.

| Audit groups | Primary owner | Integration responsibility |
| --- | --- | --- |
| AU01–AU10 | U04 | U05 account/onboarding and U16 auth-source/security contracts; U18 installed origin transition |
| AU11–AU22 | U05 | U04 native challenge/reauth/grant lifecycle; U16 administrator counterparts |
| AC01–AC05, AC07, AC11–AC13 | U05 | U07 linked account/access consequences; native key/profile/security authority stays Forgejo |
| AC06, AC08 | U13 | User/profile/follow/activity navigation; U05 own-account entrypoints |
| AC09, CO01–CO02 | U06 | U09 refs/changes and U15 package bytes; U07 separately owns environment creation |
| AC10 | U02 | Native-backed about/settings/help/tool guidance with U05/U16 fields; no native-page help fallback |
| CO03–CO16, CO18 | U09 | U11 diff consumers, U12 ref/settings authority and native copy lifecycle |
| CO17, WK01–WK06 | U13 | U09 repository code/commit-search views reuse native search contracts |
| IS01–IS15, BD01–BD04 | U10 | U11 conversation/asset reuse; boards remain native issue projects, not environments |
| PR01–PR12 | U11 | U09 diff/ref and U14 actual checks/trust/run integration |
| RS01–RS16, HK01–HK06, OR01–OR09 | U12 | U05 personal clients/hooks, U14 Actions authority, U16 instance counterparts and lifecycle decisions |
| CI01–CI15 | U14 | U11 checks/trust; U16 site-scope configuration/runners; Cockpit still owns local capacity |
| RE01–RE04, WI01–WI05, PK01–PK06 | U15 | U04 package consent; U12 wiki setting entrypoints; U16 instance package/quota views |
| AD01–AD23 | U16 | U04/U05 sensitive account/auth mechanisms; reuse U12/U14/U15 operations; retain native ordinary-user moderation/federation entrypoints |
| UI01–UI04 | U17 | Each feature implements its own safe links/content/auth; U18 performs verified default-route/live ingress cutover |
| UI05 | U07 | U08 retained earlier access evidence, U20 final terminal/access/preservation and P11 separate operator integration |

Soda-only preferences/development keys/environment records/terminal are U05/U07; cross-cutting API/data/build are U02/U03/U04; default routing U18; focused whole-product polish U19; final native architecture/upgrade proof U20. U17 integrates these alongside the register rather than treating missing features as its own parking lot. E tracks and outside operator/media work retain the boundaries above.

## 10. Definition of done, execution and handoff

### Per-feature completion

**Source-complete** means connected production native/provider functionality, explicit API/adapter, React workflow, required persistence/config/build/license wiring and focused tests—not a stub, mocked success, unavailable placeholder or manual Linux checklist.

Track separate evidence states in the existing register/handoff:

1. **Planned/review pending:** contracts and decisions only.
2. **Source-complete:** real callers/tests exist; no execution implied.
3. **Source-tested/built:** exact revisions, actual suites and artifact identities recorded.
4. **Installed-verified:** exact native target/client/actors/configuration and real positive/negative outcomes recorded.
5. **Blocked integration/execution:** precise unresolved contract/decision/permission/failure and owner; not a native-page waiver or milestone PASS.

A U milestone is accepted only against its stated complete exit and evidence. Conditional native settings distinguish enabled/disabled/denied/missing/incompatible/unavailable; source defaults are not live observations. Update the one register when native versions/fields/authority change, not just when a page appears. U08 is accepted only in its recorded historical scope.

### Verification matrix to maintain

- **Native Forgejo:** exact built source/patch/architecture, shared web/API semantics, actor/resource/scope/unit/config gates, token/challenge lifecycle, revocation and bounded expensive work/cancellation. Test real outcomes, not only interface advertisement.
- **Go/API:** typed complete results/IDs, read/write separation, partial-update/clear behavior, pagination, denied/malformed/oversized/timeouts, secret redaction and no privilege fallback/uncertain replay.
- **Auth/data:** headless challenges/WebAuthn origins/consent/recovery/reauth, CSRF/session fixation, encrypted grants/refresh/logout races and native/Soda populated migration/refusal/rollback limits.
- **React/browser:** real forms/keyboard/focus, safe content/downloads, route/page/query/account races/drafts, all authentication/navigation/mail/provider-origin paths and own-workspace terminal lifetime.
- **Build/update:** verified source extraction/ordered patches/real notices, exact native image pairing, missing assets, all config consumers, rebase/security review and equivalent-API patch retirement. No automatic CI/update system is implied.
- **Installed/preservation:** real native Git/collaboration/admin/provider/terminal and direct SSH/shared-workload/data outcomes, exact client reachability/pins, final fresh/upgrade/lifecycle and retained Cockpit/console/runner integration.
- **Architectures:** independent matching-native x86_64/aarch64 and each explicitly selected userspace/media profile; no sibling/emulation inference or optional-media prerequisite.

Each coherent commit/handoff names changed paths and owners, native contract/authority/data effects, authored versus executed tests, exact bytes/targets, preserved failures and next concrete work. Keep Git history/unrelated work intact; do not amend without permission.

**Execution:** already authorized local builds/automated tests need no renewed generic permission request. Concrete new native patch/dependency scope follows its review; deployment, real provider/account/repository/import/runner mutations, installed fixtures, origins/networks, publication/submission, lifecycle and cleanup retain exact separate scopes. Never expose credentials in source/argv/tracing/logs/screenshots, overwrite private inputs/evidence or infer permission to reset the VM. This plan revision runs none of those actions.

**Next work:** reconcile U01 and the affected U02–U20 assignments with the selected official template-override direction at the top of this plan. Do not proceed with the former mandatory native build/headless API sequence. Preserve useful source/evidence and bounded U08 acceptance (1/20).

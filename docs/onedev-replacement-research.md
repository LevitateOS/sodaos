# Replacing Forgejo with OneDev — source-grounded research

**Research date:** 12 September 2026. **Status:** reference research, not active work. The user [chose to retain Forgejo](architecture.md#forge-selection) rather than introduce OneDev's Java/JVM server environment. No OneDev plugin proof or replacement is selected.

**Scope if reconsidered:** The user clarified **zero users and zero deployed installations**. The [pre-release replacement scope](architecture.md#pre-release-replacement-scope) excludes a customer migration programme. Development fixtures do not create a legacy support obligation.

**Finding, not an adoption recommendation:** OneDev genuinely supports native Java/JVM application plugins; it does not provide a native Go plugin API. That capability does not resolve the user's objection to operating the Java environment. The source findings and conditional integration options below are retained for reference.

Current product requirements remain with [Architecture](architecture.md), [Project OS](project-os.md), [native pages](forgejo-soda-pages-plan.md), [Runners](runners-port.md), and [Services/AI](services-and-ai-plan.md). This report evaluates those requirements; it does not silently change them. Installed state and permissions remain in the [handoff](implementation-status.md).

## 1. What was actually researched

- SodaOS source at `a921a2ea22b6ac3e15e3a1639c340afeda9f1091`, initially clean.
- Soda's architecture, native-page/session contracts, actual Forgejo client and authorization callers, SQLite migrations, native account markers, runner registration, container recipes, frontend/template inventory, and planned AI/runtime requirements.
- OneDev **16.6.2**, reported as the latest GitHub release on the research date. Annotated tag `v16.6.2` resolves to commit `1c84e0b18e31364a933d9b6503e635008d989e51`. Its GitHub repository is now `theonedev/onedev`; the old `theonedev/server` URL redirects there. Development/issues live at `code.onedev.io`, not GitHub's issue count. [R1]
- The complete public OneDev source archive, selected implementation call paths, and the exact **commons-loader/bootstrap 4.2.1** source JARs used by that release. Also the published parent POM **1.3.0**.
- Current official documentation, pricing and distribution license; published Docker tag metadata; two current upstream security advisories and NVD records; one independent plugin example.

Evidence levels used below:

- **Source-verified:** an implementation or its actual caller was inspected. This establishes a mechanism, not a successful Soda integration.
- **Documented/published:** upstream says it or publishes the artifact metadata. It is not a local runtime result.
- **Proposed/unproven:** an engineering recommendation or a capability that still needs execution evidence.

**No OneDev server/plugin was built or started, no appliance/fixture was changed, no credential was read, and no import, registration, provider job or migration was performed.** No browser, load, native-architecture, security-audit or upgrade acceptance is claimed.

Public downloads, parsed documents, hashes, metadata and failed retrievals are retained under `.artifacts/onedev-research-34y1pE/`. The docs index records URLs and retrieval times. Exact source links below use the inspected commit; documentation/pricing are rolling pages observed on the date above.

## 2. Correcting the original Forgejo premise

Three different mechanisms were being conflated:

| Mechanism | What it actually permits |
| --- | --- |
| Templates, CSS and browser JavaScript | Change presentation and run client-side integration against available interfaces. |
| REST APIs, webhooks, Git hooks and CI | Integrate external programs at specified boundaries. |
| In-process server plugins | Add executable application modules, routes, native-session-aware handlers and extension implementations inside the running application. |

Forgejo offers the first two. They do **not** amount to a supported general-purpose server plugin system for Soda's application-level extensions. Modifying and rebuilding Forgejo can add arbitrary functionality, but that is downstream source maintenance—the approach Soda explicitly rejected.

The current Forgejo v15 administrator documentation is stronger than Soda's wording: custom resources/page modifications are **“unsupported”**, and updates may break them **“without any warning.”** It calls template changes the most dangerous customization type because incompatible changes occur regularly. Documented configuration settings have a different, stronger compatibility commitment. [R2]

Therefore:

- It was wrong to present Forgejo UI customization as the third-party server-extension capability on which to base this product.
- It was also too generous to describe the customization as merely a supported interface with occasional version sensitivity. There is an official mechanism, but upstream explicitly disclaims support for these modifications.
- “Forgejo cannot execute any third-party code at all” would be too broad: hooks/actions/external integrations exist. The missing capability is the **application plugin boundary**, not all extensibility whatsoever.

The cost is visible in Soda's implementation: native pages hosted through dashboard query selectors, a separate OAuth identity/grant layer, coordinated rather than atomic cross-system logout, and version-sensitive presentation overrides. These are real implemented integrations, but they do not establish the originally assumed plugin platform.

The factual support description in the [customization guide](forgejo-frontend-integration.md#shared-presentation-components) is corrected with this research. Existing selected routes and security contracts are not changed by that correction.

## 3. Does OneDev genuinely load third-party server code?

**Yes. This is source-verified, not inferred from a marketing claim.**

### Loading and packaging

OneDev's pinned bootstrap loads libraries from the normal installation library directory and **`site/lib`**, including bundled plugin libraries. It constructs the application classloader from them. The product source even ships a `site/lib/put_custom_plugins_and_libs_here` marker. [R3]

The loader then:

1. Reads `META-INF/onedev-plugin.properties` from classpath artifacts.
2. Resolves the declared Java module class.
3. Requires it to extend `AbstractPluginModule`.
4. Orders modules by declared dependencies.
5. Composes their Google Guice bindings using `Modules.override`.
6. Starts the plugin manager and plugin lifecycle. [R4]

`AbstractPluginModule` exposes `contribute(...)` for extension implementations and provides plugin lifecycle integration. This is executable JVM application code, not CSS injection, webhook forwarding or a renamed configuration file.

The repository also contains a Maven archetype with an **external group/artifact branch**. It generates a separate plugin project referencing a selected OneDev product version and declaring its own module class. That is direct evidence that plugins are not restricted to modules authored inside OneDev's repository. [R5]

**Practical implication:** a Soda-owned plugin can in principle be built outside the OneDev tree and loaded into an unchanged upstream installation. A derived packaging image containing that plugin is not the same thing as maintaining patches to OneDev's source. The exact packaged startup still needs to be demonstrated.

### What this does not promise

- No verified hot-install/hot-unload workflow: plan for restart-based deployment.
- No plugin sandbox: these modules share the application process, classpath and available authority.
- No verified stable, independently versioned plugin API/ABI or LTS compatibility guarantee.
- No demonstrated broad maintained third-party marketplace comparable to Jenkins or VS Code.
- No native Go or TypeScript server-plugin ABI. This is a **Java/JVM** extension model.

A bad plugin can break startup, conflict with dependencies, bypass intended checks or compromise the forge. Trusted installation and upgrade testing are part of the cost of getting this power.

## 4. The actual extension surface, with Soda-specific limits

| Soda need | Inspected OneDev mechanism | Finding |
| --- | --- | --- |
| Real native pages and routes | `WebApplicationConfigurator`; Wicket page/resource mappers | **Strong fit.** Built-in plugins mount real pages and downloads this way. No dashboard-query host workaround is necessary for a new page. |
| Repository navigation | `ProjectMenuContribution` | **Strong fit.** `ProjectPage` actually calls contributions with the native project and merges menu items. |
| Project settings | `ProjectSettingContribution`, `ContributedProjectSetting` | **Real extension.** Built-in settings hosting is gated by native project-management permission. That is not automatically the right gate for every Soda member view. |
| Global administration extensions | `AdministrationMenuContribution`, `AdministrationSettingContribution` | **Real extension, wrong default audience for some Soda operations.** Core renders them within its site-administrator branch. |
| Global Spaces/operator navigation | `MainMenuCustomization` and Guice binding overrides | **Needs a focused prototype.** It is a replacement customization interface, not an inspected additive global-menu contribution equivalent to the project menu. |
| Authenticated backend endpoints | `ServletConfigurator`, `JerseyConfigurator`, `FilterChainConfigurator` | **Strong capability.** Authentication, route order, CSRF and actual endpoint authorization still have to be implemented correctly. |
| Current native actor and permissions | Shiro subject and `SecurityUtils`, native services | **Major improvement.** A server plugin can consult the real request subject rather than trusting a user ID emitted into HTML. |
| Session destruction notifications | `SessionListener` wired to Jetty `HttpSessionListener` | **Useful real seam.** Potential for server-side terminal revocation, but not proof of atomic logout across a separate Go service. |
| Build/report integration | `BuildTabContribution`, report pages, job executors | **Mature first-party usage.** The unit-test plugin contributes tabs, navigation, routes and storage synchronization. |
| Repository/issue/event automation | `GitPreReceiveChecker`, `@Listen`/listener registry, import contributions, scripting | **Real backend integration.** Choose the native event/extension interface rather than polling when it fits. |
| Custom development runtime | `WorkspaceProvisioner` | **Real extension.** Its lifecycle is not automatically compatible with Soda's lasting shared roots. |

The strongest concrete exemplar is `UnitTestModule`: it contributes a project menu, build tabs and Wicket routes, with native permission filtering. `WebApplication` consumes the configurators; `ProjectPage` consumes the menus. This is a functioning upstream architecture, not an unused interface discovered by name. [R6], [R7], [R8]

### Two traps to avoid

**Operator ≠ site administrator.** Soda deliberately distinguishes appliance operator, forge site administrator, repository owner and project member. Moving Runners/Tailnet into OneDev's native administrator extension area must not turn every forge administrator into an appliance operator, or force the Soda operator to become a site administrator. Use a dedicated plugin page with the correct policy and solve navigation separately. [R7]

**Not every UI position has an equally good seam.** A project sidebar entry is directly supported by a contribution. An exact repository-header button/side drawer or additive global navigation has not been proved equally straightforward. `MainMenuCustomization` is resolved as one binding; the public tree does not contain its default implementation. A blanket promise that every part of OneDev's shell can be extended additively would repeat the original mistake.

The prototype should prove the desired placements without replacing native authentication pages, copying an entire shell or using classpath shadowing as a disguised source fork.

## 5. Developer experience and maintenance reality

### Technology changes

The inspected release uses:

- Java **17**, Maven and Google Guice.
- Apache Wicket **7.18.0** for the native server-rendered UI.
- Jetty **9.4.57.v20241219** and Apache Shiro **1.13.0**.
- Hibernate-backed persistence; default embedded HSQLDB.
- OneDev commons **4.2.1** and agent **3.2.10**. [R9]

These are research baselines, not new Soda dependency selections.

A plugin does **not** require rewriting Soda's backend in Java. It does require a small Java-facing integration owner if we want the benefits of native server extensions. Go remains appropriate for the host/helper/setup/project/runtime code. The existing Go-first repository policy would need an explicit, bounded decision for this adapter, not a quiet language-policy violation.

Lit/TypeScript can remain the implementation of Soda-owned interactive views. A Wicket page can supply the shell/mount and load the compiled assets. But Forgejo's templates, Fomantic assumptions, selectors and module initialization cannot simply be reused as OneDev UI.

### Documentation drift is concrete

The current “Build From Source” guide still says **JDK 11**. The actual parent POM specifies compiler release **17**, the distribution installs a Java 17 runtime, and the incompatibility notes say Java 17 became required in 13.1.0. The external plugin archetype still contains Java **1.8** source/target defaults. [R5], [R9], [R10]

That does not disprove plugin support. It does mean copying an old plugin tutorial/archetype verbatim is not a verified setup recipe for 16.6.2. Use current artifacts and fix the generated plugin's own build configuration; do not assume a no-maintenance SDK.

An independent Redmine importer demonstrates external plugins exist, but its POM targets OneDev **6.3.10**, not the current release. It is evidence of the mechanism, not a current supported plugin ecosystem. The public development-guide sitemap exposes a build guide, not a comprehensive current plugin compatibility contract. Attempts to query public plugin discussions through OneDev's REST endpoint returned HTTP 406; no maintainer support promise is inferred from that failed lookup. [R11]

### Governance and ecosystem risk

OneDev is actively developed and dogfoods its own forge; an empty GitHub issue count is not abandonment. Development is vendor-led and visibly concentrated: the retrieved GitHub contributor snapshot attributes 4,226 commits to `robinshine`, versus 188 to the next listed contributor. That is a narrow mirror statistic, not a count of current maintainers or a complete bus-factor assessment. The proprietary module, vendor-hosted Maven repository and concentrated ownership make continuity/support and reproducible dependency availability important adoption questions. Do not assume Forgejo's community governance or a large third-party plugin support network transfers with the product. [Contributor metadata](https://api.github.com/repos/theonedev/onedev/contributors?per_page=8)

### Maintenance considerations, not an upgrade project

A smaller native plugin still needs to compile and work with the OneDev version we select. The first proof targets that version only; it does not require an adjacent-version upgrade rehearsal or a compatibility matrix.

OneDev's incompatibility history records changed contributed-setting formats, removed PR review resources and an older workspace-data format reset. These are useful maintenance facts, not work to reproduce historical upgrades for nonexistent customers. [R12]

## 6. Native authentication and authorization

### Why the plugin helps

Soda currently has a second browser session backed by Forgejo OAuth grants. Its frontend passes an expected native-user ID as a consistency guard, and the backend independently uses the acting grant. The guides correctly admit that this is not proof of the live native browser session and that coordinated logout is not atomic.

Inside OneDev, a plugin handler can use the application's own authenticated request subject. Native project permissions and actor identity no longer have to be reconstructed from DOM context plus a separate OAuth round trip. This is a real architectural advantage.

### OneDev SSO is not a drop-in replacement for Forgejo OAuth

The inspected `OpenIdConnector` authenticates **OneDev against another identity provider**. It is an OIDC client. The `SsoProviderResource` manages those external providers. This does not demonstrate that OneDev itself offers Forgejo-compatible OAuth authorization/token/introspection endpoints to Soda. [R13]

No equivalent general OAuth-provider flow was identified in the reviewed public implementation/documentation. Therefore `internal/forgejo/client.go`'s application creation, authorization-code exchange, refresh and scope handling cannot be ported by changing URLs.

Native access tokens exist, with owner-permission or project-role authorizations and optional expiry. But asking every user for a personal token, or using one appliance administrator token for all operations, would be a worse substitute for native-session integration. [R14]

**API nuance:** OneDev's REST chain uses `noSessionCreation, authcBasic, authcBearer`; the authentication filters skip token login when a subject is already authenticated. `noSessionCreation` alone does not mean existing browser sessions are ignored. This review does not claim the blanket “REST can never use cookies” limitation that applies to Soda's inspected Forgejo API. Cookie/CSRF behavior for the exact new endpoint still requires tests. [R15]

### Recommended boundary if OneDev is selected

```text
Browser
  → OneDev native session + Soda Java plugin pages/endpoints
    → private, narrowly authenticated calls to Soda's Go backend
      → existing restricted Go host helper
        → project accounts, tmux, Podman and systemd
```

Proposed ownership:

- OneDev remains the human identity, native session, Git and collaboration authority.
- The plugin resolves native actor/project and performs the appropriate native checks.
- Soda retains appliance-operator policy, environment membership/account bindings, local lifecycle and terminal admission.
- The Go backend accepts only the trusted integration caller, not a browser-supplied identity header. Keep the trust boundary explicit and inaccessible directly from public ingress.
- No OneDev database queries from Go, borrowed cookies, permissive CORS, root Docker socket, or universal admin token.

Whether the existing browser Go API remains behind a bounded plugin gateway or a small set of UI-facing calls moves into the plugin should be decided by the prototype. Do not build a new OAuth identity provider or generic federation layer merely to preserve the old adapter shape.

### Authorization must remain explicit

OneDev's own `development.md` says service methods generally **do not check permissions** unless they take a subject; a `User` argument can mean attribution only. They generally do not audit unless stated. Native service access is not permission enforcement. Plugin handlers must use the actual subject and correct native checks before side effects. [R16]

OneDev uses a hierarchical project tree with user/group/role authorizations rather than Forgejo's user/organization repository ownership. Define the new integration's native checks deliberately, retaining the distinction between appliance operator and project authority. This is authorization design for fresh users/projects, not migration of an old role inventory.

### Logout, revocation and factors

`WebSession.logout()` logs out the native subject and replaces the session. Product servlet configuration dispatches actual HTTP session-destruction events to contributed listeners. That creates a credible path to server-side Soda terminal cancellation, stronger than intercepting a browser logout link. [R17]

Still unproven: cancellation ordering across the Go boundary, session rotation, suspended tabs, concurrent opening/ending, transport failures, server restart, permission loss and remember-me behavior. Do not promise atomic cross-service logout until those races are demonstrated.

OneDev documents TOTP/recovery-code 2FA. Native WebAuthn/passkey support was **not identified** in the inspected public code/dependencies/docs. That remains a feature-selection question if native passkeys are required; it is not a reason to build a second identity provider for the first plugin proof. There are no customer passwords, factors or sessions to migrate. [R18]

## 7. Built-in OneDev workspaces and AI: useful, but not Project OS parity

Current OneDev is broader than an older “Git + CI + Kanban” comparison would suggest. Its official docs describe browser terminals, per-ref workspaces, command shortcuts, exposed development ports, user data, coding-agent templates and AI task automation. Community Edition includes server Docker/shell provisioners. Remote and Kubernetes workspace provisioners require Enterprise. [R19]

This may reduce the amount of **future** Soda AI integration we need. It does not justify replacing completed Project OS behavior with an apparently similar screen.

| Requirement | OneDev native workspace | Current Soda contract |
| --- | --- | --- |
| Unit of ownership | Workspace bound to project, user, ref/spec | Shared persistent environment linked to a repository |
| Human accounts | Container run-as or server/agent process privileges | Multiple actual project-local Linux accounts/homes |
| Persistence | Workspace lifecycle and selected reusable user data | Whole mutable root, tools, accounts, SSH keys and service data |
| Browser closure | Docs say workspace survives closing/reopening browser | Managed session retention plus independent SSH/runtime lifetime |
| Server maintenance | Workspace service cancels tasks when OneDev stops | Forge web-service maintenance must not imply recreating project roots |
| External access | Built-in terminal and port links | Ordinary SSH/PTY/SCP/SFTP, native services, planned Tailnet |
| End/delete | Native workspace cleanup/user-data handling | Deleting lasting projects is not selected scope |
| Desktop | No demonstrated complete Soda KDE/account/Lock equivalent | Planned Project OS desktop profiles and per-account sessions |

Source matters here: `DefaultWorkspaceService` sets a workspace's user/ref/spec, cancels running workspace tasks on `SystemStopping`, and delegates cleanup on entity removal. `WorkspaceProvisioner` explicitly includes provisioning and deletion methods. That is not an assertion that all OneDev workspace data is disposable on browser close; it is evidence that the lifecycle differs. [R20]

The docs say configured reusable user data is saved after workspace deletion. Source shutdown comments describe uploads during stop as well. If native workspaces are selected later, check that lifecycle then; it is not established Project OS parity. The older 15.1.0 workspace format transition also deleted saved user data. [R12], [R19], [R20]

### Recommendation

Initially keep Soda's existing Project OS as an independent runtime and use OneDev's plugin surface to integrate it. Separately evaluate native workspaces for disposable/AI tasks where their lifecycle fits.

A custom `WorkspaceProvisioner` is a legitimate later option, not a compulsory first step. Reusing it is only a win if native creation/deletion, user ownership, scheduling and shutdown semantics can preserve Soda's contracts without a second synchronized lifecycle manager.

### AI authorization needs special review

The source `BearerAuthenticationFilter` maps an active native workspace token to the workspace user's subject. It does not demonstrate a project-only, read-only execution capability. An AI workspace token must not be assumed to satisfy Soda's planned isolated read/review versus trusted publication boundary. Use dedicated minimal-permission automation identities and prove the actual access limits. [R15]

Built-in AI users may help with issue assignment, PR review/fixes and failed-build investigation. They have not been shown to implement Soda's exact bounded resolver/reviewer loop, real named-terminal projection, account isolation or desktop computer-use requirements. Inspect those workflows when the corresponding feature is selected. Completing AI, desktops or a custom workspace provisioner is not a prerequisite for the initial forge/plugin integration.

## 8. CI/Runners: useful native functionality, incompatible execution platform

OneDev has its own scheduler, build-spec format, job executors, agents, caches, reports and artifacts. Its native build spec is **`.onedev-buildspec.yml`**, not Forgejo Actions YAML. Native CI agents are included in CE; do not confuse them with EE-only **remote workspace** provisioners. [R21]

Consequences for Soda:

- Replace the Forgejo-specific runner integration with OneDev agents; do not adopt old registrations or convert their descriptors.
- Author OneDev jobs for the workflows Soda actually needs. There is no customer workflow/secret/history inventory to translate and no general Actions-to-OneDev converter to build.
- Reuse useful shell scripts, tools and job-image work. The scheduler/executor integration changes.
- Reuse applicable local lifecycle locking, dedicated-account and root-only management code. Native provider-online/busy state should come from OneDev, not a duplicated scheduler.

OneDev's agent guide says the server pushes agent updates automatically. When implementing runner packaging, inspect that behavior against Soda's pinned artifacts; an automatically changing client cannot silently be called the same sealed artifact. Agent packaging is not a prerequisite for proving a native plugin page. [R21]

### Do not copy the quickstart's trust model

The Docker quickstart mounts the host Docker socket into the forge server. Server-shell jobs run with the server process's privileges. These are easy ways to make demos work, but not acceptable shortcuts around Soda's restricted web/helper design. [R22]

Preferred investigation: keep the forge server without an appliance engine socket; place execution on separately confined agents/capacity, and verify the selected executor. Docker/buildx-oriented code and an OCI-compatible runtime do not prove compatibility with **rootless Podman on CoreOS**. No native Podman executor was identified in the inspected public plugin inventory. Socket compatibility, services/networks, cancellation, process cleanup, storage and SELinux all need real proof.

## 9. Licensing, cost and distribution: do not stop at the MIT badge

The public repository root is MIT. However, the **distribution license** explicitly distinguishes:

- The `io.onedev.server-ee*` module: commercial license.
- Other files: MIT **unless separately licensed**.
- The EE module ships with the binary distribution, with subscription-only functionality and restrictions on decompiling/modifying/reusing it. [R25]

Also, `server-product/pom.xml` directly depends on `server-ee`. The `ce` profile excludes that module from the source reactor, but it does not remove the product dependency. **`mvn package -Pce` must not be described as proof of an entirely EE-free, fully source-rebuildable distribution.** It can consume the prebuilt EE artifact. The public tree has an EE submodule reference, but attempts to read its source/license at the pinned submodule revision led to sign-in; the source-JAR request returned 404. The applicable publicly inspected distribution terms are in `server-product/system/license.txt`. [R9], [R25]

This is not a claim that CE requires a paid subscription, or that a Soda plugin must be proprietary. It is a warning that **free-to-use, MIT core, and entirely open-source shipped product are different statements**.

### Current price and relevant edition split

Official pricing on the research date: **CE free; EE $6 per user/month**, minimum order 12 user-months. Disabled users, service accounts and AI users are excluded from the stated license calculation. This is published pricing, not a cost for an existing Soda user base or a requirement to adopt EE. [R26]

| Relevant feature | Advertised edition |
| --- | --- |
| Git/collaboration, CI engine, agents, package registries | CE |
| LDAP, OIDC SSO, 2FA, custom issue workflows | CE |
| AI users/task automation | CE; model/provider costs are separate |
| Server Docker/shell workspaces | CE |
| Remote/Kubernetes workspace provisioners | EE |
| Shared custom dashboards, cross-project code search | EE |
| Audit log, time tracking, security/compliance scan | EE |
| CI debugging web terminal | EE; not the same as workspace terminal access |
| Account disabling while preserving history | EE |
| HA/scalability and separate LFS/artifact/package storage | EE |

Brand name and light/dark logo changes are implemented in the inspected public native branding page, without a subscription check there. That is useful, but is not legal clearance for every white-label/OEM distribution choice or permission to modify the EE module.

Before distribution, confirm the selected bundle's redistribution, branding and offline-use terms and required notices. No EE feature set or commercial support programme is selected. Do not work around EE restrictions by overriding license checks. If a completely open-source distributable base is non-negotiable, this licensing/package fact is an immediate decision point.

## 10. Operations and security

### Appliance fit

- Official docs give a **2 GB minimum** and a 2-core/2-GB small-install example. That is not a measured Soda capacity plan, especially with workspaces, indexing, CI, Cockpit and multiple Project OS roots sharing the host. No comparative benchmark was run. [R22]
- Published `1dev/server:16.6.2` metadata includes **linux/amd64 and linux/arm64**. Its observed manifest digest is `sha256:decb70d84571e472d5e903ef0b7ad513d7f0bf26cee8ee7d9f3df42771968f9f`. These are available artifacts, not native x86_64/aarch64 proof. No image was pulled or executed. [R27]
- Native web/SSH defaults are 6610/6611. Caddy reverse proxying is documented. Existing Soda origin, Git advertisement, cookies, WebSockets and streaming need their own tested mapping. A public domain is not inherently required; the quickstart uses localhost. Do not turn proxy examples into a domain-purchase requirement. [R22], [R28]
- The default database is **HSQLDB**, not Forgejo/Soda SQLite. PostgreSQL, MySQL and MariaDB configurations are supplied. No new external database is automatically required, but backup/restore procedures are different. [R29]
- Upstream backup documentation separates the database from repository/attachment/site data. Use native mechanisms when needed; a bespoke paired Forgejo-to-OneDev backup/rollback system is not part of this replacement. [R30]
- The official image runs its entrypoint as root and updates a writable installation under `/opt/onedev` before starting. This is not a drop-in fit for Soda's capability-dropped read-only Go container. Review the real container identity/filesystem requirements rather than assuming equivalent confinement. [R31]

### Specific security findings worth considering

1. **Full-trust plugin boundary.** JVM plugins are not safely installable by arbitrary repository owners. Restrict `site/lib` writes to the appliance's trusted deployment path.
2. **Permissions are not automatic in service calls.** Follow the upstream service-method warning and native subject checks. Do not implement authenticated-but-unrestricted adapters.
3. **Execution separation remains essential.** A Docker socket or server-shell executor can collapse the separation between repository-controlled jobs and forge/host authority. Rootless alone is not complete job isolation.
4. **Workspace token scope is material if used for automation.** The observed user-subject mapping needs deliberate automation-user permissions and negative cross-project tests when that feature is implemented.
5. **Recent vulnerabilities exist and have fixes.** Upstream reports CVE-2026-44647, LFS pointer path traversal/read, fixed in 15.0.2; and CVE-2026-49248, archive symlink write/RCE, fixed in 15.0.7. The researched 16.6.2 is newer than those fixed versions. These advisories do not establish that 16.6.2 is affected, nor that it is comprehensively safe. [R32]
6. **Dependency maintenance needs an answer.** The inspected POM pins Jetty 9.4.57; Jetty's official page marks the 9.4 line EOL/unsupported and recommends its newer supported line. Wicket/Shiro versions also merit a current dependency review. Version age is not an exploit demonstration, but it is a real maintenance question for an appliance identity server. [R9], [R33]
7. **Private LAN is not a security waiver.** A forge processes untrusted repository contents, build artifacts and browser input even on a trusted team's private network.

This was a source/design review plus advisory lookup, **not a penetration test or full dependency/SBOM vulnerability audit**. CVE counts cannot be used to rank Forgejo and OneDev security because exposure, reporting and maintenance differ.

## 11. Reuse code, not a legacy compatibility layer

At the baseline, `appliance/forgejo/templates/` contains **253 tracked templates, 17,203 lines**. That is an inventory, **not a list of pages to rebuild**: use OneDev's native collaboration/account/settings pages and port only Soda-owned features and branding. The existing Lit workspace and Go runtime are useful code, not an obligation to preserve every Forgejo-specific interface.

| Area | Replacement treatment |
| --- | --- |
| CoreOS/Podman/systemd/Cockpit foundation | Reuse; check changed service placement/confinement. |
| Project OS/accounts/SSH/mise/workloads | Reuse implementation and product behavior; no conversion of retained development roots. |
| Go host helper and terminal/account mechanisms | Reuse with a correctly authenticated new caller and fresh account bindings. |
| Soda application state | Initialize fresh users, projects and memberships; no legacy ID/schema/configuration compatibility layer. |
| Lit workspace and project/terminal controls | Reuse with OneDev shell, assets and session integration. |
| Forgejo OAuth client and browser coordinator | Replace directly; no old-session, callback or bookmark compatibility requirement. |
| Forgejo templates/themes/navigation adapters | Retire the coupling when replacing the integration; do not reproduce Forgejo inside Wicket. |
| Branding/artwork/licenses | Reuse original assets and preserve their notices. |
| Runners | Implement fresh OneDev capacity integration, not registration/workflow/history migration. |
| Setup/Caddy/staging/build inventories | Wire the new installation path and actual artifacts; no retained-target cutover branches. |
| Tests | Reuse applicable authorization, persistence and terminal scenarios; change affected native-page callers rather than reproduce all Forgejo UI tests. |
| CLI tools/examples | Keep Git; update only Soda-owned Forgejo-specific usage that is still needed. |
| Tailnet/Services/AI/desktops | Retain their feature ownership; completing these roadmaps does not gate the first plugin proof. |

**Effort:** the customer-migration estimates are withdrawn. No OneDev implementation work or schedule is selected.

## 12. Conditional integration outline — not active work

These options apply only if the [forge decision](architecture.md#forge-selection) is reopened. They are not a current task list or execution approval. The [handoff](implementation-status.md#current-permissions) continues to govern fixture, host and provider effects.

### First: prove the native plugin boundary

Use one separately packaged plugin against the selected stock OneDev version, fresh application state and synthetic users/projects. No Forgejo source fixture or data importer is needed.

Demonstrate:

- A new native page with a project navigation/settings entry and one existing Soda Lit component.
- The actual native actor and correct operator/project visibility, including the global menu placement that remains uncertain.
- A small private Go call using that native session: test the allowed case and anonymous/unauthorized/direct-backend rejection. If it mutates, check CSRF; session expiry/logout must not leave that caller authorized as the old actor.

That settles the missing platform capability. Do not require multiple OneDev versions, both architectures, fleet upgrades, a load benchmark, CI provider jobs or completed AI/desktop features before this proof. If it needs a core fork or copied native shell, report that concrete blocker rather than adding more workaround infrastructure.

### Then: replace the actual Soda integration

If OneDev is selected, wire the real Go backend, reusable Lit views and fresh account/project creation to the proven plugin boundary. Replace obsolete Forgejo-specific source, assets, configuration and tests together; do not carry two providers for compatibility. Keep native OneDev authentication and collaboration rather than porting Forgejo's UI.

Use the existing test owners to cover changed authorization, project persistence and terminal lifetime. Validate native runtime/runner behavior on the architecture and executor actually being implemented; those checks belong with their feature, not a separate all-product prerequisite.

Selected development repositories or files can be copied once if useful. That is optional data handling, not a deliverable requiring a general importer, identity remapper, historical PR archive or migration rehearsal suite. Do not point fresh application state at old fixture account bindings or erase the old fixtures to make tests pass.

## 13. Decisions only if OneDev is reconsidered

- **Platform proof:** current plugin build recipe, native menu placement and session-to-Go authorization. Demonstrate the exact interfaces instead of requiring a broad vendor support guarantee first.
- **Product selection:** whether the mixed-license distribution and bounded Java plugin layer are acceptable. Native passkeys remain a feature question if required, not a factor-migration task.
- **When implementing runners/workspaces:** actual Podman/executor compatibility, client update behavior and task credential isolation.
- **Before distribution:** applicable licensing, dependency security and native build/release checks. No HA/fleet-support programme is selected.

**Current decision:** retain Forgejo. This evaluation does not select a JVM dependency, a forge replacement, a custom fork or a new frontend.

## Sources

[R1]: https://github.com/theonedev/onedev/releases/tag/v16.6.2
[R2]: https://forgejo.org/docs/v15.0/admin/advanced/customization/
[R3]: https://code.onedev.io/onedev/~maven/io/onedev/commons-bootstrap/4.2.1/commons-bootstrap-4.2.1-sources.jar
[R4]: https://code.onedev.io/onedev/~maven/io/onedev/commons-loader/4.2.1/commons-loader-4.2.1-sources.jar
[R5]: https://github.com/theonedev/onedev/tree/1c84e0b18e31364a933d9b6503e635008d989e51/server-plugin/server-plugin-archetype/src/main/resources/archetype-resources
[R6]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-plugin/server-plugin-report-unittest/src/main/java/io/onedev/server/plugin/report/unittest/UnitTestModule.java
[R7]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/web/page/layout/LayoutPage.java
[R8]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/web/page/project/ProjectPage.java
[R9]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/pom.xml
[R10]: https://docs.onedev.io/development-guide/build-from-source
[R11]: https://github.com/DevCharly/onedev-plugin-import-redmine/blob/main/pom.xml
[R12]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-product/system/incompatibilities/incompatibilities.md
[R13]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-plugin/server-plugin-sso-openid/src/main/java/io/onedev/server/plugin/sso/openid/OpenIdConnector.java
[R14]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/model/AccessToken.java
[R15]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/security/BearerAuthenticationFilter.java
[R16]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/development.md
[R17]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-product/src/main/java/io/onedev/server/product/ProductServletConfigurator.java
[R18]: https://docs.onedev.io/tutorials/security/working-with-2fa
[R19]: https://docs.onedev.io/tutorials/workspace/working-with-workspaces
[R20]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/workspace/DefaultWorkspaceService.java
[R21]: https://docs.onedev.io/tutorials/cicd/agent-farm
[R22]: https://docs.onedev.io/installation-guide/run-as-docker-container
[R25]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-product/system/license.txt
[R26]: https://onedev.io/pricing
[R27]: https://hub.docker.com/v2/repositories/1dev/server/tags/16.6.2
[R28]: https://docs.onedev.io/administration-guide/reverse-proxy-setup
[R29]: https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-product/system/conf/hibernate.properties
[R30]: https://docs.onedev.io/administration-guide/backup-restore
[R31]: https://github.com/theonedev/onedev/tree/1c84e0b18e31364a933d9b6503e635008d989e51/server-product/docker
[R32]: https://github.com/theonedev/onedev/security/advisories/GHSA-55g8-94r5-cj37
[R33]: https://jetty.org/download.html

Additional exact implementation and research references:

- [Published parent POM 1.3.0: Java 17 compiler release](https://code.onedev.io/onedev/~maven/io/onedev/parent/1.3.0/parent-1.3.0.pom).
- [Product dependency on the EE module](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-product/pom.xml).
- [Global main-menu customization interface](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/web/page/layout/MainMenuCustomization.java).
- [Wicket extension consumption and AJAX lifecycle](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/web/WebApplication.java).
- [Core REST/authentication/extension wiring](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/CoreModule.java).
- [Basic authentication filter](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/security/BasicAuthenticationFilter.java).
- [Native web-session logout](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/web/WebSession.java).
- [Native permission helpers](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/security/SecurityUtils.java).
- [Workspace provisioner extension interface](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/model/support/administration/workspaceprovisioner/WorkspaceProvisioner.java).
- [Native branding page](https://github.com/theonedev/onedev/blob/1c84e0b18e31364a933d9b6503e635008d989e51/server-core/src/main/java/io/onedev/server/web/page/admin/brandingsetting/BrandingSettingPage.java).
- [LFS read advisory, CVE-2026-44647](https://github.com/theonedev/onedev/security/advisories/GHSA-59wq-74xg-w85v).
- [NVD: archive symlink advisory](https://nvd.nist.gov/vuln/detail/CVE-2026-49248); the retrieved NVD keyword result also includes recent fixed authorization issues. Not a complete vulnerability assessment.
- [Jetty release/support status](https://jetty.org/download.html); captured page marks 9.4 EOL/unsupported.

Reference labels in the text link to these primary sources. Source-JAR citations are downloadable official source archives; the inspected Java excerpts remain in the local evidence directory. Failed EE/source/discussion retrievals are retained as failures, not used as evidence of inaccessible implementation details.

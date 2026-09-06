# Source-first implementation plan

**Status:** This is the original source-first plan; actual completion and native evidence are recorded in [implementation status](implementation-status.md). No milestone is marked complete merely by writing a plan.

**Follow-up tooling:** The [native artifact and acceptance porting plan](native-porting-plan.md) proposes selective predecessor reuse, CoreOS media delivery and x86_64-first execution support for M15–M18. Its P01–P13 milestones are planned, not implemented, and do not authorize execution or a release/Updates platform.

**Governing scope:** [Architecture](architecture.md), [deferred and excluded work](deferred.md), [branding](branding.md) and the [existing asset inventory](../assets/README.md).

**Goal:** Implement the complete current SodaOS product in source before the x86 machine becomes available. Build artifacts and execute validation later. “Complete” includes the dashboard, native integration, Soda Project OS, deployment/provisioning source, and the retained Tailnet and Runners Cockpit features—not only a frontend or a set of interfaces.

## 1. Execution contract

### Now: source implementation

- Implement real application behavior, native command integration, templates, configuration, image recipes, service definitions and installation/provisioning source.
- Read upstream documentation/source to choose concrete versions and supported interfaces. Lack of an x86 machine is not a reason to leave unrelated application code unimplemented.
- Port selected predecessor code with its necessary callers, dependencies and tests. Do not copy the predecessor's entire architecture or build/release system.
- Write or adapt test source alongside each feature. Tests are prepared now and **executed later**.
- Review source and diffs; keep implementation and documentation coherent. This is not runtime validation evidence.

### Later: execution

Do **not** run builds, compilation/type-check pipelines, unit/integration/browser/native tests, image generation, artifact publication, installation, live enrollment, runner registration or machine restarts during the source-only phase. Do not start an automatic CI workflow that performs those operations on an agent's behalf.

Writing a build script is source work; invoking it is a later operation. Writing input validation, authorization and error handling is implementation work; deferring test execution is not permission to omit those behaviors.

### Meaning of source completion

A milestone is **source-complete, unbuilt and unvalidated** when its production behavior and callers are implemented, its necessary configuration/packaging source is present, and its test source and instructions are written. A fake backend, mock-only success path, unwired handler, missing native helper or documentation-only command does not complete a production feature.

Generated bundles, binaries and images are intentionally absent until the build phase. Record their source recipes and expected installation destinations; do not fabricate generated artifacts, dependency hashes or test results.

Native assumptions can still prove wrong later. Localize those assumptions in concrete runtime/configuration code, implement one source-backed candidate, and keep the rest of the product moving. Do not solve uncertainty by building a generic backend framework or promising that unexecuted code works.

## 2. Scope and choices to settle while implementing

The baseline is:

- immutable CoreOS-style host; Fedora CoreOS is the leading candidate;
- persistent Rocky + mise development environments, with real project-local Linux accounts;
- direct `ssh user@project-ip` and ordinary SSH commands, SCP and SFTP;
- shared installed tools/files/services and ordinary personal repository checkouts;
- Forgejo-backed human identity, with its associated project/repository owner as the working project-administrator rule;
- a Go + HTMX Soda dashboard and its own product database;
- operator-only Cockpit, including the reused Tailnet and Runners pages and backing logic;
- source support for x86-64 and AArch64, with native execution later on each architecture.

Choose the following engineering details during their owning milestones rather than implementing every alternative. Record the concrete choices in the architecture or owning source documentation when made; this plan does not silently select an engine, release or filesystem layout.

| Choice | Required source decision | Owner |
| --- | --- | --- |
| Dependency baseline | Concrete Go/frontend versions, relevant predecessor revisions and source dependency declarations | M01 |
| Host baseline | One source-backed CoreOS-style host/release candidate for the native integrations; packaging follows that candidate | M01, M13 |
| Soda database | One engine/driver and ordinary schema/migrations, not a database-backend framework | M02 |
| Forgejo integration | One release, native provisioning/API/OAuth interfaces, browser URLs and operator bootstrap flow | M03 |
| Project userspace | One Rocky/mise baseline, initialization model, filesystem layout and persistence strategy | M05, M08 |
| Host/runtime boundary | Actual caller/helper transport, native execution ownership, project IP network and ordinary start/stop behavior | M06 |
| Project service execution | Preferred nested Podman implementation and one compatible native workload interface | M09 |
| Operator integrations | Cockpit/Tailscale/runner delivery, native permissions and service wiring on the selected host | M10–M12 |
| Appliance delivery | One complete upstream-compatible host provisioning and component installation path | M13 |

The tentative `~/shared` and `~/repo-name` layout is a useful starting point, not an instruction to install binaries in `/etc` or invent a new repository format.

For nesting, implement the preferred path on documented native assumptions. If source investigation already demonstrates a concrete blocker, use the project-scoped host fallback permitted by the architecture and record why. Otherwise, do not implement a dormant second runtime merely because later testing might find a problem. Neither choice is a runtime validation claim.

## 3. Work organization and order

### Suggested source ownership

These are organizational paths, not a requirement to create a service or abstraction for every directory. Retain useful predecessor file organization where it makes a port smaller.

```text
go.mod
cmd/                    dashboard/setup and only the required native helper executables
internal/               configuration, database, Forgejo, web and project/native logic
web/                    Go templates and dashboard static source
project-os/             Rocky image recipe, initialization and project-native configuration
appliance/              host provisioning, service definitions and installation configuration
cockpit/                retained Tailnet/Runners packages and their shared UI/native support
assets/                 existing canonical branding assets
scripts/                narrow build/staging/validation entrypoints, authored but not run now
tests/                  cross-component and installed-journey test source where needed
docs/                   implementation choices, operator instructions and execution handoff
```

Native transport and database choices must result in real implementations. Small interfaces for external commands, HTTP and test injection are useful; an abstract multi-cloud, multi-runtime or generic resource platform is not.

### Source milestone map

All milestones below start **not started**. Dependencies are source dependencies, not requirements to build or run earlier milestones.

| Milestone | Deliverable | Source dependencies |
| --- | --- | --- |
| M01 | Repository and application foundation | None |
| M02 | Soda database and product persistence | M01 |
| M03 | Forgejo service, integration and operator bootstrap | M01, M02 |
| M04 | Authentication, people, profiles and public-key dashboard | M02, M03 |
| M05 | Soda Project OS source | M01 |
| M06 | Host/runtime integration, persistence and direct-IP addressing | M01, M05 |
| M07 | Project creation, ownership and joining end to end | M02–M06 |
| M08 | Shared files, installed mise tools and ordinary development | M05, M06 |
| M09 | Native project service/workload execution | M05, M06, M08 |
| M10 | Operator Cockpit foundation and native delivery | M01 |
| M11 | Tailnet page and backing logic port | M03, M06, M10 |
| M12 | Runners page and backing logic port | M03, M10 |
| M13 | Complete appliance provisioning and component packaging source | M03–M12 |
| M14 | Product integration, execution entrypoints and source handoff | M01–M13 |

Recommended serial order is M01 through M14. With multiple implementers, M05/M06 and M10 can proceed alongside the database/authentication work; M11 and M12 can proceed independently once their dependencies exist. Coordinate shared configuration, command contracts, frontend dependencies and packaging files. Do not create competing implementations of them or overwrite another implementer's work.

## 4. Source implementation milestones

### M01 — Repository and application foundation

**Implement:**

- Establish the new Go module and application entrypoint, configuration loading, actual HTTP server composition, static/template loading and ordinary shutdown handling.
- Select a concrete source dependency baseline and upstream host/release candidate for later native integration code. Record host compatibility assumptions as unverified. Keep the Go + HTMX dashboard separate from the retained Cockpit frontend's implementation technology.
- Establish configuration for public browser endpoints versus service-internal endpoints, database location, runtime integration and secret-file inputs. No credentials in tracked examples or logs.
- Set up the small native process/HTTP boundaries needed by real implementations and test doubles. Do not add a generic privileged command API.
- Integrate the existing branding source into the initial dashboard shell; preserve required license/copyright notices for reused code.
- Record which predecessor revision/files will be used for Tailnet, Runners and their shared dependencies. Recheck the predecessor's current source before copying; the architecture's inspected revision is a reference, not a mandatory freeze.
- Add source development instructions explicitly separating implementation from later build/test commands.

**Test source:** configuration parsing, public/internal URL handling, template routing and the narrow external-command boundary.

**Source exit:** the application has a real composition root and source-owned configuration conventions for later milestones. Foundation code is not presented as a completed product.

### M02 — Soda database and product persistence

**Implement:**

- Select and wire one database engine/driver, connection lifecycle and ordinary schema initialization/migrations.
- Persist the required Soda profiles, stable Forgejo identity association, development-access public keys, projects, environment associations and memberships.
- Provide concrete query/mutation code used by the application, not a generic resource inventory or a provider-permission mirror.
- Implement normal input validation and database constraints for the supported create/read/update operations.
- Keep runtime facts distinct from product relationships: a row alone must not imply that account or environment provisioning succeeded.
- Supply the persistent database path and access requirements to deployment source as the database code is added.

**Test source:** schema initialization, persistence round trips, identity association and ordinary validation/query cases using the chosen engine.

**Source exit:** the store is implemented and connected to application startup. No private-resource selector schema or speculative lifecycle framework is added.

### M03 — Forgejo service, integration and operator bootstrap

**Implement:**

- Choose a Forgejo release and author its container/service configuration with persistent repositories, database and configuration, using native supported interfaces.
- Implement the actual Forgejo client operations required for identity lookup, operator-created users and the associated repository/owner relationship.
- Implement a concrete operator bootstrap path through trusted native console/setup access: establish Forgejo administration, the dashboard's OAuth client configuration and the authorized Soda operator identity. Host root credentials, Forgejo credentials and dashboard authorization remain distinct.
- Resolve the selected version's real OAuth capabilities/scopes from its documentation/source; do not assume OIDC or invent missing API endpoints. Use a bounded native operator setup step if upstream requires one, not a second Soda password system.
- Provide native browser and Git endpoints for containerized Forgejo. Git access follows normal Forgejo credentials/permissions; project SSH key registration does not silently register Git credentials too.
- Keep browser-facing URLs, internal service addresses and Tailnet-refresh integration configuration explicit, without importing the predecessor's fixed loopback port assumptions.
- Add service credentials/configuration inputs and their restricted storage; do not make live accounts or call a real authenticated Forgejo instance now.

**Test source:** native API payloads/responses, bootstrap command/configuration generation, identity/owner lookup and credential redaction.

**Source exit:** bootstrap and provider integration are real source paths with documented operator inputs, not fake identities or manual database edits standing in for onboarding.

### M04 — Authentication, people, profiles and keys

**Implement:**

- Wire Forgejo login, callback, identity association, sessions and logout into the actual Go + HTMX application.
- Implement normal session protections, OAuth request binding, CSRF protection and server-side authorization. Deferring edge-case work does not defer basic access checks.
- Implement profile display/editing and public SSH-key registration with actual persistence and usable form/error responses.
- Implement the operator-only create-person flow, invoking Forgejo and creating the associated Soda profile without pretending a failed provider call succeeded.
- Supply navigation and useful empty, pending and error states for the supported pages; keep secrets out of rendered pages and logs.
- Keep later key propagation, provider-disable synchronization, session termination across systems and complex ownership remapping outside this milestone.

**Test source:** handlers/templates, sessions, access denial, valid/invalid key input, user creation and normal provider failure reporting. Do not run the tests or a browser yet.

**Source exit:** login, operator people management, profiles and public-key registration have real frontend-to-provider/database code paths.

### M05 — Soda Project OS

**Implement:**

- Choose the Rocky release and author the project userspace image recipe, with mise, OpenSSH, Linux account tools, Git and the required development/runtime essentials.
- Implement the chosen initialization/process model for SSH and project-native services. A pod is not treated as an init system or Linux user database.
- Author the project-local account/home/group setup invoked by the host integration. Use native tools, preserve ordinary SSH commands/SCP/SFTP, and grant project administration through the selected native permissions rather than host accounts.
- Generate SSH host keys per project at runtime, retain them in persistent state, and exclude real user keys or credentials from the image.
- Implement one concrete persistence arrangement covering account records, homes, shared files, host keys, installed tools, system-package changes, service configuration and service data.
- Provide actual startup/configuration source for that arrangement. Do not rely on a writable container layer while also defining normal startup to replace and discard that container.
- Represent both CPU architectures through native upstream inputs without claiming either image has been built.

**Test source:** initialization/account command cases and later opt-in guest checks for SSH, filesystem layout and state preservation.

**Source exit:** the project OS is fully described by buildable source recipes and native configuration, with no guest agent required merely to replace ordinary account/SSH tools. Buildability remains unexecuted.

### M06 — Host/runtime integration and direct-IP projects

**Implement:**

- Implement the actual privileged boundary between the unprivileged dashboard service and the required host Podman/account operations. Include the client, helper/transport if needed, native service wiring and permission configuration together.
- Restrict that boundary to supported project operations. Resolve server-owned project identities and arguments; never expose arbitrary command strings, arbitrary host Podman flags or an unrestricted host socket to users.
- Implement native project environment creation, startup and inspection, with the storage/network configuration selected for M05. Use native runtime state for observed facts.
- Author the ordinary stop/start and boot-start definitions without silently replacing mutable project state. This is basic lifecycle wiring, not a disaster-recovery controller.
- Implement direct project-IP assignment and readback using one native network arrangement. Accept real deployment interface/subnet/address inputs; illustrative addresses in documentation are not live defaults.
- Include host-to-project and developer-to-project connectivity requirements in the native configuration. Do not add DNS, an SSH gateway or a general IPAM platform.
- Implement the bounded invocation of project-local account/key setup for M07, with native errors surfaced honestly.

**Test source:** command construction/parsing, project scoping, permission checks and generated service/network/storage configuration using test doubles.

**Source exit:** production callers can reach real, implemented native operations through the configured deployment boundary. A helper that only exists on the host but is inaccessible to the dashboard container does not count as wired.

### M07 — Project creation, ownership and joining

**Implement:**

- Build the project list/discovery, create and detail pages against actual database/provider/runtime implementations.
- Associate an environment with its Forgejo project/repository and implement the working owner-as-project-administrator rule for the supported ordinary owner case. Do not build organization/transfer policy machinery.
- Wire project creation through the real runtime integration and show the actual resulting project IP and operational result.
- Implement **Add me to this project** from browser request through account creation, public-key installation, home/shared-resource permissions and membership persistence.
- Cover Alice's explicit join as well as Bob's; do not depend on an unstated automatic creator enrollment.
- Display ordinary `ssh user@ip` connection guidance and distinguish project administration from operator-only host access.
- Handle normal native errors without a false joined/ready state. Do not add a generic retry/reconciliation engine or destructive rebuild shortcut.

**Test source:** create/join handlers and their concrete service integration through provider/runtime test doubles, project-owner authorization, missing-key guidance and resulting SSH information.

**Source exit:** project creation and joining are complete application-to-native code paths, not catalog-only features.

### M08 — Shared files and installed mise tools

**Implement:**

- Finalize and implement the shared-files access and home layout using native ownership/permissions and links or mounts. `~/shared` and `~/repo-name` can be used as the documented example without a special repository format.
- Configure an actual shared mise installation/data/configuration arrangement outside individual homes, with a concrete native installation/update path for the project administrator.
- Wire executable resolution for ordinary interactive shells and non-interactive SSH commands. Two users must resolve the same installed project tool, not silently install separate copies.
- Preserve normal personal tools and optional repository `mise.toml` configuration through native mechanisms; do not implement Soda-owned version selection or downloads.
- Document ordinary clone/edit/commit/push workflows and native Forgejo/Git credential setup. Do not create a checkout synchronization or automatic worktree-management feature.
- Carry filesystem/configuration changes into the project image and provisioning source rather than leaving them as instructions each developer must repeat.

**Test source:** shared-path/environment configuration, native permission setup and a later two-user installed-tool scenario, including non-interactive invocation.

**Source exit:** sharing is implemented in project OS/provisioning source, not satisfied by a common version file or cache. Toolchain cloning and selectors remain absent.

### M09 — Native project workload execution

**Implement:**

- Implement the preferred nested Podman path from documented native storage, cgroup, networking and permission requirements. Supply the required project OS configuration and host launch settings together.
- Establish which native runtime/owner holds project-shared workloads. Do not silently substitute an unrelated rootless engine per user for a shared project runtime.
- Select and wire one native workload interface, such as a compatible provider for `podman compose up`, without inventing `pod up` or a Soda service-definition format.
- Implement normal image build context, bind-mount, volume, port and service-access behavior from the developer's project environment. Merely forwarding a command to a remote socket is not enough if its paths refer to the wrong filesystem.
- If a concrete source-level blocker requires the permitted host fallback, implement that single project-scoped path, including its real ownership/storage/network/path mapping. Do not expose control of unrelated projects or appliance services.
- Supply a small ordinary repository/workload example for later native exercise, including a database service and normal TCP connection details. It is a test/example, not a mandatory project template or a service catalog.
- Document native start/stop and developer-owned changes. Do not add private service copies, routing selectors, live-state promotion or merge cleanup.

**Test source:** selected native command/configuration generation and a later ordinary workload/build/mount/TCP scenario.

**Source exit:** the chosen workload path is implemented across project OS, host configuration and developer commands. Its native feasibility is explicitly unverified until the later x86 stage; the unused alternative is not a second product backend.

### M10 — Operator Cockpit foundation

**Implement:**

- Choose a supported Cockpit delivery approach for the selected host from upstream documentation and author its native service/install source.
- Implement root/operator-only access through native Cockpit/host authentication configuration, not through developer host accounts or a new Soda password store.
- Port the shared frontend/native components actually required by Tailnet and Runners, their package manifests, entrypoints and dependency declarations.
- Keep stock host-management pages intact. Do not restore predecessor Projects/People/workspace pages or add the reserved Updates feature to this port.
- Integrate the existing login/page branding source and its installation paths without redrawing the artwork or copying a second independent palette.
- Maintain the current Cockpit frontend technology where useful; Go + HTMX applies to the Soda dashboard, not a compulsory rewrite of these pages.

**Test source:** shared page/protocol/component tests and authored native checks for operator-only access and package discovery.

**Source exit:** there is one real source/install path for stock Cockpit plus the two retained extensions, without importing the old developer UI.

### M11 — Tailnet page and backing logic

**Implement:**

- Port the predecessor's Tailscale/Tailnet page, store, native bridge, relevant components, commands and Go support identified in architecture section 17.
- Preserve native browser sign-in, state/address/peer display, exit-node selection and advertisement, and the native LAN-access preference.
- Preserve native ownership of Tailscale state and preferences. Do not add a competing enrollment database or reset unrelated native preferences.
- Adapt Forgejo address-refresh integration to the new containerized service and configured public/internal endpoints.
- Wire the required native Tailscale service/socket/command access through the operator Cockpit delivery selected in M10.
- Keep project SSH guidance IP-based. Host Tailnet enrollment does not automatically prove project-IP routing; author the chosen connectivity configuration without turning Tailnet into human identity management.
- Port focused existing tests/error handling with the feature. No live Tailnet sign-in or network mutation now.

**Test source:** retained/adapted state, native command, stream and UI cases; later operator sign-in/connectivity/exit-node journey instructions.

**Source exit:** the page and its native/Forgejo logic are ported and wired through installation source, not merely visually copied.

### M12 — Runners page and backing logic

**Implement:**

- Port the Runners page, store, protocol, components, coordinator/helper/launch executables and required Go logic as one coherent feature.
- Preserve local Forgejo/GitHub registration and capacity/status views, native start/stop/restart, and explicit removal behavior and warnings.
- Adapt the bundled Forgejo connection to M03 rather than retaining a hardcoded predecessor endpoint.
- Author the native service accounts, persistent state paths, service/policy configuration and provider-client packaging source for the new host. Runner accounts are runtime identities, not human developer host accounts.
- Preserve unprivileged local execution boundaries and secret handling. Do not make runner jobs project administrators or silently execute them inside developer workspaces.
- Keep workflows, scheduling, labels, registration authority, results and history with the providers. Soda owns local capacity, not a CI scheduler.
- Represent both architecture-specific provider inputs in source. Do not download/install runner clients, register real runners or run CI jobs now.
- Retain useful existing validation and focused error handling; the scope deferrals are not instructions to strip working behavior from this port.

**Test source:** adapted existing model, protocol, native lifecycle, authorization and UI tests, plus later opt-in native/provider job instructions.

**Source exit:** the full page-to-local-service feature and its required installation source are present, without the predecessor's unrelated release machinery.

### M13 — Complete appliance provisioning and packaging source

**Implement:**

- Choose and author one complete native provisioning path for the selected immutable host, including operator-only console/SSH administration. For Fedora CoreOS, use its supported provisioning mechanisms rather than retaining Anaconda by assumption.
- Compose actual startup/install configuration for the Soda dashboard/database, Forgejo, project runtime, Cockpit, Tailnet and runner services.
- Integrate persistent paths, native ownership, credentials, network inputs and ordinary service startup dependencies across those components. Do not overwrite persistent application/account state during a normal restart.
- Author build/staging recipes for every newly required binary, frontend bundle, container image and native configuration package. Specify real source inputs and consistent output/install paths; generated outputs come later.
- Install the dashboard, Forgejo, Cockpit and terminal branding where supported, preserving relative asset imports. Installer-specific artwork is used only if the selected installer actually supports it.
- Provide operator configuration examples with clearly illustrative network/image/URL values and secret-file inputs, and a concrete installation/bootstrap walkthrough matching M03.
- Include both architecture mappings without a sibling-build barrier. No new release pipeline, signing ceremony, updater or generic image orchestration platform is added.

**Test source:** configuration/staging expectations and later provisioning/startup checks; these files are authored, not executed now.

**Source exit:** no required component depends on an unwritten installer, missing copy step, inaccessible helper, undefined service command or assumed preinstalled Cockpit package. Native host compatibility remains a later observation.

### M14 — Full product integration and execution handoff

**Implement:**

- Connect and finish the full dashboard navigation and real page states: authentication, profile/keys, operator people creation, project discovery/create/join and direct-IP connection guidance. Complete branding, basic accessibility and role-specific navigation without relying on hidden buttons as authorization.
- Close source integration gaps across provider identity, project ownership, native provisioning, shared paths, runtime connections, frontend dependencies, installation destinations and both retained Cockpit pages.
- Remove accidental predecessor-only callers/pages/configuration and dead stubs coherently. This is not permission to delete existing assets, other agents' work or the predecessor repository.
- Provide narrow, explicitly invoked build/staging entrypoints using the selected native tools and source inputs. Include native-architecture checks. Do not invoke them or enable automatic build/test publication workflows now.
- Aggregate the already-authored feature tests and prepare opt-in installed-journey tests/instructions for the architecture's Alice/Bob flow plus Tailnet and Runners. Separate read-only observations from installation, registration, stop/start, removal and reboot actions requiring an actual named target.
- Keep test coverage inside current scope. Do not add private branching or a disaster-recovery framework just to make the validation plan larger.
- Prepare the later execution instructions: prerequisites, commands, expected native observations and where failures are reported. No new acceptance manifest/schema or manufactured PASS records.
- Record the source handoff, native assumptions and any required later generated files. Required production behavior must not remain a TODO disguised as a future validation task.

**Source exit:** all M01–M13 components are implemented and connected, with build/provisioning/test source and operator instructions ready. Mark the result **source-complete, unbuilt, unvalidated**—not “working,” “tested” or “ready to deploy.”

## 5. Later stages — held until target access and execution authorization

These stages are deliberately **not prerequisites for starting or completing unrelated source milestones**. They are recorded so “implementation complete” is not confused with “usable appliance proved.” No work is currently authorized on an unspecified machine.

### M15 — Native x86 build and source-test execution

Once the actual x86 builder/target and permissions are available, resolve real toolchain/dependency inputs, generate required artifacts, execute the authored source tests and build the component artifacts using the prepared entrypoints. Record actual revisions and outputs. Correct compilation/dependency/integration defects in their owning source milestones.

No publication, cloud enrollment, destructive install or restart is implied merely by obtaining build access.

### M16 — Native runtime assumptions and focused corrections

Exercise the highest-risk assumptions on the selected native host/project combination first:

- usable project IPs from host and developer clients;
- nested Podman, or the explicitly selected project-scoped fallback;
- ordinary build contexts, bind mounts, service ports and shared runtime ownership;
- shared mise installation resolution by two users;
- persistence under the actual normal startup/stop/start definitions;
- operator-only Cockpit/native helper access and containerized Forgejo integration.

If a candidate fails, fix the concrete mechanism and its callers. Reopen M05/M06/M08/M09/M13 as needed. Do not claim nesting works, remove required shared functionality or add a VM backend to hide the failure. This corrective work is expected to remain possible after source completion.

### M17 — Installed current-scope validation

Run the complete architecture section 16 journey on the actual installation, including Alice/Bob identities, explicit joins, direct-IP SSH/SCP/SFTP, ordinary Git use, shared files/tools, native project services and state preservation through authorized stop/start and reboot. Use a second project to check that project-local access/runtime operations do not act on the other project.

Also exercise the retained operator features: native Tailnet operation and real local Forgejo/GitHub runner registration, service control and provider jobs where the operator supplies the required accounts/tokens/permissions. Exercise destructive runner removal only with explicit disposable test resources and authorization. Missing external access remains unverified, not silently passed.

Report observed behavior and remaining gaps. This stage does not introduce the deferred edge-case matrix.

### M18 — Independent AArch64 build and native evidence

When matching AArch64 hardware/access is available, build and validate the same source on that architecture. Do not cross-compile/emulate and call that native installed evidence. An unavailable sibling does not block x86 source work, its build or its useful validation.

The source can support both architectures before either is executed. Verified completion on one architecture is not proof on the other.

## 6. Coverage of the current architecture

| Required outcome | Source owner |
| --- | --- |
| Usable Go + HTMX dashboard and Soda database | M01, M02, M04, M14 |
| Forgejo service, identity and initial operator/user onboarding | M03, M04 |
| Project-owner administration without human host accounts | M05, M06, M07 |
| Persistent Rocky + mise project userspace | M05, M08, M13 |
| Real account/key provisioning and ordinary SSH/SCP/SFTP | M05, M06, M07 |
| Direct project IPs and native connectivity | M06, M11, M13 |
| Shared files and actual shared installed tools | M05, M08 |
| Ordinary repositories and native service/workload commands | M03, M08, M09 |
| Stock operator Cockpit plus Tailnet/Runners | M10, M11, M12 |
| Installation, startup, credentials and branding integration | M01, M03, M10, M13, M14 |
| Authored tests, build recipes and later native handoff | Every owning milestone; aggregated in M14 |
| Both target architectures without premature proof claims | M05, M12, M13; execution in M15–M18 |

The [deferred document](deferred.md) remains authoritative: no private-resource branching/selectors, merge-driven promotion/cleanup, identity-remapping product, general reconciliation/recovery platform or project deletion/archival workflow is smuggled into these milestones. The predecessor's separately reserved Updates work in #61 is not part of this implementation assignment.

## 7. Per-milestone handoff discipline

For each milestone, record a short update in the implementation handoff:

- source files/features actually implemented and any relevant predecessor source attribution;
- concrete engineering choices and remaining native assumptions;
- test/build/provisioning source added or adapted;
- **Build: not run. Validation: not run.** during the source-only phase;
- exact unfinished source work or external inputs, and the next milestone.

Do not mark a milestone complete because its outline, interfaces or mocks exist. Do not let a missing native observation become an excuse to leave independent production source unfinished. Conversely, do not claim a hardware-dependent outcome has been proved by source inspection.

**First assignment: M01.** Follow its source exit, then continue through the source dependency order. Native builds and validation remain held for the later stages.

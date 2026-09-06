# SodaOS architecture

**Repository:** [`levitateos/sodaos`](https://github.com/levitateos/sodaos) (new repo: I remove the dash)

**Supersedes:** [`levitateos/soda-os`](https://github.com/levitateos/soda-os)

**Status:** Initial product direction for the new repository: a usable development pod and Soda Project OS, with selected operator integrations reused from the predecessor. This document distinguishes current scope, provisional ideas and native mechanisms still requiring proof. It is not a claim that the system has been built or validated.

**Scope boundary:** [Deferred and excluded work](deferred.md) records ideas and edge-case work that must stay outside the first version. Those items are not prerequisites for the first end-to-end proof and must not be silently restored as requirements.

**Implementation order:** Follow the [unified frontend/backend implementation plan](dashboard-implementation-plan.md) for the selected React/Forgejo-extension direction. It leads the core product, including production native environment/access integration and U08/U20 acceptance. The [native support porting plan](native-porting-plan.md) is subordinate: outside VM/SSH/evidence/artifact tools, provisioning support and retained host-operator integrations only. Shared contracts follow the core plan; optional media and helper ports are not a second core gate. The [initial M01–M18 plan](implementation-plan.md) describes historical Go + HTMX/native source work and proof stages. Planning does not authorize implementation or native execution; apply the current action/target boundary. Native evidence below remains required for readiness, not a blanket gate on unrelated source work.

**Dashboard direction update:** the user has selected client-rendered TypeScript/React + PatternFly + Vite+ + Zustand, backed by Go and Forgejo, with no SSR, Tailwind or TanStack. [Dashboard planning](dashboard-plan.md) inventories the proposed pages, dependencies and authentication work. The Go + HTMX descriptions below still describe the existing implementation; the React migration is not implemented. The unified frontend includes developer and administrator views, but Forgejo remains upstream owner of its data, rules, permissions and administration. Soda's backend is a development-environment/access extension and bounded API adapter, not a replacement forge backend. Other native, identity and authority boundaries remain in force.

## 1. Purpose

SodaOS gives a team persistent, shared development environments on a centrally operated machine. People use lightweight clients, a browser and ordinary SSH; the project environment runs their development tools, builds, agents, databases and services.

A person has one human identity and can join multiple projects. A project is a lasting development environment, not merely a repository entry or a disposable application container. People join it and work with ordinary repository checkouts, shared files, installed tools and services.

Soda supplies the usable environment and its appliance integration. Developers retain their normal Git, tool-installation and container workflows; Soda does not manage development branches, merges or cleanup for them. Private toolchain/service branching and selection machinery are deferred, not prerequisites for delivering the shared environment.

## 2. Selected direction

| Area | Direction |
| --- | --- |
| Host | An immutable, CoreOS-style appliance host. The implemented source candidate is Fedora CoreOS stable 44.20260817.3.2 with native rpm-ostree extensions; compatibility is unvalidated. |
| Runtime | Podman-based project environments and containerized application services, with native host tooling for operator integrations where appropriate. |
| Project base | Rocky Linux + mise. |
| Project lifetime | Persistent and mutable; users develop inside the environment rather than routinely discard and reconstruct it. |
| Human identity | Forgejo is the identity provider. Developers do not need individual Linux accounts on the host. |
| Developer interface | Selected client-rendered React/PatternFly dashboard over the Go API; the existing implementation is still Go + HTMX. |
| Host administration | Stock Cockpit plus the predecessor's Tailnet and Runners pages and backing logic, accessible only to the root/operator identity. No custom Cockpit developer workspace UI. |
| Project administration | Working rule: the owner of the associated Forgejo project/repository administers the project pod, not the host. |
| Soda data | A dashboard database for Soda-specific profiles, public SSH keys, projects and memberships. |
| Joining | **Add me to this project** creates the person's Linux account inside the project environment and establishes usable SSH access. |
| SSH | `{user}@{project-ip}`, for example `alice@192.168.1.101`. No project DNS naming scheme. |
| Development resources | Shared installed tools, files and services, alongside ordinary personal repository checkouts. Soda-managed private resource branches and switching are deferred. |

Implemented source choices: SQLite through modernc.org/sqlite, Rocky 9.6, mise 2026.9.1, Forgejo 15.0.7, create-once persistent Podman writable roots, and native Butane/Ignition plus rpm-ostree provisioning. See [implementation status](implementation-status.md) and [installation](installation.md). These are concrete source decisions, **unbuilt and unvalidated**, not inherited installation proof.

## 3. System shape

```text
Developer's browser
    ├── Forgejo authentication
    └── Soda dashboard: profile, projects and membership

Immutable Soda host
    ├── Native operator access and Cockpit
    │   ├── stock host-management pages
    │   ├── retained Tailnet page + native Tailscale integration
    │   └── retained Runners page + local CI runner integration
    ├── Podman
    ├── Soda dashboard service + persistent Soda database
    ├── Forgejo service/pod + persistent Forgejo data
    └── Persistent project environments
        ├── Project A — Rocky Linux + mise — 192.168.1.101
        │   ├── alice: Linux account, home and ordinary repository checkouts
        │   ├── bob: Linux account, home and ordinary repository checkouts
        │   ├── shared project files and installed toolchain
        │   └── project services (runtime placement still to be proved)
        └── Project B — Rocky Linux + mise — 192.168.1.102
            ├── alice: a separate project-local Linux account
            └── that project's files, tools and services

Developer's SSH client
    ├── alice@192.168.1.101
    └── alice@192.168.1.102
```

This is an immutable host OS plus a project userspace image, not necessarily two independently maintained operating-system distributions. A project container shares the host kernel; Rocky supplies its userspace.

**Soda Project OS** is the working name for that purpose-built userspace and its required runtime composition. Packaging SSH, account tools and development essentials into this image is intentional product assembly, not something each developer should repeat manually.

A Podman pod groups containers. It does not itself supply a unified Linux user database, init system or persistent filesystem. The exact container/pod arrangement must provide the product model above; the terms are not interchangeable.

In this document, a **workspace** means one person's working context within a project: their project-local account and personal files alongside access to shared resources. It does not imply a host Linux account, a separate project container per person or a Soda-managed resource branch.

A possible home layout is:

```text
~/shared/       access to the same project-shared files for Alice and Bob
~/repo-name/    an ordinary repository checkout in this person's home
```

These are layout ideas, not fixed paths or a new repository convention. Shared tool installations would live outside individual homes in the project filesystem. `/etc` is a candidate for configuration, not an assumed installation root for tool binaries; select actual paths using Rocky and mise's native conventions. Links, mounts and permissions remain implementation choices. Soda does not need to manage Git worktrees or synchronize personal checkouts.

## 4. Identities and roles

### Host operator

The root/operator identity administers the appliance through native host access and Cockpit, including the retained Tailnet and Runners pages. It also has administrator access to the Soda dashboard and can create people there. Project administrators do not acquire access to these host-level pages merely by owning a project.

Dashboard administrator access does not mean the dashboard web process must run as root. The service privilege and the authorization of a human request are separate implementation concerns.

The implemented bootstrap uses operator-native Forgejo installation, a restricted operator API token and `soda-setup` to create the OAuth application and record the operator's stable provider ID. See [operator setup](operator-setup.md). This does not mean that:

- a Forgejo site administrator automatically has host-root privileges;
- the host root password is a Forgejo password; or
- a second Soda password database is needed for ordinary users.

The relationship between host root, the dashboard administrator and Forgejo administration must be explicitly designed.

### Soda user

A Soda user authenticates through Forgejo and has a Soda-specific profile linked to that Forgejo identity. The profile should use Forgejo's stable user identifier as its association, rather than treating a mutable username as the permanent identifier.

The person can create projects, discover existing projects and use **Add me to this project**. They do not need a host Linux account or a host home directory to do this.

The intended dashboard administrator is the operator. Ordinary Soda users do not acquire host administration rights merely by creating or joining a project.

The initial flow is a trusted team's straightforward project join. Do not introduce invitation, approval or enterprise-role machinery for it. More elaborate access-lifecycle policies are deferred.

### Project administrator

The working rule is that the owner of the associated Forgejo project/repository is the administrator of the project's development pod. This is project-local administration, not host-root or Soda dashboard administration.

The native Linux and Podman permissions must implement that project-local authority in the chosen runtime arrangement. This rule does not require a new Soda role hierarchy or a copy of all Forgejo permissions. Organization ownership, transfers and more elaborate ownership mappings are deferred.

### Project Linux user

Joining creates a real Linux account inside the project environment. That account owns the person's project-local processes and personal files and has access to the intended shared project resources.

For example, Alice can have an `alice` account in both Project A and Project B without having an `alice` account on the host. Homes such as `/home/alice` inside a project are compatible with this model; the rejected arrangement is a separate developer account/home on the host.

The project IP address distinguishes the environments. There is no reason to preserve the old host-level derived username or hashed-name convention. Username changes, reuse and generalized account-remapping machinery are deferred; retain the stable Forgejo identity association for the ordinary account path.

Runtime service accounts are separate from developer identities: `soda` UID/GID 2000 runs the dashboard; per-runner noninteractive native accounts run CI jobs. They are not human host onboarding.

## 5. Forgejo and the Soda database

Forgejo is both the bundled Git/collaboration service and the human identity provider. It runs as a containerized appliance service, with persistent repositories, database and configuration.

The Soda dashboard uses Forgejo authentication rather than maintaining competing human passwords. Forgejo's OAuth2-provider interface is the candidate browser integration; its exact capabilities, scopes and session flow must be verified against the selected version. Do not assume OAuth2 supplies every OIDC feature.

The operator's **create user** action must result in a Forgejo-backed human identity and the associated Soda profile. The exact native Forgejo API/onboarding flow and the dashboard's authority to invoke it remain to be selected.

The Soda database legitimately owns information needed by Soda, including:

- the association to the Forgejo identity;
- Soda-specific profile information;
- public SSH keys used for development-environment access;
- project records, their environment associations and the associated Forgejo ownership;
- project membership.

Do not add private-resource selection state for the deferred branching feature.

This is not an independently assumed closed field list or a proposed database schema. Do not add an inventory of speculative fields, copied provider permissions or a generic resource model.

Soda's profiles are not a second identity provider. Forgejo owns human identity and dashboard authentication; OpenSSH authenticates project access using the public keys registered through Soda. Soda owns the additional information and product relationships it needs.

Soda must not request a user's private SSH key for onboarding. Public-key registration and installation when joining a project are in scope. Cross-environment key changes, revocation propagation, existing-session termination and account/membership drift repair are deferred; the first version does not promise automatic synchronization of those events.

## 6. The Soda dashboard

The existing developer interface is a dedicated Go + HTMX application, with the selected React/Go migration governed by the leading core plan. It is not a set of custom Cockpit packages. The unified frontend includes upstream-authorized Forgejo administration and Soda environment/access administration; neither grants host-root authority.

Its initial purpose is to make these outcomes coherent:

1. Authenticate through Forgejo.
2. Manage the person's Soda profile and register development-access public keys.
3. Create and discover projects.
4. Join a project and obtain a usable project-local account.
5. See the project's IP address and connect as oneself.
6. Work in the provided environment using ordinary tools and project services.

Private-resource branching, shared/private selectors and merge-related cleanup are not first-version dashboard features.

The dashboard's backend may perform the narrow host/runtime operations needed for the supported outcomes. The implemented boundary is a root:soda systemd Unix socket exposing only fixed project operations, with stateless project-local account setup through Podman exec. Its native permissions remain to be proved. A generic privileged command endpoint, full host-management API or permanent custom guest agent is not a selected mechanism.

Run the necessary native operations and report their actual results. Detailed cross-system retry, rollback and recovery orchestration is deferred. This does not require either the old synchronous-only restrictions or a new general background-job platform.

Stock Cockpit remains the operator tool for host services, logs, networking, storage and diagnostics. The predecessor's Tailnet and Runners pages are explicit retained extensions, not a return to custom Cockpit developer pages. They stay in Cockpit rather than being rewritten into the Soda dashboard. Developers do not log into Cockpit as host users or switch into project accounts to manage their resources through the browser.

## 7. Joining a project

**Add me to this project** is an essential product operation, not a handoff to a list of manual Linux commands.

The successful postcondition is:

> This person has the intended Linux account and resource access inside this project and can connect using their registered SSH key at the displayed project address.

The operation establishes the guest account, installs the appropriate public keys and grants the necessary native permissions. Ordinary SSH sessions, commands, SCP and SFTP should retain their native behavior.

Linux inside the project remains authoritative for the resulting account, home, permissions and processes. Soda membership records express the product relationship; they must not pretend that a failed account-creation operation succeeded.

Use native account tools and report failures without destructively recreating existing project work. General retry, concurrent-request, partial-failure recovery and drift-repair machinery is deferred, not an additional prerequisite for this first join flow.

## 8. Persistent, mutable project environments

The project environment is a lasting place where a team works. Users can install development tools, change files, run services and retain project-local state, using normal Linux permissions. Adding a member or installing a project tool must not require building a new OS image.

Mutability belongs to the project environment, not the appliance host. The project administrator manages project-local system setup and shared installations. Ordinary user operations and project administration must not grant host administration; prove the necessary permissions with the selected native runtime.

Persistence must cover Linux account records, SSH host keys, homes, shared files, installed tools, service data and configuration. Selecting volumes or a writable container layer remains an engineering decision to prove. Normal stop/start and host reboot must not unexpectedly discard that state.

Do not substitute routine deletion/recreation for the persistent development experience. Broader backup/restore, disaster recovery, resource-pressure policies and long-term replacement/rebuild machinery are deferred. No project-deletion, archival or transfer workflow is added to the first version.

## 9. Rocky Linux + mise

Rocky Linux supplies the project userspace: Linux accounts, SSH, system libraries and base administration tools. The source recipe selects Rocky 9.6 and explicitly installs the native account, SSH, Git, mise and workload dependencies in `project-os/Containerfile`; native build/behavior remains unverified.

Provide mise for development-tool installation and version selection. Developers and the project administrator use normal mise and OS package commands, within their native permissions. Soda does not become another version manager, downloader, package format or tool catalog.

The baseline remains an **actual project-shared installed toolchain**. Members use the installed shared tools by default; they must not each download and install the same tools independently. A common configuration file or download cache alone is not that shared installation. Verify the shared paths, permissions and executable resolution with real tools and two users.

Repositories remain ordinary repositories. A project may choose a native mise configuration such as `mise.toml` to declare tool versions, or install tools directly. Soda does not require a new repository schema, a mandatory toolchain declaration or a specially packaged development repository. Personal tools and configuration can coexist with the shared installation through normal native mechanisms.

Soda-managed cloning of installed toolchains, private selectors and promotion/cleanup are deferred. The first version does not need to solve portable copies of arbitrary installed tools.

## 10. Project services

The first version provides a development environment in which the team can start and use project services with normal container tools. Shared project services, including databases, remain in scope; a Soda-managed service catalog, cloning interface and per-user routing selector do not.

Repositories may carry ordinary `Containerfile`/`Dockerfile` image recipes and Compose or native Podman workload definitions. An image recipe builds an image; the workload definition describes how services run. Soda wires the native tools to the project's runtime rather than introducing its own service-definition format.

A command such as `podman compose up` is a candidate familiar interface, subject to selecting and proving a compatible native Compose provider and runtime connection. Docker CLI compatibility is not implied, and neither `docker up` nor `pod up` is a newly promised Soda command.

Developers use the services' normal addresses, ports and protocols, including PostgreSQL TCP. Private copies, consistent database-cloning automation, shared/private switching and the effects on existing shells or connections are deferred. Developers may still use ordinary native tools themselves; deferral does not ban normal development practices.

## 11. Container and service arrangement

Podman is the intended host runtime. The initial direction is a container-based project environment, not parallel VM, Docker and Podman project backends.

Investigate these arrangements in this order:

1. **Nested Podman — preferred if workable.** Run Podman inside the Rocky development container to create the project's service containers/pods. This is the intended meaning of “pods in pods,” not an assertion that a Podman pod is itself a nested runtime. Prove storage, networking, cgroups and permissions on the selected host and project userspace.
2. **Project-scoped host Podman — fallback to investigate.** If nesting is not workable, run project workloads beside the development container on the CoreOS host. The developer should still be able to invoke normal workload commands from inside the project environment, through an appropriate native remote connection or narrow integration. Soda must tie those workloads, storage and networking to the correct project.

Neither path has been validated. A remote Podman connection alone does not establish project-scoped authority. Do not hand developers an unrestricted host Podman socket and call it project isolation; project administration must not grant control over the appliance or unrelated projects.

Select the smallest native arrangement that works for a trusted team. Do not invent a general container proxy/orchestrator merely to preserve a guessed CLI. If neither arrangement meets the development experience cleanly, return the concrete gap for a decision rather than silently adding a VM fallback or deleting service support.

## 12. Ordinary repositories and contribution flow

Developers clone repositories, install tools, run services, edit, commit, push and merge using normal tools. Soda supplies their development environment; it does not manage the repository's branch lifecycle.

Git carries reproducible changes such as application code, migrations, seed scripts, service definitions and optional toolchain configuration. It does not carry running databases or installed binaries. Pulling changes does not apply migrations, replace tools or restart services; developers and the project administrator perform those operations through their ordinary workflows and native permissions.

Associate the development environment with its Forgejo project/repository for the project-owner administration rule in section 4. The environment remains a lasting development place, not a Git branch or a replacement for repository layout. A checkout on an external Git host does not replace Forgejo as Soda's human identity authority.

This ownership association does not imply mirroring every repository, organization or team permission into Soda. Joining a development environment and receiving Git repository access remain distinct operations; broader permission synchronization is deferred.

The idea of temporarily branching installed tools or service state is recorded in [deferred.md](deferred.md). Even if revisited, a Git merge is not authorization for Soda to promote live state, delete a temporary pod/toolchain or restart a developer's processes. Cleanup and return to shared resources are developer responsibilities, not a merge automation feature.

## 13. Networking and access

Each project has its own IP address reachable from the intended developer network. Developers connect directly:

```text
ssh alice@192.168.1.101
ssh alice@192.168.1.102
```

These are illustrative private-network addresses. `project-a` is a project label or shorthand for its IP, not a hostname Soda must resolve. The dashboard displays the actual project IP; the first version needs no project DNS naming scheme or custom SSH gateway.

Prove native IP assignment, reachability and SSH exposure with the selected Podman networking arrangement. An address reachable only inside Podman does not satisfy developer access. Ordinary service addresses and ports must also work; per-workspace shared/private routing is deferred.

Trusted LAN and private remote/cloud access remain relevant contexts. Reuse the Tailnet page and native Tailscale logic for operator-managed private connectivity. Host enrollment alone does not make every project IP reachable; prove the actual project route in the chosen deployment. Native Tailnet device names may be displayed without creating a project DNS requirement. Tailscale supplies connectivity, not Soda's human identity authority. Do not assume publicly exposed host administration or development services.

## 14. Host provisioning and operator access

The installation goal is operator-only host administration, not creating the first developer as a host Linux user.

The earlier idea of entering only a root password during Anaconda expressed that goal. It is not a requirement to keep Anaconda if Fedora CoreOS is selected. Fedora CoreOS normally uses Ignition; the implemented provisioning source now uses Butane/Ignition and native extension requests, with actual installation still held for native verification.

Cockpit is not assumed to be present in a default CoreOS installation. Verify its supported delivery method, available management features and root/operator-only authentication configuration on the selected host. Do not recreate its host-management pages in the Soda dashboard.

### Retained operator Cockpit pages

The predecessor's **Tailnet** and **Runners** pages, including their backing logic, are selected for reuse. These are explicit exceptions to stock-only Cockpit, not deferred ideas:

| Page | Carry over |
| --- | --- |
| Tailnet (named **Tailscale** in the predecessor UI) | Native browser sign-in, connection state, device addresses and visible peers, exit-node selection/advertisement and the native LAN-access preference. Reuse the frontend state/native bridge and supporting Tailnet/Forgejo integration. |
| Runners | Local CI runner registration and capacity/status views for bundled Forgejo and GitHub, with native start/stop/restart/removal operations. Reuse the UI, coordinator/helper/launch logic and required service integration. |

Both are operator-only. Forgejo and GitHub continue to own CI workflows, registration authority, scheduling, results and history; Soda manages local execution capacity, not another CI platform. Native runner service accounts are runtime identities, not developer host accounts. Do not conflate a runner with a person's project workspace.

Port these features with their necessary callers, native service/policy wiring and focused tests, rather than copying only the page markup. Adapt host packaging, persistent paths and the Forgejo connection/address-refresh integration to the selected CoreOS and containerized-Forgejo arrangement. Reuse is not proof that the predecessor's fixed endpoints, host network layout or native packaging work unchanged here.

The implemented bootstrap order and administrator association are documented in [installation](installation.md) and [operator setup](operator-setup.md). Native console access must remain a way for the operator to administer the host; normal developer authentication does not need a host Linux account.

The immutable host's maintenance follows the selected upstream host model. No release pipeline, update ceremony, image-signing policy or cross-image update coupling is selected by this document. Retaining Tailnet and Runners does not authorize taking over the separately reserved Updates page/work.

## 15. Ownership and decision discipline

Soda deliberately owns its dashboard, Soda-specific database information, project membership and the operations that turn membership into a usable development environment. Those responsibilities are justified by this product model; they are not automatically scope creep.

Upstream systems retain the mechanisms they already own: Forgejo authentication and Git collaboration, Linux accounts and permissions, OpenSSH access, Podman execution and native database/tool operations.

For a proposed mechanism:

1. Identify the required user outcome.
2. Identify the native owner of each resulting fact.
3. Verify the exact upstream capability and its actual limitations.
4. Add the smallest Soda integration that supplies the complete outcome.
5. Keep unproved mechanisms and unresolved product choices explicit.

Necessary integration means delivering a usable development pod, not automating users' repository workflows. Ordinary mise, Git and container commands remain developer tools; Soda must connect those tools to the correct project environment rather than require developers to assemble the appliance integration themselves.

State is not forbidden: Soda has an explicitly required database. It does not imply a generic workflow engine, reconciliation system, authorization framework or guest agent. Consult [deferred.md](deferred.md) before turning an edge case or future idea into an implementation prerequisite.

Deleting obsolete implementation should remove its supporting callers, tests and documentation coherently. A negative line-count target must not constrain delivery of genuinely new capabilities.

## 16. First end-to-end proof

This journey is owned by U08 and repeated in U20 of the [leading core implementation plan](dashboard-implementation-plan.md), which retains its concrete native assertions. The [native support plan](native-porting-plan.md) supplies outside transport/fixture/artifact tools, not another product suite or runtime owner. Lack of x86 access does not block implementing the current scope in source. Deferred machinery stays out of both the source scope and this proof:

- An operator establishes Forgejo and dashboard administration without creating developer host accounts.
- The operator creates Alice and Bob through the intended dashboard flow, backed by Forgejo identities.
- Both sign in and register public SSH keys.
- Alice, the owner of the associated Forgejo project/repository, creates its Rocky + mise development environment and is its project administrator, not a host administrator.
- Alice and Bob select **Add me to this project** and obtain their project-local accounts. This proof does not depend on an unstated automatic join for the creator.
- Both connect by ordinary SSH as themselves at the displayed project IP.
- Both can use ordinary home-directory repository checkouts and the project's shared files.
- The project administrator installs a shared development tool through mise; both users execute that same installation without separate tool downloads.
- A normal repository-defined service is started from inside the development environment using native workload commands. The proven runtime is either nested Podman or the project-scoped host fallback, not unrestricted host access.
- Developers use ordinary Git and native tool/service commands; Soda does not interpret merges as resource promotion or cleanup.
- The environment retains its intended state across normal stop/start and host reboot.

Private resource branches, dashboard selectors and the deferred edge-case/recovery matrices are not acceptance requirements for this first proof.

AArch64 and x86-64 remain equal target architectures. Verify architecture-specific container/runtime behavior on matching native hardware and record gaps honestly. Successful source checks or compilation do not prove a usable installed development environment.

## 17. Relationship to the previous repository

This new repository is an architectural restart, not a compatibility layer over [`levitateos/soda-os`](https://github.com/LevitateOS/soda-os).

The previous model of host Linux developer accounts, derived host workspace usernames, custom Cockpit developer pages, a minimal-catalog-only backend and exclusively private installed dependencies is not the governing design here. This does not prohibit reuse of the operator-only Tailnet and Runners pages described in section 14.

### Selected predecessor reuse

The source references below were inspected at [`856c6b9`](https://github.com/LevitateOS/soda-os/tree/856c6b961dd704ce4f6ba05615a964134375de5f). They identify useful source, not an installed validation result or a frozen dependency version:

- **Tailnet:** [`cockpit/src/pages/TailscalePage.tsx`](https://github.com/LevitateOS/soda-os/blob/856c6b961dd704ce4f6ba05615a964134375de5f/cockpit/src/pages/TailscalePage.tsx), `cockpit/src/tailscale/`, their UI components and `cockpit/soda-tailscale/`; supporting `internal/tailnet/`, `cmd/soda-tailnet/`, `cmd/soda-forgejo-tailnet/` and the existing Forgejo address-refresh integration. The predecessor's [`docs/networking.md`](https://github.com/LevitateOS/soda-os/blob/856c6b961dd704ce4f6ba05615a964134375de5f/docs/networking.md) explains the old behavior, not the new project's network contract.
- **Runners:** [`cockpit/src/pages/RunnersPage.tsx`](https://github.com/LevitateOS/soda-os/blob/856c6b961dd704ce4f6ba05615a964134375de5f/cockpit/src/pages/RunnersPage.tsx), `cockpit/src/runners/`, their UI components and `cockpit/soda-runners/`; `internal/runners/`, `cmd/soda-runners/`, `cmd/soda-runner-helper/`, `cmd/soda-runner-launch/` and the required runner service/policy/package wiring. The predecessor's [CI runner guide](https://github.com/LevitateOS/soda-os/blob/856c6b961dd704ce4f6ba05615a964134375de5f/docs/public/30-Develop/40-ci-runners.md) records the native user journey.

Bring over the relevant tests with the logic and verify the adapted native journeys on the selected host. Both features have now been ported in source, as recorded in [implementation status](implementation-status.md); neither has been built or validated here.

The shared-resource direction from [the previous repository's issue #80](https://github.com/LevitateOS/soda-os/issues/80) informs this architecture, but its full private-resource branching and switching experience is deferred in this repository. Shared installed tools, files and project services remain current scope. Its host-account assumptions are replaced by project-local accounts and direct-IP SSH. This narrowing does not modify or close the previous repository's issue.

The previous audit issues must be reassessed against this architecture rather than implemented blindly or automatically transferred. Reuse code, assets and evidence only where they fit the selected outcome; old implementation and tests do not create new product requirements.

Writing this document does not authorize deleting the previous repository, closing its issues, migrating existing environments, implementing the new system or publishing artifacts. In particular, it does not silently cancel or redirect the previous repository's separately reserved [Updates work in #61](https://github.com/LevitateOS/soda-os/issues/61). Any handoff remains an explicit coordination decision.

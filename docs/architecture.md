# SodaOS architecture

**Repository:** [`levitateos/sodaos`](https://github.com/levitateos/sodaos) (new repo: I remove the dash)

**Supersedes:** [`levitateos/soda-os`](https://github.com/levitateos/soda-os)

**Status:** Initial product direction for the new repository. This document records the selected model and distinguishes it from implementation questions still requiring investigation. It is not a claim that the system has been built or validated.

## 1. Purpose

SodaOS gives a team persistent, shared development environments on a centrally operated machine. People use lightweight clients, a browser and ordinary SSH; the project environment runs their development tools, builds, agents, databases and services.

A person has one human identity and can join multiple projects. A project is a lasting development environment, not merely a repository entry or a disposable application container. People join it, work together, temporarily diverge where needed, and return to its shared resources.

Soda handles the necessary integration. It must not move that work back onto developers merely to make its own architecture smaller. Equally, it should not become a generic cloud platform, identity product or replacement for upstream Linux tools.

## 2. Selected direction

| Area | Direction |
| --- | --- |
| Host | An immutable, CoreOS-style appliance host. Fedora CoreOS is the leading candidate; its exact composition is not yet selected or validated. |
| Runtime | Podman-based project environments and appliance services. |
| Project base | Rocky Linux + mise. |
| Project lifetime | Persistent and mutable; users develop inside the environment rather than routinely discard and reconstruct it. |
| Human identity | Forgejo is the identity provider. Developers do not need individual Linux accounts on the host. |
| Developer interface | A Soda dashboard written in Go + HTMX. |
| Host administration | Stock Cockpit, accessible only to the root/operator identity. No custom Soda Cockpit pages. |
| Soda data | A dashboard database for Soda-specific profiles, public SSH keys, projects, memberships and required environment choices. |
| Joining | **Add me to this project** creates the person's Linux account inside the project environment and establishes usable SSH access. |
| SSH | `{user}@{project-address}`, not `{user}-{project}@{host-address}`. |
| Development resources | Project-shared installed tools, files and services, with independently selectable workspace-private overrides. |

The database engine, Rocky release, host release, image layout and provisioning mechanisms remain open. None should be inferred from the old repository's choices.

## 3. System shape

```text
Developer's browser
    ├── Forgejo authentication
    └── Soda dashboard: profile, projects, membership and environment choices

Immutable Soda host
    ├── Native operator access and stock Cockpit
    ├── Podman
    ├── Soda dashboard service + persistent Soda database
    ├── Forgejo service/pod + persistent Forgejo data
    └── Persistent project environments
        ├── Project A — Rocky Linux + mise
        │   ├── alice: Linux account, home and SSH access
        │   ├── bob: Linux account, home and SSH access
        │   ├── shared project files and installed toolchain
        │   └── shared services and private overrides
        └── Project B — Rocky Linux + mise
            ├── alice: a separate project-local Linux account
            └── that project's files, tools and services

Developer's SSH client
    ├── alice@project-a
    └── alice@project-b
```

This is an immutable host OS plus a project userspace image, not necessarily two independently maintained operating-system distributions. A project container shares the host kernel; Rocky supplies its userspace.

**Soda Project OS** is the working name for that purpose-built userspace and its required runtime composition. Packaging SSH, account tools and development essentials into this image is intentional product assembly, not something each developer should repeat manually.

A Podman pod groups containers. It does not itself supply a unified Linux user database, init system or persistent filesystem. The exact container/pod arrangement must provide the product model above; the terms are not interchangeable.

In this document, a **workspace** means one person's working context within a project: their project-local account, personal files and private overrides alongside access to shared resources. It does not imply a host Linux account or an entire separate project container for every person.

## 4. Identities and roles

### Host operator

The root/operator identity administers the appliance through native host access and stock Cockpit. It also has administrator access to the Soda dashboard and can create people there.

Dashboard administrator access does not mean the dashboard web process must run as root. The service privilege and the authorization of a human request are separate implementation concerns.

The exact mechanism that establishes the operator's first dashboard session is unresolved. In particular, do not silently assume that:

- a Forgejo site administrator automatically has host-root privileges;
- the host root password is a Forgejo password; or
- a second Soda password database is needed for ordinary users.

The relationship between host root, the dashboard administrator and Forgejo administration must be explicitly designed.

### Soda user

A Soda user authenticates through Forgejo and has a Soda-specific profile linked to that Forgejo identity. The profile should use Forgejo's stable user identifier as its association, rather than treating a mutable username as the permanent identifier.

The person can create projects, discover existing projects and use **Add me to this project**. They do not need a host Linux account or a host home directory to do this.

The intended dashboard administrator is the operator. Ordinary Soda users do not acquire host administration rights merely by creating or joining a project.

Whether every Soda user may join every project, or whether some projects restrict joining, remains an explicit product decision. Do not invent invitation, approval or enterprise-role machinery by default.

### Project Linux user

Joining creates a real Linux account inside the project environment. That account owns the person's project-local processes and personal files and has access to the intended shared project resources.

For example, Alice can have an `alice` account in both Project A and Project B without having an `alice` account on the host. Homes such as `/home/alice` inside a project are compatible with this model; the rejected arrangement is a separate developer account/home on the host.

The project address distinguishes the environments. There is no reason to preserve the old host-level derived username or hashed-name convention.

Runtime service accounts required by the host's native tooling are a separate concern from developer identities. Their arrangement has not been selected.

## 5. Forgejo and the Soda database

Forgejo is both the bundled Git/collaboration service and the human identity provider. It runs as a containerized appliance service, with persistent repositories, database and configuration.

The Soda dashboard uses Forgejo authentication rather than maintaining competing human passwords. Forgejo's OAuth2-provider interface is the candidate browser integration; its exact capabilities, scopes and session flow must be verified against the selected version. Do not assume OAuth2 supplies every OIDC feature.

The operator's **create user** action must result in a Forgejo-backed human identity and the associated Soda profile. The exact native Forgejo API/onboarding flow and the dashboard's authority to invoke it remain to be selected.

The Soda database legitimately owns information needed by Soda, including:

- the association to the Forgejo identity;
- Soda-specific profile information;
- public SSH keys used for development-environment access;
- project records and their environment associations;
- project membership;
- shared/private resource selections where native configuration does not already provide the needed representation.

This is not an independently assumed closed field list or a proposed database schema. Do not add an inventory of speculative fields, copied provider permissions or a generic resource model.

Soda's profiles are not a second identity provider. Forgejo owns human authentication; Soda owns the additional information and product relationships it needs.

Soda must not request a user's private SSH key for onboarding. Public-key registration, propagation to joined environments, later key changes and access revocation need a clear behavior contract. Whether key changes affect existing memberships immediately or through another explicit operation is not yet decided.

## 6. The Soda dashboard

The developer interface is a dedicated Go + HTMX application, not a set of custom Cockpit packages. The operator also uses it for Soda administration, including creating people.

Its purpose is to make these outcomes coherent:

1. Authenticate through Forgejo.
2. Manage the person's Soda profile and development-access keys.
3. Create and discover projects.
4. Join a project and obtain a usable project-local account.
5. See the project's SSH address and connect as oneself.
6. Use shared installed tools and services.
7. Create and select independent private overrides.
8. Return to the shared project resources.

The dashboard's backend may perform the narrow host/runtime operations needed for those outcomes. The service privilege boundary and project-account provisioning mechanism require investigation. A generic privileged command endpoint, full host-management API or permanent custom guest agent is not a selected mechanism.

Operation execution, progress reporting and partial-failure handling also remain to be designed. Do not automatically import either the old synchronous-only restrictions or a new general background-job platform. Select what the actual operations require.

Stock Cockpit remains an operator tool for host services, logs, networking, storage and diagnostics. Developers do not log into Cockpit as host users or switch into project accounts to manage their resources through the browser.

## 7. Joining a project

**Add me to this project** is an essential product operation, not a handoff to a list of manual Linux commands.

The successful postcondition is:

> This person has the intended Linux account and resource access inside this project and can connect using their registered SSH key at the displayed project address.

The operation establishes the guest account, installs the appropriate public keys and grants the necessary native permissions. Ordinary SSH sessions, commands, SCP and SFTP should retain their native behavior.

Linux inside the project remains authoritative for the resulting account, home, permissions and processes. Soda membership records express the product relationship; they must not pretend that a failed account-creation operation succeeded.

Repeated requests and partial failures must not create duplicate accounts or destroy existing project work. The supported retry or repair behavior must be derived from the chosen native mechanism, not invented as an elaborate workflow before that mechanism is known.

## 8. Persistent, mutable project environments

The project environment is a lasting place where a team works. Users can install development tools, change files, run services and retain project-local state. Adding a member or installing a project tool must not require building a new OS image.

Mutability belongs to the project environment, not the appliance host. How much project-local system administration or package installation users receive is unresolved; project-local privilege must not be confused with host-root privilege.

The persistence design must cover the actual environment: Linux account records, SSH host keys, homes, shared files, installed tools, service data and configuration. Selecting volumes or a writable container layer is an engineering decision still to be proved.

Do not assume projects are routinely deleted and recreated, or make such recreation a prerequisite for normal project management. Do not substitute a disposable-container workflow for the requested persistent development experience.

Host reboot and normal environment stop/start must not unexpectedly discard project state. Longer-term image maintenance and preservation of user-made system changes need explicit design; this document does not select an automatic replacement/rebuild policy.

No project-deletion, archival, transfer or destructive rebuild workflow was requested as part of this architecture. Do not expand the product around hypothetical deletion scenarios.

## 9. Rocky Linux + mise

Rocky Linux is the selected project-userspace direction. It supplies the Linux account environment, SSH, system libraries and base administration tools. The exact supported Rocky version and base package set remain to be selected.

mise owns development-tool installation and version selection. Soda integrates the shared/private experience rather than implementing another version manager, downloader or package format.

The intended steady state is an **actual project-shared installed toolchain**:

- Members use the same installed project tools by default.
- Each member must not independently download and install the same shared tools.
- A shared version/configuration file or download cache alone is not sufficient.
- A person can clone the installed project toolchain into a private workspace toolchain.
- Changes to the private toolchain must not mutate the shared one.
- Workspace-specific tools and configuration can coexist with shared resources.
- A person can independently disable the private toolchain selection and return to the project toolchain.

mise alone has not been proved to provide this complete arrangement. Verify permissions, installation paths, executable resolution, relocatability and the behavior of representative installed tools. Do not assume a directory copy is sufficient, or respond to a gap by removing the shared-installed-tool requirement.

The method of applying reviewed toolchain changes to the shared environment remains open.

## 10. Shared services and private overrides

Each project supports both shared services and separate workspace-private services. Neither one global host environment nor exclusively private services meets the requirement.

Through their primary Soda dashboard session, a person can choose whether their workspace's service endpoint uses the shared project service or a private service. Service and toolchain selections are independent.

For a stateful service, creating a private copy includes the relevant data, not only a container definition. Database copying must produce a consistent database through an appropriate native mechanism.

The defining example is PostgreSQL:

1. Alice and Bob use the project's shared PostgreSQL service.
2. Alice chooses to create a private copy through Soda.
3. Soda establishes the private service and a consistent copy of its relevant data.
4. Alice's workspace service selection points to that instance.
5. Alice makes and tests mutations without changing the shared database or Bob's selected service.
6. After reproducible changes are integrated into the shared environment, Alice disables the private override and uses the shared service again.

This must work for PostgreSQL TCP and other applicable non-HTTP protocols. An HTTP-only link or reverse proxy does not satisfy the requirement.

Address/port representation and the routing mechanism are unresolved. Likewise, disabling an override does not silently imply deletion of its private state. Whether that state is retained or explicitly removed needs a product decision.

## 11. Container and service arrangement

Podman is the intended host runtime. The initial direction is a container-based project environment, not parallel support for VM, Docker and Podman project backends.

Two arrangements need comparison before selecting the service implementation:

- **Nested Podman:** project services run within the project environment. This keeps the environment self-contained but requires proof of nested storage, networking, cgroup and privilege behavior.
- **Sibling containers:** project services run beside the development container on the host. This avoids some nesting constraints but requires Soda to compose their project/workspace ownership, networks, storage and endpoint selections.

Neither arrangement has been selected. Packaging software in the Rocky image does not itself establish runtime networking, persistence or permissions.

A broadly privileged project container or unrestricted host Podman socket is not an assumed shortcut. Establish the narrow access actually needed for a trusted team's development workflow without inventing an enterprise hostile-tenant platform.

If the chosen container arrangement cannot meet the required experience cleanly, return the demonstrated gap for a decision. Do not silently implement a VM fallback or delete required functionality.

## 12. Git and contribution flow

Git carries reproducible project changes: application code, database migrations, seed scripts, service definitions and toolchain configuration. It does not carry running databases or installed binaries.

The intended lifecycle is:

1. Start from project-shared tools and services.
2. Clone only the tools and/or services that need private changes.
3. Develop and test privately.
4. Commit and push the reproducible changes.
5. Pull and apply the reviewed changes to the shared project environment.
6. Disable the private selections and return to the updated shared resources.

Pulling changes and applying them are distinct operations. The exact project update interaction and who may apply shared changes remain unresolved. Do not invent an automatic promotion or database-state merge workflow.

Forgejo owns its repositories and collaboration. A Soda project is not automatically identical to a Forgejo repository, organization or team. Repositories on external Git hosts do not change the choice of Forgejo as Soda's human identity provider.

Joining the development environment must not silently be equated with repository permission. Decide any membership-to-Git-access integration explicitly rather than copying provider permissions into Soda by accident.

## 13. Networking and access

Developers connect as themselves to the project's reachable address:

```text
ssh alice@project-a
ssh alice@project-b
```

The displayed address may be an IP address or a stable hostname. It must be usable from the intended developer network; a Podman-internal address alone does not establish that outcome.

Project addressing, DNS, SSH exposure and shared/private service routing require a coherent native network design. The exact mechanisms are open. Do not introduce a custom SSH gateway merely to retain an old host-account convention.

Trusted LAN and private remote/cloud access remain relevant deployment contexts. Do not assume publicly exposed host administration or development services. The chosen access mechanism must support ordinary developer clients and must not become the human identity authority merely because it provides connectivity.

## 14. Host provisioning and operator access

The installation goal is operator-only host administration, not creating the first developer as a host Linux user.

The earlier idea of entering only a root password during Anaconda expressed that goal. It is not a requirement to keep Anaconda if Fedora CoreOS is selected. Fedora CoreOS normally uses Ignition; the actual installation/provisioning experience must be selected and verified against its native mechanisms.

Cockpit is not assumed to be present in a default CoreOS installation. Verify its supported delivery method, available management features and root-only authentication configuration on the selected host. Do not recreate its host-management pages in the Soda dashboard.

Bootstrap order and the initial dashboard/Forgejo administrator relationship remain unresolved. Native console access must remain a way for the operator to administer the host; normal developer authentication does not need a host Linux account.

The immutable host's maintenance follows the selected upstream host model. No release pipeline, update ceremony, image-signing policy or cross-image update coupling is selected by this document.

## 15. Ownership and decision discipline

Soda deliberately owns its dashboard, Soda-specific database information, project membership and the operations that turn membership into a usable development environment. Those responsibilities are justified by this product model; they are not automatically scope creep.

Upstream systems retain the mechanisms they already own: Forgejo authentication and Git collaboration, Linux accounts and permissions, OpenSSH access, Podman execution and native database/tool operations.

For a proposed mechanism:

1. Identify the required user outcome.
2. Identify the native owner of each resulting fact.
3. Verify the exact upstream capability and its actual limitations.
4. Add the smallest Soda integration that supplies the complete outcome.
5. Keep unproved mechanisms and unresolved product choices explicit.

Do not confuse necessary integration with a competing subsystem. Conversely, the existence of upstream primitives is not a reason to return developers to manually assembling the experience.

State is not forbidden: Soda has an explicitly required database. Generic workflow engines, reconciliation systems, authorization frameworks or guest agents are not selected simply because some operations require state or coordination.

Deleting obsolete implementation should remove its supporting callers, tests and documentation coherently. A negative line-count target must not constrain delivery of genuinely new capabilities.

## 16. First end-to-end proof

Before building a broad platform, demonstrate this one complete journey:

- An operator establishes Forgejo and dashboard administration without creating developer host accounts.
- The operator creates Alice and Bob through the intended dashboard flow, backed by Forgejo identities.
- Alice signs in and registers a public SSH key.
- Alice creates a Rocky + mise project environment.
- Bob registers his key and selects **Add me to this project**.
- Both connect by ordinary SSH as themselves at the project address.
- Both use shared project files and the same installed toolchain without separate tool downloads.
- Alice uses a private toolchain copy without mutating the shared installation.
- Alice creates a consistent private PostgreSQL copy and selects it without changing Bob's shared service.
- Reproducible changes are committed, pushed, pulled and explicitly applied to the shared environment.
- Alice can independently disable toolchain and service overrides and return to the shared resources, without an unrequested destructive cleanup.
- The environment retains its intended state across normal stop/start and host reboot.

AArch64 and x86-64 remain equal target architectures. Verify architecture-specific container/runtime behavior on matching native hardware and record gaps honestly. Successful source checks or compilation do not prove a usable installed development environment.

## 17. Relationship to the previous repository

This new repository is an architectural restart, not a compatibility layer over [`levitateos/soda-os`](https://github.com/LevitateOS/soda-os).

The previous model of host Linux developer accounts, derived host workspace usernames, custom Cockpit developer pages, a minimal-catalog-only backend and exclusively private installed dependencies is not the governing design here.

The shared-resource requirements from [the previous repository's issue #80](https://github.com/LevitateOS/soda-os/issues/80) are incorporated above. Its host-account assumptions are replaced by the project-local account model described here.

The previous audit issues must be reassessed against this architecture rather than implemented blindly or automatically transferred. Reuse code, assets and evidence only where they fit the selected outcome; old implementation and tests do not create new product requirements.

Writing this document does not authorize deleting the previous repository, closing its issues, migrating existing environments, implementing the new system or publishing artifacts. In particular, it does not silently cancel or redirect the previous repository's separately reserved [Updates work in #61](https://github.com/LevitateOS/soda-os/issues/61). Any handoff remains an explicit coordination decision.

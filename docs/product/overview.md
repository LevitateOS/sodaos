# Soda OS product overview

Soda OS is **the operating system for software factories**: run your coding agents
on your infrastructure, with controlled access and reviewable results. It turns
human-authorized Forgejo work into bounded, isolated executions on an
operator-managed Fedora CoreOS appliance for a trusted team. Persistent shared
Projects support human development and intervention.

Developers work through native Forgejo, ordinary SSH, Git, mise and container tools.
Soda supplies the integration that creates project environments, membership, access
keys, managed terminals, local CI runner capacity and disposable agent workspaces.
Developers do not receive individual Linux accounts on the host.

Public Internet hosting and a production hosting platform are outside this product.
Private LAN access and optional Tailscale access are in scope. Trying Soda or using
its baseline services must not require buying or owning a domain.

## Major concepts

| Concept | Meaning |
| --- | --- |
| **Host** | The Fedora CoreOS appliance. Operator administration only. |
| **Work Item** | A Forgejo issue or pull request with a human-authorized objective. |
| **Attempt** | One explicit admission with recorded inputs and limits for implementation, verification and at most one repair. |
| **Run** | One bounded agent execution against recorded inputs, with its own identity and resource records. |
| **Workspace** | A disposable, isolated environment allocated to a Run, separate from a persistent Project. |
| **Project** | A persistent shared development environment for one repository's authorized members. |
| **Project OS** | The userspace foundation inside a project (accounts, tools, persistence, workloads). |
| **Spaces** | Fountain-hosted Spaces page and persistent Workspace panel for project work. |
| **Runner** | Local CI capacity on the appliance, operated by the Soda operator. |
| **Tailnet** | Private connectivity for host and project reachability. |

Detailed ownership lives in [Projects](projects.md), [Spaces](spaces.md),
[Architecture](../architecture/overview.md) and [Project OS](../reference/project-os.md).

## Software factory workflow

Forgejo owns the software lifecycle: accounts, repositories, issues, pull
requests, review, merge and CI. Soda admits work, assigns execution, and records
the resources it consumes. Its ledger is not a second issue or project-management
system. Factory workspaces are disposable and separate from persistent Projects;
removing one must not remove a Project's accounts, tools or data.

Replaceable coding agents perform bounded work, starting with Codex. OpenCode,
Muse Code and Oh My Pi may be added later through the same authority boundary. A
fresh review in a separate workspace checks an agent's proposed change; a human decides
whether to merge. A repairable failure may receive one bounded repair before
returning to a human. CI and review bind to the exact candidate commit; a changed
candidate requires new verification. Closing or cancelling work withdraws further
execution and publication authority. Terminal outcome and cleanup completion are
separate records; another attempt requires explicit human admission.

Follow the [first-task walkthrough](../public/30-Use-Soda/15-software-factory.md).
The [factory reference](../reference/factory.md) owns the operator commands and
configuration. [Architecture](../architecture/overview.md) and
[Trust](../architecture/trust.md) own execution and authority boundaries.

## System shape

```text
Fedora CoreOS host — operator administration only
├── Stock branded Cockpit, tailscaled, CI runners
├── soda-factory — admission, execution ledger and narrow publication
│   └── Rootless Podman — disposable agent workspaces and restricted networking
└── Podman
    ├── Fountain (maintained Forgejo fork) — identity, Git, collaboration and extension host
    ├── Soda extension service — environments and broker integration
    ├── Caddy — private HTTPS endpoints
    └── Persistent Project OS containers
        ├── Project-local accounts, homes, SSH
        ├── Shared installed tools and files
        └── Nested workloads and persistent service data
```

Forgejo, Soda and Caddy are separate containers. Forgejo is not a Podman pod.

## Product roles

| Role | Authority |
| --- | --- |
| Host root / operator | Appliance administration, Cockpit, Soda operator setup |
| Forgejo site administrator | Forgejo administration; not automatically Soda operator |
| Repository owner | May create that repository's project environment |
| Project member | Project-local Linux account after explicit Join |
| Soda operator | The Forgejo user ID recorded at setup; runner and appliance settings |

Owning a repository does not grant host access. Site or organization administration
does not grant project root inside another team's environment.

## What Soda owns vs Forgejo

- **Forgejo** owns identity, passwords and factors, native sessions, permissions,
  Git, collaboration and administrator workflows.
- **Soda** owns environment associations, membership after confirmed Join,
  development-access public keys, extension operations and broker grants, local
  runner capacity and selected Tailnet enrollment policy for projects. Factory
  responsibilities include admission, execution identity, permitted operations,
  time and resource limits, cancellation, infrastructure audit and cleanup.

Soda does not invent a second password system, CI scheduler or forge-owned workflow
engine. Native Git authorization remains separate from project membership.

## Frontend model

Fountain supplies Forgejo's browser chrome, authentication and collaboration UI,
plus the native extension host. Soda contributes the Spaces page, persistent
Workspace panel and operator pages through its separately packaged extension.

There is no standalone Soda web frontend. Fountain owns its maintained fork and
extension SDK; Soda consumes that host contract.

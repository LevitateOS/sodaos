# Soda OS product overview

Soda OS is a **factory-first development appliance** for a trusted team. Its
product direction is to turn approved software work into bounded, isolated
executions on an operator-managed Fedora CoreOS host. Persistent shared Projects
support human development and intervention.

Developers work through native Forgejo, ordinary SSH, Git, mise and container tools.
Soda supplies the integration that creates project environments, membership, access
keys, managed terminals and local CI runner capacity. Developers do not receive
individual Linux accounts on the host.

Public Internet hosting and a production hosting platform are outside this product.
Private LAN access and optional Tailscale access are in scope. Trying Soda or using
its baseline services must not require buying or owning a domain.

## Major concepts

| Concept | Meaning |
| --- | --- |
| **Host** | The Fedora CoreOS appliance. Operator administration only. |
| **Project** | A persistent shared development environment for one repository's authorized members. |
| **Project OS** | The userspace foundation inside a project (accounts, tools, persistence, workloads). |
| **Spaces** | Sodaspaces: the Forgejo-native workspace entry and environment drawer. |
| **Runner** | Local CI capacity on the appliance, operated by the Soda operator. |
| **Tailnet** | Private connectivity for host and project reachability. |
| **Work Item** | A Forgejo issue or pull request with a human-authorized objective. |
| **Run** | One bounded execution against recorded inputs; Soda owns its execution and resource records. |
| **Workspace** | A disposable, isolated environment allocated to a Run, separate from a persistent Project. |

Detailed ownership lives in [Projects](projects.md), [Spaces](spaces.md),
[Architecture](../architecture/overview.md) and [Project OS](../reference/project-os.md).

## Intended software factory

Forgejo owns the software lifecycle: accounts, repositories, issues, pull
requests, review, merge and CI. Soda admits work, assigns execution, and records
the resources it consumes. Its ledger is not a second issue or project-management
system. Factory workspaces are disposable and separate from persistent Projects;
removing one must not remove a Project's accounts, tools or data.

Replaceable coding agents perform bounded work, starting with Codex. OpenCode,
Muse Code and Oh My Pi may be added later through the same authority boundary. A
fresh, independent reviewer checks an agent's proposed change; a human decides
whether to merge. A failed attempt may receive one bounded repair before returning
to a human. These are target capabilities, not claims that the current appliance has
agent execution, review or repair interfaces. See [Architecture](../architecture/overview.md)
and [Trust](../architecture/trust.md) for their intended boundaries.

## System shape

```text
Fedora CoreOS host — operator administration only
├── Stock branded Cockpit, tailscaled, CI runners
└── Podman
    ├── Stock Forgejo — identity, Git, collaboration
    ├── Soda Go API/OAuth service — environments, grants, adapters
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
  development-access public keys, protected adapter sessions and grants, local
  runner capacity and selected Tailnet enrollment policy for projects. The intended
  factory adds work admission, execution and resource accounting.

Soda does not invent a second password system, CI scheduler or forge-owned workflow
engine. Native Git authorization remains separate from project membership.

## Frontend model

Stock Forgejo supplies the browser chrome, authentication and collaboration UI.
Soda mounts Spaces, operator settings and related views through supported Forgejo
customization and Lit components under the configured Forgejo origin at `/-/soda/`.

There is no standalone Soda web frontend and no downstream Forgejo fork.

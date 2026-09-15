# Soda OS product overview

Soda OS is a **development appliance**: an operator-managed Fedora CoreOS host that
provides persistent, shared development environments for a trusted team.

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

Detailed ownership lives in [Projects](projects.md), [Spaces](spaces.md),
[Architecture](../architecture/overview.md) and [Project OS](../reference/project-os.md).

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
  runner capacity and selected Tailnet enrollment policy for projects.

Soda does not invent a second password system, CI scheduler or forge-owned workflow
engine. Native Git authorization remains separate from project membership.

## Frontend model

Stock Forgejo supplies the browser chrome, authentication and collaboration UI.
Soda mounts Spaces, operator settings and related views through supported Forgejo
customization and Lit components under the configured Forgejo origin at `/-/soda/`.

There is no standalone Soda web frontend and no downstream Forgejo fork.

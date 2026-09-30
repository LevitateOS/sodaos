# Architecture overview

Soda OS is a Fedora CoreOS appliance for software factories. People authorize
factory operation; Soda coordinates eligible issues through execution, review,
correction and automatic native merge. Fountain supplies Forgejo collaboration
and generic extension capabilities. Repository environments and Spaces support
both agent work and human development or intervention.

Product concepts: [overview](../product/overview.md). Trust and privilege:
[trust](trust.md). Networking: [networking](networking.md). Release:
[release](release.md).

## Topology

| Placement | Mechanism |
| --- | --- |
| Host | Fedora CoreOS, native rpm-ostree layering, Podman, systemd |
| Native operator services | Stock branded Cockpit, Tailnet/Runners management, `tailscaled`, CI runner services, restricted Soda project helper |
| Appliance applications | Separate Podman containers for the maintained Forgejo extension host, Soda's Go API service and Caddy |
| Persistent application data | Soda SQLite database and a separate Forgejo volume (including installed extension packages) |
| Identity Broker | Host userspace `soda-identity`, private administration/execution sockets and encrypted subscription custody |
| Factory execution | Unprivileged `soda-factory` operator command, execution ledger, narrow publisher and rootless Podman workspaces |
| Projects | Persistent Project OS containers with project-local accounts, writable roots, SSH and shared installations |
| Project workloads | Nested Podman inside the project |

This table locates existing services. The operator command's disposable execution
model does not prescribe the target factory's environment lifetime. Environment
reuse and the review session's shared container are governed by the
[product contract](../product/overview.md#software-factory-workflow).

**Forgejo is a standalone container, not a Podman pod.** A pod groups containers; it
is not a user database, init system or filesystem. Project containers share the host
kernel; the Project OS distribution supplies userspace.

The `soda-dashboard` command, container, config and data names identify the Go
backend. Do not rename persistent records or roots merely because the UI is called
Spaces.

## Component ownership

| Component | Responsibility |
| --- | --- |
| Fountain / native Forgejo | Identity, Git, collaboration, Actions UI/scheduling, native sessions and generic extension contributions |
| Soda Forgejo extension | Native browser pages, persistent workspace panel and narrowly scoped calls into the Soda service |
| Soda Go service (`web`, `web/api`, `web/auth`) | Environment/project APIs, membership and operator policy, terminal streams and runner/Tailnet operations |
| Factory control (`factory/control`) | Readiness and execution coordination, verification and merge eligibility, execution records, limits and cancellation |
| Factory workspace and publisher (`host/workspace`, `host/publish`) | Assigned execution resources and permitted native publication; run authority remains distinct from Project membership |
| Identity Broker (`identity/control`) | Explicit subscription ownership/delegation and serialized native execution leases; [credential boundary](../reference/credentials.md#identity-broker) |
| `soda-host` daemon | Privileged project, terminal and Tailnet companion execution |
| Project OS | Developer accounts, tools, persistence, nested workloads |
| Caddy | Private HTTPS termination for configured origins |
| Cockpit | Host administration (all interfaces, root/operator) |
| Local runners | Appliance CI capacity; Forgejo owns workflows and results |

These responsibilities describe the target boundaries, not completion of the
automatic lifecycle. Current commands: [factory reference](../reference/factory.md).
Go package placement: [Go ownership](../development/go.md).

## Factory flow

The [product overview](../product/overview.md#software-factory-workflow) owns the
issue-to-merge loop and its remaining design questions. Soda owns readiness,
dependency handling, agent coordination and the decision to request a merge under
authorized policy. Fountain owns native repository permissions and the actual
collaboration operations; Forgejo Actions owns CI scheduling and results. Soda
does not maintain competing issue, PR or CI records.

Spaces exposes the live work and intervention needs. Browser presence is not the
execution grant for background factory work. The Identity Broker supplies
authorized AI CLI access; [Trust](trust.md) separates provider sponsorship,
execution identity and native publication/merge authority.

Reusable environments and disposable run resources have different lifetimes.
The environment design must preserve that distinction without making a fresh
container for every stage a product requirement.

## Control and data flow

1. Browser users authenticate with Forgejo. Native routes and extension
   contributions share that session; the browser sends no separate Soda login
   credential to the Soda service.
2. The extension carries live, narrowly scoped native authority across a private
   bridge. The Soda service checks its own project membership, operator policy and
   operation-specific grants before acting. Browser cookies and raw Forgejo
   session IDs remain inside Forgejo.
3. Mutating project operations go through the host Unix-socket helper as fixed
   operations, not arbitrary commands.
4. Terminal attach allocates a managed tmux session under the member's project
   account. Native session and repository authority govern the disposable browser
   attachment, not the lifetime of the shell.
5. Runner and Tailnet operator settings mutate appliance-local state only for the
   configured Soda operator identity.

## Persistence boundaries

| State | Owner |
| --- | --- |
| Forgejo database and repos | Forgejo volume |
| Soda SQLite, environments, preferences and product grants | Soda data volume |
| Factory admissions, attempts, runs and resource ledger | Protected factory `execution.db` under the configured operator root |
| Run-owned checkout, scratch, containers and network | Disposable only where exclusively assigned to the run; recorded ownership governs cleanup |
| Retained factory results | Protected factory operator state, separate from disposable resources and public logs |
| Provider connection custody | Broker-owned encrypted database and separate private key; transient enrollment/tool auth in tmpfs |
| Project accounts, homes, tools, service data | Project persistent root |
| Runner registration and local capacity | Host runner state |
| Host OS and layered packages | rpm-ostree / CoreOS |

Persistent application containers and project roots start existing state. Replacement,
`--rm` and pruning are not repair strategies.

Ending a factory run releases its authority and resources, not every resource it
used. Cleanup may remove exclusively owned temporary state; it must preserve
reused containers, persistent Project data and unrelated work. The current
operator command's fresh workspaces follow its documented disposable lifecycle.

## Related source

- Appliance topology: `appliance/services/`
- Project images and units: `project-os/`
- Entrypoints: `cmd/soda-dashboard`, `cmd/soda-host`, `cmd/soda-factory`

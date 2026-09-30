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
| Native operator services | Stock branded Cockpit, Tailnet/Runners management, `tailscaled`, restricted Soda project helper |
| Appliance applications | Separate Podman containers for the maintained Forgejo extension host, Soda's Go API service and Caddy |
| Persistent application data | Soda SQLite database and a separate Forgejo volume (including installed extension packages) |
| Identity Broker | Host userspace `soda-identity`, private administration/execution sockets and encrypted subscription custody |
| Factory execution | Unprivileged `soda-factory` operator command, execution ledger, narrow publisher and rootless Podman workspaces |
| Projects | Persistent Project OS containers with project-local accounts, writable roots, SSH and shared installations |
| Project workloads | Nested Podman inside the project |

This table locates existing services. The target replaces disposable factory
workers with execution inside the same persistent Project used by humans; the
[environment model](../product/projects.md#environment-relationships) owns that
relationship. The two runtime paths in the current tree are not a target dual
backend or a compatibility requirement.

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
| Project OS | Human and factory role accounts, tools, persistence, nested workloads |
| Caddy | Private HTTPS termination for configured origins |
| Cockpit | Host administration (all interfaces, root/operator) |
| CI runners | Separately managed capacity for native Forgejo workflows; Soda-provisioned local execution is deferred |

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

## Shared Project execution

One Project execution foundation owns native accounts, container selection,
supervised processes and terminal attachment for human and factory work. The
factory controller owns attempt/run transitions; human terminal control owns the
member's interactive session; the broker owns provider leases. Sharing execution
machinery does not combine those authorities or make browser presence necessary.
Go placement and dependency direction remain governed by
[Go ownership](../development/go.md), not a second runtime abstraction here.

Replace the separate disposable worker execution path when implementing this
model. The existing factory credential return freezes and kills its whole worker;
it cannot be reused unchanged inside a Project containing human sessions and
services. The existing managed human CLI path demonstrates narrower native
execution binding, but does not establish unattended factory compatibility.

Before implementation planning relies on this model, establish these native facts
with focused integration evidence:

- Role accounts can use the approved toolchain and assigned services while native
  permissions protect human state, the other role's Git state and shared resources.
- Source preparation binds an exact repository/candidate to the intended checkout
  and role without shared writable Git metadata or agent access to publisher secrets.
- A broker lease binds to the actual supervised run; credential return and stopping
  all its descendants leave unrelated Project processes running. Restarted containers
  and reused role accounts cannot satisfy an old execution binding.
- Spaces can observe that real CLI through authorized read-only attachments and
  route intervention through factory authority without exposing input or credentials.
- Project Stop/Start and human takeover preserve dirty work and service data while
  invalidating old process bindings and honoring the admission hold.

These are unresolved integration facts, not claims of installed support. If a
selected native boundary is insufficient, revise the design from that evidence
before building orchestration. Missing generic forge authority follows the
[Fountain boundary](trust.md#fountain-consumption-boundary).

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
| Attempt-owned coding checkout and branch | Preserved across sequential runs and pauses under the [Project model](../product/projects.md#persistence); not a human checkout |
| Run-owned review checkout, scratch, processes and transient secrets | Disposable only where exclusively assigned; recorded ownership and confirmed termination/credential return govern cleanup |
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

# Architecture overview

Soda OS is a Fedora CoreOS appliance for human-authorized software work. Forgejo
owns collaboration, Soda owns bounded execution, agents perform tasks and people
authorize work and merge verified results. Persistent Projects support manual
development and intervention through Forgejo, SSH, Git, mise and container tools.

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

**Forgejo is a standalone container, not a Podman pod.** A pod groups containers; it
is not a user database, init system or filesystem. Project containers share the host
kernel; the Project OS distribution supplies userspace.

The `soda-dashboard` command, container, config and data names identify the Go
backend. Do not rename persistent records or roots merely because the UI is called
Spaces.

## Component ownership

| Component | Responsibility |
| --- | --- |
| Forgejo | Identity, Git, collaboration, Actions UI/scheduling, native sessions and extension contributions |
| Soda Forgejo extension | Native browser pages, persistent workspace panel and narrowly scoped calls into the Soda service |
| Soda Go service (`web`, `web/api`, `web/auth`) | Environment/project APIs, membership and operator policy, terminal streams and runner/Tailnet operations |
| `soda-factory` (`factory/control`) | Admission, fixed implementation/verification/repair loop, execution records, cancellation and reconciliation |
| Factory workspace and publisher (`host/workspace`, `host/publish`) | Disposable OCI execution and permitted Forgejo publication, separate from persistent Project authority |
| Identity Broker (`identity/control`) | Explicit subscription ownership/delegation and serialized native execution leases; [credential boundary](../reference/credentials.md#identity-broker) |
| `soda-host` daemon | Privileged project, terminal and Tailnet companion execution |
| Project OS | Developer accounts, tools, persistence, nested workloads |
| Caddy | Private HTTPS termination for configured origins |
| Cockpit | Host administration (all interfaces, root/operator) |
| Local runners | Appliance CI capacity; Forgejo owns workflows and results |

Go package placement for developers: [Go ownership](../development/go.md).

## Factory flow

The operator interface is [the factory command](../reference/factory.md). Forgejo
remains the collaboration control plane; there is no factory dashboard or second
CI scheduler implied by this flow.

1. A human selects Forgejo work for Soda to admit. Forgejo remains the source of
   truth for its issue, branch, pull request, review and CI state. Soda keeps only
   the execution record and resource ledger needed to run and account for the work.
2. Soda assigns a replaceable coding agent to a disposable workspace separate from
   persistent Project roots. Codex is the first agent; OpenCode, Muse Code and
   Oh My Pi are later candidates, not required adapters in the initial path.
3. Agent changes reach Forgejo through the narrow publisher, which checks the
   repository, assigned branch, expected revision and active run. A fresh reviewer
   with a separate identity examines the candidate. At most one bounded repair follows a
   failed check before the work returns to a human.
4. Review and CI evidence bind to the exact candidate commit. A repair receives a
   new review in another fresh workspace and new CI results; older evidence cannot
   authorize it.
5. A human makes the merge decision. Forgejo runs and displays CI; Soda does not
   replace its workflow scheduler or collaboration records.
6. Completion, cancellation or expired limits end execution and publication
   authority. Cleanup is recorded separately from the outcome. Restart handling
   reconciles recorded resources; a human must explicitly admit another attempt.

The factory does not require changing the current Forgejo, Soda, Caddy or Project
container topology. Disposable workspaces and their cleanup must have their own
resource boundary; the Project persistence rules below still apply.

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
| Run checkout, scratch, containers and network | Disposable run resources, reconciled from recorded ownership |
| Retained factory results | Protected factory operator state, separate from disposable resources and public logs |
| Provider connection custody | Broker-owned encrypted database and separate private key; transient enrollment/tool auth in tmpfs |
| Project accounts, homes, tools, service data | Project persistent root |
| Runner registration and local capacity | Host runner state |
| Host OS and layered packages | rpm-ostree / CoreOS |

Persistent application containers and project roots start existing state. Replacement,
`--rm` and pruning are not repair strategies.

Factory workspaces instead receive fresh writable state and are destroyed at the
end of their run. Their cleanup never selects persistent human Projects or
unrelated host resources.

## Related source

- Appliance topology: `appliance/services/`
- Project images and units: `project-os/`
- Entrypoints: `cmd/soda-dashboard`, `cmd/soda-host`, `cmd/soda-factory`

# Architecture overview

Soda OS runs as an operator-managed Fedora CoreOS appliance. Developers use native
Forgejo pages, ordinary SSH, Git, mise and container tools. Soda integrates those
pieces; it does not ask developers to assemble missing account, key or runtime wiring.

Product concepts: [overview](../product/overview.md). Trust and privilege:
[trust](trust.md). Networking: [networking](networking.md). Release:
[release](release.md).

## Topology

| Placement | Mechanism |
| --- | --- |
| Host | Fedora CoreOS, native rpm-ostree layering, Podman, systemd |
| Native operator services | Stock branded Cockpit, Tailnet/Runners management, `tailscaled`, CI runner services, restricted Soda project helper |
| Appliance applications | Separate Podman containers for stock Forgejo, Soda's Go API/OAuth service and Caddy |
| Persistent application data | Separate Soda SQLite database and upstream-owned Forgejo data |
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
| Forgejo | Identity, Git, collaboration, Actions UI/scheduling, native sessions |
| Soda Go service (`web`, `web/api`, `web/auth`) | OAuth adapter, environment APIs, Spaces pages, terminal WS, runner/Tailnet settings |
| `soda-host` daemon | Privileged project, terminal and Tailnet companion execution |
| Project OS | Developer accounts, tools, persistence, nested workloads |
| Caddy | Private HTTPS termination for configured origins |
| Cockpit | Host administration (all interfaces, root/operator) |
| Local runners | Appliance CI capacity; Forgejo owns workflows and results |

Go package placement for developers: [Go ownership](../development/go.md).

## Control and data flow

1. Browser users authenticate with Forgejo. Soda OAuth creates a separate adapter
   session and encrypted grants under `/-/soda/`.
2. Spaces and settings views call protected Soda APIs with expected-actor, CSRF,
   origin and session checks.
3. Mutating project operations go through the host Unix-socket helper as fixed
   operations, not arbitrary commands.
4. Terminal attach allocates a managed tmux session under the member's project
   account; the browser holds a disposable attachment, not shell lifetime.
5. Runner and Tailnet operator settings mutate appliance-local state only for the
   configured Soda operator identity.

## Persistence boundaries

| State | Owner |
| --- | --- |
| Forgejo database and repos | Forgejo volume |
| Soda SQLite, OAuth grants, environment rows | Soda data volume |
| Project accounts, homes, tools, service data | Project persistent root |
| Runner registration and local capacity | Host runner state |
| Host OS and layered packages | rpm-ostree / CoreOS |

Application containers and project roots start existing state. Replacement,
`--rm` and pruning are not repair strategies.

## Related source

- Appliance topology: `appliance/services/`
- Project images and units: `project-os/`
- Entrypoints: `cmd/soda-dashboard`, `cmd/soda-host`

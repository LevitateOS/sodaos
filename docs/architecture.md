# SodaOS architecture

SodaOS provides persistent, shared development environments on an operator-managed
appliance. Developers use native Forgejo pages, ordinary SSH, Git, mise and container
tools. Soda supplies the integration; developers should not have to assemble missing
account, key or runtime wiring themselves.

[Current work](sodaspaces-plan.md) is a **Sodaspaces** repository tab/environment drawer
inside Forgejo's native frontend. Both standalone React and original Go/HTMX Soda
frontends are removed; the Go API/OAuth service remains. The new UI is not implemented.
See the [handoff](implementation-status.md) for source versus installed evidence.

## Topology

| Placement | Implemented mechanism |
| --- | --- |
| Host | Fedora CoreOS, native rpm-ostree layering, Podman and systemd |
| Native operator services | Stock Cockpit plus retained Tailnet/Runners pages, `tailscaled`, CI runner services and restricted Soda project helper |
| Appliance applications | Separate Podman containers for stock Forgejo, Soda's Go API/OAuth service and Caddy |
| Persistent application data | Separate Soda SQLite database and upstream-owned Forgejo data/database |
| Projects | Persistent Rocky + mise containers, project-local accounts/homes, writable roots, SSH and shared installations/files |
| Project workloads | Nested Podman; bounded native x86_64 proof exists, not universal or final-image compatibility |

**Forgejo is a standalone container, not a Podman pod.** A pod groups containers;
it is not a user database, init system or filesystem. Project containers share the
host kernel; Rocky supplies userspace. See `appliance/services/`, `project-os/`,
[installation](installation.md) and [development environment](development-environment.md)
for actual paths, images, units and permissions. Dependency baselines belong in
source recipes/locks, not repeated prose version requirements.

The `soda-dashboard` command/container/config/data names still identify the Go backend.
Do not rename persistent records or roots simply because the UI is called Sodaspaces.

## Authority and identity

- **Forgejo** owns identity, passwords/factors, native sessions, permissions, Git,
  collaboration and administrator workflows. Use supported customization/APIs;
  never access its database directly or copy its business rules into Soda.
- **Soda** owns only its additional preferences, development-access public keys,
  environment associations/memberships and protected adapter sessions/grants.
  Native Git key management is separate. No second password or provider-role inventory.
- **Host root, configured Soda operator, Forgejo site administrator, organization
  owner/admin and repository owner are distinct authorities.** Owning a repository
  does not grant appliance access. Arbitrary site/org administrators are not project
  root. Setup tokens are not ordinary acting-user credentials.
- A repository's human owner can create its project environment. Organization-owned
  creation is currently unsupported. Existing environment/member visibility resolves
  current native ownership by stable repository ID, including organization `is_owner`,
  or explicit Soda operator authority. A stored creator is not permanent authorization.
- Developers have Linux accounts **inside projects**, not human host accounts/homes.
  Stable Forgejo identity binds membership to its original Linux login. Native rename
  or transfer does not silently remap Linux users, ownership or already installed keys.
  Linux eligibility restrictions belong to provisioning, not Forgejo account creation.
- Runtime identities such as Soda UID/GID 2000 and native runner accounts are not
  developer host onboarding. The web process's privileges and human authorization
  are separate boundaries.

## Frontend and session boundary

Use stock Forgejo's native handlers, forms, scripts/styles and authentication, with
supported dedicated template hooks first and necessary targeted overrides second.
Do not deploy unchanged copies of the template tree. See [integration details](forgejo-frontend-integration.md).
A missing JSON endpoint does not imply a missing native workflow.

**No downstream Forgejo fork, source patch set or custom executable.** If supported
integration cannot meet a requirement, explain its actual constraint and return
for a decision. Do not substitute scraping, an HTML relay, borrowed cookies, an
iframe or weakened native security. No replacement component library is selected.

Keep OAuth state/PKCE/callback binding, encrypted session-bound grants, serialized
refresh, logout-winning persistence, request/response bounds and CSRF/origin checks.
Native OAuth tokens are not native web sessions. Different ports do not isolate
cookies. Root and successful Soda OAuth completion currently redirect only to
configured native Forgejo; that is not a completed authenticated drawer connection.
Native WebAuthn origins/RP-ID, session revocation and Git protocols remain upstream-owned.

## Projects and explicit joining

**Add me to this project** must install the person's development public keys and
create the intended real project-local Linux account, then record membership only
after confirmed native success. A row, mock or manual-command checklist is not the
feature. The creator also explicitly joins. Never request a private SSH key.

The root:soda Unix-socket helper exposes fixed operations, not arbitrary commands,
host Podman flags or a generic forwarding surface. Resolve actor and native target
from trusted state. Existing Linux accounts/homes/permissions remain native facts;
report incomplete provisioning/persistence honestly rather than claiming success
or destructively recreating a reservation.

Joining is separate from native Git authorization. Later key rotation/revocation,
Linux offboarding and unrelated active-session termination are not automatically
synchronized. The requested browser terminal uses the user's existing project-local
account/home; opening it must not create, join, start a project or expose host root.

## Shared resources and persistence

Projects are lasting, mutable environments, not disposable per-user Codespaces.
Personal ordinary checkouts coexist with actual shared installed tools, files and
services. A shared download cache or matching version strings alone is insufficient.
Keep native mise/OS installation, Git and workload definitions/commands; Soda does
not interpret merges as live-state promotion, service restart or cleanup.

Normal startup starts the **existing** container. Preserve accounts, homes, SSH host
keys, installed packages/tools, shared files, configuration, dirty work and service
volumes/data across stop/start and host reboot. Never use `--rm`, `--replace`, pruning
or deletion as repair. Native workloads and database/HTTP access remain product scope;
see [project services](project-services.md) and [validation](native-validation.md).

The runtime is a trusted-team namespaced boundary, not hostile-tenant isolation.
Investigate project-scoped host workloads only after a concrete nested-runtime blocker.
No dormant fallback, unrestricted host socket, privileged parent or VM substitute is
implemented or authorized. Bring a genuine architecture gap back for a decision.

## Networking and operator tools

Project access is ordinary `user@project-ip`, SSH/PTY/SCP/SFTP and native service
ports—not project DNS or a custom SSH gateway. An observed bridge address does not
prove client reachability. Host Tailnet enrollment does not imply an advertised,
approved or working project subnet route. Keep actual client/route evidence explicit.

Browser/OAuth origins, listeners and Forgejo Git advertisement are distinct configured
facts. Do not infer endpoints from predecessor ports or another browser hostname.
Cockpit is loopback-first and root/operator-only; do not silently expose administration
or development services publicly. Retain Tailnet/Runners backing logic, native service
wiring, dependencies and tests, not just page appearance. Providers own CI workflows,
registration authority, scheduling and results; Soda owns local capacity, not a scheduler.

## Scope, safety and proof

[Deferred scope](deferred.md) excludes managed private-resource branches/selectors,
merge automation, generalized reconciliation/recovery/identity remapping, project
archival/deletion and a new release/update platform. Deferral does not remove ordinary
authorization, validation, error handling or persistence requirements.

[Native validation](native-validation.md) owns product journeys. Outside
[support tools](native-support.md) supply transport/artifacts/evidence, not copied
product tests or another readiness gate. Native x86_64 and aarch64 are independent
targets; cross-build/emulation is not installed proof and one need not block the other.
Rehearse preserved-state changes before separately approved deployment. Build/test,
install/restart, routing, provider actions and exact cleanup have distinct permission
scopes. Keep secrets in restricted files/input channels, never argv/logs/screenshots.
Preserve all unrelated work, private inputs, backups and failed evidence.

The predecessor repository is separate. Preserve [reuse attribution](predecessor-reuse.md),
[licenses](licensing.md) and canonical artwork; do not modify it, close its issues or
import its separately reserved Updates platform. Historical architecture/plans remain
in Git; they are not current implementation mandates or execution permission.

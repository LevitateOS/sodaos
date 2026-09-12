# SodaOS architecture

SodaOS provides persistent, shared development environments on an operator-managed
appliance. Developers use native Forgejo pages, ordinary SSH, Git, mise and container
tools. Soda supplies the integration; developers should not have to assemble missing
account, key or runtime wiring themselves.

The user explicitly reaffirmed that this is a **development appliance**. The
[Services catalog](services-and-ai-plan.md#development-appliance-scope--user-decision)
helps people run useful tools on their local network, with optional Tailscale
access. Trying SodaOS or using its baseline Services must not require buying or
owning a domain. Public Internet hosting and a production hosting platform are
outside this design. Private access must still work, preserve application data and
respect native authentication; solve app-specific requirements within that scope.

The [CoreOS product strategy](os-product-strategy.md) records the user's custom-distro
direction and proposed host features, their value, effort and limits as service
features. It distinguishes today's upstream-based delivery from future boot/storage/
recovery ownership. It does not replace the current implementation plan or select
the deferred features for execution.

[Sodaspaces](sodaspaces-plan.md) provides the repository environment drawer and
shared workspace inside Forgejo's native frontend. The Go API/OAuth service remains;
neither a standalone React nor Go/HTMX Soda frontend is selected.

The [native page integration guide](forgejo-soda-pages-plan.md) owns the dashboard
hosts, fixed bookmark bridges, shared connection and coordinated logout. Forgejo
supplies its actual header, profile menu and native authentication; Lit supplies
Soda's management/workspace views. Installed revisions, delivery and acceptance
status belong only to the [current handoff](implementation-status.md).

The [Lit implementation sequence](lit-migration-plan.md) retains the shared page/
drawer workspace and native terminal acceptance obligations. The
[frontend improvement guide](frontend-improvement-plan.md) owns token consolidation,
template checking and typed composition. This port changes presentation and entry,
not project or runner backend responsibilities or atomic authentication semantics.

The [terminal guide](terminal-integration.md) owns the selected tmux mechanism:
a private supervised server under each original project account. Soda retains
access/lifetime authority; tmux retains live terminal state. This is not a
replacement for ordinary SSH or a separate public terminal server.

## Topology

| Placement | Implemented mechanism |
| --- | --- |
| Host | Fedora CoreOS, native rpm-ostree layering, Podman and systemd |
| Native operator services | Stock Cockpit plus Tailnet, native Forgejo Runners management, `tailscaled`, CI runner services and restricted Soda project helper |
| Appliance applications | Separate Podman containers for stock Forgejo, Soda's Go API/OAuth service and Caddy |
| Persistent application data | Separate Soda SQLite database and upstream-owned Forgejo data/database |
| Projects | Persistent Rocky + mise containers, project-local accounts/homes, writable roots, SSH and shared installations/files |
| Project workloads | Nested Podman; bounded native x86_64 proof exists, not universal or final-image compatibility |

**Forgejo is a standalone container, not a Podman pod.** A pod groups containers;
it is not a user database, init system or filesystem. Project containers share the
host kernel; Rocky supplies userspace. The [Project OS baseline](project-os.md)
consolidates native ownership, supported tools, persistent/runtime state and bounded
same-root maintenance. Its [selected creation profiles](project-os.md#selected-environment-profiles)
extend requested scope to Rocky/Fedora headless and KDE, with GNOME deferred.
Only Rocky headless is currently implemented. All profiles inherit the
[same Project OS foundation](project-os.md#one-foundation-for-every-profile): native
accounts, shared tools/services, persistent roots, access and maintenance. KDE adds
a per-user graphical session to that environment. No VM backend is selected by the
profile decision; first investigate the existing runtime and bring a concrete
compatibility blocker back for a decision. CoreOS and existing projects keep their topology.

Profiles are [batteries included](project-os.md#batteries-included-by-default):
Soda supplies the complete non-preference development and interface foundation.
Personal application choices, accounts and repository versions remain native user
decisions. A minimal image plus manual prerequisite instructions is not a completed
Project OS, and selecting an optional supported app must include its dependencies.
See `appliance/services/`, `project-os/`, [installation](installation.md) and
[development environment](development-environment.md) for implementation and usage.
Dependency baselines belong in source recipes/locks, not repeated prose version rules.

The `soda-dashboard` command/container/config/data names still identify the Go backend.
Do not rename persistent records or roots simply because the UI is called Sodaspaces.

The [settings contract](sodaspaces-plan.md#settings-pages-and-os-selection) places
Sodaspaces and CLI-based AI automation in repository settings and local Sodarunners
capacity in global Soda-operator settings. “Move runners to the dashboard” means
this bounded native-interface extension, not reviving either removed frontend.
Forgejo retains Actions settings/scheduling/permissions. The [combined plan](native-pages-runners-plan.md#6-retire-only-the-cockpit-runner-presentation)
owns Cockpit Runners retirement; Tailnet stays in Cockpit. Marketplace apps, persistent
Project OS roots, isolated AI jobs and account-owned desktop sessions have distinct
native lifetimes and credentials; sharing UI does not combine their privileges.

The backend also renders [original robot avatars](avatars.md) from embedded SVG
parts using Forgejo's supported provider setting. Only the avatar namespace is
public and credential-stripped on the Forgejo origin. Protected Sodaspaces API/OAuth
and terminal routes share that origin under `/-/soda/` with their normal credentials.

## Authority and identity

- **Forgejo** owns identity, passwords/factors, native sessions, permissions, Git,
  collaboration and administrator workflows. Use supported customization/APIs;
  never access its database directly or copy its business rules into Soda.
- **Soda** owns its additional identity/access records: preferences, development-access
  public keys, environment associations/memberships and protected adapter sessions/grants.
  Its appliance integration also owns local runner state/lifecycle and the selected
  catalog recipes/installed configuration. Native Git keys and collaboration remain
  upstream-owned. No second password, provider-role inventory or CI scheduler.
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
  Current web administration and native wheel/SSH rights are not automatically
  synchronized on transfer; see the [native authority boundary](project-os.md#ownership-and-trust).
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
iframe or weakened native security. Management and terminal controls now use
[Lit](lit.md), loaded on demand; the page and drawer share a multi-session workspace
with flat terminal owners. Xterm/transport remain imperative. The Lit and terminal
guides own their rendering and session contracts; the handoff records evidence.

Keep OAuth state/PKCE/callback binding, encrypted session-bound grants, serialized
refresh, logout-winning persistence, request/response bounds and CSRF/origin checks.
Native OAuth tokens are not native web sessions. Different ports do not isolate
cookies. Root redirects to configured native Forgejo home; OAuth can return to a
repository resolved by stored ID through the acting grant, never a supplied URL.
Soda's expected-user header guards page/session consistency, not native browser
session authenticity. Native WebAuthn origins/RP-ID, session revocation and Git
protocols stay upstream-owned.

The bookmark handlers redirect only to fixed native views. Private collection and
operation authority stays in the existing protected APIs, including original-actor,
CSRF, scope and current-session checks. Native UI visibility does not confer Soda
operator or project authority. Page loads and redirects never register a runner,
create a terminal or change project lifecycle state. No fabricated native context,
HTML relay, borrowed cookie or replacement password authority is used.

## Projects and explicit joining

**Add me to this project** must create the intended real project-local Linux account
and install any explicitly selected external-SSH public keys, then record membership
only after confirmed native success. Account-only Join without external SSH keys
follows the [Project OS access contract](project-os.md#access-credentials-and-connectivity).
A row, mock or manual-command checklist is not the feature. The creator explicitly
joins as well. Never request a private SSH key.

The root:soda Unix-socket helper exposes fixed operations, not arbitrary commands,
host Podman flags or a generic forwarding surface. Resolve actor and native target
from trusted state. Existing Linux accounts/homes/permissions remain native facts;
report incomplete provisioning/persistence honestly rather than claiming success
or destructively recreating a reservation.

Joining is separate from native Git authorization. Later key rotation/revocation,
Linux offboarding and unrelated active-session termination are not automatically
synchronized. The requested browser terminal uses the user's existing project-local
account/home; opening it must not create, join, start a project or expose host root.
Its [resumable terminal plan](sodaspaces-plan.md#resumable-terminal-decision--tmux)
is separate from ordinary SSH access and does not collect private SSH keys.

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
No dormant workload fallback, unrestricted host socket or privileged parent is
implemented or authorized. The requested Linux desktop integration is scoped in the
[desktop design](services-and-ai-plan.md#4-desktop-workspaces) and must preserve the
Project OS contracts. A VM remains an architectural option requiring a concrete
runtime decision, not a selected desktop or nested-workload replacement.

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
product tests or another readiness gate. [AGENTS.md](../AGENTS.md#permissions-and-preservation)
owns execution, preservation and evidence policy; the handoff records current grants.
[Installation](installation.md#retained-sodaspaces-cutover) owns maintenance procedures.

The predecessor repository is separate. Preserve [reuse attribution](predecessor-reuse.md),
[licenses](licensing.md) and canonical artwork; do not modify it, close its issues or
import its separately reserved Updates platform. Historical architecture/plans remain
in Git; they are not current implementation mandates or execution permission.

# SodaOS architecture

SodaOS provides persistent, shared development environments on an operator-managed
appliance. Developers use native Forgejo pages, ordinary SSH, Git, mise and container
tools. Soda supplies the integration; developers should not have to assemble missing
account, key or runtime wiring themselves.

The [CoreOS product strategy](os-product-strategy.md) records the user's custom-distro
direction and proposed host features, their value, effort and limits as service
features. It distinguishes today's upstream-based delivery from future boot/storage/
recovery ownership. It does not replace the current implementation plan or select
the deferred features for execution.

[Current work](sodaspaces-plan.md) is a **Sodaspaces** repository button/environment drawer
inside Forgejo's native frontend. Both standalone React and original Go/HTMX Soda
frontends are removed; the Go API/OAuth service remains. A read-only native drawer
was implemented and passed native x86_64 build/stage and isolated exported-payload
browser checks. The complete mounted management/terminal UI now has bounded native
x86_64 OAuth, Create/Join, lifecycle, key-revocation and SSH/PTY/transfer proof on the
isolated fixture; retained delivery and broader product acceptance remain separate.
See the [handoff](implementation-status.md) for source versus installed evidence.

The user also selected a global **Spaces** link and authenticated Soda-owned
Go/template page at `/-/soda/spaces`, sharing the existing workspace drawer/sessions.
This is a bounded extension of the API backend, not restoration of Forgejo workflow
adapters or either removed frontend. The HTML shell, fixed OAuth return, shared
multi-session workspace, bounded `/api/spaces` collection and ID-keyed contracts
now have local source/test coverage, not concurrent native proof; [the leading plan](sodaspaces-plan.md#spaces-page--selected-not-implemented)
owns this scope. The [Lit implementation sequence](lit-migration-plan.md) now covers
both surfaces; steps 1–5 and 6a/6b, including layouts, observed attention and extended
journey source ports, have local coverage. The [frontend improvement guide](frontend-improvement-plan.md)
preserves that baseline while defining mandatory token consolidation, template
checking and typed composition before further UI expansion.
The selected [native session mechanism](sodaspaces-plan.md#resumable-terminal-decision--tmux)
is stock Rocky tmux under each original project account, with a private supervised
server per managed browser terminal. Soda retains access/lifetime authority; tmux
retains the live terminal state. This source is implemented with bounded isolated
`22d8591` same-shell reload/cleanup evidence; broader native safety/UX proof remains
open. It is not a replacement for ordinary SSH or a separate public terminal server.

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

The backend also renders [original robot avatars](avatars.md) from embedded SVG
parts using Forgejo's supported provider setting. Only the avatar namespace is
public and credential-stripped on the Forgejo origin. Protected Sodaspaces API/OAuth
and terminal routes share that origin under `/-/soda/` with their normal credentials.

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
with flat terminal owners. Xterm/transport remain imperative; measured layouts and
compact projections and observed attention have local coverage, while broader
native/CLI acceptance remains outstanding. Local rendering tests are not installed
CLI compatibility or full terminal acceptance.

Keep OAuth state/PKCE/callback binding, encrypted session-bound grants, serialized
refresh, logout-winning persistence, request/response bounds and CSRF/origin checks.
Native OAuth tokens are not native web sessions. Different ports do not isolate
cookies. Root redirects to configured native Forgejo home; OAuth can return to a
repository resolved by stored ID through the acting grant, never a supplied URL.
Soda's expected-user header guards page/session consistency, not native browser
session authenticity. Native-page wiring passed the isolated local browser journey; native
WebAuthn origins/RP-ID, session revocation and Git protocols stay upstream-owned.

The Spaces HTML handler uses Soda's own session/acting grant to authorize its shell;
listing and actions remain protected APIs. Its fixed OAuth return is transaction-bound
through an append-only schema-v6 flag. The selected shell uses
canonical Soda assets, configured-origin native links and an explicitly labelled
Soda account—not a fabricated Forgejo navbar/account/notification context. Loading
assets cannot supply native CSRF or authentication, and template overrides cannot
install Go handlers upstream. Lit renders only the Soda workspace. Page-only CSP,
styles/clipboard and fixed return have local checks; installed integration remains pending;
no copied native authentication logic, HTML relay or borrowed cookies/tokens. Existing JSON actor/CSRF protection and independent logout boundaries
remain intact. A future global Runners link/page must enforce the configured Soda
operator boundary server-side; Forgejo site administration is not a substitute.

## Projects and explicit joining

**Add me to this project** must create the intended real project-local Linux account
and install any explicitly selected external-SSH public keys, then record membership
only after confirmed native success. The current implementation still requires a
nonempty key set; the selected browser-only account path is unimplemented. A row,
mock or manual-command checklist is not the feature. The creator explicitly joins
as well. Never request a private SSH key.

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

# Product scope

This document owns feature disposition and scope for the target architecture.
It is a scope boundary, not a backlog tracker and not permission to strip working
validation or error handling.

The automatic issue-to-merge loop, including dependency and requirements blockers,
environment reuse, visible CLI execution, review/fix and reconsideration of waiting
issues, is core scope. Its behavior is owned by the
[factory product contract](overview.md#software-factory-workflow).

## Feature disposition

These decisions define the first rebuilt factory and its supporting product.
They do not assert that the target behavior is implemented or authorize an
implementation sequence. Existing interface behavior remains in the references.

- **Retain:** keep the capability and its owning contract. Integration changes may
  still be needed when consuming Fountain.
- **Adapt:** keep the capability's purpose but change its workflow or integration
  to serve the target factory.
- **Retire:** remove the obsolete implementation, callers and fixtures together;
  do not preserve a parallel legacy path.
- **Defer:** exclude the expansion from the first working factory. Do not expose
  a placeholder as usable functionality or make it a prerequisite for that path.

### Factory, identity and collaboration

| Capability | Decision | Contract and rationale |
| --- | --- | --- |
| Native repositories, issues, PRs, reviews, permissions and Actions | Retain | Fountain supplies native Forgejo collaboration; Soda consumes it rather than maintaining competing records. [Ownership](overview.md#what-soda-owns-vs-forgejo). |
| Soda extension pages, panel and native authorization | Adapt | Consume Fountain's generic host and authority contracts, including unattended operations; correct Soda callers before declaring host gaps. [Trust](../architecture/trust.md#fountain-consumption-boundary). |
| Work items, attempts, runs and resource accounting | Adapt | Preserve recorded objectives, revisions, bounded execution and duplicate handling; admit and reconsider work through readiness policy instead of mandatory operator commands. [Factory lifecycle](overview.md#software-factory-workflow). |
| Candidate publication, reviews and CI evidence | Adapt | Keep scoped native writes and evidence tied to the candidate commit; add automatic native merge and dependent-work reassessment within authorized policy. [Factory authority](../architecture/trust.md#factory-authority-boundary). |
| Coding, review and correction sessions | Adapt | Use the prepared repository environment and another reviewer session in the same container. The review/fix protocol remains a separate design decision. [Factory lifecycle](overview.md#software-factory-workflow). |
| Cancellation, limits, outcomes and cleanup | Adapt | Retain bounded work and attributable outcomes; cleanup releases owned run resources without destroying reused environments or unrelated state. [Persistence boundaries](../architecture/overview.md#persistence-boundaries). |
| Identity Broker enrollment, account custody and provider selection | Retain | Keep supported Codex and Muse subscription connections, explicit sponsorship and no silent billing/account/provider fallback. Native forge sign-in remains separate. [Credentials](../reference/credentials.md#identity-broker). |
| Broker execution leases and credential return | Adapt | Bind credentials to the assigned agent session in a reusable environment. Preserve provider-specific concurrency and revocation rules; whole-container termination cannot remain the universal return mechanism. [Credentials](../reference/credentials.md#identity-broker). |

### Spaces, project environments and access

| Capability | Decision | Contract and rationale |
| --- | --- | --- |
| Spaces page and persistent Workspace panel | Adapt | Preserve native navigation and terminal continuity; expose factory CLI activity, issue/PR context, blockers and intervention alongside human work. [Spaces](spaces.md). |
| Human terminals, tmux persistence, tabs and splits | Retain | Human development, debugging and intervention remain product functions. Viewing, hiding and ending have distinct effects; automation does not commandeer personal shells. [Terminal](../reference/terminal.md). |
| Project-local accounts, explicit human Join and membership | Retain | Keep real accounts and current repository-write admission, stable identity bindings and no developer host accounts. Agent identities do not implicitly become human members. [Projects](projects.md#explicit-joining). |
| Development-access public keys and ordinary SSH/SCP/SFTP | Retain | Support external editors and direct project access with explicit public-key installation. These keys are separate from native Git and AI account credentials. [Project access](projects.md#access). |
| Native Git authentication and packaged `tea`/`gh` CLIs | Retain | People use their own native credentials. Factory publication uses separately authorized native authority; do not restore Soda-managed Git credential mediation. [Project CLIs](../guides/project-clis.md). |
| Repository environment creation and preparation | Adapt | Keep the repository association and controlled lifecycle; automatically prepare or reuse an appropriate environment with the required checkout, tools and services. Human Join remains explicit. [Projects](projects.md). |
| Rocky headless Project OS | Retain | Use the supported userspace foundation for the first factory. The host remains Fedora CoreOS; project userspace is a separate concern. [Selected profiles](projects.md#selected-profiles). |
| Persistent homes, dirty work, shared mise tools, packages and files | Retain | Agent automation does not make development state disposable. Start/stop and maintenance preserve the existing root. [Project OS](../reference/project-os.md#persistent-state-and-lifecycle). |
| Nested workloads, Compose/systemd services and volumes | Retain | Repository databases and services belong to the development environment; preserve their data and native administration. Agent access must be explicitly assigned. [Project services](../guides/project-services.md). |
| Personal checkouts and factory checkout/worktree allocation | Adapt | Preserve human Git workflows while assigning issue work without overwriting unrelated changes. Concurrency and sharing policy remain to be designed. [Environment decisions](overview.md#rules-to-settle-before-implementation). |

### Appliance, networking, CI and delivery

| Capability | Decision | Contract and rationale |
| --- | --- | --- |
| Private LAN access, project IPs and native service ports | Retain | A factory still needs reachable development services and human access. An observed address is not proof of a working route. [Networking](../architecture/networking.md). |
| Caddy/private HTTPS and client trust setup | Retain | Keep one configured native browser origin, local or supplied certificates and explicit client trust. Baseline use requires no purchased domain or public hosting. [Operator setup](../guides/operator-setup.md). |
| Optional host Tailnet and project companions | Adapt | Preserve operator policy and project-scoped enrollment through native extension controls. Keep opt-in behavior and separate device identities; factory readiness must not require Tailscale where LAN connectivity suffices. [Tailnet model](../architecture/networking.md#tailnet-model). |
| Native Forgejo CI with separately managed capacity | Retain | Required checks must run and identify the candidate commit. Soda observes Actions; coding/review agents are not CI runners. [Runners](../reference/runners.md). |
| Soda-provisioned local CI execution and Runner OS | Defer | Keep the isolated-job requirement. Complete the first factory using separately managed Actions capacity; do not restore shared host-account execution to obtain local capacity. [Runner boundary](../reference/runners.md#deferred-execution-boundary). |
| Operator bootstrap, appliance identity and privileged helper | Retain | Keep explicit setup, the configured operator boundary, restricted secret inputs and fixed privileged operations. Factory sponsorship is a separate authority decision. [Trust](../architecture/trust.md). |
| Stock Cockpit and host diagnosis/recovery | Retain | Keep native host administration under operator access, including logs, storage, networking and services. [Cockpit](../development/cockpit.md). |
| Soda branding, avatars and console guidance | Retain | Preserve the existing presentation and attribution through supported host/extension mechanisms. These do not justify a separate Soda shell or login system. [Branding](../design/branding.md). |
| Build, installation, updates, backup and recovery | Retain | Preserve the CoreOS candidate and native qualification model, architecture-specific evidence, existing-state protection and ordinary backup responsibilities. Runtime factory automation does not imply production deployment or release publication. [Release](../architecture/release.md). |

## Retire

- **Legacy forge authentication and Soda-managed Git:** the removed OAuth login,
  parallel sessions, login relays and Git credential mediation stay removed. The
  Identity Broker's supported AI provider custody and Tailnet provider enrollment
  are different capabilities and remain under their own contracts.
- **The old factory as a separate mandatory workflow:** retire manual admission
  per issue, mandatory human merge, fixed one-repair policy and a fresh disposable
  container for every stage. Useful operator actions may address the same current
  factory; they must not preserve a second execution engine or legacy format.
- **Shared host-account runner execution and experimental compatibility:** retire
  remaining obsolete artifacts and callers with their replacement or removal.
  Any real experimental resource cleanup is explicit maintenance, not a permanent
  alternative CI subsystem. The existing cleanup behavior remains described
  by the runner reference until it is removed.
- **Superseded presentation and forge patches:** retire the standalone Soda
  dashboard/login shell, custom Cockpit project pages and Soda-specific copies or
  patches of native Forgejo behavior superseded by Fountain's generic extension
  contract. Keep Soda's own extension assets and supported branding.

## Deferred

1. **Private toolchain and service branches** — temporary personal branches of shared
   tools or services that return to the shared environment after integration.
2. **Access-lifecycle edge cases** — automatic synchronization of key rotation,
   Linux offboarding and unrelated active-session termination across systems.
3. **Identity and ownership remapping** — silent remapping of Linux users when
   Forgejo usernames or repository ownership change.
4. **General operation-recovery machinery** — fleet-wide reconciliation, repair
   orchestrators and automatic destructive recovery.
5. **Broader recovery and lifecycle management** — project archival/deletion
   programmes and general fleet orchestration beyond the selected release model.
6. **Factory expansion** — production deployment after merge, multihost execution,
   general-purpose orchestration beyond the factory loop, workspace snapshots or
   conversation resume, and execution of hostile external contributions. The
   factory targets a trusted team on one operator-managed appliance.
7. **Additional Project OS profiles and graphical desktops** — Fedora headless,
   Rocky/Fedora KDE and GNOME. The first factory uses Rocky headless; neither a
   profile label nor a design sheet establishes a supported image or desktop path.
8. **Additional broker/provider adapters** — keep the supported provider routes;
   expand only after the native authentication, subscription and execution contract
   is established. Extra adapters do not gate the first automatic factory.

Local runner execution is deferred as specified in the disposition table above.
These deferrals do not remove working human project access or optional private
networking.

## Not pursuing

- Public Internet application hosting as a Soda product surface
- Mutually untrusted tenant hosting
- Domain ownership or preprovisioned per-app public certificates as a prerequisite
  for trying Soda or using baseline services
- A second standalone Soda web frontend
- Predecessor host developer accounts, Cockpit Projects, managed checkouts or the
  predecessor Updates platform

## Still required

Deferral does not remove:

- authorized factory operation, distinct execution identities, time and resource
  limits, cancellation and reconciliation of recorded run resources
- exact-candidate verification and review, native merge authorization and an
  understandable intervention outcome when work cannot finish
- ordinary authorization, validation and honest error handling
- persistence and no-destructive-repair rules for project roots
- native installation, update and recovery qualification for the release candidate
- usable project-local accounts, SSH, shared tools, reachable project addresses and
  persistent project state

Release and update architecture: [Release](../architecture/release.md).
Host capability strategy (non-normative): [Host strategy](../research/host-strategy.md).

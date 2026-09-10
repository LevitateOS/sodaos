# Local CI runners

## Selected settings destination

Move Soda's local capacity/service controls to **Global SodaOS settings →
Sodarunners**, not repository settings. The
[settings plan](sodaspaces-plan.md#settings-pages-and-os-selection) owns integration.
The configured Soda operator is a separate authority from a Forgejo site/repository
administrator; enforce it server-side for listing, status and all mutations.
Expose only the existing supported controls and actually installed execution
capabilities, including the current host-only limitation. OCI AI execution remains
separate source/native work, not an option enabled by a settings label.

Keep native repository **Actions → Runners**, secrets and variables for provider-owned
scope and workflows. Repository AI configuration can refer to supported runner labels
and show useful availability/errors; it does not register host accounts, allocate
arbitrary host capacity or replace Forgejo scheduling/registration authority.

Reuse the existing runner coordinator/helper/provider clients, secret-input handling,
service lifecycle and focused tests. Preserve the Cockpit Runners page until the new
operator surface reaches real effects and its removal is coordinated. Tailnet stays
in Cockpit. A page relocation does not change one-slot capacity, runner privileges,
deletion semantics or authorize provider mutations. The port record below describes
existing mechanisms; the global settings replacement is not implemented yet.

## Existing port and operation

The predecessor's page, protocol, coordinator/helper/launch, native provider clients, lifecycle and focused tests are ported. Adaptations:

- Only the native root/operator may administer runners. The old regular-UID administrator check was incompatible with this appliance and has been replaced together with its tests.
- Root invokes the bounded native helper directly; non-operator requests are rejected. No human host account provisioning or polkit grant for project owners is added.
- The coordinator reads the configured internal/public Forgejo origins from `/etc/soda/dashboard.json`. The page consumes the configured browser URL, not a guessed `hostname:30000` link. Registration still belongs to Forgejo/GitHub.
- New executables and GitHub client files stage under writable native `/usr/local` paths. Fedora's native `forgejo-runner` package supplies `/usr/bin/forgejo-runner`; the selected predecessor source used 12.13.2 on Fedora 44. Actual host package closure must be checked later.
- Each runner has one slot, a noninteractive unprivileged runtime account and persistent native state. Jobs have network access and execute repository code: use only trusted repositories/contributors. They are not project workspaces.

GitHub client version 2.337.0 and both native archive/checksum inputs are retained in `appliance/locks/github-runner-source.toml`. No archive was downloaded, installed or executed during the port. The later build recipe stages that exact client; existing runner copies retain their own version.

For later operation: create the native registration token/UUID in Forgejo or GitHub; use operator Cockpit **Runners** to register/start, inspect status/capacity and control the local service. Tokens go through the existing secret-aware input path, not command arguments or browser persistence. Removing a local runner is explicitly destructive to its local files; provider history/registration cleanup remains provider-owned.

No native runner account, provider registration or job has been created by this source work. Existing/new tests remain unexecuted.

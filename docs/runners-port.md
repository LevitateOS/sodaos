# Local CI runners

The predecessor's page, protocol, coordinator/helper/launch, native provider clients, lifecycle and focused tests are ported. Adaptations:

- Only the native root/operator may administer runners. The old regular-UID administrator check was incompatible with this appliance and has been replaced together with its tests.
- Root invokes the bounded native helper directly; non-operator requests are rejected. No human host account provisioning or polkit grant for project owners is added.
- The coordinator reads the configured internal/public Forgejo origins from `/etc/soda/dashboard.json`. The page consumes the configured browser URL, not a guessed `hostname:30000` link. Registration still belongs to Forgejo/GitHub.
- New executables and GitHub client files stage under writable native `/usr/local` paths. Fedora's native `forgejo-runner` package supplies `/usr/bin/forgejo-runner`; the selected predecessor source used 12.13.2 on Fedora 44. Actual host package closure must be checked later.
- Each runner has one slot, a noninteractive unprivileged runtime account and persistent native state. Jobs have network access and execute repository code: use only trusted repositories/contributors. They are not project workspaces.

GitHub client version 2.337.0 and both native archive/checksum inputs are retained in `appliance/locks/github-runner-source.toml`. No archive was downloaded, installed or executed during the port. The later build recipe stages that exact client; existing runner copies retain their own version.

For later operation: create the native registration token/UUID in Forgejo or GitHub; use operator Cockpit **Runners** to register/start, inspect status/capacity and control the local service. Tokens go through the existing secret-aware input path, not command arguments or browser persistence. Removing a local runner is explicitly destructive to its local files; provider history/registration cleanup remains provider-owned.

No native runner account, provider registration or job has been created by this source work. Existing/new tests remain unexecuted.

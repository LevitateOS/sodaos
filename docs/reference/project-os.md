# Project OS baseline

Project OS is the userspace foundation inside every Soda project: native accounts,
shared tools, persistent roots, access and nested workloads.

Product model: [Projects](../product/projects.md). Everyday use:
[Develop](../guides/develop.md). Terminal contract: [Terminal](terminal.md).

## One foundation for every profile

Distribution and desktop choices extend this foundation. They do not create
independent machines, new account systems or a second development product.

| Contract | Owner |
| --- | --- |
| Accounts, permissions, shared installations, persistence, maintenance | This document |
| Personal checkouts, editors, mise workflows | [Develop](../guides/develop.md) |
| Nested workloads and project networking | [Project services](../guides/project-services.md) |
| Packaged forge CLIs | [Project CLIs](../guides/project-clis.md) |
| Managed terminal lifetime | [Terminal](terminal.md) |
| Spaces presentation | [Spaces](../product/spaces.md), [Spaces UX](../design/spaces-ux.md) |

A desktop session belongs to the existing project-local account. Terminal, desktop
apps and ordinary SSH use that account's real home, checkouts, permissions and shared
tools. A desktop is not a large terminal lease: it has its own display/input scope.
Ending a desktop must not end managed tmux terminals, SSH sessions, user services or
nested workloads, and the reverse is also true.

## Batteries included

Every supported profile arrives as a complete working Project OS. Soda installs and
wires the non-preference foundation.

| Supplied by Soda | Remains a user or repository choice |
| --- | --- |
| Native account/home/group setup, locale, shell startup, permissions, CA trust | Personal dotfiles, preferred shell, appearance |
| Working terminal, tmux, basic editor/pager, diagnostics | Preferred editor/IDE, keybindings, theme |
| Git, packaged forge CLIs, shared mise, standard build/debug tools | Repository language versions, build commands, trust decisions |
| Native workload engine, Compose support, storage and access integration | Which services to run and how they are configured |
| On KDE: functioning user desktop, display/input transport, fonts, clipboard, file manager, basic graphical editor, browser | Preferred apps, browser profile, personal accounts |

Missing standard build prerequisites are a packaging gap, not a task for each project
administrator. Dependency baselines belong in source recipes and locks.

## Profiles

| Profile | Distribution | Interface |
| --- | --- | --- |
| Rocky headless | Rocky Linux | Terminal (default) |
| Rocky KDE | Rocky Linux | Terminal and KDE Plasma |
| Fedora Server | Fedora Linux | Terminal / headless |
| Fedora KDE | Fedora Linux | Terminal and KDE Plasma |

GNOME is deferred. Mise is required for every supported profile. The creation UI
exposes a single Project OS dropdown; server-side code resolves a bounded profile ID
to installed artifacts. Profiles are not runtime selectors and do not by themselves
select a VM backend.

## Ownership and trust

| Owner | Responsibility |
| --- | --- |
| Appliance/operator | CoreOS/kernel, host Podman/network/storage, restricted helper |
| Project OS | systemd PID 1, accounts/groups, SSH, shared tools, nested engine, managed terminals |
| Soda backend | Repository/project/user associations, membership login, browser authorization, terminal leases |
| Developers / project administrators | Personal checkouts, preferences, credentials; shared package/tool/service admin via project permissions |

This is a **trusted-team container sharing the host kernel**, not hostile-tenant
isolation. Project root/wheel can read every project home and credential. Unix
permissions protect against ordinary peers, not project sudo or appliance root.

New members get Bash, a real home and `soda-project` group membership. Unassociated
account collisions refuse instead of adopting an existing Linux user. Wheel grants
follow the creation-time owner label and are not automatically synchronized when
Forgejo ownership transfers.

Source owners include `internal/host/`, `project-os/rootfs/usr/libexec/soda/`,
`internal/web/api/` and project sudoers under `project-os/rootfs/etc/sudoers.d/`.

## Supported userspace

- Keep the distribution's native package mechanisms, systemd, OpenSSH, Git, Bash and
  OS Python. Versions live in `project-os/Containerfile` and locks.
- Shared mise lives in `/opt/mise` with global config in `/etc/mise/config.toml`.
- Keep personal Git clones, repository `mise.toml`, explicit trust decisions and
  native Compose/systemd files.
- Managed browser terminals use stock packaged tmux under Soda supervision; do not
  force ordinary SSH logins into managed sessions.

## Persistent state and lifecycle

| State | Contract |
| --- | --- |
| Accounts, homes, `/var/lib/soda/accounts/` | Preserve identities, UID/GID/groups, checkouts and dirty work |
| SSH host keys and managed authorized_keys | Preserve host identity; change keys only through explicit actions |
| `/srv/project/shared`, `~/shared` | Real shared files with setgid directory semantics |
| `/opt/mise`, `/etc/mise`, installed packages | Real persistent installations, not merely a download cache |
| Nested container storage and volumes | Persistent nested images, layers, volumes and secrets |
| `/run`, PTYs, sockets, processes | Runtime only; startup recreates needed runtime directories |

Lifecycle verbs:

- **Create** reserves and starts the shared container once; it does not Join or clone.
- **Join** provisions the real account and records membership only after success.
- **Open / refresh / navigation / Hide** never create, join, start, repair or update.
- **End terminal** ends that managed terminal's supervised processes only.
- **Stop / Start** stop or start the same existing container; they never recreate it.

At boot, `project-init` reasserts managed directory modes, prepares bounded network
sysctl mounts and generates only missing SSH host keys before marking readiness.
The readiness marker is not proof of SSH reachability, account provisioning or
application health.

## Access and credentials

- Project access is ordinary `user@project-ip` plus native service ports.
- Managed-key writes are root-owned authorized_keys updates through explicit API/helper
  actions; never request private keys.
- Host Tailnet enrollment does not by itself provide project reachability.
  See [Networking](../architecture/networking.md).

## Same-root maintenance

Required package, helper or config additions apply to the **same** retained project
root. Maintenance must declare package/service effects and preserve state. Image
replacement, fleet updaters and silent installation on terminal Open are out of
scope ([Product scope](../product/scope.md)).

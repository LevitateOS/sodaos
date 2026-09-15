# Projects

A **project** is a lasting shared development environment bound to a Forgejo
repository. It is not a disposable per-user Codespace and not a host Linux account.

Canonical runtime contracts live in [Project OS](../reference/project-os.md).
This document owns the product model: what a project is, who may create and join it,
how profiles work, and what must persist.

## Creation and ownership

- A repository's human owner may create its project environment.
- Organization-owned creation is unsupported.
- Environment and member visibility resolve current native ownership by stable
  repository ID (including organization `is_owner`) or explicit Soda operator
  authority. A stored creator is not permanent authorization.
- Creation chooses an immutable **Project OS profile**. Existing environments keep
  their original profile; there is no live distro/desktop switcher.

## Selected profiles

Every profile shares the same Project OS foundation: native accounts, shared tools,
persistent roots, access and maintenance. Distribution and desktop choices extend
that foundation; they do not create a second product.

| Profile | Distribution | Interface |
| --- | --- | --- |
| Rocky headless | Rocky Linux | Terminal (default) |
| Rocky KDE | Rocky Linux | Terminal and KDE Plasma |
| Fedora Server | Fedora Linux | Terminal / headless |
| Fedora KDE | Fedora Linux | Terminal and KDE Plasma |

GNOME profiles are deferred. Mise is required for every supported Project OS
profile and for the separate Runner OS job image.

Profiles describe initial userspace and interface, not independent backends.
Server-side code resolves a bounded profile ID to installed, architecture-compatible
artifacts. The browser must not supply arbitrary image references, package lists or
runtime flags.

## Explicit joining

**Add me to this project** creates the intended project-local Linux account, installs
any explicitly selected external-SSH public keys, and records membership only after
confirmed native success.

- The creator must Join as well.
- Account-only Join without external SSH keys follows the Project OS access contract.
- A database row, mock or manual checklist is not Join.
- Never request a private SSH key.
- Joining is separate from native Git authorization.
- Stable Forgejo identity binds membership to its original Linux login. Native rename
  or transfer does not silently remap Linux users, ownership or installed keys.

Developers have Linux accounts **inside projects**, not human host accounts or homes.

## Persistence

Normal startup starts the **existing** container. Preserve accounts, homes, SSH host
keys, installed packages and tools, shared files, configuration, dirty work and
service volumes across stop/start and host reboot.

Never use `--rm`, `--replace`, pruning or deletion as repair.

Projects are a trusted-team namespaced boundary, not hostile-tenant isolation.

## Access

Project access is ordinary `user@project-ip`, SSH/PTY/SCP/SFTP and native service
ports. Clients need a real route; host Tailnet enrollment alone does not provide it.

Managed browser terminals use the member's existing project-local account and home.
Opening a terminal must not create, join or start a project, and must not expose
host root. See [Terminal](../reference/terminal.md).

## Related guides

- Everyday use: [Develop in a project](../guides/develop.md)
- Nested workloads: [Project services](../guides/project-services.md)
- Packaged CLIs: [Project CLIs](../guides/project-clis.md)
- Spaces entry: [Spaces](spaces.md)

# SodaOS

Persistent, shared development environments on an immutable appliance host. Developers use a browser, ordinary SSH, Git, mise and container tools—not individual Linux accounts on the host.

**Current handoff: source-complete, unbuilt, unvalidated (M01–M14).** This is not a tested appliance or a ready-to-deploy release. Native builds and validation remain held for explicitly authorized targets. See [implementation status](docs/implementation-status.md).

## How it fits together

```text
CoreOS host — operator administration only
├── Native systemd services
│   ├── Cockpit + Soda Tailnet/Runners pages
│   ├── tailscaled
│   ├── Restricted Soda project helper
│   └── Local CI runners
└── Podman
    ├── Soda dashboard — Go + HTMX, persistent SQLite database
    ├── Forgejo — identity and Git, persistent data and its own SQLite database
    ├── Caddy — private HTTPS endpoints
    └── Persistent project environments — Rocky Linux + mise
        ├── Project-local Linux accounts, homes and SSH
        ├── Shared files and installed tools
        └── Project-local services through nested Podman
```

The implemented host candidate is **Fedora CoreOS**, with native packages requested through **rpm-ostree layering**. Tailscale runs on that host as `tailscaled.service`; it is not embedded in Forgejo or the projects.

**Forgejo is currently a standalone Podman container, not a Podman pod.** Its Quadlet is [`appliance/services/forgejo.container`](appliance/services/forgejo.container). The dashboard and HTTPS proxy also have separate container definitions.

These are source choices, not native compatibility evidence. Nested Podman, package layering and real client routing still need validation.

## Developer workflow

1. The operator establishes Forgejo and Soda administration, then creates people through Soda.
2. Developers sign in through Forgejo and register public SSH keys in their Soda profiles. Private keys stay on their clients.
3. A repository's human owner creates its project environment. That owner administers the project, not the host.
4. Each person explicitly selects **Add me to this project**, including the creator.
5. Developers connect directly to the project's IP, for example `ssh alice@192.168.1.101`, and use ordinary SSH commands, SCP and SFTP.
6. Personal checkouts coexist with shared files, actual shared mise installations and project services. Git credentials remain separately managed by Forgejo.

Projects retain their writable userspace across normal stop/start. Replacing or deleting a project container is not normal startup or maintenance.

## Private connectivity and operator tools

Cockpit is loopback-first and permits only the native root/operator identity. Its retained Tailnet page controls host Tailscale; Runners manages local Forgejo/GitHub execution capacity. Providers still own CI workflows, scheduling and results.

Browser services and Forgejo Git SSH bind to an explicitly selected private appliance address. For Tailnet access, that listener must accept the host's Tailscale IP. Forgejo's SSH advertisement refresh checks this rather than inventing a reachable endpoint; browser/OAuth origins remain configured separately.

Project IPs use a separate routed bridge subnet. Developer clients need a LAN route through the appliance or an advertised and approved Tailscale subnet route. **Enrolling the host in Tailscale does not automatically make project IPs reachable.** No project DNS or project SSH gateway is used.

## Repository map

| Path | Purpose |
| --- | --- |
| `cmd/`, `internal/` | Go dashboard, database, provider clients and native helpers |
| `cockpit/` | Retained TypeScript/React Tailnet and Runners pages and tests |
| `project-os/` | Rocky project image, accounts, SSH, shared tools and workload configuration |
| `appliance/` | Native services, Quadlets, configuration and activation source |
| `scripts/` | Explicit later build, staging, provisioning, installation and check entrypoints |
| `tests/` | Authored staging checks and opt-in installed journeys/fixtures |
| `assets/` | Canonical branding and attribution |
| `docs/` | Architecture, scope, implementation handoff and operator guides |

## Read next

- [Architecture](docs/architecture.md) and [deferred scope](docs/deferred.md)
- [Implementation plan](docs/implementation-plan.md) and [current handoff](docs/implementation-status.md)
- [Installation](docs/installation.md) and [operator bootstrap](docs/operator-setup.md)
- [Development environment](docs/development-environment.md), [project services](docs/project-services.md) and [provider CLIs](docs/project-clis.md)
- [Operator welcome](docs/console-welcome.md), [branding review](docs/branding-review.md) and [screenshot guidance](docs/screenshot-capture.md)
- [Predecessor reuse inventory](docs/predecessor-reuse.md)
- [Later native validation](docs/native-validation.md)
- [Coding-agent instructions](AGENTS.md)

## Development and validation

The current phase permits source changes and authored tests/recipes, **not their execution**. Do not run builds, compilation, type checks, tests, dependency installation, provisioning or live registration without later explicit authorization.

The implementation targets native x86_64 and aarch64 independently. Evidence on one is not evidence on the other, and an unavailable sibling does not block useful authorized work.

Selected native/Cockpit code is reused from the predecessor repository, with attribution in the handoff and retained license files. Vendored HTMX and its license are under `internal/web/static/`.

# SodaOS

Persistent, shared development environments on an immutable appliance host. Developers use a browser, ordinary SSH, Git, mise and container tools—not individual Linux accounts on the host.

**The local Soda dashboard is running, with Forgejo OAuth sign-in verified in a real browser.** Open **https://localhost:24443** using the tunnels and private operator credentials in [local dashboard access](docs/local-testing.md). Recorded native x86_64 build/source checks passed, but the merged tree, including branding, console and project-CLI follow-ups, has not been rebuilt or retested. Full developer/project journeys remain unvalidated. This is not a ready-to-deploy release. See [implementation status](docs/implementation-status.md) for evidence and remaining work.

**Next dashboard direction:** a client-rendered React/PatternFly frontend over upstream Forgejo and Soda's Go environment/access extension. See the [full milestone plan](docs/dashboard-implementation-plan.md) and [page/dependency inventory](docs/dashboard-plan.md). Implementation has started with a React preview, JSON session/profile/key APIs and migration source; it has not been built or deployed. See [current progress](docs/implementation-status.md#core-implementation-started). The topology below describes the existing installed system.

**Plan ownership:** the dashboard implementation plan leads the core—including production native environments and their acceptance. The [native support porting plan](docs/native-porting-plan.md) covers outside VM/SSH/evidence/artifact helpers, provisioning and host-operator integrations. It follows the core's contracts; optional media is not a core prerequisite. The active [support tool source and recipes](docs/native-support.md) are now authored, with tests not yet executed; no new native readiness is claimed.

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

Native package layering and initial services have been exercised on the isolated x86_64 host. Nested Podman, real client routing and the full product journey still need validation.

## Developer workflow

1. The operator establishes Forgejo and Soda administration, then creates people through Soda.
2. Developers sign in through Forgejo and register public SSH keys in their Soda profiles. Private keys stay on their clients.
3. A repository's human owner selects it from the Projects repository picker and creates its environment. Repositories with an existing environment are excluded. That owner administers the project, not the host.
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
| `dashboard/` | New client-rendered React preview and tests; source in progress, not deployed |
| `cockpit/` | Retained TypeScript/React Tailnet and Runners pages and tests |
| `project-os/` | Rocky project image, accounts, SSH, shared tools and workload configuration |
| `appliance/` | Native services, Quadlets, configuration and activation source |
| `scripts/` | Explicit build, staging, provisioning, installation, check and local test-VM entrypoints |
| `tools/soda-{artifacts,acceptance}/` | Outside artifact/VM/transport/evidence tools, never installed as appliance commands |
| `tests/` | Authored staging checks and opt-in installed journeys/fixtures |
| `assets/` | Canonical branding and attribution |
| `docs/` | Architecture, scope, implementation handoff and operator guides |

## Read next

- [Architecture](docs/architecture.md) and [deferred scope](docs/deferred.md)
- [Current frontend/backend milestone plan](docs/dashboard-implementation-plan.md), [page/dependency inventory](docs/dashboard-plan.md) and [current handoff](docs/implementation-status.md)
- [Historical initial M01–M18 plan](docs/implementation-plan.md)
- [Installation](docs/installation.md) and [operator bootstrap](docs/operator-setup.md)
- [Development environment](docs/development-environment.md), [project services](docs/project-services.md) and [provider CLIs](docs/project-clis.md)
- [Operator welcome](docs/console-welcome.md), [branding review](docs/branding-review.md) and [screenshot guidance](docs/screenshot-capture.md)
- [Predecessor reuse inventory](docs/predecessor-reuse.md) and [subordinate native support porting plan](docs/native-porting-plan.md)
- [Local test host access](docs/local-testing.md) and [native validation](docs/native-validation.md)
- [Coding-agent instructions](AGENTS.md)

## Development and validation

Native execution has started on the x86_64 builder and its isolated `soda-test` VM. Installation on other targets, host networking changes, provider enrollment and destructive lifecycle checks still require explicit scope and permission. Keep source-check, host-boot and end-to-end evidence separate; do not report unexecuted checks as passed.

The implementation targets native x86_64 and aarch64 independently. Evidence on one is not evidence on the other, and an unavailable sibling does not block useful authorized work.

Selected native/Cockpit code is reused from the predecessor repository, with attribution in the handoff and retained license files. Vendored HTMX and its license are under `internal/web/static/`.

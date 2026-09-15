# SodaOS

Persistent, shared development environments on an operator-managed Fedora CoreOS
appliance. Developers use native Forgejo, SSH, Git, mise and container tools—not
individual Linux accounts on the host.

Soda integrates Forgejo's native frontend with **Spaces** (repository environment
entry and drawer), a protected Go environment/access API, OAuth, native
provisioning, Tailnet/Runners operator settings and stock branded Cockpit.

Documentation index: [docs/README.md](docs/README.md).

## System

```text
Fedora CoreOS host — operator administration only
├── Stock branded Cockpit, tailscaled, trusted project Tailnet companions and CI runners
└── Podman
    ├── Stock Forgejo — native frontend, identity/Git and its own persistent data
    ├── Soda Go API/OAuth service — separate SQLite/grants
    ├── Caddy — configured private HTTPS endpoints
    └── Persistent Project OS containers
        ├── Project-local accounts, homes, SSH and personal checkouts
        ├── Actual shared installed tools/files
        └── Native nested workloads and persistent service data
```

Forgejo, Soda and Caddy are separate containers; **Forgejo is not a Podman pod**.
See [architecture](docs/architecture/overview.md) and `appliance/services/` for
ownership and placement. The repository's human owner administers its project, not
the host. Each user explicitly joins. Native Git authorization is separate. Normal
startup preserves the existing container/root; see
[Project OS](docs/reference/project-os.md).

Project access is ordinary `user@project-ip`, SCP/SFTP and native service ports.
Clients need a real route; host Tailnet enrollment alone does not provide it.
Cockpit is loopback-first/root-only. Browser origins, Git advertisement and project
routing are separate configuration.

## Source and guides

| Area | Source / documentation |
| --- | --- |
| Product and architecture | [Overview](docs/product/overview.md), [Architecture](docs/architecture/overview.md), [Scope](docs/product/scope.md) |
| Host capability strategy | [Host strategy](docs/research/host-strategy.md) |
| API / auth / Forgejo customization | `cmd/`, `internal/`, [API](docs/reference/api.md), [Credentials](docs/reference/credentials.md), [Forgejo](docs/reference/forgejo.md) |
| Installation / operator access | `appliance/`, `scripts/`, [Installation](docs/guides/installation.md), [Operator setup](docs/guides/operator-setup.md), [Media](docs/guides/media.md) |
| Project environments | `project-os/`, [Project OS](docs/reference/project-os.md), [Develop](docs/guides/develop.md), [Services](docs/guides/project-services.md), [CLIs](docs/guides/project-clis.md) |
| Cockpit, runners, support tools | `assets/branding/cockpit/`, `tools/`, [Cockpit](docs/development/cockpit.md), [Runners](docs/reference/runners.md), [Native support](docs/development/native-support.md) |
| Branding and reuse | `assets/`, [Branding](docs/design/branding.md), [Attribution](docs/research/predecessor-reuse.md), [Console](docs/design/console-welcome.md), [Screenshots](docs/design/screenshot-capture.md) |
| Public handbook | [Handbook](docs/public/10-Start-here/10-index.md), [Authoring](docs/public/README.md) |
| Release | [Release architecture](docs/architecture/release.md), [Release workflow](docs/development/release.md) |
| Development | [Development](docs/development/README.md), [Go](docs/development/go.md), [Local testing](docs/guides/local-testing.md), [AGENTS.md](AGENTS.md) |

Native x86_64 and aarch64 are independent targets. Builds, installation, provider
mutations and destructive cleanup need applicable approval. Preserve credentials,
project state, backups and failed evidence.

## License

Original SodaOS code, documentation and configuration are licensed under
[Apache-2.0](LICENSE). [NOTICE](NOTICE) defines attribution and scope. This does not
relicense inherited code, canonical artwork or third-party material; their existing
terms apply, and a missing inherited license is not an Apache grant.

Preserve Forgejo and other dependency licenses. Actual-artifact corresponding-source,
notice/font delivery and inherited-rights clearance remain required under
[licensing](docs/research/licensing.md), not satisfied by adding the original-code license.

# SodaOS

**The operating system for software factories.** Run your coding agents on your
infrastructure, with controlled access and reviewable results.

A person authorizes work in Forgejo. Soda gives a coding agent a fresh workspace
with time and resource limits. Results return as a pull request, checked by CI
and a fresh review against the exact candidate commit. One bounded repair receives
new verification; a person decides what merges. Every run ends with a recorded
outcome and separately tracked cleanup.

Persistent shared Projects support human development, debugging and intervention
through native Forgejo, SSH, Git, mise and container tools. Developers do not
receive individual Linux accounts on the host.

Soda integrates Forgejo's native frontend with **Spaces** (repository environment
entry and drawer), a protected Go environment/access API, OAuth, native
provisioning, Tailnet/Runners operator settings and stock branded Cockpit. The
`soda-factory` operator command owns explicit admission and bounded execution.

Start with the [first factory task](docs/public/30-Use-Soda/15-software-factory.md).
The [operator reference](docs/reference/factory.md) describes configuration and
commands; [manual development](docs/guides/develop.md) is a separate path.

Documentation index: [docs/README.md](docs/README.md).

## System

```text
Fedora CoreOS host — operator administration only
├── Stock branded Cockpit, tailscaled, trusted project Tailnet companions and CI runners
├── soda-factory — unprivileged operator command, execution ledger and publisher
│   └── Rootless Podman — disposable agent workspaces and restricted networking
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
See [architecture](docs/architecture/overview.md) and `system/host/services/` for
ownership and placement. The repository's human owner administers its project, not
the host. Each user explicitly joins. Native Git authorization is separate. Normal
startup preserves the existing container/root; see
[Project OS](docs/reference/project-os.md).

Project access is ordinary `user@project-ip`, SCP/SFTP and native service ports.
Clients need a real route; host Tailnet enrollment alone does not provide it.
Cockpit listens on all interfaces/root-only. Browser origins, Git advertisement and project
routing are separate configuration.

## Source and guides

| Area | Source / documentation |
| --- | --- |
| Product and architecture | [Overview](docs/product/overview.md), [Architecture](docs/architecture/overview.md), [Scope](docs/product/scope.md) |
| Bounded software work | [Factory command](cmd/soda-factory/README.md), `internal/factory/`, [Operator reference](docs/reference/factory.md), [First task](docs/public/30-Use-Soda/15-software-factory.md) |
| Host capability strategy | [Host strategy](docs/research/host-strategy.md) |
| API / auth / Forgejo customization | `cmd/`, `internal/`, [API](docs/reference/api.md), [Credentials](docs/reference/credentials.md), [Forgejo](docs/reference/forgejo.md) |
| Installation / operator access | `system/`, `scripts/`, [Installation](docs/guides/installation.md), [Operator setup](docs/guides/operator-setup.md), [Media](docs/guides/media.md) |
| Project environments | `system/project/`, [Project OS](docs/reference/project-os.md), [Develop](docs/guides/develop.md), [Services](docs/guides/project-services.md), [CLIs](docs/guides/project-clis.md) |
| Cockpit and support tools | `assets/branding/cockpit/`, `tools/`, [Cockpit](docs/development/cockpit.md), [Native support](docs/development/native-support.md) |
| Branding and reuse | `assets/`, [Branding](docs/design/branding.md), [Attribution](docs/research/predecessor-reuse.md), [Console](docs/design/console-welcome.md), [Screenshots](docs/design/screenshot-capture.md) |
| Public handbook | [Handbook](docs/public/10-Start-here/10-index.md), [Authoring](docs/public/README.md) |
| Release | [Release architecture](docs/architecture/release.md), [Release workflow](docs/development/release.md) |
| Development | [Development](docs/development/README.md), [Go](docs/development/go.md), [Local testing](docs/guides/local-testing.md), [AGENTS.md](AGENTS.md) |

Supported platform scope belongs to [Release architecture](docs/architecture/release.md#architectures).
Builds, installation, provider mutations and destructive cleanup need applicable
approval. Preserve credentials, project state, backups and failed evidence.

## Rust package guides

Every Cargo package has a local guide to its purpose, prerequisites and use.
Start with the installed commands for appliance or project work. Library guides
are for Rust consumers; build and fixture tools are for release developers and
operators. Run their Cargo examples from this repository's root with the
toolchain selected by [rust-toolchain.toml](rust-toolchain.toml).

| Package guide | Use |
| --- | --- |
| [Activation](cmd/soda-activate/README.md) | Activate private browser and Git access. |
| [Console welcome](cmd/soda-console-welcome/README.md) | Read appliance network and access guidance. |
| [Factory command](cmd/soda-factory/README.md) | Inspect and retire supervised factory runs. |
| [Forgejo domain](cmd/soda-forgejo-domain/README.md) | Control the Forgejo writer domain during recovery. |
| [Forgejo migration](cmd/soda-forgejo-migrate/README.md) | Prepare database-password configuration at startup. |
| [Identity Compose](cmd/soda-identity-compose/README.md) | Give a selected nested service Muse access. |
| [Identity broker](cmd/soda-identity/README.md) | Operate the private provider-credential broker. |
| [Image import](cmd/soda-image-import/README.md) | Import bundled images into host Podman storage. |
| [Installer](cmd/soda-install/README.md) | Install to disk and continue appliance setup. |
| [Muse launcher](cmd/soda-muse/README.md) | Run Muse from a provisioned project account. |
| [Muse maintenance](cmd/soda-muse-maintain/README.md) | Restore packaged Muse tools in a retained project. |
| [PostgreSQL maintenance](cmd/soda-pg-maintenance/README.md) | Back up, restore and provision appliance databases. |
| [Project terminal helpers](cmd/soda-project-terminal/README.md) | Use managed terminals and understand their service helpers. |
| [Operator setup](cmd/soda-setup/README.md) | Bootstrap the native operator binding and configuration. |
| [Host service](lib/host/README.md) | Operate project provisioning and privileged host capabilities. |
| [Release inputs](lib/release-inputs/README.md) | Validate release metadata from Rust. |
| [Release build library](lib/soda-release-build/README.md) | Fetch inputs and run native build primitives. |
| [Release delivery library](lib/soda-release-deliver/README.md) | Validate, sign and publish with protected release authority. |
| [Release image library](lib/soda-release-image/README.md) | Assemble host content, candidates and installation media. |
| [Release commands](lib/soda-release-tools/README.md) | Produce candidates and media and inspect artifacts. |
| [Unix HTTP library](lib/unix-http/README.md) | Make bounded HTTP requests over Unix sockets. |
| [Wire time library](lib/wire-time/README.md) | Parse and format shared protocol timestamps. |
| [Acceptance support](tools/acceptance/README.md) | Run bounded native checks and collect evidence. |
| [Candidate setup](tools/candidate-setup/README.md) | Prepare an isolated development builder. |
| [Lab credentials](tools/lab-credentials/README.md) | Inspect credentials and select a rotation runbook. |
| [PostgreSQL fixture](tools/pg-fixture/README.md) | Start a disposable local test database. |
| [Release assets](tools/release-assets/README.md) | Fetch and stage assets and render provisioning. |
| [Test VM](tools/test-vm/README.md) | Start and access a prepared development VM. |

## Go package guides

Command guides describe installed or development-tool use. Internal package
guides explain their user-facing feature and the integration surface for Go
consumers in this module. The test packages document focused checks and their
prerequisites. Toolchain and dependency pins, including the local Forgejo SDK
replacement, live in [go.mod](go.mod).

| Package guide | Use |
| --- | --- |
| [Dashboard](cmd/soda-dashboard/README.md) | Operate the Soda backend and factory coordinator. |
| [Forgejo extension](cmd/soda-extension/README.md) | Run the native Forgejo extension bridge. |
| [Tailnet command](cmd/soda-tailnet/README.md) | Inspect Tailnet status and operator guidance. |
| [Installed acceptance](internal/acceptance/README.md) | Invoke scoped native client probes. |
| [Architecture checks](internal/archcheck/README.md) | Check Go package dependency boundaries. |
| [Avatar renderer](internal/avatar/README.md) | Render deterministic robot SVGs. |
| [Configuration](internal/config/README.md) | Load and validate dashboard/operator configuration. |
| [Factory records](internal/factory/README.md) | Use canonical factory records and validation. |
| [Factory coordinator](internal/factory/control/README.md) | Coordinate readiness, dispatch and supervised run lifecycles. |
| [File locks](internal/filelock/README.md) | Hold advisory locks around cooperating processes. |
| [Forgejo client](internal/forgejo/README.md) | Observe and act through native Forgejo APIs. |
| [Factory publication](internal/forgejo/publish/README.md) | Validate candidates and perform conditional Git/PR publication. |
| [Host client](internal/host/README.md) | Call the privileged Rust host daemon over Unix sockets. |
| [Identity records](internal/identity/README.md) | Use canonical provider connection, grant and execution records. |
| [Identity client](internal/identity/client/README.md) | Call the private identity broker. |
| [Project records](internal/project/README.md) | Use project identity, profile and preparation contracts. |
| [Store](internal/store/README.md) | Persist application records through the PostgreSQL store. |
| [Strict JSON](internal/strictjson/README.md) | Decode one bounded JSON object. |
| [Tailnet integration](internal/tailnet/README.md) | Read native status and validate companion wire inputs. |
| [Web server](internal/web/README.md) | Wire the backend mux, authentication and API. |
| [Web API](internal/web/api/README.md) | Integrate product HTTP and terminal routes. |
| [Web authentication](internal/web/auth/README.md) | Admit native extension identities and account operations. |
| [Source and presentation tests](scripts/README.md) | Check templates, branding, wiring and wire contracts. |
| [Build tests](tests/build/README.md) | Check packaging and native helper interfaces. |
| [PNG comparison](tools/png-equal/README.md) | Compare decoded image pixels. |
| [Avatar catalog](tools/soda-avatars/README.md) | Generate local artwork inspection sheets. |
| [Installed probe command](tools/soda-installed-probes/README.md) | Observe HTTPS, developer access, Git and workloads. |
| [Rootfs server](tools/soda-rootfs-server/README.md) | Serve public installer rootfs files to fixture VMs. |

## License

Original SodaOS code, documentation and configuration are licensed under
[Apache-2.0](LICENSE). [NOTICE](NOTICE) defines attribution and scope. This does not
relicense inherited code, canonical artwork or third-party material; their existing
terms apply, and a missing inherited license is not an Apache grant.

Preserve Forgejo and other dependency licenses. Actual-artifact corresponding-source,
notice/font delivery and inherited-rights clearance remain required under
[licensing](docs/research/licensing.md), not satisfied by adding the original-code license.

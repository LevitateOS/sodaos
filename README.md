# SodaOS

Persistent, shared development environments on an operator-managed Fedora CoreOS
appliance. Developers use native Forgejo, SSH, Git, mise and container tools—not
individual Linux accounts on the host.

**Current direction:** Forgejo's native frontend with a **Sodaspaces** repository
button/environment drawer, without adding a new tab. Both standalone Soda frontends (React and Go/HTMX) are
removed. The protected Go environment/access API, OAuth, native provisioning and
separate Cockpit Tailnet/Runners pages remain. **Sodaspaces is not implemented yet.**

The isolated `soda-test` guest still has historical `8b823db` React preview/HTMX
bytes; source cleanup has not been deployed. Bounded native x86_64 first-product
proof is accepted, not final product, fresh-install or aarch64 acceptance. See
[current work](docs/sodaspaces-plan.md), [handoff/evidence](docs/implementation-status.md)
and [local access](docs/local-testing.md). This is not a ready-to-deploy release.

## System

```text
Fedora CoreOS host — operator administration only
├── Native Cockpit + Tailnet/Runners, tailscaled, project helper and CI runners
└── Podman
    ├── Stock Forgejo — native frontend, identity/Git and its own persistent data
    ├── Soda Go API/OAuth service — separate SQLite/grants
    ├── Caddy — configured private HTTPS endpoints
    └── Persistent Rocky + mise projects
        ├── Project-local accounts, homes, SSH and personal checkouts
        ├── Actual shared installed tools/files
        └── Native nested workloads and persistent service data
```

Forgejo, Soda and Caddy are separate containers; **Forgejo is not a Podman pod**.
See [architecture](docs/architecture.md) and `appliance/services/` for ownership and
placement. The repository's human owner administers its project, not the host.
Each user explicitly joins with a public development key; native Git authorization
is separate. Normal startup preserves the existing container/root, not replacement.

Project access is ordinary `user@project-ip`, SCP/SFTP and native service ports.
Clients need a real route; host Tailnet enrollment alone does not provide it.
Cockpit is loopback-first/root-only. Browser origins, Git advertisement and project
routing are separate configuration—not inferred from an old port or hostname.

## Source and guides

| Area | Source / documentation |
| --- | --- |
| Product decisions and remaining work | [Architecture](docs/architecture.md), [Sodaspaces plan](docs/sodaspaces-plan.md), [deferred scope](docs/deferred.md) |
| API/auth/native integration | `cmd/`, `internal/`, [API](docs/dashboard-api.md), [credentials](docs/dashboard-credentials.md), [Forgejo customization](docs/forgejo-frontend-integration.md) |
| Installation/operator access | `appliance/`, `scripts/`, [installation](docs/installation.md), [bootstrap](docs/operator-setup.md), [native validation](docs/native-validation.md) |
| Project environments | `project-os/`, [development guide](docs/development-environment.md), [services](docs/project-services.md), [CLIs](docs/project-clis.md) |
| Cockpit and outside tooling | `cockpit/`, `tools/`, [Cockpit](docs/cockpit-port.md), [runners](docs/runners-port.md), [native support](docs/native-support.md) |
| Branding and reuse | `assets/`, [attribution](docs/predecessor-reuse.md), [console](docs/console-welcome.md), [branding review](docs/branding-review.md), [capture rules](docs/screenshot-capture.md) |
| Public handbook | [Release-day handbook](docs/public/10-Start-here/10-index.md), [authoring/sync](docs/public/README.md), [editorial review](docs/public-docs-review.md); intended product documentation, not current acceptance |
| Evidence and coding guidance | [Current handoff](docs/implementation-status.md), [local test host](docs/local-testing.md), [AGENTS.md](AGENTS.md) |

Native x86_64 and aarch64 are independent targets. Builds/tests, installation,
restart, routing, provider mutations and destructive cleanup need their applicable
scope; neither this README nor a tool flag grants it. Preserve credentials, project
state, backups and failed evidence. Historical plans/audits remain in Git, not active
roadmaps; the handoff explains where to find them.

## License

Original SodaOS code, documentation and configuration are licensed under
[Apache-2.0](LICENSE). [NOTICE](NOTICE) defines attribution and scope. This does not
relicense inherited code, canonical artwork or third-party material; their existing
terms apply, and a missing inherited license is not an Apache grant.

Preserve Forgejo and other dependency licenses. Actual-artifact corresponding-source,
notice/font delivery and inherited-rights clearance remain required under
[licensing](docs/licensing.md), not satisfied by adding the original-code license.

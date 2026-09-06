# SodaOS

Persistent shared Rocky + mise development environments on an immutable appliance host.

See [architecture](docs/architecture.md), [implementation plan](docs/implementation-plan.md), [source handoff](docs/implementation-status.md), and [deferred scope](docs/deferred.md).

## Development phase

Source implementation only. Do not run compilation, builds, type checks or tests until the later native execution phase is authorized. Author test/build/provisioning sources with their features; do not report unexecuted checks as passed.

The new dashboard/backend use Go + HTMX. Tailnet and Runners retain their predecessor Cockpit technology. Credentials belong in restricted runtime files, never source or command traces.

Third-party HTMX source and its license are under `internal/web/static/`. Selected Go/native and Cockpit code is reused from the user's predecessor repository, attributed in the handoff.

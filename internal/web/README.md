# Forge and Factory HTTP applications

This package constructs two products from shared domain, API and storage
packages. Their process lifecycle and private listeners live in `web/serve`.

- `NewForge(config.Config, *store.Store)` constructs Project, terminal,
  application, account and extension administration. It never creates a Factory
  coordinator, claims its ledger lock or starts worker MCP.
- `NewFactory(config.Config, *store.Store)` supplies native runtime/collaboration
  ports to `factory/service.NewApplication` and constructs intake and Factory API.
  The shared application owns coordinator/background assembly and explicit
  startup/admission-stop/drain lifetime. Its private extension handler
  refuses unrelated Project, account and terminal routes.

The Fountain extension sends Factory routes directly to its configured Factory
socket, preserving native authority for verification at the destination.
Forge refuses Factory browser routes. `factory_control_socket` supplies the narrow
Project stop/start and authority interface. Human terminal ownership stays in
Forge. Missing Factory connections never create a local coordinator or retry a
mutation. Forge health does not depend on Factory availability.

The products share the existing PostgreSQL schema and canonical Project records
in this repository. This is a process and product-lifecycle separation, not
independent database ownership or support for another forge/execution backend.
Factory retains the configured publication root and worker IPC path so existing
Project mounts and retained runs keep their identities.

Factory startup binds worker MCP before `StartCoordinator`, takes exclusive
ledger ownership and reconciles before admitting work. Shutdown closes admission,
drains HTTP and worker requests, then closes the shared application before Store.
The application delegates to the existing coordinator gate/ledger lock. Forge drains
its own HTTP listeners and terminal peers without touching Factory.

`web.Extension(forgeSocket, factorySocket)` remains the native Fountain integration.
Its frontend adapter selects each product's private extension listener using the
canonical Factory route boundary. Native authority, same-origin checks and
capability-bound worker admission remain required across the product boundary.

Operator commands: [Factory service](../../cmd/soda-factory-server/README.md),
[Forge backend](../../cmd/soda-dashboard/README.md). Route contracts:
[HTTP API](../../docs/reference/api.md). Configuration:
[Service configuration](../../docs/reference/configuration.md).

# api

Serve Soda's product HTTP and WebSocket operations through verified native
Forgejo extension authority. This internal Go package is a backend integration
library, consumed by [`web`](../README.md); it has no standalone application.

## Integrate

`api.New(cfg, db, forgejoClient, hostClient, authService)` constructs an `API`.
Supply admitted configuration, an open store and the existing service clients.
The facade then attaches the identity broker client and factory coordinator.
Use `API.ExtensionHandler()` on the dedicated private extension listener.

`api.Extension(socket)` constructs the extension-side HTTP bridge and returns a
handler plus a close function for its idle transport. The socket must be the
private Soda listener. The bridge admits a fixed route set, checks native
authority, bounds ordinary messages, and strips cookies and authority headers
from responses. WebSocket streams retain their separate framing contract.

## Available operations

The handlers provide:

- Repository selection, creation profiles, shared project creation and Join.
- Project membership, access keys, OS state, lifecycle and preparation controls.
- Spaces inventory, managed terminals and native WebSocket attachment.
- Provider enrollment, connection grants, leases and authorized identity launch.
- Factory policy/grants, acceptance, lifecycle actions, recorded state and
  read-only run output.
- Operator Tailnet settings and authorized project Tailnet operations.

The exact methods, routes and request contracts belong to
[HTTP API](../../../docs/reference/api.md). Browser clients use Forgejo's native
extension API base and current mount context. The private `/api/...` paths are
service routes rather than independent public browser endpoints.

## Effects and constraints

Requests can persist database records and invoke host, broker or native Forgejo
operations. Protected calls verify the live actor, contribution, repository
authority and session generation; mutations also check same-origin context.
Operator actions require the configured stable Soda operator ID. Displayed IDs
and browser-supplied labels do not establish authority.

The API maintains bounded terminal peer state. Call `CloseTerminals()` while
draining the server to cancel and join those peers. Native lifecycle and
credential retirement remain owned by the host and identity broker.

See [Projects](../../../docs/product/projects.md),
[Terminal](../../../docs/reference/terminal.md),
[Credentials](../../../docs/reference/credentials.md) and
[Factory](../../../docs/reference/factory.md) for the behavior these handlers
expose. [Trust](../../../docs/architecture/trust.md) owns their authority limits.

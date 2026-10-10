# web

Wire Soda's dashboard HTTP facade, native extension bridge and factory
coordinator. This internal Go package is for backend integration consumers;
it has no executable. Operators run
[`soda-dashboard`](../../cmd/soda-dashboard/README.md), and people use native
Forgejo contributions.

## Construct the backend

Call `web.New(config.Config, *store.Store)` with admitted configuration and an
already opened encrypted store. It constructs the Forgejo and host clients,
[`auth`](auth/README.md), [`api`](api/README.md), identity client and factory
coordinator. Optional configured background Forgejo services supply readiness,
publication, review, merge and check adapters.

The serving process owns this lifecycle:

1. Call `StartCoordinator(ctx)` to acquire exclusive ledger ownership and settle
   outstanding work before accepting requests.
2. Serve the `Server` as an `http.Handler`, using coordinator HTTP admission as
   shown by the dashboard entry point.
3. Mount `ExtensionHandler()` on the dedicated private extension Unix listener.
4. Stop admission and drain HTTP, call `CloseTerminals()`, then
   `CloseCoordinator()` during shutdown.

The configured publication root must be writable for the coordinator lock.
Startup and coordinator reconciliation can write state and invoke retirement.
`SetForgejo` and `SetHost` replace facade clients and the corresponding API
dependencies for integration use.

## HTTP and extension boundaries

The facade serves the `/healthz` probe, redirects `/` to the configured Forgejo
origin, and serves the public namespaced avatar route. It enforces canonical
Soda paths and rejects encoded aliases.
Product routes are supplied by the private extension handler. Authenticated
webhook intake is wired with its separately configured secret; an absent or
unreadable secret disables admission.

`web.Extension(socket)` constructs the SDK application used by
[`soda-extension`](../../cmd/soda-extension/README.md), including the supported
Forgejo username policy, and returns its transport close function. That socket
must target the dedicated private backend listener. Native authority is checked
at both ends of the bridge.

Handler effects and callable routes belong to [HTTP API](../../docs/reference/api.md),
factory lifecycle to [Factory](../../docs/reference/factory.md), and native
authority to [Trust](../../docs/architecture/trust.md). Configuration belongs to
[Service configuration](../../docs/reference/configuration.md).

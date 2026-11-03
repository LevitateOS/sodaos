# soda-extension

Connect Fountain's native extension runtime to Forge and Factory. One
compiled backend serves two explicitly selected native packages:

- Required `soda` provides Spaces, the Workspace panel, username policy and the
  authenticated lifecycle controller for optional Tailnet.
- Optional `tailnet` provides its native admin settings page using HTTP only.
  It registers no policy, contribution authorizer or lifecycle handler.

Each package's `run` wrapper supplies its exact selection:

```sh
./backend --package-id=soda --soda-socket=/ipc/soda.sock --factory-socket=/factory/service.sock
./backend --package-id=tailnet --soda-socket=/ipc/soda.sock
```

Missing or unknown package IDs and positional arguments are refused.
`--soda-socket PATH` selects an absolute private Soda HTTP socket; its default
comes from `SODA_EXTENSION_SERVICE_SOCKET`. Fountain launches and supervises the
backend with its SDK transport and callback authority.

The required `soda` package also requires an absolute `--factory-socket PATH`.
It sends Factory requests directly to that executable's private extension
listener; Project, terminal and account requests use the Forge socket. The
existing frontend assets and native mount context are reused. The adapter does
not launch either product or choose a fallback when a service is unavailable.
The appliance mounts only Factory's extension socket directory into Fountain;
the separate Project control socket is not available to the extension.
The optional `tailnet` package does not accept a Factory socket.

Both private bridge hops verify the live native actor and exact package /
contribution context. Host Tailnet settings require a verified native site
administrator. Workspace contribution authorization permits only Soda's Spaces
page and Workspace panel; its username policy admits compatible Linux logins
and rejects `root`. The configured Soda operator remains relevant to other
product APIs; package startup does not read a separate operator identity file.

Configuration or transport failures print to standard error and exit `1`.
Closing the process closes its idle backend transport connections.

[Forgejo customization](../../docs/reference/forgejo.md) owns package integration;
[HTTP API](../../docs/reference/api.md) owns the service routes.

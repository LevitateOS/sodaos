# soda-extension

Connect Forgejo's native extension runtime to Soda's private backend. This is
the process backend of the installed Soda extension package, serving the Spaces
page, Tailnet page and Workspace panel declared in its manifest.

People open those contributions through native Forgejo navigation. Operators
install the package through the appliance's extension installation flow; the
Forgejo host launches and supervises the backend with its SDK transport.

## Package invocation

The packaged [run wrapper](../../system/containers/extension/run) invokes:

```sh
./backend --soda-socket=/ipc/soda.sock
```

`backend` is the compiled `cmd/soda-extension` program inside that package. The
native extension host must supply its transport, callback authority and data
directory. This invocation describes that managed process environment.

`--soda-socket PATH` selects an absolute private Soda HTTP socket; its default
comes from `SODA_EXTENSION_SERVICE_SOCKET`. Positional arguments are refused.
The receiving listener is the dashboard's extension socket, prepared by
[`soda-dashboard`](../soda-dashboard/README.md).

## Required identity and authority

The extension reads `operator-id` from the SDK-provided private data directory.
It requires a bounded regular file, no group/other write permission, and a
canonical positive Forgejo ID. Activation creates that file for the configured
Soda operator.

Contribution authorization permits the global Spaces page and Workspace panel;
the admin Tailnet page additionally requires the configured operator ID.
The username policy admits create/rename requests only for project-compatible
Linux logins and rejects `root`.

## Behavior and failures

The process forwards only admitted extension routes over the private socket.
Both the bridge and backend verify live native actor/admission context. Product
handlers then apply repository permissions, session checks and same-origin
mutation rules. Soda supplies no separate browser login or session cookie.

Configuration, identity or transport failures print to standard error and exit
`1`. Closing the process closes its idle backend transport connections.

[Forgejo customization](../../docs/reference/forgejo.md) owns the package and
native host integration. See [HTTP API](../../docs/reference/api.md) for service
routes, [Spaces](../../docs/product/spaces.md) for the user surface, and
[Operator setup](../../docs/guides/operator-setup.md) for installation prerequisites.

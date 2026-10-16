# soda-tailnet

Print read-only Tailnet connection guidance for a Soda appliance. Host operators
can run it directly; the host console welcome also calls it when available.

Private LAN operation works without Tailnet enrollment. The
[Networking guide](../../docs/architecture/networking.md) owns endpoint and
project reachability rules.

## Run

On an installed host:

```sh
/usr/bin/soda-tailnet
```

From a source checkout with the pinned Go toolchain:

```sh
go run ./cmd/soda-tailnet
```

The command takes no options and ignores extra arguments. It queries
the existing private `soda-host` settings view with a two-second deadline.
The caller needs access to `/run/soda/host.sock`.

## Read the output

The message reports connection state, administrator approval or expired authentication. When connected, it shows
the observed Tailnet identity and a usable endpoint: MagicDNS when enabled,
otherwise the observed IPv4 address.

Unavailable status directs an administrator to Admin → Extensions to enable
Tailnet if needed, then open its Settings. The helper performs no enrollment, configuration
change, service restart or network mutation.

A printed endpoint is an observed host identity. Browser services retain their
configured HTTPS origins, and host Tailnet enrollment does not establish project
subnet routing or client reachability.

Exit status `0` means the message was written, including unavailable-status
guidance. A failure to write output reports to standard error and exits `1`.

See [Operator setup](../../docs/guides/operator-setup.md) for appliance setup and
[HTTP API](../../docs/reference/api.md) for native Tailnet settings operations.

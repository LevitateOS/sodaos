# Host operation client

Go callers in Soda use this package to request privileged project operations
from the Rust `soda-host` daemon. It owns the Unix HTTP client and terminal wire
types; the daemon and native executors live in [`lib/host`](../../lib/host/README.md).

## Use from a Soda service

Construct `host.NewClient(socket)` with the configured private host socket.
For example, inside a function in this module:

```go
import "github.com/levitateos/sodaos/internal/host"

client := host.NewClient("/run/soda/host.sock")
environment, err := client.Inspect(ctx, "p0123456789abcdef01234567")
if err != nil {
    return err
}
// Use environment only after the observation succeeds.
```

`ctx` is the caller's context and the example ID must be replaced with the
selected project's recorded ID. A running installed daemon and socket access
are required; constructing a client does not authorize the requested operation.

## Available operations

- [`Client`](client.go): create, inspect, join and observe project access and
  connection details, using canonical `project` records.
- [`Lifecycle`](lifecycle.go), [`AccessKeys`](access_keys.go) and
  [`ObserveOS`](os.go): lifecycle changes, managed access keys and OS observation.
- [`OpenTerminal` and `TerminalStates`](terminal_client.go): private WebSocket
  terminal attachment and bounded terminal operations. Close owned streams.
- [`Prepare`](prepare.go), [`FactoryLaunch`](factory_client.go) and
  [`PrepareCandidate`](factory_candidate.go): supervised preparation, execution
  and candidate operations.
- [`TailnetSettings`](tailnet.go) and related methods: native Tailnet policy,
  enrollment and project operations.

Ordinary HTTP calls have a four-minute timeout; streaming lifetime is explicit.
Mutating methods can create accounts, change containers, alter keys or start
execution. An unavailable reply does not confirm absence or rollback. Identity
delivery results can contain credentials and must stay in trusted service code.

See [Projects](../../docs/product/projects.md),
[terminal reference](../../docs/reference/terminal.md),
[factory interfaces](../../docs/architecture/factory-interfaces.md) and
[trust model](../../docs/architecture/trust.md) for the owning contracts.

# Private identity broker client

Trusted Go services use this package to call Soda's identity broker over Unix
HTTP. It transports canonical [`identity`](../README.md) records for enrollment,
delegation and supervised execution. It has no broker server or CLI.

## Use inside the Soda module

Construct `client.New(socket)` with the service's configured admin socket:

```go
import identityclient "github.com/levitateos/sodaos/internal/identity/client"

broker := identityclient.New("/run/soda/identity/admin.sock")
connections, err := broker.Connections(ctx, ownerID)
if err != nil {
    return err
}
// Use connections as metadata; no credential delivery occurs here.
```

`ctx` and `ownerID` come from trusted service admission. The running identity
service and appropriate Unix socket permissions are prerequisites. This socket
must never be mounted in a project or exposed to browser callers.

The [`Client` methods](client.go) cover connection/availability reads,
enrollment start/read/cancel, grant creation/revocation, lease acquisition/end,
process registration, credential return/rejection and execution reconciliation.
Choose the method for the intended operation; the broker enforces custody.

## Behavior and effects

Calls use a 30-second HTTP timeout, refuse redirects and bound decoded responses
to 512 KiB. Broker refusal codes map to the `identity` error vocabulary;
transport failures report that the broker is unavailable.

Enrollment, grants and lease methods can change broker state. `Register` can
return private credential bytes, and `Return` transmits them back to custody.
Keep these values out of logs, browser responses and project-wide sockets.
The client does not persist credentials or implement provider login itself.

See [credentials](../../../docs/reference/credentials.md) for user enrollment
and exposure rules, [trust model](../../../docs/architecture/trust.md) for the
service boundary, and [identity service](../../../cmd/soda-identity/README.md)
for operator use.

# tailnet

This package owns the Go Tailnet request/response records, their validation and
canonical MagicDNS names. It performs no process execution or network I/O.

[`control_types.go`](control_types.go) defines host and enrollment settings,
Project options/actions and their results. Request validation enforces revision,
binding, confirmation and field contracts. Response validation rejects
inconsistent native observations. `ProjectSelection.Validate()` handles the
creating human's explicit Project choice; defaults do not change saved choices.

The Go [host client](../host/tailnet.go) reads these records from the private
Rust host service. Native container ownership, credentials and Tailscale
execution belong to [the Rust Tailnet owners](../../lib/host/src/tailnet).
Validation does not grant authority to perform an operation.

Operators can use [`soda-tailnet`](../../cmd/soda-tailnet/README.md) for console
guidance through the existing host settings client. The
[networking guide](../../docs/architecture/networking.md) owns endpoint and
reachability rules.

Run `go test ./internal/tailnet` for the domain and wire checks.

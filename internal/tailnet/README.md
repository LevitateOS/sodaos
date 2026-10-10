# tailnet

Read the appliance's local Tailscale identity and validate Soda's Tailnet
request/response records. This internal Go package serves status and service
integration consumers; it has no executable. Operators can use
[`soda-tailnet`](../../cmd/soda-tailnet/README.md) for console guidance.

## Observe local status

`tailnet.New(tailnet.Options{})` selects `/usr/bin/tailscale`. An explicit
`Options.CLI` selects a caller-owned executable for controlled integration use.
Call `Client.Status(ctx)` with a bounded context to run `tailscale status --json`.

The returned `Status` carries backend state, observed IPv4, canonical MagicDNS
identity, expiry and pending-authentication facts. `Status.URLHost()` chooses
MagicDNS when enabled and available, otherwise IPv4. `Client.Endpoint(ctx)`
requires a running, unexpired node and an IPv4 address before returning a usable
host identity.

The CLI must exist, and the caller must have access to the local Tailscale
service. Output collection is bounded to 64 KiB on stdout and 32 KiB on stderr.
Unavailable CLI/output, unenrolled state, missing IPv4 and invalid MagicDNS
identity have distinct sentinel errors. `CanonicalMagicDNSName` validates and
normalizes native DNS names.

## Validate service records

[`control_types.go`](control_types.go) defines host and enrollment settings,
project options/actions and their results. Request `Validate()` methods enforce
the action-specific revision, binding, confirmation and field contracts.
Response `Validate()` methods reject inconsistent or unobservable native state.

`ProjectSelection.Validate()` handles the creating human's explicit project
selection: disabled/omitted means Off; enabled requires the selected revision
and binding. Operator defaults do not retroactively change that choice.

These records cross the Go host-client and API boundaries. Native host actions,
credential custody and companion execution belong to the Rust host service.
Validation alone neither executes an action nor establishes its authorization.

## Limits and authority

Status reads execute only the selected Tailscale status command; validators do
no IO. The package does not enroll nodes, launch companions, edit networking or
persist policy. An observed address is not a client reachability receipt, and
host enrollment does not establish project subnet routing.

[Networking](../../docs/architecture/networking.md) owns the access model;
[HTTP API](../../docs/reference/api.md) owns the operator/project routes. See
[Trust](../../docs/architecture/trust.md) for authority and
[Project OS](../../docs/reference/project-os.md) for runtime boundaries.

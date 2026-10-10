# Factory coordinator

`control` coordinates Soda's recorded factory work inside the dashboard backend.
It combines durable records, fresh native observations, host execution and broker
settlement. Backend integrators consume this internal Go package; operators use
Soda's admitted factory controls and the `soda-factory` CLI.

## Construct and start

`internal/web/server.go` is the normal integration owner. Its core construction
is:

```go
coordinator := control.NewCoordinator(db, hostClient, brokerClient)
```

The dependencies are a `*store.Store`, `HostFactory` and `BrokerExecution`.
The host owns privileged Project execution; the broker owns credential custody.
The coordinator receives execution metadata rather than credential bytes.

Call `Start(ctx, lockPath)` before serving. It takes an exclusive file lock,
reconciles recorded runs and dispatch recovery, and advances configured
publication/check/merge recovery. All backends serving the same ledger must use
the same protected, writable lock path. A second coordinator refuses ownership.
On shutdown, `StopAdmission` rejects new work and `Close` waits for admitted HTTP
handlers before releasing ownership.

## Integration surfaces

| Surface | Use |
| --- | --- |
| `Status`, `Stop`, `Reconcile` | Read recorded runs and retire/settle admitted executions. |
| `OperatorHandler`, `WithOperatorPrincipal` | Private `POST /operator/factory` with the listener's authenticated OS principal. |
| Acceptance, policy and grant methods | Record authorized issue decisions and revision-checked settings. |
| `ObserveIssueEvent`, `Dispatch` | Assess native issue evidence and launch bounded recorded assignments. |
| `PublishPass`, `CheckPass`, `MergePass` | Advance persisted publication, review/check and conditional merge work. |
| Repository pause/resume, run retry/takeover and Project stop methods | Apply explicit lifecycle decisions while preserving unresolved effects. |

The web facade wires observation and operation adapters through `AcceptanceReads`,
`Readiness`, `DispatchReads`, `Publication`, `Reviews`, `Checks` and `Merges`.
Missing dependencies yield unavailable/waiting outcomes rather than guessed
evidence. These methods can change database records and invoke host, broker or
native Forgejo operations; they are not status-only helpers.

## Operator behavior

`Status` returns bounded recorded metadata with credential-adjacent paths
redacted; it does not query the host or broker. The private operator endpoint
accepts only status, stop and reconcile. Its listener supplies the principal;
request bodies cannot supply human identity or policy authority.

Mutating commands use durable IDs: identical replay returns the recorded result,
changed content conflicts, and an unfinished duplicate returns accepted status.
Unknown termination, credential return or native effects remain fenced. A failed
observation does not authorize a new execution or erase earlier effects.

Follow [the operator contract](../../../docs/reference/factory.md),
[factory interfaces](../../../docs/architecture/factory-interfaces.md), and
[trust boundaries](../../../docs/architecture/trust.md). Actual dependency wiring
is in [the dashboard facade](../../web/server.go).

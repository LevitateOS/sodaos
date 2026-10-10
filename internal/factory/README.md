# Factory records and validation

`factory` defines Soda's canonical factory records and their pure validation.
It is an internal Go package consumed by the dashboard, coordinator, store,
host client and Forgejo adapters. Operators encounter these records through
factory status, policy, issue acceptance and run controls.

Use the [factory operator interface](../../docs/reference/factory.md) for
existing CLI operations and the [product overview](../../docs/product/overview.md)
for the intended workflow. This package has no executable or persistent state.

## Consumer entrypoints

Import `github.com/levitateos/sodaos/internal/factory` from Soda code. Important
record families include:

| Records | Purpose |
| --- | --- |
| `Run`, `RunAdmission`, `Outcome` | Execution identity, captured authority/profile, provenance and settlement. |
| `Command` | Idempotent operator and settings mutations. |
| `RepositoryPolicy`, `Capacity`, `OperatorGrant`, `Sponsorship` | Explicit operating authority and resource limits. |
| `Acceptance`, `IssueControl`, readiness records | Accepted native issue inputs and eligibility observations. |
| `Assignment`, result and allowance records | Bounded dispatch, attempts and reported work. |
| `Publication`, review, check and merge records | Persisted native-operation intent and outcome evidence. |

Construct records using admitted inputs and call their `Validate` methods before
crossing persistence or execution boundaries. For example, validating a stop
record performs no stop:

```go
cmd := factory.Command{
    ID: factory.NewID(), Type: factory.CommandStop,
    Target: runID, Principal: principal,
    Digest: factory.CommandDigest(factory.CommandStop, runID),
}
err := cmd.Validate()
```

Here `runID` is a recorded run and `principal` comes from the admitting service.
`NewID` creates a random 32-character lowercase hexadecimal identity. Reuse the
recorded command ID for an identical replay; changed content requires a new ID.
Status reads carry no durable command.

## Constraints

Validation checks identifiers, stage transitions, evidence bindings and bounded
payloads. A run includes a pinned image, input commit, deadline, harness/model
provenance and broker lease/binding facts. Validating those values does not grant
authority or prove that a native process exists.

Project and broker identities remain owned by `project` and `identity`; consumers
reference those types directly. Runtime decisions belong to
[factory/control](control/README.md), and SQL belongs to
[store](../store/README.md).

See [factory interfaces](../../docs/architecture/factory-interfaces.md) and the
[trust model](../../docs/architecture/trust.md) for authority boundaries.

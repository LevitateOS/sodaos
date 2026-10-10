# Appliance PostgreSQL store

`store` owns Soda's PostgreSQL schema and typed row operations. The dashboard
and factory coordinator consume this internal Go package to retain users,
development keys, Projects, membership, preparation decisions and factory
ledgers. Operators configure the database through appliance setup; end users
use product APIs and controls rather than direct SQL.

## Open the production store

The dashboard loads the DSN from its restricted `database_dsn_file` and the
32-byte encryption key from `grant_key_file`, then uses:

```go
db, err := store.OpenEncrypted(dsn, key)
```

Handle the error before using `db` and call `db.Close()` when its owner shuts
down. Opening connects through the PostgreSQL driver, checks the encryption key
against existing keyed state, initializes an empty database, or validates an
existing current schema. It refuses an unversioned nonempty database, an
incompatible schema or the wrong key; it does not silently migrate those states.

The database role needs access to the selected database and permission to create
the schema on first initialization. `Open(dsn)` is the unencrypted entrypoint;
the serving dashboard uses `OpenEncrypted`. `SchemaVersion()` reports the source's
current schema format without connecting.

## Consume typed records

| API area | Examples |
| --- | --- |
| Users and saved keys | `UpsertUser`, `User`, key inventory and confirmed removal. |
| Projects and members | `CreateProject`, `Project`, `ProjectByRepository`, `Join`. |
| Preparation and grants | Recorded input decisions, approvals and revision-checked authority. |
| Factory ledger | `FactoryRun`, `FactoryRuns`, `FactoryCommand`, assignments, reservations, usage and publication/merge records. |

For example, an integration reads a recorded run with
`run, err := db.FactoryRun(ctx, runID)`. Use `errors.Is(err, store.ErrNotFound)`
for an absent row. `ErrCommandConflict` signals a reused command identity with
different content; `ErrGrantKey` signals missing or incorrect encryption input.
`ErrLastKeyConfirmationRequired` preserves the explicit final-key removal gate.

## Persistence boundaries

Typed methods enforce record validity, transactions and their stated revision
conditions. Store writes do not provision accounts, start containers, retire
native processes or prove Forgejo effects. The coordinator and host boundaries
perform those operations and persist their evidence here. Keep product SQL in
this package; consumers use its exported records and methods.

`OpenEphemeral` creates a disposable PostgreSQL database for integration fixtures
and returns its cleanup function. It requires a suitable fixture superuser DSN;
it is separate from opening the appliance store.

See [service configuration](../../docs/reference/configuration.md),
[operator setup](../../docs/guides/operator-setup.md),
[database backup/restore](../../cmd/soda-pg-maintenance/README.md), and
[Go persistence ownership](../../docs/development/go.md#hard-invariants-persistence).

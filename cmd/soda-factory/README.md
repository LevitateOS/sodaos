# soda-factory

Inspect and retire supervised factory runs through the dashboard backend's
private operator interface. This command is for an operator whose OS identity
is admitted by that endpoint. The backend owns the durable run and command
records; this CLI keeps no database.

The [Factory reference](../../docs/reference/factory.md) owns the complete
operator contract. The [Trust model](../../docs/architecture/trust.md) explains
factory authority and execution boundaries.

## Prerequisites

Have a running dashboard backend configured with its private operator Unix
socket and an admitted peer UID. Supply the socket explicitly with `--socket`;
filesystem access alone does not establish endpoint authorization.

For a source checkout, build from the repository root with the pinned toolchain:

```sh
cargo build --locked -p soda-factory
```

## Read recorded state

```sh
target/debug/soda-factory --socket /run/soda/operator/operator.sock status
target/debug/soda-factory --socket /run/soda/operator/operator.sock status RUN_ID
```

Replace `RUN_ID` with a recorded run identity. Status returns the backend's
bounded recorded metadata, with credential-adjacent paths redacted. It carries
no durable command, so omit `--command` for status.

## Retire or reconcile runs

```sh
target/debug/soda-factory --socket /run/soda/operator/operator.sock \
  --command STOP_COMMAND_ID stop RUN_ID
target/debug/soda-factory --socket /run/soda/operator/operator.sock \
  --command RECONCILE_COMMAND_ID reconcile
```

Supply a distinct client-generated `COMMAND_ID` for each new action. Repeating
the same ID and payload retrieves its recorded outcome; using that ID for
different content conflicts. An unfinished repeated command can return an
accepted response; refresh its recorded state.

`stop` retires one recorded run and closes its broker execution. `reconcile`
settles outstanding runs through the same retirement path. Unconfirmed effects,
termination or credential return remain fenced and require intervention. These
commands do not admit, launch, retry or publish work.

Successful backend responses are written to standard output. Errors go to
standard error with exit status `1`, including unavailable endpoints, missing
runs and command identity conflicts. The client's request timeout is eleven
minutes; an observation failure does not prove the backend action was undone.

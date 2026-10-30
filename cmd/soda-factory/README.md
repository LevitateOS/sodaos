# soda-factory

Inspect and retire supervised factory runs through the dashboard backend's
private operator interface. This command is for an operator whose OS identity
is admitted by that endpoint. The backend owns the durable run
records; this CLI keeps no database.

The [Factory reference](../../docs/reference/factory.md) owns the complete
operator contract. The [Trust model](../../docs/architecture/trust.md) explains
factory authority and execution boundaries.

## Prerequisites

Have a running dashboard backend configured with its private operator Unix
socket and an admitted peer UID. Supply the socket explicitly with `--socket`;
filesystem access alone does not establish endpoint authorization.

For a source checkout, build from the repository root with the installed toolchain:

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
no durable command.

## Retire or reconcile runs

```sh
target/debug/soda-factory --socket /run/soda/operator/operator.sock stop RUN_ID
target/debug/soda-factory --socket /run/soda/operator/operator.sock reconcile
```

Each action uses the current recorded run state. An already settled run returns
its outcome without another host stop. No separate command receipt is stored.

`stop` retires one recorded run and closes its broker execution. `reconcile`
settles outstanding runs through the same retirement path. Unconfirmed effects,
termination or credential return remain fenced and require intervention. These
commands do not admit, launch, retry or publish work.

Successful backend responses are written to standard output. Errors go to
standard error with exit status `1`, including unavailable endpoints, missing
runs. The client's request timeout is eleven
minutes; an observation failure does not prove the backend action was undone.

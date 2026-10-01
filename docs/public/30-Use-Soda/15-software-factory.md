# Supervise software factory runs

Factory work runs as supervised agent executions inside a Project, with durable
run records, exact process binding and credential return. The operator
interface inspects recorded runs, retires one run and settles outstanding
work. Automatic intake remains unavailable: nothing admits issues, launches
runs or publishes results on its own.

## Inspect recorded runs

Use the private operator command against the dashboard backend's operator
socket:

```sh
soda-factory --socket /run/soda/operator/factory.sock status
soda-factory --socket /run/soda/operator/factory.sock status RUN_ID
```

Status returns bounded recorded metadata for each run: Project, role, input
revision, deadline, harness and model provenance, broker lease facts, outcome
and whether the run is settled. Credential-adjacent paths are redacted. A
run whose outcome is recorded but unsettled still needs reconcile.

Run outcomes are `succeeded`, `failed`, `cancelled` and `needs-human`.
Settlement is recorded separately: a finished execution is not proof that
retirement, credential return and broker closure all confirmed.

## Stop one run

```sh
soda-factory --socket /run/soda/operator/factory.sock --command COMMAND_ID stop RUN_ID
```

Stop retires exactly the recorded run's processes, seals its broker
execution and settles the run when both confirm. The command ID is yours to
generate: repeating the same ID with the same run replays the stored
outcome, and reusing an ID for a different run conflicts instead of acting
twice. If retirement or closure stays uncertain, the run stays unsettled for
reconcile; never treat a fenced stop as proof the run ended.

Human sessions and sibling runs are never targeted: a stop addresses one run
identity and its recorded unit and process group only.

## Settle outstanding work

```sh
soda-factory --socket /run/soda/operator/factory.sock --command COMMAND_ID reconcile
```

Reconcile retires every outstanding recorded run and settles accounting for
each one whose effects confirm. It cannot launch, retry or publish work.
Runs whose effects stay unresolved are reported as fenced with their
reasons; resolve the underlying outage and reconcile again with a fresh
command ID. Dashboard startup settles outstanding runs before serving.

## Understand the execution boundary

Each run executes a fixed CLI entrypoint under a recorded identity: run ID,
Project container, host unit and systemd incarnation. The Identity Broker
delivers subscription credentials only after the host attests that exact
waiting binding, and takes the state back after the run's descendants
retire. A run never sees the broker or host sockets. Unknown credential
return keeps the connection unavailable until the run is accounted for.

The only proved factory harness is Codex CLI `0.157.1`, pinned per run with
its executable digest. Other harnesses stay unavailable until their own
supervised-run proof passes. Soda uses subscription access only and never
switches to paid API billing if the subscription becomes unavailable.

Persistent human projects remain available for investigation and manual
development. See [Projects and workspaces](20-projects-and-workspaces.md) and
[Connect and develop](../40-Develop/10-connect-and-develop.md).

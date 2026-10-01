# Supervised factory-run operator interface

This reference describes the existing `cmd/soda-factory` operator interface:
status reads recorded run state, stop retires one run, and reconcile settles
outstanding runs. Runs execute as supervised Project CLI runs with durable
start/stop receipts, exact run/container/unit/incarnation binding and
credential return; the [Identity Broker](credentials.md#identity-broker) owns
subscription custody and the host owns native execution. Automatic intake
remains unavailable: nothing admits, launches, retries or publishes work.
The target automatic lifecycle belongs to the
[product overview](../product/overview.md#software-factory-workflow); these
command constraints do not define its limits. Component responsibilities
belong to [architecture](../architecture/overview.md), and authority to
[trust](../architecture/trust.md).

## Operator endpoint

The `factory/control` coordinator runs inside the dashboard backend and owns
the factory run/command ledger in that service's database. Operator commands
reach it through `POST /operator/factory` on a private Unix socket that is
not reachable through the public HTTP mux. The listener admits only its
configured OS peer UID and records that peer as the command principal; no
command can supply a human native identity, accept requirements, change
repository policy, enlarge sponsorship or clear a native reservation.

Build the operator binary with the repository's pinned Go toolchain:

```sh
go build -o .artifacts/soda-factory ./cmd/soda-factory
.artifacts/soda-factory --socket /run/soda/operator/factory.sock status
.artifacts/soda-factory --socket /run/soda/operator/factory.sock status RUN_ID
.artifacts/soda-factory --socket /run/soda/operator/factory.sock --command COMMAND_ID stop RUN_ID
.artifacts/soda-factory --socket /run/soda/operator/factory.sock --command COMMAND_ID reconcile
```

Status is a read without a durable command and returns bounded recorded run
metadata with credential-adjacent paths redacted. Stop and reconcile carry a
client-generated command ID: the same ID with the same payload replays its
stored outcome, and reuse for different content conflicts. A duplicated
unfinished command returns 202 so the caller refreshes instead of executing
twice.

## Stop and reconcile

Stop retires one recorded run through the host's exact run boundary and seals
its broker execution, then settles the run when retirement and closure
confirm. A completed run reports `succeeded`, a failed run `failed`, and a
retired run `cancelled`. Uncertain retirement or broker closure stays fenced:
the run keeps its unsettled state and the command outcome records the
uncertainty. No control reports `cancelled` while the native effect or
process retirement is still unresolved.

Reconcile applies the same path to every outstanding recorded run: it
retires remaining execution, closes broker executions and settles
accounting. It cannot launch, retry, spend or publish. Runs whose effects
stay unresolved are listed as fenced with their reasons. Settled runs never
reopen. Dashboard startup takes exclusive coordinator ownership of the
database and settles outstanding runs before serving.

Run identities are never reused. A stop tombstone bars its identity from
future work even when the run never reached the host, and a closed broker
execution never acquires a replacement lease.

## Run provenance and custody

Each recorded run carries its Project, role, input revision, deadline, pinned
image and harness/model provenance, broker lease and generation, exact
process binding and credential delegation/return facts. The only proved
factory harness is Codex CLI `0.157.1`; other harnesses stay unavailable
until their own supervised-run proof passes.

The [Identity Broker](credentials.md#identity-broker) delivers subscription
credentials only after the host attests the exact waiting run binding, and
only for that lease and binding. Credential state is staged outside the
checkout and returned after the run's descendants retire; a consumed start
marker followed by an absent unit stays fenced rather than starting twice.
The coordinator receives lease and binding metadata, never credential bytes.
Unknown credential return keeps the connection unavailable until the run is
accounted for.

## Retained publication validation

Candidate publication is a separately persisted conditional operation built
on the retained publisher validation. `host/publish` still imports a bounded
bundle into a fresh bare repository, disables hooks and inherited Git
configuration, checks ancestry against the admitted input, rejects changes
under `.forgejo/` or `.github/workflows/`, and rejects known before/after
credential literals in decoded Git objects reachable beyond the admitted
base, including deleted historical files and commit messages. Each
validation scans at most 10,000 objects, 4 MiB per decoded object and 32 MiB
total decoded content; exceeding a limit denies validation. This literal
check does not prevent intentional encoding or exfiltration by code that can
read the injected credential. Source bundles stay bounded to 4 MiB. No push,
direct PR or review submission path remains.

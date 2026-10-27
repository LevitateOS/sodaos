# Supervised factory-run operator interface

This reference describes the Rust `cmd/soda-factory` operator interface:
status reads recorded run state, stop retires one run, and reconcile settles
outstanding runs. Runs execute as supervised Project CLI runs with durable
start/stop receipts, exact run/container/unit/incarnation binding and
credential return; the [Identity Broker](credentials.md#identity-broker) owns
subscription custody and the host owns native execution. The dashboard's
automatic lifecycle uses native intake and current repository authority;
these operator commands cannot admit, launch, retry or publish work.
The automatic lifecycle belongs to the
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

Build the operator binary from the pinned Cargo workspace:

```sh
cargo build --locked -p soda-factory
target/debug/soda-factory --socket /run/soda/operator/operator.sock status
target/debug/soda-factory --socket /run/soda/operator/operator.sock status RUN_ID
target/debug/soda-factory --socket /run/soda/operator/operator.sock --command COMMAND_ID stop RUN_ID
target/debug/soda-factory --socket /run/soda/operator/operator.sock --command COMMAND_ID reconcile
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

While it owns the database, the same coordinator revisits durable automatic
work every ten seconds through its serialized dispatch, publication, checks
and merge passes. The same serving owner recovers accepted-intent graph
operations and records funded provisional plans, refreshes plans against actual
dependency outputs before coding, and consults complete eligible candidate sets
before fresh dispatch. Consultation configuration belongs to
[service configuration](configuration.md#dashboard-fields); its product limits
belong to [planning and priority](../product/overview.md#planning-and-priority-judgments).
It inspects outstanding runs and settles only confirmed
terminal execution; a live CLI stays supervised by the host. Delayed native
CI and completed review/correction runs can therefore advance without another
issue delivery. Each pass retains its existing bounded readiness budget and
every attempt retains its original allowance. Shutdown cancels and joins this
work before releasing database ownership.

Run identities are never reused. A stop tombstone bars its identity from
future work even when the run never reached the host, and a closed broker
execution never acquires a replacement lease.

## Run provenance and custody

Each recorded run carries its Project, role, input revision, deadline, pinned
image and harness/model provenance, broker lease and generation, exact
process binding and credential delegation/return facts. Repository policy
explicitly selects each role's harness family, version and model. Admission
requires that selection to match the staged harness pin and Project image;
it cannot substitute another profile. The supported subscription credential
routes belong to the [credential contract](credentials.md#authentication-and-billing-choice).

Muse's admitted role profile also binds reasoning effort and Contributor mode.
The common launch handoff takes that profile from the run's retained policy,
including investigation and review, which have no execution manifest. The
headless Muse command receives explicit `--reasoning-effort` and
`--contributor-mode on|off`. Supervised native sessions pass the effort through
MSP `turn/start` and retain per-process Contributor mode. Implementation children
select the fixed per-run `factory-task` native agent definition; its admitted
model/effort and actual child model, prompt, lineage and result must match before
output is released. Planned execution must match the same profile in its staged
manifest. A changed profile cannot replay an existing native run.
Older launches without profile fields retain their recorded behavior. This
requires the matching host implementation and a pinned CLI that supports the
flags; changing policy metadata alone does not establish those capabilities.

The [Identity Broker](credentials.md#identity-broker) delivers subscription
credentials only after the host attests the exact waiting run binding, and
only for that lease and binding. Credential state is staged outside the
checkout and returned after the run's descendants retire; a consumed start
marker followed by an absent unit stays fenced rather than starting twice.
The coordinator receives lease and binding metadata, never credential bytes.
Unknown credential return keeps the connection unavailable until the run is
accounted for.

Fresh reviewer candidate preparation requires an absolute deadline supplied
by its caller. The host caps snapshot work by that deadline and records it
for the detached setup/check worker; repeating the start cannot replace it.
Shorter HTTP or helper observation limits can report uncertainty without
extending the worker's deadline. Cleanup has bounded termination and join
grace periods. An unconfirmed child or output join keeps maintenance release
blocked, even when the original process group appears quiet. Manual
preparation uses the same child custody without a candidate deadline.

## Retained publication validation

Candidate publication is a separately persisted conditional operation built
on the retained publisher validation. `forgejo/publish` still imports a bounded
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

## Storage

The ledger uses PostgreSQL. Unconfigured intake refuses requests as unavailable;
it never bypasses authentication. The [trust model](../architecture/trust.md)
owns the authorization rules.

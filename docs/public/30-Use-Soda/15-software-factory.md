# Operate an AI software factory

Authorize objectives and operating policy, then supervise coding, review, verification and policy-governed merge in persistent SodaOS Projects.

SodaOS coordinates eligible issues through bounded agent executions. Forgejo
owns the collaboration records and native merge authority. Spaces provides
visibility and intervention; the private operator command inspects recorded runs,
retires one run and settles outstanding execution. Those command limits do not
define the automatic factory's scope.

## Prepare the factory

Configure private access and a repository with a meaningful check command.
Establish its required CI checks and review/protection policy. Coding agents,
review agents and CI runners have separate responsibilities; follow
[CI runners](50-ci-runners.md) for verification capacity.

Authorize three things separately: the repository's factory policy and native
actors, appliance capacity and execution limits, and provider-account sponsorship.
Choose supported harness/model profiles and finite allowances. Accept a clear,
bounded issue objective with its required outcomes and dependencies. An arbitrary
comment, label or agent instruction does not supply those grants.

## From an issue to a reviewed change

1. Create or adopt the exact objective as a currently authorized maintainer.
2. Resolve prerequisite outcomes and material questions. A blocked issue waits
   for the needed evidence or decision rather than guessing an answer.
3. Observe the coding execution in its persistent Project. Record the issue,
   attempt and candidate commit; a CLI exit alone is not completed software.
4. Inspect the generated pull request and separate review session. Corrections
   create a new candidate that needs current verification.
5. Confirm required check results and review evidence for the exact candidate.
   Supply human approval when the repository's policy requires it.
6. Confirm the native merge and issue outcome. Soda reconsiders dependent work
   after the required outcome, within the remaining authority and limits.

For a small exercise, ask for a testable change to one access rule in a disposable
repository. Include active/disabled users, allowed/denied roles and the expected
tests in the accepted requirements. Inspect a failing check and a corrected
candidate before expanding unattended work. Keep actual PR, review, CI and merge
references; do not substitute an agent summary for those records.

See [Review a factory candidate](35-collaboration.md#review-a-factory-candidate).
Merge is distinct from application deployment and from the business owner's
acceptance of the running application. Authorize and verify those steps separately.

## The everyday journey

Open New Project and answer **What do you want to build?** with a plain
description. Explore ideas turns it into a direction you review (**Review your
product direction**), accepting each choice or asking for changes. Supported,
reversible details settle on their own; only consequential assumptions wait
for you. Then authorize a finite planning allowance, choose a repository, and
let generation turn the accepted direction into real native issues you can
follow through review and policy-governed merge in Forgejo. Saved requests stay private to you;
resume any of them later from your list.

## Inspect recorded runs

Use the private operator command against the dashboard backend's operator
socket:

```sh
soda-factory --socket /run/soda/operator/operator.sock status
soda-factory --socket /run/soda/operator/operator.sock status RUN_ID
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
soda-factory --socket /run/soda/operator/operator.sock --command COMMAND_ID stop RUN_ID
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
soda-factory --socket /run/soda/operator/operator.sock --command COMMAND_ID reconcile
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

Select a harness and model supported by the chosen release and execution profile.
Each run records its harness and executable identity. Do not infer universal
provider or subscription compatibility. Soda's subscription connection path does
not switch to paid API billing when that subscription becomes unavailable.

Persistent human projects remain available for investigation and manual
development. See [Projects and workspaces](20-projects-and-workspaces.md) and
[Connect and develop](../40-Develop/10-connect-and-develop.md).

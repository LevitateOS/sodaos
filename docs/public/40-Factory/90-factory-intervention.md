# Factory intervention

Factory intervention resolves blocked work, interrupted runs and uncertain effects through the responsible owner and a bounded explicit action, preserving shared Project state.

## Identify the actual blocker

Read the current work state and its owner before acting. A missing acceptance or unresolved requirement belongs with an authorized repository maintainer. A dependency needs its native prerequisite outcome or an authorized graph change.

An unprepared Project belongs with the Project preparation owner. Missing appliance capacity belongs with the Soda operator; an unavailable or exhausted sponsored account belongs with its provider owner. A required CI check that cannot run belongs with the repository's Forgejo Actions runner owner.

Factory distinguishes blocked objectives from resource waits and execution failures. The status identifies the observed cause, what is waiting, which limit remains and the next responsible owner. If evidence is stale or the cause is uncertain, reconcile or reassess the existing work before starting another attempt.

Status and stop/reconcile commands support operator intervention; they are not required command steps in the normal automatic issue path.

## Choose the action that matches the situation

| Action | Effect and required conditions |
| --- | --- |
| Pause | Closes further dispatch, stops assigned agents and preserves their work. |
| Resume | Revalidates accepted inputs, Project readiness and grants, then continues with the attempt's remaining allowance in fresh sessions. It does not resume an old CLI conversation. |
| Cancel | Withdraws further execution, publication and merge authority. |
| Retry | Starts a new bounded attempt only after the previous execution, native effects and credentials are accounted for. Its allowance does not erase earlier consumption or the provider's rolling usage. |
| Takeover | First stops assigned runs, settles credentials and establishes outstanding native operation outcomes. Once confirmed, an authorized maintainer who joined the Project receives retained work in a separate checkout and terminal. It never transfers a live agent terminal, credential home or provider lease. Returning work requires fresh assessment, review and checks. |

These actions have different consequences and should not be treated as synonyms. A provider grant withdrawal, repository permission change or operator stop can also prevent continuation. None of them removes already recorded evidence, authorizes an unverified merge or destroys the shared Project. Ordinary cleanup never recreates the Project to end a Factory run.

## Reconcile uncertain effects and bounded repairs

When a response is lost or a process ends unexpectedly, Factory checks the original operation and its receipts to determine whether it committed, failed or remains uncertain. An uncertain native effect, process termination or provider lease return stays visible until reconciled; do not repeat the action or assume credentials have returned safely. Reconciliation settles the existing operation and does not revive the old execution or enlarge its allowance.

If a completed-work inventory reveals an unexplained stall, Factory can admit a bounded diagnosis and repair. The Strategist receives the affected dependency graph, protected work and remaining repair allowance, then proposes a scoped change for validation. A valid repair does not erase completed outcomes, active manual work or unrelated tasks.

New readiness is confirmed separately after the native effects are observed. If the diagnosis or repair allowance is exhausted, the Feed identifies the remaining owner and required decision.

Whole-host service, storage or appliance recovery belongs to [Recovery](../70-Operate/80-recovery.md), not run intervention. For routine activity and evidence, see the [Factory Feed](70-factory-feed.md).

## Operator command support

An appliance operator admitted by the private endpoint can inspect recorded runs, stop one run or reconcile outstanding executions with `soda-factory`. Replace `RUN_ID` with the recorded run and each command-ID placeholder with a distinct identifier for that new action.

Status is a read. Stop retires the named run; reconcile retires outstanding executions and settles accounting, so it is a mutation rather than a passive refresh. These commands cannot admit, launch, retry or publish work, change policy or increase an allowance.

```sh
soda-factory --socket /run/soda/operator/operator.sock status
soda-factory --socket /run/soda/operator/operator.sock status RUN_ID
soda-factory --socket /run/soda/operator/operator.sock \
  --command STOP_COMMAND_ID stop RUN_ID
soda-factory --socket /run/soda/operator/operator.sock \
  --command RECONCILE_COMMAND_ID reconcile
```

Status returns recorded run metadata. Stop and reconcile report their recorded settlement outcome; unconfirmed process retirement, native effects or credential return remain fenced for intervention.

After a lost response, reuse the same command ID and payload to inspect its recorded outcome; do not create a new action to retry an uncertain one.

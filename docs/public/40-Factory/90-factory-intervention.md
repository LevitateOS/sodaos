# Factory intervention

Intervene when Factory identifies a blocker that needs an owner, an exhausted allowance, an explicit decision or safe reconciliation of an interrupted operation. The status and stop/reconcile controls support this work; they are operational support for the automatic issue journey, not a required command step for every issue.

## Identify the actual blocker

Read the current work state and its owner before acting. A missing acceptance or unresolved requirement belongs with an authorized repository maintainer. A dependency needs its native prerequisite outcome or an authorized graph change. An unprepared Project belongs with the Project preparation owner. Missing appliance capacity belongs with the Soda operator; an unavailable or exhausted sponsored account belongs with its provider owner. A required CI check that cannot run belongs with the repository's Forgejo Actions runner owner.

Factory distinguishes blocked objectives from resource waits and execution failures. The status identifies the observed cause, what is waiting, which limit remains and the next responsible owner. If evidence is stale or the cause is uncertain, reconcile or reassess the existing work before starting another attempt.

## Choose the action that matches the situation

Pause closes further dispatch, stops the assigned agents and preserves their work. Resume revalidates accepted inputs, Project readiness and grants, then uses the attempt's remaining allowance with fresh sessions; it does not resume an old CLI conversation. Cancel withdraws further execution, publication and merge authority. Retry explicitly starts a new bounded attempt only after the previous execution, native effects and credentials are accounted for. Its new attempt allowance does not erase earlier consumption or the provider's rolling usage.

Takeover first stops the assigned runs, settles their credentials and establishes the outcome of outstanding native operations. After those steps are confirmed, an authorized maintainer who has joined the Project receives the retained work in their own separate checkout and terminal. Takeover never transfers a live agent terminal, credential home or provider lease. Handing the work back requires fresh assessment, review and checks.

These actions have different consequences and should not be treated as synonyms. A provider grant withdrawal, repository permission change or operator stop can also prevent continuation. None of them removes already recorded evidence, authorizes an unverified merge or destroys the shared Project. Ordinary cleanup never recreates the Project to end a Factory run.

## Reconcile uncertain effects and bounded repairs

When a response is lost or a process ends unexpectedly, Factory checks the original operation and its receipts to determine whether it committed, failed or remains uncertain. An uncertain native effect, process termination or provider lease return stays visible until reconciled; do not repeat the action or assume credentials have returned safely. Reconciliation settles the existing operation and does not revive the old execution or enlarge its allowance.

If a completed-work inventory reveals an unexplained stall, Factory can admit a bounded diagnosis and repair. The Strategist receives the affected dependency graph, protected work and remaining repair allowance, then proposes a scoped change for validation. A valid repair does not erase completed outcomes, active manual work or unrelated tasks. New readiness is confirmed separately after the native effects are observed. If the diagnosis or repair allowance is exhausted, the Feed identifies the remaining owner and required decision.

Whole-host service, storage or appliance recovery belongs to [Recovery](../70-Operate/80-recovery.md), not run intervention. For routine activity and evidence, see the [Factory Feed](70-factory-feed.md).

## Operator command support

An appliance operator admitted by the private endpoint can inspect recorded runs, stop one run or reconcile outstanding executions with `soda-factory`:

```sh
soda-factory --socket /run/soda/operator/operator.sock status
soda-factory --socket /run/soda/operator/operator.sock status RUN_ID
soda-factory --socket /run/soda/operator/operator.sock \
  --command STOP_COMMAND_ID stop RUN_ID
soda-factory --socket /run/soda/operator/operator.sock \
  --command RECONCILE_COMMAND_ID reconcile
```

Replace `RUN_ID` with the recorded run and each command-ID placeholder with a distinct identifier for that new action. After a lost response, reuse the same command ID and payload to inspect its recorded outcome; do not create a new action to retry an uncertain one. Reconcile retires outstanding executions and settles accounting, so it is a mutation rather than a passive refresh. These commands cannot admit, launch, retry or publish work, change policy or increase an allowance. Unconfirmed process retirement, native effects or credential return remain fenced for intervention.

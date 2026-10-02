# CI runners

Supply separately managed Forgejo Actions capacity for candidate verification.
Local Soda execution is unavailable and no local runner controls ship:
there is no local Runners page, registration, Start, Stop, Remove or Restart.

Forgejo owns workflows, scheduling and results. Follow
[Forgejo Actions administration](https://forgejo.org/docs/latest/admin/actions/)
for external capacity. Keep runner credentials out of repository files,
screenshots and shared logs.

## Verify factory candidates

CI jobs are verification executions, separate from implementation and review agents.
Use the selected Forgejo Actions workflow to evaluate each candidate once on
`pull_request` events for `opened` and `synchronize`. Bind checkout and results
to the actual candidate commit; branch names alone are insufficient.

Soda observes Forgejo's result and rejects evidence for another commit or multiple
candidate evaluations. A new repair commit requires a new CI evaluation and fresh
review. CI configuration is trusted policy, outside the agent's permitted changes.
See the [factory walkthrough](15-software-factory.md#prepare-the-factory).

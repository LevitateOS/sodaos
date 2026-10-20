# CI runners

Native Forgejo Actions runners supply required check results for each exact candidate; Factory coding and review agents do not provide CI capacity.

## Configure required checks

1. **Provide native Actions capacity.** A repository owner or administrator configures the repository's workflows and arranges runner capacity through Forgejo. Follow the official [runner installation, registration and configuration procedure](https://forgejo.org/docs/latest/admin/actions/) for that environment. Use repository, organization or instance scope appropriate to the intended repositories. Runner administration, operating-system setup, credentials and workflow code belong to the team's external execution environment.
2. **Set the Factory requirements.** The authorized owner names a nonempty set of required native checks in [Factory policy](20-factory-authorization.md#authorize-repository-work). Each check must report against the candidate revision considered for merge; a green result from an earlier commit cannot satisfy the current candidate.
3. **Confirm the candidate results.** Factory associates each observed result with the exact candidate. A runner outage or workflow that cannot run is an infrastructure wait or failure, not a passing check.

Registration supplies sensitive runner connection material. Store it in the
runner's protected configuration as directed by its setup screen; do not put it
in the repository, a screenshot or a Factory prompt. Match workflow job labels
to the runner's configured execution environment, and confirm that the runner
can reach the private Forgejo address and required dependency sources.

Workflows live in the repository's `.forgejo/workflows/` directory. Use the
repository's real build and test commands, with fixtures suited to its accepted
outcomes. The official [Actions concepts](https://forgejo.org/docs/latest/user/actions/basic-concepts/)
explain workflow events, jobs, steps and runner labels.

### Example: require the catalog checks

Suppose the catalog repository has a workflow that checks search behavior and
its interface. Use an actual pull request to establish the names reported by
Forgejo before selecting the required checks; do not guess a status name from
a workflow filename.

| What to inspect | Expected evidence |
| --- | --- |
| Workflow configuration | The candidate's pull-request event runs the intended behavior and interface jobs on an eligible runner. |
| Native check names | The exact reported names selected in Factory policy, such as the sample names “Search behavior” and “Interface checks.” |
| Candidate revision | Both results belong to the current pull-request head and its verified target/base. |
| Job logs and result | The intended commands and fixtures ran successfully; a skipped or cancelled job does not pass. |

Use the [personal contribution procedure](../30-Projects-and-development/30-collaboration.md#make-and-publish-a-small-change)
to push a normal candidate through the configured workflow, then inspect its
native check results. If no eligible runner accepts a job, fix runner scope,
labels or connectivity with its owner. If the command runs and fails, inspect
the failure as candidate evidence rather than removing the required check.

## Read check results

Factory waits for the configured required checks and associates observed results with the candidate. Missing, pending, failed, cancelled or stale results remain visible as such. A changed commit requires applicable checks for the new candidate. Repository branch protections and native merge rules continue to apply alongside Factory's review and acceptance evidence.

If the checks cannot run, restore or configure the native Actions capacity, then let the candidate's checks report. Do not substitute a Builder checkpoint, a reviewer's conclusion or a local command for a required native result. For how checks combine with review and native delivery, see [Review and merge](../20-Factory/50-review-and-merge.md).

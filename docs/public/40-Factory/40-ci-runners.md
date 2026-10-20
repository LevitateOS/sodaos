# CI runners

Native Forgejo Actions runners supply required check results for each exact candidate; Factory coding and review agents do not provide CI capacity.

## Configure required checks

1. **Provide native Actions capacity.** A repository owner or administrator configures the repository's workflows and arranges runner capacity through Forgejo. Runner administration, operating-system setup, credentials and workflow code belong to the team's external execution environment. Scope runner permissions to the repositories and code they execute.
2. **Set the Factory requirements.** The authorized owner names a nonempty set of required native checks in Factory policy. Each check must report against the candidate revision considered for merge; a green result from an earlier commit cannot satisfy the current candidate.
3. **Confirm the candidate results.** Factory associates each observed result with the exact candidate. A runner outage or workflow that cannot run is an infrastructure wait or failure, not a passing check.

## Read check results

Factory waits for the configured required checks and associates observed results with the candidate. Missing, pending, failed, cancelled or stale results remain visible as such. A changed commit requires applicable checks for the new candidate. Repository branch protections and native merge rules continue to apply alongside Factory's review and acceptance evidence.

If the checks cannot run, restore or configure the native Actions capacity, then let the candidate's checks report. Do not substitute a Builder checkpoint, a reviewer's conclusion or a local command for a required native result. For how checks combine with review and native delivery, see [Review and merge](50-review-and-merge.md).

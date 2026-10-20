# CI runners

Factory relies on required checks reported by native Forgejo Actions for the exact pull request candidate. Repository maintainers provide and configure the Actions capacity through Forgejo; Soda's coding and review agents do not act as CI runners.

## Configure required checks

An authorized repository owner or administrator configures native Actions workflows and runner capacity for the repository, then names the required checks in Factory policy. The required set must be nonempty. Each check must report against the candidate revision that Factory is considering for merge, so a green result from an earlier commit cannot satisfy current policy.

Runner administration, operating-system setup, credentials and workflow code belong to the repository's Forgejo Actions configuration and the team's external execution environment. Keep runner permissions scoped to the repositories and code they execute. A workflow that cannot run because its runner is offline or unavailable is an infrastructure wait or failure, not a passing check.

## Read check results

Factory waits for the configured required checks and associates observed results with the candidate. Missing, pending, failed, cancelled or stale results remain visible as such. A changed commit requires applicable checks for the new candidate. Repository branch protections and native merge rules continue to apply alongside Factory's review and acceptance evidence.

If the checks cannot run, restore or configure the native Actions capacity, then let the candidate's checks report. Do not substitute a Builder checkpoint, a reviewer's conclusion or a local command for a required native result. For how checks combine with review and native delivery, see [Review and merge](50-review-and-merge.md).

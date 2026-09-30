# Soda OS product overview

Soda OS is **the operating system for software factories**. It takes repository
issues through readiness assessment, coding, pull requests, review and correction,
automatic merge, and reconsideration of dependent work. AI coding CLIs run in the
repository's development environment, with account access supplied by Soda's
Identity Broker. People observe the work and intervene through Spaces.

This document owns the target factory behavior. The [factory reference](../reference/factory.md)
describes the implemented operator commands; those commands do not define the
limits of this product contract. [Trust](../architecture/trust.md) owns authority
and credential boundaries. [Projects](projects.md) owns the environment model.
This contract does not select a scheduler, wire protocol or implementation sequence.

Developers work through native Forgejo, ordinary SSH, Git, mise and container tools.
Soda supplies the integration that creates project environments, membership, access
keys, managed terminals, local CI integration and agent execution.
Developers do not receive individual Linux accounts on the host.

Public Internet hosting and a production hosting platform are outside this product.
Private LAN access and optional Tailscale access are in scope. Trying Soda or using
its baseline services must not require buying or owning a domain.

## Major concepts

| Concept | Meaning |
| --- | --- |
| **Host** | The Fedora CoreOS appliance. Operator administration only. |
| **Work Item** | A repository issue whose objective, dependencies and required outcomes drive factory work. |
| **Attempt** | One bounded effort to complete a work item, with recorded inputs, runs and outcome. |
| **Run** | One bounded agent execution against recorded inputs, with its own identity and resource records. |
| **Workspace** | The checkout or worktree assigned to work inside a repository environment; its lifetime need not equal an agent process's lifetime. |
| **Project** | One persistent repository environment shared by joined humans and authorized factory roles, with separate accounts and checkouts. |
| **Project OS** | The userspace foundation inside a project (accounts, tools, persistence, workloads). |
| **Session / view** | A session is a supervised human terminal or factory CLI execution; a Spaces view is an authorized attachment, not its lifetime owner. |
| **Spaces** | Fountain-hosted view of repository environments and live human and factory CLI sessions. |
| **Runner** | Native Forgejo Actions capacity, separate from coding/review agents; Soda-provisioned local execution is deferred. |
| **Tailnet** | Private connectivity for host and project reachability. |

Detailed ownership lives in [Projects](projects.md), [Spaces](spaces.md),
[Architecture](../architecture/overview.md) and [Project OS](../reference/project-os.md).
The [feature disposition](scope.md#feature-disposition) identifies what the rebuilt
factory retains, adapts, retires and defers, including supporting appliance features.

## Software factory workflow

Forgejo remains the source of truth for repositories, issues, pull requests,
reviews, merge results and CI. Soda coordinates the following loop for repositories
authorized for factory operation. Routine eligible issues advance automatically;
the normal path requires neither a per-issue operator admission command nor a
person performing each merge.

1. **Consider the issue.** A new issue enters readiness assessment. Relevant issue
   changes and dependency outcomes cause waiting work to be reconsidered. Soda
   records which objective and repository revision an attempt uses.
2. **Determine readiness.** An issue is blocked when a dependency is unresolved or
   when required information is missing, including questions raised by an agent.
   Discovering such a blocker during coding or review returns work to assessment.
   Soda exposes the reason, the relevant dependency or question, and
   the information needed to proceed. It must not assume an answer to clear a
   blocker. Resolving a blocker triggers reassessment; it does not bypass the other
   readiness conditions. Waiting for compute or an AI account is distinguishable
   from a blocked objective.
3. **Prepare the repository environment.** Soda uses the repository's persistent
   [Project](projects.md#environment-relationships), creating it when authorized
   and absent. Prepare the assigned checkout, toolchain and development services
   before coding. Human and factory work share that container while preserving
   unrelated dirty work, accounts and service data.
4. **Run the coding agent.** Soda starts the selected AI coding CLI with a prompt
   assembled from the issue, acceptance requirements and relevant repository
   context. The Identity Broker supplies the authorized provider connection.
   The agent's live CLI activity is visible through Spaces and attributable to its
   issue, repository and attempt.
5. **Publish the candidate.** Soda creates a pull request for the coding work and
   links it to the issue and attempt. The candidate identifies an exact revision;
   a successful CLI exit alone does not establish completion.
6. **Review and correct.** Soda starts another agent session in the same repository
   container to review the candidate. The reviewer reports findings; the coding
   agent owns corrections. A fresh review session checks each revised candidate
   under the [review rules](#review-and-correction).
7. **Merge automatically.** Soda performs the native forge merge when the required
   review and checks pass for the exact candidate and current repository policy
   permits it. Missing evidence, changed candidate state or unmet merge conditions
   prevent the merge. Human intervention is an exception path rather than a
   mandatory final step for every issue.
8. **Reconsider dependent work.** After confirmed merge and the corresponding issue
   outcome, Soda reevaluates issues that may now be unblocked and continues the
   same loop. Creating a PR or an agent claiming success does not count as a merge.

The operating rules below bound execution, correction and resource use. Duplicate
observations must not duplicate active work or erase consumed limits.

Closing or cancelling work withdraws further execution, publication and merge authority.
Recorded outcomes remain distinct from cleanup, and ending one agent must not
destroy a reused repository environment or unrelated work. The operating rules
below govern interruption and retry; automatic destructive recovery and
conversation snapshots remain outside scope.

## Factory operating rules

These are target product rules. The [operator reference](../reference/factory.md)
continues to describe the existing implementation. Who may enable work, sponsor
accounts and intervene is owned by [Trust](../architecture/trust.md#factory-authority-boundary).

### Readiness and blockers

| Condition | Meaning and next action |
| --- | --- |
| Not authorized | Repository automation, accepted issue input or a required grant is absent. Do not start an AI assessment or execution; show the missing authorization. |
| Blocked | A prerequisite outcome or material requirement is unresolved. Record the dependency/question and what would resolve it; wait for relevant evidence. |
| Ready / queued | The objective and authority permit work, but an execution slot, environment, provider lease or usage allowance is unavailable. State which resource is missing. |
| Active | The attempt is assessing, coding, correcting, reviewing or waiting for its candidate's checks within its limits. |
| Needs intervention | Execution failed, attempt time/correction limits were exhausted, a decision is disputed, credentials need reconnection, or safe continuation/termination cannot be established. Do not silently retry. |
| Cancelled / completed | Cancellation ends automatic progress; completion requires the confirmed native merge and issue outcome. Neither implies cleanup succeeded. |

Dependencies identify a forge issue and a required outcome, with an authorized
maintainer's confirmation of the relation. Agents may discover missing prerequisites
and raise a blocker, but cannot remove dependencies or declare them waived. For a
code dependency, resolution requires the linked change to be merged into its stated
target branch. Closing an issue without that outcome is insufficient. Non-code
prerequisites require a recorded maintainer resolution and supporting evidence.
Cycles and unavailable dependency evidence remain blocked until resolved; Soda
does not choose a cycle-breaking deletion. Dependency transport belongs to the
native forge contract, not a competing issue database.

A requirements blocker is a specific unanswered question that materially changes
the expected behavior, acceptance, authority or destructive impact and cannot be
answered from accepted requirements or repository evidence. The agent explains
the consequence and asks the smallest useful question. Ordinary implementation
choices, naming and equivalent technical approaches remain agent decisions.
Relevant trusted evidence can answer a factual question; a new product choice or
waiver needs a write-authorized maintainer. An unsupported assumption cannot clear
a blocker.

Reassess on accepted issue changes, dependency outcomes, blocker answers and
resource/authority changes. A blocker discovered mid-run stops affected coding,
publication and merge, preserves work and releases execution resources. Material
changes to the objective, acceptance or dependencies require reassessment and new
prompt inputs; cosmetic metadata changes do not. A blocked item is not repeatedly
sent to an agent while its inputs are unchanged. Resolved blockers can resume
automatically within remaining authority and limits.
Among eligible queued work, take the oldest ready issue first; dependency-blocked
work does not occupy an execution slot.

### Prompt assembly

Each run receives a recorded assignment assembled from:

1. Controller policy: repository/target, assigned role and checkout, permitted
   actions, limits, required evidence, and how to report a blocker or stop.
2. Accepted issue title/body, acceptance outcomes, authorized clarifications and
   dependency outcomes, identified by their input revision.
3. Repository instructions from the approved base, relevant source/docs/tests,
   environment setup and verification commands. Include only context relevant to
   the objective; do not dump unrelated conversations or private account files.
4. Stage inputs: the recorded base for coding; the candidate diff, exact head/base
   and verification results for review; consolidated findings for correction.
5. Required output: changed revision or findings, checks actually run, unresolved
   requirements and remaining risks. A narrative success claim is not evidence.

Record the selected harness/model/provider connection, prompt template revision
and input identities so the assignment is inspectable. Credentials are delivered
through the broker, never prompt text. Issue text, source, comments and tool output
cannot override controller policy or grant privileges. Repository instructions
guide implementation within that policy; material conflicts become explicit
blockers. An agent cannot choose another account, provider or model to evade limits.

### Concurrency and usage limits

The appliance operator sets finite execution capacity and CPU, memory, process and
scratch limits with headroom for host services and human work. A repository policy
may narrow that allocation; a provider owner separately limits sponsored account
use. The tightest applicable limit wins. Missing required limits prevent enablement.

The initial configurable defaults are:

| Limit | Default |
| --- | --- |
| Active issues across the appliance | 2 |
| Active attempts / executing factory sessions per repository | 1 / 1 |
| Cumulative active-attempt time | 120 minutes |
| Correction-and-reverification cycles after the initial candidate | At most 3 |
| Factory agent-session time per provider connection, across repositories | 6 hours per rolling 24 hours |
| Automatic usage-billed spending / overages | Zero |

These values are recorded when the responsible authorities enable their grants,
and may be lowered. Increasing a configurable limit requires its owning authority;
concurrent attempts inside one repository are outside the first operating model.
Host resource amounts depend on available capacity and must be supplied explicitly,
not guessed from the concurrency default.

Only one attempt may be active in a repository, with one factory agent session at
a time in its container. Other issues queue; different repositories may use the
appliance's configured slots. The coding checkout and review checkout are separate
from human checkouts and from each other under the
[checkout model](projects.md#accounts-and-checkout-ownership). Shared tools/services
do not become run-owned. Human changes to factory work require takeover first.
Broker constraints still apply: multiple appliance slots do not create concurrent
Codex credential streams or entitle one run to another connection.

An attempt has a finite cumulative active-time allowance, including assessment,
coding, correction, review and CI waits after activation. Before an agent begins,
reserve its capacity, account lease and remaining usage allowance. Resource or
rolling-quota waits queue the attempt; they do not occupy an active slot. Queued,
blocked or explicitly paused time is excluded only after its agents are confirmed stopped;
resumption uses the remaining allowance and fresh sessions, not a restored
conversation. Every correction-and-reverification cycle consumes its allowance
before starting, including unsuccessful cycles. Automatic retries, new delivery
IDs, accepted input edits and repeated observations cannot replenish it. Repeated
non-progress or exhaustion ends automatic continuation and requests intervention.

The provider-owner budget counts total leased factory session time across all
repositories using that connection in a rolling 24-hour window, including
readiness, failed runs and correction/review sessions. Overlapping sessions add
their durations; a lease remains counted until its assigned processes have stopped.
Native token/quota observations may further restrict admission where reliable,
but agent-hours are not a claimed token or currency measurement.
Human CLI use shares the upstream subscription's actual quota and can reduce
availability even though it is outside this factory allocation.

Soda-managed factory execution remains subscription-only, with zero automatic
usage-billed spending or overages. No run may silently change account/provider,
raise a limit, split itself into unaccounted agents or evade a provider restriction.
At a usage threshold, stop further execution and wait for allowance or an authorized
policy change; reconnection is required for rejected credentials. Any native limit
that cannot be enforced or observed must not be represented as a guaranteed cap.
The [credential contract](../reference/credentials.md#authentication-and-billing-choice)
owns supported billing and authentication routes.

### Review and correction

The coding role alone changes the candidate branch. The reviewer works from a
separate checkout of its exact candidate in the same container, receiving the
accepted objective, repository guidance, diff and test evidence, not the coding
conversation or a prewritten approval. It may inspect and run checks; it reports
findings rather than publishing fixes or merging. Sharing a container is not an
isolation or independent-reasoning guarantee.

Findings distinguish concrete correctness, security, acceptance and required-check
failures from optional suggestions. Each blocking finding identifies the defect,
relevant location/evidence and required outcome. The coding agent receives the
consolidated findings and either corrects them or explains a concrete disagreement.
A fresh review session reassesses the revised head and outstanding findings. A
reviewer's own change cannot receive that reviewer's approval. Disagreement without
progress goes to a maintainer instead of an unlimited agent debate. Every changed
candidate requires new required CI and review within the same remaining limits.
An actionable CI failure becomes correction input. Missing capacity or an
infrastructure failure cannot be resolved by weakening verification; wait for the
resource or report the intervention needed.

### Automatic merge conditions

Merge is allowed only when all of these remain true at the native merge operation:

- The issue and attempt remain authorized and active, with no blocker, pause,
  cancellation, takeover or exhausted limit, and the accepted objective/requirements
  still match the recorded inputs.
- The PR belongs to the assigned repository, branch and issue; its current head is
  exactly the reviewed candidate, and the intended target/base is still valid.
- Independent review passes with no unresolved blocking findings; every configured
  check in the nonempty required-check set passes for this candidate and the base
  against which it was verified. Missing, skipped, cancelled, pending or failed
  evidence is not a pass.
- Native repository permissions, branch protection, approvals and mergeability
  permit the selected merge method. Soda does not bypass an additional native
  human-review requirement or weaken it automatically.
- The candidate has not changed the factory's authoritative policy, credentials,
  grants or the CI workflow/verification definitions used to approve itself. Such
  changes require separate administrator adoption and leave the automatic path.

A changed candidate invalidates review and CI; a changed target base or required
policy requires reassessment and verification against the new state. An authority
or state check followed by an unchecked merge is insufficient: the native operation
must reject stale candidate authority/state. Failure or uncertainty pauses merging
instead of substituting a direct push. Confirm the native merge and its issue
outcome before marking completion, then reassess dependents using authoritative
forge state. An independently performed human merge is observed the same way; it
does not make unverified work a successful factory merge.

### Human intervention

Spaces shows the issue/PR, current stage, exact candidate, evidence, blocker or
failure, consumed/remaining limits and who can resolve it. Routine work advances
without approval prompts. Human actions follow the
[authority rules](../architecture/trust.md#factory-authority-boundary):

- **Clarify/resolve:** update accepted requirements or dependency evidence and
  trigger reassessment; this does not override other blockers or merge checks.
- **Pause/resume:** stop new factory actions, stop the assigned agents and preserve
  work. Resume revalidates authority and inputs and uses remaining allowances.
- **Cancel:** end the attempt's execution/publication/merge authority. Reopening
  the issue or receiving another event does not undo an explicit factory cancel.
- **Take over:** confirm factory execution has stopped and credential return or
  revocation has completed, then hand its candidate and unfinished work to the
  maintainer's own checkout and terminal under the
  [Project model](projects.md#accounts-and-checkout-ownership). Handing work back
  requires fresh assessment, review and checks; viewing CLI output never takes
  control, and a human never borrows the factory's provider credentials.
- **Retry:** after the previous execution is confirmed stopped and its resources
  accounted for, a write-authorized maintainer may explicitly start a new attempt
  under current policy. It receives that policy's fresh attempt time and correction
  allowances. All earlier consumption remains in the issue history and applicable
  rolling provider budget; a retry must queue if that budget is unavailable. The
  new allowance requires an explicit maintainer action.
- **Raise limits:** only the authority owning that limit may change it, subject
  to the other grants. Manual input cannot certify failed checks as passing.

Controller interruption never silently resumes an old conversation or creates a
duplicate agent. Stop/reconcile the recorded execution and surface intervention
before retry. If termination or credential return cannot be confirmed, keep that
execution/connection unavailable for replacement work and report cleanup separately
from the attempt outcome. Never repair by deleting the persistent environment.

## Implementation boundaries

The operating policy and [environment/session model](projects.md) are settled.
The remaining work is to verify and design their native boundaries: Fountain's
durable scoped authority and conditional merge, native dependency/clarification
representation, approved environment inputs and provisioning, and the
[shared execution integration](../architecture/overview.md#shared-project-execution)
including account/checkout permissions and broker attestation/credential return.
Limit enforcement, event delivery and interruption handling also need native evidence.
Those facts must be established before an implementation plan claims the complete
loop is feasible. They are not permission to restore legacy authentication or a
second issue, CI or execution system.

## System shape

```text
Fedora CoreOS host — operator administration only
├── Stock branded Cockpit, tailscaled
├── Soda factory control — readiness, execution, verification and native merge
├── Identity Broker — authorized AI CLI account custody
└── Podman
    ├── Fountain (maintained Forgejo fork) — identity, Git, collaboration and extension host
    ├── Soda extension service — environments and broker integration
    ├── Caddy — private HTTPS endpoints
    └── Repository development environments
        ├── Project-local accounts, homes, SSH
        ├── Shared installed tools and files
        └── Nested workloads and persistent service data
```

Forgejo, Soda and Caddy are separate containers. Forgejo is not a Podman pod.
Human and factory execution share the persistent repository Project under the
[environment model](projects.md#environment-relationships).

## Product roles

| Role | Authority |
| --- | --- |
| Host root / operator | Appliance administration, Cockpit, Soda operator setup |
| Forgejo site administrator | Forgejo administration; not automatically Soda operator |
| Repository owner | May create that repository's project environment |
| Project member | Project-local Linux account after explicit Join |
| Soda operator | The Forgejo user ID recorded at setup; runner and appliance settings |

Owning a repository does not grant host access. Site or organization administration
does not grant project root inside another team's environment.

## What Soda owns vs Forgejo

- **Forgejo** owns identity, passwords and factors, native sessions, permissions,
  Git, collaboration and administrator workflows.
- **Soda** owns environment associations, membership after confirmed Join,
  development-access public keys, extension operations and broker grants, local
  runner capacity and selected Tailnet enrollment policy for projects. Factory
  responsibilities include readiness and dependency handling, execution identity,
  agent coordination, verification, automatic merge under native repository policy,
  time and resource limits, cancellation, infrastructure audit and cleanup.

Soda does not invent a second password system, CI scheduler or forge-owned workflow
engine. Native Git authorization remains separate from project membership.

## Frontend model

Fountain supplies Forgejo's browser chrome, authentication and collaboration UI,
plus the native extension host. Soda contributes the Spaces page, persistent
Workspace panel and operator pages through its separately packaged extension.

There is no standalone Soda web frontend. Fountain owns its maintained fork and
extension SDK; Soda consumes that host contract. Missing native authentication or
authorization capabilities must be resolved through generic Fountain contracts,
as required by the [trust boundary](../architecture/trust.md#fountain-consumption-boundary).

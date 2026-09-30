# Soda OS product overview

Soda OS is **the operating system for software factories**. It takes repository
issues through readiness assessment, coding, pull requests, review and correction,
automatic merge, and reconsideration of dependent work. AI coding CLIs run in the
repository's development environment, with account access supplied by Soda's
Identity Broker. People observe the work and intervene through Spaces.

This document owns the target factory behavior. The [factory reference](../reference/factory.md)
describes the implemented operator commands; those commands do not define the
limits of this product contract. [Trust](../architecture/trust.md) owns authority
and credential boundaries. This contract does not select a scheduler, wire
protocol, container layout or implementation sequence.

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
| **Project** | A persistent shared development environment for one repository's authorized members. |
| **Project OS** | The userspace foundation inside a project (accounts, tools, persistence, workloads). |
| **Spaces** | Fountain-hosted view of repository environments and live human and factory CLI sessions. |
| **Runner** | Local CI capacity on the appliance, operated by the Soda operator. |
| **Tailnet** | Private connectivity for host and project reachability. |

Detailed ownership lives in [Projects](projects.md), [Spaces](spaces.md),
[Architecture](../architecture/overview.md) and [Project OS](../reference/project-os.md).

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
3. **Prepare the repository environment.** Soda provides the repository checkout,
   toolchain and required development services before starting coding work. It
   reuses a suitable existing container and prepares one when none exists. A
   separate checkout or worktree may provide the working state for an issue.
   Reuse must preserve unrelated dirty work, accounts and service data.
4. **Run the coding agent.** Soda starts the selected AI coding CLI with a prompt
   assembled from the issue, acceptance requirements and relevant repository
   context. The Identity Broker supplies the authorized provider connection.
   The agent's live CLI activity is visible through Spaces and attributable to its
   issue, repository and attempt.
5. **Publish the candidate.** Soda creates a pull request for the coding work and
   links it to the issue and attempt. The candidate identifies an exact revision;
   a successful CLI exit alone does not establish completion.
6. **Review and correct.** Soda starts another agent session in the same repository
   container to review the candidate and participate in fixing findings. Review
   is a distinct assignment from the original coding session. The division of
   fixes between reviewer and coding agent, checkout separation and review context
   are decisions to settle before implementation. Any changed candidate requires
   verification of its new revision.
7. **Merge automatically.** Soda performs the native forge merge when the required
   review and checks pass for the exact candidate and current repository policy
   permits it. Missing evidence, changed candidate state or unmet merge conditions
   prevent the merge. Human intervention is an exception path rather than a
   mandatory final step for every issue.
8. **Reconsider dependent work.** After confirmed merge and the corresponding issue
   outcome, Soda reevaluates issues that may now be unblocked and continues the
   same loop. Creating a PR or an agent claiming success does not count as a merge.

Limits bound execution, correction and resource use. Exhausted limits, unresolved
requirements, unavailable credentials or failed verification must produce an
understandable waiting, blocked or intervention outcome. The product does not
require exactly one repair or a fixed number of agents for every issue; the review
protocol and limits must be established explicitly. Duplicate observations must
not silently duplicate active work or erase consumed limits.

Closing or cancelling work withdraws further execution, publication and merge authority.
Recorded outcomes remain distinct from cleanup, and ending one agent must not
destroy a reused repository environment or unrelated work. Restart, retry and
resume behavior remain bounded design decisions; this contract does not require
automatic destructive recovery or conversation snapshots.

## Rules to settle before implementation

The loop above establishes the product behavior. Its owning feature and trust
contracts still need to settle:

- Who can enable repository automation and sponsor execution, provider use and
  merge authority, including withdrawal of that authority.
- How declared dependencies and unanswered requirements are represented and who
  or what can confirm their resolution.
- How the repository declares its correct environment, and how containers,
  checkouts, worktrees and concurrent issues share resources.
- Agent/provider selection, prompt context, the review/fix protocol and the checks
  that make an exact candidate eligible for automatic merge.
- Concurrency, usage and time limits; cancellation, interrupted execution and
  explicit human intervention.

These are product and architecture decisions, not tasks to implement by guessing
from the older operator workflow. The [operator walkthrough](../public/30-Use-Soda/15-software-factory.md)
remains guidance for that command interface.

## System shape

```text
Fedora CoreOS host — operator administration only
├── Stock branded Cockpit, tailscaled, CI runners
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
The relationship between factory execution and existing persistent Project roots
is governed by the environment decisions above and [Projects](projects.md).

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

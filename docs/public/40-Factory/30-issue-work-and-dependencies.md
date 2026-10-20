# Issue work and dependencies

Accepted native issues describe the work Factory may perform. Soda checks each issue's accepted objective, prerequisites, repository readiness and available authority, then creates a bounded execution plan when the work can proceed. Native issues and their dependency relationships remain the source of truth.

## From issue to executable plan

A maintainer with current code-write authority accepts an issue's exact requirements and dependency relationships, or explicitly adopts them when they originate elsewhere. Material changes to the objective, acceptance conditions or dependencies suspend affected work until the changed input is accepted and reassessed. Labels, comments, agent confidence and reopening an issue do not supply acceptance.

The Architect investigates the accepted issue and turns it into an executable task graph. The plan identifies work units, outcomes, responsible lanes, prerequisite edges, checks and expected outputs. It should cover the accepted outcome without adding unrelated scope. The controller checks the graph against the issue and repository context before Builder execution begins. A plan submission alone does not prove the tasks ran or that an output passed its checks.

The Builder works in the assigned checkout of the persistent Project, reports trusted checkpoints and submits task outputs for observation. A checkpoint identifies what was actually produced and the evidence available for it. The controller validates outputs against the planned task and checks before allowing dependent work to continue. Independent tasks can progress while another lane is blocked or fails; downstream tasks wait for the required result. A fresh assessment is required when task inputs, accepted requirements or prerequisite results change.

## Resolve prerequisites and research questions

Dependency edges name a required issue outcome. A code prerequisite is satisfied by the required change merged to its stated target branch; merely closing an issue does not establish that result. A non-code prerequisite requires a recorded authorized resolution and supporting evidence. The Factory Feed shows the blocker, its owner and the evidence needed to proceed. Agents may identify a missing prerequisite, but cannot waive or remove it.

Some questions can be answered through bounded research rather than a code change. Factory can create or reuse follow-up work within the existing authorized allowance, release or settle the blocked parent's execution slot, and return the accepted evidence to the parent for fresh context and reassessment. A negative result is evidence that an investigated answer was not found under stated conditions. An inconclusive result means the evidence is insufficient to decide. Neither result is silently converted into a satisfied prerequisite; the authorized owner decides whether the parent can proceed or its requirements should change.

## When work becomes stale or cannot proceed

An unreadable or incomplete prerequisite remains unresolved. A cycle remains blocked until an authorized maintainer changes the native dependency graph. If issue content or dependencies change during a run, Soda preserves the observed work but stops publication and continuation that rely on stale inputs. Reassessment determines which plans and evidence still apply. A failed task does not erase independent completed work, and it does not authorize a dependent task to use a missing result.

See [Review and merge](50-review-and-merge.md) for candidate evidence and delivery, and [Factory intervention](90-factory-intervention.md) when a run, allowance or external resource needs explicit action.

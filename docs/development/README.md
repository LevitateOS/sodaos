# Development

How to work on the Soda OS repository itself.

## Setup

1. Use the pinned Go, Bun and Python versions from manifests/locks.
2. From the repository root: `bun install --frozen-lockfile`.
3. Run focused checks for the code you change; see the command table in
   [AGENTS.md](../../AGENTS.md#commands).

## Where code belongs

| Concern | Document |
| --- | --- |
| Go package placement and style | [Go ownership](go.md) |
| Go file/role convention | [Go packages](go-packages.md) |
| Bun / TypeScript / Lit browser assets | [TypeScript](typescript.md), [Lit](lit.md) |
| Python tooling | [Python](python.md) |
| Forgejo presentation customization | [Forgejo](../reference/forgejo.md) |
| Cockpit branding | [Cockpit](cockpit.md) |

Product and security boundaries remain in [Architecture](../architecture/overview.md).
Do not treat package paths as the product vocabulary.

## Build, test and release

| Task | Document |
| --- | --- |
| Native validation journeys | [Testing](testing.md) |
| Support tool effects and candidates | [Native support](native-support.md) |
| Running the release pipeline | [Release workflow](release.md) |
| Local fixture access | [Local testing](../guides/local-testing.md) |

## Factory implementation planning

The [factory implementation plan](factory-implementation-plan.md) records the
current source baseline, owning contracts, exclusions and
[requirement deliverables](factory-implementation-plan.md#deliverable-map).
The [source change inventory](factory-implementation-plan.md#source-change-inventory)
locates coordinated code, caller, configuration and fixture changes. The
[factory interfaces](../architecture/factory-interfaces.md) specify target APIs,
records, transitions and process bindings. The
[dependency graph](factory-implementation-plan.md#dependency-graph) separates
parallel work from integration that requires demonstrated native capabilities;
the [early milestones](factory-implementation-plan.md#early-integration-milestones)
bound implementation/proof, pass scope and failure reconsideration for the uncertain
native integrations. The [implementation tasks](factory-implementation-plan.md#reviewable-implementation-tasks)
assign concrete outcomes, owners, prerequisites, changes, checks and completion
conditions. The [integration and qualification sequence](factory-implementation-plan.md#integration-cutover-and-qualification)
joins those outputs, replaces old callers and schedules feature availability,
the real factory demonstration and native qualification after their prerequisites.
Final plan review remains. The
[earlier extension transition plan](forgejo-extensions-plan.md) supplies evidence
and unfinished delivery work for reconciliation, not a competing execution queue.
Plans are removed after completion and absorption into the owning guides.

## Permissions

Destructive host operations, provider mutations, publication and fixture cleanup
require explicit task approval. Documentation is not a grant. Prefer asking over
inferring permission from historical handoff text.

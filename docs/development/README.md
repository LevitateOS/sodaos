# Development

Execution is stopped after the 18 October 2026 wind-down. [Remaining outcomes](factory-remaining-tasks.md#consolidated-remaining-outcomes--execution-stopped) is the single current roadmap/backlog; contract and evidence owners retain distinct technical detail. Reading a plan does not dispatch it.

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

## Source ownership and desired tree

The [source ownership index](ideal-filetree-plan/ideal-filetree-plan.md) describes
language, packages and placement. Update guidance affected by the current change.

## Build, test and release

| Task | Document |
| --- | --- |
| Native validation journeys | [Testing](testing.md) |
| Support tool effects and candidates | [Native support](native-support.md) |
| Running the release pipeline | [Release workflow](release.md) |
| Local fixture access | [Local testing](../guides/local-testing.md) |

## Remaining implementation and qualification

The [remaining-outcome catalog](factory-remaining-tasks.md) is the single execution
entrypoint: 53 Factory and 12 Canlang outcomes remain, with all
shared criteria counted once. Its six task packets and Canlang recipes supply
concrete source starts, prerequisite outputs, next slices and acceptance.
Implementation remains stopped until instructed.

The [workflow contracts](factory-workflow-contracts.md) own schemas, algorithms,
limits and decisive cases; [factory interfaces](../architecture/factory-interfaces.md)
own component APIs and records. Product, trust and [feed design](../design/factory-feed-and-ralph-integration.md)
own their respective behavior, authority and presentation decisions. Old parallel
plans and run schedules were absorbed and removed; Git retains their history.

## Canlang application integration

The [production runtime contract](canlang-soda-os-implementation-plan/production-workerd.md)
owns target/provider, storage, process-lifetime and image contracts and technical
substeps. Its [source allocation](canlang-soda-os-implementation-plan/proposed-tree.md)
distinguishes existing producers from proposed leaves. The
[application guide](../guides/canlang-applications.md) owns developer/operator
procedures; [testing](testing.md#canlang-qualification-boundaries) owns G1–G4
evidence distinctions. Historical runtime acceptance does not prove direct workerd
or the composed installed release; P32/R55 stays open after document absorption.

## Permissions

Destructive host operations, provider mutations, publication and fixture cleanup
require explicit task approval. Documentation is not a grant. Prefer asking over
inferring permission from historical handoff text.

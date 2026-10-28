# Development

**Required:** follow the [freshness policy](freshness.md). Resolve current upstream inputs before creating baselines; update stale locks and tools instead of retaining them for reproducibility.

[Factory execution map](factory-remaining-tasks.md#task-list) is the current backlog, including the complete MCP, issue, research, knowledge and feed designs. Work from the user’s current request; historical wind-down status does not block new development.

How to work on the Soda OS repository itself.

## Setup

1. Keep the installed Go, Rust and Bun tools current under the [freshness policy](freshness.md). Let builds and tests establish actual compatibility.
2. From the repository root: `bun install --frozen-lockfile`.
3. Run focused checks for the code you change; see the command table in
   [Testing](testing.md#source-checks).

## Where code belongs

| Concern | Document |
| --- | --- |
| Go package placement and style | [Go ownership](go.md) |
| Go file/role convention | [Go packages](go-packages.md) |
| Bun / TypeScript / Lit browser assets | [TypeScript](typescript.md), [Lit](lit.md) |
| Forgejo presentation customization | [Forgejo](../reference/forgejo.md) |
| Cockpit branding | [Cockpit](cockpit.md) |

Product and security boundaries remain in [Architecture](../architecture/overview.md).
Do not treat package paths as the product vocabulary.

## Source ownership

The [source ownership guide](ownership.md) describes
language, packages and placement. Update guidance affected by the current change.

## Build, test and release

| Task | Document |
| --- | --- |
| Native validation journeys | [Testing](testing.md) |
| Banning unit tests per language | [Banning unit tests](no-unit-tests.md) |
| Why unit tests are forbidden | [Why unit tests are forbidden](why-no-unit-tests.md) |
| Support tool effects and candidates | [Native support](native-support.md) |
| Running the release pipeline | [Release workflow](release.md) |
| RC6 installed results, failures and handoff | [25 October qualification record](rc6-qualification-2026-10-25.md) |
| Production gaps after the 28 October founder demo | [Production gaps](production-gaps.md) |
| Updating the retained local VM's front end | [Front-end VM update](frontend-vm-update.md) |
| Local fixture access | [Local testing](../guides/local-testing.md) |

## Remaining implementation and qualification

The [remaining-outcome catalog](factory-remaining-tasks.md) is the single execution
entrypoint: 53 Factory and 12 Canlang outcomes remain, with all
shared criteria counted once. Its six task packets and Canlang recipes supply
concrete source starts, prerequisite outputs, next slices and acceptance.
Take the smallest useful slice covered by the current request.

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

## Public handbook refresh

The [public handbook cutover](public-handbook/ideal-filetree-plan.md) records the
implemented six-section, 41-page handbook, its source evidence and page
boundaries. The [handbook remaining tasks](public-handbook/cutover.md) owns the
current priorities, all-page allocation, examples, captures and website handoff.
Its [current todo status](public-handbook/cutover.md#what-can-proceed-now)
separates ready source work, partial preparation and tasks awaiting named inputs.
Public copy describes the intended finished product; implementation status
belongs in the existing product backlogs.

## Permissions

Destructive host operations, provider mutations, publication and fixture cleanup
follow the current task’s authorization. Reuse authorization already given; ask
only when an action falls outside it. Documentation itself grants no permission.

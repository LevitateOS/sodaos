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

## Permissions

Destructive host operations, provider mutations, publication and fixture cleanup
require explicit task approval. Documentation is not a grant. Prefer asking over
inferring permission from historical handoff text.

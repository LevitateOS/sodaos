# Tracked-file inventory index

## Selected integration inventory (2026-10-16)

Source `92069938` has **2,907 tracked paths**. The [installer ledger](../maps/release-docker-current.md#installer-integration-scope)
accounts for its 18-path delta against `730fc205`, with the three added leaves in
system, lib and tests inventories. The earlier 76-path Docker reconciliation
remains at `76daa366`; intervening components-v4 and thirteen newer canonical
commits remain pending. Neither the membership count nor these scoped rows
renew the historical whole-tree/body/review coverage below.

Each inventoried path has one row per scoped source duty; older inventory gaps
remain explicit in the [coverage index](../README.md).

Current tracked path count at `9a184b7f`: **2,629**. Root counts below describe the current tracked tree; detailed named-duty/body selectors remain pinned to `519b76bd`.

Delta navigation at `deb79186` (2026-10-12): **2,772** tracked paths. The
`299e3d20..93fb3513` feature delta (93 added, 80 modified, 0 deleted outside
this plan) plus the incremental `93fb3513..deb79186` upkeep (1 added, 24
modified) are fully inventoried with current responsibility rows; 16 tracked
paths added between `9a184b7f` and `299e3d20` and untouched by the delta still
await rows and are named in the [coverage index](../README.md). The structural
tree pin, body-census pin and review pins below retain their recorded scope.

The source-root grouping describes current paths rather than target packages.
Former navigation filenames remain stable: `native-packages.md` covers `lib/`,
`appliance.md` covers `system/`, and the former Project OS page redirects to its
current inventory. No former-root rows count twice.

| Current root | Paths | Inventory |
| --- | ---: | --- |
| `(root)` | 20 | [root-files.md](root-files.md) |
| `.agents` | 1 | [agent-plans.md](agent-plans.md) |
| `.githooks` | 1 | [git-hooks.md](git-hooks.md) |
| `assets` | 175 | [assets.md](assets.md) |
| `cmd` | 354 | [cmd.md](cmd.md) |
| `docs` | 319 | [docs.md](docs.md) |
| `frontend` | 357 | [frontend.md](frontend.md) |
| `internal` | 615 | [backend.md](backend.md) |
| `lib` | 428 | [native-packages.md](native-packages.md) |
| `scripts` | 81 | [scripts.md](scripts.md) |
| `system` | 68 | [appliance.md](appliance.md) |
| `tests` | 156 | [tests.md](tests.md) |
| `tools` | 197 | [tools.md](tools.md) |

Generated/captured/third-party artifacts have producer/consumer dispositions;
their presence does not imply authored-code, rendering or native-execution proof.

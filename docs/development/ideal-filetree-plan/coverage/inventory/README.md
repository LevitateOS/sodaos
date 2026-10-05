# Tracked-file inventory index

### Every tracked file

Each row records the current file, its kind and lifecycle, and its candidate slice ownership. A single-owner file belongs to that slice throughout; binary inputs have file-level attribution. The linked interval maps give the deeper resolution for mixed files. References and line intervals are for the recorded commit; live working-tree line numbers can drift.


Snapshot, lifecycle definitions and limits: [coverage index](../README.md).
The current-root grouping describes recorded source paths, not the desired
product architecture. Each path occurs once across the following inventories.

| Source root at snapshot | Inventory |
| --- | --- |
| `(root files)` | [root-files](root-files.md) |
| `.agents` | [agent-plans](agent-plans.md) |
| `.githooks` | [git-hooks](git-hooks.md) |
| `appliance` | [appliance](appliance.md) |
| `assets` | [assets](assets.md) |
| `cmd` | [cmd](cmd.md) |
| `docs` | [docs](docs.md) |
| `factory-os` | [factory-os](factory-os.md) |
| `frontend` | [frontend](frontend.md) |
| `internal` | [backend](backend.md) |
| `project-os` | [project-os](project-os.md) |
| `rust` | [native-packages](native-packages.md) |
| `scripts` | [scripts](scripts.md) |
| `tests` | [tests](tests.md) |
| `tools` | [tools](tools.md) |

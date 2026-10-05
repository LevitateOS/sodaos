# Slice catalog index

## Review slice catalog

Catalog snapshot: **2026-10-05**, committed source `0d8d3b8ebb5a7df36759685529f27f417d96fac8`. These **80 candidate slices** describe existing responsibilities and current evidence. They are review units that can span frontend, Go, Rust, SQL and tests; a slice does not imply a package, service, process or sidecar. References exclude later working-tree changes. The complete proposed tree retains the older structural baseline recorded under [source coverage](../package-ownership.md#source-coverage), with the obsolete-leaf corrections and latest affected-file upkeep recorded in the coverage ledger.

The accumulated delta after `26d420f2` has been reconciled: factory settlement,
native host stop handling, broker closure regressions and native fixture logging,
plus the plan's documentation split and incoming links. Unchanged source and
cards reuse their recorded inspections; individual evidence-status fields keep
their original scope. This checkpoint adds no slice and no runtime proof.

Purpose and authority describe the inspected contract and implementation; they do not approve an intended model or judge code validity. Owned data names the responsibility to review, including shared records where the boundary remains unclear. Source links are current paths at the catalog snapshot, not proposed destinations. Dependencies include other slices and existing native services.

Tests list representative assertions or verification drivers and state their scope. No tests, builds, native operations or installed journeys were executed for this pass. References to predecessor helpers or retired source do not retain those implementations in the ideal target. Every tracked file and the distinct responsibilities in mixed files are accounted for in [complete source-to-slice coverage](../coverage/README.md#complete-source-to-slice-coverage). Establishing each slice’s intended model and reviewing implementation validity remain the next stages.

| Capability group | Slices |
| --- | ---: |
| [Projects](projects.md#projects) | 12 |
| [Identity brokering](identity-brokering.md#identity-brokering) | 10 |
| [Factory coordination](factory-coordination.md#factory-coordination) | 12 |
| [Spaces and terminals](spaces-and-terminals.md#spaces-and-terminals) | 6 |
| [Forgejo integration](forgejo-integration.md#forgejo-integration) | 9 |
| [Networking](networking.md#networking) | 7 |
| [Operator administration](operator-administration.md#operator-administration) | 7 |
| [Release and installation](release-and-installation.md#release-and-installation) | 11 |
| [Shared supporting slices](shared-supporting-slices.md#shared-supporting-slices) | 6 |

Refresh affected slice records during the existing post-merge upkeep: purpose, entrypoints, data and authority boundaries, dependencies, source files and test scope. Preserve unresolved questions until the owner establishes the intended model. Advance the catalog and coverage snapshot only to source actually inspected; they do not advance complete structural reconciliation or imply fresh behavioral verification.

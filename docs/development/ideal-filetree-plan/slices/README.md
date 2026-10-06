# Slice catalog index

R02 note @HEAD `cb221d9c`: source citations below are audit-pinned to the
`0d8d3b8e` snapshot (refs exclude later tree changes, per the catalog
statement). ~400 citations name pre-move paths (`rust/soda-identity`,
`rust/identity-providers`, `rust/soda-release-*`, `rust/soda-host`,
`rust/soda-project-*`, `appliance/`, `project-os/`) with audit line
anchors; they are intentionally NOT path-substituted, because splits
shifted spans (see STALE banners in coverage maps). Current locations for
moved files live in the coverage inventory/maps R02 notes.

## Review slice catalog

Catalog snapshot: **2026-10-05**, committed source `0d8d3b8ebb5a7df36759685529f27f417d96fac8`. These **80 candidate slices** describe existing responsibilities and current evidence. They are review units that can span frontend, Go, Rust, SQL and tests; a slice does not imply a package, service, process or sidecar. References exclude later working-tree changes. The complete proposed tree retains the older structural baseline recorded under [source coverage](../package-ownership.md#source-coverage), with the obsolete-leaf corrections and latest affected-file upkeep recorded in the coverage ledger.

The accumulated delta after `26d420f2` has been reconciled: factory settlement,
native host stop handling, broker closure regressions and native fixture logging,
plus the plan's documentation split and incoming links. Unchanged source and
cards reuse their recorded inspections; individual evidence-status fields keep
their original scope. This checkpoint adds no slice and no runtime proof.

Purpose and authority describe the inspected contract and implementation; they do not approve an intended model or judge code validity. Owned data names the responsibility to review, including shared records where the boundary remains unclear. Source links are current paths at the catalog snapshot, not proposed destinations. Dependencies include other slices and existing native services.

Tests list representative assertions or verification drivers and state their scope. No tests, builds, native operations or installed journeys were executed for this pass. References to predecessor helpers or retired source do not retain those implementations in the ideal target. Every tracked file and the distinct responsibilities in mixed files are accounted for in [complete source-to-slice coverage](../coverage/README.md#complete-source-to-slice-coverage). The later [completed source-audit matrix](../reviews/README.md#completed-source-audit-coverage) records intended models, validity findings and independently challenged exact targets for all 80 slices at f7e9cf9d. This catalog snapshot retains its original structural scope; no runtime verification follows.

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

## Validity review records

Use the [shared review format](../review-format.md) for each slice. Establish its
intended model as part of the audit and record source coverage, findings,
behavioral evidence and correction readiness separately. A completed inventory
does not imply a completed review.

When a slice review begins, create its single record at `reviews/<slice-id>.md`
relative to the plan folder and link it from that slice card. Add the actual
record path to the proposed tree and appropriate documentation coverage. The format itself does not start a review. All 80 actual source-review records
now exist and their completion/evidence limits are linked in the
[review index](../reviews/README.md).

The [assignment map](../review-assignments.md#one-primary-per-slice) gives every
slice one primary and a different worker to challenge consequential conclusions.
Its [collaboration rules](../review-assignments.md#shared-boundary-exchanges)
require affected owners to exchange evidence and preserve unresolved boundaries.
The coordinator owns shared coverage and tree updates; assignment is not audit
execution or product decision authority.

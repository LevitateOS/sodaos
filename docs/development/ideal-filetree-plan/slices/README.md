# Slice catalog index

## First-principles refresh (2026-10-13)

The current request covers SodaOS and its Forgejo integration: list the
subsystems, split them into slices, establish each purpose and its necessary
constraints, compare the implementation with that model, plan improvements and
start implementing. The nine groups and 80 stable slice IDs below cover the
existing application, platform, delivery and development responsibilities.
A subsystem is a capability boundary; its slices can span languages and processes.

The refresh starts from clean Soda commit
`fde19669440455a77e4c81c4f08ccc60a72276eb`, tree
`17b232afb7c4202b3e359cf20ee4f623e23d063e` (2,804 tracked paths), and the clean
local Forgejo SDK checkout `124907625d984755bb685168d0daddf8ef1cb36e`.
The first-principles tables in each group record current inspected owners,
purpose, constraints, assessment, desired difference and ordered next work.
Historical cards and full review records below retain their original source
and evidence scope. This refresh does not claim a new line-by-line review of
all 2,804 paths or qualify an installed appliance.

| Subsystem | Slices | Main code owners | Purpose |
| --- | --- | --- | --- |
| [Projects](projects.md#first-principles-refresh-2026-10-13) | P01–P12 (12) | `internal/project`, Project API/store operations, `lib/host/src`, `system/project` | Give authorized humans and factory roles persistent repository environments with distinct accounts and lifetimes. |
| [Identity brokering](identity-brokering.md#first-principles-refresh-2026-10-13) | I01–I10 (10) | `cmd/soda-identity`, `internal/identity`, Muse helpers and host bindings | Keep provider credentials private while lending bounded, revocable execution authority. |
| [Factory coordination](factory-coordination.md#first-principles-refresh-2026-10-13) | F01–F12 (12) | `internal/factory/control`, `internal/factory`, factory Store/API operations, `cmd/soda-factory` | Advance authorized work through readiness, execution, verification and policy-bound merge without duplicate effects or erased provider usage. |
| [Spaces and terminals](spaces-and-terminals.md#first-principles-refresh-2026-10-13) | S01–S06 (6) | `frontend/spaces`, terminal API/host operations, `cmd/soda-project-terminal` | Let people observe and control their work while preserving session lifetime across browser navigation. |
| [Forgejo integration](forgejo-integration.md#first-principles-refresh-2026-10-13) | G01–G09 (9) | `cmd/soda-extension`, `internal/forgejo`, `internal/web/auth`, `frontend/forgejo` | Integrate Soda with native identity, collaboration, evidence and conditional mutations. |
| [Networking](networking.md#first-principles-refresh-2026-10-13) | N01–N07 (7) | `internal/tailnet`, host Tailnet operations, `cmd/soda-tailnet`, proxy/service definitions | Establish truthful private reachability without conflating enrollment, routing and authority. |
| [Operator administration](operator-administration.md#first-principles-refresh-2026-10-13) | O01–O07 (7) | Setup/activation/database/Forgejo recovery commands, host services | Bootstrap and maintain the appliance while preserving existing state and explicit operator authority. |
| [Release and installation](release-and-installation.md#first-principles-refresh-2026-10-13) | D01–D11 (11) | `lib/soda-release-*`, `lib/release-inputs`, installer/import commands, release/acceptance tools | Produce, verify, distribute and install exact artifacts with separate build, release and disk-write authority. |
| [Shared support](shared-supporting-slices.md#first-principles-refresh-2026-10-13) | H01–H06 (6) | `lib/unix-http`, `lib/wire-time`, `internal/store`, parsing/config/lock helpers, assets and checks | Supply bounded mechanics and honest evidence while leaving domain decisions with their owners. |
| **Total** | **80** | | |

“First principles” here means the necessary conditions for the purpose to be
true: the right authority, one source of truth, explicit resource ownership,
bounded work and observable completion. It does not mean replacing native
libraries or already-correct code. Group verdicts are scoped source assessments;
uncertain native behavior remains a verification task. The selected first changes
and their checks are in the [implementation tasks](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-tasks.md#first-principles-implementation-2026-10-13).

Keep the completion dimensions separate:

| Dimension | Scope of this refresh |
| --- | --- |
| Responsibility coverage | All 80 slice purposes, principles and next steps are recorded. Current named owners/callers were inspected; whole-file responsibility coverage retains its earlier baseline. |
| Established workflows | Selected current boundaries were retraced, including newer factory planning/recovery/retry duties. Full native and installed workflows remain separate evidence work. |
| Independent challenge | Each lane's consequential scoped conclusions and selected corrections received another lane's review; dispositions and limits are recorded in the group cards. |
| Exact allocation | Selected corrections name existing defining source and test files in the task list. All other slices retain their current owners; no new package/process or complete ideal-tree cutover is implied. |

## Historical catalog evidence

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

The [guidance register](../review-assignments.md#guidance-conflicts-and-controlling-decisions)
preserves controlling owner decisions. These slice maps retain historical review
coverage, not worker assignments. The [remaining outcomes](../../factory-remaining-tasks.md)
own current missing coverage and execution.
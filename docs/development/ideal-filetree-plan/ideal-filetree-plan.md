# SodaOS ideal file tree plan

This is the living plan for reorganizing the entire SodaOS repository by
responsibility and decomposing oversized implementations into concrete concern
files. It covers every tracked source, test, document, asset, manifest, hidden
file, and system definition. The proposed paths in the linked tree do not
implement a move or
change product behavior. Existing ownership and trust boundaries remain the
starting point for the refactor.

Implementation is deferred until time is available. Keeping this plan current
is a separate, smaller task: update it after every merge, including merges that
do not change the proposed structure. This is the single forward-looking plan;
revise it in place rather than create a replacement plan or accumulate a merge
diary. Maintaining it does not authorize executing the refactor.

Last maintained: **2026-10-05**. The latest coverage pass accounts for all
**1,597 tracked paths** and their mixed responsibilities across **80 candidate
review slices**, at `26d420f28cb432bac2d69a177676fed3d1ee43c3`. The complete
proposed tree, package counts and historical decomposition use the structural
baseline `d7e565aa1019753997a99fd430ba103d7a472b48`, corrected here to exclude
11 positively evidenced obsolete proposed leaves. The plan folder layout is
also reflected in the proposed tree; this documentation split does not advance
either source review baseline. Complete structural
reconciliation of later changes remains pending. Pending source cutovers remain
explicit; proposed paths are not landed code.

Pending source reconciliation: the checkout advanced during this documentation
split to `c659c4ea5fefc85d7abc4a914117df4664e39b91`. The delta after the recorded
coverage snapshot changes the five paths below, including one new test file.
Their affected slice cards, responsibility maps and structural proposals still
need an upkeep pass; this split does not establish their code validity.

- [internal/factory/control/settle.go](../../../internal/factory/control/settle.go)
- [internal/factory/control/settle_test.go](../../../internal/factory/control/settle_test.go) — added after the snapshot
- [internal/factory/control/st15_demo_native_test.go](../../../internal/factory/control/st15_demo_native_test.go)
- [rust/soda-host/src/pfactory.rs](../../../rust/soda-host/src/pfactory.rs)
- [rust/soda-identity/tests/broker.rs](../../../rust/soda-identity/tests/broker.rs)

Decided language policy (owner): network-facing servers stay Go, system and
privileged applications are Rust, Python is eliminated from the tracked tree.
Open language options from earlier revisions are closed: the Rust host, installer,
import, acceptance and release crates are the retained owners, and their Go or
Python predecessors retire at cutover. The proposed tree is the
post-cutover target: it contains no pre-port implementation, even where the
cutover commit is still pending. Pending retirements are called out as
decided-pending with the PR or cutover that executes them.

At the reconciled structural baseline, the source has 30 Cargo packages under
`rust/`. The recommendation merges
the three release-asset packages into one and folds the sole-consumer identity
providers into the broker; the Rust host library is retained as the daemon
foundation. That leaves 27 Cargo packages. Its daemon entrypoint belongs to the
same `soda-host` Cargo package, with a `[[bin]]` path under `cmd/soda-host`; it
does not add a package or a second service. The four
landed release crates (`soda-release-build`, `soda-release-deliver`,
`soda-release-image`, `soda-release-tools`) are placed 1:1 under `lib/`; whether
they consolidate further, and where their duplicated build helpers dedupe, is an
open owner decision in the pending filetree layout. The tree places
installed commands in `cmd/`, named implementation libraries in `lib/`, support
tools in `tools/`, authored browser surfaces in `frontend/`, and image/service
definitions in `system/`. Canonical assets stay in `assets/`; the website handbook
source stays at `docs/public/`.


The plan is maintained as one artifact in this folder. Each section has one
canonical file; the coverage indexes link the deeper source maps. The original
single-file plan has been split without executing the proposed code refactor.

| Plan section | Owns |
| --- | --- |
| [Package ownership](package-ownership.md) | Structural source baseline, package consolidation and existing process topology |
| [Port assessment](port-assessment.md) | Retained language owners, integration concerns and predecessor cutovers |
| [Placement](placement.md) | File granularity, current Go packages and placement rules |
| [Complete proposed tree](proposed-tree.md) | The entire desired repository tree, including this plan folder |
| [Decomposition index](decomposition/README.md) | Historical oversized-file concern reviews, grouped by source component |
| [Integration](integration.md) | Retained documents/data, wiring changes and review/implementation limits |
| [Slice catalog](slices/README.md) | All 80 candidate review slices in nine capability groups |
| [Coverage index](coverage/README.md) | Snapshot counts, retirement/generated dispositions and unresolved boundaries |
| [Tracked-file inventories](coverage/inventory/README.md) | Every recorded tracked path, with links to its slices and interval map |
| [Responsibility maps](coverage/maps/README.md) | Exact responsibility intervals inside mixed and oversized source files |
| [Maintenance](maintenance.md) | Required post-merge reconciliation and deferred execution workflow |

Start the validity review from the slice catalog: establish the intended model
for each slice before comparing implementation assumptions and tests. Use the
coverage inventories and responsibility maps to check that the review reaches
every file and every distinct responsibility. Candidate slices and proposed
packages remain separate concepts.

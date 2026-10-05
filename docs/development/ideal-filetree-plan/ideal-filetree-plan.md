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
**1,719 tracked paths** and their mixed responsibilities across **80 candidate
review slices**, at `0d8d3b8ebb5a7df36759685529f27f417d96fac8`. The complete
proposed tree, package counts and historical decomposition use the structural
baseline `d7e565aa1019753997a99fd430ba103d7a472b48`, corrected here to exclude
11 positively evidenced obsolete proposed leaves. The plan folder layout is
also reflected in the proposed tree. The five-file source delta and the
documentation split are reconciled in the catalog and coverage ledger. Complete structural
reconciliation of later changes remains pending. Pending source cutovers remain
explicit; proposed paths are not landed code.

The separately [pinned review input baseline](review-baseline.md) records source
`f7e9cf9db616f93de351dd44c9bb7456feb608b9` plus the uncommitted working guidance.
Its committed delta from `0d8d3b8e` modifies 22 Markdown files only. The all-slice source audit is now complete at that pin across **80 reviews**:
responsibility coverage, established workflow analysis, independent challenge
and explicit target allocation. The [completion matrix](reviews/README.md#completed-source-audit-coverage)
links each actual record and its evidence. Open defects and named owner/producer
questions remain visible; their dependent corrections stay pending. No tests,
builds or native/installed qualification were performed. Audit completion does
not advance earlier structural evidence or declare the code correct.

Latest upkeep reconciles the complete delta from `26d420f2` to `0d8d3b8e`.
The five source/test paths below have current responsibility maps, test-scope
descriptions and affected concern allocations. The new settlement test has an
inventory entry and target leaf. The documentation split replaces one tracked
plan with 122 sections, all attributed to H06 documentation upkeep; the total
inventory therefore grows by 122 paths, not just the one new test. Unchanged
source coverage is reused after checking the committed delta. This is source
reconciliation, not a validity audit or new behavioral qualification.

- [internal/factory/control/settle.go](../../../internal/factory/control/settle.go)
- [internal/factory/control/settle_test.go](../../../internal/factory/control/settle_test.go) — new PostgreSQL regression scenarios with stub host/broker
- [internal/factory/control/st15_demo_native_test.go](../../../internal/factory/control/st15_demo_native_test.go)
- [rust/soda-host/src/pfactory.rs](../../../rust/soda-host/src/pfactory.rs)
- [rust/soda-identity/tests/broker.rs](../../../rust/soda-identity/tests/broker.rs)

## Retained Forgejo presentation

The owner also confirmed on 2026-10-05 that Soda's design deliberately overrides
**all Forgejo pages**. Preserve that full scope and design in the desired tree.
[GUIDANCE-08](review-assignments.md#guidance-conflicts-and-controlling-decisions)
and [G08](reviews/G08.md) record this controlling decision. The audit checks
correctness and native integration within it; override breadth is not grounds
for removal, stock-page replacement or redesign.

## Owner language policy

Decided language policy (owner): network-facing servers stay Go, system and
privileged applications are Rust, Python is eliminated from the tracked tree.
Open language options from earlier revisions are closed: the Rust host, installer,
import, acceptance and release crates are the retained owners, and their Go or
Python predecessors retire at cutover. The proposed tree is the
post-cutover target: it contains no pre-port implementation, even where the
cutover commit is still pending. Pending retirements are called out as
decided-pending with the PR or cutover that executes them.

Reviewers must check the [guidance conflicts and controlling decisions](review-assignments.md#guidance-conflicts-and-controlling-decisions)
before allocating languages. The policy above records an owner decision; the
old blanket Go instruction in AGENTS.md is superseded. Conflicting guide tables,
cutover status or proposals to change an established owner remain explicit
questions until reconciled. A specified correction cites the applicable decision,
names each Go/Rust owner and exact target, and never derives implementation
authority or landed status from this plan.

The pinned audit source has **32 Cargo packages** under `rust/`, including the
landed Project account and factory-role helper crates. The proposed tree has
**27 packages**: merge the three release-asset packages into one, fold identity
providers into the broker, and fold the two Project helpers into the existing
Project-terminal package. The helper consolidation is still a proposed package
change; their Rust ports and compiled build inputs already exist. The Rust host
already has its daemon binary and release compile wiring in source. Its proposed
entrypoint move to `cmd/soda-host` stays in the same `soda-host` package and
service. The four
landed release crates (`soda-release-build`, `soda-release-deliver`,
`soda-release-image`, `soda-release-tools`) retain their identities under `lib/`.
The selected pure OCI and delivered-content helpers consolidate into those
existing owners as specified in [port assessment](port-assessment.md); no further
crate merger is selected. The tree places
installed commands in `cmd/`, named implementation libraries in `lib/`, support
tools in `tools/`, authored browser surfaces in `frontend/`, and image/service
definitions in `system/`. Canonical assets stay in `assets/`; the website handbook
source stays at `docs/public/`.


## Plan sections

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
| [Review input baseline](review-baseline.md) | Exact source commit, committed delta scope and byte identities of uncommitted guidance |
| [Slice review format](review-format.md) | Intended models, review coverage, findings, evidence limits and explicit correction targets |
| [Review assignments and collaboration](review-assignments.md) | One primary and challenger per slice, shared-boundary exchanges and coordinated plan updates |
| [Slice review records](reviews/README.md) | Actual per-slice audit records and linked shared-boundary questions |
| [Coverage index](coverage/README.md) | Snapshot counts, retirement/generated dispositions and unresolved boundaries |
| [Tracked-file inventories](coverage/inventory/README.md) | Every recorded tracked path, with links to its slices and interval map |
| [Responsibility maps](coverage/maps/README.md) | Exact responsibility intervals inside mixed and oversized source files |
| [Maintenance](maintenance.md) | Required post-merge reconciliation and deferred execution workflow |

Start the validity review from the slice catalog: establish the intended model
for each slice using the [shared review format](review-format.md) before comparing
implementation assumptions and tests. Use the
coverage inventories and responsibility maps to check that the review reaches
every file and every distinct responsibility. Candidate slices and proposed
packages remain separate concepts.

Use the [completion criteria](review-format.md#completion-criteria) to report
responsibility coverage, complete workflow analysis, independent challenges and
explicit target allocations separately. Unresolved questions remain visible and
block their dependent implementation instructions; review completion never
authorizes executing the plan.

The review baseline, format and assignments are new H06 planning documents awaiting
inclusion in a committed source baseline. They are present in the proposed tree;
the pinned
1,719-path inventory continues to describe `0d8d3b8e`.

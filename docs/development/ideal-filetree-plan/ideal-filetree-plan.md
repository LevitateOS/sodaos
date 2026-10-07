# SodaOS ideal file tree plan

This is the living plan for reorganizing the entire SodaOS repository by
responsibility and decomposing oversized implementations into concrete concern
files. It covers every tracked source, test, document, asset, manifest, hidden
file, and system definition. The proposed paths in the linked tree do not
implement a move or
change product behavior. Existing ownership and trust boundaries remain the
starting point for the refactor.

Further implementation is deferred until selected. Keeping this plan current
is a separate, smaller task: update it after every merge, including merges that
do not change the proposed structure. This is the single forward-looking plan;
revise it in place rather than create a replacement plan or accumulate a merge
diary. Maintaining it does not authorize executing the refactor.

The [current task state](implementation-tasks.md#current-task-state-2026-10-07)
is reconciled at canonical HEAD `04286c48`, application source `0b073439`, on
**2026-10-07**, preserving the existing working plan edits.
The selected [library adoption](library-adoption.md) is largely source complete:
L00–L09, L11–L15 and L18, plus L10.N3. Remaining work includes L10.N4's provider
transport fit, L16.G aggregate secret collection and CFG01 parser fit; optional
L16 matcher adoption is deferred. The [finding allocation](execution-findings.md)
assigns all 76 challenged dispositions to existing subtasks with exact owner,
scope, prerequisites and acceptance; older slice findings remain in their parent
records. Correctness precedes costly boundary decisions and settled caller cuts.
Current responsibility coverage is reconciled; full R02.targets desired-tree
regeneration follows settled replacement boundaries. Broader R03/R04 verification
retains its separate gates.
The verified adoption source `d7eca882` differs from this checkout only in
Markdown, so its recorded checks retain their original scope. No fresh runtime
verification was performed for this documentation refresh.

Further unrelated restructuring dispatch remains parked. Completed work and
parked checkpoint branches remain preserved; superseded generic-engine splits
must consume the current library adapters if an application seam is selected.
The task list and lane schedule remain the execution plan. This scoped update
does not regenerate the complete tree or advance historical audit coverage.
The non-normative
[investigation](../../research/library-reuse-investigation.md) supports the
selected direction; product/trust owners remain authoritative.

Last maintained: **2026-10-07**. The task refresh also accounts for already-landed
B07 publication/merge placement, C06 stdin-delivery and C07 enrollment corrections,
rootfs-server placement and H04 `root_chain` retirement. The scoped L18 retirement maps and
A34/B27/C41 assessment remain complete at their recorded scope. The execution
schedule provides
[parallel implementation lanes](implementation-lanes.md) and a
[dependency-ordered task list](implementation-tasks.md), prepared against
`de65ff68`. That checkout's delta from the audit source is documentation only;
the scheduling update itself does not advance the audit or historical structural
baseline. Implementation started under run 20261005; checked task rows record
landed work and remaining unchecked rows retain their specific source/native scope.
Audit/structural baselines above are preserved historical identities, not current source. The latest coverage pass accounts for all
**1,719 tracked paths** and their mixed responsibilities across **80 candidate
review slices**, at `0d8d3b8ebb5a7df36759685529f27f417d96fac8`. The complete
proposed tree, package counts and historical decomposition use the structural
baseline `d7e565aa1019753997a99fd430ba103d7a472b48`, corrected here to exclude
11 positively evidenced obsolete proposed leaves. The plan folder layout is
also reflected in the proposed tree. The five-file source delta and the
documentation split are reconciled in the catalog and coverage ledger.
The task list owns current M/C/V status. Current source inventory/maps are
reconciled at their newer recorded snapshot; complete target reconciliation
remains pending. R02.targets must refresh the desired-tree allocation, slice
cards and decomposition after replacement boundaries settle, preserving current
source coverage and historical counts. The appliance map/inventory has a separately reconciled
scope at `3036bb47`; it does not refresh the whole catalog. Parked checkpoints
are assessed separately in the library chapter and excluded from merged coverage.
The task list is current at the scoped source/status level; the full structural
plan is not yet current. Proposed paths do not establish landed code.

The separately [pinned review input baseline](review-baseline.md) records source
`f7e9cf9db616f93de351dd44c9bb7456feb608b9` plus the uncommitted working guidance.
Its committed delta from `0d8d3b8e` modifies 22 Markdown files only. The all-slice source audit is now complete at that pin across **80 reviews**:
responsibility coverage, established workflow analysis, independent challenge
and explicit target allocation. The [completion matrix](reviews/README.md#completed-source-audit-coverage)
links each actual record and its evidence. Open defects and named owner/producer
questions remain visible; their dependent corrections stay pending. No tests,
builds or native/installed qualification were performed. Audit completion does
not advance earlier structural evidence or declare the code correct.

The historical catalog upkeep reconciles the delta from `26d420f2` to `0d8d3b8e`.
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
- Historical `rust/soda-host/src/pfactory.rs`; current host Factory owner is
  [lib/host/src/factory/mod.rs](../../../lib/host/src/factory/mod.rs).
- Historical `rust/soda-identity/tests/broker.rs`; current broker subject is
  [cmd/soda-identity/tests/broker.rs](../../../cmd/soda-identity/tests/broker.rs).

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
Project-terminal package. The helper package fold is canonical at the investigation pin; preserve its
completed placement and installed executable identities. Remaining behavioral
findings stay separate from that completed consolidation. The Rust host
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
| [Implementation lanes](implementation-lanes.md) | Two workers plus coordinator by default, optional third worker, exclusive file ownership and verification order |
| [Library adoption](library-adoption.md) | Selected finding allocations, bounded adoption packets, prerequisites and parked-work dispositions |
| [Implementation tasks](implementation-tasks.md) | 27 primary work packets covering all 80 slices, shared extraction and precise dependent gates |
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

The two implementation scheduling documents are new H06 planning additions
outside the older coverage snapshot and enter the proposed tree immediately.
The review inputs/records are already committed at the scheduling checkout;
the pinned 1,719-path inventory continues to describe `0d8d3b8e` rather than
claiming a full new structural reconciliation.

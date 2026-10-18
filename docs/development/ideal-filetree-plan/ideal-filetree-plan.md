# Soda source ownership and desired tree

This document owns language policy and the index of structural responsibility,
placement and desired-tree contracts. The [remaining-outcome catalog](../factory-remaining-tasks.md)
owns execution. Filetree R02 retains incomplete source/duty coverage beyond the
historical pins; this consolidation is not a new body audit or native proof.

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
already has its daemon binary and release compile wiring in source. Retain its
entrypoint at `lib/host/src/main.rs` in the existing `soda-host` package and
service. The earlier `cmd/soda-host` placement proposal is superseded by the
[host ownership assessment](package-ownership.md#recommended-package-changes).
This source placement correction preserves the decided Go/Rust/Python policy.
The four
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
| [Library adoption](library-adoption.md) | Library boundary contracts and scoped adoption evidence |
| [Slice catalog](slices/README.md) | All 80 candidate review slices in nine capability groups |
| [Review input baseline](review-baseline.md) | Exact source commit, committed delta scope and byte identities of uncommitted guidance |
| [Slice review format](review-format.md) | Intended models, review coverage, findings, evidence limits and explicit correction targets |
| [Review assignments and collaboration](review-assignments.md) | Owner guidance decisions and provenance |
| [Slice review records](reviews/README.md) | Actual per-slice audit records and linked shared-boundary questions |
| [Coverage index](coverage/README.md) | Snapshot counts, retirement/generated dispositions and unresolved boundaries |
| [Tracked-file inventories](coverage/inventory/README.md) | Every recorded tracked path, with links to its slices and interval map |
| [Responsibility maps](coverage/maps/README.md) | Exact responsibility intervals inside mixed and oversized source files |
| [Maintenance](maintenance.md) | Proportionate upkeep of changed responsibilities and coverage limits |

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

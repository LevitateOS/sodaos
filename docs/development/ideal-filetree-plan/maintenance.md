# Maintaining the ideal file tree plan

## Keeping the plan current after every merge

The person or agent handling a merge owns the post-merge update of this plan as
part of that work. Update the plan after every merge into the maintained branch,
before considering the merge task complete. Pulling merged changes into this
checkout also requires reconciliation if this plan still records an older
baseline. A planning update never starts the deferred refactor.

1. Read the current HEAD, working-tree changes and this plan's last reconciled
   source. Compare the accumulated changes since that source, not only the last
   merge commit. Preserve unrelated work; if the baseline is unavailable,
   inventory the current source before claiming reconciliation.
2. Refresh the tracked file inventory, source-to-slice ledger and size flags.
   Map every added path and every new responsibility inside changed mixed files.
   Recheck named fields sharing SQL/struct lines, obsolete units and generated
   artifacts separately; unresolved owners or overlaps keep the map pending.
   Inspect changed source
   plus affected callers, tests, manifests and build/install payloads. Reuse the
   existing concern reviews for unchanged code; a full architectural audit or
   application build is not required for routine plan upkeep. Add a real concern
   review for every newly oversized code file before claiming zero unreviewed
   files. Keep pending-branch observations separate from merged-source coverage.
3. Update the complete desired tree, package ownership, port recommendations,
   affected concern notes, source references and coverage counts together.
   Account for additions, deletions, renamed files, changed large-file seams
   and any actual new process boundary. Do not infer sidecars from package splits.
4. Remove proposals already implemented or superseded and mark unsettled choices
   honestly. Retiring a port removes its conditional split targets; retaining it
   requires closing its documented import, state and delivery seams. Keep one
   current plan rather than an append-only log of merges. Omit dead or decided
   predecessor implementations from the primary target even before physical
   deletion; record the retirement and its real cutover dependencies separately.
   Check live data, mixed client/wire files, command callers and embedded programs
   before deleting a whole language/package subtree from the plan.
5. Record the structural, catalog and coverage source commits, maintenance date
   and actual review scopes separately. Do not advance one based on another.
   Preserve the initial full-review baseline; a delta review does not claim a
   new full architectural review or fresh runtime verification. Check source
   coverage, destinations, merge targets, links and filename/module conflicts.
   For a merge with no structural impact, still advance that baseline/date and
   state that the reviewed delta required no proposal changes.

Use the resulting commit recorded after a merge, squash or rebase, rather than
the pre-merge branch tip. If upkeep is incomplete, keep the previous reconciled
baseline and state which source changes still need review; never label the plan
current solely because a pull succeeded. A follow-up documentation change can
record the code merge it reconciles without creating an endless self-update
cycle for its own bookkeeping-only commit.

The maintained artifact is the Markdown plan folder, entered through
[ideal-filetree-plan.md](ideal-filetree-plan.md). Update the affected topic files,
slice cards, inventories and responsibility maps together; keep their indexes
and links current when adding, moving or removing a plan section.
Ignored scratch inventories and
renderers are optional analysis aids, not prerequisites for future maintainers.
Do not blindly regenerate from old split reports after source changes: re-read
the affected definitions and preserve later owner decisions in the affected
plan sections.

## Executing when time is available

Keep implementation deferred until the owner selects work that fits available
time. Before that work starts, reconcile this plan against current source and
resolve the affected ownership/language choices. Select a small, coherent
boundary including its real callers, manifests, tests and installation wiring;
do not begin the entire repository migration merely because the plan exists.
Replace proposed seams with implemented paths as work lands, remove obsolete
alternatives, and record build, source-check and native verification separately.
Completed choices belong in their owning guides; this folder remains the current
plan for outstanding restructuring while that work remains deferred.

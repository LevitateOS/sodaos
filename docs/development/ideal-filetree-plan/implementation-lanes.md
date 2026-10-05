# Parallel implementation schedule

This is the execution schedule for the existing desired tree and the 80-slice
audit, not a new architecture. The [task list](implementation-tasks.md) contains
27 primary work packets plus the shared extraction packet A00. Implementation
is deferred. Creating this schedule does not authorize source changes, tests,
builds, Git operations, deployment, provider calls or publication.

Prepared on **2026-10-06** against clean checkout
`de65ff684c29cf9510134d01917ec4e5a7afdcc3`. Its committed delta from the audit
source `f7e9cf9d` changes documentation only; unchanged source assessments are
reused at their recorded scope. Historical structural reconciliation is still
pending. Refresh the affected source and guidance before dispatching a packet.

The retained [tree](proposed-tree.md), [placement](placement.md),
[ownership](package-ownership.md), [review records](reviews/README.md) and
[responsibility maps](coverage/maps/README.md) remain the authorities for exact
proposed defining files and existing caller/test/build/install duties. This
schedule assigns work; it does not override their requirements or evidence limits.

## Capacity and lanes

Use **three GPT-6.1 Sol implementation workers and one coordinator/integrator**,
within the four available concurrent slots. Reuse reviewer knowledge where
helpful. Rotate implementation, challenge and verification through these slots;
do not assume extra simultaneous reviewers or launch all 80 slices separately.

| Lane | Primary packet queue | Responsibility |
| --- | --- | --- |
| A — Projects, identity and Spaces | A00–A08 | Close the 28 P/I/S slice allocations and their actual Go, Rust, browser and system handoffs. Own broker, host and Project-terminal implementation files. |
| B — Factory and Forgejo | B01–B08 | Close the 21 F/G slice allocations. Own Go dashboard/domain/Store/API/client/publication files and browser Spaces/Tailnet files. Preserve full Soda Forgejo design. |
| C — Platform, release and verification | C01–C11 | Close the 31 N/O/D/H slice allocations. Own release/install/setup/maintenance/tool/asset implementations, system definitions and Forgejo presentation files. Route native host and Go-store changes to A/B. |
| Coordinator | R00–R04 below | Dispatch ready tasks, apply shared root wiring during exclusive handoffs, reconcile boundary changes, arrange independent review and maintain the plan. |

Packet leads own completion across their slice duties; **the physical file owner
alone edits each file**. For example, C02 leads H01/H02 but sends host changes to
A and Go Store/client changes to B. A02 sends factory grant/Store changes to B.
The handoff is part of the packet, not another untracked implementation.

Queues are not fixed personnel assignments. After a packet releases its files,
an available worker can take another lane's ready packet through an explicit
whole-file handoff. Prefer the longest remaining dependency chain and critical
native assumptions; do not wait for every packet in a lane or wave to finish.

## File ownership and conflict avoidance

Work in the canonical checkout. Do not introduce worktrees or change Git state
through this schedule. At R00, record the literal current and target paths for
the selected packet from its review/map; directory patterns below identify the
default owner, not permission to edit every file under that directory.

| Current/target shared surface | Default physical writer | Handoff rule |
| --- | --- | --- |
| `rust/soda-host/**` → `lib/host/**`; host binary entrypoint; provider-neutral Factory terminal models/pumps | A | A owns whole current monoliths, module roots and their private/oracle fixtures. B/C submit exact native changes. Descendant files transfer only after defining extraction is integrated. |
| `rust/soda-identity/**`, `rust/identity-providers/**` → `cmd/soda-identity/**` | A | One Controller/State/Store/Tx and provider-support owner. Codex/Muse child work may split after construction/import/test ownership is fixed. |
| Project terminal/account/factory-role crates → `cmd/soda-project-terminal/**` | A | One guest-package module/fixture owner; preserve the three installed executable identities. |
| Go `cmd/`, `internal/` application/domain/Store/API/client files; `frontend/spaces/**`, `frontend/tailnet/**` | B | A/C send exact changes. B owns current `grants`, `dispatch`, `review_cycle`, `merge`, Store and API monoliths before releasing extracted leaves. |
| `internal/store/schema.go` and its Rust broker mirror | B defining schema; A mirror | Reconcile together under H02. Existing mirror/drift assertion remains; no new schema generator or independent authority. |
| `cmd/soda-forgejo-tailnet/**` and helper-exclusive `internal/forgejo/tailnet*.go` → same-host Rust helper | A during C04 port | Reserve these exact files from B first. Preserve all other Go Forgejo/domain/status clients; release compiler/install joins remain C/R01 duties. |
| Other Rust commands/crates, `rust/soda-json/**`, assets, Forgejo templates/locales/styles, system/image/rootfs definitions | C | A/B request primitive or payload changes; mixed setup/install/maintenance files have one current writer. |
| Go installed-probe/test tooling, `internal/acceptance/**`, cross-component build/installed tests and source scripts | C | Preserve real subjects. B/A provide production changes; current shared fixtures/helpers remain single definitions with named consumers. |
| `rust/soda-release-image/src/{sys,build}.rs` and other release crate composition roots | C | Coordinator takes a short exclusive whole-file handoff for shared compile/stage joins. C pauses edits to those files until the handoff returns. |
| Root workspace/manifests/locks, every crate-local `Cargo.toml` dependency/bin/test declaration, package scripts, TS includes, shared payload manifests and plan indexes | Coordinator | Workers submit package/bin/dependency/source/destination tuples. Apply a coherent batch once; A/C do not concurrently edit even private crate manifests, and workers do not regenerate shared locks. |

If a packet crosses an ownership row, name the recipient and exact change before
dispatch. A conflicting file stays queued; its worker can research, prepare an
unapplied proposal or implement a disjoint ready task. Do not use concurrent
hunk editing of a shared file as a substitute for ownership.

## Coordinator checklist

- [ ] **R00 — Dispatch preparation.** Refresh affected source/guidance and the
  selected findings' readiness. Record the scoped implementation/operational
  instruction, one lead, literal source/target file list, recipients, dependency
  results and actual smallest verification. Dependencies apply to the relevant
  M/C/V subtask's required boundary output, not completion of an entire packet.
  Existing normal-mode restrictions
  still apply; this planning command authorizes only these documents.
- [ ] **R01 — Integrate each coherent packet.** Complete module/import/manifest,
  compiler, payload, source-check and caller joins in the same reviewable change.
  Keep M (behavior-preserving changes), C (named correctness corrections) and V
  (verification) distinguishable. Preserve binary/service identities and direct
  defining owners; replace obsolete callers without compatibility facades.
- [ ] **R02 — Reconcile coverage and retirements.** Every original file and mixed
  responsibility must have its implemented defining destination or evidenced
  retirement. Preserve generated/data/license duties separately. Check all
  current Cargo members and Go packages, source-path tests and installed outputs;
  retire `rust/`, old system roots and proven predecessors only as their actual
  last callers/build/install/test duties close. Resolve historical structural
  drift rather than copying old deletion tables.
- [ ] **R03 — Integrated source verification.** On a stable agreed snapshot,
  run the applicable existing Go/Rust/Bun/source/architecture checks. Reuse a
  passing result only for unchanged exercised source. Repeat an affected check
  for new changes or unresolved failures, not automatically for every worker.
- [ ] **R04 — Selected native qualification.** After cheap boundary checks and
  producer/caller gates pass, qualify the selected matching candidate and actual
  user workflows under separately scoped operations. Keep source, build, stage,
  installed, provider and retained delivery evidence distinct. Signing, uploading,
  merging and publishing are separate operations, not automatic refactor steps.

## Start order and critical dependencies

These are future scheduling priorities, not started tasks or elapsed estimates.
The measured dependency chain and file availability should control reassignment.

1. After R00, start **A05 broker core**, **B01 authority/Store**, and **C08 release
   producer** in parallel, reserving their disjoint files. Run C01's bounded JSON
   correction before dependent malformed-input verification; promote C02 native
   drain/attachment checks and A07 PTY/relay checks before costly native runs.
   This is an overlapping queue, not a requirement that each lead edit four
   packets simultaneously.
2. Integrate A00 per monolith when its same-file corrections and defining seams
   are settled. Release extracted files to A01–A04/A06–A08, C03/C04 and native
   B03/B04 handoffs. Independent C06 maintenance and C07 enrollment work can fill
   free slots; neither waits for the entire host refactor.
3. B01/B02 plus actual Project/broker authority handoffs feed **B03 dispatch**;
   B03 registration/settlement and B07 adapter readiness feed **B04 review loop**;
   fresh assessment plus native reachability evidence feed **B05 completion**.
   B06 transport and B08 presentation can proceed independently on reserved files.
4. **C08 producer → C09 inputs/media/assets and C10 delivery → C11 qualification
   drivers → R03/R04** is the release integration chain. Pure primitive tests,
   operator commands, C09 independent input/asset work and browser checks can
   run earlier; only their actual compiler/staging joins wait for C08 readiness.
   C10 equivalence gates
   block duplicate retirement only, not unrelated delivery corrections.
5. When a worker waits for evidence or a shared file, select another ready M/C
   subtask. A block on SDK reachability, Muse invocation applicability or optional
   Tailnet semantics must not stall unrelated packets.

## Verification and wall-time rules

Use existing checks and fixtures. Do not build a scheduler, new test framework,
sidecar, broad compatibility mechanism or replacement service to execute this plan.

- First test the actual changed parser/HTTP/ABI/child/transaction subject and
  bounded failure path. A faithful port can preserve a bad test assumption.
  Real PostgreSQL tests use the existing explicitly disposable fixture when its
  operational scope is authorized; they do not reuse historical SQLite proof.
- Then check affected package/module/public import and explicit binary selectors.
  Seven asset binaries, acceptance driver/remote, Project helper binaries and
  host/helper outputs must retain real compile/install subjects and fixtures.
- Build Forgejo assets once per stable relevant input and reuse prepared frontend
  and Forgejo tests where their existing scripts permit it. A move or split gets
  existing coverage; add a regression only for a named established contract/bug.
- Keep one owner for resource-heavy builds/native fixtures. Do not run competing
  full release/VM jobs or provider operations. Put caches/artifacts on `/home`
  under the owning task's existing artifact conventions, outside source scans;
  the root filesystem restriction remains in force.
- Run a full candidate build only after affected cheap tests, binary staging,
  ABI/lifetime and input/producer assumptions pass. Qualification of a stable
  candidate can cover several integrated packets. A later shipping change
  invalidates only evidence whose exercised source or binding changed.
- Do not promise hours from slice counts. Record the first bounded packet's
  observed editing/review/build times, then update priorities from actual cost
  and the remaining dependency chain. Evidence waits and operational availability
  remain visible rather than becoming invented delivery estimates.

Use the [task blockers](implementation-tasks.md#gates-and-readiness) as precise
dependent holds. No unknown owner/producer/parser decision is filled in by a
mechanical move. Independently challenge consequential fixes with A→B, B→C,
C→A; exchange the actual revision, changed defining units, requirement/finding,
real test subject, performed evidence and limits. The coordinator resolves
cross-file conflicts before integration.

## Keeping the queue current

After each integrated change, refresh affected task dependencies and evidence.
After every separately authorized merge, follow [maintenance](maintenance.md):
reconcile the resulting source, desired tree, review freshness, coverage and this
queue together. Mark M/C/V separately; a completed move does not resolve an open
defect, and a passing unit test does not complete native qualification. Remove
superseded tasks in place. These two scheduling documents are H06 planning
additions outside the older pinned inventories until a later baseline includes
them; they do not advance the audit or historical structural evidence.

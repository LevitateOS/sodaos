# Parallel implementation schedule

This remains the execution schedule for the existing task list and 80-slice
audit. The [task list](implementation-tasks.md) retains its 27 primary packets
and A00, with L00–L18 adoption subpackets defined in the
[library-adoption chapter](library-adoption.md). Completed structural entries
remain completed at their recorded scope; pending generic-engine splits selected
for replacement are superseded. Implementation remains deferred. The current
instruction authorizes planning and reviewed documentation commits only.

Prepared on **2026-10-06** against clean checkout
`de65ff684c29cf9510134d01917ec4e5a7afdcc3`. Its committed delta from the audit
source `f7e9cf9d` changes documentation only; unchanged source assessments are
reused at their recorded scope. Historical structural reconciliation is still
pending. The selective adoption reconciliation uses source
`72e4bb9015b6d6a622b45638104c74851a137473`; it does not advance those historical
baselines. Refresh affected source and guidance before a later authorized dispatch.

The retained [tree](proposed-tree.md), [placement](placement.md),
[ownership](package-ownership.md), [review records](reviews/README.md) and
[responsibility maps](coverage/maps/README.md) remain the authorities for exact
proposed defining files and existing caller/test/build/install duties. This
schedule assigns work; it does not override their requirements or evidence limits.

## Capacity and lanes

Use built-in Codex subagents: **up to three workers and one coordinator**,
within the four concurrent slots. Default new subagents to **GPT-6 Luna with low
reasoning**, as requested by the owner; choose stronger settings only for a
specific unresolved problem that warrants their cost. Give each a bounded
self-contained assignment. Rotate implementation, challenge and verification
through these slots; reuse sufficient evidence and stop reviews once their
question is answered. Existing tmux implementation checkpoints stay parked; this
planning round does not restart workers, create worktrees or schedule timers.

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
  instruction, selected L packet and finding IDs, one lead, literal source/target
  file list, recipients, dependency results and actual smallest verification.
  Check the packet-specific LA gate before its cutover; do not turn a blocked
  library or native proof into a whole-plan hold. Dependencies apply to the relevant
  M/C/V subtask's required boundary output, not completion of an entire packet.
  Existing normal-mode restrictions
  still apply; this planning command authorizes only these documents.
- [ ] **R01 — Integrate each coherent packet.** Complete module/import/manifest,
  compiler, payload, source-check and caller joins in the same reviewable change.
  Commit bounded coherent batches early and often when commits are authorized;
  the coordinator stages explicit paths and keeps other workers’ changes out.
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

The selected order follows the [adoption packets](library-adoption.md#execution-packets),
not the old dispatch of already completed broker/Store/release moves. These are
future priorities; no source task has been dispatched by this reconciliation.

1. **R00 and L00 per adoption.** Establish the selected exact files and physical
   writer. Start the small PG deadline/transaction and Hyper Unix/upgrade proofs
   early alongside existing-dependency repairs. A failed proof holds only that
   cutover; do not build large adapters or run release qualification around an
   unresolved native boundary.
2. **Immediate repairs, disjoint files first.** C leads L01 Phase/QMP deadlines,
   URL redaction and CoreOS writer finalization, and L02 strict SPKI/signature
   admission. A leads L03 fail-closed randomness and hash/curve replacements.
   C's L12 same-FD bounds/cancellation and L13 archive EOF/budget repairs can
   fill free slots on reserved files. A lead is not an extra worker: assign one
   actual writer or an explicit whole-file handoff for each simultaneous change.
3. **Prepare profiles and database cutover.** C leads caller-specific L04
   JSON/Base64 profiles, routing A/B callers to their physical owners. B's L07
   native SQL parameters can run alongside A's L08 driver proof; L08 cutover
   requires both L07 and LA-G2. Completed broker/provider/schema moves are not
   reopened. The canonical Go schema and Rust drift assertion remain a B/A join.
4. **Transport adoption.** A's L09 clients move against existing peers before
   ordinary identity and host HTTP. The host upgrade/read-ahead/WS pump is one
   coupled change after its LA-G3 proof. C's candidate fixture can follow the
   needed shared adapter independently. C's L10 setup HTTPS and A's provider
   handoff can proceed independently; unrelated JSON or syscall migrations do
   not gate them. Keep each shared host module and its real tests under one writer.
5. **Caller-specific format and resource changes.** L05 SSH consumes only its
   L03/L04 curve/hash/Base64 outputs. L06 CA consumes L02 and the relevant L04
   PEM profile plus Caddy/extension evidence, with no SSH dependency. L11
   URL/IP/time adapters can interleave by caller. L12 rooted/bounded custody
   precedes temporary convenience. L16 matcher adoption follows L01 completion
   and its actual L12 secret-input custody, not the reverse.
6. **Independent release and cleanup units.** C's L13 archive/XML/OCI work and
   L14 CLI/Cargo-target discovery interleave on disjoint owners, with coordinator
   manifest/cache joins at R01. B's L15 SDK admission waits only on its exact
   external-scope gate. L17 CFG01 stays held on native effective-config evidence;
   retained locale duties proceed independently. L18 removals follow current
   last-caller proof and named A/B handoffs; no whole pipeline retirement is a
   prerequisite for orphaned testoci deletion.
7. **Integrate and qualify the actual changed subjects.** R01/R02 reconcile
   callers, deletion and retained policy; R03/R04 follow applicable cheap gates.
   Remaining domain/presentation M/C/V tasks retain their existing dependencies
   and can use free slots. Select the longest ready chain rather than imposing
   a global codec, lane or wave barrier.

| Shared adoption surface | Exclusive physical writer / recipient |
| --- | --- |
| Host/identity HTTP, upgrade pump, SSH, PG Store/Tx and guest callers | A; C sends candidate/factory/setup changes, B sends canonical schema changes |
| Acceptance evidence redaction and pump-finalization file (L01/L16) | C; one writer across both stages |
| Setup/installer CA and publication; release archive/CLI/OCI/Cargo target code | C; hand host ELF/provider/FD changes to A and Go callers to B |
| Shared JSON policy and release/acceptance callers | C; A owns host/identity/guest callers, B Go admission |
| Go SQL/testoci and external SDK admission | B; A receives Rust SQL changes; sibling SDK requires LA-G6 |
| Every dependency declaration, workspace/bin/test selector and lock | Coordinator; workers submit exact requests and pause shared-file edits |

The literal paths recorded at R00 override broad directory examples only through
an explicit custody handoff. The parked A/B/C checkpoint assessment remains in
[one register](library-adoption.md#parked-checkpoints); useful seams are reassessed
with their replacement owner, never integrated wholesale to clear a queue.

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

Use the [task blockers](implementation-tasks.md#gates-and-readiness) and
[LA gates](library-adoption.md#readiness-gates) as precise dependent holds.
No unknown owner/producer/parser decision is filled in by a
mechanical move. Independently challenge consequential fixes with A→B, B→C,
C→A; exchange the actual revision, changed defining units, requirement/finding,
real test subject, performed evidence and limits. The coordinator resolves
cross-file conflicts before integration.

## Keeping the queue current

After each integrated change, refresh affected task dependencies and evidence.
After every separately authorized merge, follow [maintenance](maintenance.md):
reconcile the resulting source, desired tree, review freshness, coverage and this
queue together. Mark M/C/V separately; a completed move does not resolve an open
defect, and a passing unit test does not complete native qualification. Mark
superseded pending splits in place and link their replacement L packet; preserve
completed entries and their historical evidence. These two scheduling documents
are H06 planning
additions outside the older pinned inventories until a later baseline includes
them; they do not advance the audit or historical structural evidence.

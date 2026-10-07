# Parallel implementation schedule

This remains the execution schedule for the existing task list and 80-slice
audit. The [task list](implementation-tasks.md) retains its 27 primary packets
and A00, with L00–L18 adoption subpackets defined in the
[library-adoption chapter](library-adoption.md). Completed structural entries
remain completed at their recorded scope; pending generic-engine splits selected
for replacement are superseded. The canonical
[current task state](implementation-tasks.md#current-task-state-2026-10-07)
records completed L00–L09, L11–L15 and L18 source scopes, completed L10.N3,
held L10.N4/CFG01, completed L16.G and deferred optional L16 adoption.
Installed/native-worker qualification remains with the wider owner packets.

Prepared on **2026-10-06** against clean checkout
`de65ff684c29cf9510134d01917ec4e5a7afdcc3`. Its committed delta from the audit
source `f7e9cf9d` changes documentation only; unchanged source assessments are
reused at their recorded scope. Historical structural reconciliation is still
pending. The selective adoption reconciliation uses source
`72e4bb9015b6d6a622b45638104c74851a137473`; it does not advance those historical
baselines. The current reconciliation is pinned at `04286c48`, application
source `0b073439`, on **2026-10-07**. The intervening audits change Markdown;
R03.L retains its original verified scope. Current source responsibility coverage
is reconciled; full R02.targets desired-tree regeneration follows settled
replacement boundaries. The [finding allocation](execution-findings.md) supplies
current scoped subtasks and dependent holds. Refresh affected source/guidance
before each dispatch. The owner has now authorized execution through the full
remaining task list, with local commits early and often. Current implemented
follow-ups are reconciled at `5794a7d9` in the existing
[finding allocation](execution-findings.md#implemented-source-follow-ups-2026-10-08);
prior audit and R03.L receipts keep their original evidence scope.
The selective source receipts `f29ff882`, `96dadd55`, `37dc05fd`, `1ee47d0c`,
`00a2d4a1`, `71cfb075`, `eb83115c`, `8bcaec9c`, `71bb28bd`, and `a94edec6` are recorded in
that same table without regenerating the full tree or census. Identity HTTP
decoder checks are complete in `1a297425`; Settings retains its separate
decoder profile. A07's child-wait failure-trigger repair completed in
`a94edec6` with 39 host checks and daemon compilation; installed activation and
native-shell qualification remain open.

The retained [tree](proposed-tree.md), [placement](placement.md),
[ownership](package-ownership.md), [review records](reviews/README.md) and
[responsibility maps](coverage/maps/README.md) remain the authorities for exact
proposed defining files and existing caller/test/build/install duties. This
schedule assigns work; it does not override their requirements or evidence limits.

## Capacity and lanes

Use built-in Codex subagents, with **two workers by default and a coordinator**.
A third worker uses the fourth slot only for an independent ready assignment
with disjoint files. Rotate implementation and review through these slots;
additional concurrency is not a reason to duplicate investigation or reviews.
Checkpoint branches remain parked; their inactive worktrees were removed and
artifacts preserved as recorded in the
[checkpoint assessment](library-adoption.md#parked-checkpoints).
Work stays in the canonical checkout.

Choose the model and reasoning level **before dispatch**, using the table below.
These are economical defaults for this repository's bounded assignments, not
claims that one model is sufficient for every problem. Official
[OpenAI model guidance](https://developers.openai.com/api/docs/guides/model-selection)
describes Luna as the efficient option for scoped work; the settings here are
our engineering allocation based on the known defects and boundary risks.

| Assignment | Subagent model | Reasoning | Scope and stopping point |
| --- | --- | --- | --- |
| Inventory/diff checks, plan upkeep, last-caller census, ordinary dependency wiring | GPT-6 Luna | low | Exact paths/question; stop with the verified result or named missing fact |
| Narrow caller edits and regressions after the contract is settled | GPT-6 Luna | low | One defining owner plus actual callers/tests; stop after affected checks |
| Absolute deadlines, streaming secrecy, transaction exclusion, HTTP upgrade/pump lifetime or rooted-file custody | GPT-6 Luna | medium | One demonstrated failure or boundary proof; stop at a passing proof or concrete limitation |
| Independent review of a routine packet | GPT-6 Luna | low | Changed defining units, acceptance and performed checks only |
| Independent review of secrecy, trust, cancellation or concurrency | GPT-6 Luna | medium | Challenge those invariants and actual regression subjects; no whole-repository reread |
| Escalation after a reproducible unresolved technical failure | GPT-6.1 Sol | medium | One reduced reproducer/question only; return the answer to the Luna worker |

Escalation is conditional, not a scheduled expensive pass. First reduce the
failure and identify whether it is an implementation error, missing evidence or
an upstream limitation. Escalate only if one scoped Luna retry leaves the same
reproducer unresolved, or conflicting concrete evidence prevents the boundary
decision. Increasing reasoning or model size cannot supply absent native evidence.
There is no default high/extra-high review, full-history fork or repeated model
comparison. Give each worker a self-contained brief with finding IDs, exact
files, retained policy, prerequisites, tests and the chosen setting.

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

## Current dispatch priority

Select these outputs from the existing task list, using exact
[finding scopes](execution-findings.md). This order replaces an early full-tree
regeneration or another engine-splitting pass. It introduces no lane barrier.

| Priority / selected output | Existing owner and model/reasoning | Dependency and concurrency rule |
| --- | --- | --- |
| 1. Trustworthy observation outcomes | C11.C-observers/custody, C08.C-evidence/artifact-cleanup and L16.G; C, except named OBS-G01 handoff to B. Luna medium for deadline/capture/custody; low for settled analyzer/Cargo/CLI-selection joins | Repair the observer before using its receipts. One writer per script/helper/evidence root; same-file format cleanup waits. Completed L01/L12 remain complete |
| 1. Identity and Muse custody/bounds | A05.C-custody/bounds, A07.C-muse; A. Luna medium; low for settled startup cap | Select transaction/budget/progress contracts first. Acquisition/Store and Muse launch roots are separately reserved; schema changes go B→A mirror through coordinator |
| 1. Go authority/lifetime and saved keys | A01.C-saved-key, B03.C/lifetime, B06.C-admission and B04.C; B. Luna medium | Reserve Store/publication/coordinator files across F08/F09 as one coherent writer; A owns exact native stop/finish handoff. Operator-body fix is independent of Q8. Loop waits only on its required Q3/Q8/native outputs |
| 1. Independent setup/backup/enrollment fixes | C05.C, C06.C-rotation, C07.C-cleanup; C. Luna low for settled result/selection joins; medium if custody changes | Preserve completed source repairs. Serial shared installer/setup roots; no automatic uncertain-operation retry or native database/provider action |
| 2. Costly boundary/profile decisions | B05.C/Q5, B06.M SDK contract, A05/C05/C10 profile decisions, L10.N4, L17 and retained helper questions; named finding owner. Luna medium | Prove exact selected producer/upstream/caller contract before dependent cut. Failed proof holds that replacement only; local typed snapshot/feature trim can proceed independently |
| 3. Settled caller/dependency/test cuts | Existing A03/A05/A07, B06/B07 and C09/C10/C11 suffixed M tasks; physical owner. Luna low after profiles settle | Remove actual last callers/bridges; preserve current policy/signed bytes/fixtures. Coordinator owns all manifest/lock and shipping tuples. Park browser/A34/C41/matcher alternatives unless value is shown |
| Integrate and qualify | R01/R03/R04 coordinator; low for receipts, medium for consequential review | Focused production-path checks and one independent review precede local coherent commit. Stable current source/producer/stage joins precede resource-heavy/native qualification; full R02.targets regeneration follows implemented boundaries |

Default capacity remains two workers plus coordinator. Start the available
rank-1 packets with disjoint files; a third worker is optional for another ready
exclusive assignment. A held specification releases capacity for independent
work. No full candidate build is a prerequisite for a source-only local fix.

## File ownership and conflict avoidance

Work in the canonical checkout. Do not introduce worktrees or change Git state
through this schedule. At R00, record the literal current and target paths for
the selected packet from its review/map; directory patterns below identify the
default owner, not permission to edit every file under that directory.

| Current/target shared surface | Default physical writer | Handoff rule |
| --- | --- | --- |
| `lib/host/**`; host binary at `lib/host/src/main.rs`, proposed entrypoint at `cmd/soda-host/main.rs`; provider-neutral Factory terminal models/pumps | A | The crate move is complete. A owns current module roots and their private/oracle fixtures. B/C submit exact native changes; any remaining entrypoint/application split uses current library adapters. |
| `cmd/soda-identity/**`, including private `providers/{codex,muse}` | A | Broker/provider consolidation is complete. Preserve one Controller/State/Store/Tx and provider-support owner; no predecessor package recreation. |
| Project terminal/account/factory-role crates → `cmd/soda-project-terminal/**` | A | One guest-package module/fixture owner; preserve the three installed executable identities. |
| Go `cmd/`, `internal/` application/domain/Store/API/client files; `frontend/spaces/**`, `frontend/tailnet/**` | B | A/C send exact changes. B owns current `grants`, `dispatch`, `review_cycle`, `merge`, Store and API monoliths before releasing extracted leaves. |
| `internal/store/schema.go` and its Rust broker mirror | B defining schema; A mirror | Reconcile together under H02. Existing mirror/drift assertion remains; no new schema generator or independent authority. |
| `cmd/soda-forgejo-tailnet/main.rs` and `lib/host/src/tailnet/forgejo.rs` | A owns current native helper; C04 retains policy/qualification | L11 implemented the Rust helper and retired exclusive Go predecessors. Preserve remaining Go Forgejo/domain/status clients; future helper edits stay with A, and release selector/installed qualification joins remain C/R01/R04 duties. |
| Other Rust commands/crates and retained JSON adapters, assets, Forgejo templates/locales/styles, system/image/rootfs definitions | C | A/B request primitive or payload changes; mixed setup/install/maintenance files have one current writer. |
| Go installed-probe/test tooling, `internal/acceptance/**`, cross-component build/installed tests and source scripts | C | Preserve real subjects. B/A provide production changes; current shared fixtures/helpers remain single definitions with named consumers. |
| OBS-G01 selected `internal/acceptance/{installed,process}.go` correction | B accountable and temporary physical writer | C releases both whole files and related fixtures for this named follow-up, as the owning observation audit requires; no concurrent C edits. Return the tooling reservation after integration. Other installed-tooling/script work remains C-owned. |
| `lib/soda-release-image/src/{sys,build}.rs` and other release crate composition roots | C | Coordinator takes a short exclusive whole-file handoff for shared compile/stage joins. C pauses edits to those files until the handoff returns. |
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
  still apply; L00 preparation and proofs are complete at their recorded scope.
  Refresh this state from the completed adoption scopes and the remaining
  L10.N4 and CFG01 gates before dispatching new work.
- [ ] **R01 — Integrate each coherent packet.** Complete module/import/manifest,
  compiler, payload, source-check and caller joins in the same reviewable change.
  Commit bounded coherent batches early and often when commits are authorized;
  the coordinator stages explicit paths and keeps other workers’ changes out.
  Keep M (behavior-preserving changes), C (named correctness corrections) and V
  (verification) distinguishable. Preserve binary/service identities and direct
  defining owners; replace obsolete callers without compatibility facades.
- [ ] **R02 — Reconcile coverage and retirements.** L18 retirement maps and the
  parked A34/B27/C41 assessment are complete at `eaed66a9`; full historical
  census/range/table reconciliation remains pending. The `rust/` subtree is
  already absent; inspect current `cmd/`, `lib/`, `tools/` and `system/` owners
  rather than schedule that move again. Use these bounded outputs:
  - [x] **R02.inventory** Current responsibility coverage is reconciled at the
    `519b76bd` snapshot with `58a99730` root/assets closure and separately counted
    subsequent authored audit documents. The current inventory/maps cover source
    and mixed duties, not just counts. Keep selective deltas current after each
    integration; this completion does not refresh full target allocation.
  - [x] **R02.tasks** Current findings are reconciled from `04286c48` into the
    existing tasks/lanes with named owner/scope/prerequisites/acceptance, explicit
    completed/withdrawn/held dispositions and superseded pending directions.
  - [ ] **R02.targets** Reconcile implemented defining destinations, current
    library adapters and evidenced retirements in the tree, ownership, ports,
    slice cards and decomposition records after replacement boundaries settle.
    Preserve generated/data/license and
    live wire/fixture duties; omit superseded engine leaves and obsolete ports.
  - [ ] **R02.joins** Close actual caller/import/manifest/test/compiler/payload/
    installed-selector mappings and remaining target collisions; retain old
    audit/runtime receipts at their original identities. Task-status upkeep does
    not itself complete these joins or native qualification.
- [ ] **R03 — Integrated source verification.** On a stable agreed snapshot,
  run the applicable existing Go/Rust/Bun/source/architecture checks. Reuse a
  passing result only for unchanged exercised source. Repeat an affected check
  for new changes or unresolved failures, not automatically for every worker.
- [x] **R03.L — Completed library-adoption source scope.** Verified source:
  `d7eca882b5ce18f25ea3287dce578366d769deda`, with documentation-only
  integration upkeep in `93b50615`.
  Focused evidence tests pass 16/16; locked offline workspace metadata and
  `cargo check --workspace --all-targets` pass for all 28 members. Reuse the
  final packet-specific tests/builds/reviews recorded in the adoption chapter
  for unchanged exercised source; no fresh Go/Bun runtime result is claimed.
  Luna low evidence census found no missing completed-packet check, and an
  independent Luna low L18 review closed the stale Go fixture-owner row.
  Manifests/locks remain unchanged. This closes the selected adoption verification
  pass, not full R02 reconciliation, other pending M/C/V work or R04 qualification.
- [ ] **R04 — Selected native qualification.** After cheap boundary checks and
  producer/caller gates pass, qualify the selected matching candidate and actual
  user workflows under separately scoped operations. Keep source, build, stage,
  installed, provider and retained delivery evidence distinct. Signing, uploading,
  merging and publishing are separate operations, not automatic refactor steps.

## Restart sequence and planned settings

The existing [task list](implementation-tasks.md) remains the queue, and the
[adoption chapter](library-adoption.md#execution-packets) remains the packet
contract. This sequence resets future dispatch. Completed investigation,
planning and structural work are reused at their recorded scope.

L01 is complete: preserve Phase `9fda53fe`, CoreOS completion `3873614f`,
bounded evidence `d80aec93`, and QMP/VM ownership `4c87f5e9`. Its 126
acceptance library tests and all three binary compile checks passed. This is
local source evidence; native QEMU and installed workflows remain unqualified.
At the L00 preparation checkpoint, L02 had not started. It has since completed
in `743dde17`; RNG01 completed in `a84447ff`, and the remaining L03 SHA/NIST
source scope completed in `52eee7ee`. Detailed acceptance counts and L04
remaining work are in the adoption chapter.

| Step | Existing packet/lead | Executor setting | Required output before dependent work |
| --- | --- | --- | --- |
| 0. Establish restart state — complete | R00 / Coordinator | Luna low for bounded checks | Preserve completed corrections and structural scopes, identify the canonical revision and reconcile stopped drafts |
| 1. Finish deadlines and evidence — complete | L01 / C | Luna medium for QMP/redaction/VM pump lifetime; Luna low for CoreOS close/error joins | Recorded absolute deadlines, final curl metadata, safe split/malformed URLs, bounded evidence and failure propagation |
| 2. Finish admission and costly boundary preparation — complete | L00 / Coordinator; named disjoint proof writers | Luna low for dependency/license/cache inventory; Luna medium for PG/Hyper/WS proofs and independent review | Record exact compiler and feature closures, locked offline checks, PG deadline/cancel/discard/transaction custody, Unix client/server and upgrade read-ahead, nonblocking WS flush/close/reap; disposition every open cutover gate before L02 |
| 3. Repair immediate trust and entropy defects — complete | L02 / C; RNG01 portion of L03 / A with C handoff | Luna low implementation; independent Luna medium trust/fail-closed review | `743dde17` completes strict trust/signature repair; `a84447ff` completes fail-closed entropy across ten packages. Preserve raw DER/TBS and role authority. Evidence and limits are recorded in the adoption chapter. |
| 4. Replace common engines and native SQL parameters — L03/L04/L07 complete | L03 / A; L04 / C; L07 / B with A Rust handoff | Luna medium to settle duplicate/alias/encoding profiles and independently review; Luna low for settled caller/SQL edits | Preserve L03 `52eee7ee` and L07 `d12bf6d3`; L04 producer/refusal checks and all engine removals are complete through `229e9cce`. Apply each dependent gate to the required verified profile |
| 5. Adopt PostgreSQL and Unix HTTP/WS — source complete | L08/L09 / A, named B/C handoffs | Luna medium | L07 and L00 proofs preceded the actual driver/client/server/upgrade cutovers. Cancellation, exclusion, flush/close/reap and affected offline graphs passed through `21387814`; installed qualification remains separate |
| 6. Adopt SSH and local CA formats — source complete | L05 / A; L06 / C | Luna medium | SSH completed in `7b42671d` and CA in `7d063f15` after their required trust/encoding profiles; no SSH prerequisite for CA. Preserve recorded caller/Caddy evidence and native limits |
| 7. Replace independent external/network adapters — source complete except held N4 | L10 / C; L11 / A with C acceptance handoff | Luna medium settled URL/HTTP boundary questions; Luna low transferred settled HTTP/URL/IP/time callers | Setup HTTP and L11 source/selected checks are complete. Host provider curl remains until resolver-inclusive cancellation/Executor custody fit passes; raw literals, bounds and unavailable/unconfirmed outcomes stay with callers |
| 8. Consolidate file/FD/process mechanics — source complete | L12 / C, A host/guest/identity handoffs; B Go ownership unchanged | Luna medium for custody/cancellation and independent review; Luna low for settled repetitive plumbing | `c5cca5e7` completes same-FD bounds before temp convenience, rooted admission, owned CLOEXEC descriptors, bounded capture/cancellation, feeder/ticker joins and checked native cleanup. 1,687 selected tests and 20 package development builds pass; installed qualification remains separate |
| 9. Replace release format and CLI emulators — source complete | L13/L14 / C | Luna low after declared format/CLI profiles; Luna medium for EOF/budget/metadata ownership and independent CLI review | 656 selected tests and ten affected offline development builds pass; complete gzip, bounded extraction/shared OCI, deterministic new output, original signed bytes, actual CLI help/refusals/tails and unchanged shipping selectors are verified. Installed/shipping qualification remains separate |
| 10. Close bounded external/configuration questions — evidence complete; CFG01 cutover held | L15 / B; L17 / C | Luna low for pinned census/corpus collection; Luna medium for demonstrated semantic mismatch | SDK input repair and retained transport checks complete at Fountain `c92db11c14`. Twenty-one native configuration cases are recorded; rust-ini 0.21.3 fails continuation admission. C owns revised fit/override/dependency checks before CFG01 cutover; CFG02 and unrelated work remain independent |
| 11. Remove dead machinery and assess parked seams — complete at scoped source/assessment boundary | L18 / C with A/B handoffs; R02 | Luna low | `eaed66a9` retires N11/TMP02/DEAD01; 40 existing tests, eight command byte/inode/mode fixtures and three offline entrypoint builds pass. A34/C41 retained duties and optional seams are assigned to C04/A07; B27 is integrated and not replayed. Full R02 census and installed qualification remain separate |
| 12. Consider optional matcher adoption — consideration complete; adoption deferred | L16 / C | Luna medium | Retain the matcher. L01/L12 completed scopes stand; L16.G aggregate input collection is complete in `7334f36b`. Reconsider only after that bound, demonstrated value and construction/streaming admission; no Aho-Corasick change gates step 1 or independent work |
| 13. Qualify the integrated changed subjects — adoption source pass complete | R03.L / Coordinator; R04 remains pending | Luna low for commands/receipts; Luna medium for unresolved consequential results | Focused evidence16/16 and all-target offline Rust workspace checks pass at `d7eca882`; final unchanged-source packet receipts are reused. Native qualification requires a ready matching candidate, demonstrated producer/caller prerequisites and scoped operations; no native qualification is claimed |

Completed rows are receipts for source work, not new dispatch assignments.
Select remaining work from the current task-state table and its exact gates;
fresh package/features/runtime changes still require scoped admission.
Optional step 12 does not hold unrelated qualification. Preparation probes,
implemented source adapters and installed qualification remain distinct.

### Restart state (2026-10-07)

The preparation assignments and L07 checkpoint below preserve their historical
scope. Their former L08/L09 production-integration holds were subsequently
closed at the source scope recorded afterward and in the task list; they are
not remaining dispatch work.

The preparation source is `4b02122b`, after completed L01. The committed
`Phase::child` correction remains `9fda53fe`: unbounded parents produce
bounded children and finite parents retain the tighter deadline. Its regression
failed before repair and three child-phase checks passed afterward. The former
QMP/redaction drafts and CoreOS/VM assignments were integrated through L01;
their old parked/unverified dispatch register is retired. Source-test completion
does not qualify native QEMU or installed workflows.

Other previously recorded correctness work remains closed only at its
documented scope: [A05.C](implementation-tasks.md#a05-broker-custody-and-execution-state)
covers the terminal execution fence;
[A07.C](implementation-tasks.md#a07-native-launch-and-interactive-attachment)
covers the specified ABI/select/pump/reap repairs;
[B02.C](implementation-tasks.md#b02-accepted-inputs-and-readiness)
covers the recorded replay/stale-screen and diagnostic corrections;
[C02.C](implementation-tasks.md#c02-private-ipc-and-postgresql-persistence)
records the H01/H02 disposition and routing; and
[C08.C](implementation-tasks.md#c08-build-controllers-and-candidate-production)
records the
release architecture/discovery corrections. These task-list records do not
qualify native transport pumping, provider/native behavior, systemd/progress
output, or any wider workflow.
[C01.C](implementation-tasks.md#c01-parser-and-filesystem-primitives) is
partial (H03-F1 panic correction only), and
[C09.C](implementation-tasks.md#c09-inputs-authenticated-media-assets-and-attribution)
is partial (D02-F1 byte-safe HTTPS predicate only; Q7 remains open); neither is
a fully closed correction packet. No broader validity claim is added here.

L00 workers own only isolated proof/inventory artifacts. Production source,
manifests and locks remain unchanged during preparation. The coordinator alone
owns shared graph resolution, fixture lifecycle, checks, tracked planning joins
and commits; tests/builds are serialized.

| L00 assignment | Exact ownership | Prerequisites and acceptance | Setting |
| --- | --- | --- | --- |
| Dependency inventory | `.artifacts/l00/dependency-inventory/inventory.md` and `resolved-graphs.json` | Read coordinator-resolved metadata and exact cached manifests; account for direct/features/transitive licenses, compiler/MSRV, source/archive/checksum coverage, runtime and affected selectors; distinguish local cache from artifact worker | Luna low |
| PostgreSQL boundary proof | `.artifacts/l00/pg-boundary/src/lib.rs` | Coordinator-owned fresh PG17 socket fixture; prove one absolute deadline, acknowledged cancel/discard/join/reconnect and whole-transaction exclusion; state sync-driver and auth/TLS limits | Luna medium |
| Unix HTTP/WS proof | `.artifacts/l00/transport-boundary/src/lib.rs` and `README.md` | Exact resolved Hyper/Tokio/tungstenite graph; prove client/server Unix I/O, bounded backend admission and shutdown, validated upgrade plus first-frame read-ahead, owner wakeup/flush/automatic Pong/close/reap | Luna medium |
| Independent acceptance review | Read-only actual proof sources, results and affected plan changes | Challenge deadline/custody assertions, swallowed failures and admission claims; report exact consequential gaps for correction | Luna medium |

These assignments closed preparation only. At that checkpoint L02's C-owned
trust/signature files remained undispatched; they are now complete. The
existing investigation, coverage, adoption chapter and task list remain the
authorities; this register creates no new queue.

L00 boundary proofs were completed in `ff47e995` and remain valid unchanged.
For L07, Luna medium reused their original receipts and hashes at baseline
`921667ff`; neither proof harness was rerun. The PostgreSQL proof continues to
hold a synchronous-only driver and select the Tokio deadline facade; L08 still
owns actual Store/DSN/auth/typed-driver integration. L07 completed in
`d12bf6d3` independently: fresh PG17.11 caller tests, focused Go/Rust checks,
and development builds passed. This revalidation does not close LA-G2/3
production integration or any native qualification. Exact results are linked
from the adoption chapter's L07 note.

Every coherent packet gets one independent review at the chosen level. Routine
checks need no additional reasoning agent. Commit passing bounded changes early
and often through the coordinator, staging explicit paths; never bundle another
worker's unverified draft. Update task state in place after integration, keeping
completed moves, correctness repairs and unperformed native checks distinct.

L08/L09 source integration completed through `21387814` with Luna medium for
each implementation and independent review assignment. A's PG/typed transaction
adapter and four clients were committed and checked before server work; C's
fixture, A's identity listeners and the coupled host upgrade/pump followed as
separate reviewed commits. The original L00 proofs were reused before dispatch;
actual cancellation/exclusion, callback admission/drain, short-write/Pong,
deadline expiry and child-reap checks now pass, as does the final affected locked
offline development build. The task-owned PG fixture is stopped. This closes
step 5 at development source scope; installed/native-worker qualification remains
separate. L18 subsequently completed the scoped parked-seam assessment, while
full R02 reconciliation remains open. No lane or task
queue was replaced.

The exclusive writers and [parked checkpoint assessment](library-adoption.md#parked-checkpoints)
still apply. Manifests/locks and shared roots stay with the coordinator; record
literal files and cross-owner recipients before dispatch. Do not use concurrent
hunk editing or create extra queues to work around a held file.

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
documents accounted in the current inventories. Their original preparation and
later source-status receipts remain separate from current planning allocation;
they do not advance historical structural or native evidence.

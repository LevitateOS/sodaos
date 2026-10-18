# Coverage index

Documentation consolidation on 18 October removed ten overlapping planning/status
files and trimmed duplicate scheduling/chronology from retained owners. Current
requirements and all 67 unmet outcomes remain in the [canonical catalog](../../factory-remaining-tasks.md)
and its linked contracts. Historical inventory rows and source/body/native pins
below remain exact snapshots; deleted-document citations use Git history. This
document-only delta neither regenerates those inventories nor closes Filetree R02.

## Executable residual — Filetree R02

Ready for scoped documentation work. Start from the actual pinned inventories/maps
below and `maps/release-docker-current.md`, then compare their source ranges to the
chosen current revision with `git diff --name-status <pin>..<revision>`. Subtract
already mapped scoped deltas; the documented accumulated components-v4/pre-730fc205
gap and any later unmapped paths are the work, not another review of unchanged
leaves. Process one subsystem's changed duties at a time in its existing map,
inventory and desired-tree chapter. Trace the real entrypoint/caller and canonical
data/type owner, record retained/moved/removed destination and any substantial
large-file concern, and reconcile affected cross-owner joins and links.

Release a source-to-duty/destination mapping for each affected leaf, with its exact
range and no duplicate owner. Use changed-path membership, source-reference and
link checks to establish that bounded coverage; an uninspected body remains
explicitly uninspected. Historical body/independent/native pins are not advanced
by structural accounting. A discovered implementation defect belongs to its
existing Factory/Canlang/maintenance outcome or a concrete unresolved finding,
not a new parallel queue. Completion requires all accumulated missing coverage
to be accounted for; this executable preparation does not claim that refresh.

## Wind-down integration batch (18 October 2026)

Exact source range: `ad4603f9..51ba6d2f`, 20 changed paths, including three added documents. This is one bounded source/duty reconciliation, separate from historical whole-tree/body/native pins and older accumulated coverage gaps. No unchanged-source checks are repeated.

| Paths in this batch | Defining responsibility / desired placement |
| --- | --- |
| `internal/factory/control/stall_detect.go`, `stall_serve.go`, `stall_detect_test.go` | F/Factory Go controller detection pass, repository isolation and mixed-prerequisite regression; retain current package, serving owner and test. Observation ordering is consumed by R37; R36/R37 parent qualification remains open. |
| `internal/store/factory_stall_events.go`, `factory_completions.go` | Store owns durable stall events/observation order and registered canonical Go DDL. DDL moved into the registered schema-test owner; no second database/process owner. |
| `internal/factory/control/dispatch_attempt.go` | Existing dispatch prompt owner returns seven wait causes in the wait slot rather than failure. Accepted real-PG prompt/source checks cover the repaired seam. |
| `cmd/soda-identity/src/schema.rs` | Existing Rust mirror/drift check, stall table and version-58 assignments/allowances parity; test formatting retained. Canonical DDL remains Go-owned. |
| `lib/soda-release-image/src/model.rs` | Existing image model binds payload schema 58; focused schema/payload tests pass. No build/install proof inferred. |
| `docs/README.md`; `docs/development/factory-remaining-tasks.md`; six `factory-remaining-tasks/01-boundaries.md`, `02-prompt-to-pr.md`, `03-review-merge-reassessment.md`, `04-controls-repair-verification.md`, `05-experience-and-development.md`, `06-candidate-release-closure.md` | Existing navigation, canonical remaining outcomes and requirement cards; dependency/acceptance owners retained. |
| `docs/design/factory-feed-and-ralph-integration.md` (added) | Proposed/adopted feed direction, eight communication verbs, role/asset choices; document remains at this desired path, with R30/R32/R42/R43/R49 implementation consumers. No React or runtime component is selected by the document alone. |
| `docs/development/canlang-soda-os-implementation-plan.md`; `canlang-soda-os-implementation-plan/production-workerd.md` and `proposed-tree.md` (two added leaves) | Existing Canlang plan and its replacement-runtime/target-tree chapters; planned Canlang provider/exec-guard/runtime and Soda release owners. Desired doc paths are their current paths; code allocations remain proposals until P14–P16 native gates pass. |

All eight source/test and twelve document paths are accounted for above. The later wind-down documentation edits update these existing owners/indices and introduce no product/source path. Existing large-function concerns are retained under the current complexity policy; no zero-debt or fresh full-body review is asserted.

## Current Docker release delta (2026-10-16)

The [current ledger](maps/release-docker-current.md#installer-integration-scope)
maps all 18 installer-integration paths in `730fc205..92069938`, including three
added leaves. The source contains 2,907 tracked paths; only the selected delta's
duties are refreshed. The one newly oversized code file is the 530-line installer
fixture; its [concern review](../decomposition/release-production.md#current-docker-release-concerns-2026-10-16)
also covers the 393-line installer and changed seams in older oversized files.

The earlier 76-path Docker scope remains at `76daa366`. Accumulated components-v4
and other drift through `730fc205` (including thirteen commits after `86659d5e`)
remain pending, alongside older inventory gaps. Structural, body, independent
review and native qualification pins are unchanged; this is not all-HEAD coverage.

## Fountain LTS delta (2026-10-13)

Scoped inputs are recorded in the [historical plan](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/ideal-filetree-plan.md#scoped-fountain-lts-maintenance-2026-10-13).
Soda PR #44's release-input change modifies sixteen existing paths; six existing
plan files carry H06 documentation upkeep. No paths are added or deleted. These
selectors supplement the older inventories/maps without replacing their pins:

| Changed paths / selected duty | Existing owner and disposition |
| --- | --- |
| `fountain-source.json`; `lib/release-inputs/src/reader/forgejo.rs` | D02 retained: exact separate Fountain repository/revision, upstream version stamp and immutable Go 1.26.9 compiler input. |
| `lib/soda-release-image/src/model.rs`; `lib/soda-release-deliver/src/buildx/mod.rs` | D03 retained: consume the matching compiler image; no new compiler or packaging owner. |
| `system/host/services/forgejo.container` | D03/H01 retained: installed Fountain version identity; same service and process boundary. |
| `lib/soda-release-build/src/forgejo.rs`, `src/production/tests.rs`, `tests/oracle/production.rs`, `tests/oracle_vectors.rs`; `lib/soda-release-deliver/tests/goldens/deliver.json` | D02/D03 verification retained: version/toolchain/production fixtures track the selected input; signed historical fixture bytes retain their original identity. |
| `go.mod`, `go.sum` | H06 retained: reconcile the actual Fountain SDK dependency graph; Soda's Go directive is unchanged. |
| `frontend/forgejo/locale.lock.json` | H05/D02 retained: verify the 15.0.10 English locale input; catalog bytes and digest are unchanged. |
| `frontend/forgejo/README.md`, `frontend/forgejo/locales/README.md`; `docs/development/release.md` | H06 retained: explain current provider/locale/provenance inputs; historical template attribution remains at its original source. |

The action-pin-only PR #43 changes `.github/workflows/ci.yml` under H06 without
adding a job or changing the CI/release authority boundary. No Soda added-path
row, target leaf, package split, port or process change follows from this delta.

Paired Fountain additions stay outside the Soda inventory: upstream
`release-notes/14287.md`, seven `services/issue/TestIsValidTeamReviewRequest/*.yml`
fixtures, `services/repository/branch_test.go` and
`routers/api/actions/runner/runner_test.go` retain provider documentation,
review-authorization, Git-ref and runner-log verification duties. Existing provider owners
retain label ownership/visibility, fork-review Actions secret isolation,
release-attachment visibility, workflow scheduling and current-tip branch sync.
The conflict joins preserve Fountain's collaboration reservation and callback
context. Review correction `463d95d9f0` preserves numeric removal of labels from
the repository's own organization without exposing other owners' labels; it
finalizes runner logs only at the acknowledged end and skips no-op task writes.
The focused runner and native label-route checks pass at that correction;
the maintained merge has identical source bytes. They do not qualify the full
Actions suite or a newly installed OS.

These are scoped membership/duty observations. The pre-existing inventory gaps,
full structural refresh and historical body/review/native evidence below remain
at their recorded scope; no new whole-tree coverage count is asserted.

## Scoped native correction delta (2026-10-13)

Incremental source/duty membership is reconciled from Soda `5d281760` to
`ac22514b`, with paired Fountain `6abafdb409..c72f5607b2`: ten added Soda
paths and one deleted panel-entry re-export. The affected frontend, tests,
scripts and documentation inventories/maps assign these to their existing
G01/S02/S04/S05/N03/H05/H06 and release-asset owners. The target delta records
the new leaves and removes the obsolete entry without a package, language or
process move. Current dirty Fountain native/content/widget patches and Soda
branding/layout patches stay pending.

C06 actual HTTP/WS handler-manager-SDK checks and F05 actual browser zoom retain
their bounded scopes. F06 serves real Fountain CSS with transcribed HTML;
native-rendered/installed conditions and I04–I09 remain open. No whole-tree,
catalog, body-census or independent review pin below is advanced by this delta.

The paired Fountain delta adds 25 test files outside Soda's tracked inventory.
They remain provider-owned G01/G08/H06 evidence under C's native boundary and
B's feature lifetime writers, not new Soda packages or adapters:

| Provider-owned added paths | Existing defining responsibility |
| --- | --- |
| `routers/web/extensions/proxy_test.go` | C06 scoped transport, body bounds/cancellation and origin checks |
| `web_src/js/modules/modal.test.js` | B common native modal lifetime |
| `web_src/js/features/{pull-view-file-lifecycle,repo-code-lifecycle,repo-findfile-lifecycle,repo-wiki-lifecycle}.test.js` | B native browsing/editor retirement |
| `web_src/js/features/{repo-issue-content,repo-issue-list,repo-issue-pr-form,repo-issue-pr-status,repo-issue-sidebar-list}.test.js` | B native collaboration/form lifetime |
| `web_src/js/features/admin/{common,config,emails}.test.js` and `web_src/js/features/{common-organization,notification,org-team,repo-legacy,repo-settings,sshkey-helper}.test.js` | B account/admin/organization and mapped native feature lifetime |
| `web_src/js/features/{dashboard-repo-list,b07-closure}.test.js` | B dashboard retirement and registration-map/source guards |
| `web_src/js/features/{native-content-lifecycle,repository-vue-lifecycle}.test.js` and `web_src/js/components/repository-retirement.test.js` | B explicit native widget/Vue resource retirement and asynchronous-work guards |

Changed SDK/manifest/manager, mount/route/callback/WS, context/template/CSS and
native entry/config/feature files retain their paired provider owners. Focused
test additions are not real-page coverage of every class; current dirty native
content, Vue/widget/markup retirement and branding joins remain pending.

## Merge upkeep at `71458cb3` (2026-10-17)

Normal merge of fetched `origin/main` (`1e1ebdbd`, base `af79f31b`) into local
`b12cd494`. The incoming delta changes exactly two release-input paths and
adds/removes no file or responsibility:

| Incoming path | Existing defining responsibility |
| --- | --- |
| `system/project/canlang-release.json` | Release-input pins (Canlang runtime/preview toolchain versions) |
| `tools/release-assets/src/fetch/canlang.rs` | Release-asset fetch owner (matching admission-code update) |

No package, language, process, or review-slice change. Historical structural,
catalog, body-census, and native pins below are unchanged; the much larger
factory/source delta stays pending and is not labeled current by this merge.

## Complete source-to-slice coverage

Current Soda path navigation is reconciled at `9a184b7f` alongside the paired
Fountain source pin `1e6970300e` (2026-10-11): all 2,629 tracked Soda paths
appear once in the inventories, with no missing or duplicate
current path rows. The 21 path additions and one removal since `63ace9cd` are
selectively mapped to their existing owners; historical addition and retirement
receipts retain their original scope. The desired tree contains these paths and
the implemented D06 Rust personal-Git payload. Detailed duty/body selectors
remain pinned to `519b76bd`; current path membership and repaired navigation do
not requalify its source or runtime conclusions.

### Feature-delta upkeep at `93fb3513` (2026-10-12)

Catalog delta pin `93fb3513`, scoped to `299e3d20..93fb3513`: 93 added, 80
modified, 0 deleted outside this plan. All 173 delta paths carry current
inventory rows; new duties carry current map sections or pinned delta rows
(121 new sections). Reuse rule: files whose delta changes add named duties,
routes, tables or tests gained refreshed rows; pure-churn files reuse their
existing rows with prior pins intact. Structural tree, body-census and
review pins retain their recorded scope: the tree stays complete at
`9a184b7f` (a partial leaf add would falsify its pin, so advancement awaits
a full re-census; all delta files are retained at current paths with
existing owners and no new directories, tasks or lanes). The S 400-line
concern review and its factual addendum are incorporated as OCR pins on the
five oversized delta files; installed credential/configuration proof stays
Q08 scope. Sixteen tracked paths added between `9a184b7f` and `299e3d20`
and untouched by this delta still await rows:
`internal/config/factory_judgment_test.go`,
`internal/factory/control/intent_generation_recovery.go`,
`internal/factory/control/planning_test.go`,
`internal/factory/control/priority_dispatch.go`,
`internal/factory/control/priority_test.go`,
`internal/factory/control/priority_typesafe.go`,
`internal/factory/control/priority_typesafe_test.go`,
`internal/factory/control/serving_planning_priority_test.go`,
`internal/factory/planning.go`, `internal/factory/planning_test.go`,
`internal/forgejo/issue_graph.go`, `internal/forgejo/issue_graph_test.go`,
`internal/store/factory_planning.go`,
`internal/web/api/factory_intent_generation.go`,
`internal/web/factory_activity_test.go`,
`tests/frontend/workspace-native-shell.test.ts`.

Final upkeep at `deb79186` (2026-10-12), incremental from `93fb3513`: 1
added, 24 modified, 0 deleted outside this plan. New duties only are
recorded (scope feed/builders, committed resume, correction wait,
worker self-bind, 13 new tests); the integrated 173-path packet stands
unchanged and historical structural/body/full-review pins are untouched.
C Watch/actions wiring was still underway at this pin, so only committed
duties are recorded; its callnames join a later delta when published.

Settled duty delta at `a835c02d` (2026-10-12), incremental from
`deb79186`: 0 added, 9 modified, 0 deleted outside this plan. Records the
now-published C46ee Watch/target/reactive callnames and the L8ade remap
duties, with L484 self-bind descriptions superseded in place and kept as
history. Structural/body/full-review pins and the integrated packet stand
unchanged.

Final interface correction at `8fbed0a7` (2026-10-12), incremental from
`a835c02d`: 0 added, 4 modified, 0 deleted outside this plan. Records the
C30c/8fbe Watch-reveal duties (connected getter, detached-view drop,
navigation extraction, reveal with filter-clear/cancel/navigation-restore)
and the extended plus new phase-regression tests; superseded dispatch-only
and hidden-surface descriptions stay as history. Connected-view retention
vs same-run recreation is recorded without claiming a continuous renderer
or socket across filter DOM removal. Structural/body/full-review pins and
the integrated packet stand unchanged.

Settled-duty upkeep at `59b18572` (2026-10-12), incremental from
`8fbed0a7`: 0 added, 8 modified, 0 deleted outside this plan. Records
the four settled canon pins only: C03 direct/forward fresh-entry duties
(isNewRouteLocation, pendingNewRoute, armNewRoute, activateNewRoute, 3
new tests), I03 /factory/intents query admission with allowlist rows,
and Helper v2 native workspace shapes with subURL/prefix/canonical-path
assertions (D06 retained, new installed-map section). The workspace-driver
shell-route tweak stays under its generic inventory row (harness
accommodation, no named duty). FirstUse caller duties follow as a second
packet when the Q FirstUse checked packet lands. Structural/body/review
pins and the integrated packet stand unchanged.

Second duty packet at `2509c22c` (2026-10-12), incremental from
`59b18572`: 0 added, 3 modified, 0 deleted outside this plan. Records
the two settled canon pins only: L-5bfb scoped dashboard HTTPS/DNS
egress for the fixed design client (self stream-socket subsuming the
removed standalone listen, http name_connect, DNS resolve; no other
duty changed) and Q-2d74 FirstUse real intent preamble with mounted
API facts (new installed-map journey section, helper mountedSpacesFacts
plus evidence fields; D06 retained, test-only, no shipping change).
Structural/body/review pins and the integrated packet stand unchanged.

Final-delta upkeep at `39d9ad6a` (2026-10-12), incremental from
`2509c22c`: 0 added, 15 modified, 0 deleted outside this plan. Records
merged duties only: S13 clarify dedup (pinned texts, stable-identity
reuse, 3 tests), U06 review-only allowance top-up (fresh usage read,
412 CAS retry, 2 tests), I91 exhaustion sentinel with chooser seam and
409 funding branches, C native-New blank entry with its test, and Q07
allowance POST+201 journey receipt. I0f4 and C/I routing fixes are not
on this pin and are claimed by no row; they join a later packet when
merged. New rows carry current spans; prior rows keep their pins.
Structural/body/review pins and the integrated packet stand unchanged.

Final routing delta at `71f79938` (2026-10-12), incremental from
`39d9ad6a`: 0 added, 4 modified, 0 deleted outside this plan. Records
the settled source only: I0f revisions/plan GET query admission with
nine allowlist rows, and C accepted-overview plus idle-Configure native
New routing (committed-flag clear, stale/canRestore guard, shared
newRouteCurrent helper, 2 regressions). C source settled at this pin.
Structural/body/review pins and the integrated packet stand unchanged.

S07 duty packet at `46848018` (2026-10-12), incremental from
`71f79938`: 0 added, 2 modified, 0 deleted outside this plan. Records
the one canon pin only: effectiveDecompositionScope projection with
triplet/options call sites and 2 regression tests. Structural/body/
review pins and the integrated packet stand unchanged.

I/U duty delta at `79b4a524` (2026-10-12), incremental from
`46848018`: 0 added, 4 modified, 0 deleted outside this plan. Records
the two canon pins only: I 409 review-refusal mapping with bridge
test, and U decided-refusal wording with regression. Structural/body/
review pins and the integrated packet stand unchanged.

U decoder duty delta at `2f382509` (2026-10-12), incremental from
`79b4a524`: 0 added, 2 modified, 0 deleted outside this plan. Records
the one canon pin only: strict composite authorization decode with
fixture shape and committed-state regression step. No durable identity
or DTO change. Structural/body/review pins and the integrated packet
stand unchanged.

Sparse-overview duty delta at `31caf48a` (2026-10-12), incremental
from `2f382509`: 0 added, 6 modified, 0 deleted outside this plan (4
V source/test files recorded here; 2 Q guide docs are Q-owned and
untouched by this packet). Records sparse by_state zeros with
validation retained, the authoritative H-R read/error boundary, the
repo-only no-intent view, removal of the unreachable composed readers
with imports (stale-selector rows marked superseded, kept as history),
and the new plus converted tests. Structural/body/review pins and the
integrated packet stand unchanged.

Historical responsibility snapshot: **2026-10-07**, source `519b76bd`, with
root/asset packet `58a99730` and this authorized documentation refresh recorded
separately. The baseline’s 12 already modified guidance inputs retain their
recorded working bytes. The existing **2,545 paths**, reconciled through `f8b22b0b`,
plus **four library-caller documents, one observation-reliability chapter,
one connected workflow-trace chapter, one adapter-challenge chapter and
one representation chapter, one concurrency/termination chapter and
one authority/state chapter, one resource-bounds chapter, one custody chapter,
one dependency/architecture-cost chapter, one test/evidence chapter and
one build/installation/operational join chapter, one total maintenance chapter
one independent consequential-challenge chapter and one finding-allocation chapter**
are accounted exactly once in the [tracked-file inventories](inventory/README.md).
At that snapshot no working source additions were present. The implementation
delta adds `tests/build/helpers_test.go` (H06) at `04ddbff1`; its current owner
and regressions are linked in the tests inventory/map. These eighteen H06 documents are the explicit current documentation delta;
ignored dependencies, caches, private inputs, failed experiment output and
`.artifacts/` are excluded from authored-source coverage and are not deletion targets.

The [workflow requirements](../workflow-requirements.md) establish contract context.
This pass accounts for source, embedded Forgejo/guest programs, frontend state and
views, SQL fields, configuration, service definitions, scripts, fixtures, generated
assets, dependency/build/shipping selectors, and documentation ownership. Mixed
units use current complete symbols, fields/columns, clauses and numeric spans;
binary/captured artifacts identify producer/consumer or an explicit uncertainty.
A dependency or reader does not acquire defining ownership of the duty it consumes.

| Coverage dimension/check | Current result |
| --- | ---: |
| Historical paths including eighteen authored documents and one H06 test addition / missing / duplicate rows | 2,564 / 0 / 0 |
| Established slice IDs represented | 80 |
| Files with current detailed maps | 1,585 |
| Current mapped page families | 75 |
| Recorded baseline named duties plus eighteen whole-document H06 duties | 20,117 + 18 = 20,135 |
| Files with more than one defining slice | 492 |
| Files carrying explicit unresolved disposition | 24 |

Each named duty has exactly one slice owner and one disposition. `retained`
means the application/artifact duty currently remains; it does not approve its
current wrapper, implementation complexity or proposed decomposition. `replaced`
identifies a predecessor whose function transferred; `obsolete` excludes the
old implementation/claim from the current direction; `unresolved` names a bounded
retention, consumer or contract question while keeping one accountable slice.
Kind (program, fixture, generated output, third-party input) is separate from use.

### Source roots

| Current root at `63ace9cd` | Tracked paths |
| --- | ---: |
| `(root)` | 20 |
| `.agents` | 1 |
| `.githooks` | 1 |
| `assets` | 175 |
| `cmd` | 353 |
| `docs` | 318 |
| `frontend` | 345 |
| `internal` | 482 |
| `lib` | 428 |
| `scripts` | 81 |
| `system` | 68 |
| `tests` | 140 |
| `tools` | 197 |

### Inspection and evidence reuse

The prior catalog counted 1,719 paths at `0d8d3b8e`; the 80-slice source-validity
audit remains pinned at `f7e9cf9d`. Current path and body accounting replaces old
intervals that survived moves, splits and engine deletion. Source identity plus
the same responsibility question permits reuse of unchanged detailed units;
matching a filename or declaration alone does not establish body coverage.

Three Luna medium primaries inspected application, native runtime and release/system
families and independently challenged another family. The coordinator owns shared
records and documentation/assets. Challenges corrected dropped Go receiver methods,
SQL mechanics/columns/domain guards, factory versus human streams and their assertions,
repository selection/readiness,
HTTP/backend lifetime, broker endpoint policy, manifest/shipping duties and asset
provenance. Luna low independently checked the census and predecessor metadata.
Grouped map rows preserve all current selectors and disjoint spans without
requiring a separate source file or package for each helper. Exact per-duty span
and input identity evidence remains in the local ignored
`.artifacts/responsibility-coverage-20261007-519b76bd/` working evidence. Current
tracked inventories and maps preserve the responsibility accounting in Git.

The 400-line threshold remains an inspection trigger. All changed mixed/oversized
program families were attributed against their current bodies and joins; locks,
fixtures and documents can legitimately remain intact. Size/counts are bookkeeping,
not correctness, performance or maintenance-cost verdicts. This pass introduces
no decomposition, parser, compatibility, service or recovery implementation.

### Replaced and retired predecessors

The older tree has 713 paths absent from the current source. The normalized
predecessor ledger distinguishes 547 Git-recognized moves,
explicit package relocations/folds/splits, library replacements and positive
retirements. Move evidence establishes a path successor, not identical content or
source validity. Current responsibilities belong to current defining paths.

| Predecessor group | Current disposition and evidence |
| --- | --- |
| Rust package moves and Project/helper/provider folds | Completed placement, current manifests/module registries and build consumers determine the successors; these are not new restructuring tasks. [Placement](../placement.md), [ownership](../package-ownership.md). |
| SHA/curve, JSON/Base64, SSH and PostgreSQL/HTTP engines | Selected primitives transferred to current typed library callers. Retained authority, format profile, bounds and resource custody stay with application callers. [Adoption](../library-adoption.md). |
| Go Forgejo Tailnet helper | L11's current Rust host binary/native units replace the old Go execution/test paths. Host and Project shipping inventories have separate consumers. [L11](../library-adoption.md#l11-url-ip-and-time-adapters). |
| Orphan OCI fixtures and helper/selector predecessors | Positive L18-DEAD01/recorded retirement evidence distinguishes removal from transfer. No replacement utility/package is inferred. [L18](../library-adoption.md#l18-dead-machinery-removal). |
| `internal/forgejo/repositories_test.go` | Deleted path with no established exact successor/retirement proof in this census; former repository-reader assertion attribution remains G03. This historical evidence question does not revive the test or block unrelated current work. |

### Bounded unresolved dispositions

- O02: Go `Client.RevokeCurrentToken` retains its recorded lifecycle uncertainty;
  current Rust setup does not by itself prove this separate helper's retirement.
- S02/H06: README's entry/drawer and a retired source-path token need reconciliation
  with the current page/panel/source model. Mixed current obligations remain mapped.
- H05: unselected system/web icon and historical Anaconda kits have no demonstrated
  current shipping consumer in the bounded census. Masters, approved visual references,
  served exports, fonts and retired implementation inputs retain distinct roles.
- D11/D01: tracked root `installer`/`soda-candidate` ELF bytes have no proved link to
  the current source-built shipping outputs; execution or removal is not authorized.

These dispositions are not a new correctness-finding queue. Existing slice-local
questions, the [requirements questions](../workflow-requirements.md#requirement-questions-versus-evidence-gates),
B03.C, L16.G, L10.N4, CFG01 and native factory capability gates retain their scopes.
A passing inventory does not close a missing producer contract or stale assertion.

### What this coverage gate establishes

Current path/responsibility accounting and its consequential independent challenge
are complete at this source/document scope, including explicit unresolved duties.
Source validity, complete behavioral workflow verification, exact future target
allocation and native qualification remain their separate review dimensions.
The prior [source audit](../reviews/README.md#completed-source-audit-coverage) is not
advanced to the adoption source by this refresh. Parked R02/R03/R04 checkpoints and
completed structural work remain distinguishable from correctness findings.

No application edits, tests, builds, provider/native operations, dependency changes
or full desired-tree regeneration ran. All 12 pre-existing dirty guidance files
and 39 dependency inputs remain byte-identical. Documentation mutations are the
explicit authored input delta and do not silently change historical source pins.

### Coverage navigation

The [current library caller maps](../library-integrations/README.md) add selected
versions, upstream capabilities, retained surfaces and reverse consumers to this
responsibility coverage. Their four H06 whole-document duties are a selective
upkeep delta; earlier source/body review and unresolved dispositions are unchanged.

The [observation-reliability audit](../observation-reliability.md) adds one H06
whole-document duty at source `06f1db91`. Its bounded current findings, preserved
repairs and unperformed checks remain separate from this inventory gate and the
historical complete source-validity review.

The [connected workflow traces](../workflow-traces.md) add one H06 whole-document
duty at source `91685799`. Their current callers, findings and source-corrected
dispositions do not advance the historical full-slice validity review.

The [adapter challenges](../library-integrations/adapter-challenges.md) add one
H06 whole-document duty at source `2dc3bce9`. Selected upstream APIs and concrete
caller-interface cuts refine the existing packets; profile/upstream gates and
parked ownership assessments do not reopen completed adoption or advance native
qualification.

The [representation audit](../data-representations.md) adds one H06
whole-document duty at source `463ce772`. Concrete producer/caller questions,
raw-byte authority and conditional consolidation decisions refine the existing
packets without advancing the historical full-slice validity review.

The [resource-bounds audit](../resource-bounds.md) adds one H06 whole-document
duty at source `88a40b5d`. Allocation/expansion gates, aggregate producer gaps and
refusal propagation refine existing packets; source-only checks and held native
profiles remain distinct from qualification and historical full-slice validity.

The [custody audit](../file-and-process-custody.md) adds one H06 whole-document
duty at source `1d8c4e11`. Opened-file policy, bounded stream consumption and
primary/cleanup joins refine existing owners; conditional path/native profiles
and dormant-method retention remain explicit without changing qualification.

The [build/operational join audit](../build-and-operational-joins.md) adds one H06
whole-document duty at source `bc28a07c`. Its source-proven path repair and
running-identity evidence correction refine existing qualification owners; no
build, installation or native workflow was performed.

The [total maintenance assessment](../integration-maintenance-result.md) adds
one H06 whole-document duty at source `797e6ec8`. All 73 integration-map entries
receive a supported disposition; local joins and optional/held candidates keep
separate scope. Source recommendations do not measure total savings or qualify
the implemented product.

The [independent consequential challenge](../independent-conclusions.md) adds
one H06 whole-document duty at source `24fc3ea7`. It challenges all 73 integration
recommendations and 75 canonical prior claims, corrects evidence associations
and records one separate CA profile question. Supported, repaired, withdrawn,
dormant and held conclusions remain distinct; agreement is not qualification.

The [finding allocation](../execution-findings.md) adds one H06 whole-document
duty at source `04286c48`. It assigns the challenged findings inside the existing
task list/lanes with one accountable owner, exact scope, prerequisites and checks.
Current source coverage remains maintained; full desired-tree regeneration follows
settled replacement boundaries. No implementation or qualification is performed.

- [Current tracked inventories](inventory/README.md)
- [Current responsibility maps](maps/README.md)
- [Stable slice catalog](../slices/README.md)
- [Baseline and historical review scope](../review-baseline.md)
- [Existing tasks](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-tasks.md) and [lane schedule](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-lanes.md)

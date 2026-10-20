# Candidate, release and independent closure

[Executable task map](../factory-remaining-tasks.md#task-list) owns the current
deliverables and output dependencies. This packet preserves the parent contracts,
technical source owners, accepted evidence and additional native/installed gates.
The Rxx.y links below identify the execution units; parent input lists describe
full qualification and do not require waiting for whole parents before using
applicable outputs. Recorded baseline and landed slices retain their actual
scope. Implementation notes support the execution map rather than define a
second schedule. This documentation restructuring starts no runtime work.

<a id="r50"></a>
## R50 — Finish protected release-evidence admission and finalization

**Execution tasks:** [R50.1](../factory-remaining-tasks.md#r50-1).

**Work:** Implement and prove. **Owner lane:** C. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Recorded baseline:** Native protected admission/finalization boundary; 88a2ce7b/f188e13b serving proof and 73fe175b tag repair accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Close the protected admission/controller/finalization mechanism in the existing candidate pipeline. Prove its exact identity, failure and recovery rules with the smallest isolated development inputs. A qualified existing upgrade source must be bound when selected; an unavailable future source remains a precise external case.

**Accept only when:** Focused valid/failed/cancelled/incomplete/mismatched/forged cases and the native protected boundary prove the executable mechanism, including preservation and recovery. Valid development evidence exercises finalization without rebuilding those tested bytes, while remaining non-qualifying for production release. As original RT01 requires, this proof precedes candidate production and does not depend on R49 or the future R51/R52 candidate. Actual shipping-byte observations are collected in R52 and consumed by R53; this task does not assert them in advance. Do not replace guards with optimistic acceptance or unreleased compatibility branches.

**Milestones:** First implement and narrowly prove admission/finalization here; after R51/R52, exercise that accepted mechanism with the real candidate under R53. These are different evidence layers, with no dependency from this prerequisite back to its future candidate.

**Implementation notes:** Start at the existing protected release collector/admission/finalization path in `lib/soda-release-tools` and the release/native-support guides. First identify the admitted controller/worker and collector access for a disposable development fixture; then drive valid evidence and forged/mismatched/incomplete/cancelled failure through that same boundary. Release a demonstrated executable mechanism to R51, including preservation/recovery, without rebuilding tested bytes. Missing privileged fixture/collector access is the precise setup input for that witness. R52 later supplies shipping observations and R53 consumes them; no dependency back to those future bytes or repetition of accepted serving/tag repairs.

**Principal source owners:** [lib/soda-release-tools](../../../lib/soda-release-tools), [tools/release-assets](../../../tools/release-assets), [docs/development/native-support.md](../../../docs/development/native-support.md), [docs/development/release.md](../../../docs/development/release.md).

<a id="r51"></a>
## R51 — Produce one coherent current x86_64 candidate

**Execution tasks:** [R51.1](../factory-remaining-tasks.md#r51-1).

**Work:** Candidate production. **Owner lane:** C. **Accepted inputs:** [R02](01-boundaries.md#r02), [R43](05-experience-and-development.md#r43), [R45](05-experience-and-development.md#r45), [R49](05-experience-and-development.md#r49), [R50](06-candidate-release-closure.md#r50), [R54](06-candidate-release-closure.md#r54).

**Recorded baseline:** Coherent current x86_64 candidate after source freeze and real immutable component availability. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use existing production/staging tools for exact Fountain/SDK/package/backend/host/Project OS/source/component identities, notices and media. Reuse unchanged outputs by real inputs and update pins only as a coherent reviewed set.

**Input gate before building:** Resolve the existing candidate pipeline's immutable Fountain release manifest/image/archive and every required component by real URL, digest and source identity. A source push is insufficient. If an input is unavailable, prepare its component release through its owning pipeline and obtain any required publication authorization for that concrete input batch before this build. Keep the missing input explicit; do not wait until R53 to discover it or invent a URL. R53 publishes the finished product after candidate qualification.

**Source freeze:** R49 supplies the complete development proof; R54 closes source/caller/selector/docs changes before these bytes are produced. Any later source, pin, asset or package change returns to this candidate boundary and invalidates affected R52/R53 receipts.

**Accept only when:** The complete stage is attributable and native-checkable for x86_64; unsupported architectures refuse before fetching inputs. No obsolete alternate build engine, guessed component/public URL or hidden mismatch between local SDK replacement and packaged source pin. Candidate production is not installed qualification.

**Implementation notes:** Consume R49 development proof, R54 source freeze, R50 protected mechanism, R45 components and Canlang P27/P28's qualified runtime/tools producers. Preflight the existing release workflow with actual immutable source/SDK/package/image/archive URLs and digests, including matching Canlang closure; resolve missing component publication through its owner before building. Then produce/stage one x86_64 candidate through `soda-release`/`soda-build`, reusing outputs whose real inputs are unchanged. Release exact candidate identities/media to R52 and P30. Current source CI and preflight plus native-checkable stage are the acceptance here; installed proof follows. A missing component is a held input, never a guessed URL or alternate engine.

**Public staging prerequisite for R52:** The [release workflow](../release.md#3-stage-the-complete-unqualified-candidate) and [registry contract](../../architecture/release.md#registry-staging-and-recommendation) require immutable unqualified public staging before installation qualification. Defining owners are `lib/soda-release-deliver/src/publish/{mod,registry,components}.rs` and existing registry tests. After candidate freeze and operator-controlled registry/GitHub access, prepare the complete asset inventory, obtain authorization for that concrete staging batch when required, and use stage/verify-images. Release to R52 only when anonymous manifest/config/component readback matches Payload v4 and version-bound Compose/installer/assets are present, with neither recommendation moved. Missing visibility/credentials hold this stage. Partial writes require observed-state reconciliation, not re-upload or a new candidate. R53 owns promotion, not this prerequisite.

**Principal source owners:** [lib/soda-release-tools](../../../lib/soda-release-tools), [tools/release-assets](../../../tools/release-assets), [lib/release-inputs](../../../lib/release-inputs), [fountain-source.json](../../../fountain-source.json).

<a id="r52"></a>
## R52 — Qualify exact candidate install, upgrade, recovery and support

**Execution tasks:** [R52.1](../factory-remaining-tasks.md#r52-1).

**Work:** Installed shipping-byte qualification. **Owner lane:** Coordinator. **Accepted inputs:** [R51](06-candidate-release-closure.md#r51), [R49](05-experience-and-development.md#r49).

**Recorded baseline:** Exact candidate fresh ISO/Docker, supported upgrade, recovery and installed journey. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use the existing native x86_64 drivers against the exact candidate for fresh ISO/Docker, supported upgrade source, maintenance/rollback/recovery, retained human/network/operator behavior and finished factory.

**Expanded design consumer:** Carry R49's joined eight-profile MCP/feed/library
experience into the installed candidate's changed-boundary checks: deployed
catalog/instructions agree, decisions and knowledge remain retrievable, Library
Admin grants and history survive restart, and persisted posts retain correct
state and attribution. Reuse valid component/journey evidence; prove the actual
packaged API/Store/UI/native joins and retained data, rather than replaying every
unchanged semantic case. Source or development-only success cannot establish this
installed scope.

**Accept only when:** Actual installed byte/mode/source/artifact/architecture readback and every required case agree. Failed-run identities remain non-qualifying. Missing immutable components or a qualified upgrade source are specific unfulfilled inputs, not an invitation to rebuild blindly or waive cases.

**Implementation notes:** Consume R51's exact candidate and R49's development result. In existing native-support/testing/installation drivers, first qualify fresh ISO and Docker against those bytes; then the named supported upgrade source, rollback/recovery and retained human/network/operator behavior. Run Canlang P30's installed UI/API/app journey on this same candidate and bind P31/G4 here, not a second release. Release protected observed byte/mode/source/artifact identities to R53. Missing qualified upgrade media or private-client input holds only its case and keeps full qualification open. Driver-only fixes reuse unchanged candidate bytes; byte-changing fixes return to R51.

**Distribution-specific acceptance:** Consume R51’s public unqualified stage through the [streamed installer](../release.md#4-exercise-the-actual-public-installation-bytes) in qualification mode. The [Docker contract](../../architecture/release.md#docker-qualification) and `lib/soda-release-deliver/src/publish/qualification.rs` own receipt admission. Next bind real installer/Compose/registry observations on a fresh supported native host: rerun and truncated/hash-invalid download refusal; setup/HTTPS; Project Join/terminal/nested workload; bounded Factory use; same-image recreation retaining `/etc`, `/var` and `/home`. Exact downloaded/installed identities and credentials/database/app/Project preservation must pass without replaying initialization. ISO proof cannot substitute. Appliance upgrade/interrupted recovery uses its bound native transition receipt; Docker cross-version upgrade/downgrade/restore is outside fresh/same-image qualification and cannot be advertised from it.

<a id="consumer-download-verification"></a>
**Consumer download verification (Q3; R51/R52/R53):** Use the existing
[GitHub immutable-release trust](../../architecture/release.md#trust-key-admission)
and the supported [`gh release verify-asset`](https://cli.github.com/manual/gh_release_verify-asset)
command. R54 prepares the procedure; R51 supplies actual staged artifacts; R52
exercises it before booting the ISO or executing a saved Docker installer.
R53 repeats downloaded-byte verification after promotion.

The guide must identify the real repository, selected release tag and filenames
from the candidate inventory. Verify each executable/media asset with
`gh release verify-asset TAG FILE --repo OWNER/REPO`; placeholders must be
replaced with that explicit release identity, never an unqualified latest lookup.
Authenticate any checksum manifest the same way before using its entries.
Preserve downstream rootfs/Compose/image checks. A matching hash from the same
unauthenticated download is insufficient release-authenticity evidence.
Exercise changed bytes, an asset absent from the selected repository/tag,
missing attestation and verification/network failure; none permits continuing
installation. Record the actual consumer authentication/access requirements.
Use GitHub's existing trust mechanism, without a new Soda signing service or
local packaging key presented as public trust. This is still unqualified until
performed on the staged/downloaded artifacts.

<a id="complete-cold-recovery"></a>
**Complete cold recovery (Q5; R45/R52/R54):** Adopt one consistent stopped
recovery point using existing disk/volume tools. R45 develops and exercises the
procedure and repairs demonstrated storage/shutdown gaps before R54 freezes
shipping instructions. R52 repeats the changed installed boundary on the
candidate:

- For the appliance, cleanly stop it and capture the boot disk and every
  attached disk/filesystem holding retained state as one offline set.
- For Docker, stop the Soda container and all writers to its retained storage;
  capture all `/etc`, `/var` and `/home` volumes together with the exact
  Compose/deployment settings and image identity. Unrelated containers need
  not stop. Include external retained mounts or report the set incomplete.
- Inventory databases and repositories, Project roots/accounts/homes/dirty work,
  application data, Factory/Identity/broker state and matching encryption/key
  material, setup/configuration and TLS/SSH trust. Project roots are not assumed
  to live only under `/home`. Do not publish credentials or backups as fixtures;
  encrypt/protect the recovery set with separately recoverable operator custody.
- Restore to a fresh isolated destination using the matching supported image
  and every part of the same set. Keep the original untouched. Before connecting
  the clone to real providers or users, inspect retained identities and pending
  operations so the recovery exercise cannot duplicate external work.
- Prove login, repositories, Project Join/terminal and dirty files, app-local
  identity/data, Factory records and secret-backed service startup after
  recovery. Record exact source/image/storage boundaries and omitted external
  services; partial database exports and app snapshots retain their narrower
  scope. Include interrupted/incomplete capture refusal.

This is an operator-triggered cold procedure, not automatic online backup or
an RPO/RTO promise. Same-image Docker recovery does not establish cross-version
restore support. Any required product changes return to R45 before candidate
creation; do not introduce a backup daemon to write the guide.

<a id="supported-sizing"></a>
**Supported sizing (Q6; R45/R52/R54):** R45 measures working basic and
small-team profiles on the current native development product using existing
workloads. R54 places those measured allocations and procedures in shipping
guidance before R51 freezes it. R52 then verifies them on the exact candidate
for both installation forms; only that result establishes release support.
Include a tested minimum for the stated basic workload and a recommended
profile for the default two active Factory issues across Projects, a human
terminal and a small private app. State actual concurrency, app count and
relevant external provider limits; fixture allocations alone do not establish
support.

Record peak memory/CPU, startup and ordinary task responsiveness, installed and
retained disk use, and temporary/free-space needs for update, rollback and the
selected backup destination. Docker figures must distinguish Soda's allocation
from the host's other workloads. Use actual low-resource failures to explain
diagnostics and recovery; never prescribe blanket data deletion. R52 records
confirmation without editing frozen shipping guidance. If the candidate fails
the stated profile or needs different values, return through R45/R54/R51 and
repeat only affected qualification; do not silently change the promised numbers
after testing. Do not guess numbers or require an exhaustive hardware benchmark
before releasing a useful supported profile.

**Application consumers (Q4/Q7):** P14/P25/P26 provide implemented setup,
restore and fixed resource-limit behavior before candidate freeze. P30 shares
this installed fixture to prove first-owner setup, two-user use and fresh
restore at the same application address. Exercise the 1 GiB RAM, two-CPU and
128-process-and-thread limits and report actual failure/restart/persistence behavior.
These caps are per application; they do not reserve host capacity or qualify
an unlimited number of apps.

**Principal source owners:** [docs/development/testing.md](../../../docs/development/testing.md), [docs/guides/installation.md](../../../docs/guides/installation.md), [docs/guides/docker.md](../../../docs/guides/docker.md), [docs/guides/media.md](../../../docs/guides/media.md).

<a id="r53"></a>
## R53 — Complete immutable public delivery and downloaded-byte verification

**Execution tasks:** [R53.1](../factory-remaining-tasks.md#r53-1).

**Work:** External publication gate. **Owner lane:** Coordinator. **Accepted inputs:** [R52](06-candidate-release-closure.md#r52).

**Recorded baseline:** Concrete immutable public release and actual downloaded-byte qualification. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Promote R51’s already staged immutable package/image/archive/media/metadata batch using R52’s protected exact-candidate evidence through R50. Obtain authorization for the concrete promotion when required, then verify public recommendations and downloads. Component inputs and unqualified public staging precede R52; neither can be postponed here.

**Accept only when:** Published URLs/digests/provenance and downloaded/installable bytes match the already qualified candidate. No force/settings/protection bypass or inferred deployment permission. This task list and the earlier origin-push instruction do not themselves authorize product release/deployment.

**Implementation notes:** Consume R52’s protected exact-candidate evidence through R50 and inspect R51’s staged remote identities. Use `lib/soda-release-deliver/src/publish` to promote the unchanged qualified batch, then verify anonymous downloads and eligible recommendation pointers. Docker qualification markers belong in editable release notes: no appending immutable assets, rebuilding or re-encoding during promotion. Docker-only promotion preserves GitHub’s appliance recommendation; all-target promotion requires both distributions qualified. Partial GHCR/GitHub changes retain artifacts and report actual remote state for reconciliation, without blind retry/overwrite/rollback. Release exact URLs/digests and public readback to R55. A mismatch holds delivery; this plan or Git push does not authorize publication.

**Principal source owners:** [tools/release-assets](../../../tools/release-assets), [docs/development/release.md](../../../docs/development/release.md).

<a id="r54"></a>
## R54 — Finish one-current-path cutover, docs and safe resource accounting

**Execution tasks:** [R54.1](../factory-remaining-tasks.md#r54-1).

**Work:** Cutover and documentation. **Owner lane:** Coordinator. **Accepted inputs:** [R41](04-controls-repair-verification.md#r41), [R43](05-experience-and-development.md#r43), [R45](05-experience-and-development.md#r45), [R49](05-experience-and-development.md#r49), [R50](06-candidate-release-closure.md#r50).

**Recorded baseline:** One-current-path source/caller/selector/docs/license/pin cutover before candidate freeze. Cleanup shutdown does not close product cutover. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Remove evidenced obsolete runner/transport/result/format paths and their actual callers/tests/assets/selectors with each replacement, then close source/staging/support/documentation gaps before R51 freezes candidate bytes. Reconcile product/architecture/reference/operator/handbook truths, licensing and final pins. Release disposable development resources only after their custody has ended; keep candidate qualification fixtures available through R52/R53 and account for their final release in R55.

**Full MCP/feed/library cutover:** Consume the expanded R30–R43 interfaces and
R49 joins. Update the single source catalog, supported harness instructions,
subject/result mirrors, persisted schemas, API/frontend callers and user-facing
decision/library maintenance guidance together. Retire superseded unreleased
formats/callers rather than shipping two authoritative paths. Package the same
existing Go/Rust/Lit owners for R51; the Library Admin profile adds no standalone
knowledge daemon or broad operator account. Keep proposal names clearly labeled
until their actual implemented contract is reflected in the canonical docs.

**Handbook resolution cutover:** Carry the
[seven selected resolutions](../factory-remaining-tasks.md#selected-product-resolutions-for-handbook-blockers)
into existing product/reference/guide/public owners as their implementations
land. Include runnable worked tasks for member removal/restoration, prepared-app
setup, first-owner access, stopped backup/fresh restore and verified downloads.
Keep prerequisites, exact permissions, expected result and partial-failure
recovery together. Make these tasks easy to find from their existing chapter
entry pages. Preserve public article summaries, meaningful anchors and the
shared authoring contract; never present a proposed button or unmeasured sizing
number as available. Candidate measurement/publication receipts may follow
under R52/R53 without changing product bytes.

**Accept only when:** No parallel legacy reader/engine or misleading existing-interface/public-readiness claim remains, and all shipping source/selector changes are complete before candidate production. Native Actions and human work are preserved, no broad prune occurs, and resources with unknown ownership remain explicit. Publication receipts and actual URLs can be recorded under R53 after qualification without changing tested product bytes; any byte-changing correction requires a new R51 candidate. Completed planning material is absorbed only after its requirements/evidence are carried forward.

**Implementation notes:** Consume completed R41/R43/R45/R49/R50 changes, then inspect only their actual superseded callers/selectors/assets and affected durable docs. Remove each obsolete path with its replacement and reconcile notices/pins before R51's source freeze; reuse accepted cutovers and the already closed cleanup. Include measured Canlang P29 contracts and the P27/P28 release schema when they are ready. Release coherent shipping source/docs plus retained qualification-resource ownership. Check affected source references/links and required changed-source gates; do not run broad prune or mistake documentation consolidation for P31/G4 acceptance. Later receipt-only documentation is allowed; changed product bytes require a new candidate.

**Principal source owners:** [docs/README.md](../../../docs/README.md), [Remaining outcomes](../factory-remaining-tasks.md#requirement-aliases-and-shared-consumers--not-additional-outcomes), [Workflow contracts](../factory-workflow-contracts.md), [docs/reference](../../../docs/reference), [docs/public](../../../docs/public).

<a id="r55"></a>
## R55 — Independently accept the complete remaining scope and close execution

**Execution tasks:** [R55.1](../factory-remaining-tasks.md#r55-1).

**Work:** Independent acceptance. **Owner lane:** Codex. **Accepted inputs:** [R46](05-experience-and-development.md#r46), [R47](05-experience-and-development.md#r47), [R48](05-experience-and-development.md#r48), [R49](05-experience-and-development.md#r49), [R50](06-candidate-release-closure.md#r50), [R51](06-candidate-release-closure.md#r51), [R52](06-candidate-release-closure.md#r52), [R53](06-candidate-release-closure.md#r53), [R54](06-candidate-release-closure.md#r54).

**Recorded baseline:** Independent acceptance of complete mapped factory scope; stopping execution does not accept unfinished backlog. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Independently close every mapped requirement/control/case at its actual revision and evidence layer, resolve substantive findings, verify focused commits and writer/command/resource release, and report precise remaining external cases if any.

**Accept only when:** Production callers and decisive evidence agree across the original 26 deliverables, all refinements, seven profiles, fourteen controls and retained appliance/UX/release requirements. Uncovered or externally gated cases keep full qualification open. Retire supervision only when no follow-up remains or the human stops it.

**Implementation notes:** Consume the existing mapped requirement results from R46–R54, P31/G4 (R52) and P30. Account separately for remaining Canlang and maintenance outcomes; deferred P08/P10 or unchanged source debt must not become artificial prerequisites for the adopted first-app release. Reconcile uncovered criteria once against production callers and decisive evidence, resolve substantive discrepancies and report any named external cases. Complete Canlang P32 by reconciling qualified release behavior with its durable owners; the earlier removal of duplicate plans does not satisfy qualification. Release only identified ended fixture/writer resources through existing ownership records; the October 18 cleanup remains closed. Independent final acceptance is an evidence judgment, not mandatory reruns, a review-of-review or a new supervisor. An unfulfilled mapped Factory/release criterion keeps this parent open; any other remaining catalog row still prevents claiming the entire 67-outcome catalog complete.

**Principal source owners:** [docs/development/factory-remaining-tasks.md](../../../docs/development/factory-remaining-tasks.md), [docs/development/factory-workflow-contracts.md](../../../docs/development/factory-workflow-contracts.md), [docs/development/testing.md](../../../docs/development/testing.md).

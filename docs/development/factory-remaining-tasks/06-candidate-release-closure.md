# Candidate, release and independent closure

[Remaining-work catalog](../factory-remaining-tasks.md#task-list) owns execution order. These packets preserve the original requirement authorities; the catalog's consolidated outcomes own current status; the closed checklist retains evidence and former leases. Implementation is stopped. The executable remainder refines each existing parent without adding counted outcomes. Source locators identify the current owner, not blanket write permission; use the catalog’s next-step disposition and actual prerequisite outputs.

<a id="r50"></a>
## R50 — Finish protected release-evidence admission and finalization

**Work:** Implement and prove. **Owner lane:** C. **Accepted inputs:** [R01](01-boundaries.md#r01).

**Current (wind-down):** Native protected admission/finalization boundary; 88a2ce7b/f188e13b serving proof and 73fe175b tag repair accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Close the protected admission/controller/finalization mechanism in the existing candidate pipeline. Prove its exact identity, failure and recovery rules with the smallest isolated development inputs. A qualified existing upgrade source must be bound when selected; an unavailable future source remains a precise external case.

**Accept only when:** Focused valid/failed/cancelled/incomplete/mismatched/forged cases and the native protected boundary prove the executable mechanism, including preservation and recovery. Valid development evidence exercises finalization without rebuilding those tested bytes, while remaining non-qualifying for production release. As original RT01 requires, this proof precedes candidate production and does not depend on R49 or the future R51/R52 candidate. Actual shipping-byte observations are collected in R52 and consumed by R53; this task does not assert them in advance. Do not replace guards with optimistic acceptance or unreleased compatibility branches.

**Milestones:** First implement and narrowly prove admission/finalization here; after R51/R52, exercise that accepted mechanism with the real candidate under R53. These are different evidence layers, with no dependency from this prerequisite back to its future candidate.

**Executable remainder:** Start at the existing protected release collector/admission/finalization path in `lib/soda-release-tools` and the release/native-support guides. First identify the admitted controller/worker and collector access for a disposable development fixture; then drive valid evidence and forged/mismatched/incomplete/cancelled failure through that same boundary. Release a demonstrated executable mechanism to R51, including preservation/recovery, without rebuilding tested bytes. Missing privileged fixture/collector access is the precise setup input for that witness. R52 later supplies shipping observations and R53 consumes them; no dependency back to those future bytes or repetition of accepted serving/tag repairs.

**Principal source owners:** [lib/soda-release-tools](../../../lib/soda-release-tools), [tools/release-assets](../../../tools/release-assets), [docs/development/native-support.md](../../../docs/development/native-support.md), [docs/development/release.md](../../../docs/development/release.md).

<a id="r51"></a>
## R51 — Produce one coherent current x86_64 candidate

**Work:** Candidate production. **Owner lane:** C. **Accepted inputs:** [R02](01-boundaries.md#r02), [R43](05-experience-and-development.md#r43), [R45](05-experience-and-development.md#r45), [R49](05-experience-and-development.md#r49), [R50](06-candidate-release-closure.md#r50), [R54](06-candidate-release-closure.md#r54).

**Current (wind-down):** Coherent current x86_64 candidate after source freeze and real immutable component availability. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use existing production/staging tools for exact Fountain/SDK/package/backend/host/Project OS/source/component identities, notices and media. Reuse unchanged outputs by real inputs and update pins only as a coherent reviewed set.

**Input gate before building:** Resolve the existing candidate pipeline's immutable Fountain release manifest/image/archive and every required component by real URL, digest and source identity. A source push is insufficient. If an input is unavailable, prepare its component release through its owning pipeline and obtain any required publication authorization for that concrete input batch before this build. Keep the missing input explicit; do not wait until R53 to discover it or invent a URL. R53 publishes the finished product after candidate qualification.

**Source freeze:** R49 supplies the complete development proof; R54 closes source/caller/selector/docs changes before these bytes are produced. Any later source, pin, asset or package change returns to this candidate boundary and invalidates affected R52/R53 receipts.

**Accept only when:** The complete stage is attributable and native-checkable for x86_64; unsupported architectures refuse before fetching inputs. No obsolete alternate build engine, guessed component/public URL or hidden mismatch between local SDK replacement and packaged source pin. Candidate production is not installed qualification.

**Executable remainder:** Consume R49 development proof, R54 source freeze, R50 protected mechanism, R45 components and Canlang P27/P28's qualified runtime/tools producers. Preflight the existing release workflow with actual immutable source/SDK/package/image/archive URLs and digests, including matching Canlang closure; resolve missing component publication through its owner before building. Then produce/stage one x86_64 candidate through `soda-release`/`soda-build`, reusing outputs whose real inputs are unchanged. Release exact candidate identities/media to R52 and P30. Current source CI and preflight plus native-checkable stage are the acceptance here; installed proof follows. A missing component is a held input, never a guessed URL or alternate engine.

**Public staging prerequisite for R52:** The [release workflow](../release.md#3-stage-the-complete-unqualified-candidate) and [registry contract](../../architecture/release.md#registry-staging-and-recommendation) require immutable unqualified public staging before installation qualification. Defining owners are `lib/soda-release-deliver/src/publish/{mod,registry,components}.rs` and existing registry tests. After candidate freeze and operator-controlled registry/GitHub access, prepare the complete asset inventory, obtain authorization for that concrete staging batch when required, and use stage/verify-images. Release to R52 only when anonymous manifest/config/component readback matches Payload v4 and version-bound Compose/installer/assets are present, with neither recommendation moved. Missing visibility/credentials hold this stage. Partial writes require observed-state reconciliation, not re-upload or a new candidate. R53 owns promotion, not this prerequisite.

**Principal source owners:** [lib/soda-release-tools](../../../lib/soda-release-tools), [tools/release-assets](../../../tools/release-assets), [lib/release-inputs](../../../lib/release-inputs), [fountain-source.json](../../../fountain-source.json).

<a id="r52"></a>
## R52 — Qualify exact candidate install, upgrade, recovery and support

**Work:** Installed shipping-byte qualification. **Owner lane:** Coordinator. **Accepted inputs:** [R51](06-candidate-release-closure.md#r51), [R49](05-experience-and-development.md#r49).

**Current (wind-down):** Exact candidate fresh ISO/Docker, supported upgrade, recovery and installed journey. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Use the existing native x86_64 drivers against the exact candidate for fresh ISO/Docker, supported upgrade source, maintenance/rollback/recovery, retained human/network/operator behavior and finished factory.

**Accept only when:** Actual installed byte/mode/source/artifact/architecture readback and every required case agree. Failed-run identities remain non-qualifying. Missing immutable components or a qualified upgrade source are specific unfulfilled inputs, not an invitation to rebuild blindly or waive cases.

**Executable remainder:** Consume R51's exact candidate and R49's development result. In existing native-support/testing/installation drivers, first qualify fresh ISO and Docker against those bytes; then the named supported upgrade source, rollback/recovery and retained human/network/operator behavior. Run Canlang P30's installed UI/API/app journey on this same candidate and bind P31/G4 here, not a second release. Release protected observed byte/mode/source/artifact identities to R53. Missing qualified upgrade media or private-client input holds only its case and keeps full qualification open. Driver-only fixes reuse unchanged candidate bytes; byte-changing fixes return to R51.

**Distribution-specific acceptance:** Consume R51’s public unqualified stage through the [streamed installer](../release.md#4-exercise-the-actual-public-installation-bytes) in qualification mode. The [Docker contract](../../architecture/release.md#docker-qualification) and `lib/soda-release-deliver/src/publish/qualification.rs` own receipt admission. Next bind real installer/Compose/registry observations on a fresh supported native host: rerun and truncated/hash-invalid download refusal; setup/HTTPS; Project Join/terminal/nested workload; bounded Factory use; same-image recreation retaining `/etc`, `/var` and `/home`. Exact downloaded/installed identities and credentials/database/app/Project preservation must pass without replaying initialization. ISO proof cannot substitute. Appliance upgrade/interrupted recovery uses its bound native transition receipt; Docker cross-version upgrade/downgrade/restore is outside fresh/same-image qualification and cannot be advertised from it.

**Principal source owners:** [docs/development/testing.md](../../../docs/development/testing.md), [docs/guides/installation.md](../../../docs/guides/installation.md), [docs/guides/docker.md](../../../docs/guides/docker.md), [docs/guides/media.md](../../../docs/guides/media.md).

<a id="r53"></a>
## R53 — Complete immutable public delivery and downloaded-byte verification

**Work:** External publication gate. **Owner lane:** Coordinator. **Accepted inputs:** [R52](06-candidate-release-closure.md#r52).

**Current (wind-down):** Concrete immutable public release and actual downloaded-byte qualification. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Promote R51’s already staged immutable package/image/archive/media/metadata batch using R52’s protected exact-candidate evidence through R50. Obtain authorization for the concrete promotion when required, then verify public recommendations and downloads. Component inputs and unqualified public staging precede R52; neither can be postponed here.

**Accept only when:** Published URLs/digests/provenance and downloaded/installable bytes match the already qualified candidate. No force/settings/protection bypass or inferred deployment permission. This task list and the earlier origin-push instruction do not themselves authorize product release/deployment.

**Executable remainder:** Consume R52’s protected exact-candidate evidence through R50 and inspect R51’s staged remote identities. Use `lib/soda-release-deliver/src/publish` to promote the unchanged qualified batch, then verify anonymous downloads and eligible recommendation pointers. Docker qualification markers belong in editable release notes: no appending immutable assets, rebuilding or re-encoding during promotion. Docker-only promotion preserves GitHub’s appliance recommendation; all-target promotion requires both distributions qualified. Partial GHCR/GitHub changes retain artifacts and report actual remote state for reconciliation, without blind retry/overwrite/rollback. Release exact URLs/digests and public readback to R55. A mismatch holds delivery; this plan or Git push does not authorize publication.

**Principal source owners:** [tools/release-assets](../../../tools/release-assets), [docs/development/release.md](../../../docs/development/release.md).

<a id="r54"></a>
## R54 — Finish one-current-path cutover, docs and safe resource accounting

**Work:** Cutover and documentation. **Owner lane:** Coordinator. **Accepted inputs:** [R41](04-controls-repair-verification.md#r41), [R43](05-experience-and-development.md#r43), [R45](05-experience-and-development.md#r45), [R49](05-experience-and-development.md#r49), [R50](06-candidate-release-closure.md#r50).

**Current (wind-down):** One-current-path source/caller/selector/docs/license/pin cutover before candidate freeze. Cleanup shutdown does not close product cutover. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Remove evidenced obsolete runner/transport/result/format paths and their actual callers/tests/assets/selectors with each replacement, then close source/staging/support/documentation gaps before R51 freezes candidate bytes. Reconcile product/architecture/reference/operator/handbook truths, licensing and final pins. Release disposable development resources only after their custody has ended; keep candidate qualification fixtures available through R52/R53 and account for their final release in R55.

**Accept only when:** No parallel legacy reader/engine or misleading existing-interface/public-readiness claim remains, and all shipping source/selector changes are complete before candidate production. Native Actions and human work are preserved, no broad prune occurs, and resources with unknown ownership remain explicit. Publication receipts and actual URLs can be recorded under R53 after qualification without changing tested product bytes; any byte-changing correction requires a new R51 candidate. Completed planning material is absorbed only after its requirements/evidence are carried forward.

**Executable remainder:** Consume completed R41/R43/R45/R49/R50 changes, then inspect only their actual superseded callers/selectors/assets and affected durable docs. Remove each obsolete path with its replacement and reconcile notices/pins before R51's source freeze; reuse accepted cutovers and the already closed cleanup. Include measured Canlang P29 contracts and the P27/P28 release schema when they are ready. Release coherent shipping source/docs plus retained qualification-resource ownership. Check affected source references/links and required changed-source gates; do not run broad prune or mistake documentation consolidation for P31/G4 acceptance. Later receipt-only documentation is allowed; changed product bytes require a new candidate.

**Principal source owners:** [docs/README.md](../../../docs/README.md), [Remaining outcomes](../factory-remaining-tasks.md#requirement-aliases-and-shared-consumers--not-additional-outcomes), [Workflow contracts](../factory-workflow-contracts.md), [docs/reference](../../../docs/reference), [docs/public](../../../docs/public).

<a id="r55"></a>
## R55 — Independently accept the complete remaining scope and close execution

**Work:** Independent acceptance. **Owner lane:** Codex. **Accepted inputs:** [R46](05-experience-and-development.md#r46), [R47](05-experience-and-development.md#r47), [R48](05-experience-and-development.md#r48), [R49](05-experience-and-development.md#r49), [R50](06-candidate-release-closure.md#r50), [R51](06-candidate-release-closure.md#r51), [R52](06-candidate-release-closure.md#r52), [R53](06-candidate-release-closure.md#r53), [R54](06-candidate-release-closure.md#r54).

**Current (wind-down):** Independent acceptance of complete mapped factory scope; stopping execution does not accept unfinished backlog. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Independently close every mapped requirement/control/case at its actual revision and evidence layer, resolve substantive findings, verify focused commits and writer/command/resource release, and report precise remaining external cases if any.

**Accept only when:** Production callers and decisive evidence agree across the original 26 deliverables, all refinements, seven profiles, fourteen controls and retained appliance/UX/release requirements. Uncovered or externally gated cases keep full qualification open. Retire supervision only when no follow-up remains or the human stops it.

**Executable remainder:** Consume the existing mapped requirement results from R46–R54, P31/G4 (R52) and P30. Account separately for remaining Canlang and maintenance outcomes; deferred P08/P10 or unchanged source debt must not become artificial prerequisites for the adopted first-app release. Reconcile uncovered criteria once against production callers and decisive evidence, resolve substantive discrepancies and report any named external cases. Complete Canlang P32 by reconciling qualified release behavior with its durable owners; the earlier removal of duplicate plans does not satisfy qualification. Release only identified ended fixture/writer resources through existing ownership records; the October 18 cleanup remains closed. Independent final acceptance is an evidence judgment, not mandatory reruns, a review-of-review or a new supervisor. An unfulfilled mapped Factory/release criterion keeps this parent open; any other remaining catalog row still prevents claiming the entire 67-outcome catalog complete.

**Principal source owners:** [docs/development/factory-remaining-tasks.md](../../../docs/development/factory-remaining-tasks.md), [docs/development/factory-workflow-contracts.md](../../../docs/development/factory-workflow-contracts.md), [docs/development/testing.md](../../../docs/development/testing.md).

# Remaining factory work

This is the canonical remaining-outcome record, reconciled for the human-directed
wind-down on 18 October 2026 at source `51ba6d2f`. Implementation and supervision
are stopped. This record does not dispatch work. It replaces the execution order
in older FW/FT/ST/RT/UX/RCX packets while preserving their requirements and accepted work.

**55 factory IDs: R01/R02 accepted at their defined scope; 53 retain unmet parent criteria. The complete factory is not ready.** Source acceptance is preserved separately from native, installed and release acceptance. The remaining-outcome tables below own current status; historical snapshots are supporting evidence.

This document owns remaining outcomes, dependencies and acceptance. The existing
[central checklist](../../.artifacts/muse-factory-workflow/checklist.md) is the
closed run's evidence and former lease record, not another active queue. All
implementation assignments are released; A/B/C below identify former responsibility
lanes, not active workers. The [workflow contracts and decisive cases](factory-workflow-contracts.md),
[testing owner](testing.md), [product](../product/overview.md) and
[trust model](../architecture/trust.md) own durable requirements. Old packet IDs
remain aliases in this catalog; their obsolete schedules are preserved in Git.

The executable second pass below is based on source `2d576eeb` (18 October 2026).
Each existing task card now gives a concrete starting point, ordered remainder,
prerequisite output and proportionate proof. These are instructions for future
authorized execution; no source implementation, native run or cleanup is reopened.

## How to execute the remaining work

Read a row's **Executable remainder** with its retained acceptance criteria.
Take the first coherent slice whose actual inputs exist; release its output to
consumers without waiting for unrelated parts of the parent. The old level graph
below describes full parent qualification, not a requirement to finish whole
plans before using accepted contracts. Local slices do not add counted outcomes.
An accepted source mechanism is a starting point, not a request to rebuild it.
Small-model manageability is aspirational; difficult authority, transaction or
native-lifetime work can use stronger executor judgment without another gate.

At this source baseline the **next-step disposition** is **17 ready slices,
47 dependency-held outcomes, 2 decision-held outcomes and 1 external-input-held
outcome**. This partitions all 67 by their first useful step, not by full
acceptance or permission to run. A ready preparation slice can still have a later
native, funded-provider or publication gate. Every implementation owner remains
unassigned; the named code owner is the existing subsystem, not a new worker.

| Disposition | Existing outcome IDs | First useful work / release |
| --- | --- | --- |
| Ready slice (13 factory) | R03, R06, R07, R10, R11, R15, R31, R32, R36, R42, R43, R47, R50 | Current native fixture preparation/qualification for R03/R06/R50; harness installation/driver alignment for R07; bounded ingress/transaction/plan/read/communication/progression work for R10/R11/R15/R31/R32/R36/R42; concrete Lit/Ralph flow design for R43; recorded Deep-CI failure triage for R47. Native fixture access and paid trials still require their concrete inputs. |
| Ready slice (4 additional) | Canlang P09, P14; BROAD-COMPLEXITY-037; Filetree R02 | Existing Files lifecycle proof, native direct-runtime assumption proof, one cohesive residual Go simplification, or missing source/duty mapping. Canlang producer work belongs to its repository owner; this Soda planning pass neither dispatches nor edits that repository. |
| Decision held (2) | Canlang P08, P10 | Select the R2 adapter/service contract or concrete external provider/effect/reconciliation contract. Existing native Files/internal durable-work behavior remains usable. Decision outputs and owners are defined below; these adopted deferred outcomes stay counted. |
| External input held (1) | Canlang P22 | Selected real client, authorized route/enrollment and TLS trust input. Local/CoreOS ingress proof is retained. |
| Dependency held (47) | All other R03–R55 rows (40); Canlang P15, P16, P25, P26, P27/P28, P29, P30 (7) | Consume the named outputs in the task cards and runtime substeps. Resolve local source slices as inputs arrive; full native/release acceptance remains distinct. |

Source inspection found concrete serving gaps at `ImportTaskEvents` and
`ValidateReassessmentAdoption` (no production callers), MCP `Observed`/`Catalog`
wiring, incomplete `serveEvidence`, and stall-generation/verification consumers.
It also confirmed existing typed delivery subjects, immutable plans, native task
journals, staged capabilities, receipt replay and the serving stall detector.
The cards target these joins. This is a scoped planning inspection, not renewed
body coverage, fresh runtime evidence or acceptance of all historical criteria.

Use existing focused Go/PG, Rust and browser tests for the changed boundary:
`go test -count=1 ./internal/factory/control -run '<affected cases>'` (and the
actual Store/API package where changed), the relevant locked Cargo package/test,
or the existing frontend selector from [testing](testing.md). A PG test skipped
for missing `SODA_PG_*` is not a pass. Native/container/funded checks use the
owning guide and exact admitted inputs under `/home`; no whole suite or candidate
rebuild is required for an unchanged component. Keep one ordinary change record,
reuse valid evidence, and add a regression only for a real behavior/failure seam.
No per-task evidence packet, new queue, mandatory model trial or review-of-review.

Release ordering includes the shared Canlang path: P14 → P15/P16 → P25 →
P26 → P27/P28 and P29 → **R54 source freeze → R51 candidate/public unqualified stage → P30 installed
journey → R52/P31/G4 → R53 → R55/P32**. R49's development assembly can use
qualified components before R51. R54 must not wait for P30/G4, and P27/P28
producer/source admission must not wait for the matching-release Create/boot
check it supplies; that installed subcriterion is consumed at P30/R52.

## Priority by downstream effort saved

At source `928761ac`, prioritize bounded units that remove shared blockers or
repeated setup, diagnosis, capture and integration work. This ranks the existing
67 outcomes, without a new queue, score, benchmark or gate. **Do not finish an
entire band before taking an independent useful slice.** Actual prerequisites
still control; unavailable fixtures/provider inputs hold only their consumers.
Canlang IDs identify producer outputs needed by Soda, not assignments or messages
to the Canlang repository.

**First actionable unit: R47's MCP transport-fixture clock boundary.** Bind request
digests to the actually admitted fixture capability instead of recalculating its
deadline in `internal/web/api/factory_mcp_test.go::catalogPayload`. Extend the
existing transport proof across that clock boundary while retaining intentional
stale-scope refusal. [R47](factory-remaining-tasks/05-experience-and-development.md#r47)
records the observed failure: required CI failed during consolidation, followed
by another successful publication cycle taking 201 seconds. This is concrete
avoidable repeat work, not a measured failure rate. The bounded fixture repair
comes before larger integration; it does not require finishing R47, repairing
every Deep-CI failure or changing production admission.
**Fixture criterion closed:** `catalogPayload` now binds the scope digest served by the actually admitted fixture capability, and the new deterministic regression pins the one-second-tick mismatch with intentional stale-input refusal unchanged. Composed R47 qualification remains open.

| Priority / existing outcomes, each counted once | Next sequence or branch | Effort saved, prerequisites and risk |
| --- | --- | --- |
| 1 — R47 (1) | Fix the clock-boundary fixture; triage other recorded failures when they obstruct the selected consumer. Full composition stays late. | Remove an observed false refusal/publication retry through a small test change; no new build system or broad cleanup. |
| 2 — R03–R11, R30, R31, R44; Canlang P09, P14 (14) | Next prefer ready R10 typed ingress and R31 bounded projections. Independent branches: R03 → R04 → R05; R07 pin/driver preparation → R08 → R09; R06 custody. Their outputs release R44, R30 activation and R11 composition. P14 assumption proof and native P09 are independent producer work. | Shared authority, result envelopes, receipts/context and custody avoid repeated diagnosis/setup assumptions. R31 projections can precede live R30. Native/harness proofs cost more and need actual inputs, but early failure prevents downstream rework. P14 retires storage/lifetime risk; a native rewrite is not assumed faster. |
| 3 — R12–R20, R32, R33, R35, R42, R43 (14) | R12/R13 and R14 → R15 → R16 → R17 → R18, then R20/R19; attach R33 output and R35 settlement. Pull R32 persisted communication and R42 query preparation forward; R43 consumes their actual records. | Connecting the existing task journal through `ImportTaskEvents` to Store/checkpoints removes manual result stitching. Reuse one authorized read/comment path for roles and screens. Preserve the combined subscribed newest-first feed and user comments → linked plans/issues/results. Full native acceptance waits for actual consumers. |
| 4 — R21–R29, R34, R36–R41 (16) | Review/correction/checks → merge → dependency release → reassessment/adoption, then stall/delivery verification and recovery. R36's ready slice can start after R10/R14 outputs; formal finish controls join their existing effect owners. | Reduce manual reconciliation on each finding, merge, stall and retry. `ValidateReassessmentAdoption` needs its production caller, not another validator. Authority, spending and crash custody make this higher-risk composition; reuse accepted mechanisms and prove changed joins. |
| 5 — R45, R46, R50; Canlang P15, P16, P25, P26, P27/P28, P29 (9) | Ready R50 can proceed independently. R03/R06/R44 release R45 → R46. Consume P14 → P15/P16 → P25 → P26 → P27/P28 and P29. Share an applicable private-client fixture with P22. | Establish reusable component/tool/image identities and procedures before candidate assembly, avoiding builds that merely discover storage/offline/lifetime assumptions. Native cutover is substantial work. Retain scoped RCX/Miniflare evidence without treating it as replacement proof; remote access holds only its cases. |
| 6 — R48, R49, R54, R51, R52, R53, R55; Canlang P30 (8) | R48 cases as role consumers become ready; complete R47 composition/R49 journey. Then R54 freeze → R51 candidate/unqualified public stage → P30/R52 installed journey → R53 promotion/public readback → R55. | Group bounded provider trials by actual configuration; reuse component proof and one matching candidate to avoid duplicate build/install/capture loops. Real qualification and external authority remain required. P31/G4 aliases R52; P32 aliases R55. |
| 7 — Canlang P08, P10, P22 (3) | Obtain the selected R2/provider contract or actual client/route/trust input when its dependent scope needs it; promote the available concrete slice. | Resolve actual waits without speculative adapters/substitute clients. P22 gates claimed remote access; P08/P10 do not become universal first-app release gates. |
| 8 — BROAD-COMPLEXITY-037, Filetree R02 (2) | Take a cohesive simplification or missing duty map when it materially eases the next selected change; otherwise follow the branches above. | Comprehension benefits are plausible, not measured throughput gains. No whole-tree refactor/audit first; keep the 58-function residual and historical coverage gap explicit. |

Release useful outputs without waiting for siblings. Existing fixtures, journals,
read models and focused checks are the reuse opportunities, not a new capture
service or wrapper framework. Apart from the observed CI cost and inspected
missing caller/read/activation joins, savings are engineering hypotheses.
Provider availability and first native storage/harness results may change the
cheapest next branch. Optional L16/CA/structural proposals stay outside the 67;
this ranking neither adopts them nor reopens accepted components.

## Coverage reconciliation at a01b40ee

This finite check reused the requirement aliases, accepted source records and
current defining callers. It found **five incomplete existing criterion groups,
not five new outcomes**. The 67 parents, priority bands and accepted component
scopes remain unchanged. Their recipes now name the concrete missing consumer
proof or ordering correction:

| Confirmed gap in the task criteria | Existing owner and finite correction |
| --- | --- |
| Public staging was implicit, making Docker qualification appear to precede the public bytes it must consume. | **R51 → R52 → R53**: frozen unqualified public stage and anonymous identity readback → actual public installer/same-image persistence receipt → unchanged promotion and correct target recommendations. [Recipe and existing release contract](factory-remaining-tasks/06-candidate-release-closure.md#r51). |
| The complete journey began at an already reviewed/repository-bound draft, leaving the accepted pre-repository user entry implicit. | **R12/R43/R49**: preserve private exploration and saved review; bind a correction and effective revision through association/explicit creation into real Generate/native work. Viewing cannot provision or spend; bounded explicit exploration remains allowed. [Entry criterion](factory-remaining-tasks/02-prompt-to-pr.md#r12), [composed journey](factory-remaining-tasks/05-experience-and-development.md#r49). |
| New feed work named workflow producers but did not explicitly preserve the existing confirmed-activity/preferences consumer. | **R42/R43**, consuming R13/R19/R26/R27: retain three actual lifecycle summaries, owner-only revisioned display choices, confirmed original closure and unchanged authority/history. [Source-backed criterion](factory-remaining-tasks/05-experience-and-development.md#r42). |
| Comment persistence and origin links did not fully specify the accepted request-to-effect recovery cases. | **R32**, consuming R12/R13/R15 with R31/R42/R43 readback: bind exact discussion/request identity; edited/deleted/inaccessible input, unavailable consumers and lost committed-effect replies yield honest state without duplicate plan/issue/spend. [Recovery criterion](factory-remaining-tasks/04-controls-repair-verification.md#r32). |
| Generic provider/grant qualification omitted explicit per-provider custody and subscription-only launch boundaries. | **R44/R48**, with R41 interruption and R52 installed consumers: distinguish Codex's exclusive stream from Muse's concurrent immutable leases, unknown retirement, revocation scope and no billing/account/auth fallback on actual admitted pins. [Defining owners and acceptance](factory-remaining-tasks/05-experience-and-development.md#r44). |

The remaining inspected joins retain existing owners: native authority →
conditional Forgejo effects under R03–R05/R19–R29; task/role/control →
checkpoint/settlement under R07–R11/R15–R18/R30–R41; Project/helper/network →
component/install/recovery under R06/R44–R46/R50–R53; and Canlang producer →
retained-data/lifecycle → installed app under the existing P recipes and shared
R52/P31/G4, R55/P32 aliases. These are coverage mappings, not fresh acceptance.

Subscription defaults, reply/edit ordering, comment routing and concrete Ralph
asset/character choices remain decisions inside R32/R42/R43, with the combined
newest-first subscribed feed and comment-to-linked-work capability adopted.
P08/P10 provider selection and P22 real-client input remain scoped holds.
Outbound notifications, broad provider parity, automatic backups and optional
library/structural changes remain proposals; no new universal gate follows.
No whole-source/body audit, native run or renewed qualification was performed.
Historical coverage drift remains Filetree R02, and runtime/provider/installed
evidence gaps remain open even where this check found an adequate task criterion.

## Historical planning inputs and preserved evidence

The former Factory baseline/workflow/experience, extension, RCX, frontend,
Canlang parent and filetree task/lane/correctness schedules were absorbed into
this catalog and their existing contract owners. Their pre-consolidation text is
available in [Git at 83cf27a3](https://github.com/LevitateOS/sodaos/tree/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development).
Deleting those duplicate records changes no acceptance or outcome count.


Original planning inputs: Soda `ddd6251bef5bb7e0dba02b269e7e419dcffce5f9` (origin `603f9c23ee548b001ddf4857a6395efca21da005`) and Fountain `c8fea1d293366b457df7608c6f3b9293d0810110` (published source). The owner-requested Soda snapshot `d308e8e9edabb6dba55692eb9250069d04170eb6` now preserves the formerly dirty identity/tooling work, task documents and reservation extraction. Committing those inputs does not qualify them or transfer writer ownership. Current Fountain cleanup changes and its retained stash remain preserved.

- Keep durable inventory/selection, immutable plans, budgets, native collaboration/recovery, authorization helpers and persisted journey views. Complete their missing callers and acceptance; do not rebuild them from historical gap prose.
- Keep scoped RCX native development acceptance at Soda730/Fountaina4fd and earlier NFX source/browser/installed proofs. Recheck changed inputs and joins; do not reopen completed component or frontend redesign as new implementation.
- Preserve the completed approved test-fixture retirement and Fountain pull/source push. Neither closes product importer custody or installed release qualification.
- Fountain full run `39061684838` finished with failure: its unit cleanup job failed and database job `117480774633` succeeded at 20:55:42 UTC. Earlier exact-c8 full/required/admission successes remain historical scoped facts; there is no unconditional current green claim.
- Preserve the original A197/198 artifacts separately from their newer canonical variants. Fountain has a distinct uncommitted two-file cleanup revision; Soda's snapshot includes the corrected reservation extraction, without transactional qualification. The R01 review binds their exact bytes and proposed checks through the central checklist. 183A remains returned, 183B incomplete, and B197 is a partial source map. The eight-file tuple overlay's thirteen selected passing tests do not accept the later increment.

R02 is a newly observed CI regression with no separate legacy-index ancestor. The other tasks carry the indexed requirements; adding this repair does not replace or omit any of the 365 historical rows.

The [independent readiness review](../../.artifacts/muse-factory-workflow/reviews/independent-20261017-01a1470a/full-readiness-review189.json) and [new requirement crosswalk](../../.artifacts/muse-factory-workflow/reviews/independent-20261017-01a1470a/reorganization199/requirement-crosswalk.json) retain the evidence and all **365 overlapping requirement/task/control rows**. Only 15 had detailed individual current observations before this reorganization; the others are not silently accepted. Mapping is coverage accounting, not 365 passed tests or mandatory reimplementation of accepted work.

## Consolidated remaining outcomes — execution stopped

All implementation owners are **unassigned** after release. Former lane labels below preserve responsibility, not an assignment. **Ready slice** means the first useful step described above is available after a new instruction; it is not full acceptance or dispatch. **Dependency held** names missing upstream outputs; the external-input list further scopes native/provider/publication holds. Design proposals and deferred restructuring are separated after the factory rows. Dependencies require the needed output, not complete acceptance of every parent plan.

R01 input/freeze and R02 focused Fountain CI repair (`56a1c17bb0`) are accepted and are not remaining implementation. R02 full-current-CI/native qualification is owned by R03/R47/R52. The 53 factory rows count each parent outcome once; shared native/release and legacy consumers are aliases below rather than additional jobs. The six task packets retain the full finite acceptance conditions.

| ID / source | State / owner | Unmet deliverable; preserved acceptance | Concrete prerequisites / hold | Finite acceptance / release |
| --- | --- | --- | --- | --- |
| [R03](factory-remaining-tasks/01-boundaries.md#r03) | Ready slice; unassigned (C) | Native extension/SDK/background authority qualification; source and 29 real-PG cases accepted. | R01; native: R02. | Current native authority, logout/revocation and package/SDK identities pass. |
| [R04](factory-remaining-tasks/01-boundaries.md#r04) | Dependency held; unassigned (C) | Complete native writer-domain and four conditional-operation recovery witnesses. | R01; native: R03. | Contention, cancellation, lost replies and restart conserve exact effects and custody. |
| [R05](factory-remaining-tasks/01-boundaries.md#r05) | Dependency held; unassigned (C) | Qualify native paged snapshots/graphs; source collectors, brackets and validation accepted. | R01; native: R03, R04. | Complete hidden/later-page inventories, exact revision and graph replay/refusals pass. |
| [R06](factory-remaining-tasks/01-boundaries.md#r06) | Ready slice; unassigned (A) | Native post-crash importer/descendant retirement; 7be597d7 source/cleanup accepted. | R01. | Leader exit, surviving descendant, timeout and lost-identity witnesses retain custody until retirement. |
| [R07](factory-remaining-tasks/01-boundaries.md#r07) | Ready slice; unassigned (A) | Install admitted harness pins and align proof tools; obtain finite funded overlap trial. Pin ruling already settled: use admitted pins first. | R01. | Both actual harnesses prove child overlap/control/cancel/parent-death retirement with exact identity and usage. |
| [R08](factory-remaining-tasks/01-boundaries.md#r08) | Dependency held; unassigned (A) | Native parent attenuation and OS-enforced role isolation. | R07. | Actual forbidden shell reads/writes and parent-only calls refuse; allowed role work succeeds. |
| [R09](factory-remaining-tasks/01-boundaries.md#r09) | Dependency held; unassigned (A) | Real pinned CLI instruction/configuration and private MCP transport proof. | R07, R08. | Both CLIs reach the fixed relay/controller with authenticated request roles and recovery. |
| [R10](factory-remaining-tasks/01-boundaries.md#r10) | Ready slice; unassigned (B) | Finish trusted ingress/consumer and D15/D16 subject wiring; accepted parity and terminal transport slices stay closed. | R01. | Full phase/role/subject/result refusal matrix plus real Store and native consumers pass. |
| [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) | Ready slice; unassigned (B) | Live/native dispatch and atomic settlement qualification; source/common-boundary acceptance retained. | R01, R10; native: R07, R08, R09, R44, R30. | One run/charge under racing/replayed admission; failed work stays charged and retirement releases once. |
| [R12](factory-remaining-tasks/02-prompt-to-pr.md#r12) | Dependency held; unassigned (B) | Real semantic provider qualification; server inputs, plan/Generate UI through cf5887a0 accepted. | R10, R11; native: R07, R08, R09, R44, R30. | Associated draft produces exact bounded semantic results; unassociated/stale/unauthorized inputs launch nothing. |
| [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13) | Dependency held; unassigned (B) | Native chosen-graph adoption qualification; 3e4de021 source repair accepted. | R12; native: R04, R05. | Single choice uses zero comparison calls; funded multiple choice replays once with exact graph readback. |
| [R14](factory-remaining-tasks/02-prompt-to-pr.md#r14) | Dependency held; unassigned (B) | Native current-inventory readiness qualification; cursor/restart/large-queue source proofs accepted. | R01; native: R05. | 256 candidates and later-page prerequisites progress without false readiness or duplicate spending. |
| [R15](factory-remaining-tasks/02-prompt-to-pr.md#r15) | Ready slice; unassigned (B) | Compose selected investigation and exact plan/head admission; journey and immutable old-revision Store proofs already pass. | R11, R14; native: R07, R08, R09, R44. | Only selected work runs; stale source/preparation cannot code; immutable revisions and assignment bindings survive. |
| [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16) | Dependency held; unassigned (A) | Connect native task events to Store and accepted dependency release. | R07, R08, R09, R15. | Independent tasks overlap; a successor starts while unrelated work runs; unknown output releases nothing. |
| [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17) | Dependency held; unassigned (A) | Production trusted task-receipt/check/Git-CAS/Store join. | R16; native: R30. | Real prepared checkpoint recovers across CAS crash/lost reply; forged/dirty/wrong-ancestry inputs refuse. |
| [R18](factory-remaining-tasks/02-prompt-to-pr.md#r18) | Dependency held; unassigned (A) | Complete-plan checked-tree publication gate. | R17. | Incomplete/stale/unchecked plans cannot publish; no-change work needs no fake commit. |
| [R19](factory-remaining-tasks/02-prompt-to-pr.md#r19) | Dependency held; unassigned (B) | Current attributable PR and publication recovery composition. | R18, R20; native: R04. | One candidate produces one attributable PR across restart/retry/lost reply; unexpected refs refuse. |
| [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20) | Dependency held; unassigned (B) | Native tuple/checkpoint qualification only; source conversion and classified historical BaseOID uses accepted. | R01; native: R05, R17. | Trusted checkpoint/current base and stale-observer CAS hold through native outage and reconciliation. |
| [R21](factory-remaining-tasks/03-review-merge-reassessment.md#r21) | Dependency held; unassigned (B) | Bounded base-advance merge-in correction sponsor. | R19, R20, R15; native: R16, R17. | Same PR/ancestry and finite correction allowance survive; no silent rebase, force or stale acceptance. |
| [R22](factory-remaining-tasks/03-review-merge-reassessment.md#r22) | Dependency held; unassigned (B) | Current independent native review and complete conversation journey. | R19, R20; native: R04, R05, R08. | Fresh reviewer covers every page/outdated/human thread; incomplete or silent output cannot clear. |
| [R23](factory-remaining-tasks/03-review-merge-reassessment.md#r23) | Dependency held; unassigned (B) | Current finding/correction/disposition/native resolution composition. | R22; native: R05. | Every current finding is accounted once; stale/unknown/duplicate items refuse and history persists. |
| [R24](factory-remaining-tasks/03-review-merge-reassessment.md#r24) | Dependency held; unassigned (B) | Serving clean-review and finite funded Jev acceptance producers. | R22, R23, R11. | Clean explicit clear uses zero delivery calls; uncertain/correction cases obey cumulative limits. |
| [R25](factory-remaining-tasks/03-review-merge-reassessment.md#r25) | Dependency held; unassigned (B) | Exact-tuple complete native check evidence. | R19, R20; native: R04, R05. | Missing/skipped/cancelled/stale/partial configured checks cannot authorize merge. |
| [R26](factory-remaining-tasks/03-review-merge-reassessment.md#r26) | Dependency held; unassigned (B) | Join current acceptance and authorization at every new merge entry. | R20, R21, R24, R25; native: R04. | Real-PG/native refusal and lost-response reconciliation preserve committed effects without re-merge. |
| [R27](factory-remaining-tasks/03-review-merge-reassessment.md#r27) | Dependency held; unassigned (B) | Composed original-closure readiness release; evidence/authority/CAS/completion source checks accepted. | R26, R14; native: R05. | Two-parent join waits for both original verified closures; reopened/stale closures cannot release. |
| [R28](factory-remaining-tasks/03-review-merge-reassessment.md#r28) | Dependency held; unassigned (B) | Two-completion/two-Jev-decision serving proof; fence/drive/replay PG checks accepted. | R27, R11; native: R05. | Each verified completion consumes one bounded decision; continue launches no investigator and other repos progress. |
| [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29) | Dependency held; Design prerequisite; unassigned (B) | Finish serial adoption/materialization composition; T35–T39, committed-prefix recovery and no-change release accepted. FW32 step driver remains deferred/unmaterialized. | R28, R13; native: R05. | Accepted contract drives exact protected native steps; lost reply/conflict preserves prefix and only confirmed completion releases. |
| [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) | Dependency held; unassigned (A) | Qualify live MCP authority/lifecycle, including role communication. | R08, R09, R10. | Cross-scope/expired/revoked calls refuse; replay/rebind cannot renew authority or expose secrets. |
| [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31) | Ready slice; unassigned (B) | Complete four canonical bounded read controls and live readback. | R30. | Authorized context/evidence/status/receipt are complete or explicitly incomplete and spend nothing. |
| [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32) | Ready slice; unassigned (B) | Planning finish consumers, eight-verb communication and human comments usable for linked plans/issues. | R30, R31, R12, R13, R15. | Formal controls, eight verbs and human comments persist with correct authorship; derived plans/issues link to discussion and report effects/failures without granting authority. |
| [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33) | Dependency held; unassigned (A) | Builder output/finish to trusted checkpoint and singleton Fixer reuse. | R30, R31, R16, R17, R18. | Racing proposals yield one trusted checkpoint; failed checks or uncertain retirement cannot advance. |
| [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34) | Dependency held; unassigned (B) | Inspector/Fixer finish to actual review/correction effects. | R30, R31, R22, R23, R24, R33. | Explicit complete clear, dispositions, native readback and observed retirement drive the common consumers. |
| [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35) | Dependency held; unassigned (A) | Incomplete/handoff/terminal settlement composition. | R30, R31, R11. | Stop/submission/reconnect/restart races conserve ownership, child custody, capacity and charges. |
| [R36](factory-remaining-tasks/04-controls-repair-verification.md#r36) | Ready slice; unassigned (B) | Generation advancement on verified delivery and full episode consumer composition. Serving detector/events/isolation/order accepted through 3af53d09. | R14, R10. | Ordinary waits do not investigate; complete independent components progress and fully delivered outcomes enter verification. |
| [R37](factory-remaining-tasks/04-controls-repair-verification.md#r37) | Dependency held; unassigned (B) | Strategist proposal/Jev choice and later repair-cycle adoption; cycle-1 episode/charge joint accepted. | R36, R29, R11; native: R07, R08, R09. | Restart/Retry/graph reshuffle cannot refill allowances; material evidence alone permits bounded next cycle. |
| [R38](factory-remaining-tasks/04-controls-repair-verification.md#r38) | Dependency held; unassigned (A) | Trusted Test Pilot manifest adoption and exact integrated-artifact checks. | R10, R13, R18, R27; native: R07, R08, R09, R30, R44, R45. | Real approved fixture/current bytes and trusted host receipts pass P9; forged/stale/missing checks refuse. |
| [R39](factory-remaining-tasks/04-controls-repair-verification.md#r39) | Dependency held; unassigned (B) | Verified app completion and bounded delivery-repair chain. | R38, R13, R29, R37. | All accepted outcomes plus exact verification complete an intent; missing fixture waits and two-run ceiling survives retry. |
| [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40) | Dependency held; unassigned (B) | Strategist/Test Pilot finish effects and seven-profile instruction/context matrix. | R30, R31, R29, R37, R38, R39. | Authenticated submission reaches durable permitted effect/readback/visible state with refusal/replay coverage. |
| [R41](factory-remaining-tasks/04-controls-repair-verification.md#r41) | Dependency held; unassigned (A) | Composed intervention/withdrawal/crash recovery. | R06, R11, R16, R26, R30, R35, R44. | Whole execution retirement precedes custody return/takeover; committed effects and human work survive. |
| [R42](factory-remaining-tasks/05-experience-and-development.md#r42) | Ready slice; unassigned (C) | Extend canonical read model with workflow producers, subscriptions and combined newest-first role/verb/comment feed. | R10; native: R13, R17, R24, R28, R29, R31, R32, R33, R34, R35, R36, R37, R39, R40. | Bounded authorized ordered/revisioned API survives reload/restart and distinguishes reports from verified effects. |
| [R43](factory-remaining-tasks/05-experience-and-development.md#r43) | Ready slice; unassigned (C) | Fresh Lit Work/Spaces feed and changed native joins; reuse suitable independent Ralph code/assets, seven roles and stable reading/drafts. | R42; native: R03. | Real persisted native-entry/navigation/reload/account/keyboard/geometry and exact terminal object assertions pass; attribution retained. |
| [R44](factory-remaining-tasks/05-experience-and-development.md#r44) | Dependency held; unassigned (A) | Current Project preparation/provider/broker/human/operator qualification. | R03, R10. | Exact peer/account/grant/rotated-credential refusals and background authority work without harming human services. |
| [R45](factory-remaining-tasks/05-experience-and-development.md#r45) | Dependency held; unassigned (C) | Current component/package/controller/maintenance qualification; scoped RCX development acceptance retained. | R03, R06, R44. | Payload v4/current declared image set, exact bytes, closed admission, offline preparation and rollback readback agree. |
| [R46](factory-remaining-tasks/05-experience-and-development.md#r46) | Dependency held; unassigned (C) | Current optional Tailnet/network qualification; actual external enrollment/routes remain separate. | R45. | Default absence and native enable/disable/restart pass; authorized real client route/trust evidence closes external cases. |
| [R47](factory-remaining-tasks/05-experience-and-development.md#r47) | Ready slice; unassigned (Coordinator) | Current composed API/Start/MCP/PostgreSQL serving and race acceptance. Deep CI 39139499605 residuals: release-build source-grep/CLI drift, Tailnet-page/navigation contract tests, PG topology/wiring and fixture/toolchain cases; wait-slot/schema repairs in 51ba6d2f are accepted. | R11, R13, R17, R20, R23, R26, R29, R32, R33, R34, R35, R37, R39, R40, R41, R42. | Real clean/correction/reassessment/repair/Test Pilot/intervention paths pass without injected acceptance or hidden skips. |
| [R48](factory-remaining-tasks/05-experience-and-development.md#r48) | Dependency held; External + engineering; unassigned (A) | Exact harness/provider transport and first-response semantic trials. | R12, R13, R16, R22, R24, R29, R32, R33, R34, R35, R40, R41. | Funded real Codex/Muse cases meet identity, prompt/result/usage and ambiguity/clean/correction contracts. |
| [R49](factory-remaining-tasks/05-experience-and-development.md#r49) | Dependency held; unassigned (Coordinator) | Complete real development factory plus persisted role feed question/reply/finding/result journey. | R39, R40, R43, R44, R45, R47, R48. | Prompt through native overlap/checkpoints/review/merge/reassessment/Test Pilot and intervention passes with bounded usage and human preservation. |
| [R50](factory-remaining-tasks/06-candidate-release-closure.md#r50) | Ready slice; External + engineering; unassigned (C) | Native protected admission/finalization boundary; 88a2ce7b/f188e13b serving proof and 73fe175b tag repair accepted. | R01. | Real protected valid/forged/failure/cancel/recovery evidence qualifies the mechanism before candidate production. |
| [R51](factory-remaining-tasks/06-candidate-release-closure.md#r51) | Dependency held; unassigned (C) | Coherent current x86_64 candidate after source freeze and real immutable component availability. | R02, R43, R45, R49, R50, R54; Canlang P27/P28 qualified producer outputs. | Source/pins/packages/images/attribution/media are attributable; the unqualified public stage and anonymous readback precede R52 without moving recommendations. |
| [R52](factory-remaining-tasks/06-candidate-release-closure.md#r52) | Dependency held; unassigned (Coordinator) | Exact candidate fresh ISO/Docker, supported upgrade, recovery and installed journey. | R51, R49; Canlang P30 supplies shared G4 evidence. | Installed byte/mode/source identities and every required preserved-data/support case match qualified inputs. |
| [R53](factory-remaining-tasks/06-candidate-release-closure.md#r53) | Dependency held; unassigned (Coordinator) | Promote the staged immutable qualified release and verify actual public bytes/recommendations. | R52. | Human authorizes the prepared batch; protected published digests/downloads equal the qualified candidate. |
| [R54](factory-remaining-tasks/06-candidate-release-closure.md#r54) | Dependency held; unassigned (Coordinator) | One-current-path source/caller/selector/docs/license/pin cutover before candidate freeze. Cleanup shutdown does not close product cutover. | R41, R43, R45, R49, R50; Canlang P27/P28 producer schema and P29 shipping procedures. | All shipping changes precede R51; obsolete paths are evidenced and human work/qualification resources preserved. |
| [R55](factory-remaining-tasks/06-candidate-release-closure.md#r55) | Dependency held; unassigned (Codex) | Independent acceptance of complete mapped factory scope; stopping execution does not accept unfinished backlog. | R46, R47, R48, R49, R50, R51, R52, R53, R54. | Production callers and decisive evidence satisfy all original outcomes/refinements/profiles/controls and concrete external cases. |

### Counts and aliases

There are **67 counted remaining outcome rows**: **53 factory**, **12 Canlang-specific**, and **2 repository maintenance** outcomes. This includes blocked, planned and deferred adopted scope; it is not a ready-work count or a claim of implementation authorization. Every counted row has a named existing ID, unmet deliverable, unassigned/human owner, prerequisite and finite release condition in this document.

| Counted family | Existing IDs | Rows |
| --- | --- | ---: |
| Factory | R03–R55; R01/R02 remain accepted at their defined scope | 53 |
| Canlang-specific | P08, P09, P10, P14, P15, P16, P22, P25, P26, the shared P27/P28 admission bundle, P29, P30 | 12 |
| Repository maintenance | BROAD-COMPLEXITY-037; Filetree R02 source/duty coverage | 2 |
| **Total** | Aliases, substeps, shared consumers and optional proposals excluded | **67** |

The 12 Canlang rows preserve **13 parent P IDs**, with P27/P28 counted once for their matching runtime/tools admission bundle; its image and Project-tool consumers are both required. P14.1–P14.4, P15.1, P16.1–P16.2, P25.1, P26.1 and P27.1 refine their existing parent rows. P31/G4 and P32 remain unmet criteria but are shared qualification/closure aliases, owned once by R52 and R55 respectively below. Thus all **15 unmet Canlang parent IDs** remain visible without counting a second candidate or closure. The 58 residual Go functions are the scope of one complexity outcome, not 58 extra rows. The historical 365-row crosswalk and 80-slice inventory describe requirement coverage, not additional assignments.

### Shared external inputs and settled decisions

- **R07/R08/R09/R16/R30/R48:** engineering supplies the admitted Codex binary and aligned proof tools; the human supplies a finite funded provider reservation for actual overlapping children and semantic trials. For Muse the owner directs the current stable build instead of a frozen pin: the installed build matches the `muse-stable` channel (verified 19 October 2026), superseding the earlier frozen Muse pin for this proof. Version numbers stay out of this record; exact build identities are observed in trial evidence. The old “pin decision needed” hold is superseded; installation and funding remain missing evidence inputs, not permission to run trials during cleanup.
- **V03 → R44/R48/R49:** the human/provider must complete normal managed Muse device-account approval; TestOnly credentials and mock accounts do not establish normal enrollment. Release condition is successful normal managed account admission with exact custody/return proof.
- **R46 / Canlang P22:** the human supplies the selected real LAN/Tailnet client, authorized enrollment and DNS/client trust where needed. Core local/native acceptance does not await an optional Tailnet purchase. Real client access closes only the named external route cases.
- **R50/R51/R53:** release owner prepares actual immutable Fountain/package/image/source inputs and protected collector access. Human publication authorization applies to the concrete component/product batch. Anonymous URL/digest/source readback, qualified supported upgrade source and exact candidate observations close the affected gates. Source push is not product release; R50 mechanism proof precedes R51 and must not acquire a dependency on future candidate bytes.
- **GUIDANCE-01–08:** the existing [decision register](ideal-filetree-plan/review-assignments.md#guidance-conflicts-and-controlling-decisions) and current AGENTS.md control. Preserve Go servers, Rust system/privileged owners, x86_64 scope and intentional page-design coverage. No language port, React integration, new coordinator or backlog dispatch follows from this reconciliation.

<a id="other-distinct-outcomes-and-shared-consumers"></a>
### Additional counted outcomes

Canlang P IDs are namespaced below to avoid confusing them with factory R IDs. Historical Miniflare receipts and accepted app/domain/UI work stay accepted at their recorded scope. The direct-workerd replacement is planned, with no executable replacement proof. Its substep IDs refine the existing P IDs; they are not extra parent outcomes. Each row in this table is counted once above and is unassigned unless a human input is named. Shared release/closure consumers and optional choices are recorded separately below.

| Existing ID / source | State / owner | Unmet deliverable | Prerequisite / hold | Finite acceptance / release |
| --- | --- | --- | --- | --- |
| [BROAD-COMPLEXITY-037 / complexity policy](complexity.md) | Ready slice; unassigned | Residual 58 Go functions above 20; max 94 in BuildDispatchPrompt. Rust/TypeScript have zero above 20. | Retain non-increasing baseline/exceptions and actual behavior; same-file custody before any future edit. | Cohesive simplification plus affected checks meets <=20 or a narrow justified cap; no baseline regeneration. |
| [Filetree R02 / source coverage](ideal-filetree-plan/coverage/README.md) | Ready slice; unassigned | Accumulated source-to-duty/desired-tree coverage beyond retained scoped pins. | Historical components-v4 and pre-730fc205 drift remains explicitly unreconciled; current batch is mapped in the living plan. | Every affected leaf/changed duty has one current owner/destination and large-file concern; historical body/native pins stay distinct. |
| [Canlang P08](#canlang-p08) | Decision held; product/runtime owner input | Asynchronous R2-compatible blob seam. Native SQLite Files acceptance remains retained. | P04/P06 contracts; R2/provider scope selection. | Actual async local/R2 adapters preserve finalized integrity and explicit failures; no R2 claim from native chunks. |
| [Canlang P09](#canlang-p09) | Ready slice; unassigned producer owner | Full retention/cleanup and interrupted-upload operational proof. | Existing authenticated native routes, 10 MiB/lost-ack proofs; P08 only for R2 claims. | Real interrupted uploads recover or fail visibly and retained file authorization/bytes survive replacement. |
| [Canlang P10](#canlang-p10) | Decision held; product/provider owner input | External provider dispatch, uncertain outcomes and reconciliation. | Selected provider contract/credential input and finite authorization; accepted internal durable occurrence/replay remains closed. | Actual provider effects/retries/unknown outcomes preserve stable identities without duplicate work or credential leaks. |
| [Canlang P14 / P14.1–P14.4](canlang-soda-os-implementation-plan/production-workerd.md#implementation-packets-and-dependencies) | Ready slice; unassigned producer owner | Direct-workerd native storage/exec-guard, D1 provider, configuration/transport/lifetime and production CLI. | P14.1 actual native decision gate first; P14.2 and P14.3 outputs feed P14.4. | Pinned real D1/DO/alarm writes survive kill, atomic failure and parent death; exclusive writer and actual app/CLI contracts pass. |
| [Canlang P15 / P15.1](canlang-soda-os-implementation-plan/production-workerd.md#implementation-packets-and-dependencies) | Dependency held; unassigned | Replacement runtime G1/G2 acceptance. | P14.2–P14.4; retain historical Miniflare first-app results. | Actual browser/MCP/Files/work, full/missing/read-only storage, concurrency/restart and provider matrix pass on exact new runtime. |
| [Canlang P16 / P16.1–P16.2](canlang-soda-os-implementation-plan/production-workerd.md#implementation-packets-and-dependencies) | Dependency held; unassigned | Production-only offline archive and image closure. | P14.1 graph contract, P14.4 runtime and P15 evidence; P16.1 precedes P16.2. | Exact binary/exec guard/licenses and app run offline as constrained UID with no Miniflare/dev closure; data/lifetime preserved. |
| [Canlang P22](#canlang-p22) | External input held; human client/trust input | Physical LAN/Tailnet routing and client trust rollout. | P01/P21; retained local/CoreOS Caddy HTTPS proof stays accepted. | A real selected private client reaches the exact app with separate app/admin origins and valid trust. |
| [Canlang P25 / P25.1](canlang-soda-os-implementation-plan/production-workerd.md#implementation-packets-and-dependencies) | Dependency held; unassigned | Replacement retained-data cutover and fresh restore under the accepted explicit-stopped-backup policy. Automatic backup/RPO/RTO choices are optional proposals below, not prerequisites for this proof. | P15.1/P16.2; snapshot a stopped baseline into a disposable clone, never overwrite newer data. | Identical app retains identity/D1/Files/work/WAL/alarm metadata through new runtime and restore; second writer excluded. |
| [Canlang P26 / P26.1](canlang-soda-os-implementation-plan/production-workerd.md#implementation-packets-and-dependencies) | Dependency held; unassigned | Replacement native G3 operational lifecycle/upgrade recovery. | P25.1; P22 holds only the physical/private-client cases. | Host/Caddy/browser/MCP survives Project stop, reboot, same-bundle replacement and restore; incompatible changed-artifact migration stays refused until proved. |
| [Canlang P27 / P27.1; P28](canlang-soda-os-implementation-plan/production-workerd.md#implementation-packets-and-dependencies) | Dependency held; unassigned | Refresh runtime/tools admission, attribution, Node pins and matching-release Project Create/boot. | P26.1 accepted producer, archive/image schema and real availability; shares factory R51 candidate. | Exact checksum/image/tool labels agree, mismatch refuses and old admitted generation remains operable. Candidate outcome counted under R51. |
| [Canlang P29](#canlang-p29) | Dependency held; unassigned | Update existing procedures for replacement measured behavior. | P22/P25/P26; current historical operator guide remains accepted for its actual runtime. | Compile/deploy/access/secret/maintenance/backup/restore procedures match real replacement operations and capability limits. |
| [Canlang P30](#canlang-p30) | Dependency held; unassigned | Integrated two-user installed UI/API/runtime journey. | P27/P28 producer outputs, P29 and R51 matching candidate; reuse G3 components without claiming composed proof. | Project compile, exact deploy, two-user browser/MCP, Project independence, reboot/work/replacement/fresh restore pass on current bytes. |

### Canlang execution recipes

These recipes own the next steps for the 12 counted Canlang rows and the shared
P31/G4 and P32 aliases. P27/P28 remain one outcome. The [runtime contract and
substeps](canlang-soda-os-implementation-plan/production-workerd.md) own provider,
process and image details; the [application guide](../guides/canlang-applications.md)
owns operator procedures. Historical Miniflare acceptance does not qualify direct
workerd. Deleting duplicated planning documents does not satisfy P31/G4 or P32/R55.

<a id="canlang-p08"></a>

#### Canlang P08 — Define and implement the asynchronous blob seam

**Executable remainder:** Decision held: the product/runtime owner must select the R2-compatible API subset and target (including credentials, integrity/finalization, deletion and uncertain-write behavior). Start in Canlang `packages/files/src/ports.ts`, `upload/`, `finalize/` and `host/sqlite-core.ts`; preserve `createDurableSqliteFileStore` for the accepted native target. After selection, introduce the async seam with its real local adapter/callers, then the selected R2 adapter. Qualification requires delayed/rejected/unknown writes, digest/finalization replay and deletion against that actual provider. P09's native lifecycle is independent; broad Cloudflare service parity is not this slice.

<a id="canlang-p09"></a>

#### Canlang P09 — Connect authenticated file routes and lifecycle

**Executable remainder:** Ready native-target slice: start at Canlang `packages/files/src/retention/index.ts::{runRetentionBatch,createFinalizedFileCleanup}`, `upload/`, `finalize/` and `host/durable-sqlite.ts`. Use existing authenticated compiled-app routes to interrupt upload before and after finalization, resume or report a durable failure, expire unfinalized data and prove bounded cleanup never removes a retained/authorized object. Release lifecycle results to P15; extend the existing durable-files/native driver rather than rebuilding routes. Check exact bytes/authorization after restart and replacement. P08 gates only the future async/R2 variant; replacement-runtime repetition is limited to its changed provider join.

<a id="canlang-p10"></a>

#### Canlang P10 — Connect durable work and external provider dispatch

**Executable remainder:** Decision held: the product/provider owner selects one real external effect, provider endpoint/API contract, least-privilege credential reference, finite trial budget and supported lookup/idempotency/reconciliation behavior. Start in Canlang `packages/work/src/{dispatch,recovery,receipt}` and `packages/services/src` at the selected adapter; existing internal occurrences remain accepted. Then join durable claim → external dispatch → recorded response/unknown outcome → reconciliation without blind duplicate effects. Exercise lost acknowledgement, retry, cancellation and restart with that actual provider. If the API offers no recoverable identity, name the manual/uncertain outcome instead of promising exactly-once remote execution; unsupported capabilities continue to refuse.

<a id="canlang-p14"></a>

#### Canlang P14 — Implement the Canlang Soda target assembler and launcher

**Executable remainder:** Ready assumption-proof slice; follow P14.1–P14.4 in the replacement chapter. Existing Canlang `soda-run.ts::{acquireWriter,provisionSodaStorage,startSodaRuntime}` and `soda-lifetime.ts::superviseWorkerd` retain the current ownership; proposed direct-runtime files are not existing implementations. First prove actual pinned D1/DO/alarm storage and inherited lock/parent-death behavior in the smallest disposable native fixture. Only that observed result releases P14.2 provider and P14.3 process/transport work, then P14.4 CLI/Identity administration. Failure returns the specific storage/lifetime assumption for engineering resolution; do not proceed to image/orchestration construction or relabel historical Miniflare G1 as replacement proof.

<a id="canlang-p15"></a>

#### Canlang P15 — Qualify the complete runtime on native Rocky

**Executable remainder:** Dependency held on P14's provider/configuration/production CLI outputs. Extend Canlang's existing `verify-native-runtime.mjs` and `verify-soda-host.mjs` against the original compiled app: first real two-user browser/MCP and guarded D1 writes, then Files/work and storage/lifetime failures. Include P09's missing lifecycle cases; P08/P10 are required only for claims about those selected providers, not the adopted first-app subset. Release exact direct-runtime G1/G2 behavior and supported/rejected inventory to P25. Preserve historical application semantics; partial provider evidence or a process kill is not full image, power-loss or installed proof.

<a id="canlang-p16"></a>

#### Canlang P16 — Finalize the application image

**Executable remainder:** Use P16.1/P16.2 as two slices. Canlang's pack-tools/installed producer owns the production-only archive after P14.4; Soda's `system/containers/canlang/{Containerfile,Application.Containerfile}` and `verify-runtime-image.mjs` consume it. Stage the prebuilt exec guard, direct binary and host modules with complete checksums/licenses and verified native ABI; then exercise offline commands in the constrained runtime/app images. Release immutable archive/image contracts and actual bytes to P25/P27/P28. P15 and image tests can progress independently after the runtime exists, but both must pass before retained-data acceptance. Missing development imports, startup downloads, wrong executable mode or mismatched app/runtime identity fail visibly.

<a id="canlang-p22"></a>

#### Canlang P22 — Implement and prove private application ingress

**Executable remainder:** External-input held: the human/operator selects an actual LAN or Tailnet client, authorized route/enrollment and concrete DNS/TLS trust installation. Start at Soda `lib/host/src/application/ingress.rs` and `system/containers/canlang/verify-ingress.mjs`; share applicable network setup with Factory R46. From that client, prove exact private app hostname, valid trust, authenticated app access and denied cross-app/admin authority, then disabled/revoked route behavior. Record only the claimed access modes. Existing local/CoreOS Caddy proof stays accepted; no public domain, purchased Tailnet or new proxy architecture is required for local mode.

<a id="canlang-p25"></a>

#### Canlang P25 — Implement stopped backup and fresh-volume restore

**Executable remainder:** Consume P15.1 native runtime and P16.2 image bytes. Start at Soda `lib/host/src/application/snapshot.rs`, `lib/host/src/bin/soda-canlang-storage.rs` and Canlang's existing recovery driver. Snapshot the stopped historical runtime into a disposable clone, reopen with the identical bundle under direct workerd, mutate/restart and restore into another fresh identity. Compare D1/Identity/session/grant/File/work facts, WAL/alarm metadata and acknowledged writes; prove second-writer refusal through shutdown. Release a measured supported data-cutover/restore contract to P26. Do not overwrite newer data, silently migrate a changed artifact or make unset automatic-backup/RPO/RTO policy a prerequisite.

<a id="canlang-p26"></a>

#### Canlang P26 — Qualify native lifecycle and upgrade recovery before release integration

**Executable remainder:** Consume P25's supported retained-data contract and the exact P16 runtime/tools fixture. Use Soda `lib/host/src/application/runtime.rs::operate`, ingress and vendor unit with the existing G3 component driver. Prove start/stop and stopped intent across reboot, Project-stop independence, same-bundle replacement, mismatch refusal before effects, and fresh restore with actual browser/MCP/Files/work. Local component work can proceed before P22's physical-client input; full claimed remote-access acceptance waits for that specific result. Release qualified component producers to P27/P28/P29. Changed-artifact migrations remain rejected; image rollback is not database rollback.

<a id="canlang-p27"></a>

#### Canlang P27 — Integrate runtime images into Soda release delivery

**Executable remainder:** Shared P27/P28 outcome: first consume P26's component-qualified producer/schema in Soda's existing Canlang fetch/admission, `system/project/canlang-release.json`, maintenance pin and release image/inventory readers. Update exact archive/runtime/tools/Node identities and attribution together; prove mismatch refusal and old admitted generation inspection/start/stop. Release producer/source admission to R54/R51. Then use R51's matching candidate for P28 fresh Project Create/boot/compiler/preview/examples/LSP and retained same-root maintenance during P30/R52. This second subcriterion cannot be a prerequisite to building its own candidate. Existing seven-role delivery code and accepted offline tooling are retained.

<a id="canlang-p28"></a>

#### Canlang P28 — Integrate Project tool assets into release staging

**Executable remainder:** Shares the single P27/P28 outcome and its executable sequence above. Start at `system/project/Containerfile`, `cmd/soda-project-maintain/src/canlang.rs` and the admitted tool archive readers. Preserve the development Miniflare closure while production consumes its separate direct-workerd subtree. Fresh matching-release Project Create/boot and retained explicit same-root refresh must consume the same qualified tool identities, reject active/mismatched maintenance and preserve preview data. Reuse compiler/example/LSP source and offline-image results; prove only the changed installed consumer at P30/R52, without a second candidate or counted outcome.

<a id="canlang-p29"></a>

#### Canlang P29 — Write the operator and developer procedures

**Executable remainder:** Consume P22's actual access limits, P25 restore procedure and P26 measured component behavior. Update existing `docs/guides/canlang-applications.md` and affected development/release references for compile/preview, bootstrap, explicit deploy/access, supported secrets, maintenance, stopped backup/restore and same-bundle upgrade/refusal. Compare each changed instruction with its existing executable owner and result; do not invent a new runbook or automatic recovery promise. Release shipping instructions to R54 before R51. P30 can append measured installed evidence without changing product bytes; missing client proof is documented as held, not advertised as working.

<a id="canlang-p30"></a>

#### Canlang P30 — Demonstrate the integrated small-team journey

**Executable remainder:** Consume P27/P28 producer admission, P29 procedures and R51's single matching candidate; coordinate its installed fixture with R52. Use the existing Spaces application controls (`frontend/spaces/sodaspaces-applications.ts`), authorized application API and host runtime, not direct native commands alone. Compile in a fresh Project, explicitly deploy the exact bundle, use two users/browser/MCP, stop the Project, reboot, run missed work, replace and restore fresh data. Include P28's matching Create/boot tools and P22's claimed client access. Release actual UI/API/native observations to R52/P31/G4; failed driver runs remain nonqualifying and unchanged candidate bytes can be reused.

<a id="canlang-p31"></a>

#### Canlang P31 — Qualify the native Soda release candidate

**Executable remainder:** Shared alias of Factory R52: consume P30 observations and bind them to R51 candidate image/tool/app identities in the existing native qualification owner. It is one installed release qualification, not a second candidate or a blanket rerun of G1–G3.

<a id="canlang-p32"></a>

#### Canlang P32 — Absorb completed planning into canonical owners

**Executable remainder:** Shared alias of Factory R55 after P31/G4. Reconcile durable measured contracts, procedures and licensing against the qualified release, then close the remaining outcome once. R54 owns the earlier shipping-source freeze. The redundant parent plan has been absorbed during documentation consolidation; that removal neither accepts G4 nor completes P32/R55.

### Requirement aliases and shared consumers — not additional outcomes

These are the remaining criteria of their existing parents, with one counted outcome owner. A source-complete alias stays accepted at that layer; the linked native/installed consumer remains open. No old checkbox, historical next-step paragraph or schedule creates another queue.

| Existing plan / IDs | Counted owner / exact remaining boundary | Prerequisite / finite release condition |
| --- | --- | --- |
| [Canlang P31 / G4](#canlang-p31) | **R52**, consuming Canlang P30: exact installed native candidate tools/runtime/app qualification | P30 and R51's matching candidate; G4 must bind the actual packaged identities and complete installed application evidence. Not another candidate-production outcome. |
| [Canlang P32](#canlang-p32) | **R55** final closure, with **R54** owning pre-candidate shipping contracts | P31/G4 accepted; reconcile remaining measured contracts/procedures/attribution against the qualified release. Earlier absorption of the redundant parent plan does not complete this outcome or infer G4 from a merge. |
| [Factory baseline FT/ST/RT/UX/FX and original D01–D26](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/factory-implementation-plan.md) | **R03–R55** via the retained 365-row primary-owner crosswalk | Reuse accepted source/fixture/installed scopes. Exact current native/installed/release gaps must satisfy their linked row; a historical whole-plan completion demand adds no job. |
| [FW01–FW50 / D01–D17 / P1–P9 / T01–T64](factory-workflow-contracts.md), [frozen contracts](factory-workflow-contracts.md) | **R03–R55**; seven profiles/fourteen controls are consumers of **R07–R09/R16–R18/R30–R41/R48** | The retained dependency graph and actual needed outputs apply. Close each authenticated producer/effect/readback and native refusal case at its real evidence layer; do not count controls again. |
| [Experience direction](../design/factory-feed-and-ralph-integration.md) and [role/verb feed design](../design/factory-feed-and-ralph-integration.md) | **R12/R13/R30/R32/R42/R43/R49** | Accepted intent/Generate source remains closed. Combined subscribed activity, newest-first posts and user comments usable for linked plans/issues are adopted. Subscription defaults/trigger details, MCP representation and artwork remain choices within these outcomes; actual producer-to-browser/effect acceptance closes them. |
| [Fountain extension transition](../architecture/trust.md#fountain-consumption-boundary) | **R03–R05/R43/R45/R51/R52/R54** | Current exact SDK/native authority, composition and candidate evidence; preserve completed native replacement and avoid restoring obsolete OAuth/shell paths. |
| [RCX / RCX-2 FX01–FX08 and G1–G4](testing.md#release-component-regression-boundaries) | **R03–R06/R45/R46/R50–R53** | Eight source repairs accepted; close only the uncovered real native/component/public-byte criteria. Required immutable inputs precede R51; qualified unchanged product publication follows R52. |
| [NFX 57-task correction](testing.md#native-frontend-composition) / FX09 | **R03/R43/R52/R54** | Later natural-link/history/terminal/installed receipts supersede older remainder snapshots. Prove changed native joins/new feed and exact new candidate; no replay of all 57 tasks. |
| [Fountain/Soda 19-task correctness continuation](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/fountain-soda-correctness-tasks.md) | **V03 → R44/R48/R49**; other 18 tasks retain their accepted scopes | Human/provider completes normal managed Muse device-account approval. TestOnly account fixtures do not close this boundary; normal exact custody/admission/return evidence does. |
| [Filetree A01–A08 / I/P/S slices](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-tasks.md#a--projects-identity-and-spaces) | **R06–R09/R41/R44/R48/R49/R52** | Only remaining native custody/isolation, Project/provider/human and installed cases. Accepted M/C/V, CON-M01/02 cutoff, SIMP-FD-1 and source adapters stay closed; real corresponding witnesses release the mapped consumer. |
| [Filetree B01–B08 / F/G slices](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-tasks.md#b--factory-and-forgejo) | **R03–R05/R10–R43/R47–R49/R52**, according to the existing requirement's domain | Current authority/accounting/dispatch/native effects and browser composition criteria are the existing factory rows. Source receipts do not close those parents; aliasing does not reopen source cuts. |
| [Filetree C01–C11 / N/O/D/H slices](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-tasks.md#c--platform-release-and-verification) | **R06/R44–R46/R50–R54**; CI evidence under **R47** | Current importer, kernel/openat2, native provider/IPC/network/component/media/installed evidence. Preserve accepted parser/OCI/Butane/observer/media/notice scopes; exact required runtime and candidate evidence closes the residual. |
| [Filetree Q1–Q9](https://github.com/LevitateOS/sodaos/blob/83cf27a3c46811617a8615e25364ee9d1ead3df0/docs/development/ideal-filetree-plan/implementation-tasks.md#gates-and-readiness) | Q1/Q2/Q3 → **R41/R44/R48/R52**; Q5 → **R03–R05/R19–R28**; Q6/Q7 → **R45/R50–R53**; Q8 → **R11/R19–R29/R37/R41**; Q9 → **R44/R45/R52** | Existing resolved source/profile decisions remain settled. Only their actual current producer/native/installed boundary is required; Q4 optional policies are proposals below. No whole-packet or old design-decision gate is added. |
| [Library L00–L18 and post-adoption findings](ideal-filetree-plan/library-adoption.md) | Retained completed source scopes; residual **R44/R45/R46/R48/R52**, plus **Filetree R02** for coverage | Exact shipped curl/Bash/Podman/kernel, TLS/provider, generator/service and native caller identities must be observed where required. L16.G is complete; optional L16 conversion and CA narrowing are proposals below. No new library-adoption or generic-engine split queue. |
| [Deep regression CI and composed evidence](testing.md) | **R47** | The recorded 39139499605 source-grep/CLI/navigation/PG/fixture residuals need specific triage and affected proof. Required publication CI passed at 476742e0; that does not erase Deep CI gaps or turn them into another cleanup gate. |
| [Filetree inventory/body/desired-tree drift](ideal-filetree-plan/coverage/README.md) and chapter-local coverage holds | **Filetree R02** | One source-to-duty/desired-tree coverage outcome absorbs these consumers. Preserve earlier structural/body/native pins; update only real missing coverage and do not claim full review from path counts. |
| [Complexity baseline and batch receipts](complexity.md) | **BROAD-COMPLEXITY-037** | One residual reduction outcome covers the recorded 58 Go functions and applicable unchanged baseline/caps. Accepted reductions and Rust/TypeScript <=20 remain accepted; no rescan or refactor is dispatched here. |

### Optional proposals and unresolved policy choices — excluded from the 67

| Existing ID / source | Owner / missing decision | Finite disposition |
| --- | --- | --- |
| [L16 / RED01 optional matcher](ideal-filetree-plan/library-adoption.md) | Unassigned; demonstrated value and explicit adoption selection | Keep current matcher and accepted L16.G bounds; select a bounded conversion only if its construction/streaming and caller criteria justify it, otherwise remain deferred. |
| [CA-ALGORITHM-PROFILE-1](ideal-filetree-plan/execution-findings.md) | Unassigned; optional narrowing needs an actual selected producer profile | Existing RSA/named-curve ECDSA/Ed25519/SHA-2 policy stands. Retain original DER/TBS/CA-use rules; actual selected-producer proof is required before narrowing. |
| [A34/C41/browser seams and Q4 optional creation/selection/history policy](#optional-proposals-and-unresolved-policy-choices--excluded-from-the-67) | Unassigned; explicit scope selection and demonstrated current need | Park optional structural seams and P01/N05 creation intent, N04/N06 selection/retry and I10 history-retention choices. Retain established native policy; no source/port assignment arises from historical parked proposals. |
| [Canlang automatic backup/RPO/RTO targets and eventual wider service breadth](#optional-proposals-and-unresolved-policy-choices--excluded-from-the-67) | Human policy choice, unassigned implementation | Existing explicit stopped backup and first-app supported/rejected matrix remain adopted. Set frequency/loss/recovery and additional provider requirements concretely before creating implementation scope; these choices do not hold accepted first-app behavior or P25's stopped-restore proof. |

### Superseded schedules and accepted work

FW/FT/ST/RT/UX/FX, RCX/RCX-2, NFX and the 365-row legacy crosswalk are requirement/evidence references. They are not additional remaining queues. RCX-2 accepted source/native development at Soda `730fc205` / Fountain `a4fd2221` is preserved; current component/native/public-byte gaps live once in R03–R06/R45/R46/R50–R53. NFX accepted source/browser/installed work through the later October 13 continuation supersedes its older 23/34 snapshot and stale rows; only changed joins/new feed and current candidate proof remain under R03/R43/R52/R54. The earlier extension transition and experience-plan baseline are historical; native extension and intent/Generate source work already accepted is not implementation backlog. Canlang historical Miniflare results do not qualify the planned direct-workerd replacement. Filetree whole-tree/body/independent/native review pins remain historical and are not renewed by merging or by this documentation pass.

### Wind-down custody and retained outputs

The only registered checkout is the clean primary `/home/vince/Projects/sodaos` on `main`; no secondary managed worktree, local task branch, remote task branch or open PR existed after fetching current origin refs. No redundant history needed deletion. Original commits, Codex recovery refs and retained former-worktree outputs remain recoverable. No stash or authored deletion was discarded.

Final released source includes `97d59c3f` (feed and planned workerd scope, with schema-test formatting), `3af53d09` (repository-isolated/observation-ordered stall events) and `51ba6d2f` (seven wait results and schema mirrors). The existing integration already contains complexity `6bf6cecf`, its `221d4ae0` simulation correction and `ad4603f9` not-found repair. Producer focused real-PG stall/prompt/source and Rust drift/payload checks are reused. Required push qualification and final synchronization are recorded by the existing push gate; Deep regression CI acceptance remains separately open.

Both current Muse goals were paused at explicit human stop; unfinished factory/complexity goals are not marked complete. Session `01a14c0e-92fb-7f61-a058-ffe06bda80ac` and session `01a14de7-5843-79f3-9f0f-88cc4307b16c` released commands/workers; ten stale exact coordinator claims were removed after release. Their TUI PIDs 1480234/2230162 exited on TERM; the two verified dead, unattached owned panes were closed individually. The host socket/process inventory found only these two tmux servers; no shared server or user viewer was killed. Existing system timers and Cursor loops were unrelated and retained.

The exact disposable PostgreSQL fixture `soda-pg-test-1759348-6377` was stopped after command release. Its existing auto-remove lifecycle removed the test container; it was not application or user data. Retain the pg-review-009 diagnostics/tool configuration, both run directories and terminal logs, all broad037 checkpoint/evidence material, integration manifest, closed checklist/monitor, `/home/vince/preserve-foreign-20261018/`, existing former-worktree retained outputs, installation media and caches. No required installation output or shared cache was pruned, and no primary dependency points to a removed checkout. Root-owned release directories remain retained rather than guessed disposable.

Local supervision relay `you-are-a-one-shot-message-relay-invoked-by-my-external-scheduler-every-25-minut` was read back as PAUSED by its existing local owner. The older named Soda relay and other Soda-only wakes were absent from that inventory; unrelated automation was retained. No host Soda/Muse check timer or mailbox wake was identified. The unrelated `sodaos-local.service`, human terminals, repositories and services remain preserved.

<a id="task-list"></a>
## Task definitions and dependency reference

### Owner addition on 18 October 2026 Ralph4days absorption

The owner requested substantial absorption of Ralph4days's useful code, artwork,
components and interaction flows into Soda OS. The intended finished Soda factory
procedure leads. The agreed design and adopted communication vocabulary are in
[Factory feed and Ralph4days integration](../design/factory-feed-and-ralph-integration.md).
These are rough scope additions for further refinement, as the owner requested;
they do not freeze API schemas, component layouts or portraits. The owner has
selected a fresh interface in Soda's Lit/TypeScript stack and ruled out React
integration.
This is added scope within the existing outcomes below: retain all 55 task IDs,
their dependency edges, historical requirements and valid completed evidence.
The consolidated outcomes above own current status; the closed checklist retains evidence.

| Owning task | Added deliverable |
| --- | --- |
| [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) | Apply existing single Factory MCP authority/lifecycle to role communication; human comments use authenticated user authority |
| [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32) | Persist typed eight-verb communication and human comments/replies; bind selected discussions to existing planning/issue requests, with formal finish and observed effects distinct |
| [R42](factory-remaining-tasks/05-experience-and-development.md#r42) | Durable role/verb/comment records and subscriptions; one authorized newest-first combined feed, linked to canonical work and exact verification |
| [R43](factory-remaining-tasks/05-experience-and-development.md#r43) | Fresh Lit combined subscribed feed with newest posts first, human comments, linked agent-created work, suitable Ralph code/assets, seven-role authors and stable reading |
| [R49](factory-remaining-tasks/05-experience-and-development.md#r49) | Demonstrate the real combined feed, comments-to-plan/issue journey, question/answer, progress/finding, submitted result and separate verified outcome |

R43 designs a fresh interface in Soda's existing Lit/TypeScript stack using
concrete Ralph flows and host lifetime constraints. Do not integrate React or
require Ralph's existing appearance. Character assignments, layouts and exact
new tool names remain design choices. Evidence
for older implementations does not accept the newly added behavior. Preserve the
historical 365-row crosswalk; these owner additions are accounted for here and in
the owning task cards rather than overwriting historical requirement text.

This is the retained Factory parent-qualification dependency order. The executable next-step guide above and task slices refine its coarse edges, including the shared Canlang candidate path. Stable R IDs identify outcomes; they are not execution positions. The six detailed packets remain grouped by subject for navigation.

This retained dependency graph is not an active schedule. **Read both dependency columns.** Implementation inputs release the named contract/consumer work. Additional gates must pass before native activation or qualification. The level is the earliest full-acceptance depth of their combined graph; it is not a wave barrier. Future execution requires a new instruction; readiness does not authorize dispatch. A slice may consume an existing accepted source contract before its parent native qualification. Identify the actual version/interface in the ordinary change; do not impose a separate contract-freeze ceremony or claim full task acceptance.

| Level | Task and remaining outcome | Accepted implementation inputs | Additional native activation / qualification gates | Lane |
| --- | --- | --- | --- | --- |
| 1 | [R01](factory-remaining-tasks/01-boundaries.md#r01) — Accept the retained inputs and freeze the first work packets | none | none | Coordinator |
| 2 | [R02](factory-remaining-tasks/01-boundaries.md#r02) — Repair the Fountain CI cleanup failure | [R01](factory-remaining-tasks/01-boundaries.md#r01) | none | C |
| 2 | [R06](factory-remaining-tasks/01-boundaries.md#r06) — Repair durable importer acquisition and process retirement | [R01](factory-remaining-tasks/01-boundaries.md#r01) | none | A |
| 2 | [R07](factory-remaining-tasks/01-boundaries.md#r07) — Prove supported native worker control on the pinned harnesses | [R01](factory-remaining-tasks/01-boundaries.md#r01) | none | A |
| 2 | [R10](factory-remaining-tasks/01-boundaries.md#r10) — Complete current phase, subject and result parity | [R01](factory-remaining-tasks/01-boundaries.md#r01) | none | B |
| 2 | [R50](factory-remaining-tasks/06-candidate-release-closure.md#r50) — Finish protected release-evidence admission and finalization | [R01](factory-remaining-tasks/01-boundaries.md#r01) | none | C |
| 3 | [R03](factory-remaining-tasks/01-boundaries.md#r03) — Qualify the current extension, SDK and background-authentication boundary | [R01](factory-remaining-tasks/01-boundaries.md#r01) | [R02](factory-remaining-tasks/01-boundaries.md#r02) | C |
| 3 | [R08](factory-remaining-tasks/01-boundaries.md#r08) — Prove parent attenuation and OS-enforced role isolation | [R07](factory-remaining-tasks/01-boundaries.md#r07) | none | A |
| 4 | [R04](factory-remaining-tasks/01-boundaries.md#r04) — Close native writer-domain and conditional-operation evidence | [R01](factory-remaining-tasks/01-boundaries.md#r01) | [R03](factory-remaining-tasks/01-boundaries.md#r03) | C |
| 4 | [R09](factory-remaining-tasks/01-boundaries.md#r09) — Prove supported instruction placement and private MCP transport | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08) | none | A |
| 4 | [R44](factory-remaining-tasks/05-experience-and-development.md#r44) — Requalify retained Project, provider, human and operator foundations | [R03](factory-remaining-tasks/01-boundaries.md#r03), [R10](factory-remaining-tasks/01-boundaries.md#r10) | none | A |
| 5 | [R05](factory-remaining-tasks/01-boundaries.md#r05) — Qualify authoritative snapshots, pagination and native graph packets | [R01](factory-remaining-tasks/01-boundaries.md#r01) | [R03](factory-remaining-tasks/01-boundaries.md#r03), [R04](factory-remaining-tasks/01-boundaries.md#r04) | C |
| 5 | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) — Qualify one live Factory MCP authority and lifecycle | [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09), [R10](factory-remaining-tasks/01-boundaries.md#r10) | none | A |
| 5 | [R45](factory-remaining-tasks/05-experience-and-development.md#r45) — Qualify current release-component composition and maintenance | [R03](factory-remaining-tasks/01-boundaries.md#r03), [R06](factory-remaining-tasks/01-boundaries.md#r06), [R44](factory-remaining-tasks/05-experience-and-development.md#r44) | none | C |
| 6 | [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) — Finish one atomic subject-aware dispatch and accounting path | [R01](factory-remaining-tasks/01-boundaries.md#r01), [R10](factory-remaining-tasks/01-boundaries.md#r10) | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09), [R44](factory-remaining-tasks/05-experience-and-development.md#r44), [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) | B |
| 6 | [R14](factory-remaining-tasks/02-prompt-to-pr.md#r14) — Finish current large-inventory selection and readiness qualification | [R01](factory-remaining-tasks/01-boundaries.md#r01) | [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 6 | [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31) — Finish all four bounded MCP read controls | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) | none | B |
| 6 | [R46](factory-remaining-tasks/05-experience-and-development.md#r46) — Qualify optional Tailnet and retained networking behavior | [R45](factory-remaining-tasks/05-experience-and-development.md#r45) | none | C |
| 7 | [R12](factory-remaining-tasks/02-prompt-to-pr.md#r12) — Connect the real semantic Product Lead planner | [R10](factory-remaining-tasks/01-boundaries.md#r10), [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09), [R44](factory-remaining-tasks/05-experience-and-development.md#r44), [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) | B |
| 7 | [R15](factory-remaining-tasks/02-prompt-to-pr.md#r15) — Connect selected investigation to immutable executable plans | [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11), [R14](factory-remaining-tasks/02-prompt-to-pr.md#r14) | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09), [R44](factory-remaining-tasks/05-experience-and-development.md#r44) | B |
| 7 | [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35) — Finish incomplete, handoff and terminal settlement semantics | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31), [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) | none | A |
| 7 | [R36](factory-remaining-tasks/04-controls-repair-verification.md#r36) — Wire serving stall detection over complete component inventories | [R14](factory-remaining-tasks/02-prompt-to-pr.md#r14), [R10](factory-remaining-tasks/01-boundaries.md#r10) | none | B |
| 8 | [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13) — Adopt one chosen semantic graph with exact scope and provenance | [R12](factory-remaining-tasks/02-prompt-to-pr.md#r12) | [R04](factory-remaining-tasks/01-boundaries.md#r04), [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 8 | [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16) — Connect real native task scheduling and dependency release | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09), [R15](factory-remaining-tasks/02-prompt-to-pr.md#r15) | none | A |
| 9 | [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17) — Produce trusted task receipts and early Git checkpoints | [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16) | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30) | A |
| 9 | [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32) — Connect Product Lead and Architect finish controls to real consumers | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31), [R12](factory-remaining-tasks/02-prompt-to-pr.md#r12), [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13), [R15](factory-remaining-tasks/02-prompt-to-pr.md#r15) | none | B |
| 10 | [R18](factory-remaining-tasks/02-prompt-to-pr.md#r18) — Gate final integration and publication on the complete checked plan | [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17) | none | A |
| 10 | [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20) — Wire fresh candidate/base tuples into every current consumer | [R01](factory-remaining-tasks/01-boundaries.md#r01) | [R05](factory-remaining-tasks/01-boundaries.md#r05), [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17) | B |
| 11 | [R19](factory-remaining-tasks/02-prompt-to-pr.md#r19) — Complete one attributable PR and publication recovery journey | [R18](factory-remaining-tasks/02-prompt-to-pr.md#r18), [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20) | [R04](factory-remaining-tasks/01-boundaries.md#r04) | B |
| 11 | [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33) — Connect Builder task output and implementation finish to checkpoints | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31), [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16), [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17), [R18](factory-remaining-tasks/02-prompt-to-pr.md#r18) | none | A |
| 12 | [R21](factory-remaining-tasks/03-review-merge-reassessment.md#r21) — Sponsor bounded merge-in correction when the base advances | [R19](factory-remaining-tasks/02-prompt-to-pr.md#r19), [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20), [R15](factory-remaining-tasks/02-prompt-to-pr.md#r15) | [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16), [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17) | B |
| 12 | [R22](factory-remaining-tasks/03-review-merge-reassessment.md#r22) — Complete independent native reviews and full conversation evidence | [R19](factory-remaining-tasks/02-prompt-to-pr.md#r19), [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20) | [R04](factory-remaining-tasks/01-boundaries.md#r04), [R05](factory-remaining-tasks/01-boundaries.md#r05), [R08](factory-remaining-tasks/01-boundaries.md#r08) | B |
| 12 | [R25](factory-remaining-tasks/03-review-merge-reassessment.md#r25) — Evaluate every configured native check on the exact current tuple | [R19](factory-remaining-tasks/02-prompt-to-pr.md#r19), [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20) | [R04](factory-remaining-tasks/01-boundaries.md#r04), [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 13 | [R23](factory-remaining-tasks/03-review-merge-reassessment.md#r23) — Account for every correction and native thread resolution | [R22](factory-remaining-tasks/03-review-merge-reassessment.md#r22) | [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 14 | [R24](factory-remaining-tasks/03-review-merge-reassessment.md#r24) — Connect clean-review and funded-Jev delivery acceptance producers | [R22](factory-remaining-tasks/03-review-merge-reassessment.md#r22), [R23](factory-remaining-tasks/03-review-merge-reassessment.md#r23), [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) | none | B |
| 15 | [R26](factory-remaining-tasks/03-review-merge-reassessment.md#r26) — Gate every merge path with current acceptance and authorization | [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20), [R21](factory-remaining-tasks/03-review-merge-reassessment.md#r21), [R24](factory-remaining-tasks/03-review-merge-reassessment.md#r24), [R25](factory-remaining-tasks/03-review-merge-reassessment.md#r25) | [R04](factory-remaining-tasks/01-boundaries.md#r04) | B |
| 15 | [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34) — Connect Inspector and Fixer finish controls to native review/correction | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31), [R22](factory-remaining-tasks/03-review-merge-reassessment.md#r22), [R23](factory-remaining-tasks/03-review-merge-reassessment.md#r23), [R24](factory-remaining-tasks/03-review-merge-reassessment.md#r24), [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33) | none | B |
| 16 | [R27](factory-remaining-tasks/03-review-merge-reassessment.md#r27) — Confirm original issue completion before releasing dependants | [R26](factory-remaining-tasks/03-review-merge-reassessment.md#r26), [R14](factory-remaining-tasks/02-prompt-to-pr.md#r14) | [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 16 | [R41](factory-remaining-tasks/04-controls-repair-verification.md#r41) — Qualify intervention, withdrawal and interrupted-work recovery | [R06](factory-remaining-tasks/01-boundaries.md#r06), [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11), [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16), [R26](factory-remaining-tasks/03-review-merge-reassessment.md#r26), [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35), [R44](factory-remaining-tasks/05-experience-and-development.md#r44) | none | A |
| 17 | [R28](factory-remaining-tasks/03-review-merge-reassessment.md#r28) — Complete the post-completion Jev reassessment fence | [R27](factory-remaining-tasks/03-review-merge-reassessment.md#r27), [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) | [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 17 | [R38](factory-remaining-tasks/04-controls-repair-verification.md#r38) — Implement trusted Test Pilot manifests and artifact verification | [R10](factory-remaining-tasks/01-boundaries.md#r10), [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13), [R18](factory-remaining-tasks/02-prompt-to-pr.md#r18), [R27](factory-remaining-tasks/03-review-merge-reassessment.md#r27) | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09), [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R44](factory-remaining-tasks/05-experience-and-development.md#r44), [R45](factory-remaining-tasks/05-experience-and-development.md#r45) | A |
| 18 | [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29) — Complete protected reassessment adoption and prefix recovery | [R28](factory-remaining-tasks/03-review-merge-reassessment.md#r28), [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13) | [R05](factory-remaining-tasks/01-boundaries.md#r05) | B |
| 19 | [R37](factory-remaining-tasks/04-controls-repair-verification.md#r37) — Connect bounded Strategist repair, sponsorship and adoption | [R36](factory-remaining-tasks/04-controls-repair-verification.md#r36), [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29), [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11) | [R07](factory-remaining-tasks/01-boundaries.md#r07), [R08](factory-remaining-tasks/01-boundaries.md#r08), [R09](factory-remaining-tasks/01-boundaries.md#r09) | B |
| 20 | [R39](factory-remaining-tasks/04-controls-repair-verification.md#r39) — Make app completion and delivery repair truthful and bounded | [R38](factory-remaining-tasks/04-controls-repair-verification.md#r38), [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13), [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29), [R37](factory-remaining-tasks/04-controls-repair-verification.md#r37) | none | B |
| 21 | [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40) — Connect Strategist and Test Pilot finish controls | [R30](factory-remaining-tasks/04-controls-repair-verification.md#r30), [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31), [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29), [R37](factory-remaining-tasks/04-controls-repair-verification.md#r37), [R38](factory-remaining-tasks/04-controls-repair-verification.md#r38), [R39](factory-remaining-tasks/04-controls-repair-verification.md#r39) | none | B |
| 22 | [R42](factory-remaining-tasks/05-experience-and-development.md#r42) — Finish the persisted factory read model and API | [R10](factory-remaining-tasks/01-boundaries.md#r10) | [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13), [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17), [R24](factory-remaining-tasks/03-review-merge-reassessment.md#r24), [R28](factory-remaining-tasks/03-review-merge-reassessment.md#r28), [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29), [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31), [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32), [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33), [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34), [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35), [R36](factory-remaining-tasks/04-controls-repair-verification.md#r36), [R37](factory-remaining-tasks/04-controls-repair-verification.md#r37), [R39](factory-remaining-tasks/04-controls-repair-verification.md#r39), [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40) | C |
| 22 | [R48](factory-remaining-tasks/05-experience-and-development.md#r48) — Qualify actual harness transport and first-response semantic behavior | [R12](factory-remaining-tasks/02-prompt-to-pr.md#r12), [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13), [R16](factory-remaining-tasks/02-prompt-to-pr.md#r16), [R22](factory-remaining-tasks/03-review-merge-reassessment.md#r22), [R24](factory-remaining-tasks/03-review-merge-reassessment.md#r24), [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29), [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32), [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33), [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34), [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35), [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40), [R41](factory-remaining-tasks/04-controls-repair-verification.md#r41) | none | A |
| 23 | [R43](factory-remaining-tasks/05-experience-and-development.md#r43) — Complete Work/Spaces and current native frontend composition | [R42](factory-remaining-tasks/05-experience-and-development.md#r42) | [R03](factory-remaining-tasks/01-boundaries.md#r03) | C |
| 23 | [R47](factory-remaining-tasks/05-experience-and-development.md#r47) — Run current composed PostgreSQL and serving-path race acceptance | [R11](factory-remaining-tasks/02-prompt-to-pr.md#r11), [R13](factory-remaining-tasks/02-prompt-to-pr.md#r13), [R17](factory-remaining-tasks/02-prompt-to-pr.md#r17), [R20](factory-remaining-tasks/03-review-merge-reassessment.md#r20), [R23](factory-remaining-tasks/03-review-merge-reassessment.md#r23), [R26](factory-remaining-tasks/03-review-merge-reassessment.md#r26), [R29](factory-remaining-tasks/03-review-merge-reassessment.md#r29), [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32), [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33), [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34), [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35), [R37](factory-remaining-tasks/04-controls-repair-verification.md#r37), [R39](factory-remaining-tasks/04-controls-repair-verification.md#r39), [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40), [R41](factory-remaining-tasks/04-controls-repair-verification.md#r41), [R42](factory-remaining-tasks/05-experience-and-development.md#r42) | none | Coordinator |
| 24 | [R49](factory-remaining-tasks/05-experience-and-development.md#r49) — Demonstrate the complete development factory with real Work/Spaces | [R39](factory-remaining-tasks/04-controls-repair-verification.md#r39), [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40), [R43](factory-remaining-tasks/05-experience-and-development.md#r43), [R44](factory-remaining-tasks/05-experience-and-development.md#r44), [R45](factory-remaining-tasks/05-experience-and-development.md#r45), [R47](factory-remaining-tasks/05-experience-and-development.md#r47), [R48](factory-remaining-tasks/05-experience-and-development.md#r48) | none | Coordinator |
| 25 | [R54](factory-remaining-tasks/06-candidate-release-closure.md#r54) — Finish one-current-path cutover, docs and safe resource accounting | [R41](factory-remaining-tasks/04-controls-repair-verification.md#r41), [R43](factory-remaining-tasks/05-experience-and-development.md#r43), [R45](factory-remaining-tasks/05-experience-and-development.md#r45), [R49](factory-remaining-tasks/05-experience-and-development.md#r49), [R50](factory-remaining-tasks/06-candidate-release-closure.md#r50) | none | Coordinator |
| 26 | [R51](factory-remaining-tasks/06-candidate-release-closure.md#r51) — Produce one coherent current x86_64 candidate | [R02](factory-remaining-tasks/01-boundaries.md#r02), [R43](factory-remaining-tasks/05-experience-and-development.md#r43), [R45](factory-remaining-tasks/05-experience-and-development.md#r45), [R49](factory-remaining-tasks/05-experience-and-development.md#r49), [R50](factory-remaining-tasks/06-candidate-release-closure.md#r50), [R54](factory-remaining-tasks/06-candidate-release-closure.md#r54) | none | C |
| 27 | [R52](factory-remaining-tasks/06-candidate-release-closure.md#r52) — Qualify exact candidate install, upgrade, recovery and support | [R51](factory-remaining-tasks/06-candidate-release-closure.md#r51), [R49](factory-remaining-tasks/05-experience-and-development.md#r49) | none | Coordinator |
| 28 | [R53](factory-remaining-tasks/06-candidate-release-closure.md#r53) — Complete immutable public delivery and downloaded-byte verification | [R52](factory-remaining-tasks/06-candidate-release-closure.md#r52) | none | Coordinator |
| 29 | [R55](factory-remaining-tasks/06-candidate-release-closure.md#r55) — Independently accept the complete remaining scope and close execution | [R46](factory-remaining-tasks/05-experience-and-development.md#r46), [R47](factory-remaining-tasks/05-experience-and-development.md#r47), [R48](factory-remaining-tasks/05-experience-and-development.md#r48), [R49](factory-remaining-tasks/05-experience-and-development.md#r49), [R50](factory-remaining-tasks/06-candidate-release-closure.md#r50), [R51](factory-remaining-tasks/06-candidate-release-closure.md#r51), [R52](factory-remaining-tasks/06-candidate-release-closure.md#r52), [R53](factory-remaining-tasks/06-candidate-release-closure.md#r53), [R54](factory-remaining-tasks/06-candidate-release-closure.md#r54) | none | Codex |

### Why this order

The historical [dependency validation](../../.artifacts/muse-factory-workflow/reviews/dependency-order-20261018-01a1470a/validation.json) and [derived graph](../../.artifacts/muse-factory-workflow/reviews/dependency-order-20261018-01a1470a/graph.json) bind the retained Factory parent graph, not the new slice/cross-family refinements: 55 unique tasks, no cycles or unknown references, every prerequisite before its consumer, and all 54 other tasks feeding R55. All 365 requirement keys/texts and the existing task checkbox states are preserved. These are planning checks, not implementation acceptance.

- **Establish inputs first:** R01/R02 retained-input and focused CI-repair outputs are already accepted; R07/R08/R09 establish native control, exclusion and transport; R10 releases typed contracts. R44 qualifies Project/provider foundations before live dispatch. R50 proves the protected release mechanism independently with narrow development inputs, as original RT01 requires.
- **Build the common workflow before its consumers:** R11 precedes semantic planning and investigation. Native scheduling and trusted checkpoint production precede publication and current-tuple qualification. Product Lead/Architect, Builder, Inspector/Fixer and Strategist/Test Pilot controls compose the already established owners; R34 consumes R33 singleton-Fixer output rather than inventing another executor.
- **Join every route before merge:** R21 base-advance correction, R24 clear/Jev acceptance and R25 exact native CI all feed R26. Review and CI can progress independently. Completion then feeds the post-completion reassessment fence and protected adoption. R21 is required for final closure through this chain.
- **Compose before freezing:** R47/R48/R49 verify the serving/native/installed development joins. R54 finishes source/caller/selector/docs cutover before R51 creates candidate bytes. R50 supplies the tested admission mechanism; R52 collects exact shipping-byte evidence and R53 consumes it for final publication/download proof. R50 never waits for its future candidate.
- **Keep real inputs explicit:** Before a live trial, bind actual harness/model/account/authority, approved preparation, bounded funding and fixture custody. Before R51, resolve the pipeline's required immutable Fountain/component manifests and bytes, including any separately authorized input-component publication. Final product publication at R53 cannot supply an input needed earlier to build the candidate. Optional external Tailnet enrollment holds its named R46 cases; it does not block unrelated core work.
- **Close the whole graph:** R55 depends transitively on all other 54 tasks. Independent acceptance, exact evidence and final resource release remain necessary; a clean DAG or a reviewed task definition is not implementation acceptance.

## Taking up a slice

There are no active assignments or workers. Use the [next-step disposition](#how-to-execute-the-remaining-work) and the actual prerequisite output in each card. Check existing edits/writers before changing a shared owner; one coherent author owns a transaction/schema or native-lifetime change at a time. No new coordinator, worktree, worker count, file-lease framework or blanket independent review is required. Reuse valid evidence and release downstream inputs as they become available.

The useful first product route is prompt → real graph → native implementation/checkpoints → one PR, followed by review/merge/reassessment. The feed/comment slices can proceed against existing accepted contracts alongside that route. R49 supplies development evidence; R50 proves the release mechanism; R54 freezes source before the single R51/R52 candidate path.

## Fourteen controls and seven profiles

| Control | Completion owner |
| --- | --- |
| `factory.context` | [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31) |
| `factory.evidence` | [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31) |
| `factory.status` | [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31) |
| `factory.receipt` | [R31](factory-remaining-tasks/04-controls-repair-verification.md#r31) |
| `factory.finish_incomplete` | [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35) |
| `factory.finish_product_plan` | [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32) |
| `factory.finish_issue_plan` | [R32](factory-remaining-tasks/04-controls-repair-verification.md#r32) |
| `factory.finish_implementation` | [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33) |
| `factory.finish_review` | [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34) |
| `factory.finish_correction` | [R34](factory-remaining-tasks/04-controls-repair-verification.md#r34) |
| `factory.finish_reassessment` | [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40) |
| `factory.finish_app_verification` | [R40](factory-remaining-tasks/04-controls-repair-verification.md#r40) |
| `factory.submit_task_output` | [R33](factory-remaining-tasks/04-controls-repair-verification.md#r33) |
| `factory.request_handoff` | [R35](factory-remaining-tasks/04-controls-repair-verification.md#r35) |

Product Lead and Architect use R32; Builder R33; Inspector and Fixer R34 (with singleton checkpoint reuse in R33); Strategist and Test Pilot R40, backed by R29/R37 and R38/R39. All seven also require R08/R09/R30/R31/R35. Every mutation needs actual downstream effect and observed retirement; read controls need complete authorized readback, not invented mutation effects.

## Requirement preservation and evidence gates

| Requirement owner | Indexed rows | New closure owners |
| --- | --- | --- |
| Workflow plan | 57 | Original FW01–FW50 and P gates mapped individually in the crosswalk |
| Workflow contracts | 108 | D01–D17, P1–P9, T01–T64, H2/H3/H4, Rust parity and fourteen controls mapped individually |
| Original implementation/UX plan | 137 | D01–D26, FT/ST/RT, FX/NF and all UX groups retained |
| RCX correction packet | 8 | R45/R46/R50–R52; retain later accepted source/native scope |
| RCX-2 design packet | 21 | Component/native/public gates map to R03–R05/R45/R46/R50–R55 as applicable |
| Native frontend continuation | 34 | Retain the 57-task owner's historical accepted scope; close changed joins/current qualification under R03/R43/R51/R52/R54/R55 |

**Both D13 requirements remain:** original implementation-plan D13 is complete exact-candidate/base native CI (**R25**); workflow-contracts D13 is the Jev gate after each verified completion (**R28/R29**). Neither supersedes the other.

P1 is R07/R16/R17; P2 R04/R05/R22; P3 R14; P4 R12/R48; P5 R05/R28/R29; P6 R09/R30; P7 R08; P8 R16/R17/R33; P9 R38/R39. H2 acceptance production is R24/R26, H3 history R23/R24/R26/R27, and H4 current tuples R20/R21/R25/R26.

The [original machine-readable scope snapshot](../../.artifacts/muse-factory-workflow/reviews/independent-20261017-01a1470a/reorganization199/remaining-task-catalog.json) preserves exact namespaced legacy keys, requirement text, prior observation, primary task and supporting tasks. Its historical dependencies do not override this revised order and the current task cards. It is not a second live status ledger. Unknown individual acceptance remains open until its mapped task verifies or repairs it.

## Detailed task packets

This overview is the dependency reference for the consolidated outcomes above. The six packet files hold the current state, deliverable, exact acceptance boundary, principal source owners and additional native gates for every task.

| Tasks | Detailed acceptance |
| --- | --- |
| R01–R10 | [Trustworthy boundaries](factory-remaining-tasks/01-boundaries.md) |
| R11–R19 | [Prompt to checked PR](factory-remaining-tasks/02-prompt-to-pr.md) |
| R20–R29 | [Current review, merge and reassessment](factory-remaining-tasks/03-review-merge-reassessment.md) |
| R30–R41 | [Role controls, repair and app verification](factory-remaining-tasks/04-controls-repair-verification.md) |
| R42–R49 | [Visible experience and development qualification](factory-remaining-tasks/05-experience-and-development.md) |
| R50–R55 | [Candidate, release and independent closure](factory-remaining-tasks/06-candidate-release-closure.md) |

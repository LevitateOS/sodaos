# Role controls, repair and app verification

[Remaining-work catalog](../factory-remaining-tasks.md#task-list) owns execution order. These packets preserve the original requirement authorities; the catalog's consolidated outcomes own current status; the closed checklist retains evidence and former leases. Implementation is stopped. The executable remainder refines each existing parent without adding counted outcomes. Source locators identify the current owner, not blanket write permission; use the catalog’s next-step disposition and actual prerequisite outputs.

<a id="r30"></a>
## R30 — Qualify one live Factory MCP authority and lifecycle

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R10](01-boundaries.md#r10).

**Current (wind-down):** Qualify live MCP authority/lifecycle, including role communication. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect one server and fixed relay to exact role/subject/repository/revision/run/incarnation authority, trusted startup, reconnect/stop and receipt lifecycle. Activate only after the native boundaries actually pass.

**Ralph absorption addition:** Extend this same MCP authority to R32's typed role
communication. Derive author, role, run, assignment and subject from admission;
apply lifetime, replay, revocation and cross-scope refusal to communication as
well as formal results. Keep the existing Factory server/relay and profiles.
Native child/lane presentation does not grant the parent's mutation capability.

**Accept only when:** Cross-scope/expired/revoked/forbidden calls refuse server-side. Replay returns the same receipt, conflict refuses, restart/rebind cannot renew authority. No secrets in prompts/argv/logs or broad operator/extension socket exposure.

**Executable remainder:** Consume R08's OS/peer boundary, R09's actual relay contract and R10's typed scope. `BuildMCPCapability` currently stages metadata; `FactoryMCPHandler` currently supplies neither `Observed` nor `Catalog`. First wire trusted host observation plus stop/revocation ordering into activation and request admission; prove the cached-stopped=false race remains refused. Then qualify reconnect/receipt recovery through the real CLI. Release live authority to R31 and formal finish consumers; R32 communication reuses it when its schema is ready. Human comments use authenticated user authority, never a borrowed agent capability. PG activation/withdrawal races and native peer/stop tests are required; staged metadata alone remains unavailable.

**Principal source owners:** [cmd/soda-dashboard/factory_mcp.go](../../../cmd/soda-dashboard/factory_mcp.go), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/mcp_admission.go](../../../internal/factory/control/mcp_admission.go), [internal/store/factory_mcp_capability.go](../../../internal/store/factory_mcp_capability.go), [lib/host/src/terminal/factory/mcp_relay.rs](../../../lib/host/src/terminal/factory/mcp_relay.rs).

<a id="r31"></a>
## R31 — Finish all four bounded MCP read controls

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30).

**Current (wind-down):** Complete four canonical bounded read controls and live readback. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Complete factory.context, factory.evidence, factory.status and factory.receipt using bounded current canonical evidence, complete pagination, honest incompleteness and exact receipt recovery.

**Accept only when:** Each admitted profile receives its allowed current context without spending, leaking or granting authority. Missing pages cannot claim completeness. Reads need no native mutation effect; they still require live authenticated transport and correct readback.

**Executable remainder:** Start at `factoryCatalogServer::{serveRead,serveEvidence,serveReceipt}`: receipt recovery exists, while evidence deliberately reports incomplete. First implement bounded current context/evidence projections over existing plan/run/review/inventory rows; add provenance-linked user comments through R32/R42 when available. Release cursor/completeness and allowed-context contracts to role consumers without waiting for live activation. Then use R30 for real read-only transport. Test missing/later pages, revisions, unauthorized repository/comment access and zero spending through the actual handlers. Context presents comments as attributed user input; it must not promote comment text into tool authority or expose unsubscribed private work.

**Read-contract status:** zero spending, unknown-capability refusal on all four reads, cross-assignment receipt non-disclosure and honest-incomplete evidence are proven through the actual handlers. Context now projects the recorded assignment behind the capability (prompt digest, exact source/preparation, outcome/reason once set; never prompt bytes, history or actor) and refuses without its assignment. Wider projections over plan/run/review/inventory rows plus pagination inputs stay open: ref/cursor are absent from the current schema by the FW40 boundary.

**Context-scope direction (A+B with Jev selection):** context serves current rows plus paged history. History scope is local lineage by default — own attempt, parents, prior attempts on the same task, carried findings — because full history is unbounded and risks exposing unsubscribed private work; anything wider goes through explicit authorized pages. At the context point, code builds fixed context bundles and Jev Choice-selects among them (same pattern as the existing priority judgments: server-only secret file, funded consultation first, low confidence or unavailable holds with attention, never an inferred bundle). Build slice: fixed bundles, one Choice question, funded once per capability with deterministic fallback to current-only.

**External design (retained, advice only):** full point-design at `.artifacts/codex-consult-context-design/response.md` (prompt retained beside it). Verified grounded: S0 field list, fingerprint/purpose registry, budget subledgers, rate limits and worker-rule references all check out against source. Key refinements over the recipe sketch: the saved choice selects a projection *template* (reads reproject live facts, no frozen copy); preparation is controller-owned before first read (never a read side effect); funding is exactly one attempt per capability with reservation owner; thresholds (0.80/0.80), collection caps and cursor shape are *proposed* needing evaluation, not settled. Not building until directed.

**Selection-plumbing slice (landed, host PG proof pending):** `context-bundle-v1` purpose registered (eleven purposes); `ApplyContextBundleAnswer` serves only a confident eligible selection, else holds snapshot_hold with attention; `Coordinator.SelectContextBundle` runs the funded Choice over code-enforced eligibility through the same receipt-cache path as existing judgments — repeat selections reuse the cached receipt without new spend, reserve refusal spends nothing, chooser failure or an unoffered selection holds snapshot_hold. Sandbox: `go vet` clean, archcheck PASS, factory pure tests PASS, control option/state tests PASS; six funded `SelectContextBundle` tests skip without SODA_PG_* and need the host fixture run.

**Bundle-content join slice (landed):** `BundleEligibility` + `AttemptContinuity` join fixed bundle content with zero new SQL — attempt serves own-attempt run IDs when the recorded assignment carries continuity, lineage serves the linked parent's outcome/reason via the existing `Assignment` read. `serveRead` projects the widest eligible bundle (`bundle` names current/attempt/lineage), keeps prompt bytes and actor identity out, and marks a linked-but-unloadable parent `lineage_incomplete` instead of claiming parent facts. Reads still spend nothing (`TestFactoryCatalogReadsSpendNothingAndDiscloseNothing` PASS).

**Receipt-foregrounding slice (landed):** `ContextBundleFingerprint` recomputes the exact selection fingerprint from shared builders, and `serveRead` looks up that receipt through a new `PriorityJudgments` dep (`*store.Store` already implements it) — a recorded preparation-time selection narrows the served sections to its bundle, hold serves current-only, and a missing/mismatched/unknown receipt falls back to the widest eligible bundle. The lookup never consults, so reads still spend nothing (contract PASS).

**Preparation-call slice (landed, host PG proof pending):** `PrepareContextBundleSelection` rebuilds the staged capability from recorded preparation, joins parent content for eligibility, and records one funded Choice through `ReservePriorityJudgment` (once per fingerprint by construction; recorded selections reuse without new spend). `launchTail` invokes it best-effort beside capability staging — refusal or an unavailable consultation never fails the launch. Sandbox: vet clean, nil-guard test PASS, handler contract tests PASS, archcheck PASS; two PG-lane tests fail-fast without the fixture by lane convention and need the host run.

**Evidence run-projection slice (landed):** `serveEvidence` projects the admitted assignment's own attempt as bounded run rows (id/role/outcome only, capped at `MCPMaxPageRefs` with `runs_truncated` marking the cap) via a new `Runs` dep (`*store.Store` already implements it). Missing assignments and unresolvable recorded runs refuse; plan/review/inventory pages plus pagination stay pending, so evidence remains honestly incomplete and the spend-nothing contract still passes.

**FW40 pagination handoff (precise, no code):** plan/review/inventory readers are scoped by repository/issue/project/owner (`PriorityReceiptsForIssue` needs issue plus cursor, `ProjectFactoryRuns`/`ProjectUnsettledRuns` are project-wide, `IntentReview` is revision-keyed) — none is assignment-scoped, so R31 cannot serve them capability-bound without two schema-side pieces that FW40 owns: (1) cursor/ref inputs on the evidence envelope (the envelope currently refuses unknown top-level fields, so this is a schema change, not a handler tweak); (2) a per-row lineage filter restricting every page to the admitted attempt plus its loaded parent chain. R31 closes pagination when evidence serves plan/review/inventory first pages plus cursor follow-ups with `complete` true only on exhaustion — a missing page still refuses or declares incomplete, never empty success. Live readback stays held on R30, comments on R32/R42.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control](../../../internal/factory/control), [internal/store/factory_mcp_submission.go](../../../internal/store/factory_mcp_submission.go).

<a id="r32"></a>
## R32 — Connect Product Lead and Architect finish controls to real consumers

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R12](02-prompt-to-pr.md#r12), [R13](02-prompt-to-pr.md#r13), [R15](02-prompt-to-pr.md#r15).

**Current (wind-down):** Planning finish consumers and typed ask/flag/learned/suggest/partial/blocked/stuck/done records. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind product-plan and issue-plan prompts/results to current common dispatch, semantic graph/manifest proposal adoption and immutable executable plans, with explicit product attention and observed terminal settlement.

**Ralph absorption addition:** Own the common typed communication producer for
all seven admitted roles: **ask, flag, learned, suggest, partial, blocked, stuck
and done**, following the
[verb blueprint](../../design/factory-feed-and-ralph-integration.md#adopted-verb-vocabulary).
Resolve the exact tool representation/schemas within Soda's existing MCP system,
reuse Ralph's useful structured payloads, and persist attributable submissions
and revisions through the existing domain/Store owners. Link canonical findings,
plans and evidence instead of creating duplicate truth. Done uses each role's
existing finish control; R33/R34/R40 retain ownership of their role-specific
downstream effects. Partial progress is nonterminal unless the role explicitly
finishes incomplete through R35. Ask/reply and suggest follow existing product
review/adoption authority; blocked/stuck are reports for deterministic assessment,
not caller-authorized readiness transitions. R42 consumes these records.

**Additional acceptance:** Exercise each verb with actual persisted readback and
the admitted role/run binding. Forged authors, wrong scope, changed-content
replay, expired/revoked capabilities and terminal-afterlife mutation refuse.
Question/answer, partial progress, known blocker and diagnosis-needed reports
remain distinct; no communication grants approval, resets allowances or turns an
agent's done report into verified completion. Update affected prompt/tool
contracts and consumers together when the extension representation is selected.

**Accept only when:** Real handler→controller→Store→permitted native effect→readback cases pass for both controls and all refusals. No-repository drafts launch nothing, model output grants no approval, and accepted finish cannot advance while the process remains uncertain.

**Executable remainder:** Work in three coherent slices. First define the eight-verb envelope and persisted human post/comment/reply mutations in the existing MCP/domain/Store owners: server-derived actor/scope, stable post/comment IDs, revision/replay rules and canonical evidence references. Choose tool representation and bounds in the owning feed design; no replacement MCP server. Second join Product Lead/Architect finish to R12/R13/R15 consumers. Third expose authorized discussions through R31 and bind a selected comment/discussion revision to an explicit planning/issue request, returning its durable result or failure. Reuse existing product scope/approval rules; do not require new approval for every comment or auto-launch on every post. R42/R43 consume records. Real-PG handler tests cover all verbs/comments, replay/conflict/revocation and origin links; native finish acceptance still needs R30 and observed retirement.

**Discussion-to-effect recovery:** The [accepted discussion contract](../../design/factory-feed-and-ralph-integration.md#subscriptions-comments-and-executable-work) requires more than persisted comments and a success link. After the trigger/routing choice for this slice is resolved and R12/R13/R15 consumers exist, bind one immutable discussion revision and request ID to their existing dispatch/native operation journals in `intent_generation.go`, `factory_intent_generation.go` and `factory_planning_revisions.go`. Next exercise edit/deletion or access loss before admission, an unavailable consumer, and a lost native-effect reply after commit. Finite acceptance: stale/refused/pending results are visible; retry recovers the original plan/issue and charge, never a duplicate or a relabeled newer comment; confirmed results retain the original discussion link. R31/R42/R43 expose the same outcome with current access checks. Trigger/edit policy remains a scoped design choice, not permission to auto-dispatch or demand approval for every comment.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r33"></a>
## R33 — Connect Builder task output and implementation finish to checkpoints

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R16](02-prompt-to-pr.md#r16), [R17](02-prompt-to-pr.md#r17), [R18](02-prompt-to-pr.md#r18).

**Current (wind-down):** Builder output/finish to trusted checkpoint and singleton Fixer reuse. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Wire factory.submit_task_output and factory.finish_implementation through actual native observation, trusted checks/Git CAS/receipt and the common plan completion owner. Reuse the executor for singleton Fixer output.

**Accept only when:** Duplicate/racing proposals produce one checkpoint and no ownership rewrite. Child/handoff identity is forbidden for singleton Fixer; failed checks, mutation after release and unobserved retirement cannot advance. P8 is a real native composition.

**Executable remainder:** Consume R16 events, R17 trusted checkpoint production, R18 final-plan gate and R30/R31 authenticated submission/readback. Start at `factoryCatalogServer::serveMutation`, `task_events.rs` and `SavePlanTaskReceipt`. Wire submit_task_output to the existing observed/check/CAS path, then finish_implementation to final-plan completion. Reuse that executor for singleton Fixer without children/handoff. Release consumed receipt/effect references to R34/R42. Race duplicate proposals and failed checks in focused PG tests; P8 then proves a real native checkpoint and retirement. Accepted submission alone must not mark a task completed or release capacity.

**Principal source owners:** [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r34"></a>
## R34 — Connect Inspector and Fixer finish controls to native review/correction

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R22](03-review-merge-reassessment.md#r22), [R23](03-review-merge-reassessment.md#r23), [R24](03-review-merge-reassessment.md#r24), [R33](04-controls-repair-verification.md#r33).

**Current (wind-down):** Inspector/Fixer finish to actual review/correction effects. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Apply exact independent role prompts, canonical review coverage/dispositions and singleton Fixer output to factory.finish_review and factory.finish_correction. Use conditional native review/comment/resolution owners.

**Accept only when:** Clear is explicit and complete; nothing-found, incomplete and disputed reports keep their distinct semantics. No direct agent merge or authority grant. Native readback, task output and observed retirement gate progression.

**Executable remainder:** Consume R22 review, R23 dispositions, R24 acceptance and R33 singleton output with R30/R31 authority/readback. In the existing MCP mutation consumer, route finish_review to native review ownership and finish_correction to the correction journal/checkpoint owner. Release report, native resolution and effect receipts to R42. Test explicit clear versus incomplete/disputed reports, stale candidate and replay through real handlers/Store; native readback must agree after observed retirement. Preserve independent actor separation and no agent merge authority.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/correction.go](../../../internal/factory/control/correction.go).

<a id="r35"></a>
## R35 — Finish incomplete, handoff and terminal settlement semantics

**Work:** Implement and integrate. **Owner lane:** A. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R11](02-prompt-to-pr.md#r11).

**Current (wind-down):** Incomplete/handoff/terminal settlement composition. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Wire factory.finish_incomplete and factory.request_handoff to the exact blocked owner/approved DAG successor and common stop/settlement path. Preserve pre-start hold/FactoryStart, receipt recovery and terminal capacity.

**Accept only when:** Blocked/nothing-found/finish are distinct. Submission is not observed termination; stop/submission/reconnect/restart races cannot release unknown children, transfer ownership arbitrarily or charge twice.

**Executable remainder:** Consume R11 settlement and R30/R31 authority/receipts. Start at MCP finish_incomplete/request_handoff, `dispatch_result.go::AccountSettledRun` and host `finish.rs`/`stop.rs`. First persist the exact blocked owner or admitted DAG successor proposal; then join stop observation, children retirement and one settlement/credential return. Release terminal state/custody facts to R41/R42. Run PG submission-versus-stop/reconnect races and one native parent-with-child restart witness. Ordinary partial communication is not terminal; unknown retirement holds custody/capacity, and handoff cannot create a new allowance or arbitrary assignee.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs), [lib/host/src/factory/stop.rs](../../../lib/host/src/factory/stop.rs).

<a id="r36"></a>
## R36 — Wire serving stall detection over complete component inventories

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R14](02-prompt-to-pr.md#r14), [R10](01-boundaries.md#r10).

**Current (wind-down):** Generation advancement on verified delivery and full episode consumer composition. Serving detector/events/isolation/order accepted through 3af53d09. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Preserve the accepted serving detector, complete component inventory and bound episode/fence; finish progress-generation and downstream consumers. Separate active work, CI, quota, preparation, permission, pause and product attention from cycles/missing producers/unexplained no-ready work.

**Accept only when:** Ordinary waits make no investigator call; a wait in one component cannot hide repairable work in another. Partial inventory cannot prove deadlock. Ready work returns to selection and fully delivered outcomes enter verification.

**Executable remainder:** Detection, repository isolation and ordered events are accepted. Start at `stall_serve.go::ServeStallRound`, `factory_stall_events.go::StallRoundGeneration` and the verified-completion transaction. First advance progress generation from a committed verified required outcome exactly once (attributable merge/completion or accepted evidenced non-code prerequisite); duplicate observation/restart, comments, edits and graph reshuffles must not increment it. Then route classified ready/wait/attention/stall/verify outputs to their existing consumers, with verify handed to R38/R39 when available. Release bound current episodes to R37. Extend serving PG cases for generation and delivered-state handoff; retain mixed-component/isolation tests. Never add another detector/timer or infer deadlock from incomplete inventory.

**Serving proof (landed):** `TestServeStallRoundGenerationReuseAndIsolation` runs two rounds on real PG — generation stays reused at zero, a stalled repo keeps one bound episode and one charge while a ready sibling resumes selection with no episode, charge or standing observation. Open finding: the detector's verify branch has no production producer — nothing writes completed readiness, delivered outcomes flow through completion fences instead. Who advances generation on verified delivery (this card vs the R37 lifecycle owner) and what produces delivery state for the verify handoff is a design decision, not implemented here.

**Principal source owners:** [internal/factory/control/stall_detect.go](../../../internal/factory/control/stall_detect.go), [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go).

<a id="r37"></a>
## R37 — Connect bounded Strategist repair, sponsorship and adoption

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R36](04-controls-repair-verification.md#r36), [R29](03-review-merge-reassessment.md#r29), [R11](02-prompt-to-pr.md#r11).

**Current (wind-down):** Strategist proposal/Jev choice and later repair-cycle adoption; cycle-1 episode/charge joint accepted. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Admit one bound repair episode under explicit sponsor and stable intent/repository/global accounting. Consume a checked Strategist proposal and finite Jev choice, then reuse protected serial graph adoption and fresh readback.

**Accept only when:** Restart, Retry, replacement issues and graph reshuffles cannot refill allowances; only material new evidence permits the allowed next cycle. No-change cannot clear a real stall; obsolete ready work retires the episode and committed prefixes remain fenced/reconciled.

**Executable remainder:** Consume R36's current episode, R11's stable charge and R29's protected serial adoption. `repair_admit.go::AdmitStallEpisodeRun` already owns cycle-1 charging. First connect a checked Strategist proposal and bounded Jev choice to that episode; then admit only the allowed later cycle with material new evidence and unchanged cumulative ledger. Release adopted repair or named attention to R39/R40. PG tests cover repeated detection, restart, Retry and reshuffled issue IDs without refill; a native/funded case consumes R07–R09. No-change cannot clear a real stall, and ready work retires obsolete diagnosis.

**Principal source owners:** [internal/store/factory_repair_grants.go](../../../internal/store/factory_repair_grants.go), [internal/store/factory_reassessment_allowances.go](../../../internal/store/factory_reassessment_allowances.go), [internal/factory/control](../../../internal/factory/control).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09). Source-only work cannot release these gates.

<a id="r38"></a>
## R38 — Implement trusted Test Pilot manifests and artifact verification

**Work:** Implement and prove. **Owner lane:** A. **Accepted inputs:** [R10](01-boundaries.md#r10), [R13](02-prompt-to-pr.md#r13), [R18](02-prompt-to-pr.md#r18), [R27](03-review-merge-reassessment.md#r27).

**Current (wind-down):** Trusted Test Pilot manifest adoption and exact integrated-artifact checks. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind approved manifest/outcome coverage to accepted intent version and integrated tree/artifact/fixture. Launch Test Pilot under the readonly profile and existing bounded host operations, retaining trusted check and cleanup receipts.

**Accept only when:** P9 uses the actual approved fixture and current integrated bytes. Missing/stale/failed checks cannot pass; forged proposed evidence is not trusted check output. Lost replies and CLI-start ordering preserve identity and no deployment occurs.

**Executable remainder:** Consume R10 delivery identity, R13 accepted outcome coverage, R18 exact integrated artifact and R27 verified delivery. Start at `internal/factory/mcp_manifest.go::ValidateManifestProposal`, `verification.go` and API `FactoryMCPDeps.Catalog`. First wire approved catalog lookup and durable adoption of the proposal; then launch its bounded readonly Test Pilot checks through existing host manifest/run owners. Bind trusted check output independently of submitted report digests. Release exact verification receipts to R39/R40. Test missing fixture, stale manifest/artifact, forged receipt and failed required check at the actual Store/handler join, then native P9 using R30/R44/R45. No deployment or agent-defined shell authority.

**Principal source owners:** [internal/factory](../../../internal/factory), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store), [lib/host/src/factory/manifest.rs](../../../lib/host/src/factory/manifest.rs), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R30](04-controls-repair-verification.md#r30), [R44](05-experience-and-development.md#r44), [R45](05-experience-and-development.md#r45). Source-only work cannot release these gates.

<a id="r39"></a>
## R39 — Make app completion and delivery repair truthful and bounded

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R38](04-controls-repair-verification.md#r38), [R13](02-prompt-to-pr.md#r13), [R29](03-review-merge-reassessment.md#r29), [R37](04-controls-repair-verification.md#r37).

**Current (wind-down):** Verified app completion and bounded delivery-repair chain. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Complete an intent only from all accepted delivered outcomes plus its exact verification receipts. Route real verification failure through the distinct bounded delivery-repair cause, stable intent ledger and protected graph owner.

**Accept only when:** Intent A can verify while unrelated B progresses. A missing fixture is a precise per-intent wait. Failure/Retry/revision cannot refill the two-run ceiling or manufacture an app intent for a manual issue; material scope questions return to the owner.

**Executable remainder:** Consume R38 trusted receipts, R13 accepted scope and R29/R37 protected repair/ledger contracts. Start at `verification.go::{DeliverySubject,DeliveryRepairSubject}` and existing intent/allowance Store owners. First make intent completion require all accepted outcomes and exact artifact verification; then route verified failure to the distinct delivery-repair cause with its two-run ceiling. Release completed/failed/waiting state to R40/R42. Real-PG cases cover unrelated intent progress, missing fixture, partial coverage, duplicate result, Retry and revision without allowance refill. A manual issue gains no synthetic app intent, and material scope change returns through existing product authority.

**Principal source owners:** [internal/factory](../../../internal/factory), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store).

<a id="r40"></a>
## R40 — Connect Strategist and Test Pilot finish controls

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R29](03-review-merge-reassessment.md#r29), [R37](04-controls-repair-verification.md#r37), [R38](04-controls-repair-verification.md#r38), [R39](04-controls-repair-verification.md#r39).

**Current (wind-down):** Strategist/Test Pilot finish effects and seven-profile instruction/context matrix. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect typed role prompts/results and observed retirement to the reassessment/repair and exact-artifact verification/app completion owners. Finish the seven-profile instruction/context matrix.

**Accept only when:** Both controls have authenticated producer→durable admission→permitted effect→native/host readback→visible state, including stale/blocked/refused/replay cases. Tool completion grants no merge or deployment authority.

**Executable remainder:** Consume R29/R37 proposal adoption, R38/R39 verification/completion and R30/R31 admission/readback. Start at MCP finish_reassessment/finish_app_verification payloads and `dispatch_prompt.go::BuildDispatchPrompt`. Wire each recorded submission to its existing consumer and retirement gate; then reconcile instructions/context for all seven profiles, including permitted communication and comment provenance. Release durable effect/readback references to R42/R48. Handler/Store tests distinguish recorded, refused, waiting and applied across replay; actual native receipts complete qualification. Tool success grants neither merge nor deployment.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store).

<a id="r41"></a>
## R41 — Qualify intervention, withdrawal and interrupted-work recovery

**Work:** Integrate and qualify. **Owner lane:** A. **Accepted inputs:** [R06](01-boundaries.md#r06), [R11](02-prompt-to-pr.md#r11), [R16](02-prompt-to-pr.md#r16), [R26](03-review-merge-reassessment.md#r26), [R30](04-controls-repair-verification.md#r30), [R35](04-controls-repair-verification.md#r35), [R44](05-experience-and-development.md#r44).

**Current (wind-down):** Composed intervention/withdrawal/crash recovery. Execution stopped; owner unassigned. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Compose pause/cancel/resume/Retry/takeover with exact native tasks, outstanding operations, MCP capability withdrawal and credential/resource return. Preserve finite consumption and human work.

**Accept only when:** Observe whole owned execution retirement before release or takeover; uncertain stop remains fenced. Restart adopts committed effects instead of duplicating them. Human terminals, checkouts, services and credentials remain distinct and usable.

**Executable remainder:** Consume R06 importer custody, R11 settlement, R16 child journal, R26 operation reconciliation, R30 capability withdrawal, R35 terminal semantics and R44 provider custody. Start at host `stop.rs`, `project/admission.rs` and the existing controller pause/cancel/Retry paths. First compose withdrawal with active parent/children and outstanding native operation; then restart/takeover after known retirement versus uncertain stop. Release observed custody-return facts to R49/R54. Focused PG races plus native interruption at those two boundaries are sufficient; preserve committed effects, charges and unrelated human checkout/terminal/service data.

**Principal source owners:** [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store), [lib/host/src/factory/stop.rs](../../../lib/host/src/factory/stop.rs), [lib/host/src/project/admission.rs](../../../lib/host/src/project/admission.rs).

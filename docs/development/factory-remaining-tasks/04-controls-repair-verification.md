# Role controls, repair and app verification

[Executable task map](../factory-remaining-tasks.md#task-list) owns the current
deliverables and output dependencies. This packet preserves the parent contracts,
technical source owners, accepted evidence and additional native/installed gates.
The Rxx.y links below identify the execution units; parent input lists describe
full qualification and do not require waiting for whole parents before using
applicable outputs. Recorded baseline and landed slices retain their actual
scope. Implementation notes support the execution map rather than define a
second schedule. This documentation restructuring starts no runtime work.

<a id="r30"></a>
## R30 — Qualify one live Factory MCP authority and lifecycle

**Execution tasks:** [R30.1](../factory-remaining-tasks.md#r30-1), [R30.2](../factory-remaining-tasks.md#r30-2), [R30.3](../factory-remaining-tasks.md#r30-3).

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R10](01-boundaries.md#r10).

**Recorded baseline:** Qualify live MCP authority/lifecycle, including role communication. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect one server and fixed relay to exact role/subject/repository/revision/run/incarnation authority, trusted startup, reconnect/stop and receipt lifecycle. Activate only after the native boundaries actually pass.

**Full design scope:** Implement discovery and admission for the
[complete tool/profile inventory](../factory-remaining-tasks.md#full-worker-design-integration),
including the new Library Admin profile. Consume R10's maintenance subject and
R11's bounded assignment, with explicit collection/operation grants and no Git
issue or policy authority. Update the current catalog, role instructions, native
configuration and direct-call enforcement together. Qualify every added tool's
role/scope/revision, byte/count limits, idempotency, revocation and terminal
capacity; hidden tools must also refuse direct calls. Neither Library Admin nor
native children acquire broader parent/operator credentials. Release the common
contract to role producers before live qualification; schema design and focused
handler work do not wait for an unrelated full-parent outcome.

**Generic System One tools:** Implement the required
[`systemone.noul`, `systemone.choice`, `systemone.score`](../../design/factory-worker-mcp.md#generic-systemone-tools)
as reusable provider adapters built into every role's MCP surface. These are
three additional generic tools alongside the 63 Factory tools. Accept supplied
state and bounded question maps with primitive-specific criteria; preserve typed
answers/distributions, Score scale, usage and invocation identity. The generic
contract requires no issue/plan/Factory purpose; hosting supplies caller authority,
server-held credentials and allowance. Reuse suitable existing TypeSafe transport
through a generic owner rather than coupling these schemas to Factory DTOs.

R11 supplies finite worker-call funding and settlement; discovery and all direct
calls enforce the same admitted provider access. Cover missing configuration,
rate limits, invalid/partial answers, exhausted allowance, timeout with unknown
execution/charge, lost reply, replay and revocation. A recorded invocation can
be recovered without reevaluation; a genuinely new attempt remains accounted.
Return valid uncertain judgments without inventing universal thresholds, extra
consultations or false results. Generic judgments cannot substitute for trusted
controller acceptance receipts, advance work or auto-publish knowledge/posts.
R48 qualifies actual calls; R42/R43 expose references when an existing Factory
action uses a consequential judgment. This does not change spend-free reads.

**Ralph absorption addition:** Extend this same MCP authority to R32's typed role
communication. Derive author, role, run, assignment and subject from admission;
apply lifetime, replay, revocation and cross-scope refusal to communication as
well as formal results. Keep the existing Factory server/relay and profiles.
Native child/lane presentation does not grant the parent's mutation capability.

**Accept only when:** Cross-scope/expired/revoked/forbidden calls refuse server-side. Replay returns the same receipt, conflict refuses, restart/rebind cannot renew authority. No secrets in prompts/argv/logs or broad operator/extension socket exposure.

**Implementation notes:** Consume R08's OS/peer boundary, R09's actual relay contract and R10's typed scope. `BuildMCPCapability` currently stages metadata; `FactoryMCPHandler` currently supplies neither `Observed` nor `Catalog`. First wire trusted host observation plus stop/revocation ordering into activation and request admission; prove the cached-stopped=false race remains refused. Then qualify reconnect/receipt recovery through the real CLI. Release live authority to R31 and formal finish consumers; R32 communication reuses it when its schema is ready. Human comments use authenticated user authority, never a borrowed agent capability. PG activation/withdrawal races and native peer/stop tests are required; staged metadata alone remains unavailable.

**Principal source owners:** [cmd/soda-dashboard/factory_mcp.go](../../../cmd/soda-dashboard/factory_mcp.go), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/mcp_admission.go](../../../internal/factory/control/mcp_admission.go), [internal/store/factory_mcp_capability.go](../../../internal/store/factory_mcp_capability.go), [lib/host/src/terminal/factory/mcp_relay.rs](../../../lib/host/src/terminal/factory/mcp_relay.rs).

<a id="r31"></a>
## R31 — Complete role context, interaction and library/decision reads

**Execution tasks:** [R31.1](../factory-remaining-tasks.md#r31-1), [R31.2](../factory-remaining-tasks.md#r31-2), [R31.3](../factory-remaining-tasks.md#r31-3), [R31.4](../factory-remaining-tasks.md#r31-4).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30).

**Recorded baseline:** Complete four canonical bounded read controls and live readback. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Complete the four baseline reads plus inbox/discussion and knowledge/decision search/read, with the required context for all seven production profiles and Library Admin. Use bounded canonical evidence, explicit grants, complete pagination, honest incompleteness and exact receipt recovery.

**Required expanded reads:** Implement `factory.inbox`, `factory.discussion`,
`factory.knowledge_search`, `factory.knowledge_read`, `factory.decision_search`
and `factory.decision_read` alongside context/evidence/status/receipt. Consume
R32's canonical objects and R42's prepared projections incrementally; role-detail
read schemas stay with R32/R33/R34/R40 and reuse this paging/access contract.
Required context follows each role's always-gets/may-retrieve contract in the
[MCP design](../../design/factory-worker-mcp.md). Library Admin receives its
objective, collection/operation grant, exact item revisions, evidence/authority,
affected references, queue completeness and allowance. No broader library grant
is inferred from local attempt lineage, repository membership or subscription.

**Expanded acceptance:** Retrieve an answer and a changed decision through a
cursor without marking either incorporated; retrieve an applicable learning on
a later assignment with its actual support/validity and exact revision. Preserve
removed/superseded historical references, source revocation and independent review
boundaries. Complete empty, incomplete and unavailable results differ. Search,
discussion, inbox and role reads perform no writes, execution or model spending.
Jev may foreground a prepared optional bundle once per capability; required facts
and authority are unchanged. Missing/uncertain selection uses S0-plus-hold, and
new library/decision pages obey the same finite bounds and access checks.

**Accept only when:** Each admitted profile receives its allowed current context without spending, leaking or granting authority. Missing pages cannot claim completeness. Reads need no native mutation effect; they still require live authenticated transport and correct readback.

**Implementation notes:** Follow the [D14 worker-context design](../factory-workflow-contracts.md#d14-worker-context) from `factoryCatalogServer::{serveRead,serveEvidence,serveReceipt}` and the existing context selection/preparation owners. First close the partial-selection gaps below: enforce S0-plus-hold fallback, reapply validated effective decisions on receipt readback, and bind the one-call reservation to capability identity rather than only a fingerprint. Then join required role facts and authorized local-lineage projections, supply the complete selection state, and add bounded context cursor pages. General evidence record pages remain FW40; provenance-linked comments remain R32/R42. Release cursor/completeness and allowed-context contracts to role consumers without waiting for live activation, then use R30 for real read-only transport. Test missing/later pages, revisions, unauthorized repository/comment access and zero spending through actual handlers. Comments remain attributed user input, never tool authority or a route to unsubscribed private work.

**Read-contract status:** zero spending, unknown-capability refusal on all four reads, cross-assignment receipt non-disclosure and honest-incomplete evidence are proven through the actual handlers. The basic context projects the recorded assignment (prompt digest, exact source/preparation, outcome/reason once set) and refuses without it. Partial optional joins now expose own-attempt run IDs and a linked parent's outcome/reason; prompt bytes, private conversations and actor identities remain excluded. Full role projections, field-level lineage grants and pagination inputs remain open; current schemas have no context cursor or general evidence ref/cursor.

**Context-scope direction (A+B with Jev selection):** [D14](../factory-workflow-contracts.md#d14-worker-context) owns the distinction between what each role **needs**, what current information it **gets**, and what optional authorized history it **could use**. Required role material survives every choice. Code builds fixed current/attempt/lineage bundles and Jev selects which to foreground; Jev cannot choose permissions, invent context or omit required findings. History defaults to local lineage: own attempt, parents, authorized prior attempts on the same task and carried findings. Wider history needs explicit authorized pages. Selection is prepared proactively by the controller, once per capability, through the existing funded TypeSafe path; reads refresh current facts without new spending. Low confidence or unavailable selection serves the basic S0 snapshot with an enforced hold and attention, never the widest or another inferred bundle.

**Integrated design and remaining coverage:** the [D14 context section](../factory-workflow-contracts.md#d14-worker-context) now contains the seven-role needs/optional-history matrix, exact Builder/Inspector/Fixer projections, bundle definitions, Choice/state, reservation/fallback rules and bounded-page design. Product Lead, Architect, Strategist and Test Pilot still need their exact role adapters; a shared snapshot does not satisfy their required inputs. The 0.80/0.80 cutoffs in source still need context-specific evaluation; additional collection caps and cursor/wire shapes remain recommended design parameters. `.artifacts/codex-consult-context-design/response.md` is retained consultation history, not a second contract owner. This integration changes documentation, not implementation or qualification status.

**Selection-plumbing slice (landed, host PG proof pending):** `context-bundle-v1` purpose registered (eleven purposes); `ApplyContextBundleAnswer` serves only a confident eligible selection, else holds snapshot_hold with attention; `Coordinator.SelectContextBundle` runs the funded Choice over code-enforced eligibility through the same receipt-cache path as existing judgments — repeat selections reuse the cached receipt without new spend, reserve refusal spends nothing, chooser failure or an unoffered selection holds snapshot_hold. Sandbox: `go vet` clean, archcheck PASS, factory pure tests PASS, control option/state tests PASS; six funded `SelectContextBundle` tests skip without SODA_PG_* and need the host fixture run.

**Bundle-content join slice (landed):** `BundleEligibility` + `AttemptContinuity` join fixed bundle content with zero new SQL — attempt serves own-attempt run IDs when the recorded assignment carries continuity, lineage serves the linked parent's outcome/reason via the existing `Assignment` read. `serveRead` projects the widest eligible bundle (`bundle` names current/attempt/lineage), keeps prompt bytes and actor identity out, and marks a linked-but-unloadable parent `lineage_incomplete` instead of claiming parent facts. Reads still spend nothing (`TestFactoryCatalogReadsSpendNothingAndDiscloseNothing` PASS).

**Receipt-foregrounding slice (landed):** `ContextBundleFingerprint` recomputes the exact selection fingerprint from shared builders, and `serveRead` looks up that receipt through a new `PriorityJudgments` dep (`*store.Store` already implements it) — a recorded preparation-time selection narrows the served sections to its bundle, hold serves current-only, and a missing/mismatched/unknown receipt falls back to the widest eligible bundle. The lookup never consults, so reads still spend nothing (contract PASS).

**Preparation-call slice (landed, host PG proof pending):** `PrepareContextBundleSelection` rebuilds the staged capability from recorded preparation, joins parent content for eligibility, and records one funded Choice through `ReservePriorityJudgment` (once per fingerprint by construction; recorded selections reuse without new spend). `launchTail` invokes it best-effort beside capability staging — refusal or an unavailable consultation never fails the launch. Sandbox: vet clean, nil-guard test PASS, handler contract tests PASS, archcheck PASS; two PG-lane tests fail-fast without the fixture by lane convention and need the host run.

**Required reconciliation of these partial slices:** current source is not yet the D14 target. Missing/mismatched receipts fall back to the widest bundle; `serveRead` uses raw `receipt.Selected` without applying the saved answer's confidence/effective hold; preparation failures do not gate launch; and a fingerprint-keyed reservation does not cap changed inputs/options at one call per capability. The selection state currently contains only version/scope, although its rubric references current facts, history previews and candidate bundles. A loaded parent also does not establish a field-level visibility grant. Resolve these in `internal/factory/control`, `internal/store` and `internal/web/api` before claiming selection acceptance: use the unique capability point, complete bounded state, validated effective decision/explicit hold and per-row grants, while preserving denial order, no read-side writes/spend, recovery and terminal precedence. Existing source/test evidence above remains evidence of those narrower slices; it does not prove these missing guarantees.

**Evidence run-projection slice (landed):** `serveEvidence` projects the admitted assignment's own attempt as bounded run rows (id/role/outcome only, capped at `MCPMaxPageRefs` with `runs_truncated` marking the cap) via a new `Runs` dep (`*store.Store` already implements it). Missing assignments and unresolvable recorded runs refuse; plan/review/inventory pages plus pagination stay pending, so evidence remains honestly incomplete and the spend-nothing contract still passes.

**Pagination handoff:** bounded context-projection pages are specified by [D14](../factory-workflow-contracts.md#d14-worker-context) with a context-only cursor; they do not implement FW40's general evidence-record pages. Plan/review/inventory readers are scoped by repository/issue/project/owner (`PriorityReceiptsForIssue` needs issue plus cursor, `ProjectFactoryRuns`/`ProjectUnsettledRuns` are project-wide, `IntentReview` is revision-keyed), so neither read path may expose them directly. Both need schema inputs and per-row capability/visibility grants over the allowed lineage; a loaded parent chain alone is insufficient. FW40 retains evidence `{ref,cursor?}` and general plan/review/inventory records. Collection exhaustion can report complete only for its declared bounded snapshot, never for absent evidence or missing pages. Live readback stays held on R30, comments on R32/R42.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control](../../../internal/factory/control), [internal/store/factory_mcp_submission.go](../../../internal/store/factory_mcp_submission.go).

<a id="r32"></a>
## R32 — Implement shared records, planning tools and Library Admin

**Execution tasks:** [R32.1](../factory-remaining-tasks.md#r32-1), [R32.2](../factory-remaining-tasks.md#r32-2), [R32.3](../factory-remaining-tasks.md#r32-3), [R32.4](../factory-remaining-tasks.md#r32-4), [R32.5](../factory-remaining-tasks.md#r32-5), [R32.6](../factory-remaining-tasks.md#r32-6), [R32.7](../factory-remaining-tasks.md#r32-7), [R32.8](../factory-remaining-tasks.md#r32-8), [R32.9](../factory-remaining-tasks.md#r32-9), [R32.14](../factory-remaining-tasks.md#r32-14), [R32.10](../factory-remaining-tasks.md#r32-10), [R32.11](../factory-remaining-tasks.md#r32-11), [R32.12](../factory-remaining-tasks.md#r32-12), [R32.13](../factory-remaining-tasks.md#r32-13).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R12](02-prompt-to-pr.md#r12), [R13](02-prompt-to-pr.md#r13), [R15](02-prompt-to-pr.md#r15).

**Recorded baseline:** Planning finish consumers and typed ask/flag/learned/suggest/partial/blocked/stuck/done records. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Implement shared communication, decision/knowledge/artifact records, Product Lead/Architect work tools and Library Admin maintenance. Bind them to real domain consumers, durable revisions/receipts and feed/index obligations, with formal planning and maintenance finishes using common settlement.

**Ralph absorption addition:** Own the common typed communication producer for
all seven admitted roles: **ask, flag, learned, suggest, partial, blocked, stuck
and done**, following the
[verb blueprint](../../design/factory-feed-and-ralph-integration.md#adopted-verb-vocabulary).
Implement the seven distinct shared MCP tools selected in the
[D14 outlet contract](../factory-workflow-contracts.md#d14-communication-outlets)
within Soda's existing server. Define their strict schemas using Ralph's useful
structured payloads, and persist attributable submissions
and revisions through the existing domain/Store owners. Link canonical findings,
plans and evidence instead of creating duplicate truth. Done uses each role's
existing finish control; R33/R34/R40 retain ownership of their role-specific
downstream effects. Partial progress is nonterminal unless the role explicitly
finishes incomplete through R35. Ask/reply and suggest follow existing product
review/adoption authority; blocked/stuck are reports for deterministic assessment,
not caller-authorized readiness transitions. R42 consumes these records.

**Communication status boundary:** actual MCP outlets are the required design.
The seven nonterminal tools are shared by all seven admitted parent roles; Done
uses each role's existing finish tool. The current source catalog remains at
fourteen tools until the seven additions land, yielding the 21-tool communication extension in
the [D14 interface breakdown](../factory-workflow-contracts.md#d14-worker-interface-layers).
Tool selection is settled; strict schemas, bounds, persistence and consumer
wiring remain R32 work. Context selection in R31 does not implement this layer.

**Full design implementation slices:** The
[tool-to-task inventory](../factory-remaining-tasks.md#full-worker-design-integration)
is the complete scope, with behavior owned by the [MCP design](../../design/factory-worker-mcp.md).
Resolve strict schemas and finite per-assignment limits within these slices:

The later [issue work-contract extension](../factory-remaining-tasks.md#issue-work-contract-extension)
adds design/implementation scope for typed native issue metadata, proposed
`factory.create_issue` / `factory.request_prerequisite`, and a recommended
Researcher interface (`research_scope`, `record_research`, `finish_research`).
R32 is the primary tool/record owner for those additions; R10/R11/R30 admit them,
R31 supplies context/read conventions, R34 supplies any required research review,
and the named readiness/completion owners validate non-code results. Resolve the
Researcher versus Architect-variant choice explicitly; do not force research to
emit a PR or bypass today's non-code acceptance with a new label.

- [ ] Implement Ask/Answer/Reply and human comment/reply records with routing,
  answer incorporation, revision history and exact discussion-to-work origin.
  Submitted human comments invoke funded JEV assessment; a new-issue result
  admits Product Lead LLM authoring and validated native creation through R13.
  Persist discussion-only, existing-work and clarification results too. Drafts,
  reads and generated agent replies do not trigger this intake. Show saved,
  assessing, drafting and actually created as distinct states with one lineage.
  Implement Flag/Suggest/Partial/Blocked/Stuck with actual finding/proposal/
  readiness/diagnosis consumers; unaffected work continues. Inspector Flag uses
  R34's canonical finding, never a second feed-only defect.
- [ ] Implement `factory.record_decision`: choice, rationale, alternatives,
  consequences, affected scope, evidence, actual decider/authority and application
  state. Reference existing human/acceptance/Jev decisions rather than creating
  competing authorities. Deliver retrieval/index inputs to R31/R42 and changed
  decision notifications to affected work; logging alone applies no graph.
- [ ] Implement Learned, knowledge amendment/feedback and artifact registration:
  explicit reusable entries, applicability, separate support/governance/validity,
  provenance, contradiction/duplicate relations, stale-source handling and usage
  feedback. Use safe staged/capture references; illustrative artifacts cannot
  establish trusted checks. Library grants/publication respect source visibility;
  do not make every feed update permanent knowledge or require human review for
  ordinary observations. Release prepared-index inputs to R42.
- [ ] Implement Product Lead's product map, revisioned plan drafts, deterministic
  comparisons and finish against R12/R13. Implement Architect's issue context,
  investigation records, executable-plan drafts and finish against R15. Seal
  exact draft revisions, preserve unresolved required inputs and prevent drafts
  from launching work. Shared `factory.validate_result` invokes existing role
  validators read-only; each role owner supplies its schema and finish recheck.
- [ ] Implement Library Admin inventory/preview and typed edit, remove, restore,
  consolidate, split, relate and supersede operations for knowledge and decisions.
  Check current grants/revisions, preserve provenance/authority/history and old
  references, and commit each bounded multi-item operation plus mappings/event
  obligation atomically. Preview reserves nothing. Removed items leave ordinary
  discovery while historical references and still-operative required decisions
  retain their proper authoritative path. Routine authorized edits need no
  separate approval queue; changing an effective choice needs its actual owner.
- [ ] Implement bounded maintenance admission triggers through R10/R11/R30,
  the Library Admin instruction/result schema and `finish_library_maintenance`
  consumer. An explicit request or configured bounded trigger can admit work;
  reads do not. Finish summarizes confirmed operations and remaining cases;
  R35/R41 settlement preserves changes already committed before interruption.
- [ ] For every mutation return stable receipt/object/revision, actual consumer
  state and permitted next route. Persist canonical change and projection
  obligation together. Recover a lost response from the original identity;
  repair feed/index projection without repeating the native effect or operation.

**Expanded acceptance:** Exercise question → answer → incorporation, decision →
read/application/supersession, Learned → later use → contradiction/amendment and
all maintenance operations through handlers and real Store readback. Cover stale
previews, conflicting decisions, revoked sources, partial inventories, atomic
consolidation/split, restore against a newer replacement, lost replies and
operation success with delayed projection. Assert correct authority and preserved
references in each case. Release each producer to R31/R42/R43 as it is usable;
R32 does not wait for the complete UI or an entire factory run to begin.

The [exception-routing design](../../design/factory-feed-components.md#failure-component-routing)
specifies the typed failure/recovery information needed by those producers:
affected object/revision, actual reason and evidence, recorded/refused/uncertain
effect, receipt identity and permitted next route. R42 projects that state into
components; no generic widget-ping tool or worker-selected success state replaces
the real domain action. The per-role matrices are required implementation cases; qualification needs
their actual producer and consumer paths.

**Additional acceptance:** Discover and call all seven shared communication tools
through every admitted parent role profile, with actual persisted readback and
the admitted role/run binding. Exercise Done through each role's existing finish
tool; a feed label or generic text submission alone does not qualify. Forged
authors, wrong scope, changed-content
replay, expired/revoked capabilities and terminal-afterlife mutation refuse.
Question/answer, partial progress, known blocker and diagnosis-needed reports
remain distinct; no communication grants approval, resets allowances or turns an
agent's done report into verified completion. Update the canonical source catalog,
discovery, prompt/tool contracts and consumers together for the selected outlets.

**Accept only when:** Every assigned shared/planning/maintenance tool reaches its real owner and durable readback, with the required authority, revision and recovery cases. Planning finishes also require their permitted native effects and observed settlement. No-repository drafts launch nothing; recorded decisions and maintenance edits cannot grant policy or issue authority, and accepted finish cannot advance while the process remains uncertain.

**Implementation notes:** Use the full implementation checklist above. First establish shared record/revision/grant/receipt schemas and one Ask → answer → incorporation path. Next connect decision recording/retrieval and Learned → later use → amendment with R31/R42, then Library Admin operations through the same item owners. In parallel, join durable Product Lead/Architect drafts and finishes to R12/R13/R15 and connect exact discussion revisions to those requests. Release producer records and focused checks incrementally to R42/R43; finish the maintenance subject/consumer with R10/R11/R30/R35. Reuse existing scope/approval rules, the current MCP server and reserved terminal capacity. Real-PG cases cover actual consumers, history, authority, conflict and replay; native planning/maintenance qualification still needs admitted transport and observed settlement. Completing only the original seven signals or two planning finishes leaves this parent open.

**Discussion-to-effect recovery:** The [accepted discussion contract](../../design/factory-feed-and-ralph-integration.md#subscriptions-comments-and-executable-work) now settles the trigger: submitted human comment → JEV assessment → Product Lead LLM authoring if new work is selected → native issue creation. Bind one immutable comment revision and request ID through funded judgment, authoring and the existing dispatch/native operation journals in `intent_generation.go`, `factory_intent_generation.go` and `factory_planning_revisions.go`. Validate title/body, acceptance criteria and Soda work metadata before R13 applies the issue. Exercise discussion-only, existing-work and clarification outcomes, edits/deletion or access loss before application, missing allowance, malformed LLM output, an unavailable consumer and a lost native-effect reply after commit. Finite acceptance: stale/refused/pending/unknown results remain visible; retry reuses recorded judgments/drafts and recovers the original native operation, never a duplicate or a relabeled newer comment. Provider timeouts do not establish zero spending or permission for blind retries. Confirmed results retain the source link; explicit edited submissions bind a new revision to earlier work, and deletion does not erase an already created issue. R31/R42/R43 expose the same outcome with current access checks. Existing readiness/pause/grant rules govern subsequent execution; ordinary authorized issue creation needs no new approval per comment.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/intent_generation.go](../../../internal/factory/control/intent_generation.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r33"></a>
## R33 — Connect Builder task output and implementation finish to checkpoints

**Execution tasks:** [R33.1](../factory-remaining-tasks.md#r33-1).

**Work:** Implement after proof. **Owner lane:** A. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R16](02-prompt-to-pr.md#r16), [R17](02-prompt-to-pr.md#r17), [R18](02-prompt-to-pr.md#r18).

**Recorded baseline:** Builder output/finish to trusted checkpoint and singleton Fixer reuse. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Wire factory.submit_task_output and factory.finish_implementation through actual native observation, trusted checks/Git CAS/receipt and the common plan completion owner. Reuse the executor for singleton Fixer output.

**Full role interface:** Implement `factory.tasks` and `factory.task_inputs`
alongside output/finish, using the same immutable plan, readiness, ownership,
input/check manifest and native checkpoint records. Fixer sees only its admitted
singleton and cannot supply a child/handoff identity. Release these records to
R42 task lanes, previews, check failure/recovery and candidate components. A
Partial report cannot change observed task counts. Qualify later-page inputs,
failed/unknown output, stale candidate, concurrent independent progress and lost
checkpoint replies; expose the exact affected task and permitted next action.

**Accept only when:** Duplicate/racing proposals produce one checkpoint and no ownership rewrite. Child/handoff identity is forbidden for singleton Fixer; failed checks, mutation after release and unobserved retirement cannot advance. P8 is a real native composition.

**Implementation notes:** Consume R16 events, R17 trusted checkpoint production, R18 final-plan gate and R30/R31 authenticated submission/readback. Start at `factoryCatalogServer::serveMutation`, `task_events.rs` and `SavePlanTaskReceipt`. Wire submit_task_output to the existing observed/check/CAS path, then finish_implementation to final-plan completion. Reuse that executor for singleton Fixer without children/handoff. Release consumed receipt/effect references to R34/R42. Race duplicate proposals and failed checks in focused PG tests; P8 then proves a real native checkpoint and retirement. Accepted submission alone must not mark a task completed or release capacity.

**Principal source owners:** [lib/host/src/factory/task_events.rs](../../../lib/host/src/factory/task_events.rs), [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/store/factory_planning_revisions.go](../../../internal/store/factory_planning_revisions.go).

<a id="r34"></a>
## R34 — Implement Inspector and Fixer work, coverage and finish controls

**Execution tasks:** [R34.3](../factory-remaining-tasks.md#r34-3), [R34.1](../factory-remaining-tasks.md#r34-1), [R34.2](../factory-remaining-tasks.md#r34-2).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R22](03-review-merge-reassessment.md#r22), [R23](03-review-merge-reassessment.md#r23), [R24](03-review-merge-reassessment.md#r24), [R33](04-controls-repair-verification.md#r33).

**Recorded baseline:** Inspector/Fixer finish to actual review/correction effects. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Apply exact independent role prompts, canonical review coverage/dispositions and singleton Fixer output to factory.finish_review and factory.finish_correction. Use conditional native review/comment/resolution owners.

**Full role interfaces:** Implement Inspector `review_scope`,
`record_review_coverage` and `assess_finding`, plus Fixer `correction_scope` and
`record_disposition`, with their existing typed finishes. Inspector's shared Flag
creates/revises the same canonical draft finding that R22 publishes; early feed
visibility remains reported/draft. Persist complete coverage/disposition sets
before finish with exact candidate and evidence bindings. R33 supplies Fixer's
singleton task views/output. Qualify missing carried findings, stale candidates,
unreliable checks, incomplete coverage, disputes, optional advice declined and
failed regression. Independent assessment owns acceptance; a Fixer report cannot
resolve its own required finding. Release finding/repro/coverage/before-after/
regression states and root-cause links to R42/R43 using the component catalog.

**Accept only when:** Clear is explicit and complete; nothing-found, incomplete and disputed reports keep their distinct semantics. No direct agent merge or authority grant. Native readback, task output and observed retirement gate progression.

**Implementation notes:** Consume R22 review, R23 dispositions, R24 acceptance and R33 singleton output with R30/R31 authority/readback. In the existing MCP mutation consumer, route finish_review to native review ownership and finish_correction to the correction journal/checkpoint owner. Release report, native resolution and effect receipts to R42. Test explicit clear versus incomplete/disputed reports, stale candidate and replay through real handlers/Store; native readback must agree after observed retirement. Preserve independent actor separation and no agent merge authority.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/review_cycle.go](../../../internal/factory/control/review_cycle.go), [internal/factory/control/correction.go](../../../internal/factory/control/correction.go).

<a id="r35"></a>
## R35 — Finish incomplete, handoff and terminal settlement semantics

**Execution tasks:** [R35.2](../factory-remaining-tasks.md#r35-2), [R35.1](../factory-remaining-tasks.md#r35-1).

**Work:** Implement and integrate. **Owner lane:** A. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R11](02-prompt-to-pr.md#r11).

**Recorded baseline:** Incomplete/handoff/terminal settlement composition. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Wire factory.finish_incomplete and factory.request_handoff to the exact blocked owner/approved DAG successor and common stop/settlement path. Preserve pre-start hold/FactoryStart, receipt recovery and terminal capacity.

**Expanded settlement:** Apply incomplete/terminal semantics to all eight
profiles, including Library Admin. Preserve drafts, open required questions,
decision/knowledge references and already committed maintenance operations.
Nonterminal reports do not freeze mutations; one terminal identity does, with
exact replay and later receipt recovery through the authorized owner after
retirement. Native failure after a submitted finish must remain visible. A late
answer requires fresh authorized admission, never revival of an old capability.

**Accept only when:** Blocked/nothing-found/finish are distinct. Submission is not observed termination; stop/submission/reconnect/restart races cannot release unknown children, transfer ownership arbitrarily or charge twice.

**Implementation notes:** Consume R11 settlement and R30/R31 authority/receipts. Start at MCP finish_incomplete/request_handoff, `dispatch_result.go::AccountSettledRun` and host `finish.rs`/`stop.rs`. First persist the exact blocked owner or admitted DAG successor proposal; then join stop observation, children retirement and one settlement/credential return. Release terminal state/custody facts to R41/R42. Run PG submission-versus-stop/reconnect races and one native parent-with-child restart witness. Ordinary partial communication is not terminal; unknown retirement holds custody/capacity, and handoff cannot create a new allowance or arbitrary assignee.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control/dispatch_result.go](../../../internal/factory/control/dispatch_result.go), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs), [lib/host/src/factory/stop.rs](../../../lib/host/src/factory/stop.rs).

<a id="r36"></a>
## R36 — Wire serving stall detection over complete component inventories

**Execution tasks:** [R36.2](../factory-remaining-tasks.md#r36-2), [R36.1](../factory-remaining-tasks.md#r36-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R14](02-prompt-to-pr.md#r14), [R10](01-boundaries.md#r10).

**Recorded baseline:** Generation advancement on verified delivery and full episode consumer composition. Serving detector/events/isolation/order accepted through 3af53d09. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Preserve the accepted serving detector, complete component inventory and bound episode/fence; finish progress-generation and downstream consumers. Separate active work, CI, quota, preparation, permission, pause and product attention from cycles/missing producers/unexplained no-ready work.

**Report consumer:** Consume R32 Blocked/Stuck records as evidence to the existing
readiness/classification owners. A known prerequisite identifies affected work,
responsible owner and release condition; an uncertain cause may route bounded
diagnosis. Concurrent waits/failures must not collapse into a global blocked
state or hide an independently ready component. Release current reason, evidence,
resolution and stale-report states to R42's exception routing; unchanged polling
or a report alone does not launch a new investigation or resolve a blocker.

**Accept only when:** Ordinary waits make no investigator call; a wait in one component cannot hide repairable work in another. Partial inventory cannot prove deadlock. Ready work returns to selection and fully delivered outcomes enter verification.

**Implementation notes:** Detection, repository isolation and ordered events are accepted. Start at `stall_serve.go::ServeStallRound`, `factory_stall_events.go::StallRoundGeneration` and the verified-completion transaction. First advance progress generation from a committed verified required outcome exactly once (attributable merge/completion or accepted evidenced non-code prerequisite); duplicate observation/restart, comments, edits and graph reshuffles must not increment it. Then route classified ready/wait/attention/stall/verify outputs to their existing consumers, with verify handed to R38/R39 when available. Release bound current episodes to R37. Extend serving PG cases for generation and delivered-state handoff; retain mixed-component/isolation tests. Never add another detector/timer or infer deadlock from incomplete inventory.

**Serving proof (landed):** `TestServeStallRoundGenerationReuseAndIsolation` runs two rounds on real PG — generation stays reused at zero, a stalled repo keeps one bound episode and one charge while a ready sibling resumes selection with no episode, charge or standing observation. Open finding: the detector's verify branch has no production producer — nothing writes completed readiness, delivered outcomes flow through completion fences instead. Who advances generation on verified delivery (this card vs the R37 lifecycle owner) and what produces delivery state for the verify handoff is a design decision, not implemented here.

**Principal source owners:** [internal/factory/control/stall_detect.go](../../../internal/factory/control/stall_detect.go), [internal/factory/control/priority_dispatch.go](../../../internal/factory/control/priority_dispatch.go).

<a id="r37"></a>
## R37 — Connect bounded Strategist repair, sponsorship and adoption

**Execution tasks:** [R37.1](../factory-remaining-tasks.md#r37-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R36](04-controls-repair-verification.md#r36), [R29](03-review-merge-reassessment.md#r29), [R11](02-prompt-to-pr.md#r11).

**Recorded baseline:** Strategist proposal/Jev choice and later repair-cycle adoption; cycle-1 episode/charge joint accepted. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Admit one bound repair episode under explicit sponsor and stable intent/repository/global accounting. Consume a checked Strategist proposal and finite Jev choice, then reuse protected serial graph adoption and fresh readback.

**Accept only when:** Restart, Retry, replacement issues and graph reshuffles cannot refill allowances; only material new evidence permits the allowed next cycle. No-change cannot clear a real stall; obsolete ready work retires the episode and committed prefixes remain fenced/reconciled.

**Implementation notes:** Consume R36's current episode, R11's stable charge and R29's protected serial adoption. `repair_admit.go::AdmitStallEpisodeRun` already owns cycle-1 charging. First connect a checked Strategist proposal and bounded Jev choice to that episode; then admit only the allowed later cycle with material new evidence and unchanged cumulative ledger. Release adopted repair or named attention to R39/R40. PG tests cover repeated detection, restart, Retry and reshuffled issue IDs without refill; a native/funded case consumes R07–R09. No-change cannot clear a real stall, and ready work retires obsolete diagnosis.

**Principal source owners:** [internal/store/factory_repair_grants.go](../../../internal/store/factory_repair_grants.go), [internal/store/factory_reassessment_allowances.go](../../../internal/store/factory_reassessment_allowances.go), [internal/factory/control](../../../internal/factory/control).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09). Source-only work cannot release these gates.

<a id="r38"></a>
## R38 — Implement trusted Test Pilot manifests and artifact verification

**Execution tasks:** [R38.1](../factory-remaining-tasks.md#r38-1).

**Work:** Implement and prove. **Owner lane:** A. **Accepted inputs:** [R10](01-boundaries.md#r10), [R13](02-prompt-to-pr.md#r13), [R18](02-prompt-to-pr.md#r18), [R27](03-review-merge-reassessment.md#r27).

**Recorded baseline:** Trusted Test Pilot manifest adoption and exact integrated-artifact checks. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Bind approved manifest/outcome coverage to accepted intent version and integrated tree/artifact/fixture. Launch Test Pilot under the readonly profile and existing bounded host operations, retaining trusted check and cleanup receipts.

**Journey boundary:** Execute the entire approved deterministic manifest before
CLI start as required by D16. Failed preparation/manifest execution is a Factory
event, never an invented Test Pilot report. R40 may record outcome-linked journey
observations and attach existing captures only within admitted capabilities.
New exploratory environments/browser execution are not implicitly authorized by
the recording tool; missing fixtures or evidence remain explicit incomplete cases.

**Accept only when:** P9 uses the actual approved fixture and current integrated bytes. Missing/stale/failed checks cannot pass; forged proposed evidence is not trusted check output. Lost replies and CLI-start ordering preserve identity and no deployment occurs.

**Implementation notes:** Consume R10 delivery identity, R13 accepted outcome coverage, R18 exact integrated artifact and R27 verified delivery. Start at `internal/factory/mcp_manifest.go::ValidateManifestProposal`, `verification.go` and API `FactoryMCPDeps.Catalog`. First wire approved catalog lookup and durable adoption of the proposal; then launch its bounded readonly Test Pilot checks through existing host manifest/run owners. Bind trusted check output independently of submitted report digests. Release exact verification receipts to R39/R40. Test missing fixture, stale manifest/artifact, forged receipt and failed required check at the actual Store/handler join, then native P9 using R30/R44/R45. No deployment or agent-defined shell authority.

**Principal source owners:** [internal/factory](../../../internal/factory), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store), [lib/host/src/factory/manifest.rs](../../../lib/host/src/factory/manifest.rs), [lib/host/src/factory/finish.rs](../../../lib/host/src/factory/finish.rs).

**Additional runtime/qualification gates:** [R07](01-boundaries.md#r07), [R08](01-boundaries.md#r08), [R09](01-boundaries.md#r09), [R30](04-controls-repair-verification.md#r30), [R44](05-experience-and-development.md#r44), [R45](05-experience-and-development.md#r45). Source-only work cannot release these gates.

<a id="r39"></a>
## R39 — Make app completion and delivery repair truthful and bounded

**Execution tasks:** [R39.1](../factory-remaining-tasks.md#r39-1).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R38](04-controls-repair-verification.md#r38), [R13](02-prompt-to-pr.md#r13), [R29](03-review-merge-reassessment.md#r29), [R37](04-controls-repair-verification.md#r37).

**Recorded baseline:** Verified app completion and bounded delivery-repair chain. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Complete an intent only from all accepted delivered outcomes plus its exact verification receipts. Route real verification failure through the distinct bounded delivery-repair cause, stable intent ledger and protected graph owner.

**Accept only when:** Intent A can verify while unrelated B progresses. A missing fixture is a precise per-intent wait. Failure/Retry/revision cannot refill the two-run ceiling or manufacture an app intent for a manual issue; material scope questions return to the owner.

**Implementation notes:** Consume R38 trusted receipts, R13 accepted scope and R29/R37 protected repair/ledger contracts. Start at `verification.go::{DeliverySubject,DeliveryRepairSubject}` and existing intent/allowance Store owners. First make intent completion require all accepted outcomes and exact artifact verification; then route verified failure to the distinct delivery-repair cause with its two-run ceiling. Release completed/failed/waiting state to R40/R42. Real-PG cases cover unrelated intent progress, missing fixture, partial coverage, duplicate result, Retry and revision without allowance refill. A manual issue gains no synthetic app intent, and material scope change returns through existing product authority.

**Principal source owners:** [internal/factory](../../../internal/factory), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store).

<a id="r40"></a>
## R40 — Implement Strategist and Test Pilot work and finish controls

**Execution tasks:** [R40.1](../factory-remaining-tasks.md#r40-1), [R40.2](../factory-remaining-tasks.md#r40-2).

**Work:** Implement and integrate. **Owner lane:** B. **Accepted inputs:** [R30](04-controls-repair-verification.md#r30), [R31](04-controls-repair-verification.md#r31), [R29](03-review-merge-reassessment.md#r29), [R37](04-controls-repair-verification.md#r37), [R38](04-controls-repair-verification.md#r38), [R39](04-controls-repair-verification.md#r39).

**Recorded baseline:** Strategist/Test Pilot finish effects and seven-profile instruction/context matrix. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Connect typed role prompts/results and observed retirement to the reassessment/repair and exact-artifact verification/app completion owners. Finish the seven-profile instruction/context matrix.

**Full role interfaces:** Implement Strategist `reassessment_scope`,
`record_diagnosis`, `draft_repair`, `preview_repair` and its finish for the admitted
completion/stall/delivery-repair cause. Use complete inventories, preserved
protected work, exact proposal revisions and a deterministic no-effect preview;
R29/R37 apply accepted changes. Implement Test Pilot `verification_scope`,
`record_journey_result` and its finish with exact outcome, delivery, fixture,
expected/observed behavior and trusted receipts from R38/R39. Journey failures
retain canonical IDs through repair; no duplicate Flag submission is required.
Persist drafts/observations and expose their limits to R42 before finishing.

**Expanded acceptance:** Distinguish incomplete inventory, ordinary wait, real
stall, refused/no-change repair, partial adoption and confirmed new readiness.
Distinguish failed/incomplete/stale journey evidence from a verified delivery;
all required coverage survives paging and revisions. Feed diagnosis/comparison/
journey components, decision logs and reusable lessons reference the same facts.
Reconcile the seven production instructions here; R32 owns Library Admin's
instruction/result schema and R30/R48 qualify the combined eight-profile matrix.

**Accept only when:** Both controls have authenticated producer→durable admission→permitted effect→native/host readback→visible state, including stale/blocked/refused/replay cases. Tool completion grants no merge or deployment authority.

**Implementation notes:** Consume R29/R37 proposal adoption, R38/R39 verification/completion and R30/R31 admission/readback. Start at MCP finish_reassessment/finish_app_verification payloads and `dispatch_prompt.go::BuildDispatchPrompt`. Wire each recorded submission to its existing consumer and retirement gate; then reconcile instructions/context for all seven profiles, including permitted communication and comment provenance. Release durable effect/readback references to R42/R48. Handler/Store tests distinguish recorded, refused, waiting and applied across replay; actual native receipts complete qualification. Tool success grants neither merge nor deployment.

**Principal source owners:** [internal/web/api/factory_mcp.go](../../../internal/web/api/factory_mcp.go), [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store).

<a id="r41"></a>
## R41 — Qualify intervention, withdrawal and interrupted-work recovery

**Execution tasks:** [R41.1](../factory-remaining-tasks.md#r41-1).

**Work:** Integrate and qualify. **Owner lane:** A. **Accepted inputs:** [R06](01-boundaries.md#r06), [R11](02-prompt-to-pr.md#r11), [R16](02-prompt-to-pr.md#r16), [R26](03-review-merge-reassessment.md#r26), [R30](04-controls-repair-verification.md#r30), [R35](04-controls-repair-verification.md#r35), [R44](05-experience-and-development.md#r44).

**Recorded baseline:** Composed intervention/withdrawal/crash recovery. The catalog owns the remaining prerequisites and release condition.

**Deliver:** Compose pause/cancel/resume/Retry/takeover with exact native tasks, outstanding operations, MCP capability withdrawal and credential/resource return. Preserve finite consumption and human work.

**Maintenance and conversation recovery:** Include R32 drafts, decisions,
knowledge and collection-maintenance effects when testing interruption. Preserve
committed edits/removals/consolidations/splits and recover their original receipts;
cancel does not roll them back or repeat them. A stopped Library Admin cannot
continue through an old capability. Open questions/proposals retain history and
their real next owner; delayed feed/index work reconciles existing obligations
without reopening the worker, making a new charge or repeating a native effect.

**Accept only when:** Observe whole owned execution retirement before release or takeover; uncertain stop remains fenced. Restart adopts committed effects instead of duplicating them. Human terminals, checkouts, services and credentials remain distinct and usable.

**Implementation notes:** Consume R06 importer custody, R11 settlement, R16 child journal, R26 operation reconciliation, R30 capability withdrawal, R35 terminal semantics and R44 provider custody. Start at host `stop.rs`, `project/admission.rs` and the existing controller pause/cancel/Retry paths. First compose withdrawal with active parent/children and outstanding native operation; then restart/takeover after known retirement versus uncertain stop. Release observed custody-return facts to R49/R54. Focused PG races plus native interruption at those two boundaries are sufficient; preserve committed effects, charges and unrelated human checkout/terminal/service data.

**Principal source owners:** [internal/factory/control](../../../internal/factory/control), [internal/store](../../../internal/store), [lib/host/src/factory/stop.rs](../../../lib/host/src/factory/stop.rs), [lib/host/src/project/admission.rs](../../../lib/host/src/project/admission.rs).

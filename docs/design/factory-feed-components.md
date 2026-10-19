# Factory feed components by worker role

This catalog expands the [worker MCP redesign](factory-worker-mcp.md) into the
feed vocabulary for the seven production roles and Library Admin. It covers the
currently designed Factory lifecycle, including incomplete work, questions,
corrections, repair and shared knowledge. All 121 numbered variants now have
renderers and producer paths; broader issue/research surfaces and domain
interactions retain their explicit task scope below.

The [feed integration design](factory-feed-and-ralph-integration.md) owns the
combined subscribed feed and its relationship to Work/Spaces. This document owns
the role-specific component inventory, content, interactions, record bindings,
state behavior and [non-happy-path selection rules](#failure-component-routing).
The worker MCP proposal owns the candidate tool surface. The
[structured issue presentation](#structured-issue-presentation) renders its work
contract in the native issue and links it to feed activity. These are linked
parts of the same design.

The [full task integration](../development/factory-remaining-tasks.md#full-worker-design-integration)
assigns this catalog's producers, projections, UI and failure/recovery cases to
R32–R43 and R47–R49, with installed consumers in R52. Component names remain
design vocabulary; all families and their actual interactions are required scope.
The [restructured visual coverage map](../development/factory-remaining-tasks.md#complete-visual-and-interaction-coverage)
maps all 121 numbered families plus issue/research surfaces to concrete Rxx.y
execution tasks; it replaces broad appended UI scope as the implementation map.

### Implementation update · 19 October 2026

The executable component catalog is [`internal/factory/feed_components.json`](../../internal/factory/feed_components.json). It contains all 121 numbered variants with their role, allowed outlet, presentation and authority. The frontend renders those variants with 19 compositions, including questions with reply choices, proposals, progress and remaining work, blockers, diagnosis, evidence/check/finding tables, journeys, dependency items, comparisons, decisions, knowledge and maintenance mappings. Large item lists show four entries with the remainder expandable; supporting detail stays collapsed. The author, role, project, time and reported status remain outside each separate card.

Shared MCP communications now carry typed content rather than being inferred from a worker's prose. A question choice opens and fills the existing reply draft; it does not submit or invoke JEV until the person presses Submit. Worker reports never acquire a confirmed check, merge or completion state from their text. Structured role finishes retain their selected component on the canonical workflow record. Components use the same light/dark CSS and reflow on narrow screens. A local browser gallery rendered all 121 variants at desktop and mobile widths without horizontal overflow; keyboard reply selection was checked through the real feed controller.

The nine Library Admin variants now correspond to actual revisioned operations: scope, edit, consolidate, split, remove/restore, supersede, relate, affected-work report and finish. Knowledge and decision posts create searchable project-library entries; users can inspect current content, relations and revision history. Decisions retain the original choice and authority across maintenance. Authenticated artifact cards lazily preview images, video, audio and text, with downloads for other supported files. Larger text is bounded and explicitly truncated. Dependency graphs draw only supplied edges and preserve a textual alternative. Controller-owned posts come from their actual recorded operations; a worker cannot publish a trusted Factory effect by selecting its component ID.

The website movie's action-specific compositions now inform the real feed: investigation dossiers connect sources to observations; decisions connect considered inputs to the recorded choice; findings show annotated source and expected/observed behavior; corrections connect findings to changes; commit trails show saved changes; integration and dependency-release cards draw their different relationships. Questions, stalled investigations, blockers, proposals, knowledge notes, journeys and handoffs each have their own layout and next-action language. Canonical plan/review/correction/judgment/verification records use their typed facts for these visuals, with existing binding and receipt details in an expandable section. Connections stay inside each card; posts remain separate and newest first, with author metadata outside. No authored movie example is presented as live work.

MCP content can include a bounded `source` excerpt with path, revision, starting line, highlighted lines and an optional changed excerpt, plus up to eight `commits` with full revisions and summaries. These render supplied evidence; they do not imply passed checks or acceptance. Excerpts render as escaped text, preserve line numbers and scroll within the card. Missing code remains missing rather than being fabricated from a finding's prose.

## 1. What makes the roles visibly different

The primary content should reflect what each role is doing:

| Role | The person's main question | Distinct visual language |
| --- | --- | --- |
| Product Lead | Are we building the right experience? | People, scenarios, outcome maps, choices and scope changes. |
| Architect | How will this work, and what depends on what? | Explanatory diagrams, interfaces, evidence, task graphs and ownership. |
| Builder | What is being built and what actually works so far? | Task lanes, working artifacts, change previews, checks and checkpoints. |
| Inspector | What was checked, and what is wrong? | Coverage, precise findings, reproductions and annotated evidence. |
| Fixer | What changed to address each problem, and did it work? | Finding-to-fix links, before/after comparisons and regression evidence. |
| Strategist | Why did work stall or change, and how will it recover? | Cause maps, inventory gaps, plan comparisons and repair impact. |
| Test Pilot | Does the complete product deliver the promised experience? | Journeys, expected/observed behavior, recordings and outcome coverage. |
| Library Admin | Is our knowledge and decision history accurate, usable and current? | Item changes, consolidation/split mappings, replacements and unresolved contradictions. |

The role is the author, the communication action explains intent, and the
component explains the work. An Architect's plan may contain an approach diagram,
task graph and check matrix in one post. Ask can appear inside any role's post,
with the question expressed in that role's context.

**There is no one-to-one rule between components, posts and MCP tools.** One
canonical plan can support several useful components. One feed post can compose
them. Several recorded events can update one continuing component. Every state
and action still comes from the owning domain record and permission checks.

### Approved role colors · 19 October 2026

Every role has one persistent color identity, shared by feed posts and role
artwork. The canonical values are the `--soda-role-*` tokens in
[`palette.css`](../../assets/branding/theme/palette.css):

| Feed author | Color |
| --- | --- |
| Product Lead | Soda red |
| Architect | Violet |
| Builder | Orange |
| Inspector | Emerald green |
| Fixer | Hot pink |
| Strategist | Cobalt blue |
| Test Pilot | Cyan |
| Library Admin | Golden yellow |
| JEV / System One | Bone white / ivory |
| Factory / System | Graphite |

The exact approved color marks the card's top edge. Role portraits, card
surfaces and details carry the same hue; text uses a readable shade in each
theme. Role identity follows the author across every post type and reported
status. Human comments use the neutral discussion treatment. The ten PS1
portraits in [`assets/branding/role-portraits/ps1-v1/`](../../assets/branding/role-portraits/ps1-v1/)
are shipped with the native Soda extension and selected from the post's
server-derived author role. System One aliases use JEV; System uses Factory.
The portrait mapping in
[`sodaspaces-factory-feed-portraits.ts`](../../frontend/spaces/sodaspaces-factory-feed-portraits.ts)
also owns the packaged image paths. Human and unrecognized authors retain their
initials rather than borrowing a worker's portrait from their name or post type.

## 2. Common anatomy and reusable components

<a id="work-post-presentation"></a>
### Default: work posts with useful visual detail

The owner's constraint is to avoid both literal tweets and entire applications
embedded in the feed. A post should give a useful account of a change, decision,
question or result, with enough visual detail to understand the work. Post size
follows that purpose; there is no character limit, fixed post count per viewport
or universal tiny-attachment rule.

The combined feed orders newest posts first. Each post is clearly separated
with its own boundary and spacing. Do not connect posts with a timeline spine,
avatar rail or stage chain, or nest successive role posts inside one issue
wrapper in the default feed. Related work remains linked and its discussion can
open on request. Diagram edges inside an individual post still explain that
post's actual work relationships.

The proposed composition is a recognizable author header, a concise explanation,
a focal role-specific visual or interaction, relevant status/next action, and
discussion/evidence access. These elements form one coherent post:

- Place the profile picture, author name/role, project, time and status outside
  the bordered content card, in an unboxed header immediately above it. Keep the
  communication verb distinct from authoritative status where both are shown.
  The header is close to its own card; a larger gap separates the previous card
  from the next header. No enclosing frame wraps the header and card together.
- A modest sentence-case title is useful when it helps scanning; a short
  explanation supplies the consequence and context.
- Give the important visual enough room to be useful: task/check progress, an
  approach diagram, an experience walkthrough, an answerable question, a finding
  or a before/after comparison. Related status or supporting evidence can appear
  beside it when needed; do not flatten all role work into a one-line link.
- Keep failure, uncertainty and pending state visible alongside progress. A
  curated view must not imply that an incomplete or failed result succeeded.
- Use a restrained footer for relevant actions and discussion. A useful reply
  preview may appear, while full threads, task boards, logs, large graphs and
  detailed evidence open on request in the existing work views.

Avoid reproducing a workspace's navigation, tabs, settings and every available
component inside each post. Avoid forcing all posts into identical height or
microblog formatting. Simple events can be brief; explanations and decisions can
use more space without becoming a whole screen. Narrow layouts reflow at readable
text size, and opening detail preserves the surrounding feed position.

The images in section 16 are proposals for this balance, not owner-approved
pixel dimensions or final role portraits. Both earlier oversized and literal
tweet mockups are superseded as default presentation references.

### Every post has a predictable reading order

1. **Who and where:** role/person/system, product/project and relevant work item.
2. **What changed:** a concise title or message and the current reported/confirmed state.
3. **Why it matters:** a short explanation in the language of the affected user.
4. **The useful visual or artifact:** enough role-specific detail to understand the work.
5. **What follows:** next machine action, a precise question, or a confirmed result.
6. **Evidence and discussion:** expandable sources, history and replies.

Role avatars help identification, but cannot carry meaning alone. Use readable
names and stable role identity; optional specialties such as Security remain a
separate descriptor. Model/provider/run details belong in an inspectable author
disclosure. They should not dominate the product narrative.

<a id="structured-issue-presentation"></a>
### Structured native issue presentation

The owner requires Soda issue metadata to be visually structured. The
[issue work contract](factory-worker-mcp.md#issue-work-contracts) therefore needs
a readable issue detail and a matching editor, as well as compact list/feed
previews. This is required design scope; the exact layout below is the proposed
composition. Researcher remains a proposed role, and visualizing a workflow does
not mean that workflow is implemented.

Keep the native issue identity, title, Open/Closed state, author and discussion.
Render the Soda block into labeled sections above the discussion. Its raw source
is available through a secondary action. The existing native issue shells in
`frontend/forgejo/templates/repo/issue/view.tmpl` and `new.tmpl` remain entry
points; the existing Lit Work detail uses the same authorized contract projection.
Do not introduce an independently editable Soda copy of the issue.

| Area, in reading order | Visible content and interaction |
| --- | --- |
| Issue header | Native number/title/state, project, author and time. A separate Factory state says unclassified, ready, running, waiting, result under review or completed only when supported by the owning records. Native Closed alone does not mean an accepted result. |
| Work summary | Labeled work type, primary and supporting expertise, and expected output. Example: **Research · Security / Backend · Knowledge report**. Expertise never appears as a new role. |
| Objective and required results | Human description and concrete questions/outcomes. Each deliverable has acceptance criteria, a result link when submitted, and its actual assessment state. Criteria are readable rows; checking a decorative checkbox cannot accept an output. |
| Workflow and current responsibility | Requested workflow, its registered stages, current role/specialty and actual assignment if any. A short “Why this route” disclosure explains the recorded routing choice. Distinguish proposed route, admitted work and observed execution; list only stages required by this output contract. |
| Waiting on / needed by | Two directional groups with the linked issue, exact required output, relationship state and release condition. Keep related/duplicate links separate. A research issue can show who needs its findings; a parent can show why an accepted finding still fails its requirement. |
| Context and boundaries | Relevant knowledge and decisions, source/components/paths, approach guidance, exclusions and inherited limits. Use short summaries with expandable references. Requested capabilities and actual granted access are distinct. |
| Results and evidence | Type-appropriate artifacts, exact source revisions, known limitations and acceptance. Reuse the same knowledge/decision records displayed in the library and feed. Show next action or unresolved outcome without inventing a completion percentage. |
| Discussion and history | Existing native conversation and attributed contract changes, with links to role activity. Preserve comments, attachments and human context outside the structured block. |

Use the established Soda typography, thin borders, restrained color and spacing.
The objective and results are the visual focus. Compact labeled fields can sit
side by side on wide screens; longer requirements and dependencies need full
width. On narrow screens these become one reading order. Keep the active blocker
and next required action visible rather than hiding them in collapsed context.
Status needs text as well as color; controls retain labels, keyboard access and
focus. A workflow diagram inside issue detail does not change the separate,
newest-first feed post layout.

**Adapt the content to the work type.** Research emphasizes questions, sources,
findings and uncertainty; implementation emphasizes behavior, scope, changes
and checks; diagnosis shows reproduction and evidence for the cause; design
shows constraints, alternatives and the specification; verification shows
expected versus observed outcomes. Decision work shows options, tradeoffs and
the actual decision owner; planning shows proposed dependencies and their
adoption; curation shows affected items, changes and preserved references. Keep
the common structure, but omit irrelevant controls such as a PR requirement on
knowledge-only work.

**Editing follows the same structure.** Offer objective, work type, expertise,
deliverables and acceptance first. Selecting a work type suggests an output
template and a supported route; advanced context, boundaries and relationships
expand when needed. Prefill a worker-created prerequisite from its originating
blocker, including the exact needed output. Show inherited values and their
source; changing work type must preserve existing text/results and surface
incompatible fields before saving, not silently discard them.

Structured and raw editing round-trip the same strict native block, preserving
the surrounding description. Validate field combinations and required-result
links, with inline errors and an error summary. Saving a draft does not admit
work. For already accepted inputs, show the meaningful change and its effect on
future routing/dependencies; distinguish the edited issue from the contract an
active worker still holds. Use the existing authorized revision/adoption path,
without adding a blanket human approval for every edit. Concurrent changes keep
the user's draft and offer comparison/reload instead of overwriting newer input.

**Failure states belong in the design.** Missing metadata offers classification
without enrollment; malformed or unsupported metadata preserves the issue and
shows the specific repair needed. No eligible role/capability gives an actionable
wait. Partial issue creation/linking shows the created issue and pending link
separately. An accepted negative research result can coexist with an unsatisfied
parent dependency; inconclusive work retains useful findings. Loading,
unavailable, stale and inaccessible data are distinct from an empty section or
success. Recheck access for links/previews/counts, preserving historical result
references without exposing restricted material.

Issue list rows summarize work type, primary expertise, expected output and the
material wait/state. A related feed post uses the existing outside-card author
header and a focused result/question/dependency preview with an “Open issue”
action. The full issue form is not embedded in every post. Opening the native
issue or Work detail must agree on identity, accepted inputs and observed state.
R42 owns that projection; R43 owns the detail, editor and compact presentations.

### Shared component catalog

These are available to all production roles and the proposed Library Admin,
composed into role-specific posts.

| ID / component | Compact view → expanded view | Useful interaction | Producer and important states |
| --- | --- | --- | --- |
| S01 · Author and work context | Role, work title and stage → run attribution, exact source and related work | Open the work or inspect attribution | Admitted identity and canonical work links; distinguish role, Factory, Jev and human authors. |
| S02 · Question and answer | Question, consequence and whether other work continues → options, recommendation, evidence and answer history | Answer a choice, add a free-text correction, inspect why it matters | Ask/Answer and authorized human decision route; open, answered, incorporated, superseded. Answered does not mean incorporated. |
| S03 · Proposal and disposition | Proposed change and expected benefit → alternatives, impact and assessment | Discuss, inspect changes, take an authorized adoption action where needed | Suggest and existing proposal consumers; proposed, assessed, adopted, declined, deferred, stale. |
| S04 · Progress and remaining work | Concrete completed milestone and next work → task/evidence links and prior updates | Inspect the completed artifact or what remains | Partial plus observed work records; reported progress does not override verified task counts. |
| S05 · Known blocker | What is missing, affected work and release condition → prerequisite, responsible owner and attempted resolution | Open the prerequisite or the actual resolving control | Blocked plus readiness owner; reported, assessed, waiting, resolved, stale. |
| S06 · Diagnosis needed | Why progress stopped and what was tried → failed approaches, evidence and remaining machine routes | Inspect diagnosis or answer a necessary question | Stuck plus bounded repair routing; diagnosis pending, assigned, resolved, exhausted. Do not label it as an already understood dependency. |
| S07 · Evidence strip | Source/check/artifact references with evidence state → exact revision, completeness and provenance | Open evidence at the referenced version | Existing evidence owners; available, incomplete, stale, inaccessible, unavailable. No invented pass from an attachment. |
| S08 · Artifact preview | Useful image, diagram, document or trace preview → safe detailed viewer and provenance | Expand, compare or open an authorized native artifact | Attach Artifact and existing capture records; illustrative, observed, failed load, revoked. |
| S09 · Discussion and reply | Relevant replies, unresolved questions and comment-to-work result → ordered revision-aware thread | Submit a comment, inspect JEV routing and follow the LLM-authored issue or existing-work link without losing reading position | Persisted comment/revision and assessment/authoring/native-operation references; sending, saved, assessing, drafting issue, created, linked existing work, discussion-only, clarification needed, unavailable, uncertain, failed, changed-source, read-only. Human submission invokes JEV; drafts, reads and agent-generated replies do not. Issue creation and worker dispatch remain distinct. |
| S10 · Revision comparison | What changed since the last version → added/removed/changed fields and rationale | Compare exact revisions or open current | Domain revisions; current, superseded, conflicting edit, deleted source. |
| S11 · Result and next stage | Role result submitted and what follows → completeness, acceptance, retirement and confirmed effects | Open resulting plan, candidate, review or delivery | Finish plus actual consumers; submitted, verifying, accepted/refused, effects pending/confirmed. |
| S12 · Incomplete work handoff | Useful work preserved and why it ended → unresolved items, evidence and restart prerequisites | Inspect preserved work or the authorized recovery action | Finish Incomplete and settlement; submitted, retirement pending, ended, recovery available. |
| S13 · Knowledge reference | Applicable lesson and its support/validity label → rationale, sources, revisions and later use | Read, challenge or propose an amendment | Canonical library entry; observed, supported, contested, stale, superseded. Detailed variants are in section 10. |
| S14 · Attention and permissions | Exact decision needed or why an action is unavailable → owner, consequence and permitted path | Use only the relevant authorized action | Existing authority/attention owners; actionable, waiting, unavailable, resolved. No generic Approve button on every post. |
| S15 · Freshness and completeness | “Source changed” or “Some evidence is missing” → affected facts, last observation and refreshing state | Open current evidence or request an allowed refresh | Read-model metadata; loading, complete, incomplete, stale, unavailable. Refresh does not execute work. |
| S16 · Recorded decision | Chosen option, reason and who decided → alternatives, tradeoffs, scope, evidence, authority and application history | Read the decision, inspect alternatives or propose a change | Record Decision or the actual human/controller decision source; distinguish logged choice, operative authority, pending application and supersession. Logging does not apply a plan. |

These are semantic component families, not mandatory boxes on every post.
A default post presents the components needed to explain its main point, with a
clear visual focus and a restrained action row. Remaining details expand in
place or open the existing work view.

## 3. Product Lead components

Lead with the user's future experience. Engineering decomposition is inspectable
detail. Preserve the [nontechnical review contract](spaces-ux.md#review-for-nontechnical-people):
concrete people/actions/results, optional correction and only consequential
unresolved choices interrupting work.

**Primary producers:** accepted intent/outcome records, `product_map`,
`draft_product_plan`, `compare_product_plans`, Ask, Suggest, Learned and
`finish_product_plan`. Tool names are candidates from the MCP redesign.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| PL01 · Product direction brief | Who benefits, the problem and intended result → supplied intent, constraints and explicit exclusions | Change something in the proposed behavior | Exact intent/proposal revision; supplied, proposed, accepted, superseded. |
| PL02 · People and access map | Who participates and who can see/change what → permission scenarios and consequential assumptions | Inspect a person's experience or correct a visibility assumption | Accepted or proposed access model; unknown permission remains a question. |
| PL03 · Experience walkthrough | Three to six meaningful actions and outcomes → branches, exceptions and an illustrative example | Step through the scenario or comment on one step | Same product proposal as the summary; illustrative screen is labeled, not presented as implemented. |
| PL04 · Outcome and acceptance map | Desired outcomes and uncovered items → observable success conditions and proposed work coverage | Open an outcome and its supporting work | Accepted outcomes and candidate mapping; proposed coverage is distinct from delivered coverage. |
| PL05 · Assumption challenge | One consequential assumption and its effect → source, alternatives and recommended resolution | Correct the assumption or answer the actual required choice | Assumption/question record; supplied, evidenced, chosen within scope, unresolved, rejected. |
| PL06 · Product alternatives | Recommended experience plus meaningful tradeoffs → alternatives evaluated against the same outcomes | Compare consequences or suggest a correction | Saved decomposition candidates; recorded recommendation, pending assessment, selected, superseded. Do not invent confidence scores. |
| PL07 · Delivery milestone story | What a usable first milestone provides and what follows → outcome coverage and real dependencies | Inspect a milestone's end-user experience | Proposed work graph; milestones are not dates or delivery promises unless separately supported. |
| PL08 · Issue and dependency preview | Cohesive proposed work items with prerequisite explanations → issue/outcome graph and resulting native links | Inspect why an edge is necessary or open an existing issue | Draft graph, adoption and native issue receipts; proposed, applying, partially applied, confirmed. |
| PL09 · Scope change comparison | Added, changed and removed behavior → reused, new and superseded work plus active-work consequences | Review the changed experience through the existing scope decision route | Revision-bound proposal; proposed, accepted, declined, stale. Declining does not undo delivered work. |
| PL10 · Product decision rationale | The recorded choice, consequence and why → alternatives, authoritative answer and affected outcomes | Read its source, discuss applicability or open library entry | Authorized product decision and K02 library projection; an agent suggestion remains a proposal. |
| PL11 · Decomposition result | Work proposal submitted and outstanding concerns → full coverage, validation and created issues | Follow adopted issues or inspect refusal/partial adoption | Product-plan finish and graph consumer; submitted versus confirmed native work. |

**Example composition:** “Expense requests will go to the employee's team manager.”
Show the experience walkthrough, a small people/access map and one open question
about substitute approvers if that consequence is unresolved. Expand to outcome
coverage and the issue graph. Routine evidence-backed choices do not become a
stack of approval forms.

**Knowledge contribution:** accepted domain definitions and product decisions,
their reasons, and the exact products/outcomes to which they apply.

## 4. Architect components

Make structure and reasoning inspectable: interfaces, prerequisites, boundaries
and checks. Diagrams should explain relationships rather than decorate a summary.

**Primary producers:** `issue_context`, `record_investigation`,
`draft_issue_plan`, `validate_result`, `finish_issue_plan`, and shared reports.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| AR01 · Investigation brief | The technical question and relevant boundaries → accepted issue, prerequisites and source | Open the contract or prerequisite being investigated | Current issue/source bindings; investigating, missing input, stale source. |
| AR02 · Evidence and investigation ledger | Important findings with their sources → inspected files/interfaces, checks and unresolved facts | Follow a finding to exact evidence | Versioned investigation records; inspected does not imply passed or exhaustive. |
| AR03 · Assumption or spike result | Hypothesis, observation and design consequence → method, limitations and alternatives | Inspect the experiment evidence or the resulting question | Admitted investigation evidence; supported, contradicted, inconclusive. No new arbitrary execution grant. |
| AR04 · Architecture and data-flow diagram | A small map of components and responsibility → interfaces, data movement and failure paths | Select a boundary to read its contract; use a text/list equivalent | Typed investigation/plan relationships; proposed versus existing structure clearly labeled. |
| AR05 · Interface contract change | What changes for a caller or consumer → before/after contract, affected consumers and transition constraints | Compare the contract or inspect impacted work | Plan/interface ownership and exact source; proposed, accepted in plan, source changed. |
| AR06 · Design decision comparison | Chosen approach and decisive tradeoff → alternatives, evidence and rejected assumptions | Compare an alternative or challenge a specific premise | Investigation decision artifact; do not attribute invented rationale to Jev. |
| AR07 · Execution dependency graph | Tasks that can run together and their joins → required outputs, ownership and dependency reasons | Focus a task/lane or switch to the ordered accessible list | Draft/accepted immutable task graph; draft, invalid overlap/cycle, ready plan, superseded. |
| AR08 · Ownership and boundary map | Who owns each independent area → exact paths/interfaces and serialized conflicts | Inspect a conflict and the planned resolution | Task/lane ownership validation; this is planned ownership, not a claim of per-path OS isolation. |
| AR09 · Verification plan | How the approach will be checked → outcome-to-check map, required integration checks and fixtures | Inspect which check covers a requirement | Proposed/accepted CheckSpecs; defined is not executed, optional is not required. |
| AR10 · Executable-plan readiness | Required inputs resolved or the exact remaining gap → coverage, ownership and source-freshness checks | Inspect why execution is held | Deterministic validation; provisional, missing fact, invalid, current and ready, stale. No decorative universal green checklist. |
| AR11 · Plan handoff brief | Approved approach, first ready work and critical constraints → sealed plan, knowledge references and acceptance | Follow the Builder's admitted task view | Plan finish/acceptance and later dispatch; submitted plan is not a running Builder. |

**Example composition:** “Permission checks belong in the service boundary.”
Show the architecture map, the reason client-only checks are insufficient in the
illustrative product, two independent task lanes and the integration check that
joins them. Expand to investigated source and the precise ownership map.

**Knowledge contribution:** interface facts, architecture rationale, validated
patterns and failed design approaches with applicability and source versions.

## 5. Builder components

Show the evolution of working software. Observed task/checkpoint records supply
state; the worker's narrative explains meaningful decisions and obstacles.

**Primary producers:** `tasks`, `task_inputs`, native task observation,
`submit_task_output`, `request_handoff`, check/checkpoint receipts,
`finish_implementation`, Partial, Flag, Suggest and artifacts.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| BU01 · Task lane board | Ready, running, waiting and verified tasks → outputs, dependencies and observed worker binding | Focus a task or inspect why it waits | Immutable plan plus observed task states; counts identify their task denominator. |
| BU02 · Current work focus | What a worker is implementing and the next meaningful result → task inputs, bounded progress and source | Open the task or watch permitted read-only output | Admitted task and native run; working, idle observation, unknown, stopped. No fake typing/activity pulse. |
| BU03 · Dependency and handoff strip | Output awaited or released and the successor it enables → checkpoint, exact input and release condition | Open predecessor evidence or the successor | Approved edge and verified handoff/checkpoint; requested, waiting, released, stale. |
| BU04 · Implementation change preview | User-visible change and affected area → exact diff, files, decisions and scoped checks | Open native diff at the candidate or discuss a change | Candidate/output records; work in progress, submitted, checkpointed, superseded. |
| BU05 · Working artifact preview | Screenshot, UI interaction or relevant generated output → authorized artifact and source provenance | Expand the preview or open a permitted running example | Admitted artifact/capture; illustrative, observed on candidate, unavailable. Do not imply a mockup is running software. |
| BU06 · Check execution strip | Required checks with current outcomes → actual command/spec, candidate, output and completeness | Inspect a failed check or its evidence | Trusted check receipts; pending, running, passed, failed, stopped, unknown. No worker-authored pass badge. |
| BU07 · Verified checkpoint | Coherent completed output and newly enabled work → tree/commit evidence, integration checks and successors | Open checkpoint or the exact change | Trusted checkpoint receipt; pending verification, verified, failed, no confirmed checkpoint. |
| BU08 · Failed approach and recovery | Attempt that failed and current next step → failure evidence, preserved work and remaining allowance | Inspect the failure or the diagnosis request | Observed failure plus Partial/Stuck; retry proposed versus actually admitted remains explicit. |
| BU09 · Plan change request | Why the accepted plan no longer fits and what change is proposed → affected tasks/interfaces, evidence and protected work | Discuss the proposal and follow its assessment | Suggest bound to current plan; pending, adopted through replacement, declined, stale. Builder cannot edit the accepted graph in the feed. |
| BU10 · Implementation result | Candidate submitted, completed tasks and remaining downstream stages → complete task/check evidence and publication state | Open candidate, PR when confirmed, or incomplete evidence | Implementation finish/publication records; submitted, verified for publication, publication pending/confirmed. Never label merged. |
| BU11 · Implementation lesson | Reusable pitfall or technique and when it applies → root cause, source and supporting check | Open library entry or add contrary evidence | Learned/K01/K03; an observation retains its support and validity label. |

**Example composition:** “The request form and permission service are ready for
integration checks.” Show task lanes and the actual checks; place a screenshot
beside the relevant UI task, with its candidate label. A failed integration check
appears beside affected tasks instead of a misleading global completion bar.

**Knowledge contribution:** demonstrated environment facts, implementation
pitfalls, useful techniques and unsuccessful approaches worth avoiding.

## 6. Inspector components

Make the review inspectable and independent. A clear review is valuable output
with visible coverage; a defect needs evidence and a specific consequence.

**Primary producers:** `review_scope`, `record_review_coverage`, `assess_finding`,
Flag with the Inspector finding schema, `finish_review`, native review and evidence.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| IN01 · Review brief | Exact candidate, accepted behavior and review focus → base/diff, policy and complete carried finding scope | Open the reviewed change or obligations | Canonical review tuple; current, source moved, incomplete inputs. |
| IN02 · Review coverage matrix | Required areas inspected and gaps → method, evidence and limitations per outcome/area | Open a coverage row and its proof | Recorded coverage bound to candidate; inspected, missing, inconclusive, stale. “Nothing found” is not “nothing inspected.” |
| IN03 · Actionable finding | Concrete defect, impact and location → expected/observed behavior, source evidence and remedy direction | Open exact native location, discuss or follow correction | One canonical finding; draft, published, carried, disputed, resolution pending, accepted resolved. |
| IN04 · Reproduction recipe | Minimal steps and observed failure → fixture, inputs, environment and evidence | Inspect or copy permitted steps; follow an existing execution result | Finding/repro references; reproducible evidence, incomplete repro, no longer reproducible on current candidate. UI does not run commands by itself. |
| IN05 · Annotated evidence comparison | Relevant code/screenshot/result difference → exact version and surrounding context | Expand evidence or compare source versions | Evidence/artifact owner; current versus stale anchors, inaccessible source. |
| IN06 · Carried-finding assessment | Whether an earlier issue remains on this candidate → prior claim, correction and independent reassessment | Inspect the same finding through rounds | Canonical finding identity and `assess_finding`; still present, resolved with evidence, insufficient evidence. |
| IN07 · Check reliability assessment | Which checks support the review and which do not → missing, skipped, stale or mismatched evidence | Inspect the exact check/candidate mismatch | Check receipts plus review assessment; green on another candidate is visibly insufficient. |
| IN08 · Optional advice | Improvement suggestion and benefit → tradeoffs and why it is optional | Discuss or follow its accepted/declined disposition | Optional review advice/Suggest; never visually escalated into a required defect. |
| IN09 · Related-findings group | Shared impact or suspected common cause → individual findings and distinct evidence | Expand all affected findings without losing identities | A view over canonical findings; grouping is explanatory and does not automatically deduplicate findings. |
| IN10 · Review verdict | Clear, findings or incomplete with coverage explanation → complete report, native publication and next permitted route | Open report or follow the correction stage | Review finish/consumer; submitted versus published/accepted, stale after candidate change. |
| IN11 · Regression pattern | Reusable failure class and detection method → confirmed scope, examples and source-linked lesson | Open or challenge the library entry | Learned/K03/K04; a corrected defect does not become a timeless claim that the component is broken. |

**Example composition:** “A manager can view expenses belonging to another team.”
Show the impact, expected versus observed response, exact location and short repro.
The expanded review includes what else was inspected. The card follows the same
finding into correction and independent acceptance.

**Knowledge contribution:** failure patterns and review methods backed by actual
evidence. The library must not smuggle Builder private reasoning into review.

## 7. Fixer components

The central unit is a finding and its disposition. People should be able to follow
why a change addresses a problem, what was verified and what remains unsettled.

**Primary producers:** `correction_scope`, `record_disposition`, singleton task
output/checkpoints, `finish_correction`, finding assessments and shared reports.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| FX01 · Correction brief | Findings to address and current candidate → complete input digest, prior rounds and allowance | Open any required finding | Full canonical correction scope; missing inputs, current, stale. Required findings cannot disappear through filtering. |
| FX02 · Finding disposition ledger | Proposed resolution for every finding → code fix, dispute, verification or declined optional advice with evidence | Inspect a disposition and its linked change | `record_disposition`; missing, proposed, evidence pending, assessed. A submitted disposition is not an accepted resolution. |
| FX03 · Before/after behavior | What was wrong and what changed → exact diff, relevant behavior and limitations | Compare candidate-bound artifacts or native changes | Finding-to-correction links; proposed, checkpointed, acceptance pending. |
| FX04 · Regression evidence pair | Original failure and new check result → same scenario, candidates, fixture and required receipts | Compare the evidence and inspect failed/unknown results | Trusted checks and finding/repro source; old/new comparison must identify both versions. |
| FX05 · Correction checkpoint | Attributable checked correction and affected findings → exact tree/commit and check evidence | Open the checked output | Shared checkpoint renderer with finding links; proposed, verified, failed. It does not resolve review threads. |
| FX06 · Evidence-backed dispute | Why a finding may not apply → original claim, counterevidence and remaining uncertainty | Inspect both arguments or follow independent assessment | Canonical disputed disposition; contested, under assessment, accepted/refused. Popular agreement is not resolution. |
| FX07 · Optional-advice disposition | Optional change taken or declined and why → benefit, scope and evidence | Discuss or open a separate proposal | Optional finding/disposition; decline is not an unresolved mandatory defect. |
| FX08 · Remaining corrections | Required items still unanswered or unverified → per-finding next step and blocker | Open the outstanding item | Complete finding set plus dispositions; counts distinguish proposed fixes from accepted fixes. |
| FX09 · Correction result | Complete correction report submitted → candidate, all dispositions, independent acceptance and thread-resolution effects | Open accepted evidence or the next review stage | Correction finish/consumer; submitted, verifying, accepted/refused, native resolution pending. |
| FX10 · Root-cause chain | Cause → defect → change → regression prevention → evidence and scope | Inspect a link or challenge the causal claim | Evidence-backed correction explanation; hypothesis is labeled until supported. |
| FX11 · Lesson correction | What earlier knowledge was wrong or incomplete → old/new entry revisions and correction evidence | Propose or inspect the library amendment | Knowledge Amend/K05; proposed amendment, supported revision, contested, superseded. |

**Example composition:** “Team membership is now checked before an expense is
returned.” Show the finding, before/after behavior and regression evidence.
“Submitted for independent acceptance” remains visible until the actual consumer
confirms the fix. Other unresolved findings remain accessible beside it.

**Knowledge contribution:** corrected assumptions, root causes and regression
recipes, including amendments to lessons that led earlier work astray.

## 8. Strategist components

Explain why a change to the work plan is necessary and what effect it will have.
Separate diagnosis, proposal and actual adoption so the feed cannot present a
hypothetical repaired graph as current reality.

**Primary producers:** `reassessment_scope`, `record_diagnosis`, `draft_repair`,
`preview_repair`, `finish_reassessment`, inventory/readiness and adoption records.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| ST01 · Reassessment trigger | What completion, stall or delivery failure prompted this assessment → exact cause and affected scope | Open the triggering evidence | Bound completion/stall/delivery-repair subject; pending assessment, current, superseded cause. |
| ST02 · Diagnosis and causal map | Best-supported cause and uncertainty → evidence, competing explanations and failed approaches | Inspect or challenge a specific causal link | Diagnosis artifact; suspected, supported, contradicted, incomplete. No causal certainty inferred from chronology alone. |
| ST03 · Work inventory health | Ready/waiting/protected work and gaps → complete sealed inventory and missing pages | Inspect a category and its exact work | Inventory/readiness source; complete, incomplete, stale. Counts identify scope and observation time. |
| ST04 · Dependency stall view | Cycle, missing producer or unsatisfied join → prerequisite reasons and affected outcomes | Focus the blocking path or inspect a dependency | Deterministic graph/stall evidence; observed stall, ordinary wait, stale graph. |
| ST05 · Repair alternative comparison | Recommended repair and tradeoffs → alternatives, evidence and expected consequences | Compare preservation, scope and readiness effects | Draft repair alternatives and recorded assessment; recommendation is not adoption. |
| ST06 · Before/after work graph | Proposed additions, changes and removals → diff against the sealed graph with dependency reasons | Switch between current/proposed or inspect an edge | Draft repair plus deterministic preview; proposed, valid, invalid, stale. |
| ST07 · Protected-work impact | What stays intact and what active work is affected → completed/manual/running work and proposed treatment | Inspect a protected item and the relevant constraint | Existing protected bindings; preserved, conflict, unresolved. A preview does not stop or reassign work. |
| ST08 · Outcome gap and recovery map | Promised outcomes lacking a viable producer → proposed work and repaired coverage | Follow an outcome through current/proposed work | Accepted outcomes, inventory and repair proposal; gap observed, proposal covers, adoption confirmed. |
| ST09 · No-change or verified-wait result | Why no graph change is needed and the exact next event → evidence and ordinary wait owner | Open the prerequisite or recorded reassessment | Evidence-backed no-change/wait result; legitimate progress outcome, not a failed repair. |
| ST10 · Adoption progress | Which proposed changes have actually applied and what remains → native effects, conflicts and reconciliation | Open confirmed issue changes or a specific failure | Adoption journal; submitted, applying, partial, confirmed, refused/stale, uncertain. |
| ST11 · Recovery lesson | Pattern, successful remedy and when it applies → cause evidence, actual adoption and later result | Open library entry or add contrary evidence | Learned/K03/K04; a proposed repair cannot be labeled a proven successful remedy. |

**Example composition:** “Two tasks are waiting for each other.” Show the cycle,
the proposed prerequisite correction, preserved completed work and the predicted
readiness change. After adoption, show the actual graph and newly ready work as
confirmed events linked to that proposal.

**Knowledge contribution:** recurring dependency problems, failed recovery
approaches and demonstrated repair recipes with limits and outcomes.

## 9. Test Pilot components

Lead with observable product behavior across the integrated application. A green
unit-test count cannot replace accepted-outcome coverage or an actual journey.

**Primary producers:** `verification_scope`, `record_journey_result`, trusted
manifest/fixture/check/artifact records, `finish_app_verification` and shared reports.
The current D16 pre-start manifest execution remains in force. These display
components do not authorize new environments, arbitrary tests or unlimited reruns.

| ID / component | Compact view → expanded view | Useful interaction | Record binding and important states |
| --- | --- | --- | --- |
| TP01 · Delivery under test | Product version and verification goal → exact integrated artifact, source, outcome revision and fixture | Inspect what was tested | Delivery/verification subject; prepared, unavailable, mismatched, current, superseded. |
| TP02 · Accepted-outcome coverage | Verified, failed and unverified outcomes → journey/check evidence per required outcome | Open an uncovered or failed outcome | Complete accepted-outcome map; denominators count required outcomes, not tests or issues. |
| TP03 · User journey walkthrough | A person's actions and observed results → steps, branches, fixture and evidence | Step through observations, inspect a branch or read the text equivalent | Recorded admitted journey; observed, failed, incomplete. Stepping through evidence does not rerun the app. |
| TP04 · Expected versus observed | The behavior mismatch and user impact → exact step, expected requirement and actual evidence | Compare artifacts or inspect the originating requirement | Journey failure/canonical requirement; confirmed mismatch, uncertain observation, stale candidate. |
| TP05 · Screenshot, recording and trace viewer | The useful captured moment → ordered screenshots, recording/trace and accessible description | Expand, scrub a recording or inspect a captured step | Authorized artifact/capture; observed versus illustrative, unavailable, revoked. No autoplay or execution on open. |
| TP06 · Environment and fixture validity | Whether the test conditions match the admitted delivery → fixture identity, preparation and missing prerequisites | Inspect the mismatch or evidence | Trusted preparation/fixture records; valid, missing, uncertain, changed. No secrets or private fixture data. |
| TP07 · Integrated check results | Required manifest outcomes → actual receipts, candidate, failed/skipped checks and completeness | Inspect the required failure and its scope | Trusted manifest execution; not started, running, passed, failed, stopped, uncertain. |
| TP08 · Failure and repair linkage | Failed journey, affected outcome and current response → reproduction, evidence and linked repair result | Follow the canonical repair or discuss the observed behavior | Verification failure plus existing repair admission; reported, assessed, repair proposed, adopted, reverified. |
| TP09 · Regression comparison | How this delivery compares with the prior relevant verification → changed outcomes, scenarios and evidence | Compare matching scenarios and versions | Linked verification history; fixed, regressed, unchanged, not comparable. Different fixtures must not masquerade as improvement. |
| TP10 · Quality observations | Specific accessibility, layout, latency or resilience observation → approved criterion, conditions and evidence | Inspect the observation or relevant recording | Only admitted checks/journeys; measured facts use units and conditions, no invented quality score or unsupported compliance claim. |
| TP11 · Application verification verdict | Verified, failed or incomplete with the important consequence → all required outcomes, receipts and actual settlement | Open final evidence or follow failed outcomes | Verification finish and completion owner; submitted versus confirmed app completion. Deployment is a separate fact. |
| TP12 · Reusable verification recipe | Scenario worth repeating and why → fixtures, steps, expected result, scope and version | Open library recipe or propose an update | Learned/K04; an executable recipe requires the existing execution authority, not just a saved command. |

**Example composition:** “Submitting and approving an expense works; cross-team
access remains unverified.” Show the user journey and outcome coverage, then open
the evidence for the unverified branch. The verdict remains incomplete until
all required current evidence exists.

**Knowledge contribution:** useful verification scenarios, cross-feature failures,
fixture limitations and recipes that improve later testing.

## 10. Knowledge components shared across roles

The feed shows when and why knowledge changed. The library shows the current
entry and its evidence/history. Both use one entry identity. Role-specific lesson
cards above are variants of these components, with different content emphasis.

| ID / component | Compact view → expanded view | Useful interaction | Record and state behavior |
| --- | --- | --- | --- |
| K01 · Discovery | Useful claim and applicability → explanation, example, sources and provenance | Read, reuse or challenge | Learned entry; observed, supported, contested. Discovery is immediately visible to permitted readers without a mandatory approval queue. |
| K02 · Decision or convention | Applicable rule and its rationale → authoritative decision/guidance, alternatives and revision | Inspect authority or propose a change | Library reference to the same decision as S16, or authoritative convention source; authority, validity and application remain separate. Learned does not create a second decision. |
| K03 · Pitfall and failed approach | What to watch for and under which conditions → failure evidence, root cause and alternatives | Open repro or related corrected work | Evidence-linked observation; applicability and version must remain visible. |
| K04 · Recipe or playbook | Reusable steps and intended outcome → prerequisites, fixtures, checks and limitations | Inspect or copy permitted instructions; navigate to an existing authorized execution route | Versioned recipe; recorded, supported, stale. Viewing it does not run it. |
| K05 · Amendment and supersession | What changed in the lesson and why → old/new revisions, evidence and affected references | Compare or propose a correction | Amendment/entry history; proposed, current, superseded. Historical uses retain their original revision. |
| K06 · Contradiction | Two incompatible claims and why the conflict matters → scopes, versions and evidence for both | Add evidence or follow investigation | Conflict record; unresolved, different-context explanation, corrected/superseded. No popularity vote settles truth. |
| K07 · Reuse with outcome | Where a lesson helped or failed → exact entry revision, later work and outcome | Open the application and provide feedback | Explicit usage feedback; cited, used with result, contradicted. Usage alone does not corroborate correctness. |
| K08 · Scope and sharing proposal | Proposed broader audience and reason → source restrictions, safe published version and authorized decision | Inspect or take the actual permitted sharing action | Suggest/library publication owner; proposed, refused, accepted. A “general” label does not remove disclosure constraints. |

The library view should support project/component/kind/applicability filters and
search. Every preview also enforces current access, including snippets, counts,
source labels and cached content. Feed subscription and knowledge permission
remain distinct. No component may reconstruct private source contents from an
otherwise public entry.

<a id="library-admin-components"></a>
### Library Admin components

These components accompany the [maintenance profile](factory-worker-mcp.md#library-admin).
Both knowledge and decision items use them with visible type and authority.
The author header stays outside the card. Group routine edits by maintenance
episode; show a consequential decision replacement separately with its rationale.

| ID / component | Default content and expansion | Interaction and authoritative state |
| --- | --- | --- |
| LA01 · Maintenance scope | Why this collection needs attention, bounded scope and outstanding cases → inventory/completeness and source reports | Inspect the queue; a partial inventory cannot establish that the whole collection is clean. |
| LA02 · Item edited | What changed and why → exact before/after, evidence and attribution | Compare revisions; stale/refused changes remain visible. Editing rationale is distinct from changing the effective decision. |
| LA03 · Consolidation | Source items → resulting item, concise reason → claim/scope mappings and retained sources | Open original revisions and current target; pending/refused/applied, with separate indexing state. Conflicting choices cannot appear silently reconciled. |
| LA04 · Split | Original item → focused items, purpose of each → evidence, authority and scope mappings | Follow original/new records; historical references stay intact. Derived decision items retain their originating authority. |
| LA05 · Removal and restoration | Removed/restored item, reason and replacement if any → history, affected references and current validity | Authorized restore where applicable; removal does not pretend the underlying decision was revoked or the issue completed. |
| LA06 · Replacement | Which knowledge claim or decision scope was superseded and why → old/new evidence, actual decision authority and affected work | Open the operative replacement; a recorded proposal is not a confirmed replacement. |
| LA07 · Relationship and conflict | Duplicate, contradiction, dependency or explanation and why it matters → both scopes/versions and evidence | Inspect or contribute evidence; unresolved conflicts stay explicit and respect each item's visibility. |
| LA08 · Affected work | Which assignments cite an edited/replaced item → historical revisions, inbox notice and incorporation state | Inspect the work's actual response. Notification does not prove an accepted plan changed. |
| LA09 · Maintenance result | Confirmed changes and unresolved cases → operation receipts, partial failures and projection status | Follow remaining work; finishing the episode does not certify the entire library or reapply its mutations. |

### Proposed research issue components

The later [issue work-contract proposal](factory-worker-mcp.md#issue-work-contracts)
also needs research questions/scope, sourced findings/comparisons, accepted
knowledge output and linked prerequisite-created/research-accepted/work-resumed
views. These extend the current catalog if the Researcher workflow is adopted;
they must use its typed records and real non-code completion/readiness states.
Display role plus specialty where useful in the outside-card header, preserve
separate newest-first posts and open full reports/evidence separately. A completed
report and a satisfied parent prerequisite can differ; simultaneous blockers,
conflicting sources, inconclusive findings and partial creation/link failures
remain visible. R42/R43 own the projection and rendering under the task extension.

## 11. Factory and Jev components around the workers

Some critical events have no worker author. Presenting them as a worker's own
announcement would blur the distinction between a report and confirmed progress.
These components complement the production and Library Admin role sets. Factory
and Jev remain system actors with their own attribution.

The [generic `systemone.*` tools](factory-worker-mcp.md#generic-systemone-tools)
return judgments to the calling worker; a call alone does not publish F04 or
create a library decision. A consequential worker post can compose F04/S07/S16
with its role-specific content and reference the invocation and supplied sources.
Keep that post's worker author and label Jev as the judgment source. Display the
actual primitive semantics when useful: yes probability for Noul, candidate
distribution for Choice, described level scale for Score. No invented Noul
confidence, unexplained score percentage or provider-generated rationale.
Routine evaluations stay in inspectable work detail. A controller's actual Jev
decision remains a system event with its own recorded effect/acceptance source.
R42/R43 own this readback and rendering; R49 qualifies consequential reuse.

| ID / component | What it shows | Actions and source of truth |
| --- | --- | --- |
| F01 · Preparation and admission | The concrete prerequisite ready/missing and the assignment affected | Open existing access/environment evidence; preparation and admission owners supply state. |
| F02 · Dispatch and role transition | Which role was actually admitted, started or retired | Open that role's work thread; distinguish selected, requested, running and observed ended. |
| F03 · Why this work is next | Eligible work, selected work and the supplied reasons | Inspect recorded selection; one eligible item needs no fabricated Jev consultation. |
| F04 · Jev assessment | Question, alternatives, supplied criteria, recorded choice and effective outcome | Inspect actual judgment; confidence is optional detail and grants no authority or correctness. |
| F05 · Native issue or PR created | Confirmed native object, its origin and what follows | Open the native object; creation receipts supply the link, not worker prose. |
| F06 · Merge and issue closure | Exact confirmed merge and original-issue closure state | Open native evidence; merge confirmed with closure pending is distinct from delivered. |
| F07 · Dependency released | The verified prerequisite and newly eligible work | Follow downstream work; readiness owner supplies release, not a feed interaction. |
| F08 · Pause, stop and takeover | Requested action, actual scope and observed effect | Use existing authorized controls; pending retirement remains visible and Stop watching stays separate. |
| F09 · Reconciliation and unavailable effect | What outcome is unknown, affected work and current recovery | Inspect the existing operation; a Retry button cannot blindly repeat an uncertain external effect. |
| F10 · Application completion | All required outcomes verified for the exact delivery and remaining separately authorized steps | Open delivery evidence; never imply deployed when only verification completed. |

## 12. Composition, ordering and noise control

### Default compositions for real events

The table describes the available composition across default post, opened thread
and work detail. The main visual and the context/state needed to understand it
appear together in the post; detailed supporting records remain available through
the relevant disclosure.

| Event | Suggested main component | Supporting detail |
| --- | --- | --- |
| Product direction ready | PL03 walkthrough | PL02 people/access, PL04 outcome coverage, only unresolved PL05 questions. |
| Architecture ready | AR04 approach diagram or AR07 task graph | AR06 rationale, AR08 ownership and AR09 checks. |
| Implementation milestone | BU07 verified checkpoint | BU01 relevant lanes, BU05 artifact and BU06 checks. |
| Defect found | IN03 finding | IN04 reproduction and IN05 evidence; coverage is expandable. |
| Correction submitted | FX03 before/after | FX04 regression evidence, FX02 disposition and actual acceptance state. |
| Plan repair proposed | ST06 graph comparison | ST02 cause, ST07 protected work and ST08 coverage impact. |
| Verification result | TP03 journey or TP04 mismatch | TP02 outcome coverage, TP05 evidence and TP11 actual verdict. |
| Knowledge changes | Relevant K component | Origin work, evidence and validity; repeated reads create no feed posts. |

The recommended emphasis follows the current event: a failed check deserves the
foreground even if a pleasant screenshot is available. The worker can supply
bounded explanations and suggested highlights, but server-known failures,
required missing evidence and authoritative state cannot be hidden by presentation.

### Root posts, updates and discussion

Keep the accepted combined newest-first feed. Use stable work threads and
separate root posts for consequential events, linked back to the same canonical
work. Routine progress updates revise the ongoing progress contribution while
retaining history. Replies remain in the thread by default; they do not reorder
the feed on every keystroke or minor correction. A meaningful new outcome can
create a new root event.

Grouping is a view over retained events. It must not destroy individual findings,
turn a failed check into success or erase a changed answer. Surface an explicit
new-updates affordance while preserving reading position and expanded content.
Do not silently reorder the feed by AI importance or invent engagement rankings.

Compact views show one main result and the exceptions that change its meaning
or available actions. Multiple exceptions may use a short grouped summary with
explicit counts; they cannot disappear behind a successful result. Expanded
views contain evidence, branches and history. A focus view can show a large task
graph, long diff or recording while keeping a stable link back to the feed item.
Every important interaction has a keyboard-accessible text/list equivalent.

### Interaction rules

- Inspecting a graph, comparison, artifact or journey is read-only. Dragging a
  node must not silently edit the accepted plan. Keep any future edit mode explicit
  and bound to a proposal/permission contract.
- Answer, comment, propose a correction and inspect evidence are common human
  actions. Approve, merge, retry, rerun, publish and share appear only where the
  owning API and current authority actually permit them.
- Simulation/mockup previews are labeled illustrative and isolated from live
  product state. Playback of recorded evidence never re-executes the scenario.
- A successful comment save does not confirm the requested work happened. Show
  linked request/assessment/authoring/adoption outcomes separately. Apply the
  [comment intake contract](factory-feed-and-ralph-integration.md#subscriptions-comments-and-executable-work):
  submitted human comment → JEV routing → Product Lead LLM issue authoring when
  selected → native creation → linked result. Keep intermediate progress beside
  the comment; a confirmed issue merits its own work post. A saved comment
  survives missing allowance, provider failure and invalid issue output. Retry
  recovers the same request; a reload never calls JEV or creates another issue.
- Unsupported actions are absent or explained, rather than attractive buttons
  that cannot perform their stated action.

## 13. Component data and MCP relationship

The producer supplies typed domain records and safe narrative/artifact references.
The server projects a known component kind and allowed interactions. The frontend
chooses its registered renderer. A worker does not send arbitrary HTML or script,
forge state labels or choose another role's author.

Suggested conceptual component envelope:

```text
stable component identity and registered kind
canonical source reference, source revision and related work references
server-derived author, role and scope
headline/summary and validated typed content for this kind
reported state, authoritative state, freshness and completeness
authorized evidence, artifact and knowledge references
allowed actions with current preconditions and expected revision
presentation grouping and disclosure state independent of workflow state
```

UI detail state such as expansion, selected comparison side and playback position
is local presentation state. Authenticated comments, answers and corrections are
domain mutations. Optimistic text may display “sending”; optimistic rendering
must never claim a graph was adopted, a fix accepted or an application verified.

The proposed MCP interface already supplies most producer families. The catalog
also makes missing producer requirements explicit:

| Component group | Needed producer contract | Existing/proposed owner |
| --- | --- | --- |
| Product experience, people/access and alternatives | Structured experience/assumption/coverage content attached to a revisioned product proposal | Product proposal and R12/R13/R15/R32 consumers; not guaranteed by a generic summary field. |
| Architecture, investigation and plan | Typed inspected evidence, interface relations, alternatives, task/ownership/check graph | Existing D03 owners plus proposed investigation/draft operations. |
| Implementation previews and task lanes | Admitted native task events, safe artifact references and trusted output/checkpoint records | R16–R19/R33 and existing native execution owners. |
| Findings, reproduction and correction comparisons | One finding identity, exact candidate anchors, coverage, disposition and evidence links | R22–R24/R34 and proposed role assessment operations. |
| Repair comparison and confirmed adoption | Sealed before-state, typed proposal, deterministic preview and actual effect journal | R36/R37/R40 with existing graph-adoption owners. |
| Journey observations and outcome proof | Approved journey/fixture scope, exact artifacts, recorded observations and trusted checks | R38–R40; exploratory execution remains a separate design question. |
| Library variants | Entry/revision, applicability, support/governance/validity, conflicts and explicit reuse | Shared knowledge proposal in the MCP design; new schemas must be allocated before implementation. |
| Recorded decisions | Actual decision owner/authority, chosen option, alternatives, scope, sources, validity and application | Shared decision tools and existing authoritative human/controller receipts; one decision identity across work, feed and library. |
| Library Admin maintenance | Typed item revisions, operation receipt, before/after mappings, preserved references and current access | Proposed Library Admin tools using existing domain/control/Store owners, projected by R42/R43. |
| Visible interaction/state | Persisted comments/questions/answers, revisioned action authority and readback | R31/R32/R42/R43 and existing domain permissions. |

Do not add one tool for every table row. Add or refine a tool only when a required
domain action lacks an ingress. A diagram, count, comparison and timeline often
share one underlying record; they are different views of the same fact.

## 14. Shared renderer primitives and reuse

Implement fresh Lit components in the existing Work/Spaces owner. This catalog
contains many semantic variants, not a requirement for a separate frontend
framework, service or one independently implemented widget per row.

Useful shared primitives are: author/context header; typed state indicator;
question/answer form; ordered scenario steps; coverage matrix; graph with list
equivalent; before/after comparison; evidence/reference strip; artifact viewer;
task lane/list; bounded result/check table; provenance/revision history; and
discussion/action footer. Role-specific renderers compose these with their
actual domain semantics.

For example, Product Lead outcome coverage, Inspector review coverage and Test
Pilot delivery coverage can reuse table and disclosure primitives while keeping
different row definitions, evidence requirements and completion rules. Builder
and Fixer reuse checkpoint presentation with different work/finding context.
Architect and Strategist reuse graph primitives with different editing authority
and states. Their views should not be flattened into one generic progress card.

Source candidates inspected for this design:

- [Ralph signal display](../../../ralph4days/src/components/workspace/task-detail/components/SignalDisplay.tsx)
  supplies useful typed question/finding/knowledge/proposal content. Its inspected
  Stuck renderer labels the reason “Blocked Because”; Soda must preserve the
  explicit distinction between known blocker and diagnosis-needed states.
- [Ralph replies](../../../ralph4days/src/components/workspace/task-detail/components/ReplyCard.tsx)
  demonstrate attributed discussion. Soda needs its own revision, access and
  result-binding behavior in Lit; this is behavior reuse, not React integration.
- [Soda work journey views](../../frontend/spaces/sodaspaces-factory-work-journey-view.ts)
  contain task/finding/correction/judgment presentation helpers. The inspected
  file explicitly describes fixture presentation; it does not establish live
  producer-backed acceptance of this catalog.
- [Soda activity](../../frontend/spaces/sodaspaces-factory-activity.ts)
  already distinguishes native issue creation, PR creation and verified merge
  activity. Reuse those canonical events and current display controls.

## 15. States, accessibility and finite presentation

Every relevant component needs loading, empty, missing/incomplete, stale,
unavailable and access-lost behavior as well as its domain states. An empty result
can mean no findings only when the producer says coverage is complete. A blank
library search or delayed feed projection must not be mistaken for an empty
project or proof that nothing happened.

On narrow screens, stack before/after panes, present graphs as expandable lists,
wrap task controls and make artifact viewers open into a focus view. Preserve
the meaningful content before metadata. Use names/icons/text alongside color,
visible keyboard focus, labeled controls, adequate touch targets and accessible
descriptions for visual evidence. Avoid autoplay, fake live indicators, forced
animation and charts with unsupported numerical precision.

Bound root components, child rows, graph nodes, artifact bytes and history pages.
Large results show the actual total and explicit paging/completeness; no silent
truncation. A graph's omitted subgraph cannot look like absent dependencies.
Counts use their real denominator and scope: tasks, findings, outcomes and
checks must not be blended into an invented overall completion percentage.

Current revision, historical revision and live execution are distinct. Historical
views are clearly labeled and cannot offer stale mutation controls. Access loss
clears content, snippets and queued previews rather than retaining private data
in an expanded card. Evidence links are rechecked when opened.

## 16. Complete examples and design coverage

The [generated post examples](factory-feed-and-ralph-integration.md#role-specific-post-examples)
explore medium-sized work posts in dark and light mode, with role-specific
questions, check progress, diagrams and findings given useful visual space.
They combine Soda's existing visual language with Ralph's author/verb and
discussion anatomy. They are design mockups; the state and action contracts in
this catalog remain authoritative.

The following connected example is illustrative, not evidence of a running
Factory or a prescribed product requirement:

1. Product Lead presents an expense-request walkthrough with a permission question.
   Its answer becomes a revision-bound product decision, visible in PL10 and K02.
2. Architect records the service-boundary investigation and an executable plan
   with independent form/service tasks and an integration check.
3. Builder's lane view tracks observed task outputs; the preview is tied to its
   candidate and a checkpoint shows what was actually verified.
4. Inspector records coverage and a cross-team visibility finding. Its repro and
   source location remain attached to the same canonical finding.
5. Fixer submits the behavior change, before/after regression evidence and a
   disposition. Independent acceptance updates the linked finding state.
6. If work stalls, Strategist explains the specific dependency problem, previews
   a repair and later shows actual adoption or a precise failure.
7. Test Pilot walks through submission, approval and rejected cross-team access
   against the integrated delivery. Complete evidence produces the appropriate
   verdict; Factory separately confirms application completion.
8. Root-cause and verification lessons enter the shared library. A later role
   sees the exact applicable revision and can challenge it if the source changes.

Acceptance of the eventual implementation should demonstrate the seven production
roles and Library Admin compositions and failure states using real persisted
records, plus:

- All eight adopted communication actions have usable representations, including
  role-specific Done and distinct Blocked/Stuck/Incomplete behavior.
- Every component has a canonical source and an accountable actor; every enabled
  action reaches its real owner and recovers cleanly after failure.
- Proposal, observed execution, verified evidence and confirmed effect never
  become interchangeable visual states.
- Native finding/task/outcome identities survive grouping, revisions and paging.
- One question-to-answer-to-incorporation path and one learning-to-later-use-to-
  amendment path remain inspectable across the feed and library.
- Decision logging, actual authority, application and supersession remain
  inspectable. Library Admin edit/remove/restore/consolidate/split preserves
  historical references and exposes conflicts, stale operations and indexing delays.
- Read-only inspection does not create work; permissions, account changes and
  revoked evidence are enforced without leaked cached previews.
- Keyboard, narrow-screen, large-content and delayed-update behavior preserve
  the story, reading position and unsent drafts.

R42 owns the authorized projections and R43 the Lit composition, with each domain
producer supplying its required records. Sections 17–20 specify the proposed
exception compositions and their acceptance cases. Producer schemas and actual
consumer/renderer behavior still require implementation and qualification.

<a id="failure-component-routing"></a>
## 17. Choosing components when work does not go to plan

The original inventory described individual failure states. That was partial
coverage: it did not specify which composition appears for a concrete failure,
what wins when several conditions coexist, or how recovery changes the view.
The following is the proposed routing contract, not evidence of implemented
behavior or a claim that every possible failure has been anticipated.
Existing workflow contracts still control the effects: [D14's worker interface](../development/factory-workflow-contracts.md#d14-worker-interface-layers),
[D15's ordered stall classification](../development/factory-workflow-contracts.md#d15--classify-stalls-and-repair-within-finite-allowances)
and [D16's exact delivery verification](../development/factory-workflow-contracts.md#d16--whole-app-verification-and-honest-completion).
This section specifies how those facts appear and how proposed MCP actions
expose them; it does not add execution authority.

### Who chooses what

The worker chooses a **domain action**: record a failed investigation, flag a
problem, ask a question, submit a disposition, report a known blocker, and so on.
It supplies the affected object, evidence and explanation. It does not choose a
free-form error widget or decide whether the workflow is effectively blocked.

The domain owner validates and records the action. Its authorized read projection
selects the **component composition** from the admitted role, record kind, exact
source revision, authoritative lifecycle/effect state, completeness and access.
The frontend renders those registered kinds. Optional worker highlights can
choose among relevant details; they cannot suppress a required exception.
No Jev call is needed to select a renderer or refresh a feed. Jev's existing
judgments remain separately attributed inputs where the workflow requires them.

```text
role action or observed system event
  → canonical record / receipt and current domain state
  → authorized projection: primary + required exceptions + supporting detail
  → registered components in the same work conversation
```

One action can update the task board, check strip, outcome map and library
reference together. The MCP contract therefore needs structured failure and
recovery data, not a separate tool for every component or a generic “ping card”
operation. All proposed mutation/error contracts must expose, when applicable:

- A stable submission/record identity, affected object references and expected
  revision; source identities include the candidate, plan, finding or outcome.
- What was recorded versus refused, the owning reason code, and an existing
  operation/receipt reference when the effect may already have happened.
- Evidence references, observed facts, uncertainty and missing required inputs.
- Affected work, the actual prerequisite owner, release condition and remaining
  permitted machine route. Unknown cause stays unknown; do not invent an owner.
- Current allowed actions and their preconditions. Only the domain owner grants
  retry, replacement, adoption, execution or completion.

Ordinary invalid tool arguments return a precise correction to the caller; they
do not produce a root feed post on every attempt. Persisted validation failures
that prevent progress appear in the relevant readiness/result component. Reading
or validating an object does not itself publish an event.

### Composition and precedence within an affected work item

This order protects the meaning of a card; it does not change scheduler priority,
the newest-first feed order, or D15's evaluation order.

1. **Enforce current visibility.** Remove inaccessible data and actions before
   composing the view. A generic access message must not reveal private objects,
   counts, titles or evidence through the explanation.
2. **Show the authoritative lifecycle and effect boundary.** Stop requested,
   retirement pending and unknown external effects constrain the available
   actions. They do not erase the useful work or its actual failures.
3. **Show whether the evidence applies.** Missing, stale, mismatched or incomplete
   evidence is visible beside any result that depends on it. Historical success
   remains history, not a current success badge.
4. **Lead with the role's concrete exception.** For example, a failed check uses
   BU06, a disputed finding FX06, and a broken journey TP04. Add S05 for a known
   prerequisite, S06 for an unresolved diagnosis, or S02 for an actual question
   only when that additional condition exists.
5. **Keep other work and recovery visible.** Unaffected tasks may progress under
   existing admission. Show preserved milestones and the next permitted action;
   do not turn one failed lane into “the whole factory is blocked.”

For one affected object, lifecycle/effect constraints and evidence validity must
remain visible even when another component is primary. Within equivalent
conditions, use the domain's existing severity/order and stable object identity,
not an AI importance score. Conditions on different objects remain distinct.
Group repeated symptoms only when there is a shared recorded cause; retain the
individual identities, impacts and release conditions.

The backend can revise the ongoing composition when recovery is confirmed.
Keep the original failure and its evidence in history. A new run, candidate or
plan gets its own identity and links to the earlier attempt. A worker's claim
that a blocker is resolved does not clear the controller's unmet prerequisite.

### Common exceptions and the actual outlet

Tool names below are proposed interfaces from the MCP design unless explicitly
identified there as current. **Observed** means the system can produce the event
without a worker call; an absent or dead worker cannot author its own update.

| Condition | Worker outlet or system source | Selected composition | Recovery and boundary |
| --- | --- | --- | --- |
| A required fact needs a consequential decision | `ask`; authorized answer route | S02 plus the affected role component; S14 if a decision is actually required | Show answer owner and affected work. Continue independent work; only authoritative incorporation resolves the dependency. |
| A defect or risk is known, but work can continue | `flag` with role-valid evidence | Relevant role exception plus S07; Inspector findings use IN03 | Do not automatically add S05 or stop the assignment merely because a flag exists. |
| A prerequisite is known and machine resolution is exhausted | `blocked`; observed readiness | S05 plus the affected role component | Name prerequisite, owner and release event. A report is assessed; it cannot override actual readiness. |
| The cause is unresolved after failed approaches | `stuck`; bounded diagnosis owner | S06 plus the role's investigation/failure evidence | Show uncertainty and remaining diagnosis route. Do not claim a dependency cycle without evidence. |
| A routine dependency, capacity or admitted check is still progressing | Observed readiness/execution; `partial` for meaningful progress | Role progress/check view; S05 when a prerequisite explanation is useful | Ordinary waiting is not Stuck or deadlock. Unchanged waits create neither repeated posts nor repeated judgments. |
| Some useful work exists, with more still possible | `partial` | S04 plus exact role progress; existing exceptions stay visible | No terminal transition and no inferred verified count. |
| The assignment must end without its complete result | `finish_incomplete` | S12 plus preserved role evidence and unresolved work | Retirement and settlement still need observation. A nonterminal Blocked/Stuck report is not this action. |
| Budget, provider allowance, deadline or permission prevents further action | Observed control state; a permitted report when available | F01 or S14/S05 at the affected stage; S12 only after an actual incomplete submission | Use the precise existing reason and authorized recovery. No automatic extra budget or generic human approval. |
| Required context is missing, or context-selection Jev is unavailable/low-confidence | Context preparation/hold and judgment records | F01 + F04 + S15, with S14 where attention is required | Serve the contracted deterministic fallback and preserve the hold. No fresh Jev call on a read; no worker-authored question if no worker was admitted. |
| Pause/stop arrives during work or a finish races with stop | Control request and observed lifecycle | F08 plus current role result/work; S11 if a finish was submitted | Display both recorded ordering and settlement. “Stop requested” is not “stopped”; submission cannot override final lifecycle policy. |
| Native process exits, crashes or loses its binding | Native observation and settlement | F02/F09 plus last trustworthy role evidence; F08 for a control-caused stop | Do not invent a last worker message. Freeze uncertain progress until observed reconciliation; a new admission is a new attempt. |
| Tool reply is lost after a possible write | `receipt` and existing operation reconciliation | Existing role component + F09 only while outcome is uncertain | Look up the original identity; exact replay follows existing idempotency. Never duplicate issues, reviews, comments or merges to repair readback. |
| A source changes or an answer is revised while work continues | Revision event; role context/read | S15 + S10 + affected role component/S02 | Identify what became stale. Preserve still-valid evidence; refresh/reassess through the owning route before dependent mutation. |
| A result is submitted but rejected, incomplete or waiting for native effects | Role finish and consumer receipt | Role result + S11; S15 for evidence gaps, F09 for uncertain effects | Name the rejected condition or pending effect. Only observed downstream confirmation advances the display. |
| A message arrives after assignment retirement | Persisted discussion/inbox and admission owner | S09/S02 with incorporation pending, alongside the historical result | Preserve the message. Route any further work through a permitted successor; do not revive the retired capability. |
| An unrecognized condition or unavailable status prevents classification | Authorized structured error/read-model state | S15 + S14 when a permitted action exists; retain last known role view labeled as such | Say what is unknown. No guessed success, cause, retry or permission. Use existing observation/support routes and a safe reference, without raw sensitive payloads. |

## 18. Non-happy-path compositions for each role

Each row names the primary role component first; `+` means required supporting
components when those facts are present. Shared rules in section 17 apply to
every row, including stop, allowance, authorization and uncertain effects. These
are scenario families: they extend the catalog's existing component states
without inventing another server or a widget for each error code.

### Product Lead

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| Consequential ambiguity about who can see or approve something | `ask` bound to the assumption → PL05 + PL02 + S02 | Hold only dependent decisions. Show the real decision owner; a plausible model guess is not an accepted access rule. |
| Two current instructions conflict, or a prior answer is corrected | Product decision revisions → PL10 + PL09 + S02 + S10 | Show the competing sources and affected outcomes. Resolve authority through the existing decision route, not “latest comment wins.” |
| A routine assumption can be chosen within scope | Draft/rationale → PL05 + PL10 | Mark it as an evidenced or in-scope choice and continue. Do not manufacture an approval interruption. |
| A required outcome has no work, measurable acceptance or failure-path coverage | Draft validation → PL04 + PL11 | Show uncovered outcomes. Revise the proposal; an attractive walkthrough cannot make the decomposition complete. |
| Proposed work duplicates existing work or creates a dependency cycle | Draft/graph validation → PL08 + PL04 | Identify overlaps or invalid edges and revise within scope. No duplicate native issue creation as a workaround. |
| Desired scope does not fit supplied constraints | `suggest`/alternatives → PL06 + PL09 + S03; S02 only for a consequential choice | Explain the tradeoff and preserved outcomes. Do not silently remove accepted requirements or promise an unsupported schedule. |
| Intent changes while decomposition is underway | Revision event → PL09 + PL04 + S15 | Show reused versus stale work. Rebind/reassess the proposal before adoption; existing completed work remains recorded. |
| No repository/Project is associated yet | Entry/admission state → PL01 + F01 | Preserve the product review. Use the existing association/Generate route; viewing a proposal does not provision or dispatch work. |
| Only part of the proposed native graph was created | Adoption journal → PL08 + PL11 + F09 | Show confirmed issue links and pending/unknown effects individually. Reconcile the committed portion before further adoption. |
| Proposal is refused or assessment is unavailable | Finish/assessment receipt → PL11 + PL06; F04 for an actual Jev judgment | Show the precise deficiency/hold and valid saved alternatives. Neither a high-quality draft nor a recorded submission is adopted work. |

### Architect

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| Required prerequisite output is missing or merely promised | `issue_context`, readiness, permitted `blocked` → AR10 + AR01 + S05 | Identify the producer and confirmation needed. Continue independent investigation; planned output is not an available input. |
| A required implementation fact cannot be established | `record_investigation`; `ask` or `stuck` when warranted → AR02 + AR10 + S02/S06 | Preserve inspected evidence and the unresolved fact. Do not admit an executable plan by filling the gap with an assumption. |
| An admitted spike disproves the approach or is inconclusive | `record_investigation` → AR03 + AR06 | Compare a supported alternative or investigate within existing authority. Failed/inconclusive experimentation remains visible. |
| Proposed tasks overlap ownership or create a cycle | Draft validation → AR07 + AR08 + AR10 | Highlight the exact conflict and serialize/revise dependencies as appropriate. No “parallel-ready” label for an invalid graph. |
| Interface changes break an existing consumer or boundary | Investigation/`suggest` → AR05 + AR08 + S03 | Show affected callers and transition constraints. An accepted plan alone does not mean those consumers were changed. |
| Required outcomes lack CheckSpecs, fixtures or integration checks | Draft validation → AR09 + AR10 | Identify uncovered behavior and the missing check input. A written test plan is not an executed test. |
| Evidence/source changes during investigation | Source revision → AR02 + AR05/AR07 + S15 | Mark exactly which decisions and graph bindings need refresh. Do not discard valid unrelated observations automatically. |
| Investigation or graph exceeds bounded delivery limits | Validation/completeness → AR02/AR07 + AR10 + S15 | Use permitted paging/references or revise the bounded proposal. Hidden graph nodes cannot look like nonexistent dependencies. |
| Plan handoff is submitted but validation rejects it | `finish_issue_plan` receipt → AR10 + AR11 + S11 | Show missing/invalid fields and preserved draft. No Builder-running card until actual admission/dispatch. |
| Investigation ends without a complete executable plan | `finish_incomplete` → AR02 + AR10 + S12 | Preserve useful findings, unresolved requirements and restart inputs. Do not relabel a provisional design ready. |

### Builder

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| A required check fails | Trusted check receipt; explanation via `partial`/`flag` → BU06 + BU08 | Show exact candidate and failed check; investigate/fix within admission. BU05 can remain as supporting preview, never evidence that all work passed. |
| A check times out, is cancelled, flaky or lacks a final receipt | Check/lifecycle observation → BU06 + S15; BU08 for attempted recovery | Unknown is not pass or confirmed defect. Distinguish execution failure from product failure; retries require the existing execution route. |
| One task waits while other tasks are runnable | Task/readiness observation → BU01 + BU03 + S05 | Limit the wait to the affected task/edge and allow existing independent work. No repository-wide blocked badge. |
| Child work crashes, is unattributable or has uncertain termination | Native task observation → BU01 + BU08 + F09 | Preserve confirmed output, label untrusted/unknown output and reconcile native binding. No checkpoint from an unattributed process. |
| Output submission conflicts with the current tree/checkpoint | `submit_task_output` validation → BU07 + BU04 + S10 | Show conflicting identities and the admitted rebase/rework path. No overwrite or invented verified checkpoint. |
| Handoff is requested before required inputs/checks are confirmed | `request_handoff` receipt → BU03 + BU06 | Show the missing predecessor evidence and keep successor unreleased. A handoff request is not dependency release. |
| Candidate changes after checks or output submission | Candidate observation → BU04 + BU06 + BU07 + S15 | Preserve old receipts as historical; require applicable evidence for the new bytes before acceptance. |
| Implementation proves the accepted plan insufficient | `suggest`, progress/failure evidence → BU09 + BU08 + S03 | Route a revision; continue only work still admitted and valid. Builder cannot edit the accepted task graph through a card. |
| Useful code is preserved but no complete candidate can be delivered | `partial` or terminal `finish_incomplete` → BU07/BU04 + S04 or S12 | Distinguish observed checkpoint from unfinished changes and ongoing from ended work. |
| Finish is recorded but PR publication fails or remains uncertain | `finish_implementation` consumer → BU10 + S11 + F09 | Follow the existing publication operation. Show a PR link only after creation is confirmed; no merged/delivered claim. |

### Inspector

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| No defect found but required areas were not inspected | `record_review_coverage` → IN02 + IN10 + S15 | Verdict remains incomplete. An empty findings collection is not a clear review without complete required coverage. |
| Candidate/base changes during review | Review tuple observation → IN01 + IN02 + S15 | Label the reviewed revision and stale coverage. Refresh through the existing review route; do not transfer clear status to new code. |
| A concrete defect has sufficient current evidence | `flag` with Inspector schema → IN03 + IN04 + IN05 | Create one canonical finding and follow publication/acceptance separately. Finding submission is not a published native thread. |
| A suspicion is plausible but reproduction/evidence is inconclusive | Investigation/coverage → IN04 + IN07 + IN02 | Show uncertainty and unverified scope. Record a qualified report only if its schema permits; do not assert an observed defect that was not observed. |
| Required source, evidence or finding pages are unavailable | Read completeness/coverage → IN02 + IN07 + S15 | Identify missing scope within access limits; no clear verdict or invented empty carried-finding set. |
| A finding location is invalid, moved or outside the reviewed candidate | Finding validation → IN03 + IN05 + S15 | Correct anchors against the exact candidate. Never publish an apparently precise link to unrelated source. |
| A carried finding lacks reassessment or contradicts the proposed clear verdict | `assess_finding`, finish validation → IN06 + IN10 | Assess each canonical finding. Grouping must not conceal an unhandled obligation. |
| Advice is useful but optional | `suggest`/optional review advice → IN08 | Keep it distinct from required findings; declining it does not block a clear result. |
| Checks are green on another candidate, skipped or unreliable | Reliability/coverage assessment → IN07 + IN02 | Name which claims lack applicable proof. No global trust in a green check count. |
| Native review publication is partial or its result is unknown | Review consumer/effect journal → IN10 + IN03 + F09 | Retain finding identities and reconcile the existing operation before retrying; no duplicate threads. |
| Required review is complete and genuinely has no findings | `finish_review` plus actual acceptance/publication → IN10 + IN02 + S11 | Show an explicit clear result and its scope. This is valid work, but it does not independently confirm merge or delivery. |

### Fixer

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| Some required findings are fixed while others remain | `record_disposition`, progress → FX02 + FX08 + FX03 | Show each finding and proposed/accepted counts separately. One successful fix cannot close the correction set. |
| A disposition is missing, duplicated, unknown or bound to an older finding revision | Disposition/result validation → FX02 + FX01 + S15 | Repair the exact set/bindings before finish. No filtered or deduplicated display that makes invalid coverage appear complete. |
| Regression checks still fail or cause another regression | Trusted checks/`flag` → FX04 + FX08 + FX10 | Preserve original finding links and report the new defect at its actual authority. A code change is not proof of resolution. |
| Fixer disputes a finding with counterevidence | Disputed disposition → FX06 + FX02 | Show both claims and pending independent assessment. Fixer cannot self-dismiss a required finding. |
| Optional advice is declined or deferred | Optional disposition → FX07 | Record rationale; do not count it as unresolved mandatory correction. A requested scope change remains a separate proposal. |
| Source moves after a correction checkpoint | Candidate observation → FX05 + FX04 + S15 | Historical checkpoint stays inspectable; acceptance requires evidence for the current candidate. |
| The old failure and new result use different fixtures or scenarios | Evidence validation → FX04 + S15 | Label comparison not established and identify the missing comparable evidence. Do not claim “fixed” from mismatched examples. |
| Checkpoint or correction cannot be completed within remaining allowance | Control state; `finish_incomplete` if admitted → FX08 + FX05 + S14/S12 | Preserve checked changes and unresolved findings; no self-issued extra round. |
| Independent acceptance rejects the proposed fix | Acceptance receipt → FX09 + FX02 + FX04 | Expose the reason and remaining scope; continue only through the permitted correction/review route. |
| Fix accepted, but native thread resolution is pending or unknown | Acceptance/effect records → FX09 + IN03 + F09 | Show acceptance separately from the native thread state. Do not rerun the fix or duplicate the thread-resolution operation. |

### Strategist

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| Inventory is incomplete, hidden, stale or still being observed | Inventory/readiness → ST03 + S15 | Enter the existing observation-attention path. Missing pages are not evidence of no ready work or a deadlock. |
| Work is legitimately waiting on a running prerequisite, capacity or a decision | Readiness/diagnosis → ST09 + ST04 + S05/S02 as applicable | Show owner and next event; unchanged waits buy no repeated diagnosis. A legitimate wait is a supported result. |
| A cycle or missing producer is actually established | Graph evidence/`record_diagnosis` → ST04 + ST02 + ST08 | Scope the diagnosis to the affected dependency component; ordinary waiting elsewhere cannot hide this repairable stall. |
| No-ready state is unexplained after deterministic checks | Bounded diagnosis → ST02 + S06 | Show competing hypotheses and allowance. Do not confidently label a cause before evidence exists. |
| Proposed repair removes or interferes with protected work | `preview_repair` → ST07 + ST06 | Show the exact preservation conflict and refuse invalid adoption. A preview does not cancel manual/running/completed work. |
| Proposed repair creates another cycle, loses an outcome or has no viable producer | Deterministic preview → ST06 + ST08 + ST04 | Show invalid proposal and the uncovered outcome. Revise within the existing grant; no adoption from prose confidence. |
| New ready work or source changes make investigation obsolete before effects begin | Reconciliation/admission observation → ST03 + ST06 + S15; F08 for retirement | Retire the stale investigation before releasing its execution slot; do not apply the obsolete proposal. |
| Adoption has already applied a prefix when a conflict or unknown effect occurs | Adoption journal → ST10 + ST06 + F09 | Reconcile the committed prefix before release or successor work. Do not present the whole proposed graph as current or silently roll it back. |
| A no-change result is submitted while the unexplained stall persists | Result validation/readiness → ST09 + ST02 + S14 | Keep attention visible without fresh evidence of ready/completed work or a valid named wait. “No change” cannot clear a continuing unexplained stall. |
| Repair is declined, judgment unavailable, or bounded repair cycles exhausted | Assessment/allowance → ST05 + ST10 + F04/S14 as applicable | Explain the actual refused/held/exhausted state and existing recovery. The repair's own graph reshuffle is not new evidence for another automatic cycle. |
| Graph adoption succeeds but the promised recovery has not happened yet | Adoption plus later readiness → ST10 + ST08 | Confirm changed graph separately from newly ready work, execution and verified outcome. Publish ST11 as a proven remedy only with supporting later results. |

### Test Pilot

| Situation | Action/source → component composition | What happens next / what must not be claimed |
| --- | --- | --- |
| Trusted required manifest fails before Test Pilot starts | Controller manifest/preparation → F01 + TP07, attributed to Factory | Preserve actual failed checks. Do not invent a Test Pilot run, report or verdict when the CLI never started. |
| Fixture/environment is missing, invalid or changed | Preparation/verification evidence → TP06 + TP01 + S15 | Distinguish environment failure from an observed product defect. Hold dependent verification until the admitted conditions are valid. |
| An admitted journey exhibits wrong behavior | `record_journey_result`/failure report → TP04 + TP03 + TP02 + TP08 | Bind requirement, exact artifact, fixture and observation; route through existing repair admission. A failure report does not itself launch repair. |
| A required branch/outcome was never exercised | Coverage/finish validation → TP02 + TP03 + TP11 | Show unverified scope, even if all executed checks passed. No completion from a smaller denominator. |
| Artifact, source or accepted-outcome revision changes during verification | Delivery identity observation → TP01 + TP02 + S15 | Preserve historical results; re-establish applicable evidence for the new subject through the authorized route. |
| Checks are skipped, timed out, stopped, flaky or lack a trusted result | Manifest/check records → TP07 + TP06 + S15 | Expose each state and uncertain conclusions. No green summary, arbitrary cherry-picked rerun or new environment from the feed. |
| Screenshot/trace cannot load or access is revoked | Artifact authorization/read → TP05 + S07/S15 | Do not discard a still-valid trusted result solely because its optional preview failed. If required proof is missing, mark the affected outcome unverified. |
| A regression comparison uses a different baseline, fixture or scenario | Comparison evidence → TP09 + TP06 | Mark not comparable. Preserve individual observations without claiming improvement or regression from an invalid pair. |
| A limited accessibility/performance observation is valid | Admitted check/journey → TP10 + TP02 | State measured criterion, conditions and limits. One keyboard check is not blanket accessibility certification. |
| Failure recurs after a repair or the repair targeted another delivery | Verification/repair bindings → TP08 + TP09 + TP04 | Link the exact repair and failed outcome; require existing reassessment/admission for further work. Do not mark an unrelated repair successful. |
| Verification cannot finish within remaining allowance | Control state; admitted incomplete finish → TP11 + TP02 + S14/S12 | Preserve exact verified and unverified outcomes. No automatic allowance extension or exploratory execution grant. |
| All required evidence passes, but settlement or deployment has not occurred | Finish/settlement → TP11 + S11; F10 only on actual completion | Distinguish submitted verdict, confirmed application completion and separately authorized deployment. |

## 19. Knowledge, transport and combined failures

### Shared library exceptions

These apply to every role's learning components as well as the library view.
Library validity and entry authority remain separate from whether a feed card
rendered successfully.

| Situation | Selected composition | Required behavior |
| --- | --- | --- |
| New learning has useful evidence but is not independently established | K01/K03/K04 + S13 | Label support and applicability accurately; immediate visibility does not make it policy or proven general truth. |
| Existing learning conflicts with new evidence | K06 + K07 and the originating role evidence | Preserve both scoped claims and their versions. Feedback/assessment may lead to an amendment; do not silently pick the newest claim. |
| A lesson is correct only for an older version or different environment | K05/S13 + S15 | Mark the applicability mismatch and link the current revision if known. Historical usage keeps its original reference. |
| An amendment is proposed but not yet authorized/applied | K05 + S03 | Show current and proposed entry separately. A proposal cannot replace governed guidance in later context. |
| Search returns no matches, is truncated, or its service fails | Library result state + S15 where relevant | Distinguish a complete empty result from incomplete/unavailable retrieval. Do not claim the library has no knowledge or run a new model call from a read. |
| Learning is recorded but its feed/search projection lags | Existing K component or receipt + S15 | Reconcile the projection from the same entry identity. Do not resubmit the learning; do not claim it is searchable before observed indexing. |
| Cross-scope publication is proposed or refused | K08 + S03/S14 | Use the authorized publication owner. Remove restricted snippets/source details from broader views; a general-purpose label grants no access. |
| Source evidence is withdrawn, inaccessible or later disproven | S13 + S15; K06/K05 when a contradiction/correction is recorded | Reassess support without disclosing hidden source contents. Do not erase provenance or pretend revoked evidence still corroborates the claim. |
| Duplicate-candidate feedback suggests two entries overlap | K07 plus authorized entry references | Link for assessment. Similar wording is not authority to merge claims, scopes or histories automatically. |

### Decision and Library Admin exceptions

| Situation | Visible result | Recovery contract |
| --- | --- | --- |
| Worker logs a choice outside its authority or attributes it to an unsupported owner | S16 + S14 | Refuse the authority claim; route an unresolved choice through Ask/Suggest. No fabricated accepted decision. |
| Decision is recorded but its plan/issue change is pending or refused | S16 + S11 | Preserve the choice and show actual application state; no issue-success badge from the log alone. |
| Two current decisions conflict or apply to different versions/scopes | LA07 + S16/K06 | Preserve both and their authority; clarify scope or route substantive resolution to the actual decision owner. |
| Item changes after maintenance preview | LA02/LA03/LA04 + S10 | Refuse stale expected revisions, reread and rebase the proposed change. Preview never reserves the item. |
| Consolidation loses a qualification or would broaden access | LA03 + S14 | Refuse or narrow the operation; preserve original items and sources until a valid operation commits. |
| Splitting a decision would create unsupported new policy | LA04 + S16 | Require mappings to existing authorized choice/scope; new choices follow the decision route. |
| A removed item is referenced by historical or active work | LA05 + LA08 | Preserve historical revision/tombstone links; notify affected work. Still-operative mandatory decisions remain available from their authoritative source. |
| Restore conflicts with a newer replacement or revoked source | LA05 + S10/S14 | Do not reinstate obsolete authority or inaccessible content; reconcile the requested visibility/validity with current evidence. |
| Multi-item operation reply is lost | Relevant LA component + receipt state | Recover the same atomic operation and reference mapping, never repeat the merge/split with new IDs. |
| Mutation commits but feed/index update fails | Relevant LA component + S15 | Show confirmed storage and pending projection separately; retry projection, not the item mutation. |
| Maintenance allowance expires or only some independent operations succeed | LA09 + S12 | Preserve committed operations, list refused/unattempted work and finish incomplete when necessary. |

### Projection and interaction failures

| Situation | Visible result | Recovery contract |
| --- | --- | --- |
| Events arrive twice or out of order | One stable component at the newest applicable source revision | Dedupe by record/event identity and respect source revisions. Arrival time cannot regress a confirmed state to an older pending state. |
| Several source records advance at different times | Relevant component + S15 for an incomplete combination | Do not claim a transactionally consistent success from incompatible revisions. Refresh the owning projections; avoid a synthetic global workflow database. |
| Live subscription disconnects or a page has gaps | Existing content labeled last observed; S15 | Reconnect/repage through bounded reads. No fake live indicator and no workflow mutation to repair a viewer. |
| User submits from a stale card or permission changes mid-click | S10/S14 on the affected interaction | Recheck authorization and expected revision on the server. Preserve safe draft text and explain the refusal; never perform a stale destructive action. |
| A comment/answer save times out | S09/S02 sending/uncertain state with its stable request identity | Recover its receipt before resubmission. A saved answer still needs explicit incorporation; do not send duplicates. |
| Account/project access changes while cards or drafts are open | Authorized new view; scoped access-lost state only where permitted | Remove old content, previews and actions. Keep private drafts isolated to their original authorized scope; never auto-submit them under the new account. |
| Client cannot render a component schema or safe artifact type | S15 with authorized plain-text summary/source link when available | Keep authoritative state and exception visible; no raw payload/HTML fallback and no success default. This is safe failure handling, not a second supported schema system. |
| Large graph, finding set or history is partially loaded | Component with explicit paging/loaded scope + S15 | Do not compute whole-work success from visible rows. Only authorized producer-supplied complete aggregates may summarize the full set. |

### Intersections that must be designed together

1. **Builder: good preview + failed integration check + one waiting lane.** BU06
   leads the affected milestone; BU05 remains supporting evidence, BU01 shows
   unaffected work and BU03/S05 explains the waiting edge. No overall success or
   blanket factory deadlock.
2. **Inspector: defect found + candidate changed + inaccessible attachment.**
   IN03 retains the canonical finding, S15 marks candidate/evidence applicability,
   and IN02 remains incomplete where required proof is missing. Do not publish
   a current clear verdict or expose the attachment from a cached preview.
3. **Fixer: some fixes accepted + one disputed + stop requested.** FX02 keeps the
   per-finding distinctions, FX06 shows the dispute and F08 constrains new work.
   Stop does not turn the dispute into acceptance or erase verified fixes.
4. **Strategist: a capacity wait in one component + a cycle in another.** ST09
   explains the wait; ST04/ST08 expose the separate repairable stall. Existing
   repository execution-slot and repair admission still control actual progress.
5. **Strategist: new ready work appears after adoption effects began.** ST10/F09
   show committed and unknown effects until reconciliation. The pre-effect
   stale-investigation retirement rule cannot discard an already applied prefix.
6. **Test Pilot: all executed checks pass + required outcome untested + old
   screenshot.** TP02/TP11 remain incomplete; TP05/S15 labels the old capture.
   A green TP07 subset cannot imply the full accepted outcome set was verified.
7. **Any role: finish reply lost + process exits + late answer arrives.** S11/F09
   and native observation settle the original submission; S02 shows the saved
   answer as not incorporated. Receipt recovery does not revive the old worker
   or authorize a fresh assignment.
8. **Knowledge: correction recorded + index lag + source access revoked.** K05
   retains the authorized correction state, S15 reports projection lag and access
   filtering removes restricted content. Neither lag nor a cached older entry
   makes the superseded guidance current again.

## 20. Acceptance of the routing design

This is a finite scenario contract with explicit unknown-case behavior, not a
promise that all possible failures have been enumerated. New domain conditions
must map to an existing appropriate composition or extend this owning catalog
and its producer contract together. Do not add untyped prose-only success paths.

R32 and the role producers own truthful typed actions/receipts; R42 owns
authorized state projection; R43 owns rendering and interactions. Their eventual
focused tests must cover the scenario families above and meaningful intersections:

- For each role, exercise a concrete domain failure, incomplete required input,
  stale source, preserved partial work and confirmed recovery. Assert component
  identity/kind, actual state, visible exception and permitted next action.
- Exercise known blocker versus unresolved diagnosis versus ordinary wait;
  nonterminal reporting versus terminal incomplete; required defect versus
  optional advice; and saved answer versus incorporated decision.
- Check deterministic composition from the same records, no paid read/model
  calls, and required exceptions surviving worker-selected highlights.
- Exercise a lost reply and duplicate/out-of-order update without duplicate
  effects, a stop/finish race, and partial native adoption with reconciliation.
- Verify the system-authored pre-Test-Pilot failure and dead-worker case, so the
  UI cannot fabricate role authorship merely to reuse a role-specific component.
- Exercise library contradiction/amendment plus projection delay and access
  loss; demonstrate later context uses the valid authorized revision.
- Check compact and expanded views, bounded paging, keyboard/text alternatives,
  account changes and action refusal. Presentation must not hide failure or
  grant execution, even when the view is incomplete or an artifact cannot load.

Use real persisted producer/read-model records for integration acceptance and
focused browser assertions for composition. This documentation supplies the
design cases; it is not itself evidence that those runtime paths pass.

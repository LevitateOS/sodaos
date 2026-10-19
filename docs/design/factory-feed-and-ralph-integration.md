# Factory feed and Ralph4days integration

Soda OS will absorb Ralph4days's agent experience into its Factory experience:
recognizable role authors, structured communication, inspectable work and useful
workspace interactions. The intended finished Soda factory procedure leads this
integration. Its current implementation does not limit the experience being
designed.

This is the durable design record of the owner's discussion on 18 October 2026.
It owns the feed and Ralph integration direction and the open presentation choices.
The [product contract](../product/overview.md),
[workflow contracts](../development/factory-workflow-contracts.md),
[factory interfaces](../architecture/factory-interfaces.md) and
[trust model](../architecture/trust.md) continue to own execution behavior and
authority. [Spaces UX](spaces-ux.md) owns the surrounding workspace experience.
The procedure below maps those contracts to a proposed presentation; it does not
replace them or create a second implementation task catalog.

The owner subsequently requested a complete role-by-role MCP redesign spanning
Factory progression, this feed and a shared knowledge library. The
[worker interface proposal](factory-worker-mcp.md) owns that full brainstorm:
shared tools, role work operations, interaction lifecycles, knowledge reuse and
end-to-end journeys. Its extra tool names and presentation recommendations are
proposals, not implemented or frozen contracts. The seven communication additions
below are a minimum requirement, not the complete redesigned interface.

<a id="factory-providers-dialog"></a>
## Default Factory Providers dialog

Provider connection is a first-class, self-service interaction in Soda Forge.
Factory exposes a **Providers** action and opens the same dialog when the user
needs to connect a provider to proceed. This is required for the initial manual
testing experience, not deferred settings polish. Users should never need an
agent to start ordinary provider login or retrieve its link and code.

- List each supported account provider, initially Codex and Muse, in its own
  clickable row. Keep the provider name and status together: a hollow circle
  (`○`) for not connected, a green check (`✓`) for connected. Include readable
  status text so color and icons are not the only indication.
- Clicking a row with `○` starts that provider's login attempt programmatically
  through Soda's existing identity enrollment API and broker. Show the returned
  verification link, an **Open sign-in** action and the code with **Copy code**
  in the dialog. The user completes authorization on the provider's own page;
  Soda handles starting and following the attempt.
- Show starting and awaiting-authorization states immediately. Follow the
  attempt automatically and switch the row to a green check only when the
  broker confirms the connection is ready. No manual refresh, pasted callback,
  agent intervention or terminal command is required.
- Preserve the current Factory view and any draft while connecting. Closing
  and reopening the dialog resumes the current pending attempt; repeated
  clicks do not create parallel login attempts. An explicit cancel stops the
  attempt. Expired, denied, cancelled or failed attempts show the actual state
  and an appropriate **Try again** action. Provider unavailability gives a
  concrete explanation, never a false connected check.
- Read saved connection state when opening the dialog and after reload. A
  disconnected or expired account returns to `○` with **Reconnect** through
  the same flow. Provider connection status describes the account connection;
  separate Project preparation or execution allowance problems remain specific
  Factory messages and must not relabel a connected account as disconnected.

Reuse `POST /api/identity/enrollments`,
`GET /api/identity/enrollments/{identityID}`,
`POST /api/identity/enrollments/{identityID}/cancel` and
`GET /api/identity/connections`, the existing `soda-identity` UI/response owners
and the single identity broker. Adapt the presentation and missing joins rather
than introducing another login service. Credentials stay with the broker; the
dialog receives connection state and the device-login link/code.

Implementation and installed acceptance are tracked in
[R44.2](../development/factory-remaining-tasks.md#r44-2). The acceptance journey
starts unconnected, completes real provider authorization, observes the check
automatically and retains it after reload. Also exercise an expired attempt and
closing/reopening a pending dialog. Existing manually connected VM credentials
do not establish that this UX works.

## Agreed direction

- Put the feed **inside Soda OS**, following real Factory work.
- Present **work posts with useful visual detail**, avoiding both literal tweets
  and entire embedded applications. The
  [post presentation contract](factory-feed-components.md#work-post-presentation)
  keeps explanations, role-specific visuals, state and discussion coherent,
  while full workspaces and long evidence remain separately inspectable.
- Consolidate subscribed activity into **one combined feed**, with the newest
  posts at the top. Users subscribe to feeds/projects; project-only views and
  threads support navigation but do not replace this combined experience.
- Let users **comment on feed posts**. Agents can read authorized comments and
  use them to make plans and/or create issues through existing Factory owners.
  Derived work links back to its originating discussion and exposes its result
  or failure. A subscription grants no access and a comment alone is not an
  automatic dispatch or a new approval requirement.
- Make the entire intended factory procedure understandable, including decisions,
  parallel tasks, reviews, corrections, waits, recovery and application verification.
- Use the marketing website's factory film as a source of component and interaction
  ideas. Ground the real feed in recorded work and evidence.
- Use the established seven production personas and the proposed Library Admin
  as recognizable authors. Keep the author separate from the event/message type.
- Extend role-scoped MCP controls so agents can explain their work and maintain
  their feed contributions during an assignment.
- Adopt Ralph's eight communication verbs: **ask, flag, learned, suggest, partial,
  blocked, stuck and done** as actual MCP outlets. The seven nonterminal actions
  each have a shared tool on Soda's existing Factory MCP server. Done submits
  through the role's existing finish tool; Factory confirms completion. The
  [outlet contract](../development/factory-workflow-contracts.md#d14-communication-outlets)
  fixes the names and role access.
- Treat Ralph4days absorption as a product integration spanning identity,
  communication, work inspection, prompts and workspace interactions.
- Absorb Ralph's useful capabilities, communication vocabulary and interaction
  behavior into a fresh Soda interface in Lit and TypeScript. The owner explicitly
  ruled out React integration. Ralph's visual design is a reference to assess;
  component layouts, styling and portraits need their own Soda design decisions.
  Reuse suitable framework-independent code and assets where they fit.
- Establish the complete procedure map before defining the final state machines,
  MCP additions, component catalog and screen layout.
- Continue design discussion before implementation. Specific tool names, the
  exact post/thread model and character assignments remain open choices.

## Complete intended procedure

The main route starts with an outcome and ends with verification of a particular
application version. New requests and accepted scope changes continue the process
while preserving prior outcomes and history. Manually authored issues can enter
the issue workflow under their existing authority; they do not manufacture an
application intent or a whole-app verification subject.

```mermaid
flowchart TD
    A["Describe the desired outcome"] --> B["Explore alternatives and assumptions"]
    B --> C["Person reviews product direction"]
    C -->|Revise| B
    C -->|Accept| D["Associate repository and prepare Project"]
    D --> E["Generate and validate issue dependency graph"]
    E --> F["Assess readiness and select eligible work"]
    F --> G["Investigate source and record implementation plan"]
    G --> H["Execute tasks, verify outputs and checkpoint"]
    H --> I["Publish candidate and pull request"]
    I --> J["Independent review"]
    J -->|Complete clear review| M["Final integration gates"]
    J -->|Findings| K["Address findings and supply correction evidence"]
    K --> L["Evaluate correction evidence"]
    L -->|Fresh review required| J
    L -->|Corrections accepted| M
    M --> N["Confirm native merge and original issue closure"]
    N --> O["Decide whether remaining work needs reassessment"]
    O -->|Reassess| P["Propose, validate and adopt plan changes"]
    O -->|Continue| Q["Recompute readiness and outcome coverage"]
    P --> Q
    Q -->|Accepted outcomes remain| F
    Q -->|All accepted outcomes delivered| R["Verify integrated application"]
    R -->|Failure| S["Plan bounded delivery repair"]
    S --> E
    R -->|Verified| T["Application version complete"]
```

This diagram shows the successful and repair routes. Waiting, invalidation and
attention paths apply throughout. Its arrows are dependencies, not a requirement
to delay checks that can already run. Product exploration and consolidation can
use the existing bounded draft service before Project association; the Product
Lead's generative Project run follows authorized association and preparation.

| Stage | Responsible actor | Visible work and components | Evidence required to advance |
| --- | --- | --- | --- |
| Capture intent | Person | Objective, organizational context, constraints and desired outcomes | Saved request and supplied information |
| Explore and review direction | Person and exploration service; JEV assesses supplied alternatives | Alternatives, material assumptions, questions and accepted or declined choices | Revision-bound product choices; unresolved material decisions return to the person |
| Prepare execution | Factory and authorized people | Repository association, persistent Project, preparation and resource standing | Required access, preparation, provider/account authority, capacity and finite allowances |
| Decompose accepted outcomes | Product Lead; JEV assesses alternatives; Factory validates and applies | Cohesive issue proposals, outcome coverage and dependency graph | Valid scope and provenance, complete coverage, valid DAG and confirmed native issue bindings |
| Select work | Factory determines eligibility; JEV prioritizes alternatives | Ready, waiting and blocked issues; selected issue and supplied selection evidence | Complete eligible inventory and fresh admission conditions |
| Investigate and plan | Architect | Source inspected, prerequisite outputs, findings, approach, immutable plan and task lanes | Current source and prerequisite bindings, resolved mandatory facts, validated executable plan |
| Implement and checkpoint | Builder and admitted workers; trusted executor checks and integrates | Ready/running/completed tasks, ownership, verified outputs, commits and joins | Verified prerequisite outputs, coherent checked checkpoints, required tasks complete on the integrated candidate |
| Publish | Factory using the submitted implementation result | Candidate, branch, PR and links to the issue and attempt | Attributable candidate and confirmed native publication |
| Review and correct | Inspector and Fixer; JEV where the correction route requires judgment | Review coverage, actionable and optional findings, inline locations, dispositions, fixes and regression evidence | Complete clear review, or complete correction evidence accepted through the permitted JEV path; current native policy still applies |
| Integrate and confirm issue completion | Factory; JEV only where the applicable delivery decision requires it | Check results, accepted evidence, merge request, confirmed merge and original issue closure | Same current candidate, current policy, required checks and reviews, confirmed native merge and original closing occurrence |
| Reconsider remaining work | JEV; Strategist when reassessment is selected; Factory adopts | Continue/reassess decision, proposed graph changes, protected work and adoption results | Fresh complete inventory, bounded validated proposal, required assessment and confirmed adoption before dependent dispatch |
| Verify application version | Trusted check executor and Test Pilot | Accepted-outcome coverage, application journeys, fixture/artifact identity, check receipts and failures | Every required outcome verified against the same integrated delivery, with current inputs and completed retirement |
| Continue or complete | Factory and person | Verified version, remaining attention or follow-up work, next change request | Exact verified delivery; failed verification follows bounded delivery repair, while new product scope returns to review |

A complete clear independent review can establish delivery acceptance without an
extra delivery JEV call. Corrected work can receive a fresh independent review or
candidate-bound JEV acceptance of complete correction evidence where policy
permits. Optional feedback, disputes, confidence and allowance limits follow
[D07 and D08](../development/factory-workflow-contracts.md#d07--corrections-and-review-cycles).
The marketing film's illustrated repeated-review route does not eliminate either
valid production route.

After each verified issue completion,
[D13](../development/factory-workflow-contracts.md#d13--reassess-current-issues-after-every-verified-completion)
governs the repository's reassessment fence and complete remaining inventory.
Reassessment preserves delivered, active, manually controlled and other protected
work. An in-scope delivery repair creates appropriate follow-up work while
retaining completed identities. Whole-app completion follows
[D16](../development/factory-workflow-contracts.md#d16--whole-app-verification-and-honest-completion).
Deployment remains a separate authorized action.

## Waiting and recovery belong in the feed

| Situation | What the person should see |
| --- | --- |
| Unanswered product question | Exact question, available options, consequence, responsible person and affected work |
| Unmet prerequisite | Required outcome, producing issue and what releases the dependency |
| Resource wait | Missing capacity, provider/account access, allowance or preparation and the next resolving event |
| Changed source, scope or policy | Which plan or evidence became stale and what must be refreshed |
| Failed task or check | Recorded failure, affected output and permitted next action |
| Genuine planning stall | Diagnosed cause and bounded Strategist repair proposal, distinct from ordinary waiting |
| Pause, stop, cancellation or takeover | Requested action, observed outcome and any unresolved retirement |
| Uncertain operation outcome | Reconciliation of the existing effect before any permitted retry |
| Failed application verification | Failed outcome/journey, exact delivery evidence and bounded follow-up repair |

These entries explain the existing workflow's recovery paths. They do not add
routine approvals for every technical task or turn an unchanged wait into another
model call. Waiting in one dependency component or application intent must not
conceal independently eligible work elsewhere.

## Role authors and visual identities

The seven roles are established in
[D14](../development/factory-workflow-contracts.md#seven-instruction-templates-and-generated-task-messages)
and [the canonical MCP profiles](../../internal/factory/mcp_profile.go).

| Primary author | Responsibility | Existing formal result control |
| --- | --- | --- |
| Product Lead | Propose issues and dependencies covering accepted product outcomes | `factory.finish_product_plan` |
| Architect | Investigate current source and plan the selected issue | `factory.finish_issue_plan` |
| Builder | Implement the task DAG and supply output/checkpoint evidence | `factory.finish_implementation` |
| Inspector | Independently review the exact candidate and carried findings | `factory.finish_review` |
| Fixer | Address the complete finding set and supply correction evidence | `factory.finish_correction` |
| Strategist | Reassess or repair the work plan against the admitted cause | `factory.finish_reassessment` |
| Test Pilot | Verify the integrated delivery against all accepted outcomes | `factory.finish_app_verification` |

Human choices use the actual person's identity. JEV decisions use JEV. Preparation,
dispatch, trusted verification, adoption, merge and other observed system effects
use Factory attribution. A role's result and the Factory's confirmation remain
distinguishable even when displayed together.

The proposed author header emphasizes the role, followed by issue or PR, round or
lane where relevant, and time. Expanded detail identifies the actual run, harness,
model and evidence. A persona can remain recognizable across different runs and
model providers. Persona names do not create additional native execution accounts.

Ralph's disciplines describe expertise such as Frontend, Backend, Data and
Security. Soda's roles describe workflow responsibility. A possible combination
is **Builder · Backend** or **Inspector · Security**. Technical specialties and
their selection are an open design choice; they must not replace the seven roles.

Ralph's character catalog associates portraits, crops, colors, icons, skills,
instructions and conventions. That pattern can give a role the same recognizable
identity in a feed avatar, issue card and larger role view. The actual mapping of
characters to Soda roles remains open. Soda also has an existing
[robot avatar system](avatars.md); it is not an established seven-role portrait
assignment. Human illustrations, robots, new artwork and reuse of existing
portraits have not been selected by this discussion.

## Feed and component proposal

The [role-by-role component catalog](factory-feed-components.md) owns the full
brainstorm of custom feed components: compact/expanded content, interactions,
canonical producers, evidence states and library links for the seven production
roles and Library Admin, including shared recorded decisions.
Its [exception routing](factory-feed-components.md#failure-component-routing)
also maps non-happy paths to concrete role components, simultaneous conditions
and permitted recovery. Server projections select these compositions from typed
work records and authoritative state; workers cannot hide failures by choosing
a successful-looking layout.
The families below are an overview, not the complete inventory. One role action
can populate several components in one post; components do not each require a
separate MCP tool or a separate source of truth.

The primary composition is one combined feed of the user's subscribed activity,
with newest posts first. This accepted direction supersedes the earlier
project-overview/per-project-feed starting proposal. Every post retains project,
repository, issue/assignment and author provenance; project filters and expandable
threads preserve context without fragmenting the primary feed. Readers can open
native changes and evidence and return to the same reading position.

Each useful entry answers: **what happened, who did it, why it matters, and what
follows**. Decision entries show the recorded choice and supplied criteria and
evidence. They must not invent an explanation attributed to JEV.

| Component family | Content |
| --- | --- |
| Direction and review | Business request, accepted outcomes, assumptions and consequential choices |
| Work graph and issue preview | Objective, prerequisites, readiness, required outputs and acceptance |
| Investigation | Inspected references, findings, affected interfaces and unresolved facts |
| Task plan and lanes | Dependencies, ownership, parallel work, verified outputs and join conditions |
| Checkpoints and changes | Coherent commits, tested trees, candidate and native PR links |
| Review and correction | Coverage, findings, native locations, dispositions, fix evidence and rounds |
| Judgment | Question, eligible choices, supplied evidence, raw/effective choice and attention state |
| Checks and integration | Results for the exact candidate, requested operations and confirmed outcomes |
| Reassessment and repair | Before/after plan, reason for change, protected work and adoption outcome |
| Application verification | Accepted-outcome coverage, approved journeys, exact delivery and receipts |
| Question and reply | Structured question, options/recommendation, blocking meaning and recorded answer |
| User discussion and derived work | Persisted user comments/replies, agent-visible context, originating discussion and linked plan/issue creation result or failure |
| Wait and attention | Cause, affected scope, responsible owner and release condition |

Useful reading behavior includes stable expanded entries, filters, access to older
history, explicit missing/truncated history and new updates that preserve the
reader's position. Playback or history browsing is separate from pausing factory
execution. Feed reads do not run agents or advance workflow state. Semantic status
colors describe outcomes; author identity also needs readable names and icons.

Newest-first post ordering and combined subscribed activity are settled. Exact
subscription unit/defaults, reply/update bump behavior, threading depth, live
update behavior and mobile layout still need design. The existing persistent Work/Spaces owner and
terminal lifetimes remain relevant constraints, as defined by [Spaces UX](spaces-ux.md).

### Subscriptions, comments and executable work

R32 owns authenticated communication/comment mutations and their stable identities;
R42 owns authorized combined readback and subscription preferences in the existing
domain/Store. R43 presents them through Work/Spaces. Server-derived human identity
and project access govern comment writes; agent identity comes from its admitted
run. Subscription membership is a filter within already permitted activity, never
a grant. Reads, counts, paging and agent context recheck access, including after
revocation, account changes or a subscription change. Do not copy private posts
into a globally readable stream or grant an agent access to everything a human
can see.

Persist the post/comment ID, author, scope, revision and discussion relationship.
Feed posts sort newest first with a stable unique tie-break and bounded
cursor paging; ordering must not depend on whichever page arrived last. Specify
reply ordering and whether replies/edits bump a root before implementing that
behavior. Preserve a reader's position, expanded threads and unsent drafts as
new posts arrive; a deliberate return-to-latest action can reveal queued updates.
Changing subscriptions or permissions must not leave another account's cached
content visible. Neither a timer nor a notification service is required by this
direction.

**Owner-settled trigger:** submitting a human comment invokes JEV to decide
whether it should become a new issue. If it should, an LLM writes the issue.
Saving a comment, recording a judgment, drafting an issue and confirming native
issue creation are separate observable outcomes. Typing a draft, opening a
thread, polling or reloading never invokes JEV. Agent replies and generated
status messages do not recursively trigger this human-comment intake.

Recommended implementation through the existing owners:

1. R32 saves the authenticated comment, exact revision and a durable assessment
   request together. The comment remains readable if assessment is unavailable.
2. The controller supplies JEV with the comment, its parent post/thread,
   relevant product direction and bounded accessible issue candidates. Use a
   Choice over `new_issue`, `existing_work`, `discussion` and `clarify`.
   `new_issue` means a coherent actionable request without a matching supplied
   issue; `existing_work` links relevant existing work; `discussion` needs no
   issue; `clarify` identifies insufficient information for a work request.
   Candidate retrieval must report missing/truncated context rather than imply
   an exhaustive duplicate search. This applies the generic
   [System One tools contract](factory-worker-mcp.md#generic-systemone-tools);
   [TypeSafe's routing cookbook](https://docs.typesafe.ai/cookbooks/function_calling)
   supports closed-set action selection. These four routes are our product
   recommendation, not provider-defined outputs or an arbitrary confidence cutoff.
3. For `new_issue`, admit bounded **Product Lead LLM** authoring through R12/R13.
   It writes title, objective/body, acceptance criteria and Soda work metadata:
   work type, expertise, expected deliverables, route and relevant dependencies.
   Preserve the exact source-comment link. JEV selects a route; it does not
   generate issue prose. A clarification that needs prose uses an LLM or a
   deterministic prompt, never an invented JEV explanation.
4. Validate the draft and current authority, then use R13's native issue/graph
   operation to create the issue. Recheck relevant issue candidates before
   creation; repeated intake for the same source resolves the existing effect.
   A confirmed issue enters normal readiness/routing. An active Factory can
   pick it up under its existing grants and allowance; a paused Factory keeps
   it queued. Creation grants neither implementation nor deployment authority.
5. R42/R43 show the linked outcome beside the comment and emit a confirmed
   issue-created post when creation actually commits. Keep assessment/drafting
   progress on the same comment result, avoiding a separate top-level post for
   every internal step. `existing_work` links the selected issue and offers its
   attributed input to the normal consumer; it never silently rewrites an
   accepted plan. `discussion` and `clarify` remain visible outcomes too.

Assessment and LLM authoring use separate bounded funding/admission records.
Missing allowance or provider access leaves a saved comment with an actionable
pending/unavailable result. Reuse a recorded judgment and draft on retry; do not
spend again merely to repair readback. A provider timeout can leave execution or
charge unknown: reconcile where supported and show uncertainty rather than
promise provider exactly-once behavior or blindly resubmit.

Bind comment ID/revision and request ID through judgment, authoring and native
creation. A changed/deleted/inaccessible source before application invalidates
that pending application. Explicitly submitting an edited comment creates a new
revision assessment linked to the previous result; it cannot create the same
issue again. If the original issue already exists, retain its provenance and
route the amendment through existing work. Deleting a comment does not silently
delete an issue already created. Concurrent callbacks and lost native replies
recover the original operation. Malformed drafts, unavailable workers and
scope changes remain visible failures with the comment preserved.

Agents receive comments as attributed input through R31's bounded reads. Human
comment submission authorizes this assessment and ordinary permitted issue
creation without another approval on each comment. Existing consequential
product decisions and issue/graph authority remain in force. Comment text is
untrusted content, not a grant or an instruction to bypass those contracts.

Acceptance joins R32/R42/R43/R49: two authorized subscribed projects in one
newest-first feed; equal-time posts and concurrent pagination; persisted comments
after reload/restart; denial across an inaccessible project; one authorized
comment → JEV → LLM-authored native issue path with origin link, no-issue and
existing-issue outcomes, replay and visible failure. Use the
same records for formal agent results and human replies. No separate social task
database, parallel backlog or mock-only browser demonstration.

## Recommended frontend stack

Use **Lit and strict TypeScript**, Soda's existing shared CSS tokens and Bun build,
inside the current Work/Spaces extension. The owner requested a fresh Soda
interface and explicitly instructed that React must not be integrated. Ralph's
capabilities and useful interaction behavior inform this new implementation;
preserving its React component tree or visual design is not a requirement.
Keep the existing Workspace owner and host/resource lifetimes.
The existing [Lit guide](../development/lit.md) and
[TypeScript guide](../development/typescript.md) own implementation conventions.

| Concern | Recommended approach |
| --- | --- |
| Feed and artifact components | Lit templates and custom elements, with elements owning independent state or resource lifetimes |
| Payloads | Typed, runtime-validated records; a known component kind selects the appropriate renderer |
| Appearance | Existing Soda CSS, typography, spacing, semantic statuses and light/dark themes |
| Reading state | Component-owned expansion, filters and drafts; preserve stable entry identity across updates |
| Work state and history | Existing Go APIs and durable record owners; the browser renders reports and confirmed outcomes |
| Build and checks | Existing Bun extension build, strict TypeScript, Lit analysis and focused component/browser checks |
| Ralph UI integration | Implement useful behaviors in fresh Lit components; assess framework-independent code and artwork individually |

Lit's [keyed list rendering](https://lit.dev/docs/templates/lists/) can preserve
entry DOM identity when items move or arrive. Stable keys support preserving open
details and input state; scroll behavior still needs deliberate design. Use
bounded history paging, with any list-windowing dependency justified by actual
volume and reading requirements. Select live update transport against the
existing admitted transport and lifecycle contracts; SSE, WebSocket or polling
has not been chosen by this discussion.

## Source reuse and React migration assessment

The first source assessment covers Ralph's character catalog, signal rendering
and replies, data hooks, UI primitives, workspace state and prompt recipes. It
identifies integration boundaries, not a completed migration or measured effort
estimate. The inspected dependency graph shows why a mechanical React-to-Lit port
would involve substantial manual work. The chosen direction is a fresh Lit
interface shaped by Soda's procedure, using Ralph as a functional reference.

| Existing source | Reuse opportunity | Work required for Soda |
| --- | --- | --- |
| Portrait PNGs, crop coordinates, icons/colors and discipline YAML | Retain useful artwork and metadata; adapt the catalog to chosen Soda identities | Decide role/specialty mapping, asset delivery, provenance and semantic status treatment |
| Typed signal payloads and MCP verbs | Reuse communication vocabulary and shape of questions, findings, knowledge and proposals | Bind to Soda capability/assignment identity and existing workflow evidence; replace completion claims with formal result plus confirmed outcome |
| `SignalDisplay.tsx` | Reference for meaningful question, finding, proposal and progress presentations | Design Soda components in Lit around role/evidence links and canonical payloads |
| `SignalSection.tsx`, `SignalEditor.tsx`, reply components | Reference for posting, replies and saved history | Implement Soda queries, answer authority, history, error handling and component lifecycle in the existing stack |
| TanStack Query hooks and cache helpers | Reference for request/cache behavior; assess any pure helpers separately | Use Soda transport, identity and state ownership without adding Ralph's React hooks |
| Radix/Base UI primitives, forms, drag/drop and resizable layouts | Reference for demonstrated interaction and accessibility requirements | Use suitable existing Soda controls and verify keyboard, focus and overlay behavior |
| Workspace kernel and Zustand store | Reuse tested ideas and assess pure TypeScript functions individually | Map to Soda's persistent owner, navigation and session lifetimes; avoid parallel owners of the same tabs/terminals |
| Rust prompt recipes and persona sections | Retain useful content, context ordering and inspectability patterns | Keep Soda's canonical instructions, phase/subject bindings, tool permissions and Go prompt owner; assess pure reusable code at the destination boundary |
| SQLite task/signal services and standalone execution runtime | Inform semantic mapping and identify genuinely useful isolated code | Soda's native issue records, existing store/coordinator and Rust host remain the owners of those responsibilities |

Concrete evidence for the UI coupling:

- [Ralph's dependency manifest](../../../ralph4days/package.json) includes React,
  TanStack React Query, Zustand, Radix/Base UI, React Hook Form, React drag/drop and
  resizable-panel packages, Tailwind and Tauri APIs.
- [The avatar view](../../../ralph4days/src/components/shared/SignalAvatar.tsx)
  reads discipline data through a React hook. Its
  [cropped image component](../../../ralph4days/src/components/ui/cropped-image.tsx)
  owns loading/cache behavior and calls a Ralph image command.
- [The query hook](../../../ralph4days/src/hooks/api/useInvoke.ts) uses TanStack
  React Query and enables calls in the Tauri environment. The
  [invocation boundary](../../../ralph4days/src/lib/tauri/invoke.ts) calls Tauri
  directly. Keeping React does not itself make these components browser-ready in
  Soda; this boundary needs adaptation.
- [Signal mutations](../../../ralph4days/src/hooks/tasks/useSignalMutations.ts)
  combine React editing/reply state with Ralph commands and task cache updates.
- [Dialog](../../../ralph4days/src/components/ui/dialog.tsx) and
  [avatar](../../../ralph4days/src/components/ui/avatar.tsx) primitives wrap Radix
  React components. Their behavior is not reusable as Lit template syntax alone.

React can technically render into a dedicated DOM container within a Lit-owned
component using [React roots](https://react.dev/reference/react-dom/client/createRoot).
Such a boundary needs explicit DOM ownership, property/event transfer, styling
and mount/unmount handling through the [Lit lifecycle](https://lit.dev/docs/components/lifecycle/).
That interoperability does not convert React components into Lit. The owner has
explicitly excluded this integration from the Factory feed plan.
[Radix's accessibility behavior](https://www.radix-ui.com/primitives/docs/overview/accessibility)
also illustrates why replacing its controls involves focus and keyboard behavior
as well as appearance.

Trace one representative flow through role avatar, typed question, reply, saved
history and refreshed feed to define the new Lit components and Soda appearance.
Include native host navigation, account changes, mobile use and persistent
terminal continuity. Implementation remains deferred during design refinement.

## MCP communication proposal

Soda's existing architecture has **one Factory MCP server implementation with
seven role-scoped profiles**. Common reads are `factory.context`,
`factory.evidence`, `factory.status` and `factory.receipt`. All profiles can report
incomplete work. Builder/Fixer can submit task outputs, and Builder can request
handoff to an approved successor. The existing fourteen-tool catalog is the
implementation baseline. The required design adds seven distinct shared MCP
communication tools; their implementation remains R32 work.

Use the [worker-interface breakdown](../development/factory-workflow-contracts.md#d14-worker-interface-layers)
for current per-role catalog counts and the distinction between working tools,
context, communication and formal results. A single role-specific finish control
is a structured result boundary, not evidence that the role has only one tool or
that its workflow is inadequate. The communication requirements below are
independently adopted additions whose implementation remains with R32.

The [worker-context design](../development/factory-workflow-contracts.md#d14-worker-context)
distinguishes what each of the seven roles needs, the current facts supplied
automatically, and additional authorized history it could use. Required task and
finding information stays available in every usable bundle. Jev proactively
chooses which code-built bundle to foreground; it cannot grant access, omit
required inputs or turn conversation text into authority. An uncertain or
unavailable selection returns the basic snapshot with an explicit hold. The
linked contract owns the role matrix, projections, limits and funding; this feed
design does not create a second context policy or make private discussions
available to workers.

Ralph contributes a useful distinction between author and communication intent.
Its signal verbs include `done`, `partial`, `stuck`, `ask`, `flag`, `learned`,
`suggest` and `blocked`, with structured payloads and corresponding views.

### Adopted verb vocabulary

The owner agreed to carry these eight verbs into the integration blueprint while
retaining Soda's existing MCP system. **Role identifies who is speaking; verb
identifies the communication; workflow state records what actually happened.**
Each action must have an actual MCP outlet. The
[canonical outlet table](../development/factory-workflow-contracts.md#d14-communication-outlets)
selects seven distinct shared tools for ask, flag, learned, suggest, partial,
blocked and stuck. Done uses each role's existing structured finish tool.
Separate callable tools are the required design; a generic communication tool
or feed-only labels do not satisfy it. All seven roles receive the shared
outlets on the existing server. The selection is settled; strict payload schemas
and implementation remain R32 work.

The following fields are the proposed semantic payloads, not a frozen wire schema.
Normal scope, submission identity and expected-input binding come from the
existing MCP admission contract.

| Verb | Proposed content | Feed component | Relationship to Soda's procedure |
| --- | --- | --- | --- |
| **ask** | Specific question; blocking meaning; optional answers and recommendation; consequence and affected work | Question with options, linked work and reply history | Route material product choices through existing review; a reply or reported blocking flag does not independently change authority or bypass readiness |
| **flag** | Problem, severity, category, affected outcome and evidence references | Finding/problem card with evidence and native location where available | Link canonical review findings rather than creating a second finding truth; distinguish informational warnings from actionable defects |
| **learned** | Reusable observation or evidenced convention; applicable scope, rationale and evidence | Knowledge/discovery card linked to its library entry | The [shared knowledge proposal](factory-worker-mcp.md#7-shared-knowledge-as-a-product-capability) gives this action a library consumer; decision choices use the separate Record Decision outlet and retain their actual authority |
| **suggest** | Proposed change, kind, reason, affected issue/outcomes and supporting evidence | Proposal with consequences and disposition | Use the appropriate existing product, reassessment or repair route; a suggestion itself does not create issues, edit dependencies or authorize new scope |
| **partial** | Progress made, remaining work and reason it remains | Progress/remaining-work component | A nonterminal progress report; ending an incomplete assignment uses its existing formal incomplete control and retirement rules |
| **blocked** | Known blocking condition, kind, affected work, relevant prerequisite/question and what would resolve it | Dependency/resource/question wait card | Server assessment determines effective readiness; resolving the reported condition triggers reassessment under the existing gates |
| **stuck** | Why meaningful progress cannot continue, attempted approaches and relevant evidence | Attention/diagnosis card | Distinguish diagnosis needed from a known blocker; existing bounded stall classification decides whether repair, a wait or owner attention is appropriate |
| **done** | Role-specific structured result and evidence references | Result-submitted card, followed by validation/confirmation | Use the existing role-specific finish control; recorded submission, accepted result, observed retirement and verified downstream completion remain distinct |

The expanded design also adds [Record Decision](factory-worker-mcp.md#decision-recording)
to every profile, with a decision feed component and one shared work/library
identity. It adds the owner-requested [Library Admin](factory-worker-mcp.md#library-admin)
maintenance profile alongside the seven production roles, with its own
[item-maintenance components](factory-feed-components.md#library-admin-components).
This extends the proposal; it does not claim these tools or the eighth profile
are present in the current source catalog.

The target communication layer lets all seven roles submit through these MCP
outlets within their admitted subject. The seven new tools are not present
in the current fourteen-tool catalog; their selected interfaces and consumers
remain to be implemented. Done uses the existing role-specific finish control,
whose downstream acceptance is tracked separately. A native Builder child does not gain
the parent's MCP mutation capability merely because it appears as a lane author;
its recorded output remains attributable through the existing parent/worker path.

### Example role and verb combinations

| Author and verb | Readable feed entry | Follow-through |
| --- | --- | --- |
| Architect · **ask** | Which departments may see this data? | Answer is recorded against the relevant product choice and affected plan |
| Inspector · **flag** | A task owner can read another employee's task | Link the finding, exact reviewed candidate and native discussion |
| Fixer · **partial** | Owner access is corrected; inactive-account coverage remains | Show current progress without implying that the complete finding set is resolved |
| Strategist · **suggest** | Split reminder scheduling from delivery retries | Validate and assess through the existing reassessment/adoption path |
| Builder · **done** | Implementation result submitted | `factory.finish_implementation` submits the candidate; Factory records validation and subsequent publication |
| Test Pilot · **done** | Application verification result submitted | `factory.finish_app_verification` carries verified/failed/incomplete outcome and trusted receipts; Factory determines the delivery's effective state |

### Post maintenance and result binding

The eight actions have the MCP outlets selected above. Editing a post's current
headline/summary, arranging supported artifact components and appending history
still need a concrete post model. Earlier example names such as
`factory.update_post`, `factory.add_component`, `factory.append_update` and
`factory.request_input` are not selected API names. Reconcile any maintenance
controls with the adopted verbs and existing result submissions to avoid duplicate
ways to ask the same question or report the same outcome.

An assignment could receive one Factory-created post with its author and scope
already bound. The agent would maintain its presentation and append meaningful
updates; formal finish controls would connect result records to that post. This
one-post-per-assignment model remains a proposal.

Tool payloads should describe typed content and evidence references. The UI owns
rendering components; arbitrary agent-authored HTML or layout is not implied.
Post revisions retain history. Native findings, plans and completion receipts keep
their existing canonical owners rather than being copied into a competing feed
truth. A question's answer follows the appropriate existing product/authority
route before it can change execution.

Authorship derives from the admitted role/run/assignment. A caller cannot select
another author, repository or scope. Communication records do not independently
authorize implementation, accept their own fixes, release dependent work, merge,
change budgets or deploy. Existing capability, lifetime and admission rules apply
to the proposed controls too. Exact schemas, idempotency, volume bounds and
permissions must be designed with the MCP additions.

## State machine design

The procedure needs connected state machines at several levels:

| Level | Owns |
| --- | --- |
| Request/application version | Product choices, accepted outcomes, delivery coverage and app verification |
| Repository coordination | Eligibility, selection, capacity and post-completion reassessment |
| Issue attempt | Investigation, implementation, publication, review, correction and integration |
| Task execution | Readiness, ownership, running work, verified outputs and checkpoints |
| Operation | Requested, executing, confirmed, failed and uncertain effects |
| Feed presentation | Post revisions, expansions, filters, history position and pending visual updates |

For each transition define **trigger → required facts → responsible actor → action
→ recorded outcome → next permitted actions**. The feed presents those outcomes
and role reports while keeping proposed, requested and confirmed states distinct.
An application's completed issues, active review and blocked tasks can coexist;
one project-wide current-step label cannot describe all of them.

The marketing film's playback and popover state machines govern play, pause, seek,
chapter preview and reading interactions. They can inform browsing behavior. Actual
workflow transitions are driven by current evidence and policy, not elapsed scene
duration or a simulated scene sequence.

## What to absorb and where it fits

| Source capability | Integration direction | Existing Soda owner to preserve |
| --- | --- | --- |
| Marketing film artifacts and issue previews | Adapt visual vocabulary for real plans, lanes, reviews, decisions and reassessment | Work/Spaces presentation and native collaboration links |
| Ralph character/discipline catalog | Reuse the identity-and-crop pattern; decide artwork and seven-role mapping | Canonical role profiles, branding and asset provenance |
| Ralph MCP signals and typed displays | Adapt communication semantics into role-authored components | Factory MCP admission and durable records |
| Ralph signal replies | Keep questions and answers beside the relevant work | Existing scope review and authorized action routes |
| Ralph task detail panels | Present complete issue/attempt journeys | Native issue truth and existing plan/review/check/merge records |
| Ralph workspace tabs and terminals | Adapt navigation between activity, plan, changes and output | Persistent Work/Spaces owner and managed terminal lifecycle |
| Ralph prompt recipes and preview | Make supplied role instructions, context and allowed tools inspectable | Soda prompt ownership, fixed policy and credential boundaries |
| Ralph specialty metadata | Consider expertise alongside workflow responsibility | Seven-role procedure and admitted execution profiles |

Ralph4days uses Rust/Tauri with a React frontend and project-local SQLite. Soda is
a mixed-language system: its network coordinator and APIs use Go, privileged/host
operations use Rust, and its browser UI uses Lit/TypeScript. Shared use of Rust
does not make the standalone runtimes or UI components directly interchangeable.
Assess reusable assets and code at their actual boundaries, and adapt components
where the destination differs. Soda keeps one coordinator, its existing data
owners and native forge authority. This discussion does not decide Ralph's
independent product future or introduce a replacement task database or runtime.

## Concept image

The first vertical image illustrates a complete sample onboarding run, with role
authors, an expanded issue thread, task lanes, review/correction evidence,
reassessment and whole-app verification. Its copy, counts, timestamps, colors and
layout are illustrative; the procedure and dependencies above control the design.

[Open the preserved vertical concept](../../.artifacts/design/factory-feed/concept-2026-10-18.png).

The PNG is a generated design output kept under ignored `.artifacts/` according to
repository policy. It is available in this checkout; the tracked design record
remains understandable without that local image.

### Role-specific post examples

The owner identified the older full-run concept above as an approved visual
reference for its restrained styling, proportions and shallow role-specific
content. The current direction changes its presentation to newest-first,
clearly separated independent posts, with no connecting spine or nested stage
chain. The [new vertical feed proposal](../../.artifacts/design/factory-feed/vertical-feed-external-headers-v5-2026-10-19.png)
explores that direction with all seven worker roles plus Factory/Jev events.
The profile picture, author name, project, time and status sit outside each
content card in its own unboxed header. Proximity joins a header to its card;
a larger gap separates consecutive posts. The card contains the message,
role-specific visuals and actions.
Its [imagegen prompt](../../.artifacts/design/factory-feed/vertical-feed-v5-prompt.md)
records the reference and exact requested changes. The generated layout remains
a proposal for review; newest-first order, post separation and metadata outside
the content card are explicit owner requirements.

The preceding role studies propose medium-sized work posts: explanation, a useful
role-specific visual, honest status and a small set of relevant actions. Soda's
typography, square controls, thin rules and restrained red accents remain;
Ralph's `SignalSection`, `SignalDisplay`, `SignalAvatar` and `ReplyCard` inform
author/verb separation and discussion. The visual space and post height adapt to
the work being explained; these examples do not fix a universal density.

- [Work posts in dark mode](../../.artifacts/design/factory-feed/work-posts-dark-v3-2026-10-19.png):
  Builder progress with task/check states and a next step, followed by an
  Architect's explanatory dependency diagram. Failure remains visible beside
  preserved work without embedding the whole task workspace.
- [Work posts in light mode](../../.artifacts/design/factory-feed/work-posts-light-v3-2026-10-19.png):
  Product Lead's proposed experience and access question, followed by an
  Inspector's expected/observed finding. The question has enough context to answer;
  detailed review evidence opens separately.

The earlier oversized single-post images and `interactive-tweets-*-v2` images
remain local artifacts but are superseded as default presentation references.
The owner rejected both extremes. Use the approved older concept's visual
character with the ordering and separation changes described above.

These are illustrative generated mockups, not screenshots of implemented feed
behavior or finalized role portraits. Their fictional content and robot avatars
remain examples. The images are local ignored artifacts; the descriptions above
preserve the design intent without requiring those files. The
[exact prompt set](../../.artifacts/design/factory-feed/work-posts-v3-prompts.md)
records the source references and built-in imagegen inputs.

## Open design choices and next design deliverable

The owner requested a complete task-list restructuring around this direction
and the subsequent MCP, issue/research and knowledge designs. The
[execution map](../development/factory-remaining-tasks.md#task-list) now separates
concrete Rxx.y deliverables while preserving the stable parent IDs and accepted
evidence. Its [visual coverage map](../development/factory-remaining-tasks.md#complete-visual-and-interaction-coverage)
assigns every component family, interaction and exception to an actual producer,
UI task and joined qualification. R49 is development demonstration; R52 proves
the changed installed consumers.

Resolve these choices inside the task that needs them; they do not hold all
implementation behind another design or approval pass:

1. Subscription unit/defaults and the exact project/issue/assignment post/thread
   structure within the adopted combined newest-first feed; reply/edit bump rules.
2. The role character set and whether/how technical specialties appear.
3. Strict payload schemas and bounds for the selected MCP outlets, permitted
   components, post maintenance and existing formal-result bindings. The separate
   shared tools and Done's role-specific finish mapping are already settled.
4. Which facts are linked from their existing owners and which new communication
   history needs durable storage, ordering, paging and revision semantics.
5. Comment-to-agent trigger/routing, revision/deletion handling and transitions
   into existing plan/issue/product-decision owners. Commenting and agents using
   comments to create plans/issues are accepted capabilities, not optional scope.
6. Mobile presentation, live update behavior and history inspection.
7. Concrete Ralph code/assets to reuse or adapt, their destination owners, and
   genuine gaps requiring new work.

The [worker interface proposal](factory-worker-mcp.md) and
[component catalog](factory-feed-components.md) now map the full procedure to
roles, proposed MCP actions, component families, evidence conditions and source
reuse candidates. The next refinement selects compact/expanded compositions and
exact producer schemas, and expands the concrete code/asset reuse inventory.
Define the fresh Lit interface using those role-specific semantics, Soda's
procedure and existing host/lifetime constraints. React integration is excluded;
visual design and portrait selection remain open.
Refine this record through the named R32/R42/R43 execution tasks; the main map
owns their output dependencies and the packets retain technical acceptance. Resolve only the choices
needed for the next slice rather than requiring another whole-design pass.

## Source references

The reviewed source snapshots were Ralph4days
`f4a073b37e4e860dae2021ae99852cf9f906c513` and the marketing website
`384920da55cc12f1f1b01dba41f64f8ddb20334f`. These identify reuse research, not
production behavior acceptance.

- [Ralph discipline content and artwork](../../../ralph4days/crates/catalog-disciplines/README.md)
- [Ralph discipline identity schema](../../../ralph4days/crates/core-contracts/src/disciplines.rs)
- [Ralph task signal tools](../../../ralph4days/crates/ai-prompt-builder/src/mcp/task_signals_server.ts)
- [Ralph signal components](../../../ralph4days/src/components/workspace/task-detail/components/SignalDisplay.tsx)
- [Ralph threaded signal view](../../../ralph4days/src/components/workspace/task-detail/components/SignalSection.tsx)
- [Ralph role avatar](../../../ralph4days/src/components/shared/SignalAvatar.tsx)
- [Ralph prompt recipe](../../../ralph4days/crates/ai-prompt-builder/src/recipes/task_exec.rs)
- [Marketing film components](../../../soda-os-website/src/app/components/organisms/home/factory-film.tsx)
- [Marketing issue preview](../../../soda-os-website/src/app/components/molecules/issue-preview.tsx)
- [Marketing authored scene sequence](../../../soda-os-website/src/app/content/factory-film.ts)
- [Soda MCP profiles and tool permissions](../../internal/factory/mcp_profile.go)
- [Soda workflow and completion contracts](../development/factory-workflow-contracts.md)

# Factory demo journey and UX blueprint

Design brief · 26 October 2026

For the spoken presentation and exact demo actions, use the
[founder presenter guide](../guides/founder-demo.md). For executable work, use the
[full frontend-to-demo checklist](../development/factory-remaining-tasks.md#current-product-ux-tasks)
in the existing Factory execution map.

## Purpose and evidence

**Owner clarification · 26 October:** optimize the actual product UI and UX.
Start new work with an empty request. Northstar is an optional persona and test
problem, not prefilled product state. Do not populate decisions, comments,
business records, app versions or successful outcomes for the presentation.
Prepared scenario runners are stopped. Acceptance concerns the reusable screens,
handoffs and real states below; authored reference applications remain test inputs.

The strongest founder demo is one continuous application journey: describe a
business process, understand the proposed experience, follow the team building
it, use the result, and change it through the same conversation. Today those
steps exist across several surfaces, but their handoffs still require too much
knowledge of repository, planning and application administration mechanics.

This brief defines the UX template and demo sequence. It does not amend product
contracts or qualify an installed demo. The social feed,
AI-authored comments on human or AI posts, visible JEV consultations underneath
the consulting agent's own work post, and JEV deciding when another role should
contribute above a threshold are committed user intent. The implementation now
uses Feed · App · Work and the bounded contribution policy documented below.
The checklist distinguishes source implementation from remaining installed
journey qualification.

The journey is the leading way of using SodaOS and Canlang, including daily use
after the first build. It must connect the existing product's capabilities rather
than become a demo-only wizard. The user's additional requirements are a working
application one click away from Soda Forge, running at its own address, every design decision published as
a post with comments, and knowledge and decisions discoverable by filtering those
same posts. Keep existing native records, role components, tools and controls;
adapt their entry points and handoffs around the journey. This brief supersedes
its earlier proposal for a separate Knowledge destination.

The gap audit below records the baseline before this implementation. Current
local progress and remaining qualification are tracked in the checklist. The
real Northstar reference app now passes in-Forge authentication, workflow and
v1→v2 retention checks; it is not a Factory-generated output. No new installed
Factory journey is claimed. The [19 October UX audit](../development/ux-audit-2026-10-19.md)
contains repaired historical findings and is supporting context, not a current
list of unchanged defects.

## Product capability map

| Area | Source-backed capability or intended contract | Consequence for the demo |
| --- | --- | --- |
| Intent | Description, private department context, saved revisions, exploration, corrections and repository binding exist. The [creation contract](../product/spaces.md#intent-first-creation) starts with the outcome. | Start with Northstar's process; preserve the request through setup and delivery. |
| Roles | [MCP profiles](../../internal/factory/mcp_profile.go) distinguish Product Lead, Architect, Builder, Inspector, Fixer, Strategist, Test Pilot, Researcher and Library Admin. | Introduce a role when its contribution matters; avoid a roster tutorial. |
| Execution | The [factory contract](../product/overview.md#software-factory-workflow) spans readiness, prepared Projects, implementation, publication, independent review/correction, merge and dependent work. | Show business progress backed by actual work; retain technical inspection as detail. |
| JEV | Existing judgments compare supplied approaches, assumptions, work choices and comment routes. [Comment routing](../../internal/web/api/factory_feed_comment.go) handles discussion, clarification, existing work or new issue authoring. The service continuation also evaluates bounded invitations for already-running roles. | Explain the resulting decision in context; inspect the recorded invitation assessment and the role's separate reply. |
| Feed | [Role posts](factory-feed-components.md) support artifacts, findings, questions, decisions and discussion; human comments have durable intake and linked outcomes. | Keep the social feed as the main communication surface. |
| Decisions and knowledge | [Project library](../../frontend/spaces/sodaspaces-factory-library.ts), decision/knowledge post components and revisioned Library Admin controls already exist. | Expose every design decision and reusable lesson as a commentable post; filter the same posts to find decisions or knowledge, with currentness and history retained. |
| Verification | [App completion](../product/overview.md#recovery-and-completion-of-an-app-version) requires integrated outcome coverage and exact evidence; [delivery admission](../../internal/factory/control/delivery_admission.go) implements scoped Test Pilot work. | Distinguish an issue being merged from the application workflow being verified. |
| Private apps | [Applications UI](../../frontend/spaces/sodaspaces-applications.ts), owner setup and native lifecycle/recovery operations exist. [Runtime guidance](../guides/canlang-applications.md) distinguishes preview, private app identity and production state. | Once ready, App and the result post open the actual application at its own address in a new tab, with its exact version, app identity and data context visible in App; preserve the separate authorized release path. |
| Operator | Repository authority, provider sponsorship, preparation, finite execution limits, app artifacts and private access have independent owners. | Preconfigure the demo's operator requirements honestly; show unresolved readiness as an actionable branch. |

## The journey owns the entry points

The main loop is **Describe → Decide → Build → Use → Improve**. A returning
person can go directly to their running app or current work. The loop explains
where they are and what is next; it does not force them to replay onboarding or
watch every Factory stage. Native work created outside this loop still appears
under its actual repository context, without fabricating a new application intent.
“Decide” includes the team's autonomous, recorded choices within the accepted
direction. Making every decision visible does not make every decision a request
for human approval.

| Journey moment | Capabilities that belong here | User-facing entry and return |
| --- | --- | --- |
| Enter or return | Native sign-in, permitted repositories/Projects, private department context, followed feeds and existing applications. | Open the application or the current project conversation. Retain the goal, active version and any request needing attention. |
| Describe and explore | Intent drafts, alternatives, assumptions, Researcher evidence, content and non-code deliverables, Product Lead walkthroughs. | Describe an outcome; review concrete behavior and comment on the proposal. A research/content result can complete its own journey without inventing an application release. |
| Decide | Product, UX, architecture, data, permission and delivery choices; JEV consultations; human corrections and autonomous choices within authority. | Design-decision posts show the options, reason, owner, effective scope and current state. Comments contain consultation and discussion. Open the same post from a filter or affected work. |
| Prepare and build | Repository association, account sponsorship, provider selection, finite limits, persistent Project, preparation, dependencies, parallel tasks and role dispatch. | Return to the retained request when prerequisites permit, with an explicit next action before starting new planning work. Meaningful work posts explain progress; missing preparation or account access opens the responsible existing control and returns to the same request. |
| Inspect, fix and integrate | Architect plans, Builder checkpoints, Inspector findings, Fixer corrections, native PRs/reviews/Actions, merge and dependent-work reassessment. | Open source, evidence, checks or work detail from the relevant post. Return to the same feed position and app context. |
| Resolve an interruption | Questions, dependency/resource waits, failed checks, Strategist diagnosis, bounded repair, cancellation and supported intervention. | A post names the cause, affected work, responsible actor and supported next action. Independent work continues. Preserve draft, work and limits through recovery. |
| Use the app | Versioned preview, Canlang interfaces and business rules, app-local identity, complete employee/manager/Operations actions and integrated verification. | The feed's result post and App open the exact ready version at its own address in a new tab. The app is never embedded in Forge or in a post. |
| Release and operate | App owner/access setup, authorized release/update, private ingress, start/stop, backup/restore and observed availability. | Carry the verified result into the release control without manual artifact assembly. Daily use remains in App; administration opens only for the responsible person and returns to the application. |
| Improve and reuse | Human change comments, useful AI role contributions, revised requirements, retained records, new verified versions, decision/knowledge curation and reuse. | Discuss a change beside the app, follow its work, try the new version and return to use. Find prior reasoning through feed filters and link it into the next decision. |
| Develop or administer directly | Persistent terminals/tmux, Spaces, explicit human Join, SSH/SCP/SFTP, personal Git access, files/tools/services, Project lifecycle, operator accounts, networking/Tailnet, native CI, host diagnostics and appliance updates/recovery. | Preserve existing tools and native settings. Offer contextual entries from Work, environment status or an actual blocker; retain direct expert access. These are available branches, not mandatory screens in the founder demo. |

Use the existing [feature disposition](../product/scope.md#feature-disposition)
to preserve scope and the [complete intended procedure](factory-feed-and-ralph-integration.md#complete-intended-procedure)
for Factory coverage. The journey does not silently enable deferred capabilities.
For each future capability, identify its entry moment, visible post or result,
action surface and return destination. A capability is integrated only when those
connections work; a navigation link alone does not complete its user experience.

## Application-journey gaps

These are source-backed observations and proposed remedies, not live reproductions.

| Gap | Current evidence | Proposed transition |
| --- | --- | --- |
| G1 · Generic walkthrough content | [intent_alternatives.go:8](../../internal/factory/intent_alternatives.go#L8) identifies hand-authored wording; its families at line 49 and mapping at line 134 select decision/intake patterns. [Walkthrough fallback](../../frontend/spaces/sodaspaces-intent-walkthrough.ts#L207) acknowledges unsupported objectives. | Product Lead authors a concrete people/actions/results walkthrough from the request; JEV compares and checks it. Include fulfillment and visibility, not just a generic approve/decline pair. |
| G2 · Review exposes too many internal choices | [intent-view.ts:325](../../frontend/spaces/sodaspaces-intent-view.ts#L325), line 427 and line 450 expose call allowances, planning operation, delivery approach, idea decisions and repository progression together. | One recommended experience, one correction control and only consequential unresolved questions. Put engineering operations and allowance administration in secondary detail. |
| G3 · Repository creation leaves the journey | [repository-picker-view.ts:108](../../frontend/spaces/sodaspaces-repository-picker-view.ts#L108) asks for an owned Forgejo repository; line 134 opens creation in another tab beside manual refresh. | Create the application's underlying repository through the authorized creation action, or return automatically to the retained request with the new repository selected. |
| G4 · Preparation requires manual backtracking | [intent.ts:839](../../frontend/spaces/sodaspaces-intent.ts#L839) tells the person to prepare the Project, reopen the plan and use Plan work before acceptance. [workspace-setup.ts:752](../../frontend/spaces/sodaspaces-workspace-setup.ts#L752) then enters configuration. | Keep one build request visible while preparation, planning and admitted execution advance. Route only missing prerequisites to their actual owner. |
| G5 · Result-to-app association remains manual | [applications.ts:395](../../frontend/spaces/sodaspaces-applications.ts#L395) links an existing app to a repository; line 431 sends empty project views to All applications. [API projection](../../internal/web/api/applications.go#L15) carries the repository link, not an exact Factory delivery association. | Carry the build request, verified candidate, preview and eventual released application together. Offer Try this version from the result that produced it. |
| G6 · First release remains an operator form | [application-setup.ts:103](../../frontend/spaces/sodaspaces-application-setup.ts#L103) asks for admitted artifact/image JSON and a numeric administrator ID; line 82 ends at installed/stopped. | Consume generated artifact references directly; show address and administrator by name, then the app's own owner setup and authorized release action. |
| G7 · Setup controls ignore operator capability | [applications.ts:431](../../frontend/spaces/sodaspaces-applications.ts#L431) mounts setup whenever no repository filter is present. [application_setup.go:16](../../internal/web/api/application_setup.go#L16) correctly restricts it to the operator. | Render supported actions by capability. For a business owner without release authority, show the responsible operator and a precise pending step. |
| G8 · Updating has no complete release journey | [applications.ts:161](../../frontend/spaces/sodaspaces-applications.ts#L161) admits inspect/install/start/stop in its action control. [application_actions.go:142](../../internal/web/api/application_actions.go#L142) supports operator-only replacement and records its result. | Connect comment → changed work → verified candidate → authorized update → observed running version, with existing records still available. |
| G9 · Opening the ready version must be direct | [openApplication](../../frontend/spaces/sodaspaces-applications.ts#L364) opens the application origin in a new tab, and [applicationOriginAllowed](../../internal/web/api/applications.go#L59) requires a hostname separate from Forge. This is the intended model (owner decision, 28 Oct): apps run at their own address and are never embedded in Forge. | From the result post and App, “Open app ↗” opens the exact ready version at its own address without manually finding artifacts or linking an app. |
| G10 · Ordinary app users lack a suitable launch projection | [application records](../../internal/store/applications.go#L74) and [API loading](../../internal/web/api/applications.go#L71) expose administrator/operator views. | Add permitted App launch/read access for ordinary users while keeping lifecycle administration and app-data permission separate. Authenticate the actual Canlang account inside the supported host integration. |

### App readiness must lead directly to use

“Immediately usable” means that a ready result opens a functioning application
at its own address in its own browser tab, one click from the result. It is not
a screenshot or a promise that a build finishes instantly. The app is never
embedded in Soda Forge (owner decision, 28 Oct). The person can submit, approve and deliver a request, leave
for its discussion, return and find the saved record. Once released, the same
workspace supports everyday company use, with preview/sample data clearly
distinguished from the running company's records.

The separate application runtime, origin and app-local identity are the
product boundary, not an obstacle. Forge links to the app; it does not frame it.
Canlang's `frame-ancestors 'none'` stays as an isolation control. Design and
verify the first-use/access path at the app's own address. Account setup and unavailable-app
states must retain the same application context and offer the real next action.

The verified preview should become usable without requiring company release
first. Company availability follows the existing release authority. Routine
app use after access is established must not require a new Factory run, a
developer terminal, deployment JSON or replaying the creation wizard.

Preserve the repairs already present: [intent polling and local draft recovery](../../frontend/spaces/sodaspaces-intent.ts#L274),
[Open project feed after creation](../../frontend/spaces/sodaspaces-project-journey-view.ts#L143),
and [automatic app inventory/status inspection](../../frontend/spaces/sodaspaces-applications.ts#L115).
The old claims that these controls are wholly absent are no longer accurate.

## Feed and agent conversation gaps

| Observation | Evidence | Proposed behavior |
| --- | --- | --- |
| Agents can already reply, but proactive selection is a separate missing handoff. | [mcp_profile.go:286](../../internal/factory/mcp_profile.go#L286) grants inbox/discussion/ask/answer/reply to all profiles; Reply is line 290. Existing [comment routing](../../internal/web/api/factory_feed_comment.go) selects a work disposition, not a set of roles to invite. | Add a bounded contribution decision on a relevant published human or AI post. Reuse role reply capability rather than inventing a second comment system. |
| Thread projection loses structured agent identity/content. | [CommentDetail and parser](../../frontend/spaces/sodaspaces-factory-feed-response.ts#L1344) retain body, author name, time and pipeline; [reply rendering](../../frontend/spaces/sodaspaces-factory-feed.ts#L1204) uses initials and plain text. | Carry validated role identity and supported structured contribution content into replies; preserve human authorship separately. |
| Replies can also enter the root feed. | [factory_feed.go:44](../../internal/store/factory_feed.go#L44) includes comments without excluding replies; [renderList](../../frontend/spaces/sodaspaces-factory-feed.ts#L1087) renders the returned entries independently. | Keep routine replies within their stable thread. A consequential new outcome can have a distinct linked root post; avoid duplicating the same contribution. |
| An open thread does not receive the same background refresh as root posts. | [Refresh path](../../frontend/spaces/sodaspaces-factory-feed.ts#L401) updates outcomes and post bodies; [thread UI](../../frontend/spaces/sodaspaces-factory-feed.ts#L1200) exposes manual Refresh replies. | Refresh open discussions without moving the reader or replacing their draft. Show new replies when the person is reading older content. |
| Generic JEV receipts do not retain the complete request or consultation thread binding. | [System One adapter](../../internal/web/api/factory_mcp_systemone.go#L108) and [judgment identity](../../internal/factory/judgment.go#L70) retain hashed input identity and returned results. | Capture permitted exact state/questions/options, work-post identity and actual call lifecycle at dispatch, then expose them as consultation comments. A renderer cannot reconstruct the original question from its hash. |
| Knowledge and decision components exist, but the proposed journey must unify their discovery and conversation. | [Library entry](../../frontend/spaces/sodaspaces-factory-feed.ts#L979) replaces the feed view; its toolbar entrance at line 984 is an icon button. The [component catalog](factory-feed-components.md#implementation-update--19-october-2026) describes decision/knowledge posts and revisioned library operations. | Filter the same posts and threads for decisions or knowledge. Automatically publish every recorded design decision and link it to its source and affected work; verify complete producer coverage rather than assuming component availability proves it. |

### Decisions and knowledge are conversational posts

Use one post-and-thread experience. In the UX, a decision is knowledge about a
choice: what was selected, why, by whom, and where it applies. A finding or lesson
may be knowledge without being a decision. **Decisions** is therefore a narrower
filter; **Knowledge** may include both. Keep the existing typed records and their
authority semantics even where their presentation and discovery are shared.

Every recorded design decision needs an identifiable, commentable post, including
autonomous engineering choices and proposed or rejected alternatives. Publish it
from the owning decision/plan operation; do not depend on an agent remembering to
write an optional recap. Existing relevant posts can provide that presentation
when they bind the actual decision. A decision reached in comments has a linked
decision post or updates its existing decision post; the original discussion
stays accessible. A status update must not create another copy of the same choice.

Reuse the producers already present: [factory.learned and factory.record_decision](../../internal/store/factory_library.go#L307)
create a communication post and library item together, and [library maintenance](../../internal/store/factory_library.go#L450)
publishes its recorded changes. The remaining coverage audit is decisions arising
through human product choices, accepted plans and controller/JEV judgments.
Current [feed subscription types](../../internal/factory/feed_subscription.go#L35)
distinguish records/posts; the required semantic decision/knowledge filters are
an explicit frontend and projection gap, not a replacement storage system.

Show the question, considered options, selected or proposed choice, reason,
decision owner, affected outcome/version and evidence. Distinguish proposed,
adopted, rejected and superseded decisions from implementation or verification
state. Comments can challenge or refine a choice; a comment alone does not change
its effective authority. JEV's actual consultation and the agent's resulting
decision use the thread pattern below.

All posts, Needs me, Decisions, Knowledge and Evidence are proposed useful views
of the same feed. Project, role, topic and current/historical state can narrow
them where needed. Preserve newest-first ordering, stable post/thread identity,
comment drafts, reading position and links across filters. Explicitly label a
filter showing only current knowledge so superseded decisions remain findable.
Filtering never invokes JEV or republishes a post.

Retain the revisioned library as the existing source for maintained knowledge,
including its source-post link, relations, revision history and Library Admin
edit/consolidate/split/supersede operations. Open these through the relevant post
and authorized detail controls. A changed lesson points back to its sources;
reusing it links the new work to the effective revision. Do not introduce a
second editable copy merely to create the feed presentation.
Keep its existing search/current-content view available through these contextual
controls. A derived item can share its source post with other items, so preserve
item identity and revision separately from source-post identity. When a comment
concerns an older revision, show that revision and a route to current content;
record whether its outcome added evidence, changed a revision or created new
work. Preserve the existing richer role components within the shared post shell.

### Useful role participation

The implemented `single-role-discussion-v1` policy considers at most four
already-admitted, running role profiles. It supplies the post, the candidate's
actual work scope and the existing discussion to three independently worded,
counterbalanced JEV Choice calls per candidate. All three must choose a useful
contribution with probability at least **0.70** and confidence at least **0.40**.
The strongest minimum probability wins, with at most one invited role per source
revision. Relevance alone is insufficient. JEV selects; that role authors its own
reply. The typed assessments are recorded separately from the reply and any
subsequent adoption or work.

An eight-case, single-source evaluation of the production request shape admitted
all three useful cases (privilege bypass, lost retained data and a missing exact
boundary check) and none of five courtesy, duplicate, out-of-scope, ambiguous or
already-planned contributions. The weakest useful case reached exactly
0.70/0.40. This small evaluation supports the initial setting; it does not prove
universal calibration. Keep real discussion outcomes available for adjustment.

An invitation grants one reply to the exact source, within an existing active
assignment; it grants no new implementation authority or worker execution.
Source revision, discussion freshness and the exact active capability are
checked again before admission. Duplicate service wakes, reads and reloads
cannot dispatch the same assessment twice. Consultation/system receipts and
worker replies do not recursively trigger invitations. Edited source revisions
can be evaluated again. Timed-out assessments remain visibly failed or expired;
they do not silently retry or fabricate a role contribution.

A useful reply says what the role noticed, why it matters, and what should happen
next. Within accepted scope, the existing decision owner can incorporate it
through the admitted work route. A new consequential product choice goes to the
human. Discussion, adopted requirements and execution remain visibly distinct;
routine role comments do not each require an approval ceremony.

### Visible TypeSafe consultations

When a Factory agent consults JEV, publish that consultation as a comment on
the same agent's relevant work post. This is part of the visible work conversation.
Use the following thread pattern:

1. **Builder's work post:** the task, current approach or consequential question.
2. **Builder asks JEV:** the actual submitted question, with expandable supplied
   context, alternatives and evaluation criteria. Label a readable summary as a
   summary; preserve the original input separately, excluding protected fields
   with explicit redaction markers.
3. **JEV result:** the actual typed choice, score or probability returned by
   TypeSafe, attributed as a recorded tool result. Do not invent explanatory
   prose on JEV's behalf. Show a governing threshold only if it actually exists.
4. **Builder's decision:** what the agent will do with the result, why, and the
   affected work. This explanation belongs to the Builder. A proposed decision,
   an implemented change and verified behavior remain distinct states.

Record the work post, assignment, source revision, call identity and timestamps.
Create the comment on submission and reflect waiting, returned, failed or
unavailable status from the actual call. Opening the thread never repeats a
consultation. Group the three independently worded calls required by the
repository's consultation practice into one expandable consultation, retaining
each question/result pair. Surface disagreement; never replace individual
results with an invented averaged verdict.

This visibility rule is separate from JEV deciding which other roles to invite.
Recording an actual consultation does not itself require a contribution score,
and its recorded request/result comments do not recursively trigger more role
invitations. A consequential new decision or finding can become an eligible
contribution event under the bounded policy above.

For the demo, open one relevant consultation from a build checkpoint, inspect
what was asked and returned, then show the agent's resulting decision. Only use
an actual consultation from that work when demonstrating live behavior; label
an illustrative thread or a consultation from a different session accordingly.

## Proposed shared shell and screen template

Use the existing Fountain/native Soda Forge host and repository context.
Proposed stable destinations are **Feed · App · Work**. Feed holds conversations,
design decisions, knowledge and evidence, with filters over the same posts. App
is the actual usable preview or running company application. Work explains
outcomes, active items and blockers and opens existing native tools. Settings and
operator diagnostics remain available in their existing ownership context.
This adapts the existing extension and native surfaces; it does not require a
new standalone frontend, competing issue store or replacement runtime.

Keep the application name, actual state and one next useful action visible.
Open detail or a thread while preserving feed position. Use the established
[newest-first independent post composition](factory-feed-components.md#work-post-presentation):
author outside the bordered card, meaningful visual content, restrained actions,
and discussion/evidence access. No default stage-chain rail or hidden ranking.

Use this reusable brief before drawing each screen:

| Field | Required design answer |
| --- | --- |
| Actor and goal | Who is here, and what outcome are they trying to achieve? |
| Entry and input | What brought them here; which request, version and draft persist? |
| Main content | What must they understand before acting; what remains optional detail? |
| Action and output | One primary action, its authority, and the actual result it produces. |
| System continuation | What proceeds automatically and what event changes the visible state? |
| Success evidence | Which observed result justifies the success message and next destination? |
| Waiting/error branch | Cause, affected scope, responsible owner, preserved work and supported recovery. |
| Return behavior | Reload, Back, account change, narrow layout and keyboard focus behavior. |

Use honest states: draft; needs your answer; preparing; waiting for account or
capacity; building; checking; changes needed; verified preview; release pending;
running version. Show the cause of a wait rather than a fabricated percentage.
Illustration, saved comment, linked issue, merged source, verified application and
released version are different results. Unknown/stale/unavailable states retain
useful context and never inherit a green success badge.

## Readiness and the operator branch

Before the demo, prepare the intended installation, repository access, model
connection/sponsorship, execution allowance, Project preparation, app origin and
client trust. Show these as actual readiness; do not imply fresh self-service
setup has been demonstrated. Department context is private metadata, not a
shared membership or permission boundary.

If readiness fails, retain the build request and name the missing prerequisite:
Reconnect account, Prepare environment, or Await release administrator. Link to
the owning control with the relevant context. Authorized routine progression
should resume through the existing recorded operation; the founder should not
have to navigate backward and reconstruct the build. Deployment remains a
separate explicit authority from Factory merge. App-local users are distinct
from Forgejo/Soda identities.

## Proposed nine-beat founder demo

The beats and fictional examples below illustrate possibilities, not a script
to seed or replay. The active checklist requires an empty request, genuine work,
real app use and an actual change; prepared scenario results must not substitute
for any of those steps.

Suggested presentation length: 8–10 minutes. These are narrative allocations,
not a promise that planning/building completes within those times. When a live
execution runs longer, show the actual wait with its cause; never substitute an
earlier result or recording for a live journey step — no recorded or earlier
result may stand in for the live journey, and never present a prepared result
as the output of a live action that did not produce it.

| Beat | Approximate stage time | Interaction and visible proof |
| --- | --- | --- |
| 1 · Need | 0:00–0:45 | Maya describes equipment requests scattered across a paid work tool, email and a spreadsheet. Submit the desired complete workflow. |
| 2 · Concrete walkthrough | 0:45–1:45 | Product Lead shows Alex requests → Jordan approves/declines → Maya fulfills → Alex sees delivery. Confirm the experience through concrete examples and corrections. |
| 3 · Build feed | 1:45–2:30 | Start the admitted build. Architect explains the approach; Builder shows an actual checkpoint. Open a relevant JEV consultation beneath its author's work post: actual question → recorded result → agent's decision. The app header states what is underway. |
| 4 · Meaningful question | 2:30–3:15 | If genuinely unresolved, Product Lead asks whether everyone may see colleagues' requests. Maya chooses own requests plus assigned approvers/Operations. Show the decision post and its comments, find the same post through Decisions/Knowledge filters, then return to the continuing work. |
| 5 · Review and fix | 3:15–4:15 | Open an actual Inspector finding and Fixer correction with before/after evidence. If review found nothing, show its truthful clear result instead of manufacturing a defect. |
| 6 · Try the workflow | 4:15–5:30 | Open the versioned app at its own address directly from its result. Alex submits €420, Jordan approves and Maya marks delivered. Alex sees the final state. Return to the feed and back to the same saved request. Test Pilot's result identifies the exercised outcomes. |
| 7 · Release and access | 5:30–6:15 | Show the explicit release/access handoff, actual application address and app account context. Open the running company version at its own address from App after confirmed availability. |
| 8 · Change by comment | 6:15–7:30 | Maya comments: “Above €1,000 also needs Finance approval before fulfillment.” JEV may invite Architect to explain the added step and Inspector to identify the boundary cases. Show the request linked to admitted work. |
| 9 · Verify and release change | 7:30–9:30 | Show €1,000 following the original path and €1,001 waiting for Robin. Operations cannot fulfill before Finance approval. Release the verified version and reopen the earlier €420 request with its history retained. |

The question and finding beats are conditional on real evidence. A rehearsed,
clearly labeled scenario can illustrate those states; it must not invent live
agent uncertainty or failed checks. The demo's persuasive proof is the complete
usable workflow and a traceable second change, not the volume of agent activity.

## Fictional fixtures and acceptance

Northstar is a fictional 30-person professional-services company. Maya is its
cofounder still running Operations; Alex requests equipment; Jordan manages
Alex; Robin handles Finance. All names, records, purchases and company facts in
the demo are fictional. Use separate app accounts when showing role permissions.

| Fixture | Expected observed behavior |
| --- | --- |
| €420 monitor request in v1 company storage | Jordan approves, Maya fulfills, Alex sees delivered; reopen this same record and history after the update. A preview record is insufficient. |
| Pre-change pending €1,500 company request | Retain its original approval route after v2. This proves that the new Finance rule does not apply retroactively. |
| Exactly €1,000 request | Manager approval remains sufficient after the policy change. |
| €1,001 request | After manager approval, wait for Robin; block fulfillment until Finance approves. |
| Declined request | Show the reason and the permitted next action; do not treat decline as delivery. |
| Unrelated employee | Cannot view another employee's private request under the selected visibility rule. |

Use a real generic approval-stage/request-state model in the baseline so the
Finance change can alter policy/logic without requiring new persisted model
declarations. This is a proposed application design, not evidence it exists.
[Retained-data updates](../guides/canlang-applications.md#assemble-and-provision-the-application)
require matching storage fingerprints; schema migrations remain unsupported.
If the actual change requires a schema migration, report that limitation instead
of resetting the app or claiming records were preserved without checking them.

## Implementation and completion evidence

The [implementation checklist](../development/factory-remaining-tasks.md#founder-demo-checklist)
owns FJ01–FJ39, their dependencies, existing source owners, parallel lanes and
observable completion conditions. It expands the earlier six broad slices into
executable tasks without creating a second backlog. Begin with the actual target,
minimal identity/state relationships and in-Forge app/authentication proof; the
Northstar application contract is one optional acceptance scenario, never
required for an arbitrary user-entered problem. Journey/feed work and existing Factory repairs
can proceed in parallel, converging on the real app and first policy update.

The [presenter guide](../guides/founder-demo.md) owns what the user says and clicks,
account roles, exact request text, records, time jumps and rehearsal. Live app use
may be combined with explicitly identified earlier real build history. A prepared
candidate must remain linked to its actual original request/comment, not be
presented as the output of a newly submitted similar request. Historical
context cannot replace any required connected-journey step.

A component's presence does not complete its journey. Verify relevant boundaries
against the actual demo installation, preserve successful evidence and investigate
concrete failures. A mockup, fixture update or source inspection does not resolve
the incomplete Factory qualification chain.

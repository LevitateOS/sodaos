# Factory experience and System One planning

## Current disposition — 18 October 2026

This is a historical direction/baseline, not an implementation queue. Later accepted intent, Generate/planning and UI source work supersedes “no implementation” descriptions below. New role/verb feed scope and remaining native/installed proof live in the [canonical remaining outcomes](factory-remaining-tasks.md#consolidated-remaining-outcomes--execution-stopped), chiefly R12/R13/R32/R42/R43/R49. Execution is stopped; accepted components are not reopened.

Soda should let a person describe what they want built, review understandable
product ideas and material assumptions, and have the Factory plan and carry out
the work. Routine technical decomposition must not create a succession of human
approval screens. System One judgments should help choose among sensible options
throughout the process, while code determines eligibility, enforces authority and
accounts for effects and spending.

The later [feed design](../design/factory-feed-and-ralph-integration.md) additionally
adopts combined subscribed activity with newest posts first and persisted user
comments that agents can use for linked plans/issues. That direction supersedes
any per-project-only presentation assumption here; existing scope, grants and
consequential product decisions still govern effects.

This records the owner's direction from the October 11, 2026 phone call.
Implementation is pending. It supplements the
[Factory implementation plan](factory-implementation-plan.md) and
[ideal file tree plan](ideal-filetree-plan/ideal-filetree-plan.md); it does not
declare either complete or silently change the current owning product contracts.
Reconcile those contracts when the new behavior is specified and implemented.

## Current behavior and requested changes

The reviewed source is Soda
`219a84844c706f7790b38253b55e131bf04e5a76` and Fountain
`b640df8088f747d113ff835c716c57c4a9505b69`.

| Area | Current implementation | Requested direction |
| --- | --- | --- |
| New Project | An owned-repository picker followed by explicit Project creation, Join and terminal actions | Start with a mostly empty full-screen description page |
| Product decisions | Exact native issue inputs and deliberate maintainer acceptance; no generated-plan review experience | Brainstorm, consolidate and consult Jev, then present understandable ideas and assumptions for acceptance, rejection or correction |
| Generated issues | No implemented intent-to-authorized-issue-batch workflow | Translate the chosen product direction into native issues and dependencies without requiring individual approval of routine implementation tasks |
| Readiness | Authoritative native facts, accepted requirements, current grants, preparation and finite funding | Keep eligibility deterministic and independently enforced |
| Next work | Oldest ready issue, within current concurrency limits | When several issues are eligible, ask Jev to choose among all eligible issue options using relevant current state |
| Planning blocked work | No implemented separate upfront planning workflow for the generated issue graph | Allow bounded planning on stable facts before unblock, then refresh against actual prerequisite outputs |
| Lifecycle customization | Policy settings and fixed orchestration; native webhooks provide hints | Explore selected configurable actions on issue creation, PR creation and verified merge; no generic hooks editor exists yet |

The current native URL and duplicate-bar repair is implemented. Broader discovery,
creation, layout and workflow changes below remain work to do.

## The person starts with an intent

The opening screen should be mostly empty, with a centered textarea headed
**What do you want to build?** Keep buttons and dropdowns basic and secondary to
the description. Repository selection should not be the opening task.

A department selector may provide organizational context, for example HR. A
**New department** action should open a simple dialog for its name and what it
does. Department storage, visibility and administration are not yet specified;
this label cannot itself grant repository, Project or account authority.

The description starts product exploration. The flow still needs an explicit,
understandable way to associate or create a native repository and obtain the
required Project and funding authority. Describing an idea must not silently
enroll accounts, acquire root privileges or authorize unlimited spending.

## Explore and consolidate before generating the work

The requested sequence is:

1. Capture the person's objective and available constraints.
2. Brainstorm plausible approaches and ideas.
3. Consolidate the ideas into a coherent proposed product direction.
4. Consult Jev about defined design alternatives using the relevant context.
5. Present material ideas and assumptions in an understandable HTML review.
6. Incorporate the person's choices and generate the native issue dependency graph.
7. Plan and execute eligible work under the existing independent grants and limits.

Jev supplies typed judgments over provided state. It does not independently
research source, generate the entire plan or confer execution authority. The
planning owner must supply evidence, frame useful alternatives and construct the
resulting plan. Keep these duties distinct rather than claiming the classifier
performed brainstorming or verification it did not perform.

## Human review is about the intended product

The owner wants approval where it matters, presented so a nontechnical person
can understand what they are deciding. The review should offer clear ways to
accept, decline or revise ideas and assumptions, explain the consequence of a
choice, and incorporate corrections into the plan.

Do not ask the person to approve every generated issue, every technical plan or
ordinary equivalent implementation choices. A generated list of twenty tasks
must not automatically mean twenty approvals or an extra screen merely accepting
the task breakdown. Once product direction is accepted, routine work should
proceed within that direction and the configured limits.

Further interruption is appropriate for an unresolved material product question,
new scope, additional spending, new access or a consequential effect outside the
person's existing request. Approval must describe that concrete choice, not ask
the person to understand internal Factory stages or extension terminology.

The current acceptance contract does **not** automatically authorize AI-created
issues. Implementation therefore needs an explicit contract for carrying accepted
intent into generated work. Bind child work to the accepted objective and choices,
record its origin and revisions, and prevent decomposition from expanding scope
or budgets. Exact native requirements and dependency changes still need proper
attribution and invalidation. This is required design work, not permission to
pretend current generated issues already have human acceptance.

An interactive local
[review prototype](/home/vince/Projects/work/soda-project-review/index.html)
illustrates intent entry, department selection, understandable ideas, one material
question, changes and a resulting work list. Its HR example is illustrative:
it makes no AI calls and creates no repository, issue, Project, account or Factory
work. Only script syntax has been checked; browser behavior is not yet verified.

## Native issues form a dependency graph

Generate native issues with clear objectives, acceptance outcomes, accepted-intent
links and explicit prerequisite outcomes. Work may have sequential portions and
independent branches. Do not impose a single chain merely because the output is
a list, or mark every issue blocked by default. Root work can be eligible while
dependent work waits.

Use the native dependency graph as the relationship authority. Code must determine
whether required outcomes are satisfied from current evidence; a model's opinion
cannot close an issue, invent a merge, waive a prerequisite or resolve a cycle.

Planning may discover that an issue should be split, combined with another or
depend on additional work. Jev can judge those alternatives. Any resulting
native changes must go through their authorized owner, preserve attribution and
accepted scope, and refresh affected revisions and dependencies. The present
conditional-operation API does not automatically supply every new issue-editing
or dependency-writing capability this may require.

## Plan before unblock where the facts permit it

The owner's question was whether per-issue brainstorming, consolidation,
decomposition and implementation planning should happen before or after unblock.
Three fresh, equivalently reworded Jev consultations all recommended bounded
early planning followed by validation against actual prerequisite results.

Separate **ready to plan** from **ready to execute**. A blocked issue may have
enough accepted, stable information to explore approaches or identify missing
prerequisites. Interfaces, source revisions or decisions that depend on unfinished
work must remain provisional. Finalize or refresh those parts when the required
outputs exist. Coding and native effects remain blocked until their own gates pass.

Planning itself consumes resources. Give it explicit authority, finite funding,
recorded inputs and a durable outcome. Do not repeatedly spend model calls on
unchanged blocked work. Reuse a plan only where its relevant facts and question
still match, and invalidate affected portions when those facts change.

## Choose among eligible work with Jev

The owner requested deterministic readiness followed by System One prioritization
when more than one issue could be worked on next. Code first constructs the
complete currently eligible candidate set. Jev receives each eligible issue's
stable identity as a Choice option and the state needed to compare it with the
others. This is a priority judgment, not a readiness judgment.

Relevant state includes the accepted objective and revision, each issue's objective
and acceptance outcomes, dependency evidence and outputs, approved source and
preparation, current capacity/account/budget constraints, known impact and risks,
available effort evidence and queue age. Unknown facts remain unknown rather than
being filled with invented estimates. The exact priority rubric, uncertainty
handling, anti-starvation rule and behavior when no candidate can be selected
remain to be specified.

Bind the answer to its candidate and input snapshot, then recheck current
authorization, requirements, dependencies, funding and capacity before dispatch.
A stale answer cannot authorize a launch. Retain the existing concurrency and
single-execution boundaries; adding semantic priority is not permission for
concurrent work inside one repository.

The [Choice API](https://docs.typesafe.ai/api) accepts at most 255 alternatives.
If a candidate set exceeds the supported limit, define a complete staged selection
procedure rather than silently omitting eligible issues. Keep priority selection
bounded and attributable, including call cost and repeated selection behavior.

## More System One judgments throughout the process

The owner explicitly wants more System One thinking throughout the procedure,
not just one consultation at the beginning.

| Judgment opportunity | Needed state | Boundary retained in code |
| --- | --- | --- |
| Compare product designs | User intent, constraints, researched alternatives and consequences | Accepted scope, authority and budget |
| Identify consequential assumptions | Proposed behavior, evidence and unresolved user intent | Whether an additional decision is required and how acceptance is recorded |
| Consolidate or split work | Issue objectives, overlap, acceptance outcomes and native dependency evidence | Authorized native mutations and revision/acceptance validity |
| Decide whether early planning is useful | Stable inputs, unknown dependency outputs, existing plan and cost | Planning admission, spending and invalidation |
| Choose next eligible work | Complete ready set, goal, dependencies, risk/impact evidence and resource constraints | Deterministic eligibility and fresh dispatch admission |
| Assess whether new facts warrant replanning | Previous input snapshot and current relevant changes | Exact source versions, preserved outcomes and durable progression |

These are judgment opportunities to specify and implement, not existing product
features. Use Choice, Noul or Score according to the answer's meaning. Define
alternatives and uncertainty handling before integrating a judgment. Evaluate
realistic cases; typed output and high confidence do not establish truth.

## Retain the complete Factory default

The retrospective advice favors a complete useful default with bounded
customization. Mandatory authorization, accepted requirements, finite funding,
independent review/correction, exact head/base checks, verified native merge,
confirmed original-issue closure and dependent satisfaction cannot become
removable hooks.

Issue-created, PR-created and verified-merge are candidate events for optional
actions. Concrete desired effects, actor, recipient, scope, budget, failure,
retry and preservation behavior must be established before a generic editor is
implemented. Native webhook subscriptions are notifications, not this feature.

Keep Fountain responsible for generic native authority and evidence; Soda for
product judgments, planning and progression; fixed host operations for native
execution; and the broker for provider custody. Keep persistent human Projects,
dirty work, terminals, tools, services and volumes. Preserve one durable serving
owner and the existing language and defining-owner choices.

## Fountain owns the native page shell

The owner's subsequent October 11 decision requires Workspace to render directly
inside Fountain's normal native page layout. Fountain owns the shared navigation,
page chrome, account controls and repository context. Soda supplies Workspace
content and content-specific controls; it must not recreate application-level
navigation or supply a parallel page shell.

The native URL and duplicate-bar fixes do not complete this requirement. The
earlier composition template embedded native content in an iframe. Workspace
must render directly in the shared native content region,
retaining authoritative handlers, account/repository checks, session revalidation
and exactly one Workspace owner. Preserve terminal renderer/socket lifetime
through the existing owned mounting boundaries. Authentication, installation and
error pages must continue to use their appropriate native layouts without
recursive composition.

Acceptance requires shared-template rendering evidence and browser checks for
global and repository Workspace, native navigation, direct entry, reload,
back/forward, query/fragment, account controls, repository context and terminal
preservation. Visual similarity alone does not establish shared ownership.
The earlier source and native/Chromium checks retain their revision-bound scope
in the [implementation plan](factory-implementation-plan.md#fountain-native-shell-correction).
The current responsibility correction remains incomplete; native feature
lifecycle, reusable host contracts, consumer cutover and installed qualification
follow the [57-task correction list](native-frontend-correction-tasks.md).
The original FX09 discovery, menu and empty-aside evidence retains its scope.

## Implementation work to prepare

These are reviewable planning outcomes, not authorization to dispatch agents,
change live services, call providers for product work or publish a release.
Dependencies apply to the required output, not an entire lane.

| Task | Outcome | Required inputs |
| --- | --- | --- |
| FX01 Intent and creation contract | Define intent, repository association, department context, Project creation and funding semantics | Owner's opening-screen requirements and existing native ownership rules |
| FX02 Product review experience | Plain-language accept/decline/change controls, consequences, material questions and revised plan | FX01 and exact acceptance/invalidation requirements |
| FX03 Accepted intent to native issues | Define attributable bounded generated work, native creation/dependency capabilities and material-change handling | FX01–FX02; Fountain capability inventory; product/trust/acceptance owner reconciliation |
| FX04 Planning lifecycle | Separate planning eligibility from coding readiness; persist bounded plans, inputs, provisional facts and invalidation | FX03 boundary; Store/coordinator/broker funding contract |
| FX05 System One judgment contracts | Specify state, rubrics, alternatives, uncertainty and evaluation for the judgments above | Exact owning inputs for each judgment; may proceed in separate bounded packets |
| FX06 Ready issue prioritization | Complete eligible Choice set, finite consultation, stale-state refusal and fairness | FX05 prioritization contract and current readiness/dispatch boundaries |
| FX07 Planning decomposition changes | Authorized split/consolidation and dependency updates that preserve objective scope and invalidate affected plans | FX03–FX05 and actual native writer capability |
| FX08 Lifecycle action scope | Identify demanded optional actions and define actor/scope/funding/retry/outcome contracts | Concrete user examples and FX03/FX05 boundaries; no generic hooks editor assumed |
| FX09 Workspace discovery and layout | Visible repository Workspace, exact contextual Project selection and reclaimed full-page aside space; direct Workspace rendering in Fountain's shared native layout without an iframe wrapper or Soda application shell | Current native route/single-owner contracts, shell ownership requirement above and screenshot findings |
| FX10 Integrated source and native proof | Demonstrate intent → product choices → issues → planning → priority → coding/review/checks/merge/closure/dependents | Implemented boundary outputs; actual Coordinator.Start owner; matching source/runtime receipts |

Shared SDK, schema mirrors, composition roots, generated contracts and tests
require one physical writer. Carry changes through existing owners rather than
creating a parallel planning daemon or duplicated acceptance ledger. Update
product guides, source selectors and the living ideal-tree plan with implemented
and merged changes; this new planning document does not refresh historical source
body reviews or qualify future behavior.

## Consultation receipts and remaining qualification

The retrospective has three fresh reworded main requests covering sixteen
principal decisions, plus three narrower followups investigating disagreement
about starting a hooks catalog before concrete action requirements exist.
Requests, responses, wording checks and bounded decisions are retained under
`.artifacts/retrospective-jev-20261011/`. The three planning-timing and priority
contract consultations are under `.artifacts/jev-issue-planning-20261011/`.
Agreement is advice over supplied facts, not source inspection or native proof.

Thirty archived request/response pairs across ten historical consultation families
were inspected. They contain structured responses; twenty-seven also have saved
HTTP 200 metadata. The operating-rules family lacks matching metadata and a
wording-audit file. Current Bash inheritance failure did not show that historical
consultations never worked. The Bash setup was repaired and checked separately.

Existing development and selected matching installed evidence remains valid only
at its recorded scope. Normal managed Muse Meta device-account approval remains
the external V03/R04 gate. TestOnly-seeded real provider runs do not close that
gate; protected production signing, public distribution and broader qualification
remain unestablished. New planning, priority and UX behavior needs its own
affected verification, not reuse of old receipts as if it already existed.

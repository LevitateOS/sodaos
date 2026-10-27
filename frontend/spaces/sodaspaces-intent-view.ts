import {html} from 'lit';
import type {
  Department,
  Exploration,
  IntentGenerationStatus,
  IntentPlanStatus,
  IntentRevision,
  PlanningAllowance,
  ProductIdea,
  SavedRequest,
} from './sodaspaces-intent-response.js';
import {explorationReview} from './sodaspaces-intent-walkthrough.js';

export interface IntentView {
  objective: string;
  saveNote: string;
  exploration: Exploration | undefined;
  explorationStale: boolean;
  changeOpen: boolean;
  changeText: string;
  generation: IntentGenerationStatus | undefined;
  generationError: string;
  generationNotice: string;
  unsavedCount: number;
  unsavedParts: string;
  savedRequests: SavedRequest[];
  savedLoaded: boolean;
  savedError: string;
  savedMore: boolean;
  department: string;
  departments: Department[];
  revision: IntentRevision | undefined;
  busy: boolean;
  blocked: boolean;
  dirty: boolean;
  error: string;
  departmentDialog: boolean;
  departmentName: string;
  departmentDescription: string;
  maxIssues: number;
  planningCalls: number;
  allowance: PlanningAllowance | undefined;
  raiseCalls: number;
  fundingNote: string;
  allowanceError: string;
  retryAllowance(): void;
  planOperation: string;
  plan: IntentPlanStatus | undefined;
  planNotice: string;
  focusFindings(): void;
  setPlanOperation(value: string): void;
  planWork(): void;
  refreshPlanStatus(): void;
  prepareProject(): void;
  set(
    field: 'objective' | 'department' | 'departmentName' | 'departmentDescription' | 'changeText',
    value: string
  ): void;
  setBound(field: 'maxIssues' | 'planningCalls', value: number): void;
  setRaise(value: number): void;
  raiseAllowance(): void;
  decide(id: string, decision: ProductIdea['decision'], correction?: string): void;
  toggleChange(open: boolean): void;
  correctProposal(text: string): void;
  refreshGeneration(): void;
  chooseApproach(approach: 'complete_first' | 'incremental'): void;
  describe(): void;
  saveDraft(): void;
  saveAndLeave(): void;
  loadSaved(): void;
  moreSaved(): void;
  resumeRequest(intent: string): void;
  revise(): void;
  decline(): void;
  continue(): void;
  dialog(open: boolean): void;
  dialogKey(event: KeyboardEvent): void;
  createDepartment(): void;
}
function input(event: Event): string {
  const node = event.target;
  return node instanceof HTMLInputElement || node instanceof HTMLTextAreaElement || node instanceof HTMLSelectElement
    ? node.value
    : '';
}
function bounds(view: IntentView) {
  const locked = view.busy || view.blocked || view.plan?.state === 'pending';
  const issues = plural(view.maxIssues, 'work item', 'work items');
  const steps = plural(view.planningCalls, 'AI planning step', 'AI planning steps');
  return html`<details class="bounds">
    <summary>More options · up to ${issues}, ${steps}</summary>
    <label
      >Maximum work items<input
        type="number"
        min="1"
        max="64"
        .value=${String(view.maxIssues)}
        ?disabled=${locked}
        @input=${(event: Event) => view.setBound('maxIssues', Number(input(event)))}
    /></label>
    <label
      >Planning calls<input
        type="number"
        min="1"
        max="100"
        .value=${String(view.planningCalls)}
        ?disabled=${locked}
        @input=${(event: Event) => view.setBound('planningCalls', Number(input(event)))}
    /></label>
    <p>Planning uses only these steps. Building is approved and budgeted separately.</p>
  </details>`;
}
function entryActions(view: IntentView) {
  const locked = view.busy || view.blocked || !view.objective.trim();
  return html`<div class="actions">
    <button ?disabled=${locked} @click=${view.saveDraft}>Save draft</button
    ><button ?disabled=${locked} @click=${view.saveAndLeave}>Save and leave</button
    ><button class="primary" ?disabled=${locked} @click=${view.describe}>Explore ideas</button>
  </div>`;
}
function terminalNote(view: IntentView) {
  if (view.revision?.state === 'declined')
    return html`<p>This direction was declined. Revise the outcome and explore again for a new draft.</p>`;
  if (view.revision?.state === 'rejected')
    return html`<p>This direction was rejected. Revise the outcome and explore again for a new draft.</p>`;
  if (view.revision?.state === 'abandoned')
    return html`<p>This draft was abandoned. Revise the outcome and explore again to restart it.</p>`;
  return '';
}
function requestState(state: SavedRequest['state']): string {
  if (state === 'accepted') return 'Accepted';
  if (state === 'declined') return 'Declined';
  if (state === 'rejected') return 'Rejected';
  if (state === 'abandoned') return 'Abandoned';
  return 'Draft';
}
function savedRequestItem(view: IntentView, item: SavedRequest) {
  const work = item.task_count ? ` · ${plural(item.task_count, 'work item', 'work items')}` : '';
  return html`<li>
    <p class="saved-objective" title=${item.objective}>${item.objective}</p>
    <p class="saved-state">${requestState(item.state)}${work}</p>
    <button
      aria-label=${'Resume ' + item.objective}
      ?disabled=${view.busy || view.blocked}
      @click=${() => view.resumeRequest(item.intent)}
    >
      Resume
    </button>
  </li>`;
}
function savedRequests(view: IntentView) {
  if (!view.savedLoaded && !view.savedError)
    return html`<section aria-label="Saved requests"><p>Loading saved requests…</p></section>`;
  if (view.savedError)
    return html`<section aria-label="Saved requests">
      <p>${view.savedError}</p>
      <button @click=${view.loadSaved}>Retry</button>
    </section>`;
  if (!view.savedRequests.length)
    return html`<section aria-label="Saved requests">
      <h3>Saved requests</h3>
      <p>No saved requests yet.</p>
    </section>`;
  return html`<section aria-label="Saved requests">
    <details class="saved">
      <summary>Resume a saved request (${view.savedRequests.length}${view.savedMore ? '+' : ''})</summary>
      <ul>
        ${view.savedRequests.map((item) => savedRequestItem(view, item))}
      </ul>
      ${view.savedMore ? html`<button @click=${view.moreSaved}>Show more</button>` : ''}
    </details>
  </section>`;
}
export function describeIntent(view: IntentView) {
  return html`<section class="intent-entry">
    <h2 tabindex="-1">What do you want to build?</h2>
    <p>Describe who will use it, what they need to do, and what a successful result looks like.</p>
    ${terminalNote(view)}
    <label
      ><span class="sr-only">Describe what you want to build</span
      ><textarea
        rows="5"
        maxlength="16000"
        placeholder="Describe the outcome you want…"
        .value=${view.objective}
        ?disabled=${view.busy || view.blocked}
        @input=${(event: Event) => view.set('objective', input(event))}
      ></textarea>
    </label>
    ${entryActions(view)}
    ${
      !view.objective.trim() && !view.busy && !view.blocked
        ? html`<p>Enter a description above to enable saving and exploring ideas.</p>`
        : ''
    }
    <p>${view.saveNote}</p>
    <p>Save a draft to return to it later. Explore ideas saves your description and uses the planning limits below.</p>
    <div class="context">
      <label
        >Department<select
          ?disabled=${view.busy || view.blocked}
          @change=${(event: Event) => view.set('department', input(event))}
        >
          <option value="" ?selected=${!view.department}>No department</option>
          ${view.departments.map(
            (item) => html`<option value=${item.id} ?selected=${view.department === item.id}>${item.name}</option>`
          )}
        </select></label
      ><button
        class="open-department"
        type="button"
        @click=${() => view.dialog(true)}
        ?disabled=${view.busy || view.blocked}
      >
        New department
      </button>
    </div>
    ${bounds(view)} ${savedRequests(view)}
  </section>`;
}
function ideaBasis(idea: ProductIdea): string {
  if (idea.source === 'autonomous' || idea.provenance === 'autonomous') return 'Suggested';
  if (idea.source === 'requested') return 'You requested';
  if (idea.source === 'assumption' || idea.assumption) return 'Assumption';
  if (idea.source === 'supported') return 'Supported by evidence';
  return '';
}
function ideaChoice(view: IntentView, idea: ProductIdea) {
  return html`<label
    >Your choice<select
      aria-label=${'Choice: ' + idea.description}
      .value=${idea.decision}
      ?disabled=${view.busy || view.blocked}
      @change=${(event: Event) => {
        const value = input(event);
        if (value === 'pending' || value === 'accept' || value === 'decline' || value === 'edit')
          view.decide(idea.id, value);
      }}
    >
      <option value="pending">Choose…</option>
      <option value="accept">Accept</option>
      <option value="decline">Decline</option>
      <option value="edit">Change</option>
    </select></label
  >`;
}
function ideaCorrectionField(view: IntentView, idea: ProductIdea) {
  if (idea.decision !== 'edit') return '';
  return html`<label
    >What should change?<textarea
      rows="3"
      maxlength="4000"
      .value=${idea.correction || ''}
      @input=${(event: Event) => view.decide(idea.id, 'edit', input(event))}
      ?disabled=${view.busy || view.blocked}
    ></textarea>
  </label>`;
}
function ideaSettled(idea: ProductIdea, settled: Set<string>): boolean {
  return settled.has(idea.id) || (idea.decision === 'accept' && idea.provenance === 'autonomous');
}
function ideaVote(idea: ProductIdea, challenged: Set<string>, referenced: Set<string>, settled: Set<string>): boolean {
  if (ideaSettled(idea, settled)) return false;
  return !challenged.has(idea.id) && referenced.has(idea.id);
}
function cardBasis(idea: ProductIdea, settled: Set<string>): string {
  if (ideaSettled(idea, settled)) return 'Decided automatically';
  return ideaBasis(idea);
}
function settledCorrection(view: IntentView, idea: ProductIdea, challenged: Set<string>, settled: Set<string>) {
  if (challenged.has(idea.id) || !ideaSettled(idea, settled)) return '';
  return html`<label
    >Correct this automatic choice<textarea
      rows="2"
      maxlength="4000"
      aria-label=${'Correct the automatic choice: ' + idea.description}
      placeholder="Leave empty to keep the automatic choice"
      .value=${idea.correction || ''}
      ?disabled=${view.busy || view.blocked}
      @input=${(event: Event) => {
        const node = event.target;
        if (node instanceof HTMLTextAreaElement) view.decide(idea.id, 'edit', node.value);
      }}
    ></textarea>
  </label>`;
}
function explorationSettled(view: IntentView): Set<string> {
  return new Set(view.exploration?.settled || []);
}
function correctionEcho(idea: ProductIdea, vote: boolean, settled: boolean) {
  if (vote || settled || !idea.correction) return '';
  return html`<p>Your correction: ${idea.correction}</p>`;
}
function productIdea(
  view: IntentView,
  idea: ProductIdea,
  challenged: Set<string>,
  referenced: Set<string>,
  settled: Set<string>
) {
  const vote = ideaVote(idea, challenged, referenced, settled);
  const basis = cardBasis(idea, settled);
  const isSettled = ideaSettled(idea, settled);
  return html`<article>
    <h3>${idea.assumption ? 'Assumption' : 'Idea'}</h3>
    <p>${idea.description}</p>
    <p class="consequence">${idea.consequence}</p>
    ${basis ? html`<p>${basis}</p>` : ''} ${vote ? ideaChoice(view, idea) : ''}
    ${vote ? ideaCorrectionField(view, idea) : ''} ${correctionEcho(idea, vote, isSettled)}
    ${settledCorrection(view, idea, challenged, settled)}
  </article>`;
}
function ideaDecided(idea: ProductIdea | undefined, allowDecline: boolean): boolean {
  if (!idea || idea.decision === 'pending') return false;
  if (idea.decision === 'decline') return allowDecline;
  return idea.decision !== 'edit' || !!idea.correction?.trim();
}
function tasksDecided(revision: IntentRevision, ideas: Map<string, ProductIdea>): boolean {
  for (const task of revision.tasks)
    for (const id of task.idea_ids) if (!ideaDecided(ideas.get(id), false)) return false;
  return true;
}
function challengesAnswered(view: IntentView, ideas: Map<string, ProductIdea>): boolean {
  for (const challenge of view.exploration?.challenges || [])
    if (!ideaDecided(ideas.get(challenge.idea), true)) return false;
  return true;
}
export function reviewReady(view: IntentView) {
  const revision = view.revision;
  if (!revision || view.dirty || view.busy || view.blocked || !revision.tasks.length) return false;
  const ideas = new Map(revision.ideas.map((idea) => [idea.id, idea]));
  return tasksDecided(revision, ideas) && challengesAnswered(view, ideas);
}
// planAcceptBlock explains why a completed plan cannot be accepted yet:
// a request-specific authored walkthrough plus its clean check must come
// first. It returns '' when acceptance may proceed (or when the plan is
// not completed, which the surrounding copy already addresses).
export function planAcceptBlock(view: IntentView) {
  const plan = view.plan;
  if (!plan || plan.revision !== view.revision?.id || plan.state !== 'completed') return '';
  if (!plan.walkthroughs)
    return 'This saved plan has no request-specific walkthrough. Author one in a new plan before accepting.';
  if (!plan.review) return '';
  if (!plan.review.consistency) return 'Check the walkthrough again before authorizing work.';
  if (plan.review.consistency.mismatches.length)
    return 'Correct or clarify the request above for the walkthrough findings (your objective and choices stay), then plan again and check before authorizing work.';
  return '';
}
// blockedCheckSummary states the checked state and next action near the top
// once an explicit Check leaves findings blocking acceptance, with a direct
// link that moves focus to the real findings below.
function blockedCheckSummary(view: IntentView) {
  const plan = view.plan;
  if (!plan || plan.revision !== view.revision?.id || plan.state !== 'completed' || !plan.review) return '';
  const block = planAcceptBlock(view);
  if (!block) return '';
  const findings = plan.review.consistency?.mismatches ?? [];
  return html`<p role="status">
    ${findings.length ? `Check found ${plural(findings.length, 'walkthrough finding', 'walkthrough findings')} blocking acceptance. ${block}` : block}
    ${findings.length ? html`<button @click=${view.focusFindings}>See findings</button>` : ''}
  </p>`;
}
function refreshExploration(view: IntentView) {
  if (view.revision?.state !== 'draft') return '';
  return html`<button ?disabled=${view.busy || view.blocked} @click=${view.revise}>Re-explore ideas</button>`;
}
function reviewLead(view: IntentView) {
  const plan = view.plan;
  if (plan && plan.revision === view.revision?.id && plan.walkthroughs) {
    const reviewed = plan.review;
    const content = reviewed ?? {content: plan.walkthroughs, challenges: [], settled: []};
    return html`<p>Authored for this request by the planning agent.</p>
      ${explorationReview(view, content)}
      ${!reviewed ? html`<p>Use Check walkthrough to ask JEV to compare these alternatives and check the recommended experience.</p>` : ''}`;
  }
  if (!view.exploration) return refreshExploration(view);
  return html`<p>
      This starter comparison helps clarify your direction. The native planning agent will author the concrete
      experience after project preparation.
    </p>
    ${explorationReview(view, view.exploration)} ${view.explorationStale && !view.dirty ? refreshExploration(view) : ''}`;
}
// allowanceHint states the pending action's call cost without promising a
// fixed price: exploring and checking each spend planning calls, while a
// repeated check of the same walkthrough reuses its saved result.
function allowanceHint(view: IntentView): string {
  const plan = view.plan;
  if (plan?.state === 'pending') return 'Planning is running; its consultations already hold their calls.';
  if (plan?.state === 'completed' && plan.walkthroughs && !plan.review)
    return 'Checking the walkthrough uses planning calls; re-checking the same walkthrough reuses the saved result.';
  if (plan?.review?.consistency?.mismatches.length)
    return 'Editing text uses no calls; submitting the change re-explores and can use planning calls, and checking a fresh walkthrough uses calls too. Unchanged saved judgments are reused, and a limits-only Update proposal just saves without re-exploring.';
  return 'Updating the proposal or re-exploring can use planning calls, except a limits-only update which just saves; unchanged consultations reuse saved results.';
}
function fundingSection(view: IntentView) {
  const allowance = view.allowance;
  if (view.revision?.state !== 'draft' || (!allowance && !view.allowanceError)) return '';
  if (!allowance)
    return html`<details aria-label="Planning allowance">
    <summary>Planning budget · could not load</summary>
    <p>${view.allowanceError}</p>
    <div class="actions">
      <button ?disabled=${view.busy || view.blocked} @click=${view.retryAllowance}>Retry allowance load</button>
    </div>
  </details>`;
  const used = allowance.reserved + allowance.consumed;
  return html`<details aria-label="Planning allowance">
    <summary>Planning budget: ${used} of ${allowance.amount} calls used</summary>
    <p>${allowanceHint(view)}</p>
    <label
      >Raise to<input
        type="number"
        min="1"
        max="100"
        .value=${String(view.raiseCalls)}
        ?disabled=${view.busy || view.blocked}
        @input=${(event: Event) => view.setRaise(Number(input(event)))}
    /></label>
    <button ?disabled=${view.busy || view.blocked} @click=${view.raiseAllowance}>Raise allowance</button>
    ${view.fundingNote ? html`<p>${view.fundingNote}</p>` : ''}
  </details>`;
}
function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`;
}
function generationText(view: IntentView, generation: IntentGenerationStatus): string {
  if (generation.authorization && generation.authorization !== view.revision?.id)
    return 'Factory work references an older accepted direction. Refresh the status after updating.';
  if (generation.state === 'pending') return 'Factory has not created work yet.';
  if (generation.state === 'committed') return 'Factory confirmed the native work.';
  if (generation.state === 'completed')
    return 'Factory work is created. Execution still needs its own access and finite funding.';
  if (generation.state === 'refused')
    return 'Factory could not create the native work; an administrator must inspect the refusal.';
  return 'Factory reported an unrecognized status. Your direction is saved.';
}
function generationStatus(view: IntentView) {
  if (view.generationError)
    return html`<p>${view.generationError}</p>
      <button ?disabled=${view.busy || view.blocked} @click=${view.refreshGeneration}>Retry status</button>`;
  const generation = view.generation;
  if (!generation) return html`<p>Loading Factory status…</p>`;
  return html`<p>${generationText(view, generation)}</p>
    <button ?disabled=${view.busy || view.blocked} @click=${view.refreshGeneration}>Refresh status</button>`;
}
function lossSummary(view: IntentView) {
  if (!view.unsavedCount) return '';
  return html`<p>
    ${plural(view.unsavedCount, 'unsaved edit is', 'unsaved edits are')} kept in this view: ${view.unsavedParts}. Update
    proposal saves them; the walkthrough, Factory status and repository choice stay on the saved version until then.
  </p>`;
}
function authorizationSection(view: IntentView) {
  const revision = view.revision;
  if (!revision || revision.state !== 'accepted') return '';
  const authorized = revision.ideas.filter((idea) => idea.decision === 'accept' || idea.decision === 'edit');
  return html`<section aria-label="Authorization and Factory work">
    <h3>Authorization and Factory work</h3>
    <p>
      Authorized: ${plural(authorized.length, 'idea', 'ideas')} and
      ${plural(revision.tasks.length, 'work item', 'work items')} from your accepted direction.
    </p>
    ${generationStatus(view)} ${lossSummary(view)}
  </section>`;
}
function deliveryApproach(view: IntentView, revision: IntentRevision) {
  return html`<details>
    <summary>Delivery approach</summary>
    <fieldset>
      <legend>Delivery approach</legend>
      <label
        >Choose how to deliver the accepted workflow<select
          aria-label="Choose how to deliver the accepted workflow"
          .value=${revision.approach || 'incremental'}
          ?disabled=${view.busy || view.blocked}
          @change=${(event: Event) => {
            const approach = input(event);
            if (approach === 'complete_first' || approach === 'incremental') view.chooseApproach(approach);
          }}
        >
          <option value="complete_first">Complete first workflow</option>
          <option value="incremental">Verified milestones</option>
        </select></label
      >
      <p>
        ${revision.approach === 'complete_first' ? 'Implement and verify the complete accepted workflow before first delivery' : 'Implement and check accepted outcomes in separate dependent work items; incomplete milestones do not claim complete delivery'}
      </p>
    </fieldset>
  </details>`;
}
// The action notice stays visible only when it adds to the status line: an
// identical message would announce twice, while a specific wait (a 202
// refusal naming the explicit retry) must stay visible beside it.
function planNoticeText(view: IntentView): string {
  if (!view.planNotice) return '';
  if (view.plan && view.planNotice === planText(view.plan)) return '';
  return view.planNotice;
}
function planText(status: IntentPlanStatus): string {
  switch (status.state) {
    case 'none':
      return 'Planning has not started for this draft.';
    case 'pending':
      return 'Planning is running. The result appears here when the run settles.';
    case 'completed':
      return 'Planning is complete. Accept the direction below to create its work.';
    case 'attention':
      return status.attention
        ? `Planning needs a product decision: ${status.attention}`
        : 'Planning needs a product decision before issues can be proposed.';
    case 'failed':
      return 'Planning failed. Your direction is saved; try again.';
    case 'cancelled':
      return 'Planning was cancelled before it could report.';
  }
}
function primaryDirectionButton(view: IntentView) {
  return html`<button
    class="primary"
    @click=${view.dirty ? view.revise : view.continue}
    ?disabled=${view.busy || view.blocked || view.plan?.state === 'pending' || (!view.dirty && (!reviewReady(view) || view.plan?.state === 'attention' || !!planAcceptBlock(view)))}
  >
    ${continueCaption(view)}
  </button>`;
}
function planSection(view: IntentView) {
  const revision = view.revision;
  if (!revision || revision.repository === '0') return '';
  if (revision.state === 'accepted')
    return html`<button @click=${view.prepareProject} ?disabled=${view.busy || view.blocked}>Prepare project</button>`;
  if (revision.state !== 'draft') return '';
  return html`<section aria-label="Draft planning">
    <h3>Build preparation</h3>
    ${planNoticeText(view) ? html`<p role="status">${planNoticeText(view)}</p>` : ''}
    ${view.plan ? html`<p role="status">${(view.plan.walkthroughs && planAcceptBlock(view)) || planText(view.plan)}</p>` : ''}
    ${view.plan?.state === 'completed' && !view.plan.walkthroughs ? html`<p>This saved plan has no request-specific walkthrough. Author one in a new plan before accepting.</p>` : ''}
    ${
      view.plan?.state === 'completed'
        ? ''
        : html`<p class="next-step" aria-live="polite">${reviewNextStep(view)}</p>
            <div class="actions">${primaryDirectionButton(view)}</div>`
    }
    <div class="actions">
      <button @click=${view.prepareProject} ?disabled=${view.busy || view.blocked}>Prepare project</button>
      <button @click=${view.refreshPlanStatus} ?disabled=${view.busy || view.blocked}>Refresh status</button>
      ${
        view.plan?.state === 'completed' && !view.plan.walkthroughs
          ? html`<button @click=${view.planWork} ?disabled=${view.busy || view.blocked || view.dirty}>
              Author walkthrough in a new plan
            </button>`
          : ''
      }
    </div>
    <details>
      <summary>Planning details</summary>
      <label
        >Planning operation<select
          aria-label="Planning operation"
          .value=${view.planOperation}
          ?disabled=${view.busy || view.blocked}
          @change=${(event: Event) => view.setPlanOperation(input(event))}
        >
          <option value="propose_intent">Review baseline</option>
          <option value="decomposition_options">Candidate issue graphs</option>
        </select></label
      >
      <div class="actions">
        <button
          @click=${view.planWork}
          ?disabled=${view.busy || view.blocked || view.dirty || view.plan?.state === 'pending'}
        >
          Plan work
        </button>
      </div>
    </details>
  </section>`;
}
function continueCaption(view: IntentView): string {
  if (view.dirty) return 'Update proposal';
  if (view.revision?.repository === '0') return 'Choose repository';
  if (view.plan?.state === 'pending') return 'Planning…';
  if (view.plan?.state === 'completed') {
    if (!view.plan.walkthroughs) return 'Plan walkthrough first';
    if (!view.plan.review) return 'Check walkthrough';
    if (view.plan.review?.consistency?.mismatches.length) return 'Correct walkthrough first';
    return 'Accept and build';
  }
  return 'Continue to planning';
}
function reviewNextStep(view: IntentView): string {
  if (view.dirty) return 'Save your changes with Update proposal before continuing.';
  if (view.plan?.state === 'pending') return 'Planning is running. You can leave and return to this saved request.';
  if (view.plan?.state === 'attention')
    return 'Review the planning question above and update your direction before continuing.';
  if (view.plan?.review?.consistency?.mismatches.length)
    return 'Correct or clarify the request above for the walkthrough findings (your objective and choices stay), then plan again and check before authorizing work.';
  if (!view.revision?.tasks.length)
    return 'Explain the missing behavior in your desired outcome, then update the proposal.';
  if (!reviewReady({...view, busy: false, blocked: false}))
    return 'Answer the questions and choose the ideas needed for this workflow before continuing.';
  if (view.revision?.repository === '0')
    return 'Choose where to keep this application. Your request and choices stay together.';
  if (view.plan?.state === 'completed') {
    if (!view.plan.walkthroughs)
      return 'This saved plan has no request-specific walkthrough. Author one in a new plan before accepting.';
    if (!view.plan.review) return 'Review the walkthrough, then check it before authorizing work.';
    return 'Accept this direction to create its work. App delivery is tracked separately.';
  }
  return 'Your request and choices are saved. Continue to planning when project preparation is ready.';
}
export function reviewIntent(view: IntentView) {
  const revision = view.revision;
  if (!revision) return '';
  const challenged = new Set((view.exploration?.challenges || []).map((challenge) => challenge.idea));
  const settled = explorationSettled(view);
  const referenced = new Set(revision.tasks.flatMap((task) => task.idea_ids));
  return html`<section class="intent-review">
    <h2 tabindex="-1">Review your product direction</h2>
    ${blockedCheckSummary(view)}
    <label
      >Desired outcome<textarea
        rows="3"
        maxlength="16000"
        .value=${view.objective}
        ?disabled=${view.busy || view.blocked}
        @input=${(event: Event) => view.set('objective', input(event))}
      ></textarea>
    </label>
    ${reviewLead(view)} ${revision.state === 'draft' ? bounds(view) : ''} ${fundingSection(view)}
    ${authorizationSection(view)} ${planSection(view)}
    ${view.generationNotice ? html`<p role="status">${view.generationNotice}</p>` : ''}
    <p>Only asked questions and work-linked choices are required. Changes update the proposal before you continue.</p>
    ${deliveryApproach(view, revision)}
    ${!revision.tasks.length ? html`<p>More product direction is needed before work can be proposed. Edit the desired outcome or change an idea to explain the missing behavior, then update the proposal.</p>` : ''}
    ${revision.ideas.filter((idea) => !ideaSettled(idea, settled)).map((idea) => productIdea(view, idea, challenged, referenced, settled))}
    <details class="recorded-choices">
      <summary>Recorded choices (${revision.ideas.filter((idea) => ideaSettled(idea, settled)).length})</summary>
      ${revision.ideas.filter((idea) => ideaSettled(idea, settled)).map((idea) => productIdea(view, idea, challenged, referenced, settled))}
    </details>
    ${
      revision.state === 'accepted' && view.dirty
        ? html`<p class="next-step" aria-live="polite">${reviewNextStep(view)}</p>
            <button class="primary" ?disabled=${view.busy || view.blocked} @click=${view.revise}>
              Update proposal
            </button>`
        : ''
    }
    <details>
      <summary>Proposed work (${revision.tasks.length})</summary>
      ${revision.tasks.map(
        (task) =>
          html`<article>
            <h3>${task.objective}</h3>
            <ul>
              ${task.outcomes.map((outcome) => html`<li>${outcome}</li>`)}
            </ul>
          </article>`
      )}
    </details>
    ${(() => {
      if (revision.state === 'accepted') return '';
      // planSection already renders the primary action for a repository-bound
      // draft without a completed plan; every other non-accepted case keeps
      // the footer primary so unbound and dirty requests stay actionable.
      const movedPrimary =
        revision.state === 'draft' && revision.repository !== '0' && view.plan?.state !== 'completed';
      if (movedPrimary)
        return html`<div class="actions">
          <button @click=${view.decline} ?disabled=${view.busy || view.blocked}>Decline proposal</button>
        </div>`;
      return html`<p class="next-step" aria-live="polite">${reviewNextStep(view)}</p>
        <div class="actions">
          <button @click=${view.decline} ?disabled=${view.busy || view.blocked}>Decline proposal</button
          >${primaryDirectionButton(view)}
        </div>`;
    })()}
  </section>`;
}
export function departmentDialog(view: IntentView) {
  if (!view.departmentDialog) return '';
  return html`<section
    role="dialog"
    aria-modal="true"
    aria-label="New department"
    aria-describedby="department-private-note"
    class="department-dialog"
    @keydown=${view.dialogKey}
  >
    <h2>New department</h2>
    <p id="department-private-note">
      Name and description are required. Private context for your ideas. It grants no access.
    </p>
    <label
      >Name<input
        maxlength="120"
        .value=${view.departmentName}
        @input=${(event: Event) => view.set('departmentName', input(event))}
    /></label>
    <label
      >What does it do?<textarea
        rows="3"
        maxlength="2000"
        .value=${view.departmentDescription}
        @input=${(event: Event) => view.set('departmentDescription', input(event))}
      ></textarea>
    </label>
    <div class="actions">
      <button @click=${() => view.dialog(false)} ?disabled=${view.busy}>Cancel</button
      ><button
        class="primary"
        @click=${view.createDepartment}
        ?disabled=${view.busy || view.blocked || !view.departmentName.trim() || !view.departmentDescription.trim()}
      >
        Create department
      </button>
    </div>
  </section>`;
}

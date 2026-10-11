import {html} from 'lit';
import type {Department, IntentRevision, ProductIdea, SavedRequest} from './sodaspaces-intent-response.js';

export interface IntentView {
  objective: string;
  saveNote: string;
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
  set(field: 'objective' | 'department' | 'departmentName' | 'departmentDescription', value: string): void;
  setBound(field: 'maxIssues' | 'planningCalls', value: number): void;
  decide(id: string, decision: ProductIdea['decision'], correction?: string): void;
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
  return html`<details class="bounds">
    <summary>Planning limits: ${view.planningCalls} calls</summary>
    <label
      >Maximum work items<input
        type="number"
        min="1"
        max="64"
        .value=${String(view.maxIssues)}
        @input=${(event: Event) => view.setBound('maxIssues', Number(input(event)))}
    /></label>
    <label
      >Planning calls<input
        type="number"
        min="1"
        max="100"
        .value=${String(view.planningCalls)}
        @input=${(event: Event) => view.setBound('planningCalls', Number(input(event)))}
    /></label>
    <p>Work still needs its own access and funded execution limits.</p>
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
  const work = item.task_count ? ` · ${item.task_count} work items` : '';
  return html`<li>
    <p>${item.objective}</p>
    <p>${requestState(item.state)}${work}</p>
    <button ?disabled=${view.busy || view.blocked} @click=${() => view.resumeRequest(item.intent)}>Resume</button>
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
    <h3>Saved requests</h3>
    <ul>
      ${view.savedRequests.map((item) => savedRequestItem(view, item))}
    </ul>
    ${view.savedMore ? html`<button @click=${view.moreSaved}>Show more</button>` : ''}
  </section>`;
}
export function describeIntent(view: IntentView) {
  return html`<section class="intent-entry">
    <h2 tabindex="-1">What do you want to build?</h2>
    ${terminalNote(view)}
    <label
      ><span class="sr-only">Describe what you want to build</span
      ><textarea
        rows="7"
        maxlength="16000"
        placeholder="Describe the outcome you want…"
        .value=${view.objective}
        ?disabled=${view.busy || view.blocked}
        @input=${(event: Event) => view.set('objective', input(event))}
      ></textarea>
    </label>
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
    ${bounds(view)} ${entryActions(view)}
    <p>${view.saveNote}</p>
    <p>Only saved drafts can be resumed after reload. Unsaved edits stay in this view.</p>
    ${savedRequests(view)}
  </section>`;
}
function productIdea(view: IntentView, idea: ProductIdea) {
  return html`<article>
    <h3>${idea.assumption ? 'Assumption' : 'Idea'}</h3>
    <p>${idea.description}</p>
    <p class="consequence">${idea.consequence}</p>
    <label
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
    >
    ${idea.decision === 'edit' ? html`<label>What should change?<textarea rows="3" maxlength="4000" .value=${idea.correction || ''} @input=${(event: Event) => view.decide(idea.id, 'edit', input(event))} ?disabled=${view.busy || view.blocked}></textarea></label>` : ''}
  </article>`;
}
export function reviewReady(view: IntentView) {
  const revision = view.revision;
  if (!revision || view.dirty || view.busy || view.blocked || !revision.tasks.length) return false;
  return (
    revision.ideas.length > 0 &&
    revision.ideas.every(
      (idea) => idea.decision !== 'pending' && (idea.decision !== 'edit' || !!idea.correction?.trim())
    )
  );
}
function deliveryApproach(view: IntentView, revision: IntentRevision) {
  return html` <fieldset>
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
  </fieldset>`;
}
export function reviewIntent(view: IntentView) {
  const revision = view.revision;
  if (!revision) return '';
  return html`<section class="intent-review">
    <h2 tabindex="-1">Review your product direction</h2>
    <label
      >Desired outcome<textarea
        rows="3"
        maxlength="16000"
        .value=${view.objective}
        ?disabled=${view.busy || view.blocked}
        @input=${(event: Event) => view.set('objective', input(event))}
      ></textarea>
    </label>
    <p>Choose the ideas and assumptions that match what you want. Changes update the proposal before you continue.</p>
    ${deliveryApproach(view, revision)}
    ${!revision.tasks.length ? html`<p>More product direction is needed before work can be proposed. Edit the desired outcome or change an idea to explain the missing behavior, then update the proposal.</p>` : ''}
    ${revision.ideas.map((idea) => productIdea(view, idea))}
    <button ?disabled=${view.busy || view.blocked || !view.dirty} @click=${view.revise}>Update proposal</button>
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
    <div class="actions">
      <button @click=${view.decline} ?disabled=${view.busy || view.blocked}>Decline proposal</button
      ><button class="primary" @click=${view.continue} ?disabled=${!reviewReady(view)}>Choose repository</button>
    </div>
    <p>Choosing a repository comes next. Creating a Project, joining and running work remain explicit actions.</p>
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

import {html} from 'lit';
import type {Department, IntentRevision, ProductIdea} from './sodaspaces-intent-response.js';

export interface IntentView {
  objective: string;
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
  describe(): void;
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
export function describeIntent(view: IntentView) {
  return html`<section class="intent-entry">
    <h2 tabindex="-1">What do you want to build?</h2>
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
          .value=${view.department}
          ?disabled=${view.busy || view.blocked}
          @change=${(event: Event) => view.set('department', input(event))}
        >
          <option value="">No department</option>
          ${view.departments.map((item) => html`<option value=${item.id}>${item.name}</option>`)}
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
    ${bounds(view)}<button
      class="primary"
      ?disabled=${view.busy || view.blocked || !view.objective.trim()}
      @click=${view.describe}
    >
      Explore ideas
    </button>
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
    class="department-dialog"
    @keydown=${view.dialogKey}
  >
    <h2>New department</h2>
    <p>Private context for your ideas. It grants no access.</p>
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

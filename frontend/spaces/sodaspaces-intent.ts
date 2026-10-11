import {LitElement, html, css} from 'lit';
import {department, departments, intentRevision, newDecisionID} from './sodaspaces-intent-response.js';
import type {Department, IntentRevision, ProductIdea} from './sodaspaces-intent-response.js';
import {describeIntent, reviewIntent, departmentDialog, reviewReady} from './sodaspaces-intent-view.js';
import type {IntentView} from './sodaspaces-intent-view.js';

type IntentAPI = (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => Promise<unknown>;
export class SodaIntent extends LitElement {
  static properties = {
    api: {attribute: false},
    blocked: {type: Boolean},
    objective: {state: true},
    department: {state: true},
    departments: {state: true},
    revision: {state: true},
    busy: {state: true},
    dirty: {state: true},
    error: {state: true},
    departmentDialog: {state: true},
    departmentName: {state: true},
    departmentDescription: {state: true},
    maxIssues: {state: true},
    planningCalls: {state: true},
  };
  static styles = css`
    :host {
      display: block;
      width: 100%;
      font: var(--soda-font-body, 16px sans-serif);
      color: var(--soda-page-text, inherit);
    }
    :host([hidden]) {
      display: none;
    }
    section {
      max-width: 720px;
      margin: auto;
    }
    .intent-entry {
      padding-block: clamp(1rem, 8vh, 6rem);
    }
    h2 {
      font: var(--soda-font-title, 32px sans-serif);
      margin: 0 0 1.5rem;
    }
    h3 {
      font: var(--soda-font-heading, 20px sans-serif);
    }
    label {
      display: block;
    }
    textarea,
    input,
    select,
    button {
      box-sizing: border-box;
      font: inherit;
      color: inherit;
      border: 1px solid var(--soda-page-edge, #aaa);
      background: var(--soda-page-surface, #fff);
      border-radius: 0;
      padding: 0.75rem;
    }
    textarea {
      width: 100%;
      resize: vertical;
    }
    input,
    select {
      display: block;
      max-width: 100%;
      margin-block: 0.5rem;
    }
    button {
      cursor: pointer;
      min-height: 44px;
    }
    button:disabled {
      opacity: 0.5;
      cursor: default;
    }
    .primary {
      background: var(--soda-button-primary-bg, #c42a31);
      color: var(--soda-button-primary-text, #fff);
    }
    .context,
    .actions {
      display: flex;
      align-items: end;
      flex-wrap: wrap;
      gap: 1rem;
      margin-block: 1rem;
    }
    .context label {
      flex: 1;
    }
    .bounds {
      margin-block: 1rem;
    }
    article {
      border-block-end: 1px solid var(--soda-page-edge, #aaa);
      padding-block: 1rem;
    }
    .consequence {
      color: var(--soda-page-muted, #666);
    }
    .department-dialog {
      position: fixed;
      z-index: 40;
      inset: 15% max(1rem, calc((100vw - 600px)/2)) auto;
      padding: 1.5rem;
      background: var(--soda-page-surface, #fff);
      border: 1px solid var(--soda-page-edge, #aaa);
      box-shadow: 0 0 0 100vmax #0008;
      max-height: 70vh;
      overflow: auto;
    }
    .department-dialog input {
      width: 100%;
    }
    .sr-only {
      position: absolute;
      width: 1px;
      height: 1px;
      overflow: hidden;
      clip-path: inset(50%);
    }
    [role='status'] {
      margin-block: 1rem;
    }
  `;
  declare api: IntentAPI | undefined;
  declare blocked: boolean;
  declare objective: string;
  declare department: string;
  declare departments: Department[];
  declare revision: IntentRevision | undefined;
  declare busy: boolean;
  declare dirty: boolean;
  declare error: string;
  declare departmentDialog: boolean;
  declare departmentName: string;
  declare departmentDescription: string;
  declare maxIssues: number;
  declare planningCalls: number;
  private request: AbortController | undefined;
  constructor() {
    super();
    this.blocked = false;
    this.objective = '';
    this.department = '';
    this.departments = [];
    this.busy = this.dirty = this.departmentDialog = false;
    this.error = this.departmentName = this.departmentDescription = '';
    this.maxIssues = 20;
    this.planningCalls = 3;
  }
  protected firstUpdated() {
    void this.loadDepartments();
  }
  disconnectedCallback() {
    this.request?.abort();
    super.disconnectedCallback();
  }
  reset() {
    this.request?.abort();
    this.revision = undefined;
    this.objective = '';
    this.dirty = false;
    this.error = '';
  }
  private async perform(operation: (signal: AbortSignal) => Promise<void>) {
    if (this.busy || this.blocked || !this.api) return false;
    const request = new AbortController();
    this.request = request;
    this.busy = true;
    this.error = '';
    try {
      await operation(request.signal);
      return !request.signal.aborted && !this.blocked;
    } catch {
      if (!request.signal.aborted)
        this.error = 'Could not save or plan this request. Your choices are kept; try again.';
      return false;
    } finally {
      if (this.request === request) this.busy = false;
    }
  }
  private async call(path: string, body: Record<string, unknown> | undefined, signal: AbortSignal) {
    if (!this.api) throw Error('Intent API unavailable');
    const response = await this.api(path, body, signal);
    if (signal.aborted || this.blocked || !this.isConnected) throw Error('Intent request retired');
    return response;
  }
  private async loadDepartments() {
    await this.perform(async (signal) => {
      this.departments = departments(await this.call('/api/factory/departments', undefined, signal));
    });
  }
  private nextRevision(state: IntentRevision['state'], repository = '0'): IntentRevision {
    const current = this.revision;
    return {
      id: newDecisionID(),
      intent: current?.intent || newDecisionID(),
      ...(current ? {predecessor: current.id} : {}),
      repository,
      ...(this.department ? {department: this.department} : {}),
      objective: this.objective.trim(),
      ...this.currentProposal(),
      state,
      max_issues: this.maxIssues,
      planning_calls: this.planningCalls,
    };
  }
  private currentProposal() {
    return {ideas: this.revision?.ideas || [], tasks: this.revision?.tasks || []};
  }
  private async save(state: IntentRevision['state'], repository: string, signal: AbortSignal) {
    const next = this.nextRevision(state, repository);
    const saved = intentRevision(await this.call('/api/factory/intents', {...next}, signal));
    if (
      saved.id !== next.id ||
      saved.intent !== next.intent ||
      saved.state !== state ||
      saved.repository !== repository
    )
      throw Error('Mismatched intent revision');
    this.revision = saved;
    return saved;
  }
  private async explore(signal: AbortSignal) {
    const saved = await this.save('draft', '0', signal);
    const proposal = intentRevision(
      await this.call('/api/factory/intents/' + saved.intent + '/explore', {revision: saved.id}, signal)
    );
    if (proposal.intent !== saved.intent || proposal.predecessor !== saved.id || proposal.state !== 'draft')
      throw Error('Mismatched proposal');
    this.revision = proposal;
    this.dirty = false;
  }
  private describe = () => {
    if (this.objective.trim()) void this.perform((signal) => this.explore(signal));
  };
  private revise = () => {
    void this.perform((signal) => this.explore(signal));
  };
  private decline = () => {
    void this.perform(async (signal) => {
      await this.save('declined', '0', signal);
      this.revision = undefined;
      this.dirty = false;
    });
  };
  async acceptRepository(repository: string): Promise<boolean> {
    if (this.revision?.state === 'accepted' && this.revision.repository === repository) return true;
    if (!reviewReady(this.view())) return false;
    return this.perform(async (signal) => {
      await this.save('accepted', repository, signal);
    });
  }
  private continue = () => {
    if (reviewReady(this.view()))
      this.dispatchEvent(new CustomEvent('intent-reviewed', {bubbles: true, composed: true}));
  };
  private decide = (id: string, decision: ProductIdea['decision'], correction?: string) => {
    if (!this.revision || this.busy || this.blocked) return;
    this.revision = {
      ...this.revision,
      ideas: this.revision.ideas.map((idea) =>
        idea.id === id ? {...idea, decision, ...(correction !== undefined ? {correction} : {})} : idea
      ),
    };
    this.dirty = true;
  };
  private createDepartment = () => {
    void this.perform(async (signal) => {
      const value = department(
        await this.call(
          '/api/factory/departments',
          {id: newDecisionID(), name: this.departmentName.trim(), description: this.departmentDescription.trim()},
          signal
        )
      );
      this.departments = [...this.departments, value];
      this.department = value.id;
      this.toggleDepartment(false);
    });
  };
  private toggleDepartment = (open: boolean) => {
    this.departmentDialog = open;
    void this.updateComplete.then(() => {
      const selector = open ? '.department-dialog input' : '.open-department';
      this.shadowRoot?.querySelector<HTMLElement>(selector)?.focus();
    });
  };
  private dialogKey = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && !this.busy) {
      event.preventDefault();
      this.toggleDepartment(false);
    }
    if (event.key !== 'Tab') return;
    this.trapDialogTab(event);
  };
  private trapDialogTab(event: KeyboardEvent) {
    const controls = Array.from(
      this.shadowRoot?.querySelectorAll<HTMLElement>(
        '.department-dialog input, .department-dialog textarea, .department-dialog button:not([disabled])'
      ) || []
    );
    const boundary = event.shiftKey ? controls[0] : controls.at(-1);
    if (this.shadowRoot?.activeElement !== boundary) return;
    event.preventDefault();
    (event.shiftKey ? controls.at(-1) : controls[0])?.focus();
  }
  private view(): IntentView {
    return {
      objective: this.objective,
      department: this.department,
      departments: this.departments,
      revision: this.revision,
      busy: this.busy,
      blocked: this.blocked,
      dirty: this.dirty,
      error: this.error,
      departmentDialog: this.departmentDialog,
      departmentName: this.departmentName,
      departmentDescription: this.departmentDescription,
      maxIssues: this.maxIssues,
      planningCalls: this.planningCalls,
      set: (field, value) => {
        this[field] = value;
      },
      setBound: (field, value) => {
        this[field] = value;
      },
      decide: this.decide,
      describe: this.describe,
      revise: this.revise,
      decline: this.decline,
      continue: this.continue,
      dialog: this.toggleDepartment,
      dialogKey: this.dialogKey,
      createDepartment: this.createDepartment,
    };
  }
  protected render() {
    const view = this.view();
    return html`<div .inert=${this.departmentDialog}>
        ${this.revision?.ideas.length ? reviewIntent(view) : describeIntent(view)}
      </div>
      <p role="status">${this.busy ? 'Saving and planning…' : this.error}</p>
      ${departmentDialog(view)}`;
  }
}
customElements.define('soda-intent', SodaIntent);

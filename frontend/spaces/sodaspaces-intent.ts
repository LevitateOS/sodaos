import {LitElement, html, css} from 'lit';
import type {PropertyValues} from 'lit';
import {object, check, id, SodaRequestError} from './sodaspaces-api.js';
import {
  decisionID,
  department,
  departments,
  exploration as parseExploration,
  intentList,
  intentRevision,
  newDecisionID,
} from './sodaspaces-intent-response.js';
import type {Department, Exploration, IntentRevision, ProductIdea, SavedRequest} from './sodaspaces-intent-response.js';
import {describeIntent, reviewIntent, departmentDialog, reviewReady} from './sodaspaces-intent-view.js';
import type {IntentView} from './sodaspaces-intent-view.js';

type IntentAPI = (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => Promise<unknown>;
export class SodaIntent extends LitElement {
  static properties = {
    api: {attribute: false},
    blocked: {type: Boolean},
    active: {type: Boolean},
    actor: {type: String},
    request: {type: String},
    exploration: {state: true},
    objective: {state: true},
    savedRequests: {state: true},
    savedLoaded: {state: true},
    savedError: {state: true},
    savedNext: {state: true},
    department: {state: true},
    departments: {state: true},
    revision: {state: true},
    busy: {state: true},
    busyNote: {state: true},
    dirty: {state: true},
    error: {state: true},
    departmentDialog: {state: true},
    departmentName: {state: true},
    departmentDescription: {state: true},
    changeOpen: {state: true},
    changeText: {state: true},
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
    .intent-review {
      text-align: left;
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
  declare active: boolean;
  declare actor: string;
  declare request: string;
  declare exploration: Exploration | undefined;
  declare savedRequests: SavedRequest[];
  declare savedLoaded: boolean;
  declare savedError: string;
  declare savedNext: string | undefined;
  declare objective: string;
  declare department: string;
  declare departments: Department[];
  declare revision: IntentRevision | undefined;
  declare busy: boolean;
  declare busyNote: string;
  declare dirty: boolean;
  declare error: string;
  declare departmentDialog: boolean;
  declare departmentName: string;
  declare departmentDescription: string;
  declare changeOpen: boolean;
  declare changeText: string;
  declare maxIssues: number;
  declare planningCalls: number;
  private operation: AbortController | undefined;
  private resumeController: AbortController | undefined;
  private resumeID = '';
  private loadedRequest = '';
  private departmentsLoaded = false;
  generationNotice = '';
  constructor() {
    super();
    this.blocked = false;
    this.active = false;
    this.actor = '';
    this.request = '';
    this.exploration = undefined;
    this.savedRequests = [];
    this.savedLoaded = false;
    this.savedError = '';
    this.savedNext = undefined;
    this.objective = '';
    this.department = '';
    this.departments = [];
    this.busy = this.dirty = this.departmentDialog = false;
    this.busyNote = '';
    this.error = this.departmentName = this.departmentDescription = this.changeText = '';
    this.changeOpen = false;
    this.maxIssues = 20;
    this.planningCalls = 8;
  }
  protected updated(changed: PropertyValues<this>) {
    if (changed.has('active') || changed.has('blocked') || (changed.has('busy') && !this.busy)) this.activate();
    if (this.resumeReady(changed)) this.resume();
  }
  private activate() {
    if (!this.active || this.blocked || !this.api) return;
    if (!this.departmentsLoaded) {
      void this.loadDepartments().then((ok) => {
        if (ok) this.fetchSaved(false);
      });
      return;
    }
    if (!this.savedLoaded && !this.savedError) this.fetchSaved(false);
  }
  private resumeReady(changed: PropertyValues<this>) {
    return changed.has('request') || changed.has('api') || changed.has('active') || (changed.has('busy') && !this.busy);
  }
  disconnectedCallback() {
    this.operation?.abort();
    this.resumeController?.abort();
    super.disconnectedCallback();
  }
  reset() {
    this.operation?.abort();
    this.resumeController?.abort();
    this.request = '';
    this.loadedRequest = '';
    this.revision = undefined;
    this.exploration = undefined;
    this.objective = '';
    this.dirty = false;
    this.error = '';
    this.changeOpen = false;
    this.changeText = '';
  }
  private async perform(operation: (signal: AbortSignal) => Promise<void>, note = 'Saving and planning…') {
    if (this.busy || this.blocked || !this.api) return false;
    const request = new AbortController();
    this.operation = request;
    this.busy = true;
    this.busyNote = note;
    this.error = '';
    try {
      await operation(request.signal);
      return !request.signal.aborted && !this.blocked;
    } catch {
      if (!request.signal.aborted)
        this.error = 'Could not save or plan this request. Your choices are kept; try again.';
      return false;
    } finally {
      if (this.operation === request) this.busy = false;
    }
  }
  private async call(path: string, body: Record<string, unknown> | undefined, signal: AbortSignal) {
    if (!this.api) throw Error('Intent API unavailable');
    const response = await this.api(path, body, signal);
    if (signal.aborted || this.blocked || !this.isConnected) throw Error('Intent request retired');
    return response;
  }
  private loadDepartments() {
    return this.perform(async (signal) => {
      this.departments = departments(await this.call('/api/factory/departments', undefined, signal));
      this.departmentsLoaded = true;
    }, 'Loading…');
  }
  private fetchSaved(more: boolean) {
    if (more && this.savedNext === undefined) return;
    const prior = this.error;
    void this.perform(async (signal) => {
      const query = more && this.savedNext ? `?cursor=${this.savedNext}&limit=20` : '?limit=20';
      try {
        const page = intentList(await this.call('/api/factory/intents' + query, undefined, signal));
        this.savedRequests = more ? [...this.savedRequests, ...page.intents] : page.intents;
        this.savedNext = page.next_cursor;
        this.savedLoaded = true;
        this.savedError = '';
      } catch {
        if (!signal.aborted) this.savedError = 'Could not load saved requests. Your edits are kept; try again.';
      }
    }, 'Loading saved requests…').then(() => {
      if (this.error === '') this.error = prior;
    });
  }
  private resumeSuperseded(id: string): boolean {
    if (!this.resumeController) return this.busy;
    if (this.resumeID === id) return true;
    this.resumeController.abort();
    return false;
  }
  private resumeAdmitted(id: string): boolean {
    if (!id || !this.api || !this.active || this.blocked) return false;
    if (id === this.loadedRequest || id === this.revision?.intent) return false;
    return !this.resumeSuperseded(id);
  }
  private resume() {
    const id = this.request.trim();
    if (!this.resumeAdmitted(id)) return;
    if (!decisionID(id)) {
      this.error = 'This saved request address is invalid. Your current edits are kept.';
      return;
    }
    const controller = new AbortController();
    this.resumeController = controller;
    this.resumeID = id;
    this.busy = true;
    this.busyNote = 'Loading saved request…';
    this.error = '';
    void this.loadRequest(id, controller);
  }
  private async loadRequest(id: string, controller: AbortController) {
    try {
      const saved = intentRevision(await this.call('/api/factory/intents/' + id, undefined, controller.signal));
      if (controller.signal.aborted || this.resumeController !== controller || this.request !== id) return;
      if (saved.intent !== id) throw Error('Mismatched intent revision');
      this.applyRevision(saved);
      this.loadedRequest = id;
      this.dispatchEvent(
        new CustomEvent('intent-resumed', {
          detail: {intent: saved.intent, revision: saved.id, state: saved.state},
          bubbles: true,
          composed: true,
        })
      );
    } catch (error) {
      this.resumeFailed(id, controller, error);
    } finally {
      if (this.resumeController === controller) {
        this.resumeController = undefined;
        this.resumeID = '';
        this.busy = false;
      }
    }
  }
  private resumeFailed(id: string, controller: AbortController, error: unknown) {
    if (controller.signal.aborted || this.resumeController !== controller) return;
    this.loadedRequest = id;
    if (error instanceof SodaRequestError && error.status === 404)
      this.error = 'This saved request is unavailable to the current account. Your current edits are kept.';
    else this.error = 'Could not load the saved request. Your current edits are kept; try again.';
  }
  private applyRevision(saved: IntentRevision) {
    this.revision = saved;
    this.exploration = undefined;
    this.objective = saved.objective;
    this.department = saved.department || '';
    this.maxIssues = saved.max_issues;
    this.planningCalls = saved.planning_calls;
    this.dirty = false;
    this.changeOpen = false;
    this.changeText = '';
  }
  private explorationStale(): boolean {
    const exploration = this.exploration;
    const revision = this.revision;
    if (!exploration || !revision) return false;
    if (this.dirty) return true;
    const content = exploration.content;
    if (content.intent !== undefined && content.intent !== revision.intent) return true;
    if (content.revision !== undefined && content.revision !== revision.id) return true;
    return false;
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
      ...(current?.approach ? {approach: current.approach} : {}),
    };
  }
  private terminalRevision(): boolean {
    const state = this.revision?.state;
    return state === 'declined' || state === 'rejected' || state === 'abandoned';
  }
  private currentProposal() {
    if (this.terminalRevision()) return {ideas: [], tasks: []};
    if (this.objective.trim() !== this.revision?.objective) return {ideas: [], tasks: []};
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
    this.savedLoaded = false;
    this.savedError = '';
    return saved;
  }
  private async explore(signal: AbortSignal) {
    const saved = await this.save('draft', '0', signal);
    try {
      const response = await this.call(
        '/api/factory/intents/' + saved.intent + '/explore',
        {revision: saved.id},
        signal
      );
      const proposal = intentRevision(response);
      if (proposal.intent !== saved.intent || proposal.predecessor !== saved.id || proposal.state !== 'draft')
        throw Error('Mismatched proposal');
      this.exploration = readExploration(response, proposal);
      this.revision = proposal;
      this.changeOpen = false;
      this.changeText = '';
      this.dirty = false;
    } catch (error) {
      if (!signal.aborted) this.exploreFailed(error);
    }
  }
  private exploreFailed(error: unknown) {
    if (error instanceof SodaRequestError && error.code === 'planning_unavailable')
      this.error =
        'Exploration exhausted the planning allowance. Your description is saved; raise the allowance or try again.';
    else if (error instanceof SodaRequestError && error.code === 'planning_funding_required')
      this.error =
        'Exploration needs a planning allowance. Your description is saved; set the planning allowance and try again.';
    else this.error = 'Exploration could not run. Your description is saved; try again.';
  }
  private describe = () => {
    if (this.objective.trim()) void this.perform((signal) => this.explore(signal));
  };
  private saveCurrent = async (signal: AbortSignal) => {
    const saved = await this.save('draft', this.revision?.repository || '0', signal);
    this.dirty = false;
    this.dispatchEvent(
      new CustomEvent('intent-saved', {
        detail: {intent: saved.intent, revision: saved.id, state: saved.state},
        bubbles: true,
        composed: true,
      })
    );
  };
  private saveDraft = () => {
    if (this.objective.trim()) void this.perform(this.saveCurrent, 'Saving…');
  };
  private saveAndLeave = () => {
    if (!this.objective.trim()) return;
    void this.perform(async (signal) => {
      await this.saveCurrent(signal);
      this.dispatchEvent(new CustomEvent('intent-leave', {bubbles: true, composed: true}));
    }, 'Saving…');
  };
  private revise = () => {
    void this.perform((signal) => this.explore(signal));
  };
  private decline = () => {
    void this.perform(async (signal) => {
      await this.save('declined', '0', signal);
      this.revision = undefined;
      this.exploration = undefined;
      this.dirty = false;
      this.changeOpen = false;
      this.changeText = '';
    });
  };
  async acceptRepository(repository: string): Promise<boolean> {
    if (!reviewReady(this.view())) return false;
    return this.perform(async (signal) => {
      if (this.revision?.state !== 'accepted' || this.revision.repository !== repository)
        await this.save('accepted', repository, signal);
      await this.generate(signal);
    });
  }
  private async generate(signal: AbortSignal) {
    const accepted = this.revision;
    if (!accepted || accepted.state !== 'accepted') throw Error('Product direction not accepted');
    try {
      const generation = object(await this.call('/api/factory/intents/' + accepted.intent + '/generate', {}, signal));
      check(generation.intent_id === accepted.intent && generation.revision_id === accepted.id);
      this.generationNotice = generationMessage(generation.state);
    } catch (error) {
      this.generationNotice = generationFailure(error);
    }
  }
  private continue = () => {
    if (reviewReady(this.view()))
      this.dispatchEvent(new CustomEvent('intent-reviewed', {bubbles: true, composed: true}));
  };
  private actorValid(): boolean {
    return this.actor !== '' && id(this.actor);
  }
  private decidedIdea(idea: ProductIdea, decision: ProductIdea['decision'], correction?: string): ProductIdea {
    const next: ProductIdea = {...idea, decision};
    if (correction !== undefined) next.correction = correction;
    else if (decision !== 'edit') delete next.correction;
    if (decision === 'pending' || !this.actorValid()) {
      delete next.provenance;
      delete next.actor;
    } else {
      next.provenance = 'human';
      next.actor = this.actor;
    }
    return next;
  }
  private decide = (id: string, decision: ProductIdea['decision'], correction?: string) => {
    if (!this.revision || this.busy || this.blocked) return;
    this.revision = {
      ...this.revision,
      ideas: this.revision.ideas.map((idea) => (idea.id === id ? this.decidedIdea(idea, decision, correction) : idea)),
    };
    this.dirty = true;
  };
  private correctProposal = (text: string) => {
    const correction = text.trim().slice(0, 4000);
    if (!correction || !this.revision || this.busy || this.blocked) return;
    const idea: ProductIdea = {
      id: newDecisionID(),
      description: correction,
      consequence: 'The proposal is updated to match the requested change.',
      assumption: false,
      decision: 'edit',
      correction,
      source: 'requested',
      ...(this.actorValid() ? {provenance: 'human' as const, actor: this.actor} : {}),
    };
    this.revision = {...this.revision, ideas: [...this.revision.ideas, idea]};
    this.revise();
  };
  private toggleChange = (open: boolean) => {
    this.changeOpen = open;
    if (open)
      void this.updateComplete.then(() => {
        this.shadowRoot?.querySelector<HTMLElement>('.change-field textarea')?.focus();
      });
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
    }).then((ok) => {
      if (ok) this.toggleDepartment(false);
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
  private saveNote(): string {
    if (!this.objective.trim() && !this.dirty) return '';
    if (this.revision && !this.dirty && this.objective.trim() === this.revision.objective) return 'Saved';
    return 'Unsaved changes';
  }
  private view(): IntentView {
    return {
      objective: this.objective,
      saveNote: this.saveNote(),
      savedRequests: this.savedRequests,
      savedLoaded: this.savedLoaded,
      savedError: this.savedError,
      savedMore: this.savedLoaded && this.savedNext !== undefined,
      department: this.department,
      departments: this.departments,
      revision: this.revision,
      exploration: this.exploration,
      explorationStale: this.explorationStale(),
      changeOpen: this.changeOpen,
      changeText: this.changeText,
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
        if ((field === 'objective' || field === 'department') && this.revision) this.dirty = true;
      },
      setBound: (field, value) => {
        this[field] = value;
        if (this.revision) this.dirty = true;
      },
      decide: this.decide,
      toggleChange: this.toggleChange,
      correctProposal: this.correctProposal,
      chooseApproach: (approach) => {
        if (!this.revision) return;
        this.revision = {...this.revision, approach};
        this.dirty = true;
      },
      describe: this.describe,
      saveDraft: this.saveDraft,
      saveAndLeave: this.saveAndLeave,
      loadSaved: () => this.fetchSaved(false),
      moreSaved: () => this.fetchSaved(true),
      resumeRequest: (intent: string) => {
        if (intent !== this.revision?.intent) this.loadedRequest = '';
        this.request = intent;
        this.resume();
      },
      revise: this.revise,
      decline: this.decline,
      continue: this.continue,
      dialog: this.toggleDepartment,
      dialogKey: this.dialogKey,
      createDepartment: this.createDepartment,
    };
  }
  private reviewing() {
    const state = this.revision?.state;
    return !!this.revision?.ideas.length && (state === 'draft' || state === 'accepted');
  }
  protected render() {
    const view = this.view();
    return html`<div .inert=${this.departmentDialog}>
        ${this.reviewing() ? reviewIntent(view) : describeIntent(view)}
      </div>
      <p role="status">${this.busy ? this.busyNote : this.error}</p>
      ${departmentDialog(view)}`;
  }
}
customElements.define('soda-intent', SodaIntent);

function readExploration(response: unknown, proposal: IntentRevision): Exploration | undefined {
  const raw = object(response);
  if (raw.exploration === undefined) return undefined;
  try {
    const exploration = parseExploration(raw.exploration);
    if (exploration.content.intent !== undefined && exploration.content.intent !== proposal.intent) return undefined;
    return exploration;
  } catch {
    return undefined;
  }
}

function generationMessage(state: unknown): string {
  if (state === 'completed') return 'Factory work is created. Execution still needs its own access and finite funding.';
  if (state === 'pending' || state === 'committed')
    return 'Your product direction is saved. Factory is waiting for confirmation of the native work.';
  check(state === 'refused');
  return 'Your product direction is saved. Factory could not create the native work; an administrator must inspect the refusal.';
}
function generationFailure(error: unknown): string {
  if (error instanceof SodaRequestError && error.code === 'native_graph_unavailable')
    return 'Your product direction is saved. An administrator must enable native work creation before Factory can proceed.';
  return 'Your product direction is saved. Native work creation could not be confirmed. Factory will inspect the original request before retrying.';
}

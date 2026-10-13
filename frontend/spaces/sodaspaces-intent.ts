import {LitElement, html, css} from 'lit';
import type {PropertyValues} from 'lit';
import {object, check, id, SodaRequestError} from './sodaspaces-api.js';
import {
  decisionID,
  department,
  departments,
  exploration as parseExploration,
  intentGeneration,
  intentList,
  intentRevision,
  newDecisionID,
  planningAllowance,
  savedReviewRead,
} from './sodaspaces-intent-response.js';
import type {
  Department,
  Exploration,
  IntentGenerationStatus,
  IntentRevision,
  PlanningAllowance,
  ProductIdea,
  SavedRequest,
  SavedReviewRead,
} from './sodaspaces-intent-response.js';
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
    generation: {state: true},
    generationError: {state: true},
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
      border: 1px solid var(--soda-page-edge);
      background: var(--soda-page-surface);
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
      background: var(--soda-button-primary-bg);
      color: var(--soda-button-primary-text);
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
      border-block-end: 1px solid var(--soda-page-edge);
      padding-block: 1rem;
    }
    .consequence {
      color: var(--soda-page-muted);
    }
    .department-dialog {
      box-sizing: border-box;
      position: fixed;
      z-index: 40;
      inset: 15% max(1rem, calc((100vw - 600px)/2)) auto;
      padding: 1.5rem;
      background: var(--soda-page-surface);
      border: 1px solid var(--soda-page-edge);
      box-shadow: var(--soda-elevation-dialog);
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
  declare generation: IntentGenerationStatus | undefined;
  declare generationError: string;
  declare maxIssues: number;
  declare planningCalls: number;
  private operation: AbortController | undefined;
  private resumeController: AbortController | undefined;
  private restoredReview: RestoredReviewIdentity | undefined;
  private resumeID = '';
  private loadedRequest = '';
  private generationIntent = '';
  private allowance: PlanningAllowance | undefined;
  private userSetCalls = false;
  private raiseCalls = 0;
  private fundingNote = '';
  private savedSnapshot: IntentRevision | undefined;
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
    this.generation = undefined;
    this.generationError = '';
    this.savedSnapshot = undefined;
    this.maxIssues = 20;
    this.planningCalls = 8;
  }
  protected willUpdate(changed: PropertyValues<this>) {
    if (changed.has('actor')) {
      this.exploration = undefined;
      this.restoredReview = undefined;
      // Abort an in-flight resume, but do not clear its controller: the
      // settling resume still owns busy cleanup. Mark the request handled
      // so the busy cascade does not refetch the old request under the
      // new actor; navigation still resumes.
      if (this.resumeController) {
        this.resumeController.abort();
        this.loadedRequest = this.request;
      }
    }
  }
  protected updated(changed: PropertyValues<this>) {
    if (changed.has('active') || changed.has('blocked') || (changed.has('busy') && !this.busy)) this.activate();
    if (this.resumeReady(changed)) this.resume();
    if (this.generationReady(changed)) this.fetchGeneration();
  }
  private generationReady(changed: PropertyValues<this>) {
    return (
      changed.has('revision') || changed.has('api') || changed.has('active') || (changed.has('busy') && !this.busy)
    );
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
    this.allowance = undefined;
    this.userSetCalls = false;
    this.raiseCalls = 0;
    this.fundingNote = '';
    this.revision = undefined;
    this.savedSnapshot = undefined;
    this.exploration = undefined;
    this.restoredReview = undefined;
    this.generation = undefined;
    this.generationError = '';
    this.generationIntent = '';
    this.objective = '';
    this.dirty = false;
    this.error = '';
    this.changeOpen = false;
    this.changeText = '';
  }
  focusReview() {
    void this.updateComplete.then(() => {
      const root = this.shadowRoot;
      if (!root) return;
      const target =
        root.querySelector<HTMLElement>('section.intent-review h2') ||
        root.querySelector<HTMLElement>('section.intent-entry h2');
      target?.focus();
    });
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
      if (this.operation === request) {
        this.busy = false;
        this.dispatchEvent(new CustomEvent('soda-intent-operation', {bubbles: true, composed: true}));
      }
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
        this.dispatchEvent(
          new CustomEvent('intent-inventory', {
            detail: {intents: [...this.savedRequests]},
            bubbles: true,
            composed: true,
          })
        );
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
  private resumeCurrent(id: string, controller: AbortController): boolean {
    return !controller.signal.aborted && this.resumeController === controller && this.request === id;
  }
  private async loadRequest(id: string, controller: AbortController) {
    try {
      const saved = intentRevision(await this.call('/api/factory/intents/' + id, undefined, controller.signal));
      if (!this.resumeCurrent(id, controller)) return;
      if (saved.intent !== id) throw Error('Mismatched intent revision');
      const read = await this.loadReview(id, controller.signal);
      if (!this.resumeCurrent(id, controller)) return;
      if (read && (read.intent !== id || read.head_revision !== saved.id)) throw Error('Mismatched review snapshot');
      this.applyRevision(saved, read);
      this.loadedRequest = id;
      this.dispatchEvent(
        new CustomEvent('intent-resumed', {
          detail: {intent: saved.intent, revision: saved.id, state: saved.state, repository: saved.repository},
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
  private generationAdmitted(revision: IntentRevision, refresh: boolean): boolean {
    if (!this.api || !this.active || this.blocked) return false;
    if (!refresh && (this.generationIntent === revision.id || this.generationError)) return false;
    return true;
  }
  private fetchGeneration(refresh = false) {
    const revision = this.revision;
    if (!revision || revision.state !== 'accepted' || !this.generationAdmitted(revision, refresh)) return;
    if (refresh) {
      this.generationIntent = '';
      this.generationError = '';
    }
    void this.loadGeneration(revision, this.error);
  }
  private loadGeneration(revision: IntentRevision, prior: string) {
    const id = revision.id;
    void this.perform(async (signal) => {
      try {
        const status = intentGeneration(
          await this.call('/api/factory/intents/' + revision.intent + '/generation', undefined, signal)
        );
        if (this.revision?.id !== id) return;
        this.generation = status;
        this.generationIntent = id;
        this.generationError = '';
      } catch {
        if (!signal.aborted)
          this.generationError = 'Factory status could not be loaded. Your direction is saved; try again.';
      }
    }, 'Loading Factory status…').then(() => {
      if (this.error === '') this.error = prior;
    });
  }
  private resumeFailed(id: string, controller: AbortController, error: unknown) {
    if (controller.signal.aborted || this.resumeController !== controller) return;
    this.loadedRequest = id;
    if (error instanceof SodaRequestError && error.status === 404)
      this.error = 'This saved request is unavailable to the current account. Your current edits are kept.';
    else this.error = 'Could not load the saved request. Your current edits are kept; try again.';
  }
  private async loadReview(id: string, signal: AbortSignal): Promise<SavedReviewRead | undefined> {
    try {
      return savedReviewRead(await this.call('/api/factory/intents/' + id + '/review', undefined, signal));
    } catch (error) {
      // Only an absent review degrades to no restored review. Network,
      // parse and all other failures rethrow so the resume fails with
      // unsaved edits kept instead of applying a bare revision.
      if (error instanceof SodaRequestError && error.status === 404) return undefined;
      throw error;
    }
  }
  private applyRevision(saved: IntentRevision, read?: SavedReviewRead) {
    if (saved.intent !== this.revision?.intent) {
      this.allowance = undefined;
      this.userSetCalls = false;
      this.raiseCalls = 0;
      this.fundingNote = '';
    }
    this.revision = saved;
    this.savedSnapshot = structuredClone(saved);
    const adopted = adoptReview(read);
    this.exploration = adopted.exploration;
    this.restoredReview = adopted.restored;
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
    if (this.restoredReview) return restoredStale(this.dirty, this.restoredReview.result, revision.id);
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
    this.savedSnapshot = structuredClone(saved);
    this.savedLoaded = false;
    this.savedError = '';
    return saved;
  }
  private async readAllowance(intent: string, signal: AbortSignal): Promise<PlanningAllowance | undefined> {
    try {
      const found = planningAllowance(
        await this.call('/api/factory/intents/' + intent + '/allowance', undefined, signal),
        intent
      );
      this.allowance = found;
      return found;
    } catch (error) {
      if (error instanceof SodaRequestError && error.status === 404) return undefined;
      throw error;
    }
  }
  private allowanceBody(intent: string, wanted: number, current: PlanningAllowance | undefined) {
    return {
      id: newDecisionID(),
      intent,
      ...(current ? {predecessor: current.id, reserved: current.reserved, consumed: current.consumed} : {}),
      amount: wanted,
      ceiling: wanted,
      unit: 'planning_calls',
      ...(current && wanted > current.amount
        ? {effect: `Authorize ${wanted - current.amount} additional planning calls (${wanted} total).`}
        : {}),
    };
  }
  private async postAllowance(
    intent: string,
    wanted: number,
    current: PlanningAllowance | undefined,
    signal: AbortSignal
  ) {
    const saved = planningAllowance(
      await this.call('/api/factory/intent-allowances', this.allowanceBody(intent, wanted, current), signal),
      intent
    );
    this.allowance = saved;
  }
  private async postAllowanceFresh(intent: string, wanted: number, signal: AbortSignal): Promise<boolean> {
    const fresh = await this.readAllowance(intent, signal);
    if (signal.aborted) return false;
    if (fresh && fresh.amount === wanted) return true;
    await this.postAllowance(intent, wanted, fresh, signal);
    return true;
  }
  private allowanceMessage(error: unknown): string {
    const status = error instanceof SodaRequestError ? error.status : 0;
    const code = error instanceof SodaRequestError ? error.code : '';
    if (status === 404)
      return 'The planning allowance is unavailable to the current account. Your description is saved; try again.';
    if (status === 409 || status === 412)
      return 'The planning allowance changed while saving. Your description is saved; try again.';
    if (code === 'invalid_allowance')
      return 'The planning allowance was not accepted. Your description is saved; check the amount and try again.';
    return 'The planning allowance could not be confirmed. Your description is saved; try again.';
  }
  private allowanceFailed(error: unknown, signal: AbortSignal) {
    if (signal.aborted) return;
    this.error = this.allowanceMessage(error);
  }
  private adoptAllowance(current: PlanningAllowance | undefined) {
    if (current && !this.userSetCalls) this.planningCalls = current.amount;
  }
  private async ensureAllowance(intent: string, signal: AbortSignal): Promise<boolean> {
    try {
      const current = await this.readAllowance(intent, signal);
      if (signal.aborted) return false;
      this.adoptAllowance(current);
      const wanted = this.planningCalls;
      if (current && current.amount === wanted) return true;
      if (this.belowFloor(current, wanted)) return false;
      try {
        await this.postAllowance(intent, wanted, current, signal);
        return true;
      } catch (error) {
        if (!(error instanceof SodaRequestError) || error.status !== 412) throw error;
        return await this.postAllowanceFresh(intent, wanted, signal);
      }
    } catch (error) {
      this.allowanceFailed(error, signal);
      return false;
    }
  }
  private belowFloor(current: PlanningAllowance | undefined, wanted: number): boolean {
    if (!current || wanted >= current.reserved + current.consumed) return false;
    this.error = `Planning calls cannot go below ${current.reserved + current.consumed} already used. Your description is saved; raise the amount or try again.`;
    return true;
  }
  private async refreshAllowance(intent: string, signal: AbortSignal) {
    try {
      await this.readAllowance(intent, signal);
    } catch {
      return;
    }
  }
  private allowanceUsed(): number {
    return (this.allowance?.reserved || 0) + (this.allowance?.consumed || 0);
  }
  private async postRaiseAllowance(
    intent: string,
    wanted: number,
    current: PlanningAllowance | undefined,
    signal: AbortSignal
  ): Promise<boolean> {
    try {
      await this.postAllowance(intent, wanted, current, signal);
      return true;
    } catch (error) {
      if (!(error instanceof SodaRequestError) || error.status !== 412) throw error;
      return await this.postAllowanceFresh(intent, wanted, signal);
    }
  }
  private raise = () => {
    const intent = this.revision?.intent;
    if (!intent) return;
    const wanted = this.raiseCalls || this.allowance?.amount || this.planningCalls;
    if (!Number.isInteger(wanted) || wanted < 1 || wanted > 100) {
      this.fundingNote = 'Enter a raise amount between 1 and 100 planning calls.';
      return;
    }
    void this.perform(async (signal) => {
      const current = await this.readAllowance(intent, signal);
      if (signal.aborted) return;
      this.fundingNote = '';
      if (current && current.amount === wanted) {
        this.fundingNote = `Planning allowance is unchanged at ${wanted} calls.`;
        return;
      }
      if (this.belowFloor(current, wanted)) return;
      try {
        if (!(await this.postRaiseAllowance(intent, wanted, current, signal))) return;
      } catch (error) {
        this.allowanceFailed(error, signal);
        return;
      }
      this.planningCalls = wanted;
      this.userSetCalls = true;
      this.raiseCalls = 0;
      const used = this.allowanceUsed();
      this.fundingNote = `Planning allowance is now ${wanted} calls (${used} used). Use Update proposal or Refresh walkthrough to try again.`;
    }, 'Raising allowance…');
  };
  private async explore(signal: AbortSignal) {
    const saved = await this.save('draft', '0', signal);
    if (!(await this.ensureAllowance(saved.intent, signal))) return;
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
      this.restoredReview = this.exploration ? {basis: saved.id, result: proposal.id} : undefined;
      this.revision = proposal;
      this.savedSnapshot = structuredClone(proposal);
      this.changeOpen = false;
      this.changeText = '';
      this.dirty = false;
    } catch (error) {
      if (!signal.aborted) this.exploreFailed(error);
    }
    this.fundingNote = '';
    if (!signal.aborted) await this.refreshAllowance(saved.intent, signal);
  }
  private exploreFailed(error: unknown) {
    if (error instanceof SodaRequestError && error.code === 'planning_unavailable')
      this.error =
        'The TypeSafe System One judgment service could not complete exploration. Your description is saved; try again.';
    else if (error instanceof SodaRequestError && error.code === 'planning_funding_required')
      this.error =
        'Exploration needs a planning allowance. Your description is saved; raise the planning allowance and try again.';
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
      this.savedSnapshot = undefined;
      this.exploration = undefined;
      this.restoredReview = undefined;
      this.generation = undefined;
      this.generationError = '';
      this.generationIntent = '';
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
    const revision = this.revision;
    if (!revision || !reviewReady(this.view())) return;
    const authorized = revision.ideas.filter((idea) => idea.decision === 'accept' || idea.decision === 'edit');
    this.dispatchEvent(
      new CustomEvent('intent-reviewed', {
        detail: {
          intent: revision.intent,
          revision: revision.id,
          approach: revision.approach,
          authorizedIdeas: authorized.length,
          tasks: revision.tasks.length,
          decisions: authorized.map((idea) => ({idea: idea.id, decision: idea.decision})),
        },
        bubbles: true,
        composed: true,
      })
    );
  };
  private actorValid(): boolean {
    return this.actor !== '' && id(this.actor);
  }
  private settledIDs(): Set<string> {
    return new Set(this.exploration?.settled || []);
  }
  private decidedIdea(idea: ProductIdea, decision: ProductIdea['decision'], correction?: string): ProductIdea {
    if (decision === 'edit' && !(correction ?? '').trim() && this.settledIDs().has(idea.id)) {
      const restored: ProductIdea = {...idea, decision: 'accept', provenance: 'autonomous'};
      delete restored.correction;
      delete restored.actor;
      return restored;
    }
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
  private ideaChanged(idea: ProductIdea): boolean {
    const saved = this.savedSnapshot?.ideas.find((entry) => entry.id === idea.id);
    return !saved || saved.decision !== idea.decision || (saved.correction || '') !== (idea.correction || '');
  }
  private unsavedScalars(snapshot: IntentRevision): {count: number; parts: string[]} {
    const parts: string[] = [];
    let count = 0;
    if (this.objective.trim() !== snapshot.objective) {
      count += 1;
      parts.push('outcome text');
    }
    if ((this.department || '') !== (snapshot.department || '')) {
      count += 1;
      parts.push('department');
    }
    if (this.maxIssues !== snapshot.max_issues || this.planningCalls !== snapshot.planning_calls) {
      count += 1;
      parts.push('planning limits');
    }
    return {count, parts};
  }
  private unsavedSummary(): {count: number; parts: string} {
    const snapshot = this.savedSnapshot;
    const revision = this.revision;
    if (!snapshot || !revision || !this.dirty) return {count: 0, parts: ''};
    const scalars = this.unsavedScalars(snapshot);
    const decided = revision.ideas.filter((idea) => this.ideaChanged(idea)).length;
    if (decided) {
      scalars.count += decided;
      scalars.parts.push(`${decided} idea decision${decided === 1 ? '' : 's'}`);
    }
    return {count: scalars.count, parts: scalars.parts.join(', ')};
  }
  private view(): IntentView {
    const unsaved = this.unsavedSummary();
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
      generation: this.generation,
      generationError: this.generationError,
      unsavedCount: unsaved.count,
      unsavedParts: unsaved.parts,
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
      allowance: this.allowance,
      raiseCalls: this.raiseCalls || this.allowance?.amount || this.planningCalls,
      fundingNote: this.fundingNote,
      setRaise: (value: number) => {
        this.raiseCalls = value;
        this.fundingNote = '';
      },
      raiseAllowance: this.raise,
      set: (field, value) => {
        this[field] = value;
        if ((field === 'objective' || field === 'department') && this.revision) this.dirty = true;
      },
      setBound: (field, value) => {
        this[field] = value;
        if (field === 'planningCalls') this.userSetCalls = true;
        if (this.revision) this.dirty = true;
      },
      decide: this.decide,
      toggleChange: this.toggleChange,
      correctProposal: this.correctProposal,
      refreshGeneration: () => this.fetchGeneration(true),
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

export interface RestoredReviewIdentity {
  basis: string;
  result: string;
}
export function adoptReview(read: SavedReviewRead | undefined): {
  exploration: Exploration | undefined;
  restored: RestoredReviewIdentity | undefined;
} {
  const review = read?.review;
  if (!review) return {exploration: undefined, restored: undefined};
  const {content, challenges, settled, recommendation, consistency} = review;
  return {
    exploration: {
      content,
      challenges,
      settled,
      ...(recommendation ? {recommendation} : {}),
      ...(consistency ? {consistency} : {}),
    },
    restored: {basis: review.basis_revision, result: review.result_revision},
  };
}
// restoredStale reports whether a basis/result-tracked exploration is out of
// date: local edits always stale it, otherwise the stored result must still
// be the head. Allowance and other non-head changes never stale it.
export function restoredStale(dirty: boolean, result: string, revisionId: string): boolean {
  if (dirty) return true;
  return result !== revisionId;
}

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
  if (error instanceof SodaRequestError && error.code === 'intent_review_required')
    return 'Your product direction is saved. The proposed work needs reconsideration before native work can be created.';
  return 'Your product direction is saved. Native work creation could not be confirmed. Factory will inspect the original request before retrying.';
}

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
  intentPlanStatus,
  intentRevision,
  planActionBody,
  planWait,
  generationWaitNotice,
  generationFailureNotice,
  planningFailedNotice,
  planningCancelledNotice,
  runPlanWork,
  newDecisionID,
  planningAllowance,
  savedReviewRead,
} from './sodaspaces-intent-response.js';
import type {
  Department,
  Exploration,
  IntentGenerationStatus,
  IntentPlanStatus,
  IntentRevision,
  IntentTask,
  PlanningAllowance,
  ProductIdea,
  SavedRequest,
  SavedReviewRead,
} from './sodaspaces-intent-response.js';
import {describeIntent, reviewIntent, departmentDialog, reviewReady} from './sodaspaces-intent-view.js';
import type {IntentView} from './sodaspaces-intent-view.js';

type IntentAPI = (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => Promise<unknown>;
// Repair record for dependency edges cut by a decline save: the task's
// pre-prune depends_on plus the idea links identifying the work across an
// explore regeneration. depIdeas pins the stable lineage of each recorded
// target (target key to its idea links at record time) so a later proposal
// that regenerates work under fresh keys can remap the edge onto the
// successor instead of dropping it; targets that stay shelved keep stable
// keys and need no lineage entry.
interface ShelvedDepRepair {
  deps: string[];
  ideas: string[];
  depIdeas?: Record<string, string[]>;
}
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
    generationNotice: {state: true},
    maxIssues: {state: true},
    planningCalls: {state: true},
    planOperation: {state: true},
    plan: {state: true},
    planNotice: {state: true},
    planIntent: {state: true},
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
    .intent-review,
    .intent-entry {
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
    [role='status'].intent-live-status {
      margin: 0;
    }
    .intent-live-status > span {
      display: block;
      margin-block: 1rem;
    }
    .scope-gate {
      border: 1px solid var(--soda-page-edge);
      padding: 1rem;
      margin-block: 1rem;
    }
    :is(button, input, select, textarea, summary, a):focus-visible {
      outline: 2px solid var(--soda-page-focus);
      outline-offset: 2px;
    }
    h2[tabindex='-1']:focus,
    h3[tabindex='-1']:focus {
      outline: none;
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
  declare planOperation: string;
  declare plan: IntentPlanStatus | undefined;
  declare planNotice: string;
  declare planIntent: string;
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
  private allowanceError = '';
  private savedSnapshot: IntentRevision | undefined;
  private scopeGateArmed = false;
  // Work shelved when a decision edit declined a linked idea: the dropped
  // tasks are kept here (never re-decided) so re-accepting the idea restores
  // them on the next save, while leaving them declined keeps them out.
  // pendingDropped carries the current save's drops from currentProposal to
  // save. Both clear with the revision; a server-replaced proposal
  // reconciles the shelf instead (drops whose ideas are gone are forgotten).
  // shelvedPrunedDeps remembers the exact pre-prune dependency edges of kept
  // tasks whose links were cut by a decline save (task key to its original
  // depends_on plus the idea links identifying the work), so re-accepting
  // the declined idea re-attaches only those recorded edges that point at
  // restored work. Shelved tasks whose own edges point at kept work that a
  // later explore may re-key carry a lineage entry here too, so the edge
  // follows the successor instead of being lost. Nothing here invents an
  // edge: repair replays recorded originals against the final live set, and
  // a target whose lineage is ambiguous stays cut. The idea links let the
  // record follow its work across an explore regeneration (fresh keys, same
  // ideas) only onto a unique exact-idea-identical successor; an ambiguous
  // split transfers to none and surfaces explicit review instead of guessing.
  private shelvedTasks: IntentTask[] = [];
  private shelvedPrunedDeps: Record<string, ShelvedDepRepair> = {};
  private pendingDropped: IntentTask[] = [];
  // pendingRestored carries the current save's shelf restorations from
  // currentProposal to save: tasks absent from the prior revision whose
  // exact pre-prune edges must still be recorded when re-accepting prunes
  // them again before their targets return.
  private pendingRestored: IntentTask[] = [];
  // Cross-device saved-history recovery (saved bytes only): the local shelf
  // above is same-browser localStorage, so a resume on another device starts
  // with no shelf. Recovery then reads ONLY the owner-private history
  // endpoint (bare GET .../history, default page): the exact predecessor
  // named by the current head, or a bounded exact walk back along the
  // predecessor chain. Unsent edits are never in history and stay
  // local-only; in-memory edges never existed in a predecessor and are never
  // treated as history. Entries stage per declined idea and only a user
  // re-accept moves them into the shelf for the next save; nothing is
  // unioned and no decision is ever changed silently.
  private savedHistory: SavedHistoryEntry[] = [];
  private savedHistoryNote = '';
  private savedHistorySkipped = 0;
  private savedHistoryHead = '';
  private departmentsLoaded = false;
  private progressTimer?: ReturnType<typeof setInterval>;
  private progressReading = false;
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
    this.planOperation = 'propose_intent';
    this.plan = undefined;
    this.planNotice = '';
    this.planIntent = '';
  }
  protected willUpdate(changed: PropertyValues<this>) {
    if (changed.has('actor')) {
      this.operation?.abort();
      this.resumeController?.abort();
      this.operation = this.resumeController = undefined;
      this.busy = false;
      this.loadedRequest = '';
      this.revision = this.savedSnapshot = undefined;
      this.savedRequests = [];
      this.savedLoaded = this.departmentsLoaded = false;
      this.savedError = this.error = '';
      this.departments = [];
      this.allowance = undefined;
      this.allowanceError = '';
      this.userSetCalls = false;
      this.raiseCalls = 0;
      this.fundingNote = '';
      this.dirty = this.changeOpen = this.departmentDialog = false;
      this.scopeGateArmed = false;
      this.shelvedTasks = [];
      this.shelvedPrunedDeps = {};
      this.pendingDropped = [];
      this.pendingRestored = [];
      this.savedHistory = [];
      this.savedHistoryNote = '';
      this.savedHistorySkipped = 0;
      this.savedHistoryHead = '';
      this.plan = this.generation = undefined;
      this.planIntent = this.generationIntent = '';
      this.planNotice = this.generationNotice = this.generationError = '';
      this.objective = '';
      this.department = '';
      this.changeText = '';
      this.restoreDraft();
      this.exploration = undefined;
      this.restoredReview = undefined;
    }
  }
  connectedCallback() {
    super.connectedCallback();
    this.progressTimer = setInterval(() => void this.refreshProgress(), 10000);
  }
  private async refreshProgress() {
    const revision = this.revision,
      actor = this.actor;
    if (
      !revision ||
      !this.active ||
      this.blocked ||
      this.busy ||
      this.progressReading ||
      document.visibilityState !== 'visible'
    )
      return;
    const accepted = revision.state === 'accepted';
    if (!accepted && (revision.state !== 'draft' || revision.repository === '0')) return;
    this.progressReading = true;
    try {
      const value = await this.call(
        '/api/factory/intents/' + revision.intent + (accepted ? '/generation' : '/plan'),
        undefined,
        AbortSignal.timeout(15000)
      );
      if (this.actor !== actor || this.revision?.id !== revision.id || !this.active) return;
      if (accepted) {
        this.generation = intentGeneration(value);
        this.generationError = '';
      } else {
        this.applyPolledPlanStatus(intentPlanStatus(value), revision.id);
      }
    } catch {
      if (this.actor === actor && this.revision?.id === revision.id) {
        if (accepted)
          this.generationError = 'Live updates interrupted. Your saved work is kept; retrying automatically.';
        else if (this.planNotice === '')
          this.planNotice = 'Live updates interrupted. Your saved work is kept; retrying automatically.';
      }
    } finally {
      this.progressReading = false;
    }
  }
  // One polled-status contract for the timer and explicit refresh: a poll
  // never erases the guidance behind the current draft. Terminal states
  // always restate their guidance (a stale running text must not survive
  // a settled run); live states update guidance only on transition; a
  // missing plan keeps whatever the last explicit action reported.
  private applyPolledPlanStatus(status: IntentPlanStatus, id: string) {
    if (status.revision !== id || this.revision?.id !== id) return;
    const previous = this.plan?.revision === id ? this.plan.state : undefined;
    this.plan = status;
    // A repeated state never erases the current guidance, so a 202 wait or
    // an explicit error survives polls. A genuine transition restates it,
    // so stale waits/failures cannot survive work starting or finishing.
    if (status.state === 'none' || previous === status.state) return;
    if (status.state === 'failed' || status.state === 'cancelled') {
      this.planNotice = status.state === 'failed' ? planningFailedNotice : planningCancelledNotice;
    } else if (status.state === 'completed') this.planNotice = '';
    else if (status.state === 'pending')
      this.planNotice = 'Your request is saved. Its planning progress and any question appear here.';
    else if (status.state === 'attention')
      this.planNotice = status.attention
        ? `Planning needs a product decision: ${status.attention}`
        : 'Planning needs a product decision before issues can be proposed.';
  }
  private draftStorageKey() {
    // Key by intent once a revision exists: the request id is only the
    // resume address, but the shelf and decisions belong to the intent, so
    // save→reload→resume must read the same entry the save wrote. Same
    // browser only — never a cross-device claim.
    return `soda-intent-draft:v1:${this.actor}:${this.revision?.intent || this.request || 'new'}`;
  }
  private rememberDraft() {
    if (!this.actor) return;
    try {
      localStorage.setItem(
        this.draftStorageKey(),
        JSON.stringify({
          base: this.revision?.id ?? '',
          objective: this.objective,
          department: this.department,
          maxIssues: this.maxIssues,
          planningCalls: this.planningCalls,
          changeText: this.changeText,
          ideas: this.revision?.ideas.map(({id, decision, correction}) => ({id, decision, correction})),
          approach: this.revision?.approach,
          // The exact restore payload: shelved originals plus the recorded
          // pre-prune edges, so reload offers the same Re-accept recovery
          // with dependencies intact. Same-browser local state only
          // (actor/request key): never a cross-device recovery claim.
          shelf: this.shelvedTasks,
          shelfDeps: this.shelvedPrunedDeps,
        })
      );
    } catch {
      /* Keep in-memory writing when storage is unavailable. */
    }
  }
  private restoreDraft() {
    if (!this.actor) return;
    try {
      const draft = JSON.parse(localStorage.getItem(this.draftStorageKey()) ?? 'null');
      if (!draft || draft.base !== (this.revision?.id ?? '') || typeof draft.objective !== 'string') return;
      this.objective = draft.objective;
      if (typeof draft.department === 'string') this.department = draft.department;
      if (Number.isInteger(draft.maxIssues) && draft.maxIssues >= 1 && draft.maxIssues <= 64)
        this.maxIssues = draft.maxIssues;
      if (Number.isInteger(draft.planningCalls) && draft.planningCalls >= 1 && draft.planningCalls <= 100)
        this.planningCalls = draft.planningCalls;
      if (typeof draft.changeText === 'string') this.changeText = draft.changeText;
      if (this.revision && Array.isArray(draft.ideas)) {
        this.revision = {
          ...this.revision,
          ideas: this.revision.ideas.map((idea) => {
            const saved = draft.ideas.find(
              (row: unknown) => !!row && typeof row === 'object' && (row as {id?: unknown}).id === idea.id
            );
            if (!saved || !['pending', 'accept', 'decline', 'edit'].includes(saved.decision)) return idea;
            if (saved.decision === idea.decision && saved.correction === idea.correction) return idea;
            return this.decidedIdea(
              idea,
              saved.decision,
              typeof saved.correction === 'string' ? saved.correction.slice(0, 4000) : undefined
            );
          }),
        };
        if (draft.approach === 'complete_first' || draft.approach === 'incremental')
          this.revision = {...this.revision, approach: draft.approach};
      }
      // Restore the exact shelf payload behind the same base gate: entries
      // and edges that fail validation are dropped, never half-applied, and
      // an unreadable shelf never clears the in-memory one.
      if (Array.isArray(draft.shelf) && draft.shelf.every(isShelvedTask)) {
        const ideas = new Set((this.revision?.ideas || []).map((idea) => idea.id));
        this.shelvedTasks = draft.shelf.filter(
          (task: IntentTask) => task.idea_ids.length > 0 && task.idea_ids.every((id: string) => ideas.has(id))
        );
        this.shelvedPrunedDeps = isShelvedDeps(draft.shelfDeps) ? draft.shelfDeps : {};
        this.requestUpdate();
      }
      this.dirty =
        !!this.revision &&
        (this.objective !== this.revision.objective ||
          this.department !== (this.revision.department || '') ||
          this.maxIssues !== this.revision.max_issues ||
          this.planningCalls !== this.revision.planning_calls ||
          JSON.stringify(this.revision.ideas) !== JSON.stringify(this.savedSnapshot?.ideas) ||
          this.revision.approach !== this.savedSnapshot?.approach);
    } catch {
      /* An unreadable local draft never blocks saved work. */
    }
  }
  protected updated(changed: PropertyValues<this>) {
    if (changed.has('active') || changed.has('blocked') || (changed.has('busy') && !this.busy)) this.activate();
    if (this.resumeReady(changed)) this.resume();
    if (this.generationReady(changed)) this.fetchGeneration();
    if (this.planReady(changed)) this.fetchPlanStatus();
  }
  private planReady(changed: PropertyValues<this>) {
    return (
      changed.has('revision') || changed.has('api') || changed.has('active') || (changed.has('busy') && !this.busy)
    );
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
    clearInterval(this.progressTimer);
    this.operation?.abort();
    this.resumeController?.abort();
    super.disconnectedCallback();
  }
  reset() {
    // Persist the recovery record BEFORE clearing: reset starts a new
    // request but the saved revision still exists, so resuming it in this
    // browser must still offer the same Re-accept recovery. Shelf recovery
    // is same-browser local state only (actor/request key, this browser) —
    // never cross-device: the history endpoint retains the saved graph but
    // the UI restores the shelf from local state alone.
    this.rememberDraft();
    this.scopeGateArmed = false;
    this.shelvedTasks = [];
    this.shelvedPrunedDeps = {};
    this.pendingDropped = [];
    this.pendingRestored = [];
    this.savedHistory = [];
    this.savedHistoryNote = '';
    this.savedHistorySkipped = 0;
    this.savedHistoryHead = '';
    this.operation?.abort();
    this.resumeController?.abort();
    this.request = '';
    this.loadedRequest = '';
    this.allowance = undefined;
    this.allowanceError = '';
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
    this.plan = undefined;
    this.planIntent = this.planNotice = '';
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
  focusFindings() {
    void this.updateComplete.then(() => {
      this.shadowRoot
        ?.querySelector<HTMLElement>('section[aria-label="Consistency check"] h3')
        ?.focus({preventScroll: false});
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
      // A refused save sets its own explicit guidance before throwing; keep
      // it so the user knows what review is required.
      if (!request.signal.aborted && !this.error)
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
      if (!this.resumeCurrent(id, controller)) return;
      await this.refreshAllowance(saved.intent, controller.signal);
      if (!this.resumeCurrent(id, controller)) return;
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
      this.allowanceError = '';
      this.userSetCalls = false;
      this.raiseCalls = 0;
      this.fundingNote = '';
    }
    this.revision = saved;
    this.savedSnapshot = structuredClone(saved);
    this.scopeGateArmed = false;
    this.reconcileShelvedTasks(saved);
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
    this.restoreDraft();
    this.maybeRecoverFromHistory();
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
    this.pendingDropped = [];
    this.pendingRestored = [];
    if (this.terminalRevision()) return {ideas: [], tasks: []};
    if (this.objective.trim() !== this.revision?.objective) return {ideas: [], tasks: []};
    const ideas = this.revision?.ideas || [];
    const effective = this.revision?.effective_outcomes?.filter((id) =>
      ideas.some((idea) => idea.id === id && (idea.decision === 'accept' || idea.decision === 'edit'))
    );
    const split = this.splitProposalTasks(ideas, this.revision?.tasks || []);
    this.pendingDropped = split.dropped;
    this.pendingRestored = split.restored;
    return {
      ideas,
      tasks: split.tasks,
      ...(effective === undefined ? {} : {effective_outcomes: effective}),
    };
  }
  // Canonical task binding (IntentRevision.Validate rejects declined
  // references in every state): drop tasks linked to declined ideas without
  // re-deciding anything, prune dependency edges left dangling by those
  // drops, and restore shelved tasks whose links are all decided again.
  // Accepted-only subset membership stays out of scope here.
  // Restore-before-prune: the restorable shelved set joins the live set
  // BEFORE any edge is cut, so a kept task depending on re-accepted work
  // keeps its edge and shelved↔shelved edges survive regardless of shelf
  // order. Kept tasks whose edges an earlier decline save cut are repaired
  // first from the recorded pre-prune originals (only edges pointing at
  // live work return); declined work stays declined until re-accepted.
  private splitProposalTasks(
    ideas: ProductIdea[],
    tasks: IntentTask[]
  ): {tasks: IntentTask[]; dropped: IntentTask[]; restored: IntentTask[]} {
    const byID = new Map(ideas.map((idea) => [idea.id, idea]));
    const dropped: IntentTask[] = [];
    const kept = tasks.filter((task) => {
      if (!task.idea_ids.some((id) => byID.get(id)?.decision === 'decline')) return true;
      dropped.push(task);
      return false;
    });
    const restorable = this.shelvedTasks.filter(
      (task) =>
        !kept.some((entry) => entry.key === task.key) &&
        task.idea_ids.length > 0 &&
        task.idea_ids.every((id) => byID.has(id) && byID.get(id)?.decision !== 'decline')
    );
    const live = new Set([...kept.map((task) => task.key), ...restorable.map((task) => task.key)]);
    const repaired = kept.map((task) => this.repairTaskDeps(task, live));
    const pruned = repaired.map((task) => this.pruneTaskDeps(task, live));
    // Shelved originals can themselves be cut (shelved while already
    // pruned by an earlier decline), so they get the same recorded repair.
    // The raw restorable entries travel to the save as pendingRestored: a
    // restored task is absent from the prior revision, so its exact
    // pre-prune edges must be captured from this shape before pruning.
    const restored = restorable.map((task) => this.pruneTaskDeps(this.repairTaskDeps(task, live), live));
    return {tasks: [...pruned, ...restored], dropped, restored: restorable};
  }
  // Re-attach only recorded pre-prune edges that point at live work, in
  // their original order. Edges to still-declined or unknown keys stay cut;
  // no edge is ever invented.
  private repairTaskDeps(task: IntentTask, live: Set<string>): IntentTask {
    const recorded = this.shelvedPrunedDeps[task.key]?.deps;
    if (!recorded || recorded.length === 0) return task;
    const current = new Set(task.depends_on || []);
    // Recorded order is the pre-prune order; stored edges keep their place
    // and only the cut edges return, appended in recorded order.
    const repaired = recorded.filter((dep) => live.has(dep) && !current.has(dep));
    if (!repaired.length) return task;
    return {...task, depends_on: [...(task.depends_on || []), ...repaired]};
  }
  private pruneTaskDeps(task: IntentTask, live: Set<string>): IntentTask {
    const depends = task.depends_on || [];
    const kept = depends.filter((dep) => live.has(dep));
    return kept.length === depends.length ? task : {...task, depends_on: kept};
  }
  // Unresolved repair records: entries whose owner is absent from both the
  // current and shelved sets (an ambiguous split kept the record under the
  // old owner key). Derived, never stored separately: the record itself is
  // the state, and the review below renders while any entry qualifies —
  // across saves, reloads and leaves — until a unique successor claims it
  // or the user confirms the drop. Never silently erased.
  private unresolvedRepairs(): {key: string; repair: ShelvedDepRepair}[] {
    const live = new Set((this.revision?.tasks || []).map((task) => task.key));
    const shelved = new Set(this.shelvedTasks.map((task) => task.key));
    return Object.entries(this.shelvedPrunedDeps)
      .filter(
        ([key, repair]) =>
          !live.has(key) &&
          !shelved.has(key) &&
          (repair.deps.length > 0 || Object.keys(repair.depIdeas || {}).length > 0)
      )
      .map(([key, repair]) => ({key, repair}));
  }
  // Resolved review scope: the explicit narrowed subset when the stored
  // revision records one, else every decided idea in revision order.
  // Pending/declined ideas never resolve in.
  private resolvedScope(revision: IntentRevision): ProductIdea[] {
    const decided = (revision.ideas || []).filter(
      (idea) => idea.decision === 'accept' || idea.decision === 'edit'
    );
    const explicit = revision.effective_outcomes;
    if (explicit && explicit.length > 0) {
      const decidedByID = new Map(decided.map((idea) => [idea.id, idea]));
      const resolved: ProductIdea[] = [];
      for (const id of explicit) {
        const idea = decidedByID.get(id);
        if (idea && !resolved.includes(idea)) resolved.push(idea);
      }
      return resolved;
    }
    return decided;
  }
  // The narrowed-to-empty save guard evaluates the PROPOSED revision, not
  // the stored one: a changed objective or a terminal reset clears the
  // proposal (no ideas, no scope), so an unrelated new request always saves
  // freely. Only a draft/accepted proposal that keeps decided ideas while
  // resolving to an empty explicit set refuses the silent save — the set
  // serializes with omitempty, so saving it would reload as unspecified
  // and downstream readers would derive (and silently authorize) the
  // remaining decided ideas.
  private scopeNarrowsToEmpty(next: IntentRevision): boolean {
    const stored = this.revision?.effective_outcomes;
    if (!stored || stored.length === 0) return false;
    if (next.effective_outcomes === undefined) return false;
    const decided = (next.ideas || []).some(
      (idea) => idea.decision === 'accept' || idea.decision === 'edit'
    );
    return decided && next.effective_outcomes.length === 0;
  }
  // Decided ideas needing an explicit review after a refused narrowing
  // save: subset members that lost their decision (re-accept to keep the
  // narrowed scope) and decided ideas outside the resolved scope (decline
  // to accept the empty scope explicitly). Rendered alongside the refusal
  // because task-unlinked ideas carry no choice control of their own.
  private scopeGateLists(): {removed: ProductIdea[]; remaining: ProductIdea[]} | undefined {
    if (!this.scopeGateArmed) return undefined;
    const revision = this.revision;
    const stored = revision?.effective_outcomes;
    if (!revision || !stored || stored.length === 0) return undefined;
    const byID = new Map(revision.ideas.map((idea) => [idea.id, idea]));
    const decidedIDs = new Set(
      revision.ideas
        .filter((idea) => idea.decision === 'accept' || idea.decision === 'edit')
        .map((idea) => idea.id)
    );
    const resolvedIDs = new Set(this.resolvedScope(revision).map((idea) => idea.id));
    return {
      removed: stored.flatMap((id) => {
        const idea = byID.get(id);
        return idea && !decidedIDs.has(id) ? [idea] : [];
      }),
      remaining: revision.ideas.filter((idea) => decidedIDs.has(idea.id) && !resolvedIDs.has(idea.id)),
    };
  }
  private scopeRecovery() {
    const gate = this.scopeGateLists();
    if (!gate || (gate.removed.length === 0 && gate.remaining.length === 0)) return '';
    return html`<div class="scope-gate" role="group" aria-label="Narrowed scope recovery">
      <p>Decide each listed idea explicitly, then save again.</p>
      ${gate.removed.map(
        (idea) => html`<p>${idea.description}
          <button
            aria-label="Re-accept narrowed idea ${idea.id}"
            ?disabled=${this.busy || this.blocked}
            @click=${() => this.recoverScope(idea.id, 'accept')}
          >
            Re-accept
          </button>
        </p>`
      )}
      ${gate.remaining.map(
        (idea) => html`<p>${idea.description}
          <button
            aria-label="Decline remaining idea ${idea.id}"
            ?disabled=${this.busy || this.blocked}
            @click=${() => this.recoverScope(idea.id, 'decline')}
          >
            Decline
          </button>
        </p>`
      )}
    </div>`;
  }
  private recoverScope = (id: string, decision: 'accept' | 'decline') => {
    if (!this.revision || this.busy || this.blocked) return;
    this.decide(id, decision);
  };
  // Keep the current save's declined-linked drops for recovery: shelved
  // tasks survive so re-accepting an idea restores them, restored keys leave
  // the shelf, and drops whose ideas are gone (a cleared new request) are
  // forgotten. Nothing here re-decides any idea. Kept tasks whose
  // dependency edges this save cut record their pre-prune originals so a
  // later re-accept re-attaches exactly those edges; restored shelf tasks
  // record theirs from the restored shape (they are absent from the prior
  // revision); entries whose edges no longer point at shelved work and
  // carry no pending lineage are forgotten.
  private shelveDroppedTasks(next: IntentRevision) {
    const posted = new Set(next.ideas.map((idea) => idea.id));
    const saved = new Set(next.tasks.map((task) => task.key));
    const kept = this.shelvedTasks.filter(
      (task) => !saved.has(task.key) && task.idea_ids.length > 0 && task.idea_ids.every((id) => posted.has(id))
    );
    for (const task of this.pendingDropped) {
      if (saved.has(task.key) || kept.some((entry) => entry.key === task.key)) continue;
      kept.push(task);
    }
    this.shelvedTasks = kept;
    this.pendingDropped = [];
    const restored = new Map(this.pendingRestored.map((task) => [task.key, task]));
    this.pendingRestored = [];
    this.captureShelvedLineage();
    this.recordPrunedDeps(next, restored);
    this.requestUpdate();
  }
  // Pin the stable lineage of shelved tasks' edges: an explore may
  // regenerate live targets under fresh keys, and shelved targets return
  // under their stable keys but may regenerate after that, so the record
  // carries every known target's idea links to remap the edge onto the
  // successor later. No remap fires while the recorded key still resolves.
  private captureShelvedLineage() {
    const known = new Map<string, string[]>();
    for (const task of this.revision?.tasks || []) known.set(task.key, [...task.idea_ids]);
    for (const task of this.shelvedTasks) known.set(task.key, [...task.idea_ids]);
    for (const task of this.shelvedTasks) {
      const depIdeas = lineageForDeps(
        task.depends_on || [],
        (dep) => known.get(dep),
        this.shelvedPrunedDeps[task.key]?.depIdeas
      );
      if (Object.keys(depIdeas).length === 0) continue;
      const prior = this.shelvedPrunedDeps[task.key];
      const deps = prior ? [...prior.deps] : [...(task.depends_on || [])];
      for (const dep of task.depends_on || []) if (!deps.includes(dep)) deps.push(dep);
      this.shelvedPrunedDeps[task.key] = {deps, ideas: [...task.idea_ids], depIdeas};
    }
    // Records whose owner is repaired/live or unassigned carry dependency
    // keys reconcile remapped onto successors: refresh their target lineage
    // from the current task graph so a later regeneration can follow every
    // hop. Only known keys gain entries; unknown keys keep their carried
    // lineage for a later regeneration. Nothing here invents an edge.
    for (const [key, repair] of Object.entries(this.shelvedPrunedDeps)) {
      if (repair.deps.length === 0) continue;
      const depIdeas = lineageForDeps(repair.deps, (dep) => known.get(dep), repair.depIdeas);
      if (Object.keys(depIdeas).length === 0) continue;
      this.shelvedPrunedDeps[key] = {deps: [...repair.deps], ideas: [...repair.ideas], depIdeas};
    }
  }
  private recordPrunedDeps(next: IntentRevision, restored: Map<string, IntentTask>) {
    const original = new Map((this.revision?.tasks || []).map((task) => [task.key, task]));
    const shelvedByKey = new Map(this.shelvedTasks.map((task) => [task.key, task]));
    const shelved = new Set(shelvedByKey.keys());
    const saved = new Set(next.tasks.map((task) => task.key));
    const posted = new Set(next.ideas.map((idea) => idea.id));
    const recorded: Record<string, ShelvedDepRepair> = {};
    // Carry forward records for work that is still kept or newly shelved:
    // a recorded edge still pointing at shelved work, or a lineage entry
    // whose target has not come back live, must survive later saves — a
    // second decline (or a staggered re-accept) must not forget the first
    // decline's cut edges.
    for (const [key, repair] of Object.entries(this.shelvedPrunedDeps)) {
      if (!saved.has(key) && !shelved.has(key)) {
        // Unresolved owner: the record outlived its work (an ambiguous
        // split kept it under the old owner key, and no unique successor
        // has claimed it yet). Carry it across saves — the post-save draft
        // re-persist keeps it bound to the new head with the shelf — while
        // its lineage still names posted work, so a later regeneration can
        // still resolve it or the user can explicitly confirm the drop.
        // Entries with no edges and no pending lineage carry nothing, and
        // entries whose ideas are gone are forgotten with the shelf. Never
        // silently erased otherwise: the next save (limit-only, leave, or
        // the save before an explore) must not lose the ambiguity.
        if (repair.deps.length === 0 && Object.keys(repair.depIdeas || {}).length === 0) continue;
        if (!repair.ideas.some((id) => posted.has(id))) continue;
        recorded[key] = copyRepair(repair);
        continue;
      }
      // A still-shelved owner keeps its entry while it carries anything:
      // its edges may point at live work a later explore re-keys, and only
      // a restore (or forgotten ideas) ends that. A kept owner keeps its
      // entry while an edge points at shelved work or a lineage target has
      // not come back live.
      if (shelved.has(key)) {
        if (repair.deps.length > 0 || Object.keys(repair.depIdeas || {}).length > 0)
          recorded[key] = copyRepair(repair);
        continue;
      }
      if (
        !repair.deps.some((dep) => shelved.has(dep)) &&
        pendingRepairLineage(repair, saved, shelved).length === 0
      )
        continue;
      recorded[key] = copyRepair(repair);
    }
    for (const task of next.tasks) {
      // A restored shelf task is absent from the prior revision: read its
      // exact pre-prune edges from the restored proposal shape, not from
      // the prior revision (which never carried it).
      const before = original.get(task.key) ?? restored.get(task.key);
      if ((before?.depends_on || []).length === 0 && !recorded[task.key]) continue;
      const initial = before?.depends_on || [];
      const after = new Set(task.depends_on || []);
      const cut = initial.filter((dep) => !after.has(dep));
      // Remember the full pre-prune order only while a cut edge still
      // points at shelved work; a fully restored task needs no record.
      // Accumulate against the exact ORIGINAL edges: a later decline sees
      // an already-pruned `before`, so union it with the carried record —
      // never discard edges recorded for still-declined ideas.
      if (cut.some((dep) => shelved.has(dep))) {
        const prior = recorded[task.key];
        // Pin lineage for every recorded edge with known ideas: a later
        // save or explore may re-key any target, and only pinned lineage
        // lets the remap follow it. While the key still resolves the
        // remap is a no-op, so stable backends see no change.
        const depIdeas = lineageForDeps(
          initial,
          (dep) => (original.get(dep) ?? shelvedByKey.get(dep))?.idea_ids,
          prior?.depIdeas
        );
        if (prior) {
          const merged = [...prior.deps];
          for (const dep of initial) if (!merged.includes(dep)) merged.push(dep);
          const ideas = [...prior.ideas];
          for (const id of task.idea_ids) if (!ideas.includes(id)) ideas.push(id);
          const mergedLineage = {...prior.depIdeas};
          for (const [dep, ids] of Object.entries(depIdeas)) if (!mergedLineage[dep]) mergedLineage[dep] = ids;
          recorded[task.key] = {
            deps: merged,
            ideas,
            ...(Object.keys(mergedLineage).length > 0 ? {depIdeas: mergedLineage} : {}),
          };
        } else {
          recorded[task.key] = {
            deps: [...initial],
            ideas: [...task.idea_ids],
            ...(Object.keys(depIdeas).length > 0 ? {depIdeas} : {}),
          };
        }
      }
    }
    this.shelvedPrunedDeps = recorded;
  }
  // Declined ideas behind shelved work: the only entries offering recovery.
  // Re-accepted ideas fall out on their own; restored work leaves the shelf
  // on the next save.
  private shelvedDeclinedIdeas(): ProductIdea[] {
    const byID = new Map((this.revision?.ideas || []).map((idea) => [idea.id, idea]));
    const seen = new Set<string>();
    const declined: ProductIdea[] = [];
    for (const task of this.shelvedTasks) {
      for (const id of task.idea_ids) {
        const idea = byID.get(id);
        if (idea?.decision === 'decline' && !seen.has(id)) {
          seen.add(id);
          declined.push(idea);
        }
      }
    }
    return declined;
  }
  private declinedWorkRecovery() {
    if (!this.shelvedTasks.length) return '';
    const declined = this.shelvedDeclinedIdeas();
    const count = this.shelvedTasks.length;
    return html`<div class="declined-work-gate" role="group" aria-label="Declined work recovery">
      <p>
        ${count} proposed work item${count === 1 ? ' was' : 's were'} removed because
        ${declined.length ? 'a linked idea was declined' : 'the linked ideas changed'}; your other work is kept and
        the proposal saved. Re-accept an idea below to restore its work on the next save, use Change something to
        regenerate work from the remaining direction, or leave the ideas declined to keep the work out.
      </p>
      ${declined.map(
        (idea) => html`<p>${idea.description}
          <button
            aria-label="Re-accept declined idea ${idea.id}"
            ?disabled=${this.busy || this.blocked}
            @click=${() => this.recoverDeclined(idea.id)}
          >
            Re-accept
          </button>
        </p>`
      )}
    </div>`;
  }
  private recoverDeclined = (id: string) => {
    if (!this.revision || this.busy || this.blocked) return;
    this.decide(id, 'accept');
  };
  // Declined ideas with no local shelf coverage: the only ones that may need
  // saved-history recovery. Terminal heads carry no proposal, so they never
  // consult history.
  private historyUncoveredIdeas(): ProductIdea[] {
    const revision = this.revision;
    if (!revision || this.terminalRevision()) return [];
    const shelved = new Set<string>();
    for (const task of this.shelvedTasks) for (const id of task.idea_ids) shelved.add(id);
    return (revision.ideas || []).filter((idea) => idea.decision === 'decline' && !shelved.has(idea.id));
  }
  // Saved-history lookup for a shelfless resume (another device): one bare
  // GET on the owner-private history endpoint per head. Permission, CAS and
  // transport failures keep the resume intact and degrade to the honest
  // unavailable note; they never block saved work and never retry in a loop.
  private maybeRecoverFromHistory(): void {
    const revision = this.revision;
    if (!revision || !this.api || !this.active || this.blocked) return;
    const uncovered = this.historyUncoveredIdeas();
    if (uncovered.length === 0) {
      this.savedHistory = [];
      this.savedHistoryNote = '';
      this.savedHistorySkipped = 0;
      return;
    }
    if (this.savedHistoryHead === revision.id) return;
    // Claim the head before the read so one resume issues one history GET.
    this.savedHistoryHead = revision.id;
    this.savedHistory = [];
    this.savedHistorySkipped = 0;
    if (!revision.predecessor) {
      this.savedHistoryNote = savedHistoryNoPredecessor;
      this.requestUpdate();
      return;
    }
    this.savedHistoryNote = '';
    const actor = this.actor;
    const headId = revision.id;
    const intent = revision.intent;
    void (async () => {
      try {
        const chain = intentHistory(
          await this.call('/api/factory/intents/' + intent + '/history', undefined, AbortSignal.timeout(20000))
        );
        if (this.actor !== actor || this.blocked || this.revision?.id !== headId) return;
        this.adoptSavedHistory(chain, uncovered);
      } catch {
        if (this.actor !== actor || this.blocked || this.revision?.id !== headId) return;
        if (this.historyUncoveredIdeas().length === 0) return;
        this.savedHistory = [];
        this.savedHistoryNote = savedHistoryUnavailable;
        this.savedHistorySkipped = 0;
        this.requestUpdate();
      }
    })();
  }
  // Bounded exact traversal of the saved chain: follow ONLY the predecessor
  // links the current head names, newest first. The read is bound to the
  // freshly loaded head first: the chain must carry the exact head with its
  // recorded predecessor (a CAS-like freshness check), or nothing recovers.
  // Each hop must stay on the same request chain. A recorded task qualifies
  // only when EVERY linked idea still means the same in the current head
  // (identity, description, consequence and effective correction) and no
  // link beyond the recovered idea is declined: replaying a task across
  // changed linked semantics would guess, so it refuses. A revision that
  // merely bears a graph for other ideas never qualifies, and a same-identity
  // idea with changed meaning stops that idea instead of guessing across it.
  // Only the qualifying tasks are staged, with demonstrably compatible
  // edges (live, staged alongside, or remapped onto a unique
  // exact-idea-identical live successor) plus repairs for recorded kept-edge
  // cuts into the recovered set. Anything unprovable is named in the preview
  // instead of silently pruned or invented — never a union, never a guess.
  private adoptSavedHistory(chain: IntentRevision[], uncovered: ProductIdea[]): void {
    const revision = this.revision;
    if (!revision || this.terminalRevision()) return;
    const byID = new Map(chain.map((entry) => [entry.id, entry]));
    // Cross-device stale-head guard (FJ-443): the chain is newest-first, so
    // the loaded head must be the authoritative newest entry. Another device
    // advancing H→H2 before the history GET yields [H2,H,P]: H is still
    // present with its predecessor, but previewing from it would decide on
    // a stale head (the backend CAS would refuse the durable save, yet the
    // preview and user decision would already be built on the wrong head).
    // Refuse with the honest unavailable state; resuming the fresh head
    // (chain[0]) re-offers that head's recovery. No guessed graph.
    const newest = chain.length > 0 ? chain[0] : undefined;
    if (!newest || newest.id !== revision.id || newest.intent !== revision.intent) {
      this.savedHistory = [];
      this.savedHistoryNote = savedHistoryUnavailable;
      this.savedHistorySkipped = 0;
      this.requestUpdate();
      return;
    }
    const headID = revision.id;
    const headEntry = byID.get(headID);
    if (!headEntry || headEntry.intent !== revision.intent || headEntry.predecessor !== revision.predecessor) {
      this.savedHistory = [];
      this.savedHistoryNote = savedHistoryUnavailable;
      this.savedHistorySkipped = 0;
      this.requestUpdate();
      return;
    }
    const nowByID = new Map(revision.ideas.map((idea) => [idea.id, idea]));
    const live = new Set(revision.tasks.map((task) => task.key));
    const liveByIdeas = new Map<string, string[]>();
    for (const task of revision.tasks) {
      const signature = [...task.idea_ids].sort().join('\0');
      const bucket = liveByIdeas.get(signature);
      if (bucket) bucket.push(task.key);
      else liveByIdeas.set(signature, [task.key]);
    }
    const pastByID = (entry: IntentRevision) => new Map(entry.ideas.map((idea) => [idea.id, idea]));
    const recovered: SavedHistoryEntry[] = [];
    const missing = new Set(uncovered.map((idea) => idea.id));
    const stagedKeys = new Set<string>();
    let skipped = 0;
    let cursor = revision.predecessor;
    for (let hops = 0; cursor && missing.size > 0 && hops < savedHistoryHopBound; hops += 1) {
      const entry = byID.get(cursor);
      if (!entry || entry.intent !== revision.intent) break;
      const wasByID = pastByID(entry);
      for (const idea of uncovered) {
        if (!missing.has(idea.id)) continue;
        const then = wasByID.get(idea.id);
        if (!then) continue;
        if (!sameIdeaMeaning(then, idea)) {
          missing.delete(idea.id);
          continue;
        }
        const linked = entry.tasks.filter((task) => task.idea_ids.includes(idea.id) && !live.has(task.key));
        const candidates = linked.filter((task) =>
          task.idea_ids.every((link) => historyLinkOK(link, idea.id, wasByID, nowByID))
        );
        skipped += linked.length - candidates.length;
        if (candidates.length === 0) continue;
        const batchKeys = new Set(candidates.map((task) => task.key));
        const tasks: IntentTask[] = [];
        const cutEdges: SavedHistoryCutEdge[] = [];
        for (const task of candidates) {
          const kept: string[] = [];
          for (const dep of task.depends_on || []) {
            if (live.has(dep) || stagedKeys.has(dep) || batchKeys.has(dep)) {
              // Same-key live edge (FJ-444/FJ-454): the key resolving live does
              // not prove the target still means the same, nor that it still
              // binds the same work. A kept B whose description/correction
              // changed must not carry the recorded A→B edge forward by key
              // identity alone — and neither may a B that kept its key while
              // its idea links changed (historical B linked b, current B
              // links c): canonical drafts permit changed bindings, so the
              // edge restores only onto an EXACT old/current idea-set match,
              // else it stays cut with an explicit warning, never silent.
              if (live.has(dep) && !stagedKeys.has(dep) && !batchKeys.has(dep)) {
                const pastTarget = entry.tasks.find((candidate) => candidate.key === dep);
                if (pastTarget && pastTarget.idea_ids.length > 0) {
                  const liveTarget = revision.tasks.find((candidate) => candidate.key === dep);
                  const lineageOK =
                    liveTarget !== undefined &&
                    liveTarget.idea_ids.length === pastTarget.idea_ids.length &&
                    [...liveTarget.idea_ids].sort().join('\0') ===
                      [...pastTarget.idea_ids].sort().join('\0');
                  const targetOK =
                    lineageOK &&
                    pastTarget.idea_ids.every((link) => {
                      const past = wasByID.get(link);
                      const now = nowByID.get(link);
                      return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
                    });
                  if (!targetOK) {
                    cutEdges.push({key: task.key, objective: task.objective, dep, kind: 'task'});
                    continue;
                  }
                }
              }
              kept.push(dep);
              continue;
            }
            // Outgoing remap (FJ-444): a recorded A→B-old edge remaps onto
            // the live successor only when the target's semantics still
            // match — same idea set AND every linked idea still means the
            // same (description/consequence/correction). Matching by idea
            // IDs alone would guess across a changed B.
            const pastTarget = entry.tasks.find((candidate) => candidate.key === dep);
            const signature = pastTarget ? [...pastTarget.idea_ids].sort().join('\0') : '';
            const successors = signature ? liveByIdeas.get(signature) || [] : [];
            if (pastTarget && successors.length === 1) {
              const successorKey = successors[0] as string;
              const liveTarget = revision.tasks.find((candidate) => candidate.key === successorKey);
              const successorOK =
                pastTarget.idea_ids.length > 0 &&
                liveTarget !== undefined &&
                [...liveTarget.idea_ids].sort().join('\0') === signature &&
                pastTarget.idea_ids.every((link) => {
                  const past = wasByID.get(link);
                  const now = nowByID.get(link);
                  return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
                });
              if (successorOK) {
                kept.push(successorKey);
                continue;
              }
            }
            cutEdges.push({key: task.key, objective: task.objective, dep, kind: 'task'});
          }
          stagedKeys.add(task.key);
          // A remap can preserve the edge count while changing its targets,
          // so compare values — never the length alone — before reusing the
          // recorded shape.
          const unchanged =
            kept.length === (task.depends_on || []).length &&
            kept.every((dep, index) => dep === (task.depends_on || [])[index]);
          tasks.push(unchanged ? {...task} : {...task, depends_on: kept});
        }
        // Repairs for the other direction (kept-depends-on-removed): a
        // recorded edge from a kept (unstaged) task into the recovered set
        // returns only onto a demonstrably compatible owner — live under the
        // recorded key or the unique exact-idea-identical live successor,
        // with every owner link still meaning the same. Anything else waits
        // unassigned (a later regeneration may claim it through the ordinary
        // shelf machinery) and the preview warns instead of guessing.
        const repairs: SavedHistoryRepair[] = [];
        for (const owner of entry.tasks) {
          if (batchKeys.has(owner.key) || stagedKeys.has(owner.key)) continue;
          const incoming = (owner.depends_on || []).filter((dep) => batchKeys.has(dep) || stagedKeys.has(dep));
          if (incoming.length === 0) continue;
          const ownerOK = owner.idea_ids.every((link) => {
            const past = wasByID.get(link);
            const now = nowByID.get(link);
            return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
          });
          // Narrowed recovery (FJ-444): stage ONLY justified cut edges into
          // the recovered set — an incoming edge whose target still means
          // the same (every linked idea identical past vs now). The full
          // historical B.depends_on (e.g., old B→[A,C] recovering A while
          // current B deliberately dropped C) must never resurrect the
          // deliberately removed C, although the preview promises only A.
          // Unjustified incoming edges warn instead of staging.
          const justified = incoming.filter((dep) => {
            const pastTarget = entry.tasks.find((candidate) => candidate.key === dep);
            if (!pastTarget || pastTarget.idea_ids.length === 0) return false;
            return pastTarget.idea_ids.every((link) => {
              const past = wasByID.get(link);
              const now = nowByID.get(link);
              return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
            });
          });
          const depIdeas: Record<string, string[]> = {};
          for (const dep of justified) {
            const pastTarget = entry.tasks.find((candidate) => candidate.key === dep);
            if (pastTarget && pastTarget.idea_ids.length > 0) depIdeas[dep] = [...pastTarget.idea_ids];
          }
          const orderedDeps = (owner.depends_on || []).filter((dep) => justified.includes(dep));
          const repair = {
            deps: orderedDeps,
            ideas: [...owner.idea_ids],
            depIdeas,
            targets: [...justified],
          };
          if (!ownerOK) {
            for (const dep of incoming)
              cutEdges.push({key: owner.key, objective: owner.objective, dep, kind: 'repair'});
            continue;
          }
          for (const dep of incoming) {
            if (!justified.includes(dep))
              cutEdges.push({key: owner.key, objective: owner.objective, dep, kind: 'repair'});
          }
          if (justified.length === 0) continue;
          const signature = [...owner.idea_ids].sort().join('\0');
          const successors = liveByIdeas.get(signature) || [];
          // Same-key live owner (FJ-454): the key resolving live does not
          // prove the work still binds the same ideas. A same-key owner
          // whose idea set changed (historical B linked b, current B links
          // c) must never resolve silently — canonical drafts permit changed
          // bindings, so the recorded edge stays cut with an explicit
          // warning instead of restoring onto changed work. Only an EXACT
          // old/current idea-set match resolves by key.
          const liveOwner = live.has(owner.key)
            ? revision.tasks.find((candidate) => candidate.key === owner.key)
            : undefined;
          const liveOwnerSameIdeas =
            liveOwner !== undefined &&
            liveOwner.idea_ids.length === owner.idea_ids.length &&
            [...liveOwner.idea_ids].sort().join('\0') === signature;
          if (live.has(owner.key) && !liveOwnerSameIdeas) {
            for (const dep of incoming)
              cutEdges.push({key: owner.key, objective: owner.objective, dep, kind: 'repair'});
            continue;
          }
          if (live.has(owner.key)) repairs.push({...repair, owner: owner.key, resolved: true});
          else if (successors.length === 1) repairs.push({...repair, owner: successors[0] as string, resolved: true});
          else repairs.push({...repair, owner: owner.key, resolved: false});
        }
        recovered.push({
          idea: idea.id,
          revision: entry.id,
          tasks,
          repairs,
          cutEdges,
          ideas: entry.ideas.map((row) => ({...row})),
        });
        missing.delete(idea.id);
      }
      cursor = entry.predecessor;
    }
    this.savedHistory = recovered;
    this.savedHistorySkipped = skipped;
    this.savedHistoryNote =
      recovered.length === 0 ? savedHistoryUnavailable : missing.size > 0 ? savedHistoryPartial : '';
    this.requestUpdate();
  }
  // User-driven re-accept from saved history: stage ONLY the recorded bytes
  // for this idea into the shelf, then decide through the ordinary path so
  // the next save drafts them onto the freshly checked current head. Rows
  // are bound to the head they were derived from: a stale row refuses and
  // cleans up instead of acting. Decisions may have moved since the lookup,
  // so every task and repair is re-verified against the fresh head before
  // it stages — changed links refuse, proven edges and repairs merge into
  // the shelf (never overwriting live work or local records), and the
  // existing save-time rules still govern the final draft. No edge is ever
  // invented.
  private recoverFromHistory = (id: string) => {
    const revision = this.revision;
    if (!revision || this.busy || this.blocked) return;
    if (this.savedHistoryHead !== revision.id) {
      this.savedHistory = [];
      this.savedHistoryNote = '';
      this.savedHistorySkipped = 0;
      return;
    }
    const slot = this.savedHistory.findIndex((row) => row.idea === id);
    const entry = slot >= 0 ? this.savedHistory[slot] : undefined;
    const idea = revision.ideas.find((row) => row.id === id);
    if (!entry || !idea || idea.decision !== 'decline') return;
    const nowByID = new Map(revision.ideas.map((row) => [row.id, row]));
    const wasByID = new Map(entry.ideas.map((row) => [row.id, row]));
    const live = new Set(revision.tasks.map((task) => task.key));
    const proposed = new Set(revision.ideas.map((row) => row.id));
    const tasks = entry.tasks.filter(
      (task) =>
        task.idea_ids.includes(id) &&
        !live.has(task.key) &&
        task.idea_ids.length > 0 &&
        task.idea_ids.every((link) => proposed.has(link)) &&
        task.idea_ids.every((link) => historyLinkOK(link, id, wasByID, nowByID))
    );
    if (tasks.length === 0) return;
    const staged = new Set(tasks.map((task) => task.key));
    for (const task of tasks) {
      if (this.shelvedTasks.some((shelved) => shelved.key === task.key)) continue;
      // Staged task edges (FJ-444): a same-key live dep still requires target
      // semantics — drop it when the live target's linked ideas changed
      // meaning, never carry it by key identity alone.
      const deps = (task.depends_on || []).filter((dep) => {
        if (staged.has(dep)) return true;
        if (!live.has(dep)) return false;
        const liveTarget = revision.tasks.find((candidate) => candidate.key === dep);
        if (!liveTarget || liveTarget.idea_ids.length === 0) return true;
        return liveTarget.idea_ids.every((link) => {
          const past = wasByID.get(link);
          const now = nowByID.get(link);
          return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
        });
      });
      this.shelvedTasks.push(
        deps.length === (task.depends_on || []).length ? {...task} : {...task, depends_on: deps}
      );
    }
    for (const repair of entry.repairs) {
      // Narrowed staging (FJ-444): merge ONLY justified edges into the
      // recovered set — deps pointing at staged work whose recorded target
      // lineage still means the same. A live-only dep (e.g., deliberately
      // removed C that is still live) must never merge, even though the
      // preview promises only A. Preserve current other edges by merging
      // the filtered subset, never the full historical list.
      const justifiedDeps = repair.deps.filter((dep) => {
        if (!staged.has(dep)) return false;
        const ids = repair.depIdeas[dep];
        if (!ids || ids.length === 0) return true;
        return ids.every((link) => {
          const past = wasByID.get(link);
          const now = nowByID.get(link);
          return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
        });
      });
      if (justifiedDeps.length === 0) continue;
      const ownerOK =
        repair.ideas.length > 0 &&
        repair.ideas.every((link) => {
          const past = wasByID.get(link);
          const now = nowByID.get(link);
          return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
        });
      if (!ownerOK) continue;
      const narrowedDepIdeas: Record<string, string[]> = {};
      for (const dep of justifiedDeps) {
        const ids = repair.depIdeas[dep];
        if (ids) narrowedDepIdeas[dep] = [...ids];
      }
      this.mergeHistoryRepair({
        ...repair,
        deps: [...justifiedDeps],
        targets: repair.targets.filter((dep) => justifiedDeps.includes(dep)),
        depIdeas: narrowedDepIdeas,
      });
    }
    this.savedHistory = this.savedHistory.filter((row) => row.idea !== id);
    if (this.historyUncoveredIdeas().length === 0) {
      this.savedHistoryNote = '';
      this.savedHistorySkipped = 0;
    }
    this.decide(id, 'accept');
  };
  // Union-merge a history-staged repair into the shelf record without
  // overwriting local entries: recorded edges accumulate like the ordinary
  // transfer path, so a local repair for the same owner survives.
  private mergeHistoryRepair(repair: SavedHistoryRepair): void {
    const prior = this.shelvedPrunedDeps[repair.owner];
    if (!prior) {
      this.shelvedPrunedDeps[repair.owner] = {
        deps: [...repair.deps],
        ideas: [...repair.ideas],
        ...(Object.keys(repair.depIdeas).length > 0 ? {depIdeas: {...repair.depIdeas}} : {}),
      };
      return;
    }
    for (const dep of repair.deps) if (!prior.deps.includes(dep)) prior.deps.push(dep);
    for (const link of repair.ideas) if (!prior.ideas.includes(link)) prior.ideas.push(link);
    if (Object.keys(repair.depIdeas).length > 0) {
      prior.depIdeas = prior.depIdeas || {};
      for (const [dep, ids] of Object.entries(repair.depIdeas)) {
        const bucket = prior.depIdeas[dep] || [];
        for (const link of ids) if (!bucket.includes(link)) bucket.push(link);
        prior.depIdeas[dep] = bucket;
      }
    }
  }
  // Saved-history recovery gate: the explicit preview (source revision, work,
  // proven edges and recorded repairs per idea) plus the honest unavailable
  // states. Rows render only for the head they were derived from; stale
  // rows never stay actionable. Unsent edits stay local-only: only saved
  // bytes are ever shown here.
  private savedHistoryRecovery() {
    const revision = this.revision;
    if (!revision || this.terminalRevision()) return '';
    if (this.savedHistoryHead !== revision.id) return '';
    const uncovered = new Set(this.historyUncoveredIdeas().map((idea) => idea.id));
    if (uncovered.size === 0) return '';
    const byID = new Map(revision.ideas.map((idea) => [idea.id, idea]));
    const live = new Set(revision.tasks.map((task) => task.key));
    const rows: {idea: ProductIdea; entry: SavedHistoryEntry; tasks: IntentTask[]; repairs: SavedHistoryRepair[]}[] =
      [];
    for (const entry of this.savedHistory) {
      if (!uncovered.has(entry.idea)) continue;
      const idea = byID.get(entry.idea);
      if (!idea || idea.decision !== 'decline') continue;
      const wasByID = new Map(entry.ideas.map((row) => [row.id, row]));
      const tasks = entry.tasks.filter(
        (task) =>
          task.idea_ids.includes(entry.idea) &&
          !live.has(task.key) &&
          task.idea_ids.every((link) => historyLinkOK(link, entry.idea, wasByID, byID))
      );
      if (tasks.length === 0) continue;
      const staged = new Set(tasks.map((task) => task.key));
      // Preview narrows like staging (FJ-444): only repairs with a
      // justified edge into the staged set render as returning; live-only
      // deps (deliberately removed edges) never promise.
      const repairs = entry.repairs.filter((repair) => {
        if (!live.has(repair.owner)) return false;
        const justified = repair.deps.filter((dep) => {
          if (!staged.has(dep)) return false;
          const ids = repair.depIdeas[dep];
          if (!ids || ids.length === 0) return true;
          return ids.every((link) => {
            const past = wasByID.get(link);
            const now = byID.get(link);
            return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
          });
        });
        if (justified.length === 0) return false;
        return repair.ideas.every((link) => {
          const past = wasByID.get(link);
          const now = byID.get(link);
          return past !== undefined && now !== undefined && sameIdeaMeaning(past, now);
        });
      });
      rows.push({idea, entry, tasks, repairs});
    }
    const pending = this.savedHistory.filter(
      (entry) =>
        uncovered.has(entry.idea) &&
        byID.get(entry.idea)?.decision === 'decline' &&
        entry.repairs.some((repair) => !live.has(repair.owner))
    ).length;
    if (rows.length === 0) {
      if (!this.savedHistoryNote) return '';
      return html`<div class="declined-work-gate" role="group" aria-label="Saved history recovery">
        <p>${this.savedHistoryNote}</p>
      </div>`;
    }
    return html`<div class="declined-work-gate" role="group" aria-label="Saved history recovery">
      <p>
        Saved history holds work for re-accept consideration. Only saved revisions are shown; unsent edits stay on
        this device, and work that was never saved cannot be recovered.
      </p>
      ${rows.map(({idea, entry, tasks, repairs}) => {
        const returning = repairs.filter((repair) => repair.targets.some((dep) => !live.has(dep)));
        return html`<p>
          Saved revision ${entry.revision} records ${tasks.length} work item${tasks.length === 1 ? '' : 's'}
          for “${idea.description}”: ${tasks
            .map(
              (task) =>
                `${task.objective}${task.depends_on?.length ? ` (depends on ${task.depends_on.join(', ')})` : ''}`
            )
            .join('; ')}
          ${returning.length > 0
            ? html`Recorded ${returning.length === 1 ? 'dependency' : 'dependencies'}
              ${returning.map((repair) => repair.targets.join(', ')).join('; ')}
              return${returning.length === 1 ? 's' : ''} on re-accept.`
            : ''}
          ${entry.cutEdges.map(
            (cut) =>
              html`Recorded edge ${cut.objective} → ${cut.dep} stays cut instead of being guessed. No edge was
                invented.`
          )}
          <button
            aria-label="Re-accept declined idea from saved history ${entry.idea}"
            ?disabled=${this.busy || this.blocked}
            @click=${() => this.recoverFromHistory(entry.idea)}
          >
            Re-accept
          </button>
        </p>`;
      })}
      ${pending > 0
        ? html`<p>
            ${pending} recorded dependenc${pending === 1 ? 'y has' : 'ies have'} no unique live successor yet, so
            ${pending === 1 ? 'it waits' : 'they wait'} unassigned instead of being guessed.
          </p>`
        : ''}
      ${this.savedHistorySkipped > 0
        ? html`<p>
            ${this.savedHistorySkipped} other recorded work item${this.savedHistorySkipped === 1 ? '' : 's'} were
            left out: their linked ideas changed or are declined in the current head. Nothing was guessed.
          </p>`
        : ''}
      ${this.savedHistoryNote ? html`<p>${this.savedHistoryNote}</p>` : ''}
    </div>`;
  };
  // Unresolved repair review: a recorded dependency whose work has no
  // unique successor yet stays visible — across saves, reloads and leaves
  // — instead of being silently erased. No edge is invented while it waits:
  // a later regeneration that produces a unique successor re-attaches it
  // automatically (Change something can force that regeneration), and the
  // per-record Confirm drop is the explicit user resolution when no
  // successor will ever come. Dropping removes the recovery record only;
  // the saved graph is untouched.
  private unresolvedRepairReview() {
    const pending = this.unresolvedRepairs();
    if (pending.length === 0) return '';
    const byID = new Map((this.revision?.ideas || []).map((idea) => [idea.id, idea]));
    return html`<div class="unresolved-repair-gate" role="group" aria-label="Unresolved dependency repair">
      <p>
        A recorded dependency has no unique successor to re-attach to, so it is kept unassigned rather than
        guessed. No dependency was invented. It resolves on its own when its work reappears uniquely (Change
        something can regenerate the work), or confirm below that the recorded repair should be dropped.
      </p>
      ${pending.map(
        ({key, repair}) => html`<p>${repair.ideas.map((id) => byID.get(id)?.description || id).join('; ')}
          <button
            aria-label="Confirm drop of unassigned repair ${key}"
            ?disabled=${this.busy || this.blocked}
            @click=${() => this.dropUnresolvedRepair(key)}
          >
            Confirm drop
          </button>
        </p>`
      )}
    </div>`;
  }
  private dropUnresolvedRepair = (key: string) => {
    if (!this.revision || this.busy || this.blocked) return;
    if (!this.unresolvedRepairs().some((entry) => entry.key === key)) return;
    delete this.shelvedPrunedDeps[key];
    this.rememberDraft();
    this.requestUpdate();
  };
  // After the server replaces the proposal, shelved drops stay only while
  // their ideas still exist and no fresh task supersedes their key: explore
  // regenerates work from the current decisions, so a cleared new request
  // forgets its drops while re-accepted ideas keep theirs for the next save.
  // Repair records survive while their task is still proposed with a stored
  // edge subset of the recorded original (a regenerated task with new edges
  // drops its stale record) and a recorded edge still points at shelved
  // work. When explore regenerated the work under a fresh key, the record
  // follows its work only onto a UNIQUE exact-idea-identical successor, so
  // a later re-accept still restores the cut edges onto the proven task. An
  // ambiguous split (two or more tasks sharing the exact lineage) transfers
  // to NONE and surfaces explicit review instead of guessing. Recorded TARGETS follow their
  // successors the same way, by the stable idea lineage pinned at record
  // time: a unique idea-identical successor remaps the edge, while zero or
  // several successors leave the edge cut (never invented) with the pending
  // lineage kept for a later regeneration to recover.
  private reconcileShelvedTasks(proposal: IntentRevision) {
    const ideas = new Set(proposal.ideas.map((idea) => idea.id));
    const live = new Set(proposal.tasks.map((task) => task.key));
    this.shelvedTasks = this.shelvedTasks.filter(
      (task) => !live.has(task.key) && task.idea_ids.length > 0 && task.idea_ids.every((id) => ideas.has(id))
    );
    const shelved = new Set(this.shelvedTasks.map((task) => task.key));
    const proposed = new Map(proposal.tasks.map((task) => [task.key, task]));
    // Index fresh keys by stable idea linkage: the same ideas under a new
    // key are the same work regenerated, whatever the key says.
    const byIdeas = new Map<string, string[]>();
    for (const task of proposal.tasks) {
      const signature = [...task.idea_ids].sort().join('\0');
      const bucket = byIdeas.get(signature);
      if (bucket) bucket.push(task.key);
      else byIdeas.set(signature, [task.key]);
    }
    const recorded: Record<string, ShelvedDepRepair> = {};
    const resolvedTargets = new Map<string, string>();
    // Remap a record's targets onto successor keys by stable idea lineage.
    // A target that still resolves under its key keeps its pinned lineage
    // for the next regeneration. A target whose lineage is ambiguous or
    // gone is left untouched: the edge stays cut rather than invented, and
    // pending lineage survives for a later regeneration.
    const remapRepair = (repair: ShelvedDepRepair): ShelvedDepRepair => {
      const entry = copyRepair(repair);
      if (!entry.depIdeas) return entry;
      const pending: Record<string, string[]> = {};
      const mapped = entry.deps.map((dep) => {
        const ids = entry.depIdeas?.[dep];
        if (!ids) return dep;
        if (proposed.has(dep)) {
          pending[dep] = [...ids];
          return dep;
        }
        const candidates = byIdeas.get([...ids].sort().join('\0')) || [];
        if (candidates.length === 1) {
          const successor = candidates[0] as string;
          resolvedTargets.set(dep, successor);
          // The lineage follows the successor key: a later regeneration
          // re-keys the successor again, and only the carried mapping lets
          // the next remap follow it. Dropping it here would strand the
          // record on a stale key the next explore cannot resolve.
          pending[successor] = [...ids];
          return successor;
        }
        pending[dep] = [...ids];
        return dep;
      });
      // Two recorded targets with identical lineage resolve onto the same
      // successor: one edge, not a duplicated one.
      const deps = [...new Set(mapped)];
      const next: ShelvedDepRepair = {deps, ideas: [...entry.ideas]};
      if (Object.keys(pending).length > 0) next.depIdeas = pending;
      return next;
    };
    const transfer = (key: string, repair: ShelvedDepRepair) => {
      const current = recorded[key];
      if (!current) {
        recorded[key] = copyRepair(repair);
        return;
      }
      for (const dep of repair.deps) if (!current.deps.includes(dep)) current.deps.push(dep);
      for (const id of repair.ideas) if (!current.ideas.includes(id)) current.ideas.push(id);
      if (repair.depIdeas) {
        current.depIdeas = current.depIdeas || {};
        for (const [dep, ids] of Object.entries(repair.depIdeas)) {
          const bucket = current.depIdeas[dep] || [];
          for (const id of ids) if (!bucket.includes(id)) bucket.push(id);
          current.depIdeas[dep] = bucket;
        }
      }
    };
    const adoptable = (task: IntentTask, repair: ShelvedDepRepair) =>
      [...(task.depends_on || [])].every((dep) => repair.deps.includes(dep)) &&
      repair.deps.some((dep) => shelved.has(dep));
    // A unique successor whose recorded targets are ALL live replays the
    // exact recorded edges on the next save: adopt it onto the successor
    // rather than dropping the record for having nothing left to wait for.
    // The lineage match below is already exact (same idea set); no edge is
    // guessed — every replayed target resolves to live work.
    const liveAdoptable = (task: IntentTask, repair: ShelvedDepRepair) =>
      repair.deps.length > 0 &&
      [...(task.depends_on || [])].every((dep) => repair.deps.includes(dep)) &&
      repair.deps.every((dep) => proposed.has(dep));
    const carriesRepair = (entry: ShelvedDepRepair) =>
      entry.deps.length > 0 || Object.keys(entry.depIdeas || {}).length > 0;
    // Owner ideas still proposed: the work may come back, so an unclaimed
    // record waits for a later regeneration or an explicit confirm drop.
    // Ideas gone means the direction left the work behind and the record is
    // forgotten with the shelf.
    const ownerIdeasProposed = (entry: ShelvedDepRepair) =>
      entry.ideas.length > 0 && entry.ideas.every((id) => ideas.has(id));
    const ambiguousOwners: string[] = [];
    for (const [key, repair] of Object.entries(this.shelvedPrunedDeps)) {
      const entry = remapRepair(repair);
      const current = proposed.get(key);
      if (current) {
        if (adoptable(current, entry)) transfer(key, entry);
        continue;
      }
      if (shelved.has(key)) {
        // The owner stays shelved under its stable key: keep the entry with
        // remapped targets so a later re-accept restores onto the successor
        // work. Entries with no edges and no pending lineage carry nothing.
        if (entry.deps.length > 0 || Object.keys(entry.depIdeas || {}).length > 0) recorded[key] = entry;
        continue;
      }
      // Owner lineage must be proven exact, never guessed from a shared
      // idea: the recorded idea links must equal the successor's idea links
      // exactly (same set). A split that shares the lineage across two or
      // more tasks proves neither successor, so the repair transfers to NONE
      // and surfaces explicit review instead of inventing an edge on every
      // twin.
      const signature = [...entry.ideas].sort().join('\0');
      const exact = proposal.tasks.filter(
        (task) => task.idea_ids.length === entry.ideas.length && [...task.idea_ids].sort().join('\0') === signature
      );
      if (exact.length === 1) {
        const sole = exact[0] as IntentTask;
        if (adoptable(sole, entry) || liveAdoptable(sole, entry)) transfer(sole.key, entry);
        // Unique but not adoptable (the successor carries out-of-record
        // edges, or a recorded target is neither live nor shelved): keep the
        // remapped record under the original owner key instead of erasing
        // it — a later regeneration or an explicit confirm drop resolves it.
        else if (carriesRepair(entry) && ownerIdeasProposed(entry)) recorded[key] = entry;
        continue;
      }
      if (exact.length > 1) {
        // Ambiguous split: keep the remapped record under the original owner
        // key so a later regeneration can still recover it, and surface
        // explicit review so the user picks the true successor or re-maps
        // manually. Saves carry the unassigned record (recordPrunedDeps) and
        // the unresolved review keeps it visible until a unique successor
        // claims it or the user confirms the drop. Entries with no edges
        // and no pending lineage carry nothing.
        if (entry.deps.length > 0 || Object.keys(entry.depIdeas || {}).length > 0) recorded[key] = entry;
        ambiguousOwners.push(key);
        continue;
      }
      // No successor carries the recorded lineage: keep the record while
      // its owner ideas are still proposed (a later regeneration may bring
      // the work back); forget it with the shelf once its ideas are gone.
      if (carriesRepair(entry) && ownerIdeasProposed(entry)) recorded[key] = entry;
    }
    // Rewrite shelved tasks' own edges onto resolved successors: the shelf
    // entry keeps its meaning (same lineage, fresh key) so a later
    // re-accept prunes nothing that came back live.
    if (resolvedTargets.size > 0)
      this.shelvedTasks = this.shelvedTasks.map((task) => {
        if (!task.depends_on || task.depends_on.length === 0) return task;
        let changed = false;
        const depends_on = task.depends_on.map((dep) => {
          const successor = !shelved.has(dep) && !proposed.has(dep) ? resolvedTargets.get(dep) : undefined;
          if (successor && successor !== dep) {
            changed = true;
            return successor;
          }
          return dep;
        });
        return changed ? {...task, depends_on} : task;
      });
    this.shelvedPrunedDeps = recorded;
    this.pendingDropped = [];
    this.pendingRestored = [];
    // Ambiguous split: the repair stayed with no owner rather than guessed,
    // so say so explicitly — the user picks the true successor or re-maps
    // manually (Change something), and no edge was invented.
    if (ambiguousOwners.length > 0)
      this.error =
        'Exploration split linked work into several items with identical idea links, so a recorded dependency repair was left unassigned instead of guessed. Review the proposed work and use Change something to correct it; no dependency was invented.';
    this.requestUpdate();
  }
  private async save(state: IntentRevision['state'], repository: string, signal: AbortSignal) {
    const next = this.nextRevision(state, repository);
    if ((state === 'draft' || state === 'accepted') && this.scopeNarrowsToEmpty(next)) {
      this.error =
        'Your latest choice emptied the explicitly narrowed scope while decided ideas remain. Review the authorized ideas explicitly before saving; nothing was saved and no remaining idea was authorized.';
      this.scopeGateArmed = true;
      throw Error('Explicit scope narrowed to empty with decided ideas remaining');
    }
    this.scopeGateArmed = false;
    const saved = intentRevision(await this.call('/api/factory/intents', {...next}, signal));
    if (
      saved.id !== next.id ||
      saved.intent !== next.intent ||
      saved.state !== state ||
      saved.repository !== repository
    )
      throw Error('Mismatched intent revision');
    this.shelveDroppedTasks(next);
    this.revision = saved;
    this.savedSnapshot = structuredClone(saved);
    this.savedLoaded = false;
    this.savedError = '';
    // Advance the draft to the saved revision so a reload resumes the same
    // base with the same shelf: save→reload→recovery keeps the Re-accept
    // path with dependencies intact.
    this.rememberDraft();
    // A committed save advances the head, so staged history rows are
    // invalid: clear them and re-evaluate recovery for the new head (a
    // no-op once the save just shelved the drops).
    this.savedHistory = [];
    this.savedHistoryNote = '';
    this.savedHistorySkipped = 0;
    this.savedHistoryHead = '';
    this.maybeRecoverFromHistory();
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
  // Review-open allowance load: read-only and guarded by the caller's
  // lifecycle and the current intent. A stale or superseded reply never
  // overwrites the current revision's allowance, and nothing here posts or
  // plans. Only an absent allowance degrades silently; other failures stay
  // visible behind a reachable retry instead of hiding the raise action.
  private async refreshAllowance(intent: string, signal: AbortSignal) {
    try {
      const found = planningAllowance(
        await this.call('/api/factory/intents/' + intent + '/allowance', undefined, signal),
        intent
      );
      if (signal.aborted || this.revision?.intent !== intent) return;
      this.allowance = found;
      this.allowanceError = '';
    } catch (error) {
      if (signal.aborted || this.revision?.intent !== intent) return;
      if (error instanceof SodaRequestError && error.status === 404) {
        this.allowance = undefined;
        this.allowanceError = '';
        return;
      }
      this.allowanceError =
        'The planning allowance could not be loaded. Your description is saved; retry the load to try again.';
    }
  }
  private retryAllowance = () => {
    const intent = this.revision?.intent;
    if (!intent || this.busy || this.blocked) return;
    void this.perform((signal) => this.refreshAllowance(intent, signal), 'Loading planning allowance…');
  };
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
  // raiseNextStep names the actual pending action after a top-up: a
  // walkthrough awaiting its check names Check walkthrough, findings name
  // the correction path, and only the remaining cases name re-exploration.
  private raiseNextStep(): string {
    const plan = this.plan?.revision === this.revision?.id ? this.plan : undefined;
    if (plan?.state === 'pending') return 'Planning is running; the result appears here when the run settles.';
    if (plan?.state === 'completed' && plan.walkthroughs && !plan.review) return 'Use Check walkthrough to try again.';
    if (plan?.review?.consistency?.mismatches.length)
      return 'Correct or clarify the request above, then update the proposal and plan again before checking.';
    return 'Use Update proposal or Re-explore ideas to try again.';
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
      this.fundingNote = `Planning allowance is now ${wanted} calls (${used} used). ${this.raiseNextStep()}`;
    }, 'Raising allowance…');
  };
  private async explore(signal: AbortSignal) {
    const saved = await this.save('draft', this.revision?.repository || '0', signal);
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
      this.reconcileShelvedTasks(proposal);
      this.savedSnapshot = structuredClone(proposal);
      this.changeOpen = false;
      this.changeText = '';
      this.dirty = false;
      // Explore advances the head: re-persist the reconciled shelf to the
      // post-explore revision so an exact-base restore still finds it after
      // save→explore→reload. Same-browser recovery only (see reset()).
      this.rememberDraft();
      // The regenerated head invalidates staged history rows: clear them
      // and re-evaluate recovery for the new head.
      this.savedHistory = [];
      this.savedHistoryNote = '';
      this.savedHistorySkipped = 0;
      this.savedHistoryHead = '';
      this.maybeRecoverFromHistory();
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
    else if (error instanceof SodaRequestError && error.code === 'invalid_proposal_task_limit')
      this.error =
        'The derived work exceeds your selected issue limit. Your description is saved; review the planning limit or narrow the approach, then explore again. Limits never increase on their own.';
    else if (error instanceof SodaRequestError && error.code === 'invalid_proposal')
      this.error =
        'The derived proposal failed review validation. Your description is saved; review the approach and explore again.';
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
    const saved = this.savedSnapshot;
    if (
      this.dirty &&
      saved?.state === 'draft' &&
      this.revision?.id === saved.id &&
      this.objective.trim() === saved.objective &&
      this.department === (saved.department || '') &&
      this.revision.approach === saved.approach &&
      JSON.stringify(this.revision.ideas) === JSON.stringify(saved.ideas)
    ) {
      // A limit edit preserves the reviewed product scope and its task keys.
      // The new draft still needs its own native plan and explicit acceptance.
      void this.perform(async (signal) => {
        await this.saveCurrent(signal);
        await this.ensureAllowance(saved.intent, signal);
      }, 'Saving planning limits…');
      return;
    }
    void this.perform((signal) => this.explore(signal));
  };
  private decline = () => {
    void this.perform(async (signal) => {
      await this.save('declined', '0', signal);
      this.scopeGateArmed = false;
      this.shelvedTasks = [];
      this.shelvedPrunedDeps = {};
      this.pendingDropped = [];
      this.pendingRestored = [];
      this.savedHistory = [];
      this.savedHistoryNote = '';
      this.savedHistorySkipped = 0;
      this.savedHistoryHead = '';
      this.rememberDraft();
      this.revision = undefined;
      this.savedSnapshot = undefined;
      this.exploration = undefined;
      this.restoredReview = undefined;
      this.generation = undefined;
      this.generationError = '';
      this.generationIntent = '';
      this.plan = undefined;
      this.planNotice = '';
      this.planIntent = '';
      this.dirty = false;
      this.changeOpen = false;
      this.changeText = '';
    });
  };
  async acceptRepository(repository: string): Promise<boolean> {
    if (!reviewReady(this.view())) return false;
    return this.perform(async (signal) => {
      const current = this.revision;
      if (!current) throw Error('Product direction unavailable');
      if (current.state !== 'accepted' || current.repository !== repository) {
        if (current.repository !== repository) {
          const bound = await this.save('draft', repository, signal);
          this.dirty = false;
          this.dispatchEvent(
            new CustomEvent('intent-saved', {
              detail: {intent: bound.intent, revision: bound.id, state: bound.state},
              bubbles: true,
              composed: true,
            })
          );
          await this.startPlanning(bound, signal);
          return;
        }
        const planned = intentPlanStatus(
          await this.call('/api/factory/intents/' + current.intent + '/plan', undefined, signal)
        );
        if (planned.revision !== current.id) throw Error('Planning status answers another draft');
        this.plan = planned;
        if (planned.state !== 'completed') {
          this.plan = planned;
          if (['none', 'failed', 'cancelled'].includes(planned.state)) await this.planLiveAttempt(current, signal);
          else this.generationNotice = 'Your request is saved. Its planning progress and any question appear here.';
          return;
        }
        if (!planned.walkthroughs) {
          this.plan = planned;
          this.planNotice =
            'This saved plan has no request-specific walkthrough. Author one in a new plan before accepting.';
          return;
        }
        if (planned.walkthroughs && !planned.review) {
          const review = parseExploration(
            await this.call('/api/factory/intents/' + current.intent + '/plan/review', {revision: current.id}, signal)
          );
          check(
            review.content.revision === current.id &&
              review.recommendation?.revision === current.id &&
              review.consistency?.revision === current.id
          );
          this.plan = {...planned, review};
          const findings = review.consistency?.mismatches.length || 0;
          this.planNotice = findings
            ? 'The authored walkthrough needs a correction. Correct or clarify the request above (your objective and choices stay with your authorship), then plan again and check before accepting.'
            : 'JEV checked the authored walkthrough. Review it above, then accept to build.';
          // An explicit Check that blocks on findings lands focus on the
          // real findings; the near-top summary keeps the state, action and
          // a second link visible without scrolling.
          if (findings > 0) this.focusFindings();
          return;
        }
        if (planned.review?.consistency?.mismatches.length) {
          this.planNotice =
            'The authored walkthrough needs a correction. Correct or clarify the request above (your objective and choices stay with your authorship), then plan again and check before accepting.';
          return;
        }
        if (!planned.review?.consistency) {
          this.planNotice = 'Check the walkthrough again before accepting this direction.';
          return;
        }
      }
      if (this.revision?.state !== 'accepted' || this.revision.repository !== repository)
        await this.save('accepted', repository, signal);
      await this.generate(signal);
      this.dispatchEvent(
        new CustomEvent('intent-accepted', {
          detail: {intent: this.revision?.intent, revision: this.revision?.id, repository, state: 'accepted'},
          bubbles: true,
          composed: true,
        })
      );
    });
  }
  private async generate(signal: AbortSignal) {
    const accepted = this.revision;
    if (!accepted || accepted.state !== 'accepted') throw Error('Product direction not accepted');
    try {
      const body = await this.call('/api/factory/intents/' + accepted.intent + '/generate', {}, signal);
      if (planWait(body)) {
        this.generationNotice = generationWaitNotice(body);
        return;
      }
      const generation = object(body);
      check(generation.intent_id === accepted.intent && generation.revision_id === accepted.id);
      this.generationNotice = generationMessage(generation.state);
    } catch (error) {
      this.generationNotice = generationFailureNotice(error);
    }
  }
  private planWork = () => {
    const revision = this.revision;
    if (!revision || revision.state !== 'draft' || this.dirty) return;
    if (this.plan?.revision === revision.id && this.plan.state === 'pending') return;
    void this.perform((signal) => this.planLiveAttempt(revision, signal), 'Starting planning…');
  };
  // One settled-retry contract for both planning buttons: the retry
  // decision never comes from cached state. A failed fetch, a mismatched
  // revision, or a live pending/attention state stops with truthful
  // guidance and never POSTs. Only a verified settled terminal plan
  // (failed, cancelled, or completed without walkthroughs) authors a
  // successor draft; only a verified none plans this revision directly.
  // The save preserves the direction, choices and finite limits; the
  // settled history stays recorded and planning targets the new revision.
  private async planLiveAttempt(revision: IntentRevision, signal: AbortSignal) {
    let fresh;
    try {
      fresh = intentPlanStatus(await this.call('/api/factory/intents/' + revision.intent + '/plan', undefined, signal));
    } catch {
      if (signal.aborted) return;
      this.planNotice =
        'Planning status could not be verified. Your request is saved; refresh status before trying again.';
      return;
    }
    if (fresh.revision !== revision.id) {
      this.planNotice = 'Planning status answers another draft. Refresh status before trying again.';
      return;
    }
    this.plan = fresh;
    if (fresh.state === 'none') return this.startPlanning(revision, signal);
    if (
      fresh.state === 'failed' ||
      fresh.state === 'cancelled' ||
      (fresh.state === 'completed' && !fresh.walkthroughs)
    ) {
      const bound = await this.save('draft', revision.repository, signal);
      this.dirty = false;
      this.dispatchEvent(
        new CustomEvent('intent-saved', {
          detail: {intent: bound.intent, revision: bound.id, state: bound.state},
          bubbles: true,
          composed: true,
        })
      );
      await this.startPlanning(bound, signal);
      return;
    }
    if (fresh.state === 'pending') {
      this.planNotice = 'Your request is saved. Its planning progress and any question appear here.';
      return;
    }
    if (fresh.state === 'attention') {
      this.planNotice = fresh.attention
        ? `Planning needs a product decision: ${fresh.attention}`
        : 'Planning needs a product decision before issues can be proposed.';
      return;
    }
    // A completed plan with walkthroughs is ready for the review flow
    // below; the retry action leaves that state and its guidance alone.
  }
  private async startPlanning(revision: IntentRevision, signal: AbortSignal) {
    const operation = this.planOperation === 'decomposition_options' ? 'decomposition_options' : 'propose_intent';
    const path = '/api/factory/intents/' + revision.intent + '/plan';
    try {
      const outcome = await runPlanWork(
        () => this.call(path, planActionBody(revision.id, operation), signal),
        () => this.call(path, undefined, signal),
        revision.id
      );
      this.planNotice = outcome.notice;
      this.generationNotice = '';
      if (outcome.status !== undefined) {
        this.plan = outcome.status;
        this.planIntent = revision.id;
      }
    } catch (error) {
      if (signal.aborted) return;
      this.planNotice =
        error instanceof SodaRequestError && error.code === 'planning_unavailable'
          ? 'Planning is unavailable. Open Prepare project to check its environment and model account. Your request is saved.'
          : 'Planning could not be confirmed. Your request is saved; refresh status before trying again.';
    }
  }
  private prepareProject = () => {
    const revision = this.revision;
    if (!revision || revision.repository === '0' || this.busy || this.blocked) return;
    this.dispatchEvent(
      new CustomEvent('intent-preparation', {
        detail: {intent: revision.intent, repository: revision.repository},
        bubbles: true,
        composed: true,
      })
    );
  };
  private refreshPlanStatus = () => {
    this.planIntent = '';
    this.fetchPlanStatus(true);
  };
  private planAdmitted(revision: IntentRevision, refresh: boolean): boolean {
    if (!this.api || !this.active || this.blocked) return false;
    if (!refresh && this.planIntent === revision.id) return false;
    return true;
  }
  private fetchPlanStatus(refresh = false) {
    const revision = this.revision;
    if (!revision || revision.state !== 'draft' || revision.repository === '0' || !this.planAdmitted(revision, refresh))
      return;
    void this.loadPlanStatus(revision);
  }
  private loadPlanStatus(revision: IntentRevision) {
    const id = revision.id;
    void this.perform(async (signal) => {
      try {
        const status = intentPlanStatus(
          await this.call('/api/factory/intents/' + revision.intent + '/plan', undefined, signal)
        );
        if (status.revision !== id) throw Error('Mismatched plan status');
        if (this.revision?.id !== id || signal.aborted) return;
        this.planIntent = id;
        this.applyPolledPlanStatus(status, id);
      } catch {
        if (!signal.aborted) {
          this.planIntent = id;
          if (this.planNotice === '')
            this.planNotice = 'Planning status could not be loaded. Your direction is saved; try again.';
        }
      }
    }, 'Loading planning status…');
  }
  private continue = () => {
    const revision = this.revision;
    if (!revision || !reviewReady(this.view())) return;
    if (revision.repository !== '0') {
      void this.acceptRepository(revision.repository);
      return;
    }
    const resolved = this.resolvedScope(revision);
    this.dispatchEvent(
      new CustomEvent('intent-reviewed', {
        detail: {
          intent: revision.intent,
          revision: revision.id,
          approach: revision.approach,
          authorizedIdeas: resolved.length,
          tasks: revision.tasks.length,
          decisions: resolved.map((idea) => ({idea: idea.id, decision: idea.decision})),
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
    this.rememberDraft();
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
      generationNotice: this.generationNotice,
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
      allowanceError: this.allowanceError,
      retryAllowance: this.retryAllowance,
      setRaise: (value: number) => {
        this.raiseCalls = value;
        this.fundingNote = '';
      },
      raiseAllowance: this.raise,
      prepareProject: this.prepareProject,
      set: (field, value) => {
        this[field] = value;
        if ((field === 'objective' || field === 'department') && this.revision) this.dirty = true;
        this.rememberDraft();
      },
      setBound: (field, value) => {
        this[field] = value;
        if (field === 'planningCalls') this.userSetCalls = true;
        if (this.revision) this.dirty = true;
        this.rememberDraft();
      },
      decide: this.decide,
      toggleChange: this.toggleChange,
      correctProposal: this.correctProposal,
      refreshGeneration: () => this.fetchGeneration(true),
      chooseApproach: (approach) => {
        if (!this.revision) return;
        this.revision = {...this.revision, approach};
        this.dirty = true;
        this.rememberDraft();
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
      planOperation: this.planOperation,
      plan: this.plan?.revision === this.revision?.id ? this.plan : undefined,
      planNotice: this.planNotice,
      focusFindings: () => this.focusFindings(),
      setPlanOperation: (value: string) => {
        this.planOperation = value;
      },
      planWork: this.planWork,
      refreshPlanStatus: this.refreshPlanStatus,
    };
  }
  private reviewing() {
    const state = this.revision?.state;
    return !!this.revision?.ideas.length && (state === 'draft' || state === 'accepted');
  }
  protected render() {
    const view = this.view();
    const status = this.busy ? this.busyNote : this.error;
    return html`<div .inert=${this.departmentDialog}>
        ${this.reviewing() ? reviewIntent(view) : describeIntent(view)}
      </div>
      <p role="status" class="intent-live-status">${status ? html`<span>${status}</span>` : ''}</p>
      ${this.scopeRecovery()} ${this.declinedWorkRecovery()} ${this.savedHistoryRecovery()} ${this.unresolvedRepairReview()} ${departmentDialog(view)}`;
  }
}
customElements.define('soda-intent', SodaIntent);

// Saved-history traversal bound: the exact predecessor first, then at most
// this many predecessor hops back along the current head's chain. Anything
// older, off-chain, or meaning-changed is reported unavailable instead of
// merged.
const savedHistoryHopBound = 8;

const savedHistoryNoPredecessor =
  'This saved request names no earlier saved revision, so there is no saved history to preview. Only newly saved revisions can supply recovery.';
const savedHistoryUnavailable =
  'No saved history can restore this declined work. Only the exact saved predecessor chain with the same idea and its recorded work can supply it; unsent edits stay on this device and nothing was restored.';
const savedHistoryPartial =
  'Saved history holds some of the declined work above. The remaining ideas name no matching saved graph, so nothing more was restored.';

// One staged history row: the recorded tasks for a declined idea, repairs
// for recorded kept-edge cuts into the recovered set, edges refused as
// unprovable, and the source revision's idea snapshot the meaning checks
// ran against (re-verified at preview and re-accept time).
interface SavedHistoryRepair {
  owner: string;
  deps: string[];
  ideas: string[];
  depIdeas: Record<string, string[]>;
  targets: string[];
  resolved: boolean;
}
interface SavedHistoryCutEdge {
  key: string;
  objective: string;
  dep: string;
  kind: 'task' | 'repair';
}
interface SavedHistoryEntry {
  idea: string;
  revision: string;
  tasks: IntentTask[];
  repairs: SavedHistoryRepair[];
  cutEdges: SavedHistoryCutEdge[];
  ideas: ProductIdea[];
}

// Idea meaning for history recovery: identity plus the effective semantic
// content. A decision is workflow state, not meaning, so it is compared by
// the callers (the recovered idea is declined by definition); the edit
// correction is meaning and always counts.
//
// A correction missing on a non-edit side is the Decline/accept/pending
// cleanup (decidedIdea deletes it) of a duplicated description, not a human
// meaning change: the shared description still carries the effective
// meaning, so it stays matchable (FJ-453: a user-created
// change→Decline→save→cross-device recovery must work — correctProposal
// creates the idea with description === correction, so the Decline-deleted
// correction exactly duplicates the shared description). A deleted
// substantive correction (one that says something the shared description
// does not, e.g. historical “restricted to requester/manager” against
// shared “public access”) is a meaning change and stays a mismatch (FJ-457),
// as does a correction that differs while both sides stay edits or any
// description/consequence change. Nothing is merged or invented here.
function sameIdeaMeaning(past: ProductIdea, now: ProductIdea): boolean {
  if (past.description !== now.description || past.consequence !== now.consequence) return false;
  const pastCorrection = past.correction || '';
  const nowCorrection = now.correction || '';
  if (pastCorrection === nowCorrection) return true;
  // Both sides carry a correction yet disagree: a human changed the meaning.
  if (pastCorrection !== '' && nowCorrection !== '') return false;
  // Exactly one side is empty. Match only when the empty side is a non-edit
  // idea whose correction the workflow transition itself deleted AND the
  // deleted correction exactly duplicates the shared description; an edit
  // idea with an emptied correction is a human change, never a match, and
  // so is a deleted substantive correction.
  const emptySide = pastCorrection === '' ? past : now;
  const deletedCorrection = pastCorrection === '' ? nowCorrection : pastCorrection;
  return emptySide.decision !== 'edit' && deletedCorrection === past.description;
}
// A recorded task link is replayable only when the linked idea still exists
// with the same meaning and, beyond the recovered idea itself, is not
// declined in the current head.
function historyLinkOK(
  link: string,
  selected: string,
  past: Map<string, ProductIdea>,
  now: Map<string, ProductIdea>
): boolean {
  const was = past.get(link);
  const is = now.get(link);
  return (
    was !== undefined &&
    is !== undefined &&
    sameIdeaMeaning(was, is) &&
    (link === selected || is.decision !== 'decline')
  );
}

// Owner-private history read: full newest-first immutable revisions for one
// request. Strictly validated like any save: a malformed page degrades to
// the honest unavailable state, never to partial guesses. The bare endpoint
// serves the default page; no limit parameter is sent.
function intentHistory(value: unknown): IntentRevision[] {
  const raw = object(value);
  check(decisionID(raw.intent));
  const revisions = raw.revisions;
  check(Array.isArray(revisions) && revisions.length <= 64);
  return (revisions as unknown[]).map((entry) => intentRevision(entry));
}

// Shelf payload validation for draft persistence: a stored entry must
// carry the full task binding (key, objective, outcomes, idea links, and
// only string dependency edges) or it is dropped; repair edges must map
// task keys to string edge lists. Anything else leaves the in-memory shelf
// untouched.
function stringList(value: unknown, countLimit: number, itemLimit: number): value is string[] {
  return (
    Array.isArray(value) &&
    value.length <= countLimit &&
    value.every((item) => typeof item === 'string' && item.length > 0 && item.length <= itemLimit)
  );
}
function isShelvedTask(value: unknown): value is IntentTask {
  if (!value || typeof value !== 'object') return false;
  const task = value as Record<string, unknown>;
  if (!decisionID(task.key) || typeof task.objective !== 'string') return false;
  // Canonical task contract: up to 64 entries per list; outcomes carry up
  // to 4000 chars each while idea/dependency links carry up to 25 chars
  // each. Count and item length need separate limits — a single shared
  // limit would reject a valid 26+-link task and drop the entire shelf.
  if (!stringList(task.outcomes, 64, 4000) || !stringList(task.idea_ids, 64, 25)) return false;
  if (task.idea_ids.length === 0) return false;
  return task.depends_on === undefined || stringList(task.depends_on, 64, 25);
}
function isShelvedDeps(value: unknown): value is Record<string, ShelvedDepRepair> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  return Object.entries(value).every(([key, repair]) => {
    if (!decisionID(key) || !repair || typeof repair !== 'object') return false;
    const {deps, ideas, depIdeas} = repair as Record<string, unknown>;
    if (!Array.isArray(deps) || deps.length > 64 || !deps.every((dep) => decisionID(dep))) return false;
    if (!(Array.isArray(ideas) && ideas.length > 0 && ideas.length <= 50 && ideas.every((id) => decisionID(id))))
      return false;
    if (depIdeas === undefined) return true;
    if (!depIdeas || typeof depIdeas !== 'object' || Array.isArray(depIdeas)) return false;
    return Object.entries(depIdeas).every(
      ([dep, ids]) =>
        decisionID(dep) &&
        Array.isArray(ids) &&
        ids.length > 0 &&
        ids.length <= 50 &&
        ids.every((id) => decisionID(id))
    );
  });
}

// Lineage entries whose target is neither live nor shelved: the edge's
// successor has not (or not yet) come back, so the lineage stays pending
// for a later regeneration instead of being forgotten.
function pendingRepairLineage(
  repair: ShelvedDepRepair,
  live: Set<string>,
  shelved: Set<string>
): string[] {
  return Object.keys(repair.depIdeas || {}).filter((dep) => !live.has(dep) && !shelved.has(dep));
}

function copyRepair(repair: ShelvedDepRepair): ShelvedDepRepair {
  return {
    deps: [...repair.deps],
    ideas: [...repair.ideas],
    ...(repair.depIdeas
      ? {depIdeas: Object.fromEntries(Object.entries(repair.depIdeas).map(([dep, ids]) => [dep, [...ids]]))}
      : {}),
  };
}

// Pin stable lineage for every recorded target whose ideas are known: a
// later save or explore may re-key any of them — kept work regenerates
// under fresh keys, and shelved work returns under its stable key but may
// regenerate after that — and only pinned lineage lets the remap follow
// the edge. Retained entries never invent edges: the remap fires only for
// a unique idea-identical successor and stays cut otherwise.
function lineageForDeps(
  deps: string[],
  lookup: (dep: string) => string[] | undefined,
  prior?: Record<string, string[]>
): Record<string, string[]> {
  const depIdeas: Record<string, string[]> = {};
  if (prior) for (const [dep, ids] of Object.entries(prior)) depIdeas[dep] = [...ids];
  for (const dep of deps) {
    if (depIdeas[dep]) continue;
    const ids = lookup(dep);
    if (ids && ids.length > 0) depIdeas[dep] = [...ids];
  }
  return depIdeas;
}

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

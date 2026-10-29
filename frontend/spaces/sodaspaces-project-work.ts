import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, id, readSodaJSON, SodaRequestError} from './sodaspaces-api.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {intentList, intentPlanStatus} from './sodaspaces-intent-response.js';
import type {SavedRequest, IntentPlanStatus} from './sodaspaces-intent-response.js';
import {renderProjectWork} from './sodaspaces-project-work-view.js';
import {SodaFactoryVerificationAllowance} from './sodaspaces-factory-verification-allowance.js';
import './sodaspaces-factory-feed.js';
import './sodaspaces-factory-acceptance.js';
import './sodaspaces-factory-context-resolution.js';
import type {SodaFactoryContextResolution} from './sodaspaces-factory-context-resolution.js';
import './sodaspaces-factory-completion-retry.js';
import './sodaspaces-factory-work-journey.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import {factoryCommandId, factoryRunStatusResponse} from './sodaspaces-factory-response.js';
import type {DeliveryRepairProgress, FactoryRunDetail} from './sodaspaces-factory-response.js';
import {deliveryRepairDecision, factoryCommandRecord, retryDecision} from './sodaspaces-factory-work-response.js';
import {terminalID} from './sodaspaces-terminal-response.js';

export interface ProjectWorkContext {
  transport: PreparedExtensionMount;
  mode: 'project' | 'all-plans';
  repository?: {id: string; name: string};
  repositoryNames: ReadonlyMap<string, string>;
  account?: string;
  verificationRuns?: readonly string[];
}

// Plan replay reads run through a small worker pool so accumulated drafts
// never fan out into unbounded parallel GETs.
const planReadConcurrency = 4;

function sameContext(a: ProjectWorkContext | undefined, b: ProjectWorkContext | undefined) {
  if (!a || !b) return a === b;
  return (
    a.transport === b.transport && a.mode === b.mode && a.repository?.id === b.repository?.id && a.account === b.account
  );
}

function listPath(context: ProjectWorkContext, cursor: string | undefined) {
  const query = new URLSearchParams({limit: '20'});
  if (context.mode === 'project') {
    check(context.repository && id(context.repository.id));
    query.set('repository', context.repository.id);
  }
  if (cursor) query.set('cursor', cursor);
  return 'factory/intents?' + query.toString();
}

function appendPlans(current: readonly SavedRequest[], incoming: readonly SavedRequest[]) {
  const plans = new Map(current.map((plan) => [plan.intent, plan]));
  for (const plan of incoming) plans.set(plan.intent, plan);
  return [...plans.values()];
}

/** Owner-private work discovery. Visibility and context changes never perform a mutation. */
export class SodaProjectWork extends LitElement {
  static properties = {
    active: {type: Boolean},
    surface: {type: String},
    plans: {state: true},
    planStates: {state: true},
    loaded: {state: true},
    busy: {state: true},
    error: {state: true},
    next: {state: true},
    requirementsIssue: {state: true},
    issueInput: {state: true},
    issueError: {state: true},
  };
  declare active: boolean;
  declare surface: 'feed' | 'work';
  declare private plans: SavedRequest[];
  declare private planStates: Map<string, IntentPlanStatus>;
  declare private loaded: boolean;
  declare private busy: boolean;
  declare private error: string;
  declare private next: string | undefined;
  declare private requirementsIssue: string | undefined;
  declare private issueInput: string;
  declare private issueError: string;
  private current: ProjectWorkContext | undefined;
  private request: AbortController | undefined;

  constructor() {
    super();
    this.active = false;
    this.surface = 'feed';
    this.plans = [];
    this.planStates = new Map();
    this.loaded = this.busy = false;
    this.error = '';
    this.next = undefined;
    this.issueInput = '';
    this.issueError = '';
  }

  protected createRenderRoot() {
    return this;
  }

  set context(context: ProjectWorkContext | undefined) {
    const changed = !sameContext(context, this.current);
    this.current = context;
    if (changed) {
      this.cancelRead();
      this.clearPlans();
      this.requirementsIssue = undefined;
      this.issueInput = '';
      this.issueError = '';
    }
    this.requestUpdate();
    if (changed) void this.load(false);
  }

  connectedCallback() {
    super.connectedCallback();
    if (!this.loaded) void this.load(false);
  }

  disconnectedCallback() {
    this.cancelRead();
    this.current = undefined;
    this.clearPlans();
    super.disconnectedCallback();
  }

  protected updated(changed: PropertyValues<this>) {
    if (!changed.has('active')) return;
    if (!this.active) this.cancelRead();
    else if (!this.loaded) void this.load(false);
  }

  refresh() {
    this.loaded = false;
    void this.querySelector<SodaFactoryVerificationRetry>('soda-factory-verification-retry')?.refresh();
    for (const control of this.querySelectorAll<SodaFactoryVerificationAllowance>(
      'soda-factory-verification-allowance'
    ))
      void control.refresh();
    this.refreshResolution();
    return this.load(false);
  }

  private refreshResolution() {
    // Explicit child refresh (FJ-446): the resolution child skips identical
    // scope+transport, so genuine Work refresh/reopen re-invokes its read.
    // Stale guards live in the child (superseded tokens never write).
    this.querySelector<SodaFactoryContextResolution>('soda-factory-context-resolution')?.refresh();
  }

  private cancelRead() {
    this.request?.abort();
    this.request = undefined;
    this.busy = false;
  }

  private clearPlans() {
    this.plans = [];
    this.planStates = new Map();
    this.loaded = false;
    this.error = '';
    this.next = undefined;
  }

  private live(request: AbortController) {
    return this.request === request && !request.signal.aborted && this.active && this.isConnected;
  }

  private admitted() {
    return this.current && this.active && this.isConnected && !this.busy;
  }

  private async load(more: boolean) {
    const context = this.current;
    if (!context || !this.admitted() || (more && !this.next)) return;
    const request = new AbortController();
    this.request = request;
    this.busy = true;
    this.error = '';
    // Work-state pre-clear (FJ-v3-work-review-266): a full refresh REPLACES
    // the map from this pass, so retire attempted statuses synchronously for
    // the current context before awaiting the refresh. Stale Checked never
    // survives a slow sibling read. Request-identity guard preserves
    // account/project-change cancellation: a superseded request never
    // touches state (see readPlanStates early return). Paginated loads keep
    // retained entries and never pre-clear.
    if (!more && this.current === context && this.request === request) this.planStates = new Map();
    const timeout = window.setTimeout(() => request.abort(), 15000);
    try {
      await this.readPage(context, request, more);
    } catch {
      if (this.request === request && this.active) this.error = this.failureMessage(more);
    } finally {
      window.clearTimeout(timeout);
      if (this.request === request) {
        this.request = undefined;
        this.busy = false;
      }
    }
  }

  private async readPage(context: ProjectWorkContext, request: AbortController, more: boolean) {
    const response = await context.transport.request(listPath(context, more ? this.next : undefined), {
      method: 'GET',
      signal: request.signal,
    });
    if (!this.live(request)) return;
    this.admitResponse(response);
    const page = intentList(await readSodaJSON(response));
    if (!this.live(request)) return;
    this.applyPage(context, page, more);
    await this.readPlanStates(context, request, more, page.intents);
  }

  private applyPage(context: ProjectWorkContext, page: ReturnType<typeof intentList>, more: boolean) {
    if (context.mode === 'project') check(page.intents.every((plan) => plan.repository === context.repository?.id));
    this.plans = more ? appendPlans(this.plans, page.intents) : page.intents;
    this.next = page.next_cursor;
    this.loaded = true;
    this.retainPlanStates();
  }

  // Owner replay of the recorded plan/check state. One GET per bound draft,
  // same intent and revision guarded by the view mapping. Replay-only: no
  // POST, no PUT, no planning side effect. A missing plan, a mismatched
  // revision, an undecided run, or a failed read stays neutral and never
  // surfaces a list error.
  //
  // Freshness: a full refresh REPLACES the map from the reads that return in
  // this pass, keyed by exact revision. A 404/409/503, a mismatched
  // revision, or an unparseable replay resolves to unknown (no entry), so a
  // formerly Checked draft can never survive its own failed replay. A 401 or
  // 403 on any plan read routes through the existing authority-loss path.
  // A paginated load fetches only the newly arrived page and keeps the
  // retained entries for earlier pages. Retries are explicit and bounded:
  // one pass per Refresh, one pass per Load more, no background loop. Plan
  // reads run through a small worker pool so accumulated drafts never fan
  // out into unbounded parallel GETs.
  private async readPlanStates(
    context: ProjectWorkContext,
    request: AbortController,
    more: boolean,
    fresh: readonly SavedRequest[]
  ) {
    const scope = more ? fresh : this.plans;
    const candidates = scope.filter(
      (plan) => plan.state === 'draft' && plan.repository !== '0' && (!more || !this.planStates.has(plan.intent))
    );
    if (!candidates.length) {
      if (!more) this.planStates = new Map();
      else this.retainPlanStates();
      return;
    }
    const replayed = new Map<string, IntentPlanStatus>();
    let authority: 401 | 403 | undefined;
    let cursor = 0;
    const readOne = async (plan: SavedRequest): Promise<void> => {
      let response: Response;
      try {
        response = await context.transport.request(`factory/intents/${plan.intent}/plan`, {
          method: 'GET',
          signal: request.signal,
        });
      } catch {
        return;
      }
      if (response.status === 401 || response.status === 403) {
        authority = response.status;
        // Immediate authority loss for the current context: retire state
        // now instead of waiting for a hung sibling or the 15s deadline.
        // Request-identity guard preserves account/project-change
        // cancellation: a superseded request never touches state.
        // loseAuthority aborts the signal, releasing hung siblings.
        if (this.request === request) this.loseAuthority(response.status === 401);
        return;
      }
      if (!response.ok) return;
      try {
        const status = intentPlanStatus(await readSodaJSON(response));
        if (status.revision !== plan.revision) return;
        replayed.set(plan.intent, status);
      } catch {
        // An unparseable replay stays neutral.
      }
    };
    const lane = async (): Promise<void> => {
      for (;;) {
        if (cursor >= candidates.length || request.signal.aborted) return;
        const plan = candidates[cursor];
        cursor += 1;
        if (!plan) return;
        await readOne(plan);
      }
    };
    await Promise.all(Array.from({length: Math.min(planReadConcurrency, candidates.length)}, () => lane()));
    // Timeout-hole repair (FJ-v3-work-timeout-251): our own 15s deadline
    // aborts the signal while we stay current. A hung read then surfaces as
    // a readOne AbortError with !live, which must still replace the prior
    // map (hung/failed replays stay neutral) and must not discard a captured
    // 401/403. Only a superseded request (account/project change, which
    // retires this.request) returns without touching state.
    if (this.request !== request) return;
    if (authority !== undefined) {
      this.loseAuthority(authority === 401);
      return;
    }
    if (!request.signal.aborted && !this.live(request)) return;
    // Still current, timed out or fresh: replace from this pass so a hung
    // read can never retain its stale Checked entry.
    if (!more) this.planStates = replayed;
    else {
      const next = new Map(this.planStates);
      for (const [intent, status] of replayed) next.set(intent, status);
      this.planStates = next;
    }
    this.retainPlanStates();
  }

  private retainPlanStates() {
    const valid = new Map<string, IntentPlanStatus>();
    for (const plan of this.plans) {
      const status = this.planStates.get(plan.intent);
      if (status && status.revision === plan.revision) valid.set(plan.intent, status);
    }
    this.planStates = valid;
  }

  private admitResponse(response: Response) {
    if (response.ok) return;
    if (response.status === 401 || response.status === 403) this.loseAuthority(response.status === 401);
    throw new SodaRequestError(response.status);
  }

  private loseAuthority(reconnectRequired: boolean) {
    this.cancelRead();
    this.current = undefined;
    this.clearPlans();
    this.dispatchEvent(
      new CustomEvent('soda-work-authority-lost', {
        bubbles: true,
        composed: true,
        detail: {reconnectRequired},
      })
    );
  }

  private failureMessage(more: boolean) {
    if (more) return 'Could not load more plans. The plans already shown are kept.';
    if (this.plans.length) return 'Could not refresh plans. Showing previously loaded plans.';
    return 'Could not load your plans. Try refreshing the list.';
  }

  private resume(intent: string) {
    if (!this.current || !this.active || !this.plans.some((plan) => plan.intent === intent)) return;
    this.dispatchEvent(new CustomEvent('soda-work-resume', {bubbles: true, composed: true, detail: {intent}}));
  }

  private newWork() {
    const context = this.current;
    if (!context || !this.active) return;
    const repository = context.mode === 'project' ? context.repository?.id : undefined;
    this.dispatchEvent(
      new CustomEvent('soda-work-new', {
        bubbles: true,
        composed: true,
        detail: repository ? {repository} : {},
      })
    );
  }

  private openRequirements() {
    const issue = this.issueInput.trim();
    if (!id(issue)) {
      this.issueError = 'Enter the native issue number from this repository.';
      return;
    }
    this.issueError = '';
    if (this.requirementsIssue === issue) {
      // Same-issue reopen: scope+transport are identical so the child would
      // skip its read forever; explicitly re-invoke it instead of no-op.
      this.refreshResolution();
      return;
    }
    this.requirementsIssue = issue;
  }

  private renderRequirements(context: ProjectWorkContext) {
    const repository = context.repository;
    if (context.mode !== 'project' || !repository) return '';
    const repositoryPath = repository.name.startsWith('/') ? repository.name : `/${repository.name}`;
    if (!nativePath(repositoryPath) || repositoryPath.split('/').length !== 3) return '';
    return html`<details class="soda-project-issue-acceptance">
      <summary>Adopt or reaccept an issue</summary>
      <p>Review an imported issue or accept edited requirements before Factory continues its work.</p>
      <label
        >Native issue number
        <input
          inputmode="numeric"
          .value=${this.issueInput}
          ?disabled=${!this.active}
          @input=${(event: Event) => {
            this.issueInput = (event.target as HTMLInputElement).value;
          }}
      /></label>
      <button type="button" class="ui button" ?disabled=${!this.active} @click=${() => this.openRequirements()}>
        Open requirements review
      </button>
      <p role="status">${this.issueError}</p>
      ${
        this.requirementsIssue
          ? html`<soda-factory-work-journey
                .context=${this.active ? {transport: context.transport, repository: repository.id, repositoryPath, issue: this.requirementsIssue} : undefined}
              ></soda-factory-work-journey>
              <soda-factory-requirements-acceptance
                .context=${this.active ? {transport: context.transport, repository: repository.id, repositoryPath, issue: this.requirementsIssue} : undefined}
              ></soda-factory-requirements-acceptance>
              <soda-factory-context-resolution
                .context=${this.active ? {transport: context.transport, repository: repository.id, issue: this.requirementsIssue} : undefined}
              ></soda-factory-context-resolution>
              <soda-factory-completion-retry
                .context=${this.active ? {transport: context.transport, repository: repository.id, issue: this.requirementsIssue} : undefined}
              ></soda-factory-completion-retry>`
          : ''
      }
    </details>`;
  }

  protected render() {
    const context = this.current;
    if (!context) return html``;
    const feed =
      context.mode === 'project' && context.repository
        ? html`<section ?hidden=${this.surface !== 'feed'} aria-label="Project conversation">
            <header class="soda-work-intro">
              <div>
                <h2>Keep the work moving</h2>
                <p>Follow your team's decisions, ask a question, or describe the next change.</p>
              </div>
              <button type="button" class="ui primary button" ?disabled=${!this.active} @click=${() => this.newWork()}>
                Describe a change
              </button>
            </header>
            <soda-factory-feed
              .context=${this.active ? {transport: context.transport, repository: context.repository.id, repositoryName: context.repository.name, repositories: context.repositoryNames, account: context.account} : undefined}
            ></soda-factory-feed>
          </section>`
        : '';
    const plans = renderProjectWork({
      mode: context.mode,
      repositoryName: context.repository?.name || 'this project',
      repositoryNames: context.repositoryNames,
      plans: this.plans,
      planStatuses: this.planStates,
      loaded: this.loaded,
      busy: this.busy,
      error: this.error,
      more: this.next !== undefined,
      disabled: !this.active,
      verificationContext:
        this.active && context.mode === 'project' && context.repository
          ? {transport: context.transport, repository: context.repository.id}
          : undefined,
      newWork: () => this.newWork(),
      resume: (intent) => this.resume(intent),
      refresh: () => {
        void this.refresh();
      },
      loadMore: () => {
        void this.load(true);
      },
    });
    return html`${feed}
      <section ?hidden=${context.mode === 'project' && this.surface !== 'work'} aria-label="Plans and delivery">
        ${plans}${this.renderRequirements(context)}<soda-factory-verification-retry
          .context=${this.active && context.mode === 'project' && context.repository ? {transport: context.transport, repository: context.repository.id, runs: context.verificationRuns ?? []} : undefined}
        ></soda-factory-verification-retry>
      </section>`;
  }
}

customElements.define('soda-project-work', SodaProjectWork);

type VerificationRetryContext = {transport: PreparedExtensionMount; repository: string; runs: readonly string[]};

// Only the selected Project's recorded issue-zero runs are read. The server's
// exact delivery eligibility owns this control; a native exit alone is no verdict.
class SodaFactoryVerificationRetry extends LitElement {
  static properties = {entries: {state: true}, message: {state: true}, busy: {state: true}};
  declare private entries: FactoryRunDetail[];
  declare private message: string;
  declare private busy: boolean;
  private current: VerificationRetryContext | undefined;
  private serial = 0;
  private pending: {run: string; command: string; action: 'retry' | 'repair_delivery'} | undefined;
  private repairs = new Map<string, {cause: string; cycle: string}>();
  private refreshTimer: ReturnType<typeof setTimeout> | undefined;
  private reading = false;
  constructor() {
    super();
    this.entries = [];
    this.message = '';
    this.busy = false;
  }
  protected createRenderRoot() {
    return this;
  }
  set context(context: VerificationRetryContext | undefined) {
    if (
      context?.transport === this.current?.transport &&
      context?.repository === this.current?.repository &&
      context?.runs.join('|') === this.current?.runs.join('|')
    )
      return;
    this.current = context;
    clearTimeout(this.refreshTimer);
    this.serial++;
    this.entries = [];
    this.message = '';
    this.pending = undefined;
    this.repairs.clear();
    this.reading = false;
    this.busy = false;
    if (context) {
      check(context.runs.length <= 64 && context.runs.every(terminalID));
      void this.load(context, this.serial);
    }
  }
  disconnectedCallback() {
    clearTimeout(this.refreshTimer);
    super.disconnectedCallback();
    this.current = undefined;
    this.serial++;
  }
  refresh() {
    if (this.current) return this.load(this.current, this.serial);
  }
  private async load(context: VerificationRetryContext, serial: number, progressOnly = false) {
    if (this.reading) return;
    this.reading = true;
    clearTimeout(this.refreshTimer);
    const runs = progressOnly
      ? context.runs.filter(
          (run) =>
            this.repairs.has(run) ||
            this.entries.some((entry) => entry.id === run && this.unsettled(entry.delivery_repair?.progress))
        )
      : context.runs;
    const reads = await Promise.allSettled(
      runs.map(async (run) => {
        const response = await context.transport.request(`factory/runs/${run}`, {signal: AbortSignal.timeout(15000)});
        check(response.ok);
        const entry = factoryRunStatusResponse(await readSodaJSON(response), run, context.repository);
        const acknowledged = this.repairs.get(run),
          progress = entry.delivery_repair?.progress;
        if (acknowledged && progress)
          check(progress.cause_id === acknowledged.cause && progress.cycle_id === acknowledged.cycle);
        if (acknowledged && !progress) entry.delivery_repair = {available: false};
        return entry;
      })
    );
    if (serial !== this.serial) return;
    this.reading = false;
    const entries = reads.flatMap((read) =>
      read.status === 'fulfilled' &&
      read.value.reconciled &&
      read.value.view?.repository === context.repository &&
      read.value.view.issue === undefined &&
      (read.value.verification_retry?.available ||
        read.value.delivery_repair?.available ||
        read.value.delivery_repair?.progress ||
        this.repairs.has(read.value.id) ||
        this.pending?.run === read.value.id)
        ? [read.value]
        : []
    );
    const retained = progressOnly
      ? this.entries
      : this.entries.filter((entry) => reads[runs.indexOf(entry.id)]?.status === 'rejected');
    const current = new Map(retained.map((entry) => [entry.id, entry]));
    for (const entry of entries) current.set(entry.id, entry);
    this.entries = [...current.values()];
    for (const entry of entries) if (entry.delivery_repair?.progress) this.repairs.delete(entry.id);
    if (reads.some((read) => read.status === 'rejected')) {
      this.message =
        'Could not read current verification or repair progress. Last confirmed states are shown; refresh verification to try again.';
      return;
    }
    if (entries.some((entry) => entry.delivery_repair?.progress)) this.message = '';
    if (this.repairs.size || this.entries.some((entry) => this.unsettled(entry.delivery_repair?.progress)))
      this.refreshTimer = setTimeout(() => {
        if (this.current === context && this.isConnected) void this.load(context, serial, true);
      }, 4000);
  }
  private unsettled(progress: DeliveryRepairProgress | undefined) {
    return progress !== undefined && !['adopted', 'attention', 'stale'].includes(progress.state);
  }
  private async requestAction(run: string, action: 'retry' | 'repair_delivery') {
    const context = this.current;
    if (!context || this.busy || (!this.pending && !this.entries.some((entry) => entry.id === run))) return;
    const pending = this.pending ?? {run, command: factoryCommandId(), action};
    if (pending.run !== run || pending.action !== action) return;
    this.pending = pending;
    const serial = this.serial;
    this.busy = true;
    try {
      const response = await context.transport.request(`factory/runs/${run}/actions`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({command_id: pending.command, action: pending.action}),
        signal: AbortSignal.timeout(90000),
      });
      check(response.ok);
      const value = await readSodaJSON(response);
      let repair: ReturnType<typeof deliveryRepairDecision> | undefined;
      if (action === 'retry') retryDecision(value, pending.command, run);
      else repair = deliveryRepairDecision(value, pending.command, run);
      if (serial === this.serial) this.confirmed(run, action, repair);
    } catch {
      try {
        const response = await context.transport.request(`factory/commands/${pending.command}`, {
          signal: AbortSignal.timeout(15000),
        });
        check(response.ok);
        const record = factoryCommandRecord(await readSodaJSON(response), pending.command);
        check(
          record.type === (action === 'retry' ? 'retry' : 'request-delivery-repair') &&
            record.target === `run/${run}` &&
            typeof record.outcome === 'string'
        );
        let repair: ReturnType<typeof deliveryRepairDecision> | undefined;
        if (action === 'retry') retryDecision(JSON.parse(record.outcome), pending.command, run);
        else repair = deliveryRepairDecision(JSON.parse(record.outcome), pending.command, run);
        if (serial === this.serial) this.confirmed(run, action, repair);
      } catch {
        if (serial === this.serial)
          this.message =
            action === 'retry'
              ? 'No Retry was confirmed. Retry this same request or refresh; the original verification is retained.'
              : 'No repair request was confirmed. Retry this same request or refresh; the original verification is retained.';
      }
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private confirmed(
    run: string,
    action: 'retry' | 'repair_delivery',
    repair?: ReturnType<typeof deliveryRepairDecision>
  ) {
    this.pending = undefined;
    if (repair) {
      this.repairs.set(run, {cause: repair.cause_id, cycle: repair.cycle_id});
      this.entries = this.entries.map((entry) =>
        entry.id === run ? {...entry, delivery_repair: {available: false}} : entry
      );
    } else this.entries = this.entries.filter((entry) => entry.id !== run);
    this.message =
      action === 'retry'
        ? 'Verification Retry queued. The original result and spending are retained.'
        : 'Application repair requested. Diagnosis is queued; the original result and spending are retained.';
    if (repair && this.current) void this.load(this.current, this.serial);
  }
  private renderRepair(progress: DeliveryRepairProgress) {
    const states = {
      collecting: 'Collecting the recorded failure and current work.',
      ready: 'Strategist diagnosis is queued.',
      investigating: 'Strategist diagnosis is in progress.',
      assessing: 'Assessing the Strategist proposal.',
      applying: 'Adopting the qualified repair graph.',
      adopted: 'Repair graph adopted. Fresh readiness and application verification are still required.',
      attention: 'Application repair needs attention.',
      stale: 'Application repair inputs changed; the retained diagnosis is stale.',
    };
    return html`<section aria-label="Application repair progress">
      <p role="status">${states[progress.state]}</p>
      <p>
        Cause ${progress.cause_id.slice(0, 8)} · failed verification ${progress.failed_run.slice(0, 8)} · source
        ${progress.source_commit.slice(0, 8)}
      </p>
      ${progress.run_id ? html`<p>Strategist run ${progress.run_id.slice(0, 8)}</p>` : ''}
      <ul>
        ${progress.failures.map((failure) => html`<li>${failure.check_ref}: ${failure.detail}</li>`)}
      </ul>
      ${progress.failure_count > progress.failures.length ? html`<p>Showing ${progress.failures.length} of ${progress.failure_count} recorded failures.</p>` : ''}
      ${progress.judgment ? html`<p>Judgment: ${progress.judgment.option}; acceptance probability ${progress.judgment.accept_probability}; confidence ${progress.judgment.confidence}.</p>` : ''}
      ${progress.reason ? html`<p>${progress.reason}</p>` : ''}
    </section>`;
  }
  protected render() {
    if (!this.entries.length && !this.message) return html``;
    return html`<section aria-label="Application verification needs attention">
      ${this.entries.length ? html`<p>Previous verification results and spending stay recorded.</p>` : ''}
      ${this.entries.map(
        (entry) => html`<div>
          Verification run ${entry.id.slice(0, 8)}
          ${entry.verification_retry?.available || (this.pending?.run === entry.id && this.pending.action === 'retry') ? html`<button type="button" class="ui button" ?disabled=${this.busy || (!!this.pending && (this.pending.run !== entry.id || this.pending.action !== 'retry'))} @click=${() => this.requestAction(entry.id, 'retry')}>${this.pending?.run === entry.id && this.pending.action === 'retry' ? 'Retry same verification request' : 'Retry application verification'}</button>` : ''}
          ${entry.delivery_repair?.available || (this.pending?.run === entry.id && this.pending.action === 'repair_delivery') ? html`<button type="button" class="ui button" ?disabled=${this.busy || (!!this.pending && (this.pending.run !== entry.id || this.pending.action !== 'repair_delivery'))} @click=${() => this.requestAction(entry.id, 'repair_delivery')}>${this.pending?.run === entry.id && this.pending.action === 'repair_delivery' ? 'Retry same repair request' : 'Request application repair'}</button>` : ''}
          ${entry.delivery_repair?.progress ? this.renderRepair(entry.delivery_repair.progress) : ''}
        </div>`
      )}
      <p role="status">${this.message}</p>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy}
        @click=${() => {
          if (this.current) void this.load(this.current, this.serial);
        }}
      >
        Refresh verification
      </button>
    </section>`;
  }
}
customElements.define('soda-factory-verification-retry', SodaFactoryVerificationRetry);

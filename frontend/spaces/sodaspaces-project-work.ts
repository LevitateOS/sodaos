import {fountainAcceptanceLinks, fountainWorkLinks} from './soda-factory-links.js';
import {LitElement, html} from 'lit';
import {fountainCommandScope, fountainFeedContext} from './soda-factory-feed-host.js';
import type {FactoryFeedContext} from '@factory/product/frontend/factory/feed/feed.js';
import type {PropertyValues} from 'lit';
import {check, id, readSodaJSON, SodaRequestError} from '@factory/product/frontend/factory/api/models.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {intentList, intentPlanStatus} from '@factory/product/frontend/factory/api/request-response.js';
import type {SavedRequest, IntentPlanStatus} from '@factory/product/frontend/factory/api/request-response.js';
import {renderProjectWork} from './sodaspaces-project-work-view.js';
import '@factory/product/frontend/factory/work/verification-allowance.js';
import type {SodaFactoryVerificationAllowance} from '@factory/product/frontend/factory/work/verification-allowance.js';
import '@factory/product/frontend/factory/feed/feed.js';
import '@factory/product/frontend/factory/work/acceptance.js';
import '@factory/product/frontend/factory/work/context-resolution.js';
import type {SodaFactoryContextResolution} from '@factory/product/frontend/factory/work/context-resolution.js';
import '@factory/product/frontend/factory/work/completion-retry.js';
import '@factory/product/frontend/factory/work/journey.js';
import {nativePath} from '@factory/product/frontend/factory/shared/format.js';
import '@factory/product/frontend/factory/work/verification-retry.js';
import type {SodaFactoryVerificationRetry} from '@factory/product/frontend/factory/work/verification-retry.js';

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
  private incoming: ProjectWorkContext | undefined;
  private factoryFeed: FactoryFeedContext | undefined;
  private identityRead: AbortController | undefined;
  private identitySerial = 0;
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
    const changed = !sameContext(context, this.incoming);
    if (!changed) {
      // Inventory changes do not replace the admitted canonical account/repository
      // or reset a reviewed issue. Keep the latest run selection even while its
      // initial identity discovery is still pending.
      this.incoming = context;
      if (context && this.current && context.verificationRuns?.join('|') !== this.current.verificationRuns?.join('|')) {
        this.current = {...this.current, verificationRuns: context.verificationRuns ?? []};
        this.requestUpdate();
      }
      return;
    }
    this.incoming = context;
    if (changed) {
      this.identityRead?.abort();
      this.identityRead = undefined;
      this.identitySerial++;
      this.current = undefined;
      this.factoryFeed = undefined;
      this.cancelRead();
      this.clearPlans();
      this.requirementsIssue = undefined;
      this.issueInput = '';
      this.issueError = '';
    }
    this.requestUpdate();
    if (changed && context && this.active) void this.prepareContext(context);
  }

  private async prepareContext(context: ProjectWorkContext) {
    if (this.identityRead || this.incoming !== context || !this.active) return;
    const request = new AbortController();
    this.identityRead = request;
    const serial = ++this.identitySerial;
    try {
      const feed = await fountainFeedContext(
        context.transport,
        {
          repository: context.repository?.id ?? '',
          repositories: context.repositoryNames,
          ...(context.account ? {account: context.account} : {}),
          global: context.mode !== 'project',
        },
        request.signal
      );
      const labels = await feed.host.repositories([], request.signal);
      const latest = this.incoming;
      if (request.signal.aborted || serial !== this.identitySerial || !latest || !sameContext(latest, context)) return;
      this.factoryFeed = feed;
      this.current = {
        ...latest,
        ...(feed.account ? {account: feed.account} : {}),
        ...(context.repository ? {repository: {...context.repository, id: feed.repository}} : {}),
        repositoryNames: new Map([...labels.refs].map(([repository, ref]) => [String(repository), ref.label])),
      };
      await this.load(false);
    } catch {
      if (!request.signal.aborted && serial === this.identitySerial && sameContext(this.incoming, context))
        this.error = 'Repository information is unavailable.';
    } finally {
      if (serial === this.identitySerial) this.identityRead = undefined;
      this.requestUpdate();
    }
  }

  connectedCallback() {
    super.connectedCallback();
    if (this.incoming && !this.current) void this.prepareContext(this.incoming);
    else if (!this.loaded) void this.load(false);
  }

  disconnectedCallback() {
    this.identityRead?.abort();
    this.identityRead = undefined;
    this.identitySerial++;
    this.incoming = undefined;
    this.factoryFeed = undefined;
    this.cancelRead();
    this.current = undefined;
    this.clearPlans();
    super.disconnectedCallback();
  }

  protected updated(changed: PropertyValues<this>) {
    if (!changed.has('active')) return;
    if (!this.active) {
      this.identityRead?.abort();
      this.identityRead = undefined;
      this.identitySerial++;
      this.cancelRead();
    } else if (this.incoming && !this.current) void this.prepareContext(this.incoming);
    else if (!this.loaded) void this.load(false);
  }

  refresh() {
    if (!this.current && this.incoming) return this.prepareContext(this.incoming);
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
                .context=${this.active ? {transport: context.transport, repository: repository.id, links: fountainWorkLinks(repositoryPath), issue: this.requirementsIssue} : undefined}
              ></soda-factory-work-journey>
              <soda-factory-requirements-acceptance
                .context=${this.active ? {transport: context.transport, repository: repository.id, links: fountainAcceptanceLinks(repositoryPath), issue: this.requirementsIssue, commandScope: (signal: AbortSignal) => fountainCommandScope(context.transport, signal)} : undefined}
              ></soda-factory-requirements-acceptance>
              <soda-factory-context-resolution
                .context=${this.active ? {transport: context.transport, repository: repository.id, issue: this.requirementsIssue, commandScope: (signal: AbortSignal) => fountainCommandScope(context.transport, signal)} : undefined}
              ></soda-factory-context-resolution>
              <soda-factory-completion-retry
                .context=${this.active ? {transport: context.transport, repository: repository.id, issue: this.requirementsIssue, commandScope: (signal: AbortSignal) => fountainCommandScope(context.transport, signal)} : undefined}
              ></soda-factory-completion-retry>`
          : ''
      }
    </details>`;
  }

  protected render() {
    const context = this.current;
    if (!context)
      return this.incoming
        ? html`<p role=${this.error ? 'alert' : 'status'}>${this.error || 'Loading work…'}</p>`
        : html``;
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
            <soda-factory-feed .context=${this.active ? this.factoryFeed : undefined}></soda-factory-feed>
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
          ? {
              transport: context.transport,
              repository: context.repository.id,
              commandScope: (signal: AbortSignal) => fountainCommandScope(context.transport, signal),
            }
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
          .context=${this.active && context.mode === 'project' && context.repository ? {transport: context.transport, repository: context.repository.id, runs: context.verificationRuns ?? [], commandScope: (signal: AbortSignal) => fountainCommandScope(context.transport, signal)} : undefined}
        ></soda-factory-verification-retry>
      </section>`;
  }
}

customElements.define('soda-project-work', SodaProjectWork);

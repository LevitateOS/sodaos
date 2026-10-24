import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, id, readSodaJSON, SodaRequestError} from './sodaspaces-api.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {intentList} from './sodaspaces-intent-response.js';
import type {SavedRequest} from './sodaspaces-intent-response.js';
import {renderProjectWork} from './sodaspaces-project-work-view.js';
import {SodaFactoryVerificationAllowance} from './sodaspaces-factory-verification-allowance.js';
import './sodaspaces-factory-feed.js';
import './soda-providers.js';
import './sodaspaces-factory-acceptance.js';
import './sodaspaces-factory-context-resolution.js';
import './sodaspaces-factory-completion-retry.js';
import './sodaspaces-factory-work-journey.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import {factoryCommandId, factoryRunStatusResponse} from './sodaspaces-factory-response.js';
import type {FactoryRunDetail} from './sodaspaces-factory-response.js';
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

function sameContext(a: ProjectWorkContext | undefined, b: ProjectWorkContext | undefined) {
  if (!a || !b) return a === b;
  return a.transport === b.transport && a.mode === b.mode && a.repository?.id === b.repository?.id;
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
    plans: {state: true},
    loaded: {state: true},
    busy: {state: true},
    error: {state: true},
    next: {state: true},
    requirementsIssue: {state: true},
    issueInput: {state: true},
    issueError: {state: true},
  };
  declare active: boolean;
  declare private plans: SavedRequest[];
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
    this.plans = [];
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
    return this.load(false);
  }

  private cancelRead() {
    this.request?.abort();
    this.request = undefined;
    this.busy = false;
  }

  private clearPlans() {
    this.plans = [];
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
    if (this.live(request)) this.applyPage(context, page, more);
  }

  private applyPage(context: ProjectWorkContext, page: ReturnType<typeof intentList>, more: boolean) {
    if (context.mode === 'project') check(page.intents.every((plan) => plan.repository === context.repository?.id));
    this.plans = more ? appendPlans(this.plans, page.intents) : page.intents;
    this.next = page.next_cursor;
    this.loaded = true;
  }

  private admitResponse(response: Response) {
    if (response.ok) return;
    if (response.status === 401 || response.status === 403) {
      this.cancelRead();
      this.current = undefined;
      this.clearPlans();
      this.dispatchEvent(
        new CustomEvent('soda-work-authority-lost', {
          bubbles: true,
          composed: true,
          detail: {reconnectRequired: response.status === 401},
        })
      );
    }
    throw new SodaRequestError(response.status);
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
        ? html`<soda-factory-feed
            .context=${this.active ? {transport: context.transport, repository: context.repository.id, repositoryName: context.repository.name, repositories: context.repositoryNames, account: context.account} : undefined}
          ></soda-factory-feed>`
        : '';
    const plans = renderProjectWork({
      mode: context.mode,
      repositoryName: context.repository?.name || 'this project',
      repositoryNames: context.repositoryNames,
      plans: this.plans,
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
    return html`${feed}${this.renderRequirements(context)}<soda-factory-verification-retry
        .context=${this.active && context.mode === 'project' && context.repository ? {transport: context.transport, repository: context.repository.id, runs: context.verificationRuns ?? []} : undefined}
      ></soda-factory-verification-retry
      >${
        context.mode === 'project'
          ? html`<details class="soda-project-plans">
              <summary>My plans</summary>
              ${plans}
            </details>`
          : plans
      }`;
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
    this.serial++;
    this.entries = [];
    this.message = '';
    this.pending = undefined;
    this.busy = false;
    if (context) {
      check(context.runs.length <= 64 && context.runs.every(terminalID));
      void this.load(context, this.serial);
    }
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.current = undefined;
    this.serial++;
  }
  refresh() {
    if (this.current) return this.load(this.current, this.serial);
  }
  private async load(context: VerificationRetryContext, serial: number) {
    const reads = await Promise.allSettled(
      context.runs.map(async (run) => {
        const response = await context.transport.request(`factory/runs/${run}`, {signal: AbortSignal.timeout(15000)});
        check(response.ok);
        return factoryRunStatusResponse(await readSodaJSON(response), run, context.repository);
      })
    );
    if (serial !== this.serial) return;
    this.entries = reads.flatMap((read) =>
      read.status === 'fulfilled' &&
      read.value.reconciled &&
      read.value.view?.repository === context.repository &&
      read.value.view.issue === undefined &&
      (read.value.verification_retry?.available ||
        read.value.delivery_repair?.available ||
        this.pending?.run === read.value.id)
        ? [read.value]
        : []
    );
    if (reads.some((read) => read.status === 'rejected'))
      this.message = 'Could not read application verification availability. Refresh verification to try again.';
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
      if (action === 'retry') retryDecision(value, pending.command, run);
      else deliveryRepairDecision(value, pending.command, run);
      if (serial === this.serial) this.confirmed(run, action);
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
        if (action === 'retry') retryDecision(JSON.parse(record.outcome), pending.command, run);
        else deliveryRepairDecision(JSON.parse(record.outcome), pending.command, run);
        if (serial === this.serial) this.confirmed(run, action);
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
  private confirmed(run: string, action: 'retry' | 'repair_delivery') {
    this.pending = undefined;
    this.entries = this.entries.filter((entry) => entry.id !== run);
    this.message =
      action === 'retry'
        ? 'Verification Retry queued. The original result and spending are retained.'
        : 'Application repair requested. Diagnosis is queued; the original result and spending are retained.';
  }
  protected render() {
    if (!this.entries.length && !this.message) return html``;
    return html`<section aria-label="Application verification needs attention">
      ${this.entries.length ? html`<p>Choose an explicit next step. Previous verification results and spending stay recorded.</p>` : ''}
      ${this.entries.map(
        (entry) => html`<p>
          Verification run ${entry.id.slice(0, 8)}
          ${entry.verification_retry?.available || (this.pending?.run === entry.id && this.pending.action === 'retry') ? html`<button type="button" class="ui button" ?disabled=${this.busy || (!!this.pending && (this.pending.run !== entry.id || this.pending.action !== 'retry'))} @click=${() => this.requestAction(entry.id, 'retry')}>${this.pending?.run === entry.id && this.pending.action === 'retry' ? 'Retry same verification request' : 'Retry application verification'}</button>` : ''}
          ${entry.delivery_repair?.available || (this.pending?.run === entry.id && this.pending.action === 'repair_delivery') ? html`<button type="button" class="ui button" ?disabled=${this.busy || (!!this.pending && (this.pending.run !== entry.id || this.pending.action !== 'repair_delivery'))} @click=${() => this.requestAction(entry.id, 'repair_delivery')}>${this.pending?.run === entry.id && this.pending.action === 'repair_delivery' ? 'Retry same repair request' : 'Request application repair'}</button>` : ''}
        </p>`
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

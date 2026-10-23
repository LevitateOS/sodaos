import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, id, readSodaJSON, SodaRequestError} from './sodaspaces-api.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {intentList} from './sodaspaces-intent-response.js';
import type {SavedRequest} from './sodaspaces-intent-response.js';
import {renderProjectWork} from './sodaspaces-project-work-view.js';
import './sodaspaces-factory-feed.js';
import './soda-providers.js';
import './sodaspaces-factory-acceptance.js';
import {nativePath} from './sodaspaces-factory-activity.js';

export interface ProjectWorkContext {
  transport: PreparedExtensionMount;
  mode: 'project' | 'all-plans';
  repository?: {id: string; name: string};
  repositoryNames: ReadonlyMap<string, string>;
  account?: string;
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
          ? html`<soda-factory-requirements-acceptance
              .context=${this.active ? {transport: context.transport, repository: repository.id, repositoryPath, issue: this.requirementsIssue} : undefined}
            ></soda-factory-requirements-acceptance>`
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
      newWork: () => this.newWork(),
      resume: (intent) => this.resume(intent),
      refresh: () => {
        void this.refresh();
      },
      loadMore: () => {
        void this.load(true);
      },
    });
    return html`${feed}${this.renderRequirements(context)}${
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

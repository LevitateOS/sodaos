import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {applicationResponse, applicationsResponse} from './sodaspaces-application-response.js';
import type {ApplicationRecord} from './sodaspaces-application-response.js';
import {check, object, id} from './sodaspaces-api.js';

export type ApplicationRequest = (
  path: string,
  body?: Record<string, unknown>,
  signal?: AbortSignal
) => Promise<unknown>;

// A subordinate Spaces section, with the existing native mount's request and
// lifetime authority. Browsing never launches or installs an application.
export class SodaApplications extends LitElement {
  static properties = {
    request: {attribute: false},
    active: {type: Boolean},
    expanded: {type: Boolean},
    repository: {type: String},
    projects: {state: true},
    sessionKey: {attribute: false},
    records: {state: true},
    notice: {state: true},
    busy: {state: true},
    complete: {state: true},
    confirmations: {state: true},
  };
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
  declare expanded: boolean;
  declare repository: string;
  private projects: {repository: string; name: string}[] = [];
  declare sessionKey: string;
  declare private records: ApplicationRecord[];
  declare private notice: string;
  declare private busy: boolean;
  declare private complete: boolean;
  declare private confirmations: Set<string>;
  private controller: AbortController | undefined;
  private epoch = 0;
  constructor() {
    super();
    this.request = undefined;
    this.active = false;
    this.expanded = false;
    this.repository = '';
    this.sessionKey = '';
    this.records = [];
    this.notice = '';
    this.busy = false;
    this.complete = true;
    this.confirmations = new Set();
  }
  protected createRenderRoot() {
    return this;
  }
  protected willUpdate(changed: PropertyValues<this>) {
    if ((changed.has('active') && !this.active) || changed.has('sessionKey')) this.invalidate();
  }
  disconnectedCallback() {
    this.invalidate();
    super.disconnectedCallback();
  }
  private invalidate() {
    this.epoch++;
    this.controller?.abort();
    this.records = [];
    this.confirmations = new Set();
    this.projects = [];
    this.notice = '';
    this.busy = false;
  }
  private current(epoch: number) {
    return this.isConnected && this.active && this.epoch === epoch;
  }
  private async run<T>(
    read: (request: ApplicationRequest, signal: AbortSignal) => Promise<T>,
    apply: (result: T) => void,
    fail: () => void
  ) {
    if (!this.active || this.busy || !this.request) return;
    this.busy = true;
    this.controller = new AbortController();
    const epoch = ++this.epoch;
    try {
      const result = await read(this.request, this.controller.signal);
      if (!this.current(epoch)) return;
      apply(result);
    } catch {
      if (this.current(epoch)) fail();
    } finally {
      if (this.current(epoch)) this.busy = false;
    }
  }
  private refresh() {
    return this.run(
      async (request, signal) => applicationsResponse(await request('/api/applications', undefined, signal)),
      (result) => {
        this.records = result.applications;
        this.complete = result.complete;
        this.confirmations = new Set();
        this.notice = this.records.length ? '' : 'No applications are assigned to you.';
        void this.inspectAll();
      },
      () => {
        this.notice =
          'Could not refresh applications. Showing the last available results. Try Refresh after reconnecting.';
      }
    );
  }
  private async inspectAll() {
    // Status inspection reads the running app; it never installs or starts it.
    await this.updateComplete;
    const epoch = this.epoch;
    for (const record of this.records.filter(record => !this.repository || record.project?.repository === this.repository)) {
      if (!this.current(epoch) || !this.request) break;
      try {
        const result = applicationResponse(
          await this.request(
            `/api/applications/${record.application.id}/actions`,
            {
              action: 'inspect',
              revision: record.application.revision,
              expected_container: record.observation?.container ?? '',
              confirm_stop: false,
            },
            this.controller?.signal
          ),
          record.application
        );
        if (!this.current(epoch)) break;
        this.records = this.records.map((row) => (row.application.id === record.application.id ? result : row));
      } catch {
        if (this.current(epoch))
          this.notice = 'Some application status checks failed. Open the app or refresh to try again.';
      }
    }
  }
  private act(record: ApplicationRecord, action: 'inspect' | 'install' | 'start' | 'stop') {
    if (!this.records.includes(record)) return;
    if (action !== 'inspect' && action !== 'install' && !record.observation) return;
    if (action === 'stop' && !this.confirmations.has(record.application.id)) return;
    const body = {
      action,
      revision: record.application.revision,
      expected_container: record.observation?.container || '',
      confirm_stop: action === 'stop',
    };
    this.confirmations = new Set();
    this.notice = '';
    return this.run(
      async (request, signal) =>
        applicationResponse(
          await request(`/api/applications/${record.application.id}/actions`, body, signal),
          record.application
        ),
      (result) => {
        this.records = this.records.map((item) => (item === record ? result : item));
      },
      () => {
        this.records = this.records.map((item) => (item === record ? {...item, observation: null} : item));
        this.notice = 'Outcome unconfirmed. Inspect the application before acting again.';
      }
    );
  }
  private confirm(record: ApplicationRecord, event: Event) {
    if (!(event.target instanceof HTMLInputElement)) return;
    const next = new Set(this.confirmations);
    if (event.target.checked) next.add(record.application.id);
    else next.delete(record.application.id);
    this.confirmations = next;
  }
  private status(record: ApplicationRecord) {
    if (!record.observation) return 'State unknown';
    if (!record.observation.container) return 'Not installed';
    if (!record.observation.running) return 'Stopped';
    return record.observation.ready ? 'Ready' : 'Running; readiness unconfirmed';
  }
  private row(record: ApplicationRecord) {
    return html`<li>
      <header class="soda-application-heading"><h2>${record.application.name}</h2><span class="soda-application-status">Last observed: ${this.status(record)}</span></header>
      <p class="soda-application-origin">${record.application.origin}</p>
      ${record.project ? html`<p>Project: <a href=${record.project.path + '/workspace'}>${record.project.name}</a></p>` : html`<p>No project linked.</p>`}
      <details
        @toggle=${(event: Event) => {
          if ((event.currentTarget as HTMLDetailsElement).open && !this.projects.length) void this.loadProjects();
        }}
      >
        <summary>${record.project ? 'Change project' : 'Link to a project'}</summary>
        <label
          >Project<select
            aria-label=${'Project for ' + record.application.name}
            ?disabled=${this.blocked()}
            @change=${(event: Event) => {
              const select = event.target;
              if (select instanceof HTMLSelectElement && select.value) void this.linkProject(record, select.value);
            }}
          >
            <option value="">Choose a project</option>
            ${this.projects.map((project) => html`<option value=${project.repository} .selected=${project.repository === record.project?.repository}>${project.name}</option>`)}
          </select></label
        >
      </details>
      <div class="soda-application-actions">${this.openApplication(record)}
      ${record.operator && record.observation && !record.observation.container ? html`<button type="button" class="ui button" ?disabled=${this.blocked()} @click=${() => void this.act(record, 'install')}>Install ${record.application.name}</button>` : ''}
      <button
        type="button"
        class="ui button"
        ?disabled=${this.blocked()}
        @click=${() => void this.act(record, 'inspect')}
      >
        Inspect ${record.application.name}
      </button>
      </div>
      <details class="soda-application-runtime"><summary>Start or stop application</summary><div class="soda-application-runtime-actions">
      <button
        type="button"
        class="ui button"
        ?disabled=${this.startBlocked(record)}
        @click=${() => void this.act(record, 'start')}
      >
        Start ${record.application.name}
      </button>
      <label
        ><input
          type="checkbox"
          .checked=${this.confirmations.has(record.application.id)}
          ?disabled=${this.stopBlocked(record)}
          @change=${(event: Event) => this.confirm(record, event)}
        />
        Stop ${record.application.name} for everyone</label
      >
      <button
        type="button"
        class="ui button"
        ?disabled=${this.stopBlocked(record) || !this.confirmations.has(record.application.id)}
        @click=${() => void this.act(record, 'stop')}
      >
        Confirm Stop ${record.application.name}
      </button>
      </div></details>
    </li>`;
  }
  private blocked() {
    return this.busy || !this.active;
  }
  private startBlocked(record: ApplicationRecord) {
    return this.blocked() || !record.observation?.container || record.observation.running;
  }
  private stopBlocked(record: ApplicationRecord) {
    return this.blocked() || !record.observation?.running;
  }
  private openApplication(record: ApplicationRecord) {
    return html`<a class="ui primary button" href=${record.application.origin} target="_blank" rel="noopener noreferrer">Open application ↗</a>`;
  }
  private loadProjects() {
    return this.run(
      async (request, signal) => {
        const projects: {repository: string; name: string}[] = [];
        let after = '';
        do {
          const page = object(
            await request('/api/spaces' + (after ? '?after=' + encodeURIComponent(after) : ''), undefined, signal)
          );
          check(Array.isArray(page.items));
          for (const item of page.items) {
            const environment = object(object(item).environment);
            check(id(environment.repository_id) && typeof environment.repository === 'string');
            projects.push({repository: environment.repository_id, name: environment.repository});
          }
          const next = typeof page.next_after === 'string' ? page.next_after : '';
          if (next === after) break;
          after = next;
        } while (after);
        return projects;
      },
      (projects) => {
        this.projects = projects;
      },
      () => {
        this.notice = 'Could not load projects. Reopen the project selector to retry.';
      }
    );
  }
  private linkProject(record: ApplicationRecord, repository: string) {
    return this.run(
      async (request, signal) =>
        applicationResponse(
          await request('/api/applications/' + record.application.id + '/project', {repository}, signal),
          record.application
        ),
      (result) => {
        this.records = this.records.map((row) => (row.application.id === record.application.id ? result : row));
        this.notice = 'Project linked.';
      },
      () => {
        this.notice = 'Could not link the project. Refresh and try again.';
      }
    );
  }
  protected render() {
    return html`<details
      class="soda-applications"
      ?open=${this.expanded}
      aria-busy=${this.busy ? 'true' : 'false'}
      @toggle=${(event: Event) => {
        if ((event.currentTarget as HTMLDetailsElement).open && !this.records.length && !this.busy) void this.refresh();
      }}
    >
      <summary>Applications</summary>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy || !this.active}
        @click=${() => void this.refresh()}
      >
        Refresh applications
      </button>
      <p role="status">${this.notice}</p>
      ${this.repository && !this.records.some((record) => record.project?.repository === this.repository) && !this.busy ? html`<p>No applications are linked to this project yet. Open All applications to link an existing app.</p>` : ''}
      ${!this.complete ? html`<p>Some application details could not be loaded. Refresh to try again.</p>` : ''}
      <ul>
        ${repeat(
          this.repository
            ? this.records.filter((record) => record.project?.repository === this.repository)
            : this.records,
          (record) => record.application.id,
          (record) => this.row(record)
        )}
      </ul>
    </details>`;
  }
}
if (!customElements.get('soda-applications')) customElements.define('soda-applications', SodaApplications);

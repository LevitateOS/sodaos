import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {applicationResponse, applicationsResponse} from './sodaspaces-application-response.js';
import type {ApplicationRecord} from './sodaspaces-application-response.js';

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
    sessionKey: {attribute: false},
    records: {state: true},
    notice: {state: true},
    busy: {state: true},
    complete: {state: true},
    confirmations: {state: true},
  };
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
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
      },
      () => {
        this.records = [];
        this.notice = 'Application records could not be read. Refresh after reconnecting.';
      }
    );
  }
  private act(record: ApplicationRecord, action: 'inspect' | 'start' | 'stop') {
    if (!this.records.includes(record)) return;
    if (action !== 'inspect' && !record.observation) return;
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
    if (!record.observation.running) return 'Stopped';
    return record.observation.ready ? 'Ready' : 'Running; readiness unconfirmed';
  }
  private row(record: ApplicationRecord) {
    return html`<li>
      <strong>${record.application.name}</strong> · Last observed: ${this.status(record)}
      <p>${record.application.origin}</p>
      ${this.openApplication(record)}
      <button
        type="button"
        class="ui button"
        ?disabled=${this.blocked()}
        @click=${() => void this.act(record, 'inspect')}
      >
        Inspect ${record.application.name}
      </button>
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
    </li>`;
  }
  private blocked() {
    return this.busy || !this.active;
  }
  private startBlocked(record: ApplicationRecord) {
    return this.blocked() || !record.observation || record.observation.running;
  }
  private stopBlocked(record: ApplicationRecord) {
    return this.blocked() || !record.observation?.running;
  }
  private openApplication(record: ApplicationRecord) {
    return record.observation?.ready
      ? html`<a href=${record.application.origin} target="_blank" rel="noopener noreferrer">Open application</a>`
      : '';
  }
  protected render() {
    return html`<details class="soda-applications" aria-busy=${this.busy ? 'true' : 'false'}>
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
      ${!this.complete ? html`<p>Only the first 32 applications are shown. Ask the operator for the remaining records.</p>` : ''}
      <ul>
        ${repeat(
          this.records,
          (record) => record.application.id,
          (record) => this.row(record)
        )}
      </ul>
    </details>`;
  }
}
if (!customElements.get('soda-applications')) customElements.define('soda-applications', SodaApplications);

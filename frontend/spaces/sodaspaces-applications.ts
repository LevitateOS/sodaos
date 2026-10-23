import {LitElement, html} from 'lit';
import './sodaspaces-application-setup.js';
import type {PropertyValues} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {applicationResponse, applicationsResponse, applicationSnapshotsResponse} from './sodaspaces-application-response.js';
import type {ApplicationRecord, ApplicationSnapshotInventory} from './sodaspaces-application-response.js';
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
    backups: {state: true},
    restoreSelections: {state: true},
    restoreConfirmations: {state: true},
  };
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
  declare expanded: boolean;
  declare repository: string;
  declare private projects: {repository: string; name: string}[];
  declare sessionKey: string;
  declare private records: ApplicationRecord[];
  declare private notice: string;
  declare private busy: boolean;
  declare private complete: boolean;
  declare private confirmations: Set<string>;
  private controller: AbortController | undefined;
  private epoch = 0;
  declare private backups: Record<string, ApplicationSnapshotInventory>;
  private backupRequests = new Map<string, string>();
  declare private restoreSelections: Record<string, string>;
  declare private restoreConfirmations: Set<string>;
  private restoreRequests = new Map<string, string>();
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
    this.projects = [];
    this.backups = {};
    this.restoreSelections = {};
    this.restoreConfirmations = new Set();
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
    this.backups = {};
    this.backupRequests.clear();
    this.restoreSelections = {};
    this.restoreConfirmations = new Set();
    this.restoreRequests.clear();
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
        this.notice =
          'Could not refresh applications. Showing the last available results. Try Refresh after reconnecting.';
      }
    ).then(() => this.inspectAll());
  }
  private async inspectAll() {
    // Status inspection reads the running app; it never installs or starts it.
    if (this.busy || !this.active || !this.request) return;
    this.busy = true;
    await this.updateComplete;
    const epoch = this.epoch;
    for (const record of this.records.filter(record => !record.restore && (!this.repository || record.project?.repository === this.repository))) {
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
    if (this.current(epoch)) this.busy = false;
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
  private loadBackups(record: ApplicationRecord) {
    return this.run(
      async (request, signal) => applicationSnapshotsResponse(await request(`/api/applications/${record.application.id}/snapshots`, undefined, signal), record.application),
      result => { this.backups = {...this.backups, [record.application.id]: result}; },
      () => { this.notice = 'Could not read backups. Existing application data is retained; try Refresh backups.'; }
    );
  }
  private createBackup(record: ApplicationRecord) {
    if (!record.operator || !record.observation?.container || record.observation.running) return;
    let snapshot = this.backupRequests.get(record.application.id);
    if (!snapshot) {
      snapshot = 'a' + crypto.randomUUID().replaceAll('-', '').slice(0, 24);
      this.backupRequests.set(record.application.id, snapshot);
    }
    const body = {id: snapshot, revision: record.application.revision, expected_container: record.observation.container};
    return this.run(
      async (request, signal) => applicationSnapshotsResponse(await request(`/api/applications/${record.application.id}/snapshots`, body, signal), record.application),
      result => {
        this.backupRequests.delete(record.application.id);
        const previous = this.backups[record.application.id]?.snapshots ?? [];
        this.backups = {...this.backups, [record.application.id]: {snapshots: [...result.snapshots, ...previous.filter(row => row.id !== body.id)], complete: this.backups[record.application.id]?.complete ?? true}};
        this.notice = 'Stopped backup saved. Start the application when you are ready.';
      },
      () => { this.notice = 'Backup outcome unconfirmed. Refresh backups before retrying; this page retains the same backup request.'; }
    );
  }
  private backupView(record: ApplicationRecord) {
    if (!record.operator) return '';
    const inventory = this.backups[record.application.id];
    return html`<details class="soda-application-backups">
      <summary>Backups and recovery</summary>
      <p>Backups capture a stopped application. They include its accounts, files and scheduled work.</p>
      <button type="button" class="ui button" ?disabled=${this.blocked()} @click=${() => void this.loadBackups(record)}>Refresh backups for ${record.application.name}</button>
      <button type="button" class="ui button" ?disabled=${this.blocked() || !!record.restore || !record.observation?.container || record.observation.running} @click=${() => void this.createBackup(record)}>Back up ${record.application.name}</button>
      ${record.observation?.running ? html`<p>Stop the application before making a backup.</p>` : ''}
      ${inventory ? html`<ul>${inventory.snapshots.map(snapshot => html`<li><time datetime=${new Date(snapshot.created_at * 1000).toISOString()}>${new Date(snapshot.created_at * 1000).toLocaleString()}</time> · bundle ${snapshot.application.bundle_sha256.slice(0, 12)}</li>`)}</ul>${!inventory.snapshots.length ? html`<p>No completed backups are available.</p>` : ''}${!inventory.complete ? html`<p>The newest 64 backups are shown.</p>` : ''}` : ''}
      ${this.restoreView(record)}
    </details>`;
  }
  private restoreView(record: ApplicationRecord) {
    if (record.restore) return html`<p role="status">Restore is pending. Reconcile this operation before starting the application.</p>
      <button type="button" class="ui button" ?disabled=${this.blocked()} @click=${() => void this.restoreApplication(record)}>Reconcile Restore ${record.application.name}</button>
      <label><input type="checkbox" .checked=${this.restoreConfirmations.has(record.application.id + ':abort')} @change=${(event: Event) => this.confirmRestore(record.application.id + ':abort', event)}>Cancel this Restore and keep the previous application, if it has not been replaced.</label>
      <button type="button" class="ui button" ?disabled=${this.blocked() || !this.restoreConfirmations.has(record.application.id + ':abort')} @click=${() => void this.restoreApplication(record, true)}>Cancel Restore ${record.application.name}</button>`;
    const inventory = this.backups[record.application.id];
    if (!inventory?.snapshots.length) return '';
    return html`<p>Restore keeps this application's address and preserves its previous data separately. Writes after the selected backup will be absent from the restored app.</p>
      <label>Backup for ${record.application.name}<select aria-label=${'Backup for ' + record.application.name} .value=${this.restoreSelections[record.application.id] ?? ''} ?disabled=${this.blocked()} @change=${(event: Event) => {
        if (!(event.target instanceof HTMLSelectElement)) return;
        this.restoreSelections = {...this.restoreSelections, [record.application.id]: event.target.value};
        this.restoreConfirmations = new Set([...this.restoreConfirmations].filter(key => key !== record.application.id));
        this.restoreRequests.delete(record.application.id);
      }}><option value="">Choose a backup</option>${inventory.snapshots.map(snapshot => html`<option value=${snapshot.id} ?disabled=${snapshot.application.bundle_sha256 !== record.application.bundle_sha256 || snapshot.application.artifact_sha256 !== record.application.artifact_sha256}>${new Date(snapshot.created_at * 1000).toLocaleString()} · ${snapshot.application.bundle_sha256.slice(0, 12)}</option>`)}</select></label>
      <label><input type="checkbox" .checked=${this.restoreConfirmations.has(record.application.id)} ?disabled=${this.blocked()} @change=${(event: Event) => this.confirmRestore(record.application.id, event)}>Restore ${record.application.name} to this backup</label>
      <button type="button" class="ui button" ?disabled=${this.blocked() || !record.observation?.container || record.observation.running || !this.restoreSelections[record.application.id] || !this.restoreConfirmations.has(record.application.id)} @click=${() => void this.restoreApplication(record)}>Restore ${record.application.name}</button>`;
  }
  private confirmRestore(key: string, event: Event) {
    if (!(event.target instanceof HTMLInputElement)) return;
    const confirmations = new Set(this.restoreConfirmations);
    if (event.target.checked) confirmations.add(key); else confirmations.delete(key);
    this.restoreConfirmations = confirmations;
  }
  private restoreApplication(record: ApplicationRecord, abort = false) {
    if (!record.operator) return;
    const pending = record.restore;
    if ((!pending && (!this.restoreConfirmations.has(record.application.id) || !record.observation?.container || record.observation.running)) || (abort && (!pending || !this.restoreConfirmations.has(record.application.id + ':abort')))) return;
    const snapshot = pending?.snapshot ?? this.restoreSelections[record.application.id];
    if (!snapshot) return;
    let operation = pending?.id ?? this.restoreRequests.get(record.application.id);
    if (!operation) { operation = 'a' + crypto.randomUUID().replaceAll('-', '').slice(0, 24); this.restoreRequests.set(record.application.id, operation); }
    const body = {action: abort ? 'abort' : 'restore', id: operation, snapshot, revision: pending?.previous.revision ?? record.application.revision, expected_container: pending?.expected_container ?? record.observation?.container ?? '', confirm: true};
    return this.run(
      async (request, signal) => {
        const result = applicationResponse(await request(`/api/applications/${record.application.id}/restores`, body, signal));
        check(result.application.id === record.application.id && result.application.origin === record.application.origin);
        return result;
      },
      result => {
        this.records = this.records.map(row => row.application.id === record.application.id ? result : row);
        this.restoreRequests.delete(record.application.id);
        this.restoreConfirmations = new Set();
        this.notice = abort ? 'Restore cancelled. The previous application remains stopped.' : 'Restored to fresh data at the same address. Start the application when you are ready.';
      },
      () => { this.records = this.records.map(row => row === record ? {...row, observation: null} : row); this.notice = 'Restore outcome unconfirmed. Refresh applications and reconcile the same operation before acting again.'; }
    );
  }
  private status(record: ApplicationRecord) {
    if (record.restore) return 'Restore pending';
    if (!record.observation) return 'State unknown';
    if (!record.observation.container) return 'Not installed';
    if (!record.observation.running) return 'Stopped';
    return record.observation.ready ? 'Ready' : 'Running; readiness unconfirmed';
  }
  private row(record: ApplicationRecord) {
    return html`<li>
      <header class="soda-application-heading"><h2>${record.application.name}</h2><span class="soda-application-status">Last observed: ${this.status(record)}</span></header>
      <p class="soda-application-origin">${record.application.origin}</p>
      <p>Runtime limits: 1 GiB memory, up to two CPUs' worth of CPU time, and 128 processes or threads.</p>
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
        ?disabled=${this.blocked() || !!record.restore}
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
      ${this.backupView(record)}
    </li>`;
  }
  private blocked() {
    return this.busy || !this.active;
  }
  private startBlocked(record: ApplicationRecord) {
    return this.blocked() || !!record.restore || !record.observation?.container || record.observation.running;
  }
  private stopBlocked(record: ApplicationRecord) {
    return this.blocked() || !!record.restore || !record.observation?.running;
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
      ${!this.repository ? html`<soda-application-setup .request=${this.request} .active=${this.active} .sessionKey=${this.sessionKey} @application-setup-complete=${() => void this.refresh()}></soda-application-setup>` : ''}
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

import {LitElement, html} from 'lit';
import './sodaspaces-application-setup.js';
import type {SodaApplicationSetup} from './sodaspaces-application-setup.js';
import './sodaspaces-application-delivery.js';
import './sodaspaces-application-surface.js';
import type {PropertyValues} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {
  applicationResponse,
  applicationsResponse,
  applicationSnapshotsResponse,
} from './sodaspaces-application-response.js';
import type {
  ApplicationRecord,
  ApplicationLaunchRecord,
  ApplicationSnapshotInventory,
  ApplicationEnvironment,
} from './sodaspaces-application-response.js';
import {check, object, id, SodaRequestError} from './sodaspaces-api.js';
import type {ApplicationState} from './sodaspaces-application-surface.js';

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
    requestedDelivery: {type: String},
    requestedCompanyScope: {type: Boolean},
    companyScope: {state: true},
    projects: {state: true},
    sessionKey: {attribute: false},
    records: {state: true},
    launchable: {state: true},
    canSetup: {state: true},
    notice: {state: true},
    busy: {state: true},
    complete: {state: true},
    loaded: {state: true},
    confirmations: {state: true},
    backups: {state: true},
    restoreSelections: {state: true},
    restoreConfirmations: {state: true},
  };
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
  declare expanded: boolean;
  declare repository: string;
  declare requestedDelivery: string;
  declare requestedCompanyScope: boolean;
  declare private companyScope: boolean;
  declare private projects: {repository: string; name: string}[];
  declare sessionKey: string;
  declare private records: ApplicationRecord[];
  declare private launchable: ApplicationLaunchRecord[];
  declare private canSetup: boolean;
  declare private notice: string;
  declare private busy: boolean;
  declare private complete: boolean;
  declare private loaded: boolean;
  declare private confirmations: Set<string>;
  private controller: AbortController | undefined;
  private epoch = 0;
  private setupCompleteQueued = false;
  private setupCompleteSettling = false;
  private setupCompletePending: (string | undefined)[] = [];
  declare private backups: Record<string, ApplicationSnapshotInventory>;
  private backupRequests = new Map<string, string>();
  private setupCompleteBusy = false;
  declare private restoreSelections: Record<string, string>;
  declare private restoreConfirmations: Set<string>;
  private restoreRequests = new Map<string, string>();
  constructor() {
    super();
    this.request = undefined;
    this.active = false;
    this.expanded = false;
    this.repository = '';
    this.requestedDelivery = '';
    this.requestedCompanyScope = false;
    this.companyScope = false;
    this.sessionKey = '';
    this.records = [];
    this.launchable = [];
    this.canSetup = false;
    this.notice = '';
    this.busy = false;
    this.complete = true;
    this.loaded = false;
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
    if ((changed.has('active') && !this.active) || changed.has('sessionKey') || changed.has('repository'))
      this.invalidate();
    // A newly selected exact delivery leaves the company surface: the scope
    // opened below only applies while no exact delivery is selected.
    if (changed.has('requestedDelivery') && this.requestedDelivery) this.companyScope = false;
    // A restored company selection re-scopes the fresh mount. The exact
    // delivery still wins: the guard above and the delivery check here keep a
    // stale scope from ever filtering an exact view.
    if (changed.has('requestedCompanyScope') && this.requestedCompanyScope && !this.requestedDelivery)
      this.companyScope = true;
    // A suspend drops the live scope with the discarded inventory, but the
    // retained selection is unchanged: resuming re-applies it even though the
    // request flag never changed. The exact delivery still wins, and a
    // session or repository switch never re-applies: only the inactive to
    // active transition carries the retained selection forward.
    if (changed.has('active') && this.active && this.requestedCompanyScope && !this.requestedDelivery)
      this.companyScope = true;
  }
  protected updated(changed: PropertyValues<this>) {
    if (
      this.active &&
      this.expanded &&
      (changed.has('active') ||
        changed.has('sessionKey') ||
        changed.has('repository') ||
        changed.has('expanded') ||
        changed.has('requestedDelivery') ||
        changed.has('request'))
    )
      void this.refresh();
  }
  disconnectedCallback() {
    this.invalidate();
    super.disconnectedCallback();
  }
  private invalidate() {
    this.epoch++;
    this.controller?.abort();
    // The company surface is a selection over the discarded inventory, never
    // a persistent filter: a context switch drops it so the next context
    // never inherits a scope that hides its previews. A browser reload starts
    // unscoped for the same reason; reopen the company application there.
    this.companyScope = false;
    this.records = [];
    this.launchable = [];
    this.canSetup = false;
    this.confirmations = new Set();
    this.projects = [];
    this.backups = {};
    this.backupRequests.clear();
    this.restoreSelections = {};
    this.restoreConfirmations = new Set();
    this.restoreRequests.clear();
    this.notice = '';
    this.busy = false;
    this.loaded = false;
    this.complete = true;
    this.setupCompleteBusy = false;
    this.setupCompleteQueued = false;
    this.setupCompleteSettling = false;
    this.setupCompletePending = [];
  }
  private current(epoch: number) {
    return this.isConnected && this.active && this.epoch === epoch;
  }
  private async run<T>(
    read: (request: ApplicationRequest, signal: AbortSignal) => Promise<T>,
    apply: (result: T) => void,
    fail: (status?: number) => void
  ) {
    if (!this.active || this.busy || !this.request) return;
    this.busy = true;
    this.controller = new AbortController();
    const epoch = ++this.epoch;
    try {
      const result = await read(this.request, this.controller.signal);
      if (!this.current(epoch)) return;
      apply(result);
    } catch (error) {
      // An expired Soda session returns 401: reconnect, then return to the
      // same application. No personal data is retained from the old session.
      if (this.current(epoch)) {
        const status = error instanceof SodaRequestError ? error.status : undefined;
        if (status === 401) {
          this.records = [];
          this.launchable = [];
          this.canSetup = false;
          this.loaded = false;
        }
        fail(status);
      }
    } finally {
      if (this.current(epoch)) this.busy = false;
      // A completion that arrived while this read was in flight owes exactly
      // one post-completion refresh; re-enter its settle once idle.
      if (this.setupCompleteBusy) void this.settleSetupComplete();
    }
  }
  private refresh() {
    return this.run(
      async (request, signal) => applicationsResponse(await request('/api/applications', undefined, signal)),
      (result) => {
        this.records = result.applications;
        this.launchable = result.launchable;
        this.canSetup = result.can_setup;
        this.complete = result.complete;
        this.loaded = true;
        this.confirmations = new Set();
        this.notice = '';
      },
      (status) => {
        this.notice =
          status === 401
            ? 'Your Soda session changed. Reconnect, then return to the same application; no personal data is retained from the old session.'
            : this.loaded
              ? 'Could not refresh applications. Showing the last available status. Refresh applications to try again.'
              : 'Applications could not be loaded. Reconnect to Soda Forge, then refresh applications.';
      }
    ).then(() => this.inspectAll());
  }
  private async inspectAll() {
    // Status inspection reads the running app; it never installs or starts it.
    if (this.busy || !this.active || !this.request) return;
    this.busy = true;
    await this.updateComplete;
    const epoch = this.epoch;
    for (const record of this.records.filter(
      (record) => !record.restore && (!this.repository || record.project?.repository === this.repository)
    )) {
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
        if (!this.current(epoch)) break;
        // A failed status read leaves the runtime unknown: clear the stale
        // prior observation so a formerly Ready app cannot keep its iframe or
        // Start–Stop enabled on unconfirmed data (same as per-record Inspect).
        this.records = this.records.map((row) =>
          row.application.id === record.application.id ? {...row, observation: null} : row
        );
        this.notice = 'Some application status checks failed. Open the app or refresh to try again.';
      }
    }
    if (this.current(epoch)) this.busy = false;
    // The status sweep is the second half of an inventory refresh: its tail
    // also releases a completion that arrived mid-read.
    if (this.setupCompleteBusy) void this.settleSetupComplete();
  }
  private act(record: ApplicationRecord, action: 'inspect' | 'install' | 'start' | 'stop' | 'verify_build') {
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
        if (action === 'verify_build')
          this.notice = result.delivery
            ? 'Build matches the verified project delivery.'
            : 'Build bytes checked. No matching verified project delivery is recorded.';
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
      async (request, signal) =>
        applicationSnapshotsResponse(
          await request(`/api/applications/${record.application.id}/snapshots`, undefined, signal),
          record.application
        ),
      (result) => {
        this.backups = {...this.backups, [record.application.id]: result};
      },
      () => {
        this.notice = 'Could not read backups. Existing application data is retained; try Refresh backups.';
      }
    );
  }
  private createBackup(record: ApplicationRecord) {
    if (!record.operator || !record.observation?.container || record.observation.running) return;
    let snapshot = this.backupRequests.get(record.application.id);
    if (!snapshot) {
      snapshot = 'a' + crypto.randomUUID().replaceAll('-', '').slice(0, 24);
      this.backupRequests.set(record.application.id, snapshot);
    }
    const body = {
      id: snapshot,
      revision: record.application.revision,
      expected_container: record.observation.container,
    };
    return this.run(
      async (request, signal) =>
        applicationSnapshotsResponse(
          await request(`/api/applications/${record.application.id}/snapshots`, body, signal),
          record.application
        ),
      (result) => {
        this.backupRequests.delete(record.application.id);
        const previous = this.backups[record.application.id]?.snapshots ?? [];
        this.backups = {
          ...this.backups,
          [record.application.id]: {
            snapshots: [...result.snapshots, ...previous.filter((row) => row.id !== body.id)],
            complete: this.backups[record.application.id]?.complete ?? true,
          },
        };
        this.notice = 'Stopped backup saved. Start the application when you are ready.';
      },
      () => {
        this.notice =
          'Backup outcome unconfirmed. Refresh backups before retrying; this page retains the same backup request.';
      }
    );
  }
  private backupView(record: ApplicationRecord) {
    if (!record.operator) return '';
    const inventory = this.backups[record.application.id];
    return html`<details class="soda-application-backups">
      <summary>Backups and recovery</summary>
      <p>Backups capture a stopped application. They include its accounts, files and scheduled work.</p>
      <button type="button" class="ui button" ?disabled=${this.blocked()} @click=${() => void this.loadBackups(record)}>
        Refresh backups for ${record.application.name}
      </button>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.blocked() || !!record.restore || !record.observation?.container || record.observation.running}
        @click=${() => void this.createBackup(record)}
      >
        Back up ${record.application.name}
      </button>
      ${record.observation?.running ? html`<p>Stop the application before making a backup.</p>` : ''}
      ${
        inventory
          ? html`<ul>
                ${inventory.snapshots.map((snapshot) => html`<li><time datetime=${new Date(snapshot.created_at * 1000).toISOString()}>${new Date(snapshot.created_at * 1000).toLocaleString()}</time> · bundle ${snapshot.application.bundle_sha256.slice(0, 12)}</li>`)}
              </ul>
              ${!inventory.snapshots.length ? html`<p>No completed backups are available.</p>` : ''}${!inventory.complete ? html`<p>The newest 64 backups are shown.</p>` : ''}`
          : ''
      }
      ${this.restoreView(record)}
    </details>`;
  }
  private restoreView(record: ApplicationRecord) {
    if (record.restore)
      return html`<p role="status">Restore is pending. Reconcile this operation before starting the application.</p>
        <button
          type="button"
          class="ui button"
          ?disabled=${this.blocked()}
          @click=${() => void this.restoreApplication(record)}
        >
          Reconcile Restore ${record.application.name}
        </button>
        <label
          ><input
            type="checkbox"
            .checked=${this.restoreConfirmations.has(record.application.id + ':abort')}
            @change=${(event: Event) => this.confirmRestore(record.application.id + ':abort', event)}
          />Cancel this Restore and keep the previous application, if it has not been replaced.</label
        >
        <button
          type="button"
          class="ui button"
          ?disabled=${this.blocked() || !this.restoreConfirmations.has(record.application.id + ':abort')}
          @click=${() => void this.restoreApplication(record, true)}
        >
          Cancel Restore ${record.application.name}
        </button>`;
    const inventory = this.backups[record.application.id];
    if (!inventory?.snapshots.length) return '';
    return html`<p>
        Restore keeps this application's address and preserves its previous data separately. Writes after the selected
        backup will be absent from the restored app.
      </p>
      <label
        >Backup for ${record.application.name}<select
          aria-label=${'Backup for ' + record.application.name}
          .value=${this.restoreSelections[record.application.id] ?? ''}
          ?disabled=${this.blocked()}
          @change=${(event: Event) => {
            if (!(event.target instanceof HTMLSelectElement)) return;
            this.restoreSelections = {...this.restoreSelections, [record.application.id]: event.target.value};
            this.restoreConfirmations = new Set(
              [...this.restoreConfirmations].filter((key) => key !== record.application.id)
            );
            this.restoreRequests.delete(record.application.id);
          }}
        >
          <option value="">Choose a backup</option>
          ${inventory.snapshots.map((snapshot) => html`<option value=${snapshot.id} ?disabled=${snapshot.application.bundle_sha256 !== record.application.bundle_sha256 || snapshot.application.artifact_sha256 !== record.application.artifact_sha256}>${new Date(snapshot.created_at * 1000).toLocaleString()} · ${snapshot.application.bundle_sha256.slice(0, 12)}</option>`)}
        </select></label
      >
      <label
        ><input
          type="checkbox"
          .checked=${this.restoreConfirmations.has(record.application.id)}
          ?disabled=${this.blocked()}
          @change=${(event: Event) => this.confirmRestore(record.application.id, event)}
        />Restore ${record.application.name} to this backup</label
      >
      <button
        type="button"
        class="ui button"
        ?disabled=${this.blocked() || !record.observation?.container || record.observation.running || !this.restoreSelections[record.application.id] || !this.restoreConfirmations.has(record.application.id)}
        @click=${() => void this.restoreApplication(record)}
      >
        Restore ${record.application.name}
      </button>`;
  }
  private confirmRestore(key: string, event: Event) {
    if (!(event.target instanceof HTMLInputElement)) return;
    const confirmations = new Set(this.restoreConfirmations);
    if (event.target.checked) confirmations.add(key);
    else confirmations.delete(key);
    this.restoreConfirmations = confirmations;
  }
  private restoreApplication(record: ApplicationRecord, abort = false) {
    if (!record.operator) return;
    const pending = record.restore;
    if (
      (!pending &&
        (!this.restoreConfirmations.has(record.application.id) ||
          !record.observation?.container ||
          record.observation.running)) ||
      (abort && (!pending || !this.restoreConfirmations.has(record.application.id + ':abort')))
    )
      return;
    const snapshot = pending?.snapshot ?? this.restoreSelections[record.application.id];
    if (!snapshot) return;
    let operation = pending?.id ?? this.restoreRequests.get(record.application.id);
    if (!operation) {
      operation = 'a' + crypto.randomUUID().replaceAll('-', '').slice(0, 24);
      this.restoreRequests.set(record.application.id, operation);
    }
    const body = {
      action: abort ? 'abort' : 'restore',
      id: operation,
      snapshot,
      revision: pending?.previous.revision ?? record.application.revision,
      expected_container: pending?.expected_container ?? record.observation?.container ?? '',
      confirm: true,
    };
    return this.run(
      async (request, signal) => {
        const result = applicationResponse(
          await request(`/api/applications/${record.application.id}/restores`, body, signal)
        );
        check(
          result.application.id === record.application.id && result.application.origin === record.application.origin
        );
        return result;
      },
      (result) => {
        this.records = this.records.map((row) => (row.application.id === record.application.id ? result : row));
        this.restoreRequests.delete(record.application.id);
        this.restoreConfirmations = new Set();
        this.notice = abort
          ? 'Restore cancelled. The previous application remains stopped.'
          : 'Restored to fresh data at the same address. Start the application when you are ready.';
      },
      () => {
        this.records = this.records.map((row) => (row === record ? {...row, observation: null} : row));
        this.notice =
          'Restore outcome unconfirmed. Refresh applications and reconcile the same operation before acting again.';
      }
    );
  }
  private status(record: ApplicationRecord) {
    if (record.restore) return 'Restore pending';
    if (!record.observation) return 'State unknown';
    if (!record.observation.container) return 'Not installed';
    if (!record.observation.running) return 'Stopped';
    return record.observation.ready ? 'Ready' : 'Running; readiness unconfirmed';
  }
  private statusText() {
    if (!this.active) return 'Reconnect to Soda Forge to see your applications.';
    if (this.busy && !this.loaded) return 'Loading applications…';
    return this.notice;
  }
  private statusRegion() {
    const text = this.statusText();
    // The live-region wrapper persists so the first status announcement is
    // not missed; only the message child renders when populated.
    // The child markup stays tight so no whitespace text node remains.
    // oxfmt-ignore
    return html`<div class="soda-applications-status" role="status">${text ? html`<p>${text}</p>` : ''}</div>`;
  }
  private environmentLabel(environment: ApplicationEnvironment) {
    return environment === 'preview'
      ? 'Preview · separate data'
      : environment === 'company'
        ? 'Company application'
        : 'Application environment unconfirmed';
  }
  private row(record: ApplicationRecord) {
    return html`<li>
      <header class="soda-application-heading">
        <h2>${record.application.name}</h2>
        <span class="soda-application-status">Last observed: ${this.status(record)}</span>
      </header>
      <p>
        ${this.environmentLabel(record.environment)} · version ${record.application.revision} ·
        ${record.application.origin}
      </p>
      <p>
        Accounts and saved data live with this application at its address, separate from Soda Forge.
        ${record.operator ? 'You are an operator of this application.' : 'An operator manages its setup and updates.'}
      </p>
      ${this.appSurface(record.application, this.appState(record), record.can_embed === true, record.environment, true)}
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
      <div class="soda-application-actions">
        ${this.openApplication(record)}
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
      <details class="soda-application-build">
        <summary>Build and delivery details</summary>
        <button
          type="button"
          class="ui button"
          ?disabled=${this.blocked() || !!record.restore || !record.observation?.container}
          @click=${() => void this.act(record, 'verify_build')}
        >
          Verify build ${record.application.name}
        </button>
        ${record.delivery ? html`<p>Build matches the verified project delivery. Source <code>${record.delivery.source_commit.slice(0, 12)}</code> · run <code>${record.delivery.run}</code>.</p>` : html`<p>Delivery provenance is unconfirmed. Verify build checks the admitted image against recorded project delivery evidence.</p>`}
      </details>
      <details class="soda-application-runtime" ?open=${this.appState(record) === 'stopped' || !!record.restore}>
        <summary>Start or stop application</summary>
        <p class="soda-application-origin">${record.application.origin}</p>
        <p>Runtime limits: 1 GiB memory, up to two CPUs' worth of CPU time, and 128 processes or threads.</p>
        <div class="soda-application-runtime-actions">
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
        </div>
      </details>
      ${this.replaceView(record)} ${this.backupView(record)}
    </li>`;
  }
  private replaceView(record: ApplicationRecord) {
    if (!record.operator || !record.observation?.container) return '';
    return html`<details class="soda-application-replace">
      <summary>Replace application version (operator)</summary>
      <p>
        Use admitted references prepared for this application's address. Entering references does not establish Factory
        verification. Company records and accounts stay with ${record.application.name}; replacement installs a stopped
        version.
      </p>
      <form @submit=${(event: SubmitEvent) => void this.replaceApplication(record, event)}>
        <label
          >Admitted artifact references<textarea
            name="replacement"
            required
            rows="3"
            placeholder='{"artifact_sha256":"…","bundle_sha256":"…","tools_sha256":"…","runtime_image":"sha256:…","image":"sha256:…"}'
          ></textarea>
        </label>
        <label
          ><input type="checkbox" name="confirm_stop" ?disabled=${this.blocked()} />Stop ${record.application.name} for
          everyone and replace its version</label
        >
        <button type="submit" class="ui button" ?disabled=${this.blocked() || !!record.restore}>
          Replace ${record.application.name}
        </button>
      </form>
    </details>`;
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
    return this.openLink(record.application.name, record.application.origin);
  }
  private openLink(name: string, origin: string) {
    return html`<a class="ui primary button" href=${origin} target="_blank" rel="noopener noreferrer"
      >Open application ↗</a
    >`;
  }
  private appSurface(
    application: ApplicationLaunchRecord['application'],
    state: ApplicationState,
    embeddable: boolean,
    environment: ApplicationEnvironment,
    canInspect: boolean
  ) {
    return html`<soda-application-surface
      .name=${application.name}
      .origin=${application.origin}
      .revision=${application.revision}
      .state=${state}
      .embeddable=${embeddable}
      .canInspect=${canInspect}
      .active=${this.active}
      .sessionKey=${this.sessionKey}
      .environment=${environment}
    ></soda-application-surface>`;
  }
  private appState(record: ApplicationRecord): 'ready' | 'running' | 'stopped' | 'not-installed' | 'unknown' {
    if (record.restore) return 'unknown';
    if (!record.observation) return 'unknown';
    if (!record.observation.container) return 'not-installed';
    if (!record.observation.running) return 'stopped';
    return record.observation.ready ? 'ready' : 'running';
  }
  private launchRow(record: ApplicationLaunchRecord) {
    const state = record.state;
    const status = {
      ready: 'Ready',
      running: 'Running; readiness unconfirmed',
      stopped: 'Stopped',
      'not-installed': 'Release pending',
      unknown: 'State unknown',
    }[state];
    return html`<li>
      <header class="soda-application-heading">
        <h2>${record.application.name}</h2>
        <span class="soda-application-status">Last observed: ${status}</span>
      </header>
      <p class="soda-application-origin">${record.application.origin}</p>
      <p>
        Accounts and saved data live with this application at its address, separate from Soda Forge. An operator manages
        its setup and updates.
      </p>
      ${record.project ? html`<p>Project: <a href=${record.project.path + '/workspace'}>${record.project.name}</a></p>` : ''}
      ${this.appSurface(record.application, state, record.can_embed, record.environment, false)}
    </li>`;
  }
  private replaceApplication(record: ApplicationRecord, event: SubmitEvent) {
    // Operator-only authorized update (FJ35): consumes the exact verified
    // artifact references, keeps app/data/owner identity, and observes the new
    // generation. The previous revision is retained for recovery.
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement)) return;
    const data = new FormData(event.currentTarget);
    let selection: Record<string, unknown>;
    try {
      const replacement = object(JSON.parse(String(data.get('replacement') ?? '')));
      check(
        Object.keys(replacement).length === 5 &&
          Object.keys(replacement).every((key) =>
            ['artifact_sha256', 'bundle_sha256', 'tools_sha256', 'runtime_image', 'image'].includes(key)
          )
      );
      selection = {replacement: {...record.application, ...replacement, revision: record.application.revision + 1}};
    } catch {
      this.notice =
        'Enter only admitted artifact references (artifact/tool/bundle digests and runtime/image). Identity stays with the current application.';
      return;
    }
    if (
      !(
        event.currentTarget.querySelector('input[name="confirm_stop"]') instanceof HTMLInputElement &&
        (event.currentTarget.querySelector('input[name="confirm_stop"]') as HTMLInputElement).checked
      )
    ) {
      this.notice = 'Confirm stopping the running application for everyone before replacing its version.';
      return;
    }
    const body = {
      action: 'replace',
      revision: record.application.revision,
      expected_container: record.observation?.container ?? '',
      confirm_stop: true,
      ...selection,
    };
    this.confirmations = new Set();
    this.notice = '';
    return this.run(
      async (request, signal) =>
        applicationResponse(await request(`/api/applications/${record.application.id}/actions`, body, signal)),
      (result) => {
        check(
          result.application.id === record.application.id &&
            result.application.origin === record.application.origin &&
            result.application.revision === record.application.revision + 1 &&
            result.application.data === record.application.data &&
            result.application.owner_team === record.application.owner_team
        );
        this.records = this.records.map((item) => (item === record ? result : item));
        this.notice =
          'The new version is installed and stopped. Start the application, then reopen the company record to check its retained history.';
      },
      (status) => {
        this.records = this.records.map((item) => (item === record ? {...item, observation: null} : item));
        this.notice =
          status === 401
            ? 'Your Soda session changed. Reconnect, then inspect the same application before retrying; the update outcome is unconfirmed.'
            : 'Update outcome unconfirmed. Inspect the application before acting again.';
      }
    );
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
  private beginPreparedSetup(event: Event) {
    if (!(event instanceof CustomEvent) || !event.detail || typeof event.detail.setupId !== 'string') return;
    void this.querySelector<SodaApplicationSetup>('soda-application-setup')?.beginPreparedSetup(event.detail.setupId);
  }
  private onSetupComplete(event?: Event) {
    // Both the delivery child (prepared-build setup) and the setup child
    // (Continue owner setup) report completion with the same bubbling event.
    // The shared parent is their only common ancestor, so it is the only
    // element that catches either. Guard against double-handling so one
    // completion refreshes the actual inventory exactly once. The event
    // carries the actual completed application identity; older emitters
    // without one stay compatible and resolve as unconfirmed, never a guess.
    // The latest identity wins.
    let applicationId: string | undefined;
    if (event instanceof CustomEvent && event.detail !== null && typeof event.detail === 'object') {
      const id = (event.detail as {applicationId?: unknown}).applicationId;
      if (typeof id === 'string' && /^a[0-9a-f]{24}$/.test(id)) applicationId = id;
    }
    this.setupCompletePending.push(applicationId);
    if (this.setupCompleteBusy) {
      this.setupCompleteQueued = true;
      return;
    }
    this.setupCompleteBusy = true;
    this.setupCompleteQueued = false;
    void this.settleSetupComplete();
  }
  private async settleSetupComplete() {
    // A completion stays unresolved until its post-completion inventory read
    // lands: the notice is resolved only against actual observation, never
    // against stale inventory. Completions arriving while busy or mid-read
    // owe exactly one coalesced refresh; the in-flight tail re-enters here
    // once idle. A completion arriving during the coalesced re-read loops for
    // one more pass instead of returning with the re-entry dropped, so the
    // latest event identity always resolves.
    if (!this.setupCompleteBusy || this.setupCompleteSettling) return;
    this.setupCompleteSettling = true;
    try {
      const session = this.sessionKey;
      while (this.setupCompleteBusy) {
        if (!this.currentSession(session)) return;
        if (this.busy) {
          // An inventory/status read is in flight, so the required
          // post-completion read cannot run yet. Owe exactly one coalesced
          // refresh; the in-flight tail re-enters here once idle.
          this.setupCompleteQueued = true;
          return;
        }
        // Idle: the refresh below IS the owed post-completion read, so consume
        // the flag first; only completions arriving during this read re-queue.
        this.setupCompleteQueued = false;
        await this.refresh();
        if (!this.currentSession(session)) return;
        if (this.setupCompleteQueued) {
          // Exactly one coalesced re-read for completions that arrived while
          // busy or mid-read. A newer completion during this read re-queues
          // once more and the loop takes another pass instead of dropping it.
          this.setupCompleteQueued = false;
          await this.refresh();
          if (!this.currentSession(session)) return;
        }
        if (this.setupCompleteQueued) continue;
        this.setupCompleteBusy = false;
        this.setupCompleteQueued = false;
        this.resolveSetupCompleteNotice();
        return;
      }
    } finally {
      this.setupCompleteSettling = false;
    }
  }
  private currentSession(session: string) {
    return this.setupCompleteBusy && this.isConnected && this.active && this.sessionKey === session;
  }
  private resolveSetupCompleteNotice() {
    // The notice names only the actually completed application: the latest
    // completion event identity reconciled against the post-completion
    // inventory. Without an identity (older emitters) or without a matching
    // record, the outcome stays unconfirmed — never a guess at another app.
    // Never clobber a failure notice and never claim state the post-completion
    // read did not observe.
    const identity = this.setupCompletePending.length
      ? this.setupCompletePending[this.setupCompletePending.length - 1]
      : undefined;
    this.setupCompletePending = [];
    if (this.notice || !this.active || !this.loaded) return;
    if (!identity) {
      this.notice =
        'Setup complete. The completed application could not be confirmed from the current inventory. Refresh applications and inspect it before acting again.';
      return;
    }
    const record = this.records.find((row) => row.application.id === identity);
    if (!record) {
      this.notice =
        'Setup complete. The completed application is not in the current inventory. Refresh applications to reconcile the completed setup.';
      return;
    }
    if (record.restore) {
      this.notice = 'Setup complete. Restore is pending. Reconcile this operation before starting the application.';
      return;
    }
    const observation = record.observation;
    if (!observation) {
      this.notice =
        'Setup complete. The application record changed but runtime is unconfirmed with no observation. Refresh applications and inspect it before acting again.';
      return;
    }
    if (!observation.container) {
      this.notice =
        'Setup complete. The application reference matched but the application is not installed. Refresh applications and inspect it before acting again.';
      return;
    }
    if (observation.running) {
      this.notice = observation.ready
        ? 'Setup complete. The application is running and ready. Sign in at its address to use it.'
        : 'Setup complete. The application is running; readiness is unconfirmed. Refresh applications to check its availability.';
      return;
    }
    this.notice =
      'Setup complete. The application is installed and stopped. Start the application, then sign in at its address.';
  }
  private openCurrentApplication() {
    this.requestedDelivery = '';
    // The company surface is its own scope, not the unfiltered inventory:
    // previews run separate data and stay hidden here. The delivery
    // selection clearing above is unchanged, so the retained selection and
    // the surrounding navigation keep their existing behavior.
    this.companyScope = true;
    this.dispatchEvent(new CustomEvent('application-delivery-cleared', {bubbles: true, composed: true}));
    this.dispatchEvent(new CustomEvent('application-company-opened', {bubbles: true, composed: true}));
  }
  private openProjectFeed() {
    if (!this.active || !/^[1-9][0-9]*$/.test(this.repository)) return;
    this.dispatchEvent(
      new CustomEvent('soda-project-feed-open', {
        bubbles: true,
        composed: true,
        detail: {repository: this.repository},
      })
    );
  }
  // The empty App view links to the project's Work, where the request and its
  // build progress live. The workspace shell owns the navigation.
  private openProjectWork() {
    if (!this.active || !/^[1-9][0-9]*$/.test(this.repository)) return;
    this.dispatchEvent(
      new CustomEvent('soda-project-work-open', {
        bubbles: true,
        composed: true,
        detail: {repository: this.repository},
      })
    );
  }
  protected render() {
    const projectRecords = this.repository
      ? this.records.filter((record) => record.project?.repository === this.repository)
      : this.records;
    // The opened company surface admits only confirmed company records. An
    // unconfirmed environment belongs to neither data lane, and a preview is
    // separate data — neither is the current company application.
    const scopedRecords = this.companyScope
      ? projectRecords.filter((record) => record.environment === 'company')
      : projectRecords;
    const visibleRecords = this.requestedDelivery
      ? scopedRecords.filter((record) => record.delivery?.delivery_id === this.requestedDelivery)
      : scopedRecords;
    const projectLaunches = this.repository
      ? this.launchable.filter((record) => record.project?.repository === this.repository)
      : this.launchable;
    const scopedLaunches = this.companyScope
      ? projectLaunches.filter((record) => record.environment === 'company')
      : projectLaunches;
    const visibleLaunches = this.requestedDelivery
      ? scopedLaunches.filter((record) => record.delivery_id === this.requestedDelivery)
      : scopedLaunches;
    const noApplications = !visibleRecords.length && !visibleLaunches.length;
    const confirmedEmpty = this.loaded && this.complete && noApplications && !this.busy && !this.notice;
    const exactUnavailable = !!this.requestedDelivery && confirmedEmpty;
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
      ${this.statusRegion()}
      ${
        this.requestedDelivery
          ? html`<p>This view belongs to the selected verified result.</p>
              ${exactUnavailable && !this.busy ? html`<p role="status">An application matching this exact result is not available to you. The current company application may be a different version.</p>` : ''}<button
                type="button"
                class="ui button"
                @click=${() => this.openCurrentApplication()}
              >
                Open current company application
              </button>`
          : ''
      }
      ${this.companyScope && !this.requestedDelivery ? html`<p>Showing the current company application; previews stay separate.</p>` : ''}
      ${!this.requestedDelivery && confirmedEmpty ? html`<p>${this.companyScope ? 'No current company application is available to you. Previews stay separate.' : this.repository ? 'No app is available here yet. Apps appear once a build is verified and released.' : 'No apps are available for your projects yet. An app appears here once a build is verified and released.'}</p>` : ''}
      ${!this.complete ? html`<p>Some application details could not be loaded. Refresh to try again.</p>` : ''}
      ${!this.canSetup && !noApplications && this.active && this.loaded && !this.busy ? html`<p>Application setup and updates are handled by an operator.</p>` : ''}
      ${this.repository && !this.requestedDelivery && confirmedEmpty ? html`<button type="button" class="ui primary button" ?disabled=${!this.active} @click=${() => this.openProjectWork()}>Open Work</button>` : ''}
      ${this.repository && !this.busy && (this.loaded || this.notice) ? html`<button type="button" class="ui button" ?disabled=${!this.active} @click=${() => this.openProjectFeed()}>View project Feed</button>` : ''}
      <ul>
        ${repeat(
          visibleRecords,
          (record) => record.application.id,
          (record) => this.row(record)
        )}
        ${repeat(
          visibleLaunches,
          (record) => record.application.id,
          (record) => this.launchRow(record)
        )}
      </ul>
      ${
        this.canSetup
          ? html`<details class="soda-application-administration" ?open=${!!this.requestedDelivery} @application-setup-complete=${(event: Event) => this.onSetupComplete(event)}>
              <summary>Prepare or update an application</summary>
              <soda-application-delivery
                .request=${this.request}
                .active=${this.active}
                .sessionKey=${this.sessionKey}
                .repository=${this.repository}
                .requestedDelivery=${this.requestedDelivery}
                .applications=${this.records}
                @application-build-setup-ready=${(event: Event) => this.beginPreparedSetup(event)}
                @application-build-applied=${() => void this.refresh()}
              ></soda-application-delivery
              ><soda-application-setup
                .request=${this.request}
                .active=${this.active}
                .sessionKey=${this.sessionKey}
              ></soda-application-setup>
            </details>`
          : ''
      }
    </details>`;
  }
}
if (!customElements.get('soda-applications')) customElements.define('soda-applications', SodaApplications);

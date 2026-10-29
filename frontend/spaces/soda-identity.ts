import {LitElement, html, nothing} from 'lit';
import {check, id, object, projectId, readSodaJSON} from './sodaspaces-api.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {availableConnectionView, connectionView, grantView, items, leaseView} from './soda-identity-response.js';
import type {Connection, Grant, Lease} from './soda-identity-response.js';
import {openProviders, providersChangedEvent} from './soda-providers-events.js';

export class SodaIdentity extends LitElement {
  static properties = {
    sharingOnly: {type: Boolean},
    busy: {state: true},
    message: {state: true},
    connections: {state: true},
    selected: {state: true},
    members: {state: true},
    grants: {state: true},
    leases: {state: true},
    available: {state: true},
  };
  private actor = '';
  private transport?: PreparedExtensionMount;
  private project = '';
  /** Settings → SodaOS → Members & access: sharing in this environment only; connecting an AI account is in the AI accounts dialog. */
  declare sharingOnly: boolean;
  declare private busy: boolean;
  declare private message: string;
  declare private connections: Connection[];
  declare private selected: string;
  declare private members: {user_id: string; login: string}[];
  declare private grants: Grant[];
  declare private leases: Lease[];
  declare private available: Connection[];
  private retired = false;
  private contextSerial = 0;
  constructor() {
    super();
    this.busy = false;
    this.message = this.selected = '';
    this.connections = this.available = [];
    this.grants = [];
    this.leases = [];
    this.members = [];
    this.sharingOnly = false;
  }
  set context(context: {actor: string; transport: PreparedExtensionMount; project: string}) {
    if (this.actor === context.actor && this.transport === context.transport && this.project === context.project)
      return;
    this.configure(context.actor, context.transport, context.project);
    void this.refresh();
  }

  protected createRenderRoot() {
    return this;
  }
  configure(actor: string, transport: PreparedExtensionMount, project = '') {
    if (!id(actor) || (project && !projectId(project))) throw Error('Invalid identity context');
    this.actor = actor;
    this.transport = transport;
    this.project = project;
    this.contextSerial++;
    this.retired = false;
    this.busy = false;
    this.connections = [];
    this.available = [];
    this.grants = [];
    this.leases = [];
    this.members = [];
    this.requestUpdate();
  }
  private retire = () => {
    this.contextSerial++;
    this.retired = true;
    this.connections = [];
    this.grants = [];
    this.leases = [];
    this.available = [];
    this.members = [];
    this.message = 'Reload this Forgejo page before managing connections.';
  };
  connectedCallback() {
    super.connectedCallback();
    // The AI accounts dialog is global; refresh when it changes a connection.
    document.addEventListener(providersChangedEvent, this.refresh);
  }
  disconnectedCallback() {
    document.removeEventListener(providersChangedEvent, this.refresh);
    this.retire();
    super.disconnectedCallback();
  }
  private blocked() {
    return this.busy || this.retired;
  }
  private requestInit(body?: unknown): RequestInit {
    if (!this.transport || this.retired) throw Error('Account connections unavailable');
    const headers: Record<string, string> = {};
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }
    return {
      method: body === undefined ? 'GET' : 'POST',
      headers,
      ...(body === undefined ? {} : {body: JSON.stringify(body)}),
      signal: AbortSignal.timeout(15000),
    };
  }
  private async request(path: string, body?: unknown): Promise<unknown> {
    const serial = this.contextSerial;
    if (!this.transport) throw Error('Account connections unavailable');
    const response = await this.transport.request(path.slice(1), this.requestInit(body));
    const result = await readSodaJSON(response);
    if (serial !== this.contextSerial || this.retired) throw Error('Soda context changed');
    if (!response.ok)
      throw Error('Identity request was not confirmed (' + response.status + '). Refresh before retrying.');
    return result;
  }
  private async act(action: () => Promise<void>) {
    if (this.blocked()) return;
    this.busy = true;
    const serial = this.contextSerial;
    try {
      await action();
      if (serial === this.contextSerial) this.message = 'Connection status refreshed.';
    } catch (error) {
      if (serial === this.contextSerial)
        this.message = error instanceof Error ? error.message : 'Request was not confirmed. Refresh status.';
    } finally {
      if (serial === this.contextSerial) this.busy = false;
    }
  }
  private async load() {
    this.connections = items(await this.request('/identity/connections'), (value) => connectionView(value, this.actor));
    if (!this.connections.some((connection) => connection.id === this.selected))
      this.selected = this.connections[0]?.id || '';
    await this.loadSelected();
    if (this.project) {
      this.available = items(
        await this.request('/environments/' + this.project + '/identity/connections'),
        availableConnectionView
      );
      const result = object(await this.request('/environments/' + this.project + '/members'));
      check(Array.isArray(result.items));
      this.members = result.items.map((raw) => {
        const row = object(raw);
        check(id(row.user_id) && typeof row.login === 'string');
        return {user_id: row.user_id, login: row.login};
      });
    }
  }
  private async loadSelected() {
    this.grants = [];
    this.leases = [];
    if (!this.selected) return;
    const selected = this.selected;
    const [grants, leases] = await Promise.all([
      this.request('/identity/connections/' + selected + '/grants'),
      this.request('/identity/connections/' + selected + '/leases'),
    ]);
    if (selected !== this.selected) return;
    this.grants = items(grants, grantView);
    this.leases = items(leases, leaseView);
  }
  refresh = () => this.act(() => this.load());
  private select = (event: Event) => {
    this.selected = (event.target as HTMLSelectElement).value;
    void this.act(() => this.loadSelected());
  };
  private delegate = (event: SubmitEvent) => {
    event.preventDefault();
    const data = new FormData(event.target as HTMLFormElement),
      user = String(data.get('user_id') || '');
    if (!id(user)) {
      this.message = 'Choose a project member.';
      return;
    }
    void this.act(async () => {
      await this.request('/environments/' + this.project + '/identity/grants', {
        connection_id: this.selected,
        user_id: user,
        confirm_subscription: data.has('subscription'),
        confirm_credential_exposure: data.has('exposure'),
      });
      await this.loadSelected();
    });
  };
  private revokeGrant(grant: Grant) {
    return this.act(async () => {
      await this.request('/identity/grants/' + grant.id + '/revoke', {});
      await this.loadSelected();
    });
  }
  private endLease(lease: Lease) {
    return this.act(async () => {
      await this.request('/identity/leases/' + lease.id + '/end', {});
      await this.loadSelected();
    });
  }
  private selectedView() {
    const connection = this.connections.find((item) => item.id === this.selected);
    if (!connection) return '';
    return html`<p>${connection.email} · ${connection.plan} · ${connection.state}</p>
      <ul>
        ${this.grants.map(
          (grant) => html`<li>
            Project ${grant.project_id}, user ${grant.user_id}: ${grant.revoked ? 'revoked' : 'authorized'}
            ${grant.revoked ? '' : html`<button type="button" @click=${() => this.revokeGrant(grant)}>Revoke delegate</button>`}
          </li>`
        )}
      </ul>
      <ul>
        ${this.leases.map(
          (lease) => html`<li>
            ${lease.kind}: ${lease.execution_id}, user ${lease.actor_id}, project ${lease.project_id}
            <button type="button" @click=${() => this.endLease(lease)}>End active use</button>
          </li>`
        )}
      </ul>`;
  }
  private projectView() {
    if (!this.project) return '';
    return html`<form @submit=${this.delegate}>
        <h3>Delegate this subscription in this project</h3>
        <label
          >Project member
          <select name="user_id" required>
            <option value="">Choose a member</option>
            ${this.members.map((member) => html`<option value=${member.user_id}>${member.login}</option>`)}
          </select></label
        >
        <label
          ><input type="checkbox" name="subscription" required />I confirm my provider terms allow this named user to
          use my subscription.</label
        >
        <label
          ><input type="checkbox" name="exposure" required />I understand the selected tool receives credentials in the
          trusted project process; code running with that account may read them.</label
        >
        <button ?disabled=${!this.selected}>Authorize named member</button>
      </form>
      <p>
        Connected accounts sign in the <code>codex</code> and <code>muse</code> commands in this project's terminals.
      </p>`;
  }
  protected render() {
    const title = this.sharingOnly ? 'AI accounts shared in this environment' : 'AI accounts';
    return html`<section aria-label=${title}>
      <h2>${title}</h2>
      <p role="status">${this.message}</p>
      <fieldset ?disabled=${this.blocked()}>
        <legend>Your AI accounts</legend>
        <button type="button" @click=${this.refresh}>Refresh status</button>
        ${this.sharingOnly ? nothing : html`<button type="button" class="ui basic button" @click=${openProviders}>AI accounts</button>`}
        <label
          >Your AI accounts
          <select .value=${this.selected} @change=${this.select}>
            ${this.connections.map((connection) => html`<option value=${connection.id}>${connection.label}</option>`)}
          </select></label
        >
        ${this.selectedView()}${this.projectView()}
      </fieldset>
    </section>`;
  }
}
customElements.define('soda-identity', SodaIdentity);

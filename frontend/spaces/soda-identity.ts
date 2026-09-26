import {LitElement, html} from 'lit';
import {connectionSuppressed} from './soda-connection.js';
import {check, id, object, projectId, readSodaJSON} from './sodaspaces-api.js';
import type {Session} from './sodaspaces-api.js';
import {
  availableConnectionView,
  connectionView,
  enrollmentView,
  grantView,
  items,
  leaseView,
} from './soda-identity-response.js';
import type {Connection, Enrollment, Grant, Lease} from './soda-identity-response.js';

export class SodaIdentity extends LitElement {
  static properties = {
    busy: {state: true},
    message: {state: true},
    connections: {state: true},
    selected: {state: true},
    enrollment: {state: true},
    grants: {state: true},
    leases: {state: true},
    available: {state: true},
  };
  private actor = '';
  private session?: Session;
  private project = '';
  declare private busy: boolean;
  declare private message: string;
  declare private connections: Connection[];
  declare private selected: string;
  declare private enrollment: Enrollment | undefined;
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
    this.enrollment = undefined;
  }
  set context(context: {actor: string; session: Session; project: string}) {
    if (
      this.actor === context.actor &&
      this.session?.csrf_token === context.session.csrf_token &&
      this.project === context.project
    )
      return;
    this.configure(context.actor, context.session, context.project);
    void this.refresh();
  }

  protected createRenderRoot() {
    return this;
  }
  configure(actor: string, session: Session, project = '') {
    if (!id(actor) || session.user.id !== actor || (project && !projectId(project)))
      throw Error('Invalid identity context');
    this.actor = actor;
    this.session = session;
    this.project = project;
    this.contextSerial++;
    this.retired = false;
    this.busy = false;
    this.connections = [];
    this.available = [];
    this.grants = [];
    this.leases = [];
    this.enrollment = undefined;
    this.requestUpdate();
  }
  private retire = () => {
    this.contextSerial++;
    this.retired = true;
    this.connections = [];
    this.grants = [];
    this.leases = [];
    this.available = [];
    this.enrollment = undefined;
    this.message = 'Reconnect Soda before managing subscriptions.';
  };
  connectedCallback() {
    super.connectedCallback();
    window.addEventListener('soda-session-retired', this.retire);
  }
  disconnectedCallback() {
    window.removeEventListener('soda-session-retired', this.retire);
    super.disconnectedCallback();
  }
  private blocked() {
    return this.busy || this.retired || connectionSuppressed();
  }
  private requestInit(body?: unknown): RequestInit {
    const session = this.session;
    if (!session || this.retired || connectionSuppressed()) throw Error('Soda connection unavailable');
    const headers: Record<string, string> = {'X-Soda-Expected-User-ID': this.actor};
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      headers['X-CSRF-Token'] = session.csrf_token;
    }
    return {
      method: body === undefined ? 'GET' : 'POST',
      credentials: 'same-origin',
      cache: 'no-store',
      redirect: 'error',
      headers,
      ...(body === undefined ? {} : {body: JSON.stringify(body)}),
      signal: AbortSignal.timeout(15000),
    };
  }
  private async request(path: string, body?: unknown): Promise<unknown> {
    const serial = this.contextSerial;
    const response = await fetch('/-/soda/api' + path, this.requestInit(body));
    const result = await readSodaJSON(response);
    if (serial !== this.contextSerial || this.retired || connectionSuppressed()) throw Error('Soda context changed');
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
      if (serial === this.contextSerial) this.message = 'Subscription status refreshed.';
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
    if (this.project)
      this.available = items(
        await this.request('/environments/' + this.project + '/identity/connections'),
        availableConnectionView
      );
    if (this.enrollment)
      this.enrollment = enrollmentView(await this.request('/identity/enrollments/' + this.enrollment.id));
  }
  private async loadSelected() {
    this.grants = [];
    this.leases = [];
    if (!this.selected) return;
    this.grants = items(await this.request('/identity/connections/' + this.selected + '/grants'), grantView);
    this.leases = items(await this.request('/identity/connections/' + this.selected + '/leases'), leaseView);
  }
  refresh = () => this.act(() => this.load());
  private select = (event: Event) => {
    this.selected = (event.target as HTMLSelectElement).value;
    void this.act(() => this.loadSelected());
  };
  private connect = (event: SubmitEvent) => {
    event.preventDefault();
    const data = new FormData(event.target as HTMLFormElement),
      label = String(data.get('label') || '').trim();
    void this.act(async () => {
      this.enrollment = enrollmentView(
        await this.request('/identity/enrollments', {label, confirm_credential_exposure: data.has('exposure')})
      );
    });
  };
  private disconnect = () =>
    this.act(async () => {
      await this.request('/identity/connections/' + this.selected + '/revoke', {});
      await this.load();
    });
  private cancel = () =>
    this.act(async () => {
      if (!this.enrollment) return;
      await this.request('/identity/enrollments/' + this.enrollment.id + '/cancel', {});
      this.enrollment = undefined;
    });
  private delegate = (event: SubmitEvent) => {
    event.preventDefault();
    const data = new FormData(event.target as HTMLFormElement),
      user = String(data.get('user_id') || '');
    if (!id(user)) {
      this.message = 'Enter a valid Forgejo user ID.';
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
  private launch = (event: SubmitEvent) => {
    event.preventDefault();
    const connection = String(new FormData(event.target as HTMLFormElement).get('connection_id') || '').trim();
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(connection)) {
      this.message = 'Enter an authorized connection ID.';
      return;
    }
    void this.startCodex(connection);
  };
  private async startCodex(connection: string) {
    if (this.blocked() || !this.project) return;
    const serial = this.contextSerial;
    this.busy = true;
    try {
      const result = object(
        await this.request('/environments/' + this.project + '/identity/launch', {
          connection_id: connection,
          cols: 80,
          rows: 24,
        })
      );
      check(typeof result.terminal_id === 'string' && /^[0-9a-f]{32}$/.test(result.terminal_id));
      this.message = 'Codex started. Open project terminal and select Codex.';
    } catch (error) {
      if (serial === this.contextSerial)
        this.message =
          error instanceof Error ? error.message : 'Codex start was not confirmed. Refresh before retrying.';
    } finally {
      if (serial === this.contextSerial) this.busy = false;
    }
  }
  private enrollmentView() {
    const enrollment = this.enrollment;
    if (!enrollment) return '';
    return html`<section>
      <p>Provider sign-in: ${enrollment.state}</p>
      ${
        enrollment.verification_url
          ? html`<a href=${enrollment.verification_url} target="_blank" rel="noopener noreferrer"
                >Open OpenAI sign-in</a
              >
              <p>Code: <strong>${enrollment.user_code}</strong></p>`
          : ''
      }
      <button type="button" @click=${this.cancel}>Cancel sign-in</button>
    </section>`;
  }
  private selectedView() {
    const connection = this.connections.find((item) => item.id === this.selected);
    if (!connection) return '';
    return html`<p>${connection.email} · ${connection.plan} · ${connection.state}</p>
      <p>Connection ID: ${connection.id}</p>
      <button type="button" @click=${this.disconnect}>Disconnect subscription and end its active uses</button>
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
          >Named member's Forgejo user ID <input name="user_id" required inputmode="numeric" pattern="[1-9][0-9]*"
        /></label>
        <label
          ><input type="checkbox" name="subscription" required />I confirm my provider terms allow this named user to
          use my subscription.</label
        >
        <label
          ><input type="checkbox" name="exposure" required />I understand Codex receives credentials in the trusted
          project process; code running with that account may read them.</label
        >
        <button ?disabled=${!this.selected}>Authorize named member</button>
      </form>
      <form @submit=${this.launch}>
        <h3>Start Codex</h3>
        <label
          >Authorized subscription
          <select name="connection_id" required>
            ${this.available.map((connection) => html`<option value=${connection.id}>${connection.label} · owner ${connection.owner_id}</option>`)}
          </select></label
        >
        <button ?disabled=${this.available.length === 0}>Start Codex in this project's terminal</button>
      </form>`;
  }
  protected render() {
    return html`<section aria-label="Codex subscription">
      <h2>Codex subscription</h2>
      <p role="status">${this.message}</p>
      <fieldset ?disabled=${this.blocked()}>
        <legend>Personal subscription connections</legend>
        <button type="button" @click=${this.refresh}>Refresh status</button>
        <form @submit=${this.connect}>
          <label>Connection label <input name="label" required maxlength="80" /></label
          ><label
            ><input type="checkbox" name="exposure" required />I trust this appliance and project administrators with my
            subscription credentials. Codex and code running in my project account may read them.</label
          ><button>Connect OpenAI subscription</button>
        </form>
        ${this.enrollmentView()}<label
          >Your connections
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
export function mountIdentity(root: HTMLElement, actor: string, session: Session, project = '') {
  const widget = new SodaIdentity();
  widget.configure(actor, session, project);
  root.append(widget);
  void widget.refresh();
  return widget;
}

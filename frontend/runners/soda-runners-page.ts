import {signOut, connectionSuppressed} from '../spaces/soda-connection.js';
import {LitElement, html} from 'lit';
import {id, object, readSodaJSON, sessionResponse} from '../spaces/sodaspaces-api.js';
import {decodeRunnerResponse} from './soda-runner-response.js';
import type {LifecycleAction, ListResponse, Runner} from './soda-runner-types.js';

const effects: Record<LifecycleAction, string> = {
  start: 'Start this existing listener now and enable host-boot start. This does not register or repair it.',
  stop: 'Stop and disable host-boot start. An active job may be interrupted; its provider outcome is not locally known.',
  restart: 'Interrupt and restart the listener. Boot start will be enabled, even if it was previously stopped.',
  remove: 'Permanently remove this local Linux account, provider credentials, dependencies, work files and uncommitted job changes. Provider registration and history remain. There is no rollback.',
};
const unconfirmed = 'Operation unconfirmed: local account, files or listener may have changed. Inspect local and Forgejo state before retrying. Refresh does not confirm the operation; no automatic retry or rollback occurred.';

class RunnerRequestError extends Error {
  constructor(readonly status: number) {super('Runner request rejected');}
}

class SodaRunners extends LitElement {
  static properties = {
    busy: {state: true}, rows: {state: true}, message: {state: true}, notice: {state: true},
    stale: {state: true}, pending: {state: true}, blocked: {state: true},
  };
  declare private busy: boolean;
  declare private rows: ListResponse | null;
  declare private message: string;
  declare private notice: string;
  declare private stale: boolean;
  declare private blocked: boolean;
  declare private pending: {id: string; action: LifecycleAction} | null;
  private actor = '';
  // One controller is both the current document/element lifetime and its request
  // generation. An abort is not proof that a dispatched native effect stopped.
  private lifetime: AbortController | null = null;
  private operation: {kind: 'runner'; sent: boolean} | null = null;
  private confirmationTrigger: HTMLButtonElement | null = null;
  private dirty = false;
  private readonly beforeDeparture = (event: BeforeUnloadEvent) => {
    if (this.dirty || this.busy) {event.preventDefault(); event.returnValue = '';}
  };
  private readonly pageHidden = () => this.retire();
  private readonly pageShown = () => this.resume();
  private readonly visibilityChanged = () => {
    if (!document.hidden && this.lifetime && !this.busy) void this.refresh();
  };

  constructor() {
    super();
    this.busy = false;
    this.rows = null;
    this.message = 'Checking local runners…';
    this.notice = '';
    this.stale = true;
    this.pending = null;
    this.blocked = false;
  }
  protected createRenderRoot() {return this;}

  connectedCallback() {
    super.connectedCallback();
    if (!this.actor) this.actor = this.dataset.actor || '';
    window.addEventListener('beforeunload', this.beforeDeparture);
    window.addEventListener('pagehide', this.pageHidden);
    window.addEventListener('soda-session-retired', this.pageHidden);
    window.addEventListener('pageshow', this.pageShown);
    document.addEventListener('visibilitychange', this.visibilityChanged);
    this.resume();
  }
  disconnectedCallback() {
    this.retire();
    window.removeEventListener('beforeunload', this.beforeDeparture);
    window.removeEventListener('pagehide', this.pageHidden);
    window.removeEventListener('soda-session-retired', this.pageHidden);
    window.removeEventListener('pageshow', this.pageShown);
    document.removeEventListener('visibilitychange', this.visibilityChanged);
    super.disconnectedCallback();
  }
  private current(lifetime: AbortController | null): lifetime is AbortController {
    return lifetime !== null && this.lifetime === lifetime && !lifetime.signal.aborted && this.isConnected;
  }
  private requireCurrent(lifetime: AbortController) {
    if (!this.current(lifetime)) throw Error('Runner page retired');
  }
  private clearToken() {
    this.querySelectorAll<HTMLInputElement>('input[type=password]').forEach(input => {input.value = '';});
  }
  private retire() {
    this.clearToken(); // Synchronous: do not leave secrets waiting for a Lit render.
    this.lifetime?.abort();
    this.lifetime = null;
    if (this.operation?.kind === 'runner') {
      this.notice = this.operation.sent ? unconfirmed : 'Operation was not sent. Check authorization and refresh before trying again.';

    }
    this.operation = null;
    this.pending = null;
    this.confirmationTrigger = null;
    this.rows = null;
    this.stale = true;
    this.busy = false;
    this.message = 'Refresh authorization before managing runners.';
  }
  private resume() {
    if (this.lifetime || !this.isConnected || connectionSuppressed()) return;
    this.lifetime = new AbortController();
    this.clearToken(); // Also discard a password restored by browser form history.
    void this.refresh();
  }
  private authorizationLost() {
    this.clearToken();
    this.rows = null;
    this.pending = null;
    this.confirmationTrigger = null;
    this.stale = true;
    this.blocked = true;
  }
  private async session(lifetime: AbortController, requireOperator: boolean) {
    this.requireCurrent(lifetime);
    try {
      if (!id(this.actor)) throw Error('Invalid original actor');
      const response = await fetch('/-/soda/api/session', {
        credentials: 'same-origin', cache: 'no-store', redirect: 'error',
        headers: {'X-Soda-Expected-User-ID': this.actor},
        signal: AbortSignal.any([lifetime.signal, AbortSignal.timeout(15000)]),
      });
      this.requireCurrent(lifetime);
      if (!response.ok) {await response.body?.cancel(); throw Error('Session unavailable');}
      const raw = await readSodaJSON(response);
      this.requireCurrent(lifetime);
      const session = sessionResponse(raw, location.origin);
      if (session.user.id !== this.actor || (requireOperator && object(raw).soda_operator !== true)) throw Error('Original operator unavailable');
      return session;
    } catch (error) {
      if (this.current(lifetime)) this.authorizationLost();
      throw error;
    }
  }
  private async request(lifetime: AbortController, path: string, body?: string): Promise<unknown> {
    try {
      const session = await this.session(lifetime, true);
      this.requireCurrent(lifetime);
      if (body !== undefined) this.operation = {kind: 'runner', sent: true};
      const pending = fetch('/-/soda/api/settings/runners' + path, {
        method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
        headers: {'X-Soda-Expected-User-ID': this.actor, ...(body === undefined ? {} : {'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token})},
        ...(body === undefined ? {} : {body}),
        signal: AbortSignal.any([lifetime.signal, AbortSignal.timeout(190000)]),
      });
      body = undefined;
      const response = await pending;
      this.requireCurrent(lifetime);
      if (!response.ok) {
        if ([401, 403].includes(response.status)) this.authorizationLost();
        await response.body?.cancel();
        throw new RunnerRequestError(response.status);
      }
      const result = await readSodaJSON(response);
      this.requireCurrent(lifetime);
      return result;
    } finally {body = undefined;}
  }
  private async load(lifetime: AbortController) {
    const rows = decodeRunnerResponse('list', await this.request(lifetime, ''));
    this.requireCurrent(lifetime);
    if (rows.runners.length > 64 || rows.forgejo_url !== location.origin || rows.runners.some(row => !/^[a-z][a-z0-9-]{0,15}$/.test(row.id))) throw Error('Invalid inventory');
    this.rows = rows;
    this.stale = false;
    this.blocked = false;
    this.message = 'Local observations refreshed. Provider online, busy and available capacity are not queried.';
  }
  private async refresh() {
    const lifetime = this.lifetime;
    if (this.busy || !this.current(lifetime)) return;
    this.busy = true;
    this.stale = true;
    this.pending = null;
    this.confirmationTrigger = null;
    this.message = 'Checking authorization and local runners…';
    try {await this.load(lifetime);}
    catch {
      if (this.current(lifetime)) this.message = 'Runner inventory unavailable. Previous observations are stale, not an empty or healthy inventory.';
    } finally {
      if (this.current(lifetime)) this.busy = false;
    }
  }
  private async mutate(path: string, body: string) {
    const lifetime = this.lifetime;
    if (this.busy || !this.current(lifetime) || this.stale || this.blocked) return;
    this.busy = true;
    this.pending = null;
    this.confirmationTrigger = null;
    this.operation = {kind: 'runner', sent: false};
    this.notice = 'Checking authorization before sending the operation…';
    try {
      const response = this.request(lifetime, path, body);
      body = '';
      decodeRunnerResponse('create', await response);
      this.requireCurrent(lifetime);
      this.notice = 'Native operation confirmed. Provider records and job results remain provider-owned.';
    } catch (error) {
      if (!this.current(lifetime)) return;
      if (error instanceof RunnerRequestError && [400, 404, 422].includes(error.status)) {
        this.notice = 'Operation rejected. Check the runner ID, provider registration fields and labels, then refresh before trying again.';
      } else if (error instanceof RunnerRequestError && [401, 403].includes(error.status)) {
        this.notice = 'Operation not authorized. Reconnect explicitly before trying again.';
      } else {
        this.notice = this.operation?.sent ? unconfirmed : 'Operation was not sent. Check authorization and refresh before trying again.';
      }
    } finally {
      body = '';
      if (this.current(lifetime)) {
        this.operation = null;
        this.stale = true;
        // A fresh list is useful after either outcome, but never confirms an
        // uncertain mutation. Keep the operation notice independent of refresh.
        if (!this.blocked) {
          try {await this.load(lifetime);}
          catch {if (this.current(lifetime)) this.message = 'Local refresh failed; shown observations are stale.';}
        }
        if (this.current(lifetime)) this.busy = false;
      }
    }
  }
  private inputChanged(event: Event) {
    if (event.target instanceof HTMLInputElement) event.target.setCustomValidity('');
    this.dirty = true;
  }
  private register(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || this.busy || this.stale || this.blocked) return;
    const form = event.currentTarget;
    const labels = form.querySelector<HTMLInputElement>('input[name=labels]');
    if (labels) {
      const pattern = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}:host$/;
      labels.setCustomValidity(labels.value.split(',').every(label => pattern.test(label)) ? '' : 'Separate name:host labels with commas and no spaces, for example soda-linux:host.');
    }
    if (!form.reportValidity()) return;
    const data = new FormData(form);
    const field = (name: string) => {const value = data.get(name); return typeof value === 'string' ? value : '';};
    let body = JSON.stringify({id: field('id'), provider: 'forgejo', registration_url: '', registration_id: field('registration_id'), labels: field('labels'), registration_token: field('registration_token')});
    data.delete('registration_token');
    this.clearToken();
    this.dirty = false;
    void this.mutate('', body);
    body = '';
  }
  private async chooseAction(event: Event, runner: string, action: LifecycleAction) {
    const lifetime = this.lifetime;
    if (this.busy || this.stale || this.blocked || !this.current(lifetime)) return;
    this.confirmationTrigger = event.currentTarget instanceof HTMLButtonElement ? event.currentTarget : null;
    this.pending = {id: runner, action};
    await this.updateComplete;
    if (this.current(lifetime) && this.pending?.id === runner && this.pending.action === action) this.querySelector<HTMLInputElement>('input[name=confirm_id]')?.focus();
  }
  private async cancelConfirmation() {
    const lifetime = this.lifetime, trigger = this.confirmationTrigger;
    this.pending = null;
    this.confirmationTrigger = null;
    await this.updateComplete;
    if (this.current(lifetime) && trigger?.isConnected) trigger.focus();
  }
  private confirmationKey(event: KeyboardEvent) {
    if (event.key === 'Escape') {event.preventDefault(); void this.cancelConfirmation();}
  }
  private confirm(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || !this.pending) return;
    const {id: runner, action} = this.pending;
    if (new FormData(event.currentTarget).get('confirm_id') !== runner) {
      this.notice = 'Type the exact runner ID to confirm.';
      this.querySelector<HTMLInputElement>('input[name=confirm_id]')?.focus();
      return;
    }
    void this.mutate(`/${runner}/${action}`, JSON.stringify({confirm_id: runner}));
  }
  private async logout() {
    if (this.busy || !this.current(this.lifetime)) return;
    this.dirty = false;
    await signOut(this.actor);
  }

  private runnerView(row: Runner, disabled: boolean) {
    return html`
      <article class="settings-runner">
        <div class="settings-runner-heading"><h3>${row.id}</h3><span>Forgejo</span></div>
        <p class="settings-help">${row.account} · ${row.architecture} · ${row.version}</p>
        <p>Service: ${row.service.load} / ${row.service.active} / ${row.service.sub}. Boot policy: ${row.service.enabled}. ${row.capacity} configured slot.</p>
        <p><a href=${location.origin} aria-label=${`Open ${row.id} in Forgejo`}>Open in Forgejo</a></p>
        <div class="settings-actions">
          ${(['start', 'stop', 'restart', 'remove'] as const).map(action => html`
            <button type="button" class=${action === 'remove' ? 'settings-danger' : ''} ?disabled=${disabled}
              @click=${(event: Event) => void this.chooseAction(event, row.id, action)}>${action} ${row.id}</button>`)}
        </div>
      </article>`;
  }
  private inventoryView(disabled: boolean) {
    if (!this.rows) return '';
    return html`
      <section aria-label="Local runner inventory">
        <h2>${this.stale ? 'Stale observations' : 'Local capacity'}</h2>
        <dl class="settings-capacity">
          <div><dt>Local runners</dt><dd>${this.rows.runner_count}</dd></div>
          <div><dt>Listening services</dt><dd>${this.rows.active_listeners}</dd></div>
          <div><dt>Configured slots</dt><dd>${this.rows.total_capacity}</dd></div>
        </dl>
        <p class="settings-help">Configured slots are not available job slots. Labels are not recorded in local descriptors; inspect them in the provider.</p>
        ${this.rows.runners.length === 0 ? html`<p>No local runners registered.</p>` : this.rows.runners.map(row => this.runnerView(row, disabled))}
      </section>`;
  }
  private registrationView(disabled: boolean) {
    return html`
      <section aria-labelledby="register-runner-title">
        <h2 id="register-runner-title">Register local runner</h2>
        <p>Registration starts a one-slot host listener and enables it at boot.</p>
        <form @submit=${(event: SubmitEvent) => this.register(event)} @input=${(event: Event) => this.inputChanged(event)} autocomplete="off">
          <fieldset ?disabled=${disabled}><legend>Provider registration</legend>
            <label>Local runner ID<input name="id" pattern="[a-z][a-z0-9\\-]{0,15}" maxlength="16" aria-describedby="runner-id-help" required></label>
            <p id="runner-id-help" class="settings-help">Use 1–16 lowercase letters, digits or hyphens, starting with a letter.</p>
              <p>An authorized Forgejo administrator must first create a system runner and supply its UUID/token. Soda does not create or reset that provider record.</p>
              <label>Forgejo runner UUID<input name="registration_id" required pattern="[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" aria-describedby="runner-uuid-help"></label>
              <p id="runner-uuid-help" class="settings-help">Copy the lowercase UUID from the native Forgejo runner registration.</p>
            <label>Labels<input name="labels" maxlength="4096" aria-describedby="runner-labels-help" required></label>
            <p id="runner-labels-help" class="settings-help">Comma-separated name:host labels, for example soda-linux:host. OCI labels are not supported.</p>
            <label>Registration token<input type="password" name="registration_token" autocomplete="new-password" maxlength="8192" aria-describedby="runner-token-help" required></label>
            <p id="runner-token-help" class="settings-help">The token is cleared after submission, when signing out or leaving this page. Never include it in screenshots or shared logs.</p>
            <button class="settings-primary">Register and start listener</button>
          </fieldset>
        </form>
      </section>`;
  }
  protected render() {
    const disabled = this.busy || this.stale || this.blocked || !this.lifetime;
    return html`
      <div class="settings-actions">
        <button type="button" ?disabled=${this.busy || !this.lifetime} @click=${() => void this.refresh()}>Refresh status</button>
        <button type="button" ?disabled=${this.busy || !this.lifetime} @click=${() => void this.logout()}>Sign out</button>
      </div>
      <p role="status">${this.message}</p>
      ${this.notice ? html`<p role="status" class="settings-notice">${this.notice}</p>` : ''}
      ${this.blocked ? html`<p>Operator authorization unavailable. <a href=${`/-/soda/login?destination=runners&expected_user_id=${this.actor}`}>Reconnect explicitly</a></p>` : ''}
      ${this.inventoryView(disabled)}
      ${this.pending ? html`
        <section class="settings-confirmation" aria-label="Confirm runner operation">
          <h2>${this.pending.action} ${this.pending.id}</h2><p>${effects[this.pending.action]}</p>
          <form @submit=${(event: SubmitEvent) => this.confirm(event)} @keydown=${(event: KeyboardEvent) => this.confirmationKey(event)}>
            <label>Exact runner ID<input name="confirm_id" autocomplete="off" required></label>
            <div class="settings-actions"><button ?disabled=${disabled}>Confirm ${this.pending.action}</button><button type="button" @click=${() => void this.cancelConfirmation()}>Cancel</button></div>
          </form>
        </section>` : ''}
      <section aria-labelledby="runner-provider-title">
        <h2 id="runner-provider-title">Provider access</h2>
        <p><a href=${location.origin + '/admin/actions/runners'}>Forgejo Actions administration</a> manages native runner registrations. It requires separate Forgejo administrator permission and may deny a Soda operator.</p>
        <p>Forgejo owns registration authority, labels, workflows, scheduling and results. Soda manages local accounts, listeners and capacity.</p>
        <p>Host execution only; isolated OCI jobs are not supported. Use only trusted repositories and contributors. Jobs can change this account’s persistent work files.</p>
      </section>
      ${this.registrationView(disabled)}`;
  }
}
customElements.define('soda-runners', SodaRunners);
export function mountRunnersPage(root: HTMLElement, actor: string) {
  if (!id(actor)) throw Error('Invalid runner actor');
  const view = document.createElement('soda-runners');
  view.dataset.actor = actor;
  root.replaceChildren(view);
  return {dispose() {view.remove();}};
}
const root = document.getElementById('soda-runners-page');
if (root && id(root.dataset.actor)) mountRunnersPage(root, root.dataset.actor);

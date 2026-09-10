import {LitElement, html} from 'lit';
import {id, object, readSodaJSON, sessionResponse} from '../spaces/sodaspaces-api.js';
import {decodeRunnerResponse} from './soda-runner-response.js';
import type {LifecycleAction, ListResponse} from './soda-runner-types.js';

const effects: Record<LifecycleAction, string> = {
  start: 'Start this existing listener now and enable host-boot start. This does not register or repair it.',
  stop: 'Stop and disable host-boot start. An active job may be interrupted; its provider outcome is not locally known.',
  restart: 'Interrupt and restart the listener. Boot start will be enabled, even if it was previously stopped.',
  remove: 'Permanently remove this local Linux account, provider credentials, dependencies, work files and uncommitted job changes. Provider registration and history remain. There is no rollback.',
};
class SodaRunners extends LitElement {
  static properties = {busy: {state: true}, rows: {state: true}, message: {state: true}, stale: {state: true}, provider: {state: true}, pending: {state: true}, blocked: {state: true}};
  declare private busy: boolean;
  declare private rows: ListResponse | null;
  declare private message: string;
  declare private stale: boolean;
  declare private provider: string;
  declare private blocked: boolean;
  declare private pending: {id: string; action: LifecycleAction} | null;
  private actor = '';
  private retired = false;
  private dirty = false;
  private readonly departure = (event: BeforeUnloadEvent) => {
    if (this.dirty || this.busy) {event.preventDefault(); event.returnValue = '';}
  };
  constructor() {
    super(); this.busy = false; this.rows = null; this.message = 'Checking local runners…';
    this.stale = true; this.provider = 'forgejo'; this.pending = null; this.blocked = false;
  }
  protected createRenderRoot() {return this;}
  connectedCallback() {
    super.connectedCallback(); this.actor = this.dataset.actor || ''; this.retired = false;
    window.addEventListener('beforeunload', this.departure);
    void this.refresh();
  }
  disconnectedCallback() {
    this.retired = true; window.removeEventListener('beforeunload', this.departure);
    this.querySelectorAll<HTMLInputElement>('input[type=password]').forEach(input => {input.value = '';});
    super.disconnectedCallback();
  }
  private async request(path: string, body?: string): Promise<unknown> {
    if (!id(this.actor) || this.retired) throw Error('Unavailable');
    const sessionRead = await fetch('/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': this.actor}, signal: AbortSignal.timeout(15000)});
    if (!sessionRead.ok) {this.blocked = true; throw Error('Reconnect');}
    const raw = await readSodaJSON(sessionRead), session = sessionResponse(raw, location.origin);
    if (session.user.id !== this.actor || object(raw).soda_operator !== true || this.retired) {this.blocked = true; throw Error('Reconnect');}
    const response = await fetch('/-/soda/api/settings/runners' + path, {
      method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
      headers: {'X-Soda-Expected-User-ID': this.actor, ...(body === undefined ? {} : {'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token})},
      ...(body === undefined ? {} : {body}), signal: AbortSignal.timeout(190000),
    });
    body = undefined;
    if (!response.ok) {if ([401, 403].includes(response.status)) this.blocked = true; await response.body?.cancel(); throw Error('Operation unavailable');}
    return readSodaJSON(response);
  }
  private async load() {
    const rows = decodeRunnerResponse('list', await this.request(''));
    if (this.retired) return;
    if (rows.runners.length > 64 || rows.forgejo_url !== location.origin || rows.runners.some(row => !/^[a-z][a-z0-9-]{0,15}$/.test(row.id))) throw Error('Invalid inventory');
    this.rows = rows; this.stale = false; this.blocked = false;
  }
  private async refresh() {
    if (this.busy || this.retired) return;
    this.busy = true;
    try {await this.load(); this.message = 'Local observations refreshed. Provider online, busy and available capacity are not queried.';}
    catch {this.stale = true; this.message = 'Runner inventory unavailable. Previous observations are stale, not an empty or healthy inventory.';}
    finally {this.busy = false; if (this.blocked) this.rows = null;}
  }
  private async mutate(path: string, body: string) {
    if (this.busy || this.retired || this.stale || this.blocked) return;
    this.busy = true; this.pending = null;
    try {
      const response = this.request(path, body); body = '';
      decodeRunnerResponse('create', await response);
      this.message = 'Native operation confirmed. Refreshing local state; provider records and job results remain provider-owned.';
    } catch {
      this.message = 'Operation unconfirmed: local account, files or listener may have changed and provider registration may remain. Inspect before retrying; no automatic retry or rollback occurred.';
    } finally {
      body = ''; this.stale = true;
      try {await this.load();} catch {this.message += ' Local refresh also failed; shown observations are stale.';}
      this.busy = false; if (this.blocked) this.rows = null;
    }
  }
  private register(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || this.busy || this.stale || this.blocked) return;
    const form = event.currentTarget, data = new FormData(form);
    const field = (name: string) => {const value = data.get(name); return typeof value === 'string' ? value : '';};
    let body = JSON.stringify({id: field('id'), provider: this.provider, registration_url: this.provider === 'github' ? field('registration_url') : '', registration_id: this.provider === 'forgejo' ? field('registration_id') : '', labels: field('labels'), registration_token: field('registration_token')});
    // Credentials never enter reactive state, attributes, storage or error text.
    data.delete('registration_token');
    const secret = form.querySelector<HTMLInputElement>('input[name=registration_token]'); if (secret) secret.value = '';
    this.dirty = false; void this.mutate('', body); body = '';
  }
  private confirm(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || !this.pending) return;
    const {id: runner, action} = this.pending;
    if (new FormData(event.currentTarget).get('confirm_id') !== runner) {this.message = 'Type the exact runner ID to confirm.'; return;}
    void this.mutate(`/${runner}/${action}`, JSON.stringify({confirm_id: runner}));
  }
  private async logout() {
    if (this.busy) return;
    this.busy = true;
    try {
      const response = await fetch('/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': this.actor}, signal: AbortSignal.timeout(15000)});
      if (!response.ok) throw Error('Session unavailable');
      const session = sessionResponse(await readSodaJSON(response), location.origin);
      const ended = await fetch('/-/soda/api/session/logout', {method: 'POST', credentials: 'same-origin', redirect: 'error', headers: {'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token, 'X-Soda-Expected-User-ID': this.actor}, body: '{}', signal: AbortSignal.timeout(15000)});
      if (ended.status !== 204) throw Error('Logout unconfirmed');
      this.dirty = false; this.busy = false; location.reload();
    } catch {this.message = 'Soda sign-out unconfirmed. Native Forgejo sign-out is separate.'; this.busy = false;}
  }
  protected render() {
    const disabled = this.busy || this.stale || this.blocked;
    return html`
      <div class="settings-actions"><button type="button" ?disabled=${this.busy} @click=${() => void this.refresh()}>Refresh status</button><button type="button" ?disabled=${this.busy} @click=${() => void this.logout()}>Sign out of Soda</button></div>
      <p role="status">${this.message}</p>
      ${this.blocked ? html`<p>Operator authorization unavailable. <a href=${`/-/soda/login?destination=runners&expected_user_id=${this.actor}`}>Reconnect explicitly</a></p>` : ''}
      ${this.rows ? html`<section aria-label="Local runner inventory"><h2>${this.stale ? 'Stale observations' : 'Local capacity'}</h2>
        <p>${this.rows.runner_count} runners · ${this.rows.active_listeners} listening · ${this.rows.total_capacity} configured slots. Not available job slots.</p>
        <p>Host execution only; isolated OCI jobs are not supported. Labels are not recorded in legacy local descriptors; inspect them in the provider.</p>
        ${this.rows.runners.length === 0 ? html`<p>No local runners registered.</p>` : this.rows.runners.map(row => html`<article><h3>${row.id} · ${row.provider}</h3>
          <p>${row.account} · ${row.architecture} · ${row.version}</p>
          <p>Service: ${row.service.load} / ${row.service.active} / ${row.service.sub}. Boot policy: ${row.service.enabled}. ${row.capacity} configured slot.</p>
          <div class="settings-actions">${(['start', 'stop', 'restart', 'remove'] as const).map(action => html`<button type="button" ?disabled=${disabled} @click=${() => {this.pending = {id: row.id, action};}}>${action} ${row.id}</button>`)}</div></article>`)}</section>` : ''}
      ${this.pending ? html`<section aria-label="Confirm runner operation"><h2>${this.pending.action} ${this.pending.id}</h2><p>${effects[this.pending.action]}</p>
        <form @submit=${(event: SubmitEvent) => this.confirm(event)}><label>Exact runner ID<input name="confirm_id" autocomplete="off" required></label><button ?disabled=${disabled}>Confirm ${this.pending.action}</button><button type="button" @click=${() => {this.pending = null;}}>Cancel</button></form></section>` : ''}
      <section><h2>Register local runner</h2><p>Registration starts a one-slot host listener and enables it at boot. Use only trusted repositories and contributors. Jobs can change this account's persistent work files.</p>
        <form @submit=${(event: SubmitEvent) => this.register(event)} @input=${() => {this.dirty = true;}} autocomplete="off">
          <fieldset ?disabled=${disabled}><legend>Provider registration</legend>
          <label>Provider<select name="provider" @change=${(event: Event) => {if (event.target instanceof HTMLSelectElement) this.provider = event.target.value;}}><option value="forgejo">Forgejo</option><option value="github">GitHub</option></select></label>
          <label>Local runner ID<input name="id" pattern="[a-z][a-z0-9-]{0,15}" maxlength="16" required></label>
          ${this.provider === 'forgejo' ? html`<p>An authorized Forgejo administrator must first create a system runner and supply its UUID/token. Soda does not create or reset that provider record.</p><label>Forgejo runner UUID<input name="registration_id" required pattern="[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"></label>` : html`<p>Use an authorized short-lived GitHub registration token. Failed registration can leave a provider record; inspect it before retrying.</p><label>GitHub registration URL<input type="url" name="registration_url" placeholder="https://github.com/owner/repository" required></label>`}
          <label>Labels (comma-separated; Forgejo requires name:host)<input name="labels" maxlength="4096" required></label>
          <label>Registration token<input type="password" name="registration_token" autocomplete="new-password" maxlength="8192" required></label>
          <button>Register and start listener</button></fieldset>
        </form>
      </section>`;
  }
}
customElements.define('soda-runners', SodaRunners);
const root = document.getElementById('soda-runners-page');
if (root && id(root.dataset.actor)) {const view = document.createElement('soda-runners'); view.dataset.actor = root.dataset.actor; root.append(view);}

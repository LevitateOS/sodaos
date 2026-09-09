// Soda owns only this mount. Native forms, authentication and terminal lifetime
// are not rendering concerns; commands remain explicit and generation guarded.
import {LitElement, html, nothing} from 'lit';
import {mountTerminal} from './sodaspaces-terminal.js';
import {object, check, id, projectId, fingerprint, sessionResponse, environmentResponse, detailResponse, savedKeysResponse, keyPreviewResponse, SodaRequestError, readSodaJSON} from './sodaspaces-api.js';
import type {Session, Environment, Detail, KeyPreview, SavedKey} from './sodaspaces-api.js';
export interface DrawerContext { expectedUserId?: string | undefined; repositoryId: string }
type DrawerTerminal = Pick<ReturnType<typeof mountTerminal>, 'started' | 'dispose' | 'invalidate'> & Partial<Pick<ReturnType<typeof mountTerminal>, 'restore' | 'retain' | 'returnToWork'>>;
type TerminalFactory = (...args: Parameters<typeof mountTerminal>) => DrawerTerminal;
const rejected = new Set([400, 401, 403, 404, 409, 413, 415, 422]);
const views = ['terminal', 'environment', 'access'] as const;
type View = typeof views[number];

export class SodaSpaces extends LitElement {
  static properties = {
    status: {state: true}, outcome: {state: true}, repository: {state: true}, session: {state: true},
    environment: {state: true}, detail: {state: true}, saved: {state: true}, keyPreview: {state: true},
    lifecycle: {state: true}, connection: {state: true}, busy: {state: true}, stale: {state: true},
    uncertain: {state: true}, connectVisible: {state: true}, canCreate: {state: true},
    selected: {state: true}, draft: {state: true}, stopConfirmed: {state: true}, emptyConfirmed: {state: true},
  };
  declare private status: string;
  declare private outcome: string;
  declare private repository: string;
  declare private session: Session | undefined;
  declare private environment: Environment | undefined;
  declare private detail: Detail | undefined;
  declare private saved: SavedKey[] | undefined;
  declare private keyPreview: KeyPreview | undefined;
  declare private lifecycle: {running: boolean; boot: boolean} | undefined;
  declare private connection: {command: string; fingerprint: string} | undefined;
  declare private busy: boolean;
  declare private stale: boolean;
  declare private uncertain: boolean;
  declare private connectVisible: boolean;
  declare private canCreate: boolean;
  declare private selected: View;
  declare private draft: string;
  declare private stopConfirmed: boolean;
  declare private emptyConfirmed: boolean;
  private binding: DrawerContext | undefined;
  private factory: TerminalFactory = mountTerminal;
  private terminal: DrawerTerminal | undefined;
  private terminalTarget: string | undefined;
  private readController: AbortController | undefined;
  private lifetime = new AbortController();
  private epoch = 0;
  private disposed = false;
  private chosen = false;

  constructor() {
    super();
    this.status = 'Refresh to inspect your shared environment.'; this.outcome = this.repository = this.draft = '';
    this.session = this.environment = this.detail = this.saved = this.keyPreview = this.lifecycle = this.connection = undefined;
    this.busy = this.stale = this.uncertain = this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
    this.connectVisible = true; this.selected = 'environment';
  }
  protected createRenderRoot() { return this; }
  configure(context: DrawerContext, factory: TerminalFactory) {
    if (this.binding || this.disposed) throw Error('Workspace binding is immutable');
    this.binding = {...context}; this.factory = factory;
    window.addEventListener('pagehide', () => this.invalidate(), {signal: this.lifetime.signal});
    window.addEventListener('pageshow', e => { if (e.persisted) this.invalidate(); }, {signal: this.lifetime.signal});
  }
  disconnectedCallback() { super.disconnectedCallback(); this.dispose(); }
  private get blocked() { return this.busy || this.stale || this.uncertain || this.disposed; }
  private get running() { return this.detail?.environment.provisioned === true && !this.detail.native_unavailable && this.detail.observed?.running === true; }
  private active(epoch: number) { return !this.disposed && !this.stale && this.epoch === epoch; }
  private select(view: View) { this.chosen = true; this.selected = view; }
  private async tabKey(event: KeyboardEvent, view: View) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const i = views.indexOf(view), next = event.key === 'Home' ? 0 : event.key === 'End' ? views.length - 1 : (i + (event.key === 'ArrowRight' ? 1 : -1) + views.length) % views.length;
    const target = views[next]; if (!target) return;
    this.select(target); const epoch = this.epoch;
    await this.updateComplete;
    if (this.active(epoch) && !this.closest('[hidden]') && this.selected === target) this.querySelector<HTMLElement>('#sodaspaces-tab-' + target)?.focus();
  }
  // Busy is checked synchronously, not only via Lit's eventually updated disabled attribute.
  private command(event: Event, action: () => void | Promise<void>, allowUncertain = false) {
    const button = event.currentTarget;
    if (!(button instanceof HTMLButtonElement) || button.disabled || button.closest('[hidden]') || this.busy || this.stale || this.disposed || (this.uncertain && !allowUncertain)) return;
    void action();
  }
  protected render() {
    const intent = new URLSearchParams({repository_id: this.binding?.repositoryId || ''});
    if (this.binding?.expectedUserId) intent.set('expected_user_id', this.binding.expectedUserId);
    const stopped = !this.lifecycle?.running && !this.lifecycle?.boot;
    const preview = this.keyPreview;
    return html`<section id="sodaspaces-data" class="soda-spaces-controls" aria-busy=${String(this.busy && !this.stale)}>
      <div class="soda-spaces-tabs" role="tablist" aria-label="Workspace views">${views.map(view => html`
        <button type="button" class="ui basic button" id=${'sodaspaces-tab-' + view} role="tab" aria-controls=${'sodaspaces-view-' + view}
          aria-selected=${String(this.selected === view)} tabindex=${this.selected === view ? 0 : -1}
          @click=${() => this.select(view)} @keydown=${(e: KeyboardEvent) => this.tabKey(e, view)}>${view[0]?.toUpperCase()}${view.slice(1)}</button>`)}</div>
      <div id="sodaspaces-view-terminal" class="soda-spaces-view" role="tabpanel" aria-labelledby="sodaspaces-tab-terminal" ?hidden=${this.selected !== 'terminal'}></div>
      <section id="sodaspaces-view-environment" class="soda-spaces-view" role="tabpanel" aria-labelledby="sodaspaces-tab-environment" ?hidden=${this.selected !== 'environment'}>
        <p>Shared resources, explicit actions. Hiding does not undo work already sent.</p>
        <a id="sodaspaces-sign-in" class="ui primary button" href=${'/-/soda/login?' + intent} ?hidden=${!this.connectVisible || this.stale}
          aria-disabled=${String(this.busy || this.stale)} @click=${(e: Event) => {if (this.busy || this.stale || this.disposed) e.preventDefault();}}>Connect to Soda</a>
        <div class="soda-spaces-actions">
          <button id="sodaspaces-refresh" type="button" class="ui basic button" ?disabled=${this.busy || this.stale} @click=${(e: Event) => this.command(e, () => this.refresh(), true)}>Refresh status</button>
          <button id="sodaspaces-reload" type="button" class="ui basic button" ?hidden=${!this.stale} @click=${(e: Event) => {if (!this.disposed && e.currentTarget instanceof HTMLElement && !e.currentTarget.closest('[hidden]')) window.location.reload();}}>Reload repository page</button>
          <button id="sodaspaces-sign-out" type="button" class="ui basic button" ?hidden=${!this.session} ?disabled=${this.busy || this.stale || !this.session}
            @click=${(e: Event) => this.command(e, () => {this.terminal?.invalidate(); return this.mutate('/api/session/logout', {}, 'Signed out of Soda.');}, true)}>Sign out of Soda</button>
          <button id="sodaspaces-create" type="button" class="ui primary button" ?hidden=${!this.canCreate} ?disabled=${this.blocked}
            @click=${(e: Event) => this.command(e, () => this.mutate('/api/environments', {repository_id: this.binding?.repositoryId}, 'Environment created. Save a public key and explicitly Join; creation does not join you.'))}>Create environment</button>
          <button id="sodaspaces-join" type="button" class="ui primary button" ?hidden=${!this.detail?.environment.provisioned || !!this.detail.login || !this.running || !this.saved?.length} ?disabled=${this.blocked}
            @click=${(e: Event) => this.command(e, () => {if (this.environment) return this.mutate('/api/environments/' + this.environment.id + '/join', {}, 'Native join confirmed. Later saved-key changes require a separate explicit Apply.');})}>Join environment</button>
        </div>
        <fieldset ?hidden=${!this.lifecycle}><legend>Shared environment</legend>
          <p>${this.lifecycle ? `Running: ${this.lifecycle.running ? 'yes' : 'no'}; starts on host boot: ${this.lifecycle.boot ? 'yes' : 'no'}. Start restores boot start; Stop disables it.` : ''}</p>
          <button type="button" class="ui primary button" ?hidden=${!!this.lifecycle?.running && this.lifecycle.boot} ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.changeLifecycle(false))}>Start</button>
          <button type="button" class="ui button danger" ?hidden=${stopped} ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.changeLifecycle(true))}>Stop</button>
          <label ?hidden=${stopped}><input type="checkbox" .checked=${this.stopConfirmed} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) this.stopConfirmed = e.target.checked;}}> I understand Stop interrupts everyone’s SSH, terminals and workloads, and disables next-boot start.</label>
        </fieldset>
      </section>
      <section id="sodaspaces-view-access" class="soda-spaces-view" role="tabpanel" aria-labelledby="sodaspaces-tab-access" ?hidden=${this.selected !== 'access'}>
        <fieldset id="sodaspaces-connection" ?hidden=${!this.connection}><legend>SSH / editor connection</legend>
          <input id="sodaspaces-command" readonly aria-label="SSH command" .value=${this.connection?.command || ''}>
          <p id="sodaspaces-fingerprint">${this.connection ? 'Ed25519 host-key fingerprint: ' + this.connection.fingerprint : ''}</p>
          <button id="sodaspaces-copy" type="button" class="ui basic button" ?disabled=${this.blocked} data-tooltip-appendto="parent" data-clipboard-target=${this.connection ? '#sodaspaces-command' : nothing}>Copy SSH connection</button>
          <p>Use ordinary SSH or your editor’s Remote SSH with this account/IP. An observed IP is not proof of laptop routing.</p>
        </fieldset>
        <fieldset id="sodaspaces-keys" ?hidden=${!this.saved}><legend>My development SSH keys (not Forgejo Git keys)</legend>
          <ul id="sodaspaces-key-list">${this.saved?.map(key => html`<li>${key.fingerprint} <button type="button" class="ui basic button" ?disabled=${this.blocked}
            @click=${(e: Event) => this.command(e, () => this.mutate('/api/me/development-keys/' + key.id, {}, 'Saved key removed. Existing project SSH access is unchanged until explicitly applied.', 'DELETE'))}>Remove saved key</button></li>`)}
            ${this.saved && !this.detail?.login ? html`<li>${this.saved.length ? 'Start must be requested from the project administrator when stopped; then explicitly Join.' : 'Save your public key, then explicitly Join when the environment is running.'}</li>` : ''}</ul>
          <label>Public SSH key <textarea id="sodaspaces-public-key" rows="3" maxlength="16384" spellcheck="false" autocomplete="off" .value=${this.draft} @input=${(e: Event) => {if (e.target instanceof HTMLTextAreaElement) this.draft = e.target.value;}}></textarea></label>
          <button id="sodaspaces-save-key" type="button" class="ui primary button" ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.saveKey())}>Save public key</button>
          <p>Removing a saved key changes future joins only. Use Review → Apply below for this project. Verify a replacement over SSH before revoking the old key.</p>
          <button type="button" class="ui basic button" ?hidden=${!this.detail?.login || !this.running} ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.reviewKeys())}>Review this project’s SSH keys</button>
          <div>${preview ? html`<p>The dedicated Soda-managed key file for this account will match the saved set. Review every removal; other accounts/files/projects and authenticated SSH sessions are unchanged.</p>
            <p>Add: ${preview.saved_fingerprints.filter(k => !preview.installed_fingerprints.includes(k)).join(', ') || 'none'}</p>
            <p>Remove: ${preview.installed_fingerprints.filter(k => !preview.saved_fingerprints.includes(k)).join(', ') || 'none'}</p>` : ''}</div>
          <label ?hidden=${!preview || preview.saved_fingerprints.length !== 0}><input type="checkbox" .checked=${this.emptyConfirmed} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) this.emptyConfirmed = e.target.checked;}}> Remove all managed keys from this account: new SSH logins using them will be denied.</label>
          <button type="button" class="ui primary button" ?hidden=${!preview} ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.applyKeys())}>Apply reviewed saved keys to this project</button>
        </fieldset>
      </section>
      <div class="soda-spaces-summary">
        <p id="sodaspaces-repository" class="soda-spaces-context">${this.repository}</p>
        <p id="sodaspaces-actor">${this.session ? `Soda account: ${this.session.user.login} (ID ${this.session.user.id})` : ''}</p>
        <p id="sodaspaces-login" class="soda-spaces-context">${this.connection ? 'Project account: ' + this.detail?.login : ''}</p>
        <p id="sodaspaces-status" role="status">${this.status}</p><p id="sodaspaces-result" role="status">${this.outcome}</p>
      </div>
    </section>`;
  }
  private retireTerminal() {
    this.terminal?.dispose(); this.terminal = undefined; this.terminalTarget = undefined;
    this.querySelector('#sodaspaces-view-terminal')?.replaceChildren();
  }
  private reset() {
    this.environment = this.detail = this.keyPreview = this.saved = this.lifecycle = this.connection = undefined;
    this.draft = ''; this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
  }
  invalidate() {
    ++this.epoch; this.stale = true; this.readController?.abort(); this.retireTerminal(); this.reset(); this.session = undefined;
    this.repository = ''; this.connectVisible = false;
    this.status = 'Page context changed. Reload the full repository page; no action was replayed or undone.';
  }
  private async api(path: string, method = 'GET', body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    const headers: Record<string, string> = {};
    const actor = method === 'POST' && path === '/api/session/logout' ? this.session?.user.id : this.binding?.expectedUserId;
    if (path !== '/api/session' && actor) headers['X-Soda-Expected-User-ID'] = actor;
    if (method !== 'GET') { if (!this.session) throw Error('Missing Soda session'); Object.assign(headers, {'Content-Type': 'application/json', 'X-CSRF-Token': this.session.csrf_token}); }
    const response = await fetch('/-/soda' + path, {method, headers, ...(body === undefined ? {} : {body: JSON.stringify(body)}), credentials: 'same-origin', cache: 'no-store', redirect: 'error', ...(signal ? {signal} : {})});
    if (!response.ok) {
      let code: string | undefined;
      try {const error = object(object(await readSodaJSON(response)).error); if (typeof error.code === 'string') code = error.code;} catch { /* Never display a response body. */ }
      throw new SodaRequestError(response.status, code);
    }
    return response.status === 204 ? null : readSodaJSON(response);
  }
  async refresh() {
    if (this.busy || this.stale || this.disposed || !this.binding) return;
    const {expectedUserId, repositoryId} = this.binding;
    const n = ++this.epoch; this.readController?.abort(); this.reset();
    const control = this.readController = new AbortController(), timeout = window.setTimeout(() => control.abort(), 15000);
    this.busy = true; this.session = undefined; this.connectVisible = false; this.status = 'Checking your account and environment…';
    try {
      const found = sessionResponse(await this.api('/api/session', 'GET', undefined, control.signal), location.origin); if (!this.active(n)) return;
      this.session = found;
      if (!expectedUserId || found.user.id !== expectedUserId) {this.retireTerminal(); this.connectVisible = true; this.status = 'Soda and native page identities differ. Sign out of Soda or reload and connect explicitly.'; return;}
      const provider = object(await this.api('/api/forgejo/me', 'GET', undefined, control.signal)); if (!this.active(n)) return; check(provider.id === expectedUserId);
      const collection = object(await this.api('/api/environments?repository_id=' + repositoryId, 'GET', undefined, control.signal)), repository = object(collection.repository); if (!this.active(n)) return;
      check(repository.id === repositoryId && Array.isArray(collection.items) && collection.items.length <= 1 && typeof collection.can_create === 'boolean');
      this.repository = `Repository ${repository.owner || ''}/${repository.name || ''} · ID ${repositoryId}`;
      if (!collection.items.length) {this.canCreate = collection.can_create; this.status = 'No shared environment. Creation is owner-only and does not join you.'; return;}
      const environment = environmentResponse(collection.items[0], repositoryId);
      this.environment = environment;
      const detail = detailResponse(await this.api(`/api/environments/${environment.id}`, 'GET', undefined, control.signal), environment); if (!this.active(n)) return;
      this.detail = detail;
      const running = this.running;
      this.status = !detail.environment.provisioned ? 'Provisioning incomplete. Ask the operator to inspect; do not recreate it.' : detail.native_unavailable || !detail.observed ? 'Native state unavailable; refresh or ask the operator to inspect.' : running ? 'Environment running.' : 'Environment stopped.';
      if (detail.environment.provisioned && !running && !detail.native_unavailable && !detail.environment_administrator) this.status += ' Ask the project administrator or Soda operator to Start it; nothing is started automatically.';
      const saved = savedKeysResponse(await this.api('/api/me/development-keys', 'GET', undefined, control.signal)); if (!this.active(n)) return; this.saved = saved;
      if (detail.environment.provisioned && detail.environment_administrator) {
        const state = object(await this.api(`/api/environments/${environment.id}/lifecycle`, 'GET', undefined, control.signal)), native = object(state.environment); if (!this.active(n)) return;
        check(native.id === environment.id && typeof native.running === 'boolean' && typeof state.boot_enabled === 'boolean');
        this.lifecycle = {running: native.running, boot: state.boot_enabled};
      }
      if (detail.login && running) {
        check(/^[a-z][a-z0-9_-]{0,30}$/.test(detail.login) && detail.login !== 'root');
        const own = object(await this.api(`/api/environments/${environment.id}/connection`, 'GET', undefined, control.signal)); if (!this.active(n)) return;
        const c = object(own.connection), native = object(c.environment);
        check(own.login === detail.login && native.id === environment.id && native.running && typeof native.ip === 'string' && /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(native.ip) && native.ip.split('.').every(v => Number(v) <= 255) && fingerprint(c.fingerprint));
        await this.updateComplete; if (!this.active(n)) return;
        check(this.ownerDocument.querySelectorAll('#sodaspaces-command').length === 1);
        this.connection = {command: `ssh ${detail.login}@${native.ip}`, fingerprint: c.fingerprint};
        if (!this.chosen) this.selected = 'terminal';
        await this.attachView(n);
      }
    } catch (error) {
      if (!this.active(n)) return;
      const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
      this.reset(); this.repository = '';
      if (e.status === 401 || e.status === 403) {this.retireTerminal(); this.session = undefined;}
      this.connectVisible = e.status === 401 || e.status === 403;
      this.status = this.connectVisible ? 'Connect through Forgejo to authorize Soda access.' : 'Could not confirm state. Refresh; do not infer absence or retry an uncertain action.';
    } finally {window.clearTimeout(timeout); if (this.active(n)) {this.busy = false; await this.updateComplete;}}
  }
  private async mutate(path: string, body: Record<string, unknown>, message: string, method = 'POST') {
    if (this.busy || this.stale || this.disposed || (this.uncertain && path !== '/api/session/logout') || !this.session) return;
    const n = this.epoch; this.busy = true; this.outcome = 'Checking current authorization…';
    let dispatched = false;
    const controller = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 255000);
    try {
      const current = sessionResponse(await this.api('/api/session', 'GET', undefined, controller.signal), location.origin);
      if (!this.active(n)) return;
      check(current.user.id === this.session.user.id && current.csrf_token === this.session.csrf_token && current.forgejo_url === location.origin);
      if (path !== '/api/session/logout') {
        check(current.user.id === this.binding?.expectedUserId);
        const provider = object(await this.api('/api/forgejo/me', 'GET', undefined, controller.signal)); if (!this.active(n)) return; check(provider.id === this.binding?.expectedUserId);
      }
      dispatched = true; this.outcome = 'Request dispatched. Closing does not cancel or undo native work.';
      const raw = await this.api(path, method, body, controller.signal), result = raw === null ? null : object(raw); if (!this.active(n)) return;
      if (path === '/api/environments') check(result && projectId(result.id) && result.repository_id === this.binding?.repositoryId && result.provisioned === true);
      else if (path.endsWith('/join')) check(typeof result?.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(result.login) && result.login !== 'root');
      else if (path.endsWith('/lifecycle')) check(result && object(result.environment).id === this.environment?.id && object(result.environment).running === (body.action === 'start') && result.boot_enabled === (body.action === 'start'));
      else if (path.endsWith('/access-keys')) check(result?.applied === true && result.login === this.detail?.login && typeof result.revision === 'string' && /^[0-9a-f]{64}$/.test(result.revision) && JSON.stringify(result.installed_fingerprints) === JSON.stringify(body.saved_fingerprints));
      else if (method === 'DELETE') check(result?.removed === true && result.existing_project_access_changed === false);
      else if (path === '/api/me/development-keys') {const keys = savedKeysResponse(result); check(typeof body.public_key === 'string'); const publicKey = body.public_key; check(keys.some(k => k.public_key?.trim().split(/\s+/).slice(0, 2).join(' ') === publicKey.trim().split(/\s+/).slice(0, 2).join(' ')));}
      else if (path === '/api/session/logout') check(result === null);
      this.outcome = message;
      if (path === '/api/session/logout') {this.invalidate(); this.status = 'Signed out of Soda, not Forgejo or Linux. Reload to connect again.'; return;}
      this.busy = false; await this.refresh();
    } catch (error) {
      if (!this.active(n)) return;
      const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
      this.uncertain = dispatched && !rejected.has(e.status);
      const reasons: Record<string, string> = {unsupported_linux_login: 'Your Forgejo username is not supported as a Linux login. No automatic rename is performed.', development_key_required: 'Save a development public key before joining.', invalid_public_key: 'Provide one public SSH key without options or private key material.', saved_keys_changed: 'Saved keys changed. Review them again before Apply.', owner_required: 'Only the current human repository owner can create this environment.', not_provisioned: 'Provisioning is incomplete. Ask the operator to inspect; do not recreate it.'};
      const reason = reasons[e.code || ''];
      this.outcome = !this.uncertain && reason ? reason : this.uncertain ? 'Outcome unconfirmed. Ask the operator to inspect; do not repeat, recreate or repair. Refresh reads state only.' : 'Request rejected. Refresh and review current state before another explicit action.';
    } finally {window.clearTimeout(timeout); if (this.active(n)) this.busy = false;}
  }
  private changeLifecycle(stop: boolean) {
    if (!this.environment) return;
    if (stop && !this.stopConfirmed) {this.outcome = 'Confirm the shared impact before Stop.'; return;}
    if (stop) this.terminal?.invalidate();
    return this.mutate(`/api/environments/${this.environment.id}/lifecycle`, {action: stop ? 'stop' : 'start', ...(stop ? {confirm_stop: true} : {})}, stop ? 'Stop confirmed; next-boot start disabled. Existing data was not recreated or deleted.' : 'Start confirmed and next-boot start enabled. Refresh connection status while services initialize.');
  }
  private saveKey() {
    const value = this.draft.trim();
    if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) {this.outcome = 'Provide one public SSH key. Never upload a private key.'; return;}
    this.draft = ''; return this.mutate('/api/me/development-keys', {public_key: value}, 'Public key saved for future joins. Existing project access is unchanged until explicitly applied.');
  }
  private async reviewKeys() {
    if (this.blocked || !this.environment || !this.detail) return;
    const n = this.epoch; this.busy = true;
    const controller = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 15000);
    try {
      const preview = keyPreviewResponse(await this.api(`/api/environments/${this.environment.id}/access-keys`, 'GET', undefined, controller.signal), this.detail.login);
      if (!this.active(n)) return;
      this.keyPreview = preview; this.emptyConfirmed = false;
    } catch {if (this.active(n)) {this.keyPreview = undefined; this.outcome = 'Key preview unavailable or changed. No update was requested; refresh and inspect.';}}
    finally {window.clearTimeout(timeout); if (this.active(n)) this.busy = false;}
  }
  private applyKeys() {
    if (!this.keyPreview || !this.environment) return;
    if (!this.keyPreview.saved_fingerprints.length && !this.emptyConfirmed) {this.outcome = 'Explicitly confirm removal of the last managed key.'; return;}
    return this.mutate(`/api/environments/${this.environment.id}/access-keys`, {revision: this.keyPreview.revision, saved_fingerprints: this.keyPreview.saved_fingerprints, confirm_empty: !this.keyPreview.saved_fingerprints.length}, 'Managed SSH key file updated for this project. Verify new-key login and old-key refusal from your SSH client. Existing sessions and browser access are not revoked.');
  }
  // Called only by the authorized refresh or deliberate Return command, never render.
  // A completed hidden read may display its result later, but cannot mount/attach now.
  private async attachView(epoch: number) {
    await this.updateComplete;
    if (!this.active(epoch) || !this.isConnected || this.closest('[hidden]') || !this.connection || !this.environment || !this.detail?.login || !this.binding?.expectedUserId) return;
    const target = this.environment.id + ':' + this.detail.login;
    if (this.terminal && this.terminalTarget !== target) this.retireTerminal();
    if (!this.terminal) {
      const mount = this.querySelector<HTMLElement>('#sodaspaces-view-terminal'); check(mount);
      this.terminalTarget = target;
      this.terminal = this.factory(mount, {expectedUserId: this.binding.expectedUserId, repositoryId: this.binding.repositoryId, environmentId: this.environment.id, login: this.detail.login});
      this.terminal.restore?.(); // Existing facade reauthorizes and attaches an exact ID only.
    }
  }
  retain() { return this.terminal?.retain?.(); }
  async returnToWork() {
    const epoch = this.epoch; await this.attachView(epoch);
    if (this.active(epoch) && !this.closest('[hidden]')) await this.terminal?.returnToWork?.();
  }
  dispose() { if (this.disposed) return; this.invalidate(); this.disposed = true; this.lifetime.abort(); this.remove(); }
}
customElements.define('soda-spaces', SodaSpaces);

export function mountSodaspaces(root: HTMLElement, context: DrawerContext, terminalFactory: TerminalFactory = mountTerminal) {
  if ((context.expectedUserId !== undefined && !id(context.expectedUserId)) || !id(context.repositoryId) || root.ownerDocument !== document) throw Error('Invalid native page context');
  const box = new SodaSpaces(); box.configure(context, terminalFactory); root.append(box);
  return {refresh: () => box.refresh(), invalidate: () => box.invalidate(), retain: () => box.retain(), returnToWork: () => box.returnToWork(),
    get ready() {return box.updateComplete;}, dispose: () => box.dispose()};
}

import {signOut} from './soda-connection.js';
// Soda owns only this mount. Native forms, authentication and terminal lifetime
// are not rendering concerns; commands remain explicit and generation guarded.
import {LitElement, html} from 'lit';
import {renderEnvironment, renderProjectOS, renderOSObservation} from './sodaspaces-environment-view.js';
import {renderConnection, renderKeys, renderForgejoKeys, renderProjectStatus} from './sodaspaces-project-view.js';
import type {TemplateResult} from 'lit';
import {object, check, id, creationProfile, osObservation, projectId, fingerprint, sessionResponse, environmentResponse, detailResponse, savedKeysResponse, profileKeysResponse, keyPreviewResponse, SodaRequestError, readSodaJSON} from './sodaspaces-api.js';
import type {OSObservation, CreationProfile, Session, Environment, Detail, KeyPreview, SavedKey, ProfileKeys} from './sodaspaces-api.js';
export interface ProjectContext {
  expectedUserId?: string | undefined;
  repositoryId: string;
  page?: boolean;
  settings?: boolean;
}
const rejected = new Set([400, 401, 403, 404, 409, 413, 415, 422]);
const views = ['environment', 'access'] as const;
type View = typeof views[number];
type Lifecycle = {
  running: boolean;
  boot: boolean;
};
// Stateless presentation: the concrete project owner admits every command and
// retains the original target, confirmation and asynchronous request lifetime.
function renderLifecycle(lifecycle: Lifecycle | undefined, blocked: boolean, confirmed: boolean, start: (event: MouseEvent) => void, stop: (event: MouseEvent) => void, confirm: (checked: boolean) => void): TemplateResult {
  const stopped = !lifecycle?.running && !lifecycle?.boot;
  return html`
    <fieldset ?hidden=${!lifecycle}>
      <legend>Shared environment</legend>
      <p>${lifecycle ? `Running: ${lifecycle.running ? 'yes' : 'no'}; starts on host boot: ${lifecycle.boot ? 'yes' : 'no'}. Start restores boot start; Stop disables it.` : ''}</p>
      <button type="button" class="ui primary button" ?hidden=${!!lifecycle?.running && lifecycle.boot}
        ?disabled=${blocked} @click=${start}>Start</button>
      <button type="button" class="ui button danger" ?hidden=${stopped}
        ?disabled=${blocked} @click=${stop}>Stop</button>
      <label ?hidden=${stopped}>
        <input type="checkbox" .checked=${confirmed}
          @change=${(event: Event) => {
      if (event.target instanceof HTMLInputElement)
        confirm(event.target.checked);
    }}>
        I understand Stop interrupts everyone’s SSH, terminals and workloads, and disables next-boot start.
      </label>
    </fieldset>
  `;
}
export class SodaProjectControls extends LitElement {
  static properties = {
    observedOS: {
      state: true
    }, osStatus: {
      state: true
    }, profiles: {
      state: true
    }, selectedProfile: {
      state: true
    }, status: {
      state: true
    }, outcome: {
      state: true
    }, repository: {
      state: true
    }, session: {
      state: true
    },
    environment: {
      state: true
    }, detail: {
      state: true
    }, saved: {
      state: true
    }, keyPreview: {
      state: true
    },
    profileKeys: {
      state: true
    }, lifecycle: {
      state: true
    }, connection: {
      state: true
    }, busy: {
      state: true
    }, stale: {
      state: true
    },
    uncertain: {
      state: true
    }, connectVisible: {
      state: true
    }, canCreate: {
      state: true
    },
    selected: {
      state: true
    }, draft: {
      state: true
    }, stopConfirmed: {
      state: true
    }, emptyConfirmed: {
      state: true
    }, useSavedKeys: {
      state: true
    },
  };
  declare private observedOS: OSObservation | undefined;
  declare private osStatus: string;
  declare private profiles: CreationProfile[];
  declare private selectedProfile: string;
  declare private status: string;
  declare private outcome: string;
  declare private repository: string;
  declare private session: Session | undefined;
  declare private environment: Environment | undefined;
  declare private detail: Detail | undefined;
  declare private saved: SavedKey[] | undefined;
  declare private profileKeys: ProfileKeys | undefined;
  declare private keyPreview: KeyPreview | undefined;
  declare private lifecycle: Lifecycle | undefined;
  declare private connection: {
    command: string;
    fingerprint: string;
  } | undefined;
  declare private busy: boolean;
  declare private stale: boolean;
  declare private uncertain: boolean;
  declare private connectVisible: boolean;
  declare private canCreate: boolean;
  declare private selected: View;
  declare private draft: string;
  declare private stopConfirmed: boolean;
  declare private emptyConfirmed: boolean;
  declare private useSavedKeys: boolean;
  private binding: ProjectContext | undefined;
  private readController: AbortController | undefined;
  private lifetime = new AbortController();
  private epoch = 0;
  private disposed = false;
  constructor() {
    super();
    this.profiles = [];
    this.selectedProfile = '';
    this.observedOS = undefined;
    this.osStatus = '';
    this.status = 'Refresh to inspect your shared environment.';
    this.outcome = this.repository = this.draft = '';
    this.session = this.environment = this.detail = this.saved = this.keyPreview = this.lifecycle = this.connection = this.profileKeys = undefined;
    this.busy = this.stale = this.uncertain = this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
    this.connectVisible = true;
    this.selected = 'environment';
    this.useSavedKeys = false;
  }
  protected createRenderRoot() {
    return this;
  }
  configure(context: ProjectContext) {
    if (this.binding || this.disposed)
      throw Error('Project binding is immutable');
    this.binding = {
      ...context
    };
    window.addEventListener('soda-session-retired', () => this.invalidate(), {signal: this.lifetime.signal});
    window.addEventListener('pagehide', () => this.invalidate(), {
      signal: this.lifetime.signal
    });
    window.addEventListener('pageshow', e => {
      if (e.persisted)
        this.invalidate();
    }, {
      signal: this.lifetime.signal
    });
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  private get blocked() {
    return this.busy || this.stale || this.uncertain || this.disposed;
  }
  private get running() {
    return this.detail?.environment.provisioned === true && !this.detail.native_unavailable && this.detail.observed?.running === true;
  }
  private active(epoch: number) {
    return !this.disposed && !this.stale && this.epoch === epoch;
  }
  private select(view: View) {
    this.selected = view;
  }
  private async tabKey(event: KeyboardEvent, view: View) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key))
      return;
    event.preventDefault();
    const i = views.indexOf(view), next = event.key === 'Home' ? 0 : event.key === 'End' ? views.length - 1 : (i + (event.key === 'ArrowRight' ? 1 : -1) + views.length) % views.length;
    const target = views[next];
    if (!target)
      return;
    this.select(target);
    const epoch = this.epoch;
    await this.updateComplete;
    if (this.active(epoch) && !this.closest('[hidden], [inert]') && this.selected === target)
      this.querySelector<HTMLElement>('[data-view="' + target + '"]')?.focus();
  }
  // Busy is checked synchronously, not only via Lit's eventually updated disabled attribute.
  private command(event: Event, action: () => void | Promise<void>, allowUncertain = false) {
    const button = event.currentTarget;
    if (!(button instanceof HTMLButtonElement) || button.disabled || button.closest('[hidden], [inert]') || this.busy || this.stale || this.disposed || (this.uncertain && !allowUncertain))
      return;
    void action();
  }
  protected render() {
    const intent = new URLSearchParams({
      repository_id: this.binding?.repositoryId || ''
    });
    if (this.binding?.settings)
      intent.set('destination', 'repository-spaces');
    if (this.binding?.expectedUserId)
      intent.set('expected_user_id', this.binding.expectedUserId);
    return html`<section data-project-controls data-repository-id=${this.binding?.repositoryId || ''} data-environment-id=${this.environment?.id || ''} class="soda-spaces-controls" aria-busy=${this.busy && !this.stale ? 'true' : 'false'}>
      <div class="soda-spaces-tabs" role="tablist" aria-label="Workspace views">${views.map(view => html`
        <button type="button" class="ui basic button" data-view=${view} role="tab" aria-controls=${'soda-project-' + this.binding?.repositoryId + '-' + view}
          aria-selected=${this.selected === view ? 'true' : 'false'} tabindex=${this.selected === view ? 0 : -1}
          @click=${() => this.select(view)} @keydown=${(e: KeyboardEvent) => this.tabKey(e, view)}>${view[0]?.toUpperCase()}${view.slice(1)}</button>`)}</div>
      <section id=${'soda-project-' + this.binding?.repositoryId + '-environment'} class="soda-spaces-view" role="tabpanel" aria-label="Environment" ?hidden=${this.selected !== 'environment'}>
        ${renderProjectOS(this.profiles, this.selectedProfile, this.detail?.environment || this.environment, this.blocked, value => {
      if (this.profiles.some(p => p.id === value))
        this.selectedProfile = value;
    })}
        ${this.environment ? renderOSObservation(this.observedOS, this.osStatus, this.blocked, event => this.command(event, () => this.inspectOS())) : ''}
        ${renderEnvironment({
      connectURL: '/-/soda/login?' + intent, connectVisible: this.connectVisible,
      busy: this.busy, stale: this.stale, signedIn: !!this.session, blocked: this.blocked, canCreate: this.canCreate,
      canJoin: !!this.detail?.environment.provisioned && !this.detail.login && this.running && !!this.saved,
      sshKeys: this.saved?.map(key => key.fingerprint) || [], useSavedKeys: this.useSavedKeys
    }, {
      selectSSH: checked => {
        this.useSavedKeys = checked;
      },
      connect: event => {
        if (this.busy || this.stale || this.disposed)
          event.preventDefault();
      },
      refresh: event => this.command(event, () => this.refresh(), true),
      reload: event => {
        if (!this.disposed && event.currentTarget instanceof HTMLElement && !event.currentTarget.closest('[hidden], [inert]'))
          window.location.reload();
      },
      logout: event => this.command(event, () => signOut(this.session?.user.id || ''), true),
      create: event => this.command(event, () => this.mutate('/api/environments', {
        repository_id: this.binding?.repositoryId, profile_id: this.selectedProfile
      }, 'Environment created. Explicitly Join for a browser terminal; external SSH keys are optional. Creation does not join you.')),
      join: event => this.command(event, () => this.joinEnvironment()),
    }, renderLifecycle(this.lifecycle, this.blocked, this.stopConfirmed, event => this.command(event, () => this.changeLifecycle(false)), event => this.command(event, () => this.changeLifecycle(true)), checked => {
      this.stopConfirmed = checked;
    }))}
      </section>
      <section id=${'soda-project-' + this.binding?.repositoryId + '-access'} class="soda-spaces-view" role="tabpanel" aria-label="Access" ?hidden=${this.selected !== 'access'}>
        ${renderConnection(this.connection, this.binding?.repositoryId || '', !!this.binding?.page, this.blocked, () => {
      void this.copyConnection();
    })}
        ${renderKeys({
      saved: this.saved, preview: this.keyPreview, joined: !!this.detail?.login, running: this.running,
      blocked: this.blocked, draft: this.draft, emptyConfirmed: this.emptyConfirmed
    }, {
      remove: (event, key) => this.command(event, () => this.mutate('/api/me/development-keys/' + key.id, {}, 'Saved key removed. Existing project SSH access is unchanged until explicitly applied.', 'DELETE')),
      draft: value => {
        this.draft = value;
      }, save: event => this.command(event, () => this.saveKey()),
      review: event => this.command(event, () => this.reviewKeys()), confirmEmpty: checked => {
        this.emptyConfirmed = checked;
      },
      apply: event => this.command(event, () => this.applyKeys()),
    })}
        ${this.saved ? renderForgejoKeys(this.profileKeys, this.blocked, page => {
      void this.reviewProfileKeys(page);
    }, key => this.selectForgejoKey(key)) : ''}
      </section>
      ${renderProjectStatus(this.repository, this.session ? `Soda account: ${this.session.user.login} (ID ${this.session.user.id})` : '', this.connection ? 'Project account: ' + this.detail?.login : '', this.status, this.outcome)}
    </section>`;
  }
  private joinEnvironment() {
    if (!this.environment) return;
    return this.mutate('/api/environments/' + this.environment.id + '/join', {
      ssh_keys: this.useSavedKeys ? 'saved' : 'none'
    }, 'Native join confirmed. Your browser terminal uses this account, not SSH. Later SSH-key changes require a separate explicit Apply.');
  }
  private selectForgejoKey(key: string) {
    if (this.blocked || this.selected !== 'access' || this.closest('[hidden], [inert]')) return;
    this.draft = key;
    this.outcome = 'Review the selected public key above, then explicitly Save public key. Joining/applying to a project remains a separate action.';
  }
  private async copyConnection() {
    if (!this.binding?.page || this.blocked || !this.connection || this.selected !== 'access' || this.closest('[hidden], [inert]'))
      return;
    try {
      await navigator.clipboard.writeText(this.connection.command);
      if (!this.disposed)
        this.outcome = 'SSH connection copied.';
    }
    catch {
      if (!this.disposed)
        this.outcome = 'Copy failed. Select and copy the displayed SSH command.';
    }
  }
  private reset() {
    this.environment = this.detail = this.keyPreview = this.saved = this.lifecycle = this.connection = this.profileKeys = undefined;
    this.profiles = [];
    this.observedOS = undefined;
    this.osStatus = '';
    this.draft = '';
    this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
  }
  invalidate() {
    ++this.epoch;
    this.stale = true;
    this.readController?.abort();
    this.reset();
    this.session = undefined;
    this.repository = '';
    this.connectVisible = false;
    this.status = 'Page context changed. Reload the full repository page; no action was replayed or undone.';
  }
  private async api(path: string, method = 'GET', body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    const headers: Record<string, string> = {};
    const actor = this.binding?.expectedUserId;
    if (path !== '/api/session' && actor)
      headers['X-Soda-Expected-User-ID'] = actor;
    if (method !== 'GET') {
      if (!this.session)
        throw Error('Missing Soda session');
      Object.assign(headers, {
        'Content-Type': 'application/json', 'X-CSRF-Token': this.session.csrf_token
      });
    }
    const response = await fetch('/-/soda' + path, {
      method, headers, ...(body === undefined ? {} : {
        body: JSON.stringify(body)
      }), credentials: 'same-origin', cache: 'no-store', redirect: 'error', ...(signal ? {
        signal
      } : {})
    });
    if (!response.ok) {
      let code: string | undefined;
      try {
        const error = object(object(await readSodaJSON(response)).error);
        if (typeof error.code === 'string')
          code = error.code;
      }
      catch { /* Never display a response body. */
      }
      throw new SodaRequestError(response.status, code);
    }
    return response.status === 204 ? null : readSodaJSON(response);
  }
  async refresh() {
    if (this.busy || this.stale || this.disposed || !this.binding)
      return;
    const {expectedUserId, repositoryId} = this.binding;
    const n = ++this.epoch;
    this.readController?.abort();
    this.reset();
    const control = this.readController = new AbortController(), timeout = window.setTimeout(() => control.abort(), 15000);
    this.busy = true;
    this.session = undefined;
    this.connectVisible = false;
    this.status = 'Checking your account and environment…';
    try {
      const found = sessionResponse(await this.api('/api/session', 'GET', undefined, control.signal), location.origin);
      if (!this.active(n))
        return;
      this.session = found;
      if (!expectedUserId || found.user.id !== expectedUserId) {
        this.connectVisible = true;
        this.status = 'Soda and native page identities differ. Sign out of Soda or reload and connect explicitly.';
        return;
      }
      const provider = object(await this.api('/api/forgejo/me', 'GET', undefined, control.signal));
      if (!this.active(n))
        return;
      check(provider.id === expectedUserId);
      const collection = object(await this.api('/api/environments?repository_id=' + repositoryId, 'GET', undefined, control.signal)), repository = object(collection.repository);
      if (!this.active(n))
        return;
      check(repository.id === repositoryId && Array.isArray(collection.items) && collection.items.length <= 1 && typeof collection.can_create === 'boolean');
      this.repository = `Repository ${repository.owner || ''}/${repository.name || ''} · ID ${repositoryId}`;
      if (!collection.items.length) {
        if (collection.can_create) {
          const available = object(await this.api('/api/repositories/' + repositoryId + '/profiles', 'GET', undefined, control.signal));
          if (!this.active(n))
            return;
          check(Array.isArray(available.items) && available.items.length === 1);
          this.profiles = available.items.map(creationProfile);
          this.selectedProfile = this.profiles[0]?.id || '';
          this.canCreate = !!this.selectedProfile;
        }
        this.status = 'No shared environment. Creation is owner-only and does not join you.';
        return;
      }
      const environment = environmentResponse(collection.items[0], repositoryId);
      this.environment = environment;
      const detail = detailResponse(await this.api(`/api/environments/${environment.id}`, 'GET', undefined, control.signal), environment);
      if (!this.active(n))
        return;
      this.detail = detail;
      const running = this.running;
      this.status = !detail.environment.provisioned ? 'Provisioning incomplete. Ask the operator to inspect; do not recreate it.' : detail.native_unavailable || !detail.observed ? 'Native state unavailable; refresh or ask the operator to inspect.' : running ? 'Environment running.' : 'Environment stopped.';
      if (detail.environment.provisioned && !running && !detail.native_unavailable && !detail.environment_administrator)
        this.status += ' Ask the project administrator or Soda operator to Start it; nothing is started automatically.';
      const saved = savedKeysResponse(await this.api('/api/me/development-keys', 'GET', undefined, control.signal));
      if (!this.active(n))
        return;
      this.saved = saved;
      if (detail.environment.provisioned && detail.environment_administrator) {
        const state = object(await this.api(`/api/environments/${environment.id}/lifecycle`, 'GET', undefined, control.signal)), native = object(state.environment);
        if (!this.active(n))
          return;
        check(native.id === environment.id && typeof native.running === 'boolean' && typeof state.boot_enabled === 'boolean');
        this.lifecycle = {
          running: native.running, boot: state.boot_enabled
        };
      }
      if (detail.login && running) {
        check(/^[a-z][a-z0-9_-]{0,30}$/.test(detail.login) && detail.login !== 'root');
        const own = object(await this.api(`/api/environments/${environment.id}/connection`, 'GET', undefined, control.signal));
        if (!this.active(n))
          return;
        const c = object(own.connection), native = object(c.environment);
        check(own.login === detail.login && native.id === environment.id && native.running && typeof native.ip === 'string' && /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(native.ip) && native.ip.split('.').every(v => Number(v) <= 255) && fingerprint(c.fingerprint));
        await this.updateComplete;
        if (!this.active(n))
          return;
        this.connection = {
          command: `ssh ${detail.login}@${native.ip}`, fingerprint: c.fingerprint
        };
      }
    }
    catch (error) {
      if (!this.active(n))
        return;
      const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
      this.reset();
      this.repository = '';
      if (e.status === 401 || e.status === 403) {
        this.session = undefined;
      }
      this.connectVisible = e.status === 401 || e.status === 403;
      this.status = e.code === 'profile_unavailable' ? 'Installed Project OS unavailable or incompatible. Nothing was reserved, pulled or started; ask the operator to inspect.' : this.connectVisible ? 'Connect through Forgejo to authorize Soda access.' : 'Could not confirm state. Refresh; do not infer absence or retry an uncertain action.';
    }
    finally {
      window.clearTimeout(timeout);
      if (this.active(n)) {
        this.busy = false;
        await this.updateComplete;
      }
    }
  }
  private async mutate(path: string, body: Record<string, unknown>, message: string, method = 'POST') {
    if (this.busy || this.stale || this.disposed || this.uncertain || !this.session)
      return;
    const n = this.epoch;
    this.busy = true;
    this.outcome = 'Checking current authorization…';
    let dispatched = false;
    const controller = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 255000);
    try {
      const current = sessionResponse(await this.api('/api/session', 'GET', undefined, controller.signal), location.origin);
      if (!this.active(n))
        return;
      check(current.user.id === this.session.user.id && current.csrf_token === this.session.csrf_token && current.forgejo_url === location.origin);
      check(current.user.id === this.binding?.expectedUserId);
      const provider = object(await this.api('/api/forgejo/me', 'GET', undefined, controller.signal));
      if (!this.active(n))
        return;
      check(provider.id === this.binding?.expectedUserId);
      dispatched = true;
      this.outcome = 'Request dispatched. Closing does not cancel or undo native work.';
      const raw = await this.api(path, method, body, controller.signal), result = raw === null ? null : object(raw);
      if (!this.active(n))
        return;
      if (path === '/api/environments')
        check(result && projectId(result.id) && result.repository_id === this.binding?.repositoryId && result.provisioned === true && creationProfile(result.profile).id === body.profile_id);
      else if (path.endsWith('/join'))
        check(typeof result?.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(result.login) && result.login !== 'root');
      else if (path.endsWith('/lifecycle'))
        check(result && object(result.environment).id === this.environment?.id && object(result.environment).running === (body.action === 'start') && result.boot_enabled === (body.action === 'start'));
      else if (path.endsWith('/access-keys'))
        check(result?.applied === true && result.login === this.detail?.login && typeof result.revision === 'string' && /^[0-9a-f]{64}$/.test(result.revision) && JSON.stringify(result.installed_fingerprints) === JSON.stringify(body.saved_fingerprints));
      else if (method === 'DELETE')
        check(result?.removed === true && result.existing_project_access_changed === false);
      else if (path === '/api/me/development-keys') {
        const keys = savedKeysResponse(result);
        check(typeof body.public_key === 'string');
        const publicKey = body.public_key;
        check(keys.some(k => k.public_key?.trim().split(/\s+/).slice(0, 2).join(' ') === publicKey.trim().split(/\s+/).slice(0, 2).join(' ')));
      }
      this.outcome = message;
      this.dispatchEvent(new CustomEvent('soda-project-changed', {
        bubbles: true, detail: {
          repositoryId: this.binding?.repositoryId
        }
      }));
      this.busy = false;
      await this.refresh();
    }
    catch (error) {
      if (!this.active(n))
        return;
      const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
      this.uncertain = dispatched && !rejected.has(e.status);
      const reasons: Record<string, string> = {
        profile_unavailable: 'Installed Project OS unavailable. No reservation was created; refresh before another explicit action.', unsupported_linux_login: 'Your Forgejo username is not supported as a Linux login. No automatic rename is performed.', invalid_public_key: 'Provide one public SSH key without options or private key material.', saved_keys_changed: 'Saved keys changed. Review them again before Apply.', owner_required: 'Only the current human repository owner can create this environment.', not_provisioned: 'Provisioning is incomplete. Ask the operator to inspect; do not recreate it.'
      };
      const reason = reasons[e.code || ''];
      this.outcome = !this.uncertain && reason ? reason : this.uncertain ? 'Outcome unconfirmed. Ask the operator to inspect; do not repeat, recreate or repair. Refresh reads state only.' : 'Request rejected. Refresh and review current state before another explicit action.';
    }
    finally {
      window.clearTimeout(timeout);
      if (this.active(n))
        this.busy = false;
    }
  }
  private async inspectOS() {
    if (this.blocked || !this.environment)
      return;
    const epoch = this.epoch, target = this.environment.id;
    this.busy = true;
    this.observedOS = undefined;
    this.osStatus = 'Reading current userspace…';
    this.readController?.abort();
    const controller = this.readController = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 15000);
    try {
      const observation = osObservation(await this.api('/api/environments/' + target + '/os', 'GET', undefined, controller.signal), target);
      if (this.active(epoch)) {
        this.observedOS = observation;
        this.osStatus = 'Read completed. Creation metadata was not changed.';
      }
    }
    catch (error) {
      if (!this.active(epoch))
        return;
      if (error instanceof SodaRequestError && (error.status === 401 || error.status === 403))
        this.invalidate();
      else
        this.osStatus = 'OS observation unavailable. Nothing was started or repaired.';
    }
    finally {
      window.clearTimeout(timeout);
      if (this.active(epoch))
        this.busy = false;
    }
  }
  private changeLifecycle(stop: boolean) {
    if (!this.environment)
      return;
    if (stop && !this.stopConfirmed) {
      this.outcome = 'Confirm the shared impact before Stop.';
      return;
    }
    return this.mutate(`/api/environments/${this.environment.id}/lifecycle`, {
      action: stop ? 'stop' : 'start', ...(stop ? {
        confirm_stop: true
      } : {})
    }, stop ? 'Stop confirmed; next-boot start disabled. Existing data was not recreated or deleted.' : 'Start confirmed and next-boot start enabled. Refresh connection status while services initialize.');
  }
  private saveKey() {
    const value = this.draft.trim();
    if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) {
      this.outcome = 'Provide one public SSH key. Never upload a private key.';
      return;
    }
    this.draft = '';
    return this.mutate('/api/me/development-keys', {
      public_key: value
    }, 'Public key saved for future joins. Existing project access is unchanged until explicitly applied.');
  }
  private async reviewProfileKeys(page: number) {
    if (this.blocked || this.selected !== 'access' || this.closest('[hidden], [inert]') || !Number.isInteger(page) || page < 1 || page > 8)
      return;
    const epoch = this.epoch;
    this.busy = true;
    const controller = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 20000);
    try {
      const keys = profileKeysResponse(await this.api('/api/me/forgejo-keys?page=' + page, 'GET', undefined, controller.signal), page);
      if (this.active(epoch))
        this.profileKeys = keys;
    }
    catch {
      if (this.active(epoch)) {
        this.profileKeys = undefined;
        this.outcome = 'Own Forgejo keys are unavailable. Nothing was imported; use native profile settings or explicitly paste a public key.';
      }
    }
    finally {
      window.clearTimeout(timeout);
      if (this.active(epoch))
        this.busy = false;
    }
  }
  private async reviewKeys() {
    if (this.blocked || !this.environment || !this.detail)
      return;
    const n = this.epoch;
    this.busy = true;
    const controller = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 15000);
    try {
      const preview = keyPreviewResponse(await this.api(`/api/environments/${this.environment.id}/access-keys`, 'GET', undefined, controller.signal), this.detail.login);
      if (!this.active(n))
        return;
      this.keyPreview = preview;
      this.emptyConfirmed = false;
    }
    catch {
      if (this.active(n)) {
        this.keyPreview = undefined;
        this.outcome = 'Key preview unavailable or changed. No update was requested; refresh and inspect.';
      }
    }
    finally {
      window.clearTimeout(timeout);
      if (this.active(n))
        this.busy = false;
    }
  }
  private applyKeys() {
    if (!this.keyPreview || !this.environment)
      return;
    if (!this.keyPreview.saved_fingerprints.length && !this.emptyConfirmed) {
      this.outcome = 'Explicitly confirm removal of the last managed key.';
      return;
    }
    return this.mutate(`/api/environments/${this.environment.id}/access-keys`, {
      revision: this.keyPreview.revision, saved_fingerprints: this.keyPreview.saved_fingerprints, confirm_empty: !this.keyPreview.saved_fingerprints.length
    }, 'Managed SSH key file updated for this project. Verify new-key login and old-key refusal from your SSH client. Existing sessions and browser access are not revoked.');
  }
  dispose() {
    if (this.disposed)
      return;
    this.invalidate();
    this.disposed = true;
    this.lifetime.abort();
    this.remove();
  }
}
customElements.define('soda-project-controls', SodaProjectControls);
export function mountProjectControls(root: HTMLElement, context: ProjectContext) {
  if ((context.expectedUserId !== undefined && !id(context.expectedUserId)) || !id(context.repositoryId) || root.ownerDocument !== document)
    throw Error('Invalid native page context');
  const box = new SodaProjectControls();
  box.configure(context);
  root.append(box);
  return {
    refresh: () => box.refresh(), invalidate: () => box.invalidate(),
    get ready() {
      return box.updateComplete;
    }, dispose: () => box.dispose()
  };
}

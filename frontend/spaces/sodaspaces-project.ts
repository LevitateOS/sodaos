import {renderWorkspaceIntro} from './sodaspaces-workspace-view.js';
import {signOut} from './soda-connection.js';
// Soda owns only this mount. Native forms, authentication and terminal lifetime
// are not rendering concerns; commands remain explicit and generation guarded.
import {LitElement, html} from 'lit';
import {renderEnvironment, renderProjectOS, renderOSObservation} from './sodaspaces-environment-view.js';
import {renderConnection, renderKeys, renderForgejoKeys, renderProjectStatus} from './sodaspaces-project-view.js';
import {renderNetwork, renderNetworkSelection} from './sodaspaces-network.js';
import {projectOptions, projectView} from '../tailnet/soda-tailnet-response.js';
import type {ProjectOptions, ProjectNetwork} from '../tailnet/soda-tailnet-response.js';
import type {TemplateResult} from 'lit';
import {object, check, id, creationProfile, osObservation, projectId, fingerprint, sessionResponse, environmentResponse, detailResponse, savedKeysResponse, profileKeysResponse, keyPreviewResponse, SodaRequestError, readSodaJSON} from './sodaspaces-api.js';
import type {OSObservation, CreationProfile, Session, Environment, Detail, KeyPreview, SavedKey, ProfileKeys} from './sodaspaces-api.js';
export interface ProjectContext {
  expectedUserId?: string | undefined;
  repositoryId: string;
  page?: boolean;
  settings?: boolean;
  session?: Session | undefined;
}
const rejected = new Set([400, 401, 403, 404, 409, 413, 415, 422]);
const views = ['environment', 'access', 'network'] as const;
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
    presentation: {state: true}, networkReview: {state: true},
    network: {state: true}, networkOptions: {state: true}, networkEnabled: {state: true}, networkConfirmed: {state: true}, networkNotice: {state: true},
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
    }, repositoryName: {
      state: true
    }, repositoryURL: {
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
  declare private presentation: 'standard' | 'journey' | 'settings';
  declare private networkReview: boolean;
  declare private network: ProjectNetwork | undefined;
  declare private networkOptions: ProjectOptions | undefined;
  declare private networkEnabled: boolean;
  declare private networkConfirmed: boolean;
  declare private networkNotice: string;
  declare private observedOS: OSObservation | undefined;
  declare private osStatus: string;
  declare private profiles: CreationProfile[];
  declare private selectedProfile: string;
  declare private status: string;
  declare private outcome: string;
  declare private repository: string;
  declare private repositoryName: string;
  declare private repositoryURL: string;
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
  private mutationPending = false;
  private outcomeNeedsAttention = false;
  get canRestore() { return !this.mutationPending; }
  constructor() {
    super();
    this.presentation = 'standard';
    this.networkReview = false;
    this.network = this.networkOptions = undefined;
    this.networkEnabled = this.networkConfirmed = false;
    this.networkNotice = '';
    this.profiles = [];
    this.selectedProfile = '';
    this.observedOS = undefined;
    this.osStatus = '';
    this.status = 'Refresh to inspect your shared environment.';
    this.outcome = this.repository = this.repositoryName = this.repositoryURL = this.draft = '';
    this.session = this.environment = this.detail = this.saved = this.keyPreview = this.lifecycle = this.connection = this.profileKeys = undefined;
    this.busy = this.stale = this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
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
    this.binding = {...context};
    check(!context.session || context.session.user.id === context.expectedUserId);
    this.session = context.session;
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
    return this.busy || this.stale || this.disposed;
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
  private command(event: Event, action: () => void | Promise<void>) {
    const button = event.currentTarget;
    if (!(button instanceof HTMLButtonElement) || button.disabled || button.closest('[hidden], [inert]') || this.blocked)
      return;
    void action();
  }
  setPresentation(presentation: 'standard' | 'journey' | 'settings') {
    if (this.presentation === presentation) return false;
    this.presentation = presentation; this.selected = 'environment';
    return true;
  }
  private createProject() {
    if (!this.canCreate || this.networkReview || !this.profiles.some(p => p.id === this.selectedProfile)) return;
    return this.mutate('/api/environments', {
      repository_id: this.binding?.repositoryId, profile_id: this.selectedProfile,
      tailnet: this.networkEnabled && this.networkOptions?.available ? {enabled: true, revision: this.networkOptions.revision, binding: this.networkOptions.binding} : {enabled: false}
    }, 'Project created. Join explicitly to set up your browser-terminal account.');
  }
  private renderJourney() {
    const joinReady = !this.stale && this.detail?.environment.provisioned && this.running && !this.detail.login && !this.detail.authority_unavailable && !this.detail.native_unavailable;
    if (joinReady) return html`<section data-project-controls data-repository-id=${this.binding?.repositoryId || ''} data-environment-id=${this.environment?.id || ''} class="soda-spaces-controls soda-project-journey soda-ready-project" aria-busy=${this.busy ? 'true' : 'false'}>
      ${renderWorkspaceIntro({kind: 'project', heading: 'Project created',
        description: html`Join ${this.repositoryName} to set up your personal account<span>on this shared development system.</span>`,
        action: html`<button class="ui primary button" ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.joinEnvironment())}>${this.mutationPending ? 'Joining project…' : 'Join project'}</button>`,
        helper: html`Then you can open your first browser terminal.`,
        feedback: html`<div class="soda-intro-feedback soda-feedback" data-tone=${this.mutationPending ? 'pending' : 'warning'} role="status">${this.mutationPending ? 'Setting up your project account…' : this.outcomeNeedsAttention ? this.outcome : ''}</div>${this.outcomeNeedsAttention && !this.busy ? html`<button class="ui button soda-quiet-action" ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.refresh())}>Refresh status</button>` : ''}`})}
    </section>`;

    if (this.environment) {
      const unavailable = !this.detail || this.detail.authority_unavailable || this.detail.native_unavailable || !this.detail.observed;
      const incomplete = this.detail && !this.detail.environment.provisioned;
      const stopped = !unavailable && this.detail?.observed?.running === false;
      const accountReady = !unavailable && this.running && !!this.detail?.login;
      return html`<section data-project-controls data-repository-id=${this.binding?.repositoryId || ''} data-environment-id=${this.environment.id} class="soda-spaces-controls soda-project-journey soda-ready-project" aria-busy=${this.busy ? 'true' : 'false'}>
        ${renderWorkspaceIntro({kind: this.busy ? 'loading' : accountReady ? 'terminal' : stopped ? 'stopped' : 'unavailable',
          heading: accountReady ? 'Your account is ready' : incomplete ? 'Project needs inspection' : stopped ? 'Project not ready' : 'Project status unavailable',
          description: html`${accountReady ? 'Your project account is ready for a browser terminal.' : incomplete ? 'Project setup is incomplete. Ask the operator to inspect this project.' : stopped ? 'This project is stopped. Start it before joining or opening a terminal.' : 'Current project access or runtime state could not be confirmed.'}`,
          action: html`${stopped && this.lifecycle ? html`<button class="ui primary button" ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.changeLifecycle(false))}>${this.mutationPending ? 'Starting project…' : 'Start project'}</button>` : ''}<button class=${stopped && this.lifecycle ? 'ui button soda-quiet-action' : 'ui primary button'} ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.refresh())}>${this.busy && !this.mutationPending ? 'Checking status…' : 'Refresh status'}</button>`,
          helper: html`${incomplete ? 'Do not recreate a reserved project.' : stopped && !this.lifecycle ? 'A project administrator must start this project.' : 'Refreshing checks the existing project without repeating an operation.'}`,
          feedback: html`<div class="soda-intro-feedback soda-feedback" data-tone="warning" role="status">${this.outcomeNeedsAttention ? this.outcome : ''}</div>`})}
      </section>`;
    }

    const configureReady = this.canCreate && !this.busy && !this.stale && !this.outcome && !this.networkReview;
    return html`<section data-project-controls data-repository-id=${this.binding?.repositoryId || ''} data-environment-id="" class="soda-spaces-controls soda-project-journey" aria-busy=${this.busy ? 'true' : 'false'}>
      <header class="soda-setup-step-heading"><p class="soda-setup-eyebrow">New project</p><h2 tabindex="-1">Configure project</h2><p>Choose the system for your project.</p></header>
        <div class="soda-configuration-fields"><div class="soda-configuration-repository"><p>Repository</p><div class="soda-config-repository"><span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span><span class="soda-config-repository-name">${this.repositoryName || 'Loading repository…'}<small>Forgejo</small></span><button class="ui button soda-quiet-action" aria-label="Change repository" ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => {this.dispatchEvent(new CustomEvent('soda-project-change-repository', {bubbles: true}));})}>Change</button></div></div>
        ${renderProjectOS(this.profiles, this.selectedProfile, undefined, this.blocked, value => {if (this.profiles.some(p => p.id === value)) this.selectedProfile = value;}, 'configure')}
        ${this.networkOptions?.available ? renderNetworkSelection(this.networkOptions, this.networkEnabled, this.blocked, enabled => {this.networkEnabled = enabled; this.networkReview = false;}, 'Enable project Tailnet') : ''}
        ${this.networkReview ? html`<p role="alert">Network availability changed. Review the option before creating.</p><button class="ui button" ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => {this.networkEnabled = this.networkReview = false;})}>Use without Tailnet</button>` : ''}
        </div><div class="soda-setup-actions"><button class="ui primary button" ?disabled=${this.blocked || !this.canCreate || this.networkReview} @click=${(e: Event) => this.command(e, () => this.createProject())}>${this.mutationPending ? 'Creating project…' : 'Create project'}</button></div>

      <p class="soda-feedback" data-tone=${this.busy ? 'pending' : 'warning'} role="status" ?hidden=${configureReady || this.mutationPending}>${this.status}</p><p class="soda-feedback" data-tone=${this.mutationPending ? 'pending' : 'warning'} role="status">${this.mutationPending ? 'Creating your project…' : this.outcome}</p>
      <button class="ui button soda-quiet-action" ?hidden=${configureReady || this.busy} ?disabled=${this.blocked} @click=${(e: Event) => this.command(e, () => this.refresh())}>Refresh status</button>
      ${this.stale ? html`<p>Access changed. Reload Spaces to reconnect; no operation will be repeated.</p>` : ''}
    </section>`;
  }
  protected render() {
    if (this.presentation === 'journey') return this.renderJourney();
    const intent = new URLSearchParams({
      repository_id: this.binding?.repositoryId || ''
    });
    if (this.binding?.settings)
      intent.set('destination', 'repository-spaces');
    if (this.binding?.expectedUserId)
      intent.set('expected_user_id', this.binding.expectedUserId);
    return html`<section data-project-controls data-repository-id=${this.binding?.repositoryId || ''} data-environment-id=${this.environment?.id || ''} class="soda-spaces-controls" aria-busy=${this.busy && !this.stale ? 'true' : 'false'}>
      ${this.binding?.settings && this.repositoryURL ? html`<div class="soda-repository-context">
        <h2>${this.repository}</h2>
        <nav aria-label="Repository destinations"><a href=${this.repositoryURL + '/settings'}>Native repository settings</a> · <a href=${this.repositoryURL + '#sodaspaces'}>Open repository and workspace drawer</a></nav>
        <p>Project OS selection affects only creation. Existing roots cannot change distribution or interface here. Create, Join and Start remain separate explicit actions.</p>
      </div>` : ''}
      <div class="soda-spaces-tabs" role="tablist" aria-label="Workspace views">${views.map(view => html`
        <button type="button" class="ui basic button" data-view=${view} role="tab" aria-controls=${'soda-project-' + this.binding?.repositoryId + '-' + view}
          aria-selected=${this.selected === view ? 'true' : 'false'} tabindex=${this.selected === view ? 0 : -1}
          @click=${() => this.select(view)} @keydown=${(e: KeyboardEvent) => this.tabKey(e, view)}>${view === 'environment' && this.presentation === 'settings' ? 'Overview' : (view[0]?.toUpperCase() || '') + view.slice(1)}</button>`)}</div>
      <section id=${'soda-project-' + this.binding?.repositoryId + '-environment'} class="soda-spaces-view" role="tabpanel" aria-label="Environment" ?hidden=${this.selected !== 'environment'}>
        ${renderProjectOS(this.profiles, this.selectedProfile, this.detail?.environment || this.environment, this.blocked, value => {
      if (this.profiles.some(p => p.id === value))
        this.selectedProfile = value;
    })}
        ${this.canCreate ? renderNetworkSelection(this.networkOptions, this.networkEnabled, this.blocked, enabled => {this.networkEnabled = enabled;}) : ''}
        ${this.environment ? html`<p data-project-network-summary>Tailnet: ${this.network ? (this.network.enabled ? 'managed' : 'Off') + ' · ' + this.network.state : 'not observed'}</p>` : ''}
        ${this.environment ? renderOSObservation(this.observedOS, this.osStatus, this.blocked, event => this.command(event, () => this.inspectOS())) : ''}
        ${renderEnvironment({
      connectURL: '/-/soda/login?' + intent, connectVisible: this.connectVisible,
      busy: this.busy, stale: this.stale, signedIn: !!this.session, blocked: this.blocked, canCreate: this.canCreate,
      canJoin: !!this.detail?.environment.provisioned && !this.detail.login && this.running && (!this.useSavedKeys || !!this.saved),
      sshKeys: this.saved?.map(key => key.fingerprint) || [], useSavedKeys: this.useSavedKeys
    }, {
      selectSSH: checked => {
        this.useSavedKeys = checked;
      },
      connect: event => {
        if (this.busy || this.stale || this.disposed)
          event.preventDefault();
      },
      refresh: event => this.command(event, () => this.refresh()),
      reload: event => {
        if (!this.disposed && event.currentTarget instanceof HTMLElement && !event.currentTarget.closest('[hidden], [inert]'))
          window.location.reload();
      },
      logout: event => this.command(event, () => signOut(this.session?.user.id || '')),
      create: event => this.command(event, () => this.createProject()),
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
      <section id=${'soda-project-' + this.binding?.repositoryId + '-network'} class="soda-spaces-view" role="tabpanel" aria-label="Network" ?hidden=${this.selected !== 'network'}>
        ${renderNetwork(this.network, this.networkNotice, !!this.detail?.environment_administrator, this.blocked, this.networkConfirmed,
          confirmed => {this.networkConfirmed = confirmed;}, (event, action) => this.command(event, () => this.changeNetwork(action)))}
        ${this.network?.state === 'connected' && this.connection && this.detail?.login ? html`<fieldset><legend>Own-account Tailnet SSH</legend>
          <input readonly aria-label="Tailnet SSH command" .value=${'ssh ' + this.detail.login + '@' + this.network.addresses[0]}>
          <p>Ed25519 host-key fingerprint: ${this.connection.fingerprint}. Same project account and host key as LAN SSH; no Tailscale SSH or automatic authentication.</p>
        </fieldset>` : ''}
      </section>
      ${renderProjectStatus(this.repository, this.session ? `Soda account: ${this.session.user.login} (ID ${this.session.user.id})` : '', this.connection ? 'Project account: ' + this.detail?.login : '', this.status, this.outcome)}
    </section>`;
  }
  private joinEnvironment() {
    if (!this.environment || !this.running || this.detail?.login || this.detail?.authority_unavailable) return;
    return this.mutate('/api/environments/' + this.environment.id + '/join', {
      ssh_keys: this.presentation !== 'journey' && this.useSavedKeys ? 'saved' : 'none'
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
    this.network = this.networkOptions = undefined;
    this.networkEnabled = this.networkConfirmed = false;
    this.networkNotice = '';
    this.repositoryURL = '';
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
    this.busy = false;
    this.readController?.abort();
    this.reset();
    this.session = undefined;
    this.repository = this.repositoryName = '';
    this.connectVisible = false;
    this.status = 'Page context changed. Reload the full repository page; no action was replayed or undone. A dispatched operation may still have completed.';
  }
  private async api(path: string, method = 'GET', body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    const headers: Record<string, string> = {};
    const actor = this.binding?.expectedUserId;
    if (actor)
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
      if (response.status === 401 || response.status === 403) this.invalidate();
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
    const prior = {profile: this.selectedProfile, network: this.networkOptions, enabled: this.networkEnabled};
    this.readController?.abort();
    this.reset();
    const control = this.readController = new AbortController(), timeout = window.setTimeout(() => control.abort(), 15000);
    this.busy = true;
    this.connectVisible = false;
    this.status = 'Checking your account and environment…';
    try {
      const found = this.session || sessionResponse(await this.api('/api/session', 'GET', undefined, control.signal), location.origin);
      if (!this.active(n))
        return;
      if (!expectedUserId || found.user.id !== expectedUserId) {
        this.invalidate();
        this.connectVisible = true;
        this.status = 'Soda and native page identities differ. Sign out of Soda or reload and connect explicitly.';
        return;
      }
      this.session = found;
      const collection = object(await this.api('/api/environments?repository_id=' + repositoryId, 'GET', undefined, control.signal)), repository = object(collection.repository);
      if (!this.active(n))
        return;
      check(repository.id === repositoryId && Array.isArray(collection.items) && collection.items.length <= 1 && typeof collection.can_create === 'boolean');
      check(typeof repository.owner === 'string' && typeof repository.name === 'string');
      for (const part of [repository.owner, repository.name]) check(part !== '' && part !== '.' && part !== '..' && part.length <= 255 && !/[\/\\\x00\r\n]/.test(part));
      this.repositoryName = `${repository.owner}/${repository.name}`;
      this.repository = `Repository ${repository.owner}/${repository.name} · ID ${repositoryId}`;
      this.repositoryURL = location.origin + '/' + encodeURIComponent(repository.owner) + '/' + encodeURIComponent(repository.name);
      if (!collection.items.length) {
        if (collection.can_create) {
          const available = object(await this.api('/api/repositories/' + repositoryId + '/profiles', 'GET', undefined, control.signal));
          if (!this.active(n))
            return;
          check(Array.isArray(available.items) && available.items.length === 1);
          this.profiles = available.items.map(creationProfile);
          this.selectedProfile = this.profiles.some(p => p.id === prior.profile) ? prior.profile : this.profiles[0]?.id || '';
          this.canCreate = !!this.selectedProfile;
          try {
            const options = projectOptions(await this.api('/api/repositories/' + repositoryId + '/tailnet-options', 'GET', undefined, control.signal));
            if (!this.active(n)) return;
            this.networkOptions = options;
            this.networkReview = prior.enabled && (!options.available || prior.network?.binding !== options.binding || prior.network.revision !== options.revision);
            this.networkEnabled = prior.enabled || (this.presentation !== 'journey' && options.available && options.default);
          } catch (error) {
            if (error instanceof SodaRequestError && error.status === 401) throw error;
            if (!this.active(n)) return;
            // A missing/off helper must not block legacy Off creation.
            this.networkOptions = undefined; this.networkEnabled = prior.enabled; this.networkReview = prior.enabled;
          }
        }
        this.status = this.presentation === 'journey' ? 'Creation is owner-only. You join separately after the project is ready.' : 'No shared environment. Creation is owner-only and does not join you.';
        return;
      }
      const environment = environmentResponse(collection.items[0], repositoryId);
      this.environment = environment;
      const detail = detailResponse(await this.api(`/api/environments/${environment.id}`, 'GET', undefined, control.signal), environment);
      if (!this.active(n))
        return;
      this.detail = detail;
      if (detail.login) check(/^[a-z][a-z0-9_-]{0,30}$/.test(detail.login) && detail.login !== 'root');
      const running = this.running;
      this.status = !detail.environment.provisioned ? 'Provisioning incomplete. Ask the operator to inspect; do not recreate it.' : detail.native_unavailable || !detail.observed ? 'Native state unavailable; refresh or ask the operator to inspect.' : running ? 'Environment running.' : 'Environment stopped.';
      if (detail.environment.provisioned && !running && !detail.native_unavailable && !detail.environment_administrator)
        this.status += ' Ask the project administrator or Soda operator to Start it; nothing is started automatically.';
      if (this.presentation !== 'journey') {
        try {
          const saved = savedKeysResponse(await this.api('/api/me/development-keys', 'GET', undefined, control.signal));
          if (!this.active(n)) return;
          this.saved = saved;
        } catch {
          if (!this.active(n)) return;
          this.outcome = 'External SSH keys unavailable. Browser-only Join remains independent.';
        }
      }
      if (detail.environment.provisioned && detail.environment_administrator) {
        const state = object(await this.api(`/api/environments/${environment.id}/lifecycle`, 'GET', undefined, control.signal)), native = object(state.environment);
        if (!this.active(n))
          return;
        check(native.id === environment.id && typeof native.running === 'boolean' && typeof state.boot_enabled === 'boolean');
        this.lifecycle = {
          running: native.running, boot: state.boot_enabled
        };
      }
      if (detail.login && running && this.presentation !== 'journey') {
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
      if (detail.environment.provisioned && (detail.login || detail.environment_administrator)) {
        try {
          const network = projectView(await this.api(`/api/environments/${environment.id}/tailnet`, 'GET', undefined, control.signal), environment.id);
          check(!network.saved);
          if (!this.active(n)) return;
          this.network = network;
        } catch (error) {
          if (error instanceof SodaRequestError && error.status === 401) throw error;
          if (!this.active(n)) return;
          this.network = undefined; this.networkNotice = 'Private network state unavailable. No disconnected state was inferred; ordinary project controls remain independent.';
        }
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
        this.dispatchEvent(new CustomEvent('soda-project-observed', {bubbles: true, detail: {
          repositoryId, environmentId: this.environment?.id || '', provisioned: this.detail?.environment.provisioned === true,
          login: this.detail?.login || '', running: this.running
        }}));
        await this.updateComplete;
      }
    }
  }
  private async mutate(path: string, body: Record<string, unknown>, message: string, method = 'POST') {
    if (this.blocked || !this.session)
      return;
    const n = this.epoch;
    this.busy = true;
    this.outcomeNeedsAttention = false;
    this.outcome = 'Sending the explicit operation with the original page identity…';
    let dispatched = false, inspectCreation = false;
    const controller = new AbortController(), timeout = window.setTimeout(() => controller.abort(), 255000);
    try {
      dispatched = true;
      this.mutationPending = true;
      this.dispatchEvent(new CustomEvent('soda-project-operation', {bubbles: true}));
      this.outcome = 'Request dispatched. Closing does not cancel or undo native work.';
      const raw = await this.api(path, method, body, controller.signal), result = raw === null ? null : object(raw);
      if (!this.active(n))
        return;
      if (path === '/api/environments') {
        check(result && projectId(result.id) && result.repository_id === this.binding?.repositoryId && result.provisioned === true && creationProfile(result.profile).id === body.profile_id);
        if (object(body.tailnet).enabled === true) {
          check(result.tailnet_outcome === 'queued' || result.tailnet_outcome === 'unconfirmed');
          this.outcomeNeedsAttention = true;
          message = result.tailnet_outcome === 'queued' ? 'Project created. Network policy saved; enrollment queued.' : 'Project created. Network setup unconfirmed; inspect Network and explicitly retry there. Do not recreate the project.';
        }
      }
      else if (path.endsWith('/join'))
        check(typeof result?.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(result.login) && result.login !== 'root');
      else if (path.endsWith('/lifecycle'))
        check(result && object(result.environment).id === this.environment?.id && object(result.environment).running === (body.action === 'start') && result.boot_enabled === (body.action === 'start'));
      else if (path.endsWith('/tailnet')) {
        const network = projectView(result, this.environment?.id || '');
        check(network.saved && network.revision !== body.revision && network.enabled === (body.action !== 'disable'));
        message = network.outcome === 'queued' ? 'Network policy saved; native work queued. Refresh observes the outcome without replay.' : 'Network policy saved; native outcome unconfirmed. Observe before retrying; no connection or disconnection was assumed.';
      }
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
      this.mutationPending = false;
      this.dispatchEvent(new CustomEvent('soda-project-operation', {bubbles: true}));
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
      const uncertain = dispatched && !rejected.has(e.status);
      if (uncertain && path === '/api/environments') {
        this.canCreate = false;
        inspectCreation = true;
        this.dispatchEvent(new CustomEvent('soda-project-changed', {bubbles: true, detail: {repositoryId: this.binding?.repositoryId}}));
      }
      this.mutationPending = false;
      const reasons: Record<string, string> = {
        profile_unavailable: 'Installed Project OS unavailable. No reservation was created; refresh before another explicit action.', unsupported_linux_login: 'Your Forgejo username is not supported as a Linux login. No automatic rename is performed.', invalid_public_key: 'Provide one public SSH key without options or private key material.', saved_keys_changed: 'Saved keys changed. Review them again before Apply.', owner_required: 'Only the current human repository owner can create this environment.', not_provisioned: 'Provisioning is incomplete. Ask the operator to inspect; do not recreate it.'
      };
      const reason = reasons[e.code || ''];
      this.outcomeNeedsAttention = true;
      this.outcome = !uncertain && reason ? reason : uncertain ? 'Outcome unconfirmed. Refresh the target before another explicit action; this does not confirm the earlier write. Never recreate a reserved project.' : 'Request rejected. Refresh and review current state before another explicit action.';
    }
    finally {
      window.clearTimeout(timeout);
      this.mutationPending = false;
      if (this.active(n)) {
        this.busy = false;
        if (inspectCreation) await this.refresh(); // Read-only recovery of the original repository, never another Create.
      }
    }
  }
  private changeNetwork(action: 'enable' | 'disable' | 'retry') {
    const network = this.network;
    if (!network || !this.environment || !this.detail?.environment_administrator || !this.networkConfirmed || this.blocked) return;
    if (action !== 'disable' && (!network.available_binding || (network.binding && network.binding !== network.available_binding))) return;
    this.networkConfirmed = false;
    return this.mutate('/api/environments/' + this.environment.id + '/tailnet', {
      action, revision: network.revision, confirm_id: network.project,
      ...(action === 'disable' ? {} : {binding: network.available_binding})
    }, 'Network policy saved; observe the native outcome.');
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
    const preview = this.keyPreview;
    this.keyPreview = undefined; this.emptyConfirmed = false;
    return this.mutate(`/api/environments/${this.environment.id}/access-keys`, {
      revision: preview.revision, saved_fingerprints: preview.saved_fingerprints, confirm_empty: !preview.saved_fingerprints.length
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
    get canRestore() { return box.canRestore; },
    refresh: () => box.refresh(), invalidate: () => box.invalidate(),
    setPresentation: (mode: 'standard' | 'journey' | 'settings') => box.setPresentation(mode),
    get ready() {
      return box.updateComplete;
    }, dispose: () => box.dispose()
  };
}

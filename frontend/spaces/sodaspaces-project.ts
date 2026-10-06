import './soda-identity.js';
import type {PreparedExtensionMount} from './soda-extension.js';
// Soda owns only this mount. Native forms, authentication and terminal lifetime
// are not rendering concerns; commands remain explicit and generation guarded.
import {LitElement, html} from 'lit';
import {renderProjectStatus} from './sodaspaces-project-view.js';
import type {ProjectOptions, ProjectNetwork} from '../tailnet/soda-tailnet-response.js';
import {object, check, id, SodaRequestError, readSodaJSON} from './sodaspaces-api.js';
import {osObservation} from './sodaspaces-project-response.js';
import {profileKeysResponse, keyPreviewResponse} from './sodaspaces-keys-response.js';
import type {KeyPreview, SavedKey, ProfileKeys} from './sodaspaces-keys-response.js';
import type {OSObservation, CreationProfile, Environment, Detail} from './sodaspaces-project-response.js';
import {renderJourney} from './sodaspaces-project-journey-view.js';
import type {JourneyViewInput} from './sodaspaces-project-journey-view.js';
import {
  renderAccessView,
  renderEnvironmentView,
  renderNetworkView,
  renderRepositoryContext,
  renderViewTabs,
} from './sodaspaces-project-settings-view.js';
import type {Lifecycle, SettingsViewInput, View} from './sodaspaces-project-settings-view.js';
import {views} from './sodaspaces-project-settings-view.js';
import {refresh as runRefresh} from './sodaspaces-project-refresh.js';
import type {RefreshInput} from './sodaspaces-project-refresh.js';
import {copyConnection} from './sodaspaces-project-connection.js';
import type {ConnectionInput} from './sodaspaces-project-connection.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';
export interface ProjectContext {
  expectedUserId: string;
  actorLogin?: string;
  repositoryId: string;
  page?: boolean;
  settings?: boolean;
  transport: PreparedExtensionMount;
  forgejoPrefix?: string;
}
function viewFromTabKey(key: string, current: View): View | undefined {
  const i = views.indexOf(current);
  if (key === 'Home') return views[0];
  if (key === 'End') return views[views.length - 1];
  if (key === 'ArrowRight') return views[(i + 1) % views.length];
  if (key === 'ArrowLeft') return views[(i - 1 + views.length) % views.length];
}

function sodaFetchInit(
  method: string,
  headers: Record<string, string>,
  body?: Record<string, unknown>,
  signal?: AbortSignal
): RequestInit {
  return {
    method,
    headers,
    ...(body === undefined
      ? {}
      : {
          body: JSON.stringify(body),
        }),
    ...(signal
      ? {
          signal,
        }
      : {}),
  };
}

async function sodaErrorCode(response: Response): Promise<string | undefined> {
  try {
    const error = object(object(await readSodaJSON(response)).error);
    if (typeof error.code === 'string') return error.code;
  } catch {
    /* Never display a response body. */
  }
}

export class SodaProjectControls extends LitElement {
  static properties = {
    presentation: {state: true},
    networkReview: {state: true},
    network: {state: true},
    networkOptions: {state: true},
    networkEnabled: {state: true},
    networkConfirmed: {state: true},
    networkNotice: {state: true},
    observedOS: {
      state: true,
    },
    osStatus: {
      state: true,
    },
    profiles: {
      state: true,
    },
    selectedProfile: {
      state: true,
    },
    status: {
      state: true,
    },
    outcome: {
      state: true,
    },
    repository: {
      state: true,
    },
    repositoryName: {
      state: true,
    },
    repositoryURL: {
      state: true,
    },
    environment: {
      state: true,
    },
    detail: {
      state: true,
    },
    saved: {
      state: true,
    },
    keyPreview: {
      state: true,
    },
    profileKeys: {
      state: true,
    },
    lifecycle: {
      state: true,
    },
    connection: {
      state: true,
    },
    busy: {
      state: true,
    },
    stale: {
      state: true,
    },
    canCreate: {
      state: true,
    },
    selected: {
      state: true,
    },
    draft: {
      state: true,
    },
    stopConfirmed: {
      state: true,
    },
    emptyConfirmed: {
      state: true,
    },
    useSavedKeys: {
      state: true,
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
  declare private environment: Environment | undefined;
  declare private detail: Detail | undefined;
  declare private saved: SavedKey[] | undefined;
  declare private profileKeys: ProfileKeys | undefined;
  declare private keyPreview: KeyPreview | undefined;
  declare private lifecycle: Lifecycle | undefined;
  declare private connection:
    | {
        command: string;
        fingerprint: string;
      }
    | undefined;
  declare private busy: boolean;
  declare private stale: boolean;
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
  private joinFailed = false;
  private joinNeedsCheck = false;
  get canRestore() {
    return !this.mutationPending;
  }
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
    this.environment =
      this.detail =
      this.saved =
      this.keyPreview =
      this.lifecycle =
      this.connection =
      this.profileKeys =
        undefined;
    this.busy = this.stale = this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
    this.selected = 'environment';
    this.useSavedKeys = false;
  }
  protected createRenderRoot() {
    return this;
  }
  configure(context: ProjectContext) {
    if (this.binding || this.disposed) throw Error('Project binding is immutable');
    this.binding = {...context};
    check(id(context.expectedUserId));
    window.addEventListener('pagehide', () => this.invalidate(), {
      signal: this.lifetime.signal,
    });
    window.addEventListener(
      'pageshow',
      (e) => {
        if (e.persisted) this.invalidate();
      },
      {
        signal: this.lifetime.signal,
      }
    );
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  private get blocked() {
    return this.busy || this.stale || this.disposed;
  }
  private get running() {
    return (
      this.detail?.environment.provisioned === true &&
      !this.detail.native_unavailable &&
      this.detail.observed?.running === true
    );
  }
  private get busyAttr(): 'true' | 'false' {
    return this.busy ? 'true' : 'false';
  }
  private get canJoin() {
    return (
      !!this.detail?.environment.provisioned &&
      this.detail.execution_allowed &&
      !this.detail.login &&
      this.running &&
      (!this.useSavedKeys || !!this.saved)
    );
  }
  private active(epoch: number) {
    return !this.disposed && !this.stale && this.epoch === epoch;
  }
  private select(view: View) {
    this.selected = view;
  }
  private async tabKey(event: KeyboardEvent, view: View) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const target = viewFromTabKey(event.key, view);
    if (!target) return;
    this.select(target);
    const epoch = this.epoch;
    await this.updateComplete;
    if (this.shouldFocusTab(epoch, target)) this.querySelector<HTMLElement>('[data-view="' + target + '"]')?.focus();
  }
  private shouldFocusTab(epoch: number, target: View) {
    return this.active(epoch) && !this.closest('[hidden], [inert]') && this.selected === target;
  }
  // Busy is checked synchronously, not only via Lit's eventually updated disabled attribute.
  private command(event: Event, action: () => void | Promise<void>) {
    const button = event.currentTarget;
    if (
      !(button instanceof HTMLButtonElement) ||
      button.disabled ||
      button.closest('[hidden], [inert]') ||
      this.blocked
    )
      return;
    void action();
  }
  setPresentation(presentation: 'standard' | 'journey' | 'settings') {
    if (this.presentation === presentation) return false;
    this.presentation = presentation;
    this.selected = 'environment';
    return true;
  }
  private createProject() {
    if (!this.canCreate || this.networkReview || !this.profiles.some((p) => p.id === this.selectedProfile)) return;
    return mutate(
      this.mutationsInput(),
      '/api/environments',
      {
        repository_id: this.binding?.repositoryId,
        profile_id: this.selectedProfile,
        tailnet: this.createTailnetBody(),
      },
      'Project created. Join explicitly to set up your browser-terminal account.'
    );
  }
  private createTailnetBody() {
    if (this.networkEnabled && this.networkOptions?.available)
      return {enabled: true, revision: this.networkOptions.revision, binding: this.networkOptions.binding};
    return {enabled: false};
  }
  private journeyViewInput(): JourneyViewInput {
    return {
      isStale: () => this.stale,
      isBusy: () => this.busy,
      isBlocked: () => this.blocked,
      readBusyAttr: () => this.busyAttr,
      readEnvironment: () => this.environment,
      readDetail: () => this.detail,
      canCreate: () => this.canCreate,
      isRunning: () => this.running,
      hasJoinFailed: () => this.joinFailed,
      needsJoinCheck: () => this.joinNeedsCheck,
      isMutationPending: () => this.mutationPending,
      needsOutcomeAttention: () => this.outcomeNeedsAttention,
      readOutcome: () => this.outcome,
      readStatus: () => this.status,
      readRepositoryName: () => this.repositoryName,
      readBindingRepository: () => this.binding?.repositoryId || '',
      hasLifecycle: () => !!this.lifecycle,
      readProfiles: () => this.profiles,
      readSelectedProfile: () => this.selectedProfile,
      readNetworkOptions: () => this.networkOptions,
      isNetworkEnabled: () => this.networkEnabled,
      needsNetworkReview: () => this.networkReview,
      requestRefresh: (event) => this.command(event, () => this.refresh()),
      refreshNow: () => this.refresh(),
      requestJoin: (event) => this.command(event, () => this.joinEnvironment()),
      requestStart: (event) => this.command(event, () => this.changeLifecycle(false)),
      requestCreate: (event) => this.command(event, () => this.createProject()),
      requestRepositoryChange: (event) => this.command(event, () => this.requestRepositoryChange()),
      clearNetworkReview: (event) => this.command(event, () => this.onClearNetworkReview()),
      selectProfile: (value) => this.onSelectProfile(value),
      setJourneyNetworkEnabled: (enabled) => this.onJourneyNetworkEnabled(enabled),
    };
  }
  private requestRepositoryChange() {
    this.dispatchEvent(new CustomEvent('soda-project-change-repository', {bubbles: true}));
  }
  private onSelectProfile(value: string) {
    if (this.profiles.some((p) => p.id === value)) this.selectedProfile = value;
  }
  private onJourneyNetworkEnabled(enabled: boolean) {
    this.networkEnabled = enabled;
    this.networkReview = false;
  }
  private onCreateNetworkEnabled(enabled: boolean) {
    this.networkEnabled = enabled;
  }
  private onClearNetworkReview() {
    this.networkEnabled = this.networkReview = false;
  }
  private shouldRenderJourney() {
    return this.presentation === 'journey';
  }
  private sectionAriaBusy() {
    return this.busy && !this.stale ? 'true' : 'false';
  }
  private sessionCaption() {
    if (!this.binding) return '';
    return this.binding.actorLogin
      ? `Forgejo account: ${this.binding.actorLogin} (ID ${this.binding.expectedUserId})`
      : `Forgejo account ID ${this.binding.expectedUserId}`;
  }
  private projectAccountCaption() {
    if (!this.connection) return '';
    return 'Project account: ' + this.detail?.login;
  }
  private boundRepositoryId() {
    return this.binding?.repositoryId || '';
  }
  private boundEnvironmentId() {
    return this.environment?.id || '';
  }
  protected render() {
    if (this.shouldRenderJourney()) return html`${renderJourney(this.journeyViewInput())}${this.renderIdentity()}`;
    return html`<section
      data-project-controls
      data-repository-id=${this.boundRepositoryId()}
      data-environment-id=${this.boundEnvironmentId()}
      class="soda-spaces-controls"
      aria-busy=${this.sectionAriaBusy()}
    >
      ${renderRepositoryContext(this.settingsViewInput())} ${renderViewTabs(this.settingsViewInput())}
      ${renderEnvironmentView(this.settingsViewInput())} ${renderAccessView(this.settingsViewInput())}
      ${renderNetworkView(this.settingsViewInput())}
      ${renderProjectStatus(this.repository, this.sessionCaption(), this.projectAccountCaption(), this.status, this.outcome)}
      ${this.renderIdentity()}
    </section>`;
  }
  private journeyIdentityHidden() {
    // Identity connections need an established project login. During the
    // first-use journey (configure, join, uncertain outcomes) the section
    // would only double the journey's own recovery controls.
    return this.shouldRenderJourney() && !this.detail?.login;
  }
  private renderIdentity() {
    if (!this.identityPresentationReady() || !this.binding || this.journeyIdentityHidden()) return html``;
    const actor = this.binding?.expectedUserId;
    if (!actor) return html``;
    return html`<soda-identity
      .context=${{actor, transport: this.binding.transport, project: this.environment?.id || ''}}
    ></soda-identity>`;
  }
  private identityPresentationReady() {
    return !this.stale && !this.disposed && this.presentation !== 'standard';
  }
  private settingsViewInput(): SettingsViewInput {
    return {
      readBindingSettings: () => !!this.binding?.settings,
      readBindingPage: () => !!this.binding?.page,
      readBindingRepository: () => this.binding?.repositoryId || '',
      readRepository: () => this.repository,
      readRepositoryURL: () => this.repositoryURL,
      readSelected: () => this.selected,
      readPresentation: () => this.presentation,
      canCreate: () => this.canCreate,
      isBusy: () => this.busy,
      isStale: () => this.stale,
      isBlocked: () => this.blocked,
      isRunning: () => this.running,
      readEnvironment: () => this.environment,
      readDetail: () => this.detail,
      readDetailEnvironment: () => this.detail?.environment || this.environment,
      readDetailLogin: () => this.detail?.login,
      readExecutionAllowed: () => this.detail?.execution_allowed === true,
      readEnvironmentAdmin: () => !!this.detail?.environment_administrator,
      readSavedKeys: () => this.saved,
      readProfileKeys: () => this.profileKeys,
      readKeyPreview: () => this.keyPreview,
      readUseSavedKeys: () => this.useSavedKeys,
      readDraft: () => this.draft,
      readEmptyConfirmed: () => this.emptyConfirmed,
      readStopConfirmed: () => this.stopConfirmed,
      readLifecycle: () => this.lifecycle,
      readConnection: () => this.connection,
      readProfiles: () => this.profiles,
      readSelectedProfile: () => this.selectedProfile,
      readNetwork: () => this.network,
      readNetworkOptions: () => this.networkOptions,
      isNetworkEnabled: () => this.networkEnabled,
      readNetworkNotice: () => this.networkNotice,
      readNetworkConfirmed: () => this.networkConfirmed,
      readObservedOS: () => this.observedOS,
      readOsStatus: () => this.osStatus,
      selectView: (view) => this.select(view),
      tabKey: (event, view) => this.tabKey(event, view),
      setCreateNetworkEnabled: (enabled) => this.onCreateNetworkEnabled(enabled),
      selectProfile: (value) => this.onSelectProfile(value),
      setUseSavedKeys: (checked) => this.onUseSavedKeys(checked),
      requestRefresh: (event) => this.command(event, () => this.refresh()),
      requestCreate: (event) => this.command(event, () => this.createProject()),
      requestJoin: (event) => this.command(event, () => this.joinEnvironment()),
      requestStart: (event) => this.command(event, () => this.changeLifecycle(false)),
      requestStop: (event) => this.command(event, () => this.changeLifecycle(true)),
      setStopConfirmed: (checked) => this.onStopConfirmed(checked),
      requestInspectOS: (event) => this.command(event, () => this.inspectOS()),
      reloadPage: (event) => this.onReloadPage(event),
      copyConnection: () => {
        void copyConnection(this.connectionInput());
      },
      removeSavedKey: (event, key) => this.removeSavedKey(event, key),
      setDraft: (value) => this.onDraft(value),
      requestSaveKey: (event) => this.command(event, () => this.saveKey()),
      requestReviewKeys: (event) => this.command(event, () => this.reviewKeys()),
      setConfirmEmpty: (checked) => this.onConfirmEmpty(checked),
      requestApplyKeys: (event) => this.command(event, () => this.applyKeys()),
      reviewProfileKeysPage: (page) => {
        void this.reviewProfileKeys(page);
      },
      selectForgejoKey: (key) => this.selectForgejoKey(key),
      setNetworkConfirmed: (confirmed) => this.onNetworkConfirmed(confirmed),
      changeProjectNetwork: (event, action) => this.command(event, () => this.changeNetwork(action)),
    };
  }
  private onUseSavedKeys(checked: boolean) {
    this.useSavedKeys = checked;
  }
  private onReloadPage(event: Event) {
    if (this.reloadAdmitted(event)) window.location.reload();
  }
  private reloadAdmitted(event: Event) {
    return (
      !this.disposed && event.currentTarget instanceof HTMLElement && !event.currentTarget.closest('[hidden], [inert]')
    );
  }
  private onStopConfirmed(checked: boolean) {
    this.stopConfirmed = checked;
  }
  private onDraft(value: string) {
    this.draft = value;
  }
  private onConfirmEmpty(checked: boolean) {
    this.emptyConfirmed = checked;
  }
  private onNetworkConfirmed(confirmed: boolean) {
    this.networkConfirmed = confirmed;
  }
  private joinEnvironment() {
    if (
      !this.environment ||
      !this.running ||
      !this.detail?.execution_allowed ||
      this.detail.login ||
      this.detail.authority_unavailable
    )
      return;
    return mutate(
      this.mutationsInput(),
      '/api/environments/' + this.environment.id + '/join',
      {
        ssh_keys: this.presentation !== 'journey' && this.useSavedKeys ? 'saved' : 'none',
      },
      'Native join confirmed. Your browser terminal uses this account, not SSH. Later SSH-key changes require a separate explicit Apply.'
    );
  }
  private removeSavedKey(event: Event, key: SavedKey) {
    this.command(event, () =>
      mutate(
        this.mutationsInput(),
        '/api/me/development-keys/' + key.id,
        {},
        'Saved key removed. Existing project SSH access is unchanged until explicitly applied.',
        'DELETE'
      )
    );
  }
  private selectForgejoKey(key: string) {
    if (this.blocked || this.selected !== 'access' || this.closest('[hidden], [inert]')) return;
    this.draft = key;
    this.outcome =
      'Review the selected public key above, then explicitly Save public key. Joining/applying to a project remains a separate action.';
  }
  private connectionInput(): ConnectionInput {
    return {
      hasPage: () => !!this.binding?.page,
      isBlocked: () => this.blocked,
      readConnection: () => this.connection,
      isAccessSelected: () => this.selected === 'access',
      isConcealed: () => !!this.closest('[hidden], [inert]'),
      isDisposed: () => this.disposed,
      setOutcome: (outcome) => {
        this.outcome = outcome;
      },
    };
  }
  private reset() {
    this.network = this.networkOptions = undefined;
    this.networkEnabled = this.networkConfirmed = false;
    this.networkNotice = '';
    this.repositoryURL = '';
    this.environment =
      this.detail =
      this.keyPreview =
      this.saved =
      this.lifecycle =
      this.connection =
      this.profileKeys =
        undefined;
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
    this.repository = this.repositoryName = '';
    this.status =
      'Page context changed. Reload the full repository page; no action was replayed or undone. A dispatched operation may still have completed.';
  }
  private requestHeaders(method: string): Record<string, string> {
    const headers: Record<string, string> = {};
    if (method === 'GET') return headers;
    headers['Content-Type'] = 'application/json';
    return headers;
  }
  private admitHttpFailure(status: number) {
    if (status !== 401 && status !== 403) return;
    this.invalidate();
  }
  private async api(
    path: string,
    method = 'GET',
    body?: Record<string, unknown>,
    signal?: AbortSignal
  ): Promise<unknown> {
    if (!this.binding) throw Error('Missing native extension');
    const response = await this.binding.transport.request(
      path.slice('/api/'.length),
      sodaFetchInit(method, this.requestHeaders(method), body, signal)
    );
    if (response.ok) return response.status === 204 ? null : readSodaJSON(response);
    this.admitHttpFailure(response.status);
    throw new SodaRequestError(response.status, await sodaErrorCode(response));
  }
  async refresh() {
    await runRefresh(this.refreshInput());
  }
  private beginEpoch() {
    return ++this.epoch;
  }
  private beginRead() {
    this.readController?.abort();
    this.reset();
    const control = (this.readController = new AbortController());
    return control;
  }
  private announceObserved(summary: {
    repositoryId: string;
    environmentId: string;
    provisioned: boolean;
    login: string;
    running: boolean;
  }) {
    this.dispatchEvent(
      new CustomEvent('soda-project-observed', {
        bubbles: true,
        detail: {
          repositoryId: summary.repositoryId,
          environmentId: summary.environmentId,
          provisioned: summary.provisioned,
          login: summary.login,
          running: summary.running,
        },
      })
    );
  }
  private announceChanged(repositoryId: string) {
    this.dispatchEvent(new CustomEvent('soda-project-changed', {bubbles: true, detail: {repositoryId}}));
  }
  private refreshInput(): RefreshInput {
    return {
      isBusy: () => this.busy,
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      hasBinding: () => !!this.binding,
      readBinding: () => this.binding,
      isActive: (n) => this.active(n),
      beginEpoch: () => this.beginEpoch(),
      beginRead: () => this.beginRead(),
      api: (path, method, body, signal) => this.api(path, method, body, signal),
      readPresentation: () => this.presentation,
      readEnvironment: () => this.environment,
      readDetail: () => this.detail,
      readStatus: () => this.status,
      readSelectedProfile: () => this.selectedProfile,
      readNetworkOptions: () => this.networkOptions,
      isNetworkEnabled: () => this.networkEnabled,
      readProfiles: () => this.profiles,
      isRunning: () => this.running,
      readJoinFailed: () => this.joinFailed,
      updated: () => this.updateComplete,
      announceObserved: (summary) => this.announceObserved(summary),
      announceChanged: (repositoryId) => this.announceChanged(repositoryId),
      setBusy: (busy) => {
        this.busy = busy;
      },
      setStatus: (status) => {
        this.status = status;
      },
      setRepositoryName: (name) => {
        this.repositoryName = name;
      },
      setRepository: (repository) => {
        this.repository = repository;
      },
      setRepositoryURL: (url) => {
        this.repositoryURL = url;
      },
      setProfiles: (profiles) => {
        this.profiles = profiles;
      },
      setSelectedProfile: (profile) => {
        this.selectedProfile = profile;
      },
      setCanCreate: (canCreate) => {
        this.canCreate = canCreate;
      },
      setNetworkOptions: (options) => {
        this.networkOptions = options;
      },
      setNetworkReview: (review) => {
        this.networkReview = review;
      },
      setNetworkEnabled: (enabled) => {
        this.networkEnabled = enabled;
      },
      setEnvironment: (environment) => {
        this.environment = environment;
      },
      setDetail: (detail) => {
        this.detail = detail;
      },
      setJoinNeedsCheck: (needed) => {
        this.joinNeedsCheck = needed;
      },
      setJoinFailed: (failed) => {
        this.joinFailed = failed;
      },
      setOutcomeNeedsAttention: (attention) => {
        this.outcomeNeedsAttention = attention;
      },
      setOutcome: (outcome) => {
        this.outcome = outcome;
      },
      setSaved: (saved) => {
        this.saved = saved;
      },
      setLifecycle: (lifecycle) => {
        this.lifecycle = lifecycle;
      },
      setConnection: (connection) => {
        this.connection = connection;
      },
      setNetwork: (network) => {
        this.network = network;
      },
      setNetworkNotice: (notice) => {
        this.networkNotice = notice;
      },
      resetState: () => this.reset(),
    };
  }
  private mutationsInput(): MutationsInput {
    return {
      isBlocked: () => this.blocked,
      hasBinding: () => !!this.binding,
      readEpoch: () => this.epoch,
      isActive: (n) => this.active(n),
      readBindingRepository: () => this.binding?.repositoryId,
      readEnvironmentId: () => this.environment?.id,
      readDetailLogin: () => this.detail?.login,
      api: (path, method, body, signal) => this.api(path, method, body, signal),
      refreshAfter: () => this.refresh(),
      announceOperation: () => this.dispatchEvent(new CustomEvent('soda-project-operation', {bubbles: true})),
      announceChanged: (repositoryId) =>
        this.dispatchEvent(new CustomEvent('soda-project-changed', {bubbles: true, detail: {repositoryId}})),
      setBusy: (busy) => {
        this.busy = busy;
      },
      setOutcomeNeedsAttention: (attention) => {
        this.outcomeNeedsAttention = attention;
      },
      setJoinFailed: (failed) => {
        this.joinFailed = failed;
      },
      setOutcome: (outcome) => {
        this.outcome = outcome;
      },
      setMutationPending: (pending) => {
        this.mutationPending = pending;
      },
      setJoinNeedsCheck: (needed) => {
        this.joinNeedsCheck = needed;
      },
      setCanCreate: (canCreate) => {
        this.canCreate = canCreate;
      },
    };
  }
  private networkChangeBlocked() {
    return (
      !this.network ||
      !this.environment ||
      !this.detail?.environment_administrator ||
      !this.networkConfirmed ||
      this.blocked
    );
  }
  private networkEnableBlocked(network: ProjectNetwork) {
    return !network.available_binding || (!!network.binding && network.binding !== network.available_binding);
  }
  private changeNetwork(action: 'enable' | 'disable' | 'retry') {
    const network = this.network;
    if (this.networkChangeBlocked()) return;
    if (action !== 'disable' && this.networkEnableBlocked(network!)) return;
    this.networkConfirmed = false;
    return mutate(
      this.mutationsInput(),
      '/api/environments/' + this.environment!.id + '/tailnet',
      {
        action,
        revision: network!.revision,
        confirm_id: network!.project,
        ...(action === 'disable' ? {} : {binding: network!.available_binding}),
      },
      'Network policy saved; observe the native outcome.'
    );
  }
  private applyOSError(epoch: number, error: unknown) {
    if (!this.active(epoch)) return;
    if (this.authLost(error)) this.invalidate();
    else this.osStatus = 'OS observation unavailable. Nothing was started or repaired.';
  }
  private authLost(error: unknown) {
    return error instanceof SodaRequestError && (error.status === 401 || error.status === 403);
  }
  private async inspectOS() {
    if (this.blocked || !this.environment) return;
    const epoch = this.epoch,
      target = this.environment.id;
    this.busy = true;
    this.observedOS = undefined;
    this.osStatus = 'Reading current userspace…';
    this.readController?.abort();
    const controller = (this.readController = new AbortController()),
      timeout = window.setTimeout(() => controller.abort(), 15000);
    try {
      const observation = osObservation(
        await this.api('/api/environments/' + target + '/os', 'GET', undefined, controller.signal),
        target
      );
      if (this.active(epoch)) {
        this.observedOS = observation;
        this.osStatus = 'Read completed. Creation metadata was not changed.';
      }
    } catch (error) {
      this.applyOSError(epoch, error);
    } finally {
      window.clearTimeout(timeout);
      if (this.active(epoch)) this.busy = false;
    }
  }
  private changeLifecycle(stop: boolean) {
    if (!this.environment) return;
    if (stop && !this.stopConfirmed) {
      this.outcome = 'Confirm the shared impact before Stop.';
      return;
    }
    return mutate(
      this.mutationsInput(),
      `/api/environments/${this.environment.id}/lifecycle`,
      {
        action: stop ? 'stop' : 'start',
        ...(stop
          ? {
              confirm_stop: true,
            }
          : {}),
      },
      stop
        ? 'Stop confirmed; next-boot start disabled. Existing data was not recreated or deleted.'
        : 'Start confirmed and next-boot start enabled. Refresh connection status while services initialize.'
    );
  }
  private saveKey() {
    const value = this.draft.trim();
    if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) {
      this.outcome = 'Provide one public SSH key. Never upload a private key.';
      return;
    }
    this.draft = '';
    return mutate(
      this.mutationsInput(),
      '/api/me/development-keys',
      {
        public_key: value,
      },
      'Public key saved for future joins. Existing project access is unchanged until explicitly applied.'
    );
  }
  private profileKeysPageAdmitted(page: number) {
    return (
      !this.blocked &&
      this.selected === 'access' &&
      !this.closest('[hidden], [inert]') &&
      Number.isInteger(page) &&
      page >= 1 &&
      page <= 8
    );
  }
  private async reviewProfileKeys(page: number) {
    if (!this.profileKeysPageAdmitted(page)) return;
    const epoch = this.epoch;
    this.busy = true;
    const controller = new AbortController(),
      timeout = window.setTimeout(() => controller.abort(), 20000);
    try {
      const keys = profileKeysResponse(
        await this.api('/api/me/forgejo-keys?page=' + page, 'GET', undefined, controller.signal),
        page
      );
      if (this.active(epoch)) this.profileKeys = keys;
    } catch {
      this.profileKeysFailed(epoch);
    } finally {
      window.clearTimeout(timeout);
      if (this.active(epoch)) this.busy = false;
    }
  }
  private profileKeysFailed(epoch: number) {
    if (!this.active(epoch)) return;
    this.profileKeys = undefined;
    this.outcome =
      'Own Forgejo keys are unavailable. Nothing was imported; use native profile settings or explicitly paste a public key.';
  }
  private async reviewKeys() {
    if (this.blocked || !this.environment || !this.detail) return;
    const n = this.epoch;
    this.busy = true;
    const controller = new AbortController(),
      timeout = window.setTimeout(() => controller.abort(), 15000);
    try {
      const preview = keyPreviewResponse(
        await this.api(`/api/environments/${this.environment.id}/access-keys`, 'GET', undefined, controller.signal),
        this.detail.login
      );
      if (!this.active(n)) return;
      this.keyPreview = preview;
      this.emptyConfirmed = false;
    } catch {
      if (this.active(n)) {
        this.keyPreview = undefined;
        this.outcome = 'Key preview unavailable or changed. No update was requested; refresh and inspect.';
      }
    } finally {
      window.clearTimeout(timeout);
      if (this.active(n)) this.busy = false;
    }
  }
  private applyKeys() {
    if (!this.keyPreview || !this.environment || !this.detail?.execution_allowed) return;
    if (!this.keyPreview.saved_fingerprints.length && !this.emptyConfirmed) {
      this.outcome = 'Explicitly confirm removal of the last managed key.';
      return;
    }
    const preview = this.keyPreview;
    this.keyPreview = undefined;
    this.emptyConfirmed = false;
    return mutate(
      this.mutationsInput(),
      `/api/environments/${this.environment.id}/access-keys`,
      {
        revision: preview.revision,
        saved_fingerprints: preview.saved_fingerprints,
        confirm_empty: !preview.saved_fingerprints.length,
      },
      'Managed SSH key file updated for this project. Verify new-key login and old-key refusal from your SSH client. Existing sessions and browser access are not revoked.'
    );
  }
  dispose() {
    if (this.disposed) return;
    this.invalidate();
    this.disposed = true;
    this.lifetime.abort();
    this.remove();
  }
}
customElements.define('soda-project-controls', SodaProjectControls);
export function mountProjectControls(root: HTMLElement, context: ProjectContext) {
  if (
    (context.expectedUserId !== undefined && !id(context.expectedUserId)) ||
    !id(context.repositoryId) ||
    root.ownerDocument !== document
  )
    throw Error('Invalid native page context');
  const box = new SodaProjectControls();
  box.configure(context);
  root.append(box);
  return {
    get canRestore() {
      return box.canRestore;
    },
    refresh: () => box.refresh(),
    invalidate: () => box.invalidate(),
    setPresentation: (mode: 'standard' | 'journey' | 'settings') => box.setPresentation(mode),
    get ready() {
      return box.updateComplete;
    },
    dispose: () => box.dispose(),
  };
}

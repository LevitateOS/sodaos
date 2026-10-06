import './soda-identity.js';
import type {PreparedExtensionMount} from './soda-extension.js';
// Soda owns only this mount. Native forms, authentication and terminal lifetime
// are not rendering concerns; commands remain explicit and generation guarded.
import {LitElement, html} from 'lit';
import {renderProjectStatus} from './sodaspaces-project-view.js';
import type {ProjectOptions, ProjectNetwork} from '../tailnet/soda-tailnet-response.js';
import {check, id} from './sodaspaces-api.js';
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
import type {MutationsInput} from './sodaspaces-project-mutations.js';
import {
  announceChanged,
  announceObserved,
  api,
  beginEpoch,
  beginRead,
  createProject,
} from './sodaspaces-project-request.js';
import type {RequestInput} from './sodaspaces-project-request.js';
import {
  changeNetwork,
  clearNetworkReview,
  setCreateNetworkEnabled,
  setJourneyNetworkEnabled,
  setNetworkConfirmed,
} from './sodaspaces-project-network.js';
import type {NetworkInput} from './sodaspaces-project-network.js';
import {
  applyKeys,
  joinEnvironment,
  removeSavedKey,
  reviewKeys,
  reviewProfileKeys,
  saveKey,
  selectForgejoKey,
  setConfirmEmpty,
  setDraft,
  setUseSavedKeys,
} from './sodaspaces-project-access.js';
import type {AccessInput} from './sodaspaces-project-access.js';
import {changeLifecycle, inspectOS, setStopConfirmed} from './sodaspaces-project-runtime.js';
import type {RuntimeInput} from './sodaspaces-project-runtime.js';
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
      requestJoin: (event) => this.command(event, () => joinEnvironment(this.accessInput())),
      requestStart: (event) => this.command(event, () => changeLifecycle(this.runtimeInput(), false)),
      requestCreate: (event) => this.command(event, () => createProject(this.requestInput())),
      requestRepositoryChange: (event) => this.command(event, () => this.requestRepositoryChange()),
      clearNetworkReview: (event) => this.command(event, () => clearNetworkReview(this.networkInput())),
      selectProfile: (value) => this.onSelectProfile(value),
      setJourneyNetworkEnabled: (enabled) => setJourneyNetworkEnabled(this.networkInput(), enabled),
    };
  }
  private requestRepositoryChange() {
    this.dispatchEvent(new CustomEvent('soda-project-change-repository', {bubbles: true}));
  }
  private onSelectProfile(value: string) {
    if (this.profiles.some((p) => p.id === value)) this.selectedProfile = value;
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
      setCreateNetworkEnabled: (enabled) => setCreateNetworkEnabled(this.networkInput(), enabled),
      selectProfile: (value) => this.onSelectProfile(value),
      setUseSavedKeys: (checked) => setUseSavedKeys(this.accessInput(), checked),
      requestRefresh: (event) => this.command(event, () => this.refresh()),
      requestCreate: (event) => this.command(event, () => createProject(this.requestInput())),
      requestJoin: (event) => this.command(event, () => joinEnvironment(this.accessInput())),
      requestStart: (event) => this.command(event, () => changeLifecycle(this.runtimeInput(), false)),
      requestStop: (event) => this.command(event, () => changeLifecycle(this.runtimeInput(), true)),
      setStopConfirmed: (checked) => setStopConfirmed(this.runtimeInput(), checked),
      requestInspectOS: (event) => this.command(event, () => inspectOS(this.runtimeInput())),
      reloadPage: (event) => this.onReloadPage(event),
      copyConnection: () => {
        void copyConnection(this.connectionInput());
      },
      removeSavedKey: (event, key) => removeSavedKey(this.accessInput(), event, key),
      setDraft: (value) => setDraft(this.accessInput(), value),
      requestSaveKey: (event) => this.command(event, () => saveKey(this.accessInput())),
      requestReviewKeys: (event) => this.command(event, () => reviewKeys(this.accessInput())),
      setConfirmEmpty: (checked) => setConfirmEmpty(this.accessInput(), checked),
      requestApplyKeys: (event) => this.command(event, () => applyKeys(this.accessInput())),
      reviewProfileKeysPage: (page) => {
        void reviewProfileKeys(this.accessInput(), page);
      },
      selectForgejoKey: (key) => selectForgejoKey(this.accessInput(), key),
      setNetworkConfirmed: (confirmed) => setNetworkConfirmed(this.networkInput(), confirmed),
      changeProjectNetwork: (event, action) => this.command(event, () => changeNetwork(this.networkInput(), action)),
    };
  }
  private onReloadPage(event: Event) {
    if (this.reloadAdmitted(event)) window.location.reload();
  }
  private reloadAdmitted(event: Event) {
    return (
      !this.disposed && event.currentTarget instanceof HTMLElement && !event.currentTarget.closest('[hidden], [inert]')
    );
  }
  private runtimeInput(): RuntimeInput {
    return {
      readEnvironment: () => this.environment,
      isBlocked: () => this.blocked,
      readEpoch: () => this.epoch,
      isActive: (generation) => this.active(generation),
      setBusy: (busy) => {
        this.busy = busy;
      },
      setOutcome: (outcome) => {
        this.outcome = outcome;
      },
      setObservedOS: (observation) => {
        this.observedOS = observation;
      },
      setOsStatus: (status) => {
        this.osStatus = status;
      },
      readStopConfirmed: () => this.stopConfirmed,
      setStopConfirmed: (confirmed) => {
        this.stopConfirmed = confirmed;
      },
      abortRead: () => {
        this.readController?.abort();
      },
      takeReadController: () => (this.readController = new AbortController()),
      invalidate: () => this.invalidate(),
      api: (path, method, body, signal) => api(this.requestInput(), path, method, body, signal),
      mutations: this.mutationsInput(),
    };
  }
  private accessInput(): AccessInput {
    return {
      readEnvironment: () => this.environment,
      readDetail: () => this.detail,
      isRunning: () => this.running,
      readPresentation: () => this.presentation,
      readUseSavedKeys: () => this.useSavedKeys,
      setUseSavedKeys: (checked) => {
        this.useSavedKeys = checked;
      },
      readDraft: () => this.draft,
      setDraft: (value) => {
        this.draft = value;
      },
      readEmptyConfirmed: () => this.emptyConfirmed,
      setEmptyConfirmed: (confirmed) => {
        this.emptyConfirmed = confirmed;
      },
      readKeyPreview: () => this.keyPreview,
      setKeyPreview: (preview) => {
        this.keyPreview = preview;
      },
      setProfileKeys: (keys) => {
        this.profileKeys = keys;
      },
      isBlocked: () => this.blocked,
      isAccessSelected: () => this.selected === 'access',
      isConcealed: () => !!this.closest('[hidden], [inert]'),
      readEpoch: () => this.epoch,
      isActive: (generation) => this.active(generation),
      setBusy: (busy) => {
        this.busy = busy;
      },
      setOutcome: (outcome) => {
        this.outcome = outcome;
      },
      runCommand: (event, action) => this.command(event, action),
      api: (path, method, body, signal) => api(this.requestInput(), path, method, body, signal),
      mutations: this.mutationsInput(),
    };
  }
  private networkInput(): NetworkInput {
    return {
      readNetwork: () => this.network,
      readEnvironment: () => this.environment,
      readDetail: () => this.detail,
      isBlocked: () => this.blocked,
      readNetworkConfirmed: () => this.networkConfirmed,
      setNetworkConfirmed: (confirmed) => {
        this.networkConfirmed = confirmed;
      },
      setNetworkEnabled: (enabled) => {
        this.networkEnabled = enabled;
      },
      setNetworkReview: (review) => {
        this.networkReview = review;
      },
      mutations: this.mutationsInput(),
    };
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
  async refresh() {
    await runRefresh(this.refreshInput());
  }
  private requestInput(): RequestInput {
    return {
      readBinding: () => this.binding,
      beginEpoch: () => ++this.epoch,
      abortRead: () => {
        this.readController?.abort();
      },
      takeReadController: () => (this.readController = new AbortController()),
      resetState: () => this.reset(),
      invalidate: () => this.invalidate(),
      dispatchObserved: (detail) => {
        this.dispatchEvent(
          new CustomEvent('soda-project-observed', {
            bubbles: true,
            detail,
          })
        );
      },
      dispatchChanged: (repositoryId) => {
        this.dispatchEvent(new CustomEvent('soda-project-changed', {bubbles: true, detail: {repositoryId}}));
      },
      readCanCreate: () => this.canCreate,
      readNetworkReview: () => this.networkReview,
      readProfiles: () => this.profiles,
      readSelectedProfile: () => this.selectedProfile,
      readNetworkEnabled: () => this.networkEnabled,
      readNetworkOptions: () => this.networkOptions,
      mutations: this.mutationsInput(),
    };
  }
  private refreshInput(): RefreshInput {
    return {
      isBusy: () => this.busy,
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      hasBinding: () => !!this.binding,
      readBinding: () => this.binding,
      isActive: (n) => this.active(n),
      beginEpoch: () => beginEpoch(this.requestInput()),
      beginRead: () => beginRead(this.requestInput()),
      api: (path, method, body, signal) => api(this.requestInput(), path, method, body, signal),
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
      announceObserved: (summary) => announceObserved(this.requestInput(), summary),
      announceChanged: (repositoryId) => announceChanged(this.requestInput(), repositoryId),
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
      api: (path, method, body, signal) => api(this.requestInput(), path, method, body, signal),
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

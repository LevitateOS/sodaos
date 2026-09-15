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
import {
  object,
  check,
  id,
  creationProfile,
  osObservation,
  projectId,
  fingerprint,
  sessionResponse,
  environmentResponse,
  detailResponse,
  savedKeysResponse,
  profileKeysResponse,
  keyPreviewResponse,
  SodaRequestError,
  readSodaJSON,
} from './sodaspaces-api.js';
import type {
  OSObservation,
  CreationProfile,
  Session,
  Environment,
  Detail,
  KeyPreview,
  SavedKey,
  ProfileKeys,
} from './sodaspaces-api.js';
export interface ProjectContext {
  expectedUserId?: string | undefined;
  repositoryId: string;
  page?: boolean;
  settings?: boolean;
  session?: Session | undefined;
}
const rejected = new Set([400, 401, 403, 404, 409, 413, 415, 422]);
const views = ['environment', 'access', 'network'] as const;
type View = (typeof views)[number];
type Lifecycle = {
  running: boolean;
  boot: boolean;
};
type RefreshPrior = {
  profile: string;
  network: ProjectOptions | undefined;
  enabled: boolean;
};

function viewFromTabKey(key: string, current: View): View | undefined {
  const i = views.indexOf(current);
  if (key === 'Home') return views[0];
  if (key === 'End') return views[views.length - 1];
  if (key === 'ArrowRight') return views[(i + 1) % views.length];
  if (key === 'ArrowLeft') return views[(i - 1 + views.length) % views.length];
}

function withExpectedUser(intent: URLSearchParams, expectedUserId: string | undefined): URLSearchParams {
  if (expectedUserId) intent.set('expected_user_id', expectedUserId);
  return intent;
}

function journeyLoginQuery(binding: ProjectContext | undefined): URLSearchParams {
  if (binding?.page && !binding.settings)
    return withExpectedUser(new URLSearchParams({destination: 'spaces'}), binding.expectedUserId);
  const intent = new URLSearchParams({repository_id: binding?.repositoryId || ''});
  if (binding?.settings) intent.set('destination', 'repository-spaces');
  return withExpectedUser(intent, binding?.expectedUserId);
}

function environmentLoginQuery(binding: ProjectContext | undefined): URLSearchParams {
  const intent = new URLSearchParams({repository_id: binding?.repositoryId || ''});
  if (binding?.settings) intent.set('destination', 'repository-spaces');
  return withExpectedUser(intent, binding?.expectedUserId);
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
    credentials: 'same-origin',
    cache: 'no-store',
    redirect: 'error',
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

function mutationObject(raw: unknown): Record<string, unknown> | null {
  return raw === null ? null : object(raw);
}

function admitRepositoryPart(part: unknown) {
  check(
    typeof part === 'string' &&
      part !== '' &&
      part !== '.' &&
      part !== '..' &&
      part.length <= 255 &&
      !/[\/\\\x00\r\n]/.test(part)
  );
}

function admitProjectLogin(login: string) {
  check(/^[a-z][a-z0-9_-]{0,30}$/.test(login) && login !== 'root');
}

function viewTabLabel(view: View, presentation: string): string {
  if (view === 'environment' && presentation === 'settings') return 'Overview';
  return (view[0]?.toUpperCase() || '') + view.slice(1);
}

function publicKeyToken(value: string | undefined) {
  if (!value) return;
  return value.trim().split(/\s+/).slice(0, 2).join(' ');
}

// Stateless presentation: the concrete project owner admits every command and
// retains the original target, confirmation and asynchronous request lifetime.
function renderLifecycle(
  lifecycle: Lifecycle | undefined,
  blocked: boolean,
  confirmed: boolean,
  start: (event: MouseEvent) => void,
  stop: (event: MouseEvent) => void,
  confirm: (checked: boolean) => void
): TemplateResult {
  const stopped = !lifecycle?.running && !lifecycle?.boot;
  return html`
    <fieldset ?hidden=${!lifecycle}>
      <legend>Shared environment</legend>
      <p>${lifecycleCaption(lifecycle)}</p>
      <button
        type="button"
        class="ui primary button"
        ?hidden=${!!lifecycle?.running && lifecycle.boot}
        ?disabled=${blocked}
        @click=${start}
      >
        Start
      </button>
      <button type="button" class="ui button danger" ?hidden=${stopped} ?disabled=${blocked} @click=${stop}>
        Stop
      </button>
      <label ?hidden=${stopped}>
        <input type="checkbox" .checked=${confirmed} @change=${(event: Event) => confirmLifecycle(event, confirm)} />
        I understand Stop interrupts everyone’s SSH, terminals and workloads, and disables next-boot start.
      </label>
    </fieldset>
  `;
}

function confirmLifecycle(event: Event, confirm: (checked: boolean) => void) {
  if (event.target instanceof HTMLInputElement) confirm(event.target.checked);
}

function lifecycleCaption(lifecycle: Lifecycle | undefined): string {
  if (!lifecycle) return '';
  return `Running: ${lifecycle.running ? 'yes' : 'no'}; starts on host boot: ${lifecycle.boot ? 'yes' : 'no'}. Start restores boot start; Stop disables it.`;
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
    session: {
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
    connectVisible: {
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
  declare private session: Session | undefined;
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
    this.session =
      this.environment =
      this.detail =
      this.saved =
      this.keyPreview =
      this.lifecycle =
      this.connection =
      this.profileKeys =
        undefined;
    this.busy = this.stale = this.canCreate = this.stopConfirmed = this.emptyConfirmed = false;
    this.connectVisible = true;
    this.selected = 'environment';
    this.useSavedKeys = false;
  }
  protected createRenderRoot() {
    return this;
  }
  configure(context: ProjectContext) {
    if (this.binding || this.disposed) throw Error('Project binding is immutable');
    this.binding = {...context};
    check(!context.session || context.session.user.id === context.expectedUserId);
    this.session = context.session;
    window.addEventListener('soda-session-retired', () => this.invalidate(), {signal: this.lifetime.signal});
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
    return this.mutate(
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
  private journeyBlocked() {
    return this.stale || (!this.environment && !this.canCreate);
  }
  private journeyJoinReady() {
    return (
      !this.stale &&
      !!this.detail?.environment.provisioned &&
      this.running &&
      !this.detail.login &&
      !this.detail.authority_unavailable &&
      !this.detail.native_unavailable
    );
  }
  private renderJourney() {
    if (this.journeyBlocked()) return this.renderJourneyUnavailable();
    if (this.journeyJoinReady()) return this.joinFailed ? this.renderJourneyJoinFailed() : this.renderJourneyJoin();
    if (this.environment) return this.renderJourneyExisting();
    return this.renderJourneyConfigure();
  }
  private journeyUnavailableHeading() {
    if (this.stale && this.connectVisible) return 'Reconnect to Forgejo';
    if (this.stale) return 'Project access changed';
    if (this.busy) return 'Checking project';
    return 'Project status unavailable';
  }
  private journeyUnavailableDescription() {
    if (this.stale && this.connectVisible) return html`Sign in again to restore your Forgejo access.`;
    if (this.stale) return html`Reload Spaces to check your current access.`;
    if (this.busy) return html`Checking your account and project.`;
    return html`Current project access or runtime state could not be confirmed.`;
  }
  private journeyUnavailableAction() {
    if (this.busy) return html``;
    if (this.stale && this.connectVisible)
      return html`<a class="ui primary button" href=${this.connectURL}>Reconnect to Forgejo</a>`;
    if (this.stale)
      return html`<button class="ui primary button" @click=${() => location.reload()}>Reload Spaces</button>`;
    return html`<button class="ui primary button" @click=${() => this.refresh()}>Refresh status</button>`;
  }
  private renderJourneyUnavailable() {
    return html`<section
      data-project-controls
      class="soda-spaces-controls soda-project-journey soda-ready-project"
      aria-busy=${this.busyAttr}
    >
      ${renderWorkspaceIntro({
        kind: this.busy ? 'loading' : 'unavailable',
        heading: this.journeyUnavailableHeading(),
        description: this.journeyUnavailableDescription(),
        action: this.journeyUnavailableAction(),
        helper: html`No operation is repeated when you refresh.`,
        feedback: html`<p role="status">${this.outcomeNeedsAttention ? this.outcome : ''}</p>`,
      })}
    </section>`;
  }
  private joinFailedHelper() {
    if (this.joinNeedsCheck) return html`Checking status won’t try to join again.`;
    return html`You haven’t joined this project yet.`;
  }
  private joinFailedFeedback() {
    if (!this.joinNeedsCheck && !this.busy)
      return html`<button
        class="ui button soda-quiet-action"
        @click=${(e: Event) => this.command(e, () => this.joinEnvironment())}
      >
        Try joining again
      </button>`;
    return html``;
  }
  private renderJourneyJoinFailed() {
    return html`<section
      data-project-controls
      class="soda-spaces-controls soda-project-journey soda-ready-project"
      aria-busy=${this.busyAttr}
    >
      ${renderWorkspaceIntro({
        kind: 'unavailable',
        heading: 'Couldn’t join project',
        description: html`<span role="status">${this.outcome}</span>`,
        action: html`<button
          class="ui primary button"
          ?disabled=${this.blocked}
          @click=${(e: Event) => this.command(e, () => this.refresh())}
        >
          ${this.busy ? 'Checking join status…' : 'Check join status'}
        </button>`,
        helper: this.joinFailedHelper(),
        feedback: this.joinFailedFeedback(),
      })}
    </section>`;
  }
  private joinAction() {
    const label = this.mutationPending ? 'Joining project…' : 'Join project';
    return html`<button
      class="ui primary button"
      ?disabled=${this.blocked}
      @click=${(e: Event) => this.command(e, () => this.joinEnvironment())}
    >
      ${label}
    </button>`;
  }
  private joinFeedback() {
    const tone = this.mutationPending ? 'pending' : 'warning';
    const text = this.mutationPending
      ? 'Setting up your project account…'
      : this.outcomeNeedsAttention
        ? this.outcome
        : '';
    return html`<div class="soda-intro-feedback soda-feedback" data-tone=${tone} role="status">${text}</div>
      ${this.joinRefreshAction()}`;
  }
  private joinRefreshAction() {
    if (!this.outcomeNeedsAttention || this.busy) return html``;
    return html`<button
      class="ui button soda-quiet-action"
      ?disabled=${this.blocked}
      @click=${(e: Event) => this.command(e, () => this.refresh())}
    >
      Refresh status
    </button>`;
  }
  private renderJourneyJoin() {
    return html`<section
      data-project-controls
      data-repository-id=${this.binding?.repositoryId || ''}
      data-environment-id=${this.environment?.id || ''}
      class="soda-spaces-controls soda-project-journey soda-ready-project"
      aria-busy=${this.busyAttr}
    >
      ${renderWorkspaceIntro({
        kind: 'project',
        heading: 'Project created',
        description: html`Join ${this.repositoryName} to set up your personal account<span
            >on this shared development system.</span
          >`,
        action: this.joinAction(),
        helper: html`Then you can open your first browser terminal.`,
        feedback: this.joinFeedback(),
      })}
    </section>`;
  }
  private journeyRuntimeUnavailable() {
    return !this.detail || this.detail.authority_unavailable || this.detail.native_unavailable || !this.detail.observed;
  }
  private journeyExistingKind(accountReady: boolean, stopped: boolean) {
    if (this.busy) return 'loading' as const;
    if (accountReady) return 'terminal' as const;
    if (stopped) return 'stopped' as const;
    return 'unavailable' as const;
  }
  private journeyExistingHeading(accountReady: boolean, incomplete: boolean, stopped: boolean) {
    if (accountReady) return 'Your account is ready';
    if (incomplete) return 'Project needs inspection';
    if (stopped) return 'Project not ready';
    return 'Project status unavailable';
  }
  private journeyExistingDescription(accountReady: boolean, incomplete: boolean, stopped: boolean) {
    if (accountReady) return html`Your project account is ready for a browser terminal.`;
    if (incomplete) return html`Project setup is incomplete. Ask the operator to inspect this project.`;
    if (stopped) return html`This project is stopped. Start it before joining or opening a terminal.`;
    return html`Current project access or runtime state could not be confirmed.`;
  }
  private journeyStartButton(stopped: boolean) {
    if (!stopped || !this.lifecycle) return html``;
    const label = this.mutationPending ? 'Starting project…' : 'Start project';
    return html`<button
      class="ui primary button"
      ?disabled=${this.blocked}
      @click=${(e: Event) => this.command(e, () => this.changeLifecycle(false))}
    >
      ${label}
    </button>`;
  }
  private journeyExistingAction(stopped: boolean) {
    const refreshClass = stopped && this.lifecycle ? 'ui button soda-quiet-action' : 'ui primary button';
    const refreshLabel = this.busy && !this.mutationPending ? 'Checking status…' : 'Refresh status';
    return html`${this.journeyStartButton(stopped)}<button
        class=${refreshClass}
        ?disabled=${this.blocked}
        @click=${(e: Event) => this.command(e, () => this.refresh())}
      >
        ${refreshLabel}
      </button>`;
  }
  private journeyExistingHelper(incomplete: boolean, stopped: boolean) {
    if (incomplete) return html`Do not recreate a reserved project.`;
    if (stopped && !this.lifecycle) return html`A project administrator must start this project.`;
    return html`Refreshing checks the existing project without repeating an operation.`;
  }
  private journeyExistingFlags() {
    const unavailable = this.journeyRuntimeUnavailable();
    return {
      incomplete: this.journeyIncomplete(),
      stopped: this.journeyStopped(unavailable),
      accountReady: this.journeyAccountReady(unavailable),
    };
  }
  private journeyIncomplete() {
    return !!this.detail && !this.detail.environment.provisioned;
  }
  private journeyStopped(unavailable: boolean) {
    if (unavailable) return false;
    return this.detail?.observed?.running === false;
  }
  private journeyAccountReady(unavailable: boolean) {
    if (unavailable || !this.running) return false;
    return !!this.detail?.login;
  }
  private renderJourneyExisting() {
    const flags = this.journeyExistingFlags();
    return html`<section
      data-project-controls
      data-repository-id=${this.binding?.repositoryId || ''}
      data-environment-id=${this.environment?.id || ''}
      class="soda-spaces-controls soda-project-journey soda-ready-project"
      aria-busy=${this.busyAttr}
    >
      ${renderWorkspaceIntro({
        kind: this.journeyExistingKind(flags.accountReady, flags.stopped),
        heading: this.journeyExistingHeading(flags.accountReady, flags.incomplete, flags.stopped),
        description: this.journeyExistingDescription(flags.accountReady, flags.incomplete, flags.stopped),
        action: this.journeyExistingAction(flags.stopped),
        helper: this.journeyExistingHelper(flags.incomplete, flags.stopped),
        feedback: html`<div class="soda-intro-feedback soda-feedback" data-tone="warning" role="status">
          ${this.outcomeNeedsAttention ? this.outcome : ''}
        </div>`,
      })}
    </section>`;
  }
  private configureReady() {
    return this.canCreate && !this.busy && !this.stale && !this.outcome && !this.networkReview;
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
  private renderConfigureNetwork() {
    if (!this.networkOptions?.available) return html``;
    return renderNetworkSelection(
      this.networkOptions,
      this.networkEnabled,
      this.blocked,
      (enabled) => this.onJourneyNetworkEnabled(enabled),
      'Enable project Tailnet'
    );
  }
  private renderNetworkReviewNotice() {
    if (!this.networkReview) return html``;
    return html`<p role="alert">Network availability changed. Review the option before creating.</p>
      <button
        class="ui button"
        ?disabled=${this.blocked}
        @click=${(e: Event) => this.command(e, () => this.onClearNetworkReview())}
      >
        Use without Tailnet
      </button>`;
  }
  private renderStaleReloadNote() {
    if (!this.stale) return html``;
    return html`<p>Access changed. Reload Spaces to reconnect; no operation will be repeated.</p>`;
  }
  private createDisabled() {
    return this.blocked || !this.canCreate || this.networkReview;
  }
  private configureStatusHidden() {
    return this.configureReady() || this.mutationPending;
  }
  private pendingTone(pending: boolean) {
    return pending ? 'pending' : 'warning';
  }
  private renderCreateAction() {
    const label = this.mutationPending ? 'Creating project…' : 'Create project';
    return html`<button
      class="ui primary button"
      ?disabled=${this.createDisabled()}
      @click=${(e: Event) => this.command(e, () => this.createProject())}
    >
      ${label}
    </button>`;
  }
  private renderConfigureFeedback() {
    return html`<p
        class="soda-feedback"
        data-tone=${this.pendingTone(this.busy)}
        role="status"
        ?hidden=${this.configureStatusHidden()}
      >
        ${this.status}
      </p>
      <p class="soda-feedback" data-tone=${this.pendingTone(this.mutationPending)} role="status">
        ${this.mutationPending ? 'Creating your project…' : this.outcome}
      </p>
      <button
        class="ui button soda-quiet-action"
        ?hidden=${this.configureReady() || this.busy}
        ?disabled=${this.blocked}
        @click=${(e: Event) => this.command(e, () => this.refresh())}
      >
        Refresh status
      </button>`;
  }
  private renderJourneyConfigure() {
    return html`<section
      data-project-controls
      data-repository-id=${this.binding?.repositoryId || ''}
      data-environment-id=""
      class="soda-spaces-controls soda-project-journey"
      aria-busy=${this.busyAttr}
    >
      <header class="soda-setup-step-heading">
        <p class="soda-setup-eyebrow">New project</p>
        <h2 tabindex="-1">Configure project</h2>
        <p>Choose the system for your project.</p>
      </header>
      <div class="soda-configuration-fields">
        <div class="soda-configuration-repository">
          <p>Repository</p>
          <div class="soda-config-repository">
            <span class="soda-journey-icon soda-repository-icon" aria-hidden="true"></span
            ><span class="soda-config-repository-name"
              >${this.repositoryName || 'Loading repository…'}<small>Forgejo</small></span
            ><button
              class="ui button soda-quiet-action"
              aria-label="Change repository"
              ?disabled=${this.blocked}
              @click=${(e: Event) => this.command(e, () => this.requestRepositoryChange())}
            >
              Change
            </button>
          </div>
        </div>
        ${renderProjectOS(this.profiles, this.selectedProfile, undefined, this.blocked, (value) => this.onSelectProfile(value), 'configure')}
        ${this.renderConfigureNetwork()} ${this.renderNetworkReviewNotice()}
      </div>
      <div class="soda-setup-actions">${this.renderCreateAction()}</div>
      ${this.renderConfigureFeedback()} ${this.renderStaleReloadNote()}
    </section>`;
  }
  private get connectURL() {
    return '/-/soda/login?' + journeyLoginQuery(this.binding);
  }
  private shouldRenderJourney() {
    return this.presentation === 'journey' || (this.stale && this.connectVisible);
  }
  private sectionAriaBusy() {
    return this.busy && !this.stale ? 'true' : 'false';
  }
  private sessionCaption() {
    if (!this.session) return '';
    return `Soda account: ${this.session.user.login} (ID ${this.session.user.id})`;
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
    if (this.shouldRenderJourney()) return this.renderJourney();
    return html`<section
      data-project-controls
      data-repository-id=${this.boundRepositoryId()}
      data-environment-id=${this.boundEnvironmentId()}
      class="soda-spaces-controls"
      aria-busy=${this.sectionAriaBusy()}
    >
      ${this.renderRepositoryContext()} ${this.renderViewTabs()} ${this.renderEnvironmentView()}
      ${this.renderAccessView()} ${this.renderNetworkView()}
      ${renderProjectStatus(this.repository, this.sessionCaption(), this.projectAccountCaption(), this.status, this.outcome)}
    </section>`;
  }
  private renderRepositoryContext() {
    if (!this.binding?.settings || !this.repositoryURL) return html``;
    return html`<div class="soda-repository-context">
      <h2>${this.repository}</h2>
      <nav aria-label="Repository destinations">
        <a href=${this.repositoryURL + '/settings'}>Native repository settings</a> ·
        <a href=${this.repositoryURL + '#sodaspaces'}>Open repository and workspace drawer</a>
      </nav>
      <p>
        Project OS selection affects only creation. Existing roots cannot change distribution or interface here. Create,
        Join and Start remain separate explicit actions.
      </p>
    </div>`;
  }
  private renderViewTabs() {
    return html`<div class="soda-spaces-tabs" role="tablist" aria-label="Workspace views">
      ${views.map(
        (view) => html` <button
          type="button"
          class="ui basic button"
          data-view=${view}
          role="tab"
          aria-controls=${'soda-project-' + this.binding?.repositoryId + '-' + view}
          aria-selected=${this.selected === view ? 'true' : 'false'}
          tabindex=${this.selected === view ? 0 : -1}
          @click=${() => this.select(view)}
          @keydown=${(e: KeyboardEvent) => this.tabKey(e, view)}
        >
          ${viewTabLabel(view, this.presentation)}
        </button>`
      )}
    </div>`;
  }
  private renderCreateNetworkSelection() {
    if (!this.canCreate) return html``;
    return renderNetworkSelection(this.networkOptions, this.networkEnabled, this.blocked, (enabled) =>
      this.onCreateNetworkEnabled(enabled)
    );
  }
  private networkSummaryText() {
    if (!this.network) return 'not observed';
    return (this.network.enabled ? 'managed' : 'Off') + ' · ' + this.network.state;
  }
  private renderNetworkSummary() {
    if (!this.environment) return html``;
    return html`<p data-project-network-summary>Tailnet: ${this.networkSummaryText()}</p>`;
  }
  private renderObservedOS() {
    if (!this.environment) return html``;
    return renderOSObservation(this.observedOS, this.osStatus, this.blocked, (event) =>
      this.command(event, () => this.inspectOS())
    );
  }
  private onUseSavedKeys(checked: boolean) {
    this.useSavedKeys = checked;
  }
  private onConnectClick(event: Event) {
    if (this.busy || this.stale || this.disposed) event.preventDefault();
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
  private renderEnvironmentView() {
    const intent = environmentLoginQuery(this.binding);
    return html`<section
      id=${'soda-project-' + this.binding?.repositoryId + '-environment'}
      class="soda-spaces-view"
      role="tabpanel"
      aria-label="Environment"
      ?hidden=${this.selected !== 'environment'}
    >
      ${renderProjectOS(this.profiles, this.selectedProfile, this.detail?.environment || this.environment, this.blocked, (value) => this.onSelectProfile(value))}
      ${this.renderCreateNetworkSelection()} ${this.renderNetworkSummary()} ${this.renderObservedOS()}
      ${renderEnvironment(
        {
          connectURL: '/-/soda/login?' + intent,
          connectVisible: this.connectVisible,
          busy: this.busy,
          stale: this.stale,
          signedIn: !!this.session,
          blocked: this.blocked,
          canCreate: this.canCreate,
          canJoin: this.canJoin,
          sshKeys: this.saved?.map((key) => key.fingerprint) || [],
          useSavedKeys: this.useSavedKeys,
        },
        {
          selectSSH: (checked) => this.onUseSavedKeys(checked),
          connect: (event) => this.onConnectClick(event),
          refresh: (event) => this.command(event, () => this.refresh()),
          reload: (event) => this.onReloadPage(event),
          logout: (event) => this.command(event, () => signOut(this.session?.user.id || '')),
          create: (event) => this.command(event, () => this.createProject()),
          join: (event) => this.command(event, () => this.joinEnvironment()),
        },
        renderLifecycle(
          this.lifecycle,
          this.blocked,
          this.stopConfirmed,
          (event) => this.command(event, () => this.changeLifecycle(false)),
          (event) => this.command(event, () => this.changeLifecycle(true)),
          (checked) => this.onStopConfirmed(checked)
        )
      )}
    </section>`;
  }
  private renderAccessView() {
    return html`<section
      id=${'soda-project-' + this.binding?.repositoryId + '-access'}
      class="soda-spaces-view"
      role="tabpanel"
      aria-label="Access"
      ?hidden=${this.selected !== 'access'}
    >
      ${renderConnection(this.connection, this.binding?.repositoryId || '', !!this.binding?.page, this.blocked, () => {
        void this.copyConnection();
      })}
      ${renderKeys(
        {
          saved: this.saved,
          preview: this.keyPreview,
          joined: !!this.detail?.login,
          running: this.running,
          blocked: this.blocked,
          draft: this.draft,
          emptyConfirmed: this.emptyConfirmed,
        },
        {
          remove: (event, key) =>
            this.command(event, () =>
              this.mutate(
                '/api/me/development-keys/' + key.id,
                {},
                'Saved key removed. Existing project SSH access is unchanged until explicitly applied.',
                'DELETE'
              )
            ),
          draft: (value) => this.onDraft(value),
          save: (event) => this.command(event, () => this.saveKey()),
          review: (event) => this.command(event, () => this.reviewKeys()),
          confirmEmpty: (checked) => this.onConfirmEmpty(checked),
          apply: (event) => this.command(event, () => this.applyKeys()),
        }
      )}
      ${this.renderForgejoKeyReview()}
    </section>`;
  }
  private renderForgejoKeyReview() {
    if (!this.saved) return html``;
    return renderForgejoKeys(
      this.profileKeys,
      this.blocked,
      (page) => {
        void this.reviewProfileKeys(page);
      },
      (key) => this.selectForgejoKey(key)
    );
  }
  private renderTailnetSSH() {
    if (this.network?.state !== 'connected' || !this.connection || !this.detail?.login) return html``;
    return html`<fieldset>
      <legend>Own-account Tailnet SSH</legend>
      <input
        readonly
        aria-label="Tailnet SSH command"
        .value=${'ssh ' + this.detail.login + '@' + this.network.addresses[0]}
      />
      <p>
        Ed25519 host-key fingerprint: ${this.connection.fingerprint}. Same project account and host key as LAN SSH; no
        Tailscale SSH or automatic authentication.
      </p>
    </fieldset>`;
  }
  private renderNetworkView() {
    return html`<section
      id=${'soda-project-' + this.binding?.repositoryId + '-network'}
      class="soda-spaces-view"
      role="tabpanel"
      aria-label="Network"
      ?hidden=${this.selected !== 'network'}
    >
      ${renderNetwork(
        this.network,
        this.networkNotice,
        !!this.detail?.environment_administrator,
        this.blocked,
        this.networkConfirmed,
        (confirmed) => this.onNetworkConfirmed(confirmed),
        (event, action) => this.command(event, () => this.changeNetwork(action))
      )}
      ${this.renderTailnetSSH()}
    </section>`;
  }
  private joinEnvironment() {
    if (!this.environment || !this.running || this.detail?.login || this.detail?.authority_unavailable) return;
    return this.mutate(
      '/api/environments/' + this.environment.id + '/join',
      {
        ssh_keys: this.presentation !== 'journey' && this.useSavedKeys ? 'saved' : 'none',
      },
      'Native join confirmed. Your browser terminal uses this account, not SSH. Later SSH-key changes require a separate explicit Apply.'
    );
  }
  private selectForgejoKey(key: string) {
    if (this.blocked || this.selected !== 'access' || this.closest('[hidden], [inert]')) return;
    this.draft = key;
    this.outcome =
      'Review the selected public key above, then explicitly Save public key. Joining/applying to a project remains a separate action.';
  }
  private copyBlocked() {
    return (
      !this.binding?.page ||
      this.blocked ||
      !this.connection ||
      this.selected !== 'access' ||
      !!this.closest('[hidden], [inert]')
    );
  }
  private noteCopy(ok: boolean) {
    if (this.disposed) return;
    this.outcome = ok ? 'SSH connection copied.' : 'Copy failed. Select and copy the displayed SSH command.';
  }
  private async copyConnection() {
    if (this.copyBlocked()) return;
    try {
      await navigator.clipboard.writeText(this.connection!.command);
      this.noteCopy(true);
    } catch {
      this.noteCopy(false);
    }
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
    this.session = undefined;
    this.repository = this.repositoryName = '';
    this.connectVisible = false;
    this.status =
      'Page context changed. Reload the full repository page; no action was replayed or undone. A dispatched operation may still have completed.';
  }
  private requestHeaders(method: string): Record<string, string> {
    const headers: Record<string, string> = {};
    const actor = this.binding?.expectedUserId;
    if (actor) headers['X-Soda-Expected-User-ID'] = actor;
    if (method === 'GET') return headers;
    if (!this.session) throw Error('Missing Soda session');
    headers['Content-Type'] = 'application/json';
    headers['X-CSRF-Token'] = this.session.csrf_token;
    return headers;
  }
  private admitHttpFailure(status: number) {
    if (status !== 401 && status !== 403) return;
    this.invalidate();
    this.connectVisible = status === 401;
  }
  private async api(
    path: string,
    method = 'GET',
    body?: Record<string, unknown>,
    signal?: AbortSignal
  ): Promise<unknown> {
    const response = await fetch('/-/soda' + path, sodaFetchInit(method, this.requestHeaders(method), body, signal));
    if (response.ok) return response.status === 204 ? null : readSodaJSON(response);
    this.admitHttpFailure(response.status);
    throw new SodaRequestError(response.status, await sodaErrorCode(response));
  }
  private refreshBlocked() {
    return this.busy || this.stale || this.disposed || !this.binding;
  }
  private beginRefreshRead(): AbortController {
    this.readController?.abort();
    this.reset();
    const control = (this.readController = new AbortController());
    this.busy = true;
    this.connectVisible = false;
    this.status = 'Checking your account and environment…';
    return control;
  }
  async refresh() {
    if (this.refreshBlocked()) return;
    const {expectedUserId, repositoryId} = this.binding!;
    const n = ++this.epoch;
    const prior: RefreshPrior = {
      profile: this.selectedProfile,
      network: this.networkOptions,
      enabled: this.networkEnabled,
    };
    const control = this.beginRefreshRead();
    const timeout = window.setTimeout(() => control.abort(), 15000);
    let recoveredJoin = false;
    try {
      recoveredJoin = await this.refreshAccount(n, expectedUserId, repositoryId, prior, control);
    } catch (error) {
      this.refreshFailed(n, error);
    } finally {
      await this.finishRefresh(n, timeout, repositoryId, recoveredJoin);
    }
  }
  private async refreshAccount(
    n: number,
    expectedUserId: string | undefined,
    repositoryId: string,
    prior: RefreshPrior,
    control: AbortController
  ): Promise<boolean> {
    if (!(await this.admitRefreshSession(n, expectedUserId, control))) return false;
    const collection = await this.loadEnvironmentCollection(n, repositoryId, control);
    if (!collection || !this.active(n)) return false;
    if (collection.items.length) return this.refreshExisting(n, repositoryId, collection.items[0], control);
    await this.refreshEmpty(n, repositoryId, prior, collection.can_create, control);
    return false;
  }
  private async admitRefreshSession(
    n: number,
    expectedUserId: string | undefined,
    control: AbortController
  ): Promise<boolean> {
    const found =
      this.session ||
      sessionResponse(await this.api('/api/session', 'GET', undefined, control.signal), location.origin);
    if (!this.active(n)) return false;
    if (expectedUserId && found.user.id === expectedUserId) {
      this.session = found;
      return true;
    }
    this.invalidate();
    this.connectVisible = true;
    this.status = 'Soda and native page identities differ. Sign out of Soda or reload and connect explicitly.';
    return false;
  }
  private admitCollection(
    collection: Record<string, unknown>,
    repository: Record<string, unknown>,
    repositoryId: string
  ) {
    check(
      repository.id === repositoryId &&
        Array.isArray(collection.items) &&
        collection.items.length <= 1 &&
        typeof collection.can_create === 'boolean'
    );
    check(typeof repository.owner === 'string' && typeof repository.name === 'string');
    for (const part of [repository.owner, repository.name]) admitRepositoryPart(part);
  }
  private applyRepositoryLabels(repository: Record<string, unknown>, repositoryId: string) {
    const owner = repository.owner as string,
      name = repository.name as string;
    this.repositoryName = `${owner}/${name}`;
    this.repository = `Repository ${owner}/${name} · ID ${repositoryId}`;
    this.repositoryURL = location.origin + '/' + encodeURIComponent(owner) + '/' + encodeURIComponent(name);
  }
  private async loadEnvironmentCollection(n: number, repositoryId: string, control: AbortController) {
    const collection = object(
      await this.api('/api/environments?repository_id=' + repositoryId, 'GET', undefined, control.signal)
    );
    const repository = object(collection.repository);
    if (!this.active(n)) return;
    this.admitCollection(collection, repository, repositoryId);
    this.applyRepositoryLabels(repository, repositoryId);
    return collection as typeof collection & {items: unknown[]; can_create: boolean};
  }
  private async refreshEmpty(
    n: number,
    repositoryId: string,
    prior: RefreshPrior,
    canCreate: boolean,
    control: AbortController
  ) {
    if (canCreate && !(await this.refreshCreateOptions(n, repositoryId, prior, control))) return;
    if (!this.active(n)) return;
    this.status =
      this.presentation === 'journey'
        ? 'Creation is owner-only. You join separately after the project is ready.'
        : 'No shared environment. Creation is owner-only and does not join you.';
  }
  private selectedCreateProfile(prior: RefreshPrior) {
    if (this.profiles.some((p) => p.id === prior.profile)) return prior.profile;
    return this.profiles[0]?.id || '';
  }
  private async refreshCreateOptions(
    n: number,
    repositoryId: string,
    prior: RefreshPrior,
    control: AbortController
  ): Promise<boolean> {
    const available = object(
      await this.api('/api/repositories/' + repositoryId + '/profiles', 'GET', undefined, control.signal)
    );
    if (!this.active(n)) return false;
    check(Array.isArray(available.items) && available.items.length === 1);
    this.profiles = available.items.map(creationProfile);
    this.selectedProfile = this.selectedCreateProfile(prior);
    this.canCreate = !!this.selectedProfile;
    return this.refreshTailnetOptions(n, repositoryId, prior, control);
  }
  private networkReviewNeeded(prior: RefreshPrior, options: ProjectOptions) {
    if (!prior.enabled) return false;
    if (!options.available) return true;
    if (prior.network?.binding !== options.binding) return true;
    return prior.network.revision !== options.revision;
  }
  private networkEnabledAfterRefresh(prior: RefreshPrior, options: ProjectOptions) {
    if (prior.enabled) return true;
    return this.presentation !== 'journey' && options.available && options.default;
  }
  private async refreshTailnetOptions(
    n: number,
    repositoryId: string,
    prior: RefreshPrior,
    control: AbortController
  ): Promise<boolean> {
    try {
      const options = projectOptions(
        await this.api('/api/repositories/' + repositoryId + '/tailnet-options', 'GET', undefined, control.signal)
      );
      if (!this.active(n)) return false;
      this.networkOptions = options;
      this.networkReview = this.networkReviewNeeded(prior, options);
      this.networkEnabled = this.networkEnabledAfterRefresh(prior, options);
      return true;
    } catch (error) {
      return this.tailnetOptionsFailed(n, prior, error);
    }
  }
  private tailnetOptionsFailed(n: number, prior: RefreshPrior, error: unknown): boolean {
    if (error instanceof SodaRequestError && error.status === 401) throw error;
    if (!this.active(n)) return false;
    this.networkOptions = undefined;
    this.networkEnabled = prior.enabled;
    this.networkReview = prior.enabled;
    return true;
  }
  private applyJoinRecovery(detail: Detail) {
    if (!this.joinFailed || detail.authority_unavailable || detail.native_unavailable) return false;
    this.joinNeedsCheck = false;
    if (!detail.login) return false;
    this.joinFailed = false;
    this.outcomeNeedsAttention = false;
    this.outcome = '';
    return true;
  }
  private environmentStatus(detail: Detail) {
    if (!detail.environment.provisioned)
      return 'Provisioning incomplete. Ask the operator to inspect; do not recreate it.';
    if (detail.native_unavailable || !detail.observed)
      return 'Native state unavailable; refresh or ask the operator to inspect.';
    if (this.running) return 'Environment running.';
    return 'Environment stopped.';
  }
  private environmentNeedsAdminStart(detail: Detail) {
    return (
      detail.environment.provisioned && !this.running && !detail.native_unavailable && !detail.environment_administrator
    );
  }
  private applyEnvironmentStatus(detail: Detail) {
    this.status = this.environmentStatus(detail);
    if (this.environmentNeedsAdminStart(detail))
      this.status += ' Ask the project administrator or Soda operator to Start it; nothing is started automatically.';
  }
  private async refreshExisting(
    n: number,
    repositoryId: string,
    item: unknown,
    control: AbortController
  ): Promise<boolean> {
    const environment = environmentResponse(item, repositoryId);
    this.environment = environment;
    const detail = detailResponse(
      await this.api(`/api/environments/${environment.id}`, 'GET', undefined, control.signal),
      environment
    );
    if (!this.active(n)) return false;
    this.detail = detail;
    const recoveredJoin = this.applyJoinRecovery(detail);
    if (detail.login) admitProjectLogin(detail.login);
    this.applyEnvironmentStatus(detail);
    if (!(await this.refreshOptionalDetails(n, environment, detail, control))) return false;
    return recoveredJoin;
  }
  private async refreshOptionalDetails(
    n: number,
    environment: Environment,
    detail: Detail,
    control: AbortController
  ): Promise<boolean> {
    if (!(await this.refreshSavedKeysIfStandard(n, control))) return false;
    if (!(await this.refreshLifecycleIfAdmin(n, environment, detail, control))) return false;
    if (!(await this.refreshConnectionIfJoined(n, environment, detail, control))) return false;
    return this.refreshProjectNetworkIfEligible(n, environment, detail, control);
  }
  private async refreshSavedKeysIfStandard(n: number, control: AbortController): Promise<boolean> {
    if (this.presentation === 'journey') return true;
    return this.refreshSavedKeys(n, control);
  }
  private async refreshSavedKeys(n: number, control: AbortController): Promise<boolean> {
    try {
      const saved = savedKeysResponse(await this.api('/api/me/development-keys', 'GET', undefined, control.signal));
      if (!this.active(n)) return false;
      this.saved = saved;
      return true;
    } catch {
      if (!this.active(n)) return false;
      this.outcome = 'External SSH keys unavailable. Browser-only Join remains independent.';
      return true;
    }
  }
  private async refreshLifecycleIfAdmin(
    n: number,
    environment: Environment,
    detail: Detail,
    control: AbortController
  ): Promise<boolean> {
    if (!detail.environment.provisioned || !detail.environment_administrator) return true;
    const state = object(
        await this.api(`/api/environments/${environment.id}/lifecycle`, 'GET', undefined, control.signal)
      ),
      native = object(state.environment);
    if (!this.active(n)) return false;
    check(
      native.id === environment.id && typeof native.running === 'boolean' && typeof state.boot_enabled === 'boolean'
    );
    this.lifecycle = {
      running: native.running,
      boot: state.boot_enabled,
    };
    return true;
  }
  private shouldLoadConnection(detail: Detail) {
    return !!detail.login && this.running && this.presentation !== 'journey';
  }
  private admitConnectionPayload(
    own: Record<string, unknown>,
    connection: Record<string, unknown>,
    native: Record<string, unknown>,
    environment: Environment,
    detail: Detail
  ) {
    check(
      own.login === detail.login &&
        native.id === environment.id &&
        native.running &&
        typeof native.ip === 'string' &&
        /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(native.ip) &&
        this.validIPv4(native.ip) &&
        fingerprint(connection.fingerprint)
    );
  }
  private validIPv4(ip: string) {
    return ip.split('.').every((octet) => Number(octet) <= 255);
  }
  private async refreshConnectionIfJoined(
    n: number,
    environment: Environment,
    detail: Detail,
    control: AbortController
  ): Promise<boolean> {
    if (!this.shouldLoadConnection(detail)) return true;
    admitProjectLogin(detail.login!);
    const own = object(
      await this.api(`/api/environments/${environment.id}/connection`, 'GET', undefined, control.signal)
    );
    if (!this.active(n)) return false;
    const c = object(own.connection),
      native = object(c.environment);
    this.admitConnectionPayload(own, c, native, environment, detail);
    await this.updateComplete;
    if (!this.active(n)) return false;
    this.connection = {
      command: `ssh ${detail.login}@${native.ip}`,
      fingerprint: c.fingerprint as string,
    };
    return true;
  }
  private shouldLoadProjectNetwork(detail: Detail) {
    return detail.environment.provisioned && (!!detail.login || !!detail.environment_administrator);
  }
  private async refreshProjectNetworkIfEligible(
    n: number,
    environment: Environment,
    detail: Detail,
    control: AbortController
  ): Promise<boolean> {
    if (!this.shouldLoadProjectNetwork(detail)) return true;
    return this.refreshProjectNetwork(n, environment, control);
  }
  private async refreshProjectNetwork(n: number, environment: Environment, control: AbortController): Promise<boolean> {
    try {
      const network = projectView(
        await this.api(`/api/environments/${environment.id}/tailnet`, 'GET', undefined, control.signal),
        environment.id
      );
      check(!network.saved);
      if (!this.active(n)) return false;
      this.network = network;
      return true;
    } catch (error) {
      return this.projectNetworkFailed(n, error);
    }
  }
  private projectNetworkFailed(n: number, error: unknown): boolean {
    if (error instanceof SodaRequestError && error.status === 401) throw error;
    if (!this.active(n)) return false;
    this.network = undefined;
    this.networkNotice =
      'Private network state unavailable. No disconnected state was inferred; ordinary project controls remain independent.';
    return true;
  }
  private refreshErrorStatus(e: SodaRequestError) {
    if (e.code === 'profile_unavailable')
      return 'Installed Project OS unavailable or incompatible. Nothing was reserved, pulled or started; ask the operator to inspect.';
    if (this.connectVisible) return 'Connect through Forgejo to authorize Soda access.';
    return 'Could not confirm state. Refresh; do not infer absence or retry an uncertain action.';
  }
  private refreshFailed(n: number, error: unknown) {
    if (!this.active(n)) return;
    const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
    this.reset();
    this.repository = '';
    if (e.status === 401 || e.status === 403) this.session = undefined;
    this.connectVisible = e.status === 401 || e.status === 403;
    this.status = this.refreshErrorStatus(e);
  }
  private async finishRefresh(n: number, timeout: number, repositoryId: string, recoveredJoin: boolean) {
    window.clearTimeout(timeout);
    if (!this.active(n)) return;
    this.busy = false;
    this.dispatchEvent(
      new CustomEvent('soda-project-observed', {
        bubbles: true,
        detail: {
          repositoryId,
          environmentId: this.environment?.id || '',
          provisioned: this.detail?.environment.provisioned === true,
          login: this.detail?.login || '',
          running: this.running,
        },
      })
    );
    if (recoveredJoin)
      this.dispatchEvent(new CustomEvent('soda-project-changed', {bubbles: true, detail: {repositoryId}}));
    await this.updateComplete;
  }
  private async mutate(path: string, body: Record<string, unknown>, message: string, method = 'POST') {
    if (this.blocked || !this.session) return;
    const n = this.epoch;
    this.busy = true;
    this.outcomeNeedsAttention = false;
    this.joinFailed = false;
    this.outcome = 'Sending the explicit operation with the original page identity…';
    let dispatched = false,
      inspectCreation = false;
    const controller = new AbortController(),
      timeout = window.setTimeout(() => controller.abort(), 255000);
    try {
      dispatched = true;
      inspectCreation = await this.dispatchMutation(n, path, body, message, method, controller);
    } catch (error) {
      inspectCreation = this.applyMutateError(path, error, dispatched);
    } finally {
      await this.finishMutation(n, timeout, inspectCreation);
    }
  }
  private async dispatchMutation(
    n: number,
    path: string,
    body: Record<string, unknown>,
    message: string,
    method: string,
    controller: AbortController
  ) {
    this.mutationPending = true;
    this.dispatchEvent(new CustomEvent('soda-project-operation', {bubbles: true}));
    this.outcome = 'Request dispatched. Closing does not cancel or undo native work.';
    const result = mutationObject(await this.api(path, method, body, controller.signal));
    if (!this.active(n)) return false;
    const next = this.checkMutationResult(path, method, body, result);
    this.completeMutation(next || message);
    this.busy = false;
    await this.refresh();
    return false;
  }
  private completeMutation(message: string) {
    this.mutationPending = false;
    this.dispatchEvent(new CustomEvent('soda-project-operation', {bubbles: true}));
    this.outcome = message;
    this.dispatchEvent(
      new CustomEvent('soda-project-changed', {
        bubbles: true,
        detail: {
          repositoryId: this.binding?.repositoryId,
        },
      })
    );
  }
  private checkMutationResult(
    path: string,
    method: string,
    body: Record<string, unknown>,
    result: Record<string, unknown> | null
  ): string | undefined {
    if (path === '/api/environments') return this.checkCreateMutation(body, result);
    if (path.endsWith('/join')) return this.checkJoinMutation(result);
    if (path.endsWith('/lifecycle')) return this.checkLifecycleMutation(body, result);
    if (path.endsWith('/tailnet')) return this.checkTailnetMutation(body, result);
    if (path.endsWith('/access-keys')) return this.checkAccessKeysMutation(body, result);
    if (method === 'DELETE') return this.checkDeleteMutation(result);
    if (path === '/api/me/development-keys') return this.checkSaveKeyMutation(body, result);
    return undefined;
  }
  private checkCreateMutation(
    body: Record<string, unknown>,
    result: Record<string, unknown> | null
  ): string | undefined {
    check(
      result &&
        projectId(result.id) &&
        result.repository_id === this.binding?.repositoryId &&
        result.provisioned === true &&
        creationProfile(result.profile).id === body.profile_id
    );
    if (object(body.tailnet).enabled !== true) return undefined;
    check(result.tailnet_outcome === 'queued' || result.tailnet_outcome === 'unconfirmed');
    this.outcomeNeedsAttention = true;
    if (result.tailnet_outcome === 'queued') return 'Project created. Network policy saved; enrollment queued.';
    return 'Project created. Network setup unconfirmed; inspect Network and explicitly retry there. Do not recreate the project.';
  }
  private checkJoinMutation(result: Record<string, unknown> | null): string | undefined {
    check(typeof result?.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(result.login) && result.login !== 'root');
    return undefined;
  }
  private checkLifecycleMutation(
    body: Record<string, unknown>,
    result: Record<string, unknown> | null
  ): string | undefined {
    check(
      result &&
        object(result.environment).id === this.environment?.id &&
        object(result.environment).running === (body.action === 'start') &&
        result.boot_enabled === (body.action === 'start')
    );
    return undefined;
  }
  private checkTailnetMutation(
    body: Record<string, unknown>,
    result: Record<string, unknown> | null
  ): string | undefined {
    const network = projectView(result, this.environment?.id || '');
    check(network.saved && network.revision !== body.revision && network.enabled === (body.action !== 'disable'));
    if (network.outcome === 'queued')
      return 'Network policy saved; native work queued. Refresh observes the outcome without replay.';
    return 'Network policy saved; native outcome unconfirmed. Observe before retrying; no connection or disconnection was assumed.';
  }
  private checkAccessKeysMutation(
    body: Record<string, unknown>,
    result: Record<string, unknown> | null
  ): string | undefined {
    check(
      result?.applied === true &&
        result.login === this.detail?.login &&
        typeof result.revision === 'string' &&
        /^[0-9a-f]{64}$/.test(result.revision) &&
        JSON.stringify(result.installed_fingerprints) === JSON.stringify(body.saved_fingerprints)
    );
    return undefined;
  }
  private checkDeleteMutation(result: Record<string, unknown> | null): string | undefined {
    check(result?.removed === true && result.existing_project_access_changed === false);
    return undefined;
  }
  private checkSaveKeyMutation(
    body: Record<string, unknown>,
    result: Record<string, unknown> | null
  ): string | undefined {
    const keys = savedKeysResponse(result);
    check(typeof body.public_key === 'string');
    const publicKey = body.public_key;
    check(keys.some((k) => publicKeyToken(k.public_key) === publicKeyToken(publicKey)));
    return undefined;
  }
  private joinErrorMessage(code: string | undefined, uncertain: boolean) {
    this.joinFailed = true;
    this.joinNeedsCheck = true;
    if (code === 'account_incomplete')
      return 'Soda could not finish setting up your project account. Ask your Soda administrator to check the account setup before you try again.';
    if (code === 'membership_not_saved')
      return 'Your account was set up, but Soda could not save your access to this project. Ask your Soda administrator to check your project access.';
    if (code === 'unsupported_linux_login')
      return 'Your Forgejo username cannot be used for a project account. Ask your Soda administrator for help choosing a supported username.';
    if (uncertain) return 'We couldn’t confirm whether you joined. Check join status before trying again.';
    return 'Your request to join was declined. Check your project access or ask your Soda administrator for help.';
  }
  private mutateReason(code: string | undefined) {
    if (code === 'profile_unavailable')
      return 'Installed Project OS unavailable. No reservation was created; refresh before another explicit action.';
    if (code === 'unsupported_linux_login')
      return 'Your Forgejo username is not supported as a Linux login. No automatic rename is performed.';
    if (code === 'invalid_public_key') return 'Provide one public SSH key without options or private key material.';
    if (code === 'saved_keys_changed') return 'Saved keys changed. Review them again before Apply.';
    if (code === 'owner_required') return 'Only the current human repository owner can create this environment.';
    if (code === 'not_provisioned')
      return 'Provisioning is incomplete. Ask the operator to inspect; do not recreate it.';
  }
  private mutateErrorMessage(path: string, e: SodaRequestError, uncertain: boolean) {
    if (path.endsWith('/join')) return this.joinErrorMessage(e.code, uncertain);
    const reason = this.mutateReason(e.code);
    if (!uncertain && reason) return reason;
    if (path === '/api/environments' && uncertain)
      return 'Project creation could not be confirmed. Check the project status before trying again.';
    if (uncertain)
      return 'We couldn’t confirm that this change finished. Refresh status to check the result before trying again.';
    return 'This change could not be applied. Refresh status and review your settings before trying again.';
  }
  private applyMutateError(path: string, error: unknown, dispatched: boolean) {
    const e = error instanceof SodaRequestError ? error : new SodaRequestError(0);
    const uncertain = dispatched && !rejected.has(e.status);
    const inspectCreation = uncertain && path === '/api/environments';
    if (inspectCreation) this.noteUnconfirmedCreation();
    this.mutationPending = false;
    this.outcomeNeedsAttention = true;
    this.outcome = this.mutateErrorMessage(path, e, uncertain);
    return inspectCreation;
  }
  private noteUnconfirmedCreation() {
    this.canCreate = false;
    this.dispatchEvent(
      new CustomEvent('soda-project-changed', {bubbles: true, detail: {repositoryId: this.binding?.repositoryId}})
    );
  }
  private async finishMutation(n: number, timeout: number, inspectCreation: boolean) {
    window.clearTimeout(timeout);
    this.mutationPending = false;
    if (!this.active(n)) return;
    this.busy = false;
    if (inspectCreation) await this.refresh();
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
    return this.mutate(
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
    return this.mutate(
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
    return this.mutate(
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
    if (!this.keyPreview || !this.environment) return;
    if (!this.keyPreview.saved_fingerprints.length && !this.emptyConfirmed) {
      this.outcome = 'Explicitly confirm removal of the last managed key.';
      return;
    }
    const preview = this.keyPreview;
    this.keyPreview = undefined;
    this.emptyConfirmed = false;
    return this.mutate(
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

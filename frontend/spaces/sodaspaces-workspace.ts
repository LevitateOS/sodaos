import type {WorkspaceView} from './sodaspaces-workspace-types.js';
import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {renderMenu, renderWelcomeSteps} from './sodaspaces-workspace-view.js';
import {renderRename} from './sodaspaces-terminal-dialog-view.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {resumeGenerationRefused} from './soda-extension.js';
import type {ContributionPresentation, ResumeValidation} from './fountain-browser-contract.js';
import {mountTerminal} from './sodaspaces-terminal.js';
import type {TerminalContext} from './sodaspaces-terminal.js';
import {emptyLayout, focusedPane, hideTab} from './sodaspaces-layout.js';
import type {WorkspaceLayout, LayoutEntry, Area} from './sodaspaces-layout.js';
import {check, id, object, projectId} from './sodaspaces-api.js';
import type {RepositoryChoices} from './sodaspaces-repository-response.js';
import type {Space} from './sodaspaces-inventory-response.js';
import type {
  Creation,
  FactoryWatch,
  Row,
  Slot,
  WorkspaceContext,
  WorkspacePresentation,
} from './sodaspaces-workspace-types.js';
import {WorkspaceMeasurement} from './sodaspaces-workspace-measurement.js';
import {
  busyAttr,
  hideCanvas,
  hideManagement,
  hideNavigation,
  hidePaneChrome,
  hideSidebarDivider,
  hideWorkspaceBody,
  renderFirstTerminalIntro,
  renderInventoryRecovery,
  renderSetupHeading,
  renderSetupTopbar,
  renderStatusBanners,
  workspaceBlocked,
  workspaceBodyClass,
  workspaceBodyStyle,
  workspaceClasses,
} from './sodaspaces-workspace-shell-view.js';
import {
  disableNewTerminal,
  hideNewTerminal,
  newTerminalButtonClass,
  newTerminalButtonLabel,
  renderBackButton,
  renderNativeManagementOption,
  renderOpenInDrawerOption,
  renderProjectSettingsButton,
  renderSpacesLink,
  renderToggleSidebarOption,
  renderToolbarProject,
  sessionsButtonHidden,
  sessionsButtonLabel,
} from './sodaspaces-workspace-toolbar-view.js';
import {
  backToReview,
  beginSetup,
  cancelSetup,
  focusSetupPanel,
  inspectCommittedRequest,
  renderSetup,
} from './sodaspaces-workspace-setup.js';
import {
  isNewRouteLocation,
  newRouteFor,
  requestIdFromLocation,
  requestIDPattern,
  requestRouteFor,
} from './sodaspaces-workspace-setup.js';
import {resolveWatchTarget, resumeSetup} from './sodaspaces-workspace-setup.js';
import type {FactoryWatchRequest, SetupActions, SetupReading} from './sodaspaces-workspace-setup.js';
import type {OverviewScope} from './sodaspaces-factory-progress-view.js';
import type {SodaIntent} from './sodaspaces-intent.js';
import type {SavedRequest} from './sodaspaces-intent-response.js';
import './sodaspaces-project-work.js';
import './sodaspaces-applications.js';
import type {SodaProjectWork} from './sodaspaces-project-work.js';
import {
  attentionRows,
  filteredSpaces,
  groupSavedRequests,
  projectRows,
  renderRequestGroups,
  renderProjectDirectory,
  renderProjectTerminals,
  rows,
} from './sodaspaces-workspace-navigation.js';
import {displayFactoryWatches, factorySection, factoryWatching, watchRun} from './sodaspaces-workspace-factory.js';
import {paneChrome} from './sodaspaces-workspace-pane-view.js';
import type {PaneChromeInput} from './sodaspaces-workspace-pane-view.js';
import {
  closeMenus as closeWorkspaceMenus,
  menuFocusOut,
  rememberFocus as rememberWorkspaceFocus,
  restoreFocus as restoreWorkspaceFocus,
  workspaceClick,
  workspaceKey,
} from './sodaspaces-workspace-focus.js';
import {
  api as inventoryApi,
  live as inventoryLive,
  loadMore,
  refresh as inventoryRefresh,
  renderMoreProjects,
} from './sodaspaces-workspace-inventory.js';
import type {InventoryInput, MoreProjectsInput} from './sodaspaces-workspace-inventory.js';
import {
  arrange,
  canSplit,
  compact as layoutCompact,
  focusPane,
  loadLayout,
  move,
  paneMinimum,
  persist as layoutPersist,
  projection as layoutProjection,
  renderPaneDivider,
  resizeSidebar,
  sidebarKey,
  split,
  toggleSidebar,
} from './sodaspaces-workspace-layout.js';
import type {LayoutInput, MinimumInput} from './sodaspaces-workspace-layout.js';
import {openInDrawer} from './sodaspaces-workspace-drawer.js';
import type {DrawerInput} from './sodaspaces-workspace-drawer.js';
import {addSlot, displaySlots, visibleSlot} from './sodaspaces-workspace-terminal-hosts.js';
import type {HostsInput} from './sodaspaces-workspace-terminal-hosts.js';
import {
  confirmedEnd,
  createTerminal,
  creationForm,
  defaultTerminalName,
  newTerminal,
  openExisting,
  openSaved as openSavedTerminal,
  rename,
  restoreLocators,
  slotName,
} from './sodaspaces-workspace-terminals.js';
import type {
  CreateInput,
  CreationFormInput,
  NewTerminalInput,
  RenameInput,
  TerminalsInput,
} from './sodaspaces-workspace-terminals.js';
type TerminalFactory = typeof mountTerminal;
const validName = (name: string) => [...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name);

// Both surfaces share this owner. Pane chrome is keyed separately; live terminal
// hosts never leave their flat parent. Rendering cannot create/attach/Return.
export class SodaSpaces extends LitElement {
  static properties = {
    setup: {state: true},
    intentNotice: {state: true},
    savedRequests: {state: true},
    project: {state: true},
    deliveryTarget: {state: true},
    complete: {state: true},
    repositoryQuery: {state: true},
    repositoryResult: {state: true},
    repositoryChoice: {state: true},
    repositoryBusy: {state: true},
    repositoryError: {state: true},
    overviewScope: {state: true},
    spaces: {
      state: true,
    },
    status: {
      state: true,
    },
    busy: {
      state: true,
    },
    layout: {
      state: true,
    },
    storageNotice: {
      state: true,
    },
    view: {
      state: true,
    },
    stale: {
      state: true,
    },
    search: {
      state: true,
    },
    thisPage: {
      state: true,
    },
    creation: {
      state: true,
    },
    creating: {
      state: true,
    },
    editing: {
      state: true,
    },
    attentionOnly: {
      state: true,
    },
    openingDrawer: {
      state: true,
    },
    nextAfter: {
      state: true,
    },
  };
  declare private spaces: Space[];
  declare private status: string;
  declare private busy: boolean;
  declare private layout: WorkspaceLayout;
  declare private storageNotice: string;
  declare private view: WorkspaceView;
  declare private stale: boolean;
  declare private search: string;
  declare private thisPage: boolean;
  declare private creation: Creation | null;
  declare private creating: boolean;
  declare private editing: {
    key: string;
    name: string;
  } | null;
  declare private attentionOnly: boolean;
  declare private openingDrawer: boolean;
  declare private nextAfter: string;
  private reconnectRequired = false;
  private observedAt = 0;
  private now = Date.now();
  private attentionTimer: number | undefined;
  private binding: WorkspaceContext | undefined;
  private routePathname = '';
  private factory: TerminalFactory = mountTerminal;
  private slots: Slot[] = [];
  private watches: FactoryWatch[] = [];
  private projects = new Map<
    string,
    {
      host: HTMLElement;
      api: ReturnType<typeof mountProjectControls>;
    }
  >();
  declare private project: string;
  declare private deliveryTarget: string;
  private deliveryRepository = '';
  private companyScopeRepository = '';
  private applicationOwner = '';
  declare private setup: 'intent' | 'repositories' | 'configure' | null;
  declare private intentNotice: string;
  declare private savedRequests: SavedRequest[];
  declare private complete: boolean;
  declare private repositoryQuery: string;
  declare private repositoryResult: RepositoryChoices | undefined;
  declare private repositoryChoice: string;
  declare private repositoryBusy: boolean;
  declare private repositoryError: string;
  private repositoryRequest: AbortController | undefined;
  private repositoryCursors: string[] = [''];
  private setupReturn: {project: string; view: WorkspaceView; mode: 'standard' | 'journey' | 'settings'} | undefined;
  private reviewAuthorization: {ideas: number; tasks: number} | undefined;
  private reviewedIntent: string | undefined;
  private setupCommitted = false;
  private workRepository: string | undefined;
  declare private overviewScope: OverviewScope | undefined;
  private managementMode: 'standard' | 'journey' | 'settings' = 'standard';
  private request: AbortController | undefined;
  private renaming = new Set<string>();
  private epoch = 0;
  private disposed = false;
  private restored = false;
  private storageLoaded = false;
  private actor: {id: string; login: string} | undefined;
  private available = false;
  private surfaceVisible = true;
  private storageKey = '';
  private restoredProjectView = '';
  private lifetime = new AbortController();
  private canvasSize = {
    width: 0,
    height: 0,
  };
  private workspaceWidth = 0;
  private cell = {
    width: 9,
    height: 20,
  };
  private measurement = new WorkspaceMeasurement(this, () => this.measure());
  private maximized: string | undefined;
  private dragged: string | undefined;
  private choosingPane: string | undefined;
  private pendingNewRoute = false;
  private pendingWorkspaceRoute = false;
  private invoker: HTMLElement | undefined;
  private lastMinimum = 0;
  constructor() {
    super();
    this.deliveryTarget = '';
    this.spaces = [];
    this.savedRequests = [];
    this.project =
      this.repositoryQuery =
      this.repositoryChoice =
      this.repositoryError =
      this.nextAfter =
      this.intentNotice =
        '';
    this.setup = null;
    this.overviewScope = undefined;
    this.complete = this.repositoryBusy = false;
    this.repositoryResult = undefined;
    this.status = 'Loading projects…';
    this.busy = this.stale = this.thisPage = this.creating = false;
    this.attentionOnly = this.openingDrawer = false;
    this.view = 'terminal';
    this.search = this.storageNotice = '';
    this.creation = this.editing = null;
    this.layout = emptyLayout(crypto.randomUUID());
  }
  protected createRenderRoot() {
    return this;
  }
  private get activeSurface() {
    return this.surfaceVisible && this.isConnected;
  }
  private get selectedSpace() {
    return this.spaces.find((s) => s.environment.repository_id === this.project);
  }
  private get setupScreen() {
    return (
      this.binding?.kind === 'page' &&
      (!!this.setup ||
        (!this.spaces.length && !this.binding.repositoryId) ||
        requestIdFromLocation(this.routePathname, this.repositoryPrefix()) !== '')
    );
  }
  private get welcomeScreen() {
    return this.setupScreen && !this.setup && this.available && this.complete;
  }
  private pageReadyForFirstTerminal() {
    return this.binding?.kind === 'page' && !this.setupScreen && this.complete && !this.selected;
  }
  private spaceReadyForFirstTerminal(space: Space | undefined) {
    if (!space?.login || space.authority_unavailable || !space.execution_allowed || space.native_unavailable)
      return false;
    if (space.observed?.running !== true || !space.environment.provisioned) return false;
    return !rows(space, this.layout.entries, this.slots).length;
  }
  private projectRunnable(space: Space) {
    return (
      !!space.login &&
      !space.authority_unavailable &&
      space.execution_allowed &&
      !space.native_unavailable &&
      space.environment.provisioned &&
      space.observed?.running === true
    );
  }
  private get firstTerminal() {
    return (
      this.view === 'terminal' &&
      this.pageReadyForFirstTerminal() &&
      this.spaceReadyForFirstTerminal(this.selectedSpace)
    );
  }
  private get inventoryRecovery() {
    return (
      this.binding?.kind === 'page' && !this.setupScreen && !this.complete && !this.selected && this.view === 'terminal'
    );
  }
  private get workspaceIntro() {
    return (
      this.firstTerminal ||
      this.inventoryRecovery ||
      (this.binding?.kind === 'page' &&
        !this.setupScreen &&
        this.view === 'project' &&
        this.managementMode === 'journey')
    );
  }
  private projectState(space: Space): 'running' | 'stopped' | 'unknown' {
    return space.authority_unavailable || space.native_unavailable || !space.observed
      ? 'unknown'
      : space.observed.running
        ? 'running'
        : 'stopped';
  }
  private projectStatus(space: Space) {
    const state = this.projectState(space);
    return state === 'unknown' ? 'Status unavailable' : state === 'running' ? 'Running' : 'Stopped';
  }
  private get selected() {
    return focusedPane(this.layout).selected || '';
  }
  private get compact() {
    return layoutCompact(this.layoutInput());
  }
  private get projection() {
    return layoutProjection(this.layoutInput());
  }
  private minimumInput(): MinimumInput {
    return {slots: this.slots, cell: this.cell, tabHeight: this.tabHeight};
  }
  private locator(slot: Slot) {
    const entry = this.layout.entries.find((entry) => entry.key === slot.key);
    check(entry);
    return entry.locator;
  }
  private get tabHeight() {
    return this.workspaceWidth < 800 ? 48 : 40;
  }
  configure(context: WorkspaceContext, factory: TerminalFactory) {
    if (this.binding) throw Error('Workspace binding is immutable');
    this.binding = {
      ...context,
    };
    if (context.kind === 'page' && context.repositoryId) this.project = context.repositoryId;
    const requestedProject = new URLSearchParams(location.search).get('repository');
    if (context.kind === 'page' && requestedProject && id(requestedProject)) this.project = requestedProject;
    this.view = context.kind === 'page' ? 'feed' : 'terminal';
    this.factory = factory;
    this.storageKey = '';
    this.routePathname = window.location.pathname;
    if (context.kind === 'page' && isNewRouteLocation(this.routePathname, this.repositoryPrefix())) {
      this.pendingNewRoute = true;
      // A direct load or reload of a project's new-request route keeps the
      // request bound to that project, as the in-project New button does.
      if (requestedProject && id(requestedProject)) this.workRepository = requestedProject;
    }
    this.addEventListener('focusout', (event) => menuFocusOut(event), {signal: this.lifetime.signal});
    // Host lifecycle only (NFX-D05): no pagehide/pageshow/visibilitychange
    // self-interpretation. The entry forwards host update/suspend/resume/
    // retire here; ordinary navigation retains every terminal and watch.
    // One mounted-workspace timer, never a navbar poller. GET refresh uses the
    // existing bounded/cancellable caller, and cannot create work.
    this.attentionTimer = window.setInterval(() => this.pollAttention(), 30000);
    this.addEventListener('soda-project-observed', (e) => this.onProjectObserved(e));
    this.addEventListener('soda-project-operation', (e) => this.onProjectOperation(e));
    this.addEventListener('soda-intent-operation', (event) => {
      if (this.pendingWorkspaceRoute && event.target === this.querySelector('soda-intent')) this.requestUpdate();
    });
    this.addEventListener('soda-project-change-repository', (e) => this.onProjectChangeRepository(e));
    this.addEventListener('soda-project-feed-open', (event) => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed || !this.activeSurface) return;
      if (this.view !== 'app' || event.detail?.repository !== this.project) return;
      this.showProjectView('feed');
    });
    this.addEventListener('soda-project-work-open', (event) => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed || !this.activeSurface) return;
      if (this.view !== 'app' || event.detail?.repository !== this.project) return;
      this.showProjectView('work');
    });
    this.addEventListener('soda-application-open', (event) => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed || !this.activeSurface) return;
      const repository = event.detail?.repository;
      if (typeof repository !== 'string' || !id(repository)) return;
      const space = this.spaces.find((space) => space.environment.repository_id === repository);
      if (!space) return;
      this.project = repository;
      this.deliveryRepository = repository;
      this.deliveryTarget =
        typeof event.detail?.deliveryId === 'string' && /^[a-f0-9]{32}$/.test(event.detail.deliveryId)
          ? event.detail.deliveryId
          : '';
      // An exact selection leaves the company surface, so a stale scope can
      // never filter the exact view.
      this.companyScopeRepository = '';
      this.rememberProjectScope();
      this.showProjectView('app');
    });
    this.addEventListener('application-delivery-cleared', () => {
      this.deliveryTarget = '';
      this.rememberProjectDelivery();
    });
    this.addEventListener('application-company-opened', () => {
      this.companyScopeRepository = this.project;
      this.rememberProjectScope();
    });
    this.addEventListener('soda-intent-open', (event) => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed || !this.activeSurface) return;
      const {id: intent, repository} = event.detail ?? {};
      if (typeof intent !== 'string' || !requestIDPattern.test(intent) || typeof repository !== 'string') return;
      if (!this.spaces.some((space) => space.environment.repository_id === repository)) return;
      this.onResumeRequest(intent);
    });
    this.addEventListener('soda-project-changed', (e) => {
      if (!(e instanceof CustomEvent)) return;
      void this.refresh();
    });
  }
  present(kind: 'page' | 'native', repositoryId?: string, route?: string) {
    if (!this.binding || this.disposed || (repositoryId !== undefined && !id(repositoryId))) return;
    // Transport/account and each terminal's locator remain unchanged. Only the
    // current content presentation and observed repository preference change.
    const pathname = this.presentedRoute(route);
    const entered = pathname !== this.routePathname || kind !== this.binding.kind;
    this.routePathname = pathname;
    this.binding = workspacePresentation(this.binding, kind, repositoryId);
    const requestedProject = new URLSearchParams(
      (route ? new URL(route, location.href) : new URL(location.href)).search
    ).get('repository');
    if (kind === 'page' && requestedProject && id(requestedProject)) {
      this.project = requestedProject;
      if (isNewRouteLocation(this.routePathname, this.repositoryPrefix())) this.workRepository = requestedProject;
    }
    this.presentView(kind, entered, repositoryId);
    this.armNewRoute(kind, entered);
    this.requestUpdate();
  }
  private presentView(kind: 'page' | 'native', entered: boolean, repositoryId?: string) {
    if (repositoryId) this.project = repositoryId;
    if (kind === 'native') {
      if (['feed', 'app', 'work', 'plans'].includes(this.view)) this.view = 'terminal';
      return;
    }
    if (this.selected) return;
    if (entered && this.view === 'terminal') this.view = 'feed';
    this.restoreProjectView();
  }
  private presentedRoute(route?: string) {
    return route ? route.split(/[?#]/)[0] || this.routePathname : window.location.pathname;
  }
  applyPresentation(presentation: WorkspacePresentation) {
    if (!this.binding || this.disposed) return;
    this.present(presentation.mode === 'page' ? 'page' : 'native', presentation.repositoryId, presentation.route);
    if (presentation.visible !== this.surfaceVisible) this.setVisible(presentation.visible);
    if (presentation.visible) this.markViewed();
  }
  private armNewRoute(kind: 'page' | 'native', entered: boolean) {
    // Only a confirmed host entry can return setup to the retained Workspace.
    // Repeated base updates while a requested /new entry is pending cannot.
    const workspace = kind === 'page' && this.routePathname === `${this.repositoryPrefix()}/workspace`;
    if (!workspace) this.pendingWorkspaceRoute = false;
    else if (entered) this.pendingWorkspaceRoute = true;
    if (kind === 'page' && isNewRouteLocation(this.routePathname, this.repositoryPrefix())) this.pendingNewRoute = true;
  }
  private pollAttention() {
    if (this.stale || this.disposed) return;
    this.now = Date.now();
    this.requestUpdate();
    if (this.shouldRefreshAttention()) void this.refresh();
  }
  private shouldRefreshAttention() {
    if (!this.activeSurface) return false;
    return this.now - this.observedAt >= 60000;
  }
  private projectEventFromHost(e: Event) {
    return this.projects.get(this.project)?.host.firstElementChild === e.target;
  }
  private pageProjectEvent(e: Event) {
    return e instanceof CustomEvent && !this.stale && !this.disposed && this.binding?.kind === 'page';
  }
  private onProjectObserved(e: Event) {
    if (!this.pageProjectEvent(e)) return;
    const detail = object((e as CustomEvent).detail);
    if (detail.repositoryId !== this.project || !this.projectEventFromHost(e)) return;
    if (this.setup === 'configure' && projectId(detail.environmentId)) void this.refresh();
  }
  private onProjectOperation(e: Event) {
    if (!this.disposed && this.projectEventFromHost(e)) this.requestUpdate();
  }
  private onProjectChangeRepository(e: Event) {
    if (this.projectEventFromHost(e)) inspectCommittedRequest(this.setupReading(), this.setupActions(), this);
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  protected updated() {
    this.activateNewRoute();
    this.display();
  }
  private activateNewRoute() {
    if (this.pendingWorkspaceRoute) {
      this.returnToWorkspace();
      return;
    }
    // One-shot fresh entry for direct new-route load/traversal/forward in a
    // populated workspace. The flag clears on any setup so Back/cancel never
    // refires. The current route is rechecked so a stale flag left by
    // native navigation can never activate.
    if (this.setup) {
      this.freshEntryFromSetup();
      this.pendingNewRoute = false;
      return;
    }
    if (!this.pendingNewRoute) return;
    const armed = this.newRouteCurrent() && this.spaces.length > 0;
    if (armed) beginSetup(this.setupReading(), this.setupActions(), this);
    if (this.setup || !isNewRouteLocation(this.routePathname, this.repositoryPrefix())) this.pendingNewRoute = false;
  }
  private returnToWorkspace() {
    if (this.stale || !this.canRestore || this.querySelector<SodaIntent>('soda-intent')?.busy) return;
    this.pendingWorkspaceRoute = this.pendingNewRoute = false;
    // The host already entered the base route. The ordinary guarded cancel
    // restores the selected view; its route request is therefore a no-op.
    if (this.setup || this.setupReturn) cancelSetup(this.setupReading(), this.setupActions(), this);
  }
  private newRouteCurrent() {
    return this.binding?.kind === 'page' && isNewRouteLocation(this.routePathname, this.repositoryPrefix());
  }
  private freshEntryFromSetup() {
    // Genuine native New arrival while a setup is active: the location no
    // longer names the reviewed request, so drop that loaded identity and
    // show a blank entry. Fresh in-progress entries (no revision) and busy
    // elements are left untouched; stale owners and in-flight project
    // mutations keep the screen, exactly like Back/cancel. The setup return
    // target is preserved for Back.
    if (!this.pendingNewRoute) return;
    if (this.stale || !this.canRestore || !this.newRouteCurrent()) return;
    const intent = this.querySelector<SodaIntent>('soda-intent');
    if (!intent || intent.busy || intent.revision === undefined) return;
    intent.reset();
    const actions = this.setupActions();
    actions.setRepositoryChoice('');
    actions.setReviewAuthorization(undefined);
    actions.setReviewedIntent(undefined);
    actions.setSetupCommitted(false);
    actions.setSetup('intent');
  }
  private geometryChanged(width: number, height: number, cell: {width: number; height: number}) {
    if (this.workspaceWidth !== this.clientWidth) return true;
    if (width !== this.canvasSize.width || height !== this.canvasSize.height) return true;
    return cell.width !== this.cell.width || cell.height !== this.cell.height;
  }
  private applyGeometry(width: number, height: number, cell: {width: number; height: number}) {
    this.workspaceWidth = this.clientWidth;
    this.canvasSize = {width, height};
    this.cell = cell;
    this.requestUpdate();
  }
  private publishMinimum() {
    const min = paneMinimum(this.minimumInput(), focusedPane(this.layout)).width;
    if (min === this.lastMinimum) return;
    this.lastMinimum = min;
    this.dispatchEvent(new CustomEvent('soda-workspace-minimum', {bubbles: true, detail: min}));
  }
  private measuredCell(cells: DOMRect | undefined) {
    if (cells?.width && cells.height) return {width: cells.width / 16, height: cells.height};
    return this.cell;
  }
  private recordCanvasGeometry() {
    const canvas = this.querySelector<HTMLElement>('.soda-workspace-canvas'),
      cells = this.querySelector('.soda-cell-measure')?.getBoundingClientRect();
    if (!this.clientWidth) return;
    const width = canvas?.clientWidth || this.canvasSize.width,
      height = canvas?.clientHeight || this.canvasSize.height;
    const cell = this.measuredCell(cells);
    if (this.geometryChanged(width, height, cell)) this.applyGeometry(width, height, cell);
  }
  private measure = () => {
    if (this.disposed || this.stale) return;
    this.recordCanvasGeometry();
    this.publishMinimum();
  };
  private readonly onWorkspaceKey = (e: KeyboardEvent) =>
    workspaceKey(
      {
        isEditing: () => !!this.editing,
        clearEditing: () => {
          this.editing = null;
        },
        restoreFocus: () => this.restoreFocus(),
      },
      e
    );
  private readonly onWorkspaceClick = (e: MouseEvent) => workspaceClick(this, e);
  private readonly onNavKey = (e: KeyboardEvent) => this.navigationEscape(e);
  private readonly onSearchInput = (e: Event) => this.setSearchFromEvent(e);
  private readonly onThisPageChange = (e: Event) => this.setThisPageFromEvent(e);
  private readonly onShowAllAttention = () => {
    this.attentionOnly = false;
  };
  private readonly onShowAttentionOnly = () => {
    this.attentionOnly = true;
  };
  private readonly onRefreshClick = () => this.refresh();
  private readonly onBeginSetup = () => {
    this.workRepository = undefined;
    beginSetup(this.setupReading(), this.setupActions(), this);
  };
  private readonly onResumeRequest = (intent: string) => {
    this.workRepository = undefined;
    resumeSetup(intent, this.setupReading(), this.setupActions(), this);
  };
  private readonly onNewWork = () => {
    this.workRepository = this.view === 'plans' ? undefined : this.project;
    beginSetup(this.setupReading(), this.setupActions(), this);
  };
  private readonly onWorkResume = (event: CustomEvent<{intent: string}>) => this.onResumeRequest(event.detail.intent);
  private readonly onWorkAuthorityLost = (event: CustomEvent<{reconnectRequired: boolean}>) => {
    this.reconnectRequired = event.detail.reconnectRequired;
    this.invalidate();
  };
  private readonly onShowWork = () => {
    this.showProjectView('work');
  };
  private readonly onShowFeed = () => this.showProjectView('feed');
  private readonly onShowApp = () => this.showProjectView('app');
  private showProjectView(view: 'feed' | 'app' | 'work') {
    if (this.stale || !this.actor || !this.selectedSpace) return;
    if (view === 'app') this.applicationOwner = `${this.actor.id}:${this.project}`;
    this.view = view;
    try {
      window.localStorage.setItem(`soda-project-view:v1:${this.actor.id}:${this.project}`, view);
      this.rememberProjectDelivery();
    } catch {
      /* The current view remains usable without browser storage. */
    }
  }
  private rememberProjectDelivery() {
    if (!this.actor || !id(this.project)) return;
    const key = `soda-project-delivery:v1:${this.actor.id}:${this.project}`;
    try {
      if (this.deliveryRepository === this.project && this.deliveryTarget)
        window.localStorage.setItem(key, this.deliveryTarget);
      else window.localStorage.removeItem(key);
    } catch {
      /* Navigation remains usable without browser storage. */
    }
  }
  private rememberProjectScope() {
    if (!this.actor || !id(this.project)) return;
    const key = `soda-project-scope:v1:${this.actor.id}:${this.project}`;
    try {
      if (this.companyScopeRepository === this.project) window.localStorage.setItem(key, 'company');
      else window.localStorage.removeItem(key);
    } catch {
      /* Navigation remains usable without browser storage. */
    }
  }
  private restoreProjectView() {
    if (this.binding?.kind !== 'page' || !this.actor || !id(this.project) || this.setup || this.view === 'project')
      return;
    const key = `soda-project-view:v1:${this.actor.id}:${this.project}`;
    if (this.restoredProjectView === key) return;
    this.restoredProjectView = key;
    this.applicationOwner = '';
    try {
      const saved = window.localStorage.getItem(key);
      this.view = saved === 'app' || saved === 'work' ? saved : 'feed';
      const delivery = window.localStorage.getItem(`soda-project-delivery:v1:${this.actor.id}:${this.project}`) ?? '';
      this.deliveryTarget = /^[a-f0-9]{32}$/.test(delivery) ? delivery : '';
      this.deliveryRepository = this.project;
      // A retained company selection restores the scope it was opened over;
      // an exact selection always wins, and another project never inherits it.
      const scope = window.localStorage.getItem(`soda-project-scope:v1:${this.actor.id}:${this.project}`);
      this.companyScopeRepository = !this.deliveryTarget && scope === 'company' ? this.project : '';
    } catch {
      this.view = 'feed';
    }
    if (this.view === 'app') this.applicationOwner = `${this.actor.id}:${this.project}`;
  }
  private readonly onShowPlans = () => {
    this.view = 'plans';
  };
  private readonly onShowTerminals = () => {
    this.view = 'terminal';
    const space = this.selectedSpace;
    if (space && !this.projectRunnable(space)) void this.showManagement(this.project, 'journey');
    else void restoreLocators(this.terminalsInput(), this.epoch, this.lifetime.signal);
    this.markViewed();
  };
  private renderDiscoveryRequests() {
    // Discovery groups exist only when navigation is visible: while setup is
    // open the composer owns the saved list, and hidden duplicates would steal
    // text matches from it.
    if (
      this.binding?.kind === 'page' ||
      hideNavigation(this.setupScreen, this.view, this.compact, this.layout.sidebar, this.stale)
    )
      return '';
    return renderRequestGroups({groups: groupSavedRequests(this.savedRequests), resume: this.onResumeRequest});
  }
  private readonly onShowSessions = () => this.showSessions();
  private readonly onNewTerminal = () => newTerminal(this.newTerminalInput(), this.layout.focused);
  private readonly onCancelSetup = () => cancelSetup(this.setupReading(), this.setupActions(), this);
  private readonly onSetupBack = () => {
    if (this.setup === 'configure') inspectCommittedRequest(this.setupReading(), this.setupActions(), this);
    else if (this.setup === 'repositories') backToReview(this.setupReading(), this.setupActions(), this);
    else cancelSetup(this.setupReading(), this.setupActions(), this);
  };
  private readonly onCapturePointer = (e: PointerEvent) => this.capture(e);
  private readonly onResizeSidebar = (e: PointerEvent) => resizeSidebar(this.layoutInput(), e);
  private readonly onReleasePointer = (e: PointerEvent) => this.releasePointer(e);
  private readonly onSidebarKey = (e: KeyboardEvent) => sidebarKey(this.layoutInput(), e);
  private navigationEscape(e: KeyboardEvent) {
    if (e.key !== 'Escape' || this.view !== 'sessions') return;
    e.preventDefault();
    e.stopPropagation();
    this.back();
  }
  private setSearchFromEvent(e: Event) {
    if (e.target instanceof HTMLInputElement) this.search = e.target.value;
  }
  private setThisPageFromEvent(e: Event) {
    if (e.target instanceof HTMLInputElement) this.thisPage = e.target.checked;
  }
  private renderNativeToolbar() {
    if (this.binding?.kind !== 'native') return '';
    return this.renderToolbar();
  }
  private renderPageToolbar() {
    if (this.binding?.kind !== 'page') return '';
    return this.renderToolbar();
  }
  private renderDrawerProjectionTabs() {
    if (this.binding?.kind !== 'native' || this.view === 'terminal' || this.stale) return '';
    return html`<div class="soda-drawer-projection-tabs" style=${`height:${this.tabHeight}px`}>
      ${paneChrome(this.paneViewInput(), focusedPane(this.layout), {x: 0, y: 0, width: this.workspaceWidth, height: this.tabHeight}, true)}
    </div>`;
  }
  private navAriaLabel() {
    return this.binding?.kind === 'page' ? 'Projects' : 'Projects and sessions';
  }
  private renderProjectsHeading() {
    if (this.binding?.kind !== 'page') return '';
    const title = html`<h2>Projects</h2>`;
    return html`<div class="soda-projects-heading">
      ${title}<span class="soda-project-count" aria-label=${this.spaces.length + ' visible projects'}
        >${this.spaces.length}</span
      >
    </div>`;
  }
  private hideTerminalSearch() {
    return this.binding?.kind === 'page' && this.spaces.length < 2 && !this.spaces.some((s) => s.terminals.length);
  }
  private renderThisPageFilter() {
    if (this.binding?.kind !== 'native' || !this.binding.pageRepositoryId) return '';
    return html`<label
      ><input type="checkbox" .checked=${this.thisPage} @change=${this.onThisPageChange} /> This page only</label
    >`;
  }
  private hideAttentionFilters() {
    return this.binding?.kind === 'page';
  }
  private renderCreateAnotherProject() {
    if (this.binding?.kind !== 'page') return '';
    return html`<button
      class="ui button soda-create-another-project"
      ?disabled=${workspaceBlocked(this.stale, this.available) || !this.canRestore}
      @click=${this.onBeginSetup}
    >
      <span aria-hidden="true">＋</span> New project
    </button>`;
  }
  private emptyFilterMessage() {
    if (!this.available) return 'Status unavailable.';
    if (this.attentionOnly) return 'No matching sessions need attention.';
    if (this.search || this.thisPage) return 'No matches.';
    return 'No authorized projects available.';
  }
  private renderEmptySpaces() {
    if (this.navigationSpaces().length) return '';
    return html`<p>${this.emptyFilterMessage()}</p>`;
  }
  private renderWelcomeFooter() {
    if (!this.welcomeScreen && !this.setup) return '';
    const step =
      this.setup === 'configure' ? 3 : this.setup === 'repositories' ? 2 : this.setup === 'intent' ? 1 : undefined;
    return renderWelcomeSteps(step);
  }
  private onRenameDraft(name: string) {
    if (this.editing) this.editing = {...this.editing, name};
  }
  private onRenameCancel() {
    this.editing = null;
    this.restoreFocus();
  }
  private onRenameSave() {
    const edit = this.editing,
      slot = this.slots.find((s) => s.key === edit?.key);
    if (slot && edit) void rename(this.renameInput(), slot, edit.name);
  }
  private renameInput(): RenameInput {
    return {
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      isActiveSurface: () => this.activeSurface,
      readEpoch: () => this.epoch,
      readEditing: () => this.editing,
      setEditing: (edit) => {
        this.editing = edit;
      },
      renaming: this.renaming,
      isValidName: (name) => validName(name),
      locator: (slot) => this.locator(slot),
      api: (path, body, signal) => this.api(path, body, signal),
      lifetimeSignal: this.lifetime.signal,
      live: (n) => this.live(n),
      setStatus: (message) => {
        this.status = message;
      },
      requestUpdate: () => this.requestUpdate(),
    };
  }
  private renameDisabled() {
    if (!this.editing) return true;
    return !validName(this.editing.name) || this.renaming.has(this.editing.key) || this.stale;
  }
  private renderRenameDialog() {
    if (!this.editing) return '';
    const project = this.slots.find((s) => s.key === this.editing?.key)?.binding.projectName || '';
    return renderRename(
      this.editing.name,
      project,
      this.renameDisabled(),
      (name) => this.onRenameDraft(name),
      () => this.onRenameCancel(),
      () => this.onRenameSave()
    );
  }
  private renderCreationDialog() {
    if (!this.creation) return '';
    return creationForm(this.creationInput(), this.creation);
  }
  private renderApplications() {
    // Visiting App creates its owner once; other project views then shelter it
    // without discarding the running app or inspecting an unvisited project.
    if (
      this.binding?.kind !== 'page' ||
      this.stale ||
      !this.actor ||
      this.applicationOwner !== `${this.actor.id}:${this.project}`
    )
      return '';
    return html`<section
      class="soda-project-app"
      aria-label="Project application"
      ?hidden=${this.view !== 'app' || this.stale || this.setupScreen}
    >
      <soda-applications
        .expanded=${true}
        .request=${this.requestApplication}
        .active=${this.activeSurface && !this.stale && !this.disposed}
        .sessionKey=${this.binding.transport.generation}
        .repository=${this.project}
        .requestedDelivery=${this.deliveryRepository === this.project ? this.deliveryTarget : ''}
        .requestedCompanyScope=${this.companyScopeRepository === this.project && !this.deliveryTarget}
      ></soda-applications>
    </section>`;
  }
  private renderNavigation() {
    return html`<nav
      class="soda-workspace-navigation"
      aria-label=${this.navAriaLabel()}
      ?hidden=${hideNavigation(this.setupScreen, this.view, this.compact, this.layout.sidebar, this.stale)}
      @keydown=${this.onNavKey}
    >
      ${this.renderProjectsHeading()}
      <label ?hidden=${this.hideTerminalSearch()}
        >${this.binding?.kind === 'page' ? 'Find a project' : 'Find a terminal'}
        <input type="search" .value=${this.search} @input=${this.onSearchInput}
      /></label>
      ${this.renderThisPageFilter()}
      <div class="soda-attention-filters" aria-label="Session attention" ?hidden=${this.hideAttentionFilters()}>
        <button
          class="ui button"
          aria-pressed=${this.attentionOnly ? 'false' : 'true'}
          @click=${this.onShowAllAttention}
        >
          All
        </button>
        <button
          class="ui button"
          aria-pressed=${this.attentionOnly ? 'true' : 'false'}
          @click=${this.onShowAttentionOnly}
        >
          Attention (${attentionRows(this.navReading(), this.spaces, this.layout.entries).length})
        </button>
        <button class="ui button" @click=${() => this.nextAttention()}>Next attention</button>
      </div>
      ${this.renderProjectNavigation()} ${this.renderDiscoveryRequests()}
      ${renderMoreProjects(this.moreProjectsInput())} ${this.renderCreateAnotherProject()} ${this.renderEmptySpaces()}
      ${this.binding?.kind === 'page' ? html`<button class="ui button soda-saved-plans" aria-pressed=${this.view === 'plans'} @click=${this.onShowPlans}>Saved plans</button>` : ''}
    </nav>`;
  }
  private requestApplication = (path: string, body?: Record<string, unknown>, signal?: AbortSignal) =>
    this.api(path, body, signal);
  private renderProjectNavigation() {
    if (this.binding?.kind === 'page')
      return renderProjectDirectory({
        reading: this.navReading(),
        spaces: this.navigationSpaces(),
        entries: this.layout.entries,
        project: this.view === 'plans' ? '' : this.project,
        projectName: (space) => this.projectName(space),
        projectState: (space) => this.projectState(space),
        selectProject: (space) => this.selectProject(space),
      });
    return repeat(
      this.navigationSpaces(),
      (space) => space.environment.id,
      (space) => projectRows(this.projectRowsInput(space))
    );
  }
  private projectRowsInput(space: Space) {
    return {
      reading: this.navReading(),
      space,
      search: this.binding?.kind === 'page' ? '' : this.search,
      attentionOnly: this.binding?.kind === 'native' && this.attentionOnly,
      selected: this.selected,
      entries: this.layout.entries,
      tree: this.layout.tree,
      kind: this.binding?.kind,
      project: this.project,
      state: this.projectState(space),
      status: this.projectStatus(space),
      projectName: this.projectName(space),
      factory: '',
      selectRow: (target: Space, row: Row) => this.selectRow(target, row),
      showManagement: (repository: string) => this.showManagement(repository),
      selectProject: (target: Space) => this.selectProject(target),
    };
  }
  private renderProjectViews() {
    if (this.binding?.kind !== 'page' || this.view === 'plans' || !this.selectedSpace || this.setupScreen) return '';
    return html`<nav class="soda-project-views" aria-label="Project views">
      <button class="ui button" aria-pressed=${this.view === 'feed'} @click=${this.onShowFeed}>Feed</button>
      <button class="ui button" aria-pressed=${this.view === 'app'} @click=${this.onShowApp}>App</button>
      <button class="ui button" aria-pressed=${this.view === 'work'} @click=${this.onShowWork}>Work</button>
      <button class="ui button" aria-pressed=${this.view === 'terminal'} @click=${this.onShowTerminals}>
        Terminals
      </button>
    </nav>`;
  }
  private projectWorkContext() {
    return this.binding && this.actor && !this.stale
      ? {
          transport: this.binding.transport,
          mode: this.view === 'plans' ? ('all-plans' as const) : ('project' as const),
          ...(this.selectedSpace ? {repository: {id: this.project, name: this.projectName(this.selectedSpace)}} : {}),
          repositoryNames: new Map(
            this.spaces.map((space) => [space.environment.repository_id, this.projectName(space)])
          ),
          account: this.actor.id,
          verificationRuns:
            this.selectedSpace?.factory_runs
              .filter((run) => run.reconciled && run.issue === undefined)
              .map((run) => run.id) ?? [],
        }
      : undefined;
  }
  private renderProjectWork() {
    const active =
      this.binding?.kind === 'page' && !this.setupScreen && ['feed', 'app', 'work', 'plans'].includes(this.view);
    return html`<soda-project-work
      .context=${this.projectWorkContext()}
      .active=${active && this.activeSurface}
      .surface=${this.view === 'feed' ? 'feed' : 'work'}
      ?hidden=${!active || this.view === 'app' || this.stale}
      @soda-work-new=${this.onNewWork}
      @soda-work-resume=${this.onWorkResume}
      @soda-work-authority-lost=${this.onWorkAuthorityLost}
    ></soda-project-work>`;
  }
  private renderFactoryActivity() {
    // Keep every watch owner connected across project, search and surface changes.
    return repeat(
      this.spaces,
      (space) => space.environment.id,
      (space) => html`<section
        class="soda-project-activity"
        aria-label=${'Activity in ' + this.projectName(space)}
        ?hidden=${!this.factoryActivityVisible(space)}
      >
        ${factorySection({
          readWatches: () => this.watches,
          isStale: () => this.stale,
          pushWatch: (watch) => {
            this.watches.push(watch);
          },
          replaceWatches: (watches) => {
            this.watches = watches;
          },
          updated: () => this.requestUpdate(),
          available: this.available,
          space,
          query: '',
        })}
      </section>`
    );
  }
  private factoryActivityVisible(space: Space) {
    if (this.stale || this.setupScreen) return false;
    if (this.binding?.kind === 'native') return this.view === 'sessions';
    return this.view === 'work' && space.environment.repository_id === this.project;
  }
  private renderTerminalDirectory() {
    if (this.binding?.kind !== 'page' || this.view !== 'terminal' || !this.selectedSpace || this.stale) return '';
    return renderProjectTerminals(this.projectRowsInput(this.selectedSpace));
  }
  private renderCanvas() {
    const recovery = renderInventoryRecovery(
      this.inventoryRecovery,
      this.busy,
      this.onRefreshClick,
      this.spaces,
      this.connectURL
    );
    const intro = renderFirstTerminalIntro(
      this.firstTerminal,
      this.selectedSpace,
      this.selectedSpace ? this.projectName(this.selectedSpace) : '',
      defaultTerminalName(this.newTerminalInput(), this.selectedSpace),
      workspaceBlocked(this.stale, this.available),
      this.creating,
      this.onNewTerminal
    );
    return html`<div class="soda-workspace-canvas" ?hidden=${hideCanvas(this.view, this.stale)}>
      ${recovery} ${intro}
      <div class="soda-workspace-chrome" ?hidden=${hidePaneChrome(this.firstTerminal, this.inventoryRecovery)}>
        ${repeat(
          this.projection.panes,
          (area) => area.pane.key,
          (area) => paneChrome(this.paneViewInput(), area.pane, area)
        )}
        ${repeat(
          this.projection.dividers,
          (divider) => divider.key,
          (divider) => renderPaneDivider(this.layoutInput(), divider)
        )}
      </div>
      <div class="soda-workspace-owners"></div>
    </div>`;
  }
  private renderWorkspaceFrame() {
    const topbar = renderSetupTopbar(this.setup, this.stale, this.canRestore, this.onCancelSetup, this.onSetupBack);
    const bodyHidden = hideWorkspaceBody(this.setupScreen, this.setup);
    const bodyClass = workspaceBodyClass(this.view);
    const bodyStyle = workspaceBodyStyle(this.setupScreen, this.compact, this.layout.sidebar, this.view);
    return html`<div class="soda-workspace-frame">
        ${topbar}
        ${this.setupScreen ? renderSetup(this.setupReading(), this.setupActions(), this, () => this.setupReading()) : ''}
        <div ?hidden=${bodyHidden} class=${bodyClass} style=${bodyStyle}>
          ${this.renderNavigation()}
          <div
            class="soda-sidebar-divider"
            role="separator"
            tabindex="0"
            aria-label="Resize project sidebar"
            aria-orientation="vertical"
            aria-valuemin="220"
            aria-valuemax="360"
            aria-valuenow=${this.layout.sidebar || 256}
            ?hidden=${hideSidebarDivider(this.setupScreen, this.compact, this.view, this.layout.sidebar)}
            @pointerdown=${this.onCapturePointer}
            @pointermove=${this.onResizeSidebar}
            @pointerup=${this.onReleasePointer}
            @keydown=${this.onSidebarKey}
          ></div>
          <div class="soda-workspace-work">
            ${this.renderPageToolbar()}
            <div
              class="soda-project-content"
              ?hidden=${!['feed', 'app', 'work', 'plans'].includes(this.view) && !(this.binding?.kind === 'native' && this.view === 'sessions')}
            >
              ${this.renderProjectWork()} ${this.renderApplications()} ${this.renderFactoryActivity()}
            </div>
            ${this.renderCanvas()}
            <div class="soda-workspace-management" ?hidden=${hideManagement(this.view, this.stale)}></div>
          </div>
        </div>
        ${this.renderWelcomeFooter()}
      </div>
      ${this.setupScreen && this.intentNotice ? html`<p class="soda-feedback" role="status">${this.intentNotice}</p>` : ''}`;
  }
  protected render() {
    const heading = renderSetupHeading(this.setupScreen && !this.welcomeScreen && this.setup !== 'intent');
    const banners = renderStatusBanners(
      this.status,
      this.setupScreen,
      this.setup,
      this.storageNotice,
      this.complete,
      this.stale,
      this.busy,
      this.onRefreshClick
    );
    return html`<section
      id="sodaspaces-data"
      data-workspace-kind=${this.binding?.kind || ''}
      class=${workspaceClasses(this.compact, this.setupScreen, this.welcomeScreen, this.setup, this.workspaceIntro)}
      aria-busy=${busyAttr(this.busy)}
      @keydown=${this.onWorkspaceKey}
      @click=${this.onWorkspaceClick}
    >
      <span class="soda-cell-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span>
      ${heading} ${this.renderNativeToolbar()} ${banners} ${this.renderDrawerProjectionTabs()}
      ${this.renderWorkspaceFrame()} ${this.renderCreationDialog()} ${this.renderRenameDialog()}
    </section>`;
  }
  private get connectURL() {
    return window.location.href;
  }
  private renderToolbarMenu() {
    if (this.workspaceIntro) return '';
    const openInDrawerOption = renderOpenInDrawerOption(
      this.binding?.kind,
      workspaceBlocked(this.stale, this.available),
      this.openingDrawer,
      this.selected,
      () => openInDrawer(this.drawerInput())
    );
    const toggleSidebarOption = renderToggleSidebarOption(this.binding?.kind, () => toggleSidebar(this.layoutInput()));
    const nativeMgmt = renderNativeManagementOption(
      this.binding,
      workspaceBlocked(this.stale, this.available),
      (repository) => this.showManagement(repository)
    );
    return renderMenu(
      'Workspace options',
      '⋯',
      html`
        <button class="ui button" ?disabled=${this.busy || this.stale} @click=${this.onRefreshClick}>
          Refresh Spaces
        </button>
        ${openInDrawerOption} ${toggleSidebarOption} ${nativeMgmt} ${this.renderTerminalDirectory()}
      `
    );
  }
  private renderToolbarTitle() {
    const kind = this.binding?.kind;
    const space = this.selectedSpace;
    return this.view === 'plans'
      ? html`<div class="soda-selected-project"><h1>Saved plans</h1></div>`
      : renderToolbarProject(
          kind,
          space,
          space ? this.projectState(space) : '',
          space ? this.projectStatus(space) : '',
          space ? this.projectName(space) : ''
        );
  }
  private renderToolbar() {
    const kind = this.binding?.kind;
    const space = this.selectedSpace;
    const settings =
      this.view === 'plans'
        ? ''
        : renderProjectSettingsButton(kind, workspaceBlocked(this.stale, this.available), space, () =>
            this.showManagement(this.project)
          );
    const back = renderBackButton(this.view, kind, this.managementMode, () => this.back());
    return html`<header class="soda-workspace-toolbar" ?hidden=${this.setupScreen}>
      <button
        class="ui button"
        data-workspace="sessions"
        ?hidden=${sessionsButtonHidden(kind, this.compact, this.layout.sidebar, this.view)}
        ?disabled=${workspaceBlocked(this.stale, this.available)}
        @click=${this.onShowSessions}
      >
        ${sessionsButtonLabel(kind)}
      </button>
      ${this.renderToolbarTitle()} ${this.renderProjectViews()}
      <button
        class=${newTerminalButtonClass(kind)}
        data-environment-id=${this.selectedSpace?.environment.id || ''}
        data-terminal-name=${defaultTerminalName(this.newTerminalInput(), this.selectedSpace)}
        aria-label="New terminal"
        title="New terminal"
        ?hidden=${(kind === 'page' && this.view !== 'terminal') || hideNewTerminal(this.firstTerminal, kind, space)}
        ?disabled=${disableNewTerminal(workspaceBlocked(this.stale, this.available), this.creating, kind, space)}
        @click=${this.onNewTerminal}
      >
        ${newTerminalButtonLabel(kind)}
      </button>
      ${settings} ${back} ${this.renderToolbarMenu()} ${renderSpacesLink()}
    </header>`;
  }
  private repositoryPrefix() {
    return this.binding?.forgejoPrefix || '';
  }
  private setupReading(): SetupReading {
    return {
      setup: this.setup,
      welcomeScreen: this.welcomeScreen,
      busy: this.busy,
      reconnectRequired: this.reconnectRequired,
      stale: this.stale,
      connectURL: this.connectURL,
      available: this.available,
      canRestore: this.canRestore,
      activeSurface: this.activeSurface,
      repositoryQuery: this.repositoryQuery,
      repositoryChoice: this.repositoryChoice,
      repositoryResult: this.repositoryResult,
      repositoryError: this.repositoryError,
      repositoryBusy: this.repositoryBusy,
      forgejoPrefix: this.repositoryPrefix(),
      project: this.project,
      view: this.view,
      managementMode: this.managementMode,
      page: this.binding?.kind === 'page',
      hasSelection: this.selected !== '',
      setupReturn: this.setupReturn,
      firstSpace: this.spaces[0],
      actorId: this.actor?.id || '',
      requestId: requestIdFromLocation(this.routePathname, this.repositoryPrefix()),
      reviewAuthorization: this.reviewAuthorization,
      reviewedIntent: this.reviewedIntent,
      setupCommitted: this.setupCommitted,
      transport: this.binding?.transport,
      suggestedRepository: this.suggestedSetupRepository(),
      overviewScope: this.overviewScope,
      committedSpace: this.spaces.find((space) => space.environment.repository_id === this.repositoryChoice),
      onBegin: this.onBeginSetup,
      onRefresh: this.onRefreshClick,
    };
  }
  private suggestedSetupRepository(): string | undefined {
    const requestId = requestIdFromLocation(this.routePathname, this.repositoryPrefix());
    if (!requestId) return this.workRepository;
    const saved = this.savedRequests?.find((request) => request.intent === requestId)?.repository;
    return saved && saved !== '0' ? saved : this.workRepository;
  }
  private setupActions(): SetupActions {
    return {
      setSetup: (setup) => {
        if (this.setup && !setup) void this.querySelector<SodaProjectWork>('soda-project-work')?.refresh();
        this.setup = setup;
      },
      setRepositoryQuery: (query) => {
        this.repositoryQuery = query;
      },
      setRepositoryChoice: (choice) => {
        this.repositoryChoice = choice;
      },
      setRepositoryResult: (result) => {
        this.repositoryResult = result;
      },
      setRepositoryCursors: (cursors) => {
        this.repositoryCursors = cursors;
      },
      setRepositoryError: (error) => {
        this.repositoryError = error;
      },
      setRepositoryBusy: (busy) => {
        this.repositoryBusy = busy;
      },
      setSetupReturn: (target) => {
        this.setupReturn = target;
      },
      setReviewAuthorization: (auth) => {
        this.reviewAuthorization = auth;
      },
      setReviewedIntent: (intent) => {
        this.reviewedIntent = intent;
      },
      setSetupCommitted: (committed) => {
        this.setupCommitted = committed;
      },
      setProject: (project) => {
        this.project = project;
      },
      setView: (view) => {
        this.view = view;
      },
      setManagementMode: (mode) => {
        this.managementMode = mode;
      },
      repositoryCursors: () => this.repositoryCursors,
      currentRepositoryRequest: () => this.repositoryRequest,
      admitRepositoryRequest: (request) => {
        this.repositoryRequest = request;
      },
      currentRepositoryQuery: () => this.repositoryQuery,
      hasSelectedSpace: () => !!this.selectedSpace,
      isStale: () => this.stale,
      isActiveSurface: () => this.activeSurface,
      selectProject: (space) => this.selectProject(space),
      rememberFocus: () => this.rememberFocus(),
      restoreFocus: () => this.restoreFocus(),
      showManagement: (repositoryId, mode) => {
        void this.showManagement(repositoryId, mode);
      },
      notifyFactory: (message) => {
        this.intentNotice = message;
      },
      api: (path, body, signal) => this.api(path, body, signal),
      syncIntentRoute: (intent) => {
        this.syncIntentRoute(intent);
      },
      setSavedRequests: (intents) => {
        this.savedRequests = intents;
      },
      setOverviewScope: (scope) => {
        this.overviewScope = scope;
      },
      requestFactoryWatch: (request) => this.requestFactoryWatch(request),
    };
  }
  private requestFactoryWatch(request: FactoryWatchRequest): void {
    const target = resolveWatchTarget(this.spaces, request);
    if (!target) return;
    if (!factoryWatching(this.watches, target.run.id))
      watchRun(
        {
          readWatches: () => this.watches,
          isStale: () => this.stale,
          pushWatch: (watch) => {
            this.watches.push(watch);
          },
          replaceWatches: (watches) => {
            this.watches = watches;
          },
          updated: () => this.requestUpdate(),
        },
        target.space,
        target.run
      );
    const watch = this.watches.find((entry) => entry.run === target.run.id);
    if (!watch) return;
    watch.hidden = false;
    if (!this.navigationShows(target.space)) {
      this.search = '';
      this.attentionOnly = false;
      this.thisPage = false;
    }
    cancelSetup(this.setupReading(), this.setupActions(), this);
    if (this.stale) return;
    this.project = target.space.environment.repository_id;
    this.view = this.binding?.kind === 'page' ? 'work' : 'sessions';
  }
  private syncIntentRoute(intent: string | null) {
    if (intent !== null) {
      if (!requestIDPattern.test(intent)) return;
      // A save finishing after a confirmed Workspace entry must not redirect
      // that entry back to its saved revision. The normal update then exits.
      if (this.pendingWorkspaceRoute) {
        this.requestUpdate();
        return;
      }
    }
    const target =
      intent === null
        ? this.setup
          ? newRouteFor(this.repositoryPrefix())
          : `${this.repositoryPrefix()}/workspace`
        : requestRouteFor(this.repositoryPrefix(), intent);
    if (target !== this.routePathname) {
      const url = target + window.location.search + window.location.hash;
      this.binding?.navigate?.(url, {replace: intent !== null});
    }
    this.requestUpdate();
  }
  private selectProject(space: Space) {
    if (this.stale || !this.available || !this.canRestore) return;
    this.project = space.environment.repository_id;
    if (this.binding?.kind === 'page') {
      this.restoreProjectView();
      this.choosingPane = undefined;
      this.binding.navigate?.(`${this.repositoryPrefix()}/workspace?repository=${encodeURIComponent(this.project)}`);
      return;
    }
    this.back();
    if (!this.projectRunnable(space)) void this.showManagement(this.project, 'journey');
  }
  markViewed(key?: string) {
    const n = this.epoch;
    void this.updateComplete.then(async () => {
      for (const slot of this.slots.filter((slot) => key === undefined || slot.key === key)) {
        await slot.terminal.ready;
        if (!this.live(n) || !visibleSlot(this.hostsInput(), slot) || !slot.host.querySelector('.is-connected'))
          continue;
        slot.readRequested = false;
        if (slot.unread) {
          slot.unread = false;
          this.requestUpdate();
        }
      }
    });
  }
  private rowAttentionReady(space: Space) {
    return !this.stale && this.available && !!space.login && !space.authority_unavailable && space.execution_allowed;
  }
  private navReading() {
    return {
      stale: this.stale,
      available: this.available,
      slots: this.slots,
      observedAt: this.observedAt,
      now: this.now,
    };
  }
  private navigationSpaces() {
    if (this.binding?.kind === 'page') {
      const query = this.search.toLocaleLowerCase();
      return this.spaces.filter((space) => this.projectName(space).toLocaleLowerCase().includes(query));
    }
    return filteredSpaces({
      reading: this.navReading(),
      spaces: this.spaces,
      entries: this.layout.entries,
      search: this.search,
      attentionOnly: this.attentionOnly,
      thisPage: this.thisPage,
      binding: this.binding,
      projectName: (space) => this.projectName(space),
    });
  }
  private navigationShows(space: Space) {
    return this.navigationSpaces().some((entry) => entry.environment.id === space.environment.id);
  }
  private nextAttention() {
    if (!this.activeSurface || this.stale || !this.available) return;
    const rows = attentionRows(this.navReading(), this.spaces, this.layout.entries),
      index = rows.findIndex(({row}) => row.key === this.selected);
    const next = rows[(index + 1) % rows.length];
    if (!next) {
      this.status = 'No currently authorized sessions need attention.';
      return;
    }
    this.search = '';
    this.thisPage = false;
    if (next.row.entry) void this.openSaved(next.row.entry);
    else if (next.row.metadata) void openExisting(this.createInput(), next.space, next.row.metadata);
  }
  private projectName(space: Space) {
    return space.environment.repository || space.environment.name || space.environment.id;
  }
  private metadataRowName(row: Row) {
    if (row.metadata?.name) return row.metadata.name;
    const proposed = this.slots.find((slot) => slot.key === row.key)?.proposedName;
    if (proposed) return proposed;
    if (row.metadata) return 'Terminal ' + row.metadata.id.slice(0, 8);
    return '';
  }
  private draftRowName(row: Row) {
    if (row.entry?.locator.kind === 'new') return 'Unsent terminal';
    return 'Saved ' + (row.entry?.locator.kind || 'unknown') + ' terminal';
  }
  private selectRow(space: Space, row: Row) {
    this.closeMenus();
    if (row.entry) void this.openSaved(row.entry, this.choosingPane);
    else if (row.metadata) void openExisting(this.createInput(), space, row.metadata, this.choosingPane);
  }
  private paneViewInput(): PaneChromeInput {
    return {
      readLayout: () => this.layout,
      readMaximized: () => this.maximized,
      setMaximized: (key) => {
        this.maximized = key;
      },
      readDragged: () => this.dragged,
      setDragged: (key) => {
        this.dragged = key;
      },
      binding: this.binding,
      spaces: this.spaces,
      slots: this.slots,
      available: this.available,
      stale: this.stale,
      creating: this.creating,
      tabHeight: this.tabHeight,
      projectionCompact: this.projection.compact,
      reading: this.navReading(),
      requestUpdate: () => this.requestUpdate(),
      closeMenus: () => this.closeMenus(),
      arrange: (layout) => arrange(this.layoutInput(), layout),
      focusPane: (key) => focusPane(this.layoutInput(), key),
      canSplit: (axis, key = this.layout.focused) => canSplit(this.layoutInput(), axis, key),
      split: (axis) => split(this.layoutInput(), axis),
      move: (key, destination, before) => move(this.layoutInput(), key, destination, before),
      openSaved: (entry) => {
        void this.openSaved(entry);
      },
      tabKey: (event, key, keys) => this.tabKey(event, key, keys),
      slotName: (slot) => slotName(this.terminalsInput(), slot),
      projectName: (space) => this.projectName(space),
      rectangle: (area) => this.rectangle(area),
      showSessions: (pane) => this.showSessions(pane),
      newTerminal: (pane) => newTerminal(this.newTerminalInput(), pane),
    };
  }
  private creationInput(): CreationFormInput {
    return {
      readCreating: () => this.creating,
      isStale: () => this.stale,
      isAvailable: () => this.available,
      readSpaces: () => this.spaces,
      readCreation: () => this.creation,
      setCreation: (creation) => {
        this.creation = creation;
      },
      projectName: (space) => this.projectName(space),
      isValidName: (name) => validName(name),
      restoreFocus: () => this.restoreFocus(),
      createTerminal: () => {
        void createTerminal(this.createInput());
      },
    };
  }
  private closeMenus() {
    closeWorkspaceMenus(this);
  }
  private rememberFocus() {
    rememberWorkspaceFocus({
      root: this,
      setInvoker: (invoker) => {
        this.invoker = invoker;
      },
    });
  }
  private restoreFocus() {
    restoreWorkspaceFocus({
      invoker: this.invoker,
      updateComplete: this.updateComplete,
      isStale: () => this.stale,
      isActiveSurface: () => this.activeSurface,
      root: this,
    });
  }
  private showSessions(pane?: string) {
    this.rememberFocus();
    this.choosingPane = pane;
    if (pane && this.binding?.kind === 'page') {
      this.onShowTerminals();
      return;
    }
    this.thisPage = false;
    if (this.binding?.kind === 'page' && !this.compact) {
      if (this.layout.sidebar === null) {
        this.layout = {...this.layout, sidebar: 256};
        this.persist();
      }
    } else {
      this.rememberProjectView();
      this.view = 'sessions';
    }
    const active = document.activeElement,
      view = this.view;
    void this.updateComplete.then(() => {
      if (
        !this.stale &&
        this.activeSurface &&
        this.view === view &&
        (document.activeElement === active || document.activeElement === document.body)
      )
        this.querySelector<HTMLElement>(
          '.soda-workspace-navigation label:not([hidden]) input, .soda-workspace-navigation .soda-project-select'
        )?.focus();
    });
  }
  private back() {
    this.view = this.binding?.kind === 'page' ? 'feed' : 'terminal';
    this.choosingPane = undefined;
    this.restoreFocus();
    this.markViewed();
  }
  private newTerminalInput(): NewTerminalInput {
    return {
      binding: this.binding,
      isStale: () => this.stale,
      readCreating: () => this.creating,
      isAvailable: () => this.available,
      isActiveSurface: () => this.activeSurface,
      readSurfaceVisible: () => this.surfaceVisible,
      readSelectedSpace: () => this.selectedSpace,
      readSpaces: () => this.spaces,
      readSlots: () => this.slots,
      readSelected: () => this.selected,
      readLayout: () => this.layout,
      readCreation: () => this.creation,
      setCreation: (creation) => {
        this.creation = creation;
      },
      readUpdateComplete: () => this.updateComplete,
      rememberFocus: () => this.rememberFocus(),
      focusCreationSelect: () => {
        this.querySelector<HTMLElement>('.soda-workspace-dialog select')?.focus();
      },
      createTerminal: () => {
        void createTerminal(this.createInput());
      },
    };
  }
  private moreProjectsInput(): MoreProjectsInput {
    return {
      nextAfter: this.nextAfter,
      stale: this.stale,
      busy: this.busy,
      more: () => {
        void loadMore(this.inventoryInput());
      },
    };
  }
  private inventoryInput(): InventoryInput {
    return {
      binding: this.binding,
      lifetimeSignal: this.lifetime.signal,
      isDisposed: () => this.disposed,
      isStale: () => this.stale,
      readEpoch: () => this.epoch,
      readBusy: () => this.busy,
      readRequest: () => this.request,
      readSpaces: () => this.spaces,
      readComplete: () => this.complete,
      readNextAfter: () => this.nextAfter,
      readSetup: () => this.setup,
      readIntentRequest: () => requestIdFromLocation(this.routePathname, this.repositoryPrefix()),
      readSelectedSpace: () => this.selectedSpace,
      readProject: () => this.project,
      readObservedAt: () => this.observedAt,
      readSlots: () => this.slots,
      readRestored: () => this.restored,
      readSurfaceVisible: () => this.surfaceVisible,
      readActor: () => this.actor,
      readView: () => this.view,
      readManagementMode: () => this.managementMode,
      readStorageLoaded: () => this.storageLoaded,
      readUpdateComplete: () => this.updateComplete,
      concealed: () => !this.activeSurface,
      setBusy: (busy) => {
        this.busy = busy;
      },
      setRequest: (request) => {
        this.request = request;
      },
      setSpaces: (spaces) => {
        this.spaces = spaces;
      },
      setComplete: (complete) => {
        this.complete = complete;
      },
      setNextAfter: (nextAfter) => {
        this.nextAfter = nextAfter;
      },
      setAvailable: (available) => {
        this.available = available;
      },
      setProject: (project) => {
        this.project = project;
        this.restoreProjectView();
      },
      stampNow: () => {
        this.now = this.observedAt = Date.now();
      },
      setStatus: (status) => {
        this.status = status;
      },
      setRestored: (restored) => {
        this.restored = restored;
      },
      setActor: (actor) => {
        this.actor = actor;
        this.restoreProjectView();
      },
      setStorageKey: (storageKey) => {
        this.storageKey = storageKey;
      },
      setSetup: (setup) => {
        this.setup = setup;
      },
      clearSetupReturn: () => {
        this.setupReturn = undefined;
      },
      setView: (view) => {
        this.view = view;
      },
      setReconnectRequired: (required) => {
        this.reconnectRequired = required;
      },
      invalidate: () => this.invalidate(),
      loadLayout: () => loadLayout(this.layoutInput()),
      restoreLocators: (n, signal) => restoreLocators(this.terminalsInput(), n, signal),
      display: () => this.display(),
      showManagement: (repositoryId, mode) => {
        void this.showManagement(repositoryId, mode);
      },
      projectRunnable: (space) => this.projectRunnable(space),
      locator: (slot) => this.locator(slot),
      confirmedEnd: (key) => confirmedEnd(this.terminalsInput(), key),
      refreshJourneyProject: () => {
        void this.projects.get(this.project)?.api.refresh();
      },
    };
  }
  private live(n: number) {
    return inventoryLive({disposed: this.disposed, stale: this.stale, epoch: this.epoch}, n);
  }
  private async api(path: string, body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    return inventoryApi(
      {
        binding: this.binding,
        disposed: this.disposed,
        stale: this.stale,
        lifetimeSignal: this.lifetime.signal,
        invalidate: () => this.invalidate(),
        setReconnectRequired: (required) => {
          this.reconnectRequired = required;
        },
      },
      path,
      body,
      signal
    );
  }
  async refresh() {
    return inventoryRefresh(this.inventoryInput());
  }
  private persist() {
    return layoutPersist(this.layoutInput());
  }
  private drawerInput(): DrawerInput {
    return {
      binding: this.binding,
      isAvailable: () => this.available,
      isActiveSurface: () => this.activeSurface,
      readOpeningDrawer: () => this.openingDrawer,
      setOpeningDrawer: (opening) => {
        this.openingDrawer = opening;
      },
      readEntries: () => this.layout.entries,
      readSelected: () => this.selected,
      readSpaces: () => this.spaces,
      readEpoch: () => this.epoch,
      live: (n) => this.live(n),
      closeMenus: () => this.closeMenus(),
      setStatus: (status) => {
        this.status = status;
      },
      api: (path, signal) => this.api(path, undefined, signal),
      persist: () => this.persist(),
      repositoryPrefix: () => this.repositoryPrefix(),
      navigate: (path) => {
        if (this.binding?.navigate) this.binding.navigate(path);
        else window.location.assign(path);
      },
    };
  }
  private terminalsInput(): TerminalsInput {
    return {
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      isAvailable: () => this.available,
      isActiveSurface: () => this.activeSurface,
      readEpoch: () => this.epoch,
      readSelected: () => this.selected,
      readSpaces: () => this.spaces,
      readLayout: () => this.layout,
      readSlots: () => this.slots,
      readMaximized: () => this.maximized,
      readProjectionPanes: () => this.projection.panes,
      readUpdateComplete: () => this.updateComplete,
      live: (n) => this.live(n),
      closeMenus: () => this.closeMenus(),
      persist: () => this.persist(),
      display: () => this.display(),
      measure: () => this.measure(),
      markViewed: (key) => this.markViewed(key),
      locator: (slot) => this.locator(slot),
      addSlot: (space, entry) => addSlot(this.hostsInput(), space, entry),
      setSpaces: (spaces) => {
        this.spaces = spaces;
      },
      setLayout: (layout) => {
        this.layout = layout;
      },
      setSlots: (slots) => {
        this.slots = slots;
      },
      setMaximized: (key) => {
        this.maximized = key;
      },
      setStatus: (status) => {
        this.status = status;
      },
      setView: (view) => {
        this.view = view;
      },
      setChoosingPane: (pane) => {
        this.choosingPane = pane;
      },
    };
  }
  private async openSaved(entry: LayoutEntry, destination?: string, focus = true) {
    return openSavedTerminal(this.terminalsInput(), entry, destination, focus);
  }
  private hostsInput(): HostsInput {
    return {
      binding: this.binding,
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      readEpoch: () => this.epoch,
      readView: () => this.view,
      readSetupScreen: () => this.setupScreen,
      readTabHeight: () => this.tabHeight,
      readSurfaceVisible: () => this.surfaceVisible,
      isActiveSurface: () => this.activeSurface,
      readSlots: () => this.slots,
      readLayout: () => this.layout,
      readActor: () => this.actor,
      readProjectionPanes: () => this.projection.panes,
      readUpdateComplete: () => this.updateComplete,
      readFactory: () => this.factory,
      findMountLayer: () => this.querySelector('.soda-workspace-owners'),
      concealed: () => !this.activeSurface,
      live: (n) => this.live(n),
      persist: () => this.persist(),
      requestUpdate: () => this.requestUpdate(),
      display: () => this.display(),
      measure: () => this.measure(),
      invalidate: () => this.invalidate(),
      focusPane: (key) => focusPane(this.layoutInput(), key),
      markViewed: (key) => this.markViewed(key),
      locator: (slot) => this.locator(slot),
      projectName: (space) => this.projectName(space),
      confirmedEnd: (key) => confirmedEnd(this.terminalsInput(), key),
      onTerminalCommand: (key, slot, binding, event) => this.onTerminalCommand(key, slot, binding, event),
      setLayout: (layout) => {
        this.layout = layout;
      },
      setStatus: (status) => {
        this.status = status;
      },
      pushSlot: (slot) => {
        this.slots.push(slot);
      },
      rectangle: (area) => this.rectangle(area),
    };
  }
  private commandAdmitted(slot: Slot, event: Event) {
    return event instanceof CustomEvent && !this.stale && !this.disposed && !slot.host.hidden && !slot.unavailable;
  }
  private beginRename(key: string, slot: Slot) {
    this.editing = {key, name: slot.metadata?.name || ''};
    void this.updateComplete.then(() => {
      if (this.editing?.key === key && !this.stale && this.surfaceVisible)
        this.querySelector<HTMLElement>('.soda-workspace-dialog input')?.focus();
    });
  }
  private onTerminalCommand(key: string, slot: Slot, binding: TerminalContext, event: Event) {
    if (!this.commandAdmitted(slot, event)) return;
    this.rememberFocus();
    const detail = (event as CustomEvent).detail;
    if (detail === 'hide') void this.hideSlot(slot);
    if (detail === 'rename') this.beginRename(key, slot);
    if (detail === 'project') void this.showManagement(binding.repositoryId);
  }
  private display() {
    displaySlots(this.hostsInput());
    for (const [id, project] of this.projects) project.host.hidden = this.view !== 'project' || id !== this.project;
    displayFactoryWatches({
      binding: this.binding,
      actor: this.actor,
      spaces: this.spaces,
      watches: this.watches,
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      isSurfaceVisible: () => this.surfaceVisible,
      projectName: (space) => this.projectName(space),
      findOwner: (runId) => this.querySelector<HTMLElement>(`[data-factory-owner="${runId}"]`),
      ownerConcealed: () => !this.activeSurface,
      invalidate: () => this.invalidate(),
      requestUpdate: () => this.requestUpdate(),
    });
  }
  private rectangle(area: Area) {
    return `left:${area.x}px;top:${area.y}px;width:${area.width}px;height:${area.height}px`;
  }
  private layoutInput(): LayoutInput {
    return {
      binding: this.binding,
      readLayout: () => this.layout,
      readMaximized: () => this.maximized,
      readWorkspaceWidth: () => this.workspaceWidth,
      readCanvasSize: () => this.canvasSize,
      readSlots: () => this.slots,
      readCell: () => this.cell,
      readTabHeight: () => this.tabHeight,
      readView: () => this.view,
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      readStorageLoaded: () => this.storageLoaded,
      readStorageKey: () => this.storageKey,
      hostRect: () => this.getBoundingClientRect(),
      canvasRect: () => this.querySelector('.soda-workspace-canvas')?.getBoundingClientRect(),
      setLayout: (layout) => {
        this.layout = layout;
      },
      setMaximized: (key) => {
        this.maximized = key;
      },
      setStorageNotice: (notice) => {
        this.storageNotice = notice;
      },
      setStatus: (status) => {
        this.status = status;
      },
      setStorageLoaded: (loaded) => {
        this.storageLoaded = loaded;
      },
      closeMenus: () => this.closeMenus(),
      requestUpdate: () => this.requestUpdate(),
      rectangle: (area) => this.rectangle(area),
      capture: (event) => this.capture(event),
      releasePointer: (event) => this.releasePointer(event),
    };
  }
  private capture(event: PointerEvent) {
    if (event.button === 0 && event.currentTarget instanceof HTMLElement) {
      event.currentTarget.setPointerCapture(event.pointerId);
      event.preventDefault();
    }
  }
  private releasePointer(event: PointerEvent) {
    if (event.currentTarget instanceof HTMLElement && event.currentTarget.hasPointerCapture(event.pointerId))
      event.currentTarget.releasePointerCapture(event.pointerId);
  }
  private tabKey(event: KeyboardEvent, key: string, keys: string[]) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const index = keys.indexOf(key),
      next =
        keys[
          event.key === 'Home'
            ? 0
            : event.key === 'End'
              ? keys.length - 1
              : (index + (event.key === 'ArrowLeft' ? -1 : 1) + keys.length) % keys.length
        ];
    const entry = this.layout.entries.find((e) => e.key === next);
    if (entry) {
      void this.openSaved(entry, undefined, false);
      void this.updateComplete.then(() => {
        if (!this.stale && this.surfaceVisible && this.view === 'terminal' && this.selected === entry.key)
          this.querySelector<HTMLElement>('#soda-tab-' + entry.key)?.focus();
      });
    }
  }
  private hideSlot(slot: Slot) {
    if (!this.stale && !this.disposed) arrange(this.layoutInput(), hideTab(this.layout, slot.key));
  }
  private createInput(): CreateInput {
    return {
      isStale: () => this.stale,
      isDisposed: () => this.disposed,
      isAvailable: () => this.available,
      isActiveSurface: () => this.activeSurface,
      readCreating: () => this.creating,
      setCreating: (creating) => {
        this.creating = creating;
      },
      readCreation: () => this.creation,
      setCreation: (creation) => {
        this.creation = creation;
      },
      readSpaces: () => this.spaces,
      readLayout: () => this.layout,
      setLayout: (layout) => {
        this.layout = layout;
      },
      setView: (view) => {
        this.view = view;
      },
      setStatus: (status) => {
        this.status = status;
      },
      isValidName: (name) => validName(name),
      addSlot: (space, entry) => addSlot(this.hostsInput(), space, entry),
      openSaved: (entry, destination) => this.openSaved(entry, destination),
      confirmedEnd: (key) => confirmedEnd(this.terminalsInput(), key),
      requestUpdate: () => this.requestUpdate(),
    };
  }
  private managementAdmitted(repositoryId: string) {
    return id(repositoryId) && !!this.actor && this.available && !this.stale && !this.disposed && this.activeSurface;
  }
  private mountProject(repositoryId: string) {
    const layer = this.querySelector('.soda-workspace-management');
    check(layer && this.actor && this.binding);
    const host = document.createElement('div');
    layer.append(host);
    const api = mountProjectControls(host, {
      repositoryId,
      expectedUserId: this.actor.id,
      actorLogin: this.actor.login,
      page: this.binding.kind === 'page',
      transport: this.binding.transport,
      forgejoPrefix: this.binding.forgejoPrefix || '',
    });
    const project = {host, api};
    this.projects.set(repositoryId, project);
    return project;
  }
  private defaultManagementMode(): 'standard' | 'journey' | 'settings' {
    if (this.binding?.kind === 'page') return 'settings';
    return 'standard';
  }
  private refreshMountedProject(
    project: {api: ReturnType<typeof mountProjectControls>},
    created: boolean,
    mode: 'standard' | 'journey' | 'settings'
  ) {
    const changed = project.api.setPresentation(mode);
    if (created || changed || mode === 'journey') void project.api.refresh();
  }
  private focusConfigureIfNeeded(n: number, repositoryId: string) {
    if (this.live(n) && this.project === repositoryId && this.setup === 'configure')
      focusSetupPanel(
        this,
        () => this.stale,
        () => this.activeSurface
      );
  }
  private async showManagement(repositoryId: string, mode?: 'standard' | 'journey' | 'settings') {
    if (!this.managementAdmitted(repositoryId)) return;
    await this.presentManagement(repositoryId, mode || this.defaultManagementMode());
  }
  private async presentManagement(repositoryId: string, mode: 'standard' | 'journey' | 'settings') {
    this.rememberFocus();
    this.project = repositoryId;
    this.managementMode = mode;
    this.view = 'project';
    const n = this.epoch;
    await this.updateComplete;
    if (!this.live(n)) return;
    let project = this.projects.get(repositoryId);
    const created = !project;
    if (!project) project = this.mountProject(repositoryId);
    this.refreshMountedProject(project, created, mode);
    this.display();
    await project.api.ready;
    this.focusConfigureIfNeeded(n, repositoryId);
  }

  setVisible(visible: boolean) {
    this.surfaceVisible = visible;
    this.requestUpdate();
    this.display();
    if (visible && !this.restored) void this.refresh();
  }
  /** Host concealment: shelter without closing any terminal or watch stream. */
  suspend() {
    if (this.disposed || this.stale) return;
    this.setVisible(false);
    for (const watch of this.watches) watch.view?.suspend();
    for (const project of this.projects.values()) project.api.suspend();
  }
  /** Host rebind after shelter/revalidation; attachments stay exact. */
  resume(validation?: ResumeValidation) {
    if (this.disposed) return;
    if (this.binding && resumeGenerationRefused(this.binding.transport.generation, validation)) {
      this.retire();
      return;
    }
    if (this.stale) return;
    this.setVisible(true);
    this.markViewed();
    for (const watch of this.watches) watch.view?.resume(validation);
    for (const project of this.projects.values()) project.api.resume(validation);
  }
  /** Host retirement: release all browser resources without End. */
  retire() {
    this.dispose();
  }
  invalidate() {
    this.reconnectRequired = false;
    if (this.stale) return;
    this.stale = true;
    this.applicationOwner = '';
    this.available = this.complete = false;
    this.measurement.retire();
    window.clearInterval(this.attentionTimer);
    ++this.epoch;
    this.request?.abort();
    this.repositoryRequest?.abort();
    this.repositoryRequest = undefined;
    this.repositoryResult = undefined;
    this.repositoryQuery = this.repositoryChoice = '';
    this.repositoryBusy = false;
    this.setup = null;
    this.setupReturn = undefined;
    this.busy = false;
    this.creation = this.editing = null;
    for (const slot of this.slots) {
      slot.metadata = undefined;
      delete slot.proposedName;
      slot.observation = undefined;
      slot.unread = slot.readRequested = false;
      slot.terminal.invalidate();
    }
    for (const watch of this.watches) watch.view?.invalidate();
    for (const project of this.projects.values()) project.api.invalidate();
    this.spaces = [];
    this.display();
    this.status =
      'Page or Soda identity changed. Reload; no action was replayed. Inspect any dispatched operation independently.';
  }
  get canRestore() {
    return [...this.projects.values()].every((project) => project.api.canRestore);
  }
  dispose() {
    if (this.disposed) return;
    this.invalidate();
    this.disposed = true;
    this.lifetime.abort();
    for (const slot of this.slots) slot.terminal.dispose();
    for (const watch of this.watches) watch.view?.dispose();
    for (const project of this.projects.values()) project.api.dispose();
    this.remove();
  }
}
customElements.define('soda-spaces', SodaSpaces);
function workspacePresentation(
  context: WorkspaceContext,
  kind: 'page' | 'native',
  repositoryId?: string
): WorkspaceContext {
  const base = {
    transport: context.transport,
    forgejoPrefix: context.forgejoPrefix || '',
    ...(context.navigate ? {navigate: context.navigate} : {}),
    ...(repositoryId ? {repositoryId} : {}),
  };
  return kind === 'page' ? {...base, kind} : {...base, kind, ...(repositoryId ? {pageRepositoryId: repositoryId} : {})};
}
function workspaceContextValid(context: WorkspaceContext) {
  if (context.kind === 'page') return context.repositoryId === undefined || id(context.repositoryId);
  return (
    context.kind === 'native' &&
    (context.repositoryId === undefined || id(context.repositoryId)) &&
    (context.pageRepositoryId === undefined || id(context.pageRepositoryId))
  );
}
export function mountSodaspaces(
  root: HTMLElement,
  context: WorkspaceContext,
  factory: TerminalFactory = mountTerminal
) {
  check(root.ownerDocument === document && workspaceContextValid(context));
  if (context.kind === 'page') root.classList.add('soda-spaces-page-mount');
  const box = new SodaSpaces();
  box.configure(context, factory);
  root.append(box);
  return {
    get canRestore() {
      return box.canRestore;
    },
    markViewed: () => box.markViewed(),
    setVisible: (visible: boolean) => box.setVisible(visible),
    present: (kind: 'page' | 'native', repositoryId?: string) => box.present(kind, repositoryId),
    update: (presentation: ContributionPresentation) => box.applyPresentation(presentation),
    suspend: () => box.suspend(),
    resume: (validation?: ResumeValidation) => box.resume(validation),
    retire: () => box.retire(),
    refresh: () => box.refresh(),
    invalidate: () => box.invalidate(),
    get ready() {
      return box.updateComplete;
    },
    dispose: () => box.dispose(),
  };
}

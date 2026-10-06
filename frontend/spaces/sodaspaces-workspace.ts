import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {
  renderMenu,
  renderRename,
  renderCreation,
  renderRepositoryPicker,
  renderWelcome,
  renderWelcomeSteps,
  renderWorkspaceIntro,
} from './sodaspaces-workspace-view.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {mountTerminal} from './sodaspaces-terminal.js';
import type {TerminalContext} from './sodaspaces-terminal.js';
import {
  emptyLayout,
  focusedPane,
  selectTab,
  hideTab,
  putEntry,
  sameLocator,
  layoutLimit,
  panes,
} from './sodaspaces-layout.js';
import type {WorkspaceLayout, LayoutEntry, Area} from './sodaspaces-layout.js';
import {check, id, object, terminalResponse, repositoryChoices, projectId} from './sodaspaces-api.js';
import type {Space, TerminalMetadata, RepositoryChoices} from './sodaspaces-api.js';
import type {Creation, FactoryWatch, Row, Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';
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
  repositorySearchPath,
  searchAdmitted,
  setupIntroAction,
  setupIntroDescription,
  setupIntroHeading,
  setupIntroHelper,
  setupIntroKind,
} from './sodaspaces-workspace-setup.js';
import {attentionRows, filteredSpaces, projectRows, rows} from './sodaspaces-workspace-navigation.js';
import {displayFactoryWatches, factorySection} from './sodaspaces-workspace-factory.js';
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
  openSaved as openSavedTerminal,
  restoreLocators,
  slotName,
} from './sodaspaces-workspace-terminals.js';
import type {TerminalsInput} from './sodaspaces-workspace-terminals.js';
type TerminalFactory = typeof mountTerminal;
const validName = (name: string) => [...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name);

// Both surfaces share this owner. Pane chrome is keyed separately; live terminal
// hosts never leave their flat parent. Rendering cannot create/attach/Return.
export class SodaSpaces extends LitElement {
  static properties = {
    setup: {state: true},
    project: {state: true},
    complete: {state: true},
    repositoryQuery: {state: true},
    repositoryResult: {state: true},
    repositoryChoice: {state: true},
    repositoryBusy: {state: true},
    repositoryError: {state: true},
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
  declare private view: 'terminal' | 'sessions' | 'project';
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
  declare private setup: 'repositories' | 'configure' | null;
  declare private complete: boolean;
  declare private repositoryQuery: string;
  declare private repositoryResult: RepositoryChoices | undefined;
  declare private repositoryChoice: string;
  declare private repositoryBusy: boolean;
  declare private repositoryError: string;
  private repositoryRequest: AbortController | undefined;
  private repositoryCursors: string[] = [''];
  private setupReturn:
    | {project: string; view: 'terminal' | 'sessions' | 'project'; mode: 'standard' | 'journey' | 'settings'}
    | undefined;
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
  private invoker: HTMLElement | undefined;
  private lastMinimum = 0;
  constructor() {
    super();
    this.spaces = [];
    this.project = this.repositoryQuery = this.repositoryChoice = this.repositoryError = this.nextAfter = '';
    this.setup = null;
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
    return this.surfaceVisible && this.isConnected && !this.closest('[hidden], [inert]');
  }
  private get selectedSpace() {
    return this.spaces.find((s) => s.environment.repository_id === this.project);
  }
  private get setupScreen() {
    return this.binding?.kind === 'page' && (!!this.setup || !this.spaces.length);
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
    return this.pageReadyForFirstTerminal() && this.spaceReadyForFirstTerminal(this.selectedSpace);
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
    this.factory = factory;
    this.storageKey = '';
    this.addEventListener('focusout', (event) => menuFocusOut(event), {signal: this.lifetime.signal});
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
    document.addEventListener(
      'visibilitychange',
      () => {
        if (document.visibilityState === 'visible') this.markViewed();
      },
      {
        signal: this.lifetime.signal,
      }
    );
    // One mounted-workspace timer, never a navbar poller. GET refresh uses the
    // existing bounded/cancellable caller, and cannot create work.
    this.attentionTimer = window.setInterval(() => this.pollAttention(), 30000);
    this.addEventListener('soda-project-observed', (e) => this.onProjectObserved(e));
    this.addEventListener('soda-project-operation', (e) => this.onProjectOperation(e));
    this.addEventListener('soda-project-change-repository', (e) => this.onProjectChangeRepository(e));
    this.addEventListener('soda-project-changed', (e) => {
      if (!(e instanceof CustomEvent)) return;
      void this.refresh();
    });
  }
  private pollAttention() {
    if (this.stale || this.disposed) return;
    this.now = Date.now();
    this.requestUpdate();
    if (this.shouldRefreshAttention()) void this.refresh();
  }
  private shouldRefreshAttention() {
    if (!this.activeSurface || document.visibilityState !== 'visible') return false;
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
    if (this.projectEventFromHost(e)) this.changeRepository();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  protected updated() {
    this.display();
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
  private readonly onBeginSetup = () => this.beginSetup();
  private readonly onShowSessions = () => this.showSessions();
  private readonly onNewTerminal = () => this.newTerminal();
  private readonly onCancelSetup = () => this.cancelSetup();
  private readonly onSetupBack = () => (this.setup === 'configure' ? this.changeRepository() : this.cancelSetup());
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
    return this.binding?.kind === 'page' ? 'Projects and terminals' : 'Projects and sessions';
  }
  private renderProjectsHeading() {
    if (this.binding?.kind !== 'page') return '';
    const title =
      !this.compact && this.view !== 'sessions'
        ? html`<button class="ui button" @click=${this.onShowSessions}>Projects</button>`
        : html`<h2>Projects</h2>`;
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
    return (
      this.binding?.kind === 'page' &&
      !attentionRows(this.navReading(), this.spaces, this.layout.entries).length &&
      !this.attentionOnly
    );
  }
  private renderCreateAnotherProject() {
    if (this.binding?.kind !== 'page') return '';
    return html`<button
      class="ui button soda-create-another-project"
      ?disabled=${workspaceBlocked(this.stale, this.available) || !this.canRestore}
      @click=${this.onBeginSetup}
    >
      <span aria-hidden="true">＋</span> Create project
    </button>`;
  }
  private emptyFilterMessage() {
    if (!this.available) return 'Status unavailable.';
    if (this.attentionOnly) return 'No matching sessions need attention.';
    if (this.search || this.thisPage) return 'No matches.';
    return 'No authorized projects available.';
  }
  private renderEmptySpaces() {
    if (
      filteredSpaces({
        reading: this.navReading(),
        spaces: this.spaces,
        entries: this.layout.entries,
        search: this.search,
        attentionOnly: this.attentionOnly,
        thisPage: this.thisPage,
        binding: this.binding,
        projectName: (space) => this.projectName(space),
      }).length
    )
      return '';
    return html`<p>${this.emptyFilterMessage()}</p>`;
  }
  private renderWelcomeFooter() {
    if (!this.welcomeScreen && !this.setup) return '';
    const step = this.setup === 'configure' ? 2 : this.setup === 'repositories' ? 1 : undefined;
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
    if (slot && edit) void this.rename(slot, edit.name);
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
    return this.creationForm(this.creation);
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
        >Find a terminal <input type="search" .value=${this.search} @input=${this.onSearchInput}
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
      ${repeat(
        filteredSpaces({
          reading: this.navReading(),
          spaces: this.spaces,
          entries: this.layout.entries,
          search: this.search,
          attentionOnly: this.attentionOnly,
          thisPage: this.thisPage,
          binding: this.binding,
          projectName: (space) => this.projectName(space),
        }),
        (space) => space.environment.id,
        (space) =>
          projectRows({
            reading: this.navReading(),
            space,
            search: this.search,
            attentionOnly: this.attentionOnly,
            selected: this.selected,
            entries: this.layout.entries,
            tree: this.layout.tree,
            kind: this.binding?.kind,
            project: this.project,
            state: this.projectState(space),
            status: this.projectStatus(space),
            projectName: this.projectName(space),
            factory: factorySection({
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
              query: this.search.toLocaleLowerCase(),
            }),
            selectRow: (target, row) => this.selectRow(target, row),
            showManagement: (repository) => this.showManagement(repository),
            selectProject: (target) => this.selectProject(target),
          })
      )}
      ${renderMoreProjects(this.moreProjectsInput())} ${this.renderCreateAnotherProject()} ${this.renderEmptySpaces()}
    </nav>`;
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
      this.defaultTerminalName(this.selectedSpace),
      workspaceBlocked(this.stale, this.available),
      this.creating,
      this.onNewTerminal
    );
    return html`<div class="soda-workspace-canvas" ?hidden=${hideCanvas(this.view, this.stale)}>
      <span class="soda-cell-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span>
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
      ${topbar} ${this.setupScreen ? this.renderSetup() : ''}
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
          ${this.renderPageToolbar()} ${this.renderCanvas()}
          <div class="soda-workspace-management" ?hidden=${hideManagement(this.view, this.stale)}></div>
        </div>
      </div>
      ${this.renderWelcomeFooter()}
    </div>`;
  }
  protected render() {
    const heading = renderSetupHeading(this.setupScreen);
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
        ${openInDrawerOption} ${toggleSidebarOption} ${nativeMgmt}
      `
    );
  }
  private renderToolbar() {
    const kind = this.binding?.kind;
    const space = this.selectedSpace;
    const project = renderToolbarProject(
      kind,
      space,
      space ? this.projectState(space) : '',
      space ? this.projectStatus(space) : '',
      space ? this.projectName(space) : ''
    );
    const settings = renderProjectSettingsButton(kind, workspaceBlocked(this.stale, this.available), space, () =>
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
      ${project}
      <button
        class=${newTerminalButtonClass(kind)}
        data-environment-id=${this.selectedSpace?.environment.id || ''}
        data-terminal-name=${this.defaultTerminalName(this.selectedSpace)}
        aria-label="New terminal"
        title="New terminal"
        ?hidden=${hideNewTerminal(this.firstTerminal, kind, space)}
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
  private clearRepositorySearch() {
    this.repositoryQuery = this.repositoryError = '';
    this.repositoryChoice = '';
    this.repositoryResult = undefined;
    this.repositoryCursors = [''];
    this.repositoryRequest?.abort();
    this.repositoryRequest = undefined;
    this.repositoryBusy = false;
  }
  private setupPanelClass() {
    return this.setup === 'repositories' ? 'soda-setup-form' : 'soda-setup-welcome';
  }
  private renderSetupBody(blocked: boolean) {
    if (this.setup === 'repositories') {
      return renderRepositoryPicker(
        {
          query: this.repositoryQuery,
          result: this.repositoryResult,
          selected: this.repositoryChoice,
          busy: this.repositoryBusy,
          error: this.repositoryError,
          blocked,
          createURL: this.repositoryPrefix() + '/repo/create',
        },
        {
          query: (value) => this.onRepositoryQuery(value),
          search: (page) => {
            void this.searchRepositories(page);
          },
          select: (value) => {
            this.repositoryChoice = value;
          },
          back: () => this.cancelSetup(),
          continue: () => this.configureProject(),
        }
      );
    }
    if (this.welcomeScreen) return renderWelcome(blocked, this.onBeginSetup);
    return renderWorkspaceIntro({
      kind: setupIntroKind(this.busy),
      heading: setupIntroHeading(this.busy, this.reconnectRequired),
      description: setupIntroDescription(this.busy, this.reconnectRequired),
      action: setupIntroAction(
        this.busy,
        this.reconnectRequired,
        this.stale,
        this.connectURL,
        blocked,
        this.onRefreshClick
      ),
      helper: setupIntroHelper(this.busy),
    });
  }
  private onRepositoryQuery(value: string) {
    this.repositoryQuery = value;
    this.repositoryCursors = [''];
    this.repositoryChoice = '';
    this.repositoryResult = undefined;
    this.repositoryError = '';
    this.repositoryRequest?.abort();
    this.repositoryRequest = undefined;
    this.repositoryBusy = false;
  }
  private setupUnavailableHeading() {
    if (this.busy) return 'Loading projects…';
    if (this.reconnectRequired) return 'Reconnect to Forgejo';
    return 'Could not load projects';
  }
  private setupUnavailableDescription() {
    if (this.busy) return html`Checking the projects you can access.`;
    if (this.reconnectRequired) return html`Sign in again to restore your Forgejo access.`;
    return html`We couldn’t load your project list. Try again.`;
  }
  private setupUnavailableAction(blocked: boolean) {
    if (this.busy) return html``;
    if (this.reconnectRequired)
      return html`<a class="ui primary button" href=${this.connectURL}>Reconnect to Forgejo</a>`;
    if (this.stale)
      return html`<button class="ui primary button" @click=${() => window.location.reload()}>Reload Spaces</button>`;
    return html`<button class="ui primary button" ?disabled=${blocked} @click=${() => this.refresh()}>
      Retry projects
    </button>`;
  }
  private setupUnavailableHelper() {
    if (this.busy) return html`This will not create or start anything.`;
    return html`Your existing projects and terminals are not replaced.`;
  }
  private renderSetupUnavailable(blocked: boolean) {
    return renderWorkspaceIntro({
      kind: this.busy ? 'loading' : 'unavailable',
      heading: this.setupUnavailableHeading(),
      description: this.setupUnavailableDescription(),
      action: this.setupUnavailableAction(blocked),
      helper: this.setupUnavailableHelper(),
    });
  }
  private renderSetup() {
    const blocked = this.stale || !this.canRestore;
    const formClass = this.setup === 'repositories' ? 'soda-setup-form' : 'soda-setup-welcome';
    return html`<div class="soda-setup-panel" ?hidden=${this.setup === 'configure'}>
      <div class=${formClass}>${this.renderSetupBody(blocked)}</div>
    </div>`;
  }
  private focusSetup() {
    void this.updateComplete.then(() => {
      if (!this.stale && this.activeSurface)
        this.querySelector<HTMLElement>('.soda-setup-panel:not([hidden]) h2, .soda-project-journey h2')?.focus();
    });
  }
  private beginSetup() {
    if (this.stale || !this.available || !this.canRestore || !this.activeSurface) return;
    this.rememberFocus();
    this.setupReturn = {project: this.project, view: this.view, mode: this.managementMode};
    this.setup = 'repositories';
    this.repositoryChoice = '';
    this.focusSetup();
    void this.searchRepositories(1);
  }
  private changeRepository() {
    if (this.stale || !this.canRestore || this.setup !== 'configure' || !this.activeSurface) return;
    this.setup = 'repositories';
    this.focusSetup();
  }
  private cancelSetup() {
    if (this.stale || !this.canRestore) return;
    this.repositoryRequest?.abort();
    this.repositoryRequest = undefined;
    this.repositoryBusy = false;
    this.setup = null;
    if (this.setupReturn) {
      this.project = this.setupReturn.project;
      this.view = this.setupReturn.view;
      this.managementMode = this.setupReturn.mode;
    }
    this.setupReturn = undefined;
    if (!this.selectedSpace && this.spaces[0]) this.selectProject(this.spaces[0]);
    this.restoreFocus();
  }
  private applyRepositorySearch(request: AbortController, query: string, result: RepositoryChoices) {
    if (this.stale || this.repositoryRequest !== request || request.signal.aborted) return;
    if (this.repositoryQuery !== query) return;
    this.repositoryCursors = this.repositoryCursors.slice(0, result.page);
    if (result.nextCursor) this.repositoryCursors.push(result.nextCursor);
    this.repositoryResult = result;
  }
  private failRepositorySearch(request: AbortController, query: string) {
    if (this.stale || this.repositoryRequest !== request) return;
    if (this.repositoryQuery === query)
      this.repositoryError = 'Could not find repositories. Search again; no project was created.';
  }
  private searchCursor(page: number) {
    if (page === 1) this.repositoryCursors = [''];
    return this.repositoryCursors[page - 1];
  }
  private async searchRepositories(page: number) {
    if (!searchAdmitted(this.stale, this.available, this.setup, this.activeSurface)) return;
    const cursor = this.searchCursor(page);
    if (cursor === undefined) return;
    this.repositoryRequest?.abort();
    const request = (this.repositoryRequest = new AbortController()),
      query = this.repositoryQuery;
    const timer = window.setTimeout(() => request.abort(), 15000);
    this.repositoryBusy = true;
    this.repositoryError = '';
    this.repositoryChoice = '';
    this.repositoryResult = undefined;
    try {
      const result = repositoryChoices(
        await this.api(repositorySearchPath(query, cursor), undefined, request.signal),
        page
      );
      this.applyRepositorySearch(request, query, result);
    } catch {
      this.failRepositorySearch(request, query);
    } finally {
      window.clearTimeout(timer);
      if (this.repositoryRequest === request) this.repositoryBusy = false;
    }
  }
  private configureProject() {
    const choice = this.repositoryResult?.items.find((item) => item.id === this.repositoryChoice);
    if (!choice || this.repositoryBusy || this.stale || !this.canRestore || this.setup !== 'repositories') return;
    this.setup = 'configure';
    void this.showManagement(choice.id, 'journey');
  }
  private selectProject(space: Space) {
    if (this.stale || !this.available || !this.canRestore) return;
    this.project = space.environment.repository_id;
    this.back();
    if (!space.login || !space.environment.provisioned || space.observed?.running !== true)
      void this.showManagement(this.project, 'journey');
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
    else if (next.row.metadata) void this.openExisting(next.space, next.row.metadata);
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
    if (row.entry) void this.openSaved(row.entry, this.choosingPane);
    else if (row.metadata) void this.openExisting(space, row.metadata, this.choosingPane);
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
      newTerminal: (pane) => this.newTerminal(pane),
    };
  }
  private creationEligible(space: Space | undefined) {
    return (
      !!space?.login &&
      space.environment.provisioned &&
      !space.authority_unavailable &&
      space.execution_allowed &&
      space.observed?.running === true
    );
  }
  private creationExplanation(space: Space | undefined) {
    if (this.creationEligible(space)) return '';
    if (!space?.login) return 'Join required.';
    if (space.authority_unavailable) return 'Status unavailable.';
    if (!space.execution_allowed) return 'Repository write access required.';
    return 'Environment stopped.';
  }
  private creationContext(space: Space | undefined) {
    return `${space?.login || 'Join required'} @ ${space ? this.projectName(space) : 'Select a project'}`;
  }
  private creationDisabled(draft: Creation, eligible: boolean) {
    return this.creating || this.stale || !this.available || !eligible || !validName(draft.name);
  }
  private onCreationEnvironment(environmentId: string) {
    if (this.creation) this.creation = {...this.creation, environmentId};
  }
  private onCreationName(name: string) {
    if (this.creation) this.creation = {...this.creation, name};
  }
  private cancelCreation() {
    this.creation = null;
    this.restoreFocus();
  }
  private creationForm(draft: Creation) {
    const space = this.spaces.find((s) => s.environment.id === draft.environmentId);
    const eligible = this.creationEligible(space);
    return renderCreation(
      {
        environmentId: draft.environmentId,
        name: draft.name,
        projects: this.spaces.map((item) => ({id: item.environment.id, name: this.projectName(item)})),
        context: this.creationContext(space),
        explanation: this.creationExplanation(space),
        busy: this.creating,
        disabled: this.creationDisabled(draft, !!eligible),
      },
      (environmentId) => this.onCreationEnvironment(environmentId),
      (name) => this.onCreationName(name),
      () => this.cancelCreation(),
      () => {
        void this.createTerminal();
      }
    );
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
    this.thisPage = false;
    if (this.binding?.kind === 'page' && !this.compact) {
      if (this.layout.sidebar === null) {
        this.layout = {...this.layout, sidebar: 256};
        this.persist();
      }
    } else this.view = 'sessions';
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
    this.view = 'terminal';
    this.choosingPane = undefined;
    this.restoreFocus();
    this.markViewed();
  }
  private defaultTerminalName(space: Space | undefined) {
    return (
      'Terminal ' +
      (Math.max(
        space?.terminals.length || 0,
        this.layout.entries.filter((e) => e.environmentId === space?.environment.id).length
      ) +
        1)
    );
  }
  private creationSpace(selected: {binding: {environmentId: string}} | undefined) {
    const preferred = this.binding?.kind === 'page' ? this.selectedSpace : undefined;
    if (preferred) return preferred;
    const native = this.binding?.kind === 'native' ? this.binding.repositoryId : undefined;
    return (
      this.spaces.find((s) => s.environment.id === selected?.binding.environmentId) ||
      this.spaces.find((s) => s.environment.repository_id === native) ||
      this.spaces[0]
    );
  }
  private pickCreationSpace() {
    const selected = this.slots.find((s) => s.key === this.selected);
    const native = this.binding?.kind === 'native' ? this.binding.repositoryId : undefined;
    if (this.binding?.kind === 'page' && this.selectedSpace) return this.selectedSpace;
    return (
      this.spaces.find((s) => s.environment.id === selected?.binding.environmentId) ||
      this.spaces.find((s) => s.environment.repository_id === native) ||
      this.spaces[0]
    );
  }
  private pageBlocksNewTerminal(space: Space | undefined) {
    if (this.binding?.kind !== 'page' || !space) return false;
    return (
      !space.login ||
      !space.execution_allowed ||
      space.authority_unavailable ||
      !space.environment.provisioned ||
      space.observed?.running !== true
    );
  }
  private newTerminalBlocked() {
    return this.stale || this.creating || !this.available || !this.activeSurface;
  }
  private focusCreationDialog(pane: string) {
    void this.updateComplete.then(() => {
      if (this.creation?.pane === pane && !this.stale && this.surfaceVisible)
        this.querySelector<HTMLElement>('.soda-workspace-dialog select')?.focus();
    });
  }
  private newTerminal(pane = this.layout.focused) {
    if (this.newTerminalBlocked()) return;
    this.rememberFocus();
    const space = this.pickCreationSpace();
    if (this.pageBlocksNewTerminal(space)) return;
    this.creation = {pane, environmentId: space?.environment.id || '', name: this.defaultTerminalName(space)};
    if (this.binding?.kind === 'page' && this.selectedSpace) {
      void this.createTerminal();
      return;
    }
    this.focusCreationDialog(pane);
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
      concealed: () => this.closest('[hidden]') !== null,
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
      concealed: () => this.closest('[hidden]') !== null,
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
      ownerConcealed: () => this.closest('[hidden]') !== null,
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
  private createAdmitted(draft: Creation | null): draft is Creation {
    return (
      !!draft &&
      !this.creating &&
      this.available &&
      !this.stale &&
      !this.disposed &&
      this.activeSurface &&
      validName(draft.name)
    );
  }
  private createSpaceReady(space: Space | undefined) {
    return (
      !!space?.login &&
      space.environment.provisioned &&
      !space.authority_unavailable &&
      space.execution_allowed &&
      space.observed?.running === true
    );
  }
  private async openCreatedSlot(space: Space, draft: Creation) {
    const entry: LayoutEntry = {key: crypto.randomUUID(), environmentId: space.environment.id, locator: {kind: 'new'}};
    this.layout = selectTab(putEntry(this.layout, entry), entry.key, draft.pane);
    this.view = 'terminal';
    this.creation = null;
    const slot = await addSlot(this.hostsInput(), space, entry);
    if (!slot) return;
    slot.proposedName = draft.name;
    this.requestUpdate();
    await slot.terminal.open(draft.name);
  }
  private async createTerminal() {
    const draft = this.creation;
    if (!this.createAdmitted(draft)) return;
    const space = this.spaces.find((p) => p.environment.id === draft.environmentId);
    if (!this.createSpaceReady(space)) return;
    if (!panes(this.layout.tree).some((p) => p.key === draft.pane)) {
      this.status = 'The destination pane changed. Choose New terminal again.';
      return;
    }
    if (this.layout.entries.length >= layoutLimit) {
      this.status = 'The working set is full; uncertain locators cannot be evicted.';
      return;
    }
    this.creating = true;
    try {
      await this.openCreatedSlot(space!, draft);
    } finally {
      this.creating = false;
    }
  }
  private openExistingBlocked(space: Space) {
    return (
      this.stale ||
      this.disposed ||
      !this.activeSurface ||
      !this.available ||
      space.authority_unavailable ||
      !space.execution_allowed
    );
  }
  private existingOrNewEntry(space: Space, metadata: TerminalMetadata) {
    let entry = this.layout.entries.find((e) => sameLocator(e.locator, {kind: 'existing', id: metadata.id}));
    if (entry) {
      check(entry.environmentId === space.environment.id);
      return entry;
    }
    entry = {
      key: crypto.randomUUID(),
      environmentId: space.environment.id,
      locator: {kind: 'existing', id: metadata.id},
    };
    this.layout = putEntry(this.layout, entry);
    return entry;
  }
  private async openExisting(space: Space, metadata: TerminalMetadata, destination?: string) {
    if (this.openExistingBlocked(space)) return;
    try {
      const entry = this.existingOrNewEntry(space, metadata);
      if (metadata.state === 'ended') {
        confirmedEnd(this.terminalsInput(), entry.key);
        return;
      }
      await this.openSaved(entry, destination);
    } catch {
      if (!this.stale)
        this.status = 'The exact session could not be opened. No locator was replaced or creation requested.';
    }
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
    if (this.live(n) && this.project === repositoryId && this.setup === 'configure') this.focusSetup();
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
  private renameAdmitted(slot: Slot, name: string) {
    return (
      !this.renaming.has(slot.key) &&
      !this.stale &&
      !this.disposed &&
      this.activeSurface &&
      validName(name) &&
      this.editing?.key === slot.key
    );
  }
  private applyRenamedMetadata(slot: Slot, value: TerminalMetadata, id: string) {
    check(value?.id === id);
    slot.metadata = value;
    slot.terminal.setName(value.name);
    if (this.editing?.key === slot.key) this.editing = null;
  }
  private async rename(slot: Slot, name: string) {
    if (!this.renameAdmitted(slot, name)) return;
    const locator = this.locator(slot);
    if (locator.kind !== 'existing') return;
    const id = locator.id,
      n = this.epoch;
    this.renaming.add(slot.key);
    this.requestUpdate();
    const request = new AbortController(),
      timeout = window.setTimeout(() => request.abort(), 15000);
    try {
      const value = terminalResponse(
        await this.api(
          `/api/environments/${slot.binding.environmentId}/terminal-sessions/${id}`,
          {action: 'rename', name},
          AbortSignal.any([request.signal, this.lifetime.signal])
        ),
        slot.binding
      );
      if (this.live(n) && value) this.applyRenamedMetadata(slot, value, id);
    } catch {
      if (this.live(n)) this.status = 'Rename was not confirmed. No retry was made.';
    } finally {
      window.clearTimeout(timeout);
      this.renaming.delete(slot.key);
      this.requestUpdate();
    }
  }
  setVisible(visible: boolean) {
    this.surfaceVisible = visible;
    this.display();
    if (visible && !this.restored) void this.refresh();
  }
  invalidate() {
    this.reconnectRequired = false;
    if (this.stale) return;
    this.stale = true;
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
export function mountSodaspaces(
  root: HTMLElement,
  context: WorkspaceContext,
  factory: TerminalFactory = mountTerminal
) {
  check(
    root.ownerDocument === document &&
      (context.kind === 'native'
        ? (context.repositoryId === undefined || id(context.repositoryId)) &&
          (context.pageRepositoryId === undefined || id(context.pageRepositoryId))
        : context.kind === 'page')
  );
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
    refresh: () => box.refresh(),
    invalidate: () => box.invalidate(),
    get ready() {
      return box.updateComplete;
    },
    dispose: () => box.dispose(),
  };
}

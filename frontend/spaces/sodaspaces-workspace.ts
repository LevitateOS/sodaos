import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {terminalObservation} from './sodaspaces-attention.js';
import {
  renderMenu,
  renderSessionTab,
  renderRename,
  renderCreation,
  renderRepositoryPicker,
  renderWelcome,
  renderWelcomeSteps,
  renderWorkspaceIntro,
} from './sodaspaces-workspace-view.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {mountTerminal} from './sodaspaces-terminal.js';
import type {TerminalContext, TerminalLocator} from './sodaspaces-terminal.js';
import {
  emptyLayout,
  focusedPane,
  paneFor,
  selectTab,
  hideTab,
  putEntry,
  forgetEntry,
  sameLocator,
  parseLayout,
  serializeLayout,
  layoutLimit,
  panes,
  splitPane,
  moveTab,
  resizeSplit,
  consolidate,
  projectLayout,
} from './sodaspaces-layout.js';
import type {WorkspaceLayout, LayoutEntry, Pane, Split, Area, Minimum, DividerArea} from './sodaspaces-layout.js';
import {
  check,
  id,
  object,
  readSodaJSON,
  spacesResponse,
  terminalResponse,
  terminalMetadata,
  terminalID,
  repositoryChoices,
  projectId,
} from './sodaspaces-api.js';
import type {Space, TerminalMetadata, RepositoryChoices} from './sodaspaces-api.js';
import type {Creation, FactoryWatch, PaneSession, Row, Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';
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
import {
  attentionRows,
  filteredSpaces,
  projectRows,
  rowAttention,
  rowName,
  rows,
} from './sodaspaces-workspace-navigation.js';
import {displayFactoryWatches, factorySection} from './sodaspaces-workspace-factory.js';
type TerminalFactory = typeof mountTerminal;
const validName = (name: string) => [...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name);

function drawerRepositoryPart(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    value.length > 0 &&
    value.length <= 255 &&
    value !== '.' &&
    value !== '..' &&
    !/[\/\\\p{Cc}]/u.test(value)
  );
}

function sodaWorkspaceInit(
  headers: Record<string, string>,
  body: Record<string, unknown> | undefined,
  signal: AbortSignal
): RequestInit {
  return {
    method: body ? 'POST' : 'GET',
    headers,
    ...(body
      ? {
          body: JSON.stringify(body),
        }
      : {}),
    signal,
  };
}
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
    return (
      this.binding?.kind !== 'page' ||
      this.workspaceWidth < (this.layout.sidebar || 220) + this.paneMinimum(focusedPane(this.layout)).width + 6
    );
  }
  private get projection() {
    return projectLayout(
      this.layout,
      {
        x: 0,
        y: 0,
        ...this.canvasSize,
      },
      (pane) => this.paneMinimum(pane),
      this.compact,
      this.maximized
    );
  }
  private locator(slot: Slot) {
    const entry = this.layout.entries.find((entry) => entry.key === slot.key);
    check(entry);
    return entry.locator;
  }
  private paneMinimum(pane: Pane): Minimum {
    const min = this.slots.find((s) => s.key === pane.selected)?.minimum;
    return {
      width: min?.width || Math.ceil(this.cell.width * 56 + 24),
      height: (min?.height || Math.ceil(this.cell.height * 12 + 40)) + this.tabHeight,
    };
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
    this.addEventListener('focusout', (event) => this.menuFocusOut(event), {signal: this.lifetime.signal});
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
    const min = this.paneMinimum(focusedPane(this.layout)).width;
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
  private readonly onWorkspaceKey = (e: KeyboardEvent) => this.workspaceKey(e);
  private readonly onWorkspaceClick = (e: MouseEvent) => this.workspaceClick(e);
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
  private readonly onResizeSidebar = (e: PointerEvent) => this.resizeSidebar(e);
  private readonly onReleasePointer = (e: PointerEvent) => this.releasePointer(e);
  private readonly onSidebarKey = (e: KeyboardEvent) => this.sidebarKey(e);
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
      ${this.paneChrome(focusedPane(this.layout), {x: 0, y: 0, width: this.workspaceWidth, height: this.tabHeight}, true)}
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
  private readonly onDividerPointerMove = (e: PointerEvent) => {
    const key = (e.currentTarget as HTMLElement).dataset.dividerKey;
    const divider = this.projection.dividers.find((item) => item.key === key);
    if (divider) this.dragDivider(e, divider);
  };
  private readonly onDividerKey = (e: KeyboardEvent) => {
    const key = (e.currentTarget as HTMLElement).dataset.dividerKey;
    const divider = this.projection.dividers.find((item) => item.key === key);
    if (divider) this.keyDivider(e, divider);
  };
  private renderPaneDivider(
    divider: {key: string; axis: string; minimum: number; maximum: number; ratio: number} & Area
  ) {
    return html`<div
      class="soda-pane-divider"
      role="separator"
      tabindex="0"
      data-divider-key=${divider.key}
      aria-label="Resize panes"
      aria-orientation=${divider.axis === 'right' ? 'vertical' : 'horizontal'}
      aria-valuemin=${Math.ceil(divider.minimum * 100)}
      aria-valuemax=${Math.floor(divider.maximum * 100)}
      aria-valuenow=${Math.round(divider.ratio * 100)}
      style=${this.rectangle(divider)}
      @pointerdown=${this.onCapturePointer}
      @pointermove=${this.onDividerPointerMove}
      @pointerup=${this.onReleasePointer}
      @keydown=${this.onDividerKey}
    ></div>`;
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
              watches: this.watches,
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
      ${this.renderMoreProjects()} ${this.renderCreateAnotherProject()} ${this.renderEmptySpaces()}
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
          (area) => this.paneChrome(area.pane, area)
        )}
        ${repeat(
          this.projection.dividers,
          (divider) => divider.key,
          (divider) => this.renderPaneDivider(divider)
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
    const openInDrawer = renderOpenInDrawerOption(
      this.binding?.kind,
      workspaceBlocked(this.stale, this.available),
      this.openingDrawer,
      this.selected,
      () => this.openInDrawer()
    );
    const toggleSidebar = renderToggleSidebarOption(this.binding?.kind, () => this.toggleSidebar());
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
        ${openInDrawer} ${toggleSidebar} ${nativeMgmt}
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
  private toggleSidebar() {
    this.layout = {...this.layout, sidebar: this.layout.sidebar === null ? 256 : null};
    this.persist();
  }
  private toggleMaximizedPane() {
    this.closeMenus();
    this.maximized = this.maximized ? undefined : this.layout.focused;
    this.requestUpdate();
  }
  private consolidatePanes() {
    this.closeMenus();
    this.arrange(consolidate(this.layout));
  }
  private showPaneSwitcher() {
    if (this.binding?.kind !== 'page' || panes(this.layout.tree).length <= 1) return false;
    return this.projection.compact || !!this.maximized;
  }
  private onFocusPaneChange(e: Event) {
    if (e.target instanceof HTMLSelectElement) this.focusPane(e.target.value);
  }
  private renderPaneSwitcher() {
    if (!this.showPaneSwitcher()) return '';
    return html`<label
      >Panes (${panes(this.layout.tree).length})
      <select aria-label="Focused pane" .value=${this.layout.focused} @change=${this.onFocusPaneChange}>
        ${panes(this.layout.tree).map((pane, index) => html`<option value=${pane.key} ?selected=${pane.key === this.layout.focused}>Pane ${index + 1}</option>`)}
      </select></label
    >`;
  }
  private renderPaneLayoutExtras() {
    if (panes(this.layout.tree).length <= 1) return '';
    const maximize = this.maximized ? 'Restore panes' : 'Maximize pane';
    return html`<div class="soda-menu-separator"></div>
      <button class="ui button" @click=${() => this.toggleMaximizedPane()}>${maximize}</button>
      <button class="ui button" @click=${() => this.consolidatePanes()}>Consolidate panes</button>`;
  }
  private renderPaneSplitHelp() {
    if (this.canSplit('right') || this.canSplit('below') || this.maximized) return '';
    return html`<p class="soda-menu-help">Make the workspace larger to split this pane.</p>`;
  }
  private renderPaneMenu() {
    if (this.binding?.kind !== 'page') return '';
    // Button labels stay inline: wrapping whitespace text nodes around them
    // breaks exact-text assertions.
    // oxfmt-ignore
    return renderMenu(
      'Pane actions',
      'Pane ⌄',
      html`
        <p class="soda-menu-heading">Pane layout</p>
        <button class="ui button" ?disabled=${!this.canSplit('right')} @click=${() => this.split('right')}>Split right</button
        ><button class="ui button" ?disabled=${!this.canSplit('below')} @click=${() => this.split('below')}>Split below</button>
        ${this.renderPaneLayoutExtras()} ${this.renderPaneSplitHelp()}
      `
    );
  }
  private paneActions() {
    return html`${this.renderPaneSwitcher()} ${this.renderPaneMenu()}`;
  }
  private visibleSlot(slot: Slot) {
    return (
      this.activeSurface &&
      document.visibilityState === 'visible' &&
      this.view === 'terminal' &&
      !slot.unavailable &&
      !slot.host.hidden &&
      !!slot.host.querySelector('.soda-terminal-screen:not([hidden])')
    );
  }
  markViewed(key?: string) {
    const n = this.epoch;
    void this.updateComplete.then(async () => {
      for (const slot of this.slots.filter((slot) => key === undefined || slot.key === key)) {
        await slot.terminal.ready;
        if (!this.live(n) || !this.visibleSlot(slot) || !slot.host.querySelector('.is-connected')) continue;
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
  private paneTabKeys(pane: Pane) {
    return this.binding?.kind === 'native' ? panes(this.layout.tree).flatMap((p) => p.tabs) : pane.tabs;
  }
  private paneSession(key: string): PaneSession[] {
    const entry = this.layout.entries.find((e) => e.key === key);
    const space = this.spaces.find((s) => s.environment.id === entry?.environmentId);
    if (!entry || !space || !space.login || !space.execution_allowed || space.authority_unavailable) return [];
    return [{entry, space, slot: this.slots.find((s) => s.key === key)}];
  }
  private paneAriaOwns(pane: Pane, navigation: boolean) {
    if (navigation || !this.slots.some((s) => s.key === pane.selected && !s.unavailable)) return '';
    return 'soda-owner-' + pane.selected;
  }
  private onTabListDragOver(e: DragEvent) {
    if (this.dragged) e.preventDefault();
  }
  private onTabListDrop(e: DragEvent, paneKey: string) {
    if (!this.dragged) return;
    e.preventDefault();
    this.move(this.dragged, paneKey);
    this.dragged = undefined;
  }
  private sessionRow(entry: LayoutEntry, slot: Slot | undefined): Row {
    return {key: entry.key, entry, ...(slot?.metadata ? {metadata: slot.metadata} : {})};
  }
  private onTabDragStart(event: DragEvent, key: string) {
    if (this.binding?.kind !== 'page') return;
    this.dragged = key;
    event.dataTransfer?.setData('application/x-soda-tab', key);
    this.requestUpdate();
  }
  private onTabDragEnd() {
    this.dragged = undefined;
    this.requestUpdate();
  }
  private onTabDrop(event: DragEvent, paneKey: string, before: string) {
    if (!this.dragged) return;
    event.preventDefault();
    event.stopPropagation();
    this.move(this.dragged, paneKey, before);
    this.dragged = undefined;
  }
  private sessionTabProps(item: PaneSession, keys: string[], pane: Pane) {
    const {entry, space, slot} = item;
    return {
      key: entry.key,
      name: slot ? this.slotName(slot) : rowName({key: entry.key, entry}, this.slots),
      project: this.projectName(space),
      selected: entry.key === pane.selected,
      unread: !!slot?.unread,
      attention: rowAttention(this.navReading(), space, this.sessionRow(entry, slot)),
      select: () => {
        void this.openSaved(entry);
      },
      keydown: (event: KeyboardEvent) => this.tabKey(event, entry.key, keys),
      dragstart: (event: DragEvent) => this.onTabDragStart(event, entry.key),
      dragend: () => this.onTabDragEnd(),
      drop: (event: DragEvent) => this.onTabDrop(event, pane.key, entry.key),
    };
  }
  private renderFocusedPaneActions(pane: Pane, navigation: boolean) {
    if (pane.key !== this.layout.focused || !pane.selected || navigation) return html``;
    return this.paneActions();
  }
  private filterOverflowTabs(e: Event) {
    if (!(e.target instanceof HTMLInputElement) || !(e.currentTarget instanceof HTMLElement)) return;
    const text = e.target.value.toLocaleLowerCase();
    for (const button of e.currentTarget.parentElement?.querySelectorAll('button') || [])
      button.hidden = !button.textContent?.toLocaleLowerCase().includes(text);
  }
  private overflowTabLabel(item: PaneSession) {
    return (item.slot ? this.slotName(item.slot) : 'Saved terminal') + ' · ' + this.projectName(item.space);
  }
  private renderTabOverflow(entries: PaneSession[]) {
    return html`<details class="soda-menu soda-tab-overflow" ?hidden=${entries.length < 2}>
      <summary aria-label="Open tabs" title="Open tabs">Tabs ⌄</summary>
      <div>
        <input
          type="search"
          aria-label="Find an open tab"
          @input=${(e: Event) => this.filterOverflowTabs(e)}
        />${entries.map((item) => html`<button class="ui button" @click=${() => this.openSaved(item.entry)}>${this.overflowTabLabel(item)}</button>`)}
      </div>
    </details>`;
  }
  private showMoveMenu(pane: Pane) {
    return (
      this.binding?.kind === 'page' && !!pane.selected && (panes(this.layout.tree).length > 1 || pane.tabs.length > 1)
    );
  }
  private moveTargetName(entries: PaneSession[], pane: Pane) {
    return entries.find((item) => item.entry.key === pane.selected)?.slot?.metadata?.name || 'terminal';
  }
  private moveToPane(pane: Pane, destination: string) {
    this.closeMenus();
    if (pane.selected) this.move(pane.selected, destination);
  }
  private moveBeforeTab(pane: Pane, before: string) {
    this.closeMenus();
    if (pane.selected) this.move(pane.selected, pane.key, before);
  }
  private beforeTabName(entries: PaneSession[], before: string) {
    return entries.find((item) => item.entry.key === before)?.slot?.metadata?.name || 'saved tab';
  }
  private renderMoveMenu(pane: Pane, entries: PaneSession[]) {
    if (!this.showMoveMenu(pane)) return html``;
    return html`<details class="soda-menu soda-move-menu">
      <summary aria-label="Move terminal to pane" title="Move terminal to pane">Move ⌄</summary>
      <div>
        <p class="soda-menu-heading"><span>Move ${this.moveTargetName(entries, pane)}</span></p>
        ${panes(this.layout.tree).map((destination, i) => html`<button class="ui button" @click=${() => this.moveToPane(pane, destination.key)}>Pane ${i + 1}${destination.key === pane.key ? ' — move to end' : ''}</button>`)}${pane.tabs.filter((key) => key !== pane.selected).map((before) => html`<button class="ui button" @click=${() => this.moveBeforeTab(pane, before)}>Move before ${this.beforeTabName(entries, before)}</button>`)}
      </div>
    </details>`;
  }
  private emptyPaneAttached(pane: Pane, entries: PaneSession[]) {
    return entries.some((e) => e.entry.key === pane.selected && e.slot && !e.slot.unavailable);
  }
  private renderEmptyPane(pane: Pane, area: Area, navigation: boolean, entries: PaneSession[]) {
    if (navigation || (pane.selected && this.emptyPaneAttached(pane, entries))) return html``;
    const message = pane.selected
      ? 'Saved terminal is not attached. Select it after current authorization.'
      : 'No terminal in this pane. Splitting creates no shell.';
    return html`<div
      class="soda-empty-pane"
      style=${`width:${area.width}px;height:${Math.max(0, area.height - this.tabHeight)}px;top:${this.tabHeight}px`}
    >
      <p>${message}</p>
      <button class="ui button" ?disabled=${!this.available || this.stale} @click=${() => this.showSessions(pane.key)}>
        Use existing terminal</button
      ><button
        class="ui button"
        ?disabled=${!this.available || this.stale || this.creating}
        @click=${() => this.newTerminal(pane.key)}
      >
        New terminal here
      </button>
    </div>`;
  }
  private dropEdgeStyle(axis: 'right' | 'below', area: Area) {
    return axis === 'right'
      ? `left:${area.width - 32}px;height:${area.height}px`
      : `top:${area.height - 32}px;width:${area.width}px`;
  }
  private onDropEdgeOver(e: DragEvent, axis: 'right' | 'below', paneKey: string) {
    if (this.dragged && this.canSplit(axis, paneKey)) e.preventDefault();
  }
  private onDropEdgeDrop(e: DragEvent, axis: 'right' | 'below', paneKey: string) {
    if (!this.dragged || !this.canSplit(axis, paneKey)) return;
    e.preventDefault();
    const key = this.dragged,
      next = splitPane(this.layout, paneKey, axis, crypto.randomUUID(), crypto.randomUUID());
    this.arrange(moveTab(next, key, next.focused));
    this.dragged = undefined;
  }
  private renderDropEdges(pane: Pane, area: Area) {
    if (!this.dragged || this.binding?.kind !== 'page') return html``;
    return (['right', 'below'] as const).map(
      (axis) =>
        html`<div
          class=${'soda-drop-edge ' + axis}
          ?hidden=${!this.canSplit(axis, pane.key)}
          style=${this.dropEdgeStyle(axis, area)}
          @dragover=${(e: DragEvent) => this.onDropEdgeOver(e, axis, pane.key)}
          @drop=${(e: DragEvent) => this.onDropEdgeDrop(e, axis, pane.key)}
        >
          Split ${axis}
        </div>`
    );
  }
  private paneChrome(pane: Pane, area: Area, navigation = false) {
    const keys = this.paneTabKeys(pane);
    const entries = keys.flatMap((key) => this.paneSession(key));
    return html`<section
      class="soda-pane-chrome"
      role="group"
      aria-label=${'Pane ' + (panes(this.layout.tree).findIndex((p) => p.key === pane.key) + 1)}
      aria-owns=${this.paneAriaOwns(pane, navigation)}
      data-pane=${pane.key}
      style=${this.rectangle({...area, height: this.tabHeight})}
    >
      <div
        class="soda-workspace-tabs"
        role="tablist"
        aria-label="Terminal sessions"
        @dragover=${(e: DragEvent) => this.onTabListDragOver(e)}
        @drop=${(e: DragEvent) => this.onTabListDrop(e, pane.key)}
      >
        ${repeat(
          entries,
          ({entry}) => entry.key,
          (item) =>
            renderSessionTab(
              this.sessionTabProps(item, keys, pane),
              navigation,
              this.binding?.kind === 'page',
              (event) => this.onTabListDragOver(event)
            )
        )}
      </div>
      ${this.renderFocusedPaneActions(pane, navigation)} ${this.renderTabOverflow(entries)}
      ${this.renderMoveMenu(pane, entries)} ${this.renderEmptyPane(pane, area, navigation, entries)}
      ${this.renderDropEdges(pane, area)}
    </section>`;
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
  private workspaceKey(event: KeyboardEvent) {
    if (event.key !== 'Escape' || !(event.target instanceof HTMLElement)) return;
    const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
    if (menu) {
      event.preventDefault();
      event.stopPropagation();
      menu.open = false;
      menu.querySelector<HTMLElement>('summary')?.focus();
    } else if (this.editing && event.target.closest('.soda-workspace-dialog')) {
      event.preventDefault();
      event.stopPropagation();
      this.editing = null;
      this.restoreFocus();
    }
  }
  private workspaceClick(event: MouseEvent) {
    if (!(event.target instanceof Element)) return;
    const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
    if (!menu) {
      this.closeMenus();
      return;
    }
    if (!event.target.closest('summary')) return;
    for (const other of this.querySelectorAll<HTMLDetailsElement>('.soda-menu[open]'))
      if (other !== menu) other.open = false;
    const summary = menu.querySelector('summary');
    const boundary = menu.closest('.soda-workspace-terminal, .soda-workspace-canvas') || this;
    if (summary)
      menu.style.setProperty(
        '--soda-menu-available-height',
        Math.max(
          0,
          Math.min(window.innerHeight, boundary.getBoundingClientRect().bottom) -
            summary.getBoundingClientRect().bottom -
            2
        ) + 'px'
      );
  }
  private menuFocusOut(event: FocusEvent) {
    if (!(event.target instanceof Element) || !(event.relatedTarget instanceof Node)) return;
    const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
    if (menu && !menu.contains(event.relatedTarget)) menu.open = false;
  }
  private closeMenus() {
    for (const menu of this.querySelectorAll<HTMLDetailsElement>('.soda-menu')) menu.open = false;
  }
  private rememberFocus() {
    this.invoker = document.activeElement instanceof HTMLElement ? document.activeElement : undefined;
    this.closeMenus();
  }
  private restoreFocus() {
    const target = this.invoker,
      active = document.activeElement;
    void this.updateComplete.then(() => {
      if (
        !this.stale &&
        this.activeSurface &&
        (document.activeElement === active || document.activeElement === document.body)
      ) {
        const visible = (node: HTMLElement) =>
          node.isConnected && !node.closest('[hidden], [inert]') && node.getClientRects().length > 0;
        const next =
          target && visible(target)
            ? target
            : [
                ...this.querySelectorAll<HTMLElement>(
                  '.soda-setup-welcome .primary, .soda-workspace-intro h2, .soda-workspace-toolbar button'
                ),
              ].find(visible);
        next?.focus();
      }
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
  private live(n: number) {
    return !this.disposed && !this.stale && this.epoch === n;
  }
  private async readSpacesResponse(response: Response) {
    if (response.ok) return readSodaJSON(response);
    if (response.status === 401 || response.status === 403) {
      this.invalidate();
      this.reconnectRequired = response.status === 401;
    }
    throw Error('Spaces request refused');
  }
  private async api(path: string, body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    check(this.binding && !this.disposed && !this.stale && !signal?.aborted);
    const headers = body ? {'Content-Type': 'application/json'} : {};
    const response = await this.binding.transport.request(
      path.slice('/api/'.length),
      sodaWorkspaceInit(headers, body, signal || this.lifetime.signal)
    );
    return this.readSpacesResponse(response);
  }
  private refreshBlocked() {
    return this.busy || this.stale || this.disposed || !this.binding;
  }
  private beginRefreshRead() {
    this.busy = true;
    this.request?.abort();
    const request = (this.request = new AbortController());
    return request;
  }
  private collectionStatus(complete: boolean, factoryIncomplete: boolean) {
    if (complete) return '';
    return factoryIncomplete
      ? 'We couldn’t load all projects, terminals and factory history. Some may be missing from this list.'
      : 'We couldn’t load all projects and terminals. Some may be missing from this list.';
  }
  private applySpacesCollection(collection: {
    items: Space[];
    complete: boolean;
    nextAfter: string;
    factoryIncomplete: boolean;
  }) {
    this.spaces = collection.items;
    this.complete = collection.complete;
    this.nextAfter = collection.nextAfter;
    this.available = true;
    if (!this.setup && !this.selectedSpace && collection.complete)
      this.project = this.spaces[0]?.environment.repository_id || '';
    this.now = this.observedAt = Date.now();
    this.status = this.collectionStatus(collection.complete, collection.factoryIncomplete);
  }
  private appendSpacesCollection(collection: {
    items: Space[];
    complete: boolean;
    nextAfter: string;
    factoryIncomplete: boolean;
  }) {
    const seen = new Set(this.spaces.map((s) => s.environment.id));
    const grown = [...this.spaces];
    for (const item of collection.items) {
      if (seen.has(item.environment.id)) continue;
      seen.add(item.environment.id);
      grown.push(item);
    }
    this.spaces = grown;
    this.nextAfter = collection.nextAfter;
    // The last page decides: earlier pages always report incomplete
    // while more follow, and degraded rows keep their own inline state.
    this.complete = collection.nextAfter === '' && collection.complete;
    this.now = this.observedAt = Date.now();
    this.status = this.collectionStatus(this.complete, collection.factoryIncomplete);
  }
  private slotLost(slot: Slot, space: Space | undefined, complete: boolean) {
    if (space) return space.authority_unavailable || !space.execution_allowed || space.login !== slot.binding.login;
    return complete;
  }
  private invalidateSlot(slot: Slot) {
    slot.unavailable = true;
    slot.metadata = undefined;
    delete slot.proposedName;
    slot.observation = undefined;
    slot.unread = slot.readRequested = false;
    slot.terminal.invalidate();
  }
  private applySlotMetadata(slot: Slot, metadata: TerminalMetadata) {
    if (metadata.state === 'ended') {
      this.confirmedEnd(slot.key);
      return;
    }
    slot.metadata = metadata;
    slot.observedAt = this.observedAt;
    slot.terminal.setName(metadata.name);
  }
  private existingSlotMetadata(slot: Slot, space: Space | undefined) {
    const locator = this.locator(slot);
    return space?.terminals.find((t) => locator.kind === 'existing' && t.id === locator.id);
  }
  private refreshSlots(complete: boolean) {
    for (const slot of this.slots) {
      const space = this.spaces.find((s) => s.environment.id === slot.binding.environmentId);
      if (this.slotLost(slot, space, complete)) {
        this.invalidateSlot(slot);
        continue;
      }
      const metadata = this.existingSlotMetadata(slot, space);
      if (metadata) this.applySlotMetadata(slot, metadata);
    }
  }
  private async restoreIfNeeded(n: number, signal: AbortSignal) {
    if (this.restored || !this.surfaceVisible || this.closest('[hidden]')) return;
    this.restored = true;
    await this.restoreLocators(n, signal);
  }
  private clearConfigureAfterCreate() {
    if (this.setup === 'configure' && this.selectedSpace?.environment.provisioned) {
      this.setup = null;
      this.setupReturn = undefined;
    }
  }
  private pageJourneyActive() {
    return this.binding?.kind === 'page' && !this.setup && !!this.selectedSpace;
  }
  private leaveJourneyManagement() {
    if (this.view === 'project' && this.managementMode === 'journey') this.view = 'terminal';
  }
  private refreshJourneyManagement() {
    if (this.view === 'project' && this.managementMode === 'journey')
      void this.projects.get(this.project)?.api.refresh();
  }
  private syncPageJourney() {
    if (!this.pageJourneyActive()) return;
    const space = this.selectedSpace!;
    if (this.projectRunnable(space)) {
      this.leaveJourneyManagement();
      return;
    }
    if (this.view === 'terminal') void this.showManagement(this.project, 'journey');
    else this.refreshJourneyManagement();
  }
  private refreshFailed(n: number) {
    if (!this.live(n)) return;
    this.available = false;
    this.complete = false;
    this.status = 'We couldn’t refresh your projects and terminals. The information shown may be out of date.';
  }
  private async finishRefreshSuccess(n: number, signal: AbortSignal) {
    if (!this.storageLoaded) this.loadLayout();
    await this.restoreIfNeeded(n, signal);
    this.clearConfigureAfterCreate();
    this.syncPageJourney();
    this.display();
  }
  private async refreshLive(n: number, request: AbortController) {
    const collection = spacesResponse(await this.api('/api/spaces', undefined, request.signal));
    if (!this.live(n)) return;
    if (this.actor && this.actor.id !== collection.actor.id) {
      this.invalidate();
      return;
    }
    this.actor = collection.actor;
    this.storageKey = 'soda-spaces:v3:' + collection.actor.id;
    this.applySpacesCollection(collection);
    this.refreshSlots(collection.complete);
    await this.updateComplete;
    if (!this.live(n)) return;
    await this.finishRefreshSuccess(n, request.signal);
  }
  async refresh() {
    if (this.refreshBlocked()) return;
    const n = this.epoch;
    const request = this.beginRefreshRead();
    const timer = window.setTimeout(() => request.abort(), 15000);
    try {
      await this.refreshLive(n, request);
    } catch {
      this.refreshFailed(n);
    } finally {
      window.clearTimeout(timer);
      if (this.live(n)) this.busy = false;
    }
  }
  private readonly onMoreProjects = () => this.loadMore();
  private renderMoreProjects() {
    if (!this.nextAfter || this.stale) return '';
    const label = this.busy ? 'Loading…' : 'Show more projects';
    return html`<button class="ui button soda-more-projects" ?disabled=${this.busy} @click=${this.onMoreProjects}>
      ${label}
    </button>`;
  }
  async loadMore() {
    if (this.refreshBlocked() || !this.nextAfter) return;
    const n = this.epoch;
    const request = this.beginRefreshRead();
    const timer = window.setTimeout(() => request.abort(), 15000);
    try {
      const collection = spacesResponse(
        await this.api(`/api/spaces?after=${this.nextAfter}`, undefined, request.signal)
      );
      if (!this.live(n)) return;
      if (this.actor && this.actor.id !== collection.actor.id) {
        this.invalidate();
        return;
      }
      this.appendSpacesCollection(collection);
      this.refreshSlots(this.complete);
      await this.updateComplete;
      if (!this.live(n)) return;
      await this.finishRefreshSuccess(n, request.signal);
    } catch {
      this.refreshFailed(n);
    } finally {
      window.clearTimeout(timer);
      if (this.live(n)) this.busy = false;
    }
  }
  private loadLayout() {
    this.storageLoaded = true;
    try {
      const current = sessionStorage.getItem(this.storageKey);
      if (current !== null) this.layout = parseLayout(current);
    } catch {
      this.layout = emptyLayout(crypto.randomUUID());
      this.storageNotice =
        'Stored layout was reset. Native terminals remain discoverable; no work was created or ended.';
      this.persist();
    }
  }
  private persist() {
    if (!this.storageLoaded || this.stale || this.disposed) return false;
    try {
      sessionStorage.setItem(this.storageKey, serializeLayout(this.layout));
      return true;
    } catch {
      this.storageNotice = 'Live workspace remains usable, but reload restoration could not be saved.';
    }
    return false;
  }
  private drawerOpenBlocked() {
    return this.binding?.kind !== 'page' || this.openingDrawer || !this.available || !this.activeSurface;
  }
  private drawerEntry() {
    const entry = this.layout.entries.find((entry) => entry.key === this.selected);
    const space = this.spaces.find((space) => space.environment.id === entry?.environmentId);
    if (
      !entry ||
      entry.locator.kind !== 'existing' ||
      !space ||
      space.authority_unavailable ||
      !space.execution_allowed ||
      !space.login
    )
      return;
    return {entry, space};
  }
  private drawerStillCurrent(n: number, request: AbortController, entry: LayoutEntry) {
    return this.live(n) && !request.signal.aborted && this.selected === entry.key;
  }
  private assignDrawer(owner: string, name: string) {
    window.location.assign(this.repositoryPrefix() + '/' + encodeURIComponent(owner) + '/' + encodeURIComponent(name));
  }
  private async openDrawerNavigation(entry: LayoutEntry, space: Space, n: number, request: AbortController) {
    const response = object(
      await this.api('/api/environments?repository_id=' + space.environment.repository_id, undefined, request.signal)
    );
    if (!this.drawerStillCurrent(n, request, entry)) return;
    const repository = object(response.repository);
    check(
      repository.id === space.environment.repository_id &&
        drawerRepositoryPart(repository.owner) &&
        drawerRepositoryPart(repository.name)
    );
    if (!this.persist()) {
      this.status = 'Save the workspace before opening it in the drawer; restoration is unavailable.';
      return;
    }
    this.assignDrawer(repository.owner, repository.name);
  }
  private async openInDrawer() {
    if (this.drawerOpenBlocked()) return;
    const selected = this.drawerEntry();
    if (!selected) return;
    const n = this.epoch,
      request = new AbortController();
    const timer = window.setTimeout(() => request.abort(), 15000);
    this.openingDrawer = true;
    this.closeMenus();
    try {
      await this.openDrawerNavigation(selected.entry, selected.space, n, request);
    } catch {
      if (this.live(n))
        this.status = 'Could not open the repository drawer. Your terminal remains here; refresh and try again.';
    } finally {
      window.clearTimeout(timer);
      this.openingDrawer = false;
    }
  }
  private canRestorePane(entry: LayoutEntry | undefined, space: Space | undefined, n: number, signal: AbortSignal) {
    return (
      !!entry &&
      !!space &&
      !space.authority_unavailable &&
      space.execution_allowed &&
      !!space.login &&
      this.live(n) &&
      !signal.aborted
    );
  }
  private async restoreSelectedPane(n: number, signal: AbortSignal, area: {pane: Pane}) {
    const entry = this.layout.entries.find((e) => e.key === area.pane.selected);
    const space = this.spaces.find((s) => s.environment.id === entry?.environmentId);
    if (!this.canRestorePane(entry, space, n, signal)) return;
    const slot = await this.addSlot(space!, entry!);
    if (slot && this.live(n)) {
      this.display();
      await slot.terminal.restore();
    }
  }
  private async restoreLocators(n: number, signal: AbortSignal) {
    await this.updateComplete;
    this.measure();
    await this.updateComplete;
    for (const area of this.projection.panes) await this.restoreSelectedPane(n, signal, area);
    this.persist();
  }
  private slotName(slot: Slot) {
    const locator = this.locator(slot);
    return (
      slot.metadata?.name ||
      slot.proposedName ||
      'Terminal ' + (locator.kind === 'existing' ? locator.id.slice(0, 8) : locator.kind)
    );
  }
  private confirmedEnd(key: string) {
    const slot = this.slots.find((s) => s.key === key),
      entry = this.layout.entries.find((e) => e.key === key);
    if (entry)
      this.spaces = this.spaces.map((space) =>
        space.environment.id !== entry.environmentId
          ? space
          : {
              ...space,
              terminals: space.terminals.filter(
                (metadata) => !(entry.locator.kind === 'existing' && metadata.id === entry.locator.id)
              ),
            }
      );
    this.layout = forgetEntry(this.layout, key);
    this.slots = this.slots.filter((s) => s.key !== key);
    this.maximized = undefined;
    this.persist();
    this.display();
    this.status = 'Native cleanup confirmed for that exact terminal.';
    queueMicrotask(() => {
      slot?.terminal.dispose();
      slot?.host.remove();
    });
  }
  private openSavedBlocked() {
    return this.stale || this.disposed || !this.available || !this.activeSurface;
  }
  private openSavedSpace(entry: LayoutEntry) {
    const space = this.spaces.find((s) => s.environment.id === entry.environmentId);
    if (!space || space.authority_unavailable || !space.execution_allowed || !space.login) return;
    return space;
  }
  private applyOpenLayout(entry: LayoutEntry, destination?: string) {
    this.closeMenus();
    this.layout = destination ? moveTab(this.layout, entry.key, destination) : selectTab(this.layout, entry.key);
    this.view = 'terminal';
    this.choosingPane = undefined;
    if (this.maximized) this.maximized = this.layout.focused;
    this.persist();
  }
  private shouldFocusOpened(slot: Slot, focus: boolean, n: number, invoker: Element | null) {
    return (
      focus && this.live(n) && this.activeSurface && this.selected === slot.key && document.activeElement === invoker
    );
  }
  private async restoreOpenedSlot(slot: Slot, n: number, focus: boolean, invoker: Element | null) {
    if (!this.live(n) || slot.unavailable) return;
    this.display();
    slot.readRequested = true;
    if (this.locator(slot).kind !== 'new') await slot.terminal.restore();
    this.markViewed(slot.key);
    if (this.shouldFocusOpened(slot, focus, n, invoker)) slot.terminal.focus();
  }
  private async openSaved(entry: LayoutEntry, destination?: string, focus = true) {
    const invoker = document.activeElement;
    if (this.openSavedBlocked()) return;
    const space = this.openSavedSpace(entry);
    if (!space) return;
    try {
      this.applyOpenLayout(entry, destination);
      const n = this.epoch,
        slot = await this.addSlot(space, entry);
      await this.updateComplete;
      if (slot) await this.restoreOpenedSlot(slot, n, focus, invoker);
    } catch {
      if (!this.stale) this.status = 'The exact saved terminal could not be selected; no replacement was requested.';
    }
  }
  private identity(space: Space): TerminalContext {
    check(this.actor && this.binding);
    return {
      transport: this.binding.transport,
      expectedUserId: this.actor.id,
      repositoryId: space.environment.repository_id,
      environmentId: space.environment.id,
      login: space.login,
      projectName: this.projectName(space),
    };
  }
  private onTerminalLocator(
    key: string,
    binding: TerminalContext,
    terminal: Slot['terminal'],
    slot: Slot,
    event: Event
  ) {
    if (!(event instanceof CustomEvent) || this.disposed || this.stale) return;
    const value: unknown = event.detail;
    if (value === null) {
      this.confirmedEnd(key);
      return;
    }
    if (!terminalID(value)) return;
    this.commitTerminalLocator(key, binding, terminal, slot, value);
  }
  private commitTerminalLocator(
    key: string,
    binding: TerminalContext,
    terminal: Slot['terminal'],
    slot: Slot,
    value: string
  ) {
    const locator: TerminalLocator = {kind: 'existing', id: value};
    try {
      this.layout = putEntry(this.layout, {key, environmentId: binding.environmentId, locator});
      this.persist();
    } catch {
      terminal.invalidate();
      slot.unavailable = true;
      this.status = 'Conflicting terminal identity was refused; the original locator was preserved.';
    }
  }
  private observationAdmitted(key: string, slot: Slot, event: Event) {
    return (
      event instanceof CustomEvent &&
      !this.stale &&
      !this.disposed &&
      !slot.unavailable &&
      this.layout.entries.some((entry) => entry.key === key)
    );
  }
  private staleObservation(slot: Slot, note: NonNullable<ReturnType<typeof terminalObservation>>) {
    const locator = this.locator(slot);
    if (note.generation < (slot.observation?.generation || 0)) return true;
    return locator.kind === 'existing' && note.id !== locator.id;
  }
  private noteUnread(slot: Slot) {
    if (this.visibleSlot(slot) || slot.unread) return;
    slot.unread = true;
    this.requestUpdate();
  }
  private applyTerminalObservation(key: string, slot: Slot, detail: unknown) {
    const note = terminalObservation(detail);
    if (!note || this.staleObservation(slot, note)) return;
    if (note.kind === 'output') {
      this.noteUnread(slot);
      return;
    }
    slot.observation = note;
    this.requestUpdate();
    if (note.state === 'ready' && slot.readRequested) this.markViewed(key);
  }
  private onTerminalObservation(key: string, slot: Slot, event: Event) {
    if (!this.observationAdmitted(key, slot, event)) return;
    try {
      this.applyTerminalObservation(key, slot, (event as CustomEvent).detail);
    } catch {
      /* Invalid/late observations cannot change a retained owner. */
    }
  }
  private onTerminalMetadata(key: string, slot: Slot, terminal: Slot['terminal'], event: Event) {
    if (
      !(event instanceof CustomEvent) ||
      this.disposed ||
      this.stale ||
      !this.layout.entries.some((e) => e.key === key)
    )
      return;
    const locator = this.locator(slot);
    if (locator.kind !== 'existing') return;
    try {
      const metadata = terminalMetadata(event.detail, slot.binding);
      if (metadata.id !== locator.id) return;
      slot.metadata = metadata;
      slot.observedAt = Date.now();
      terminal.setName(metadata.name);
      this.requestUpdate();
    } catch {
      /* Invalid observation cannot change the binding. */
    }
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
  private geometrySize(detail: unknown) {
    const value = object(detail);
    if (typeof value.width !== 'number' || typeof value.height !== 'number') return;
    if (!Number.isFinite(value.width) || !Number.isFinite(value.height)) return;
    if (value.width <= 0 || value.height <= 0 || value.width > 10000 || value.height > 10000) return;
    return {width: value.width as number, height: value.height as number};
  }
  private onTerminalGeometry(slot: Slot, event: Event) {
    if (!(event instanceof CustomEvent) || this.stale || this.disposed) return;
    const size = this.geometrySize(event.detail);
    if (!size) return;
    slot.minimum = size;
    this.measure();
    this.requestUpdate();
  }
  private onTerminalFocusIn(key: string) {
    const pane = paneFor(this.layout.tree, key);
    if (pane && this.layout.focused !== pane.key) this.focusPane(pane.key);
  }
  private async addSlot(space: Space, entry: LayoutEntry, metadata?: TerminalMetadata): Promise<Slot | undefined> {
    if (this.stale || this.disposed) return;
    let existing = this.slots.find((s) => s.key === entry.key);
    if (existing) return existing;
    const n = this.epoch;
    await this.updateComplete;
    if (!this.live(n) || this.closest('[hidden]')) return;
    existing = this.slots.find((s) => s.key === entry.key);
    if (existing) return existing;
    const layer = this.querySelector('.soda-workspace-owners');
    check(layer);
    const {key, locator} = entry,
      binding = this.identity(space),
      host = document.createElement('div');
    host.className = 'soda-workspace-terminal';
    host.id = 'soda-owner-' + key;
    host.setAttribute('role', 'tabpanel');
    layer.append(host);
    const terminal = this.factory(host, binding, locator);
    const slot: Slot = {
      key,
      binding,
      metadata,
      host,
      terminal,
      unavailable: false,
      unread: false,
      readRequested: false,
      observedAt: metadata ? Date.now() : 0,
      observation: undefined,
    };
    this.slots.push(slot);
    if (metadata) terminal.setName(metadata.name);
    host.addEventListener('soda-terminal-locator', (event) =>
      this.onTerminalLocator(key, binding, terminal, slot, event)
    );
    host.addEventListener('soda-terminal-observation', (event) => this.onTerminalObservation(key, slot, event));
    host.addEventListener('pointerdown', (event) => {
      if (event.isTrusted) this.markViewed(key);
    });
    host.addEventListener('soda-terminal-metadata', (event) => this.onTerminalMetadata(key, slot, terminal, event));
    host.addEventListener('soda-terminal-authority-lost', () => {
      if (!this.disposed) this.invalidate();
    });
    host.addEventListener('soda-terminal-command', (event) => this.onTerminalCommand(key, slot, binding, event));
    host.addEventListener('soda-terminal-geometry', (event) => this.onTerminalGeometry(slot, event));
    host.addEventListener('focusin', () => this.onTerminalFocusIn(key));
    this.display();
    this.requestUpdate();
    return slot;
  }
  private slotHidden(slot: Slot, area: Area | undefined) {
    return this.stale || this.setupScreen || slot.unavailable || this.view !== 'terminal' || !area;
  }
  private applySlotGeometry(slot: Slot, area: Area) {
    slot.host.style.cssText =
      this.rectangle({...area, y: area.y + this.tabHeight, height: Math.max(0, area.height - this.tabHeight)}) +
      `;--soda-terminal-height:${Math.max(0, area.height - this.tabHeight)}px`;
    slot.host.setAttribute('aria-labelledby', 'soda-tab-' + slot.key);
  }
  private displaySlot(slot: Slot, area: Area | undefined) {
    slot.host.hidden = this.slotHidden(slot, area);
    if (area) this.applySlotGeometry(slot, area);
    if (slot.host.hidden || !this.activeSurface) slot.readRequested = false;
    slot.terminal.setVisible(this.surfaceVisible && !slot.host.hidden && !this.closest('[hidden]'));
  }
  private display() {
    const projection = this.projection;
    for (const slot of this.slots)
      this.displaySlot(
        slot,
        projection.panes.find((area) => area.pane.selected === slot.key)
      );
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
  private arrange(layout: WorkspaceLayout) {
    if (this.stale || this.disposed) return;
    this.layout = layout;
    if (!panes(layout.tree).some((p) => p.key === this.maximized)) this.maximized = undefined;
    this.persist();
    this.requestUpdate();
  }
  private focusPane(key: string) {
    if (panes(this.layout.tree).some((p) => p.key === key)) {
      if (this.maximized) this.maximized = key;
      this.arrange({
        ...this.layout,
        focused: key,
      });
    }
  }
  private splitBlocked() {
    return (
      this.stale ||
      this.view !== 'terminal' ||
      this.binding?.kind !== 'page' ||
      !!this.maximized ||
      panes(this.layout.tree).length >= layoutLimit
    );
  }
  private emptyPaneMinimum() {
    return this.paneMinimum({kind: 'pane', key: '', tabs: [], selected: null});
  }
  private splitFits(axis: Split['axis'], area: Area & {pane: Pane}) {
    const min = this.paneMinimum(area.pane),
      empty = this.emptyPaneMinimum();
    if (axis === 'right')
      return area.width >= min.width + empty.width + 6 && area.height >= Math.max(min.height, empty.height);
    return area.height >= min.height + empty.height + 6 && area.width >= Math.max(min.width, empty.width);
  }
  private canSplit(axis: Split['axis'], key = this.layout.focused) {
    if (this.splitBlocked()) return false;
    const area = this.projection.panes.find((a) => a.pane.key === key);
    if (!area || this.projection.compact) return false;
    return this.splitFits(axis, area);
  }
  private split(axis: Split['axis']) {
    if (!this.canSplit(axis)) return;
    this.closeMenus();
    this.arrange(splitPane(this.layout, this.layout.focused, axis, crypto.randomUUID(), crypto.randomUUID()));
  }
  private move(key: string, destination: string, before?: string) {
    try {
      this.arrange(moveTab(this.layout, key, destination, before));
    } catch {
      this.status = 'The pane destination changed; no terminal was replaced.';
    }
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
  private dragDivider(event: PointerEvent, divider: DividerArea) {
    if (!(event.currentTarget instanceof HTMLElement) || !event.currentTarget.hasPointerCapture(event.pointerId))
      return;
    const canvas = this.querySelector('.soda-workspace-canvas')?.getBoundingClientRect();
    if (!canvas) return;
    this.adjustDivider(
      divider,
      ((divider.axis === 'right' ? event.clientX - canvas.x : event.clientY - canvas.y) - divider.origin) /
        divider.extent
    );
  }
  private keyDivider(event: KeyboardEvent, divider: DividerArea) {
    if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    this.adjustDivider(
      divider,
      event.key === 'Home'
        ? divider.minimum
        : event.key === 'End'
          ? divider.maximum
          : divider.ratio + (event.key === 'ArrowLeft' || event.key === 'ArrowUp' ? -0.05 : 0.05)
    );
  }
  private adjustDivider(divider: DividerArea, ratio: number) {
    this.arrange({
      ...this.layout,
      tree: resizeSplit(this.layout.tree, divider.key, Math.max(divider.minimum, Math.min(divider.maximum, ratio))),
    });
  }
  private resizeSidebar(event: PointerEvent) {
    if (event.currentTarget instanceof HTMLElement && event.currentTarget.hasPointerCapture(event.pointerId))
      this.setSidebar(event.clientX - this.getBoundingClientRect().left);
  }
  private sidebarKey(event: KeyboardEvent) {
    if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) {
      event.preventDefault();
      this.setSidebar(
        event.key === 'Home'
          ? 220
          : event.key === 'End'
            ? 360
            : (this.layout.sidebar || 256) + (event.key === 'ArrowLeft' ? -10 : 10)
      );
    }
  }
  private setSidebar(width: number) {
    this.arrange({
      ...this.layout,
      sidebar: Math.max(220, Math.min(360, Math.round(width))),
    });
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
    if (!this.stale && !this.disposed) this.arrange(hideTab(this.layout, slot.key));
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
    const slot = await this.addSlot(space, entry);
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
        this.confirmedEnd(entry.key);
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

import {LitElement, html} from 'lit';
import type {ReactiveController} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {attentionReason, terminalObservation} from './sodaspaces-attention.js';
import type {TerminalObservation} from './sodaspaces-attention.js';
import {renderMenu, renderSessionTab, renderProjectNavigation, renderRename, renderCreation} from './sodaspaces-workspace-view.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {mountTerminal} from './sodaspaces-terminal.js';
import type {TerminalContext, TerminalLocator} from './sodaspaces-terminal.js';
import {emptyLayout, focusedPane, paneFor, selectTab, hideTab, putEntry, forgetEntry, sameLocator, parseLayout, migrateLayout, serializeLayout, layoutLimit, panes, splitPane, moveTab, resizeSplit, consolidate, projectLayout} from './sodaspaces-layout.js';
import type {WorkspaceLayout, LayoutEntry, Pane, Split, Area, Minimum, DividerArea} from './sodaspaces-layout.js';
import {check, id, object, readSodaJSON, sessionResponse, spacesResponse, terminalResponse, terminalMetadata, terminalID} from './sodaspaces-api.js';
import type {Space, TerminalMetadata} from './sodaspaces-api.js';
export type WorkspaceContext = {
  kind: 'native';
  expectedUserId?: string | undefined;
  repositoryId: string;
  pageRepositoryId?: string;
} | {
  kind: 'page';
  expectedUserId: string;
};
type TerminalFactory = typeof mountTerminal;
interface Slot {
  key: string;
  binding: TerminalContext;
  unread: boolean;
  readRequested: boolean;
  observedAt: number;
  observation: TerminalObservation | undefined;
  metadata: TerminalMetadata | undefined;
  minimum?: Minimum;
  unavailable: boolean;
  host: HTMLElement;
  terminal: ReturnType<typeof mountTerminal>;
}
interface Row {
  key: string;
  entry?: LayoutEntry;
  metadata?: TerminalMetadata;
}
interface Creation {
  pane: string;
  environmentId: string;
  name: string;
}
const validName = (name: string) => [...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name);
// Only subscriptions live here. Geometry, layout and minimum-size publication
// remain with SodaSpaces. A disconnected workspace is permanently disposed, so
// retirement must also block late font promises and already queued observer calls.
class WorkspaceMeasurement implements ReactiveController {
  private observer: ResizeObserver | undefined;
  private lifetime: AbortController | undefined;
  private retired = false;

  constructor(private readonly host: LitElement, private readonly measure: () => void) {
    host.addController(this);
  }

  hostUpdated() {
    if (this.retired || this.observer || !this.host.isConnected) return;
    // Connection precedes the first render. Wait for the actual canvas, and
    // retry on a later update if the initial render did not contain it.
    const canvas = this.host.querySelector('.soda-workspace-canvas');
    if (!canvas) return;
    const lifetime = this.lifetime = new AbortController();
    const notify = () => {
      if (!lifetime.signal.aborted && this.host.isConnected) this.measure();
    };
    this.observer = new ResizeObserver(notify);
    this.observer.observe(canvas);
    this.observer.observe(this.host);
    document.fonts.addEventListener('loadingdone', notify, {signal: lifetime.signal});
    window.visualViewport?.addEventListener('resize', notify, {signal: lifetime.signal});
    void document.fonts.ready.then(notify);
  }

  hostDisconnected() {
    this.retire();
  }

  retire() {
    this.retired = true;
    this.observer?.disconnect();
    this.observer = undefined;
    this.lifetime?.abort();
  }
}

// Both surfaces share this owner. Pane chrome is keyed separately; live terminal
// hosts never leave their flat parent. Rendering cannot create/attach/Return.
export class SodaSpaces extends LitElement {
  static properties = {
    spaces: {
      state: true
    }, status: {
      state: true
    }, busy: {
      state: true
    }, layout: {
      state: true
    }, storageNotice: {
      state: true
    }, view: {
      state: true
    }, stale: {
      state: true
    }, search: {
      state: true
    }, thisPage: {
      state: true
    }, creation: {
      state: true
    }, creating: {
      state: true
    }, editing: {
      state: true
    }, attentionOnly: {
      state: true
    }, openingDrawer: {
      state: true
    }
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
  private observedAt = 0;
  private now = Date.now();
  private attentionTimer: number | undefined;
  private binding: WorkspaceContext | undefined;
  private factory: TerminalFactory = mountTerminal;
  private slots: Slot[] = [];
  private projects = new Map<string, {
    host: HTMLElement;
    api: ReturnType<typeof mountProjectControls>;
  }>();
  private project = '';
  private request: AbortController | undefined;
  private renaming = new Set<string>();
  private epoch = 0;
  private disposed = false;
  private restored = false;
  private storageLoaded = false;
  private storageWritable = true;
  private available = false;
  private surfaceVisible = true;
  private storageKey = '';
  private lifetime = new AbortController();
  private canvasSize = {
    width: 0, height: 0
  };
  private workspaceWidth = 0;
  private cell = {
    width: 9, height: 20
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
    this.status = 'Refresh to inspect Spaces.';
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
  private get selected() {
    return focusedPane(this.layout).selected || '';
  }
  private get compact() {
    return this.binding?.kind !== 'page' || this.workspaceWidth < (this.layout.sidebar || 220) + this.paneMinimum(focusedPane(this.layout)).width + 6;
  }
  private get projection() {
    return projectLayout(this.layout, {
      x: 0, y: 0, ...this.canvasSize
    }, pane => this.paneMinimum(pane), this.compact, this.maximized);
  }
  private locator(slot: Slot) {
    const entry = this.layout.entries.find(entry => entry.key === slot.key);
    check(entry);
    return entry.locator;
  }
  private paneMinimum(pane: Pane): Minimum {
    const min = this.slots.find(s => s.key === pane.selected)?.minimum;
    return {
      width: min?.width || Math.ceil(this.cell.width * 56 + 24), height: (min?.height || Math.ceil(this.cell.height * 12 + 40)) + this.tabHeight
    };
  }
  private get tabHeight() {
    return this.workspaceWidth < 800 ? 48 : 40;
  }
  configure(context: WorkspaceContext, factory: TerminalFactory) {
    if (this.binding)
      throw Error('Workspace binding is immutable');
    this.binding = {
      ...context
    };
    this.factory = factory;
    this.storageKey = 'soda-spaces:v2:' + context.expectedUserId;
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
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible')
        this.markViewed();
    }, {
      signal: this.lifetime.signal
    });
    // One mounted-workspace timer, never a navbar poller. GET refresh uses the
    // existing bounded/cancellable caller, and cannot Return or create work.
    this.attentionTimer = window.setInterval(() => {
      if (this.stale || this.disposed)
        return;
      this.now = Date.now();
      this.requestUpdate();
      if (this.activeSurface && document.visibilityState === 'visible' && this.now - this.observedAt >= 60000)
        void this.refresh();
    }, 30000);
    this.addEventListener('soda-project-changed', e => {
      if (!(e instanceof CustomEvent))
        return;
      void this.refresh();
    });
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  protected updated() {
    this.display();
  }
  private measure = () => {
    if (this.disposed || this.stale)
      return;
    const canvas = this.querySelector<HTMLElement>('.soda-workspace-canvas'), cells = this.querySelector('.soda-cell-measure')?.getBoundingClientRect();
    if (!this.clientWidth)
      return;
    const width = canvas?.clientWidth || this.canvasSize.width, height = canvas?.clientHeight || this.canvasSize.height;
    const cell = cells?.width && cells.height ? {
      width: cells.width / 16, height: cells.height
    } : this.cell;
    if (this.workspaceWidth !== this.clientWidth || width !== this.canvasSize.width || height !== this.canvasSize.height || cell.width !== this.cell.width || cell.height !== this.cell.height) {
      this.workspaceWidth = this.clientWidth;
      this.canvasSize = {
        width, height
      };
      this.cell = cell;
      this.requestUpdate();
    }
    const min = this.paneMinimum(focusedPane(this.layout)).width;
    if (min !== this.lastMinimum) {
      this.lastMinimum = min;
      this.dispatchEvent(new CustomEvent('soda-workspace-minimum', {
        bubbles: true, detail: min
      }));
    }
  };
  protected render() {
    const blocked = this.stale || !this.available, connect = this.binding?.kind === 'page' ? 'destination=spaces' : 'repository_id=' + this.binding?.repositoryId;
    const navigation = this.view === 'sessions' || (!this.compact && this.layout.sidebar !== null);
    return html`<section id="sodaspaces-data" class=${'soda-workspace ' + (this.compact ? 'is-compact' : 'is-wide')} aria-busy=${this.busy ? 'true' : 'false'} @keydown=${(e: KeyboardEvent) => this.workspaceKey(e)}>
      <header class="soda-workspace-toolbar">
        <button class="ui button" data-workspace="sessions" ?disabled=${blocked} @click=${() => this.showSessions()}>Sessions</button>
        <button class="ui button" aria-label="New terminal" title="New terminal" ?disabled=${blocked || this.creating} @click=${() => this.newTerminal()}>＋</button>
        ${this.view === 'terminal' ? this.paneActions() : ''}
        ${this.view !== 'terminal' ? html`<button class="ui button" @click=${() => this.back()}>Back to terminal</button>` : ''}
        ${renderMenu('Workspace options', '⋯', html`
          <button class="ui button" ?disabled=${this.busy || this.stale} @click=${() => this.refresh()}>Refresh Spaces</button>
          ${this.binding?.kind === 'page' ? html`<button class="ui button" @click=${() => this.toggleSidebar()}>Toggle sidebar</button>` : ''}
          ${this.binding?.kind === 'native' ? html`<button class="ui button" ?disabled=${blocked} @click=${() => this.showManagement(this.binding?.kind === 'native' ? this.binding.repositoryId : '')}>Repository environment / access</button>` : ''}
        `)}
        ${this.binding?.kind === 'native' ? html`<a href="/-/soda/spaces" aria-label="Open in Spaces" title="Open in Spaces">↗</a>` : ''}
        ${this.binding?.kind === 'page' ? html`<button class="ui button" aria-label="Open in drawer" title=${this.openingDrawer ? 'Opening repository…' : 'Open in drawer'} ?disabled=${blocked || this.openingDrawer || !this.selected} @click=${() => this.openInDrawer()}>${this.workspaceWidth < 800 ? '↘' : this.openingDrawer ? 'Opening repository…' : 'Open in drawer'}</button>` : ''}
        <a ?hidden=${this.available && !this.stale} href=${'/-/soda/login?' + connect + (this.binding?.expectedUserId ? '&expected_user_id=' + this.binding.expectedUserId : '')}>Connect to Soda</a>
      </header>
      <p id="sodaspaces-status" role="status" ?hidden=${!this.status}>${this.status}</p><p role="status" ?hidden=${!this.storageNotice}>${this.storageNotice}</p>
      ${this.binding?.kind === 'native' && this.view !== 'terminal' && !this.stale ? html`<div class="soda-drawer-projection-tabs" style=${`height:${this.tabHeight}px`}>${this.paneChrome(focusedPane(this.layout), {
        x: 0, y: 0, width: this.workspaceWidth, height: this.tabHeight
      }, true)}</div>` : ''}
      <div class=${'soda-workspace-body' + (this.view === 'sessions' ? ' is-navigating' : '')} style=${!this.compact && this.layout.sidebar !== null && this.view !== 'sessions' ? `grid-template-columns:${this.layout.sidebar}px 6px minmax(0,1fr)` : 'grid-template-columns:minmax(0,1fr)'}>
        <nav class="soda-workspace-navigation" aria-label="Projects and sessions" ?hidden=${!navigation || this.stale} @keydown=${(e: KeyboardEvent) => {
        if (e.key === 'Escape' && this.view === 'sessions') {
          e.preventDefault();
          e.stopPropagation();
          this.back();
        }
      }}>
          <label>Find a session <input type="search" .value=${this.search} @input=${(e: Event) => {
        if (e.target instanceof HTMLInputElement)
          this.search = e.target.value;
      }}></label>
          ${this.binding?.kind === 'native' && this.binding.pageRepositoryId ? html`<label><input type="checkbox" .checked=${this.thisPage} @change=${(e: Event) => {
        if (e.target instanceof HTMLInputElement)
          this.thisPage = e.target.checked;
      }}> This page only</label>` : ''}
          <div class="soda-attention-filters" aria-label="Session attention">
            <button class="ui button" aria-pressed=${this.attentionOnly ? 'false' : 'true'} @click=${() => {
        this.attentionOnly = false;
      }}>All</button>
            <button class="ui button" aria-pressed=${this.attentionOnly ? 'true' : 'false'} @click=${() => {
        this.attentionOnly = true;
      }}>Attention (${this.attentionRows().length})</button>
            <button class="ui button" @click=${() => this.nextAttention()}>Next attention</button>
          </div>
          ${repeat(this.filteredSpaces(), space => space.environment.id, space => this.projectRows(space))}
          ${!this.filteredSpaces().length ? html`<p>${this.available ? this.attentionOnly ? 'No matching sessions need attention.' : this.search || this.thisPage ? 'No matches.' : 'No authorized projects available.' : 'Status unavailable.'}</p>` : ''}
        </nav>
        <div class="soda-sidebar-divider" role="separator" tabindex="0" aria-label="Resize project sidebar" aria-orientation="vertical" aria-valuemin="220" aria-valuemax="360" aria-valuenow=${this.layout.sidebar || 256} ?hidden=${this.compact || this.layout.sidebar === null || this.view === 'sessions'} @pointerdown=${(e: PointerEvent) => this.capture(e)} @pointermove=${(e: PointerEvent) => this.resizeSidebar(e)} @pointerup=${(e: PointerEvent) => this.releasePointer(e)} @keydown=${(e: KeyboardEvent) => this.sidebarKey(e)}></div>
        <div class="soda-workspace-work">
          <div class="soda-workspace-canvas" ?hidden=${this.view !== 'terminal' || this.stale}>
            <span class="soda-cell-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span>
            <div class="soda-workspace-chrome">${repeat(this.projection.panes, area => area.pane.key, area => this.paneChrome(area.pane, area))}
              ${repeat(this.projection.dividers, divider => divider.key, divider => html`<div class="soda-pane-divider" role="separator" tabindex="0" aria-label="Resize panes" aria-orientation=${divider.axis === 'right' ? 'vertical' : 'horizontal'} aria-valuemin=${Math.ceil(divider.minimum * 100)} aria-valuemax=${Math.floor(divider.maximum * 100)} aria-valuenow=${Math.round(divider.ratio * 100)} style=${this.rectangle(divider)} @pointerdown=${(e: PointerEvent) => this.capture(e)} @pointermove=${(e: PointerEvent) => this.dragDivider(e, divider)} @pointerup=${(e: PointerEvent) => this.releasePointer(e)} @keydown=${(e: KeyboardEvent) => this.keyDivider(e, divider)}></div>`)}
            </div><div class="soda-workspace-owners"></div>
          </div>
          <div class="soda-workspace-management" ?hidden=${this.view !== 'project' || this.stale}></div>
        </div>
      </div>
      ${this.creation ? this.creationForm(this.creation) : ''}
      ${this.editing ? renderRename(this.editing.name, this.slots.find(s => s.key === this.editing?.key)?.binding.projectName || '', !validName(this.editing.name) || this.renaming.has(this.editing.key) || this.stale, name => {
        if (this.editing)
          this.editing = {
            ...this.editing, name
          };
      }, () => {
        this.editing = null;
        this.restoreFocus();
      }, () => {
        const edit = this.editing, slot = this.slots.find(s => s.key === edit?.key);
        if (slot && edit)
          void this.rename(slot, edit.name);
      }) : ''}
    </section>`;
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
  private paneActions() {
    return html`${this.binding?.kind === 'page' && panes(this.layout.tree).length > 1 && (this.projection.compact || this.maximized) ? html`<label>Panes (${panes(this.layout.tree).length}) <select aria-label="Focused pane" .value=${this.layout.focused} @change=${(e: Event) => {
      if (e.target instanceof HTMLSelectElement)
        this.focusPane(e.target.value);
    }}>${panes(this.layout.tree).map((pane, index) => html`<option value=${pane.key} ?selected=${pane.key === this.layout.focused}>Pane ${index + 1}</option>`)}</select></label>` : ''}
      ${this.binding?.kind === 'page' ? renderMenu('Pane actions', '⊞', html`
        <p>Splits need at least 56 columns × 12 rows in each child.</p>
        <button class="ui button" ?disabled=${!this.canSplit('right')} @click=${() => this.split('right')}>Split right</button><button class="ui button" ?disabled=${!this.canSplit('below')} @click=${() => this.split('below')}>Split below</button>
        <button class="ui button" @click=${() => this.toggleMaximizedPane()}>${this.maximized ? 'Restore panes' : 'Maximize pane'}</button>
        <button class="ui button" @click=${() => this.consolidatePanes()}>Consolidate panes</button>
      `) : ''}`;
  }
  private visibleSlot(slot: Slot) {
    return this.activeSurface && document.visibilityState === 'visible' && this.view === 'terminal' && !slot.unavailable &&
      !slot.host.hidden && !!slot.host.querySelector('.soda-terminal-screen:not([hidden])');
  }
  markViewed(key?: string) {
    const n = this.epoch;
    void this.updateComplete.then(async () => {
      for (const slot of this.slots.filter(slot => key === undefined || slot.key === key)) {
        await slot.terminal.ready;
        if (!this.live(n) || !this.visibleSlot(slot) || !slot.host.querySelector('.is-connected'))
          continue;
        slot.readRequested = false;
        if (slot.unread) {
          slot.unread = false;
          this.requestUpdate();
        }
      }
    });
  }
  private rowAttention(space: Space, row: Row) {
    if (this.stale || !this.available || !space.login || space.authority_unavailable)
      return '';
    if (space.native_unavailable)
      return 'Native status unavailable; refresh';
    const slot = this.slots.find(slot => slot.key === row.key);
    return attentionReason(row.metadata, slot?.metadata ? slot.observedAt : this.observedAt, this.now, slot?.observation?.state);
  }
  private attentionRows() {
    const seen = new Set<string>();
    return this.spaces.flatMap(space => this.rows(space).filter(row => {
      const key = row.metadata?.id || row.key;
      if (!this.rowAttention(space, row) || seen.has(key))
        return false;
      seen.add(key);
      return true;
    }).map(row => ({
      space, row
    })));
  }
  private nextAttention() {
    if (!this.activeSurface || this.stale || !this.available)
      return;
    const rows = this.attentionRows(), index = rows.findIndex(({row}) => row.key === this.selected);
    const next = rows[(index + 1) % rows.length];
    if (!next) {
      this.status = 'No currently authorized sessions need attention.';
      return;
    }
    this.search = '';
    this.thisPage = false;
    if (next.row.entry)
      void this.openSaved(next.row.entry);
    else if (next.row.metadata)
      void this.openExisting(next.space, next.row.metadata);
  }
  private filteredSpaces() {
    const query = this.search.toLocaleLowerCase();
    return this.spaces.filter(space => (!this.attentionOnly || this.rows(space).some(row => this.rowAttention(space, row))) && (!this.thisPage || this.binding?.kind === 'native' && space.environment.repository_id === this.binding.pageRepositoryId) && (!query || this.projectName(space).toLocaleLowerCase().includes(query) || this.rows(space).some(row => this.rowName(row).toLocaleLowerCase().includes(query))));
  }
  private projectName(space: Space) {
    return space.environment.repository || space.environment.name || space.environment.id;
  }
  private rows(space: Space): Row[] {
    if (space.authority_unavailable || !space.login)
      return [];
    const entries = this.layout.entries.filter(e => e.environmentId === space.environment.id), seen = new Set<string>();
    const rows = space.terminals.map(metadata => {
      const entry = entries.find(e => sameLocator(e.locator, {
        kind: 'existing', id: metadata.id
      }) || sameLocator(e.locator, {
        kind: 'pending', requestId: metadata.request_id
      }));
      if (entry)
        seen.add(entry.key);
      const observed = this.slots.find(s => s.key === entry?.key)?.metadata || metadata;
      return {
        key: entry?.key || metadata.id, ...(entry ? {
          entry
        } : {}), metadata: observed
      };
    });
    return [...rows, ...entries.filter(e => !seen.has(e.key)).map(entry => {
      const metadata = this.slots.find(s => s.key === entry.key)?.metadata;
      return {
        key: entry.key, entry, ...(metadata ? {
          metadata
        } : {})
      };
    })];
  }
  private rowName(row: Row) {
    return row.metadata?.name || (row.metadata ? 'Terminal ' + row.metadata.id.slice(0, 8) : row.entry?.locator.kind === 'new' ? 'Unsent terminal' : 'Saved ' + (row.entry?.locator.kind || 'unknown') + ' terminal');
  }
  private projectRows(space: Space) {
    const query = this.search.toLocaleLowerCase(), rows = this.rows(space).filter(row => !query || this.projectName(space).toLocaleLowerCase().includes(query) || this.rowName(row).toLocaleLowerCase().includes(query));
    const shown = this.attentionOnly ? rows.filter(row => this.rowAttention(space, row)) : rows;
    return renderProjectNavigation(this.projectName(space), space.authority_unavailable || space.native_unavailable ? 'Status unavailable' : space.observed?.running ? 'Running' : 'Stopped', shown.map(row => ({
      key: row.key, name: this.rowName(row), unread: !!this.slots.find(slot => slot.key === row.key)?.unread, attention: this.rowAttention(space, row),
      description: row.metadata?.retain_until ? 'Kept until ' + new Date(row.metadata.effective_until * 1000).toLocaleTimeString()
        : row.metadata && row.metadata.state !== 'ready' ? row.metadata.state
          : row.entry && !paneFor(this.layout.tree, row.entry.key) ? 'Hidden; review current lifetime' : row.entry ? 'In this window' : '',
      disabled: this.stale || !this.available || !space.login || space.authority_unavailable,
      select: () => {
        if (row.entry)
          void this.openSaved(row.entry, this.choosingPane);
        else if (row.metadata)
          void this.openExisting(space, row.metadata, this.choosingPane);
      },
    })), this.stale || !this.available, () => {
      void this.showManagement(space.environment.repository_id);
    });
  }
  private paneChrome(pane: Pane, area: Area, navigation = false) {
    const keys = this.binding?.kind === 'native' ? panes(this.layout.tree).flatMap(p => p.tabs) : pane.tabs;
    const entries = keys.flatMap(key => {
      const entry = this.layout.entries.find(e => e.key === key), space = this.spaces.find(s => s.environment.id === entry?.environmentId);
      return entry && space && space.login && !space.authority_unavailable ? [{
        entry, space, slot: this.slots.find(s => s.key === key)
      }] : [];
    });
    return html`<section class="soda-pane-chrome" role="group" aria-label=${'Pane ' + (panes(this.layout.tree).findIndex(p => p.key === pane.key) + 1)} aria-owns=${!navigation && this.slots.some(s => s.key === pane.selected && !s.unavailable) ? 'soda-owner-' + pane.selected : ''} data-pane=${pane.key} style=${this.rectangle({
      ...area, height: this.tabHeight
    })}>
      <div class="soda-workspace-tabs" role="tablist" aria-label="Terminal sessions" @dragover=${(e: DragEvent) => {
        if (this.dragged)
          e.preventDefault();
      }} @drop=${(e: DragEvent) => {
        if (this.dragged) {
          e.preventDefault();
          this.move(this.dragged, pane.key);
          this.dragged = undefined;
        }
      }}>
        ${repeat(entries, ({entry}) => entry.key, ({entry, space, slot}) => renderSessionTab({
        key: entry.key, name: slot ? this.slotName(slot) : this.rowName({
          key: entry.key, entry
        }), project: this.projectName(space),
        selected: entry.key === pane.selected, unread: !!slot?.unread, attention: this.rowAttention(space, {
          key: entry.key, entry, ...(slot?.metadata ? {
            metadata: slot.metadata
          } : {})
        }),
        select: () => {
          void this.openSaved(entry);
        }, keydown: event => this.tabKey(event, entry.key, keys),
        dragstart: event => {
          if (this.binding?.kind === 'page') {
            this.dragged = entry.key;
            event.dataTransfer?.setData('application/x-soda-tab', entry.key);
            this.requestUpdate();
          }
        },
        dragend: () => {
          this.dragged = undefined;
          this.requestUpdate();
        },
        drop: event => {
          if (this.dragged) {
            event.preventDefault();
            event.stopPropagation();
            this.move(this.dragged, pane.key, entry.key);
            this.dragged = undefined;
          }
        },
      }, navigation, this.binding?.kind === 'page', event => {
        if (this.dragged)
          event.preventDefault();
      }))}
      </div>
      <details class="soda-menu soda-tab-overflow"><summary aria-label="Open tabs">⌄</summary><div><input type="search" aria-label="Find an open tab" @input=${(e: Event) => {
        if (e.target instanceof HTMLInputElement && e.currentTarget instanceof HTMLElement) {
          const text = e.target.value.toLocaleLowerCase();
          for (const button of e.currentTarget.parentElement?.querySelectorAll('button') || [])
            button.hidden = !button.textContent?.toLocaleLowerCase().includes(text);
        }
      }}>${entries.map(({entry, space, slot}) => html`<button class="ui button" @click=${() => this.openSaved(entry)}>${slot ? this.slotName(slot) : 'Saved terminal'} · ${this.projectName(space)}</button>`)}</div></details>
      ${this.binding?.kind === 'page' && pane.selected ? html`<details class="soda-menu"><summary aria-label="Move terminal to pane" title="Move terminal to pane">⇢</summary><div>${panes(this.layout.tree).map((destination, i) => html`<button class="ui button" @click=${() => {
        this.closeMenus();
        if (pane.selected)
          this.move(pane.selected, destination.key);
      }}>Pane ${i + 1}${destination.key === pane.key ? ' — move to end' : ''}</button>`)}${pane.tabs.filter(key => key !== pane.selected).map(before => html`<button class="ui button" @click=${() => {
        this.closeMenus();
        if (pane.selected)
          this.move(pane.selected, pane.key, before);
      }}>Move before ${entries.find(item => item.entry.key === before)?.slot?.metadata?.name || 'saved tab'}</button>`)}</div></details>` : ''}
      ${!navigation && (!pane.selected || !entries.some(e => e.entry.key === pane.selected && e.slot && !e.slot.unavailable)) ? html`<div class="soda-empty-pane" style=${`width:${area.width}px;height:${Math.max(0, area.height - this.tabHeight)}px;top:${this.tabHeight}px`}><p>${pane.selected ? 'Saved terminal is not attached. Select it after current authorization.' : 'No terminal in this pane. Splitting creates no shell.'}</p><button class="ui button" ?disabled=${!this.available || this.stale} @click=${() => this.showSessions(pane.key)}>Use existing terminal</button><button class="ui button" ?disabled=${!this.available || this.stale || this.creating} @click=${() => this.newTerminal(pane.key)}>New terminal here</button></div>` : ''}
      ${this.dragged && this.binding?.kind === 'page' ? (['right', 'below'] as const).map(axis => html`<div class=${'soda-drop-edge ' + axis} ?hidden=${!this.canSplit(axis, pane.key)} style=${axis === 'right' ? `left:${area.width - 32}px;height:${area.height}px` : `top:${area.height - 32}px;width:${area.width}px`} @dragover=${(e: DragEvent) => {
        if (this.dragged && this.canSplit(axis, pane.key))
          e.preventDefault();
      }} @drop=${(e: DragEvent) => {
        if (this.dragged && this.canSplit(axis, pane.key)) {
          e.preventDefault();
          const key = this.dragged, next = splitPane(this.layout, pane.key, axis, crypto.randomUUID(), crypto.randomUUID());
          this.arrange(moveTab(next, key, next.focused));
          this.dragged = undefined;
        }
      }}>Split ${axis}</div>`) : ''}
    </section>`;
  }
  private creationForm(draft: Creation) {
    const space = this.spaces.find(s => s.environment.id === draft.environmentId), eligible = space?.login && space.environment.provisioned && !space.authority_unavailable && space.observed?.running === true;
    return renderCreation({
      environmentId: draft.environmentId, name: draft.name,
      projects: this.spaces.map(space => ({
        id: space.environment.id, name: this.projectName(space)
      })),
      context: `${space?.login || 'Join required'} @ ${space ? this.projectName(space) : 'Select a project'}`,
      explanation: eligible ? '' : !space?.login ? 'Join required.' : space.authority_unavailable ? 'Status unavailable.' : 'Environment stopped.',
      busy: this.creating, disabled: this.creating || this.stale || !this.available || !eligible || !validName(draft.name)
    }, environmentId => {
      if (this.creation)
        this.creation = {
          ...this.creation, environmentId
        };
    }, name => {
      if (this.creation)
        this.creation = {
          ...this.creation, name
        };
    }, () => {
      this.creation = null;
      this.restoreFocus();
    }, () => {
      void this.createTerminal();
    });
  }
  private workspaceKey(event: KeyboardEvent) {
    if (event.key !== 'Escape' || !(event.target instanceof HTMLElement))
      return;
    const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
    if (menu) {
      event.preventDefault();
      event.stopPropagation();
      menu.open = false;
      menu.querySelector<HTMLElement>('summary')?.focus();
    }
    else if (this.editing && event.target.closest('.soda-workspace-dialog')) {
      event.preventDefault();
      event.stopPropagation();
      this.editing = null;
      this.restoreFocus();
    }
  }
  private closeMenus() {
    for (const menu of this.querySelectorAll<HTMLDetailsElement>('.soda-menu'))
      menu.open = false;
  }
  private rememberFocus() {
    this.invoker = document.activeElement instanceof HTMLElement ? document.activeElement : undefined;
    this.closeMenus();
  }
  private restoreFocus() {
    const target = this.invoker, active = document.activeElement;
    void this.updateComplete.then(() => {
      if (!this.stale && this.activeSurface && target?.isConnected && (document.activeElement === active || document.activeElement === document.body))
        target.focus();
    });
  }
  private showSessions(pane?: string) {
    this.rememberFocus();
    this.choosingPane = pane;
    this.thisPage = false;
    this.view = 'sessions';
    const active = document.activeElement;
    void this.updateComplete.then(() => {
      if (!this.stale && this.activeSurface && this.view === 'sessions' && (document.activeElement === active || document.activeElement === document.body))
        this.querySelector<HTMLInputElement>('.soda-workspace-navigation input')?.focus();
    });
  }
  private back() {
    this.view = 'terminal';
    this.choosingPane = undefined;
    this.restoreFocus();
    this.markViewed();
  }
  private newTerminal(pane = this.layout.focused) {
    if (this.stale || this.creating || !this.available || !this.activeSurface)
      return;
    this.rememberFocus();
    const selected = this.slots.find(s => s.key === this.selected);
    const native = this.binding?.kind === 'native' ? this.binding.repositoryId : undefined;
    const space = this.spaces.find(s => s.environment.id === selected?.binding.environmentId) || this.spaces.find(s => s.environment.repository_id === native) || this.spaces[0];
    this.creation = {
      pane, environmentId: space?.environment.id || '', name: 'Terminal ' + ((space?.terminals.length || 0) + 1)
    };
    void this.updateComplete.then(() => {
      if (this.creation?.pane === pane && !this.stale && this.surfaceVisible)
        this.querySelector<HTMLElement>('.soda-workspace-dialog select')?.focus();
    });
  }
  private live(n: number) {
    return !this.disposed && !this.stale && this.epoch === n;
  }
  private async api(path: string, body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    const actor = this.binding?.expectedUserId;
    check(actor && !this.disposed && !this.stale && !signal?.aborted);
    const headers: Record<string, string> = {
      'X-Soda-Expected-User-ID': actor
    };
    if (body) {
      const current = sessionResponse(await this.api('/api/session', undefined, signal), location.origin);
      if (current.user.id !== actor) {
        this.invalidate();
        throw Error('Soda actor changed');
      }
      check(!this.stale && !this.disposed && !signal?.aborted);
      headers['X-CSRF-Token'] = current.csrf_token;
      headers['Content-Type'] = 'application/json';
    }
    const response = await fetch('/-/soda' + path, {
      method: body ? 'POST' : 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, ...(body ? {
        body: JSON.stringify(body)
      } : {}), signal: signal || this.lifetime.signal
    });
    if (!response.ok) {
      if (response.status === 401 || response.status === 403)
        this.invalidate();
      throw Error('Spaces request refused');
    }
    return readSodaJSON(response);
  }
  async refresh() {
    if (this.busy || this.stale || this.disposed || !this.binding)
      return;
    if (!this.binding.expectedUserId) {
      this.status = 'Connect through a signed native Forgejo page; no actor was inferred.';
      return;
    }
    const n = this.epoch;
    this.busy = true;
    this.request?.abort();
    const request = this.request = new AbortController();
    const timer = window.setTimeout(() => request.abort(), 15000);
    try {
      const session = sessionResponse(await this.api('/api/session', undefined, request.signal), location.origin);
      if (!this.live(n))
        return;
      if (session.user.id !== this.binding.expectedUserId) {
        this.invalidate();
        return;
      }
      const collection = spacesResponse(await this.api('/api/spaces', undefined, request.signal), session.user.id);
      if (!this.live(n))
        return;
      this.spaces = collection.items;
      this.available = true;
      this.now = this.observedAt = Date.now();
      this.status = collection.complete ? '' : 'Spaces is incomplete or partly unavailable; missing rows are not proof of absence.';
      for (const slot of this.slots) {
        const space = this.spaces.find(s => s.environment.id === slot.binding.environmentId);
        if (space && (space.authority_unavailable || space.login !== slot.binding.login) || !space && collection.complete) {
          slot.unavailable = true;
          slot.metadata = undefined;
          slot.observation = undefined;
          slot.unread = slot.readRequested = false;
          slot.terminal.invalidate();
          continue;
        }
        const locator = this.locator(slot), metadata = space?.terminals.find(t => locator.kind === 'existing' && t.id === locator.id);
        if (metadata) {
          if (metadata.state === 'ended')
            this.confirmedEnd(slot.key);
          else {
            slot.metadata = metadata;
            slot.observedAt = this.observedAt;
            slot.terminal.setName(metadata.name);
          }
        }
      }
      await this.updateComplete;
      if (!this.live(n))
        return;
      if (!this.storageLoaded)
        this.loadLayout();
      if (!this.restored && this.surfaceVisible && !this.closest('[hidden]')) {
        this.restored = true;
        await this.restoreLocators(n, request.signal);
      }
      this.display();
    }
    catch {
      if (this.live(n)) {
        this.available = false;
        this.status = 'Could not confirm Spaces. No creation, join, Start or replacement was requested.';
      }
    }
    finally {
      window.clearTimeout(timer);
      if (this.live(n))
        this.busy = false;
    }
  }
  private loadLayout() {
    this.storageLoaded = true;
    try {
      const current = sessionStorage.getItem(this.storageKey);
      if (current !== null)
        this.layout = parseLayout(current);
      else {
        const legacy = sessionStorage.getItem('soda-spaces:v1:' + this.binding?.expectedUserId);
        if (legacy !== null)
          this.layout = migrateLayout(legacy, () => crypto.randomUUID());
      }
    }
    catch {
      this.storageWritable = false;
      this.storageNotice = 'Stored workspace is invalid, obsolete or inaccessible. No locators were guessed or overwritten.';
    }
  }
  private persist() {
    if (!this.storageWritable || !this.storageLoaded || this.stale || this.disposed)
      return false;
    try {
      sessionStorage.setItem(this.storageKey, serializeLayout(this.layout));
      return true;
    }
    catch {
      this.storageNotice = 'Live workspace remains usable, but reload restoration could not be saved.';
    }
    return false;
  }
  private async openInDrawer() {
    if (this.binding?.kind !== 'page' || this.openingDrawer || !this.available || !this.activeSurface)
      return;
    const entry = this.layout.entries.find(entry => entry.key === this.selected);
    const space = this.spaces.find(space => space.environment.id === entry?.environmentId);
    if (!entry || entry.locator.kind !== 'existing' || !space || space.authority_unavailable || !space.login)
      return;
    const n = this.epoch, request = new AbortController();
    const timer = window.setTimeout(() => request.abort(), 15000);
    this.openingDrawer = true;
    try {
      // Resolve current native names by stable ID; stored project labels may be
      // stale after a rename/transfer. This existing endpoint also checks actor
      // and repository authority. No new route, credential or return URL.
      const response = object(await this.api('/api/environments?repository_id=' + space.environment.repository_id, undefined, request.signal));
      if (!this.live(n) || request.signal.aborted || this.selected !== entry.key)
        return;
      const repository = object(response.repository);
      const part = (value: unknown): value is string => typeof value === 'string' && value.length > 0 && value.length <= 255 && value !== '.' && value !== '..' && !/[\/\\\p{Cc}]/u.test(value);
      check(repository.id === space.environment.repository_id && part(repository.owner) && part(repository.name));
      if (!this.persist()) {
        this.status = 'Save the workspace before opening it in the drawer; restoration is unavailable.';
        return;
      }
      // Native navigation preserves beforeunload. Only actual pagehide detaches;
      // a cancelled departure leaves the existing terminal mounted and attached.
      window.location.assign('/' + encodeURIComponent(repository.owner) + '/' + encodeURIComponent(repository.name) + '#sodaspaces');
    }
    catch {
      if (this.live(n))
        this.status = 'Could not open the repository drawer. Your terminal remains here; refresh and try again.';
    }
    finally {
      window.clearTimeout(timer);
      this.openingDrawer = false;
    }
  }
  private async restoreLocators(n: number, signal: AbortSignal) {
    if (this.storageWritable)
      for (const space of this.spaces) {
        if (!this.live(n) || signal.aborted)
          return;
        try {
          const legacy = sessionStorage.getItem(`soda-terminal:${this.binding?.expectedUserId}:${space.environment.id}`);
          if (legacy === 'pending' || legacy?.startsWith('pending:'))
            this.storageNotice = 'An older creation outcome remains unconfirmed. Its locator was preserved; no session was guessed or creation retried.';
          if (!terminalID(legacy) || this.layout.entries.some(e => sameLocator(e.locator, {
            kind: 'existing', id: legacy
          })) || this.layout.entries.length >= layoutLimit || space.authority_unavailable || !space.login)
            continue;
          const metadata = terminalResponse(await this.api(`/api/environments/${space.environment.id}/terminal-sessions/${legacy}`, undefined, signal), this.identity(space));
          if (!this.live(n) || signal.aborted)
            return;
          if (metadata?.id !== legacy || metadata.state === 'ended')
            continue;
          const entry: LayoutEntry = {
            key: crypto.randomUUID(), environmentId: space.environment.id, locator: {
              kind: 'existing', id: legacy
            }
          };
          this.layout = putEntry(this.layout, entry);
          if (!this.selected)
            this.layout = selectTab(this.layout, entry.key);
        }
        catch { /* Optional legacy import cannot replace known working-set locators. */
        }
      }
    await this.updateComplete;
    this.measure();
    await this.updateComplete;
    for (const area of this.projection.panes) {
      const entry = this.layout.entries.find(e => e.key === area.pane.selected), space = this.spaces.find(s => s.environment.id === entry?.environmentId);
      if (entry && space && !space.authority_unavailable && space.login && this.live(n) && !signal.aborted) {
        const slot = await this.addSlot(space, entry);
        if (slot && this.live(n)) {
          this.display();
          await slot.terminal.restore();
        }
      }
    }
    this.persist();
  }
  private slotName(slot: Slot) {
    const locator = this.locator(slot);
    return slot.metadata?.name || 'Terminal ' + (locator.kind === 'existing' ? locator.id.slice(0, 8) : locator.kind);
  }
  private confirmedEnd(key: string) {
    const slot = this.slots.find(s => s.key === key), entry = this.layout.entries.find(e => e.key === key);
    if (entry)
      this.spaces = this.spaces.map(space => space.environment.id !== entry.environmentId ? space : {
        ...space, terminals: space.terminals.filter(metadata => !(entry.locator.kind === 'existing' && metadata.id === entry.locator.id || entry.locator.kind === 'pending' && metadata.request_id === entry.locator.requestId))
      });
    this.layout = forgetEntry(this.layout, key);
    this.slots = this.slots.filter(s => s.key !== key);
    this.maximized = undefined;
    this.persist();
    this.display();
    this.status = 'Native cleanup confirmed for that exact terminal.';
    queueMicrotask(() => {
      slot?.terminal.dispose();
      slot?.host.remove();
    });
  }
  private async openSaved(entry: LayoutEntry, destination?: string, focus = true) {
    const invoker = document.activeElement;
    if (this.stale || this.disposed || !this.available || !this.activeSurface)
      return;
    const space = this.spaces.find(s => s.environment.id === entry.environmentId);
    if (!space || space.authority_unavailable || !space.login)
      return;
    try {
      this.closeMenus();
      this.layout = destination ? moveTab(this.layout, entry.key, destination) : selectTab(this.layout, entry.key);
      this.view = 'terminal';
      this.choosingPane = undefined;
      if (this.maximized)
        this.maximized = this.layout.focused;
      this.persist();
      const n = this.epoch, slot = await this.addSlot(space, entry);
      await this.updateComplete;
      if (slot && this.live(n) && !slot.unavailable) {
        this.display();
        slot.readRequested = true;
        if (this.locator(slot).kind !== 'new')
          await slot.terminal.restore();
        this.markViewed(slot.key);
        if (focus && this.live(n) && this.activeSurface && this.selected === slot.key && document.activeElement === invoker)
          slot.terminal.focus();
      }
    }
    catch {
      if (!this.stale)
        this.status = 'The exact saved terminal could not be selected; no replacement was requested.';
    }
  }
  private identity(space: Space): TerminalContext {
    return {
      expectedUserId: this.binding?.expectedUserId || '', repositoryId: space.environment.repository_id, environmentId: space.environment.id, login: space.login, projectName: this.projectName(space)
    };
  }
  private async addSlot(space: Space, entry: LayoutEntry, metadata?: TerminalMetadata): Promise<Slot | undefined> {
    if (this.stale || this.disposed)
      return;
    let existing = this.slots.find(s => s.key === entry.key);
    if (existing)
      return existing;
    const n = this.epoch;
    await this.updateComplete;
    if (!this.live(n) || this.closest('[hidden]'))
      return;
    existing = this.slots.find(s => s.key === entry.key);
    if (existing)
      return existing;
    const layer = this.querySelector('.soda-workspace-owners');
    check(layer);
    const {key, locator} = entry, binding = this.identity(space), host = document.createElement('div');
    host.className = 'soda-workspace-terminal';
    host.id = 'soda-owner-' + key;
    host.setAttribute('role', 'tabpanel');
    layer.append(host);
    const terminal = this.factory(host, binding, locator);
    const slot: Slot = {
      key, binding, metadata, host, terminal, unavailable: false, unread: false, readRequested: false, observedAt: metadata ? Date.now() : 0, observation: undefined
    };
    this.slots.push(slot);
    if (metadata)
      terminal.setName(metadata.name);
    host.addEventListener('soda-terminal-locator', event => {
      if (!(event instanceof CustomEvent) || this.disposed || this.stale)
        return;
      const value: unknown = event.detail;
      if (value === null) {
        this.confirmedEnd(key);
        return;
      }
      let locator: TerminalLocator;
      if (typeof value === 'string' && value.startsWith('pending:') && terminalID(value.slice(8)))
        locator = {
          kind: 'pending', requestId: value.slice(8)
        };
      else if (terminalID(value))
        locator = {
          kind: 'existing', id: value
        };
      else
        return;
      try {
        this.layout = putEntry(this.layout, {
          key, environmentId: binding.environmentId, locator
        });
        this.persist();
      }
      catch {
        terminal.invalidate();
        slot.unavailable = true;
        this.status = 'Conflicting terminal identity was refused; the original locator was preserved.';
      }
    });
    host.addEventListener('soda-terminal-observation', event => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed || slot.unavailable || !this.layout.entries.some(entry => entry.key === key))
        return;
      try {
        const note = terminalObservation(event.detail), locator = this.locator(slot);
        if (!note || note.generation < (slot.observation?.generation || 0) ||
          locator.kind === 'existing' && note.id !== locator.id || locator.kind === 'pending' && note.requestId !== locator.requestId)
          return;
        if (note.kind === 'output') {
          if (!this.visibleSlot(slot) && !slot.unread) {
            slot.unread = true;
            this.requestUpdate();
          }
        }
        else {
          slot.observation = note;
          this.requestUpdate();
          if (note.state === 'ready' && slot.readRequested)
            this.markViewed(key);
        }
      }
      catch { /* Invalid/late observations cannot change a retained owner. */
      }
    });
    host.addEventListener('pointerdown', event => {
      if (event.isTrusted)
        this.markViewed(key);
    });
    host.addEventListener('soda-terminal-metadata', event => {
      if (!(event instanceof CustomEvent) || this.disposed || this.stale || !this.layout.entries.some(e => e.key === key))
        return;
      const locator = this.locator(slot);
      if (locator.kind !== 'existing')
        return;
      try {
        const metadata = terminalMetadata(event.detail, slot.binding);
        if (metadata.id === locator.id) {
          slot.metadata = metadata;
          slot.observedAt = Date.now();
          terminal.setName(metadata.name);
          this.requestUpdate();
        }
      }
      catch { /* Invalid observation cannot change the binding. */
      }
    });
    host.addEventListener('soda-terminal-authority-lost', () => {
      if (!this.disposed)
        this.invalidate();
    });
    host.addEventListener('soda-terminal-command', event => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed || slot.host.hidden || slot.unavailable)
        return;
      this.rememberFocus();
      if (event.detail === 'hide')
        void this.hideSlot(slot);
      if (event.detail === 'rename') {
        this.editing = {
          key, name: slot.metadata?.name || ''
        };
        void this.updateComplete.then(() => {
          if (this.editing?.key === key && !this.stale && this.surfaceVisible)
            this.querySelector<HTMLElement>('.soda-workspace-dialog input')?.focus();
        });
      }
      if (event.detail === 'project')
        void this.showManagement(binding.repositoryId);
    });
    host.addEventListener('soda-terminal-geometry', event => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed)
        return;
      const value = object(event.detail);
      if (typeof value.width !== 'number' || typeof value.height !== 'number' || !Number.isFinite(value.width) || !Number.isFinite(value.height) || value.width <= 0 || value.height <= 0 || value.width > 10000 || value.height > 10000)
        return;
      slot.minimum = {
        width: value.width, height: value.height
      };
      this.measure();
      this.requestUpdate();
    });
    host.addEventListener('focusin', () => {
      const pane = paneFor(this.layout.tree, key);
      if (pane && this.layout.focused !== pane.key)
        this.focusPane(pane.key);
    });
    this.display();
    this.requestUpdate();
    return slot;
  }
  private display() {
    const projection = this.projection;
    for (const slot of this.slots) {
      const area = projection.panes.find(area => area.pane.selected === slot.key);
      slot.host.hidden = this.stale || slot.unavailable || this.view !== 'terminal' || !area;
      if (area) {
        slot.host.style.cssText = this.rectangle({
          ...area, y: area.y + this.tabHeight, height: Math.max(0, area.height - this.tabHeight)
        }) + `;--soda-terminal-height:${Math.max(0, area.height - this.tabHeight)}px`;
        slot.host.setAttribute('aria-labelledby', 'soda-tab-' + slot.key);
      }
      if (slot.host.hidden || !this.activeSurface)
        slot.readRequested = false;
      slot.terminal.setVisible(this.surfaceVisible && !slot.host.hidden && !this.closest('[hidden]'));
    }
    for (const [id, project] of this.projects)
      project.host.hidden = this.view !== 'project' || id !== this.project;
  }
  private rectangle(area: Area) {
    return `left:${area.x}px;top:${area.y}px;width:${area.width}px;height:${area.height}px`;
  }
  private arrange(layout: WorkspaceLayout) {
    if (this.stale || this.disposed)
      return;
    this.layout = layout;
    if (!panes(layout.tree).some(p => p.key === this.maximized))
      this.maximized = undefined;
    this.persist();
    this.requestUpdate();
  }
  private focusPane(key: string) {
    if (panes(this.layout.tree).some(p => p.key === key)) {
      if (this.maximized)
        this.maximized = key;
      this.arrange({
        ...this.layout, focused: key
      });
    }
  }
  private canSplit(axis: Split['axis'], key = this.layout.focused) {
    if (this.stale || this.view !== 'terminal' || this.binding?.kind !== 'page' || this.maximized || panes(this.layout.tree).length >= layoutLimit)
      return false;
    const area = this.projection.panes.find(a => a.pane.key === key);
    if (!area || this.projection.compact)
      return false;
    const min = this.paneMinimum(area.pane), empty = this.paneMinimum({
      kind: 'pane', key: '', tabs: [], selected: null
    });
    return axis === 'right' ? area.width >= min.width + empty.width + 6 && area.height >= Math.max(min.height, empty.height) : area.height >= min.height + empty.height + 6 && area.width >= Math.max(min.width, empty.width);
  }
  private split(axis: Split['axis']) {
    if (!this.canSplit(axis))
      return;
    this.closeMenus();
    this.arrange(splitPane(this.layout, this.layout.focused, axis, crypto.randomUUID(), crypto.randomUUID()));
  }
  private move(key: string, destination: string, before?: string) {
    try {
      this.arrange(moveTab(this.layout, key, destination, before));
    }
    catch {
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
    if (!canvas)
      return;
    this.adjustDivider(divider, ((divider.axis === 'right' ? event.clientX - canvas.x : event.clientY - canvas.y) - divider.origin) / divider.extent);
  }
  private keyDivider(event: KeyboardEvent, divider: DividerArea) {
    if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key))
      return;
    event.preventDefault();
    this.adjustDivider(divider, event.key === 'Home' ? divider.minimum : event.key === 'End' ? divider.maximum : divider.ratio + (event.key === 'ArrowLeft' || event.key === 'ArrowUp' ? -0.05 : 0.05));
  }
  private adjustDivider(divider: DividerArea, ratio: number) {
    this.arrange({
      ...this.layout, tree: resizeSplit(this.layout.tree, divider.key, Math.max(divider.minimum, Math.min(divider.maximum, ratio)))
    });
  }
  private resizeSidebar(event: PointerEvent) {
    if (event.currentTarget instanceof HTMLElement && event.currentTarget.hasPointerCapture(event.pointerId))
      this.setSidebar(event.clientX - this.getBoundingClientRect().left);
  }
  private sidebarKey(event: KeyboardEvent) {
    if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) {
      event.preventDefault();
      this.setSidebar(event.key === 'Home' ? 220 : event.key === 'End' ? 360 : (this.layout.sidebar || 256) + (event.key === 'ArrowLeft' ? -10 : 10));
    }
  }
  private setSidebar(width: number) {
    this.arrange({
      ...this.layout, sidebar: Math.max(220, Math.min(360, Math.round(width)))
    });
  }
  private tabKey(event: KeyboardEvent, key: string, keys: string[]) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key))
      return;
    event.preventDefault();
    const index = keys.indexOf(key), next = keys[event.key === 'Home' ? 0 : event.key === 'End' ? keys.length - 1 : (index + (event.key === 'ArrowLeft' ? -1 : 1) + keys.length) % keys.length];
    const entry = this.layout.entries.find(e => e.key === next);
    if (entry) {
      void this.openSaved(entry, undefined, false);
      void this.updateComplete.then(() => {
        if (!this.stale && this.surfaceVisible && this.view === 'terminal' && this.selected === entry.key)
          this.querySelector<HTMLElement>('#soda-tab-' + entry.key)?.focus();
      });
    }
  }
  private async hideSlot(slot: Slot) {
    if (this.stale || this.disposed)
      return;
    this.arrange(hideTab(this.layout, slot.key));
    await slot.terminal.retain();
  }
  private async createTerminal() {
    const draft = this.creation;
    if (!draft || this.creating || !this.available || this.stale || this.disposed || !this.activeSurface || !validName(draft.name))
      return;
    const space = this.spaces.find(p => p.environment.id === draft.environmentId);
    if (!space?.login || !space.environment.provisioned || space.authority_unavailable || space.observed?.running !== true)
      return;
    if (!panes(this.layout.tree).some(p => p.key === draft.pane)) {
      this.status = 'The destination pane changed. Choose New terminal again.';
      return;
    }
    if (this.layout.entries.length >= layoutLimit) {
      this.status = 'The working set is full; uncertain locators cannot be evicted.';
      return;
    }
    this.creating = true;
    try {
      const entry: LayoutEntry = {
        key: crypto.randomUUID(), environmentId: space.environment.id, locator: {
          kind: 'new'
        }
      };
      this.layout = selectTab(putEntry(this.layout, entry), entry.key, draft.pane);
      this.view = 'terminal';
      this.creation = null;
      const slot = await this.addSlot(space, entry);
      if (slot)
        await slot.terminal.open(draft.name);
    }
    finally {
      this.creating = false;
    }
  }
  private async openExisting(space: Space, metadata: TerminalMetadata, destination?: string) {
    if (this.stale || this.disposed || !this.activeSurface || !this.available || space.authority_unavailable)
      return;
    try {
      let entry = this.layout.entries.find(e => sameLocator(e.locator, {
        kind: 'existing', id: metadata.id
      }) || sameLocator(e.locator, {
        kind: 'pending', requestId: metadata.request_id
      }));
      if (entry)
        check(entry.environmentId === space.environment.id);
      else {
        entry = {
          key: crypto.randomUUID(), environmentId: space.environment.id, locator: {
            kind: 'existing', id: metadata.id
          }
        };
        this.layout = putEntry(this.layout, entry);
      }
      if (metadata.state === 'ended') {
        this.confirmedEnd(entry.key);
        return;
      }
      await this.openSaved(entry, destination);
    }
    catch {
      if (!this.stale)
        this.status = 'The exact session could not be opened. No locator was replaced or creation requested.';
    }
  }
  private async showManagement(repositoryId: string) {
    if (!id(repositoryId) || !this.binding?.expectedUserId || !this.available || this.stale || this.disposed || !this.activeSurface)
      return;
    this.rememberFocus();
    this.project = repositoryId;
    this.view = 'project';
    const n = this.epoch;
    await this.updateComplete;
    if (!this.live(n))
      return;
    let project = this.projects.get(repositoryId);
    if (!project) {
      const layer = this.querySelector('.soda-workspace-management');
      check(layer);
      const host = document.createElement('div');
      layer.append(host);
      const api = mountProjectControls(host, {
        repositoryId, expectedUserId: this.binding.expectedUserId, page: this.binding.kind === 'page'
      });
      project = {
        host, api
      };
      this.projects.set(repositoryId, project);
      void api.refresh();
    }
    this.display();
  }
  private async rename(slot: Slot, name: string) {
    if (this.renaming.has(slot.key) || this.stale || this.disposed || !this.activeSurface || !validName(name) || this.editing?.key !== slot.key)
      return;
    const locator = this.locator(slot);
    if (locator.kind !== 'existing')
      return;
    const id = locator.id, n = this.epoch;
    this.renaming.add(slot.key);
    this.requestUpdate();
    const request = new AbortController(), timeout = window.setTimeout(() => request.abort(), 15000);
    try {
      const value = terminalResponse(await this.api(`/api/environments/${slot.binding.environmentId}/terminal-sessions/${id}`, {
        action: 'rename', name
      }, AbortSignal.any([request.signal, this.lifetime.signal])), slot.binding);
      if (this.live(n)) {
        check(value?.id === id);
        slot.metadata = value;
        slot.terminal.setName(value.name);
        if (this.editing?.key === slot.key)
          this.editing = null;
      }
    }
    catch {
      if (this.live(n))
        this.status = 'Rename was not confirmed. No retry was made.';
    }
    finally {
      window.clearTimeout(timeout);
      this.renaming.delete(slot.key);
      this.requestUpdate();
    }
  }
  setVisible(visible: boolean) {
    this.surfaceVisible = visible;
    this.display();
  }
  async retain() {
    await Promise.all(this.slots.map(s => s.terminal.retain()));
  }
  async returnToWork() {
    if (!this.restored)
      await this.refresh();
    const slot = this.slots.find(s => s.key === this.selected);
    if (!this.stale && this.surfaceVisible && this.view === 'terminal' && !this.closest('[hidden]') && slot?.metadata?.retain_until && (slot.metadata.state === 'ready' || slot.metadata.state === 'opening'))
      await slot.terminal.returnToWork();
  }
  invalidate() {
    if (this.stale)
      return;
    this.stale = true;
    this.measurement.retire();
    window.clearInterval(this.attentionTimer);
    ++this.epoch;
    this.request?.abort();
    this.busy = false;
    this.creation = this.editing = null;
    for (const slot of this.slots) {
      slot.metadata = undefined;
      slot.observation = undefined;
      slot.unread = slot.readRequested = false;
      slot.terminal.invalidate();
    }
    for (const project of this.projects.values())
      project.api.invalidate();
    this.spaces = [];
    this.display();
    this.status = 'Page or Soda identity changed. Reload; no action was replayed.';
  }
  dispose() {
    if (this.disposed)
      return;
    this.invalidate();
    this.disposed = true;
    this.lifetime.abort();
    for (const slot of this.slots)
      slot.terminal.dispose();
    for (const project of this.projects.values())
      project.api.dispose();
    this.remove();
  }
}
customElements.define('soda-spaces', SodaSpaces);
export function mountSodaspaces(root: HTMLElement, context: WorkspaceContext, factory: TerminalFactory = mountTerminal) {
  check(root.ownerDocument === document && (context.kind === 'native' ? id(context.repositoryId) && (context.expectedUserId === undefined || id(context.expectedUserId)) && (context.pageRepositoryId === undefined || id(context.pageRepositoryId)) : context.kind === 'page' && id(context.expectedUserId)));
  const box = new SodaSpaces();
  box.configure(context, factory);
  root.append(box);
  return {
    markViewed: () => box.markViewed(), setVisible: (visible: boolean) => box.setVisible(visible), refresh: () => box.refresh(), invalidate: () => box.invalidate(), retain: () => box.retain(), returnToWork: () => box.returnToWork(), get ready() {
      return box.updateComplete;
    }, dispose: () => box.dispose()
  };
}

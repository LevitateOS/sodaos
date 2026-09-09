import {LitElement, html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import {mountProjectControls} from './sodaspaces-project.js';
import {mountTerminal} from './sodaspaces-terminal.js';
import type {TerminalContext, TerminalLocator} from './sodaspaces-terminal.js';
import {emptyLayout, focusedPane, paneFor, selectTab, hideTab, putEntry, forgetEntry, sameLocator, parseLayout, migrateLayout, serializeLayout, layoutLimit, panes, splitPane, moveTab, resizeSplit, consolidate, projectLayout, minimumSize} from './sodaspaces-layout.js';
import type {WorkspaceLayout, LayoutEntry, Pane, Split, Area, Minimum, DividerArea} from './sodaspaces-layout.js';
import {check, id, object, readSodaJSON, sessionResponse, spacesResponse, terminalResponse, terminalMetadata, terminalID} from './sodaspaces-api.js';
import type {Space, TerminalMetadata} from './sodaspaces-api.js';
export type DrawerContext = {kind: 'native'; expectedUserId?: string | undefined; repositoryId: string} | {kind: 'page'; expectedUserId: string};
type TerminalFactory = typeof mountTerminal;
interface Slot {key: string; binding: TerminalContext; metadata?: TerminalMetadata; minimum?: Minimum; host: HTMLElement; terminal: ReturnType<typeof mountTerminal>}
// The only workspace owner on either surface. Terminal hosts occupy one stable,
// flat layer; tab/project changes never move them through a different parent.
export class SodaSpaces extends LitElement {
  static properties = {spaces: {state: true}, status: {state: true}, busy: {state: true}, layout: {state: true}, storageNotice: {state: true}, project: {state: true}, management: {state: true}, stale: {state: true}};
  declare private spaces: Space[];
  declare private status: string;
  declare private busy: boolean;
  declare private layout: WorkspaceLayout;
  declare private storageNotice: string;
  private get selected() {return focusedPane(this.layout).selected || '';}
  private locator(slot: Slot) {const entry = this.layout.entries.find(entry => entry.key === slot.key); check(entry); return entry.locator;}
  private isHidden(slot: Slot) {return !paneFor(this.layout.tree, slot.key);}
  declare private project: string;
  declare private management: boolean;
  declare private stale: boolean;
  private binding: DrawerContext | undefined;
  private factory: TerminalFactory = mountTerminal;
  private slots: Slot[] = [];
  private projects = new Map<string, {host: HTMLElement; api: ReturnType<typeof mountProjectControls>}>();
  private request: AbortController | undefined;
  private epoch = 0;
  private disposed = false;
  private restored = false;
  private storageLoaded = false;
  private storageWritable = true;
  private available = false;
  private storageKey = '';
  private lifetime = new AbortController();
  private canvasSize = {width: 0, height: 0};
  private cell = {width: 9, height: 20};
  private observer: ResizeObserver | undefined;
  private maximized: string | undefined;
  private dragged: string | undefined;
  private get projection() {return projectLayout(this.layout, {x: 0, y: 0, ...this.canvasSize}, pane => this.paneMinimum(pane), this.binding?.kind !== 'page' || this.canvasSize.width < 800, this.maximized);}
  private paneMinimum(pane: Pane): Minimum {const min = this.slots.find(s => s.key === pane.selected)?.minimum; return {width: min?.width || Math.ceil(this.cell.width * 56 + 24), height: (min?.height || Math.ceil(this.cell.height * 12 + 120)) + 40};}
  protected firstUpdated() {
    const canvas = this.querySelector('.soda-workspace-canvas'); if (!canvas) return;
    this.observer = new ResizeObserver(this.measure); this.observer.observe(canvas);
    document.fonts.addEventListener('loadingdone', this.measure, {signal: this.lifetime.signal});
    window.visualViewport?.addEventListener('resize', this.measure, {signal: this.lifetime.signal});
    void document.fonts.ready.then(() => this.measure());
  }
  protected updated() {this.display();}
  private measure = () => {
    if (this.disposed || this.stale) return;
    const canvas = this.querySelector<HTMLElement>('.soda-workspace-canvas'), cells = this.querySelector('.soda-cell-measure')?.getBoundingClientRect();
    if (!canvas?.clientWidth || !canvas.clientHeight) return;
    const width = canvas.clientWidth, height = canvas.clientHeight, cell = cells?.width && cells.height ? {width: cells.width / 16, height: cells.height} : this.cell;
    if (width !== this.canvasSize.width || height !== this.canvasSize.height || cell.width !== this.cell.width || cell.height !== this.cell.height) {this.canvasSize = {width, height}; this.cell = cell; this.requestUpdate();}
  };
  constructor() {super(); this.spaces = []; this.status = 'Refresh to inspect Spaces.'; this.busy = this.management = this.stale = false; this.project = ''; this.storageNotice = ''; this.layout = emptyLayout(crypto.randomUUID());}
  protected createRenderRoot() {return this;}
  configure(context: DrawerContext, factory: TerminalFactory) {
    if (this.binding) throw Error('Workspace binding is immutable'); this.binding = {...context}; this.factory = factory;
    this.storageKey = 'soda-spaces:v2:' + context.expectedUserId;
    window.addEventListener('pagehide', () => this.invalidate(), {signal: this.lifetime.signal});
    window.addEventListener('pageshow', e => {if (e.persisted) this.invalidate();}, {signal: this.lifetime.signal});
    this.addEventListener('soda-project-changed', e => {if (!(e instanceof CustomEvent)) return; const value = object(e.detail); if (value.logout) this.invalidate(); else void this.refresh();});
  }
  disconnectedCallback() {super.disconnectedCallback(); this.dispose();}
  protected render() {
    const blocked = this.busy || this.stale || !this.available, current = this.spaces.find(p => p.environment.id === this.project);
    const connect = this.binding?.kind === 'page' ? 'destination=spaces' : 'repository_id=' + this.binding?.repositoryId;
    return html`<section id="sodaspaces-data" class="soda-workspace" aria-busy=${String(this.busy)}>
      <div class="soda-workspace-toolbar"><button class="ui button" ?disabled=${this.busy || this.stale} @click=${() => this.refresh()}>Refresh Spaces</button>
        <a href="/-/soda/spaces">Open in Spaces</a><a ?hidden=${this.available} href=${'/-/soda/login?' + connect + (this.binding?.expectedUserId ? '&expected_user_id=' + this.binding.expectedUserId : '')}>Connect to Soda</a></div>
      <p id="sodaspaces-status" role="status">${this.status}</p><p role="status" ?hidden=${!this.storageNotice}>${this.storageNotice}</p>
      <div class="soda-workspace-projects" aria-label="Projects">${this.spaces.map(space => html`<button class="ui button" ?disabled=${this.stale} aria-pressed=${String(this.project === space.environment.id)} @click=${() => this.chooseProject(space)}>${space.environment.name || space.environment.repository || space.environment.id}</button>`)}</div>
      <div class="soda-workspace-actions"><button class="ui button" ?disabled=${blocked || !current?.login || !current.environment.provisioned || current.authority_unavailable || current.observed?.running !== true} @click=${() => this.createTerminal()}>New terminal</button>
        <button class="ui button" ?disabled=${blocked || !this.binding?.expectedUserId || (!current && this.binding?.kind !== 'native')} @click=${() => this.showManagement()}>Environment / access</button>
        ${this.binding?.kind === 'native' ? html`<button class="ui button" ?disabled=${blocked || !this.binding.expectedUserId} @click=${() => {this.project = ''; void this.showManagement();}}>Repository on the left</button>` : ''}
        <button class="ui button" ?disabled=${this.stale} @click=${() => {this.management = false; this.display();}}>Terminals</button></div>
      ${this.binding?.kind === 'page' ? html`<div class="soda-pane-actions"><button class="ui button" ?disabled=${!this.canSplit('right')} title="Each child needs 56 columns and 12 rows" @click=${() => this.split('right')}>Split right</button><button class="ui button" ?disabled=${!this.canSplit('below')} title="Each child needs 56 columns and 12 rows" @click=${() => this.split('below')}>Split below</button><button class="ui button" @click=${() => {this.maximized = this.maximized ? undefined : this.layout.focused; this.requestUpdate();}}>${this.maximized ? 'Restore panes' : 'Maximize pane'}</button><button class="ui button" @click=${() => this.arrange(consolidate(this.layout))}>Consolidate panes</button></div>` : ''}
      ${panes(this.layout.tree).length > 1 ? html`<label>Panes (${panes(this.layout.tree).length}) <select aria-label="Focused pane" .value=${this.layout.focused} @change=${(e: Event) => {if (e.target instanceof HTMLSelectElement) this.focusPane(e.target.value);}}>${panes(this.layout.tree).map((pane, index) => html`<option value=${pane.key}>Pane ${index + 1}</option>`)}</select></label>` : ''}
      <div class="soda-workspace-label" ?hidden=${this.management || !this.slots.some(slot => slot.key === this.selected)}>${this.slots.filter(slot => slot.key === this.selected).map(slot => html`<span>${slot.binding.login} · ${slot.binding.environmentId}</span><button class="ui button" ?disabled=${this.stale} @click=${() => this.hideSlot(slot)}>Hide session</button>${slot.metadata ? html`<span>Retained until ${new Date(slot.metadata.effective_until * 1000).toLocaleString()} (hard limit ${new Date(slot.metadata.hard_until * 1000).toLocaleString()})</span>` : html`<span>Lifetime metadata not yet observed. Refresh reads it without extending retention.</span>`}<label>Session name <input maxlength="160" .value=${slot.metadata?.name || ''} ?disabled=${blocked || this.locator(slot).kind !== 'existing'} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) void this.rename(slot, e.target.value);}}></label>`)}</div>
      <div class="soda-workspace-canvas" ?hidden=${this.management}><span class="soda-cell-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span><div class="soda-workspace-chrome">${repeat(this.projection.panes, area => area.pane.key, area => this.paneChrome(area.pane, area))}${this.projection.dividers.map(divider => html`<div class="soda-pane-divider" role="separator" tabindex="0" aria-label="Resize panes" aria-orientation=${divider.axis === 'right' ? 'vertical' : 'horizontal'} aria-valuemin=${Math.ceil(divider.minimum * 100)} aria-valuemax=${Math.floor(divider.maximum * 100)} aria-valuenow=${Math.round(divider.ratio * 100)} style=${this.rectangle(divider)} @pointerdown=${(e: PointerEvent) => {if (e.button === 0 && e.currentTarget instanceof HTMLElement) {e.currentTarget.setPointerCapture(e.pointerId); e.preventDefault();}}} @pointermove=${(e: PointerEvent) => this.dragDivider(e, divider)} @pointerup=${(e: PointerEvent) => {if (e.currentTarget instanceof HTMLElement && e.currentTarget.hasPointerCapture(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId);}} @keydown=${(e: KeyboardEvent) => this.keyDivider(e, divider)}></div>`)}</div><div class="soda-workspace-owners"></div></div><div class="soda-workspace-management" ?hidden=${!this.management}></div>
      <p ?hidden=${this.slots.length !== 0 || this.management}>Select a project and explicitly create a terminal, or open an existing session below. Nothing is created by loading this workspace.</p>
      <div class="soda-workspace-hidden" aria-label="Hidden sessions">${this.slots.filter(slot => this.isHidden(slot)).map(slot => html`<button class="ui button" ?disabled=${this.stale} @click=${() => this.selectSlot(slot)}>Show ${slot.metadata?.name || slot.binding.login + ' · ' + slot.binding.environmentId}</button>`)}</div>
      <div class="soda-workspace-unresolved" ?hidden=${this.management}>${this.layout.entries.filter(entry => !this.slots.some(slot => slot.key === entry.key)).map(entry => {const space = this.spaces.find(s => s.environment.id === entry.environmentId); return space && !space.authority_unavailable && space.login ? html`<button class="ui button" ?disabled=${blocked} @click=${() => this.openSaved(entry)}>Review saved ${entry.locator.kind} terminal in ${space.environment.repository || space.environment.id}</button>` : '';})}</div>
      <div class="soda-workspace-existing" ?hidden=${this.management}>${this.spaces.flatMap(space => space.terminals.map(terminal => html`<button class="ui button" ?disabled=${blocked} @click=${() => this.openExisting(space, terminal)}>${terminal.name || terminal.login + ' · ' + terminal.id.slice(0, 8)} · ${space.environment.name || space.environment.id} · ${terminal.state}</button>`))}</div>
    </section>`;
  }
  private live(n: number) {return !this.disposed && !this.stale && this.epoch === n;}
  private async api(path: string, body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown> {
    const actor = this.binding?.expectedUserId; check(actor && !this.disposed && !this.stale && !signal?.aborted);
    const headers: Record<string, string> = {'X-Soda-Expected-User-ID': actor};
    if (body) {
      const current = sessionResponse(await this.api('/api/session', undefined, signal), location.origin);
      if (current.user.id !== actor) {this.invalidate(); throw Error('Soda actor changed');}
      check(!this.stale && !this.disposed && !signal?.aborted);
      headers['X-CSRF-Token'] = current.csrf_token; headers['Content-Type'] = 'application/json';
    }
    const response = await fetch('/-/soda' + path, {method: body ? 'POST' : 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, ...(body ? {body: JSON.stringify(body)} : {}), ...(signal ? {signal} : {})});
    if (!response.ok) {if (response.status === 401 || response.status === 403) this.invalidate(); throw Error('Spaces request refused');}
    return readSodaJSON(response);
  }
  async refresh() {
    if (this.busy || this.stale || this.disposed || !this.binding) return;
    if (!this.binding.expectedUserId) {this.status = 'Connect through a signed native Forgejo page; no actor was inferred.'; return;}
    const n = this.epoch; this.busy = true; this.request?.abort(); const request = this.request = new AbortController();
    const timer = window.setTimeout(() => request.abort(), 15000);
    try {
      const session = sessionResponse(await this.api('/api/session', undefined, request.signal), location.origin);
      if (!this.live(n)) return;
      if (session.user.id !== this.binding.expectedUserId) {this.invalidate(); return;}
      const collection = spacesResponse(await this.api('/api/spaces', undefined, request.signal), session.user.id); if (!this.live(n)) return;
      this.spaces = collection.items; this.available = true; this.status = collection.complete ? 'Authorized Spaces loaded.' : 'Spaces is incomplete or partly unavailable; missing rows are not proof of absence.';
      const nativeRepository = this.binding.kind === 'native' ? this.binding.repositoryId : undefined;
      if (!this.project) this.project = (nativeRepository ? this.spaces.find(s => s.environment.repository_id === nativeRepository) : this.spaces[0])?.environment.id || '';
      for (const slot of this.slots) {
        const space = this.spaces.find(s => s.environment.id === slot.binding.environmentId);
        if (space?.authority_unavailable) slot.terminal.invalidate();
        const locator = this.locator(slot), metadata = space?.terminals.find(t => locator.kind === 'existing' && t.id === locator.id); if (metadata) {if (metadata.state === 'ended') this.confirmedEnd(slot.key); else slot.metadata = metadata;}
      }
      await this.updateComplete; if (!this.live(n)) return;
      if (!this.storageLoaded) this.loadLayout();
      if (!this.restored && !this.closest('[hidden]')) {this.restored = true; await this.restoreLocators(n, request.signal);}
      this.display();
    } catch {if (this.live(n)) {this.available = false; this.status = 'Could not confirm Spaces. No creation, join, Start or replacement was requested.';}}
    finally {window.clearTimeout(timer); if (this.live(n)) this.busy = false;}
  }
  private loadLayout() {
    this.storageLoaded = true;
    try {
      const current = sessionStorage.getItem(this.storageKey);
      if (current !== null) this.layout = parseLayout(current);
      else {const legacy = sessionStorage.getItem('soda-spaces:v1:' + this.binding?.expectedUserId); if (legacy !== null) this.layout = migrateLayout(legacy, () => crypto.randomUUID());}
    } catch {this.storageWritable = false; this.storageNotice = 'Stored workspace is invalid, obsolete or inaccessible. No locators were guessed or overwritten.';}
  }
  private persist() {
    if (!this.storageWritable || !this.storageLoaded || this.stale || this.disposed) return;
    try {sessionStorage.setItem(this.storageKey, serializeLayout(this.layout));}
    catch {this.storageNotice = 'Live workspace remains usable, but reload restoration could not be saved.';}
  }
  private async restoreLocators(n: number, signal: AbortSignal) {
    // Legacy exact IDs need fresh authorized metadata before import. Never infer
    // absence from a null receipt; preserve old raw storage, including pending.
    if (this.storageWritable) for (const space of this.spaces) {
      if (!this.live(n) || signal.aborted) return;
      try {
        const legacy = sessionStorage.getItem(`soda-terminal:${this.binding?.expectedUserId}:${space.environment.id}`);
        if (legacy === 'pending' || legacy?.startsWith('pending:')) this.storageNotice = 'An older creation outcome remains unconfirmed. Its locator was preserved; no session was guessed or creation retried.';
        if (!terminalID(legacy) || this.layout.entries.some(e => sameLocator(e.locator, {kind: 'existing', id: legacy})) || this.layout.entries.length >= layoutLimit || space.authority_unavailable || !space.login) continue;
        const metadata = terminalResponse(await this.api(`/api/environments/${space.environment.id}/terminal-sessions/${legacy}`, undefined, signal), this.identity(space));
        if (!this.live(n) || signal.aborted) return;
        if (metadata?.id !== legacy || metadata.state === 'ended') continue;
        const entry: LayoutEntry = {key: crypto.randomUUID(), environmentId: space.environment.id, locator: {kind: 'existing', id: legacy}};
        this.layout = putEntry(this.layout, entry);
        if (!this.selected) this.layout = selectTab(this.layout, entry.key);
      } catch { /* Optional legacy import cannot replace known working-set locators. */ }
    }
    await this.updateComplete; this.measure(); await this.updateComplete;
    for (const area of this.projection.panes) {
      const entry = this.layout.entries.find(e => e.key === area.pane.selected), space = this.spaces.find(s => s.environment.id === entry?.environmentId);
      if (entry && space && !space.authority_unavailable && space.login && this.live(n) && !signal.aborted) {
        const slot = await this.addSlot(space, entry);
        if (slot && this.live(n)) {this.display(); await slot.terminal.restore();}
      }
    }
    this.persist();
  }
  private openSlots() {return panes(this.layout.tree).flatMap(pane => pane.tabs).flatMap(key => {const slot = this.slots.find(s => s.key === key); return slot ? [slot] : [];});}
  private slotName(slot: Slot) {const locator = this.locator(slot); return slot.metadata?.name || slot.binding.login + ' · ' + (locator.kind === 'existing' ? locator.id.slice(0, 8) : locator.kind);}
  private confirmedEnd(key: string) {
    const slot = this.slots.find(s => s.key === key);
    this.layout = forgetEntry(this.layout, key); this.slots = this.slots.filter(s => s.key !== key);
    this.persist(); this.display(); this.status = 'Native cleanup confirmed for that exact terminal.';
    // Retire only the acknowledged owner, after its receipt callback returns.
    queueMicrotask(() => {slot?.terminal.dispose(); slot?.host.remove();});
  }
  private async openSaved(entry: LayoutEntry) {
    if (this.busy || this.stale || this.disposed || !this.available) return;
    const space = this.spaces.find(s => s.environment.id === entry.environmentId);
    if (!space || space.authority_unavailable || !space.login) return;
    this.busy = true;
    try {this.layout = selectTab(this.layout, entry.key); this.management = false; const slot = await this.addSlot(space, entry); if (slot) this.selectSlot(slot);}
    finally {this.busy = false;}
  }
  private identity(space: Space): TerminalContext {return {expectedUserId: this.binding?.expectedUserId || '', repositoryId: space.environment.repository_id, environmentId: space.environment.id, login: space.login, projectName: space.environment.name || space.environment.repository || space.environment.id};}
  private async addSlot(space: Space, entry: LayoutEntry, metadata?: TerminalMetadata): Promise<Slot | undefined> {
    if (this.stale || this.disposed) return;
    const existing = this.slots.find(s => s.key === entry.key);
    if (existing) return existing;
    const n = this.epoch; await this.updateComplete; if (!this.live(n) || this.closest('[hidden]')) return;
    const layer = this.querySelector('.soda-workspace-owners'); check(layer);
    const host = document.createElement('div'); host.className = 'soda-workspace-terminal'; layer.append(host);
    const {key, locator} = entry, binding = this.identity(space);
    host.id = 'soda-owner-' + key; host.setAttribute('role', 'tabpanel'); host.setAttribute('aria-label', binding.login + ' in ' + binding.environmentId);
    const terminal = this.factory(host, binding, undefined, locator);
    const slot: Slot = {key, binding, ...(metadata ? {metadata} : {}), host, terminal}; this.slots.push(slot);
    host.addEventListener('soda-terminal-locator', event => {
      if (!(event instanceof CustomEvent) || this.disposed || this.stale) return; const value: unknown = event.detail;
      if (value === null) {this.confirmedEnd(key); return;}
      let locator: TerminalLocator;
      if (typeof value === 'string' && value.startsWith('pending:') && terminalID(value.slice(8))) locator = {kind: 'pending', requestId: value.slice(8)};
      else if (terminalID(value)) locator = {kind: 'existing', id: value};
      else return;
      try {this.layout = putEntry(this.layout, {key, environmentId: binding.environmentId, locator}); this.persist();}
      catch {terminal.invalidate(); this.status = 'Conflicting terminal identity was refused; the original locator was preserved.';}
    });
    host.addEventListener('soda-terminal-metadata', event => {
      if (!(event instanceof CustomEvent) || this.disposed || this.stale || !this.layout.entries.some(e => e.key === key)) return;
      const locator = this.locator(slot); if (locator.kind !== 'existing') return;
      try {const metadata = terminalMetadata(event.detail, slot.binding); if (metadata.id === locator.id) {slot.metadata = metadata; this.requestUpdate();}} catch { /* Invalid observation cannot change the binding. */ }
    });
    host.addEventListener('soda-terminal-geometry', event => {
      if (!(event instanceof CustomEvent) || this.stale || this.disposed) return;
      const value = object(event.detail); if (typeof value.width !== 'number' || typeof value.height !== 'number' || !Number.isFinite(value.width) || !Number.isFinite(value.height) || value.width <= 0 || value.height <= 0 || value.width > 10000 || value.height > 10000) return;
      slot.minimum = {width: value.width, height: value.height}; this.requestUpdate();
    });
    host.addEventListener('focusin', () => {const pane = paneFor(this.layout.tree, key); if (pane && this.layout.focused !== pane.key) this.focusPane(pane.key);});
    this.display(); return slot;
  }
  private selectSlot(slot: Slot) {if (this.stale || this.disposed) return; this.layout = selectTab(this.layout, slot.key); this.project = slot.binding.environmentId; this.management = false; this.requestUpdate(); this.display(); this.persist(); if (this.locator(slot).kind !== 'new') void slot.terminal.restore();}
  private display() {
    const projection = this.projection;
    for (const slot of this.slots) {
      const area = projection.panes.find(area => area.pane.selected === slot.key);
      slot.host.hidden = this.stale || this.management || !area;
      if (area) {slot.host.style.cssText = this.rectangle({...area, y: area.y + 40, height: Math.max(0, area.height - 40)}); slot.host.setAttribute('aria-labelledby', 'soda-tab-' + slot.key);}
      slot.terminal.setVisible(!slot.host.hidden && !this.closest('[hidden]'));
    }
    for (const [id, project] of this.projects) project.host.hidden = !this.management || id !== (this.spaces.find(s => s.environment.id === this.project)?.environment.repository_id || (this.binding?.kind === 'native' ? this.binding.repositoryId : ''));
  }
  private rectangle(area: Area) {return `left:${area.x}px;top:${area.y}px;width:${area.width}px;height:${area.height}px`;}
  private paneChrome(pane: Pane, area: Area) {
    const slots = this.binding?.kind === 'native' ? this.openSlots() : pane.tabs.flatMap(key => {const slot = this.slots.find(s => s.key === key); return slot ? [slot] : [];});
    return html`<section class="soda-pane-chrome" data-pane=${pane.key} style=${this.rectangle({...area, height: 40})}>
      <div class="soda-workspace-tabs" role="tablist" aria-label="Terminal sessions" @dragover=${(e: DragEvent) => {if (this.dragged) e.preventDefault();}} @drop=${(e: DragEvent) => {if (this.dragged) {e.preventDefault(); this.move(this.dragged, pane.key); this.dragged = undefined;}}}>${repeat(slots, slot => slot.key, slot => html`<button id=${'soda-tab-' + slot.key} role="tab" class="ui button" draggable=${this.binding?.kind === 'page' ? 'true' : 'false'} @dragstart=${(e: DragEvent) => {if (this.binding?.kind === 'page') {this.dragged = slot.key; e.dataTransfer?.setData('application/x-soda-tab', slot.key);}}} @dragend=${() => {this.dragged = undefined;}} aria-controls=${'soda-owner-' + slot.key} tabindex=${slot.key === pane.selected ? '0' : '-1'} @keydown=${(e: KeyboardEvent) => this.tabKey(e, slot)} aria-selected=${String(slot.key === pane.selected)} @click=${() => this.selectSlot(slot)}>${this.slotName(slot)}</button>`)}</div>
      ${!pane.tabs.length ? html`<button class="ui button" @click=${() => {this.focusPane(pane.key); this.querySelector<HTMLElement>('.soda-workspace-existing button')?.focus();}}>Use existing terminal</button>` : ''}
      ${this.binding?.kind === 'page' && pane.selected ? html`<select aria-label="Move terminal to pane" .value=${pane.key} @change=${(e: Event) => {if (e.target instanceof HTMLSelectElement && pane.selected) this.move(pane.selected, e.target.value);}}>${panes(this.layout.tree).map((p, index) => html`<option value=${p.key}>Pane ${index + 1}</option>`)}</select>` : ''}
    </section>`;
  }
  private arrange(layout: WorkspaceLayout) {if (this.stale || this.disposed) return; this.layout = layout; if (!panes(layout.tree).some(p => p.key === this.maximized)) this.maximized = undefined; this.persist(); this.requestUpdate();}
  private focusPane(key: string) {if (panes(this.layout.tree).some(p => p.key === key)) this.arrange({...this.layout, focused: key});}
  private canSplit(axis: Split['axis']) {
    if (this.stale || this.management || this.binding?.kind !== 'page' || this.maximized || panes(this.layout.tree).length >= layoutLimit) return false;
    const area = this.projection.panes.find(a => a.pane.key === this.layout.focused); if (!area || this.projection.compact) return false;
    const min = this.paneMinimum(area.pane), empty = this.paneMinimum({kind: 'pane', key: '', tabs: [], selected: null});
    return axis === 'right' ? area.width >= min.width + empty.width + 6 && area.height >= Math.max(min.height, empty.height) : area.height >= min.height + empty.height + 6 && area.width >= Math.max(min.width, empty.width);
  }
  private split(axis: Split['axis']) {if (!this.canSplit(axis)) return; this.arrange(splitPane(this.layout, this.layout.focused, axis, crypto.randomUUID(), crypto.randomUUID()));}
  private move(key: string, destination: string) {try {this.arrange(moveTab(this.layout, key, destination));} catch {this.status = 'The pane destination changed; no terminal was replaced.';}}
  private dragDivider(event: PointerEvent, divider: DividerArea) {
    if (!(event.currentTarget instanceof HTMLElement) || !event.currentTarget.hasPointerCapture(event.pointerId)) return;
    const canvas = this.querySelector('.soda-workspace-canvas')?.getBoundingClientRect(); if (!canvas) return;
    this.adjustDivider(divider, ((divider.axis === 'right' ? event.clientX - canvas.x : event.clientY - canvas.y) - divider.origin) / divider.extent);
  }
  private keyDivider(event: KeyboardEvent, divider: DividerArea) {
    if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault(); this.adjustDivider(divider, event.key === 'Home' ? divider.minimum : event.key === 'End' ? divider.maximum : divider.ratio + (event.key === 'ArrowLeft' || event.key === 'ArrowUp' ? -0.05 : 0.05));
  }
  private adjustDivider(divider: DividerArea, ratio: number) {this.arrange({...this.layout, tree: resizeSplit(this.layout.tree, divider.key, Math.max(divider.minimum, Math.min(divider.maximum, ratio)))});}
  private tabKey(event: KeyboardEvent, slot: Slot) {
    const pane = paneFor(this.layout.tree, slot.key), tabs = this.openSlots().filter(s => this.binding?.kind === 'native' || pane?.tabs.includes(s.key)), index = tabs.indexOf(slot);
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault(); const next = tabs[event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowLeft' ? -1 : 1) + tabs.length) % tabs.length];
    if (next) {this.selectSlot(next); void this.updateComplete.then(() => this.querySelector<HTMLElement>('[aria-controls="soda-owner-' + next.key + '"]')?.focus());}
  }
  private async hideSlot(slot: Slot) {
    if (this.stale || this.disposed) return;
    this.layout = hideTab(this.layout, slot.key);
    this.display(); this.persist(); this.requestUpdate();
    await slot.terminal.retain();
  }
  private chooseProject(space: Space) {this.project = space.environment.id; void this.showManagement();}
  private async createTerminal() {
    if (this.busy || !this.available || this.stale) return;
    const space = this.spaces.find(p => p.environment.id === this.project); if (!space || !space.login || space.authority_unavailable || space.observed?.running !== true) return;
    if (this.layout.entries.length >= layoutLimit) {this.status = 'The working set is full; uncertain locators cannot be evicted.'; return;}
    this.busy = true;
    try {const entry: LayoutEntry = {key: crypto.randomUUID(), environmentId: space.environment.id, locator: {kind: 'new'}}; this.layout = selectTab(putEntry(this.layout, entry), entry.key); this.management = false; const slot = await this.addSlot(space, entry); if (slot) await slot.terminal.open();}
    finally {this.busy = false;}
  }
  private async openExisting(space: Space, metadata: TerminalMetadata) {
    if (this.busy || this.stale || this.disposed || space.authority_unavailable) return;
    this.busy = true;
    try {
      let entry = this.layout.entries.find(e => sameLocator(e.locator, {kind: 'existing', id: metadata.id}) || sameLocator(e.locator, {kind: 'pending', requestId: metadata.request_id}));
      if (entry) check(entry.environmentId === space.environment.id);
      else {entry = {key: crypto.randomUUID(), environmentId: space.environment.id, locator: {kind: 'existing', id: metadata.id}}; this.layout = putEntry(this.layout, entry);}
      if (metadata.state === 'ended') {this.confirmedEnd(entry.key); return;}
      this.layout = selectTab(this.layout, entry.key); this.management = false;
      const slot = await this.addSlot(space, entry, metadata); if (slot) this.selectSlot(slot);
    } catch {if (!this.stale) this.status = 'The exact session could not be opened. No locator was replaced or creation requested.';}
    finally {this.busy = false;}
  }
  private async showManagement() {
    const space = this.spaces.find(s => s.environment.id === this.project), repositoryId = space?.environment.repository_id || (this.binding?.kind === 'native' ? this.binding.repositoryId : undefined);
    if (!repositoryId || !this.binding?.expectedUserId || !this.available || this.stale || this.disposed) return;
    const key = repositoryId; this.management = true; const n = this.epoch;
    await this.updateComplete; if (!this.live(n)) return;
    let project = this.projects.get(key);
    if (!project) {
      const layer = this.querySelector('.soda-workspace-management'); check(layer);
      const host = document.createElement('div'); layer.append(host);
      const api = mountProjectControls(host, {repositoryId, expectedUserId: this.binding?.expectedUserId, page: this.binding?.kind === 'page'}); project = {host, api}; this.projects.set(key, project); void api.refresh();
    }
    this.display();
  }
  private async rename(slot: Slot, name: string) {
    if (this.busy || this.stale || this.disposed || this.closest('[hidden]') || this.management || this.selected !== slot.key) return;
    const locator = this.locator(slot); if (locator.kind !== 'existing') return;
    const id = locator.id, n = this.epoch; this.busy = true; const request = new AbortController(), timeout = window.setTimeout(() => request.abort(), 15000);
    try {const value = terminalResponse(await this.api(`/api/environments/${slot.binding.environmentId}/terminal-sessions/${id}`, {action: 'rename', name}, request.signal), slot.binding); if (this.live(n)) {check(value?.id === id); slot.metadata = value; this.requestUpdate();}}
    catch {if (this.live(n)) this.status = 'Rename was not confirmed. No retry was made.';} finally {window.clearTimeout(timeout); if (this.live(n)) this.busy = false;}
  }
  async retain() {await Promise.all(this.slots.map(s => s.terminal.retain()));}
  async returnToWork() {if (!this.restored) await this.refresh(); const slot = this.slots.find(s => s.key === this.selected); if (!this.stale && !this.closest('[hidden]')) await slot?.terminal.returnToWork();}
  invalidate() {if (this.stale) return; this.stale = true; ++this.epoch; this.request?.abort(); this.busy = false; for (const slot of this.slots) slot.terminal.invalidate(); for (const project of this.projects.values()) project.api.invalidate(); this.spaces = []; this.display(); this.status = 'Page or Soda identity changed. Reload; no action was replayed.';}
  dispose() {if (this.disposed) return; this.invalidate(); this.disposed = true; this.observer?.disconnect(); this.lifetime.abort(); for (const slot of this.slots) slot.terminal.dispose(); for (const project of this.projects.values()) project.api.dispose(); this.remove();}
}
customElements.define('soda-spaces', SodaSpaces);
export function mountSodaspaces(root: HTMLElement, context: DrawerContext, factory: TerminalFactory = mountTerminal) {
  check(root.ownerDocument === document && (context.kind === 'native' ? id(context.repositoryId) && (context.expectedUserId === undefined || id(context.expectedUserId)) : context.kind === 'page' && id(context.expectedUserId)));
  const box = new SodaSpaces(); box.configure(context, factory); root.append(box);
  return {refresh: () => box.refresh(), invalidate: () => box.invalidate(), retain: () => box.retain(), returnToWork: () => box.returnToWork(), get ready() {return box.updateComplete;}, dispose: () => box.dispose()};
}

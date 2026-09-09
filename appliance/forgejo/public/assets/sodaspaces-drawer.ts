import {LitElement, html} from 'lit';
import {mountProjectControls} from './sodaspaces-project.js';
import {mountTerminal} from './sodaspaces-terminal.js';
import type {TerminalContext, TerminalLocator} from './sodaspaces-terminal.js';
import {check, id, projectId, object, readSodaJSON, sessionResponse, spacesResponse, terminalResponse, terminalMetadata, terminalID} from './sodaspaces-api.js';
import type {Space, TerminalMetadata} from './sodaspaces-api.js';
export type DrawerContext = {kind: 'native'; expectedUserId?: string | undefined; repositoryId: string} | {kind: 'page'; expectedUserId: string};
type TerminalFactory = typeof mountTerminal;
interface Slot {key: string; binding: TerminalContext; locator: TerminalLocator; metadata?: TerminalMetadata; host: HTMLElement; terminal: ReturnType<typeof mountTerminal>; closed: boolean; hidden: boolean}
interface Saved {environmentId: string; id?: string; requestId?: string; hidden?: boolean}
// The only workspace owner on either surface. Terminal hosts occupy one stable,
// flat layer; tab/project changes never move them through a different parent.
export class SodaSpaces extends LitElement {
  static properties = {spaces: {state: true}, status: {state: true}, busy: {state: true}, selected: {state: true}, project: {state: true}, management: {state: true}, stale: {state: true}};
  declare private spaces: Space[];
  declare private status: string;
  declare private busy: boolean;
  declare private selected: string;
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
  private unresolved: Saved[] = [];
  private storageWritable = true;
  private available = false;
  private storageKey = '';
  private lifetime = new AbortController();
  constructor() {super(); this.spaces = []; this.status = 'Refresh to inspect Spaces.'; this.busy = this.management = this.stale = false; this.selected = this.project = '';}
  protected createRenderRoot() {return this;}
  configure(context: DrawerContext, factory: TerminalFactory) {
    if (this.binding) throw Error('Workspace binding is immutable'); this.binding = {...context}; this.factory = factory;
    this.storageKey = 'soda-spaces:v1:' + context.expectedUserId;
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
      <p id="sodaspaces-status" role="status">${this.status}</p>
      <div class="soda-workspace-projects" aria-label="Projects">${this.spaces.map(space => html`<button class="ui button" ?disabled=${this.stale} aria-pressed=${String(this.project === space.environment.id)} @click=${() => this.chooseProject(space)}>${space.environment.name || space.environment.repository || space.environment.id}</button>`)}</div>
      <div class="soda-workspace-actions"><button class="ui button" ?disabled=${blocked || !current?.login || !current.environment.provisioned || current.authority_unavailable || current.observed?.running !== true} @click=${() => this.createTerminal()}>New terminal</button>
        <button class="ui button" ?disabled=${blocked || !this.binding?.expectedUserId || (!current && this.binding?.kind !== 'native')} @click=${() => this.showManagement()}>Environment / access</button>
        ${this.binding?.kind === 'native' ? html`<button class="ui button" ?disabled=${blocked || !this.binding.expectedUserId} @click=${() => {this.project = ''; void this.showManagement();}}>Repository on the left</button>` : ''}
        <button class="ui button" ?disabled=${this.stale} @click=${() => {this.management = false; this.display();}}>Terminals</button></div>
      <div class="soda-workspace-tabs" role="tablist" aria-label="Terminal sessions">${this.slots.filter(slot => !slot.hidden).map(slot => html`<button role="tab" class="ui button" aria-controls=${'soda-owner-' + slot.key} tabindex=${slot.key === this.selected ? '0' : '-1'} @keydown=${(e: KeyboardEvent) => this.tabKey(e, slot)} aria-selected=${String(this.selected === slot.key && !this.management)} @click=${() => this.selectSlot(slot)}>${slot.metadata?.name || slot.binding.login + ' · ' + (slot.locator.kind === 'existing' ? slot.locator.id.slice(0, 8) : slot.locator.kind === 'pending' ? 'pending' : 'new')}${slot.closed ? ' · ended' : ''}</button>`)}</div>
      <div class="soda-workspace-label" ?hidden=${this.management || !this.slots.some(slot => slot.key === this.selected)}>${this.slots.filter(slot => slot.key === this.selected).map(slot => html`<span>${slot.binding.login} · ${slot.binding.environmentId}</span><button class="ui button" ?disabled=${this.stale} @click=${() => this.hideSlot(slot)}>Hide session</button>${slot.metadata ? html`<span>Retained until ${new Date(slot.metadata.effective_until * 1000).toLocaleString()} (hard limit ${new Date(slot.metadata.hard_until * 1000).toLocaleString()})</span>` : html`<span>Lifetime metadata not yet observed. Refresh reads it without extending retention.</span>`}<label>Session name <input maxlength="160" .value=${slot.metadata?.name || ''} ?disabled=${blocked || slot.locator.kind !== 'existing' || slot.closed} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) void this.rename(slot, e.target.value);}}></label>`)}</div>
      <div class="soda-workspace-owners" ?hidden=${this.management}></div><div class="soda-workspace-management" ?hidden=${!this.management}></div>
      <p ?hidden=${this.slots.length !== 0 || this.management}>Select a project and explicitly create a terminal, or open an existing session below. Nothing is created by loading this workspace.</p>
      <div class="soda-workspace-hidden" aria-label="Hidden sessions">${this.slots.filter(slot => slot.hidden).map(slot => html`<button class="ui button" ?disabled=${this.stale} @click=${() => this.selectSlot(slot)}>Show ${slot.metadata?.name || slot.binding.login + ' · ' + slot.binding.environmentId}</button>`)}</div>
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
    const n = ++this.epoch; this.busy = true; this.request?.abort(); const request = this.request = new AbortController();
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
        const metadata = space?.terminals.find(t => slot.locator.kind === 'existing' && t.id === slot.locator.id); if (metadata) slot.metadata = metadata;
      }
      await this.updateComplete; if (!this.live(n)) return;
      if (!this.restored && !this.closest('[hidden]')) {this.restored = true; await this.restoreLocators(n, request.signal);}
      this.display();
    } catch {if (this.live(n)) {this.available = false; this.status = 'Could not confirm Spaces. No creation, join, Start or replacement was requested.';}}
    finally {window.clearTimeout(timer); if (this.live(n)) this.busy = false;}
  }
  private saved(): Saved[] {
    try {
      const text = sessionStorage.getItem(this.storageKey); if (!text) return [];
      check(text.length <= 16384); const value = object(JSON.parse(text)); check(value.version === 1 && Array.isArray(value.entries) && value.entries.length <= 64);
      return value.entries.map((raw: unknown) => {const item = object(raw); check(projectId(item.environmentId) && (terminalID(item.id) && item.requestId === undefined || terminalID(item.requestId) && item.id === undefined)); check(item.hidden === undefined || typeof item.hidden === 'boolean'); return {environmentId: item.environmentId, ...(item.hidden === true ? {hidden: true} : {}), ...(typeof item.id === 'string' ? {id: item.id} : {requestId: String(item.requestId)})};});
    } catch {this.storageWritable = false; this.status = 'Stored workspace is invalid or obsolete. No locators were guessed or overwritten.'; return [];}
  }
  private persist() {
    if (!this.storageWritable) return;
    const entries = [...this.unresolved, ...this.slots.flatMap((slot): Saved[] => slot.closed || slot.locator.kind === 'new' ? [] : [{environmentId: slot.binding.environmentId, ...(slot.hidden ? {hidden: true} : {}), ...(slot.locator.kind === 'existing' ? {id: slot.locator.id} : {requestId: slot.locator.requestId})}])];
    if (entries.length > 64) {this.status = 'The saved working set is full; live use remains available without guaranteed reload restoration.'; return;}
    try {sessionStorage.setItem(this.storageKey, JSON.stringify({version: 1, entries}));} catch {this.status = 'Live workspace remains usable, but reload restoration could not be saved.';}
  }
  private async restoreLocators(n: number, signal: AbortSignal) {
    const saved = this.saved(), legacyIDs = new Set<string>();
    // Import exact legacy IDs only for already authorized associations. Never guess pending.
    for (const space of this.spaces) {try {const legacy = sessionStorage.getItem(`soda-terminal:${this.binding?.expectedUserId}:${space.environment.id}`); if (legacy === 'pending' || legacy?.startsWith('pending:')) this.status = 'An older creation outcome remains unconfirmed. Its locator was preserved; no session was guessed or creation retried.'; if (terminalID(legacy) && !saved.some(s => s.id === legacy) && saved.length < 64) {saved.push({environmentId: space.environment.id, id: legacy}); legacyIDs.add(legacy);}} catch { /* optional locator storage */ }}
    this.unresolved = saved.filter(item => !item.id || !legacyIDs.has(item.id));
    for (const locator of saved) {
      if (!this.live(n) || this.closest('[hidden]')) return;
      const space = this.spaces.find(p => p.environment.id === locator.environmentId); if (!space || space.authority_unavailable || !space.login) continue;
      const binding = this.identity(space), path = locator.id ? 'terminal-sessions/' + locator.id : 'terminal-attempts/' + locator.requestId;
      try {
        const metadata = terminalResponse(await this.api(`/api/environments/${locator.environmentId}/${path}`, undefined, signal), binding); if (!this.live(n)) return;
        if (!metadata && locator.id && legacyIDs.has(locator.id)) continue;
        let slot: Slot | undefined;
        if (metadata) {check(locator.id ? metadata.id === locator.id : metadata.request_id === locator.requestId); if (metadata.state !== 'ended') slot = await this.addSlot(space, {kind: 'existing', id: metadata.id}, metadata);}
        else slot = await this.addSlot(space, locator.id ? {kind: 'existing', id: locator.id} : {kind: 'pending', requestId: String(locator.requestId)});
        if (slot && locator.hidden) {slot.hidden = true; if (this.selected === slot.key) this.selected = ''; this.display();}
        if (slot || metadata?.state === 'ended') this.unresolved = this.unresolved.filter(item => item !== locator);
      } catch {if (this.live(n)) this.status = 'Some saved locators could not be authorized; no replacement was created.';}
    }
    this.persist();
  }
  private identity(space: Space): TerminalContext {return {expectedUserId: this.binding?.expectedUserId || '', repositoryId: space.environment.repository_id, environmentId: space.environment.id, login: space.login, projectName: space.environment.name || space.environment.repository || space.environment.id};}
  private async addSlot(space: Space, locator: TerminalLocator, metadata?: TerminalMetadata): Promise<Slot | undefined> {
    if (this.slots.length >= 64 || this.stale || this.disposed) return;
    const existing = this.slots.find(s => locator.kind === 'existing' && s.locator.kind === 'existing' && s.locator.id === locator.id);
    if (existing) {this.selectSlot(existing); return existing;}
    const n = this.epoch; await this.updateComplete; if (!this.live(n) || this.closest('[hidden]')) return;
    const layer = this.querySelector('.soda-workspace-owners'); check(layer);
    const host = document.createElement('div'); host.className = 'soda-workspace-terminal'; layer.append(host);
    const key = crypto.randomUUID(), binding = this.identity(space);
    host.id = 'soda-owner-' + key; host.setAttribute('role', 'tabpanel'); host.setAttribute('aria-label', binding.login + ' in ' + binding.environmentId);
    const terminal = this.factory(host, binding, undefined, locator);
    const slot: Slot = {key, binding, locator, ...(metadata ? {metadata} : {}), host, terminal, closed: false, hidden: false}; this.slots.push(slot);
    host.addEventListener('soda-terminal-locator', event => {
      if (!(event instanceof CustomEvent) || this.disposed || this.stale) return; const value: unknown = event.detail;
      if (value === null) slot.closed = true;
      else if (typeof value === 'string' && value.startsWith('pending:') && terminalID(value.slice(8))) slot.locator = {kind: 'pending', requestId: value.slice(8)};
      else if (terminalID(value)) slot.locator = {kind: 'existing', id: value};
      else return;
      this.persist(); this.requestUpdate();
    });
    host.addEventListener('soda-terminal-metadata', event => {
      if (!(event instanceof CustomEvent) || this.disposed || this.stale || slot.locator.kind !== 'existing') return;
      try {const metadata = terminalMetadata(event.detail, slot.binding); if (metadata.id === slot.locator.id) {slot.metadata = metadata; this.requestUpdate();}} catch { /* Invalid observation cannot change the binding. */ }
    });
    this.selectSlot(slot); return slot;
  }
  private selectSlot(slot: Slot) {if (this.stale || this.disposed) return; slot.hidden = false; this.selected = slot.key; this.project = slot.binding.environmentId; this.management = false; this.requestUpdate(); this.display(); if (slot.locator.kind !== 'new' && !slot.closed) void slot.terminal.restore();}
  private display() {for (const slot of this.slots) slot.host.hidden = this.stale || slot.hidden || this.management || slot.key !== this.selected; for (const [id, project] of this.projects) project.host.hidden = !this.management || id !== (this.spaces.find(s => s.environment.id === this.project)?.environment.repository_id || (this.binding?.kind === 'native' ? this.binding.repositoryId : ''));}
  private tabKey(event: KeyboardEvent, slot: Slot) {
    const tabs = this.slots.filter(s => !s.hidden), index = tabs.indexOf(slot);
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault(); const next = tabs[event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowLeft' ? -1 : 1) + tabs.length) % tabs.length];
    if (next) {this.selectSlot(next); void this.updateComplete.then(() => this.querySelector<HTMLElement>('[aria-controls="soda-owner-' + next.key + '"]')?.focus());}
  }
  private async hideSlot(slot: Slot) {
    if (this.stale || this.disposed) return;
    slot.hidden = true; if (this.selected === slot.key) this.selected = this.slots.find(s => !s.hidden)?.key || '';
    this.display(); this.persist(); this.requestUpdate();
    if (slot.closed) {slot.terminal.dispose(); slot.host.remove(); this.slots = this.slots.filter(s => s !== slot); this.requestUpdate();}
    else await slot.terminal.retain();
  }
  private chooseProject(space: Space) {this.project = space.environment.id; void this.showManagement();}
  private async createTerminal() {
    if (this.busy || !this.available || this.stale) return;
    const space = this.spaces.find(p => p.environment.id === this.project); if (!space || !space.login || space.authority_unavailable || space.observed?.running !== true) return;
    this.busy = true; try {const slot = await this.addSlot(space, {kind: 'new'}); if (slot) await slot.terminal.open();} finally {this.busy = false;}
  }
  private async openExisting(space: Space, metadata: TerminalMetadata) {if (this.busy || this.stale) return; this.busy = true; try {await this.addSlot(space, {kind: 'existing', id: metadata.id}, metadata);} finally {this.busy = false;}}
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
    if (this.busy || this.stale || this.disposed || this.closest('[hidden]') || this.management || this.selected !== slot.key || slot.locator.kind !== 'existing') return;
    const id = slot.locator.id, n = this.epoch; this.busy = true; const request = new AbortController(), timeout = window.setTimeout(() => request.abort(), 15000);
    try {const value = terminalResponse(await this.api(`/api/environments/${slot.binding.environmentId}/terminal-sessions/${id}`, {action: 'rename', name}, request.signal), slot.binding); if (this.live(n)) {check(value?.id === id); slot.metadata = value; this.requestUpdate();}}
    catch {if (this.live(n)) this.status = 'Rename was not confirmed. No retry was made.';} finally {window.clearTimeout(timeout); if (this.live(n)) this.busy = false;}
  }
  async retain() {await Promise.all(this.slots.filter(s => !s.closed).map(s => s.terminal.retain()));}
  async returnToWork() {if (!this.restored) await this.refresh(); const slot = this.slots.find(s => s.key === this.selected); if (!this.stale && !this.closest('[hidden]')) await slot?.terminal.returnToWork();}
  invalidate() {if (this.stale) return; this.stale = true; ++this.epoch; this.request?.abort(); this.busy = false; for (const slot of this.slots) slot.terminal.invalidate(); for (const project of this.projects.values()) project.api.invalidate(); this.spaces = []; this.display(); this.status = 'Page or Soda identity changed. Reload; no action was replayed.';}
  dispose() {if (this.disposed) return; this.invalidate(); this.disposed = true; this.lifetime.abort(); for (const slot of this.slots) slot.terminal.dispose(); for (const project of this.projects.values()) project.api.dispose(); this.remove();}
}
customElements.define('soda-spaces', SodaSpaces);
export function mountSodaspaces(root: HTMLElement, context: DrawerContext, factory: TerminalFactory = mountTerminal) {
  check(root.ownerDocument === document && (context.kind === 'native' ? id(context.repositoryId) && (context.expectedUserId === undefined || id(context.expectedUserId)) : context.kind === 'page' && id(context.expectedUserId)));
  const box = new SodaSpaces(); box.configure(context, factory); root.append(box);
  return {refresh: () => box.refresh(), invalidate: () => box.invalidate(), retain: () => box.retain(), returnToWork: () => box.returnToWork(), get ready() {return box.updateComplete;}, dispose: () => box.dispose()};
}

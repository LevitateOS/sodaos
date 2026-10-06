import {terminalObservation} from './sodaspaces-attention.js';
import {check, object} from './sodaspaces-api.js';
import {terminalID, terminalMetadata} from './sodaspaces-terminal-response.js';
import type {Space} from './sodaspaces-inventory-response.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import type {Area, LayoutEntry, PaneArea, WorkspaceLayout} from './sodaspaces-layout.js';
import {paneFor, putEntry} from './sodaspaces-layout.js';
import type {TerminalContext, TerminalLocator} from './sodaspaces-terminal.js';
import type {Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';

export interface HostsInput {
  binding: WorkspaceContext | undefined;
  isStale: () => boolean;
  isDisposed: () => boolean;
  readEpoch: () => number;
  readView: () => 'terminal' | 'sessions' | 'project';
  readSetupScreen: () => boolean;
  readTabHeight: () => number;
  readSurfaceVisible: () => boolean;
  isActiveSurface: () => boolean;
  readSlots: () => Slot[];
  readLayout: () => WorkspaceLayout;
  readActor: () => {id: string; login: string} | undefined;
  readProjectionPanes: () => PaneArea[];
  readUpdateComplete: () => Promise<boolean>;
  readFactory: () => (host: HTMLElement, binding: TerminalContext, locator: TerminalLocator) => Slot['terminal'];
  findMountLayer: () => Element | null;
  concealed: () => boolean;
  live: (n: number) => boolean;
  persist: () => boolean;
  requestUpdate: () => void;
  display: () => void;
  measure: () => void;
  invalidate: () => void;
  focusPane: (key: string) => void;
  markViewed: (key?: string) => void;
  locator: (slot: Slot) => TerminalLocator;
  projectName: (space: Space) => string;
  confirmedEnd: (key: string) => void;
  onTerminalCommand: (key: string, slot: Slot, binding: TerminalContext, event: Event) => void;
  setLayout: (layout: WorkspaceLayout) => void;
  setStatus: (status: string) => void;
  pushSlot: (slot: Slot) => void;
  rectangle: (area: Area) => string;
}

export function visibleSlot(input: HostsInput, slot: Slot) {
  return (
    input.isActiveSurface() &&
    document.visibilityState === 'visible' &&
    input.readView() === 'terminal' &&
    !slot.unavailable &&
    !slot.host.hidden &&
    !!slot.host.querySelector('.soda-terminal-screen:not([hidden])')
  );
}

export function identity(input: HostsInput, space: Space): TerminalContext {
  const actor = input.readActor();
  check(actor && input.binding);
  return {
    transport: input.binding.transport,
    expectedUserId: actor.id,
    repositoryId: space.environment.repository_id,
    environmentId: space.environment.id,
    login: space.login,
    projectName: input.projectName(space),
  };
}

export function onTerminalLocator(
  input: HostsInput,
  key: string,
  binding: TerminalContext,
  terminal: Slot['terminal'],
  slot: Slot,
  event: Event
) {
  if (!(event instanceof CustomEvent) || input.isDisposed() || input.isStale()) return;
  const value: unknown = event.detail;
  if (value === null) {
    input.confirmedEnd(key);
    return;
  }
  if (!terminalID(value)) return;
  commitTerminalLocator(input, key, binding, terminal, slot, value);
}

export function commitTerminalLocator(
  input: HostsInput,
  key: string,
  binding: TerminalContext,
  terminal: Slot['terminal'],
  slot: Slot,
  value: string
) {
  const locator: TerminalLocator = {kind: 'existing', id: value};
  try {
    input.setLayout(putEntry(input.readLayout(), {key, environmentId: binding.environmentId, locator}));
    input.persist();
  } catch {
    terminal.invalidate();
    slot.unavailable = true;
    input.setStatus('Conflicting terminal identity was refused; the original locator was preserved.');
  }
}

export function observationAdmitted(input: HostsInput, key: string, slot: Slot, event: Event) {
  return (
    event instanceof CustomEvent &&
    !input.isStale() &&
    !input.isDisposed() &&
    !slot.unavailable &&
    input.readLayout().entries.some((entry) => entry.key === key)
  );
}

export function staleObservation(
  input: HostsInput,
  slot: Slot,
  note: NonNullable<ReturnType<typeof terminalObservation>>
) {
  const locator = input.locator(slot);
  if (note.generation < (slot.observation?.generation || 0)) return true;
  return locator.kind === 'existing' && note.id !== locator.id;
}

export function noteUnread(input: HostsInput, slot: Slot) {
  if (visibleSlot(input, slot) || slot.unread) return;
  slot.unread = true;
  input.requestUpdate();
}

export function applyTerminalObservation(input: HostsInput, key: string, slot: Slot, detail: unknown) {
  const note = terminalObservation(detail);
  if (!note || staleObservation(input, slot, note)) return;
  if (note.kind === 'output') {
    noteUnread(input, slot);
    return;
  }
  slot.observation = note;
  input.requestUpdate();
  if (note.state === 'ready' && slot.readRequested) input.markViewed(key);
}

export function onTerminalObservation(input: HostsInput, key: string, slot: Slot, event: Event) {
  if (!observationAdmitted(input, key, slot, event)) return;
  try {
    applyTerminalObservation(input, key, slot, (event as CustomEvent).detail);
  } catch {
    /* Invalid/late observations cannot change a retained owner. */
  }
}

export function onTerminalMetadata(
  input: HostsInput,
  key: string,
  slot: Slot,
  terminal: Slot['terminal'],
  event: Event
) {
  if (
    !(event instanceof CustomEvent) ||
    input.isDisposed() ||
    input.isStale() ||
    !input.readLayout().entries.some((e) => e.key === key)
  )
    return;
  const locator = input.locator(slot);
  if (locator.kind !== 'existing') return;
  try {
    const metadata = terminalMetadata(event.detail, slot.binding);
    if (metadata.id !== locator.id) return;
    slot.metadata = metadata;
    slot.observedAt = Date.now();
    terminal.setName(metadata.name);
    input.requestUpdate();
  } catch {
    /* Invalid observation cannot change the binding. */
  }
}

export function geometrySize(detail: unknown) {
  const value = object(detail);
  if (typeof value.width !== 'number' || typeof value.height !== 'number') return;
  if (!Number.isFinite(value.width) || !Number.isFinite(value.height)) return;
  if (value.width <= 0 || value.height <= 0 || value.width > 10000 || value.height > 10000) return;
  return {width: value.width as number, height: value.height as number};
}

export function onTerminalGeometry(input: HostsInput, slot: Slot, event: Event) {
  if (!(event instanceof CustomEvent) || input.isStale() || input.isDisposed()) return;
  const size = geometrySize(event.detail);
  if (!size) return;
  slot.minimum = size;
  input.measure();
  input.requestUpdate();
}

export function onTerminalFocusIn(input: HostsInput, key: string) {
  const layout = input.readLayout();
  const pane = paneFor(layout.tree, key);
  if (pane && layout.focused !== pane.key) input.focusPane(pane.key);
}

export async function addSlot(
  input: HostsInput,
  space: Space,
  entry: LayoutEntry,
  metadata?: TerminalMetadata
): Promise<Slot | undefined> {
  if (input.isStale() || input.isDisposed()) return;
  let existing = input.readSlots().find((s) => s.key === entry.key);
  if (existing) return existing;
  const n = input.readEpoch();
  await input.readUpdateComplete();
  if (!input.live(n) || input.concealed()) return;
  existing = input.readSlots().find((s) => s.key === entry.key);
  if (existing) return existing;
  const layer = input.findMountLayer();
  check(layer);
  const {key, locator} = entry,
    binding = identity(input, space),
    host = document.createElement('div');
  host.className = 'soda-workspace-terminal';
  host.id = 'soda-owner-' + key;
  host.setAttribute('role', 'tabpanel');
  layer.append(host);
  const terminal = input.readFactory()(host, binding, locator);
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
  input.pushSlot(slot);
  if (metadata) terminal.setName(metadata.name);
  host.addEventListener('soda-terminal-locator', (event) =>
    onTerminalLocator(input, key, binding, terminal, slot, event)
  );
  host.addEventListener('soda-terminal-observation', (event) => onTerminalObservation(input, key, slot, event));
  host.addEventListener('pointerdown', (event) => {
    if (event.isTrusted) input.markViewed(key);
  });
  host.addEventListener('soda-terminal-metadata', (event) => onTerminalMetadata(input, key, slot, terminal, event));
  host.addEventListener('soda-terminal-authority-lost', () => {
    if (!input.isDisposed()) input.invalidate();
  });
  host.addEventListener('soda-terminal-command', (event) => input.onTerminalCommand(key, slot, binding, event));
  host.addEventListener('soda-terminal-geometry', (event) => onTerminalGeometry(input, slot, event));
  host.addEventListener('focusin', () => onTerminalFocusIn(input, key));
  input.display();
  input.requestUpdate();
  return slot;
}

export function slotHidden(input: HostsInput, slot: Slot, area: Area | undefined) {
  return input.isStale() || input.readSetupScreen() || slot.unavailable || input.readView() !== 'terminal' || !area;
}

export function applySlotGeometry(input: HostsInput, slot: Slot, area: Area) {
  const tabHeight = input.readTabHeight();
  slot.host.style.cssText =
    input.rectangle({...area, y: area.y + tabHeight, height: Math.max(0, area.height - tabHeight)}) +
    `;--soda-terminal-height:${Math.max(0, area.height - tabHeight)}px`;
  slot.host.setAttribute('aria-labelledby', 'soda-tab-' + slot.key);
}

export function displaySlot(input: HostsInput, slot: Slot, area: Area | undefined) {
  slot.host.hidden = slotHidden(input, slot, area);
  if (area) applySlotGeometry(input, slot, area);
  if (slot.host.hidden || !input.isActiveSurface()) slot.readRequested = false;
  slot.terminal.setVisible(input.readSurfaceVisible() && !slot.host.hidden && !input.concealed());
}

export function displaySlots(input: HostsInput) {
  const panes = input.readProjectionPanes();
  for (const slot of input.readSlots())
    displaySlot(
      input,
      slot,
      panes.find((area) => area.pane.selected === slot.key)
    );
}

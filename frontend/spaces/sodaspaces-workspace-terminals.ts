import type {Space} from './sodaspaces-api.js';
import type {LayoutEntry, PaneArea, WorkspaceLayout} from './sodaspaces-layout.js';
import {forgetEntry, moveTab, selectTab} from './sodaspaces-layout.js';
import type {TerminalLocator} from './sodaspaces-terminal.js';
import type {Slot} from './sodaspaces-workspace-types.js';

export interface TerminalsInput {
  isStale: () => boolean;
  isDisposed: () => boolean;
  isAvailable: () => boolean;
  isActiveSurface: () => boolean;
  readEpoch: () => number;
  readSelected: () => string;
  readSpaces: () => Space[];
  readLayout: () => WorkspaceLayout;
  readSlots: () => Slot[];
  readMaximized: () => string | undefined;
  readProjectionPanes: () => PaneArea[];
  readUpdateComplete: () => Promise<boolean>;
  live: (n: number) => boolean;
  closeMenus: () => void;
  persist: () => boolean;
  display: () => void;
  measure: () => void;
  markViewed: (key?: string) => void;
  locator: (slot: Slot) => TerminalLocator;
  addSlot: (space: Space, entry: LayoutEntry) => Promise<Slot | undefined>;
  setSpaces: (spaces: Space[]) => void;
  setLayout: (layout: WorkspaceLayout) => void;
  setSlots: (slots: Slot[]) => void;
  setMaximized: (key: string | undefined) => void;
  setStatus: (status: string) => void;
  setView: (view: 'terminal' | 'sessions' | 'project') => void;
  setChoosingPane: (pane: string | undefined) => void;
}

export function canRestorePane(
  input: TerminalsInput,
  entry: LayoutEntry | undefined,
  space: Space | undefined,
  n: number,
  signal: AbortSignal
) {
  return (
    !!entry &&
    !!space &&
    !space.authority_unavailable &&
    space.execution_allowed &&
    !!space.login &&
    input.live(n) &&
    !signal.aborted
  );
}

export async function restoreSelectedPane(input: TerminalsInput, n: number, signal: AbortSignal, area: PaneArea) {
  const layout = input.readLayout();
  const entry = layout.entries.find((e) => e.key === area.pane.selected);
  const space = input.readSpaces().find((s) => s.environment.id === entry?.environmentId);
  if (!canRestorePane(input, entry, space, n, signal)) return;
  const slot = await input.addSlot(space!, entry!);
  if (slot && input.live(n)) {
    input.display();
    await slot.terminal.restore();
  }
}

export async function restoreLocators(input: TerminalsInput, n: number, signal: AbortSignal) {
  await input.readUpdateComplete();
  input.measure();
  await input.readUpdateComplete();
  for (const area of input.readProjectionPanes()) await restoreSelectedPane(input, n, signal, area);
  input.persist();
}

export function slotName(input: TerminalsInput, slot: Slot) {
  const locator = input.locator(slot);
  return (
    slot.metadata?.name ||
    slot.proposedName ||
    'Terminal ' + (locator.kind === 'existing' ? locator.id.slice(0, 8) : locator.kind)
  );
}

export function confirmedEnd(input: TerminalsInput, key: string) {
  const slot = input.readSlots().find((s) => s.key === key),
    entry = input.readLayout().entries.find((e) => e.key === key);
  if (entry)
    input.setSpaces(
      input.readSpaces().map((space) =>
        space.environment.id !== entry.environmentId
          ? space
          : {
              ...space,
              terminals: space.terminals.filter(
                (metadata) => !(entry.locator.kind === 'existing' && metadata.id === entry.locator.id)
              ),
            }
      )
    );
  input.setLayout(forgetEntry(input.readLayout(), key));
  input.setSlots(input.readSlots().filter((s) => s.key !== key));
  input.setMaximized(undefined);
  input.persist();
  input.display();
  input.setStatus('Native cleanup confirmed for that exact terminal.');
  queueMicrotask(() => {
    slot?.terminal.dispose();
    slot?.host.remove();
  });
}

export function openSavedBlocked(input: TerminalsInput) {
  return input.isStale() || input.isDisposed() || !input.isAvailable() || !input.isActiveSurface();
}

export function openSavedSpace(input: TerminalsInput, entry: LayoutEntry) {
  const space = input.readSpaces().find((s) => s.environment.id === entry.environmentId);
  if (!space || space.authority_unavailable || !space.execution_allowed || !space.login) return;
  return space;
}

export function applyOpenLayout(input: TerminalsInput, entry: LayoutEntry, destination?: string) {
  input.closeMenus();
  const layout = input.readLayout();
  input.setLayout(destination ? moveTab(layout, entry.key, destination) : selectTab(layout, entry.key));
  input.setView('terminal');
  input.setChoosingPane(undefined);
  if (input.readMaximized()) input.setMaximized(input.readLayout().focused);
  input.persist();
}

export function shouldFocusOpened(
  input: TerminalsInput,
  slot: Slot,
  focus: boolean,
  n: number,
  invoker: Element | null
) {
  return (
    focus &&
    input.live(n) &&
    input.isActiveSurface() &&
    input.readSelected() === slot.key &&
    document.activeElement === invoker
  );
}

export async function restoreOpenedSlot(
  input: TerminalsInput,
  slot: Slot,
  n: number,
  focus: boolean,
  invoker: Element | null
) {
  if (!input.live(n) || slot.unavailable) return;
  input.display();
  slot.readRequested = true;
  if (input.locator(slot).kind !== 'new') await slot.terminal.restore();
  input.markViewed(slot.key);
  if (shouldFocusOpened(input, slot, focus, n, invoker)) slot.terminal.focus();
}

export async function openSaved(input: TerminalsInput, entry: LayoutEntry, destination?: string, focus = true) {
  const invoker = document.activeElement;
  if (openSavedBlocked(input)) return;
  const space = openSavedSpace(input, entry);
  if (!space) return;
  try {
    applyOpenLayout(input, entry, destination);
    const n = input.readEpoch(),
      slot = await input.addSlot(space, entry);
    await input.readUpdateComplete();
    if (slot) await restoreOpenedSlot(input, slot, n, focus, invoker);
  } catch {
    if (!input.isStale())
      input.setStatus('The exact saved terminal could not be selected; no replacement was requested.');
  }
}

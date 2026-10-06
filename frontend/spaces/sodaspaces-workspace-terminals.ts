import {check} from './sodaspaces-api.js';
import {terminalResponse} from './sodaspaces-terminal-response.js';
import type {Space} from './sodaspaces-inventory-response.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import type {LayoutEntry, PaneArea, WorkspaceLayout} from './sodaspaces-layout.js';
import {forgetEntry, layoutLimit, moveTab, panes, putEntry, sameLocator, selectTab} from './sodaspaces-layout.js';
import type {TerminalLocator} from './sodaspaces-terminal.js';
import type {Creation, Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';
import {renderCreation} from './sodaspaces-terminal-dialog-view.js';

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

export interface CreationFormInput {
  readCreating: () => boolean;
  isStale: () => boolean;
  isAvailable: () => boolean;
  readSpaces: () => Space[];
  readCreation: () => Creation | null;
  setCreation: (creation: Creation | null) => void;
  projectName: (space: Space) => string;
  isValidName: (name: string) => boolean;
  restoreFocus: () => void;
  createTerminal: () => void;
}

export function creationEligible(space: Space | undefined) {
  return (
    !!space?.login &&
    space.environment.provisioned &&
    !space.authority_unavailable &&
    space.execution_allowed &&
    space.observed?.running === true
  );
}

export function creationExplanation(space: Space | undefined) {
  if (creationEligible(space)) return '';
  if (!space?.login) return 'Join required.';
  if (space.authority_unavailable) return 'Status unavailable.';
  if (!space.execution_allowed) return 'Repository write access required.';
  return 'Environment stopped.';
}

export function creationContext(input: CreationFormInput, space: Space | undefined) {
  return `${space?.login || 'Join required'} @ ${space ? input.projectName(space) : 'Select a project'}`;
}

export function creationDisabled(input: CreationFormInput, draft: Creation, eligible: boolean) {
  return input.readCreating() || input.isStale() || !input.isAvailable() || !eligible || !input.isValidName(draft.name);
}

export function onCreationEnvironment(input: CreationFormInput, environmentId: string) {
  const creation = input.readCreation();
  if (creation) input.setCreation({...creation, environmentId});
}

export function onCreationName(input: CreationFormInput, name: string) {
  const creation = input.readCreation();
  if (creation) input.setCreation({...creation, name});
}

export function cancelCreation(input: CreationFormInput) {
  input.setCreation(null);
  input.restoreFocus();
}

export function creationForm(input: CreationFormInput, draft: Creation) {
  const space = input.readSpaces().find((s) => s.environment.id === draft.environmentId);
  const eligible = creationEligible(space);
  return renderCreation(
    {
      environmentId: draft.environmentId,
      name: draft.name,
      projects: input.readSpaces().map((item) => ({id: item.environment.id, name: input.projectName(item)})),
      context: creationContext(input, space),
      explanation: creationExplanation(space),
      busy: input.readCreating(),
      disabled: creationDisabled(input, draft, !!eligible),
    },
    (environmentId) => onCreationEnvironment(input, environmentId),
    (name) => onCreationName(input, name),
    () => cancelCreation(input),
    () => {
      input.createTerminal();
    }
  );
}

export interface NewTerminalInput {
  binding: WorkspaceContext | undefined;
  isStale: () => boolean;
  readCreating: () => boolean;
  isAvailable: () => boolean;
  isActiveSurface: () => boolean;
  readSurfaceVisible: () => boolean;
  readSelectedSpace: () => Space | undefined;
  readSpaces: () => Space[];
  readSlots: () => Slot[];
  readSelected: () => string;
  readLayout: () => WorkspaceLayout;
  readCreation: () => Creation | null;
  setCreation: (creation: Creation | null) => void;
  readUpdateComplete: () => Promise<boolean>;
  rememberFocus: () => void;
  focusCreationSelect: () => void;
  createTerminal: () => void;
}

export function defaultTerminalName(input: NewTerminalInput, space: Space | undefined) {
  return (
    'Terminal ' +
    (Math.max(
      space?.terminals.length || 0,
      input.readLayout().entries.filter((e) => e.environmentId === space?.environment.id).length
    ) +
      1)
  );
}

export function creationSpace(input: NewTerminalInput, selected: {binding: {environmentId: string}} | undefined) {
  const preferred = input.binding?.kind === 'page' ? input.readSelectedSpace() : undefined;
  if (preferred) return preferred;
  const native = input.binding?.kind === 'native' ? input.binding.repositoryId : undefined;
  return (
    input.readSpaces().find((s) => s.environment.id === selected?.binding.environmentId) ||
    input.readSpaces().find((s) => s.environment.repository_id === native) ||
    input.readSpaces()[0]
  );
}

export function pickCreationSpace(input: NewTerminalInput) {
  const selected = input.readSlots().find((s) => s.key === input.readSelected());
  const native = input.binding?.kind === 'native' ? input.binding.repositoryId : undefined;
  if (input.binding?.kind === 'page' && input.readSelectedSpace()) return input.readSelectedSpace();
  return (
    input.readSpaces().find((s) => s.environment.id === selected?.binding.environmentId) ||
    input.readSpaces().find((s) => s.environment.repository_id === native) ||
    input.readSpaces()[0]
  );
}

export function pageBlocksNewTerminal(input: NewTerminalInput, space: Space | undefined) {
  if (input.binding?.kind !== 'page' || !space) return false;
  return (
    !space.login ||
    !space.execution_allowed ||
    space.authority_unavailable ||
    !space.environment.provisioned ||
    space.observed?.running !== true
  );
}

export function newTerminalBlocked(input: NewTerminalInput) {
  return input.isStale() || input.readCreating() || !input.isAvailable() || !input.isActiveSurface();
}

export function focusCreationDialog(input: NewTerminalInput, pane: string) {
  void input.readUpdateComplete().then(() => {
    if (input.readCreation()?.pane === pane && !input.isStale() && input.readSurfaceVisible())
      input.focusCreationSelect();
  });
}

export function newTerminal(input: NewTerminalInput, pane: string) {
  if (newTerminalBlocked(input)) return;
  input.rememberFocus();
  const space = pickCreationSpace(input);
  if (pageBlocksNewTerminal(input, space)) return;
  input.setCreation({pane, environmentId: space?.environment.id || '', name: defaultTerminalName(input, space)});
  if (input.binding?.kind === 'page' && input.readSelectedSpace()) {
    input.createTerminal();
    return;
  }
  focusCreationDialog(input, pane);
}

export interface CreateInput {
  isStale: () => boolean;
  isDisposed: () => boolean;
  isAvailable: () => boolean;
  isActiveSurface: () => boolean;
  readCreating: () => boolean;
  setCreating: (creating: boolean) => void;
  readCreation: () => Creation | null;
  setCreation: (creation: Creation | null) => void;
  readSpaces: () => Space[];
  readLayout: () => WorkspaceLayout;
  setLayout: (layout: WorkspaceLayout) => void;
  setView: (view: 'terminal' | 'sessions' | 'project') => void;
  setStatus: (status: string) => void;
  isValidName: (name: string) => boolean;
  addSlot: (space: Space, entry: LayoutEntry) => Promise<Slot | undefined>;
  openSaved: (entry: LayoutEntry, destination?: string) => Promise<void>;
  confirmedEnd: (key: string) => void;
  requestUpdate: () => void;
}

export function createAdmitted(input: CreateInput, draft: Creation | null): draft is Creation {
  return (
    !!draft &&
    !input.readCreating() &&
    input.isAvailable() &&
    !input.isStale() &&
    !input.isDisposed() &&
    input.isActiveSurface() &&
    input.isValidName(draft.name)
  );
}

export function createSpaceReady(space: Space | undefined) {
  return (
    !!space?.login &&
    space.environment.provisioned &&
    !space.authority_unavailable &&
    space.execution_allowed &&
    space.observed?.running === true
  );
}

export async function openCreatedSlot(input: CreateInput, space: Space, draft: Creation) {
  const entry: LayoutEntry = {key: crypto.randomUUID(), environmentId: space.environment.id, locator: {kind: 'new'}};
  input.setLayout(selectTab(putEntry(input.readLayout(), entry), entry.key, draft.pane));
  input.setView('terminal');
  input.setCreation(null);
  const slot = await input.addSlot(space, entry);
  if (!slot) return;
  slot.proposedName = draft.name;
  input.requestUpdate();
  await slot.terminal.open(draft.name);
}

export async function createTerminal(input: CreateInput) {
  const draft = input.readCreation();
  if (!createAdmitted(input, draft)) return;
  const space = input.readSpaces().find((p) => p.environment.id === draft.environmentId);
  if (!createSpaceReady(space)) return;
  if (!panes(input.readLayout().tree).some((p) => p.key === draft.pane)) {
    input.setStatus('The destination pane changed. Choose New terminal again.');
    return;
  }
  if (input.readLayout().entries.length >= layoutLimit) {
    input.setStatus('The working set is full; uncertain locators cannot be evicted.');
    return;
  }
  input.setCreating(true);
  try {
    await openCreatedSlot(input, space!, draft);
  } finally {
    input.setCreating(false);
  }
}

export function openExistingBlocked(input: CreateInput, space: Space) {
  return (
    input.isStale() ||
    input.isDisposed() ||
    !input.isActiveSurface() ||
    !input.isAvailable() ||
    space.authority_unavailable ||
    !space.execution_allowed
  );
}

export function existingOrNewEntry(input: CreateInput, space: Space, metadata: TerminalMetadata) {
  let entry = input.readLayout().entries.find((e) => sameLocator(e.locator, {kind: 'existing', id: metadata.id}));
  if (entry) {
    check(entry.environmentId === space.environment.id);
    return entry;
  }
  entry = {
    key: crypto.randomUUID(),
    environmentId: space.environment.id,
    locator: {kind: 'existing', id: metadata.id},
  };
  input.setLayout(putEntry(input.readLayout(), entry));
  return entry;
}

export async function openExisting(input: CreateInput, space: Space, metadata: TerminalMetadata, destination?: string) {
  if (openExistingBlocked(input, space)) return;
  try {
    const entry = existingOrNewEntry(input, space, metadata);
    if (metadata.state === 'ended') {
      input.confirmedEnd(entry.key);
      return;
    }
    await input.openSaved(entry, destination);
  } catch {
    if (!input.isStale())
      input.setStatus('The exact session could not be opened. No locator was replaced or creation requested.');
  }
}

export interface RenameInput {
  isStale: () => boolean;
  isDisposed: () => boolean;
  isActiveSurface: () => boolean;
  readEpoch: () => number;
  readEditing: () => {key: string; name: string} | null;
  setEditing: (editing: {key: string; name: string} | null) => void;
  renaming: Set<string>;
  isValidName: (name: string) => boolean;
  locator: (slot: Slot) => TerminalLocator;
  api: (path: string, body: Record<string, unknown>, signal: AbortSignal) => Promise<unknown>;
  lifetimeSignal: AbortSignal;
  live: (generation: number) => boolean;
  setStatus: (message: string) => void;
  requestUpdate: () => void;
}

function renameAdmitted(input: RenameInput, slot: Slot, name: string) {
  return (
    !input.renaming.has(slot.key) &&
    !input.isStale() &&
    !input.isDisposed() &&
    input.isActiveSurface() &&
    input.isValidName(name) &&
    input.readEditing()?.key === slot.key
  );
}

function applyRenamedMetadata(input: RenameInput, slot: Slot, value: TerminalMetadata, id: string) {
  check(value?.id === id);
  slot.metadata = value;
  slot.terminal.setName(value.name);
  if (input.readEditing()?.key === slot.key) input.setEditing(null);
}

export async function rename(input: RenameInput, slot: Slot, name: string) {
  if (!renameAdmitted(input, slot, name)) return;
  const locator = input.locator(slot);
  if (locator.kind !== 'existing') return;
  const id = locator.id,
    n = input.readEpoch();
  input.renaming.add(slot.key);
  input.requestUpdate();
  const request = new AbortController(),
    timeout = window.setTimeout(() => request.abort(), 15000);
  try {
    const value = terminalResponse(
      await input.api(
        `/api/environments/${slot.binding.environmentId}/terminal-sessions/${id}`,
        {action: 'rename', name},
        AbortSignal.any([request.signal, input.lifetimeSignal])
      ),
      slot.binding
    );
    if (input.live(n) && value) applyRenamedMetadata(input, slot, value, id);
  } catch {
    if (input.live(n)) input.setStatus('Rename was not confirmed. No retry was made.');
  } finally {
    window.clearTimeout(timeout);
    input.renaming.delete(slot.key);
    input.requestUpdate();
  }
}

import {html} from 'lit';
import {check, readSodaJSON, spacesResponse} from './sodaspaces-api.js';
import type {Space} from './sodaspaces-api.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import type {TerminalLocator} from './sodaspaces-terminal.js';
import type {Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';

type SpacesCollection = ReturnType<typeof spacesResponse>;

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

export interface LiveReading {
  disposed: boolean;
  stale: boolean;
  epoch: number;
}

export function live(reading: LiveReading, n: number) {
  return !reading.disposed && !reading.stale && reading.epoch === n;
}

export interface ReadSpacesInput {
  invalidate: () => void;
  setReconnectRequired: (required: boolean) => void;
}

export async function readSpacesResponse(input: ReadSpacesInput, response: Response) {
  if (response.ok) return readSodaJSON(response);
  if (response.status === 401 || response.status === 403) {
    input.invalidate();
    input.setReconnectRequired(response.status === 401);
  }
  throw Error('Spaces request refused');
}

export interface ApiInput {
  binding: WorkspaceContext | undefined;
  disposed: boolean;
  stale: boolean;
  lifetimeSignal: AbortSignal;
  invalidate: () => void;
  setReconnectRequired: (required: boolean) => void;
}

export async function api(
  input: ApiInput,
  path: string,
  body?: Record<string, unknown>,
  signal?: AbortSignal
): Promise<unknown> {
  check(input.binding && !input.disposed && !input.stale && !signal?.aborted);
  const headers = body ? {'Content-Type': 'application/json'} : {};
  const response = await input.binding.transport.request(
    path.slice('/api/'.length),
    sodaWorkspaceInit(headers, body, signal || input.lifetimeSignal)
  );
  return readSpacesResponse(input, response);
}

export interface InventoryInput {
  binding: WorkspaceContext | undefined;
  lifetimeSignal: AbortSignal;
  isDisposed: () => boolean;
  isStale: () => boolean;
  readEpoch: () => number;
  readBusy: () => boolean;
  readRequest: () => AbortController | undefined;
  readSpaces: () => Space[];
  readComplete: () => boolean;
  readNextAfter: () => string;
  readSetup: () => 'repositories' | 'configure' | null;
  readSelectedSpace: () => Space | undefined;
  readProject: () => string;
  readObservedAt: () => number;
  readSlots: () => Slot[];
  readRestored: () => boolean;
  readSurfaceVisible: () => boolean;
  readActor: () => {id: string; login: string} | undefined;
  readView: () => 'terminal' | 'sessions' | 'project';
  readManagementMode: () => 'standard' | 'journey' | 'settings';
  readStorageLoaded: () => boolean;
  readUpdateComplete: () => Promise<boolean>;
  concealed: () => boolean;
  setBusy: (busy: boolean) => void;
  setRequest: (request: AbortController) => void;
  setSpaces: (spaces: Space[]) => void;
  setComplete: (complete: boolean) => void;
  setNextAfter: (nextAfter: string) => void;
  setAvailable: (available: boolean) => void;
  setProject: (project: string) => void;
  stampNow: () => void;
  setStatus: (status: string) => void;
  setRestored: (restored: boolean) => void;
  setActor: (actor: {id: string; login: string}) => void;
  setStorageKey: (storageKey: string) => void;
  setSetup: (setup: 'repositories' | 'configure' | null) => void;
  clearSetupReturn: () => void;
  setView: (view: 'terminal' | 'sessions' | 'project') => void;
  setReconnectRequired: (required: boolean) => void;
  invalidate: () => void;
  loadLayout: () => void;
  restoreLocators: (n: number, signal: AbortSignal) => Promise<void>;
  display: () => void;
  showManagement: (repositoryId: string, mode?: 'standard' | 'journey' | 'settings') => void;
  projectRunnable: (space: Space) => boolean;
  locator: (slot: Slot) => TerminalLocator;
  confirmedEnd: (key: string) => void;
  refreshJourneyProject: () => void;
}

function reading(input: InventoryInput): LiveReading {
  return {disposed: input.isDisposed(), stale: input.isStale(), epoch: input.readEpoch()};
}

function apiInput(input: InventoryInput): ApiInput {
  return {
    binding: input.binding,
    disposed: input.isDisposed(),
    stale: input.isStale(),
    lifetimeSignal: input.lifetimeSignal,
    invalidate: input.invalidate,
    setReconnectRequired: input.setReconnectRequired,
  };
}

export function refreshBlocked(input: InventoryInput) {
  return input.readBusy() || input.isStale() || input.isDisposed() || !input.binding;
}

export function beginRefreshRead(input: InventoryInput) {
  input.setBusy(true);
  input.readRequest()?.abort();
  const request = new AbortController();
  input.setRequest(request);
  return request;
}

export function collectionStatus(complete: boolean, factoryIncomplete: boolean) {
  if (complete) return '';
  return factoryIncomplete
    ? 'We couldn’t load all projects, terminals and factory history. Some may be missing from this list.'
    : 'We couldn’t load all projects and terminals. Some may be missing from this list.';
}

export function applySpacesCollection(input: InventoryInput, collection: SpacesCollection) {
  input.setSpaces(collection.items);
  input.setComplete(collection.complete);
  input.setNextAfter(collection.nextAfter);
  input.setAvailable(true);
  if (!input.readSetup() && !input.readSelectedSpace() && collection.complete)
    input.setProject(input.readSpaces()[0]?.environment.repository_id || '');
  input.stampNow();
  input.setStatus(collectionStatus(collection.complete, collection.factoryIncomplete));
}

export function appendSpacesCollection(input: InventoryInput, collection: SpacesCollection) {
  const seen = new Set(input.readSpaces().map((s) => s.environment.id));
  const grown = [...input.readSpaces()];
  for (const item of collection.items) {
    if (seen.has(item.environment.id)) continue;
    seen.add(item.environment.id);
    grown.push(item);
  }
  input.setSpaces(grown);
  input.setNextAfter(collection.nextAfter);
  // The last page decides: earlier pages always report incomplete
  // while more follow, and degraded rows keep their own inline state.
  input.setComplete(collection.nextAfter === '' && collection.complete);
  input.stampNow();
  input.setStatus(collectionStatus(input.readComplete(), collection.factoryIncomplete));
}

export function slotLost(slot: Slot, space: Space | undefined, complete: boolean) {
  if (space) return space.authority_unavailable || !space.execution_allowed || space.login !== slot.binding.login;
  return complete;
}

export function invalidateSlot(slot: Slot) {
  slot.unavailable = true;
  slot.metadata = undefined;
  delete slot.proposedName;
  slot.observation = undefined;
  slot.unread = slot.readRequested = false;
  slot.terminal.invalidate();
}

export function applySlotMetadata(input: InventoryInput, slot: Slot, metadata: TerminalMetadata) {
  if (metadata.state === 'ended') {
    input.confirmedEnd(slot.key);
    return;
  }
  slot.metadata = metadata;
  slot.observedAt = input.readObservedAt();
  slot.terminal.setName(metadata.name);
}

export function existingSlotMetadata(input: InventoryInput, slot: Slot, space: Space | undefined) {
  const locator = input.locator(slot);
  return space?.terminals.find((t) => locator.kind === 'existing' && t.id === locator.id);
}

export function refreshSlots(input: InventoryInput, complete: boolean) {
  for (const slot of input.readSlots()) {
    const space = input.readSpaces().find((s) => s.environment.id === slot.binding.environmentId);
    if (slotLost(slot, space, complete)) {
      invalidateSlot(slot);
      continue;
    }
    const metadata = existingSlotMetadata(input, slot, space);
    if (metadata) applySlotMetadata(input, slot, metadata);
  }
}

export function restoreIfNeeded(input: InventoryInput, n: number, signal: AbortSignal) {
  if (input.readRestored() || !input.readSurfaceVisible() || input.concealed()) return;
  input.setRestored(true);
  return input.restoreLocators(n, signal);
}

export function clearConfigureAfterCreate(input: InventoryInput) {
  if (input.readSetup() === 'configure' && input.readSelectedSpace()?.environment.provisioned) {
    input.setSetup(null);
    input.clearSetupReturn();
  }
}

export function pageJourneyActive(input: InventoryInput) {
  return input.binding?.kind === 'page' && !input.readSetup() && !!input.readSelectedSpace();
}

export function leaveJourneyManagement(input: InventoryInput) {
  if (input.readView() === 'project' && input.readManagementMode() === 'journey') input.setView('terminal');
}

export function refreshJourneyManagement(input: InventoryInput) {
  if (input.readView() === 'project' && input.readManagementMode() === 'journey') input.refreshJourneyProject();
}

export function syncPageJourney(input: InventoryInput) {
  if (!pageJourneyActive(input)) return;
  const space = input.readSelectedSpace()!;
  if (input.projectRunnable(space)) {
    leaveJourneyManagement(input);
    return;
  }
  if (input.readView() === 'terminal') void input.showManagement(input.readProject(), 'journey');
  else refreshJourneyManagement(input);
}

export function refreshFailed(input: InventoryInput, n: number) {
  if (!live(reading(input), n)) return;
  input.setAvailable(false);
  input.setComplete(false);
  input.setStatus('We couldn’t refresh your projects and terminals. The information shown may be out of date.');
}

export async function finishRefreshSuccess(input: InventoryInput, n: number, signal: AbortSignal) {
  if (!input.readStorageLoaded()) input.loadLayout();
  await restoreIfNeeded(input, n, signal);
  clearConfigureAfterCreate(input);
  syncPageJourney(input);
  input.display();
}

export async function refreshLive(input: InventoryInput, n: number, request: AbortController) {
  const collection = spacesResponse(await api(apiInput(input), '/api/spaces', undefined, request.signal));
  if (!live(reading(input), n)) return;
  const actor = input.readActor();
  if (actor && actor.id !== collection.actor.id) {
    input.invalidate();
    return;
  }
  input.setActor(collection.actor);
  input.setStorageKey('soda-spaces:v3:' + collection.actor.id);
  applySpacesCollection(input, collection);
  refreshSlots(input, collection.complete);
  await input.readUpdateComplete();
  if (!live(reading(input), n)) return;
  await finishRefreshSuccess(input, n, request.signal);
}

export async function refresh(input: InventoryInput) {
  if (refreshBlocked(input)) return;
  const n = input.readEpoch();
  const request = beginRefreshRead(input);
  const timer = window.setTimeout(() => request.abort(), 15000);
  try {
    await refreshLive(input, n, request);
  } catch {
    refreshFailed(input, n);
  } finally {
    window.clearTimeout(timer);
    if (live(reading(input), n)) input.setBusy(false);
  }
}

export async function loadMore(input: InventoryInput) {
  if (refreshBlocked(input) || !input.readNextAfter()) return;
  const n = input.readEpoch();
  const request = beginRefreshRead(input);
  const timer = window.setTimeout(() => request.abort(), 15000);
  try {
    const collection = spacesResponse(
      await api(apiInput(input), `/api/spaces?after=${input.readNextAfter()}`, undefined, request.signal)
    );
    if (!live(reading(input), n)) return;
    const actor = input.readActor();
    if (actor && actor.id !== collection.actor.id) {
      input.invalidate();
      return;
    }
    appendSpacesCollection(input, collection);
    refreshSlots(input, input.readComplete());
    await input.readUpdateComplete();
    if (!live(reading(input), n)) return;
    await finishRefreshSuccess(input, n, request.signal);
  } catch {
    refreshFailed(input, n);
  } finally {
    window.clearTimeout(timer);
    if (live(reading(input), n)) input.setBusy(false);
  }
}

export interface MoreProjectsInput {
  nextAfter: string;
  stale: boolean;
  busy: boolean;
  more: () => void;
}

export function renderMoreProjects(input: MoreProjectsInput) {
  if (!input.nextAfter || input.stale) return '';
  const label = input.busy ? 'Loading…' : 'Show more projects';
  return html`<button class="ui button soda-more-projects" ?disabled=${input.busy} @click=${() => input.more()}>
    ${label}
  </button>`;
}

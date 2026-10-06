import {check, object} from './sodaspaces-api.js';
import type {Space} from './sodaspaces-inventory-response.js';
import type {LayoutEntry} from './sodaspaces-layout.js';
import type {WorkspaceContext} from './sodaspaces-workspace-types.js';

export interface DrawerInput {
  binding: WorkspaceContext | undefined;
  isAvailable: () => boolean;
  isActiveSurface: () => boolean;
  readOpeningDrawer: () => boolean;
  setOpeningDrawer: (opening: boolean) => void;
  readEntries: () => LayoutEntry[];
  readSelected: () => string;
  readSpaces: () => Space[];
  readEpoch: () => number;
  live: (n: number) => boolean;
  closeMenus: () => void;
  setStatus: (status: string) => void;
  api: (path: string, signal: AbortSignal) => Promise<unknown>;
  persist: () => boolean;
  repositoryPrefix: () => string;
}

export function drawerRepositoryPart(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    value.length > 0 &&
    value.length <= 255 &&
    value !== '.' &&
    value !== '..' &&
    !/[\/\\\p{Cc}]/u.test(value)
  );
}

export function drawerOpenBlocked(input: DrawerInput) {
  return (
    input.binding?.kind !== 'page' || input.readOpeningDrawer() || !input.isAvailable() || !input.isActiveSurface()
  );
}

export function drawerEntry(input: DrawerInput) {
  const entry = input.readEntries().find((entry) => entry.key === input.readSelected());
  const space = input.readSpaces().find((space) => space.environment.id === entry?.environmentId);
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

export function drawerStillCurrent(input: DrawerInput, n: number, request: AbortController, entry: LayoutEntry) {
  return input.live(n) && !request.signal.aborted && input.readSelected() === entry.key;
}

export function assignDrawer(input: DrawerInput, owner: string, name: string) {
  window.location.assign(input.repositoryPrefix() + '/' + encodeURIComponent(owner) + '/' + encodeURIComponent(name));
}

export async function openDrawerNavigation(
  input: DrawerInput,
  entry: LayoutEntry,
  space: Space,
  n: number,
  request: AbortController
) {
  const response = object(
    await input.api('/api/environments?repository_id=' + space.environment.repository_id, request.signal)
  );
  if (!drawerStillCurrent(input, n, request, entry)) return;
  const repository = object(response.repository);
  check(
    repository.id === space.environment.repository_id &&
      drawerRepositoryPart(repository.owner) &&
      drawerRepositoryPart(repository.name)
  );
  if (!input.persist()) {
    input.setStatus('Save the workspace before opening it in the drawer; restoration is unavailable.');
    return;
  }
  assignDrawer(input, repository.owner, repository.name);
}

export async function openInDrawer(input: DrawerInput) {
  if (drawerOpenBlocked(input)) return;
  const selected = drawerEntry(input);
  if (!selected) return;
  const n = input.readEpoch(),
    request = new AbortController();
  const timer = window.setTimeout(() => request.abort(), 15000);
  input.setOpeningDrawer(true);
  input.closeMenus();
  try {
    await openDrawerNavigation(input, selected.entry, selected.space, n, request);
  } catch {
    if (input.live(n))
      input.setStatus('Could not open the repository drawer. Your terminal remains here; refresh and try again.');
  } finally {
    window.clearTimeout(timer);
    input.setOpeningDrawer(false);
  }
}

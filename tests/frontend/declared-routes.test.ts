import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {navigateDrawer, openDrawerNavigation} from '../../frontend/spaces/sodaspaces-workspace-drawer';
import type {DrawerInput} from '../../frontend/spaces/sodaspaces-workspace-drawer';
import type {LayoutEntry} from '../../frontend/spaces/sodaspaces-layout';
import type {Space} from '../../frontend/spaces/sodaspaces-inventory-response';

const entry = {key: 'k1', environmentId: 'env1', locator: {kind: 'existing', id: 't1'}} as unknown as LayoutEntry;
const space = {
  environment: {id: 'env1', repository_id: '7'},
  authority_unavailable: false,
  execution_allowed: true,
  login: 'alice',
} as unknown as Space;

function drawerInput(overrides: Partial<DrawerInput>): DrawerInput {
  return {
    binding: {kind: 'page'},
    isAvailable: () => true,
    isActiveSurface: () => true,
    readOpeningDrawer: () => false,
    setOpeningDrawer: () => {},
    readEntries: () => [entry],
    readSelected: () => 'k1',
    readSpaces: () => [space],
    readEpoch: () => 1,
    live: () => true,
    closeMenus: () => {},
    setStatus: () => {},
    api: async () => ({repository: {id: '7', owner: 'alice', name: 'demo repo'}}),
    persist: () => true,
    repositoryPrefix: () => '/forge',
    navigate: () => {},
    ...overrides,
  } as unknown as DrawerInput;
}

test('drawer success navigates through the host with the exact canonical repository path', async () => {
  const navigated: string[] = [];
  const paths: string[] = [];
  await openDrawerNavigation(
    drawerInput({
      api: async (path) => {
        paths.push(path);
        return {repository: {id: '7', owner: 'alice', name: 'demo repo'}};
      },
      navigate: (path) => navigated.push(path),
    }),
    entry,
    space,
    1,
    new AbortController()
  );
  assert.deepEqual(paths, ['/api/environments?repository_id=7']);
  assert.deepEqual(navigated, ['/forge/alice/demo%20repo']);
});

test('drawer success stays put when persistence is unavailable', async () => {
  const navigated: string[] = [];
  const statuses: string[] = [];
  await openDrawerNavigation(
    drawerInput({
      persist: () => false,
      setStatus: (status) => statuses.push(status),
      navigate: (path) => navigated.push(path),
    }),
    entry,
    space,
    1,
    new AbortController()
  );
  assert.deepEqual(navigated, []);
  assert.equal(statuses.length, 1);
});

test('drawer success stays put when the selection changed during the request', async () => {
  const navigated: string[] = [];
  let selected = 'k1';
  await openDrawerNavigation(
    drawerInput({
      readSelected: () => selected,
      api: async () => {
        selected = 'other';
        return {repository: {id: '7', owner: 'alice', name: 'demo'}};
      },
      navigate: (path) => navigated.push(path),
    }),
    entry,
    space,
    1,
    new AbortController()
  );
  assert.deepEqual(navigated, []);
});

test('navigateDrawer encodes owner and name as single path segments', () => {
  const navigated: string[] = [];
  navigateDrawer(drawerInput({navigate: (path) => navigated.push(path)}), 'al/ice', 'de?mo');
  assert.deepEqual(navigated, ['/forge/al%2Fice/de%3Fmo']);
});

test('spaces page declares the bounded new and saved-request subroutes', async () => {
  const manifest = JSON.parse(
    await readFile(new URL('../../system/containers/extension/extension.json', import.meta.url), 'utf8')
  ) as {pages: {id: string; subroutes?: unknown}[]};
  const spaces = manifest.pages.find((page) => page.id === 'spaces');
  assert.deepEqual(spaces?.subroutes, [
    {pattern: 'new', title: 'New Project'},
    {pattern: 'requests/{request}', title: 'Build request'},
  ]);
});

test('pinned contract mirror exposes declared-route navigation and owner identity', async () => {
  const mirror = await readFile(
    new URL('../../frontend/spaces/fountain-browser-contract.d.ts', import.meta.url),
    'utf8'
  );
  assert.match(mirror, /navigate: \(path: string\) => void/);
  assert.match(mirror, /ownerId\?: string/);
});

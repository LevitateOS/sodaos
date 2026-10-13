import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {beginSetup, cancelSetup, resumeSetup} from '../../frontend/spaces/sodaspaces-workspace-setup';
import type {
  SetupActions,
  SetupHost,
  SetupReading,
  SetupReturn,
} from '../../frontend/spaces/sodaspaces-workspace-setup';

const intent = 'd0123456789abcdef01234567';

function readingWith(overrides: Partial<SetupReading>): SetupReading {
  return {
    setup: null,
    welcomeScreen: false,
    busy: false,
    reconnectRequired: false,
    stale: false,
    connectURL: '',
    available: true,
    canRestore: true,
    activeSurface: true,
    repositoryQuery: '',
    repositoryChoice: '',
    repositoryResult: undefined,
    repositoryError: '',
    repositoryBusy: false,
    forgejoPrefix: '',
    project: '1',
    view: 'terminal',
    managementMode: 'standard',
    setupReturn: undefined,
    firstSpace: undefined,
    actorId: '2',
    requestId: '',
    reviewAuthorization: undefined,
    reviewedIntent: undefined,
    setupCommitted: false,
    transport: undefined,
    suggestedRepository: undefined,
    overviewScope: undefined,
    committedSpace: undefined,
    onBegin: () => {},
    onRefresh: () => {},
    page: true,
    hasSelection: true,
    ...overrides,
  } as SetupReading;
}

function stubs() {
  const calls: {name: string; value: unknown}[] = [];
  const record = (name: string, value: unknown) => calls.push({name, value});
  const actions = {
    setSetup: (value: unknown) => record('setSetup', value),
    setRepositoryQuery: (value: unknown) => record('setRepositoryQuery', value),
    setRepositoryChoice: (value: unknown) => record('setRepositoryChoice', value),
    setRepositoryResult: (value: unknown) => record('setRepositoryResult', value),
    setRepositoryCursors: (value: unknown) => record('setRepositoryCursors', value),
    setRepositoryError: (value: unknown) => record('setRepositoryError', value),
    setRepositoryBusy: (value: unknown) => record('setRepositoryBusy', value),
    setSetupReturn: (value: unknown) => record('setSetupReturn', value),
    setReviewAuthorization: (value: unknown) => record('setReviewAuthorization', value),
    setReviewedIntent: (value: unknown) => record('setReviewedIntent', value),
    setSetupCommitted: (value: unknown) => record('setSetupCommitted', value),
    setProject: (value: unknown) => record('setProject', value),
    setView: (value: unknown) => record('setView', value),
    setManagementMode: (value: unknown) => record('setManagementMode', value),
    repositoryCursors: () => [''],
    currentRepositoryRequest: () => undefined,
    admitRepositoryRequest: () => {},
    currentRepositoryQuery: () => '',
    hasSelectedSpace: () => true,
    isStale: () => false,
    isActiveSurface: () => true,
    selectProject: () => {},
    rememberFocus: () => record('rememberFocus', true),
    restoreFocus: () => record('restoreFocus', true),
    showManagement: () => {},
    notifyFactory: (value: unknown) => record('notifyFactory', value),
    api: () => Promise.reject(new Error('unexpected api call')),
    syncIntentRoute: (value: unknown) => record('syncIntentRoute', value),
    setSavedRequests: (value: unknown) => record('setSavedRequests', value),
    setOverviewScope: (value: unknown) => record('setOverviewScope', value),
    requestFactoryWatch: () => {},
  } as unknown as SetupActions;
  const host = {
    updateComplete: Promise.resolve(true),
    querySelector: () => undefined,
  } as unknown as SetupHost;
  const last = (name: string) => calls.filter((call) => call.name === name).at(-1)?.value;
  return {actions, host, last};
}

test('setup from the compact sessions overlay returns to the retained terminal', () => {
  const {actions, host, last} = stubs();
  beginSetup(readingWith({view: 'sessions', hasSelection: true}), actions, host);
  assert.deepEqual(last('setSetupReturn'), {project: '1', view: 'terminal', mode: 'standard'});
});

test('setup from the sessions list without a selection returns to the list', () => {
  const {actions, host, last} = stubs();
  beginSetup(readingWith({view: 'sessions', hasSelection: false}), actions, host);
  assert.deepEqual(last('setSetupReturn'), {project: '1', view: 'sessions', mode: 'standard'});
});

test('setup from terminal and project views keeps its return view', () => {
  for (const view of ['terminal', 'project'] as const) {
    const {actions, host, last} = stubs();
    beginSetup(readingWith({view, hasSelection: true}), actions, host);
    assert.deepEqual(last('setSetupReturn'), {project: '1', view, mode: 'standard'});
  }
});

test('setup from a native sessions list keeps its return view', () => {
  const {actions, host, last} = stubs();
  beginSetup(readingWith({view: 'sessions', hasSelection: true, page: false}), actions, host);
  assert.deepEqual(last('setSetupReturn'), {project: '1', view: 'sessions', mode: 'standard'});
});

test('resumed setup from the compact sessions overlay returns to the retained terminal', () => {
  const {actions, host, last} = stubs();
  resumeSetup(intent, readingWith({view: 'sessions', hasSelection: true}), actions, host);
  assert.deepEqual(last('setSetupReturn'), {project: '1', view: 'terminal', mode: 'standard'});
});

test('cancelled setup restores the captured terminal view', () => {
  const {actions, host, last} = stubs();
  const setupReturn: SetupReturn = {project: '1', view: 'terminal', mode: 'standard'};
  cancelSetup(readingWith({setup: 'intent', setupReturn}), actions, host);
  assert.equal(last('setSetup'), null);
  assert.equal(last('setView'), 'terminal');
  assert.equal(last('setSetupReturn'), undefined);
});

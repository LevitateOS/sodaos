import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {
  childScopeFromReading,
  detailRunsForIssue,
  factoryWatchRequestDetail,
  intentIdentity,
  isNewRouteLocation,
  inventoryIntents,
  newRouteFor,
  overviewScopeDetail,
  requestIdFromLocation,
  requestRouteFor,
  resolveWatchTarget,
  sameScope,
} from '../../frontend/spaces/sodaspaces-workspace-setup';
import type {SetupReading} from '../../frontend/spaces/sodaspaces-workspace-setup';
import type {Space} from '../../frontend/spaces/sodaspaces-inventory-response';
import type {PreparedExtensionMount} from '../../frontend/spaces/soda-extension';
import type {WorkOverviewContext} from '../../frontend/spaces/sodaspaces-factory-work';
import type {OverviewScope} from '../../frontend/spaces/sodaspaces-factory-progress-view';

const id = 'd0123456789abcdef01234567';
const transport = {} as unknown as PreparedExtensionMount;
const capabilities = {
  watch: {available: true},
  stop: {available: true},
  retry: {available: false},
  takeover: {available: true},
  pause: {available: true},
  resume: {available: true},
};
const scope: OverviewScope = {repository: '7', intentID: id, issues: [{issue: '5', capabilities}]};
const context: WorkOverviewContext = {
  transport,
  repository: '7',
  repositoryPath: '/forgejo.test/alice/Alpha',
  intentID: id,
};

function setupReadingWith(overrides: Partial<SetupReading>): SetupReading {
  return {
    setup: 'intent',
    stale: false,
    canRestore: true,
    activeSurface: true,
    setupCommitted: true,
    reviewedIntent: id,
    repositoryChoice: '7',
    transport,
    overviewScope: scope,
    ...overrides,
  } as SetupReading;
}

function scopeEvent(detail: unknown): Event {
  return new CustomEvent('soda-factory-overview-scope', {detail});
}

test('request id reads only from app-root request routes', () => {
  assert.equal(requestIdFromLocation('/workspace/requests/' + id, ''), id);
  assert.equal(requestIdFromLocation('/forge/workspace/requests/' + id, '/forge'), id);
  assert.equal(requestIdFromLocation('/team/tools/workspace/requests/' + id, '/team/tools'), id);
  assert.equal(requestIdFromLocation('/workspace-tools/forge/workspace/requests/' + id, '/workspace-tools/forge'), id);
  const negatives: [string, string][] = [
    ['/workspace', ''],
    ['/workspace/new', ''],
    ['/workspace/requests', ''],
    ['/workspace/requests/', ''],
    ['/workspace/requests/bad', ''],
    ['/workspace/requests/d0123456789abcdef0123456', ''],
    ['/workspace/requests/d0123456789ABCDEF01234567', ''],
    ['/workspace/requests/' + id + '/extra', ''],
    ['/user2/repo1/workspace/requests/' + id, ''],
    ['/a/b/workspace/requests/' + id, ''],
    ['/team/tools/workspace/requests/' + id, ''],
    ['/team/tools/workspace/requests/' + id, '/team'],
    ['/forgeX/workspace/requests/' + id, '/forge'],
    ['/workspace/requests/' + id, '/forge'],
    ['/issues', ''],
    ['/', ''],
  ];
  for (const [path, prefix] of negatives) {
    assert.equal(requestIdFromLocation(path, prefix), '', `${prefix} ${path}`);
  }
});

test('request and new routes preserve the sub-path prefix', () => {
  assert.equal(requestRouteFor('', id), '/workspace/requests/' + id);
  assert.equal(requestRouteFor('/forge', id), '/forge/workspace/requests/' + id);
  assert.equal(requestRouteFor('/team/tools', id), '/team/tools/workspace/requests/' + id);
  assert.equal(requestRouteFor('/workspace-tools/forge', id), '/workspace-tools/forge/workspace/requests/' + id);
  assert.equal(newRouteFor(''), '/workspace/new');
  assert.equal(newRouteFor('/forge'), '/forge/workspace/new');
  assert.equal(newRouteFor('/team/tools'), '/team/tools/workspace/new');
  assert.equal(newRouteFor('/workspace-tools/forge'), '/workspace-tools/forge/workspace/new');
});

test('new route matches only the exact admitted fresh-entry path', () => {
  assert.equal(isNewRouteLocation('/workspace/new', ''), true);
  assert.equal(isNewRouteLocation('/forge/workspace/new', '/forge'), true);
  assert.equal(isNewRouteLocation('/team/tools/workspace/new', '/team/tools'), true);
  assert.equal(isNewRouteLocation('/workspace-tools/forge/workspace/new', '/workspace-tools/forge'), true);
  const negatives: [string, string][] = [
    ['/workspace', ''],
    ['/workspace/new/', ''],
    ['/workspace/new/extra', ''],
    ['/workspace/newx', ''],
    ['/workspace/requests/' + id, ''],
    ['/team/tools', '/team/tools'],
    ['/workspace-tools/forge', '/workspace-tools/forge'],
    ['/team/tools/workspace/new', ''],
    ['/team/tools/workspace/new', '/team'],
    ['/forgeX/workspace/new', '/forge'],
    ['/workspace/new', '/forge'],
    ['/', ''],
  ];
  for (const [path, prefix] of negatives) {
    assert.equal(isNewRouteLocation(path, prefix), false, `${prefix} ${path}`);
  }
});

test('intent identity reads the saved/resumed detail only', () => {
  assert.equal(
    intentIdentity(new CustomEvent('intent-saved', {detail: {intent: id, revision: 'r', state: 'draft'}})),
    id
  );
  assert.equal(intentIdentity(new CustomEvent('intent-saved', {detail: null})), '');
  assert.equal(intentIdentity(new CustomEvent('intent-saved', {detail: {intent: 7}})), '');
  assert.equal(intentIdentity(new CustomEvent('intent-saved')), '');
});

const savedEntry = {
  intent: id,
  revision: 'dffffffffffffffffffffffff',
  state: 'draft',
  objective: 'Build a tracker',
  repository: '0',
  created_unix: 7,
  effective_count: 0,
  task_count: 0,
};

test('inventory intents parse the saved list and reject malformed payloads', () => {
  const parsed = inventoryIntents(new CustomEvent('intent-inventory', {detail: {intents: [savedEntry]}}));
  assert.equal(parsed.length, 1);
  assert.equal(parsed[0]?.intent, id);
  assert.equal(parsed[0]?.objective, 'Build a tracker');
  assert.deepEqual(inventoryIntents(new CustomEvent('intent-inventory', {detail: {intents: []}})), []);
  assert.deepEqual(inventoryIntents(new CustomEvent('intent-inventory', {detail: null})), []);
  assert.deepEqual(inventoryIntents(new CustomEvent('intent-inventory')), []);
  assert.deepEqual(
    inventoryIntents(new CustomEvent('intent-inventory', {detail: {intents: [{...savedEntry, intent: 'bad'}]}})),
    []
  );
});

test('overview scope feed keeps valid entries and retires on malformed payloads', () => {
  assert.deepEqual(
    overviewScopeDetail(scopeEvent({repository: '7', intentID: id, issues: [{issue: '5', capabilities}]})),
    scope
  );
  assert.deepEqual(overviewScopeDetail(scopeEvent({repository: '', issues: []})), {repository: '', issues: []});
  assert.deepEqual(
    overviewScopeDetail(scopeEvent({repository: '7', issues: [{issue: ''}, {issue: '5'}, 'nope', null]})),
    {repository: '7', issues: [{issue: '5'}]}
  );
  assert.deepEqual(
    overviewScopeDetail(
      scopeEvent({repository: '7', issues: [{issue: '5', capabilities: {pause: {available: 'yes'}}}]})
    ),
    {repository: '7', issues: [{issue: '5'}]}
  );
  for (const detail of [
    null,
    undefined,
    {},
    {repository: 7, issues: []},
    {repository: '7'},
    {repository: '7', issues: {}},
  ]) {
    assert.equal(overviewScopeDetail(scopeEvent(detail)), undefined);
  }
  assert.equal(overviewScopeDetail(scopeEvent({repository: '7', intentID: 7, issues: []})), undefined);
});

test('scope equality compares values so refresh re-dispatch does not churn the root', () => {
  assert.equal(sameScope(undefined, undefined), true);
  assert.equal(sameScope(scope, scope), true);
  assert.equal(sameScope(scope, {repository: '7', intentID: id, issues: [{issue: '5', capabilities}]}), true);
  assert.equal(sameScope(scope, undefined), false);
  assert.equal(sameScope(scope, {repository: '', issues: []}), false);
  assert.equal(sameScope(scope, {repository: '7', intentID: 'dffffffffffffffffffffffff', issues: []}), false);
  assert.equal(sameScope(scope, {repository: '7', intentID: id, issues: [{issue: '6'}]}), false);
});

test('work children derive only from a live scope matching the retained identity', () => {
  const children = childScopeFromReading(setupReadingWith({}), context);
  assert.deepEqual(children, {scope, transport, repository: '7', repositoryPath: '/forgejo.test/alice/Alpha'});
  assert.deepEqual(
    childScopeFromReading(setupReadingWith({overviewScope: {repository: '7', issues: [{issue: '5'}]}}), context)?.scope,
    {repository: '7', issues: [{issue: '5'}]}
  );
  for (const reading of [
    setupReadingWith({overviewScope: undefined}),
    setupReadingWith({overviewScope: {repository: '', issues: []}}),
    setupReadingWith({overviewScope: {repository: '8', intentID: id, issues: [{issue: '5'}]}}),
    setupReadingWith({
      overviewScope: {repository: '7', intentID: 'dffffffffffffffffffffffff', issues: [{issue: '5'}]},
    }),
    setupReadingWith({stale: true}),
    setupReadingWith({setup: 'repositories'}),
  ]) {
    assert.equal(childScopeFromReading(reading, context), undefined);
  }
});

const watchProjectA = 'p' + '1'.repeat(24),
  watchProjectB = 'p' + '2'.repeat(24);

function watchEvent(detail: unknown): Event {
  return new CustomEvent('soda-factory-watch-request', {detail});
}

function watchSpace(environment: string, repository: string, runs: unknown[]): Space {
  return {environment: {id: environment, repository_id: repository}, factory_runs: runs} as unknown as Space;
}

test('watch requests resolve to the exact live space run or stay unresolved', () => {
  const live = {id: 'run-live', role: 'soda-coder', issue: '5', reconciled: false},
    settled = {id: 'run-old', role: 'soda-coder', issue: '5', reconciled: true};
  const spaces = [watchSpace(watchProjectA, '7', [live, settled]), watchSpace(watchProjectB, '8', [])];
  assert.equal(
    resolveWatchTarget(spaces, {run: 'run-live', repository: '7', environment: watchProjectA})?.run.id,
    'run-live'
  );
  assert.equal(resolveWatchTarget(spaces, {run: 'run-live', repository: '7'})?.space.environment.id, watchProjectA);
  for (const request of [
    {run: 'run-live', repository: '8', environment: watchProjectA},
    {run: 'run-live', repository: '7', environment: watchProjectB},
    {run: 'run-live', repository: '8', environment: watchProjectB},
    {run: 'run-missing', repository: '7'},
    {run: 'run-missing', repository: '7', environment: watchProjectA},
  ]) {
    assert.equal(resolveWatchTarget(spaces, request), undefined);
  }
});

test('watch request payloads admit exact bindings and ignore malformed events', () => {
  assert.deepEqual(
    factoryWatchRequestDetail(watchEvent({run: 'run-live', repository: '7', environment: watchProjectA})),
    {run: 'run-live', repository: '7', environment: watchProjectA}
  );
  assert.deepEqual(factoryWatchRequestDetail(watchEvent({run: 'run-live', repository: '7'})), {
    run: 'run-live',
    repository: '7',
  });
  for (const detail of [
    undefined,
    null,
    'run-live',
    [],
    {},
    {run: ''},
    {run: 'run-live'},
    {repository: '7'},
    {run: 'run-live', repository: ''},
    {run: 'run-live', repository: '7', environment: ''},
    {run: 'run-live', repository: '7', environment: 7},
  ]) {
    assert.equal(factoryWatchRequestDetail(watchEvent(detail)), undefined);
  }
});

test('detail runs select the single live run and bind its owning project', () => {
  const live = {id: 'run-live', role: 'soda-coder', issue: '5', reconciled: false},
    other = {id: 'run-other', role: 'soda-coder', issue: '9', reconciled: false};
  assert.deepEqual(detailRunsForIssue(watchSpace(watchProjectA, '7', [live, other]), '5'), {
    runIDs: ['run-live'],
    runID: 'run-live',
    projectID: watchProjectA,
    environmentID: watchProjectA,
  });
  assert.deepEqual(detailRunsForIssue(watchSpace(watchProjectA, '7', [live, {...live, id: 'run-live-2'}]), '5'), {
    runIDs: ['run-live', 'run-live-2'],
    projectID: watchProjectA,
    environmentID: watchProjectA,
  });
  assert.deepEqual(detailRunsForIssue(watchSpace(watchProjectA, '7', [{...live, reconciled: true}]), '5'), {
    runIDs: ['run-live'],
    projectID: watchProjectA,
    environmentID: watchProjectA,
  });
  assert.deepEqual(detailRunsForIssue(undefined, '5'), {runIDs: []});
});

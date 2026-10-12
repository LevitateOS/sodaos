import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {
  discoveryEmptyMessage,
  groupSavedRequests,
  recentRowKeys,
  requestRepositoryLabel,
  splitAttentionRows,
} from '../../frontend/spaces/sodaspaces-workspace-navigation';
import type {SavedRequest} from '../../frontend/spaces/sodaspaces-intent-response';
import type {NavReading} from '../../frontend/spaces/sodaspaces-workspace-navigation';
import type {Space} from '../../frontend/spaces/sodaspaces-inventory-response';
import type {TerminalMetadata} from '../../frontend/spaces/sodaspaces-terminal-response';
import type {Row, Slot} from '../../frontend/spaces/sodaspaces-workspace-types';

const base = {attentionOnly: false, search: '', thisPage: false};
test('empty discovery distinguishes loading, unavailable, filtered and truly empty', () => {
  assert.equal(discoveryEmptyMessage({...base, available: false, busy: true}), 'Loading your projects and terminals…');
  assert.equal(discoveryEmptyMessage({...base, available: false, busy: false}), 'Status unavailable.');
  assert.equal(
    discoveryEmptyMessage({...base, available: true, busy: false, attentionOnly: true}),
    'No matching sessions need attention.'
  );
  assert.equal(discoveryEmptyMessage({...base, available: true, busy: false, search: 'x'}), 'No matches.');
  assert.equal(discoveryEmptyMessage({...base, available: true, busy: false}), 'No authorized projects available.');
});

const now = 1000000;
const metadata: TerminalMetadata = {
  id: 'a'.repeat(32),
  environment_id: 'p' + '1'.repeat(24),
  repository_id: '7',
  user_id: '1',
  login: 'alice',
  name: '',
  created_at: 1,
  ready: true,
  attached: false,
  state: 'ready',
};
const slot = (key: string, observedAt: number) => ({key, observedAt, metadata}) as unknown as Slot;
const reading = (slots: Slot[]): NavReading => ({stale: false, available: true, slots, observedAt: now, now});
const space = {
  login: 'alice',
  execution_allowed: true,
  authority_unavailable: false,
  native_unavailable: false,
} as unknown as Space;

test('recent keys order stable identities by last observation with deterministic ties', () => {
  const slots = [slot('fresh', now), slot('old', now - 90001)];
  const shown: Row[] = [{key: 'old', metadata}, {key: 'fresh', metadata}, {key: 'unsaved'}, {key: 'old', metadata}];
  assert.deepEqual(recentRowKeys(shown, slots, 5), ['fresh', 'old', 'unsaved']);
  assert.deepEqual(recentRowKeys(shown, slots, 1), ['fresh']);
  assert.deepEqual(recentRowKeys(shown, slots, 0), []);
  assert.deepEqual(recentRowKeys([{key: 'b'}, {key: 'a'}], [], 5), ['a', 'b']);
});

test('attention split keeps ordinary sessions out of the attention group', () => {
  const slots = [slot('fresh', now), slot('old', now - 90001)];
  const shown: Row[] = [
    {key: 'fresh', metadata},
    {key: 'old', metadata},
  ];
  const groups = splitAttentionRows(reading(slots), space, shown);
  assert.deepEqual(
    groups.attention.map((row) => row.key),
    ['old']
  );
  assert.deepEqual(
    groups.sessions.map((row) => row.key),
    ['fresh']
  );
});

const summary = (intent: string, state: SavedRequest['state'], created_unix: number): SavedRequest => ({
  intent,
  revision: 'r' + intent,
  state,
  objective: 'Build ' + intent,
  repository: '0',
  created_unix,
  effective_count: 0,
  task_count: 0,
});
test('saved requests group by lifecycle with recent first and stable identity ties', () => {
  const groups = groupSavedRequests([
    summary('d3', 'abandoned', 30),
    summary('d1', 'draft', 10),
    summary('d2', 'accepted', 20),
    summary('d0', 'draft', 10),
    summary('d4', 'rejected', 40),
  ]);
  assert.deepEqual(
    groups.drafts.map((row) => row.intent),
    ['d0', 'd1']
  );
  assert.deepEqual(
    groups.accepted.map((row) => row.intent),
    ['d2']
  );
  assert.deepEqual(
    groups.closed.map((row) => row.intent),
    ['d4', 'd3']
  );
});
test('unbound requests label their missing repository honestly', () => {
  assert.equal(requestRepositoryLabel('0'), 'No repository yet');
  assert.equal(requestRepositoryLabel(''), 'No repository yet');
  assert.equal(requestRepositoryLabel('42'), '42');
});

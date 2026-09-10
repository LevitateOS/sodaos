import test from 'node:test';
import assert from 'node:assert/strict';
import {spacesResponse, terminalResponse} from '../../frontend/spaces/sodaspaces-api';
const binding = {expectedUserId: '1', repositoryId: '7', environmentId: 'p0123456789abcdef01234567', login: 'original-alice'};
const terminal = {id: 'a'.repeat(32), request_id: 'b'.repeat(32), environment_id: binding.environmentId, repository_id: '7', user_id: '1', login: binding.login,
  name: '編集', created_at: 100, hard_until: 43300, retain_until: 1900, effective_until: 1900, ready: true, attached: true, state: 'ready'};
const row = {environment: {id: binding.environmentId, repository_id: '7', owner_id: '1', name: 'repo', repository: 'alice/repo', provisioned: true}, login: binding.login,
  environment_administrator: false, authority_unavailable: false, native_unavailable: false, observed: {id: binding.environmentId, running: true}, terminals: [terminal]};
test('terminal metadata preserves exact binding, correlated attempt and effective deadlines', () => {
  assert.deepEqual(terminalResponse({terminal}, binding), terminal);
  assert.equal(terminalResponse({terminal: null}, binding), null);
  assert.equal(terminalResponse({terminal: {...terminal, state: 'ended', ready: false, attached: false}}, binding)?.state, 'ended');
});
for (const delta of [{user_id: '2'}, {repository_id: '8'}, {login: 'renamed'}, {environment_id: 'other'}, {request_id: 'pending'}, {name: 'a'.repeat(81)}, {name: '\u202econtrol'}, {hard_until: 99999}, {retain_until: 50000}, {effective_until: 0}, {state: 'ended'}, {state: 'agent-waiting'}]) test(`invalid terminal metadata ${JSON.stringify(delta)}`, () => {
  assert.throws(() => terminalResponse({terminal: {...terminal, ...delta}}, binding));
});
test('Spaces validates visible rows and does not turn incomplete into complete', () => {
  const result = spacesResponse({items: [row], complete: false}, '1');
  assert.equal(result.complete, false); assert.deepEqual(result.items[0]?.terminals, [terminal]);
  assert.throws(() => spacesResponse({items: [row], complete: true}, '2'));
  assert.throws(() => spacesResponse({items: [row, row], complete: true}, '1'));
  assert.throws(() => spacesResponse({items: [{...row, terminals: [terminal, terminal]}], complete: true}, '1'));
  assert.throws(() => spacesResponse({items: Array(33).fill(row), complete: true}, '1'));
});
test('degraded collection cannot carry session metadata or elevation', () => {
  assert.throws(() => spacesResponse({items: [{...row, authority_unavailable: true}], complete: false}, '1'));
  assert.throws(() => spacesResponse({items: [{...row, authority_unavailable: true, terminals: [], environment_administrator: true}], complete: false}, '1'));
  assert.equal(spacesResponse({items: [{...row, authority_unavailable: true, terminals: []}], complete: false}, '1').items.length, 1);
});

import test from 'node:test';
import assert from 'node:assert/strict';
import {spacesResponse, terminalResponse, repositoryChoices} from '../../frontend/spaces/sodaspaces-api';
const binding = {expectedUserId: '1', repositoryId: '7', environmentId: 'p0123456789abcdef01234567', login: 'original-alice'};
const terminal = {id: 'a'.repeat(32), environment_id: binding.environmentId, repository_id: '7', user_id: '1', login: binding.login,
  name: '編集', created_at: 100, ready: true, attached: true, state: 'ready'};
const row = {environment: {id: binding.environmentId, repository_id: '7', owner_id: '1', name: 'repo', repository: 'alice/repo', provisioned: true}, login: binding.login,
  environment_administrator: false, authority_unavailable: false, native_unavailable: false, observed: {id: binding.environmentId, running: true}, terminals: [terminal]};
test('repository choices bind stable identity and distinguish existing reservations from Create', () => {
  const repo = {id: '7', owner: 'alice', name: 'repo', can_create: true, project: null};
  const result = {items: [repo], page: 1, more: false, limited: false};
  assert.equal(repositoryChoices(result, 1).items[0]?.canCreate, true);
  assert.equal(repositoryChoices({...result, items: [{...repo, can_create: false, project: {id: binding.environmentId, provisioned: false}}]}, 1).items[0]?.project?.provisioned, false);
  for (const delta of [{page: 2}, {more: true, limited: true}, {limited: true}, {items: [repo, repo]}, {items: Array(13).fill(repo)}, {items: [{...repo, id: '07'}]}, {items: [{...repo, owner: '../private'}]}, {items: [{...repo, can_create: false}]}, {items: [{...repo, project: {id: binding.environmentId, provisioned: true}}]}]) assert.throws(() => repositoryChoices({...result, ...delta}, 1));
  assert.equal(repositoryChoices({...result, page: 100, limited: true}, 100).limited, true);
});
test('terminal metadata preserves exact native binding without retired lifetime fields', () => {
  assert.deepEqual(terminalResponse({terminal}, binding), terminal);
  assert.equal(terminalResponse({terminal: null}, binding), null);
  assert.deepEqual(terminalResponse({terminal: {...terminal, request_id: 'b'.repeat(32), hard_until: 43300, retain_until: 1900}}, binding), terminal);
  assert.equal(terminalResponse({terminal: {...terminal, state: 'ended', ready: false, attached: false}}, binding)?.state, 'ended');
});
for (const delta of [{user_id: '2'}, {repository_id: '8'}, {login: 'renamed'}, {environment_id: 'other'}, {id: 'pending'}, {name: 'a'.repeat(81)}, {name: '\u202econtrol'}, {created_at: 0}, {created_at: true}, {created_at: NaN}, {state: 'ended'}, {state: 'agent-waiting'}]) test(`invalid terminal metadata ${JSON.stringify(delta)}`, () => {
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

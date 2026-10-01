import test from 'node:test';
import assert from 'node:assert/strict';
import {
  spacesResponse,
  terminalResponse,
  repositoryChoices,
  factoryAuthorityText,
  factoryControlText,
  factoryCommandId,
} from '../../frontend/spaces/sodaspaces-api';
const binding = {
  expectedUserId: '1',
  repositoryId: '7',
  environmentId: 'p0123456789abcdef01234567',
  login: 'original-alice',
};
const terminal = {
  id: 'a'.repeat(32),
  environment_id: binding.environmentId,
  repository_id: '7',
  user_id: '1',
  login: binding.login,
  name: '編集',
  created_at: 100,
  ready: true,
  attached: true,
  state: 'ready',
};
const row = {
  environment: {
    id: binding.environmentId,
    repository_id: '7',
    owner_id: '1',
    name: 'repo',
    repository: 'alice/repo',
    provisioned: true,
  },
  login: binding.login,
  environment_administrator: false,
  execution_allowed: true,
  authority_unavailable: false,
  native_unavailable: false,
  observed: {id: binding.environmentId, running: true},
  terminals: [terminal],
};
test('repository choices bind stable identity and distinguish existing reservations from Create', () => {
  const repo = {id: '7', owner: 'alice', name: 'repo', can_create: true, project: null};
  const result = {items: [repo], next_cursor: 'opaque-next'};
  assert.equal(repositoryChoices(result, 1).items[0]?.canCreate, true);
  assert.equal(repositoryChoices(result, 1).nextCursor, 'opaque-next');
  assert.equal(
    repositoryChoices(
      {...result, items: [{...repo, can_create: false, project: {id: binding.environmentId, provisioned: false}}]},
      1
    ).items[0]?.project?.provisioned,
    false
  );
  for (const delta of [
    {page: 1},
    {more: false},
    {limited: false},
    {next_cursor: 3},
    {next_cursor: 'x'.repeat(4097)},
    {items: [repo, repo]},
    {items: Array(13).fill(repo)},
    {items: [{...repo, id: '07'}]},
    {items: [{...repo, owner: '../private'}]},
    {items: [{...repo, can_create: false}]},
    {items: [{...repo, project: {id: binding.environmentId, provisioned: true}}]},
  ])
    assert.throws(() => repositoryChoices({...result, ...delta}, 1));
  assert.equal(repositoryChoices({items: [repo]}, 100).nextCursor, '');
});
test('terminal metadata preserves exact native binding without retired lifetime fields', () => {
  assert.deepEqual(terminalResponse({terminal}, binding), terminal);
  assert.equal(terminalResponse({terminal: null}, binding), null);
  assert.deepEqual(
    terminalResponse(
      {terminal: {...terminal, request_id: 'b'.repeat(32), hard_until: 43300, retain_until: 1900}},
      binding
    ),
    terminal
  );
  assert.equal(
    terminalResponse({terminal: {...terminal, state: 'ended', ready: false, attached: false}}, binding)?.state,
    'ended'
  );
});
for (const delta of [
  {user_id: '2'},
  {repository_id: '8'},
  {login: 'renamed'},
  {environment_id: 'other'},
  {id: 'pending'},
  {name: 'a'.repeat(81)},
  {name: '\u202econtrol'},
  {created_at: 0},
  {created_at: true},
  {created_at: NaN},
  {state: 'ended'},
  {state: 'agent-waiting'},
])
  test(`invalid terminal metadata ${JSON.stringify(delta)}`, () => {
    assert.throws(() => terminalResponse({terminal: {...terminal, ...delta}}, binding));
  });
test('Spaces validates visible rows and does not turn incomplete into complete', () => {
  const actor = {id: '1', login: 'soda-tester'};
  const result = spacesResponse({actor, items: [row], complete: false});
  assert.equal(result.complete, false);
  assert.deepEqual(result.items[0]?.terminals, [terminal]);
  assert.throws(() => spacesResponse({actor: {...actor, id: '2'}, items: [row], complete: true}));
  assert.throws(() => spacesResponse({actor, items: [row, row], complete: true}));
  assert.throws(() => spacesResponse({actor, items: [{...row, terminals: [terminal, terminal]}], complete: true}));
  assert.throws(() => spacesResponse({actor, items: Array(33).fill(row), complete: true}));
});
test('degraded collection cannot carry session metadata or elevation', () => {
  const actor = {id: '1', login: 'soda-tester'};
  assert.throws(() => spacesResponse({actor, items: [{...row, authority_unavailable: true}], complete: false}));
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, authority_unavailable: true, terminals: [], environment_administrator: true}],
      complete: false,
    })
  );
  assert.equal(
    spacesResponse({actor, items: [{...row, authority_unavailable: true, terminals: []}], complete: false}).items
      .length,
    1
  );
});
test('Spaces admits the factory authority verdict and renders its status', () => {
  const actor = {id: '1', login: 'soda-tester'};
  const ready = {effective: true, dispatch_open: true, missing: []};
  const parsed = spacesResponse({actor, items: [{...row, factory_authority: ready}], complete: true});
  assert.deepEqual(parsed.items[0]?.factory_authority, ready);
  assert.equal(parsed.items.length, 1);
  const absent = spacesResponse({actor, items: [row], complete: true});
  assert.equal(absent.items[0]?.factory_authority, undefined);
  assert.equal(factoryAuthorityText(undefined), '');
  assert.equal(factoryAuthorityText(ready), ' · Factory ready');
  assert.equal(
    factoryAuthorityText({effective: false, dispatch_open: false, missing: ['policy_paused', 'dispatch_closed']}),
    ' · Factory needs: policy_paused, dispatch_closed'
  );
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, factory_authority: {effective: true, dispatch_open: true, missing: ['policy_paused']}}],
      complete: true,
    })
  );
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, factory_authority: {effective: 'yes', dispatch_open: true, missing: []}}],
      complete: true,
    })
  );
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, factory_authority: {effective: false, dispatch_open: false, missing: ['Policy Paused']}}],
      complete: true,
    })
  );
});
test('Spaces admits the factory control state and renders its status', () => {
  const actor = {id: '1', login: 'soda-tester'};
  const paused = {dispatch_open: false, paused: true, unsettled_runs: 2, withdrawal_cause: 'control_paused'};
  const parsed = spacesResponse({actor, items: [{...row, factory_control: paused}], complete: true});
  assert.deepEqual(parsed.items[0]?.factory_control, paused);
  const absent = spacesResponse({actor, items: [row], complete: true});
  assert.equal(absent.items[0]?.factory_control, undefined);
  assert.equal(factoryControlText(undefined), '');
  assert.equal(factoryControlText({dispatch_open: true, paused: false, unsettled_runs: 0}), '');
  assert.equal(factoryControlText(paused), ' · Factory paused, dispatch closed: control_paused, 2 unsettled');
  assert.equal(factoryControlText({dispatch_open: true, paused: false, unsettled_runs: 1}), ' · Factory 1 unsettled');
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, factory_control: {...paused, unsettled_runs: -1}}],
      complete: true,
    })
  );
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, factory_control: {...paused, withdrawal_cause: 'Control Paused'}}],
      complete: true,
    })
  );
  assert.throws(() =>
    spacesResponse({
      actor,
      items: [{...row, factory_control: {dispatch_open: false, paused: false, unsettled_runs: 0}}],
      complete: true,
    })
  );
});
test('Factory command identities are idempotent ledger keys', () => {
  const first = factoryCommandId();
  assert.match(first, /^[a-f0-9]{32}$/);
  assert.notEqual(first, factoryCommandId());
});

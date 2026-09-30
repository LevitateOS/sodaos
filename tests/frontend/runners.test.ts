import assert from 'node:assert/strict';
import test from 'node:test';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response';
import type {Runner} from '../../frontend/runners/soda-runner-types';

const origin = 'https://forgejo.example.test';
const exampleRunner = (): Runner => ({
  id: 'one',
  provider: 'forgejo',
  registration_url: origin,
  account: 'soda-runner-one',
  architecture: 'x86-64',
  version: 'fixture',
  capacity: 1,
  service: {load: 'loaded', active: 'active', sub: 'running', enabled: 'enabled'},
});

test('runner response enforces native one-slot validation', () => {
  assert.throws(() =>
    decodeRunnerResponse('list', {runners: [], runner_count: 1, active_listeners: 0, total_capacity: 0})
  );
  assert.throws(() => decodeRunnerResponse('remove', {ok: false}));
  assert.throws(() =>
    decodeRunnerResponse('list', {runners: [], runner_count: 0, active_listeners: -1, total_capacity: 0})
  );
  assert.equal(decodeRunnerResponse('stop', {ok: true}).ok, true);
  const omittedOrigin = {
    complete: true,
    unavailable: [],
    runners: [],
    runner_count: 0,
    active_listeners: 0,
    total_capacity: 0,
  };
  assert.deepEqual(decodeRunnerResponse('list', omittedOrigin), omittedOrigin);
});

test('partial runner response retains known rows and refuses false completeness or counts', () => {
  const row = exampleRunner();
  const partial = {
    forgejo_url: origin,
    complete: false,
    unavailable: ['broken'],
    runners: [row, {...row, id: 'two', account: 'soda-runner-two', service: null, version: ''}],
    runner_count: 2,
    active_listeners: 1,
    total_capacity: 2,
  };
  assert.deepEqual(decodeRunnerResponse('list', partial), partial);
  for (const change of [
    {complete: true},
    {total_capacity: 3},
    {active_listeners: 2},
    {unavailable: ['one']},
    {unavailable: ['../bad']},
    {unavailable: ['broken', 'broken']},
  ])
    assert.throws(() => decodeRunnerResponse('list', {...partial, ...change}));
  assert.throws(() =>
    decodeRunnerResponse('list', {...partial, runners: [{...row, account: 'root'}, partial.runners[1]]})
  );
});

import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {intentPlanLaunch, intentPlanStatus, planActionBody} from '../../frontend/spaces/sodaspaces-intent-response';

const launch = {
  assignment_id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
  run_id: 'b1b2c3d4e5f60718293a4b5c6d7e8f90',
  revision_id: 'd888888888888888888888888',
};

test('plan launch binds assignment, run and revision', () => {
  const parsed = intentPlanLaunch(launch);
  assert.equal(parsed.assignment, launch.assignment_id);
  assert.equal(parsed.run, launch.run_id);
  assert.equal(parsed.revision, launch.revision_id);
});

test('plan action posts revision and operation only', () => {
  assert.deepEqual(planActionBody('d888888888888888888888888', 'propose_intent'), {
    revision: 'd888888888888888888888888',
    operation: 'propose_intent',
  });
  assert.deepEqual(planActionBody('d888888888888888888888888', 'decomposition_options'), {
    revision: 'd888888888888888888888888',
    operation: 'decomposition_options',
  });
  assert.throws(() => planActionBody('d9999', 'propose_intent'));
  assert.throws(() => planActionBody('d888888888888888888888888', 'nope'));
});

test('plan launch rejects substituted identities', () => {
  assert.throws(() => intentPlanLaunch({...launch, revision_id: 'd9999'}));
  assert.throws(() => intentPlanLaunch({...launch, run_id: 'short'}));
  assert.throws(() => intentPlanLaunch({...launch, assignment_id: undefined}));
});

test('plan status reads every reported shape', () => {
  const base = {revision: 'd888888888888888888888888'};
  assert.equal(intentPlanStatus({...base, state: 'none'}).state, 'none');
  const pending = intentPlanStatus({
    ...base,
    state: 'pending',
    assignment_id: launch.assignment_id,
    run_id: launch.run_id,
  });
  assert.equal(pending.state, 'pending');
  assert.equal(pending.run, launch.run_id);
  const completed = intentPlanStatus({...base, state: 'completed', summary: 'two views ready', candidates: 2});
  assert.equal(completed.state, 'completed');
  assert.equal(completed.candidates, 2);
  const attention = intentPlanStatus({...base, state: 'attention', summary: 'waiting', attention: 'Decide the queue.'});
  assert.equal(attention.state, 'attention');
  assert.equal(attention.attention, 'Decide the queue.');
  assert.equal(attention.candidates, 0);
  assert.equal(intentPlanStatus({...base, state: 'failed', summary: 'crashed'}).state, 'failed');
  assert.equal(intentPlanStatus({...base, state: 'cancelled', summary: 'stopped'}).state, 'cancelled');
});

test('plan status rejects unknown states and out-of-shape payloads', () => {
  const base = {revision: 'd888888888888888888888888'};
  assert.throws(() => intentPlanStatus({...base, state: 'running'}));
  assert.throws(() => intentPlanStatus({...base, state: 'completed', candidates: 4}));
  assert.throws(() => intentPlanStatus({...base, state: 'completed', candidates: -1}));
  assert.throws(() => intentPlanStatus({state: 'none'}));
  assert.throws(() => intentPlanStatus({...base, state: 'attention', attention: ''}));
});

import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {SodaRequestError} from '../../frontend/spaces/sodaspaces-api.js';
import {
  generationFailureNotice,
  generationWaitNotice,
  intentPlanLaunch,
  intentPlanStatus,
  planActionBody,
  planWait,
  runPlanWork,
} from '../../frontend/spaces/sodaspaces-intent-response';

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

test('plan work renders the wait without launch or refetch', async () => {
  let fetched = false;
  const outcome = await runPlanWork(
    async () => ({
      launched: false,
      waits: [{reason: 'preparation', detail: 'no ready role preparation matches the current decisions'}],
    }),
    async () => {
      fetched = true;
      return {};
    },
    'd888888888888888888888888'
  );
  assert.equal(outcome.status, undefined);
  assert.ok(outcome.notice.includes('Planning is waiting'));
  assert.ok(outcome.notice.includes('no ready role preparation matches the current decisions'));
  assert.ok(outcome.notice.includes('No work was started'));
  assert.equal(fetched, false);
});

test('plan work launches then reports pending through stub transport', async () => {
  const outcome = await runPlanWork(
    async () => launch,
    async () => ({
      revision: 'd888888888888888888888888',
      state: 'pending',
      assignment_id: launch.assignment_id,
      run_id: launch.run_id,
    }),
    'd888888888888888888888888'
  );
  assert.equal(outcome.status?.state, 'pending');
  assert.equal(outcome.status?.run, launch.run_id);
  assert.ok(outcome.notice.includes('Planning is running'));
});

test('plan work reports completed candidates', async () => {
  const outcome = await runPlanWork(
    async () => launch,
    async () => ({
      revision: 'd888888888888888888888888',
      state: 'completed',
      summary: 'two views ready',
      candidates: 2,
    }),
    'd888888888888888888888888'
  );
  assert.equal(outcome.status?.state, 'completed');
  assert.equal(outcome.status?.candidates, 2);
});

test('plan work refuses substituted revisions', async () => {
  await assert.rejects(
    runPlanWork(
      async () => ({...launch, revision_id: 'd999999999999999999999999'}),
      async () => ({revision: 'd888888888888888888888888', state: 'none'}),
      'd888888888888888888888888'
    )
  );
  await assert.rejects(
    runPlanWork(
      async () => launch,
      async () => ({revision: 'd999999999999999999999999', state: 'pending'}),
      'd888888888888888888888888'
    )
  );
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

test('generate wait renders the recorded-plan wait without claiming work', () => {
  const body = {
    launched: false,
    waits: [{reason: 'already_assigned', detail: 'recorded decomposition plan is still running'}],
  };
  assert.equal(planWait(body), true);
  assert.equal(
    generationWaitNotice(body),
    'Factory work is waiting: recorded decomposition plan is still running No native work was started.'
  );
  assert.equal(planWait({launched: true, waits: []}), false);
  assert.equal(
    generationWaitNotice({launched: false, waits: []}),
    'Factory work is waiting: the required preparation is not ready No native work was started.'
  );
});

test('generate failure maps recorded-plan refusals to actionable notices', () => {
  assert.match(
    generationFailureNotice(new SodaRequestError(409, 'planning_choice_unavailable')),
    /completed decomposition plan/
  );
  assert.match(generationFailureNotice(new SodaRequestError(409, 'planning_attention')), /product decision/);
  assert.match(generationFailureNotice(new SodaRequestError(412, 'stale_planning_choice')), /revise it and plan again/);
  assert.match(generationFailureNotice(new SodaRequestError(503)), /could not be confirmed/);
  assert.match(generationFailureNotice(Error('boom')), /could not be confirmed/);
});

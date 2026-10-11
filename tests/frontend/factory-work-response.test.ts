import test from 'node:test';
import assert from 'node:assert/strict';
import {
  factoryCommandRecord,
  factoryIssueDetail,
  factoryRepositorySummary,
  pauseReceipt,
  pendingRuns,
  resumeReceipt,
  retryDecision,
  stopReceipt,
  takeoverRecord,
} from '../../frontend/spaces/sodaspaces-factory-work-response';

const repository = '7';
const issue = '12';
const head = 'a'.repeat(40);
const base = 'b'.repeat(40);
const command = 'c'.repeat(32);
const run = 'd'.repeat(32);

const detail = {
  repository,
  issue,
  status: {valid: true, revision: 4, reasons: []},
  readiness: {
    readiness: 'blocked',
    reason: 'prereq_code_pending',
    revision: 9,
    assessed_unix: 1720000000,
    blockers: [
      {
        code: 'prereq_code_pending',
        detail: '11',
        endpoint_repository: '7',
        endpoint_issue: '11',
        resolution: 'Wait for issue #11 to finish its coded outcome.',
      },
    ],
  },
  checks: {
    verdict: 'pending',
    reason: 'checks_pending',
    resolution: 'Wait for the pending checks to report.',
    head_oid: head,
    base_oid: base,
    pr_number: '34',
    pr_id: '56',
    policy_revision: 2,
    native_revision: 8,
    revision: 5,
    assessed_unix: 1720000100,
    results: [{context: 'build', passed: false, state: 'pending'}],
  },
  merge: {
    stage: 'open',
    resolution: 'The merge is still reconciling.',
    head_oid: head,
    base_oid: base,
    pr_number: '34',
    pr_id: '56',
    revision: 1,
  },
};

test('issue detail binds acceptance, readiness, checks and merge to the exact issue', () => {
  const parsed = factoryIssueDetail(detail, repository, issue);
  assert.equal(parsed.status.valid, true);
  assert.equal(parsed.status.withdrawn, false);
  assert.equal(parsed.readiness?.readiness, 'blocked');
  assert.equal(parsed.readiness?.blockers[0]?.endpoint_issue, '11');
  assert.equal(parsed.checks?.verdict, 'pending');
  assert.equal(parsed.checks?.head_oid, head);
  assert.equal(parsed.merge?.stage, 'open');
  const minimal = factoryIssueDetail({repository, issue, status: {valid: false, revision: 0}}, repository, issue);
  assert.equal(minimal.readiness, undefined);
  assert.equal(minimal.checks, undefined);
  assert.equal(minimal.merge, undefined);
  for (const delta of [
    {repository: '8'},
    {issue: '13'},
    {status: {valid: 'yes', revision: 0}},
    {status: {valid: true, revision: -1}},
    {readiness: {...detail.readiness, readiness: 'launching'}},
    {readiness: {...detail.readiness, blockers: [{code: '', resolution: 'x'}]}},
    {readiness: {...detail.readiness, blockers: [{code: 'x', resolution: 'x', detail: 'bad\u0007'}]}},
    {checks: {...detail.checks, verdict: 'passing'}},
    {checks: {...detail.checks, head_oid: 'short'}},
    {checks: {...detail.checks, pr_number: '0x12'}},
    {merge: {...detail.merge, stage: 'merging'}},
    {merge: {...detail.merge, reason: 'invented_reason'}},
    {merge: {...detail.merge, merged_commit: 'short'}},
  ])
    assert.throws(() => factoryIssueDetail({...detail, ...delta}, repository, issue));
});

test('repository summary binds queue, setup preparation and effective authority', () => {
  const parsed = factoryRepositorySummary(
    {
      repository,
      queue: {queued: 2, active: 1},
      effective: {effective: false, dispatch_open: true, missing: ['sponsorship_missing']},
      preparation: {project: 'p0123456789abcdef01234567', ready: false},
    },
    repository
  );
  assert.equal(parsed.queue.queued, 2);
  assert.equal(parsed.effective.missing.length, 1);
  assert.equal(parsed.preparation?.ready, false);
  const bare = factoryRepositorySummary(
    {repository, queue: {queued: 0, active: 0}, effective: {effective: true, dispatch_open: true, missing: []}},
    repository
  );
  assert.equal(bare.preparation, undefined);
  for (const delta of [
    {repository: '8'},
    {queue: {queued: -1, active: 0}},
    {effective: {effective: true, dispatch_open: true, missing: ['Bad Code']}},
    {preparation: {project: 'nope', ready: true}},
  ])
    assert.throws(() =>
      factoryRepositorySummary(
        {
          repository,
          queue: {queued: 0, active: 0},
          effective: {effective: true, dispatch_open: true, missing: []},
          ...delta,
        },
        repository
      )
    );
});

test('pause and resume receipts bind the exact command and stay factual', () => {
  const paused = pauseReceipt(
    {
      command_id: command,
      paused: true,
      withdrawal: {repository, revision: 6, cause: 'control_paused', closed_by: 'os-uid:1000'},
      runs: [{id: run, confirmed: true, uncertain: false, outcome: 'cancelled'}],
    },
    command,
    repository
  );
  assert.equal(paused.runs[0]?.outcome, 'cancelled');
  const resumed = resumeReceipt(
    {
      command_id: command,
      revision: 7,
      reopened: true,
      effective: {effective: true, dispatch_open: true, missing: []},
    },
    command
  );
  assert.equal(resumed.reopened, true);
  assert.throws(() =>
    pauseReceipt(
      {command_id: command, paused: false, withdrawal: {repository, revision: 0, cause: 'x', closed_by: 'y'}, runs: []},
      command,
      repository
    )
  );
  assert.throws(() =>
    pauseReceipt(
      {
        command_id: command,
        paused: true,
        withdrawal: {repository: '8', revision: 0, cause: 'x', closed_by: 'y'},
        runs: [],
      },
      command,
      repository
    )
  );
  assert.throws(() =>
    resumeReceipt(
      {
        command_id: 'e'.repeat(32),
        revision: 0,
        reopened: false,
        effective: {effective: false, dispatch_open: false, missing: []},
      },
      command
    )
  );
});

test('run intervention outcomes bind the exact run and command', () => {
  const stopped = stopReceipt({run_id: run, confirmed: true, uncertain: false}, run);
  assert.equal(stopped.confirmed, true);
  const uncertain = stopReceipt({run_id: run, confirmed: false, uncertain: true, reason: 'effect_indeterminate'}, run);
  assert.equal(uncertain.uncertain, true);
  const retried = retryDecision(
    {prior_run: run, command_id: command, reason: 'run_failed', queued: true},
    command,
    run
  );
  assert.equal(retried.queued, true);
  const taken = takeoverRecord(
    {
      run,
      member: 'soda-tester',
      project: 'p0123456789abcdef01234567',
      destination: '/home/soda-tester/takeover/d',
      command_id: command,
      copied: '2024-06-30T00:00:00Z',
    },
    command,
    run
  );
  assert.equal(taken.member, 'soda-tester');
  assert.throws(() => stopReceipt({run_id: 'e'.repeat(32), confirmed: true, uncertain: false}, run));
  assert.throws(() =>
    retryDecision({prior_run: run, command_id: command, reason: 'run_failed', queued: false}, command, run)
  );
  assert.throws(() =>
    takeoverRecord(
      {
        run,
        member: 'soda-tester',
        project: 'p0123456789abcdef01234567',
        destination: '/home/other/takeover',
        command_id: command,
        copied: '2024-06-30T00:00:00Z',
      },
      command,
      run
    )
  );
});

test('command read-back and pending-runs refusals stay attributable', () => {
  const record = factoryCommandRecord(
    {id: command, type: 'stop', principal: 'os-uid:1000', digest: 'f'.repeat(64), target: run},
    command
  );
  assert.equal(record.type, 'stop');
  const pending = pendingRuns({error: 'pending_runs', runs: [run], message: 'Recorded runs are not settled.'});
  assert.deepEqual(pending.runs, [run]);
  assert.throws(() =>
    factoryCommandRecord({id: command, type: 'stop', principal: 'os-uid:1000', digest: 'short'}, command)
  );
  assert.throws(() => pendingRuns({error: 'pending_runs', runs: ['short'], message: 'x'}));
});

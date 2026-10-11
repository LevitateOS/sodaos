import test from 'node:test';
import assert from 'node:assert/strict';
import {
  describeIntervention,
  describeIssueDetail,
  interventionResultText,
  nativeIssueLink,
  nativePullLink,
} from '../../frontend/spaces/sodaspaces-factory-work-view';
import type {FactoryIssueDetail} from '../../frontend/spaces/sodaspaces-factory-work-response';

const path = '/soda-tester/demo';
const head = 'a'.repeat(40);
const base = 'b'.repeat(40);
const base_: FactoryIssueDetail = {
  repository: '7',
  issue: '12',
  status: {valid: true, withdrawn: false, revision: 1, reasons: []},
};
const checks = {
  verdict: 'pass',
  reason: 'checks_pass',
  resolution: 'Every required check succeeded on the exact head.',
  head_oid: head,
  base_oid: base,
  pr_number: '34',
  pr_id: '56',
  policy_revision: 2,
  native_revision: 8,
  revision: 5,
  assessed_unix: 1720000100,
  results: [{context: 'build', passed: true, state: 'success'}],
};
const merged = {
  stage: 'merged',
  resolution: 'The exact candidate merged and its completion is confirmed.',
  head_oid: head,
  base_oid: base,
  merged_commit: head,
  pr_number: '34',
  pr_id: '56',
  revision: 1,
  reason: 'merge_completed',
  finished_unix: 1720000200,
};

test('delivery needs both merge and closure evidence', () => {
  const delivered = describeIssueDetail(
    {
      ...base_,
      readiness: {readiness: 'completed', reason: 'eligible', revision: 1, assessed_unix: 1, blockers: []},
      merge: merged,
    },
    path
  );
  assert.equal(delivered.delivery, 'delivered');
  const pending = describeIssueDetail(
    {
      ...base_,
      readiness: {readiness: 'active', reason: 'eligible', revision: 1, assessed_unix: 1, blockers: []},
      merge: merged,
    },
    path
  );
  assert.equal(pending.delivery, 'merge_confirmed_closure_pending');
  const orphan = describeIssueDetail(
    {...base_, readiness: {readiness: 'completed', reason: 'eligible', revision: 1, assessed_unix: 1, blockers: []}},
    path
  );
  assert.equal(orphan.delivery, 'recorded_complete_without_merge');
  assert.equal(describeIssueDetail(base_, path).delivery, 'unrecorded');
});

test('check and readiness verdicts map to distinct delivery states', () => {
  const failed = describeIssueDetail(
    {...base_, checks: {...checks, verdict: 'failed', results: [{context: 'build', passed: false, state: 'failure'}]}},
    path
  );
  assert.equal(failed.delivery, 'checks_failed');
  assert.deepEqual(failed.failedChecks, ['build']);
  assert.equal(
    describeIssueDetail({...base_, checks: {...checks, verdict: 'pending'}}, path).delivery,
    'checks_pending'
  );
  assert.equal(
    describeIssueDetail({...base_, checks: {...checks, verdict: 'refused'}}, path).delivery,
    'evidence_refused'
  );
  for (const [readiness, delivery] of [
    ['blocked', 'waiting'],
    ['needs_intervention', 'needs_attention'],
    ['active', 'working'],
    ['queued', 'queued'],
    ['not_authorized', 'not_authorized'],
    ['cancelled', 'cancelled'],
  ] as const)
    assert.equal(
      describeIssueDetail(
        {...base_, readiness: {readiness, reason: 'reason', revision: 1, assessed_unix: 1, blockers: []}},
        path
      ).delivery,
      delivery
    );
});

test('detail models carry evidence, links and attributable blockers', () => {
  const model = describeIssueDetail(
    {
      ...base_,
      status: {valid: false, withdrawn: false, revision: 2, reasons: ['objectives_changed']},
      readiness: {
        readiness: 'blocked',
        reason: 'prereq_code_pending',
        revision: 1,
        assessed_unix: 1,
        blockers: [
          {
            code: 'prereq_code_pending',
            resolution: 'Wait for issue #11.',
            endpoint_repository: '7',
            endpoint_issue: '11',
          },
        ],
      },
      checks,
    },
    path
  );
  assert.equal(model.acceptanceInvalid, true);
  assert.match(model.acceptanceText, /objectives_changed/);
  assert.equal(model.blockers[0]?.endpoint, 'repository 7 issue #11');
  assert.equal(model.issueLink, '/soda-tester/demo/issues/12');
  assert.equal(model.pullLink, '/soda-tester/demo/pulls/34');
  assert.match(model.candidate, /PR #34/);
  assert.equal(nativeIssueLink('/bad/../path', '12'), undefined);
  assert.equal(nativeIssueLink(path, '0'), undefined);
  assert.equal(nativePullLink(path, 'nope'), undefined);
});

test('intervention outcomes separate confirmed, pending, uncertain and failed', () => {
  const run = 'd'.repeat(32);
  const command = 'c'.repeat(32);
  const paused = interventionResultText({
    kind: 'pause',
    receipt: {
      command_id: command,
      paused: true,
      repository: '7',
      revision: 1,
      cause: 'control_paused',
      closed_by: 'os-uid:1000',
      runs: [
        {id: run, confirmed: true, uncertain: false, outcome: 'cancelled'},
        {id: 'e'.repeat(32), confirmed: false, uncertain: true},
      ],
    },
  });
  assert.equal(paused.result, 'confirmed');
  assert.match(paused.text, /1 run stays fenced, not cancelled/);
  assert.equal(
    interventionResultText({kind: 'stop', receipt: {run_id: run, confirmed: true, uncertain: false}}).result,
    'confirmed'
  );
  assert.equal(
    interventionResultText({kind: 'stop', receipt: {run_id: run, confirmed: false, uncertain: true}}).result,
    'uncertain'
  );
  assert.equal(
    interventionResultText({kind: 'stop', receipt: {run_id: run, confirmed: false, uncertain: false}}).result,
    'pending'
  );
  assert.equal(
    interventionResultText({
      kind: 'retry',
      receipt: {prior_run: run, command_id: command, reason: 'run_failed', queued: true},
    }).result,
    'confirmed'
  );
  const stopped = interventionResultText({
    kind: 'project_stop',
    receipt: {
      project: 'p0123456789abcdef01234567',
      running: false,
      hold: true,
      hold_synced: false,
      withdrawal_revision: 2,
      runs: [],
    },
  });
  assert.equal(stopped.result, 'confirmed');
  assert.match(stopped.text, /did not sync/);
  assert.equal(interventionResultText({kind: 'pending_runs', pending: {runs: [run], message: 'x'}}).result, 'failed');
  assert.equal(
    interventionResultText({
      kind: 'command',
      record: {id: command, type: 'stop', principal: 'os-uid:1000', digest: 'f'.repeat(64)},
    }).result,
    'pending'
  );
  assert.equal(interventionResultText({kind: 'refused', status: 403, message: 'Denied.'}).result, 'failed');
  assert.equal(interventionResultText({kind: 'unconfirmed', commandID: command}).result, 'pending');
});

test('interventions pair scope, impact and pending-versus-confirmed results', () => {
  const paused = describeIntervention({
    kind: 'pause',
    enabled: true,
    result: 'pending',
    resultText: 'Pause requested; awaiting confirmation.',
    target: 'repository 7',
  });
  assert.equal(paused.label, 'Pause Factory work');
  assert.match(paused.scope, /repository 7/);
  assert.match(paused.impact, /Retained work/);
  assert.equal(paused.result, 'pending');
  const stopped = describeIntervention({
    kind: 'project_stop',
    enabled: false,
    result: 'idle',
    resultText: 'Not requested.',
    target: 'project p1',
  });
  assert.equal(stopped.enabled, false);
  assert.match(stopped.impact, /explicit confirmation/);
});

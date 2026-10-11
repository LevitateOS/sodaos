import test from 'node:test';
import assert from 'node:assert/strict';
import {
  describeDirectionRevision,
  describeOptionalEffects,
  describePrerequisites,
  describeWhyNext,
  describeWorkOverview,
  intentGeneration,
} from '../../frontend/spaces/sodaspaces-factory-progress-view';
import type {FactoryIssueDetail} from '../../frontend/spaces/sodaspaces-factory-work-response';

const path = '/soda-tester/demo';
const head = 'a'.repeat(40);
const base = 'b'.repeat(40);
const taskA = 'd' + 'a'.repeat(24);
const taskB = 'd' + 'b'.repeat(24);
const revisionId = 'd' + 'c'.repeat(24);
const intentId = 'd' + 'e'.repeat(24);

const revision = {
  id: revisionId,
  intent: intentId,
  repository: '7',
  objective: 'Let a team track demo work.',
  ideas: [
    {
      id: 'd' + 'f'.repeat(24),
      description: 'A demo dashboard.',
      consequence: 'Teams see demo state.',
      assumption: false,
      decision: 'accept' as const,
    },
  ],
  tasks: [
    {key: taskA, objective: 'Build demo list.', outcomes: ['List shows demos.'], idea_ids: []},
    {
      key: taskB,
      objective: 'Build demo detail.',
      outcomes: ['Detail shows one demo.'],
      idea_ids: [],
      depends_on: [taskA],
    },
  ],
  state: 'accepted' as const,
  max_issues: 8,
  planning_calls: 12,
};

function issueDetail(issue: string, readiness: string, extra: Record<string, unknown> = {}): FactoryIssueDetail {
  return {
    repository: '7',
    issue,
    status: {valid: true, withdrawn: false, revision: 1, reasons: []},
    readiness: {readiness, reason: 'reason', revision: 1, assessed_unix: 1720000000, blockers: []},
    ...extra,
  };
}

test('generation records bind tasks to native issues', () => {
  const parsed = intentGeneration(
    {intent_id: intentId, revision_id: revisionId, state: 'completed', task_issue_ids: {[taskA]: 11, [taskB]: 12}},
    intentId,
    revisionId
  );
  assert.equal(parsed.state, 'completed');
  assert.deepEqual(parsed.bindings, [
    {task: taskA, issue: '11'},
    {task: taskB, issue: '12'},
  ]);
  const pending = intentGeneration(
    {intent_id: intentId, revision_id: revisionId, state: 'pending'},
    intentId,
    revisionId
  );
  assert.deepEqual(pending.bindings, []);
  for (const delta of [
    {intent_id: 'd' + '0'.repeat(24)},
    {state: 'running'},
    {task_issue_ids: {nope: 11}},
    {task_issue_ids: {[taskA]: 0}},
    {task_issue_ids: {[taskA]: '11'}},
  ])
    assert.throws(() =>
      intentGeneration(
        {intent_id: intentId, revision_id: revisionId, state: 'completed', ...delta},
        intentId,
        revisionId
      )
    );
});

test('overview counts factual states and names unavailable reads', () => {
  const generation = intentGeneration(
    {intent_id: intentId, revision_id: revisionId, state: 'completed', task_issue_ids: {[taskA]: 11, [taskB]: 12}},
    intentId,
    revisionId
  );
  const model = describeWorkOverview({
    revision,
    generation,
    summary: {
      repository: '7',
      queue: {queued: 1, active: 1},
      effective: {effective: true, dispatch_open: true, missing: []},
      preparation: {project: 'p0123456789abcdef01234567', ready: true},
    },
    issues: [
      {
        ...issueDetail('11', 'completed'),
        checks: {
          verdict: 'pass',
          reason: 'checks_pass',
          resolution: 'Every required check succeeded on the exact head.',
          head_oid: head,
          base_oid: base,
          pr_number: '34',
          pr_id: '56',
          policy_revision: 1,
          native_revision: 1,
          revision: 1,
          assessed_unix: 1,
          results: [],
        },
        merge: {
          stage: 'merged',
          resolution: 'The exact candidate merged.',
          head_oid: head,
          base_oid: base,
          pr_number: '34',
          pr_id: '56',
          revision: 1,
        },
      },
      issueDetail('12', 'blocked'),
    ],
    unreadIssues: ['13'],
    repositoryPath: path,
    intentState: 'ready',
    generationState: 'ready',
    summaryState: 'ready',
  });
  assert.equal(model.direction, 'Let a team track demo work.');
  assert.equal(model.counts.delivered, 1);
  assert.equal(model.counts.waiting, 1);
  assert.match(model.countsText, /1 delivered · 1 waiting/);
  assert.equal(model.attention.length, 1);
  assert.match(model.attention[0]?.text ?? '', /could not be read/);
  assert.equal(model.dependencies.length, 2);
  assert.deepEqual(model.dependencies[1]?.dependsOn, ['issue #11']);
  assert.equal(model.dependencies[1]?.issueLink, '/soda-tester/demo/issues/12');
  assert.equal(model.setupText, 'Project setup ready.');
  assert.match(model.availabilityNote, /1 work item/);
});

test('overview distinguishes missing, waiting and unavailable', () => {
  const empty = describeWorkOverview({
    issues: [],
    unreadIssues: [],
    repositoryPath: path,
    intentState: 'missing',
    generationState: 'missing',
    summaryState: 'missing',
  });
  assert.equal(empty.direction, 'No accepted request recorded.');
  assert.equal(empty.countsText, 'No counted work.');
  assert.equal(empty.availabilityNote, '');
  const failed = describeWorkOverview({
    issues: [],
    unreadIssues: [],
    repositoryPath: 'bad path',
    intentState: 'unavailable',
    generationState: 'unavailable',
    summaryState: 'unavailable',
  });
  assert.match(failed.direction, /could not be read/);
  assert.match(failed.availabilityNote, /accepted direction.*native work mappings.*queue and setup/);
});

test('why-next needs no consultation for one eligible issue', () => {
  const single = describeWhyNext([issueDetail('11', 'queued')], path, 1720003600);
  assert.equal(single.eligible.length, 1);
  assert.match(single.selection, /only eligible work/);
  assert.match(single.eligible[0]?.ageText ?? '', /1 hour ago/);
  const none = describeWhyNext([issueDetail('11', 'blocked')], path, 1720003600);
  assert.equal(none.selection, 'No work is currently eligible to start.');
  const multi = describeWhyNext(
    [
      {
        ...issueDetail('11', 'queued'),
        readiness: {readiness: 'queued', reason: 'r', revision: 1, assessed_unix: 1720000000, blockers: []},
      },
      {
        ...issueDetail('12', 'queued'),
        readiness: {readiness: 'queued', reason: 'r', revision: 1, assessed_unix: 0, blockers: []},
      },
    ],
    path,
    1720003600
  );
  assert.match(multi.selection, /No recorded priority comparison/);
  assert.equal(multi.eligible[0]?.issue, '11');
  assert.equal(multi.eligible[1]?.ageText, 'assessment time unknown');
});

test('prerequisites name waiting reasons without inventing plans', () => {
  const rows = describePrerequisites(
    [
      {
        ...issueDetail('11', 'blocked'),
        readiness: {
          readiness: 'blocked',
          reason: 'prereq_code_pending',
          revision: 1,
          assessed_unix: 1,
          blockers: [
            {
              code: 'prereq_code_pending',
              resolution: 'Wait for issue #10.',
              endpoint_repository: '7',
              endpoint_issue: '10',
            },
          ],
        },
      },
    ],
    path
  );
  assert.match(rows[0]?.waitingText ?? '', /Wait for issue #10/);
  assert.equal(rows[0]?.planNote, 'No recorded plan for this work.');
  assert.equal(rows[0]?.link, '/soda-tester/demo/issues/11');
});

test('revision and allowance stay attributable and exact', () => {
  const model = describeDirectionRevision(revision, 'ready');
  assert.equal(model.headline, 'Let a team track demo work.');
  assert.match(model.attribution, new RegExp(revisionId));
  assert.equal(model.taskCount, 2);
  assert.match(model.allowanceText, /12 calls accepted/);
  assert.match(model.allowanceText, /Consumption record unavailable/);
  assert.equal(describeDirectionRevision(undefined, 'missing').headline, 'No accepted direction recorded.');
  assert.match(describeDirectionRevision(undefined, 'unavailable').headline, /could not be read/);
});

test('optional effects stay unavailable until their own checkpoint passes', () => {
  const models = describeOptionalEffects({issue_label: false, review_request: false, completion_note: false});
  assert.equal(models.length, 3);
  for (const model of models) assert.match(model.stateText, /has not passed/);
  const passed = describeOptionalEffects({issue_label: true, review_request: false, completion_note: false});
  assert.match(passed[0]?.stateText ?? '', /configuration is not yet exposed/);
  assert.match(passed[1]?.stateText ?? '', /has not passed/);
});

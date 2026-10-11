import test from 'node:test';
import assert from 'node:assert/strict';
import {issuePlan, requestOverview} from '../../frontend/spaces/sodaspaces-factory-progress-response';

const intent = 'd' + 'e'.repeat(24);
const revision = 'd' + 'c'.repeat(24);
const task = 'd' + 'a'.repeat(24);
const idea = 'd' + 'f'.repeat(24);
const snapshot = 'a'.repeat(64);

const overview = {
  direction: {
    objective: 'Let a team track demo work.',
    approach: 'incremental',
    ideas_by_decision: {accept: 2, pending: 1},
    ideas_by_source: {requested: 2, assumption: 1},
    effective_outcomes: [idea],
    ideas_total: 3,
    tasks_total: 2,
    unresolved_assumptions: 1,
  },
  context: {repository: '7'},
  generation: {state: 'completed'},
  setup: {state: 'ready', project: 'p0123456789abcdef01234567'},
  work_items: {
    by_state: {delivered: 1, working: 1, waiting: 0, attention: 1},
    total: 3,
    outcomes_total: 2,
    outcomes_evidenced: 0,
  },
  mappings: [
    {
      task_key: task,
      issue_id: '100',
      issue_index: '11',
      acceptance_id: idea,
      created_unix: 1720000000,
      effective: true,
    },
  ],
  attention: [
    {code: 'work_attention', issue: '12'},
    {code: 'unresolved_assumption', idea},
  ],
  intent,
  revision,
  state: 'accepted',
  observed_unix: 1720000300,
};

test('request overview binds direction, counts, mappings and attention', () => {
  const parsed = requestOverview(overview, intent);
  assert.equal(parsed.direction.tasks_total, 2);
  assert.equal(parsed.direction.unresolved_assumptions, 1);
  assert.equal(parsed.work_items.delivered, 1);
  assert.equal(parsed.work_items.outcomes_evidenced, 0);
  assert.equal(parsed.mappings[0]?.issue_index, '11');
  assert.equal(parsed.attention.length, 2);
  assert.equal(parsed.generation, 'completed');
  assert.equal(parsed.setup.state, 'ready');
  const draft = requestOverview(
    {
      ...overview,
      context: {repository: '0'},
      generation: {state: 'not_requested'},
      setup: {state: 'repository_unassigned'},
    },
    intent
  );
  assert.equal(draft.context.repository, '0');
  for (const delta of [
    {intent: 'd' + '0'.repeat(24)},
    {state: 'archived'},
    {direction: {...overview.direction, ideas_by_decision: {maybe: 1}}},
    {direction: {...overview.direction, effective_outcomes: ['short']}},
    {direction: {...overview.direction, approach: 'big_bang'}},
    {context: {repository: 'x'}},
    {generation: {state: ''}},
    {setup: {state: 'built'}},
    {work_items: {...overview.work_items, by_state: {delivered: 1}}},
    {mappings: [{...overview.mappings[0], issue_index: '0'}]},
    {attention: [{code: 'invented_code'}]},
    {attention: [{code: 'work_attention', issue: 'nope'}]},
  ])
    assert.throws(() => requestOverview({...overview, ...delta}, intent));
});

const candidate = {
  id: 'plan-1',
  repository: '7',
  issue: '12',
  readiness: 'queued',
  first_seen_unix: 1720000000,
};

test('issue plan binds freshness, plan and comparisons', () => {
  const parsed = issuePlan(
    {
      repository: '7',
      issue: '12',
      observed_unix: 1720000300,
      freshness: {state: 'current', provisional: false},
      plan: {
        candidate,
        missing: [],
        steps: ['Implement the list.', 'Cover the empty state.'],
        strategy: 'incremental_acceptance',
        snapshot,
        outcome: 'planned',
        created_unix: 1720000100,
      },
      comparisons: [
        {
          candidates: [
            {
              id: 'cand-12',
              repository: '7',
              issue: '12',
              readiness: 'queued',
              reason: 'eligible',
              first_seen_unix: 1720000000,
              assessed_unix: 1720000050,
              age_seconds: 300,
              impact_known: true,
              risk_known: false,
              effort_known: true,
              blocker_codes: [],
            },
          ],
          calls: [{selected: 'cand-12', model: 'jev-1', confidence: 0.8, input_tokens: 10, output_tokens: 5}],
          probabilities: {'cand-12': 0.8},
          snapshot,
          selected: 'cand-12',
          outcome: 'planned',
          created_unix: 1720000200,
          candidates_total: 2,
          other_count: 1,
          probability_count: 2,
          selected_visible: true,
          probabilities_full: false,
        },
      ],
    },
    '7',
    '12'
  );
  assert.equal(parsed.freshness.state, 'current');
  assert.equal(parsed.plan?.steps.length, 2);
  assert.equal(parsed.comparisons[0]?.selected, 'cand-12');
  assert.equal(parsed.comparisons[0]?.other_count, 1);
  assert.equal(parsed.comparisons[0]?.candidates[0]?.age_seconds, 300);
  assert.equal(parsed.next_before_unix, undefined);
  const empty = issuePlan(
    {
      repository: '7',
      issue: '12',
      observed_unix: 1,
      freshness: {state: 'missing', provisional: false},
      comparisons: [],
    },
    '7',
    '12'
  );
  assert.equal(empty.plan, undefined);
  assert.deepEqual(empty.comparisons, []);
  for (const delta of [
    {repository: '8'},
    {freshness: {state: 'fresh', provisional: false}},
    {freshness: {state: 'stale', provisional: true, reasons: ['invented']}},
    {plan: {candidate, missing: [], steps: [], strategy: 'x', snapshot: 'short', outcome: 'planned', created_unix: 1}},
    {
      comparisons: [
        {
          candidates: [],
          calls: [],
          snapshot,
          outcome: 'planned',
          created_unix: 1,
          candidates_total: 0,
          other_count: 0,
          probability_count: 0,
          selected_visible: false,
          probabilities_full: true,
          probabilities: {'cand-12': 1.5},
        },
      ],
    },
    {next_before_unix: 5},
    {next_before_unix: 5, next_before_snapshot: 'short'},
    {comparisons: Array.from({length: 21}, () => ({}))},
  ])
    assert.throws(() =>
      issuePlan(
        {
          repository: '7',
          issue: '12',
          observed_unix: 1,
          freshness: {state: 'missing', provisional: false},
          comparisons: [],
          ...delta,
        },
        '7',
        '12'
      )
    );
});

test('masked foreign choices stay visible without enumeration', () => {
  const parsed = issuePlan(
    {
      repository: '7',
      issue: '12',
      observed_unix: 1,
      freshness: {state: 'missing', provisional: false},
      comparisons: [
        {
          candidates: [],
          calls: [{model: 'jev-1', confidence: 0.6}],
          snapshot,
          outcome: 'planned',
          created_unix: 1,
          candidates_total: 3,
          other_count: 3,
          probability_count: 3,
          selected_visible: false,
          probabilities_full: false,
        },
      ],
    },
    '7',
    '12'
  );
  assert.equal(parsed.comparisons[0]?.selected, undefined);
  assert.equal(parsed.comparisons[0]?.selected_visible, false);
  assert.equal(parsed.comparisons[0]?.other_count, 3);
});

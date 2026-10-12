import test from 'node:test';
import assert from 'node:assert/strict';
import {
  factoryAccess,
  issuePlan,
  lifecycleEffectActions,
  lifecycleEffectConfig,
  planningAllowance,
  requestOverview,
  revisionHistory,
} from '../../frontend/spaces/sodaspaces-factory-progress-response';

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
  const relabeled = requestOverview(
    {...overview, direction: {...overview.direction, ideas_by_source: {requested: 1, unclassified: 2}}},
    intent
  );
  assert.equal(relabeled.direction.ideas_by_source.unclassified, 2);
  assert.throws(() =>
    requestOverview({...overview, direction: {...overview.direction, ideas_by_source: {legacy: 1}}}, intent)
  );
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

test('planning allowance binds bounded units and consumption', () => {
  const allowanceId = 'd' + 'b'.repeat(24);
  const parsed = planningAllowance(
    {
      id: allowanceId,
      intent,
      predecessor: 'd' + 'a'.repeat(24),
      owner: '1001',
      amount: 12,
      unit: 'planning_calls',
      reserved: 3,
      consumed: 4,
      ceiling: 20,
      effect: 'Extra comparison round.',
      created_unix: 1720000000,
    },
    intent
  );
  assert.equal(parsed.amount, 12);
  assert.equal(parsed.unit, 'planning_calls');
  assert.equal(parsed.owner, '1001');
  for (const delta of [
    {intent: 'd' + '0'.repeat(24)},
    {id: 'short'},
    {unit: 'minutes'},
    {amount: 101},
    {amount: 21, ceiling: 20},
    {reserved: 9, consumed: 4},
    {reserved: -1},
    {owner: '0'},
    {effect: '  '},
    {created_unix: 0},
  ])
    assert.throws(() =>
      planningAllowance(
        {
          id: allowanceId,
          intent,
          owner: '1001',
          amount: 12,
          unit: 'planning_calls',
          reserved: 3,
          consumed: 4,
          ceiling: 20,
          created_unix: 1,
          ...delta,
        },
        intent
      )
    );
});

test('lifecycle configs bind enablement, proof, actor and targets', () => {
  const parsed = lifecycleEffectConfig(
    {
      event: 'pr_created',
      revision: '4',
      enabled: true,
      capable: true,
      actor_id: '1001',
      repository_id: '7',
      reviewer_ids: ['11', '12'],
    },
    'pr_created',
    '7'
  );
  assert.equal(parsed.enabled, true);
  assert.deepEqual(parsed.reviewer_ids, ['11', '12']);
  const fresh = lifecycleEffectConfig(
    {event: 'issue_created', revision: '0', enabled: false, capable: false, actor_id: '', repository_id: '7'},
    'issue_created',
    '7'
  );
  assert.equal(fresh.enabled, false);
  assert.equal(fresh.capable, false);
  for (const delta of [
    {event: 'issue_deleted'},
    {revision: 'x'},
    {enabled: 'yes'},
    {repository_id: '8'},
    {reviewer_ids: ['11', '12', '13', '14']},
    {reviewer_ids: ['0']},
    {note: 'x'.repeat(1001)},
  ])
    assert.throws(() =>
      lifecycleEffectConfig(
        {
          event: 'pr_created',
          revision: '4',
          enabled: true,
          capable: true,
          actor_id: '1001',
          repository_id: '7',
          ...delta,
        },
        'pr_created',
        '7'
      )
    );
});

test('lifecycle actions bind attempts, revocation and per-target results', () => {
  const parsed = lifecycleEffectActions(
    {
      actions: [
        {
          id: 'op-1',
          event: 'verified_merge',
          action: 'original_completion_note',
          actor_id: '1001',
          issue_id: '12',
          attempts: 1,
          max_attempts: 2,
          revoked: false,
          settled: true,
          results: {'issue-12': 'delivered'},
        },
      ],
    },
    '12',
    'verified_merge'
  );
  assert.equal(parsed.length, 1);
  assert.equal(parsed[0]?.results['issue-12'], 'delivered');
  for (const action of [
    {
      id: '',
      event: 'verified_merge',
      action: 'original_completion_note',
      actor_id: '1001',
      issue_id: '12',
      attempts: 0,
      max_attempts: 2,
      revoked: false,
      settled: false,
    },
    {
      id: 'op-1',
      event: 'pr_created',
      action: 'original_completion_note',
      actor_id: '1001',
      issue_id: '12',
      attempts: 0,
      max_attempts: 2,
      revoked: false,
      settled: false,
    },
    {
      id: 'op-1',
      event: 'verified_merge',
      action: 'pr_reviewer_request',
      actor_id: '1001',
      issue_id: '12',
      attempts: 0,
      max_attempts: 2,
      revoked: false,
      settled: false,
    },
    {
      id: 'op-1',
      event: 'verified_merge',
      action: 'original_completion_note',
      actor_id: '1001',
      issue_id: '13',
      attempts: 0,
      max_attempts: 2,
      revoked: false,
      settled: false,
    },
    {
      id: 'op-1',
      event: 'verified_merge',
      action: 'original_completion_note',
      actor_id: '1001',
      issue_id: '12',
      attempts: 3,
      max_attempts: 2,
      revoked: false,
      settled: false,
    },
    {
      id: 'op-1',
      event: 'verified_merge',
      action: 'original_completion_note',
      actor_id: '1001',
      issue_id: '12',
      attempts: 0,
      max_attempts: 2,
      revoked: false,
      settled: false,
      results: {'issue-12': 'invented'},
    },
  ])
    assert.throws(() => lifecycleEffectActions({actions: [action]}, '12', 'verified_merge'));
});

test('lifecycle actions admit unclaimed rows and require a claimed actor', () => {
  const unclaimed = lifecycleEffectActions(
    {
      actions: [
        {
          id: 'op-9',
          event: 'issue_created',
          action: 'issue_label_assignee',
          actor_id: '0',
          issue_id: '12',
          attempts: 0,
          max_attempts: 2,
          revoked: false,
          settled: false,
        },
      ],
    },
    '12',
    'issue_created'
  );
  assert.equal(unclaimed.length, 1);
  assert.equal(unclaimed[0]?.actor_id, '0');
  assert.throws(() =>
    lifecycleEffectActions(
      {
        actions: [
          {
            id: 'op-9',
            event: 'issue_created',
            action: 'issue_label_assignee',
            actor_id: '0',
            issue_id: '12',
            attempts: 1,
            max_attempts: 2,
            revoked: false,
            settled: false,
          },
        ],
      },
      '12',
      'issue_created'
    )
  );
});

test('lifecycle actions bind unequal identities and a distinct native target', () => {
  const parsed = lifecycleEffectActions(
    {
      actions: [
        {
          id: 'op-2',
          event: 'pr_created',
          action: 'pr_reviewer_request',
          actor_id: '1001',
          issue_id: '12',
          target_id: '34',
          config_owner: '1002',
          target_link: '/soda/progress/pulls/3',
          attempts: 1,
          max_attempts: 2,
          revoked: false,
          settled: false,
          results: {'11': 'pending'},
        },
        {
          id: 'op-3',
          event: 'pr_created',
          action: 'pr_reviewer_request',
          actor_id: '0',
          issue_id: '12',
          target_id: '0',
          config_owner: '0',
          attempts: 0,
          max_attempts: 2,
          revoked: false,
          settled: false,
        },
      ],
    },
    '12',
    'pr_created'
  );
  assert.equal(parsed.length, 2);
  assert.equal(parsed[0]?.actor_id, '1001');
  assert.equal(parsed[0]?.config_owner, '1002');
  assert.equal(parsed[0]?.target_id, '34');
  assert.equal(parsed[0]?.target_link, '/soda/progress/pulls/3');
  assert.equal(parsed[1]?.actor_id, '0');
  assert.equal(parsed[1]?.config_owner, undefined);
  assert.equal(parsed[1]?.target_id, undefined);
  assert.equal(parsed[1]?.target_link, undefined);
  for (const bad of [{target_id: 'x'}, {config_owner: '-1'}, {target_link: 'https://forgejo.test/x'}]) {
    assert.throws(() =>
      lifecycleEffectActions(
        {
          actions: [
            {
              id: 'op-2',
              event: 'pr_created',
              action: 'pr_reviewer_request',
              actor_id: '1001',
              issue_id: '12',
              attempts: 1,
              max_attempts: 2,
              revoked: false,
              settled: false,
              ...bad,
            },
          ],
        },
        '12',
        'pr_created'
      )
    );
  }
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

const accessEntry = {
  intent,
  id: 'd' + 'b'.repeat(24),
  unit: 'planning_calls',
  status: 'ok',
  amount: 40,
  reserved: 5,
  consumed: 12,
  available: 23,
  ceiling: 100,
  editable: {editable: true},
  effect: 'cover review queue',
};

const accessFixture = {
  repository: '7',
  observed_unix: 1728670000,
  viewer: {
    code_write: true,
    owner_or_admin: true,
    environment_owner: false,
    operator: false,
    member: true,
    session_current: true,
  },
  effective: {effective: true, dispatch_open: true, missing: []},
  required_actions: [],
  planning: [accessEntry],
};

test('repository access binds standing, verdict and bounded planning listing', () => {
  const parsed = factoryAccess(accessFixture, '7');
  assert.equal(parsed.repository, '7');
  assert.equal(parsed.viewer.owner_or_admin, true);
  assert.equal(parsed.viewer.session_current, true);
  assert.equal(parsed.effective.effective, true);
  assert.equal(parsed.effective.dispatch_open, true);
  assert.equal(parsed.planning.length, 1);
  assert.equal(parsed.planning[0]?.available, 23);
  assert.equal(parsed.planning[0]?.effect, 'cover review queue');
  assert.equal(parsed.planning_truncated, false);
  const shown = factoryAccess({...accessFixture, planning_truncated: true}, '7');
  assert.equal(shown.planning_truncated, true);
});

test('repository access rejects wrong scope, overlong listings and inconsistent entries', () => {
  assert.throws(() => factoryAccess(accessFixture, '8'));
  assert.throws(() => factoryAccess({...accessFixture, repository: '8'}, '7'));
  const many = Array.from({length: 21}, () => accessEntry);
  assert.throws(() => factoryAccess({...accessFixture, planning: many}, '7'));
  assert.throws(() => factoryAccess({...accessFixture, planning: [{...accessEntry, available: 24}]}, '7'));
  assert.throws(() => factoryAccess({...accessFixture, planning: [{...accessEntry, consumed: 35, available: 0}]}, '7'));
  assert.throws(() => factoryAccess({...accessFixture, viewer: {...accessFixture.viewer, member: 'yes'}}, '7'));
});

const revisionPrior = 'd' + '1'.repeat(24);
const taskB = 'd' + 'b'.repeat(24);

const historyCurrent = {
  revision,
  state: 'accepted',
  repository: '7',
  created_unix: 1728670000,
  generation: {state: 'committed', submitted_keys: [task, taskB]},
  comparison: {
    tasks: {
      added: [],
      removed: [],
      changed: [
        {
          key: task,
          changed_fields: ['objective'],
          old_objective: 'Track work.',
          new_objective: 'Track demo work.',
        },
      ],
      reused: [taskB],
      unchanged_count: 1,
    },
    ideas: {
      added: [],
      removed: [],
      changed: [
        {
          id: idea,
          from_decision: 'pending',
          to_decision: 'accept',
          correction_changed: false,
          content_changed: true,
        },
      ],
    },
    effective_outcomes: {added: [idea], removed: []},
    state_change: {from: 'draft', to: 'accepted'},
    edges_changed: false,
  },
  work: [{task_key: task, issue_index: '12', state: 'working', authority: 'current', content_version: 2}],
};

const historyFirst = {
  revision: revisionPrior,
  state: 'draft',
  repository: '7',
  created_unix: 1728660000,
  generation: {state: 'not_requested'},
  comparison: {
    tasks: {added: [], removed: [], changed: [], reused: [], unchanged_count: 0},
    ideas: {added: [], removed: [], changed: []},
    effective_outcomes: {added: [], removed: []},
    initial: true,
    edges_changed: false,
  },
  work: [],
};

const historyFixture = {intent, observed_unix: 1728670000, revisions: [historyCurrent, historyFirst]};

test('revision history binds before/after and affected work to its intent', () => {
  const parsed = revisionHistory(historyFixture, intent);
  assert.equal(parsed.intent, intent);
  assert.equal(parsed.revisions.length, 2);
  assert.equal(parsed.truncated, false);
  const current = parsed.revisions[0];
  assert.equal(current?.state, 'accepted');
  assert.equal(current?.generation.state, 'committed');
  assert.deepEqual(current?.generation.submitted_keys, [task, taskB]);
  assert.equal(current?.generation.submitted_differs, false);
  assert.equal(current?.comparison.tasks.changed[0]?.old_objective, 'Track work.');
  assert.equal(current?.comparison.ideas.changed[0]?.to_decision, 'accept');
  assert.deepEqual(current?.comparison.effective_added, [idea]);
  assert.equal(current?.comparison.state_change?.to, 'accepted');
  assert.deepEqual(current?.comparison.tasks.reused, [taskB]);
  assert.equal(current?.comparison.edges_changed, false);
  assert.equal(current?.work[0]?.issue_index, '12');
  assert.equal(current?.work[0]?.content_version, 2);
  assert.equal(parsed.revisions[1]?.comparison.initial, true);
  const cut = revisionHistory({...historyFixture, truncated: true}, intent);
  assert.equal(cut.truncated, true);
});

test('revision history rejects wrong scope, bad pages and unpaired texts', () => {
  assert.throws(() => revisionHistory(historyFixture, task));
  assert.throws(() => revisionHistory({...historyFixture, revisions: []}, intent));
  const many = Array.from({length: 6}, () => historyFirst);
  assert.throws(() => revisionHistory({...historyFixture, revisions: many}, intent));
  const dirtyFirst = {
    ...historyFirst,
    comparison: {...historyFirst.comparison, effective_outcomes: {added: [idea], removed: []}},
  };
  assert.throws(() => revisionHistory({...historyFixture, revisions: [dirtyFirst]}, intent));
  const strayText = {
    ...historyCurrent,
    comparison: {
      ...historyCurrent.comparison,
      tasks: {
        added: [],
        removed: [],
        changed: [{key: task, changed_fields: ['depends_on'], old_objective: 'Track work.'}],
        unchanged_count: 1,
      },
    },
  };
  assert.throws(() => revisionHistory({...historyFixture, revisions: [strayText]}, intent));
  const badWork = {
    ...historyCurrent,
    work: [{task_key: task, issue_index: '12', state: 'shipped', authority: 'current'}],
  };
  assert.throws(() => revisionHistory({...historyFixture, revisions: [badWork]}, intent));
  const badGeneration = {...historyCurrent, generation: {state: 'dreamed'}};
  assert.throws(() => revisionHistory({...historyFixture, revisions: [badGeneration]}, intent));
  const badCount = {
    ...historyCurrent,
    comparison: {...historyCurrent.comparison, tasks: {...historyCurrent.comparison.tasks, unchanged_count: 2}},
  };
  assert.throws(() => revisionHistory({...historyFixture, revisions: [badCount]}, intent));
  const edgedFirst = {...historyFirst, comparison: {...historyFirst.comparison, edges_changed: true}};
  assert.throws(() => revisionHistory({...historyFixture, revisions: [edgedFirst]}, intent));
  const keyedAbsent = {...historyFirst, generation: {state: 'not_requested', submitted_keys: [task]}};
  assert.throws(() => revisionHistory({...historyFixture, revisions: [keyedAbsent]}, intent));
  const differsAbsent = {...historyFirst, generation: {state: 'not_requested', submitted_differs: true}};
  assert.throws(() => revisionHistory({...historyFixture, revisions: [differsAbsent]}, intent));
  const emptyRecord = {...historyCurrent, generation: {state: 'committed'}};
  assert.throws(() => revisionHistory({...historyFixture, revisions: [emptyRecord]}, intent));
  const bareDiffers = {
    ...historyCurrent,
    generation: {state: 'committed', submitted_keys: [task], submitted_differs: true},
  };
  assert.equal(
    revisionHistory({...historyFixture, revisions: [bareDiffers]}, intent).revisions[0]?.generation.submitted_differs,
    true
  );
});

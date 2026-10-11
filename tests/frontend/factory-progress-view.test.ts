import test from 'node:test';
import assert from 'node:assert/strict';
import {
  describeAccessStanding,
  describeAllowance,
  describeDirectionRevision,
  describeLifecycleEffect,
  describePlanningReference,
  describePrerequisites,
  describeRecordedChoice,
  describeRecordedPlan,
  describeRequestOverview,
  describeRevisionHistory,
  describeWhyNext,
  describeWorkOverview,
  intentGeneration,
} from '../../frontend/spaces/sodaspaces-factory-progress-view';
import {
  factoryAccess,
  issuePlan,
  planningAllowance,
  requestOverview,
  revisionHistory,
} from '../../frontend/spaces/sodaspaces-factory-progress-response';
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
    closure: {state: 'none'},
    capabilities: {
      watch: {available: true},
      stop: {available: true},
      retry: {available: true},
      takeover: {available: true},
      pause: {available: true},
      resume: {available: true},
    },
    observed_unix: 1720000300,
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
          stale: false,
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
          stale: false,
        },
        closure: {state: 'confirmed', occurrence: '99', closed_unix: 1720000200},
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
  assert.match(model.allowanceText, /12/);
  assert.match(model.allowanceText, /Live allowance reads separately/);
  assert.equal(describeDirectionRevision(undefined, 'missing').headline, 'No accepted direction recorded.');
  assert.match(describeDirectionRevision(undefined, 'unavailable').headline, /could not be read/);
});

test('allowance shows bounded units, consumption and attribution', () => {
  const allowance = planningAllowance(
    {
      id: 'd' + 'b'.repeat(24),
      intent: intentId,
      predecessor: 'd' + 'a'.repeat(24),
      owner: '1001',
      amount: 12,
      unit: 'planning_calls',
      reserved: 3,
      consumed: 4,
      ceiling: 20,
      effect: 'Extra comparison round.',
      created_unix: 1,
    },
    intentId
  );
  const model = describeAllowance(allowance, 'ready');
  assert.match(model.summaryText, /12 planning calls/);
  assert.match(model.summaryText, /3 reserved, 4 consumed, 5 remaining/);
  assert.match(model.attributionText, /owner 1001/);
  assert.match(model.attributionText, /adjusts/);
  assert.match(model.effectText, /Extra comparison round/);
  assert.equal(describeAllowance(undefined, 'missing').summaryText, 'No planning allowance recorded.');
  assert.match(describeAllowance(undefined, 'unavailable').summaryText, /could not be read/);
});

test('request overview projects evidenced counts and named attention', () => {
  const view = requestOverview(
    {
      direction: {
        objective: 'Let a team track demo work.',
        ideas_by_decision: {accept: 2},
        ideas_by_source: {requested: 2},
        effective_outcomes: [],
        ideas_total: 2,
        tasks_total: 2,
        unresolved_assumptions: 0,
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
          task_key: taskA,
          issue_id: '100',
          issue_index: '11',
          acceptance_id: revisionId,
          created_unix: 1,
          effective: true,
        },
        {
          task_key: taskB,
          issue_id: '101',
          issue_index: '12',
          acceptance_id: revisionId,
          created_unix: 1,
          effective: false,
        },
      ],
      attention: [{code: 'work_attention', issue: '12'}, {code: 'generation_refused'}],
      intent: intentId,
      revision: revisionId,
      state: 'accepted',
      observed_unix: 1720000300,
    },
    intentId
  );
  const model = describeRequestOverview(view, path);
  assert.equal(model.direction, 'Let a team track demo work.');
  assert.match(model.countsText, /1 delivered · 1 working · 1 needs your decision/);
  assert.equal(model.attention.length, 2);
  assert.equal(model.attention[0]?.link, '/soda-tester/demo/issues/12');
  assert.equal(model.mappings[1]?.effective, false);
  assert.equal(model.setupText, 'Project setup ready.');
  assert.match(model.outcomeNote, /0 of 2 accepted product outcomes evidenced/);
  assert.match(model.observedText, /Observed/);
  const draft = describeRequestOverview(requestOverview({...overviewDraft(), intent: intentId}, intentId), path);
  assert.match(draft.direction, /Request is draft/);
  assert.equal(draft.setupText, 'No repository assigned to this request.');
});

function overviewDraft() {
  return {
    direction: {
      objective: 'Draft idea.',
      ideas_by_decision: {},
      ideas_by_source: {},
      effective_outcomes: [],
      ideas_total: 0,
      tasks_total: 0,
      unresolved_assumptions: 0,
    },
    context: {repository: '0'},
    generation: {state: 'not_requested'},
    setup: {state: 'repository_unassigned'},
    work_items: {
      by_state: {delivered: 0, working: 0, waiting: 0, attention: 0},
      total: 0,
      outcomes_total: 0,
      outcomes_evidenced: 0,
    },
    mappings: [],
    attention: [],
    intent: intentId,
    revision: revisionId,
    state: 'draft',
    observed_unix: 0,
  };
}

test('recorded plans name freshness, strategy and missing prerequisites', () => {
  const plan = issuePlan(
    {
      repository: '7',
      issue: '12',
      observed_unix: 1,
      freshness: {state: 'stale', provisional: true, reasons: ['acceptance_changed']},
      plan: {
        candidate: {id: 'plan-1', repository: '7', issue: '12', readiness: 'queued', first_seen_unix: 1},
        missing: [{code: 'prereq_code_pending', resolution: 'Wait for issue #11.'}],
        steps: ['Implement the list.'],
        strategy: 'incremental_acceptance',
        snapshot: 'a'.repeat(64),
        outcome: 'planned',
        created_unix: 1,
      },
      comparisons: [],
    },
    '7',
    '12'
  );
  const model = describeRecordedPlan(plan.plan, plan.freshness);
  assert.match(model.freshnessText, /stale: acceptance changed/i);
  assert.match(model.freshnessText, /Recorded as provisional/);
  assert.match(model.strategyText, /incremental_acceptance/);
  assert.match(model.missingText, /Wait for issue #11/);
  assert.deepEqual(model.steps, ['Implement the list.']);
  assert.equal(
    describeRecordedPlan(undefined, {state: 'missing', provisional: false, reasons: []}).freshnessText,
    'No recorded plan.'
  );
  assert.equal(
    describeRecordedPlan(undefined, {state: 'provisional', provisional: true, reasons: []}).freshnessText,
    'Provisional plan: conveys no permission to execute.'
  );
});

test('recorded choices explain selection, fairness and masked winners', () => {
  const comparison = {
    candidates: [
      {
        id: 'cand-12',
        repository: '7',
        issue: '12',
        readiness: 'queued',
        first_seen_unix: 1720000000,
        assessed_unix: 1720000050,
        age_seconds: 3700,
        impact_known: true,
        risk_known: false,
        effort_known: true,
      },
    ],
    calls: [{selected: 'cand-12', model: 'jev-1', confidence: 0.8}],
    probabilities: {'cand-12': 0.8},
    snapshot: 'a'.repeat(64),
    selected: 'cand-12',
    outcome: 'planned',
    created_unix: 1720000200,
    candidates_total: 1,
    other_count: 0,
    probability_count: 1,
    selected_visible: true,
    probabilities_full: true,
  };
  const plan = issuePlan(
    {
      repository: '7',
      issue: '12',
      observed_unix: 1,
      freshness: {state: 'current', provisional: false},
      comparisons: [comparison],
    },
    '7',
    '12'
  );
  const model = describeRecordedChoice(plan, path);
  assert.match(model.selection, /Recorded choice: issue #12/);
  assert.match(model.selection, /0\.8/);
  assert.match(model.judgmentText, /jev-1/);
  assert.match(model.candidates[0]?.ageText ?? '', /1 hour/);
  assert.match(model.candidates[0]?.estimatesText ?? '', /impact, effort recorded/);
  const masked = describeRecordedChoice(
    issuePlan(
      {
        repository: '7',
        issue: '12',
        observed_unix: 1,
        freshness: {state: 'missing', provisional: false},
        comparisons: [{...comparison, candidates: [], selected_visible: false, other_count: 2}],
      },
      '7',
      '12'
    ),
    path
  );
  assert.equal(masked.selection, "Another repository's work was selected.");
  const none = describeRecordedChoice(
    issuePlan(
      {
        repository: '7',
        issue: '12',
        observed_unix: 1,
        freshness: {state: 'missing', provisional: false},
        comparisons: [],
      },
      '7',
      '12'
    ),
    path
  );
  assert.match(none.selection, /No recorded priority comparison/);
});

test('lifecycle effects gate on proof and name actor, targets and results', () => {
  const config = {
    event: 'pr_created',
    revision: '4',
    enabled: true,
    capable: true,
    actor_id: '1001',
    repository_id: '7',
    reviewer_ids: ['11', '12'],
  };
  const actions = [
    {
      id: 'op-1',
      event: 'pr_created',
      action: 'pr_reviewer_request',
      actor_id: '1001',
      issue_id: '12',
      attempts: 1,
      max_attempts: 2,
      revoked: false,
      settled: false,
      results: {'11': 'delivered', '12': 'pending'},
    },
  ];
  const model = describeLifecycleEffect('pr_created', config, 'ready', actions);
  assert.equal(model.stateText, 'Active.');
  assert.match(model.configText, /Actor 1001/);
  assert.match(model.configText, /Reviewers 11, 12/);
  assert.match(model.resultsText[0] ?? '', /1\/2 attempts/);
  assert.match(model.resultsText[0] ?? '', /11: delivered; 12: pending/);
  assert.equal(model.canConfigure, true);
  assert.equal(describeLifecycleEffect('pr_created', {...config, capable: false}, 'ready', []).canConfigure, false);
  assert.match(
    describeLifecycleEffect('pr_created', {...config, capable: false}, 'ready', []).stateText,
    /checkpoint has not passed/
  );
  assert.equal(describeLifecycleEffect('pr_created', {...config, enabled: false}, 'ready', []).stateText, 'Disabled.');
  assert.equal(
    describeLifecycleEffect('pr_created', {...config, revision: '0'}, 'ready', []).stateText,
    'Not configured yet.'
  );
  assert.equal(describeLifecycleEffect('pr_created', undefined, 'unavailable', []).canConfigure, false);
});

test('each optional effect gates on its own native checkpoint', () => {
  const effect = {
    event: 'pr_created',
    revision: '4',
    enabled: true,
    capable: true,
    actor_id: '1001',
    repository_id: '7',
    reviewer_ids: ['11', '12'],
  };
  assert.equal(describeLifecycleEffect('pr_created', {...effect, capable: true}, 'ready', []).canConfigure, true);
  assert.equal(describeLifecycleEffect('issue_created', {...effect, capable: false}, 'ready', []).canConfigure, false);
  assert.equal(describeLifecycleEffect('verified_merge', {...effect, capable: false}, 'ready', []).canConfigure, false);
  assert.match(
    describeLifecycleEffect('issue_created', {...effect, capable: false}, 'ready', []).stateText,
    /checkpoint has not passed/
  );
});

const accessViewer = {
  code_write: true,
  owner_or_admin: true,
  environment_owner: false,
  operator: false,
  member: true,
  session_current: true,
};

function accessWith(planning: unknown[], overrides: Record<string, unknown> = {}) {
  return factoryAccess(
    {
      repository: '7',
      observed_unix: 1728670000,
      viewer: accessViewer,
      effective: {effective: true, dispatch_open: true, missing: []},
      required_actions: [],
      planning,
      ...overrides,
    },
    '7'
  );
}

const planningEntry = {
  intent: intentId,
  id: 'd' + 'b'.repeat(24),
  unit: 'planning_calls',
  status: 'ok',
  amount: 40,
  reserved: 5,
  consumed: 12,
  available: 23,
  ceiling: 100,
  editable: {editable: true},
};

test('planning reference finds this request or states the listing gap', () => {
  const access = accessWith([planningEntry]);
  const found = describePlanningReference(access, 'ready', intentId);
  assert.equal(found.found, true);
  assert.match(found.referenceText, /23 available/);
  assert.match(found.referenceText, /You may adjust it/);
  const missing = describePlanningReference(access, 'ready', 'd' + 'f'.repeat(24));
  assert.equal(missing.found, false);
  assert.match(missing.referenceText, /No bound allowance/);
  const truncated = describePlanningReference(
    accessWith([planningEntry], {planning_truncated: true}),
    'ready',
    'd' + 'f'.repeat(24)
  );
  assert.match(truncated.referenceText, /first 20 allowances shown/);
  assert.match(truncated.truncatedNote, /first 20 allowances/);
  assert.match(describePlanningReference(undefined, 'unavailable', intentId).referenceText, /could not be read/);
});

test('access standing states powers, verdict and unassigned resolvers', () => {
  const open = describeAccessStanding(accessWith([planningEntry]), 'ready');
  assert.match(open.powersText, /owner or admin/);
  assert.match(open.powersText, /Session current/);
  assert.match(open.verdictText, /effective and open/);
  assert.equal(open.actionsText.length, 0);
  const blocked = describeAccessStanding(
    accessWith([], {
      effective: {effective: false, dispatch_open: false, missing: ['policy_missing']},
      required_actions: [
        {code: 'policy_missing', resolve_by: 'owner_or_admin', setting: 'policy'},
        {code: 'mystery_code', resolve_by: '', setting: ''},
      ],
    }),
    'ready'
  );
  assert.match(blocked.verdictText, /not effective: policy_missing/);
  assert.match(blocked.actionsText[0] ?? '', /owner_or_admin via policy/);
  assert.match(blocked.actionsText[1] ?? '', /unassigned/);
  assert.match(describeAccessStanding(undefined, 'missing').powersText, /No permission record/);
});

const historyIdea = 'd' + '2'.repeat(24);

function historyWith(entries: unknown[], overrides: Record<string, unknown> = {}) {
  return revisionHistory({intent: intentId, observed_unix: 1728670000, revisions: entries, ...overrides}, intentId);
}

const historyNow = {
  revision: revisionId,
  state: 'accepted',
  repository: '7',
  created_unix: 1728670000,
  generation: {state: 'committed'},
  comparison: {
    tasks: {
      added: [{key: taskB, objective: 'Ship demo.', outcomes: ['demo scope'], idea_ids: [historyIdea]}],
      removed: [],
      changed: [
        {
          key: taskA,
          changed_fields: ['objective'],
          old_objective: 'Track work.',
          new_objective: 'Track demo work.',
        },
      ],
      unchanged_count: 1,
    },
    ideas: {
      added: [],
      removed: [],
      changed: [
        {
          id: historyIdea,
          from_decision: 'pending',
          to_decision: 'accept',
          correction_changed: false,
          content_changed: false,
        },
      ],
    },
    effective_outcomes: {added: [historyIdea], removed: []},
    state_change: {from: 'draft', to: 'accepted'},
  },
  work: [{task_key: taskA, issue_index: '12', state: 'working', authority: 'current'}],
};

const historyOld = {
  revision: 'd' + '3'.repeat(24),
  state: 'draft',
  repository: '7',
  created_unix: 1728660000,
  generation: {state: 'not_requested'},
  comparison: {
    tasks: {added: [], removed: [], changed: [], unchanged_count: 0},
    ideas: {added: [], removed: [], changed: []},
    effective_outcomes: {added: [], removed: []},
    initial: true,
  },
  work: [],
};

test('revision history composes before/after and affected work per revision', () => {
  const model = describeRevisionHistory(historyWith([historyNow, historyOld]), 'ready');
  assert.match(model.stateText, /2 revisions shown, newest first/);
  assert.equal(model.truncatedNote, '');
  const current = model.entries[0];
  assert.match(current?.headerText ?? '', /accepted/);
  assert.match(current?.tasksText ?? '', /1 added, 0 removed, 1 changed, 1 unchanged/);
  assert.match(current?.taskLines[0] ?? '', new RegExp(`\\+ ${taskB}: Ship demo\\.`));
  assert.match(current?.taskLines[1] ?? '', /objective: Track work\. → Track demo work\./);
  assert.match(current?.ideaLines[0] ?? '', /pending → accept \(decision only\)/);
  assert.match(current?.scopeLines[0] ?? '', new RegExp(`Effective outcomes added: ${historyIdea}`));
  assert.match(current?.scopeLines[1] ?? '', /State draft → accepted/);
  assert.match(current?.workLines[0] ?? '', /working \(current\)/);
  assert.match(model.entries[1]?.lineageText ?? '', /First revision/);
  const cut = describeRevisionHistory(historyWith([historyNow], {truncated: true}), 'ready');
  assert.match(cut.truncatedNote, /beyond this page/);
  assert.match(describeRevisionHistory(undefined, 'unavailable').stateText, /could not be read/);
  assert.match(describeRevisionHistory(undefined, 'missing').stateText, /No revision history/);
});

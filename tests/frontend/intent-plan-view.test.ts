import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {reviewIntent} from '../../frontend/spaces/sodaspaces-intent-view';
import type {IntentView} from '../../frontend/spaces/sodaspaces-intent-view';
import type {IntentPlanStatus, IntentRevision} from '../../frontend/spaces/sodaspaces-intent-response';

function textOf(value: unknown): string {
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  if (Array.isArray(value)) return value.map(textOf).join('');
  if (value && typeof value === 'object' && 'strings' in value && 'values' in value) {
    const template = value as {strings: ArrayLike<string>; values: unknown[]};
    let out = '';
    for (let i = 0; i < template.strings.length; i++) {
      out += template.strings[i];
      if (i < template.values.length) out += textOf(template.values[i]);
    }
    return out;
  }
  return '';
}

const revision: IntentRevision = {
  id: 'd888888888888888888888888',
  intent: 'd777777777777777777777777',
  repository: '7',
  objective: 'Decide vacation requests with manager review',
  ideas: [],
  tasks: [],
  state: 'draft',
  max_issues: 10,
  planning_calls: 3,
};

function viewWith(plan: IntentPlanStatus | undefined, planNotice = '', operation = 'propose_intent'): IntentView {
  const noop = () => {};
  return {
    objective: revision.objective,
    saveNote: '',
    exploration: undefined,
    explorationStale: false,
    changeOpen: false,
    changeText: '',
    generation: undefined,
    generationError: '',
    unsavedCount: 0,
    unsavedParts: '',
    savedRequests: [],
    savedLoaded: true,
    savedError: '',
    savedMore: false,
    department: '',
    departments: [],
    revision,
    busy: false,
    blocked: false,
    dirty: false,
    error: '',
    departmentDialog: false,
    departmentName: '',
    departmentDescription: '',
    maxIssues: 10,
    planningCalls: 3,
    allowance: undefined,
    raiseCalls: 3,
    fundingNote: '',
    planOperation: operation,
    plan,
    planNotice,
    set: noop,
    setBound: noop,
    setRaise: noop,
    raiseAllowance: noop,
    decide: noop,
    toggleChange: noop,
    correctProposal: noop,
    refreshGeneration: noop,
    chooseApproach: noop,
    describe: noop,
    saveDraft: noop,
    saveAndLeave: noop,
    loadSaved: noop,
    moreSaved: noop,
    resumeRequest: noop,
    revise: noop,
    decline: noop,
    continue: noop,
    dialog: noop,
    dialogKey: noop,
    createDepartment: noop,
    setPlanOperation: noop,
    planWork: noop,
    refreshPlanStatus: noop,
  };
}

test('draft review renders the plan action with operation choices', () => {
  const text = textOf(reviewIntent(viewWith(undefined)));
  assert.ok(text.includes('Plan work'));
  assert.ok(text.includes('Refresh status'));
  assert.ok(text.includes('Review baseline'));
  assert.ok(text.includes('Candidate issue graphs'));
  assert.ok(!text.includes('Planning is running'));
});

test('draft review renders pending and reported plan results', () => {
  const base = {revision: revision.id, candidates: 0};
  assert.ok(textOf(reviewIntent(viewWith({...base, state: 'pending'}))).includes('Planning is running'));
  const completed = textOf(
    reviewIntent(viewWith({...base, state: 'completed', summary: 'two views ready', candidates: 2}))
  );
  assert.ok(completed.includes('Planning proposed 2 candidates'));
  const attention = textOf(
    reviewIntent(viewWith({...base, state: 'attention', summary: 'waiting', attention: 'Decide the queue.'}))
  );
  assert.ok(attention.includes('Planning needs a product decision: Decide the queue.'));
  const failed = textOf(reviewIntent(viewWith({...base, state: 'failed', summary: 'crashed'})));
  assert.ok(failed.includes('Planning failed'));
});

test('draft review renders the plan notice', () => {
  const text = textOf(reviewIntent(viewWith(undefined, 'Planning is running. The status below follows the run.')));
  assert.ok(text.includes('Planning is running. The status below follows the run.'));
});

test('repository-less draft renders no plan action', () => {
  const unbound = viewWith(undefined);
  unbound.revision = {...revision, repository: '0'};
  assert.ok(!textOf(reviewIntent(unbound)).includes('Plan work'));
});

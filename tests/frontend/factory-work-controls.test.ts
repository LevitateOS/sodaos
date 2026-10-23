import test from 'node:test';
import assert from 'node:assert/strict';
import {before, after, type TestContext} from 'node:test';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';

// Real-engine coverage for the Factory work-progress elements
// (V01–V07): each view mounts with stub reads, renders stub data,
// and every control posts its exact request and reports the
// confirmed outcome. One uninterrupted headless pass.

const INTENT = 'd' + 'e'.repeat(24);
const REVISION = 'd' + 'c'.repeat(24);
const TASK_A = 'd' + 'a'.repeat(24);
const TASK_B = 'd' + 'b'.repeat(24);
const IDEA = 'd' + 'f'.repeat(24);
const RUN = 'd'.repeat(32);
const PROJECT = 'p0123456789abcdef01234567';
const ENVIRONMENT = 'p' + 'f'.repeat(24);
const ALLOWANCE = 'd' + 'b'.repeat(24);
const ALLOWANCE_NEXT = 'd' + '1'.repeat(24);
const HEAD = 'a'.repeat(40);
const BASE = 'b'.repeat(40);
const SNAPSHOT = 'a'.repeat(64);

interface StubRoute {
  readonly method: string;
  readonly path: string;
  readonly action?: string;
  readonly status: number;
  readonly json: unknown;
  readonly echoCommand?: boolean;
  readonly echoAcceptance?: boolean;
  readonly loseReply?: boolean;
  readonly receiptPending?: boolean;
}
interface WorkCall {
  readonly method: string;
  readonly path: string;
  readonly body: unknown;
}

const root = path.resolve(import.meta.dirname, '../..');
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;

before(async () => {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-work-fixture.ts')],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Work fixture build failed: ${build.logs.join('\n')}`);
  const [fixture] = build.outputs;
  assert(fixture);
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(req) {
      const url = new URL(req.url);
      if (url.pathname === '/assets/factory-work-fixture.js')
        return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><main></main><script type="module" src="/assets/factory-work-fixture.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {
  await browser?.close();
  server?.stop(true);
});

type WorkView = 'overview' | 'detail' | 'plan' | 'direction' | 'effects';

async function openView(t: TestContext, view: WorkView, routes: StubRoute[]): Promise<Page> {
  const page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.waitForFunction(
    () => typeof window.factoryWork !== 'undefined' && typeof window.factoryWork.mount === 'function'
  );
  await page.evaluate(
    ([vista, table]) => {
      window.factoryWork.setRoutes(table);
      window.factoryWork.mount(vista);
    },
    [view, routes] as const
  );
  return page;
}

async function workCalls(page: Page): Promise<WorkCall[]> {
  return page.evaluate(() => window.factoryWork.calls());
}

const intentRevision = {
  id: REVISION,
  intent: INTENT,
  repository: '7',
  objective: 'Let a team track demo work.',
  state: 'accepted',
  max_issues: 8,
  planning_calls: 12,
  ideas: [
    {
      id: IDEA,
      description: 'A demo dashboard.',
      consequence: 'Teams see demo state.',
      assumption: false,
      decision: 'accept',
    },
  ],
  tasks: [
    {key: TASK_A, objective: 'Build demo list.', outcomes: ['List shows demos.'], idea_ids: []},
    {
      key: TASK_B,
      objective: 'Build demo detail.',
      outcomes: ['Detail shows one demo.'],
      idea_ids: [],
      depends_on: [TASK_A],
    },
  ],
};

const allAvailable = {
  watch: {available: true},
  stop: {available: true},
  retry: {available: true},
  takeover: {available: true},
  pause: {available: true},
  resume: {available: true},
};

const issueDetail = {
  repository: '7',
  issue: '12',
  status: {valid: true, revision: 4, reasons: []},
  closure: {state: 'none'},
  capabilities: allAvailable,
  observed_unix: 1720000300,
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
  attempt: {id: 'e'.repeat(32), role: 'coder', stage: 'assigned', run: RUN, attempts: 1, created_unix: 1720000000},
  checks: {
    verdict: 'pending',
    reason: 'checks_pending',
    resolution: 'Wait for the pending checks to report.',
    head_oid: HEAD,
    base_oid: BASE,
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
    head_oid: HEAD,
    base_oid: BASE,
    pr_number: '34',
    pr_id: '56',
    revision: 1,
  },
  links: {issue_url: '/soda-tester/demo/issues/12', pull_url: '/soda-tester/demo/pulls/34'},
};

const attentionCompletion = {
  completion_id: 'c'.repeat(32),
  cycle_id: 'b'.repeat(32),
  state: 'attention',
  repository: '7',
  generation: 1,
  retryable: true,
  wait_reason: 'decision_not_actionable',
};
const attentionIssueDetail = {
  ...issueDetail,
  readiness: {readiness: 'queued', reason: 'eligible', revision: 9, assessed_unix: 1720000000},
  journey: {
    repository: '7',
    issue: '12',
    observed_unix: 1720000100,
    phases: [],
    tasks: [
      {key: 'task_a', lane: 'a', state: 'complete'},
      {key: 'task_b', lane: 'b', state: 'complete'},
    ],
    commits: [{commit: HEAD, base: BASE, pr_number: '34', pr_id: '56'}],
    findings: [],
    corrections: [],
    judgments: [
      {
        kind: 'reassessment-decision',
        selected: 'continue',
        model: 'test-prior-low-confidence',
        cycle_id: 'a'.repeat(32),
        cycle_state: 'superseded',
        confidence: 0.28,
        probability: 0.9,
        wait: false,
      },
      {
        kind: 'reassessment-decision',
        selected: 'continue',
        model: 'test-low-confidence',
        cycle_id: attentionCompletion.cycle_id,
        cycle_state: 'attention',
        confidence: 0.24,
        probability: 0.9,
        wait: true,
        wait_reason: 'decision_not_actionable',
      },
    ],
    gates: [
      {...attentionCompletion, cycle_id: 'a'.repeat(32), state: 'superseded', retryable: false, wait_reason: ''},
      attentionCompletion,
    ],
    adoptions: [],
  },
};

const allowance = {
  id: ALLOWANCE,
  intent: INTENT,
  owner: '1001',
  amount: 40,
  ceiling: 100,
  reserved: 5,
  consumed: 12,
  unit: 'planning_calls',
  created_unix: 1720000000,
};

const access = {
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
  planning: [
    {
      intent: INTENT,
      id: ALLOWANCE,
      unit: 'planning_calls',
      status: 'ok',
      amount: 40,
      reserved: 5,
      consumed: 12,
      available: 23,
      ceiling: 100,
      editable: {editable: true},
    },
  ],
};

const historyEntry = {
  revision: REVISION,
  state: 'accepted',
  repository: '7',
  created_unix: 1728670000,
  generation: {state: 'committed', submitted_keys: [TASK_A, TASK_B]},
  comparison: {
    tasks: {added: [], removed: [], changed: [], reused: [TASK_A, TASK_B], unchanged_count: 2},
    ideas: {added: [], removed: [], changed: []},
    effective_outcomes: {added: [], removed: []},
    state_change: {from: 'draft', to: 'accepted'},
    edges_changed: false,
  },
  work: [{task_key: TASK_A, issue_index: '12', state: 'working', authority: 'current', content_version: 2}],
};

const historyFirst = {
  revision: 'd' + '3'.repeat(24),
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

const requestOverviewRoute = {
  direction: {
    objective: 'Let a team track demo work.',
    approach: 'incremental',
    ideas_by_decision: {accept: 2, pending: 1},
    ideas_by_source: {requested: 2, assumption: 1},
    effective_outcomes: [],
    ideas_total: 3,
    tasks_total: 2,
    unresolved_assumptions: 1,
  },
  context: {repository: '7'},
  generation: {state: 'completed'},
  setup: {state: 'ready', project: PROJECT},
  work_items: {by_state: {waiting: 1}, total: 1, outcomes_total: 2, outcomes_evidenced: 0},
  mappings: [
    {
      task_key: TASK_A,
      issue_id: '100',
      issue_index: '12',
      acceptance_id: IDEA,
      created_unix: 1720000000,
      effective: true,
    },
    {
      task_key: TASK_B,
      issue_id: '101',
      issue_index: '13',
      acceptance_id: IDEA,
      created_unix: 1720000000,
      effective: false,
    },
  ],
  attention: [{code: 'work_attention', issue: '12'}],
  intent: INTENT,
  revision: REVISION,
  state: 'accepted',
  observed_unix: 1720000300,
};

const acceptanceAccess = {
  repository: '7',
  observed_unix: 1720000300,
  viewer: {
    code_write: true,
    session_current: true,
    owner_or_admin: true,
    environment_owner: false,
    operator: false,
    member: true,
  },
  effective: {effective: true, dispatch_open: false, missing: []},
  required_actions: [],
  planning: [],
  planning_truncated: false,
};
const requirementsPreview = {
  repository: '7',
  issue: '12',
  title_digest: 'a'.repeat(64),
  title_occurrence: 'original',
  body_digest: 'b'.repeat(64),
  content_version: 3,
  native_revision: 41,
  comments: [
    {id: '21', digest: 'c'.repeat(64), content_version: 2},
    {id: '22', digest: 'd'.repeat(64), content_version: 1},
  ],
  dependencies: [{occurrence: '23', depends_on: '90'}],
};

test('maintainer reaccepts exact reviewed requirements and code prerequisite on a real engine', async (t) => {
  const prior = 'd' + '2'.repeat(24),
    prerequisite = 'd' + '3'.repeat(24);
  const page = await openView(t, 'detail', [
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: acceptanceAccess},
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12',
      status: 200,
      json: {
        ...issueDetail,
        status: {
          valid: false,
          revision: 4,
          reasons: ['objective_changed'],
          acceptance: {id: prior, issue_index: '12', approver: '1', native_revision: 40, content_version: 2},
        },
      },
    },
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12/acceptance-preview?comment_id=21&comment_id=22',
      status: 200,
      json: requirementsPreview,
    },
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/11',
      status: 200,
      json: {
        ...issueDetail,
        issue: '11',
        status: {
          valid: true,
          revision: 2,
          acceptance: {id: prerequisite, issue_index: '11', approver: '1', native_revision: 40, content_version: 1},
        },
      },
    },
    {
      method: 'POST',
      path: 'repositories/7/factory/issues/12/acceptances',
      status: 201,
      json: {depth: 2},
      echoAcceptance: true,
    },
  ]);
  const panel = page.getByRole('region', {name: 'Issue requirements acceptance'});
  await panel.getByLabel('Requirement comment IDs', {exact: true}).fill('21');
  await panel.getByLabel('Resolution comment IDs', {exact: true}).fill('22');
  await panel.getByRole('button', {name: 'Review current requirements', exact: true}).click();
  await page.waitForSelector('[data-requirements-preview]');
  assert.equal(
    await panel.getByRole('link', {name: 'Review comment 21', exact: true}).getAttribute('href'),
    '/soda-tester/demo/issues/12#issuecomment-21'
  );
  await panel.getByLabel('Prerequisite repository ID', {exact: true}).fill('7');
  await panel.getByLabel('Prerequisite issue number', {exact: true}).fill('11');
  await panel.getByLabel('Required outcome', {exact: true}).selectOption('code');
  const save = panel.getByRole('button', {name: 'Accept revised requirements', exact: true});
  assert.equal(await save.isDisabled(), true);
  await panel.getByRole('checkbox').check();
  await save.click();
  await panel
    .getByRole('status')
    .filter({hasText: 'Requirements accepted.'})
    .waitFor({timeout: 5000})
    .catch(async () => {
      throw Error(await panel.innerText());
    });
  const writes = (await workCalls(page)).filter((call) => call.method === 'POST');
  assert.equal(writes.length, 1);
  const body = writes[0]?.body as Record<string, unknown>;
  assert.match(String(body.command_id), /^[a-f0-9]{32}$/);
  assert.match(String(body.decision_id), /^d[a-f0-9]{24}$/);
  assert.equal(body.predecessor, prior);
  assert.equal(body.content_digest, requirementsPreview.body_digest);
  assert.equal(body.title_occurrence, 'original');
  assert.equal(body.native_revision, 41);
  assert.deepEqual(body.sources, [requirementsPreview.comments[0]]);
  assert.deepEqual(body.resolutions, [requirementsPreview.comments[1]]);
  assert.deepEqual(body.prerequisites, [
    {
      occurrence: '23',
      depends_on: '90',
      endpoint_repository: '7',
      endpoint_issue: '11',
      outcome: 'code',
      prerequisite_acceptance: prerequisite,
    },
  ]);
});

test('requirements preview changes and explicit stale refusal require a new review', async (t) => {
  const page = await openView(t, 'detail', [
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: acceptanceAccess},
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12/acceptance-preview',
      status: 200,
      json: {...requirementsPreview, comments: [], dependencies: []},
    },
    {
      method: 'POST',
      path: 'repositories/7/factory/issues/12/acceptances',
      status: 409,
      json: {error: 'objective_changed', message: 'The issue title or body changed; accept the new revision.'},
    },
  ]);
  const panel = page.getByRole('region', {name: 'Issue requirements acceptance'});
  await panel.getByRole('button', {name: 'Review current requirements', exact: true}).click();
  await page.waitForSelector('[data-requirements-preview]');
  await panel.getByLabel('Requirement comment IDs', {exact: true}).fill('21');
  assert.equal(await panel.locator('[data-requirements-preview]').count(), 0);
  await panel.getByLabel('Requirement comment IDs', {exact: true}).fill('');
  await panel.getByRole('button', {name: 'Review current requirements', exact: true}).click();
  await panel.getByRole('checkbox').check();
  await panel.getByRole('button', {name: 'Accept requirements', exact: true}).click();
  await panel.getByRole('status').filter({hasText: 'The issue title or body changed'}).waitFor();
  assert.equal(await panel.locator('[data-requirements-preview]').count(), 0);
  assert.equal(await panel.getByRole('button', {name: 'Retry same acceptance'}).count(), 0);
  assert.equal((await workCalls(page)).filter((call) => call.method === 'POST').length, 1);
});

test('unconfirmed acceptance retries its exact command and payload', async (t) => {
  const routes: StubRoute[] = [
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: acceptanceAccess},
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12/acceptance-preview',
      status: 200,
      json: {...requirementsPreview, comments: [], dependencies: []},
    },
    {
      method: 'POST',
      path: 'repositories/7/factory/issues/12/acceptances',
      status: 503,
      json: {message: 'Native evidence unavailable.'},
    },
  ];
  const page = await openView(t, 'detail', routes);
  const panel = page.getByRole('region', {name: 'Issue requirements acceptance'});
  await panel.getByRole('button', {name: 'Review current requirements', exact: true}).click();
  await panel.getByRole('checkbox').check();
  await panel.getByRole('button', {name: 'Accept requirements', exact: true}).click();
  const retry = panel.getByRole('button', {name: 'Retry same acceptance', exact: true});
  await retry.waitFor();
  const original = (await workCalls(page)).find((call) => call.method === 'POST');
  assert(original);
  await page.evaluate(
    (next) => window.factoryWork.setRoutes(next),
    [
      ...routes.filter((route) => route.method !== 'POST'),
      {
        method: 'POST',
        path: 'repositories/7/factory/issues/12/acceptances',
        status: 201,
        json: {depth: 1},
        echoAcceptance: true,
      },
    ]
  );
  await retry.click();
  await panel.getByRole('status').filter({hasText: 'Requirements accepted.'}).waitFor();
  assert.deepEqual((await workCalls(page)).find((call) => call.method === 'POST')?.body, original.body);
});

test('read-only viewer cannot preview requirements or retry completion review', async (t) => {
  const page = await openView(t, 'detail', [
    {
      method: 'GET',
      path: 'repositories/7/factory/access',
      status: 200,
      json: {...acceptanceAccess, viewer: {...acceptanceAccess.viewer, code_write: false}},
    },
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: attentionIssueDetail},
  ]);
  const panel = page.getByRole('region', {name: 'Issue requirements acceptance'});
  await panel.getByRole('status').filter({hasText: 'needs current repository code-write authority'}).waitFor();
  assert.equal(await panel.getByRole('button', {name: 'Review current requirements', exact: true}).isDisabled(), true);
  const completion = page.getByRole('region', {name: 'Completion review needs attention'});
  await completion.getByRole('button', {name: 'Retry completion review', exact: true}).waitFor();
  assert.equal(await completion.getByRole('button', {name: 'Retry completion review', exact: true}).isDisabled(), true);
  assert.equal(
    (await workCalls(page)).some((call) => call.path.includes('acceptance-preview') || call.method === 'POST'),
    false
  );
});

test('work overview renders the authoritative request overview on a real engine', async (t) => {
  const page = await openView(t, 'overview', [
    {method: 'GET', path: `factory/intents/${INTENT}/overview`, status: 200, json: requestOverviewRoute},
  ]);
  const overview = page.locator('soda-factory-work-overview');
  await overview.getByText('Let a team track demo work.').waitFor();
  await overview.getByText('1 waiting').waitFor();
  assert.equal(
    await overview.getByRole('link', {name: 'issue #12', exact: true}).getAttribute('href'),
    '/soda-tester/demo/issues/12'
  );
  assert.equal(
    await overview.getByRole('link', {name: 'Open issue #12', exact: true}).getAttribute('href'),
    '/soda-tester/demo/issues/12'
  );
  const calls = await workCalls(page);
  assert.deepEqual(
    calls.map((call) => call.path),
    [`factory/intents/${INTENT}/overview`]
  );
});

test('work detail retries the same completion command after lost replies and runs all interventions', async (t) => {
  const page = await openView(t, 'detail', [
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: attentionIssueDetail},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'POST',
      path: 'repositories/7/factory/actions',
      action: 'retry_completion',
      status: 200,
      echoCommand: true,
      loseReply: true,
      receiptPending: true,
      json: {
        command_id: '',
        event_id: attentionCompletion.completion_id,
        prior_cycle: attentionCompletion.cycle_id,
        successor_cycle: '',
        repository: '7',
        queued: true,
      },
    },
    {
      method: 'GET',
      path: `factory/runs/${RUN}`,
      status: 200,
      json: {
        run: {
          id: RUN,
          project_id: PROJECT,
          role: 'soda-coder',
          harness: 'codex-fixture',
          model: 'fixture',
          input_sha: HEAD,
          reconciled: false,
        },
        context_selection: {
          fingerprint: SNAPSHOT,
          scope_digest: SNAPSHOT,
          required_digest: SNAPSHOT,
          availability: 'complete',
          reason: 'context_selection_attention',
          action: 'hold',
          can_use_current: true,
        },
      },
    },
    {
      method: 'POST',
      path: `factory/runs/${RUN}/actions`,
      action: 'use_current_context',
      status: 200,
      echoCommand: true,
      json: {
        command_id: '',
        run_id: RUN,
        bundle: 'current',
        scope_digest: SNAPSHOT,
        fingerprint: SNAPSHOT,
        required_digest: SNAPSHOT,
      },
    },
    {
      method: 'POST',
      path: 'repositories/7/factory/actions',
      action: 'pause',
      status: 200,
      echoCommand: true,
      json: {
        command_id: '',
        paused: true,
        withdrawal: {repository: '7', revision: 6, cause: 'control_paused', closed_by: 'os-uid:1000'},
        runs: [{id: RUN, confirmed: true, uncertain: false, outcome: 'cancelled'}],
      },
    },
    {
      method: 'POST',
      path: 'repositories/7/factory/actions',
      action: 'resume',
      status: 200,
      echoCommand: true,
      json: {
        command_id: '',
        revision: 7,
        reopened: true,
        effective: {effective: true, dispatch_open: true, missing: []},
      },
    },
    {
      method: 'POST',
      path: `factory/runs/${RUN}/actions`,
      action: 'stop',
      status: 200,
      json: {run_id: RUN, confirmed: true, uncertain: false},
    },
    {
      method: 'POST',
      path: `factory/runs/${RUN}/actions`,
      action: 'retry',
      status: 200,
      echoCommand: true,
      json: {prior_run: RUN, command_id: '', reason: 'run_failed', queued: true},
    },
    {
      method: 'POST',
      path: `factory/runs/${RUN}/actions`,
      action: 'takeover',
      status: 200,
      echoCommand: true,
      json: {
        run: RUN,
        member: 'soda-tester',
        project: PROJECT,
        destination: '/home/soda-tester/takeover/d',
        command_id: '',
        copied: '2024-06-30T00:00:00Z',
      },
    },
    {
      method: 'POST',
      path: `environments/${ENVIRONMENT}/lifecycle`,
      status: 200,
      json: {
        environment: {id: PROJECT, ip: '10.0.0.2', running: false},
        boot_enabled: true,
        control: {
          withdrawal: {repository: '7', revision: 9, cause: 'project_stop', closed_by: 'os-uid:1000'},
          runs: [{id: RUN, confirmed: true, uncertain: false, outcome: 'cancelled'}],
          hold: true,
          hold_synced: true,
        },
      },
    },
  ]);
  const detail = page.locator('soda-factory-work-detail');
  const journey = detail.getByRole('region', {name: 'Work journey', exact: true});
  await journey.getByText(/Confidence 0\.28\. Cycle aaaaaaaaa: superseded\./).waitFor();
  await journey.getByText(/Confidence 0\.24\. Cycle bbbbbbbbb: attention\./).waitFor();
  await journey.getByText(/not a current wait/).waitFor();
  await journey
    .getByText(/Waiting: decision_not_actionable/)
    .first()
    .waitFor();
  await journey.getByText('Task task_a (lane a): complete, no native issue mapped yet.', {exact: true}).waitFor();
  await journey.getByText('Candidate aaaaaaaaaaaa over base bbbbbbbbbbbb: PR #34.').waitFor();
  const completion = detail.getByRole('region', {name: 'Completion review needs attention'});
  const retryCompletion = completion.getByRole('button', {name: 'Retry completion review', exact: true});
  await retryCompletion.waitFor();
  assert.equal((await workCalls(page)).filter((call) => call.method === 'POST').length, 0);
  await retryCompletion.click();
  await completion.getByText(/No retry was confirmed/).waitFor();
  const firstRetry = (await workCalls(page)).find(
    (call) => (call.body as {action?: string} | undefined)?.action === 'retry_completion'
  );
  await retryCompletion.click();
  await completion
    .getByText('Completion review retry queued. The original command receipt was recovered.', {exact: true})
    .waitFor();
  assert.equal(await retryCompletion.isDisabled(), true);
  const retries = (await workCalls(page)).filter(
    (call) => (call.body as {action?: string} | undefined)?.action === 'retry_completion'
  );
  assert.equal(retries.length, 2);
  assert.deepEqual(retries[1]?.body, firstRetry?.body);
  assert.deepEqual((firstRetry?.body as {completion: unknown}).completion, {
    event_id: attentionCompletion.completion_id,
    expected_prior_cycle: attentionCompletion.cycle_id,
  });
  assert.equal((await workCalls(page)).filter((call) => call.path.startsWith('factory/commands/')).length, 2);
  const contextChoice = detail.getByRole('region', {name: 'Context choice needs attention'});
  await contextChoice.getByRole('button', {name: 'Use current required context', exact: true}).waitFor();
  assert.equal(
    (await workCalls(page)).filter((call) => call.method === 'POST').length,
    2,
    'a context hold must not choose on read'
  );
  await contextChoice.getByRole('button', {name: 'Use current required context', exact: true}).click();
  await contextChoice
    .getByText('Current required context selected. Factory will recheck this run before starting.', {exact: true})
    .waitFor();
  assert.equal(
    await detail.getByRole('link', {name: 'Native issue'}).getAttribute('href'),
    '/soda-tester/demo/issues/12'
  );
  assert.equal(
    await detail.getByRole('link', {name: 'Native pull request'}).getAttribute('href'),
    '/soda-tester/demo/pulls/34'
  );

  await detail.getByRole('button', {name: 'Pause Factory work'}).click();
  await detail.getByText('Pause confirmed. Dispatch closed; 1 run stopped.').waitFor();
  await detail.getByRole('button', {name: 'Resume Factory work'}).click();
  await detail.getByText('Resume confirmed. Dispatch reopened.').waitFor();
  await detail.getByRole('button', {name: 'Stop run'}).click();
  await detail.getByText('Stop confirmed.').waitFor();
  await detail.getByRole('button', {name: 'Retry run'}).click();
  await detail.getByText('Retry queued. The prior run stays settled; no other attempt was created.').waitFor();
  await detail.getByRole('button', {name: 'Take over run'}).click();
  await detail.getByText('Takeover confirmed into /home/soda-tester/takeover/d.').waitFor();
  await detail.getByRole('button', {name: 'Stop Project for everyone'}).click();
  await detail.getByText('Click again to confirm stopping the project for everyone.').waitFor();
  await detail.getByRole('button', {name: 'Stop Project for everyone'}).click();
  await detail.getByText(/Project stop confirmed\. 1 run stopped\./).waitFor();

  const calls = await workCalls(page);
  const posts = calls.filter((call) => call.method === 'POST');
  assert.equal(posts.length, 9);
  const currentChoice = posts.find((call) => (call.body as {action?: string}).action === 'use_current_context');
  assert.deepEqual((currentChoice?.body as {context: unknown}).context, {
    fingerprint: SNAPSHOT,
    scope_digest: SNAPSHOT,
    required_digest: SNAPSHOT,
  });
  const pause = posts.find((call) => (call.body as {action?: string}).action === 'pause');
  const pauseBody = pause?.body as {command_id: string} | undefined;
  assert(pauseBody !== undefined);
  assert.match(pauseBody.command_id, /^[0-9a-f]{32}$/);
  assert.equal(pause?.path, 'repositories/7/factory/actions');
  const stop = posts.find((call) => (call.body as {action?: string}).action === 'stop');
  assert.equal(stop?.path, `factory/runs/${RUN}/actions`);
  const projectStop = posts.find((call) => call.path === `environments/${ENVIRONMENT}/lifecycle`);
  assert.deepEqual(projectStop?.body, {action: 'stop', confirm_stop: true});
});

test('direction revision shows allowance reference and history, then adjusts allowance', async (t) => {
  const page = await openView(t, 'direction', [
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {method: 'GET', path: `factory/intents/${INTENT}/allowance`, status: 200, json: allowance},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/revisions?limit=2`,
      status: 200,
      json: {intent: INTENT, observed_unix: 1728670000, revisions: [historyEntry, historyFirst]},
    },
    {
      method: 'POST',
      path: 'factory/intent-allowances',
      status: 201,
      json: {...allowance, id: ALLOWANCE_NEXT, predecessor: ALLOWANCE, amount: 60, ceiling: 60},
    },
  ]);
  const direction = page.locator('soda-factory-direction-revision');
  await direction.getByText(/23 available/).waitFor();
  await direction.getByText('2 revisions shown, newest first.').waitFor();
  await direction.getByText('Submitted graph: 2 submitted keys.').waitFor();
  await direction.locator('input[name="amount"]').fill('60');
  await direction.locator('input[name="effect"]').fill('cover review queue');
  await direction.getByRole('button', {name: 'Adjust allowance'}).click();
  await direction.getByText('Allowance adjusted to 60 planning calls.').waitFor();

  const calls = await workCalls(page);
  const post = calls.find((call) => call.method === 'POST' && call.path === 'factory/intent-allowances');
  const body = post?.body as Record<string, unknown>;
  assert.equal(body.intent, INTENT);
  assert.equal(body.predecessor, ALLOWANCE);
  assert.equal(body.amount, 60);
  assert.equal(body.ceiling, 60);
  assert.equal(body.unit, 'planning_calls');
  assert.equal(body.reserved, 5);
  assert.equal(body.consumed, 12);
  assert.equal(body.effect, 'cover review queue');
});

test('allowance increase with a blank effect records the mechanical delta effect', async (t) => {
  const page = await openView(t, 'direction', [
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {method: 'GET', path: `factory/intents/${INTENT}/allowance`, status: 200, json: allowance},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/revisions?limit=2`,
      status: 200,
      json: {intent: INTENT, observed_unix: 1728670000, revisions: [historyEntry, historyFirst]},
    },
    {
      method: 'POST',
      path: 'factory/intent-allowances',
      status: 201,
      json: {...allowance, id: ALLOWANCE_NEXT, predecessor: ALLOWANCE, amount: 60, ceiling: 60},
    },
  ]);
  const direction = page.locator('soda-factory-direction-revision');
  await direction.getByText(/23 available/).waitFor();
  await direction.locator('input[name="amount"]').fill('60');
  await direction.getByRole('button', {name: 'Adjust allowance'}).click();
  await direction.getByText('Allowance adjusted to 60 planning calls.').waitFor();

  const calls = await workCalls(page);
  const post = calls.find((call) => call.method === 'POST' && call.path === 'factory/intent-allowances');
  const body = post?.body as Record<string, unknown>;
  assert.equal(body.amount, 60);
  assert.equal(body.ceiling, 60);
  assert.equal(body.reserved, 5);
  assert.equal(body.consumed, 12);
  assert.equal(body.effect, 'Authorize 20 additional planning calls (60 total).');
});

test('allowance below recorded usage is refused locally without posting', async (t) => {
  const page = await openView(t, 'direction', [
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {method: 'GET', path: `factory/intents/${INTENT}/allowance`, status: 200, json: allowance},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/revisions?limit=2`,
      status: 200,
      json: {intent: INTENT, observed_unix: 1728670000, revisions: [historyEntry, historyFirst]},
    },
  ]);
  const direction = page.locator('soda-factory-direction-revision');
  await direction.getByText(/23 available/).waitFor();
  await direction.locator('input[name="amount"]').fill('10');
  await direction.getByRole('button', {name: 'Adjust allowance'}).click();
  await direction.getByText('Allowance amount must cover 17 reserved and consumed planning calls.').waitFor();

  const calls = await workCalls(page);
  assert.equal(calls.filter((call) => call.method === 'POST' && call.path === 'factory/intent-allowances').length, 0);
});

test('optional effects gate per checkpoint and save exact identity', async (t) => {
  const prConfig = {
    event: 'pr_created',
    revision: '4',
    enabled: true,
    capable: true,
    actor_id: '1001',
    repository_id: '7',
    reviewer_ids: ['11', '12'],
  };
  const page = await openView(t, 'effects', [
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/issue_created',
      status: 200,
      json: {event: 'issue_created', revision: '0', enabled: false, capable: false, actor_id: '', repository_id: '7'},
    },
    {method: 'GET', path: 'repositories/7/factory/lifecycle/pr_created', status: 200, json: prConfig},
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/verified_merge',
      status: 200,
      json: {event: 'verified_merge', revision: '0', enabled: false, capable: true, actor_id: '', repository_id: '7'},
    },
    {method: 'GET', path: 'repositories/7/factory/lifecycle/issue_created/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/lifecycle/pr_created/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/lifecycle/verified_merge/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'PUT',
      path: 'repositories/7/factory/lifecycle/pr_created',
      status: 200,
      json: {...prConfig, revision: '5'},
    },
  ]);
  const effects = page.locator('soda-factory-optional-effects');
  await effects.getByText('Pull request human review').waitFor();
  await effects.getByText('its native capability is not runnable yet').waitFor();
  await effects.getByText('Not configured yet.').waitFor();
  await effects.getByText(/You hold: code write, owner or admin/).waitFor();
  const pr = effects.locator('li', {hasText: 'Pull request human review'});
  await pr.locator('input[name="reviewer_ids"]').fill('11, 12, 13');
  await pr.getByRole('button', {name: 'Save effect configuration'}).click();
  await effects.getByText('Effect configuration saved.').waitFor();

  const calls = await workCalls(page);
  const put = calls.find((call) => call.method === 'PUT');
  assert.deepEqual(put?.body, {revision: 4, enabled: true, reviewer_ids: [11, 12, 13]});
});

test('optional effects map an unavailable executor refusal distinctly', async (t) => {
  const prConfig = {
    event: 'pr_created',
    revision: '4',
    enabled: true,
    capable: true,
    actor_id: '1001',
    repository_id: '7',
    reviewer_ids: ['11', '12'],
  };
  const page = await openView(t, 'effects', [
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/issue_created',
      status: 200,
      json: {event: 'issue_created', revision: '0', enabled: false, capable: false, actor_id: '', repository_id: '7'},
    },
    {method: 'GET', path: 'repositories/7/factory/lifecycle/pr_created', status: 200, json: prConfig},
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/verified_merge',
      status: 200,
      json: {event: 'verified_merge', revision: '0', enabled: false, capable: true, actor_id: '', repository_id: '7'},
    },
    {method: 'GET', path: 'repositories/7/factory/lifecycle/issue_created/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/lifecycle/pr_created/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/lifecycle/verified_merge/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'PUT',
      path: 'repositories/7/factory/lifecycle/pr_created',
      status: 409,
      json: {error: {code: 'executor_unavailable'}},
    },
  ]);
  const effects = page.locator('soda-factory-optional-effects');
  await effects.getByText('Pull request human review').waitFor();
  const pr = effects.locator('li', {hasText: 'Pull request human review'});
  await pr.getByRole('button', {name: 'Save effect configuration'}).click();
  await effects.getByText('No lifecycle executor is configured yet; the saved preference is kept.').waitFor();
});

test('optional effects link a journaled pull number and leave unclaimed rows text-only', async (t) => {
  const page = await openView(t, 'effects', [
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/issue_created',
      status: 200,
      json: {event: 'issue_created', revision: '0', enabled: false, capable: true, actor_id: '', repository_id: '7'},
    },
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/pr_created',
      status: 200,
      json: {
        event: 'pr_created',
        revision: '4',
        enabled: true,
        capable: true,
        actor_id: '1001',
        submitter_id: '7',
        executor_ready: true,
        repository_id: '7',
        reviewer_ids: ['11'],
      },
    },
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/verified_merge',
      status: 200,
      json: {event: 'verified_merge', revision: '0', enabled: false, capable: true, actor_id: '', repository_id: '7'},
    },
    {method: 'GET', path: 'repositories/7/factory/lifecycle/issue_created/actions?issue=12', status: 200, json: []},
    {
      method: 'GET',
      path: 'repositories/7/factory/lifecycle/pr_created/actions?issue=12',
      status: 200,
      json: {
        actions: [
          {
            id: 'action-bound-1',
            event: 'pr_created',
            action: 'pr_reviewer_request',
            actor_id: '7',
            issue_id: '12',
            target_id: '34',
            config_owner: '99',
            pr_number: '34',
            attempts: 1,
            max_attempts: 5,
            revoked: false,
            settled: false,
            results: {'34': 'delivered'},
          },
          {
            id: 'action-unclaimed-2',
            event: 'pr_created',
            action: 'pr_reviewer_request',
            actor_id: '0',
            issue_id: '12',
            attempts: 0,
            max_attempts: 5,
            revoked: false,
            settled: false,
            results: {},
          },
        ],
      },
    },
    {method: 'GET', path: 'repositories/7/factory/lifecycle/verified_merge/actions?issue=12', status: 200, json: []},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
  ]);
  const effects = page.locator('soda-factory-optional-effects');
  await effects.getByText('Pull request human review').waitFor();
  const pr = effects.locator('li', {hasText: 'Pull request human review'});
  await pr.getByText(/Submitter 7, executor ready/).waitFor();
  await pr.getByText(/Submitted by 7, authorized by 99/).waitFor();
  const links = pr.getByRole('link', {name: 'Native target'});
  assert.equal(await links.count(), 1);
  assert.equal(await links.first().getAttribute('href'), '/soda-tester/demo/pulls/34');
  await pr.getByText(/Submitter unclaimed/).waitFor();
  await pr.getByText(/Native target unresolved/).waitFor();
});

test('work plan renders recorded steps and the empty comparison note', async (t) => {
  const page = await openView(t, 'plan', [
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12/plan?limit=5',
      status: 200,
      json: {
        repository: '7',
        issue: '12',
        observed_unix: 1720000300,
        freshness: {state: 'current', provisional: false},
        plan: {
          candidate: {id: 'plan-1', repository: '7', issue: '12', readiness: 'queued', first_seen_unix: 1720000000},
          missing: [],
          steps: ['Implement the list.', 'Cover the empty state.'],
          strategy: 'incremental_acceptance',
          snapshot: SNAPSHOT,
          outcome: 'planned',
          created_unix: 1720000100,
        },
        comparisons: [],
      },
    },
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
  ]);
  const plan = page.locator('soda-factory-work-plan');
  await plan.getByText('Implement the list.').waitFor();
  await plan.getByText('Cover the empty state.').waitFor();
  await plan.getByText('No recorded priority comparison').waitFor();
});

test('overview scope feeds derived plan, detail and revision mounts, then retires', async (t) => {
  const page = await openView(t, 'overview', [
    {method: 'GET', path: `factory/intents/${INTENT}/overview`, status: 200, json: requestOverviewRoute},
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12/plan?limit=5',
      status: 200,
      json: {
        repository: '7',
        issue: '12',
        observed_unix: 1720000300,
        freshness: {state: 'current', provisional: false},
        plan: {
          candidate: {id: 'plan-1', repository: '7', issue: '12', readiness: 'queued', first_seen_unix: 1720000000},
          missing: [],
          steps: ['Implement the list.'],
          strategy: 'incremental_acceptance',
          snapshot: SNAPSHOT,
          outcome: 'planned',
          created_unix: 1720000100,
        },
        comparisons: [],
      },
    },
    {method: 'GET', path: `factory/intents/${INTENT}/allowance`, status: 200, json: allowance},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/revisions?limit=2`,
      status: 200,
      json: {intent: INTENT, observed_unix: 1728670000, revisions: [historyEntry, historyFirst]},
    },
  ]);
  await page.waitForFunction(() => {
    const el = document.querySelector('soda-factory-work-overview') as unknown as {
      boundScope?: {issues: unknown[]};
      busy?: boolean;
    };
    return (el?.boundScope?.issues.length ?? 0) > 0 && el?.busy === false;
  });
  const feed = (await page.evaluate(() => {
    const el = document.querySelector('soda-factory-work-overview') as unknown as {
      boundScope: {repository: string; intentID?: string; issues: {issue: string; capabilities?: unknown}[]};
    };
    return el.boundScope;
  })) as {repository: string; intentID?: string; issues: {issue: string; capabilities?: unknown}[]};
  assert.equal(feed.repository, '7');
  assert.equal(feed.intentID, INTENT);
  assert.deepEqual(
    feed.issues.map((entry) => entry.issue),
    ['12']
  );
  assert.equal(feed.issues[0]?.capabilities, undefined);

  await page.evaluate(() => {
    (window as unknown as {lastScope?: unknown}).lastScope = undefined;
    document.querySelector('main')?.addEventListener('soda-factory-overview-scope', (event) => {
      (window as unknown as {lastScope?: unknown}).lastScope = (event as CustomEvent).detail;
    });
    (document.querySelector('soda-factory-work-overview') as unknown as {refresh: () => void}).refresh();
  });
  await page.waitForFunction(() => (window as unknown as {lastScope?: unknown}).lastScope !== undefined);
  const evented = (await page.evaluate(
    () => (window as unknown as {lastScope: {repository: string; issues: {issue: string}[]}}).lastScope
  )) as {repository: string; issues: {issue: string}[]};
  assert.equal(evented.repository, '7');
  assert.deepEqual(
    evented.issues.map((entry) => entry.issue),
    ['12']
  );

  const before = (await workCalls(page)).length;
  await page.evaluate(() => {
    window.factoryWork.mount('plan');
    window.factoryWork.mount('detail');
    window.factoryWork.mount('direction');
  });
  await page.locator('soda-factory-work-plan').getByText('Implement the list.').waitFor();
  await page.locator('soda-factory-work-detail').getByRole('link', {name: 'Native issue'}).waitFor();
  await page.locator('soda-factory-direction-revision').getByText('2 revisions shown, newest first.').waitFor();
  const fresh = (await workCalls(page)).slice(before).filter((call) => call.method === 'GET');
  const paths = fresh.map((call) => call.path);
  assert.ok(paths.includes('repositories/7/factory/issues/12/plan?limit=5'));
  assert.ok(paths.includes('repositories/7/factory/issues/12'));
  assert.ok(paths.includes(`factory/intents/${INTENT}/allowance`));
  assert.ok(paths.includes(`factory/intents/${INTENT}/revisions?limit=2`));

  await page.evaluate(() => {
    (window as unknown as {lastScope?: unknown}).lastScope = undefined;
    (document.querySelector('soda-factory-work-overview') as unknown as {context: unknown}).context = undefined;
    for (const tag of ['soda-factory-work-plan', 'soda-factory-work-detail', 'soda-factory-direction-revision'])
      (document.querySelector(tag) as unknown as {context: unknown}).context = undefined;
  });
  await page.waitForFunction(() => (window as unknown as {lastScope?: unknown}).lastScope !== undefined);
  const retired = (await page.evaluate(
    () => (window as unknown as {lastScope: {repository: string; issues: unknown[]}}).lastScope
  )) as {repository: string; issues: unknown[]};
  assert.equal(retired.repository, '');
  assert.deepEqual(retired.issues, []);
  await page.waitForFunction(() => {
    const text = (tag: string) => document.querySelector(tag)?.textContent ?? '';
    return (
      !text('soda-factory-work-overview').includes('Let a team track demo work.') &&
      text('soda-factory-work-plan') === '' &&
      text('soda-factory-work-detail') === '' &&
      text('soda-factory-direction-revision').includes('No planning allowance recorded.')
    );
  });
  assert.equal(await page.locator('soda-factory-work-plan').textContent(), '');
  assert.equal(await page.locator('soda-factory-work-detail').textContent(), '');
});

test('intent overview failures surface unavailable with no fallback reads', async (t) => {
  const bait: StubRoute[] = [
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/generation`,
      status: 200,
      json: {intent_id: INTENT, revision_id: REVISION, state: 'completed', task_issue_ids: {[TASK_A]: 12}},
    },
    {
      method: 'GET',
      path: 'repositories/7/factory',
      status: 200,
      json: {
        repository: '7',
        queue: {queued: 0, active: 0},
        effective: {effective: true, dispatch_open: true, missing: []},
      },
    },
  ];
  const failures: {route: StubRoute; message: string}[] = [
    {
      route: {method: 'GET', path: `factory/intents/${INTENT}/overview`, status: 500, json: {}},
      message: 'Factory read failed (500).',
    },
    {
      route: {
        method: 'GET',
        path: `factory/intents/${INTENT}/overview`,
        status: 200,
        json: {
          ...requestOverviewRoute,
          work_items: {...requestOverviewRoute.work_items, by_state: {waiting: '1'}},
        },
      },
      message: 'Invalid or mismatched Soda response',
    },
  ];
  for (const failure of failures) {
    const page = await openView(t, 'overview', [failure.route, ...bait]);
    const overview = page.locator('soda-factory-work-overview');
    await overview.getByText(failure.message).waitFor();
    const calls = await workCalls(page);
    assert.deepEqual(
      calls.map((call) => call.path),
      [`factory/intents/${INTENT}/overview`]
    );
    const scope = (await page.evaluate(() => {
      const el = document.querySelector('soda-factory-work-overview') as unknown as {
        boundScope: {repository: string; intentID?: string; issues: unknown[]};
      };
      return el.boundScope;
    })) as {repository: string; intentID?: string; issues: unknown[]};
    assert.equal(scope.repository, '7');
    assert.equal(scope.intentID, INTENT);
    assert.deepEqual(scope.issues, []);
    assert.equal(await overview.getByText('No counted work.').count(), 0);
  }
});

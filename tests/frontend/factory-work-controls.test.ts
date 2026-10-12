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

test('work overview renders direction, dependencies and summary on a real engine', async (t) => {
  const page = await openView(t, 'overview', [
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/generation`,
      status: 200,
      json: {
        intent_id: INTENT,
        revision_id: REVISION,
        state: 'completed',
        task_issue_ids: {[TASK_A]: 12, [TASK_B]: 13},
      },
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
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
  ]);
  const overview = page.locator('soda-factory-work-overview');
  await overview.getByText('Let a team track demo work.').waitFor();
  await overview.getByText('Build demo detail.').waitFor();
  const open = overview.getByRole('link', {name: 'Open issue #13'});
  assert.equal(await open.getAttribute('href'), '/soda-tester/demo/issues/13');
});

test('work detail renders evidence links and runs all six interventions', async (t) => {
  const page = await openView(t, 'detail', [
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
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
  assert.equal(posts.length, 6);
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
      json: {...allowance, id: ALLOWANCE_NEXT, predecessor: ALLOWANCE, amount: 60, reserved: 0, consumed: 0},
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
  assert.equal(body.unit, 'planning_calls');
  assert.equal(body.effect, 'cover review queue');
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
  await effects.getByText('its native capability checkpoint has not passed').waitFor();
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

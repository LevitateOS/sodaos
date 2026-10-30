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

type WorkView = 'overview' | 'detail' | 'plan' | 'direction';

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
    '/soda-tester/demo/issues/12#issuepost-21'
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
  assert.deepEqual((firstRetry?.body as {completion: unknown} | undefined)?.completion, {
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
  assert.deepEqual((currentChoice?.body as {context: unknown} | undefined)?.context, {
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

test('held context refresh with the same run ID enables use-current from the new selection', async (t) => {
  const FRESH = 'b'.repeat(64);
  const runPayload = (selection: unknown) => ({
    run: {
      id: RUN,
      project_id: PROJECT,
      role: 'soda-coder',
      harness: 'codex-fixture',
      model: 'fixture',
      input_sha: HEAD,
      reconciled: false,
    },
    ...(selection === undefined ? {} : {context_selection: selection}),
  });
  const held = {
    fingerprint: SNAPSHOT,
    scope_digest: SNAPSHOT,
    required_digest: SNAPSHOT,
    availability: 'incomplete',
    reason: 'missing required inputs',
    action: 'hold',
    can_use_current: false,
  };
  const refreshed = {
    fingerprint: FRESH,
    scope_digest: FRESH,
    required_digest: FRESH,
    availability: 'complete',
    reason: 'required context is complete',
    action: 'hold',
    can_use_current: true,
  };
  const page = await openView(t, 'detail', [
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    {method: 'GET', path: `factory/runs/${RUN}`, status: 200, json: runPayload(held)},
  ]);
  const detail = page.locator('soda-factory-work-detail');
  const contextChoice = detail.getByRole('region', {name: 'Context choice needs attention'});
  const useCurrent = contextChoice.getByRole('button', {name: 'Use current required context', exact: true});
  await contextChoice.getByText('Required current context is unavailable.').waitFor();
  assert.equal(await useCurrent.isDisabled(), true);
  await page.evaluate(
    (next) => window.factoryWork.setRoutes(next),
    [
      {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
      {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
      {method: 'GET', path: `factory/runs/${RUN}`, status: 200, json: runPayload(refreshed)},
    ]
  );
  await detail.getByRole('button', {name: 'Refresh detail', exact: true}).click();
  await contextChoice.getByText('The required current context is complete.').waitFor();
  assert.equal(await useCurrent.isDisabled(), false);
  await page.evaluate(
    (next) => window.factoryWork.setRoutes(next),
    [
      {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
      {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
      {method: 'GET', path: `factory/runs/${RUN}`, status: 200, json: runPayload(undefined)},
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
          scope_digest: FRESH,
          fingerprint: FRESH,
          required_digest: FRESH,
        },
      },
    ]
  );
  await useCurrent.click();
  // The resolve receipt dispatches soda-context-resolved, so the parent
  // immediately reloads the resolved run and the hold prompt unmounts; the
  // request payload plus the resolved readback are the durable proof.
  await page.waitForFunction(
    () => document.querySelector('soda-factory-work-detail [aria-label="Context choice needs attention"]') === null
  );
  const currentChoice = (await workCalls(page)).find(
    (call) => (call.body as {action?: string} | undefined)?.action === 'use_current_context'
  );
  assert.deepEqual((currentChoice?.body as {context: unknown} | undefined)?.context, {
    fingerprint: FRESH,
    scope_digest: FRESH,
    required_digest: FRESH,
  });
  assert.equal(await detail.getByRole('region', {name: 'Context choice needs attention'}).count(), 0);
});

test('stale in-flight context read resolving after scope change writes nothing', async (t) => {
  const page = await openView(t, 'overview', []);
  const outcome = await page.evaluate(async () => {
    const RUN_B = 'b'.repeat(32);
    const RUN_STALE = 'e'.repeat(32);
    const pending: {path: string; resolve: (value: Response) => void}[] = [];
    const deferred = {
      request: (path: string) => new Promise<Response>((resolve) => void pending.push({path, resolve})),
    };
    const serving = {
      request: async (path: string) => {
        if (path.includes('/factory/issues/')) return Response.json({journey: {lineage: {latest_run: RUN_B}}});
        return Response.json({
          run: {
            id: RUN_B,
            project_id: 'p0123456789abcdef01234567',
            role: 'soda-coder',
            harness: 'codex-fixture',
            model: 'fixture',
            input_sha: 'a'.repeat(40),
            reconciled: false,
          },
          context_selection: {
            fingerprint: 'b'.repeat(64),
            scope_digest: 'b'.repeat(64),
            required_digest: 'b'.repeat(64),
            availability: 'complete',
            reason: 'required context is complete',
            action: 'hold',
            can_use_current: true,
          },
        });
      },
    };
    const host = document.createElement('soda-factory-context-resolution') as unknown as HTMLElement & {
      context: unknown;
      updateComplete: Promise<unknown>;
      selectedRun?: {id: string};
    };
    document.querySelector('main')?.append(host);
    host.context = {transport: deferred, repository: '7', issue: '12'};
    await host.updateComplete;
    const firstPaths = pending.map((entry) => entry.path);
    // Supersede the deferred lookup with a newer transport before the stale read lands.
    host.context = {transport: serving, repository: '7', issue: '12'};
    await host.updateComplete;
    let settled = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      if (host.selectedRun?.id === RUN_B) {
        settled = true;
        break;
      }
      await new Promise((done) => setTimeout(done, 10));
    }
    const staleIssue = pending.shift();
    staleIssue?.resolve(Response.json({journey: {lineage: {latest_run: RUN_STALE}}}));
    await new Promise((done) => setTimeout(done, 0));
    await new Promise((done) => setTimeout(done, 0));
    const staleRun = pending.shift();
    staleRun?.resolve(
      Response.json({
        run: {
          id: RUN_STALE,
          project_id: 'p0123456789abcdef01234567',
          role: 'soda-coder',
          harness: 'codex-fixture',
          model: 'fixture',
          input_sha: 'a'.repeat(40),
          reconciled: false,
        },
        context_selection: {
          fingerprint: 'c'.repeat(64),
          scope_digest: 'c'.repeat(64),
          required_digest: 'c'.repeat(64),
          availability: 'incomplete',
          reason: 'stale read must not win',
          action: 'hold',
          can_use_current: false,
        },
      })
    );
    await new Promise((done) => setTimeout(done, 0));
    await new Promise((done) => setTimeout(done, 0));
    await host.updateComplete;
    const button = (host as unknown as Element).querySelector('button');
    return {
      firstPaths,
      settled,
      hadStaleIssue: staleIssue !== undefined,
      hadStaleRun: staleRun !== undefined,
      leftoverPaths: pending.map((entry) => entry.path),
      selectedId: host.selectedRun?.id,
      buttonDisabled: button ? button.disabled : null,
    };
  });
  assert.deepEqual(outcome.firstPaths, ['repositories/7/factory/issues/12']);
  assert.equal(outcome.settled, true);
  assert.equal(outcome.hadStaleIssue, true);
  assert.equal(outcome.hadStaleRun, true);
  assert.deepEqual(outcome.leftoverPaths, []);
  assert.equal(outcome.selectedId, 'b'.repeat(32));
  assert.equal(outcome.buttonDisabled, false);
});

test('direction revision shows history without any planning-allowance read', async (t) => {
  const page = await openView(t, 'direction', [
    {method: 'GET', path: `factory/intents/${INTENT}`, status: 200, json: intentRevision},
    {
      method: 'GET',
      path: `factory/intents/${INTENT}/revisions?limit=2`,
      status: 200,
      json: {intent: INTENT, observed_unix: 1728670000, revisions: [historyEntry, historyFirst]},
    },
  ]);
  const direction = page.locator('soda-factory-direction-revision');
  await direction.getByText('2 revisions shown, newest first.').waitFor();
  await direction.getByText('Submitted graph: 2 submitted keys.').waitFor();
  assert.doesNotMatch(await direction.innerText(), /allowance|planning calls/i);

  const calls = await workCalls(page);
  assert.equal(calls.filter((call) => /allowance|factory\/access/.test(call.path)).length, 0);
});

test('work plan renders the empty comparison note', async (t) => {
  const page = await openView(t, 'plan', [
    {
      method: 'GET',
      path: 'repositories/7/factory/issues/12/plan?limit=5',
      status: 200,
      json: {
        repository: '7',
        issue: '12',
        observed_unix: 1720000300,
        comparisons: [],
      },
    },
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: issueDetail},
  ]);
  const plan = page.locator('soda-factory-work-plan');
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
        comparisons: [],
      },
    },
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
  await page.locator('soda-factory-work-plan').getByText('No recorded priority comparison').waitFor();
  await page.locator('soda-factory-work-detail').getByRole('link', {name: 'Native issue'}).waitFor();
  await page.locator('soda-factory-direction-revision').getByText('2 revisions shown, newest first.').waitFor();
  const fresh = (await workCalls(page)).slice(before).filter((call) => call.method === 'GET');
  const paths = fresh.map((call) => call.path);
  assert.ok(paths.includes('repositories/7/factory/issues/12/plan?limit=5'));
  assert.ok(paths.includes('repositories/7/factory/issues/12'));
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
      text('soda-factory-direction-revision').includes('No accepted direction recorded.')
    );
  });
  assert.equal(await page.locator('soda-factory-work-plan').textContent(), '');
  assert.equal(await page.locator('soda-factory-work-detail').textContent(), '');
});

test('same-issue refresh re-reads the resolution child and enables use-current', async (t) => {
  const page = await openView(t, 'overview', []);
  const outcome = await page.evaluate(async () => {
    const RUN_A = 'a'.repeat(32);
    let runs = 0;
    let complete = false;
    const transport = {
      request: async (path: string) => {
        if (path === 'repositories/7/factory/issues/12')
          return Response.json({journey: {lineage: {latest_run: RUN_A}}});
        if (path === `factory/runs/${RUN_A}`) {
          runs += 1;
          return Response.json({
            run: {
              id: RUN_A,
              project_id: 'p0123456789abcdef01234567',
              role: 'soda-coder',
              harness: 'codex-fixture',
              model: 'fixture',
              input_sha: 'a'.repeat(40),
              reconciled: false,
            },
            context_selection: {
              fingerprint: complete ? 'b'.repeat(64) : 'a'.repeat(64),
              scope_digest: complete ? 'b'.repeat(64) : 'a'.repeat(64),
              required_digest: complete ? 'b'.repeat(64) : 'a'.repeat(64),
              availability: complete ? 'complete' : 'incomplete',
              reason: complete ? 'required context is complete' : 'missing required inputs',
              action: 'hold',
              can_use_current: complete,
            },
          });
        }
        return Response.json({}, {status: 404});
      },
    };
    type Host = HTMLElement & {
      context: unknown;
      updateComplete: Promise<unknown>;
      selectedRun?: {id: string};
      refresh?: () => void;
    };
    const host = document.createElement('soda-factory-context-resolution') as unknown as Host;
    document.querySelector('main')?.append(host);
    host.context = {transport, repository: '7', issue: '12'};
    await host.updateComplete;
    for (let attempt = 0; attempt < 100 && runs === 0; attempt++) await new Promise((done) => setTimeout(done, 10));
    await host.updateComplete;
    for (let attempt = 0; attempt < 100 && !host.querySelector('button'); attempt++)
      await new Promise((done) => setTimeout(done, 10));
    await host.updateComplete;
    const disabledBefore = (host.querySelector('button') as HTMLButtonElement | null)?.disabled ?? null;
    const readsBefore = runs;
    // Same-issue re-entry without an explicit refresh skips its read.
    host.context = {transport, repository: '7', issue: '12'};
    await host.updateComplete;
    await new Promise((done) => setTimeout(done, 50));
    const readsAfterResend = runs;
    // Corrected inputs land server-side; the explicit refresh re-reads.
    complete = true;
    host.refresh?.();
    for (let attempt = 0; attempt < 100 && runs < readsBefore + 1; attempt++)
      await new Promise((done) => setTimeout(done, 10));
    await host.updateComplete;
    let enabled = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      await host.updateComplete;
      if ((host.querySelector('button') as HTMLButtonElement | null)?.disabled === false) {
        enabled = true;
        break;
      }
      await new Promise((done) => setTimeout(done, 10));
    }
    return {readsBefore, readsAfterResend, runsAfter: runs, disabledBefore, enabled};
  });
  assert.equal(outcome.readsBefore, 1);
  assert.equal(outcome.readsAfterResend, 1);
  assert.equal(outcome.disabledBefore, true);
  assert.equal(outcome.runsAfter, 2);
  assert.equal(outcome.enabled, true);
});

test('delayed use-current confirmation for issue A never renders on issue B', async (t) => {
  const page = await openView(t, 'overview', []);
  const outcome = await page.evaluate(async () => {
    const RUN_A = 'a'.repeat(32);
    const RUN_B = 'b'.repeat(32);
    const DIGEST = 'b'.repeat(64);
    const payload = (id: string) => ({
      run: {
        id,
        project_id: 'p0123456789abcdef01234567',
        role: 'soda-coder',
        harness: 'codex-fixture',
        model: 'fixture',
        input_sha: 'a'.repeat(40),
        reconciled: false,
      },
      context_selection: {
        fingerprint: DIGEST,
        scope_digest: DIGEST,
        required_digest: DIGEST,
        availability: 'complete',
        reason: 'required context is complete',
        action: 'hold',
        can_use_current: true,
      },
    });
    let releaseA: ((value: Response) => void) | undefined;
    const transport = {
      request: async (path: string, init?: RequestInit) => {
        if (path === 'repositories/7/factory/issues/12')
          return Response.json({journey: {lineage: {latest_run: RUN_A}}});
        if (path === 'repositories/7/factory/issues/13')
          return Response.json({journey: {lineage: {latest_run: RUN_B}}});
        if ((init?.method ?? 'GET') === 'GET' && path === `factory/runs/${RUN_A}`) return Response.json(payload(RUN_A));
        if ((init?.method ?? 'GET') === 'GET' && path === `factory/runs/${RUN_B}`) return Response.json(payload(RUN_B));
        if (path === `factory/runs/${RUN_A}/actions`) {
          await new Promise<Response>((resolve) => {
            releaseA = resolve;
          });
          const body = JSON.parse(String(init?.body ?? '{}')) as {command_id?: string};
          return Response.json({
            command_id: body.command_id,
            run_id: RUN_A,
            bundle: 'current',
            scope_digest: DIGEST,
            fingerprint: DIGEST,
            required_digest: DIGEST,
          });
        }
        return Response.json({}, {status: 404});
      },
    };
    type Host = HTMLElement & {
      context: unknown;
      updateComplete: Promise<unknown>;
      selectedRun?: {id: string};
    };
    const host = document.createElement('soda-factory-context-resolution') as unknown as Host;
    let resolved = 0;
    host.addEventListener('soda-context-resolved', () => {
      resolved += 1;
    });
    document.querySelector('main')?.append(host);
    const button = () => host.querySelector('button') as HTMLButtonElement | null;
    const sleep = (ms: number) => new Promise((done) => setTimeout(done, ms));
    host.context = {transport, repository: '7', issue: '12'};
    await host.updateComplete;
    for (let attempt = 0; attempt < 100 && host.selectedRun?.id !== RUN_A; attempt++) {
      await sleep(10);
      await host.updateComplete;
    }
    for (let attempt = 0; attempt < 100 && button()?.disabled !== false; attempt++) await sleep(10);
    await host.updateComplete;
    button()?.click();
    await host.updateComplete;
    await sleep(20);
    // Open issue B while A's confirmation reply is still delayed.
    host.context = {transport, repository: '7', issue: '13'};
    await host.updateComplete;
    for (let attempt = 0; attempt < 100 && host.selectedRun?.id !== RUN_B; attempt++) {
      await sleep(10);
      await host.updateComplete;
    }
    let bEnabled = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      await host.updateComplete;
      if (button()?.disabled === false) {
        bEnabled = true;
        break;
      }
      await sleep(10);
    }
    const statusOnB = host.textContent ?? '';
    releaseA?.(Response.json({}));
    for (let attempt = 0; attempt < 20; attempt++) {
      await sleep(10);
      await host.updateComplete;
    }
    return {
      selectedId: host.selectedRun?.id,
      bEnabled,
      statusOnB,
      statusAfter: host.textContent ?? '',
      buttonDisabledAfter: button()?.disabled ?? null,
      resolved,
    };
  });
  assert.equal(outcome.selectedId, 'b'.repeat(32));
  assert.equal(outcome.bEnabled, true);
  assert.equal(outcome.statusOnB.includes('Current required context selected'), false);
  assert.equal(outcome.statusAfter.includes('Current required context selected'), false);
  assert.equal(outcome.buttonDisabledAfter, false);
  assert.equal(outcome.resolved, 0);
});

test('same-issue project refresh during a delayed use-current unbinds the stale action', async (t) => {
  // True Project Work regression (FJ-450): the Refresh plans click is an
  // ordinary UI click on the real soda-project-work element — never a
  // direct child refresh() call. A delayed issue-based Use-current POST is
  // pending when the refresh lands a changed same-issue selection; the old
  // busy must clear, the new Use-current must enable and bind the new
  // digests, and the original server op is preserved (no retry, no
  // command-status readback once its reply finally arrives).
  const page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  // Same-origin blank document (no fixture bundle): a secure http origin
  // keeps crypto.randomUUID available while avoiding double custom-element
  // registration with the fixture shell.
  await page.goto(new URL('/blank-450', server.url.href).href);
  const bundle = await Bun.build({
    entrypoints: [path.join(root, 'frontend/spaces/sodaspaces-project-work.ts')],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(bundle.success, `Project work bundle build failed: ${bundle.logs.join('\n')}`);
  const [entry] = bundle.outputs;
  assert(entry);
  await page.addScriptTag({content: await entry.text(), type: 'module'});
  await page.waitForFunction(() => customElements.get('soda-project-work') !== undefined);
  const outcome = await page.evaluate(async () => {
    const RUN_A = 'a'.repeat(32);
    const DIGEST_A = 'a'.repeat(64);
    const DIGEST_B = 'b'.repeat(64);
    const posted: {command_id?: unknown; context?: unknown}[] = [];
    let commandReads = 0;
    const state = {refreshed: false, releaseOld: undefined as undefined | (() => void), resolved: 0};
    const selection = (digest: string) => ({
      fingerprint: digest,
      scope_digest: digest,
      required_digest: digest,
      availability: 'complete',
      reason: 'required context is complete',
      action: 'hold',
      can_use_current: true,
    });
    const transport = {
      request: async (pathStr: string, init?: RequestInit) => {
        const method = init?.method ?? 'GET';
        if (method === 'GET' && pathStr.startsWith('factory/intents')) return Response.json({intents: []});
        if (pathStr === 'repositories/7/factory/issues/12')
          return Response.json({journey: {lineage: {latest_run: RUN_A}}});
        if (method === 'GET' && pathStr === `factory/runs/${RUN_A}`)
          return Response.json({
            run: {
              id: RUN_A,
              project_id: 'p0123456789abcdef01234567',
              role: 'soda-coder',
              harness: 'codex-fixture',
              model: 'fixture',
              input_sha: 'a'.repeat(40),
              reconciled: false,
            },
            context_selection: selection(state.refreshed ? DIGEST_B : DIGEST_A),
          });
        if (pathStr === `factory/runs/${RUN_A}/actions`) {
          const body = JSON.parse(String(init?.body ?? '{}')) as {command_id?: unknown; context?: unknown};
          posted.push(body);
          if (posted.length === 1) await new Promise<void>((resolve) => void (state.releaseOld = resolve));
          const context = body.context as {fingerprint?: unknown; scope_digest?: unknown; required_digest?: unknown};
          return Response.json({
            command_id: body.command_id,
            run_id: RUN_A,
            bundle: 'current',
            scope_digest: context?.scope_digest,
            fingerprint: context?.fingerprint,
            required_digest: context?.required_digest,
          });
        }
        if (method === 'GET' && pathStr.startsWith('factory/commands/')) {
          commandReads += 1;
          return Response.json({}, {status: 404});
        }
        return Response.json({}, {status: 404});
      },
    };
    type Host = HTMLElement & {context: unknown; active: boolean; surface: string; updateComplete: Promise<unknown>};
    const host = document.createElement('soda-project-work') as unknown as Host;
    host.addEventListener('soda-context-resolved', () => {
      state.resolved += 1;
    });
    document.body.append(host);
    host.context = {
      transport,
      mode: 'project',
      repository: {id: '7', name: '/soda-tester/demo'},
      repositoryNames: new Map(),
    };
    host.active = true;
    host.surface = 'work';
    const sleep = (ms: number) => new Promise((done) => setTimeout(done, ms));
    const buttonByName = (name: string) =>
      ([...host.querySelectorAll('button')].find((entry) => entry.textContent?.trim() === name) ??
        null) as HTMLButtonElement | null;
    const childStatus = () =>
      host.querySelector('soda-factory-context-resolution')?.querySelector('p[role="status"]')?.textContent ?? '';
    for (let attempt = 0; attempt < 100 && !buttonByName('Open requirements review'); attempt++) {
      await sleep(10);
      await host.updateComplete;
    }
    const details = host.querySelector('details') as HTMLDetailsElement | null;
    const input = host.querySelector('input[inputmode="numeric"]') as HTMLInputElement | null;
    const opened = !!details && !!input && buttonByName('Open requirements review') !== null;
    if (!details || !input) {
      return {
        pendingStatus: '',
        opened,
        initiallyEnabled: false,
        enabledAfterRefresh: false,
        statusAfterRefresh: '',
        afterOld: {status: '', enabled: false, posts: posted.length, resolved: state.resolved, commandReads},
        successAfterNew: false,
        resolvedFinal: state.resolved,
        posts: [] as {command: unknown; context: unknown}[],
      };
    }
    details.open = true;
    input.value = '12';
    input.dispatchEvent(new Event('input', {bubbles: true, composed: true}));
    await host.updateComplete;
    // Ordinary UI clicks from here on: no direct child method calls.
    buttonByName('Open requirements review')?.click();
    let useCurrent: HTMLButtonElement | null = null;
    for (let attempt = 0; attempt < 100; attempt++) {
      await sleep(10);
      await host.updateComplete;
      useCurrent = buttonByName('Use current required context');
      if (useCurrent?.disabled === false) break;
    }
    const initiallyEnabled = useCurrent?.disabled === false;
    useCurrent?.click();
    for (let attempt = 0; attempt < 100 && childStatus() !== 'Recording your current-context choice…'; attempt++)
      await sleep(10);
    const pendingStatus = childStatus();
    // Same-issue inputs are corrected server-side; the genuine Project
    // Work refresh click re-reads the same issue and lands a changed
    // selection while the first POST is still delayed.
    state.refreshed = true;
    buttonByName('Refresh plans')?.click();
    let enabledAfterRefresh = false;
    for (let attempt = 0; attempt < 200; attempt++) {
      await sleep(10);
      await host.updateComplete;
      if (buttonByName('Use current required context')?.disabled === false) {
        enabledAfterRefresh = true;
        break;
      }
    }
    const statusAfterRefresh = childStatus();
    // The original server op finally replies; it must be ignored (never
    // rendered, never retried, no command-status readback).
    state.releaseOld?.();
    for (let attempt = 0; attempt < 50; attempt++) await sleep(10);
    await host.updateComplete;
    const afterOld = {
      status: childStatus(),
      enabled: buttonByName('Use current required context')?.disabled === false,
      posts: posted.length,
      resolved: state.resolved,
      commandReads,
    };
    buttonByName('Use current required context')?.click();
    let successAfterNew = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      await sleep(10);
      await host.updateComplete;
      if (childStatus().includes('Current required context selected. Factory will recheck this run before starting.')) {
        successAfterNew = true;
        break;
      }
    }
    return {
      pendingStatus,
      opened,
      initiallyEnabled,
      enabledAfterRefresh,
      statusAfterRefresh,
      afterOld,
      successAfterNew,
      resolvedFinal: state.resolved,
      posts: posted.map((body) => ({
        command: body.command_id,
        context: body.context,
      })),
    };
  });
  assert.equal(outcome.opened, true);
  assert.equal(outcome.initiallyEnabled, true);
  assert.equal(outcome.pendingStatus, 'Recording your current-context choice…');
  assert.equal(outcome.enabledAfterRefresh, true);
  assert.equal(outcome.statusAfterRefresh.includes('Recording your current-context choice'), false);
  assert.equal(outcome.statusAfterRefresh.includes('Current required context selected'), false);
  assert.equal(outcome.afterOld.posts, 1);
  assert.equal(outcome.afterOld.resolved, 0);
  assert.equal(outcome.afterOld.commandReads, 0);
  assert.equal(outcome.afterOld.enabled, true);
  assert.equal(outcome.afterOld.status.includes('Current required context selected'), false);
  assert.equal(outcome.successAfterNew, true);
  assert.equal(outcome.resolvedFinal, 1);
  assert.equal(outcome.posts.length, 2);
  assert.deepEqual(outcome.posts[0]?.context, {
    fingerprint: 'a'.repeat(64),
    scope_digest: 'a'.repeat(64),
    required_digest: 'a'.repeat(64),
  });
  assert.deepEqual(outcome.posts[1]?.context, {
    fingerprint: 'b'.repeat(64),
    scope_digest: 'b'.repeat(64),
    required_digest: 'b'.repeat(64),
  });
  assert.match(String(outcome.posts[0]?.command), /^[a-f0-9]{32}$/);
  assert.match(String(outcome.posts[1]?.command), /^[a-f0-9]{32}$/);
  assert.notEqual(outcome.posts[0]?.command, outcome.posts[1]?.command);
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

test('work journey dedups delivered evidence on unchanged rebind with preserved history', async (t) => {
  const successor = 'd' + 'a'.repeat(24);
  const acceptance = 'd' + 'b'.repeat(24);
  const completion = '9'.repeat(32);
  const publication = '8'.repeat(32);
  const manifest = '7'.repeat(64);
  const journey = {
    repository: '7',
    issue: '12',
    observed_unix: 1720000100,
    phases: [],
    tasks: [{key: 'task-plan', lane: 'a', issue: '12', state: 'complete'}],
    commits: [],
    findings: [],
    corrections: [],
    judgments: [],
    gates: [],
    adoptions: [
      {intent: INTENT, revision: REVISION, state: 'adopted', task: 'task-plan', issue_id: '12', acceptance},
      {intent: INTENT, revision: successor, state: 'adopted', task: 'task-plan', issue_id: '12', acceptance},
    ],
    delivered_tasks: [{task: 'task-plan', issue: '12', acceptance, completion, publication, generation: 2}],
    verifications: [{intent: INTENT, revision: REVISION, source_commit: HEAD, manifest}],
  };
  const page = await openView(t, 'detail', [
    {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: {...issueDetail, journey}},
    {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
  ]);
  const detail = page.locator('soda-factory-work-detail');
  const workJourney = detail.getByRole('region', {name: 'Work journey', exact: true});
  await workJourney.getByText('1 delivered tasks').waitFor();
  await workJourney.getByText('1 verified applications').waitFor();
  assert.equal(await workJourney.getByText('Task task-plan delivered on native issue #12').count(), 1);
  assert.equal(await workJourney.getByText(`Verified application revision ${REVISION.slice(0, 9)}`).count(), 1);
  await workJourney.getByText(`at revision ${REVISION.slice(0, 9)}`).waitFor();
  await workJourney.getByText(`at revision ${successor.slice(0, 9)}`).waitFor();
  assert.equal(await workCalls(page).then((calls) => calls.filter((call) => call.method === 'POST').length), 0);
  // Exact evidence values render as text: the delivered row names the
  // canonical completion, publication and generation; the verified row
  // names the recorded source and manifest.
  await workJourney
    .getByText(
      `Task task-plan delivered on native issue #12 (acceptance ${acceptance.slice(0, 9)}): ` +
        `completion ${completion.slice(0, 9)} via publication ${publication.slice(0, 9)} (generation 2).`,
      {exact: true}
    )
    .waitFor();
  await workJourney
    .getByText(
      `Verified application revision ${REVISION.slice(0, 9)} of intent ${INTENT.slice(0, 9)}: ` +
        `trusted check passed at source ${HEAD.slice(0, 12)} (manifest ${manifest.slice(0, 12)}).`,
      {exact: true}
    )
    .waitFor();
  // Native navigation lives only on the "Open issue" anchors (one task
  // row plus two preserved adoption rows); every anchor names the
  // exact native issue URL, never an evidence identity.
  const issueLinks = workJourney.getByRole('link', {name: 'Open issue'});
  assert.equal(await issueLinks.count(), 3);
  for (let i = 0; i < 3; i++) {
    assert.equal(await issueLinks.nth(i).getAttribute('href'), '/soda-tester/demo/issues/12');
  }
  // Refresh re-reads through GET only: no POST, PUT, PATCH or DELETE
  // may fire while the journey reloads.
  await page.evaluate(
    (next) => window.factoryWork.setRoutes(next),
    [
      {method: 'GET', path: 'repositories/7/factory/issues/12', status: 200, json: {...issueDetail, journey}},
      {method: 'GET', path: 'repositories/7/factory/access', status: 200, json: access},
    ]
  );
  await page.locator('soda-factory-work-journey').getByRole('button', {name: 'Refresh journey', exact: true}).click();
  await page.waitForFunction(() => window.factoryWork.calls().length >= 1);
  await workJourney.getByText('1 delivered tasks').waitFor();
  const refreshCalls = await workCalls(page);
  assert.ok(refreshCalls.length >= 1);
  assert.ok(refreshCalls.some((call) => call.path === 'repositories/7/factory/issues/12'));
  assert.equal(refreshCalls.filter((call) => call.method !== 'GET').length, 0);
});

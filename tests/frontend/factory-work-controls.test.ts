import test from 'node:test';
import assert from 'node:assert/strict';
import {before, after, type TestContext} from 'node:test';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import {factoryAssetSource} from './fixtures/factory-asset-source.js';

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
  readonly readReplies?: unknown[];
  readonly echoCommand?: boolean;
  readonly echoAcceptance?: boolean;
  readonly loseReply?: boolean;
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
  assert.equal('command_id' in body, false);
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

test('unconfirmed acceptance repeats its exact decision', async (t) => {
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
  await standaloneDecisions();
  await standaloneAllowance();
  await standaloneRepair();
  await standaloneActivity();
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
  await standalonePresentation();
});

test('work detail repeats a completion request directly and runs all interventions', async (t) => {
  const contextRun = {
    run: {
      id: RUN,
      project_id: PROJECT,
      role: 'soda-coder',
      harness: 'codex-fixture',
      model: 'fixture',
      input_sha: HEAD,
      reconciled: false,
    },
    view: {repository: '7', issue: '12'},
    context_selection: {
      fingerprint: SNAPSHOT,
      scope_digest: SNAPSHOT,
      required_digest: SNAPSHOT,
      availability: 'complete',
      reason: 'context_selection_attention',
      action: 'hold',
      can_use_current: true,
    },
  };
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
      json: contextRun,
    },
    {
      method: 'POST',
      path: `factory/runs/${RUN}/actions`,
      action: 'use_current_context',
      status: 200,
      json: {
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
      json: {
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
      json: {
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
      json: {
        run: RUN,
        member: 'soda-tester',
        project: PROJECT,
        destination: '/home/soda-tester/takeover/d',
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
    .getByText('Completion review retry queued. Factory will reassess the recorded result in a new bounded attempt.', {
      exact: true,
    })
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
  assert.equal((await workCalls(page)).filter((call) => call.path.startsWith('factory/commands/')).length, 0);
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
  assert.deepEqual(pause?.body, {action: 'pause'});
  const resume = posts.find((call) => (call.body as {action?: string}).action === 'resume');
  assert.deepEqual(resume?.body, {action: 'resume'});
  assert.equal(pause?.path, 'repositories/7/factory/actions');
  const stop = posts.find((call) => (call.body as {action?: string}).action === 'stop');
  assert.equal(stop?.path, `factory/runs/${RUN}/actions`);
  assert.deepEqual(stop?.body, {action: 'stop'});
  const takeover = posts.find((call) => (call.body as {action?: string}).action === 'takeover');
  assert.deepEqual(takeover?.body, {action: 'takeover'});
  const projectStop = posts.find((call) => call.path === `environments/${ENVIRONMENT}/lifecycle`);
  assert.deepEqual(projectStop?.body, {action: 'stop', confirm_stop: true});

  await page.evaluate(() =>
    window.factoryWork.setRoutes([
      {
        method: 'POST',
        path: 'repositories/7/factory/actions',
        action: 'pause',
        status: 200,
        json: {paused: true, withdrawal: {repository: '7', revision: 8}, runs: []},
        loseReply: true,
      },
    ])
  );
  await detail.getByRole('button', {name: 'Pause Factory work'}).click();
  await detail.getByText('No reply was confirmed. Refresh before acting again.').waitFor();
  const afterLostPause = await workCalls(page);
  assert.equal(afterLostPause.filter((call) => call.path.startsWith('factory/commands/')).length, 0);
  assert.deepEqual(afterLostPause.at(-1)?.body, {action: 'pause'});

  await page.evaluate(
    (run) =>
      window.factoryWork.setRoutes([
        {
          method: 'POST',
          path: `factory/runs/${run}/actions`,
          action: 'stop',
          status: 200,
          json: {run_id: run, confirmed: true, uncertain: false},
          loseReply: true,
        },
      ]),
    RUN
  );
  await detail.getByRole('button', {name: 'Stop run'}).click();
  await detail
    .getByRole('region', {name: 'Collaborator controls'})
    .getByText('No reply was confirmed. Refresh before acting again.')
    .waitFor();
  const afterLostStop = await workCalls(page);
  assert.equal(afterLostStop.filter((call) => call.path.startsWith('factory/commands/')).length, 0);
  assert.deepEqual(afterLostStop.at(-1)?.body, {action: 'stop'});

  await page.evaluate(
    (run) =>
      window.factoryWork.setRoutes([
        {
          method: 'POST',
          path: `factory/runs/${run}/actions`,
          action: 'takeover',
          status: 200,
          json: {run, member: 'soda-tester', destination: '/home/soda-tester/takeover/d'},
          loseReply: true,
        },
      ]),
    RUN
  );
  await detail.getByRole('button', {name: 'Take over run'}).click();
  await detail
    .getByRole('listitem')
    .filter({has: page.getByRole('button', {name: 'Take over run', exact: true})})
    .getByText('No reply was confirmed. Refresh before acting again.')
    .waitFor();
  const afterLostTakeover = await workCalls(page);
  assert.equal(afterLostTakeover.filter((call) => call.path.startsWith('factory/commands/')).length, 0);
  assert.deepEqual(afterLostTakeover.at(-1)?.body, {action: 'takeover'});

  await page.evaluate(
    (run) =>
      window.factoryWork.setRoutes([
        {
          method: 'POST',
          path: `factory/runs/${run}/actions`,
          action: 'retry',
          status: 200,
          json: {queued: true},
          loseReply: true,
        },
      ]),
    RUN
  );
  await detail.getByRole('button', {name: 'Retry run'}).click();
  await detail
    .getByRole('listitem')
    .filter({has: page.getByRole('button', {name: 'Retry run', exact: true})})
    .getByText('No reply was confirmed. Retry the same request or refresh.')
    .waitFor();
  const afterLostRetry = await workCalls(page);
  assert.equal(afterLostRetry.filter((call) => call.method === 'POST').length, 1);
  assert.equal(afterLostRetry.filter((call) => call.path.startsWith('factory/commands/')).length, 0);

  await page.evaluate(
    ([run, runJSON]) =>
      window.factoryWork.setRoutes([
        {method: 'GET', path: `factory/runs/${run}`, status: 200, json: runJSON},
        {
          method: 'POST',
          path: `factory/runs/${run}/actions`,
          action: 'use_current_context',
          status: 200,
          json: {run_id: run, bundle: 'current'},
          loseReply: true,
        },
      ]),
    [RUN, contextRun] as const
  );
  await contextChoice.getByRole('button', {name: 'Use current required context', exact: true}).click();
  await contextChoice
    .getByText('No result was confirmed. Retry this same choice or refresh the work detail.', {exact: true})
    .waitFor();
  const afterLostChoice = await workCalls(page);
  assert.equal(afterLostChoice.filter((call) => call.path.startsWith('factory/commands/')).length, 0);
  assert.equal('command_id' in (afterLostChoice.at(-1)?.body as Record<string, unknown>), false);
  await standaloneDetail();
  await standaloneVerification();
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
    view: {repository: '7', issue: '12'},
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
      {method: 'GET', path: `factory/runs/${RUN}`, status: 200, json: runPayload(undefined), readReplies: [runPayload(refreshed)]},
      {
        method: 'POST',
        path: `factory/runs/${RUN}/actions`,
        action: 'use_current_context',
        status: 200,
        json: {
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
      generation: 'declared-context-deferred',
      assetBase: '/assets',
      request: (path: string) => new Promise<Response>((resolve) => void pending.push({path, resolve})),
    };
    const serving = {
      generation: 'declared-context-serving',
      assetBase: '/assets',
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
          view: {repository: '7', issue: '12'},
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
    host.context = {transport: deferred, commandScope: async () => 'declared-profile:provider:account', repository: '7', issue: '12'};
    await host.updateComplete;
    const firstPaths = pending.map((entry) => entry.path);
    // Supersede the deferred lookup with a newer transport before the stale read lands.
    host.context = {transport: serving, commandScope: async () => 'declared-profile:provider:account', repository: '7', issue: '12'};
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
  // Retired issue responses cannot start the next run read.
  assert.equal(outcome.hadStaleRun, false);
  assert.deepEqual(outcome.leftoverPaths, []);
  assert.equal(outcome.selectedId, 'b'.repeat(32));
  assert.equal(outcome.buttonDisabled, false);
  await standaloneResolution();
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
  await standaloneProgress();
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
      text('soda-factory-direction-revision') === ''
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
      generation: 'declared-context-generation',
      assetBase: '/assets',
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
            view: {repository: '7', issue: '12'},
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
    host.context = {transport, commandScope: async () => 'declared-profile:provider:account', repository: '7', issue: '12'};
    await host.updateComplete;
    for (let attempt = 0; attempt < 100 && runs === 0; attempt++) await new Promise((done) => setTimeout(done, 10));
    await host.updateComplete;
    for (let attempt = 0; attempt < 100 && !host.querySelector('button'); attempt++)
      await new Promise((done) => setTimeout(done, 10));
    await host.updateComplete;
    const disabledBefore = (host.querySelector('button') as HTMLButtonElement | null)?.disabled ?? null;
    const readsBefore = runs;
    // Same-issue re-entry without an explicit refresh skips its read.
    host.context = {transport, commandScope: async () => 'declared-profile:provider:account', repository: '7', issue: '12'};
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
      view: {repository: '7', issue: id === RUN_A ? '12' : '13'},
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
      generation: 'declared-context-generation',
      assetBase: '/assets',
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
          return Response.json({
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
    host.context = {transport, commandScope: async () => 'declared-profile:provider:account', repository: '7', issue: '12'};
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
    host.context = {transport, commandScope: async () => 'declared-profile:provider:account', repository: '7', issue: '13'};
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
      generation: 'declared-project-work-generation',
      assetBase: '/assets',
      request: async (pathStr: string, init?: RequestInit) => {
        const method = init?.method ?? 'GET';
        if (pathStr === 'factory/identity') return Response.json({host: 'a'.repeat(32), actor: '1', native_actor: '1'});
        if (
          pathStr === 'factory/native-repositories/resolve' &&
          method === 'POST' &&
          JSON.parse(String(init?.body)).repository === '7'
        )
          return Response.json({
            host: 'a'.repeat(32),
            actor: '1',
            repository: '7',
            native_repository: '7',
            name: 'soda-tester/demo',
          });
        if (pathStr === 'repositories?q=') return Response.json({items: [], next_cursor: ''});
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
            view: {repository: '7', issue: '12'},
            context_selection: selection(state.refreshed ? DIGEST_B : DIGEST_A),
          });
        if (pathStr === `factory/runs/${RUN_A}/actions`) {
          const body = JSON.parse(String(init?.body ?? '{}')) as {command_id?: unknown; context?: unknown};
          posted.push(body);
          if (posted.length === 1) await new Promise<void>((resolve) => void (state.releaseOld = resolve));
          const context = body.context as {fingerprint?: unknown; scope_digest?: unknown; required_digest?: unknown};
          return Response.json({
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
        posts: [] as {hasCommand: boolean; context: unknown}[],
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
        hasCommand: 'command_id' in body,
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
  assert.equal(outcome.posts[0]?.hasCommand, false);
  assert.equal(outcome.posts[1]?.hasCommand, false);
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
  await standaloneJourney();
});

async function standaloneJourney() {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-journey-fixture.ts')],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Standalone journey build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const pathname = new URL(request.url).pathname;
      if (pathname === '/journey.js')
        return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      const asset = factoryAssetSource(pathname);
      if (asset) return asset;
      if (pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/journey.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400)
      errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`);
  });
  try {
    await page.goto(host.url.href);
    const journey = page.locator('soda-factory-work-journey');
    await journey.getByRole('region', {name: 'Work journey', exact: true}).waitFor();
    assert.equal(
      await journey.getByRole('link', {name: 'Open issue', exact: true}).getAttribute('href'),
      '/desktop/repositories/9007199254740993/issues/12'
    );
    assert.equal(
      await journey.getByRole('link', {name: 'Open pull request', exact: true}).getAttribute('href'),
      '/desktop/repositories/9007199254740993/pulls/21'
    );
    const calls = await page.evaluate(() => window.factoryJourneyFixture.calls());
    await page.evaluate(() => window.factoryJourneyFixture.rebindLinks());
    await page.waitForFunction(
      () => document.querySelector('a')?.getAttribute('href') === '/another-host-route/repositories/9007199254740993/issues/12'
    );
    assert.equal(await page.evaluate(() => window.factoryJourneyFixture.calls()), calls);
    await journey.getByRole('button', {name: 'Refresh journey', exact: true}).click();
    await page.waitForFunction((before) => window.factoryJourneyFixture.calls() === before + 1, calls);
    await journey.getByRole('region', {name: 'Work journey', exact: true}).waitFor();
    assert.equal(await journey.getByRole('link', {name: 'Open issue', exact: true}).count(), 1);
    await page.evaluate(() => window.factoryJourneyFixture.failRefresh(503));
    await journey.getByRole('button', {name: 'Refresh journey', exact: true}).click();
    await journey.getByText('Could not refresh. The last loaded work history is still shown.').waitFor();
    assert.equal(await journey.getByRole('region', {name: 'Work journey', exact: true}).count(), 1);
    await page.evaluate(() => window.factoryJourneyFixture.failRefresh(403));
    await journey.getByRole('button', {name: 'Refresh journey', exact: true}).click();
    await journey.getByText('Could not load the work journey. Refresh to try again.').waitFor();
    assert.equal(await journey.getByRole('region', {name: 'Work journey', exact: true}).count(), 0);
    assert.equal(await journey.getByRole('link').count(), 0);
    const background = () => page.locator('main').evaluate((node) => getComputedStyle(node).backgroundColor);
    assert.equal(await background(), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate((node) => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await background(), 'rgb(16, 16, 16)');
    await page.evaluate(() => window.factoryJourneyFixture.mount('empty'));
    await journey.getByText('No work journey has been recorded.').waitFor();
    for (const state of ['unavailable', 'denied'] as const) {
      await page.evaluate((state) => window.factoryJourneyFixture.mount(state), state);
      await journey.getByText('Could not load the work journey. Refresh to try again.').waitFor();
      assert.equal(await journey.getByRole('region', {name: 'Work journey', exact: true}).count(), 0);
    }
    await page.evaluate(() => window.factoryJourneyFixture.hold());
    await journey.getByText('Loading work history…').waitFor();
    await page.evaluate(() => window.factoryJourneyFixture.mount('empty'));
    assert.equal(await page.evaluate(() => window.factoryJourneyFixture.heldAborted()), true);
    await page.evaluate(() => window.factoryJourneyFixture.release());
    await journey.getByText('No work journey has been recorded.').waitFor();
    assert.equal(await journey.getByRole('link').count(), 0);
    await page.evaluate(() => window.factoryJourneyFixture.hold());
    await journey.getByText('Loading work history…').waitFor();
    await page.evaluate(() => {
      window.factoryJourneyFixture.retireGeneration();
      window.factoryJourneyFixture.release();
    });
    await page.waitForFunction(() => document.querySelector('soda-factory-work-journey')?.textContent === '');
    assert.equal(await page.evaluate(() => window.factoryJourneyFixture.heldAborted()), true);
    await page.evaluate(() => window.factoryJourneyFixture.mount('recorded'));
    await journey.getByRole('region', {name: 'Work journey', exact: true}).waitFor();
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
    host.stop(true);
  }
}

async function completionRetryRecovery(page: Page) {
  const retry = page.locator('soda-factory-completion-retry');
  const button = () => retry.getByRole('button', {name: 'Retry completion review', exact: true});
  const savedButton = () => retry.getByRole('button', {name: 'Retry saved completion review', exact: true});
  const refresh = () => retry.getByRole('button', {name: 'Refresh completion reviews', exact: true});
  const posts = () => page.evaluate(() => window.factoryDecisionsFixture.calls().filter(call => call.method === 'POST'));
  const status = (message: string) => retry.getByRole('status').filter({hasText: message}).waitFor();
  const recovered = () => status('An unconfirmed completion review retry is saved.');
  await page.evaluate(() => window.factoryDecisionsFixture.mount('unknown'));
  await button().click();
  await status('No retry was confirmed.');
  const first = (await posts())[0]!;
  const original = await page.evaluate(() => window.factoryDecisionsFixture.saved());
  assert(original);
  const record = JSON.parse(original) as {scope: string; repository: string; issue: string; command: string; event: string; cycle: string};
  assert.deepEqual(first.body, {command_id: record.command, action: 'retry_completion', completion: {
    event_id: record.event, expected_prior_cycle: record.cycle,
  }});
  assert.equal(record.repository, '9007199254740993');
  assert.equal(record.issue, '12');
  await page.evaluate(() => window.factoryDecisionsFixture.remount());
  await recovered();
  assert.equal((await posts()).length, 0);
  await page.reload();
  await recovered();
  assert.equal((await posts()).length, 0);
  await button().click();
  await status('Completion review retry queued.');
  assert.deepEqual((await posts())[0], first);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), null);

  // Recovery is still offered when a newer attention gate replaces the saved
  // cycle, or the bounded history no longer includes any of its gates.
  for (const omit of [false, true]) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.remount();}, original);
    await recovered();
    await page.evaluate(omit => window.factoryDecisionsFixture.advanceGate(omit), omit);
    await refresh().click();
    await savedButton().waitFor();
    if (!omit) assert.equal(await button().isDisabled(), true);
    assert.equal((await posts()).length, 0);
    await savedButton().click();
    await status('Completion review retry queued.');
    assert.deepEqual((await posts())[0], first);
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), null);
  }
  // Neither a 2xx with another decision nor a current replay refusal proves
  // that the original request never applied. Only its exact receipt settles it.
  for (const patch of [
    {command_id: 'e'.repeat(32)}, {event_id: 'e'.repeat(32)}, {prior_cycle: 'e'.repeat(32)},
    {successor_cycle: 'e'.repeat(32)}, {repository: '9007199254740992'}, {queued: false},
  ]) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.remount();}, original);
    await recovered();
    await page.evaluate(patch => window.factoryDecisionsFixture.receipt(patch), patch);
    await button().click();
    await status('No retry was confirmed.');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), original);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
  }
  await page.evaluate(() => window.factoryDecisionsFixture.receipt({}, 403));
  await button().click();
  await status('No retry was confirmed.');
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), original);
  assert.deepEqual((await posts())[1], first);
  await page.evaluate(() => window.factoryDecisionsFixture.receipt({}));
  await button().click();
  await status('Completion review retry queued.');

  for (const corrupt of ['{broken', JSON.stringify({...record, issue: '13'}), JSON.stringify({...record, scope: 'another-account'}), JSON.stringify({...record, extra: true})]) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.remount();}, corrupt);
    await status('Saved completion retry is unreadable');
    assert.equal(await button().isDisabled(), true);
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), corrupt);
    assert.equal((await posts()).length, 0);
  }
  await page.evaluate(() => {window.factoryDecisionsFixture.seed(null); window.factoryDecisionsFixture.storageFailure('read'); window.factoryDecisionsFixture.remount();});
  await status('Saved completion retry is unreadable');
  assert.equal(await button().isDisabled(), true);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('none'));
  await refresh().click();
  await page.waitForFunction(() => !(document.querySelector('soda-factory-completion-retry button') as HTMLButtonElement)?.disabled);
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('write'));
  await button().click();
  await status('Completion retry could not be saved.');
  const unsent = await page.evaluate(() => window.factoryDecisionsFixture.pending());
  assert(unsent);
  assert.equal((await posts()).length, 0);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), null);
  await page.evaluate(() => window.factoryDecisionsFixture.remount());
  await recovered();
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('none'));
  await button().click();
  await status('Completion review retry queued.');
  assert.equal(((await posts())[0]!.body as {command_id: string}).command_id, unsent.command);

  // Profile/account changes select different records, and an identity read
  // failure cannot create a new request in an assumed account.
  await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.account('B'); window.factoryDecisionsFixture.remount();}, original);
  await button().click();
  await status('Completion review retry queued.');
  assert.notEqual(((await posts())[0]!.body as {command_id: string}).command_id, record.command);
  await page.evaluate(() => {window.factoryDecisionsFixture.account('A'); window.factoryDecisionsFixture.remount();});
  await recovered();
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), original);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {window.factoryDecisionsFixture.identity(false); window.factoryDecisionsFixture.remount();});
  await status('Completion retry identity could not be read.');
  assert.equal(await button().isDisabled(), true);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryDecisionsFixture.identity(true));
  await refresh().click();
  await recovered();
  await page.evaluate(() => window.factoryDecisionsFixture.account('B'));
  await button().click();
  await status('The saved completion retry or account changed.');
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {window.factoryDecisionsFixture.account('A'); window.factoryDecisionsFixture.remount();});
  await recovered();

  // A late receipt must not erase or announce a different saved command.
  const replacement = JSON.stringify({...record, command: 'e'.repeat(32)});
  await page.evaluate(() => window.factoryDecisionsFixture.hold());
  await button().click();
  await page.waitForFunction(() => window.factoryDecisionsFixture.calls().some(call => call.method === 'POST'));
  await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.release();}, replacement);
  await status('The saved completion retry changed.');
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), replacement);
  assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
  assert.equal((await posts()).length, 1);
  await page.evaluate(() => window.factoryDecisionsFixture.remount());
  await recovered();
  assert.equal((await posts()).length, 0);
  await button().click();
  await status('Completion review retry queued.');
  assert.equal(((await posts())[0]!.body as {command_id: string}).command_id, 'e'.repeat(32));
  for (const retirement of ['context', 'generation']) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.remount();}, original);
    await recovered();
    await page.evaluate(() => window.factoryDecisionsFixture.hold());
    await button().click();
    await page.waitForFunction(() => window.factoryDecisionsFixture.calls().some(call => call.method === 'POST'));
    await page.evaluate(retirement => {
      if (retirement === 'context') window.factoryDecisionsFixture.retire();
      else window.factoryDecisionsFixture.retireGeneration();
      window.factoryDecisionsFixture.release();
    }, retirement);
    await page.waitForFunction(() => document.querySelector('soda-factory-completion-retry')?.textContent === '');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.aborted()), true);
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), original);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
    await page.evaluate(() => window.factoryDecisionsFixture.remount());
    await recovered();
    assert.equal((await posts()).length, 0);
    await button().click();
    await status('Completion review retry queued.');
    assert.deepEqual((await posts())[0], first);
  }
  // A failed receipt-removal retains a document tombstone; a subsequent real
  // reload may recover the durable old request and safely replay that exact ID.
  await page.evaluate(value => {window.factoryDecisionsFixture.seed(value); window.factoryDecisionsFixture.remount();}, original);
  await recovered();
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('remove'));
  await button().click();
  await status('Completion review retry queued.');
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), original);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.pending()), undefined);
  await page.evaluate(() => {window.factoryDecisionsFixture.storageFailure('none'); window.factoryDecisionsFixture.remount();});
  await page.waitForFunction(() => !(document.querySelector('soda-factory-completion-retry button') as HTMLButtonElement)?.disabled);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.pending()), undefined);
  assert.equal((await posts()).length, 0);
  await page.reload();
  await recovered();
  assert.equal((await posts()).length, 0);
  await button().click();
  await status('Completion review retry queued.');
  assert.deepEqual((await posts())[0], first);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.saved()), null);
  // A genuinely newer attention cycle can start its own bounded request once
  // the earlier receipt has settled, without remounting the control.
  await page.evaluate(() => window.factoryDecisionsFixture.advanceGate());
  await refresh().click();
  await button().click();
  await status('Completion review retry queued.');
  const next = (await posts())[1]!.body as {command_id: string; completion: {event_id: string; expected_prior_cycle: string}};
  assert.notEqual(next.command_id, record.command);
  assert.deepEqual(next.completion, {event_id: 'e'.repeat(32), expected_prior_cycle: 'f'.repeat(32)});
}

async function acceptanceRecovery(page: Page) {
  const element = page.locator('soda-factory-requirements-acceptance');
  const retry = () => element.getByRole('button', {name: 'Retry same acceptance', exact: true});
  const review = () => element.getByRole('button', {name: 'Review current requirements', exact: true});
  const refresh = () => element.getByRole('button', {name: 'Refresh acceptance authority', exact: true});
  const status = (message: string) => element.getByRole('status').filter({hasText: message}).waitFor();
  const recovered = () => status('An unconfirmed acceptance is saved.');
  const posts = () => page.evaluate(() => window.factoryDecisionsFixture.calls().filter(call => call.method === 'POST'));
  await page.evaluate(() => {window.factoryDecisionsFixture.mount('unknown'); window.factoryDecisionsFixture.fullReview();});
  await element.getByLabel('Requirement comment IDs', {exact: true}).fill('21');
  await element.getByLabel('Resolution comment IDs', {exact: true}).fill('22');
  await review().click();
  await element.getByLabel('Prerequisite repository ID', {exact: true}).fill('9007199254740993');
  await element.getByLabel('Prerequisite issue number', {exact: true}).fill('13');
  await element.getByLabel('Required outcome', {exact: true}).selectOption('code');
  await element.getByRole('checkbox').check();
  await element.getByRole('button', {name: 'Accept revised requirements', exact: true}).click();
  await status('Retry the same acceptance');
  const first = (await posts())[0]!;
  const original = await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance());
  assert(original);
  const record = JSON.parse(original) as {scope: string; repository: string; issue: string; decision: Record<string, unknown>};
  assert.deepEqual(record.decision, first.body);
  assert.equal(record.decision.predecessor, 'd' + '1'.repeat(24));
  assert.deepEqual(record.decision.sources, [{id: '21', digest: 'c'.repeat(64), content_version: 2}]);
  assert.deepEqual(record.decision.resolutions, [{id: '22', digest: 'c'.repeat(64), content_version: 2}]);
  assert.deepEqual(record.decision.prerequisites, [{occurrence: '23', depends_on: '90', endpoint_repository: '9007199254740993', endpoint_issue: '13', outcome: 'code', prerequisite_acceptance: 'd' + '2'.repeat(24)}]);
  await page.evaluate(() => window.factoryDecisionsFixture.receipt({error: 'objective_changed', message: 'Declared later refusal'}, 409));
  await retry().click();
  await status('Declared later refusal');
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), original);
  assert.deepEqual((await posts())[1], first);
  await page.evaluate(() => window.factoryDecisionsFixture.remount());
  await recovered();
  assert.equal((await posts()).length, 0);
  assert.equal(await element.getByLabel('Requirement comment IDs', {exact: true}).inputValue(), '21');
  assert.equal(await element.getByLabel('Resolution comment IDs', {exact: true}).inputValue(), '22');
  await page.reload();
  await recovered();
  assert.equal((await posts()).length, 0);
  assert.equal(await review().isDisabled(), true);
  await element.getByText(/Saved acceptance: body revision 3; native revision\s+41/).waitFor();
  // A later current head is allowed: the receipt still confirms this exact old
  // decision, without approving a newer preview or rewriting its prerequisites.
  await page.evaluate(() => window.factoryDecisionsFixture.receipt({head: 'd' + '4'.repeat(24), depth: 4}));
  await retry().click();
  await status('Requirements accepted.');
  assert.deepEqual((await posts())[0], first);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), null);
  for (const patch of [{decision_id: 'd' + '5'.repeat(24)}, {head: 'invalid'}, {depth: 0}, {depth: 1.5}]) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seedAcceptance(value); window.factoryDecisionsFixture.remount();}, original);
    await recovered();
    await page.evaluate(patch => window.factoryDecisionsFixture.receipt(patch), patch);
    await retry().click();
    await status('Retry the same acceptance');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), original);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
  }
  // A refusal after recovery must retain the unknown prior decision, including
  // evidence errors that would require a new review on the first known attempt.
  for (const code of [403, 409]) {
    await page.evaluate(code => window.factoryDecisionsFixture.receipt({error: 'objective_changed', message: 'Declared current refusal'}, code), code);
    await retry().click();
    await status('Declared current refusal');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), original);
    assert.deepEqual((await posts()).at(-1), first);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
  }
  await page.evaluate(() => window.factoryDecisionsFixture.receipt({}));
  await retry().click();
  await status('Requirements accepted.');
  for (const corrupt of ['{broken', JSON.stringify({...record, issue: '13'}), JSON.stringify({...record, scope: 'another-account'}),
    JSON.stringify({...record, decision: {...record.decision, title_digest: ''}}),
    JSON.stringify({...record, decision: {...record.decision, sources: [{id: '21'}]}}),
    JSON.stringify({...record, decision: {...record.decision, extra: true}})]) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seedAcceptance(value); window.factoryDecisionsFixture.remount();}, corrupt);
    await status('Saved acceptance is unreadable');
    assert.equal(await review().isDisabled(), true);
    assert.equal((await posts()).length, 0);
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), corrupt);
  }
  await page.evaluate(() => {window.factoryDecisionsFixture.seedAcceptance(null); window.factoryDecisionsFixture.storageFailure('read'); window.factoryDecisionsFixture.remount();});
  await status('Saved acceptance is unreadable');
  assert.equal(await review().isDisabled(), true);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('none'));
  await refresh().click();
  await review().click();
  await element.getByRole('checkbox').check();
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('write'));
  await element.getByRole('button', {name: 'Accept requirements', exact: true}).click();
  await status('Acceptance could not be saved.');
  const unsent = await page.evaluate(() => window.factoryDecisionsFixture.pendingAcceptance());
  assert(unsent);
  assert.equal((await posts()).length, 0);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), null);
  await page.evaluate(() => window.factoryDecisionsFixture.remount());
  await recovered();
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('none'));
  await retry().click();
  await status('Requirements accepted.');
  assert.deepEqual((await posts())[0]!.body, unsent);

  await page.evaluate(value => {window.factoryDecisionsFixture.seedAcceptance(value); window.factoryDecisionsFixture.account('B'); window.factoryDecisionsFixture.remount();}, original);
  await review().click();
  await element.getByRole('checkbox').check();
  await element.getByRole('button', {name: 'Accept requirements', exact: true}).click();
  await status('Requirements accepted.');
  assert.notEqual(((await posts())[0]!.body as {decision_id: string}).decision_id, record.decision.decision_id);
  await page.evaluate(() => {window.factoryDecisionsFixture.account('A'); window.factoryDecisionsFixture.remount();});
  await recovered();
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), original);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {window.factoryDecisionsFixture.identity(false); window.factoryDecisionsFixture.remount();});
  await status('Acceptance identity could not be read.');
  assert.equal(await review().isDisabled(), true);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryDecisionsFixture.identity(true));
  await refresh().click();
  await recovered();
  await page.evaluate(() => window.factoryDecisionsFixture.account('B'));
  await retry().click();
  await status('The saved acceptance or account changed.');
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {window.factoryDecisionsFixture.account('A'); window.factoryDecisionsFixture.remount();});
  await recovered();

  const replacement = JSON.stringify({...record, decision: {...record.decision, decision_id: 'd' + '6'.repeat(24)}});
  await page.evaluate(() => window.factoryDecisionsFixture.hold());
  await retry().click();
  await page.waitForFunction(() => window.factoryDecisionsFixture.calls().some(call => call.method === 'POST'));
  await page.evaluate(value => {window.factoryDecisionsFixture.seedAcceptance(value); window.factoryDecisionsFixture.release();}, replacement);
  await status('The saved acceptance changed.');
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), replacement);
  assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
  await page.evaluate(() => window.factoryDecisionsFixture.remount());
  await recovered();
  await retry().click();
  await status('Requirements accepted.');
  assert.equal(((await posts())[0]!.body as {decision_id: string}).decision_id, 'd' + '6'.repeat(24));
  for (const retirement of ['context', 'generation']) {
    await page.evaluate(value => {window.factoryDecisionsFixture.seedAcceptance(value); window.factoryDecisionsFixture.remount();}, original);
    await recovered();
    await page.evaluate(() => window.factoryDecisionsFixture.hold());
    await retry().click();
    await page.waitForFunction(() => window.factoryDecisionsFixture.calls().some(call => call.method === 'POST'));
    await page.evaluate(retirement => {
      if (retirement === 'context') window.factoryDecisionsFixture.retire();
      else window.factoryDecisionsFixture.retireGeneration();
      window.factoryDecisionsFixture.release();
    }, retirement);
    await page.waitForFunction(() => document.querySelector('soda-factory-requirements-acceptance')?.textContent === '');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.aborted()), true);
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), original);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
    await page.evaluate(() => window.factoryDecisionsFixture.remount());
    await recovered();
    assert.equal((await posts()).length, 0);
    await retry().click();
    await status('Requirements accepted.');
    assert.deepEqual((await posts())[0], first);
  }
  await page.evaluate(value => {window.factoryDecisionsFixture.seedAcceptance(value); window.factoryDecisionsFixture.remount();}, original);
  await recovered();
  await page.evaluate(() => window.factoryDecisionsFixture.storageFailure('remove'));
  await retry().click();
  await status('Requirements accepted.');
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), original);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.pendingAcceptance()), undefined);
  await page.evaluate(() => {window.factoryDecisionsFixture.storageFailure('none'); window.factoryDecisionsFixture.remount();});
  await review().waitFor();
  assert.equal(await retry().count(), 0);
  assert.equal((await posts()).length, 0);
  await page.reload();
  await recovered();
  assert.equal((await posts()).length, 0);
  await retry().click();
  await status('Requirements accepted.');
  assert.deepEqual((await posts())[0], first);
  assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.savedAcceptance()), null);
}

async function standaloneAllowance() {
  const build = await Bun.build({entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-allowance-fixture.ts')], target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'}});
  assert(build.success, `Standalone allowance build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/allowance.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    const asset = factoryAssetSource(pathname);
    if (asset) return asset;
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/allowance.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400) errors.push(`Standalone allowance asset failed: ${response.url()}`);
  });
  try {

    await page.goto(host.url.href);
    const element = page.locator('soda-factory-verification-allowance');
    const input = () => element.getByLabel('Total verification runs', {exact: true});
    const save = () => element.getByRole('button', {name: 'Increase verification limits', exact: true});
    const retry = () => element.getByRole('button', {name: 'Retry same limit change', exact: true});
    const refresh = () => element.getByRole('button', {name: 'Refresh verification limits', exact: true});
    const status = (text: string) => element.getByRole('status').filter({hasText: text}).waitFor();
    const posts = () => page.evaluate(() => window.factoryAllowanceFixture.calls().filter(call => call.method === 'POST'));
    const recovered = () => status('An unconfirmed verification limit change is saved.');
    await input().waitFor(); await save().click(); await status('Increase at least one limit');
    assert.equal((await posts()).length, 0);
    await page.evaluate(() => window.factoryAllowanceFixture.mount('unknown'));
    await input().fill('3'); await save().click(); await status('The current limits were not confirmed.');
    const first = (await posts())[0]!;
    const original = await page.evaluate(() => window.factoryAllowanceFixture.saved()); assert(original);
    const record = JSON.parse(original) as {scope: string; repository: string; intent: string; change: {command_id: string; expected_revision: string; limits: {runs: number; active_minutes: number}}};
    assert.deepEqual(first.body, record.change); assert.equal(record.repository, '9007199254740993');
    assert.equal(record.change.expected_revision, ''); assert.deepEqual(record.change.limits, {runs: 3, active_minutes: 60});
    await page.evaluate(() => {window.factoryAllowanceFixture.rebind(); window.factoryAllowanceFixture.advance();});
    await refresh().click(); await recovered();
    await element.getByText('2 of 4 verification runs used; 148 active seconds used of 60 minutes.', {exact: true}).waitFor();
    await retry().click(); await status('Verification limits saved.');
    assert.deepEqual((await posts())[1], first); assert.equal(await input().inputValue(), '4');
    await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.remount();}, original);
    await recovered(); assert.equal((await posts()).length, 0);
    await page.reload(); await recovered(); assert.equal((await posts()).length, 0); assert.equal(await input().isDisabled(), true);
    await retry().click(); await status('Verification limits saved.'); assert.deepEqual((await posts())[0], first);
    for (const patch of [{command_id: 'f'.repeat(32)}, {intent: 'd' + '2'.repeat(24)}, {revision: 'invalid'}, {limits: {runs: 2, active_minutes: 60}}]) {
      await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.remount();}, original);
      await recovered(); await page.evaluate(patch => window.factoryAllowanceFixture.receipt(patch), patch);
      await retry().click(); await status('The current limits were not confirmed.');
      assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), original);
      await page.evaluate(() => window.factoryAllowanceFixture.receipt({})); await retry().click(); await status('Verification limits saved.');
      assert.deepEqual((await posts())[0], (await posts())[1]);
    }
    await page.evaluate(() => window.factoryAllowanceFixture.mount('stale'));
    await input().fill('3'); await save().click(); await status('Verification limits changed.');
    assert.equal(await retry().count(), 0); assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), null);
    await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.mount('stale', false, true);}, original);
    await recovered(); await retry().click(); await status('The current limits were not confirmed.');
    assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), original);
    for (const bad of ['{broken', JSON.stringify({...record, repository: '1'}), JSON.stringify({...record, scope: 'foreign'}), JSON.stringify({...record, change: {...record.change, limits: {runs: 0, active_minutes: 60}}})]) {
      await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.remount();}, bad);
      await status('Saved verification limit change is unreadable'); assert.equal(await input().isDisabled(), true);
      assert.equal((await posts()).length, 0); assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), bad);
    }
    await page.evaluate(() => {window.factoryAllowanceFixture.seed(null); window.factoryAllowanceFixture.storageFailure('read'); window.factoryAllowanceFixture.remount();});
    await status('Saved verification limit change is unreadable'); assert.equal((await posts()).length, 0);
    await page.evaluate(() => window.factoryAllowanceFixture.storageFailure('none')); await refresh().click();
    await input().fill('3'); await page.evaluate(() => window.factoryAllowanceFixture.storageFailure('write'));
    await save().click(); await status('Verification limit change could not be saved.');
    const unsent = await page.evaluate(() => window.factoryAllowanceFixture.pending()); assert(unsent); assert.equal((await posts()).length, 0);
    await page.evaluate(() => window.factoryAllowanceFixture.remount()); await recovered();
    await page.evaluate(() => window.factoryAllowanceFixture.storageFailure('none')); await retry().click(); await status('Verification limits saved.');
    assert.deepEqual((await posts())[0]!.body, unsent);
    await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.account('B'); window.factoryAllowanceFixture.remount();}, original);
    await input().fill('4'); await save().click(); await status('Verification limits saved.');
    assert.notEqual(((await posts())[0]!.body as {command_id: string}).command_id, record.change.command_id);
    await page.evaluate(() => {window.factoryAllowanceFixture.account('A'); window.factoryAllowanceFixture.remount();});
    await recovered(); assert.equal((await posts()).length, 0); await page.evaluate(() => window.factoryAllowanceFixture.account('B'));
    await retry().click(); await status('The saved limit change or account changed.'); assert.equal((await posts()).length, 0);
    await page.evaluate(() => {window.factoryAllowanceFixture.account('A'); window.factoryAllowanceFixture.identity(false); window.factoryAllowanceFixture.remount();});
    await status('Allowance identity or request scope could not be read.'); assert.equal((await posts()).length, 0);
    await page.evaluate(() => window.factoryAllowanceFixture.identity(true)); await refresh().click(); await recovered();
    await page.evaluate(() => window.factoryAllowanceFixture.scope('1')); await refresh().click();
    await status('Allowance identity or request scope could not be read.'); assert.equal(await retry().isDisabled(), true);
    for (const retirement of ['context', 'generation', 'disconnect']) {
      await page.evaluate(() => window.factoryAllowanceFixture.mount('ready', true));
      await page.waitForFunction(() => window.factoryAllowanceFixture.calls().length > 0);
      await page.evaluate(retirement => {
        if (retirement === 'context') window.factoryAllowanceFixture.retire(); else if (retirement === 'generation') window.factoryAllowanceFixture.retireGeneration(); else window.factoryAllowanceFixture.disconnect();
        window.factoryAllowanceFixture.release();
      }, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-verification-allowance')?.textContent);
      assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.aborted()), true);
      await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.remount();}, original);
      await recovered(); await page.evaluate(() => window.factoryAllowanceFixture.holdAction()); await retry().click();
      await page.waitForFunction(() => window.factoryAllowanceFixture.calls().some(call => call.method === 'POST'));
      await page.evaluate(retirement => {
        if (retirement === 'context') window.factoryAllowanceFixture.retire(); else if (retirement === 'generation') window.factoryAllowanceFixture.retireGeneration(); else window.factoryAllowanceFixture.disconnect();
        window.factoryAllowanceFixture.release();
      }, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-verification-allowance')?.textContent);
      assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), original);
      assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.aborted()), true);
    }
    const replacement = JSON.stringify({...record, change: {...record.change, command_id: 'f'.repeat(32)}});
    await page.evaluate(() => window.factoryAllowanceFixture.remount()); await recovered();
    await page.evaluate(() => window.factoryAllowanceFixture.holdAction()); await retry().click();
    await page.waitForFunction(() => window.factoryAllowanceFixture.calls().some(call => call.method === 'POST'));
    await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.release();}, replacement);
    await status('The saved limit change changed.'); assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), replacement);
    await page.evaluate(value => {window.factoryAllowanceFixture.seed(value); window.factoryAllowanceFixture.remount();}, original);
    await recovered(); await page.evaluate(() => window.factoryAllowanceFixture.storageFailure('remove')); await retry().click(); await status('Verification limits saved.');
    assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), original); assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.pending()), undefined);
    await page.evaluate(() => {window.factoryAllowanceFixture.storageFailure('none'); window.factoryAllowanceFixture.remount();});
    await save().waitFor(); assert.equal(await retry().count(), 0);
    await page.reload(); await recovered(); await retry().click(); await status('Verification limits saved.'); assert.deepEqual((await posts())[0], first);
    await page.evaluate(() => window.factoryAllowanceFixture.mount('unknown'));
    await input().fill('3'); await save().click(); await status('The current limits were not confirmed.');
    await refresh().click(); await status('Verification limits confirmed.'); assert.equal((await posts()).length, 1);
    assert.equal(await page.evaluate(() => window.factoryAllowanceFixture.saved()), null);
    await page.evaluate(() => window.factoryAllowanceFixture.mount('read-only'));
    await element.getByText('2 of 2 verification runs used; 148 active seconds used of 60 minutes.', {exact: true}).waitFor(); assert.equal(await save().count(), 0);
    for (const state of ['offline', 'denied'] as const) {
      await page.evaluate(state => window.factoryAllowanceFixture.mount(state), state); await status('Could not read verification limits.');
      assert.equal(await save().count(), 0); assert.equal((await posts()).length, 0);
    }
    await page.evaluate(() => document.fonts.ready.then(() => undefined)); assert.deepEqual(errors, []);
  } finally { await page.close(); host.stop(true); }
}

async function standaloneRepair() {
  const build = await Bun.build({entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-repair-fixture.ts')], target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'}});
  assert(build.success, `Standalone repair build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/repair.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    const asset = factoryAssetSource(pathname);
    if (asset) return asset;
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/repair.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400) errors.push(`Standalone repair asset failed: ${response.url()}`);
  });
  try {

    await page.goto(host.url.href);
    const element = page.locator('soda-factory-repair');
    const save = () => element.getByRole('button', {name: 'Save repair grant', exact: true});
    const retry = () => element.getByRole('button', {name: 'Retry same repair change', exact: true});
    const refresh = () => element.getByRole('button', {name: 'Refresh repair funding', exact: true});
    const runs = () => element.getByLabel('Investigator run limit', {exact: true});
    const active = () => element.getByLabel('Authorize repository repair spending');
    const status = (text: string) => element.locator('[data-repair-message]').filter({hasText: text}).waitFor();
    const writes = () => page.evaluate(() => window.factoryRepairFixture.calls().filter(call => call.method === 'PUT'));
    const recovered = () => status('A repair change is unconfirmed.');
    const select = async () => {await runs().fill('1'); await element.getByLabel('Proposal judgment limit').fill('1'); await element.getByLabel('Active minute limit').fill('30'); await active().check(); await element.getByLabel('Effect of this change').fill('Investigate the recorded stalled work.');};
    await runs().waitFor(); assert.equal((await writes()).length, 0); assert.equal(await active().isChecked(), false);
    await page.evaluate(() => window.factoryRepairFixture.mount('unknown')); await select(); await save().click(); await recovered();
    const first = (await writes())[0]!;
    const original = await page.evaluate(() => window.factoryRepairFixture.saved()); assert(original);
    const record = JSON.parse(original) as {scope: string; repository: string; change: {command_id: string; expected_revision: number; active: boolean; investigator_runs: number; proposal_judgments: number; active_minutes: number; effect: string}};
    assert.deepEqual(first.body, record.change); assert.equal(record.repository, '9007199254740993'); assert.equal(record.change.active, true);
    await page.evaluate(() => {window.factoryRepairFixture.rebind(); window.factoryRepairFixture.advance();}); await refresh().click(); await recovered();
    await retry().click(); await status('Repair grant confirmed.'); assert.deepEqual((await writes())[1], first);
    assert.match(await element.innerText(), /1 investigator runs, 0 proposal judgments, 30 active minutes reserved/);
    await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.remount();}, original); await recovered();
    assert.equal((await writes()).length, 0); await page.reload(); await recovered(); assert.equal((await writes()).length, 0);
    assert.equal(await runs().inputValue(), '1'); assert.equal(await runs().isDisabled(), true); assert.equal(await active().isChecked(), true);
    await retry().click(); await status('Repair grant confirmed.'); assert.deepEqual((await writes())[0], first);
    // Withdrawal is a separate captured command and never refunds existing charges.
    await active().uncheck(); await element.getByLabel('Effect of this change').fill('Withdraw future repair authority.'); await save().click(); await status('Repair grant confirmed.');
    assert.equal(((await writes())[1]!.body as {active: boolean}).active, false);
    assert.match(await element.innerText(), /Repair grant withdrawn/); assert.match(await element.innerText(), /30 active minutes reserved/);
    await page.evaluate(() => window.factoryRepairFixture.mount('unknown')); await select(); await save().click(); await recovered(); await retry().click(); await status('Repair grant confirmed.');
    await active().uncheck(); await element.getByLabel('Effect of this change').fill('Withdraw future authority; retain all recorded charges.'); await save().click(); await recovered();
    const withdrawal = (await writes()).at(-1)!;
    const withdrawnRaw = await page.evaluate(() => window.factoryRepairFixture.saved()); assert(withdrawnRaw);
    await page.reload(); await recovered(); assert.equal((await writes()).length, 0); assert.equal(await active().isChecked(), false);
    await retry().click(); await recovered(); assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), withdrawnRaw);
    await page.evaluate(() => window.factoryRepairFixture.mount('active', false, true)); await recovered(); await retry().click(); await status('Repair grant confirmed.');
    assert.deepEqual((await writes())[0], withdrawal); assert.match(await element.innerText(), /Repair grant withdrawn/); assert.match(await element.innerText(), /30 active minutes reserved/);
    await page.evaluate(() => window.factoryRepairFixture.mount('active')); await runs().fill('2'); await save().click(); await status('Describe the effect'); assert.equal((await writes()).length, 0);
    for (const patch of [{command_id: 'f'.repeat(32)}, {revision: 999}]) {
      await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.remount();}, original); await recovered();
      await page.evaluate(patch => window.factoryRepairFixture.receipt(patch), patch); await retry().click(); await recovered();
      assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), original);
      await page.evaluate(() => window.factoryRepairFixture.receipt({})); await retry().click(); await status('Repair grant confirmed.'); assert.deepEqual((await writes())[0], (await writes())[1]);
    }
    await page.evaluate(() => window.factoryRepairFixture.mount('stale')); await select(); await save().click(); await status('Repair authority changed.');
    assert.equal(await retry().count(), 0); assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), null);
    await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.mount('stale', false, true);}, original); await recovered(); await retry().click(); await recovered();
    assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), original); assert.equal((await writes()).length, 1);
    for (const bad of ['{broken', JSON.stringify({...record, scope: 'foreign'}), JSON.stringify({...record, repository: '1'}), JSON.stringify({...record, change: {...record.change, investigator_runs: 5}})]) {
      await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.remount();}, bad);
      await status('Saved repair change is unreadable'); assert.equal(await runs().isDisabled(), true); assert.equal((await writes()).length, 0); assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), bad);
    }
    await page.evaluate(() => {window.factoryRepairFixture.seed(null); window.factoryRepairFixture.storageFailure('read'); window.factoryRepairFixture.remount();}); await status('Saved repair change is unreadable'); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryRepairFixture.storageFailure('none')); await refresh().click(); await select();
    await page.evaluate(() => window.factoryRepairFixture.storageFailure('write')); await save().click(); await status('Repair change could not be saved.');
    const unsent = await page.evaluate(() => window.factoryRepairFixture.pending()); assert(unsent); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryRepairFixture.remount()); await recovered(); await page.evaluate(() => window.factoryRepairFixture.storageFailure('none')); await retry().click(); await status('Repair grant confirmed.'); assert.deepEqual((await writes())[0]!.body, unsent);
    await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.account('B'); window.factoryRepairFixture.remount();}, original); await select(); await save().click(); await status('Repair grant confirmed.');
    assert.notEqual(((await writes())[0]!.body as {command_id: string}).command_id, record.change.command_id);
    await page.evaluate(() => {window.factoryRepairFixture.account('A'); window.factoryRepairFixture.remount();}); await recovered(); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryRepairFixture.account('B')); await retry().click(); await status('The saved repair change or account changed.'); assert.equal((await writes()).length, 0);
    await page.evaluate(() => {window.factoryRepairFixture.account('A'); window.factoryRepairFixture.identity(false); window.factoryRepairFixture.remount();}); await status('Repair account identity could not be read.'); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryRepairFixture.identity(true)); await refresh().click(); await recovered();
    await page.evaluate(() => window.factoryRepairFixture.fail(403)); await refresh().click(); await recovered(); assert.equal(await save().count(), 0); assert.equal(await retry().isDisabled(), true);
    await page.evaluate(() => window.factoryRepairFixture.fail(0)); await refresh().click(); await recovered();
    for (const retirement of ['context', 'generation', 'disconnect']) {
      await page.evaluate(() => window.factoryRepairFixture.mount('unfunded', true)); await page.waitForFunction(() => window.factoryRepairFixture.calls().length > 0);
      const invalid = await page.evaluate(() => window.factoryRepairFixture.invalid());
      await page.evaluate(retirement => {window.factoryRepairFixture.fail(401); if (retirement === 'context') window.factoryRepairFixture.retire(); else if (retirement === 'generation') window.factoryRepairFixture.retireGeneration(); else window.factoryRepairFixture.disconnect(); window.factoryRepairFixture.release();}, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-repair')?.textContent); assert.equal(await page.evaluate(() => window.factoryRepairFixture.aborted()), true); assert.equal(await page.evaluate(() => window.factoryRepairFixture.invalid()), invalid);
      await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.remount();}, original); await recovered(); await page.evaluate(() => window.factoryRepairFixture.holdWrite()); await retry().click();
      await page.waitForFunction(() => window.factoryRepairFixture.calls().some(call => call.method === 'PUT'));
      await page.evaluate(retirement => {if (retirement === 'context') window.factoryRepairFixture.retire(); else if (retirement === 'generation') window.factoryRepairFixture.retireGeneration(); else window.factoryRepairFixture.disconnect(); window.factoryRepairFixture.release();}, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-repair')?.textContent); assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), original); assert.equal(await page.evaluate(() => window.factoryRepairFixture.aborted()), true);
    }
    const replacement = JSON.stringify({...record, change: {...record.change, command_id: 'f'.repeat(32)}});
    await page.evaluate(() => window.factoryRepairFixture.remount()); await recovered(); await page.evaluate(() => window.factoryRepairFixture.holdWrite()); await retry().click(); await page.waitForFunction(() => window.factoryRepairFixture.calls().some(call => call.method === 'PUT'));
    await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.release();}, replacement); await status('The saved repair change was replaced.'); assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), replacement);
    await page.evaluate(value => {window.factoryRepairFixture.seed(value); window.factoryRepairFixture.remount();}, original); await recovered(); await page.evaluate(() => window.factoryRepairFixture.storageFailure('remove')); await retry().click(); await status('Repair grant confirmed.');
    assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), original); assert.equal(await page.evaluate(() => window.factoryRepairFixture.pending()), undefined);
    await page.evaluate(() => {window.factoryRepairFixture.storageFailure('none'); window.factoryRepairFixture.remount();}); await save().waitFor(); assert.equal(await retry().count(), 0);
    await page.reload(); await recovered(); await retry().click(); await status('Repair grant confirmed.'); assert.deepEqual((await writes())[0], first);
    // An exact current-head read can settle a recorded command without a PUT.
    await page.evaluate(() => window.factoryRepairFixture.mount('unknown')); await select(); await save().click(); await recovered(); await refresh().click();
    assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), null); assert.equal((await writes()).length, 1);
    await page.evaluate(() => window.factoryRepairFixture.mount('unfunded')); await select(); await page.evaluate(() => window.factoryRepairFixture.failReadAfterWrite(503)); await save().click();
    await status('Repair change recorded. Current history could not be read;'); assert.equal(await page.evaluate(() => window.factoryRepairFixture.saved()), null); assert.equal(await retry().count(), 0); assert.equal((await writes()).length, 1);
    await page.evaluate(() => window.factoryRepairFixture.fail(0)); await refresh().click(); await element.getByText('Repair grant active.', {exact: false}).waitFor(); assert.equal((await writes()).length, 1);
    await page.evaluate(() => window.factoryRepairFixture.mount('active')); await runs().waitFor(); await page.evaluate(() => window.factoryRepairFixture.foreign(true)); await refresh().click(); await status('Repair funding could not be confirmed.'); assert.equal(await runs().isDisabled(), true); assert.equal((await writes()).length, 0);
    for (const state of ['offline', 'denied'] as const) {await page.evaluate(state => window.factoryRepairFixture.mount(state), state); await status(state === 'denied' ? 'current repository owner' : 'Repair funding could not be confirmed.'); assert.equal(await save().count(), 0); assert.equal((await writes()).length, 0);}
    await page.evaluate(() => document.fonts.ready.then(() => undefined)); assert.deepEqual(errors, []);
  } finally { await page.close(); host.stop(true); }
}

async function standaloneActivity() {
  const build = await Bun.build({entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-activity-fixture.ts')], target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'}});
  assert(build.success, `Standalone activity build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/activity.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    const asset = factoryAssetSource(pathname);
    if (asset) return asset;
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/activity.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400) errors.push(`Standalone activity asset failed: ${response.url()}`);
  });
  try {

    await page.goto(host.url.href);
    const element = page.locator('soda-factory-activity');
    const box = () => element.getByRole('checkbox', {name: 'Pull requests created', exact: true});
    const save = () => element.getByRole('button', {name: 'Save activity display settings', exact: true});
    const refresh = () => element.getByRole('button', {name: 'Refresh activity', exact: true});
    const status = (text: string) => element.locator('[data-activity-message]').filter({hasText: text}).waitFor();
    const writes = () => page.evaluate(() => window.factoryActivityFixture.calls().filter(call => call.method === 'PUT'));
    const link = () => element.getByRole('link', {name: 'Pull request awaiting review', exact: true});
    await link().waitFor(); assert.equal(await link().getAttribute('href'), '/desktop/pull/12');
    await element.getByText('Feed observation time unknown.', {exact: true}).waitFor(); assert.match(await element.innerText(), /time unknown/);
    await box().uncheck(); const unsaved = await page.evaluate(() => window.factoryActivityFixture.saved()); assert(unsaved);
    const count = await page.evaluate(() => window.factoryActivityFixture.calls().length);
    await page.evaluate(() => window.factoryActivityFixture.links('/another-host'));
    await page.waitForFunction(() => document.querySelector('soda-factory-activity a[href="/another-host/pull/12"]'));
    assert.equal(await box().isChecked(), false); assert.equal(await page.evaluate(() => window.factoryActivityFixture.calls().length), count);
    await page.evaluate(() => window.factoryActivityFixture.links()); await page.waitForFunction(() => document.querySelectorAll('soda-factory-activity a').length === 0);
    await element.locator('li').filter({hasText: 'Pull request awaiting review'}).waitFor(); assert.equal(await box().isChecked(), false);
    await page.evaluate(() => window.factoryActivityFixture.remount()); await status('Saved activity preferences restored.');
    assert.equal((await writes()).length, 0); await page.reload(); await status('Saved activity preferences restored.');
    assert.equal(await box().isChecked(), false); assert.equal((await writes()).length, 0);
    await save().click(); await status('Activity display settings saved.');
    const first = (await writes())[0]!;
    assert.deepEqual(first.body, {expected_revision: 3, settings: {issue_created: true, pr_created: false, verified_merge: true}});
    assert.equal(await page.evaluate(() => window.factoryActivityFixture.saved()), null); assert.equal(await link().count(), 0);
    // Unknown replies are reconciled against current preferences by GET alone.
    await page.evaluate(() => window.factoryActivityFixture.mount('unknown')); await box().uncheck(); await save().click(); await status('Factory activity is unavailable.');
    const unknown = await page.evaluate(() => window.factoryActivityFixture.saved()); assert(unknown);
    await refresh().click(); await page.waitForFunction(() => window.factoryActivityFixture.saved() === null);
    assert.equal((await writes()).length, 1); assert.equal(await link().count(), 0);
    await page.evaluate(value => {window.factoryActivityFixture.seed(value); window.factoryActivityFixture.remount();}, unknown);
    await status('Saved activity preferences restored.'); assert.equal((await writes()).length, 0);
    await page.reload(); await status('Saved activity preferences restored.'); await save().click(); await status('Activity display settings saved.'); assert.deepEqual((await writes())[0], first);
    // A stale selection needs an explicit review against the new current head.
    await page.evaluate(() => window.factoryActivityFixture.mount('recorded')); await box().uncheck();
    await page.evaluate(() => window.factoryActivityFixture.advance()); await save().click(); await status('Activity settings changed.');
    const stale = await page.evaluate(() => window.factoryActivityFixture.saved()); assert(stale);
    await refresh().click(); await status('Saved activity preferences restored.');
    await element.getByRole('button', {name: 'Review preferences against current activity', exact: true}).click();
    assert.equal((await writes()).length, 1); await save().click(); await status('Activity display settings saved.');
    assert.equal(((await writes())[1]!.body as {expected_revision: number}).expected_revision, 4);
    const record = JSON.parse(unsaved) as {scope: string; repository: string; draft: {expected_revision: number; settings: Record<string, boolean>; submitted: boolean}};
    for (const bad of ['{broken', JSON.stringify({...record, repository: '1'}), JSON.stringify({...record, scope: 'foreign'}), JSON.stringify({...record, draft: {...record.draft, settings: {issue_created: true}}})]) {
      await page.evaluate(value => {window.factoryActivityFixture.seed(value); window.factoryActivityFixture.remount();}, bad);
      await status('Saved activity preferences are unreadable'); assert.equal(await box().isDisabled(), true); assert.equal((await writes()).length, 0); assert.equal(await page.evaluate(() => window.factoryActivityFixture.saved()), bad);
    }
    await page.evaluate(() => {window.factoryActivityFixture.seed(null); window.factoryActivityFixture.storageFailure('read'); window.factoryActivityFixture.remount();}); await status('Saved activity preferences are unreadable'); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryActivityFixture.storageFailure('none')); await refresh().click();
    await page.evaluate(() => window.factoryActivityFixture.storageFailure('write')); await box().uncheck(); await save().click(); await status('Activity preferences could not be saved locally.'); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryActivityFixture.remount()); await status('Saved activity preferences restored.');
    await page.evaluate(() => window.factoryActivityFixture.storageFailure('none')); await save().click(); await status('Activity display settings saved.'); assert.deepEqual((await writes())[0], first);
    await page.evaluate(value => {window.factoryActivityFixture.seed(value); window.factoryActivityFixture.account('B'); window.factoryActivityFixture.remount();}, unsaved);
    await box().waitFor(); assert.equal(await box().isChecked(), true); assert.equal((await writes()).length, 0);
    await page.evaluate(() => {window.factoryActivityFixture.account('A'); window.factoryActivityFixture.remount();}); await status('Saved activity preferences restored.'); assert.equal(await box().isChecked(), false);
    await page.evaluate(() => window.factoryActivityFixture.account('B')); await save().click(); await status('Saved preferences or account changed.'); assert.equal((await writes()).length, 0);
    await page.evaluate(() => {window.factoryActivityFixture.account('A'); window.factoryActivityFixture.identity(false); window.factoryActivityFixture.remount();}); await status('Activity account identity is unavailable.'); assert.equal(await box().isDisabled(), true);
    await page.evaluate(() => window.factoryActivityFixture.identity(true)); await refresh().click(); await status('Saved activity preferences restored.');
    await page.evaluate(() => window.factoryActivityFixture.editable(false)); assert.equal(await save().isDisabled(), true);
    await element.locator('form').evaluate(form => form.dispatchEvent(new SubmitEvent('submit', {bubbles: true, cancelable: true}))); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryActivityFixture.editable(true));
    await page.evaluate(() => window.factoryActivityFixture.fail(503)); await refresh().click(); await status('Last confirmed activity is shown'); assert.equal(await save().isDisabled(), true); await link().waitFor();
    for (const code of [401, 403]) {await page.evaluate(code => window.factoryActivityFixture.fail(code), code); await refresh().click(); await status(code === 403 ? 'current repository owner' : 'Factory activity is unavailable.'); assert.equal(await element.getByRole('link').count(), 0);}
    for (const retirement of ['context', 'generation', 'disconnect']) {
      await page.evaluate(() => window.factoryActivityFixture.mount('recorded', true)); await page.waitForFunction(() => window.factoryActivityFixture.calls().length > 0);
      await page.evaluate(retirement => {if (retirement === 'context') window.factoryActivityFixture.retire(); else if (retirement === 'generation') window.factoryActivityFixture.retireGeneration(); else window.factoryActivityFixture.disconnect(); window.factoryActivityFixture.release();}, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-activity')?.textContent); assert.equal(await page.evaluate(() => window.factoryActivityFixture.aborted()), true);
      await page.evaluate(() => window.factoryActivityFixture.mount('recorded')); await box().uncheck();
      await page.evaluate(() => window.factoryActivityFixture.holdWrite()); await save().click(); await page.waitForFunction(() => window.factoryActivityFixture.calls().some(call => call.method === 'PUT'));
      const saved = await page.evaluate(() => window.factoryActivityFixture.saved()); assert(saved);
      await page.evaluate(retirement => {if (retirement === 'context') window.factoryActivityFixture.retire(); else if (retirement === 'generation') window.factoryActivityFixture.retireGeneration(); else window.factoryActivityFixture.disconnect(); window.factoryActivityFixture.release();}, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-activity')?.textContent); assert.equal(await page.evaluate(() => window.factoryActivityFixture.aborted()), true); assert.equal(await page.evaluate(() => window.factoryActivityFixture.saved()), saved);
    }
    const replacement = JSON.stringify({...record, draft: {...record.draft, settings: {...record.draft.settings, verified_merge: false}}});
    await page.evaluate(value => {window.factoryActivityFixture.seed(value); window.factoryActivityFixture.remount();}, unsaved); await status('Saved activity preferences restored.');
    await page.evaluate(() => window.factoryActivityFixture.holdWrite()); await save().click(); await page.waitForFunction(() => window.factoryActivityFixture.calls().some(call => call.method === 'PUT'));
    await page.evaluate(value => {window.factoryActivityFixture.seed(value); window.factoryActivityFixture.release();}, replacement); await status('Factory activity is unavailable.'); assert.equal(await page.evaluate(() => window.factoryActivityFixture.saved()), replacement);
    await page.evaluate(value => {window.factoryActivityFixture.seed(value); window.factoryActivityFixture.remount();}, unsaved); await status('Saved activity preferences restored.');
    await page.evaluate(() => window.factoryActivityFixture.storageFailure('remove')); await save().click(); await status('Activity display settings saved.'); assert.equal(await page.evaluate(() => window.factoryActivityFixture.draft()), undefined);
    await page.evaluate(() => {window.factoryActivityFixture.storageFailure('none'); window.factoryActivityFixture.remount();}); await box().waitFor(); assert.equal(await box().isChecked(), true); assert.equal((await writes()).length, 0);
    await page.reload(); await status('Saved activity preferences restored.'); await save().click(); await status('Activity display settings saved.'); assert.deepEqual((await writes())[0], first);
    await page.evaluate(() => window.factoryActivityFixture.mount('truncated')); await element.getByText('Older entries exist beyond this page.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryActivityFixture.mismatch(true)); await refresh().click(); await status('Factory activity is unavailable.'); assert.equal(await save().isDisabled(), true);
    await page.evaluate(() => window.factoryActivityFixture.mount('empty')); await element.getByText('No recorded activity yet.', {exact: true}).waitFor(); assert.equal((await writes()).length, 0);
    await page.evaluate(() => window.factoryActivityFixture.mount('read-only')); await box().waitFor(); assert.equal(await box().isDisabled(), true); assert.equal((await writes()).length, 0);
    await page.evaluate(() => document.fonts.ready.then(() => undefined)); assert.deepEqual(errors, []);
  } finally { await page.close(); host.stop(true); }
}

async function standaloneDecisions() {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-decisions-fixture.ts')],
    target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Standalone decisions build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({
    hostname: '127.0.0.1', port: 0,
    fetch(request) {
      const pathname = new URL(request.url).pathname;
      if (pathname === '/decisions.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      const asset = factoryAssetSource(pathname);
      if (asset) return asset;
      if (pathname !== '/') return new Response(null, {status: 404});
      return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/decisions.js"></script>', {headers: {'Content-Type': 'text/html'}});
    },
  });
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400)
      errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`);
  });
  try {
    await page.goto(host.url.href);
    const acceptance = page.locator('soda-factory-requirements-acceptance');
    const retry = page.locator('soda-factory-completion-retry');
    const review = () => acceptance.getByRole('button', {name: 'Review current requirements', exact: true});
    const accept = () => acceptance.getByRole('button', {name: 'Accept requirements', exact: true});
    const retryButton = () => retry.getByRole('button', {name: 'Retry completion review', exact: true});
    const posts = () => page.evaluate(() => window.factoryDecisionsFixture.calls().filter(call => call.method === 'POST'));
    const preview = async () => {
      await review().click();
      await acceptance.getByRole('checkbox').check();
    };
    await acceptance.getByRole('link', {name: 'Review issue #12', exact: true}).waitFor();
    assert.equal(await acceptance.getByRole('link', {name: 'Review issue #12', exact: true}).getAttribute('href'), '/desktop/issue/12');
    await acceptance.getByRole('textbox').first().fill('21');
    await preview();
    const count = await page.evaluate(() => window.factoryDecisionsFixture.calls().length);
    await page.evaluate(() => window.factoryDecisionsFixture.rebindLinks());
    await page.waitForFunction(() => document.querySelector('soda-factory-requirements-acceptance a')?.getAttribute('href') === '/another-host/issue/12');
    assert.equal(await acceptance.getByRole('link', {name: 'Review comment 21', exact: true}).getAttribute('href'), '/another-host/issue/12/discussion/21');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.calls().length), count);
    assert.equal(await acceptance.getByRole('checkbox').isChecked(), true);
    await page.evaluate(() => window.factoryDecisionsFixture.removeLinks());
    await page.waitForFunction(() => document.querySelectorAll('soda-factory-requirements-acceptance a').length === 0);
    assert.equal(await acceptance.getByRole('checkbox').isChecked(), true);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await page.evaluate(() => window.factoryDecisionsFixture.mount('unknown'));
    await preview();
    await accept().click();
    const repeat = acceptance.getByRole('button', {name: 'Retry same acceptance', exact: true});
    await repeat.waitFor();
    const first = (await posts())[0];
    assert(first);
    assert.equal(first.path, 'repositories/9007199254740993/factory/issues/12/acceptances');
    await page.evaluate(() => window.factoryDecisionsFixture.rebindLinks());
    await repeat.click();
    await acceptance.getByRole('status').filter({hasText: 'Requirements accepted.'}).waitFor();
    assert.deepEqual((await posts())[1], first);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), ['accepted']);
    await page.evaluate(() => window.factoryDecisionsFixture.mount('unknown'));
    await preview();
    await accept().click();
    await repeat.waitFor();
    await page.evaluate(() => window.factoryDecisionsFixture.retireGeneration());
    await repeat.click();
    await page.waitForFunction(() => document.querySelector('soda-factory-requirements-acceptance')?.textContent === '');
    assert.equal((await posts()).length, 1);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
    await page.evaluate(() => window.factoryDecisionsFixture.mount('unknown'));
    await retryButton().click();
    await retry.getByRole('status').filter({hasText: 'No retry was confirmed.'}).waitFor();
    const firstRetry = (await posts())[0];
    assert(firstRetry);
    assert.equal(firstRetry.path, 'repositories/9007199254740993/factory/actions');
    await retryButton().click();
    await retry.getByRole('status').filter({hasText: 'Completion review retry queued.'}).waitFor();
    assert.deepEqual((await posts())[1], firstRetry);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), ['retry']);
    await page.evaluate(() => window.factoryDecisionsFixture.mount('stale'));
    await preview();
    await accept().click();
    await acceptance.getByRole('status').filter({hasText: 'Requirements changed.'}).waitFor();
    assert.equal(await acceptance.getByRole('button', {name: 'Retry same acceptance', exact: true}).count(), 0);
    assert.equal(await acceptance.getByRole('checkbox').count(), 0);
    assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
    await page.evaluate(() => window.factoryDecisionsFixture.mount('denied'));
    await acceptance.getByRole('status').filter({hasText: 'code-write authority'}).waitFor();
    await retryButton().waitFor();
    assert.equal(await review().isDisabled(), true);
    assert.equal(await retryButton().isDisabled(), true);
    assert.deepEqual(await posts(), []);
    await page.evaluate(() => window.factoryDecisionsFixture.mount('offline'));
    await acceptance.getByRole('status').filter({hasText: 'Declared offline peer.'}).waitFor();
    await retry.getByRole('status').filter({hasText: 'Could not load completion reviews.'}).waitFor();
    await page.evaluate(() => window.factoryDecisionsFixture.mount('loading'));
    await acceptance.getByRole('status').filter({hasText: 'Checking current acceptance authority…'}).waitFor();
    await retry.getByRole('status').filter({hasText: 'Loading completion reviews…'}).waitFor();
    await page.evaluate(() => window.factoryDecisionsFixture.retire());
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.aborted()), true);
    await page.evaluate(() => window.factoryDecisionsFixture.mount('ready', true));
    await acceptance.getByRole('status').filter({hasText: 'Checking current acceptance authority…'}).waitFor();
    await retry.getByRole('status').filter({hasText: 'Loading completion reviews…'}).waitFor();
    await page.evaluate(() => {
      window.factoryDecisionsFixture.retire();
      window.factoryDecisionsFixture.release();
    });
    await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => resolve())));
    assert.equal(await acceptance.textContent(), '');
    assert.equal(await retry.textContent(), '');
    assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.aborted()), true);
    // Delayed action confirmations may describe an applied server effect. Retirement
    // cancels only this UI lifetime; it must neither replay nor announce that effect.
    for (const control of ['acceptance', 'retry'] as const) {
      await page.evaluate(() => window.factoryDecisionsFixture.mount('ready'));
      if (control === 'acceptance') await preview();
      else await page.waitForFunction(() => {
        const button = document.querySelector('soda-factory-completion-retry button') as HTMLButtonElement | null;
        return button && !button.disabled;
      });
      await page.evaluate(() => window.factoryDecisionsFixture.hold());
      if (control === 'acceptance') await accept().click();
      else await retryButton().click();
      await page.waitForFunction(() => window.factoryDecisionsFixture.calls().some(call => call.method === 'POST'));
      if (control === 'acceptance') await page.evaluate(() => window.factoryDecisionsFixture.retire());
      else await page.evaluate(() => window.factoryDecisionsFixture.retireGeneration());
      await page.evaluate(() => window.factoryDecisionsFixture.release());
      await page.waitForFunction((tag) => document.querySelector(tag)?.textContent === '',
        control === 'acceptance' ? 'soda-factory-requirements-acceptance' : 'soda-factory-completion-retry');
      assert.equal(await page.evaluate(() => window.factoryDecisionsFixture.aborted()), true);
      assert.deepEqual(await page.evaluate(() => window.factoryDecisionsFixture.events()), []);
      assert.equal((await posts()).length, 1);
    }
    await completionRetryRecovery(page);
    await acceptanceRecovery(page);
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
    host.stop(true);
  }
}

async function contextChoiceRecovery(page: Page) {
  const control = page.locator('soda-factory-context-resolution');
  const choose = () => control.getByRole('button', {name: 'Use current required context', exact: true});
  const retry = (index = 1) => control.getByRole('button', {name: `Retry saved context choice ${index}`, exact: true});
  const refresh = () => control.getByRole('button', {name: 'Refresh current context', exact: true});
  const status = (text: string) => control.getByRole('status').filter({hasText: text}).waitFor();
  const posts = () => page.evaluate(() => window.factoryContextFixture.calls().filter(call => call.method === 'POST'));
  const recovered = () => status('An unconfirmed context choice is saved.');
  const confirmed = () => status('Current required context selected.');
  await page.evaluate(() => window.factoryContextFixture.mount('unknown'));
  await choose().click();
  await status('No result was confirmed.');
  const first = (await posts())[0]!;
  const original = await page.evaluate(() => window.factoryContextFixture.saved());
  assert(original);
  const journal = JSON.parse(original) as {scope: string; repository: string; issue: string;
    choices: {run: string; fingerprint: string; scope_digest: string; required_digest: string}[]};
  const saved = journal.choices[0]!;
  assert.equal(journal.scope, 'declared-profile:provider:account');
  assert.equal(journal.repository, '9007199254740993');
  assert.equal(journal.issue, '12');
  assert.deepEqual(first.body, {action: 'use_current_context', context: {
    fingerprint: saved.fingerprint, scope_digest: saved.scope_digest, required_digest: saved.required_digest,
  }});
  await page.evaluate(() => window.factoryContextFixture.remount());
  await recovered();
  assert.deepEqual(await posts(), []);
  await page.reload();
  await recovered();
  assert.deepEqual(await posts(), []);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
  // A current "serve" projection is not an authenticated replay receipt.
  await page.evaluate(saved => window.factoryContextFixture.recorded(saved), saved);
  await refresh().click();
  await page.waitForFunction(() => document.querySelector('soda-factory-context-resolution')?.textContent?.includes('An unconfirmed context choice is saved.'));
  assert.deepEqual(await posts(), []);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
  assert.equal(await choose().count(), 0);
  await retry().click();
  await confirmed();
  assert.deepEqual((await posts())[0], first);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), null);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 1);

  // Recovery retains the original tuple after current requirements change,
  // and when no current selection is projected at all.
  for (const omit of [false, true]) {
    await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount();}, original);
    await recovered();
    await page.evaluate(({saved, omit}) => {
      window.factoryContextFixture.recorded(saved);
      if (omit) window.factoryContextFixture.omitSelection();
      else window.factoryContextFixture.advanceRequired();
    }, {saved, omit});
    await refresh().click();
    await retry().waitFor();
    assert.deepEqual(await posts(), []);
    await retry().click();
    await confirmed();
    assert.deepEqual((await posts())[0], first);
  }
  // An issue's new latest run cannot retarget its older saved request.
  await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount();}, original);
  await recovered();
  await page.evaluate(() => window.factoryContextFixture.latestRun('b'.repeat(32)));
  await refresh().click();
  await page.waitForFunction(() => window.factoryContextFixture.selectedRun() === 'b'.repeat(32));
  assert.deepEqual(await posts(), []);
  await retry().click();
  await confirmed();
  assert.deepEqual((await posts())[0], first);
  await choose().click();
  await confirmed();
  assert.equal((await posts())[1]?.path, `factory/runs/${'b'.repeat(32)}/actions`);
  for (const state of ['offline', 'denied'] as const) {
    await page.evaluate(({original, state}) => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount(state);}, {original, state});
    await status('Could not load current context.');
    await retry().click();
    await status('could not be verified');
    assert.deepEqual(await posts(), []);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
  }
  for (const patch of [{run_id: 'b'.repeat(32)}, {fingerprint: 'b'.repeat(64)},
    {scope_digest: 'b'.repeat(64)}, {required_digest: 'b'.repeat(64)}, {bundle: 'captured'}, {bundle: undefined}]) {
    await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount();}, original);
    await recovered();
    await page.evaluate(patch => window.factoryContextFixture.receipt(patch), patch);
    await retry().click();
    await status('No result was confirmed.');
    assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
    assert.deepEqual((await posts())[0], first);
  }
  await page.evaluate(() => window.factoryContextFixture.receipt({}, 403));
  await retry().click();
  await status('Its exact request remains saved.');
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
  await page.evaluate(() => window.factoryContextFixture.receipt({}));
  await retry().click();
  await confirmed();

  const second = {...saved, fingerprint: 'b'.repeat(64), scope_digest: 'b'.repeat(64), required_digest: 'b'.repeat(64)};
  // A changed same-run selection may be chosen without overwriting its earlier
  // uncertain request. Retiring the old UI never clears its retained tuple.
  await page.evaluate(() => {window.factoryContextFixture.mount('complete', 'run'); window.factoryContextFixture.holdAction();});
  await choose().click();
  await page.waitForFunction(() => window.factoryContextFixture.calls().some(call => call.method === 'POST'));
  assert.equal(JSON.parse((await page.evaluate(() => window.factoryContextFixture.saved()))!).choices.length, 1);
  await page.evaluate(() => window.factoryContextFixture.revise());
  await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
  await choose().click();
  await page.waitForFunction(() => window.factoryContextFixture.calls().filter(call => call.method === 'POST').length === 2);
  assert.deepEqual(JSON.parse((await page.evaluate(() => window.factoryContextFixture.saved()))!).choices, [saved, second]);
  await page.evaluate(() => window.factoryContextFixture.release());
  await confirmed();
  assert.deepEqual(JSON.parse((await page.evaluate(() => window.factoryContextFixture.saved()))!).choices, [saved]);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 1);
  await page.evaluate(() => window.factoryContextFixture.remount());
  await recovered();
  assert.deepEqual(await posts(), []);

  const full = JSON.stringify({...journal, choices: Array.from({length: 32}, (_, index) => ({
    ...saved, fingerprint: index.toString(16).padStart(64, '0'),
  }))});
  await page.evaluate(full => {window.factoryContextFixture.seed(full); window.factoryContextFixture.remount();}, full);
  await recovered();
  await choose().click();
  await status('Saved context choices need to be checked');
  assert.deepEqual(await posts(), []);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), full);
  await page.evaluate(() => window.factoryContextFixture.receipt({}, 403));
  await retry().click();
  await status('Its exact request remains saved.');
  assert.equal((await posts()).length, 1);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), full);
  // Unknown bytes, foreign scope and duplicate identities are not overwritten.
  for (const corrupt of ['{broken', JSON.stringify({...journal, scope: 'foreign'}),
    JSON.stringify({...journal, repository: '7'}), JSON.stringify({...journal, issue: '13'}),
    JSON.stringify({...journal, extra: true}), JSON.stringify({...journal, choices: [saved, saved]}),
    JSON.stringify({...journal, choices: [{...saved, required_digest: 'invalid'}]})]) {
    await page.evaluate(value => {window.factoryContextFixture.seed(value); window.factoryContextFixture.remount();}, corrupt);
    await status('Saved context choices are unreadable');
    assert.equal(await choose().isDisabled(), true);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), corrupt);
    assert.deepEqual(await posts(), []);
  }
  await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount();}, original);
  await recovered();
  await page.evaluate(() => {window.factoryContextFixture.changeScope('another-profile:provider:account'); window.factoryContextFixture.clearCalls();});
  await refresh().click();
  await page.waitForFunction(() => document.querySelector('[aria-label="Saved context choices"]') === null);
  assert.deepEqual(await posts(), []);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), null);
  await page.evaluate(() => window.factoryContextFixture.changeScope('declared-profile:provider:account'));
  // The refresh button remains offered by calling the public refresh port.
  await page.evaluate(() => window.factoryContextFixture.refresh());
  await recovered();
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);

  for (const mismatch of ['repository', 'issue', 'run'] as const) {
    await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount();}, original);
    await recovered();
    await page.evaluate(mismatch => window.factoryContextFixture.wireScope(
      mismatch === 'repository' ? '7' : '9007199254740993', mismatch === 'issue' ? '13' : '12',
      mismatch === 'run' ? 'b'.repeat(32) : undefined), mismatch);
    await retry().click();
    await status('could not be verified');
    assert.deepEqual(await posts(), []);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
  }
  // Failures to read or persist storage never dispatch; failed writes preserve
  // their exact identity across remount in the same document.
  await page.evaluate(() => {window.factoryContextFixture.mount('complete'); window.factoryContextFixture.storageFailure('read'); window.factoryContextFixture.remount();});
  await status('Saved context choices are unreadable');
  assert.equal(await choose().isDisabled(), true);
  assert.deepEqual(await posts(), []);
  await page.evaluate(() => {window.factoryContextFixture.storageFailure('none'); window.factoryContextFixture.mount('complete'); window.factoryContextFixture.storageFailure('write');});
  await choose().click();
  await status('The context choice could not be saved.');
  assert.deepEqual(await posts(), []);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), null);
  const unsaved = await page.evaluate(() => window.factoryContextFixture.pending().choices);
  assert.deepEqual(unsaved, [saved]);
  await page.evaluate(() => window.factoryContextFixture.remount());
  await recovered();
  await page.evaluate(() => window.factoryContextFixture.storageFailure('none'));
  await retry().click();
  await confirmed();
  assert.deepEqual((await posts())[0], first);

  // A matching reply may remove only its tuple from the current collection.
  for (const keepOriginal of [true, false]) {
    await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount(); window.factoryContextFixture.holdAction();}, original);
    await recovered();
    await retry().click();
    await page.waitForFunction(() => window.factoryContextFixture.calls().some(call => call.method === 'POST'));
    const replacement = JSON.stringify({...journal, choices: keepOriginal ? [saved, second] : [second]});
    await page.evaluate(value => {window.factoryContextFixture.replaceRaw(value); window.factoryContextFixture.release();}, replacement);
    if (keepOriginal) {
      await confirmed();
      assert.deepEqual(JSON.parse((await page.evaluate(() => window.factoryContextFixture.saved()))!).choices, [second]);
      assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 1);
    } else {
      await status('No saved choice was cleared.');
      assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), replacement);
      assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
    }
  }
  // Retired responses cannot acknowledge or clear a saved choice.
  await page.evaluate(original => {window.factoryContextFixture.seed(original); window.factoryContextFixture.remount(); window.factoryContextFixture.holdAction();}, original);
  await recovered();
  await retry().click();
  await page.waitForFunction(() => window.factoryContextFixture.calls().some(call => call.method === 'POST'));
  await page.evaluate(() => {window.factoryContextFixture.retire(); window.factoryContextFixture.release();});
  await page.waitForFunction(() => document.querySelector('soda-factory-context-resolution')?.textContent === '');
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
  // A failed removal is visible; a new document can explicitly replay the same
  // server-confirmed tuple, without automatically dispatching on reload.
  await page.evaluate(() => {window.factoryContextFixture.remount(); window.factoryContextFixture.storageFailure('remove');});
  await recovered();
  await retry().click();
  await status('Saved-choice cleanup could not be persisted');
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), original);
  assert.deepEqual(await page.evaluate(() => window.factoryContextFixture.pending().choices), []);
  await page.reload();
  await recovered();
  assert.deepEqual(await posts(), []);
  await retry().click();
  await confirmed();
  assert.deepEqual((await posts())[0], first);
  assert.equal(await page.evaluate(() => window.factoryContextFixture.saved()), null);
}

async function standaloneResolution() {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-context-fixture.ts')],
    target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Standalone context build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({
    hostname: '127.0.0.1', port: 0,
    fetch(request) {
      const pathname = new URL(request.url).pathname;
      if (pathname === '/context.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      const asset = factoryAssetSource(pathname);
      if (asset) return asset;
      if (pathname !== '/') return new Response(null, {status: 404});
      return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/context.js"></script>', {headers: {'Content-Type': 'text/html'}});
    },
  });
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400)
      errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`);
  });
  try {
    await page.goto(host.url.href);
    const element = page.locator('soda-factory-context-resolution');
    const button = element.getByRole('button', {name: 'Use current required context', exact: true});
    const status = element.getByRole('status');
    const posts = () => page.evaluate(() => window.factoryContextFixture.calls().filter(call => call.method === 'POST'));
    await button.waitFor();
    assert.equal(await button.isEnabled(), true);
    assert.deepEqual(await page.evaluate(() => window.factoryContextFixture.calls().map(call => call.path)), [
      'repositories/9007199254740993/factory/issues/12', `factory/runs/${'a'.repeat(32)}`,
    ]);
    await page.evaluate(() => window.factoryContextFixture.rebind());
    assert.equal(await page.evaluate(() => window.factoryContextFixture.calls().length), 2);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await page.evaluate(() => window.factoryContextFixture.mount('unknown'));
    await button.click();
    await status.filter({hasText: 'No result was confirmed.'}).waitFor();
    const first = (await posts())[0];
    assert(first);
    await page.evaluate(() => window.factoryContextFixture.rebind());
    await button.click();
    await status.filter({hasText: 'Current required context selected.'}).waitFor();
    assert.deepEqual((await posts())[1], first);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 1);
    for (const state of ['incomplete', 'offline', 'denied', 'stale'] as const) {
      await page.evaluate(state => window.factoryContextFixture.mount(state), state);
      if (state === 'incomplete') {
        await button.waitFor();
        assert.equal(await button.isDisabled(), true);
      } else if (state === 'stale') {
        await button.click();
        await status.filter({hasText: 'This context choice is no longer available.'}).waitFor();
        assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
      } else {
        await status.filter({hasText: 'Could not load current context.'}).waitFor();
        assert.equal(await button.count(), 0);
      }
    }
    await page.evaluate(() => window.factoryContextFixture.mount('complete', 'issue', true));
    await status.filter({hasText: 'Loading current context…'}).waitFor();
    await page.evaluate(() => window.factoryContextFixture.supplyRun());
    await button.waitFor();
    assert.equal(await page.evaluate(() => window.factoryContextFixture.aborted()), true);
    await page.evaluate(() => window.factoryContextFixture.release());
    await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => resolve())));
    assert.equal(await page.evaluate(() => window.factoryContextFixture.calls().length), 1);
    assert.equal(await button.isEnabled(), true);
    // A same-ID changed direct selection retires its earlier confirmation;
    // repeated identical direct input leaves the active action bound.
    await page.evaluate(() => window.factoryContextFixture.mount('complete', 'run'));
    assert.deepEqual(await page.evaluate(() => window.factoryContextFixture.calls()), []);
    await page.evaluate(() => window.factoryContextFixture.holdAction());
    await button.click();
    await status.filter({hasText: 'Recording your current-context choice…'}).waitFor();
    await page.waitForFunction(() => window.factoryContextFixture.calls().some(call => call.method === 'POST'));
    await page.evaluate(() => window.factoryContextFixture.rebind());
    assert.equal(await button.isDisabled(), true);
    await page.evaluate(() => window.factoryContextFixture.revise());
    await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.aborted()), true);
    await page.evaluate(() => window.factoryContextFixture.release());
    await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => resolve())));
    assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
    assert.equal((await posts()).length, 1);
    await button.click();
    await status.filter({hasText: 'Current required context selected.'}).waitFor();
    assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 1);
    assert.equal(((await posts())[1]?.body as {context: {fingerprint: string}}).context.fingerprint, 'b'.repeat(64));
    for (const entry of ['issue', 'run'] as const) {
      await page.evaluate(mode => window.factoryContextFixture.mount('complete', mode), entry);
      await button.waitFor();
      await page.evaluate(() => window.factoryContextFixture.holdAction());
      await button.click();
      await status.filter({hasText: 'Recording your current-context choice…'}).waitFor();
    await page.waitForFunction(() => window.factoryContextFixture.calls().some(call => call.method === 'POST'));
      await page.evaluate(() => {
        window.factoryContextFixture.retireGeneration();
        window.factoryContextFixture.release();
      });
      await page.waitForFunction(() => document.querySelector('soda-factory-context-resolution')?.textContent === '');
      assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
      assert.equal(await page.evaluate(() => window.factoryContextFixture.aborted()), true);
      assert.equal((await posts()).length, 1);
    }
    await page.evaluate(() => window.factoryContextFixture.mount('complete', 'run'));
    await page.evaluate(() => window.factoryContextFixture.retireGeneration());
    await button.click();
    await page.waitForFunction(() => document.querySelector('soda-factory-context-resolution')?.textContent === '');
    assert.deepEqual(await posts(), []);
    await page.evaluate(() => window.factoryContextFixture.mount('complete', 'issue', true));
    await status.filter({hasText: 'Loading current context…'}).waitFor();
    await page.evaluate(() => {
      window.factoryContextFixture.disconnect();
      window.factoryContextFixture.release();
    });
    await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => resolve())));
    assert.equal(await page.evaluate(() => window.factoryContextFixture.aborted()), true);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.calls().length), 1);
    await page.evaluate(() => window.factoryContextFixture.mount('account-unavailable'));
    await status.filter({hasText: 'Could not verify the current account.'}).waitFor();
    assert.equal(await button.isDisabled(), true);
    assert.equal(await element.getByText('The required current context is complete.', {exact: false}).count(), 1);
    assert.deepEqual(await posts(), []);
    // Host account custody is separate from the transport generation. Reads
    // remain usable without it; missing, malformed, changed or unavailable
    // custody never becomes a mutation under a fallback identity.
    await page.evaluate(() => {
      window.factoryContextFixture.mount('complete', 'run');
      window.factoryContextFixture.missingScope();
    });
    await status.filter({hasText: 'Could not verify the current account.'}).waitFor();
    assert.equal(await button.isDisabled(), true);
    assert.deepEqual(await posts(), []);
    for (const scope of ['another-profile:provider:account', '', 'invalid account']) {
      await page.evaluate(() => window.factoryContextFixture.mount('complete', 'run'));
      await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
      await page.evaluate(scope => window.factoryContextFixture.changeScope(scope), scope);
      await button.click();
      await status.filter({hasText: scope ? (scope.includes(' ') ? 'could not be verified' : 'account changed') : 'could not be verified'}).waitFor();
      assert.deepEqual(await posts(), []);
      assert.equal(await button.isDisabled(), true);
      assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
    }
    // Explicitly refresh the host identity; refreshing itself sends no write.
    await page.evaluate(() => window.factoryContextFixture.changeScope('another-profile:provider:account'));
    await element.getByRole('button', {name: 'Refresh current context', exact: true}).click();
    await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
    assert.deepEqual(await posts(), []);
    await button.click();
    await status.filter({hasText: 'Current required context selected.'}).waitFor();
    assert.equal((await posts()).length, 1);
    await page.evaluate(() => window.factoryContextFixture.mount('complete', 'run'));
    await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
    await page.evaluate(() => window.factoryContextFixture.failScope());
    await button.click();
    await status.filter({hasText: 'could not be verified'}).waitFor();
    assert.deepEqual(await posts(), []);
    // A current run response must prove its wire ID, repository, issue and
    // exact displayed selection, even when a parent supplied direct run data.
    for (const mismatch of ['repository', 'issue', 'run', 'selection'] as const) {
      await page.evaluate(() => window.factoryContextFixture.mount('complete', 'run'));
      await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
      await page.evaluate(mismatch => {
        if (mismatch === 'selection') window.factoryContextFixture.revisePeer();
        else window.factoryContextFixture.wireScope(
          mismatch === 'repository' ? '7' : '9007199254740993',
          mismatch === 'issue' ? '13' : '12',
          mismatch === 'run' ? 'b'.repeat(32) : undefined);
      }, mismatch);
      await button.click();
      await status.filter({hasText: 'could not be verified'}).waitFor();
      assert.deepEqual(await posts(), []);
      assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
      if (mismatch === 'selection') {
        await element.getByRole('button', {name: 'Refresh current context', exact: true}).click();
        await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
        assert.deepEqual(await posts(), []);
        await button.click();
        await status.filter({hasText: 'Current required context selected.'}).waitFor();
        assert.equal(((await posts())[0]?.body as {context: {fingerprint: string}}).context.fingerprint, 'b'.repeat(64));
      }
    }
    // Scope checks and fresh run reads share the action's selection lifetime.
    await page.evaluate(() => window.factoryContextFixture.mount('complete', 'run'));
    await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
    await page.evaluate(() => window.factoryContextFixture.holdRead());
    await button.click();
    await page.waitForFunction(() => window.factoryContextFixture.heldCount() === 1);
    await page.evaluate(() => {
      window.factoryContextFixture.revise();
      window.factoryContextFixture.release();
    });
    await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
    assert.equal(await page.evaluate(() => window.factoryContextFixture.aborted()), true);
    assert.deepEqual(await posts(), []);
    await button.click();
    await status.filter({hasText: 'Current required context selected.'}).waitFor();
    assert.equal(((await posts())[0]?.body as {context: {fingerprint: string}}).context.fingerprint, 'b'.repeat(64));
    for (const initial of [true, false]) {
      await page.evaluate(initial => window.factoryContextFixture.mount('complete', 'run', initial ? 'scope' : false), initial);
      if (!initial) {
        await page.waitForFunction(() => !(document.querySelector('soda-factory-context-resolution button') as HTMLButtonElement)?.disabled);
        await page.evaluate(() => window.factoryContextFixture.holdScope());
        await button.click();
      }
      await page.waitForFunction(() => window.factoryContextFixture.heldCount() === 1);
      await page.evaluate(() => {
        window.factoryContextFixture.retire();
        window.factoryContextFixture.release();
      });
      await page.waitForFunction(() => document.querySelector('soda-factory-context-resolution')?.textContent === '');
      assert.equal(await page.evaluate(() => window.factoryContextFixture.aborted()), true);
      assert.deepEqual(await posts(), []);
      assert.equal(await page.evaluate(() => window.factoryContextFixture.events()), 0);
    }
    await contextChoiceRecovery(page);
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
    host.stop(true);
  }
}

async function standalonePresentation() {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-presentation-fixture.ts')],
    target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Standalone presentation build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({
    hostname: '127.0.0.1', port: 0,
    fetch(request) {
      const pathname = new URL(request.url).pathname;
      if (pathname === '/presentation.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      const asset = factoryAssetSource(pathname);
      if (asset) return asset;
      if (pathname !== '/') return new Response(null, {status: 404});
      return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/presentation.js"></script>', {headers: {'Content-Type': 'text/html'}});
    },
  });
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400)
      errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`);
  });
  try {
    await page.goto(host.url.href);
    await page.getByRole('region', {name: 'Work detail', exact: true}).waitFor();
    assert.equal(await page.getByRole('link', {name: 'Native issue', exact: true}).getAttribute('href'), '/desktop/repositories/9007199254740993/issues/12');
    assert.equal(await page.getByRole('link', {name: 'Native pull request', exact: true}).getAttribute('href'), '/desktop/repositories/9007199254740993/pulls/21');
    assert.equal(await page.locator('a[href^="/recorded-forge"]').count(), 0);
    await page.evaluate(() => window.factoryPresentationFixture.show('detail', 'recorded', true, '/another-shell'));
    assert.equal(await page.getByRole('link', {name: 'Native issue', exact: true}).getAttribute('href'), '/another-shell/repositories/9007199254740993/issues/12');
    await page.evaluate(() => window.factoryPresentationFixture.show('detail', 'recorded', false));
    assert.equal(await page.getByRole('link').count(), 0);
    await page.getByText('Native issue link unavailable.').waitFor();
    await page.evaluate(() => window.factoryPresentationFixture.show('detail', 'delivered'));
    await page.getByRole('status').filter({hasText: 'Delivered: merge confirmed and original-issue closure confirmed.'}).waitFor();
    await page.evaluate(() => window.factoryPresentationFixture.show('detail', 'blocked'));
    await page.getByRole('status').filter({hasText: 'Waiting on prerequisites.'}).waitFor();
    await page.getByText('Wait for the prerequisite code.', {exact: false}).waitFor();
    await page.evaluate(() => window.factoryPresentationFixture.show('request'));
    await page.getByRole('status').filter({hasText: '1 waiting'}).waitFor();
    await page.getByText('0 of 1 accepted product outcomes evidenced.', {exact: true}).waitFor();
    assert.equal(await page.getByRole('link', {name: 'issue #12', exact: true}).getAttribute('href'), '/desktop/repositories/9007199254740993/issues/12');
    await page.evaluate(() => window.factoryPresentationFixture.show('plan'));
    assert((await page.getByRole('link').all()).length >= 2);
    for (const link of await page.getByRole('link').all()) assert.equal(await link.getAttribute('href'), '/desktop/repositories/9007199254740993/issues/12');
    await page.evaluate(() => window.factoryPresentationFixture.show('overview', 'unavailable'));
    await page.getByText('Accepted direction could not be read.', {exact: true}).waitFor();
    for (const state of ['missing', 'unavailable'] as const) {
      await page.evaluate(state => window.factoryPresentationFixture.show('direction', state), state);
      await page.getByText(state === 'missing' ? 'No accepted direction recorded.' : 'Accepted direction could not be read.', {exact: true}).waitFor();
    }
    await page.evaluate(() => window.factoryPresentationFixture.show('history'));
    await page.getByRole('status').filter({hasText: '1 revision shown, newest first.'}).waitFor();
    await page.evaluate(() => window.factoryPresentationFixture.show('history', 'unavailable'));
    await page.getByRole('status').filter({hasText: 'Revision history could not be read.'}).waitFor();
    await page.evaluate(() => window.factoryPresentationFixture.show('access'));
    await page.getByRole('region', {name: 'Access standing', exact: true}).waitFor();
    await page.evaluate(() => window.factoryPresentationFixture.show('interventions'));
    assert.equal(await page.getByRole('button', {name: 'Pause Factory work', exact: true}).isDisabled(), true);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
    host.stop(true);
  }
}

async function standaloneProgress() {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-progress-fixture.ts')],
    target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Standalone progress build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({
    hostname: '127.0.0.1', port: 0,
    fetch(request) {
      const pathname = new URL(request.url).pathname;
      if (pathname === '/progress.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      const asset = factoryAssetSource(pathname);
      if (asset) return asset;
      if (pathname !== '/') return new Response(null, {status: 404});
      return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/progress.js"></script>', {headers: {'Content-Type': 'text/html'}});
    },
  });
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400)
      errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`);
  });
  try {
    await page.goto(host.url.href);
    const overview = page.locator('soda-factory-work-overview');
    const plan = page.locator('soda-factory-work-plan');
    const direction = page.locator('soda-factory-direction-revision');
    const request = overview.getByRole('region', {name: 'Request overview', exact: true});
    await request.getByRole('status').filter({hasText: '1 waiting'}).waitFor();
    await plan.getByText('No recorded priority comparison names this issue.').waitFor();
    await direction.getByText('1 revision shown, newest first.').waitFor();
    assert.deepEqual(await page.evaluate(() => window.factoryProgressFixture.scope().issues.map(item => item.issue)), ['12']);
    assert.equal(await request.getByRole('link', {name: 'issue #12', exact: true}).getAttribute('href'), '/desktop/repositories/9007199254740993/issues/12');
    const initial = await page.evaluate(() => window.factoryProgressFixture.calls());
    assert.equal(initial.length, 5);
    assert(initial.every(path => !/allowance|factory\/access|factory\/actions/.test(path)));
    await page.evaluate(() => window.factoryProgressFixture.rebind());
    await page.waitForFunction(() => document.querySelector('soda-factory-work-overview a')?.getAttribute('href') === '/another-shell/repositories/9007199254740993/issues/12');
    for (const link of await plan.getByRole('link').all()) assert.equal(await link.getAttribute('href'), '/another-shell/repositories/9007199254740993/issues/12');
    assert.deepEqual(await page.evaluate(() => window.factoryProgressFixture.calls()), initial);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await page.evaluate(() => window.factoryProgressFixture.fail(503));
    await overview.getByRole('button', {name: 'Refresh overview', exact: true}).click();
    await overview.getByText('Last loaded work is still shown.', {exact: false}).waitFor();
    assert.equal(await request.count(), 1);
    await page.evaluate(() => window.factoryProgressFixture.fail(403));
    await overview.getByRole('button', {name: 'Refresh overview', exact: true}).click();
    await page.waitForFunction(() => document.querySelectorAll('soda-factory-work-overview a').length === 0);
    assert.equal(await request.count(), 0);
    assert.deepEqual(await page.evaluate(() => window.factoryProgressFixture.scope().issues), []);
    await page.evaluate(() => window.factoryProgressFixture.mount('partial'));
    await plan.getByText('1 work item could not be read.').waitFor();
    await plan.getByRole('button', {name: 'Refresh plan', exact: true}).waitFor();
    assert.equal(await plan.getByRole('link', {name: 'Issue #12', exact: true}).count(), 1);
    await page.evaluate(() => window.factoryProgressFixture.mount('missing'));
    await direction.getByText('No accepted direction recorded.', {exact: true}).waitFor();
    await direction.getByText('No revision history recorded yet.', {exact: true}).waitFor();
    for (const state of ['denied', 'offline'] as const) {
      await page.evaluate(state => window.factoryProgressFixture.mount(state), state);
      await direction.getByText('Accepted direction could not be read.', {exact: true}).waitFor();
      assert.equal(await overview.getByRole('link').count(), 0);
    }
    // Each component starts only its first-stage read. Ignored-abort peers must
    // not trigger plan/history follow-ups or republish a retired child scope.
    for (const retirement of ['context', 'generation', 'disconnect'] as const) {
      await page.evaluate(() => window.factoryProgressFixture.mount('ready', true));
      await page.waitForFunction(() => window.factoryProgressFixture.calls().length === 3);
      await page.evaluate(mode => {
        if (mode === 'context') window.factoryProgressFixture.retire();
        else if (mode === 'generation') window.factoryProgressFixture.retireGeneration();
        else window.factoryProgressFixture.disconnect();
        window.factoryProgressFixture.release();
      }, retirement);
      await page.waitForFunction(() => [...document.querySelectorAll('soda-factory-work-overview, soda-factory-work-plan, soda-factory-direction-revision')].every(node => node.textContent === ''));
      assert.equal(await page.evaluate(() => window.factoryProgressFixture.aborted()), true);
      assert.equal(await page.evaluate(() => window.factoryProgressFixture.calls().length), 3);
      assert.deepEqual(await page.evaluate(() => window.factoryProgressFixture.scope()), {repository: '', issues: []});
    }
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
    host.stop(true);
  }
}

async function verificationRecovery(page: Page) {
  const element = page.locator('soda-factory-verification-retry');
  const fresh = (repair: boolean) => element.getByRole('button', {name: repair ? 'Request application repair' : 'Retry application verification', exact: true});
  const repeat = (repair: boolean) => element.getByRole('button', {name: repair ? 'Retry same repair request' : 'Retry same verification request', exact: true});
  const refresh = () => element.getByRole('button', {name: 'Refresh verification', exact: true});
  const status = (message: string) => element.getByRole('status').filter({hasText: message}).waitFor();
  const posts = () => page.evaluate(() => window.factoryVerificationFixture.calls().filter(call => call.method === 'POST'));
  const recovered = () => status('An unconfirmed verification request is saved.');
  const settled = async (repair: boolean) => {
    if (repair) await element.getByRole('region', {name: 'Application repair progress'}).waitFor();
    else await status('Verification Retry queued.');
  };
  const samples: {raw: string; request: {path: string; method: string; body?: unknown}}[] = [];
  for (const repair of [false, true]) {
    await page.evaluate(repair => window.factoryVerificationFixture.mount(repair ? 'unknown-repair' : 'unknown-retry'), repair);
    await fresh(repair).click();
    await status(repair ? 'No repair request was confirmed.' : 'No Retry was confirmed.');
    const request = (await posts())[0]!;
    const raw = await page.evaluate(() => window.factoryVerificationFixture.saved());
    assert(raw);
    const record = JSON.parse(raw) as {run: string; command: string; action: string; repository: string};
    assert.equal(record.repository, '9007199254740993');
    assert.equal(record.run, 'a'.repeat(32));
    assert.deepEqual(request.body, repair ? {action: 'repair_delivery'} : {action: 'retry', command_id: record.command});
    samples.push({raw, request});
    await page.evaluate(() => window.factoryVerificationFixture.remount());
    await recovered();
    await repeat(repair).waitFor();
    assert.equal((await posts()).length, 0);
    // Recovery must still freshly prove a saved run after host inventory drops it.
    await page.evaluate(() => window.factoryVerificationFixture.selected(false));
    await recovered();
    await repeat(repair).waitFor();
    assert.equal((await posts()).length, 0);
    await page.reload();
    await recovered();
    await repeat(repair).waitFor();
    assert.equal((await posts()).length, 0);
    if (repair) assert.equal(await fresh(false).isDisabled(), true);
    await repeat(repair).click();
    await settled(repair);
    assert.deepEqual((await posts())[0], request);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), null);
    // Refusal does not prove that the original saved request never applied.
    await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.remount();}, raw);
    await recovered();
    await repeat(repair).waitFor();
    for (const code of [403, 409]) {
      await page.evaluate(code => window.factoryVerificationFixture.fail(code), code);
      await repeat(repair).click();
      await status(repair ? 'No repair request was confirmed.' : 'No Retry was confirmed.');
      assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), raw);
      assert.deepEqual((await posts()).at(-1), request);
    }
    await page.evaluate(() => window.factoryVerificationFixture.fail(0));
    await repeat(repair).click();
    await settled(repair);

    await page.evaluate(repair => window.factoryVerificationFixture.mount(repair ? 'repair' : 'retry'), repair);
    await fresh(repair).waitFor();
    await page.evaluate(() => window.factoryVerificationFixture.storageFailure('write'));
    await fresh(repair).click();
    await status('Verification request could not be saved.');
    const unsent = await page.evaluate(() => window.factoryVerificationFixture.pending());
    assert(unsent);
    assert.equal((await posts()).length, 0);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), null);
    await page.evaluate(() => window.factoryVerificationFixture.remount());
    await recovered();
    await repeat(repair).waitFor();
    assert.equal((await posts()).length, 0);
    await page.evaluate(() => window.factoryVerificationFixture.storageFailure('none'));
    await repeat(repair).click();
    await settled(repair);
    assert.deepEqual((await posts())[0]!.body, repair ? {action: 'repair_delivery'} : {action: 'retry', command_id: unsent.command});

    for (const retirement of ['context', 'generation']) {
      await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.remount();}, raw);
      await recovered();
      await repeat(repair).waitFor();
      await page.evaluate(() => window.factoryVerificationFixture.hold('action'));
      await repeat(repair).click();
      await page.waitForFunction(() => window.factoryVerificationFixture.calls().some(call => call.method === 'POST'));
      await page.evaluate(retirement => {
        if (retirement === 'context') window.factoryVerificationFixture.retire();
        else window.factoryVerificationFixture.retireGeneration();
        window.factoryVerificationFixture.release();
      }, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-verification-retry')?.textContent);
      assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), raw);
      assert.equal(await page.evaluate(() => window.factoryVerificationFixture.aborted()), true);
      await page.evaluate(() => window.factoryVerificationFixture.remount());
      await recovered();
      await repeat(repair).waitFor();
      assert.equal((await posts()).length, 0);
      await repeat(repair).click();
      await settled(repair);
      assert.deepEqual((await posts())[0], request);
    }
    const replacement = JSON.stringify({...JSON.parse(raw), command: 'f'.repeat(32), action: 'retry'});
    await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.remount();}, raw);
    await recovered();
    await repeat(repair).waitFor();
    await page.evaluate(() => window.factoryVerificationFixture.hold('action'));
    await repeat(repair).click();
    await page.waitForFunction(() => window.factoryVerificationFixture.calls().some(call => call.method === 'POST'));
    await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.release();}, replacement);
    await status('The saved verification request changed.');
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), replacement);
    assert.equal(await element.getByRole('region', {name: 'Application repair progress'}).count(), 0);
    // A failed removal leaves a document tombstone. Reload can replay the old
    // durable request with its original identity rather than minting another.
    await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.remount();}, raw);
    await recovered();
    await repeat(repair).waitFor();
    await page.evaluate(() => window.factoryVerificationFixture.storageFailure('remove'));
    await repeat(repair).click();
    await settled(repair);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), raw);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.pending()), undefined);
    await page.evaluate(() => {window.factoryVerificationFixture.storageFailure('none'); window.factoryVerificationFixture.remount();});
    await fresh(false).waitFor();
    assert.equal(await repeat(repair).count(), 0);
    assert.equal((await posts()).length, 0);
    await page.reload();
    await recovered();
    await repeat(repair).click();
    await settled(repair);
    assert.deepEqual((await posts())[0], request);
  }
  const original = samples[0]!.raw;
  const record = JSON.parse(original) as {command: string};
  for (const corrupt of ['{broken', JSON.stringify({...JSON.parse(original), scope: 'another-account'}),
    JSON.stringify({...JSON.parse(original), repository: '1'}), JSON.stringify({...JSON.parse(original), action: 'stop'}),
    JSON.stringify({...JSON.parse(original), extra: true})]) {
    await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.remount();}, corrupt);
    await status('Saved verification request is unreadable');
    await fresh(false).waitFor();
    assert.equal(await fresh(false).isDisabled(), true);
    assert.equal((await posts()).length, 0);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), corrupt);
  }
  await page.evaluate(() => {window.factoryVerificationFixture.seed(null); window.factoryVerificationFixture.storageFailure('read'); window.factoryVerificationFixture.remount();});
  await status('Saved verification request is unreadable');
  assert.equal(await fresh(false).isDisabled(), true);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryVerificationFixture.storageFailure('none'));
  await refresh().click();
  await fresh(false).waitFor();
  for (const patch of [{view: {repository: '1'}}, {view: {repository: '9007199254740993', issue: '12'}}, {run: {id: 'b'.repeat(32), reconciled: true}}]) {
    await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.mount('retry', true, true);}, original);
    await recovered();
    await page.evaluate(patch => {window.factoryVerificationFixture.read(patch); window.factoryVerificationFixture.selected(false); window.factoryVerificationFixture.release();}, patch);
    await status('Could not read current verification');
    assert.equal(await repeat(false).count(), 0);
    assert.equal((await posts()).length, 0);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), original);
  }
  await page.evaluate(value => {window.factoryVerificationFixture.seed(value); window.factoryVerificationFixture.account('B'); window.factoryVerificationFixture.remount();}, original);
  await fresh(false).click();
  await settled(false);
  assert.notEqual(((await posts())[0]!.body as {command_id: string}).command_id, record.command);
  await page.evaluate(() => {window.factoryVerificationFixture.account('A'); window.factoryVerificationFixture.remount();});
  await recovered();
  assert.equal(await page.evaluate(() => window.factoryVerificationFixture.saved()), original);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {window.factoryVerificationFixture.identity(false); window.factoryVerificationFixture.remount();});
  await status('Verification identity could not be read.');
  assert.equal(await fresh(false).isDisabled(), true);
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.factoryVerificationFixture.identity(true));
  await refresh().click();
  await recovered();
  await repeat(false).waitFor();
  await page.evaluate(() => window.factoryVerificationFixture.account('B'));
  await repeat(false).click();
  await status('The saved verification request or account changed.');
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {window.factoryVerificationFixture.account('A'); window.factoryVerificationFixture.remount();});
  await recovered();
  await repeat(false).click();
  await settled(false);
  assert.deepEqual((await posts())[0], samples[0]!.request);
}

async function standaloneVerification() {
  const build = await Bun.build({entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-verification-fixture.ts')], target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'}});
  assert(build.success, `Standalone verification build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0];
  assert(script);
  const host = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/verification.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    const asset = factoryAssetSource(pathname);
    if (asset) return asset;
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/verification.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (response.url().startsWith(host.url.origin) && response.status() >= 400) errors.push(`Standalone verification asset failed: ${response.url()}`);
  });
  try {
    await page.goto(host.url.href);
    const element = page.locator('soda-factory-verification-retry');
    const retry = () => element.getByRole('button', {name: 'Retry application verification', exact: true});
    const repair = () => element.getByRole('button', {name: 'Request application repair', exact: true});
    const repeatRetry = () => element.getByRole('button', {name: 'Retry same verification request', exact: true});
    const repeatRepair = () => element.getByRole('button', {name: 'Retry same repair request', exact: true});
    const refresh = () => element.getByRole('button', {name: 'Refresh verification', exact: true});
    const posts = () => page.evaluate(() => window.factoryVerificationFixture.calls().filter(call => call.method === 'POST'));
    const status = (message: string) => element.getByRole('status').filter({hasText: message}).waitFor();
    await retry().waitFor();
    assert.equal(await retry().isEnabled(), true);
    assert.equal(await repair().count(), 0);
    assert.deepEqual(await page.evaluate(() => window.factoryVerificationFixture.calls()), [{path: `factory/runs/${'a'.repeat(32)}`, method: 'GET'}]);
    await page.evaluate(() => window.factoryVerificationFixture.rebind());
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.calls().length), 1);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await retry().click();
    await status('Verification Retry queued.');
    const first = (await posts())[0]!;
    assert.equal(first.path, `factory/runs/${'a'.repeat(32)}/actions`);
    assert.match((first.body as {command_id: string}).command_id, /^[a-f0-9]{32}$/);
    assert.equal((first.body as {action: string}).action, 'retry');
    for (const kind of ['retry', 'repair'] as const) {
      await page.evaluate(kind => window.factoryVerificationFixture.mount(kind === 'retry' ? 'unknown-retry' : 'unknown-repair'), kind);
      await (kind === 'retry' ? retry() : repair()).click();
      await status(kind === 'retry' ? 'No Retry was confirmed.' : 'No repair request was confirmed.');
      const pending = (await posts())[0]!;
      await page.evaluate(() => window.factoryVerificationFixture.rebind());
      await refresh().click();
      await (kind === 'retry' ? repeatRetry() : repeatRepair()).click();
      if (kind === 'retry') await status('Verification Retry queued.');
      else await element.getByRole('region', {name: 'Application repair progress'}).getByRole('status').filter({hasText: 'Strategist diagnosis is queued.'}).waitFor();
      assert.deepEqual((await posts())[1], pending);
      if (kind === 'repair') assert.deepEqual(pending.body, {action: 'repair_delivery'});
    }
    for (const patch of [{command_id: 'b'.repeat(32)}, {prior_run: 'b'.repeat(32)}, {queued: false}]) {
      await page.evaluate(() => window.factoryVerificationFixture.mount('retry'));
      await retry().waitFor();
      await page.evaluate(patch => window.factoryVerificationFixture.receipt(patch), patch);
      await retry().click();
      await status('No Retry was confirmed.');
      const pending = (await posts())[0]!;
      await page.evaluate(() => window.factoryVerificationFixture.receipt({}));
      await repeatRetry().click();
      await status('Verification Retry queued.');
      assert.deepEqual((await posts())[1], pending);
    }
    for (const patch of [{run_id: 'b'.repeat(32)}, {cause_id: ''}, {cycle_id: ''}, {state: 'adopted'}]) {
      await page.evaluate(() => window.factoryVerificationFixture.mount('repair'));
      await repair().waitFor();
      await page.evaluate(patch => window.factoryVerificationFixture.receipt(patch), patch);
      await repair().click();
      await status('No repair request was confirmed.');
      assert.equal(await element.getByRole('region', {name: 'Application repair progress'}).count(), 0);
      await page.evaluate(() => window.factoryVerificationFixture.receipt({}));
      await repeatRepair().click();
      await element.getByRole('region', {name: 'Application repair progress'}).waitFor();
      assert.deepEqual((await posts())[0], (await posts())[1]);
    }
    // A failed refresh cannot leave last-loaded capabilities actionable. A
    // denied read clears the facts; restored authority may discover them again.
    await page.evaluate(() => window.factoryVerificationFixture.mount('retry'));
    await retry().waitFor();
    await page.evaluate(() => window.factoryVerificationFixture.fail(503));
    await refresh().click();
    await status('Last confirmed states are shown');
    assert.equal(await retry().isDisabled(), true);
    assert.equal((await posts()).length, 0);
    await page.evaluate(() => window.factoryVerificationFixture.fail(0));
    await refresh().click();
    await page.waitForFunction(() => !(document.querySelector('soda-factory-verification-retry button') as HTMLButtonElement)?.disabled);
    for (const code of [401, 403]) {
      await page.evaluate(code => window.factoryVerificationFixture.fail(code), code);
      await refresh().click();
      await status('Could not read current verification');
      assert.equal(await retry().count(), 0);
      await page.evaluate(() => window.factoryVerificationFixture.fail(0));
      await refresh().click();
      await retry().waitFor();
    }
    for (const patch of [{run: {id: 'b'.repeat(32), reconciled: true}}, {view: {repository: '9007199254740992'}},
      {view: {repository: '9007199254740993', issue: '12'}},
      {delivery_repair: {available: false, progress: {failed_run: 'b'.repeat(32), cause_id: 'c'.repeat(32), cycle_id: 'd'.repeat(32), state: 'attention'}}}]) {
      await page.evaluate(() => window.factoryVerificationFixture.mount('retry', true));
      await status('Loading verification and repair progress…');
      await page.evaluate(patch => {window.factoryVerificationFixture.read(patch); window.factoryVerificationFixture.release();}, patch);
      await status('Could not read current verification');
      assert.equal(await retry().count(), 0);
      assert.equal((await posts()).length, 0);
    }
    await page.evaluate(() => window.factoryVerificationFixture.mount('empty'));
    await page.waitForFunction(() => document.querySelector('soda-factory-verification-retry')?.textContent === '');
    assert.equal((await posts()).length, 0);
    for (const state of ['offline', 'denied'] as const) {
      await page.evaluate(state => window.factoryVerificationFixture.mount(state), state);
      await status('Could not read current verification');
      assert.equal(await retry().count(), 0);
      assert.equal((await posts()).length, 0);
    }
    await page.evaluate(() => window.factoryVerificationFixture.mount('loading'));
    await status('Loading verification and repair progress…');
    await page.evaluate(() => window.factoryVerificationFixture.retire());
    await page.waitForFunction(() => document.querySelector('soda-factory-verification-retry')?.textContent === '');
    for (const retirement of ['context', 'generation', 'disconnect']) {
      await page.evaluate(() => window.factoryVerificationFixture.mount('retry', true));
      await status('Loading verification and repair progress…');
      await page.evaluate(retirement => {
        if (retirement === 'context') window.factoryVerificationFixture.retire();
        else if (retirement === 'generation') window.factoryVerificationFixture.retireGeneration();
        else window.factoryVerificationFixture.disconnect();
        window.factoryVerificationFixture.release();
      }, retirement);
      await page.waitForFunction(() => !document.querySelector('soda-factory-verification-retry')?.textContent);
      assert.equal(await page.evaluate(() => window.factoryVerificationFixture.aborted()), true);
      assert.equal((await posts()).length, 0);
      for (const kind of ['retry', 'repair'] as const) {
        await page.evaluate(kind => window.factoryVerificationFixture.mount(kind), kind);
        await (kind === 'retry' ? retry() : repair()).waitFor();
        await page.evaluate(() => window.factoryVerificationFixture.hold('action'));
        await (kind === 'retry' ? retry() : repair()).click();
        await page.waitForFunction(() => window.factoryVerificationFixture.calls().some(call => call.method === 'POST'));
        await page.evaluate(retirement => {
          if (retirement === 'context') window.factoryVerificationFixture.retire();
          else if (retirement === 'generation') window.factoryVerificationFixture.retireGeneration();
          else window.factoryVerificationFixture.disconnect();
          window.factoryVerificationFixture.release();
        }, retirement);
        await page.waitForFunction(() => !document.querySelector('soda-factory-verification-retry')?.textContent);
        assert.equal(await page.evaluate(() => window.factoryVerificationFixture.aborted()), true);
        assert.equal((await posts()).length, 1);
      }
    }
    await page.evaluate(() => window.factoryVerificationFixture.mount('repair'));
    await repair().click();
    const progress = element.getByRole('region', {name: 'Application repair progress'});
    await progress.getByRole('status').filter({hasText: 'Strategist diagnosis is queued.'}).waitFor();
    await page.evaluate(() => window.factoryVerificationFixture.progress('investigating'));
    await progress.getByRole('status').filter({hasText: 'Strategist diagnosis is in progress.'}).waitFor();
    assert.equal((await posts()).length, 1);
    await page.evaluate(() => window.factoryVerificationFixture.progress('attention'));
    await progress.getByRole('status').filter({hasText: 'Application repair needs attention.'}).waitFor();
    await progress.getByText('Judgment: accept; acceptance probability 0.87; confidence 0.24.', {exact: true}).waitFor();
    assert.equal((await posts()).length, 1);
    await page.evaluate(() => window.factoryVerificationFixture.mount('investigating'));
    await progress.waitFor();
    await page.evaluate(() => window.factoryVerificationFixture.retire());
    const reads = await page.evaluate(() => window.factoryVerificationFixture.calls().length);
    await page.waitForTimeout(4200);
    assert.equal(await page.evaluate(() => window.factoryVerificationFixture.calls().length), reads);
    await page.evaluate(() => window.factoryVerificationFixture.mount('adopted'));
    await progress.getByRole('status').filter({hasText: 'Fresh readiness and application verification are still required.'}).waitFor();
    assert.equal((await posts()).length, 0);
    await verificationRecovery(page);
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
    assert.deepEqual(errors, []);
  } finally {
    await page.close();
    host.stop(true);
  }
}

async function standaloneDetail() {
  const build = await Bun.build({entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-detail-fixture.ts')], target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'}});
  assert(build.success, `Standalone detail build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0]; assert(script);
  const host = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/detail.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    const asset = factoryAssetSource(pathname); if (asset) return asset;
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/detail.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  const page = await browser.newPage({viewport: {width: 1280, height: 900}}); page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => { if (response.url().startsWith(host.url.origin) && response.status() >= 400) errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`); });
  try {
    await page.goto(host.url.href);
    const detail = page.locator('soda-factory-work-detail');
    const retry = detail.getByRole('button', {name: 'Retry run', exact: true});
    const watch = detail.getByRole('button', {name: 'Watch output', exact: true});
    const posts = () => page.evaluate(() => window.factoryDetailFixture.calls().filter(call => call.method === 'POST'));
    await watch.waitFor();
    assert.equal(await detail.getByRole('link', {name: 'Native issue', exact: true}).getAttribute('href'), '/desktop/repositories/9007199254740993/issues/12');
    assert.equal(await detail.getByRole('link', {name: 'Native pull request', exact: true}).getAttribute('href'), '/desktop/repositories/9007199254740993/pulls/21');
    const initial = await page.evaluate(() => window.factoryDetailFixture.calls());
    await page.evaluate(() => window.factoryDetailFixture.rebind());
    await page.waitForFunction(() => document.querySelector('soda-factory-work-detail a')?.getAttribute('href') === '/another-shell/repositories/9007199254740993/issues/12');
    assert.deepEqual(await page.evaluate(() => window.factoryDetailFixture.calls()), initial);
    await watch.click();
    assert.deepEqual(await page.evaluate(() => window.factoryDetailFixture.watches()), [{run: 'a'.repeat(32), repository: '9007199254740993', environment: 'p' + 'e'.repeat(24)}]);
    assert.deepEqual(await posts(), []);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await page.evaluate(() => window.factoryDetailFixture.mount('unknown'));
    await retry.click();
    await detail.getByText('No reply was confirmed. Retry the same request or refresh.', {exact: true}).waitFor();
    const first = (await posts())[0]; assert(first);
    await page.evaluate(() => window.factoryDetailFixture.rebind());
    await retry.click();
    await detail.getByText('Retry queued. The prior run stays settled; no other attempt was created.', {exact: true}).waitFor();
    assert.deepEqual((await posts())[1], first);
    assert.equal(first.path, `factory/runs/${'a'.repeat(32)}/actions`);
    await page.evaluate(() => window.factoryDetailFixture.mount('read-only'));
    await watch.waitFor(); assert.equal(await retry.isDisabled(), true); assert.equal(await watch.isDisabled(), true);
    for (const state of ['offline', 'denied'] as const) {
      await page.evaluate(state => window.factoryDetailFixture.mount(state), state);
      await detail.getByText('Declared detail read unavailable.', {exact: true}).waitFor();
      assert.equal(await retry.isDisabled(), true); assert.equal(await watch.count(), 0); assert.deepEqual(await posts(), []);
    }
    await page.evaluate(() => window.factoryDetailFixture.mount('ready')); await watch.waitFor();
    const runReads = await page.evaluate(() => window.factoryDetailFixture.calls().filter(call => call.path.startsWith('factory/runs/')).length);
    await page.evaluate(() => window.factoryDetailFixture.fail(403));
    await detail.getByRole('button', {name: 'Refresh detail', exact: true}).click();
    await detail.getByText('Declared detail read unavailable.', {exact: true}).waitFor();
    assert.equal(await detail.getByRole('link').count(), 0); assert.equal(await retry.isDisabled(), true);
    assert.equal(await page.evaluate(() => window.factoryDetailFixture.calls().filter(call => call.path.startsWith('factory/runs/')).length), runReads);
    for (const mode of ['context', 'generation', 'disconnect'] as const) {
      await page.evaluate(() => window.factoryDetailFixture.mount('ready', true));
      await detail.getByText('Loading work detail…', {exact: true}).waitFor(); assert.equal(await retry.isDisabled(), true);
      await page.evaluate(mode => {
        if (mode === 'context') window.factoryDetailFixture.retire();
        else if (mode === 'generation') window.factoryDetailFixture.retireGeneration();
        else window.factoryDetailFixture.disconnect();
        window.factoryDetailFixture.release();
      }, mode);
      await page.waitForFunction(() => !document.querySelector('soda-factory-work-detail')?.textContent);
      assert.equal(await page.evaluate(() => window.factoryDetailFixture.aborted()), true);
      assert.equal(await page.evaluate(() => window.factoryDetailFixture.calls().length), 1);
    }
    await page.evaluate(() => window.factoryDetailFixture.mount('ready')); await watch.waitFor();
    await page.evaluate(() => window.factoryDetailFixture.holdRetry()); await retry.click();
    await detail.getByText('Requested; awaiting confirmation.', {exact: true}).waitFor();
    await page.evaluate(() => { window.factoryDetailFixture.retireGeneration(); window.factoryDetailFixture.release(); });
    await page.waitForFunction(() => document.querySelector('soda-factory-work-detail')?.textContent === '');
    assert.equal(await page.evaluate(() => window.factoryDetailFixture.aborted()), true); assert.equal((await posts()).length, 1);
    await page.evaluate(() => window.factoryDetailFixture.mount('ready')); await watch.waitFor();
    await page.evaluate(() => window.factoryDetailFixture.retireGeneration()); await retry.click();
    await page.waitForFunction(() => document.querySelector('soda-factory-work-detail')?.textContent === ''); assert.deepEqual(await posts(), []);
    await runRetryRecovery(page);
    await page.evaluate(() => document.fonts.ready.then(() => undefined)); assert.deepEqual(errors, []);
  } finally { await page.close(); host.stop(true); }
}

async function runRetryRecovery(page: Page) {
  const detail = page.locator('soda-factory-work-detail');
  const retry = detail.getByRole('button', {name: 'Retry run', exact: true});
  const savedNotice = () => detail.getByText('An unconfirmed retry is saved. Retry the same request to check its recorded result.', {exact: true});
  const posts = () => page.evaluate(() => window.factoryDetailFixture.calls().filter(call => call.method === 'POST'));
  const ready = () => page.waitForFunction(() => [...document.querySelectorAll('soda-factory-work-detail button')].some(node => node.textContent?.trim() === 'Retry run' && !(node as HTMLButtonElement).disabled));
  const queued = () => detail.getByText('Retry queued. The prior run stays settled; no other attempt was created.', {exact: true}).waitFor();
  const lost = () => detail.getByText('No reply was confirmed. Retry the same request or refresh.', {exact: true}).waitFor();
  const originalRaw = await page.evaluate(() => window.factoryDetailFixture.saved());
  assert(originalRaw); const original = JSON.parse(originalRaw) as {command: string};
  await page.reload(); await savedNotice().waitFor(); assert.deepEqual(await posts(), []);
  await retry.click(); await queued();
  assert.equal(((await posts())[0]?.body as {command_id: string}).command_id, original.command);
  assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved()), null);

  // An unknown reply survives an element replacement and uses its old target.
  await page.evaluate(() => window.factoryDetailFixture.mount('unknown'));
  await retry.click(); await lost();
  const retained = (await posts())[0]; assert(retained);
  await page.evaluate(() => window.factoryDetailFixture.mount('ready')); await savedNotice().waitFor();
  assert.deepEqual(await posts(), []); await retry.click(); await queued();
  assert.deepEqual((await posts())[0], retained);

  // Unknown bytes, foreign work scopes and an unprovable old run are preserved.
  for (const raw of ['{unrecognized', JSON.stringify({scope: 'foreign', repository: '9007199254740993', issue: '12', command: 'c'.repeat(32), run: 'a'.repeat(32)})]) {
    await page.evaluate(raw => { window.factoryDetailFixture.seed(raw); window.factoryDetailFixture.mount('ready'); }, raw);
    await detail.getByText('Saved retry is unreadable or belongs to another scope.', {exact: false}).waitFor();
    assert.equal(await retry.isDisabled(), true); assert.deepEqual(await posts(), []);
    assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved()), raw);
  }
  const oldRun = JSON.stringify({scope: 'declared:detail:primary', repository: '9007199254740993', issue: '12', command: 'c'.repeat(32), run: 'f'.repeat(32)});
  await page.evaluate(raw => { window.factoryDetailFixture.seed(raw); window.factoryDetailFixture.mount('ready'); }, oldRun);
  await detail.getByText('Retry identity or its saved run could not be read.', {exact: false}).waitFor();
  assert.equal(await retry.isDisabled(), true); assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved()), oldRun);
  assert.deepEqual(await posts(), []);
  await page.evaluate(() => window.factoryDetailFixture.seed(''));

  // A saved target outside today's list can recover after a fresh matching read.
  const readableOldRun = JSON.stringify({scope: 'declared:detail:primary', repository: '9007199254740993', issue: '12', command: 'c'.repeat(32), run: 'e'.repeat(32)});
  await page.evaluate(raw => { window.factoryDetailFixture.seed(raw); window.factoryDetailFixture.mount('ready'); }, readableOldRun);
  await savedNotice().waitFor(); assert.deepEqual(await posts(), []);
  await retry.click(); await queued();
  assert.equal((await posts())[0]?.path, `factory/runs/${'e'.repeat(32)}/actions`);
  assert.equal(((await posts())[0]?.body as {command_id: string}).command_id, 'c'.repeat(32));

  // A storage read denial cannot be mistaken for no pending command.
  const known = JSON.stringify({scope: 'declared:detail:primary', repository: '9007199254740993', issue: '12', command: 'd'.repeat(32), run: 'a'.repeat(32)});
  await page.evaluate(raw => { window.factoryDetailFixture.seed(raw); window.factoryDetailFixture.storageFailure('read'); window.factoryDetailFixture.mount('ready'); }, known);
  await detail.getByText('Saved retry is unreadable or belongs to another scope.', {exact: false}).waitFor();
  assert.equal(await retry.isDisabled(), true); assert.deepEqual(await posts(), []);
  assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved()), known);
  await page.evaluate(() => { window.factoryDetailFixture.storageFailure('none'); window.factoryDetailFixture.mount('ready'); });
  await savedNotice().waitFor(); await retry.click(); await queued();
  assert.equal(((await posts())[0]?.body as {command_id: string}).command_id, 'd'.repeat(32));

  // Failed persistence retains the chosen identity in this document and sends nothing.
  await page.evaluate(() => { window.factoryDetailFixture.storageFailure('write'); window.factoryDetailFixture.mount('ready'); });
  await retry.click(); await detail.getByText('Retry identity could not be saved.', {exact: false}).waitFor();
  assert.deepEqual(await posts(), []);
  const attempted = JSON.parse(await page.evaluate(() => window.factoryDetailFixture.attempted())) as {command: string};
  await page.evaluate(() => window.factoryDetailFixture.mount('ready')); await savedNotice().waitFor();
  assert.deepEqual(await posts(), []);
  await page.evaluate(() => window.factoryDetailFixture.storageFailure('none'));
  await retry.click(); await queued();
  assert.equal(((await posts())[0]?.body as {command_id: string}).command_id, attempted.command);

  // A replay refusal does not prove that the original uncertain effect never applied.
  await page.evaluate(() => window.factoryDetailFixture.mount('unknown')); await retry.click(); await lost();
  const unknown = (await posts())[0]; assert(unknown);
  const unknownRaw = await page.evaluate(() => window.factoryDetailFixture.saved()); assert(unknownRaw);
  await page.evaluate(() => window.factoryDetailFixture.fail(403));
  await retry.click(); await detail.getByText('Refused:', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved()), unknownRaw);
  assert.deepEqual((await posts())[1], unknown);
  await page.evaluate(() => { window.factoryDetailFixture.fail(0); window.factoryDetailFixture.mount('ready'); });
  await savedNotice().waitFor(); await retry.click(); await queued(); assert.deepEqual((await posts())[0], unknown);

  // Account switches neither expose nor replay another account's command.
  await page.evaluate(() => { window.factoryDetailFixture.account('a'); window.factoryDetailFixture.mount('unknown'); });
  await retry.click(); await lost(); const a = (await posts())[0]; assert(a);
  const aRaw = await page.evaluate(() => window.factoryDetailFixture.saved()); assert(aRaw);
  await page.evaluate(() => { window.factoryDetailFixture.account('b'); window.factoryDetailFixture.mount('unknown'); });
  await ready(); assert.equal(await savedNotice().count(), 0); assert.deepEqual(await posts(), []);
  await retry.click(); await lost(); const bRaw = await page.evaluate(() => window.factoryDetailFixture.saved()); assert(bRaw);
  await page.evaluate(() => { window.factoryDetailFixture.account('a'); window.factoryDetailFixture.mount('ready'); });
  await savedNotice().waitFor(); assert.deepEqual(await posts(), []); await retry.click(); await queued();
  assert.deepEqual((await posts())[0], a); assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved('b')), bRaw);
  await page.evaluate(() => { window.factoryDetailFixture.account('b'); window.factoryDetailFixture.mount('ready'); });
  await savedNotice().waitFor(); await retry.click(); await queued();

  // A late acknowledgement cannot settle a different captured command.
  await page.evaluate(() => { window.factoryDetailFixture.account('late'); window.factoryDetailFixture.mount('ready'); });
  await ready(); await page.evaluate(() => window.factoryDetailFixture.holdRetry()); await retry.click();
  await detail.getByText('Requested; awaiting confirmation.', {exact: true}).waitFor();
  const replacement = JSON.stringify({scope: 'declared:detail:late', repository: '9007199254740993', issue: '12', command: 'b'.repeat(32), run: 'a'.repeat(32)});
  await page.evaluate(raw => window.factoryDetailFixture.seed(raw), replacement);
  await detail.getByRole('button', {name: 'Refresh detail', exact: true}).click(); await savedNotice().waitFor();
  await page.evaluate(() => window.factoryDetailFixture.release());
  await detail.getByText('Saved retry changed while waiting.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.factoryDetailFixture.saved()), replacement);
  assert.equal((await posts()).length, 1);
  await retry.click(); await queued();
  assert.equal(((await posts())[1]?.body as {command_id: string}).command_id, 'b'.repeat(32));
}

import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {fixture, openSession, setupWorkspaceDriver} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

type PlanSeed = {objective: string; repository: string; state: 'draft' | 'accepted' | 'abandoned'};
async function openProjectPlans(page: Page) {
  const work = page.locator('soda-project-work:visible');
  const plans = work.locator('details.soda-project-plans');
  if ((await plans.count()) && !(await plans.evaluate((details) => (details as HTMLDetailsElement).open)))
    await plans.locator('summary').click();
  return work;
}
async function seedPlans(page: Page, plans: PlanSeed[]) {
  const ids = await page.evaluate(async (plans) => {
    const fixture = window.workspaceFixture;
    const ids: string[] = [];
    for (const [index, plan] of plans.entries()) {
      const intent = 'd' + (1000 + index).toString(16).padStart(24, '0');
      await fixture.request('/api/factory/intents', 'POST', {
        ...plan,
        intent,
        id: intent,
        ideas: [],
        tasks: [],
        max_issues: 10,
        planning_calls: 0,
      });
      ids.push(intent);
    }
    fixture.calls.length = 0;
    await fixture.api.refresh();
    return ids;
  }, plans);
  const work = await openProjectPlans(page);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  return ids;
}

async function writes(page: Page) {
  return page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length);
}

test('Project Work opens explicit requirements review for an imported native issue without creating work', async (t) => {
  const page = await fixture(t);
  const work = page.locator('soda-project-work:visible');
  await work
    .getByText('Adopt or reaccept an issue', {exact: true})
    .click()
    .catch(async () => {
      throw Error(await page.locator('.soda-workspace').innerText());
    });
  await work.getByLabel('Native issue number', {exact: true}).fill('13');
  await work.getByRole('button', {name: 'Open requirements review', exact: true}).click();
  const acceptance = work.getByRole('region', {name: 'Issue requirements acceptance'});
  await work.locator('soda-factory-work-journey').getByRole('button', {name: 'Refresh journey', exact: true}).waitFor();
  assert.equal(
    await acceptance
      .getByRole('link', {name: 'Review issue #13 in the native forge', exact: true})
      .getAttribute('href'),
    '/alice/Alpha/issues/13'
  );
  const requests = await page.evaluate(() => window.workspaceFixture.calls.map((call) => call.path));
  assert(requests.some((path) => path.endsWith('/api/repositories/7/factory/access')));
  assert(requests.some((path) => path.endsWith('/api/repositories/7/factory/issues/13')));
  assert.equal(await writes(page), 0, 'opening review must not accept an issue or create a plan');
  await work.getByLabel('Native issue number', {exact: true}).fill('not-an-issue');
  await work.getByRole('button', {name: 'Open requirements review', exact: true}).click();
  await work.getByRole('status').filter({hasText: 'Enter the native issue number'}).waitFor();
  assert.equal(await writes(page), 0);

  // An issue-zero verifier is reachable from ordinary Project Work, without
  // pretending it belongs to the imported issue or starting work on a read.
  await page.evaluate(async () => {
    window.workspaceFixture.enableVerificationRetry();
    await window.workspaceFixture.api.refresh();
  });
  const verification = work.getByRole('region', {name: 'Application verification needs attention'});
  await verification.getByRole('button', {name: 'Retry application verification', exact: true}).waitFor();
  assert.equal(await writes(page), 0, 'reading verification availability must not queue work');
  await verification.getByRole('button', {name: 'Retry application verification', exact: true}).click();
  await verification.getByRole('status').filter({hasText: 'Verification Retry queued'}).waitFor();
  assert.equal(
    await verification.getByRole('button', {name: 'Retry application verification', exact: true}).count(),
    0
  );
  const retryCalls = await page.evaluate(() =>
    window.workspaceFixture.calls.filter((call) => call.method === 'POST' && call.body?.action === 'retry')
  );
  assert.equal(retryCalls.length, 1);
  assert(retryCalls[0]?.path.endsWith(`/api/factory/runs/${'d'.repeat(32)}/actions`));
  assert.match(String(retryCalls[0]?.body?.command_id), /^[a-f0-9]{32}$/);
  assert.equal(retryCalls[0]?.body?.action, 'retry');
  assert.equal(await writes(page), 1, 'lost reply recovery must read the same command, never issue another Retry');
  await page.evaluate(async () => {
    window.workspaceFixture.calls.length = 0;
    window.workspaceFixture.enableVerificationRetry(true);
    await window.workspaceFixture.api.refresh();
  });
  await verification.getByRole('button', {name: 'Refresh verification', exact: true}).click();
  await verification.getByRole('button', {name: 'Request application repair', exact: true}).waitFor();
  assert.equal(
    await verification.getByRole('button', {name: 'Retry application verification', exact: true}).count(),
    0
  );
  assert.equal(await writes(page), 0, 'reading repair availability must not admit a diagnosis');
  await verification.getByRole('button', {name: 'Request application repair', exact: true}).click();
  await verification.getByRole('status').filter({hasText: 'No repair request was confirmed'}).waitFor();
  assert.doesNotMatch(
    await verification.innerText(),
    /Application repair requested/,
    'a foreign run receipt cannot acknowledge repair'
  );
  await page.evaluate(async () => {
    window.workspaceFixture.allowRepairReceipt();
    await window.workspaceFixture.api.refresh();
  });
  await verification.getByRole('button', {name: 'Refresh verification', exact: true}).click();
  await verification.getByRole('button', {name: 'Retry same repair request', exact: true}).click();
  await verification
    .getByRole('status')
    .filter({hasText: 'Application repair requested. Diagnosis is queued'})
    .waitFor();
  const repairCalls = await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method === 'POST'));
  assert.equal(repairCalls.length, 2);
  assert.deepEqual(
    repairCalls[0]?.body,
    repairCalls[1]?.body,
    'uncertain repair retains the exact command and action across refresh'
  );
  assert.equal(repairCalls[0]?.body?.action, 'repair_delivery');
  assert(repairCalls.every((call) => call.path.endsWith(`/api/factory/runs/${'d'.repeat(32)}/actions`)));
  assert.equal(await verification.getByRole('button', {name: 'Request application repair', exact: true}).count(), 0);
  // Authoritative progress survives the queued acknowledgment and advances
  // through GET-only observation, without resubmitting or a manual refresh.
  await page.evaluate(() => window.workspaceFixture.setRepairProgress('investigating'));
  const progress = verification.getByRole('region', {name: 'Application repair progress'});
  await progress.getByRole('status').filter({hasText: 'Strategist diagnosis is in progress.'}).waitFor();
  await progress
    .getByText('priority-state: Priority did not persist after the shared-state update.', {exact: true})
    .waitFor();
  assert.equal(await writes(page), 2, 'observing Strategist/judgment progress must not queue more work');
  const beforeSwitch = await page.evaluate(() => window.workspaceFixture.calls.length);
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  assert.equal(await page.getByRole('region', {name: 'Application verification needs attention'}).count(), 0);
  await page.waitForTimeout(4200);
  const afterSwitch = await page.evaluate((start) => window.workspaceFixture.calls.slice(start), beforeSwitch);
  assert(
    !afterSwitch.some((call) => call.path.endsWith(`/api/factory/runs/${'d'.repeat(32)}`)),
    'context change must retire repair polling'
  );
  // A later visit reads the adopted graph, without claiming that the app
  // itself has passed a new verification.
  await page.evaluate(() => window.workspaceFixture.setRepairProgress('adopted'));
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
  await progress
    .getByRole('status')
    .filter({hasText: 'Repair graph adopted. Fresh readiness and application verification are still required.'})
    .waitFor();
  await page.evaluate(() => window.workspaceFixture.setRepairProgress('investigating', true));
  await verification.getByRole('button', {name: 'Refresh verification', exact: true}).click();
  await verification
    .getByRole('status')
    .filter({hasText: 'Could not read current verification or repair progress.'})
    .waitFor();
  await progress.getByRole('status').filter({hasText: 'Repair graph adopted.'}).waitFor();
  // An independent retained Attention snapshot must show its actual low
  // confidence judgment, rather than treating worker success as adoption.
  const attentionPage = await fixture(t);
  await attentionPage.evaluate(async () => {
    window.workspaceFixture.enableVerificationRetry(true);
    window.workspaceFixture.setRepairProgress('attention');
    await window.workspaceFixture.api.refresh();
  });
  const attention = attentionPage.getByRole('region', {name: 'Application repair progress'});
  await attention.getByRole('status').filter({hasText: 'Application repair needs attention.'}).waitFor();
  await attention.getByText('Judgment: accept; acceptance probability 0.87; confidence 0.24.', {exact: true}).waitFor();
  assert.equal(await writes(attentionPage), 0, 'reading an Attention diagnosis cannot queue work');
});

test('cold Project entry loads scoped work, keeps the directory quiet, and opens planning explicitly', async (t) => {
  const page = await fixture(t);
  await seedPlans(page, [
    {objective: 'Improve search results', repository: '7', state: 'accepted'},
    {objective: 'Other repository plan', repository: '8', state: 'draft'},
  ]);
  const work = page.locator('soda-project-work:visible');
  await work.getByRole('heading', {name: 'Improve search results', exact: true}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Work', exact: true}).getAttribute('aria-pressed'), 'true');
  assert.equal(await work.getByText('Other repository plan', {exact: true}).count(), 0);
  const directory = page.locator('.soda-workspace-navigation');
  assert.doesNotMatch(await directory.innerText(), /Build|Edit|Tailnet policy|Factory needs|In this window/);
  assert.equal(await directory.locator('.soda-session-list').count(), 0);
  const requests = await page.evaluate(() => window.workspaceFixture.calls.map((call) => call.path));
  assert(requests.some((path) => path.endsWith('/api/factory/intents?limit=20&repository=7')));
  assert.equal(await writes(page), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.evaluate(() => {
    document.documentElement.style.colorScheme = 'light';
  });
  await captureSpacesComponent(page, 'project-work-light');
  await page.evaluate(() => {
    document.documentElement.style.colorScheme = 'dark';
  });
  await captureSpacesComponent(page, 'project-work-dark');
  const limits = work.locator('details.soda-verification-limits');
  await limits.locator('summary').click();
  const allowance = limits.getByRole('region', {name: 'Verification allowance'});
  await allowance
    .getByText('2 of 2 verification runs used; 148 active seconds used of 60 minutes.', {exact: true})
    .waitFor();
  assert.equal(await writes(page), 0, 'reading verification limits must not grant or start work');
  await allowance.getByRole('button', {name: 'Increase verification limits', exact: true}).click();
  await allowance.getByRole('status').filter({hasText: 'Increase at least one limit'}).waitFor();
  assert.equal(await writes(page), 0, 'unchanged verification limits must not issue a command');
  await allowance.getByLabel('Total verification runs', {exact: true}).fill('3');
  await allowance.getByRole('button', {name: 'Increase verification limits', exact: true}).click();
  await allowance.getByRole('status').filter({hasText: 'The current limits were not confirmed'}).waitFor();
  assert.equal(await allowance.getByLabel('Total verification runs', {exact: true}).isDisabled(), true);
  assert.equal(await allowance.getByLabel('Total active minutes', {exact: true}).isDisabled(), true);
  await page.evaluate(async () => {
    const fixture = window.workspaceFixture;
    const first = fixture.calls.find((call) => call.method === 'POST' && call.path.endsWith('/verification-allowance'));
    if (!first) throw Error('Missing first allowance change');
    // Another owner tab advances the head while this tab retains its lost reply.
    await fixture
      .request(first.path, 'POST', {
        command_id: 'e'.repeat(32),
        expected_revision: first.body?.command_id,
        limits: {runs: 4, active_minutes: 60},
      })
      .catch(() => undefined);
  });
  await allowance.getByRole('button', {name: 'Refresh verification limits', exact: true}).click();
  await allowance
    .getByText('2 of 4 verification runs used; 148 active seconds used of 60 minutes.', {exact: true})
    .waitFor();
  await allowance.getByRole('button', {name: 'Retry same limit change', exact: true}).click();
  await allowance
    .getByRole('status')
    .filter({hasText: 'Verification limits saved. Previous usage is retained.'})
    .waitFor();
  await allowance
    .getByText('2 of 4 verification runs used; 148 active seconds used of 60 minutes.', {exact: true})
    .waitFor();
  const limitCalls = await page.evaluate(() =>
    window.workspaceFixture.calls.filter(
      (call) => call.method === 'POST' && call.path.endsWith('/verification-allowance')
    )
  );
  assert.equal(limitCalls.length, 3);
  assert.deepEqual(limitCalls[0], limitCalls[2], 'a lost reply must retry the same immutable allowance command');
  assert.match(String(limitCalls[0]?.body?.command_id), /^[a-f0-9]{32}$/);
  assert.equal(limitCalls[0]?.body?.expected_revision, '');
  assert.deepEqual(limitCalls[0]?.body?.limits, {runs: 3, active_minutes: 60});
  assert.equal(await allowance.getByLabel('Total verification runs', {exact: true}).inputValue(), '4');
  assert.equal(await writes(page), 3, 'increasing limits must not launch verification');
  await captureSpacesComponent(page, 'project-work-verification-limits');
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  assert.equal(
    await page.getByRole('region', {name: 'Verification allowance'}).count(),
    0,
    'verification limits stay scoped to their accepted Project intent'
  );
  await work.getByRole('button', {name: 'New work', exact: true}).click();
  await page.getByLabel('Describe what you want to build', {exact: true}).waitFor();
  assert.equal(new URL(page.url()).pathname, '/workspace/new');
  assert.equal(await writes(page), 3, 'entering planning must not create a Project or work');
});

test('Saved plans exposes unassigned and closed plans and resumes accepted progress through its original route', async (t) => {
  const page = await fixture(t);
  const ids = await seedPlans(page, [
    {objective: 'A draft without a repository', repository: '0', state: 'draft'},
    {objective: 'Improve search results', repository: '7', state: 'accepted'},
    {objective: 'An earlier direction', repository: '8', state: 'abandoned'},
  ]);
  await page.getByRole('button', {name: 'Saved plans', exact: true}).click();
  const work = page.locator('soda-project-work:visible');
  await work.getByText('Not linked to a repository', {exact: true}).waitFor();
  assert.equal(await work.locator('.soda-work-item').count(), 3);
  assert.equal(await work.getByText('Closed', {exact: true}).count(), 1);
  assert.equal(await work.getByText('alice/Beta', {exact: true}).count(), 1);
  await work.getByRole('button', {name: 'View progress: Improve search results', exact: true}).click();
  await page.locator('soda-factory-work-overview').getByText('Improve search results', {exact: true}).waitFor();
  assert.equal(new URL(page.url()).pathname, '/workspace/requests/' + ids[1]);
  assert.equal(await writes(page), 0, 'resuming a request must only read existing work');
});

test('partial plan inventory survives a failed next page and retry reaches the remaining work', async (t) => {
  const page = await fixture(t);
  await seedPlans(
    page,
    Array.from({length: 21}, (_, index) => ({
      objective: 'Planned change ' + (index + 1),
      repository: '7',
      state: 'draft',
    }))
  );
  const work = page.locator('soda-project-work:visible');
  await work.getByText('More plans are available.', {exact: false}).waitFor();
  assert.equal(await work.locator('.soda-work-item').count(), 20);
  await page.evaluate(() => window.workspaceFixture.setStatus(503));
  await work.getByRole('button', {name: 'Load more plans', exact: true}).click();
  await work.getByText('Could not load more plans.', {exact: false}).waitFor();
  assert.equal(await work.locator('.soda-work-item').count(), 20);
  assert.equal(await work.locator('.soda-work-empty').count(), 0);
  await page.evaluate(() => window.workspaceFixture.setStatus(200));
  await work.getByRole('button', {name: 'Load more plans', exact: true}).click();
  await work.getByRole('heading', {name: 'Planned change 21', exact: true}).waitFor();
  assert.equal(await work.locator('.soda-work-item').count(), 21);
  assert.equal(await work.getByRole('button', {name: 'Load more plans', exact: true}).count(), 0);
  assert.equal(await writes(page), 0);
});

test('Work, Project and Terminals navigation retain the exact terminal owner and original session', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const owner = await page.locator('.soda-workspace-terminal:visible').elementHandle();
  const renderer = await page.locator('.soda-workspace-terminal:visible .xterm').elementHandle();
  const socket = await page.evaluate(() => String(window.workspaceFixture.sockets[0]?.url));
  await page.getByRole('button', {name: 'Work', exact: true}).click();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  assert.equal(new URL(page.url()).pathname, '/workspace');
  assert.equal(new URL(page.url()).searchParams.get('repository'), '8');
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  assert(await owner?.evaluate((node) => node.isConnected && !node.hasAttribute('hidden')));
  assert(await renderer?.evaluate((node) => node.isConnected));
  assert.equal(await page.locator('.soda-selected-project h1').innerText(), 'alice/Beta');
  const observed = await page.evaluate(() => ({
    count: window.workspaceFixture.sockets.length,
    closed: window.workspaceFixture.sockets[0]?.closed,
    url: String(window.workspaceFixture.sockets[0]?.url),
    attached: window.workspaceFixture.sockets[0]?.sent
      .filter((frame) => frame.action === 'attach')
      .map((frame) => frame.id),
  }));
  assert.deepEqual(observed, {count: 1, closed: 0, url: socket, attached: ['a'.repeat(32)]});
  assert.equal(await writes(page), 0);
});

test('native page entry opens Work once and preserves a deliberately selected terminal view', async (t) => {
  const page = await fixture(t, 'native');
  await page.evaluate(async () => {
    await window.workspaceFixture.api.refresh();
    window.workspaceFixture.presentRoute('/workspace');
  });
  await (
    await openProjectPlans(page)
  )
    .getByRole('heading', {name: 'What would you like to change?', exact: true})
    .waitFor();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.evaluate(() => window.workspaceFixture.presentRoute('/workspace'));
  assert.equal(await page.getByRole('button', {name: 'Terminals', exact: true}).getAttribute('aria-pressed'), 'true');
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await openSession(page, 'Build');
  const renderer = await page.locator('.soda-workspace-terminal:visible .xterm').elementHandle();
  await page.evaluate(() => {
    window.workspaceFixture.presentRoute('/explore/repos');
    window.workspaceFixture.presentRoute('/workspace');
  });
  assert.equal(await page.getByRole('button', {name: 'Terminals', exact: true}).getAttribute('aria-pressed'), 'true');
  assert(await renderer?.evaluate((node) => node.isConnected));
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 0);
  assert.equal(await writes(page), 0);
});

test('compact Projects navigation opens another Project work view without starting a terminal', async (t) => {
  const page = await fixture(t);
  await page.setViewportSize({width: 390, height: 740});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  assert.equal(new URL(page.url()).pathname, '/workspace');
  assert.equal(new URL(page.url()).searchParams.get('repository'), '8');
  await (
    await openProjectPlans(page)
  )
    .getByRole('heading', {name: 'What would you like to change?', exact: true})
    .waitFor();
  assert.equal(await page.locator('.soda-selected-project h1').innerText(), 'alice/Beta');
  assert(await page.getByRole('button', {name: 'Terminals', exact: true}).isVisible());
  await captureSpacesComponent(page, 'project-work-compact');
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await writes(page), 0);
});

test('Factory watch stays connected across project filtering and work-view changes', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    window.workspaceFixture.enableFactoryRun();
    await window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: /^Watch · soda-coder/}).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  const owner = await page.locator('soda-factory-watch').elementHandle();
  const renderer = await page.locator('.soda-factory .xterm').elementHandle();
  await page.getByLabel('Find a project', {exact: true}).fill('Beta');
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  assert(await owner?.evaluate((node) => node.isConnected));
  assert(await renderer?.evaluate((node) => node.isConnected));
  await page.getByLabel('Find a project', {exact: true}).fill('');
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((socket) => socket.closed)), [0]);
  assert.equal(await writes(page), 0);
});

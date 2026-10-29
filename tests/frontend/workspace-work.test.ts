import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {fixture, openSession, setupWorkspaceDriver} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

type PlanSeed = {
  objective: string;
  repository: string;
  state: 'draft' | 'accepted' | 'abandoned';
  tasks?: Array<{key: string; objective: string; outcomes: string[]; idea_ids: string[]}>;
  effective_outcomes?: string[];
};
async function openProjectPlans(page: Page) {
  const tab = page.getByRole('button', {name: 'Work', exact: true});
  if (await tab.isVisible()) await tab.click();
  const work = page.locator('soda-project-work:visible');
  const plans = work.locator('details.soda-project-plans');
  if ((await plans.count()) && !(await plans.evaluate((details) => (details as HTMLDetailsElement).open)))
    await plans.locator(':scope > summary').click();
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
        tasks: plan.tasks ?? [],
        ...(plan.effective_outcomes ? {effective_outcomes: plan.effective_outcomes} : {}),
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
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openProjectPlans(page);
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
  // An independent retained Attention snapshot must show its actual low
  // confidence judgment, rather than treating worker success as adoption.
  const attentionPage = await fixture(t);
  await attentionPage.evaluate(async () => {
    window.workspaceFixture.enableVerificationRetry(true);
    window.workspaceFixture.setRepairProgress('attention');
    await window.workspaceFixture.api.refresh();
  });
  await openProjectPlans(attentionPage);
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
  for (const theme of ['light', 'dark'])
    for (const width of [800, 390]) {
      await page.setViewportSize({width, height: 900});
      await page.evaluate((value) => {
        document.documentElement.style.colorScheme = value;
      }, theme);
      await captureSpacesComponent(page, `project-work-${theme}-${width}`);
    }
  await page.setViewportSize({width: 1440, height: 1000});
  await page.evaluate(() => {
    document.documentElement.style.colorScheme = 'dark';
  });
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
  await openProjectPlans(page);
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

test('native page entry opens Feed and preserves a deliberately selected terminal view', async (t) => {
  const page = await fixture(t, 'native');
  await page.evaluate(async () => {
    await window.workspaceFixture.api.refresh();
    window.workspaceFixture.presentRoute('/workspace');
  });
  assert.equal(await page.getByRole('button', {name: 'Feed', exact: true}).getAttribute('aria-pressed'), 'true');
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
  await openProjectPlans(page);
  await page.getByRole('button', {name: /^Watch · Coding agent/}).click();
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

test('Feed, App and Work share one project and retain their owners; returning opens the saved App view', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  const feedTab = page.getByRole('button', {name: 'Feed', exact: true});
  assert.equal(await feedTab.getAttribute('aria-pressed'), 'true');
  await page.getByRole('region', {name: 'Project conversation', exact: true}).waitFor();
  const feed = await page.locator('soda-factory-feed').elementHandle();
  assert.equal(await page.locator('soda-applications').count(), 0, 'Feed cannot mount an unvisited app');
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  assert.equal(await page.locator('soda-applications').count(), 0, 'Terminals cannot mount an unvisited app');
  // The Feed reads only the launch list (for Open app ↗); no app is read or inspected.
  assert.deepEqual(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.path.includes('/applications/'))),
    [],
    'unvisited applications cannot be read or inspected'
  );
  await feedTab.click();
  await page.getByRole('button', {name: 'App', exact: true}).click();
  await page.getByRole('region', {name: 'Project application', exact: true}).waitFor();
  const app = await page.locator('soda-applications').elementHandle();
  assert(await feed?.evaluate((node) => node.isConnected));
  await page.locator('soda-applications').evaluate((node) => {
    node.dispatchEvent(new CustomEvent('soda-project-feed-open', {bubbles: true, detail: {repository: '8'}}));
  });
  assert.equal(await page.getByRole('button', {name: 'App', exact: true}).getAttribute('aria-pressed'), 'true');
  await page.getByRole('button', {name: 'View project Feed', exact: true}).click();
  assert.equal(await feedTab.getAttribute('aria-pressed'), 'true', 'App guidance returns to this project’s Feed');
  assert(await app?.evaluate((node) => node === document.querySelector('soda-applications')));
  assert.equal(await writes(page), 0, 'App guidance navigation cannot create work');
  await page.getByRole('button', {name: 'Work', exact: true}).click();
  await page.getByRole('region', {name: 'Plans and delivery', exact: true}).waitFor();
  assert(await app?.evaluate((node) => node.isConnected));
  await feedTab.click();
  assert(await feed?.evaluate((node) => node === document.querySelector('soda-factory-feed')));
  assert(await app?.evaluate((node) => node === document.querySelector('soda-applications')));
  await page.getByRole('button', {name: 'App', exact: true}).click();
  await page.evaluate(() => {
    const workspace = document.querySelector('soda-spaces');
    if (!workspace) throw Error('missing workspace');
    workspace.dispatchEvent(
      new CustomEvent('soda-application-open', {bubbles: true, detail: {repository: '7', deliveryId: 'e'.repeat(32)}})
    );
  });
  await page.waitForFunction(
    () =>
      document.querySelector<HTMLElement & {requestedDelivery: string}>('soda-applications')?.requestedDelivery ===
      'e'.repeat(32)
  );
  assert.equal(await writes(page), 0, 'view selection cannot create work or launch an application');
  await page.reload();
  await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async () => {
    window.workspaceFixture = window.createWorkspaceFixture('page');
    await window.workspaceFixture.api.ready;
    await window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: 'App', exact: true}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'App', exact: true}).getAttribute('aria-pressed'), 'true');
  assert.equal(
    await page
      .locator('soda-applications')
      .evaluate((node) => (node as HTMLElement & {requestedDelivery: string}).requestedDelivery),
    'e'.repeat(32),
    'reload retains the exact delivery instead of substituting a current app'
  );
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  assert.equal(await feedTab.getAttribute('aria-pressed'), 'true', 'a different project has its own view preference');
  assert.equal(await page.locator('soda-applications').count(), 0, 'a different project has not visited App');
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
  assert.equal(await page.getByRole('button', {name: 'App', exact: true}).getAttribute('aria-pressed'), 'true');
  const restoredApp = await page.locator('soda-applications').elementHandle();
  await page.evaluate(async () => {
    window.workspaceFixture.setUser('2');
    await window.workspaceFixture.api.refresh();
  });
  assert.equal(await page.locator('soda-applications').count(), 0, 'identity invalidation retires the app owner');
  assert.equal(await restoredApp?.evaluate((node) => node.isConnected), false);
});

test('Draft list stays truthful without a recorded plan check: unplanned, tasked, and scoped drafts', async (t) => {
  const page = await fixture(t);
  const taskKey = 'd' + (3000).toString(16).padStart(24, '0');
  const outcome = 'd' + (3001).toString(16).padStart(24, '0');
  const ids = await seedPlans(page, [
    {objective: 'Unplanned draft without scope', repository: '7', state: 'draft'},
    {
      objective: 'Tasks without a recorded check stay Draft',
      repository: '7',
      state: 'draft',
      tasks: [
        {
          key: taskKey,
          objective: 'Do the scoped work',
          outcomes: ['Demonstrate the selected outcome'],
          idea_ids: ['idea-1'],
        },
      ],
    },
    {
      objective: 'Saved scope without a recorded check stays Draft',
      repository: '7',
      state: 'draft',
      tasks: [
        {
          key: taskKey,
          objective: 'Do the scoped work',
          outcomes: ['Demonstrate the selected outcome'],
          idea_ids: ['idea-1'],
        },
      ],
      effective_outcomes: [outcome],
    },
  ]);
  const work = page.locator('soda-project-work:visible');
  for (const objective of [
    'Unplanned draft without scope',
    'Tasks without a recorded check stay Draft',
    'Saved scope without a recorded check stays Draft',
  ])
    await work.getByRole('heading', {name: objective, exact: true}).waitFor();
  assert.equal(await work.locator('.soda-work-item').count(), 3);
  assert.equal(await work.getByText('Draft', {exact: true}).count(), 3);
  assert.equal(await work.getByText('Checked', {exact: true}).count(), 0);
  assert.equal(await work.getByText('Blocked', {exact: true}).count(), 0);
  for (const objective of [
    'Unplanned draft without scope',
    'Tasks without a recorded check stay Draft',
    'Saved scope without a recorded check stays Draft',
  ])
    await work.getByRole('button', {name: `Continue planning: ${objective}`, exact: true}).waitFor();
  assert.equal(await work.getByRole('button', {name: /^Review check:/, exact: false}).count(), 0);
  assert.equal(await work.getByRole('button', {name: /^Resolve blockers:/, exact: false}).count(), 0);
  const calls = await page.evaluate(() =>
    window.workspaceFixture.calls.filter((call) => call.method === 'GET').map((call) => call.path)
  );
  assert(calls.some((path) => path.endsWith('/api/factory/intents?limit=20&repository=7')));
  assert.equal(await writes(page), 0, 'reading saved scope must not start planning or a check');
  await work
    .getByRole('button', {name: 'Continue planning: Saved scope without a recorded check stays Draft', exact: true})
    .click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  assert.equal(new URL(page.url()).pathname, '/workspace/requests/' + ids[2]);
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'Saved scope without a recorded check stays Draft';
  });
  assert.equal(await writes(page), 0, 'resuming a draft must only read existing work');
});

test('Work return replays the recorded plan check without creating work', async (t) => {
  const page = await fixture(t);
  const ids = await seedPlans(page, [
    {objective: 'Unplanned draft stays Draft', repository: '7', state: 'draft'},
    {objective: 'Checked draft shows its recorded check', repository: '7', state: 'draft'},
    {objective: 'Needs-correction draft names its blockers', repository: '7', state: 'draft'},
    {objective: 'Plan-ready draft waits for its check', repository: '7', state: 'draft'},
    {objective: 'Needs-decision draft waits for its owner', repository: '7', state: 'draft'},
  ]);
  // Planning and checks were recorded earlier through the Describe flow; the
  // Work return below must only replay them with owner GETs.
  await page.evaluate(async (ids) => {
    const fixture = window.workspaceFixture;
    for (const intent of ids.slice(1)) {
      await fixture.request('/api/factory/intents/' + intent + '/plan', 'POST', {
        revision: intent,
        operation: 'propose_intent',
      });
    }
    fixture.calls.length = 0;
    await fixture.api.refresh();
  }, ids);
  const work = await openProjectPlans(page);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await work.getByText('Checked', {exact: true}).waitFor();
  await work.getByText('Needs correction', {exact: true}).waitFor();
  await work.getByText('Plan ready', {exact: true}).waitFor();
  await work.getByText('Needs your decision', {exact: true}).waitFor();
  assert.equal(await work.locator('.soda-work-state[data-state="needs-decision"]').count(), 1);
  assert.equal(await work.locator('.soda-work-item').count(), 5);
  assert.equal(await work.getByText('Draft', {exact: true}).count(), 1);
  await work.getByRole('button', {name: 'Continue planning: Unplanned draft stays Draft', exact: true}).waitFor();
  await work.getByRole('button', {name: 'Review check: Checked draft shows its recorded check', exact: true}).waitFor();
  await work
    .getByRole('button', {name: 'Resolve blockers: Needs-correction draft names its blockers', exact: true})
    .waitFor();
  await work
    .getByRole('button', {name: 'Check walkthrough: Plan-ready draft waits for its check', exact: true})
    .waitFor();
  await work
    .getByRole('button', {name: 'Review decision: Needs-decision draft waits for its owner', exact: true})
    .waitFor();
  assert.equal(await work.getByRole('button', {name: /^Continue planning: Needs-decision/}).count(), 0);
  const calls = await page.evaluate(() =>
    window.workspaceFixture.calls.map((call) => ({method: call.method, path: call.path}))
  );
  for (const intent of ids)
    assert(
      calls.some((call) => call.method === 'GET' && call.path.endsWith('/api/factory/intents/' + intent + '/plan')),
      'owner replays the recorded plan for ' + intent
    );
  assert.equal(await writes(page), 0, 'replaying recorded plan/check state must not POST');
  await work.getByRole('button', {name: 'Review check: Checked draft shows its recorded check', exact: true}).click();
  await page.waitForFunction(() => window.location.pathname.startsWith('/workspace/requests/d'));
  assert.equal(new URL(page.url()).pathname, '/workspace/requests/' + ids[1]);
  assert.equal(await writes(page), 0, 'resuming a checked draft must only read existing work');
});

test('Work return keeps edited, unbound, and decided work neutral without claiming a check', async (t) => {
  const page = await fixture(t);
  const ids = await seedPlans(page, [
    {objective: 'Unplanned draft stays Draft', repository: '7', state: 'draft'},
    {objective: 'Edited draft stays Draft after its direction changes', repository: '7', state: 'draft'},
    {objective: 'Accepted direction stays Accepted', repository: '7', state: 'accepted'},
    {objective: 'Unbound draft stays Draft without a plan read', repository: '0', state: 'draft'},
  ]);
  await page.evaluate(async (ids) => {
    const fixture = window.workspaceFixture;
    // Plan the edited draft, then change its direction: the recorded check
    // belongs to the earlier revision and must not label the new head.
    await fixture.request('/api/factory/intents/' + ids[1] + '/plan', 'POST', {
      revision: ids[1],
      operation: 'propose_intent',
    });
    const successor = 'd' + (2000).toString(16).padStart(24, '0');
    await fixture.request('/api/factory/intents', 'POST', {
      objective: 'Edited draft stays Draft after its direction changes',
      repository: '7',
      state: 'draft',
      intent: ids[1],
      id: successor,
      ideas: [],
      tasks: [],
      max_issues: 10,
      planning_calls: 0,
    });
    fixture.calls.length = 0;
    await fixture.api.refresh();
  }, ids);
  const work = await openProjectPlans(page);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await work
    .getByRole('heading', {name: 'Edited draft stays Draft after its direction changes', exact: true})
    .waitFor();
  assert.equal(await work.locator('.soda-work-item').count(), 3);
  assert.equal(await work.getByText('Draft', {exact: true}).count(), 2);
  assert.equal(await work.getByText('Accepted', {exact: true}).count(), 1);
  assert.equal(await work.getByText('Checked', {exact: true}).count(), 0);
  assert.equal(await work.getByText('Needs correction', {exact: true}).count(), 0);
  assert.equal(await work.getByText('Plan ready', {exact: true}).count(), 0);
  await work
    .getByRole('button', {name: 'Continue planning: Edited draft stays Draft after its direction changes', exact: true})
    .waitFor();
  await work.getByRole('button', {name: 'View progress: Accepted direction stays Accepted', exact: true}).waitFor();
  assert.equal(await work.getByRole('button', {name: /^Review check:/, exact: false}).count(), 0);
  assert.equal(await work.getByRole('button', {name: /^Resolve blockers:/, exact: false}).count(), 0);
  const calls = await page.evaluate(() =>
    window.workspaceFixture.calls.map((call) => ({method: call.method, path: call.path}))
  );
  for (const intent of ids.slice(0, 2))
    assert(
      calls.some((call) => call.method === 'GET' && call.path.endsWith('/api/factory/intents/' + intent + '/plan')),
      'owner replays the recorded plan for ' + intent
    );
  assert(
    !calls.some((call) => call.path.endsWith('/api/factory/intents/' + ids[2] + '/plan')),
    'decided work keeps its Accepted state without a plan read'
  );
  assert.equal(await writes(page), 0, 'guard reads must not start planning or a check');
  // Saved plans shows the unbound draft neutrally without reading a plan.
  await page.evaluate(() => {
    window.workspaceFixture.calls.length = 0;
  });
  await page.getByRole('button', {name: 'Saved plans', exact: true}).click();
  const saved = page.locator('soda-project-work:visible');
  await saved.getByText('Not linked to a repository', {exact: true}).waitFor();
  await saved.getByRole('heading', {name: 'Unbound draft stays Draft without a plan read', exact: true}).waitFor();
  await saved.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await saved
    .getByRole('button', {name: 'Continue planning: Unbound draft stays Draft without a plan read', exact: true})
    .waitFor();
  assert.equal(await saved.getByText('Accepted', {exact: true}).count(), 1);
  const savedCalls = await page.evaluate(() =>
    window.workspaceFixture.calls.map((call) => ({method: call.method, path: call.path}))
  );
  assert(
    !savedCalls.some((call) => call.path.endsWith('/api/factory/intents/' + ids[3] + '/plan')),
    'unbound drafts stay Draft without a plan read'
  );
  assert(
    !savedCalls.some((call) => call.path.endsWith('/api/factory/intents/' + ids[2] + '/plan')),
    'decided work keeps its Accepted state without a plan read'
  );
  assert.equal(await writes(page), 0, 'saved-plans replay must not start planning or a check');
});

async function seedCheckedDraft(page: Page, objective: string) {
  const ids = await seedPlans(page, [{objective, repository: '7', state: 'draft'}]);
  const intent = ids[0];
  if (!intent) throw Error('missing seeded intent');
  await page.evaluate(async (intent) => {
    const fixture = window.workspaceFixture;
    await fixture.request('/api/factory/intents/' + intent + '/plan', 'POST', {
      revision: intent,
      operation: 'propose_intent',
    });
    fixture.calls.length = 0;
    await fixture.api.refresh();
  }, intent);
  const work = await openProjectPlans(page);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await work.getByText('Checked', {exact: true}).waitFor();
  return {work, intent};
}

async function failPlanReads(page: Page, intent: string, status: number) {
  await page.evaluate(
    ({intent, status}) => {
      const original = window.fetch.bind(window);
      Object.defineProperty(window, 'fetch', {
        configurable: true,
        value: (input: RequestInfo | URL, init?: RequestInit) => {
          const url = String(input);
          if (url.includes('/api/factory/intents/' + intent + '/plan') && (init?.method || 'GET') === 'GET')
            return Promise.resolve(new Response(null, {status}));
          return original(input, init);
        },
      });
    },
    {intent, status}
  );
}

for (const status of [404, 409, 503])
  test(`a formerly Checked draft stays neutral when its plan replay returns ${status}`, async (t) => {
    const page = await fixture(t);
    const objective = 'Checked draft with a failed replay';
    const {work, intent} = await seedCheckedDraft(page, objective);
    await failPlanReads(page, intent, status);
    await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
    await work.getByText('Draft', {exact: true}).waitFor();
    assert.equal(await work.getByText('Checked', {exact: true}).count(), 0, 'a failed replay never retains Checked');
    assert.equal(await work.getByText('Needs correction', {exact: true}).count(), 0);
    assert.equal(await work.getByText('Plan ready', {exact: true}).count(), 0);
    await work.getByRole('button', {name: 'Continue planning: ' + objective, exact: true}).waitFor();
    assert.equal(await work.locator('.soda-work-feedback').count(), 0, 'a failed replay never surfaces a list error');
    assert.equal(await writes(page), 0, 'replaying a failed plan must not start planning or a check');
  });

test('a formerly Checked draft stays neutral when its recorded revision no longer matches', async (t) => {
  const page = await fixture(t);
  const objective = 'Checked draft with a superseded replay';
  const {work, intent} = await seedCheckedDraft(page, objective);
  await page.evaluate((intent) => {
    const original = window.fetch.bind(window);
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const response = await original(input, init);
        if (
          url.includes('/api/factory/intents/' + intent + '/plan') &&
          (init?.method || 'GET') === 'GET' &&
          response.ok
        ) {
          const replay = (await response.json()) as Record<string, unknown>;
          return Response.json({...replay, revision: 'd' + 'f'.repeat(24)});
        }
        return response;
      },
    });
  }, intent);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await work.getByText('Draft', {exact: true}).waitFor();
  // Pre-clear shows Draft immediately while the replay is still in flight
  // (busy Loading feedback). Wait for the refresh to settle before asserting
  // no list error surfaces.
  await page.waitForFunction(
    () => document.querySelectorAll('soda-project-work .soda-work-feedback').length === 0,
    undefined,
    {timeout: 2000}
  );
  assert.equal(
    await work.getByText('Checked', {exact: true}).count(),
    0,
    'a mismatched revision never retains Checked'
  );
  await work.getByRole('button', {name: 'Continue planning: ' + objective, exact: true}).waitFor();
  assert.equal(await work.locator('.soda-work-feedback').count(), 0, 'a mismatched replay never surfaces a list error');
  assert.equal(await writes(page), 0, 'replaying a mismatched revision must not start planning or a check');
});

for (const status of [401, 403])
  test(`a ${status} plan replay routes through the authority-loss path`, async (t) => {
    const page = await fixture(t);
    const {work, intent} = await seedCheckedDraft(page, 'Checked draft with a lost authority');
    await page.evaluate(
      ({intent, status}) => {
        (window as unknown as {authorityLost: unknown[]}).authorityLost = [];
        window.addEventListener('soda-work-authority-lost', (event) => {
          (window as unknown as {authorityLost: unknown[]}).authorityLost.push((event as CustomEvent).detail);
        });
        const original = window.fetch.bind(window);
        Object.defineProperty(window, 'fetch', {
          configurable: true,
          value: (input: RequestInfo | URL, init?: RequestInit) => {
            const url = String(input);
            if (url.includes('/api/factory/intents/' + intent + '/plan') && (init?.method || 'GET') === 'GET')
              return Promise.resolve(new Response(null, {status}));
            return original(input, init);
          },
        });
      },
      {intent, status}
    );
    await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
    await page.waitForFunction(() => (window as unknown as {authorityLost: unknown[]}).authorityLost.length > 0);
    assert.deepEqual(
      await page.evaluate(() => (window as unknown as {authorityLost: unknown[]}).authorityLost),
      [{reconnectRequired: status === 401}],
      'plan authority loss reuses the existing reconnect path'
    );
    assert.equal(await page.locator('.soda-work-item').count(), 0, 'authority loss retires the work list');
    assert.equal(await writes(page), 0, 'authority loss must not start planning or a check');
  });

test('account and project changes retire stale Checked work without writes', async (t) => {
  const page = await fixture(t);
  await seedCheckedDraft(page, 'Checked draft retired on change');
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  // A fresh project restores its own Feed surface, so open Work before
  // asserting Beta shows its own empty list instead of inherited checks.
  await page.getByRole('button', {name: 'Work', exact: true}).click();
  const beta = page.locator('soda-project-work:visible');
  await beta.getByRole('heading', {name: 'What would you like to change?', exact: true}).waitFor();
  await beta.getByText('No plans linked to this project yet', {exact: true}).waitFor();
  assert.equal(await beta.locator('.soda-work-item').count(), 0, 'another project never inherits Checked work');
  assert.equal(await beta.getByText('Checked', {exact: true}).count(), 0);
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
  const alpha = page.locator('soda-project-work:visible');
  await alpha.getByText('Checked', {exact: true}).waitFor();
  await page.evaluate(async () => {
    window.workspaceFixture.setUser('2');
    await window.workspaceFixture.api.refresh();
  });
  await page.waitForFunction(() => document.querySelectorAll('.soda-work-item').length === 0);
  assert.equal(await page.getByText('Checked', {exact: true}).count(), 0, 'an account change retires stale checks');
  assert.equal(await writes(page), 0, 'change-driven retirement must not start planning or a check');
});

test('delayed plan replays never apply after a project change', async (t) => {
  const page = await fixture(t);
  const {work, intent} = await seedCheckedDraft(page, 'Checked draft raced by a project change');
  await page.evaluate((intent) => {
    (window as unknown as {planGate: Promise<void>; openPlanGate: () => void}).planGate = new Promise<void>(
      (resolve) => {
        (window as unknown as {openPlanGate: () => void}).openPlanGate = resolve;
      }
    );
    const original = window.fetch.bind(window);
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/api/factory/intents/' + intent + '/plan') && (init?.method || 'GET') === 'GET')
          await (window as unknown as {planGate: Promise<void>}).planGate;
        return original(input, init);
      },
    });
  }, intent);
  await page.evaluate(() => {
    window.workspaceFixture.calls.length = 0;
  });
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await page.waitForFunction(() =>
    window.workspaceFixture.calls.some((call) => call.path.includes('/api/factory/intents?'))
  );
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  // A fresh project restores its own Feed surface; open Work so the gated
  // Alpha replay has a visible foreign list it must never land in.
  await page.getByRole('button', {name: 'Work', exact: true}).click();
  const beta = page.locator('soda-project-work:visible');
  await beta.getByRole('heading', {name: 'What would you like to change?', exact: true}).waitFor();
  await beta.getByText('No plans linked to this project yet', {exact: true}).waitFor();
  await page.evaluate(() => (window as unknown as {openPlanGate: () => void}).openPlanGate());
  assert.equal(await beta.locator('.soda-work-item').count(), 0, 'the late replay never lands in another project');
  assert.equal(await beta.getByText('Checked', {exact: true}).count(), 0);
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
  const alpha = page.locator('soda-project-work:visible');
  await alpha.getByText('Checked', {exact: true}).waitFor();
  assert.equal(await writes(page), 0, 'a discarded replay must not start planning or a check');
});

test('delayed plan replays never apply after an account change', async (t) => {
  const page = await fixture(t);
  const {work, intent} = await seedCheckedDraft(page, 'Checked draft raced by an account change');
  await page.evaluate((intent) => {
    (window as unknown as {planGate: Promise<void>; openPlanGate: () => void}).planGate = new Promise<void>(
      (resolve) => {
        (window as unknown as {openPlanGate: () => void}).openPlanGate = resolve;
      }
    );
    const original = window.fetch.bind(window);
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/api/factory/intents/' + intent + '/plan') && (init?.method || 'GET') === 'GET')
          await (window as unknown as {planGate: Promise<void>}).planGate;
        return original(input, init);
      },
    });
  }, intent);
  await page.evaluate(() => {
    window.workspaceFixture.calls.length = 0;
  });
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await page.waitForFunction(() =>
    window.workspaceFixture.calls.some((call) => call.path.includes('/api/factory/intents?'))
  );
  await page.evaluate(async () => {
    window.workspaceFixture.setUser('2');
    await window.workspaceFixture.api.refresh();
  });
  await page.evaluate(() => (window as unknown as {openPlanGate: () => void}).openPlanGate());
  await page.waitForFunction(() => document.querySelectorAll('.soda-work-item').length === 0);
  assert.equal(await page.getByText('Checked', {exact: true}).count(), 0, 'the late replay never lands after logout');
  assert.equal(await writes(page), 0, 'a discarded replay must not start planning or a check');
});

test('paginated plan replay bounds parallel reads and fetches only the new page', async (t) => {
  const page = await fixture(t);
  const ids = await seedPlans(
    page,
    Array.from({length: 21}, (_, index) => ({
      objective: 'Planned change ' + (index + 1),
      repository: '7',
      state: 'draft' as const,
    }))
  );
  const last = ids[20];
  if (!last) throw Error('missing final intent');
  await page.waitForFunction(
    () =>
      window.workspaceFixture.calls.filter((call) => call.method === 'GET' && call.path.includes('/plan')).length >= 20
  );
  await page.evaluate(() => {
    (window as unknown as {planFlight: {inflight: number; max: number}}).planFlight = {inflight: 0, max: 0};
    const original = window.fetch.bind(window);
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/api/factory/intents/') && url.includes('/plan') && (init?.method || 'GET') === 'GET') {
          const flight = (window as unknown as {planFlight: {inflight: number; max: number}}).planFlight;
          flight.inflight += 1;
          flight.max = Math.max(flight.max, flight.inflight);
          try {
            await new Promise((resolve) => setTimeout(resolve, 25));
            return await original(input, init);
          } finally {
            flight.inflight -= 1;
          }
        }
        return original(input, init);
      },
    });
    window.workspaceFixture.calls.length = 0;
  });
  const work = await openProjectPlans(page);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await work.getByText('More plans are available.', {exact: false}).waitFor();
  await page.waitForFunction(
    () =>
      window.workspaceFixture.calls.filter((call) => call.method === 'GET' && call.path.includes('/plan')).length >= 20
  );
  assert.equal(await work.locator('.soda-work-item').count(), 20);
  // Let the full refresh settle (Refresh enabled again) before measuring the
  // paginated load, so in-flight replays cannot overlap the new-page reads.
  await page.waitForFunction(() => {
    const refresh = [...document.querySelectorAll('soda-project-work button')].find((button) =>
      button.textContent?.includes('Refresh plans')
    );
    return refresh instanceof HTMLButtonElement && !refresh.disabled;
  });
  const first = await page.evaluate(() => ({
    max: (window as unknown as {planFlight: {inflight: number; max: number}}).planFlight.max,
    planReads: window.workspaceFixture.calls.filter((call) => call.method === 'GET' && call.path.includes('/plan'))
      .length,
  }));
  assert.equal(first.planReads, 20, 'a full refresh replays every bound draft once');
  assert.ok(first.max <= 4, `plan replay stays within its small worker pool (saw ${first.max} in flight)`);
  assert.ok(first.max >= 2, `plan replay still overlaps reads through the pool (saw ${first.max} in flight)`);
  await page.evaluate(() => {
    window.workspaceFixture.calls.length = 0;
  });
  await work.getByRole('button', {name: 'Load more plans', exact: true}).click();
  await work.getByRole('heading', {name: 'Planned change 21', exact: true}).waitFor();
  // The replayed plan read follows the list response; wait for it to start
  // before asserting the paginated load fetched exactly the new page.
  await page.waitForFunction(
    () =>
      window.workspaceFixture.calls.filter((call) => call.method === 'GET' && call.path.includes('/plan')).length >= 1
  );
  assert.equal(await work.locator('.soda-work-item').count(), 21);
  const more = await page.evaluate(
    (last) => ({
      reads: window.workspaceFixture.calls
        .filter((call) => call.method === 'GET' && call.path.includes('/plan'))
        .map((call) => call.path),
      last,
    }),
    last
  );
  assert.equal(more.reads.length, 1, 'a paginated load fetches only the newly arrived page');
  assert.ok(more.reads[0]?.endsWith('/api/factory/intents/' + last + '/plan'));
  assert.equal(await writes(page), 0, 'paginated replay must not start planning or a check');
});

test('a formerly Checked draft stays neutral when its plan replay hangs past the request deadline', async (t) => {
  const page = await fixture(t);
  const objective = 'Checked draft with a hung replay';
  const {work, intent} = await seedCheckedDraft(page, objective);
  await page.evaluate((intent) => {
    const original = window.fetch.bind(window);
    const origSetTimeout = window.setTimeout.bind(window);
    Object.defineProperty(window, 'setTimeout', {
      configurable: true,
      value: (fn: TimerHandler, ms?: number, ...args: unknown[]) => {
        if (ms === 15000) return origSetTimeout(fn as () => void, 80);
        return origSetTimeout(fn as () => void, ms, ...(args as []));
      },
    });
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/api/factory/intents/' + intent + '/plan') && (init?.method || 'GET') === 'GET')
          return new Promise<Response>((_, reject) => {
            const signal = init?.signal;
            if (signal?.aborted) {
              reject(new DOMException('Aborted', 'AbortError'));
              return;
            }
            signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), {once: true});
          });
        return original(input, init);
      },
    });
  }, intent);
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  await work.getByText('Draft', {exact: true}).waitFor();
  // Pre-clear shows Draft immediately while the hung read is still in flight
  // (busy Loading feedback). Wait for the shortened deadline to retire the
  // hung read before asserting no list error surfaces.
  await page.waitForFunction(
    () => document.querySelectorAll('soda-project-work .soda-work-feedback').length === 0,
    undefined,
    {timeout: 2000}
  );
  assert.equal(await work.getByText('Checked', {exact: true}).count(), 0, 'a hung replay never retains Checked');
  assert.equal(await work.getByText('Needs correction', {exact: true}).count(), 0);
  assert.equal(await work.getByText('Plan ready', {exact: true}).count(), 0);
  await work.getByRole('button', {name: 'Continue planning: ' + objective, exact: true}).waitFor();
  assert.equal(await work.locator('.soda-work-feedback').count(), 0, 'a hung replay never surfaces a list error');
  assert.equal(await writes(page), 0, 'replaying a hung plan must not start planning or a check');
});

for (const status of [401, 403])
  test(`a ${status} plan replay still routes through authority-loss when another read hangs past the deadline`, async (t) => {
    const page = await fixture(t);
    const ids = await seedPlans(page, [
      {objective: 'Checked draft with a fast authority loss', repository: '7', state: 'draft'},
      {objective: 'Checked draft with a hung replay', repository: '7', state: 'draft'},
    ]);
    const authIntent = ids[0],
      hungIntent = ids[1];
    if (!authIntent || !hungIntent) throw Error('missing seeded intents');
    await page.evaluate(async (ids) => {
      const fixture = window.workspaceFixture;
      for (const intent of ids) {
        await fixture.request('/api/factory/intents/' + intent + '/plan', 'POST', {
          revision: intent,
          operation: 'propose_intent',
        });
      }
      fixture.calls.length = 0;
      await fixture.api.refresh();
    }, ids);
    const work = await openProjectPlans(page);
    await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
    await work.getByText('Checked', {exact: true}).first().waitFor();
    await page.evaluate(
      ({authIntent, hungIntent, status}) => {
        (window as unknown as {authorityLost: unknown[]}).authorityLost = [];
        window.addEventListener('soda-work-authority-lost', (event) => {
          (window as unknown as {authorityLost: unknown[]}).authorityLost.push((event as CustomEvent).detail);
        });
        const original = window.fetch.bind(window);
        const origSetTimeout = window.setTimeout.bind(window);
        Object.defineProperty(window, 'setTimeout', {
          configurable: true,
          value: (fn: TimerHandler, ms?: number, ...args: unknown[]) => {
            if (ms === 15000) return origSetTimeout(fn as () => void, 80);
            return origSetTimeout(fn as () => void, ms, ...(args as []));
          },
        });
        Object.defineProperty(window, 'fetch', {
          configurable: true,
          value: (input: RequestInfo | URL, init?: RequestInit) => {
            const url = String(input);
            if ((init?.method || 'GET') === 'GET') {
              if (url.includes('/api/factory/intents/' + authIntent + '/plan'))
                return Promise.resolve(new Response(null, {status}));
              if (url.includes('/api/factory/intents/' + hungIntent + '/plan'))
                return new Promise<Response>((_, reject) => {
                  const signal = init?.signal;
                  if (signal?.aborted) {
                    reject(new DOMException('Aborted', 'AbortError'));
                    return;
                  }
                  signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), {
                    once: true,
                  });
                });
            }
            return original(input, init);
          },
        });
      },
      {authIntent, hungIntent, status}
    );
    await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
    await page.waitForFunction(() => (window as unknown as {authorityLost: unknown[]}).authorityLost.length > 0);
    assert.deepEqual(
      await page.evaluate(() => (window as unknown as {authorityLost: unknown[]}).authorityLost),
      [{reconnectRequired: status === 401}],
      'a captured authority loss survives a hung sibling past the deadline'
    );
    assert.equal(await page.locator('.soda-work-item').count(), 0, 'authority loss retires the work list');
    assert.equal(await writes(page), 0, 'authority loss must not start planning or a check');
  });

for (const status of [401, 403])
  test(`a ${status} plan replay clears state immediately while its sibling stays blocked`, async (t) => {
    const page = await fixture(t);
    const ids = await seedPlans(page, [
      {objective: 'Checked draft with an immediate authority loss', repository: '7', state: 'draft'},
      {objective: 'Checked draft with a still-blocked replay', repository: '7', state: 'draft'},
    ]);
    const authIntent = ids[0],
      hungIntent = ids[1];
    if (!authIntent || !hungIntent) throw Error('missing seeded intents');
    await page.evaluate(async (ids) => {
      const fixture = window.workspaceFixture;
      for (const intent of ids) {
        await fixture.request('/api/factory/intents/' + intent + '/plan', 'POST', {
          revision: intent,
          operation: 'propose_intent',
        });
      }
      fixture.calls.length = 0;
      await fixture.api.refresh();
    }, ids);
    const work = await openProjectPlans(page);
    await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
    await work.getByText('Checked', {exact: true}).first().waitFor();
    // No deadline shortening here: cleanup must land well before the 15s
    // timeout while the sibling stays blocked. A hungReleased-only check
    // could pass after the deadline (abort never opens the gate), so race
    // against a short explicit timer and assert the deadline never fired.
    await page.evaluate(
      ({authIntent, hungIntent, status}) => {
        (window as unknown as {authorityLost: unknown[]}).authorityLost = [];
        window.addEventListener('soda-work-authority-lost', (event) => {
          (window as unknown as {authorityLost: unknown[]}).authorityLost.push((event as CustomEvent).detail);
        });
        (window as unknown as {deadlineFired: boolean}).deadlineFired = false;
        (window as unknown as {gateOpened: boolean}).gateOpened = false;
        (window as unknown as {hungGate: Promise<void>; openHungGate: () => void}).hungGate = new Promise<void>(
          (resolve) => {
            (window as unknown as {openHungGate: () => void}).openHungGate = () => {
              (window as unknown as {gateOpened: boolean}).gateOpened = true;
              resolve();
            };
          }
        );
        const origSetTimeout = window.setTimeout.bind(window);
        Object.defineProperty(window, 'setTimeout', {
          configurable: true,
          value: (fn: TimerHandler, ms?: number, ...args: unknown[]) => {
            if (ms === 15000)
              return origSetTimeout(
                () => {
                  (window as unknown as {deadlineFired: boolean}).deadlineFired = true;
                  (fn as () => void)();
                },
                ms,
                ...(args as [])
              );
            return origSetTimeout(fn as () => void, ms, ...(args as []));
          },
        });
        const original = window.fetch.bind(window);
        Object.defineProperty(window, 'fetch', {
          configurable: true,
          value: (input: RequestInfo | URL, init?: RequestInit) => {
            const url = String(input);
            if ((init?.method || 'GET') === 'GET') {
              if (url.includes('/api/factory/intents/' + authIntent + '/plan'))
                return Promise.resolve(new Response(null, {status}));
              if (url.includes('/api/factory/intents/' + hungIntent + '/plan'))
                return new Promise<Response>((_, reject) => {
                  const signal = init?.signal;
                  if (signal?.aborted) {
                    reject(new DOMException('Aborted', 'AbortError'));
                    return;
                  }
                  signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), {
                    once: true,
                  });
                  (window as unknown as {hungGate: Promise<void>}).hungGate.then(() =>
                    reject(new DOMException('Aborted', 'AbortError'))
                  );
                });
            }
            return original(input, init);
          },
        });
      },
      {authIntent, hungIntent, status}
    );
    await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
    // Short explicit timer: cleanup must land well before the 15s deadline.
    await page.waitForFunction(
      () => (window as unknown as {authorityLost: unknown[]}).authorityLost.length > 0,
      undefined,
      {
        timeout: 2000,
      }
    );
    assert.deepEqual(
      await page.evaluate(() => (window as unknown as {authorityLost: unknown[]}).authorityLost),
      [{reconnectRequired: status === 401}],
      'authority loss fires without waiting for the hung sibling'
    );
    assert.equal(
      await page.evaluate(() => (window as unknown as {deadlineFired: boolean}).deadlineFired),
      false,
      'immediate cleanup lands with no 15s deadline fired'
    );
    assert.equal(
      await page.evaluate(() => (window as unknown as {gateOpened: boolean}).gateOpened),
      false,
      'cleanup lands while the blocked sibling gate stays closed'
    );
    assert.equal(await page.locator('.soda-work-item').count(), 0, 'immediate authority loss retires the work list');
    assert.equal(
      await work.getByText('Checked', {exact: true}).count(),
      0,
      'stale Checked never survives authority loss'
    );
    assert.equal(await writes(page), 0, 'immediate authority loss must not start planning or a check');
    await page.evaluate(() => (window as unknown as {openHungGate: () => void}).openHungGate());
    await page.waitForTimeout(300);
    assert.equal(
      await page.evaluate(() => (window as unknown as {deadlineFired: boolean}).deadlineFired),
      false,
      'releasing the hung sibling never needs the deadline'
    );
    assert.equal(await page.locator('.soda-work-item').count(), 0, 'released sibling never restores retired work');
    assert.equal(await writes(page), 0, 'releasing the hung sibling must not start planning or a check');
  });

test('a slow refresh pre-clears stale Checked work before the deadline while its sibling stays blocked', async (t) => {
  const page = await fixture(t);
  const objective = 'Checked draft with a slow replay';
  const {work, intent} = await seedCheckedDraft(page, objective);
  await page.evaluate((intent) => {
    (window as unknown as {deadlineFired: boolean}).deadlineFired = false;
    (window as unknown as {planStarted: boolean}).planStarted = false;
    (window as unknown as {planSettled: boolean}).planSettled = false;
    (window as unknown as {gateOpened: boolean}).gateOpened = false;
    (window as unknown as {hungGate: Promise<void>; openHungGate: () => void}).hungGate = new Promise<void>(
      (resolve) => {
        (window as unknown as {openHungGate: () => void}).openHungGate = () => {
          (window as unknown as {gateOpened: boolean}).gateOpened = true;
          resolve();
        };
      }
    );
    const origSetTimeout = window.setTimeout.bind(window);
    Object.defineProperty(window, 'setTimeout', {
      configurable: true,
      value: (fn: TimerHandler, ms?: number, ...args: unknown[]) => {
        if (ms === 15000)
          return origSetTimeout(
            () => {
              (window as unknown as {deadlineFired: boolean}).deadlineFired = true;
              (fn as () => void)();
            },
            ms,
            ...(args as [])
          );
        return origSetTimeout(fn as () => void, ms, ...(args as []));
      },
    });
    const original = window.fetch.bind(window);
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.includes('/api/factory/intents/' + intent + '/plan') && (init?.method || 'GET') === 'GET') {
          (window as unknown as {planStarted: boolean}).planStarted = true;
          return new Promise<Response>((_, reject) => {
            const signal = init?.signal;
            if (signal?.aborted) {
              (window as unknown as {planSettled: boolean}).planSettled = true;
              reject(new DOMException('Aborted', 'AbortError'));
              return;
            }
            signal?.addEventListener(
              'abort',
              () => {
                (window as unknown as {planSettled: boolean}).planSettled = true;
                reject(new DOMException('Aborted', 'AbortError'));
              },
              {once: true}
            );
            (window as unknown as {hungGate: Promise<void>}).hungGate.then(() => {
              (window as unknown as {planSettled: boolean}).planSettled = true;
              reject(new DOMException('Aborted', 'AbortError'));
            });
          });
        }
        return original(input, init);
      },
    });
  }, intent);
  await page.evaluate(() => {
    window.workspaceFixture.calls.length = 0;
  });
  await work.getByRole('button', {name: 'Refresh plans', exact: true}).click();
  // Race pre-clear against a short explicit timer: stale Checked must retire
  // synchronously at refresh start, while the sibling read stays blocked and
  // the 15s deadline never fires.
  await work.getByText('Draft', {exact: true}).waitFor({timeout: 2000});
  await page.waitForFunction(() => (window as unknown as {planStarted: boolean}).planStarted === true, undefined, {
    timeout: 2000,
  });
  assert.equal(await work.getByText('Checked', {exact: true}).count(), 0, 'slow refresh pre-clears stale Checked');
  assert.equal(
    await page.evaluate(() => (window as unknown as {deadlineFired: boolean}).deadlineFired),
    false,
    'pre-clear lands with no 15s deadline fired'
  );
  assert.equal(
    await page.evaluate(() => (window as unknown as {planStarted: boolean}).planStarted),
    true,
    'sibling read started and stays blocked'
  );
  assert.equal(
    await page.evaluate(() => (window as unknown as {planSettled: boolean}).planSettled),
    false,
    'sibling read remains unresolved at pre-clear'
  );
  assert.equal(
    await page.evaluate(() => (window as unknown as {gateOpened: boolean}).gateOpened),
    false,
    'sibling gate stays closed at pre-clear'
  );
  assert.equal(await writes(page), 0, 'pre-clearing a slow replay must not start planning or a check');
  await page.evaluate(() => (window as unknown as {openHungGate: () => void}).openHungGate());
  await work.getByText('Draft', {exact: true}).waitFor();
  assert.equal(
    await work.getByText('Checked', {exact: true}).count(),
    0,
    'released sibling never restores stale Checked'
  );
  assert.equal(
    await page.evaluate(() => (window as unknown as {deadlineFired: boolean}).deadlineFired),
    false,
    'releasing the hung sibling never needs the deadline'
  );
  assert.equal(await writes(page), 0, 'releasing the hung sibling must not start planning or a check');
});

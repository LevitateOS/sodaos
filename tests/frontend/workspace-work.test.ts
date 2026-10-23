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
  await work.getByText('Adopt or reaccept an issue', {exact: true}).click().catch(async () => { throw Error(await page.locator('.soda-workspace').innerText()); });
  await work.getByLabel('Native issue number', {exact: true}).fill('13');
  await work.getByRole('button', {name: 'Open requirements review', exact: true}).click();
  const acceptance = work.getByRole('region', {name: 'Issue requirements acceptance'});
  assert.equal(
    await acceptance
      .getByRole('link', {name: 'Review issue #13 in the native forge', exact: true})
      .getAttribute('href'),
    '/alice/Alpha/issues/13'
  );
  const requests = await page.evaluate(() => window.workspaceFixture.calls.map((call) => call.path));
  assert(requests.some((path) => path.endsWith('/api/repositories/7/factory/access')));
  assert.equal(await writes(page), 0, 'opening review must not accept an issue or create a plan');
  await work.getByLabel('Native issue number', {exact: true}).fill('not-an-issue');
  await work.getByRole('button', {name: 'Open requirements review', exact: true}).click();
  await work.getByRole('status').filter({hasText: 'Enter the native issue number'}).waitFor();
  assert.equal(await writes(page), 0);
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
  await work.getByRole('button', {name: 'New work', exact: true}).click();
  await page.getByLabel('Describe what you want to build', {exact: true}).waitFor();
  assert.equal(new URL(page.url()).pathname, '/workspace/new');
  assert.equal(await writes(page), 0, 'entering planning must not create a Project or work');
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
  await (await openProjectPlans(page))
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
  await (await openProjectPlans(page))
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

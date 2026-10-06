import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, chooseFirstRepository, fixture} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

test('setup: pending creation keeps its selection and disables navigation without duplicate writes', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/api/environments') && init?.method === 'POST')
          await new Promise<void>((resolve) =>
            window.addEventListener('release-creation', () => resolve(), {once: true})
          );
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Creating project…', exact: true}).waitFor();
  for (const name of ['Back', 'Change repository', 'Cancel setup', 'Creating project…'])
    assert(await page.getByRole('button', {name, exact: true}).isDisabled(), name);
  assert.equal(await page.locator('.soda-config-repository-name').innerText(), 'alice/Alpha\nForgejo');
  await captureSpacesComponent(page, 'pending-creation');
  await page.evaluate(() => window.dispatchEvent(new Event('release-creation')));
  await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 1);
});
for (const outcome of ['uncertain', 'incomplete', 'rejected'] as const)
  test(`first use: ${outcome} creation is inspected, not replayed`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate((outcome) => {
      window.workspaceFixture.setCreateOutcome(outcome);
      return window.workspaceFixture.api.refresh();
    }, outcome);
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page
      .getByText(
        outcome === 'rejected'
          ? /Installed Project OS unavailable\. No reservation/
          : /Project creation could not be confirmed/
      )
      .waitFor();
    await page.getByRole('button', {name: 'Refresh status', exact: true}).click();
    if (outcome === 'uncertain') await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    else if (outcome === 'incomplete') await page.getByRole('heading', {name: 'Project needs inspection'}).waitFor();
    else await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 1);
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  });
test('first use: keyboard selection, Back and Change retain the repository without writes', async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('button', {name: 'Create project', exact: true}).focus();
  await page.keyboard.press('Enter');
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  assert(await page.getByRole('button', {name: 'Continue', exact: true}).isDisabled());
  assert.equal(
    await page.getByRole('link', {name: 'Create a new repository'}).getAttribute('href'),
    '/forgejo.test/repo/create'
  );
  await page.getByRole('radio', {name: /alice\/Alpha/}).focus();
  await page.keyboard.press('Space');
  await page.getByRole('button', {name: 'Continue', exact: true}).focus();
  await page.keyboard.press('Enter');
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  assert(await page.getByRole('radio', {name: /alice\/Alpha/}).isChecked());
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.waitForFunction(() => !!document.querySelector('.soda-project-journey button.primary:not([disabled])'));
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.getByRole('heading', {name: 'Create your first project'}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
});
test('first use: superseded search and retired actor cannot publish stale private choices', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input), location.origin);
        if (url.pathname.endsWith('/api/repositories') && url.searchParams.get('q') === 'Alpha')
          return new Promise<Response>((resolve) =>
            window.addEventListener(
              'release-old-search',
              () =>
                resolve(
                  Response.json({
                    items: [{id: '7', owner: 'alice', name: 'Alpha', can_create: true, project: null}],
                  })
                ),
              {once: true}
            )
          );
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  const search = page.getByRole('searchbox', {name: 'Search repositories'});
  await search.fill('Alpha');
  await search.press('Enter');
  await search.fill('Beta');
  await search.press('Enter');
  await page.getByRole('radio', {name: /alice\/Beta/}).check();
  await page.evaluate(() => window.dispatchEvent(new Event('release-old-search')));
  assert.equal(await page.getByRole('radio', {name: /alice\/Alpha/}).count(), 0);
  assert(await page.getByRole('radio', {name: /alice\/Beta/}).isChecked());
  await search.fill('Alpha');
  await search.press('Enter');
  await page.evaluate(async () => {
    window.workspaceFixture.setUser('2');
    await window.workspaceFixture.api.refresh();
    window.dispatchEvent(new Event('release-old-search'));
  });
  await page.getByRole('button', {name: 'Reload Spaces', exact: true}).waitFor();
  assert.equal(await page.getByRole('radio').count(), 0);
  assert.equal(await page.getByRole('heading', {name: 'Create your first project'}).count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
});
for (const administrator of [true, false])
  test(`first use: stopped project Start is explicit and authorized (${administrator})`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await page.evaluate(async (administrator) => {
      const f = window.workspaceFixture,
        space = f.spaces[0];
      if (!space?.observed) throw Error('fixture');
      space.observed.running = false;
      space.environment_administrator = administrator;
      await f.api.refresh();
    }, administrator);
    await page.getByRole('heading', {name: 'Project not ready'}).waitFor();
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 1);
    if (administrator) {
      await page.getByRole('button', {name: 'Start project', exact: true}).click();
      await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
      assert.equal(
        await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length),
        2
      );
    } else assert.equal(await page.getByRole('button', {name: 'Start project', exact: true}).count(), 0);
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  });
test('first use: second-project setup cancellation preserves the live renderer, input target and layout', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.is-connected').waitFor();
  const screen = await page.locator('.xterm').elementHandle();
  const before = await page.evaluate(() => ({
    layout: sessionStorage.getItem('soda-spaces:v3:1'),
    writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length,
  }));
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Beta/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  assert(await screen?.evaluate((node) => node.isConnected));
  assert.deepEqual(
    await page.evaluate(() => ({
      layout: sessionStorage.getItem('soda-spaces:v3:1'),
      writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length,
    })),
    before
  );
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((s) => s.closed)), [0]);
});

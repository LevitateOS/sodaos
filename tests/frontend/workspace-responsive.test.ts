import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, chooseFirstRepository, fixture, openSession} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

for (const viewport of [
  {width: 320, height: 568},
  {width: 720, height: 422},
])
  test(`welcome ${viewport.width}/${viewport.height}: keyboard action and short-screen content stay reachable`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.setViewportSize(viewport);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
    const surface = page.locator('.soda-workspace');
    assert(await surface.evaluate((node) => node.scrollWidth <= node.clientWidth), 'intent overflows horizontally');
    await page.getByRole('heading', {name: 'What do you want to build?'}).focus();
    await page.keyboard.press('Tab');
    const description = page.getByLabel('Describe what you want to build', {exact: true});
    assert(await description.evaluate((node) => node.matches(':focus')), 'description is the first keyboard action');
    await description.fill('Build a leave tracker');
    const action = page.getByRole('button', {name: 'Explore ideas', exact: true});
    await action.scrollIntoViewIfNeeded();
    const box = await action.boundingBox();
    assert(box && box.y >= 0 && box.y + box.height <= viewport.height, 'planning action cannot be reached');
    await action.focus();
    await page.keyboard.press('Enter');
    await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
    assert.equal(
      await page.evaluate(
        () =>
          window.workspaceFixture.calls.filter((call) => call.method !== 'GET' && !call.path.includes('/api/factory/'))
            .length
      ),
      0
    );
  });
for (const viewport of [
  {width: 320, height: 568},
  {width: 720, height: 422},
])
  test(`workspace intro ${viewport.width}/${viewport.height}: long identity and actions remain reachable`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.setViewportSize(viewport);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await page.evaluate(async () => {
      const f = window.workspaceFixture,
        space = f.spaces[0];
      if (!space) throw Error('fixture');
      space.environment.repository = 'alice/' + 'a-very-long-project-name-'.repeat(8);
      await f.api.refresh();
    });
    for (const action of ['Join project', 'New terminal']) {
      await page.getByRole('button', {name: action, exact: true}).scrollIntoViewIfNeeded();
      const button = await page.getByRole('button', {name: action, exact: true}).boundingBox();
      assert(
        button && button.y >= 0 && button.y + button.height <= viewport.height,
        'primary action cannot be reached'
      );
      assert(
        await page.locator('.soda-workspace').evaluate((node) => node.scrollWidth <= node.clientWidth),
        'long identity overflows'
      );
      if (action === 'Join project') await page.getByRole('button', {name: action, exact: true}).click();
    }
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  });
for (const theme of ['light', 'dark'])
  test(`coherence short ${theme}: long names, bounded menus and original disconnected identity`, async (t) => {
    const page = await fixture(t);
    await page.setViewportSize({width: 390, height: 422});
    await page.evaluate(async (theme) => {
      document.documentElement.style.colorScheme = theme;
      const f = window.workspaceFixture,
        space = f.spaces[0];
      if (!space) throw Error('fixture');
      space.environment.repository = 'alice/' + 'long-project-name-'.repeat(10);
      if (space.terminals[0]) space.terminals[0].name = 'Build ' + 'long-terminal-name-'.repeat(3);
      await f.api.refresh();
    }, theme);
    await openSession(page, 'Build');
    const screen = await page.locator('.soda-workspace-terminal:visible .xterm').elementHandle();
    const owner = await page.locator('.soda-workspace-terminal:visible').elementHandle();
    const locator = await page.locator('.soda-workspace-terminal:visible').getAttribute('id');
    const baseline = await page.evaluate(() => ({
      writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length,
      sockets: window.workspaceFixture.sockets.length,
    }));
    await page.locator('.soda-workspace-terminal:visible').getByLabel('Terminal actions', {exact: true}).click();
    await captureSpacesComponent(page, `short-menu-${theme}-390`);
    const menu = page.locator('.soda-workspace-terminal:visible .soda-menu[open] > div');
    const bounds = await menu.boundingBox();
    assert(bounds && bounds.y + bounds.height <= 422 && bounds.x >= 0, 'short-screen menu is clipped');
    await menu.getByRole('button', {name: 'End terminal…'}).scrollIntoViewIfNeeded();
    await page.getByLabel('Workspace options', {exact: true}).click();
    assert.equal(await page.locator('.soda-menu[open]').count(), 1, 'multiple menus remain open');
    await page.getByRole('button', {name: 'Refresh Spaces', exact: true}).focus();
    await page.getByLabel('Workspace options', {exact: true}).focus();
    await page.keyboard.press('Shift+Tab');
    assert.equal(await page.locator('.soda-menu[open]').count(), 0, 'Tab departure left a menu open');
    assert.deepEqual(
      await page.evaluate(() => ({
        writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length,
        sockets: window.workspaceFixture.sockets.length,
      })),
      baseline
    );
    assert(await screen?.evaluate((node) => node.isConnected));
    await page.evaluate(() => {
      const f = window.workspaceFixture;
      f.setStatus(503);
      f.sockets[0]?.close();
    });
    await page.getByRole('button', {name: 'Reconnect terminal', exact: true}).waitFor();
    await captureSpacesComponent(page, `disconnected-${theme}-390`);
    assert(await owner?.evaluate((node) => node.isConnected));
    assert.equal(await page.locator('.soda-workspace-terminal:visible').getAttribute('id'), locator);
    assert.equal(
      await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length),
      baseline.writes
    );
  });
test('current native panels allocation preserves available Spaces width and quiet controls', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    // This case opts native chrome into the branding button contract while
    // leaving the retained mount in its separate native panels allocation.
    document.querySelector('.page-content')?.classList.add('soda-page');
    const button = document.createElement('button');
    button.className = 'ui button';
    button.id = 'native-button-probe';
    button.textContent = 'Native action';
    document.body.append(button);
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await chooseFirstRepository(page);
  const layout = await page.locator('main[data-extension-panel]').boundingBox();
  const workspace = await page.locator('.soda-workspace').boundingBox();
  assert(layout && workspace, 'authored native allocation and real workspace must be visible');
  assert(
    Math.abs(workspace.x - layout.x) <= 1 && Math.abs(workspace.width - layout.width) <= 1,
    'workspace must use the width supplied by native panels'
  );
  const back = page.getByRole('button', {name: 'Back', exact: true});
  assert.equal(
    await back.evaluate((node) => getComputedStyle(node).borderTopColor),
    'rgba(0, 0, 0, 0)',
    'native button rules boxed the quiet Back action'
  );
  assert.equal(
    await page.locator('#native-button-probe').evaluate((node) => getComputedStyle(node).borderTopWidth),
    '1px'
  );
  assert(
    (await page.locator('#native-button-probe').boundingBox())!.height >= 44,
    'native controls outside Spaces lost their target size'
  );
});

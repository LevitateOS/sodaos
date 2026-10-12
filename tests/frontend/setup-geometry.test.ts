import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, chooseFirstRepository, fixture} from './fixtures/workspace-driver';
setupWorkspaceDriver();

// F03: setup/configure keeps its repository context and step actions reachable
// at compact and short-height viewports. Clipping below the fold is only a
// defect when scrolling cannot honestly reveal the action.
for (const viewport of [
  {width: 320, height: 568},
  {width: 720, height: 422},
])
  test(
    `setup configure ${viewport.width}x${viewport.height}: context and step actions stay reachable`,
    {timeout: 30000},
    async (t) => {
      const page = await fixture(t, 'page', undefined, true);
      await page.setViewportSize(viewport);
      await page.evaluate(() => window.workspaceFixture.api.refresh());
      await chooseFirstRepository(page);
      await page.getByRole('heading', {name: 'Configure project'}).waitFor();
      assert.equal(await page.locator('.soda-config-repository-name').innerText(), 'alice/Alpha\nForgejo');
      assert(
        await page.locator('.soda-workspace-frame').evaluate((node) => node.scrollWidth <= node.clientWidth + 1),
        'setup frame overflows horizontally'
      );
      for (const name of ['Back', 'Change repository', 'Cancel setup', 'Create project']) {
        const action = page.getByRole('button', {name, exact: true});
        await action.scrollIntoViewIfNeeded();
        const box = await action.boundingBox();
        assert(box && box.y >= 0 && box.y + box.height <= viewport.height, `${name} cannot be reached`);
        assert(await action.isEnabled(), `${name} is not enabled`);
      }
      await page.getByRole('button', {name: 'Back', exact: true}).click();
      await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
      assert.equal(await page.getByRole('heading', {name: 'Choose a repository'}).count(), 0);
      await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
      await page.getByRole('heading', {name: 'Choose a repository'}).waitFor();
      await page.getByRole('radio', {name: /alice\/Alpha/}).check();
      await page.getByRole('button', {name: 'Continue', exact: true}).click();
      await page.getByRole('heading', {name: 'Configure project'}).waitFor();
      const create = page.getByRole('button', {name: 'Create project', exact: true});
      await create.scrollIntoViewIfNeeded();
      const createBox = await create.boundingBox();
      assert(
        createBox && createBox.y >= 0 && createBox.y + createBox.height <= viewport.height,
        'Create project cannot be reached after Back round-trip'
      );
      await create.click();
      await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    }
  );

import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('compact setup: Back from creation restores the same connected terminal', async (t) => {
  const page = await fixture(t, 'page');
  await page.setViewportSize({width: 360, height: 700});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const before = await page.locator('.soda-workspace-terminal:not([hidden])').getAttribute('id');
  // Creation is only reachable through the sessions overlay at compact widths.
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.getByRole('button', {name: 'New project', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal:not([hidden])').getAttribute('id'), before);
});

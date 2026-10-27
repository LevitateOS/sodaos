import test from 'node:test';
import assert from 'node:assert/strict';
import {
  reviewFirstIntent,
  completeBoundIntentPlanning,
  fixture,
  setupWorkspaceDriver,
} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('first use: inventory-click accepted resume restores its overview with native links and no writes', async (t) => {
  const page = await fixture(t, 'page', undefined, true, '/forgejo.test');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Prepare project', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await completeBoundIntentPlanning(page);
  const intent = await page.evaluate(() => {
    const calls = window.workspaceFixture.calls as {body?: Record<string, unknown>}[];
    const accepted = calls.find((call) => (call.body as Record<string, unknown> | undefined)?.state === 'accepted');
    return String((accepted?.body as Record<string, unknown> | undefined)?.intent || '');
  });
  assert.match(intent, /^d[0-9a-f]{24}$/);
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  await page.waitForFunction(() => {
    const el = document.querySelector('soda-intent');
    const h2 = Array.from(el?.shadowRoot?.querySelectorAll('h2') || []).find(
      (h) => h.textContent?.trim() === 'Review your product direction'
    );
    if (!h2) return false;
    const r = h2.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  });
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert.equal(await page.locator('soda-factory-work-overview').count(), 0);
  const resume = page.getByRole('button', {name: /^Resume Build a small project tracker$/});
  await resume.waitFor();
  const settledCalls = await page.evaluate(() => window.workspaceFixture.calls.length);
  await resume.click();
  const overview = page.locator('soda-factory-work-overview');
  await overview.getByText('Build a small project tracker').waitFor();
  await overview.getByText('Native work generated.').waitFor();
  const link = overview.getByRole('link', {name: 'issue #5', exact: true});
  assert.equal(await link.getAttribute('href'), '/forgejo.test/alice/Alpha/issues/5');
  const writes = await page.evaluate(
    (settled) => window.workspaceFixture.calls.slice(settled).filter((call) => (call.method || 'GET') !== 'GET').length,
    settledCalls
  );
  assert.equal(writes, 0);
});

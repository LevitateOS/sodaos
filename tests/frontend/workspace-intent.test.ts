import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('intent-first entry stores department and corrected product review before repository association', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert.equal(await page.getByRole('radio').count(), 0);
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.some((call) => call.path.includes('/api/repositories'))),
    false
  );
  await page.getByRole('button', {name: 'New department', exact: true}).click();
  const dialog = page.getByRole('dialog', {name: 'New department'});
  await dialog.getByLabel('Name', {exact: true}).fill('People operations');
  await dialog.getByLabel('What does it do?', {exact: true}).fill('Supports staff onboarding');
  await dialog.getByRole('button', {name: 'Create department', exact: true}).click();
  await dialog.waitFor({state: 'detached'});
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  assert(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled());
  await page.getByRole('combobox', {name: /^Choice:/}).selectOption('edit');
  await page
    .getByLabel('What should change?', {exact: true})
    .fill('Track leave requests without external integrations');
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page.getByRole('button', {name: 'Choose repository', exact: true}).waitFor();
  await page.getByText('Proposed work (1)', {exact: true}).click();
  await page.getByRole('heading', {name: 'Track leave requests without external integrations'}).waitFor();
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project', exact: true}).waitFor();
  const calls = await page.evaluate(() => window.workspaceFixture.calls);
  const accepted = calls.find((call) => call.body?.state === 'accepted');
  assert(accepted?.body);
  assert.equal(accepted.body.repository, '7');
  assert.equal(accepted.body.max_issues, 20);
  assert.equal(accepted.body.planning_calls, 3);
  assert.equal(typeof accepted.body.department, 'string');
  assert.equal(calls.filter((call) => call.method !== 'GET' && !call.path.includes('/api/factory/')).length, 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
});

test('an unavailable proposal keeps the objective editable and permits a bounded retry', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    let fail = true;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/explore') && fail) {
          fail = false;
          return Promise.resolve(new Response(null, {status: 503}));
        }
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText('Could not save or plan this request. Your choices are kept; try again.', {exact: true})
    .waitFor();
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.body?.state === 'accepted').length),
    0
  );
});

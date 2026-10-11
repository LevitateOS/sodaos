import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, reviewFirstIntent} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

test('intent-first entry stores department and corrected product review before repository association', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await captureSpacesComponent(page, 'intent-entry');
  assert.equal(await page.getByRole('radio').count(), 0);
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.some((call) => call.path.includes('/api/repositories'))),
    false
  );
  await page.getByRole('button', {name: 'New department', exact: true}).click();
  const dialog = page.getByRole('dialog', {name: 'New department'});
  await dialog.getByLabel('Name', {exact: true}).fill('People operations');
  await dialog.getByLabel('What does it do?', {exact: true}).fill('Supports staff onboarding');
  await captureSpacesComponent(page, 'intent-department');
  await dialog.getByRole('button', {name: 'Create department', exact: true}).click();
  await dialog.waitFor({state: 'detached'});
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  assert(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled());
  await page.getByLabel('Choose how to deliver the accepted workflow', {exact: true}).selectOption('complete_first');
  await page
    .getByText('Implement and verify the complete accepted workflow before first delivery', {exact: true})
    .waitFor();
  await page.getByRole('combobox', {name: /^Choice:/}).selectOption('edit');
  await page
    .getByLabel('What should change?', {exact: true})
    .fill('Track leave requests without external integrations');
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page.getByRole('button', {name: 'Choose repository', exact: true}).waitFor();
  await page.getByText('Proposed work (1)', {exact: true}).click();
  await page.getByRole('heading', {name: 'Track leave requests without external integrations'}).waitFor();
  await captureSpacesComponent(page, 'intent-corrected-review');
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project', exact: true}).waitFor();
  const calls = await page.evaluate(() => window.workspaceFixture.calls);
  const accepted = calls.find((call) => call.body?.state === 'accepted');
  assert(accepted?.body);
  assert.equal(accepted.body.repository, '7');
  assert.equal(accepted.body.max_issues, 20);
  assert.equal(accepted.body.planning_calls, 8);
  assert.equal(accepted.body.approach, 'complete_first');
  assert.equal(typeof accepted.body.department, 'string');
  assert.equal(calls.filter((call) => call.method !== 'GET' && !call.path.includes('/api/factory/')).length, 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(calls.filter((call) => call.path.endsWith('/generate')).length, 1);
});

test('native generation failure retains acceptance and explains the remaining enrollment', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/generate'))
          return Promise.resolve(Response.json({error: {code: 'native_graph_unavailable'}}, {status: 503}));
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project', exact: true}).waitFor();
  await page
    .getByText(
      'Your product direction is saved. An administrator must enable native work creation before Factory can proceed.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.body?.state === 'accepted').length),
    1
  );
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter((call) => call.method !== 'GET' && !call.path.includes('/api/factory/'))
          .length
    ),
    0
  );
});

test('save draft reports saved state and save-and-leave signals return without losing text', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const seen: {name: string; detail: unknown}[] = [];
    for (const name of ['intent-saved', 'intent-leave'])
      document.addEventListener(name, (event) => seen.push({name, detail: (event as CustomEvent).detail}));
    (window as unknown as {intentEvents: {name: string; detail: unknown}[]}).intentEvents = seen;
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert(await page.getByRole('button', {name: 'Save draft', exact: true}).isDisabled());
  assert(await page.getByRole('button', {name: 'Save and leave', exact: true}).isDisabled());
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByText('Unsaved changes', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const saved = await page.evaluate(
    () => (window as unknown as {intentEvents: {name: string; detail: unknown}[]}).intentEvents
  );
  assert.equal(saved.length, 1);
  assert.equal(saved[0]?.name, 'intent-saved');
  const detail = saved[0]?.detail as {intent: string; revision: string; state: string};
  assert(/^d[0-9a-f]{24}$/.test(detail.intent));
  assert(/^d[0-9a-f]{24}$/.test(detail.revision));
  assert.equal(detail.state, 'draft');
  const calls = await page.evaluate(() => window.workspaceFixture.calls);
  assert(calls.some((call) => call.path.endsWith('/api/factory/intents') && call.body?.state === 'draft'));
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker for staff');
  await page.getByText('Unsaved changes', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Save and leave', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const left = await page.evaluate(
    () => (window as unknown as {intentEvents: {name: string; detail: unknown}[]}).intentEvents
  );
  assert.deepEqual(
    left.map((entry) => entry.name),
    ['intent-saved', 'intent-saved', 'intent-leave']
  );
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/api/factory/intents') && init?.method === 'POST')
          return Promise.resolve(new Response(null, {status: 500}));
        return original(input, init);
      },
    });
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker for staff teams');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page
    .getByText('Could not save or plan this request. Your choices are kept; try again.', {exact: true})
    .waitFor();
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker for staff teams'
  );
  await page.getByText('Unsaved changes', {exact: true}).waitFor();
});

test('department dialog validates, retains the objective and restores focus on cancel or save', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'New department', exact: true}).click();
  const dialog = page.getByRole('dialog', {name: 'New department'});
  await dialog.getByText('Name and description are required.', {exact: false}).waitFor();
  assert(await dialog.getByRole('button', {name: 'Create department', exact: true}).isDisabled());
  await dialog.getByLabel('Name', {exact: true}).fill('People operations');
  assert(await dialog.getByRole('button', {name: 'Create department', exact: true}).isDisabled());
  await dialog.getByRole('button', {name: 'Cancel', exact: true}).click();
  await dialog.waitFor({state: 'detached'});
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
  await page.waitForFunction(() => {
    const active = document.querySelector('soda-intent')?.shadowRoot?.activeElement;
    return active instanceof HTMLButtonElement && active.classList.contains('open-department');
  });
  await page.getByRole('button', {name: 'New department', exact: true}).click();
  await dialog.getByLabel('Name', {exact: true}).fill('People operations');
  await dialog.getByLabel('What does it do?', {exact: true}).fill('Supports staff onboarding');
  await dialog.getByRole('button', {name: 'Create department', exact: true}).click();
  await dialog.waitFor({state: 'detached'});
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
  const departmentValue = await page.evaluate(() => {
    const select = document.querySelector('soda-intent')?.shadowRoot?.querySelector('select');
    return select instanceof HTMLSelectElement ? select.value : 'missing';
  });
  assert.notEqual(departmentValue, '');
  assert.notEqual(departmentValue, 'missing');
  await page.waitForFunction(() => {
    const active = document.querySelector('soda-intent')?.shadowRoot?.activeElement;
    return active instanceof HTMLButtonElement && active.classList.contains('open-department');
  });
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

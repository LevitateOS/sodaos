import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {setupWorkspaceDriver, fixture, reviewFirstIntent} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

interface IntentStore {
  arm(id: string): void;
  release(id: string): void;
  seed(entries: Record<string, unknown>[]): void;
  forbid: boolean;
  sessionLost: boolean;
}
// Test-local private intent peer: records saved revisions, serves the declared
// current-revision read and owner-private list, and gates/refuses loads on
// demand. Shared fixtures stay untouched.
async function installIntentStore(page: Page) {
  await page.evaluate(() => {
    const records = new Map<string, Record<string, unknown>>();
    const gates = new Map<string, () => void>();
    const single = (url: string) => url.match(/\/api\/factory\/intents\/([^/?#]+)$/)?.[1] || '';
    const store = {
      forbid: false,
      sessionLost: false,
      arm(id: string) {
        gates.set(id, () => {});
      },
      release(id: string) {
        const resolve = gates.get(id);
        gates.delete(id);
        resolve?.();
      },
      seed(entries: Record<string, unknown>[]) {
        for (const entry of entries) records.set(String(entry.intent), entry);
      },
    };
    (window as unknown as {intentStore: typeof store}).intentStore = store;
    (window as unknown as {resumed: unknown[]}).resumed = [];
    document.addEventListener('intent-resumed', (event) =>
      (window as unknown as {resumed: unknown[]}).resumed.push((event as CustomEvent).detail)
    );
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const method = init?.method || 'GET';
        const id = single(url);
        if (id && method === 'GET') {
          if (store.sessionLost) return new Response(null, {status: 403});
          if (store.forbid) return new Response(null, {status: 404});
          if (gates.has(id)) await new Promise<void>((resolve) => gates.set(id, resolve));
          const record = records.get(id);
          if (!record) return new Response(null, {status: 404});
          return Response.json({...record, owner: '1', created_unix: 1});
        }
        if (url.includes('/api/factory/intents?') && method === 'GET') {
          if (store.sessionLost) return new Response(null, {status: 403});
          if (store.forbid) return new Response(null, {status: 404});
          const intents = [...records.values()].map((record) => ({
            intent: record.intent,
            revision: record.id,
            state: record.state,
            objective: String(record.objective).slice(0, 280),
            repository: record.repository,
            created_unix: 1,
            effective_count: 0,
            task_count: Array.isArray(record.tasks) ? record.tasks.length : 0,
          }));
          return Response.json({intents});
        }
        if (url.endsWith('/api/factory/intents') && method === 'POST') {
          const response = await original(input, init);
          if (response.ok && typeof init?.body === 'string') {
            const body = JSON.parse(init.body) as Record<string, unknown>;
            records.set(String(body.intent), {...body});
          }
          return response;
        }
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
}
async function resetIntent(page: Page) {
  await page.evaluate(() => {
    (document.querySelector('soda-intent') as unknown as {reset(): void}).reset();
  });
}
async function resumeIntent(page: Page, id: string) {
  await page.evaluate((intent: string) => {
    (document.querySelector('soda-intent') as unknown as {request: string}).request = intent;
  }, id);
}
async function intentObjective(page: Page): Promise<string> {
  return page.evaluate(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement ? area.value : 'missing';
  });
}
async function lastPostedBody(page: Page, state: string): Promise<Record<string, unknown>> {
  return page.evaluate((wanted: string) => {
    const calls = window.workspaceFixture.calls;
    const found = [...calls].reverse().find((call) => call.body?.state === wanted);
    return (found?.body || {}) as Record<string, unknown>;
  }, state);
}

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

test('saved draft resumes by request id while superseded and refused loads keep safe context', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByText('No saved requests yet.', {exact: true}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('First draft');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const first = await lastPostedBody(page, 'draft');
  const id1 = String(first.intent);
  assert(/^d[0-9a-f]{24}$/.test(id1));
  await page.getByText('First draft', {exact: true}).waitFor();
  const callsBefore = await page.evaluate(() => window.workspaceFixture.calls.length);
  await resetIntent(page);
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === '';
  });
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), callsBefore);
  await resumeIntent(page, id1);
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'First draft';
  });
  await page.getByText('Saved', {exact: true}).waitFor();
  const resumed = await page.evaluate(() => (window as unknown as {resumed: {intent: string}[]}).resumed);
  assert.deepEqual(
    resumed.map((entry) => entry.intent),
    [id1]
  );
  await resetIntent(page);
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Second draft');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Second draft', {exact: true}).waitFor();
  const second = await lastPostedBody(page, 'draft');
  const id2 = String(second.intent);
  assert.notEqual(id2, id1);
  await resetIntent(page);
  await page.evaluate((id: string) => {
    (window as unknown as {intentStore: IntentStore}).intentStore.arm(id);
  }, id1);
  await resumeIntent(page, id1);
  await resumeIntent(page, id2);
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'Second draft';
  });
  await page.evaluate((id: string) => {
    (window as unknown as {intentStore: IntentStore}).intentStore.release(id);
  }, id1);
  await page.waitForTimeout(200);
  assert.equal(await intentObjective(page), 'Second draft');
  const later = await page.evaluate(() => (window as unknown as {resumed: {intent: string}[]}).resumed);
  assert(later.length >= 2);
  assert(later.slice(1).every((entry) => entry.intent === id2));
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.forbid = true;
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Local unsaved edits');
  await resumeIntent(page, id1);
  await page
    .getByText('This saved request is unavailable to the current account. Your current edits are kept.', {exact: true})
    .waitFor();
  assert.equal(await intentObjective(page), 'Local unsaved edits');
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.forbid = false;
  });
  const resumedBeforeSwitch = await page.evaluate(() => (window as unknown as {resumed: unknown[]}).resumed.length);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.sessionLost = true;
    window.workspaceFixture.setUser('2');
  });
  await resetIntent(page);
  await resumeIntent(page, id1);
  await page.waitForTimeout(300);
  assert.equal(await intentObjective(page), '');
  assert.equal(
    await page.evaluate(() => (window as unknown as {resumed: unknown[]}).resumed.length),
    resumedBeforeSwitch
  );
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.sessionLost = false;
    window.workspaceFixture.setUser('1');
  });
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method === 'DELETE').length),
    0
  );
});

test('declined direction resumes for revision and re-explores as a new draft', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page.getByRole('button', {name: 'Decline proposal', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  const declined = await lastPostedBody(page, 'declined');
  assert(/^d[0-9a-f]{24}$/.test(String(declined.intent)));
  await page.getByText('Declined', {exact: false}).first().waitFor();
  await resumeIntent(page, String(declined.intent));
  await page
    .getByText('This direction was declined. Revise the outcome and explore again for a new draft.', {exact: true})
    .waitFor();
  assert.equal(await intentObjective(page), 'Build a leave tracker');
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker for staff');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const successor = await lastPostedBody(page, 'draft');
  assert.equal(successor.predecessor, declined.id);
  assert.deepEqual(successor.ideas, []);
});

test('saved inventory lists states and separates empty from unavailable', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByText('No saved requests yet.', {exact: true}).waitFor();
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.seed([
      {
        intent: 'd111111111111111111111111',
        id: 'd111111111111111111111112',
        state: 'accepted',
        objective: 'Accepted tracker',
        repository: '7',
        tasks: [{key: 'd111111111111111111111113'}],
      },
      {
        intent: 'd222222222222222222222222',
        id: 'd222222222222222222222223',
        state: 'rejected',
        objective: 'Rejected planner',
        repository: '0',
        tasks: [],
      },
      {
        intent: 'd333333333333333333333333',
        id: 'd333333333333333333333334',
        state: 'abandoned',
        objective: 'Abandoned board',
        repository: '0',
        tasks: [],
      },
    ]);
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Fresh draft');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Accepted tracker', {exact: true}).waitFor();
  await page.getByText('Accepted', {exact: false}).first().waitFor();
  await page.getByText('Rejected planner', {exact: true}).waitFor();
  await page.getByText('Rejected', {exact: false}).first().waitFor();
  await page.getByText('Abandoned board', {exact: true}).waitFor();
  await page.getByText('Abandoned', {exact: false}).first().waitFor();
  await page.getByText('Fresh draft', {exact: true}).waitFor();
  await resetIntent(page);
  await page.locator('li', {hasText: 'Fresh draft'}).getByRole('button', {name: 'Resume'}).click();
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'Fresh draft';
  });
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.forbid = true;
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Another draft');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Could not load saved requests. Your edits are kept; try again.', {exact: true}).waitFor();
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.forbid = false;
  });
  await page.getByRole('button', {name: 'Retry', exact: true}).click();
  await page.getByText('Another draft', {exact: true}).waitFor();
});

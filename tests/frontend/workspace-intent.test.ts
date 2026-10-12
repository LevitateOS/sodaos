import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {setupWorkspaceDriver, fixture, reviewFirstIntent} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

type ExplorationMode = 'supported' | 'unsupported' | 'challenges' | 'votes' | 'settled';
interface IntentStore {
  arm(id: string): void;
  release(id: string): void;
  seed(entries: Record<string, unknown>[]): void;
  forbid: boolean;
  sessionLost: boolean;
  exploration: ExplorationMode | undefined;
  generation: {state: string; staleAuth?: boolean; fail?: boolean} | undefined;
  allowanceStale: number;
  allowanceRefuse: boolean;
  allowanceCalls: {method: string; path: string; body: Record<string, unknown> | null}[];
  seedAllowance(entry: Record<string, unknown>): void;
}
// Test-local private intent peer: records saved revisions, serves the declared
// current-revision read and owner-private list, and gates/refuses loads on
// demand. Shared fixtures stay untouched.
async function installIntentStore(page: Page) {
  await page.evaluate(() => {
    const records = new Map<string, Record<string, unknown>>();
    const gates = new Map<string, () => void>();
    const single = (url: string) => url.match(/\/api\/factory\/intents\/([^/?#]+)$/)?.[1] || '';
    const challengeTexts = [
      {
        assumption: 'Everyone on the team can see each request.',
        consequence: 'Individual dates are exposed to the whole team.',
        recommendation: 'Limit visibility to the employee and their manager.',
      },
      {
        assumption: 'Undecided requests wait silently.',
        consequence: 'Nobody knows a request is waiting.',
        recommendation: 'Show undecided requests to the reviewer.',
      },
    ];
    const buildExploration = (
      mode: ExplorationMode,
      json: {id: string; intent: string; objective: string; ideas: {id: string; decision?: string}[]}
    ) => {
      const shared = {
        key: 'shared_review',
        name: 'Shared review workflow',
        audience: 'A person and their reviewer',
        summary: 'Requests are reviewed by another person before completion.',
        steps: [
          {
            actor: 'The person',
            action: 'chooses dates and submits a request',
            result: 'The reviewer can see the request',
          },
          {
            actor: 'The reviewer',
            action: 'makes a decision',
            result: 'The person sees the decision',
            branches: [
              {outcome: 'Approved', result: 'The person sees confirmation'},
              {outcome: 'Declined', result: 'The person sees the reason and can revise the request'},
            ],
          },
        ],
        visibility: 'The person and the reviewer see the request.',
        consequences: 'Reviews take reviewer time.',
        evidence: 'Authored pattern; no external research was performed.',
        outcomes: [json.objective],
      };
      const content = {
        producer: 'soda-authored-v1',
        supported: mode !== 'unsupported',
        intent: json.intent,
        revision: json.id,
        objective: json.objective,
        facts: [json.objective],
        ...(mode === 'unsupported'
          ? {}
          : {
              alternatives: [
                {
                  key: 'single_person',
                  name: 'Small single-person workflow',
                  audience: 'One person managing their own requests',
                  summary: 'Requests stay with one person.',
                  steps: [
                    {
                      actor: 'The person',
                      action: 'records the request',
                      result: 'The request is tracked',
                    },
                  ],
                  visibility: 'Only the person sees the request.',
                  consequences: 'No review step exists.',
                  evidence: 'Authored pattern; no external research was performed.',
                  outcomes: [json.objective],
                },
                shared,
              ],
            }),
        unknowns: [
          'Who can see a request beyond the people named in the walkthrough is not established.',
          'What happens while no decision has been made yet is not established.',
        ],
      };
      return {
        content,
        challenges:
          mode === 'challenges'
            ? json.ideas.slice(0, 2).map((idea, index) => ({idea: idea.id, ...challengeTexts[index]}))
            : [],
        settled: [] as string[],
        ...(mode === 'unsupported'
          ? {recommendation: null, consistency: null}
          : {
              recommendation: {
                producer: 'soda-authored-v1',
                intent: json.intent,
                revision: json.id,
                objective: json.objective,
                recommended: 'shared_review',
                alternative: shared,
                outcomes: [json.objective],
                constraints: ['Only the reviewer can approve.'],
                milestones: [
                  {name: 'First usable request flow', result: 'A person can submit and track a request.'},
                  {name: 'Review decisions', result: 'Reviewers can approve or decline with reasons.'},
                ],
                advice: 'f'.repeat(64),
              },
              consistency: {
                producer: 'soda-authored-v1',
                intent: json.intent,
                revision: json.id,
                mismatches: [],
                confidence: 0.92,
                advice: 'e'.repeat(64),
              },
            }),
      };
    };
    const allowances = new Map<string, Record<string, unknown>>();
    const store = {
      forbid: false,
      sessionLost: false,
      exploration: undefined as ExplorationMode | undefined,
      generation: undefined as {state: string; staleAuth?: boolean; fail?: boolean} | undefined,
      allowanceStale: 0,
      allowanceRefuse: false,
      allowanceCalls: [] as {method: string; path: string; body: Record<string, unknown> | null}[],
      seedAllowance(entry: Record<string, unknown>) {
        allowances.set(String(entry.intent), entry);
      },
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
        if (url.includes('/api/factory/intents/') && url.endsWith('/generation') && method === 'GET') {
          if (!store.generation) return original(input, init);
          if (store.generation.fail) return new Response(null, {status: 503});
          const intent = url.split('/').at(-2) || '';
          const record = records.get(intent);
          const revision = String(record?.id || 'd000000000000000000000000');
          return Response.json({
            state: store.generation.state,
            revision_id: revision,
            intent_id: intent,
            operation_id: 'op-1',
            actor_id: '1',
            native_revision: 3,
            authorization_revision: store.generation.staleAuth ? 'd999999999999999999999999' : revision,
            not_after: 9,
            payload: {},
            intent_digest: 'd',
            task_issue_ids: store.generation.state === 'pending' ? {} : {k1: '1'},
          });
        }
        if (url.includes('/api/factory/intents/') && url.endsWith('/allowance') && method === 'GET') {
          if (store.sessionLost) return new Response(null, {status: 403});
          store.allowanceCalls.push({method, path: url, body: null});
          const found = allowances.get(url.split('/').at(-2) || '');
          if (!found) return Response.json({error: {code: 'allowance_not_found'}}, {status: 404});
          return Response.json(found);
        }
        if (url.endsWith('/api/factory/intent-allowances') && method === 'POST') {
          if (store.sessionLost) return new Response(null, {status: 403});
          if (store.allowanceRefuse) return Response.json({error: {code: 'not_found'}}, {status: 404});
          const body = JSON.parse(String(init?.body || '{}')) as Record<string, unknown>;
          store.allowanceCalls.push({method, path: url, body});
          const current = allowances.get(String(body.intent));
          if (store.allowanceStale > 0 || (current && body.predecessor !== current.id)) {
            store.allowanceStale = Math.max(0, store.allowanceStale - 1);
            return Response.json({error: {code: 'stale_revision'}}, {status: 412});
          }
          const saved = {
            id: String(body.id),
            intent: String(body.intent),
            amount: Number(body.amount),
            reserved: Number(body.reserved || 0),
            consumed: Number(body.consumed || 0),
            unit: 'planning_calls',
          };
          allowances.set(saved.intent, saved);
          return Response.json(saved, {status: 201});
        }
        if (url.endsWith('/explore') && method === 'POST') {
          const response = await original(input, init);
          if (!response.ok || !store.exploration) return response;
          const json = (await response.json()) as {
            id: string;
            intent: string;
            objective: string;
            ideas: {id: string; decision?: string}[];
          };
          if (store.exploration === 'settled') {
            const settled: string[] = [];
            const ideas = json.ideas.map((idea) => {
              if (idea.decision !== 'pending') return idea;
              settled.push(idea.id);
              return {...idea, decision: 'accept', source: 'autonomous', provenance: 'autonomous'};
            });
            const exploration = buildExploration('supported', {...json, ideas});
            exploration.settled = settled;
            return Response.json({...json, ideas, exploration});
          }
          if (store.exploration === 'votes') {
            const extra = {
              id: 'daaaaaaaaaaaaaaaaaaaaaaaa',
              description: 'Suggested nightly summary',
              consequence: 'A summary is sent every night.',
              assumption: false,
              decision: 'pending',
              source: 'autonomous',
              provenance: 'autonomous',
            };
            const ideas = json.ideas.some((idea) => idea.id === extra.id) ? json.ideas : [...json.ideas, extra];
            const exploration = buildExploration('supported', {...json, ideas});
            exploration.challenges = [
              {idea: json.ideas[0]?.id || '', ...challengeTexts[0]},
              {idea: extra.id, ...challengeTexts[1]},
            ];
            return Response.json({...json, ideas, exploration});
          }
          return Response.json({...json, exploration: buildExploration(store.exploration, json)});
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
    const target = document.querySelector('soda-intent') as unknown as {request: string; active: boolean};
    target.active = true;
    target.request = intent;
  }, id);
}
async function intentObjective(page: Page): Promise<string> {
  return page.evaluate(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement ? area.value : 'missing';
  });
}
async function setActor(page: Page, actor: string) {
  await page.evaluate((value: string) => {
    (document.querySelector('soda-intent') as unknown as {actor: string}).actor = value;
  }, actor);
}
async function updateProposal(page: Page) {
  const before = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, before);
}
async function setExplorationMode(page: Page, mode: ExplorationMode) {
  await page.evaluate((value: ExplorationMode) => {
    (window as unknown as {intentStore: IntentStore}).intentStore.exploration = value;
  }, mode);
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
  await installIntentStore(page);
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
  await installIntentStore(page);
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
  await installIntentStore(page);
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
  await page.getByText('Exploration could not run. Your description is saved; try again.', {exact: true}).waitFor();
  await page.getByText('Saved', {exact: true}).waitFor();
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

test('saved inventory names each resume action and publishes list updates', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {inventories: unknown[]}).inventories = [];
    document.addEventListener('intent-inventory', (event) =>
      (window as unknown as {inventories: unknown[]}).inventories.push((event as CustomEvent).detail)
    );
    (window as unknown as {intentStore: IntentStore}).intentStore.seed([
      {
        intent: 'd111111111111111111111111',
        id: 'd111111111111111111111112',
        state: 'draft',
        objective: 'Seeded tracker',
        repository: '0',
        tasks: [],
      },
    ]);
  });
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Fresh draft');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByRole('button', {name: 'Resume Seeded tracker', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Resume Fresh draft', exact: true}).waitFor();
  const inventories = await page.evaluate(
    () => (window as unknown as {inventories: {intents: {objective: string}[]}[]}).inventories
  );
  assert(inventories.length >= 1);
  const latest = inventories.at(-1);
  assert.deepEqual((latest?.intents || []).map((item) => item.objective).sort(), ['Fresh draft', 'Seeded tracker']);
});

test('review leads with the recommended walkthrough, illustration and text equivalent', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'supported');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page
    .getByLabel('Describe what you want to build', {exact: true})
    .fill('I need time off my manager can approve');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const flow = page.locator('section[aria-label="Recommended product flow"]');
  await flow.getByRole('heading', {name: 'Here is how it would work'}).waitFor();
  await flow.getByText('Shared review workflow', {exact: true}).waitFor();
  await flow.getByText('The reviewer', {exact: false}).first().waitFor();
  await flow.getByText('Approved:', {exact: false}).waitFor();
  await flow.getByText('Who sees each result: The person and the reviewer see the request.', {exact: true}).waitFor();
  await flow.getByText('Only the reviewer can approve.', {exact: true}).waitFor();
  await flow.getByText('First usable request flow: A person can submit and track a request.', {exact: true}).waitFor();
  assert.equal(await flow.locator('figure').count(), 1);
  await flow.getByText('Illustrated flow: Shared review workflow.', {exact: false}).waitFor();
  const compared = page.locator('section[aria-label="Compared approaches"]');
  await compared.getByText('Small single-person workflow', {exact: true}).waitFor();
  await compared.getByText('Shared review workflow', {exact: true}).waitFor();
  assert.equal(await compared.getByRole('button').count(), 0);
  await page
    .getByText('Who can see a request beyond the people named in the walkthrough is not established.', {exact: true})
    .waitFor();
  await page.getByText('The walkthrough was checked against the accepted direction.', {exact: true}).waitFor();
  assert.equal(await page.locator('section[aria-label="Questions needing your answer"]').count(), 0);
});

test('change something submits a correction and stale walkthroughs refresh on update', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'supported');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page.getByLabel('What should change?', {exact: true}).fill('Only counting business days');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .nth(1)
    .waitFor();
  const draft = await lastPostedBody(page, 'draft');
  const added = ((draft.ideas || []) as {decision: string; correction?: string; source?: string}[]).find(
    (idea) => idea.decision === 'edit' && idea.correction === 'Only counting business days'
  );
  assert(added);
  assert.equal(added.source, 'requested');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await page
    .getByText('The walkthrough below is out of date. Update the proposal to refresh it.', {exact: true})
    .waitFor();
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page
    .getByText('The walkthrough below is out of date. Update the proposal to refresh it.', {exact: true})
    .waitFor({state: 'detached'});
});

test('assumption challenges batch recommendations with retained answers', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'challenges');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page.getByLabel('What should change?', {exact: true}).fill('Track overtime too');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const questions = page.locator('section[aria-label="Questions needing your answer"]');
  await questions.getByText('Everyone on the team can see each request.', {exact: true}).waitFor();
  await questions.getByText('Undecided requests wait silently.', {exact: true}).waitFor();
  await questions
    .getByText('Recommended resolution: Limit visibility to the employee and their manager.', {exact: true})
    .waitFor();
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled(), true);
  const first = questions.locator('article').first();
  await first
    .getByLabel('Explain a correction to Everyone on the team can see each request.', {exact: true})
    .fill('Only the manager sees dates');
  assert.equal(
    await first
      .getByLabel('Explain a correction to Everyone on the team can see each request.', {exact: true})
      .inputValue(),
    'Only the manager sees dates'
  );
  const second = questions.locator('article').nth(1);
  await second
    .getByRole('button', {name: 'Use recommended resolution for Undecided requests wait silently.', exact: true})
    .click();
  await second.getByText('Answered: using the recommended resolution.', {exact: true}).waitFor();
  await page
    .getByText('The walkthrough below is out of date. Update the proposal to refresh it.', {exact: true})
    .waitFor();
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page
    .getByText('The walkthrough below is out of date. Update the proposal to refresh it.', {exact: true})
    .waitFor({state: 'detached'});
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled(), false);
});

test('unsupported objectives show unknowns without fabricated alternatives', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'unsupported');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page
    .getByText('No authored comparison covers this objective yet. Answer the questions below instead.', {exact: true})
    .waitFor();
  await page.getByText('What happens while no decision has been made yet is not established.', {exact: true}).waitFor();
  assert.equal(await page.locator('section[aria-label="Recommended product flow"]').count(), 0);
  assert.equal(await page.locator('section[aria-label="Compared approaches"]').count(), 0);
  assert.equal(await page.locator('section[aria-label="Questions needing your answer"]').count(), 0);
});

test('planning and judgment failures preserve the draft and identify the actual cause', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    const original = window.fetch;
    let calls = 0;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/explore')) {
          calls += 1;
          if (calls === 1)
            return Promise.resolve(Response.json({error: {code: 'planning_unavailable'}}, {status: 503}));
          if (calls === 2)
            return Promise.resolve(Response.json({error: {code: 'planning_funding_required'}}, {status: 409}));
        }
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText(
      'The TypeSafe System One judgment service could not complete exploration. Your description is saved; try again.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText(
      'Exploration needs a planning allowance. Your description is saved; set the planning allowance and try again.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
});

test('human decisions record attributable provenance when the actor is known', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await setActor(page, '1');
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await updateProposal(page);
  const stamped = await lastPostedBody(page, 'draft');
  const accepted = ((stamped.ideas || []) as {decision: string; provenance?: string; actor?: string}[]).find(
    (idea) => idea.decision === 'accept'
  );
  assert.equal(accepted?.provenance, 'human');
  assert.equal(accepted?.actor, '1');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('pending');
  await updateProposal(page);
  const cleared = await lastPostedBody(page, 'draft');
  const reset = ((cleared.ideas || []) as {decision: string; provenance?: string; actor?: string}[])[0];
  assert.equal(reset?.decision, 'pending');
  assert.equal(reset?.provenance, undefined);
  assert.equal(reset?.actor, undefined);
});

test('idea cards keep votes only where a decision is required', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'votes');
  await setActor(page, '1');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const cards = page.locator('section.intent-review > article');
  assert.equal(await cards.count(), 2);
  assert.equal(await cards.nth(0).getByRole('combobox').count(), 0);
  assert.equal(await cards.nth(1).getByRole('combobox').count(), 0);
  await cards.nth(1).getByText('Suggested', {exact: true}).waitFor();
  const questions = page.locator('section[aria-label="Questions needing your answer"]');
  assert.equal(await questions.locator('article').count(), 2);
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page.getByLabel('What should change?', {exact: true}).fill('Track overtime too');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.waitForFunction(() => {
    const host = document.querySelector('soda-intent') as unknown as {shadowRoot: ShadowRoot; busy: boolean};
    return !host.busy && host.shadowRoot?.querySelectorAll('section.intent-review > article').length === 3;
  });
  assert.equal(await cards.nth(0).getByRole('combobox').count(), 0);
  assert.equal(await cards.nth(1).getByRole('combobox').count(), 0);
  assert.equal(await cards.nth(2).getByRole('combobox').count(), 1);
  await cards.nth(2).getByText('You requested', {exact: true}).waitFor();
  await questions
    .locator('article')
    .nth(0)
    .getByLabel('Explain a correction to Everyone on the team can see each request.', {exact: true})
    .fill('Only the manager sees dates');
  await questions
    .locator('article')
    .nth(1)
    .getByLabel('Explain a correction to Undecided requests wait silently.', {exact: true})
    .fill('Send the summary to the person');
  await page
    .getByText('The walkthrough below is out of date. Update the proposal to refresh it.', {exact: true})
    .waitFor();
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page
    .getByText('The walkthrough below is out of date. Update the proposal to refresh it.', {exact: true})
    .waitFor({state: 'detached'});
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled(), false);
  const draft = await lastPostedBody(page, 'draft');
  const ideas = (draft.ideas || []) as {
    description: string;
    decision: string;
    correction?: string;
    source?: string;
    provenance?: string;
    actor?: string;
  }[];
  const corrected = ideas.find((idea) => idea.description === 'Suggested nightly summary');
  assert.equal(corrected?.decision, 'edit');
  assert.equal(corrected?.correction, 'Send the summary to the person');
  assert.equal(corrected?.source, 'autonomous');
  assert.equal(corrected?.provenance, 'human');
  assert.equal(corrected?.actor, '1');
});

test('accepted review shows authorization, generation progress and unsaved loss', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setActor(page, '1');
  await page.evaluate(() => {
    (window as unknown as {reviewed: unknown}).reviewed = null;
    document.addEventListener('intent-reviewed', (event) => {
      (window as unknown as {reviewed: unknown}).reviewed = (event as CustomEvent).detail;
    });
  });
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await page.getByLabel('Choose how to deliver the accepted workflow', {exact: true}).selectOption('complete_first');
  await updateProposal(page);
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.waitForFunction(() => (window as unknown as {reviewed: unknown}).reviewed !== null);
  const reviewed = await page.evaluate(() => (window as unknown as {reviewed: Record<string, unknown>}).reviewed);
  assert.equal(reviewed.approach, 'complete_first');
  assert.equal(reviewed.authorizedIdeas, 1);
  assert.equal(reviewed.tasks, 1);
  assert.equal(((reviewed.decisions || []) as {decision: string}[]).length, 1);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project', exact: true}).waitFor();
  await page.waitForFunction(() => window.workspaceFixture.calls.some((call) => call.body?.state === 'accepted'));
  const accepted = await lastPostedBody(page, 'accepted');
  const intent = String(accepted.intent);
  assert(/^d[0-9a-f]{24}$/.test(intent));
  const committed = ((accepted.ideas || []) as {decision: string; provenance?: string; actor?: string}[])[0];
  assert.equal(committed?.decision, 'accept');
  assert.equal(committed?.provenance, 'human');
  assert.equal(committed?.actor, '1');
  await page.close();
  const fresh = await fixture(t, 'page', undefined, true);
  await installIntentStore(fresh);
  await fresh.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await fresh.evaluate((record: Record<string, unknown>) => {
    const store = (window as unknown as {intentStore: IntentStore}).intentStore;
    store.seed([record]);
    store.generation = {state: 'pending'};
  }, accepted);
  await resumeIntent(fresh, intent);
  await fresh.getByText('Authorized: 1 idea and 1 work item from your accepted direction.', {exact: true}).waitFor();
  await fresh.getByText('Factory has not created work yet.', {exact: true}).waitFor();
  await fresh.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.generation = {state: 'completed'};
  });
  await fresh.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await fresh
    .getByText('Factory work is created. Execution still needs its own access and finite funding.', {exact: true})
    .waitFor();
  await fresh.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.generation = {state: 'completed', staleAuth: true};
  });
  await fresh.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await fresh
    .getByText('Factory work references an older accepted direction. Refresh the status after updating.', {exact: true})
    .waitFor();
  await fresh.getByLabel('Desired outcome', {exact: true}).fill('Build a leave tracker for everyone');
  await fresh
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await fresh
    .getByText(
      '2 unsaved edits are kept in this view: outcome text, 1 idea decision. ' +
        'Update proposal saves them; the walkthrough, Factory status and repository choice stay on the saved version until then.',
      {exact: true}
    )
    .waitFor();
});

test('settled autonomous baseline advances without per-card votes or questions', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'settled');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const cards = page.locator('section.intent-review > article');
  assert.equal(await cards.count(), 1);
  assert.equal(await cards.nth(0).getByRole('combobox').count(), 0);
  await cards.nth(0).getByText('Decided automatically', {exact: true}).waitFor();
  assert.equal(await page.locator('section[aria-label="Questions needing your answer"]').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled(), false);
  await setActor(page, '1');
  const field = 'Correct the automatic choice: Build a leave tracker';
  await cards.nth(0).getByLabel(field, {exact: true}).fill('Track overtime too');
  await cards.nth(0).getByLabel(field, {exact: true}).fill('');
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const kept = ((restored.ideas || []) as {decision: string; provenance?: string; actor?: number}[])[0];
  assert.equal(kept?.decision, 'accept');
  assert.equal(kept?.provenance, 'autonomous');
  assert.equal(kept?.actor, undefined);
  assert.equal(await cards.nth(0).getByRole('combobox').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Choose repository', exact: true}).isDisabled(), false);
  await cards.nth(0).getByLabel(field, {exact: true}).fill('Track leave without external integrations');
  await updateProposal(page);
  const changed = await lastPostedBody(page, 'draft');
  const edited = ((changed.ideas || []) as {decision: string; correction?: string; provenance?: string}[])[0];
  assert.equal(edited?.decision, 'edit');
  assert.equal(edited?.correction, 'Track leave without external integrations');
  assert.equal(edited?.provenance, 'human');
  assert.equal(await cards.nth(0).getByRole('combobox').count(), 1);
});

async function allowanceLog(page: Page) {
  return page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.allowanceCalls);
}
async function setPlanningCalls(page: Page, value: string) {
  await page.locator('summary', {hasText: 'Planning limits:'}).click();
  await page.getByLabel('Planning calls', {exact: true}).fill(value);
}
async function seedAllowance(page: Page, entry: Record<string, unknown>) {
  await page.evaluate((record: Record<string, unknown>) => {
    (window as unknown as {intentStore: IntentStore}).intentStore.seedAllowance(record);
  }, entry);
}
async function exploreCallCount(page: Page): Promise<number> {
  return page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.path.endsWith('/explore')).length);
}
async function revisionPostCount(page: Page): Promise<number> {
  return page.evaluate(
    () =>
      window.workspaceFixture.calls.filter(
        (call) => call.method === 'POST' && call.path.endsWith('/api/factory/intents')
      ).length
  );
}

test('explore initializes the independent allowance before the first funded exploration', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const log = await allowanceLog(page);
  assert.equal(log.length, 2);
  assert.equal(log[0]?.method, 'GET');
  assert.equal(log[1]?.method, 'POST');
  const body = (log[1]?.body || {}) as Record<string, unknown>;
  assert.equal(body.amount, 8);
  assert.equal(body.ceiling, 8);
  assert.equal(body.unit, 'planning_calls');
  assert.match(String(body.intent), /^d[0-9a-f]{24}$/);
  assert.match(String(body.id), /^d[0-9a-f]{24}$/);
  assert.equal(body.predecessor, undefined);
  assert.equal(body.effect, undefined);
  assert.equal(body.reserved, undefined);
  assert.equal(body.consumed, undefined);
});

test('allowance-only decrease preserves usage and creates no product revision', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const draft = await lastPostedBody(page, 'draft');
  const intent = String(draft.intent);
  await seedAllowance(page, {
    id: 'd777777777777777777777777',
    intent,
    amount: 8,
    reserved: 2,
    consumed: 3,
    unit: 'planning_calls',
  });
  await setPlanningCalls(page, '6');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const log = await allowanceLog(page);
  const posts = log.filter((call) => call.method === 'POST');
  assert.equal(posts.length, 1);
  const body = (posts[0]?.body || {}) as Record<string, unknown>;
  assert.equal(body.predecessor, 'd777777777777777777777777');
  assert.equal(body.amount, 6);
  assert.equal(body.ceiling, 6);
  assert.equal(body.reserved, 2);
  assert.equal(body.consumed, 3);
  assert.equal(body.effect, undefined);
  assert.equal(await revisionPostCount(page), 2);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter((call) => call.method !== 'GET' && !call.path.includes('/api/factory/'))
          .length
    ),
    0
  );
});

test('raising planning calls states the authorization effect', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const draft = await lastPostedBody(page, 'draft');
  await seedAllowance(page, {
    id: 'd777777777777777777777777',
    intent: String(draft.intent),
    amount: 8,
    reserved: 0,
    consumed: 0,
    unit: 'planning_calls',
  });
  await setPlanningCalls(page, '12');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const log = await allowanceLog(page);
  const posts = log.filter((call) => call.method === 'POST');
  assert.equal(posts.length, 1);
  const body = (posts[0]?.body || {}) as Record<string, unknown>;
  assert.equal(body.amount, 12);
  assert.equal(body.effect, 'Authorize 4 additional planning calls (12 total).');
});

test('lowering below used calls fails the floor without exploring', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const draft = await lastPostedBody(page, 'draft');
  await seedAllowance(page, {
    id: 'd777777777777777777777777',
    intent: String(draft.intent),
    amount: 8,
    reserved: 2,
    consumed: 3,
    unit: 'planning_calls',
  });
  await setPlanningCalls(page, '4');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText(
      'Planning calls cannot go below 5 already used. Your description is saved; raise the amount or try again.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await exploreCallCount(page), 0);
  const log = await allowanceLog(page);
  assert.equal(log.filter((call) => call.method === 'POST').length, 0);
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
});

test('stale allowance retries once then reports honestly with the draft kept', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.allowanceStale = 99;
  });
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText('The planning allowance changed while saving. Your description is saved; try again.', {exact: true})
    .waitFor();
  assert.equal(await exploreCallCount(page), 0);
  const log = await allowanceLog(page);
  assert.equal(log.filter((call) => call.method === 'POST').length, 2);
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
});

test('refused allowance blocks exploration as unavailable with the draft kept', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.allowanceRefuse = true;
  });
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText('The planning allowance is unavailable to the current account. Your description is saved; try again.', {
      exact: true,
    })
    .waitFor();
  assert.equal(await exploreCallCount(page), 0);
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
});

test('untouched control adopts externally raised funding instead of clobbering it', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.getByText('Saved', {exact: true}).waitFor();
  const draft = await lastPostedBody(page, 'draft');
  await seedAllowance(page, {
    id: 'd777777777777777777777777',
    intent: String(draft.intent),
    amount: 20,
    reserved: 0,
    consumed: 0,
    unit: 'planning_calls',
  });
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const log = await allowanceLog(page);
  assert.equal(log.filter((call) => call.method === 'POST').length, 0);
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {planningCalls: number}).planningCalls
    ),
    20
  );
});

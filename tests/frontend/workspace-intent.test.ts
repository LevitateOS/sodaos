import test from 'node:test';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {existsSync, mkdirSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
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
  exploreFundingRequired: boolean;
  allowanceCalls: {method: string; path: string; body: Record<string, unknown> | null}[];
  planCalls: {revision: string; operation: string}[];
  planReplyUnconfirmed: boolean;
  planBare: boolean;
  planMismatch: boolean;
  planStatusFail: number;
  planStatusMismatch: boolean;
  planOutcome: string;
  setPlanState(id: string, state: string, attention?: string): void;
  reviewCalls: string[];
  seedAllowance(entry: Record<string, unknown>): void;
  seedHistory(intent: string, revisions: Record<string, unknown>[]): void;
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
    const histories = new Map<string, Record<string, unknown>[]>();
    const plans = new Map<string, {revision: string; state: string; candidates: number}>();
    const store = {
      forbid: false,
      sessionLost: false,
      exploration: undefined as ExplorationMode | undefined,
      generation: undefined as {state: string; staleAuth?: boolean; fail?: boolean} | undefined,
      allowanceStale: 0,
      allowanceRefuse: false,
      exploreFundingRequired: false,
      allowanceCalls: [] as {method: string; path: string; body: Record<string, unknown> | null}[],
      planCalls: [] as {revision: string; operation: string}[],
      planReplyUnconfirmed: false,
      planBare: false,
      planMismatch: false,
      planStatusFail: 0,
      planStatusMismatch: false,
      planOutcome: 'completed' as string,
      reviewCalls: [] as string[],
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
      seedHistory(intent: string, revisions: Record<string, unknown>[]) {
        histories.set(String(intent), revisions);
      },
      setPlanState(id: string, state: string, attention?: string) {
        const entry = plans.get(String(id));
        if (entry) plans.set(String(id), {...entry, state, ...(attention ? {attention} : {})});
      },
    };
    (window as unknown as {intentStore: typeof store}).intentStore = store;
    (window as unknown as {resumed: unknown[]}).resumed = [];
    document.addEventListener('intent-resumed', (event) =>
      (window as unknown as {resumed: unknown[]}).resumed.push((event as CustomEvent).detail)
    );
    const buildPlanWalkthroughs = (record: Record<string, unknown>) => {
      const objective = String(record.objective || '');
      const alternatives = ['team_list', 'guided_steps'].map((key) => ({
        key,
        name: key === 'team_list' ? 'Team project list' : 'Guided project creation',
        audience: 'Project owners and their team',
        summary: 'Track the next action on a project.',
        steps: [
          {
            actor: 'The project owner',
            action: 'adds a project and its next action',
            result: 'The team can see who will do what next',
          },
        ],
        visibility: 'The project team sees its work',
        consequences: 'Updates remain visible in the project',
        evidence: 'The saved project tracker request',
        outcomes: [objective],
      }));
      return {
        producer: 'soda-planning-agent-v1',
        supported: true,
        intent: String(record.intent),
        revision: String(record.id),
        objective,
        facts: [objective],
        unknowns: ['The team can refine labels after trying the first version.'],
        alternatives,
      };
    };
    const buildPlanReview = (
      record: Record<string, unknown>,
      walkthroughs: ReturnType<typeof buildPlanWalkthroughs>,
      mismatched: boolean
    ) => {
      const selected = walkthroughs.alternatives[0] as Record<string, unknown>;
      return {
        content: walkthroughs,
        challenges: [],
        settled: [],
        recommendation: {
          producer: walkthroughs.producer,
          intent: walkthroughs.intent,
          revision: walkthroughs.revision,
          objective: walkthroughs.objective,
          recommended: selected.key,
          alternative: selected,
          outcomes: selected.outcomes,
          constraints: [],
          milestones: [{name: 'Track a project', result: 'The team sees its next action'}],
          advice: 'a'.repeat(64),
        },
        consistency: {
          producer: 'soda-authored-v1',
          intent: walkthroughs.intent,
          revision: walkthroughs.revision,
          mismatches: mismatched
            ? [
                {
                  class: 'outcome_gap',
                  detail: 'The walkthrough does not deliver a requested outcome; recompose the affected steps.',
                },
              ]
            : [],
          confidence: 0.9,
          advice: 'b'.repeat(64),
        },
      };
    };
    const walkthroughState = new Map<
      string,
      {content?: ReturnType<typeof buildPlanWalkthroughs>; reviewed?: unknown}
    >();
    const planEntry = (record: Record<string, unknown>) => {
      const key = String(record.id);
      let entry = walkthroughState.get(key);
      if (!entry) {
        entry = {};
        walkthroughState.set(key, entry);
      }
      if (!entry.content) entry.content = buildPlanWalkthroughs(record);
      return entry;
    };
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const method = init?.method || 'GET';
        const id = single(url);
        // Owner-private saved-history read: only bytes the test seeded are
        // recoverable. Unseeded intents 404 like the native foreign-request
        // refusal, so unsent or in-memory-only work can never present as
        // history.
        if (method === 'GET') {
          const historyFor = url.match(/\/api\/factory\/intents\/(d[0-9a-f]{24})\/history$/);
          if (historyFor) {
            if (store.sessionLost) return new Response(null, {status: 403});
            if (store.forbid) return new Response(null, {status: 404});
            const seeded = histories.get(historyFor[1] || '');
            if (!seeded) return new Response(null, {status: 404});
            return Response.json({intent: historyFor[1], revisions: seeded});
          }
        }
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
            authorization_revision: `${store.generation.staleAuth ? 'd999999999999999999999999' : revision}:${'f'.repeat(64)}`,
            not_after: 9,
            payload: {},
            intent_digest: 'd',
            task_issue_ids: store.generation.state === 'pending' ? {} : {k1: '1'},
          });
        }
        if (url.includes('/api/factory/intents/') && url.endsWith('/plan/review') && method === 'POST') {
          const record = records.get(url.split('/').at(-3) || '');
          if (!record || record.state !== 'draft' || record.repository === '0')
            return new Response(null, {status: 409});
          const body = JSON.parse(String(init?.body || '{}')) as {revision?: string};
          if (body.revision !== record.id) return new Response(null, {status: 412});
          if (!plans.get(String(record.id)) || store.planBare)
            return Response.json({error: {code: 'walkthrough_unavailable'}}, {status: 409});
          const entry = planEntry(record as Record<string, unknown>);
          entry.reviewed = buildPlanReview(
            record as Record<string, unknown>,
            entry.content as ReturnType<typeof buildPlanWalkthroughs>,
            store.planMismatch
          );
          store.reviewCalls.push(String(record.id));
          return Response.json(entry.reviewed);
        }
        if (url.includes('/api/factory/intents/') && url.endsWith('/plan')) {
          const record = records.get(url.split('/').at(-2) || '');
          if (!record || record.state !== 'draft' || record.repository === '0')
            return new Response(null, {status: 409});
          if (method === 'POST') {
            const body = JSON.parse(String(init?.body || '{}')) as {revision: string; operation: string};
            if (body.revision !== record.id || body.operation !== 'propose_intent')
              return new Response(null, {status: 412});
            // A settled revision refuses a second plan POST with zero added
            // spend, like the native closed-attempt admission: recovery must
            // save a successor draft and plan its new id.
            if (['completed', 'failed', 'cancelled'].includes(plans.get(String(body.revision))?.state ?? ''))
              return Response.json({error: {code: 'attempt_closed'}}, {status: 409});
            store.planCalls.push(body);
            plans.set(String(record.id), {revision: String(record.id), state: store.planOutcome, candidates: 1});
            walkthroughState.delete(String(record.id));
            return Response.json(
              {
                assignment_id: 'a'.repeat(32),
                run_id: 'b'.repeat(32),
                revision_id: record.id,
                ...(store.planReplyUnconfirmed ? {launch_confirmed: false} : {}),
              },
              {status: store.planReplyUnconfirmed ? 202 : 201}
            );
          }
          if (store.planStatusFail > 0) {
            store.planStatusFail -= 1;
            return new Response(null, {status: 500});
          }
          if (store.planStatusMismatch) {
            // A well-formed but different revision: the client must refuse
            // it as another draft rather than fail strict parsing.
            const id = String(record.id);
            return Response.json({
              revision: id.slice(0, -1) + (id.endsWith('0') ? '1' : '0'),
              state: 'failed',
            });
          }
          const completed = plans.get(String(record.id));
          if (!completed) return Response.json({revision: record.id, state: 'none'});
          // Only a completed plan carries authored walkthroughs, like the
          // native shape: failed/cancelled/pending plans parse strictly
          // without them.
          if (completed.state !== 'completed' || store.planBare) return Response.json(completed);
          const entry = planEntry(record as Record<string, unknown>);
          return Response.json({
            ...completed,
            walkthroughs: entry.content,
            ...(entry.reviewed ? {review: entry.reviewed} : {}),
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
          if (store.exploreFundingRequired)
            return Response.json({error: {code: 'planning_funding_required'}}, {status: 409});
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
// The declared planning peer settles a bounded proposal; all actions below
// use the emitted workspace's ordinary route and visible controls.
async function planBoundIntent(page: Page, launchUnconfirmed = false) {
  const draft = await lastPostedBody(page, 'draft');
  assert.equal(draft.repository, '7');
  const planning = page.locator('section[aria-label="Draft planning"]');
  if (launchUnconfirmed) {
    await planning
      .getByText(
        'Planning was accepted, but its start response was not confirmed. The status below follows the existing run.',
        {exact: true}
      )
      .waitFor();
    assert.equal(await planning.getByText(/No work was started/).count(), 0);
  }
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  await page.getByRole('button', {name: 'Accept and build', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  return draft;
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

test('intent-first entry binds a draft before native planning and accepted generation', async (t) => {
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
  await page.locator('summary').filter({hasText: 'Delivery approach'}).click();
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
  assert.equal(
    await page.locator('section[aria-label="Draft planning"]').count(),
    0,
    'unbound draft keeps no Draft planning section'
  );
  assert.equal(
    await page.locator('section.intent-review').getByRole('button', {name: 'Choose repository', exact: true}).count(),
    1,
    'unbound draft keeps its primary action in the review footer'
  );
  await page.getByText('Proposed work (1)', {exact: true}).click();
  await page.getByRole('heading', {name: 'Track leave requests without external integrations'}).waitFor();
  await captureSpacesComponent(page, 'intent-corrected-review');
  for (const theme of ['light', 'dark'])
    for (const width of [1440, 800, 390]) {
      await page.setViewportSize({width, height: 900});
      await page.evaluate((value) => {
        document.documentElement.style.colorScheme = value;
      }, theme);
      await captureSpacesComponent(page, `decide-${theme}-${width}`);
    }
  await page.setViewportSize({width: 1440, height: 1000});
  await page.evaluate(() => {
    document.documentElement.style.colorScheme = 'light';
  });
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planReplyUnconfirmed = true;
  });
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  const calls = await page.evaluate(() => window.workspaceFixture.calls);
  const bound = [...calls].reverse().find((call) => call.body?.state === 'draft');
  assert(bound?.body);
  assert.equal(bound.body.repository, '7', 'choosing a repository must save a draft that can be natively planned');
  assert.equal(bound.body.max_issues, 20);
  assert.equal(bound.body.approach, 'complete_first');
  assert.equal(typeof bound.body.department, 'string');
  assert.equal(calls.filter((call) => call.method !== 'GET' && !call.path.includes('/api/factory/')).length, 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(calls.filter((call) => call.path.endsWith('/generate')).length, 0);
  assert.equal(calls.filter((call) => call.body?.state === 'accepted').length, 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length),
    1
  );
  await planBoundIntent(page, true);
  const accepted = await lastPostedBody(page, 'accepted');
  assert.equal(accepted.predecessor, bound.body.id);
  assert.equal(accepted.repository, bound.body.repository);
  assert.deepEqual(accepted.ideas, bound.body.ideas);
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.path.endsWith('/generate')).length),
    1
  );
  assert.deepEqual(await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls), [
    {revision: String(bound.body.id), operation: 'propose_intent'},
  ]);
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
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await planBoundIntent(page);
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

test('review refusal retains acceptance and asks for reconsideration without retry', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/generate'))
          return Promise.resolve(Response.json({error: {code: 'intent_review_required'}}, {status: 409}));
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await reviewFirstIntent(page);
  const draft = await lastPostedBody(page, 'draft');
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await planBoundIntent(page);
  await page
    .getByText(
      'Your product direction is saved. The proposed work needs reconsideration before native work can be created.',
      {
        exact: true,
      }
    )
    .waitFor();
  const accepted = await lastPostedBody(page, 'accepted');
  assert.equal(accepted.intent, draft.intent);
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

test('completed plan without a walkthrough cannot accept and offers an explicit new plan', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  // A retained older plan settled without request-specific walkthroughs.
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planBare = true;
  });
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  const accept = page.getByRole('button', {name: 'Plan walkthrough first', exact: true});
  await accept.waitFor();
  assert.equal(await accept.isDisabled(), true);
  await page
    .getByText('This saved plan has no request-specific walkthrough. Author one in a new plan before accepting.', {
      exact: true,
    })
    .first()
    .waitFor();
  const replan = page.getByRole('button', {name: 'Author walkthrough in a new plan', exact: true});
  await replan.waitFor();
  assert.equal(await replan.isEnabled(), true);
  const noAccept = () =>
    page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.body?.state === 'accepted').length);
  const noGenerate = () =>
    page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.path.endsWith('/generate')).length);
  assert.equal(await noAccept(), 0);
  assert.equal(await noGenerate(), 0);
  const bound = await lastPostedBody(page, 'draft');
  const intent = String(bound.intent);
  const revision = String(bound.id);
  const draftsBefore = await revisionPostCount(page);
  // The explicit recovery mints a successor draft and plans its new id:
  // the settled revision is never replanned (its native attempt is
  // closed), while the predecessor chain, saved records and consumed
  // limits stay intact and the bare result still refuses to build.
  const planRunsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  const settledPlansBefore = await page.evaluate(
    (settled: string) =>
      (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.filter(
        (call) => call.revision === settled
      ).length,
    revision
  );
  await replan.click();
  await page.waitForFunction(
    (started: number) => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > started,
    planRunsBefore
  );
  await page.waitForFunction(
    (settled: string) =>
      ((document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || '') !== settled,
    revision
  );
  const successor = await lastPostedBody(page, 'draft');
  assert.notEqual(String(successor.id), revision);
  assert.equal(successor.intent, intent);
  assert.equal(successor.predecessor, revision);
  assert.equal(successor.repository, bound.repository);
  assert.equal(successor.objective, bound.objective);
  assert.deepEqual(successor.ideas, bound.ideas);
  assert.deepEqual(successor.tasks, bound.tasks);
  assert.equal(successor.max_issues, bound.max_issues);
  assert.equal(successor.planning_calls, bound.planning_calls);
  const replanned = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  assert.equal(replanned, String(successor.id));
  assert.equal(await revisionPostCount(page), draftsBefore + 1);
  const planCalls = await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls);
  assert.equal(planCalls.filter((call) => call.revision === revision).length, settledPlansBefore);
  assert.deepEqual(planCalls.at(-1), {revision: String(successor.id), operation: 'propose_intent'});
  assert.equal(await noAccept(), 0);
  assert.equal(await noGenerate(), 0);
  assert.equal(await accept.isDisabled(), true);
  // A fresh authored walkthrough unblocks the explicit check and build.
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planBare = false;
  });
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  await page.getByRole('button', {name: 'Accept and build', exact: true}).click();
  await page.waitForFunction(() => window.workspaceFixture.calls.some((call) => call.body?.state === 'accepted'));
  const accepted = await lastPostedBody(page, 'accepted');
  assert.equal(accepted.intent, intent);
  assert.equal(accepted.predecessor, String(successor.id));
  assert.equal(await noGenerate(), 1);
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.reviewCalls.length),
    1
  );
});

test('failed plan retries through the primary Continue action with a successor draft', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  const draftsAtReview = await revisionPostCount(page);
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  // The first planning POST records the failed outcome directly: no
  // successor exists yet, so the identical revision is planned once.
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const bound = await lastPostedBody(page, 'draft');
  const intent = String(bound.intent);
  const revision = String(bound.id);
  assert.equal(await revisionPostCount(page), draftsAtReview + 1);
  assert.deepEqual(await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls), [
    {revision, operation: 'propose_intent'},
  ]);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await planning.getByText('Planning failed. Your direction is saved; try again.', {exact: true}).first().waitFor();
  const settledPlansBefore = await page.evaluate(
    (settled: string) =>
      (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.filter(
        (call) => call.revision === settled
      ).length,
    revision
  );
  // The retry run behaves differently from the failed one.
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'pending';
  });
  const primary = planning.getByRole('button', {name: 'Continue to planning', exact: true});
  await primary.waitFor();
  assert.equal(await primary.isEnabled(), true);
  const planRunsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  await primary.click();
  await page.waitForFunction(
    (started: number) => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > started,
    planRunsBefore
  );
  await page.waitForFunction(
    (settled: string) =>
      ((document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || '') !== settled,
    revision
  );
  // The explicit retry mints a successor draft and plans its new id: the
  // failed revision is never replanned (its native attempt is closed),
  // while the predecessor chain, saved records and consumed limits stay
  // intact.
  const successor = await lastPostedBody(page, 'draft');
  assert.notEqual(String(successor.id), revision);
  assert.equal(successor.intent, intent);
  assert.equal(successor.predecessor, revision);
  assert.equal(successor.repository, bound.repository);
  assert.equal(successor.objective, bound.objective);
  assert.deepEqual(successor.ideas, bound.ideas);
  assert.deepEqual(successor.tasks, bound.tasks);
  assert.equal(successor.max_issues, bound.max_issues);
  assert.equal(successor.planning_calls, bound.planning_calls);
  const replanned = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  assert.equal(replanned, String(successor.id));
  assert.equal(await revisionPostCount(page), draftsAtReview + 2);
  const planCalls = await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls);
  assert.equal(planCalls.filter((call) => call.revision === revision).length, settledPlansBefore);
  assert.deepEqual(planCalls.at(-1), {revision: String(successor.id), operation: 'propose_intent'});
  await page.getByText('Planning is running. The status below follows the run.', {exact: true}).first().waitFor();
});

test('failed plan retries through Plan work with a successor draft', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const bound = await lastPostedBody(page, 'draft');
  const revision = String(bound.id);
  const settledPlansBefore = await page.evaluate(
    (settled: string) =>
      (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.filter(
        (call) => call.revision === settled
      ).length,
    revision
  );
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'pending';
  });
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByText('Planning details', {exact: true}).click();
  const planWork = planning.getByRole('button', {name: 'Plan work', exact: true});
  await planWork.waitFor();
  assert.equal(await planWork.isEnabled(), true);
  const planRunsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  await planWork.click();
  await page.waitForFunction(
    (started: number) => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > started,
    planRunsBefore
  );
  const successor = await lastPostedBody(page, 'draft');
  assert.notEqual(String(successor.id), revision);
  assert.equal(successor.predecessor, revision);
  const planCalls = await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls);
  assert.equal(planCalls.filter((call) => call.revision === revision).length, settledPlansBefore);
  assert.deepEqual(planCalls.at(-1), {revision: String(successor.id), operation: 'propose_intent'});
});

test('repeated polls preserve failed guidance and transitions restate it', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const revision = String((await lastPostedBody(page, 'draft')).id);
  const planning = page.locator('section[aria-label="Draft planning"]');
  const refresh = planning.getByRole('button', {name: 'Refresh status', exact: true});
  const runningNotice = 'Planning is running. The status below follows the run.';
  const failedNotice = 'Planning failed. Your direction is saved; try again.';
  const pendingNotice = 'Your request is saved. Its planning progress and any question appear here.';
  // The explicit post reports the settled failure directly: a run that is
  // already failed when the follow-up fetch lands is never announced as
  // running.
  await planning.getByText(failedNotice, {exact: true}).first().waitFor();
  assert.equal(await planning.getByText(runningNotice, {exact: true}).count(), 0);
  // A repeated failed poll keeps the failure guidance instead of blanking it.
  await refresh.click();
  await planning.getByText(failedNotice, {exact: true}).first().waitFor();
  // A genuine pending transition restates the guidance.
  await page.evaluate(
    (id: string) => (window as unknown as {intentStore: IntentStore}).intentStore.setPlanState(id, 'pending'),
    revision
  );
  await refresh.click();
  await planning.getByText(pendingNotice, {exact: true}).waitFor();
  // A genuine failed transition restates the failure guidance.
  await page.evaluate(
    (id: string) => (window as unknown as {intentStore: IntentStore}).intentStore.setPlanState(id, 'failed'),
    revision
  );
  await refresh.click();
  await planning.getByText(failedNotice, {exact: true}).first().waitFor();
  // The failure message is visible exactly once: the action notice is
  // suppressed when it duplicates the status line, keeping one accessible
  // announcement.
  assert.equal(await planning.getByText(failedNotice, {exact: true}).count(), 1);
  // Repeating the failed state keeps the failure guidance.
  await refresh.click();
  await planning.getByText(failedNotice, {exact: true}).first().waitFor();
  assert.equal(await planning.getByText(failedNotice, {exact: true}).count(), 1);
  // A genuine completed transition clears the notice for the review flow.
  await page.evaluate(
    (id: string) => (window as unknown as {intentStore: IntentStore}).intentStore.setPlanState(id, 'completed'),
    revision
  );
  await refresh.click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).waitFor();
  // Raw document queries cannot pierce the component shadow root; pierce it
  // explicitly when asserting rendered text is gone.
  await page.waitForFunction(
    (text: string) =>
      (document
        .querySelector('soda-intent')
        ?.shadowRoot?.querySelector('section[aria-label="Draft planning"]')
        ?.textContent?.includes(text) ?? true) === false,
    failedNotice
  );
});

test('run-active plan sends no new plan POST from either button', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'pending';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  const running = planning.getByRole('button', {name: 'Planning…', exact: true});
  await running.waitFor();
  assert.equal(await running.isDisabled(), true);
  await planning.getByText('Planning details', {exact: true}).click();
  // While a run is live, Plan work is not offered at all.
  assert.equal(await planning.getByRole('button', {name: 'Plan work', exact: true}).count(), 0);
  assert.deepEqual(await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls), [
    {revision: String((await lastPostedBody(page, 'draft')).id), operation: 'propose_intent'},
  ]);
});

test('explicit retry with an unverifiable live status posts nothing and says so', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await planning.getByText('Planning failed. Your direction is saved; try again.', {exact: true}).first().waitFor();
  const draftsBefore = await revisionPostCount(page);
  const postsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  // The live verification GET keeps failing, so the retry must stop with
  // truthful guidance instead of deciding from the stale cached failure.
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planStatusFail = 10;
  });
  await planning.getByText('Planning details', {exact: true}).click();
  await planning.getByRole('button', {name: 'Plan work', exact: true}).click();
  await planning
    .getByText('Planning status could not be verified. Your request is saved; refresh status before trying again.', {
      exact: true,
    })
    .waitFor();
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length),
    postsBefore
  );
  assert.equal(await revisionPostCount(page), draftsBefore);
});

test('explicit retry with a mismatched live status posts nothing and says so', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await planning.getByText('Planning failed. Your direction is saved; try again.', {exact: true}).first().waitFor();
  const draftsBefore = await revisionPostCount(page);
  const postsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  // The live status answers another draft, so the retry must stop instead
  // of minting a successor from the stale cached failure.
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planStatusMismatch = true;
  });
  await planning.getByText('Planning details', {exact: true}).click();
  await planning.getByRole('button', {name: 'Plan work', exact: true}).click();
  await planning
    .getByText('Planning status answers another draft. Refresh status before trying again.', {exact: true})
    .waitFor();
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length),
    postsBefore
  );
  assert.equal(await revisionPostCount(page), draftsBefore);
});

test('explicit retry against a live pending run posts nothing and says so', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const revision = String((await lastPostedBody(page, 'draft')).id);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await planning.getByText('Planning failed. Your direction is saved; try again.', {exact: true}).first().waitFor();
  const draftsBefore = await revisionPostCount(page);
  const postsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  // The cached failure is stale: the run is actually live again, so the
  // retry must report progress instead of minting a successor.
  await page.evaluate(
    (id: string) => (window as unknown as {intentStore: IntentStore}).intentStore.setPlanState(id, 'pending'),
    revision
  );
  await planning.getByText('Planning details', {exact: true}).click();
  await planning.getByRole('button', {name: 'Plan work', exact: true}).click();
  await planning
    .getByText('Your request is saved. Its planning progress and any question appear here.', {exact: true})
    .waitFor();
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length),
    postsBefore
  );
  assert.equal(await revisionPostCount(page), draftsBefore);
});

test('explicit retry against live attention posts nothing and asks for the decision', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'failed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const revision = String((await lastPostedBody(page, 'draft')).id);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await planning.getByText('Planning failed. Your direction is saved; try again.', {exact: true}).first().waitFor();
  const draftsBefore = await revisionPostCount(page);
  const postsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  // The cached failure is stale: the run now waits on a product decision,
  // so the retry must surface that instead of planning directly.
  await page.evaluate(
    (id: string) =>
      (window as unknown as {intentStore: IntentStore}).intentStore.setPlanState(
        id,
        'attention',
        'Choose who approves equipment requests'
      ),
    revision
  );
  await planning.getByText('Planning details', {exact: true}).click();
  await planning.getByRole('button', {name: 'Plan work', exact: true}).click();
  await planning
    .getByText('Planning needs a product decision: Choose who approves equipment requests', {exact: true})
    .first()
    .waitFor();
  // In attention Plan work would only duplicate the primary, so it is gone.
  assert.equal(await planning.getByRole('button', {name: 'Plan work', exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length),
    postsBefore
  );
  assert.equal(await revisionPostCount(page), draftsBefore);
  // A product question keeps Edit request as the primary: it only focuses
  // the Desired outcome, with no save and no planning call.
  const callsBefore = await page.evaluate(() => window.workspaceFixture.calls.length);
  await planning.getByRole('button', {name: 'Edit request', exact: true}).click();
  await page.waitForFunction(
    () => document.querySelector('soda-intent')?.shadowRoot?.activeElement?.tagName === 'TEXTAREA'
  );
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), callsBefore);
});

test('immediate cancelled plan reports cancellation without a running text', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'cancelled';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  const cancelledNotice = 'Planning was cancelled before it could report.';
  await planning.getByText(cancelledNotice, {exact: true}).first().waitFor();
  assert.equal(
    await planning.getByText('Planning is running. The status below follows the run.', {exact: true}).count(),
    0
  );
});

test('unconfirmed start keeps its notice over a pending status without a running text', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'pending';
    (window as unknown as {intentStore: IntentStore}).intentStore.planReplyUnconfirmed = true;
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  // Only the launch response can lift start uncertainty: a pending
  // follow-up fetch must not replace it with a running text.
  await planning
    .getByText(
      'Planning was accepted, but its start response was not confirmed. The status below follows the existing run.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await planning.getByText('Planning is running. The status below follows the run.', {exact: true}).count(),
    0
  );
});

test('compared approaches describe JEV review state honestly before and after the check', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const compared = page.locator('section[aria-label="Compared approaches"]');
  await compared.getByText('Team project list', {exact: true}).waitFor();
  // Before the check, no recommendation may be claimed.
  await compared
    .getByText(
      'JEV will compare these alternatives when you check the walkthrough; you can correct them before acceptance.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await compared.getByText(/JEV recommends/, {exact: false}).count(), 0);
  // After a real check, the actual recommendation is named.
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  await compared
    .getByText('JEV recommends Team project list; you can correct it before acceptance.', {exact: true})
    .waitFor();
});

test('immediate completed plan clears the notice without a running text', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'completed';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  // The settled completion is announced by the status line; no action
  // notice and no running text may survive it.
  await planning
    .getByText('Planning is complete. Check the walkthrough with JEV before accepting.', {exact: true})
    .waitFor();
  assert.equal(
    await planning.getByText('Planning is running. The status below follows the run.', {exact: true}).count(),
    0
  );
});

test('immediate stopped plan retries from a successor draft without a running text', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planOutcome = 'attention';
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.waitForFunction(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > 0);
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning
    .getByText('Planning stopped before it could propose work. Your request is saved.', {exact: true})
    .waitFor();
  assert.equal(
    await planning.getByText('Planning is running. The status below follows the run.', {exact: true}).count(),
    0
  );
  // The planner asked nothing, so the primary retries like a failed plan:
  // one successor draft, then one planning call for its new revision.
  assert.equal(await planning.getByRole('button', {name: 'Edit request', exact: true}).count(), 0);
  const draftsBefore = await revisionPostCount(page);
  const postsBefore = await page.evaluate(
    () => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length
  );
  await planning.getByRole('button', {name: 'Continue to planning', exact: true}).click();
  await page.waitForFunction(
    (before: number) => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.length > before,
    postsBefore
  );
  assert.equal(await revisionPostCount(page), draftsBefore + 1);
  const successor = String((await lastPostedBody(page, 'draft')).id);
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.planCalls.at(-1)?.revision),
    successor
  );
});

test('walkthrough findings block acceptance until the direction is corrected', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planMismatch = true;
  });
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  const blocked = page.getByRole('button', {name: 'Correct walkthrough first', exact: true});
  await blocked.waitFor();
  assert.equal(await blocked.isDisabled(), true);
  await page.getByText('Needs attention before acceptance', {exact: true}).waitFor();
  // The blocked status line states the actual next action instead of
  // inviting acceptance while findings block it.
  await planning
    .getByText(
      'Correct or clarify the request above for the walkthrough findings (your objective and choices stay), then plan again and check before authorizing work.',
      {exact: true}
    )
    .waitFor();
  // The check notice names the supported correction path: the authored
  // walkthrough cannot be edited directly, so the request is corrected
  // while the objective and choices stay, then planned and checked again.
  await planning
    .getByText(
      'The authored walkthrough needs a correction. Correct or clarify the request above (your objective and choices stay with your authorship), then plan again and check before accepting.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await planning
      .getByText('Planning is complete. JEV checked the walkthrough; Accept and build creates its work.', {exact: true})
      .count(),
    0
  );
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.body?.state === 'accepted').length),
    0
  );
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.path.endsWith('/generate')).length),
    0
  );
  // Refreshing restores the reviewed content without consulting again.
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await blocked.waitFor();
  assert.equal(await blocked.isDisabled(), true);
  assert.equal(
    await page.evaluate(() => (window as unknown as {intentStore: IntentStore}).intentStore.reviewCalls.length),
    1
  );
  // The correction-path guidance survives the refresh on both lines.
  await planning
    .getByText(
      'Correct or clarify the request above for the walkthrough findings (your objective and choices stay), then plan again and check before authorizing work.',
      {exact: true}
    )
    .waitFor();
  await planning
    .getByText(
      'The authored walkthrough needs a correction. Correct or clarify the request above (your objective and choices stay with your authorship), then plan again and check before accepting.',
      {exact: true}
    )
    .waitFor();
});

test('explicit check on a blocked plan summarizes state near the top and focuses the findings', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await reviewFirstIntent(page);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.planMismatch = true;
  });
  const planning = page.locator('section[aria-label="Draft planning"]');
  await planning.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await page.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  // The state/action summary sits near the top, directly under the review
  // heading and its state chip.
  await page
    .locator('section.intent-review > h2 + .state-chip')
    .getByText('Checked · needs correction', {exact: true})
    .waitFor();
  const summary = page.locator('section.intent-review > h2 + .state-chip + p');
  await summary.getByText('Check found 1 walkthrough finding blocking acceptance.', {exact: false}).waitFor();
  await summary
    .getByText(
      'Correct or clarify the request above for the walkthrough findings (your objective and choices stay), then plan again and check before authorizing work.',
      {exact: false}
    )
    .waitFor();
  // The explicit Check lands keyboard focus on the real findings.
  await page.waitForFunction(() => {
    const root = document.querySelector('soda-intent')?.shadowRoot;
    const target = root?.querySelector('section[aria-label="Consistency check"] h3');
    return !!target && root?.activeElement === target;
  });
  await page.getByText('Needs attention before acceptance', {exact: true}).waitFor();
  // The summary link returns focus to the real findings after moving away.
  await page.evaluate(() => {
    (document.querySelector('soda-intent')?.shadowRoot?.activeElement as HTMLElement | null)?.blur();
  });
  await summary.getByRole('button', {name: 'See findings', exact: true}).click();
  await page.waitForFunction(() => {
    const root = document.querySelector('soda-intent')?.shadowRoot;
    const target = root?.querySelector('section[aria-label="Consistency check"] h3');
    return !!target && root?.activeElement === target;
  });
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
test('proposal refusals name the task limit only for task-limit failures', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.evaluate(() => {
    const original = window.fetch;
    const codes = ['invalid_proposal_task_limit', 'invalid_proposal'];
    (window as unknown as {refusedExplores: number}).refusedExplores = 0;
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/explore') && codes.length > 0) {
          const code = codes.shift();
          (window as unknown as {refusedExplores: number}).refusedExplores += 1;
          return Promise.resolve(Response.json({error: {code}}, {status: 400}));
        }
        return original(input, init);
      },
    });
    return window.workspaceFixture.api.refresh();
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText(
      'The derived work exceeds your selected issue limit. Your description is saved; review the planning limit or narrow the approach, then explore again. Limits never increase on their own.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page
    .getByText(
      'The derived proposal failed review validation. Your description is saved; review the approach and explore again.',
      {exact: true}
    )
    .waitFor();
  // Both failures came from explicit clicks; the UI never retried on its own.
  assert.equal(await page.evaluate(() => (window as unknown as {refusedExplores: number}).refusedExplores), 2);
  assert.equal(
    await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(),
    'Build a leave tracker'
  );
});

test('a request resumes by id while superseded and refused loads keep safe context', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const id1 = 'd111111111111111111111111';
  const id2 = 'd222222222222222222222222';
  await page.evaluate(
    ([first, second]: string[]) => {
      (window as unknown as {intentStore: IntentStore}).intentStore.seed([
        {
          intent: first,
          id: 'd111111111111111111111112',
          state: 'draft',
          objective: 'First draft',
          repository: '0',
          tasks: [],
        },
        {
          intent: second,
          id: 'd222222222222222222222223',
          state: 'draft',
          objective: 'Second draft',
          repository: '0',
          tasks: [],
        },
      ]);
    },
    [id1, id2]
  );
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await resumeIntent(page, id1);
  await page.waitForFunction(() => {
    const area = document.querySelector('soda-intent')?.shadowRoot?.querySelector('textarea');
    return area instanceof HTMLTextAreaElement && area.value === 'First draft';
  });
  const resumed = await page.evaluate(() => (window as unknown as {resumed: {intent: string}[]}).resumed);
  assert.deepEqual(
    resumed.map((entry) => entry.intent),
    [id1]
  );
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
  await flow.locator('summary').filter({hasText: 'Delivery milestones'}).click();
  await flow.getByText('First usable request flow: A person can submit and track a request.', {exact: true}).waitFor();
  assert.equal(await flow.locator('figure').count(), 1);
  await flow.locator('summary').filter({hasText: 'Flow diagram'}).click();
  await flow.getByText('Illustrated flow: Shared review workflow.', {exact: false}).waitFor();
  const compared = page.locator('section[aria-label="Compared approaches"]');
  await compared.getByText('Small single-person workflow', {exact: true}).waitFor();
  await compared.getByText('Shared review workflow', {exact: true}).waitFor();
  assert.equal(await compared.getByRole('button').count(), 0);
  await page
    .getByText('Who can see a request beyond the people named in the walkthrough is not established.', {exact: true})
    .waitFor();
  await page.getByText('The walkthrough was checked against the saved direction.', {exact: true}).waitFor();
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
  await page.locator('summary').filter({hasText: 'Delivery approach'}).click();
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
  await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await planBoundIntent(page);
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
    (window as unknown as {intentStore: IntentStore}).intentStore.generation = {state: 'committed'};
  });
  await fresh.getByRole('button', {name: 'Refresh status', exact: true}).click();
  await fresh.getByText('Factory confirmed the native work.', {exact: true}).waitFor();
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
  assert.equal(await page.locator('section.intent-review > article').count(), 0);
  await page.locator('summary').filter({hasText: 'Recorded choices'}).click();
  const cards = page.locator('section.intent-review > article, .recorded-choices > article');
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

async function setMaxIssues(page: Page, value: string) {
  await page.locator('summary', {hasText: 'More options'}).click();
  await page.getByLabel('Maximum work items', {exact: true}).fill(value);
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

test('back-to-review transition focuses the review heading through focusReview', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await setExplorationMode(page, 'supported');
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await updateProposal(page);
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  // C's focusSetupPanel consumer will call focusReview here once integrated;
  // invoke it directly to prove the U contract in the real transition.
  await page.evaluate(async () => {
    const host = document.querySelector('soda-intent') as unknown as {
      focusReview(): void;
      updateComplete: Promise<unknown>;
    };
    host.focusReview();
    await host.updateComplete;
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  const focused = await page.evaluate(() => {
    const root = document.querySelector('soda-intent')?.shadowRoot;
    const target = root?.querySelector('section.intent-review h2');
    return {isTarget: !!target && root?.activeElement === target, text: target?.textContent || ''};
  });
  assert.equal(focused.isTarget, true);
  assert.equal(focused.text, 'Review your product direction');
});

test('resumed accepted request exports its durable repository on intent-resumed', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.evaluate(() => {
    (window as unknown as {intentStore: IntentStore}).intentStore.seed([
      {
        id: 'd444444444444444444444444',
        intent: 'd555555555555555555555555',
        repository: '7',
        objective: 'Accepted tracker',
        ideas: [],
        tasks: [],
        state: 'accepted',
        max_issues: 20,
        planning_calls: 8,
      },
    ]);
  });
  await resumeIntent(page, 'd555555555555555555555555');
  await page.waitForFunction(() => (window as unknown as {resumed: unknown[]}).resumed.length > 0);
  const detail = await page.evaluate(() => (window as unknown as {resumed: Record<string, unknown>[]}).resumed[0]);
  assert.deepEqual(detail, {
    intent: 'd555555555555555555555555',
    revision: 'd444444444444444444444444',
    state: 'accepted',
    repository: '7',
  });
});

test('intent live status persists as one zero-space container across loading and error', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  // The role=status container exists before any status message occurs.
  await page.evaluate(() => {
    const node = document.querySelector('soda-intent')?.shadowRoot?.querySelector('p.intent-live-status');
    if (!(node instanceof HTMLParagraphElement)) throw Error('Missing persistent live-status container');
    if (node.getAttribute('role') !== 'status') throw Error('Live-status container lacks role=status');
    (window as unknown as {liveStatusNode: HTMLParagraphElement}).liveStatusNode = node;
  });
  // Idle: present in the tree, empty, and taking no visible space.
  const idle = await page.evaluate(() => {
    const node = (window as unknown as {liveStatusNode: HTMLElement}).liveStatusNode;
    const style = getComputedStyle(node);
    return {
      text: node.textContent,
      height: node.getBoundingClientRect().height,
      display: style.display,
      visibility: style.visibility,
      ariaHidden: node.getAttribute('aria-hidden'),
    };
  });
  assert.equal(idle.text, '');
  assert.equal(idle.height, 0);
  assert.notEqual(idle.display, 'none');
  assert.notEqual(idle.visibility, 'hidden');
  assert.equal(idle.ariaHidden, null);
  // Gate the draft save so the busy note stays observable, then fail it.
  await page.evaluate(() => {
    const pristine = window.fetch.bind(window);
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    (window as unknown as {releaseIntentSave: () => void}).releaseIntentSave = () => release();
    Object.defineProperty(window, 'fetch', {
      configurable: true,
      value: (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes('/api/factory/intents') && init?.method === 'POST')
          return gate.then(() => Promise.resolve(new Response(null, {status: 500})));
        return pristine(input, init);
      },
    });
  });
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  // Busy note appears in the SAME container node.
  await page.waitForFunction(
    () => (window as unknown as {liveStatusNode: Element}).liveStatusNode.textContent === 'Saving and planning…'
  );
  assert.equal(
    await page.evaluate(
      () =>
        document.querySelector('soda-intent')?.shadowRoot?.querySelector('p.intent-live-status') ===
        (window as unknown as {liveStatusNode: unknown}).liveStatusNode
    ),
    true
  );
  await page.evaluate(() => (window as unknown as {releaseIntentSave: () => void}).releaseIntentSave());
  // The failed save reports in the SAME container node.
  await page.waitForFunction(
    () =>
      (window as unknown as {liveStatusNode: Element}).liveStatusNode.textContent ===
      'Could not save or plan this request. Your choices are kept; try again.'
  );
  assert.equal(
    await page.evaluate(
      () =>
        document.querySelector('soda-intent')?.shadowRoot?.querySelector('p.intent-live-status') ===
        (window as unknown as {liveStatusNode: unknown}).liveStatusNode
    ),
    true
  );
});

// FJ-358 explicit-scope read-projection regressions: drive a real three-idea
// draft, then resume a successor carrying a stored explicit subset using real
// posted shapes (in-test injection; the shared fixtures stay untouched).
async function resumeExplicitSubset(
  page: Page,
  subset: number[],
  options?: {state?: string; decline?: number[]}
): Promise<{intent: string; ids: string[]; revision: string}> {
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await setExplorationMode(page, 'supported');
  await setActor(page, '1');
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await updateProposal(page);
  for (const change of ['Track overtime too', 'Summarize nightly state']) {
    const before = await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
    );
    await page.getByRole('button', {name: 'Change something', exact: true}).click();
    await page.locator('.change-field').getByLabel('What should change?', {exact: true}).fill(change);
    await page.getByRole('button', {name: 'Submit change', exact: true}).click();
    await page.waitForFunction((prev: string) => {
      const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
      return !host.busy && (host.revision?.id || '') !== prev;
    }, before);
  }
  assert.equal(await page.getByRole('combobox', {name: /^Choice:/}).count(), 3);
  const current = await page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown};
    return JSON.parse(JSON.stringify(host.revision)) as Record<string, unknown>;
  });
  const ids = (current.ideas as {id: string}[]).map((idea) => idea.id);
  assert.equal(ids.length, 3);
  const subsetIDs = subset.map((index) => ids[index]);
  const declinedIDs = (options?.decline ?? []).map((index) => ids[index]);
  const next = `${String(current.id).slice(0, -1)}${String(current.id).endsWith('0') ? '1' : '0'}`;
  await page.evaluate(
    ({entry}: {entry: Record<string, unknown>}) => {
      (window as unknown as {intentStore: IntentStore}).intentStore.seed([entry]);
    },
    {
      entry: {
        ...current,
        id: next,
        predecessor: current.id,
        ...(options?.state ? {state: options.state} : {}),
        ideas: (current.ideas as Record<string, unknown>[]).map((idea, index) => {
          if (!options?.decline?.includes(index)) return idea;
          const narrowed: Record<string, unknown> = {...idea, decision: 'decline'};
          delete narrowed.correction;
          delete narrowed.provenance;
          delete narrowed.actor;
          return narrowed;
        }),
        // Canonically valid stored shape: no declined idea in the effective
        // set, and work links cover exactly the remaining explicit subset,
        // mirroring the backend resolved-scope contract.
        tasks: (current.tasks as {idea_ids: string[]}[]).filter(
          (task) =>
            task.idea_ids.every((id) => subsetIDs.includes(id)) && !task.idea_ids.some((id) => declinedIDs.includes(id))
        ),
        effective_outcomes: subsetIDs.filter((id) => !declinedIDs.includes(id)),
      },
    }
  );
  await resetIntent(page);
  await resumeIntent(page, String(current.intent));
  await page.waitForFunction((wanted: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && host.revision?.id === wanted;
  }, next);
  return {intent: String(current.intent), ids, revision: next};
}

test('explicit subset with a removed idea stays narrowed instead of broadening', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {ids} = await resumeExplicitSubset(page, [0, 1]);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .nth(1)
    .selectOption('decline');
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  assert.deepEqual(saved.effective_outcomes, [ids[0]]);
  const ideas = saved.ideas as {id: string; decision: string}[];
  assert.equal(ideas.find((idea) => idea.id === ids[1])?.decision, 'decline');
  assert.equal(ideas.find((idea) => idea.id === ids[2])?.decision, 'edit');
});

const scopeGateMessage =
  'Your latest choice emptied the explicitly narrowed scope while decided ideas remain. Review the authorized ideas explicitly before saving; nothing was saved and no remaining idea was authorized.';

test('narrowing an explicit subset to empty refuses the silent save until explicitly reviewed', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {ids} = await resumeExplicitSubset(page, [0]);
  // Canonical stored shape: only the subset idea carries a work-linked
  // choice, so the excluded decided ideas have no control of their own.
  assert.equal(await page.getByRole('combobox', {name: /^Choice:/}).count(), 1);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  const before = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  const draftsBefore = await page.evaluate(
    () => window.workspaceFixture.calls.filter((call) => call.body?.state === 'draft').length
  );
  await page.getByRole('button', {name: 'Update proposal', exact: true}).click();
  await page.getByText(scopeGateMessage, {exact: true}).waitFor();
  await page.waitForFunction(() => {
    const host = document.querySelector('soda-intent') as unknown as {busy: boolean};
    return !host.busy;
  });
  assert.equal(
    await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
    ),
    before
  );
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.body?.state === 'draft').length),
    draftsBefore
  );
  assert.equal(
    await page
      .getByRole('combobox', {name: /^Choice:/})
      .first()
      .inputValue(),
    'decline'
  );
  // The refusal renders the excluded decided ideas with direct controls:
  // the removed member offers re-acceptance, the remaining ones decline.
  await page.getByRole('button', {name: `Re-accept narrowed idea ${ids[0]}`, exact: true}).waitFor();
  await page.getByRole('button', {name: `Decline remaining idea ${ids[1]}`, exact: true}).click();
  await page.getByRole('button', {name: `Decline remaining idea ${ids[2]}`, exact: true}).click();
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  assert.deepEqual(
    (saved.ideas as {decision: string}[]).map((idea) => idea.decision),
    ['decline', 'decline', 'decline']
  );
  assert.deepEqual(saved.effective_outcomes, []);
  assert.equal(await page.getByText(scopeGateMessage, {exact: true}).count(), 0);
  assert.equal(await page.getByRole('group', {name: 'Narrowed scope recovery'}).count(), 0);
});

test('intent-reviewed projects the resolved explicit subset instead of all decided ideas', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids, revision} = await resumeExplicitSubset(page, [0, 1]);
  await page.evaluate(() => {
    (window as unknown as {reviewed: unknown}).reviewed = null;
    document.addEventListener('intent-reviewed', (event) => {
      (window as unknown as {reviewed: unknown}).reviewed = (event as CustomEvent).detail;
    });
  });
  await page.getByRole('button', {name: 'Choose repository', exact: true}).click();
  await page.waitForFunction(() => (window as unknown as {reviewed: unknown}).reviewed !== null);
  const reviewed = await page.evaluate(() => (window as unknown as {reviewed: Record<string, unknown>}).reviewed);
  assert.equal(reviewed.intent, intent);
  assert.equal(reviewed.revision, revision);
  assert.equal(reviewed.authorizedIdeas, 2);
  assert.equal(reviewed.tasks, 2);
  assert.deepEqual(reviewed.decisions, [
    {idea: ids[0], decision: 'accept'},
    {idea: ids[1], decision: 'edit'},
  ]);
});

test('a changed objective saves a cleared new request freely despite an emptied explicit subset', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await resumeExplicitSubset(page, [0]);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await page.getByLabel('Desired outcome', {exact: true}).fill('Build a shift scheduler');
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  assert.deepEqual(saved.ideas, []);
  assert.deepEqual(saved.tasks, []);
  assert.equal('effective_outcomes' in saved, false);
  assert.equal(await page.getByText(scopeGateMessage, {exact: true}).count(), 0);
});

test('a terminal revision starts a new request freely despite an emptied explicit subset', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {revision} = await resumeExplicitSubset(page, [0], {state: 'declined', decline: [0]});
  await page
    .getByText('This direction was declined. Revise the outcome and explore again for a new draft.', {exact: true})
    .waitFor();
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a shift scheduler');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, revision);
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  const saved = await lastPostedBody(page, 'draft');
  assert.deepEqual(saved.ideas, []);
  assert.deepEqual(saved.tasks, []);
  assert.equal('effective_outcomes' in saved, false);
  assert.equal(await page.getByText(scopeGateMessage, {exact: true}).count(), 0);
});

// FJ-376 declined-task binding regressions: the posted proposal — not the
// fixture store — is proved against the canonical Go IntentRevision.Validate
// declined-outcome rules (internal/factory/intent.go checkEffectiveOutcomes
// + indexRevisionTasks). The committed suite asserts that shape structurally
// so it passes on a clean checkout with zero private paths; a repo-external
// validator binary may additionally prove an exact posted body against the
// real Go path via the run-dir-only probe at the end of this file.
function canonicalShapeViolation(body: Record<string, unknown>): string | null {
  const ideas = new Map(
    ((body.ideas as {id: string; decision: string}[]) || []).map((idea) => [idea.id, idea.decision])
  );
  for (const id of (body.effective_outcomes as string[]) || []) {
    const decision = ideas.get(id);
    if (decision === undefined || decision === 'decline' || decision === 'pending')
      return 'effective outcomes must be authorized choices';
  }
  for (const task of (body.tasks as {idea_ids: string[]}[]) || []) {
    for (const id of task.idea_ids || []) {
      if (!ideas.has(id)) return 'task references unknown product choice';
      if (ideas.get(id) === 'decline') return 'task references declined outcome';
    }
  }
  return null;
}

function checkCanonical(label: string, body: Record<string, unknown>): void {
  const violation = canonicalShapeViolation(body);
  assert.equal(violation, null, `canonical declined-outcome shape rejected ${label}: ${violation}`);
}

function expectCanonicalInvalid(label: string, body: Record<string, unknown>, pattern: RegExp): void {
  const violation = canonicalShapeViolation(body) || '';
  assert.match(violation, pattern, `pre-fix shape ${label} must violate the canonical declined-outcome rule`);
}

// Plain three-idea draft with no explicit subset: every idea carries linked
// work, mirroring the backend explore contract.
async function driveTaskLinkedDraft(page: Page): Promise<{intent: string; ids: string[]}> {
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await setExplorationMode(page, 'supported');
  await setActor(page, '1');
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a leave tracker');
  await page.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await updateProposal(page);
  for (const change of ['Track overtime too', 'Summarize nightly state']) {
    const before = await page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
    );
    await page.getByRole('button', {name: 'Change something', exact: true}).click();
    await page.locator('.change-field').getByLabel('What should change?', {exact: true}).fill(change);
    await page.getByRole('button', {name: 'Submit change', exact: true}).click();
    await page.waitForFunction((prev: string) => {
      const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
      return !host.busy && (host.revision?.id || '') !== prev;
    }, before);
  }
  assert.equal(await page.getByRole('combobox', {name: /^Choice:/}).count(), 3);
  const current = await page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown};
    return JSON.parse(JSON.stringify(host.revision)) as Record<string, unknown>;
  });
  const ids = (current.ideas as {id: string}[]).map((idea) => idea.id);
  const tasks = current.tasks as {idea_ids: string[]}[];
  assert.equal(ids.length, 3);
  const first = ids[0];
  assert.ok(first);
  assert.ok(
    tasks.some((task) => task.idea_ids.includes(first)),
    'idea A must carry linked work'
  );
  return {intent: String(current.intent), ids};
}

test('declining a task-linked idea drops its work, saves canonically valid state, and re-accept restores it', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  const fullTasks = await page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {revision: {tasks: unknown}};
    return JSON.parse(JSON.stringify(host.revision.tasks)) as {key: string; idea_ids: string[]}[];
  });
  const linked = fullTasks.filter((task) => task.idea_ids.includes(declinedA));
  assert.ok(linked.length > 0);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  const ideas = saved.ideas as {id: string; decision: string}[];
  assert.equal(ideas.find((idea) => idea.id === declinedA)?.decision, 'decline');
  const tasks = saved.tasks as {key: string; idea_ids: string[]}[];
  assert.equal(
    tasks.some((task) => task.idea_ids.includes(declinedA)),
    false,
    'the proposal must carry no work referencing declined A'
  );
  for (const task of fullTasks.filter((task) => !task.idea_ids.includes(declinedA)))
    assert.ok(
      tasks.some((kept) => kept.key === task.key),
      `unrelated work ${task.key} must survive the decline`
    );
  // Negative control: the pre-fix shape (declined idea, unchanged tasks)
  // violates the canonical declined-outcome rule.
  expectCanonicalInvalid('prefix-shape', {...saved, tasks: fullTasks}, /task references declined outcome/);
  // The shipped proposal carries the canonical shape; fixture storage
  // alone is not proof.
  checkCanonical('declined-task-drop', saved);
  // Real recovery path: the removed work names its declined idea with a
  // reachable re-decision.
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const restoredTasks = restored.tasks as {key: string; idea_ids: string[]}[];
  // The exact shelved bindings return (re-exploration regenerates the
  // remaining work under fresh keys, so only the declined-linked originals
  // are key-stable here).
  for (const task of linked)
    assert.ok(
      restoredTasks.some((kept) => kept.key === task.key),
      `shelved work ${task.key} must return on re-accept`
    );
  assert.equal(restoredTasks.length, fullTasks.length);
  checkCanonical('declined-task-restored', restored);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

test('partial narrowing with a declined task-linked idea drops its work and recovers by click', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {ids} = await resumeExplicitSubset(page, [0, 1]);
  const keptIdea = ids[0];
  const narrowedIdea = ids[1];
  assert.ok(keptIdea);
  assert.ok(narrowedIdea);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .nth(1)
    .selectOption('decline');
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  assert.deepEqual(saved.effective_outcomes, [keptIdea]);
  const tasks = saved.tasks as {key: string; idea_ids: string[]}[];
  assert.equal(
    tasks.some((task) => task.idea_ids.includes(narrowedIdea)),
    false,
    'the narrowed proposal must carry no work referencing declined idea 1'
  );
  assert.ok(
    tasks.some((task) => task.idea_ids.includes(keptIdea)),
    'remaining subset work must survive'
  );
  checkCanonical('subset-declined-task-drop', saved);
  await page.getByRole('button', {name: `Re-accept declined idea ${narrowedIdea}`, exact: true}).click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const restoredTasks = restored.tasks as {key: string; idea_ids: string[]}[];
  assert.ok(
    restoredTasks.some((task) => task.idea_ids.includes(narrowedIdea)),
    'subset work must return on re-accept'
  );
  checkCanonical('subset-declined-task-restored', restored);
});

test('a terminal seed with a declined idea carries no declined effective or task links', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {ids} = await resumeExplicitSubset(page, [0], {state: 'declined', decline: [0]});
  const terminalIdea = ids[0];
  assert.ok(terminalIdea);
  const loaded = await page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown};
    return JSON.parse(JSON.stringify(host.revision)) as Record<string, unknown>;
  });
  const ideas = loaded.ideas as {id: string; decision: string}[];
  const declined = new Set(ideas.filter((idea) => idea.decision === 'decline').map((idea) => idea.id));
  assert.ok(declined.has(terminalIdea));
  assert.equal(
    ((loaded.effective_outcomes as string[]) || []).some((id) => declined.has(id)),
    false
  );
  assert.equal(
    ((loaded.tasks as {idea_ids: string[]}[]) || []).some((task) => task.idea_ids.some((id) => declined.has(id))),
    false
  );
  checkCanonical('terminal-declined-seed', loaded);
  await page
    .getByText('This direction was declined. Revise the outcome and explore again for a new draft.', {exact: true})
    .waitFor();
});

// FJ-381 shelf/dependency recovery regressions: the released 381 run-dir
// evidence (decline/prune, Re-accept restore, save→reload,
// save→explore→reload, reset record) encoded as committed browser
// regressions. Same-browser localStorage scope only — no cross-device
// claim. Posted shapes mirror the structural canonical assertions above;
// no private binary, no absolute paths (380 boundary holds).
async function chainKeptOntoDeclined(page: Page): Promise<string | null> {
  return page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown; dirty: boolean};
    const rev = JSON.parse(JSON.stringify(host.revision)) as {
      tasks: {key: string; idea_ids: string[]; depends_on?: string[]}[];
    };
    const [a, b, c] = rev.tasks;
    if (!a || !b || !c) return null;
    b.depends_on = [a.key];
    host.revision = {...(host.revision as Record<string, unknown>), tasks: rev.tasks};
    host.dirty = true;
    return a.key;
  });
}

// A limits-only save with decisions untouched: revise() takes the saveCurrent
// path (no re-explore) and advances the draft base to a saved revision, so a
// same-browser reload resumes the exact head the shelf was persisted against.
// Mirrors the probe's limits-only save ahead of its reload legs.
async function saveUnchangedLimits(page: Page) {
  await page.evaluate(() => {
    (document.querySelector('soda-intent') as unknown as {dirty: boolean}).dirty = true;
  });
  await updateProposal(page);
}

async function draftShelfEntry(page: Page, intent: string): Promise<Record<string, unknown> | null> {
  return page.evaluate((key: string) => {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as Record<string, unknown>) : null;
  }, `soda-intent-draft:v1:1:${intent}`);
}

function shelfKeys(entry: Record<string, unknown> | null): string[] {
  return ((entry?.shelf as {key: string}[] | undefined) || []).map((task) => task.key);
}

test('declining chained work prunes the kept edge and re-accept restores it without invented edges', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  const otherIdea = ids[2];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  assert.ok(otherIdea);
  // Update proposal re-explores once decisions change, regenerating kept-task
  // keys: locate work by idea linkage, never by pre-save key.
  const aKey = await chainKeptOntoDeclined(page);
  assert.ok(aKey, 'the draft must carry three work items to chain');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  const tasks = saved.tasks as {key: string; idea_ids: string[]; depends_on?: string[]}[];
  assert.equal(
    tasks.some((task) => task.idea_ids.includes(declinedA)),
    false,
    'no saved work may reference declined A'
  );
  const keptB = tasks.find((task) => (task.idea_ids || []).includes(keptIdea));
  assert.ok(keptB, 'kept B work must survive the decline');
  assert.deepEqual(keptB.depends_on || [], [], 'the B→A edge must prune while A is declined');
  assert.ok(
    tasks.some((task) => (task.idea_ids || []).includes(otherIdea)),
    'unrelated C work must survive the decline'
  );
  checkCanonical('shelf-decline-pruned-kept-edge', saved);
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
  const entry = await draftShelfEntry(page, intent);
  const shelved = ((entry?.shelf as {key: string; idea_ids: string[]}[] | undefined) || []).filter((task) =>
    (task.idea_ids || []).includes(declinedA)
  );
  assert.ok(shelved.length > 0, 'the declined A work must persist in the same-browser shelf');
  const shelvedKeys = new Set(shelved.map((task) => task.key));
  const repairs = Object.values(
    (entry?.shelfDeps as Record<string, {deps: string[]; ideas: string[]}> | undefined) || {}
  );
  assert.ok(
    repairs.some(
      (repair) => (repair.ideas || []).includes(keptIdea) && (repair.deps || []).some((dep) => shelvedKeys.has(dep))
    ),
    'the cut B→A edge must persist as a repair record'
  );
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as {key: string; idea_ids: string[]; depends_on?: string[]}[];
  assert.ok(
    revived.some((task) => task.key === aKey),
    'shelved A work must return on re-accept under its shelved key'
  );
  const revivedB = revived.find((task) => (task.idea_ids || []).includes(keptIdea));
  assert.ok(revivedB, 'kept B work must still be proposed after re-accept');
  assert.deepEqual(revivedB.depends_on || [], [aKey], 're-accept must restore the recorded B→A edge');
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  checkCanonical('shelf-decline-restored-edge', restored);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

test('saved decline recovery survives a same-browser reload with dependencies intact', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const aKey = await chainKeptOntoDeclined(page);
  assert.ok(aKey, 'the draft must carry three work items to chain');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  checkCanonical('shelf-reload-decline-drop', await lastPostedBody(page, 'draft'));
  // Settle the shelf against a saved head before reloading: the decline
  // update re-explores, leaving the draft ahead of the stored revision.
  await saveUnchangedLimits(page);
  const before = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  const settled = await draftShelfEntry(page, intent);
  assert.equal(settled?.base, before, 'the limits-only save must persist the shelf at the saved head');
  await resetIntent(page);
  await resumeIntent(page, intent);
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
  const after = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  assert.equal(after, before, 'reload must resume the same head so the shelf still applies');
  const entry = await draftShelfEntry(page, intent);
  const shelved = ((entry?.shelf as {key: string; idea_ids: string[]}[] | undefined) || []).filter((task) =>
    (task.idea_ids || []).includes(declinedA)
  );
  assert.ok(shelved.length > 0, 'reload must re-offer the shelved A work');
  const shelvedKeys = new Set(shelved.map((task) => task.key));
  const repairs = Object.values(
    (entry?.shelfDeps as Record<string, {deps: string[]; ideas: string[]}> | undefined) || {}
  );
  assert.ok(
    repairs.some(
      (repair) => (repair.ideas || []).includes(keptIdea) && (repair.deps || []).some((dep) => shelvedKeys.has(dep))
    ),
    'reload must keep the recorded B→A repair'
  );
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as {key: string; idea_ids: string[]; depends_on?: string[]}[];
  assert.ok(
    revived.some((task) => task.key === aKey),
    'shelved A work must return after reload recovery'
  );
  const revivedB = revived.find((task) => (task.idea_ids || []).includes(keptIdea));
  assert.ok(revivedB, 'kept B work must still be proposed after reload recovery');
  assert.deepEqual(revivedB.depends_on || [], [aKey], 'reload recovery must restore the B→A edge intact');
  checkCanonical('shelf-reload-restored-edge', restored);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

test('save explore reload keeps the shelf at the advanced head', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const aKey = await chainKeptOntoDeclined(page);
  assert.ok(aKey, 'the draft must carry three work items to chain');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const declinedEntry = await draftShelfEntry(page, intent);
  assert.ok(shelfKeys(declinedEntry).length > 0, 'the decline save must persist a shelf');
  const headBefore = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page.locator('.change-field').getByLabel('What should change?', {exact: true}).fill('Add a weekly digest');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, headBefore);
  const headAfter = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  const exploreEntry = await draftShelfEntry(page, intent);
  assert.equal(exploreEntry?.base, headAfter, 'explore must re-persist the shelf at the advanced head');
  assert.ok(shelfKeys(exploreEntry).length > 0, 'explore must keep the shelf for the advanced head');
  // Settle the advanced shelf against a saved head before reloading.
  await saveUnchangedLimits(page);
  const settledHead = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  const settledEntry = await draftShelfEntry(page, intent);
  assert.equal(settledEntry?.base, settledHead, 'the limits-only save must persist the advanced shelf');
  assert.ok(shelfKeys(settledEntry).length > 0, 'the limits-only save must keep the advanced shelf');
  await resetIntent(page);
  await resumeIntent(page, intent);
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as {key: string; idea_ids: string[]}[];
  assert.ok(
    revived.some((task) => (task.idea_ids || []).includes(declinedA)),
    'shelved A work must return after explore reload recovery'
  );
  assert.ok(
    revived.some((task) => (task.idea_ids || []).includes(keptIdea)),
    'kept B direction must survive explore reload recovery'
  );
  checkCanonical('shelf-explore-reload-restored', restored);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

test('reset preserves the recovery record instead of clearing it', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  const keptIdea = ids[1];
  assert.ok(keptIdea);
  const aKey = await chainKeptOntoDeclined(page);
  assert.ok(aKey, 'the draft must carry three work items to chain');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  // Settle the shelf against a saved head so the reset record matches the
  // resume head exactly.
  await saveUnchangedLimits(page);
  const savedEntry = await draftShelfEntry(page, intent);
  const shelved = ((savedEntry?.shelf as {key: string; idea_ids: string[]}[] | undefined) || []).filter((task) =>
    (task.idea_ids || []).includes(declinedA)
  );
  assert.ok(shelved.length > 0, 'the decline save must persist the A work');
  const shelvedKeys = new Set(shelved.map((task) => task.key));
  const savedRepairs = Object.values(
    (savedEntry?.shelfDeps as Record<string, {deps: string[]; ideas: string[]}> | undefined) || {}
  );
  assert.ok(
    savedRepairs.some(
      (repair) => (repair.ideas || []).includes(keptIdea) && (repair.deps || []).some((dep) => shelvedKeys.has(dep))
    ),
    'the decline save must record the B→A repair'
  );
  await resetIntent(page);
  const resetEntry = await draftShelfEntry(page, intent);
  assert.ok(resetEntry, 'reset must keep the same-browser draft entry');
  assert.ok(
    shelfKeys(resetEntry).some((key) => shelvedKeys.has(key)),
    'reset must persist the recovery record before clearing'
  );
  const resetRepairs = Object.values(
    (resetEntry?.shelfDeps as Record<string, {deps: string[]; ideas: string[]}> | undefined) || {}
  );
  assert.ok(
    resetRepairs.some(
      (repair) => (repair.ideas || []).includes(keptIdea) && (repair.deps || []).some((dep) => shelvedKeys.has(dep))
    ),
    'reset must keep the recorded repair instead of overwriting it'
  );
  await resumeIntent(page, intent);
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
});

// Run-dir-only evidence probe for the real Go IntentRevision.Validate path:
// configured solely via SODA_CANONICAL_VALIDATOR (a validator executable)
// with evidence landing under SODA_CANONICAL_EVIDENCE_DIR or the OS temp
// dir — never the repo, never an absolute private path in this file. Skips
// explicitly when unconfigured; the structural assertions above stay the
// committed proof.
test('canonical validator probe proves a declined-task drop against the real Go path', async (t) => {
  const validator = process.env.SODA_CANONICAL_VALIDATOR || '';
  if (validator === '' || !existsSync(validator)) {
    t.skip(
      'SODA_CANONICAL_VALIDATOR names no validator binary; structural canonical-shape assertions remain the committed proof'
    );
    return;
  }
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  checkCanonical('probe-declined-task-drop', saved);
  const evidenceDir = process.env.SODA_CANONICAL_EVIDENCE_DIR || tmpdir();
  mkdirSync(evidenceDir, {recursive: true});
  const file = join(evidenceDir, 'fux-380-canonical-declined-task-drop.json');
  writeFileSync(file, JSON.stringify(saved));
  let output = '';
  try {
    output = execFileSync(validator, [file], {encoding: 'utf8'});
  } catch (error) {
    const stdout = (error as {stdout?: string}).stdout || String(error);
    assert.fail(`canonical Go IntentRevision.Validate rejected probe-declined-task-drop: ${stdout}`);
  }
  assert.match(output, /valid/);
});

// FJ-395 stabilization regressions (released 389/390/396/397 shelf behavior):
// committed proof for the joint pass, driven through the existing
// workspace-driver fixture and the test-local intent peer above — no private
// paths, no fixture changes, canonically valid posted shapes throughout. The
// backend re-mints task keys on every save/update, so every dependency edge
// below is compared in idea space (the stable linkage) against the posted
// body, the live revision and the persisted same-browser shelf; only the
// idea linkage is stable across key generations.
type PostedTask = {key: string; idea_ids: string[]; depends_on?: string[]};

// Chain kept B onto A and C: locate work by idea linkage, never by pre-save
// key, since the backend re-mints keys on updates.
async function chainBOntoAC(page: Page, ids: string[]): Promise<{aKey: string; bKey: string; cKey: string} | null> {
  return page.evaluate((wanted: string[]) => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown; dirty: boolean};
    const rev = JSON.parse(JSON.stringify(host.revision)) as {
      tasks: {key: string; idea_ids: string[]; depends_on?: string[]}[];
    };
    const taskFor = (id: string) => rev.tasks.find((task) => task.idea_ids.includes(id));
    const a = taskFor(wanted[0] || '');
    const b = taskFor(wanted[1] || '');
    const c = taskFor(wanted[2] || '');
    if (!a || !b || !c || a.key === b.key || b.key === c.key || a.key === c.key) return null;
    b.depends_on = [a.key, c.key];
    host.revision = {...(host.revision as Record<string, unknown>), tasks: rev.tasks};
    host.dirty = true;
    return {aKey: a.key, bKey: b.key, cKey: c.key};
  }, ids);
}

// Chain A onto B: the 396/397 direction, where restored work carries the edge.
async function chainAOntoB(page: Page, ids: string[]): Promise<{aKey: string; bKey: string} | null> {
  return page.evaluate((wanted: string[]) => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown; dirty: boolean};
    const rev = JSON.parse(JSON.stringify(host.revision)) as {
      tasks: {key: string; idea_ids: string[]; depends_on?: string[]}[];
    };
    const taskFor = (id: string) => rev.tasks.find((task) => task.idea_ids.includes(id));
    const a = taskFor(wanted[0] || '');
    const b = taskFor(wanted[1] || '');
    if (!a || !b || a.key === b.key) return null;
    a.depends_on = [b.key];
    host.revision = {...(host.revision as Record<string, unknown>), tasks: rev.tasks};
    host.dirty = true;
    return {aKey: a.key, bKey: b.key};
  }, ids);
}

async function revisionTasks(page: Page): Promise<PostedTask[]> {
  const revision = await page.evaluate(
    () =>
      JSON.parse(
        JSON.stringify((document.querySelector('soda-intent') as unknown as {revision: unknown}).revision)
      ) as {
        tasks: PostedTask[];
      }
  );
  return revision.tasks;
}

// Decline by current description: card order and keys shift across backend
// re-exploration, while descriptions stay stable.
async function declineIdeaByDescription(page: Page, id: string) {
  const descs = await page.evaluate(() =>
    (
      (
        document.querySelector('soda-intent') as unknown as {
          revision?: {ideas?: {id: string; description: string}[]};
        }
      ).revision?.ideas || []
    ).map((idea) => ({id: idea.id, description: idea.description}))
  );
  const desc = descs.find((idea) => idea.id === id)?.description || id;
  await page
    .locator('article', {hasText: desc.slice(0, 40)})
    .getByRole('combobox')
    .selectOption('decline');
}

function taskForIdea(tasks: PostedTask[], idea: string): PostedTask | undefined {
  return tasks.find((task) => task.idea_ids.includes(idea));
}

// Resolve dependency keys to the sorted idea sets of their owners, drawing
// on both the posted tasks and the shelved entries: keys are re-minted by
// the backend on saves, so edges are compared in idea space, never by key.
function edgeIdeaSets(
  deps: string[] | undefined,
  tasks: PostedTask[],
  shelf: {key: string; idea_ids: string[]}[]
): string[][] {
  return (deps || []).map((dep) => {
    const owner = tasks.find((task) => task.key === dep) || shelf.find((task) => task.key === dep);
    return [...(owner?.idea_ids || [])].sort();
  });
}

function shelfOf(entry: Record<string, unknown> | null): {key: string; idea_ids: string[]; depends_on?: string[]}[] {
  return (entry?.shelf as {key: string; idea_ids: string[]; depends_on?: string[]}[] | undefined) || [];
}

function repairOwnedBy(
  entry: Record<string, unknown> | null,
  idea: string
): {key: string; repair: {deps: string[]; ideas: string[]; depIdeas?: Record<string, string[]>}} | null {
  const all =
    (entry?.shelfDeps as
      | Record<string, {deps: string[]; ideas: string[]; depIdeas?: Record<string, string[]>}>
      | undefined) || {};
  const hit = Object.entries(all).find(([, repair]) => repair.ideas.includes(idea));
  return hit ? {key: hit[0], repair: hit[1]} : null;
}

test('repeated declines accumulate the cut edges and each re-accept restores only its own', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB, ideaC] = ids;
  assert.ok(ideaA && ideaB && ideaC);
  const chained = await chainBOntoAC(page, ids);
  assert.ok(chained, 'the draft must chain B onto A and C');
  // Decline A + update: B→A prunes, B→C stays, both originals recorded.
  await declineIdeaByDescription(page, ideaA);
  await updateProposal(page);
  let saved = await lastPostedBody(page, 'draft');
  checkCanonical('repeat-decline-drop-A', saved);
  let tasks = saved.tasks as PostedTask[];
  let entry = await draftShelfEntry(page, intent);
  let shelf = shelfOf(entry);
  let live = await revisionTasks(page);
  let keptB = taskForIdea(tasks, ideaB);
  assert.ok(keptB, 'kept B work must survive the first decline');
  assert.deepEqual(
    edgeIdeaSets(keptB.depends_on, [...tasks, ...live], shelf),
    [[ideaC]],
    'first update prunes B→A and keeps B→C'
  );
  let repairB = repairOwnedBy(entry, ideaB);
  assert.ok(repairB, 'the first update must record the cut edges');
  assert.deepEqual(
    [...new Set(edgeIdeaSets(repairB.repair.deps, [...tasks, ...live], shelf).flat())].sort(),
    [ideaA, ideaC].sort(),
    'the first record holds the exact original edges'
  );
  // Decline C + update: B fully sheltered, the record must still hold BOTH.
  await declineIdeaByDescription(page, ideaC);
  await updateProposal(page);
  saved = await lastPostedBody(page, 'draft');
  checkCanonical('repeat-decline-drop-AC', saved);
  tasks = saved.tasks as PostedTask[];
  entry = await draftShelfEntry(page, intent);
  shelf = shelfOf(entry);
  live = await revisionTasks(page);
  keptB = taskForIdea(tasks, ideaB);
  assert.ok(keptB, 'kept B work must survive the second decline');
  assert.deepEqual(keptB.depends_on || [], [], 'B edges stay fully sheltered while A and C are declined');
  repairB = repairOwnedBy(entry, ideaB);
  assert.ok(repairB, 'the second update must keep a repair record');
  assert.deepEqual(
    [...new Set(edgeIdeaSets(repairB.repair.deps, [...tasks, ...live], shelf).flat())].sort(),
    [ideaA, ideaC].sort(),
    'the second update must accumulate, never overwrite, the original edges'
  );
  // Re-accept A while C stays declined: B→A returns, B→C stays cut.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaA}`, exact: true}).click();
  await updateProposal(page);
  saved = await lastPostedBody(page, 'draft');
  checkCanonical('repeat-decline-reaccept-A', saved);
  tasks = saved.tasks as PostedTask[];
  entry = await draftShelfEntry(page, intent);
  shelf = shelfOf(entry);
  live = await revisionTasks(page);
  assert.ok(
    tasks.some((task) => task.idea_ids.includes(ideaA)),
    'shelved A work must return on re-accept'
  );
  assert.equal(
    tasks.some((task) => task.idea_ids.includes(ideaC)),
    false,
    'C work stays out while declined'
  );
  keptB = taskForIdea(tasks, ideaB);
  assert.ok(keptB, 'kept B work must still be proposed after re-accepting A');
  assert.deepEqual(
    edgeIdeaSets(keptB.depends_on, [...tasks, ...live], shelf),
    [[ideaA]],
    're-accept of A restores B→A only'
  );
  // Re-accept C: the full original edge set returns, nothing invented.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaC}`, exact: true}).click();
  await updateProposal(page);
  saved = await lastPostedBody(page, 'draft');
  checkCanonical('repeat-decline-reaccept-AC', saved);
  tasks = saved.tasks as PostedTask[];
  entry = await draftShelfEntry(page, intent);
  shelf = shelfOf(entry);
  live = await revisionTasks(page);
  keptB = taskForIdea(tasks, ideaB);
  assert.ok(keptB, 'kept B work must still be proposed after re-accepting C');
  assert.deepEqual(
    edgeIdeaSets(keptB.depends_on, [...tasks, ...live], shelf)
      .map((s) => s.join(','))
      .sort(),
    [[ideaA], [ideaC]].map((s) => s.join(',')).sort(),
    're-accept of C restores the full original edge set'
  );
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

test('recordPrunedDeps merges a later decline against the exact originals instead of overwriting', async (t) => {
  // The live backend re-mints task keys on every save, so the overwrite shape
  // (a LATER save reading already-pruned CURRENT dependencies) only arises
  // against stable keys. This drives the shipped recordPrunedDeps method
  // itself through that exact faithful sequence: B→[A,C] → B→[C] → B→[].
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  await driveTaskLinkedDraft(page);
  const d = (n: number) => 'd' + n.toString(16).padStart(24, '0');
  const bKey = d(501);
  const aKey = d(502);
  const cKey = d(503);
  const bIdea = d(511);
  const result = await page.evaluate(
    ([bKey, aKey, cKey, bIdea]: string[]) => {
      // The arrangement always passes exactly four non-empty keys; narrow
      // once at the source so the task fields below stay honest strings.
      if (!bKey || !aKey || !cKey || !bIdea) throw Error('test keys must be non-empty strings');
      const host = document.querySelector('soda-intent') as unknown as {
        revision: Record<string, unknown>;
        shelvedTasks: {key: string; idea_ids: string[]; depends_on?: string[]; objective: string; outcomes: string[]}[];
        shelvedPrunedDeps: Record<string, {deps: string[]; ideas: string[]}>;
        recordPrunedDeps(next: unknown, restored: Map<string, unknown>): void;
      };
      // Keep the live revision shape intact (Lit renders from it) and swap
      // only the task list between the two faithful save states.
      const base = host.revision;
      const taskB = (deps: string[]) => ({
        key: bKey,
        idea_ids: [bIdea],
        objective: 'b',
        outcomes: ['o'],
        depends_on: deps,
      });
      const taskA = () => ({key: aKey, idea_ids: ['idea-a'], objective: 'a', outcomes: ['o']});
      const taskC = () => ({key: cKey, idea_ids: ['idea-c'], objective: 'c', outcomes: ['o']});
      // First decline save: B→[A,C] pruned to B→[C], A shelved.
      host.revision = {...base, tasks: [taskB([aKey, cKey])]};
      host.shelvedTasks = [taskA()];
      host.shelvedPrunedDeps = {};
      host.recordPrunedDeps({ideas: [], tasks: [taskB([cKey])]}, new Map());
      const afterFirst = JSON.parse(JSON.stringify(host.shelvedPrunedDeps));
      // Second decline save: B→[C] pruned to B→[], C newly shelved. The prior
      // revision already carries the pruned B→[C] here.
      host.revision = {...base, tasks: [taskB([cKey])]};
      host.shelvedTasks = [taskA(), taskC()];
      host.recordPrunedDeps({ideas: [], tasks: [taskB([])]}, new Map());
      const afterSecond = JSON.parse(JSON.stringify(host.shelvedPrunedDeps));
      host.revision = base;
      return {afterFirst, afterSecond};
    },
    [bKey, aKey, cKey, bIdea]
  );
  assert.deepEqual(result.afterFirst[bKey]?.deps, [aKey, cKey], 'first save records the exact original edges');
  assert.deepEqual(
    result.afterSecond[bKey]?.deps,
    [aKey, cKey],
    'second save merges against the exact originals instead of overwriting with the pruned [C]'
  );
});

test('shelf validation keeps a 26+ link task yet still rejects an over-length link', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent} = await driveTaskLinkedDraft(page);
  await page.locator('article').first().getByRole('combobox').selectOption('decline');
  await updateProposal(page);
  checkCanonical('wide-shelf-decline-drop', await lastPostedBody(page, 'draft'));
  const key = `soda-intent-draft:v1:1:${intent}`;
  // Widen one shelved task to 30 dependency links: the canonical task shape
  // allows up to 64 entries of up to 25 chars each, so the count side of
  // reload validation must keep it (a single shared 25 limit would discard
  // the entire shelf here).
  const wide = await page.evaluate(
    ([storageKey, count]: [string, number]) => {
      const raw = localStorage.getItem(storageKey);
      const entry = raw ? (JSON.parse(raw) as {shelf?: {key: string; depends_on?: string[]}[]}) : null;
      if (!entry || !Array.isArray(entry.shelf) || entry.shelf.length === 0) return null;
      const links = Array.from({length: count}, (_, i) => 'd' + (100 + i).toString(16).padStart(24, '0'));
      entry.shelf[0] = {...entry.shelf[0], depends_on: links} as {key: string; depends_on?: string[]};
      localStorage.setItem(storageKey, JSON.stringify(entry));
      return {shelfTask: entry.shelf[0]?.key || '', linkCount: links.length};
    },
    [key, 30] as [string, number]
  );
  assert.ok(wide, 'the decline save must persist a shelf to widen');
  assert.equal(wide.linkCount, 30);
  // Simulate the reload leg through the real restore path (reset+resume in
  // this environment loads a different backend head, so invoke the shipped
  // restoreDraft validation directly after clearing the in-memory shelf).
  const clearAndRestore = () =>
    page.evaluate(() => {
      const host = document.querySelector('soda-intent') as unknown as {
        shelvedTasks: unknown[];
        shelvedPrunedDeps: unknown;
        restoreDraft(): void;
      };
      host.shelvedTasks = [];
      host.shelvedPrunedDeps = {};
      host.restoreDraft();
    });
  const shelvedCount = () =>
    page.evaluate(
      () => (document.querySelector('soda-intent') as unknown as {shelvedTasks?: unknown[]}).shelvedTasks?.length || 0
    );
  await clearAndRestore();
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  assert.ok(await shelvedCount(), 'reload must keep the widened 30-link shelf task instead of discarding the shelf');
  // The length side stays strict: a single 26-char link exceeds the 25-char
  // item limit, so the entry fails validation and the shelf is dropped —
  // count and item length are separate limits.
  await page.evaluate((storageKey: string) => {
    const raw = localStorage.getItem(storageKey);
    const entry = raw ? (JSON.parse(raw) as {shelf?: {key: string; depends_on?: string[]}[]}) : null;
    if (!entry || !Array.isArray(entry.shelf) || entry.shelf.length === 0) throw Error('missing shelf to narrow');
    entry.shelf[0] = {...entry.shelf[0], depends_on: ['d' + '0'.repeat(25)]} as {key: string; depends_on?: string[]};
    localStorage.setItem(storageKey, JSON.stringify(entry));
  }, key);
  await clearAndRestore();
  assert.equal(await shelvedCount(), 0, 'reload must drop a shelf entry carrying an over-length link');
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

test('cross-explore remaps a shelved edge onto the regenerated successor key', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB] = ids;
  assert.ok(ideaA && ideaB);
  const chained = await chainAOntoB(page, ids);
  assert.ok(chained, 'the draft must chain A onto B');
  // Decline A + update: A shelved with its B edge; lineage to live B pinned.
  await declineIdeaByDescription(page, ideaA);
  await updateProposal(page);
  let saved = await lastPostedBody(page, 'draft');
  checkCanonical('cross-explore-decline-A', saved);
  let entry = await draftShelfEntry(page, intent);
  // The update re-keys kept work and the reconcile consumes pinned lineage
  // immediately, so the proof is the remapped state: the shelved A edge
  // already points at live B under a fresh key, never at the stale one.
  let shelf = shelfOf(entry);
  let live = await revisionTasks(page);
  let shelvedA = shelf.find((task) => task.idea_ids.includes(ideaA));
  assert.ok(shelvedA, 'shelved A work must persist with its edge');
  assert.deepEqual(
    edgeIdeaSets(shelvedA.depends_on, live, shelf),
    [[ideaB]],
    'the update must remap the edge onto live B'
  );
  const freshB = live.find((task) => task.idea_ids.includes(ideaB));
  assert.ok(freshB && (shelvedA.depends_on || [])[0] === freshB.key, 'the remapped edge must track the current B key');
  // Update re-explores and regenerates kept work under fresh keys.
  const before = await revisionTasks(page);
  const oldB = taskForIdea(before, ideaB);
  assert.ok(oldB);
  const headBefore = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page.locator('.change-field').getByLabel('What should change?', {exact: true}).fill('Add a weekly digest');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, headBefore);
  const afterExplore = await revisionTasks(page);
  const newB = taskForIdea(afterExplore, ideaB);
  assert.ok(newB, 'explore must regenerate the kept B work');
  entry = await draftShelfEntry(page, intent);
  shelf = shelfOf(entry);
  shelvedA = shelf.find((task) => task.idea_ids.includes(ideaA));
  assert.ok(shelvedA, 'shelved A work must survive explore');
  assert.deepEqual(
    edgeIdeaSets(shelvedA.depends_on, afterExplore, shelf),
    [[ideaB]],
    'reconcile must rewrite the shelved edge onto the successor key'
  );
  assert.notEqual(shelvedA.depends_on?.[0], oldB.key, 'the rewritten edge must not point at the stale key');
  // Re-accept A + update: the edge restores onto the successor, nothing invented.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaA}`, exact: true}).click();
  await updateProposal(page);
  saved = await lastPostedBody(page, 'draft');
  checkCanonical('cross-explore-reaccept-A', saved);
  const tasks = saved.tasks as PostedTask[];
  const fresh = shelfOf(await draftShelfEntry(page, intent));
  const revivedA = taskForIdea(tasks, ideaA);
  assert.ok(revivedA, 'shelved A work must return on re-accept');
  assert.deepEqual(edgeIdeaSets(revivedA.depends_on, tasks, fresh), [[ideaB]], 're-accept restores A→successor-B');
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
});

test('ambiguous regenerated lineage keeps the edge cut with recovery intact', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB] = ids;
  assert.ok(ideaA && ideaB);
  const chained = await chainAOntoB(page, ids);
  assert.ok(chained, 'the draft must chain A onto B');
  await declineIdeaByDescription(page, ideaA);
  await updateProposal(page);
  checkCanonical('ambiguous-decline-A', await lastPostedBody(page, 'draft'));
  // Settle the shelf against a saved head: the limits-only save re-pins the
  // stable lineage for the shelved edge, so the framed reconcile below hits
  // the true ambiguous branch (recorded target gone, two successors) rather
  // than a pass-through with no lineage to remap.
  await saveUnchangedLimits(page);
  const pinned = await page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {
      shelvedPrunedDeps: Record<string, {deps: string[]; ideas: string[]; depIdeas?: Record<string, string[]>}>;
    };
    return JSON.parse(JSON.stringify(host.shelvedPrunedDeps)) as typeof host.shelvedPrunedDeps;
  });
  const pinnedRepair = Object.values(pinned).find((repair) => repair.ideas.includes(ideaA));
  assert.ok(
    pinnedRepair && Object.keys(pinnedRepair.depIdeas || {}).length > 0,
    'the save must pin lineage for the shelved A edge before framing ambiguity'
  );
  // Drive the shipped reconcile against a regenerated proposal where the
  // recorded target's lineage matches TWO successors: the edge must stay
  // cut (never invented onto either twin) with the pending lineage kept for
  // a later regeneration.
  const framed = await page.evaluate((idea: string) => {
    const host = document.querySelector('soda-intent') as unknown as {
      revision: {tasks: {key: string; idea_ids: string[]; depends_on?: string[]}[]; ideas: {id: string}[]};
      shelvedTasks: {key: string; idea_ids: string[]; depends_on?: string[]}[];
      shelvedPrunedDeps: Record<string, {deps: string[]; ideas: string[]; depIdeas?: Record<string, string[]>}>;
      reconcileShelvedTasks(proposal: unknown): void;
    };
    const liveB = host.revision.tasks.find((task) => task.idea_ids.includes(idea));
    if (!liveB) return null;
    const before = JSON.parse(JSON.stringify(host.shelvedPrunedDeps)) as typeof host.shelvedPrunedDeps;
    const twinA = 'd' + 'e'.repeat(22) + 'a1';
    const twinB = 'd' + 'e'.repeat(22) + 'b2';
    const proposal = {
      ...(JSON.parse(JSON.stringify(host.revision)) as Record<string, unknown>),
      tasks: [
        ...host.revision.tasks.filter((task) => task.key !== liveB.key),
        {...liveB, key: twinA},
        {...liveB, key: twinB},
      ],
    };
    host.reconcileShelvedTasks(proposal);
    const after = JSON.parse(JSON.stringify(host.shelvedPrunedDeps)) as typeof host.shelvedPrunedDeps;
    const shelf = JSON.parse(JSON.stringify(host.shelvedTasks)) as typeof host.shelvedTasks;
    return {twinA, twinB, before, after, shelf};
  }, ideaB);
  assert.ok(framed, 'the kept B work must be live to frame the ambiguity');
  const repairBefore = Object.values(framed.before).find((repair) => repair.ideas.includes(ideaA));
  const repairAfter = Object.values(framed.after).find((repair) => repair.ideas.includes(ideaA));
  assert.ok(repairBefore && repairAfter, 'the shelved A edge must keep its repair record');
  assert.deepEqual(
    [...repairAfter.deps].sort(),
    [...repairBefore.deps].sort(),
    'ambiguous lineage must not remap the edge onto either successor'
  );
  assert.equal(repairAfter.deps.includes(framed.twinA), false, 'no invented edge onto the first twin');
  assert.equal(repairAfter.deps.includes(framed.twinB), false, 'no invented edge onto the second twin');
  assert.ok(
    Object.keys(repairAfter.depIdeas || {}).length > 0,
    'the pending lineage must survive for a later regeneration'
  );
  const shelfDeps = framed.shelf.flatMap((task) => task.depends_on || []);
  assert.equal(shelfDeps.includes(framed.twinA), false, 'the shelf must not point at the first twin');
  assert.equal(shelfDeps.includes(framed.twinB), false, 'the shelf must not point at the second twin');
  // Recoverable review: the declined work is still offered for re-accept.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaA}`, exact: true}).waitFor();
  // And recovery still functions afterwards: re-accepting restores the work
  // with every edge resolving to live work.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaA}`, exact: true}).click();
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  checkCanonical('ambiguous-reaccept-A', saved);
  const tasks = saved.tasks as PostedTask[];
  const shelf = shelfOf(await draftShelfEntry(page, intent));
  assert.ok(
    tasks.some((task) => task.idea_ids.includes(ideaA)),
    'shelved A work must return on re-accept'
  );
  const revivedA = taskForIdea(tasks, ideaA);
  assert.deepEqual(
    edgeIdeaSets(revivedA?.depends_on, tasks, shelf),
    [[ideaB]],
    're-accept restores A→B onto live work'
  );
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
});

test('staggered re-accept records the restored edge and recovers it on the later return', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB] = ids;
  assert.ok(ideaA && ideaB);
  const chained = await chainAOntoB(page, ids);
  assert.ok(chained, 'the draft must chain A onto B');
  // Decline A and B, then save ONCE: both shelve, no record can exist yet
  // (the capture only pins lineage to non-shelved targets).
  await declineIdeaByDescription(page, ideaA);
  await declineIdeaByDescription(page, ideaB);
  await updateProposal(page);
  let saved = await lastPostedBody(page, 'draft');
  checkCanonical('staggered-decline-AB', saved);
  let entry = await draftShelfEntry(page, intent);
  let shelf = shelfOf(entry);
  assert.deepEqual(
    [...new Set(shelf.flatMap((task) => task.idea_ids))].sort(),
    [ideaA, ideaB].sort(),
    'both declined tasks must shelve in the single save'
  );
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  // Re-accept A alone + update: A returns without its B edge (correct
  // prune), but the cut edge is recorded from the restored shape.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaA}`, exact: true}).click();
  await updateProposal(page);
  saved = await lastPostedBody(page, 'draft');
  checkCanonical('staggered-reaccept-A', saved);
  let tasks = saved.tasks as PostedTask[];
  entry = await draftShelfEntry(page, intent);
  shelf = shelfOf(entry);
  const loneA = taskForIdea(tasks, ideaA);
  assert.ok(loneA, 'shelved A work must return on re-accept');
  assert.deepEqual(edgeIdeaSets(loneA.depends_on, tasks, shelf), [], 'A returns edgeless while B is still declined');
  assert.equal(
    tasks.some((task) => task.idea_ids.includes(ideaB)),
    false,
    'B work stays out while declined'
  );
  const recordA = repairOwnedBy(entry, ideaA);
  assert.ok(recordA, 'the staggered update must record the restored A edge from the restored shape');
  assert.deepEqual(
    [...new Set(edgeIdeaSets(recordA.repair.deps, tasks, shelf).flat())].sort(),
    [ideaB].sort(),
    'the record must hold the exact restored edge, not the absent revision'
  );
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  // Re-accept B + update: the recorded edge restores A→B, nothing invented.
  await page.getByRole('button', {name: `Re-accept declined idea ${ideaB}`, exact: true}).click();
  await updateProposal(page);
  saved = await lastPostedBody(page, 'draft');
  checkCanonical('staggered-reaccept-AB', saved);
  tasks = saved.tasks as PostedTask[];
  entry = await draftShelfEntry(page, intent);
  shelf = shelfOf(entry);
  const finalA = taskForIdea(tasks, ideaA);
  assert.ok(finalA, 'A work must still be proposed after re-accepting B');
  assert.deepEqual(
    edgeIdeaSets(finalA.depends_on, tasks, shelf),
    [[ideaB]],
    're-accept of B restores the recorded A→B edge'
  );
  assert.ok(
    tasks.some((task) => task.idea_ids.includes(ideaB)),
    'shelved B work must return on re-accept'
  );
  const live = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

// FJ-423 unresolved-repair committed regressions (released 417/420/421
// run-dir evidence encoded with existing fixtures only): zero-match
// retention, unique-not-adoptable then live-target adoption, unresolved
// save→reload (incl. limit-only), and repeated A→A2→A3 lineage carry.
// Same-browser localStorage scope only; every posted shape passes the
// structural canonical assertions; no private paths, no fixture changes.
// Edges are compared in idea space (backend re-mints keys on saves).
async function armUnresolvedSplit(page: Page, idea: string) {
  await page.evaluate((splitIdea: string) => {
    const w = window as unknown as {__splitIdea?: string; fetch: typeof fetch};
    w.__splitIdea = splitIdea;
    if ((w.fetch as unknown as {__splitWrapped?: boolean}).__splitWrapped) return;
    const nativeFetch = w.fetch;
    const inner = nativeFetch.bind(window);
    const wrapped = async (input: RequestInfo | URL, init?: RequestInit) => {
      const response = await inner(input, init);
      const url = String(input);
      const target = (window as unknown as {__splitIdea?: string}).__splitIdea;
      if (!target || !url.endsWith('/explore') || (init?.method || 'GET') !== 'POST' || !response.ok) return response;
      const json = (await response.clone().json()) as {
        tasks: {key: string; idea_ids: string[]}[];
      };
      let serial = 0;
      for (const task of json.tasks || []) {
        for (const id of [task.key, ...(task.idea_ids || [])]) {
          const n = Number.parseInt(String(id).slice(1), 16);
          if (Number.isSafeInteger(n) && n > serial) serial = n;
        }
      }
      const fresh = () => 'd' + (serial += 0x100000).toString(16).padStart(24, '0');
      const tasks: unknown[] = [];
      for (const task of json.tasks || []) {
        tasks.push(task);
        if ((task.idea_ids || []).length === 1 && task.idea_ids[0] === target) tasks.push({...task, key: fresh()});
      }
      return Response.json({...json, tasks});
    };
    (wrapped as unknown as {__splitWrapped?: boolean}).__splitWrapped = true;
    // Keep the mock a full typeof fetch: nothing under test calls
    // preconnect, so forward it to the native implementation it wraps.
    w.fetch = Object.assign(wrapped, {
      preconnect: (...args: Parameters<typeof fetch.preconnect>) => {
        nativeFetch.preconnect(...args);
      },
    });
  }, idea);
}

async function disarmUnresolvedSplit(page: Page) {
  await page.evaluate(() => {
    delete (window as unknown as {__splitIdea?: string}).__splitIdea;
  });
}

async function unresolvedGateCount(page: Page): Promise<number> {
  return page.getByRole('group', {name: 'Unresolved dependency repair'}).count();
}

// Chain kept B onto declined-to-be A by idea linkage, decline A, then a
// split-armed explore leaves the B→A repair retained under the old owner
// key with no unique successor: the shared setup for the 423 legs.
async function driveUnresolvedAmbiguity(page: Page): Promise<{intent: string; declinedA: string; keptIdea: string}> {
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA && keptIdea);
  const chained = await page.evaluate(
    (wanted: string[]) => {
      const host = document.querySelector('soda-intent') as unknown as {revision: unknown; dirty: boolean};
      const rev = JSON.parse(JSON.stringify(host.revision)) as {
        tasks: {key: string; idea_ids: string[]; depends_on?: string[]}[];
      };
      const taskFor = (id: string) => rev.tasks.find((task) => task.idea_ids.includes(id));
      const a = taskFor(wanted[0] || '');
      const b = taskFor(wanted[1] || '');
      if (!a || !b || a.key === b.key) return null;
      b.depends_on = [a.key];
      host.revision = {...(host.revision as Record<string, unknown>), tasks: rev.tasks};
      host.dirty = true;
      return {aKey: a.key, bKey: b.key};
    },
    [declinedA, keptIdea]
  );
  assert.ok(chained, 'the draft must chain kept B onto declined-to-be A');
  await declineIdeaByDescription(page, declinedA);
  await updateProposal(page);
  checkCanonical('unresolved-decline-A', await lastPostedBody(page, 'draft'));
  await armUnresolvedSplit(page, keptIdea);
  const headBefore = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page.locator('.change-field').getByLabel('What should change?', {exact: true}).fill('Split the tracking work');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, headBefore);
  const afterExplore = await revisionTasks(page);
  const twins = afterExplore.filter((task) => task.idea_ids.length === 1 && task.idea_ids[0] === keptIdea);
  assert.equal(twins.length, 2, 'the armed explore must propose two same-idea B tasks');
  const entry = await draftShelfEntry(page, intent);
  const retained = repairOwnedBy(entry, keptIdea);
  assert.ok(retained, 'the unassigned repair must be retained, not dropped silently');
  assert.ok(retained.repair.deps.length > 0, 'the retained repair must still carry the cut edge');
  return {intent, declinedA, keptIdea};
}

test('unresolved repair survives a limits-only save and reload with review visible', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, declinedA, keptIdea} = await driveUnresolvedAmbiguity(page);
  await page.getByRole('group', {name: 'Unresolved dependency repair'}).waitFor();
  const before = await draftShelfEntry(page, intent);
  const retainedBefore = repairOwnedBy(before, keptIdea);
  assert.ok(retainedBefore, 'ambiguity setup must retain the repair under the old owner key');
  await saveUnchangedLimits(page);
  checkCanonical('unresolved-limits-save', await lastPostedBody(page, 'draft'));
  const settled = await draftShelfEntry(page, intent);
  const retained = repairOwnedBy(settled, keptIdea);
  assert.ok(retained, 'the limit-only save must carry the unassigned repair, not erase it');
  assert.deepEqual(
    [...retained.repair.deps].sort(),
    [...retainedBefore.repair.deps].sort(),
    'the carried repair must keep its exact recorded edges'
  );
  assert.ok(
    Object.keys((retained as unknown as {repair: {depIdeas?: Record<string, string[]>}}).repair.depIdeas || {}).length >
      0,
    'the carried repair must keep its pending lineage for a later regeneration'
  );
  assert.ok((await unresolvedGateCount(page)) > 0, 'ambiguity stays visible after the save until resolved');
  await resetIntent(page);
  await resumeIntent(page, intent);
  await page.getByRole('group', {name: 'Unresolved dependency repair'}).waitFor();
  const reopened = await draftShelfEntry(page, intent);
  const retainedAfter = repairOwnedBy(reopened, keptIdea);
  assert.ok(retainedAfter, 'reload must keep the carried unassigned repair');
  assert.deepEqual(
    [...retainedAfter.repair.deps].sort(),
    [...retainedBefore.repair.deps].sort(),
    'reload must keep the exact recorded edges'
  );
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
});

test('zero-match successor keeps the carried repair through save and reload', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, declinedA, keptIdea} = await driveUnresolvedAmbiguity(page);
  await declineIdeaByDescription(page, keptIdea);
  await updateProposal(page);
  checkCanonical('unresolved-zero-decline', await lastPostedBody(page, 'draft'));
  let entry = await draftShelfEntry(page, intent);
  assert.ok(repairOwnedBy(entry, keptIdea), 'zero successors must retain, not erase, the carried repair');
  await page.getByRole('group', {name: 'Unresolved dependency repair'}).waitFor();
  await saveUnchangedLimits(page);
  checkCanonical('unresolved-zero-save', await lastPostedBody(page, 'draft'));
  entry = await draftShelfEntry(page, intent);
  assert.ok(repairOwnedBy(entry, keptIdea), 'the save must carry the zero-match repair');
  await resetIntent(page);
  await resumeIntent(page, intent);
  await page.getByRole('group', {name: 'Unresolved dependency repair'}).waitFor();
  entry = await draftShelfEntry(page, intent);
  assert.ok(repairOwnedBy(entry, keptIdea), 'reload must keep the zero-match repair');
  await disarmUnresolvedSplit(page);
  await page.getByRole('button', {name: `Re-accept declined idea ${keptIdea}`, exact: true}).click();
  await updateProposal(page);
  checkCanonical('unresolved-zero-reaccept', await lastPostedBody(page, 'draft'));
  entry = await draftShelfEntry(page, intent);
  const claimed = repairOwnedBy(entry, keptIdea);
  assert.ok(claimed, 'the returned work must claim the retained repair');
  const live = await revisionTasks(page);
  assert.ok(
    live.some((task) => task.key === claimed.key),
    'the claimed repair must sit on the unique successor key'
  );
  assert.equal(await unresolvedGateCount(page), 0, 'a claimed repair is resolved: no more review');
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  checkCanonical('unresolved-zero-restore', saved);
  const tasks = saved.tasks as PostedTask[];
  const shelf = shelfOf(await draftShelfEntry(page, intent));
  const revivedB = taskForIdea(tasks, keptIdea);
  assert.ok(revivedB, 'returned B work must still be proposed after re-accepting A');
  assert.deepEqual(
    edgeIdeaSets(revivedB.depends_on, tasks, shelf),
    [[declinedA]],
    're-accept restores the retained B→A edge onto live work'
  );
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
});

test('unique successor with a live target adopts the carried repair exactly', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, declinedA, keptIdea} = await driveUnresolvedAmbiguity(page);
  let entry = await draftShelfEntry(page, intent);
  assert.ok(repairOwnedBy(entry, keptIdea), 'the ambiguous repair must wait under the old owner key');
  await disarmUnresolvedSplit(page);
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  checkCanonical('unresolved-live-target-save', await lastPostedBody(page, 'draft'));
  entry = await draftShelfEntry(page, intent);
  const claimed = repairOwnedBy(entry, keptIdea);
  assert.ok(claimed, 'the unique successor must adopt the carried repair onto its live target');
  const live = await revisionTasks(page);
  assert.ok(
    live.some((task) => task.key === claimed.key),
    'the adopted repair must sit on a live successor key'
  );
  assert.equal(await unresolvedGateCount(page), 0, 'an adopted repair is resolved: no more review');
  await saveUnchangedLimits(page);
  const saved = await lastPostedBody(page, 'draft');
  checkCanonical('unresolved-live-target-replay', saved);
  const tasks = saved.tasks as PostedTask[];
  const shelf = shelfOf(await draftShelfEntry(page, intent));
  const revivedB = taskForIdea(tasks, keptIdea);
  assert.ok(revivedB, 'kept B work must still be proposed after the replay save');
  assert.deepEqual(
    edgeIdeaSets(revivedB.depends_on, tasks, shelf),
    [[declinedA]],
    'the replay save restores the exact B→A edge onto live work'
  );
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
  assert.equal(
    repairOwnedBy(await draftShelfEntry(page, intent), keptIdea),
    null,
    'a fully replayed repair is consumed'
  );
});

test('repeated regeneration carries target lineage across every hop', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, declinedA, keptIdea} = await driveUnresolvedAmbiguity(page);
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  checkCanonical('unresolved-repeat-reaccept-A', await lastPostedBody(page, 'draft'));
  let live = await revisionTasks(page);
  const aLive = live.find((task) => task.idea_ids.length === 1 && task.idea_ids[0] === declinedA);
  assert.ok(aLive, 're-accepted A work must regenerate live');
  const twins = live.filter((task) => task.idea_ids.length === 1 && task.idea_ids[0] === keptIdea);
  assert.equal(twins.length, 2, 'the armed explore must keep B twinned while A returns');
  let entry = await draftShelfEntry(page, intent);
  const retained = repairOwnedBy(entry, keptIdea);
  assert.ok(retained, 'B must stay unassigned while twinned');
  assert.deepEqual(
    [...retained.repair.deps].sort(),
    [aLive.key].sort(),
    'the remap must follow the target onto live A'
  );
  const carried = (retained as unknown as {repair: {depIdeas?: Record<string, string[]>}}).repair.depIdeas || {};
  assert.deepEqual(carried[aLive.key], [declinedA], 'the remap must carry target lineage under the successor key');
  assert.ok((await unresolvedGateCount(page)) > 0, 'unresolved B stays visible while twinned');
  await disarmUnresolvedSplit(page);
  const headBefore = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page
    .locator('.change-field')
    .getByLabel('What should change?', {exact: true})
    .fill('Consolidate the tracking work');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, headBefore);
  live = await revisionTasks(page);
  const singles = live.filter((task) => task.idea_ids.length === 1 && task.idea_ids[0] === keptIdea);
  assert.equal(singles.length, 1, 'the disarmed explore must regenerate a unique B successor');
  const aNew = live.find((task) => task.idea_ids.length === 1 && task.idea_ids[0] === declinedA);
  assert.ok(aNew, 'the second regeneration must re-key live A');
  assert.notEqual(aNew.key, aLive.key, 'A must carry a fresh key after the second regeneration');
  entry = await draftShelfEntry(page, intent);
  const claimed = repairOwnedBy(entry, keptIdea);
  assert.ok(claimed, 'the unique B successor must claim the repair via the carried lineage');
  assert.equal(claimed.key, singles[0]?.key, 'the repair must move onto the unique successor key');
  assert.deepEqual([...claimed.repair.deps].sort(), [aNew.key].sort(), 'the claim must track the newest A key');
  assert.equal(await unresolvedGateCount(page), 0, 'a claimed repair is resolved: no more review');
  await saveUnchangedLimits(page);
  const saved = await lastPostedBody(page, 'draft');
  checkCanonical('unresolved-repeat-replay', saved);
  const tasks = saved.tasks as PostedTask[];
  const shelf = shelfOf(await draftShelfEntry(page, intent));
  const revivedB = taskForIdea(tasks, keptIdea);
  assert.ok(revivedB, 'kept B work must still be proposed after the replay save');
  assert.deepEqual(
    edgeIdeaSets(revivedB.depends_on, tasks, shelf),
    [[declinedA]],
    'the replay save restores the exact B→A edge across both regenerations'
  );
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
  assert.equal(repairOwnedBy(await draftShelfEntry(page, intent), keptIdea), null, 'replay consumes the record');
});

// FJ-385 saved-history (cross-device) recovery regressions: the local shelf
// is same-browser localStorage, so removing the draft entry simulates a
// resume on another device. Only previously-saved graph bytes (seeded
// history) are recoverable: in-memory-only edges never existed in any
// predecessor and are never treated as history, and merely graph-bearing
// revisions never qualify. Shared fixtures stay untouched; the private peer
// above serves the owner-private history read from seeded bytes alone.
interface HistoryTask {
  key: string;
  objective: string;
  outcomes: string[];
  idea_ids: string[];
  depends_on?: string[];
}
interface HistoryIdea {
  id: string;
  description: string;
  consequence: string;
  decision: string;
}
async function clearDraftEntry(page: Page, intent: string) {
  await page.evaluate((key: string) => localStorage.removeItem(key), `soda-intent-draft:v1:1:${intent}`);
}
async function seedCurrentHead(page: Page, head: Record<string, unknown>) {
  await page.evaluate(
    ({entry}: {entry: Record<string, unknown>}) => {
      (window as unknown as {intentStore: IntentStore}).intentStore.seed([entry]);
    },
    {entry: head}
  );
}
async function seedSavedHistory(page: Page, intent: string, revisions: Record<string, unknown>[]) {
  await page.evaluate(
    ({wanted, entries}: {wanted: string; entries: Record<string, unknown>[]}) => {
      (window as unknown as {intentStore: IntentStore}).intentStore.seedHistory(wanted, entries);
    },
    {wanted: intent, entries: revisions}
  );
}
async function currentRevision(page: Page): Promise<Record<string, unknown>> {
  return page.evaluate(() => {
    const host = document.querySelector('soda-intent') as unknown as {revision: unknown};
    return JSON.parse(JSON.stringify(host.revision)) as Record<string, unknown>;
  });
}
async function resumeWithoutShelf(page: Page, intent: string) {
  // Reset first: reset() re-persists the in-memory shelf under the old head,
  // so the entry must be cleared after the reset to simulate a new device.
  await resetIntent(page);
  await clearDraftEntry(page, intent);
  await resumeIntent(page, intent);
}
// Crafted-chain ids: retag the second hex digit, which fixture-serial ids
// leave as zero, so crafted revisions and tasks stay valid and unique.
function taggedID(id: string, mark: string): string {
  return `d${mark}${id.slice(2)}`;
}
function freshMark(base: string, want: string): string {
  return base[1] === want ? '7' : want;
}

test('saved history previews the exact predecessor and re-accept drafts it onto the current head', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  const preDecline = await currentRevision(page);
  const aTasks = (preDecline.tasks as HistoryTask[]).filter((task) => task.idea_ids.includes(declinedA));
  assert.ok(aTasks.length > 0, 'idea A must carry linked work before the decline');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const dropped = await lastPostedBody(page, 'draft');
  assert.equal(dropped.predecessor, preDecline.id, 'the decline save must name the pre-decline head');
  const proposal = await currentRevision(page);
  assert.equal(proposal.predecessor, dropped.id, 'the explore proposal must name the decline save');
  await seedCurrentHead(page, proposal);
  await seedSavedHistory(page, intent, [proposal, dropped, preDecline]);
  await resumeWithoutShelf(page, intent);
  // A new device has no shelf: no local recovery is offered.
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
  // Saved history previews the exact predecessor's recorded bytes.
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate.waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  for (const task of aTasks) await gate.getByText(task.objective, {exact: false}).waitFor();
  await gate.getByText(String(preDecline.id), {exact: false}).waitFor();
  // The user re-accept drives recovery: a new draft on the current head.
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.intent, intent);
  assert.equal(restored.predecessor, proposal.id, 'recovery must draft onto the freshly checked current head');
  const revived = restored.tasks as HistoryTask[];
  for (const task of aTasks)
    assert.ok(
      revived.some((row) => row.key === task.key),
      `recorded work ${task.key} must return on re-accept`
    );
  checkCanonical('history-exact-restore', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  assert.equal(await page.getByRole('group', {name: 'Saved history recovery'}).count(), 0);
});

test('a pruned or unrelated predecessor reports honest unavailability with no restore', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const dropped = await lastPostedBody(page, 'draft');
  const proposal = await currentRevision(page);
  await seedCurrentHead(page, proposal);
  // Pruned predecessor: present in history but carrying no work for A.
  await seedSavedHistory(page, intent, [proposal, dropped]);
  await resumeWithoutShelf(page, intent);
  const unavailable = 'No saved history can restore this declined work';
  await page.getByText(unavailable, {exact: false}).waitFor();
  assert.equal(
    await page
      .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
      .count(),
    0
  );
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
  // Unrelated chain: the named predecessor is absent from history entirely.
  await seedSavedHistory(page, intent, [proposal]);
  await resumeWithoutShelf(page, intent);
  await page.getByText(unavailable, {exact: false}).waitFor();
  assert.equal(
    await page
      .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
      .count(),
    0
  );
  // Re-accepting without saved bytes restores nothing: no invented work.
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('accept');
  await updateProposal(page);
  const noRestore = await lastPostedBody(page, 'draft');
  assert.equal(
    (noRestore.tasks as HistoryTask[]).some((task) => task.idea_ids.includes(declinedA)),
    false,
    'no saved bytes means no restored work'
  );
  checkCanonical('history-unavailable-no-restore', noRestore);
});

test('an in-memory-only edge is not offered as saved history', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const preDecline = await currentRevision(page);
  const aTasks = (preDecline.tasks as HistoryTask[]).filter((task) => task.idea_ids.includes(declinedA));
  assert.ok(aTasks.length > 0);
  // The edge below is chained in memory and never saved: the decline save
  // prunes it, so no predecessor carries it.
  const chainedA = await chainKeptOntoDeclined(page);
  assert.ok(chainedA, 'the draft must carry chained work');
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const dropped = await lastPostedBody(page, 'draft');
  assert.equal(dropped.predecessor, preDecline.id);
  const proposal = await currentRevision(page);
  await seedCurrentHead(page, proposal);
  await seedSavedHistory(page, intent, [proposal, dropped, preDecline]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  // The unsent edge key is previewed nowhere.
  assert.equal(await gate.getByText(String(chainedA), {exact: false}).count(), 0);
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(declinedA)),
    'recorded A work must still return'
  );
  const revivedB = revived.find((task) => task.idea_ids.includes(keptIdea));
  assert.ok(revivedB, 'kept B work must still be proposed');
  assert.deepEqual(
    revivedB.depends_on || [],
    [],
    'the in-memory-only edge was never saved and must not return cross-device'
  );
  checkCanonical('history-memory-edge-not-offered', restored);
});

test('repeated explored lineage traverses to the relevant saved graph without mass-merge', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  const aTasks = preTasks.filter((task) => task.idea_ids.includes(declinedA));
  assert.ok(aTasks.length > 0);
  assert.ok(
    preTasks.some((task) => !task.idea_ids.includes(declinedA)),
    'the chain must carry other-idea work to prove no mass-merge'
  );
  // Hand-built explored chain: the decline save drops A, a regeneration
  // re-keys the kept work, and a limits-only save advances the head. The
  // two middle hops are merely graph-bearing for other ideas.
  const tag = (id: string, mark: string) => `d${mark}${id.slice(2)}`;
  const hopMark = (base: string, want: string) => (base[1] === want ? '7' : want);
  const preID = String(pre.id);
  // One recorded edge inside the old graph: the first A task depends on the
  // first non-A task, so the traversal must remap it onto the regenerated
  // unique successor instead of pruning it silently.
  const bSeed = preTasks.find((task) => !task.idea_ids.includes(declinedA));
  assert.ok(bSeed, 'the chain must carry other-idea work to remap onto');
  const h1Tasks = preTasks.map((task) => ({...task}));
  const firstA = h1Tasks.find((task) => task.idea_ids.includes(declinedA));
  assert.ok(firstA);
  firstA.depends_on = [bSeed.key];
  const h1: Record<string, unknown> = {...pre, tasks: h1Tasks};
  const h2 = {
    ...pre,
    id: tag(preID, hopMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => (idea.id === declinedA ? {...idea, decision: 'decline'} : idea)),
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  const rekey = new Map(preTasks.map((task) => [task.key, tag(task.key, '9')]));
  const h3 = {
    ...h2,
    id: tag(preID, hopMark(preID, '5')),
    predecessor: h2.id,
    tasks: (h2.tasks as HistoryTask[]).map((task) => ({
      ...task,
      key: String(rekey.get(task.key)),
      depends_on: (task.depends_on || []).map((dep) => String(rekey.get(dep) || dep)),
    })),
  };
  const h4 = {...h3, id: tag(preID, hopMark(preID, '6')), predecessor: h3.id};
  const allIds = [
    String(h1.id),
    String(h2.id),
    String(h3.id),
    String(h4.id),
    ...preTasks.map((task) => task.key),
    ...rekey.values(),
  ];
  assert.equal(new Set(allIds).size, allIds.length, 'crafted chain ids must be unique');
  await seedCurrentHead(page, h4);
  await seedSavedHistory(page, intent, [h4, h3, h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  // The walk skipped the merely graph-bearing hops and names the relevant one.
  await gate.getByText(String(h1.id), {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.predecessor, h4.id, 'recovery must draft onto the explored head');
  const revived = restored.tasks as HistoryTask[];
  for (const task of aTasks)
    assert.ok(
      revived.some((row) => row.key === task.key),
      `relevant saved work ${task.key} must return`
    );
  for (const task of preTasks.filter((row) => !row.idea_ids.includes(declinedA)))
    assert.ok(!revived.some((row) => row.key === task.key), `older non-accepted work ${task.key} must not merge in`);
  assert.equal(revived.length, (h4.tasks as HistoryTask[]).length + aTasks.length);
  const revivedFirstA = revived.find((row) => row.key === firstA.key);
  assert.ok(revivedFirstA, 'the recorded A task with the edge must return');
  assert.deepEqual(
    revivedFirstA.depends_on || [],
    [rekey.get(bSeed.key)],
    'the recorded edge must remap onto the unique regenerated successor instead of pruning silently'
  );
  checkCanonical('history-traversal-no-merge', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

test('saved history restores a recorded kept-edge cut into the recovered set', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  const aTasks = preTasks.filter((task) => task.idea_ids.includes(declinedA));
  assert.ok(aTasks.length > 0);
  assert.ok(preTasks.some((task) => task.idea_ids.includes(keptIdea)));
  // The saved predecessor records kept B depending on removed A: the
  // decline save prunes that edge, so only history still knows it.
  const h1Tasks = preTasks.map((task) => ({...task}));
  const recordedB = h1Tasks.find((task) => task.idea_ids.includes(keptIdea));
  assert.ok(recordedB);
  recordedB.depends_on = aTasks.map((task) => task.key);
  const h1 = {...pre, tasks: h1Tasks};
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => (idea.id === declinedA ? {...idea, decision: 'decline'} : idea)),
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  // The recorded cut is previewed as returning, not silently dropped.
  await gate.getByText('returns on re-accept', {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.predecessor, h2.id);
  const revived = restored.tasks as HistoryTask[];
  for (const task of aTasks)
    assert.ok(
      revived.some((row) => row.key === task.key),
      `recorded work ${task.key} must return`
    );
  const revivedB = revived.find((task) => task.idea_ids.includes(keptIdea));
  assert.ok(revivedB, 'kept B work must still be proposed');
  assert.deepEqual(
    revivedB.depends_on || [],
    aTasks.map((task) => task.key),
    'the recorded B→A edge must return on re-accept instead of staying cut'
  );
  checkCanonical('history-repair-kept-edge', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

test('saved history leaves an unprovable removed-edge cut with an explicit warning', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  const bSeed = preTasks.find((task) => !task.idea_ids.includes(declinedA));
  assert.ok(bSeed);
  // The old graph records A depending on old B; the head regenerated B
  // into twins, so the edge proves neither successor.
  const h1Tasks = preTasks.map((task) => ({...task}));
  const firstA = h1Tasks.find((task) => task.idea_ids.includes(declinedA));
  assert.ok(firstA);
  firstA.depends_on = [bSeed.key];
  const h1 = {...pre, tasks: h1Tasks};
  const preID = String(pre.id);
  const twinA = taggedID(bSeed.key, '8');
  const twinB = taggedID(bSeed.key, '9');
  assert.equal(new Set([twinA, twinB, bSeed.key]).size, 3, 'twinned keys must stay unique');
  const twins = [
    {...bSeed, key: twinA, depends_on: [] as string[]},
    {...bSeed, key: twinB, depends_on: [] as string[]},
  ];
  const h2 = {
    ...pre,
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => (idea.id === declinedA ? {...idea, decision: 'decline'} : idea)),
    tasks: [...preTasks.filter((task) => !task.idea_ids.includes(declinedA) && task.key !== bSeed.key), ...twins],
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  // The unprovable edge is warned about instead of guessed onto a twin.
  await gate.getByText('stays cut instead of being guessed', {exact: false}).waitFor();
  assert.equal(await gate.getByText(twinA, {exact: false}).count(), 0);
  assert.equal(await gate.getByText(twinB, {exact: false}).count(), 0);
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  const revivedFirstA = revived.find((row) => row.key === firstA.key);
  assert.ok(revivedFirstA, 'the recorded A work must still return');
  assert.deepEqual(revivedFirstA.depends_on || [], [], 'the unprovable edge must stay cut, never remapped by guess');
  checkCanonical('history-unprovable-edge-cut', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

test('stale history rows refuse after the head advances and refresh on the new head', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  const preDecline = await currentRevision(page);
  await page
    .getByRole('combobox', {name: /^Choice:/})
    .first()
    .selectOption('decline');
  await updateProposal(page);
  const dropped = await lastPostedBody(page, 'draft');
  const proposal = await currentRevision(page);
  await seedCurrentHead(page, proposal);
  await seedSavedHistory(page, intent, [proposal, dropped, preDecline]);
  await resumeWithoutShelf(page, intent);
  const historyButton = `Re-accept declined idea from saved history ${declinedA}`;
  await page.getByRole('button', {name: historyButton, exact: true}).waitFor();
  // Advance the head without re-accepting: a limits-only save.
  await saveUnchangedLimits(page);
  const advanced = await currentRevision(page);
  assert.notEqual(String(advanced.id), String(proposal.id));
  // The stale rows refuse: no history re-accept stays actionable, and the
  // frozen page (which cannot carry the new head) reports unavailable
  // through the head-binding check instead of recovering.
  assert.equal(await page.getByRole('button', {name: historyButton, exact: true}).count(), 0);
  await page.getByText('No saved history can restore this declined work', {exact: false}).waitFor();
  // A stale row applied by hand still refuses instead of staging.
  const refused = await page.evaluate((id: string) => {
    const host = document.querySelector('soda-intent') as unknown as {
      revision: {id: string; ideas: {id: string; decision: string}[]};
      savedHistory: {
        idea: string;
        revision: string;
        tasks: {key: string; objective: string; outcomes: string[]; idea_ids: string[]}[];
        repairs: [];
        cutEdges: [];
        ideas: {id: string; description: string; consequence: string; decision: string}[];
      }[];
      savedHistoryHead: string;
      shelvedTasks: unknown[];
      recoverFromHistory(row: string): void;
    };
    host.savedHistoryHead = 'd0000000000000000000000';
    host.savedHistory = [
      {
        idea: id,
        revision: 'd0000000000000000000000',
        tasks: [{key: 'd111111111111111111111111', objective: 'Stale work', outcomes: ['o'], idea_ids: [id]}],
        repairs: [],
        cutEdges: [],
        ideas: [{id, description: 'd', consequence: 'c', decision: 'decline'}],
      },
    ];
    host.recoverFromHistory(id);
    return {shelved: host.shelvedTasks.length, rows: host.savedHistory.length};
  }, declinedA);
  assert.deepEqual(refused, {shelved: 0, rows: 0}, 'a stale row must refuse and clean up, never stage');
  // Refresh: seeding the new head's chain restores the same recovery.
  const fresh = await currentRevision(page);
  await seedCurrentHead(page, fresh);
  await seedSavedHistory(page, intent, [fresh, proposal, dropped, preDecline]);
  await resumeWithoutShelf(page, intent);
  await page.getByRole('button', {name: historyButton, exact: true}).waitFor();
  await page
    .getByRole('group', {name: 'Saved history recovery'})
    .getByText(String(preDecline.id), {exact: false})
    .waitFor();
});

test('saved history refuses recorded work whose linked idea changed meaning', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  assert.ok(preTasks.some((task) => task.idea_ids.includes(declinedA)));
  // The old graph carries a dual-linked task: pure A work plus joint A+B
  // work. The head changed B's meaning, so only the pure work recovers.
  const dualKey = taggedID(preTasks[0]?.key || String(pre.id), 'd');
  const dual: HistoryTask = {
    key: dualKey,
    objective: 'Joint A and B tracking work',
    outcomes: ['Demonstrate the selected outcome'],
    idea_ids: [declinedA, keptIdea],
  };
  const h1 = {...pre, tasks: [...preTasks, dual]};
  const preID = String(pre.id);
  const headB = (pre.ideas as HistoryIdea[]).find((idea) => idea.id === keptIdea);
  assert.ok(headB);
  const h2 = {
    ...pre,
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return {...idea, decision: 'decline'};
      if (idea.id === keptIdea) return {...idea, description: `${idea.description} (audited)`};
      return idea;
    }),
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  assert.equal(await gate.getByText('Joint A and B tracking work', {exact: false}).count(), 0);
  await gate.getByText('Nothing was guessed', {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(declinedA)),
    'pure recorded A work must still return'
  );
  assert.equal(
    revived.some((task) => task.key === dualKey),
    false,
    'work linked to the meaning-changed idea must refuse'
  );
  checkCanonical('history-linked-meaning-refused', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

test('saved history refuses recorded work whose linked correction changed', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  assert.ok(preTasks.some((task) => task.idea_ids.includes(declinedA)));
  const dualKey = taggedID(preTasks[0]?.key || String(pre.id), 'd');
  const dual: HistoryTask = {
    key: dualKey,
    objective: 'Joint A and B tracking work',
    outcomes: ['Demonstrate the selected outcome'],
    idea_ids: [declinedA, keptIdea],
  };
  const withCorrection = (correction: string) =>
    (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return idea;
      if (idea.id === keptIdea) return {...idea, decision: 'edit', correction};
      return idea;
    });
  const h1 = {...pre, ideas: withCorrection('Original correction'), tasks: [...preTasks, dual]};
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    ideas: withCorrection('Revised correction').map((idea) =>
      idea.id === declinedA ? {...idea, decision: 'decline'} : idea
    ),
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  assert.equal(await gate.getByText('Joint A and B tracking work', {exact: false}).count(), 0);
  await gate.getByText('Nothing was guessed', {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(declinedA)),
    'pure recorded A work must still return'
  );
  assert.equal(
    revived.some((task) => task.key === dualKey),
    false,
    'work linked to the changed correction must refuse'
  );
  checkCanonical('history-linked-correction-refused', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

// FJ-427 genuine single-unavailable regression: the committed unresolved legs
// always keep two same-idea successors until the target returns. This drives
// the shipped reconcile against a regenerated proposal with a UNIQUE
// successor that is NOT adoptable (it carries an out-of-record edge), so the
// record must stay retained under the old owner key and visible — then a real
// target-live event (visible re-accept + update) adopts it exactly: one
// successor, the recorded edge restored, nothing guessed.
test('unique nonadoptable successor stays retained and visible until its live target adopts it exactly', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, declinedA, keptIdea} = await driveUnresolvedAmbiguity(page);
  // Pin lineage exactly like the ambiguous leg so the framed remap below hits
  // the true single-unavailable branch.
  await saveUnchangedLimits(page);
  const otherIdea = await page.evaluate(
    ([declined, kept]: string[]) => {
      const host = document.querySelector('soda-intent') as unknown as {revision?: {ideas?: {id: string}[]}};
      const ids = (host.revision?.ideas || []).map((idea) => idea.id);
      return ids.find((id) => id !== declined && id !== kept) || '';
    },
    [declinedA, keptIdea]
  );
  assert.ok(otherIdea, 'the draft must carry a third idea to frame the out-of-record edge');
  const depsBefore = (repairOwnedBy(await draftShelfEntry(page, intent), keptIdea)?.repair.deps || []).slice().sort();
  assert.ok(depsBefore.length > 0, 'ambiguity setup must retain the repair before framing');
  // A regenerated unique B successor carrying an edge the record never saw:
  // present exactly once, but adoptable by neither rule (the extra edge is in
  // neither the record nor the live set it proves).
  const framed = await page.evaluate(
    ([kept, other]: string[]) => {
      const host = document.querySelector('soda-intent') as unknown as {
        revision: {tasks: {key: string; idea_ids: string[]; depends_on?: string[]}[]; ideas: {id: string}[]};
        shelvedPrunedDeps: Record<string, {deps: string[]}>;
        reconcileShelvedTasks(proposal: unknown): void;
      };
      const rev = JSON.parse(JSON.stringify(host.revision)) as typeof host.revision;
      const keptId = kept || '';
      const otherId = other || '';
      const twins = rev.tasks.filter((task) => task.idea_ids.length === 1 && task.idea_ids[0] === keptId);
      const cTask = rev.tasks.find((task) => task.idea_ids.includes(otherId));
      const firstTwin = twins[0];
      if (twins.length !== 2 || !cTask || !firstTwin) return null;
      const soleKey = 'd' + 'e'.repeat(22) + 'a1';
      const sole = {...firstTwin, key: soleKey, depends_on: [cTask.key]};
      const proposal = {
        ...rev,
        tasks: [...rev.tasks.filter((task) => !(task.idea_ids.length === 1 && task.idea_ids[0] === kept)), sole],
      };
      host.reconcileShelvedTasks(proposal);
      return {soleKey, ownerKeys: Object.keys(host.shelvedPrunedDeps)};
    },
    [keptIdea, otherIdea]
  );
  assert.ok(framed, 'the kept B work must be twinned to frame the single-unavailable proposal');
  const entry = await draftShelfEntry(page, intent);
  const retained = repairOwnedBy(entry, keptIdea);
  assert.ok(retained, 'the single-unavailable repair must be retained, not dropped silently');
  assert.notEqual(retained.key, framed.soleKey, 'the unique successor must not claim the unadoptable record');
  assert.deepEqual(
    [...retained.repair.deps].sort(),
    depsBefore,
    'the retained record must keep its exact recorded edges'
  );
  assert.equal(framed.ownerKeys.includes(framed.soleKey), false, 'no repair may sit on the unadoptable successor key');
  await page.getByRole('group', {name: 'Unresolved dependency repair'}).waitFor();
  const liveBefore = await revisionTasks(page);
  assert.equal(
    liveBefore.some((task) => task.key === framed.soleKey),
    false,
    'the framed successor must not materialize as live work: no guess'
  );
  // Later events, fully real: disarm the split, regenerate (a change-submit
  // explore rebuilds work from the ideas and collapses the twins into one
  // successor, unlike a re-accept explore that re-keys the submitted shape),
  // then the target-live event re-accepts A and updates.
  await disarmUnresolvedSplit(page);
  const exploreHead = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  await page.getByRole('button', {name: 'Change something', exact: true}).click();
  await page
    .locator('.change-field')
    .getByLabel('What should change?', {exact: true})
    .fill('Consolidate the tracking work');
  await page.getByRole('button', {name: 'Submit change', exact: true}).click();
  await page.waitForFunction((prev: string) => {
    const host = document.querySelector('soda-intent') as unknown as {revision?: {id: string}; busy: boolean};
    return !host.busy && (host.revision?.id || '') !== prev;
  }, exploreHead);
  const liveAfter = await revisionTasks(page);
  const uniqueB = liveAfter.filter((task) => task.idea_ids.length === 1 && task.idea_ids[0] === keptIdea);
  assert.equal(uniqueB.length, 1, 'the disarmed regeneration must collapse the twins into one successor');
  const claimed = repairOwnedBy(await draftShelfEntry(page, intent), keptIdea);
  assert.ok(claimed, 'the unique successor must claim the retained repair');
  assert.equal(claimed.key, uniqueB[0]?.key, 'the repair must move onto the unique successor key');
  assert.equal(await unresolvedGateCount(page), 0, 'a claimed repair is resolved: no more review');
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const saved = await lastPostedBody(page, 'draft');
  checkCanonical('unique-nonadoptable-adopted-exactly', saved);
  const tasks = saved.tasks as PostedTask[];
  assert.equal(
    tasks.some((task) => task.key === framed.soleKey),
    false,
    'no invented edge onto the framed key'
  );
  const singles = tasks.filter((task) => task.idea_ids.length === 1 && task.idea_ids[0] === keptIdea);
  assert.equal(singles.length, 1, 'adoption must land on exactly one successor, never a second twin');
  const shelf = shelfOf(await draftShelfEntry(page, intent));
  assert.deepEqual(
    edgeIdeaSets(singles[0]?.depends_on, tasks, shelf),
    [[declinedA]],
    'adoption restores the exact B→A edge onto live work'
  );
  assert.equal(await unresolvedGateCount(page), 0, 'an adopted repair is resolved: no more review');
  assert.equal(
    repairOwnedBy(await draftShelfEntry(page, intent), keptIdea),
    null,
    'a fully replayed repair is consumed'
  );
  const liveKeys = new Set(tasks.map((task) => task.key));
  for (const task of tasks)
    for (const dep of task.depends_on || [])
      assert.ok(liveKeys.has(dep), `no invented edges: ${dep} must be live work`);
});

// FJ-427 real-reload regression: the committed reload legs reset/re-request on
// the same component instance. This remounts the workspace (the old tree is
// disposed and a new soda-intent instance is created: in-memory revisions,
// shelf and repairs are gone, while the same-browser localStorage entry and
// the window peer survive) and proves shelf/repair recovery plus the
// saved-history path across it, using only the existing fixtures.
test('saved decline recovery survives a new component instance with dependencies and history intact', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA && keptIdea);
  const preDecline = await currentRevision(page);
  const aKey = await chainKeptOntoDeclined(page);
  assert.ok(aKey, 'the draft must carry three work items to chain');
  await declineIdeaByDescription(page, declinedA);
  await updateProposal(page);
  const dropped = await lastPostedBody(page, 'draft');
  assert.equal(dropped.predecessor, preDecline.id, 'the decline save must name the pre-decline head');
  checkCanonical('new-instance-decline-drop', dropped);
  // Settle the shelf against a saved head before recreating the instance.
  await saveUnchangedLimits(page);
  const headId = String((await currentRevision(page)).id);
  const settled = await draftShelfEntry(page, intent);
  assert.equal(settled?.base, headId, 'the limits-only save must persist the shelf at the saved head');
  // New component instance: the old tree is disposed and a fresh soda-intent
  // is created (localStorage and the window peer survive; in-memory shelf
  // and repairs are gone). The remount presents the current route, so settle
  // for the fresh element instead of the blank entry.
  await page.evaluate(() => {
    (window as unknown as {__oldIntent?: unknown}).__oldIntent = document.querySelector('soda-intent');
  });
  await page.evaluate(() => window.workspaceFixture.remount());
  await page.waitForFunction(() => {
    const host = document.querySelector('soda-intent') as unknown as {busy: boolean} | null;
    return !!host && !host.busy;
  });
  assert.equal(
    await page.evaluate(
      () => document.querySelector('soda-intent') !== (window as unknown as {__oldIntent?: unknown}).__oldIntent
    ),
    true,
    'remount must create a new soda-intent instance: no in-memory state survives'
  );
  await resumeIntent(page, intent);
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  const after = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  assert.equal(after, headId, 'the new instance must resume the same head so the shelf still applies');
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
  const entry = await draftShelfEntry(page, intent);
  const shelved = ((entry?.shelf as {key: string; idea_ids: string[]}[] | undefined) || []).filter((task) =>
    (task.idea_ids || []).includes(declinedA)
  );
  assert.ok(shelved.length > 0, 'the new instance must re-offer the shelved A work');
  assert.ok(
    shelved.some((task) => task.key === aKey),
    'the new instance must keep the exact shelved A key'
  );
  const shelvedKeys = new Set(shelved.map((task) => task.key));
  const repairs = Object.values(
    (entry?.shelfDeps as Record<string, {deps: string[]; ideas: string[]}> | undefined) || {}
  );
  assert.ok(
    repairs.some(
      (repair) => (repair.ideas || []).includes(keptIdea) && (repair.deps || []).some((dep) => shelvedKeys.has(dep))
    ),
    'the new instance must keep the recorded B→A repair'
  );
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  checkCanonical('new-instance-restored-edge', restored);
  const revived = restored.tasks as PostedTask[];
  assert.ok(
    revived.some((task) => task.key === aKey),
    'shelved A work must return after new-instance recovery'
  );
  const revivedB = revived.find((task) => (task.idea_ids || []).includes(keptIdea));
  assert.ok(revivedB, 'kept B work must still be proposed after new-instance recovery');
  assert.deepEqual(revivedB.depends_on || [], [aKey], 'new-instance recovery must restore the B→A edge intact');
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
  // History leg across the same new instance: a fresh decline of the
  // correction-free idea, then a shelf-less resume proves the saved-history
  // path here too. (A declined edit-idea loses its correction across the
  // explore, which the history rule honestly treats as changed meaning.)
  const preDecline2 = await currentRevision(page);
  await declineIdeaByDescription(page, declinedA);
  await updateProposal(page);
  const dropped2 = await lastPostedBody(page, 'draft');
  assert.equal(dropped2.predecessor, preDecline2.id, 'the second decline save must name its pre-decline head');
  checkCanonical('new-instance-history-decline', dropped2);
  const proposal2 = await currentRevision(page);
  await seedCurrentHead(page, proposal2);
  await seedSavedHistory(page, intent, [proposal2, dropped2, preDecline2]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  await gate.getByText(String(preDecline2.id), {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const viaHistory = await lastPostedBody(page, 'draft');
  assert.equal(viaHistory.predecessor, proposal2.id, 'history recovery must draft onto the current head');
  const revived2 = viaHistory.tasks as HistoryTask[];
  assert.ok(
    revived2.some((task) => task.idea_ids.includes(declinedA)),
    'recorded A work must return after new-instance history recovery'
  );
  checkCanonical('new-instance-history-restored', viaHistory);
  assert.equal(await page.getByRole('group', {name: 'Saved history recovery'}).count(), 0);
});

// FJ-427 visible limit-edit regression: the committed limits-only leg forces
// dirty through private state. This drives the real visible Planning calls
// input to a narrowed value and saves through Update proposal: scope narrows,
// no re-explore fires, and the declined-work recovery survives — with no
// private-state mutation anywhere in the test.
test('a visible work-item limit edit narrows scope while preserving recovery without re-exploring', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  assert.ok(declinedA);
  await declineIdeaByDescription(page, declinedA);
  await updateProposal(page);
  checkCanonical('visible-limit-decline-drop', await lastPostedBody(page, 'draft'));
  await page.getByRole('group', {name: 'Declined work recovery'}).waitFor();
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
  const exploresBefore = await exploreCallCount(page);
  const draftsBefore = await revisionPostCount(page);
  // Visible limit edit only: the work-item limit input, no decision touched.
  await setMaxIssues(page, '12');
  await page.getByText('More options · up to 12 work items', {exact: true}).waitFor();
  await updateProposal(page);
  const narrowed = await lastPostedBody(page, 'draft');
  assert.equal(narrowed.max_issues, 12, 'the visible edit must narrow the saved planning scope');
  assert.equal(await exploreCallCount(page), exploresBefore, 'a limits-only save must not re-explore');
  assert.equal(await revisionPostCount(page), draftsBefore + 1, 'a limits-only save must post exactly one draft');
  const kept = (narrowed.ideas as {id: string; decision: string}[]).find((idea) => idea.id === declinedA);
  assert.equal(kept?.decision, 'decline', 'the decline decision must survive the limits-only save');
  checkCanonical('visible-limit-narrowed', narrowed);
  const head = await page.evaluate(
    () => (document.querySelector('soda-intent') as unknown as {revision?: {id: string}}).revision?.id || ''
  );
  const settledEntry = await draftShelfEntry(page, intent);
  assert.equal(settledEntry?.base, head, 'the limits-only save must persist the shelf at the narrowed head');
  const shelved = shelfOf(settledEntry).filter((task) => task.idea_ids.includes(declinedA));
  assert.ok(shelved.length > 0, 'the limits-only save must keep the shelved A work');
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).waitFor();
  // Recovery still functions afterwards on the narrowed head.
  await page.getByRole('button', {name: `Re-accept declined idea ${declinedA}`, exact: true}).click();
  await updateProposal(page);
  const recovered = await lastPostedBody(page, 'draft');
  assert.ok(
    (recovered.tasks as {idea_ids: string[]}[]).some((task) => task.idea_ids.includes(declinedA)),
    'shelved A work must return after the narrowed save'
  );
  checkCanonical('visible-limit-recovered', recovered);
  assert.equal(await page.getByRole('group', {name: 'Declined work recovery'}).count(), 0);
});

// FJ-453 effective-meaning match: the real UI-created change→decline→save→
// cross-device path. Submitting a change creates the idea with
// description=correction (decision edit); Decline deletes the correction
// itself, so the history walk must still match on unchanged effective
// meaning (description/consequence identical, correction removed by the
// Decline transition) instead of stopping on the strict comparison.
test('saved history recovers a UI-created change decline cross-device on unchanged effective meaning', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const corrected = ids[1];
  assert.ok(corrected);
  const preDecline = await currentRevision(page);
  const before = (preDecline.ideas as {id: string; decision: string; correction?: string}[]).find(
    (idea) => idea.id === corrected
  );
  assert.equal(before?.decision, 'edit', 'the UI-created change must be an edit idea');
  assert.ok(before?.correction, 'the UI-created change must carry a correction');
  assert.ok(
    (preDecline.tasks as HistoryTask[]).some((task) => task.idea_ids.includes(corrected)),
    'the correction idea must carry linked work before the decline'
  );
  await declineIdeaByDescription(page, corrected);
  await updateProposal(page);
  const declineSave = await lastPostedBody(page, 'draft');
  const head = await currentRevision(page);
  const after = (declineSave.ideas as {id: string; decision: string; description: string; correction?: string}[]).find(
    (idea) => idea.id === corrected
  );
  assert.equal(after?.decision, 'decline', 'the human decline decision must be preserved');
  assert.equal(after?.correction, undefined, 'Decline itself deletes the correction (system cleanup)');
  assert.equal(
    (after as {description: string}).description,
    (before as unknown as {description: string}).description,
    'description unchanged: system-deleted correction, not human-changed meaning'
  );
  assert.equal(
    (declineSave.tasks as HistoryTask[]).some((task) => task.idea_ids.includes(corrected)),
    false,
    'the decline save must drop the correction-linked work'
  );
  checkCanonical('history-effective-meaning-decline-drop', declineSave);
  // Cross-device: a shelfless resume over the exact saved chain.
  await seedCurrentHead(page, head);
  await seedSavedHistory(page, intent, [head, declineSave, preDecline]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  const reaccept = gate.getByRole('button', {
    name: `Re-accept declined idea from saved history ${corrected}`,
    exact: true,
  });
  await reaccept.waitFor();
  await gate.getByText(String(preDecline.id), {exact: false}).waitFor();
  await reaccept.click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.predecessor, head.id, 'history recovery must draft onto the current head (CAS preserved)');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(corrected)),
    'recorded correction work must return after cross-device history recovery'
  );
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  checkCanonical('history-effective-meaning-restored', restored);
  assert.equal(await page.getByRole('group', {name: 'Saved history recovery'}).count(), 0);
});

// FJ-453 effective-meaning mismatch: a human-emptied correction on an edit
// idea is a meaning change, never the Decline cleanup match. The linked
// work refuses with honest unavailability — no merge, no invention — while
// pure recorded work still returns.
test('saved history refuses recorded work whose edit correction was emptied', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  assert.ok(preTasks.some((task) => task.idea_ids.includes(declinedA)));
  // The old graph carries a dual-linked task: pure A work plus joint A+B
  // work. The head emptied B's correction while B stays an edit idea, so
  // only the pure work recovers.
  const dualKey = taggedID(preTasks[0]?.key || String(pre.id), 'd');
  const dual: HistoryTask = {
    key: dualKey,
    objective: 'Joint A and B tracking work',
    outcomes: ['Demonstrate the selected outcome'],
    idea_ids: [declinedA, keptIdea],
  };
  const withCorrection = (correction: string) =>
    (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return idea;
      if (idea.id === keptIdea) return {...idea, decision: 'edit', correction};
      return idea;
    });
  const h1 = {...pre, ideas: withCorrection('Original correction'), tasks: [...preTasks, dual]};
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    ideas: withCorrection('').map((idea) => (idea.id === declinedA ? {...idea, decision: 'decline'} : idea)),
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  assert.equal(await gate.getByText('Joint A and B tracking work', {exact: false}).count(), 0);
  await gate.getByText('Nothing was guessed', {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(declinedA)),
    'pure recorded A work must still return'
  );
  assert.equal(
    revived.some((task) => task.key === dualKey),
    false,
    'work linked to the emptied correction must refuse'
  );
  checkCanonical('history-edit-correction-emptied-refused', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

// FJ-457 deleted-correction narrowing, negative leg: a deleted SUBSTANTIVE
// correction is a meaning change, never the Decline/accept/pending cleanup
// match. Shared description “public access” with a historical correction
// “restricted to requester/manager” cleared by an accept transition must
// MISMATCH: the joint recorded work refuses with honest unavailability
// while pure recorded work still returns. The pre-457 over-equivalence (any
// nonempty correction matching an empty non-edit side) would have silently
// equated the two meanings.
test('saved history refuses recorded work whose deleted correction was substantive', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  assert.ok(preTasks.some((task) => task.idea_ids.includes(declinedA)));
  // The old graph carries a dual-linked task: pure A work plus joint A+B
  // work. The head kept B's shared description but cleared its substantive
  // correction through an accept, so only the pure work recovers.
  const dualKey = taggedID(preTasks[0]?.key || String(pre.id), 'd');
  const dual: HistoryTask = {
    key: dualKey,
    objective: 'Joint A and B tracking work',
    outcomes: ['Demonstrate the selected outcome'],
    idea_ids: [declinedA, keptIdea],
  };
  const sharedDescription = 'public access';
  const substantive = 'restricted to requester/manager';
  const h1 = {
    ...pre,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return idea;
      if (idea.id === keptIdea)
        return {...idea, description: sharedDescription, decision: 'edit', correction: substantive};
      return idea;
    }),
    tasks: [...preTasks, dual],
  };
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return {...idea, decision: 'decline'};
      if (idea.id === keptIdea) {
        // The accept transition itself cleans up the correction (decidedIdea
        // deletes it): model the cleared non-edit side with no correction
        // key at all, so this test exercises the deleted-correction path
        // instead of a both-sides-carry mismatch.
        const cleared: Record<string, unknown> = {...idea, description: sharedDescription, decision: 'accept'};
        delete cleared.correction;
        return cleared;
      }
      return idea;
    }),
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .waitFor();
  assert.equal(await gate.getByText('Joint A and B tracking work', {exact: false}).count(), 0);
  await gate.getByText('Nothing was guessed', {exact: false}).waitFor();
  await gate
    .getByRole('button', {name: `Re-accept declined idea from saved history ${declinedA}`, exact: true})
    .click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(declinedA)),
    'pure recorded A work must still return'
  );
  assert.equal(
    revived.some((task) => task.key === dualKey),
    false,
    'work linked to the substantively corrected idea must refuse'
  );
  checkCanonical('history-deleted-substantive-refused', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

// FJ-457 deleted-correction narrowing, positive leg: a deleted correction
// that exactly duplicates the shared description IS the
// Decline/accept/pending cleanup match (the FJ-453 UI-created change shape:
// a change submit creates the idea with description === correction, so the
// transition-deleted correction duplicates the description). The joint
// recorded work must recover through the workflow cleanup path.
test('saved history recovers recorded work whose deleted correction duplicated the description', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const declinedA = ids[0];
  const keptIdea = ids[1];
  assert.ok(declinedA);
  assert.ok(keptIdea);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  assert.ok(preTasks.some((task) => task.idea_ids.includes(declinedA)));
  // The old graph carries a dual-linked task: pure A work plus joint A+B
  // work. The head kept B's shared description while the accept transition
  // cleaned up its duplicated correction, so both recover.
  const dualKey = taggedID(preTasks[0]?.key || String(pre.id), 'd');
  const dual: HistoryTask = {
    key: dualKey,
    objective: 'Joint A and B tracking work',
    outcomes: ['Demonstrate the selected outcome'],
    idea_ids: [declinedA, keptIdea],
  };
  const sharedDescription = 'Summarize nightly state';
  const h1 = {
    ...pre,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return idea;
      if (idea.id === keptIdea)
        return {...idea, description: sharedDescription, decision: 'edit', correction: sharedDescription};
      return idea;
    }),
    tasks: [...preTasks, dual],
  };
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => {
      if (idea.id === declinedA) return {...idea, decision: 'decline'};
      if (idea.id === keptIdea) {
        // The accept transition itself cleans up the correction (decidedIdea
        // deletes it): model the cleared non-edit side with no correction
        // key at all, so this test exercises the deleted-correction path
        // instead of a both-sides-carry match.
        const cleared: Record<string, unknown> = {...idea, description: sharedDescription, decision: 'accept'};
        delete cleared.correction;
        return cleared;
      }
      return idea;
    }),
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    tasks: preTasks.filter((task) => !task.idea_ids.includes(declinedA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  const reaccept = gate.getByRole('button', {
    name: `Re-accept declined idea from saved history ${declinedA}`,
    exact: true,
  });
  await reaccept.waitFor();
  await gate.getByText('Joint A and B tracking work', {exact: false}).waitFor();
  await reaccept.click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.predecessor, h2.id, 'history recovery must draft onto the current head');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(declinedA)),
    'pure recorded A work must still return'
  );
  assert.ok(
    revived.some((task) => task.key === dualKey),
    'work linked through the duplicated-description cleanup must return'
  );
  checkCanonical('history-deleted-duplicate-restored', restored);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
});

// FJ-454 outgoing same-key: a live dep under a known key now requires EXACT
// old/current idea-set equality. Historical B linked b while current B
// links c under its stable key (meanings of B and C unchanged, only the
// linkage moved), so the recorded A→B edge stays cut with an explicit
// warning instead of being restored onto changed work. Pure A work still
// recovers.
test('saved history keeps a same-key live target edge cut when its idea linkage changed', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB, ideaC] = ids;
  assert.ok(ideaA && ideaB && ideaC);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  const aSeed = preTasks.find((task) => task.idea_ids.includes(ideaA));
  const bSeed = preTasks.find((task) => task.idea_ids.includes(ideaB));
  assert.ok(aSeed && bSeed);
  // The old graph records A depending on B under B's stable key.
  const h1Tasks = preTasks.map((task) => ({...task}));
  const recordedA = h1Tasks.find((task) => task.key === aSeed.key);
  assert.ok(recordedA);
  recordedA.depends_on = [bSeed.key];
  const h1 = {...pre, tasks: h1Tasks};
  const preID = String(pre.id);
  // Same-key changed lineage: the head keeps B's task key while rebinding
  // its ideas onto C. Meanings of B and C are untouched, only the linkage
  // moved — a canonical-valid saved draft with changed bindings.
  const h2 = {
    ...pre,
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => (idea.id === ideaA ? {...idea, decision: 'decline'} : idea)),
    tasks: preTasks
      .filter((task) => !task.idea_ids.includes(ideaA))
      .map((task) => (task.key === bSeed.key ? {...task, idea_ids: [ideaC]} : task)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  const reaccept = gate.getByRole('button', {
    name: `Re-accept declined idea from saved history ${ideaA}`,
    exact: true,
  });
  await reaccept.waitFor();
  await gate.getByText('stays cut instead of being guessed', {exact: false}).waitFor();
  await reaccept.click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.predecessor, h2.id, 'history recovery must draft onto the current head');
  const revived = restored.tasks as HistoryTask[];
  const revivedA = revived.find((task) => task.key === aSeed.key);
  assert.ok(revivedA, 'the recorded A work must still return');
  assert.deepEqual(revivedA.depends_on || [], [], 'the A→B edge across changed lineage must stay cut, never restored');
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  checkCanonical('history-samekey-target-cut', restored);
});

// FJ-454 same-key repair owner: a live repair owner under a known key also
// requires EXACT old/current idea-set equality. The recorded B→A edge owner
// keeps its key while rebinding onto C, so the repair never resolves
// silently: the edge stays cut with an explicit warning instead of
// restoring onto changed work.
test('saved history never resolves a same-key repair owner whose idea linkage changed', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB, ideaC] = ids;
  assert.ok(ideaA && ideaB && ideaC);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  const aSeed = preTasks.find((task) => task.idea_ids.includes(ideaA));
  const bSeed = preTasks.find((task) => task.idea_ids.includes(ideaB));
  assert.ok(aSeed && bSeed);
  // The old graph records kept B depending on removed A (repair direction).
  const h1Tasks = preTasks.map((task) => ({...task}));
  const recordedB = h1Tasks.find((task) => task.key === bSeed.key);
  assert.ok(recordedB);
  recordedB.depends_on = [aSeed.key];
  const h1 = {...pre, tasks: h1Tasks};
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => (idea.id === ideaA ? {...idea, decision: 'decline'} : idea)),
    tasks: preTasks
      .filter((task) => !task.idea_ids.includes(ideaA))
      .map((task) => (task.key === bSeed.key ? {...task, idea_ids: [ideaC]} : task)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  const reaccept = gate.getByRole('button', {
    name: `Re-accept declined idea from saved history ${ideaA}`,
    exact: true,
  });
  await reaccept.waitFor();
  await gate.getByText('stays cut instead of being guessed', {exact: false}).waitFor();
  await reaccept.click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  const revived = restored.tasks as HistoryTask[];
  assert.ok(
    revived.some((task) => task.idea_ids.includes(ideaA)),
    'the recorded A work must still return'
  );
  const aKeys = new Set(revived.filter((task) => task.idea_ids.includes(ideaA)).map((task) => task.key));
  for (const task of revived.filter((row) => row.idea_ids.includes(ideaC)))
    for (const dep of task.depends_on || [])
      assert.ok(!aKeys.has(dep), `no silent restore onto changed work: ${task.key} must not depend on ${dep}`);
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  checkCanonical('history-samekey-owner-cut', restored);
});

// FJ-454 no over-refusal: an unchanged-linkage same-key transfer still
// succeeds. Historical B linked b and current B still links b under its
// stable key, so the recorded A→B edge restores exactly instead of staying
// cut.
test('saved history restores a same-key live target edge when its idea linkage is unchanged', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await installIntentStore(page);
  const {intent, ids} = await driveTaskLinkedDraft(page);
  const [ideaA, ideaB] = ids;
  assert.ok(ideaA && ideaB);
  const pre = await currentRevision(page);
  const preTasks = pre.tasks as HistoryTask[];
  const aSeed = preTasks.find((task) => task.idea_ids.includes(ideaA));
  const bSeed = preTasks.find((task) => task.idea_ids.includes(ideaB));
  assert.ok(aSeed && bSeed);
  // The old graph records A depending on B under B's stable key; the head
  // keeps that key bound to the same ideas.
  const h1Tasks = preTasks.map((task) => ({...task}));
  const recordedA = h1Tasks.find((task) => task.key === aSeed.key);
  assert.ok(recordedA);
  recordedA.depends_on = [bSeed.key];
  const h1 = {...pre, tasks: h1Tasks};
  const preID = String(pre.id);
  const h2 = {
    ...pre,
    id: taggedID(preID, freshMark(preID, '4')),
    predecessor: preID,
    ideas: (pre.ideas as HistoryIdea[]).map((idea) => (idea.id === ideaA ? {...idea, decision: 'decline'} : idea)),
    tasks: preTasks.filter((task) => !task.idea_ids.includes(ideaA)),
  };
  await seedCurrentHead(page, h2);
  await seedSavedHistory(page, intent, [h2, h1]);
  await resumeWithoutShelf(page, intent);
  const gate = page.getByRole('group', {name: 'Saved history recovery'});
  const reaccept = gate.getByRole('button', {
    name: `Re-accept declined idea from saved history ${ideaA}`,
    exact: true,
  });
  await reaccept.waitFor();
  await reaccept.click();
  await updateProposal(page);
  const restored = await lastPostedBody(page, 'draft');
  assert.equal(restored.predecessor, h2.id, 'history recovery must draft onto the current head');
  const revived = restored.tasks as HistoryTask[];
  const revivedA = revived.find((task) => task.key === aSeed.key);
  assert.ok(revivedA, 'the recorded A work must still return');
  assert.deepEqual(
    revivedA.depends_on || [],
    [bSeed.key],
    'the A→B edge across unchanged lineage must restore exactly: no over-refusal'
  );
  const live = new Set(revived.map((task) => task.key));
  for (const task of revived)
    for (const dep of task.depends_on || []) assert.ok(live.has(dep), `no invented edges: ${dep} must be live work`);
  checkCanonical('history-samekey-unchanged-restored', restored);
});

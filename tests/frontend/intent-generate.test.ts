import {test, afterAll, afterEach} from 'bun:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {SodaRequestError} from '../../frontend/spaces/sodaspaces-api.js';
import type {SodaIntent} from '../../frontend/spaces/sodaspaces-intent.js';

// Bun shares
// imported Lit classes across files, so the JSDOM window owns the
// HTMLElement base and the custom-element registry in one realm.
const dom = typeof window === 'undefined' ? new JSDOM('', {url: 'https://soda.test/'}) : {window};
const forcedGlobals: Record<string, unknown> = {
  window: dom.window,
  document: dom.window.document,
  customElements: dom.window.customElements,
  HTMLElement: dom.window.HTMLElement,
  HTMLInputElement: dom.window.HTMLInputElement,
  HTMLTextAreaElement: dom.window.HTMLTextAreaElement,
  HTMLSelectElement: dom.window.HTMLSelectElement,
  Element: dom.window.Element,
  Node: dom.window.Node,
  Event: dom.window.Event,
  CustomEvent: dom.window.CustomEvent,
};
const savedGlobals = new Map<string, unknown>();
const missingGlobals = new Set<string>();
for (const [key, value] of Object.entries(forcedGlobals)) {
  if (key in globalThis) savedGlobals.set(key, (globalThis as Record<string, unknown>)[key]);
  else missingGlobals.add(key);
  (globalThis as Record<string, unknown>)[key] = value;
}
afterAll(() => {
  for (const [key, value] of savedGlobals) (globalThis as Record<string, unknown>)[key] = value;
  for (const key of missingGlobals) delete (globalThis as Record<string, unknown>)[key];
});

const {SodaIntent: SodaIntentElement} = await import('../../frontend/spaces/sodaspaces-intent.js');
assert.ok(SodaIntentElement);
assert.equal(
  customElements.get('soda-intent'),
  SodaIntentElement,
  'fixture registry must own the imported constructor'
);
const {configureProject} = await import('../../frontend/spaces/sodaspaces-workspace-setup.js');
const elements: SodaIntent[] = [];
afterEach(() => {
  for (const el of elements.splice(0)) el.remove();
});
function createIntent(): SodaIntent {
  const el = document.createElement('soda-intent');
  assert(el instanceof SodaIntentElement, 'fixture must construct the real registered component');
  elements.push(el);
  return el;
}

const intentID = 'd000000000000000000000000';
const draftID = 'd222222222222222222222222';
const acceptID = 'dbbbbbbbbbbbbbbbbbbbbbbbb';
const draft = {
  id: draftID,
  intent: intentID,
  repository: '0',
  objective: 'Build a leave approval workflow so managers can review time-off requests',
  state: 'draft',
  max_issues: 10,
  planning_calls: 8,
  ideas: [
    {
      id: 'd555555555555555555555555',
      description: 'Capture leave requests',
      consequence: 'Requests can be recorded',
      assumption: false,
      decision: 'accept',
    },
  ],
  tasks: [
    {
      key: 'd700000000000000000000001',
      objective: 'Capture leave requests',
      outcomes: ['Requests can be recorded'],
      idea_ids: ['d555555555555555555555555'],
    },
  ],
};
const accepted = {...draft, id: acceptID, state: 'accepted', repository: '7'};

interface Call {
  path: string;
  body: unknown;
}
function stubApi(routes: Record<string, unknown>, calls: Call[]) {
  return async (path: string, body?: Record<string, unknown>): Promise<unknown> => {
    calls.push({path, body});
    if (path === '/api/factory/intents' && body !== undefined) {
      const saved = {...accepted, ...body};
      routes[`/api/factory/intents/${intentID}/generation`] = {state: 'pending', revision_id: saved.id};
      return structuredClone(saved);
    }
    if (!(path in routes)) throw new Error('unstubbed ' + path);
    const value = routes[path];
    if (value instanceof Error) throw value;
    return structuredClone(value);
  };
}

function baseRoutes(generate: unknown): Record<string, unknown> {
  return {
    '/api/factory/departments': [],
    '/api/factory/intents?limit=20': {intents: []},
    [`/api/factory/intents/${intentID}`]: draft,
    [`/api/factory/intents/${intentID}/review`]: {intent: intentID, head_revision: draftID, status: 'missing'},
    [`/api/factory/intents/${intentID}/generation`]: {state: 'pending', revision_id: acceptID},
    [`/api/factory/intents/${intentID}/generate`]: generate,
  };
}

async function settle(el: SodaIntent) {
  await el.updateComplete;
  for (let i = 0; i < 200 && el.busy; i++) {
    await el.updateComplete;
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  await el.updateComplete;
}

async function acceptThrough(el: SodaIntent): Promise<void> {
  const notices: string[] = [];
  const navigation: string[] = [];
  const reading = {
    repositoryResult: {items: [{id: '7'}]},
    repositoryChoice: '7',
    repositoryBusy: false,
    stale: false,
    canRestore: true,
    setup: 'repositories',
  } as Parameters<typeof configureProject>[0];
  const actions = {
    setRepositoryBusy: () => {},
    setRepositoryError: (message: string) => assert.fail(message),
    isStale: () => false,
    isActiveSurface: () => true,
    notifyFactory: (message: string) => notices.push(message),
    setSetup: (value: string) => assert.equal(value, 'configure'),
    setSetupCommitted: (value: boolean) => assert.equal(value, true),
    showManagement: (repository: string) => navigation.push(repository),
  } as unknown as Parameters<typeof configureProject>[1];
  await configureProject(reading, actions, {
    updateComplete: el.updateComplete,
    querySelector: <E extends Element>() => el as unknown as E,
  });
  await settle(el);
  assert.deepEqual(notices, [el.generationNotice], 'workspace must forward the truthful notice');
  assert.notEqual(notices[0], '');
  assert.deepEqual(navigation, ['7']);
}

function savedRevisionID(calls: Call[]): string {
  const saved = calls.find((c) => c.path === '/api/factory/intents' && c.body);
  assert.ok(saved);
  return (saved.body as {id: string}).id;
}

function generatePosts(calls: Call[]): Call[] {
  return calls.filter((c) => c.path === `/api/factory/intents/${intentID}/generate`);
}

function refreshButton(el: SodaIntent): HTMLButtonElement | undefined {
  return Array.from(el.shadowRoot!.querySelectorAll('button')).find(
    (b) => b.textContent?.trim() === 'Refresh status'
  ) as HTMLButtonElement | undefined;
}

test('generate 202 wait reaches workspace notification with one POST and usable refresh', async () => {
  const calls: Call[] = [];
  const routes = baseRoutes({
    launched: false,
    waits: [{reason: 'already_assigned', detail: 'recorded decomposition plan is still running'}],
  });
  const el = createIntent();
  el.api = stubApi(routes, calls);
  el.actor = '1';
  el.request = intentID;
  el.active = true;
  document.body.appendChild(el);
  await settle(el);
  assert.equal(el.revision?.id, draftID);

  await acceptThrough(el);
  assert.equal(
    el.generationNotice,
    'Factory work is waiting: recorded decomposition plan is still running No native work was started.'
  );
  assert.equal(generatePosts(calls).length, 1);
  assert.equal(el.revision?.id, savedRevisionID(calls));
  assert.equal(el.revision?.state, 'accepted');
  assert.equal(el.error, '');
  assert.equal(el.busy, false);

  await settle(el);
  assert.equal(generatePosts(calls).length, 1, 'wait must not retry the generate POST');
  const refresh = refreshButton(el);
  assert.ok(refresh, 'refresh control missing after wait');
  assert.equal(refresh.disabled, false);
  const beforeRefresh = calls.length;
  refresh.click();
  await settle(el);
  assert.equal(generatePosts(calls).length, 1, 'refresh must only re-read generation status');
  assert.ok(calls.slice(beforeRefresh).some((c) => c.path.endsWith('/generation') && c.body === undefined));
  assert.ok(
    calls.slice(beforeRefresh).every((c) => c.body === undefined),
    'refresh must be read-only'
  );
  assert.ok(
    calls.some((c) => c.path === `/api/factory/intents/${intentID}/generation` && c.body === undefined),
    'refresh must issue the generation status read'
  );
  assert.equal(
    el.generationNotice,
    'Factory work is waiting: recorded decomposition plan is still running No native work was started.'
  );
});

test('generate stale_planning_choice refusal preserves direction with usable refresh', async () => {
  const calls: Call[] = [];
  const routes = baseRoutes(new SodaRequestError(412, 'stale_planning_choice'));
  const el = createIntent();
  el.api = stubApi(routes, calls);
  el.actor = '1';
  el.request = intentID;
  el.active = true;
  document.body.appendChild(el);
  await settle(el);

  await acceptThrough(el);
  assert.equal(
    el.generationNotice,
    'Your product direction is saved. The direction changed while Factory was planning; revise it and plan again before retrying.'
  );
  assert.equal(generatePosts(calls).length, 1);
  assert.equal(el.revision?.id, savedRevisionID(calls));
  assert.equal(el.revision?.state, 'accepted');
  assert.equal(el.error, '');

  await settle(el);
  assert.equal(generatePosts(calls).length, 1, 'refusal must not retry the generate POST');
  const refresh = refreshButton(el);
  assert.ok(refresh, 'refresh control missing after refusal');
  assert.equal(refresh.disabled, false);
  const beforeRefresh = calls.length;
  refresh.click();
  await settle(el);
  assert.equal(generatePosts(calls).length, 1, 'refresh must only re-read generation status');
  assert.ok(calls.slice(beforeRefresh).some((c) => c.path.endsWith('/generation') && c.body === undefined));
  assert.ok(
    calls.slice(beforeRefresh).every((c) => c.body === undefined),
    'refresh must be read-only'
  );
});

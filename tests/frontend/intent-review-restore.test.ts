import test, {after} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {join} from 'node:path';
import {JSDOM} from 'jsdom';
import {savedReviewRead} from '../../frontend/spaces/sodaspaces-intent-response.js';
import {hflowCurrent} from './fixtures/saved-review';
import type {SodaIntent} from '../../frontend/spaces/sodaspaces-intent.js';

const dom = new JSDOM('', {url: 'https://soda.test/'});
// Dispatch and instanceof checks must use one DOM realm: force the fixture
// globals over any host-provided ones, then restore after this file.
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
after(() => {
  for (const [key, value] of savedGlobals) (globalThis as Record<string, unknown>)[key] = value;
  for (const key of missingGlobals) delete (globalThis as Record<string, unknown>)[key];
});

const {
  adoptReview,
  restoredStale,
  SodaIntent: SodaIntentElement,
} = await import('../../frontend/spaces/sodaspaces-intent.js');
const {SodaRequestError} = await import('../../frontend/spaces/sodaspaces-api.js');
assert.ok(SodaIntentElement);

test('adoptReview keeps the stored presentation with basis/result identity', () => {
  const adopted = adoptReview(savedReviewRead(JSON.parse(hflowCurrent)));
  assert.ok(adopted.exploration);
  assert.deepEqual(adopted.restored, {
    basis: 'd111111111111111111111111',
    result: 'd222222222222222222222222',
  });
  assert.equal(adopted.exploration.content.alternatives?.length, 2);
  assert.equal(adopted.exploration.recommendation?.milestones.length, 4);
});

test('adoptReview leaves missing reads empty', () => {
  assert.deepEqual(adoptReview(undefined), {exploration: undefined, restored: undefined});
  assert.deepEqual(
    adoptReview(
      savedReviewRead({
        intent: 'd000000000000000000000000',
        head_revision: 'd333333333333333333333333',
        status: 'missing',
      })
    ),
    {exploration: undefined, restored: undefined}
  );
});

test('restoredStale marks moved heads and local edits only', () => {
  assert.equal(restoredStale(false, 'd222222222222222222222222', 'd222222222222222222222222'), false);
  assert.equal(restoredStale(false, 'd222222222222222222222222', 'd333333333333333333333333'), true);
  assert.equal(restoredStale(true, 'd222222222222222222222222', 'd222222222222222222222222'), true);
});

const headRevision = {
  id: 'd222222222222222222222222',
  intent: 'd000000000000000000000000',
  repository: '0',
  objective: 'Time-off requests my manager can approve',
  state: 'draft',
  max_issues: 10,
  planning_calls: 8,
  ideas: [
    {
      id: 'd444444444444444444444444',
      description: 'Everyone can see each request',
      consequence: 'Dates are exposed',
      assumption: true,
      decision: 'pending',
    },
  ],
  tasks: [],
};

function stubApi(routes: Record<string, unknown>, calls: string[]) {
  return async (path: string): Promise<unknown> => {
    calls.push(path);
    if (!(path in routes)) throw new Error('unstubbed ' + path);
    const value = routes[path];
    if (value instanceof Error) throw value;
    return structuredClone(value);
  };
}

function baseRoutes(review: unknown): Record<string, unknown> {
  return {
    '/api/factory/departments': [],
    '/api/factory/intents?limit=20': {intents: []},
    '/api/factory/intents/d000000000000000000000000': headRevision,
    '/api/factory/intents/d000000000000000000000000/review': review,
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

async function mount(api: (path: string) => Promise<unknown>, request: string): Promise<SodaIntent> {
  const el = document.createElement('soda-intent') as SodaIntent;
  el.api = api;
  el.actor = '1';
  el.request = request;
  el.active = true;
  document.body.appendChild(el);
  await settle(el);
  return el;
}

function rendered(el: SodaIntent): string {
  return el.shadowRoot?.textContent || '';
}

test('fresh resume restores the illustrated flow without Explore', async () => {
  const calls: string[] = [];
  const el = mount(stubApi(baseRoutes(JSON.parse(hflowCurrent)), calls), 'd000000000000000000000000');
  const text = rendered(await el);
  assert.match(text, /Here is how it would work/);
  assert.match(text, /Everyone can see each request/);
  assert.match(text, /If Approved/);
  assert.doesNotMatch(text, /out of date/);
  assert.ok(!calls.some((path) => path.endsWith('/explore')), 'resume explored instead of restoring');
  assert.ok(calls.includes('/api/factory/intents/d000000000000000000000000/review'));
});

// The G07 source join: TestSavedReviewJoinCaptureRound records one live
// Explore→save→read round as revision.json/review.json (API response bodies
// only). This case replays those exact bytes through the production
// component, but only when explicitly requested: set SODA_G07_JOIN_DIR to
// the capture directory (absolute or relative to the working directory).
// Unrequested it stays skipped and is never joined evidence; requested,
// missing or malformed files fail closed. Qualification sets the input and
// executes all 13 cases with 0 skipped.
const joinDir = process.env.SODA_G07_JOIN_DIR;
test(
  'live round restores the captured flow, recommendation and verdicts',
  {skip: joinDir ? false : 'unrequested: set SODA_G07_JOIN_DIR to the capture directory'},
  async () => {
    assert.ok(joinDir);
    let revision: unknown;
    let rawRead: unknown;
    try {
      revision = JSON.parse(readFileSync(join(joinDir, 'revision.json'), 'utf8'));
      rawRead = JSON.parse(readFileSync(join(joinDir, 'review.json'), 'utf8'));
    } catch {
      assert.fail(
        `G07 join files missing or malformed under ${joinDir}: run TestSavedReviewJoinCaptureRound in internal/web/api to regenerate revision.json/review.json`
      );
    }
    const read = savedReviewRead(rawRead);
    const review = read.review;
    assert.ok(review);
    const recommendation = review.recommendation;
    assert.ok(recommendation);
    assert.ok(review.settled.length + review.challenges.length > 0);
    const intent = read.intent;
    const routes: Record<string, unknown> = {
      '/api/factory/departments': [],
      '/api/factory/intents?limit=20': {intents: []},
      [`/api/factory/intents/${intent}`]: revision,
      [`/api/factory/intents/${intent}/review`]: rawRead,
    };
    const calls: string[] = [];
    const el = await mount(stubApi(routes, calls), intent);
    const text = rendered(el);
    assert.match(text, /Here is how it would work/);
    let at = -1;
    for (const milestone of recommendation.milestones) {
      const next = text.indexOf(milestone.name, at + 1);
      assert.ok(next > at, `milestone out of order: ${milestone.name}`);
      at = next;
    }
    for (const step of recommendation.alternative.steps) {
      assert.ok(text.includes(step.action), `step missing: ${step.action}`);
    }
    assert.deepEqual(el.exploration?.settled, review.settled);
    assert.equal(el.exploration?.challenges.length, review.challenges.length);
    assert.ok(!calls.some((path) => path.endsWith('/explore')), 'resume explored instead of restoring');
    for (const path of calls) assert.ok(path in routes, `effect replay: ${path}`);
  }
);

test('moved head keeps the review with an out-of-date notice', async () => {
  const moved = {...headRevision, id: 'd333333333333333333333333'};
  const read = JSON.parse(hflowCurrent) as Record<string, unknown>;
  read.head_revision = 'd333333333333333333333333';
  read.status = 'stale';
  const routes = baseRoutes(read);
  routes['/api/factory/intents/d000000000000000000000000'] = moved;
  const el = await mount(stubApi(routes, []), 'd000000000000000000000000');
  const text = rendered(el);
  assert.match(text, /out of date/);
  assert.match(text, /Here is how it would work/);
});

test('missing review renders no flow and no error', async () => {
  const el = await mount(
    stubApi(
      baseRoutes({
        intent: 'd000000000000000000000000',
        head_revision: 'd222222222222222222222222',
        status: 'missing',
      }),
      []
    ),
    'd000000000000000000000000'
  );
  assert.doesNotMatch(rendered(el), /Here is how it would work/);
  assert.equal(el.error, '');
  assert.equal(el.revision?.id, 'd222222222222222222222222');
});

test('absent review endpoint degrades without error', async () => {
  const routes = baseRoutes(JSON.parse(hflowCurrent));
  routes['/api/factory/intents/d000000000000000000000000/review'] = new SodaRequestError(404);
  const el = await mount(stubApi(routes, []), 'd000000000000000000000000');
  assert.doesNotMatch(rendered(el), /Here is how it would work/);
  assert.equal(el.error, '');
  assert.equal(el.revision?.id, 'd222222222222222222222222');
});

test('failed review load preserves the prior revision and unsaved correction', async () => {
  const calls: string[] = [];
  const el = await mount(stubApi(baseRoutes(JSON.parse(hflowCurrent)), calls), 'd000000000000000000000000');
  const field = el.shadowRoot?.querySelector<HTMLTextAreaElement>('textarea[aria-label^="Explain a correction"]');
  assert.ok(field);
  field.value = 'Only direct managers approve';
  field.dispatchEvent(new Event('input', {bubbles: true, composed: true}));
  await el.updateComplete;
  assert.equal(el.dirty, true);
  const routes = baseRoutes(JSON.parse(hflowCurrent));
  routes['/api/factory/intents/d000000000000000000000001'] = {
    ...headRevision,
    id: 'd111111111111111111111112',
    intent: 'd000000000000000000000001',
  };
  routes['/api/factory/intents/d000000000000000000000001/review'] = new SodaRequestError(500);
  el.api = stubApi(routes, calls);
  el.request = 'd000000000000000000000001';
  await settle(el);
  assert.equal(el.revision?.intent, 'd000000000000000000000000');
  assert.match(el.error, /kept/);
  const kept = el.shadowRoot?.querySelector<HTMLTextAreaElement>('textarea[aria-label^="Explain a correction"]');
  assert.equal(kept?.value, 'Only direct managers approve');
});

async function poisonedReviewKeepsEdits(name: string, poison: unknown) {
  const calls: string[] = [];
  const el = await mount(stubApi(baseRoutes(JSON.parse(hflowCurrent)), calls), 'd000000000000000000000000');
  const field = el.shadowRoot?.querySelector<HTMLTextAreaElement>('textarea[aria-label^="Explain a correction"]');
  assert.ok(field, name);
  field.value = 'Only direct managers approve';
  field.dispatchEvent(new Event('input', {bubbles: true, composed: true}));
  await el.updateComplete;
  assert.equal(el.dirty, true, name);
  const routes = baseRoutes(JSON.parse(hflowCurrent));
  routes['/api/factory/intents/d000000000000000000000001'] = {
    ...headRevision,
    id: 'd111111111111111111111112',
    intent: 'd000000000000000000000001',
  };
  routes['/api/factory/intents/d000000000000000000000001/review'] = poison;
  el.api = stubApi(routes, calls);
  el.request = 'd000000000000000000000001';
  await settle(el);
  assert.equal(el.revision?.intent, 'd000000000000000000000000', name);
  assert.match(el.error, /kept/, name);
  const kept = el.shadowRoot?.querySelector<HTMLTextAreaElement>('textarea[aria-label^="Explain a correction"]');
  assert.equal(kept?.value, 'Only direct managers approve', name);
}

test('poisoned review responses fail the resume with edits kept', async () => {
  await poisonedReviewKeepsEdits('network', new TypeError('network down'));
  await poisonedReviewKeepsEdits('parse', {
    intent: 'd000000000000000000000001',
    head_revision: 'd111111111111111111111112',
    status: 'current',
  });
  await poisonedReviewKeepsEdits('wrong-intent', JSON.parse(hflowCurrent));
  const mixedHead = JSON.parse(hflowCurrent) as Record<string, unknown>;
  const mixedReview = mixedHead.review as Record<string, unknown>;
  mixedHead.intent = 'd000000000000000000000001';
  mixedHead.head_revision = 'd333333333333333333333333';
  mixedHead.status = 'stale';
  mixedReview.intent = 'd000000000000000000000001';
  (mixedReview.content as Record<string, unknown>).intent = 'd000000000000000000000001';
  (mixedReview.recommendation as Record<string, unknown>).intent = 'd000000000000000000000001';
  (mixedReview.consistency as Record<string, unknown>).intent = 'd000000000000000000000001';
  await poisonedReviewKeepsEdits('mixed-head', mixedHead);
});

test('superseded resume never applies stale review data', async () => {
  let releaseReview!: () => void;
  const reviewGate = new Promise<void>((resolve) => {
    releaseReview = resolve;
  });
  const reviewB = {
    intent: 'd000000000000000000000001',
    head_revision: 'd888888888888888888888888',
    status: 'missing',
  };
  let reviewRequested = false;
  const api = async (path: string): Promise<unknown> => {
    if (path === '/api/factory/intents/d000000000000000000000000/review') {
      reviewRequested = true;
      await reviewGate;
    }
    const routes: Record<string, unknown> = {
      '/api/factory/departments': [],
      '/api/factory/intents?limit=20': {intents: []},
      '/api/factory/intents/d000000000000000000000000': headRevision,
      '/api/factory/intents/d000000000000000000000000/review': JSON.parse(hflowCurrent),
      '/api/factory/intents/d000000000000000000000001': {
        ...headRevision,
        id: 'd888888888888888888888888',
        intent: 'd000000000000000000000001',
        ideas: [],
      },
      '/api/factory/intents/d000000000000000000000001/review': reviewB,
    };
    if (!(path in routes)) throw new Error('unstubbed ' + path);
    return structuredClone(routes[path]);
  };
  const el = document.createElement('soda-intent') as SodaIntent;
  el.api = api;
  el.actor = '1';
  el.request = 'd000000000000000000000000';
  el.active = true;
  document.body.appendChild(el);
  for (let i = 0; i < 200 && !reviewRequested; i++) {
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  assert.ok(reviewRequested, 'supersede landed before the review fetch started');
  el.request = 'd000000000000000000000001';
  await settle(el);
  releaseReview();
  await settle(el);
  assert.equal(el.revision?.intent, 'd000000000000000000000001');
  assert.equal(el.exploration, undefined);
  assert.equal(el.error, '');
  assert.doesNotMatch(rendered(el), /Here is how it would work/);
});

test('gated account switch discards the old actor response', async () => {
  let releaseReview!: () => void;
  const reviewGate = new Promise<void>((resolve) => {
    releaseReview = resolve;
  });
  let reviewRequested = false;
  const api = async (path: string): Promise<unknown> => {
    if (path === '/api/factory/intents/d000000000000000000000000/review') {
      reviewRequested = true;
      await reviewGate;
    }
    const routes = baseRoutes(JSON.parse(hflowCurrent));
    if (!(path in routes)) throw new Error('unstubbed ' + path);
    return structuredClone(routes[path]);
  };
  const el = document.createElement('soda-intent') as SodaIntent;
  el.api = api;
  el.actor = '1';
  el.request = 'd000000000000000000000000';
  el.active = true;
  document.body.appendChild(el);
  for (let i = 0; i < 200 && !reviewRequested; i++) {
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  assert.ok(reviewRequested, 'switch landed before the review fetch started');
  el.actor = '2';
  await el.updateComplete;
  releaseReview();
  await settle(el);
  assert.equal(el.exploration, undefined);
  assert.equal(el.revision, undefined);
  assert.equal(el.error, '');
  assert.equal(el.busy, false);
  assert.doesNotMatch(rendered(el), /Here is how it would work/);
});

test('account switch clears the restored review', async () => {
  const el = await mount(stubApi(baseRoutes(JSON.parse(hflowCurrent)), []), 'd000000000000000000000000');
  assert.match(rendered(el), /Here is how it would work/);
  el.actor = '2';
  await el.updateComplete;
  assert.equal(el.exploration, undefined);
  assert.doesNotMatch(rendered(el), /Here is how it would work/);
  assert.equal(el.revision?.id, 'd222222222222222222222222');
});

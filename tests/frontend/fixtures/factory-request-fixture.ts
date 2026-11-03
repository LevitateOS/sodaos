import {explorationKey} from '@factory/product/frontend/factory/request/exploration-record.js';
import {planningKey} from '@factory/product/frontend/factory/request/planning-record.js';
import {generationKey} from '@factory/product/frontend/factory/request/generation-record.js';
import {saveKey} from '@factory/product/frontend/factory/request/save-record.js';
import {SodaRequestError} from '@factory/product/frontend/factory/api/models.js';
import {SodaIntent} from '@factory/product/frontend/factory/request/request.js';
import {
  declaredGeneration,
  declaredExploration,
  requestStoryContext,
} from '@factory/product/frontend/storybook/request-fixtures.js';
import type {IntentRevision} from '@factory/product/frontend/factory/api/request-response.js';
import type {IntentAPI} from '@factory/product/frontend/factory/request/request.js';
import {readRequestDraft, requestDraftKey} from '@factory/product/frontend/factory/request/draft.js';
import type {RequestStoryState} from '@factory/product/frontend/storybook/request-fixtures.js';

function mount(state: RequestStoryState, profile = 'primary') {
  const request = (current = new SodaIntent());
  const context = requestStoryContext(state, profile);
  const original = context.api;
  calls.length = 0;
  Object.assign(request, {
    ...context,
    api: async (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => {
      calls.push({path, ...(body ? {body} : {}), api: 'draft'});
      return original(path, body, signal);
    },
  });
  document.querySelector('main')!.replaceChildren(request);
}

type LifetimeKind = 'save' | 'resume' | 'explore' | 'plan' | 'generate' | 'poll';
let current: SodaIntent;
let binding: ReturnType<typeof requestStoryContext>;
const calls: {path: string; body?: Record<string, unknown>; api: string}[] = [];
const held: {signal: AbortSignal | undefined; release: () => void}[] = [];
const events: string[] = [];
let failures = false;
let capturedAPI: IntentAPI;
const secondIntent = 'd' + '4'.repeat(24);
let lifetimeSerial = 0;
async function lifetime(kind: LifetimeKind) {
  const context = requestStoryContext('review', 'lifetime');
  let saved = (await context.api('/api/factory/intents/' + context.request)) as IntentRevision;
  const idea = saved.ideas[0]!;
  saved = {
    ...saved,
    repository: ['plan', 'generate', 'poll'].includes(kind) ? '7' : '0',
    state: kind === 'generate' || kind === 'poll' ? 'accepted' : 'draft',
    ideas: [{...idea, decision: 'accept', provenance: 'human', actor: context.actor}],
    tasks: [
      {
        key: 'd' + '5'.repeat(24),
        objective: 'Track release tasks.',
        outcomes: ['Tasks are visible.'],
        idea_ids: [idea.id],
      },
    ],
  };
  const proposedIdeas = saved.ideas,
    proposedTasks = saved.tasks;
  calls.length = held.length = events.length = 0;
  failures = false;
  let generationReads = 0;
  let paused = false;
  const peer =
    (name: string): IntentAPI =>
    async (path, body, signal) => {
      calls.push({path, ...(body ? {body: structuredClone(body)} : {}), api: name});
      const candidate =
        (kind === 'save' && path === '/api/factory/intents' && !!body) ||
        (kind === 'resume' && path === '/api/factory/intents/' + context.request && !body) ||
        (kind === 'explore' && path.endsWith('/explore') && !!body) ||
        (kind === 'plan' && path.endsWith('/plan') && !!body) ||
        (kind === 'generate' && path.endsWith('/generate') && !!body) ||
        (kind === 'poll' && path.endsWith('/generation') && ++generationReads > 1);
      const pause = candidate && !paused;
      if (pause) paused = true;
      if (pause) await new Promise<void>((release) => held.push({signal, release}));
      // Ignore cancellation deliberately: the component must refuse late results/errors.
      if (pause && failures) throw Error('Declared late failure');
      if (path === '/api/factory/identity' && !body) return {host: 'a'.repeat(32), actor: context.actor};
      if (path === '/api/factory/departments' && !body) return [];
      if (path === '/api/factory/intents' && body) {
        saved = structuredClone(body) as unknown as IntentRevision;
        return {...saved, owner: context.actor, created_unix: 1, planning_calls: 0};
      }
      if (path.endsWith('/review')) {
        const id = path.split('/').at(-2)!;
        return {intent: id, head_revision: id === secondIntent ? 'd' + '7'.repeat(24) : saved.id, status: 'missing'};
      }
      if (path.endsWith('/explore') && body)
        return {
          ...saved,
          id: 'd' + '6'.repeat(24),
          predecessor: saved.id,
          ideas: proposedIdeas,
          tasks: proposedTasks,
          owner: context.actor,
          created_unix: 1,
          planning_calls: 0,
          exploration: declaredExploration(saved),
        };
      if (path.includes('/revisions/') && path.endsWith('/plan'))
        return {intent: saved.intent, revision: saved.id, state: 'none'};
      if (path.endsWith('/plan'))
        return body
          ? {revision_id: saved.id, assignment_id: 'b'.repeat(32), run_id: 'a'.repeat(32), launch_confirmed: true}
          : {revision: saved.id, state: 'none', candidates: 0};
      if (path.includes('/revisions/') && path.endsWith('/generation'))
        return {intent_id: saved.intent, revision_id: saved.id, state: 'not_requested'};
      if (path.endsWith('/generation') || path.endsWith('/generate'))
        return body && body.revision !== saved.id
          ? Promise.reject(new SodaRequestError(412, 'stale_revision'))
          : path.endsWith('/generate')
            ? declaredGeneration(saved)
            : {intent_id: saved.intent, revision_id: saved.id, state: 'waiting'};
      if (path === '/api/factory/intents/' + secondIntent)
        return {...saved, intent: secondIntent, id: 'd' + '7'.repeat(24), objective: 'Another saved request.'};
      if (path === '/api/factory/intents/' + context.request) return structuredClone(saved);
      throw Error('Unsupported declared request');
    };
  current = new SodaIntent();
  binding = {
    ...context,
    api: peer('A'),
    request: kind === 'save' || kind === 'explore' ? '' : context.request,
    draftScope: 'declared-request:lifetime:' + kind + ':' + ++lifetimeSerial,
  };
  capturedAPI = binding.api;
  Object.assign(current, binding);
  for (const name of ['intent-saved', 'intent-resumed', 'intent-accepted', 'soda-intent-operation'])
    current.addEventListener(name, () => events.push(name));
  document.querySelector('main')!.replaceChildren(current);
  return;
}
function retire(
  kind: 'active' | 'blocked' | 'api' | 'profile' | 'actor' | 'request' | 'disconnect' | 'reset' | 'cycle'
) {
  if (kind === 'active') current.active = false;
  if (kind === 'blocked') current.blocked = true;
  if (kind === 'api')
    current.api = async (path, body) => {
      calls.push({path, ...(body ? {body} : {}), api: 'B'});
      return path.endsWith('/departments') ? [] : null;
    };
  if (kind === 'profile') current.draftScope += ':another-profile';
  if (kind === 'actor') current.actor = '9007199254740992';
  if (kind === 'request') current.request = secondIntent;
  if (kind === 'disconnect') current.remove();
  if (kind === 'reset') current.reset();
  if (kind === 'cycle') {
    current.active = false;
    current.active = true;
  }
}
const storageGet = Storage.prototype.getItem,
  storageSet = Storage.prototype.setItem;
const address = () => ({
  scope: current.draftScope,
  actor: current.actor,
  subject: current.revision?.intent || current.request || 'new',
});
const draftKey = () => requestDraftKey(address());
type SaveMode = 'lost' | 'missing' | 'ok' | 'changed' | 'owner' | 'host' | 'head' | 'obsolete' | 'denied';
let saveMode: SaveMode = 'lost';
let saveProfile = '';
let saveEntry = true;
let savePause: 'save' | 'receipt' | 'identity' | undefined;
const removeStored = Storage.prototype.removeItem;
const remoteKey = () => 'declared-save-peer:' + saveProfile;
const remote = (): Record<string, unknown> | undefined => {
  const value = storageGet.call(localStorage, remoteKey());
  return value ? (JSON.parse(value) as Record<string, unknown>) : undefined;
};
async function saveMount(profile: string, mode: SaveMode = 'lost', entry = true) {
  saveProfile = profile;
  saveMode = mode;
  saveEntry = entry;
  savePause = undefined;
  calls.length = events.length = held.length = 0;
  const context = requestStoryContext('review', 'save-peer');
  const initial = (await context.api('/api/factory/intents/' + context.request)) as IntentRevision;
  const first = initial.ideas[0]!,
    second = {...first, id: 'd' + '4'.repeat(24), description: 'Review release tasks.'};
  initial.ideas = [first, second].map((idea) => ({
    ...idea,
    decision: 'accept',
    provenance: 'human',
    actor: context.actor,
  }));
  initial.tasks = [
    {key: 'd' + '5'.repeat(24), objective: 'Track tasks.', outcomes: ['Tasks are visible.'], idea_ids: [first.id]},
    {
      key: 'd' + '6'.repeat(24),
      objective: 'Review tasks.',
      outcomes: ['Reviews are visible.'],
      idea_ids: [second.id],
      depends_on: ['d' + '5'.repeat(24)],
    },
  ];
  const api: IntentAPI = async (path, body, signal) => {
    calls.push({path, ...(body ? {body: structuredClone(body)} : {}), api: 'save-peer'});
    const pause =
      savePause &&
      ((savePause === 'save' && path === '/api/factory/intents' && body) ||
        (savePause === 'receipt' && path.includes('/revisions/')) ||
        (savePause === 'identity' && path.endsWith('/identity')));
    if (pause) {
      savePause = undefined;
      await new Promise<void>((release) => held.push({signal, release}));
    }
    if (path.endsWith('/identity')) {
      if (saveMode === 'denied') throw new SodaRequestError(401, 'unauthorized');
      return {
        host: (saveMode === 'host' ? 'b' : 'a').repeat(32),
        actor: saveMode === 'owner' ? '9007199254740992' : context.actor,
      };
    }
    if (path === '/api/factory/departments') return [];
    if (path === '/api/factory/intents' && body) {
      if (saveMode !== 'missing')
        storageSet.call(
          localStorage,
          remoteKey(),
          JSON.stringify({...body, owner: context.actor, created_unix: 42, planning_calls: 0})
        );
      if (saveMode === 'lost' || saveMode === 'missing') throw Error('Declared lost save reply');
      return remote();
    }
    const saved = remote();
    if (path.includes('/revisions/')) {
      if (!saved || path !== '/api/factory/intents/' + saved.intent + '/revisions/' + saved.id)
        throw new SodaRequestError(404, 'intent_not_found');
      return saveMode === 'changed' ? {...saved, objective: 'A changed receipt body.'} : saved;
    }
    const head = saved ?? initial;
    if (path.endsWith('/review'))
      return {
        intent: head.intent,
        head_revision: saveMode === 'head' || saveMode === 'obsolete' ? 'd' + '8'.repeat(24) : head.id,
        status: 'missing',
      };
    if (path.endsWith('/history')) return {revisions: []};
    if (path === '/api/factory/intents/' + head.intent)
      return saveMode === 'head' || saveMode === 'obsolete'
        ? {...head, id: 'd' + '8'.repeat(24), objective: 'A later saved head.'}
        : head;
    if (/^\/api\/factory\/intents\/d[0-9a-f]{24}$/.test(path)) throw new SodaRequestError(404, 'intent_not_found');
    throw new SodaRequestError(503, 'planning_unavailable');
  };
  current = new SodaIntent();
  binding = {
    ...context,
    api,
    draftScope: 'declared-save:' + profile,
    request: entry ? '' : String(remote()?.intent ?? context.request),
  };
  capturedAPI = api;
  Object.assign(current, binding);
  for (const name of ['intent-saved', 'intent-accepted']) current.addEventListener(name, () => events.push(name));
  document.querySelector('main')!.replaceChildren(current);
  history.replaceState(null, '', '?save=' + encodeURIComponent(profile) + '&entry=' + entry);
}
function saveAddress() {
  return {
    scope: 'declared-save:' + saveProfile,
    actor: '9007199254740993',
    subject: saveEntry ? 'new' : 'd' + '1'.repeat(24),
  };
}
type GenerationMode =
  | 'lost'
  | 'missing'
  | 'ok'
  | 'pending'
  | 'committed'
  | 'refused'
  | 'cancelled'
  | 'owner'
  | 'host'
  | 'denied'
  | 'operation'
  | 'revision'
  | 'authorization'
  | 'unavailable';
let generationMode: GenerationMode = 'lost',
  generationProfile = '',
  generationAdvanced = false;
let generationPause: 'receipt' | 'post' | 'identity' | undefined;
const generationAddress = () => ({
  scope: 'declared-generation:' + generationProfile,
  actor: '9007199254740993',
  subject: 'd' + '1'.repeat(24),
});
const generationRemoteKey = () => 'declared-generation-peer:' + generationProfile;
async function generationMount(profile: string, mode: GenerationMode = 'lost', advanced = false) {
  generationMode = mode;
  generationProfile = profile;
  generationAdvanced = advanced;
  generationPause = undefined;
  calls.length = events.length = held.length = 0;
  const context = requestStoryContext('review', 'generation-peer');
  const source = (await context.api('/api/factory/intents/' + context.request)) as IntentRevision;
  const selected: IntentRevision = {
    ...source,
    repository: '7',
    state: 'accepted',
    ideas: source.ideas.map((idea) => ({...idea, decision: 'accept', provenance: 'human', actor: context.actor})),
  };
  const result = () => {
    const raw = storageGet.call(localStorage, generationRemoteKey());
    if (!raw) return {intent_id: selected.intent, revision_id: selected.id, state: 'not_requested'};
    const record = JSON.parse(raw) as Record<string, unknown>;
    if (generationMode === 'operation') record.operation_id = 'intent-' + 'd' + '9'.repeat(24);
    if (generationMode === 'revision') record.revision_id = 'd' + '9'.repeat(24);
    if (generationMode === 'authorization') record.authorization_revision = selected.id + ':' + 'b'.repeat(64);
    if (['pending', 'committed', 'refused', 'cancelled'].includes(generationMode)) record.state = generationMode;
    return record;
  };
  const api: IntentAPI = async (path, body, signal) => {
    calls.push({path, ...(body ? {body: structuredClone(body)} : {}), api: 'generation-peer'});
    const receiptPath = '/api/factory/intents/' + selected.intent + '/revisions/' + selected.id + '/generation';
    const pause =
      generationPause &&
      ((generationPause === 'receipt' && path === receiptPath) ||
        (generationPause === 'post' && path.endsWith('/generate')) ||
        (generationPause === 'identity' && path.endsWith('/identity')));
    if (pause) {
      generationPause = undefined;
      await new Promise<void>((release) => held.push({signal, release}));
    }
    if (path === '/api/factory/identity') {
      if (generationMode === 'denied') throw new SodaRequestError(401, 'unauthorized');
      return {
        host: (generationMode === 'host' ? 'b' : 'a').repeat(32),
        actor: generationMode === 'owner' ? '9007199254740992' : context.actor,
      };
    }
    if (path === '/api/factory/departments') return [];
    const head = generationAdvanced
      ? {...selected, id: 'd' + '8'.repeat(24), objective: 'A later saved direction.'}
      : selected;
    if (path === '/api/factory/intents/' + selected.intent) return head;
    if (path.endsWith('/review')) return {intent: selected.intent, head_revision: head.id, status: 'missing'};
    if (path.endsWith('/generate')) {
      if (generationAdvanced || body?.revision !== selected.id) throw new SodaRequestError(412, 'stale_revision');
      if (generationMode !== 'missing')
        storageSet.call(localStorage, generationRemoteKey(), JSON.stringify(declaredGeneration(selected)));
      if (generationMode === 'lost' || generationMode === 'missing') throw Error('Declared lost work-creation reply');
      return result();
    }
    if (path === receiptPath) {
      if (generationMode === 'unavailable') throw Error('Declared unavailable generation read');
      return result();
    }
    if (path.endsWith('/generation')) return {...result(), revision_id: head.id};
    throw Error('Unsupported declared generation peer request');
  };
  current = new SodaIntent();
  binding = {...context, api, draftScope: generationAddress().scope};
  capturedAPI = api;
  Object.assign(current, binding);
  for (const name of ['intent-saved', 'intent-accepted']) current.addEventListener(name, () => events.push(name));
  document.querySelector('main')!.replaceChildren(current);
  history.replaceState(null, '', '?generation=' + encodeURIComponent(profile) + '&advanced=' + advanced);
}

type PlanningMode =
  | 'lost'
  | 'missing'
  | 'ok'
  | 'unconfirmed'
  | 'unavailable'
  | 'operation'
  | 'revision'
  | 'assignment'
  | 'run'
  | 'state'
  | 'start'
  | 'none'
  | 'host'
  | 'owner'
  | 'denied'
  | 'bind';
let planningMode: PlanningMode = 'lost',
  planningProfile = '',
  planningAdvanced = false;
let planningPause: 'receipt' | 'post' | undefined;
const planningAddress = () => ({
  scope: 'declared-planning:' + planningProfile,
  actor: '9007199254740993',
  subject: 'd' + '1'.repeat(24),
});
const planningRemoteKey = () => 'declared-planning-peer:' + planningProfile;
async function planningMount(profile: string, mode: PlanningMode = 'lost', advanced = false) {
  planningMode = mode;
  planningProfile = profile;
  planningAdvanced = advanced;
  planningPause = undefined;
  calls.length = events.length = held.length = 0;
  const context = requestStoryContext('review', 'planning-peer');
  const source = (await context.api('/api/factory/intents/' + context.request)) as IntentRevision;
  const selected: IntentRevision = {
    ...source,
    repository: '7',
    ideas: source.ideas.map((idea) => ({...idea, decision: 'accept', provenance: 'human', actor: context.actor})),
  };
  const result = () => {
    const raw = storageGet.call(localStorage, planningRemoteKey());
    if (!raw || planningMode === 'none') return {intent: selected.intent, revision: selected.id, state: 'none'};
    const record = JSON.parse(raw) as Record<string, unknown>;
    if (planningMode === 'operation') record.operation = 'decomposition_options';
    if (planningMode === 'revision') record.revision = 'd' + '9'.repeat(24);
    if (planningMode === 'assignment') record.assignment_id = 'c'.repeat(32);
    if (planningMode === 'run') record.run_id = 'c'.repeat(32);
    if (planningMode === 'state') record.state = 'invented';
    if (planningMode === 'start') record.launch_confirmed = true;
    return record;
  };
  const receiptPath = '/api/factory/intents/' + selected.intent + '/revisions/' + selected.id + '/plan';
  const api: IntentAPI = async (path, body, signal) => {
    calls.push({path, ...(body ? {body: structuredClone(body)} : {}), api: 'planning-peer'});
    if (
      (planningPause === 'receipt' && path === receiptPath) ||
      (planningPause === 'post' && path.endsWith('/plan') && body)
    ) {
      planningPause = undefined;
      await new Promise<void>((release) => held.push({signal, release}));
    }
    if (path === '/api/factory/identity') {
      if (planningMode === 'denied') throw new SodaRequestError(401, 'unauthorized');
      return {
        host: (planningMode === 'host' ? 'b' : 'a').repeat(32),
        actor: planningMode === 'owner' ? '9007199254740992' : context.actor,
      };
    }
    if (path === '/api/factory/departments') return [];
    const head = planningAdvanced
      ? {...selected, id: 'd' + '8'.repeat(24), objective: 'A later saved direction.'}
      : selected;
    if (path === '/api/factory/intents/' + selected.intent) return head;
    if (path.endsWith('/review')) return {intent: selected.intent, head_revision: head.id, status: 'missing'};
    if (path === receiptPath) {
      if (
        storageGet.call(localStorage, planningRemoteKey()) &&
        (planningMode === 'unavailable' || planningMode === 'unconfirmed' || planningMode === 'bind')
      )
        throw Error('Declared unavailable planning read');
      return result();
    }
    if (path.endsWith('/plan') && body) {
      if (planningAdvanced || body.revision !== selected.id) throw new SodaRequestError(412, 'stale_revision');
      if (planningMode !== 'missing')
        storageSet.call(
          localStorage,
          planningRemoteKey(),
          JSON.stringify({
            intent: selected.intent,
            revision: selected.id,
            operation: body.operation,
            assignment_id: 'a'.repeat(32),
            run_id: 'b'.repeat(32),
            state: 'pending',
            candidates: 0,
          })
        );
      if (planningMode === 'lost' || planningMode === 'missing') throw Error('Declared lost planning reply');
      if (planningMode === 'bind') fixture.planningStorage('write');
      return {
        revision_id: selected.id,
        assignment_id: 'a'.repeat(32),
        run_id: 'b'.repeat(32),
        ...(planningMode === 'unconfirmed' ? {launch_confirmed: false} : {}),
      };
    }
    if (path.endsWith('/plan')) return {revision: head.id, state: 'none', candidates: 0};
    throw Error('Unsupported declared planning peer request');
  };
  current = new SodaIntent();
  binding = {...context, api, draftScope: planningAddress().scope};
  capturedAPI = api;
  Object.assign(current, binding);
  for (const name of ['intent-saved', 'intent-accepted']) current.addEventListener(name, () => events.push(name));
  document.querySelector('main')!.replaceChildren(current);
  history.replaceState(null, '', '?planning=' + encodeURIComponent(profile) + '&advanced=' + advanced);
}

type ExplorationMode =
  | 'refused'
  | 'lost'
  | 'missing'
  | 'ok'
  | 'unavailable'
  | 'intent'
  | 'basis'
  | 'result'
  | 'predecessor'
  | 'owner'
  | 'content'
  | 'advice'
  | 'state'
  | 'host'
  | 'actor'
  | 'denied';
let explorationMode: ExplorationMode = 'lost',
  explorationProfile = '',
  explorationAdvanced = false;
let explorationPause: 'post' | 'receipt' | 'identity' | undefined;
const explorationAddress = () => ({
  scope: 'declared-exploration:' + explorationProfile,
  actor: '9007199254740993',
  subject: 'd' + '1'.repeat(24),
});
const explorationRemoteKey = () => 'declared-exploration-peer:' + explorationProfile;
const explorationRemote = (): {input?: Record<string, unknown>; result?: Record<string, unknown>} =>
  JSON.parse(storageGet.call(localStorage, explorationRemoteKey()) || '{}');
async function explorationMount(profile: string, mode: ExplorationMode = 'lost', advanced = false) {
  explorationProfile = profile;
  explorationMode = mode;
  explorationAdvanced = advanced;
  explorationPause = undefined;
  calls.length = events.length = held.length = 0;
  const context = requestStoryContext('review', 'exploration-peer'),
    source = (await context.api('/api/factory/intents/' + context.request)) as IntentRevision;
  const api: IntentAPI = async (path, body, signal) => {
    calls.push({path, ...(body ? {body: structuredClone(body)} : {}), api: 'exploration-peer'});
    const pause =
      explorationPause &&
      ((explorationPause === 'post' && path.endsWith('/explore') && body) ||
        (explorationPause === 'receipt' && path.endsWith('/exploration')) ||
        (explorationPause === 'identity' && path.endsWith('/identity')));
    if (pause) {
      explorationPause = undefined;
      await new Promise<void>((release) => held.push({signal, release}));
    }
    if (path === '/api/factory/identity') {
      if (explorationMode === 'denied') throw new SodaRequestError(401, 'unauthorized');
      return {
        host: (explorationMode === 'host' ? 'b' : 'a').repeat(32),
        actor: explorationMode === 'actor' ? '9007199254740992' : context.actor,
      };
    }
    if (path === '/api/factory/departments') return [];
    let remote = explorationRemote();
    if (path === '/api/factory/intents' && body) {
      const input = {...body, owner: context.actor, created_unix: 42, planning_calls: 0};
      storageSet.call(localStorage, explorationRemoteKey(), JSON.stringify({input}));
      return input;
    }
    const input = (remote.input || source) as unknown as IntentRevision;
    if (path.endsWith('/explore') && body) {
      if (explorationMode === 'refused') throw new SodaRequestError(400, 'invalid_proposal');
      if (explorationAdvanced || body.revision !== input.id || remote.result)
        throw new SodaRequestError(412, 'stale_revision');
      const result = {
        ...input,
        id: 'd' + '6'.repeat(24),
        predecessor: input.id,
        ideas: source.ideas,
        tasks: source.tasks,
        owner: context.actor,
        created_unix: 43,
        planning_calls: 0,
        exploration: declaredExploration(input),
      };
      if (explorationMode !== 'missing')
        storageSet.call(localStorage, explorationRemoteKey(), JSON.stringify({input: remote.input, result}));
      if (explorationMode === 'lost' || explorationMode === 'missing') throw Error('Declared lost exploration reply');
      return result;
    }
    if (path.endsWith('/exploration')) {
      if (path !== '/api/factory/intents/' + input.intent + '/revisions/' + input.id + '/exploration')
        throw new SodaRequestError(404, 'not_found');
      if (explorationMode === 'unavailable') throw Error('Declared unavailable exploration read');
      const record: Record<string, unknown> = {
        intent: input.intent,
        basis_revision: input.id,
        state: remote.result ? 'recorded' : 'none',
        ...(remote.result ? {result: remote.result} : {}),
      };
      if (explorationMode === 'intent') record.intent = 'd' + '9'.repeat(24);
      if (explorationMode === 'basis') record.basis_revision = 'd' + '9'.repeat(24);
      if (explorationMode === 'state') record.state = 'invented';
      const result = record.result as Record<string, unknown> | undefined;
      if (result) {
        if (explorationMode === 'result') result.id = input.id;
        if (explorationMode === 'predecessor') result.predecessor = 'd' + '9'.repeat(24);
        if (explorationMode === 'owner') result.owner = '9007199254740992';
        const advice = result.exploration as {content: {revision: string}; consistency?: unknown};
        if (explorationMode === 'content') advice.content.revision = result.id as string;
        if (explorationMode === 'advice') advice.consistency = {intent: input.intent, revision: result.id};
      }
      return record;
    }
    const head = explorationAdvanced
      ? {...(remote.result || input), id: 'd' + '8'.repeat(24), objective: 'A later saved exploration direction.'}
      : remote.result || input;
    if (path === '/api/factory/intents/' + source.intent) return structuredClone(head);
    if (path.endsWith('/review')) return {intent: source.intent, head_revision: head.id, status: 'missing'};
    if (path.endsWith('/history')) return {intent: source.intent, revisions: []};
    if (path.includes('/revisions/')) return remote.input;
    throw Error('Unsupported declared exploration peer request');
  };
  current = new SodaIntent();
  binding = {...context, api, draftScope: explorationAddress().scope};
  capturedAPI = api;
  Object.assign(current, binding);
  for (const name of ['intent-saved', 'intent-accepted']) current.addEventListener(name, () => events.push(name));
  document.querySelector('main')!.replaceChildren(current);
  history.replaceState(null, '', '?exploration=' + encodeURIComponent(profile) + '&advanced=' + advanced);
}
const fixture = {
  explorationMount,
  explorationMode: (mode: ExplorationMode) => {
    explorationMode = mode;
  },
  explorationAdvanced: () => {
    explorationAdvanced = true;
  },
  explorationPause: (kind: 'post' | 'receipt' | 'identity') => {
    explorationPause = kind;
  },
  explorationRaw: () => storageGet.call(localStorage, explorationKey(explorationAddress())),
  explorationSeed: (raw: string) => storageSet.call(localStorage, explorationKey(explorationAddress()), raw),
  explorationStorage(kind: 'read' | 'write' | 'archive' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-intent-exploration:'))
        throw Error('Declared exploration read denial');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (
        (kind === 'write' && key.startsWith('soda-intent-exploration:')) ||
        (kind === 'archive' && key.startsWith('soda-intent-exploration-receipt:'))
      )
        throw Error('Declared exploration write denial');
      storageSet.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-intent-exploration:'))
        throw Error('Declared exploration remove denial');
      removeStored.call(this, key);
    };
  },

  planningMount,
  planningMode: (mode: PlanningMode) => {
    planningMode = mode;
  },
  planningAdvanced: () => {
    planningAdvanced = true;
  },
  planningPause: (kind: 'receipt' | 'post') => {
    planningPause = kind;
  },
  planningRaw: () => storageGet.call(localStorage, planningKey(planningAddress())),
  planningSeed: (raw: string) => storageSet.call(localStorage, planningKey(planningAddress()), raw),
  planningStorage(kind: 'read' | 'write' | 'archive' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-intent-planning:')) throw Error('Declared planning read denial');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (
        (kind === 'write' && key.startsWith('soda-intent-planning:')) ||
        (kind === 'archive' && key.startsWith('soda-intent-planning-receipt:'))
      )
        throw Error('Declared planning write denial');
      storageSet.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-intent-planning:')) throw Error('Declared planning removal denial');
      removeStored.call(this, key);
    };
  },

  generationMount,
  generationMode: (mode: GenerationMode) => {
    generationMode = mode;
  },
  generationAdvanced: () => {
    generationAdvanced = true;
  },
  generationPause: (kind: 'receipt' | 'post' | 'identity') => {
    generationPause = kind;
  },
  generationRaw: () => storageGet.call(localStorage, generationKey(generationAddress())),
  generationSeed: (raw: string) => storageSet.call(localStorage, generationKey(generationAddress()), raw),
  generationStorage(kind: 'read' | 'write' | 'archive' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-intent-generation:')) throw Error('Declared generation read denial');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (
        (kind === 'write' && key.startsWith('soda-intent-generation:')) ||
        (kind === 'archive' && key.startsWith('soda-intent-generation-receipt:'))
      )
        throw Error('Declared generation write denial');
      storageSet.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-intent-generation:'))
        throw Error('Declared generation removal denial');
      removeStored.call(this, key);
    };
  },

  saveMount,
  saveMode: (mode: SaveMode) => {
    saveMode = mode;
  },
  savePause: (kind: 'save' | 'receipt' | 'identity') => {
    savePause = kind;
  },
  saveRaw: () => storageGet.call(localStorage, saveKey(saveAddress())),
  seedSave: (raw: string) => storageSet.call(localStorage, saveKey(saveAddress()), raw),
  saveRemote: remote,
  saveStorageFailure(kind: 'read' | 'write' | 'archive' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-intent-save:')) throw Error('Declared save read failure');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (
        (kind === 'write' && key.startsWith('soda-intent-save:')) ||
        (kind === 'archive' && key.startsWith('soda-intent-save-receipt:'))
      )
        throw Error('Declared save write failure');
      storageSet.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-intent-save:')) throw Error('Declared save removal failure');
      removeStored.call(this, key);
    };
  },
  mount,
  draftMount(state: RequestStoryState, profile: string) {
    history.replaceState(null, '', '?state=' + state + '&profile=' + encodeURIComponent(profile));
    mount(state, profile);
  },
  saved: () => storageGet.call(localStorage, draftKey()),
  draft: () => readRequestDraft(address()),
  seed: (value: string) => storageSet.call(localStorage, draftKey(), value),
  storageFailure(kind: 'read' | 'write' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-intent-draft:')) throw Error('Declared draft read failure');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && key.startsWith('soda-intent-draft:')) throw Error('Declared draft write failure');
      storageSet.call(this, key, value);
    };
  },
  lifetime,
  retire,
  calls: () => calls,
  events: () => events,
  clearEvents: () => {
    events.length = 0;
  },
  held: () => held.length,
  aborted: () => held.length > 0 && held.every((item) => item.signal?.aborted),
  release(fail = false) {
    failures = fail;
    for (const item of held) item.release();
  },
  repeat() {
    current.api = capturedAPI;
    current.active = true;
    current.blocked = false;
    current.actor = binding.actor;
    current.draftScope = binding.draftScope;
    current.request = binding.request;
  },
  accept: () => {
    void current.acceptRepository('7');
  },
  poll: () => {
    void (current as unknown as {refreshProgress: () => Promise<void>}).refreshProgress();
  },
  reattach() {
    document.querySelector('main')!.append(current);
    current.active = true;
    current.blocked = false;
  },
  snapshot: () => ({
    revision: current.revision?.id,
    intent: current.revision?.intent,
    objective: current.objective,
    error: current.error,
    busy: current.busy,
    planNotice: current.planNotice,
    generationNotice: current.generationNotice,
    generationError: current.generationError,
  }),
};
declare global {
  interface Window {
    factoryRequestFixture: typeof fixture;
  }
}
window.factoryRequestFixture = fixture;
const params = new URLSearchParams(location.search);
const state = params.get('state');
if (params.has('exploration'))
  void explorationMount(params.get('exploration')!, 'lost', params.get('advanced') === 'true');
else if (params.has('planning')) void planningMount(params.get('planning')!, 'lost', params.get('advanced') === 'true');
else if (params.has('generation'))
  void generationMount(params.get('generation')!, 'lost', params.get('advanced') === 'true');
else if (params.has('save')) void saveMount(params.get('save')!, 'lost', params.get('entry') !== 'false');
else mount(state === 'review' ? 'review' : 'entry', params.get('profile') ?? 'primary');

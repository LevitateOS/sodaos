import {SodaFactoryFeed} from '@factory/product/frontend/factory/feed/feed.js';
import type {FactoryFeedContext} from '@factory/product/frontend/factory/feed/feed.js';
import {feedStoryPeer} from '@factory/product/frontend/storybook/fixtures.js';
import type {FeedStoryState} from '@factory/product/frontend/storybook/fixtures.js';
import {feedRevisionBodyDigest} from '@factory/product/frontend/factory/feed/feed-response.js';

// Declared local read/write peer. It records dispatch and idempotent acknowledgement;
// it does not create backend work, connect a provider or execute a model.
const postID = 'a'.repeat(32),
  replyID = 'b'.repeat(32);
type Hold = 'response' | 'body' | 'reject';
const calls: {account: string; path: string; method: string; body: string}[] = [];
const signals: {account: string; path: string; signal: AbortSignal}[] = [];
const peers = new Map<string, ReturnType<typeof makePeer>>();
let storageFault = '';
const revisionWrites: string[] = [];
const reviewWrites: string[] = [];
const pendingKey = (key: string) => key.startsWith('soda-feed-retry:') || key.startsWith('soda-feed-review:');
const storageGet = Storage.prototype.getItem,
  storageSet = Storage.prototype.setItem,
  storageRemove = Storage.prototype.removeItem;
Storage.prototype.getItem = function (key) {
  if (storageFault === 'read' && pendingKey(key)) throw Error('Declared unreadable pending command');
  if (storageFault === 'draft-read' && key.startsWith('soda-feed-revision:'))
    throw Error('Declared unreadable revision writing');
  return storageGet.call(this, key);
};
Storage.prototype.setItem = function (key, value) {
  if (key.startsWith('soda-feed-retry:')) revisionWrites.push(value);
  if (key.startsWith('soda-feed-review:')) reviewWrites.push(value);
  if (storageFault === 'write' && pendingKey(key)) throw Error('Declared pending command write failure');
  return storageSet.call(this, key, value);
};
Storage.prototype.removeItem = function (key) {
  if (storageFault === 'remove' && pendingKey(key)) throw Error('Declared pending command removal failure');
  return storageRemove.call(this, key);
};
const created: string[] = [],
  revoked: string[] = [];
const create = URL.createObjectURL,
  revoke = URL.revokeObjectURL;
URL.createObjectURL = (blob) => {
  const url = create.call(URL, blob);
  created.push(url);
  return url;
};
URL.revokeObjectURL = (url) => {
  revoked.push(url);
  revoke.call(URL, url);
};
let element: SodaFactoryFeed,
  account = 'A',
  global = false;
function makePeer(owner: string) {
  // This declared peer's ledger survives browser reload independently of the
  // production localStorage record being tested; it starts no real backend work.
  const peerKey = 'declared-revision-peer:' + owner;
  const retained = JSON.parse(sessionStorage.getItem(peerKey) ?? 'null');
  const retries = new Map<string, {input: string; receipt: Record<string, unknown>}>(retained?.retries ?? []);
  const rules = new Map<string, Hold>(),
    failures = new Map<string, number>(),
    releases: (() => void)[] = [],
    commands = new Map<string, unknown>();
  let revision = retained?.revision ?? 1,
    held = 0,
    productChange = retained?.productChange ?? false,
    artifacts = false,
    reviewed = retained?.reviewed ?? false,
    unchanged = false,
    mismatch = '',
    savedBody = retained?.body ?? 'Request from ' + owner,
    canRetry = retained?.canRetry ?? true,
    loseNextRevision = false,
    loseNextReview = false,
    reviewAccess = retained?.reviewAccess ?? true,
    reviewEffects = retained?.reviewEffects ?? 0;
  const retainPeer = () =>
    sessionStorage.setItem(
      peerKey,
      JSON.stringify({
        revision,
        body: savedBody,
        canRetry,
        retries: [...retries],
        productChange,
        reviewed,
        reviewAccess,
        reviewEffects,
      })
    );
  const post = () => ({
    post_id: postID,
    repository: '7',
    revision,
    author_name: owner,
    actor: '1',
    at: 1720000000,
    body: savedBody,
    intake: 'recorded',
    judgment: 'applied',
    route: 'discussion',
    draft: 'pending',
    effect: 'pending',
    can_retry: productChange ? false : canRetry,
    ...(productChange
      ? {
          product_change: {
            intent: 'd' + '1'.repeat(24),
            accepted_revision: 'd' + '2'.repeat(24),
            ...(reviewed ? {draft_revision: 'd' + '3'.repeat(24)} : {}),
            state: reviewed ? 'draft' : 'review_required',
            can_review: reviewAccess,
          },
        }
      : {}),
  });
  const wait = () => {
    held++;
    return new Promise<void>((resolve) => releases.push(resolve));
  };
  const capture = (path: string, signal?: AbortSignal | null) => {
    if (signal) signals.push({account: owner, path, signal});
  };
  const transport = {
    generation: 'declared-feed-' + owner,
    assetBase: '/',
    async request(path: string, init?: RequestInit) {
      const method = init?.method ?? 'GET';
      calls.push({account: owner, path, method, body: String(init?.body ?? '')});
      capture(path, init?.signal);
      const mode = [...rules].find(([prefix]) => (method + ' ' + path).startsWith(prefix))?.[1];
      const status = [...failures].find(([prefix]) => (method + ' ' + path).startsWith(prefix))?.[1];
      if (status) {
        if (mode === 'response') await wait();
        return Response.json({error: {code: 'declared_denial'}}, {status});
      }
      if (path.startsWith('factory/artifacts/'))
        return new Response('Private artifact bytes', {headers: {'Content-Type': 'text/plain'}});
      let value: unknown;
      if (method === 'PUT' && path === 'factory/feed/subscription') value = {subscriber: '1', revision: 2};
      else if (method === 'POST' && path === 'factory/feed/posts') {
        const body = JSON.parse(String(init?.body));
        value = commands.get(body.idempotency_key);
        if (!value) {
          value = {...post(), post_id: 'c'.repeat(32), body: body.body};
          commands.set(body.idempotency_key, value);
        }
      } else if (method === 'POST' && path.endsWith('/retry')) {
        const input = JSON.parse(String(init?.body));
        const found = retries.get(input.command_id);
        if (found) {
          if (found.input !== String(init?.body))
            return Response.json({error: {code: 'changed_command'}}, {status: 409});
          value = found.receipt;
        } else {
          if (input.expected_revision !== revision)
            return Response.json({error: {code: 'stale_revision'}}, {status: 409});
          savedBody = input.body;
          if (!unchanged) revision++;
          value = {
            command_id: input.command_id,
            repository: input.repository,
            author: '1',
            post_id: postID,
            expected_revision: input.expected_revision,
            body_digest: await feedRevisionBodyDigest(savedBody),
            revision,
          };
          retries.set(input.command_id, {input: String(init?.body), receipt: value as Record<string, unknown>});
          retainPeer();
        }
        if (loseNextRevision) {
          loseNextRevision = false;
          throw Error('Declared committed edit reply lost');
        }
      } else if (method === 'GET' && path.includes('/retry/')) {
        const found = retries.get(path.split('/retry/')[1]!.split('?')[0]!);
        if (!found) return Response.json({error: {code: 'comment_not_found'}}, {status: 404});
        value = found.receipt;
      } else if (method === 'POST' && path.endsWith('/review')) {
        if (JSON.parse(String(init?.body)).revision !== revision)
          return Response.json({error: {code: 'stale_revision'}}, {status: 409});
        if (!reviewed) {
          reviewed = true;
          reviewEffects++;
          retainPeer();
        }
        if (loseNextReview) {
          loseNextReview = false;
          throw Error('Declared committed review reply lost');
        }
        value = post();
      } else if (path === 'factory/feed/subscription')
        value = {subscriber: '1', repositories: ['7'], kinds: [], revision: 1};
      else if (path.startsWith('factory/feed?'))
        value = {
          entries: [
            {kind: 'post', id: postID, repository: '7', seq: 1, actor: '1', author: owner, post: post()},
            ...(artifacts
              ? [
                  {
                    kind: 'record',
                    id: 'd'.repeat(32),
                    repository: '7',
                    seq: 2,
                    payload: {
                      communication: {
                        kind: 'artifact',
                        title: 'Private evidence',
                        summary: 'Recorded bytes',
                        artifacts: [
                          {
                            id: 'e'.repeat(32),
                            name: 'private.txt',
                            media_type: 'text/plain',
                            href: '/api/factory/artifacts/' + 'e'.repeat(32) + '?repository=7',
                          },
                        ],
                      },
                    },
                  },
                ]
              : []),
          ],
          truncated: false,
        };
      else if (path.includes('/replies?'))
        value = {
          items: [{...post(), post_id: replyID, body: 'Reply from ' + owner, parent_id: postID}],
          next_before: 0,
        };
      else if (path === 'factory/feed/posts/' + postID) value = post();
      else if (path === 'repositories/7/factory') value = {};
      else throw Error('Unexpected declared feed request ' + method + ' ' + path);
      const row = structuredClone(value) as Record<string, unknown>;
      const source = method === 'GET' && path === 'factory/feed/posts/' + postID;
      const receipt = (method === 'POST' && path.endsWith('/retry')) || (method === 'GET' && path.includes('/retry/'));
      if (source && mismatch === 'source-post') row.post_id = 'f'.repeat(32);
      if (source && mismatch === 'source-repository') row.repository = '8';
      if (source && mismatch === 'source-numeric-repository') row.repository = 7;
      if (source && mismatch === 'source-revision') row.revision = '1';
      if (source && mismatch === 'source-unsafe-revision') row.revision = Number.MAX_SAFE_INTEGER;
      if (source && mismatch === 'source-readonly') row.can_retry = false;
      if (receipt && mismatch === 'receipt-post') row.post_id = 'f'.repeat(32);
      if (receipt && mismatch === 'receipt-command') row.command_id = 'f'.repeat(32);
      if (receipt && mismatch === 'receipt-repository') row.repository = '8';
      if (receipt && mismatch === 'receipt-numeric-repository') row.repository = 7;
      if (receipt && mismatch === 'receipt-expected') row.expected_revision = revision;
      if (receipt && mismatch === 'receipt-digest') row.body_digest = 'f'.repeat(64);
      if (receipt && mismatch === 'receipt-author') row.author = '2';
      if (receipt && mismatch === 'receipt-revision') row.revision = revision + 1;
      if (receipt && mismatch === 'receipt-numeric-revision') row.revision = String(revision);
      if (source && mismatch === 'review-invalid-intent' && row.product_change)
        (row.product_change as Record<string, unknown>).intent = '/foreign';
      if (method === 'POST' && path.endsWith('/review') && mismatch === 'review-foreign-intent' && row.product_change)
        (row.product_change as Record<string, unknown>).intent = 'd' + 'f'.repeat(24);
      if (row.product_change && mismatch === 'review-accepted')
        (row.product_change as Record<string, unknown>).accepted_revision = 'd' + 'f'.repeat(24);
      if (row.product_change && mismatch === 'review-missing-draft')
        delete (row.product_change as Record<string, unknown>).draft_revision;
      if (row.product_change && mismatch === 'review-invalid-draft')
        (row.product_change as Record<string, unknown>).draft_revision = '/invalid';
      const snapshot = JSON.stringify(row);
      if (mode === 'response' || mode === 'reject') {
        await wait();
        if (mode === 'reject') throw Error('Declared retired read failure');
      }
      if (mode === 'body')
        return new Response(
          new ReadableStream<Uint8Array>({
            start(controller) {
              held++;
              releases.push(() => {
                controller.enqueue(new TextEncoder().encode(snapshot));
                controller.close();
              });
            },
          }),
          {headers: {'Content-Type': 'application/json'}}
        );
      return new Response(snapshot, {headers: {'Content-Type': 'application/json'}});
    },
  };
  const host = {
    draftScope: 'declared:feed:' + owner,
    async repositories(_wanted: readonly string[], signal: AbortSignal) {
      capture('@catalogue', signal);
      if (rules.has('@catalogue')) await wait();
      return {refs: new Map([['7', {path: '/desktop/7', label: 'team/project'}]]), listedAll: true};
    },
    async applicationOrigins(signal: AbortSignal) {
      capture('@applications', signal);
      if (rules.has('@applications')) await wait();
      return new Map<string, string>();
    },
    links() {
      return {
        repository: '/desktop/7',
        feed: () => '/desktop/7',
        request: (id: string) => '/desktop/request/' + id,
        issue: (id: string | number) => '/desktop/issue/' + id,
        pull: (id: string | number) => '/desktop/pull/' + id,
        factoryStatus: () => '/desktop/work',
      };
    },
    navigate(path: string) {
      document.body.dataset.feedNavigation = path;
    },
  };
  return {
    transport,
    host,
    hold(prefix: string, mode: Hold) {
      rules.set(prefix, mode);
    },
    release() {
      rules.clear();
      releases.splice(0).forEach((release) => release());
    },
    held: () => held,
    productChange(value: boolean) {
      productChange = value;
      retainPeer();
    },
    artifacts(value: boolean) {
      artifacts = value;
    },
    fail(prefix: string, status: number) {
      if (status) failures.set(prefix, status);
      else failures.delete(prefix);
    },
    mismatch(value: string) {
      mismatch = value;
    },
    unchanged(value: boolean) {
      unchanged = value;
    },
    loseRevision() {
      loseNextRevision = true;
    },
    readonly(value: boolean) {
      canRetry = !value;
      retainPeer();
    },
    advance(body: string) {
      savedBody = body;
      revision++;
      retainPeer();
    },
    loseReview() {
      loseNextReview = true;
    },
    reviewAccess(value: boolean) {
      reviewAccess = value;
      retainPeer();
    },
    reviewEffects: () => reviewEffects,
  };
}
function peer() {
  let value = peers.get(account);
  if (!value) {
    value = makePeer(account);
    peers.set(account, value);
  }
  return value;
}
function context(): FactoryFeedContext {
  return {
    transport: peer().transport,
    host: peer().host,
    repository: '7',
    repositoryName: 'team/project',
    account,
    global,
  };
}
function mount(owner = 'A', combined = false) {
  account = owner;
  global = combined;
  element = new SodaFactoryFeed();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context();
}
const fixture = {
  mount,
  story(state: FeedStoryState) {
    element = new SodaFactoryFeed();
    document.querySelector('main')!.replaceChildren(element);
    element.context = feedStoryPeer(state).context;
  },
  calls: () => calls,
  hold(prefix: string, mode: Hold = 'response') {
    peer().hold(prefix, mode);
  },
  held: () => peer().held(),
  release: () => peers.forEach((peer) => peer.release()),
  aborted(owner: string, path: string) {
    return signals
      .filter((row) => row.account === owner && row.path.startsWith(path))
      .some((row) => row.signal.aborted);
  },
  switch(owner: string, combined = false) {
    account = owner;
    global = combined;
    element.context = context();
  },
  rebind() {
    element.context = context();
  },
  retireGeneration() {
    peer().transport.generation += ':retired';
  },
  retire() {
    element.context = undefined;
  },
  disconnect() {
    element.remove();
  },
  selectPost() {
    element.post = postID;
  },
  productChange(value: boolean) {
    peer().productChange(value);
  },
  fail(prefix: string, status: number) {
    peer().fail(prefix, status);
  },
  artifacts(value: boolean) {
    peer().artifacts(value);
  },
  urls: () => ({created, revoked}),
  mismatch(value: string) {
    peer().mismatch(value);
  },
  unchanged(value: boolean) {
    peer().unchanged(value);
  },
  loseRevision() {
    peer().loseRevision();
  },
  readonly(value: boolean) {
    peer().readonly(value);
  },
  storageFault(value: '' | 'read' | 'write' | 'remove' | 'draft-read') {
    storageFault = value;
  },
  revisionWrites: () => revisionWrites,
  reviewWrites: () => reviewWrites,
  loseReview() {
    peer().loseReview();
  },
  reviewAccess(value: boolean) {
    peer().reviewAccess(value);
  },
  reviewEffects: () => peer().reviewEffects(),
  reviewRecordKey(owner = account) {
    return 'soda-feed-review:v1:' + encodeURIComponent('declared:feed:' + owner) + ':7:' + postID + ':1';
  },
  reviewPending(owner = account) {
    return localStorage.getItem(this.reviewRecordKey(owner));
  },
  advance(body: string) {
    peer().advance(body);
  },
  revisionRecordKey(owner = account) {
    return 'soda-feed-retry:v1:' + encodeURIComponent('declared:feed:' + owner) + ':7:' + postID + ':1';
  },
  revisionPending(owner = account) {
    return localStorage.getItem(this.revisionRecordKey(owner));
  },
  pending(owner = account) {
    return localStorage.getItem('soda-feed-pending:v1:declared:feed:' + owner + ':project:7');
  },
  revisionDraft(owner = account) {
    return localStorage.getItem('soda-feed-revision:v2:declared:feed:' + owner + ':7:' + postID);
  },
};
declare global {
  interface Window {
    factoryFeedLifetime: typeof fixture;
  }
}
window.factoryFeedLifetime = fixture;
mount();

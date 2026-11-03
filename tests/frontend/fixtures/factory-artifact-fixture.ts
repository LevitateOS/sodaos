import {SodaFeedArtifact} from '@factory/product/frontend/factory/feed/feed-artifact.js';
import {SodaFactoryFeed} from '@factory/product/frontend/factory/feed/feed.js';
import {artifactStoryPeer} from '@factory/product/frontend/storybook/artifact-fixtures.js';
import type {ArtifactStoryState} from '@factory/product/frontend/storybook/artifact-fixtures.js';
import type {FactoryFeedContext} from '@factory/product/frontend/factory/feed/feed.js';
import type {FeedArtifact} from '@factory/product/frontend/factory/feed/feed-response.js';

let peer: ReturnType<typeof artifactStoryPeer>, element: SodaFeedArtifact, feed: SodaFactoryFeed;
const peers: ReturnType<typeof artifactStoryPeer>[] = [],
  created: string[] = [],
  revoked: string[] = [];
const create = URL.createObjectURL,
  revoke = URL.revokeObjectURL,
  blobText = Blob.prototype.text;
URL.createObjectURL = (blob) => {
  const url = create.call(URL, blob);
  created.push(url);
  return url;
};
URL.revokeObjectURL = (url) => {
  revoked.push(url);
  revoke.call(URL, url);
};
let holdDecode = false;
const decode: (() => void)[] = [];
Blob.prototype.text = async function () {
  const text = await blobText.call(this);
  if (holdDecode) await new Promise<void>((resolve) => decode.push(resolve));
  return text;
};
function nextPeer(state: ArtifactStoryState) {
  peer = artifactStoryPeer(state);
  peers.push(peer);
  return peer;
}
function mount(state: ArtifactStoryState = 'text') {
  nextPeer(state);
  element = new SodaFeedArtifact();
  element.artifact = peer.artifact;
  element.context = peer.context;
  document.querySelector('main')!.replaceChildren(element);
}
function mountFeed() {
  nextPeer('text');
  const binary = peer,
    controller = new AbortController();
  const communication = {
    kind: 'artifact',
    title: 'Recorded evidence',
    summary: 'Preview bytes explicitly.',
    artifacts: [binary.artifact],
  };
  const transport = {
    generation: 'declared-feed-artifacts',
    assetBase: '/',
    async request(path: string, init?: RequestInit) {
      if (path.startsWith('factory/artifacts/')) return binary.context.transport.request(path, init);
      if (path.startsWith('factory/feed?'))
        return Response.json({
          entries: [
            {
              kind: 'record',
              id: 'b'.repeat(32),
              repository: binary.context.repository,
              seq: 1,
              payload: {communication},
            },
          ],
          truncated: false,
        });
      if (path === 'factory/feed/subscription')
        return Response.json({subscriber: '1', repositories: [binary.context.repository], revision: 1});
      if (path.includes('/replies?'))
        return Response.json({
          items: [
            {
              post_id: 'c'.repeat(32),
              at: 1720000000,
              author_name: 'Human',
              communication: {...communication, title: 'Reply evidence'},
            },
          ],
          next_before: 0,
        });
      if (path.endsWith('/factory')) return Response.json({});
      throw Error('Unexpected declared feed request');
    },
  };
  const context: FactoryFeedContext = {
    transport,
    repository: binary.context.repository,
    host: {
      draftScope: 'declared:artifact-feed',
      async repositories() {
        return {
          refs: new Map([[binary.context.repository, {path: '/desktop/repo', label: 'team/project'}]]),
          listedAll: true,
        };
      },
      links() {
        return {
          repository: '/desktop/repo',
          feed: () => '/desktop/feed',
          request: (id: string) => '/desktop/request/' + id,
          issue: (id: string | number) => '/desktop/issue/' + id,
          pull: (id: string | number) => '/desktop/pull/' + id,
          factoryStatus: () => '/desktop/work',
        };
      },
      navigate() {},
    },
  };
  feed = new SodaFactoryFeed();
  feed.context = context;
  // This controller represents the fixture's explicit host retirement action.
  controller.signal.addEventListener('abort', () => {
    feed.context = undefined;
  });
  document.querySelector('main')!.replaceChildren(feed);
  return controller;
}
let feedController: AbortController | undefined;
const fixture = {
  mount,
  mountFeed() {
    feedController = mountFeed();
  },
  retireFeed() {
    feedController?.abort();
  },
  hold: (value: 'headers' | 'body') => peer.hold(value),
  held: () => peer.held(),
  release: () => peers.forEach((peer) => peer.release()),
  calls: () => peers.flatMap((peer) => peer.calls),
  peerIndex: () => peers.length - 1,
  aborted: (index?: number) => (index === undefined ? peer : peers[index]!).aborted(),
  cancelled: () => peer.cancelled(),
  urls: () => ({created, revoked}),
  rebind() {
    element.artifact = {...peer.artifact};
    element.context = {...peer.context};
  },
  replace(artifact: Partial<FeedArtifact>) {
    element.artifact = {...element.artifact!, ...artifact};
  },
  nextContext() {
    const current = nextPeer('text');
    element.context = current.context;
  },
  fail: (status: number) => peer.fail(status),
  text: (value: string) => peer.text(value),
  retire: () => peer.retire(),
  retireGeneration: () => peer.retireGeneration(),
  disconnect() {
    element.remove();
  },
  holdDecode() {
    holdDecode = true;
  },
  decoding: () => decode.length,
  releaseDecode() {
    holdDecode = false;
    decode.splice(0).forEach((resolve) => resolve());
  },
};
declare global {
  interface Window {
    factoryArtifact: typeof fixture;
  }
}
window.factoryArtifact = fixture;
mount();

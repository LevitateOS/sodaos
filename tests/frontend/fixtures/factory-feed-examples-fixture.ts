import {feedStoryPeer} from '@factory/product/frontend/storybook/fixtures.js';
import type {FeedStoryState} from '@factory/product/frontend/storybook/fixtures.js';

// Fountain already registered the production element; standalone owns startup.
if (!customElements.get('soda-factory-feed')) await import('@factory/product/frontend/factory/feed/feed.js');

let peer: ReturnType<typeof feedStoryPeer>;
function prepare(state: FeedStoryState) {
  peer = feedStoryPeer(state);
}
function mount(state: FeedStoryState) {
  prepare(state);
  const element = document.createElement('soda-factory-feed');
  element.context = peer.context;
  document.querySelector('main')!.replaceChildren(element);
}
const fixture = {
  mount,
  prepare,
  calls: () => peer.calls,
  ledger: () => peer.ledger(),
  release: () => peer.release(),
  request: (path: string, init?: RequestInit) => peer.context.transport.request(path, init),
};
declare global {
  interface Window {
    factoryFeedExamples: typeof fixture;
  }
}
window.factoryFeedExamples = fixture;
mount('activity');

import {SodaFactoryWatch} from '@factory/product/frontend/factory/watch/watch.js';
import {watchRenderer, watchStoryPeer} from '@factory/product/frontend/storybook/watch-fixtures.js';
import type {WatchStoryState} from '@factory/product/frontend/storybook/watch-fixtures.js';

let view: SodaFactoryWatch;
let peer: ReturnType<typeof watchStoryPeer>;
let hides = 0;
let authorityLosses = 0;
function mount(state: WatchStoryState, holdRead = false) {
  peer = watchStoryPeer(state);
  if (holdRead) peer.holdRead();
  view = new SodaFactoryWatch();
  view.configure(peer.context, async () => watchRenderer);
  hides = authorityLosses = 0;
  view.addEventListener('soda-factory-command', () => hides++);
  view.addEventListener('soda-factory-authority-lost', () => authorityLosses++);
  document.querySelector('main')!.replaceChildren(view);
}
const fixture = {
  mount,
  reads: () => peer.reads,
  peers: () => peer.peers.map((item) => ({state: item.readyState, sent: item.sent})),
  hides: () => hides,
  authorityLosses: () => authorityLosses,
  lose: () => peer.peers.at(-1)?.lose(),
  bad: () => peer.peers.at(-1)?.frame({type: 'output', cursor: -1, next: 0, data: ''}),
  retireGeneration: () => peer.retireGeneration(),
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  retire: () => view.retire(),
  stop: () => view.stop(),
  suspend: () => view.suspend(),
  resume: () => view.resume({generation: peer.context.transport.generation}),
  late: () => peer.peers[0]?.output('Retired output must not render.', 0),
};
declare global {
  interface Window {
    factoryWatchFixture: typeof fixture;
  }
}
window.factoryWatchFixture = fixture;
mount('idle');

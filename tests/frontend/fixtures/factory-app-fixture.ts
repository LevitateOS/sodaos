import {SodaFactoryApp} from '@factory/product/frontend/factory/app/factory-app.js';
import {appStoryPeer} from '@factory/product/frontend/storybook/app-fixtures.js';
import type {AppStoryState} from '@factory/product/frontend/storybook/app-fixtures.js';
import type {FactoryRoute} from '@factory/product/frontend/factory/app/routes.js';

let element: SodaFactoryApp, peer: ReturnType<typeof appStoryPeer>;
const peers: ReturnType<typeof appStoryPeer>[] = [];
function mount(state: AppStoryState = 'ready', account = 'Alex') {
  element = new SodaFactoryApp();
  document.querySelector('#root')!.replaceChildren(element);
  connect(state, account);
}
function connect(state: AppStoryState, account: string) {
  peer = appStoryPeer(state, account);
  peers.push(peer);
  peer.attach(element);
}
const fixture = {
  mount,
  connect,
  select(route: FactoryRoute) {
    peer.select(route);
  },
  calls: () => peers.flatMap((peer) => peer.calls.map((call) => ({path: call.path, method: call.method}))),
  resolved: () => peers.flatMap((peer) => peer.resolved),
  hold: () => peer.hold(),
  held: () => peer.held(),
  release: () => peers.forEach((peer) => peer.release()),
  aborted: () => peers.some((peer) => peer.aborted()),
  childAborted: () => peers.some((peer) => peer.calls.some((call) => call.signal?.aborted)),
  retire: () => peer.retire(),
  retireGeneration: () => peer.retireGeneration(),
  retireBeforeConnect: () => peer.retireBeforeConnect(),
  preConnectAborted: () => peer.preConnectAborted(),
  unavailable(value: boolean) {
    peer.unavailable(value);
  },
  mismatch(value: string) {
    peer.mismatch(value);
  },
  failIssue: (status: number) => peer.failIssue(status),
  holdIssue: () => peer.holdIssue(),
  releaseIssue: () => peers.forEach((peer) => peer.releaseIssue()),
  issueHeld: () => peer.issueHeld(),
  holdFacts: () => peer.holdFacts(),
  releaseFacts: () => peers.forEach((peer) => peer.releaseFacts()),
  factsHeld: () => peer.factsHeld(),
  factsMode: (value: string) => peer.factsMode(value),
  links: () => peers.flatMap((peer) => peer.links()),
  holdLink: () => peer.holdLink(),
  releaseLink: () => peers.forEach((peer) => peer.releaseLink()),
  linkAborted: () => peers.some((peer) => peer.linkAborted()),
  watchRun: 'a'.repeat(32),
};
declare global {
  interface Window {
    factoryAppFixture: typeof fixture;
  }
}
window.factoryAppFixture = fixture;
mount();

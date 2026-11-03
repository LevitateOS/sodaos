import {SodaFactoryLibrary} from '@factory/product/frontend/factory/library/library.js';
import type {LibrarySource} from '@factory/product/frontend/factory/library/library.js';
import {
  libraryStoryPeer,
  libraryItemID,
  libraryRepository,
} from '@factory/product/frontend/storybook/library-fixtures.js';
import type {LibraryStoryState} from '@factory/product/frontend/storybook/library-fixtures.js';

let peer: ReturnType<typeof libraryStoryPeer>, element: SodaFactoryLibrary;
const peers: ReturnType<typeof libraryStoryPeer>[] = [],
  sources: LibrarySource[] = [];
let navigations = 0;
function mount(state: LibraryStoryState = 'recorded', held = false) {
  peer = libraryStoryPeer(state);
  peers.push(peer);
  if (held) peer.hold();
  element = new SodaFactoryLibrary();
  element.context = peer.context;
  element.addEventListener('library-source-open', (event) =>
    sources.push((event as CustomEvent<LibrarySource>).detail)
  );
  element.addEventListener('library-navigate', () => navigations++);
  document.querySelector('main')!.replaceChildren(element);
}
const fixture = {
  mount,
  calls: () => peers.flatMap((peer) => peer.calls),
  sources: () => sources,
  navigations: () => navigations,
  select(id: string | undefined) {
    element.itemID = id;
  },
  itemID: libraryItemID,
  repository: libraryRepository,
  refresh: () => element.refresh(),
  rebind() {
    element.context = {...peer.context};
  },
  switchRepository(repository: string) {
    peer = libraryStoryPeer('recorded', repository);
    peers.push(peer);
    element.context = peer.context;
  },
  pendingSource(id?: string) {
    element.pendingSource = id;
  },
  hold: () => peer.hold(),
  release: () => peers.forEach((peer) => peer.release()),
  aborted: () => peers.some((peer) => peer.aborted()),
  retire() {
    element.context = undefined;
  },
  disconnect() {
    element.remove();
  },
  retireGeneration: () => peer.retireGeneration(),
  fail: (code: number) => peer.fail(code),
  mismatch: (value: Parameters<typeof peer.mismatch>[0]) => peer.mismatch(value),
};
declare global {
  interface Window {
    factoryLibraryFixture: typeof fixture;
  }
}
window.factoryLibraryFixture = fixture;
mount();

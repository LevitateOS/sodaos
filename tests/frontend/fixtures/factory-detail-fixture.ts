import {SodaFactoryWorkDetail} from '@factory/product/frontend/factory/work/detail.js';
import {detailStoryPeer} from '@factory/product/frontend/storybook/detail-fixtures.js';
import type {DetailStoryState} from '@factory/product/frontend/storybook/detail-fixtures.js';
import {presentationLinks} from '@factory/product/frontend/storybook/work-presentation-fixtures.js';

let peer: ReturnType<typeof detailStoryPeer>;
let element: SodaFactoryWorkDetail;
let account = 'primary';
let attempted = '';
const storage = {get: Storage.prototype.getItem, set: Storage.prototype.setItem, remove: Storage.prototype.removeItem};
function storageKey(scope = account) {
  return `soda-work-run-retry:v1:${encodeURIComponent(`declared:detail:${scope}`)}:${peer.context.repository}:${peer.context.issue}`;
}
const watches: unknown[] = [];
function context() {
  const captured = account;
  return {
    ...peer.context,
    commandScope: async (signal: AbortSignal) => {
      signal.throwIfAborted();
      return `declared:detail:${captured}`;
    },
  };
}
function mount(state: DetailStoryState, holdRead = false) {
  peer = detailStoryPeer(state);
  if (holdRead) peer.hold('read');
  element = new SodaFactoryWorkDetail();
  document.querySelector('main')!.replaceChildren(element);
  element.addEventListener('soda-factory-watch-request', (event) => watches.push((event as CustomEvent).detail));
  watches.length = 0;
  element.context = context();
}
const fixture = {
  mount,
  calls: () => peer.calls,
  watches: () => watches,
  rebind() {
    element.context = {
      ...context(),
      links: presentationLinks('/another-shell'),
      capabilities: Object.fromEntries(
        Object.entries(peer.context.capabilities).reverse()
      ) as typeof peer.context.capabilities,
    };
  },
  holdRetry: () => peer.hold('retry'),
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  retire: () => {
    element.context = undefined;
  },
  retireGeneration: () => peer.retireGeneration(),
  disconnect: () => element.remove(),
  fail: (status: number) => peer.fail(status),
  account(value: string) {
    account = value;
  },
  saved: (scope = account) => storage.get.call(localStorage, storageKey(scope)),
  seed(value: string, scope = account) {
    if (value) storage.set.call(localStorage, storageKey(scope), value);
    else storage.remove.call(localStorage, storageKey(scope));
  },
  attempted: () => attempted,
  storageFailure(mode: 'read' | 'write' | 'none') {
    Storage.prototype.getItem =
      mode === 'read'
        ? function () {
            throw Error('Declared unreadable storage');
          }
        : storage.get;
    Storage.prototype.setItem =
      mode === 'write'
        ? function (key, value) {
            if (key.startsWith('soda-work-run-retry:')) attempted = value;
            throw Error('Declared unwritable storage');
          }
        : storage.set;
    Storage.prototype.removeItem =
      mode === 'write'
        ? function () {
            throw Error('Declared unwritable storage');
          }
        : storage.remove;
  },
};
declare global {
  interface Window {
    factoryDetailFixture: typeof fixture;
  }
}
window.factoryDetailFixture = fixture;
mount('ready');

import {SodaFactoryVerificationAllowance} from '@factory/product/frontend/factory/work/verification-allowance.js';
import {
  allowanceStoryPeer,
  allowanceRepository,
  allowanceIntent,
} from '@factory/product/frontend/storybook/allowance-fixtures.js';
import type {AllowanceStoryState} from '@factory/product/frontend/storybook/allowance-fixtures.js';
import {readAllowanceChange} from '@factory/product/frontend/factory/work/allowance-record.js';
let peer: ReturnType<typeof allowanceStoryPeer>, element: SodaFactoryVerificationAllowance;
let account = 'A',
  available = true;
const address = () => ({
  scope: `declared:allowance:${account}`,
  repository: allowanceRepository,
  intent: allowanceIntent,
});
const key = () =>
  `soda-work-verification-allowance:v1:${encodeURIComponent(address().scope)}:${allowanceRepository}:${allowanceIntent}`;
const storage = {get: Storage.prototype.getItem, set: Storage.prototype.setItem, remove: Storage.prototype.removeItem};
const context = () => ({
  ...peer.context,
  commandScope: async (signal: AbortSignal) => {
    signal.throwIfAborted();
    if (!available) throw Error('Declared identity unavailable');
    return address().scope;
  },
});
function mount(state: AllowanceStoryState, held = false, preserve = false) {
  if (!preserve)
    for (const saved of Object.keys(localStorage))
      if (saved.startsWith('soda-work-verification-allowance:v1:declared%3Aallowance%3A'))
        storage.remove.call(localStorage, saved);
  peer = allowanceStoryPeer(state);
  if (held) peer.hold();
  element = new SodaFactoryVerificationAllowance();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context();
}
const fixture = {
  mount,
  remount() {
    mount('ready', false, true);
  },
  calls: () => peer.calls,
  refresh: () => element.refresh(),
  rebind() {
    element.context = context();
  },
  account(value: string) {
    account = value;
  },
  identity(value: boolean) {
    available = value;
  },
  saved: () => storage.get.call(localStorage, key()),
  pending: () => readAllowanceChange(address()).pending,
  seed(value: string | null) {
    if (value === null) storage.remove.call(localStorage, key());
    else storage.set.call(localStorage, key(), value);
  },
  hold: () => peer.hold(),
  holdAction: () => peer.hold(true),
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  retire() {
    element.context = undefined;
  },
  disconnect() {
    element.remove();
  },
  retireGeneration: () => peer.retireGeneration(),
  fail: (code: number) => peer.fail(code),
  receipt: (patch: Record<string, unknown>) => peer.receipt(patch),
  advance: () => peer.advance(),
  scope: (repo: string) => peer.scope(repo),
  storageFailure(kind: 'read' | 'write' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-work-verification-allowance:')) throw Error('Declared read denial');
      return storage.get.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && key.startsWith('soda-work-verification-allowance:')) throw Error('Declared write denial');
      storage.set.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-work-verification-allowance:'))
        throw Error('Declared removal denial');
      storage.remove.call(this, key);
    };
  },
};
declare global {
  interface Window {
    factoryAllowanceFixture: typeof fixture;
  }
}
window.factoryAllowanceFixture = fixture;
mount('ready', false, true);

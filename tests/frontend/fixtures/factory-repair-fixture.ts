import {SodaFactoryRepair} from '@factory/product/frontend/factory/work/repair-funding.js';
import {repairStoryPeer, repairRepository} from '@factory/product/frontend/storybook/repair-fixtures.js';
import type {RepairStoryState} from '@factory/product/frontend/storybook/repair-fixtures.js';
import {readRepairChange} from '@factory/product/frontend/factory/work/repair-record.js';
let peer: ReturnType<typeof repairStoryPeer>, element: SodaFactoryRepair;
let account = 'A',
  available = true;
const address = () => ({scope: `declared:repair:${account}`, repository: repairRepository});
const key = () => `soda-work-repair-funding:v1:${encodeURIComponent(address().scope)}:${repairRepository}`;
const storage = {get: Storage.prototype.getItem, set: Storage.prototype.setItem, remove: Storage.prototype.removeItem};
const context = () => ({
  ...peer.context,
  commandScope: async (signal: AbortSignal) => {
    signal.throwIfAborted();
    if (!available) throw Error('Declared identity unavailable');
    return address().scope;
  },
});
function mount(state: RepairStoryState, held = false, preserve = false) {
  if (!preserve)
    for (const saved of Object.keys(localStorage))
      if (saved.startsWith('soda-work-repair-funding:v1:declared%3Arepair%3A'))
        storage.remove.call(localStorage, saved);
  peer = repairStoryPeer(state);
  if (held) peer.hold();
  element = new SodaFactoryRepair();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context();
}
let invalid = 0;
document.addEventListener('soda-session-invalid', () => invalid++);
const fixture = {
  mount,
  remount() {
    mount('unfunded', false, true);
  },
  calls: () => peer.calls,
  invalid: () => invalid,
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
  pending: () => readRepairChange(address()).pending,
  seed(value: string | null) {
    if (value === null) storage.remove.call(localStorage, key());
    else storage.set.call(localStorage, key(), value);
  },
  hold: () => peer.hold(),
  holdWrite: () => peer.hold(true),
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
  failReadAfterWrite: (code: number) => peer.failReadAfterWrite(code),
  receipt: (patch: Record<string, unknown>) => peer.receipt(patch),
  advance: () => peer.advance(),
  foreign: (value: boolean) => peer.foreign(value),
  storageFailure(kind: 'read' | 'write' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-work-repair-funding:')) throw Error('Declared read denial');
      return storage.get.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && key.startsWith('soda-work-repair-funding:')) throw Error('Declared write denial');
      storage.set.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-work-repair-funding:')) throw Error('Declared removal denial');
      storage.remove.call(this, key);
    };
  },
};
declare global {
  interface Window {
    factoryRepairFixture: typeof fixture;
  }
}
window.factoryRepairFixture = fixture;
mount('unfunded', false, true);

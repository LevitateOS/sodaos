import {SodaFactoryVerificationRetry} from '@factory/product/frontend/factory/work/verification-retry.js';
import {verificationStoryPeer} from '@factory/product/frontend/storybook/verification-fixtures.js';
import type {VerificationStoryState} from '@factory/product/frontend/storybook/verification-fixtures.js';
import type {DeliveryRepairProgress} from '@factory/product/frontend/factory/api/run-response.js';
import {readVerificationRequest} from '@factory/product/frontend/factory/work/verification-record.js';
let peer: ReturnType<typeof verificationStoryPeer>;
let element: SodaFactoryVerificationRetry;
let account = 'A';
let identityAvailable = true;
let selected = true;
const scope = () => `declared:verification:${account}`;
const address = () => ({scope: scope(), repository: peer.context.repository});
const key = () => `soda-work-verification:v1:${encodeURIComponent(scope())}:${peer.context.repository}`;
const storage = {get: Storage.prototype.getItem, set: Storage.prototype.setItem, remove: Storage.prototype.removeItem};
function context() {
  return {
    ...peer.context,
    runs: selected ? peer.context.runs : [],
    commandScope: async (signal: AbortSignal) => {
      signal.throwIfAborted();
      if (!identityAvailable) throw Error('Declared identity unavailable');
      return scope();
    },
  };
}
function mount(state: VerificationStoryState, held = false, preserve = false) {
  if (!preserve)
    for (const saved of Object.keys(localStorage))
      if (saved.startsWith('soda-work-verification:v1:declared%3Averification%3A')) localStorage.removeItem(saved);
  selected = true;
  peer = verificationStoryPeer(state);
  if (held) peer.hold('read');
  element = new SodaFactoryVerificationRetry();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context();
}
const fixture = {
  mount,
  calls: () => peer.calls,
  remount(state: VerificationStoryState = 'retry') {
    mount(state, false, true);
  },
  account(value: string) {
    account = value;
  },
  identity(available: boolean) {
    identityAvailable = available;
  },
  selected(enabled: boolean) {
    selected = enabled;
    element.context = context();
  },
  saved: () => storage.get.call(localStorage, key()),
  pending: () => readVerificationRequest(address()).pending,
  seed(value: string | null) {
    if (value === null) storage.remove.call(localStorage, key());
    else storage.set.call(localStorage, key(), value);
  },
  storageFailure(kind: 'read' | 'write' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-work-verification:')) throw Error('Declared storage read denial');
      return storage.get.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && key.startsWith('soda-work-verification:')) throw Error('Declared storage write denial');
      storage.set.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-work-verification:'))
        throw Error('Declared storage removal denial');
      storage.remove.call(this, key);
    };
  },
  rebind() {
    element.context = {...context(), runs: [...context().runs]};
  },
  refresh: () => element.refresh(),
  hold: (kind: 'read' | 'action') => peer.hold(kind),
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  retire() {
    element.context = undefined;
  },
  retireGeneration: () => peer.retireGeneration(),
  disconnect: () => element.remove(),
  fail: (status: number) => peer.fail(status),
  receipt: (patch: Record<string, unknown>) => peer.receipt(patch),
  read: (patch: Record<string, unknown>) => peer.read(patch),
  progress: (next: DeliveryRepairProgress['state']) => peer.progress(next),
};
declare global {
  interface Window {
    factoryVerificationFixture: typeof fixture;
  }
}
window.factoryVerificationFixture = fixture;
mount('retry', false, true);

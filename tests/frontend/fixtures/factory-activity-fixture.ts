import {SodaFactoryActivity} from '@factory/product/frontend/factory/work/activity.js';
import {activityStoryPeer, activityRepository} from '@factory/product/frontend/storybook/activity-fixtures.js';
import type {ActivityStoryState} from '@factory/product/frontend/storybook/activity-fixtures.js';
import {readActivityDraft} from '@factory/product/frontend/factory/work/activity-draft.js';
let peer: ReturnType<typeof activityStoryPeer>, element: SodaFactoryActivity;
let account = 'A',
  identity = true,
  prefix: string | undefined = '/desktop',
  editable = true;
const address = () => ({scope: `declared:activity:${account}`, repository: activityRepository});
const key = () => `soda-work-activity:v1:${encodeURIComponent(address().scope)}:${activityRepository}`;
const storage = {get: Storage.prototype.getItem, set: Storage.prototype.setItem, remove: Storage.prototype.removeItem};
function context() {
  const {presentItem: _links, ...base} = peer.context;
  const currentPrefix = prefix;
  return {
    ...base,
    editable: editable && base.editable,
    commandScope: async (signal: AbortSignal) => {
      signal.throwIfAborted();
      if (!identity) throw Error('Declared identity unavailable');
      return address().scope;
    },
    ...(currentPrefix === undefined
      ? {}
      : {
          presentItem: (item: Parameters<NonNullable<typeof peer.context.presentItem>>[0]) => ({
            href:
              item.kind === 'pr_created' ? `${currentPrefix}/pull/${item.pr}` : `${currentPrefix}/issue/${item.issue}`,
          }),
        }),
  };
}
function mount(state: ActivityStoryState, held = false, preserve = false) {
  if (!preserve)
    for (const saved of Object.keys(localStorage))
      if (saved.startsWith('soda-work-activity:v1:declared%3Aactivity%3A')) storage.remove.call(localStorage, saved);
  peer = activityStoryPeer(state);
  if (held) peer.hold();
  element = new SodaFactoryActivity();
  document.querySelector('main')!.replaceChildren(element);
  element.context = context();
}
const fixture = {
  mount,
  remount() {
    mount('recorded', false, true);
  },
  calls: () => peer.calls,
  account(value: string) {
    account = value;
  },
  identity(value: boolean) {
    identity = value;
  },
  links(value?: string) {
    prefix = value;
    element.context = context();
  },
  editable(value: boolean) {
    editable = value;
    element.context = context();
  },
  saved: () => storage.get.call(localStorage, key()),
  draft: () => readActivityDraft(address()).draft,
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
  mismatch: (value: boolean) => peer.mismatch(value),
  advance: () => peer.advance(),
  storageFailure(kind: 'read' | 'write' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-work-activity:')) throw Error('Declared read denial');
      return storage.get.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && key.startsWith('soda-work-activity:')) throw Error('Declared write denial');
      storage.set.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-work-activity:')) throw Error('Declared removal denial');
      storage.remove.call(this, key);
    };
  },
};
declare global {
  interface Window {
    factoryActivityFixture: typeof fixture;
  }
}
window.factoryActivityFixture = fixture;
mount('recorded', false, true);

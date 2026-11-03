import {SodaFactoryContextResolution} from '@factory/product/frontend/factory/work/context-resolution.js';
import {contextStoryPeer} from '@factory/product/frontend/storybook/context-fixtures.js';
import {writeStored} from '@factory/product/frontend/factory/shared/drafts.js';
import {readContextChoices} from '@factory/product/frontend/factory/work/context-record.js';
import type {PendingContextChoice} from '@factory/product/frontend/factory/work/context-record.js';
import type {ContextStoryState} from '@factory/product/frontend/storybook/context-fixtures.js';

let peer: ReturnType<typeof contextStoryPeer>;
let element: SodaFactoryContextResolution;
let events = 0;
let entry: 'issue' | 'run' = 'issue';
const storageKey = (scope = peer?.scope() ?? 'declared-profile:provider:account') =>
  `soda-work-context:v1:${encodeURIComponent(scope)}:9007199254740993:12`;
const knownKeys = new Set<string>();
const storageGet = Storage.prototype.getItem,
  storageSet = Storage.prototype.setItem,
  storageRemove = Storage.prototype.removeItem;
function mount(
  state: ContextStoryState,
  mode: 'issue' | 'run' = 'issue',
  holdRead: boolean | 'scope' = false,
  preserve = false
) {
  if (!preserve) {
    knownKeys.add(storageKey());
    for (const key of Object.keys(localStorage)) if (key.startsWith('soda-work-context:')) knownKeys.add(key);
    for (const key of knownKeys) writeStored(key, '');
  }
  peer = contextStoryPeer(state);
  entry = mode;
  if (holdRead) peer.hold(holdRead === true ? 'read' : 'scope');
  element = new SodaFactoryContextResolution();
  document.querySelector('main')!.replaceChildren(element);
  element.addEventListener('soda-context-resolved', () => events++);
  events = 0;
  element.context = peer.context(entry);
}
const fixture = {
  mount,
  calls: () => peer.calls,
  clearCalls: () => peer.clearCalls(),
  saved: () => storageGet.call(localStorage, storageKey()),
  pending: () => readContextChoices({scope: peer.scope(), repository: '9007199254740993', issue: '12'}),
  remount: (state: ContextStoryState = 'complete') => mount(state, entry, false, true),
  latestRun: (id: string) => peer.latestRun(id),
  selectedRun: () => (element as unknown as {selectedRun?: {id: string}}).selectedRun?.id,
  seed(value: string | null) {
    knownKeys.add(storageKey());
    writeStored(storageKey(), value ?? '');
  },
  replaceRaw(value: string) {
    storageSet.call(localStorage, storageKey(), value);
  },
  account(value: string) {
    peer.changeScope(value);
    element.context = peer.context(entry);
  },
  recorded: (choice: PendingContextChoice) => peer.recorded(choice),
  receipt: (patch: Record<string, unknown>, status?: number) => peer.receipt(patch, status),
  advanceRequired() {
    peer.advanceRequired();
  },
  omitSelection() {
    peer.omitSelection();
  },
  storageFailure(kind: 'read' | 'write' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && key.startsWith('soda-work-context:')) throw Error('Declared storage read denial');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && key.startsWith('soda-work-context:')) throw Error('Declared storage write denial');
      storageSet.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && key.startsWith('soda-work-context:')) throw Error('Declared storage removal denial');
      storageRemove.call(this, key);
    };
  },
  events: () => events,
  refresh: () => element.refresh(),
  rebind: () => {
    element.context = peer.context(entry);
  },
  revise: () => {
    peer.revise();
    element.context = peer.context(entry);
  },
  supplyRun: () => {
    entry = 'run';
    element.context = peer.context(entry);
  },
  holdAction: () => peer.hold('action'),
  holdRead: () => peer.hold('read'),
  holdScope: () => peer.hold('scope'),
  changeScope: (value: string) => peer.changeScope(value),
  failScope: () => peer.failScope(),
  wireScope: (repository: string, issue: string, id?: string) => peer.wireScope(repository, issue, id),
  missingScope: () => {
    const {commandScope: _, ...context} = peer.context(entry);
    element.context = context;
  },
  revisePeer: () => peer.revise(),
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  heldCount: () => peer.heldCount(),
  retire: () => {
    element.context = undefined;
  },
  disconnect: () => element.remove(),
  retireGeneration: () => peer.retireGeneration(),
};
declare global {
  interface Window {
    factoryContextFixture: typeof fixture;
  }
}
window.factoryContextFixture = fixture;
mount('complete', 'issue', false, true);

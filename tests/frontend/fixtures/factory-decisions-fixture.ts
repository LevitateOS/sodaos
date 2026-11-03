import {SodaFactoryRequirementsAcceptance} from '@factory/product/frontend/factory/work/acceptance.js';
import {SodaFactoryCompletionRetry} from '@factory/product/frontend/factory/work/completion-retry.js';
import {decisionLinks, decisionStoryPeer} from '@factory/product/frontend/storybook/decision-fixtures.js';
import type {DecisionStoryState} from '@factory/product/frontend/storybook/decision-fixtures.js';
import {decisionIssue, decisionRepository} from '@factory/product/frontend/storybook/decision-fixtures.js';
import {readCompletionRetry} from '@factory/product/frontend/factory/work/completion-retry-record.js';
import {readAcceptance} from '@factory/product/frontend/factory/work/acceptance-record.js';

let peer: ReturnType<typeof decisionStoryPeer>;
let acceptance: SodaFactoryRequirementsAcceptance;
let retry: SodaFactoryCompletionRetry;
const events: string[] = [];
let account = 'A';
let identityAvailable = true;
const scope = () => `declared-decisions:account:${account}`;
const address = () => ({scope: scope(), repository: decisionRepository, issue: decisionIssue});
const storageKey = () =>
  `soda-work-completion-retry:v1:${encodeURIComponent(scope())}:${decisionRepository}:${decisionIssue}`;
const acceptanceKey = () =>
  `soda-work-acceptance:v1:${encodeURIComponent(scope())}:${decisionRepository}:${decisionIssue}`;
const commandScope = async (signal: AbortSignal) => {
  signal.throwIfAborted();
  if (!identityAvailable) throw Error('Declared identity unavailable');
  return scope();
};
const acceptanceContext = () => ({...peer.acceptance, commandScope});
const recoveryKey = (key: string) =>
  key.startsWith('soda-work-completion-retry:') || key.startsWith('soda-work-acceptance:');
const storageGet = Storage.prototype.getItem,
  storageSet = Storage.prototype.setItem,
  storageRemove = Storage.prototype.removeItem;
function mount(state: DecisionStoryState, holdReads = false, preserve = false) {
  if (!preserve) {
    for (const key of Object.keys(localStorage))
      if (recoveryKey(key) && key.includes(':v1:declared-decisions%3Aaccount%3A')) localStorage.removeItem(key);
  }
  peer = decisionStoryPeer(state);
  if (holdReads) peer.hold();
  acceptance = new SodaFactoryRequirementsAcceptance();
  retry = new SodaFactoryCompletionRetry();
  acceptance.addEventListener('soda-factory-requirements-accepted', () => events.push('accepted'));
  retry.addEventListener('soda-completion-retry-queued', () => events.push('retry'));
  document.querySelector('main')!.replaceChildren(acceptance, retry);
  acceptance.context = acceptanceContext();
  retry.context = {
    ...peer.retry,
    commandScope,
  };
  events.length = 0;
}
const fixture = {
  mount,
  remount() {
    mount('ready', false, true);
  },
  account(value: string) {
    account = value;
  },
  identity(available: boolean) {
    identityAvailable = available;
  },
  saved: () => storageGet.call(localStorage, storageKey()),
  pending: () => readCompletionRetry(address()).pending,
  savedAcceptance: () => storageGet.call(localStorage, acceptanceKey()),
  pendingAcceptance: () => readAcceptance(address()).pending,
  seedAcceptance(value: string | null) {
    if (value === null) localStorage.removeItem(acceptanceKey());
    else storageSet.call(localStorage, acceptanceKey(), value);
  },
  fullReview: () => peer.fullReview(),
  seed(value: string | null) {
    if (value === null) localStorage.removeItem(storageKey());
    else storageSet.call(localStorage, storageKey(), value);
  },
  advanceGate: (omit = false) => peer.advanceGate(omit),
  receipt: (patch: Record<string, unknown>, status?: number) => peer.receipt(patch, status),
  storageFailure(kind: 'read' | 'write' | 'remove' | 'none') {
    Storage.prototype.getItem = function (key: string) {
      if (kind === 'read' && recoveryKey(key)) throw Error('Declared storage read denial');
      return storageGet.call(this, key);
    };
    Storage.prototype.setItem = function (key: string, value: string) {
      if (kind === 'write' && recoveryKey(key)) throw Error('Declared storage write denial');
      storageSet.call(this, key, value);
    };
    Storage.prototype.removeItem = function (key: string) {
      if (kind === 'remove' && recoveryKey(key)) throw Error('Declared storage removal denial');
      storageRemove.call(this, key);
    };
  },
  calls: () => peer.calls,
  events: () => events,
  rebindLinks() {
    acceptance.context = {...acceptanceContext(), links: decisionLinks('/another-host')};
  },
  removeLinks() {
    const {links: _links, ...context} = acceptanceContext();
    acceptance.context = context;
  },
  hold: () => peer.hold(),
  release: () => peer.release(),
  aborted: () => peer.aborted(),
  retire() {
    acceptance.context = undefined;
    retry.context = undefined;
  },
  retireGeneration: () => peer.retireGeneration(),
};
declare global {
  interface Window {
    factoryDecisionsFixture: typeof fixture;
  }
}
window.factoryDecisionsFixture = fixture;
mount('ready', false, true);

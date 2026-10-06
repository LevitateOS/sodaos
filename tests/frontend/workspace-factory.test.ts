import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import type {Space} from '../../frontend/spaces/sodaspaces-inventory-response';
import type {FactoryRun} from '../../frontend/spaces/sodaspaces-factory-response';
import type {FactoryWatchMutation} from '../../frontend/spaces/sodaspaces-workspace-factory';
import type {FactoryWatch} from '../../frontend/spaces/sodaspaces-workspace-types';

const dom = new JSDOM('', {url: 'https://soda.test/'});
for (const [key, value] of Object.entries({
  window: dom.window,
  document: dom.window.document,
  customElements: dom.window.customElements,
  HTMLElement: dom.window.HTMLElement,
  Element: dom.window.Element,
  Node: dom.window.Node,
  Event: dom.window.Event,
  CustomEvent: dom.window.CustomEvent,
})) {
  if (!(key in globalThis)) (globalThis as Record<string, unknown>)[key] = value;
}

const {unwatchRun, watchRun} = await import('../../frontend/spaces/sodaspaces-workspace-factory.js');

function makeWatch(run: string, disposed: string[]): FactoryWatch {
  return {
    run,
    environmentId: 'env',
    repositoryId: 'repo',
    role: 'role',
    hidden: false,
    view: {
      dispose: () => {
        disposed.push(run);
      },
    } as unknown as NonNullable<FactoryWatch['view']>,
  };
}

function makeSpace(): Space {
  return {
    execution_allowed: true,
    authority_unavailable: false,
    environment: {id: 'env', repository_id: 'repo'},
  } as Space;
}

function makeRun(id: string): FactoryRun {
  return {id, role: 'role', reconciled: true};
}

test('closing two watches before rerender removes both without resurrection', () => {
  const disposed: string[] = [];
  let watches = [makeWatch('a', disposed), makeWatch('b', disposed)];
  // Owner wiring mirror: the mutation input reads the live owner list on every call.
  const mutation: FactoryWatchMutation = {
    readWatches: () => watches,
    isStale: () => false,
    pushWatch: (watch) => {
      watches.push(watch);
    },
    replaceWatches: (next) => {
      watches = next;
    },
    updated: () => {},
  };
  unwatchRun(mutation, 'a');
  unwatchRun(mutation, 'b');
  assert.deepEqual(
    watches.map((watch) => watch.run),
    []
  );
  assert.deepEqual(disposed, ['a', 'b']);
});

test('watch and unwatch after removals see the current list', () => {
  const disposed: string[] = [];
  let watches = [makeWatch('a', disposed), makeWatch('b', disposed)];
  const mutation: FactoryWatchMutation = {
    readWatches: () => watches,
    isStale: () => false,
    pushWatch: (watch) => {
      watches.push(watch);
    },
    replaceWatches: (next) => {
      watches = next;
    },
    updated: () => {},
  };
  const space = makeSpace();
  unwatchRun(mutation, 'a');
  watchRun(mutation, space, makeRun('c'));
  assert.deepEqual(
    watches.map((watch) => watch.run),
    ['b', 'c']
  );
  unwatchRun(mutation, 'b');
  unwatchRun(mutation, 'c');
  assert.deepEqual(
    watches.map((watch) => watch.run),
    []
  );
});

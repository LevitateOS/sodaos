import test from 'node:test';
import assert from 'node:assert/strict';
import {nativeAppSubUrl} from '../../frontend/spaces/soda-native-paths.ts';
import {prepareExtensionMount} from '../../frontend/spaces/soda-extension.ts';
import type {ExtensionMountContext} from '../../frontend/spaces/soda-extension.ts';

const origin = 'https://forgejo.example.test';
const base: ExtensionMountContext = {
  extensionId: 'soda',
  pageId: 'spaces',
  appSubUrl: '',
  apiBase: origin + '/-/native/pages/soda/spaces/api/',
  assetBase: origin + '/-/native/assets/soda/',
  sessionGeneration: 'native-generation',
};

test('native links use the supplied installation subpath independently of private API routing', () => {
  assert.equal(nativeAppSubUrl(base), '');
  assert.equal(nativeAppSubUrl({...base, appSubUrl: '/forgejo'}), '/forgejo');
  assert.equal(nativeAppSubUrl({...base, appSubUrl: '/forgejo', apiBase: origin + '/other/private/api/'}), '/forgejo');
  for (const appSubUrl of [undefined, '/foo%2fbar', '/foo%5cbar', '//other', '/forgejo/', '/.', '/a/..', '/a?q=1'])
    assert.throws(() => nativeAppSubUrl({...base, appSubUrl: appSubUrl as string}));
});

test('native transport retains same-origin and observed repository identity validation', () => {
  const prior = Object.getOwnPropertyDescriptor(globalThis, 'location');
  Object.defineProperty(globalThis, 'location', {configurable: true, value: new URL(origin)});
  try {
    const root = {} as HTMLElement;
    const transport = prepareExtensionMount(root, {...base, repositoryId: '8'});
    assert.equal(transport.generation, base.sessionGeneration);
    transport.dispose();
    for (const apiBase of [
      'https://other.example.test/-/native/pages/soda/spaces/api/',
      origin + '/-/native/pages/soda/spaces/api/?redirect=/repo/create',
    ])
      assert.throws(() => prepareExtensionMount(root, {...base, apiBase}));
    assert.throws(() => prepareExtensionMount(root, {...base, repositoryId: '0'}));
  } finally {
    if (prior) Object.defineProperty(globalThis, 'location', prior);
    else Reflect.deleteProperty(globalThis, 'location');
  }
});

test('host-scoped transport is preferred and unexpected contract versions refuse before effects', async () => {
  const prior = Object.getOwnPropertyDescriptor(globalThis, 'location');
  Object.defineProperty(globalThis, 'location', {configurable: true, value: new URL(origin)});
  try {
    const root = {} as HTMLElement;
    assert.throws(() => prepareExtensionMount(root, {...base, contractVersion: 2 as 1}));
    const calls: string[] = [];
    const socket = {close() {}} as unknown as WebSocket;
    const transport = prepareExtensionMount(root, {
      ...base,
      contractVersion: 1,
      request: async (path: string) => {
        calls.push(path);
        return new Response('{}', {status: 200});
      },
      stream: (path: string) => {
        calls.push(path);
        return {socket, close: () => socket.close()};
      },
    });
    const response = await transport.request('projects', {method: 'GET'});
    assert.equal(response.status, 200);
    assert.equal(transport.websocket('environments/1/terminal'), socket);
    assert.deepEqual(calls, ['projects', 'environments/1/terminal']);
    transport.dispose();
    await assert.rejects(() => transport.request('projects'));
  } finally {
    if (prior) Object.defineProperty(globalThis, 'location', prior);
    else Reflect.deleteProperty(globalThis, 'location');
  }
});

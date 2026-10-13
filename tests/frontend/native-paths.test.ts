import test from 'node:test';
import assert from 'node:assert/strict';
import {nativeAppSubUrl} from '../../frontend/spaces/soda-native-paths.ts';
import {prepareExtensionMount} from '../../frontend/spaces/soda-extension.ts';
import type {ExtensionMountContext} from '../../frontend/spaces/soda-extension.ts';

const origin = 'https://forgejo.example.test';
const hostSocket = {close() {}} as unknown as WebSocket;
const base: ExtensionMountContext = {
  extensionId: 'soda',
  pageId: 'spaces',
  appSubUrl: '',
  apiBase: origin + '/-/native/pages/soda/spaces/api/',
  assetBase: origin + '/-/native/assets/soda/',
  sessionGeneration: 'native-generation',
  contractVersion: 1,
  request: async () => new Response('{}', {status: 200}),
  stream: () => ({socket: hostSocket, close: () => hostSocket.close()}),
  navigate: () => {},
};

test('native links use the supplied installation subpath independently of private API routing', () => {
  assert.equal(nativeAppSubUrl(base), '');
  assert.equal(nativeAppSubUrl({...base, appSubUrl: '/forgejo'}), '/forgejo');
  assert.equal(nativeAppSubUrl({...base, appSubUrl: '/forgejo', apiBase: origin + '/other/private/api/'}), '/forgejo');
  for (const appSubUrl of [undefined, '/foo%2fbar', '/foo%5cbar', '//other', '/forgejo/', '/.', '/a/..', '/a?q=1'])
    assert.throws(() => nativeAppSubUrl({...base, appSubUrl: appSubUrl as string}));
});

test('native transport retains Soda domain validation and delegates generic scoping to the host', () => {
  const root = {} as HTMLElement;
  const transport = prepareExtensionMount(root, {...base, repositoryId: '8'});
  assert.equal(transport.generation, base.sessionGeneration);
  assert.equal(transport.assetBase, base.assetBase);
  transport.dispose();
  // Generic URL/session scoping (origin, query, encoded separators) is the
  // host's scoped helpers' job (H-N); Soda does not duplicate it for apiBase.
  const delegated = prepareExtensionMount(root, {
    ...base,
    apiBase: 'https://other.example.test/-/native/pages/soda/spaces/api/?redirect=/repo/create',
  });
  delegated.dispose();
  assert.throws(() => prepareExtensionMount(root, {...base, repositoryId: '0'}));
  assert.throws(() => prepareExtensionMount(root, {...base, extensionId: 'other'}));
  assert.throws(() => prepareExtensionMount(root, {...base, ownerId: 'other'}));
  assert.throws(() => prepareExtensionMount(root, {...base, sessionGeneration: ''}));
});

test('host-scoped transport is required and unexpected contract versions refuse before effects', async () => {
  const root = {} as HTMLElement;
  assert.throws(() => prepareExtensionMount(root, {...base, contractVersion: 2 as 1}));
  assert.throws(() => prepareExtensionMount(root, {...base, request: undefined as never}));
  assert.throws(() => prepareExtensionMount(root, {...base, stream: undefined as never}));
  assert.throws(() => prepareExtensionMount(root, {...base, navigate: undefined as never}));
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
    navigate: (path: string) => {
      calls.push(path);
    },
  });
  const response = await transport.request('projects', {method: 'GET'});
  assert.equal(response.status, 200);
  assert.equal(transport.websocket('environments/1/terminal'), socket);
  transport.navigate?.('/workspace/new');
  assert.deepEqual(calls, ['projects', 'environments/1/terminal', '/workspace/new']);
  transport.dispose();
  await assert.rejects(() => transport.request('projects'));
  assert.throws(() => transport.websocket('environments/1/terminal'));
  assert.throws(() => transport.navigate?.('/workspace'));
});

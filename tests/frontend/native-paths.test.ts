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

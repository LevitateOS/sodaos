import test from 'node:test';
import assert from 'node:assert/strict';
import {forgejoPrefix} from '../../frontend/spaces/soda-native-paths.ts';
import type {ExtensionMountContext} from '../../frontend/spaces/soda-extension.ts';

const origin = 'https://forgejo.example.test';
const base: ExtensionMountContext = {
  extensionId: 'soda',
  pageId: 'spaces',
  apiBase: origin + '/-/extensions/pages/soda/spaces/api/',
  assetBase: origin + '/-/extensions/pages/soda/spaces/assets/',
  sessionGeneration: 'native-generation',
};

test('native page and panel links retain the Forgejo installation prefix', () => {
  const prior = Object.getOwnPropertyDescriptor(globalThis, 'location');
  Object.defineProperty(globalThis, 'location', {configurable: true, value: new URL(origin)});
  try {
    assert.equal(forgejoPrefix(base), '');
    assert.equal(
      forgejoPrefix({...base, apiBase: origin + '/forgejo/-/extensions/pages/soda/spaces/api/'}),
      '/forgejo'
    );
    assert.equal(
      forgejoPrefix({
        extensionId: 'soda',
        panelId: 'workspace',
        apiBase: origin + '/forgejo/-/extensions/panels/soda/workspace/api/',
        assetBase: base.assetBase,
        sessionGeneration: base.sessionGeneration,
      }),
      '/forgejo'
    );
    for (const apiBase of [
      'https://other.example.test/-/extensions/pages/soda/spaces/api/',
      origin + '/-/extensions/pages/soda/runners/api/',
      origin + '/foo%2fbar/-/extensions/pages/soda/spaces/api/',
      origin + '/-/extensions/pages/soda/spaces/api/?redirect=/repo/create',
    ]) {
      assert.throws(() => forgejoPrefix({...base, apiBase}));
    }
  } finally {
    if (prior) Object.defineProperty(globalThis, 'location', prior);
    else Reflect.deleteProperty(globalThis, 'location');
  }
});

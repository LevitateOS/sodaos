// Isolated lifecycle probe for the component-browser factory. Never loaded by a
// shared suite process: the sibling .test.ts spawns exactly this file under
// `bun test` with SODA_FORGEJO_LAYOUT_ORIGIN set, so the origin-gated factory
// and the playwright mock cannot leak into other suites.
import test from 'node:test';
import assert from 'node:assert/strict';
import {mock} from 'bun:test';

assert.equal(process.env.SODA_FORGEJO_LAYOUT_ORIGIN, 'http://localhost:3300');

const behavior = {
  newPageError: null as unknown,
  closeError: null as unknown,
  closeCalls: 0,
};
const fakePage = {};
const fakeBrowser = {
  newPage: async () => {
    if (behavior.newPageError) throw behavior.newPageError;
    return fakePage;
  },
  close: async () => {
    behavior.closeCalls++;
    if (behavior.closeError) throw behavior.closeError;
  },
};
mock.module('playwright', () => ({chromium: {launch: async () => fakeBrowser}}));

const {openComponentBrowser} = await import('./component-browser');

test('factory closes the browser when page setup fails', async () => {
  behavior.newPageError = Error('no page');
  behavior.closeError = null;
  behavior.closeCalls = 0;
  await assert.rejects(openComponentBrowser(), /no page/);
  assert.equal(behavior.closeCalls, 1);
});

test('factory preserves the setup error when close also fails', async () => {
  behavior.newPageError = Error('no page');
  behavior.closeError = Error('close broke');
  behavior.closeCalls = 0;
  await assert.rejects(openComponentBrowser(), /no page/);
  assert.equal(behavior.closeCalls, 1);
});

test('successful handoff leaves close ownership to the caller', async () => {
  behavior.newPageError = null;
  behavior.closeError = null;
  behavior.closeCalls = 0;
  const fixture = await openComponentBrowser();
  assert.equal(typeof fixture.render, 'function');
  assert.equal(behavior.closeCalls, 0);
  await fixture.close();
  assert.equal(behavior.closeCalls, 1);
});

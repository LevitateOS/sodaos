import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';
import {test} from 'node:test';
import assert from 'node:assert/strict';

const source = readFileSync(new URL('../../.artifacts/forgejo-js/login-theme.js', import.meta.url), 'utf8');
function page({stored = null, dark = false, blocked = false, prefix = '', buttonPresent = true}: {stored?: string | null; dark?: boolean; blocked?: boolean; prefix?: string; buttonPresent?: boolean} = {}) {
  type Callback = (...args: unknown[]) => void;
  const events: Record<string, Callback> = {}, clicks: Record<string, Callback> = {}, systemEvents: Record<string, Callback> = {}, attributes: Record<string, string> = {};
  let domReady = false;
  const root: {dataset: {theme: string; sodaLoginTheme?: string}} = {dataset: {theme: 'forgejo-auto'}};
  const button = {hidden: true, setAttribute: (key: string, value: string) => { attributes[key] = value; }, addEventListener: (key: string, fn: Callback) => {clicks[key] = fn;}};
  const system = {matches: dark, addEventListener: (key: string, fn: Callback) => {systemEvents[key] = fn;}};
  const storage = new Map([[`soda.login.theme:${prefix || '/'}`, stored]]);
  runInNewContext(source, {
    document: {
      documentElement: root,
      currentScript: {dataset: {appSubUrl: prefix}},
      getElementById: () => domReady && buttonPresent ? button : null,
      addEventListener: (key: string, fn: Callback) => {events[key] = (...args: unknown[]) => {domReady = true; return fn(...args);};},
    },
    window: {matchMedia: () => system, addEventListener: (key: string, fn: Callback) => {events[key] = fn;}},
    localStorage: {getItem: (key: string) => {if (blocked) throw Error('blocked'); return storage.get(key);}, setItem: (key: string, value: string) => {if (blocked) throw Error('blocked'); storage.set(key, value);}},
  });
  return {root, button, attributes, storage, events, clicks, system, systemEvents};
}
test('system preference applies before DOM readiness and follows system changes', () => {
  const p = page({dark: true});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  assert(p.systemEvents.change);
  p.system.matches = false; p.systemEvents.change();
  assert.equal(p.root.dataset.sodaLoginTheme, 'light');
  assert.equal(p.root.dataset.theme, 'forgejo-auto');
});
test('explicit preference wins over the system', () => {
  const p = page({stored: 'light', dark: true});
  assert(p.systemEvents.change);
  p.systemEvents.change();
  assert.equal(p.root.dataset.sodaLoginTheme, 'light');
});
test('toggle persists the next mode and exposes its next action', () => {
  const p = page();
  assert.equal(p.button.hidden, true);
  assert(p.events.DOMContentLoaded);
  p.events.DOMContentLoaded();
  assert(p.clicks.click);
  p.clicks.click();
  assert.equal(p.storage.get('soda.login.theme:/'), 'dark');
  assert.equal(p.attributes['aria-label'], 'Switch to light theme');
  assert.equal(p.button.hidden, false);
  assert(p.clicks.click);
  p.clicks.click();
  assert.equal(p.attributes['aria-label'], 'Switch to dark theme');
});
test('pages without a guest toggle still apply the pre-paint preference safely', () => {
  const p = page({dark: true, buttonPresent: false});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  assert(p.events.DOMContentLoaded);
  p.events.DOMContentLoaded();
  assert.deepEqual(p.clicks, {});
});
test('blocked storage still allows an in-memory choice', () => {
  const p = page({blocked: true});
  assert(p.events.DOMContentLoaded);
  p.events.DOMContentLoaded();
  assert(p.clicks.click);
  p.clicks.click();
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
});
test('cross-tab changes synchronize; clearing preference restores system mode', () => {
  const p = page();
  assert(p.events.storage);
  p.events.storage({key: 'soda.login.theme:/', newValue: 'dark'});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  assert(p.events.storage);
  p.events.storage({key: 'unrelated', newValue: 'light'});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  assert(p.events.storage);
  p.events.storage({key: null, newValue: null});
  assert.equal(p.root.dataset.sodaLoginTheme, 'light');
});
test('invalid stored values fall back safely and app subpaths have distinct keys', () => {
  const p = page({stored: 'invalid', dark: true, prefix: '/forge'});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  assert(p.events.DOMContentLoaded);
  p.events.DOMContentLoaded();
  assert(p.clicks.click);
  p.clicks.click();
  assert.equal(p.storage.get('soda.login.theme:/forge'), 'light');
});

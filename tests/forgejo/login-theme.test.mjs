import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';
import {test} from 'node:test';
import assert from 'node:assert/strict';

const source = readFileSync(new URL('../../assets/branding/forgejo/login-theme.js', import.meta.url), 'utf8');
function page({stored = null, dark = false, blocked = false, prefix = ''} = {}) {
  const events = {}, clicks = {}, systemEvents = {}, attributes = {};
  const root = {dataset: {theme: 'forgejo-auto'}};
  const button = {hidden: true, setAttribute: (key, value) => { attributes[key] = value; }, addEventListener: (key, fn) => {clicks[key] = fn;}};
  const system = {matches: dark, addEventListener: (key, fn) => {systemEvents[key] = fn;}};
  const storage = new Map([[`soda.login.theme:${prefix || '/'}`, stored]]);
  runInNewContext(source, {
    document: {documentElement: root, currentScript: {dataset: {appSubUrl: prefix}}, getElementById: () => button, addEventListener: (key, fn) => {events[key] = fn;}},
    window: {matchMedia: () => system, addEventListener: (key, fn) => {events[key] = fn;}},
    localStorage: {getItem: key => {if (blocked) throw Error('blocked'); return storage.get(key);}, setItem: (key, value) => {if (blocked) throw Error('blocked'); storage.set(key, value);}},
  });
  return {root, button, attributes, storage, events, clicks, system, systemEvents};
}
test('system preference applies before DOM readiness and follows system changes', () => {
  const p = page({dark: true});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  p.system.matches = false; p.systemEvents.change();
  assert.equal(p.root.dataset.sodaLoginTheme, 'light');
  assert.equal(p.root.dataset.theme, 'forgejo-auto');
});
test('explicit preference wins over the system', () => {
  const p = page({stored: 'light', dark: true});
  p.systemEvents.change();
  assert.equal(p.root.dataset.sodaLoginTheme, 'light');
});
test('toggle persists the next mode and exposes its next action', () => {
  const p = page(); p.events.DOMContentLoaded(); p.clicks.click();
  assert.equal(p.storage.get('soda.login.theme:/'), 'dark');
  assert.equal(p.attributes['aria-label'], 'Switch to light theme');
  assert.equal(p.button.hidden, false);
  p.clicks.click();
  assert.equal(p.attributes['aria-label'], 'Switch to dark theme');
});
test('blocked storage still allows an in-memory choice', () => {
  const p = page({blocked: true}); p.events.DOMContentLoaded(); p.clicks.click();
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
});
test('cross-tab changes synchronize; clearing preference restores system mode', () => {
  const p = page();
  p.events.storage({key: 'soda.login.theme:/', newValue: 'dark'});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  p.events.storage({key: 'unrelated', newValue: 'light'});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  p.events.storage({key: null, newValue: null});
  assert.equal(p.root.dataset.sodaLoginTheme, 'light');
});
test('invalid stored values fall back safely and app subpaths have distinct keys', () => {
  const p = page({stored: 'invalid', dark: true, prefix: '/forge'});
  assert.equal(p.root.dataset.sodaLoginTheme, 'dark');
  p.events.DOMContentLoaded(); p.clicks.click();
  assert.equal(p.storage.get('soda.login.theme:/forge'), 'light');
});

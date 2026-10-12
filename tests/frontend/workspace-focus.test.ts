import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';

const dom = new JSDOM('', {url: 'https://soda.test/'});
for (const [key, value] of Object.entries({
  document: dom.window.document,
  HTMLElement: dom.window.HTMLElement,
  Element: dom.window.Element,
  Node: dom.window.Node,
})) {
  if (!(key in globalThis)) (globalThis as Record<string, unknown>)[key] = value;
}

const {rememberFocus, restoreFocus} = await import('../../frontend/spaces/sodaspaces-workspace-focus.js');

function mount() {
  document.body.innerHTML = '';
  const root = document.createElement('div');
  root.innerHTML =
    '<details class="soda-menu" open><summary aria-label="Terminal actions">...</summary>' +
    '<div><button type="button">Rename terminal</button></div></details>' +
    '<div class="soda-workspace-dialog"><input></div>';
  document.body.append(root);
  return {
    root,
    summary: root.querySelector<HTMLElement>('summary')!,
    item: root.querySelector<HTMLButtonElement>('.soda-menu button')!,
    input: root.querySelector<HTMLElement>('.soda-workspace-dialog input')!,
  };
}

function stubRects(node: HTMLElement) {
  Object.defineProperty(node, 'getClientRects', {configurable: true, value: () => [{x: 0}]});
}

test('remembering focus inside an actions menu keeps the persistent menu control', () => {
  const {root, summary, item} = mount();
  item.focus();
  assert.equal(document.activeElement, item);
  let invoker: HTMLElement | undefined;
  rememberFocus({root, setInvoker: (value) => (invoker = value)});
  assert.equal(invoker, summary);
});

test('remembering focus with nothing focused keeps no invoker', () => {
  const {root} = mount();
  assert.equal(document.activeElement, document.body);
  let invoker: HTMLElement | undefined = root;
  rememberFocus({root, setInvoker: (value) => (invoker = value)});
  assert.equal(invoker, undefined);
});

test('rename cancel restores focus to the terminal actions control', async () => {
  const {root, summary, item, input} = mount();
  item.focus();
  let invoker: HTMLElement | undefined;
  rememberFocus({root, setInvoker: (value) => (invoker = value)});
  input.focus();
  root.querySelector('.soda-workspace-dialog')!.remove();
  assert.equal(document.activeElement, document.body);
  stubRects(summary);
  restoreFocus({
    invoker,
    updateComplete: Promise.resolve(true),
    isStale: () => false,
    isActiveSurface: () => true,
    root,
  });
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(document.activeElement, summary);
});

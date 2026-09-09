import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import test from 'node:test';
import {JSDOM} from 'jsdom';

const script = await readFile(new URL('../../.artifacts/forgejo-js/repository-actions.js', import.meta.url), 'utf8');

function fixture() {
  const dom = new JSDOM(`<!doctype html><body>
    <details class="soda-repository-actions" open>
      <summary>Repository actions</summary>
      <div class="soda-repository-action-menu">
        <form><button id="watch" type="button">Watch</button></form>
        <div class="ui modal"><button id="modal-action" type="button">Modal action</button></div>
      </div>
    </details>
    <button id="outside" type="button">Outside</button>
  </body>`, {runScripts: 'outside-only'});
  dom.window.eval(script);
  const details = dom.window.document.querySelector<HTMLDetailsElement>('.soda-repository-actions');
  const summary = dom.window.document.querySelector<HTMLElement>('summary');
  const watch = dom.window.document.getElementById('watch');
  const modalAction = dom.window.document.getElementById('modal-action');
  const outside = dom.window.document.getElementById('outside');
  assert(details && summary && watch && modalAction && outside);
  return {dom, details, summary, watch, modalAction, outside};
}

test('Escape closes the disclosure and restores its summary focus once', () => {
  const f = fixture();
  let focuses = 0;
  f.summary.focus = () => { focuses++; };
  f.dom.window.eval(script);
  const event = new f.dom.window.KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
  f.watch.dispatchEvent(event);
  assert.equal(f.details.open, false);
  assert.equal(event.defaultPrevented, true);
  assert.equal(focuses, 1, 'enhancement remains idempotent');
  f.dom.window.close();
});

test('outside click closes without intercepting the click or stealing focus', () => {
  const f = fixture();
  f.outside.focus();
  const event = new f.dom.window.MouseEvent('click', {bubbles: true, cancelable: true});
  f.outside.dispatchEvent(event);
  assert.equal(f.details.open, false);
  assert.equal(event.defaultPrevented, false);
  assert.equal(f.dom.window.document.activeElement, f.outside);
  f.dom.window.close();
});

test('focusout closes only for a real focus destination outside', () => {
  const f = fixture();
  f.watch.dispatchEvent(new f.dom.window.FocusEvent('focusout', {bubbles: true, relatedTarget: null}));
  assert.equal(f.details.open, true, 'window blur and replaced controls keep the disclosure open');
  f.watch.dispatchEvent(new f.dom.window.FocusEvent('focusout', {bubbles: true, relatedTarget: f.outside}));
  assert.equal(f.details.open, false);
  assert.notEqual(f.dom.window.document.activeElement, f.summary);
  f.dom.window.close();
});

test('HTMX-like form replacement can restore focus without closing', () => {
  const f = fixture();
  f.watch.focus();
  f.watch.dispatchEvent(new f.dom.window.FocusEvent('focusout', {bubbles: true, relatedTarget: null}));
  const oldForm = f.watch.closest('form');
  assert(oldForm);
  const replacement = f.dom.window.document.createElement('form');
  replacement.innerHTML = '<button id="unwatch" type="button">Unwatch</button>';
  oldForm.replaceWith(replacement);
  const unwatch = f.dom.window.document.getElementById('unwatch');
  assert(unwatch);
  unwatch.focus();
  assert.equal(f.details.open, true);
  assert.equal(f.dom.window.document.activeElement, unwatch);
  f.dom.window.close();
});

test('native modal and action events remain untouched', () => {
  const f = fixture();
  const modalEscape = new f.dom.window.KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
  f.modalAction.dispatchEvent(modalEscape);
  assert.equal(f.details.open, true);
  assert.equal(modalEscape.defaultPrevented, false);

  const actionClick = new f.dom.window.MouseEvent('click', {bubbles: true, cancelable: true});
  f.watch.dispatchEvent(actionClick);
  assert.equal(actionClick.defaultPrevented, false);
  assert.equal(f.details.open, true);
  f.dom.window.close();
});

import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import test from 'node:test';
import {JSDOM} from 'jsdom';

const script = await readFile(new URL('../../.artifacts/forgejo-js/repository-actions.js', import.meta.url), 'utf8');

function fixture(initialCompact = true) {
  const dom = new JSDOM(`<!doctype html><body>
    <div class="soda-repository-header">
      <details class="soda-repository-actions" open>
        <summary>Repository actions</summary>
        <div class="soda-repository-action-menu">
          <button id="unavailable" type="button" disabled>Unavailable</button>
          <a id="feed" href="/repo.rss">Feed</a>
          <form><button id="watch" type="button">Watch</button></form>
          <div class="ui modal"><button id="modal-action" type="button">Modal action</button></div>
        </div>
      </details>
    </div>
    <button id="outside" type="button">Outside</button>
  </body>`, {runScripts: 'outside-only'});
  const doc = dom.window.document;
  const details = doc.querySelector<HTMLDetailsElement>('.soda-repository-actions');
  const summary = doc.querySelector<HTMLElement>('summary');
  const panel = doc.querySelector<HTMLElement>('.soda-repository-action-menu');
  const feed = doc.getElementById('feed');
  const watch = doc.getElementById('watch');
  const modalAction = doc.getElementById('modal-action');
  const outside = doc.getElementById('outside');
  const header = doc.querySelector<HTMLElement>('.soda-repository-header');
  assert(details && summary && panel && feed && watch && modalAction && outside && header);

  let compact = initialCompact;
  const nativeGetComputedStyle = dom.window.getComputedStyle.bind(dom.window);
  Object.defineProperty(dom.window, 'getComputedStyle', {value: (element: Element) => {
    if (element === summary) return {display: compact ? 'flex' : 'none'};
    return nativeGetComputedStyle(element);
  }});
  let resize: (() => void) | undefined;
  let observed: Element | undefined;
  class TestResizeObserver {
    constructor(callback: () => void) { resize = callback; }
    observe(target: Element) { observed = target; }
  }
  Object.defineProperty(dom.window, 'ResizeObserver', {value: TestResizeObserver});
  dom.window.eval(script);
  assert.equal(observed, header);
  return {
    dom, details, summary, panel, feed, watch, modalAction, outside,
    setCompact(value: boolean, beforeResize?: () => void) {
      compact = value;
      beforeResize?.();
      assert(resize);
      resize();
    },
  };
}

test('initial mode follows CSS and wide actions ignore dismissal', () => {
  const compact = fixture(true);
  assert.equal(compact.details.open, false, 'initial compact disclosure is closed');
  compact.dom.window.close();

  const wide = fixture(false);
  assert.equal(wide.details.open, true, 'wide no-JS default remains open');
  wide.outside.focus();
  wide.outside.click();
  wide.watch.dispatchEvent(new wide.dom.window.FocusEvent('focusout', {bubbles: true, relatedTarget: wide.outside}));
  const escape = new wide.dom.window.KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
  wide.watch.dispatchEvent(escape);
  assert.equal(wide.details.open, true);
  assert.equal(escape.defaultPrevented, false);
  wide.dom.window.close();
});

test('mode transitions open wide actions and preserve a focused action on compaction', () => {
  const f = fixture(true);
  f.setCompact(false);
  assert.equal(f.details.open, true);
  f.watch.focus();
  f.setCompact(true);
  assert.equal(f.details.open, true, 'resize must not hide the focused native action');
  assert.equal(f.dom.window.document.activeElement, f.watch);
  f.outside.focus();
  assert.equal(f.details.open, false, 'leaving the compact disclosure dismisses it');
  f.setCompact(false);
  assert.equal(f.details.open, true);
  f.setCompact(true);
  assert.equal(f.details.open, false, 'ordinary wide-to-compact transition closes');
  f.dom.window.close();
});

test('expanding to wide from its hidden summary focuses the first available native action', () => {
  const f = fixture(true);
  f.summary.focus();
  f.setCompact(false, () => {
    f.summary.blur();
    assert.equal(f.dom.window.document.activeElement, f.dom.window.document.body);
  });
  assert.equal(f.dom.window.document.activeElement, f.feed);
  assert.equal(f.details.open, true);
  f.dom.window.close();
});

test('Escape closes compact actions and setup remains idempotent', () => {
  const f = fixture(true);
  f.details.open = true;
  let focuses = 0;
  f.summary.focus = () => { focuses++; };
  f.dom.window.eval(script);
  const event = new f.dom.window.KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
  f.watch.dispatchEvent(event);
  assert.equal(f.details.open, false);
  assert.equal(event.defaultPrevented, true);
  assert.equal(focuses, 1);
  f.dom.window.close();
});

test('outside click and focusout close compact actions without stealing focus', () => {
  const f = fixture(true);
  f.details.open = true;
  f.outside.focus();
  const event = new f.dom.window.MouseEvent('click', {bubbles: true, cancelable: true});
  f.outside.dispatchEvent(event);
  assert.equal(f.details.open, false);
  assert.equal(event.defaultPrevented, false);
  assert.equal(f.dom.window.document.activeElement, f.outside);

  f.details.open = true;
  f.watch.dispatchEvent(new f.dom.window.FocusEvent('focusout', {bubbles: true, relatedTarget: null}));
  assert.equal(f.details.open, true);
  f.watch.dispatchEvent(new f.dom.window.FocusEvent('focusout', {bubbles: true, relatedTarget: f.outside}));
  assert.equal(f.details.open, false);
  assert.notEqual(f.dom.window.document.activeElement, f.summary);
  f.dom.window.close();
});

test('HTMX-like form replacement restores focus without closing compact actions', () => {
  const f = fixture(true);
  f.details.open = true;
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

test('native modal and action events remain untouched in compact mode', () => {
  const f = fixture(true);
  f.details.open = true;
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

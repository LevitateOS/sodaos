import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import test from 'node:test';
import {JSDOM} from 'jsdom';

const scripts = new Map(
  await Promise.all(
    ['repository-switcher', 'personal-settings'].map(
      async (name) =>
        [name, await readFile(new URL(`../../.artifacts/forgejo-js/${name}.js`, import.meta.url), 'utf8')] as const
    )
  )
);

function run(dom: JSDOM, name: string): void {
  const script = scripts.get(name);
  assert(script);
  dom.window.eval('(function(){' + script + '\n})()');
}

function browser() {
  const dom = new JSDOM('<!doctype html><body><button id="outside">Outside</button></body>', {
    url: 'https://fixture.invalid/',
    runScripts: 'outside-only',
    pretendToBeVisual: true,
  });
  Object.defineProperty(dom.window.document, 'readyState', {value: 'complete', configurable: true});
  Object.defineProperty(dom.window, 'matchMedia', {
    value: () => Object.assign(new dom.window.EventTarget(), {matches: true}),
  });
  Object.defineProperty(dom.window.HTMLElement.prototype, 'scrollIntoView', {value: () => {}});
  Object.defineProperty(dom.window.HTMLElement.prototype, 'checkVisibility', {value: () => true});
  const frames = new Map<number, FrameRequestCallback>();
  let frameID = 0;
  Object.defineProperty(dom.window, 'requestAnimationFrame', {
    value: (callback: FrameRequestCallback) => {
      frames.set(++frameID, callback);
      return frameID;
    },
  });
  Object.defineProperty(dom.window, 'cancelAnimationFrame', {value: (id: number) => frames.delete(id)});
  return {dom, frames};
}

function append(dom: JSDOM, markup: string): HTMLElement {
  const template = dom.window.document.createElement('template');
  template.innerHTML = markup;
  const element = template.content.firstElementChild;
  assert(element instanceof dom.window.HTMLElement);
  dom.window.document.body.append(element);
  return element;
}

function switcherMarkup(repo: string): string {
  return `<soda-repository-switcher class="soda-repository-breadcrumb" data-user-id="1" data-owner-id="2" data-owner-name="sample" data-repo-name="${repo}" data-repo-link="/sample/${repo}">
    <div class="soda-repository-switcher" data-kind="repository" hidden><div class="menu"><input class="search"><div role="status"></div></div></div>
    <template data-symbol="repo"><svg></svg></template><template data-symbol="check"><svg></svg></template>
  </soda-repository-switcher>`;
}

test('late native switchers own Fomantic, pending requests and frames until disconnect', async () => {
  const {dom, frames} = browser();
  const commands: string[] = [];
  let setups = 0,
    disconnects = 0;
  let show: (() => void) | undefined;
  Object.defineProperty(dom.window, 'jQuery', {
    value: () => ({
      dropdown(setting: string | Record<string, unknown>) {
        if (typeof setting === 'string') {
          commands.push(setting);
          return;
        }
        const callback = setting.onShow;
        assert.equal(typeof callback, 'function');
        if (typeof callback === 'function')
          show = () => {
            callback();
          };
        setups++;
      },
    }),
  });
  Object.defineProperty(dom.window, 'ResizeObserver', {
    value: class {
      observe() {}
      disconnect() {
        disconnects++;
      }
    },
  });
  let signal: AbortSignal | null | undefined;
  let finishBody: ((body: string) => void) | undefined;
  Object.defineProperty(dom.window, 'fetch', {
    value: async (_url: string, options: RequestInit) => {
      signal = options.signal;
      return {
        ok: true,
        redirected: false,
        text: () =>
          new Promise<string>((resolve) => {
            finishBody = resolve;
          }),
      };
    },
  });
  run(dom, 'repository-switcher');
  const owner = append(dom, switcherMarkup('first'));
  const picker = owner.querySelector<HTMLElement>('[data-kind="repository"]');
  assert(picker && show);
  assert.equal(picker.hidden, false);
  picker.classList.add('active');
  show();
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert(signal && finishBody);
  assert.equal(frames.size, 1);
  owner.remove();
  assert.equal(signal.aborted, true);
  assert.equal(frames.size, 0);
  assert.equal(disconnects, 1);
  assert(commands.includes('destroy'));
  finishBody(JSON.stringify({ok: true, data: [{repository: {full_name: 'sample/late', link: '/sample/late'}}]}));
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(owner.querySelectorAll('a.item').length, 0, 'late response cannot populate a retired picker');
  dom.window.document.body.append(owner);
  assert.equal(setups, 2, 'reconnecting owns one fresh dropdown');
  owner.remove();
  const next = append(dom, switcherMarkup('second'));
  assert.equal(setups, 3, 'a later repository gets its own owner');
  assert.equal(next.dataset.repoName, 'second');
  next.remove();
  dom.window.close();
});

test('initial switcher boot waits for document readiness and disconnect cancels that wait', () => {
  const {dom} = browser();
  Object.defineProperty(dom.window.document, 'readyState', {value: 'loading', configurable: true});
  let setups = 0;
  Object.defineProperty(dom.window, 'jQuery', {
    value: () => ({
      dropdown(setting: unknown) {
        if (typeof setting === 'object') setups++;
      },
    }),
  });
  Object.defineProperty(dom.window, 'ResizeObserver', {
    value: class {
      observe() {}
      disconnect() {}
    },
  });
  const initial = append(dom, switcherMarkup('initial'));
  run(dom, 'repository-switcher');
  assert.equal(setups, 0);
  initial.remove();
  dom.window.document.dispatchEvent(new dom.window.Event('DOMContentLoaded'));
  assert.equal(setups, 0, 'retired initial markup cannot initialize after readiness');
  Object.defineProperty(dom.window.document, 'readyState', {value: 'complete'});
  const next = append(dom, switcherMarkup('entered'));
  assert.equal(setups, 1, 'boot definitions remain ready for later native entries');
  next.remove();
  dom.window.close();
});

function settingsMarkup(error = false): string {
  return `<soda-personal-settings class="page-content soda-settings-shell">
    <nav class="soda-settings-nav"><button class="soda-settings-current">Current</button><section class="soda-settings-nav-group"><button>Account</button><a href="/user/settings" aria-current="page">Profile</a></section></nav>
    ${error ? '<div class="ui error message">Server error</div>' : ''}
    <a class="soda-avatar-trigger" href="#avatar-settings">Avatar</a>
    <details id="avatar-settings" data-settings-editor open><summary>Avatar</summary><div class="soda-settings-disclosure-body"><form><input name="avatar"></form></div></details>
    <dialog id="avatar-dialog"><button class="soda-avatar-close">Close</button></dialog>
    <details id="identity-settings" data-settings-editor open><summary>Identity</summary><input name="name"></details>
    <button class="show-panel" data-panel="#panel">Show</button><div id="panel"><input name="panel"></div>
  </soda-personal-settings>`;
}

test('native settings entry and disconnect own nav, dialog, hash and deferred focus without duplicating forms', () => {
  const {dom, frames} = browser();
  Object.defineProperty(dom.window.HTMLDialogElement.prototype, 'showModal', {
    value: function (this: HTMLDialogElement) {
      this.open = true;
    },
  });
  Object.defineProperty(dom.window.HTMLDialogElement.prototype, 'close', {
    value: function (this: HTMLDialogElement) {
      this.open = false;
      this.dispatchEvent(new dom.window.Event('close'));
    },
  });
  run(dom, 'personal-settings');
  const owner = append(dom, settingsMarkup());
  const nav = owner.querySelector<HTMLElement>('.soda-settings-nav');
  const current = owner.querySelector<HTMLButtonElement>('.soda-settings-current');
  const trigger = owner.querySelector<HTMLElement>('.soda-avatar-trigger');
  const dialog = owner.querySelector<HTMLDialogElement>('dialog');
  const avatar = owner.querySelector<HTMLDetailsElement>('#avatar-settings');
  const identity = owner.querySelector<HTMLDetailsElement>('#identity-settings');
  const panelButton = owner.querySelector<HTMLButtonElement>('.show-panel');
  assert(nav && current && trigger && dialog && avatar && identity && panelButton);
  assert(nav.classList.contains('is-enhanced'));
  assert.equal(owner.querySelectorAll('form').length, 1);
  assert(dialog.querySelector('form'));
  current.click();
  assert.equal(current.getAttribute('aria-expanded'), 'true');
  trigger.click();
  assert.equal(dialog.open, true);
  panelButton.click();
  assert.equal(frames.size, 1);
  owner.remove();
  assert.equal(dialog.open, false);
  assert.equal(frames.size, 0);
  assert(avatar.querySelector('form'), 'disconnect restores the same native avatar form');
  assert.equal(avatar.hidden, false);
  assert.equal(nav.classList.contains('is-enhanced'), false);
  identity.open = false;
  current.setAttribute('aria-expanded', 'true');
  dom.window.document.getElementById('outside')?.dispatchEvent(new dom.window.Event('pointerdown', {bubbles: true}));
  dom.window.history.replaceState(null, '', '#identity-settings');
  dom.window.dispatchEvent(new dom.window.HashChangeEvent('hashchange'));
  assert.equal(current.getAttribute('aria-expanded'), 'true', 'retired global handlers do not change the old nav');
  assert.equal(identity.open, false, 'retired hash handlers do not reopen editors');
  dom.window.document.body.append(owner);
  assert.equal(identity.open, true, 're-entry reads the current hash');
  current.click();
  assert.equal(current.getAttribute('aria-expanded'), 'true', 're-entry adds only one listener');
  assert.equal(owner.querySelectorAll('form').length, 1);
  owner.remove();
  dom.window.close();
});

test('server errors retain the inline native avatar editor on late settings entry', () => {
  const {dom} = browser();
  run(dom, 'personal-settings');
  const owner = append(dom, settingsMarkup(true));
  const avatar = owner.querySelector<HTMLDetailsElement>('#avatar-settings');
  assert(avatar && avatar.open && !avatar.hidden && avatar.querySelector('form'));
  assert.equal(owner.querySelector('dialog form'), null);
  owner.remove();
  dom.window.close();
});

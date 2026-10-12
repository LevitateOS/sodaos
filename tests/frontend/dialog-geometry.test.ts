import test, {before, after} from 'node:test';
import assert from 'node:assert/strict';
import {mkdir, rm} from 'node:fs/promises';
import {resolve} from 'node:path';
import type {Page} from 'playwright';
import {setupWorkspaceDriver, action, fixture, openSession} from './fixtures/workspace-driver';
import {buildExtensionStylesheets} from '../../scripts/build-soda-extension.js';
setupWorkspaceDriver();

// F04: terminal and department dialogs keep compact sizing, scroll
// containment, dismissal and focus, with opaque H-ASSET colors and no
// background bleed. Extension stylesheets come from the real builder; only
// document-native sheets stay sourced (E e04/dea42955 owns exact values).
// Colors are asserted at the driver's default scheme only.
const root = resolve(import.meta.dir, '..', '..');
let assets = '';
before(async () => {
  assets = resolve(root, '.artifacts', 'dialog-geometry-emitted');
  await rm(assets, {recursive: true, force: true});
  await mkdir(assets, {recursive: true});
  await buildExtensionStylesheets(assets);
});
after(async () => {
  await rm(assets, {recursive: true, force: true});
});

async function emittedFixture(
  t: Parameters<typeof fixture>[0],
  viewport: {width: number; height: number},
  firstUse: boolean,
  mode: 'native' | 'page' = 'page'
): Promise<Page> {
  const page = await fixture(t, mode);
  await page.setViewportSize(viewport);
  await page.route('**/assets/sodaspaces-drawer.css', (route) =>
    route.fulfill({path: resolve(assets, 'sodaspaces-drawer.css')})
  );
  await page.route('**/assets/sodaspaces-page.css', (route) =>
    route.fulfill({path: resolve(assets, 'sodaspaces-page.css')})
  );
  await page.route('**/assets/sodaspaces-terminal.css', (route) =>
    route.fulfill({path: resolve(assets, 'sodaspaces-terminal.css')})
  );
  // Document components.css stays sourced: it is the single :root value
  // source the emitted extension sheets inherit. Replacing it drops every
  // token and collapses the layout the dialogs mount into.
  await page.reload();
  await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(() => window.sessionStorage.clear());
  await page.evaluate(
    async ({firstUse, mode}) => {
      window.workspaceFixture = window.createWorkspaceFixture(mode, firstUse, '');
      await window.workspaceFixture.api.ready;
    },
    {firstUse, mode}
  );
  return page;
}

async function dialogGeometry(page: Page, name: string, viewport: {width: number; height: number}) {
  const dialog = page.getByRole('dialog', {name});
  await dialog.waitFor();
  const box = await dialog.boundingBox();
  assert(
    box && box.x >= 0 && box.y >= 0 && box.x + box.width <= viewport.width && box.y + box.height <= viewport.height,
    `${name} dialog escapes the viewport`
  );
  const state = await dialog.evaluate((node) => {
    const style = getComputedStyle(node);
    node.scrollTop = node.scrollHeight;
    return {
      background: style.backgroundColor,
      border: style.borderTopColor,
      color: style.color,
      scrolls: node.scrollTop > 0 || node.scrollHeight <= node.clientHeight,
      horizontal: node.scrollWidth - node.clientWidth,
    };
  });
  assert.match(state.background, /^rgb\(/, `${name} dialog background bleeds: ${state.background}`);
  assert.notEqual(state.border, 'rgba(0, 0, 0, 0)', `${name} dialog border is transparent`);
  assert(state.scrolls, `${name} dialog overflow does not scroll`);
  assert(state.horizontal <= 1, `${name} dialog overflows horizontally`);
  return dialog;
}

for (const viewport of [
  {width: 320, height: 568},
  {width: 720, height: 422},
  // Measured 200% browser-zoom equivalent of the 1440x1000 standard viewport.
  {width: 1440 / 2, height: 1000 / 2},
]) {
  test(
    `terminal rename dialog ${viewport.width}x${viewport.height}: sized, opaque, focused, cancellable`,
    {timeout: 30000},
    async (t) => {
      const page = await emittedFixture(t, viewport, false);
      await page.evaluate(() => window.workspaceFixture.api.refresh());
      await openSession(page, 'Build');
      await action(page, 'Rename terminal');
      const dialog = await dialogGeometry(page, 'Rename terminal', viewport);
      await page.waitForFunction(
        () => document.activeElement?.tagName === 'INPUT' && !!document.activeElement?.closest('[role=dialog]'),
        null,
        {timeout: 5000}
      );
      const name = dialog.getByLabel('Session name', {exact: true});
      await name.fill('Compact name');
      await dialog.getByRole('button', {name: 'Cancel', exact: true}).click();
      await dialog.waitFor({state: 'detached'});
      await page.waitForFunction(
        () => document.activeElement?.getAttribute('aria-label') === 'Terminal actions',
        null,
        {
          timeout: 5000,
        }
      );
    }
  );
  test(
    `terminal creation dialog ${viewport.width}x${viewport.height}: sized, opaque, escape-dismissed`,
    {timeout: 30000},
    async (t) => {
      const page = await emittedFixture(t, viewport, false, 'native');
      await page.evaluate(() => window.workspaceFixture.api.refresh());
      await page.getByRole('button', {name: 'New terminal', exact: true}).click();
      const dialog = await dialogGeometry(page, 'New terminal', viewport);
      assert.equal(
        await page.evaluate(() => document.activeElement?.getAttribute('aria-label')),
        'Project',
        'creation dialog does not take initial focus'
      );
      await page.keyboard.press('Escape');
      await dialog.waitFor({state: 'detached'});
      await page.waitForFunction(() => document.activeElement?.getAttribute('aria-label') === 'New terminal', null, {
        timeout: 5000,
      });
    }
  );
  test(
    `department dialog ${viewport.width}x${viewport.height}: sized, opaque, focused, cancellable`,
    {timeout: 30000},
    async (t) => {
      const page = await emittedFixture(t, viewport, true);
      await page.evaluate(() => window.workspaceFixture.api.refresh());
      await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
      await page.getByRole('button', {name: 'New department', exact: true}).click();
      const dialog = await dialogGeometry(page, 'New department', viewport);
      await page.waitForFunction(
        () => {
          const active = document.querySelector('soda-intent')?.shadowRoot?.activeElement;
          return active instanceof HTMLInputElement && !!active.closest('.department-dialog');
        },
        null,
        {timeout: 5000}
      );
      await dialog.getByRole('button', {name: 'Cancel', exact: true}).click();
      await dialog.waitFor({state: 'detached'});
      await page.waitForFunction(
        () => {
          const active = document.querySelector('soda-intent')?.shadowRoot?.activeElement;
          return active instanceof HTMLButtonElement && active.classList.contains('open-department');
        },
        null,
        {timeout: 5000}
      );
    }
  );
}

// Viewport coverage note: the compact/short matrix viewports above verify
// the dialogs across small allocations. This is viewport coverage only,
// pending actual 200% zoom qualification. The CSS `zoom` property is
// deliberately not used: it keeps the viewport while scaling rendering,
// which is not browser zoom and would assert non-production overflow
// semantics.

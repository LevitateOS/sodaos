import test, {before, after} from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {setupSettingsGeometryDriver, settingsFixture} from './fixtures/settings-geometry-driver';
import {browserZoomFixture} from './fixtures/browser-zoom';
import {
  setupWorkspaceDriver,
  fixture,
  openSession,
  action,
  paneAction,
  chooseFirstRepository,
} from './fixtures/workspace-driver';
let browser: Awaited<ReturnType<typeof browserZoomFixture>>;
before(async () => {
  browser = await browserZoomFixture();
});
after(async () => {
  await browser?.close();
});
setupSettingsGeometryDriver(() => browser.context.newPage());
setupWorkspaceDriver(() => browser.context.newPage());

// F05: actual browser zoom, keyboard, compact and short-height reachability.
// Chromium's tabs.setZoom/getZoom changes and verifies its actual per-tab
// browser zoom. These authored component fixtures exercise source behavior;
// they do not establish native template rendering or installed behavior.
// Physical window for every zoom case: 1440x1000.

const PHYSICAL = {width: 1440, height: 1000};

async function applyZoom(page: Page, factor: 2 | 4): Promise<{width: number; height: number}> {
  const css = {
    width: Math.round(PHYSICAL.width / factor),
    height: Math.round(PHYSICAL.height / factor),
  };
  const actual = await browser.setZoom(page, factor);
  assert.deepEqual(
    actual,
    {width: css.width, height: css.height, dpr: factor},
    `zoom ${factor}x did not take effect (a resized viewport would report dpr 1)`
  );
  return css;
}

async function assertDialogInViewport(page: Page, name: string, css: {width: number; height: number}) {
  const dialog = page.getByRole('dialog', {name});
  await dialog.waitFor();
  const box = await dialog.boundingBox();
  assert(
    box && box.x >= 0 && box.y >= 0 && box.x + box.width <= css.width && box.y + box.height <= css.height,
    `${name} dialog escapes the zoomed viewport: ${JSON.stringify(box)} vs ${css.width}x${css.height}`
  );
  const state = await dialog.evaluate((node) => {
    node.scrollTop = node.scrollHeight;
    return {
      scrolls: node.scrollTop > 0 || node.scrollHeight <= node.clientHeight,
      horizontal: node.scrollWidth - node.clientWidth,
    };
  });
  assert(state.scrolls, `${name} dialog overflow does not scroll internally`);
  assert(state.horizontal <= 1, `${name} dialog overflows horizontally`);
  return dialog;
}

const tabs = ['Overview', 'Access', 'Network', 'Factory'] as const;

for (const factor of [2, 4] as const) {
  test(`settings zoom ${factor}x: every tab panel stays usable with reachable controls`, async (t) => {
    // Authored host allocation: 600 physical px, expressed in zoomed CSS px.
    const page = await settingsFixture(t, {}, {...PHYSICAL, allocation: Math.round(600 / factor)});
    await applyZoom(page, factor);
    for (const tab of tabs) {
      await page.getByRole('tab', {name: tab, exact: true}).click();
      const panel = page.locator('[role=tabpanel]:visible');
      await panel.waitFor();
      const geometry = await panel.evaluate((node) => {
        const style = getComputedStyle(node);
        const box = node.getBoundingClientRect();
        return {
          content: box.height - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom),
          horizontal: node.scrollWidth - node.clientWidth,
          scrollable: node.scrollHeight - node.clientHeight,
        };
      });
      assert(geometry.content >= 44, `${tab} zoom ${factor}x panel content squeezed to ${geometry.content}px`);
      assert(geometry.horizontal <= 1, `${tab} zoom ${factor}x panel overflows horizontally`);
      assert(geometry.scrollable >= 0, `${tab} zoom ${factor}x panel lost its scroll metrics`);
      const first = panel.locator('button:not([disabled]), [href], input:not([disabled]), select, textarea').first();
      await first.waitFor();
      await first.scrollIntoViewIfNeeded();
      assert(await first.isVisible(), `${tab} zoom ${factor}x first control unreachable`);
      assert(await first.isEnabled(), `${tab} zoom ${factor}x first control disabled`);
      const focus = await panel.evaluate((node) => {
        const target = node.querySelector('button:not([disabled]), [href], input:not([disabled]), select, textarea');
        if (!(target instanceof HTMLElement)) return 'none';
        target.focus();
        return node.contains(document.activeElement) ? 'inside' : 'escaped';
      });
      assert.equal(focus, 'inside', `${tab} zoom ${factor}x panel has no keyboard-focusable control`);
      const scrolls = await panel.evaluate((node) => {
        node.scrollTop = node.scrollHeight;
        return node.scrollTop > 0 || node.scrollHeight <= node.clientHeight;
      });
      assert(scrolls, `${tab} zoom ${factor}x panel overflow does not scroll`);
    }
  });
}

test('settings zoom 4x: arrow/Home/End tab keyboard navigation still follows focus', async (t) => {
  const page = await settingsFixture(t, {}, {...PHYSICAL, allocation: 150});
  await applyZoom(page, 4);
  const selected = () => page.locator('[role=tab][aria-selected=true]').innerText();
  await page.getByRole('tab', {name: 'Overview', exact: true}).click();
  await page.keyboard.press('End');
  assert.equal(await selected(), 'Factory');
  await page.keyboard.press('Home');
  assert.equal(await selected(), 'Overview');
  await page.keyboard.press('ArrowRight');
  assert.equal(await selected(), 'Access');
});

test('welcome zoom 2x: keyboard reaches the description and planning action', {timeout: 30000}, async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  const css = await applyZoom(page, 2);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  assert(
    await page.locator('.soda-workspace').evaluate((node) => node.scrollWidth <= node.clientWidth),
    'welcome overflows horizontally at zoom 2x'
  );
  await page.getByRole('heading', {name: 'What do you want to build?'}).focus();
  await page.keyboard.press('Tab');
  const description = page.getByLabel('Describe what you want to build', {exact: true});
  assert(
    await description.evaluate((node) => node.matches(':focus')),
    'description is not the first keyboard action at zoom 2x'
  );
  await description.fill('Build a leave tracker');
  const explore = page.getByRole('button', {name: 'Explore ideas', exact: true});
  await explore.scrollIntoViewIfNeeded();
  const box = await explore.boundingBox();
  assert(box && box.y >= 0 && box.y + box.height <= css.height, 'planning action cannot be reached at zoom 2x');
  await explore.focus();
  await page.keyboard.press('Enter');
  await page.getByRole('heading', {name: 'Review your product direction'}).waitFor();
});

test('setup configure zoom 2x: context and step actions stay reachable', {timeout: 30000}, async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  const css = await applyZoom(page, 2);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await chooseFirstRepository(page);
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  assert.equal(await page.locator('.soda-config-repository-name').innerText(), 'alice/Alpha\nForgejo');
  assert(
    await page.locator('.soda-workspace-frame').evaluate((node) => node.scrollWidth <= node.clientWidth + 1),
    'setup frame overflows horizontally at zoom 2x'
  );
  for (const name of ['Back', 'Change repository', 'Cancel setup', 'Create project']) {
    const stepAction = page.getByRole('button', {name, exact: true});
    await stepAction.scrollIntoViewIfNeeded();
    const box = await stepAction.boundingBox();
    assert(box && box.y >= 0 && box.y + box.height <= css.height, `${name} cannot be reached at zoom 2x`);
    assert(await stepAction.isEnabled(), `${name} is not enabled at zoom 2x`);
  }
});

test('terminal rename dialog zoom 4x: sized, focused, cancellable', {timeout: 30000}, async (t) => {
  const page = await fixture(t);
  const css = await applyZoom(page, 4);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await action(page, 'Rename terminal');
  const dialog = await assertDialogInViewport(page, 'Rename terminal', css);
  await page.waitForFunction(
    () => document.activeElement?.tagName === 'INPUT' && !!document.activeElement?.closest('[role=dialog]'),
    null,
    {timeout: 5000}
  );
  await dialog.getByLabel('Session name', {exact: true}).fill('Zoomed name');
  await dialog.getByRole('button', {name: 'Cancel', exact: true}).click();
  await dialog.waitFor({state: 'detached'});
  await page.waitForFunction(() => document.activeElement?.getAttribute('aria-label') === 'Terminal actions', null, {
    timeout: 5000,
  });
});

test('terminal creation dialog zoom 4x: sized, focused, escape-dismissed', {timeout: 30000}, async (t) => {
  // Native mode shows the project-chooser dialog; page mode with a selected
  // project creates directly (see dialog-geometry's creation cases).
  const page = await fixture(t, 'native');
  const css = await applyZoom(page, 4);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  const dialog = await assertDialogInViewport(page, 'New terminal', css);
  assert.equal(
    await page.evaluate(() => document.activeElement?.getAttribute('aria-label')),
    'Project',
    'creation dialog does not take initial focus at zoom 4x'
  );
  const create = dialog.getByRole('button', {name: 'Create terminal', exact: true});
  await create.scrollIntoViewIfNeeded();
  assert(await create.isVisible(), 'Create terminal unreachable inside the zoom 4x dialog');
  await page.keyboard.press('Escape');
  await dialog.waitFor({state: 'detached'});
  await page.waitForFunction(() => document.activeElement?.getAttribute('aria-label') === 'New terminal', null, {
    timeout: 5000,
  });
});

test('department dialog zoom 4x: border box stays in viewport, actions reachable', {timeout: 30000}, async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  const css = await applyZoom(page, 4);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  await page.getByRole('button', {name: 'New department', exact: true}).click();
  const dialog = await assertDialogInViewport(page, 'New department', css);
  await page.waitForFunction(
    () => {
      const active = document.querySelector('soda-intent')?.shadowRoot?.activeElement;
      return active instanceof HTMLInputElement && !!active.closest('.department-dialog');
    },
    null,
    {timeout: 5000}
  );
  const create = dialog.getByRole('button', {name: 'Create department', exact: true});
  await create.scrollIntoViewIfNeeded();
  const box = await create.boundingBox();
  assert(
    box && box.x >= 0 && box.y >= 0 && box.x + box.width <= css.width && box.y + box.height <= css.height,
    'Create department cannot be reached at zoom 4x'
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
});

test('workspace zoom 4x: toolbar scrolls, menu stays on-screen, terminal keeps height', {timeout: 30000}, async (t) => {
  const page = await fixture(t);
  const css = await applyZoom(page, 4);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  // Pane projection recomputes asynchronously after the browser zoom change.
  await page.waitForFunction(
    () => (document.querySelector('.soda-workspace-terminal:not([hidden])')?.getBoundingClientRect().height ?? 0) >= 44,
    null,
    {timeout: 5000}
  );
  const terminal = await page.locator('.soda-workspace-terminal:visible').boundingBox();
  assert(terminal && terminal.height >= 44, `terminal collapsed at zoom 4x: ${JSON.stringify(terminal)}`);
  for (const name of ['New terminal', 'Project settings']) {
    const control = page.getByRole('button', {name, exact: true});
    await control.scrollIntoViewIfNeeded();
    assert(await control.isVisible(), `${name} unreachable at zoom 4x`);
    assert(await control.isEnabled(), `${name} disabled at zoom 4x`);
  }
  await page.getByLabel('Workspace options', {exact: true}).click();
  const item = page.getByRole('button', {name: 'Refresh Spaces', exact: true});
  await item.scrollIntoViewIfNeeded();
  const itemBox = await item.boundingBox();
  assert(
    itemBox && itemBox.x >= 0 && itemBox.x + itemBox.width <= css.width,
    `workspace menu spills off-screen at zoom 4x: ${JSON.stringify(itemBox)}`
  );
  await page.keyboard.press('Escape');
});

test('empty pane zoom 4x: selection message and actions stay inside the pane', {timeout: 30000}, async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await paneAction(page, 'Split right');
  await page.locator('.soda-empty-pane:visible').waitFor();
  const css = await applyZoom(page, 4);
  const pane = page.locator('.soda-empty-pane:visible');
  await pane.waitFor();
  // Pane projection recomputes asynchronously after the browser zoom change.
  await page.waitForFunction(
    () => (document.querySelector('.soda-empty-pane')?.getBoundingClientRect().height ?? 0) >= 60,
    null,
    {timeout: 5000}
  );
  const box = await pane.boundingBox();
  assert(
    box && box.x >= 0 && box.y >= 0 && box.x + box.width <= css.width && box.y + box.height <= css.height,
    `empty pane escapes the zoom 4x viewport: ${JSON.stringify(box)}`
  );
  assert(
    await page.locator('.soda-workspace').evaluate((node) => node.scrollWidth <= node.clientWidth + 1),
    'workspace overflows horizontally beside the empty pane at zoom 4x'
  );
  for (const name of ['Use existing terminal', 'New terminal here']) {
    const control = pane.getByRole('button', {name, exact: true});
    await control.scrollIntoViewIfNeeded();
    assert(await control.isVisible(), `${name} is hidden in the empty pane at zoom 4x`);
    assert(await control.isEnabled(), `${name} is disabled in the empty pane at zoom 4x`);
    const controlBox = await control.boundingBox();
    assert(
      controlBox &&
        box &&
        controlBox.x >= box.x - 1 &&
        controlBox.y >= box.y - 1 &&
        controlBox.x + controlBox.width <= box.x + box.width + 1 &&
        controlBox.y + controlBox.height <= box.y + box.height + 1,
      `${name} renders outside the empty pane box at zoom 4x`
    );
  }
  const focus = await pane.evaluate((node) => {
    const target = node.querySelector('button:not([disabled])');
    if (!(target instanceof HTMLElement)) return 'none';
    target.focus();
    return node.contains(document.activeElement) ? 'inside' : 'escaped';
  });
  assert.equal(focus, 'inside', 'empty pane has no keyboard-focusable selection action at zoom 4x');
});

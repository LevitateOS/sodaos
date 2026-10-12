import test from 'node:test';
import assert from 'node:assert/strict';
import {setupSettingsGeometryDriver, settingsFixture, type SettingsViewport} from './fixtures/settings-geometry-driver';
setupSettingsGeometryDriver();

// F01/F02: installed settings show mostly Identity connections while the selected
// Overview/Access/Network/Factory panel shrinks to its 32 px padding. Each tab
// keeps a usable measured content box beside a rendered Identity sibling across
// wide, compact and short-height allocations.
const tabs = ['Overview', 'Access', 'Network', 'Factory'] as const;
const viewports: ({name: string} & SettingsViewport)[] = [
  {name: 'wide', width: 1440, height: 1000, allocation: 600},
  {name: 'compact', width: 320, height: 568, allocation: 568},
  {name: 'short', width: 720, height: 422, allocation: 422},
];
for (const viewport of viewports)
  for (const tab of tabs)
    test(`settings ${tab} ${viewport.name}: selected panel keeps usable measured content beside Identity`, async (t) => {
      const page = await settingsFixture(t, {}, viewport);
      const identity = await page.locator('soda-identity').boundingBox();
      assert(identity && identity.height > 0, 'Identity sibling did not render beside settings');
      await page.getByRole('tab', {name: tab, exact: true}).click();
      const panel = page.locator('[role=tabpanel]:visible');
      await panel.waitFor();
      assert.equal(await page.getByRole('tab', {name: tab, exact: true}).getAttribute('aria-selected'), 'true');
      const geometry = await panel.evaluate((node) => {
        const style = getComputedStyle(node);
        const box = node.getBoundingClientRect();
        const column = node.parentElement;
        const siblings: Record<string, number> = {};
        for (const selector of [
          '.soda-repository-context',
          '.soda-spaces-tabs',
          '.soda-spaces-summary',
          'soda-identity',
        ]) {
          const sibling = column?.querySelector(selector);
          siblings[selector] = sibling?.getBoundingClientRect().height ?? -1;
        }
        return {
          height: box.height,
          content: box.height - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom),
          scrollable: node.scrollHeight - node.clientHeight,
          horizontal: node.scrollWidth - node.clientWidth,
          siblings,
        };
      });
      const floor = viewport.allocation * 0.2;
      assert(
        geometry.content >= floor,
        `${tab} ${viewport.name} panel content ${geometry.content}px below ${floor}px floor ` +
          JSON.stringify(geometry.siblings)
      );
      assert(
        geometry.horizontal <= 1,
        `${tab} ${viewport.name} panel overflows horizontally by ${geometry.horizontal}px`
      );
      await panel.locator('button:not([disabled]), [href], input:not([disabled]), select, textarea').first().waitFor();
      const focusable = await panel.evaluate((node) => {
        const target = node.querySelector('button:not([disabled]), [href], input:not([disabled]), select, textarea');
        if (!(target instanceof HTMLElement)) return 'none';
        target.focus();
        return node.contains(document.activeElement) ? 'inside' : 'escaped';
      });
      assert.equal(focusable, 'inside', `${tab} ${viewport.name} panel has no keyboard-focusable control`);
      const scrolls = await panel.evaluate((node) => {
        node.scrollTop = node.scrollHeight;
        return node.scrollTop > 0 || node.scrollHeight <= node.clientHeight;
      });
      assert(scrolls, `${tab} ${viewport.name} panel overflow does not scroll`);
    });

test('settings tabs: arrow keys wrap and Home/End jump with selection following focus', async (t) => {
  const page = await settingsFixture(t);
  const selected = () => page.locator('[role=tab][aria-selected=true]').innerText();
  const focused = () => page.evaluate(() => document.activeElement?.textContent?.trim());
  await page.getByRole('tab', {name: 'Overview', exact: true}).click();
  await page.keyboard.press('End');
  assert.equal(await selected(), 'Factory');
  assert.equal(await focused(), 'Factory');
  await page.keyboard.press('Home');
  assert.equal(await selected(), 'Overview');
  assert.equal(await focused(), 'Overview');
  await page.keyboard.press('ArrowLeft');
  assert.equal(await selected(), 'Factory');
  assert.equal(await focused(), 'Factory');
  await page.keyboard.press('ArrowRight');
  assert.equal(await selected(), 'Overview');
  assert.equal(await focused(), 'Overview');
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length),
    0,
    'keyboard tab selection must not write'
  );
});

// Viewport coverage note: the compact (320x568) and short (720x422) matrix
// viewports verify the settings composition across small allocations, and
// the reduced-size test below derives its viewport/allocation from the
// standard base as a reduced CSS viewport/allocation proxy. Final native
// fonts/container/browser behavior (F06) remains open.

test('settings tabs stay usable at reduced CSS viewport/allocation proxy', async (t) => {
  // The proxy halves the CSS viewport and the measured allocation.
  const base = {width: 1440, height: 1000, allocation: 600};
  const viewport = {width: base.width / 2, height: base.height / 2, allocation: base.allocation / 2};
  for (const tab of tabs) {
    const page = await settingsFixture(t, {}, viewport);
    await page.getByRole('tab', {name: tab, exact: true}).click();
    const panel = page.locator('[role=tabpanel]:visible');
    await panel.waitFor();
    const geometry = await panel.evaluate((node) => {
      const style = getComputedStyle(node);
      const box = node.getBoundingClientRect();
      const column = node.parentElement;
      const siblings: Record<string, number> = {};
      for (const selector of [
        '.soda-repository-context',
        '.soda-spaces-tabs',
        '.soda-spaces-summary',
        'soda-identity',
      ]) {
        const sibling = column?.querySelector(selector);
        siblings[selector] = sibling?.getBoundingClientRect().height ?? -1;
      }
      return {
        content: box.height - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom),
        horizontal: node.scrollWidth - node.clientWidth,
        siblings,
      };
    });
    assert(
      (geometry.siblings['soda-identity'] ?? 0) > 0,
      `${tab} reduced-proxy Identity sibling crushed to zero ` + JSON.stringify(geometry.siblings)
    );
    // One control row (44px) stays visible even at the halved allocation.
    assert(
      geometry.content >= 44,
      `${tab} reduced-proxy panel content squeezed to ${geometry.content}px ` + JSON.stringify(geometry.siblings)
    );
    assert(geometry.horizontal <= 1, `${tab} reduced-proxy panel overflows horizontally`);
    const first = panel.locator('button:not([disabled]), [href], input:not([disabled]), select, textarea').first();
    await first.scrollIntoViewIfNeeded();
    assert(await first.isVisible(), `${tab} reduced-proxy first control unreachable`);
    assert(await first.isEnabled(), `${tab} reduced-proxy first control disabled`);
    const scrolls = await panel.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
      return node.scrollTop > 0 || node.scrollHeight <= node.clientHeight;
    });
    assert(scrolls, `${tab} reduced-proxy panel overflow does not scroll`);
    assert(
      await panel.evaluate((node) => {
        const column = node.parentElement;
        return !!column && column.scrollHeight <= column.clientHeight + 1;
      }),
      `${tab} reduced-proxy column spills its allocation`
    );
  }
});

test('settings access with no saved keys keeps the add-key form reachable at compact width', async (t) => {
  const page = await settingsFixture(t, {saved: [], installed: []}, {width: 320, height: 568, allocation: 568});
  await page.getByRole('tab', {name: 'Access', exact: true}).click();
  const panel = page.locator('[role=tabpanel]:visible');
  await panel.waitFor();
  // The subject is the actual empty key list, not the default saved fixture:
  // the joined panel renders no saved-key entries and no remove actions.
  await panel.locator('fieldset[data-control=keys]').waitFor();
  assert.equal(
    await panel.locator('fieldset[data-control=keys] ul li').count(),
    0,
    'empty state still lists a saved key'
  );
  assert.equal(await panel.getByRole('button', {name: 'Remove saved key', exact: true}).count(), 0);
  const keyField = panel.getByLabel('Public SSH key');
  await keyField.waitFor();
  await keyField.scrollIntoViewIfNeeded();
  assert(await keyField.isVisible(), 'empty access panel hides the key field');
  assert(await keyField.isEnabled(), 'empty access panel disables the key field');
  const save = panel.getByRole('button', {name: 'Save public key', exact: true});
  await save.scrollIntoViewIfNeeded();
  assert(await save.isVisible(), 'empty access panel hides the save action');
});

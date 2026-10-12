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

// Zoom note: browser zoom rescales the CSS viewport, and this settings
// composition uses px/%/vh/vw units (no rem-dependent geometry), so the
// compact (320x568) and short (720x422) matrix viewports already exercise
// zoom-equivalent allocations, including 200% on a 1440-wide screen.

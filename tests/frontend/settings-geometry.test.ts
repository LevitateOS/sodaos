import test from 'node:test';
import assert from 'node:assert/strict';
import {setupSettingsGeometryDriver, settingsFixture} from './fixtures/settings-geometry-driver';
setupSettingsGeometryDriver();

// F01: installed settings show mostly Identity connections while the selected
// Overview/Access/Network/Factory panel shrinks to its 32 px padding. Each tab
// keeps a usable measured content box beside a rendered Identity sibling.
const tabs = ['Overview', 'Access', 'Network', 'Factory'] as const;
for (const tab of tabs)
  test(`settings ${tab}: selected panel keeps usable measured content beside Identity`, async (t) => {
    const page = await settingsFixture(t);
    const identity = await page.locator('soda-identity').boundingBox();
    assert(identity && identity.height > 0, 'Identity sibling did not render beside settings');
    await page.getByRole('tab', {name: tab, exact: true}).click();
    const panel = page.locator('[role=tabpanel]:visible');
    await panel.waitFor();
    assert.equal(await page.getByRole('tab', {name: tab, exact: true}).getAttribute('aria-selected'), 'true');
    const geometry = await panel.evaluate((node) => {
      const style = getComputedStyle(node);
      const box = node.getBoundingClientRect();
      return {
        height: box.height,
        content: box.height - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom),
        scrollable: node.scrollHeight - node.clientHeight,
        horizontal: node.scrollWidth - node.clientWidth,
      };
    });
    assert(
      geometry.content >= 120,
      `${tab} panel content squeezed to ${geometry.content}px of ${geometry.height}px panel`
    );
    assert(geometry.horizontal <= 1, `${tab} panel overflows horizontally by ${geometry.horizontal}px`);
    await panel.locator('button:not([disabled]), [href], input:not([disabled]), select, textarea').first().waitFor();
    const focusable = await panel.evaluate((node) => {
      const target = node.querySelector('button:not([disabled]), [href], input:not([disabled]), select, textarea');
      if (!(target instanceof HTMLElement)) return 'none';
      target.focus();
      return node.contains(document.activeElement) ? 'inside' : 'escaped';
    });
    assert.equal(focusable, 'inside', `${tab} panel has no keyboard-focusable control`);
  });

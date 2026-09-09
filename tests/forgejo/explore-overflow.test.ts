import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';

// Read-only native pages in the existing local preview. No login or data writes.
const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
test('native Explore overflow settles and remains usable after resizing', { skip: !origin }, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const page = await browser.newPage();
    for (const route of ['repos', 'users', 'organizations']) {
      await page.goto(`${origin}/explore/${route}`);
      await page.evaluate(async () => {
        await customElements.whenDefined('overflow-menu');
        await document.fonts.ready;
      });
      // Return to desktop too: moving tabs into the popup must not strand them.
      for (const width of [1440, 768, 700, 390, 320, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.waitForTimeout(500); // Native updateItems is debounced by 100ms.
        const state = await page.locator('.soda-tabs').evaluate(async tabs => {
          let mutations = 0;
          const observer = new MutationObserver(records => { mutations += records.length; });
          observer.observe(tabs, { childList: true, subtree: true });
          await new Promise(resolve => setTimeout(resolve, 700));
          observer.disconnect();
          return {
            mutations,
            visible: [...tabs.querySelectorAll('.overflow-menu-items > .item')].map(item => item.textContent.trim()),
            overflow: document.documentElement.scrollWidth > innerWidth,
          };
        });
        assert.equal(state.mutations, 0, `${route} ${width}px: tabs must stop being reparented`);
        assert.equal(state.overflow, false, `${route} ${width}px: page overflow`);
        if (route === 'repos' && width >= 700) {
          const searchWidth = await page.locator('#repo-search-form input[type="search"]').evaluate(el => el.getBoundingClientRect().width);
          assert(searchWidth >= 180, `${width}px: search input must remain usable beside native filters`);
        }
        if (width >= 700) {
          assert.deepEqual(state.visible, ['Repositories', 'Users', 'Organizations']);
        } else {
          assert(!state.visible.includes('Organizations'));
          const button = page.locator('.soda-tabs .overflow-menu-button');
          await button.click();
          const organization = page.getByRole('menuitem', { name: 'Organizations' });
          await organization.waitFor({ state: 'visible' });
          assert.equal(await organization.getAttribute('href'), '/explore/organizations');
          await organization.focus();
          await page.keyboard.press('Escape');
          await organization.waitFor({ state: 'hidden' });
        }
      }
    }
  } finally {
    await browser.close();
  }
});

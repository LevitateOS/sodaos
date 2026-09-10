import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import {chromium} from 'playwright';

// Uses only the authorized local screenshot account. Provider links open native
// GET forms; no migration is submitted and no external provider is contacted.
test('migration chooser fits its viewport and reaches native provider forms', {
  skip: process.env.SODA_FORGEJO_MIGRATION_REVIEW !== '1',
}, async t => {
  const root = new URL('../../../', import.meta.url);
  const origin = 'http://localhost:3300';
  const header = await readFile(new URL('appliance/forgejo/templates/custom/header.tmpl', root), 'utf8');
  const revision = header.match(/name="soda-presentation-revision" content="([^"]+)"/)?.[1];
  assert(revision);
  const context = await chromium.launchPersistentContext(new URL('.local/screenshot-fixture-profile', root).pathname, {
    channel: 'chrome', headless: true, chromiumSandbox: true,
  });
  try {
    // Routing disables the fixture HTTP cache; requests still reach native Forgejo.
    await context.route(`${origin}/**`, route => route.continue());
    const page = context.pages()[0] || await context.newPage();
    await page.emulateMedia({reducedMotion: 'reduce'});
    const styles = await page.request.get(`${origin}/assets/soda/forgejo/onboarding.css`);
    assert(styles.ok());
    assert((await styles.body()).equals(await readFile(new URL('assets/branding/forgejo/onboarding.css', root))));
    let links: string[] = [];
    for (const theme of ['light', 'dark']) for (const width of [1440, 1100, 900, 761, 760, 390, 320]) {
      await t.test(`${theme} at ${width}px`, async () => {
        await page.setViewportSize({width, height: 1000});
        const response = await page.goto(`${origin}/repo/migrate`);
        assert.equal(response?.status(), 200);
        assert.equal(page.url(), `${origin}/repo/migrate`);
        assert.equal(await page.locator('meta[name="soda-presentation-revision"]').getAttribute('content'), revision);
        await page.evaluate(async theme => {
          const link = document.createElement('link');
          link.rel = 'stylesheet'; link.href = `/assets/css/theme-forgejo-${theme}.css`;
          await new Promise((resolve, reject) => { link.onload = resolve; link.onerror = reject; document.head.append(link); });
          document.documentElement.dataset.theme = `forgejo-${theme}`;
          document.documentElement.style.colorScheme = theme;
          await document.fonts.ready;
        }, theme);
        const cards = page.locator('.soda-migrate-provider');
        assert.equal(await cards.count(), 10);
        assert.equal(await page.locator('h1').count(), 1);
        assert.equal(await page.locator('.soda-migrate-provider--git').count(), 1);
        const bounds = await page.locator('.soda-migrate-layout').evaluate(el => ({
          width: el.clientWidth, scroll: el.scrollWidth,
          left: el.getBoundingClientRect().left, right: el.getBoundingClientRect().right, viewport: innerWidth,
        }));
        assert(bounds.scroll <= bounds.width, 'chooser content must not overflow');
        assert(bounds.left >= 0 && bounds.right <= bounds.viewport, 'chooser must fit the viewport');
        for (const card of await cards.all()) {
          const rect = await card.boundingBox(); assert(rect);
          assert(rect.height >= 44 && rect.x >= 0 && rect.x + rect.width <= width);
        }
        for (const logo of await page.locator('.soda-migrate-logo > svg').all()) {
          const size = await logo.evaluate(el => ({width: el.getBoundingClientRect().width, height: el.getBoundingClientRect().height, padding: getComputedStyle(el).padding}));
          assert(size.width >= 32 && size.height >= 32, 'provider logos must be readable');
          assert.equal(size.padding, '0px', 'native icon padding must not shrink the mark');
        }
        const first = cards.first(); await first.focus();
        assert.equal(await first.evaluate(el => getComputedStyle(el).outlineStyle), 'solid');
        await page.keyboard.press('Tab');
        assert(await cards.nth(1).evaluate(el => el === document.activeElement));
        links = await cards.evaluateAll(elements => elements.map(el => {
          if (!(el instanceof HTMLAnchorElement)) throw new Error('Expected native navigation link');
          return el.href;
        }));
        assert.equal(new Set(links).size, 10);
        for (const card of [first, cards.nth(1)]) {
          await page.mouse.move(0, 0);
          const resting = await card.evaluate(el => getComputedStyle(el).backgroundColor);
          await card.hover();
          assert.notEqual(await card.evaluate(el => getComputedStyle(el).backgroundColor), resting, 'hover must distinguish the active source');
        }
      });
    }
    for (const href of links) {
      await t.test(`native provider ${new URL(href).searchParams.get('service_type')}`, async () => {
        const response = await page.goto(href);
        assert.equal(response?.status(), 200);
        assert.equal(page.url(), href);
        assert.equal(await page.locator('input[name="service"]').inputValue(), new URL(href).searchParams.get('service_type'));
        assert(await page.locator('input[name="clone_addr"]').isVisible());
        assert.equal(await page.locator('form.ui.form').getAttribute('method'), 'post');
        assert.equal(await page.locator('.soda-migrate-selection').count(), 0, 'chooser styling must not reach native provider forms');
      });
    }
  } finally { await context.close(); }
});

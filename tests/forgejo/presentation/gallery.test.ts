import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';

test('production reference compositions retain native controls across themes and widths', {skip: process.env.SODA_FORGEJO_LAYOUT_ORIGIN !== 'http://localhost:3300'}, async t => {
  const browser = await chromium.launch({channel:'chrome',headless:true});
  try {
    const page = await browser.newPage();
    for (const theme of ['light','dark']) for (const width of [1440,1101,1100,960,900,800,720,640,390,320]) {
      await t.test(`${theme} at ${width}px`, async () => {
        await page.setViewportSize({width,height:width < 500 ? 844 : 1000});
        await page.emulateMedia({colorScheme: theme === 'light' ? 'dark' : 'light'});
        const reviewOrigin = process.env.SODA_FORGEJO_REVIEW_ORIGIN;
        if (reviewOrigin) assert.equal(reviewOrigin, 'http://localhost:8140');
        await page.goto(reviewOrigin ? `${reviewOrigin}/gallery-${theme}.html` : new URL(`../../../.artifacts/forgejo-presentation/gallery-${theme}.html`, import.meta.url).href);
        await page.evaluate(() => document.fonts.ready);
        const bounds = await page.evaluate(() => ({width:innerWidth,scroll:document.documentElement.scrollWidth}));
        assert.equal(bounds.scroll,bounds.width,'composition must not overflow the viewport');
        if (reviewOrigin) {
          assert.equal(await page.locator('body').evaluate(el => getComputedStyle(el).backgroundColor), theme === 'light' ? 'rgb(255, 255, 255)' : 'rgb(16, 16, 16)');
          const primary = await page.locator('#button-reference .ui.primary.button').first().evaluate(el => ({bg: getComputedStyle(el).backgroundColor, fg: getComputedStyle(el).color, radius: getComputedStyle(el).borderRadius}));
          assert.deepEqual(primary, {bg: 'rgb(223, 0, 27)', fg: 'rgb(255, 255, 255)', radius: '0px'});
          assert.equal(await page.locator('.soda-page-title').evaluate(el => getComputedStyle(el).fontFamily.includes('Barlow Condensed')), true);
        }
        for (const id of ['title','date','error','disabled']) {
          assert(await page.locator(`#${id}`).evaluate(el => el.getBoundingClientRect().height >= 44),id);
        }
        await page.locator('#title').focus();
        assert.notEqual(await page.locator('#title').evaluate(el => getComputedStyle(el).outlineStyle),'none');
        assert(await page.locator('#disabled').isDisabled());
        for (const button of await page.locator('button[disabled]').all()) assert(await button.isDisabled());
        await page.locator('summary').first().click();
        assert(await page.locator('#branch').isVisible());
        const layout=await page.locator('.soda-editor-layout').evaluate(el => ({cols:getComputedStyle(el).gridTemplateColumns.split(' ').length}));
        assert.equal(layout.cols,width<=1100?1:2);
        for (const button of await page.locator('.ui.labeled.button > button').all()) {
          const edges = await button.evaluate(el => ({top:getComputedStyle(el).borderTopRightRadius,bottom:getComputedStyle(el).borderBottomRightRadius}));
          assert.deepEqual(edges,{top:'0px',bottom:'0px'});
        }
        const surface=await page.locator('.soda-p-section').first().evaluate(el => ({border:getComputedStyle(el).borderTopWidth,bg:getComputedStyle(el).backgroundColor}));
        assert.equal(surface.border,'0px');
        assert.equal(surface.bg,'rgba(0, 0, 0, 0)');
      });
    }
    if (process.env.SODA_FORGEJO_REVIEW_ORIGIN === 'http://localhost:8140') {
      await page.goto('http://localhost:8140/gallery-light.html');
      await page.evaluate(() => {
        document.documentElement.dataset.theme = 'soda-auto';
        document.querySelector('main')!.dataset.signed = 'true';
        document.querySelector<HTMLLinkElement>('link[href*="theme-soda-light.css"]')!.href = '/assets/soda/forgejo/css/theme-soda-auto.css';
        // Native head_navbar's image/home-link seam; no navigation replacement.
        document.body.insertAdjacentHTML('afterbegin', '<nav id="navbar"><a id="navbar-logo" class="item" href="/" aria-label="Home"><img width="30" height="30" src="/assets/soda/source/soda-symbol-brutalist.svg" alt="Logo" aria-hidden="true"></a></nav>');
      });
      for (const colorScheme of ['light','dark'] as const) {
        await page.emulateMedia({colorScheme});
        const expected = colorScheme === 'light' ? 'rgb(255, 255, 255)' : 'rgb(16, 16, 16)';
        await page.waitForFunction(bg => getComputedStyle(document.body).backgroundColor === bg, expected);
        const logo = await page.locator('#navbar-logo').evaluate(el => ({bg:getComputedStyle(el).backgroundImage, width:el.getBoundingClientRect().width}));
        assert(logo.bg.includes(colorScheme === 'light' ? 'soda-symbol-brutalist.svg' : 'soda-symbol-brutalist-dark.svg'));
        assert.equal(logo.width,44);
      }
    }
  } finally {await browser.close();}
});

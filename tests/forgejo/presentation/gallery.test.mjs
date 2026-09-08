import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';

test('production reference compositions retain native controls across themes and widths', {skip: process.env.SODA_FORGEJO_LAYOUT_ORIGIN !== 'http://localhost:3300'}, async t => {
  const browser = await chromium.launch({channel:'chrome',headless:true});
  try {
    const page = await browser.newPage();
    for (const theme of ['light','dark']) for (const width of [1440,900,390,320]) {
      await t.test(`${theme} at ${width}px`, async () => {
        await page.setViewportSize({width,height:width < 500 ? 844 : 1000});
        await page.goto(new URL(`../../../.artifacts/forgejo-presentation/gallery-${theme}.html`, import.meta.url).href);
        await page.evaluate(() => document.fonts.ready);
        const bounds = await page.evaluate(() => ({width:innerWidth,scroll:document.documentElement.scrollWidth}));
        assert.equal(bounds.scroll,bounds.width,'composition must not overflow the viewport');
        for (const id of ['title','date','error','disabled']) {
          assert(await page.locator(`#${id}`).evaluate(el => el.getBoundingClientRect().height >= 44),id);
        }
        await page.locator('#title').focus();
        assert.notEqual(await page.locator('#title').evaluate(el => getComputedStyle(el).outlineStyle),'none');
        assert(await page.locator('#disabled').isDisabled());
        assert(await page.locator('button[disabled]').isDisabled());
        await page.locator('summary').first().click();
        assert(await page.locator('#branch').isVisible());
        const layout=await page.locator('.soda-editor-layout').evaluate(el => ({cols:getComputedStyle(el).gridTemplateColumns.split(' ').length}));
        assert.equal(layout.cols,width<=900?1:2);
        for (const button of await page.locator('.ui.labeled.button > button').all()) {
          const edges = await button.evaluate(el => ({top:getComputedStyle(el).borderTopRightRadius,bottom:getComputedStyle(el).borderBottomRightRadius}));
          assert.deepEqual(edges,{top:'0px',bottom:'0px'});
        }
        const surface=await page.locator('.soda-p-section').first().evaluate(el => ({border:getComputedStyle(el).borderTopWidth,bg:getComputedStyle(el).backgroundColor}));
        assert.equal(surface.border,'0px');
        assert.equal(surface.bg,'rgba(0, 0, 0, 0)');
      });
    }
  } finally {await browser.close();}
});

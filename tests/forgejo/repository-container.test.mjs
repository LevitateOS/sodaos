import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';
const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;

test('ordinary repository pages share container insets and navigation gap', {skip: !origin}, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const browser = await chromium.launch({channel:'chrome',headless:true});
  try {
    const page = await browser.newPage();
    for (const width of [1440,390]) {
      await page.setViewportSize({width,height:1000});
      let reference;
      for (const suffix of ['', '/projects', '/issues', '/releases']) {
        const url = `${origin}/alice/activity-workbench${suffix}`;
        const response = await page.goto(url);
        assert.equal(response.status(),200,url);
        assert.equal(page.url(),url,'redirect is not page coverage');
        const layout = await page.locator('.page-content > .ui.container').first().evaluate(el => {
          const rect = el.getBoundingClientRect();
          const nav = document.querySelector('.soda-repository-header').getBoundingClientRect();
          return {padding:getComputedStyle(el).padding,left:rect.left,width:rect.width,gap:rect.top-nav.bottom};
        });
        assert.equal(layout.padding,'0px',`${suffix || '/'} at ${width}px`);
        reference ||= layout;
        assert.deepEqual(layout,reference,`${suffix || '/'} must use the repository body's shared spacing at ${width}px`);
      }
    }
  } finally {await browser.close();}
});

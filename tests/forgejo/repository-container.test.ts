import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';
const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;

test('ordinary repository pages share container insets and navigation gap', {skip: !origin}, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const browser = await chromium.launch({channel:'chrome',headless:true});
  try {
    const page = await browser.newPage();
    for (const width of [1440,960,800,720,390]) {
      await page.setViewportSize({width,height:1000});
      let reference;
      for (const suffix of ['', '/projects', '/issues', '/releases']) {
        const url: string = `${origin}/alice/activity-workbench${suffix}`;
        const response = await page.goto(url);
        assert(response); assert.equal(response.status(),200,url);
        assert.equal(page.url(),url,'redirect is not page coverage');
        const layout = await page.locator('.page-content > .ui.container').first().evaluate(el => {
          const rect = el.getBoundingClientRect();
          const header = document.querySelector('.soda-repository-header');
          if(!header) throw Error('Missing repository navigation');
          const nav = header.getBoundingClientRect();
          return {padding:getComputedStyle(el).padding,left:rect.left,width:rect.width,gap:rect.top-nav.bottom};
        });
        assert.equal(layout.padding,'0px',`${suffix || '/'} at ${width}px`);
        reference ||= layout;
        assert.deepEqual(layout,reference,`${suffix || '/'} must use the repository body's shared spacing at ${width}px`);
      }
    }
  } finally {await browser.close();}
});

test('repository identity and workspace actions share one row with native secondary actions available', {skip: !origin}, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const browser = await chromium.launch({channel: 'chrome', headless: true, chromiumSandbox: true});
  try {
    const page = await browser.newPage();
    const errors: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    for (const [width, pane] of [[1440,1440], [960,960], [800,800], [720,720], [390,390], [320,320], [1440,720], [1440,480]]) {
      assert(width && pane);
      await page.setViewportSize({width, height: 1000});
      const response = await page.goto(`${origin}/alice/activity-workbench`);
      assert.equal(response?.status(), 200);
      await page.locator('#sodaspaces-button').waitFor({state: 'visible'});
      // A constrained native pane must respond to its own width, even when
      // the browser is wide. No workspace or backend operation is launched.
      await page.evaluate(pane => {document.body.style.width = `${pane}px`;}, pane);
      await page.evaluate(() => document.fonts.ready);
      const header = page.locator('.soda-repository-header');
      const row = header.locator('.repo-header');
      const layout = await row.evaluate(el => {
        const identity = el.querySelector(':scope > .flex-item')?.getBoundingClientRect();
        const actions = el.querySelector('.repo-buttons')?.getBoundingClientRect();
        if (!identity || !actions) throw Error('Missing repository identity/actions');
        return {sameRow: identity.top < actions.bottom && actions.top < identity.bottom, clear: identity.right <= actions.left};
      });
      assert.deepEqual(layout, {sameRow: true, clear: true}, `${width}/${pane}: compact identity row`);
      assert.equal(await header.locator('.repo-buttons #sodaspaces-button').count(), 1);
      const disclosure = header.locator('.soda-repository-actions');
      const summary = disclosure.locator('summary');
      const menu = disclosure.locator('.soda-repository-action-menu');
      const beforeHeight = await row.evaluate(el => el.getBoundingClientRect().height);
      assert.equal(await disclosure.count(), 1);
      await summary.focus();
      await page.keyboard.press('Enter');
      await menu.waitFor({state: 'visible'});
      const bounds = await menu.boundingBox();
      assert(bounds && bounds.x >= 0 && bounds.x + bounds.width <= pane, `${width}/${pane}: actions stay within pane`);
      assert.equal(await row.evaluate(el => el.getBoundingClientRect().height), beforeHeight, 'opening actions must not add a header row');
      for (const action of ['watch', 'star']) {
        const form = menu.locator(`form[action$="/${action}"]`);
        assert.equal(await form.count(), 1, 'one native form per action');
        assert.equal(await form.getAttribute('hx-target'), 'this');
        assert(await form.locator('.text').isVisible(), 'action names remain readable on mobile');
        assert(await form.locator('button').isDisabled(), 'guest action guards survive grouping');
      }
      const watchers = menu.locator('a[href$="/watchers"]');
      await watchers.focus();
      await page.keyboard.press('Escape');
      await menu.waitFor({state: 'hidden'});
      assert(await summary.evaluate(el => document.activeElement === el));
      await page.keyboard.press('Space');
      await menu.waitFor({state: 'visible'});
      await header.locator('#sodaspaces-button').focus();
      await menu.waitFor({state: 'hidden'});
      await summary.click();
      await menu.waitFor({state: 'visible'});
      await page.mouse.click(8, 600);
      await menu.waitFor({state: 'hidden'});
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      await summary.click();
      const [destination] = await Promise.all([page.waitForNavigation(), watchers.click()]);
      assert.equal(destination?.status(), 200, 'count link remains a usable native read-only destination');
      assert.equal(new URL(page.url()).pathname, '/alice/activity-workbench/watchers');
    }
    assert.deepEqual(errors, [], 'native page keyboard controls must not throw');
  } finally {await browser.close();}
});

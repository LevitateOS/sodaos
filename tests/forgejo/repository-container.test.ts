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
    for (const [width, pane] of [[1440,1440], [1001,1001], [1000,1000], [960,960], [800,800], [720,720], [390,390], [320,320], [1440,720], [1440,480]]) {
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
      const compact = pane <= 1000;
      if (compact) {
        await menu.waitFor({state: 'hidden'});
        await summary.focus();
        await page.keyboard.press('Enter');
      } else {
        assert(!(await summary.isVisible()), 'wide pages expose actions without a menu trigger');
      }
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
      if (compact) {
        await menu.waitFor({state: 'hidden'});
        assert(await summary.evaluate(el => document.activeElement === el));
        await page.keyboard.press('Space');
        await menu.waitFor({state: 'visible'});
        await header.locator('#sodaspaces-button').focus();
        await menu.waitFor({state: 'hidden'});
        await summary.click();
        await menu.waitFor({state: 'visible'});
      } else {
        assert(await menu.isVisible(), 'Escape cannot hide full-width actions');
        assert(await watchers.evaluate(el => document.activeElement === el));
        await header.locator('#sodaspaces-button').focus();
        assert(await menu.isVisible(), 'focus leaving cannot hide full-width actions');
      }
      await page.mouse.click(8, 600);
      await menu.waitFor({state: compact ? 'hidden' : 'visible'});
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
      if (compact) await summary.click();
      const [destination] = await Promise.all([page.waitForNavigation(), watchers.click()]);
      assert.equal(destination?.status(), 200, 'count link remains a usable native read-only destination');
      assert.equal(new URL(page.url()).pathname, '/alice/activity-workbench/watchers');
    }
    assert.deepEqual(errors, [], 'native page keyboard controls must not throw');
  } finally {await browser.close();}
});

test('repository actions switch modes on resize without replacing native forms', {skip: !origin}, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const browser = await chromium.launch({channel: 'chrome', headless: true, chromiumSandbox: true});
  try {
    const page = await browser.newPage({viewport: {width: 1440, height: 1000}});
    await page.goto(`${origin}/alice/activity-workbench`);
    const details = page.locator('.soda-repository-actions');
    const summary = details.locator('summary');
    const menu = details.locator('.soda-repository-action-menu');
    const nativeForm = await menu.locator('form[action$="/watch"]').elementHandle();
    assert(nativeForm);
    await menu.waitFor({state: 'visible'});
    await page.setViewportSize({width: 800, height: 1000});
    await menu.waitFor({state: 'hidden'});
    await summary.focus();
    await page.setViewportSize({width: 1440, height: 1000});
    await page.waitForFunction(() => document.querySelector('.soda-repository-action-menu')?.contains(document.activeElement));
    await menu.locator('a[href$="/watchers"]').focus();
    // The available pane, not the outer browser, decides the compact mode.
    await page.evaluate(() => {document.body.style.width = '720px';});
    await summary.waitFor({state: 'visible'});
    assert(await menu.isVisible(), 'resizing keeps a focused native action available');
    await page.keyboard.press('Escape');
    await menu.waitFor({state: 'hidden'});
    await page.evaluate(() => {document.body.style.width = '';});
    await summary.waitFor({state: 'hidden'});
    await menu.waitFor({state: 'visible'});
    assert(await nativeForm.evaluate(el => el === document.querySelector('.soda-repository-actions form[action$="/watch"]')));
    assert.equal(await menu.locator('form[action$="/watch"]').count(), 1);

    const fallback = await browser.newPage({javaScriptEnabled: false, viewport: {width: 1440, height: 1000}});
    await fallback.goto(`${origin}/alice/activity-workbench`);
    assert(!(await fallback.locator('.soda-repository-actions > summary').isVisible()));
    assert(await fallback.locator('.soda-repository-actions a[href$="/watchers"]').isVisible(), 'wide native actions remain available without JavaScript');
  } finally {await browser.close();}
});

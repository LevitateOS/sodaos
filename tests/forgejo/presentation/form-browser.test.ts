import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import {chromium} from 'playwright';

// Read-only review of the existing authorized local screenshot fixture.
// No repository, account, token, runner or integration is created by this test.
test('native form workspaces fit desktop/mobile and preserve interactive controls', {
  skip: process.env.SODA_FORGEJO_FORM_REVIEW !== '1',
  timeout: 180000,
}, async () => {
  const origin = 'http://localhost:3300';
  const root = new URL('../../../', import.meta.url);
  const context = await chromium.launchPersistentContext(new URL('.local/screenshot-fixture-profile', root).pathname, {
    channel: 'chrome', headless: true, chromiumSandbox: true,
  });
  try {
    const writes: string[] = [];
    await context.route(`${origin}/**`, route => {
      if (!['GET', 'HEAD'].includes(route.request().method())) {
        writes.push(new URL(route.request().url()).pathname);
        return route.abort();
      }
      return route.continue(); // also disables the fixture's HTTP cache
    });
    const page = context.pages()[0] ?? await context.newPage();
    page.setDefaultTimeout(5000);
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    const style = await page.request.get(`${origin}/assets/soda/forgejo/form-pages.css`);
    assert(style.ok());
    assert((await style.body()).equals(await readFile(new URL('assets/branding/forgejo/form-pages.css', root))));
    const routes = [
      '/repo/create', '/org/create', '/vince/activity-playground/fork',
      '/soda-screenshot/-/projects/new', '/vince/activity-playground/issues/new',
      '/user/settings/applications/tokens/new', '/user/settings/hooks/forgejo/new',
      '/user/settings/packages/rules/add',
    ];
    await page.goto(`${origin}/repo/migrate`);
    const migrations = await page.locator('.soda-migrate-provider').evaluateAll(es => es.map(el => {
      assertAnchor(el);
      return new URL(el.href).pathname + new URL(el.href).search;
      function assertAnchor(e: Element): asserts e is HTMLAnchorElement { if (!(e instanceof HTMLAnchorElement)) throw new Error('Expected provider link'); }
    }));
    assert.equal(migrations.length, 10);
    routes.push(...migrations);
    const layoutFailures: string[] = [];
    for (const theme of ['light', 'dark'] as const) for (const width of [1440, 390]) {
      await page.setViewportSize({width, height: 1000});
      for (const route of routes) {
        try {
          const response = await page.goto(origin + route);
          assert.equal(response?.status(), 200, `${theme} ${width} ${route}`);
          assert.equal(page.url(), origin + route, `${theme} ${width} ${route}`);
          await page.evaluate(async theme => {
            const link = document.createElement('link');
            link.rel = 'stylesheet'; link.href = `/assets/css/theme-forgejo-${theme}.css`;
            await new Promise((resolve, reject) => {link.onload = resolve; link.onerror = reject; document.head.append(link);});
            document.documentElement.dataset.theme = `forgejo-${theme}`;
            document.documentElement.style.colorScheme = theme;
            await document.fonts.ready;
          }, theme);
          assert.equal(await page.locator('h1').count(), 1, route);
          const layout = page.locator('.soda-form-layout, .soda-form-editor').first();
          assert(await layout.isVisible());
          const geometry = await layout.evaluate(el => ({
            left: el.getBoundingClientRect().left, right: el.getBoundingClientRect().right,
            width: el.clientWidth, scroll: el.scrollWidth,
            document: document.documentElement.scrollWidth,
          }));
          assert(geometry.left >= 0 && geometry.right <= width, `${route} ${JSON.stringify(geometry)}`);
          assert(geometry.scroll <= geometry.width + 1, `${route} ${JSON.stringify(geometry)}`);
          assert(geometry.document <= width, `${route} ${JSON.stringify(geometry)}`);
          if (await page.locator('.soda-form-story').count()) {
            const story = await page.locator('.soda-form-story').boundingBox();
            const form = await page.locator('.soda-form-content').boundingBox();
            assert(story && form);
            if (width === 1440) assert(form.x > story.x + story.width, 'form must sit beside the story');
            else assert(form.y >= story.y + story.height, 'form must follow the story on mobile');
          }
          for (const image of await layout.locator('img.soda-page-art').all()) assert(await image.evaluate(el => el instanceof HTMLImageElement && el.complete && el.naturalWidth > 0));
        } catch (error) {
          layoutFailures.push(`${theme} ${width} ${route}: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
    }
    assert.deepEqual(layoutFailures, []);
    await page.goto(`${origin}/repo/create`);
    const form = page.locator('form');
    assert.equal(await form.getAttribute('method'), 'post');
    assert.equal(await form.locator('input[name="repo_name"]').evaluate(el => el instanceof HTMLInputElement && el.checkValidity()), false);
    const privateChoice = form.locator('input[name="private"]');
    await privateChoice.check();
    assert.equal(await form.evaluate(el => el instanceof HTMLFormElement && new FormData(el).get('private')), 'on');
    await privateChoice.uncheck();
    assert.equal(await form.evaluate(el => el instanceof HTMLFormElement && new FormData(el).has('private')), false);
    const templateOptions = form.locator('details').first();
    assert.equal(await templateOptions.getAttribute('open'), null);
    await templateOptions.locator('summary').focus();
    await page.keyboard.press('Enter');
    assert.notEqual(await templateOptions.getAttribute('open'), null);
    assert(await form.locator('#repo_template_search').isVisible());
    await form.locator('input[name="auto_init"]').check();
    assert(await form.locator('input[name="license"]').locator('..').isVisible());
    await page.goto(`${origin}/org/create`);
    await page.locator('input[name="visibility"][value="2"]').check();
    const card = page.locator('.ui.radio.checkbox').last();
    assert(await card.locator('input').isChecked());
    const hitAreas = await card.evaluate(el => {
      const input = el.querySelector('input')?.getBoundingClientRect();
      const label = el.querySelector('label');
      return {right: input?.right ?? 0, text: (label?.getBoundingClientRect().left ?? 0) + parseFloat(label ? getComputedStyle(label).paddingLeft : '0')};
    });
    assert(hitAreas.text > hitAreas.right, 'radio must not overlap its label');
    for (const [route, trigger, panel] of [
      ['/user/settings/keys', '[data-panel="#add-ssh-key-panel"]', '#add-ssh-key-panel'],
      ['/user/settings/keys', '[data-panel="#add-gpg-key-panel"]', '#add-gpg-key-panel'],
    ]) {
      assert(route && trigger && panel);
      await page.goto(origin + route);
      await page.locator(`.show-panel${trigger}`).click();
      assert(await page.locator(`${panel} .soda-form-panel`).isVisible());
    }
    assert.deepEqual(writes, [], 'review must never submit creation forms');
    assert.deepEqual(errors, []);
  } finally { await context.close(); }
});

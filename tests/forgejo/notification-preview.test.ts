import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
// Browser-only signed-in markup and response fixtures. Never log in or mutate
// native notifications; run the actual bundled Forgejo HTMX and authored script.
test('notification preview uses native HTMX with stable accessible lifecycle', { skip: !origin }, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const footerSource = await readFile(new URL('../../appliance/forgejo/templates/custom/footer.tmpl', import.meta.url), 'utf8');
  const footer = footerSource.replace('{{if .IsSigned}}', '').replace(/{{end}}\s*$/, '')
    .replaceAll('{{AppSubUrl}}', '').replaceAll('{{AssetUrlPrefix}}', '/assets')
    .replace(/{{ctx.Locale.Tr "([^"]+)"}}/g, (_, key) => key)
    .replace(/{{svg [^}]+}}/g, '×');
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 }, colorScheme: 'dark' });
    const errors: string[] = [];
    let fixtureHTML: string | undefined;
    page.on('pageerror', error => errors.push(error.message));
    await page.route('**/explore/repos', async route => {
      const response = await route.fetch();
      const html = (await response.text())
        .replace('</head>', '<link rel="stylesheet" href="/assets/soda/forgejo/notification-preview.css"></head>')
        .replace('<div class="navbar-right ui secondary menu">', '<div class="navbar-right ui secondary menu"><a class="item not-mobile" id="fixture-desktop-bell" href="/notifications"><span class="notification_count">5</span></a>')
        .replace('</nav>', '<a id="fixture-mobile-bell" class="only-mobile" href="/notifications"><span class="notification_count">5</span></a></nav>')
        .replace('</body>', `${footer}</body>`);
      fixtureHTML = html;
      await route.fulfill({ response, body: html });
    });
    let mode = 'populated';
    let requests = 0;
    const methods: string[] = [];
    let release: (() => void) | undefined;
    const markup = (label: string) => `<div class="soda-notification-preview-list" data-soda-notification-fragment><ul>${Array.from({ length: 5 }, (_, i) => `<li><a href="/team/repo/issues/${i + 1}"><span class="soda-notification-preview-entry"><span class="soda-notification-preview-repo">team/repository-with-a-long-name</span><span class="soda-notification-preview-title">${i ? 'Additional notification with a long title that wraps onto another line' : label}</span><span class="soda-notification-preview-time">recently${i === 0 ? ' · Pinned' : ''}</span></span></a></li>`).join('')}</ul></div>`;
    await page.route('**/notifications?*', async route => {
      requests++;
      methods.push(route.request().method());
      assert.equal(route.request().headers()['hx-request'], 'true');
      const url = new URL(route.request().url());
      assert.equal(url.searchParams.get('perPage'), '5');
      assert.equal(url.searchParams.get('soda-preview'), 'true');
      if (mode === 'delayed') {
        await new Promise<void>(resolve => { release = resolve; });
        await route.fulfill({ contentType: 'text/html', body: markup('Stale response') }).catch(() => {});
      } else if (mode === 'error') {
        await route.fulfill({ status: 500, contentType: 'text/html', body: 'Private error body must not appear' });
      } else if (mode === 'unexpected') {
        await route.fulfill({ contentType: 'text/html', body: '<main>Unexpected login page</main>' });
      } else if (mode === 'expired') {
        await route.fulfill({ status: 204, headers: { 'HX-Redirect': '/user/login' } });
      } else {
        await route.fulfill({ contentType: 'text/html', body: mode === 'empty'
          ? '<div class="soda-notification-preview-list" data-soda-notification-fragment><p>No unread or pinned notifications.</p></div>'
          : markup('Latest notification') });
      }
    });
    await page.goto(`${origin}/explore/repos`);
    await page.waitForLoadState('load');
    const panel = page.locator('#soda-notification-preview');
    const bell = page.locator('#fixture-desktop-bell');
    const close = panel.locator('.soda-notification-preview-close');
    assert.equal(requests, 0);
    await bell.focus();
    await page.keyboard.press('Enter');
    await panel.getByText('Latest notification').waitFor();
    assert.equal(await bell.getAttribute('aria-expanded'), 'true');
    assert.equal(requests, 1);
    await page.keyboard.press('Escape');
    await panel.waitFor({ state: 'hidden' });
    assert.equal(await bell.evaluate(e => e === document.activeElement), true);
    assert.equal(await panel.locator('.soda-notification-preview-content').textContent(), '');
    const modifierHandled = await bell.evaluate(link => {
      let handled;
      link.addEventListener('click', event => { handled = event.defaultPrevented; event.preventDefault(); }, { once: true });
      link.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, ctrlKey: true }));
      return handled;
    });
    assert.equal(modifierHandled, false, 'modified click keeps native link behavior');
    assert.equal(requests, 1);

    for (const next of ['empty', 'error', 'unexpected']) {
      mode = next;
      await bell.click();
      await panel.getByText(next === 'empty' ? 'No unread or pinned notifications.' : 'Could not load notifications.', { exact: false }).waitFor();
      assert(!await panel.textContent().then(text => { assert(text !== null); return text.includes('Private error body') || text.includes('Unexpected login'); }));
      if (next !== 'empty') {
        mode = 'populated';
        await panel.getByRole('button', { name: 'Retry' }).click();
        await panel.getByText('Latest notification').waitFor();
      }
      await close.click();
    }

    mode = 'delayed';
    await bell.click();
    await panel.getByText('Loading notifications…').waitFor();
    await close.click();
    mode = 'populated';
    await bell.click();
    await panel.getByText('Latest notification').waitFor();
    assert(release);
    release();
    await page.waitForTimeout(150);
    assert(!await panel.textContent().then(text => { assert(text !== null); return text.includes('Stale response'); }));
    await page.locator('#navbar-logo').focus();
    await panel.waitFor({ state: 'hidden' });

    for (const colorScheme of ['light','dark'] as const) for (const width of [320,390,640,720,800,960,1440]) {
      await page.emulateMedia({colorScheme});
      await page.setViewportSize({ width, height: 900 });
      const active = width < 768 ? page.locator('#fixture-mobile-bell') : bell;
      await active.click();
      await panel.getByText('Latest notification').waitFor();
      const bounds = await panel.boundingBox();
      assert(bounds);
      assert(bounds.x >= 0 && bounds.x + bounds.width <= width, `${width}px horizontal containment`);
      assert(bounds.y >= 0 && bounds.y + bounds.height <= 900, `${width}px vertical containment`);
      const design = await panel.evaluate(el => {
        const header = el.querySelector('.soda-notification-preview-header')!;
        const action = el.querySelector('.soda-notification-preview-all')!;
        return {border:getComputedStyle(el).borderTopWidth,shadow:getComputedStyle(el).boxShadow,
          headerBg:getComputedStyle(header).backgroundColor,headerFg:getComputedStyle(header).color,
          bg:getComputedStyle(el).backgroundColor,fg:getComputedStyle(el).color,
          actionCase:getComputedStyle(action).textTransform,actionHeight:action.getBoundingClientRect().height};
      });
      assert.equal(design.border,'2px');assert.equal(design.shadow,'none');
      assert.equal(design.headerBg,design.fg);assert.equal(design.headerFg,design.bg);
      assert.equal(design.actionCase,'uppercase');assert(design.actionHeight>=44);
      await close.click();
    }
    await page.emulateMedia({ colorScheme: 'light' });
    // Outside click dismisses without navigating or stealing focus.
    await bell.click();
    await panel.getByText('Latest notification').waitFor();
    await page.locator('.soda-page-intro h1').click();
    await panel.waitFor({ state: 'hidden' });
    assert.equal(await bell.getAttribute('href'), '/notifications');
    assert.deepEqual([...new Set(methods)], ['GET']);
    assert.deepEqual(errors, []);

    mode = 'expired';
    await bell.click();
    await page.waitForURL('**/user/login');
    assert.equal(await page.locator('#soda-notification-preview').count(), 0);

    for (const fallback of ['no-js', 'no-htmx', 'no-popover']) {
      const fallbackPage = await browser.newPage({ javaScriptEnabled: fallback !== 'no-js' });
      const html = fixtureHTML;
      assert(html);
      await fallbackPage.route('**/explore/repos', route => route.fulfill({ contentType: 'text/html', body: html }));
      await fallbackPage.route('**/notifications', route => route.fulfill({ contentType: 'text/html', body: '<main>Native notifications fallback</main>' }));
      if (fallback === 'no-htmx') {
        await fallbackPage.route('**/assets/js/index.js*', route => route.fulfill({ contentType: 'text/javascript', body: '' }));
      }
      if (fallback === 'no-popover') {
        await fallbackPage.addInitScript(() => { Object.defineProperty(HTMLElement.prototype, 'showPopover', {value: undefined}); });
      }
      await fallbackPage.goto(`${origin}/explore/repos`);
      assert.equal(await fallbackPage.locator('#soda-notification-preview').isVisible(), false);
      await fallbackPage.locator('#fixture-desktop-bell').click();
      await fallbackPage.waitForURL('**/notifications');
      await fallbackPage.getByText('Native notifications fallback').waitFor();
      await fallbackPage.close();
    }
  } finally {
    await browser.close();
  }
});

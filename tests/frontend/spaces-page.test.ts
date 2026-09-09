import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {chromium} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {object} from '../../appliance/forgejo/public/assets/sodaspaces-api';

test('authorized Go HTML/CSP boots one emitted Spaces page with its original actor', {skip: !process.env.SODA_SPACES_PAGE_HTML}, async () => {
  const root = path.resolve(import.meta.dirname, '../..');
  const fixture = object(await Bun.file(process.env.SODA_SPACES_PAGE_HTML || '').json());
  const {html, csp, origin} = fixture; assert(typeof html === 'string' && typeof csp === 'string' && typeof origin === 'string');
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}), errors: string[] = [], calls: string[] = [], missing: string[] = [];
  const page = await browser.newPage(); page.on('pageerror', error => errors.push(error.message));
  await page.route('**/*', async route => {
    const request = route.request(), url = new URL(request.url());
    assert.equal(url.origin, origin); assert.equal(request.method(), 'GET');
    if (url.pathname === '/-/soda/spaces') {await route.fulfill({body: html, contentType: 'text/html', headers: {'Content-Security-Policy': csp, 'Cache-Control': 'private, no-store'}}); return;}
    if (url.pathname.startsWith('/-/soda/api/')) {
      calls.push(url.pathname); assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      const json = url.pathname.endsWith('/session') ? {user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-only', forgejo_url: origin} : {items: [], complete: true};
      await route.fulfill({json}); return;
    }
    const source = Object.entries(payload).find(([dest]) => dest === 'public' + url.pathname)?.[1];
    if (!source) {missing.push(url.pathname); await route.abort(); return;}
    const file = source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : source.startsWith('@build/terminal-assets/') ? path.join(root, '.artifacts/browser-terminal/vendor', path.basename(source)) : path.join(root, source);
    assert(await Bun.file(file).exists());
    await route.fulfill({body: Buffer.from(await Bun.file(file).bytes()), contentType: source.endsWith('.js') ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : source.endsWith('.svg') ? 'image/svg+xml' : source.endsWith('.woff2') ? 'font/woff2' : 'image/png'});
  });
  try {
    await page.goto(origin + '/-/soda/spaces'); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.equal(await page.locator('#spaces-page').getAttribute('data-soda-actor'), '1'); assert.equal(await page.locator('soda-spaces').count(), 1);
    assert.equal(await page.locator('soda-terminal').count(), 0);
    assert.equal(await page.getByRole('navigation', {name: 'Native Forgejo', exact: true}).getByRole('link', {name: 'Issues', exact: true}).getAttribute('href'), origin + '/issues');
    for (const colorScheme of ['light', 'dark'] as const) {
      await page.emulateMedia({colorScheme});
      const colors = await page.locator('body').evaluate(node => {
        const probe = document.createElement('span'); probe.style.color = 'var(--soda-page-canvas)'; node.append(probe);
        const expected = getComputedStyle(probe).color; probe.remove();
        return {actual: getComputedStyle(node).backgroundColor, expected, font: getComputedStyle(node).fontFamily};
      });
      assert.equal(colors.actual, colors.expected); assert.match(colors.font, /Barlow/);
    }
    assert.deepEqual(calls, ['/-/soda/api/session', '/-/soda/api/spaces']);
    assert.deepEqual(errors, []); assert.deepEqual(missing, []);
  } finally {await browser.close();}
});

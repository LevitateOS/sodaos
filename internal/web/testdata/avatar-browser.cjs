// Opt-in local browser check, driven by Go's test-owned HTTP server.
const assert = require('node:assert/strict');
const { createHash } = require('node:crypto');
const { chromium } = require('playwright');

(async () => {
  const origin = process.argv[2];
  assert.match(origin, /^http:\/\/127\.0\.0\.1:\d+$/);
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 640, height: 400 }, deviceScaleFactor: 1 });
    const errors = [];
    page.on('pageerror', error => errors.push(String(error)));
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
    page.on('request', request => {
      if (!request.url().startsWith(origin + '/')) errors.push('unexpected network request');
    });
    for (const sample of [0, 1, 8, 23, 59, 99]) {
      const hash = createHash('md5').update(`soda-avatar-preview-${String(sample).padStart(3, '0')}`).digest('hex');
      await page.setContent(`<style>body{margin:0}img{display:block;width:128px;height:128px}</style>
        <img id="actual" src="${origin}/-/soda/avatars/v1/${hash}?s=128&d=identicon">`);
      await page.evaluate(() => Promise.all([...document.images].map(image => image.decode())));
      const actual = await page.locator('#actual').screenshot();
      // Keep the same screen position to avoid position-dependent SVG antialiasing.
      await page.locator('#actual').evaluate((image, url) => { image.src = url; }, `${origin}/reference/${hash}`);
      await page.locator('#actual').evaluate(image => image.decode());
      const reference = await page.locator('#actual').screenshot();
      assert.ok(actual.equals(reference), `HTTP/CSP changed rendered pixels for sample ${sample}`);
    }
    assert.deepEqual(errors, []);
    console.log('Six production HTTP avatars match reference pixels; no CSP errors or external requests.');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });

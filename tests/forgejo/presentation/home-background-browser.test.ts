import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';
const origin = 'http://localhost:8140';
const enabled = process.env.SODA_FORGEJO_REVIEW_ORIGIN === origin;

test('welcome frame contains all content and selects one responsive background', {skip: !enabled}, async () => {
 const browser = await chromium.launch({channel: 'chrome', headless: true});
 try {
  for (const theme of ['light', 'dark']) for (const width of [320, 390, 768, 1024, 1440, 2560]) {
   const page = await browser.newPage({viewport: {width, height: 1000}, colorScheme: theme === 'light' ? 'dark' : 'light'});
   const failures: string[] = [], backgrounds: string[] = [];
   page.on('pageerror', error => failures.push(error.message));
   page.on('response', response => {
    if (response.status() >= 400) failures.push(response.url());
    if (response.url().includes('/backgrounds/')) backgrounds.push(response.url());
   });
   await page.goto(`${origin}/home-${theme}.html`, {waitUntil: 'networkidle'});
   await page.evaluate(() => document.fonts.ready);
   const size = width < 768 ? 'mobile' : width < 1200 ? 'tablet' : 'desktop';
   const image = `${size}-${theme === 'dark' ? 'night' : 'day'}.webp`;
   assert.equal(backgrounds.length, 1, 'only the active background should download');
   assert(backgrounds[0]!.endsWith(image), image);
   const layout = await page.evaluate(() => {
    const shell = document.querySelector<HTMLElement>('.full.height')!;
    const footer = document.querySelector('footer')!;
    const frame = shell.getBoundingClientRect(), end = footer.getBoundingClientRect();
    return {
     overflow: document.documentElement.scrollWidth > innerWidth,
     left: frame.left, right: innerWidth - frame.right,
     footerLeft: end.left, footerRight: innerWidth - end.right,
     boundary: end.top - frame.bottom,
     background: getComputedStyle(shell).backgroundColor,
     footerBackground: getComputedStyle(footer).backgroundColor,
     panelBorder: getComputedStyle(shell).borderBottomWidth,
     footerBorder: getComputedStyle(footer).borderTopWidth,
    };
   });
   assert.equal(layout.overflow, false, `${theme}/${width}: overflow`);
   assert(layout.left >= 12, 'photographic gutter must remain visible');
   assert(Math.abs(layout.left - layout.right) < 1, 'frame must be centered');
   assert(Math.abs(layout.left - layout.footerLeft) < 1 && Math.abs(layout.right - layout.footerRight) < 1, 'footer belongs inside the frame');
   assert(Math.abs(layout.boundary) < 1, 'no gap before native footer');
   assert.equal(layout.background, theme === 'dark' ? 'rgb(16, 16, 16)' : 'rgb(255, 255, 255)');
   assert.equal(layout.footerBackground, layout.background);
   assert.equal(layout.panelBorder, '0px');
   assert.equal(layout.footerBorder, '1px', 'footer owns the shared divider');
   assert.deepEqual(failures, []);
   await page.close();
  }
 } finally { await browser.close(); }
});

// Render static design documents without an HTTP server, app, credentials or network.
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
import {chromium} from 'playwright';
import {packageManager} from '../../../package.json';

assert.equal(Bun.argv.length, 2, 'usage: bun docs/design/spaces/render-sheets.ts');
assert.equal(`bun@${Bun.version}`, packageManager, 'Use the root-pinned Bun');
process.umask(0o077);
const root = path.resolve(import.meta.dir, '../../..');
const out = path.join(root, '.artifacts/spaces-redesign', String(Date.now()));
await mkdir(path.dirname(out), {recursive: true, mode: 0o700});
await mkdir(out, {mode: 0o700});
const assets: Record<string, {file: string; type: string}> = {
  '/focus.svg': {file: 'docs/design/spaces/focus.svg', type: 'image/svg+xml'},
  '/split.svg': {file: 'docs/design/spaces/split.svg', type: 'image/svg+xml'},
  '/mobile-states.svg': {file: 'docs/design/spaces/mobile-states.svg', type: 'image/svg+xml'},
  '/drawer-work.svg': {file: 'docs/design/spaces/drawer-work.svg', type: 'image/svg+xml'},
  '/drawer-sessions.svg': {file: 'docs/design/spaces/drawer-sessions.svg', type: 'image/svg+xml'},
  '/drawer-compact.svg': {file: 'docs/design/spaces/drawer-compact.svg', type: 'image/svg+xml'},
  '/sheets.css': {file: 'docs/design/spaces/sheets.css', type: 'text/css'},
  '/palette.css': {file: 'assets/branding/theme/palette.css', type: 'text/css'},
  '/symbol.svg': {file: 'assets/branding/source/soda-symbol.svg', type: 'image/svg+xml'},
  '/barlow.woff2': {file: 'assets/branding/fonts/barlow/barlow-latin-400-normal.woff2', type: 'font/woff2'},
  '/barlow-semibold.woff2': {file: 'assets/branding/fonts/barlow/barlow-latin-600-normal.woff2', type: 'font/woff2'},
  '/mono.woff2': {file: 'assets/branding/fonts/ibm-plex-mono/ibm-plex-mono-latin-400-normal.woff2', type: 'font/woff2'},
};
const origin = 'https://spaces-design.invalid'; // Fulfilled in-memory; never contacted.
const browser = await chromium.launch({headless: true, chromiumSandbox: true});
try {
  const context = await browser.newContext({serviceWorkers: 'block'});
  const rejected: string[] = [];
  const pageErrors: string[] = [];
  await context.route('**/*', async route => {
    const request = route.request();
    const url = new URL(request.url());
    const asset = assets[url.pathname];
    if (url.origin !== origin || request.method() !== 'GET' || !asset || url.search) {
      rejected.push(request.url());
      await route.abort();
      return;
    }
    await route.fulfill({status: 200, contentType: asset.type,
      headers: {'Content-Security-Policy': "default-src 'none'; style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'none'"},
      body: Buffer.from(await Bun.file(path.join(root, asset.file)).arrayBuffer())});
  });
  const page = await context.newPage();
  page.on('pageerror', error => pageErrors.push(error.message));
  const captures: Array<{file: string; width: number; height: number; textNodes: number; monoCellWidth: number}> = [];
  for (const sheet of [
    {name: 'focus', width: 1440, height: 1024},
    {name: 'split', width: 1920, height: 1160},
    {name: 'mobile-states', width: 1280, height: 1120},
    {name: 'drawer-work', width: 1600, height: 1120},
    {name: 'drawer-sessions', width: 1440, height: 1040},
    {name: 'drawer-compact', width: 1280, height: 1120},
  ]) {
    await page.setViewportSize({width: sheet.width, height: sheet.height});
    const response = await page.goto(`${origin}/${sheet.name}.svg`);
    assert.equal(response?.status(), 200);
    assert.equal(await page.locator('parsererror').count(), 0, 'Invalid SVG');
    const observed = await page.evaluate(async () => {
      const fonts = await Promise.all(['14px Barlow', '600 14px Barlow', '14px Plex'].map(font => document.fonts.load(font)));
      await document.fonts.ready;
      const textNodes = [...document.querySelectorAll('text')];
      const overflow = textNodes.filter(node => {
        const box = node.getBoundingClientRect();
        return box.x < -1 || box.y < -1 || box.right > innerWidth + 1 || box.bottom > innerHeight + 1;
      }).map(node => node.textContent);
      const canvas = document.createElementNS('http://www.w3.org/1999/xhtml', 'canvas');
      if (!(canvas instanceof HTMLCanvasElement)) throw new Error('Canvas unavailable');
      const paint = canvas.getContext('2d');
      if (!paint) throw new Error('Canvas text measurement unavailable');
      paint.font = '14px Plex';
      return {textNodes: textNodes.length, overflow, loaded: fonts.every(faces => faces.length > 0),
        monoCellWidth: paint.measureText('0').width};
    });
    assert(observed.loaded, 'Canonical fonts must load');
    assert.deepEqual(observed.overflow, [], 'Drawing text outside the sheet');
    const file = `${sheet.name}.png`;
    await page.screenshot({path: path.join(out, file)});
    captures.push({file, width: sheet.width, height: sheet.height,
      textNodes: observed.textNodes, monoCellWidth: observed.monoCellWidth});
  }
  assert.deepEqual(rejected, [], 'Unexpected request, blocked before network');
  assert.deepEqual(pageErrors, []);
  const hashes: Record<string, string> = {};
  for (const file of [...Object.values(assets).map(asset => asset.file), 'docs/design/spaces/render-sheets.ts']) {
    hashes[file] = new Bun.CryptoHasher('sha256').update(await Bun.file(path.join(root, file)).arrayBuffer()).digest('hex');
  }
  const revision = Bun.spawnSync(['git', 'rev-parse', 'HEAD'], {cwd: root});
  const status = Bun.spawnSync(['git', 'status', '--porcelain'], {cwd: root});
  assert.equal(revision.exitCode, 0); assert.equal(status.exitCode, 0);
  await Bun.write(path.join(out, 'render.json'), JSON.stringify({
    kind: 'STATIC DESIGN — not a live UI, behavior test or installed evidence',
    revision: revision.stdout.toString().trim(), dirty: status.stdout.length > 0,
    browser: browser.version(), network: 'All page requests fulfilled from a fixed local asset map; no listener',
    hashes, captures,
  }, null, 2) + '\n');
  console.log(`Rendered ${captures.length} static design sheets: ${out}`);
} finally {
  await browser.close(); // Closes the fresh context on success and every failure.
}

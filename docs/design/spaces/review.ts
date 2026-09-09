// Isolated design-preview build/serve/capture, never an appliance entrypoint.
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
import {chromium, type Page} from 'playwright';

const mode = Bun.argv[2];
assert((mode === '--check' || mode === '--serve') && Bun.argv.length === 3,
  'usage: bun docs/design/spaces/review.ts --check|--serve');
assert(Bun.version === '1.4.2', 'Use the pinned Bun version');
process.umask(0o077);
const root = path.resolve(import.meta.dir, '../../..');
const out = path.join(root, '.artifacts/spaces-design', String(Date.now()));
await mkdir(path.dirname(out), {recursive: true, mode: 0o700});
await mkdir(out, {mode: 0o700});
const build = await Bun.build({entrypoints: [path.join(import.meta.dir, 'preview.ts')], target: 'browser', outdir: out});
assert(build.success, 'Design preview emission failed');
const assets: Record<string, {file: string; type: string}> = {
  '/': {file: path.join(import.meta.dir, 'index.html'), type: 'text/html'},
  '/preview.js': {file: path.join(out, 'preview.js'), type: 'text/javascript'},
  '/preview.css': {file: path.join(import.meta.dir, 'preview.css'), type: 'text/css'},
  '/palette.css': {file: path.join(root, 'assets/branding/theme/palette.css'), type: 'text/css'},
  '/symbol.svg': {file: path.join(root, 'assets/branding/source/soda-symbol.svg'), type: 'image/svg+xml'},
  '/barlow.woff2': {file: path.join(root, 'assets/branding/fonts/barlow/barlow-latin-400-normal.woff2'), type: 'font/woff2'},
  '/barlow-semibold.woff2': {file: path.join(root, 'assets/branding/fonts/barlow/barlow-latin-600-normal.woff2'), type: 'font/woff2'},
  '/mono.woff2': {file: path.join(root, 'assets/branding/fonts/ibm-plex-mono/ibm-plex-mono-latin-400-normal.woff2'), type: 'font/woff2'},
};
const server = Bun.serve({hostname: '127.0.0.1', port: mode === '--serve' ? 33450 : 0,
  fetch(request) {
    const asset = assets[new URL(request.url).pathname];
    if (request.method !== 'GET' || !asset) return new Response('Design preview: no API or route here', {status: 404});
    return new Response(Bun.file(asset.file), {headers: {'Content-Type': asset.type, 'Cache-Control': 'no-store',
      'Content-Security-Policy': "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self'; connect-src 'none'; base-uri 'none'; form-action 'none'"}});
  },
});
if (mode === '--serve') {
  console.log(`Design preview ONLY: ${server.url}\nNo API/project connections. Ctrl-C stops this preview.\nGenerated output: ${out}`);
  const stop = () => { void server.stop(true); };
  process.once('SIGINT', stop); process.once('SIGTERM', stop);
} else {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}).catch(async error => {
    await server.stop(true); throw error;
  });
  try {
    const context = await browser.newContext();
    const errors: string[] = [];
    const captures: Array<{file: string; width: number; height: number; scene: string; theme: string}> = [];
    let outsideRequests = 0;
    const page = await context.newPage();
    page.on('pageerror', error => errors.push(error.message));
    await context.route('**/*', route => {
      if (new URL(route.request().url()).origin !== server.url.origin) { outsideRequests++; return route.abort(); }
      return route.continue();
    });
    async function load(width: number, height: number, scene = 'working', theme = 'dark') {
      await page.setViewportSize({width, height});
      await page.goto(server.url + `?scene=${scene}&theme=${theme}`);
      await page.locator('.terminal-tab').first().waitFor();
      await page.evaluate(() => document.fonts.ready);
    }
    async function capture(file: string, scene: string, theme: string) {
      assert(await page.getByText('DESIGN PREVIEW', {exact: true}).isVisible());
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'Horizontal page overflow');
      const viewport = page.viewportSize(); assert(viewport);
      await page.screenshot({path: path.join(out, file)});
      captures.push({file, ...viewport, scene, theme});
    }
    async function tabIDs(p: Page) { return p.locator('[role=tab]').evaluateAll(nodes => nodes.map(n => n.id).sort()); }
    try {
      await load(1440, 960);
      assert.equal(await page.locator('[role=tab]').count(), 6);
      assert.equal(await page.locator('.pane:visible').count(), 2);
      await capture('desktop-dark.png', 'working', 'dark');
      const original = await tabIDs(page);
      await page.getByRole('tab', {name: 'acme/api · shell', exact: true}).focus();
      await page.keyboard.press('ArrowRight');
      assert.equal(await page.getByRole('tab', {name: 'acme/api · tests', exact: true}).getAttribute('aria-selected'), 'true');
      await page.getByRole('button', {name: 'Actions for tests in acme/api', exact: true}).click();
      await page.getByRole('button', {name: 'Move to another pane', exact: true}).click();
      assert.equal(await page.locator('.pane[data-group="1"] [role=tab][aria-selected=true]').getAttribute('id'), 'tab-sample-api-tests');
      assert.deepEqual(await tabIDs(page), original);
      await page.getByRole('button', {name: 'Actions for tests in acme/api', exact: true}).click();
      await page.getByRole('button', {name: 'Move to another pane', exact: true}).click();
      await page.getByRole('button', {name: 'Details for acme/api', exact: true}).click();
      assert(await page.locator('#inspector').isVisible());
      assert.deepEqual(await tabIDs(page), original);
      await page.getByRole('button', {name: 'Close project details', exact: true}).click();
      await page.getByRole('button', {name: 'Maximize pane 1', exact: true}).click();
      assert.equal(await page.locator('.pane:visible').count(), 1);
      assert.deepEqual(await tabIDs(page), original);
      await page.getByRole('button', {name: 'Restore pane layout', exact: true}).click();
      assert.equal(await page.locator('.pane:visible').count(), 2);
      await page.locator('#search').fill('no-match');
      assert(await page.getByText('No matching projects or terminal labels. Your open views have not changed.').isVisible());
      assert.deepEqual(await tabIDs(page), original);
      await page.locator('#search').fill('');
      await page.locator('#layout').selectOption('grid');
      assert.equal(await page.locator('.pane:visible').count(), 4);
      assert.deepEqual(await tabIDs(page), original);
      await page.locator('#layout').selectOption('single');
      assert.equal(await page.locator('.pane:visible').count(), 1);
      assert.deepEqual(await tabIDs(page), original);
      await page.getByRole('button', {name: 'Hide tests — keeps the terminal; does not End', exact: true}).click();
      assert.equal(await page.locator('[role=tab]').count(), 5);
      assert.equal(await page.locator('[data-session="sample-api-tests"]').getAttribute('data-status'), 'kept');
      await page.locator('[data-session="sample-api-tests"]').click();
      assert.equal(await page.locator('[role=tab]').count(), 6);
      assert(await page.getByText('Kept · 30m remaining', {exact: true}).isVisible());
      await page.getByRole('button', {name: 'Simulate Continue working', exact: true}).click();
      await page.getByRole('button', {name: 'Actions for tests in acme/api', exact: true}).click();
      await page.getByRole('button', {name: 'End terminal', exact: true}).click();
      await page.getByRole('button', {name: 'Cancel', exact: true}).click();
      assert.equal(await page.locator('[data-session="sample-api-tests"]').getAttribute('data-status'), 'connected');
      await page.getByRole('button', {name: 'Actions for tests in acme/api', exact: true}).click();
      await page.getByRole('button', {name: 'End terminal', exact: true}).click();
      await page.getByRole('button', {name: 'End sample terminal', exact: true}).click();
      assert.equal(await page.locator('[data-session="sample-api-tests"]').getAttribute('data-status'), 'ended');
      assert.equal(await page.locator('[data-session="sample-web-dev"]').getAttribute('data-status'), 'connected');
      await page.locator('#new-terminal').click();
      await page.getByRole('button', {name: 'Cancel', exact: true}).click();
      assert.equal(await page.locator('[role=tab]').count(), 6);
      await page.locator('#new-terminal').click();
      await page.getByRole('button', {name: 'Create sample terminal in acme/web', exact: true}).click();
      assert.equal(await page.locator('[role=tab]').count(), 7);
      assert(await page.getByRole('tab', {name: 'acme/web · Terminal 7', exact: true}).isVisible());
      await load(1440, 960, 'away');
      await capture('retention-dark.png', 'away', 'dark');
      await page.locator('[data-session="sample-api-tests"]').click();
      assert(await page.getByText('Native cleanup unconfirmed', {exact: true}).isVisible());
      await capture('uncertain-dark.png', 'away-uncertain', 'dark');
      await load(390, 844, 'working', 'light');
      assert.equal(await page.locator('.pane:visible').count(), 1);
      await capture('mobile-light.png', 'working', 'light');
      await page.getByRole('button', {name: 'Show projects and terminals', exact: true}).click();
      await capture('mobile-switcher-light.png', 'navigator', 'light');
      await page.locator('[data-session="sample-web-dev"]').click();
      assert.equal(await page.locator('.pane:visible').getAttribute('data-group'), '1');
      assert.equal(await page.locator('[role=tab]').count(), 6);
      await load(320, 720, 'away', 'light');
      await capture('mobile-retention-light.png', 'away', 'light');
      await load(1440, 960, 'working', 'light');
      await capture('desktop-light.png', 'working', 'light');
      assert.equal(outsideRequests, 0); assert.deepEqual(errors, []);
      const sourceHashes: Record<string, string> = {};
      for (const file of ['index.html', 'preview.css', 'preview.ts', 'review.ts']) sourceHashes[file] = new Bun.CryptoHasher('sha256').update(await Bun.file(path.join(import.meta.dir, file)).bytes()).digest('hex');
      const git = Bun.spawnSync(['git', 'rev-parse', 'HEAD'], {cwd: root, stdout: 'pipe', stderr: 'pipe'});
      const status = Bun.spawnSync(['git', 'status', '--porcelain'], {cwd: root, stdout: 'pipe', stderr: 'pipe'});
      assert.equal(git.exitCode, 0); assert.equal(status.exitCode, 0);
      const result = {kind: 'DESIGN MOCKUP — not product evidence', revision: git.stdout.toString().trim(), dirty: status.stdout.toString().length !== 0, sourceHashes, captures, checks: 'sample tab identity, keyboard selection, moving, filtering, layout/maximize, details, explicit project-scoped creation, Hide versus End, cancellation, sibling isolation, mobile selection, overflow, no outside requests/page errors', browser: browser.version()};
      await Bun.write(path.join(out, 'review.json'), JSON.stringify(result, null, 2) + '\n');
      console.log(`PASS design-only browser checks; captures: ${out}`);
    } finally { await context.close(); }
  } finally { await browser.close(); await server.stop(true); }
}

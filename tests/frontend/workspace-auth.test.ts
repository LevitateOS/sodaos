import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {chromium, type Browser, type Page} from 'playwright';
import payload from '../../internal/release/build/forgejo-payload.json';

const root = path.resolve(import.meta.dirname, '../..');

// Leave-shell proof runs the emitted shipping bundle, not a hand-rolled load
// handler: the same sodaspaces-shell.js the Go template references, with only
// /api/session and /api/spaces stubbed (empty inventory, no terminals).
const workspacePage = `<!doctype html><title>Workspace auth fixture</title>
<link rel="stylesheet" href="/assets/sodaspaces-shell.css">
<body class="soda-workspace-shell">
<span class="sodaspaces-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span>
<nav id="sodaspaces-surfaces" hidden aria-label="Workspace surface">
<button type="button" class="ui button" id="soda-surface-forge">Forge</button>
<button type="button" class="ui button" id="soda-surface-terminal">Terminal</button>
</nav>
<iframe id="soda-forgejo-frame" title="Forgejo" src="about:blank"></iframe>
<div id="soda-workspace-divider" role="separator" tabindex="0" aria-orientation="vertical" aria-label="Workspace width"></div>
<div id="soda-workspace-root" data-actor="1"></div>
<script type="module" src="/assets/sodaspaces-shell.js"></script>
</body>`;

async function launchBrowser() {
  try {
    return await chromium.launch({headless: true, chromiumSandbox: true});
  } catch {
    return await chromium.launch({headless: true, chromiumSandbox: true, channel: 'chrome'});
  }
}

function frameDocument(pathname: string) {
  return `<!doctype html><title>${pathname}</title><p>${pathname}</p>`;
}

async function gotoWorkspace(page: Page, origin: string) {
  await page.goto(origin + '/workspace');
  await page.locator('soda-spaces').waitFor();
  assert.equal(await page.locator('#soda-workspace-root.soda-spaces-page-mount').count(), 1);
}

async function driveFrame(page: Page, path: string) {
  await page.locator('#soda-forgejo-frame').evaluate((frame, next) => {
    (frame as HTMLIFrameElement).src = next;
  }, path);
}

async function leaveThroughFrame(page: Page, path: string) {
  await driveFrame(page, path);
  await page.waitForURL((url) => url.pathname + url.search === path);
  assert.equal(await page.locator('#soda-forgejo-frame').count(), 0);
}

test('shipping shell mounts, tracks the frame, and leaves login, consent, and callback', async (t) => {
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    async fetch(req) {
      const url = new URL(req.url);
      if (url.pathname.startsWith('/assets/')) {
        const source = Object.entries(payload).find(([target]) => target === 'public' + url.pathname)?.[1];
        if (!source) return new Response(null, {status: 404});
        const file = source.startsWith('@build/forgejo-js/')
          ? path.join(root, '.artifacts/forgejo-js', path.basename(source).split('?')[0])
          : path.join(root, source);
        const contentType = /\.m?js$/.test(source)
          ? 'text/javascript'
          : source.endsWith('.css')
            ? 'text/css'
            : 'application/octet-stream';
        return new Response(Bun.file(file), {headers: {'Content-Type': contentType}});
      }
      if (url.pathname === '/workspace') {
        return new Response(workspacePage, {headers: {'Content-Type': 'text/html'}});
      }
      if (url.pathname === '/-/soda/api/session') {
        return Response.json({user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-only', forgejo_url: url.origin});
      }
      if (url.pathname === '/-/soda/api/spaces') {
        return Response.json({items: [], complete: true});
      }
      return new Response(frameDocument(url.pathname), {headers: {'Content-Type': 'text/html'}});
    },
  });
  t.after(() => server.stop(true));
  const browser: Browser = await launchBrowser();
  t.after(() => browser.close());
  const page = await browser.newPage({viewport: {width: 1440, height: 900}});
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  const origin = server.url.origin;

  await gotoWorkspace(page, origin);
  // Ordinary framed navigation stays in the shell and names the child path.
  await driveFrame(page, '/forge');
  await page.waitForFunction(() => new URL(location.href).searchParams.get('to') === '/forge');
  assert.equal(await page.locator('soda-spaces').count(), 1);

  await leaveThroughFrame(page, '/user/login');
  await gotoWorkspace(page, origin);
  await leaveThroughFrame(page, '/login/oauth/grant');
  await gotoWorkspace(page, origin);
  await leaveThroughFrame(page, '/-/soda/oauth/callback?code=fixture-code&state=fixture-state');
  assert.deepEqual(errors, []);
});

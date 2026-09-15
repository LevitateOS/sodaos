import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {chromium, type Browser, type Page} from 'playwright';
import payload from '../../internal/release/build/forgejo-payload.json';

const root = path.resolve(import.meta.dirname, '../..');

const workspacePage = `<!doctype html><title>Workspace auth fixture</title>
<body class="soda-workspace-shell">
<iframe id="soda-forgejo-frame" title="Forgejo" src="/forge"></iframe>
<div id="soda-workspace-root" data-actor="1"></div>
<script type="module">
import {applyWorkspaceFrameNavigation} from '/assets/sodaspaces-frame.js';
const frame = document.querySelector('#soda-forgejo-frame');
frame.addEventListener('load', () => {
  const loc = frame.contentWindow.location;
  applyWorkspaceFrameNavigation(window, loc.pathname + loc.search);
});
</script>
</body>`;

async function launchBrowser() {
  try {
    return await chromium.launch({headless: true, chromiumSandbox: true});
  } catch {
    return await chromium.launch({headless: true, chromiumSandbox: true, channel: 'chrome'});
  }
}

async function gotoWorkspace(page: Page, origin: string) {
  await page.goto(origin + '/workspace');
  await page.waitForFunction(() => new URL(location.href).searchParams.get('to') === '/forge');
}

async function leaveThroughFrame(page: Page, path: string) {
  await page.locator('#soda-forgejo-frame').evaluate((frame, next) => {
    (frame as HTMLIFrameElement).src = next;
  }, path);
  await page.waitForURL((url) => url.pathname + url.search === path);
  assert.equal(await page.locator('#soda-forgejo-frame').count(), 0);
}

test('browser shell leaves login, consent, and callback', async (t) => {
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    async fetch(req) {
      const url = new URL(req.url);
      if (url.pathname.startsWith('/assets/')) {
        const source = Object.entries(payload).find(([target]) => target === 'public' + url.pathname)?.[1];
        if (!source) return new Response(null, {status: 404});
        const file = source.startsWith('@build/forgejo-js/')
          ? path.join(root, '.artifacts/forgejo-js', path.basename(source))
          : path.join(root, source);
        return new Response(Bun.file(file), {headers: {'Content-Type': 'text/javascript'}});
      }
      if (url.pathname === '/workspace') {
        return new Response(workspacePage, {headers: {'Content-Type': 'text/html'}});
      }
      return new Response(`<!doctype html><title>${url.pathname}</title><p>${url.pathname}</p>`, {
        headers: {'Content-Type': 'text/html'},
      });
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
  assert.equal(new URL(page.url()).pathname, '/workspace');
  await leaveThroughFrame(page, '/user/login');
  await gotoWorkspace(page, origin);
  await leaveThroughFrame(page, '/login/oauth/grant');
  await gotoWorkspace(page, origin);
  await leaveThroughFrame(page, '/-/soda/oauth/callback?code=fixture-code&state=fixture-state');
  assert.deepEqual(errors, []);
});

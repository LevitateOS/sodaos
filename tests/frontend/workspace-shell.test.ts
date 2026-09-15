import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {chromium} from 'playwright';
import payload from '../../internal/release/build/forgejo-payload.json';

const root = path.resolve(import.meta.dirname, '../..');

test('browser shell split keeps Forgejo chrome in the frame', async (t) => {
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
        const contentType = source.endsWith('.css') ? 'text/css' : 'text/javascript';
        return new Response(Bun.file(file), {headers: {'Content-Type': contentType}});
      }
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        `<!doctype html><title>Workspace shell fixture</title>
<link rel="stylesheet" href="/assets/sodaspaces.css">
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
<script type="module">
import {bindWorkspaceShellLayout} from '/assets/sodaspaces-shell-layout.js';
bindWorkspaceShellLayout(document);
</script>
</body>`,
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  t.after(() => server.stop(true));
  let browser;
  try {
    browser = await chromium.launch({headless: true, chromiumSandbox: true});
  } catch {
    browser = await chromium.launch({headless: true, chromiumSandbox: true, channel: 'chrome'});
  }
  t.after(() => browser.close());
  const page = await browser.newPage({viewport: {width: 1440, height: 900}});
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto(server.url.href);
  assert.equal(await page.locator('#navbar, #soda-notification-preview').count(), 0);
  assert.equal(await page.locator('#sodaspaces-surfaces').isHidden(), true);
  assert.equal(await page.locator('#soda-forgejo-frame').isHidden(), false);
  assert.equal(await page.locator('#soda-workspace-root').isHidden(), false);
  await page.setViewportSize({width: 390, height: 900});
  await page.waitForFunction(() => document.body.classList.contains('sodaspaces-compact'));
  assert.equal(await page.locator('#sodaspaces-surfaces').isHidden(), false);
  assert.equal(await page.locator('#soda-workspace-root').isHidden(), true);
  await page.getByRole('button', {name: 'Terminal', exact: true}).click();
  assert.equal(await page.locator('#soda-forgejo-frame').isHidden(), true);
  assert.equal(await page.locator('#soda-workspace-root').isHidden(), false);
  assert.deepEqual(errors, []);
});

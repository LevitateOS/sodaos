import {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import type {State} from './drawer-fixture';

const root = path.resolve(import.meta.dirname, '../../..');
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;
// F01 composed settings allocation. Unlike the bare drawer mount, this serves
// the production stylesheets and the production management-container structure,
// so host sizing comes from the real rules. The 600 px management height
// reproduces the installed fixed flex-parent measurement; the harness adds only
// that allocation plus a layout-neutral mount wrapper.
export function setupSettingsGeometryDriver() {
  before(async () => {
    const build = await Bun.build({
      entrypoints: [path.join(root, 'tests/frontend/fixtures/drawer-fixture.ts')],
      target: 'browser',
      format: 'esm',
      define: {'process.env.NODE_ENV': '"production"'},
    });
    assert(build.success, `Settings geometry fixture build failed: ${build.logs.join('\n')}`);
    const [fixture] = build.outputs;
    assert(fixture);
    server = Bun.serve({
      hostname: '127.0.0.1',
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === '/assets/drawer-fixture.js')
          return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
        const local =
          url.pathname === '/assets/sodaspaces-drawer.css'
            ? 'frontend/spaces/sodaspaces-project.css'
            : url.pathname === '/assets/sodaspaces-page.css'
              ? 'frontend/spaces/sodaspaces-workspace.css'
              : url.pathname === '/assets/soda/forgejo/components.css'
                ? 'assets/branding/forgejo/components.css'
                : url.pathname === '/assets/soda/forgejo/components-buttons.css'
                  ? 'assets/branding/forgejo/components-buttons.css'
                  : '';
        if (local)
          return new Response(Bun.file(path.join(root, local)), {headers: {'Content-Type': 'text/css'}});
        if (url.pathname !== '/') return new Response(null, {status: 404});
        return new Response(
          '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><style>.soda-workspace-management{height:600px}main{display:contents}body{margin:0}</style><div class="soda-workspace-management"><div><main></main></div></div><script type="module" src="/assets/drawer-fixture.js"></script>',
          {headers: {'Content-Type': 'text/html'}}
        );
      },
    });
    browser = await chromium.launch({headless: true, chromiumSandbox: true});
  });
  after(async () => {
    await browser?.close();
    server?.stop(true);
  });
}

const connections = [
  {owner_id: '1', provider_id: 'codex', id: 'conn-alpha', label: 'Alpha', email: 'alice@example.test', plan: 'team', state: 'connected'},
  {owner_id: '1', provider_id: 'Muse', id: 'conn-beta', label: 'Beta', email: 'alice@example.test', plan: 'solo', state: 'connected'},
  {owner_id: '1', provider_id: 'codex', id: 'conn-gamma', label: 'Gamma', email: 'alice@example.test', plan: 'team', state: 'expired'},
];

export async function settingsFixture(t: TestContext, extra: Partial<State> = {}): Promise<Page> {
  const page = await browser.newPage();
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async ({extra, connections}) => {
    window.drawerFixture = window.createDrawerFixture(extra);
    window.drawerFixture.setReply(async (call) => {
      if (call.url.includes('/identity/connections'))
        return Response.json(call.url.includes('/environments/') ? [] : connections);
      if (call.url.endsWith('/grants') || call.url.endsWith('/leases')) return Response.json([]);
      return null;
    });
    window.drawerFixture.api.setPresentation('settings');
    await window.drawerFixture.api.refresh();
  }, {extra, connections});
  await page.getByRole('heading', {name: 'Identity connections'}).waitFor();
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
  return page;
}

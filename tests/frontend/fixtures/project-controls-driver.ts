import {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import type {State} from './drawer-fixture';

const root = path.resolve(import.meta.dirname, '../../..');
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;
export function setupProjectControlsDriver() {
  before(async () => {
    const build = await Bun.build({
      entrypoints: [path.join(root, 'tests/frontend/fixtures/drawer-fixture.ts')],
      target: 'browser',
      format: 'esm',
      define: {'process.env.NODE_ENV': '"production"'},
    });
    assert(build.success, `Project controls fixture build failed: ${build.logs.join('\n')}`);
    const [fixture] = build.outputs;
    assert(fixture);
    server = Bun.serve({
      hostname: '127.0.0.1',
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === '/assets/drawer-fixture.js')
          return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
        if (url.pathname !== '/') return new Response(null, {status: 404});
        return new Response(
          '<!doctype html><link rel="icon" href="data:,"><button id="native">Native action</button><input id="native-input" value="unsaved"><main></main><script type="module" src="/assets/drawer-fixture.js"></script>',
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
export async function fixture(t: TestContext, extra: Partial<State> = {}) {
  const page = await browser.newPage();
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async (extra) => {
    window.drawerFixture = window.createDrawerFixture(extra);
    await window.drawerFixture.api.ready;
  }, extra);
  return page;
}
export async function refresh(page: Page) {
  await page.evaluate(() => window.drawerFixture.api.refresh());
}
export async function click(page: Page, text: string) {
  await page.evaluate(async (text) => {
    (await window.drawerFixture.showButton(text)).click();
  }, text);
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
}
export const writes = (page: Page) =>
  page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET'));

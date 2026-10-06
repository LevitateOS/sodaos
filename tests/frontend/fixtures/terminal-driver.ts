import {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import type {TerminalFixtureOptions} from './terminal-fixture';

const root = path.resolve(import.meta.dirname, '../../..');
export const env = 'p0123456789abcdef01234567';
export const id = 'a'.repeat(32);
let browser: Browser;
export let server: ReturnType<typeof Bun.serve>;
export function setupTerminalDriver() {
  before(async () => {
    const build = await Bun.build({
      entrypoints: [path.join(root, 'tests/frontend/fixtures/terminal-fixture.ts')],
      target: 'browser',
      format: 'esm',
      minify: true,
      define: {'process.env.NODE_ENV': '"production"'},
      plugins: [
        {
          name: 'terminal-vendor',
          setup(build) {
            build.onResolve({filter: /^\.\/soda-terminal\/(?:xterm|addon-fit)\.mjs$/}, (args) => ({
              path: args.path,
              external: true,
            }));
          },
        },
      ],
    });
    assert(build.success, `Native terminal fixture build failed: ${build.logs.join('\n')}`);
    const [module] = build.outputs;
    assert(module && build.outputs.length === 1);
    server = Bun.serve({
      hostname: '127.0.0.1',
      port: 0,
      fetch(request) {
        const pathname = new URL(request.url).pathname;
        if (pathname === '/assets/terminal-fixture.js')
          return new Response(module, {headers: {'Content-Type': 'text/javascript'}});
        const css =
          pathname === '/assets/soda/forgejo/components.css'
            ? 'assets/branding/forgejo/components.css'
            : pathname === '/assets/sodaspaces-page.css'
              ? 'frontend/spaces/sodaspaces-workspace.css'
              : pathname === '/assets/sodaspaces-terminal.css'
                ? 'frontend/spaces/sodaspaces-terminal.css'
                : '';
        if (css) return new Response(Bun.file(path.join(root, css)), {headers: {'Content-Type': 'text/css'}});
        if (pathname !== '/') return new Response(null, {status: 404});
        return new Response(
          '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><button id="native">Native</button><div id="mount" style="display:flex;height:500px;width:800px"></div><script type="module" src="/assets/terminal-fixture.js"></script>',
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
export async function fixture(t: TestContext, options: TerminalFixtureOptions = {}) {
  const page = await browser.newPage(),
    errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.waitForFunction(() => !!window.createTerminalFixture);
  await page.evaluate(async (options) => {
    window.terminalFixture = window.createTerminalFixture(options);
    await window.terminalFixture.api.ready;
  }, options);
  return page;
}
export async function action(page: Page, name: string | RegExp) {
  if ((await page.locator('.soda-menu').getAttribute('open')) === null)
    await page.getByLabel('Terminal actions', {exact: true}).click();
  await page.getByRole('button', {name, exact: true}).click();
}
export async function opening(page: Page) {
  await action(page, /^(Open|Reconnect) terminal$/);
  await page.waitForFunction(() => window.terminalFixture.sockets.length === 1);
}
export async function ready(page: Page) {
  await opening(page);
  await page.evaluate(() => window.terminalFixture.ready());
}
export async function end(page: Page) {
  await action(page, 'End terminal…');
  await page.getByRole('dialog').getByRole('button', {name: 'End terminal', exact: true}).click();
}
// HTTP native actions are distinct from the new non-starting reservation POST.
export const actions = (page: Page) =>
  page.evaluate(() => window.terminalFixture.writes().filter((call) => call.url.includes('/terminal-sessions/')));
export const inputFrames = (page: Page) =>
  page.evaluate(() => window.terminalFixture.socket().sent.filter((frame) => frame.type !== 'resize'));
export const existing = {id, login: 'original-alice', repository_id: '7'};

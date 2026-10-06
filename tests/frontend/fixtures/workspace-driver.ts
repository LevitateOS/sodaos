import {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Locator, type Page} from 'playwright';
import path from 'node:path';
import type {} from './workspace-fixture';
import {captureSpacesComponent} from '../../../scripts/screenshot';

const root = path.resolve(import.meta.dirname, '../../..');
let browser: Browser, server: ReturnType<typeof Bun.serve>;
// Lit preserves template whitespace in text nodes, so raw text contents vary with
// source formatting. Collapse runs like rendered HTML does.
export async function textContents(locator: Locator): Promise<string[]> {
  return (await locator.allTextContents()).map((s) => s.replace(/\s+/g, ' ').trim());
}
export function setupWorkspaceDriver() {
  before(async () => {
    const build = await Bun.build({
      entrypoints: [path.join(root, 'tests/frontend/fixtures/workspace-fixture.ts')],
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
    assert(build.success, `Native workspace fixture build failed: ${build.logs.join('\n')}`);
    const [fixture] = build.outputs;
    assert(fixture && build.outputs.length === 1);
    server = Bun.serve({
      hostname: '127.0.0.1',
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === '/assets/workspace-fixture.js')
          return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
        const local =
          url.pathname === '/assets/sodaspaces-drawer.css'
            ? 'frontend/spaces/sodaspaces-project.css'
            : url.pathname === '/assets/sodaspaces-page.css'
              ? 'frontend/spaces/sodaspaces-workspace.css'
              : url.pathname === '/assets/sodaspaces-terminal.css'
                ? 'frontend/spaces/sodaspaces-terminal.css'
                : url.pathname.startsWith('/assets/soda-terminal/')
                  ? '.artifacts/browser-terminal/vendor/' + path.basename(url.pathname)
                  : url.pathname.startsWith('/assets/soda/forgejo/icons/')
                    ? 'assets/branding/icons/octicons/' + path.basename(url.pathname)
                    : url.pathname.startsWith('/assets/soda/')
                      ? 'assets/branding/' + url.pathname.slice('/assets/soda/'.length)
                      : '';
        if (local)
          return new Response(Bun.file(path.join(root, local)), {
            headers: {
              'Content-Type': /\.m?js$/.test(local)
                ? 'text/javascript'
                : local.endsWith('.css')
                  ? 'text/css'
                  : local.endsWith('.svg')
                    ? 'image/svg+xml'
                    : 'application/octet-stream',
            },
          });
        if (url.pathname !== '/') return new Response(null, {status: 404});
        return new Response(
          '<!doctype html><style>.ui.button{display:inline-flex;justify-content:center;text-align:center}</style><meta name="soda-component-fixture" content="spaces"><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><style>main{height:calc(100dvh - 40px)}body{margin:0}</style><input id="native-draft" value="unsaved"><main></main><script type="module" src="/assets/workspace-fixture.js"></script>',
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
export async function fixture(
  t: TestContext,
  mode: 'native' | 'page' = 'page',
  beforeMount?: () => void,
  firstUse = false,
  subpath = ''
) {
  const page = await browser.newPage({viewport: {width: 1440, height: 1000}}),
    errors: string[] = [];
  page.setDefaultTimeout(5000);
  page.on('pageerror', (e) => errors.push(e.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.waitForFunction(() => !!window.createWorkspaceFixture);
  if (firstUse)
    await page.evaluate(() => {
      // Exercise native container/button CSS precedence without claiming native HTML.
      const mount = document.querySelector('main');
      if (!mount) throw Error('Missing fixture mount');
      const shell = document.createElement('div'),
        container = document.createElement('div');
      shell.className = 'page-content soda-page';
      container.className = 'soda-page-container';
      mount.dataset.view = 'spaces';
      mount.before(shell);
      shell.append(container);
      container.append(mount);
    });
  if (beforeMount) await page.evaluate(beforeMount);
  await page.evaluate(
    async ({mode, firstUse, subpath}) => {
      window.workspaceFixture = window.createWorkspaceFixture(mode, firstUse, subpath);
      await window.workspaceFixture.api.ready;
    },
    {mode, firstUse, subpath}
  );
  return page;
}
export async function openSession(page: Page, name: string) {
  await page.getByRole('button', {name: /^(Sessions|Projects)$/}).click();
  await page.locator('.soda-session-list button').filter({hasText: name}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
}
export async function action(page: Page, name: string) {
  await page.locator('.soda-workspace-terminal:not([hidden]) summary[aria-label="Terminal actions"]').click();
  await page.locator('.soda-workspace-terminal:not([hidden])').getByRole('button', {name, exact: true}).click();
}
export async function create(page: Page, name = 'New build') {
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
  await action(page, 'Rename terminal');
  await page.getByLabel('Session name', {exact: true}).fill(name);
  await page.getByRole('button', {name: 'Save name', exact: true}).click();
}
export async function paneAction(page: Page, name: string) {
  await page.getByLabel('Pane actions', {exact: true}).click();
  await page.getByRole('button', {name, exact: true}).click();
}
export async function chooseFirstRepository(page: Page, capture?: string) {
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  if (capture) await captureSpacesComponent(page, 'picker-' + capture);
  const picker = {
    frame: await page.locator('.soda-workspace-frame').boundingBox(),
    heading: await page.getByRole('heading', {name: 'Choose a repository'}).boundingBox(),
    action: await page.getByRole('button', {name: 'Continue', exact: true}).boundingBox(),
  };
  const selection = await page
    .locator('.soda-repository-choice:has(input:checked)')
    .evaluate((node) => ({background: getComputedStyle(node).backgroundColor, edge: getComputedStyle(node).boxShadow}));
  assert.notEqual(selection.edge, 'none', 'selection needs a shape treatment beyond radio color');
  assert.equal(
    await page.getByRole('navigation', {name: 'Repository pages'}).count(),
    0,
    'single-page results should not show pagination'
  );
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
  await page.waitForFunction(() => !!document.querySelector('.soda-project-journey button.primary:not([disabled])'));
  return picker;
}

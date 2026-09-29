import test, {after, before} from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {chromium, type Browser} from 'playwright';
import type {} from './fixtures/persistent-panel-fixture';

const root = path.resolve(import.meta.dirname, '../..');
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;

before(async () => {
  const build = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/persistent-panel-fixture.ts')],
    target: 'browser',
    format: 'esm',
    minify: true,
    define: {'process.env.NODE_ENV': '"production"'},
    plugins: [
      {
        name: 'terminal-vendor',
        setup(builder) {
          builder.onResolve({filter: /^\.\/soda-terminal\/(?:xterm|addon-fit)\.mjs$/}, (args) => ({
            path: args.path,
            external: true,
          }));
        },
      },
    ],
  });
  assert(build.success, `Persistent panel fixture build failed: ${build.logs.join('\n')}`);
  const [script] = build.outputs;
  assert(script);
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url);
      if (url.pathname === '/assets/persistent-panel.js')
        return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      const styles: Record<string, string> = {
        'components.css': 'assets/branding/forgejo/components.css',
        'sodaspaces-page.css': 'frontend/spaces/sodaspaces-workspace.css',
        'sodaspaces-drawer.css': 'frontend/spaces/sodaspaces-project.css',
        'sodaspaces-terminal.css': 'frontend/spaces/sodaspaces-terminal.css',
      };
      const file = url.pathname.startsWith('/assets/soda-terminal/')
        ? '.artifacts/browser-terminal/vendor/' + path.basename(url.pathname)
        : url.pathname.startsWith('/assets/')
          ? styles[url.pathname.slice('/assets/'.length)]
          : undefined;
      if (file)
        return new Response(Bun.file(path.join(root, file)), {
          headers: {'Content-Type': file.endsWith('.mjs') ? 'text/javascript' : 'text/css'},
        });
      if (url.pathname === '/native/one' || url.pathname === '/native/two')
        return new Response(
          `<!doctype html><title>${url.pathname}</title><a href="/native/two">Other repository</a><form action="/native/two"><input name="draft" value="unsaved"></form>`,
          {
            headers: {'Content-Type': 'text/html'},
          }
        );
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" href="data:,"><style>body{margin:0}.extension-workspace-layout{display:flex;height:calc(100dvh - 48px)}iframe{flex:1;min-width:0}.extension-workspace-panels{width:50%;min-width:0}.extension-workspace-panel,[data-extension-panel]{height:100%;min-height:0}h2{position:absolute;clip-path:inset(50%)}</style><main data-extension-workspace><button id="focus-native">Focus browsing</button><div class="extension-workspace-layout"><iframe title="Forgejo pages" src="/native/one"></iframe><aside class="extension-workspace-panels"><section class="extension-workspace-panel"><h2>Workspace</h2><div data-extension-panel></div></section></aside></div></main><script type="module" src="/assets/persistent-panel.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  browser = await chromium.launch({
    headless: true,
    chromiumSandbox: true,
    ...(process.env.CHROME ? {executablePath: process.env.CHROME} : {}),
  });
});

after(async () => {
  await browser?.close();
  server?.stop(true);
});

test('native panel keeps its Lit/xterm owner, socket and target while native pages navigate', async (t) => {
  const page = await browser.newPage({viewport: {width: 1280, height: 900}});
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.locator('[data-extension-panel] soda-spaces').waitFor();
  assert.equal(
    await page.evaluate(() => window.persistentPanelFixture.model.calls.filter((call) => call.method !== 'GET').length),
    0
  );
  assert.equal(await page.evaluate(() => window.persistentPanelFixture.model.sockets.length), 0);
  await page.getByRole('button', {name: 'Sessions', exact: true}).click();
  await page.locator('.soda-session-list button').filter({hasText: 'Build'}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
  const screen = await page.locator('.xterm').first().elementHandle();
  assert(screen);
  const renderer = await page.evaluateHandle(
    () => (document.querySelector('soda-terminal') as HTMLElement & {terminal?: object}).terminal
  );
  assert(await renderer.evaluate((value) => value !== undefined), 'xterm renderer is mounted');
  const socket = await page.evaluateHandle(() => window.persistentPanelFixture.model.sockets[0]);
  const original = await page.evaluate(() => {
    const active = window.persistentPanelFixture.model.sockets[0];
    if (!active) throw Error('Missing retained socket');
    return {target: active.sent[0]?.id, action: active.sent[0]?.action};
  });
  assert.equal(original.action, 'attach');
  const native = page.frameLocator('iframe[title="Forgejo pages"]');
  await native.locator('input[name=draft]').fill('pending native form');
  await page.locator('.xterm-helper-textarea').focus();
  assert.equal(await native.locator('input[name=draft]').inputValue(), 'pending native form');
  await native.getByRole('link', {name: 'Other repository'}).click();
  await native.locator('input[name=draft]').waitFor();
  await native.locator('body').evaluate(() => history.back());
  await page.waitForFunction(
    () =>
      document.querySelector<HTMLIFrameElement>('iframe[title="Forgejo pages"]')?.contentWindow?.location.pathname ===
      '/native/one'
  );
  await native.locator('input[name=draft]').fill('submitted draft');
  await native.locator('form').evaluate((form) => (form as HTMLFormElement).requestSubmit());
  await page.waitForFunction(
    () =>
      document.querySelector<HTMLIFrameElement>('iframe[title="Forgejo pages"]')?.contentWindow?.location.search ===
      '?draft=submitted+draft'
  );
  await page.setViewportSize({width: 390, height: 780});
  await page.locator('#focus-native').focus();
  assert(await screen.evaluate((node) => node.isConnected));
  assert(
    await page.evaluate(
      (original) =>
        (document.querySelector('soda-terminal') as HTMLElement & {terminal?: object}).terminal === original,
      renderer
    ),
    'xterm renderer remains the same object'
  );
  assert(await page.evaluate((original) => window.persistentPanelFixture.model.sockets[0] === original, socket));
  const preserved = await page.evaluate(() => {
    const model = window.persistentPanelFixture.model;
    return {
      sockets: model.sockets.length,
      closed: model.sockets[0]?.closed,
      action: model.sockets[0]?.sent[0]?.action,
      target: model.sockets[0]?.sent[0]?.id,
      writes: model.calls.filter((call) => call.method !== 'GET').length,
      owner: document.querySelectorAll('[data-extension-panel] soda-spaces').length,
      screenWidth: document.querySelector('.xterm')?.getBoundingClientRect().width || 0,
    };
  });
  const {screenWidth, ...state} = preserved;
  assert.deepEqual(state, {
    sockets: 1,
    closed: 0,
    action: 'attach',
    target: original.target,
    writes: 0,
    owner: 1,
  });
  assert(screenWidth > 0, 'xterm remains laid out in the native panel');
  await page.evaluate(async () => {
    window.persistentPanelFixture.model.setUser('2');
    await (document.querySelector('soda-spaces') as HTMLElement & {refresh(): Promise<void>}).refresh();
  });
  await page.waitForFunction(() => window.persistentPanelFixture.model.sockets[0]?.closed === 1);
  assert.equal(await page.locator('.soda-workspace-terminal .is-connected').count(), 0);
});

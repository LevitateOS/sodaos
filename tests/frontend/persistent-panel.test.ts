import test, {after, before} from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {chromium, type Browser, type Page} from 'playwright';
import type {} from './fixtures/persistent-panel-fixture';

const root = path.resolve(import.meta.dirname, '../..');
const generation = 'fixture-generation';
const assets = '/forge/-/native/assets/soda/';
const globalPath = '/forge/workspace?view=global#notes';
const repoPath = '/forge/team/repo/workspace?view=repo#details';
const nativePath = '/forge/team/repo/issues?q=open#details';
const newPath = '/forge/workspace/new?from=fixture#notes';
const requestPath = '/forge/workspace/requests/d' + '1'.repeat(24) + '?view=request#details';
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;
let liveGeneration = generation;

// Authored HTML follows current native base/content/panels envelopes. It does
// not establish Go-rendered or installed behavior.
function nativeDocument(url: URL): string {
  const repository = url.pathname.startsWith('/forge/team/repo');
  const direct = url.pathname.includes('/workspace');
  const title = direct ? (repository ? 'Repository Workspace' : 'Global Workspace') : 'Native issues';
  const context = repository ? ' data-extension-repository-id="7"' : '';
  return `<!doctype html><html><head><title>${title}</title><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="soda-component-fixture" content="native-host"><link rel="icon" href="data:,"><link rel="stylesheet" href="${assets}soda-terminal/xterm.css"><style>body{margin:0}#navbar{display:flex;gap:1rem;padding:8px}.page-content{padding:8px}.extension-workspace-panels:not([hidden]){height:calc(100dvh - 120px)}.extension-workspace-panel,[data-extension-panel]{height:100%;min-height:0}.extension-workspace-panel>h2{position:absolute;clip-path:inset(50%)}</style></head>
  <body data-page-data='{}'><nav id="navbar" aria-label="Native pages">
  <a href="${globalPath}">Global Workspace</a><a href="${repoPath}">Repository Workspace</a><a href="${nativePath}">Native issues</a><a href="${newPath}">New project</a><a href="${requestPath}">Saved request</a></nav>
  <main class="page-content${repository ? ' repository' : ''}">${repository ? '<div class="repo-header">team/repo</div>' : ''}<section class="${direct ? 'extension-page extension-page-owner-presented' : 'native-issues'}"><h1 id="details">${title}</h1><span id="notes">Native content</span><form action="/forge/team/repo/issues" method="get"><input name="draft" value="unsaved"><button type="submit">Submit native search</button></form></section></main>
  <aside id="extension-panels" class="extension-workspace-panels" aria-label="Workspace panels" data-extension-panels data-app-sub-url="/forge" data-extension-subroutes='["/workspace/new","/workspace/requests/{request}"]'${context} data-extension-session-generation="${generation}"${direct ? '' : ' hidden inert'}>
  <section class="extension-workspace-panel"><h2>Workspace</h2><div data-extension-panel data-extension-contract-version="1" data-app-sub-url="/forge" data-extension-id="soda" data-extension-panel-id="workspace" data-extension-owner-id="workspace" data-extension-entry="${assets}persistent-panel.js" data-extension-api-base="/forge/-/native/pages/soda/spaces/api/" data-extension-asset-base="${assets}" data-extension-session-generation="${generation}"${context}></div></section></aside>
  <script type="module" src="${assets}persistent-panel.js"></script></body></html>`;
}

before(async () => {
  const builds = await Promise.all([
    Bun.build({
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
    }),
    Bun.build({
      entrypoints: [path.resolve(root, '../forgejo-ext/web_src/js/features/extensions.js')],
      target: 'browser',
      format: 'esm',
    }),
  ]);
  const [fixtureBuild, nativeBuild] = builds;
  assert(fixtureBuild && nativeBuild);
  for (const build of builds) assert(build.success, `Authored native fixture build failed: ${build.logs.join('\n')}`);
  const fixture = fixtureBuild.outputs[0],
    native = nativeBuild.outputs[0];
  assert(fixture && native);
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url);
      if (url.pathname === assets + 'persistent-panel.js')
        return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname === '/assets/native-host.js')
        return new Response(native, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname === '/forge/-/native/session')
        return new Response(null, {status: 204, headers: {'X-Extension-Session-Generation': liveGeneration}});
      if (url.pathname.startsWith(assets)) {
        const styles: Record<string, string> = {
          'components.css': 'assets/branding/forgejo/components.css',
          'sodaspaces.css': 'frontend/spaces/sodaspaces.css',
          'sodaspaces-page.css': 'frontend/spaces/sodaspaces-workspace.css',
          'sodaspaces-drawer.css': 'frontend/spaces/sodaspaces-project.css',
          'sodaspaces-terminal.css': 'frontend/spaces/sodaspaces-terminal.css',
        };
        const relative = url.pathname.slice(assets.length);
        const file = relative.startsWith('soda-terminal/')
          ? '.artifacts/browser-terminal/vendor/' + path.basename(relative)
          : styles[relative];
        if (file)
          return new Response(Bun.file(path.join(root, file)), {
            headers: {'Content-Type': file.endsWith('.mjs') ? 'text/javascript' : 'text/css'},
          });
      }
      if (
        [
          '/forge/workspace',
          '/forge/workspace/new',
          '/forge/team/repo/workspace',
          '/forge/team/repo/issues',
          requestPath.split('?')[0],
        ].includes(url.pathname)
      )
        return new Response(nativeDocument(url), {
          headers: {'Content-Type': 'text/html', 'X-Extension-Session-Generation': liveGeneration},
        });
      return new Response(null, {status: 404});
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

async function enter(page: Page, name: string, expected: string) {
  await page.getByRole('navigation', {name: 'Native pages'}).getByRole('link', {name, exact: true}).click();
  await page.waitForURL(server.url.origin + expected);
  await page.waitForFunction(
    (route) => window.persistentPanelFixture.lifecycle.some((call) => call.presentation?.route === route),
    expected
  );
}

for (const [order, first] of [
  ['native-first', nativePath],
  ['workspace-first', repoPath],
] as const) {
  test(`authored current native host retains the actual Soda terminal (${order})`, async (t) => {
    liveGeneration = generation;
    const page = await browser.newPage({viewport: {width: 1280, height: 900}});
    page.setDefaultTimeout(5000);
    const errors: string[] = [];
    page.on('pageerror', (error) => errors.push(error.message));
    t.after(async () => {
      await page.close();
      assert.deepEqual(errors, []);
      liveGeneration = generation;
    });
    await page.goto(server.url.origin + first);
    await page.locator('[data-extension-panel] soda-spaces').waitFor({state: 'attached'});
    if (order === 'native-first') {
      assert.equal(await page.locator('[data-extension-panels]').isVisible(), false);
      await enter(page, 'Global Workspace', globalPath);
    }
    await page.getByRole('button', {name: 'team/repo', exact: true}).click();
    await page.getByRole('button', {name: 'Terminals', exact: true}).click();
    await page.getByLabel('Workspace options', {exact: true}).click();
    await page.locator('.soda-session-list button').filter({hasText: 'Build'}).click();
    await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
    const terminal = await page.locator('soda-terminal').first().elementHandle();
    const screen = await page.locator('.xterm').first().elementHandle();
    assert(terminal && screen, 'real terminal element and xterm screen are required');
    const renderer = await terminal.evaluateHandle((node) => {
      const value = (node as HTMLElement & {terminal?: object}).terminal;
      if (!value) throw Error('Missing actual xterm renderer');
      return value;
    });
    const socket = await page.evaluateHandle(() => {
      const value = window.persistentPanelFixture.model.sockets[0];
      if (!value) throw Error('Missing exact retained socket');
      return value;
    });
    const originalTarget = await socket.evaluate((value) => value.sent[0]?.id);
    assert.equal(await socket.evaluate((value) => value.sent[0]?.action), 'attach');
    assert.equal(originalTarget, 'a'.repeat(32));

    await page.locator('input[name=draft]').fill('pending native form');
    await page.locator('.xterm-helper-textarea').focus();
    assert.equal(await page.locator('input[name=draft]').inputValue(), 'pending native form');
    await enter(page, 'Repository Workspace', repoPath);
    assert.equal(await page.title(), 'Repository Workspace');
    assert.equal(await page.locator('.repo-header').textContent(), 'team/repo');
    assert.deepEqual(
      await page.evaluate(
        () =>
          [...window.persistentPanelFixture.lifecycle].reverse().find((call) => call.method === 'update')?.presentation
      ),
      {
        route: repoPath,
        params: {view: 'repo'},
        repositoryId: '7',
        mode: 'page',
        visible: true,
      }
    );
    await enter(page, 'Global Workspace', globalPath);
    await page.goBack();
    await page.waitForURL(server.url.origin + repoPath);
    await page.waitForFunction(() => document.title === 'Repository Workspace');
    await page.goForward();
    await page.waitForURL(server.url.origin + globalPath);
    await page.waitForFunction(() => document.title === 'Global Workspace');
    await enter(page, 'Native issues', nativePath);
    assert.equal(await page.locator('[data-extension-panels]').isVisible(), false);
    await page.locator('input[name=draft]').fill('submitted draft');
    await page.getByRole('button', {name: 'Submit native search'}).click();
    await page.waitForURL(server.url.origin + '/forge/team/repo/issues?draft=submitted+draft');
    await enter(page, 'Global Workspace', globalPath);
    await enter(page, 'New project', newPath);
    await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
    assert.equal(await page.getByLabel('Describe what you want to build', {exact: true}).inputValue(), '');
    await enter(page, 'Saved request', requestPath);
    await page.waitForFunction(
      () =>
        document.querySelector('soda-intent')?.shadowRoot?.querySelector<HTMLTextAreaElement>('textarea')?.value ===
        'Retained fixture request'
    );
    await page.getByRole('button', {name: 'Back', exact: true}).click();
    await enter(page, 'Global Workspace', globalPath);

    const resumes = await page.evaluate(
      () => window.persistentPanelFixture.lifecycle.filter((call) => call.method === 'resume').length
    );
    await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})));
    await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true})));
    await page.waitForFunction(
      (count) => window.persistentPanelFixture.lifecycle.filter((call) => call.method === 'resume').length > count,
      resumes
    );
    await page.setViewportSize({width: 390, height: 780});
    assert(await terminal.evaluate((node) => node.isConnected));
    assert(await screen.evaluate((node) => node.isConnected));
    assert(
      await terminal.evaluate((node, value) => (node as HTMLElement & {terminal?: object}).terminal === value, renderer)
    );
    assert(await page.evaluate((value) => window.persistentPanelFixture.model.sockets[0] === value, socket));
    assert.equal(await socket.evaluate((value) => value.closed), 0);
    assert.equal(await socket.evaluate((value) => value.sent[0]?.id), originalTarget);
    assert.equal(await page.locator('iframe').count(), 0);
    assert.equal(await page.locator('[data-extension-panel] soda-spaces').count(), 1);
    const state = await page.evaluate(() => {
      const {model, lifecycle, initialWrites} = window.persistentPanelFixture;
      return {
        sockets: model.sockets.length,
        mounts: lifecycle.filter((call) => call.method === 'mount').length,
        retired: lifecycle.filter((call) => call.method === 'retire').length,
        writes: model.calls.filter((call) => call.method !== 'GET').length - initialWrites,
        suspend: lifecycle.some((call) => call.method === 'suspend'),
      };
    });
    assert.deepEqual(state, {sockets: 1, mounts: 1, retired: 0, writes: 0, suspend: true});
    assert(
      await screen.evaluate((node) => node.getBoundingClientRect().width > 0),
      'retained xterm is laid out after returning to Workspace'
    );
    liveGeneration = 'fixture-session-changed';
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await page.waitForFunction(() =>
      window.persistentPanelFixture.lifecycle.some(
        (call) => call.method === 'retire' && call.reason === 'session-changed'
      )
    );
    assert((await socket.evaluate((value) => value.closed)) > 0);
    assert.equal(await page.locator('.is-connected').count(), 0);
    assert.equal(
      await page.evaluate(() =>
        window.persistentPanelFixture.model.calls.some(
          (call) => (call.path.endsWith('/api/environments') && call.method === 'POST') || call.body?.action === 'end'
        )
      ),
      false
    );
  });
}

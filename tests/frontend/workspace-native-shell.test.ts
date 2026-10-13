import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {setupWorkspaceDriver, fixture, openSession} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('Fountain direct Workspace preserves native paths, history and the terminal owner', async (t) => {
  const build = await Bun.build({
    entrypoints: [path.resolve(import.meta.dirname, '../../../forgejo-ext/web_src/js/features/extensions.js')],
    target: 'browser',
    format: 'esm',
  });
  assert(build.success);
  const module = await build.outputs[0]?.text();
  assert(module);
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const generation = await page.evaluate(() => window.workspaceFixture.generation);
  await page.route('**/native-shell.js', (route) => route.fulfill({contentType: 'text/javascript', body: module}));
  await page.route('**/forge/-/native/session', (route) =>
    route.fulfill({status: 204, headers: {'X-Extension-Session-Generation': generation}})
  );
  await page.route('**/forge/**/workspace?*', (route) => {
    const url = new URL(route.request().url());
    const repo = url.pathname.includes('/team/repo/');
    return route.fulfill({
      contentType: 'text/html',
      headers: {'X-Extension-Session-Generation': generation},
      body: `<html><head><title>${repo ? 'team/repo' : 'Global'} Workspace</title></head><body><main data-workspace-direct data-extension-repository-id="${repo ? '7' : ''}" data-workspace-session-generation="${generation}"><div data-extension-workspace-direct-content>${repo ? '<div class="repo-header">team/repo</div>' : ''}<h1 id="details">Workspace</h1></div></main></body></html>`,
    });
  });
  await page.evaluate(() => {
    const f = window.workspaceFixture;
    f.api.dispose();
    history.replaceState(null, '', '/forge/workspace?view=first#notes');
    const navbar = document.createElement('nav');
    navbar.id = 'navbar';
    navbar.innerHTML =
      '<a id="native-repository" href="/forge/team/repo/workspace?view=all#details">Repository Workspace</a><a id="native-global" href="/forge/workspace?view=global#notes">Global Workspace</a><a id="native-account" href="/forge/user/settings/security">Account controls</a>';
    const host = document.createElement('main');
    host.dataset.extensionWorkspace = '';
    host.dataset.workspaceDirect = '';
    host.dataset.extensionRepositoryId = '';
    host.dataset.workspacePath = '/forge/workspace?view=first';
    host.dataset.appSubUrl = '/forge';
    host.dataset.workspaceSessionGeneration = f.generation;
    host.innerHTML =
      '<div data-extension-workspace-direct-content><h1>Workspace</h1></div><aside class="extension-workspace-panels"><section class="extension-workspace-panel"></section></aside>';
    f.root.before(navbar, host);
    host.querySelector('section')?.append(f.root);
    const apiBase = location.origin + '/-/native/pages/soda/spaces/api/';
    window.mountWorkspacePanel(f.root, {
      extensionId: 'soda',
      panelId: 'workspace',
      ownerId: 'workspace',
      appSubUrl: '',
      apiBase,
      assetBase: location.origin + '/assets/',
      sessionGeneration: f.generation,
      contractVersion: 1,
      request: (path: string, init: RequestInit = {}) =>
        fetch(apiBase + path, {
          ...init,
          credentials: 'same-origin',
          redirect: 'error',
          headers: new Headers({
            ...Object.fromEntries(new Headers(init.headers)),
            'X-Extension-Session-Generation': f.generation,
          }),
        }),
      stream: (path: string) => {
        const url = new URL(apiBase + path, location.origin);
        url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
        const socket = new WebSocket(url);
        return {socket, close: (code?: number, reason?: string) => socket.close(code, reason)};
      },
      navigate: () => {},
    });
  });
  await page.addScriptTag({
    type: 'module',
    content: 'import {initExtensionWorkspace} from "/native-shell.js"; initExtensionWorkspace();',
  });
  await page.locator('soda-spaces .is-connected').waitFor();
  const renderer = await page.locator('.xterm').elementHandle();
  const sockets = await page.evaluate(() => window.workspaceFixture.sockets.length);
  await page.locator('#native-repository').click();
  await page.waitForURL('**/forge/team/repo/workspace?view=all#details');
  assert.equal(await page.title(), 'team/repo Workspace');
  assert.equal(await page.locator('.repo-header').textContent(), 'team/repo');
  await page.locator('#native-global').click();
  await page.waitForURL('**/forge/workspace?view=global#notes');
  await page.goBack();
  await page.waitForURL('**/forge/team/repo/workspace?view=all#details');
  await page.waitForFunction(() => document.title === 'team/repo Workspace');
  await page.goForward();
  await page.waitForURL('**/forge/workspace?view=global#notes');
  await page.waitForFunction(() => document.title === 'Global Workspace');
  assert.equal(await page.locator('iframe').count(), 0);
  assert.equal(await page.locator('soda-spaces').count(), 1);
  assert(await renderer?.evaluate((node) => node.isConnected));
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), sockets);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.at(-1)?.closed), 0);
  assert.equal(await page.locator('#native-account').getAttribute('href'), '/forge/user/settings/security');
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length),
    0
  );
});

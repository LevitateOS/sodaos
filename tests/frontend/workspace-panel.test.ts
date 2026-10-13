import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('native Workspace presentation retains one owner, renderer and socket', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await page.evaluate(() => {
    const f = window.workspaceFixture;
    f.api.dispose();
    const host = document.createElement('main');
    host.dataset.extensionWorkspace = '';
    host.dataset.workspaceDirect = '';
    host.dataset.extensionRepositoryId = '7';
    f.root.before(host);
    const aside = document.createElement('aside');
    aside.className = 'extension-workspace-panels';
    const section = document.createElement('section');
    section.className = 'extension-workspace-panel';
    host.append(aside);
    aside.append(section);
    section.append(f.root);
    const apiBase = location.origin + '/-/native/pages/soda/spaces/api/';
    (window as unknown as {workspacePanelHandle: unknown}).workspacePanelHandle = window.mountWorkspacePanel(f.root, {
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
  await page.locator('soda-spaces .is-connected').waitFor();
  const terminal = await page.locator('.xterm').elementHandle();
  const owner = await page.locator('soda-spaces').elementHandle();
  assert.equal(await page.locator('soda-spaces').count(), 1);
  assert.equal(await page.locator('iframe').count(), 0, 'direct Workspace has no full-page frame');
  const sockets = await page.evaluate(() => window.workspaceFixture.sockets.length);
  for (const [full, repository] of [
    [false, '7'],
    [true, '8'],
    [true, '7'],
  ] as const) {
    await page.evaluate(
      ({full, repository}) => {
        const handle = (window as unknown as {workspacePanelHandle: {update: (p: unknown) => void}})
          .workspacePanelHandle;
        handle.update({
          mode: full ? 'page' : 'panel',
          route: '/workspace',
          params: {},
          repositoryId: repository,
          visible: true,
        });
      },
      {full, repository}
    );
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await page.locator('soda-spaces').count(), 1);
    assert(await owner?.evaluate((node) => node.isConnected));
    assert(await terminal?.evaluate((node) => node.isConnected), 'renderer is never disconnected by presentation');
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), sockets);
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.at(-1)?.closed), 0);
  }
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length),
    0
  );
});

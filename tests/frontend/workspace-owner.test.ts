import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {setupWorkspaceDriver, fixture, openSession} from './fixtures/workspace-driver';

setupWorkspaceDriver();

test('the manifest declares no global Workspace page or panel', async () => {
  const manifest = JSON.parse(
    await readFile(new URL('../../system/containers/extension/extension.json', import.meta.url), 'utf8')
  ) as {pages: Array<{id: string; title: string}>; panels?: unknown[]};
  assert(!manifest.pages.some((page) => page.id === 'spaces' || page.title === 'Workspace'));
  assert.equal(manifest.panels, undefined);
});

test('shared Workspace entry returns a real MountHandle without host-ancestor workarounds', async () => {
  const entry = await readFile(new URL('../../frontend/spaces/soda-spaces-entry.ts', import.meta.url), 'utf8');
  for (const callback of ['update', 'suspend', 'resume', 'retire', 'dispose']) {
    assert(entry.includes(callback), `entry handles ${callback}`);
  }
  assert(!entry.includes('frameElement'), 'no frame detection');
  assert(!entry.includes('data-workspace-direct'), 'no competing-panel detection');
  assert(!entry.includes('data-extension-workspace'), 'no host ancestor inspection');
  const transport = await readFile(new URL('../../frontend/spaces/soda-extension.ts', import.meta.url), 'utf8');
  assert(!transport.includes('extensionURL'), 'no duplicate generic URL construction');
  assert(!transport.includes('new WebSocket'), 'no local stream fallback');
  assert(transport.includes('hostRequest'), 'host request helper is consumed');
  assert(transport.includes('hostStream'), 'host stream helper is consumed');
  assert(transport.includes('hostNavigate'), 'host navigate helper is consumed');
});

test('canonical mirror carries the shared-owner and navigation contract', async () => {
  const mirror = await readFile(
    new URL('../../frontend/spaces/fountain-browser-contract.d.ts', import.meta.url),
    'utf8'
  );
  assert(mirror.includes('ownerId'), 'mirror declares ownerId');
  assert(mirror.includes('navigate'), 'mirror declares navigate');
});

test('setup requests host push/replace navigation and waits for the entered presentation', async (t) => {
  const page = await fixture(t, 'page', undefined, false, '/forgejo.test');
  await page.evaluate(async () => {
    await window.workspaceFixture.api.refresh();
    window.workspaceFixture.presentRoute('/forgejo.test/workspace');
    window.workspaceFixture.deferNavigation(true);
  });
  const before = await page.evaluate(() => ({url: location.href, history: history.length}));
  await page.getByRole('button', {name: 'New project', exact: true}).click();
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.navigations), [
    {path: '/forgejo.test/workspace/new', replace: false},
  ]);
  // A repeated current-base update is not entry into the requested new route.
  await page.evaluate(() => window.workspaceFixture.presentRoute('/forgejo.test/workspace'));
  await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => ({url: location.href, history: history.length})), before);
  await page.evaluate(() => window.workspaceFixture.presentRoute('/forgejo.test/workspace/new'));
  await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
  await page.getByRole('button', {name: 'Save draft', exact: true}).click();
  await page.waitForFunction(() => window.workspaceFixture.navigations.length === 2);
  const saved = await page.evaluate(() => window.workspaceFixture.navigations[1]);
  assert(saved);
  assert.match(saved.path, /^\/forgejo\.test\/workspace\/requests\/d[0-9a-f]{24}$/);
  assert.equal(saved.replace, true);
  assert.deepEqual(await page.evaluate(() => ({url: location.href, history: history.length})), before);
  await page.evaluate((route) => window.workspaceFixture.presentRoute(route), saved.path);
  await page.getByText('Saved', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back', exact: true}).click();
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.navigations.at(-1)), {
    path: '/forgejo.test/workspace',
    replace: false,
  });
  assert.deepEqual(await page.evaluate(() => ({url: location.href, history: history.length})), before);
});

for (const outcome of ['success', 'failure'] as const)
  test(`host Workspace entry waits for ${outcome} save and restores the same terminal without rerouting`, async (t) => {
    const page = await fixture(t);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await openSession(page, 'Build');
    const screen = await page.locator('.xterm').elementHandle();
    const before = await page.evaluate(() => ({
      layout: sessionStorage.getItem('soda-spaces:v3:1'),
      writes: window.workspaceFixture.calls.filter(
        (call) => call.method !== 'GET' && !call.path.includes('/api/factory/')
      ).length,
      sockets: window.workspaceFixture.sockets.length,
    }));
    await page.getByRole('button', {name: 'New project', exact: true}).click();
    await page.getByLabel('Describe what you want to build', {exact: true}).fill('Build a small project tracker');
    await page.evaluate((outcome) => {
      const original = window.fetch;
      Object.defineProperty(window, 'fetch', {
        configurable: true,
        value: async (input: RequestInfo | URL, init?: RequestInit) => {
          if (String(input).endsWith('/api/factory/intents') && init?.method === 'POST') {
            await new Promise<void>((resolve) =>
              window.addEventListener('release-intent-save', () => resolve(), {once: true})
            );
            if (outcome === 'failure') throw Error('Synthetic save failure');
          }
          return original(input, init);
        },
      });
    }, outcome);
    await page.getByRole('button', {name: 'Save draft', exact: true}).click();
    await page.waitForFunction(() => (document.querySelector('soda-intent') as unknown as {busy: boolean})?.busy);
    const requests = await page.evaluate(() => window.workspaceFixture.navigations.length);
    await page.evaluate(() => {
      // Authored host delivery, deliberately separate from consumer navigation.
      history.pushState({}, '', '/workspace');
      window.workspaceFixture.presentRoute();
    });
    await page.getByRole('heading', {name: 'What do you want to build?', exact: true}).waitFor();
    await page.evaluate(() => window.dispatchEvent(new Event('release-intent-save')));
    await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
    assert(await screen?.evaluate((node) => node.isConnected));
    assert.equal(await page.evaluate(() => location.pathname), '/workspace');
    assert.equal(await page.evaluate(() => window.workspaceFixture.navigations.length), requests);
    assert.deepEqual(
      await page.evaluate(() => ({
        layout: sessionStorage.getItem('soda-spaces:v3:1'),
        writes: window.workspaceFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.path.includes('/api/factory/')
        ).length,
        sockets: window.workspaceFixture.sockets.length,
      })),
      before
    );
    assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((socket) => socket.closed)), [0]);
  });

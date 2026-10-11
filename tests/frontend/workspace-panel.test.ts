import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('native panel yields saved terminal restoration to the full Spaces page', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await page.evaluate(() => {
    const f = window.workspaceFixture;
    f.api.dispose();
    const host = document.createElement('div');
    host.dataset.extensionWorkspace = '';
    const frame = document.createElement('iframe');
    frame.dataset.extensionWorkspaceFrame = '';
    f.root.before(host);
    host.append(frame, f.root);
    window.mountWorkspacePanel(f.root, {
      extensionId: 'soda',
      panelId: 'workspace',
      appSubUrl: '',
      apiBase: location.origin + '/-/native/panels/soda/workspace/api/',
      assetBase: location.origin + '/assets/',
      sessionGeneration: f.generation,
    });
    host.addEventListener('soda-spaces-opening', () => {
      host.dataset.panelAtPageStart = String(f.root.querySelectorAll('soda-spaces').length);
    });
  });
  // Wait for the actual native page identity before restoring any owner.
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  assert.equal(await page.locator('soda-spaces').count(), 0, 'full Spaces must have only its page workspace owner');
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1, 'panel must not attach');

  await page.locator('iframe').evaluate((frame: HTMLIFrameElement) => {
    frame.src = '/alice/Alpha';
  });
  await page.locator('soda-spaces .is-connected').waitFor();
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.sockets.length),
    2,
    'repository browsing restores panel'
  );
  const terminal = await page.locator('.xterm').elementHandle();
  await page.locator('iframe').evaluate((frame: HTMLIFrameElement) => {
    frame.src = '/alice/Alpha/issues';
  });
  await page.waitForFunction(
    () => document.querySelector('iframe')?.contentWindow?.location.pathname === '/alice/Alpha/issues'
  );
  assert(await terminal?.evaluate((node) => node.isConnected), 'ordinary browsing must preserve terminal renderer');
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 2);

  await page.locator('iframe').evaluate((frame: HTMLIFrameElement) => {
    frame.src = '/alice/Beta/workspace';
  });
  await page.waitForFunction(() => !document.querySelector('soda-spaces'));
  await page.frameLocator('iframe').locator('soda-spaces .is-connected').waitFor({state: 'attached'});
  assert.equal(
    await page.locator('[data-extension-workspace]').getAttribute('data-panel-at-page-start'),
    '0',
    'native page entry must retire the panel before mounting its own workspace'
  );
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((socket) => socket.closed)), [1, 1]);
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length),
    0
  );
});

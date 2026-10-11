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
    const aside = document.createElement('aside');
    aside.className = 'extension-workspace-panels';
    const section = document.createElement('section');
    section.className = 'extension-workspace-panel';
    host.append(frame, aside);
    aside.append(section);
    section.append(f.root);
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
  assert.equal(
    await page.locator('.extension-workspace-panels').evaluate((node) => node instanceof HTMLElement && node.hidden),
    true
  );

  await page.locator('iframe').evaluate((frame: HTMLIFrameElement) => {
    frame.src = '/alice/Alpha';
  });
  await page.locator('soda-spaces .is-connected').waitFor();
  assert.equal(
    await page.locator('.extension-workspace-panels').evaluate((node) => node instanceof HTMLElement && node.hidden),
    false
  );
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

  await page.evaluate(() => {
    const host = document.querySelector<HTMLElement>('[data-extension-workspace]');
    const aside = host?.querySelector('.extension-workspace-panels');
    if (!host || !aside) throw Error('Missing native panel host');
    const other = document.createElement('section');
    other.className = 'extension-workspace-panel';
    other.id = 'other-product-panel';
    other.textContent = 'Another product';
    aside.append(other);
    host.dispatchEvent(new Event('soda-spaces-opening'));
  });
  assert.equal(
    await page.locator('.extension-workspace-panels').evaluate((node) => node instanceof HTMLElement && node.hidden),
    false
  );
  assert.equal(
    await page
      .locator('.extension-workspace-panel')
      .first()
      .evaluate((node) => node instanceof HTMLElement && node.hidden),
    true,
    'only the retired Soda section is concealed when another panel remains'
  );
  await page.locator('#other-product-panel').evaluate((node) => node.remove());

  await page.locator('iframe').evaluate((frame: HTMLIFrameElement) => {
    frame.src = '/alice/Beta/workspace';
  });
  await page.waitForFunction(() => !document.querySelector('soda-spaces'));
  await page.frameLocator('iframe').locator('soda-spaces .is-connected').waitFor({state: 'attached'});
  await page.waitForFunction(() => document.querySelector<HTMLElement>('.extension-workspace-panels')?.hidden);
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

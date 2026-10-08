import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, action, create} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('direct New reserves one default-named locator before creating; Rename is separate', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await create(page, 'Named build');
  const result = await page.evaluate(() => ({
    frames: window.workspaceFixture.sockets.flatMap((s) => s.sent.filter((f) => f.action === 'create')),
    saved: sessionStorage.getItem('soda-spaces:v3:1'),
    legacy: sessionStorage.getItem('soda-terminal:1:p' + '1'.repeat(24)),
  }));
  assert.equal(result.frames.length, 1);
  assert.equal(result.frames[0]?.name, 'Terminal 3');
  assert.match(String(result.frames[0]?.id), /^[a-f0-9]{32}$/);
  assert(!('request_id' in (result.frames[0] || {})));
  assert.equal(result.legacy, null);
  assert(result.saved);
  assert(!result.saved.includes('synthetic-only'));
});
test('End confirmation defaults Cancel and unknown cleanup preserves exact locator', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await action(page, 'End terminal…');
  assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Cancel');
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.body?.action === 'end').length),
    0
  );
  assert.match(await page.getByRole('dialog', {name: 'End terminal confirmation'}).innerText(), /Build.*alice\/Alpha/s);
  await page.keyboard.press('Escape');
  assert.equal(await page.getByRole('dialog').count(), 0);
  await page.evaluate(() => window.workspaceFixture.setUnknownEnd());
  await action(page, 'End terminal…');
  await page.getByRole('button', {name: 'End terminal', exact: true}).click();
  await page.getByText('End was not confirmed.', {exact: false}).waitFor();
  assert.match(
    await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''),
    /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/
  );
});
test('changed actor invalidates siblings and clears private navigation observations', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await page.evaluate(async () => {
    window.workspaceFixture.setUser('2');
    await window.workspaceFixture.api.refresh();
  });
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 1);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 0);
  assert.equal(await page.locator('.soda-session-list button').count(), 0);
});
test('a terminal admission actor mismatch invalidates every sibling without creating a replacement', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await openSession(page, 'Edit');
  await page.evaluate(() => window.workspaceFixture.setUser('2'));
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.getByRole('button', {name: 'Reload Spaces', exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((socket) => socket.closed)), [1, 1]);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.sockets.flatMap((socket) => socket.sent).filter((frame) => frame.action === 'create')
          .length
    ),
    0
  );
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
});
test('Hide preserves renderer/socket and showing through the shared navigation does not Return', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const screen = await page.locator('.xterm').elementHandle();
  await action(page, 'Hide terminal');
  assert.equal(await page.locator('.xterm').count(), 1);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  await openSession(page, 'Build');
  assert(await screen?.evaluate((el) => el.isConnected));
  assert.deepEqual(
    await page.evaluate(() => ({
      sockets: window.workspaceFixture.sockets.length,
      closed: window.workspaceFixture.sockets[0]?.closed,
      returns: window.workspaceFixture.calls.filter((c) => c.body?.action === 'return').length,
    })),
    {sockets: 1, closed: 0, returns: 0}
  );
});
test('project drafts and independent terminal owners survive detail switching', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await action(page, 'Project settings');
  await page.locator('soda-project-controls [aria-busy=false]').waitFor();
  await page.getByRole('tab', {name: 'Access', exact: true}).click();
  await page.locator('soda-project-controls textarea').fill('unsent public key draft');
  await page.locator('.soda-project-select').filter({hasText: 'alice/Beta'}).click();
  await page.getByRole('button', {name: 'Project settings', exact: true}).click();
  await page.locator('soda-project-controls:visible [aria-busy=false]').waitFor();
  await page.locator('.soda-project-select').filter({hasText: 'alice/Alpha'}).click();
  await page.getByRole('button', {name: 'Project settings', exact: true}).click();
  assert.equal(await page.locator('soda-project-controls:visible textarea').inputValue(), 'unsent public key draft');
  assert.equal(await page.locator('.xterm').count(), 1);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
});
test('pending rename keeps original ID while another project is deliberately selected', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await action(page, 'Rename terminal');
  await page.getByLabel('Session name', {exact: true}).fill('Original project build');
  await page.evaluate(() =>
    window.workspaceFixture.pause(
      new Promise<void>((resolve) => {
        window.setTimeout(resolve, 600);
      })
    )
  );
  await page.getByRole('button', {name: 'Save name', exact: true}).click();
  await page.locator('.soda-project-select').filter({hasText: 'alice/Beta'}).click();
  await page.waitForFunction(() => window.workspaceFixture.calls.some((c) => c.body?.action === 'rename'));
  const calls = await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.body?.action === 'rename'));
  assert.equal(calls.length, 1);
  assert(calls[0]?.path.endsWith('/terminal-sessions/' + 'a'.repeat(32)));
  assert.equal(calls[0]?.body?.name, 'Original project build');
});
test('search and This page change navigation only; New defaults to selected original project', async (t) => {
  const page = await fixture(t, 'native');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Other project');
  const calls = await page.evaluate(() => window.workspaceFixture.calls.length);
  await page.getByRole('button', {name: 'Sessions', exact: true}).click();
  await page.getByLabel('This page only').check();
  assert.equal(await page.locator('.soda-session-list button').count(), 2);
  await page.getByLabel('Find a terminal').fill('missing');
  await page.getByText('No matches.', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back to terminal', exact: true}).click();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), calls);
  assert.equal(await page.locator('.xterm').count(), 1);
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  assert.equal(
    await page.getByRole('dialog', {name: 'New terminal'}).getByLabel('Project', {exact: true}).inputValue(),
    'p' + '2'.repeat(24)
  );
  await page.getByLabel('Terminal name', {exact: true}).fill('a\u200bb');
  assert(await page.getByRole('button', {name: 'Create terminal', exact: true}).isDisabled());
  await page.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), calls);
});
test('Factory view binds one read-only run while refresh and hide retain its renderer', async (t) => {
  const page = await fixture(t),
    runId = 'd'.repeat(32);
  await page.evaluate(async () => {
    window.workspaceFixture.enableFactoryRun();
    await window.workspaceFixture.api.refresh();
  });
  await openSession(page, 'Build');
  await page.getByRole('button', {name: 'Projects', exact: true}).click();

  const watch = (watching: boolean) =>
    page.getByRole('button', {
      name: new RegExp(`^${watching ? 'Watching' : 'Watch'} · soda-coder · issue #42 · active$`),
    });
  await watch(false).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  await page.waitForFunction(() =>
    document.querySelector('.soda-factory .xterm-rows')?.textContent?.includes('Fixture factory output')
  );
  assert.match(await page.locator('.soda-factory [role="status"]').innerText(), /Watching soda-coder · issue #42/);
  assert.equal(await page.locator('.soda-factory .soda-terminal-context span[title="running · live"]').count(), 1);
  const screen = await page.locator('.soda-factory .xterm').elementHandle();

  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert(await screen?.evaluate((element) => element.isConnected));
  await page.locator('.soda-factory summary[aria-label="Factory view actions"]').click();
  await page.locator('.soda-factory').getByRole('button', {name: 'Hide view', exact: true}).click();
  assert(await screen?.evaluate((element) => element.isConnected));
  assert.equal(await page.locator('.soda-factory-box[hidden] .xterm').count(), 1);

  const original = await page.evaluate(() => ({
    sockets: window.workspaceFixture.sockets.map((socket) => ({
      url: String(socket.url),
      closed: socket.closed,
      sent: socket.sent,
    })),
    calls: window.workspaceFixture.calls.map((call) => ({path: call.path, method: call.method})),
  }));
  const firstFactory = original.sockets.findIndex((socket) => socket.url.endsWith(`/factory/runs/${runId}/output`));
  assert(firstFactory >= 0);
  assert.equal(original.sockets[firstFactory]?.closed, 0);
  assert.deepEqual(original.sockets[firstFactory]?.sent, [
    {run_id: runId, repository_id: '7', session_generation: 'fixture-generation', cursor: 0},
  ]);
  assert.deepEqual(
    original.calls.filter((call) => call.method !== 'GET'),
    []
  );
  assert(original.calls.some((call) => call.path.endsWith(`/api/factory/runs/${runId}`)));

  await watch(true).click();
  await page.waitForFunction((index) => window.workspaceFixture.sockets[index]?.closed === 1, firstFactory);
  assert.equal(await page.evaluate((index) => window.workspaceFixture.sockets[index]?.closed, firstFactory), 1);
  await watch(false).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  await page.waitForFunction(() =>
    document.querySelector('.soda-factory .xterm-rows')?.textContent?.includes('Fixture factory output')
  );
  assert.equal(await page.locator('.soda-factory-box:not([hidden])').count(), 1);
  const factorySockets = await page.evaluate((id) => {
    const indices = window.workspaceFixture.sockets.flatMap((socket, index) =>
      String(socket.url).endsWith(`/factory/runs/${id}/output`) ? [index] : []
    );
    return {
      indices,
      current: indices.at(-1) ?? -1,
    };
  }, runId);
  const factoryIndex = factorySockets.current;
  assert.equal(factorySockets.indices.length, 2);
  assert(factoryIndex >= 0);
  assert.equal(await page.evaluate((index) => window.workspaceFixture.sockets[index]?.closed, factoryIndex), 0);
  const terminalScreen = page.locator('.soda-factory .xterm');
  await terminalScreen.click();
  await page.keyboard.type('ignored keyboard input');
  const helper = page.locator('.soda-factory .xterm-helper-textarea');
  assert.equal(await helper.count(), 1);
  await helper.evaluate((element) => {
    const clipboard = new DataTransfer();
    clipboard.setData('text/plain', 'ignored pasted input');
    element.dispatchEvent(new ClipboardEvent('paste', {clipboardData: clipboard, bubbles: true, cancelable: true}));
  });
  await page.setViewportSize({width: 1280, height: 900});
  await page.evaluate(
    () => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
  );
  const stream = await page.evaluate((index) => {
    const socket = window.workspaceFixture.sockets[index];
    return {
      url: String(socket?.url),
      sent: socket?.sent,
      received: socket?.received,
      methods: window.workspaceFixture.calls.map((call) => call.method),
    };
  }, factoryIndex);
  assert(new URL(stream.url).pathname.endsWith(`/api/factory/runs/${runId}/output`));
  assert.deepEqual(stream.sent, [
    {run_id: runId, repository_id: '7', session_generation: 'fixture-generation', cursor: 0},
  ]);
  assert.equal(stream.received?.[0]?.run_id, runId);
  assert.deepEqual(
    {type: stream.received?.[1]?.type, cursor: stream.received?.[1]?.cursor, next: stream.received?.[1]?.next},
    {type: 'output', cursor: 0, next: new TextEncoder().encode('Fixture factory output\r\n').length}
  );
  assert(stream.methods?.every((method) => method === 'GET'));

  await page.locator('.soda-factory-box:not([hidden])').getByRole('button', {name: 'Close view', exact: true}).click();
  await page.waitForFunction((index) => window.workspaceFixture.sockets[index]?.closed === 1, factoryIndex);
  await page.evaluate((index) => {
    const socket = window.workspaceFixture.sockets[index],
      status = socket?.received.find((frame) => frame.type === 'status');
    if (socket && status) socket.receive(status);
  }, factoryIndex);
  assert.equal(await page.locator('.soda-factory-box:not([hidden])').count(), 0);
  assert.equal(await watch(false).count(), 1);
  const afterClose = await page.evaluate(() => ({
    closed: window.workspaceFixture.sockets.map((socket) => socket.closed),
    methods: window.workspaceFixture.calls.map((call) => call.method),
  }));
  assert.equal(afterClose.closed.length, 3);
  assert.equal(afterClose.closed[0], 0, 'closing a Factory view leaves the terminal socket open');
  assert.deepEqual(afterClose.closed.slice(1), [1, 1]);
  assert.equal(afterClose.closed[factoryIndex], 1);
  assert(afterClose.methods.every((method) => method === 'GET'));
});

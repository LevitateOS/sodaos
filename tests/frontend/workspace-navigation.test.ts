import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import type {Page} from 'playwright';
import {factoryAssetSource} from './fixtures/factory-asset-source';
import {
  setupWorkspaceDriver,
  fixture,
  openSession,
  action,
  create,
  chooseFirstRepository,
  enterCreatedProjectTerminals,
} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('repository context selects by exact native ID despite colliding display names', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    window.workspaceFixture.api.dispose();
    const f = window.createWorkspaceFixture('page', false, '', '8');
    window.workspaceFixture = f;
    for (const space of f.spaces) space.environment.repository = 'shared/display-name';
    await f.api.refresh();
  });
  const selected = page.locator('.soda-project-select[aria-pressed="true"]');
  await selected.waitFor();
  assert.equal(await selected.locator('..').getAttribute('data-environment-id'), 'p' + '2'.repeat(24));
  assert.equal(await selected.count(), 1);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.path.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
});

test('repository Workspace selects its native observed Project without commands', async (t) => {
  const page = await fixture(t);
  await page.goto(new URL('/alice/Beta/workspace', page.url()).href);
  await page.locator('.soda-project-select[aria-pressed="true"]').waitFor();
  assert.equal(
    await page.locator('.soda-project-select[aria-pressed="true"]').getAttribute('aria-label'),
    'alice/Beta'
  );
  assert.equal(await page.locator('.soda-selected-project h1').innerText(), 'alice/Beta');
  assert.equal(
    await page.evaluate(
      () =>
        window.workspacePageModel.calls.filter(
          (call) => call.method !== 'GET' && !call.path.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
});

test('repository Workspace with no Project opens only its observed configuration without first-Project fallback', async (t) => {
  const page = await fixture(t);
  await page.goto(new URL('/alice/Missing/workspace', page.url()).href);
  await page.getByRole('heading', {name: 'Configure project', exact: true}).waitFor();
  assert.equal(await page.locator('.soda-project-select[aria-pressed="true"]').count(), 0);
  assert.equal(await page.locator('[data-project-controls]').getAttribute('data-repository-id'), '9');
  assert.equal(
    await page.locator('[data-project-controls]').getByRole('button', {name: 'Create project', exact: true}).count(),
    1
  );
  assert.equal(
    await page.evaluate(
      () =>
        window.workspacePageModel.calls.filter(
          (call) => call.method !== 'GET' && !call.path.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
});

for (const firstUse of [false, true])
  test(`Project settings mount Factory controls through the workspace${firstUse ? ' after first use' : ''}`, async (t) => {
    const page = await fixture(t, 'page', undefined, firstUse);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    if (firstUse) {
      await chooseFirstRepository(page);
      assert.equal(
        await page.locator('soda-factory-settings').count(),
        0,
        'setup must keep Factory out of the journey'
      );
      await page.getByRole('button', {name: 'Create project', exact: true}).click();
      await enterCreatedProjectTerminals(page);
      await page.getByRole('button', {name: 'Join project', exact: true}).click();
      await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
    }
    const existingOwner = firstUse ? await page.locator('soda-project-controls').elementHandle() : undefined;
    const writesBeforeSettings = await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.path.endsWith('/api/factory/native-repositories/resolve')
        ).length
    );
    await page.getByRole('button', {name: 'Project settings', exact: true}).click();
    await page.getByRole('tab', {name: 'Factory', exact: true}).click();
    assert.equal(await page.locator('[data-view="factory"]').getAttribute('aria-selected'), 'true');
    const settings = page.locator('soda-factory-settings');
    await settings.waitFor({state: 'visible'});
    assert.equal(await settings.count(), 1);
    await settings.locator('form[data-action="policy"]').waitFor({state: 'visible'});
    const discovery = await page.evaluate(() =>
      window.workspaceFixture.calls.filter((call) => call.path.endsWith('/api/factory/native-repositories/resolve'))
    );
    assert(discovery.length > 0);
    assert.equal(
      discovery.at(-1)?.body?.repository,
      await page.locator('[data-project-controls]').getAttribute('data-repository-id')
    );
    assert(await settings.getByRole('button', {name: 'Save repository policy', exact: true}).isVisible());
    if (firstUse)
      assert(await existingOwner?.evaluate((node) => node.isConnected), 'settings must reuse the journey owner');
    assert.equal(
      await page.evaluate(
        () =>
          window.workspaceFixture.calls.filter(
            (call) => call.method !== 'GET' && !call.path.endsWith('/api/factory/native-repositories/resolve')
          ).length
      ),
      writesBeforeSettings,
      'opening settings must not submit a command'
    );
  });

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
  await page.getByRole('textbox', {name: 'Public SSH key', exact: true}).fill('unsent public key draft');
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
  await page.getByRole('button', {name: 'Project settings', exact: true}).click();
  await page.locator('soda-project-controls:visible [aria-busy=false]').waitFor();
  await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
  await page.getByRole('button', {name: 'Project settings', exact: true}).click();
  assert.equal(
    await page.getByRole('textbox', {name: 'Public SSH key', exact: true}).inputValue(),
    'unsent public key draft'
  );
  assert.equal(await page.locator('.xterm').count(), 1);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 0);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter(
          (c) => c.method !== 'GET' && !c.path.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
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
  await page.getByRole('button', {name: 'alice/Beta', exact: true}).click();
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
test('Factory watch resolves canonical run metadata and retains the native Project stream binding', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    window.workspaceFixture.setFactoryIdentity('7', '41', '11');
    window.workspaceFixture.enableFactoryRun();
    await window.workspaceFixture.api.refresh();
  });
  const work = page.getByRole('button', {name: 'Work', exact: true});
  if (await work.isVisible()) await work.click();
  await page.getByRole('button', {name: /^Watch · Coding agent · issue #42 · active$/}).click();
  await page.locator('.soda-factory.is-connected .xterm-rows').waitFor();
  await page.waitForFunction(() =>
    document.querySelector('.soda-factory .xterm-rows')?.textContent?.includes('Fixture factory output')
  );
  const evidence = await page.evaluate(() => ({
    calls: window.workspaceFixture.calls,
    native: window.workspaceFixture.spaces[0]?.environment,
    peer: window.workspaceFixture.sockets.find((socket) =>
      String(socket.url).endsWith('/factory/runs/' + 'd'.repeat(32) + '/output')
    )?.sent,
  }));
  assert.equal(evidence.native?.repository_id, '7');
  assert.equal(evidence.native?.owner_id, '1');
  assert.deepEqual(evidence.peer, [
    {run_id: 'd'.repeat(32), repository_id: '7', session_generation: 'fixture-generation', cursor: 0},
  ]);
  const runRead = evidence.calls.findIndex((call) => call.path.endsWith('/api/factory/runs/' + 'd'.repeat(32)));
  assert(runRead > 0);
  assert(
    evidence.calls
      .slice(0, runRead)
      .some((call) => call.path.endsWith('/factory/native-repositories/resolve') && call.body?.repository === '7')
  );
  assert.equal(
    evidence.calls.filter(
      (call) => call.method !== 'GET' && !call.path.endsWith('/factory/native-repositories/resolve')
    ).length,
    0
  );
});

test('Factory watch refuses native run metadata or unavailable discovery before opening an output stream', async (t) => {
  for (const denied of [false, true]) {
    const page = await fixture(t);
    await page.evaluate(async () => {
      window.workspaceFixture.setFactoryIdentity('7', '41', '11');
      window.workspaceFixture.enableFactoryRun();
      await window.workspaceFixture.api.refresh();
    });
    const work = page.getByRole('button', {name: 'Work', exact: true});
    if (await work.isVisible()) await work.click();
    await page.evaluate((denied) => {
      if (denied) window.workspaceFixture.denyFactoryDiscovery();
      else window.workspaceFixture.setFactoryRunRepository('7');
    }, denied);
    await page.getByRole('button', {name: /^Watch · Coding agent · issue #42 · active$/}).click();
    await page
      .locator('.soda-factory')
      .getByText('Could not authorize or attach. Sign in again or inspect the exact run; nothing was replayed.', {
        exact: true,
      })
      .waitFor();
    const evidence = await page.evaluate(() => ({
      reads: window.workspaceFixture.calls.filter((call) => call.path.endsWith('/api/factory/runs/' + 'd'.repeat(32)))
        .length,
      peers: window.workspaceFixture.sockets.filter((socket) => String(socket.url).includes('/factory/runs/')).length,
    }));
    assert.equal(evidence.peers, 0);
    assert.equal(evidence.reads, denied ? 0 : 1);
  }
});

test('closing a Factory watch during discovery prevents a late run read or stream', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    window.workspaceFixture.setFactoryIdentity('7', '41', '11');
    window.workspaceFixture.enableFactoryRun();
    await window.workspaceFixture.api.refresh();
  });
  const work = page.getByRole('button', {name: 'Work', exact: true});
  if (await work.isVisible()) await work.click();
  const before = await page.evaluate(() => {
    window.workspaceFixture.holdFactoryDiscovery();
    return window.workspaceFixture.calls.filter((call) => call.path.endsWith('/factory/native-repositories/resolve'))
      .length;
  });
  await page.getByRole('button', {name: /^Watch · Coding agent · issue #42 · active$/}).click();
  await page.waitForFunction(
    (before) =>
      window.workspaceFixture.calls.filter((call) => call.path.endsWith('/factory/native-repositories/resolve'))
        .length > before,
    before
  );
  await page.getByRole('button', {name: /^Watching · Coding agent · issue #42 · active$/}).click();
  await page.evaluate(async () => {
    window.workspaceFixture.releaseFactoryDiscovery();
    await new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
  });
  assert.equal(await page.locator('.soda-factory').count(), 0);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.calls.filter((call) => call.path.endsWith('/api/factory/runs/' + 'd'.repeat(32))).length
    ),
    0
  );
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.sockets.filter((socket) => String(socket.url).includes('/factory/runs/')).length
    ),
    0
  );
});

test('Factory view binds one read-only run while refresh and hide retain its renderer', async (t) => {
  const page = await fixture(t),
    runId = 'd'.repeat(32);
  await page.evaluate(async () => {
    window.workspaceFixture.enableFactoryRun();
    await window.workspaceFixture.api.refresh();
  });
  await openSession(page, 'Build');
  await page.getByRole('button', {name: 'Work', exact: true}).click();

  const watch = (watching: boolean) =>
    page.getByRole('button', {
      name: new RegExp(`^${watching ? 'Watching' : 'Watch'} · Coding agent · issue #42 · active$`),
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
    original.calls.filter(
      (call) => call.method !== 'GET' && !call.path.endsWith('/api/factory/native-repositories/resolve')
    ),
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
      methods: window.workspaceFixture.calls
        .filter((call) => !call.path.endsWith('/api/factory/native-repositories/resolve'))
        .map((call) => call.method),
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
    methods: window.workspaceFixture.calls
      .filter((call) => !call.path.endsWith('/api/factory/native-repositories/resolve'))
      .map((call) => call.method),
  }));
  assert.equal(afterClose.closed.length, 3);
  assert.equal(afterClose.closed[0], 0, 'closing a Factory view leaves the terminal socket open');
  assert.deepEqual(afterClose.closed.slice(1), [1, 1]);
  assert.equal(afterClose.closed[factoryIndex], 1);
  assert(afterClose.methods.every((method) => method === 'GET'));
  await standaloneWatch(page);
});

async function standaloneWatch(parent: Page) {
  const root = path.resolve(import.meta.dirname, '../..');
  const build = await Bun.build({entrypoints: [path.join(root, 'tests/frontend/fixtures/factory-watch-fixture.ts')], target: 'browser', format: 'esm', define: {'process.env.NODE_ENV': '"production"'}});
  assert(build.success, `Standalone Watch build failed: ${build.logs.join('\n')}`);
  const script = build.outputs[0]; assert(script);
  const host = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/watch.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
    if (pathname === '/xterm.css') return new Response(Bun.file(path.join(root, 'node_modules/@xterm/xterm/css/xterm.css')), {headers: {'Content-Type': 'text/css'}});
    const asset = factoryAssetSource(pathname); if (asset) return asset;
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><link rel="stylesheet" href="/xterm.css"><main class="soda-factory-root" data-factory-scheme="light"></main><script type="module" src="/watch.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  const browser = parent.context().browser(); assert(browser);
  const page = await browser.newPage({viewport: {width: 1100, height: 800}}); page.setDefaultTimeout(7000);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => { if (response.url().startsWith(host.url.origin) && response.status() >= 400) errors.push(`Standalone asset failed (${response.status()}): ${response.url()}`); });
  try {
    await page.goto(host.url.href);
    const view = page.locator('soda-factory-watch');
    const ready = () => page.waitForFunction(() => document.querySelector('.soda-factory .xterm-rows')?.textContent?.includes('Declared live output.'));
    await view.getByRole('button', {name: 'Watch run', exact: true}).click(); await ready();
    assert.deepEqual(await page.evaluate(() => window.factoryWatchFixture.peers()[0]?.sent), [{observe: 'a'.repeat(32), after: 0, capability: 'declared-watch:idle'}]);
    const terminal = await view.locator('.xterm').elementHandle(); assert(terminal);
    await view.locator('.soda-terminal-screen').click(); await page.keyboard.type('must not send input');
    await page.setViewportSize({width: 390, height: 740});
    assert.equal(await page.evaluate(() => window.factoryWatchFixture.peers()[0]?.sent.length), 1);
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(255, 255, 255)');
    await page.locator('main').evaluate(node => node.setAttribute('data-factory-scheme', 'dark'));
    assert.equal(await page.locator('main').evaluate(node => getComputedStyle(node).backgroundColor), 'rgb(16, 16, 16)');
    await page.evaluate(() => { window.factoryWatchFixture.suspend(); window.factoryWatchFixture.resume(); });
    assert.equal(await page.evaluate(() => window.factoryWatchFixture.peers().length), 1);
    assert.equal(await page.evaluate(() => window.factoryWatchFixture.reads().length), 1);
    assert(await terminal.evaluate(node => node.isConnected));
    await view.getByRole('button', {name: 'Stop watching', exact: true}).click();
    await page.evaluate(() => window.factoryWatchFixture.late());
    assert.equal(await page.evaluate(() => window.factoryWatchFixture.peers()[0]?.state), 3);
    assert.doesNotMatch(await view.innerText(), /Retired output must not render/);
    await view.getByRole('button', {name: 'Watch again', exact: true}).click();
    await page.waitForFunction(() => document.querySelector('.soda-factory .xterm-rows')?.textContent?.includes('Resumed output.'));
    const retained = await view.locator('.xterm').elementHandle(); assert(retained);
    await page.evaluate(() => window.factoryWatchFixture.lose());
    await page.waitForFunction(() => window.factoryWatchFixture.peers().length === 3 && window.factoryWatchFixture.peers()[2]?.sent.length === 1);
    const peers = await page.evaluate(() => window.factoryWatchFixture.peers());
    assert.equal(peers[2]?.sent[0]?.after, new TextEncoder().encode('Resumed output.\r\n').length);
    assert(await retained.evaluate(node => node.isConnected && node === document.querySelector('.soda-factory .xterm')));
    await view.getByRole('button', {name: 'Stop watching', exact: true}).waitFor();
    await view.getByRole('button', {name: 'Stop watching', exact: true}).click();
    for (const state of ['ended', 'offline', 'denied', 'invalid'] as const) {
      await page.evaluate(state => window.factoryWatchFixture.mount(state), state);
      await view.getByRole('button', {name: 'Watch run', exact: true}).click();
      if (state === 'ended') {
        await view.getByRole('status').filter({hasText: 'This run has finished'}).waitFor();
        await view.getByText('Technical details', {exact: true}).click();
        await view.getByText('Recorded final output.', {exact: false}).waitFor();
        assert.equal(await page.evaluate(() => window.factoryWatchFixture.peers().length), 0);
      } else if (state === 'offline') await view.getByRole('status').filter({hasText: 'Could not authorize or attach.'}).waitFor();
      else if (state === 'denied') {
        await view.getByRole('status').filter({hasText: 'Page context changed.'}).waitFor();
        assert.equal(await page.evaluate(() => window.factoryWatchFixture.authorityLosses()), 1);
        assert.equal(await view.getByRole('button', {name: 'Watch again', exact: true}).count(), 0);
      } else await view.getByRole('status').filter({hasText: 'Invalid or overloaded stream.'}).waitFor();
    }
    await page.evaluate(() => window.factoryWatchFixture.mount('live'));
    await view.getByRole('button', {name: 'Watch run', exact: true}).click(); await ready();
    await page.evaluate(() => { window.factoryWatchFixture.retireGeneration(); window.factoryWatchFixture.late(); });
    await view.getByRole('status').filter({hasText: 'Page context changed.'}).waitFor();
    assert.equal(await view.locator('.xterm').count(), 0);
    assert.equal(await page.evaluate(() => window.factoryWatchFixture.peers()[0]?.state), 3);
    for (const mode of ['generation', 'retire'] as const) {
      await page.evaluate(() => window.factoryWatchFixture.mount('live', true));
      await view.getByRole('button', {name: 'Watch run', exact: true}).click();
      await page.waitForFunction(() => window.factoryWatchFixture.reads().length === 1);
      await page.evaluate(mode => {
        if (mode === 'generation') window.factoryWatchFixture.retireGeneration(); else window.factoryWatchFixture.retire();
        window.factoryWatchFixture.release();
      }, mode);
      await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => resolve())));
      assert.equal(await page.evaluate(() => window.factoryWatchFixture.peers().length), 0);
      assert.equal(await page.evaluate(() => window.factoryWatchFixture.aborted()), true);
    }
    await page.evaluate(() => document.fonts.ready.then(() => undefined)); assert.deepEqual(errors, []);
  } finally { await page.close(); host.stop(true); }
}

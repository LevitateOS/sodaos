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

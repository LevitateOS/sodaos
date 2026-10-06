import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, textContents} from './fixtures/workspace-driver';
setupWorkspaceDriver();

test('observed unread coalesces noisy hidden output, survives refresh, and clears only deliberate viewing', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await openSession(page, 'Edit');
  const screen = await page.locator('.xterm').first().elementHandle();
  await page.locator('#native-draft').focus();
  await page.evaluate(() => {
    const sockets = window.workspaceFixture.sockets;
    for (let i = 0; i < 100; i++)
      sockets[0]?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('log\r\n')})});
    sockets[1]?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('visible\r\n')})});
  });
  const tab = page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true});
  await tab.locator('.soda-unread').waitFor();
  assert.equal(
    await page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}).locator('.soda-unread').count(),
    0
  );
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await tab.locator('.soda-unread').count(), 1);
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  assert.equal(await page.locator('.soda-attention-filters:visible').count(), 0);
  await openSession(page, 'Build');
  await page.waitForFunction(() => !document.querySelector('[aria-selected=true] .soda-unread'));
  assert(await screen?.evaluate((node) => node.isConnected));
  await page.evaluate(() => {
    window.workspaceFixture.api.setVisible(false);
    window.workspaceFixture.sockets[0]?.onmessage?.({
      data: JSON.stringify({type: 'output', data: btoa('behind Forge')}),
    });
    window.workspaceFixture.api.setVisible(true);
  });
  await tab.locator('.soda-unread').waitFor();
  await page.evaluate(() => window.workspaceFixture.api.markViewed());
  await page.waitForFunction(() => !document.querySelector('[aria-selected=true] .soda-unread'));
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
});
test('attention has stable authorized counts/order and exact navigation without creation or lifetime actions', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const f = window.workspaceFixture,
      a = f.spaces[0]?.terminals[0],
      b = f.spaces[0]?.terminals[1],
      c = f.spaces[1]?.terminals[0];
    if (!a || !b || !c) throw Error('missing metadata');
    a.attached = true;
    b.state = 'ending';
    b.ready = false;
    c.state = 'ending';
    c.ready = false;
    return f.api.refresh();
  });
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.getByRole('button', {name: 'Attention (3)', exact: true}).click();
  assert.deepEqual(await textContents(page.locator('.soda-session-list > button > span')), [
    'Build',
    'Edit',
    'Other project',
  ]);
  await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}).waitFor();
  await page.getByText('Native transition is still in progress.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.evaluate(() => {
    const b = window.workspaceFixture.spaces[1];
    if (b) {
      b.authority_unavailable = true;
      b.environment_administrator = false;
      b.terminals = [];
      b.login = '';
    }
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: 'Attention (2)', exact: true}).waitFor();
  await page.evaluate(() => {
    window.workspaceFixture.setUser('2');
    return window.workspaceFixture.api.refresh();
  });
  assert.equal(await page.locator('.soda-session-list button').count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
  assert.equal(
    await page.evaluate(
      () => window.workspaceFixture.sockets.flatMap((s) => s.sent).filter((f) => f.action === 'create').length
    ),
    0
  );
});
test('stale metadata and late transport generations cannot claim cleanup or replay unread', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const peer = await page.evaluateHandle(() => window.workspaceFixture.sockets[0]);
  await page.evaluate(() =>
    window.workspaceFixture.sockets[0]?.onmessage?.({data: JSON.stringify({type: 'closed', reason: 'unavailable'})})
  );
  await page.getByText('Attachment ended or unavailable.', {exact: false}).waitFor();
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await peer.evaluate((socket) => socket?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('late')})}));
  assert.equal(await page.locator('.soda-unread').count(), 0);
  await page.getByRole('button', {name: 'Attention (1)', exact: true}).waitFor();
  assert.match(
    await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''),
    /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/
  );
});

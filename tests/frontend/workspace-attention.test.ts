import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, textContents} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
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
  assert.equal(await tab.getByRole('img', {name: 'Unread output'}).count(), 1);
  assert.equal(
    await page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}).locator('.soda-unread').count(),
    0
  );
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await tab.locator('.soda-unread').count(), 1);
  await page.getByRole('button', {name: 'Work', exact: true}).click();
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
  const page = await fixture(t, 'native');
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
  await page.getByRole('button', {name: 'Sessions', exact: true}).click();
  await page.getByRole('button', {name: 'Attention (3)', exact: true}).click();
  assert.deepEqual(await textContents(page.locator('.soda-session-list > button > span')), [
    'Build',
    'Edit',
    'Other project',
  ]);
  await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Sessions', exact: true}).click();
  await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}).waitFor();
  await page.getByText('Native transition is still in progress.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.getByRole('button', {name: 'Sessions', exact: true}).click();
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
  await page
    .getByText(
      'The terminal is unavailable. Retry or ask an appliance administrator to inspect it. No creation or input was replayed.',
      {exact: true}
    )
    .waitFor();
  await page.getByRole('button', {name: 'Work', exact: true}).click();
  await peer.evaluate((socket) => socket?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('late')})}));
  assert.equal(await page.locator('.soda-unread').count(), 0);
  const attention = page.getByRole('button', {name: 'alice/Alpha', exact: true}).locator('.soda-project-attention');
  await attention.waitFor();
  assert.equal(await attention.getAttribute('aria-label'), '1 terminal needs attention');
  assert.match(
    await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''),
    /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/
  );
});
test('authorized collection reports an observed other writer without needing a failed attachment', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.workspaceFixture,
      terminal = f.spaces[0]?.terminals[0];
    if (!terminal) throw Error('fixture');
    terminal.attached = true;
    await f.api.refresh();
  });
  const attention = page.getByRole('button', {name: 'alice/Alpha', exact: true}).locator('.soda-project-attention');
  await attention.waitFor();
  assert.equal(await attention.getAttribute('aria-label'), '1 terminal needs attention');
  await page.setViewportSize({width: 390, height: 844});
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  assert.equal(await page.getByRole('tab').count(), 0, 'observed sessions do not become open tabs');
  assert.equal(await page.locator('.soda-project-terminals:visible').count(), 0);
  assert.equal(await page.locator('.soda-workspace-toolbar .soda-project-views').count(), 1);
  await page.getByLabel('Workspace options', {exact: true}).click();
  assert.equal(await page.locator('.soda-project-terminals .soda-session-list button:visible').count(), 2);
  await page.locator('.soda-project-terminals .soda-session-list button').filter({hasText: 'Build'}).click();
  await page.getByText('An existing writer is attached.', {exact: false}).waitFor();
  await page.evaluate(() => document.fonts.ready);
  const geometry = await page.locator('.soda-workspace').evaluate((node) => {
    const header = node.querySelector('.soda-workspace-toolbar')!.getBoundingClientRect();
    const status = node
      .querySelector('.soda-workspace-terminal:not([hidden]) .soda-terminal-status')!
      .getBoundingClientRect();
    return {
      headerHeight: header.height,
      errorOffset: status.top - header.top,
      overflow: node.scrollWidth - node.clientWidth,
    };
  });
  await captureSpacesComponent(page, 'compact-terminal-failure');
  assert(
    geometry.headerHeight <= 144,
    'compact project navigation consumes too much terminal height: ' + JSON.stringify(geometry)
  );
  assert(geometry.errorOffset <= 208, 'terminal failure is buried below repeated navigation');
  assert(geometry.overflow <= 1, 'compact terminal chrome must fit the viewport');
  assert.equal(await page.locator('.soda-project-terminals:visible').count(), 0, 'selection closes the session menu');
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.evaluate(async () => {
    const f = window.workspaceFixture,
      terminal = f.spaces[0]?.terminals[0];
    if (!terminal) throw Error('fixture');
    terminal.attached = false;
    await f.api.refresh();
  });
  assert.equal(await page.locator('.soda-attention-filters:visible').count(), 0);
  assert.equal(await attention.count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.body).length), 0);
});

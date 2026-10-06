import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, paneAction} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
import {parseLayout} from '../../frontend/spaces/sodaspaces-layout';
setupWorkspaceDriver();

test('real xterm owners survive pane moves, keyboard divider, maximize, compact and consolidation without effects', async (t) => {
  const page = await fixture(t);
  await page.setViewportSize({width: 1920, height: 1440});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  for (const name of ['Build', 'Edit', 'Other project']) await openSession(page, name);
  const screens = await page.locator('.xterm').elementHandles(),
    hosts = await page.locator('.soda-workspace-terminal').elementHandles();
  const before = await page.evaluate(() => window.workspaceFixture.calls.length);
  await paneAction(page, 'Split right');
  assert.equal(await page.locator('.soda-pane-chrome').count(), 2);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), before);
  await page.getByLabel('Move terminal to pane', {exact: true}).click();
  await captureSpacesComponent(page, 'move-menu-1920');
  await page.getByRole('button', {name: 'Pane 2', exact: true}).click();
  await page.waitForFunction(() => document.querySelectorAll('.soda-workspace-terminal:not([hidden])').length === 2);
  const panes = await page
    .locator('.soda-workspace-terminal:visible')
    .evaluateAll((nodes) => nodes.map((node) => node.getBoundingClientRect().width));
  assert.equal(panes.length, 2);
  assert(
    panes.every((width) => width >= 800),
    'two terminals should share the available width after the sidebar'
  );
  await captureSpacesComponent(page, 'split-workspace-1920');
  await page.evaluate(() => {
    const f = window.workspaceFixture,
      names = [...document.querySelectorAll('.soda-pane-chrome [aria-selected=true]')].map((tab) =>
        tab.textContent?.trim()
      );
    const visible = f.spaces
      .flatMap((space) => space.terminals)
      .filter((terminal) => names.includes(terminal.name))
      .map((terminal) => terminal.id);
    for (const socket of f.sockets.filter((socket) => visible.includes(String(socket.sent[0]?.id))))
      socket.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('visible pane')})});
  });
  assert.equal(await page.locator('.soda-workspace-tabs .soda-unread').count(), 0);
  await page.waitForFunction(() =>
    window.workspaceFixture.sockets.every((s) => {
      const resize = s.sent.filter((f) => f.type === 'resize').at(-1);
      return Number(resize?.cols) >= 56 && Number(resize?.rows) >= 12;
    })
  );
  await page.getByRole('separator', {name: 'Resize panes', exact: true}).press('ArrowLeft');
  const stored = await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1'));
  await paneAction(page, 'Maximize pane');
  assert.equal(await page.locator('.soda-pane-chrome').count(), 1);
  await paneAction(page, 'Restore panes');
  await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 2);
  await page.setViewportSize({width: 720, height: 900});
  await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 1);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1')), stored);
  await page.getByLabel('Focused pane', {exact: true}).selectOption({label: 'Pane 1'});
  await page.getByRole('group', {name: 'Pane 1', exact: true}).waitFor();
  const compactSelection = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  assert.deepEqual(compactSelection.tree, parseLayout(stored || '').tree);
  assert.equal(compactSelection.sidebar, parseLayout(stored || '').sidebar);
  await page.setViewportSize({width: 1920, height: 1440});
  await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 2);
  await paneAction(page, 'Consolidate panes');
  assert.equal(await page.locator('.soda-pane-chrome').count(), 1);
  for (const handle of [...screens, ...hosts]) assert(await handle.evaluate((el) => el.isConnected));
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((s) => s.closed)), [0, 0, 0]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), before);
  assert.equal(
    await page.evaluate(
      () =>
        window.workspaceFixture.sockets
          .flatMap((s) => s.sent)
          .filter((f) => f.type === 'resize' && (!f.cols || !f.rows)).length
    ),
    0
  );
});
test('sidebar bounds, tab overflow, pointer reorder and edge split preserve owners without IO', async (t) => {
  const page = await fixture(t);
  await page.setViewportSize({width: 1920, height: 1440});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  for (const name of ['Build', 'Edit', 'Other project']) await openSession(page, name);
  const hosts = await page.locator('.soda-workspace-terminal').elementHandles(),
    before = await page.evaluate(() => window.workspaceFixture.calls.length);
  const sidebar = page.getByRole('separator', {name: 'Resize project sidebar', exact: true});
  await sidebar.press('End');
  assert.equal(await sidebar.getAttribute('aria-valuenow'), '360');
  const box = await sidebar.boundingBox();
  assert(box);
  await page.mouse.move(box.x + 3, box.y + 80);
  await page.mouse.down();
  await page.mouse.move(230, box.y + 80);
  await page.mouse.up();
  const layout = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  assert(layout.sidebar !== null && layout.sidebar >= 220 && layout.sidebar < 360);
  await page.getByLabel('Workspace options', {exact: true}).click();
  await page.getByRole('button', {name: 'Toggle sidebar', exact: true}).click();
  assert.equal(parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || '')).sidebar, null);
  await page.keyboard.press('Escape');
  await page.getByLabel('Open tabs', {exact: true}).click();
  await page.getByLabel('Find an open tab', {exact: true}).fill('Build');
  assert.equal(await page.locator('.soda-tab-overflow button:visible').count(), 1);
  await page.locator('.soda-tab-overflow button:visible').click();
  assert.equal(
    await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).getAttribute('aria-selected'),
    'true'
  );
  await page
    .getByRole('tab', {name: 'Other project · alice/Beta', exact: true})
    .dragTo(page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}));
  assert.match((await page.getByRole('tab').allTextContents())[0] || '', /Other project/);
  const edit = page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}),
    origin = await edit.boundingBox();
  assert(origin);
  await page.mouse.move(origin.x + origin.width / 2, origin.y + origin.height / 2);
  await page.mouse.down();
  await page.mouse.move(origin.x + 25, origin.y + 65, {steps: 8});
  const edge = await page.locator('.soda-drop-edge.right:visible').boundingBox();
  assert(edge);
  await page.mouse.move(edge.x + edge.width / 2, edge.y + edge.height / 2, {steps: 8});
  await page.mouse.up();
  await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 2);
  for (const host of hosts) assert(await host.evaluate((node) => node.isConnected));
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), before);
  assert.deepEqual(
    await page.evaluate(() => window.workspaceFixture.sockets.map((socket) => socket.closed)),
    [0, 0, 0]
  );
});

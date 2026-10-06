import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, action} from './fixtures/workspace-driver';
import {projectView, newManagedTerminal, terminalMenu} from '../installed/sodaspaces-controls.ts';
setupWorkspaceDriver();

for (const failure of ['unavailable', 'unsafe-path', 'wrong-repository', 'storage'])
  test(`Open in drawer preserves the terminal when ${failure} prevents a safe handoff`, async (t) => {
    const page = await fixture(t);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await openSession(page, 'Build');
    const screen = await page.locator('.xterm').elementHandle(),
      original = page.url();
    await page.evaluate((failure) => {
      if (failure === 'unavailable') window.workspaceFixture.setStatus(503);
      else if (failure === 'storage') {
        const save = Storage.prototype.setItem;
        Storage.prototype.setItem = function (key, value) {
          if (key.startsWith('soda-spaces:v3:')) throw Error('Synthetic storage failure');
          save.call(this, key, value);
        };
      } else {
        const fetch = window.fetch;
        Object.defineProperty(window, 'fetch', {
          configurable: true,
          value: async (input: RequestInfo | URL, init?: RequestInit) =>
            String(input).includes('/api/environments?')
              ? Response.json({
                  repository: {
                    id: failure === 'wrong-repository' ? '8' : '7',
                    owner: 'alice',
                    name: failure === 'unsafe-path' ? '../elsewhere' : 'Alpha',
                  },
                })
              : fetch(input, init),
        });
      }
    }, failure);
    await page.getByLabel('Workspace options', {exact: true}).click();
    await page.getByRole('button', {name: 'Open in drawer', exact: true}).click();
    await page
      .getByText(
        failure === 'storage'
          ? 'Save the workspace before opening it in the drawer; restoration is unavailable.'
          : 'Could not open the repository drawer. Your terminal remains here; refresh and try again.',
        {exact: true}
      )
      .waitFor();
    assert.equal(page.url(), original);
    assert(await screen?.evaluate((node) => node.isConnected));
    assert.equal(
      await page.evaluate(() => window.workspaceFixture.calls.filter((call) => call.method !== 'GET').length),
      0
    );
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  });
for (const mode of ['native', 'page'] as const)
  for (const theme of ['light', 'dark'] as const)
    for (const width of [1440, 390])
      test(`${mode}/${theme}/${width}: resolved tokens, focus, menu and warning preserve native draft and terminal`, async (t) => {
        const page = await fixture(t, mode);
        await page.setViewportSize({width, height: 900});
        await page.evaluate((theme) => {
          document.documentElement.style.colorScheme = theme;
        }, theme);
        await page.evaluate(() => window.workspaceFixture.api.refresh());
        await openSession(page, 'Build');
        assert(
          await page.locator('.soda-workspace-toolbar').evaluate((node) => node.scrollWidth <= node.clientWidth),
          'workspace controls overflow'
        );
        const screen = await page.locator('.xterm').elementHandle();
        await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).focus();
        await page.keyboard.press('Tab');
        await page.keyboard.press('Shift+Tab');
        const selected = await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).evaluate((node) => {
          const style = getComputedStyle(node),
            probe = document.createElement('span');
          node.append(probe);
          probe.style.color = 'var(--soda-page-focus)';
          const focus = getComputedStyle(probe).color;
          probe.style.color = node.closest('[data-workspace-kind=page]')
            ? 'var(--soda-page-hover)'
            : 'var(--soda-button-primary-bg)';
          const background = getComputedStyle(probe).color;
          probe.remove();
          return {
            background: style.backgroundColor,
            expected: background,
            focus: style.outlineColor,
            expectedFocus: focus,
            height: node.getBoundingClientRect().height,
          };
        });
        assert.equal(selected.background, selected.expected);
        assert.equal(selected.focus, selected.expectedFocus);
        if (mode === 'native') assert.equal(selected.height, 44);
        else {
          const strip = await page.locator('.soda-pane-chrome').boundingBox();
          assert(strip && Math.abs(selected.height - strip.height) <= 1, 'page tab must fill its strip');
        }
        await action(page, 'End terminal…');
        const warning = await page.getByRole('dialog', {name: 'End terminal confirmation'}).evaluate((node) => {
          const style = getComputedStyle(node),
            probe = document.createElement('span');
          probe.style.color = 'var(--soda-page-warning-bg)';
          node.append(probe);
          const expected = getComputedStyle(probe).color;
          probe.remove();
          return {actual: style.backgroundColor, expected};
        });
        assert.equal(warning.actual, warning.expected);
        await page.keyboard.press('Escape');
        assert(await screen?.evaluate((node) => node.isConnected));
        assert.equal(await page.locator('#native-draft').inputValue(), 'unsaved');
        assert.equal(
          await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length),
          0
        );
      });
for (const mode of ['native', 'page'] as const)
  test(`${mode}: three exact sessions across two projects retain real xterm owners`, async (t) => {
    const page = await fixture(t, mode);
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 0);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
    for (const name of ['Build', 'Edit', 'Other project']) await openSession(page, name);
    assert.equal(await page.locator('.xterm').count(), 3);
    await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).click();
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    assert.equal(await page.locator('.xterm').count(), 3);
    assert.deepEqual(
      await page.evaluate(() => ({
        sockets: window.workspaceFixture.sockets.length,
        closed: window.workspaceFixture.sockets.map((s) => s.closed),
        actions: window.workspaceFixture.sockets.map((s) => s.sent[0]?.action),
        writes: window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length,
      })),
      {sockets: 3, closed: [0, 0, 0], actions: ['attach', 'attach', 'attach'], writes: 0}
    );
    assert.equal(await page.locator('#native-draft').inputValue(), 'unsaved');
    const callsBeforeHide = await page.evaluate(() => window.workspaceFixture.calls.length);
    await page.evaluate(() => {
      const f = window.workspaceFixture;
      f.api.setVisible(false);
      f.api.setVisible(true);
    });
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
    // Visibility is presentation-only since lifetime custody was removed: re-show
    // issues no new calls and the exact terminals keep their xterm owners.
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), callsBeforeHide);
    assert.equal(await page.locator('.xterm').count(), 3);
  });
test('installed control paths use the actual emitted project, chooser and original-target End UI', async (t) => {
  const page = await fixture(t, 'native');
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  const controls = await projectView(page, '7', 'Access');
  assert.equal(await controls.getAttribute('data-environment-id'), 'p' + '1'.repeat(24));
  assert.equal(await controls.locator('[data-control=copy]').getAttribute('data-clipboard-target'), '#soda-command-7');
  await assert.rejects(
    () => newManagedTerminal(page, 'alice/Alpha', 'Wrong target', 'p' + '2'.repeat(24)),
    /observed original project/
  );
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page
    .getByRole('dialog', {name: 'New terminal', exact: true})
    .getByRole('button', {name: 'Cancel', exact: true})
    .click();
  await newManagedTerminal(page, 'alice/Alpha', 'Installed path fixture', 'p' + '1'.repeat(24));
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  await terminalMenu(page, 'End terminal…');
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.body?.action === 'end').length),
    0
  );
  await page
    .getByRole('dialog', {name: 'End terminal confirmation', exact: true})
    .getByRole('button', {name: 'End terminal', exact: true})
    .click();
  await page.waitForFunction(() => !document.querySelector('soda-terminal'));
  assert.equal(
    await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.body?.action === 'end').length),
    1
  );
});

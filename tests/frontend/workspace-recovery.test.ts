import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, chooseFirstRepository, fixture, openSession} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

for (const state of ['missing-observation', 'native-unavailable', 'authority-unavailable'])
  test(`project header: ${state} never claims running or stopped`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await page.evaluate(async (state) => {
      const f = window.workspaceFixture,
        space = f.spaces[0];
      if (!space) throw Error('fixture');
      if (state === 'missing-observation') space.observed = null;
      else if (state === 'native-unavailable') space.native_unavailable = true;
      else {
        space.authority_unavailable = true;
        space.environment_administrator = false;
      }
      await f.api.refresh();
    }, state);
    assert.equal(await page.locator('.soda-project-state').innerText(), 'Status unavailable');
    assert.equal(await page.locator('.soda-project-status').innerText(), 'Status unavailable');
    assert.equal(await page.getByRole('heading', {name: 'Open your first terminal'}).count(), 0);
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 1);
  });
for (const [code, message] of [
  ['account_incomplete', 'Ask your Soda administrator to check the account setup'],
  ['membership_not_saved', 'Soda could not save your access'],
  ['unsupported_linux_login', 'username cannot be used for a project account'],
  ['unknown_failure', 'We couldn’t confirm whether you joined'],
])
  test(`first use: ${code} Join explains recovery without replay`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await page.evaluate((code) => window.workspaceFixture.setJoinFailure(code || ''), code);
    await page.getByRole('button', {name: 'Join project', exact: true}).click();
    await page.getByRole('heading', {name: 'Couldn’t join project'}).waitFor();
    assert((await page.locator('.soda-ready-project').innerText()).includes(message || ''));
    assert(
      !/Outcome unconfirmed|reserved project|earlier write/.test(await page.locator('.soda-ready-project').innerText())
    );
    assert.equal(await page.getByRole('heading', {name: 'Open your first terminal'}).count(), 0);
    assert.equal(await page.getByRole('button', {name: 'Try joining again', exact: true}).count(), 0);
    if (code === 'membership_not_saved') {
      await page.evaluate(() => {
        const space = window.workspaceFixture.spaces[0];
        if (space) space.login = 'alice';
      });
      await page.getByRole('button', {name: 'Check join status', exact: true}).click();
      await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
      assert.equal(
        await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length),
        2
      );
      assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
      return;
    }
    await page.getByRole('button', {name: 'Check join status', exact: true}).click();
    await page.getByText('You haven’t joined this project yet.').waitFor();
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 2);
    await page.evaluate(() => window.workspaceFixture.setJoinFailure(false));
    await page.getByRole('button', {name: 'Try joining again', exact: true}).click();
    await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 3);
  });
test('first use: incomplete and failed inventories never render guessed welcome', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(async () => {
    window.workspaceFixture.setComplete(false);
    await window.workspaceFixture.api.refresh();
  });
  assert.equal(await page.getByRole('heading', {name: 'Create your first project'}).count(), 0);
  await page.getByRole('heading', {name: 'Could not load projects'}).waitFor();
  await page.evaluate(async () => {
    window.workspaceFixture.setStatus(503);
    await window.workspaceFixture.api.refresh();
  });
  assert.equal(await page.getByRole('heading', {name: 'Create your first project'}).count(), 0);
});
test('departure during authorization cannot dispatch a late collection read', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.workspaceFixture;
    let release: (() => void) | undefined;
    f.pause(
      new Promise<void>((resolve) => {
        release = resolve;
      })
    );
    const pending = f.api.refresh();
    f.api.dispose();
    release?.();
    await pending;
  });
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 1);
});
for (const theme of ['light', 'dark'])
  for (const width of [1440, 390])
    test(`coherence recovery ${theme}/${width}: retry, stopped and Join failure keep explicit actions`, async (t) => {
      const page = await fixture(t, 'page', undefined, true);
      await page.setViewportSize({width, height: 844});
      await page.evaluate(async (theme) => {
        document.documentElement.style.colorScheme = theme;
        window.workspaceFixture.setStatus(503);
        await window.workspaceFixture.api.refresh();
      }, theme);
      await page.getByRole('heading', {name: 'Could not load projects'}).waitFor();
      assert(await page.getByRole('button', {name: 'Retry projects'}).isEnabled());
      assert.equal(await page.locator('.soda-workspace-navigation:visible').count(), 0);
      await captureSpacesComponent(page, `inventory-error-${theme}-${width}`);
      await page.evaluate(() => {
        const f = window.workspaceFixture;
        f.setStatus(200);
        f.pause(
          new Promise<void>((resolve) =>
            window.addEventListener(
              'release-inventory',
              () => {
                f.pause(undefined);
                resolve();
              },
              {once: true}
            )
          )
        );
      });
      await page.getByRole('button', {name: 'Retry projects'}).click();
      await page.getByRole('heading', {name: 'Loading projects…'}).waitFor();
      assert.equal(await page.locator('.soda-workspace').getByRole('button').count(), 0);
      await captureSpacesComponent(page, `loading-${theme}-${width}`);
      await page.evaluate(() => window.dispatchEvent(new Event('release-inventory')));
      await page.getByRole('heading', {name: 'Create your first project'}).waitFor();
      const primary = page.getByRole('button', {name: 'Create project', exact: true});
      await primary.hover();
      assert(
        await primary.evaluate((node) => {
          const probe = document.createElement('span');
          probe.style.color = 'var(--soda-button-primary-hover)';
          node.append(probe);
          const match = getComputedStyle(node).backgroundColor === getComputedStyle(probe).color;
          probe.remove();
          return match;
        })
      );
      await primary.click();
      await page.getByRole('button', {name: 'Back', exact: true}).click();
      await page.waitForFunction(() => document.activeElement?.textContent?.includes('Create project'));
      await chooseFirstRepository(page);
      await page.getByRole('button', {name: 'Create project', exact: true}).click();
      await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
      await page.evaluate(async () => {
        const f = window.workspaceFixture,
          space = f.spaces[0];
        if (!space?.observed) throw Error('fixture');
        space.observed.running = false;
        space.environment_administrator = false;
        await f.api.refresh();
      });
      await page.getByRole('heading', {name: 'Project not ready'}).waitFor();
      assert.equal(await page.getByRole('button', {name: 'Start project', exact: true}).count(), 0);
      await page.getByText('A project administrator must start this project.').waitFor();
      await captureSpacesComponent(page, `stopped-${theme}-${width}`);
      await page.evaluate(async () => {
        const f = window.workspaceFixture,
          space = f.spaces[0];
        if (!space?.observed) throw Error('fixture');
        space.observed.running = true;
        await f.api.refresh();
        f.setJoinFailure(true);
      });
      await page.getByRole('button', {name: 'Join project', exact: true}).click();
      await page.getByRole('heading', {name: 'Couldn’t join project'}).waitFor();
      await captureSpacesComponent(page, `join-error-${theme}-${width}`);
      assert.equal(
        await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length),
        2
      );
      assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
      assert(await page.locator('.soda-workspace').evaluate((node) => node.scrollWidth <= node.clientWidth));
    });
for (const status of [401, 403])
  test(`native recovery: ${status} project reads never masquerade as new configuration`, async (t) => {
    const page = await fixture(t, 'page', undefined, true);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    await chooseFirstRepository(page);
    await page.getByRole('button', {name: 'Create project', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    await page.evaluate((status) => window.workspaceFixture.setStatus(status), status);
    await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
    await page.getByRole('heading', {name: 'Project access changed'}).waitFor();
    assert.equal(await page.getByRole('heading', {name: 'Configure project', exact: true}).count(), 0);
    assert.equal(await page.getByRole('button', {name: 'Reload Spaces', exact: true}).isEnabled(), true);
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 1);
    await captureSpacesComponent(page, status === 401 ? 'project-reconnect' : 'project-access-changed');
  });
test('terminal handshake with a mismatched native generation is refused without attaching', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const outcome = await page.evaluate(async () => {
    const f = window.workspaceFixture;
    const environment = f.spaces[0]?.environment;
    if (!environment) throw Error('Missing fixture project');
    const peer = new f.Socket('ws://fixture/' + environment.id + '/terminal-sessions');
    let ready = false;
    peer.onmessage = () => {
      ready = true;
    };
    peer.send(
      JSON.stringify({
        action: 'attach',
        id: 'b'.repeat(32),
        repository_id: environment.repository_id,
        session_generation: 'mismatched-generation-for-proof',
        cols: 80,
        rows: 24,
      })
    );
    const deadline = Date.now() + 5000;
    while (peer.readyState !== 3 && Date.now() < deadline) await new Promise((resolve) => setTimeout(resolve, 10));
    return {
      closed: peer.readyState === 3,
      ready,
      attached: f.spaces[0]?.terminals.find((terminal) => terminal.id === 'b'.repeat(32))?.attached === true,
    };
  });
  assert.equal(outcome.closed, true, 'mismatched handshake must be refused');
  assert.equal(outcome.ready, false, 'mismatched handshake must not report ready');
  assert.equal(outcome.attached, false, 'mismatched handshake must not attach');
  assert.equal(
    await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').count(),
    1,
    'legitimate session stays connected'
  );
});
test('partial project list explains missing items and retries reads without replacing a terminal', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  const terminal = await page.locator('.xterm').elementHandle();
  await page.evaluate(async () => {
    const f = window.workspaceFixture;
    f.setComplete(false);
    await f.api.refresh();
  });
  await page
    .getByText('We couldn’t load all projects and terminals. Some may be missing from this list.', {exact: true})
    .waitFor();
  await page.setViewportSize({width: 390, height: 844});
  assert(await page.locator('#sodaspaces-status').evaluate((node) => node.scrollWidth <= node.clientWidth));
  await captureSpacesComponent(page, 'partial-list-mobile');
  await page.evaluate(() => window.workspaceFixture.setComplete(true));
  await page.getByRole('button', {name: 'Retry loading', exact: true}).click();
  await page.locator('#sodaspaces-status').waitFor({state: 'hidden'});
  assert(await terminal?.evaluate((node) => node.isConnected));
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
});

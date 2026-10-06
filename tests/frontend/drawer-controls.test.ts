import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh, click, writes} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

test('project control mount is inert; refresh only reads and never owns a terminal', async (t) => {
  const page = await fixture(t);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), 0);
  await refresh(page);
  await refresh(page);
  assert(
    await page.evaluate(() =>
      window.drawerFixture.calls.every((c) => c.url.startsWith('/-/extensions/pages/soda/spaces/api/'))
    )
  );
  assert.deepEqual(await writes(page), []);
  assert.deepEqual(
    await page.evaluate(() => ({
      count: window.drawerFixture.terminals.length,
      native: document.getElementById('native')?.textContent,
      input: document.querySelector<HTMLInputElement>('#native-input')?.value,
    })),
    {count: 0, native: 'Native action', input: 'unsaved'}
  );
});
test('project view tabs and app switches dispatch no reads or writes', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  const count = await page.evaluate(() => window.drawerFixture.calls.length);
  assert.equal(await page.locator('[role=tab][aria-selected=true]').innerText(), 'Environment');
  await click(page, 'Access');
  assert.equal(await page.getByRole('tabpanel', {name: 'Access'}).getAttribute('hidden'), null);
  await click(page, 'Environment');
  await page.evaluate(() => {
    window.dispatchEvent(new Event('blur'));
    document.dispatchEvent(new Event('visibilitychange'));
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), count);
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  await page.getByRole('tab', {name: 'Environment', exact: true}).press('ArrowRight');
  assert.equal(await page.locator('[role=tab][aria-selected=true]').innerText(), 'Access');
  assert.equal(await page.evaluate(() => document.activeElement?.textContent?.trim()), 'Access');
});
test('hidden access view cannot remove a key through a synthetic click', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.button('Remove saved key').click());
  assert.deepEqual(await writes(page), []);
});
test('compact-surface inert project controls cannot dispatch through a synthetic click', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Access');
  await page.locator('textarea').fill('ssh-ed25519 YWJj');
  const changed = await page.evaluate(async () => {
    const f = window.drawerFixture,
      button = await window.drawerFixture.showButton('Save public key');
    button.closest('soda-project-controls')?.setAttribute('inert', '');
    const before = f.calls.length;
    button.click();
    return f.calls.length - before;
  });
  assert.equal(changed, 0);
  assert.deepEqual(await writes(page), []);
});
test('unconfirmed Create keeps its notice without a permanent lock or another reservation', async (t) => {
  const page = await fixture(t, {absent: true});
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (call.method !== 'POST' || !call.url.endsWith('/api/environments')) return null;
      window.drawerFixture.state.absent = false;
      window.drawerFixture.state.provisioned = false;
      return new Response(null, {status: 502});
    })
  );
  await refresh(page);
  await click(page, 'Create environment');
  assert(await page.evaluate(() => window.drawerFixture.api.canRestore));
  await refresh(page);
  assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).count(), 0);
  assert.match(await page.locator('[data-control=result]').innerText(), /Project creation could not be confirmed/);
  assert.match(await page.locator('main').innerText(), /Provisioning incomplete/);
  assert.equal((await writes(page)).length, 1);
});
test('stale project controls cannot refresh/replay on focus', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  const count = await page.evaluate(() => window.drawerFixture.calls.length);
  await page.evaluate(async () => {
    window.dispatchEvent(new Event('pagehide'));
    window.dispatchEvent(new Event('focus'));
    await window.drawerFixture.api.refresh();
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), count);
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.equal(await page.locator('[data-control=refresh]').isDisabled(), true);
  assert.equal(await page.locator('#native').count(), 1);
});
for (const kind of ['HTML', 'oversized', '401', '403'] as const)
  test(`${kind} response cannot expose actions`, async (t) => {
    const page = await fixture(t);
    await page.evaluate(
      (kind) =>
        window.drawerFixture.setReply(async () =>
          kind === 'HTML'
            ? new Response('<html>login</html>', {headers: {'Content-Type': 'text/html'}})
            : kind === 'oversized'
              ? Response.json({padding: 'x'.repeat(65537)})
              : new Response(null, {status: Number(kind)})
        ),
      kind
    );
    await refresh(page);
    assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).count(), 0);
    assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  });
test('repeated project refresh never acquires terminal ownership', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await refresh(page);
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.doesNotMatch(await page.locator('main').innerText(), /terminal session ended/);
});
test('closing during the initial native read cannot publish controls or dispatch a mutation', async (t) => {
  const page = await fixture(t, {absent: true});
  await page.evaluate(async () => {
    let release: ((value: Response) => void) | undefined;
    const f = window.drawerFixture;
    f.setReply(async (call) =>
      call.url.includes('/api/environments?')
        ? new Promise((resolve) => {
            release = resolve;
          })
        : null
    );
    const pending = f.api.refresh();
    f.api.dispose();
    if (!release) throw Error('native read was not pending');
    release(Response.json({repository: {id: '7', owner: 'alice', name: 'demo'}, can_create: true, items: []}));
    await pending;
  });
  assert.deepEqual(await writes(page), []);
});
test('unknown key-save response cannot claim a confirmed key', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Access');
  await page.locator('textarea').fill('ssh-ed25519 YWJj');
  await click(page, 'Save public key');
  assert.match(
    await page.locator('[data-control=result]').innerText(),
    /We couldn’t confirm that this change finished/
  );
  assert(await page.evaluate(() => window.drawerFixture.api.canRestore));
  assert(!(await page.getByRole('button', {name: 'Save public key', exact: true}).isDisabled()));
  await refresh(page);
  assert.match(
    await page.locator('[data-control=result]').innerText(),
    /We couldn’t confirm that this change finished/
  );
});
test('hidden create and lifecycle actions cannot dispatch through their handlers', async (t) => {
  const page = await fixture(t, {admin: false});
  await refresh(page);
  await click(page, 'Create environment');
  await click(page, 'Start');
  assert.deepEqual(await writes(page), []);
});
test('copy uses own displayed login/IP without changing native access', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Copy SSH connection');
  assert.equal(await page.locator('[data-control=copy]').getAttribute('data-clipboard-target'), '#soda-command-7');
  assert.equal(await page.locator('[data-control=command]').inputValue(), 'ssh alice@10.89.0.2');
  assert.deepEqual(await writes(page), []);
});
test('reactive project view/Hide updates preserve draft identity and selection', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Access');
  await page.locator('textarea').fill('unsent public key');
  assert.deepEqual(
    await page.evaluate(async () => {
      const f = window.drawerFixture,
        input = f.root.querySelector('textarea'),
        controls = f.root.firstElementChild;
      if (!input || !controls) throw Error('missing fixture nodes');
      input.setSelectionRange(2, 5);
      f.button('Environment').click();
      await f.api.ready;
      f.root.hidden = true;
      f.root.hidden = false;
      f.button('Access').click();
      await f.api.ready;
      return {
        input: input === f.root.querySelector('textarea'),
        value: input.value,
        start: input.selectionStart,
        end: input.selectionEnd,
        controls: controls === f.root.firstElementChild,
        terminals: f.terminals.length,
      };
    }),
    {input: true, value: 'unsent public key', start: 2, end: 5, controls: true, terminals: 0}
  );
});
test('hidden completed project reads never attach a terminal on showing details', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.drawerFixture;
    f.root.hidden = true;
    await f.api.refresh();
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  await page.evaluate(async () => {
    const f = window.drawerFixture;
    f.root.hidden = false;
    await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.deepEqual(await writes(page), []);
});

test('render readiness after disposal cannot call terminal factory; reconnect remains retired', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.drawerFixture;
    const pending = f.api.refresh();
    const element = f.root.firstElementChild;
    f.api.dispose();
    if (element) f.root.append(element);
    await pending;
    await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.deepEqual(await writes(page), []);
});
for (const state of [{provisioned: false}, {unavailable: true}])
  test(`unavailable/incomplete state refuses terminal: ${JSON.stringify(state)}`, async (t) => {
    const page = await fixture(t, state);
    await refresh(page);
    assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
    assert.deepEqual(await writes(page), []);
  });

import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh, click, writes} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

test('create never implicitly joins, saves keys or starts; rapid clicks dispatch once', async (t) => {
  const page = await fixture(t, {absent: true});
  await refresh(page);
  await page.evaluate(async () => {
    const b = await window.drawerFixture.showButton('Create environment');
    b.click();
    b.click();
  });
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert.equal(sent[0]?.url, '/-/extensions/pages/soda/spaces/api/environments');
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {
    repository_id: '7',
    profile_id: 'rocky-headless',
    tailnet: {enabled: false},
  });
  assert.equal(sent[0]?.headers['x-extension-session-generation'], 'fixture-generation');
});
test('managed Create submits the reviewed binding, while an explicit Off ignores the managed default', async (t) => {
  for (const enabled of [true, false]) {
    const page = await fixture(t, {absent: true, tailnetAvailable: true, tailnetDefault: true});
    await refresh(page);
    const selection = page.getByRole('checkbox', {name: /Use appliance-managed Tailnet/});
    assert(await selection.isChecked());
    if (!enabled) await selection.uncheck();
    assert.deepEqual(await writes(page), []);
    await click(page, 'Create environment');
    const sent = await writes(page);
    assert.equal(sent.length, 1);
    assert.deepEqual(
      JSON.parse(sent[0]?.body || '{}').tailnet,
      enabled ? {enabled: true, revision: 'b'.repeat(32), binding: 'a'.repeat(32)} : {enabled: false}
    );
  }
});
test('network failure after Create keeps the project and offers network recovery, not recreation', async (t) => {
  const page = await fixture(t, {
    absent: true,
    tailnetAvailable: true,
    tailnetDefault: true,
    tailnetCreateOutcome: 'unconfirmed',
  });
  await refresh(page);
  await click(page, 'Create environment');
  assert.match(await page.locator('main').innerText(), /Project created\. Network setup unconfirmed/);
  assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).count(), 0);
  assert.equal((await writes(page)).length, 1);
  await refresh(page);
  assert.equal((await writes(page)).length, 1);
});

test('shared Network panel requires current administration and explicit target confirmation; double activation sends once', async (t) => {
  const page = await fixture(t, {tailnetAvailable: true});
  await refresh(page);
  await click(page, 'Network');
  await click(page, 'Use managed network');
  assert.deepEqual(await writes(page), []);
  await page.getByRole('checkbox', {name: /I confirm changing network access/}).check();
  await page.evaluate(async () => {
    const button = await window.drawerFixture.showButton('Use managed network');
    button.click();
    button.click();
  });
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {
    action: 'enable',
    revision: '0',
    confirm_id: 'p0123456789abcdef01234567',
    binding: 'a'.repeat(32),
  });
  assert.equal(sent[0]?.headers['x-extension-session-generation'], 'fixture-generation');
  assert.equal(await page.getByRole('checkbox', {name: /I confirm changing network access/}).isChecked(), false);
  assert.equal((await writes(page)).length, 1);
});
test('members get only observed own-account network access, not mutation controls or automatic authentication', async (t) => {
  const page = await fixture(t, {
    admin: false,
    tailnetAvailable: true,
    tailnetEnabled: true,
    tailnetRevision: 'b'.repeat(32),
    tailnetState: 'connected',
  });
  await refresh(page);
  await click(page, 'Network');
  assert.equal(await page.getByRole('button', {name: 'Turn Off', exact: true}).count(), 0);
  assert.equal(
    await page.getByRole('textbox', {name: 'Tailnet SSH command', exact: true}).inputValue(),
    'ssh alice@100.64.0.2'
  );
  assert.deepEqual(await writes(page), []);
  await page.evaluate(() => window.dispatchEvent(new Event('pagehide')));
  assert.equal(await page.getByRole('textbox', {name: 'Tailnet SSH command', exact: true}).count(), 0);
  assert.equal(await page.getByText('100.64.0.2', {exact: true}).count(), 0);
});
test('network startup uncertainty is not shown as connected and never replays on refresh', async (t) => {
  const page = await fixture(t, {tailnetAvailable: true});
  await refresh(page);
  await click(page, 'Network');
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/tailnet') && call.method === 'POST'
        ? Response.json({error: {code: 'tailnet_unconfirmed', message: 'private diagnostic'}}, {status: 502})
        : null
    )
  );
  await page.getByRole('checkbox', {name: /I confirm changing network access/}).check();
  await click(page, 'Use managed network');
  await page
    .getByText(
      'We couldn’t confirm that this change finished. Refresh status to check the result before trying again.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await page.getByText('private diagnostic', {exact: true}).count(), 0);
  await refresh(page);
  assert.equal((await writes(page)).length, 1);
  assert.equal(await page.getByRole('textbox', {name: 'Tailnet SSH command', exact: true}).count(), 0);
});
test('legacy OS observation is explicit, read-only and never becomes a creation profile', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((c) => c.url.endsWith('/os')).length), 0);
  await click(page, 'Inspect current OS');
  await page.getByText('Rocky Linux 9.7 · rocky 9.7', {exact: true}).waitFor();
  await page.getByText(/Legacy \/ unknown creation profile/).waitFor();
  assert.deepEqual(await writes(page), []);
  await page.evaluate(() => {
    window.drawerFixture.state.running = false;
  });
  await refresh(page);
  assert.equal(await page.getByText('Rocky Linux 9.7 · rocky 9.7', {exact: true}).count(), 0);
  await click(page, 'Inspect current OS');
  await page.getByText('OS release not read: environment is stopped.', {exact: true}).waitFor();
  assert.deepEqual(await writes(page), []);
});

test('OS text stays text, malformed receipts clear observations, and denial invalidates context', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/os')
        ? Response.json({
            environment: {id: 'p0123456789abcdef01234567', running: true},
            os_release: {id: 'rocky', version: '9.7', name: '<img src=x onerror=alert(1)>'},
            os_release_unavailable: false,
          })
        : null
    )
  );
  await click(page, 'Inspect current OS');
  await page.getByText('<img src=x onerror=alert(1)> · rocky 9.7', {exact: true}).waitFor();
  assert.equal(await page.locator('[aria-label="Observed project userspace"] img').count(), 0);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/os')
        ? Response.json({
            environment: {id: 'wrong-target', running: true},
            os_release: null,
            os_release_unavailable: true,
          })
        : null
    )
  );
  await click(page, 'Inspect current OS');
  await page.getByText('OS observation unavailable. Nothing was started or repaired.', {exact: true}).waitFor();
  assert.equal(await page.getByText('<img src=x onerror=alert(1)> · rocky 9.7', {exact: true}).count(), 0);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) =>
      call.url.endsWith('/os') ? Response.json({error: {code: 'forbidden'}}, {status: 403}) : null
    )
  );
  await click(page, 'Inspect current OS');
  await page.getByText(/Page context changed/).waitFor();
  assert.deepEqual(await writes(page), []);
});

test('Stop requires explicit shared-impact confirmation and Start is separate', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Stop');
  assert.deepEqual(await writes(page), []);
  await page.locator('input[type=checkbox]').first().check();
  await click(page, 'Stop');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'stop', confirm_stop: true});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
});
test('stopped environment has explicit Start only, and no terminal', async (t) => {
  const page = await fixture(t, {running: false});
  await refresh(page);
  await click(page, 'Start');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {action: 'start'});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
});

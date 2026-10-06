// Emitted Lit in Chromium; synthetic API/renderer/transport, not native proof.
import test from 'node:test';
import assert from 'node:assert/strict';
import {
  setupTerminalDriver,
  fixture,
  opening,
  actions,
  inputFrames,
  id,
  env,
  server,
  existing,
} from './fixtures/terminal-driver';
setupTerminalDriver();

test('mounting is inert and does not own Forgejo markup or beforeunload', async (t) => {
  const page = await fixture(t);
  assert.deepEqual(
    await page.evaluate(() => {
      const f = window.terminalFixture;
      window.dispatchEvent(new Event('beforeunload'));
      f.api.dispose();
      return {
        calls: f.calls.length,
        sockets: f.sockets.length,
        before: f.before(),
        native: !!document.getElementById('native'),
        children: f.root.children.length,
      };
    }),
    {calls: 0, sockets: 0, before: 1, native: true, children: 0}
  );
});
test('server locator is published before Create; native generation and bounded Unicode IO', async (t) => {
  const page = await fixture(t);
  await opening(page);
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), id);
  assert.deepEqual(await inputFrames(page), []);
  const reserved = await page.evaluate(() => window.terminalFixture.writes());
  assert.equal(reserved.length, 1);
  assert(reserved[0]?.url.endsWith('/terminal-sessions'));
  assert.deepEqual(JSON.parse(reserved[0]?.body || '{}'), {cols: 80, rows: 24, name: ''});
  await page.evaluate(() => window.terminalFixture.ready());
  assert.equal(
    await page.evaluate(() => window.terminalFixture.socket().url),
    server.url.origin.replace('http:', 'ws:') + `/-/extensions/panels/soda/workspace/api/environments/${env}/terminal`
  );
  assert.deepEqual((await inputFrames(page))[0], {
    action: 'create',
    id,
    name: '',
    repository_id: '7',
    session_generation: 'fixture-generation',
    cols: 80,
    rows: 24,
  });
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.term().osc), [0, 1, 2, 8, 52]);
  const calls = await page.evaluate(() => window.terminalFixture.calls);
  assert.equal(calls.length, 2);
  assert(!calls.some((call) => call.url.endsWith('/session')));
  for (const call of calls) {
    assert.equal(call.credentials, 'same-origin');
    assert.equal(call.redirect, 'error');
    assert.equal(call.headers['x-extension-session-generation'], 'fixture-generation');
  }
  await page.evaluate(() => {
    const f = window.terminalFixture;
    f.term().input('héllo\r');
    f.socket().message({type: 'output', data: btoa(String.fromCharCode(...new TextEncoder().encode('世界')))});
  });
  const data = (await inputFrames(page))[1]?.data;
  assert.equal(typeof data, 'string');
  if (typeof data !== 'string') throw Error('input missing');
  assert.equal(Buffer.from(data, 'base64').toString(), 'héllo\r');
  assert.equal(
    await page.evaluate(() => {
      const value = window.terminalFixture.term().writes[0];
      return typeof value === 'string' ? value : new TextDecoder().decode(value);
    }),
    '世界'
  );
});
test('sub-URL deployments prefix API and socket URLs', async (t) => {
  const page = await fixture(t, {subURL: '/forgejo.test'});
  await opening(page);
  const reserved = await page.evaluate(() => window.terminalFixture.writes());
  assert.equal(reserved.length, 1);
  assert(reserved[0]?.url.startsWith('/forgejo.test/-/extensions/panels/soda/workspace/api/environments/'));
  assert(reserved[0]?.url.endsWith('/terminal-sessions'));
  assert.equal(
    await page.evaluate(() => window.terminalFixture.socket().url),
    server.url.origin.replace('http:', 'ws:') +
      `/forgejo.test/-/extensions/panels/soda/workspace/api/environments/${env}/terminal`
  );
});
for (const state of ['ended', undefined])
  test(`confirmed native absence/exit cannot reopen the same ID (${state})`, async (t) => {
    const page = await fixture(t, {
      locator: {kind: 'existing', id},
      ...(state ? {existing: {...existing, state}} : {}),
    });
    await page.evaluate(() => window.terminalFixture.api.restore());
    const count = await page.evaluate(() => window.terminalFixture.calls.length);
    await page.evaluate(async () => {
      const f = window.terminalFixture;
      await f.api.open();
      await f.api.restore();
    });
    assert.equal(await page.evaluate(() => window.terminalFixture.calls.length), count);
    assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
    assert.equal(await page.evaluate(() => window.terminalFixture.locator()), null);
    assert.deepEqual(await actions(page), []);
  });
test('unavailable lookup is not absence; an explicit refresh may recover', async (t) => {
  const page = await fixture(t, {locator: {kind: 'existing', id}, existing});
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.setReply(async () => new Response(null, {status: 503}));
    await f.api.restore();
  });
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.locators), []);
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.setReply(undefined);
    await f.api.restore();
    f.socket().open();
  });
  assert.deepEqual((await inputFrames(page))[0], {
    action: 'attach',
    id,
    repository_id: '7',
    session_generation: 'fixture-generation',
    cols: 80,
    rows: 24,
  });
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.writes()), []);
});
test('lost creation reply uses the issued ID, not another Create or request namespace', async (t) => {
  const page = await fixture(t);
  await opening(page);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.socket().open();
    f.api.disconnect();
    await f.api.restore();
    f.socket().open();
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 2);
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.sockets.map((s) => s.sent[0]?.action)), [
    'create',
    'attach',
  ]);
  assert.equal((await inputFrames(page))[0]?.id, id);
  assert(!(await page.evaluate(() => window.terminalFixture.calls.some((c) => c.url.includes('terminal-attempts')))));
});
test('late open after hiding sends no Create, but keeps the pre-issued locator', async (t) => {
  const page = await fixture(t);
  await opening(page);
  await page.evaluate(() => {
    const f = window.terminalFixture;
    f.root.hidden = true;
    f.socket().open();
  });
  assert.deepEqual(await inputFrames(page), []);
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), id);
});
test('missing, retired pending and malformed locators refuse before mounting', async (t) => {
  const page = await fixture(t);
  assert.deepEqual(
    await page.evaluate(() => {
      const f = window.terminalFixture;
      return [
        undefined,
        null,
        'pending',
        {},
        {kind: 'other'},
        {kind: 'existing', id: 'bad'},
        {kind: 'pending', requestId: 'b'.repeat(32)},
      ].map((value) => {
        try {
          f.mountInvalidLocator(value);
          return false;
        } catch {
          return true;
        }
      });
    }),
    Array(7).fill(true)
  );
  assert.equal(await page.locator('soda-terminal').count(), 1);
});
for (const state of ['opening', 'ending'])
  test(`${state} is an observation, not a permanent admission flag`, async (t) => {
    const page = await fixture(t, {locator: {kind: 'existing', id}, existing: {...existing, state}});
    await page.evaluate(() => window.terminalFixture.api.restore());
    assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
    assert.deepEqual(await page.evaluate(() => window.terminalFixture.writes()), []);
    assert.match(await page.locator('#mount').innerText(), /transition is still in progress/);
  });
test('observed writer refuses takeover without creating work', async (t) => {
  const page = await fixture(t, {locator: {kind: 'existing', id}, existing: {...existing, attached: true}});
  await page.evaluate(() => window.terminalFixture.api.restore());
  await page.getByText('An existing writer is attached.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
});

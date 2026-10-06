import test from 'node:test';
import assert from 'node:assert/strict';
import {setupTerminalDriver, fixture, action, ready, inputFrames, id, server} from './fixtures/terminal-driver';
setupTerminalDriver();

test('native focus and hidden views cannot forward input', async (t) => {
  const page = await fixture(t);
  await ready(page);
  await page.locator('#native').focus();
  await page.evaluate(() => window.terminalFixture.term().input('native field'));
  assert.equal((await inputFrames(page)).length, 1);
  await page.evaluate(() => {
    const f = window.terminalFixture;
    f.api.focus();
    f.api.setVisible(false);
    f.term().input('hidden');
  });
  assert.equal((await inputFrames(page)).length, 1);
  await page.evaluate(() => {
    const f = window.terminalFixture;
    f.api.setVisible(true);
    f.api.focus();
    f.term().input('focused');
  });
  assert.equal((await inputFrames(page)).length, 2);
});
test('retired renderer cannot send through a successor attachment', async (t) => {
  const page = await fixture(t);
  await ready(page);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.api.disconnect();
    await f.api.restore();
    await f.ready();
    f.terms[0]?.input('stale input');
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 2);
  assert.equal((await inputFrames(page)).length, 1);
});
test('disposal and late socket callbacks close exactly once', async (t) => {
  const page = await fixture(t);
  await ready(page);
  await action(page, 'End terminal…');
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.button('End terminal').click();
    f.api.dispose();
    f.socket().onclose?.();
    f.socket().message({type: 'ready'});
    f.term().input('not replayed');
    await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.equal(await page.evaluate(() => window.terminalFixture.socket().closed), 1);
  assert.equal((await inputFrames(page)).length, 1);
});
test('input and output queues stay bounded', async (t) => {
  const page = await fixture(t, {slow: true});
  await ready(page);
  await page.evaluate(() => {
    for (let i = 0; i < 65; i++)
      window.terminalFixture.socket().message({type: 'output', data: btoa('\0'.repeat(4096))});
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
});
for (const overload of ['paste', 'transport'])
  test(`${overload} overload detaches without replay`, async (t) => {
    const page = await fixture(t);
    await ready(page);
    await page.evaluate((kind) => {
      const f = window.terminalFixture;
      if (kind === 'transport') f.socket().bufferedAmount = 65537;
      f.term().input(kind === 'paste' ? 'x'.repeat(65537) : 'blocked');
    }, overload);
    assert.equal((await inputFrames(page)).length, 1);
    assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  });
for (const frame of [
  null,
  {type: 'ready', extra: true},
  {type: 'output', data: ''},
  {type: 'output', data: '!'},
  {type: 'output', data: 'YQ==', extra: true},
  {type: 'session', id},
  {type: 'navigate', url: 'https://elsewhere.test'},
  'x'.repeat(32769),
])
  test(`invalid frame: ${JSON.stringify(frame).slice(0, 70)}`, async (t) => {
    const page = await fixture(t);
    await ready(page);
    await page.evaluate((frame) => window.terminalFixture.socket().message(frame), frame);
    assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
    assert.equal((await inputFrames(page)).length, 1);
    assert.equal(new URL(page.url()).origin, server.url.origin);
  });

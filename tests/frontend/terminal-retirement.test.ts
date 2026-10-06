import test from 'node:test';
import assert from 'node:assert/strict';
import {setupTerminalDriver, fixture, action, opening, ready, actions} from './fixtures/terminal-driver';
setupTerminalDriver();

for (const event of ['pagehide', 'pageshow'])
  test(`${event} retires access without End or replay`, async (t) => {
    const page = await fixture(t);
    await ready(page);
    await page.evaluate(async (event) => {
      const f = window.terminalFixture;
      window.dispatchEvent(event === 'pageshow' ? new PageTransitionEvent(event, {persisted: true}) : new Event(event));
      await f.api.ready;
      f.socket().message({type: 'output', data: 'YWJj'});
      window.dispatchEvent(new Event('focus'));
      f.root.querySelector('button')?.click();
    }, event);
    assert.deepEqual(
      await page.evaluate(() => ({
        disposed: window.terminalFixture.term().disposed,
        state: window.terminalFixture.socket().readyState,
        count: window.terminalFixture.sockets.length,
        output: window.terminalFixture.term().writes.length,
      })),
      {disposed: 1, state: 3, count: 1, output: 0}
    );
    assert.equal(await page.getByRole('button', {name: 'Reconnect terminal', includeHidden: true}).count(), 0);
    assert.deepEqual(await actions(page), []);
  });
test('visibility changes preserve renderer and perform no lifetime operation', async (t) => {
  const page = await fixture(t);
  await ready(page);
  assert.deepEqual(
    await page.evaluate(async () => {
      const f = window.terminalFixture,
        screen = f.root.querySelector('.soda-terminal-screen'),
        input = f.term().textarea;
      window.dispatchEvent(new Event('blur'));
      f.api.setVisible(false);
      f.root.hidden = true;
      f.socket().message({type: 'output', data: 'YWJj'});
      f.root.hidden = false;
      f.api.setVisible(true);
      await f.api.ready;
      return {
        state: f.socket().readyState,
        disposed: f.term().disposed,
        count: f.sockets.length,
        screen: screen === f.root.querySelector('.soda-terminal-screen'),
        input: input === f.term().textarea,
      };
    }),
    {state: 1, disposed: 0, count: 1, screen: true, input: true}
  );
  assert.deepEqual(await actions(page), []);
  assert.equal(await page.getByText('Keep for two hours', {exact: true}).count(), 0);
  assert.equal(await page.getByText('Continue working', {exact: true}).count(), 0);
});
test('late readiness does not steal native focus', async (t) => {
  const page = await fixture(t);
  await opening(page);
  await page.locator('#native').focus();
  await page.evaluate(() => window.terminalFixture.ready());
  assert.equal(await page.evaluate(() => document.activeElement?.id), 'native');
});
test('focus-triggered retirement cannot leave a late observer', async (t) => {
  const page = await fixture(t);
  await opening(page);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.term().textarea.addEventListener('focus', () => f.api.dispose(), {once: true});
    await f.ready();
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.observers()), {observed: 0, disconnected: 0});
  assert.deepEqual(await actions(page), []);
});
test('hidden controls and late renderer import cannot dispatch', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.root.hidden = true;
    f.button('Open terminal').click();
    await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.calls.length), 0);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    f.root.hidden = false;
    let release: (() => void) | undefined;
    f.waitRenderer(
      new Promise((resolve) => {
        release = resolve;
      })
    );
    f.button('Open terminal').click();
    f.api.dispose();
    release?.();
    await f.api.ready;
  });
  assert.equal(
    await page.evaluate(() => window.terminalFixture.sockets.length + window.terminalFixture.terms.length),
    0
  );
});
test('late reservation result cannot create a socket after retirement', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.terminalFixture;
    let release: ((value: Response) => void) | undefined;
    await new Promise<void>((entered) => {
      f.setReply(
        async () =>
          new Promise((resolve) => {
            release = resolve;
            entered();
          })
      );
      f.button('Open terminal').click();
    });
    f.api.invalidate();
    release?.(Response.json({id: 'a'.repeat(32)}));
    await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
});
for (const options of [{user: '2'}, {login: 'other'}, {responseStatus: 401}, {responseStatus: 403}])
  test(`actual reservation authorization rejects before native transport: ${JSON.stringify(options)}`, async (t) => {
    const page = await fixture(t, options);
    await action(page, 'Open terminal');
    await page.getByText('Could not authorize or attach.', {exact: false}).waitFor();
    assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
  });

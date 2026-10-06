import test from 'node:test';
import assert from 'node:assert/strict';
import {setupTerminalDriver, fixture, action, ready, end, actions, id} from './fixtures/terminal-driver';
setupTerminalDriver();

test('End is separately confirmed with native generation and survives attachment closing first', async (t) => {
  const page = await fixture(t);
  await ready(page);
  assert.deepEqual(
    await page.evaluate(() => {
      const f = window.terminalFixture;
      let escaped = false;
      f.root.addEventListener('keydown', () => {
        escaped = true;
      });
      const escape = new KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
      f.term().textarea.dispatchEvent(escape);
      f.term().key(new KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true, shiftKey: true, cancelable: true}));
      return {prevented: escape.defaultPrevented, escaped, focus: document.activeElement?.getAttribute('aria-label')};
    }),
    {prevented: true, escaped: false, focus: 'Terminal actions'}
  );
  await action(page, 'End terminal…');
  assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Cancel');
  assert.deepEqual(await actions(page), []);
  await page.evaluate(() => {
    const button = window.terminalFixture.button('End terminal');
    button.click();
    button.click();
  });
  await page.getByText('Native cleanup confirmed', {exact: false}).waitFor();
  const sent = await actions(page);
  assert.equal(sent.length, 1);
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'end'});
  assert(sent[0]?.url.endsWith('/terminal-sessions/' + id));
  assert.equal(sent[0]?.headers['x-extension-session-generation'], 'fixture-generation');
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), null);
});
test('unconfirmed End keeps the locator, but does not prevent a later explicit action', async (t) => {
  const page = await fixture(t);
  await ready(page);
  await page.evaluate(() =>
    window.terminalFixture.setReply(async (call) => (call.method === 'POST' ? Response.json({}) : null))
  );
  await end(page);
  await page.getByText('End was not confirmed.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), id);
  await page.evaluate(() => window.terminalFixture.setReply(undefined));
  await end(page);
  await page.getByText('Native cleanup confirmed', {exact: false}).waitFor();
  assert.equal((await actions(page)).length, 2);
});

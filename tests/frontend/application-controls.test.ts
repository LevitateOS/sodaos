import test, {before, after} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser} from 'playwright';
import type {} from './fixtures/application-fixture';

let browser: Browser, server: ReturnType<typeof Bun.serve>;
before(async () => {
  const result = await Bun.build({
    entrypoints: [import.meta.dir + '/fixtures/application-fixture.ts'],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(result.success, result.logs.join('\n'));
  const bundle = result.outputs[0];
  assert(bundle);
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      if (new URL(request.url).pathname === '/fixture.js')
        return new Response(bundle, {headers: {'Content-Type': 'text/javascript'}});
      return new Response('<!doctype html><main></main><script type="module" src="/fixture.js"></script>', {
        headers: {'Content-Type': 'text/html'},
      });
    },
  });
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {
  await browser?.close();
  server?.stop(true);
});

test('Spaces applications only read on explicit Refresh, fence double activation, and require confirmed Stop', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture());
  await page.getByText('Applications', {exact: true}).click();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.length), 0);
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.getByText('Stopped', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((c) => c.body).length), 0);
  await page.evaluate(() => {
    const button = Array.from(document.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Start Example'
    );
    assertButton(button);
    button.click();
    button.click();
    function assertButton(value: HTMLButtonElement | undefined): asserts value is HTMLButtonElement {
      if (!value) throw Error('missing start');
    }
  });
  await page.getByText('Ready', {exact: false}).waitFor();
  const writes = await page.evaluate(() => window.applicationFixture.calls.filter((c) => c.body));
  assert.equal(writes.length, 1);
  assert.deepEqual(writes[0]?.body, {
    action: 'start',
    revision: 1,
    expected_container: 'a'.repeat(64),
    confirm_stop: false,
  });
  assert(await page.getByRole('button', {name: 'Confirm Stop Example', exact: true}).isDisabled());
  await page.getByRole('checkbox', {name: 'Stop Example for everyone', exact: true}).check();
  await page.evaluate(() => {
    window.applicationFixture.fail = true;
  });
  await page.getByRole('button', {name: 'Confirm Stop Example', exact: true}).click();
  await page.getByText('Outcome unconfirmed. Inspect the application before acting again.', {exact: true}).waitFor();
  assert.equal(await page.getByText('private diagnostic', {exact: true}).count(), 0);
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isDisabled());
  assert.equal(await page.getByRole('link', {name: 'Open application', exact: true}).count(), 0);
});

test('suspended or replaced session discards pending application data without replaying an action', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture());
  await page.getByText('Applications', {exact: true}).click();
  await page.evaluate(() => {
    window.applicationFixture.delay = true;
  });
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.evaluate(async () => {
    const f = window.applicationFixture;
    f.element.active = false;
    f.element.sessionKey = 'generation-2';
    await f.element.updateComplete;
    f.release?.();
  });
  await page.waitForTimeout(50);
  assert.equal(await page.getByText('https://app.example.test', {exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((c) => c.body).length), 0);
  assert(await page.getByRole('button', {name: 'Refresh applications', exact: true}).isDisabled());
});

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

test('Opening Applications loads current status, fences double Start and requires confirmed Stop', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture());
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Stopped', {exact: false}).waitFor();
  await page.waitForFunction(() => window.applicationFixture.calls.some(c => c.body?.action === 'inspect'));
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((c) => c.body && c.body.action !== 'inspect').length), 0);
  assert.equal(await page.getByRole('button', {name: 'Start Example', exact: true}).isVisible(), false);
  await page.getByText('Start or stop application', {exact: true}).click();
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
  const writes = await page.evaluate(() => window.applicationFixture.calls.filter((c) => c.body && c.body.action !== 'inspect'));
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
  assert.equal(await page.getByRole('link', {name: /Open application/}).count(), 1);
});

test('suspended or replaced session discards pending application data without replaying an action', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture());
  await page.getByText('Applications', {exact: true}).click();
  await page.getByText('Stopped', {exact: false}).waitFor();
  await page.waitForFunction(() => window.applicationFixture.calls.some(c => c.body?.action === 'inspect'));
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
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter((c) => c.body && c.body.action !== 'inspect').length), 0);
  assert(await page.getByRole('button', {name: 'Refresh applications', exact: true}).isDisabled());
});

test('operator backup and Restore require explicit actions and reconcile the durable operation after a lost reply', async (t) => {
  const page = await browser.newPage();
  t.after(() => page.close());
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.waitForFunction(() => window.applicationFixture.calls.some(c => c.body?.action === 'inspect'));
  await page.getByText('Backups and recovery', {exact: true}).click();
  assert.equal(await page.evaluate(() => window.applicationFixture.calls.filter(c => c.path.endsWith('/snapshots') || c.path.endsWith('/restores')).length), 0);
  await page.getByRole('button', {name: 'Back up Example', exact: true}).click();
  await page.getByText('Stopped backup saved. Start the application when you are ready.', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Refresh backups for Example', exact: true}).click();
  await page.getByLabel('Backup for Example', {exact: true}).selectOption('a' + '8'.repeat(24));
  assert(await page.getByRole('button', {name: 'Restore Example', exact: true}).isDisabled());
  await page.getByRole('checkbox', {name: 'Restore Example to this backup', exact: true}).check();
  await page.evaluate(() => { window.applicationFixture.failRestore = true; });
  await page.getByRole('button', {name: 'Restore Example', exact: true}).click();
  await page.getByText('Restore outcome unconfirmed. Refresh applications and reconcile the same operation before acting again.', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Refresh applications', exact: true}).click();
  await page.getByText('Restore is pending. Reconcile this operation before starting the application.', {exact: true}).waitFor();
  await page.getByText('Start or stop application', {exact: true}).click();
  assert(await page.getByRole('button', {name: 'Start Example', exact: true}).isDisabled());
  assert(await page.getByRole('button', {name: 'Inspect Example', exact: true}).isDisabled());
  await page.getByRole('button', {name: 'Reconcile Restore Example', exact: true}).click();
  await page.getByText('Restored to fresh data at the same address. Start the application when you are ready.', {exact: true}).waitFor();
  const restores = await page.evaluate(() => window.applicationFixture.calls.filter(c => c.path.endsWith('/restores')));
  assert.equal(restores.length, 2);
  assert.deepEqual(restores[0]?.body, restores[1]?.body);
  assert.equal(restores[0]?.body?.snapshot, 'a' + '8'.repeat(24));
  assert.equal(await page.getByRole('link', {name: /Open application/}).getAttribute('href'), 'https://app.example.test');
  assert.equal(await page.getByText('lost reply', {exact: true}).count(), 0);
});


test('operator first-owner setup uses an ephemeral app link and finishes the same provisional generation', async (t) => {
  const page = await browser.newPage(); t.after(() => page.close());
  const errors: string[] = []; page.on('pageerror', error => errors.push(error.message));
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createApplicationFixture === 'function');
  await page.evaluate(() => window.createApplicationFixture(true));
  await page.getByText('Applications', {exact: true}).click();
  await page.waitForFunction(() => window.applicationFixture.calls.some(c => c.body?.action === 'inspect'));
  await page.getByText('Set up a new application (operator)', {exact: true}).click();
  await page.getByLabel('Admitted application references', {exact: true}).fill(JSON.stringify({name: 'NewApp', artifact_sha256: '1'.repeat(64), bundle_sha256: '2'.repeat(64), tools_sha256: '3'.repeat(64), runtime_image: 'sha256:' + '4'.repeat(64), image: 'sha256:' + '5'.repeat(64), origin: 'https://new.example.test', port: 24000}));
  await page.getByLabel('Soda administrator user ID', {exact: true}).fill('41');
  await page.getByRole('button', {name: 'Begin owner setup', exact: true}).click();
  const link = page.getByRole('link', {name: 'Create application owner ↗', exact: true});
  await link.waitFor();
  assert.equal(await link.getAttribute('href'), 'https://new.example.test/setup#operation=a' + 'c'.repeat(24) + '&token=' + 'd'.repeat(64));
  assert.equal(await link.getAttribute('rel'), 'noopener noreferrer');
  const create = await page.evaluate(() => window.applicationFixture.calls.find(c => c.path === '/api/applications/setups' && c.body));
  assert(create?.body); assert.equal(create.body.administrator_id, '41');
  assert.equal('owner_team' in (create.body.application as Record<string, unknown>), false);
  assert.match(String((create.body.application as Record<string, unknown>).data), /^a[0-9a-f]{24}$/);
  await page.evaluate(() => {window.applicationFixture.setupExpiresAt = Math.floor(Date.now() / 1000) - 1;});
  await page.getByRole('button', {name: 'Refresh owner setups', exact: true}).click();
  await page.getByText(/The link has expired/).waitFor();
  assert.equal(await link.count(), 0);
  assert(await page.getByRole('button', {name: 'Open owner setup NewApp', exact: true}).isDisabled());
  assert(await page.getByRole('button', {name: 'Inspect owner setup NewApp', exact: true}).isEnabled());
  await page.getByRole('button', {name: 'Confirm owner and finish NewApp', exact: true}).click();
  await page.getByText('Owner confirmed. The application is installed and stopped. Refresh applications to start it.', {exact: true}).waitFor();
  const completed = await page.evaluate(() => window.applicationFixture.calls.find(c => c.path.endsWith('/actions') && c.body?.action === 'complete'));
  assert.deepEqual(completed?.body, {action: 'complete', expected_container: 'c'.repeat(64), confirm: true});
  assert.deepEqual(errors, []);
});

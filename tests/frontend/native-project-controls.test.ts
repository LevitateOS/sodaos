import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import {join, resolve} from 'node:path';
import type {State} from './fixtures/drawer-fixture';

const root = resolve(import.meta.dirname, '../..');
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;

before(async () => {
  const build = await Bun.build({
    entrypoints: [join(root, 'tests/frontend/fixtures/drawer-fixture.ts')],
    target: 'browser',
    format: 'esm',
    minify: true,
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, `Native project fixture build failed: ${build.logs.join('\n')}`);
  const [fixture] = build.outputs;
  assert(fixture && build.outputs.length === 1);
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const pathname = new URL(request.url).pathname;
      if (pathname === '/assets/project-controls-fixture.js')
        return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
      if (pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><button id="native">Native action</button><input id="native-input" value="unsaved"><main></main><script type="module" src="/assets/project-controls-fixture.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {
  await browser?.close();
  server?.stop(true);
});

async function fixture(t: TestContext, extra: Partial<State> = {}) {
  const page = await browser.newPage();
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async (state) => {
    window.drawerFixture = window.createDrawerFixture(state);
    await window.drawerFixture.api.ready;
  }, extra);
  return page;
}
async function refresh(page: Page) {
  await page.evaluate(() => window.drawerFixture.api.refresh());
}
async function click(page: Page, label: string) {
  await page.evaluate(async (text) => {
    (await window.drawerFixture.showButton(text)).click();
  }, label);
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
}
const writes = (page: Page) => page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET'));

test('native project controls mount inertly and preserve Forgejo content', async (t) => {
  const page = await fixture(t);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), 0);
  await refresh(page);
  assert.deepEqual(await writes(page), []);
  assert.deepEqual(
    await page.evaluate(() => ({
      native: document.getElementById('native')?.textContent,
      input: document.querySelector<HTMLInputElement>('#native-input')?.value,
      terminals: window.drawerFixture.terminals.length,
    })),
    {native: 'Native action', input: 'unsaved', terminals: 0}
  );
});

test('Create dispatches once with native generation and does not join or start', async (t) => {
  const page = await fixture(t, {absent: true});
  await refresh(page);
  await page.evaluate(async () => {
    const button = await window.drawerFixture.showButton('Create environment');
    button.click();
    button.click();
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

test('Create uses the reviewed network binding and keeps network failure separate', async (t) => {
  const page = await fixture(t, {
    absent: true,
    tailnetAvailable: true,
    tailnetDefault: true,
    tailnetCreateOutcome: 'unconfirmed',
  });
  await refresh(page);
  assert(await page.getByRole('checkbox', {name: /Use appliance-managed Tailnet/}).isChecked());
  await click(page, 'Create environment');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}').tailnet, {
    enabled: true,
    revision: 'b'.repeat(32),
    binding: 'a'.repeat(32),
  });
  assert.match(await page.locator('main').innerText(), /Project created\. Network setup unconfirmed/);
  await refresh(page);
  assert.equal((await writes(page)).length, 1);
});

test('network changes require target confirmation and dispatch once', async (t) => {
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
});

test('Stop requires shared-impact confirmation and never opens a terminal', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Stop');
  assert.deepEqual(await writes(page), []);
  await page.locator('input[type=checkbox]').first().check();
  await click(page, 'Stop');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {action: 'stop', confirm_stop: true});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
});

test('Join is explicit and leaves saved SSH keys out by default', async (t) => {
  const page = await fixture(t, {admin: false, member: false});
  await refresh(page);
  await click(page, 'Join environment');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert(sent[0]?.url.endsWith('/join'));
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {ssh_keys: 'none'});
});

test('private key input is rejected before a native request', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Access');
  await page.locator('textarea').fill(['-----BEGIN OPENSSH', ' PRIVATE KEY-----'].join(''));
  await click(page, 'Save public key');
  assert.deepEqual(await writes(page), []);
  assert.match(await page.locator('[data-control=result]').innerText(), /Never upload a private key/);
});

test('saved-key removal does not apply keys to the environment', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  page.once('dialog', (dialog) => {
    assert.match(dialog.message(), /may remove your final saved development key/);
    void dialog.dismiss();
  });
  await click(page, 'Remove saved key');
  assert.deepEqual(await writes(page), []);
  page.once('dialog', (dialog) => {
    assert.match(dialog.message(), /may remove your final saved development key/);
    void dialog.accept();
  });
  await click(page, 'Remove saved key');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert.equal(sent[0]?.method, 'DELETE');
  assert(sent[0]?.url.endsWith('/me/development-keys/1'));
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {confirm_last: true});
  assert.match(await page.locator('[data-control=result]').innerText(), /Existing project SSH access is unchanged/);
});

test('removing the last installed key needs reviewed explicit Apply', async (t) => {
  const page = await fixture(t, {saved: []});
  await refresh(page);
  await click(page, 'Review this project’s SSH keys');
  await click(page, 'Apply reviewed saved keys to this project');
  assert.deepEqual(await writes(page), []);
  await page.locator('input[type=checkbox]').nth(1).check();
  await click(page, 'Apply reviewed saved keys to this project');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {
    revision: 'a'.repeat(64),
    saved_fingerprints: [],
    confirm_empty: true,
  });
});

test('Forgejo public-key selection stays a read until explicit profile save', async (t) => {
  const page = await fixture(t, {saved: []});
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/me/forgejo-keys?page=1'))
        return Response.json({
          items: [
            {
              id: '7',
              title: '<script>not markup</script>',
              fingerprint: 'SHA256:' + 'A'.repeat(43),
              public_key: 'ssh-ed25519 YWJj\n',
            },
          ],
          page: 1,
          more: false,
        });
      if (call.method === 'POST' && call.url.endsWith('/me/development-keys'))
        return Response.json({items: [{id: '1', fingerprint: 'SHA256:' + 'A'.repeat(43)}]});
      return null;
    })
  );
  await refresh(page);
  await click(page, 'Review my Forgejo public keys');
  assert.deepEqual(await writes(page), []);
  assert.equal(await page.locator('soda-project-controls script').count(), 0);
  await click(page, 'Select for review');
  assert.equal(await page.locator('textarea').inputValue(), 'ssh-ed25519 YWJj\n');
  assert.deepEqual(await writes(page), []);
  await click(page, 'Save public key');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert(sent[0]?.url.endsWith('/me/development-keys'));
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {public_key: 'ssh-ed25519 YWJj'});
});

test('an unconfirmed Create does not retry a reservation', async (t) => {
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
  await refresh(page);
  assert.equal((await writes(page)).length, 1);
  assert.match(await page.locator('[data-control=result]').innerText(), /Project creation could not be confirmed/);
});

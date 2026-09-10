import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import {chromium} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response';

const root = path.resolve(import.meta.dirname, '../..');
const files: Record<string, string> = payload;
const origin = 'https://forgejo.example.test';
test('runner response shares Cockpit one-slot validation', () => {
  assert.throws(() => decodeRunnerResponse('list', {runners: [], runner_count: 1, active_listeners: 0, total_capacity: 0}));
  assert.throws(() => decodeRunnerResponse('remove', {ok: false}));
  assert.throws(() => decodeRunnerResponse('list', {runners: [], runner_count: 0, active_listeners: -1, total_capacity: 0}));
  assert.equal(decodeRunnerResponse('start', {ok: true}).ok, true);
});
test('actual Go settings HTML and emitted Lit reach runner mutations with fresh guards', {skip: !process.env.SODA_RUNNERS_PAGE_HTML}, async t => {
  const fixture: unknown = await Bun.file(process.env.SODA_RUNNERS_PAGE_HTML || '').json();
  assert(fixture && typeof fixture === 'object' && 'html' in fixture && typeof fixture.html === 'string' && 'csp' in fixture && typeof fixture.csp === 'string');
  const html = fixture.html, csp = fixture.csp;
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const page = await browser.newPage(); const errors: string[] = []; page.on('pageerror', e => errors.push(e.message));
  let registered = false, failList = false, failMutation = false, denied = false, sessionReads = 0;
  const mutations: {path: string; body: Record<string, unknown>}[] = [];
  await page.route(origin + '/**', async route => {
    const request = route.request(), url = new URL(request.url()), pathname = url.pathname;
    if (pathname === '/-/soda/settings/runners') {await route.fulfill({body: html, contentType: 'text/html', headers: {'Content-Security-Policy': csp}}); return;}
    if (pathname === '/-/soda/api/session') {
      sessionReads++; assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      await route.fulfill({status: denied ? 403 : 200, contentType: 'application/json', body: JSON.stringify({user: {id: '1', login: 'alice'}, csrf_token: 'csrf-alice', soda_operator: true, forgejo_url: origin})}); return;
    }
    if (pathname.startsWith('/-/soda/api/settings/runners')) {
      assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      if (request.method() === 'POST') {
        assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
        const body: unknown = request.postDataJSON(); assert(body && typeof body === 'object' && !Array.isArray(body));
        mutations.push({path: pathname, body: body as Record<string, unknown>}); registered = !pathname.endsWith('/remove');
        await route.fulfill({status: failMutation ? 502 : 200, contentType: 'application/json', body: failMutation ? '{"error":{"message":"must-not-render-native-secret"}}' : '{"ok":true}'}); return;
      }
      const runners = registered ? [{id: 'one', provider: 'forgejo', registration_url: origin, account: 'soda-runner-one', architecture: 'x86-64', version: 'fixture', capacity: 1, service: {load: 'loaded', active: 'active', sub: 'running', enabled: 'enabled'}}] : [];
      await route.fulfill({status: failList ? 503 : 200, contentType: 'application/json', body: JSON.stringify({forgejo_url: origin, runners, runner_count: runners.length, active_listeners: runners.length, total_capacity: runners.length})}); return;
    }
    const source = files['public' + pathname];
    if (source) {
      const file = source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : path.join(root, source);
      await route.fulfill({body: Buffer.from(await Bun.file(file).arrayBuffer()), contentType: pathname.endsWith('.js') ? 'text/javascript' : pathname.endsWith('.css') ? 'text/css' : 'application/octet-stream'}); return;
    }
    await route.fulfill({status: 404});
  });
  await page.goto(origin + '/-/soda/settings/runners');
  await page.getByText('No local runners registered.').waitFor();
  assert.equal(mutations.length, 0);
  await page.getByLabel('Local runner ID', {exact: true}).fill('one');
  await page.getByLabel('Forgejo runner UUID').fill('33834eef-e758-48c4-a676-1745426747aa');
  await page.getByLabel('Labels (comma-separated; Forgejo requires name:host)').fill('native:host');
  await page.getByLabel('Registration token').fill('synthetic-secret-never-store');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  await page.getByRole('button', {name: 'start one', exact: true}).waitFor();
  assert.equal(await page.getByLabel('Registration token').inputValue(), '');
  assert.equal(mutations[0]?.body.registration_token, 'synthetic-secret-never-store');
  assert.equal(mutations[0]?.body.registration_url, '');
  assert(!await page.evaluate(() => document.documentElement.outerHTML.includes('synthetic-secret-never-store') || JSON.stringify(localStorage).includes('synthetic-secret-never-store') || JSON.stringify(sessionStorage).includes('synthetic-secret-never-store')));
  for (const action of ['start', 'stop', 'restart']) {
    await page.getByRole('button', {name: `${action} one`, exact: true}).click();
    await page.getByLabel('Exact runner ID').fill('wrong'); await page.getByRole('button', {name: `Confirm ${action}`, exact: true}).click();
    await page.getByText('Type the exact runner ID to confirm.').waitFor();
    await page.getByLabel('Exact runner ID').fill('one'); await page.getByRole('button', {name: `Confirm ${action}`, exact: true}).click();
    await page.waitForFunction(() => !document.querySelector<HTMLButtonElement>('button')?.disabled);
  }
  assert.equal(mutations.length, 4); assert(sessionReads >= 8);
  failMutation = true;
  await page.getByRole('button', {name: 'remove one', exact: true}).click();
  await page.getByText(/Permanently remove this local Linux account/).waitFor();
  await page.getByLabel('Exact runner ID').fill('one'); await page.getByRole('button', {name: 'Confirm remove', exact: true}).click();
  await page.getByText(/Operation unconfirmed: local account/).waitFor();
  assert(!(await page.locator('body').innerText()).includes('must-not-render-native-secret'));
  failList = true; await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.getByRole('heading', {name: 'Stale observations'}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Register and start listener'}).isDisabled(), true);
  denied = true; await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.getByText('Reconnect explicitly').waitFor();
  assert.equal(mutations.length, 5); assert.deepEqual(errors, []);
});

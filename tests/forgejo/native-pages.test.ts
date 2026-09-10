import assert from 'node:assert/strict';
import {readFile, writeFile, unlink} from 'node:fs/promises';
import {randomUUID} from 'node:crypto';
import path from 'node:path';
import {test} from 'node:test';
import {chromium} from 'playwright';
import {buildForgejoModule} from '../../scripts/build-forgejo';

const root = path.resolve(import.meta.dirname, '../..');
const origin = 'http://localhost:3300';

// Uses actual Forgejo HTML, scripts, authentication and assets. No response mocks,
// provider/project operations, cookie borrowing or native-header fixtures.
// Prepare current canonical preview assets first. Without a Soda backend, the
// native host must show stable connection failure; the separate connection journey
// supplies real Soda/OAuth proof.
test('native Forgejo hosts all three bounded Soda views', {
  skip: process.env.SODA_FORGEJO_NATIVE_PAGES !== '1',
  timeout: 120000,
}, async t => {
  const context = await chromium.launchPersistentContext(path.join(root, '.local/screenshot-fixture-profile'), {
    channel: 'chrome', headless: true, chromiumSandbox: true,
  });
  t.after(() => context.close());
  // Disable the browser HTTP cache while retaining real, unmodified responses.
  await context.route(`${origin}/**`, route => route.continue());
  const page = context.pages()[0] || await context.newPage();
  const errors: string[] = [], badAssets: string[] = [], sodaRequests: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => {
    if (new URL(response.url()).pathname.startsWith('/assets/') && response.status() >= 400) badAssets.push(new URL(response.url()).pathname);
  });
  page.on('request', request => {
    const url = new URL(request.url());
    if (url.pathname.startsWith('/-/soda/')) sodaRequests.push(request.method() + ' ' + url.pathname);
  });

  const revision = (await readFile(path.join(root, 'appliance/forgejo/templates/custom/header.tmpl'), 'utf8')).match(/name="soda-presentation-revision" content="([^"]+)"/)?.[1];
  assert(revision);
  // This native entry preserves the destination even when remember-me first has
  // to restore the native session; the raw root query alone does not guarantee it.
  await page.goto(`${origin}/user/login?redirect_to=${encodeURIComponent('/')}`);
  assert.equal(page.url(), origin + '/', 'Renew the authorized screenshot fixture login before this review');
  const baselineResponse = await page.request.get(origin + '/');
  const baselinePolicy = baselineResponse.headers()['content-security-policy'];
  const baseline = await page.evaluate(html => new DOMParser().parseFromString(html, 'text/html').querySelector('.soda-dashboard')?.outerHTML, await baselineResponse.text());
  assert(baseline);
  assert.equal(await page.locator('#sodaspaces-root').count(), 1);
  const nativeActor = await page.locator('#soda-settings-link').getAttribute('data-actor');
  assert(nativeActor);

  const views = [
    {query: '?soda-view=spaces', title: 'Spaces', destination: '/-/soda/spaces', repository: ''},
    {query: '?soda-view=runners', title: 'Runners', destination: '/-/soda/settings/runners', repository: ''},
    {query: '?soda-view=repository-spaces&repository_id=9223372036854775807', title: 'Repository Spaces settings', destination: '/-/soda/repositories/9223372036854775807/settings/spaces', repository: '9223372036854775807'},
  ];
  for (const view of views) for (const width of [1440, 390]) {
    await t.test(`${view.title} at ${width}px`, async () => {
      await page.setViewportSize({width, height: 1000});
      const response = await page.goto(origin + '/' + view.query);
      assert.equal(response?.status(), 200);
      assert.equal(page.url(), origin + '/' + view.query);
      assert.equal(response?.headers()['content-security-policy'], baselinePolicy, 'Soda must not replace native document policy');
      assert.equal(await page.locator('meta[name="soda-presentation-revision"]').getAttribute('content'), revision);
      const mount = page.locator('#soda-native-content');
      await mount.getByRole('button', {name: 'Retry connection'}).waitFor();
      assert((await page.title()).startsWith(view.title + ' - '));
      assert.equal(await mount.getAttribute('data-actor'), nativeActor);
      assert.equal(await mount.getAttribute('data-repository-id'), view.repository);
      assert.equal(await mount.getAttribute('data-destination'), view.destination);
      assert.equal(await page.locator('main,[role=main]').count(), 1);
      assert.equal(await page.locator('#sodaspaces-root,.soda-dashboard,.soda-workspace,soda-runners').count(), 0);
      assert.equal(await page.locator('.soda-native-page form').count(), 0, 'Step 1 mounts no privileged controls');
      assert(await page.locator('#navbar').isVisible());
      assert.equal(await page.locator('#navbar a[href="/admin"]').count(), 0, 'fixture remains a non-admin');
      const bounds = await page.locator('.soda-native-page').boundingBox();
      assert(bounds && bounds.width > 0 && bounds.x >= 0 && bounds.x + bounds.width <= width);
    });
  }

  await page.setViewportSize({width: 1440, height: 1000});
  await page.goto(origin + '/?soda-view=spaces');
  const profile = page.locator('#navbar details').filter({has: page.locator('a[data-url="/user/logout"]')});
  await profile.locator('summary').focus();
  await page.keyboard.press('Enter');
  assert(await profile.locator('a[href="/user/settings"]').isVisible());
  assert(await profile.locator('a[href="/soda-screenshot"]').isVisible());
  await profile.locator('a[href="/user/settings"]').click();
  assert.equal(page.url(), origin + '/user/settings');
  assert.equal(await page.locator('form[action="/user/settings"]').getAttribute('method'), 'post');
  await page.goto(origin + '/?soda-view=spaces');
  const bell = page.locator('#navbar a[href="/notifications"]').filter({visible: true}).first();
  const notification = page.waitForResponse(response => {
    const url = new URL(response.url());
    return url.pathname === '/notifications' && url.searchParams.get('soda-preview') === 'true';
  });
  await bell.click();
  assert.equal((await notification).status(), 200);
  assert(await page.locator('#soda-notification-preview').isVisible());
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('#soda-notification-preview').isVisible(), false);

  for (const query of ['?soda-view=unknown', '?soda-view=spaces&soda-view=runners']) {
    const response = await page.goto(origin + '/' + query);
    assert(response);
    assert.equal(await page.locator('#soda-native-content').count(), 0);
    assert.equal(await page.locator('#sodaspaces-root').count(), 1);
    const body = await page.evaluate(html => new DOMParser().parseFromString(html, 'text/html').querySelector('.soda-dashboard')?.outerHTML, await response.text());
    assert.equal(body, baseline, 'unchanged native dashboard HTML, independent of responsive Vue rendering');
  }
  for (const query of [
    '?soda-view=spaces&repository_id=1', '?soda-view=repository-spaces',
    '?soda-view=repository-spaces&repository_id=01', '?soda-view=repository-spaces&repository_id=0',
    '?soda-view=repository-spaces&repository_id=1&repository_id=2',
    '?soda-view=repository-spaces&repository_id=9223372036854775808',
    '?soda-view=repository-spaces&repository_id=%3Cscript%3E',
  ]) {
    await page.goto(origin + '/' + query);
    assert.equal(await page.locator('#soda-native-content,#sodaspaces-root').count(), 0);
    assert.equal(await page.locator('[role=alert]').innerText(), 'Invalid Soda page destination.');
  }

  // A unique, test-owned public script imports real packaged modules and exercises
  // xterm layout/output locally. It opens no websocket, project or native terminal.
  const smokeName = `native-page-smoke-${randomUUID()}.js`;
  const smokePath = path.join(root, '.artifacts/local-forgejo/public/assets', smokeName);
  const compiled = await buildForgejoModule(path.join(root, 'tests/forgejo/fixtures/native-page-smoke.ts'), 'public/assets/' + smokeName);
  await writeFile(smokePath, await compiled.text(), {flag: 'wx'});
  t.after(() => unlink(smokePath));
  await page.goto(origin + '/?soda-view=spaces');
  await page.addScriptTag({type: 'module', url: origin + '/assets/' + smokeName});
  await page.locator('#soda-native-content[data-asset-probe]').waitFor();
  assert.equal(await page.locator('.xterm,.soda-workspace').count(), 0, 'test probe disposes its local renderer');
  assert(sodaRequests.every(request => request === 'GET /-/soda/api/session'), 'page host must not request Soda data or effects');
  assert.deepEqual(badAssets, []);
  assert.deepEqual(errors, []);
});

test('native login returns guests to Soda views and native logout still works', {
  skip: process.env.SODA_FORGEJO_NATIVE_PAGES !== '1', timeout: 60000,
}, async t => {
  // A fresh browser logs in only as the authorized fixture account. Its password
  // stays in the ignored private file and is never logged or sent via argv.
  const browser = await chromium.launch({channel: 'chrome', headless: true, chromiumSandbox: true});
  t.after(() => browser.close());
  const page = await browser.newPage();
  for (const query of ['?soda-view=spaces', '?soda-view=runners', '?soda-view=repository-spaces&repository_id=1']) {
    await page.goto(origin + '/' + query);
    assert.equal(await page.locator('#soda-native-content').count(), 0);
    assert.equal(await page.locator('#soda-settings-link').count(), 0);
  }
  const destination = '/?soda-view=spaces';
  await page.goto(`${origin}/user/login?redirect_to=${encodeURIComponent(destination)}`);
  const saved = await readFile(path.join(root, '.local/screenshot-fixture/create-output.txt'), 'utf8');
  const password = saved.match(/generated random password is '([^']+)'/)?.[1];
  assert(password, 'Authorized fixture credential unavailable');
  // Sanitize a failed fill: Playwright's diagnostics may include its argument.
  try {
    await page.locator('input[name="user_name"]').fill('soda-screenshot');
    await page.locator('input[name="password"]').fill(password);
  } catch { throw new Error('Could not fill the authorized fixture login form'); }
  await page.locator('form[action="/user/login"] button.ui.primary').click();
  await page.waitForURL(origin + destination);
  await page.locator('#soda-native-content').getByRole('button', {name: 'Retry connection'}).waitFor();
  for (const query of ['?soda-view=runners', '?soda-view=repository-spaces&repository_id=1']) {
    await page.goto(`${origin}/user/login?redirect_to=${encodeURIComponent('/' + query)}`);
    assert.equal(page.url(), origin + '/' + query);
    assert.equal(await page.locator('#soda-native-content').count(), 1);
  }
  // Native account forms use Go's cross-origin request protection, not hidden
  // CSRF fields. The upstream logout route is outside that middleware; retain
  // its existing behavior and test the protected account route without editing it.
  const denied = await page.request.post(origin + '/user/settings', {
    headers: {Origin: 'https://cross-site.invalid', 'Sec-Fetch-Site': 'cross-site'},
    maxRedirects: 0,
  });
  assert.equal(denied.status(), 403);
  await page.reload();
  assert.equal(await page.locator('#soda-native-content').count(), 1);
  const profile = page.locator('#navbar details').filter({has: page.locator('a[data-url="/user/logout"]')});
  await profile.locator('summary').click();
  const response = page.waitForResponse(value => new URL(value.url()).pathname === '/user/logout' && value.request().method() === 'POST');
  await profile.locator('a[data-url="/user/logout"]').click();
  // This read-only preview has no Soda backend: choose the explicit native-only escape.
  await page.getByRole('button', {name: 'Sign out of Forgejo only'}).click();
  assert.equal((await response).status(), 200);
  await page.waitForURL(origin + '/');
  assert.equal(await page.locator('#soda-settings-link,#soda-native-content').count(), 0);
});

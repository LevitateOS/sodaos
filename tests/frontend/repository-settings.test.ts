import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import {chromium} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {creationProfile} from '../../frontend/spaces/sodaspaces-api';

const profile = {id: 'rocky-headless', distribution: 'rocky', version: '10.2', interface: 'headless', architecture: 'amd64', image: 'sha256:' + 'a'.repeat(64), revision: 'b'.repeat(40)};
test('creation identity rejects unsupported and incomplete metadata', () => {
  assert.deepEqual(creationProfile(profile), profile);
  for (const bad of [{...profile, id: 'fedora-kde'}, {...profile, revision: ''}, {...profile, image: 'latest'}, {...profile, architecture: 'x86'}, {...profile, version: '<script>'}]) assert.throws(() => creationProfile(bad));
});
test('Go repository settings mounts shared creation/access controls and preserves immutable/legacy display', {skip: !process.env.SODA_REPOSITORY_SETTINGS_HTML}, async t => {
  const raw: unknown = await Bun.file(process.env.SODA_REPOSITORY_SETTINGS_HTML || '').json();
  assert(raw && typeof raw === 'object' && 'html' in raw && typeof raw.html === 'string' && 'csp' in raw && typeof raw.csp === 'string');
  const html = raw.html, csp = raw.csp;
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const page = await browser.newPage(), errors: string[] = []; page.on('pageerror', e => errors.push(e.message));
  const origin = 'https://forgejo.example.test', files: Record<string, string> = payload, id = 'p0123456789abcdef01234567';
  let exists = false, legacy = false, available = true;
  const writes: unknown[] = [];
  await page.route(origin + '/**', async route => {
    const request = route.request(), pathname = new URL(request.url()).pathname;
    if (pathname === '/-/soda/repositories/7/settings/spaces') {await route.fulfill({body: html, contentType: 'text/html', headers: {'Content-Security-Policy': csp}}); return;}
    const env = {id, repository_id: '7', provisioned: true, profile: legacy ? null : profile};
    if (pathname.startsWith('/-/soda/api/')) {
      if (!pathname.endsWith('/session')) assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      if (request.method() === 'POST') {
        assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
        assert.equal(pathname, '/-/soda/api/environments');
        const body: unknown = request.postDataJSON(); writes.push(body);
        assert.deepEqual(body, {repository_id: '7', profile_id: 'rocky-headless'}); exists = true;
        await route.fulfill({status: 201, json: env}); return;
      }
      if (pathname.endsWith('/profiles') && !available) {await route.fulfill({status: 503, json: {error: {code: 'profile_unavailable'}}}); return;}
      const body = pathname.endsWith('/session') ? {user: {id: '1', login: 'alice'}, csrf_token: 'csrf-alice', forgejo_url: origin}
        : pathname.endsWith('/forgejo/me') ? {id: '1'}
          : pathname.endsWith('/profiles') ? {items: [profile]}
            : pathname.endsWith('/environments') ? {repository: {id: '7', owner: 'current', name: 'renamed'}, can_create: !exists, items: exists ? [env] : []}
              : pathname.endsWith('/development-keys') ? {items: []}
                : {environment: env, observed: {id, running: true}, login: '', environment_administrator: false, native_unavailable: false, authority_unavailable: false};
      await route.fulfill({json: body}); return;
    }
    const source = files['public' + pathname];
    if (source) {
      const file = source.startsWith('@build/forgejo-js/') ? path.resolve('.artifacts/forgejo-js', path.basename(source)) : path.resolve(source);
      await route.fulfill({body: Buffer.from(await Bun.file(file).arrayBuffer()), contentType: pathname.endsWith('.js') ? 'text/javascript' : pathname.endsWith('.css') ? 'text/css' : 'application/octet-stream'}); return;
    }
    await route.fulfill({status: 404});
  });
  await page.goto(origin + '/-/soda/repositories/7/settings/spaces');
  await page.getByRole('combobox', {name: 'Project OS'}).waitFor();
  assert.equal(await page.getByRole('combobox').locator('option').count(), 1);
  await page.getByRole('combobox').selectOption('rocky-headless'); assert.equal(writes.length, 0);
  await page.getByRole('button', {name: 'Create environment', exact: true}).click();
  await page.getByRole('button', {name: 'Join environment', exact: true}).waitFor();
  assert.equal(writes.length, 1); assert.equal(await page.getByRole('combobox').count(), 0);
  await page.getByText('Immutable creation identity').click(); await page.getByText(profile.image, {exact: true}).waitFor();
  legacy = true; await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.getByText(/Legacy \/ unknown creation profile/).waitFor();
  assert.equal(await page.getByRole('combobox').count(), 0);
  exists = false; available = false; await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.getByText(/Installed Project OS unavailable or incompatible/).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).isVisible(), false);
  assert.equal(writes.length, 1); assert.deepEqual(errors, []);
});

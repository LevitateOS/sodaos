import assert from 'node:assert/strict';
import test from 'node:test';
import {chromium} from 'playwright';
import {creationProfile} from '../../frontend/spaces/sodaspaces-api';

const profile = {id: 'rocky-headless', distribution: 'rocky', version: '10.2', interface: 'headless', architecture: 'amd64', image: 'sha256:' + 'a'.repeat(64), revision: 'b'.repeat(40)};
test('creation identity rejects unsupported and incomplete metadata', () => {
  assert.deepEqual(creationProfile(profile), profile);
  for (const bad of [{...profile, id: 'fedora-kde'}, {...profile, revision: ''}, {...profile, image: 'latest'}, {...profile, architecture: 'x86'}, {...profile, version: '<script>'}]) assert.throws(() => creationProfile(bad));
});
test('native repository settings mounts shared creation/access controls and preserves immutable/legacy display', {skip: !process.env.SODA_PAGE_ORIGIN}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const page = await browser.newPage({ignoreHTTPSErrors: true, storageState: process.env.SODA_PAGE_STATE || ''}), errors: string[] = []; page.on('pageerror', e => errors.push(e.message));
  const origin = process.env.SODA_PAGE_ORIGIN || '', actor = process.env.SODA_PAGE_ACTOR || '', id = 'p0123456789abcdef01234567';
  let exists = false, legacy = false, available = true, unsafeName = false, canCreate = true;
  let repositoryStatus = 200, currentActor = actor;
  const writes: unknown[] = [];
  await page.route(origin + '/**', async route => {
    const request = route.request(), pathname = new URL(request.url()).pathname;
    const env = {id, repository_id: '7', provisioned: true, profile: legacy ? null : profile};
    if (pathname.startsWith('/-/soda/api/')) {
      if (!pathname.endsWith('/session')) assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      if (request.method() === 'POST') {
        assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
        assert.equal(pathname, '/-/soda/api/environments');
        const body: unknown = request.postDataJSON(); writes.push(body);
        assert.deepEqual(body, {repository_id: '7', profile_id: 'rocky-headless'}); exists = true;
        await route.fulfill({status: 201, json: env}); return;
      }
      if (pathname.endsWith('/environments') && repositoryStatus !== 200) {
        await route.fulfill({status: repositoryStatus, json: {error: {code: repositoryStatus === 503 ? 'provider_unavailable' : 'repository_not_found', message: 'Repository unavailable'}}}); return;
      }
      if (pathname.endsWith('/profiles') && !available) {await route.fulfill({status: 503, json: {error: {code: 'profile_unavailable'}}}); return;}
      const body = pathname.endsWith('/session') ? {user: {id: currentActor, login: 'alice'}, csrf_token: 'csrf-alice', forgejo_url: origin}
        : pathname.endsWith('/forgejo/me') ? {id: actor}
          : pathname.endsWith('/profiles') ? {items: [profile]}
            : pathname.endsWith('/environments') ? {repository: {id: '7', owner: 'current', name: unsafeName ? '../foreign' : 'renamed'}, can_create: canCreate && !exists, items: exists ? [env] : []}
              : pathname.endsWith('/development-keys') ? {items: []}
                : {environment: env, observed: {id, running: true}, login: '', environment_administrator: false, native_unavailable: false, authority_unavailable: false};
      await route.fulfill({json: body}); return;
    }
    await route.continue();
  });
  await page.goto(origin + '/?soda-view=repository-spaces&repository_id=7');
  await page.getByRole('combobox', {name: 'Project OS'}).waitFor();
  assert.equal(await page.getByRole('combobox').locator('option').count(), 1);
  assert.equal(await page.locator('soda-project-controls').getByRole('link', {name: 'Native repository settings', exact: true}).getAttribute('href'), origin + '/current/renamed/settings');
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
  unsafeName = true; await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.waitForFunction(() => !document.querySelector('[data-project-controls][aria-busy=true]'));
  assert.equal(await page.locator('soda-project-controls').getByRole('link', {name: 'Native repository settings', exact: true}).count(), 0);
  // A visible member repository is not owner/create authority. Each denied
  // observation must clear the previous repository metadata, not keep stale links.
  unsafeName = false; available = true; canCreate = false;
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.waitForFunction(() => !document.querySelector('[data-project-controls][aria-busy=true]'));
  assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).count(), 0);
  for (const status of [403, 404, 503]) {
    repositoryStatus = status;
    await page.getByRole('button', {name: 'Refresh status'}).click();
    await page.waitForFunction(() => !document.querySelector('[data-project-controls][aria-busy=true]'));
    assert.equal(await page.locator('soda-project-controls').getByRole('link', {name: 'Native repository settings', exact: true}).count(), 0);
    assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).count(), 0);
  }
  repositoryStatus = 200; currentActor = actor === '999' ? '998' : '999';
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.waitForFunction(() => !document.querySelector('[data-project-controls][aria-busy=true]'));
  assert.equal(await page.getByRole('button', {name: 'Create environment', exact: true}).count(), 0);
  assert.equal(writes.length, 1); assert.deepEqual(errors, []);
});

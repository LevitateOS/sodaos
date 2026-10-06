import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh, click, writes} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

test('saved-key removal truthfully does not dispatch a native apply', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Remove saved key');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert.equal(sent[0]?.method, 'DELETE');
  assert(!sent[0]?.url.endsWith('/access-keys'));
  assert.match(await page.locator('[data-control=result]').innerText(), /Existing project SSH access is unchanged/);
});
test('review then explicit Apply confirms last-key removal', async (t) => {
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
test('an unconfirmed key apply requires another target preview, not permanent page lockout', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Review this project’s SSH keys');
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (!call.url.endsWith('/access-keys')) return null;
      return call.method === 'POST'
        ? new Response(null, {status: 502})
        : Response.json({login: 'alice', revision: 'b'.repeat(64), installed_fingerprints: [], saved_fingerprints: []});
    })
  );
  await click(page, 'Apply reviewed saved keys to this project');
  assert.match(
    await page.locator('[data-control=result]').innerText(),
    /We couldn’t confirm that this change finished/
  );
  assert(await page.evaluate(() => window.drawerFixture.api.canRestore));
  assert.equal(
    await page.getByRole('button', {name: 'Apply reviewed saved keys to this project', exact: true}).count(),
    0
  );
  await click(page, 'Review this project’s SSH keys');
  assert.equal((await writes(page)).length, 1);
  assert.match(
    await page.locator('[data-control=result]').innerText(),
    /We couldn’t confirm that this change finished/
  );
});
test('own Forgejo key selection is explicit and does not install or silently save a profile key', async (t) => {
  const page = await fixture(t, {saved: []});
  await refresh(page);
  await page.evaluate(() =>
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/me/forgejo-keys?page=1'))
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
      if (call.method === 'POST' && call.url.endsWith('/api/me/development-keys'))
        return Response.json({
          items: [{id: '1', fingerprint: 'SHA256:' + 'A'.repeat(43), public_key: 'ssh-ed25519 YWJj\n'}],
        });
      return null;
    })
  );
  await click(page, 'Review my Forgejo public keys');
  assert.deepEqual(await writes(page), []);
  assert.equal(await page.locator('soda-project-controls script').count(), 0);
  await click(page, 'Select for review');
  assert.equal(await page.locator('textarea').inputValue(), 'ssh-ed25519 YWJj\n');
  assert.deepEqual(await writes(page), []);
  await click(page, 'Save public key');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert.equal(sent[0]?.url, '/-/extensions/pages/soda/spaces/api/me/development-keys');
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {public_key: 'ssh-ed25519 YWJj'});
});
test('private-key paste is refused before request dispatch', async (t) => {
  const page = await fixture(t);
  await refresh(page);
  await click(page, 'Access');
  await page.locator('textarea').fill('-----BEGIN OPENSSH ' + 'PRIVATE KEY-----');
  await click(page, 'Save public key');
  assert.deepEqual(await writes(page), []);
  assert.match(await page.locator('[data-control=result]').innerText(), /Never upload a private key/);
});
test('nonadministrator has no lifecycle controls; nonmember joins separately', async (t) => {
  const page = await fixture(t, {admin: false, member: false});
  await refresh(page);
  assert.equal(await page.evaluate(() => window.drawerFixture.button('Start').closest('fieldset')?.hidden), true);
  await click(page, 'Join environment');
  const sent = await writes(page);
  assert.equal(sent.length, 1);
  assert(sent[0]?.url.endsWith('/join'));
});
test('browser-only Join is available without a public key and never imports saved keys implicitly', async (t) => {
  for (const saved of [[], ['SHA256:' + 'A'.repeat(43)]]) {
    const page = await fixture(t, {admin: false, member: false, saved});
    await refresh(page);
    assert(await page.getByRole('button', {name: 'Join environment', exact: true}).isVisible());
    await click(page, 'Join environment');
    const sent = await writes(page);
    assert.equal(sent.length, 1);
    assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {ssh_keys: 'none'});
  }
});
test('external SSH at Join requires explicit saved-key selection', async (t) => {
  const page = await fixture(t, {admin: false, member: false});
  await refresh(page);
  await page.getByLabel('Also install my saved public keys for external SSH').check();
  await click(page, 'Join environment');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {ssh_keys: 'saved'});
});

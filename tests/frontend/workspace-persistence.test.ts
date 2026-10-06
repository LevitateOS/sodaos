import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, create} from './fixtures/workspace-driver';
import {parseLayout, focusedPane} from '../../frontend/spaces/sodaspaces-layout';
setupWorkspaceDriver();

test('legacy pending and unknown IDs never select/create a session', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    sessionStorage.setItem('soda-terminal:1:p' + '1'.repeat(24), 'pending');
    sessionStorage.setItem('soda-terminal:1:p' + '2'.repeat(24), 'd'.repeat(32));
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await page.locator('soda-terminal').count(), 0);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-terminal:1:p' + '1'.repeat(24))), 'pending');
});
test('obsolete caches are ignored; native inventory still permits deliberate discovery', async (t) => {
  const page = await fixture(t);
  const original = await page.evaluate(() => {
    const text = JSON.stringify({
      version: 1,
      entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32)}, {requestId: 'd'.repeat(32)}],
    });
    sessionStorage.setItem('soda-spaces:v1:1', text);
    sessionStorage.setItem('soda-spaces:v2:1', text);
    return text;
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.locator('soda-terminal').count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await openSession(page, 'Build');
  const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  assert.equal(saved.entries.length, 1);
  assert.equal(saved.entries[0]?.locator.kind, 'existing');
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v1:1')), original);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), original);
});
test('current arrangement restores the exact selected ID and owner key, never a replacement', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  await openSession(page, 'Edit');
  const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  const selected = focusedPane(saved).selected;
  assert(selected);
  await page.reload();
  await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async () => {
    window.workspaceFixture = window.createWorkspaceFixture('native');
    await window.workspaceFixture.api.ready;
    await window.workspaceFixture.api.refresh();
  });
  await page.locator('.is-connected').waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal').getAttribute('id'), 'soda-owner-' + selected);
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map((s) => s.sent[0]?.id)), [
    'b'.repeat(32),
  ]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
});
for (const raw of ['{', JSON.stringify({version: 2}), ' '.repeat(32769)])
  test('corrupt/obsolete current cache resets without native effects: ' + raw.length, async (t) => {
    const page = await fixture(t);
    await page.evaluate((raw) => sessionStorage.setItem('soda-spaces:v3:1', raw), raw);
    await page.evaluate(() => window.workspaceFixture.api.refresh());
    assert.equal(await page.locator('soda-terminal').count(), 0);
    const empty = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
    assert.equal(empty.entries.length, 0);
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter((c) => c.method !== 'GET').length), 0);
    await create(page);
    const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
    assert.equal(saved.entries.length, 1);
    assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  });
test('storage write failure does not disable the live workspace', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const write = Storage.prototype.setItem;
    Storage.prototype.setItem = function (key, value) {
      if (key.startsWith('soda-spaces:v3:')) throw new DOMException('Quota exceeded', 'QuotaExceededError');
      write.call(this, key, value);
    };
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build');
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1')), null);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  await page.getByText('Live workspace remains usable', {exact: false}).waitFor();
});

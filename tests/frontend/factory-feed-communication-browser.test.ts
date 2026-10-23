import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh} from './fixtures/project-controls-driver';

setupProjectControlsDriver();

test('an MCP question renders its content once and answer choices edit a reply without submitting it', async (t) => {
  const page = await fixture(t), postId = 'f'.repeat(32);
  await page.evaluate((postId) => {
    window.drawerFixture.api.setPresentation('settings');
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false, project: {id: `p${'1'.repeat(24)}`, provisioned: true}}], next_cursor: ''});
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({entries: [{kind: 'post', id: postId, repository: 7, seq: 42, actor: 'architect', author: 'Architect', project: 'soda-tester/demo', status: 'ask', activity: 'ask', payload: {communication: {kind: 'ask', component_id: 'S02', title: 'Who can reassign tasks?', summary: 'The reassignment permission changes the access plan.', question: 'Should HR be allowed to reassign?', options: ['Allow HR to reassign', 'Keep the original owner']}}}], truncated: false});
      return null;
    });
  }, postId);
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Who can reassign tasks?'}).waitFor();
  assert.equal(await feed.getByText('The reassignment permission changes the access plan.', {exact: true}).count(), 1);
  await feed.getByRole('button', {name: 'Allow HR to reassign', exact: true}).focus();
  await page.keyboard.press('Enter');
  const draft = feed.locator(`textarea[data-feed-draft="post:${postId}"]`);
  assert.equal(await draft.inputValue(), 'Allow HR to reassign');
  assert.equal(await draft.isVisible(), true);
  assert.equal(await draft.evaluate((node) => document.activeElement === node), true);
  assert.equal(await feed.locator(`[data-comment-submit="post:${postId}"]`).isEnabled(), true);
  const writes = await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET'));
  assert.equal(writes.length, 0);
  assert.equal(await feed.getByText('Loading discussion text…').count(), 0);
  const bodyReads = await page.evaluate((postId) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${postId}`)), postId);
  assert.equal(bodyReads.length, 0);
});

test('findings show supplied source literally and corrections retain before and after', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription')) return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}], next_cursor: ''});
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [
        {kind: 'post', id: 'a'.repeat(32), repository: 7, seq: 43, actor: 'fixer', author: 'Fixer', payload: {communication: {kind: 'partial', component_id: 'FX04', title: 'Ownership corrected', summary: 'Compare the person making the request.', source: {path: 'router.ts', start_line: 12, code: 'return !!task.ownerId;', after: 'return task.ownerId === staff.id;'}, commits: [{revision: 'b'.repeat(40), summary: 'Restrict access to owner'}]}}},
        {kind: 'post', id: 'b'.repeat(32), repository: 7, seq: 42, actor: 'inspector', author: 'Inspector', payload: {communication: {kind: 'flag', component_id: 'IN03', title: 'Ownership defect', summary: 'Another employee can open the task.', source: {path: 'router.ts', start_line: 12, code: '<script>window.untrustedSourceRan = true</script>\nreturn !!task.ownerId;', highlight_lines: [13]}, items: [{label: 'Other employee access', state: 'not passed'}]}}}
      ], truncated: false});
      return null;
    });
  });
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Ownership defect'}).waitFor();
  assert.equal(await feed.locator('[data-action="finding"] .soda-action-source script').count(), 0);
  assert.match(await feed.locator('[data-action="finding"] .soda-action-source code').innerText(), /<script>window.untrustedSourceRan/);
  assert.equal(await feed.locator('[data-change="flagged"] > span').last().innerText(), 'return !!task.ownerId;');
  assert.equal(await feed.locator('[data-action="finding"] [data-tone="positive"]').count(), 0);
  assert.equal(await feed.locator('[data-action="correction"] pre').count(), 2);
  assert.equal(await feed.locator('.soda-action-commits code').innerText(), 'bbbbbbbb');
  assert.equal(await feed.locator('.soda-action-commits li > span').innerText(), 'Restrict access to owner');
});

test('recorded suggestion and answer resolutions remain readable in the feed', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription')) return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}], next_cursor: ''});
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [
        {kind: 'post', id: 'a'.repeat(32), repository: 7, seq: 45, actor: 'architect', payload: {communication: {kind: 'suggest', title: 'Cache the policy', summary: 'Avoid repeated reads.', disposition: 'applied', disposition_reason: 'Cached only within the assignment.'}}},
        {kind: 'post', id: 'b'.repeat(32), repository: 7, seq: 44, actor: 'architect', payload: {communication: {kind: 'suggest', title: 'Share the cache', summary: 'Share reads between assignments.', disposition: 'refused', disposition_reason: '<script>window.untrustedResolutionRan = true</script> would cross assignment scope.'}}},
        {kind: 'post', id: 'c'.repeat(32), repository: 7, seq: 43, actor: 'architect', payload: {communication: {kind: 'ask', title: 'Who can read tasks?', summary: 'The access policy needs clarification.', question_state: 'incorporated'}}},
        {kind: 'post', id: 'd'.repeat(32), repository: 7, seq: 42, actor: 'builder', payload: {communication: {kind: 'answer', title: 'Only the owner', summary: 'The owner alone can read their tasks.', disposition: 'incorporated'}}},
        {kind: 'post', id: 'e'.repeat(32), repository: 7, seq: 41, actor: 'architect', payload: {communication: {kind: 'suggest', title: 'Consider a bounded cache', summary: 'Still awaiting a decision.'}}}
      ], truncated: false});
      return null;
    });
  });
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Cache the policy', exact: true}).waitFor();
  assert.equal(await feed.getByText('Suggestion applied', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Suggestion declined', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Answer incorporated', {exact: true}).count(), 2);
  assert.equal(await feed.getByText('Proposed', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Recorded detail unavailable', {exact: true}).count(), 0);
  assert.equal(await feed.locator('[data-post-resolution="applied"]').innerText(), 'Cached only within the assignment.');
  assert.match(await feed.locator('[data-post-resolution="refused"]').innerText(), /<script>window.untrustedResolutionRan/);
  assert.equal(await feed.locator('[data-post-resolution] script').count(), 0);
});

import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

const recordId = 'a'.repeat(32);
const postId = 'c'.repeat(32);
const commentId = 'd'.repeat(32);

test('every feed author loads its own role portrait, while human and unknown authors keep their initials', async (t) => {
  const page = await fixture(t);
  const actors: [string, string][] = [
    ['product_lead', 'product-lead'],
    ['architect', 'architect'],
    ['builder', 'builder'],
    ['inspector', 'inspector'],
    ['fixer', 'fixer'],
    ['strategist', 'strategist'],
    ['test_pilot', 'test-pilot'],
    ['library_admin', 'library-admin'],
    ['jev', 'jev'],
    ['factory', 'factory'],
    ['System One', 'jev'],
    ['systemone', 'jev'],
    ['system', 'factory'],
    ['17', ''],
    ['unknown_role', ''],
  ];
  await page.evaluate((actors) => {
    window.drawerFixture.api.setPresentation('settings');
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}], next_cursor: ''});
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries: actors.map(([actor], index) => ({
            kind: actor === '17' ? 'post' : 'record',
            id: (index + 1).toString(16).padStart(32, '0'),
            repository: 7,
            seq: 100 - index,
            actor,
            author: actor === '17' ? 'Architect' : 'Builder',
            payload: {communication: {kind: 'flag', title: 'Portrait identity', summary: 'The author role owns this portrait.'}},
          })),
          truncated: false,
        });
      return null;
    });
  }, actors);
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Portrait identity', exact: true}).first().waitFor();
  assert.equal(await feed.locator('.soda-feed-post').count(), actors.length);
  for (const [index, [actor, file]] of actors.entries()) {
    const kind = actor === '17' ? 'post' : 'record';
    const card = feed.locator(`#feed-${kind}-${(index + 1).toString(16).padStart(32, '0')}`);
    const image = card.locator('.soda-feed-avatar img');
    if (!file) {
      assert.equal(await image.count(), 0, actor);
      assert.equal(await card.locator('.soda-feed-avatar').textContent(), actor === '17' ? 'A' : 'B');
      continue;
    }
    assert.equal(await card.getAttribute('data-role'), file.replaceAll('-', '_'));
    assert.equal(await image.getAttribute('src'), `/assets/role-portraits/ps1-v1/${file}.png`);
    assert.equal(await image.getAttribute('alt'), '');
    await image.scrollIntoViewIfNeeded();
    const size = await image.evaluate(async (node) => {
      const img = node as HTMLImageElement;
      await img.decode();
      return [img.naturalWidth, img.naturalHeight];
    });
    assert.deepEqual(size, [1254, 1254], actor);
  }
});

test('Project feed reads newest-first with separate enriched posts and live comment outcomes', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await page.evaluate(
    async ({recordId, postId, commentId}) => {
      const state = {feed: 0, comments: [] as string[]};
      (window as unknown as {feedProbeState?: unknown}).feedProbeState = state;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription') && call.method === 'GET')
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q=') && call.method === 'GET')
          return Response.json({
            items: [
              {
                id: '7',
                owner: 'soda-tester',
                name: 'demo',
                can_create: false,
                project: {id: `p${'1'.repeat(24)}`, provisioned: true},
              },
            ],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?') && call.method === 'GET') {
          state.feed++;
          if (call.url.includes('cursor=')) {
            return Response.json({
              entries: [{kind: 'record', id: 'e'.repeat(32), repository: 7, seq: 30}],
              truncated: false,
            });
          }
          return Response.json({
            entries: [
              {kind: 'record', id: recordId, repository: 7, seq: 42, issue: 3},
              {
                kind: 'post',
                id: postId,
                repository: 7,
                seq: 41,
                actor: 'Product Lead',
                author: 'Ada',
                project: 'soda-tester/demo',
                at: 1720000000,
                status: 'assessing',
              },
            ],
            next: 'v1:41',
            truncated: true,
          });
        }
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST') {
          const body = JSON.parse((call as unknown as {body?: string}).body ?? '{}') as Record<string, unknown>;
          state.comments.push(typeof body.body === 'string' ? body.body : '');
          return Response.json({
            comment_id: commentId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${postId}`) && call.method === 'GET')
          return Response.json({
            comment_id: postId,
            revision: 0,
            repository: 7,
            author: '1',
            author_name: 'Ada',
            at: 1720000000,
            edited: false,
            deleted: false,
            revoked: false,
            body: 'Should the demo cover substitute approvers?',
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${commentId}`) && call.method === 'GET')
          return Response.json({
            comment_id: commentId,
            revision: 0,
            repository: 7,
            author: '1',
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            deleted: false,
            revoked: false,
            body: 'Please add substitute approvers.',
            intake: 'recorded',
            judgment: 'applied',
            route: 'new_issue',
            draft: 'applied',
            effect: 'verified',
            issue: 12,
            source: {
              kind: 'record',
              id: recordId,
              issue: 3,
              status: 'implement:finished:succeeded',
              summary: 'Built the form service.',
            },
          });
        return null;
      });
    },
    {recordId, postId, commentId}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  // Separate posts with the outside-card author context and stored text.
  await feed.getByText('Recorded work').first().waitFor();
  await feed.getByText('Ada').first().waitFor();
  await feed.getByText('Product Lead').first().waitFor();
  await feed.getByText('Assessing').first().waitFor();
  await feed.getByText('Should the demo cover substitute approvers?').waitFor();
  // Cursor history: older entries append behind the confirmed page.
  await feed.getByRole('button', {name: 'Load older entries'}).click();
  await feed.getByRole('button', {name: 'Back to latest'}).waitFor();
  // Drafts survive refreshes with reading position intact.
  const draft = feed.locator(`textarea[data-feed-draft="record:${recordId}"]`);
  await feed.locator(`[data-feed-drawer-toggle="record:${recordId}"]`).click();
  await draft.fill('half-written reply');
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.').waitFor();
  assert.equal(await draft.inputValue(), 'half-written reply');
  // Submit persists through the real intake shape and exposes progress.
  await feed.locator(`[data-comment-submit="record:${recordId}"]`).click();
  await feed.getByText('Comment saved.').waitFor();
  const posted = await page.evaluate(() => {
    const drawer = window.drawerFixture as unknown as {calls: {url: string; method: string; body?: string}[]};
    return drawer.calls
      .filter((call) => call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST')
      .map((call) => JSON.parse(call.body ?? '{}'));
  });
  assert.equal(posted.length, 1);
  assert.equal(posted[0]?.repository, 7);
  // Commenting from a work card carries its authorized bound source issue
  // and names the source entry for server-side resolution.
  assert.equal(posted[0]?.issue, 3);
  assert.equal(posted[0]?.source_kind, 'record');
  assert.equal(posted[0]?.source_id, recordId);
  assert.equal(posted[0]?.body, 'half-written reply');
  assert.match(
    typeof posted[0]?.idempotency_key === 'string' ? (posted[0]?.idempotency_key as string) : '',
    /^[A-Za-z0-9_-]{1,64}$/
  );
  assert.equal('author' in (posted[0] ?? {}), false);
  // The refresh after saving re-reads progress without another submission.
  const issueLink = feed.getByRole('link', {name: 'Open linked issue'});
  await issueLink.waitFor();
  assert.equal(await issueLink.getAttribute('href'), '/soda-tester/demo/issues/12');
  await feed
    .locator('.soda-feed-outcome details')
    .evaluateAll((nodes) => nodes.forEach((node) => ((node as HTMLDetailsElement).open = true)));
  // Source details stay available on demand.
  await feed.getByText(/About recorded work aaaaaaaaaaaa… · issue #3/).waitFor();
  await feed
    .locator('.soda-feed-outcome-source')
    .getByText(/Built the form service\./)
    .waitFor();
});

test('Factory-loop families render their own sections and failures stay in front', async (t) => {
  const page = await fixture(t);
  page.setDefaultTimeout(5000);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    (window as unknown as {feedArgvVariant: string}).feedArgvVariant = 'valid';
  });
  await page.evaluate(async () => {
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription') && call.method === 'GET')
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q=') && call.method === 'GET')
        return Response.json({
          items: [
            {
              id: '7',
              owner: 'soda-tester',
              name: 'demo',
              can_create: false,
              project: {id: `p${'1'.repeat(24)}`, provisioned: true},
            },
          ],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?') && call.method === 'GET')
        return Response.json({
          entries: [
            {
              kind: 'record',
              id: '1'.repeat(32),
              repository: 7,
              seq: 60,
              actor: 'factory',
              status: 'merged',
              at: 1720000060,
              issue: 3,
              pr: 42,
              ref: 'c'.repeat(40),
              summary: 'Built <candidate> & merged it.',
            },
            {
              kind: 'record',
              id: '2'.repeat(32),
              repository: 7,
              seq: 59,
              actor: 'test_pilot',
              status: 'verified',
              at: 1720000059,
            },
            {
              kind: 'record',
              id: '3'.repeat(32),
              repository: 7,
              seq: 58,
              actor: 'fixer',
              status: 'submitted',
              at: 1720000058,
            },
            {
              kind: 'record',
              id: '4'.repeat(32),
              repository: 7,
              seq: 57,
              actor: 'inspector',
              status: 'findings_submitted',
              at: 1720000057,
            },
            {
              kind: 'record',
              id: '5'.repeat(32),
              repository: 7,
              seq: 56,
              actor: 'builder',
              status: 'check_failed',
              at: 1720000056,
            },
            {
              kind: 'record',
              id: '6'.repeat(32),
              repository: 7,
              seq: 55,
              actor: 'strategist',
              status: 'proposed',
              at: 1720000055,
            },
            {
              kind: 'record',
              id: '7'.repeat(32),
              repository: 7,
              seq: 54,
              actor: 'architect',
              status: 'submitted',
              at: 1720000054,
              payload: {plan: {
                approach: 'Use the recorded direct argv without treating it as a summary.',
                acceptance_coverage: ['docs'],
                checks: [{id: 'canonical-check', required: true, scope: 'integration', argv: (() => {
                  const variant = (window as unknown as {feedArgvVariant: string}).feedArgvVariant;
                  (window as unknown as {feedArgvRead: string}).feedArgvRead = variant;
                  if (variant === 'aggregate') return ['x'.repeat(8192), 'y'.repeat(8193)];
                  if (variant === 'unicode') return ['😀'.repeat(4095), 'abcde'];
                  if (variant === 'count') return Array.from({length: 65}, () => 'a');
                  if (variant === 'nul') return ['node', 'bad\0argument'];
                  return ['node', '-e', '<candidate>' + 'x'.repeat(1240)];
                })()}],
              }},
            },
            {
              kind: 'record',
              id: '8'.repeat(32),
              repository: 7,
              seq: 53,
              actor: 'product_lead',
              status: 'submitted',
              at: 1720000053,
            },
            {
              kind: 'record',
              id: '9'.repeat(32),
              repository: 7,
              seq: 52,
              actor: 'jev',
              status: 'recorded',
              at: 1720000052,
            },
            {
              kind: 'record',
              id: 'a'.repeat(32),
              repository: 7,
              seq: 51,
              actor: 'library_admin',
              status: 'recorded',
              at: 1720000051,
            },
            {
              kind: 'record',
              id: 'b'.repeat(32),
              repository: 7,
              seq: 50,
              actor: 'mystery_actor',
              status: 'frobnicate',
              at: 1720000050,
            },
            {
              kind: 'record', id: 'c'.repeat(32), repository: 7, seq: 49,
              actor: 'researcher', author: 'Researcher', status: 'research:finished:succeeded',
              at: 1720000049, summary: 'Answered the bound repository inquiry.',
            },
          ],
          truncated: false,
        });
      return null;
    });
  });
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  for (const family of [
    'integration',
    'verification',
    'correction',
    'review',
    'work',
    'reassessment',
    'issue',
    'plan',
    'decision',
    'knowledge',
    'fallback',
  ])
    assert.equal(await feed.locator(`[data-feed-family="${family}"]`).count(), family === 'knowledge' ? 2 : 1, family);
  // Readable role authors, not canonical ids, in the outside-card headers.
  await feed.getByText('Test Pilot').first().waitFor();
  await feed.getByText('Product Lead').first().waitFor();
  await feed.getByText('Library Admin').first().waitFor();
  const researcher = feed.locator('#feed-record-' + 'c'.repeat(32));
  assert.equal(await researcher.locator('[data-feed-family="knowledge"]').count(), 1);
  await researcher.locator('.soda-feed-header').getByText('Researcher', {exact: true}).waitFor();
  assert.equal(await researcher.locator('.soda-feed-header').getByText('Inspector', {exact: true}).count(), 0);
  // The failed check leads in its headline/status; the unknown status never reads done.
  await feed.locator('details').evaluateAll((nodes) => {
    for (const node of nodes) (node as HTMLDetailsElement).open = true;
  });
  assert.equal(await feed.getByText('This work needs attention. See the result above.', {exact: true}).count(), 0);
  await feed.getByText(/frobnicate is not interpreted yet/).waitFor();
  // Family anatomy: what it covers plus recorded state, with source links.
  await feed.getByText('Application verification completed', {exact: true}).waitFor();
  await feed.locator('[data-feed-family="work"]').getByText('Implementation work needs attention', {exact: true}).waitFor();
  await feed.getByText(/Recorded status: check_failed/).waitFor();
  assert.equal(await feed.locator('[data-feed-family="work"] a[href="/soda-tester/demo/pulls"]').count(), 1);
  assert.equal(await feed.locator('[data-feed-family="plan"] a[href="/soda-tester/demo/issues"]').count(), 1);
  // Enriched recorded content: bound records link natively by number,
  // the candidate reference and the recorded summary render as text.
  const integration = feed.locator('[data-feed-family="integration"]');
  const issueLink = integration.getByRole('link', {name: 'Open issue #3'});
  await issueLink.waitFor();
  assert.equal(await issueLink.getAttribute('href'), '/soda-tester/demo/issues/3');
  const pullLink = integration.getByRole('link', {name: 'Open pull request #42'});
  await pullLink.waitFor();
  assert.equal(await pullLink.getAttribute('href'), '/soda-tester/demo/pulls/42');
  await integration.getByText('Built <candidate> & merged it.').waitFor();
  await integration.locator('.soda-feed-rows').getByText('c'.repeat(40), {exact: true}).waitFor();
  // The summary is text, never markup.
  assert.equal(await feed.locator('candidate').count(), 0);
  const investigation = feed.locator('#feed-record-' + '7'.repeat(32));
  const command = investigation.locator('.soda-feed-checks tbody th').first();
  await command.waitFor();
  const preview = (await command.textContent()) ?? '';
  assert(preview.startsWith('canonical-check — node -e <candidate>'));
  assert(preview.endsWith('… (command preview)'));
  assert([...preview].length < 560, 'long accepted argv must have a bounded, labelled preview');
  assert.equal(await investigation.getByText('Recorded detail unavailable', {exact: true}).count(), 0);
  // The actual accepted 1,251-byte argument is valid; the canonical total,
  // argument-count and NUL fences still reject malformed records.
  for (const variant of ['aggregate', 'unicode', 'count', 'nul']) {
    await page.evaluate((variant) => {
      (window as unknown as {feedArgvVariant: string}).feedArgvVariant = variant;
    }, variant);
    await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
    await page.waitForFunction((variant) => {
      const current = document.querySelector('soda-factory-feed') as unknown as {reading: boolean; view: {busy: boolean}};
      return (window as unknown as {feedArgvRead: string}).feedArgvRead === variant && !current.reading && !current.view.busy;
    }, variant);
    await investigation.getByText('Recorded detail unavailable', {exact: true}).waitFor();
    assert.equal(await investigation.locator('.soda-feed-checks').count(), 0, variant);
  }
});

test('Designed blocks render recorded segments; durable continuation lands without resubmission', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await page.evaluate(
    async ({recordId, commentId}) => {
      const state = {reads: 0, posts: 0};
      (window as unknown as {feedDurableState?: unknown}).feedDurableState = state;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription') && call.method === 'GET')
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q=') && call.method === 'GET')
          return Response.json({
            items: [
              {
                id: '7',
                owner: 'soda-tester',
                name: 'demo',
                can_create: false,
                project: {id: `p${'1'.repeat(24)}`, provisioned: true},
              },
            ],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?') && call.method === 'GET')
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: recordId,
                repository: 7,
                seq: 70,
                actor: 'builder',
                status: 'implement:finished:succeeded>checks:6>published',
                issue: 3,
                pr: 42,
                ref: 'c'.repeat(40),
                summary: 'Form and permission service ready.',
              },
              {
                kind: 'record',
                id: 'f'.repeat(32),
                repository: 7,
                seq: 69,
                actor: 'inspector',
                status: 'review:finished:succeeded:request-changes>findings:2',
                summary: 'Cross-team read is possible.',
              },
              {
                kind: 'record',
                id: 'e'.repeat(32),
                repository: 7,
                seq: 68,
                actor: 'fixer',
                status: 'correct:finished:succeeded>findings:2',
                ref: 'd'.repeat(40),
              },
            ],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST') {
          state.posts++;
          return Response.json({
            comment_id: commentId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${commentId}`) && call.method === 'GET') {
          state.reads++;
          // Durable continuation: stored state advances server-side
          // between reads; the browser must pick it up by refreshing
          // the KNOWN pending row, never by resubmitting.
          if (state.reads < 2)
            return Response.json({
              comment_id: commentId,
              revision: 0,
              repository: 7,
              author_name: 'alice',
              at: 1720000000,
              edited: false,
              body: 'Please add substitute approvers.',
              intake: 'recorded',
              judgment: 'pending',
              draft: 'pending',
              effect: 'pending',
              issue: 0,
            });
          return Response.json({
            comment_id: commentId,
            revision: 0,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Please add substitute approvers.',
            intake: 'recorded',
            judgment: 'applied',
            route: 'existing_work',
            draft: 'pending',
            effect: 'verified',
            issue: 0,
            existing_issue: 9,
          });
        }
        return null;
      });
    },
    {recordId, commentId}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  // Designed Builder component: brief + check table + candidate rows.
  const work = feed.locator('[data-feed-family="work"]');
  await work.getByText('Form and permission service ready.').waitFor();
  assert.equal(await work.locator('.soda-feed-checks table').count(), 1);
  await work.getByText('6 receipts').waitFor();
  // Designed Inspector component: verdict + findings rows.
  const review = feed.locator('[data-feed-family="review"]');
  await review.getByText(/2 recorded findings/).waitFor();
  await review.getByText('request changes').waitFor();
  // Designed Fixer component: before/after panels.
  const correction = feed.locator('[data-feed-family="correction"]');
  assert.equal(await correction.locator('.soda-feed-beforeafter').count(), 1);
  await correction.getByText(/Submitted for independent acceptance/).waitFor();
  // Expandable record details are native, keyboard-operable <details>.
  assert.equal(await work.locator('details.soda-feed-record-details').count(), 1);
  // Submit once; the durable continuation result must appear on feed
  // refresh (known-pending re-read) with no second POST.
  const draft = feed.locator(`textarea[data-feed-draft="record:${recordId}"]`);
  await feed.locator(`[data-feed-drawer-toggle="record:${recordId}"]`).click();
  await draft.fill('Please add substitute approvers.');
  await feed.locator(`[data-comment-submit="record:${recordId}"]`).click();
  await feed.getByText(/Saved\. JEV is assessing/).waitFor();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  const linked = feed.getByRole('link', {name: 'Open linked issue'});
  await linked.waitFor();
  assert.equal(await linked.getAttribute('href'), '/soda-tester/demo/issues/9');
  await feed.getByText(/Feedback added to issue #9/).waitFor();
  const durable = await page.evaluate(
    () => (window as unknown as {feedDurableState?: {reads: number; posts: number}}).feedDurableState
  );
  assert.equal(durable?.posts, 1);
  assert.ok((durable?.reads ?? 0) >= 2);
});

test('a project comment works before the first worker post', async (t) => {
  const page = await fixture(t);
  await page.evaluate(
    ({commentId}) => {
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: [], revision: 1});
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST')
          return Response.json({
            comment_id: commentId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        if (call.url.endsWith('/api/factory/feed/comments/' + commentId))
          return Response.json({
            comment_id: commentId,
            revision: 0,
            repository: 7,
            author: '1',
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            deleted: false,
            revoked: false,
            body: 'Thanks for the update.',
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {commentId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('No activity yet.').waitFor();
  await feed.getByLabel('Project comment', {exact: true}).fill('Thanks for the update.');
  await feed.getByRole('button', {name: 'Post comment', exact: true}).click();
  await feed.getByText('Kept as discussion.').waitFor();
  assert.equal(await feed.getByLabel('Project comment', {exact: true}).inputValue(), '');
  assert(await feed.getByRole('button', {name: 'Post comment', exact: true}).isDisabled());
  const posts = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((c) => c.url.endsWith('/api/factory/feed/comments') && c.method === 'POST')
      .map((c) => JSON.parse(c.body ?? '{}'))
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].repository, 7);
  assert.equal(posts[0].body, 'Thanks for the update.');
  assert.equal('source_id' in posts[0], false);
  assert.equal('reply_to' in posts[0], false);
});

test('Project switch during feed loading starts the new read and releases Refresh', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const feed = document.createElement('soda-factory-feed') as HTMLElement & {
      context: unknown;
      updateComplete: Promise<unknown>;
    };
    feed.dataset.feedRace = 'true';
    document.body.append(feed);
    let count = 0;
    let release: (response: Response) => void = () => {};
    const emptyFeed = () => Response.json({entries: [], truncated: false});
    const transport = {
      request: async (path: string) => {
        if (path.startsWith('factory/feed?')) {
          count++;
          if (count === 1)
            return new Promise<Response>((resolve) => {
              release = resolve;
            });
          return emptyFeed();
        }
        if (path === 'repositories?q=') return Response.json({items: [], next_cursor: ''});
        throw Error('Unexpected feed read');
      },
    };
    Object.assign(feed, {releaseFirst: () => release(emptyFeed()), readCount: () => count});
    feed.context = {transport, repository: '7'};
    await feed.updateComplete;
    while (count === 0) await new Promise((resolve) => setTimeout(resolve, 0));
    feed.context = {transport, repository: '8'};
  });
  const feed = page.locator('soda-factory-feed[data-feed-race="true"]');
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).waitFor();
  await page.waitForFunction(
    () => !document.querySelector<HTMLButtonElement>('soda-factory-feed[data-feed-race="true"] button')?.disabled
  );
  await feed.evaluate((el) => (el as HTMLElement & {releaseFirst(): void}).releaseFirst());
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  assert.ok(await feed.evaluate((el) => (el as HTMLElement & {readCount(): number}).readCount()) >= 3);
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
});

test('an uncertain comment retry resends the original payload and keeps newer writing', async (t) => {
  const page = await fixture(t);
  await page.evaluate(
    ({commentId}) => {
      let attempts = 0;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST') {
          attempts++;
          return attempts === 1
            ? Response.json({error: {code: 'reply_lost'}}, {status: 503})
            : Response.json({
                comment_id: commentId,
                revision: 0,
                intake: 'recorded',
                judgment: 'applied',
                route: 'discussion',
                draft: 'pending',
                effect: 'pending',
                issue: 0,
              });
        }
        if (call.url.endsWith('/api/factory/feed/comments/' + commentId))
          return Response.json({
            comment_id: commentId,
            revision: 0,
            body: 'Original request',
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {commentId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('No activity yet.').waitFor();
  const draft = feed.getByLabel('Project comment', {exact: true});
  await draft.fill('Original request');
  await feed.getByRole('button', {name: 'Post comment', exact: true}).click();
  await feed.locator('[data-feed-message]').filter({hasText: 'draft is kept'}).waitFor();
  await draft.fill('New writing after the uncertain reply');
  await feed.getByRole('button', {name: 'Post comment', exact: true}).click();
  await feed.getByText('Previous comment saved. Your newer edits are kept, ready to send.', {exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'New writing after the uncertain reply');
  const posts = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((call) => call.method === 'POST' && call.url.endsWith('/api/factory/feed/comments'))
      .map((call) => JSON.parse(call.body ?? '{}'))
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].body, 'Original request');
  assert.deepEqual(posts[1], posts[0], 'retry must not attach new writing to the original idempotency key');
});

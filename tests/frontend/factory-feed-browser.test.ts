import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh} from './fixtures/project-controls-driver';
setupProjectControlsDriver();

const recordId = 'a'.repeat(32);
const postId = 'c'.repeat(32);
const commentId = 'd'.repeat(32);

test('feed distinguishes a failed read from no matches and keeps drafts through recovery', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    document.body.dataset.feedUnavailable = 'true';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
      if (!call.url.includes('/api/factory/feed?')) return null;
      if (document.body.dataset.feedUnavailable === 'true') return Response.json({}, {status: 503});
      const filtered = new URL(call.url, location.origin).searchParams.get('view') === 'decisions';
      return Response.json({
        entries: filtered
          ? []
          : ['judgment:applied', 'draft:valid', 'effect:created', undefined].map((status, index) => ({
              kind: 'record',
              id: String(index + 1).repeat(32),
              repository: 7,
              seq: index + 1,
              ...(status ? {status} : {}),
            })),
        truncated: false,
      });
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity could not be loaded.', exact: true}).waitFor();
  assert.equal(await feed.getByText('No activity yet.', {exact: true}).count(), 0);
  const draft = feed.getByLabel('Project comment', {exact: true});
  await draft.fill('Keep this unsent note.');
  await page.evaluate(() => {
    document.body.dataset.feedUnavailable = 'false';
  });
  await feed.getByRole('button', {name: 'Retry loading activity', exact: true}).click();
  await feed.locator('.soda-feed-status').getByText('Decision recorded', {exact: true}).waitFor();
  await feed.locator('.soda-feed-status').getByText('Proposal ready', {exact: true}).waitFor();
  await feed.locator('.soda-feed-status').getByText('Work created', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-status').getByText('Completed', {exact: true}).count(), 0);
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  await feed.getByRole('button', {name: 'Decisions', exact: true}).click();
  await feed.getByRole('heading', {name: 'No matching posts.', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Clear filters', exact: true}).click();
  await feed.locator('.soda-feed-status').getByText('Decision recorded', {exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  await page.evaluate(() => {
    document.body.dataset.feedUnavailable = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Showing the last loaded activity. Refresh to check for updates.', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-post').count(), 4);
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a failed per-post body GET keeps its tombstone and Retry recovers the same text', async (t) => {
  const page = await fixture(t);
  const post = 'b'.repeat(32);
  await page.evaluate(
    ({post}) => {
      window.drawerFixture.api.setPresentation('settings');
      let reads = 0;
      (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [{kind: 'post', id: post, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000}],
            truncated: false,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${post}`) && call.method === 'GET') {
          reads++;
          if (reads === 1) return new Response(null, {status: 503});
          return Response.json({
            comment_id: post,
            revision: 0,
            repository: 7,
            author_name: 'Ada',
            at: 1720000000,
            edited: false,
            body: 'Recovered discussion text.',
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {post}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  const retry = feed.getByRole('button', {name: 'Retry loading discussion text', exact: true});
  await retry.waitFor();
  const draft = feed.getByLabel('Project comment', {exact: true});
  await draft.fill('Hold this unsent note.');
  await retry.click();
  await feed.getByText('Recovered discussion text.', {exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'Hold this unsent note.');
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    2
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

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
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries: actors.map(([actor], index) => ({
            kind: actor === '17' ? 'post' : 'record',
            id: (index + 1).toString(16).padStart(32, '0'),
            repository: 7,
            seq: 100 - index,
            actor,
            author: actor === '17' ? 'Architect' : 'Builder',
            payload: {
              communication: {kind: 'flag', title: 'Portrait identity', summary: 'The author role owns this portrait.'},
            },
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
  await feed.locator('.soda-feed-header').getByText('Product Lead').first().waitFor();
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
              payload: {
                plan: {
                  approach: 'Use the recorded direct argv without treating it as a summary.',
                  acceptance_coverage: ['docs'],
                  checks: [
                    {
                      id: 'canonical-check',
                      required: true,
                      scope: 'integration',
                      argv: (() => {
                        const variant = (window as unknown as {feedArgvVariant: string}).feedArgvVariant;
                        (window as unknown as {feedArgvRead: string}).feedArgvRead = variant;
                        if (variant === 'aggregate') return ['x'.repeat(8192), 'y'.repeat(8193)];
                        if (variant === 'unicode') return ['😀'.repeat(4095), 'abcde'];
                        if (variant === 'count') return Array.from({length: 65}, () => 'a');
                        if (variant === 'nul') return ['node', 'bad\0argument'];
                        return ['node', '-e', '<candidate>' + 'x'.repeat(1240)];
                      })(),
                    },
                  ],
                },
              },
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
              kind: 'record',
              id: 'c'.repeat(32),
              repository: 7,
              seq: 49,
              actor: 'researcher',
              author: 'Researcher',
              status: 'research:finished:succeeded',
              at: 1720000049,
              summary: 'Answered the bound repository inquiry.',
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
  await feed.locator('.soda-feed-header').getByText('Test Pilot').first().waitFor();
  await feed.locator('.soda-feed-header').getByText('Product Lead').first().waitFor();
  await feed.locator('.soda-feed-header').getByText('Library Admin').first().waitFor();
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
  await feed
    .locator('[data-feed-family="work"]')
    .getByText('Implementation work needs attention', {exact: true})
    .waitFor();
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
      const current = document.querySelector('soda-factory-feed') as unknown as {
        reading: boolean;
        view: {busy: boolean};
      };
      return (
        (window as unknown as {feedArgvRead: string}).feedArgvRead === variant && !current.reading && !current.view.busy
      );
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
  // A separately resumed new-work request retains six historical failures.
  // Its explicit permission must keep polling and show pending authoring.
  const resumedPage = await fixture(t);
  await resumedPage.evaluate(
    ({commentId}) => {
      window.drawerFixture.api.setPresentation('settings');
      const state = {reads: 0, released: false};
      (window as unknown as {resumeFeedState?: unknown}).resumeFeedState = state;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
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
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [{kind: 'post', id: commentId, repository: 7, seq: 71, actor: '1', status: 'draft:pending'}],
            truncated: false,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${commentId}`)) {
          state.reads++;
          return Response.json({
            comment_id: commentId,
            revision: 0,
            repository: 7,
            author: '1',
            author_name: 'alice',
            at: 1720000000,
            body: 'Please add substitute approvers.',
            intake: 'recorded',
            judgment: 'applied',
            route: 'new_issue',
            draft: state.released ? 'applied' : 'pending',
            effect: state.released ? 'verified' : 'pending',
            issue: state.released ? 10 : 0,
            attempts: 6,
            attempt_limit: 6,
            resume: !state.released,
            can_retry: false,
          });
        }
        return null;
      });
    },
    {commentId}
  );
  await refresh(resumedPage);
  await resumedPage.evaluate(() => window.drawerFixture.mountFeed());
  const resumedFeed = resumedPage.locator('soda-factory-feed');
  await resumedFeed.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await resumedFeed.locator(`[data-feed-drawer-toggle="post:${commentId}"]`).click();
  assert.match(await resumedFeed.innerText(), /Routed to new work\. Product Lead issue authoring pending\./);
  assert.equal(await resumedFeed.getByText(/stopped — attempts exhausted/).count(), 0);
  await resumedPage.evaluate(() => {
    (window as unknown as {resumeFeedState: {released: boolean}}).resumeFeedState.released = true;
  });
  await resumedFeed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  assert.match(await resumedFeed.innerText(), /Issue #10 created\./);
  assert.equal(
    await resumedPage.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length),
    0
  );
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
  assert.ok((await feed.evaluate((el) => (el as HTMLElement & {readCount(): number}).readCount())) >= 3);
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

test('clarification answer draft survives reopen with zero writes and clears only on submit', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const clarifyId = 'd'.repeat(32);
  await page.evaluate(
    ({record, clarifyId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: clarifyId, key: 'answer-key-1'}])
      );
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${clarifyId}`) && call.method === 'GET')
          return Response.json({
            comment_id: clarifyId,
            revision: 3,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Original request text',
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'clarify',
            detail: 'What should change, and where should it apply?',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${clarifyId}/retry`) && call.method === 'POST')
          return Response.json({ok: true});
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, clarifyId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).click();
  const answer = feed.locator('[data-revision-body]');
  await answer.waitFor();
  await answer.fill('My saved answer');
  const draftKey = `soda-feed-revision-draft:v1:fixture-generation:7:${clarifyId}:3`;
  assert.deepEqual(
    await page.evaluate((key) => {
      const parsed = JSON.parse(window.localStorage.getItem(key) ?? 'null');
      return {body: parsed.body, original: parsed.original, revision: parsed.revision, savedAt: typeof parsed.savedAt};
    }, draftKey),
    {body: 'My saved answer', original: 'Original request text', revision: 3, savedAt: 'number'}
  );
  // Simulate a reload: drop the element and mount a fresh feed. The stored
  // draft must survive while the in-memory panel is gone.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  const writesBefore = await page.evaluate(
    () => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length
  );
  const retriesBefore = await page.evaluate(
    (clarifyId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${clarifyId}/retry`))
        .length,
    clarifyId
  );
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'My saved answer');
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length),
    writesBefore,
    'restoring a saved answer must not write'
  );
  assert.equal(
    await page.evaluate(
      (clarifyId) =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${clarifyId}/retry`))
          .length,
      clarifyId
    ),
    retriesBefore,
    'restoration must never auto-submit'
  );
  await feed.getByRole('button', {name: 'Submit answer', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  assert.equal(await page.evaluate((key) => window.localStorage.getItem(key), draftKey), null);
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
});

test('revision draft keeps text on failure, requires review on source change, discards explicitly', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const reviseId = 'e'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-1'}])
      );
      (window as unknown as {revState: {revision: number; failNext: boolean}}).revState = {
        revision: 5,
        failNext: false,
      };
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; failNext: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: state.revision === 5 ? 'Original request v5' : 'Original request v6 changed',
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.failNext)
            return Response.json({error: {message: 'Temporary failure. Your edits are kept.'}}, {status: 503});
          return Response.json({ok: true});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Revised request v5 kept');
  const draftKey5 = `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:5`;
  const readDraftBody = (key: string) =>
    page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null')?.body ?? null, key);
  assert.equal(await readDraftBody(draftKey5), 'Revised request v5 kept');
  await page.evaluate(() => {
    (window as unknown as {revState: {revision: number; failNext: boolean}}).revState.failNext = true;
  });
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.locator('[data-feed-message]').filter({hasText: 'kept'}).waitFor();
  assert.equal(await body.inputValue(), 'Revised request v5 kept');
  assert.equal(await readDraftBody(draftKey5), 'Revised request v5 kept');
  // Source revision advances while the draft is unsent; reopening must keep
  // the saved text, require review, and never auto-submit.
  await page.evaluate(() => {
    const state = (window as unknown as {revState: {revision: number; failNext: boolean}}).revState;
    state.revision = 6;
    state.failNext = false;
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  const retriesBefore = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .length,
    reviseId
  );
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Revised request v5 kept');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  assert.equal(
    await page.evaluate(
      (reviseId) =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
          .length,
      reviseId
    ),
    retriesBefore,
    'stale restoration must never auto-submit'
  );
  // Explicit discard clears every stored revision for the comment; the next
  // reopen serves the fresh source text.
  await feed.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
  assert.equal(
    await page.evaluate((reviseId) => {
      const prefix = `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:`;
      const kept: string[] = [];
      for (let index = 0; index < window.localStorage.length; index++) {
        const key = window.localStorage.key(index);
        if (key?.startsWith(prefix)) kept.push(key);
      }
      return kept.length;
    }, reviseId),
    0
  );
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Original request v6 changed');
});

test('a delayed revision success keeps later unsent input instead of clearing it', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const reviseId = 'f'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-1'}])
      );
      (window as unknown as {revGate: {release: boolean; revision: number}}).revGate = {release: false, revision: 5};
      window.drawerFixture.setReply(async (call) => {
        const gate = (window as unknown as {revGate: {release: boolean; revision: number}}).revGate;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: gate.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Original request v' + gate.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          while (!gate.release) await new Promise((resolve) => setTimeout(resolve, 10));
          gate.revision = 6;
          return Response.json({ok: true, revision: 6});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Submitted revision text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  // While the retry POST is pending, the textareas stay editable and later
  // keystrokes persist as newer unsent input.
  await body.fill('Submitted revision text plus newer unsent words');
  const draftKey5 = `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:5`;
  await page.waitForFunction(
    (key) =>
      (JSON.parse(window.localStorage.getItem(key) ?? 'null') as {body?: string} | null)?.body ===
      'Submitted revision text plus newer unsent words',
    draftKey5
  );
  await page.evaluate(() => {
    (window as unknown as {revGate: {release: boolean}}).revGate.release = true;
  });
  await feed.locator('[data-feed-message]').filter({hasText: 'newer edits are kept'}).waitFor();
  assert.equal(await body.inputValue(), 'Submitted revision text plus newer unsent words');
  assert.equal(await feed.locator('.soda-feed-revision').count(), 1);
  const draftKey6 = `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:6`;
  assert.equal(
    await page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null')?.body ?? null, draftKey6),
    'Submitted revision text plus newer unsent words',
    'surviving text is re-keyed under the adopted live revision'
  );
  assert.equal(
    await page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null')?.body ?? null, draftKey5),
    'Submitted revision text plus newer unsent words',
    'the spent revision draft keeps divergent newer text instead of deleting it'
  );
  assert.equal(
    await page.evaluate(
      (reviseId) =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
          .length,
      reviseId
    ),
    1,
    'the delayed success must never auto-submit the newer input'
  );
});

test('repeated source changes restore the newest revision draft and ignore other accounts', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const reviseId = 'b'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-1'}])
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 5};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: `Original request v${state.revision}`,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST')
          return Response.json({ok: true});
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  await feed.locator('[data-revision-body]').fill('Draft written against revision 5');
  // Source advances to 6: reopening restores the revision 5 draft for review,
  // then the reviewer rewrites it against revision 6.
  await page.evaluate(() => {
    (window as unknown as {revState: {revision: number}}).revState.revision = 6;
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Draft written against revision 5');
  await feed.locator('[data-revision-body]').fill('Draft rewritten against revision 6');
  // A tempting decoy from another account at a higher revision must be ignored.
  await page.evaluate((reviseId) => {
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:other-account:7:${reviseId}:9`,
      JSON.stringify({body: 'Other account newer text', revision: 9, savedAt: Date.now()})
    );
  }, reviseId);
  await page.evaluate(() => {
    (window as unknown as {revState: {revision: number}}).revState.revision = 7;
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  const retriesBefore = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .length,
    reviseId
  );
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Draft rewritten against revision 6');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  assert.equal(
    await page.evaluate(
      (reviseId) =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
          .length,
      reviseId
    ),
    retriesBefore,
    'newest-draft restoration must never auto-submit'
  );
});

test('successful revision submit adopts the live revision and drops obsolete drafts', async (t) => {
  const page = await fixture(t);
  const record = 'b'.repeat(32);
  const reviseId = 'f'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-2'}])
      );
      // Same text saved at an older revision: an ancestor, safe to retire.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
        JSON.stringify({body: 'Fresh v6 text', revision: 5, savedAt: 1})
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 6};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Fresh v6 text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const keys = await page.evaluate(() =>
    Object.keys(window.localStorage).filter((key) => key.includes('soda-feed-revision-draft'))
  );
  assert.deepEqual(keys, [], 'obsolete and submitted drafts are gone after success');
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
  // Reopening at the incremented revision restores source text with no
  // stale warning and performs no write.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Source v7');
  assert.equal(await feed.locator('.soda-feed-revision [role="alert"]').count(), 0);
  // A second submit uses the live revision, not the spent one.
  await feed.locator('[data-revision-body]').fill('Fresh v7 text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].expected_revision, 6);
  assert.equal(posts[1].expected_revision, 7);
});

test('newer writing during submit adopts the live revision for a second submit', async (t) => {
  const page = await fixture(t);
  const record = 'c'.repeat(32);
  const reviseId = '0'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-3'}])
      );
      (window as unknown as {revState: {revision: number; hold: boolean}}).revState = {revision: 6, hold: true};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; hold: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.hold)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('First writing v6');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('First writing v6 plus more');
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
      exact: true,
    })
    .waitFor();
  assert.equal(await body.inputValue(), 'First writing v6 plus more');
  const draftKey7 = `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:7`;
  const kept = await page.evaluate(
    (key) => JSON.parse(window.localStorage.getItem(key) ?? 'null')?.body ?? null,
    draftKey7
  );
  assert.equal(kept, 'First writing v6 plus more', 'surviving text is re-keyed under the live revision');
  await page.evaluate(() => {
    (window as unknown as {revState: {revision: number; hold: boolean}}).revState.hold = false;
  });
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].expected_revision, 6);
  assert.equal(posts[1].expected_revision, 7, 'second submit uses the adopted live revision');
});

test('keystrokes during refetch survive with the adopted revision', async (t) => {
  const page = await fixture(t);
  const record = 'd'.repeat(32);
  const reviseId = '1'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-4'}])
      );
      (window as unknown as {revState: {revision: number; holdGet: boolean}}).revState = {
        revision: 6,
        holdGet: false,
      };
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; holdGet: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET') {
          if (state.holdGet)
            await new Promise<void>((resolve) => {
              (window as unknown as {releaseGet: () => void}).releaseGet = resolve;
            });
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision = 7;
          // Hold the panel's authorized refetch inside the fixture so the
          // newer keystrokes provably land during a blocked GET.
          state.holdGet = true;
          return Response.json({ok: true});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Base text v6');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  // Wait until the panel's refetch GET is provably blocked, then type
  // during that GET.
  await page.waitForFunction(() => (window as unknown as {releaseGet?: unknown}).releaseGet !== undefined);
  await body.fill('Base text v6 plus refetch-time words');
  await page.evaluate(() => {
    (window as unknown as {releaseGet: () => void}).releaseGet();
  });
  await feed
    .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
      exact: true,
    })
    .waitFor();
  assert.equal(await body.inputValue(), 'Base text v6 plus refetch-time words');
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].expected_revision, 6);
});

test('unverifiable revision keeps text with a review requirement', async (t) => {
  const page = await fixture(t);
  const record = 'e'.repeat(32);
  const reviseId = '2'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-5'}])
      );
      (window as unknown as {revState: {revision: number; holdPost: boolean; failGet: boolean}}).revState = {
        revision: 6,
        holdPost: true,
        failGet: false,
      };
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; holdPost: boolean; failGet: boolean}})
          .revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET') {
          if (state.failGet) return Response.json({error: {message: 'Refetch unavailable.'}}, {status: 500});
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.holdPost)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          return Response.json({ok: true});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Unsent kept text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  // Fail only the post-submit refetch; the opening read stays healthy.
  await page.evaluate(() => {
    (window as unknown as {revState: {revision: number; holdPost: boolean; failGet: boolean}}).revState.failGet = true;
  });
  await body.fill('Unsent kept text plus more');
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText(
      'Request updated, but the live revision could not be confirmed. Review the restored text before submitting.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await body.inputValue(), 'Unsent kept text plus more');
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .length,
    reviseId
  );
  assert.equal(posts, 1, 'no second submit is attempted without a verified revision');
});

test('cross-tab divergent drafts survive submit and the re-key never overwrites', async (t) => {
  const page = await fixture(t);
  const record = 'f'.repeat(32);
  const reviseId = '3'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-divergent'}])
      );
      (window as unknown as {revState: {revision: number; holdPost: boolean}}).revState = {
        revision: 6,
        holdPost: true,
      };
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; holdPost: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.holdPost)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Source v6');
  // Another tab saves divergent text under the adopted revision and an
  // older revision after this panel opened.
  await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:${account}:7:${reviseId}:7`,
      JSON.stringify({body: 'Tab B divergent text', revision: 7, savedAt: Date.now()})
    );
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
      JSON.stringify({body: 'Older divergent text', revision: 5, savedAt: 1})
    );
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:${account}:7:${reviseId}:4`,
      JSON.stringify({body: 'Tab A submitted text', revision: 4, savedAt: 1})
    );
  }, reviseId);
  await body.fill('Tab A submitted text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('Tab A submitted text plus live words');
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
      exact: true,
    })
    .waitFor();
  // The live panel keeps our newer text; neither divergent draft was
  // deleted nor overwritten, while the matching ancestor retired.
  assert.equal(await body.inputValue(), 'Tab A submitted text plus live words');
  assert.equal(await feed.locator('.soda-feed-revision').count(), 1);
  const stored = await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const at = (revision: number) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:${account}:7:${reviseId}:${revision}`) ?? 'null'
      )?.body ?? null;
    return {adopted: at(7), older: at(5), ancestor: at(4), spent: at(6)};
  }, reviseId);
  assert.equal(stored.adopted, 'Tab B divergent text', 'adopted-revision divergent draft is never overwritten');
  assert.equal(stored.older, 'Older divergent text', 'older divergent draft is never deleted');
  assert.equal(stored.ancestor, null, 'matching ancestor text still retires');
  assert.equal(stored.spent, 'Tab A submitted text plus live words', 'newer unsent input stays persisted');
  // A second submit from the live panel uses the adopted live CAS.
  await page.evaluate(() => {
    (window as unknown as {revState: {holdPost: boolean}}).revState.holdPost = false;
  });
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].expected_revision, 6);
  assert.equal(posts[1].expected_revision, 7, 'second submit uses the adopted live revision');
});

test('unverified revision disables Submit, blocks forced posts, and refresh restores the live CAS', async (t) => {
  const page = await fixture(t);
  const record = '2'.repeat(32);
  const reviseId = '4'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-unverified'}])
      );
      (window as unknown as {revState: {revision: number; holdPost: boolean; failGet: boolean}}).revState = {
        revision: 6,
        holdPost: true,
        failGet: false,
      };
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; holdPost: boolean; failGet: boolean}})
          .revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET') {
          if (state.failGet) return Response.json({error: {message: 'Refetch unavailable.'}}, {status: 503});
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.holdPost)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          // No revision field: only the authorized refetch can confirm live.
          return Response.json({ok: true});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Kept text v6');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('Kept text v6 plus more');
  // Fail only the post-submit refetch; the opening read stays healthy.
  await page.evaluate(() => {
    (window as unknown as {revState: {failGet: boolean}}).revState.failGet = true;
  });
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText(
      'Request updated, but the live revision could not be confirmed. Review the restored text before submitting.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await body.inputValue(), 'Kept text v6 plus more');
  assert.equal(await feed.locator('[data-revision-submit]').isDisabled(), true, 'Submit stays disabled');
  await feed.getByRole('button', {name: 'Refresh revision status', exact: true}).waitFor();
  // A forced reviseComment attempt past the disabled button sends zero POST.
  const postsBefore = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .length,
    reviseId
  );
  await page.evaluate(() =>
    (document.querySelector('soda-factory-feed') as unknown as {reviseComment: () => Promise<void>}).reviseComment()
  );
  await page.waitForTimeout(500);
  const postsAfter = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .length,
    reviseId
  );
  assert.equal(postsAfter, postsBefore, 'unverified guard sends zero POST');
  assert.equal(await body.inputValue(), 'Kept text v6 plus more');
  // The explicit refresh adopts the fresh revision with the kept text.
  await page.evaluate(() => {
    const state = (window as unknown as {revState: {revision: number; holdPost: boolean; failGet: boolean}}).revState;
    state.failGet = false;
    state.holdPost = false;
  });
  await feed.getByRole('button', {name: 'Refresh revision status', exact: true}).click();
  await feed.getByText('Live revision confirmed. Review the kept text before submitting.', {exact: true}).waitFor();
  assert.equal(await body.inputValue(), 'Kept text v6 plus more');
  assert.equal(await feed.locator('[data-revision-submit]').isDisabled(), false, 'Submit re-enables with live CAS');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].expected_revision, 6);
  assert.equal(posts[1].expected_revision, 7, 'second submit uses the refreshed live revision');
});

test('unverified revision without a revision field keeps text and blocks submits', async (t) => {
  const page = await fixture(t);
  const record = '5'.repeat(32);
  const reviseId = '6'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-missing'}])
      );
      (window as unknown as {revState: {revision: number; holdPost: boolean; omitRevision: boolean}}).revState = {
        revision: 6,
        holdPost: true,
        omitRevision: false,
      };
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; holdPost: boolean; omitRevision: boolean}})
          .revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET') {
          const detail = {
            comment_id: reviseId,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          };
          // Missing-revision variant: a 200 without a revision field.
          return Response.json(state.omitRevision ? detail : {...detail, revision: state.revision});
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.holdPost)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          return Response.json({ok: true});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Missing revision kept text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('Missing revision kept text plus more');
  await page.evaluate(() => {
    (window as unknown as {revState: {omitRevision: boolean}}).revState.omitRevision = true;
  });
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText(
      'Request updated, but the live revision could not be confirmed. Review the restored text before submitting.',
      {exact: true}
    )
    .waitFor();
  assert.equal(await body.inputValue(), 'Missing revision kept text plus more');
  assert.equal(await feed.locator('[data-revision-submit]').isDisabled(), true, 'Submit stays disabled');
  // A forced reviseComment attempt past the disabled button sends zero POST.
  const postsBefore = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .length,
    reviseId
  );
  await page.evaluate(() =>
    (document.querySelector('soda-factory-feed') as unknown as {reviseComment: () => Promise<void>}).reviseComment()
  );
  await page.waitForTimeout(500);
  assert.equal(
    await page.evaluate(
      (reviseId) =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
          .length,
      reviseId
    ),
    postsBefore,
    'unverified guard sends zero POST'
  );
  assert.equal(await body.inputValue(), 'Missing revision kept text plus more');
  // The explicit refresh still cannot confirm a revision, so it stays
  // unverified with an explicit message instead of inventing a CAS.
  await feed.getByRole('button', {name: 'Refresh revision status', exact: true}).click();
  await feed
    .getByText('The live revision is still unavailable. Your text is kept; try refreshing again.', {exact: true})
    .waitFor();
  assert.equal(await feed.locator('[data-revision-submit]').isDisabled(), true, 'Submit stays disabled');
  assert.equal(await body.inputValue(), 'Missing revision kept text plus more');
});

test('stale draft lineage retires after in-panel evolve and submit', async (t) => {
  const page = await fixture(t);
  const record = '9'.repeat(32);
  const reviseId = '7'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-lineage'}])
      );
      // Superseded text saved earlier at v5 on this panel's line.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
        JSON.stringify({body: 'Superseded draft A', revision: 5, lineage: 'lineage-a', savedAt: 1})
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 6};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  // The stale v5 draft restores for review against v6, never auto-submits.
  assert.equal(await body.inputValue(), 'Superseded draft A');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  // Evolving to B in-panel stays on the restored lineage.
  await body.fill('Evolved draft B');
  const storedV6 = await page.evaluate(
    (reviseId) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:6`) ?? 'null'
      ) as {body?: string; lineage?: string} | null,
    reviseId
  );
  assert.equal(storedV6?.body, 'Evolved draft B');
  assert.equal(storedV6?.lineage, 'lineage-a', 'in-panel persists inherit the restored lineage');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  // The same-lineage superseded v5 retires with the submitted v6.
  assert.deepEqual(
    await page.evaluate(() =>
      Object.keys(window.localStorage).filter((key) => key.includes('soda-feed-revision-draft'))
    ),
    [],
    'superseded same-lineage draft retires with the submit'
  );
  // Reopening at v7 restores the clean adopted state, never A.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Source v7');
  assert.equal(await feed.locator('.soda-feed-revision [role="alert"]').count(), 0);
});

test('submit preserves a foreign lineage draft', async (t) => {
  const page = await fixture(t);
  const record = '8'.repeat(32);
  const reviseId = '9'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-foreign'}])
      );
      // Another tab's unsent draft on its own lineage.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
        JSON.stringify({body: 'Foreign tab kept text', revision: 5, lineage: 'foreign-tab', savedAt: 1})
      );
      // This panel's own saved draft at the live revision.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:6`,
        JSON.stringify({body: 'Own v6 text', revision: 6, lineage: 'own-tab', savedAt: 2})
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 6};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  // The live saved draft restores directly with no stale warning.
  assert.equal(await body.inputValue(), 'Own v6 text');
  assert.equal(await feed.locator('.soda-feed-revision [role="alert"]').count(), 0);
  await body.fill('Own submitted B');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const stored = await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const at = (revision: number) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:${account}:7:${reviseId}:${revision}`) ?? 'null'
      ) as {body?: string; lineage?: string} | null;
    return {older: at(5), spent: at(6)};
  }, reviseId);
  assert.equal(stored.older?.body, 'Foreign tab kept text', 'foreign lineage draft survives the submit');
  assert.equal(stored.older?.lineage, 'foreign-tab');
  assert.equal(stored.spent, null, 'submitted draft retires');
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].expected_revision, 6);
});

test('submit preserves a shared-lineage late write after a stale restore', async (t) => {
  const page = await fixture(t);
  const record = '5'.repeat(32);
  const reviseId = '4'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-shared'}])
      );
      // Tab A opens v5 and saves draft A on the shared lineage.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
        JSON.stringify({body: 'Draft A text', revision: 5, lineage: 'lineage-shared', savedAt: 1})
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 6};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  // Tab B restores the stale v5 draft A against live v6, inheriting the
  // shared lineage.
  assert.equal(await body.inputValue(), 'Draft A text');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  // Tab A, still open on v5, types newer unsent C into key5. Its
  // compare-and-swap succeeds: the stored text still equals A's tracked
  // text, and the write stays on the shared lineage.
  await page.evaluate((reviseId) => {
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:5`,
      JSON.stringify({body: 'Late write C text', revision: 5, lineage: 'lineage-shared', savedAt: 2})
    );
  }, reviseId);
  // Tab B revises A into B and submits against the live v6 CAS.
  await body.fill('Submitted B text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].expected_revision, 6);
  // The late shared-lineage write survives; the submitted v6 draft retires.
  const stored = await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const at = (revision: number) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:${account}:7:${reviseId}:${revision}`) ?? 'null'
      ) as {body?: string; lineage?: string} | null;
    return {older: at(5), spent: at(6)};
  }, reviseId);
  assert.equal(stored.older?.body, 'Late write C text', 'shared-lineage late write survives the submit');
  assert.equal(stored.older?.lineage, 'lineage-shared');
  assert.equal(stored.spent, null, 'submitted draft retires');
  // A later reopen offers C for review instead of resurrecting stale A.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Late write C text');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
});

test('keystroke after adopted-key collision keeps foreign text and flags a conflict', async (t) => {
  const page = await fixture(t);
  const record = '6'.repeat(32);
  const reviseId = '8'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-collision'}])
      );
      (window as unknown as {revState: {revision: number; hold: boolean}}).revState = {revision: 6, hold: true};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; hold: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.hold)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Local submit text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('Local submit text plus newer words');
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
      exact: true,
    })
    .waitFor();
  // Another tab lands a foreign write on the adopted key.
  const localAfterCollision = 'Local submit text plus newer words plus keystroke';
  await page.evaluate((reviseId) => {
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:7`,
      JSON.stringify({body: 'Foreign adopted text', revision: 7, lineage: 'foreign-adopted', savedAt: Date.now()})
    );
  }, reviseId);
  await body.fill(localAfterCollision);
  // The keystroke never overwrites: the foreign text stays stored, the panel
  // keeps the local text, and an explicit conflict is surfaced.
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText(
      'Another tab saved a different draft for this revision. Your text is kept here and the saved draft was left untouched.',
      {exact: true}
    )
    .waitFor();
  assert.equal(
    await page.evaluate(
      (reviseId) =>
        JSON.parse(
          window.localStorage.getItem(`soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:7`) ?? 'null'
        )?.body ?? null,
      reviseId
    ),
    'Foreign adopted text',
    'stored foreign text stays intact'
  );
  assert.equal(await body.inputValue(), localAfterCollision, 'panel keeps the local text');
  // An explicit Submit still sends the live panel text exactly once.
  await page.evaluate(() => {
    (window as unknown as {revState: {hold: boolean}}).revState.hold = false;
  });
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].expected_revision, 6);
  assert.equal(posts[1].expected_revision, 7);
  assert.equal(posts[1].body, localAfterCollision, 'second submit sends the live text');
  assert.equal(
    await page.evaluate(
      (reviseId) =>
        JSON.parse(
          window.localStorage.getItem(`soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:7`) ?? 'null'
        )?.body ?? null,
      reviseId
    ),
    'Foreign adopted text',
    'submit never overwrites the foreign draft'
  );
});

test('reload-before-submit retires the persisted ancestor chain without resurrection', async (t) => {
  const page = await fixture(t);
  const record = '1'.repeat(32);
  const reviseId = '2'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-reload'}])
      );
      // Unsent draft saved earlier at v5 on this panel's line.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
        JSON.stringify({body: 'Reload draft A', revision: 5, lineage: 'lineage-reload', savedAt: 1})
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 6};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  // The stale v5 draft restores for review against v6, never auto-submits.
  assert.equal(await body.inputValue(), 'Reload draft A');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  // Evolving to B in-panel persists the ancestor chain alongside the text.
  await body.fill('Reload draft B');
  const storedV6 = await page.evaluate(
    (reviseId) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:6`) ?? 'null'
      ) as {body?: string; lineage?: string; ancestors?: {revision?: number; body?: string}[]} | null,
    reviseId
  );
  assert.equal(storedV6?.body, 'Reload draft B');
  assert.equal(storedV6?.lineage, 'lineage-reload', 'in-panel persists inherit the restored lineage');
  assert.deepEqual(
    (storedV6?.ancestors ?? []).map((row) => [row.revision, row.body]),
    [[5, 'Reload draft A']],
    'the superseded v5 snapshot persists with the v6 draft'
  );
  // Reload before submitting: remount and reopen at the same live revision.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  // The persisted v6 text restores directly with no stale-source warning.
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Reload draft B');
  assert.equal(await feed.locator('.soda-feed-revision [role="alert"]').count(), 0);
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  // The persisted ancestor chain retires with the submit; nothing resurrects.
  assert.deepEqual(
    await page.evaluate(() =>
      Object.keys(window.localStorage).filter((key) => key.includes('soda-feed-revision-draft'))
    ),
    [],
    'persisted ancestor chain retires with the submit'
  );
  // Reopening at v7 restores the clean adopted state, never A.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Source v7');
  assert.equal(await feed.locator('.soda-feed-revision [role="alert"]').count(), 0);
});

test('revision advances retire the full ancestor chain only when unchanged', async (t) => {
  const page = await fixture(t);
  const record = '3'.repeat(32);
  const reviseId = '0'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-chain'}])
      );
      // Unsent draft saved earlier at v5 on this panel's line.
      window.localStorage.setItem(
        `soda-feed-revision-draft:v1:${account}:7:${reviseId}:5`,
        JSON.stringify({body: 'Chain draft A', revision: 5, lineage: 'lineage-chain', savedAt: 1})
      );
      (window as unknown as {revState: {revision: number}}).revState = {revision: 6};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  // v5 restores for review against live v6; evolving to B persists the chain.
  assert.equal(await body.inputValue(), 'Chain draft A');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  await body.fill('Chain draft B');
  // The source advances to v7 before anything submits; reopen at the new live revision.
  await page.evaluate(() => {
    (window as unknown as {revState: {revision: number}}).revState.revision = 7;
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  // The newest superseded draft restores for review; evolving to C extends the chain.
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Chain draft B');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
  await feed.locator('[data-revision-body]').fill('Chain draft C');
  const storedV7 = await page.evaluate(
    (reviseId) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:7`) ?? 'null'
      ) as {body?: string; ancestors?: {revision?: number; body?: string}[]} | null,
    reviseId
  );
  assert.equal(storedV7?.body, 'Chain draft C');
  assert.deepEqual(
    (storedV7?.ancestors ?? []).map((row) => [row.revision, row.body]),
    [
      [5, 'Chain draft A'],
      [6, 'Chain draft B'],
    ],
    'v5 and v6 snapshots persist with the v7 draft'
  );
  // Another tab on the same lineage lands newer unsent text on v6 after the chain was recorded.
  await page.evaluate((reviseId) => {
    window.localStorage.setItem(
      `soda-feed-revision-draft:v1:fixture-generation:7:${reviseId}:6`,
      JSON.stringify({body: 'Late unsent B2', revision: 6, lineage: 'lineage-chain', savedAt: 3})
    );
  }, reviseId);
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].expected_revision, 7);
  // The unchanged v5 ancestor retires; the late-edited v6 text and the spent v7 draft resolve correctly.
  const stored = await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const at = (revision: number) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:${account}:7:${reviseId}:${revision}`) ?? 'null'
      ) as {body?: string; lineage?: string} | null;
    return {older: at(5), late: at(6), spent: at(7)};
  }, reviseId);
  assert.equal(stored.older, null, 'unchanged v5 ancestor retires with the submit');
  assert.equal(stored.late?.body, 'Late unsent B2', 'late-edited same-lineage text survives the submit');
  assert.equal(stored.late?.lineage, 'lineage-chain');
  assert.equal(stored.spent, null, 'submitted draft retires');
  // A later reopen offers the surviving newer text for review instead of resurrecting A.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Late unsent B2');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
});

test('changed-after-rekey double submit retires the adopted snapshot without resurrection', async (t) => {
  const page = await fixture(t);
  const record = '4'.repeat(32);
  const reviseId = '5'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-rekey-104'}])
      );
      (window as unknown as {revState: {revision: number; hold: boolean}}).revState = {revision: 6, hold: true};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; hold: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.hold)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  // Submit A at v6, then type B while the POST waits.
  await body.fill('Submitted A text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('Adopted B text');
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
      exact: true,
    })
    .waitFor();
  // Success adopts v7 and re-keys B there; the v6 B snapshot persists in the chain.
  const afterAdopt = await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const at = (revision: number) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:${account}:7:${reviseId}:${revision}`) ?? 'null'
      ) as {body?: string; ancestors?: {revision?: number; body?: string}[]} | null;
    return {spent: at(6), adopted: at(7)};
  }, reviseId);
  assert.equal(afterAdopt.spent?.body, 'Adopted B text');
  assert.equal(afterAdopt.adopted?.body, 'Adopted B text');
  assert.deepEqual(
    (afterAdopt.adopted?.ancestors ?? []).map((row) => [row.revision, row.body]),
    [[6, 'Adopted B text']],
    'the own adopted B text persists as a snapshot in the chain'
  );
  // Edit C at v7 and submit C against the adopted live revision.
  await body.fill('Edited C text');
  await page.evaluate(() => {
    (window as unknown as {revState: {hold: boolean}}).revState.hold = false;
  });
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[0].expected_revision, 6);
  assert.equal(posts[0].body, 'Submitted A text');
  assert.equal(posts[1].expected_revision, 7, 'second submit uses the adopted live revision');
  assert.equal(posts[1].body, 'Edited C text', 'v8 carries C');
  // The unchanged v6 B snapshot retires with the second submit.
  assert.deepEqual(
    await page.evaluate(() =>
      Object.keys(window.localStorage).filter((key) => key.includes('soda-feed-revision-draft'))
    ),
    [],
    'unchanged adopted B retires with the C submit'
  );
  // Reopening at v8 restores the clean source text, never B.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Source v8');
  assert.equal(await feed.locator('.soda-feed-revision [role="alert"]').count(), 0);
});

test('changed-after-rekey late edit preserves the adopted text for review', async (t) => {
  const page = await fixture(t);
  const record = '6'.repeat(32);
  const reviseId = '7'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const account = 'fixture-generation';
      window.localStorage.setItem(
        `soda-feed-intake:v1:${account}:record:${record}`,
        JSON.stringify([{commentId: reviseId, key: 'revise-key-rekey-104b'}])
      );
      (window as unknown as {revState: {revision: number; hold: boolean}}).revState = {revision: 6, hold: true};
      window.drawerFixture.setReply(async (call) => {
        const state = (window as unknown as {revState: {revision: number; hold: boolean}}).revState;
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: 7, seq: 1}], truncated: false});
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}`) && call.method === 'GET')
          return Response.json({
            comment_id: reviseId,
            revision: state.revision,
            repository: 7,
            author_name: 'alice',
            at: 1720000000,
            edited: false,
            body: 'Source v' + state.revision,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            route: 'discussion',
            can_retry: true,
            issue: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`) && call.method === 'POST') {
          if (state.hold)
            await new Promise<void>((resolve) => {
              (window as unknown as {releasePost: () => void}).releasePost = resolve;
            });
          const posted = JSON.parse(call.body || '{}');
          if (posted.expected_revision !== state.revision)
            return Response.json({error: {message: 'Stale revision.'}}, {status: 409});
          state.revision += 1;
          return Response.json({ok: true, revision: state.revision});
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  await body.fill('Submitted A text');
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await body.fill('Adopted B text');
  await page.waitForFunction(() => (window as unknown as {releasePost?: unknown}).releasePost !== undefined);
  await page.evaluate(() => {
    (window as unknown as {releasePost: () => void}).releasePost();
  });
  await feed
    .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
      exact: true,
    })
    .waitFor();
  // The adopted B snapshot is edited again on its own line before the second submit.
  await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const key6 = `soda-feed-revision-draft:v1:${account}:7:${reviseId}:6`;
    const current = JSON.parse(window.localStorage.getItem(key6) ?? 'null') as {
      body?: string;
      lineage?: string;
      revision?: number;
    } | null;
    window.localStorage.setItem(
      key6,
      JSON.stringify({body: 'Late-edited B2 text', revision: 6, lineage: current?.lineage ?? 'own-lineage', savedAt: 3})
    );
  }, reviseId);
  await body.fill('Edited C text');
  await page.evaluate(() => {
    (window as unknown as {revState: {hold: boolean}}).revState.hold = false;
  });
  await feed.getByRole('button', {name: 'Submit updated request', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posts = await page.evaluate(
    (reviseId) =>
      window.drawerFixture.calls
        .filter((call) => call.url.endsWith(`/api/factory/feed/comments/${reviseId}/retry`))
        .map((call) => JSON.parse(call.body ?? '{}')),
    reviseId
  );
  assert.equal(posts.length, 2);
  assert.equal(posts[1].expected_revision, 7);
  assert.equal(posts[1].body, 'Edited C text');
  const stored = await page.evaluate((reviseId) => {
    const account = 'fixture-generation';
    const at = (revision: number) =>
      JSON.parse(
        window.localStorage.getItem(`soda-feed-revision-draft:v1:${account}:7:${reviseId}:${revision}`) ?? 'null'
      ) as {body?: string} | null;
    return {older: at(6), spent: at(7)};
  }, reviseId);
  assert.equal(stored.older?.body, 'Late-edited B2 text', 'changed adopted text is preserved for review');
  assert.equal(stored.spent, null, 'submitted C draft retires');
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Revise or retry request', exact: true}).click();
  await feed.locator('[data-revision-body]').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Late-edited B2 text');
  await feed
    .locator('.soda-feed-revision [role="alert"]')
    .getByText('Source request changed since this draft was saved. Review the restored text before submitting.', {
      exact: true,
    })
    .waitFor();
});

test('a slow source GET resolving after Back-to-activity leaves activity view intact', async (t) => {
  const page = await fixture(t);
  const item = 'b'.repeat(32);
  const source = 'c'.repeat(32);
  await page.evaluate(
    ({item, source}) => {
      const record = {
        id: item,
        revision: 1,
        title: 'Decision record',
        body: 'Recorded rationale.',
        source_post: source,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.includes('/api/factory/library/item?')) return Response.json({item: record, history: []});
        if (call.url.includes('/api/factory/library?')) return Response.json({items: [record], next_cursor: ''});
        if (call.url.endsWith(`/api/factory/feed/comments/${source}`) && call.method === 'GET')
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {sourceHolds?: Record<string, (response: Response) => void>};
            if (!holds.sourceHolds) holds.sourceHolds = {};
            (holds.sourceHolds as Record<string, (response: Response) => void>)[source] = resolve;
          });
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {item, source}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).click();
  await feed.getByRole('button', {name: 'Open source conversation'}).click();
  const pending = feed.getByRole('button', {name: 'Opening source'});
  await pending.waitFor();
  assert.equal(await pending.isDisabled(), true);
  await feed.getByRole('button', {name: 'Back to activity'}).click();
  await feed.getByText('No activity yet.').waitFor();
  await page.evaluate((source) => {
    const holds = (window as unknown as {sourceHolds: Record<string, (response: Response) => void>}).sourceHolds;
    const release = holds[source];
    if (!release) throw Error('source GET was never issued');
    release(
      Response.json({
        comment_id: source,
        revision: 0,
        repository: 7,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        body: 'Late source body.',
        intake: 'recorded',
        judgment: 'pending',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      })
    );
  }, source);
  await page.waitForTimeout(250);
  assert.equal(await feed.getByText('Source conversation').count(), 0);
  assert.equal(await feed.getByText('Late source body.').count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 0);
  await feed.getByText('No activity yet.').waitFor();
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('two rapid source clicks render the newer source only', async (t) => {
  const page = await fixture(t);
  const item = 'b'.repeat(32);
  const first = 'c'.repeat(32);
  const second = 'e'.repeat(32);
  await page.evaluate(
    ({item, first, second}) => {
      const record = (revision: number, source: string, body: string) => ({
        id: item,
        revision,
        title: 'Decision record',
        body,
        source_post: source,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.includes('/api/factory/library/item?'))
          return Response.json({
            item: record(2, first, 'Recorded rationale.'),
            history: [{item: record(1, second, 'Older rationale.'), reason: 'Revised', actor: 'Ada'}],
          });
        if (call.url.includes('/api/factory/library?'))
          return Response.json({items: [record(2, first, 'Recorded rationale.')], next_cursor: ''});
        for (const source of [first, second]) {
          if (call.url.endsWith(`/api/factory/feed/comments/${source}`) && call.method === 'GET')
            return new Promise<Response>((resolve) => {
              const holds = window as unknown as {sourceHolds?: Record<string, (response: Response) => void>};
              if (!holds.sourceHolds) holds.sourceHolds = {};
              (holds.sourceHolds as Record<string, (response: Response) => void>)[source] = resolve;
            });
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {item, first, second}
  );
  const feed = page.locator('soda-factory-feed');
  const resolveSource = (source: string, body: string) =>
    page.evaluate(
      ({source, body}) => {
        const holds = (window as unknown as {sourceHolds: Record<string, (response: Response) => void>}).sourceHolds;
        const release = holds[source];
        if (!release) throw Error('source GET was never issued');
        release(
          Response.json({
            comment_id: source,
            revision: 0,
            repository: 7,
            author_name: 'Ada',
            at: 1720000000,
            edited: false,
            body,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          })
        );
      },
      {source, body}
    );
  await feed.getByRole('heading', {name: 'Activity', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).click();
  await feed.getByText('Recent revisions · 1').click();
  await feed.getByText('Revision 1 · Revised · Ada').click();
  const openers = feed.getByRole('button', {name: 'Open source conversation'});
  await openers.nth(0).click();
  await feed.getByRole('button', {name: 'Opening source'}).waitFor();
  // The first button now shows its pending state, so the remaining opener is
  // the revision's newer source.
  await feed.getByRole('button', {name: 'Open source conversation'}).nth(0).click();
  await feed.getByRole('button', {name: 'Opening source'}).waitFor();
  assert.equal(await feed.getByRole('button', {name: 'Opening source'}).count(), 1);
  await resolveSource(first, 'First source body text.');
  await page.waitForTimeout(250);
  assert.equal(await feed.getByRole('button', {name: 'Back to feed'}).count(), 0);
  assert.equal(await feed.getByText('First source body text.').count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Opening source'}).count(), 1);
  await resolveSource(second, 'Second source body text.');
  await feed.getByRole('button', {name: 'Back to feed'}).waitFor();
  await feed.getByText('Second source body text.').waitFor();
  assert.equal(await feed.getByText('First source body text.').count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 0);
  await feed.getByRole('button', {name: 'Back to feed'}).waitFor();
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a failed source GET after a project switch leaves the new context intact', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const item = 'b'.repeat(32);
    const source = 'c'.repeat(32);
    const record = {
      id: item,
      revision: 1,
      title: 'Decision record',
      body: 'Recorded rationale.',
      source_post: source,
    };
    const holds = window as unknown as {
      sourceHolds?: Record<string, (response: Response) => void>;
      guardTransport?: unknown;
    };
    let sourceCalls = 0;
    const transport = {
      request: async (path: string) => {
        if (path.startsWith('factory/feed?')) return Response.json({entries: [], truncated: false});
        if (path === 'repositories?q=') return Response.json({items: [], next_cursor: ''});
        if (path.startsWith('factory/library/item?')) return Response.json({item: record, history: []});
        if (path.startsWith('factory/library?')) return Response.json({items: [record], next_cursor: ''});
        if (path === `factory/feed/comments/${source}`) {
          sourceCalls++;
          if (sourceCalls === 1)
            return new Promise<Response>((resolve) => {
              if (!holds.sourceHolds) holds.sourceHolds = {};
              (holds.sourceHolds as Record<string, (response: Response) => void>)[source] = resolve;
            });
          return Response.json({}, {status: 503});
        }
        throw Error('Unexpected library guard read: ' + path);
      },
    };
    holds.guardTransport = transport;
    const feed = document.createElement('soda-factory-feed') as HTMLElement & {
      context: unknown;
      updateComplete: Promise<unknown>;
    };
    feed.dataset.guardRace = 'true';
    document.body.append(feed);
    feed.context = {transport, repository: '7'};
    await feed.updateComplete;
  });
  const feed = page.locator('soda-factory-feed[data-guard-race="true"]');
  await feed.getByRole('heading', {name: 'Activity', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).click();
  await feed.getByRole('button', {name: 'Open source conversation'}).click();
  await feed.getByRole('button', {name: 'Opening source'}).waitFor();
  await page.evaluate(() => {
    const feed = document.querySelector('soda-factory-feed[data-guard-race="true"]') as HTMLElement & {
      context: unknown;
    };
    const holds = window as unknown as {guardTransport: unknown};
    feed.context = {transport: holds.guardTransport, repository: '8'};
  });
  await feed.getByText('No activity yet.').waitFor();
  await page.evaluate(() => {
    const holds = (window as unknown as {sourceHolds: Record<string, (response: Response) => void>}).sourceHolds;
    const release = holds['c'.repeat(32)];
    if (!release) throw Error('source GET was never issued');
    release(new Response(null, {status: 503}));
  });
  await page.waitForTimeout(250);
  await feed.getByText('No activity yet.').waitFor();
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
  assert.equal(await feed.getByRole('button', {name: 'Back to feed'}).count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 0);
  // A failure against the current context still surfaces truthfully.
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).click();
  await feed.getByRole('button', {name: 'Open source conversation'}).click();
  await feed.getByText('The combined feed is unavailable.').waitFor();
  assert.equal(await feed.getByRole('button', {name: 'Back to feed'}).count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 1);
});

test('a slow source GET resolving after Back-to-results never opens its thread', async (t) => {
  const page = await fixture(t);
  const item = 'b'.repeat(32);
  const source = 'c'.repeat(32);
  await page.evaluate(
    ({item, source}) => {
      const record = {
        id: item,
        revision: 1,
        title: 'Decision record',
        body: 'Recorded rationale.',
        source_post: source,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.includes('/api/factory/library/item?')) return Response.json({item: record, history: []});
        if (call.url.includes('/api/factory/library?')) return Response.json({items: [record], next_cursor: ''});
        if (call.url.endsWith(`/api/factory/feed/comments/${source}`) && call.method === 'GET')
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {sourceHolds?: Record<string, (response: Response) => void>};
            if (!holds.sourceHolds) holds.sourceHolds = {};
            (holds.sourceHolds as Record<string, (response: Response) => void>)[source] = resolve;
          });
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {item, source}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).click();
  await feed.getByRole('button', {name: 'Open source conversation'}).click();
  await feed.getByRole('button', {name: 'Opening source'}).waitFor();
  await feed.getByRole('button', {name: 'Back to results'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).waitFor();
  await page.evaluate((source) => {
    const holds = (window as unknown as {sourceHolds: Record<string, (response: Response) => void>}).sourceHolds;
    const release = holds[source];
    if (!release) throw Error('source GET was never issued');
    release(
      Response.json({
        comment_id: source,
        revision: 0,
        repository: 7,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        body: 'Late source body.',
        intake: 'recorded',
        judgment: 'pending',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      })
    );
  }, source);
  await page.waitForTimeout(250);
  assert.equal(await feed.getByText('Source conversation').count(), 0);
  assert.equal(await feed.getByText('Late source body.').count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Back to feed'}).count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 1);
  await feed.getByRole('button', {name: 'Decision record'}).waitFor();
  assert.equal(await feed.getByRole('status').innerText(), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('typing a library search while a source GET is in flight drops the stale thread', async (t) => {
  const page = await fixture(t);
  const item = 'b'.repeat(32);
  const source = 'c'.repeat(32);
  await page.evaluate(
    ({item, source}) => {
      const record = {
        id: item,
        revision: 1,
        title: 'Decision record',
        body: 'Recorded rationale.',
        source_post: source,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.includes('/api/factory/library/item?')) return Response.json({item: record, history: []});
        if (call.url.includes('/api/factory/library?')) return Response.json({items: [record], next_cursor: ''});
        if (call.url.endsWith(`/api/factory/feed/comments/${source}`) && call.method === 'GET')
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {sourceHolds?: Record<string, (response: Response) => void>};
            if (!holds.sourceHolds) holds.sourceHolds = {};
            (holds.sourceHolds as Record<string, (response: Response) => void>)[source] = resolve;
          });
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {item, source}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'Decision record'}).click();
  await feed.getByRole('button', {name: 'Open source conversation'}).click();
  await feed.getByRole('button', {name: 'Opening source'}).waitFor();
  await feed.getByLabel('Search the library').fill('rationale');
  // Typing alone clears the pending source state before the late response.
  await feed.getByRole('button', {name: 'Open source conversation'}).waitFor();
  await page.evaluate((source) => {
    const holds = (window as unknown as {sourceHolds: Record<string, (response: Response) => void>}).sourceHolds;
    const release = holds[source];
    if (!release) throw Error('source GET was never issued');
    release(
      Response.json({
        comment_id: source,
        revision: 0,
        repository: 7,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        body: 'Late source body.',
        intake: 'recorded',
        judgment: 'pending',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      })
    );
  }, source);
  await page.waitForTimeout(250);
  assert.equal(await feed.getByText('Late source body.').count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Back to feed'}).count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 1);
  // The detail view stays intact with its source button usable again.
  await feed.getByText('Recorded rationale.').waitFor();
  await feed.getByRole('button', {name: 'Open source conversation'}).waitFor();
  assert.equal(await feed.getByRole('status').innerText(), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a slow source GET resolving after a related-item move keeps the newer item', async (t) => {
  const page = await fixture(t);
  const first = 'a'.repeat(32);
  const second = 'b'.repeat(32);
  const source = 'c'.repeat(32);
  await page.evaluate(
    ({first, second, source}) => {
      const records: Record<string, unknown> = {
        [first]: {
          id: first,
          revision: 1,
          title: 'First record',
          body: 'First rationale.',
          source_post: source,
          relations: [{kind: 'see_also', target: second, evidence: []}],
        },
        [second]: {id: second, revision: 1, title: 'Second record', body: 'Second rationale.'},
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
        if (call.url.includes('/api/factory/library/item?')) {
          const id = new URL(call.url, location.origin).searchParams.get('id') ?? '';
          const record = records[id];
          if (!record) return Response.json({}, {status: 404});
          return Response.json({item: record, history: []});
        }
        if (call.url.includes('/api/factory/library?'))
          return Response.json({items: [records[first]], next_cursor: ''});
        if (call.url.endsWith(`/api/factory/feed/comments/${source}`) && call.method === 'GET')
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {sourceHolds?: Record<string, (response: Response) => void>};
            if (!holds.sourceHolds) holds.sourceHolds = {};
            (holds.sourceHolds as Record<string, (response: Response) => void>)[source] = resolve;
          });
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {first, second, source}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.getByRole('button', {name: 'First record'}).click();
  await feed.getByRole('button', {name: 'Open source conversation'}).click();
  await feed.getByRole('button', {name: 'Opening source'}).waitFor();
  await feed.getByRole('button', {name: 'bbbbbbbb'}).click();
  await feed.getByText('Second rationale.').waitFor();
  await page.evaluate((source) => {
    const holds = (window as unknown as {sourceHolds: Record<string, (response: Response) => void>}).sourceHolds;
    const release = holds[source];
    if (!release) throw Error('source GET was never issued');
    release(
      Response.json({
        comment_id: source,
        revision: 0,
        repository: 7,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        body: 'Late source body.',
        intake: 'recorded',
        judgment: 'pending',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      })
    );
  }, source);
  await page.waitForTimeout(250);
  assert.equal(await feed.getByText('Source conversation').count(), 0);
  assert.equal(await feed.getByText('Late source body.').count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Back to feed'}).count(), 0);
  await feed.getByText('Second rationale.').waitFor();
  assert.equal(await feed.getByText('First rationale.').count(), 0);
  assert.equal(await feed.locator('soda-factory-library').count(), 1);
  assert.equal(await feed.getByRole('status').innerText(), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a role reply link pages the original thread until the older target is found and focused', async (t) => {
  const page = await fixture(t);
  const work = 'a'.repeat(32);
  const invite = 'b'.repeat(32);
  const target = 'c'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (before)
            return Response.json({
              items: [
                {
                  comment_id: target,
                  repository: 7,
                  revision: 0,
                  actor: 'architect',
                  author_name: 'Architect',
                  assignment: work,
                  reply_to: invite,
                  body: 'The role’s actual reply from the older page.',
                  at: 1729000000,
                  edited: false,
                },
              ],
              next_before: 0,
            });
          return Response.json({
            items: [
              {
                comment_id: invite,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted the architect.',
                at: 1729000001,
                edited: false,
                contribution: {
                  source: work,
                  revision: 0,
                  state: 'responded',
                  profile: 'architect',
                  reply: target,
                  body: 'Consulted the architect.',
                  roles: [],
                  replies: [],
                },
              },
            ],
            next_before: 5,
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const link = feed.getByRole('link', {name: 'Read the role'});
  await link.waitFor();
  await link.click();
  await feed.locator(`#comment-${target}`).getByText('The role’s actual reply from the older page.').waitFor();
  assert.equal(await page.evaluate(() => location.hash), '');
  await page.waitForFunction((id) => document.activeElement?.id === id, `comment-${target}`);
  assert.ok(
    (await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.url.includes('/api/factory/feed/replies?') && call.url.includes('before=')
        ).length
    )) >= 1,
    'the older replies page is read'
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a role reply link explains honestly when the target is unavailable or out of view', async (t) => {
  const page = await fixture(t);
  const work = 'd'.repeat(32);
  const invite = 'e'.repeat(32);
  const target = 'f'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      const state = {commentGone: true};
      (window as unknown as {roleReplyProbe?: unknown}).roleReplyProbe = state;
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [
              {
                comment_id: invite,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted the architect.',
                at: 1729000001,
                edited: false,
                contribution: {
                  source: work,
                  revision: 0,
                  state: 'responded',
                  profile: 'architect',
                  reply: target,
                  body: 'Consulted the architect.',
                  roles: [],
                  replies: [],
                },
              },
            ],
            next_before: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${target}`) && call.method === 'GET') {
          if (state.commentGone) return new Response(null, {status: 404});
          return Response.json({
            comment_id: target,
            repository: 7,
            revision: 0,
            author_name: 'Architect',
            at: 1729000000,
            edited: false,
            body: 'The role’s reply lives outside this thread’s view.',
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const link = feed.getByRole('link', {name: 'Read the role'});
  await link.waitFor();
  await link.click();
  await feed
    .getByText('The role’s reply could not be read here; it may have been removed or is not shared with you.', {
      exact: false,
    })
    .waitFor();
  assert.equal(await feed.locator(`#comment-${target}`).count(), 0);
  assert.equal(await page.evaluate(() => location.hash), '');
  await page.evaluate(() => {
    const probe = (window as unknown as {roleReplyProbe?: {commentGone: boolean}}).roleReplyProbe;
    if (probe) probe.commentGone = false;
  });
  await link.click();
  await feed
    .getByText('The role’s reply is not in this thread under the current view. Clear the filters', {exact: false})
    .waitFor();
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('same-second thread replies keep the server canonical order across commentId orderings', async (t) => {
  // Production request+reply pairs share one Unix-second `at`. The server
  // thread API returns seq DESC (newest first), so the payload lists the
  // reply before the request; the client must still render the request
  // first, whichever commentId sorts first.
  const work = '1'.repeat(32);
  const at = 1729000000;
  const expectedDatetime = new Date(at * 1000).toISOString();
  for (const [request, reply] of [
    ['a'.repeat(32), 'f'.repeat(32)],
    ['f'.repeat(32), 'a'.repeat(32)],
  ]) {
    const page = await fixture(t);
    await page.evaluate(
      ({work, request, reply, at}) => {
        window.drawerFixture.api.setPresentation('settings');
        window.drawerFixture.setReply(async (call) => {
          if (call.url.endsWith('/api/factory/feed/subscription'))
            return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
          if (call.url.includes('/api/repositories?q='))
            return Response.json({
              items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
              next_cursor: '',
            });
          if (call.url.includes('/api/factory/feed?'))
            return Response.json({
              entries: [
                {
                  kind: 'record',
                  id: work,
                  repository: 7,
                  seq: 1,
                  actor: 'builder',
                  author: 'Builder',
                  status: 'assigned',
                  summary: 'Working on the demo.',
                },
              ],
              truncated: false,
            });
          if (call.url.includes('/api/factory/feed/replies?'))
            // Actual server payload order: seq DESC, reply (newer) first.
            return Response.json({
              items: [
                {
                  comment_id: reply,
                  repository: 7,
                  revision: 0,
                  author: '2',
                  author_name: 'Fixer',
                  at,
                  edited: false,
                  deleted: false,
                  revoked: false,
                  body: 'Fixed in the follow-up.',
                  intake: 'recorded',
                  judgment: 'applied',
                  route: 'discussion',
                  draft: 'pending',
                  effect: 'pending',
                  issue: 0,
                },
                {
                  comment_id: request,
                  repository: 7,
                  revision: 0,
                  author: '1',
                  author_name: 'Ada',
                  at,
                  edited: false,
                  deleted: false,
                  revoked: false,
                  body: 'Can this cover substitute approvers?',
                  intake: 'recorded',
                  judgment: 'applied',
                  route: 'discussion',
                  draft: 'pending',
                  effect: 'pending',
                  issue: 0,
                },
              ],
              next_before: 0,
            });
          return null;
        });
      },
      {work, request, reply, at}
    );
    await refresh(page);
    await page.evaluate(() => {
      window.drawerFixture.mountFeed();
    });
    const feed = page.locator('soda-factory-feed');
    await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
    await feed.locator(`#comment-${request}`).getByText('Can this cover substitute approvers?').waitFor();
    await feed.locator(`#comment-${reply}`).getByText('Fixed in the follow-up.').waitFor();
    const order = await feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
    assert.deepEqual(order, [`comment-${request}`, `comment-${reply}`]);
    // Attribution and timestamps survive the reorder intact.
    assert.equal(await feed.locator(`#comment-${request} .soda-feed-comment-who strong`).textContent(), 'Ada');
    assert.equal(await feed.locator(`#comment-${reply} .soda-feed-comment-who strong`).textContent(), 'Fixer');
    assert.equal(await feed.locator(`#comment-${request} time`).getAttribute('datetime'), expectedDatetime);
    assert.equal(await feed.locator(`#comment-${reply} time`).getAttribute('datetime'), expectedDatetime);
    assert.equal(
      await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length),
      0
    );
  }
});

test('older thread pages prepend and refreshes preserve canonical reply order', async (t) => {
  const page = await fixture(t);
  const work = '2'.repeat(32);
  // Canonical seq order is oldest, middle, newest; oldest and middle share
  // one Unix second across the page boundary. Ids are chosen so a
  // commentId tiebreak would misorder them (newest sorts first).
  const oldest = 'f'.repeat(32);
  const middle = 'a'.repeat(32);
  const newest = '5'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, oldest, middle, newest, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const discussion = (comment_id: string, author_name: string, body: string, stamp: number) => ({
        comment_id,
        repository: 7,
        revision: 0,
        author: '1',
        author_name,
        at: stamp,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Every page arrives seq DESC, as the server sends it.
          if (before)
            return Response.json({
              items: [discussion(oldest, 'Ada', 'Can this cover substitute approvers?', at)],
              next_before: 0,
            });
          return Response.json({
            items: [
              discussion(newest, 'Fixer', 'Verified on the follow-up.', at + 10),
              discussion(middle, 'Builder', 'Covered in the candidate.', at),
            ],
            next_before: 99,
          });
        }
        return null;
      });
    },
    {work, oldest, middle, newest, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#comment-${middle}`).getByText('Covered in the candidate.').waitFor();
  assert.deepEqual(await orderOf(), [`comment-${middle}`, `comment-${newest}`]);
  // Older-page prepend keeps ascending canonical order across the
  // same-second page boundary.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${oldest}`).getByText('Can this cover substitute approvers?').waitFor();
  assert.deepEqual(await orderOf(), [`comment-${oldest}`, `comment-${middle}`, `comment-${newest}`]);
  // A quiet feed refresh re-merges the newest page; dedup keeps every
  // canonical position instead of reordering.
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.').waitFor();
  assert.deepEqual(await orderOf(), [`comment-${oldest}`, `comment-${middle}`, `comment-${newest}`]);
  // Reload replace re-reads the newest page, then paging restores the
  // older reply in front again.
  await feed.getByRole('button', {name: 'Refresh replies'}).click();
  await feed.locator('.soda-feed-thread').getByText('Covered in the candidate.').waitFor();
  assert.deepEqual(await orderOf(), [`comment-${middle}`, `comment-${newest}`]);
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${oldest}`).getByText('Can this cover substitute approvers?').waitFor();
  assert.deepEqual(await orderOf(), [`comment-${oldest}`, `comment-${middle}`, `comment-${newest}`]);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a filter change during a role-reply lookup leaves the stale thread untouched', async (t) => {
  const page = await fixture(t);
  const work = 'a'.repeat(32);
  const invite = 'b'.repeat(32);
  const target = 'c'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?')) {
          const filtered = new URL(call.url, location.origin).searchParams.get('view') === 'decisions';
          return Response.json({
            entries: filtered
              ? []
              : [
                  {
                    kind: 'record',
                    id: work,
                    repository: 7,
                    seq: 1,
                    actor: 'builder',
                    author: 'Builder',
                    status: 'assigned',
                    summary: 'Working on the demo.',
                  },
                ],
            truncated: false,
          });
        }
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before)
            return Response.json({
              items: [
                {
                  comment_id: invite,
                  repository: 7,
                  revision: 0,
                  actor: 'builder',
                  author_name: 'Builder',
                  assignment: work,
                  reply_to: '',
                  body: 'Consulted the architect.',
                  at: 1729000001,
                  edited: false,
                  contribution: {
                    source: work,
                    revision: 0,
                    state: 'responded',
                    profile: 'architect',
                    reply: target,
                    body: 'Consulted the architect.',
                    roles: [],
                    replies: [],
                  },
                },
              ],
              next_before: 5,
            });
          (window as unknown as {replyHeld?: boolean}).replyHeld = true;
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {replyHolds?: Record<string, (response: Response) => void>};
            if (!holds.replyHolds) holds.replyHolds = {};
            (holds.replyHolds as Record<string, (response: Response) => void>)[target] = resolve;
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const link = feed.getByRole('link', {name: 'Read the role'});
  await link.waitFor();
  await link.click();
  // The older page stays in flight while the user moves on to a filter.
  await page.waitForFunction(() => (window as unknown as {replyHeld?: boolean}).replyHeld === true);
  await feed.getByRole('button', {name: 'Decisions', exact: true}).click();
  await feed.getByRole('heading', {name: 'No matching posts.', exact: true}).waitFor();
  await page.evaluate(
    ({target, work, invite}: {target: string; work: string; invite: string}) => {
      const holds = (window as unknown as {replyHolds: Record<string, (response: Response) => void>}).replyHolds;
      const release = holds[target];
      if (!release) throw Error('older replies GET was never issued');
      release(
        Response.json({
          items: [
            {
              comment_id: target,
              repository: 7,
              revision: 0,
              actor: 'architect',
              author_name: 'Architect',
              assignment: work,
              reply_to: invite,
              body: 'The late role reply the user left behind.',
              at: 1729000000,
              edited: false,
            },
          ],
          next_before: 0,
        })
      );
    },
    {target, work, invite}
  );
  await page.waitForTimeout(250);
  // The stale lookup resolves silently: no focus jump, no message write.
  assert.equal(await page.evaluate(() => document.activeElement?.id ?? ''), '');
  assert.equal(await feed.locator(`#comment-${target}`).count(), 0);
  assert.equal(await feed.getByText('Looking for the role’s reply…').count(), 0);
  await feed.getByRole('heading', {name: 'No matching posts.', exact: true}).waitFor();
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a failed older page during a role-reply lookup surfaces the error once', async (t) => {
  const page = await fixture(t);
  const work = 'd'.repeat(32);
  const invite = 'e'.repeat(32);
  const target = 'f'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (before) return Response.json({}, {status: 503});
          return Response.json({
            items: [
              {
                comment_id: invite,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted the architect.',
                at: 1729000001,
                edited: false,
                contribution: {
                  source: work,
                  revision: 0,
                  state: 'responded',
                  profile: 'architect',
                  reply: target,
                  body: 'Consulted the architect.',
                  roles: [],
                  replies: [],
                },
              },
            ],
            next_before: 5,
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const link = feed.getByRole('link', {name: 'Read the role'});
  await link.waitFor();
  await link.click();
  // The failed page reports its error instead of blaming the thread.
  await feed
    .locator('[data-role-reply-status]')
    .getByText('Replies could not be loaded. Try Refresh replies.', {exact: false})
    .waitFor();
  assert.equal(await feed.getByText('is not in this thread', {exact: false}).count(), 0);
  assert.equal(await feed.locator(`#comment-${target}`).count(), 0);
  await page.waitForTimeout(250);
  // The same older GET is never retried, and the missing reply is never probed.
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.url.includes('/api/factory/feed/replies?') && call.url.includes('before=')
        ).length
    ),
    1
  );
  assert.equal(
    await page.evaluate(
      (target) =>
        window.drawerFixture.calls.filter((call) => call.url.includes(`/api/factory/feed/comments/${target}`)).length,
      target
    ),
    0
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a late reply burst after a full thread load stays pageable with no gaps', async (t) => {
  const page = await fixture(t);
  const work = '1'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Every stamp ties, so only server order and cursors carry the
          // thread. A full 20-row load completes 1..20; then a 25-row burst
          // arrives with nothing removed. Every page carries a real
          // LIMIT-sized span: the quiet latest page 45..26 stays truncated,
          // and the older page 25..6 overlaps the kept head while the kept
          // 1..5 complete the coverage below its span.
          if (!before && document.body.dataset.burst !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 6), next_before: 6});
          return Response.json({items: descending(5, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  // Fully loaded: no older pages remain.
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  // The disjoint burst keeps its cursor, so the gap stays pageable.
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  // The overlapping older page splices into exact canonical order, and the
  // kept 1..5 complete the coverage below its span, so paging closes with no
  // gaps and nothing deleted.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(6)}`)
    .getByText('Thread reply 6.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 45}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a permission-denied older page during a role-reply lookup says so and keeps order', async (t) => {
  const page = await fixture(t);
  const work = '2'.repeat(32);
  const invite = '3'.repeat(32);
  const target = '4'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (before) return Response.json({}, {status: 403});
          return Response.json({
            items: [
              {
                comment_id: invite,
                repository: 7,
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                reply_to: '',
                body: 'Consulted the architect.',
                at: 1729000001,
                edited: false,
                contribution: {
                  source: work,
                  revision: 0,
                  state: 'responded',
                  profile: 'architect',
                  reply: target,
                  body: 'Consulted the architect.',
                  roles: [],
                  replies: [],
                },
              },
            ],
            next_before: 5,
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const link = feed.getByRole('link', {name: 'Read the role'});
  await link.waitFor();
  await link.click();
  await feed
    .locator('[data-role-reply-status]')
    .getByText('One or more subscribed projects are not visible to you.', {exact: false})
    .waitFor();
  assert.equal(await feed.getByText('is not in this thread', {exact: false}).count(), 0);
  // The loaded reply keeps its place; the denied page is attempted once.
  await feed.locator(`#comment-${invite}`).waitFor();
  assert.deepEqual(
    await feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id)),
    [`comment-${invite}`]
  );
  await page.waitForTimeout(250);
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.url.includes('/api/factory/feed/replies?') && call.url.includes('before=')
        ).length
    ),
    1
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('entering the library during a role-reply lookup leaves the stale lookup silent', async (t) => {
  const page = await fixture(t);
  const work = 'a'.repeat(32);
  const invite = 'b'.repeat(32);
  const target = 'c'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      const record = {
        id: 'd'.repeat(32),
        revision: 1,
        title: 'Decision record',
        body: 'Recorded rationale.',
        source_post: target,
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/library/item?')) return Response.json({item: record, history: []});
        if (call.url.includes('/api/factory/library?')) return Response.json({items: [record], next_cursor: ''});
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before)
            return Response.json({
              items: [
                {
                  comment_id: invite,
                  repository: 7,
                  revision: 0,
                  actor: 'builder',
                  author_name: 'Builder',
                  assignment: work,
                  reply_to: '',
                  body: 'Consulted the architect.',
                  at: 1729000001,
                  edited: false,
                  contribution: {
                    source: work,
                    revision: 0,
                    state: 'responded',
                    profile: 'architect',
                    reply: target,
                    body: 'Consulted the architect.',
                    roles: [],
                    replies: [],
                  },
                },
              ],
              next_before: 5,
            });
          (window as unknown as {replyHeld?: boolean}).replyHeld = true;
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {replyHolds?: Record<string, (response: Response) => void>};
            if (!holds.replyHolds) holds.replyHolds = {};
            (holds.replyHolds as Record<string, (response: Response) => void>)[target] = resolve;
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`[data-role-reply-open="${target}"]`).click();
  // The older page stays in flight while the user enters the library.
  await page.waitForFunction(() => (window as unknown as {replyHeld?: boolean}).replyHeld === true);
  await feed.getByRole('button', {name: 'Knowledge & decisions'}).click();
  await feed.locator('soda-factory-library').waitFor();
  await page.evaluate(
    ({target, work, invite}) => {
      const holds = (window as unknown as {replyHolds: Record<string, (response: Response) => void>}).replyHolds;
      const release = holds[target];
      if (!release) throw Error('older replies GET was never issued');
      release(
        Response.json({
          items: [
            {
              comment_id: target,
              repository: 7,
              revision: 0,
              actor: 'architect',
              author_name: 'Architect',
              assignment: work,
              reply_to: invite,
              body: 'The late role reply the user left behind.',
              at: 1729000000,
              edited: false,
            },
          ],
          next_before: 0,
        })
      );
    },
    {target, work, invite}
  );
  await page.waitForTimeout(250);
  // The library stays open with no status write and no focus steal.
  await feed.locator('soda-factory-library').waitFor();
  await feed.getByRole('button', {name: 'Back to activity'}).waitFor();
  assert.equal(await feed.locator(`#comment-${target}`).count(), 0);
  assert.equal(await feed.locator('section.soda-feed p[role="status"]').innerText(), '');
  assert.equal(await page.evaluate(() => document.activeElement?.id ?? ''), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('closing the drawer during a role-reply lookup leaves the stale lookup silent', async (t) => {
  const page = await fixture(t);
  const work = 'e'.repeat(32);
  const invite = 'f'.repeat(32);
  const target = '1'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before)
            return Response.json({
              items: [
                {
                  comment_id: invite,
                  repository: 7,
                  revision: 0,
                  actor: 'builder',
                  author_name: 'Builder',
                  assignment: work,
                  reply_to: '',
                  body: 'Consulted the architect.',
                  at: 1729000001,
                  edited: false,
                  contribution: {
                    source: work,
                    revision: 0,
                    state: 'responded',
                    profile: 'architect',
                    reply: target,
                    body: 'Consulted the architect.',
                    roles: [],
                    replies: [],
                  },
                },
              ],
              next_before: 5,
            });
          (window as unknown as {replyHeld?: boolean}).replyHeld = true;
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {replyHolds?: Record<string, (response: Response) => void>};
            if (!holds.replyHolds) holds.replyHolds = {};
            (holds.replyHolds as Record<string, (response: Response) => void>)[target] = resolve;
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const statusOf = () =>
    page.evaluate(() => {
      const anchor = document.querySelector('[data-role-reply-open]');
      return anchor?.closest('.soda-feed-consultation')?.querySelector('[data-role-reply-status]')?.textContent ?? null;
    });
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`[data-role-reply-open="${target}"]`).click();
  // The older page stays in flight while the user closes the drawer.
  await page.waitForFunction(() => (window as unknown as {replyHeld?: boolean}).replyHeld === true);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator('.soda-feed-drawer[data-open="false"]').waitFor();
  await page.evaluate(
    ({target, work, invite}) => {
      const holds = (window as unknown as {replyHolds: Record<string, (response: Response) => void>}).replyHolds;
      const release = holds[target];
      if (!release) throw Error('older replies GET was never issued');
      release(
        Response.json({
          items: [
            {
              comment_id: target,
              repository: 7,
              revision: 0,
              actor: 'architect',
              author_name: 'Architect',
              assignment: work,
              reply_to: invite,
              body: 'The late role reply the user left behind.',
              at: 1729000000,
              edited: false,
            },
          ],
          next_before: 0,
        })
      );
    },
    {target, work, invite}
  );
  await page.waitForTimeout(250);
  // The drawer stays closed with no status write and no focus steal: the
  // stale lookup never resolves its own status line.
  assert.equal(await feed.locator('.soda-feed-drawer[data-open="false"]').count(), 1);
  assert.equal(await statusOf(), 'Looking for the role’s reply…');
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
  assert.equal(await page.evaluate(() => document.activeElement?.id ?? ''), '');
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('selecting another role reply leaves the first lookup silent while the second focuses', async (t) => {
  const page = await fixture(t);
  const work = '2'.repeat(32);
  const inviteOne = '3'.repeat(32);
  const inviteTwo = '4'.repeat(32);
  const targetOne = '5'.repeat(32);
  const targetTwo = '6'.repeat(32);
  await page.evaluate(
    ({work, inviteOne, inviteTwo, targetOne, targetTwo}) => {
      window.drawerFixture.api.setPresentation('settings');
      const invite = (comment_id: string, target: string, body: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        actor: 'builder',
        author_name: 'Builder',
        assignment: work,
        reply_to: '',
        body,
        at: 1729000001,
        edited: false,
        contribution: {
          source: work,
          revision: 0,
          state: 'responded',
          profile: 'architect',
          reply: target,
          body,
          roles: [],
          replies: [],
        },
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before)
            return Response.json({
              items: [
                invite(inviteTwo, targetTwo, 'Consulted the tester.'),
                invite(inviteOne, targetOne, 'Consulted the architect.'),
              ],
              next_before: 5,
            });
          (window as unknown as {replyHeld?: boolean}).replyHeld = true;
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {replyHolds?: Record<string, (response: Response) => void>};
            if (!holds.replyHolds) holds.replyHolds = {};
            (holds.replyHolds as Record<string, (response: Response) => void>).shared = resolve;
          });
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${targetOne}`) && call.method === 'GET')
          return Response.json({
            comment_id: targetOne,
            revision: 0,
            repository: 7,
            author_name: 'Architect',
            at: 1729000000,
            edited: false,
            deleted: false,
            revoked: false,
            body: 'The first role reply the user left behind.',
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        return null;
      });
    },
    {work, inviteOne, inviteTwo, targetOne, targetTwo}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const statusOf = (target: string) =>
    page.evaluate((id) => {
      const anchor = document.querySelector(`[data-role-reply-open="${id}"]`);
      return anchor?.closest('.soda-feed-consultation')?.querySelector('[data-role-reply-status]')?.textContent ?? null;
    }, target);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).click();
  // The shared older page stays in flight while the user selects the other reply.
  await page.waitForFunction(() => (window as unknown as {replyHeld?: boolean}).replyHeld === true);
  await feed.locator(`[data-role-reply-open="${targetTwo}"]`).click();
  await page.evaluate(
    ({targetTwo, work, inviteTwo}) => {
      const holds = (window as unknown as {replyHolds: Record<string, (response: Response) => void>}).replyHolds;
      const release = holds.shared;
      if (!release) throw Error('older replies GET was never issued');
      release(
        Response.json({
          items: [
            {
              comment_id: targetTwo,
              repository: 7,
              revision: 0,
              actor: 'tester',
              author_name: 'Tester',
              assignment: work,
              reply_to: inviteTwo,
              body: 'The second role’s reply from the older page.',
              at: 1729000003,
              edited: false,
            },
          ],
          next_before: 0,
        })
      );
    },
    {targetTwo, work, inviteTwo}
  );
  await page.waitForFunction((id) => document.activeElement?.id === id, `comment-${targetTwo}`);
  // The second lookup completes and focuses; the first never resolves.
  assert.equal(await statusOf(targetTwo), '');
  assert.equal(await statusOf(targetOne), 'Looking for the role’s reply…');
  assert.equal(await feed.locator(`#comment-${targetOne}`).count(), 0);
  assert.equal(
    await page.evaluate(
      (id) => window.drawerFixture.calls.filter((call) => call.url.includes(`/api/factory/feed/comments/${id}`)).length,
      targetOne
    ),
    0
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('same-timestamp overlapped burst pages merge into exact canonical order', async (t) => {
  const page = await fixture(t);
  const work = '7'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--)
          rows.push({
            comment_id: seqId(seq),
            repository: 7,
            revision: 0,
            author: '1',
            author_name: 'Ada',
            at,
            edited: false,
            deleted: false,
            revoked: false,
            body: `Thread reply ${seq}.`,
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Actual descending server pages at one shared timestamp: the kept
          // window (1..20), the refresh burst (45..26), then the older page
          // (25..6) overlapping the kept window across 6..20. A real 20-row
          // page ending at seq 6 with seq 1..5 still stored carries
          // next_before 6 (LIMIT 21 sees the rows below); the client
          // derives exhaustion from its already-complete coverage instead.
          if (!before && document.body.dataset.burst !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 6), next_before: 6});
          return Response.json({items: descending(5, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  // The disjoint burst appends above the kept window and stays pageable.
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(45)}`)
    .getByText('Thread reply 45.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(25)}`)
    .getByText('Thread reply 25.')
    .waitFor();
  // The overlapped older page splices into exact canonical order: seq1..45
  // with no duplicates and no gaps, even though every timestamp ties.
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 45}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an overlapping quiet refresh with a changed payload shows the fresh content in canonical order', async (t) => {
  const page = await fixture(t);
  const work = '8'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number, body?: string) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: body ?? `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number, changed?: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--)
          rows.push(discussion(seq, seq === changed ? 'Returned consultation.' : undefined));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real 20-row DESC pages with the server's strict seq<before
          // paging: an ordinary older page can never repeat a served row,
          // so the changed payload arrives through an overlapping quiet
          // refresh over the retained window instead.
          if (before) return Response.json({items: descending(5, 1), next_before: 0});
          if (document.body.dataset.changed === 'true')
            return Response.json({items: descending(25, 6, 15), next_before: 6});
          return Response.json({items: descending(25, 6), next_before: 6});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(6)}`)
    .getByText('Thread reply 6.')
    .waitFor();
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  await page.evaluate(() => {
    document.body.dataset.changed = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  // The fresh page wins on content while the overlap keeps canonical order.
  await feed.locator(`#comment-${seqId(15)}`).getByText('Returned consultation.').waitFor();
  assert.equal(await feed.locator(`#comment-${seqId(15)}`).count(), 1);
  assert.equal(await feed.getByText('Thread reply 15.', {exact: false}).count(), 0);
  assert.deepEqual(
    await feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id)),
    Array.from({length: 25}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a deleted reply stays absent after a complete quiet refresh without reinsertion', async (t) => {
  const page = await fixture(t);
  const work = '9'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number, skip: number[] = []) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) {
          if (!skip.includes(seq)) rows.push(discussion(seq));
        }
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          // The server excludes deleted and revoked replies instead of
          // tombstoning them: the complete refresh page simply omits seq
          // 10 (deleted) and seq 5 (revoked).
          if (document.body.dataset.deleted === 'true')
            return Response.json({items: descending(20, 1, [10, 5]), next_before: 0});
          return Response.json({items: descending(20, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.deleted = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(11)}`)
    .getByText('Thread reply 11.')
    .waitFor();
  // The complete page excludes whatever the server omits: seq 10
  // (deleted) and seq 5 (revoked) never reinsert and their removed bodies
  // leave with them.
  assert.equal(await feed.locator(`#comment-${seqId(10)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(5)}`).count(), 0);
  assert.equal(await feed.getByText('Thread reply 10.', {exact: false}).count(), 0);
  assert.equal(await feed.getByText('Thread reply 5.', {exact: false}).count(), 0);
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`).filter(
      (id) => id !== `comment-${seqId(10)}` && id !== `comment-${seqId(5)}`
    )
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a same-timestamp multi-page burst lands in exact canonical order across disjoint pages', async (t) => {
  const page = await fixture(t);
  const work = 'a'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Kept window 1..20, then a >40-arrival same-second burst: the
          // quiet latest page 65..46, then the disjoint older page 45..26,
          // then the overlapping page 25..6, every timestamp ties through.
          // A real 20-row page ending at seq 6 with seq 1..5 still stored
          // carries next_before 6 (LIMIT 21 sees the rows below); the
          // client derives exhaustion from its already-complete coverage.
          if (!before && document.body.dataset.burst !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(65, 46), next_before: 46});
          if (before === '46') return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 6), next_before: 6});
          return Response.json({items: descending(5, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(65)}`)
    .getByText('Thread reply 65.')
    .waitFor();
  // The burst appends above the kept window and stays pageable.
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 46)}`),
  ]);
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(26)}`)
    .getByText('Thread reply 26.')
    .waitFor();
  // The disjoint older page anchors below the window whose cursor fetched
  // it — between its neighbors — instead of prepending in front of the
  // kept window. No duplicates, no gaps across the covered range.
  const disjoint = await orderOf();
  assert.deepEqual(disjoint, [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 40}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  assert.equal(new Set(disjoint).size, disjoint.length);
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(6)}`)
    .getByText('Thread reply 6.')
    .waitFor();
  // The later overlap repairs the full thread: seq 1..65 with no duplicates
  // and no gaps, even though every timestamp ties.
  const repaired = await orderOf();
  assert.deepEqual(
    repaired,
    Array.from({length: 65}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(new Set(repaired).size, repaired.length);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a truncated refresh never deletes cached entries outside its span', async (t) => {
  const page = await fixture(t);
  const work = 'b'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          // Every refresh page below stays truncated: older pages exist, so
          // cached entries outside each page's span must survive it.
          if (document.body.dataset.second === 'true')
            return Response.json({items: descending(69, 50), next_before: 50});
          if (document.body.dataset.burst === 'true')
            return Response.json({items: descending(65, 46), next_before: 46});
          return Response.json({items: descending(20, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(65)}`)
    .getByText('Thread reply 65.')
    .waitFor();
  // The truncated burst page covers only the newest window: the whole kept
  // window survives it, bodies intact.
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 46)}`),
  ]);
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.second = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(69)}`)
    .getByText('Thread reply 69.')
    .waitFor();
  // The overlapping truncated page slides the window forward without
  // deleting the entries it no longer covers: 46..49 stay with their text.
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 24}, (_, index) => `comment-${seqId(index + 46)}`),
  ]);
  await feed
    .locator(`#comment-${seqId(46)}`)
    .getByText('Thread reply 46.')
    .waitFor();
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed A→B→A role-reply race keeps the final selection focused', async (t) => {
  const page = await fixture(t);
  const work = 'c'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  const targetOne = seqId(1);
  const targetTwo = seqId(2);
  await page.evaluate(
    ({work, at, targetOne, targetTwo}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const invite = (seq: number, target: string, body: string) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        actor: 'builder',
        author_name: 'Builder',
        assignment: work,
        reply_to: '',
        body,
        at,
        edited: false,
        contribution: {
          source: work,
          revision: 0,
          state: 'responded',
          profile: 'architect',
          reply: target,
          body,
          roles: [],
          replies: [],
        },
      });
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real 20-row DESC newest page (22..3, next_before 3) with both
          // invites on it; the older tail (2..1) holds both targets and
          // stays pending while the user reselects.
          if (!before) {
            // Thread pages mix worker invite rows with plain discussion
            // rows, exactly as the server serves them, so the page keeps
            // both shapes instead of forcing the filler into the invite
            // shape (a fabricated contribution would render role replies).
            const rows: Array<ReturnType<typeof invite> | ReturnType<typeof discussion>> = [
              invite(22, targetOne, 'Consulted the architect.'),
              invite(21, targetTwo, 'Consulted the tester.'),
            ];
            for (let seq = 20; seq >= 3; seq--) rows.push(discussion(seq));
            return Response.json({items: rows, next_before: 3});
          }
          (window as unknown as {replyHeld?: boolean}).replyHeld = true;
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {replyHolds?: Record<string, (response: Response) => void>};
            if (!holds.replyHolds) holds.replyHolds = {};
            (holds.replyHolds as Record<string, (response: Response) => void>).shared = resolve;
          });
        }
        return null;
      });
    },
    {work, at, targetOne, targetTwo}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const statusOf = (target: string) =>
    page.evaluate((id) => {
      const anchor = document.querySelector(`[data-role-reply-open="${id}"]`);
      return anchor?.closest('.soda-feed-consultation')?.querySelector('[data-role-reply-status]')?.textContent ?? null;
    }, target);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).waitFor();
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).click();
  // The shared older page stays in flight while the user moves to the
  // other reply and then back to the first one.
  await page.waitForFunction(() => (window as unknown as {replyHeld?: boolean}).replyHeld === true);
  await feed.locator(`[data-role-reply-open="${targetTwo}"]`).click();
  await page.waitForFunction(
    (id) => {
      const anchor = document.querySelector(`[data-role-reply-open="${id}"]`);
      return (
        anchor?.closest('.soda-feed-consultation')?.querySelector('[data-role-reply-status]')?.textContent ===
        'Looking for the role’s reply…'
      );
    },
    targetTwo
  );
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).click();
  await page.evaluate(
    ({targetOne, targetTwo}) => {
      const holds = (window as unknown as {replyHolds: Record<string, (response: Response) => void>}).replyHolds;
      const release = holds.shared;
      if (!release) throw Error('older replies GET was never issued');
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at: 1729000000,
        edited: false,
        body: `Thread reply ${seq}.`,
      });
      release(Response.json({items: [discussion(2), discussion(1)], next_before: 0}));
    },
    {targetOne, targetTwo}
  );
  // The final A selection wins: it focuses while the stale A and B runs
  // stay silent and B never steals focus.
  await page.waitForFunction((id) => document.activeElement?.id === id, `comment-${targetOne}`);
  assert.equal(await statusOf(targetOne), '');
  assert.equal(await statusOf(targetTwo), 'Looking for the role’s reply…');
  assert.equal(await feed.locator(`#comment-${targetTwo}`).count(), 1);
  assert.deepEqual(
    await feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id)),
    Array.from({length: 22}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(
    await page.evaluate(
      ([one, two]) =>
        window.drawerFixture.calls.filter(
          (call) => call.url.includes(`/api/factory/feed/comments/${one}`) || call.url.includes(`/api/factory/feed/comments/${two}`)
        ).length,
      [targetOne, targetTwo]
    ),
    0
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an empty complete refresh clears a thread whose last reply was removed', async (t) => {
  const page = await fixture(t);
  const work = 'd'.repeat(32);
  const lone = 'e'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, lone, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          // A real single-row complete page; once the last reply is
          // deleted the server serves an empty complete page.
          if (document.body.dataset.deleted === 'true') return Response.json({items: [], next_before: 0});
          return Response.json({
            items: [
              {
                comment_id: lone,
                repository: 7,
                revision: 0,
                author: '1',
                author_name: 'Ada',
                at,
                edited: false,
                deleted: false,
                revoked: false,
                body: 'The only reply.',
                intake: 'recorded',
                judgment: 'applied',
                route: 'discussion',
                draft: 'pending',
                effect: 'pending',
                issue: 0,
              },
            ],
            next_before: 0,
          });
        }
        return null;
      });
    },
    {work, lone, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#comment-${lone}`).getByText('The only reply.').waitFor();
  await page.evaluate(() => {
    document.body.dataset.deleted = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  // The empty complete refresh authoritatively clears the thread instead
  // of keeping the removed body.
  await feed.getByText('No replies yet.', {exact: true}).waitFor();
  assert.equal(await feed.locator(`#comment-${lone}`).count(), 0);
  assert.equal(await feed.getByText('The only reply.', {exact: false}).count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a latest truncated refresh drops a cached head the server omits', async (t) => {
  const page = await fixture(t);
  const work = 'f'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real DESC pages: the newest page carries the 21st row's seq as
          // its cursor. Once the cached newest (seq 40) is deleted, the
          // latest page serves 39..20 and authoritatively covers the head
          // above it while the older unserved tail stays pageable.
          if (!before && document.body.dataset.deleted === 'true')
            return Response.json({items: descending(39, 20), next_before: 20});
          if (!before) return Response.json({items: descending(40, 21), next_before: 21});
          if (before === '21') return Response.json({items: descending(20, 1), next_before: 0});
          return Response.json({items: descending(19, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(21)}`)
    .getByText('Thread reply 21.')
    .waitFor();
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 40}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.deleted = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(39)}`)
    .getByText('Thread reply 39.')
    .waitFor();
  // The latest page covers the head above its newest row: the removed seq
  // 40 drops out while the older tail stays with its text. The already
  // cached 1..19 complete the coverage, so the client derives exhaustion
  // instead of paging the fresh cursor.
  assert.equal(await feed.locator(`#comment-${seqId(40)}`).count(), 0);
  assert.equal(await feed.getByText('Thread reply 40.', {exact: false}).count(), 0);
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 39}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a disjoint truncated refresh reconciles a removed head without losing the tail', async (t) => {
  const page = await fixture(t);
  const work = 'd'.repeat(32);
  const base = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, base}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      // Real-shaped rows: distinct stamps rising with the thread, served
      // newest-first with an opaque older-side cursor.
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at: base + seq,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real DESC paging: newest 60..41 (cursor 41), then 40..21
          // (cursor 21), then 20..1. Once every cached 41..60 reply is
          // removed server-side while 1..40 remain, the newest page serves
          // 40..21 — truncated and fully disjoint from everything kept.
          if (!before && document.body.dataset.deleted === 'true')
            return Response.json({items: descending(40, 21), next_before: 21});
          if (!before) return Response.json({items: descending(60, 41), next_before: 41});
          if (before === '41') return Response.json({items: descending(40, 21), next_before: 21});
          if (before === '21') return Response.json({items: descending(20, 1), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, base}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#comment-${seqId(41)}`).getByText('Thread reply 41.').waitFor();
  // Only the newest page is loaded: 60..41 with the older side pageable.
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 41)}`)
  );
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.deleted = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.locator(`#comment-${seqId(40)}`).getByText('Thread reply 40.').waitFor();
  // The disjoint page heads older than the kept head, so the kept head is
  // reconciled away: 21..40 in order, the fresh cursor keeps the older
  // side pageable, and no removed body persists.
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 21)}`)
  );
  assert.equal(await feed.getByText('Thread reply 60.', {exact: false}).count(), 0);
  assert.equal(await feed.getByText('Thread reply 41.', {exact: false}).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(60)}`).count(), 0);
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${seqId(1)}`).getByText('Thread reply 1.').waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 40}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a queued role reply never replays after a project switch', async (t) => {
  const page = await fixture(t);
  const work = 'e'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  const targetOne = seqId(1);
  const targetTwo = seqId(2);
  await page.evaluate(
    ({work, at, targetOne, targetTwo}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const invite = (seq: number, target: string, body: string) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        actor: 'builder',
        author_name: 'Builder',
        assignment: work,
        reply_to: '',
        body,
        at,
        edited: false,
        contribution: {
          source: work,
          revision: 0,
          state: 'responded',
          profile: 'architect',
          reply: target,
          body,
          roles: [],
          replies: [],
        },
      });
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        // The new project is empty; only the old project serves the record.
        if (call.url.includes('/api/factory/feed?')) {
          if (document.body.dataset.switched === 'true') return Response.json({entries: [], truncated: false});
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        }
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real 20-row DESC newest page (22..3, next_before 3) with both
          // invites on it; the older tail (2..1) holds both targets and
          // stays pending while the user reselects and switches projects.
          if (!before) {
            const rows: unknown[] = [invite(22, targetOne, 'Consulted the architect.'), invite(21, targetTwo, 'Consulted the tester.')];
            for (let seq = 20; seq >= 3; seq--) rows.push(discussion(seq));
            return Response.json({items: rows, next_before: 3});
          }
          (window as unknown as {replyHeld?: boolean}).replyHeld = true;
          return new Promise<Response>((resolve) => {
            const holds = window as unknown as {replyHolds?: Record<string, (response: Response) => void>};
            if (!holds.replyHolds) holds.replyHolds = {};
            (holds.replyHolds as Record<string, (response: Response) => void>).shared = resolve;
          });
        }
        return null;
      });
    },
    {work, at, targetOne, targetTwo}
  );
  await refresh(page);
  await page.evaluate(() => {
    // Record the transport the fixture mounts with so the test can switch
    // projects on the same live feed element through the real setter.
    const cls = customElements.get('soda-factory-feed') as unknown as {prototype: object};
    let target: object | null = cls.prototype;
    let descriptor: PropertyDescriptor | undefined;
    while (target && !descriptor) {
      descriptor = Object.getOwnPropertyDescriptor(target, 'context');
      target = Object.getPrototypeOf(target);
    }
    const original = descriptor?.set;
    if (!original || !descriptor) throw Error('feed context setter not found');
    Object.defineProperty(cls.prototype, 'context', {
      configurable: true,
      enumerable: descriptor.enumerable ?? false,
      set(value: unknown) {
        (window as unknown as {feedTransport?: unknown}).feedTransport = (
          value as {transport?: unknown}
        )?.transport;
        original.call(this, value);
      },
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).waitFor();
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).click();
  // The shared older page stays in flight while the user moves to the
  // other reply and then back to the first one, queueing A behind B.
  await page.waitForFunction(() => (window as unknown as {replyHeld?: boolean}).replyHeld === true);
  await feed.locator(`[data-role-reply-open="${targetTwo}"]`).click();
  await page.waitForFunction(
    (id) => {
      const anchor = document.querySelector(`[data-role-reply-open="${id}"]`);
      return (
        anchor?.closest('.soda-feed-consultation')?.querySelector('[data-role-reply-status]')?.textContent ===
        'Looking for the role’s reply…'
      );
    },
    targetTwo
  );
  await feed.locator(`[data-role-reply-open="${targetOne}"]`).click();
  // Switch projects before the shared page settles: the queued A selection
  // belongs to the old project and must never replay in the new one.
  await page.evaluate(() => {
    document.body.dataset.switched = 'true';
    const el = document.querySelector('soda-factory-feed') as unknown as {context: unknown};
    el.context = {
      transport: (window as unknown as {feedTransport?: unknown}).feedTransport,
      repository: '8',
    };
  });
  await page.waitForFunction(
    () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/feed?')).length >= 2
  );
  await page.evaluate(({targetOne, targetTwo}) => {
    const holds = (window as unknown as {replyHolds: Record<string, (response: Response) => void>}).replyHolds;
    const release = holds.shared;
    if (!release) throw Error('older replies GET was never issued');
    const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
    const discussion = (seq: number) => ({
      comment_id: seqId(seq),
      repository: 7,
      revision: 0,
      author: '1',
      author_name: 'Ada',
      at: 1729000000,
      edited: false,
      body: `Thread reply ${seq}.`,
    });
    void targetOne;
    void targetTwo;
    release(Response.json({items: [discussion(2), discussion(1)], next_before: 0}));
  }, {targetOne, targetTwo});
  // Let both stale runs and the drain settle, then prove nothing from the
  // old project leaked into the new one: no stale message, no focus theft,
  // no role-reply rows, and no per-target detail reads.
  await page.waitForTimeout(400);
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
  assert.equal(
    await page.evaluate(() => (document.activeElement?.id ?? '').startsWith('comment-')),
    false
  );
  assert.equal(await feed.locator('[data-role-reply-open]').count(), 0);
  assert.equal(
    await page.evaluate(
      ([one, two]) =>
        window.drawerFixture.calls.filter(
          (call) =>
            call.url.endsWith(`/api/factory/feed/comments/${one}`) ||
            call.url.endsWith(`/api/factory/feed/comments/${two}`)
        ).length,
      [targetOne, targetTwo]
    ),
    0
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a same-second disjoint refresh reconciles a removed head without timestamp inference', async (t) => {
  const page = await fixture(t);
  const work = 'd'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      // Every row shares one stamp: direction and removal must come from
      // cursors and server order alone, never from `at` comparison.
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real DESC paging where every stamp ties: newest 60..41 (cursor
          // 41), then 40..21 (cursor 21), then 20..1. Once every cached
          // 41..60 reply is removed server-side, the newest page serves
          // 40..21 — truncated and fully disjoint from everything kept.
          if (!before && document.body.dataset.deleted === 'true')
            return Response.json({items: descending(40, 21), next_before: 21});
          if (!before) return Response.json({items: descending(60, 41), next_before: 41});
          if (before === '41') return Response.json({items: descending(40, 21), next_before: 21});
          if (before === '21') return Response.json({items: descending(20, 1), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#comment-${seqId(41)}`).getByText('Thread reply 41.').waitFor();
  // Only the newest page is loaded: 60..41 with the older side pageable.
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 41)}`)
  );
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.deleted = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.locator(`#comment-${seqId(40)}`).getByText('Thread reply 40.').waitFor();
  // The tied stamps prove nothing, yet the disjoint page still reconciles:
  // 21..40 in order, the fresh cursor keeps the older side pageable, and
  // no removed body persists or resurrects.
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 21)}`)
  );
  assert.equal(await feed.getByText('Thread reply 60.', {exact: false}).count(), 0);
  assert.equal(await feed.getByText('Thread reply 41.', {exact: false}).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(60)}`).count(), 0);
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${seqId(1)}`).getByText('Thread reply 1.').waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 40}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an edited-timestamp refresh keeps canonical order with zero timestamp inference', async (t) => {
  const page = await fixture(t);
  const work = 'e'.repeat(32);
  const at = 1729000000;
  const editedAt = 1729000900;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at, editedAt}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      // `at` is mutable on edit: the refresh re-serves the same identities
      // with bumped stamps and edited bodies. Order and membership must
      // follow identities and server order only.
      const discussion = (seq: number, stamp: number, edited: boolean) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at: stamp,
        edited,
        deleted: false,
        revoked: false,
        body: edited ? `Thread reply ${seq} (edited).` : `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number, stamp: number, edited: boolean) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq, stamp, edited));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before && document.body.dataset.edited === 'true')
            return Response.json({items: descending(60, 41, editedAt, true), next_before: 41});
          if (!before) return Response.json({items: descending(60, 41, at, false), next_before: 41});
          if (before === '41') return Response.json({items: descending(40, 21, at, false), next_before: 21});
          if (before === '21') return Response.json({items: descending(20, 1, at, false), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at, editedAt}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#comment-${seqId(41)}`).getByText('Thread reply 41.').waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 41)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.edited = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.locator(`#comment-${seqId(60)}`).getByText('Thread reply 60 (edited).').waitFor();
  // Bumped stamps change content, never order or membership: the same 20
  // identities stay in canonical order with the fresh bodies, nothing
  // drops, and the older side stays pageable through the kept cursor.
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 41)}`)
  );
  await feed
    .locator(`#comment-${seqId(41)}`)
    .getByText('Thread reply 41 (edited).')
    .waitFor();
  assert.equal(await feed.getByText('Thread reply 60.', {exact: true}).count(), 0);
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${seqId(21)}`).getByText('Thread reply 21.').waitFor();
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${seqId(1)}`).getByText('Thread reply 1.').waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 60}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an authoritative older completion purges deleted oldest rows without stranding paging', async (t) => {
  const page = await fixture(t);
  const work = '6'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Every stamp ties, so only server order carries the thread.
          // Cache 1..20 complete; then 21..45 arrive while 1..5 are
          // deleted/revoked server-side.
          if (!before && document.body.dataset.mid !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 6), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.mid = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(45)}`)
    .getByText('Thread reply 45.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  // The authoritative older completion purges the omitted cached prefix
  // while the surviving tail keeps canonical order and paging closes.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(6)}`)
    .getByText('Thread reply 6.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 40}, (_, index) => `comment-${seqId(index + 6)}`)
  );
  assert.equal(await feed.locator(`#comment-${seqId(1)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(5)}`).count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an authoritative empty older page clears the covered cached tail', async (t) => {
  const page = await fixture(t);
  const work = '7'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Cache 1..20 complete; the quiet page 45..26 stays truncated
          // (seq 25 still stored below it). Everything below 26 is removed
          // before the older side is read, so it completes empty.
          if (!before && document.body.dataset.mid !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26' && document.body.dataset.stage === 'older')
            return Response.json({items: [], next_before: 0});
          if (before === '26') return Response.json({items: descending(25, 25), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  await page.evaluate(() => {
    document.body.dataset.mid = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(45)}`)
    .getByText('Thread reply 45.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  await page.evaluate(() => {
    document.body.dataset.stage = 'older';
  });
  await feed.getByRole('button', {name: 'Show more'}).click();
  await page.waitForFunction(
    () => document.querySelectorAll('soda-factory-feed .soda-feed-thread li.soda-feed-comment').length === 20
  );
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`)
  );
  assert.equal(await feed.locator(`#comment-${seqId(1)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(20)}`).count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a truncated older page keeps the genuinely unserved tail pageable', async (t) => {
  const page = await fixture(t);
  const work = '0'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Cache 1..20 complete; 21..65 arrive with nothing removed. The
          // older pages stay truncated until the bottom is served, so the
          // cached tail below each span must survive unpurged.
          if (!before && document.body.dataset.burst !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(65, 46), next_before: 46});
          if (before === '46') return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 6), next_before: 6});
          return Response.json({items: descending(5, 1), next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(65)}`)
    .getByText('Thread reply 65.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 46)}`),
  ]);
  // The disjoint truncated page covers nothing cached: the whole kept tail
  // stays and the older side stays pageable — no over-delete.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(26)}`)
    .getByText('Thread reply 26.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 40}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  await feed.locator(`#comment-${seqId(5)}`).getByText('Thread reply 5.').waitFor();
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  // The overlapping completion below it keeps the tail and closes paging.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(6)}`)
    .getByText('Thread reply 6.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 65}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('an overlapping older page drops a covered deleted row without misplacing the tail', async (t) => {
  const page = await fixture(t);
  const work = '3'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number, skip: number[] = []) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) {
          if (!skip.includes(seq)) rows.push(discussion(seq));
        }
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Cache 1..20 complete; then 21..45 arrive while seq 20 is
          // deleted/revoked. The older page overlaps the cached 5..19 while
          // new 21..25 sits above the last overlap, ending at seq 5 with
          // more below — a real truncated 20-row page.
          if (!before && document.body.dataset.mid !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 5, [20]), next_before: 5});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  await page.evaluate(() => {
    document.body.dataset.mid = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(45)}`)
    .getByText('Thread reply 45.')
    .waitFor();
  // The deleted seq 20 sat between two served rows of the same page, so it
  // drops instead of landing after the fresh span; the unserved 1..4 stay
  // and the already-complete coverage closes paging.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(21)}`)
    .getByText('Thread reply 21.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 19}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 5}, (_, index) => `comment-${seqId(index + 21)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  assert.equal(await feed.locator(`#comment-${seqId(20)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(4)}`).getByText('Thread reply 4.').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a complete disjoint older page purges the deleted cached tail and closes paging', async (t) => {
  const page = await fixture(t);
  const work = '9'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Every stamp ties, so only server order and cursors carry the
          // thread. Cache 1..20 complete; then 21..65 arrive while 1..20
          // are deleted/revoked server-side. The quiet latest page 65..46
          // stays truncated, the older page 45..26 stays truncated, and the
          // final 25..21 completes — each disjoint from the kept tail.
          if (!before && document.body.dataset.burst !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(65, 46), next_before: 46});
          if (before === '46') return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 21), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(65)}`)
    .getByText('Thread reply 65.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 46)}`),
  ]);
  // The disjoint truncated older page covers nothing cached: the whole kept
  // tail stays and the older side stays pageable — no over-delete.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed
    .locator(`#comment-${seqId(26)}`)
    .getByText('Thread reply 26.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 40}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  // The complete disjoint page enumerates everything below its anchor, so
  // the deleted 1..20 purge while surviving 21..65 keeps exact order and
  // paging closes.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await page.waitForFunction(
    () => document.querySelectorAll('soda-factory-feed .soda-feed-thread li.soda-feed-comment').length === 45
  );
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 45}, (_, index) => `comment-${seqId(index + 21)}`)
  );
  assert.equal(await feed.locator(`#comment-${seqId(1)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(20)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(21)}`).getByText('Thread reply 21.').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a complete disjoint older page below the head purges the deleted cached tail', async (t) => {
  const page = await fixture(t);
  const work = '8'.repeat(32);
  const at = 1729000000;
  const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
  await page.evaluate(
    ({work, at}) => {
      const seqId = (seq: number) => seq.toString(16).padStart(32, '0');
      const discussion = (seq: number) => ({
        comment_id: seqId(seq),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Thread reply ${seq}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const descending = (from: number, to: number) => {
        const rows = [];
        for (let seq = from; seq >= to; seq--) rows.push(discussion(seq));
        return rows;
      };
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Every stamp ties, so only server order and cursors carry the
          // thread. Cache 1..20 complete; then 21..45 arrive while 1..20
          // are deleted/revoked server-side. The quiet latest page 45..26
          // stays truncated (20 newest of 25 live rows), so the older page
          // lands directly below the head anchor; its 25..21 completion
          // then proves the deleted tail gone.
          if (!before && document.body.dataset.burst !== 'true')
            return Response.json({items: descending(20, 1), next_before: 0});
          if (!before) return Response.json({items: descending(45, 26), next_before: 26});
          if (before === '26') return Response.json({items: descending(25, 21), next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed
    .locator(`#comment-${seqId(1)}`)
    .getByText('Thread reply 1.')
    .waitFor();
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`)
  );
  await page.evaluate(() => {
    document.body.dataset.burst = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .locator(`#comment-${seqId(45)}`)
    .getByText('Thread reply 45.')
    .waitFor();
  assert.deepEqual(await orderOf(), [
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 1)}`),
    ...Array.from({length: 20}, (_, index) => `comment-${seqId(index + 26)}`),
  ]);
  // The truncated page directly below the head keeps the kept tail and stays
  // pageable — no over-delete.
  await feed.getByRole('button', {name: 'Show more'}).waitFor();
  // Its completion enumerates everything below the head anchor, so the
  // deleted 1..20 purge while surviving 21..45 keeps exact order and paging
  // closes.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await page.waitForFunction(
    () => document.querySelectorAll('soda-factory-feed .soda-feed-thread li.soda-feed-comment').length === 25
  );
  assert.deepEqual(
    await orderOf(),
    Array.from({length: 25}, (_, index) => `comment-${seqId(index + 21)}`)
  );
  assert.equal(await feed.locator(`#comment-${seqId(1)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(20)}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${seqId(21)}`).getByText('Thread reply 21.').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a nested recovery retires the submitted parent outcome instead of duplicating it', async (t) => {
  const page = await fixture(t);
  const work = '7'.repeat(32);
  const comment = 'd'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const pending = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      // The stored outcome starts pending; once the thread carries the
      // comment, the recovery read reports the assessed discussion state.
      const recovered = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const threadRow = () => ({
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Please add substitute approvers.',
        ...pending,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST') {
          document.body.dataset.posted = '1';
          return Response.json(pending);
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
          return Response.json(document.body.dataset.recovered === '1' ? recovered : pending);
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({items: document.body.dataset.posted === '1' ? [threadRow()] : [], next_before: 0});
        return null;
      });
    },
    {work, comment, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText('No replies yet.', {exact: true}).waitFor();
  // The submit stores the parent outcome while the thread is still empty.
  const draft = feed.locator(`textarea[data-feed-draft="record:${work}"]`);
  await draft.fill('Please add substitute approvers.');
  await feed.locator(`[data-comment-submit="record:${work}"]`).click();
  await feed
    .locator('.soda-feed-outcome')
    .getByText(/Saved\. JEV is assessing/)
    .first()
    .waitFor();
  // The post-submit thread reload recovers the same comment as a nested
  // row, so the check below starts nested with a pre-existing outcome.
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.recovered = '1';
  });
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  // The recovered thread row owns the single coherent control: the fresh
  // state shows once, the stale parent text is gone, and no second
  // outcome duplicates it above the thread.
  await nested.getByText('Kept as discussion.', {exact: true}).waitFor();
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(await feed.getByText('Kept as discussion.', {exact: true}).count(), 1);
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 1);
});

test('a feed refresh never resurrects a retired nested outcome', async (t) => {
  const page = await fixture(t);
  const work = '8'.repeat(32);
  const comment = 'f'.repeat(32);
  const mid = 'a'.repeat(32);
  const old = '9'.repeat(32);
  const intent = 'd' + 'b'.repeat(24);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, mid, old, intent, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const commentBase = {comment_id: comment, repository: 7, revision: 0, intake: 'recorded'};
      const pending = {
        ...commentBase,
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      // The recovered read carries a reviewable change, so the single
      // surface owns a Review control instead of a retry control. The
      // thread serves the same recovered lane once the server has it.
      const recoveredLane = {
        ...commentBase,
        judgment: 'applied',
        route: 'new_issue',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
        product_change: {
          intent,
          accepted_revision: 'd' + 'c'.repeat(24),
          state: 'review_required',
          can_review: true,
          verified: false,
          evidence_status: 'unavailable',
        },
      };
      const recovered = {...recoveredLane};
      const settled = (comment_id: string, body: string) => ({
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const threadRow = () => ({
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Please add substitute approvers.',
        ...(document.body.dataset.recovered === '1' ? recoveredLane : pending),
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST') {
          document.body.dataset.posted = '1';
          return Response.json(pending);
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
          return Response.json(document.body.dataset.recovered === '1' ? recovered : pending);
        if (call.url.includes('/api/factory/feed/replies?')) {
          if (document.body.dataset.posted !== '1') return Response.json({items: [], next_before: 0});
          // Real DESC paging: the newest page carries only the submitted
          // comment and ends at cursor 20; the older page resumes below it
          // with the settled tail — no synthetic overlap.
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before) return Response.json({items: [threadRow()], next_before: 20});
          if (before === '20')
            return Response.json({
              items: [settled(mid, 'Settled earlier.'), settled(old, 'Settled earlier tail.')],
              next_before: 0,
            });
          return Response.json({items: [], next_before: 0});
        }
        return null;
      });
    },
    {work, comment, mid, old, intent, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const nonGets = () =>
    page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length);
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText('No replies yet.', {exact: true}).waitFor();
  // The submit stores the parent outcome while the thread is still empty.
  const draft = feed.locator(`textarea[data-feed-draft="record:${work}"]`);
  await draft.fill('Please add substitute approvers.');
  await feed.locator(`[data-comment-submit="record:${work}"]`).click();
  await feed
    .locator('.soda-feed-outcome')
    .getByText(/Saved\. JEV is assessing/)
    .first()
    .waitFor();
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  // An explicit Refresh feed while the thread already renders the pending
  // comment must retire the redundant parent outcome instead of serving it
  // a second time: exactly one outcome, one retry control, reads only.
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  // The retired intake record is gone, so no later refresh can rehydrate it.
  await page.waitForFunction(
    (comment) => {
      const raws: string[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith('soda-feed-intake:v1:')) raws.push(localStorage.getItem(key) ?? '');
      }
      return raws.length > 0 && raws.every((raw) => !raw.includes(comment));
    },
    comment
  );
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 0);
  assert.equal(await nonGets(), 1);
  // The recovered thread row owns the single coherent Review control: the
  // fresh state shows once and no second outcome duplicates it.
  await page.evaluate(() => {
    document.body.dataset.recovered = '1';
  });
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await nested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor();
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(
    await feed.getByText('Review this proposed change before updating the accepted direction.', {exact: true}).count(),
    1
  );
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await nonGets(), 1);
  // A second Refresh feed still shows exactly one outcome with reads only:
  // the retired intake record must not rehydrate the parent duplicate.
  const replyCallCount = () =>
    page.evaluate(
      () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length
    );
  const seenReplies = await replyCallCount();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await page.waitForFunction((seen) => {
    const calls = (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls;
    return calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length > seen;
  }, seenReplies);
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await nested.locator('.soda-feed-outcome').count(), 1);
  assert.equal(
    await feed.getByText('Review this proposed change before updating the accepted direction.', {exact: true}).count(),
    1
  );
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await nonGets(), 1);
  // Real DESC paging evidence: the older page behind cursor 20 serves the
  // tail through Show more.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.getByText('Settled earlier tail.', {exact: true}).waitFor();
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  assert.ok(
    await page.evaluate(() =>
      window.drawerFixture.calls.some(
        (call) => call.url.includes('/api/factory/feed/replies?') && call.url.includes('before=20')
      )
    )
  );
  // The tail rows own their own settled surfaces; the submitted comment
  // still owns exactly one.
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 3);
  assert.equal(await nested.locator('.soda-feed-outcome').count(), 1);
  // A third Refresh feed keeps every surface single with reads only while
  // the quiet newest-page refresh preserves the paged tail.
  const seenBeforeThird = await replyCallCount();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await page.waitForFunction((seen) => {
    const calls = (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls;
    return calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length > seen;
  }, seenBeforeThird);
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 3);
  assert.equal(await nested.locator('.soda-feed-outcome').count(), 1);
  assert.equal(
    await feed.getByText('Review this proposed change before updating the accepted direction.', {exact: true}).count(),
    1
  );
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await nonGets(), 1);
  // A remounted feed (memory cleared, stored intake kept) converges back to
  // the single nested surface once the drawer thread reloads.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  const reloaded = page.locator('soda-factory-feed');
  await reloaded.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const reloadedNested = reloaded.locator(`#comment-${comment}`);
  await reloadedNested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor();
  await reloaded.getByRole('button', {name: 'Show more'}).waitFor();
  await page.waitForTimeout(500);
  assert.equal(await reloaded.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await reloadedNested.locator('.soda-feed-outcome').count(), 1);
  assert.equal(
    await reloaded
      .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
      .count(),
    1
  );
  assert.equal(await reloaded.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await reloaded.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await nonGets(), 1);
});

test('an unrelated older page never invalidates an in-flight retry for a cached comment', async (t) => {
  const page = await fixture(t);
  const work = '6'.repeat(32);
  const cached = 'b'.repeat(32);
  const mid = 'a'.repeat(32);
  const old = '9'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, cached, mid, old, at}) => {
      const discussion = (comment_id: string, judgment: string, body: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment,
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Newest page caches the watched pending comment plus one
          // settled row; the older page serves one unrelated settled row.
          if (!before)
            return Response.json({
              items: [
                discussion(cached, 'pending', 'Watch this cached reply.'),
                discussion(mid, 'applied', 'Settled earlier.'),
              ],
              next_before: 10,
            });
          if (before === '10')
            return Response.json({items: [discussion(old, 'applied', 'Unrelated older reply.')], next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${cached}`) && call.method === 'GET') {
          // Hold the retry GET open until the older page has landed, so the
          // older load races a genuinely in-flight retry.
          while (document.body.dataset.retryReleased !== '1') await new Promise((run) => setTimeout(run, 25));
          return Response.json(discussion(cached, 'applied', 'Watch this cached reply.'));
        }
        return null;
      });
    },
    {work, cached, mid, old, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const watched = feed.locator(`#comment-${cached}`);
  await watched.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await watched.getByRole('button', {name: 'Check progress', exact: true}).click();
  // The retry GET is genuinely in flight before the older page loads.
  await page.waitForFunction(
    (id) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length > 0,
    cached
  );
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#comment-${old}`).getByText('Unrelated older reply.').waitFor();
  // Only now does the retry resolve; the untouched cached row must still
  // accept its newer state instead of dropping as superseded.
  await page.evaluate(() => {
    document.body.dataset.retryReleased = '1';
  });
  await watched.getByText('Kept as discussion.', {exact: true}).waitFor();
  assert.equal(await watched.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await watched.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.deepEqual(await orderOf(), [`comment-${old}`, `comment-${mid}`, `comment-${cached}`]);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  const commentReads = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    cached
  );
  assert.equal(commentReads, 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a late progress GET after an authoritative removal never resurrects the outcome', async (t) => {
  const page = await fixture(t);
  const work = '5'.repeat(32);
  const comment = 'e'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, at}) => {
      const pending = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const recovered = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const threadRow = () => ({
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Please add substitute approvers.',
        ...pending,
      });
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/comments') && call.method === 'POST') {
          document.body.dataset.posted = '1';
          return Response.json(pending);
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET') {
          // The default recovery read reports the stored pending state; the
          // gated retry holds open until the authoritative refresh has
          // landed, so it resolves late against the removed row.
          if (document.body.dataset.gated !== '1') return Response.json(pending);
          while (document.body.dataset.progressReleased !== '1') await new Promise((run) => setTimeout(run, 25));
          return Response.json(recovered);
        }
        if (call.url.includes('/api/factory/feed/replies?'))
          // The post-submit thread reload recovers the submitted comment as
          // a nested row; the authoritative refresh afterwards serves the
          // complete thread without it once it is removed server-side.
          return Response.json({
            items: document.body.dataset.posted === '1' && document.body.dataset.removed !== '1' ? [threadRow()] : [],
            next_before: 0,
          });
        return null;
      });
    },
    {work, comment, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByText('No replies yet.', {exact: true}).waitFor();
  // The submit stores a pending parent outcome while the thread is empty.
  const draft = feed.locator(`textarea[data-feed-draft="record:${work}"]`);
  await draft.fill('Please add substitute approvers.');
  await feed.locator(`[data-comment-submit="record:${work}"]`).click();
  await feed
    .locator('.soda-feed-outcome')
    .getByText(/Saved\. JEV is assessing/)
    .first()
    .waitFor();
  // The post-submit thread reload recovers the same comment as a nested row
  // carrying its own Check progress control.
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.gated = '1';
  });
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  // The gated retry is genuinely in flight before the removal lands; the
  // earlier recovery read already reported the stored pending state.
  await page.waitForFunction(
    (id) =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length >= 2,
    comment
  );
  await page.evaluate(() => {
    document.body.dataset.removed = '1';
  });
  await feed.getByRole('button', {name: 'Refresh replies'}).click();
  // The authoritative refresh drops the removed row while the gated retry
  // is still held open.
  await page.waitForFunction(
    (id) => document.querySelectorAll(`soda-factory-feed #comment-${id}`).length === 0,
    comment
  );
  await feed.getByText('No replies yet.', {exact: true}).waitFor();
  // Only now does the late retry resolve; the stored outcome keeps its
  // submitted pending state instead of adopting the removed row.
  await page.evaluate(() => {
    document.body.dataset.progressReleased = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.locator(`#comment-${comment}`).count(), 0);
  assert.equal(await feed.getByText('Kept as discussion.', {exact: true}).count(), 0);
  assert.equal(
    await feed
      .locator('.soda-feed-outcome')
      .getByText(/Saved\. JEV is assessing/)
      .count(),
    1
  );
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 1);
  const commentReads = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    comment
  );
  assert.equal(commentReads, 2);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 1);
});

test('a recovery that lands before an older page survives the older merge', async (t) => {
  const page = await fixture(t);
  const work = '4'.repeat(32);
  const cached = 'b'.repeat(32);
  const mid = 'a'.repeat(32);
  const old = '9'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, cached, mid, old, at}) => {
      const discussion = (comment_id: string, judgment: string, body: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment,
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Real DESC paging: the newest page ends at cursor 10 and the
          // older page resumes below it. The older page is held open until
          // the progress recovery has already landed.
          if (!before)
            return Response.json({
              items: [
                discussion(cached, 'pending', 'Watch this cached reply.'),
                discussion(mid, 'applied', 'Settled earlier.'),
              ],
              next_before: 10,
            });
          if (before === '10') {
            while (document.body.dataset.pageReleased !== '1') await new Promise((run) => setTimeout(run, 25));
            return Response.json({items: [discussion(old, 'applied', 'Unrelated older reply.')], next_before: 0});
          }
          return Response.json({items: [], next_before: 0});
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${cached}`) && call.method === 'GET')
          return Response.json(discussion(cached, 'applied', 'Watch this cached reply.'));
        return null;
      });
    },
    {work, cached, mid, old, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-comment')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const watched = feed.locator(`#comment-${cached}`);
  await watched.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  // The older page starts first and stays in flight while the recovery lands.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await page.waitForFunction(
    () => window.drawerFixture.calls.filter((call) => call.url.includes('before=10')).length > 0
  );
  await watched.getByRole('button', {name: 'Check progress', exact: true}).click();
  // The recovery completes while the older page is still in flight.
  await watched.getByText('Kept as discussion.', {exact: true}).waitFor();
  assert.equal(await watched.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  await page.evaluate(() => {
    document.body.dataset.pageReleased = '1';
  });
  await feed.locator(`#comment-${old}`).getByText('Unrelated older reply.').waitFor();
  // The older merge keeps the recovered state instead of overwriting it
  // with its stale pre-await snapshot.
  assert.equal(await watched.getByText('Kept as discussion.', {exact: true}).count(), 1);
  assert.equal(await watched.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.deepEqual(await orderOf(), [`comment-${old}`, `comment-${mid}`, `comment-${cached}`]);
  assert.equal(await feed.getByRole('button', {name: 'Show more'}).count(), 0);
  const commentReads = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    cached
  );
  assert.equal(commentReads, 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed second outcome never overwrites a newer check of the first comment', async (t) => {
  const page = await fixture(t);
  const work = 'a0'.repeat(16);
  const commentA = 'ab'.repeat(16);
  const commentB = 'cd'.repeat(16);
  await page.evaluate(
    ({work, commentA, commentB}) => {
      window.drawerFixture.api.setPresentation('settings');
      const pending = (comment_id: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const appliedDiscussion = (comment_id: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${commentA}`) && call.method === 'GET') {
          if (document.body.dataset.checkedA === '1') return Response.json(appliedDiscussion(commentA));
          return Response.json(pending(commentA));
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${commentB}`) && call.method === 'GET') {
          if (document.body.dataset.holdB === '1') {
            while (document.body.dataset.releaseB !== '1') await new Promise((run) => setTimeout(run, 25));
          }
          return Response.json(pending(commentB));
        }
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({items: [], next_before: 0});
        return null;
      });
      const intakeKey = `soda-feed-intake:v1:fixture-generation:record:${work}`;
      window.localStorage.setItem(
        intakeKey,
        JSON.stringify([
          {commentId: commentA, key: 'k1'},
          {commentId: commentB, key: 'k2'},
        ])
      );
    },
    {work, commentA, commentB}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`[data-comment-retry="${commentA}"]`).first().waitFor();
  await feed.locator(`[data-comment-retry="${commentB}"]`).first().waitFor();
  await feed.getByText(/Saved\. JEV is assessing/).first().waitFor();
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 2);
  await page.evaluate(() => {
    document.body.dataset.holdB = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await page.waitForFunction(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length >= 3,
    commentB
  );
  await page.evaluate(() => {
    document.body.dataset.checkedA = '1';
  });
  await feed.locator(`[data-comment-retry="${commentA}"]`).click();
  await feed.getByText('Kept as discussion.', {exact: true}).waitFor();
  assert.equal(await feed.getByText('Kept as discussion.', {exact: true}).count(), 1);
  await page.evaluate(() => {
    document.body.dataset.releaseB = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.getByText('Kept as discussion.', {exact: true}).count(), 1);
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 1);
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 2);
  assert.equal(await feed.locator(`[data-comment-retry="${commentA}"]`).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${commentB}"]`).count(), 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a closed drawer refresh updates the cached thread instead of reopening stale', async (t) => {
  const page = await fixture(t);
  const work = 'ef'.repeat(16);
  const comment = 'be'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const pending = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const applied = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const threadRow = (advanced: boolean) => ({
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Please add substitute approvers.',
        ...(advanced ? applied : pending),
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({items: [threadRow(document.body.dataset.advanced === '1')], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
          return Response.json(document.body.dataset.advanced === '1' ? applied : pending);
        return null;
      });
      const intakeKey = `soda-feed-intake:v1:fixture-generation:record:${work}`;
      window.localStorage.setItem(intakeKey, JSON.stringify([{commentId: comment, key: 'k1'}]));
    },
    {work, comment, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-comment-retry="${comment}"]`).first().waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await page.evaluate(() => {
    document.body.dataset.advanced = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length >= 2,
    comment
  );
  await page.waitForTimeout(500);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const reopened = feed.locator(`#comment-${comment}`);
  await reopened.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await reopened.getByText('Kept as discussion.', {exact: true}).waitFor();
  assert.equal(await reopened.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('explicit tombstones and access loss retire parent and thread without resurrection', async (t) => {
  const page = await fixture(t);
  const work = '12'.repeat(16);
  const nestedGone = '34'.repeat(16);
  const parentGone = '56'.repeat(16);
  const settled = '78'.repeat(16);
  const oldTail = '9a'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, nestedGone, parentGone, settled, oldTail, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const pending = (comment_id: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const settledRow = (comment_id: string, body: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      const pendingRow = (comment_id: string, body: string) => ({
        ...settledRow(comment_id, body),
        judgment: 'pending',
        route: 'existing_work',
        effect: 'pending',
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (!before) {
            if (document.body.dataset.tombstoned === '1')
              return Response.json({items: [settledRow(settled, 'Settled tail.')], next_before: 20});
            return Response.json({
              items: [pendingRow(nestedGone, 'Watch this pending reply.'), settledRow(settled, 'Settled tail.')],
              next_before: 20,
            });
          }
          if (before === '20') return Response.json({items: [settledRow(oldTail, 'Older tail.')], next_before: 0});
          return Response.json({items: [], next_before: 0});
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${nestedGone}`) && call.method === 'GET') {
          if (document.body.dataset.holdNested === '1' && document.body.dataset.tombstoned !== '1') {
            while (document.body.dataset.releaseNested !== '1')
              await new Promise((run) => setTimeout(run, 25));
            return Response.json(pending(nestedGone));
          }
          if (document.body.dataset.tombstoned === '1')
            return Response.json({comment_id: nestedGone, deleted: true});
          return Response.json(pending(nestedGone));
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${parentGone}`) && call.method === 'GET') {
          if (document.body.dataset.tombstoned === '1') return new Response(null, {status: 403});
          return Response.json(pending(parentGone));
        }
        return null;
      });
      const intakeKey = `soda-feed-intake:v1:fixture-generation:record:${work}`;
      window.localStorage.setItem(
        intakeKey,
        JSON.stringify([
          {commentId: parentGone, key: 'p1'},
          {commentId: nestedGone, key: 'n1'},
        ])
      );
    },
    {work, nestedGone, parentGone, settled, oldTail, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-comment-retry="${nestedGone}"]`).first().waitFor();
  await feed.locator(`[data-comment-retry="${parentGone}"]`).first().waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const nested = feed.locator(`#comment-${nestedGone}`);
  await nested.getByText('Watch this pending reply.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.holdNested = '1';
  });
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  await page.waitForFunction(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length > 0,
    nestedGone
  );
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await page.evaluate(() => {
    document.body.dataset.tombstoned = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await page.waitForFunction(
    (ids) => {
      const calls = (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls;
      const gone = ids as unknown as string[];
      return gone.every(
        (id) => calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length >= 2
      );
    },
    [parentGone, nestedGone]
  );
  await page.evaluate(() => {
    document.body.dataset.releaseNested = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.locator(`#comment-${nestedGone}`).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${nestedGone}"]`).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${parentGone}"]`).count(), 0);
  assert.equal(await feed.locator(`#comment-${settled}`).count(), 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#comment-${settled}`).getByText('Settled tail.', {exact: true}).waitFor();
  assert.equal(await feed.locator(`#comment-${nestedGone}`).count(), 0);
  const seenReplies = await page.evaluate(
    () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length
  );
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await page.waitForFunction(
    (seen) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.includes('/api/factory/feed/replies?')
      ).length > seen,
    seenReplies
  );
  await page.waitForTimeout(500);
  assert.equal(await feed.locator(`#comment-${nestedGone}`).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${nestedGone}"]`).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${parentGone}"]`).count(), 0);
  assert.equal(await feed.locator(`#comment-${settled}`).count(), 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a second server advance after retirement still surfaces on refresh and reopen', async (t) => {
  const page = await fixture(t);
  const work = 'ec'.repeat(16);
  const comment = 'db'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const pending = {
        comment_id: comment,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const advanced = {...pending, judgment: 'applied', route: 'discussion'};
      // The second advance carries a reviewable change, so the thread row
      // owns a Review control instead of the settled discussion text.
      const lane = {
        ...pending,
        judgment: 'applied',
        route: 'new_issue',
        product_change: {
          intent: 'd' + 'b'.repeat(24),
          accepted_revision: 'd' + 'c'.repeat(24),
          state: 'review_required',
          can_review: true,
          verified: false,
          evidence_status: 'unavailable',
        },
      };
      const pipeline = () => {
        const stage = document.body.dataset.stage ?? '0';
        if (stage === '2') return lane;
        if (stage === '1') return advanced;
        return pending;
      };
      const threadRow = () => ({
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Please add substitute approvers.',
        ...pipeline(),
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({items: [threadRow()], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
          return Response.json(pipeline());
        return null;
      });
    },
    {work, comment, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const commentReads = () =>
    page.evaluate(
      (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
      comment
    );
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // Seed the stored intake record only after mount: nothing unsettled ever
  // renders, so no background poll can retire or refresh the row — the two
  // Refresh feeds below are the only reads that touch it.
  await page.evaluate(
    ({work, comment}) => {
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:record:${work}`,
        JSON.stringify([{commentId: comment, key: 'k1'}])
      );
    },
    {work, comment}
  );
  // The first advance retires the intake record and updates the cached
  // thread row while the drawer stays closed.
  await page.evaluate(() => {
    document.body.dataset.stage = '1';
  });
  const seenFirst = await commentReads();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction(
    ([id, seen]) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length > (seen as number),
    [comment, seenFirst]
  );
  await page.waitForFunction(
    (id) => {
      const raws: string[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith('soda-feed-intake:v1:')) raws.push(localStorage.getItem(key) ?? '');
      }
      return raws.length > 0 && raws.every((raw) => !raw.includes(id));
    },
    comment
  );
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await nested.getByText('Kept as discussion.', {exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  // The second advance must still surface: the retired row stays watched,
  // so this refresh re-reads it and the reopen shows the current state.
  await page.evaluate(() => {
    document.body.dataset.stage = '2';
  });
  const seenGets = await commentReads();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await page.waitForFunction(
    ([id, seen]) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length > (seen as number),
    [comment, seenGets]
  );
  await page.waitForTimeout(500);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await nested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor();
  assert.equal(await feed.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await nested.locator('.soda-feed-outcome').count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 1);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a replies page in flight across a direct-GET tombstone never resurrects the removed row', async (t) => {
  const page = await fixture(t);
  const work = '3c'.repeat(16);
  const gone = '4d'.repeat(16);
  const kept = '5e'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, gone, kept, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const row = (comment_id: string, body: string) => ({
        comment_id,
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?')) {
          // The refreshed page holds open; when released it serves the
          // stale pre-removal snapshot with the removed row still listed.
          if (
            document.body.dataset.holdPage === '1' &&
            !new URL(call.url, location.origin).searchParams.get('before')
          ) {
            while (document.body.dataset.releasePage !== '1') await new Promise((run) => setTimeout(run, 25));
          }
          return Response.json({items: [row(gone, 'This reply will be removed.'), row(kept, 'This reply stays.')], next_before: 0});
        }
        if (call.url.endsWith(`/api/factory/feed/comments/${gone}`) && call.method === 'GET') {
          if (document.body.dataset.tombstoned === '1') return Response.json({comment_id: gone, deleted: true});
          return Response.json({
            comment_id: gone,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            route: 'existing_work',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        }
        return null;
      });
    },
    {work, gone, kept, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const removed = feed.locator(`#comment-${gone}`);
  await removed.getByText('This reply will be removed.', {exact: true}).waitFor();
  await removed.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  // Start the replies page first and keep it in flight.
  await page.evaluate(() => {
    document.body.dataset.holdPage = '1';
  });
  await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
  await page.waitForFunction(
    () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/feed/replies?')).length > 1
  );
  // The direct-GET tombstone lands while the page is still in flight.
  await page.evaluate(() => {
    document.body.dataset.tombstoned = '1';
  });
  await removed.getByRole('button', {name: 'Check progress', exact: true}).click();
  await page.waitForFunction(
    (id) => document.querySelectorAll(`soda-factory-feed #comment-${id}`).length === 0,
    gone
  );
  // Only now does the held page complete with the stale snapshot: the
  // removed row must stay removed instead of resurrecting.
  await page.evaluate(() => {
    document.body.dataset.releasePage = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.locator(`#comment-${gone}`).count(), 0);
  assert.equal(await feed.locator(`#comment-${kept}`).count(), 1);
  await feed.getByText('This reply stays.', {exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a removed post carrying pipeline never upserts progress or Check progress', async (t) => {
  const page = await fixture(t);
  const target = (20).toString(16).padStart(32, '0');
  await page.evaluate(({target}) => {
    window.drawerFixture.api.setPresentation('settings');
    const entries: Record<string, unknown>[] = [];
    for (let seq = 20; seq >= 1; seq--) {
      entries.push({
        kind: 'post',
        id: seq.toString(16).padStart(32, '0'),
        repository: 7,
        seq,
        actor: 'Builder',
        author: 'Ada',
        project: 'soda-tester/demo',
        at: 1720000000 + seq,
        status: 'assessing',
      });
    }
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({entries, truncated: false});
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
        if (comment === target)
          return Response.json({
            comment_id: comment,
            repository: 7,
            revision: 0,
            deleted: true,
            intake: 'recorded',
            judgment: 'pending',
            route: 'existing_work',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        return Response.json({
          comment_id: comment,
          repository: 7,
          revision: 0,
          author_name: 'Ada',
          at: 1720000000,
          edited: false,
          deleted: false,
          revoked: false,
          body: `Discussion post ${parseInt(comment, 16)}.`,
          intake: 'recorded',
          judgment: 'applied',
          route: 'discussion',
          draft: 'pending',
          effect: 'pending',
          issue: 0,
        });
      }
      return null;
    });
  }, {target});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor();
  await feed.getByText('Discussion post 19.', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-post').count(), 20);
  assert.equal(await feed.locator(`[data-comment-retry="${target}"]`).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a bodyless comment without flags stays unreadable and never purges', async (t) => {
  const page = await fixture(t);
  const work = 'ab'.repeat(16);
  const comment = 'cd'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [
              {
                comment_id: comment,
                repository: 7,
                revision: 0,
                author: '1',
                author_name: 'Ada',
                at,
                edited: false,
                deleted: false,
                revoked: false,
                body: 'Hold for malformed check.',
                intake: 'recorded',
                judgment: 'pending',
                route: 'existing_work',
                draft: 'pending',
                effect: 'pending',
                issue: 0,
              },
            ],
            next_before: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET') {
          if (document.body.dataset.malformed === '1') return Response.json({comment_id: comment});
          return Response.json({
            comment_id: comment,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            route: 'existing_work',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        }
        return null;
      });
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:record:${work}`,
        JSON.stringify([{commentId: comment, key: 'k1'}])
      );
    },
    {work, comment, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-comment-retry="${comment}"]`).first().waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Hold for malformed check.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await page.evaluate(() => {
    document.body.dataset.malformed = '1';
  });
  await nested.getByRole('button', {name: 'Check progress', exact: true}).click();
  await page.waitForTimeout(500);
  assert.equal(await feed.locator(`#comment-${comment}`).count(), 1);
  await nested.getByText('Hold for malformed check.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a mismatched tombstone never purges a different comment', async (t) => {
  const page = await fixture(t);
  const work = 'aa'.repeat(16);
  const pending = 'bb'.repeat(16);
  const settled = 'cc'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, pending, settled, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const row = (comment_id: string, body: string, advanced = false) => ({
        comment_id,
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment: advanced ? 'applied' : 'pending',
        route: advanced ? 'discussion' : 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: 7,
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [row(pending, 'Pending under binding check.'), row(settled, 'Settled survivor.')],
            next_before: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${pending}`) && call.method === 'GET') {
          if (document.body.dataset.mismatched === '1') return Response.json({comment_id: settled, deleted: true});
          return Response.json({
            comment_id: pending,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            route: 'existing_work',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        }
        return null;
      });
    },
    {work, pending, settled, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const target = feed.locator(`#comment-${pending}`);
  await target.getByText('Pending under binding check.', {exact: true}).waitFor();
  await target.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await feed.locator(`#comment-${settled}`).getByText('Settled survivor.', {exact: true}).waitFor();
  await page.evaluate(
    ({work, pending}) => {
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:record:${work}`,
        JSON.stringify([{commentId: pending, key: 'k1'}])
      );
    },
    {work, pending}
  );
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForTimeout(500);
  await page.evaluate(() => {
    document.body.dataset.mismatched = '1';
  });
  await target.getByRole('button', {name: 'Check progress', exact: true}).click();
  await page.waitForTimeout(500);
  assert.equal(await feed.locator(`#comment-${pending}`).count(), 1);
  await target.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  assert.equal(await feed.locator(`#comment-${settled}`).count(), 1);
  await feed.locator(`#comment-${settled}`).getByText('Settled survivor.', {exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a retried post body that reports removed keeps no Check progress', async (t) => {
  const page = await fixture(t);
  const post = 'dd'.repeat(16);
  await page.evaluate(({post}) => {
    window.drawerFixture.api.setPresentation('settings');
    let reads = 0;
    (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries: [{kind: 'post', id: post, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000}],
          truncated: false,
        });
      if (call.url.endsWith(`/api/factory/feed/comments/${post}`) && call.method === 'GET') {
        reads++;
        if (reads === 1) return new Response(null, {status: 503});
        return Response.json({
          comment_id: post,
          repository: 7,
          revision: 0,
          deleted: true,
          intake: 'recorded',
          judgment: 'pending',
          route: 'existing_work',
          draft: 'pending',
          effect: 'pending',
          issue: 0,
        });
      }
      return null;
    });
    window.drawerFixture.mountFeed();
  }, {post});
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  const retry = feed.getByRole('button', {name: 'Retry loading discussion text', exact: true});
  await retry.waitFor();
  const draft = feed.getByLabel('Project comment', {exact: true});
  await draft.fill('Hold this unsent note.');
  await retry.click();
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .waitFor();
  assert.equal(await draft.inputValue(), 'Hold this unsent note.');
  assert.equal(await feed.locator(`[data-comment-retry="${post}"]`).count(), 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    2
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a cached thread refresh survives reload across close, advance and reopen', async (t) => {
  const page = await fixture(t);
  const work = 'ef'.repeat(16);
  const comment = 'be'.repeat(16);
  const at = 1729000000;
  await page.evaluate(
    ({work, comment, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const pipeline = () =>
        document.body.dataset.stage === '1'
          ? {
              comment_id: comment,
              repository: 7,
              revision: 0,
              intake: 'recorded',
              judgment: 'applied',
              route: 'discussion',
              draft: 'pending',
              effect: 'pending',
              issue: 0,
            }
          : {
              comment_id: comment,
              repository: 7,
              revision: 0,
              intake: 'recorded',
              judgment: 'pending',
              route: 'existing_work',
              draft: 'pending',
              effect: 'pending',
              issue: 0,
            };
      const fillers = Array.from({length: 19}, (_, index) => ({
        comment_id: (101 + index).toString(16).padStart(32, '0'),
        repository: 7,
        revision: 0,
        author: '1',
        author_name: 'Ada',
        at,
        edited: false,
        deleted: false,
        revoked: false,
        body: `Filler reply ${index + 1}.`,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      }));
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?')) {
          const entries: Record<string, unknown>[] = [
            {
              kind: 'record',
              id: work,
              repository: 7,
              seq: 20,
              actor: 'builder',
              author: 'Builder',
              status: 'assigned',
              summary: 'Working on the demo.',
            },
          ];
          for (let seq = 19; seq >= 1; seq--) {
            entries.push({kind: 'record', id: seq.toString(16).padStart(32, '0'), repository: 7, seq});
          }
          return Response.json({entries, truncated: false});
        }
        if (call.url.includes('/api/factory/feed/replies?'))
          return Response.json({
            items: [
              ...fillers,
              {
                author: '1',
                author_name: 'Ada',
                at,
                edited: false,
                deleted: false,
                revoked: false,
                body: 'Please add substitute approvers.',
                ...pipeline(),
              },
            ],
            next_before: 0,
          });
        if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
          return Response.json(pipeline());
        return null;
      });
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:record:${work}`,
        JSON.stringify([{commentId: comment, key: 'k1'}])
      );
    },
    {work, comment, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-comment-retry="${comment}"]`).first().waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction((id) => {
    const raws: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith('soda-feed-intake:v1:')) raws.push(localStorage.getItem(key) ?? '');
    }
    return raws.length > 0 && raws.every((raw) => !raw.includes(id as string));
  }, comment);
  // Actual reload: drop the element and mount a fresh feed. In-memory
  // watches clear while stored intake (already retired) stays gone.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await nested.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await page.evaluate(() => {
    document.body.dataset.stage = '1';
  });
  const seenGets = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    comment
  );
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction(
    ([id, seen]) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length > (seen as number),
    [comment, seenGets]
  );
  await page.waitForTimeout(500);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const reopened = feed.locator(`#comment-${comment}`);
  await reopened.getByText('Please add substitute approvers.', {exact: true}).waitFor();
  await reopened.getByText('Kept as discussion.', {exact: true}).waitFor();
  assert.equal(await reopened.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a cached change progression survives a genuine reload across close, advance and reopen', async (t) => {
  const page = await fixture(t);
  const work = 'ef'.repeat(16);
  const comment = 'be'.repeat(16);
  const intent = 'd' + 'b'.repeat(24);
  const acceptedRevision = 'd' + 'c'.repeat(24);
  const draftRevision = 'd' + 'e'.repeat(24);
  const at = 1729000000;
  const setupFeed = ({
    work,
    comment,
    intent,
    acceptedRevision,
    draftRevision,
    at,
    seedIntake,
  }: {
    work: string;
    comment: string;
    intent: string;
    acceptedRevision: string;
    draftRevision: string;
    at: number;
    seedIntake: boolean;
  }) => {
    window.drawerFixture.api.setPresentation('settings');
    const change = () =>
      document.body.dataset.stage === '1'
        ? {
            comment_id: comment,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'applied',
            route: 'new_issue',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
            product_change: {
              intent,
              accepted_revision: acceptedRevision,
              draft_revision: draftRevision,
              state: 'accepted',
              can_review: false,
              verified: false,
              evidence_status: 'unavailable',
            },
          }
        : {
            comment_id: comment,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'applied',
            route: 'new_issue',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
            product_change: {
              intent,
              accepted_revision: acceptedRevision,
              state: 'review_required',
              can_review: true,
              verified: false,
              evidence_status: 'unavailable',
            },
          };
    const fillers = Array.from({length: 19}, (_, index) => ({
      comment_id: (101 + index).toString(16).padStart(32, '0'),
      repository: 7,
      revision: 0,
      author: '1',
      author_name: 'Ada',
      at,
      edited: false,
      deleted: false,
      revoked: false,
      body: `Filler reply ${index + 1}.`,
      intake: 'recorded',
      judgment: 'applied',
      route: 'discussion',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    }));
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) {
        const entries: Record<string, unknown>[] = [
          {
            kind: 'record',
            id: work,
            repository: 7,
            seq: 20,
            actor: 'builder',
            author: 'Builder',
            status: 'assigned',
            summary: 'Working on the demo.',
          },
        ];
        for (let seq = 19; seq >= 1; seq--) {
          entries.push({kind: 'record', id: seq.toString(16).padStart(32, '0'), repository: 7, seq});
        }
        return Response.json({entries, truncated: false});
      }
      if (call.url.includes('/api/factory/feed/replies?'))
        return Response.json({
          items: [
            ...fillers,
            {
              author: '1',
              author_name: 'Ada',
              at,
              edited: false,
              deleted: false,
              revoked: false,
              body: 'Please adopt the new direction.',
              ...change(),
            },
          ],
          next_before: 0,
        });
      if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
        return Response.json(change());
      return null;
    });
    if (seedIntake)
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:record:${work}`,
        JSON.stringify([{commentId: comment, key: 'k1'}])
      );
  };
  const ids = {work, comment, intent, acceptedRevision, draftRevision, at};
  await page.evaluate(setupFeed, {...ids, seedIntake: true});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  let feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  let nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please adopt the new direction.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await nested
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction((id) => {
    const raws: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith('soda-feed-intake:v1:')) raws.push(localStorage.getItem(key) ?? '');
    }
    return raws.length > 0 && raws.every((raw) => !raw.includes(id as string));
  }, comment);
  // A genuine reload: real navigation drops the in-memory watches and
  // threads while localStorage (intake already retired) persists.
  await page.reload();
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(() => {
    window.drawerFixture = window.createDrawerFixture({});
    return window.drawerFixture.api.ready;
  });
  await page.evaluate(setupFeed, {...ids, seedIntake: false});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please adopt the new direction.', {exact: true}).waitFor();
  await nested.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await page.evaluate(() => {
    document.body.dataset.stage = '1';
  });
  const seenGets = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    comment
  );
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction(
    ([id, seen]) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length > (seen as number),
    [comment, seenGets],
    {timeout: 8000}
  );
  await page.waitForTimeout(500);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const reopened = feed.locator(`#comment-${comment}`);
  await reopened.getByText('Please adopt the new direction.', {exact: true}).waitFor();
  await reopened
    .getByText('The changed direction is accepted. Follow its build and verification before applying a version.', {
      exact: true,
    })
    .waitFor();
  assert.equal(
    await reopened.getByText('Review this proposed change before updating the accepted direction.', {exact: true}).count(),
    0
  );
  assert.equal(await reopened.getByRole('button', {name: 'Review proposed change', exact: true}).count(), 0);
  assert.equal(await reopened.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await feed.getByText(/Saved\. JEV is assessing/).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed healthy body never restores a removed post during refresh', async (t) => {
  const page = await fixture(t);
  const target = (20).toString(16).padStart(32, '0');
  await page.evaluate(({target}) => {
    window.drawerFixture.api.setPresentation('settings');
    const healthy = (comment: string) => ({
      comment_id: comment,
      repository: 7,
      revision: 0,
      author_name: 'Ada',
      at: 1720000000,
      edited: false,
      deleted: false,
      revoked: false,
      body: `Discussion post ${parseInt(comment, 16)}.`,
      intake: 'recorded',
      judgment: 'pending',
      route: 'existing_work',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    });
    const entries: Record<string, unknown>[] = [];
    for (let seq = 20; seq >= 1; seq--) {
      entries.push({
        kind: 'post',
        id: seq.toString(16).padStart(32, '0'),
        repository: 7,
        seq,
        actor: 'Builder',
        author: 'Ada',
        project: 'soda-tester/demo',
        at: 1720000000 + seq,
        status: 'assessing',
      });
    }
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) {
        if (call.url.includes('cursor=')) return Response.json({entries: [], truncated: false, next: ''});
        return Response.json({entries, truncated: true, next: 'v1:1'});
      }
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
        if (comment === target && document.body.dataset.removed === '1')
          return Response.json({comment_id: comment, deleted: true});
        if (
          comment === target &&
          document.body.dataset.gated === '1' &&
          !(window as unknown as {heldBody?: boolean}).heldBody
        ) {
          (window as unknown as {heldBody?: boolean}).heldBody = true;
          while (document.body.dataset.released !== '1') await new Promise((run) => setTimeout(run, 25));
          return Response.json(healthy(comment));
        }
        return Response.json(healthy(comment));
      }
      return null;
    });
  }, {target});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion post 20.', {exact: true}).waitFor();
  const retry = feed.locator(`[data-comment-retry="${target}"]`);
  await retry.waitFor();
  const seenGets = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    target
  );
  await page.evaluate(() => {
    document.body.dataset.gated = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  // The head post's body GET is genuinely held open before the removal lands.
  await page.waitForFunction(
    ([id, seen]) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length > (seen as number),
    [target, seenGets]
  );
  await page.waitForTimeout(200);
  await page.evaluate(() => {
    document.body.dataset.removed = '1';
  });
  await feed.getByRole('button', {name: 'Load older entries', exact: true}).click();
  // The second pass reads the removal while the first pass is still held.
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor();
  // Only now does the delayed healthy body resolve, over the tombstone.
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(
    await feed
      .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
      .count(),
    1
  );
  assert.equal(await feed.getByText('Discussion post 20.', {exact: true}).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${target}"]`).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed retry body never restores a removed post after refresh', async (t) => {
  const page = await fixture(t);
  const target = (20).toString(16).padStart(32, '0');
  await page.evaluate(({target}) => {
    window.drawerFixture.api.setPresentation('settings');
    let reads = 0;
    (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
    const healthy = {
      comment_id: target,
      repository: 7,
      revision: 0,
      author_name: 'Ada',
      at: 1720000000,
      edited: false,
      deleted: false,
      revoked: false,
      body: 'Recovered discussion post.',
      intake: 'recorded',
      judgment: 'pending',
      route: 'existing_work',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    };
    const entries: Record<string, unknown>[] = [];
    for (let seq = 20; seq >= 1; seq--) {
      entries.push({
        kind: 'post',
        id: seq.toString(16).padStart(32, '0'),
        repository: 7,
        seq,
        actor: 'Builder',
        author: 'Ada',
        project: 'soda-tester/demo',
        at: 1720000000 + seq,
        status: 'assessing',
      });
    }
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries, truncated: false});
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
        if (comment !== target)
          return Response.json({
            comment_id: comment,
            repository: 7,
            revision: 0,
            author_name: 'Ada',
            at: 1720000000,
            edited: false,
            deleted: false,
            revoked: false,
            body: `Discussion post ${parseInt(comment, 16)}.`,
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        reads++;
        if (reads === 1) return new Response(null, {status: 503});
        if (document.body.dataset.removed === '1') return Response.json({comment_id: comment, deleted: true});
        if (document.body.dataset.gated === '1' && !(window as unknown as {heldRetry?: boolean}).heldRetry) {
          (window as unknown as {heldRetry?: boolean}).heldRetry = true;
          while (document.body.dataset.released !== '1') await new Promise((run) => setTimeout(run, 25));
        }
        return Response.json(healthy);
      }
      return null;
    });
  }, {target});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  const draft = feed.getByLabel('Project comment', {exact: true});
  await draft.fill('Hold this unsent note.');
  await page.evaluate(() => {
    document.body.dataset.gated = '1';
  });
  await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).click();
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 2,
    target
  );
  await page.waitForTimeout(200);
  await page.evaluate(() => {
    document.body.dataset.removed = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor();
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(
    await feed
      .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
      .count(),
    1
  );
  assert.equal(await feed.getByText('Recovered discussion post.', {exact: true}).count(), 0);
  assert.equal(await draft.inputValue(), 'Hold this unsent note.');
  assert.equal(await feed.locator(`[data-comment-retry="${target}"]`).count(), 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    3
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a missing post body retires its Check progress controls', async (t) => {
  const page = await fixture(t);
  const target = (20).toString(16).padStart(32, '0');
  await page.evaluate(({target}) => {
    window.drawerFixture.api.setPresentation('settings');
    const healthy = (comment: string) => ({
      comment_id: comment,
      repository: 7,
      revision: 0,
      author_name: 'Ada',
      at: 1720000000,
      edited: false,
      deleted: false,
      revoked: false,
      body: `Discussion post ${parseInt(comment, 16)}.`,
      intake: 'recorded',
      judgment: 'pending',
      route: 'existing_work',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    });
    const entries: Record<string, unknown>[] = [];
    for (let seq = 20; seq >= 1; seq--) {
      entries.push({
        kind: 'post',
        id: seq.toString(16).padStart(32, '0'),
        repository: 7,
        seq,
        actor: 'Builder',
        author: 'Ada',
        project: 'soda-tester/demo',
        at: 1720000000 + seq,
        status: 'assessing',
      });
    }
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries, truncated: false});
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
        if (comment === target && document.body.dataset.gone === '1') return new Response(null, {status: 404});
        return Response.json(healthy(comment));
      }
      return null;
    });
  }, {target});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion post 20.', {exact: true}).waitFor();
  await feed.locator(`[data-comment-retry="${target}"]`).waitFor();
  await page.evaluate(() => {
    document.body.dataset.gone = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed
    .getByText('That discussion post is no longer visible. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor();
  assert.equal(await feed.getByText('Discussion post 20.', {exact: true}).count(), 0);
  assert.equal(await feed.locator(`[data-comment-retry="${target}"]`).count(), 0);
  assert.equal(await feed.locator('.soda-feed-outcome').getByText(/Finding the issue/).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a revoked post with empty body and pipeline stays removed', async (t) => {
  const page = await fixture(t);
  const target = (20).toString(16).padStart(32, '0');
  await page.evaluate(({target}) => {
    window.drawerFixture.api.setPresentation('settings');
    let reads = 0;
    (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
    const entries: Record<string, unknown>[] = [];
    for (let seq = 20; seq >= 1; seq--) {
      entries.push({
        kind: 'post',
        id: seq.toString(16).padStart(32, '0'),
        repository: 7,
        seq,
        actor: 'Builder',
        author: 'Ada',
        project: 'soda-tester/demo',
        at: 1720000000 + seq,
        status: 'assessing',
      });
    }
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries, truncated: false});
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
        if (comment !== target)
          return Response.json({
            comment_id: comment,
            repository: 7,
            revision: 0,
            author_name: 'Ada',
            at: 1720000000,
            edited: false,
            deleted: false,
            revoked: false,
            body: `Discussion post ${parseInt(comment, 16)}.`,
            intake: 'recorded',
            judgment: 'applied',
            route: 'discussion',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
          });
        reads++;
        if (reads === 1) return new Response(null, {status: 503});
        return Response.json({
          comment_id: comment,
          repository: 7,
          revision: 0,
          revoked: true,
          body: '',
          author_name: 'Ada',
          at: 1720000000,
          edited: false,
          intake: 'recorded',
          judgment: 'pending',
          route: 'existing_work',
          draft: 'pending',
          effect: 'pending',
          issue: 0,
        });
      }
      return null;
    });
  }, {target});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  const draft = feed.getByLabel('Project comment', {exact: true});
  await draft.fill('Hold this unsent note.');
  await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).click();
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor();
  assert.equal(await draft.inputValue(), 'Hold this unsent note.');
  assert.equal(await feed.locator(`[data-comment-retry="${target}"]`).count(), 0);
  // The Retry control stays: it re-reads the same removal instead of
  // reviving pipeline progress, so the recovery path is never stranded.
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    2
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a closed unavailable change recovers after reload when the server restores review_required', async (t) => {
  const page = await fixture(t);
  const work = 'e1'.repeat(16);
  const comment = 'bf'.repeat(16);
  const intent = 'd' + 'b'.repeat(24);
  const acceptedRevision = 'd' + 'c'.repeat(24);
  const at = 1729000000;
  const setupFeed = ({
    work,
    comment,
    intent,
    acceptedRevision,
    at,
    seedIntake,
  }: {
    work: string;
    comment: string;
    intent: string;
    acceptedRevision: string;
    at: number;
    seedIntake: boolean;
  }) => {
    window.drawerFixture.api.setPresentation('settings');
    const change = () =>
      document.body.dataset.stage === '1'
        ? {
            comment_id: comment,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'applied',
            route: 'new_issue',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
            product_change: {
              intent,
              accepted_revision: acceptedRevision,
              state: 'review_required',
              can_review: true,
              verified: false,
              evidence_status: 'unavailable',
            },
          }
        : {
            comment_id: comment,
            repository: 7,
            revision: 0,
            intake: 'recorded',
            judgment: 'applied',
            route: 'new_issue',
            draft: 'pending',
            effect: 'pending',
            issue: 0,
            product_change: {
              intent,
              accepted_revision: acceptedRevision,
              state: 'unavailable',
              can_review: false,
              verified: false,
              evidence_status: 'unavailable',
            },
          };
    const fillers = Array.from({length: 19}, (_, index) => ({
      comment_id: (101 + index).toString(16).padStart(32, '0'),
      repository: 7,
      revision: 0,
      author: '1',
      author_name: 'Ada',
      at,
      edited: false,
      deleted: false,
      revoked: false,
      body: `Filler reply ${index + 1}.`,
      intake: 'recorded',
      judgment: 'applied',
      route: 'discussion',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    }));
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) {
        const entries: Record<string, unknown>[] = [
          {
            kind: 'record',
            id: work,
            repository: 7,
            seq: 20,
            actor: 'builder',
            author: 'Builder',
            status: 'assigned',
            summary: 'Working on the demo.',
          },
        ];
        for (let seq = 19; seq >= 1; seq--) {
          entries.push({kind: 'record', id: seq.toString(16).padStart(32, '0'), repository: 7, seq});
        }
        return Response.json({entries, truncated: false});
      }
      if (call.url.includes('/api/factory/feed/replies?'))
        return Response.json({
          items: [
            ...fillers,
            {
              author: '1',
              author_name: 'Ada',
              at,
              edited: false,
              deleted: false,
              revoked: false,
              body: 'Please adopt the new direction.',
              ...change(),
            },
          ],
          next_before: 0,
        });
      if (call.url.endsWith(`/api/factory/feed/comments/${comment}`) && call.method === 'GET')
        return Response.json(change());
      return null;
    });
    if (seedIntake)
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:record:${work}`,
        JSON.stringify([{commentId: comment, key: 'k1'}])
      );
  };
  const ids = {work, comment, intent, acceptedRevision, at};
  await page.evaluate(setupFeed, {...ids, seedIntake: true});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  let feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  let nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please adopt the new direction.', {exact: true}).waitFor();
  await nested.getByText('The proposed change detail is currently unavailable. Check progress to reload it.').waitFor();
  await nested.getByRole('button', {name: 'Check progress', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction((id) => {
    const raws: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith('soda-feed-intake:v1:')) raws.push(localStorage.getItem(key) ?? '');
    }
    return raws.length > 0 && raws.every((raw) => !raw.includes(id as string));
  }, comment);
  // A genuine reload: real navigation drops the in-memory watches and
  // threads while localStorage (intake already retired) persists.
  await page.reload();
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(() => {
    window.drawerFixture = window.createDrawerFixture({});
    return window.drawerFixture.api.ready;
  });
  await page.evaluate(setupFeed, {...ids, seedIntake: false});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity'}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  nested = feed.locator(`#comment-${comment}`);
  await nested.getByText('Please adopt the new direction.', {exact: true}).waitFor();
  await nested.getByText('The proposed change detail is currently unavailable. Check progress to reload it.').waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await page.evaluate(() => {
    document.body.dataset.stage = '1';
  });
  const seenGets = await page.evaluate(
    (id) => window.drawerFixture.calls.filter((call) => call.url.endsWith(`/api/factory/feed/comments/${id}`)).length,
    comment
  );
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
  await page.waitForFunction(
    ([id, seen]) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length > (seen as number),
    [comment, seenGets],
    {timeout: 8000}
  );
  await page.waitForTimeout(500);
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const reopened = feed.locator(`#comment-${comment}`);
  await reopened.getByText('Please adopt the new direction.', {exact: true}).waitFor({timeout: 8000});
  await reopened
    .getByText('Review this proposed change before updating the accepted direction.', {exact: true})
    .waitFor({timeout: 8000});
  await reopened.getByRole('button', {name: 'Review proposed change', exact: true}).waitFor();
  assert.equal(
    await reopened
      .getByText('The proposed change detail is currently unavailable. Check progress to reload it.')
      .count(),
    0
  );
  assert.equal(await reopened.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  // The cached refresh updates the one change row in place: fillers stay
  // exactly once, nothing duplicates or resurrects.
  assert.equal(await feed.getByText('Filler reply 1.', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Filler reply 19.', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Please adopt the new direction.', {exact: true}).count(), 1);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed 503 body failure never overwrites a newer successful body read', async (t) => {
  const page = await fixture(t);
  const target = (21).toString(16).padStart(32, '0');
  await page.evaluate(
    ({target}) => {
      window.drawerFixture.api.setPresentation('settings');
      let reads = 0;
      (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
      const healthy = {
        comment_id: target,
        repository: 7,
        revision: 0,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Recovered discussion text.',
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {kind: 'post', id: target, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000},
            ],
            truncated: false,
          });
        const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
        if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
          reads++;
          if (reads === 1) {
            while (document.body.dataset.released !== '1') await new Promise((run) => setTimeout(run, 25));
            return new Response(null, {status: 503});
          }
          return Response.json(healthy);
        }
        return null;
      });
    },
    {target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Loading discussion text…', {exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Recovered discussion text.', {exact: true}).waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.getByText('Recovered discussion text.', {exact: true}).count(), 1);
  assert.equal(
    await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).count(),
    0
  );
  assert.equal(await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    2
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed 404 retry never overwrites a newer successful body read', async (t) => {
  const page = await fixture(t);
  const target = (22).toString(16).padStart(32, '0');
  await page.evaluate(
    ({target}) => {
      window.drawerFixture.api.setPresentation('settings');
      let reads = 0;
      (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
      const healthy = {
        comment_id: target,
        repository: 7,
        revision: 0,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Recovered discussion post.',
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {kind: 'post', id: target, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000},
            ],
            truncated: false,
          });
        const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
        if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
          reads++;
          if (reads === 1) return new Response(null, {status: 503});
          if (reads === 2) {
            while (document.body.dataset.released !== '1') await new Promise((run) => setTimeout(run, 25));
            return new Response(null, {status: 404});
          }
          return Response.json(healthy);
        }
        return null;
      });
    },
    {target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).click();
  // The retry is genuinely in flight before the fresher read lands.
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 2,
    target
  );
  await page.waitForTimeout(200);
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Recovered discussion post.', {exact: true}).waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.getByText('Recovered discussion post.', {exact: true}).count(), 1);
  assert.equal(
    await feed
      .getByText('That discussion post is no longer visible. Its pipeline record is kept where stored.', {exact: true})
      .count(),
    0
  );
  assert.equal(
    await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).count(),
    0
  );
  assert.equal(await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    3
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a delayed malformed retry body never overwrites a newer successful body read', async (t) => {
  const page = await fixture(t);
  const target = (23).toString(16).padStart(32, '0');
  await page.evaluate(
    ({target}) => {
      window.drawerFixture.api.setPresentation('settings');
      let reads = 0;
      (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
      const healthy = {
        comment_id: target,
        repository: 7,
        revision: 0,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Recovered discussion post.',
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {kind: 'post', id: target, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000},
            ],
            truncated: false,
          });
        const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
        if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
          reads++;
          if (reads === 1) return new Response(null, {status: 503});
          if (reads === 2) {
            while (document.body.dataset.released !== '1') await new Promise((run) => setTimeout(run, 25));
            // Bodyless without deleted/revoked flags: malformed, never a deletion.
            return Response.json({comment_id: comment});
          }
          return Response.json(healthy);
        }
        return null;
      });
    },
    {target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).click();
  // The retry is genuinely in flight before the fresher read lands.
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 2,
    target
  );
  await page.waitForTimeout(200);
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Recovered discussion post.', {exact: true}).waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(await feed.getByText('Recovered discussion post.', {exact: true}).count(), 1);
  assert.equal(
    await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).count(),
    0
  );
  assert.equal(await feed.getByRole('button', {name: 'Retry loading discussion text', exact: true}).count(), 0);
  assert.equal(
    await page.evaluate(() => (window as unknown as {postBodyReads?: () => number}).postBodyReads?.() ?? 0),
    3
  );
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a newer Check progress never orphans an in-flight body retry in loading', async (t) => {
  const page = await fixture(t);
  const target = (24).toString(16).padStart(32, '0');
  await page.evaluate(
    ({target}) => {
      window.drawerFixture.api.setPresentation('settings');
      let reads = 0;
      let heldRetry = false;
      (window as unknown as {postBodyReads?: () => number}).postBodyReads = () => reads;
      (window as unknown as {rearmRetry?: () => void}).rearmRetry = () => {
        heldRetry = false;
      };
      const failedPipeline = {
        comment_id: target,
        repository: 7,
        revision: 0,
        intake: 'recorded',
        judgment: 'failed',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      const healthy = {
        comment_id: target,
        repository: 7,
        revision: 0,
        author_name: 'Ada',
        at: 1720000000,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Recovered discussion text.',
        intake: 'recorded',
        judgment: 'pending',
        route: 'existing_work',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {kind: 'post', id: target, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000},
            ],
            truncated: false,
          });
        const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
        if (/^[a-f0-9]{32}$/.test(comment) && call.method === 'GET') {
          reads++;
          if (document.body.dataset.removed === '1') return Response.json({comment_id: comment, deleted: true});
          // Mount order is deterministic: the outcomes pass reads first and
          // seeds the stored outcome, then the body pass fails fast so the
          // post carries a Retry control beside Check progress.
          if (reads === 1) return Response.json(failedPipeline);
          if (reads === 2) return new Response(null, {status: 503});
          // The re-fail pass behind failBody fails fast the same way later.
          if (document.body.dataset.failBody === '1') return new Response(null, {status: 503});
          // The first gated GET is the held retry; later gated GETs (the
          // Check progress read, quiet polls) serve the current state at once.
          if (document.body.dataset.gated === '1' && !heldRetry) {
            heldRetry = true;
            while (document.body.dataset.released !== '1') await new Promise((run) => setTimeout(run, 25));
          }
          return Response.json(healthy);
        }
        return null;
      });
      window.localStorage.setItem(
        `soda-feed-intake:v1:fixture-generation:post:${target}`,
        JSON.stringify([{commentId: target, key: 'k1'}])
      );
    },
    {target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor();
  // The seeded stored outcome lands through the outcomes pass: with the
  // shared clock it never lands promptly, so this bounds the pre-fix wait.
  await feed.getByRole('button', {name: 'Check progress', exact: true}).waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.gated = '1';
  });
  // Dispatched directly: the retry replaces its own button with loading the
  // moment it fires, which a waiting click would chase instead of firing.
  await page.evaluate(() => {
    const retry = document.querySelector('soda-factory-feed [data-post-body-retry]');
    if (!(retry instanceof HTMLElement)) throw Error('missing body retry');
    retry.click();
  });
  // The retry is genuinely in flight before the Check progress lands.
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 3,
    target,
    {timeout: 8000}
  );
  await feed.getByText('Loading discussion text…', {exact: true}).waitFor({timeout: 8000});
  // Dispatched directly: the post's drawer pull-tab overlaps the outcome's
  // Check button in this layout, but the test targets the progress/body
  // ordering underneath, not pointer hit-testing.
  await page.evaluate(
    ({target}) => {
      const check = document.querySelector(`soda-factory-feed [data-comment-retry="${target}"]`);
      if (!(check instanceof HTMLElement)) throw Error('missing check progress');
      check.click();
    },
    {target}
  );
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 4,
    target,
    {timeout: 8000}
  );
  await page.waitForTimeout(500);
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  // The in-flight retry still lands its text instead of stranding in
  // loading, while the newer pipeline update stands beside it.
  await feed.getByText('Recovered discussion text.', {exact: true}).waitFor({timeout: 8000});
  assert.equal(await feed.getByText('Loading discussion text…', {exact: true}).count(), 0);
  assert.equal(
    await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).count(),
    0
  );
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 1);
  // A removal that lands while a stale retry is still in flight still wins.
  await page.evaluate(() => {
    document.body.dataset.failBody = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click({timeout: 8000});
  await feed
    .getByText('Discussion text unavailable; the post keeps its place.', {exact: true})
    .waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.failBody = '';
    document.body.dataset.released = '0';
    (window as unknown as {rearmRetry?: () => void}).rearmRetry?.();
  });
  await page.evaluate(() => {
    const retry = document.querySelector('soda-factory-feed [data-post-body-retry]');
    if (!(retry instanceof HTMLElement)) throw Error('missing body retry');
    retry.click();
  });
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 7,
    target,
    {timeout: 8000}
  );
  await page.waitForTimeout(200);
  await page.evaluate(() => {
    document.body.dataset.removed = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click({timeout: 8000});
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(
    await feed
      .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
      .count(),
    1
  );
  assert.equal(await feed.getByText('Recovered discussion text.', {exact: true}).count(), 0);
  assert.equal(await feed.getByRole('button', {name: 'Check progress', exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a slow body parse keeps a newer removal and a newer body update', async (t) => {
  const page = await fixture(t);
  const target = (30).toString(16).padStart(32, '0');
  const other = (31).toString(16).padStart(32, '0');
  await page.evaluate(({target, other}) => {
    window.drawerFixture.api.setPresentation('settings');
    let held = false,
      targetReads = 0,
      otherReads = 0;
    const healthy = (comment: string, body: string) => ({
      comment_id: comment,
      repository: 7,
      revision: 0,
      author_name: 'Ada',
      at: 1720000000,
      edited: false,
      deleted: false,
      revoked: false,
      body,
      intake: 'recorded',
      judgment: 'applied',
      route: 'discussion',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    });
    const entries = [target, other].map((id, index) => ({
      kind: 'post',
      id,
      repository: 7,
      seq: 5 - index,
      actor: '1',
      author_name: 'Ada',
      at: 1720000000,
    }));
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) return Response.json({entries, truncated: false});
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (!/^[a-f0-9]{32}$/.test(comment) || call.method !== 'GET') return null;
      if (comment === target) {
        targetReads++;
        if (targetReads === 1) return new Response(null, {status: 503});
        if (document.body.dataset.removed === '1') return Response.json({comment_id: comment, deleted: true});
        // The retry headers land at once; only the response body parses
        // slowly, so a newer removal or body update lands mid-parse.
        if (document.body.dataset.holdBody === '1' && !held) {
          held = true;
          const payload = JSON.stringify(healthy(comment, 'Target discussion text.'));
          return new Response(
            new ReadableStream({
              start(controller) {
                const timer = setInterval(() => {
                  if (document.body.dataset.released === '1') {
                    clearInterval(timer);
                    controller.enqueue(new TextEncoder().encode(payload));
                    controller.close();
                  }
                }, 25);
              },
            }),
            {headers: {'Content-Type': 'application/json'}}
          );
        }
        return Response.json(healthy(comment, 'Target discussion text.'));
      }
      if (comment === other) {
        otherReads++;
        if (otherReads === 1) return new Response(null, {status: 503});
        return Response.json(healthy(comment, 'Other discussion text.'));
      }
      return null;
    });
  }, {target, other});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator('[data-post-body-retry]').nth(1).waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.holdBody = '1';
  });
  await feed.locator(`[data-post-body-retry="${target}"]`).click({timeout: 8000});
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 2,
    target,
    {timeout: 8000}
  );
  // Another post's body succeeds while the target response body is still
  // parsing; the delayed target payload must never erase it.
  await feed.locator(`[data-post-body-retry="${other}"]`).click({timeout: 8000});
  await feed.getByText('Other discussion text.', {exact: true}).waitFor({timeout: 8000});
  // A removal for the target lands while its response body is still
  // parsing; the delayed healthy payload must drop over the tombstone.
  await page.evaluate(() => {
    document.body.dataset.removed = '1';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click({timeout: 8000});
  await feed
    .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
    .first()
    .waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.released = '1';
  });
  await page.waitForTimeout(500);
  assert.equal(
    await feed
      .getByText('That discussion post was removed. Its pipeline record is kept where stored.', {exact: true})
      .count(),
    1
  );
  assert.equal(await feed.getByText('Target discussion text.', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Other discussion text.', {exact: true}).count(), 1);
  assert.equal(await feed.getByText('Loading discussion text…', {exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});

test('a slower older body pipeline never overwrites a newer Check progress pipeline', async (t) => {
  const page = await fixture(t);
  const target = (32).toString(16).padStart(32, '0');
  await page.evaluate(({target}) => {
    window.drawerFixture.api.setPresentation('settings');
    let reads = 0,
      checkHeld = false,
      retryHeld = false;
    const failedPipeline = {
      comment_id: target,
      repository: 7,
      revision: 0,
      intake: 'recorded',
      judgment: 'failed',
      route: 'existing_work',
      draft: 'pending',
      effect: 'pending',
      issue: 0,
    };
    const newerPipeline = {...failedPipeline, judgment: 'pending'};
    const olderBody = {
      ...failedPipeline,
      author_name: 'Ada',
      at: 1720000000,
      edited: false,
      deleted: false,
      revoked: false,
      body: 'Recovered discussion text.',
    };
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: [7], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries: [
            {kind: 'post', id: target, repository: 7, seq: 5, actor: '1', author_name: 'Ada', at: 1720000000},
          ],
          truncated: false,
        });
      // Opening the drawer reads its (empty here) thread; that replies page
      // is not a comment GET and leaves the race counters below untouched.
      if (call.url.includes('/api/factory/feed/replies?')) return Response.json({items: [], next_before: 0});
      const comment = call.url.split('/api/factory/feed/comments/')[1]?.split('?')[0] ?? '';
      if (!/^[a-f0-9]{32}$/.test(comment) || call.method !== 'GET') return null;
      reads++;
      if (reads === 1) return Response.json(failedPipeline);
      if (reads === 2) return new Response(null, {status: 503});
      // The Check starts first but its response lands first with the newer
      // pipeline; the retry starts second and its older body parses later.
      if (document.body.dataset.holdCheck === '1' && !checkHeld) {
        checkHeld = true;
        while (document.body.dataset.checkReleased !== '1') await new Promise((run) => setTimeout(run, 25));
        return Response.json(newerPipeline);
      }
      if (document.body.dataset.holdBody === '1' && !retryHeld) {
        retryHeld = true;
        const payload = JSON.stringify(olderBody);
        return new Response(
          new ReadableStream({
            start(controller) {
              const timer = setInterval(() => {
                if (document.body.dataset.bodyReleased === '1') {
                  clearInterval(timer);
                  controller.enqueue(new TextEncoder().encode(payload));
                  controller.close();
                }
              }, 25);
            },
          }),
          {headers: {'Content-Type': 'application/json'}}
        );
      }
      return Response.json(newerPipeline);
    });
    window.localStorage.setItem(
      `soda-feed-intake:v1:fixture-generation:post:${target}`,
      JSON.stringify([{commentId: target, key: 'k1'}])
    );
  }, {target});
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Discussion text unavailable; the post keeps its place.', {exact: true}).waitFor({timeout: 8000});
  await feed.getByRole('button', {name: 'Check progress', exact: true}).waitFor({timeout: 8000});
  // Reachable path: the retryable outcome lives in the comment drawer, so a
  // pointing or keyboarding user opens the drawer first. The closed drawer's
  // zero-height body keeps the outcome's clipped box under the full-width
  // pull-tab, which wins hit-testing there; open, tab and outcome lay out in
  // separate rows and both controls are pointer-reachable.
  const toggle = feed.locator(`[data-feed-drawer-toggle="post:${target}"]`);
  await toggle.scrollIntoViewIfNeeded({timeout: 5000});
  await toggle.click({timeout: 5000});
  await feed.locator('.soda-feed-drawer[data-open="true"]').waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.holdCheck = '1';
    document.body.dataset.holdBody = '1';
  });
  // Keyboard proof: the ordinary pointer click above leaves focus on the
  // pull-tab, so one Tab reaches Check progress and Enter activates it.
  await page.keyboard.press('Tab');
  assert.equal(
    await page.evaluate(() =>
      document.activeElement instanceof HTMLElement ? (document.activeElement.dataset.commentRetry ?? '') : ''
    ),
    target
  );
  await page.keyboard.press('Enter');
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 3,
    target,
    {timeout: 8000}
  );
  const retry = feed.getByRole('button', {name: 'Retry loading discussion text', exact: true});
  await retry.scrollIntoViewIfNeeded({timeout: 5000});
  await retry.click({timeout: 5000});
  await page.waitForFunction(
    (id) =>
      (window as unknown as {drawerFixture: {calls: {url: string}[]}}).drawerFixture.calls.filter((call) =>
        call.url.endsWith(`/api/factory/feed/comments/${id}`)
      ).length >= 4,
    target,
    {timeout: 8000}
  );
  await feed.getByText('Loading discussion text…', {exact: true}).waitFor({timeout: 8000});
  // The Check lands first with the newer pipeline; the older body payload
  // parsing later must keep the retry text but drop its pipeline.
  await page.evaluate(() => {
    document.body.dataset.checkReleased = '1';
  });
  // Outcome rows share their paragraph with the Check progress control, so
  // the assertion matches on contained text rather than an exact paragraph.
  const newerOutcome = feed.locator('.soda-feed-outcome p', {hasText: 'Saved. JEV is assessing your comment.'});
  const olderOutcome = feed.locator('.soda-feed-outcome p', {
    hasText: 'Assessment failed after its bounded attempts',
  });
  await newerOutcome.waitFor({timeout: 8000});
  await page.evaluate(() => {
    document.body.dataset.bodyReleased = '1';
  });
  await feed.getByText('Recovered discussion text.', {exact: true}).waitFor({timeout: 8000});
  await page.waitForTimeout(300);
  assert.equal(await newerOutcome.count(), 1);
  assert.equal(await olderOutcome.count(), 0);
  assert.equal(await feed.getByText('Loading discussion text…', {exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.method !== 'GET').length), 0);
});
